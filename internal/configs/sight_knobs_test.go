package configs

import "testing"

func TestSightKnobDefaultsAndValidation(t *testing.T) {
	var b Balance
	b.Validate()
	if b.DarknessCombatPenalty != 0.80 || b.DazzleCap != 0.80 || b.LightDazzleAbove != 75 {
		t.Fatalf("defaults: dark %v dazzle %v edge %v", b.DarknessCombatPenalty, b.DazzleCap, b.LightDazzleAbove)
	}
	b.DazzleCap = 1.5
	b.LightDazzleAbove = 40 // not above LightDimBelow (50)
	b.Validate()
	if b.DazzleCap != 0.80 || b.LightDazzleAbove != 75 {
		t.Errorf("out of range did not revert: dazzle %v edge %v", b.DazzleCap, b.LightDazzleAbove)
	}
	b.DazzleCap = 0.6
	b.LightDazzleAbove = 80
	b.Validate()
	if b.DazzleCap != 0.6 || b.LightDazzleAbove != 80 {
		t.Errorf("legal values were changed: dazzle %v edge %v", b.DazzleCap, b.LightDazzleAbove)
	}
	cfg := GetConfig()
	cfg.Balance = b
	SetConfigForTest(t, cfg)
	if got := GetLightingConfig().DazzleAbove; got != 80 {
		t.Errorf("accessor DazzleAbove = %d, want 80", got)
	}
}
