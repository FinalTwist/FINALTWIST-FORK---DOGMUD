package shops

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/items"
	"gopkg.in/yaml.v2"
)

// The secondhand shelf (baubles slice D): held entries, the listed cap and
// its eviction order, the rollback reinsert, and the saved times.

var shelfT0 = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

// shelfEntry is a piece of affixed gear shelved at addedAt and held until
// holdUntil (zero: listed at once). Its ItemId names it in assertions.
func shelfEntry(id int, addedAt, holdUntil time.Time) AffixedStockEntry {
	return AffixedStockEntry{
		Item:      items.Item{ItemId: id, Affixed: true, Spec: &items.ItemSpec{Value: 100, Name: "piece"}},
		Price:     100,
		AddedAt:   addedAt,
		HoldUntil: holdUntil,
	}
}

func shelfIds(si *ShopInventory) []int {
	out := []int{}
	for _, e := range si.AffixedStock {
		out = append(out, e.Item.ItemId)
	}
	return out
}

// Spec tests 1 to 3: a held entry is not listed, counts toward HeldCount
// only, and is never evicted; the listed cap counts listed entries only. A
// hold that ends lists the entry.
func TestShelf_HeldEntriesAreNotListedNorEvicted(t *testing.T) {
	now := shelfT0.Add(time.Hour)
	si := &ShopInventory{AffixedStock: []AffixedStockEntry{
		shelfEntry(1, shelfT0, time.Time{}),
		shelfEntry(2, shelfT0, shelfT0.Add(72*time.Hour)),
		shelfEntry(3, shelfT0.Add(time.Minute), time.Time{}),
		shelfEntry(4, shelfT0, shelfT0.Add(24*time.Hour)),
	}}

	if got := si.ListedIndexes(now); !reflect.DeepEqual(got, []int{0, 2}) {
		t.Fatalf("listed: got %v, want [0 2] (the held ones are out of sight)", got)
	}
	if got := si.HeldCount(now); got != 2 {
		t.Fatalf("held: got %d, want 2", got)
	}
	if removed := si.EnforceAffixedCap(0, now); removed != 0 || len(si.AffixedStock) != 4 {
		t.Fatalf("a cap of 0 removes nothing: removed %d, left %v", removed, shelfIds(si))
	}
	if removed := si.EnforceAffixedCap(1, now); removed != 1 {
		t.Fatalf("cap 1 over two listed entries: removed %d, want 1", removed)
	}
	if got := shelfIds(si); !reflect.DeepEqual(got, []int{2, 3, 4}) {
		t.Fatalf("the earlier listed entry goes and both held ones stay: got %v, want [2 3 4]", got)
	}
	if removed := si.EnforceAffixedCap(1, now); removed != 0 {
		t.Fatalf("at the cap nothing more goes: removed %d", removed)
	}

	// Spec test 2: item 4's hold ends; it lists, after item 3, in slice order.
	later := shelfT0.Add(24 * time.Hour)
	if got := si.ListedIndexes(later); !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("hold over: got %v, want [1 2]", got)
	}
	if si.AffixedStock[2].Held(later) || !si.AffixedStock[2].Held(later.Add(-time.Second)) {
		t.Fatal("held strictly before HoldUntil, listed from it on")
	}
}

// Spec test 4: over the cap, the entry with the earliest ListedAt goes. An
// entry shelved first but held until later outlives an unheld entry shelved
// after it. Entries saved before AddedAt existed (both times zero) list
// earliest, and a tie goes to the lower index.
func TestShelf_EvictsTheEntryListedEarliest(t *testing.T) {
	si := &ShopInventory{AffixedStock: []AffixedStockEntry{
		shelfEntry(1, shelfT0, shelfT0.Add(2*time.Hour)), // shelved first, listed at T0+2h
		shelfEntry(2, shelfT0.Add(time.Minute), time.Time{}),
		shelfEntry(3, shelfT0.Add(2*time.Minute), time.Time{}),
	}}
	if removed := si.EnforceAffixedCap(2, shelfT0.Add(3*time.Hour)); removed != 1 {
		t.Fatalf("removed %d, want 1", removed)
	}
	if got := shelfIds(si); !reflect.DeepEqual(got, []int{1, 3}) {
		t.Fatalf("item 2 listed earliest and goes, item 1 was held and stays: got %v, want [1 3]", got)
	}

	old := &ShopInventory{AffixedStock: []AffixedStockEntry{
		shelfEntry(7, time.Time{}, time.Time{}),
		shelfEntry(8, time.Time{}, time.Time{}),
		shelfEntry(9, shelfT0, time.Time{}),
	}}
	old.EnforceAffixedCap(2, shelfT0)
	if got := shelfIds(old); !reflect.DeepEqual(got, []int{8, 9}) {
		t.Fatalf("pre-change entries list earliest, ties to the lower index: got %v, want [8 9]", got)
	}
}

// Spec test 14: RestoreAffixedStock puts an entry back at its index with its
// fields unchanged; an index out of range is clamped to the ends.
func TestShelf_RestoreAffixedStockPutsTheEntryBackUnchanged(t *testing.T) {
	a := shelfEntry(1, shelfT0, time.Time{})
	b := shelfEntry(2, shelfT0.Add(time.Minute), shelfT0.Add(time.Hour))
	b.Price, b.AddedRound = 37, 99
	c := shelfEntry(3, shelfT0, time.Time{})
	si := &ShopInventory{AffixedStock: []AffixedStockEntry{a, b, c}}

	if _, ok := si.RemoveAffixedStock(1); !ok {
		t.Fatal("remove")
	}
	si.RestoreAffixedStock(1, b)
	if got := shelfIds(si); !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Fatalf("back in its place: got %v", got)
	}
	got := si.AffixedStock[1]
	if got.Price != 37 || got.AddedRound != 99 || !got.AddedAt.Equal(b.AddedAt) || !got.HoldUntil.Equal(b.HoldUntil) {
		t.Fatalf("fields changed: %+v", got)
	}

	si.RestoreAffixedStock(99, shelfEntry(4, shelfT0, time.Time{}))
	si.RestoreAffixedStock(-1, shelfEntry(5, shelfT0, time.Time{}))
	if got := shelfIds(si); !reflect.DeepEqual(got, []int{5, 1, 2, 3, 4}) {
		t.Fatalf("out-of-range indexes clamp to the ends: got %v", got)
	}
}

// AddedAt and HoldUntil survive the shop save (yaml.v2, as SaveShop uses),
// zero times are omitted, and a shop file written before they existed loads
// with both zero: listed.
func TestShelf_TimesRoundTripAndOldFilesLoadListed(t *testing.T) {
	in := ShopInventory{AffixedStock: []AffixedStockEntry{shelfEntry(5, shelfT0, shelfT0.Add(72*time.Hour))}}
	data, err := yaml.Marshal(&in)
	if err != nil {
		t.Fatal(err)
	}
	var out ShopInventory
	if err := yaml.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	e := out.AffixedStock[0]
	if !e.AddedAt.Equal(shelfT0) || !e.HoldUntil.Equal(shelfT0.Add(72*time.Hour)) {
		t.Fatalf("times lost in the round trip: %+v\n%s", e, data)
	}

	listed := ShopInventory{AffixedStock: []AffixedStockEntry{shelfEntry(6, time.Time{}, time.Time{})}}
	raw, err := yaml.Marshal(&listed)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "hold_until") || strings.Contains(string(raw), "added_at") {
		t.Fatalf("zero times must be omitted:\n%s", raw)
	}

	old := "gold: 10\naffixed_stock:\n- item:\n    itemid: 5\n  price: 7\n  added_round: 3\n"
	var loaded ShopInventory
	if err := yaml.Unmarshal([]byte(old), &loaded); err != nil {
		t.Fatal(err)
	}
	le := loaded.AffixedStock[0]
	if !le.AddedAt.IsZero() || !le.HoldUntil.IsZero() || le.Held(shelfT0) || len(loaded.ListedIndexes(shelfT0)) != 1 {
		t.Fatalf("an old entry loads listed with zero times: %+v", le)
	}
}
