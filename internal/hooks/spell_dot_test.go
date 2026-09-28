package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Audit row 9: the mob-on-mob dot read its duration from a nil caster (skill
// 0, stat 100). The fixture caster has spellcasting 3 and willpower 100, so
// the old arm gives 30 triggers here and the caster's own numbers give 33.
func TestSpellDot_MobOnMobDurationReadsTheCaster(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := dotSpellForParityTest()

	resolveMobSpellAgainstMob(f.casterMob, f.targetMob, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), spell.EffectMagnitude)

	recs := f.targetMob.Character.GetConditions(conditions.ConditionIdPoisoned)
	require.Len(t, recs, 1)
	assert.Equal(t, calcSpellDuration(spell.BaseFolds, 3, 100)/3, recs[0].TriggersLeft)
	assert.Equal(t, 1, countContaining(drainPlain(3), "Blight afflicts Ghoul!"))
}

// A player's dot on a player fell through to applyPlayerEffect's default arm,
// which said "takes effect" and applied nothing.
func TestSpellDot_PlayerOnPlayerAfflicts(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := dotSpellForParityTest()

	resolveAgainstPlayer(f.casterUser, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), spell.EffectMagnitude)

	recs := f.targetUser.Character.GetConditions(conditions.ConditionIdPoisoned)
	require.Len(t, recs, 1)
	assert.Equal(t, calcSpellDuration(spell.BaseFolds, 3, 100)/3, recs[0].TriggersLeft)
	assert.Equal(t, 1, countContaining(drainPlain(2), "Aliceia's Blight afflicts you!"))
}
