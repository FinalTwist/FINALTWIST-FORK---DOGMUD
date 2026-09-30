package baubles

import (
	"testing"
	"time"
)

// A buyback off a shop's shelf (baubles slice D) returns a sold record to
// its unsold status by Restore's rule (ruling 2), keeps the sale on the
// record, and leaves a record an admin retired meanwhile retired. The sale
// still counts in SalesSince.
func TestMarkBoughtReturnsTheUnsoldStatus(t *testing.T) {
	withCatalog(t)
	named := seedRecord(t, Record{Name: `Bone Dice`, NameSimple: `dice`, Tier: TierAverage, Value: 12, Status: StatusReady, Generator: GeneratorOpenAI})
	generic := seedRecord(t, Record{Name: `Trinket`, NameSimple: `trinket`, Tier: TierAverage, Value: 11, Status: StatusFallback, Generator: GeneratorLocal})
	retired := seedRecord(t, Record{Name: `Rude Name`, Tier: TierAverage, Value: 13, Status: StatusReady, Generator: GeneratorOpenAI})
	seedRecord(t, Record{Name: `Tin Cup`, NameSimple: `cup`, Tier: TierAverage, Value: 10, Status: StatusReady, Generator: GeneratorOpenAI}) // never sold: its SoldAt is zero

	start := time.Now().UTC().Add(-time.Second)
	for _, r := range []Record{named, generic, retired} {
		if !MarkSold(r.Id, 6, 42) {
			t.Fatalf("sell %s", r.Id)
		}
	}
	if err := Retire(retired.Id, `Admin`); err != nil {
		t.Fatal(err)
	}

	for id, want := range map[string]Status{named.Id: StatusReady, generic.Id: StatusFallback, retired.Id: StatusRetired} {
		if !MarkBought(id, 7) {
			t.Fatalf("buy back %s", id)
		}
		got, _ := Get(id)
		if got.Status != want {
			t.Errorf("%s bought back: status %s, want %s", id, got.Status, want)
		}
		if got.SoldValue != 6 || got.SoldAt.IsZero() {
			t.Errorf("%s: the sale stays on the record: %+v", id, got)
		}
	}
	if MarkBought(`B0000404`, 7) {
		t.Fatal("no such record")
	}
	if n, gold := SalesSince(start); n != 3 || gold != 18 {
		t.Fatalf("a buyback does not erase a sale: %d sales, %d gold, want 3 and 18", n, gold)
	}
	if n, _ := SalesSince(time.Time{}); n != 3 {
		t.Fatalf("a record never sold (the tin cup) is not a sale: %d, want 3", n)
	}
}

// unsoldStatus is Restore's rule, shared (ruling 2).
func TestUnsoldStatusIsRestoresRule(t *testing.T) {
	for g, want := range map[Generator]Status{
		GeneratorOpenAI: StatusReady, GeneratorCorpus: StatusReady,
		GeneratorLocal: StatusFallback, GeneratorAdmin: StatusFallback, ``: StatusFallback,
	} {
		if got := (Record{Generator: g}).unsoldStatus(); got != want {
			t.Errorf("generator %q: %s, want %s", g, got, want)
		}
	}
}
