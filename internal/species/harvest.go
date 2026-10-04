package species

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// HarvestEntry is one thing a carcass can give: a material named by its
// component_tag, how many for a MEDIUM body, which tool is needed to take it,
// and whether it is a rare part that only a sharp eye notices.
//
//   - {item: raw-meat, qty: 2, tool: knife}
//   - {item: bone,     qty: 2, tool: cleaver}
//   - {item: fang,     qty: 1, tool: bone_saw, rare: true}
//
// Tool defaults to knife when empty: every carcass job needs a blade.
type HarvestEntry struct {
	Item string         `yaml:"item"`
	Qty  int            `yaml:"qty"`
	Tool items.ToolType `yaml:"tool,omitempty"`
	Rare bool           `yaml:"rare,omitempty"`
}

// ToolOrDefault is the entry's tool, or knife when none is authored.
func (e HarvestEntry) ToolOrDefault() items.ToolType {
	if e.Tool == `` {
		return items.ToolKnife
	}
	return e.Tool
}

// HarvestTable is what a carcass gives to `skin` and to `butcher`
// (wilderness-trades plan, phase 2). It is authored on a species, and a mob
// may author its own to override the species per section: a mob's non-empty
// Skin replaces the species Skin, and likewise Butcher. That is how a unique
// pelt (the Cascade Pass predators' thick pelt) replaces a plain wolf hide
// without restating the meat and bone.
type HarvestTable struct {
	Skin    []HarvestEntry `yaml:"skin,omitempty"`
	Butcher []HarvestEntry `yaml:"butcher,omitempty"`
}

// Empty reports whether the table gives nothing at all. A nil table is empty.
func (h *HarvestTable) Empty() bool {
	return h == nil || (len(h.Skin) == 0 && len(h.Butcher) == 0)
}

// MergeHarvest resolves a mob's harvest: each section of override that is
// non-empty replaces the same section of base. Either argument may be nil.
// The result is a fresh value; neither input is modified.
func MergeHarvest(base, override *HarvestTable) HarvestTable {
	out := HarvestTable{}
	if base != nil {
		out.Skin = append([]HarvestEntry(nil), base.Skin...)
		out.Butcher = append([]HarvestEntry(nil), base.Butcher...)
	}
	if override != nil {
		if len(override.Skin) > 0 {
			out.Skin = append([]HarvestEntry(nil), override.Skin...)
		}
		if len(override.Butcher) > 0 {
			out.Butcher = append([]HarvestEntry(nil), override.Butcher...)
		}
	}
	return out
}

// ScaleHarvestQty scales an authored (medium-body) quantity by body size:
// small halves it, large doubles it. A positive quantity never scales below
// one: a hare still has a pelt.
func ScaleHarvestQty(qty int, size Size) int {
	if qty <= 0 {
		return 0
	}
	f := float64(qty)
	switch size {
	case Small:
		f *= 0.5
	case Large:
		f *= 2
	}
	n := int(math.Round(f))
	if n < 1 {
		n = 1
	}
	return n
}

// Validate checks every entry: a known material tag, a positive quantity and
// a known tool type. tagExists is a callback so this package does not need
// the item registry loaded (pass a func over items.FindSpecByComponentTag).
func (h *HarvestTable) Validate(tagExists func(tag string) bool) error {
	if h == nil {
		return nil
	}
	for section, entries := range map[string][]HarvestEntry{`skin`: h.Skin, `butcher`: h.Butcher} {
		for i, e := range entries {
			if e.Item == `` {
				return fmt.Errorf("harvest %s[%d]: no item tag", section, i)
			}
			if e.Qty <= 0 {
				return fmt.Errorf("harvest %s[%d] %q: qty must be positive, got %d", section, i, e.Item, e.Qty)
			}
			if e.Tool != `` && !items.IsKnownToolType(e.Tool) {
				return fmt.Errorf("harvest %s[%d] %q: unknown tool %q", section, i, e.Item, e.Tool)
			}
			if tagExists != nil && !tagExists(e.Item) {
				return fmt.Errorf("harvest %s[%d]: no item carries component_tag %q", section, i, e.Item)
			}
		}
	}
	return nil
}

// ValidateSpeciesHarvest panics on any species whose harvest table names an
// unknown material, tool or quantity. Called from main after species and items
// are loaded, the same shape as ValidateSpeciesConditionIds. Panicking at boot
// is the point: a harvest entry that silently yields nothing is the bug the
// phase 0 corpse fix was cleaning up.
func ValidateSpeciesHarvest(tagExists func(tag string) bool) {
	for _, sp := range allSpecies {
		if err := sp.Harvest.Validate(tagExists); err != nil {
			panic(fmt.Sprintf("species %q (id %d): %v", sp.Name, sp.SpeciesId, err))
		}
	}
}
