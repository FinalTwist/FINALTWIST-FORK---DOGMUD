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
