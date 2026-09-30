package baubles

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// Heat (docs/baubles Phase 6c): hot for BaubleStolenHeatHours after the
// latest theft, cooled by a return, hot again after a fresh theft.
func TestStolenBaubleHeat(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	setBaubleConfig(t, func(b *configs.Balance) { b.BaubleStolenHeatHours = 72 })

	rec, err := Create(Record{Name: "Tarnished Brass Thimble", NameSimple: "thimble", Tier: TierCheap, Value: 4, Status: StatusReady})
	if err != nil {
		t.Fatal(err)
	}
	t0 := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	week := 72 * time.Hour // the heat, three days
	get := func() Record { r, _ := Get(rec.Id); return r }

	if get().Hot(t0) {
		t.Fatal("an honest bauble is never hot")
	}
	MarkStolen(rec.Id, Theft{ByUserId: 1, FromMob: 2, Zone: "Thornwall City"}, t0)
	if !get().Hot(t0) || !get().Hot(t0.Add(week-time.Second)) {
		t.Fatal("hot for three days after the theft")
	}
	if get().Hot(t0.Add(week)) {
		t.Fatal("cold once the three days are out")
	}
	if get().StolenZone != "Thornwall City" {
		t.Fatal("the theft's zone is kept")
	}

	it := items.New(items.BaubleItemId)
	it.Bauble = rec.Id
	if !ItemIsHotIn(it, "Thornwall City", t0) || ItemIsHotIn(items.New(1), "Thornwall City", t0) {
		t.Fatal("ItemIsHotIn follows the record, and only for baubles")
	}
	if ItemIsHotIn(it, "Greenford", t0) {
		t.Fatal("hot only in the area it was stolen in")
	}
	if ItemIsHotIn(it, "Thornwall City", t0.Add(week)) {
		t.Fatal("and not even there once it cools")
	}

	if !get().StolenGoods() {
		t.Fatal("stolen goods until given back")
	}
	MarkReturned(rec.Id, 1, nil, t0.Add(time.Hour))
	if get().Hot(t0.Add(2*time.Hour)) || get().StolenGoods() {
		t.Fatal("given back, it cools at once and is no longer stolen goods")
	}
	MarkStolen(rec.Id, Theft{ByUserId: 1, FromMob: 2}, t0.Add(3*time.Hour))
	if !get().Hot(t0.Add(4*time.Hour)) || !get().StolenGoods() {
		t.Fatal("stolen again, hot again")
	}

	if get().RecognizedSinceTheft() {
		t.Fatal("not recognised yet")
	}
	MarkRecognized(rec.Id, 1, t0.Add(4*time.Hour))
	if !get().RecognizedSinceTheft() {
		t.Fatal("recognised since this theft")
	}
	MarkStolen(rec.Id, Theft{ByUserId: 1, FromMob: 2}, t0.Add(5*time.Hour))
	if get().RecognizedSinceTheft() {
		t.Fatal("a fresh theft can be recognised afresh")
	}
}

// Zones grouped in BaubleHeatAreas are one area; any other zone is its own.
func TestHeatAreas(t *testing.T) {
	setBaubleConfig(t, func(b *configs.Balance) {
		b.BaubleHeatAreas = map[string][]string{`New Plymouth`: {`New Plymouth Docks`, ` New Plymouth Merchant `}}
	})
	if HeatArea(`New Plymouth Docks`) != HeatArea(`new plymouth merchant`) {
		t.Fatal("two quarters of one city are one area")
	}
	if HeatArea(`New Plymouth Outskirts`) == HeatArea(`New Plymouth Docks`) {
		t.Fatal("a zone not listed is its own area")
	}
	if HeatArea(`Greenford`) != HeatArea(`greenford`) || HeatArea(`Greenford`) == HeatArea(`Stillwater`) {
		t.Fatal("an unlisted zone is its own area, whatever its case")
	}
	if HeatArea(`New Plymouth`) == HeatArea(`New Plymouth Docks`) {
		t.Fatal("an area's own name is not one of its zones")
	}
}

// A bauble's return earns credit once ever; ReturnCredits counts a thief's
// credited returns per faction and reports the latest.
func TestReturnCreditsAreOncePerBaubleAndPerFaction(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	t0 := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

	mk := func() string {
		r, err := Create(Record{Name: "Bone Dice", NameSimple: "dice", Tier: TierCheap, Value: 4, Status: StatusReady})
		if err != nil {
			t.Fatal(err)
		}
		MarkStolen(r.Id, Theft{ByUserId: 1, FromMob: 2}, t0)
		return r.Id
	}
	a, b := mk(), mk()
	MarkReturned(a, 1, []string{`town`, `guild`}, t0.Add(time.Hour))
	MarkReturned(b, 1, []string{`town`}, t0.Add(2*time.Hour))
	MarkReturned(a, 1, []string{`town`}, t0.Add(3*time.Hour)) // again: no new credit

	if n := ReturnCredits(1, `town`, 0); n != 2 {
		t.Fatalf("town: %d credits", n)
	}
	if n := ReturnCredits(1, `guild`, 0); n != 1 {
		t.Fatalf("guild: %d credits", n)
	}
	if n := ReturnCredits(9, `town`, 0); n != 0 {
		t.Fatal("another player's credits are not theirs")
	}
	r, _ := Get(a)
	if !r.ReturnedAt.Equal(t0.Add(3*time.Hour)) || !r.ReturnCreditAt.Equal(t0.Add(time.Hour)) {
		t.Fatalf("the latest return is kept, the first credit too: %+v", r)
	}
}

// A gift to a mob is remembered by mob id (MarkGiven) until the next theft.
func TestAGiftIsRememberedUntilTheNextTheft(t *testing.T) {
	SetDirForTest(t.TempDir())
	rec, err := Create(Record{Name: "Bone Dice", NameSimple: "dice", Tier: TierCheap, Value: 4, Status: StatusReady})
	if err != nil {
		t.Fatal(err)
	}
	if !MarkGiven(rec.Id, 12) {
		t.Fatal("MarkGiven")
	}
	r, _ := Get(rec.Id)
	if !r.GivenTo(12) || r.GivenTo(13) || r.GivenTo(0) {
		t.Fatalf("given to mob 12 only: %+v", r)
	}
	MarkStolen(rec.Id, Theft{ByUserId: 1, FromMob: 13}, time.Now())
	if r, _ = Get(rec.Id); r.GivenTo(12) {
		t.Fatal("a theft ends the gift")
	}
	if MarkGiven("B9999999", 12) {
		t.Fatal("no such record")
	}
}

// A bauble shelved while hot anywhere (Hot, not HotIn: its buyer could carry
// it back into the theft's area) is held until its heat ends, StolenAt plus
// the heat; anything else is listed at once (baubles slice D).
func TestShelfHoldUntilIsTheEndOfTheHeat(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	setBaubleConfig(t, func(b *configs.Balance) { b.BaubleStolenHeatHours = 72 })

	rec, err := Create(Record{Name: "Bone Dice", NameSimple: "dice", Tier: TierAverage, Value: 12, Status: StatusReady})
	if err != nil {
		t.Fatal(err)
	}
	it := items.Item{ItemId: items.BaubleItemId, Bauble: rec.Id}
	t0 := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

	if !ShelfHoldUntil(it, t0).IsZero() {
		t.Fatal("an honest bauble is listed at once")
	}
	MarkStolen(rec.Id, Theft{ByUserId: 1, FromMob: 2, Zone: "Thornwall City"}, t0)
	if got := ShelfHoldUntil(it, t0.Add(time.Hour)); !got.Equal(t0.Add(72 * time.Hour)) {
		t.Fatalf("hot: held until %v, want %v", got, t0.Add(72*time.Hour))
	}
	if !ShelfHoldUntil(it, t0.Add(72*time.Hour)).IsZero() {
		t.Fatal("cold once the heat is out")
	}
	MarkReturned(rec.Id, 1, nil, t0.Add(2*time.Hour))
	if !ShelfHoldUntil(it, t0.Add(3*time.Hour)).IsZero() {
		t.Fatal("given back, it is not hot, so not held")
	}
	if !ShelfHoldUntil(items.Item{ItemId: 5}, t0).IsZero() {
		t.Fatal("anything that is not a bauble is listed at once")
	}
}
