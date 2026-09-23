package lightscale

import (
	"math"
	"testing"
)

func TestCombineOfNothingIsAbsent(t *testing.T) {
	if got := Combine(8, Absent(), Absent()); !math.IsInf(got, -1) {
		t.Fatalf("want -Inf, got %v", got)
	}
}

func TestCombineOfOneTermIsThatTerm(t *testing.T) {
	if got := Combine(8, 35, Absent()); got != 35 {
		t.Fatalf("want 35, got %v", got)
	}
}

// Two equal sources read exactly one doubling step brighter. This is the
// definition of the step, so it is the load-bearing test of the whole scale.
func TestTwoEqualSourcesAddOneStep(t *testing.T) {
	got := Combine(8, 35, 35)
	if math.Abs(got-43) > 1e-9 {
		t.Fatalf("want 43, got %v", got)
	}
}

func TestFourEqualSourcesAddTwoSteps(t *testing.T) {
	got := Combine(8, 35, 35, 35, 35)
	if math.Abs(got-51) > 1e-9 {
		t.Fatalf("want 51, got %v", got)
	}
}

// A much weaker source adds almost nothing. This is what stops a lantern from
// mattering at noon, which the halving rule the spec originally used could not
// express.
//
// The plan's original bound here was (70, 70.05]. That is wrong: the same
// combine formula the two equal-source tests above pin gives
// best + step*log2(1 + 2^((30-70)/8)) = 70 + 8*log2(1.03125) =
// 70.35515295486763, about seven times the plan's stated ceiling. Verified by
// computing it independently in Python, not just by running this package's
// own code. The bound below is corrected to the real value with a small
// float-precision tolerance.
func TestFarWeakerSourceBarelyContributes(t *testing.T) {
	got := Combine(8, 70, 30)
	if got <= 70 || math.Abs(got-70.35515295486763) > 1e-9 {
		t.Fatalf("want approx 70.355, got %v", got)
	}
}

func TestCombineIsOrderIndependent(t *testing.T) {
	a := Combine(8, 12, 55, 31)
	b := Combine(8, 55, 31, 12)
	if math.Abs(a-b) > 1e-9 {
		t.Fatalf("%v != %v", a, b)
	}
}

func TestAttenuateHalfIsMinusOneStep(t *testing.T) {
	got := Attenuate(8, 60, 0.5)
	if math.Abs(got-52) > 1e-9 {
		t.Fatalf("want 52, got %v", got)
	}
}

func TestAttenuateByOneIsIdentity(t *testing.T) {
	if got := Attenuate(8, 60, 1); got != 60 {
		t.Fatalf("want 60, got %v", got)
	}
}

// A sky fraction of zero is not "contributes zero", it is "there is no sky
// here". A cave must not receive a term at all.
func TestAttenuateByZeroIsAbsent(t *testing.T) {
	if got := Attenuate(8, 60, 0); !math.IsInf(got, -1) {
		t.Fatalf("want -Inf, got %v", got)
	}
}

func TestAttenuateOfAbsentIsAbsent(t *testing.T) {
	if got := Attenuate(8, Absent(), 0.5); !math.IsInf(got, -1) {
		t.Fatalf("want -Inf, got %v", got)
	}
}

func TestNonPositiveStepDoesNotPanicOrNaN(t *testing.T) {
	if got := Combine(0, 10, 10); math.IsNaN(got) {
		t.Fatalf("NaN from zero step")
	}
	if got := Attenuate(-4, 10, 0.5); math.IsNaN(got) {
		t.Fatalf("NaN from negative step")
	}
}
