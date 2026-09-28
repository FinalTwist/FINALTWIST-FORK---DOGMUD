package hooks

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shieldSpellForParityTest() *spells.SpellData {
	return &spells.SpellData{
		SpellId: "test-ward", Name: "Ward", AttackType: combatvocab.AttackNone,
		DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle,
		EffectType: "shield", EffectMagnitude: 75, BaseFolds: 4, PrimaryStat: "willpower",
		Schools: []string{spells.SchoolEnhancement},
	}
}

// parityShieldBonus is the shield's strength on the fixture's equalised
// caster, from the fixture's numbers: willpower 100 plus spellcasting 3
// times SkillWeight, a third of that, scaled by the spell's magnitude 75.
// Test binaries read the Go default SkillWeight, not the shipped one.
func parityShieldBonus() float64 {
	weighted := int(math.Round(3 * float64(configs.GetBalanceConfig().SkillWeight)))
	return float64(int(math.Round(float64((100+weighted)/3) * 75 / 100.0)))
}

// shieldRecord returns c's one Minor Shield record, or fails the test.
func shieldRecord(t *testing.T, c *characters.Character) *conditions.Condition {
	t.Helper()
	recs := c.GetConditions(conditions.ConditionIdMinorShield)
	require.Len(t, recs, 1, "the shield must leave one Minor Shield record")
	return recs[0]
}

// Slice 3b (audit row 15): a shield on a charmed pet applied nothing, since
// a mob target had no shield arm.
func TestSpellShield_APetCanBeShielded(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	f.targetMob.Character.Charm(1, -1, "")
	spell := shieldSpellForParityTest()

	resolveAgainstMob(f.casterUser, f.targetMob, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), spell.EffectMagnitude)

	rec := shieldRecord(t, &f.targetMob.Character)
	assert.Equal(t, parityShieldBonus(), rec.Magnitude)
	assert.Equal(t, calcSpellDuration(4, 3, 100), rec.TriggersLeft)
	assert.Equal(t, 1, countContaining(drainPlain(3), "A shimmering barrier surrounds Ghoul"))
}

// Slice 3b (audit row 3): a creature's shield on a player applied nothing.
func TestSpellShield_ACreatureShieldsAPlayer(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := shieldSpellForParityTest()

	resolveMobSpellAgainstPlayer(f.casterMob, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), spell.EffectMagnitude)

	assert.Equal(t, parityShieldBonus(), shieldRecord(t, f.targetUser.Character).Magnitude)
	assert.Equal(t, 1, countContaining(drainPlain(2), "A shimmering magical barrier forms around you"))
}

// Owner ruling 3: a shield does not crit. The player-to-player arm multiplied
// it by 1.5 on a crit no cast could reach.
func TestSpellShield_ACritChangesNothing(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	caster := actions.NewUserActorInRoom(f.casterUser, f.room)
	target := actions.NewUserActorInRoom(f.targetUser, f.room)

	applySpellEffect(newSpellEffectCtx(f.casterUser.Character, caster, target, f.room,
		shieldSpellForParityTest(), 75, spellContestAttackCrit()))

	assert.Equal(t, parityShieldBonus(), shieldRecord(t, f.targetUser.Character).Magnitude,
		"a crit must not strengthen a shield")
}
