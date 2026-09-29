package baubles

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

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
