package usercommands

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// potionMagnitudeApplication reports how a potion applies a condition that
// reads one of conditions.ScaledKinds from its magnitude (lighting plan 5c):
// the item's Magnitude and the condition's trigger count, both scaled by the
// drink path's existing potency multiplier (aging phase times the crafter's
// skill). Infra reach is capped at LightInfraReachCap. ok is false for a
// condition with no scaled kind or an item with no Magnitude, which keep the
// duration-only path.
func potionMagnitudeApplication(itemSpec *items.ItemSpec, spec *conditions.ConditionSpec, durationMult float64) (magnitude float64, triggers int, ok bool) {
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
	magnitude = itemSpec.Magnitude * durationMult
	if kind == conditions.EffectInfraReach {
		if limit := float64(configs.GetLightingConfig().InfraReachCap); magnitude > limit {
			magnitude = limit
		}
	}
	triggers = int(math.Round(float64(spec.TriggerCount) * durationMult))
	if triggers < 1 {
		triggers = 1
	}
	return magnitude, triggers, true
}
