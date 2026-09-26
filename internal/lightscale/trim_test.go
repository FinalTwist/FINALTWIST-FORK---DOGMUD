package lightscale

import (
	"math"
	"testing"
)

func TestTrimLightLandsExactlyOnTarget(t *testing.T) {
	for _, others := range []float64{0, 20, 50, 60, 73} {
		out := Trim(8, others, 100, 74, Brightens)
		if got := Combine(8, others, out); math.Abs(got-74) > 1e-9 {
			t.Errorf("others %v: Combine(others, Trim) = %v, want 74", others, got)
		}
	}
}

func TestTrimLightIsCappedAtFullStrength(t *testing.T) {
	if got := Trim(8, 0, 54, 74, Brightens); got != 54 {
		t.Errorf("a weak lantern in a faint room runs at %v, want its full 54", got)
	}
}

func TestTrimLightInAnUnlitRoomIsTheTarget(t *testing.T) {
	if got := Trim(8, Absent(), 90, 74, Brightens); got != 74 {
		t.Errorf("a strong glow in a cave trims to %v, want 74", got)
	}
	if got := Trim(8, Absent(), 54, 74, Brightens); got != 54 {
		t.Errorf("a weak lantern in a cave runs at %v, want 54", got)
	}
}

func TestTrimLightGoesDarkWhenTheRoomIsAlreadyBright(t *testing.T) {
	for _, others := range []float64{74, 80} {
		if got := Trim(8, others, 90, 74, Brightens); !math.IsInf(got, -1) {
			t.Errorf("others %v: Trim = %v, want Absent", others, got)
		}
	}
}

// Darkness is the same function inverted (5d wires it): the least darkness
// that keeps the room at or above the bearer's floor.
func TestTrimDarknessIsTheInverse(t *testing.T) {
	cases := []struct{ others, max, target, want float64 }{
		{70, 40, 25, 40},        // wants 45, capped at full strength
		{30, 40, 25, 5},         // cuts just to the floor
		{20, 40, 25, 0},         // already below the floor: no darkness
		{Absent(), 40, -30, 30}, // an unlit room counts as 0
	}
	for _, c := range cases {
		if got := Trim(8, c.others, c.max, c.target, Darkens); got != c.want {
			t.Errorf("Trim(dark, others %v, max %v, target %v) = %v, want %v", c.others, c.max, c.target, got, c.want)
		}
	}
}

// A room within a hair of the target needs a term below zero, the darkest
// natural light, so the source is not needed at all rather than "lit" at a
// meaningless negative value.
func TestTrimLightJustBelowTargetIsAbsent(t *testing.T) {
	for _, d := range []float64{1e-3, 1e-9, 1e-14, 1e-15} {
		if got := Trim(8, 74-d, 90, 74, Brightens); !math.IsInf(got, -1) {
			t.Errorf("others 74-%v: Trim = %v, want Absent", d, got)
		}
	}
	// Just far enough below that a real (non-negative) term is needed.
	if got := Trim(8, 70, 90, 74, Brightens); !(got >= 0 && got < 74) {
		t.Errorf("others 70: Trim = %v, want a term in [0, 74)", got)
	}
}

func TestTrimNaNTargetOrMaxIsNeverNaN(t *testing.T) {
	if got := Trim(8, 20, 90, math.NaN(), Brightens); !math.IsInf(got, -1) {
		t.Errorf("NaN target, light: Trim = %v, want Absent", got)
	}
	if got := Trim(8, 20, math.NaN(), 74, Brightens); !math.IsInf(got, -1) {
		t.Errorf("NaN max, light: Trim = %v, want Absent", got)
	}
	if got := Trim(8, 20, 90, math.NaN(), Darkens); got != 0 {
		t.Errorf("NaN target, darkness: Trim = %v, want 0", got)
	}
	if got := Trim(8, 20, math.NaN(), 74, Darkens); got != 0 {
		t.Errorf("NaN max, darkness: Trim = %v, want 0", got)
	}
}

// Mirrors TestNonPositiveStepDoesNotPanicOrNaN in lightscale_test.go: a
// non-positive step must not panic or hand back NaN or Inf, for either
// polarity.
func TestTrimNonPositiveStepDoesNotPanicOrNaN(t *testing.T) {
	for _, step := range []float64{0, -4} {
		if got := Trim(step, 20, 90, 74, Brightens); math.IsNaN(got) {
			t.Errorf("step %v, light: Trim = NaN", step)
		}
		if got := Trim(step, 20, 90, 74, Darkens); math.IsNaN(got) {
			t.Errorf("step %v, darkness: Trim = NaN", step)
		}
	}
}

func TestTrimNaNOthersBehavesAsAbsent(t *testing.T) {
	if got, want := Trim(8, math.NaN(), 90, 74, Brightens), Trim(8, Absent(), 90, 74, Brightens); got != want {
		t.Errorf("NaN others, light: Trim = %v, want %v (same as Absent others)", got, want)
	}
	if got := Trim(8, math.NaN(), 40, 25, Darkens); got != 0 {
		t.Errorf("NaN others, darkness: Trim = %v, want 0", got)
	}
}

func TestTrimDarknessNonPositiveMaxIsZero(t *testing.T) {
	for _, max := range []float64{0, -5} {
		if got := Trim(8, 70, max, 25, Darkens); got != 0 {
			t.Errorf("max %v: Trim(dark) = %v, want 0", max, got)
		}
	}
}

func TestTrimDarknessAtTargetIsZero(t *testing.T) {
	if got := Trim(8, 25, 40, 25, Darkens); got != 0 {
		t.Errorf("others == target: Trim(dark) = %v, want 0", got)
	}
}
