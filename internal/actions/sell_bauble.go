package actions

import (
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

// Selling baubles (docs/baubles, Phase 2).
//
// A bauble is item 900 plus a catalog id, so none of the ItemId-keyed sale
// machinery can price it or stock it: EvaluateBuyRules would refuse it (the
// carrier has no vendor_categories), and the legacy path would stock item 900
// and resell it as a generic "Curious Trinket". Baubles therefore take their
// own branch, like affixed loot does: priced from the catalog value times the
// shop buy ratio, never added to stock, and the record marked sold.

// baubleShopBuys reports whether a living-economy shop buys baubles: its
// craft_support is listed in Balance.BaubleBuyerCraftSupports (by default
// general stores and jewellers). Legacy merchants, which have no
// ShopInventory, all buy them; they are general traders by construction.
func baubleShopBuys(shopInv *shops.ShopInventory) bool {
	for _, cs := range configs.GetBalanceConfig().BaubleBuyerCraftSupports {
		if strings.EqualFold(strings.TrimSpace(cs), shopInv.CraftSupport) {
			return true
		}
	}
	return false
}

// Merchant lines for bauble refusals.
const (
	baubleSayUnknown    = "I'm afraid I don't buy those."
	baubleSayNotBuyer   = "I'm not interested in trinkets. Try a general store or a jeweller."
	baubleSayCantAfford = "I can't afford that right now."
)

// BaubleOffer is what one merchant would pay for one bauble. Price is 0 when
// the merchant will not buy it, and Refusal is the line it says instead.
// Broke marks a refusal for lack of gold rather than lack of interest.
type BaubleOffer struct {
	Price   int
	Refusal string
	Broke   bool
}

// BaublePrice is the gold a merchant pays for a bauble worth value: the
// catalog value times the shop buy ratio, rounded up, at least 1. Same
// spread as affixed loot, and like it, no scarcity curve and no barter bonus.
func BaublePrice(value int) int {
	price := int(math.Ceil(float64(value) * shops.PricingConfigFromBalance().BuyRatio))
	if price < 1 {
		price = 1
	}
	return price
}

// baubleOfferFor decides what this merchant would pay for this bauble.
// shopInv is the merchant's living-economy shop, nil for a legacy merchant.
// The gold check here is the living-economy reserve (the same one
// EvaluateBuyRules applies); whether the merchant has the gold at all is
// checked at the sale, as for every other item.
func baubleOfferFor(item items.Item, shopInv *shops.ShopInventory) BaubleOffer {
	rec, ok := baubles.Get(item.Bauble)
	if !ok {
		return BaubleOffer{Refusal: baubleSayUnknown}
	}
	if shopInv != nil && !baubleShopBuys(shopInv) {
		return BaubleOffer{Refusal: baubleSayNotBuyer}
	}

	price := BaublePrice(rec.Value)

	if shopInv != nil {
		ratio := float64(configs.GetBalanceConfig().ShopGoldReserveRatio)
		if ratio <= 0 {
			ratio = 0.50
		}
		if !shopInv.CanAfford(price, shopInv.GoldReserve(ratio)) {
			return BaubleOffer{Refusal: baubleSayCantAfford, Broke: true}
		}
	}
	return BaubleOffer{Price: price}
}

// BaubleOfferFrom is baubleOfferFor for a merchant mob, resolving its shop.
// Used by the offer and appraise commands.
func BaubleOfferFrom(item items.Item, mob *mobs.Mob) BaubleOffer {
	if mob == nil || !item.IsBauble() {
		return BaubleOffer{Refusal: baubleSayUnknown}
	}
	shopInv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)
	return baubleOfferFor(item, shopInv)
}

// sellBaubleToMerchant is sellOneToMerchant's branch for a bauble. item has
// already been found in the seller's inventory and is known to be a bauble.
func sellBaubleToMerchant(seller Actor, item items.Item, room *rooms.Room,
	mob *mobs.Mob, shopInv *shops.ShopInventory,
	awardProgression bool) (soldValue int, res SellStopReason) {

	char := seller.GetCharacter()

	offer := baubleOfferFor(item, shopInv)
	if offer.Price <= 0 {
		merchantSay(room, mob, offer.Refusal)
		if offer.Broke {
			return 0, SellStopMerchantBroke
		}
		return 0, SellStopRejected
	}
	sellValue := offer.Price

	// Gold-model gate, exactly as for every other item: only players are
	// constrained by, and draw down, the merchant's gold.
	if seller.IsPlayer() {
		merchantGold := mob.Character.Gold
		if shopInv != nil {
			merchantGold = shopInv.Gold
		}
		if merchantGold < sellValue {
			merchantSay(room, mob, baubleSayCantAfford)
			return 0, SellStopMerchantBroke
		}
		if shopInv != nil {
			shopInv.Gold -= sellValue
		} else {
			mob.Character.Gold -= sellValue
		}
	}

	char.Gold += sellValue
	char.RemoveItem(item)

	if seller.IsPlayer() {
		events.AddToQueue(events.ItemOwnership{UserId: seller.GetUserId(), Item: item, Gained: false})
		events.AddToQueue(events.EquipmentChange{UserId: seller.GetUserId(), GoldChange: sellValue})
	} else {
		events.AddToQueue(events.ItemOwnership{MobInstanceId: seller.GetMobInstanceId(), Item: item, Gained: false})
	}

	// The bauble is not stocked: it leaves the world, and its gold is the
	// only trace. The shop's gold changed, so a living-economy shop is saved.
	if shopInv != nil {
		shopInv.BuysCount++
		if err := shops.SaveShop(shopInv.Zone, shopInv.MobId, shopInv.RoomId); err != nil {
			mudlog.Error("SELL", "msg", "SaveShop failed", "error", err)
		}
	}
	baubles.MarkSold(item.Bauble, sellValue, seller.GetUserId())

	// Progression: first sale of the command only, as for every other item
	// (see sellOneToMerchant).
	if awardProgression {
		saleProgression(seller, mob)
	}

	return sellValue, SellStopSoldAll
}
