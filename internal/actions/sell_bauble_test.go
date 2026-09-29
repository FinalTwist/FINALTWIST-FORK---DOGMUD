package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Bauble selling (docs/baubles Phase 2). These run against the same legacy
// merchant the other sell tests use, plus a registered living-economy shop
// where a test needs one.

// seedBaubleSale seeds the carrier (and the normal sell item the legacy
// merchant stocks) and points the bauble catalog at a temp dir.
func seedBaubleSale(t *testing.T) {
	t.Helper()
	restore := items.SeedItemsForTest(map[int]*items.ItemSpec{
		items.BaubleItemId: {
			ItemId:     items.BaubleItemId,
			Name:       "Curious Trinket",
			NameSimple: "trinket",
			Type:       items.Object,
			Subtype:    items.Mundane,
			Weight:     0.2,
			Value:      1,
			NotSalable: true,
		},
		sellTestItemId: {
			ItemId: sellTestItemId,
			Name:   "iron sword",
			Type:   items.Weapon,
			Value:  100,
		},
	})
	baubles.SetDirForTest(t.TempDir())
	t.Cleanup(func() {
		restore()
		items.SetBaubleResolver(nil)
	})
}

// newBauble creates a catalog record and the item that points at it.
func newBauble(t *testing.T, name string, noun string, value int, status baubles.Status) items.Item {
	t.Helper()
	rec, err := baubles.Create(baubles.Record{
		Name:       name,
		NameSimple: noun,
		Tier:       baubles.TierAverage,
		Value:      value,
		WeightLbs:  0.6,
		Status:     status,
		Source:     baubles.SourceSearch,
	})
	require.NoError(t, err)
	it := items.New(items.BaubleItemId)
	it.Bauble = rec.Id
	return it
}

func TestBaublePrice(t *testing.T) {
	// BuyRatio defaults to 0.50; rounded up, at least 1.
	assert.Equal(t, 1, BaublePrice(1))
	assert.Equal(t, 3, BaublePrice(5))
	assert.Equal(t, 6, BaublePrice(12))
	assert.Equal(t, 100, BaublePrice(200))
	assert.Equal(t, 1, BaublePrice(0))
}

func TestSell_Bauble_LegacyMerchantPaysCatalogValueAndDoesNotStock(t *testing.T) {
	seedBaubleSale(t)
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()

	seller := newSellerActor(t, true)
	char := seller.GetCharacter()
	b := newBauble(t, "Painted Wooden Horse", "horse", 12, baubles.StatusFallback)
	require.True(t, char.StoreItem(b))
	merchantBefore := merchantInstance().Character.Gold

	res := Sell(seller, SellOptions{ItemName: "horse", Quantity: 1})

	require.Equal(t, SellStopSoldAll, res.Reason, "res=%+v", res)
	assert.Equal(t, 1, res.Sold)
	assert.Equal(t, 6, char.Gold, "paid Value*BuyRatio from the catalog")
	assert.Equal(t, merchantBefore-6, merchantInstance().Character.Gold, "player sale drains the merchant")
	assert.Equal(t, "Painted Wooden Horse", res.LastItemName)
	_, still := char.FindInBackpack("horse")
	assert.False(t, still, "the bauble leaves the seller")
	assert.Equal(t, 0, shopQty(merchantInstance(), items.BaubleItemId), "the carrier is never stocked")

	rec, _ := baubles.Get(b.Bauble)
	assert.Equal(t, baubles.StatusSold, rec.Status)
	assert.Equal(t, 6, rec.SoldValue)
}

func TestSell_Bauble_ByGenericKeyword(t *testing.T) {
	seedBaubleSale(t)
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()

	seller := newSellerActor(t, true)
	char := seller.GetCharacter()
	require.True(t, char.StoreItem(newBauble(t, "Chipped Clay Cup", "cup", 4, baubles.StatusReady)))

	res := Sell(seller, SellOptions{ItemName: "bauble", Quantity: 1})
	assert.Equal(t, 1, res.Sold, "every bauble answers to `bauble`")
	assert.Equal(t, 2, char.Gold)
}

func TestSell_Bauble_UnknownRecordIsRefused(t *testing.T) {
	seedBaubleSale(t)
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()

	seller := newSellerActor(t, true)
	char := seller.GetCharacter()
	orphan := items.New(items.BaubleItemId)
	orphan.Bauble = "B9999999"
	require.True(t, char.StoreItem(orphan))

	res := Sell(seller, SellOptions{ItemName: "bauble", Quantity: 1})
	assert.Equal(t, 0, res.Sold)
	assert.Equal(t, 0, char.Gold)
}

func TestSell_Bauble_AllBaublesAreMixed(t *testing.T) {
	seedBaubleSale(t)
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()

	seller := newSellerActor(t, true)
	char := seller.GetCharacter()
	require.True(t, char.StoreItem(newBauble(t, "Tarnished Copper Button", "button", 2, baubles.StatusFallback)))
	require.True(t, char.StoreItem(newBauble(t, "Painted Wooden Horse", "horse", 12, baubles.StatusReady)))

	res := Sell(seller, SellOptions{ItemName: "bauble", Quantity: UnlimitedSell})

	assert.Equal(t, 2, res.Sold)
	assert.Equal(t, 1+6, res.TotalGold)
	assert.True(t, res.Mixed, "two different baubles must not be pluralised as one name")
}

func TestSell_SameItemTwiceIsNotMixed(t *testing.T) {
	seedBaubleSale(t)
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()

	seller := newSellerActor(t, true, sellTestItemId, sellTestItemId)
	res := Sell(seller, SellOptions{ItemName: "iron sword", Quantity: UnlimitedSell})
	assert.Equal(t, 2, res.Sold)
	assert.False(t, res.Mixed)
	assert.Equal(t, "iron sword", res.LastItemName)
}

func TestSell_Bauble_LivingShopByCraftSupport(t *testing.T) {
	seedBaubleSale(t)
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 0)()

	shops.ClearCache()
	_ = shops.RemoveShopFile("TestZone", 2, 1)
	defer shops.RemoveShopFile("TestZone", 2, 1)
	defer shops.ClearCache()

	si := shops.RegisterShop("TestZone", 2, 1, shops.ShopInventory{Gold: 1000})

	seller := newSellerActor(t, true)
	char := seller.GetCharacter()
	b := newBauble(t, "Painted Wooden Horse", "horse", 12, baubles.StatusReady)
	require.True(t, char.StoreItem(b))

	// A blacksmith does not buy trinkets.
	si.CraftSupport = shops.CraftSupportBlacksmithing
	res := Sell(seller, SellOptions{ItemName: "horse", Quantity: 1})
	assert.Equal(t, 0, res.Sold)
	assert.Equal(t, SellStopRejected, res.Reason)
	assert.Equal(t, 1000, si.Gold)

	// A general store does, from its own gold, and shelves nothing.
	si.CraftSupport = shops.CraftSupportGeneral
	res = Sell(seller, SellOptions{ItemName: "horse", Quantity: 1})
	require.Equal(t, 1, res.Sold, "res=%+v", res)
	assert.Equal(t, 6, char.Gold)
	assert.Equal(t, 994, si.Gold, "a living-economy sale drains the shop's gold")
	assert.Nil(t, si.GetStock(items.BaubleItemId), "the carrier is never stocked")
	assert.Len(t, si.AffixedStock, 0, "baubles are not resold like affixed loot")
}

func TestSell_Bauble_LivingShopReserveRefuses(t *testing.T) {
	seedBaubleSale(t)
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 0)()

	shops.ClearCache()
	_ = shops.RemoveShopFile("TestZone", 2, 1)
	defer shops.RemoveShopFile("TestZone", 2, 1)
	defer shops.ClearCache()

	// Starting gold 1000 keeps 500 in reserve; 503 on hand cannot pay 6.
	si := shops.RegisterShop("TestZone", 2, 1, shops.ShopInventory{Gold: 503, StartingGold: 1000, CraftSupport: shops.CraftSupportJewelcrafting})

	seller := newSellerActor(t, true)
	char := seller.GetCharacter()
	require.True(t, char.StoreItem(newBauble(t, "Painted Wooden Horse", "horse", 12, baubles.StatusReady)))

	res := Sell(seller, SellOptions{ItemName: "horse", Quantity: 1})
	assert.Equal(t, 0, res.Sold)
	assert.Equal(t, SellStopMerchantBroke, res.Reason)
	assert.Equal(t, 503, si.Gold)
}

func TestSell_Bauble_LegacyGetSellPriceNeverPricesIt(t *testing.T) {
	seedBaubleSale(t)
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()

	b := newBauble(t, "Painted Wooden Horse", "horse", 12, baubles.StatusReady)
	assert.Equal(t, 0, merchantInstance().GetSellPrice(b), "the ItemId-keyed path must never price a bauble")

	offer := BaubleOfferFrom(b, merchantInstance())
	assert.Equal(t, 6, offer.Price, "the bauble offer is the one to use")
}
