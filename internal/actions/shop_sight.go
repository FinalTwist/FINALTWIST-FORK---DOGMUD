package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// ShopSightRefusal is the sight gate on dealing (lighting plan 5b): below the
// faces band you cannot make out the goods, so list, buy and sell refuse. It
// mirrors ShopClosedForSleep beside it. Above faces the deal goes through and
// the bartering discount takes SightMult, so a dazzled haggler bargains worse.
//
// A nil room reads as no refusal, since callers that reach this gate have
// already handled "no room at all" their own way, so this never has to.
// Gating here also keeps a typed-nil *rooms.Room from reaching
// messaging.LightBand, which takes it as a non-nil RoomVisibility and would
// panic reading it.
func ShopSightRefusal(c *characters.Character, room *rooms.Room) bool {
	if room == nil {
		return false
	}
	return messaging.LightBand(c, room) < messaging.BandFaces
}

// ShopSightRefusalText is the one line every refusing verb prints.
const ShopSightRefusalText = "You can't make out the goods well enough to deal."

// barterDiscount is the ONE place a shop's bartering discount is computed
// (lighting plan 5b): the raw skill-derived discount, capped at 15% same as
// before, times SightMult, so a dazzled haggler bargains worse than a
// comfortable one, who reads SightMult 1.0 and sees no change at all.
//
// A nil room reads as comfortable (SightMult 1.0, unmultiplied), the same
// reading messaging.SightMult itself gives a nil room. This only matters to
// a caller exercising tryPurchaseFromInventory directly with no Room set on
// its Actor fixture, and a typed-nil *rooms.Room must never reach SightMult
// (see ShopSightRefusal's doc comment).
func barterDiscount(char *characters.Character, room *rooms.Room) float64 {
	barterSkill := char.GetSkillLevel(skills.Bartering)
	if barterSkill <= 0 {
		return 0
	}
	discount := float64(barterSkill) / 50.0 * 0.15
	if discount > 0.15 {
		discount = 0.15
	}
	if room == nil {
		return discount
	}
	return discount * messaging.SightMult(char, room)
}
