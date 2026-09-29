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
	// Matching is viewer-agnostic, so a hidden word that matched would
	// confirm the hidden text to anyone who guessed it. The trinket answers
	// only to its generic words, for its finder too.
	if part, full := itm.NameMatch(`horse`, true); part || full {
		t.Error("a hidden word must not match the trinket")
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

// A finder-only bauble matches by its generic words alone. Its hidden
// words ("horse") match for nobody, its finder included (matching is not
// viewer-aware; the finder calls it a trinket): otherwise `look horse`
// would confirm a hidden word to anyone, and a whole-word hidden match
// (strength 3) would beat a real "Horseshoe" the word only starts
// (strength 2), so `get horse` would pick up the trinket. A moderated
// player-key bauble and a server-key bauble keep matching by their real
// words.
func TestFinderOnlyBaubleMatchesOnlyByItsGenericWords(t *testing.T) {
	restore := SeedItemsForTest(map[int]*ItemSpec{
		BaubleItemId: {ItemId: BaubleItemId, Name: `Curious Trinket`, NameSimple: `trinket`, Type: Object, Subtype: Mundane, Weight: 0.2, Value: 1},
		7401:         {ItemId: 7401, Name: `Horseshoe`, NameSimple: `horseshoe`, Type: Object, Value: 4},
	})
	t.Cleanup(func() { restore(); SetBaubleResolver(nil) })
	SetBaubleResolver(testBaubleResolver(map[string]BaubleView{
		`kept`: {
			Name: `Trinket`, NameSimple: `trinket`, Value: 12, WeightLbs: 0.2,
			FinderUserId: 7, PlayerText: true,
			Finder: &BaubleView{Name: `Painted Wooden Horse`, NameSimple: `horse`, Value: 12, WeightLbs: 0.2, PlayerText: true},
		},
		`moderated`: {Name: `Carved Bone Horse`, NameSimple: `figurine`, Value: 12, PlayerText: true},
		`server`:    {Name: `Tin Rocking Horse`, NameSimple: `toy`, Value: 12},
	}))
	kept := New(BaubleItemId)
	kept.Bauble = `kept`
	moderated := New(BaubleItemId)
	moderated.Bauble = `moderated`
	server := New(BaubleItemId)
	server.Bauble = `server`
	shoe := New(7401)

	pick := func(input string, list ...Item) Item {
		part, full := FindMatchIn(input, list...)
		if full.ItemId != 0 {
			return full
		}
		return part
	}

	for _, word := range []string{`horse`, `wooden horse`, `painted`, `hor`} {
		if part, full := kept.NameMatch(word, true); part || full {
			t.Errorf("%q: a hidden word matches the finder-only bauble", word)
		}
		if st := matchStrength(&kept, word); st != 0 {
			t.Errorf("%q: a hidden word ranks the finder-only bauble at %d", word, st)
		}
		if got := pick(word, kept); got.ItemId != 0 {
			t.Errorf("%q: look finds the finder-only bauble alone, got %+v", word, got)
		}
	}
	for _, list := range [][]Item{{kept, shoe}, {shoe, kept}} {
		if got := pick(`horse`, list...); got.ItemId != 7401 {
			t.Fatalf("get horse -> the real horseshoe, got %+v", got)
		}
	}
	if got := pick(`trinket`, shoe, kept); got.Bauble != `kept` {
		t.Fatalf("trinket names the finder-only bauble, got %+v", got)
	}
	for _, b := range []Item{moderated, server} {
		if part, _ := b.NameMatch(`horse`, true); !part {
			t.Errorf("%s: a shown bauble matches by its real words", b.Bauble)
		}
		if got := pick(`horse`, b); got.Bauble != b.Bauble {
			t.Errorf("%s: horse finds it, got %+v", b.Bauble, got)
		}
	}
}
