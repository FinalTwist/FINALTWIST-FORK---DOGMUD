package conditions

import (
	"reflect"
	"testing"
)

// Ids clear of seedRegistry's 100/101/106 and of other packages' test fixtures.
const (
	refreshTestCadenceConditionId   = 9301 // RoundInterval 2, TriggerCount 5
	refreshTestPermanentConditionId = 9302 // RoundInterval 1, TriggerCount 3, granted permanent
	refreshTestUnheldConditionId    = 9303 // never added
	refreshTestDeadConditionId      = 9304 // held (indexed by Validate) but no live spec
)

// RefreshCondition must top TriggersLeft back up without resetting RoundCounter.
// AddCondition resets both, which would starve any condition whose RoundInterval is
// above one: refreshed every round, RoundCounter would be zeroed every round
// and Trigger's `RoundCounter % RoundInterval == 0` would never see anything
// but 0 % N == 0 on round 1, then reset again before round 2 ever accumulates.
func TestRefreshCondition_KeepsCadenceAcrossRepeatedRefreshes(t *testing.T) {
	restore := SeedConditionsForTest(map[int]*ConditionSpec{
		refreshTestCadenceConditionId: {ConditionId: refreshTestCadenceConditionId, Name: "Test Cadence", TriggerCount: 5, RoundInterval: 2},
	})
	defer restore()

	bs := New()
	if !bs.AddCondition(refreshTestCadenceConditionId, false) {
		t.Fatal("precondition: could not grant the cadence condition")
	}

	triggerTotal := 0
	// Trigger then refresh here; production refreshes first (Room.RoundTick)
	// and triggers after. The count is the same either way only because
	// RefreshCondition leaves RoundCounter alone, which is the point under test.
	for round := 1; round <= 6; round++ {
		triggered := bs.Trigger()
		triggerTotal += len(triggered)

		if !bs.RefreshCondition(refreshTestCadenceConditionId) {
			t.Fatalf("round %d: RefreshCondition returned false for a held condition", round)
		}

		idx := bs.conditionIds[refreshTestCadenceConditionId]
		if got := bs.List[idx].TriggersLeft; got != 5 {
			t.Fatalf("round %d: TriggersLeft = %d, want 5 (topped back up)", round, got)
		}
	}

	if triggerTotal != 3 {
		t.Errorf("triggered %d times across 6 rounds at RoundInterval 2, want 3 (rounds 2, 4, 6)", triggerTotal)
	}
}

// A permanent held condition (isPermanent true on AddCondition) must stay permanent
// through a refresh: RefreshCondition must not read the spec's finite TriggerCount
// over top of TriggersLeftUnlimited, and must not touch Permanent.
func TestRefreshCondition_LeavesAPermanentConditionPermanent(t *testing.T) {
	restore := SeedConditionsForTest(map[int]*ConditionSpec{
		refreshTestPermanentConditionId: {ConditionId: refreshTestPermanentConditionId, Name: "Test Perma", TriggerCount: 3, RoundInterval: 1},
	})
	defer restore()

	bs := New()
	if !bs.AddCondition(refreshTestPermanentConditionId, true) {
		t.Fatal("precondition: could not grant the permanent condition")
	}

	if !bs.RefreshCondition(refreshTestPermanentConditionId) {
		t.Fatal("RefreshCondition returned false for a held permanent condition")
	}

	idx := bs.conditionIds[refreshTestPermanentConditionId]
	if !bs.List[idx].Permanent {
		t.Error("RefreshCondition cleared Permanent on a permanent condition")
	}
	if got := bs.List[idx].TriggersLeft; got != TriggersLeftUnlimited {
		t.Errorf("TriggersLeft = %d, want TriggersLeftUnlimited", got)
	}
}

// An id never granted is not held, so nothing to refresh.
func TestRefreshCondition_UnheldIdReturnsFalse(t *testing.T) {
	restore := SeedConditionsForTest(map[int]*ConditionSpec{
		refreshTestUnheldConditionId: {ConditionId: refreshTestUnheldConditionId, Name: "Test Unheld", TriggerCount: 3, RoundInterval: 1},
	})
	defer restore()

	bs := New()
	if bs.RefreshCondition(refreshTestUnheldConditionId) {
		t.Error("RefreshCondition returned true for an id never added")
	}
}

// A save can carry a condition id whose content was later removed. Conditions.Validate
// indexes it into conditionIds before checking whether GetConditionSpec finds anything,
// so the id is "held" by the conditionIds map despite having no live spec. Refresh
// must decline rather than guess a TriggerCount.
func TestRefreshCondition_HeldDeadIdReturnsFalse(t *testing.T) {
	restore := SeedConditionsForTest(map[int]*ConditionSpec{})
	defer restore()

	bs := Conditions{List: []*Condition{{ConditionId: refreshTestDeadConditionId, TriggersLeft: 3}}}
	bs.Validate()

	if _, ok := bs.conditionIds[refreshTestDeadConditionId]; !ok {
		t.Fatal("precondition: Validate should have indexed the dead id anyway")
	}
	if bs.RefreshCondition(refreshTestDeadConditionId) {
		t.Error("RefreshCondition returned true for a held condition with no live spec")
	}
}

// A stacking record can only be added through AddConditionMagnitude (see
// TestAddConditionRefusesAStackingSpec / TestAddConditionScaledRefusesAStackingSpec),
// so RefreshCondition must decline it too: topping TriggersLeft back up to the
// spec's TriggerCount would either revive an expired-but-unpruned record
// with no stacks (phantom end line on its next tick) or, on a live one,
// misreport its duration as the spec default rather than its longest stack.
// Room condition paths call this (rooms.go), so a stacking bleed authored into a
// room's buffids would otherwise get exactly this treatment on every visit.
func TestRefreshCondition_RefusesAStackingSpec(t *testing.T) {
	spec := stackingSpec()
	restore := SeedConditionsForTest(map[int]*ConditionSpec{spec.ConditionId: spec})
	defer restore()

	bs := New()
	bs.AddConditionMagnitude(spec.ConditionId, 3, -2)
	bs.AddConditionMagnitude(spec.ConditionId, 5, -3)
	wantStacks := append([]Stack{}, bs.List[0].Stacks...)
	wantTriggersLeft := bs.List[0].TriggersLeft

	if bs.RefreshCondition(spec.ConditionId) {
		t.Fatal("RefreshCondition must refuse a stacking spec")
	}
	if got := bs.List[0].Stacks; !reflect.DeepEqual(got, wantStacks) {
		t.Fatalf("Stacks = %+v, want unchanged %+v", got, wantStacks)
	}
	if got := bs.List[0].TriggersLeft; got != wantTriggersLeft {
		t.Fatalf("TriggersLeft = %d, want unchanged %d", got, wantTriggersLeft)
	}
}
