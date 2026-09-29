package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/costs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
)

// FleeRefusal says why a flee did not begin.
type FleeRefusal int

const (
	FleeOK FleeRefusal = iota
	FleeRefuseRooted
	FleeRefuseNoFlee
	FleeRefuseAlready
	FleeRefuseNotInCombat
	FleeRefuseGrappled
	FleeRefuseProne
	FleeRefuseNotReady
)

// FleeBegin reports one flee command. Accepted means the fleer is now
// Disengaging and the round will resolve the escape; Short means it could
// not pay in full and brings no Skullduggery to the blocker contest.
type FleeBegin struct {
	Accepted bool
	Short    bool
	Refusal  FleeRefusal
}

// FleeGate is every refusal a flee command can know before it transitions,
// in the order the player has always been told them. It is exported so a
// behaviour-tree action can decide to fire instead of kiting when a flee
// could not begin. Grapple and standing duplicate CombatPhase's position veto
// on purpose: the veto is registered by the hooks package, so without these
// two checks a flee's outcome would depend on which packages are linked.
func FleeGate(c *characters.Character) FleeRefusal {
	switch {
	case c.HasConditionFlag(conditions.NoMovement):
		return FleeRefuseRooted
	case c.HasConditionFlag(conditions.NoFlee):
		return FleeRefuseNoFlee
	case c.IsDisengaging():
		return FleeRefuseAlready
	case !c.IsInCombat():
		return FleeRefuseNotInCombat
	case c.IsStandingGrapple() || c.IsGroundGrapple():
		return FleeRefuseGrappled
	case !c.IsStanding():
		return FleeRefuseProne
	}
	return FleeOK
}

// BeginFlee is the flee command, shared by players and mobs: the gates, the
// pending admission, the Disengaging transition, then one quote and partial
// commit of the flee cost. Shortage never refuses a flee (it is the only way
// out of a fight); it drops Skullduggery from the contest instead. The escape
// itself happens on the next round, in ResolveFlee. preferredExit is carried
// to that round; "" means a random passable exit.
func BeginFlee(actor Actor, preferredExit string) FleeBegin {
	c := actor.GetCharacter()

	// Any command that is not the attempt already in flight owns no pending
	// admission, so retract an orphan before any refusal returns.
	if !c.IsDisengaging() {
		c.CancelFleeAdmission()
	}
	if r := FleeGate(c); r != FleeOK {
		return FleeBegin{Refusal: r}
	}

	// Publish a pending handoff before the transition. Cost belongs only to an
	// accepted Disengaging transition.
	c.PublishFleeAdmission(characters.FleeAdmission{PreferredExit: preferredExit})
	if c.CombatPhase == nil {
		c.CancelFleeAdmission()
		return FleeBegin{Refusal: FleeRefuseNotReady}
	}
	if err := c.CombatPhase.TransitionToDisengaging(state.TransitionReason{
		Trigger: combatphase.TriggerFleeCommand,
		Actor:   state.ActorRef{UserId: actor.GetUserId(), MobInstanceId: actor.GetMobInstanceId()},
	}); err != nil {
		c.CancelFleeAdmission()
		return FleeBegin{Refusal: fleeVetoReason(c)}
	}

	bal := configs.GetBalanceConfig()
	modifier := 1.0
	if mutations.IsFlying(c.Mutations) {
		modifier = float64(bal.FlightFleeStaminaMult)
	}
	quote := c.QuoteActionCost(characters.ActionCostRequest{
		Action:   costs.ActionFlee,
		Pool:     characters.PoolStamina,
		Base:     float64(bal.FleeStaminaCost),
		Modifier: modifier,
		Units:    1,
	})
	short := c.CommitCost(quote, characters.CostPartial).Short()
	c.PublishFleeAdmission(characters.FleeAdmission{
		IncludeSkill:  !short,
		Ready:         true,
		PreferredExit: preferredExit,
	})
	return FleeBegin{Accepted: true, Short: short}
}

// fleeVetoReason names a transition the machine refused after FleeGate let it
// through (a veto registered elsewhere, or combat ending in between).
func fleeVetoReason(c *characters.Character) FleeRefusal {
	switch {
	case !c.IsInCombat():
		return FleeRefuseNotInCombat
	case c.IsStandingGrapple() || c.IsGroundGrapple():
		return FleeRefuseGrappled
	case !c.IsStanding():
		return FleeRefuseProne
	}
	return FleeRefuseNotReady
}

// FleeOutcome reports one flee resolution (Task 3 fills it in).
type FleeOutcome struct{}

// ResolveFlee resolves a flee on its round (Task 3).
func ResolveFlee(actor Actor, room *rooms.Room) FleeOutcome { return FleeOutcome{} }
