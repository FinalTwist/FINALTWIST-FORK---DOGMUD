package baubles

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/util"
)

// promotable is a record Promote accepts: named by the model on the
// server's key, moderated, never hand-edited, found indoors.
func promotable() Record {
	return Record{
		Name: `Painted Wooden Spool`, NameSimple: `spool`,
		Description: `A wooden thread spool painted with a band of faded blue.`,
		Material:    `wood`, Tier: TierCheap, Value: 4, WeightLbs: 0.2, Status: StatusReady,
		Source: SourceSearch, Zone: `ashwick`, Biome: `interior`, Generator: GeneratorOpenAI,
		Model: `gpt-test`, PromptVersion: 3, Moderated: true,
	}
}

func TestPromoteRefusals(t *testing.T) {
	withCatalog(t)
	withCorpus(t, testSeed, ``)
	cases := []struct {
		name   string
		change func(r *Record)
		want   error
	}{
		{`player key`, func(r *Record) { r.PlayerKey = true }, ErrPromotePlayerKey},
		{`unmoderated`, func(r *Record) { r.Moderated = false }, ErrPromoteUnmoderated},
		{`hand-edited`, func(r *Record) { r.HandEdited = true }, ErrPromoteEdited},
		{`retired`, func(r *Record) { r.Status = StatusRetired }, ErrPromoteRetired},
		{`generic`, func(r *Record) { r.Generator = GeneratorLocal }, ErrPromoteNotModel},
		{`from the corpus`, func(r *Record) { r.Generator = GeneratorCorpus }, ErrPromoteNotModel},
		{`unmapped biome`, func(r *Record) { r.Biome = `water` }, ErrPromoteNoGroup},
		{`no biome`, func(r *Record) { r.Biome = `` }, ErrPromoteNoGroup},
		{`too big for a pocket`, func(r *Record) {
			r.Source, r.Name, r.NameSimple = SourcePickpocket, `Bronze Funeral Urn`, `urn`
		}, ErrPromoteTooBig},
		// interior-cheap's own seed entry, and dwelling-cheap's, which
		// Fallback merges into the same pool.
		{`a name in its pool's seed`, func(r *Record) {
			r.Name, r.NameSimple = `Bent Tin Thimble`, `thimble`
		}, ErrPromoteNameTaken},
		{`a name in its group's seed`, func(r *Record) {
			r.Name, r.NameSimple = `chipped clay marble`, `marble`
		}, ErrPromoteNameTaken},
	}
	for _, c := range cases {
		r := promotable()
		c.change(&r)
		rec := seedRecord(t, r)
		if _, err := Promote(rec.Id); !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
	if _, promoted := CorpusCounts(); promoted != 0 {
		t.Fatalf("nothing refused may reach the overlay, got %d", promoted)
	}
	if _, err := Promote(`B9999999`); !errors.Is(err, ErrNoRecord) {
		t.Fatalf("no record: %v", err)
	}
	ClearCorpusForTest()
	if _, err := Promote(seedRecord(t, promotable()).Id); !errors.Is(err, ErrNoCorpus) {
		t.Fatalf("no corpus loaded: %v", err)
	}
}

// A sold find is promotable. It is saved with its provenance, used at once,
// read back on reload, and refused a second time.
func TestPromoteASoldFindAndUseIt(t *testing.T) {
	withCatalog(t)
	seedPath, overlayPath, _ := withCorpus(t, testSeed, ``)
	r := promotable()
	r.Status, r.SoldValue = StatusSold, 3
	rec := seedRecord(t, r)

	key, err := Promote(rec.Id)
	if err != nil || key != `interior-cheap` {
		t.Fatalf("promote: %q %v", key, err)
	}
	data, err := os.ReadFile(overlayPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`from_record: ` + rec.Id, `Painted Wooden Spool`, `model: gpt-test`, `biome: interior`, `promoted_at:`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("the overlay holds %q:\n%s", want, data)
		}
	}
	if _, err := Promote(rec.Id); !errors.Is(err, ErrPromoteDuplicate) {
		t.Fatalf("twice: %v", err)
	}
	if _, err := Promote(seedRecord(t, promotable()).Id); !errors.Is(err, ErrPromoteNameTaken) {
		t.Fatalf("another record with a name already promoted to that pool: %v", err)
	}
	recent := []string{`Bent Tin Thimble`, `Chipped Clay Marble`}
	if res := Fallback(Place{Biome: `interior`}, TierCheap, SourceSearch, recent, first); res.Reply.Name != `Painted Wooden Spool` || res.Model != `corpus:interior-cheap` {
		t.Fatalf("the promoted entry is in the pool at once: %+v", res)
	}
	LoadCorpusFrom(seedPath, overlayPath)
	if _, promoted := CorpusCounts(); promoted != 1 {
		t.Fatalf("read back from disk: %d", promoted)
	}

	p := promotable()
	p.Source, p.Name, p.NameSimple, p.WeightLbs = SourcePickpocket, `Carved Walnut Button`, `button`, 0.1
	if key, err := Promote(seedRecord(t, p).Id); err != nil || key != `pocket-cheap` {
		t.Fatalf("a pickpocketed find goes to the pocket pool: %q %v", key, err)
	}
}

// Persist before publish: a promotion whose save fails changes nothing.
func TestPromoteChangesNothingWhenTheSaveFails(t *testing.T) {
	withCatalog(t)
	dir := t.TempDir()
	seedPath := filepath.Join(dir, seedFileName)
	writeTestFile(t, seedPath, testSeed)
	blocker := filepath.Join(dir, `blocker`)
	writeTestFile(t, blocker, `a file where the overlay's directory should be`)
	LoadCorpusFrom(seedPath, filepath.Join(blocker, overlayFileName))
	t.Cleanup(ClearCorpusForTest)

	if _, err := Promote(seedRecord(t, promotable()).Id); err == nil {
		t.Fatal("a save that cannot be written must fail the promotion")
	}
	if _, promoted := CorpusCounts(); promoted != 0 {
		t.Fatal("and nothing is published in memory either")
	}
}

func TestRetireRemovesItsCorpusEntry(t *testing.T) {
	withCatalog(t)
	_, overlayPath, _ := withCorpus(t, testSeed, ``)
	k := promotable()
	k.Name, k.NameSimple = `Carved Walnut Button`, `button`
	keep := seedRecord(t, k)
	gone := seedRecord(t, promotable())
	for _, id := range []string{keep.Id, gone.Id} {
		if _, err := Promote(id); err != nil {
			t.Fatal(err)
		}
	}
	if err := Retire(gone.Id, `Admin`); err != nil {
		t.Fatal(err)
	}
	l := CorpusList(`interior-cheap`)
	if len(l.Promoted) != 1 || l.Promoted[0].FromRecord != keep.Id {
		t.Fatalf("only the retired record's entry goes: %+v", l.Promoted)
	}
	data, _ := os.ReadFile(overlayPath)
	if strings.Contains(string(data), gone.Id) {
		t.Fatal("and it is gone from disk")
	}
}

// A promoted record's text can change after promotion: an edit or a
// regeneration removes what was promoted from it, and says how much.
func TestEditAndRegenRemoveTheRecordsCorpusEntry(t *testing.T) {
	withCatalog(t)
	_, overlayPath, _ := withCorpus(t, testSeed, ``)
	edited := seedRecord(t, promotable())
	p := promotable()
	p.Name, p.NameSimple = `Carved Walnut Button`, `button`
	regen := seedRecord(t, p)
	for _, id := range []string{edited.Id, regen.Id} {
		if _, err := Promote(id); err != nil {
			t.Fatal(err)
		}
	}

	_, removed, err := Edit(edited.Id, `desc`, `A wooden spool with a band of blue paint, most of it worn away.`, `Admin`)
	if err != nil || removed != 1 {
		t.Fatalf("an edit removes its one promoted entry: %d %v", removed, err)
	}
	_, removed, err = ApplyRegenerated(regen.Id, GenResult{Reply: goodReply(), Generator: GeneratorOpenAI, Model: `gpt-test`, Moderated: true}, `Admin`, first)
	if err != nil || removed != 1 {
		t.Fatalf("a regeneration removes its one promoted entry: %d %v", removed, err)
	}
	if l := CorpusList(`interior-cheap`); len(l.Promoted) != 0 {
		t.Fatalf("both gone from memory: %+v", l.Promoted)
	}
	data, _ := os.ReadFile(overlayPath)
	if strings.Contains(string(data), edited.Id) || strings.Contains(string(data), regen.Id) {
		t.Fatalf("and from disk:\n%s", data)
	}
	if _, removed, err := Edit(edited.Id, `value`, `3`, `Admin`); err != nil || removed != 0 {
		t.Fatalf("a record with nothing promoted removes nothing: %d %v", removed, err)
	}
}

// Promotion looks only at whose hand wrote the text: a regenerated record
// (moderated, on the server's key) and a retired then restored one are
// promotable; a hand-edited one is not.
func TestPromoteLooksAtHandEditedNotEditedBy(t *testing.T) {
	withCatalog(t)
	withCorpus(t, testSeed, ``)

	r := promotable()
	r.Moderated, r.HandEdited = false, true
	regen := seedRecord(t, r)
	reply := goodReply()
	reply.Name, reply.NameSimple = `Painted Clay Owl`, `owl`
	if _, _, err := ApplyRegenerated(regen.Id, GenResult{Reply: reply, Generator: GeneratorOpenAI, Model: `gpt-test`, Moderated: true}, `Admin`, first); err != nil {
		t.Fatal(err)
	}
	if _, err := Promote(regen.Id); err != nil {
		t.Fatalf("freshly regenerated, moderated, server-key text is promotable: %v", err)
	}

	p := promotable()
	p.Name, p.NameSimple = `Carved Walnut Button`, `button`
	restored := seedRecord(t, p)
	if err := Retire(restored.Id, `Admin`); err != nil {
		t.Fatal(err)
	}
	if err := Restore(restored.Id, `Admin`); err != nil {
		t.Fatal(err)
	}
	if _, err := Promote(restored.Id); err != nil {
		t.Fatalf("retired then restored, the text is still the model's: %v", err)
	}

	q := promotable()
	q.Name, q.NameSimple = `Tarnished Brass Thimble`, `thimble`
	edited := seedRecord(t, q)
	if _, _, err := Edit(edited.Id, `material`, `copper`, `Admin`); err != nil {
		t.Fatal(err)
	}
	if _, err := Promote(edited.Id); !errors.Is(err, ErrPromoteEdited) {
		t.Fatalf("hand-edited text is never promoted: %v", err)
	}
}

// Removal names the entry (its name, any case, or the record it came from),
// never a position that shifts as entries come and go.
func TestRemoveCorpusEntry(t *testing.T) {
	withCatalog(t)
	_, overlayPath, _ := withCorpus(t, testSeed, ``)
	rec := seedRecord(t, promotable())
	p := promotable()
	p.Name, p.NameSimple = `Carved Walnut Button`, `button`
	other := seedRecord(t, p)
	for _, id := range []string{rec.Id, other.Id} {
		if _, err := Promote(id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := RemoveCorpusEntry(`interior-cheap`, `Silver Spoon`); err == nil {
		t.Fatal("no entry by that name")
	}
	if _, err := RemoveCorpusEntry(`dwelling-cheap`, `Chipped Clay Marble`); err == nil || !strings.Contains(err.Error(), `seed`) {
		t.Fatalf("seed entries are not removable in game: %v", err)
	}
	removed, err := RemoveCorpusEntry(`Interior-Cheap`, `painted wooden SPOOL`)
	if err != nil || removed.FromRecord != rec.Id {
		t.Fatalf("remove by name: %+v %v", removed, err)
	}
	removed, err = RemoveCorpusEntry(`interior-cheap`, strings.ToLower(other.Id))
	if err != nil || removed.Name != `Carved Walnut Button` {
		t.Fatalf("remove by record id: %+v %v", removed, err)
	}
	if _, promoted := CorpusCounts(); promoted != 0 {
		t.Fatal("removed from memory")
	}
	data, _ := os.ReadFile(overlayPath)
	if strings.Contains(string(data), rec.Id) || strings.Contains(string(data), other.Id) {
		t.Fatal("and from disk")
	}
}

// Two overlay entries with one name (only a hand edit of the file can do
// that) are never removed by guesswork.
func TestRemoveCorpusEntryRefusesAnAmbiguousName(t *testing.T) {
	withCatalog(t)
	twice := `entries:
  interior-cheap:
    - name: Painted Wooden Spool
      name_simple: spool
      description: A wooden thread spool painted with a band of faded blue.
      weight_lbs: 0.2
      value: 4
      from_record: B0000007
      promoted_at: 2026-09-28T12:00:00Z
    - name: Painted Wooden Spool
      name_simple: spool
      description: A wooden thread spool painted with a band of faded blue.
      weight_lbs: 0.2
      value: 4
      from_record: B0000008
      promoted_at: 2026-09-28T12:00:00Z
`
	withCorpus(t, testSeed, twice)
	if _, err := RemoveCorpusEntry(`interior-cheap`, `Painted Wooden Spool`); err == nil {
		t.Fatal("two entries share the name: name the record instead")
	}
	if removed, err := RemoveCorpusEntry(`interior-cheap`, `B0000008`); err != nil || removed.FromRecord != `B0000008` {
		t.Fatalf("by record id: %+v %v", removed, err)
	}
}

// An overlay that could not be read or moved aside is never written: a
// save would replace entries the pool never saw. Retire still retires.
func TestABrokenOverlayRefusesEveryWrite(t *testing.T) {
	withCatalog(t)
	quarantineOverlay = func(string) (string, error) { return ``, errors.New(`disk says no`) }
	t.Cleanup(func() { quarantineOverlay = util.QuarantineCorrupt })
	corrupt := "entries: {this is: [not closed\n"
	_, overlayPath, rep := withCorpus(t, testSeed, corrupt)
	if !rep.OverlayBroken {
		t.Fatalf("fixture: %+v", rep)
	}
	rec := seedRecord(t, promotable())
	if _, err := Promote(rec.Id); !errors.Is(err, ErrOverlayBroken) {
		t.Fatalf("promote: %v", err)
	}
	if _, err := RemoveCorpusEntry(`interior-cheap`, `Painted Wooden Spool`); !errors.Is(err, ErrOverlayBroken) {
		t.Fatalf("remove: %v", err)
	}
	err := Retire(rec.Id, `Admin`)
	if !errors.Is(err, ErrCorpusCleanup) || !errors.Is(err, ErrOverlayBroken) {
		t.Fatalf("retire reports the overlay it could not clean: %v", err)
	}
	if got, _ := Get(rec.Id); got.Status != StatusRetired {
		t.Fatal("and retires the record all the same")
	}
	if data, err := os.ReadFile(overlayPath); err != nil || string(data) != corrupt {
		t.Fatalf("the file is untouched: %v", err)
	}
}

// An overlay entry that fails its checks is not used, but a save keeps it.
func TestAnUnusableOverlayEntryIsKeptOnSave(t *testing.T) {
	withCatalog(t)
	_, overlayPath, rep := withCorpus(t, testSeed, `entries:
  interior-cheap:
    - name: Stone Idol Head
      name_simple: idol
      description: A stone head broken from a small idol, far heavier than it looks.
      weight_lbs: 40
      value: 3
      from_record: B0000009
      promoted_at: 2026-09-28T12:00:00Z
`)
	if rep.Promoted != 0 || len(rep.Skipped) != 1 {
		t.Fatalf("report: %+v", rep)
	}
	rec := seedRecord(t, promotable())
	if _, err := Promote(rec.Id); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(overlayPath)
	if !strings.Contains(string(data), `Stone Idol Head`) || !strings.Contains(string(data), rec.Id) {
		t.Fatalf("both entries are saved:\n%s", data)
	}
	l := CorpusList(`interior-cheap`)
	if len(l.Promoted) != 2 || l.Unused[1] == `` {
		t.Fatalf("listed, with why the first is unused: %+v", l)
	}
}

// The overlay lives in the catalog's directory. The catalog loader reads
// only catalog-* files, so reloading, writing, retrying shards and pruning
// the catalog never touch it.
func TestOverlaySurvivesTheCatalog(t *testing.T) {
	dir := withCatalog(t)
	seedPath := filepath.Join(t.TempDir(), seedFileName)
	writeTestFile(t, seedPath, testSeed)
	overlayPath := filepath.Join(dir, overlayFileName)
	LoadCorpusFrom(seedPath, overlayPath)
	t.Cleanup(ClearCorpusForTest)
	if _, err := Promote(seedRecord(t, promotable()).Id); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(overlayPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := loadFrom(dir); err != nil {
		t.Fatal(err)
	}
	if Count() != 1 {
		t.Fatalf("the catalog read its one record and nothing else, got %d", Count())
	}
	another, _ := Create(Record{Name: `Another Find`, Generator: GeneratorLocal})
	SaveAll()
	// The catalog prune (applySweep): a record goes only after
	// minUnseenSweeps sweeps in a row found nothing pointing at it and keep
	// has passed since its last evidence, which the first such sweep moves
	// to now. So sweep that often with no refs and no keep window; both
	// records go, rewriting their shard in the overlay's directory.
	now := time.Now().Add(time.Minute)
	pruned := 0
	for i := 0; i < minUnseenSweeps; i++ {
		_, n, shardErrors := applySweep(now, map[string]bool{}, 0)
		if shardErrors != 0 {
			t.Fatalf("sweep %d: %d shard writes failed", i+1, shardErrors)
		}
		pruned += n
	}
	if _, ok := Get(another.Id); ok || pruned != 2 {
		t.Fatalf("the prune removes both records: pruned %d, the seeded one still there: %v", pruned, ok)
	}

	after, err := os.ReadFile(overlayPath)
	if err != nil || string(after) != string(before) {
		t.Fatalf("the overlay is untouched: %v", err)
	}
	if aside, _ := filepath.Glob(filepath.Join(dir, overlayFileName+`.corrupt-*`)); len(aside) != 0 {
		t.Fatalf("never quarantined as a shard: %v", aside)
	}
	LoadCorpusFrom(seedPath, overlayPath)
	if _, promoted := CorpusCounts(); promoted != 1 {
		t.Fatal("and still loads")
	}
}

// Export prints the promoted entries in the seed's format (no provenance),
// ready to paste into bauble-corpus.yaml.
func TestExportPromotedIsSeedFormat(t *testing.T) {
	withCatalog(t)
	withCorpus(t, testSeed, ``)
	if _, err := Promote(seedRecord(t, promotable()).Id); err != nil {
		t.Fatal(err)
	}
	out, err := ExportPromoted()
	if err != nil {
		t.Fatal(err)
	}
	var doc seedDoc
	if err := decodeStrict([]byte(out), &doc); err != nil {
		t.Fatalf("parses strictly as a seed: %v\n%s", err, out)
	}
	if l := doc.Entries[`interior-cheap`]; len(l) != 1 || l[0].Name != `Painted Wooden Spool` {
		t.Fatalf("export: %+v", doc)
	}
}
