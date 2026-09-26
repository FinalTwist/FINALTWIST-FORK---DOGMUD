package messaging

import "testing"

// TestBandThroughWindowEdges pins every band edge at strength 0 and at the
// capped strength 24, plus infrared reach. Edges use the shipped 25/50.
func TestBandThroughWindowEdges(t *testing.T) {
	cases := []struct {
		name     string
		light    int
		strength int
		reach    int
		want     Band
	}{
		{"s0 pitch dark", 0, 0, 0, BandDark},
		{"s0 just below blind", 24, 0, 0, BandDark},
		{"s0 at blind edge", 25, 0, 0, BandShapes},
		{"s0 just below dim", 49, 0, 0, BandShapes},
		{"s0 at dim edge", 50, 0, 0, BandFaces},
		{"s0 just below dazzle", 74, 0, 0, BandFaces},
		{"s0 at dazzle edge", 75, 0, 0, BandDazzled},
		{"s24 below floor", 0, 24, 0, BandDark},
		{"s24 at floor", 1, 24, 0, BandShapes},
		{"s24 at shifted dim", 26, 24, 0, BandFaces},
		{"s24 just below shifted dazzle", 50, 24, 0, BandFaces},
		{"s24 at shifted dazzle", 51, 24, 0, BandDazzled},
		{"s30 caps at 24", 51, 30, 0, BandDazzled},
		{"negative strength reads as 0", 74, -5, 0, BandFaces},
		{"reach reads shapes in the dark", 0, 0, 10, BandShapes},
		{"reach has a limit", -11, 0, 10, BandDark},
	}
	for _, c := range cases {
		if got := BandThroughWindow(c.light, c.strength, c.reach, 25, 50); got != c.want {
			t.Errorf("%s: BandThroughWindow(%d, s=%d, r=%d) = %v, want %v",
				c.name, c.light, c.strength, c.reach, got, c.want)
		}
	}
}

// TestLightBandAgreesWithParticipantSight is the guard that keeps the two
// optics answers from drifting: LightBand only ever SPLITS ParticipantSight's
// full tier into faces and dazzled, never moves an edge.
func TestLightBandAgreesWithParticipantSight(t *testing.T) {
	c := newChar(t)
	for light := -30; light <= 100; light++ {
		room := sightLight(light)
		band := LightBand(c, room)
		sight := ParticipantSight(c, room)
		var want SightDecision
		switch band {
		case BandDark:
			want = SightNone
		case BandShapes:
			want = SightShapes
		default:
			want = SightFull
		}
		if sight != want {
			t.Errorf("light %d: LightBand %v but ParticipantSight %v", light, band, sight)
		}
	}
}

func TestLightBandBlindedIsDark(t *testing.T) {
	c := newChar(t)
	setBlinded(t, c)
	if got := LightBand(c, sightLight(100)); got != BandDark {
		t.Fatalf("a Blinded observer in a blazing room = %v, want dark", got)
	}
}

func TestLightBandNilsReadFaces(t *testing.T) {
	if got := LightBand(nil, sightLight(0)); got != BandFaces {
		t.Errorf("nil observer = %v, want faces", got)
	}
	if got := LightBand(newChar(t), nil); got != BandFaces {
		t.Errorf("nil room = %v, want faces", got)
	}
}

func TestBandString(t *testing.T) {
	for b, want := range map[Band]string{BandDark: "dark", BandShapes: "shapes", BandFaces: "faces", BandDazzled: "dazzled"} {
		if b.String() != want {
			t.Errorf("Band(%d).String() = %q, want %q", b, b.String(), want)
		}
	}
}
