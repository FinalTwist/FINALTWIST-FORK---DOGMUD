package actions

import (
	"errors"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// ErrHouseholdBauble refuses taking a household's bauble off the floor: it
// belongs to the house, and taking it is theft, which only `steal`
// attempts (stealHouseholdBauble). Every taker is held to it, a player's
// `get`, a mob's, a companion's or a scavenger's (owner ruling 2026-09-29).
var ErrHouseholdBauble = errors.New(`that belongs to this household`)

// GetItemResult is the result of a GetItemFromFloor call.
type GetItemResult struct {
	Item  items.Item
	Found bool
	Err   error
}

// GetItemFromFloor searches the room floor (or stash) for an item matching
// itemName, then atomically moves it into the actor's backpack using
// TransferItemToBackpack. The stash flag mirrors room.FindOnFloor /
// room.RemoveItem / room.AddItem semantics — pass true to search the stash.
// A household's bauble is refused with ErrHouseholdBauble (Found, nothing moved).
func GetItemFromFloor(actor Actor, itemName string, stash bool) GetItemResult {
	room := actor.GetRoom()

	matchItem, found := room.FindOnFloor(itemName, stash)
	if !found {
		return GetItemResult{Found: false}
	}

	// Gates: each refuses with the item it found and an error, and moves
	// nothing; the caller words the refusal. A household's bauble is never
	// picked up (it is only ever on the floor, never in a stash).
	if !stash && matchItem.BaubleBelongsTo(room.RoomId) {
		return GetItemResult{Item: matchItem, Found: true, Err: ErrHouseholdBauble}
	}

	char := actor.GetCharacter()
	err := TransferItemToBackpack(
		matchItem,
		char,
		actor.GetUserId(),
		actor.GetMobInstanceId(),
		func(i items.Item) { room.RemoveItem(i, stash) },
		func(i items.Item) { room.AddItem(i, stash) },
	)

	return GetItemResult{Item: matchItem, Found: true, Err: err}
}

// GetGoldFromFloor moves all gold currently on the room floor into the actor's
// character wallet. It delegates to FloorPickupGold which validates the amount.
func GetGoldFromFloor(actor Actor, amount int) error {
	return FloorPickupGold(amount, actor.GetRoom(), actor.GetCharacter())
}
