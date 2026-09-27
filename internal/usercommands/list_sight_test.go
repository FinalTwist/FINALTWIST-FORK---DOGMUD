package usercommands

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var listSightTag = regexp.MustCompile(`<[^>]*>`)

func listSightPlain(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, strings.TrimSpace(listSightTag.ReplaceAllString(l, "")))
	}
	return out
}

// listSightRoom seeds one room (id 8410), one browsing player (id 8411) and
// one merchant mob (instance 8412, legacy Character.Shop) carrying a single
// sellable item, in the given biome. "cave" (SkyLight 0, no lamp) is dark;
// "city" (Lamp 60) is comfortably lit.
func listSightRoom(t *testing.T, biome string) (*users.UserRecord, *rooms.Room) {
	t.Helper()

	t.Cleanup(rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"cave": {BiomeId: "cave", SkyLight: rooms.SkyLightPtr(0.0)},
		"city": {BiomeId: "city", Lamp: rooms.LampPtr(60)},
	}))

	const listSightItemId = 84001
	t.Cleanup(itemsSeedForListSight())

	merchant := &mobs.Mob{
		MobId:      841,
		InstanceId: 8412,
		HomeRoomId: 8410,
		Zone:       "TestZone",
		Character: characters.Character{
			Name:       "Keeper",
			RoomId:     8410,
			Conditions: conditions.New(),
			Shop: characters.Shop{
				{ItemId: listSightItemId, Price: 50, Quantity: 5, QuantityMax: 5},
			},
		},
	}
	merchant.Character.HealthMax.Value = 100
	merchant.Character.Health = 100
	mobs.SetInstanceForTest(8412, merchant)
	t.Cleanup(func() { mobs.SetInstanceForTest(8412, nil) })

	room := &rooms.Room{RoomId: 8410, Zone: "TestZone", Title: "Shop", Biome: biome}
	cleanupRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{8410: room},
		map[string]*rooms.ZoneConfig{
			"TestZone": {Name: "TestZone", RoomId: 8410, RoomIds: map[int]struct{}{8410: {}}},
		},
	)
	t.Cleanup(cleanupRooms)
	room.AddMob(8412)

	u := users.NewTestUser(8411, "shopper", "Shopper", 98411)
	u.Character.RoomId = 8410
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{8411: u}))
	events.DrainQueuedMessagesForTest(8411)

	return u, room
}

func itemsSeedForListSight() func() {
	return items.SeedItemsForTest(map[int]*items.ItemSpec{
		84001: {ItemId: 84001, Name: "tin cup", Type: items.Object, Value: 50},
	})
}

// TestList_DarkRoom_RefusesToDeal pins the lighting plan 5b shop sight gate:
// below the faces band a customer cannot make out the goods, so `list`
// refuses with the shared line and shows nothing.
func TestList_DarkRoom_RefusesToDeal(t *testing.T) {
	user, room := listSightRoom(t, "cave")
	require.Less(t, room.LightLevel(), 25, "fixture room must read below the blind edge")

	handled, err := List("", user, room, 0)
	require.NoError(t, err)
	assert.True(t, handled)

	lines := listSightPlain(events.DrainQueuedMessagesForTest(8411))
	require.Len(t, lines, 1)
	assert.Equal(t, "You can't make out the goods well enough to deal.", lines[0])
}

// TestList_LitRoom_Lists is the control: a comfortably lit room lists the
// merchant's stock as before.
func TestList_LitRoom_Lists(t *testing.T) {
	user, room := listSightRoom(t, "city")
	require.GreaterOrEqual(t, room.LightLevel(), 50, "fixture room must read at or above the dim edge")

	handled, err := List("", user, room, 0)
	require.NoError(t, err)
	assert.True(t, handled)

	lines := listSightPlain(events.DrainQueuedMessagesForTest(8411))
	require.NotEmpty(t, lines)
	for _, l := range lines {
		assert.NotEqual(t, "You can't make out the goods well enough to deal.", l)
	}
}
