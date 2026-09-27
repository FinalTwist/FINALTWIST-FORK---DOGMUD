package actions

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var shopSightTag = regexp.MustCompile(`<[^>]*>`)

func shopSightPlain(line string) string {
	return strings.TrimSpace(shopSightTag.ReplaceAllString(line, ""))
}

// darkenSellRoom mutates the seedSellRoom fixture (room 1, normally pinned
// comfortably lit) to read below the blind edge: no lamp, no sky. Returns the
// room for assertions.
func darkenSellRoom(t *testing.T) *rooms.Room {
	t.Helper()
	room := rooms.LoadRoom(1)
	require.NotNil(t, room, "room 1 must exist")
	room.Lamp = nil
	room.SkyLight = rooms.SkyLightPtr(0.0)
	require.Less(t, room.LightLevel(), 25, "fixture room must read below the blind edge")
	return room
}

// TestBuy_DarkRoom_Refuses pins the lighting plan 5b shop sight gate on the
// shared Buy entry point: below the faces band a buyer can't make out the
// goods, so the purchase refuses with the shared line before any purchase
// logic runs, and no gold or stock moves.
func TestBuy_DarkRoom_Refuses(t *testing.T) {
	defer seedSellItemSpecs()()
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()
	darkenSellRoom(t)

	buyer := newSellerActor(t, true)
	char := buyer.GetCharacter()
	char.Gold = 1000
	events.DrainQueuedMessagesForTest(buyer.GetUserId())

	res := Buy(buyer, BuyOptions{Request: "iron sword"})

	assert.False(t, res.Success)
	assert.Equal(t, BuyReasonNoSight, res.Reason)
	assert.Equal(t, 1000, char.Gold, "gold must not move on a sight refusal")

	lines := events.DrainQueuedMessagesForTest(buyer.GetUserId())
	require.Len(t, lines, 1)
	assert.Equal(t, ShopSightRefusalText, shopSightPlain(lines[0]))
}

// TestSell_DarkRoom_Refuses is Buy's twin on the Sell entry point.
func TestSell_DarkRoom_Refuses(t *testing.T) {
	defer seedSellItemSpecs()()
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()
	darkenSellRoom(t)

	seller := newSellerActor(t, true, sellTestItemId)
	char := seller.GetCharacter()
	goldBefore := char.Gold
	events.DrainQueuedMessagesForTest(seller.GetUserId())

	res := Sell(seller, SellOptions{ItemName: "iron sword", Quantity: 1})

	assert.Equal(t, 0, res.Sold)
	assert.Equal(t, SellStopNoSight, res.Reason)
	assert.Equal(t, goldBefore, char.Gold, "gold must not move on a sight refusal")
	_, stillHeld := char.FindInBackpack("iron sword")
	assert.True(t, stillHeld, "the item must stay in the seller's backpack")

	lines := events.DrainQueuedMessagesForTest(seller.GetUserId())
	require.Len(t, lines, 1)
	assert.Equal(t, ShopSightRefusalText, shopSightPlain(lines[0]))
}

// TestBarterDiscount_DazzledLessThanComfortable pins barterDiscount's sight
// coupling (lighting plan 5b): a comfortable haggler (light 60, faces band,
// no dazzle) keeps the full skill-derived discount; a dazzled one (light 90)
// bargains worse, at exactly SightMult's 0.88 for a normal observer under the
// Go test default lighting config (DazzleCap 0.80, dim/dazzle edges 50/75).
// 0.15 (skill 50, at the cap) * 0.88 = 0.132 exactly.
func TestBarterDiscount_DazzledLessThanComfortable(t *testing.T) {
	char := &characters.Character{Skills: map[string]int{string(skills.Bartering): 50}}

	comfortable := &rooms.Room{RoomId: 90001, SkyLight: rooms.SkyLightPtr(0.0), Lamp: rooms.LampPtr(60)}
	dazzled := &rooms.Room{RoomId: 90002, SkyLight: rooms.SkyLightPtr(0.0), Lamp: rooms.LampPtr(90)}

	comfortableDiscount := barterDiscount(char, comfortable)
	dazzledDiscount := barterDiscount(char, dazzled)

	assert.InDelta(t, 0.15, comfortableDiscount, 1e-9,
		"a comfortable haggler at skill 50 keeps the full 15% discount")
	assert.InDelta(t, 0.132, dazzledDiscount, 1e-9,
		"0.15 x 0.88 exactly: dazzle costs SightMult 0.88 at skill 50")
	assert.Less(t, dazzledDiscount, comfortableDiscount,
		"a dazzled haggler must bargain worse than a comfortable one")
}
