package baubles

// The generic trinket: what every bauble is when it is not named by the
// model. That is always the case with no OpenAI API key, and it is the
// fallback whenever the model is switched off, over budget, too slow, fails,
// refuses, or answers with something that does not pass validation.
//
// It is deliberately plain (owner ruling, Phase 4): one name, a short
// description, and a value and weight drawn at random, the value inside the
// tier the search already chose. Players see "Trinket", sellers get the
// tier's price, and nothing about it depends on where it was found.

const (
	genericName       = `Trinket`
	genericNameSimple = `trinket`

	// Generic weights are drawn in tenths of a pound from this range.
	genericMinWeightTenths = 1 // 0.1 lb
	genericMaxWeightTenths = 8 // 0.8 lb
)

// genericDescriptions are the simple descriptions a generic trinket may get.
var genericDescriptions = []string{
	`A small trinket of no particular make. Someone might pay a little for it.`,
	`A little trinket, worn smooth by many hands. It might be worth a coin or two.`,
	`An ordinary trinket, the kind of thing people lose and other people find.`,
	`A small, scuffed trinket. A merchant might give something for it.`,
}

// GenericTrinket is the fallback bauble, in the same shape the model gives.
// randn(n) must return an int in [0, n) (util.Rand); a nil randn gives the
// first description, the tier's midpoint value and the lightest weight, which
// tests rely on.
func GenericTrinket(tier ValueTier, randn func(n int) int) Reply {
	pick := func(n int) int {
		if randn == nil || n <= 1 {
			return 0
		}
		v := randn(n)
		if v < 0 || v >= n {
			return 0
		}
		return v
	}
	tenths := genericMinWeightTenths + pick(genericMaxWeightTenths-genericMinWeightTenths+1)
	return Reply{
		Name:        genericName,
		NameSimple:  genericNameSimple,
		Description: genericDescriptions[pick(len(genericDescriptions))],
		WeightLbs:   float64(tenths) / 10,
		Value:       tier.RollValue(randn),
	}
}
