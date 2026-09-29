package items

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestPotionMagnitudeApplication(t *testing.T) {
	configs.SetConfigForTest(t, configs.GetConfig())
	heat := &conditions.ConditionSpec{ConditionId: 9801, Name: "Test Tincture", TriggerCount: 400,
		Effects: map[conditions.EffectKind]conditions.EffectValue{
			conditions.EffectNightVisionStrength: {Literal: 12}, conditions.EffectInfraReach: {UsesMagnitude: true}}}
	plain := &conditions.ConditionSpec{ConditionId: 9802, Name: "Test Brew", TriggerCount: 400}
	tincture := &ItemSpec{ItemId: 39998, Magnitude: 20}

	cases := []struct {
		name         string
		mult         float64
		wantMag      float64
		wantTriggers int
	}{
		{"mob drink, unscaled", 1.0, 20, 400},
		{"fresh, alchemy 30", 1.0 * 1.3, 26, 520},
		{"peak, alchemy 50", 1.3 * 1.5, 39, 780},
		{"peak, alchemy 100 caps", 1.3 * 2.0, 50, 1040},
	}
	for _, c := range cases {
		mag, trig, ok := PotionMagnitudeApplication(tincture, heat, c.mult)
		if !ok || math.Abs(mag-c.wantMag) > 1e-9 || trig != c.wantTriggers {
			t.Errorf("%s: (%v, %d, %v), want (%v, %d, true)", c.name, mag, trig, ok, c.wantMag, c.wantTriggers)
		}
	}
	night := &conditions.ConditionSpec{ConditionId: 9803, Name: "Test Night Draught", TriggerCount: 100,
		Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectNightVisionStrength: {UsesMagnitude: true}}}
	if mag, _, ok := PotionMagnitudeApplication(&ItemSpec{ItemId: 39996, Magnitude: 20}, night, 1.5); !ok || mag != configs.LightWindowShiftCap {
		t.Errorf("a nightvision potion's 30 caps at the window shift cap: (%v, %v)", mag, ok)
	}
	if _, _, ok := PotionMagnitudeApplication(tincture, plain, 1.3); ok {
		t.Error("a condition with no scaled kind keeps the duration-only path")
	}
	if _, _, ok := PotionMagnitudeApplication(&ItemSpec{ItemId: 39997}, heat, 1.3); ok {
		t.Error("an item with no magnitude must not apply a zero-strength record")
	}
}
