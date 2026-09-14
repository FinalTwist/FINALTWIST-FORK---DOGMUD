package conditions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ids clear of the other fixtures in this package.
const (
	immunityTestStoneStomachId = 9401
	immunityTestVenomId        = 9402
	immunityTestHarmlessId     = 9403
	immunityTestRealVenomId    = 9404
)

// Stone Stomach's poison-immunity flag was read by nothing: the potion
// promised immunity and delivered a dexterity penalty. Both primitives now
// refuse a poison-flagged spec while it is held.
func TestPoisonImmunityRefusesPoisonConditions(t *testing.T) {
	restore := SeedConditionsForTest(map[int]*ConditionSpec{
		immunityTestStoneStomachId: {ConditionId: immunityTestStoneStomachId, Name: "Test Stone Stomach", TriggerCount: 5, RoundInterval: 1, Flags: []Flag{PoisonImmunity}},
		immunityTestVenomId:        {ConditionId: immunityTestVenomId, Name: "Test Venom", TriggerCount: 5, RoundInterval: 1, Flags: []Flag{Poison}},
		immunityTestHarmlessId:     {ConditionId: immunityTestHarmlessId, Name: "Test Harmless", TriggerCount: 5, RoundInterval: 1},
	})
	defer restore()

	bs := New()
	require.True(t, bs.AddCondition(immunityTestStoneStomachId, false))
	assert.False(t, bs.AddCondition(immunityTestVenomId, false), "a poison buff is refused while immune")
	assert.False(t, bs.HasCondition(immunityTestVenomId))
	assert.False(t, bs.AddConditionScaled(immunityTestVenomId, 0.5), "the scaled primitive refuses too")
	assert.False(t, bs.HasCondition(immunityTestVenomId))
	assert.True(t, bs.AddCondition(immunityTestHarmlessId, false), "a non-poison buff still lands")

	unprotected := New()
	assert.True(t, unprotected.AddCondition(immunityTestVenomId, false), "without immunity poison lands")
}

// The real Venom (buff 39) is a negative health tick, and until this slice it
// carried no flags at all, so CancelBuffsWithFlag(Poison) in Purge Affliction
// and Cleansing Wave matched nothing and the immunity would have refused
// nothing real. A spec shaped like the shipped file must be refused.
func TestPoisonImmunityRefusesAHealthTickVenom(t *testing.T) {
	restore := SeedConditionsForTest(map[int]*ConditionSpec{
		immunityTestStoneStomachId: {ConditionId: immunityTestStoneStomachId, Name: "Test Stone Stomach", TriggerCount: 5, RoundInterval: 1, Flags: []Flag{PoisonImmunity}},
		immunityTestRealVenomId: {ConditionId: immunityTestRealVenomId, Name: "Test Real Venom", TriggerCount: 6, RoundInterval: 1,
			TickPool: "health", TickPercent: -2, TickMin: 1, Flags: []Flag{Poison}},
	})
	defer restore()

	immune := New()
	require.True(t, immune.AddCondition(immunityTestStoneStomachId, false))
	assert.False(t, immune.AddCondition(immunityTestRealVenomId, false), "the venom a mob actually applies is refused")
	assert.False(t, immune.HasCondition(immunityTestRealVenomId))

	unprotected := New()
	assert.True(t, unprotected.AddCondition(immunityTestRealVenomId, false), "without immunity the venom lands")
}

const immunityTestDeadConditionId = 9405 // held (indexed by Validate) but no live spec

// A save can carry a buff id whose spec is gone, and Validate indexes it
// anyway. HasFlag dereferenced GetBuffSpec without a nil check, so once AddBuff
// started asking HasFlag(PoisonImmunity) on every add, a character holding a
// dead id ahead of the immunity buff would crash on the next venom crit.
func TestHasFlagSurvivesAHeldConditionWithNoSpec(t *testing.T) {
	restore := SeedConditionsForTest(map[int]*ConditionSpec{
		immunityTestStoneStomachId: {ConditionId: immunityTestStoneStomachId, Name: "Test Stone Stomach", TriggerCount: 5, RoundInterval: 1, Flags: []Flag{PoisonImmunity}},
		immunityTestVenomId:        {ConditionId: immunityTestVenomId, Name: "Test Venom", TriggerCount: 5, RoundInterval: 1, Flags: []Flag{Poison}},
	})
	defer restore()

	// The dead id is first in the list, so the flag scan reaches it before the
	// immunity buff it is looking for.
	bs := Conditions{List: []*Condition{{ConditionId: immunityTestDeadConditionId, TriggersLeft: 3}}}
	bs.Validate()
	if _, ok := bs.conditionIds[immunityTestDeadConditionId]; !ok {
		t.Fatal("precondition: Validate should have indexed the dead id anyway")
	}
	require.True(t, bs.AddCondition(immunityTestStoneStomachId, false))

	assert.True(t, bs.HasFlag(PoisonImmunity, false), "the flag is found past the dead id")
	assert.False(t, bs.AddCondition(immunityTestVenomId, false), "and the immunity still refuses poison")
}
