package mutators

import (
	"math"
	"testing"
)

func TestSkyLightValidate(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	for _, ok := range []*float64{nil, f(0), f(0.5), f(1)} {
		spec := MutatorSpec{MutatorId: "sky-ok", SkyLight: ok}
		if err := spec.Validate(); err != nil {
			t.Errorf("skylight %v refused: %v", ok, err)
		}
	}
	for _, bad := range []float64{-0.1, 1.5, math.NaN()} {
		spec := MutatorSpec{MutatorId: "sky-bad", SkyLight: f(bad)}
		if err := spec.Validate(); err == nil {
			t.Errorf("skylight %v accepted; it must lie within 0 to 1", bad)
		}
	}
}
