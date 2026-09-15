package usercommands

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Owner ruling 2026-09-13: a skill-too-low refusal must not expose real skill
// numbers. `craft blackrazor` used to say "(requires 65, you have 54)".
func TestCraftSkillTooLowText_NoNumbersAndBandsByGap(t *testing.T) {
	digits := regexp.MustCompile(`[0-9]`)

	cases := []struct {
		name     string
		minimum  int
		level    int
		wantBand string
	}{
		{"owner example is close", 65, 54, "just beyond"},
		{"exactly at the close edge", 50, 40, "just beyond"},
		{"one short of the close edge", 50, 39, "well beyond"},
		{"untrained", 65, 0, "well beyond"},
		{"low tier one short", 5, 4, "just beyond"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := craftSkillTooLowText("blacksmithing", tc.minimum, tc.level)
			assert.False(t, digits.MatchString(got), "refusal leaks a number: %q", got)
			assert.Contains(t, got, tc.wantBand)
			assert.Contains(t, got, "blacksmithing")
		})
	}
}

// Both refusal sites in craft.go (the shared InitiateCraft result and the
// enchanting sub-path) must route through the helper, so the raw format cannot
// come back at one of them.
func TestCraft_SkillRefusalsUseTheHelper(t *testing.T) {
	src := craftSource(t)
	assert.False(t, strings.Contains(src, "you have %d"), "craft.go: a refusal prints the player's raw skill again")
	assert.False(t, strings.Contains(src, "requires %d"), "craft.go: a refusal prints the recipe's raw minimum again")
	calls := strings.Count(src, "craftSkillTooLowText(") - strings.Count(src, "func craftSkillTooLowText(")
	require.Equal(t, 2, calls,
		"expected the helper at both refusal sites (InitiateCraft result and craftEnchanting)")
}
