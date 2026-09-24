package characters

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// bareFlagDefaults and bareFlagNoDefault name the two bestVisionNumber call
// sites below, so neither reads as a bare true/false at the call site. Only
// nightvision defaults on a bare flag; see bestVisionNumber's doc comment.
const (
	bareFlagDefaults  = true
	bareFlagNoDefault = false
)

// NightVisionStrength reports how far DOWN the light scale this character's
// usable band shifts, in scale points.
//
// Zero means no night sight at all. A character holding a vision flag that
// declares no strength of its own falls back to the configured default, so a
// bare flag still means something; see LightDefaultVisionStrength.
//
// The strongest source wins across conditions and mutations alike. It is
// never summed: the window MOVES rather than widening, so two abilities
// cannot combine into a window wider than the better one grants.
func (c *Character) NightVisionStrength() int {
	return c.bestVisionNumber(conditions.EffectNightVisionStrength, conditions.NightVision, bareFlagDefaults)
}

// InfraReach reports how far BELOW the window floor this character still
// reads shapes by sensing heat. Zero means not at all.
//
// Independent of NightVisionStrength on purpose: a creature can sense heat
// deeply while being no better than anyone else at using faint light.
func (c *Character) InfraReach() int {
	return c.bestVisionNumber(conditions.EffectInfraReach, conditions.InfraredVision, bareFlagNoDefault)
}

// bestVisionNumber is the shared body. effectKind is the numeric channel
// (conditions.Conditions.Effect, which already aggregates every held,
// unexpired condition by MAX for these two kinds; see effects.go's isMax).
// flag is the boolean channel, checked both on conditions and on mutations
// via mutations.FlagValue, which independently scales each candidate mutation
// by its own rank and also returns the largest scaled value, never a sum.
//
// Both channels are compared here in float64, and the winner is rounded to
// the nearest int exactly once, at the return. Rounding, like truncation, is
// a monotonic non-decreasing function of its input, so for a monotonic
// function f, max(f(a), f(b)) == f(max(a, b)) always: casting each source to
// int before comparing, versus comparing the two float64 sources and casting
// only the winner, can NEVER pick a different winner. What comparing first
// buys is not a different WINNER, it is one rounding instead of two, and it
// matches the codebase's existing convention of a single math.Round at the
// point where a float becomes an int (see internal/characters/companions.go
// and internal/characters/cast_helpers.go), rather than int()'s silent
// truncation.
//
// defaultOnBareFlag says whether a flag carrying no number falls back to the
// configured default. Only nightvision does: "sees in the dark" plainly
// means at least a little, while a bare infrared flag ("senses heat" with no
// stated range) has no sensible fallback and reads reach 0.
func (c *Character) bestVisionNumber(effectKind conditions.EffectKind, flag conditions.Flag, defaultOnBareFlag bool) int {
	best := c.Conditions.Effect(effectKind)

	if m := mutations.FlagValue(c.Mutations, string(flag)); m > best {
		best = m
	}

	if best == 0 && defaultOnBareFlag && c.HasFlagFromAnySource(flag) {
		best = float64(configs.GetBalanceConfig().LightDefaultVisionStrength)
	}

	return int(math.Round(best))
}
