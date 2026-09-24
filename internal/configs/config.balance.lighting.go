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
	//
	// The fallback is NOT the bare literal 65: the whole-arc review found
	// that an unconditional 65 can itself land below an operator's
	// legitimately elevated LightBlindBelow (e.g. blind=90), reproducing the
	// exact contradiction this check exists to prevent. The fallback is
	// clamped up to LightBlindBelow whenever 65 would sit below it, so the
	// invariant "exits is never below blind" holds even after a revert, not
	// only for values that pass validation untouched.
	//
	// This clamps rather than reverting the whole group (the
	// DarknessShapesCombatPenalty style) on purpose: the blind/dim pair here
	// was independently valid, and discarding it over an unrelated exits
	// typo would surprise an operator debugging their config more than a
	// single knob quietly self-correcting to the nearest valid value. The
	// fallback is 65 in the overwhelmingly common case (any LightBlindBelow
	// at or below 65, which includes every shipped default), so this only
	// changes behaviour for the unusual configs that raise blind above 65.
	exitsInRange := b.LightExitsAbove >= -100 && b.LightExitsAbove <= 100 && b.LightExitsAbove != 0
	if !exitsInRange || b.LightExitsAbove < b.LightBlindBelow {
		exitsFallback := ConfigInt(65)
		if exitsFallback < b.LightBlindBelow {
			exitsFallback = b.LightBlindBelow
		}
		b.LightExitsAbove = exitsFallback
	}

	// LightDefaultVisionStrength is clamped to [0, 24] first, then zero
	// (whether authored directly or reached by clamping a negative) is
	// defaulted to 12. Doing it in that order means the accepted AUTHORED
	// range is effectively [1, 24], not [0, 24]: zero is not a value an
	// operator can choose, it is indistinguishable from the field being
	// unset, following the same idiom as ProgressMult (0 means "use the
	// default", not "shift by nothing"; see the struct field comment for
	// why a bare flag cannot mean a shift of zero).
	//
	// The upper bound 24 is windowShiftCap in internal/messaging/window.go,
	// duplicated here on purpose: internal/configs cannot import
	// internal/messaging (messaging depends on configs, not the reverse).
	// If windowShiftCap ever changes, this literal must change with it. A
	// value above the cap is clamped down to 24 rather than reverted to the
	// default, matching how LightExitsAbove clamps rather than reverts for
	// its own out-of-range case above: the operator's intent (a strong
	// shift) is still honoured, just capped at the strongest the window
	// model can express.
	if b.LightDefaultVisionStrength < 0 {
		b.LightDefaultVisionStrength = 0
	}
	if b.LightDefaultVisionStrength > 24 {
		b.LightDefaultVisionStrength = 24
	}
	if b.LightDefaultVisionStrength == 0 {
		b.LightDefaultVisionStrength = 12
	}
}
