package configs

// validateLighting sets defaults for the graded room lighting thresholds
// introduced by the graded lighting arc. It is a separate file from
// config.balance.combat.go's DARKNESS section because those knobs price the
// COMBAT penalty for fighting blind or by shapes, while these knobs define
// the light SCALE itself that Task 3's Room.LightLevel() and Task 4's
// ParticipantSight will read. Plans 2 and 3 add more knobs here (a dazzle
// threshold, ambient light by time of day, NightVision strength), so this
// area gets its own file up front rather than being folded into
// validateMisc and split out later.
func (b *Balance) validateLighting() {
	// LightBlindBelow and LightDimBelow are validated as a PAIR, following
	// the DarknessShapesCombatPenalty precedent in validateCombat: an
	// inverted or out-of-range pair reverts BOTH rather than leaving one
	// knob correct and the other wrong. See the struct field comment for why
	// zero is coerced rather than honoured for these two.
	blindInRange := b.LightBlindBelow >= -100 && b.LightBlindBelow <= 100 && b.LightBlindBelow != 0
	dimInRange := b.LightDimBelow >= -100 && b.LightDimBelow <= 100 && b.LightDimBelow != 0
	if !blindInRange || !dimInRange || b.LightBlindBelow >= b.LightDimBelow {
		b.LightBlindBelow = 25
		b.LightDimBelow = 50
	}

	// LightExitsAbove is checked after the pair above so it sees the final,
	// valid LightBlindBelow rather than a value that is about to be
	// reverted. It gets its own range clamp plus the one cross-axis rule
	// that keeps it coherent: it must not sit below LightBlindBelow, or a
	// blind observer would see through an exit. It is deliberately NOT
	// required to sit above LightDimBelow; see the struct field comment.
	exitsInRange := b.LightExitsAbove >= -100 && b.LightExitsAbove <= 100 && b.LightExitsAbove != 0
	if !exitsInRange || b.LightExitsAbove < b.LightBlindBelow {
		b.LightExitsAbove = 65
	}
}
