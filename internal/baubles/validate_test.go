package baubles

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// Every code point the v10 review showed surviving CleanReply, plus the
// others the spec names (S4): each is dropped or turned into a plain space,
// never kept.
func TestCleanReplyStripsInvisibleAndFormatCharacters(t *testing.T) {
	cases := map[string]rune{
		`zero width space`:   '\U0000200B',
		`right-to-left mark`: '\U0000202E',
		`byte order mark`:    '\U0000FEFF',
		`line separator`:     '\U00002028',
		`paragraph sep`:      '\U00002029',
		`hangul filler`:      '\U00003164',
		`hangul choseong f`:  '\U0000115F',
		`hangul jungseong f`: '\U00001160',
		`halfwidth filler`:   '\U0000FFA0',
		`private use`:        '\U0000E000',
		`combining stroke`:   '\U00000336',
		`ideographic space`:  '\U00003000',
	}
	for name, r := range cases {
		reply := goodReply()
		reply.Name = `Painted` + string(r) + ` Wooden Horse`
		reply.Description = `A child's toy` + string(r) + ` horse, its red paint flaking from the mane.`
		got, err := CleanReply(reply)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if strings.ContainsRune(got.Name, r) || strings.ContainsRune(got.Description, r) {
			t.Errorf("%s (U+%04X) survived", name, r)
		}
		if strings.Contains(got.Name, `  `) {
			t.Errorf("%s: spaces not collapsed in %q", name, got.Name)
		}
	}

	// NBSP and the other spaces become an ordinary space.
	reply := goodReply()
	reply.Name = "Painted\U000000A0Wooden\U00002003Horse"
	if got, err := CleanReply(reply); err != nil || got.Name != `Painted Wooden Horse` {
		t.Fatalf("NBSP and em space read as spaces: %q %v", got.Name, err)
	}

	// An OSC sequence (not the CSI colour codes ansiRE knows) leaves no
	// escape or bell behind. It goes in the description: its "8" would
	// refuse a name for its digit before the escape was ever looked at.
	reply = goodReply()
	reply.Description = "A child's toy horse\x1b]8;;evil\x07, its red paint flaking from the mane."
	got, err := CleanReply(reply)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(got.Description, "\x1b\x07") {
		t.Fatalf("OSC control bytes survived: %q", got.Description)
	}

	// Fullwidth markup is folded by NFKC before the markup is stripped.
	reply = goodReply()
	reply.Name = "Painted \U0000FF1Cb\U0000FF1EWooden Horse"
	if got, err := CleanReply(reply); err != nil || strings.ContainsAny(got.Name, "<>\U0000FF1C\U0000FF1E") {
		t.Fatalf("fullwidth angle brackets: %q %v", got.Name, err)
	}
}

// Text that reads as a link is refused in every field (spec S4).
func TestCleanReplyRefusesLinks(t *testing.T) {
	cases := map[string]func(*Reply){
		`scheme`:      func(r *Reply) { r.Description = `A toy horse. Details at http://example.org today.` },
		`www`:         func(r *Reply) { r.Description = `A toy horse from WWW.shop, still boxed in paper.` },
		`bare domain`: func(r *Reply) { r.Name = `Evil.com Horse` },
		`path`:        func(r *Reply) { r.Description = `A toy horse with a tag reading evil.co/x on its belly.` },
		`fullwidth`: func(r *Reply) {
			r.Description = "A toy horse stamped \U0000FF57\U0000FF57\U0000FF57\U0000FF0Eshop in red."
		},
		`material`: func(r *Reply) { r.Material = `pine.io` },
	}
	for name, change := range cases {
		r := goodReply()
		change(&r)
		if _, err := CleanReply(r); !errors.Is(err, ErrUnusableReply) {
			t.Errorf("%s must be refused, got %v", name, err)
		}
	}
	// Ordinary sentences are not links.
	r := goodReply()
	r.Description = `A cup. It is old, and chipped at the rim! Who left it here?`
	if _, err := CleanReply(r); err != nil {
		t.Fatalf("plain sentences pass: %v", err)
	}
}

// Lengths are runes, not bytes (spec S4).
func TestCleanReplyCountsRunes(t *testing.T) {
	const eAcute = "\U000000E9"
	r := goodReply()
	r.Name = strings.Repeat(eAcute, maxNameLen-2) + ` A` // 40 runes, 78 bytes
	if _, err := CleanReply(r); err != nil {
		t.Fatalf("a %d-rune name fits: %v", utf8.RuneCountInString(r.Name), err)
	}
	r.Name = strings.Repeat(eAcute, maxNameLen-1) + ` A` // 41 runes
	if _, err := CleanReply(r); !errors.Is(err, ErrUnusableReply) {
		t.Fatal("a 41-rune name is too long")
	}
	r = goodReply()
	r.Description = strings.Repeat(eAcute, minDescriptionLen-1) // 19 runes, 38 bytes
	if _, err := CleanReply(r); !errors.Is(err, ErrUnusableReply) {
		t.Fatal("19 runes is too short, however many bytes")
	}
}

// A refusal quotes at most 60 runes of the offending text: a reply can be
// a megabyte, and the error is logged at Warn.
func TestCleanReplyErrorsQuoteLittle(t *testing.T) {
	r := goodReply()
	r.Name = strings.Repeat(`Long `, 100)
	_, err := CleanReply(r)
	if err == nil {
		t.Fatal("refused")
	}
	if strings.Contains(err.Error(), strings.Repeat(`Long `, 13)) {
		t.Fatalf("the error quotes more than 60 runes: %d bytes", len(err.Error()))
	}
}

// Curly quotes, en and em dashes and the ellipsis fold to ASCII (owner
// ruling 15), so ordinary typography from a model reads the same on every
// route and can pass the player-key allowlist (Task 7).
func TestCleanReplyFoldsTypography(t *testing.T) {
	r := goodReply()
	r.Name = "Mara\U00002019s Wooden Horse"
	r.Description = "A child\U00002018s toy horse \U00002014 its paint flaking \U00002013 marked \U0000201CMara\U0000201D\U00002026 still loved."
	got, err := CleanReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != `Mara's Wooden Horse` {
		t.Errorf("name folded: got %q", got.Name)
	}
	if want := `A child's toy horse - its paint flaking - marked "Mara"... still loved.`; got.Description != want {
		t.Errorf("description folded:\n got %q\nwant %q", got.Description, want)
	}
}

// A bauble may not carry a real item's whole name, however it is cased or
// spaced (spec S3): "Hooded Lantern" on a trinket would pass for the real
// one in a shop list or a trade.
func TestCleanReplyRefusesAnAuthoredItemsName(t *testing.T) {
	restore := items.SeedItemsForTest(map[int]*items.ItemSpec{
		10: {ItemId: 10, Name: `Painted Wooden Horse`, NameSimple: `toyhorse`},
	})
	defer restore()
	for _, name := range []string{`Painted Wooden Horse`, `painted  wooden horse`, "Painted\U000000A0Wooden Horse"} {
		r := goodReply()
		r.Name = name
		if _, err := CleanReply(r); !errors.Is(err, ErrUnusableReply) {
			t.Errorf("%q is a real item's name: refused, got %v", name, err)
		}
	}
	r := goodReply()
	r.Name = `Painted Wooden Horses`
	if _, err := CleanReply(r); err != nil {
		t.Fatalf("a name that is not a real item's passes: %v", err)
	}
}
