package usercommands

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
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
// sellable item, in the given biome. All four biomes below are registered
// here for this test only, not shipped ones: "cave" (SkyLight 0, no lamp) is
// dark (below the blind edge); "shapes" (Lamp 37) sits between the blind edge
// (25) and the dim edge (50), the band a normal observer still only makes out
// shapes in; "city" (Lamp 60) is comfortably lit (faces, no dazzle); "dazzle"
// (Lamp 90) is above the dazzle edge (75), still full sight.
func listSightRoom(t *testing.T, biome string) (*users.UserRecord, *rooms.Room) {
	t.Helper()

	t.Cleanup(rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"cave":   {BiomeId: "cave", SkyLight: rooms.SkyLightPtr(0.0)},
		"shapes": {BiomeId: "shapes", SkyLight: rooms.SkyLightPtr(0.0), Lamp: rooms.LampPtr(37)},
		"city":   {BiomeId: "city", Lamp: rooms.LampPtr(60)},
		"dazzle": {BiomeId: "dazzle", SkyLight: rooms.SkyLightPtr(0.0), Lamp: rooms.LampPtr(90)},
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

// listedStockNames returns the item names List's own row-building path
// (partitionShopStock -> buildItemRows, the exact functions renderMobMerchant
// Listing calls) produces for the seeded merchant's current stock.
//
// This sidesteps internal/templates.Process: this package's TestMain
// redirects FilePaths.DataFiles to an empty temp dir (so save writes can't
// race a root-guard walk), and internal/templates.readFile's fileSystems
// slice is never populated in a bare `go test` run here (no module import
// chain calls templates.RegisterFS), so every Process call in this test
// binary renders empty content regardless of DataFiles, a pre-existing gap
// in that package, outside this task. buildItemRows is where List() commits
// to a stock NAME before handing rows to that renderer, so asserting on its
// output still proves the sight gate is choosing to show (or not show) this
// merchant's actual goods.
func listedStockNames(t *testing.T) []string {
	t.Helper()
	merchant := mobs.GetInstance(8412)
	require.NotNil(t, merchant, "fixture merchant must exist")
	merchant.Character.Shop.Restock()
	itemStock, _, _, _ := partitionShopStock(merchant.Character.Shop.GetInstock())
	_, rows := buildItemRows(itemStock, true, false)
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row[1]) // columns are Qty, Name, Type, [Price]
	}
	return names
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
	assert.Equal(t, actions.ShopSightRefusalText, lines[0])
}

// TestList_ShapesRoom_RefusesToDeal is dark's sibling in the OTHER refused
// band: light 37 sits strictly between the blind edge (25) and the dim edge
// (50), so a normal observer makes out shapes only, not faces. Refusal must
// not be an accident of total darkness; the shapes band refuses too.
func TestList_ShapesRoom_RefusesToDeal(t *testing.T) {
	user, room := listSightRoom(t, "shapes")
	require.GreaterOrEqual(t, room.LightLevel(), 25, "fixture room must read at or above the blind edge")
	require.Less(t, room.LightLevel(), 50, "fixture room must read below the dim edge")

	handled, err := List("", user, room, 0)
	require.NoError(t, err)
	assert.True(t, handled)

	lines := listSightPlain(events.DrainQueuedMessagesForTest(8411))
	require.Len(t, lines, 1)
	assert.Equal(t, actions.ShopSightRefusalText, lines[0])
}

// TestList_LitRoom_Lists is the control: a comfortably lit room lists the
// merchant's stock as before, by name.
func TestList_LitRoom_Lists(t *testing.T) {
	user, room := listSightRoom(t, "city")
	require.GreaterOrEqual(t, room.LightLevel(), 50, "fixture room must read at or above the dim edge")

	handled, err := List("", user, room, 0)
	require.NoError(t, err)
	assert.True(t, handled)

	lines := listSightPlain(events.DrainQueuedMessagesForTest(8411))
	require.NotEmpty(t, lines)
	joined := strings.Join(lines, "\n")
	assert.NotContains(t, joined, actions.ShopSightRefusalText)

	names := listedStockNames(t)
	assert.Contains(t, names, "Tin Cup", "the merchant's stock name must actually appear, not just an absent refusal")
}

// TestList_DazzledRoom_StillLists pins the far edge of full sight: dazzled
// (light 90, above the dazzle edge of 75) is still faces-or-better, so
// ShopSightRefusal reads false and List works exactly like a comfortable
// room, with no refusal line. Dazzle costs a haggler's discount (see
// internal/actions/shop_sight_test.go's barterDiscount tests), not the deal
// itself.
func TestList_DazzledRoom_StillLists(t *testing.T) {
	user, room := listSightRoom(t, "dazzle")
	require.GreaterOrEqual(t, room.LightLevel(), 75, "fixture room must read at or above the dazzle edge")

	assert.False(t, actions.ShopSightRefusal(user.Character, room),
		"dazzled is still full sight and must not refuse")

	handled, err := List("", user, room, 0)
	require.NoError(t, err)
	assert.True(t, handled)

	lines := listSightPlain(events.DrainQueuedMessagesForTest(8411))
	require.NotEmpty(t, lines)
	for _, l := range lines {
		assert.NotEqual(t, actions.ShopSightRefusalText, l)
	}

	names := listedStockNames(t)
	assert.Contains(t, names, "Tin Cup", "the merchant's stock name must actually appear, not just an absent refusal")
}
