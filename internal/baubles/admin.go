package baubles

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
)

// Catalog administration (Phase 5): statistics, retiring and restoring a
// record's text, hand edits, and a look at the prompt a find would send.
// Used by the admin `bauble` command (internal/usercommands/admin.bauble.go).
// Every change here reaches items already in the world at once, because
// items read their text from the catalog.

// Stats is a summary of the whole catalog.
type Stats struct {
	Total       int
	ByStatus    map[Status]int
	ByGenerator map[Generator]int
	ByTier      map[ValueTier]int
	TopRegions  []RegionCount // most finds first, at most 10
	Unsold      int           // records not sold and not vanished (still in the world, or lost in a save)
	UnsoldValue int           // their catalog value, in gold
	Household   int           // finds left in a household
	Stolen      int           // records marked stolen
	Vanished    int           // left untaken until they vanished
	Tokens      int           // model tokens spent on names, all time
	Edited      int           // records an admin has changed by hand
}

// RegionCount is one row of Stats.TopRegions.
type RegionCount struct {
	Region string
	Count  int
}

// CatalogStats summarises the catalog.
func CatalogStats() Stats {
	cat.mu.RLock()
	defer cat.mu.RUnlock()

	st := Stats{
		ByStatus:    map[Status]int{},
		ByGenerator: map[Generator]int{},
		ByTier:      map[ValueTier]int{},
	}
	regions := map[string]int{}
	for _, r := range cat.records {
		st.Total++
		st.ByStatus[r.Status]++
		st.ByGenerator[r.Generator]++
		st.ByTier[r.Tier]++
		regions[r.Region]++
		st.Tokens += r.Tokens
		if r.EditedBy != `` {
			st.Edited++
		}
		if r.Household {
			st.Household++
		}
		if r.Stolen {
			st.Stolen++
		}
		if !r.VanishedAt.IsZero() {
			st.Vanished++
		}
		if r.Status != StatusSold && r.VanishedAt.IsZero() {
			st.Unsold++
			st.UnsoldValue += r.Value
		}
	}
	for region, n := range regions {
		st.TopRegions = append(st.TopRegions, RegionCount{Region: region, Count: n})
	}
	sort.Slice(st.TopRegions, func(a, b int) bool {
		if st.TopRegions[a].Count != st.TopRegions[b].Count {
			return st.TopRegions[a].Count > st.TopRegions[b].Count
		}
		return st.TopRegions[a].Region < st.TopRegions[b].Region
	})
	if len(st.TopRegions) > 10 {
		st.TopRegions = st.TopRegions[:10]
	}
	return st
}

// ErrNoRecord is returned for an id the catalog does not hold.
var ErrNoRecord = errors.New(`no such bauble record`)

// Retire withdraws a record's text: every item pointing at it shows a plain
// "Trinket" until it is restored. Value, weight and provenance are kept.
// For a name or description that should not be in the game.
func Retire(id string, admin string) error {
	if _, ok := Update(id, func(r *Record) {
		r.Status = StatusRetired
		r.EditedBy = admin
	}); !ok {
		return ErrNoRecord
	}
	return nil
}

// Restore undoes Retire. A sold record stays sold.
func Restore(id string, admin string) error {
	if _, ok := Update(id, func(r *Record) {
		if r.Status != StatusRetired {
			return
		}
		r.Status = StatusFallback
		if r.Generator.Named() {
			r.Status = StatusReady
		}
		if r.SoldValue > 0 {
			r.Status = StatusSold
		}
		r.EditedBy = admin
	}); !ok {
		return ErrNoRecord
	}
	return nil
}

// EditFields are the fields an admin may change by hand.
var EditFields = []string{`name`, `keyword`, `desc`, `material`, `value`, `weight`, `tier`}

// Edit changes one field of a record by hand. Text goes through the same
// checks as a model's answer (CleanReply), so an edit cannot put markup in a
// name or a real item's keyword on a bauble. Value is clamped to the tier;
// changing the tier re-clamps the value into the new one. The record is
// marked as edited by admin.
func Edit(id string, field string, value string, admin string) (Record, error) {
	rec, ok := Get(id)
	if !ok {
		return Record{}, ErrNoRecord
	}
	value = strings.TrimSpace(value)
	reply := Reply{
		Name:        rec.Name,
		NameSimple:  rec.NameSimple,
		Description: rec.Description,
		Material:    rec.Material,
		WeightLbs:   rec.WeightLbs,
		Value:       rec.Value,
	}
	tier := rec.Tier

	switch strings.ToLower(field) {
	case `name`:
		reply.Name = value
	case `keyword`, `name_simple`, `namesimple`:
		reply.NameSimple = value
	case `desc`, `description`:
		reply.Description = value
	case `material`:
		reply.Material = value
	case `value`:
		n, err := strconv.Atoi(value)
		if err != nil {
			return Record{}, fmt.Errorf(`value must be a whole number of gold`)
		}
		reply.Value = n
	case `weight`:
		w, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return Record{}, fmt.Errorf(`weight must be a number of pounds`)
		}
		reply.WeightLbs = w
	case `tier`:
		t, ok := ParseTier(strings.ToLower(value))
		if !ok {
			return Record{}, fmt.Errorf(`tier must be cheap, average or rare`)
		}
		tier = t
	default:
		return Record{}, fmt.Errorf(`field must be one of: %s`, strings.Join(EditFields, `, `))
	}

	cleaned, err := CleanReply(reply)
	if err != nil {
		return Record{}, err
	}
	limited := ApplyLimitsFor(cleaned, tier, rec.Source)

	updated, ok := Update(id, func(r *Record) {
		r.Name = limited.Reply.Name
		r.NameSimple = limited.Reply.NameSimple
		r.Description = limited.Reply.Description
		r.Material = limited.Reply.Material
		r.Tier = limited.Tier
		r.Value = limited.Reply.Value
		r.WeightLbs = limited.Reply.WeightLbs
		r.EditedBy = admin
		// Hand-written text was never moderated as it now reads.
		r.Moderated = false
		r.HandEdited = true
	})
	if !ok {
		return Record{}, ErrNoRecord
	}
	return updated, nil
}

// ApplyRegenerated replaces a record's text and numbers with a fresh model
// answer (the admin `bauble regen` command). Provenance, status history and
// the theft fields are kept. It refuses a generic trinket: regenerating is
// for getting a NEW model name, and a failed call must not wipe one.
//
// randn picks a player-key find's rolled value, mirroring Mint (spec S3): a
// value a player's own key proposed is never trusted, even clamped, so the
// server rolls it instead and keeps the proposal in ValueProposed. Pass
// util.Rand in production, nil for the deterministic midpoint, a fixed
// function in tests.
func ApplyRegenerated(id string, res GenResult, admin string, randn func(n int) int) (Record, error) {
	if res.Generator != GeneratorOpenAI {
		return Record{}, errors.New(`the model did not answer; the record is unchanged`)
	}
	rec, ok := Get(id)
	if !ok {
		return Record{}, ErrNoRecord
	}
	limited := ApplyLimitsFor(res.Reply, rec.Tier, rec.Source)
	if res.PlayerKey {
		limited.Reply.Value = limited.Tier.RollValue(randn)
	}
	updated, ok := Update(id, func(r *Record) {
		r.Name = limited.Reply.Name
		r.NameSimple = limited.Reply.NameSimple
		r.Description = limited.Reply.Description
		r.Material = limited.Reply.Material
		r.Value = limited.Reply.Value
		r.ValueProposed = limited.ProposedValue
		r.WeightLbs = limited.Reply.WeightLbs
		r.WeightProposed = limited.ProposedWeight
		r.Generator = GeneratorOpenAI
		r.Model = res.Model
		r.PromptVersion = res.PromptVersion
		r.Tokens += res.Tokens
		r.Moderated = res.Moderated
		r.PlayerKey = res.PlayerKey
		if r.Status == StatusFallback || r.Status == StatusRetired {
			r.Status = StatusReady
		}
		r.EditedBy = admin + ` (regen)`
		// The text is the model's again, as moderation passed it.
		r.HandEdited = false
	})
	if !ok {
		return Record{}, ErrNoRecord
	}
	return updated, nil
}

// PromptPreview renders the messages a request would send, one string per
// message. modules/baubles installs it; nil when no module provides one.
type PromptPreview func(req GenRequest) []string

var promptPreview atomic.Pointer[PromptPreview]

// SetPromptPreview installs the prompt renderer. nil uninstalls it.
func SetPromptPreview(f PromptPreview) {
	if f == nil {
		promptPreview.Store(nil)
		return
	}
	promptPreview.Store(&f)
}

// PreviewPrompt renders req as it would be sent, or false when no renderer
// is installed.
func PreviewPrompt(req GenRequest) ([]string, bool) {
	p := promptPreview.Load()
	if p == nil || *p == nil {
		return nil, false
	}
	return (*p)(req), true
}

// LooksLikeId reports whether s is shaped like a catalog id (B and digits,
// any case), so admin commands can tell an id from an item name.
func LooksLikeId(s string) bool {
	_, ok := seqOf(strings.ToUpper(strings.TrimSpace(s)))
	return ok
}
