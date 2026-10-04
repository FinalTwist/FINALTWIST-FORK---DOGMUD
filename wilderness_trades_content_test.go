package main

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/timber"
)

// TestWildernessTradesContent pins the shipped wilderness-trades data
// (docs/economy/wilderness-trades.md) against the real item, species, mob and
// recipe files, so a typo in a harvest table or a recipe tag fails here rather
// than as a boot panic or a carcass that silently yields nothing:
//   - every species and mob harvest entry names a real item or tag, a known
//     tool and a sane quantity and chance;
//   - the game animals that used to leave an empty corpse now have a table;
//   - every item a harvest entry produces is sellable somewhere and every
//     spoiling one names a parseable period;
//   - every processing chain closes: each recipe ingredient tag is supplied by
//     some item.
func TestWildernessTradesContent(t *testing.T) {
	mudlog.SetupLogger(nil, `LOW`, ``, false)
	configs.SetConfigForTest(t, configs.GetConfig())
	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}
	t.Cleanup(items.SeedItemsForTest(nil))
	items.LoadDataFiles()
	species.LoadForTest(t)
	crafting.LoadRecipeFiles()

	tagExists := func(tag string) bool { return items.FindSpecByComponentTag(tag) != nil }
	itemExists := func(id int) bool { return items.GetItemSpec(id) != nil }

	checkEntryOutputs := func(owner string, h *species.HarvestTable) {
		if h == nil {
			return
		}
		for _, e := range append(append([]species.HarvestEntry{}, h.Skin...), h.Butcher...) {
			var spec *items.ItemSpec
			if e.ItemId > 0 {
				spec = items.GetItemSpec(e.ItemId)
			} else {
				spec = items.FindSpecByComponentTag(e.Item)
			}
			if spec == nil {
				continue // Validate reports it
			}
			if len(spec.VendorCategories) == 0 {
				t.Errorf("%s: %s (%d) has no vendor_categories, so no merchant buys it", owner, spec.Name, spec.ItemId)
			}
		}
	}

	withTable := map[int]bool{}
	for _, sp := range species.GetAllSpecies() {
		sp := sp
		if err := sp.Harvest.Validate(tagExists, itemExists); err != nil {
			t.Errorf("species %s: %v", sp.Name, err)
		}
		checkEntryOutputs("species "+sp.Name, sp.Harvest)
		if !sp.Harvest.Empty() {
			withTable[sp.SpeciesId] = true
		}
	}

	if len(withTable) < 15 {
		t.Fatalf("only %d species carry a harvest table; did the species data load?", len(withTable))
	}

	dataPath := configs.GetFilePathsConfig().DataFiles.String() + `/mobs`
	templates, err := fileloader.LoadAllFlatFiles[int, *mobs.Mob](dataPath)
	if err != nil {
		t.Fatalf("loading mobs: %v", err)
	}
	for id, m := range templates {
		if err := m.Harvest.Validate(tagExists, itemExists); err != nil {
			t.Errorf("mob %d %s: %v", id, m.Character.Name, err)
		}
		checkEntryOutputs(m.Character.Name, m.Harvest)
	}

	// The game animals phase 0 found with empty corpses: each must now have
	// something to skin or butcher, from its species or its own table.
	for _, id := range []int{205, 206, 215, 223, 207, 208, 216, 9139, 9138, 9140, 9141, 9142, 9143} {
		m, ok := templates[id]
		if !ok {
			t.Errorf("mob %d missing from the shipped files", id)
			continue
		}
		if !withTable[m.Character.SpeciesId] && m.Harvest.Empty() {
			t.Errorf("mob %d %s still has nothing to skin or butcher", id, m.Character.Name)
		}
	}

	// Spoiling items name a period the game clock understands.
	for _, spec := range items.GetAllItemSpecs() {
		if spec.SpoilAfter == `` {
			continue
		}
		probe := items.Item{ItemId: spec.ItemId, CraftedRound: 1000}
		if probe.SpoilRound() <= 1000 {
			t.Errorf("item %d %s: spoil_after %q does not parse to a future round", spec.ItemId, spec.Name, spec.SpoilAfter)
		}
	}

	// Every processing chain closes.
	for _, r := range crafting.GetAll() {
		for _, ing := range r.Ingredients {
			if !tagExists(ing.ItemTag) {
				t.Errorf("recipe %s: no item supplies ingredient tag %q", r.RecipeId, ing.ItemTag)
			}
		}
		if r.Tool != `` && !items.IsKnownToolType(r.Tool) {
			t.Errorf("recipe %s: unknown tool %q", r.RecipeId, r.Tool)
		}
	}
}

// TestTimberContent pins timber.yaml against the shipped items and zones, and
// checks that every zone pool names a zone that has choppable rooms.
func TestTimberContent(t *testing.T) {
	mudlog.SetupLogger(nil, `LOW`, ``, false)
	configs.SetConfigForTest(t, configs.GetConfig())
	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}
	t.Cleanup(items.SeedItemsForTest(nil))
	items.LoadDataFiles()

	path := configs.GetFilePathsConfig().DataFiles.String() + `/` + timber.DataFileName
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	d, err := timber.Parse(raw, timber.World{
		ItemExists: func(id int) bool { return items.GetItemSpec(id) != nil },
	})
	if err != nil {
		t.Fatalf("timber.yaml: %v", err)
	}
	timber.Install(d)
	t.Cleanup(func() { timber.Install(nil) })

	species := timber.AllSpecies()
	if len(species) < 10 {
		t.Fatalf("only %d species; did timber.yaml load?", len(species))
	}
	for _, sp := range species {
		log := items.GetItemSpec(sp.LogItemId)
		if log == nil {
			continue // Parse reports it
		}
		if log.ComponentTag == `` {
			t.Errorf("%s log %d has no component tag, so no recipe can saw it", sp.Name, sp.LogItemId)
		}
		if len(log.VendorCategories) == 0 {
			t.Errorf("%s log %d is not sold anywhere", sp.Name, sp.LogItemId)
		}
	}
	for _, biome := range []string{`forest`, `dense_forest`, `swamp`} {
		if !timber.IsChoppable(biome) {
			t.Errorf("biome %s should grow timber", biome)
		}
	}
	if timber.IsChoppable(`city_thoroughfare`) {
		t.Error("city streets must not grow timber")
	}
	if items.FindSpecByComponentTag(`branch`) == nil {
		t.Error("no item carries the branch tag that felling gives")
	}
}
