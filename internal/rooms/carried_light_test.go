package rooms

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/lightscale"
)

// Every carried source is its own term: two equal torches are one doubling
// step brighter than one, not the flat single term plan 1 shipped.
func TestCarriedSourcesEachJoinTheCombine(t *testing.T) {
	cfg := modelCfg()
	zero := 0.0
	cave := Room{SkyLight: &zero}

	one := cave.composeWith(cfg, 60, 1, []float64{56})
	two := cave.composeWith(cfg, 60, 1, []float64{56, 56})
	if one.Level != 56 {
		t.Errorf("one torch in a cave = %d, want 56", one.Level)
	}
	if two.Level != 64 {
		t.Errorf("two torches in a cave = %d, want 64 (one step of 8 brighter)", two.Level)
	}
	if !one.Carried || cave.composeWith(cfg, 60, 1, nil).Carried {
		t.Error("Carried must be true exactly when a carried term is present")
	}
	if want := lightscale.Combine(cfg.DoublingStep, 56, 56); math.Abs(two.Raw-want) > 1e-9 {
		t.Errorf("Raw = %v, want %v", two.Raw, want)
	}
	if empty := cave.composeWith(cfg, 60, 1, nil); !math.IsInf(empty.Raw, -1) || empty.Level != 0 {
		t.Errorf("an empty cave: Raw %v Level %d, want -Inf and 0", empty.Raw, empty.Level)
	}
}
