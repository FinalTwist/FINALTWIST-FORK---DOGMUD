package baubles

import (
	"testing"
)

func TestParseCorpusKey(t *testing.T) {
	cases := []struct {
		key    string
		prefix string
		tier   ValueTier
		ok     bool
	}{
		{`interior-cheap`, `interior`, TierCheap, true},
		{`city_backstreet-rare`, `city_backstreet`, TierRare, true},
		{`Pocket-Average`, `pocket`, TierAverage, true},
		{`cheap`, ``, TierCheap, true},
		{`interior`, ``, ``, false},
		{`interior-legendary`, ``, ``, false},
		{`-cheap`, ``, ``, false},
	}
	for _, c := range cases {
		prefix, tier, ok := parseCorpusKey(c.key)
		if prefix != c.prefix || tier != c.tier || ok != c.ok {
			t.Errorf("%q: got (%q, %q, %v), want (%q, %q, %v)", c.key, prefix, tier, ok, c.prefix, c.tier, c.ok)
		}
	}
	if corpusKey(`dwelling`, TierRare) != `dwelling-rare` || corpusKey(``, TierCheap) != `cheap` {
		t.Fatal("corpusKey is parseCorpusKey's inverse")
	}
}

// An entry gets the checks a model's answer gets, plus an exact weight.
func TestCheckEntry(t *testing.T) {
	good := CorpusEntry{Name: `Bent Tin Thimble`, NameSimple: `thimble`, Description: `A tin thimble, pressed a little out of shape.`, Material: `Tin`, WeightLbs: 0.1, Value: 3}
	got, err := checkEntry(good)
	if err != nil {
		t.Fatal(err)
	}
	if got.Material != `tin` || got.NameSimple != `thimble` || got.Value != 3 || got.WeightLbs != 0.1 {
		t.Fatalf("cleaned like a reply, numbers untouched: %+v", got)
	}
	bad := map[string]func(e *CorpusEntry){
		`digits in the name`: func(e *CorpusEntry) { e.Name = `Tin Thimble 2` },
		`too heavy`:          func(e *CorpusEntry) { e.WeightLbs = 30 },
		`not a tenth`:        func(e *CorpusEntry) { e.WeightLbs = 0.15 },
		`no weight`:          func(e *CorpusEntry) { e.WeightLbs = 0 },
		`short description`:  func(e *CorpusEntry) { e.Description = `Tiny.` },
	}
	for name, change := range bad {
		e := good
		change(&e)
		if _, err := checkEntry(e); err == nil {
			t.Errorf("%s: must be refused", name)
		}
	}
}

// A typo in a field name is an error, not a silently empty field.
func TestDecodeStrictRefusesUnknownFields(t *testing.T) {
	var doc seedDoc
	if err := decodeStrict([]byte("entries:\n  cheap:\n    - name: X\n      valeu: 3\n"), &doc); err == nil {
		t.Fatal("an unknown field must be refused")
	}
	if err := decodeStrict([]byte(``), &doc); err != nil {
		t.Fatalf("an empty document is empty, not an error: %v", err)
	}
}
