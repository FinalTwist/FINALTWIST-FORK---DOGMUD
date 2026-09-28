package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// merchantSay makes the merchant speak a line to the room synchronously. The
// canonical implementation now lives in actions.Sell (unexported); this thin
// helper keeps the offer command — which previously shared usercommands'
// package-local merchantSay — working after the sell lift. See the long-form
// rationale on the async-pipeline pitfall in internal/actions/sell.go.
func merchantSay(room *rooms.Room, mob *mobs.Mob, line string) {
	if mob == nil || room == nil {
		return
	}
	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.Say(actor, line)
	room.SendText(messaging.CategorySpeech,
		actions.FormatSayText(mob.Character.Name, result.Text, false, "mobname", "saytext-mob"))
}

func Offer(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	item, found := user.Character.FindInBackpack(rest)
	if !found {
		user.SendText(messaging.CategorySystem, "You don't have that item.")
		return true, nil
	}

	itemSpec := item.GetSpec()
	if itemSpec.ItemId < 1 {
		return true, nil
	}

	buyers := room.GetMobs(rooms.FindMerchant)
	if item.IsBauble() {
		buyers = actions.BaubleBuyersInRoom(room) // fences who keep no shop buy baubles too
	}
	for _, mobId := range buyers {

		mob := mobs.GetInstance(mobId)
		if mob == nil {
			continue
		}

		user.Character.CancelConditionsWithFlag(conditions.Hidden)

		// Baubles are priced from their catalog record (docs/baubles).
		if item.IsBauble() {
			offer := actions.BaubleOfferFrom(item, mob)
			if offer.Price <= 0 {
				merchantSay(room, mob, offer.Refusal)
				continue
			}
			merchantSay(room, mob, fmt.Sprintf(`I can give you <ansi fg="gold">%d gold</ansi> for that <ansi fg="itemname">%s</ansi>.`, offer.Price, item.DisplayName()))
			break
		}

		if item.IsSpecial() {

			merchantSay(room, mob, "I'm afraid I don't buy those.")

			continue
		}

		sellValue := mob.GetSellPrice(item)

		if sellValue <= 0 {

			merchantSay(room, mob, "I'm not interested in that.")

			continue
		}

		merchantSay(room, mob, fmt.Sprintf(`I can give you <ansi fg="gold">%d gold</ansi> for that <ansi fg="itemname">%s</ansi>.`, sellValue, item.DisplayName()))

		break
	}

	return true, nil
}
