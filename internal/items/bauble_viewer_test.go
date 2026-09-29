package items

import (
	"strings"
	"testing"
)

// A finder-only bauble (text a player's own key wrote that the server could
// not moderate; owner ruling 2026-09-29) reads as the generic trinket
// through every viewer-agnostic accessor, so no render path built on them,
// today's or a new one, can show its text to anyone. Only the viewer-aware
// accessors (bauble_viewer.go) show it, and only to its finder.
func TestFinderOnlyBaubleIsGenericToEveryoneButItsFinder(t *testing.T) {
	restore := SeedItemsForTest(map[int]*ItemSpec{
		BaubleItemId: {ItemId: BaubleItemId, Name: `Curious Trinket`, NameSimple: `trinket`, Type: Object, Subtype: Mundane},
	})
	defer restore()
	SetBaubleResolver(func(id string) (BaubleView, bool) {
		return BaubleView{
			Name: `Trinket`, NameSimple: `trinket`, Description: `A small trinket of no particular make.`, Value: 12, WeightLbs: 0.2,
			FinderUserId: 7,
			Finder: &BaubleView{Name: `Painted Wooden Horse`, NameSimple: `horse`,
				Description: `A child's toy horse, its red paint flaking.`, Value: 12, WeightLbs: 0.2},
		}, id == `b1`
	})
	defer SetBaubleResolver(nil)

	itm := New(BaubleItemId)
	itm.Bauble = `b1`
	spec := itm.GetSpec()
	for accessor, text := range map[string]string{
		`GetSpec().Name`: spec.Name, `GetSpec().NameSimple`: spec.NameSimple, `GetSpec().Description`: spec.Description,
		`Name`: itm.Name(), `NameSimple`: itm.NameSimple(), `DisplayName`: itm.DisplayName(),
		`NameComplex`: itm.NameComplex(), `GetLongDescription`: itm.GetLongDescription(),
	} {
		if strings.Contains(strings.ToLower(text), `horse`) {
			t.Errorf("%s shows the finder's own text to everyone: %q", accessor, text)
		}
	}
	for _, viewer := range []int{0, 8} {
		if got := itm.DisplayNameFor(viewer); got != `Trinket` {
			t.Errorf("viewer %d reads the generic name, got %q", viewer, got)
		}
		if got := itm.NameFor(viewer) + itm.LongDescriptionFor(viewer) + itm.GetSpecFor(viewer).NameSimple; strings.Contains(strings.ToLower(got), `horse`) {
			t.Errorf("viewer %d reads the finder's text: %q", viewer, got)
		}
	}
	if itm.DisplayNameFor(7) != `Painted Wooden Horse` || itm.NameFor(7) != `Painted Wooden Horse` ||
		itm.GetSpecFor(7).NameSimple != `horse` || !strings.Contains(itm.LongDescriptionFor(7), `toy horse`) {
		t.Fatalf("the finder reads their own text: %q %q", itm.DisplayNameFor(7), itm.LongDescriptionFor(7))
	}
	// The finder types the words they read; matching shows nobody any text.
	if part, _ := itm.NameMatch(`horse`, true); !part {
		t.Error("the finder's word matches the trinket")
	}
	if _, full := itm.NameMatch(`trinket`, true); !full {
		t.Error("the generic keyword still names it in full")
	}
}

// For anything that is not a finder-only bauble, each viewer-aware accessor
// is its viewer-agnostic twin.
func TestViewerAccessorsAreTheirTwinsForAnyOtherItem(t *testing.T) {
	restore := SeedItemsForTest(map[int]*ItemSpec{
		10: {ItemId: 10, Name: `Hooded Lantern`, NameSimple: `lamp`, Description: `A lantern with a hood.`, Type: Object, Subtype: Mundane},
	})
	defer restore()
	itm := New(10)
	if itm.DisplayNameFor(7) != itm.DisplayName() || itm.NameFor(7) != itm.Name() ||
		itm.LongDescriptionFor(7) != itm.GetLongDescription() || itm.GetSpecFor(7).Name != itm.GetSpec().Name {
		t.Fatal("an ordinary item reads the same to every viewer")
	}
}
