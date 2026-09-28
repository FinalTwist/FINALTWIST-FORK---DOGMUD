package hooks

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/targeting"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/textutil"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// calcSpellDuration computes a universal spell duration in rounds based on
// the spell's fold count, the caster's spellcasting skill, and willpower.
// Higher folds, skill, and willpower all extend duration.
// Formula: baseFolds × (10 + willpower/20 + spellcastingSkill/2)
func calcSpellDuration(baseFolds int, spellcastingSkill int, willpower int) int {
	if baseFolds < 1 {
		baseFolds = 4
	}
	duration := float64(baseFolds) * (10.0 + float64(willpower)/20.0 + float64(spellcastingSkill)/2.0)
	if duration < 10 {
		duration = 10
	}
	return int(math.Round(duration))
}

// resolveSpell is called when fold accumulation completes for a player caster.
// It dispatches to per-target resolution based on spell type and effect.
//
// Why this is NOT merged with resolveMobSpell:
//   - resolveSpell handles the "identify" spell type (no mob equivalent).
//   - HarmArea populates only mob targets for players; resolveMobSpell also
//     hits players in the room (mobs can cleave all occupants).
//   - HelpArea is player-only (mobs never cast area healing in this engine).
//   - Both target paths take the non-harm shortcut (AttackType ==
//     combatvocab.AttackNone); the mob path gained it in M4b-2.
//   - Post-resolution: player fires the onMagic script and consumes a
//     component; mob does neither.
//   - The per-target helpers (resolveAgainstMob vs resolveMobSpellAgainstMob,
//     resolveAgainstPlayer vs resolveMobSpellAgainstPlayer) have fundamentally
//     different signatures, messaging, and combat-record calls.
//
// Extracting the 6-line loop skeleton into a shared wrapper would require
// function-parameter callbacks or an interface, adding abstraction without
// meaningful savings. Keep them separate and well-documented instead.
// playerHarmTargetPermitted reports whether a player-cast spell of this type
// may land on mob right now.
//
// Spells fold over several rounds, so the target set chosen by InitiateCast is
// stale by the time the spell resolves: a mob can be charmed into a companion,
// or a builder can flag it protected, in between. Harmful spells therefore
// re-run the same authorization policy at resolution (review finding 3).
//
// Help spells are exempt — they legitimately target companions.
func playerHarmTargetPermitted(spellData *spells.SpellData, mob *mobs.Mob) bool {
	if spellData.IsHarm() {
		return !mobs.CheckPlayerHarm(mob).Blocked()
	}
	return true
}

func resolveSpell(user *users.UserRecord, cs activity.CastingData, spellData *spells.SpellData, room *rooms.Room) (anyLanded bool) {

	side := spellAttackSideFor(spellData, user.Character, combat.SightRoom(room))
	magnitude := spellData.EffectMagnitude

	// --- Identify: resolve against caster's item, no targets ---
	if spellData.EffectType == "identify" {
		resolveIdentify(user, cs.SpellRest, room)
		// Uncontested: there is no defence to beat, so it landed.
		return true
	}

	// --- Populate area targets for HarmArea ---
	if spellData.IsHarm() && spellData.Targeting == combatvocab.TargetArea {
		allMobs := room.GetMobs(rooms.FindAll)
		filtered := make([]int, 0, len(allMobs))
		for _, mId := range allMobs {
			// Spare companions, non-combatants and attack-immune mobs.
			if !playerHarmTargetPermitted(spellData, mobs.GetInstance(mId)) {
				continue
			}
			filtered = append(filtered, mId)
		}
		cs.TargetMobInstanceIds = filtered
	}

	// --- Populate area targets for HelpArea ---
	if !spellData.IsHarm() && spellData.Targeting == combatvocab.TargetArea {
		cs.TargetUserIds = room.GetPlayers(rooms.FindAll)
		// Apply to ally mobs only (charmed/companion). REPLACES any residual
		// TargetMobInstanceIds from the cast's pre-resolution step —
		// otherwise the caster's pre-spell aggro target (an enemy mob) gets
		// healed alongside intended allies. Symmetric with HarmArea above.
		allMobs := room.GetMobs(rooms.FindAll)
		allies := make([]int, 0, len(allMobs))
		for _, mId := range allMobs {
			if m := mobs.GetInstance(mId); m != nil && m.Character.IsCharmed() {
				allies = append(allies, mId)
			}
		}
		cs.TargetMobInstanceIds = allies
	}

	// --- Resolve against mob targets ---
	// castFumbled tracks whether ANY per-target roll fumbled (ZScore <= -2.0).
	// A fumble gates the post-target effects (summon, charm, Go hooks) below
	// so a summon-spell caster who fumbles doesn't still get the companion.
	castFumbled := false
	// anyLanded (the NAMED RETURN) drives U10b-1 Task 13's ONE progression
	// award for the cast. ONE CAST IS ONE RESOLVED ACTION: a three-target spell
	// that beat one defence pays a single full-weight event, not three events
	// and not one per target hit. A cast every target defended pays the failure
	// fraction.
	targetsResolved := 0
	for _, mobInstId := range cs.TargetMobInstanceIds {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil || mob.Character.Health < 1 {
			continue
		}
		if mob.Character.RoomId != room.RoomId {
			continue // target left the room before spell resolved
		}
		if !playerHarmTargetPermitted(spellData, mob) {
			continue // gained protection while the spell was folding
		}
		fumbled, landed := resolveAgainstMob(user, mob, room, spellData, side, magnitude)
		castFumbled = castFumbled || fumbled
		anyLanded = anyLanded || landed
		targetsResolved++
	}

	// --- Resolve against player targets ---
	for _, targetUserId := range cs.TargetUserIds {
		targetUser := users.GetByUserId(targetUserId)
		if targetUser == nil {
			continue
		}
		if targetUser.Character.RoomId != room.RoomId {
			user.SendText(messaging.CategorySpellDisruption, fmt.Sprintf(`Your spell dissipates, unspent. <ansi fg="username">%s</ansi> is no longer here.`, targetUser.Character.Name))
			continue // target left the room before spell resolved
		}
		// Skip downed players for harm spells — they're already down.
		if targetUser.Character.Health < 1 && spellData.IsHarm() {
			continue
		}
		if spellData.AttackType == combatvocab.AttackNone {
			// Non-harm cast: uncontested, an attack win by construction.
			// Uncontested means it LANDED: there was no defence to beat.
			applySpellEffect(newSpellEffectCtx(user.Character, actions.NewUserActorInRoom(user, room), actions.NewUserActorInRoom(targetUser, room), room, spellData, magnitude, combat.ChannelDefenceResult{DamageMultiplier: 1}))
			anyLanded = true
		} else {
			fumbled, landed := resolveAgainstPlayer(user, targetUser, room, spellData, side, magnitude)
			castFumbled = castFumbled || fumbled
			anyLanded = anyLanded || landed
		}
		targetsResolved++
	}

	// --- Empty room / no valid targets feedback ---
	// Summons target nothing in the room, so the generic line does not apply.
	isSummon := spellData != nil && spellData.SummonMobId > 0
	isCharm := spellData != nil && spellData.EffectType == "charm"
	if targetsResolved == 0 && !isSummon {
		if isCharm {
			// Charm used to resolve AFTER this loop, reading
			// TargetMobInstanceIds[0] directly, so it did not care whether the
			// target survived the fold. Now that it resolves inside the loop it
			// does, and a 36-fold channel that finds nothing must still say so
			// -- silence after spending 120 conviction reads as a broken
			// command. The wording is charm's because "erupts outward" suits a
			// blast, not a held gaze.
			user.SendText(messaging.CategorySpellDisruption,
				`Your gaze finds nothing to hold, and the will you gathered scatters.`)
		} else {
			user.SendText(messaging.CategorySpellDisruption, `Your spell erupts outward but finds no targets.`)
			sendVisualRoomText(room, messaging.CategorySpellDisruption, fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s spell crackles through the air harmlessly.`,
				user.Character.Name), user.UserId)
		}
	}

	// --- Run spell script onMagic (if present) ---
	// Send YAML magic text (if defined).
	if spellData != nil && spellData.Narration(spells.PhaseMagic).Len() > 0 {
		tCtx := textutil.TokenContext{
			ActorName:      user.Character.GetCharacterName(true),
			ActorPlainName: user.Character.GetCharacterName(false),
		}
		if len(cs.TargetUserIds) > 0 {
			if tUser := users.GetByUserId(cs.TargetUserIds[0]); tUser != nil {
				tCtx.ActeeName = tUser.Character.GetCharacterName(true)
				tCtx.ActeePlainName = tUser.Character.GetCharacterName(false)
			}
		} else if len(cs.TargetMobInstanceIds) > 0 {
			if tMob := mobs.GetInstance(cs.TargetMobInstanceIds[0]); tMob != nil {
				tCtx.ActeeName = tMob.Character.GetCharacterName(true)
				tCtx.ActeePlainName = tMob.Character.GetCharacterName(false)
			}
		}
		roles := spellData.Narrate(spells.PhaseMagic, tCtx)
		if roles.Actor != "" {
			user.SendText(spellSchoolCategory(spellData), roles.Actor)
		}
		// Audio channel, as before this refactor: filed, not changed here.
		if roles.Observer != "" {
			if r := rooms.LoadRoom(user.Character.RoomId); r != nil {
				r.SendText(spellSchoolCategory(spellData), roles.Observer, user.UserId)
			}
		}
	}
	// Fumble gate for the post-target effects (summon / charm / Go hooks).
	// A fumbled cast consumed conviction + component but should NOT also land
	// the primary effect. A single flavor message; individual blocks skip
	// silently so we don't spam the player.
	if castFumbled && spellData != nil &&
		(spellData.SummonMobId > 0 || spellData.EffectType == "charm" ||
			cs.SpellId == "fold-anchor" || cs.SpellId == "fold-recall" || cs.SpellId == "purge-affliction") {
		user.SendText(messaging.CategorySpellDisruption, `<ansi fg="red">The weave unravels — the spell fails to take shape.</ansi>`)
	}

	// Resolve companion summon (if configured)
	if !castFumbled && spellData != nil && spellData.SummonMobId > 0 {
		resolveCompanionSummon(user, spellData, cs.SpellRest, room)
	}
	// Charm used to resolve HERE, in a second private contest run after the
	// loop above had already contested every target and thrown the verdict
	// away. It now resolves inside the loop, in applyMobEffect's "charm" arm,
	// off that one contest.
	//
	// Removing this block also fixes two live defects. The player no longer
	// sees a resist line and a success line for the same cast. And charm no
	// longer succeeds against a mob that died, left the room, or gained harm
	// protection mid-fold: this block read TargetMobInstanceIds[0] directly and
	// so ignored every filter the loop applies.

	// --- Go spell hooks — dispatch before JS scripts ---
	// Fumble aborts the hook body but falls through to the component-consume
	// block below so the catalyst is still used up.
	if !castFumbled {
		switch cs.SpellId {
		case "fold-anchor":
			resolveFoldAnchor(actions.NewUserActorInRoom(user, room))
			// Uncontested utility cast: no defence to beat.
			return true
		case "fold-recall":
			resolveFoldRecall(actions.NewUserActorInRoom(user, room))
			// Uncontested utility cast: no defence to beat.
			return true
		case "purge-affliction":
			// A mob target is read here for the same reason the other help
			// spells read it: a charmed companion is a legitimate target, and
			// this switch used to fall through to the self-cast arm whenever
			// only TargetMobInstanceIds was set, so naming a poisoned
			// companion purged the CASTER.
			//
			// Both named-target arms go through purgeTarget.stillPresent,
			// which mirrors the target loops' admission above: a target that
			// died or left mid-fold is neither purged nor narrated, and the
			// loop's own "finds no targets" line is all the caster reads. A
			// failed check must fall through to NOTHING, not to the self-cast
			// arm below.
			switch {
			case len(cs.TargetUserIds) > 0:
				if targetUser := users.GetByUserId(cs.TargetUserIds[0]); targetUser != nil {
					t := purgeTarget{char: targetUser.Character, user: targetUser, name: targetUser.Character.Name}
					if t.stillPresent(room, false) {
						resolvePurgeAffliction(user, room, t)
					}
				}
			case len(cs.TargetMobInstanceIds) > 0:
				if tMob := mobs.GetInstance(cs.TargetMobInstanceIds[0]); tMob != nil {
					t := purgeTarget{char: &tMob.Character, name: tMob.Character.Name, display: mobDisplayName(tMob, room, user.UserId)}
					if t.stillPresent(room, true) {
						resolvePurgeAffliction(user, room, t)
					}
				}
			default:
				resolvePurgeAffliction(user, room, purgeTarget{char: user.Character, user: user, name: user.Character.Name}) // self-cast
			}
			// Uncontested utility cast: no defence to beat.
			return true
		}
	}

	// --- Consume component if required ---
	if spellData.ComponentTag != "" {
		consumeSpellComponent(user, spellData.ComponentTag)
	}

	return anyLanded
}

// runSpellChannelAttack is THE spell-contest seam (U6b Task 4): every spell
// resolver — player-cast and mob-cast — runs its ONE contest through it and
// threads the ChannelDefenceResult into the effect appliers, which consume it
// instead of rolling their own. It defaults to the canonical resolver;
// same-package dispatch tests replace it briefly with a literal outcome and
// restore it with t.Cleanup. Tests that need the seam's REAL side effects
// (cost admission, the progression bonus tier) leave this alone and swap the
// contest core via combat.SetChannelAttackContestRunnerForTest instead.
var runSpellChannelAttack = combat.ResolveChannelAttack

// spellAttackSideFor builds the caster's half of the one spell contest. The
// hit contest finally honours the spell's U9 primarystat: the score is the
// spell's own casting stat plus the school's governing skill, weighted by
// SkillWeight — the deleted hit-gate helper multiplied the weighted skill by
// a config skill factor (x3) on top, the x15-per-rank outlier U6b removes.
//
// StatName mirrors CasterStatValue's default: an empty primarystat reads as
// willpower there, so the progression events must name willpower too, not "".
//
// room is the cast's room (lighting plan 5b): the caster must see to aim, so
// Mult carries the sight row of the situational table. Nil is unity.
func spellAttackSideFor(spellData *spells.SpellData, casterChar *characters.Character, room messaging.RoomVisibility) combat.AttackSide {
	castSkill := skills.Spellcasting
	if spellData.HasSchool(spells.SchoolManifestation) {
		castSkill = skills.Manifestation
	}
	statName := spellData.PrimaryStat
	if statName == "" {
		statName = "willpower"
	}
	return combat.AttackSide{
		Stat:      spellData.CasterStatValue(casterChar.Stats),
		StatName:  statName,
		Skill:     castSkill,
		SkillRank: casterChar.GetSkillLevel(castSkill),
		// Task 17: composed with the shared situational layer. Prone and
		// stamina are 1.0 on both spell channels by the declared table: you
		// cast fine from the ground, and the conviction-depletion penalty is
		// already applied in the damage term (calcSpellDamageForCharacter),
		// so it must not reach accuracy a second time here. The sight row
		// (lighting plan 5b) does apply. ForceCrit is per-target and set by
		// each resolveAgainst* call site.
		Mult: combat.SituationalAttackMult(casterChar, room, spellData.Attack()),
	}
}

// scaleSpellDamageByDefence applies the threaded contest's damage multiplier:
// 1.0 on an attack win, 0.0 on a defensive crit, 0.0-0.5 on a rolled
// defensive win, exactly 0.5 on a floored save — the same semantics
// ExecuteSkillMove documents. A defended hit deals at least 1 damage unless
// the defence critted.
func scaleSpellDamageByDefence(dmg int, out combat.ChannelDefenceResult) int {
	mult := out.DamageMultiplier
	if mult >= 1.0 {
		return dmg
	}
	dmg = int(math.Round(float64(dmg) * mult))
	if dmg < 1 && mult > 0 {
		dmg = 1
	}
	return dmg
}

// resolveAgainstMob runs the ONE channel contest and applies the effect to a
// mob. Returns true if the cast fumbled (the seam's self-relative
// AttackerFumble). A fumble aborts any post-target spell effects (summon,
// charm, Go hooks) in the caller's main flow; component consumption still
// fires (the failed binding uses up the catalyst regardless).
// landed reports that this target's contest was WON outright -- the caster's
// roll beat the defence. It is the spell channel's equivalent of melee's
// CleanHit, and U10b-1 Task 13 uses it to decide whether the cast's ONE
// progression award pays full weight or the failure fraction.
//
// A DEFENDED cast is not landed even though it still deals partial damage on
// the shared mitigation curve, matching SkillMoveResult.Hit's contract. A
// fumble is not landed either: it aborts before success.
func resolveAgainstMob(user *users.UserRecord, mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, side combat.AttackSide, magnitude int) (fumbled bool, landed bool) {
	caster := actions.NewUserActorInRoom(user, room)
	target := actions.NewMobActorInRoom(mob, room)

	// Non-harm cast at a mob (a heal on your companion, an area mend over
	// allies): uncontested, exactly as the player-target loop has always
	// treated it. On master this ran a quell contest, so a companion could
	// "defend" its own heal, a fumble backfired on the caster, and a
	// defensive crit earned the companion a counter-swing at its owner.
	// The empty eligible set would already skip the contest; the explicit
	// shortcut makes the rule visible and independent of that detail.
	// BEHAVIOUR CHANGE from master, own commit.
	if spellData.AttackType == combatvocab.AttackNone {
		// Every reachable non-harm arm (heal, condition, default) returns 0
		// today, so threading it through is not a behaviour change; it just
		// stops the record silently pinning itself to 0 if a future arm
		// starts reporting a real amount (an area mend's total, say).
		c := newSpellEffectCtx(user.Character, caster, target, room, spellData, magnitude,
			combat.ChannelDefenceResult{DamageMultiplier: 1})
		recordSpellResolution(c, applySpellEffect(c))
		return false, true
	}

	// Task 17: the sleeping-victim forced crit reaches the spell channel.
	side.ForceCrit = combat.SleepingForceCrit(&mob.Character)

	// Charm alone carries an in-combat penalty (spec 4.1). A mind braced for
	// violence is harder to reach, and one braced against YOU is hardest.
	//
	// Normalise Mult FIRST. Zero is the zero value and AttackSide.score() reads
	// it as "unset, 1.0" (defence_multiplier.go:78-83), so multiplying into an
	// unset Mult yields 0, which reads back as 1.0 -- the penalty would vanish
	// silently while producing an entirely plausible number.
	if spellData.EffectType == "charm" {
		if side.Mult == 0 {
			side.Mult = 1.0
		}
		side.Mult *= charmInCombatMult(&mob.Character, user.UserId)
	}
	out := runSpellChannelAttack(combat.SightRoom(room), spellData.Attack(), side, user.Character, &mob.Character)
	c := newSpellEffectCtx(user.Character, caster, target, room, spellData, magnitude, out)

	// Backfire on fumble, resolved BEFORE success per the seam's contract: a
	// fumbled cast aborts even a winning roll.
	if out.AttackerFumble {
		applySpellBackfire(c)
		return true, false
	}

	// Boss-interrupt, for every pairing (interruptSpellTarget).
	interruptSpellTarget(c)

	recordSpellResolution(c, applySpellEffect(c))

	// U6b Task 10: the MOB defender's crit defence counters the player caster.
	fireSpellCounterTier(room, out, spellData.Attack(),
		&mob.Character, user.Character, nil, user)

	return false, !out.Defended
}

// spellSchoolCategory picks the messaging Category from a spell's
// first declared school. Falls back to CategorySpellElemental if the
// spell has no school tag — the historical default for damage spells.
// A spell with multiple schools (rare) uses the first; the school
// list order in YAML is the author's preference.
func spellSchoolCategory(spellData *spells.SpellData) messaging.Category {
	if spellData == nil || len(spellData.Schools) == 0 {
		return messaging.CategorySpellElemental
	}
	switch spellData.Schools[0] {
	case spells.SchoolElemental:
		return messaging.CategorySpellElemental
	case spells.SchoolEnhancement:
		return messaging.CategorySpellEnhancement
	case spells.SchoolMental:
		return messaging.CategorySpellMental
	case spells.SchoolVital:
		return messaging.CategorySpellVital
	case spells.SchoolManifestation:
		return messaging.CategorySpellManifestation
	}
	return messaging.CategorySpellElemental
}

// sendSpellChannelDefenceMessages renders one canonical defence triad and
// applies the spell path's existing visual audience routing. Nil user records
// represent mob participants, which do not receive private player text.
func sendSpellChannelDefenceMessages(room *rooms.Room, category messaging.Category,
	out combat.ChannelDefenceResult, attackerName, defenderName, attackName string,
	attackerUser, defenderUser *users.UserRecord, indexOverride ...int) {
	if defenderUser != nil {
		if text := combat.ChannelDefenceShortageText(out, defenderUser.Character); text != "" {
			defenderUser.SendText(messaging.CategorySystem, text)
		}
	}
	triad := combat.RenderChannelDefenceMessages(out, combat.ChannelDefenceIdentities{
		Attacker: attackerName,
		Defender: defenderName,
	}, attackName, indexOverride...)
	if triad.ToRoom == "" {
		return
	}
	// Through the seam: a participant who cannot see the other party reads
	// "something" in place of their name. SendTextVisualToUser used to drop
	// these lines entirely, so a defender in the dark was never told they had
	// defended at all.
	messaging.SendTrio(messaging.Trio{
		Actor:    messaging.Say(category, string(triad.ToAttacker)),
		Actee:    messaging.Say(category, string(triad.ToDefender)),
		Observer: messaging.Say(category, string(triad.ToRoom)),
	}, spellAudience(attackerUser, attackerName, defenderUser, defenderName, room))
}

// spellDefenceIdentity returns the display-ready identity for either kind of
// spell participant. Mob identities retain the room's duplicate index.
func spellDefenceIdentity(char *characters.Character, user *users.UserRecord, room *rooms.Room) string {
	if char == nil {
		return ""
	}
	if user != nil {
		return char.GetPlayerName(user.UserId).String()
	}
	if room != nil && char.MobInstanceId > 0 {
		if mob := mobs.GetInstance(char.MobInstanceId); mob != nil {
			return mobDisplayName(mob, room, 0)
		}
	}
	return char.GetMobName(0).String()
}

// setMobSpellAggro sets reciprocal aggro between the caster and the
// mob target immediately after a hostile spell lands.
//
// Note: applyMobEffect_condition does NOT call this helper — its aggro block
// is gated on spell Type being Harm*. Kept inline there.
func setMobSpellAggro(user *users.UserRecord, mob *mobs.Mob) {
	if !mob.Character.IsInCombat() {
		if user != nil {
			targeting.Commit(&mob.Character, state.ActorRef{UserId: user.UserId}, targeting.ReasonAttack)
		}
	}
	if user != nil && !user.Character.IsInCombat() {
		targeting.Commit(user.Character, state.ActorRef{MobInstanceId: mob.InstanceId}, targeting.ReasonAttack)
	}
}

func applyMobEffect_condition(
	user *users.UserRecord,
	casterChar *characters.Character,
	mob *mobs.Mob,
	room *rooms.Room,
	spellData *spells.SpellData,
	out combat.ChannelDefenceResult,
	critTag string,
	mName string,
) int {
	// U6b Task 4: a condition is a binary status — a defended cast narrates the
	// channel defence triad and applies nothing. Hostile intent still aggros
	// (the harm-type gate below is shared with the landed path).
	if out.Defended {
		sendSpellChannelDefenceMessages(room, spellSchoolCategory(spellData), out,
			spellDefenceIdentity(casterChar, user, room), mName, spellData.Name, user, nil)
		if spellData.IsHarm() {
			setMobSpellAggro(user, mob)
		}
		return 0
	}
	for _, conditionId := range spellData.ConditionIds {
		applySpellCondition(mob, spellData, casterChar, conditionId)
	}
	// Conditional aggro for harmful condition spells — kept inline because it is
	// gated on Harm* spell types; not consolidated in Task 7's setMobSpellAggro.
	if spellData.IsHarm() {
		if !mob.Character.IsInCombat() {
			if user != nil {
				targeting.Commit(&mob.Character, state.ActorRef{UserId: user.UserId}, targeting.ReasonAttack)
			}
		}
		if user != nil && !user.Character.IsInCombat() {
			targeting.Commit(user.Character, state.ActorRef{MobInstanceId: mob.InstanceId}, targeting.ReasonAttack)
		}
	}
	if user != nil {
		user.SendText(spellSchoolCategory(spellData), fmt.Sprintf(
			`Your %s takes effect on %s!%s`,
			spellData.Name, mName, critTag))
		sendVisualRoomText(room, spellSchoolCategory(spellData), fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> affects %s!`,
			user.Character.Name, spellData.Name, mName), user.UserId)
	}
	return 0
}

// applyMobEffect_heal handles the "heal" EffectType case for applyMobEffect —
// a caster (mob or player) casting a HelpSingle heal at ANOTHER mob (e.g. an
// ally construct healing a boss, or a player healing a charmed companion).
// Prior to Chunk B of the crash-site boss-mechanics work this case did not
// exist: applyMobEffect's switch only handled damage/dot/knockdown/condition, so
// a mob-to-mob (or player-to-companion) "heal" cast silently fell through to
// applyMobEffect_default and did nothing. Mirrors applyMobSelfEffect's
// "heal" case (percentage-of-max regen via the Regenerating record) but targets
// `mob` instead of the caster. Returns 0 (no damage dealt) to match the
// applyMobEffect_* int-return convention.
func applyMobEffect_heal(
	casterChar *characters.Character,
	mob *mobs.Mob,
	room *rooms.Room,
	spellData *spells.SpellData,
	magnitude int,
	mName string,
) int {
	skillLevel := 0
	willpower := 0
	casterName := "Something"
	if casterChar != nil {
		skillLevel = casterChar.GetSkillLevel(skills.Spellcasting)
		willpower = spellData.CasterStatValue(casterChar.Stats)
		casterName = casterChar.Name
	}
	regenMult := float64(magnitude)
	if regenMult < 1.0 {
		regenMult = 1.0
	}
	durationRounds := calcSpellDuration(spellData.BaseFolds, skillLevel, willpower) / 2
	if durationRounds < 6 {
		durationRounds = 6
	}
	_ = mob.Character.AddConditionMagnitude(conditions.ConditionIdRegenerating, durationRounds, regenMult, "heal spell")
	sendVisualRoomText(room, messaging.CategorySpellVital, fmt.Sprintf(
		`<ansi fg="cyan">%s</ansi>'s %s washes over %s, knitting wounds shut.`,
		casterName, spellData.Name, mName))
	return 0
}

func applyMobEffect_default(
	user *users.UserRecord,
	casterChar *characters.Character,
	room *rooms.Room,
	spellData *spells.SpellData,
	out combat.ChannelDefenceResult,
	mName string,
) int {
	if out.Defended {
		sendSpellChannelDefenceMessages(room, spellSchoolCategory(spellData), out,
			spellDefenceIdentity(casterChar, user, room), mName, spellData.Name, user, nil)
		return 0
	}
	// A spell whose narration resolveSpell's Go hook owns gets no generic
	// line: the player-target twin in applyPlayerEffect's default arm skips
	// it too. A help spell aimed at a charmed companion lands here.
	if user != nil && !spellNarratedByGoHook(spellData.SpellId) {
		user.SendText(spellSchoolCategory(spellData), fmt.Sprintf(
			`Your %s takes effect on %s.`,
			spellData.Name, mName))
	}
	return 0
}

// applyMobEffectArms is the pre-unification switch for a MOB target (PM and
// MM). The dispatcher routes here every effect that has no unified applier
// yet. user is nil when a mob casts; casterChar is nil for an anonymous
// caster.
//
// U6b Task 4: `out` is the resolver's ONE channel contest, threaded through.
func applyMobEffectArms(c spellEffectCtx) int {
	user, casterChar, mob, room := c.casterUser(), c.casterChar, c.targetMob(), c.room
	spellData, magnitude, out := c.spell, c.magnitude, c.out
	critTag := ""
	if out.AttackerCrit {
		critTag = ` <ansi fg="yellow">[CRIT!]</ansi>`
	}
	viewerId := 0
	if user != nil {
		viewerId = user.UserId
	}
	mName := mobDisplayName(mob, room, viewerId)

	switch spellData.EffectType {
	case "condition":
		return applyMobEffect_condition(user, casterChar, mob, room, spellData, out, critTag, mName)
	case "heal":
		if user != nil {
			events.AddToQueue(events.Healed{HealerUserId: user.UserId, MobInstanceId: mob.InstanceId})
		}
		return applyMobEffect_heal(casterChar, mob, room, spellData, magnitude, mName)
	case "charm":
		// Charm resolves HERE, off the contest this cast already ran, rather
		// than in a second private contest after the target loop. See
		// applyMobEffect_charm.
		return applyMobEffect_charm(user, mob, room, spellData, out, mName)
	default:
		return applyMobEffect_default(user, casterChar, room, spellData, out, mName)
	}
}

// resolveAgainstPlayer runs the ONE channel contest and applies the effect to
// a player. Returns true if the cast fumbled (the seam's self-relative
// AttackerFumble). See resolveAgainstMob for the fumble semantics carrying
// over to summon/charm/Go-hook gating.
//
// Crit-received toughening for the defender now fires INSIDE the seam's bonus
// tier (combat.ResolveChannelAttack -> awardChannelDefenceBonus), which is why
// there is no direct ApplyProgression call here any more — the U9-era block
// this function used to carry became a duplicate the moment the seam saw the
// crit, and the once-per-round dedupe would have masked the double-fire
// rather than prevented it.
func resolveAgainstPlayer(user *users.UserRecord, target *users.UserRecord, room *rooms.Room, spellData *spells.SpellData, side combat.AttackSide, magnitude int) (fumbled bool, landed bool) {

	// Task 17: the sleeping-victim forced crit reaches the spell channel.
	side.ForceCrit = combat.SleepingForceCrit(target.Character)
	out := runSpellChannelAttack(combat.SightRoom(room), spellData.Attack(), side, user.Character, target.Character)
	c := newSpellEffectCtx(user.Character, actions.NewUserActorInRoom(user, room),
		actions.NewUserActorInRoom(target, room), room, spellData, magnitude, out)

	// Backfire on fumble, resolved BEFORE success per the seam's contract.
	if out.AttackerFumble {
		applySpellBackfire(c)
		return true, false
	}

	interruptSpellTarget(c)

	recordSpellResolution(c, applySpellEffect(c))

	// Set reciprocal aggro for harm spells. The harmful appliers commit their
	// own; this still serves harmful condition spells until slice 3b.
	if spellData.IsHarm() {
		if !user.Character.IsInCombat() {
			targeting.Commit(user.Character, state.ActorRef{UserId: target.UserId}, targeting.ReasonAttack)
		}
		if !target.Character.IsInCombat() {
			targeting.Commit(target.Character, state.ActorRef{UserId: user.UserId}, targeting.ReasonAttack)
		}
	}

	// U6b Task 10: the defending player's crit defence counters the caster.
	fireSpellCounterTier(room, out, spellData.Attack(),
		target.Character, user.Character, target, user)

	return false, !out.Defended
}

// applyPlayerEffectArms is the pre-unification switch for a player caster
// and a PLAYER target (PP). The dispatcher routes here every effect that has
// no unified applier yet.
//
// U6b Task 4: `out` is the resolver's ONE channel contest, threaded through
// (help spells with no defense pass an uncontested attack win). Non-damage
// effects are binary statuses: a defended cast narrates the channel defence
// triad and applies nothing, mirroring ExecuteSkillMove's StatusApplied split.
func applyPlayerEffectArms(c spellEffectCtx) {
	user, target, room := c.casterUser(), c.targetUser(), c.room
	spellData, magnitude, out := c.spell, c.magnitude, c.out

	critTag := ""
	if out.AttackerCrit {
		critTag = ` <ansi fg="yellow">[CRIT!]</ansi>`
	}

	if out.Defended {
		sendSpellChannelDefenceMessages(room, spellSchoolCategory(spellData), out,
			spellDefenceIdentity(user.Character, user, room),
			spellDefenceIdentity(target.Character, target, room), spellData.Name, user, target)
		return
	}

	switch spellData.EffectType {
	case "purge":
		target.Character.CancelConditionsWithFlag(conditions.Poison)
		if target.UserId != user.UserId {
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
					`<ansi fg="green">Your %s cleanses <ansi fg="username">%s</ansi> of afflictions.%s</ansi>`,
					spellData.Name, target.Character.Name, critTag)),
				Actee: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
					`<ansi fg="green"><ansi fg="username">%s</ansi>'s %s purges the toxins from your body.</ansi>`,
					user.Character.Name, spellData.Name)),
				Observer: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
					`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> cleanses <ansi fg="username">%s</ansi>.`,
					user.Character.Name, spellData.Name, target.Character.Name)),
			}, spellAudience(user, user.Character.Name, target, target.Character.Name, room))
		} else {
			// SELF-CAST: one line to the caster and one to the room, naming them
			// once, the shape case "shield" below already has. An area spell
			// puts the caster in its own target list (resolveSpell), so every
			// Cleansing Wave reaches this branch, not only a deliberate self-cast.
			// critTag stays on the caster's line so a crit on yourself is not lost.
			// The room line goes through SendTrio (messaging M4d PR 3 Task 3) so
			// a shapes-only observer reads "a figure" for the caster instead of
			// the name; target == user here, so there is no Actee.
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
					`<ansi fg="green">You purge the afflictions from your body.%s</ansi>`, critTag)),
				Actee: messaging.NoLine,
				Observer: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
					`<ansi fg="cyan">%s</ansi> cleanses <ansi fg="username">%s</ansi> of afflictions.`,
					spellData.Name, target.Character.Name)),
			}, messaging.Audience{
				Actor:     user,
				ActorId:   user.UserId,
				ActorName: user.Character.Name,
				ActeeName: messaging.NoName,
				Room:      room,
			})
		}

	case "heal":
		skillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
		// Magnitude from YAML is the regen multiplier (e.g. 3 = 3x base regen)
		regenMult := float64(magnitude)
		if regenMult < 1.0 {
			regenMult = 1.0
		}
		if out.AttackerCrit {
			// Crit: boost the multiplier portion above 1x by 2x
			regenMult = 1.0 + (regenMult-1.0)*2.0
		}
		durationRounds := calcSpellDuration(spellData.BaseFolds, skillLevel, spellData.CasterStatValue(user.Character.Stats)) / 2
		if durationRounds < 6 {
			durationRounds = 6
		}
		_ = target.Character.AddConditionMagnitude(conditions.ConditionIdRegenerating, durationRounds, regenMult, "heal spell")
		if target.UserId != user.UserId {
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
					`<ansi fg="green">You weave restorative magic around <ansi fg="username">%s</ansi>.%s</ansi>`,
					target.Character.Name, critTag)),
				Actee: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
					`<ansi fg="green"><ansi fg="username">%s</ansi>'s %s envelops you in healing energy. Your wounds begin to mend.</ansi>`,
					user.Character.Name, spellData.Name)),
				Observer: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
					`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> envelops <ansi fg="username">%s</ansi> in healing light.`,
					user.Character.Name, spellData.Name, target.Character.Name)),
			}, spellAudience(user, user.Character.Name, target, target.Character.Name, room))
		} else {
			// SELF-CAST: see case "purge". The room line reuses the wording
			// applyMobSelfEffect already uses for a mob healing itself, and now
			// goes through SendTrio the same way.
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
					`<ansi fg="green">A warm glow of healing magic envelops you. Your wounds begin to mend.%s</ansi>`, critTag)),
				Actee: messaging.NoLine,
				Observer: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
					`<ansi fg="username">%s</ansi> channels restorative magic.`,
					user.Character.Name)),
			}, messaging.Audience{
				Actor:     user,
				ActorId:   user.UserId,
				ActorName: user.Character.Name,
				ActeeName: messaging.NoName,
				Room:      room,
			})
		}

	case "condition":
		for _, conditionId := range spellData.ConditionIds {
			applySpellCondition(target, spellData, user.Character, conditionId)
		}
		// M1 audit defect: this case told the caster and the target and left
		// the room out, while its sibling `case "heal":` above broadcasts. A
		// spell visibly taking hold on someone is not a private exchange.
		// Shape and exclusions mirror the heal line; the category follows this
		// case's own two lines rather than heal's, because a condition is not
		// necessarily vital magic.
		//
		// KNOWN AND DEFERRED: the condition's own start text ALSO narrates this
		// moment to the target and the room, through the event AddCondition queues
		// above, so a condition with authored start text reaches each audience
		// twice. The messaging arc's M6 merges them into one line per audience.
		// See docs/superpowers/specs/completed/2026-09-11-messaging-m3-item5a-narration-defects-design.md.
		if target.UserId != user.UserId {
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.Say(spellSchoolCategory(spellData), fmt.Sprintf(
					`Your %s takes effect on <ansi fg="username">%s</ansi>!%s`,
					spellData.Name, target.Character.Name, critTag)),
				Actee: messaging.Say(spellSchoolCategory(spellData), fmt.Sprintf(
					`<ansi fg="username">%s</ansi>'s %s takes effect on you!`,
					user.Character.Name, spellData.Name)),
				Observer: messaging.Say(spellSchoolCategory(spellData), fmt.Sprintf(
					`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> settles over <ansi fg="username">%s</ansi>.`,
					user.Character.Name, spellData.Name, target.Character.Name)),
			}, spellAudience(user, user.Character.Name, target, target.Character.Name, room))
		} else {
			// SELF-CAST: see case "purge". The caster line stays, reworded,
			// rather than being dropped: a condition with no authored start text
			// would otherwise leave a self-caster reading nothing at all.
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.Say(spellSchoolCategory(spellData), fmt.Sprintf(
					`Your %s takes effect.%s`, spellData.Name, critTag)),
				Actee: messaging.NoLine,
				Observer: messaging.Say(spellSchoolCategory(spellData), fmt.Sprintf(
					`<ansi fg="cyan">%s</ansi> settles over <ansi fg="username">%s</ansi>.`,
					spellData.Name, target.Character.Name)),
			}, messaging.Audience{
				Actor:     user,
				ActorId:   user.UserId,
				ActorName: user.Character.Name,
				ActeeName: messaging.NoName,
				Room:      room,
			})
		}

	case "shield":
		skillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
		weightedSkill := int(math.Round(float64(skillLevel) * float64(configs.GetBalanceConfig().SkillWeight)))
		shieldBonus := (spellData.CasterStatValue(user.Character.Stats) + weightedSkill) / 3
		if shieldBonus < 1 {
			shieldBonus = 1
		}
		// Scale shield strength by spell magnitude (100 = 1.0x baseline)
		if magnitude > 0 {
			shieldBonus = int(math.Round(float64(shieldBonus) * float64(magnitude) / 100.0))
			if shieldBonus < 1 {
				shieldBonus = 1
			}
		}
		duration := calcSpellDuration(spellData.BaseFolds, skillLevel, spellData.CasterStatValue(user.Character.Stats))
		if out.AttackerCrit {
			shieldBonus = int(float64(shieldBonus) * 1.5)
		}
		_ = target.Character.AddConditionMagnitude(conditions.ConditionIdMinorShield, duration, float64(shieldBonus), "spell")
		if target.UserId != user.UserId {
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.Say(spellSchoolCategory(spellData), fmt.Sprintf(
					`A shimmering magical barrier forms around <ansi fg="username">%s</ansi>, bolstering their defenses.`,
					target.Character.Name)),
				Actee: messaging.Say(spellSchoolCategory(spellData),
					`A shimmering magical barrier forms around you, bolstering your defenses.`),
				Observer: messaging.Say(spellSchoolCategory(spellData), fmt.Sprintf(
					`A shimmering barrier surrounds <ansi fg="username">%s</ansi>.`, target.Character.Name)),
			}, spellAudience(user, user.Character.Name, target, target.Character.Name, room))
		} else {
			// SELF-CAST: the caster is the target, so there is no third-person
			// line to send them, and the room line excludes them. It now goes
			// through SendTrio, same as the other three self-cast branches above.
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.Say(spellSchoolCategory(spellData),
					`A shimmering magical barrier forms around you, bolstering your defenses.`),
				Actee: messaging.NoLine,
				Observer: messaging.Say(spellSchoolCategory(spellData), fmt.Sprintf(
					`A shimmering barrier surrounds <ansi fg="username">%s</ansi>.`, target.Character.Name)),
			}, messaging.Audience{
				Actor:     target,
				ActorId:   target.UserId,
				ActorName: target.Character.Name,
				ActeeName: messaging.NoName,
				Room:      room,
			})
		}

	default:
		// A spell whose narration resolveSpell's Go hook owns gets no generic
		// line here. fold-anchor, fold-recall and purge-affliction declare no
		// effect_type, so they land in this arm, and the caster was told
		// "Your Purge Affliction takes effect." before the hook said it
		// properly.
		if spellNarratedByGoHook(spellData.SpellId) {
			break
		}
		if target.UserId == user.UserId {
			// SELF-CAST: a single line to the caster, no room broadcast at all.
			// The purge/heal/condition/shield self-cast branches above each
			// pair a safe caster line with a room line that names the caster
			// (target.Character.Name, since target == user here); those room
			// lines now go through SendTrio too (messaging M4d PR 3 Task 3), so
			// a shapes-only observer reads "a figure" instead of the name. This
			// line has no such pairing, so no second party, and Actee/ActeeName
			// are unset.
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.Say(spellSchoolCategory(spellData), fmt.Sprintf(
					`Your %s takes effect.`, spellData.Name)),
				Actee:    messaging.NoLine,
				Observer: messaging.NoLine,
			}, messaging.Audience{
				Actor:     user,
				ActorId:   user.UserId,
				ActorName: user.Character.Name,
				ActeeName: messaging.NoName,
			})
		} else {
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.Say(spellSchoolCategory(spellData), fmt.Sprintf(
					`Your %s takes effect on <ansi fg="username">%s</ansi>.`,
					spellData.Name, target.Character.Name)),
				Actee:    messaging.NoLine,
				Observer: messaging.NoLine,
			}, spellAudience(user, user.Character.Name, target, target.Character.Name, room))
		}
	}
}

// spellNarratedByGoHook reports whether resolveSpell's Go hook switch owns a
// spell's narration. KEEP IT IN STEP WITH THAT SWITCH: a spell added there
// without being added here is told twice, once by applyPlayerEffect's default
// arm and once by its hook.
func spellNarratedByGoHook(spellId string) bool {
	switch spellId {
	case "fold-anchor", "fold-recall", "purge-affliction":
		return true
	}
	return false
}

// calcSpellDamage and calcMobSpellDamage have been unified into
// calcSpellDamageForCharacter() in combat_shared_helpers.go (Stage 38.1).

// consumeSpellComponent removes the first matching component item from caster's inventory.
func consumeSpellComponent(user *users.UserRecord, tag string) {
	for i, itm := range user.Character.Items {
		if itm.GetSpec().ComponentTag == tag {
			user.Character.Items = append(user.Character.Items[:i], user.Character.Items[i+1:]...)
			user.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="yellow">You consume a %s as a spell component.</ansi>`, tag))
			return
		}
	}
}

// resolveMobSpell is called when a mob's fold accumulation completes.
// resolveMobSpell is called when fold accumulation completes for a mob caster.
// It dispatches to per-target resolution based on spell type and effect.
//
// Why this is NOT merged with resolveSpell (see that function for details):
//   - HarmArea here populates both mob AND player targets; player casters only
//     hit mobs (players in the room are excluded from player-cast area spells).
//   - Mob targets include a self-cast branch (applyMobSelfEffect) for help
//     spells; player casters never self-target via this dispatcher.
//   - No onMagic script, no component consumption.
//   - Per-target helpers are entirely separate from the player equivalents.
func resolveMobSpell(mob *mobs.Mob, cs activity.CastingData, spellData *spells.SpellData, room *rooms.Room) (anyLanded bool) {
	// Go spell hooks — dispatch position-mutating / non-target spells before
	// the type-based effect routing below. Mirrors the player path in
	// resolveSpell. Stage 3.0d.
	switch cs.SpellId {
	case "fold-anchor":
		resolveFoldAnchor(actions.NewMobActorInRoom(mob, room))
		return true // uncontested utility cast: no defence to beat
	case "fold-recall":
		actor := actions.NewMobActorInRoom(mob, room)
		if !validateFoldRecall(actor) {
			// The recall could not be validated. Nothing resolved, so the
			// cast did not land.
			return false
		}
		resolveFoldRecall(actor)
		return true
	}

	// drain_area is a boss-ability effect type: it drains every living
	// player in the room and heals the caster by the aggregate lifesteal
	// (actions.ExecuteDrainArea). It bypasses the HarmArea target
	// population + per-target opposed-roll dispatch below entirely — the
	// area drain resolves its own per-player hit/miss via ExecuteSkillMove
	// inside ExecuteDrainArea, so running it through the generic
	// spellAttack-vs-defense roll here would double-roll each player.
	// Reachable ONLY at fold-cast completion (handleMobFoldCasting calls
	// resolveMobSpell here), so a spell authored with EffectType
	// "drain_area" and BaseFolds >= 2 telegraphs and is interruptible for
	// free — this function never runs until the cast finishes.
	if spellData.EffectType == "drain_area" {
		resolveMobDrainArea(mob, room, spellData)
		return true // uncontested area drain
	}

	side := spellAttackSideFor(spellData, &mob.Character, combat.SightRoom(room))
	magnitude := spellData.EffectMagnitude

	if spellData.IsHarm() && spellData.Targeting == combatvocab.TargetArea {
		cs.TargetMobInstanceIds, cs.TargetUserIds = mobAreaHarmTargets(mob, room)
	}

	for _, mobInstId := range cs.TargetMobInstanceIds {
		if mobInstId == mob.InstanceId {
			// Self-cast (HelpSingle with self target)
			applyMobSelfEffect(mob, room, spellData, magnitude)
			continue
		}
		if target := mobs.GetInstance(mobInstId); target != nil && target.Character.Health > 0 && target.Character.RoomId == room.RoomId {
			anyLanded = resolveMobSpellAgainstMob(mob, target, room, spellData, side, magnitude) || anyLanded
		}
	}
	for _, userId := range cs.TargetUserIds {
		if target := users.GetByUserId(userId); target != nil && target.Character.RoomId == room.RoomId {
			anyLanded = resolveMobSpellAgainstPlayer(mob, target, room, spellData, side, magnitude) || anyLanded
		}
	}

	return anyLanded
}

// resolveMobDrainArea is the resolution handler for a mob-cast spell whose
// EffectType is "drain_area" (the Core Guardian's "core recharge" ability
// design — see docs/superpowers/plans/completed/2026-07-06-crashsite-boss-mechanics.md
// Chunk D). It drains every living player in the room and heals the caster
// by the aggregate lifesteal via actions.ExecuteDrainArea (which mirrors the
// single-target vampire ExecuteDrain math exactly).
//
// Author's note for the spell YAML that will invoke this (Task D2): give it
// effect_type: drain_area, a type that reads as a room-wide harm ability
// (e.g. harm-area) for AI-targeting purposes even though this handler
// ignores the generic HarmArea per-target dispatch, and base_folds >= 2 so
// it telegraphs via the existing fold-cast windup and is interruptible via
// the disruptor system — this function only runs once fold accumulation
// completes (handleMobFoldCasting -> resolveMobSpell -> here), so telegraph
// and interrupt are inherited for free; no changes needed here for either.
func resolveMobDrainArea(mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData) {
	result := actions.ExecuteDrainArea(actions.NewMobActorInRoom(mob, room))

	if !result.Executed {
		sendVisualRoomText(room, messaging.CategorySpellDisruption, fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> crackles through the air, finding no one to drain.`,
			mobDisplayName(mob, room, 0), spellData.Name))
		return
	}

	// Core Charge (crash-site-boss-mechanics Chunk D, the Core Guardian's
	// drain-fed discharge gate): incremented HERE, at drain *resolution*,
	// not at cast-initiation. Build-time decision (Task D3): an interrupted
	// drain never reaches this point at all -- resolveMobDrainArea is only
	// entered once a fold-cast completes (see the doc comment above), so a
	// disruptor-cancelled drain automatically denies the charge without any
	// extra guard, satisfying spec §10.4 ("an interrupted drain ... denies
	// the charge/heal entirely").
	//
	// BehaviorState (mob.BTreeState) is a per-mob-instance store that lives
	// on the mob itself (internal/mobs/mobs.go); behaviortree.EnsureBTreeState
	// lazily initializes and returns it. It is the EXACT SAME object the
	// btree's own `increment_state`/`state_greater_than` actions/conditions
	// read and write during tree evaluation (see internal/behaviortree/
	// actions_state.go, conditions_state.go) -- there is no separate storage
	// to keep in sync. Writing it here from Go is therefore equivalent to a
	// btree `increment_state` call, just triggered from the spell-resolution
	// side (which is the only place that knows "the drain actually landed")
	// rather than from the tree (which cannot observe fold-cast completion
	// directly). This lets the Core Guardian's btree
	// (9562-the_core_guardian.yaml) gate its core-discharge purely on
	// `state_greater_than core_charge N`, with zero Go-side awareness of
	// discharge itself.
	chargeState := behaviortree.EnsureBTreeState(mob)
	chargeState.Set("core_charge", chargeState.GetInt("core_charge")+1)

	for _, pr := range result.PlayerResults {
		target := users.GetByUserId(pr.UserId)
		if target == nil {
			continue
		}
		c := newSpellEffectCtx(&mob.Character, actions.NewMobActorInRoom(mob, room),
			actions.NewUserActorInRoom(target, room), room, spellData, 0, pr.MoveResult.Defence)
		if !pr.MoveResult.Hit && pr.MoveResult.Damage == 0 {
			// Defended with zero damage (a defensive crit). This used to be a
			// silent miss; U6b Task 9 speaks the defence triad so the player
			// who fully stopped the pull learns what saved them.
			sendSpellChannelDefenceMessages(room, spellSchoolCategory(spellData), pr.MoveResult.Defence,
				spellDefenceIdentity(&mob.Character, nil, room),
				spellDefenceIdentity(target.Character, target, room),
				spellData.Name, nil, target)
			continue
		}
		if pr.MoveResult.Hit {
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.NoLine,
				Actee: messaging.Say(c.category(), fmt.Sprintf(
					`%s's <ansi fg="cyan">%s</ansi> saps your strength! (<ansi fg="damage">%s</ansi>)`,
					c.casterName(), spellData.Name,
					combat.GetDamageDescription(pr.MoveResult.Damage, target.Character.HealthMax.Value))),
				Observer: messaging.NoLine,
			}, c.audience())
			if !target.Character.IsInCombat() {
				targeting.Commit(target.Character, state.ActorRef{MobInstanceId: mob.InstanceId}, targeting.ReasonAttack)
			}
		} else {
			// Defended, but the drain still landed a partial pull. Since
			// Task 13 a defended maneuver can deal partial damage; say so
			// instead of letting the player's HP drop with no message at all.
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.NoLine,
				Actee: messaging.Say(c.category(), fmt.Sprintf(
					`%s's <ansi fg="cyan">%s</ansi> fails to take full hold of you, but still saps a little of your strength! (<ansi fg="damage">%s</ansi>)`,
					c.casterName(), spellData.Name,
					combat.GetDamageDescription(pr.MoveResult.Damage, target.Character.HealthMax.Value))),
				Observer: messaging.NoLine,
			}, c.audience())
		}
	}

	sendVisualRoomText(room, spellSchoolCategory(spellData), fmt.Sprintf(
		`%s's <ansi fg="cyan">%s</ansi> tears the life from everyone in the room!`,
		mobDisplayName(mob, room, 0), spellData.Name))
}

// applyMobSelfEffect handles self-targeted help spells (heal, minor-shield).
func applyMobSelfEffect(mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, magnitude int) {
	switch spellData.EffectType {
	case "heal":
		skillLevel := mob.Character.GetSkillLevel(skills.Spellcasting)
		regenMult := float64(magnitude)
		if regenMult < 1.0 {
			regenMult = 1.0
		}
		durationRounds := calcSpellDuration(spellData.BaseFolds, skillLevel, spellData.CasterStatValue(mob.Character.Stats)) / 2
		if durationRounds < 6 {
			durationRounds = 6
		}
		_ = mob.Character.AddConditionMagnitude(conditions.ConditionIdRegenerating, durationRounds, regenMult, "heal spell")
		sendVisualRoomText(room, messaging.CategorySpellVital, fmt.Sprintf(
			`%s channels restorative magic.`, mobDisplayName(mob, room, 0)))
	case "condition":
		for _, conditionId := range spellData.ConditionIds {
			applySpellCondition(mob, spellData, &mob.Character, conditionId)
		}
	case "shield":
		skillLevel := mob.Character.GetSkillLevel(skills.Spellcasting)
		weightedSkill := int(math.Round(float64(skillLevel) * float64(configs.GetBalanceConfig().SkillWeight)))
		shieldBonus := (spellData.CasterStatValue(mob.Character.Stats) + weightedSkill) / 3
		if shieldBonus < 1 {
			shieldBonus = 1
		}
		// Scale shield strength by spell magnitude (100 = 1.0x baseline)
		if magnitude > 0 {
			shieldBonus = int(math.Round(float64(shieldBonus) * float64(magnitude) / 100.0))
			if shieldBonus < 1 {
				shieldBonus = 1
			}
		}
		duration := calcSpellDuration(spellData.BaseFolds, skillLevel, spellData.CasterStatValue(mob.Character.Stats))
		_ = mob.Character.AddConditionMagnitude(conditions.ConditionIdMinorShield, duration, float64(shieldBonus), "spell")
		sendVisualRoomText(room, spellSchoolCategory(spellData), fmt.Sprintf(
			`A shimmering barrier forms around %s.`, mobDisplayName(mob, room, 0)))
	}
}

// landed carries the same meaning as on the player path: the contest was WON
// outright. See resolveAgainstMob.
func resolveMobSpellAgainstMob(caster *mobs.Mob, target *mobs.Mob, room *rooms.Room,
	spellData *spells.SpellData, side combat.AttackSide, magnitude int) (landed bool) {
	// Non-harm effects (a heal, or a condition buff cast on an ally mob) are
	// a cooperative cast, not an attack — the target should not roll defense
	// against a friendly effect, and a "fumble" backfire makes no sense for
	// it either. Bypass the contest/backfire gate entirely and apply
	// directly, as an uncontested attack win. (Crash-site boss-mechanics
	// Chunk B: the Repair Frame add heals Warden-Prime / the Core Guardian
	// this way.) Widened from EffectType == "heal": a mob buffing an ally
	// with a condition spell is just as cooperative and was contesting
	// before this change.
	casterActor := actions.NewMobActorInRoom(caster, room)
	targetActor := actions.NewMobActorInRoom(target, room)
	if spellData.AttackType == combatvocab.AttackNone {
		applySpellEffect(newSpellEffectCtx(&caster.Character, casterActor, targetActor, room, spellData,
			magnitude, combat.ChannelDefenceResult{DamageMultiplier: 1}))
		// Uncontested cooperative cast: no defence to beat, so it landed.
		return true
	}
	// Task 17: the sleeping-victim forced crit reaches the spell channel.
	side.ForceCrit = combat.SleepingForceCrit(&target.Character)
	out := runSpellChannelAttack(combat.SightRoom(room), spellData.Attack(), side, &caster.Character, &target.Character)
	c := newSpellEffectCtx(&caster.Character, casterActor, targetActor, room, spellData, magnitude, out)
	if out.AttackerFumble {
		applySpellBackfire(c)
		return false
	}
	interruptSpellTarget(c)
	recordSpellResolution(c, applySpellEffect(c))

	// U6b Task 10: the defending mob's crit defence counters the mob caster.
	fireSpellCounterTier(room, out, spellData.Attack(),
		&target.Character, &caster.Character, nil, nil)

	return !out.Defended
}

// resolveMobSpellAgainstPlayer runs the ONE channel contest for a mob-cast
// spell at a player and applies the effect through applySpellEffect. Crit-received toughening
// for the defender fires inside the seam's bonus tier — the U9-era direct
// block this function used to carry became a duplicate and was deleted with
// the collapse (U6b Task 4).
// landed carries the same meaning as on the player path: the contest was WON
// outright. See resolveAgainstMob.
func resolveMobSpellAgainstPlayer(caster *mobs.Mob, target *users.UserRecord, room *rooms.Room,
	spellData *spells.SpellData, side combat.AttackSide, magnitude int) (landed bool) {
	// Task 17: the sleeping-victim forced crit reaches the spell channel.
	side.ForceCrit = combat.SleepingForceCrit(target.Character)
	out := runSpellChannelAttack(combat.SightRoom(room), spellData.Attack(), side, &caster.Character, target.Character)
	c := newSpellEffectCtx(&caster.Character, actions.NewMobActorInRoom(caster, room),
		actions.NewUserActorInRoom(target, room), room, spellData, magnitude, out)
	if out.AttackerFumble {
		applySpellBackfire(c)
		return false
	}
	interruptSpellTarget(c)
	recordSpellResolution(c, applySpellEffect(c))

	// U6b Task 10: the PLAYER defender's crit defence counters the mob caster.
	fireSpellCounterTier(room, out, spellData.Attack(),
		target.Character, &caster.Character, target, nil)

	return !out.Defended
}

// applyMobOnPlayerArms is the pre-unification switch for a MOB caster and a
// PLAYER target (MP), moved out of resolveMobSpellAgainstPlayer unchanged.
// The dispatcher routes here every effect that has no unified applier yet.
func applyMobOnPlayerArms(c spellEffectCtx) int {
	caster, target, room := c.casterMob(), c.targetUser(), c.room
	spellData, out := c.spell, c.out
	critTag := ""
	if out.AttackerCrit {
		critTag = ` <ansi fg="yellow">[CRIT!]</ansi>`
	}
	switch spellData.EffectType {
	case "condition":
		// Binary status: a defended cast narrates the triad and applies nothing.
		if out.Defended {
			sendSpellChannelDefenceMessages(room, spellSchoolCategory(spellData), out,
				spellDefenceIdentity(&caster.Character, nil, room),
				spellDefenceIdentity(target.Character, target, room), spellData.Name, nil, target)
			if spellData.IsHarm() {
				if !target.Character.IsInCombat() {
					targeting.Commit(target.Character, state.ActorRef{MobInstanceId: caster.InstanceId}, targeting.ReasonAttack)
				}
			}
			break
		}
		for _, conditionId := range spellData.ConditionIds {
			applySpellCondition(target, spellData, &caster.Character, conditionId)
		}
		// Set aggro for harmful condition spells
		if spellData.IsHarm() {
			if !target.Character.IsInCombat() {
				targeting.Commit(target.Character, state.ActorRef{MobInstanceId: caster.InstanceId}, targeting.ReasonAttack)
			}
		}
		messaging.SendTrio(messaging.Trio{
			Actor: messaging.NoLine,
			Actee: messaging.Say(spellSchoolCategory(spellData), fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> takes effect on you!%s`,
				caster.Character.Name, spellData.Name, critTag)),
			Observer: messaging.Say(spellSchoolCategory(spellData), fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> affects <ansi fg="username">%s</ansi>!`,
				caster.Character.Name, spellData.Name, target.Character.Name)),
		}, spellAudience(nil, caster.Character.Name, target, target.Character.Name, room))
	default:
		if out.Defended {
			sendSpellChannelDefenceMessages(room, spellSchoolCategory(spellData), out,
				spellDefenceIdentity(&caster.Character, nil, room),
				spellDefenceIdentity(target.Character, target, room), spellData.Name, nil, target)
			break
		}
		messaging.SendTrio(messaging.Trio{
			Actor: messaging.NoLine,
			Actee: messaging.Say(spellSchoolCategory(spellData), fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> takes effect on you.`,
				caster.Character.Name, spellData.Name)),
			Observer: messaging.NoLine,
		}, spellAudience(nil, caster.Character.Name, target, target.Character.Name, room))
	}
	return 0
}

// resolveIdentify finds the named item on the caster and renders
// the identify template with descriptive item properties.
func resolveIdentify(user *users.UserRecord, itemName string, room *rooms.Room) {

	if itemName == "" {
		user.SendText(messaging.CategorySystem, "Identify what? (Usage: cast identify <item>)")
		return
	}

	// Search backpack and equipped items as a single pool
	matchItem, _, found := user.Character.FindItem(itemName)

	if !found {
		user.SendText(messaging.CategorySystem, "You can't seem to identify that.")
		return
	}

	iSpec := matchItem.GetSpec()

	type identifyDetails struct {
		Item     *items.Item
		ItemSpec *items.ItemSpec
	}

	details := identifyDetails{
		Item:     &matchItem,
		ItemSpec: &iSpec,
	}

	user.SendText(messaging.CategorySpellMental,
		fmt.Sprintf(`You concentrate on the <ansi fg="item">%s</ansi>...`,
			matchItem.DisplayName()),
	)
	sendVisualRoomText(room, messaging.CategorySpellMental,
		fmt.Sprintf(
			`<ansi fg="username">%s</ansi> concentrates on their <ansi fg="item">%s</ansi>...`,
			user.Character.Name, matchItem.DisplayName()),
		user.UserId,
	)

	identifyTxt, _ := templates.Process("descriptions/identify", details, user.UserId)
	user.SendText(messaging.CategorySpellMental, identifyTxt)
}

// charmInCombatMult is charm's attack-side penalty for reaching into a mind that
// is already fighting.
//
// Restored 2026-08-24. Spec 4.1's mechanics table lists both multipliers as
// UNCHANGED across the U10c rewrite; slice B deleted them along with
// resolveCharmSpell and put nothing in their place. Nothing else covered for it:
// combat.SituationalAttackMult returns a flat 1.0 for every channel except melee
// and ranged, and the defy defence carries no combat term -- so charm quietly got
// easier mid-fight while charm.yaml, charm.template and a gameplay tip all went on
// telling players it had got harder.
//
// Literals rather than balance knobs by owner ruling 2026-08-24.
func charmInCombatMult(target *characters.Character, casterUserId int) float64 {
	if target == nil || !target.IsInCombat() {
		return 1.0
	}
	if target.CurrentCombatTarget().UserId == casterUserId {
		return 0.75 // fighting the caster -- steepest
	}
	return 0.85 // fighting someone else -- moderate
}
