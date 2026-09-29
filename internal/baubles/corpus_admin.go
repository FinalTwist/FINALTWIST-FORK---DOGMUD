package baubles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v3"
)

// Corpus administration: promoting a model's name into the overlay,
// removing one, and what Retire, Edit and ApplyRegenerated need. The admin
// `bauble promote` and `bauble corpus` commands call these, and so will the
// /build queue (web builder rework arc). Callers hold the mud lock;
// corpusWriteMu serialises them as well.

var (
	ErrNoCorpus           = errors.New(`the fallback corpus is not loaded`)
	ErrOverlayBroken      = errors.New(`the promoted file could not be read or set aside, so nothing is written to it until "bauble corpus reload" succeeds`)
	ErrCorpusCleanup      = errors.New(`the record was changed, but the fallback corpus entries promoted from it could not be removed`)
	ErrPromoteNotModel    = errors.New(`only a find named by the model can be promoted`)
	ErrPromotePlayerKey   = errors.New(`it was named on a player's own key`)
	ErrPromoteUnmoderated = errors.New(`its text was never passed by moderation`)
	ErrPromoteEdited      = errors.New(`an admin has changed its text by hand`)
	ErrPromoteRetired     = errors.New(`it is retired`)
	ErrPromoteNoGroup     = errors.New(`it was found in a biome with no corpus group`)
	ErrPromoteTooBig      = errors.New(`it is too big for a pocket`)
	ErrPromoteDuplicate   = errors.New(`it is already in the corpus`)
	ErrPromoteNameTaken   = errors.New(`its pool already has a find by that name`)
)

// promotionKey is the overlay key a record goes under: pocket-<tier> for a
// pickpocketed find, else <biome>-<tier>, and only for a biome with a group
// (lookup skips a biome without one).
func (p *corpusPool) promotionKey(rec Record) (string, error) {
	if rec.Source == SourcePickpocket {
		return corpusKey(pocketPrefix, rec.Tier), nil
	}
	b := normKey(rec.Biome)
	if _, ok := p.groups[b]; b == `` || !ok {
		return ``, ErrPromoteNoGroup
	}
	return corpusKey(b, rec.Tier), nil
}

// mergedKeys is key and every key Fallback merges with it into one pool: a
// biome's key and its group's. A pocket or group key stands alone.
func (p *corpusPool) mergedKeys(key string) []string {
	keys := []string{key}
	if prefix, tier, ok := parseCorpusKey(key); ok {
		if g, ok := p.groups[prefix]; ok && g != prefix {
			keys = append(keys, corpusKey(g, tier))
		}
	}
	return keys
}

// nameInPool reports a seed or overlay entry (used or not) called name in
// the pool key belongs to.
func (p *corpusPool) nameInPool(key, name string) bool {
	n := normKey(name)
	for _, k := range p.mergedKeys(key) {
		for _, e := range p.seed[k] {
			if normKey(e.Name) == n {
				return true
			}
		}
		for _, s := range p.promoted[k] {
			if normKey(s.raw.Name) == n {
				return true
			}
		}
	}
	return false
}

// withPromoted is a copy of the pool with change applied to a copy of its
// overlay. The pool itself is never changed: readers may hold it.
func (p *corpusPool) withPromoted(change func(m map[string][]promotedSlot)) *corpusPool {
	next := *p
	next.promoted = make(map[string][]promotedSlot, len(p.promoted))
	for k, v := range p.promoted {
		next.promoted[k] = append([]promotedSlot(nil), v...)
	}
	change(next.promoted)
	for k, v := range next.promoted {
		if len(v) == 0 {
			delete(next.promoted, k)
		}
	}
	return &next
}

// saveOverlay writes every overlay entry, used or not, through util.Save.
func saveOverlay(path string, promoted map[string][]promotedSlot) error {
	doc := overlayDoc{Entries: map[string][]PromotedEntry{}}
	for k, slots := range promoted {
		for _, s := range slots {
			doc.Entries[k] = append(doc.Entries[k], s.raw)
		}
	}
	data, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return util.Save(path, data)
}

// Promote copies a record's text into the overlay, under its exact
// <biome>-<tier> or pocket-<tier> key, with its provenance. Only text the
// model wrote on the server's key and moderation passed, never hand-edited
// (HandEdited; EditedBy alone is fine, since Retire, Restore and regen set
// it without writing text) and not retired (a sold record is fine), and
// only a name its pool does not already have. It returns the key.
func Promote(id string) (string, error) {
	corpusWriteMu.Lock()
	defer corpusWriteMu.Unlock()
	p := corpus.Load()
	if p == nil {
		return ``, ErrNoCorpus
	}
	if p.overlayBroken {
		return ``, ErrOverlayBroken
	}
	rec, ok := Get(id)
	if !ok {
		return ``, ErrNoRecord
	}
	switch {
	case rec.Status == StatusRetired:
		return ``, ErrPromoteRetired
	case rec.Generator != GeneratorOpenAI:
		return ``, ErrPromoteNotModel
	case rec.PlayerKey:
		return ``, ErrPromotePlayerKey
	case rec.HandEdited: // Edit sets it and leaves Moderated: an edit is refused as an edit
		return ``, ErrPromoteEdited
	case !rec.Moderated:
		return ``, ErrPromoteUnmoderated
	case !rec.Tier.Valid():
		return ``, fmt.Errorf(`its tier %q is not a tier`, rec.Tier)
	}
	key, err := p.promotionKey(rec)
	if err != nil {
		return ``, err
	}
	for _, slots := range p.promoted {
		for _, s := range slots {
			if s.raw.FromRecord == rec.Id {
				return ``, ErrPromoteDuplicate
			}
		}
	}
	if p.nameInPool(key, rec.Name) {
		return ``, ErrPromoteNameTaken
	}
	entry := CorpusEntry{Name: rec.Name, NameSimple: rec.NameSimple, Description: rec.Description, Material: rec.Material, WeightLbs: rec.WeightLbs, Value: rec.Value}
	cleaned, err := checkEntry(entry)
	if err != nil {
		return ``, err
	}
	if rec.Source == SourcePickpocket && TooBigFor(cleaned.reply(), SourcePickpocket) {
		return ``, ErrPromoteTooBig
	}
	slot := promotedSlot{
		raw: PromotedEntry{
			CorpusEntry: entry, FromRecord: rec.Id, Zone: rec.Zone, Biome: rec.Biome,
			Model: rec.Model, PromptVersion: rec.PromptVersion, PromotedAt: time.Now().UTC(),
		},
		use: cleaned,
		ok:  true,
	}
	next := p.withPromoted(func(m map[string][]promotedSlot) { m[key] = append(m[key], slot) })
	if err := saveOverlay(next.overlayPath, next.promoted); err != nil {
		return ``, err
	}
	corpus.Store(next)
	return key, nil
}

// RemoveCorpusEntry removes one promoted entry from a key, named by its
// name (any case) or by the record it was promoted from (FromRecord), never
// by a position that shifts as entries come and go. A name two entries
// share is refused: name the record instead. Seed entries are tracked
// content: edit the file.
func RemoveCorpusEntry(key, which string) (PromotedEntry, error) {
	corpusWriteMu.Lock()
	defer corpusWriteMu.Unlock()
	p := corpus.Load()
	if p == nil {
		return PromotedEntry{}, ErrNoCorpus
	}
	if p.overlayBroken {
		return PromotedEntry{}, ErrOverlayBroken
	}
	k, w := normKey(key), normKey(which)
	at := -1
	for i, s := range p.promoted[k] {
		if normKey(s.raw.Name) != w && normKey(s.raw.FromRecord) != w {
			continue
		}
		if at >= 0 {
			return PromotedEntry{}, fmt.Errorf(`more than one promoted entry in %s matches %q; name its record instead (bauble corpus list %s)`, k, which, k)
		}
		at = i
	}
	if at < 0 {
		for _, e := range p.seed[k] {
			if normKey(e.Name) == w {
				return PromotedEntry{}, fmt.Errorf(`%q is a seed entry: edit bauble-corpus.yaml, then bauble corpus reload`, e.Name)
			}
		}
		return PromotedEntry{}, fmt.Errorf(`%s has no promoted entry called %q`, k, which)
	}
	removed := p.promoted[k][at].raw
	next := p.withPromoted(func(m map[string][]promotedSlot) {
		s := m[k]
		m[k] = append(append([]promotedSlot(nil), s[:at]...), s[at+1:]...)
	})
	if err := saveOverlay(next.overlayPath, next.promoted); err != nil {
		return PromotedEntry{}, err
	}
	corpus.Store(next)
	return removed, nil
}

// removePromotedFrom removes every overlay entry promoted from recordId
// (Retire, Edit, ApplyRegenerated). It returns how many went. With the
// overlay broken it cannot know what the file holds, so it refuses.
func removePromotedFrom(recordId string) (int, error) {
	corpusWriteMu.Lock()
	defer corpusWriteMu.Unlock()
	p := corpus.Load()
	if p == nil {
		return 0, nil
	}
	if p.overlayBroken {
		return 0, ErrOverlayBroken
	}
	count := 0
	next := p.withPromoted(func(m map[string][]promotedSlot) {
		for k, slots := range m {
			kept := slots[:0]
			for _, s := range slots {
				if s.raw.FromRecord == recordId {
					count++
					continue
				}
				kept = append(kept, s)
			}
			m[k] = kept
		}
	})
	if count == 0 {
		return 0, nil
	}
	if err := saveOverlay(next.overlayPath, next.promoted); err != nil {
		return 0, err
	}
	corpus.Store(next)
	return count, nil
}

// CorpusKeyCount is one row of CorpusKeys.
type CorpusKeyCount struct {
	Key      string
	Seed     int
	Promoted int // every overlay entry, used or not
}

// CorpusKeys lists every key with entries, sorted.
func CorpusKeys() []CorpusKeyCount {
	p := corpus.Load()
	if p == nil {
		return nil
	}
	counts := map[string]*CorpusKeyCount{}
	get := func(k string) *CorpusKeyCount {
		c, ok := counts[k]
		if !ok {
			c = &CorpusKeyCount{Key: k}
			counts[k] = c
		}
		return c
	}
	for k, list := range p.seed {
		get(k).Seed = len(list)
	}
	for k, slots := range p.promoted {
		get(k).Promoted = len(slots)
	}
	out := make([]CorpusKeyCount, 0, len(counts))
	for _, c := range counts {
		out = append(out, *c)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Key < out[b].Key })
	return out
}

// CorpusListing is one key's entries.
type CorpusListing struct {
	Seed     []CorpusEntry
	Promoted []PromotedEntry // in overlay order
	Unused   map[int]string  // position in Promoted, from 1 -> why it is not used
}

// CorpusList lists one key's entries.
func CorpusList(key string) CorpusListing {
	out := CorpusListing{Unused: map[int]string{}}
	p := corpus.Load()
	if p == nil {
		return out
	}
	k := normKey(key)
	out.Seed = append(out.Seed, p.seed[k]...)
	for i, s := range p.promoted[k] {
		out.Promoted = append(out.Promoted, s.raw)
		if !s.ok {
			out.Unused[i+1] = s.why
		}
	}
	return out
}

// ExportPromoted renders the usable promoted entries in the seed's format
// (no provenance), to copy into bauble-corpus.yaml.
func ExportPromoted() (string, error) {
	doc := seedDoc{Entries: map[string][]CorpusEntry{}}
	if p := corpus.Load(); p != nil {
		for k, slots := range p.promoted {
			for _, s := range slots {
				if s.ok {
					doc.Entries[k] = append(doc.Entries[k], s.raw.CorpusEntry)
				}
			}
		}
	}
	data, err := yaml.Marshal(doc)
	return string(data), err
}
