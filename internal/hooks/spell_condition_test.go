package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hexSpellForConditionTest is a harmful condition spell, mind-fog's shape.
func hexSpellForConditionTest() *spells.SpellData {
	return &spells.SpellData{
		SpellId: "test-hex", Name: "Hex", AttackType: combatvocab.AttackSpell,
		DamageType: combatvocab.DamageMental, Targeting: combatvocab.TargetSingle,
		EffectType: "condition", ConditionIds: []int{100}, PrimaryStat: "willpower",
		Schools: []string{spells.SchoolMental},
	}
}

// Slice 3b: one condition applier narrates every pairing to the room with
// the same line. Mob on mob was silent; player on mob and mob on player
// said "affects".
func TestSpellCondition_EveryPairingSettlesInViewOfTheRoom(t *testing.T) {
	for _, p := range spellParityPairings() {
		t.Run(p.name, func(t *testing.T) {
			f := newSpellParityFixture(t, spellContestAttackWin())
			target := "Ghoul"
			if p.tgt == combat.User {
				target = "Bobrick"
			}

			p.cast(f, conditionSpellForParityTest())

			assert.Equal(t, 1, countContaining(drainPlain(3), "Bless settles over "+target+"."),
				"a watcher sees the condition take hold")
		})
	}
}

// Owner ruling 2: a player's harmful spell on a mob is an assault. The old
// player-on-mob condition arm committed aggro but never called
// SeedAggression.
func TestSpellCondition_HarmfulOnAMobIsAnAssault(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := hexSpellForConditionTest()

	resolveAgainstMob(f.casterUser, f.targetMob, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), 0)

	attacked := events.DrainQueuedPlayerAttackedMobsForTest(0)
	require.Len(t, attacked, 1, "a player's harmful condition on a mob is aggression")
	assert.Equal(t, 101, attacked[0].MobInstanceId)
	assert.Equal(t, state.ActorRef{UserId: 1}, f.targetMob.Character.CurrentCombatTarget())
}

// A mob's harmful condition on another mob starts the fight. Every aggro
// commit in the old arm needed a player caster.
func TestSpellCondition_MobOnMobHarmStartsAFight(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := hexSpellForConditionTest()

	resolveMobSpellAgainstMob(f.casterMob, f.targetMob, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), 0)

	assert.Equal(t, state.ActorRef{MobInstanceId: 100}, f.targetMob.Character.CurrentCombatTarget())
}
