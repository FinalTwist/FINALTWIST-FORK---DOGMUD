package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/targeting"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Spell effect unification (parity slices 3a and 3b,
// docs/superpowers/specs/2026-09-28-spell-effect-unification-design.md).
//
// Every spell effect on one target is applied through one spellEffectCtx and
// one dispatcher, whoever casts it and whoever it hits. The four contested
// resolvers in spell_resolution.go keep their names and their one
// runSpellChannelAttack call each; after the contest they build a context and
// hand it here.

// recordSpell is the analytics seam for one resolved cast. It defaults to
// combat.RecordSpell; same-package tests replace it with a recorder and
// restore it with t.Cleanup, the pattern runSpellChannelAttack uses.
var recordSpell = combat.RecordSpell

// spellEffectCtx is everything one spell effect needs about one target.
type spellEffectCtx struct {
	casterChar *characters.Character // nil only in tests that pass no caster
	caster     actions.Actor         // *actions.UserActor, *actions.MobActor, or nil
	target     actions.Actor         // *actions.UserActor or *actions.MobActor, never nil
	room       *rooms.Room
	spell      *spells.SpellData
	magnitude  int
	out        combat.ChannelDefenceResult // zero-valued win for uncontested casts
}

func newSpellEffectCtx(casterChar *characters.Character, caster, target actions.Actor, room *rooms.Room,
	spell *spells.SpellData, magnitude int, out combat.ChannelDefenceResult) spellEffectCtx {
	return spellEffectCtx{casterChar: casterChar, caster: caster, target: target, room: room,
		spell: spell, magnitude: magnitude, out: out}
}

// spellCasterActor wraps whichever caster a legacy call site was handed: the
// player record when there is one, else the registered mob behind casterChar,
// else nil (an anonymous caster, which only tests produce).
func spellCasterActor(user *users.UserRecord, casterChar *characters.Character, room *rooms.Room) actions.Actor {
	if user != nil {
		return actions.NewUserActorInRoom(user, room)
	}
	if casterChar != nil && casterChar.MobInstanceId > 0 {
		if m := mobs.GetInstance(casterChar.MobInstanceId); m != nil {
			return actions.NewMobActorInRoom(m, room)
		}
	}
	return nil
}

func actorUser(a actions.Actor) *users.UserRecord {
	if ua, ok := a.(*actions.UserActor); ok && ua != nil {
		return ua.User
	}
	return nil
}

func actorMob(a actions.Actor) *mobs.Mob {
	if ma, ok := a.(*actions.MobActor); ok && ma != nil {
		return ma.Mob
	}
	return nil
}

func actorRefOf(a actions.Actor) state.ActorRef {
	if a == nil {
		return state.ActorRef{}
	}
	return state.ActorRef{UserId: a.GetUserId(), MobInstanceId: a.GetMobInstanceId()}
}

func (c spellEffectCtx) casterUser() *users.UserRecord { return actorUser(c.caster) }
func (c spellEffectCtx) casterMob() *mobs.Mob          { return actorMob(c.caster) }
func (c spellEffectCtx) targetUser() *users.UserRecord { return actorUser(c.target) }
func (c spellEffectCtx) targetMob() *mobs.Mob          { return actorMob(c.target) }

func (c spellEffectCtx) targetChar() *characters.Character { return c.target.GetCharacter() }

// casterRef names the caster for aggro and harm attribution. It reads the
// actor, not the character: a mob's Character.MobInstanceId is not reliably
// set, its Mob.InstanceId is.
func (c spellEffectCtx) casterRef() state.ActorRef {
	if c.caster == nil {
		return charActorRef(c.casterChar)
	}
	return actorRefOf(c.caster)
}

func (c spellEffectCtx) targetRef() state.ActorRef { return actorRefOf(c.target) }

// viewerId is the user id mob names are rendered for: the player caster's,
// or 0 when a mob casts.
func (c spellEffectCtx) viewerId() int {
	if u := c.casterUser(); u != nil {
		return u.UserId
	}
	return 0
}

// spellEffectName is a party's name exactly as the spell lines print it, and
// so exactly as SendTrio must hide it: players in the username tag, mobs
// through mobDisplayName with the room's duplicate index.
func spellEffectName(a actions.Actor, room *rooms.Room, viewerId int) string {
	switch v := a.(type) {
	case *actions.UserActor:
		return fmt.Sprintf(`<ansi fg="username">%s</ansi>`, v.User.Character.Name)
	case *actions.MobActor:
		return mobDisplayName(v.Mob, room, viewerId)
	}
	return "something"
}

func (c spellEffectCtx) casterName() string { return spellEffectName(c.caster, c.room, c.viewerId()) }
func (c spellEffectCtx) targetName() string { return spellEffectName(c.target, c.room, c.viewerId()) }

func (c spellEffectCtx) critTag() string {
	if c.out.AttackerCrit {
		return ` <ansi fg="yellow">[CRIT!]</ansi>`
	}
	return ""
}

func (c spellEffectCtx) category() messaging.Category { return spellSchoolCategory(c.spell) }

// audience is the SendTrio audience for a line between caster and target.
// A mob side gets no private line (spellAudience stores no nil recipient),
// and the room line excludes whichever sides are players.
func (c spellEffectCtx) audience() messaging.Audience {
	return spellAudience(c.casterUser(), c.casterName(), c.targetUser(), c.targetName(), c.room)
}

func spellSourceTarget(a actions.Actor) combat.SourceTarget {
	if a != nil && a.IsPlayer() {
		return combat.User
	}
	return combat.Mob
}

// applySpellEffect applies one spell effect to one target and returns the
// damage it dealt (0 for effects that deal none). Until slice 3b, effects
// without a unified applier run on the per-pairing arms they always had.
func applySpellEffect(c spellEffectCtx) int {
	switch c.spell.EffectType {
	case "damage":
		return applySpellDamage(c)
	case "dot":
		return applySpellDot(c)
	case "knockdown":
		return applySpellKnockdown(c)
	}
	switch {
	case c.targetMob() != nil:
		return applyMobEffectArms(c)
	case c.casterMob() != nil:
		return applyMobOnPlayerArms(c)
	default:
		applyPlayerEffectArms(c)
		return 0
	}
}

// commitHarmfulSpellAggro is the one place a harmful spell starts a fight,
// for every pairing. fresh is whether the target was out of combat BEFORE
// this cast landed (the applier reads it first, because harm can end the
// target's fight). The target turns on its caster only when fresh, so an
// established fight is not yanked around; the caster turns on the target
// when it is not already fighting.
//
// A player's harm on a mob is also an assault (owner ruling, 2026-09-28):
// actions.SeedAggression fires PlayerAttackedMob on every cast and, when
// fresh, the opinion bump and the assault crime. Freshness is judged per
// target from the mob's own prior combat, exactly as usercommands/throw.go's
// engageAfterThrow judges it for an area throw.
func commitHarmfulSpellAggro(c spellEffectCtx, fresh bool) {
	tc := c.targetChar()
	if fresh {
		targeting.Commit(tc, c.casterRef(), targeting.ReasonAttack)
	}
	if c.casterChar != nil && !c.casterChar.IsInCombat() {
		targeting.Commit(c.casterChar, c.targetRef(), targeting.ReasonAttack)
	}
	if u, m := c.casterUser(), c.targetMob(); u != nil && m != nil {
		actions.SeedAggression(u, m, c.room, fresh)
	}
}

// applySpellDamage is the one damage applier (slice 3a). The resolver ran
// the ONE contest; this consumes it. A defended cast lands partial damage, a
// defensive crit negates it, and either way the cast was an attack.
func applySpellDamage(c spellEffectCtx) int {
	tc := c.targetChar()
	fresh := !tc.IsInCombat()
	dmg := scaleSpellDamageByDefence(
		calcSpellDamageForCharacter(c.spell, c.casterChar, tc, c.magnitude, c.out.AttackerCrit), c.out)
	sendSpellChannelDefenceMessages(c.room, c.category(), c.out,
		spellDefenceIdentity(c.casterChar, c.casterUser(), c.room),
		spellDefenceIdentity(tc, c.targetUser(), c.room), c.spell.Name, c.casterUser(), c.targetUser())
	if c.out.DefensiveCrit {
		dmg = 0
	} else {
		tc.ApplyHarm(characters.PoolHealth, dmg, c.casterRef())
		cancelDamageConditions(tc)
		// on_spell_hit item procs fire only on a harm hit that dealt damage;
		// the proc's own chance and cooldown pace an area cast.
		if dmg > 0 {
			dispatchItemProcs("on_spell_hit", c.casterChar, tc, nil, dmg)
		}
	}
	commitHarmfulSpellAggro(c, fresh)
	if c.out.Defended {
		return dmg // the defence triad above already told everyone
	}
	dmgDesc := combat.GetDamageDescription(dmg, tc.HealthMax.Value)
	messaging.SendTrio(messaging.Trio{
		Actor: messaging.Say(c.category(), fmt.Sprintf(
			`Your %s strikes %s! (<ansi fg="damage">%s</ansi>)%s`,
			c.spell.Name, c.targetName(), dmgDesc, c.critTag())),
		Actee: messaging.Say(c.category(), fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> strikes you! (<ansi fg="damage">%s</ansi>)%s`,
			c.casterName(), c.spell.Name, dmgDesc, c.critTag())),
		Observer: messaging.Say(c.category(), fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> strikes %s!`,
			c.casterName(), c.spell.Name, c.targetName())),
	}, c.audience())
	return dmg
}

// spellCasterStatAndSkill is the caster side of a spell's duration: the
// spell's own primarystat through CasterStatValue and the school's cast
// skill (Manifestation for that school, Spellcasting otherwise), the formula
// the mob-on-player dot always used. A nil caster reads stat 100 and skill 0,
// the old anonymous-caster default, which only tests reach.
//
// Not actions.GetSpellStatAndSkill: that one returns Perception, the FOLD
// stat, and would move every dot off its declared primarystat.
func spellCasterStatAndSkill(spell *spells.SpellData, caster *characters.Character) (stat int, skill int) {
	if caster == nil {
		return 100, 0
	}
	castSkill := skills.Spellcasting
	if spell.HasSchool(spells.SchoolManifestation) {
		castSkill = skills.Manifestation
	}
	return spell.CasterStatValue(caster.Stats), caster.GetSkillLevel(castSkill)
}

// applySpellDot is the one damage-over-time applier (slice 3a). The
// affliction is binary: it lands only on an attack win, and a defended cast
// narrates the defence triad and applies nothing. Either way the cast was an
// attack.
func applySpellDot(c spellEffectCtx) int {
	tc := c.targetChar()
	fresh := !tc.IsInCombat()
	if c.out.Defended {
		sendSpellChannelDefenceMessages(c.room, c.category(), c.out,
			spellDefenceIdentity(c.casterChar, c.casterUser(), c.room),
			spellDefenceIdentity(tc, c.targetUser(), c.room), c.spell.Name, c.casterUser(), c.targetUser())
		commitHarmfulSpellAggro(c, fresh)
		return 0
	}
	stat, skill := spellCasterStatAndSkill(c.spell, c.casterChar)
	dotDuration := calcSpellDuration(c.spell.BaseFolds, skill, stat) / 3
	if dotDuration < 3 {
		dotDuration = 3
	}
	// Condition 121 ticks every round, so dotDuration is the trigger count.
	// The record's negative magnitude is the harm per tick, floored at one.
	dotAmount := c.magnitude
	if dotAmount < 1 {
		dotAmount = 1
	}
	// Names are read BEFORE the condition lands: AddConditionMagnitude below
	// can add the "poisoned" adjective to the target's own rendered name, and
	// SendTrio's redaction must see the exact string the line prints.
	casterName, targetName := c.casterName(), c.targetName()
	// The character door, on purpose: an immune target refuses the record,
	// and nothing that did not happen may be narrated.
	afflicted := tc.AddConditionMagnitude(conditions.ConditionIdPoisoned, dotDuration, -float64(dotAmount), "spell") == nil
	commitHarmfulSpellAggro(c, fresh)
	if !afflicted {
		return 0
	}
	messaging.SendTrio(messaging.Trio{
		Actor: messaging.Say(c.category(), fmt.Sprintf(
			`Your %s afflicts %s!%s`, c.spell.Name, targetName, c.critTag())),
		Actee: messaging.Say(c.category(), fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> afflicts you!%s`, casterName, c.spell.Name, c.critTag())),
		Observer: messaging.Say(c.category(), fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> afflicts %s!`, casterName, c.spell.Name, targetName)),
	}, spellAudience(c.casterUser(), casterName, c.targetUser(), targetName, c.room))
	return 0
}

// applySpellKnockdown is the one knockdown applier (slice 3a). The defence
// scales the DAMAGE (a defended cast still lands a partial hit, a defensive
// crit negates it), while the knockdown is binary and lands only on an
// attack win: ExecuteSkillMove's Hit/StatusApplied split.
func applySpellKnockdown(c spellEffectCtx) int {
	tc := c.targetChar()
	fresh := !tc.IsInCombat()
	dmg := scaleSpellDamageByDefence(
		calcSpellDamageForCharacter(c.spell, c.casterChar, tc, c.magnitude, c.out.AttackerCrit), c.out)
	// Names are read before ApplyHarm and TransitionToSupine: a lethal hit
	// adds the "dead" adjective to the target's own rendered name, and
	// SendTrio's redaction must see the exact string the line prints.
	casterName, targetName := c.casterName(), c.targetName()
	if c.out.DefensiveCrit {
		dmg = 0
	} else {
		tc.ApplyHarm(characters.PoolHealth, dmg, c.casterRef())
		cancelDamageConditions(tc)
		if dmg > 0 {
			dispatchItemProcs("on_spell_hit", c.casterChar, tc, nil, dmg)
		}
	}
	// Spell knockdowns put the target on its back (Supine); a target already
	// grappled or down takes the damage but no knockdown is narrated.
	knocked := false
	if !c.out.Defended {
		knocked = true
		if err := tc.Position.TransitionToSupine(
			position.SupineData{MinRecoveryRounds: 1},
			state.TransitionReason{Trigger: position.TriggerKnockdownSpell},
		); err != nil {
			mudlog.Warn("applySpellKnockdown: TransitionToSupine failed", "target", c.targetRef(), "err", err)
			knocked = false
		}
	}
	commitHarmfulSpellAggro(c, fresh)
	sendSpellChannelDefenceMessages(c.room, c.category(), c.out,
		spellDefenceIdentity(c.casterChar, c.casterUser(), c.room),
		spellDefenceIdentity(tc, c.targetUser(), c.room), c.spell.Name, c.casterUser(), c.targetUser())
	if c.out.Defended {
		return dmg // the triad above already narrated it; a defended cast never knocks down
	}
	dmgDesc := combat.GetDamageDescription(dmg, tc.HealthMax.Value)
	if knocked {
		messaging.SendTrio(messaging.Trio{
			Actor: messaging.Say(c.category(), fmt.Sprintf(
				`Your %s slams %s to the ground! (<ansi fg="damage">%s</ansi>)%s`,
				c.spell.Name, targetName, dmgDesc, c.critTag())),
			Actee: messaging.Say(c.category(), fmt.Sprintf(
				`%s's <ansi fg="cyan">%s</ansi> slams you to the ground! (<ansi fg="damage">%s</ansi>)%s`,
				casterName, c.spell.Name, dmgDesc, c.critTag())),
			Observer: messaging.Say(c.category(), fmt.Sprintf(
				`%s's <ansi fg="cyan">%s</ansi> knocks %s to the ground!`,
				casterName, c.spell.Name, targetName)),
		}, spellAudience(c.casterUser(), casterName, c.targetUser(), targetName, c.room))
		return dmg
	}
	messaging.SendTrio(messaging.Trio{
		Actor: messaging.Say(c.category(), fmt.Sprintf(
			`Your %s strikes %s, but %s is already down. (<ansi fg="damage">%s</ansi>)%s`,
			c.spell.Name, targetName, targetName, dmgDesc, c.critTag())),
		Actee: messaging.Say(c.category(), fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> strikes you, but you're already down. (<ansi fg="damage">%s</ansi>)%s`,
			casterName, c.spell.Name, dmgDesc, c.critTag())),
		Observer: messaging.NoLine,
	}, spellAudience(c.casterUser(), casterName, c.targetUser(), targetName, c.room))
	return dmg
}

// ── Test-only wrappers. Slice 3b's last task deletes them once no test
// names them. ─────────────────────────────────────────────────────────────

// applyMobEffect applies a spell effect to a mob target. user may be nil
// (a mob caster); casterChar may be nil (an anonymous caster).
func applyMobEffect(user *users.UserRecord, casterChar *characters.Character, mob *mobs.Mob, room *rooms.Room,
	spellData *spells.SpellData, magnitude int, out combat.ChannelDefenceResult) int {
	return applySpellEffect(newSpellEffectCtx(casterChar, spellCasterActor(user, casterChar, room),
		actions.NewMobActorInRoom(mob, room), room, spellData, magnitude, out))
}

// applyPlayerEffect applies a player's spell effect to a player target.
func applyPlayerEffect(user *users.UserRecord, target *users.UserRecord, room *rooms.Room,
	spellData *spells.SpellData, magnitude int, out combat.ChannelDefenceResult) {
	applySpellEffect(newSpellEffectCtx(user.Character, actions.NewUserActorInRoom(user, room),
		actions.NewUserActorInRoom(target, room), room, spellData, magnitude, out))
}
