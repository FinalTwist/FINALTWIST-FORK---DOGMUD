package configs

import "testing"

// Graded lighting arc, plan 1 task 2. LightBlindBelow and LightDimBelow are
// validated as a PAIR, following the DarknessShapesCombatPenalty precedent,
// so a typo cannot ship a world where the shapes band is empty or inverted.
// LightExitsAbove is checked separately against LightBlindBelow only.

// TestLightingThresholds_ZeroDefaultsCorrectly covers the trap that a Go
// test binary never loads config.yaml, so every knob arrives zero-valued
// unless validation fills it in. Zero is coerced for all three lighting
// knobs (see the struct field comment for why), so an all-zero Balance
// must land on the arc's documented defaults.
func TestLightingThresholds_ZeroDefaultsCorrectly(t *testing.T) {
	b := Balance{}
	b.Validate()
	if b.LightBlindBelow != 25 {
		t.Fatalf("zero LightBlindBelow must default to 25, got %v", b.LightBlindBelow)
	}
	if b.LightDimBelow != 50 {
		t.Fatalf("zero LightDimBelow must default to 50, got %v", b.LightDimBelow)
	}
	if b.LightExitsAbove != 65 {
		t.Fatalf("zero LightExitsAbove must default to 65, got %v", b.LightExitsAbove)
	}
}

// TestLightingThresholds_InvertedBlindDimPairRevertsBoth proves an inverted
// pair (blind at or above dim) reverts BOTH knobs to their defaults, exactly
// as the field's own doc comment promises.
func TestLightingThresholds_InvertedBlindDimPairRevertsBoth(t *testing.T) {
	b := Balance{LightBlindBelow: 60, LightDimBelow: 40, LightExitsAbove: 70}
	b.Validate()
	if b.LightBlindBelow != 25 {
		t.Fatalf("inverted pair must revert LightBlindBelow to 25, got %v", b.LightBlindBelow)
	}
	if b.LightDimBelow != 50 {
		t.Fatalf("inverted pair must revert LightDimBelow to 50, got %v", b.LightDimBelow)
	}
	// LightExitsAbove was valid on its own (70 >= the reverted blind of 25)
	// and is not part of the blind/dim coherence check, so it survives.
	if b.LightExitsAbove != 70 {
		t.Fatalf("LightExitsAbove outside the blind/dim pair must survive, got %v", b.LightExitsAbove)
	}
}

// TestLightingThresholds_OutOfRangePairRevertsBoth covers a LightDimBelow
// above the -100..100 scale, which is out of range even though it is not
// inverted relative to LightBlindBelow.
func TestLightingThresholds_OutOfRangePairRevertsBoth(t *testing.T) {
	b := Balance{LightBlindBelow: 25, LightDimBelow: 150}
	b.Validate()
	if b.LightBlindBelow != 25 {
		t.Fatalf("out-of-range pair must revert LightBlindBelow to 25, got %v", b.LightBlindBelow)
	}
	if b.LightDimBelow != 50 {
		t.Fatalf("out-of-range pair must revert LightDimBelow to 50, got %v", b.LightDimBelow)
	}
}

// TestLightingThresholds_ExitsBelowBlindReverts proves LightExitsAbove is
// checked against LightBlindBelow: a value below the blind threshold would
// let a blind observer see through an exit, which the validator rejects.
func TestLightingThresholds_ExitsBelowBlindReverts(t *testing.T) {
	b := Balance{LightBlindBelow: 25, LightDimBelow: 50, LightExitsAbove: 10}
	b.Validate()
	if b.LightExitsAbove != 65 {
		t.Fatalf("LightExitsAbove below LightBlindBelow must revert to 65, got %v", b.LightExitsAbove)
	}
	// The blind/dim pair was valid on its own and must not be touched by an
	// unrelated LightExitsAbove failure.
	if b.LightBlindBelow != 25 || b.LightDimBelow != 50 {
		t.Fatalf("a LightExitsAbove failure must not revert the blind/dim pair, got blind=%v dim=%v", b.LightBlindBelow, b.LightDimBelow)
	}
}

// TestLightingThresholds_ExitsOutOfRangeReverts covers a LightExitsAbove
// above the -100..100 scale.
func TestLightingThresholds_ExitsOutOfRangeReverts(t *testing.T) {
	b := Balance{LightBlindBelow: 25, LightDimBelow: 50, LightExitsAbove: 200}
	b.Validate()
	if b.LightExitsAbove != 65 {
		t.Fatalf("out-of-range LightExitsAbove must revert to 65, got %v", b.LightExitsAbove)
	}
}

// TestLightingThresholds_AuthoredSetSurvives proves a legal, in-range,
// correctly-ordered set of all three knobs passes validation untouched.
// This is the case that matters most: a validator that clobbers legitimate
// operator values is worse than no validator.
func TestLightingThresholds_AuthoredSetSurvives(t *testing.T) {
	b := Balance{LightBlindBelow: 20, LightDimBelow: 55, LightExitsAbove: 72}
	b.Validate()
	if b.LightBlindBelow != 20 {
		t.Fatalf("authored LightBlindBelow must survive validation, got %v", b.LightBlindBelow)
	}
	if b.LightDimBelow != 55 {
		t.Fatalf("authored LightDimBelow must survive validation, got %v", b.LightDimBelow)
	}
	if b.LightExitsAbove != 72 {
		t.Fatalf("authored LightExitsAbove must survive validation, got %v", b.LightExitsAbove)
	}
}

// TestLightingThresholds_NegativeBlindBelowIsTheNeverBlindEscapeHatch proves
// the documented way to disable blindness entirely: LightBlindBelow at the
// scale floor of -100 is honoured, not coerced, because it is not zero.
func TestLightingThresholds_NegativeBlindBelowIsTheNeverBlindEscapeHatch(t *testing.T) {
	b := Balance{LightBlindBelow: -100, LightDimBelow: 50, LightExitsAbove: 65}
	b.Validate()
	if b.LightBlindBelow != -100 {
		t.Fatalf("LightBlindBelow of -100 must survive validation as the never-blind escape hatch, got %v", b.LightBlindBelow)
	}
}

// TestLightingThresholds_ExitsEqualToBlindSurvives proves LightExitsAbove
// may equal LightBlindBelow (exits visible the instant an observer is no
// longer blind) without being treated as invalid.
func TestLightingThresholds_ExitsEqualToBlindSurvives(t *testing.T) {
	b := Balance{LightBlindBelow: 25, LightDimBelow: 50, LightExitsAbove: 25}
	b.Validate()
	if b.LightExitsAbove != 25 {
		t.Fatalf("LightExitsAbove equal to LightBlindBelow must survive validation, got %v", b.LightExitsAbove)
	}
}

// TestLightingThresholds_EqualBlindDimPairReverts pins the boundary the
// InvertedPair test does not reach: blind and dim EQUAL, not just inverted.
// The comparison is `>=`, not `>`, precisely so this case reverts too; an
// equal pair leaves the shapes band empty, which the struct comment already
// says is invalid. Without this test, `>=` reads as an arbitrary choice a
// future editor could "fix" to `>` to match the DarknessShapesCombatPenalty
// precedent, where an equal pair is allowed.
func TestLightingThresholds_EqualBlindDimPairReverts(t *testing.T) {
	b := Balance{LightBlindBelow: 40, LightDimBelow: 40, LightExitsAbove: 65}
	b.Validate()
	if b.LightBlindBelow != 25 {
		t.Fatalf("equal pair must revert LightBlindBelow to 25, got %v", b.LightBlindBelow)
	}
	if b.LightDimBelow != 50 {
		t.Fatalf("equal pair must revert LightDimBelow to 50, got %v", b.LightDimBelow)
	}
}

// The remaining tests pin the -100..100 clamp at its actual edges rather
// than from the middle (LightDimBelow: 150 and LightExitsAbove: 200 above
// only prove "clearly out of range reverts", not that the cutoff sits at
// exactly 100/-100 rather than off by one).
//
// Some edges are unreachable and are deliberately NOT tested, because the
// three knobs constrain each other:
//   - LightBlindBelow can never survive AT exactly 100: LightDimBelow must
//     be strictly greater and cannot exceed 100 itself, so no valid dim
//     exists above a blind of 100.
//   - LightDimBelow can never survive AT exactly -100: LightBlindBelow must
//     be strictly less and cannot go below -100 itself, so no valid blind
//     exists below a dim of -100.
// Testing either would assert a case this validator cannot produce.

// TestLightingThresholds_BlindLowerBoundRejected pins -101 as rejected. -100
// itself is already pinned accepted by the never-blind escape hatch test
// above. Dim is a comfortably valid 50 so only the range check can fire
// (blind -101 < dim 50, so the order check does not).
func TestLightingThresholds_BlindLowerBoundRejected(t *testing.T) {
	b := Balance{LightBlindBelow: -101, LightDimBelow: 50}
	b.Validate()
	if b.LightBlindBelow != 25 || b.LightDimBelow != 50 {
		t.Fatalf("LightBlindBelow below -100 must revert the pair to 25/50, got blind=%v dim=%v", b.LightBlindBelow, b.LightDimBelow)
	}
}

// TestLightingThresholds_BlindUpperBoundRejected pins 101 as rejected. The
// mirror accept case (exactly 100) is unreachable, per the note above, so
// only the reject side exists to pin. This value also fails the order check
// (no valid dim can exceed it), which is unavoidable rather than a gap in
// isolation.
func TestLightingThresholds_BlindUpperBoundRejected(t *testing.T) {
	b := Balance{LightBlindBelow: 101, LightDimBelow: 50}
	b.Validate()
	if b.LightBlindBelow != 25 || b.LightDimBelow != 50 {
		t.Fatalf("LightBlindBelow above 100 must revert the pair to 25/50, got blind=%v dim=%v", b.LightBlindBelow, b.LightDimBelow)
	}
}

// TestLightingThresholds_DimUpperBoundAccepted pins exactly 100 as accepted
// for LightDimBelow, isolated from the order check by a comfortably lower
// blind.
func TestLightingThresholds_DimUpperBoundAccepted(t *testing.T) {
	b := Balance{LightBlindBelow: 25, LightDimBelow: 100}
	b.Validate()
	if b.LightBlindBelow != 25 || b.LightDimBelow != 100 {
		t.Fatalf("LightDimBelow of exactly 100 must survive validation, got blind=%v dim=%v", b.LightBlindBelow, b.LightDimBelow)
	}
}

// TestLightingThresholds_DimUpperBoundRejected pins 101 as rejected, isolated
// from the order check (25 < 101, so only the range check fires).
func TestLightingThresholds_DimUpperBoundRejected(t *testing.T) {
	b := Balance{LightBlindBelow: 25, LightDimBelow: 101}
	b.Validate()
	if b.LightBlindBelow != 25 || b.LightDimBelow != 50 {
		t.Fatalf("LightDimBelow above 100 must revert the pair to 25/50, got blind=%v dim=%v", b.LightBlindBelow, b.LightDimBelow)
	}
}

// TestLightingThresholds_DimLowerBoundRejected pins -101 as rejected. The
// mirror accept case (exactly -100) is unreachable, per the note above. This
// value also fails the order check (no valid blind can sit below it), which
// is unavoidable rather than a gap in isolation.
func TestLightingThresholds_DimLowerBoundRejected(t *testing.T) {
	b := Balance{LightBlindBelow: -100, LightDimBelow: -101}
	b.Validate()
	if b.LightBlindBelow != 25 || b.LightDimBelow != 50 {
		t.Fatalf("LightDimBelow below -100 must revert the pair to 25/50, got blind=%v dim=%v", b.LightBlindBelow, b.LightDimBelow)
	}
}

// TestLightingThresholds_ExitsLowerBoundAccepted pins exactly -100 as
// accepted for LightExitsAbove. The only blind value at or below -100 is
// -100 itself, so that is what pairs with it here.
func TestLightingThresholds_ExitsLowerBoundAccepted(t *testing.T) {
	b := Balance{LightBlindBelow: -100, LightDimBelow: 50, LightExitsAbove: -100}
	b.Validate()
	if b.LightExitsAbove != -100 {
		t.Fatalf("LightExitsAbove of exactly -100 must survive validation, got %v", b.LightExitsAbove)
	}
}

// TestLightingThresholds_ExitsLowerBoundRejected pins -101 as rejected. This
// value also fails the exits-versus-blind cross-check (no valid blind can
// sit below it), which is unavoidable rather than a gap in isolation.
func TestLightingThresholds_ExitsLowerBoundRejected(t *testing.T) {
	b := Balance{LightBlindBelow: -100, LightDimBelow: 50, LightExitsAbove: -101}
	b.Validate()
	if b.LightExitsAbove != 65 {
		t.Fatalf("LightExitsAbove below -100 must revert to 65, got %v", b.LightExitsAbove)
	}
}

// TestLightingThresholds_ExitsUpperBoundAccepted pins exactly 100 as
// accepted for LightExitsAbove, isolated from the cross-check by a
// comfortably lower blind.
func TestLightingThresholds_ExitsUpperBoundAccepted(t *testing.T) {
	b := Balance{LightBlindBelow: 25, LightDimBelow: 50, LightExitsAbove: 100}
	b.Validate()
	if b.LightExitsAbove != 100 {
		t.Fatalf("LightExitsAbove of exactly 100 must survive validation, got %v", b.LightExitsAbove)
	}
}

// TestLightingThresholds_ExitsUpperBoundRejected pins 101 as rejected,
// isolated from the cross-check (101 is not below blind's 25, so only the
// range check fires).
func TestLightingThresholds_ExitsUpperBoundRejected(t *testing.T) {
	b := Balance{LightBlindBelow: 25, LightDimBelow: 50, LightExitsAbove: 101}
	b.Validate()
	if b.LightExitsAbove != 65 {
		t.Fatalf("LightExitsAbove above 100 must revert to 65, got %v", b.LightExitsAbove)
	}
}

// TestLightingThresholds_ElevatedBlindWithInvalidExitsDoesNotFallBelowBlind
// is the case the whole-arc review found: every other test in this file
// authors the default LightBlindBelow of 25, and the hardcoded exits
// fallback of 65 is always above 25, so no existing test could see this.
// An operator who legitimately raises LightBlindBelow above 65 and
// separately authors an invalid LightExitsAbove used to get the fallback
// 65 unconditionally, landing BELOW their own valid blind threshold: the
// exact contradiction (a blind observer who can see through an exit) this
// validator exists to prevent. The fallback must be re-checked against the
// final LightBlindBelow, not assigned and left alone.
func TestLightingThresholds_ElevatedBlindWithInvalidExitsDoesNotFallBelowBlind(t *testing.T) {
	b := Balance{LightBlindBelow: 90, LightDimBelow: 95, LightExitsAbove: -50}
	b.Validate()
	if b.LightBlindBelow != 90 || b.LightDimBelow != 95 {
		t.Fatalf("a valid blind/dim pair must survive untouched, got blind=%v dim=%v", b.LightBlindBelow, b.LightDimBelow)
	}
	if b.LightExitsAbove < b.LightBlindBelow {
		t.Fatalf("LightExitsAbove fallback must never sit below the final LightBlindBelow, got exits=%v blind=%v", b.LightExitsAbove, b.LightBlindBelow)
	}
	if b.LightExitsAbove != 90 {
		t.Fatalf("LightExitsAbove fallback should clamp up to the elevated LightBlindBelow of 90, got %v", b.LightExitsAbove)
	}
}

// TestLightingThresholds_ElevatedBlindWithOutOfRangeExitsDoesNotFallBelowBlind
// is the same hazard reached through the range check instead of the
// cross-axis check: LightExitsAbove is out of range (200) rather than
// merely below blind, but the unconditional 65 fallback would still land
// below an elevated LightBlindBelow.
func TestLightingThresholds_ElevatedBlindWithOutOfRangeExitsDoesNotFallBelowBlind(t *testing.T) {
	b := Balance{LightBlindBelow: 90, LightDimBelow: 95, LightExitsAbove: 200}
	b.Validate()
	if b.LightExitsAbove < b.LightBlindBelow {
		t.Fatalf("LightExitsAbove fallback must never sit below the final LightBlindBelow, got exits=%v blind=%v", b.LightExitsAbove, b.LightBlindBelow)
	}
	if b.LightExitsAbove != 90 {
		t.Fatalf("LightExitsAbove fallback should clamp up to the elevated LightBlindBelow of 90, got %v", b.LightExitsAbove)
	}
}

// TestLightingThresholds_ShippedDefaultsUnchangedByFallbackFix pins that the
// fallback fix does not move the arc's shipped defaults: an all-zero
// Balance, exactly what a Go test binary loads, must still land on
// 25/50/65. The clamp-up branch must never fire when the final
// LightBlindBelow is at or below the hardcoded 65.
func TestLightingThresholds_ShippedDefaultsUnchangedByFallbackFix(t *testing.T) {
	b := Balance{}
	b.Validate()
	if b.LightBlindBelow != 25 || b.LightDimBelow != 50 || b.LightExitsAbove != 65 {
		t.Fatalf("shipped defaults must remain 25/50/65, got blind=%v dim=%v exits=%v", b.LightBlindBelow, b.LightDimBelow, b.LightExitsAbove)
	}
}
