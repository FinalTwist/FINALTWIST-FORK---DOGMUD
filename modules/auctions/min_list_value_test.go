package auctions

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func TestTooTrivialToAuction(t *testing.T) {
	defer func(old int) { auctionMinListValue = old }(auctionMinListValue)
	auctionMinListValue = 100

	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		1: {ItemId: 1, Name: "trinket", Value: 50},   // below floor
		2: {ItemId: 2, Name: "at floor", Value: 100}, // exactly at floor -> listable
		3: {ItemId: 3, Name: "prize", Value: 500},    // above floor
	})
	defer cleanup()

	if !tooTrivialToAuction(items.New(1)) {
		t.Error("value 50 < floor 100 should be too trivial to auction")
	}
	if tooTrivialToAuction(items.New(2)) {
		t.Error("value 100 == floor should be listable")
	}
	if tooTrivialToAuction(items.New(3)) {
		t.Error("value 500 should be listable")
	}
}

func TestTooTrivialToAuction_FloorDisabled(t *testing.T) {
	defer func(old int) { auctionMinListValue = old }(auctionMinListValue)
	auctionMinListValue = 0 // floor off -> everything listable

	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		1: {ItemId: 1, Name: "trinket", Value: 1},
	})
	defer cleanup()

	if tooTrivialToAuction(items.New(1)) {
		t.Error("with the floor at 0, even a 1g item should be listable")
	}
}

// A hot stolen bauble (docs/baubles Phase 6c) is refused; a cooled one, an
// honest one and anything that is not a bauble are not.
func TestAuctionRefusesAHotStolenBauble(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false) // the catalog logs each theft
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		items.BaubleItemId: {ItemId: items.BaubleItemId, Name: "Curious Trinket", NameSimple: "trinket", Value: 1},
		1:                  {ItemId: 1, Name: "prize", Value: 500},
	})
	defer cleanup()
	baubles.SetDirForTest(t.TempDir())
	defer items.SetBaubleResolver(nil)

	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	defer func(old func() time.Time) { auctionClock = old }(auctionClock)
	auctionClock = func() time.Time { return now }

	bauble := func(stolenAt time.Time) items.Item {
		rec, err := baubles.Create(baubles.Record{Name: "Silver Reliquary", NameSimple: "reliquary", Tier: baubles.TierRare, Value: 150, Status: baubles.StatusReady})
		if err != nil {
			t.Fatal(err)
		}
		if !stolenAt.IsZero() {
			baubles.MarkStolen(rec.Id, baubles.Theft{ByUserId: 1, FromMob: 2, Zone: "Thornwall City"}, stolenAt)
		}
		it := items.New(items.BaubleItemId)
		it.Bauble = rec.Id
		return it
	}

	const here = "Thornwall City"
	if !auctionRefusesStolen(bauble(now.Add(-time.Hour)), here) {
		t.Error("a stolen bauble hot here is refused")
	}
	if auctionRefusesStolen(bauble(now.Add(-time.Hour)), "Greenford") {
		t.Error("listed from another town it is just a trinket")
	}
	if auctionRefusesStolen(bauble(now.Add(-8*24*time.Hour)), here) {
		t.Error("a cooled stolen bauble may be listed")
	}
	if auctionRefusesStolen(bauble(time.Time{}), here) || auctionRefusesStolen(items.New(1), here) {
		t.Error("honest goods may be listed")
	}
}
