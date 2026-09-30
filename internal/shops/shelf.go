package shops

import (
	"slices"
	"time"
)

// The secondhand shelf (baubles slice D). AffixedStock holds unique items a
// shop bought from players and resells: affix-scaled gear, and average or
// rare baubles. An entry is LISTED (list shows it, buy sells it) unless it
// is HELD: a bauble still hot when shelved waits out of sight until
// HoldUntil. Listed entries are capped (Balance.ShopAffixedStockCap) and
// the cap is enforced lazily, on an add, on list and on buy. Held entries
// never count against that cap and are never evicted; a shop refuses a new
// hot bauble once it holds that many (BackroomFull).
//
// Every mutation happens in a command or a sale, under the mud lock, like
// the rest of ShopInventory (it has no lock of its own).

// ShelfNow is the one clock the shelf reads: list, buy, the sale that
// shelves an item and the held-count refusal all call it, so a test that
// sets it sees one consistent held state. A variable for tests.
var ShelfNow = time.Now

// Held reports whether the entry is still out of sight at now.
func (e AffixedStockEntry) Held(now time.Time) bool {
	return now.Before(e.HoldUntil)
}

// ListedAt is when the entry went on show: the end of its hold, or when it
// was shelved. An entry saved before these fields existed has both zero and
// so counts as listed earliest.
func (e AffixedStockEntry) ListedAt() time.Time {
	if !e.HoldUntil.IsZero() {
		return e.HoldUntil
	}
	return e.AddedAt
}

// HeldCount is how many entries are held at now.
func (si *ShopInventory) HeldCount(now time.Time) int {
	n := 0
	for _, e := range si.AffixedStock {
		if e.Held(now) {
			n++
		}
	}
	return n
}

// BackroomFull reports whether the shop must refuse to shelve an item whose
// hold would run until holdUntil (baubles.ShelfHoldUntil): it would be held
// at now, and the shop already holds limit entries (owner ruling 6). An item
// listed at once is never refused here, and limit <= 0 refuses nothing, as
// in EnforceAffixedCap. The one rule for every caller that shelves: the
// bauble sale refuses the offer, the auction's shopkeeper does not shelve.
func (si *ShopInventory) BackroomFull(holdUntil, now time.Time, limit int) bool {
	if limit <= 0 || !now.Before(holdUntil) {
		return false
	}
	return si.HeldCount(now) >= limit
}

// ListedIndexes returns the AffixedStock indexes of the entries listed at
// now, in slice order. This is THE shelf order: list renders it unsorted and
// buy counts `buy 2.name` in it.
func (si *ShopInventory) ListedIndexes(now time.Time) []int {
	out := make([]int, 0, len(si.AffixedStock))
	for i, e := range si.AffixedStock {
		if !e.Held(now) {
			out = append(out, i)
		}
	}
	return out
}

// EnforceAffixedCap removes listed entries while more than limit are listed
// at now, the one listed earliest (ListedAt) first, ties to the lower index,
// and returns how many it removed. limit <= 0 removes nothing. Held entries
// are never removed. A removed item is gone; a bauble's record keeps the
// sale that shelved it. A caller that changes a living shop saves it.
func (si *ShopInventory) EnforceAffixedCap(limit int, now time.Time) int {
	if limit <= 0 {
		return 0
	}
	removed := 0
	for {
		listed := si.ListedIndexes(now)
		if len(listed) <= limit {
			return removed
		}
		oldest := listed[0]
		for _, idx := range listed[1:] {
			if si.AffixedStock[idx].ListedAt().Before(si.AffixedStock[oldest].ListedAt()) {
				oldest = idx
			}
		}
		si.AffixedStock = slices.Delete(si.AffixedStock, oldest, oldest+1)
		removed++
	}
}

// RestoreAffixedStock puts an entry back at idx (clamped to the list) with
// its price, round and times unchanged: buy's rollback when the buyer cannot
// take the item after all.
func (si *ShopInventory) RestoreAffixedStock(idx int, e AffixedStockEntry) {
	idx = max(0, min(idx, len(si.AffixedStock)))
	si.AffixedStock = slices.Insert(si.AffixedStock, idx, e)
}
