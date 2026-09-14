package characters

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

func (c *Character) IsDisabled() bool {
	return c.Health <= 0
}

func (c *Character) HasConditionFlag(conditionFlag conditions.Flag) bool {
	return c.Conditions.HasFlag(conditionFlag, false)
}

// HasFlagFromAnySource returns true if the character has the given flag from
// either active conditions OR permanent mutation effects. Use this instead of
// HasConditionFlag when the check should also honor mutation-granted flags.
func (c *Character) HasFlagFromAnySource(conditionFlag conditions.Flag) bool {
	if c.Conditions.HasFlag(conditionFlag, false) {
		return true
	}
	return mutations.HasMutationFlag(c.Mutations, string(conditionFlag))
}

func (c *Character) CancelConditionsWithFlag(conditionFlag conditions.Flag) bool {
	if c.Conditions.HasFlag(conditionFlag, true) {
		c.Validate(true)
		// Hidden flag is special: the Awareness FSM mirrors the condition
		// via Awareness_Cascades.go. If a caller cancels the condition
		// directly (eat/drink/give/get/equip/spotted-on-entry/etc.)
		// without driving the FSM, IsHidden() stays true because the
		// FSM is still in Hidden — split source of truth. Drive the
		// FSM out here so every caller gets correct behavior without
		// having to know about the cascade. Re-entrancy: the cascade
		// re-fires CancelConditionsWithFlag(Hidden), but the condition is
		// already cancelled by the Validate above so HasFlag returns
		// false → no recursion.
		//
		// ⚠️ THE CONDITION IS ABOUT THE STATE, NOT THE ARGUMENT. It used
		// to read `conditionFlag == conditions.Hidden`, which asked "was Hidden the
		// flag you SELECTED by?" rather than "did stealth just end?".
		// Condition 9 Hidden carries BOTH `hidden` and `cancel-on-combat`, so
		// CancelCombatConditions -> CancelConditionsWithFlag(CancelIfCombat) really
		// did strip it while this rescue sat out, leaving the FSM in
		// Hidden and IsHidden() true. That is how a cross-room sniper
		// stayed hidden shot after shot (reported 2026-08-31): shoot.go's
		// own guard called CancelCombatConditions expecting stealth to drop,
		// and it silently did nothing.
		//
		// Asking the FSM directly is correct for every caller and cannot
		// drift again: if stealth is still on after a cancel, end it.
		if c.Awareness != nil && !c.Conditions.HasFlag(conditions.Hidden, false) &&
			c.Awareness.State() == awareness.Hidden {
			_ = c.Awareness.TransitionToRevealing(
				state.TransitionReason{Trigger: awareness.TriggerObserverSearch})
		}
		return true
	}
	return false
}

// CancelCombatConditions cancels all active conditions with the CancelIfCombat flag
// AND strips matching conditions from the permaConditionIds list so they don't
// re-apply during Validate(). Call this when a character enters combat
// (as attacker or defender) or dies.
//
// Without the permanent condition strip, mobs seeded with CancelIfCombat-tagged
// conditions via `buffids:` (e.g. Hidden on ambushers) would see the condition
// re-applied every Validate() call — the combat system would strip the
// active instance but the next Validate would put it right back. This
// surfaced as "(hidden)" tags persisting on ambushers mid-combat.
func (c *Character) CancelCombatConditions() {
	filtered := make([]int, 0, len(c.permanentConditionIds))
	for _, id := range c.permanentConditionIds {
		spec := conditions.GetConditionSpec(id)
		if spec == nil {
			filtered = append(filtered, id)
			continue
		}
		keep := true
		for _, f := range spec.Flags {
			if f == conditions.CancelIfCombat {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, id)
		}
	}
	c.permanentConditionIds = filtered

	c.CancelConditionsWithFlag(conditions.CancelIfCombat)
}

func (c *Character) HasCondition(conditionId int) bool {
	return c.Conditions.HasCondition(conditionId)
}

// RefreshCondition tops a held condition's triggers back up without resetting its
// round cadence or running Validate: a refresh changes no statmod or flag,
// so there is nothing for Validate to rebuild.
func (c *Character) RefreshCondition(conditionId int) bool {
	return c.Conditions.RefreshCondition(conditionId)
}

func (c *Character) AddCondition(conditionId int, isPermanent bool) error {
	conditionId = int(math.Abs(float64(conditionId)))
	if !c.Conditions.AddCondition(conditionId, isPermanent) {
		return fmt.Errorf(`failed to add buff. target: "%s" buffId: %d`, c.Name, conditionId)
	}
	// Chunk 6 (Perception): blind-source conditions trigger Sighted → Blinded.
	// Guard against re-entry: only fire if state is currently Sighted.
	if (conditionId == perception.ConditionIdBlinded || conditionId == perception.ConditionIdFlashbangBlindness) &&
		c.Perception != nil && c.Perception.State() == perception.Sighted {
		_ = c.Perception.TransitionTo(perception.Blinded,
			state.TransitionReason{Trigger: perception.TriggerConditionApplied, Metadata: map[string]any{"buffId": conditionId}})
	}
	c.Validate()
	return nil
}

// AddConditionScaled adds a condition with its duration scaled by durationMult.
func (c *Character) AddConditionScaled(conditionId int, durationMult float64) error {
	conditionId = int(math.Abs(float64(conditionId)))
	if !c.Conditions.AddConditionScaled(conditionId, durationMult) {
		return fmt.Errorf(`failed to add buff. target: "%s" buffId: %d`, c.Name, conditionId)
	}
	// Chunk 6 (Perception): see AddCondition above.
	if (conditionId == perception.ConditionIdBlinded || conditionId == perception.ConditionIdFlashbangBlindness) &&
		c.Perception != nil && c.Perception.State() == perception.Sighted {
		_ = c.Perception.TransitionTo(perception.Blinded,
			state.TransitionReason{Trigger: perception.TriggerConditionApplied, Metadata: map[string]any{"buffId": conditionId}})
	}
	c.Validate()
	return nil
}

// AddConditionMagnitude applies a record synchronously for an exact trigger count
// with a per-instance magnitude. It is what every former combat-condition site
// calls: those effects must be in place within the same round tick (a shout,
// a ward, a bleed) and their appliers narrate the moment themselves, so the
// record is silent-start or quiet and the event path's start notice is not
// wanted. The prune pass still narrates the end. triggers 0 means the spec's
// own triggercount. Every record that uses this door today ticks once a
// round, so the trigger count is the rounds; a stacking record takes it as the
// new stack's rounds. source overwrites the held record's Source on every
// call, so a stacking record carries its last applier's source.
func (c *Character) AddConditionMagnitude(conditionId int, triggers int, magnitude float64, source string) error {
	conditionId = int(math.Abs(float64(conditionId)))
	if !c.Conditions.AddConditionMagnitude(conditionId, triggers, magnitude) {
		return fmt.Errorf(`failed to add buff. target: "%s" buffId: %d`, c.Name, conditionId)
	}
	for _, b := range c.Conditions.GetConditions(conditionId) {
		b.Source = source
	}
	_ = c.Validate()
	return nil
}

func (c *Character) TrackConditionStarted(conditionId int) {
	c.Conditions.Started(conditionId)
}

func (c *Character) GetConditions(conditionId ...int) []*conditions.Condition {
	return c.Conditions.GetConditions(conditionId...)
}

func (c *Character) RemoveCondition(conditionId int) {
	conditionId = int(math.Abs(float64(conditionId)))
	c.Conditions.RemoveCondition(conditionId)
	// Chunk 6 (Perception): clearing a blind-source condition may flip
	// Blinded → Sighted, but only if no other blind source remains.
	if (conditionId == perception.ConditionIdBlinded || conditionId == perception.ConditionIdFlashbangBlindness) &&
		c.Perception != nil && c.Perception.State() == perception.Blinded && !c.HasAnyBlindSource() {
		_ = c.Perception.TransitionTo(perception.Sighted,
			state.TransitionReason{Trigger: perception.TriggerConditionExpired, Metadata: map[string]any{"buffId": conditionId}})
	}
	c.Validate()
}

// Used with SpawnInfo to gift spawning mobs with permanent conditions
func (c *Character) SetPermanentConditions(conditionIds []int) {
	c.permanentConditionIds = conditionIds
}

// RemovePermanentCondition removes a condition ID from the permanent condition list so
// it won't be re-applied during Validate(). Use this when a permanent condition
// should be permanently lost (e.g., revealing a hidden mob).
func (c *Character) RemovePermanentCondition(conditionId int) {
	for i, id := range c.permanentConditionIds {
		if id == conditionId {
			c.permanentConditionIds = append(c.permanentConditionIds[:i], c.permanentConditionIds[i+1:]...)
			return
		}
	}
}

func (c *Character) reapplyPermanentConditions(removedItems ...items.Item) {

	conditionIdCount := map[int]int{}

	for _, conditionId := range c.permanentConditionIds {
		conditionIdCount[conditionId] = 100 // Special case permanent conditions associated with certain mobs
	}

	// Apply any conditions that come from a species
	if rInfo := species.GetSpecies(c.SpeciesId); rInfo != nil {
		for _, conditionId := range rInfo.ConditionIds {
			conditionIdCount[conditionId] = 100 // Don't allow species conditions to be removed, keep this number high
		}
	}

	// Apply any conditions from pet
	if c.Pet.Exists() {
		for _, conditionId := range c.Pet.GetConditions() {
			conditionIdCount[conditionId] = 100 // Don't allow pet conditions to be removed, keep this number high
		}
	}

	// Track any conditions that come from an item
	// If these don't show up as still being required by an item (such as a yaml file was changed)
	// This will cause them to be removed.
	for _, b := range c.Conditions.List {
		if b.Permanent {
			if _, ok := conditionIdCount[b.ConditionId]; !ok {
				conditionIdCount[b.ConditionId] = 0
			}
		}
	}

	// Make a list of all item conditions provided by existing worn items
	for _, itm := range c.GetAllWornItems() {
		spec := itm.GetSpec()
		for _, conditionId := range spec.WornConditionIds {
			conditionIdCount[conditionId] = conditionIdCount[conditionId] + 1
		}

	}
	// Remove any conditions that come specifically from item
	for _, removedItem := range removedItems {
		iSpec := removedItem.GetSpec()
		if len(iSpec.WornConditionIds) > 0 {
			for _, conditionId := range iSpec.WornConditionIds {
				conditionIdCount[conditionId] = conditionIdCount[conditionId] - 1
			}
		}
	}

	for conditionId, ct := range conditionIdCount {
		if ct < 1 {
			c.RemoveCondition(conditionId)
		} else {
			c.AddCondition(conditionId, true)
		}
	}
}
