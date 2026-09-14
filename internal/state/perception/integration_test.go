package perception_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// seedBlindConditions registers minimal BuffSpec entries for the two blind-source
// buff IDs (3 = Blinded, 77 = Flashbang Blindness) into the global buff
// registry and returns a cleanup function that restores the original state.
//
// Without seeded specs, buffs.AddBuff returns false (spec not found) and
// characters.AddBuff returns an error, causing every PE-INT test to fail
// at setup rather than at the assertion under test.
//
// TriggerCount=5, RoundInterval=1 gives a non-zero lifespan so the buff
// is not immediately expired on add.
func seedBlindConditions(t *testing.T) func() {
	t.Helper()
	cleanup := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		perception.ConditionIdBlinded: {
			ConditionId:   perception.ConditionIdBlinded,
			Name:          "Blinded",
			Description:   "Your eyes are blinded.",
			TriggerCount:  5,
			RoundInterval: 1,
		},
		perception.ConditionIdFlashbangBlindness: {
			ConditionId:   perception.ConditionIdFlashbangBlindness,
			Name:          "Flashbang Blindness",
			Description:   "A flash of light sears your vision.",
			TriggerCount:  3,
			RoundInterval: 1,
		},
	})
	return cleanup
}

// TestMain initialises the mudlog logger once for all integration tests in
// this file. buffs.Validate() calls mudlog.Warn when it encounters an unknown
// buffId; without initialisation the logger panics on a nil receiver.
func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	m.Run()
}

// PE-INT-001: AddBuff(3) → Blinded.
func TestIntegration_ConditionBlindedAppliesBlinded(t *testing.T) {
	defer seedBlindConditions(t)()

	c := characters.New()
	if c.Perception.State() != perception.Sighted {
		t.Fatalf("initial state = %v, want Sighted", c.Perception.State())
	}
	if err := c.AddCondition(perception.ConditionIdBlinded, false); err != nil {
		t.Fatalf("AddBuff(3): %v", err)
	}
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after AddBuff(3), state = %v, want Blinded", c.Perception.State())
	}
}

// PE-INT-002: AddBuff(3) + AddBuff(77) + RemoveBuff(3) → still Blinded.
//
// The second source used to be the ConditionBlinded enum entry. The enum is
// gone and buffs 3 and 77 are the two blind sources that remain, so the
// overlap this test exists to pin is now buff-on-buff.
func TestIntegration_OverlapKeepsBlinded(t *testing.T) {
	defer seedBlindConditions(t)()

	c := characters.New()
	if err := c.AddCondition(perception.ConditionIdBlinded, false); err != nil {
		t.Fatalf("AddBuff(3): %v", err)
	}
	if err := c.AddCondition(perception.ConditionIdFlashbangBlindness, false); err != nil {
		t.Fatalf("AddBuff(77): %v", err)
	}
	c.RemoveCondition(perception.ConditionIdBlinded)
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after removing buff 3 but buff 77 still active, state = %v, want Blinded", c.Perception.State())
	}
}

// PE-INT-003: Add both, remove both → Sighted.
func TestIntegration_AllSourcesClearedReturnsSighted(t *testing.T) {
	defer seedBlindConditions(t)()

	c := characters.New()
	_ = c.AddCondition(perception.ConditionIdBlinded, false)
	_ = c.AddCondition(perception.ConditionIdFlashbangBlindness, false)
	c.RemoveCondition(perception.ConditionIdBlinded)
	c.RemoveCondition(perception.ConditionIdFlashbangBlindness)
	if c.Perception.State() != perception.Sighted {
		t.Errorf("after clearing all sources, state = %v, want Sighted", c.Perception.State())
	}
}

// PE-INT-004 (PE-010 from spec matrix): re-applying buff while already
// Blinded is a no-op (no ErrInvalidTransition propagated, no log spam).
func TestIntegration_ReapplyConditionNoOp(t *testing.T) {
	defer seedBlindConditions(t)()

	c := characters.New()
	if err := c.AddCondition(perception.ConditionIdBlinded, false); err != nil {
		t.Fatalf("first AddBuff: %v", err)
	}
	// Re-add the same buff (the buff system stacks duration, but the
	// blind-source state is the same). The current-state guard in
	// AddBuff prevents the transition from firing twice.
	if err := c.AddCondition(perception.ConditionIdBlinded, false); err != nil {
		t.Fatalf("second AddBuff: %v", err)
	}
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after duplicate AddBuff, state = %v, want Blinded", c.Perception.State())
	}
}

// PE-INT-005: Flashbang (buff 77) drives the same Perception transitions.
func TestIntegration_FlashbangBlindness(t *testing.T) {
	defer seedBlindConditions(t)()

	c := characters.New()
	if err := c.AddCondition(perception.ConditionIdFlashbangBlindness, false); err != nil {
		t.Fatalf("AddBuff(77): %v", err)
	}
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after AddBuff(77), state = %v, want Blinded", c.Perception.State())
	}
	c.RemoveCondition(perception.ConditionIdFlashbangBlindness)
	if c.Perception.State() != perception.Sighted {
		t.Errorf("after RemoveBuff(77), state = %v, want Sighted", c.Perception.State())
	}
}

// PE-INT-007: Mixed source order — flashbang first, then buff 3, then the
// flashbang removed → still Blinded (buff 3 still active).
func TestIntegration_MixedSourceOrder(t *testing.T) {
	defer seedBlindConditions(t)()

	c := characters.New()
	if err := c.AddCondition(perception.ConditionIdFlashbangBlindness, false); err != nil {
		t.Fatalf("AddBuff(77): %v", err)
	}
	if c.Perception.State() != perception.Blinded {
		t.Fatalf("after AddBuff(77), state = %v, want Blinded", c.Perception.State())
	}
	if err := c.AddCondition(perception.ConditionIdBlinded, false); err != nil {
		t.Fatalf("AddBuff(3): %v", err)
	}
	// Re-adding-while-already-Blinded path; state must remain Blinded.
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after AddBuff(3) while already blinded, state = %v, want Blinded", c.Perception.State())
	}
	c.RemoveCondition(perception.ConditionIdFlashbangBlindness)
	// Buff 3 still active → still Blinded.
	if c.Perception.State() != perception.Blinded {
		t.Errorf("after RemoveBuff(77) (buff 3 still active), state = %v, want Blinded", c.Perception.State())
	}
	c.RemoveCondition(perception.ConditionIdBlinded)
	if c.Perception.State() != perception.Sighted {
		t.Errorf("after RemoveBuff (no sources left), state = %v, want Sighted", c.Perception.State())
	}
}
