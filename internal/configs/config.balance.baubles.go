package configs

import "strings"

// Bauble defaults (docs/baubles/implementation-plan.md). These MUST match the
// shipped block in _datafiles/config.yaml; TestBaubleShippedConfigMatchesDefaults
// pins the two together.
const (
	defaultBaubleSearchChancePct = 1.0 // a biome not in the table
	defaultBaubleSkillMaxBonus   = 1.0 // search skill at SkillSoftCap doubles the chance
	defaultBaubleRollsPerWindow  = 2
	defaultBaubleFeatureMinutes  = 60
	defaultBaubleUntakenHours    = 24
	defaultBaubleWindowMinutes   = 60
	defaultBaubleRevealSeconds   = 3
	maxBaubleRevealSeconds       = 30

	defaultBaublePickpocketChancePct = 50
	defaultBaublePickpocketMaxWeight = 1.0
	maxBaublePickpocketMaxWeight     = 5.0
	defaultBaublePickpocketGraceSecs = 5.0
	maxBaublePickpocketGraceSecs     = 25.0

	defaultStealPocketSeconds    = 3.0
	defaultStealPocketMinSeconds = 1.5
	defaultStealPocketMaxSeconds = 6.0
	maxStealPocketSeconds        = 20.0
	maxBaubleSkillMaxBonus       = 10.0

	defaultBaubleTierWeightCheap   = 70
	defaultBaubleTierWeightAverage = 25
	defaultBaubleTierWeightRare    = 5

	// Finds that must be stolen lean richer (owner ruling, 2026-09-28).
	defaultBaubleHouseholdTierWeightCheap    = 45
	defaultBaubleHouseholdTierWeightAverage  = 40
	defaultBaubleHouseholdTierWeightRare     = 15
	defaultBaublePickpocketTierWeightCheap   = 50
	defaultBaublePickpocketTierWeightAverage = 40
	defaultBaublePickpocketTierWeightRare    = 10

	// Stolen baubles (owner rulings, 2026-09-28).
	defaultBaubleStolenHeatHours = 72 // three real days
	defaultBaubleFenceBuyPct     = 60
	defaultBaubleReturnsPerCatch = 3

	defaultBaubleCheapMin   = 1
	defaultBaubleCheapMax   = 6
	defaultBaubleAverageMin = 10
	defaultBaubleAverageMax = 15
	defaultBaubleRareMin    = 40
	defaultBaubleRareMax    = 200
)

// defaultBaubleHeatAreas groups zones that make one city, so a bauble stolen
// in one of them is hot in all of them (and nowhere else). New Plymouth is
// nine zones; its outskirts are outside the city. Every other town is one
// zone, its own area already.
func defaultBaubleHeatAreas() map[string][]string {
	return map[string][]string{
		`New Plymouth`: {
			`New Plymouth Common`, `New Plymouth Crafting`, `New Plymouth Docks`,
			`New Plymouth Merchant`, `New Plymouth Noble`, `New Plymouth Old Quarter`,
			`New Plymouth Sewers`, `New Plymouth Temple`,
		},
	}
}

// defaultBaubleFenceGroups are the mob groups that make a merchant a fence.
var defaultBaubleFenceGroups = []string{`fence`}

// defaultBaubleBuyerCraftSupports mirrors shops.CraftSupportGeneral and
// shops.CraftSupportJewelcrafting (configs cannot import shops).
var defaultBaubleBuyerCraftSupports = []string{`general`, `jewelcrafting`}

// defaultBaubleBiomeChancePct is the chance, in percent per roll, of a
// search finding a bauble, by room biome (_datafiles/world/dogmud/biomes).
// Where people live and lose things, finds are common; out in the wild they
// are rare. Lower-case biome ids; a biome not listed uses
// BaubleSearchChancePct.
func defaultBaubleBiomeChancePct() map[string]float64 {
	return map[string]float64{
		// Built places people use.
		`interior`: 5.0, // houses, halls, shops, temples
		`fort`:     4.0, // fortified dwellings
		`ruins`:    3.5, // abandoned buildings, picked over but full of leavings
		`sewer`:    3.0, // what washes down the drains
		`dungeon`:  2.0, // built underground places, old leavings
		// Streets.
		`city_backstreet`:   2.5, // alleys and yards where things get dropped and kicked aside
		`city_thoroughfare`: 2.0, // main ways: busy, but swept and picked over
		// Travelled open country.
		`road`:     1.0,  // travellers drop things
		`shore`:    1.0,  // things wash up
		`farmland`: 0.75, // homesteads and fields
		`cave`:     0.75, // someone sheltered here once
		`land`:     0.5,  // generic open ground, including some road zones
		`river`:    0.5,
		// Wilderness.
		`forest`:       0.25,
		`dense_forest`: 0.25,
		`plains`:       0.25,
		`swamp`:        0.25,
		`cliffs`:       0.25,
		`mountains`:    0.25,
		`desert`:       0.25,
		`snow`:         0.25,
		`spiderweb`:    0.25,
		// Nowhere to search.
		`water`: 0, // deep water
		`ether`: 0, // not a place in the world at all
	}
}

// validateBaubles sets bauble defaults.
//
// There is no "0 disables" knob here: an absent key decodes as 0, so zero has
// to mean "use the default". BaublesEnabled is the on switch, and it is off
// unless the config says otherwise: an absent key is false.
//
// The value ladder is validated AS A WHOLE: every tier needs 1 <= min <= max,
// and the tiers must climb without touching (cheap max < average min, average
// max < rare min), because a price is what tells a player which tier they found.
// If any of the six numbers breaks that, all six fall back to the defaults
// rather than leaving a half-repaired ladder.
func (b *Balance) validateBaubles() {
	if !(b.BaubleSearchChancePct > 0) { // NaN from YAML `.nan` too
		b.BaubleSearchChancePct = defaultBaubleSearchChancePct
	}
	if b.BaubleSearchChancePct > 100 {
		b.BaubleSearchChancePct = 100
	}

	// The biome table: absent (or empty) takes the whole default table. A
	// table that is given is used as given, so an operator can list only the
	// biomes they care about and let the rest fall back to
	// BaubleSearchChancePct. Here, unlike the scalars, 0 is honoured: a
	// listed biome at 0 never finds anything. Keys are lower-cased;
	// negatives become 0 and anything over 100 becomes 100.
	if len(b.BaubleBiomeChancePct) == 0 {
		b.BaubleBiomeChancePct = defaultBaubleBiomeChancePct()
	} else {
		cleaned := make(map[string]float64, len(b.BaubleBiomeChancePct))
		for biome, pct := range b.BaubleBiomeChancePct {
			if !(pct >= 0) { // negative, or NaN
				pct = 0
			}
			if pct > 100 {
				pct = 100
			}
			cleaned[strings.ToLower(strings.TrimSpace(biome))] = pct
		}
		b.BaubleBiomeChancePct = cleaned
	}

	if !(b.BaubleSkillMaxBonus > 0) {
		b.BaubleSkillMaxBonus = defaultBaubleSkillMaxBonus
	}
	if b.BaubleSkillMaxBonus > maxBaubleSkillMaxBonus {
		b.BaubleSkillMaxBonus = maxBaubleSkillMaxBonus
	}
	if b.BaubleRollsPerWindow <= 0 {
		b.BaubleRollsPerWindow = defaultBaubleRollsPerWindow
	}
	if b.BaubleFeatureWindowMinutes <= 0 {
		b.BaubleFeatureWindowMinutes = defaultBaubleFeatureMinutes
	}
	if b.BaubleUntakenHours <= 0 {
		b.BaubleUntakenHours = defaultBaubleUntakenHours
	}
	if b.BaubleWindowMinutes <= 0 {
		b.BaubleWindowMinutes = defaultBaubleWindowMinutes
	}
	if b.BaubleRevealSeconds <= 0 {
		b.BaubleRevealSeconds = defaultBaubleRevealSeconds
	}
	if b.BaubleRevealSeconds > maxBaubleRevealSeconds {
		b.BaubleRevealSeconds = maxBaubleRevealSeconds
	}

	// Pickpocketing. The chance: absent (0) is the default, a negative value
	// is "never", over 100 is 100.
	// (NaN from YAML `.nan` is the default too.)
	switch {
	case b.BaublePickpocketChancePct != b.BaublePickpocketChancePct || b.BaublePickpocketChancePct == 0:
		b.BaublePickpocketChancePct = defaultBaublePickpocketChancePct
	case b.BaublePickpocketChancePct < 0:
		b.BaublePickpocketChancePct = -1
	case b.BaublePickpocketChancePct > 100:
		b.BaublePickpocketChancePct = 100
	}
	if !(b.BaublePickpocketMaxWeight > 0) {
		b.BaublePickpocketMaxWeight = defaultBaublePickpocketMaxWeight
	}
	if b.BaublePickpocketMaxWeight > maxBaublePickpocketMaxWeight {
		b.BaublePickpocketMaxWeight = maxBaublePickpocketMaxWeight
	}
	if !(b.BaublePickpocketGraceSecs > 0) {
		b.BaublePickpocketGraceSecs = defaultBaublePickpocketGraceSecs
	}
	if b.BaublePickpocketGraceSecs > maxBaublePickpocketGraceSecs {
		b.BaublePickpocketGraceSecs = maxBaublePickpocketGraceSecs
	}

	// Stolen baubles. Absent (0) or negative takes the default; the fence
	// never pays more than the whole value.
	if b.BaubleStolenHeatHours <= 0 {
		b.BaubleStolenHeatHours = defaultBaubleStolenHeatHours
	}
	// Heat areas: absent (or empty) takes the default grouping; a table
	// that is given is used as given.
	if len(b.BaubleHeatAreas) == 0 {
		b.BaubleHeatAreas = defaultBaubleHeatAreas()
	}
	if b.BaubleFenceBuyPct <= 0 {
		b.BaubleFenceBuyPct = defaultBaubleFenceBuyPct
	}
	if b.BaubleFenceBuyPct > 100 {
		b.BaubleFenceBuyPct = 100
	}
	if len(b.BaubleFenceGroups) == 0 {
		b.BaubleFenceGroups = append(ConfigSliceString(nil), defaultBaubleFenceGroups...)
	}
	// A return must be worth less than a catch (owner ruling), so at least
	// two returns make one catch.
	if b.BaubleReturnsPerCatch <= 0 {
		b.BaubleReturnsPerCatch = defaultBaubleReturnsPerCatch
	}
	if b.BaubleReturnsPerCatch < 2 {
		b.BaubleReturnsPerCatch = 2
	}

	validateTierWeights(&b.BaubleTierWeightCheap, &b.BaubleTierWeightAverage, &b.BaubleTierWeightRare,
		defaultBaubleTierWeightCheap, defaultBaubleTierWeightAverage, defaultBaubleTierWeightRare)
	validateTierWeights(&b.BaubleHouseholdTierWeightCheap, &b.BaubleHouseholdTierWeightAverage, &b.BaubleHouseholdTierWeightRare,
		defaultBaubleHouseholdTierWeightCheap, defaultBaubleHouseholdTierWeightAverage, defaultBaubleHouseholdTierWeightRare)
	validateTierWeights(&b.BaublePickpocketTierWeightCheap, &b.BaublePickpocketTierWeightAverage, &b.BaublePickpocketTierWeightRare,
		defaultBaublePickpocketTierWeightCheap, defaultBaublePickpocketTierWeightAverage, defaultBaublePickpocketTierWeightRare)

	ladderOk := b.BaubleCheapMinValue >= 1 &&
		b.BaubleCheapMinValue <= b.BaubleCheapMaxValue &&
		b.BaubleCheapMaxValue < b.BaubleAverageMinValue &&
		b.BaubleAverageMinValue <= b.BaubleAverageMaxValue &&
		b.BaubleAverageMaxValue < b.BaubleRareMinValue &&
		b.BaubleRareMinValue <= b.BaubleRareMaxValue
	if !ladderOk {
		b.BaubleCheapMinValue = defaultBaubleCheapMin
		b.BaubleCheapMaxValue = defaultBaubleCheapMax
		b.BaubleAverageMinValue = defaultBaubleAverageMin
		b.BaubleAverageMaxValue = defaultBaubleAverageMax
		b.BaubleRareMinValue = defaultBaubleRareMin
		b.BaubleRareMaxValue = defaultBaubleRareMax
	}

	if len(b.BaubleBuyerCraftSupports) == 0 {
		b.BaubleBuyerCraftSupports = append(ConfigSliceString(nil), defaultBaubleBuyerCraftSupports...)
	}
}

// validateTierWeights checks one set of cheap, average and rare weights: a
// negative weight is invalid and takes its default; weights that are all
// zero (absent) take all three defaults. A single explicit zero is honoured,
// so a tier can be switched off.
func validateTierWeights(cheap, average, rare *ConfigInt, defCheap, defAverage, defRare ConfigInt) {
	if *cheap < 0 {
		*cheap = defCheap
	}
	if *average < 0 {
		*average = defAverage
	}
	if *rare < 0 {
		*rare = defRare
	}
	if *cheap+*average+*rare == 0 {
		*cheap, *average, *rare = defCheap, defAverage, defRare
	}
}
