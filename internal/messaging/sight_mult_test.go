package messaging

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
)

type fixedLight int

func (f fixedLight) LightLevel() int { return int(f) }

func TestSightScoreMultiplierWorkedValues(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.Validate() // dark 0.80, dazzle 0.80
	bal := cfg.Balance
	cases := []struct {
		dark, bright, want float64
	}{
		{0, 0, 1.0},
		{0.52, 0, 0.896}, // normal eyes at 37
		{0.96, 0, 0.808}, // normal eyes at 26
		{1, 0, 0.80},
		{0, 0.6, 0.88},   // normal eyes at 90
		{0, 0.76, 0.848}, // nightvision 24 at 70
		{0, 0.88, 0.824}, // Cat's Eye at 73
		{0, 1, 0.80},
	}
	for _, c := range cases {
		if got := SightScoreMultiplier(c.dark, c.bright, bal); !near(got, c.want) {
			t.Errorf("(%v, %v) = %v, want %v", c.dark, c.bright, got, c.want)
		}
	}
	bal.DazzleCap = 0.5
	if got := SightScoreMultiplier(0, 1, bal); got != 0.5 {
		t.Errorf("DazzleCap not read: %v", got)
	}
}

func TestSightMultFollowsTheRamp(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.Validate()
	configs.SetConfigForTest(t, cfg)
	c := characters.New()
	for _, tc := range []struct {
		light int
		want  float64
	}{{60, 1.0}, {37, 0.896}, {90, 0.88}, {0, 0.80}} {
		if got := SightMult(c, fixedLight(tc.light)); !near(got, tc.want) {
			t.Errorf("light %d: %v, want %v", tc.light, got, tc.want)
		}
	}
	if got := SightMult(c, nil); got != 1.0 {
		t.Error("nil room must be unity")
	}
	if got := SightMult(nil, fixedLight(0)); got != 1.0 {
		t.Error("nil observer must be unity")
	}
}
