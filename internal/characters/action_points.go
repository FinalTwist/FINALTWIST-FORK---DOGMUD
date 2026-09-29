package characters

// SettleActionPoints brings a MOB's action points up to date at turn, lazily.
//
// Players regain one point per turn in hooks.ActionPoints, which loops the
// online users every turn so the {ap} prompt token stays live. Mobs never
// regained anything and nothing set them at spawn, so every live mob held 0
// (movement parity 4b, fact V15). Looping every mob instance at 20 turns a
// second to fix that would cost far more than it buys, so a mob instead
// records the turn it was last settled and adds the elapsed turns, at the
// player rate, whenever something is about to read or spend its points.
//
// Never settled, or a stamp from the future (the turn counter restarts at 0
// on a reboot), means full.
//
// NEVER call this for a player: the per-turn hook already credits them, and a
// settle would credit the same turns twice. actions.ChargeMove and
// actions.QuoteMove settle only non-player actors.
func (c *Character) SettleActionPoints(turn uint64) {
	max := c.ActionPointsMax.Value
	if !c.ActionPointsSettled || turn < c.ActionPointsSettledTurn {
		c.ActionPoints = max
	} else if elapsed := turn - c.ActionPointsSettledTurn; elapsed > 0 {
		if elapsed > uint64(max) {
			elapsed = uint64(max)
		}
		c.ActionPoints += int(elapsed)
	}
	if c.ActionPoints > max {
		c.ActionPoints = max
	}
	c.ActionPointsSettled = true
	c.ActionPointsSettledTurn = turn
}
