package usercommands

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestPotionMagnitudeApplication(t *testing.T) {
	configs.SetConfigForTest(t, configs.GetConfig())
	heat := &conditions.ConditionSpec{ConditionId: 9801, Name: "Test Tincture", TriggerCount: 400,
		Effects: map[conditions.EffectKind]conditions.EffectValue{
			conditions.EffectNightVisionStrength: {Literal: 12}, conditions.EffectInfraReach: {UsesMagnitude: true}}}
	plain := &conditions.ConditionSpec{ConditionId: 9802, Name: "Test Brew", TriggerCount: 400}
	tincture := &items.ItemSpec{ItemId: 39998, Magnitude: 20}

	cases := []struct {
		name         string
		mult         float64
		wantMag      float64
		wantTriggers int
	}{
		{"fresh, alchemy 30", 1.0 * 1.3, 26, 520},
		{"peak, alchemy 50", 1.3 * 1.5, 39, 780},
		{"peak, alchemy 100 caps", 1.3 * 2.0, 50, 1040},
	}
	for _, c := range cases {
		mag, trig, ok := potionMagnitudeApplication(tincture, heat, c.mult)
		if !ok || math.Abs(mag-c.wantMag) > 1e-9 || trig != c.wantTriggers {
			t.Errorf("%s: (%v, %d, %v), want (%v, %d, true)", c.name, mag, trig, ok, c.wantMag, c.wantTriggers)
		}
	}
	if _, _, ok := potionMagnitudeApplication(tincture, plain, 1.3); ok {
		t.Error("a condition with no scaled kind keeps the duration-only path")
	}
	if _, _, ok := potionMagnitudeApplication(&items.ItemSpec{ItemId: 39997}, heat, 1.3); ok {
		t.Error("an item with no magnitude must not apply a zero-strength record")
	}
}
