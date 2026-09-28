package messaging

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// Normal eyes: blind 25, dim 50, dazzle 75, width 25.
func TestComfortDistanceNormalEyes(t *testing.T) {
	cases := []struct {
		light        int
		dark, bright float64
	}{
		{60, 0, 0}, {50, 0, 0}, {74, 0, 0},
		{37, 0.52, 0}, {26, 0.96, 0}, {25, 1, 0}, {0, 1, 0},
		{75, 0, 0}, {90, 0, 0.6}, {100, 0, 1},
	}
	for _, c := range cases {
		d, b := comfortDistance(c.light, 0, 25, 50, 75)
		if !near(d, c.dark) || !near(b, c.bright) {
			t.Errorf("light %d: (%v, %v), want (%v, %v)", c.light, d, b, c.dark, c.bright)
		}
	}
}

// Nightvision 24: edges 1, 26, 51; the bright ramp is still 25 wide, so the
// cap arrives at 76, well inside daylight.
func TestComfortDistanceShiftedWindow(t *testing.T) {
	cases := []struct {
		light        int
		dark, bright float64
	}{
		{30, 0, 0}, {20, 0.24, 0}, {1, 1, 0},
		{70, 0, 0.76}, {73, 0, 0.88}, {76, 0, 1}, {90, 0, 1},
	}
	for _, c := range cases {
		d, b := comfortDistance(c.light, 24, 25, 50, 75)
		if !near(d, c.dark) || !near(b, c.bright) {
			t.Errorf("light %d: (%v, %v), want (%v, %v)", c.light, d, b, c.dark, c.bright)
		}
	}
}

func TestComfortDistanceStrengthIsClamped(t *testing.T) {
	d1, b1 := comfortDistance(70, 40, 25, 50, 75)
	d2, b2 := comfortDistance(70, 24, 25, 50, 75)
	if d1 != d2 || b1 != b2 {
		t.Error("a strength above the cap must clamp exactly as the window does")
	}
}

// A Blinded observer is fully dark, even in a room that would otherwise
// dazzle: ComfortDistance never lets bright light substitute for sight.
func TestComfortDistanceBlindedIsFullyDark(t *testing.T) {
	c := newChar(t)
	setBlinded(t, c)
	dark, bright := ComfortDistance(c, sightLight(100))
	if dark != 1 || bright != 0 {
		t.Errorf("Blinded observer in a blazing room = (%v, %v), want (1, 0)", dark, bright)
	}
}

// Shipped knobs: floor 0.90, cap 50, dark cap 0.80. The cap is the dark
// fraction that makes SightScoreMultiplier read exactly infraMult.
func TestInfraDarkCap(t *testing.T) {
	cases := []struct {
		name         string
		light, reach int
		want         float64
		ok           bool
	}{
		{"reach 1", 0, 1, 0.49, true},
		{"reach 25", 0, 25, 0.25, true},
		{"reach 30, shipped condition 85", 0, 30, 0.20, true},
		{"reach 50 costs nothing", 0, 50, 0, true},
		{"reach above the cap clamps", 0, 80, 0, true},
		{"a faint room is in reach", 12, 30, 0.20, true},
		{"at minus reach", -30, 30, 0.20, true},
		{"below minus reach", -31, 30, 0, false},
		{"no reach", 0, 0, 0, false},
	}
	for _, c := range cases {
		got, ok := infraDarkCap(c.light, c.reach, 0.90, 50, 0.80)
		if ok != c.ok || !near(got, c.want) {
			t.Errorf("%s: (%v, %v), want (%v, %v)", c.name, got, ok, c.want, c.ok)
		}
	}
	if _, ok := infraDarkCap(0, 30, 0.90, 50, 1.0); ok {
		t.Error("a dark cap of 1.0 means no dark penalty exists; infra must not apply")
	}
}

const comfortInfraConditionId = 7951

func withInfraReach(t *testing.T, reach float64) *characters.Character {
	t.Helper()
	configs.SetConfigForTest(t, configs.GetConfig())
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		comfortInfraConditionId: {ConditionId: comfortInfraConditionId, Name: "Test Heat Sight",
			Flags:   []conditions.Flag{conditions.InfraredVision},
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectInfraReach: {Literal: reach}}},
	}))
	c := newChar(t)
	if err := c.AddCondition(comfortInfraConditionId, true); err != nil {
		t.Fatalf("AddCondition: %v", err)
	}
	return c
}

func TestComfortDistanceInfraEasesTheDarkOnly(t *testing.T) {
	c := withInfraReach(t, 30)
	if dark, bright := ComfortDistance(c, sightLight(0)); !near(dark, 0.20) || bright != 0 {
		t.Errorf("pitch dark, reach 30 = (%v, %v), want (0.20, 0)", dark, bright)
	}
	if dark, _ := ComfortDistance(c, sightLight(-40)); dark != 1 {
		t.Errorf("below minus reach = %v, want the full dark 1", dark)
	}
	if dark, _ := ComfortDistance(c, sightLight(45)); !near(dark, 0.20) {
		t.Errorf("dim light where the natural ramp is 0.2 = %v, want 0.20 (the better of the two)", dark)
	}
	if dark, _ := ComfortDistance(c, sightLight(48)); !near(dark, 0.08) {
		t.Errorf("light 48, natural ramp 0.08 beats infra = %v, want 0.08", dark)
	}
	if _, bright := ComfortDistance(c, sightLight(100)); bright != 1 {
		t.Errorf("glare must cost infravision in full: bright = %v, want 1", bright)
	}
	if m := SightMult(c, sightLight(0)); !near(m, 0.96) {
		t.Errorf("SightMult in the dark at reach 30 = %v, want 0.96", m)
	}
}
