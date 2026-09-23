package messaging

import "testing"

// TestSightThroughWindow is the whole window model in one table. Every row is
// a light value and an observer's two numbers; the expected value is the tier
// that observer reads.
//
// The rows are taken from the spec's ladder and from the band edges, because a
// band model is wrong at its edges long before it is wrong in the middle.
func TestSightThroughWindow(t *testing.T) {
	// A normal observer: no shift, no reach.
	const normal, noReach = 0, 0

	tests := []struct {
		name     string
		light    int
		strength int
		reach    int
		want     SightDecision
	}{
		// Normal observer, unshifted bands 25 / 50 / 75.
		{"normal, pitch dark", 0, normal, noReach, SightNone},
		{"normal, just below blind", 24, normal, noReach, SightNone},
		{"normal, bottom of dim", 25, normal, noReach, SightShapes},
		{"normal, top of dim", 49, normal, noReach, SightShapes},
		{"normal, bottom of perfect", 50, normal, noReach, SightFull},
		{"normal, top of perfect", 74, normal, noReach, SightFull},
		{"normal, dazzled still sees", 75, normal, noReach, SightFull},
		{"normal, dazzled hard", 100, normal, noReach, SightFull},

		// Max nightvision: every edge drops 24, floor at 1.
		{"nv24, pitch dark is blind", 0, 24, noReach, SightNone},
		{"nv24, light at the shifted blind edge", 1, 24, noReach, SightShapes},
		{"nv24, top of shifted dim", 25, 24, noReach, SightShapes},
		{"nv24, bottom of shifted perfect", 26, 24, noReach, SightFull},
		{"nv24, top of shifted perfect", 50, 24, noReach, SightFull},
		{"nv24, dazzled in daylight", 70, 24, noReach, SightFull},

		// The cap: strength above the cap behaves as the cap.
		{"strength above cap clamps", 26, 99, noReach, SightFull},

		// Negative strength is nonsense and must not widen the window.
		{"negative strength behaves as zero", 25, -50, noReach, SightShapes},

		// Infra reach only matters at or below the floor.
		{"infra reach reads an unlit cave", 0, 24, 30, SightShapes},
		{"infra reach reads shallow magical dark", -30, 24, 30, SightShapes},
		{"infra reach bottoms out", -31, 24, 30, SightNone},
		{"reach does not help above the floor", 24, 0, 30, SightNone},
		{"reach without strength still reads dark", 0, 0, 10, SightShapes},

		// windowFloor itself is otherwise unpinnable: windowShiftCap (24) never
		// pushes shiftedBlind below 1, so the shapes branch's "light >=
		// windowFloor" term is dead for any real strength, and no existing row
		// puts a reach-carrying observer at light exactly windowFloor with no
		// strength to shift the blind edge out of the way. This row does: at
		// light 1 the shapes branch fails outright (1 is nowhere near the
		// unshifted blind edge of 25), so the result comes entirely from the
		// reach branch's "light <= windowFloor" gate. Move windowFloor to 0
		// and this reddens to SightNone.
		{"reach pins windowFloor at exactly 1", 1, 0, 5, SightShapes},
		// The companion edge: one step above the pin, reach has already
		// stopped mattering under either floor value (1 or 0), which is what
		// "reach only operates at or below the floor" is supposed to mean.
		{"reach stops one step above the floor", 2, 0, 5, SightNone},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SightThroughWindow(tc.light, tc.strength, tc.reach, 25, 50)
			if got != tc.want {
				t.Errorf("SightThroughWindow(light=%d, strength=%d, reach=%d) = %v, want %v",
					tc.light, tc.strength, tc.reach, got, tc.want)
			}
		})
	}
}
