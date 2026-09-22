package rooms

import (
	"github.com/GoMudEngine/GoMud/internal/gametime"
)

// Light levels on the graded scale. Plan 1 maps the previous three-value
// visibility model onto exactly these three points, chosen so that every
// shipped room keeps its current classification against every threshold
// defined in internal/configs/config.balance.lighting.go:
//
//   - LightBlindBelow (default 25): below this a normal observer is blind.
//   - LightDimBelow (default 50): below this an observer reads shapes only.
//   - LightExitsAbove (default 65): at or above this, exits are visible.
//
// LightDark (0) sits below LightBlindBelow (0 < 25), so a dark room stays
// blind. LightRoomOnly (60) sits at or above LightDimBelow but strictly
// below LightExitsAbove (50 <= 60 < 65), so the room reads but its exits
// do not. LightFull (70) sits at or above LightExitsAbove (70 >= 65), so
// both the room and its exits read. Those three named knobs are config,
// not Go constants, and can move; if their shipped defaults ever change,
// these three constants must be re-checked against the new values, or the
// mapping this plan depends on for behaviour preservation silently breaks.
//
// Later plans in this arc make the scale continuous. Nothing outside this
// file should assume a room's light is one of these three values.
const (
	// LightDark is a room with no light at all. A normal observer is blind.
	LightDark = 0
	// LightRoomOnly is lit enough to see the room but not down an exit.
	// It is what the old model called visibility 1.
	LightRoomOnly = 60
	// LightFull is lit enough to see the room and its exits. It is what the
	// old model called visibility 2.
	LightFull = 70
)

// LightLevel reports the room's light on the graded scale.
//
// Plan 1 deliberately computes this from the same inputs the previous
// GetVisibility used, then maps the result onto the three constants above.
// The point of this plan is the SCALE and its consumers, not new lighting
// behaviour, so a diff in what any room reports here is a defect.
func (r *Room) LightLevel() int {
	switch r.legacyVisibility() {
	case 0:
		return LightDark
	case 1:
		return LightRoomOnly
	default:
		return LightFull
	}
}

// legacyVisibility is the body of the old GetVisibility, moved here
// verbatim. GetVisibility (rooms.go) now calls through to this so the old
// and new models cannot drift apart while both exist. Task 5 deletes this
// once GetVisibility and its callers are gone.
//
// 0 = none (darkness). 1 = can see this room. 2 = can see this room and all exits
func (r *Room) legacyVisibility() int {

	visibility := 2 // default to max visibility
	// At night visibility decreases by one
	if gametime.IsNight() {
		visibility -= 1
	}

	biome := r.GetBiome()
	// First calculate natural lighting level for biome
	if biome.IsDark() { // If a naturally dark biome (cave), minimize visibility
		visibility -= 2
		if visibility < 0 {
			visibility = 0
		}
	} else if biome.IsLit() { // If the biome is naturally lit (streets with lanterns), increase visibility by one
		visibility += 1
		if visibility > 2 {
			visibility = 2
		}
	}

	// Apply any mutators
	for mut := range r.ActiveMutators {
		spec := mut.GetSpec()
		if spec.LightMod != 0 {
			visibility += spec.LightMod
		}
	}

	// min/max visibility
	if visibility < 0 {
		visibility = 0
	} else if visibility > 2 {
		visibility = 2
	}

	// If someone has light, cancel the darkness
	if visibility < 2 { // no need to increase light if it's already maxed
		if len(r.GetMobs(FindHasLight)) > 0 || len(r.GetPlayers(FindHasLight)) > 0 {
			visibility += 1
			if visibility > 2 {
				visibility = 2
			}
		}
	}

	return visibility
}
