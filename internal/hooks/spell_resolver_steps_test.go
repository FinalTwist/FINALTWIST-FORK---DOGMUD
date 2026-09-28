package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spellParityPairing drives one caster-target pairing through its real
// resolver. PM player to mob, PP player to player (room 1 is Pvp), MM mob to
// mob, MP mob to player.
type spellParityPairing struct {
	name      string
	src, tgt  combat.SourceTarget
	caster    func(*spellParityFixture) *characters.Character
	target    func(*spellParityFixture) *characters.Character
	casterRef func(*spellParityFixture) state.ActorRef
	cast      func(*spellParityFixture, *spells.SpellData)
}

func spellParityPairings() []spellParityPairing {
	userCaster := func(f *spellParityFixture) *characters.Character { return f.casterUser.Character }
	mobCaster := func(f *spellParityFixture) *characters.Character { return &f.casterMob.Character }
	userRef := func(f *spellParityFixture) state.ActorRef { return state.ActorRef{UserId: f.casterUser.UserId} }
	mobRef := func(f *spellParityFixture) state.ActorRef {
		return state.ActorRef{MobInstanceId: f.casterMob.InstanceId}
	}
	return []spellParityPairing{
		{name: "PM", src: combat.User, tgt: combat.Mob, caster: userCaster, casterRef: userRef,
			target: func(f *spellParityFixture) *characters.Character { return &f.targetMob.Character },
			cast: func(f *spellParityFixture, s *spells.SpellData) {
				resolveAgainstMob(f.casterUser, f.targetMob, f.room, s,
					spellAttackSideFor(s, f.casterUser.Character, nil), s.EffectMagnitude)
			}},
		{name: "PP", src: combat.User, tgt: combat.User, caster: userCaster, casterRef: userRef,
			target: func(f *spellParityFixture) *characters.Character { return f.targetUser.Character },
			cast: func(f *spellParityFixture, s *spells.SpellData) {
				resolveAgainstPlayer(f.casterUser, f.targetUser, f.room, s,
					spellAttackSideFor(s, f.casterUser.Character, nil), s.EffectMagnitude)
			}},
		{name: "MM", src: combat.Mob, tgt: combat.Mob, caster: mobCaster, casterRef: mobRef,
			target: func(f *spellParityFixture) *characters.Character { return &f.targetMob.Character },
			cast: func(f *spellParityFixture, s *spells.SpellData) {
				resolveMobSpellAgainstMob(f.casterMob, f.targetMob, f.room, s,
					spellAttackSideFor(s, &f.casterMob.Character, nil), s.EffectMagnitude)
			}},
		{name: "MP", src: combat.Mob, tgt: combat.User, caster: mobCaster, casterRef: mobRef,
			target: func(f *spellParityFixture) *characters.Character { return f.targetUser.Character },
			cast: func(f *spellParityFixture, s *spells.SpellData) {
				resolveMobSpellAgainstPlayer(f.casterMob, f.targetUser, f.room, s,
					spellAttackSideFor(s, &f.casterMob.Character, nil), s.EffectMagnitude)
			}},
	}
}

// Every caster kind is hurt by its own backfire, the room sees it, and it is
// recorded. PP and MM used to record nothing.
func TestSpellBackfire_EveryCasterIsHurtToldAndRecorded(t *testing.T) {
	for _, p := range spellParityPairings() {
		t.Run(p.name, func(t *testing.T) {
			f := newSpellParityFixture(t, combat.ChannelDefenceResult{AttackerFumble: true})
			spell := physicalHarmSpellForCollapseTest()
			caster := p.caster(f)

			p.cast(f, spell)

			assert.Equal(t, 1000-spell.EffectMagnitude/4, caster.Health, "a backfire wounds its caster")
			assert.Equal(t, 1, countContaining(drainPlain(3), "spell backfires!"), "the room sees every backfire")
			require.Len(t, f.records, 1, "every backfire is recorded")
			assert.True(t, f.records[0].backfire)
			assert.Equal(t, p.src, f.records[0].src)
			assert.Equal(t, p.tgt, f.records[0].tgt)
		})
	}
}

// Every landed cast is recorded once, with the damage it dealt. PP and MM
// used to record nothing.
func TestSpellRecord_EveryPairingRecordsALandedCast(t *testing.T) {
	for _, p := range spellParityPairings() {
		t.Run(p.name, func(t *testing.T) {
			f := newSpellParityFixture(t, spellContestAttackWin())
			spell := physicalHarmSpellForCollapseTest()
			target := p.target(f)

			p.cast(f, spell)

			require.Len(t, f.records, 1)
			assert.True(t, f.records[0].hit)
			assert.False(t, f.records[0].backfire)
			assert.Equal(t, p.src, f.records[0].src)
			assert.Equal(t, p.tgt, f.records[0].tgt)
			assert.Equal(t, 1000-target.Health, f.records[0].dmg)
		})
	}
}

// A configured boss-interrupt spell cancels any casting target's cast. Only
// a player's spell on a mob used to.
func TestSpellInterrupt_ACastingPlayerIsInterrupted(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	f.targetUser.Character.Activity = activity.NewMachine()
	require.NoError(t, f.targetUser.Character.Activity.TransitionToCasting(
		activity.CastingData{SpellId: "fold-anchor"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin}))
	require.True(t, f.targetUser.Character.IsCasting())
	spell := physicalHarmSpellForCollapseTest()
	spell.SpellId = "neural-stun" // a default boss-interrupt id, config.balance.misc.go:334

	resolveAgainstPlayer(f.casterUser, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), spell.EffectMagnitude)

	assert.False(t, f.targetUser.Character.IsCasting(), "the target's cast must be cancelled")
	assert.Equal(t, 1, countContaining(drainPlain(2), "your spell collapses!"))
}

// The drain's personal lines were raw SendText with the mob's name baked in,
// so a drained player in the dark read the name they could not see.
func TestMobDrainArea_DrainedPlayerInTheDarkReadsSomething(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	t.Cleanup(combat.SetChannelAttackContestRunnerForTest(attackWinContest(t)))
	f.room.Lamp = nil
	darken(t, 1)
	spell := &spells.SpellData{SpellId: "test-core-recharge", Name: "Core Recharge", EffectType: "drain_area",
		AttackType: combatvocab.AttackSpell, DamageType: combatvocab.DamagePhysical, Targeting: combatvocab.TargetArea}

	resolveMobDrainArea(f.casterMob, f.room, spell)

	target := drainPlain(2)
	assert.Equal(t, 1, countContaining(target, "Something's Core Recharge saps your strength!"))
	assert.Equal(t, 0, countContaining(target, "Skeleton"))
}
