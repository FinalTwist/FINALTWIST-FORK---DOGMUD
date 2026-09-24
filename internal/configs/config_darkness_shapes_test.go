package configs

import "testing"

// M4d PR 2, owner ruling 6 (2026-09-20): an infrared combatant who only
// makes out SHAPES in the dark takes a REDUCED darkness combat penalty, not
// the full blind penalty and not zero. DarknessShapesCombatPenalty is
// validated as a PAIR with DarknessCombatPenalty so a typo cannot ship a
// world where seeing shapes is worse than seeing nothing.

// TestDarknessShapesCombatPenalty_ZeroRejected covers the trap that a Go
// test binary never loads config.yaml, so every knob arrives zero-valued
// unless validation fills it in.
func TestDarknessShapesCombatPenalty_ZeroRejected(t *testing.T) {
	b := Balance{}
	b.Validate()
	if b.DarknessShapesCombatPenalty != 0.90 {
		t.Fatalf("zero DarknessShapesCombatPenalty must default to 0.90, got %v", b.DarknessShapesCombatPenalty)
	}
	if b.DarknessCombatPenalty != 0.80 {
		t.Fatalf("zero DarknessCombatPenalty must default to 0.80, got %v", b.DarknessCombatPenalty)
	}
}

// TestDarknessShapesCombatPenalty_InvertedPairRevertsBoth proves an inverted
// pair (shapes worse than blind) reverts BOTH knobs to their defaults,
// exactly as the field's own doc comment promises.
func TestDarknessShapesCombatPenalty_InvertedPairRevertsBoth(t *testing.T) {
	b := Balance{DarknessCombatPenalty: 0.80, DarknessShapesCombatPenalty: 0.50}
	b.Validate()
	if b.DarknessCombatPenalty != 0.80 {
		t.Fatalf("inverted pair must revert DarknessCombatPenalty to 0.80, got %v", b.DarknessCombatPenalty)
	}
	if b.DarknessShapesCombatPenalty != 0.90 {
		t.Fatalf("inverted pair must revert DarknessShapesCombatPenalty to 0.90, got %v", b.DarknessShapesCombatPenalty)
	}
}

// TestDarknessShapesCombatPenalty_OutOfRangeRevertsBoth covers a
// DarknessShapesCombatPenalty above 1.0, which is out of range even though
// it is not inverted relative to DarknessCombatPenalty.
func TestDarknessShapesCombatPenalty_OutOfRangeRevertsBoth(t *testing.T) {
	b := Balance{DarknessCombatPenalty: 0.80, DarknessShapesCombatPenalty: 1.5}
	b.Validate()
	if b.DarknessCombatPenalty != 0.80 {
		t.Fatalf("out-of-range pair must revert DarknessCombatPenalty to 0.80, got %v", b.DarknessCombatPenalty)
	}
	if b.DarknessShapesCombatPenalty != 0.90 {
		t.Fatalf("out-of-range pair must revert DarknessShapesCombatPenalty to 0.90, got %v", b.DarknessShapesCombatPenalty)
	}
}

// TestDarknessShapesCombatPenalty_AuthoredPairSurvives proves a legal,
// in-range, correctly-ordered pair passes validation untouched.
func TestDarknessShapesCombatPenalty_AuthoredPairSurvives(t *testing.T) {
	b := Balance{DarknessCombatPenalty: 0.70, DarknessShapesCombatPenalty: 0.85}
	b.Validate()
	if b.DarknessCombatPenalty != 0.70 {
		t.Fatalf("authored DarknessCombatPenalty must survive validation, got %v", b.DarknessCombatPenalty)
	}
	if b.DarknessShapesCombatPenalty != 0.85 {
		t.Fatalf("authored DarknessShapesCombatPenalty must survive validation, got %v", b.DarknessShapesCombatPenalty)
	}
}

// TestDarknessShapesCombatPenalty_EqualToBlindSurvives proves the pair may
// be equal (shapes exactly as bad as blind, just never worse) without being
// treated as inverted.
func TestDarknessShapesCombatPenalty_EqualToBlindSurvives(t *testing.T) {
	b := Balance{DarknessCombatPenalty: 0.80, DarknessShapesCombatPenalty: 0.80}
	b.Validate()
	if b.DarknessShapesCombatPenalty != 0.80 {
		t.Fatalf("shapes penalty equal to the blind penalty must survive validation, got %v", b.DarknessShapesCombatPenalty)
	}
}
