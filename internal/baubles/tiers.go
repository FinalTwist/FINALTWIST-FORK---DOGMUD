// Package baubles holds the engine-side rules for bauble loot: small,
// non-usable, non-wearable objects that exist only to be sold. See
// docs/baubles/implementation-plan.md.
//
// This file is the value ladder. A bauble is generated INTO a tier that the
// game chooses first (PickTier, from the configured weights); the model then
// picks a value inside that tier's range, and code clamps whatever it answers
// back into the range. The ranges are deterministic and do not touch: with
// the defaults nothing is ever worth 7 to 9 or 16 to 39 gold, so a price
// tells a player which tier they found.
//
// The ladder lives in config.yaml (Balance.Bauble*Value); configs validates
// it as a whole and falls back to the defaults below if it is broken.
package baubles

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// ValueTier is one rung of the bauble value ladder.
type ValueTier string

const (
	TierCheap   ValueTier = `cheap`
	TierAverage ValueTier = `average`
	TierRare    ValueTier = `rare`
)

// ValueRange is an inclusive gold range.
type ValueRange struct {
	Min int
	Max int
}

// defaultTierRanges is the owner's ladder and the configs defaults. It is
// what TestTierRangesAreTheSpec pins. The live ladder is tierRanges().
var defaultTierRanges = map[ValueTier]ValueRange{
	TierCheap:   {Min: 1, Max: 6},
	TierAverage: {Min: 10, Max: 15},
	TierRare:    {Min: 40, Max: 200},
}

// tierRanges is the configured ladder (already validated by configs).
func tierRanges() map[ValueTier]ValueRange {
	b := configs.GetBalanceConfig()
	return map[ValueTier]ValueRange{
		TierCheap:   {Min: int(b.BaubleCheapMinValue), Max: int(b.BaubleCheapMaxValue)},
		TierAverage: {Min: int(b.BaubleAverageMinValue), Max: int(b.BaubleAverageMaxValue)},
		TierRare:    {Min: int(b.BaubleRareMinValue), Max: int(b.BaubleRareMaxValue)},
	}
}

// Tiers lists every tier, cheapest first.
func Tiers() []ValueTier {
	return []ValueTier{TierCheap, TierAverage, TierRare}
}

// ParseTier reads a tier name as stored in the catalog or config.
func ParseTier(s string) (ValueTier, bool) {
	t := ValueTier(s)
	return t, t.Valid()
}

// Valid reports whether t is one of the three tiers.
func (t ValueTier) Valid() bool {
	_, ok := defaultTierRanges[t]
	return ok
}

// Range is the tier's gold range. An unknown tier gets the cheap range: a
// corrupt or missing tier must never make an object worth more.
func (t ValueTier) Range() ValueRange {
	ranges := tierRanges()
	if r, ok := ranges[t]; ok {
		return r
	}
	return ranges[TierCheap]
}

// Contains reports whether v lies inside the range.
func (r ValueRange) Contains(v int) bool {
	return v >= r.Min && v <= r.Max
}

// Clamp moves v into the range.
func (r ValueRange) Clamp(v int) int {
	if v < r.Min {
		return r.Min
	}
	if v > r.Max {
		return r.Max
	}
	return v
}

// ClampValue moves a proposed value into the tier's range.
func (t ValueTier) ClampValue(v int) int {
	return t.Range().Clamp(v)
}

// RollValue draws a value uniformly from the tier's range, for baubles named
// without the model (the local fallback). randn(n) must return an int in
// [0, n), for example util.Rand. A nil randn gives the range's midpoint, so
// callers and tests can be fully deterministic.
func (t ValueTier) RollValue(randn func(n int) int) int {
	r := t.Range()
	if randn == nil {
		return (r.Min + r.Max) / 2
	}
	return r.Clamp(r.Min + randn(r.Max-r.Min+1))
}

// PromptLine is the sentence the generation prompt uses to tell the model
// which range its value must fall in.
func (t ValueTier) PromptLine() string {
	r := t.Range()
	return fmt.Sprintf(`This is a %s find. Its value must be a whole number of gold from %d to %d inclusive; finer or more unusual objects sit higher in that range.`, t.label(), r.Min, r.Max)
}

func (t ValueTier) label() string {
	if t.Valid() {
		return string(t)
	}
	return string(TierCheap)
}
