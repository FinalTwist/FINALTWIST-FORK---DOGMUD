package messaging

import "testing"

func TestLightTrimTargetSitsJustUnderTheDazzleEdge(t *testing.T) {
	cases := []struct{ strength, want int }{{0, 74}, {24, 50}, {40, 50}, {-5, 74}}
	for _, c := range cases {
		if got := LightTrimTarget(c.strength, 75); got != float64(c.want) {
			t.Errorf("LightTrimTarget(%d) = %v, want %d", c.strength, got, c.want)
		}
		// The target itself must read as faces, never dazzled, for that observer.
		if b := BandThroughWindow(c.want, c.strength, 0, 25, 50, 75); b != BandFaces {
			t.Errorf("strength %d: the target %d reads %v, want faces", c.strength, c.want, b)
		}
	}
}
