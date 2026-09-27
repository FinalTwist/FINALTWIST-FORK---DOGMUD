package actions

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
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

// dimSellRoom mutates the seedSellRoom fixture into the shapes band: at or
// above the blind edge (25) but below the dim edge (50), the OTHER refused
// band, distinct from total darkness.
func dimSellRoom(t *testing.T) *rooms.Room {
	t.Helper()
	room := rooms.LoadRoom(1)
	require.NotNil(t, room, "room 1 must exist")
	room.SkyLight = rooms.SkyLightPtr(0.0)
	room.Lamp = rooms.LampPtr(37)
	require.GreaterOrEqual(t, room.LightLevel(), 25, "fixture room must read at or above the blind edge")
	require.Less(t, room.LightLevel(), 50, "fixture room must read below the dim edge")
	return room
}

// dazzleSellRoom mutates the seedSellRoom fixture above the dazzle edge (75):
// still full sight, just bright enough to cost a haggler's discount.
func dazzleSellRoom(t *testing.T) *rooms.Room {
	t.Helper()
	room := rooms.LoadRoom(1)
	require.NotNil(t, room, "room 1 must exist")
	room.SkyLight = rooms.SkyLightPtr(0.0)
	room.Lamp = rooms.LampPtr(90)
	require.GreaterOrEqual(t, room.LightLevel(), 75, "fixture room must read at or above the dazzle edge")
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

// TestSell_ShapesRoom_Refuses is TestSell_DarkRoom_Refuses's sibling in the
// OTHER refused band: light 37 sits strictly between the blind edge (25) and
// the dim edge (50), so a normal observer makes out shapes only, not faces.
// Refusal must not be an accident of total darkness.
func TestSell_ShapesRoom_Refuses(t *testing.T) {
	defer seedSellItemSpecs()()
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()
	dimSellRoom(t)

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

// TestSell_DazzledRoom_StillSells pins the far edge of full sight: dazzled
// (light 90, above the dazzle edge of 75) is still faces-or-better, so
// ShopSightRefusal reads false and a sale completes normally. Dazzle costs a
// haggler's discount (see TestBarterDiscount_DazzledLessThanComfortable
// below), not the deal itself.
func TestSell_DazzledRoom_StillSells(t *testing.T) {
	defer seedSellItemSpecs()()
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()
	room := dazzleSellRoom(t)

	seller := newSellerActor(t, true, sellTestItemId)
	char := seller.GetCharacter()
	goldBefore := char.Gold

	assert.False(t, ShopSightRefusal(char, room), "dazzled is still full sight and must not refuse")

	res := Sell(seller, SellOptions{ItemName: "iron sword", Quantity: 1})

	assert.Equal(t, 1, res.Sold)
	assert.NotEqual(t, SellStopNoSight, res.Reason)
	assert.Greater(t, char.Gold, goldBefore, "the sale must complete in a dazzled room")
}

// TestBuy_DazzledRoom_StillBuys is TestSell_DazzledRoom_StillSells's Buy-side
// twin: dazzled (light 90) still completes a purchase, and the price paid
// reflects the 0.88 SightMult multiplier on the bartering discount (0.15 x
// 0.88 = 0.132), the same pin TestBarterDiscount_DazzledLessThanComfortable
// uses below. Exercises tryPurchaseFromInventory directly (bypassing
// room/merchant discovery), same as TestBuy_AffixedStockItem in buy_test.go:
// the ShopInventory path needs no loaded shop data files, unlike the full
// Buy() entry point (see the file comment on seedSellRoom's siblings).
func TestBuy_DazzledRoom_StillBuys(t *testing.T) {
	defer seedSellItemSpecs()()

	cfg := configs.GetConfig()
	cfg.Balance.BarterMaxDiscount = 0.15
	configs.SetConfigForTest(t, cfg)

	dazzled := &rooms.Room{RoomId: 90020, SkyLight: rooms.SkyLightPtr(0.0), Lamp: rooms.LampPtr(90)}
	require.GreaterOrEqual(t, dazzled.LightLevel(), 75, "fixture room must read at or above the dazzle edge")

	m := &mobs.Mob{}
	m.Character.Name = "Buyer"
	m.Character.Gold = 1000
	m.Character.Conditions = conditions.New()
	m.Character.Stats.Strength.ValueAdj = 100 // carry capacity
	m.Character.Skills = map[string]int{string(skills.Bartering): 50}
	buyer := &MobActor{Mob: m, Room: dazzled}

	assert.False(t, ShopSightRefusal(buyer.GetCharacter(), dazzled),
		"dazzled is still full sight and must not refuse")

	// Compute the expected price the same way tryPurchaseFromInventory does,
	// so this test proves the DISCOUNT reflects SightMult rather than
	// asserting a hand-derived magic number.
	pricingCfg := shops.PricingConfigFromBalance()
	entry := shops.StockEntry{ItemId: sellTestItemId, Current: 5, MaxStock: 10, RestockQty: 5}
	basePrice := shops.CalcSellPrice(100, entry.Current, shops.PricingBaseline(&entry, pricingCfg), pricingCfg)
	expectedDiscount := 0.15 * 0.88 // 0.15 cap x SightMult 0.88 at light 90, skill 50
	expectedPrice := shops.ApplyBarterSellDiscount(basePrice, expectedDiscount)
	plainPrice := shops.ApplyBarterSellDiscount(basePrice, 0.15) // undazzled (SightMult 1.0)
	require.NotEqual(t, plainPrice, expectedPrice,
		"fixture must actually distinguish the dazzled price from the plain 0.15 price, or rounding on this small base price could make the test vacuous")

	shopInv := &shops.ShopInventory{Gold: 1000, Stock: []shops.StockEntry{entry}}
	shopMob := &mobs.Mob{Character: characters.Character{Name: "Merchant"}}

	res := tryPurchaseFromInventory(buyer, "iron sword", shopMob, shopInv)

	require.True(t, res.Success, "%+v", res)
	assert.Less(t, m.Character.Gold, 1000, "gold must go down on a successful purchase")
	assert.Equal(t, 1000-expectedPrice, m.Character.Gold,
		"price paid must reflect the 0.88 dazzle discount multiplier")
	assert.NotEqual(t, 1000-plainPrice, m.Character.Gold,
		"the dazzled price must differ from the plain (non-dazzled) 0.15 price")
}

// TestBuy_BarterMaxDiscount_NonDefaultKnobReachesPrice pins the KNOB WIRING
// itself, not just the discount math: every other test in this file pins
// Balance.BarterMaxDiscount at 0.15, which is ALSO the pre-lighting-plan-5b
// hardcoded literal and the shipped default, so a buy.go call site that
// regressed to that literal instead of reading configs.GetBalanceConfig()
// would still pass every one of them. Pinning a value the old literal never
// took (0.30) is the only way to catch that regression: a comfortable
// (non-dazzled) skill-50 buyer must pay a price reflecting a 0.30 discount,
// not 0.15.
func TestBuy_BarterMaxDiscount_NonDefaultKnobReachesPrice(t *testing.T) {
	defer seedSellItemSpecs()()

	cfg := configs.GetConfig()
	cfg.Balance.BarterMaxDiscount = 0.30
	configs.SetConfigForTest(t, cfg)

	comfortable := &rooms.Room{RoomId: 90022, SkyLight: rooms.SkyLightPtr(0.0), Lamp: rooms.LampPtr(60)}
	require.GreaterOrEqual(t, comfortable.LightLevel(), 50, "fixture room must read at or above the dim edge")
	require.Less(t, comfortable.LightLevel(), 75, "fixture room must read below the dazzle edge")

	m := &mobs.Mob{}
	m.Character.Name = "Buyer"
	m.Character.Gold = 1000
	m.Character.Conditions = conditions.New()
	m.Character.Stats.Strength.ValueAdj = 100 // carry capacity
	m.Character.Skills = map[string]int{string(skills.Bartering): 50}
	buyer := &MobActor{Mob: m, Room: comfortable}

	assert.False(t, ShopSightRefusal(buyer.GetCharacter(), comfortable),
		"a comfortably lit room must not refuse")

	pricingCfg := shops.PricingConfigFromBalance()
	entry := shops.StockEntry{ItemId: sellTestItemId, Current: 5, MaxStock: 10, RestockQty: 5}
	basePrice := shops.CalcSellPrice(100, entry.Current, shops.PricingBaseline(&entry, pricingCfg), pricingCfg)
	pinnedPrice := shops.ApplyBarterSellDiscount(basePrice, 0.30) // the pinned knob, comfortable so SightMult 1.0
	oldLiteralPrice := shops.ApplyBarterSellDiscount(basePrice, 0.15)
	require.NotEqual(t, pinnedPrice, oldLiteralPrice,
		"fixture must actually distinguish the pinned 0.30 knob from the old hardcoded 0.15, or rounding could make the test vacuous")

	shopInv := &shops.ShopInventory{Gold: 1000, Stock: []shops.StockEntry{entry}}
	shopMob := &mobs.Mob{Character: characters.Character{Name: "Merchant"}}

	res := tryPurchaseFromInventory(buyer, "iron sword", shopMob, shopInv)

	require.True(t, res.Success, "%+v", res)
	assert.Equal(t, 1000-pinnedPrice, m.Character.Gold,
		"price paid must reflect the pinned 0.30 knob, not a literal 0.15")
}

// TestShopSightRefusal_Bands is a direct, fixture-free pin of every band
// ShopSightRefusal must tell apart: dark and shapes refuse, faces and
// dazzled do not.
func TestShopSightRefusal_Bands(t *testing.T) {
	char := &characters.Character{}

	dark := &rooms.Room{RoomId: 90010, SkyLight: rooms.SkyLightPtr(0.0)}
	shapes := &rooms.Room{RoomId: 90011, SkyLight: rooms.SkyLightPtr(0.0), Lamp: rooms.LampPtr(37)}
	faces := &rooms.Room{RoomId: 90012, SkyLight: rooms.SkyLightPtr(0.0), Lamp: rooms.LampPtr(60)}
	dazzled := &rooms.Room{RoomId: 90013, SkyLight: rooms.SkyLightPtr(0.0), Lamp: rooms.LampPtr(90)}

	assert.True(t, ShopSightRefusal(char, dark), "dark must refuse")
	assert.True(t, ShopSightRefusal(char, shapes), "shapes must refuse")
	assert.False(t, ShopSightRefusal(char, faces), "faces must not refuse")
	assert.False(t, ShopSightRefusal(char, dazzled), "dazzled must not refuse")
}

// TestBarterDiscount_DazzledLessThanComfortable pins barterDiscount's sight
// coupling (lighting plan 5b): a comfortable haggler (light 60, faces band,
// no dazzle) keeps the full skill-derived discount; a dazzled one (light 90)
// bargains worse, at exactly SightMult's 0.88 for a normal observer under the
// Go test default lighting config (DazzleCap 0.80, dim/dazzle edges 50/75).
// 0.15 (skill 50, at the cap) * 0.88 = 0.132 exactly.
//
// The cap itself is the shipped knob Balance.BarterMaxDiscount, never a Go
// literal (see barterDiscount's doc comment), so it is pinned explicitly here
// via SetConfigForTest rather than trusted to stay 0.15 by coincidence: the
// 0.132 pin holds only as long as the pinned knob does.
func TestBarterDiscount_DazzledLessThanComfortable(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.BarterMaxDiscount = 0.15
	configs.SetConfigForTest(t, cfg)
	maxFrac := float64(configs.GetBalanceConfig().BarterMaxDiscount)

	char := &characters.Character{Skills: map[string]int{string(skills.Bartering): 50}}

	comfortable := &rooms.Room{RoomId: 90001, SkyLight: rooms.SkyLightPtr(0.0), Lamp: rooms.LampPtr(60)}
	dazzled := &rooms.Room{RoomId: 90002, SkyLight: rooms.SkyLightPtr(0.0), Lamp: rooms.LampPtr(90)}

	comfortableDiscount := barterDiscount(char, comfortable, maxFrac)
	dazzledDiscount := barterDiscount(char, dazzled, maxFrac)

	assert.InDelta(t, 0.15, comfortableDiscount, 1e-9,
		"a comfortable haggler at skill 50 keeps the full 15% discount")
	assert.InDelta(t, 0.132, dazzledDiscount, 1e-9,
		"0.15 x 0.88 exactly: dazzle costs SightMult 0.88 at skill 50")
	assert.Less(t, dazzledDiscount, comfortableDiscount,
		"a dazzled haggler must bargain worse than a comfortable one")
}
