package aicompanion

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

// Buying and selling (F10.1 to F10.8). The companion learns what a shop
// sells, and at what price, the way a player does: by looking at its stock
// (`browse`, the equivalent of `list`). It remembers what it saw, compares
// shops, keeps an emergency reserve, and buys and sells through the
// ordinary mob buy and sell commands. Every purchase and sale is checked
// afterwards against its purse and pack.

// ShopRecord is what the companion remembers about one merchant's shop.
type ShopRecord struct {
	Merchant string              `yaml:"merchant"`
	RoomId   int                 `yaml:"room"`
	SeenUnix int64               `yaml:"seen"`
	Wares    map[int]*WareRecord `yaml:"wares,omitempty"`
}

// WareRecord is one item as last seen in a shop, and what the shop paid
// the companion for one, if it ever sold one there.
type WareRecord struct {
	Name     string `yaml:"name"`
	Price    int    `yaml:"price"`
	Qty      int    `yaml:"qty,omitempty"`
	SoldFor  int    `yaml:"sold_for,omitempty"`
	SeenUnix int64  `yaml:"seen"`
}

// Purse is the authored money habit of a companion (F10.5).
type Purse struct {
	Reserve int    `yaml:"reserve"` // never spent except on needs or when asked
	Style   string `yaml:"style"`   // thrifty, ordinary, free
}

// largeShare is the fraction of the purse above which a purchase counts as
// large and needs the owner's say-so.
func (p Purse) largeShare() float64 {
	switch p.Style {
	case `thrifty`:
		return 0.33
	case `free`:
		return 0.75
	}
	return 0.5
}

// ware is one item a merchant in the room has for sale right now.
type ware struct {
	ItemId int
	Name   string
	Price  int
	Qty    int
}

// shopListing is one merchant's stock as a player's `list` would show it.
type shopListing struct {
	MerchantName       string
	MerchantMobId      int
	MerchantInstanceId int
	Wares              []ware
}

// browseShops reads every awake merchant's stock in the room, priced the
// same way usercommands/list.go prices it.
func browseShops(room *rooms.Room) []shopListing {
	var out []shopListing
	cfg := shops.PricingConfigFromBalance()
	for _, id := range room.GetMobs(rooms.FindMerchant) {
		m := mobs.GetInstance(id)
		if m == nil || actions.TargetAsleep(&m.Character) {
			continue
		}
		l := shopListing{MerchantName: m.Character.Name, MerchantMobId: int(m.MobId), MerchantInstanceId: id}
		if inv := shops.GetShopInventory(m.Zone, int(m.MobId), m.HomeRoomId); inv != nil {
			for _, entry := range inv.Stock {
				if entry.Current <= 0 {
					continue
				}
				itm := items.New(entry.ItemId)
				if itm.ItemId == 0 {
					continue
				}
				spec := itm.GetSpec()
				restock := shops.PricingBaseline(&entry, cfg)
				price := shops.CalcSellPrice(spec.Value, entry.Current, restock, cfg)
				l.Wares = append(l.Wares, ware{ItemId: entry.ItemId, Name: itm.Name(), Price: price, Qty: entry.Current})
			}
		} else {
			for _, si := range m.Character.Shop.GetInstock() {
				if si.ItemId == 0 {
					continue
				}
				itm := items.New(si.ItemId)
				if itm.ItemId == 0 {
					continue
				}
				price := si.Price
				if price <= 0 {
					price = itm.GetSpec().Value
				}
				l.Wares = append(l.Wares, ware{ItemId: si.ItemId, Name: itm.Name(), Price: price, Qty: si.Quantity})
			}
		}
		sort.Slice(l.Wares, func(i, j int) bool { return l.Wares[i].Name < l.Wares[j].Name })
		out = append(out, l)
	}
	return out
}

// rememberShop records a listing in the companion's shop memory (F10.3).
func (m *Mind) rememberShop(l shopListing, roomId int, nowUnix int64) {
	if m.Shops == nil {
		m.Shops = map[int]*ShopRecord{}
	}
	rec, ok := m.Shops[l.MerchantMobId]
	if !ok {
		rec = &ShopRecord{}
		m.Shops[l.MerchantMobId] = rec
	}
	rec.Merchant, rec.RoomId, rec.SeenUnix = l.MerchantName, roomId, nowUnix
	old := rec.Wares
	rec.Wares = map[int]*WareRecord{}
	for _, w := range l.Wares {
		wr := &WareRecord{Name: w.Name, Price: w.Price, Qty: w.Qty, SeenUnix: nowUnix}
		if prev, ok := old[w.ItemId]; ok {
			wr.SoldFor = prev.SoldFor
		}
		rec.Wares[w.ItemId] = wr
	}
	// Remember what it sold here even for items no longer on the shelf.
	for id, prev := range old {
		if _, still := rec.Wares[id]; !still && prev.SoldFor > 0 {
			prev.Qty = 0
			rec.Wares[id] = prev
		}
	}
}

// describeListing renders a browse result as the companion would recall it.
func describeListing(l shopListing, refs map[int]string) string {
	if len(l.Wares) == 0 {
		return fmt.Sprintf(`%s has nothing for sale right now.`, l.MerchantName)
	}
	var parts []string
	for _, w := range l.Wares {
		ref := ``
		if r, ok := refs[w.ItemId]; ok {
			ref = `[` + r + `] `
		}
		parts = append(parts, fmt.Sprintf(`%s%s for %d gold`, ref, w.Name, w.Price))
	}
	return fmt.Sprintf(`%s sells: %s.`, l.MerchantName, strings.Join(parts, `; `))
}

// cheapestKnown finds the cheapest remembered ware matching a supply.
func cheapestKnown(mind *Mind, s Supply) (rec *ShopRecord, w *WareRecord, itemId int) {
	for _, sr := range mind.Shops {
		for id, wr := range sr.Wares {
			if wr.Qty <= 0 || wr.Price <= 0 {
				continue
			}
			itm := items.New(id)
			if itm.ItemId == 0 || !s.matches(&itm) {
				continue
			}
			if w == nil || wr.Price < w.Price {
				rec, w, itemId = sr, wr, id
			}
		}
	}
	return rec, w, itemId
}

// priceLines tells the model where it last saw what it needs, and for how
// much (F10.4).
func priceLines(mob *mobs.Mob, p *Profile, mind *Mind, nowUnix int64) []string {
	var out []string
	for _, n := range supplyNeeds(mob, p.Supplies) {
		rec, w, _ := cheapestKnown(mind, n.Supply)
		if w == nil {
			out = append(out, fmt.Sprintf(`You do not know anywhere that sells %s.`, n.Supply.Name))
			continue
		}
		out = append(out, fmt.Sprintf(`%s: the cheapest you know is %s at %d gold from %s ([%s], seen %s ago).`,
			n.Supply.Name, w.Name, w.Price, rec.Merchant, placeRef(rec.RoomId), humanizeElapsed(nowUnix-w.SeenUnix)))
	}
	return out
}

// purchaseCheck applies the money rules to a purchase of qty at price:
// never more than the purse; not into the reserve, and nothing large, unless
// it meets a need or the owner asked (F10.5, F10.8).
func purchaseCheck(gold int, price int, qty int, purse Purse, meetsNeed bool, ownerAsked bool) string {
	cost := price * qty
	switch {
	case price <= 0 || qty <= 0:
		return `nothing to pay for`
	case cost > gold:
		return `cannot afford it`
	case meetsNeed || ownerAsked:
		return ``
	case gold-cost < purse.Reserve:
		return `it would dip into your emergency money`
	case float64(cost) > float64(gold)*purse.largeShare():
		return `too dear to buy without talking it over first`
	}
	return ``
}

// browseAction answers a browse: it reads the stock, remembers it, and
// gives the result back as a follow-up.
func (m *AICompanionModule) browseAction(c *controller, mob *mobs.Mob, room *rooms.Room) actionOutcome {
	listings := browseShops(room)
	if len(listings) == 0 {
		return actionOutcome{Refused: `no merchant here is open`}
	}
	now := time.Now().Unix()
	var parts []string
	for _, l := range listings {
		c.mind.rememberShop(l, room.RoomId, now)
		parts = append(parts, describeListing(l, nil))
		c.mind.recordInteraction(fmt.Sprintf(`npc:%d`, l.MerchantMobId), `browse`, true, now)
	}
	mob.Command(`emote looks over what is for sale.`)
	c.mind.addLine(Line{Kind: `event`, Text: `You looked over what was for sale.`}, m.cfg.WorkingMemoryLines)
	c.dirty = true
	return actionOutcome{Perceived: `You looked over what is for sale here. ` + strings.Join(parts, ` `) +
		` (The wares now appear among the things here with [s] refs.)`}
}
