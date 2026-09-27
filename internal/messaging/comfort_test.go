package messaging

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// Normal eyes: blind 25, dim 50, dazzle 75, width 25.
func TestComfortDistanceNormalEyes(t *testing.T) {
	cases := []struct {
		light        int
		dark, bright float64
	}{
		{60, 0, 0}, {50, 0, 0}, {74, 0, 0},
		{37, 0.52, 0}, {26, 0.96, 0}, {25, 1, 0}, {0, 1, 0},
		{75, 0, 0}, {90, 0, 0.6}, {100, 0, 1},
	}
	for _, c := range cases {
		d, b := comfortDistance(c.light, 0, 25, 50, 75)
		if !near(d, c.dark) || !near(b, c.bright) {
			t.Errorf("light %d: (%v, %v), want (%v, %v)", c.light, d, b, c.dark, c.bright)
		}
	}
}

// Nightvision 24: edges 1, 26, 51; the bright ramp is still 25 wide, so the
// cap arrives at 76, well inside daylight.
func TestComfortDistanceShiftedWindow(t *testing.T) {
	cases := []struct {
		light        int
		dark, bright float64
	}{
		{30, 0, 0}, {20, 0.24, 0}, {1, 1, 0},
		{70, 0, 0.76}, {73, 0, 0.88}, {76, 0, 1}, {90, 0, 1},
	}
	for _, c := range cases {
		d, b := comfortDistance(c.light, 24, 25, 50, 75)
		if !near(d, c.dark) || !near(b, c.bright) {
			t.Errorf("light %d: (%v, %v), want (%v, %v)", c.light, d, b, c.dark, c.bright)
		}
	}
}

func TestComfortDistanceStrengthIsClamped(t *testing.T) {
	d1, b1 := comfortDistance(70, 40, 25, 50, 75)
	d2, b2 := comfortDistance(70, 24, 25, 50, 75)
	if d1 != d2 || b1 != b2 {
		t.Error("a strength above the cap must clamp exactly as the window does")
	}
}

// A Blinded observer is fully dark, even in a room that would otherwise
// dazzle: ComfortDistance never lets bright light substitute for sight.
func TestComfortDistanceBlindedIsFullyDark(t *testing.T) {
	c := newChar(t)
	setBlinded(t, c)
	dark, bright := ComfortDistance(c, sightLight(100))
	if dark != 1 || bright != 0 {
		t.Errorf("Blinded observer in a blazing room = (%v, %v), want (1, 0)", dark, bright)
	}
}
