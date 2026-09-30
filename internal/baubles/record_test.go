package baubles

import (
	"strings"
	"testing"
)

// A finder-only record (owner ruling 2026-09-29) shows the generic trinket
// to everyone and its own text to its finder alone; its value and weight
// are the item's for everyone. Retired text wins over both. Finder-only is
// DERIVED (KeptToFinder: PlayerKey and not Moderated), never stored, so a
// record written before this slice, which has no such field, is kept to its
// finder too (review finding a).
func TestFinderOnlyRecordView(t *testing.T) {
	// Exactly the shape the pre-slice moderate left: a player's own key,
	// accepted unmoderated.
	r := Record{Id: `b0000001`, Status: StatusReady, Name: `Painted Wooden Horse`, NameSimple: `horse`,
		Description: `A child's toy horse, its red paint flaking.`, Material: `pine`, Value: 12, WeightLbs: 0.3,
		FoundByUserId: 7, PlayerKey: true, Moderated: false}
	if !r.KeptToFinder() {
		t.Fatal("unmoderated player-key text is its finder's alone")
	}
	v := r.View()
	if !v.PlayerText {
		t.Fatal("player-key text is marked, so no model prompt carries it (Task 9c)")
	}
	if v.Name != genericName || v.NameSimple != genericNameSimple || strings.Contains(v.Description, `horse`) {
		t.Fatalf("everyone sees the generic trinket: %+v", v)
	}
	if v.Value != 12 || v.WeightLbs != 0.3 {
		t.Fatalf("the numbers are the item's for everyone: %+v", v)
	}
	if v.FinderUserId != 7 || v.Finder == nil || v.Finder.Name != r.Name || v.Finder.Description != r.Description || v.Finder.Finder != nil {
		t.Fatalf("the finder's own view: %+v", v.Finder)
	}
	if again := r.View(); again.Description != v.Description {
		t.Fatal("the generic description is the same every time")
	}
	if r.MaterialFor(7) != `pine` || r.MaterialFor(8) != `` || r.MaterialFor(0) != `` {
		t.Fatal("the material is the finder's alone")
	}

	r.FoundByUserId = 0 // nobody to keep it for: nobody reads it
	if v := r.View(); v.Finder != nil || v.Name != genericName {
		t.Fatalf("no finder, no finder view: %+v", v)
	}
	r.FoundByUserId, r.Status = 7, StatusRetired
	if v := r.View(); v.Finder != nil || v.Name != retiredName {
		t.Fatalf("retired text wins: %+v", v)
	}
	r.Status, r.Moderated = StatusReady, true
	if v := r.View(); v.Name != r.Name || v.Finder != nil || r.MaterialFor(8) != `pine` || !v.PlayerText {
		t.Fatalf("moderated player-key text is everyone's, and still never a model's: %+v", v)
	}
	r.PlayerKey = false
	if v := r.View(); v.Name != r.Name || v.Finder != nil || v.PlayerText || r.KeptToFinder() {
		t.Fatalf("server-key text is everyone's: %+v", v)
	}
}

// Shelvable is the shared rule (owner ruling 5, baubles slice D) for whether
// a sold or won bauble goes on a shop's resale shelf rather than leaving the
// world: worth more than the cheap tier, and not retired (ruling 1). Both
// the player-sale path (internal/actions) and the auction win path
// (modules/auctions) read it from here, so there is exactly one rule to
// change. The boundary is read from the tier config, never hardcoded.
func TestRecord_Shelvable(t *testing.T) {
	max := TierCheap.Range().Max

	atMax := Record{Status: StatusReady, Value: max}
	if atMax.Shelvable() {
		t.Fatal("a bauble worth exactly the cheap tier max must not be shelvable")
	}

	aboveMax := Record{Status: StatusReady, Value: max + 1}
	if !aboveMax.Shelvable() {
		t.Fatal("a bauble one gold above the cheap tier max must be shelvable")
	}

	retired := Record{Status: StatusRetired, Value: max + 1}
	if retired.Shelvable() {
		t.Fatal("a retired bauble must never be shelvable, whatever its value")
	}
}
