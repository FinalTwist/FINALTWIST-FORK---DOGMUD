package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Movement parity 4b. The price of one room step, and the hidden-detection
// contests on arrival, are shared by players and mobs here. The command
// wrappers (usercommands.Go, mobcommands.Go) keep their own gates, lock
// handling and narration and call these once, right before relocating.

// MoveRefusal says why a step was not paid for.
type MoveRefusal int

const (
	MoveOK MoveRefusal = iota
	MoveRefuseEncumbered
	MoveRefuseTired
	MoveRefuseExhausted
)

// MoveCharge reports one priced step.
type MoveCharge struct {
	Refusal     MoveRefusal
	ActionCost  int     // 10, or 50 over carry capacity
	StaminaCost float64 // fractional stamina price, banked through the carry
	Winded      bool    // paid, and stamina is now under a quarter of its reachable max
	Never       bool    // refused, and this actor could not pay the step even fully rested
}

// OK reports whether the step was (or, for a quote, would be) paid.
func (m MoveCharge) OK() bool { return m.Refusal == MoveOK }

// The hardcoded step prices (fact V2). Not knobs; the spec keeps them as they
// are.
const (
	moveActionCost           = 10
	moveEncumberedActionCost = 50
)

// MovePrice is the price of one step into dest for c: action points, and the
// fractional stamina cost. It reads the destination biome by NAME
// (rooms.GetBiome(dest.Biome)), exactly as the player path always has.
func MovePrice(c *characters.Character, dest *rooms.Room) (int, float64) {
	actionCost := moveActionCost
	if c.GetCarriedWeight() > c.CarryCapacity() {
		actionCost = moveEncumberedActionCost
	}
	terrain := 1.0
	if biome, _ := rooms.GetBiome(dest.Biome); biome != nil {
		terrain = biome.GetMovementCost()
	}
	stamina := c.GetMovementStaminaCost(terrain)
	if mutations.IsFlying(c.Mutations) {
		// Winged Flight glides over terrain: movement barely tires you.
		stamina *= float64(configs.GetBalanceConfig().FlightMoveStaminaMult)
	}
	return actionCost, stamina
}

// settleMoverActionPoints brings a mob's points up to date. Players are
// credited per turn by hooks.ActionPoints and must never be settled here.
func settleMoverActionPoints(actor Actor) {
	if actor.IsPlayer() {
		return
	}
	actor.GetCharacter().SettleActionPoints(util.GetTurnCount())
}

func apRefusal(actionCost int) MoveRefusal {
	if actionCost == moveEncumberedActionCost {
		return MoveRefuseEncumbered
	}
	return MoveRefuseTired
}

// QuoteMove prices a step without paying for it, for callers that must decide
// before issuing one (the path walker, wander, behaviour-tree steps, the AI
// companion). Settling a mob's points is bookkeeping, not a charge.
func QuoteMove(actor Actor, dest *rooms.Room) MoveCharge {
	c := actor.GetCharacter()
	settleMoverActionPoints(actor)
	ap, st := MovePrice(c, dest)
	q := MoveCharge{ActionCost: ap, StaminaCost: st}
	switch {
	case c.ActionPoints < ap:
		q.Refusal = apRefusal(ap)
		q.Never = ap > c.ActionPointsMax.Value
	case !c.CanAffordCostFloat(characters.PoolStamina, st):
		q.Refusal = MoveRefuseExhausted
		q.Never = st > float64(c.EffectivePoolMax(characters.PoolStamina))
	}
	return q
}

// ChargeMove pays for one step into dest: action points first, then stamina,
// refunding the points when stamina refuses. Wrappers call it AFTER their
// lock checks and any exit-message requeue, so a door that stays locked costs
// nothing and a requeued step is charged once (fact V4).
func ChargeMove(actor Actor, dest *rooms.Room) MoveCharge {
	c := actor.GetCharacter()
	settleMoverActionPoints(actor)
	ap, st := MovePrice(c, dest)
	m := MoveCharge{ActionCost: ap, StaminaCost: st}
	if !c.DeductActionPoints(ap) {
		m.Refusal = apRefusal(ap)
		m.Never = ap > c.ActionPointsMax.Value
		return m
	}
	if !c.ApplyCostFloatOrRefuse(characters.PoolStamina, st) {
		c.ActionPoints += ap
		m.Refusal = MoveRefuseExhausted
		m.Never = st > float64(c.EffectivePoolMax(characters.PoolStamina))
		return m
	}
	// EffectivePoolMax, not the raw max: current stamina is reserve-clamped,
	// so a raw denominator nags a reserved character at a full pool.
	m.Winded = c.Stamina < c.EffectivePoolMax(characters.PoolStamina)/4
	return m
}

// QuoteMobStep quotes the step a mob would take through exitName from the room
// it stands in. A step that does not resolve (no room, no such exit, no
// destination, or `home`) quotes as affordable, so the caller's existing
// handling of a bad step is unchanged.
func QuoteMobStep(mob *mobs.Mob, exitName string) MoveCharge {
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return MoveCharge{}
	}
	ex := FindExit(room, exitName)
	if !ex.Found {
		return MoveCharge{}
	}
	dest := rooms.LoadRoom(ex.RoomId)
	if dest == nil {
		return MoveCharge{}
	}
	return QuoteMove(NewMobActorInRoom(mob, room), dest)
}

// movementTrainsSearch reports whether this move should record a search use.
// Moved unchanged from usercommands/go.go by movement parity 4b; the long
// rationale lives in internal/actions/context.md ("Movement").
//
// A zero or negative MovementSearchTrainChance switches the feature off.
func movementTrainsSearch() bool {
	chance := float64(configs.GetBalanceConfig().MovementSearchTrainChance)
	if chance <= 0 {
		return false
	}
	// Resolving against 100,000 keeps a knob as small as 0.00001 meaningful.
	const resolution = 100000
	return util.Rand(resolution) < int(chance*resolution)
}

// TrainSearchOnMove is the rare Search training a completed step earns. Walking
// is not a contest, so won is always true; the rarity gate is the rule.
func TrainSearchOnMove(actor Actor) {
	if movementTrainsSearch() {
		actor.AwardResolved(true, actor.GetCharacter().CandidateFor(string(skills.Search)))
	}
}
