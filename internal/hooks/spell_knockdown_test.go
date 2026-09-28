package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A player's knockdown on a player fell through to the default arm and
// applied nothing.
func TestSpellKnockdown_PlayerOnPlayerKnocksDown(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := knockdownSpellForParityTest()

	resolveAgainstPlayer(f.casterUser, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), spell.EffectMagnitude)

	assert.True(t, f.targetUser.Character.IsSupine() || f.targetUser.Character.IsProne(),
		"the target must be knocked down")
	assert.Less(t, f.targetUser.Character.Health, 1000, "the knockdown also deals damage")
	assert.Equal(t, 1, countContaining(drainPlain(2), "Aliceia's Shove slams you to the ground!"))
}

const knockdownTranceConditionId = 7101 // clear of the fixture's 100/101 and narration's 7001-7009

// The mob-on-player knockdown dealt damage but skipped cancelDamageConditions,
// so a condition that ends on damage survived a hit the player-cast
// knockdown would have broken.
func TestSpellKnockdown_MobOnPlayerBreaksDamageFragileConditions(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		knockdownTranceConditionId: {ConditionId: knockdownTranceConditionId, Name: "Test Trance",
			RoundInterval: 5, TriggerCount: 3, Flags: []conditions.Flag{conditions.CancelOnDamage}},
	}))
	require.True(t, f.targetUser.Character.Conditions.AddCondition(knockdownTranceConditionId, false))
	require.True(t, f.targetUser.Character.HasConditionFlag(conditions.CancelOnDamage))
	spell := knockdownSpellForParityTest()

	resolveMobSpellAgainstPlayer(f.casterMob, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), spell.EffectMagnitude)

	assert.Less(t, f.targetUser.Character.Health, 1000, "the knockdown must deal damage")
	assert.False(t, f.targetUser.Character.HasConditionFlag(conditions.CancelOnDamage),
		"damage must break a condition that ends on damage")
}
