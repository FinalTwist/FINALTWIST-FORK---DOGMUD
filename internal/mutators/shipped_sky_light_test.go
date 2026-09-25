package mutators

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/fileloader"
)

// shippedMutatorDir is the dogmud world's mutator folder. A test binary never
// reads config.yaml, so this test loads it explicitly.
const shippedMutatorDir = "../../_datafiles/world/dogmud/mutators"

func loadShippedMutators(t *testing.T) map[string]*MutatorSpec {
	t.Helper()
	specs, err := fileloader.LoadAllFlatFiles[string, *MutatorSpec](shippedMutatorDir)
	if err != nil {
		t.Fatalf("loading shipped mutators: %v", err)
	}
	if len(specs) == 0 {
		t.Fatal("no shipped mutators loaded; the test would be vacuous")
	}
	return specs
}

// TestShippedWeatherSkyLight pins the owner's gentle grading: light weather
// lets 0.7 of the sky through, heavy weather 0.5, and nothing else filters it.
func TestShippedWeatherSkyLight(t *testing.T) {
	want := map[string]float64{
		"weather-fog":      0.7,
		"weather-overcast": 0.7,
		"weather-rain":     0.7,
		"weather-snow":     0.7,
		"weather-storm":    0.5,
		"weather-dust":     0.5,
		"weather-blizzard": 0.5,
	}
	for id, spec := range loadShippedMutators(t) {
		w, filtered := want[id]
		switch {
		case filtered && spec.SkyLight == nil:
			t.Errorf("%s: no skylight, want %v", id, w)
		case filtered && *spec.SkyLight != w:
			t.Errorf("%s: skylight %v, want %v", id, *spec.SkyLight, w)
		case !filtered && spec.SkyLight != nil:
			t.Errorf("%s: skylight %v, but only weather filters the sky", id, *spec.SkyLight)
		}
		delete(want, id)
	}
	for id := range want {
		t.Errorf("%s is not shipped", id)
	}
}
