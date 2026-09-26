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

// TestSightKnobBoundaries pins the DazzleCap and LightDazzleAbove range
// edges directly, isolated from the fallback-clamping cases in
// config_lighting_thresholds_test.go.
func TestSightKnobBoundaries(t *testing.T) {
	t.Run("DazzleAbove equal to DimBelow reverts", func(t *testing.T) {
		var b Balance
		b.LightDazzleAbove = 50 // equals the default LightDimBelow
		b.Validate()
		if b.LightDazzleAbove != 75 {
			t.Errorf("LightDazzleAbove equal to LightDimBelow must revert to 75, got %v", b.LightDazzleAbove)
		}
	})
	t.Run("DazzleAbove upper bound accepted", func(t *testing.T) {
		var b Balance
		b.LightDazzleAbove = 100
		b.Validate()
		if b.LightDazzleAbove != 100 {
			t.Errorf("LightDazzleAbove of exactly 100 must survive validation, got %v", b.LightDazzleAbove)
		}
	})
	t.Run("DazzleAbove upper bound rejected", func(t *testing.T) {
		var b Balance
		b.LightDazzleAbove = 101
		b.Validate()
		if b.LightDazzleAbove != 75 {
			t.Errorf("LightDazzleAbove above 100 must revert to 75, got %v", b.LightDazzleAbove)
		}
	})
	t.Run("DazzleCap upper bound accepted", func(t *testing.T) {
		var b Balance
		b.DazzleCap = 1.0
		b.Validate()
		if b.DazzleCap != 1.0 {
			t.Errorf("DazzleCap of exactly 1.0 must survive validation, got %v", b.DazzleCap)
		}
	})
	t.Run("DazzleCap zero rejected", func(t *testing.T) {
		var b Balance
		b.DazzleCap = 0
		b.Validate()
		if b.DazzleCap != 0.80 {
			t.Errorf("zero DazzleCap must default to 0.80, got %v", b.DazzleCap)
		}
	})
}
