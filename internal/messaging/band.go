package messaging

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// Band is what an observer can make out at a room's light, one step finer than
// SightDecision: it splits full sight into reading faces and being dazzled.
//
// The constants run DARKEST TO BRIGHTEST, the opposite of SightDecision's
// best-to-worst order, because the one consumer (internal/lightnotice) asks
// "did it get darker?" and an ordered comparison should read that way.
//
// Dazzled carries no penalty yet: an observer there reads fully. Plan 5 gives
// it teeth. It exists now so a notice can tell a player the light stabs at
// their eyes.
type Band uint8

const (
	BandDark Band = iota
	BandShapes
	BandFaces
	BandDazzled
)

func (b Band) String() string {
	switch b {
	case BandDark:
		return "dark"
	case BandShapes:
		return "shapes"
	case BandFaces:
		return "faces"
	case BandDazzled:
		return "dazzled"
	}
	return "unknown"
}

// BandThroughWindow is SightThroughWindow with the full tier split at the
// observer's shifted dazzle edge (windowDazzleEdge minus strength, strength
// clamped exactly as SightThroughWindow clamps it). It never moves a lower
// edge: dark, shapes and faces-or-dazzled are SightThroughWindow's answers.
func BandThroughWindow(light, strength, reach, blindBelow, dimBelow int) Band {
	switch SightThroughWindow(light, strength, reach, blindBelow, dimBelow) {
	case SightNone:
		return BandDark
	case SightShapes:
		return BandShapes
	}
	strength = clampShift(strength)
	if light >= windowDazzleEdge-strength {
		return BandDazzled
	}
	return BandFaces
}

// LightBand is ParticipantSight's band-grained twin, for a caller that needs
// to know about dazzle. It is optics only, exactly like ParticipantSight: it
// does not consult sleep. A Blinded observer is dark; a nil observer or a nil
// room reads faces, matching ParticipantSight's full-sight default at the
// non-dazzled tier.
//
// It reads the narrow lighting config rather than the 400-field Balance copy
// ParticipantSight takes; both carry the same two edges.
func LightBand(observer *characters.Character, room RoomVisibility) Band {
	if observer == nil || room == nil {
		return BandFaces
	}
	if observer.Perception != nil && observer.Perception.State() == perception.Blinded {
		return BandDark
	}
	cfg := configs.GetLightingConfig()
	return BandThroughWindow(
		room.LightLevel(),
		observer.NightVisionStrength(),
		observer.InfraReach(),
		cfg.BlindBelow,
		cfg.DimBelow,
	)
}
