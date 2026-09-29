package baubles

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"gopkg.in/yaml.v3"
)

// shippedWorld is the real dogmud world, anchored on this file: the shared
// test binary's working directory is not the package's.
func shippedWorld(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal(`runtime.Caller failed`)
	}
	return filepath.Join(filepath.Dir(here), `..`, `..`, `_datafiles`, `world`, `dogmud`)
}

// loadShippedItems loads the real conditions and items, as boot does before
// the corpus, and puts the test binary's items back afterwards.
func loadShippedItems(t *testing.T, world string) {
	t.Helper()
	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = configs.ConfigString(world)
	cfg.Network.LogoutRounds = 3 // condition 0 refuses 0; a test binary never reads config.yaml
	configs.SetConfigForTest(t, cfg)
	t.Cleanup(items.SeedItemsForTest(map[int]*items.ItemSpec{}))
	conditions.LoadDataFiles()
	items.LoadDataFiles()
	// Without this the name and keyword checks below would pass anything.
	if !items.AuthoredKeyword(`lantern`) || !items.AuthoredName(`Hooded Lantern`) {
		t.Fatal(`the authored item snapshots are empty: items did not load`)
	}
}

func readShippedSeed(t *testing.T, world string) ([]byte, seedDoc) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(world, seedFileName))
	if err != nil {
		t.Fatalf(`read the seed: %v`, err)
	}
	var doc seedDoc
	if err := decodeStrict(data, &doc); err != nil {
		t.Fatalf(`the seed does not parse: %v`, err)
	}
	return data, doc
}

// TestShippedCorpusSeed is the seed's CI gate: every entry loads, reads as
// player copy, keeps the keyword its author wrote, and fits its pool.
func TestShippedCorpusSeed(t *testing.T) {
	world := shippedWorld(t)
	loadShippedItems(t, world)
	data, doc := readShippedSeed(t, world)

	for i, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		if n := utf8.RuneCountInString(line); n > 80 {
			t.Errorf(`line %d is %d columns; wrap at 80`, i+1, n)
		}
		if strings.ContainsAny(line, "\u2013\u2014") {
			t.Errorf(`line %d has an en or em dash`, i+1)
		}
	}

	names := map[string]string{}
	for _, key := range sortedKeys(doc.Entries) {
		prefix, tier, ok := parseCorpusKey(key)
		if !ok {
			t.Errorf(`%q is not a pool key`, key)
			continue
		}
		smallOnly := prefix == pocketPrefix || prefix == ``
		for i, e := range doc.Entries[key] {
			where := fmt.Sprintf(`%s #%d %q`, key, i+1, e.Name)
			if prev, dup := names[normKey(e.Name)]; dup {
				t.Errorf(`%s: the same name as %s`, where, prev)
			}
			names[normKey(e.Name)] = where
			if !tier.Range().Contains(e.Value) {
				t.Errorf(`%s: value %d is outside %s %+v`, where, e.Value, tier, tier.Range())
			}
			if strings.IndexFunc(e.Description+e.Material, unicode.IsDigit) >= 0 {
				t.Errorf(`%s: no numbers in the text`, where)
			}
			cleaned, err := checkEntry(e)
			if err != nil {
				t.Errorf(`%s: %v`, where, err)
				continue
			}
			if cleaned.NameSimple != strings.ToLower(e.NameSimple) {
				t.Errorf(`%s: keyword %q is taken (reserved, or a real item's word); players would type %q. Write that, or pick another noun`, where, e.NameSimple, cleaned.NameSimple)
			}
			if smallOnly && TooBigFor(cleaned.reply(), SourcePickpocket) {
				t.Errorf(`%s: pocket and bare-tier entries must fit a pocket`, where)
			}
		}
	}

	rep := LoadCorpusFrom(filepath.Join(world, seedFileName), filepath.Join(t.TempDir(), overlayFileName))
	t.Cleanup(ClearCorpusForTest)
	if rep.SeedErr != nil || len(rep.Skipped) > 0 {
		t.Fatalf("every seed entry must load:\n%v\n%s", rep.SeedErr, strings.Join(rep.Skipped, "\n"))
	}
}

// Every biome where a search can find something has a group, so its finds
// reach a group pool. BaseChance is the Go default here, which
// TestBaubleShippedConfigMatchesDefaults pins to the shipped config.
func TestEveryFindableBiomeHasACorpusGroup(t *testing.T) {
	world := shippedWorld(t)
	_, doc := readShippedSeed(t, world)
	files, err := os.ReadDir(filepath.Join(world, `biomes`))
	if err != nil || len(files) == 0 {
		t.Fatalf(`no biome files read (%v): this test would prove nothing`, err)
	}
	known := map[string]bool{`dwelling`: true, `street`: true, `underground`: true, `ruins`: true, `waterside`: true, `wild`: true}
	checked := 0
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), `.yaml`) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(world, `biomes`, f.Name()))
		if err != nil {
			t.Fatal(err)
		}
		var b struct {
			BiomeId string `yaml:"biomeid"`
		}
		if err := yaml.Unmarshal(data, &b); err != nil {
			t.Fatalf(`%s: %v`, f.Name(), err)
		}
		id := normKey(b.BiomeId)
		if BaseChance(id) <= 0 {
			continue
		}
		checked++
		g, ok := doc.Groups[id]
		if !ok {
			t.Errorf(`biome %s finds baubles (%.2f%% per roll) but has no corpus group`, id, BaseChance(id))
			continue
		}
		if !known[g] {
			t.Errorf(`biome %s is in group %q, which is not one of the six`, id, g)
		}
	}
	if checked == 0 {
		t.Fatal(`no biome with a chance above zero: the check ran on nothing`)
	}
}
