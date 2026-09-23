package rooms

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
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
// Plan 1 deliberately computes this from the same inputs the old visibility
// model used (legacyVisibility, below), then maps the result onto the three
// constants above. The point of this plan is the SCALE and its consumers,
// not new lighting behaviour, so a diff in what any room reports here is a
// defect.
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

// IsLit reports whether the room's CURRENT light (LightLevel, above) is
// enough for a normal observer to see anything at all: at or above
// LightBlindBelow.
//
// This is the room-level predicate plan 1 promised and never built. It is
// defined purely in terms of LightLevel(), so it does not change what any
// room reports; it only gives production callers a name for "is this room
// lit" that reads room light instead of reaching past it into a biome's
// DarkArea/LitArea flags, which describe the biome's natural tendency, not
// what a mutator, time of day or someone's torch left the room at.
//
// Reads configs.GetLightingConfig(), not configs.GetBalanceConfig(): the
// latter copies a 424-field struct under two read locks (about 99 ns)
// against about 10 ns for the narrow accessor, and LightLevel and IsLit are
// both called from per-round loops.
func (r *Room) IsLit() bool {
	return r.LightLevel() >= configs.GetLightingConfig().BlindBelow
}

// legacyVisibility is the body of the room's old three-value visibility
// accessor, moved here verbatim when that old accessor was still a
// call-through to it. That accessor itself was deleted in Task 5 once
// LightLevel and its callers took over, but this body stays, since
// LightLevel still computes from it.
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
