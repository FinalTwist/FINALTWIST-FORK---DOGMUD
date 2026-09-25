package rooms

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/lightscale"
)

// Level must be exactly what LightLevel reports: the two share one computation.
func TestLightTermsLevelMatchesLightLevel(t *testing.T) {
	cfg := modelCfg()
	zero, half, open := 0.0, 0.5, 1.0
	lamp36, lamp55 := 36, 55
	for _, r := range []Room{
		{SkyLight: &zero},
		{SkyLight: &half},
		{SkyLight: &open, Lamp: &lamp36},
		{SkyLight: &open, Lamp: &lamp55},
		{SkyLight: &zero, Lamp: &lamp55},
	} {
		for _, celestial := range []float64{lightscale.Absent(), 0, 36, 60, 70} {
			for _, occ := range []int{0, 1, 2} {
				for _, mod := range []int{0, 1, 2} {
					want := r.lightLevelWithMutatorBridge(cfg, celestial, mod, occ)
					got := r.composeLight(cfg, celestial, mod, occ).Level
					if got != want {
						t.Errorf("sky=%v lamp=%v celestial=%v occ=%d mod=%d: terms.Level %d, LightLevel %d",
							r.SkyLight, r.Lamp, celestial, occ, mod, got, want)
					}
				}
			}
		}
	}
}

func TestLightTermsReportEachTerm(t *testing.T) {
	cfg := modelCfg()
	open, zero := 1.0, 0.0
	lamp := 40

	lit := Room{SkyLight: &open, Lamp: &lamp}
	got := lit.composeLight(cfg, 60, 1, 2)
	if !got.HasLamp || got.Lamp != 40 {
		t.Errorf("lamp = (%v, %d), want (true, 40)", got.HasLamp, got.Lamp)
	}
	if got.OcclusionSteps != 2 {
		t.Errorf("OcclusionSteps = %d, want 2", got.OcclusionSteps)
	}
	if got.LightMod != 1 {
		t.Errorf("LightMod = %d, want 1", got.LightMod)
	}
	wantSky := lightscale.Attenuate(cfg.DoublingStep, 60, 0.25)
	if math.Abs(got.Sky-wantSky) > 1e-9 {
		t.Errorf("Sky = %v, want %v (sky after two occlusion steps)", got.Sky, wantSky)
	}
	if got.Carried {
		t.Error("Carried = true with nobody in the room")
	}

	cave := Room{SkyLight: &zero}
	if c := cave.composeLight(cfg, 60, 0, 0); !math.IsInf(c.Sky, -1) || c.HasLamp {
		t.Errorf("a cave has no sky term and no lamp, got Sky=%v HasLamp=%v", c.Sky, c.HasLamp)
	}
}
