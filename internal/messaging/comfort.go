package messaging

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// ComfortDistance reports how far a room's light sits outside the observer's
// own comfortable band, as two fractions of the way to the cap: dark (below
// the dim edge, 1 at the blind edge) and bright (above the dazzle edge, 0 AT
// the edge itself and 1 one ramp-width beyond it). At most one is non-zero.
// Bright reading 0 at the dazzle edge is deliberate, not a rounding gap: the
// band already calls that light dazzled with no cost from this function, and
// the ramp only starts pricing it a step further in. It is the geometry
// behind the sight penalty (lighting plan 5b); SightScoreMultiplier turns it
// into a number. A nil observer or room is comfortable; a Blinded observer is
// fully dark. Infravision caps the dark fraction (infraDarkCap) and never the
// bright one.
func ComfortDistance(observer *characters.Character, room RoomVisibility) (dark, bright float64) {
	if observer == nil || room == nil {
		return 0, 0
	}
	if observer.Perception != nil && observer.Perception.State() == perception.Blinded {
		return 1, 0
	}
	cfg := configs.GetLightingConfig()
	light := room.LightLevel()
	dark, bright = comfortDistance(light, observer.NightVisionStrength(), cfg.BlindBelow, cfg.DimBelow, cfg.DazzleAbove)
	// With no dark penalty there is nothing for infravision to ease, so skip
	// InfraReach, which log-combines every held source on each call.
	if dark == 0 {
		return dark, bright
	}
	if capped, ok := infraDarkCap(light, observer.InfraReach(), cfg.InfraPenaltyFloor, cfg.InfraReachCap, cfg.DarkCap); ok && capped < dark {
		dark = capped
	}
	return dark, bright
}

// comfortDistance is the pure form. The bright ramp is as wide as the dark
// one, so a strong window is punished by excess light as fast as it is helped
// by faint light. It takes no infra reach: ComfortDistance applies
// infraDarkCap to the dark side afterwards, and nothing softens the bright
// side. A negative
// LightBlindBelow (the never-blind escape hatch) widens the dark-to-dim span
// that both ramps take their width from, so a never-blind config also
// softens dazzle, not only darkness.
func comfortDistance(light, strength, blindBelow, dimBelow, dazzleAbove int) (dark, bright float64) {
	s := clampShift(strength)
	blind, dim, dazzle := blindBelow-s, dimBelow-s, dazzleAbove-s
	width := float64(dim - blind)
	if width <= 0 {
		width = 1
	}
	switch {
	case light < dim:
		dark = float64(dim-light) / width
		if dark > 1 {
			dark = 1
		}
	case light >= dazzle:
		bright = float64(light-dazzle) / width
		if bright > 1 {
			bright = 1
		}
	}
	return dark, bright
}

// infraDarkCap is the dark fraction infravision holds an observer to
// (lighting plan 5c). Infravision's sight multiplier runs linearly from floor
// at the first point of reach to 1.0 at reachCap; expressed as a dark
// fraction against darkCap (Balance.DarknessCombatPenalty) it makes
// SightScoreMultiplier read exactly that multiplier, so every caller of
// ComfortDistance and SightMult gets max(natural ramp, infra) with no change.
// ok is false when infravision does not apply: no reach, light below minus
// reach, or no dark penalty to ease. It never touches the bright side; glare
// costs an infravision creature in full (owner ruling 2026-09-28).
func infraDarkCap(light, reach int, floor float64, reachCap int, darkCap float64) (float64, bool) {
	if reach <= 0 || light < -reach || reachCap <= 0 || darkCap >= 1 {
		return 0, false
	}
	r := min(reach, reachCap)
	mult := floor + (1-floor)*float64(r)/float64(reachCap)
	return (1 - mult) / (1 - darkCap), true
}
