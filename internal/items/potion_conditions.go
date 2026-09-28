package items

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/conditions"
)

// PotionMagnitudeApplication reports how a potion applies a condition that
// reads one of conditions.ScaledKinds from its magnitude (lighting plan 5c):
// the item's Magnitude and the condition's trigger count, both scaled by
// durationMult. The player drink path passes its potency multiplier (aging
// phase times the crafter's skill); the mob drink path passes 1. The result
// is capped by conditions.CapScaledMagnitude (infra reach at
// LightInfraReachCap, nightvision at the window shift cap). ok is false for a condition with no scaled
// kind or an item with no Magnitude, which keep the duration-only path.
//
// It lives here rather than beside either drink command so that both the
// player's (internal/usercommands) and a mob's (internal/mobcommands) drink
// apply a potion identically; a companion drinking the Pitsense Tincture
// through the mob path used to get reach 0.
func PotionMagnitudeApplication(itemSpec *ItemSpec, spec *conditions.ConditionSpec, durationMult float64) (magnitude float64, triggers int, ok bool) {
	if itemSpec == nil || spec == nil || itemSpec.Magnitude <= 0 {
		return 0, 0, false
	}
	kind, scaled := spec.ScaledKind()
	if !scaled {
		return 0, 0, false
	}
	if durationMult <= 0 {
		durationMult = 1
	}
	magnitude = conditions.CapScaledMagnitude(kind, itemSpec.Magnitude*durationMult)
	triggers = int(math.Round(float64(spec.TriggerCount) * durationMult))
	if triggers < 1 {
		triggers = 1
	}
	return magnitude, triggers, true
}

// PotionEffectConditionIds returns every condition id that only potions
// grant: named by a potion's ConditionIds and by no non-potion item's
// ConditionIds or WornConditionIds. The Purging Draught strips this set
// (lighting plan 5c), which replaced a hardcoded id block that shipped
// potions had already outgrown. It is computed on each call from the loaded
// specs, so an admin item reload cannot leave it stale.
func PotionEffectConditionIds() map[int]bool {
	potion := map[int]bool{}
	other := map[int]bool{}
	for _, spec := range GetAllItemSpecs() {
		if spec.Type == Potion {
			for _, id := range spec.ConditionIds {
				potion[id] = true
			}
			continue
		}
		for _, id := range spec.ConditionIds {
			other[id] = true
		}
		for _, id := range spec.WornConditionIds {
			other[id] = true
		}
	}
	for id := range other {
		delete(potion, id)
	}
	return potion
}
