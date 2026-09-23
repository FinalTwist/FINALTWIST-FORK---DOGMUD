package configs

import "testing"

func TestLightingDefaultsAreTheShippedCalibration(t *testing.T) {
	var b Balance
	b.Validate()
	if b.LightDoublingStep != 8 {
		t.Errorf("LightDoublingStep = %v, want 8", b.LightDoublingStep)
	}
	// WorldLatitude is deliberately NOT asserted here. Unlike every other
	// knob in this test, zero is a HONOURED value for WorldLatitude (see the
	// field comment and validateLighting): a bare Balance{} validates to
	// WorldLatitude == 0, not 46.5. TestWorldLatitudeRangeAndZero below
	// already covers this knob correctly, asserting zero survives and only
	// out-of-range values revert to 46.5. An assertion here that a bare
	// struct becomes 46.5 would contradict that test and the deliberate
	// zero-honouring design; this is a plan defect in the original test,
	// not a change to validation behaviour.
	if b.LightEquinoxNoon != 70 {
		t.Errorf("LightEquinoxNoon = %v, want 70", b.LightEquinoxNoon)
	}
	if b.LightMoonsFull != 35 {
		t.Errorf("LightMoonsFull = %v, want 35", b.LightMoonsFull)
	}
	if b.LightStarlight != 10 {
		t.Errorf("LightStarlight = %v, want 10", b.LightStarlight)
	}
	if b.LightMoonWeightSwiftmoon != 4 || b.LightMoonWeightWanderer != 1 || b.LightMoonWeightEye != 0.5 {
		t.Errorf("moon weights = %v/%v/%v, want 4/1/0.5",
			b.LightMoonWeightSwiftmoon, b.LightMoonWeightWanderer, b.LightMoonWeightEye)
	}
}

func TestDoublingStepRejectsNonPositive(t *testing.T) {
	for _, v := range []ConfigFloat{0, -3} {
		var b Balance
		b.LightDoublingStep = v
		b.Validate()
		if b.LightDoublingStep != 8 {
			t.Errorf("step %v survived validation as %v", v, b.LightDoublingStep)
		}
	}
}

// Latitude beyond the polar circles produces days with no sunrise or no sunset,
// which the model handles but which is almost never intended. Out of range
// reverts; zero is HONOURED and means "no latitude, fall back to NightHours".
func TestWorldLatitudeRangeAndZero(t *testing.T) {
	var b Balance
	b.WorldLatitude = 0
	b.Validate()
	if b.WorldLatitude != 0 {
		t.Errorf("zero latitude was coerced to %v; zero must be honoured", b.WorldLatitude)
	}

	for _, v := range []ConfigFloat{-91, 91} {
		var b2 Balance
		b2.WorldLatitude = v
		b2.Validate()
		if b2.WorldLatitude != 46.5 {
			t.Errorf("latitude %v survived as %v", v, b2.WorldLatitude)
		}
	}
}

// Starlight must sit below the all-moons-full value or the moon curve inverts
// and a full moon reads darker than a new one. Validated as a pair, the
// LightBlindBelow/LightDimBelow precedent.
func TestMoonRangeIsValidatedAsAPair(t *testing.T) {
	var b Balance
	b.LightStarlight = 40
	b.LightMoonsFull = 20
	b.Validate()
	if b.LightStarlight != 10 || b.LightMoonsFull != 35 {
		t.Errorf("inverted pair survived as %v/%v", b.LightStarlight, b.LightMoonsFull)
	}
}

func TestMoonWeightsRejectAllZero(t *testing.T) {
	var b Balance
	b.LightMoonWeightSwiftmoon = 0
	b.LightMoonWeightWanderer = 0
	b.LightMoonWeightEye = 0
	b.Validate()
	if b.LightMoonWeightSwiftmoon != 4 {
		t.Errorf("all-zero weights survived as %v", b.LightMoonWeightSwiftmoon)
	}
}

func TestGetLightingConfigMirrorsBalance(t *testing.T) {
	c := GetConfig()
	c.Balance.LightBlindBelow = 30
	c.Balance.LightDoublingStep = 11
	c.Balance.WorldLatitude = 12.5
	SetConfigForTest(t, c)

	got := GetLightingConfig()
	if got.BlindBelow != 30 {
		t.Errorf("BlindBelow = %d, want 30", got.BlindBelow)
	}
	if got.DoublingStep != 11 {
		t.Errorf("DoublingStep = %v, want 11", got.DoublingStep)
	}
	if got.WorldLatitude != 12.5 {
		t.Errorf("WorldLatitude = %v, want 12.5", got.WorldLatitude)
	}
}
