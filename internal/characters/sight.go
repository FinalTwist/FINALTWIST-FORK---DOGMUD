// Vision-related helpers on Character. Currently houses HasAnyBlindSource
// for chunk 6's Perception machine. Future messaging framework will add
// CanSee / CanSeeClearly / CanSeeShapes here.
package characters

import (
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// HasAnyBlindSource returns true if any active blind source is currently
// affecting this character. Used by Perception expire-paths in AddCondition and
// RemoveCondition to decide whether to fire the Blinded→Sighted transition when
// one of multiple overlapping sources clears.
//
// Sources checked:
//   - Condition 3 (Blinded) — _datafiles/world/dogmud/buffs/3-blinded.yaml
//   - Condition 77 (Flashbang Blindness) — _datafiles/world/dogmud/buffs/77-flashbang_blindness.yaml
//
// Note: uses TriggersLeft > 0 rather than HasCondition to correctly detect the
// "just removed" state. RemoveCondition marks a condition expired (TriggersLeft=0) but
// defers the actual map-entry prune to the next game-tick Advance call.
// HasCondition checks map membership only and returns true for expired conditions;
// TriggersLeft > 0 returns false immediately after RemoveCondition fires.
func (c *Character) HasAnyBlindSource() bool {
	if c == nil {
		return false
	}
	if c.Conditions.TriggersLeft(perception.ConditionIdBlinded) > 0 {
		return true
	}
	if c.Conditions.TriggersLeft(perception.ConditionIdFlashbangBlindness) > 0 {
		return true
	}
	return false
}
