package content

import (
	"os"
	"path"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

// biomeRecord is the minimal shape needed to classify. The authoritative
// struct is rooms.BiomeInfo; this package deliberately does not import the
// room model, because a content package should not pull in the world runtime
// just to read two fields, and arch_test.go forbids it. The trade is that a
// rename of the `biomeid` or `indoor` yaml key would slip past this test,
// which is acceptable because such a rename breaks room loading loudly and
// immediately.
type biomeRecord struct {
	BiomeId string `yaml:"biomeid"`
	Indoor  bool   `yaml:"indoor"`
}

func loadShippedBiomes(t *testing.T) map[string]biomeRecord {
	t.Helper()
	dir := "../../../_datafiles/world/dogmud/biomes"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read biomes dir: %v", err)
	}
	out := map[string]biomeRecord{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, err := os.ReadFile(path.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		var rec biomeRecord
		if err := yaml.Unmarshal(b, &rec); err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		if rec.BiomeId == "" {
			t.Fatalf("%s: missing biomeid", e.Name())
		}
		out[strings.ToLower(rec.BiomeId)] = rec
	}
	if len(out) == 0 {
		t.Fatal("no biomes loaded; this guard would pass vacuously")
	}
	return out
}

// GUARD 1, the load-bearing one. Every indoor biome must be classified into
// exactly one prose class. Without this, adding a biome with `indoor: true`
// silently serves prose about roofs and windowpanes inside it, which is the
// exact defect the underground split was written to fix.
func TestEveryIndoorBiomeIsClassified(t *testing.T) {
	for id, rec := range loadShippedBiomes(t) {
		if !rec.Indoor {
			continue
		}
		under, surface := undergroundBiomes[id], surfaceIndoorBiomes[id]
		switch {
		case under && surface:
			t.Errorf("biome %q is in BOTH undergroundBiomes and surfaceIndoorBiomes; it must be in exactly one", id)
		case !under && !surface:
			t.Errorf("biome %q has indoor:true but is not classified.\n"+
				"Add it to undergroundBiomes (felt through stone: seepage, draughts, mineral cold)\n"+
				"or surfaceIndoorBiomes (a built structure: roofs, eaves, windows)\n"+
				"in modules/weather/content/emotes.go, and author its prose in\n"+
				"_datafiles/world/dogmud/weather/emotes/*.yaml if it needs its own voice.", id)
		}
	}
}

// GUARD 2. A classified biome that no longer exists is a dead key, which means
// a biome was renamed or removed and the maps were not updated.
func TestClassificationMapsHaveNoDeadKeys(t *testing.T) {
	biomes := loadShippedBiomes(t)
	for _, m := range []struct {
		name string
		set  map[string]bool
	}{
		{"undergroundBiomes", undergroundBiomes},
		{"surfaceIndoorBiomes", surfaceIndoorBiomes},
	} {
		for id := range m.set {
			rec, ok := biomes[id]
			if !ok {
				t.Errorf("%s names %q, which is not a biome. Was it renamed or removed?", m.name, id)
				continue
			}
			if !rec.Indoor {
				t.Errorf("%s names %q, which has indoor:false. Only indoor biomes have a prose class.", m.name, id)
			}
		}
	}
}

// GUARD 3. A biome key authored in a weather table that is not a real biome
// can never be selected: bandedSectionLines looks up the room's biome id and
// falls through to "default". The pool is dead content and nothing says so.
func TestAuthoredBiomeKeysAreRealBiomes(t *testing.T) {
	biomes := loadShippedBiomes(t)
	root := os.DirFS("../../../_datafiles/world/dogmud")

	check := func(t *testing.T, where string, keys []string) {
		t.Helper()
		for _, k := range keys {
			if k == "default" {
				continue
			}
			if _, ok := biomes[strings.ToLower(k)]; !ok {
				t.Errorf("%s authors biome key %q, which is not a biome.\n"+
					"These lines can never render: the lookup falls through to \"default\".\n"+
					"Re-key them to a real biome or remove them.", where, k)
			}
		}
	}

	sectionKeys := func(sec TableSection) []string {
		var out []string
		for k := range sec.Outdoor {
			out = append(out, k)
		}
		for k := range sec.Indoor {
			out = append(out, k)
		}
		for k := range sec.Underground {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}

	tables, err := LoadEmotes(root, "weather/emotes")
	if err != nil {
		t.Fatalf("LoadEmotes: %v", err)
	}
	if len(tables) == 0 {
		t.Fatal("no tables loaded; this guard would pass vacuously")
	}
	for wt, tbl := range tables {
		check(t, string(wt), sectionKeys(tbl.TableSection))
		for season, sec := range tbl.Seasonal {
			check(t, string(wt)+" season:"+season, sectionKeys(sec))
		}
	}

	seasonal, err := LoadSeasonalEmotes(root, "weather/emotes/seasons")
	if err != nil {
		t.Fatalf("LoadSeasonalEmotes: %v", err)
	}
	if len(seasonal) == 0 {
		t.Fatal("no ambience tables loaded; this guard would pass vacuously")
	}
	for k, sec := range seasonal {
		check(t, "ambience "+k.Track+"/"+k.Season, sectionKeys(sec))
	}
}

// GUARD 4. Every non-empty shipped pool meets the depth floor.
func TestShippedPoolsMeetMinimumDepth(t *testing.T) {
	root := os.DirFS("../../../_datafiles/world/dogmud")

	checkSection := func(t *testing.T, where string, sec TableSection) {
		t.Helper()
		for biome, lines := range sec.Outdoor {
			if err := ValidatePool(lines); err != nil {
				t.Errorf("%s outdoor/%s: %v", where, biome, err)
			}
		}
		for _, pair := range []struct {
			name  string
			pools map[string]IndoorPool
		}{{"indoor", sec.Indoor}, {"underground", sec.Underground}} {
			for biome, pool := range pair.pools {
				if err := ValidatePool(pool.Mild); err != nil {
					t.Errorf("%s %s/%s/mild: %v", where, pair.name, biome, err)
				}
				if err := ValidatePool(pool.Strong); err != nil {
					t.Errorf("%s %s/%s/strong: %v", where, pair.name, biome, err)
				}
			}
		}
	}

	tables, err := LoadEmotes(root, "weather/emotes")
	if err != nil {
		t.Fatalf("LoadEmotes: %v", err)
	}
	if len(tables) == 0 {
		t.Fatal("no tables loaded; this guard would pass vacuously")
	}
	for wt, tbl := range tables {
		checkSection(t, string(wt), tbl.TableSection)
		for season, sec := range tbl.Seasonal {
			checkSection(t, string(wt)+" season:"+season, sec)
		}
	}

	seasonal, err := LoadSeasonalEmotes(root, "weather/emotes/seasons")
	if err != nil {
		t.Fatalf("LoadSeasonalEmotes: %v", err)
	}
	if len(seasonal) == 0 {
		t.Fatal("no ambience tables loaded; this guard would pass vacuously")
	}
	for k, sec := range seasonal {
		checkSection(t, "ambience "+k.Track+"/"+k.Season, sec)
	}
}
