package messaging

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// ComfortDistance reports how far a room's light sits outside the observer's
// own comfortable band, as two fractions of the way to the cap: dark (below
// the dim edge, 1 at the blind edge) and bright (above the dazzle edge, 1 one
// ramp-width beyond it). At most one is non-zero. It is the geometry behind
// the sight penalty (lighting plan 5b); SightScoreMultiplier turns it into a
// number. A nil observer or room is comfortable; a Blinded observer is fully
// dark.
func ComfortDistance(observer *characters.Character, room RoomVisibility) (dark, bright float64) {
	if observer == nil || room == nil {
		return 0, 0
	}
	if observer.Perception != nil && observer.Perception.State() == perception.Blinded {
		return 1, 0
	}
	cfg := configs.GetLightingConfig()
	return comfortDistance(room.LightLevel(), observer.NightVisionStrength(), cfg.BlindBelow, cfg.DimBelow, cfg.DazzleAbove)
}

// comfortDistance is the pure form. The bright ramp is as wide as the dark
// one, so a strong window is punished by excess light as fast as it is helped
// by faint light; infra reach does not soften either side.
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
