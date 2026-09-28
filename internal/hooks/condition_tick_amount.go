package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
)

// tickPoolMax is the holder's maximum for a tick_pool name ("health",
// "stamina", "conviction"); an unknown pool is 0.
func tickPoolMax(c *characters.Character, pool string) int {
	switch pool {
	case "health":
		return c.HealthMax.Value
	case "stamina":
		return c.StaminaMax.Value
	case "conviction":
		return c.ConvictionMax.Value
	}
	return 0
}

// setTickAmountAtApply computes a tick_pool condition's per-round amount
// from the holder's pool and the applier's scale, where the condition
// lands. It replaces the post-queue SetTickAmount calls that found no
// record on a first application.
func setTickAmountAtApply(c *characters.Character, spec *conditions.ConditionSpec, conditionId int, scale float64) {
	if spec == nil || spec.TickPool == "" {
		return
	}
	if scale <= 0 {
		scale = 1.0
	}
	amt := conditions.ComputeTickAmount(tickPoolMax(c, spec.TickPool), spec.TickPercent, spec.TickVariance, spec.TickMin, scale)
	c.Conditions.SetTickAmount(conditionId, amt)
}

// fillZeroTickAmount computes and caches the per-trigger amount for a
// tick_pool condition whose TickAmount is still 0. Shared by the player and
// mob round ticks. A condition queued through the event lands with its
// amount already computed (setTickAmountAtApply), so this is now the
// fallback for synchronous character-level adds only, such as
// Character.AddConditionMagnitude for a former combat condition, which never
// pass through Condition_ApplyConditions.
// Scaling is 1.0: a synchronous add carries no applier scale.
func fillZeroTickAmount(c *characters.Character, cond *conditions.Condition, spec *conditions.ConditionSpec) int {
	if spec == nil || spec.TickPool == "" || cond.TickAmount != 0 {
		return cond.TickAmount
	}
	amt := conditions.ComputeTickAmount(tickPoolMax(c, spec.TickPool), spec.TickPercent, spec.TickVariance, spec.TickMin, 1.0)
	c.Conditions.SetTickAmount(cond.ConditionId, amt)
	return amt
}
