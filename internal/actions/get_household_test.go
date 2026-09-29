package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A household's bauble on the floor is refused to every taker, not only a
// player's `get` (owner ruling 2026-09-29): mobs, companions and scavengers
// all pick up through GetItemFromFloor. Nothing moves. A bauble that is no
// household's is taken as ever.
func TestGetItemFromFloor_RefusesAHouseholdsBaubleToEveryTaker(t *testing.T) {
	seedBaubleSale(t)
	char := newTestChar()
	room := newTestRoom()
	room.RoomId = 9701
	actor := newStubActor(char, room) // not a player: a mob's, companion's or scavenger's get

	theirs := newBauble(t, "Small Child's Doll", "doll", 3, baubles.StatusReady)
	theirs.BaubleHousehold = room.RoomId
	room.Items = append(room.Items, theirs)

	result := GetItemFromFloor(actor, "doll", false)
	require.True(t, result.Found, "the doll is found")
	require.ErrorIs(t, result.Err, ErrHouseholdBauble)
	assert.Equal(t, 0, countCharItems(char), "nothing taken")
	assert.Equal(t, 1, countFloorItems(room), "the doll stays where it lies")

	room.Items[0].BaubleHousehold = 0 // nobody's household's now
	result = GetItemFromFloor(actor, "doll", false)
	require.True(t, result.Found)
	require.NoError(t, result.Err)
	assert.Equal(t, 1, countCharItems(char), "taken as ever")
}
