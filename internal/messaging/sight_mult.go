package messaging

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
)

// SightScoreMultiplier is the ONE place the sight ramp turns into a number
// (lighting plan 5b). dark and bright are ComfortDistance's fractions; the
// multiplier runs from 1.0 at the edge of the comfortable band to
// Balance.DarknessCombatPenalty at the blind edge and to Balance.DazzleCap one
// ramp-width above the dazzle edge. It prices every opposed or difficulty
// roll, not only combat; the voice contests are exempt by never asking.
//
// It lives here, not in combat, because mobs (the shopkeeper crafter) and the
// crafting round ticks need the same body and sit below combat in the import
// graph.
func SightScoreMultiplier(dark, bright float64, bal configs.Balance) float64 {
	mult := 1.0 - dark*(1.0-float64(bal.DarknessCombatPenalty)) - bright*(1.0-float64(bal.DazzleCap))
	if mult < 0 {
		return 0
	}
	return mult
}

// SightMult is the sight ramp for one roller in one room: apply it to the
// score of whoever needs to SEE for the roll. In a detection roll that is the
// observer; in a theft the thief; in a search, track, defuse, forage, craft or
// concentration roll the actor. The voice contests never call it. It composes
// ComfortDistance and SightScoreMultiplier so no site can pair them
// differently.
func SightMult(c *characters.Character, room RoomVisibility) float64 {
	dark, bright := ComfortDistance(c, room)
	return SightScoreMultiplier(dark, bright, configs.GetBalanceConfig())
}
