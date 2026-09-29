package characters

// FleeAdmission is the flee command's handoff to the round that resolves the
// flee. The command publishes a pending one (Ready false) before it asks
// CombatPhase for Disengaging, and a ready one once the cost is settled.
// IncludeSkill is false when the fleer could not pay in full and so brings no
// Skullduggery to the blocker contest. PreferredExit, when set and still
// passable at resolution, is the exit the flee takes instead of a random one
// (a kiting archer falls back toward home).
type FleeAdmission struct {
	IncludeSkill  bool
	Ready         bool
	PreferredExit string
}

// fleeHandoff holds at most one admission. It is a value, not a pointer, so a
// mob template's shallow copy cannot share one between instances.
type fleeHandoff struct {
	set       bool
	admission FleeAdmission
}

// PublishFleeAdmission replaces any admission with a.
func (c *Character) PublishFleeAdmission(a FleeAdmission) {
	if c == nil {
		return
	}
	c.fleeHandoff = fleeHandoff{set: true, admission: a}
}

// TakeFleeAdmission consumes a READY admission. A pending one is left in
// place for the command to finish, and the second result is false, so a round
// that observes Disengaging inside the command's handoff window cannot
// consume an attempt whose cost is not decided yet.
func (c *Character) TakeFleeAdmission() (FleeAdmission, bool) {
	if c == nil || !c.fleeHandoff.set || !c.fleeHandoff.admission.Ready {
		return FleeAdmission{}, false
	}
	a := c.fleeHandoff.admission
	c.fleeHandoff = fleeHandoff{}
	return a, true
}

// CancelFleeAdmission retracts a pending or ready admission and reports
// whether there was one. CombatPhase terminal hooks use it when combat ends
// before the flee round can resolve.
func (c *Character) CancelFleeAdmission() bool {
	if c == nil || !c.fleeHandoff.set {
		return false
	}
	c.fleeHandoff = fleeHandoff{}
	return true
}
