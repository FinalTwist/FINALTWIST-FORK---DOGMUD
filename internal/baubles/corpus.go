package baubles

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v3"
)

// The fallback corpus (docs/superpowers/specs/2026-09-28-baubles-hardening-
// and-corpus-design.md, slice C): hand-written bauble text, keyed by where a
// find was made, used whenever no model names it. Two layers:
//
//   - the seed, <DataFiles>/bauble-corpus.yaml: tracked, authored content.
//   - the overlay, <DataFiles>/baubles/corpus.promoted.yaml: living state,
//     model names an admin promoted (`bauble promote`, corpus_admin.go).
//
// Both are read into one immutable pool behind an atomic pointer, so
// Fallback reads it off the mud lock. Writers build a new pool, save the
// overlay, and only then swap the pointer (persist before publish).

const (
	seedFileName    = `bauble-corpus.yaml`
	overlayFileName = `corpus.promoted.yaml`
	// pocketPrefix keys the pools for pickpocketed finds: pocket-<tier>.
	pocketPrefix = `pocket`
	// fallbackRecentNames is how many of a zone's newest named finds a
	// fallback avoids repeating.
	fallbackRecentNames = 12
)

// CorpusEntry is one piece of bauble text, in the shape a model answers.
type CorpusEntry struct {
	Name        string  `yaml:"name"`
	NameSimple  string  `yaml:"name_simple"`
	Description string  `yaml:"description"`
	Material    string  `yaml:"material,omitempty"`
	WeightLbs   float64 `yaml:"weight_lbs"`
	Value       int     `yaml:"value"`
}

func (e CorpusEntry) reply() Reply {
	return Reply{Name: e.Name, NameSimple: e.NameSimple, Description: e.Description, Material: e.Material, WeightLbs: e.WeightLbs, Value: e.Value}
}

func entryFrom(r Reply) CorpusEntry {
	return CorpusEntry{Name: r.Name, NameSimple: r.NameSimple, Description: r.Description, Material: r.Material, WeightLbs: r.WeightLbs, Value: r.Value}
}

// PromotedEntry is an overlay entry: the text, plus where it came from.
type PromotedEntry struct {
	CorpusEntry `yaml:",inline"`
	// FromRecord is informational: it dangles once the catalog prunes the
	// record. Retire removes the entries that name it.
	FromRecord    string    `yaml:"from_record"`
	Zone          string    `yaml:"zone,omitempty"`
	Biome         string    `yaml:"biome,omitempty"`
	Model         string    `yaml:"model,omitempty"`
	PromptVersion int       `yaml:"prompt_version,omitempty"`
	PromotedAt    time.Time `yaml:"promoted_at"`
}

// seedDoc is the seed file (and the export format).
type seedDoc struct {
	Groups  map[string]string        `yaml:"groups,omitempty"` // biome -> group
	Entries map[string][]CorpusEntry `yaml:"entries"`
}

// overlayDoc is the overlay file.
type overlayDoc struct {
	Entries map[string][]PromotedEntry `yaml:"entries"`
}

func normKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// parseCorpusKey splits "interior-cheap" into ("interior", cheap) and a
// bare tier "cheap" into ("", cheap). Biome ids use underscores, never
// hyphens, so the last hyphen is the split.
func parseCorpusKey(key string) (prefix string, tier ValueTier, ok bool) {
	key = normKey(key)
	if t, ok := ParseTier(key); ok {
		return ``, t, true
	}
	i := strings.LastIndex(key, `-`)
	if i <= 0 {
		return ``, ``, false
	}
	t, ok := ParseTier(key[i+1:])
	if !ok {
		return ``, ``, false
	}
	return key[:i], t, true
}

// corpusKey is parseCorpusKey's inverse.
func corpusKey(prefix string, tier ValueTier) string {
	if prefix == `` {
		return string(tier)
	}
	return prefix + `-` + string(tier)
}

// decodeStrict decodes YAML refusing unknown fields, so a typo in a field
// name fails loudly. An empty document decodes to nothing.
func decodeStrict(data []byte, out any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// checkEntry runs an entry through what a model's answer must pass
// (CleanReply, which also refuses an authored item's name since slice H)
// and requires a weight already inside the bauble bounds, in tenths of a
// pound. It returns the cleaned entry. Values are not checked here: they
// are clamped into the tier when used.
func checkEntry(e CorpusEntry) (CorpusEntry, error) {
	cleaned, err := CleanReply(e.reply())
	if err != nil {
		return CorpusEntry{}, err
	}
	if ClampWeight(e.WeightLbs) != e.WeightLbs {
		return CorpusEntry{}, fmt.Errorf(`%w: weight %v lb is not a tenth of a pound from %.1f to %.1f`, ErrUnusableReply, e.WeightLbs, MinWeightLbs, MaxWeightLbs)
	}
	return entryFrom(cleaned), nil
}

// promotedSlot is one overlay entry as loaded. raw is what is saved, exactly
// as read or promoted; use is the checked text. An entry that fails the
// checks (ok false) is kept and saved again, never used: the overlay is
// living state, and a rewrite must not drop it.
type promotedSlot struct {
	raw PromotedEntry
	use CorpusEntry
	ok  bool
	why string
}

// corpusPool is one immutable snapshot of the corpus.
type corpusPool struct {
	seedPath    string
	overlayPath string
	groups      map[string]string        // biome -> group
	seed        map[string][]CorpusEntry // key -> checked entries
	promoted    map[string][]promotedSlot
	// overlayBroken: the overlay file could not be read and could not be
	// moved aside either, so it still sits where a save would write. Every
	// overlay writer refuses (ErrOverlayBroken) until a reload clears it.
	overlayBroken bool
}

var (
	corpus atomic.Pointer[corpusPool]
	// corpusWriteMu serialises the writers (load, promote, remove, retire,
	// edit, regen). They also run under the mud lock; this keeps tests and
	// any future caller honest. Readers never take it.
	corpusWriteMu sync.Mutex
	// quarantineOverlay moves a corrupt overlay aside. A variable so a test
	// can make it fail.
	quarantineOverlay = util.QuarantineCorrupt
)

// knownPrefix reports a key prefix a pool may have: a biome in groups, a
// group, pocket, or none (a bare tier).
func (p *corpusPool) knownPrefix(prefix string) bool {
	if prefix == `` || prefix == pocketPrefix {
		return true
	}
	if _, ok := p.groups[prefix]; ok {
		return true
	}
	for _, g := range p.groups {
		if g == prefix {
			return true
		}
	}
	return false
}

// CorpusReport is what a load found.
type CorpusReport struct {
	Seed          int      // seed entries in use
	Promoted      int      // overlay entries in use
	Skipped       []string // one line per entry or key not used, and why
	SeedErr       error    // the seed could not be read or parsed
	SeedKept      bool     // SeedErr on a reload: the seed already in use was kept
	Quarantined   string   // where a corrupt overlay was moved
	OverlayBroken bool     // a corrupt overlay could not be moved aside: no writes until a reload
}

// LoadCorpus reads the seed and the overlay of the configured world. Call
// after items.LoadDataFiles (entries are checked against the authored item
// names) and after Load (the overlay sits in the catalog's directory). It
// never fails: a broken seed is logged at ERROR and the corpus runs
// without it (a reload keeps the seed already in use); a corrupt overlay is
// quarantined and starts empty, or, when it cannot be moved aside, is left
// in place and marked broken so nothing overwrites it.
func LoadCorpus() CorpusReport {
	return LoadCorpusFrom(
		util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, seedFileName),
		util.FilePath(catalogDir(), `/`, overlayFileName),
	)
}

// ReloadCorpus reads the loaded corpus's own files again (the admin
// `bauble corpus reload`), or the configured world's when none is loaded.
func ReloadCorpus() CorpusReport {
	if p := corpus.Load(); p != nil {
		return LoadCorpusFrom(p.seedPath, p.overlayPath)
	}
	return LoadCorpus()
}

// LoadCorpusFrom is LoadCorpus from these two files. Tests use it.
func LoadCorpusFrom(seedPath, overlayPath string) CorpusReport {
	corpusWriteMu.Lock()
	defer corpusWriteMu.Unlock()
	pool, rep := readCorpus(seedPath, overlayPath, corpus.Load())
	corpus.Store(pool)
	for _, s := range rep.Skipped {
		mudlog.Warn(`baubles.LoadCorpus`, `skipped`, s)
	}
	if rep.SeedErr != nil {
		mudlog.Error(`baubles.LoadCorpus`, `seed`, seedPath, `error`, rep.SeedErr, `keptSeedInUse`, rep.SeedKept)
	}
	mudlog.Info(`baubles.LoadCorpus()`, `seed`, rep.Seed, `promoted`, rep.Promoted, `skipped`, len(rep.Skipped))
	return rep
}

// ClearCorpusForTest empties the corpus: every fallback is a generic
// trinket again.
func ClearCorpusForTest() {
	corpus.Store(nil)
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// readCorpus reads both files into a new pool. prev is the pool in use (nil
// at boot): when the seed cannot be read and prev was read from the same
// seed file, prev's groups and seed entries are kept.
func readCorpus(seedPath, overlayPath string, prev *corpusPool) (*corpusPool, CorpusReport) {
	pool := &corpusPool{
		seedPath:    seedPath,
		overlayPath: overlayPath,
		groups:      map[string]string{},
		seed:        map[string][]CorpusEntry{},
		promoted:    map[string][]promotedSlot{},
	}
	var rep CorpusReport
	seedName := filepath.Base(seedPath)

	// The seed: authored content. Missing means a world with no corpus.
	var sd seedDoc
	if data, err := os.ReadFile(seedPath); err == nil {
		if err := decodeStrict(data, &sd); err != nil {
			rep.SeedErr = err
			sd = seedDoc{}
		}
	} else if !os.IsNotExist(err) {
		rep.SeedErr = err
	}
	if rep.SeedErr != nil && prev != nil && prev.seedPath == seedPath {
		// A reload that cannot read the seed keeps the seed in use: a typo
		// in a hand edit must not empty every pool until the next reload.
		// The maps are never written after a pool is built, so sharing them
		// is safe. The overlay below is still read again.
		pool.groups, pool.seed = prev.groups, prev.seed
		for _, list := range prev.seed {
			rep.Seed += len(list)
		}
		rep.SeedKept = true
	}
	for _, biome := range sortedKeys(sd.Groups) {
		b, g := normKey(biome), normKey(sd.Groups[biome])
		if b == `` || g == `` || b == pocketPrefix || g == pocketPrefix {
			rep.Skipped = append(rep.Skipped, fmt.Sprintf(`%s groups %q: %q is not a usable group`, seedName, biome, sd.Groups[biome]))
			continue
		}
		pool.groups[b] = g
	}
	for _, key := range sortedKeys(sd.Entries) {
		k := normKey(key)
		if prefix, _, ok := parseCorpusKey(k); !ok || !pool.knownPrefix(prefix) {
			rep.Skipped = append(rep.Skipped, fmt.Sprintf(`%s %q: not a biome, group, pocket or tier key`, seedName, key))
			continue
		}
		for i, e := range sd.Entries[key] {
			cleaned, err := checkEntry(e)
			if err != nil {
				rep.Skipped = append(rep.Skipped, fmt.Sprintf(`%s %s #%d %q: %v`, seedName, k, i+1, e.Name, err))
				continue
			}
			pool.seed[k] = append(pool.seed[k], cleaned)
			rep.Seed++
		}
	}

	// The overlay: living state. Absent is empty; unreadable or
	// unparseable is quarantined, never treated as absent.
	var od overlayDoc
	raw, err := util.ReadLivingState(overlayPath)
	if err == nil {
		err = decodeStrict(raw, &od)
	}
	if err != nil && !errors.Is(err, util.ErrStateAbsent) {
		moved, qErr := quarantineOverlay(overlayPath)
		rep.Quarantined = moved
		if qErr != nil {
			// The bad file is still where a save would write, and a save
			// would replace entries this pool never saw. Refuse every
			// overlay write until a reload can read it or move it aside.
			pool.overlayBroken = true
			rep.OverlayBroken = true
		}
		mudlog.Error(`baubles.LoadCorpus`, `action`, `quarantined corrupt overlay`, `error`, err, `quarantinedTo`, moved, `quarantineError`, qErr, `overlayBroken`, rep.OverlayBroken)
		od = overlayDoc{}
	}
	for _, key := range sortedKeys(od.Entries) {
		k := normKey(key)
		prefix, _, keyOk := parseCorpusKey(k)
		for i, e := range od.Entries[key] {
			slot := promotedSlot{raw: e}
			if !keyOk || !pool.knownPrefix(prefix) {
				slot.why = `not a biome, group, pocket or tier key`
			} else if cleaned, err := checkEntry(e.CorpusEntry); err != nil {
				slot.why = err.Error()
			} else {
				slot.use, slot.ok = cleaned, true
				rep.Promoted++
			}
			if !slot.ok {
				rep.Skipped = append(rep.Skipped, fmt.Sprintf(`%s %s #%d %q: %s`, overlayFileName, k, i+1, e.Name, slot.why))
			}
			pool.promoted[k] = append(pool.promoted[k], slot)
		}
	}
	return pool, rep
}

// GroupOf is the corpus group of a biome, if it has one.
func GroupOf(biome string) (string, bool) {
	p := corpus.Load()
	if p == nil {
		return ``, false
	}
	g, ok := p.groups[normKey(biome)]
	return g, ok
}

// CorpusCounts is how many seed and promoted entries are in use.
func CorpusCounts() (seed int, promoted int) {
	p := corpus.Load()
	if p == nil {
		return 0, 0
	}
	for _, list := range p.seed {
		seed += len(list)
	}
	for _, slots := range p.promoted {
		for _, s := range slots {
			if s.ok {
				promoted++
			}
		}
	}
	return seed, promoted
}
