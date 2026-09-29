package items

import (
	"regexp"
	"strings"
	"sync/atomic"
	"unicode"

	"github.com/GoMudEngine/GoMud/internal/util"
)

// BaubleItemId is the one carrier item every bauble is an instance of
// (_datafiles/world/dogmud/items/other-0/900-curious_trinket.yaml). A bauble
// is that item plus Item.Bauble, a record id in the bauble catalog
// (internal/baubles), which supplies its name, description, value and
// weight. See docs/baubles/implementation-plan.md.
const BaubleItemId = 900

// baubleKeywords are words every bauble answers to in NameMatch, whatever
// its generated name, so `sell bauble` and `drop trinket` always work.
var baubleKeywords = []string{`bauble`, `trinket`}

// BaubleView is what the catalog tells the item layer about one bauble.
// DisplayName is optional; when set it is rendered verbatim, as for any
// authored DisplayName.
type BaubleView struct {
	Name        string
	NameSimple  string
	DisplayName string
	Description string
	Value       int
	WeightLbs   float64

	// FinderUserId and Finder make a finder-only view (internal/baubles
	// Record.KeptToFinder, owner ruling 2026-09-29): the fields above are what
	// everyone sees, Finder what player FinderUserId sees, and only through
	// the viewer-aware accessors (bauble_viewer.go). Nil Finder: everyone
	// sees the fields above.
	FinderUserId int
	Finder       *BaubleView

	// PlayerText: the record's text was written by a player's own key,
	// moderated or not. Such text never goes into any language model's
	// prompt (spec S3): ModelName and ModelDescription (Task 9c) show the
	// carrier's own name instead.
	PlayerText bool
}

// BaubleResolver looks a catalog record up by id.
type BaubleResolver func(id string) (BaubleView, bool)

// baubleResolver is installed by internal/baubles at boot. This package
// never imports that one (it imports this), so the lookup is a function
// value. Atomic because GetSpec is read from more than the game loop.
var baubleResolver atomic.Pointer[BaubleResolver]

// SetBaubleResolver installs the catalog lookup. nil uninstalls it, after
// which every bauble shows as the plain carrier item.
func SetBaubleResolver(f BaubleResolver) {
	if f == nil {
		baubleResolver.Store(nil)
		return
	}
	baubleResolver.Store(&f)
}

// IsBauble reports whether this item is a catalog-backed bauble.
func (i *Item) IsBauble() bool {
	return i.Bauble != ``
}

// baubleSpec overlays a bauble's catalog fields on the carrier's spec. With
// no resolver, or a record that cannot be found, the carrier spec is
// returned unchanged, so a missing record degrades to "Curious Trinket"
// rather than to an error.
func baubleSpec(base ItemSpec, id string) ItemSpec {
	return baubleSpecFor(base, id, 0)
}

// baubleSpecFor is baubleSpec as viewerUserId sees it: a finder-only
// bauble's own text for its finder, the generic view for everyone else.
// Viewer 0 is nobody, so baubleSpec is always the generic view.
func baubleSpecFor(base ItemSpec, id string, viewerUserId int) ItemSpec {
	p := baubleResolver.Load()
	if p == nil || *p == nil {
		return base
	}
	v, ok := (*p)(id)
	if !ok {
		return base
	}
	if v.Finder != nil && viewerUserId > 0 && viewerUserId == v.FinderUserId {
		v = *v.Finder
	}
	if v.Name != `` {
		base.Name = v.Name
	}
	if v.NameSimple != `` {
		base.NameSimple = v.NameSimple
	}
	base.DisplayName = v.DisplayName
	if v.Description != `` {
		base.Description = v.Description
	}
	base.Value = v.Value
	if v.WeightLbs > 0 {
		base.Weight = v.WeightLbs
	}
	return base
}

// baubleFillerWords are dropped from the front of what a player types, so
// `get a doll` and `look the doll` work like `get doll`.
var baubleFillerWords = map[string]bool{`a`: true, `an`: true, `the`: true, `some`: true}

// matchWords splits already-normalised text into lowercase words: letters
// and digits only, so "half-burnt" is two words and "child's" (with its
// apostrophe already removed by util.NormalizeForMatch) is one.
func matchWords(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// baubleWordMatch matches what a player typed against a bauble's names word
// by word. Every word typed must match a word of the name, in the same
// order, though words may be skipped:
//
//	"Small Child's Doll"  <- doll, childs doll, child's doll, small doll,
//	                         a doll, small childs doll          (full)
//	                      <- dol, sm doll, chi do               (partial)
//	                      <- doll small, horse                  (no match)
//
// full is set when every typed word equals a whole word of the name;
// partial when every typed word is at least the start of one. Both input
// and names arrive already through util.NormalizeForMatch.
func baubleWordMatch(input string, names ...string) (partial bool, full bool) {
	typed := matchWords(input)
	for len(typed) > 0 && baubleFillerWords[typed[0]] {
		typed = typed[1:]
	}
	if len(typed) == 0 {
		return false, false
	}
	for _, name := range names {
		words := matchWords(name)
		if len(words) == 0 {
			continue
		}
		if wordsInOrder(typed, words, sameWord) {
			return true, true
		}
		if wordsInOrder(typed, words, startsWord) {
			partial = true
		}
	}
	return partial, false
}

// possessiveRE matches a possessive 's (straight or curly apostrophe) at the
// end of a word.
var possessiveRE = regexp.MustCompile(`(?i)['’]s\b`)

// withoutPossessives is a name with possessive 's dropped, lowercased, so
// "Small Child's Doll" also reads as "small child doll" and `get child doll`
// matches as well as `get childs doll`.
func withoutPossessives(name string) string {
	return util.NormalizeForMatch(possessiveRE.ReplaceAllString(name, ``))
}

// wordsInOrder reports whether each typed word matches some word of the
// name, each one after the last word matched. match is called as
// match(typedWord, nameWord).
func wordsInOrder(typed []string, words []string, match func(typedWord string, nameWord string) bool) bool {
	pos := 0
	for _, t := range typed {
		found := false
		for pos < len(words) {
			w := words[pos]
			pos++
			if match(t, w) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func sameWord(typedWord string, nameWord string) bool {
	return typedWord == nameWord
}

func startsWord(typedWord string, nameWord string) bool {
	return strings.HasPrefix(nameWord, typedWord)
}

// matchStrength ranks how well input names i, to choose between a bauble
// and a real item that both match (FindMatchIn): 4 an exact name or
// keyword; 3 every typed word a whole word of the name, in order; 2 every
// typed word the start of a word; 1 the name merely contains it; 0 no match.
// It reads the same name variants NameMatch does.
func matchStrength(i *Item, input string) int {
	part, full := i.NameMatch(input, true)
	if !full {
		// With contains allowed, NameMatch stops at the first name variant
		// that contains input ("rag doll" for `doll`) before it reaches an
		// exact one (the keyword "doll"), so ask for the exact one too.
		_, full = i.NameMatch(input, false)
	}
	if full {
		return 4
	}
	if !part {
		return 0
	}
	in := util.NormalizeForMatch(input)
	names := []string{util.NormalizeForMatch(i.Name()), util.NormalizeForMatch(i.NameSimple()), withoutPossessives(i.Name())}
	names = append(names, i.baubleFinderNames()...)
	if len(i.Adjectives) > 0 {
		names = append(names, util.NormalizeForMatch(strings.Join(i.Adjectives, " ")+" "+i.NameSimple()))
	}
	wordPart, wordFull := baubleWordMatch(in, names...)
	switch {
	case wordFull:
		return 3
	case wordPart:
		return 2
	}
	return 1
}

// baubleFinderNames is a finder-only bauble's own name and keyword,
// normalised for matching, so its finder can type the words they read
// (`drop horse`). Matching shows nobody any text. Nil for any other item.
func (i *Item) baubleFinderNames() []string {
	if i.Bauble == `` {
		return nil
	}
	p := baubleResolver.Load()
	if p == nil || *p == nil {
		return nil
	}
	v, ok := (*p)(i.Bauble)
	if !ok || v.Finder == nil {
		return nil
	}
	return []string{util.NormalizeForMatch(v.Finder.Name), util.NormalizeForMatch(v.Finder.NameSimple), withoutPossessives(v.Finder.Name)}
}

// anyBauble reports whether any item in the list is a bauble.
func anyBauble(items []Item) bool {
	for idx := range items {
		if items[idx].IsBauble() {
			return true
		}
	}
	return false
}

// strongestBauble is the bauble input matches best (matchStrength), the
// first in list order on a tie, and its strength; 0 when no bauble matches.
func strongestBauble(input string, items []Item) (best Item, strength int) {
	for idx := range items {
		i := items[idx]
		if !i.IsBauble() {
			continue
		}
		if st := matchStrength(&i, input); st > strength {
			best, strength = i, st
		}
	}
	return best, strength
}

// findMatchWithBaubles is FindMatchIn (no explicit N.) over a list holding a
// bauble. The real items are chosen among themselves by the list-order
// rule (findMatchInOrder), exactly as if no bauble were there, and only
// that choice is weighed against the best bauble (strongestBauble):
//   - a real item named in full wins outright;
//   - a bauble named in full (its exact name or keyword) beats a real item
//     matched only in part;
//   - otherwise the real item wins on an equal or stronger match: a bauble
//     named by a whole word ("button" for "Tarnished Copper Button") beats
//     a real item the word only starts ("Buttoned Leather Vest");
//   - except that a household's bauble (BaubleHousehold set: it lies where
//     it belongs, and taking it is theft) never beats a real item on a
//     partial match, so `get candle` finds the Candlestick, not the Stub of
//     Candle. `steal` names household baubles on their own
//     (householdBaubleNamed).
func findMatchWithBaubles(input string, items []Item) (pMatch Item, fMatch Item) {
	real := make([]Item, 0, len(items))
	for idx := range items {
		if !items[idx].IsBauble() {
			real = append(real, items[idx])
		}
	}
	realPart, realFull := findMatchInOrder(input, 1, real)
	if realFull.ItemId > 0 {
		return Item{}, realFull
	}

	bauble, strength := strongestBauble(input, items)
	switch {
	case strength == 0:
		return realPart, Item{}
	case strength == 4:
		return Item{}, bauble
	case realPart.ItemId > 0 && bauble.BaubleHousehold != 0:
		return realPart, Item{}
	case realPart.ItemId > 0 && matchStrength(&realPart, input) >= strength:
		return realPart, Item{}
	}
	return bauble, Item{}
}
