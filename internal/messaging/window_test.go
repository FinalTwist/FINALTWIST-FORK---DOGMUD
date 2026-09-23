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
		{"nv24, at the floor reads shapes", 1, 24, noReach, SightShapes},
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
