package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countSpellContests wraps the fixture's pinned contest with a counter. The
// restore is a t.Cleanup registered after the fixture's own, so it runs
// first and hands the seam back to the fixture's pin.
func countSpellContests(t *testing.T) *int {
	t.Helper()
	n := 0
	pinned := runSpellChannelAttack
	runSpellChannelAttack = func(v messaging.RoomVisibility, a combatvocab.Attack, s combat.AttackSide,
		atk, def *characters.Character) combat.ChannelDefenceResult {
		n++
		return pinned(v, a, s, atk, def)
	}
	t.Cleanup(func() { runSpellChannelAttack = pinned })
	return &n
}

// conditionSpellForParityTest is a help spell that queues condition 100,
// which seedAllRegistries defines with no tick pool and no scaled kind, so
// applySpellCondition takes the plain AddCondition door.
func conditionSpellForParityTest() *spells.SpellData {
	return &spells.SpellData{
		SpellId: "test-bless", Name: "Bless", AttackType: combatvocab.AttackNone,
		DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle,
		EffectType: "condition", ConditionIds: []int{100}, BaseFolds: 4, PrimaryStat: "willpower",
		Schools: []string{spells.SchoolEnhancement},
	}
}

// Parity slice 3b: a help spell is uncontested and recorded once as a landed
// cast on every pairing. MP was contested (audit row 3); PP skipped the
// contest only inside resolveSpell; MM recorded nothing.
func TestSpellHelp_EveryPairingIsUncontestedAndRecordedOnce(t *testing.T) {
	for _, p := range spellParityPairings() {
		t.Run(p.name, func(t *testing.T) {
			f := newSpellParityFixture(t, spellContestAttackWin())
			contests := countSpellContests(t)

			p.cast(f, conditionSpellForParityTest())

			assert.Zero(t, *contests, "a help spell runs no contest")
			require.Len(t, f.records, 1, "one record per resolved cast")
			assert.Equal(t, p.src, f.records[0].src)
			assert.Equal(t, p.tgt, f.records[0].tgt)
			assert.True(t, f.records[0].hit, "an uncontested cast landed")
		})
	}
}
