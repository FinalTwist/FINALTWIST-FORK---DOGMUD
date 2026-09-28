package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
)

// fillZeroTickAmount computes and caches the per-trigger amount for a
// tick_pool condition whose TickAmount is still 0. A condition applied
// through the event queue is not in the list when its applier would snapshot
// the amount, so it arrives at 0 (area and mutator conditions, hazard-room
// DoTs, and every potion). Shared by the player and mob round ticks (drink
// path unification); before it, only the player tick filled the amount and
// every mob heal-over-time and damage-over-time was inert.
// Scaling is 1.0: the applier's caster scaling is already lost by then (the
// "ticks" parity slice moves the computation to apply time).
func fillZeroTickAmount(c *characters.Character, cond *conditions.Condition, spec *conditions.ConditionSpec) int {
	if spec == nil || spec.TickPool == "" || cond.TickAmount != 0 {
		return cond.TickAmount
	}
	var maxPool int
	switch spec.TickPool {
	case "health":
		maxPool = c.HealthMax.Value
	case "stamina":
		maxPool = c.StaminaMax.Value
	case "conviction":
		maxPool = c.ConvictionMax.Value
	}
	amt := conditions.ComputeTickAmount(maxPool, spec.TickPercent, spec.TickVariance, spec.TickMin, 1.0)
	c.Conditions.SetTickAmount(cond.ConditionId, amt)
	return amt
}
