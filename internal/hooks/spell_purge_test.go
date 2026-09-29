package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func purgeSpellForParityTest() *spells.SpellData {
	return &spells.SpellData{
		SpellId: "test-cleanse", Name: "Cleanse", AttackType: combatvocab.AttackNone,
		DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle,
		EffectType: "purge", BaseFolds: 4, PrimaryStat: "willpower",
		Schools: []string{spells.SchoolVital},
	}
}

// poisonForPurgeTest gives c one spell-poison record.
func poisonForPurgeTest(t *testing.T, c *characters.Character) {
	t.Helper()
	require.NoError(t, c.AddConditionMagnitude(conditions.ConditionIdPoisoned, 10, -5, "test"))
}

func poisoned(c *characters.Character) bool {
	return len(c.GetConditions(conditions.ConditionIdPoisoned)) > 0
}

// Slice 3b: Cleansing Wave over a companion said it took effect and cleansed
// nothing, and so did a creature's cleanse on a player.
func TestSpellPurge_CompanionsAndPlayersAreCleansed(t *testing.T) {
	t.Run("PM", func(t *testing.T) {
		f := newSpellParityFixture(t, spellContestAttackWin())
		spell := purgeSpellForParityTest()
		poisonForPurgeTest(t, &f.targetMob.Character)

		resolveAgainstMob(f.casterUser, f.targetMob, f.room, spell,
			spellAttackSideFor(spell, f.casterUser.Character, nil), 0)

		assert.False(t, poisoned(&f.targetMob.Character), "the companion is cleansed")
		assert.Equal(t, 1, countContaining(drainPlain(1), "Your Cleanse cleanses Ghoul"))
	})
	t.Run("MP", func(t *testing.T) {
		f := newSpellParityFixture(t, spellContestAttackWin())
		spell := purgeSpellForParityTest()
		poisonForPurgeTest(t, f.targetUser.Character)

		resolveMobSpellAgainstPlayer(f.casterMob, f.targetUser, f.room, spell,
			spellAttackSideFor(spell, &f.casterMob.Character, nil), 0)

		assert.False(t, poisoned(f.targetUser.Character), "the player is cleansed")
		assert.Equal(t, 1, countContaining(drainPlain(2), "Cleanse purges the toxins from your body."))
	})
}

// Slice 3b: a spell with no effect of its own tells both sides. The
// player-to-player default arm told the target nothing.
func TestSpellDefault_TheTargetIsTold(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := &spells.SpellData{SpellId: "test-curiosity", Name: "Curiosity", AttackType: combatvocab.AttackNone,
		DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle, PrimaryStat: "willpower"}

	resolveAgainstPlayer(f.casterUser, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), 0)

	assert.Equal(t, 1, countContaining(drainPlain(1), "Your Curiosity takes effect on Bobrick."))
	assert.Equal(t, 1, countContaining(drainPlain(2), "Curiosity takes effect on you."))
}
