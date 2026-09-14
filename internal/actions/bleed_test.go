package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestBleedPerRound(t *testing.T) {
	cases := []struct {
		strength       int
		divisor, floor configs.ConfigInt
		want           int
	}{
		{100, 50, 1, 2},
		{149, 50, 1, 2}, // integer division, not rounding
		{150, 50, 1, 3},
		{30, 50, 1, 1}, // below the divisor: the floor
		{100, 33, 1, 3},
		{100, 50, 4, 4}, // a floor above the quotient wins
		// The shipped knobs at Strength 100 (config_bleed_stacks_test.go's
		// TestShippedBleedTuningMeetsTheSliceTargets mirrors this formula
		// because configs cannot import actions): rake, drain, hamstring 2;
		// maul 2; throttle 3.
		{100, 35, 1, 2},
	}
	for _, c := range cases {
		if got := bleedPerRound(c.strength, c.divisor, c.floor); got != c.want {
			t.Errorf("bleedPerRound(%d, %d, %d) = %d, want %d", c.strength, c.divisor, c.floor, got, c.want)
		}
	}
}
