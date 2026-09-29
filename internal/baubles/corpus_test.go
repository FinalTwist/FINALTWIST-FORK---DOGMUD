package baubles

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/util"
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

// testSeed is a small seed: interior and fort in the dwelling group, ruins
// its own group, a bare cheap pool whose one entry is priced far above the
// tier (the clamp test), and a pocket pool with one entry too heavy for a
// pocket.
const testSeed = `groups:
  interior: dwelling
  fort: dwelling
  ruins: ruins
entries:
  interior-cheap:
    - name: Bent Tin Thimble
      name_simple: thimble
      description: A tin thimble, pressed a little out of shape by a careless heel.
      material: tin
      weight_lbs: 0.1
      value: 3
  dwelling-cheap:
    - name: Chipped Clay Marble
      name_simple: marble
      description: A small clay marble, glazed blue long ago and chipped since.
      material: clay
      weight_lbs: 0.1
      value: 2
  cheap:
    - name: Knotted Twine Bracelet
      name_simple: bracelet
      description: A bracelet of knotted brown twine, frayed where a wrist rubbed it.
      material: twine
      weight_lbs: 0.1
      value: 99
  pocket-cheap:
    - name: Brass Snuff Spoon
      name_simple: spoon
      description: A tiny brass spoon for snuff, its bowl no bigger than a fingernail.
      material: brass
      weight_lbs: 0.1
      value: 2
    - name: Heavy Pewter Tankard
      name_simple: tankard
      description: A pewter tankard with a hinged lid, far too heavy for any pocket.
      material: pewter
      weight_lbs: 1.5
      value: 5
  ruins-average:
    - name: Faded Mosaic Tile
      name_simple: tile
      description: A square tile from an old floor, painted with half of a red bird.
      material: fired clay
      weight_lbs: 0.3
      value: 13
`

const testSpoolOverlay = `entries:
  interior-cheap:
    - name: Painted Wooden Spool
      name_simple: spool
      description: A wooden thread spool painted with a band of faded blue.
      material: wood
      weight_lbs: 0.2
      value: 4
      from_record: B0000007
      zone: ashwick
      biome: interior
      model: gpt-test
      prompt_version: 3
      promoted_at: 2026-09-28T12:00:00Z
`

func writeTestFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}

// withCorpus writes a seed and an overlay (either may be empty, meaning no
// file) to a temp dir and loads them. The corpus is cleared afterwards.
func withCorpus(t *testing.T, seed, overlay string) (seedPath, overlayPath string, rep CorpusReport) {
	t.Helper()
	dir := t.TempDir()
	seedPath = filepath.Join(dir, seedFileName)
	overlayPath = filepath.Join(dir, `baubles`, overlayFileName)
	if seed != `` {
		writeTestFile(t, seedPath, seed)
	}
	if overlay != `` {
		writeTestFile(t, overlayPath, overlay)
	}
	t.Cleanup(ClearCorpusForTest)
	return seedPath, overlayPath, LoadCorpusFrom(seedPath, overlayPath)
}

func TestLoadCorpusReadsBothLayers(t *testing.T) {
	_, _, rep := withCorpus(t, testSeed, testSpoolOverlay)
	if rep.Seed != 6 || rep.Promoted != 1 || len(rep.Skipped) != 0 || rep.SeedErr != nil || rep.Quarantined != `` {
		t.Fatalf("report: %+v", rep)
	}
	if g, ok := GroupOf(`Interior`); !ok || g != `dwelling` {
		t.Fatalf("interior is in dwelling, got %q %v", g, ok)
	}
	if _, ok := GroupOf(`water`); ok {
		t.Fatal("water has no group")
	}
	if seed, promoted := CorpusCounts(); seed != 6 || promoted != 1 {
		t.Fatalf("counts: %d seed, %d promoted", seed, promoted)
	}
}

// Every entry must pass what a model's answer passes; one that fails is
// skipped and reported, and a key no pool can have is skipped whole.
func TestLoadCorpusSkipsWhatAModelAnswerCouldNotPass(t *testing.T) {
	seed := testSeed + `  interior-rare:
    - name: Silver Cup Number 2
      name_simple: cup
      description: A silver cup with a number scratched on its base.
      weight_lbs: 0.5
      value: 90
    - name: Stone Idol Head
      name_simple: idol
      description: A stone head broken from a small idol, far heavier than it looks.
      weight_lbs: 40
      value: 120
    - name: Plain Pewter Cup
      name_simple: cup
      description: A plain pewter drinking cup, dented on one side.
      material: pewter
      weight_lbs: 0.5
      value: 60
  castle-cheap:
    - name: Rusty Castle Nail
      name_simple: nail
      description: A square iron nail, rusted almost through.
      weight_lbs: 0.1
      value: 1
`
	_, _, rep := withCorpus(t, seed, ``)
	if rep.Seed != 7 || len(rep.Skipped) != 3 {
		t.Fatalf("seven usable, three skipped (digits, weight, unknown key): %+v", rep)
	}
}

// A broken seed is logged and the corpus runs without it: the overlay
// still loads (a bare tier key needs no groups) and nothing panics.
func TestMalformedSeedIsLoggedNotFatal(t *testing.T) {
	overlay := `entries:
  cheap:
    - name: Painted Wooden Spool
      name_simple: spool
      description: A wooden thread spool painted with a band of faded blue.
      weight_lbs: 0.2
      value: 4
      from_record: B0000007
      promoted_at: 2026-09-28T12:00:00Z
`
	_, _, rep := withCorpus(t, "entries: [this is not a map\n", overlay)
	if rep.SeedErr == nil || rep.Seed != 0 || rep.Promoted != 1 {
		t.Fatalf("report: %+v", rep)
	}
}

// A corrupt overlay is living state: moved aside (never deleted), logged,
// and the overlay restarts empty. An empty file is empty, not corrupt.
func TestCorruptOverlayIsQuarantined(t *testing.T) {
	seedPath, overlayPath, rep := withCorpus(t, testSeed, "entries: {this is: [not closed\n")
	if rep.Quarantined == `` || rep.Promoted != 0 || rep.Seed != 6 {
		t.Fatalf("report: %+v", rep)
	}
	if _, err := os.Stat(overlayPath); !os.IsNotExist(err) {
		t.Fatalf("the corrupt file is moved aside: %v", err)
	}
	if _, err := os.Stat(rep.Quarantined); err != nil {
		t.Fatalf("the quarantined copy is kept for recovery: %v", err)
	}
	writeTestFile(t, overlayPath, ``)
	if rep := LoadCorpusFrom(seedPath, overlayPath); rep.Quarantined != `` {
		t.Fatalf("an empty overlay must not be quarantined: %+v", rep)
	}
}

func TestNoCorpusFilesIsAnEmptyCorpus(t *testing.T) {
	_, _, rep := withCorpus(t, ``, ``)
	if rep.Seed != 0 || rep.Promoted != 0 || rep.SeedErr != nil || rep.Quarantined != `` {
		t.Fatalf("report: %+v", rep)
	}
}

// A reload that cannot read the seed keeps the seed already in use (a typo
// in a hand edit must not empty every pool); the overlay is still read
// again. A first load has nothing to keep.
func TestReloadWithABrokenSeedKeepsTheSeedInUse(t *testing.T) {
	seedPath, overlayPath, _ := withCorpus(t, testSeed, ``)
	writeTestFile(t, seedPath, "entries: [this is not a map\n")
	writeTestFile(t, overlayPath, testSpoolOverlay)
	rep := LoadCorpusFrom(seedPath, overlayPath)
	if rep.SeedErr == nil || !rep.SeedKept || rep.Seed != 6 || rep.Promoted != 1 {
		t.Fatalf("report: %+v", rep)
	}
	if g, ok := GroupOf(`fort`); !ok || g != `dwelling` {
		t.Fatalf("the kept seed keeps its groups too, got %q %v", g, ok)
	}
	if seed, _ := CorpusCounts(); seed != 6 {
		t.Fatalf("the kept seed is in use: %d", seed)
	}
}

// A corrupt overlay that cannot be moved aside is still where a save would
// write: the pool marks it broken (writers refuse, Task 7) and the next
// reload that can move it aside clears the mark.
func TestAnOverlayThatCannotBeQuarantinedIsBroken(t *testing.T) {
	quarantineOverlay = func(string) (string, error) { return ``, errors.New(`disk says no`) }
	t.Cleanup(func() { quarantineOverlay = util.QuarantineCorrupt })
	corrupt := "entries: {this is: [not closed\n"
	seedPath, overlayPath, rep := withCorpus(t, testSeed, corrupt)
	if !rep.OverlayBroken || rep.Quarantined != `` || rep.Promoted != 0 || rep.Seed != 6 {
		t.Fatalf("report: %+v", rep)
	}
	if data, err := os.ReadFile(overlayPath); err != nil || string(data) != corrupt {
		t.Fatalf("the file is left exactly as it was: %v", err)
	}
	quarantineOverlay = util.QuarantineCorrupt
	if rep := LoadCorpusFrom(seedPath, overlayPath); rep.OverlayBroken || rep.Quarantined == `` {
		t.Fatalf("moved aside on the next reload, and writable again: %+v", rep)
	}
}
