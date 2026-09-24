# M3 item 9 PR 2: the outdoors / indoors / underground split

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give weather three prose classes instead of two, author the
underground section that 124 of the 161 indoor rooms have needed all along,
bring every non-empty pool to depth 6, and guard the biome coupling so a new
biome cannot silently inherit the wrong prose.

**Architecture:** `TableSection` gains `Underground`; `Table` embeds
`TableSection` inline so the seasonal variants and the seasonal-ambience tables
inherit it rather than duplicating a field pair. Two Go maps in
`modules/weather/content` classify biomes, and class resolves from the
`(biome, indoor)` pair `engine.EmitAmbient` already passes, so the emitter and
the entire sim side stay untouched.

**Tech Stack:** Go, `gopkg.in/yaml.v2` (`,inline` on an embedded struct), the
`internal/narration` snapshot harness.

Spec: [`docs/superpowers/specs/2026-09-16-messaging-m3-item9-weather-emotes-design.md`](../specs/2026-09-16-messaging-m3-item9-weather-emotes-design.md)
Depends on: [PR 1](2026-09-16-messaging-m3-item9-weather-core-join.md), which must be merged first.

---

## Task ordering is load-bearing

**Content is authored before the class resolution is flipped.** If `cave` starts
resolving to `underground` while that section is empty,
`bandedSectionLines` correctly refuses to fall back (silence beats wrong prose)
and 124 rooms go quiet. Authoring first means the window never opens, not even
between commits inside this PR.

The depth validator is wired in **last**, for the same reason inverted: shipped
pools are 1 to 4 deep, so a minimum of 6 enforced before the padding pass would
fail every load.

---

## File Structure

| File | Responsibility | Change |
|---|---|---|
| `modules/weather/content/emotes.go` | Schema, load, class resolution, validation | Modify |
| `modules/weather/content/emotes_test.go` | Store unit tests | Modify |
| `modules/weather/content/biome_coupling_test.go` | The four shipped-data guards | **Create** |
| `modules/weather/content/context.md` | Package doc | Modify |
| `internal/rooms/biomes.go` | Biome record | Modify: comment on `Indoor` only |
| `internal/rooms/context.md` | Package doc | Modify: record the coupling |
| `_datafiles/world/dogmud/weather/emotes/*.yaml` | 9 weather tables | Modify: all three sections |
| `_datafiles/world/dogmud/weather/emotes/seasons/*.yaml` | 6 ambience tables | Modify: depth, and the `jungle` fix |
| `internal/narration/testdata/stores/weather_emotes.golden` | Frozen output | Re-recorded, once, in Task 8 |
| `docs/superpowers/audits/messaging-m6-content-ledger.md` | Deferred-text ledger | Modify: item 9's rows |

---

### Task 0: The `underground` section parses

**Files:**
- Modify: `modules/weather/content/emotes.go`
- Modify: `modules/weather/content/emotes_test.go`

- [ ] **Step 1: Write the failing test**

Add to `modules/weather/content/emotes_test.go`:

```go
// The three sections are peers on the same struct, and Table embeds
// TableSection so a new section is declared once rather than twice (Table used
// to carry its own copy of Outdoor and Indoor alongside Seasonal's
// TableSection).
func TestUndergroundSectionParses(t *testing.T) {
	src := []byte(`
weather: rain
outdoor:
  default: ["outdoor line"]
indoor:
  default:
    mild: []
    strong: ["indoor line"]
underground:
  default:
    mild: []
    strong: ["underground line"]
seasonal:
  winter:
    underground:
      default:
        mild: []
        strong: ["winter underground line"]
`)
	tbl, err := ParseEmoteTable(src)
	if err != nil {
		t.Fatalf("ParseEmoteTable: %v", err)
	}
	if got := tbl.Underground["default"].Strong; len(got) != 1 || got[0] != "underground line" {
		t.Fatalf("base underground section did not parse: %#v", got)
	}
	// The seasonal variants inherit the new section through the embed. If
	// Table had kept its own field pair this would still be empty.
	if got := tbl.Seasonal["winter"].Underground["default"].Strong; len(got) != 1 {
		t.Fatalf("seasonal underground section did not parse: %#v", got)
	}
	// The existing sections must be unaffected by the embed.
	if got := tbl.Outdoor["default"]; len(got) != 1 || got[0] != "outdoor line" {
		t.Fatalf("outdoor section regressed: %#v", got)
	}
	if got := tbl.Indoor["default"].Strong; len(got) != 1 {
		t.Fatalf("indoor section regressed: %#v", got)
	}
	if tbl.Weather != "rain" {
		t.Fatalf("weather key regressed: %q", tbl.Weather)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./modules/weather/content/ -run TestUndergroundSectionParses -v`

Expected: FAIL to compile, `tbl.Underground undefined`.

- [ ] **Step 3: Change the schema**

In `modules/weather/content/emotes.go`, replace the `TableSection` and `Table`
declarations (`:29` and `:39`):

```go
// TableSection is one outdoor/indoor/underground set of biome-keyed lines.
//
// Indoor and Underground are felt-banded (IndoorPool); Outdoor is a flat list
// because weather outdoors is never attenuated. The three are PROSE CLASSES,
// not room flags: a house and a cave are both sheltered, but rain on a roof
// and water finding a seam in rock are different sentences, and 124 of the
// game's 161 indoor rooms are underground.
type TableSection struct {
	Outdoor     map[string][]string   `yaml:"outdoor"`
	Indoor      map[string]IndoorPool `yaml:"indoor"`
	Underground map[string]IndoorPool `yaml:"underground"`
}

// Table holds the ambient lines for one weather type, keyed by biome with a
// "default" fallback, split by prose class (spec §9.4). The base sections are
// embedded rather than repeated so that adding a class adds it once and the
// per-season variants inherit it.
//
// The spec's per-line weights are an unneeded refinement for shipped defaults;
// builders wanting bias can repeat a line.
type Table struct {
	Weather      string `yaml:"weather"`
	TableSection `yaml:",inline"`
	// Seasonal holds optional per-season variants, keyed by season NAME
	// (matching across tracks by design: "winter" is temperate's winter).
	// Missing seasons/sections fall through to the base lines (spec §6).
	Seasonal map[string]TableSection `yaml:"seasonal"`
}
```

- [ ] **Step 4: Run the test**

Run: `go test ./modules/weather/content/ -run TestUndergroundSectionParses -v`

Expected: PASS.

- [ ] **Step 5: Confirm the shipped files still parse and behaviour is unchanged**

Run: `go test ./modules/weather/...`

Expected: PASS, including `TestShippedDogmudEmoteTables`.

Run: `go test ./internal/narration/ -run 'TestSnapshotStores/weather_emotes'`

Expected: PASS with no `-update`. The schema gained a section that no file
authors and no code reads yet, so nothing a player sees can have changed.

- [ ] **Step 6: Commit**

```bash
git add modules/weather/content/emotes.go modules/weather/content/emotes_test.go
git commit -m "feat(weather): underground as a third prose class in the schema

TableSection gains Underground; Table embeds TableSection inline so the
per-season variants and the seasonal-ambience tables inherit the new class
instead of Table carrying a duplicate field pair.

No file authors the section and no code reads it yet, so the golden is
byte-identical.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 1: Author the underground prose, reviewed sample first

Authored BEFORE the class resolution is flipped, so caves are never silent.

**Files:**
- Modify: `_datafiles/world/dogmud/weather/emotes/rain.yaml`

- [ ] **Step 1: Author `rain`'s underground section to depth 6**

Append to `_datafiles/world/dogmud/weather/emotes/rain.yaml`:

```yaml
underground:
  default:
    mild: []
    strong:
      - "Water finds a seam in the rock and begins a slow, patient dripping."
      - "The stone sweats; every surface turns slick and cold to the touch."
      - "Somewhere deeper in, water is running that was not running before."
      - "A thin trickle threads down the wall and pools in the hollows."
      - "The air turns heavy and mineral, thick with the smell of wet rock."
      - "Drips quicken overhead, counting out a rhythm with no pattern to it."
```

**Voice rules, all four mandatory:**
- 80 character hard wrap, enforced by `shipped_emotes_test.go`
- no roofs, panes, shutters, eaves, rafters, floorboards, windows or doors
- no raw numbers
- describe what is **felt in this room**, not the weather outside it

`mild: []` stays empty and is deliberate: light rain does not register through
stone. Do not pad it.

- [ ] **Step 2: Verify it parses and wraps**

Run: `go test ./modules/weather/content/ -run TestShippedDogmudEmoteTables -v`

Expected: PASS. A line over 80 characters fails here and names the line.

- [ ] **Step 3: Stop for owner acceptance**

Show the six lines to the owner before authoring the other eight files. This
is the item 8 procedure: one reviewed sample sets the voice, and getting the
voice wrong across nine files is expensive to undo.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/weather/emotes/rain.yaml
git commit -m "content(weather): underground prose for rain, the reviewed sample

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Author the remaining eight underground sections

**Files:**
- Modify: `blizzard.yaml`, `dust.yaml`, `fog.yaml`, `frost.yaml`,
  `heatwave.yaml`, `overcast.yaml`, `snow.yaml`, `storm.yaml`

- [ ] **Step 1: Apply the Task 1 procedure to each file in this order**

`storm`, `blizzard`, `dust`, `snow`, `frost`, `fog`, `heatwave`, `overcast`.

Severe types first, because `shipped_emotes_test.go` already requires
`storm`, `blizzard` and `dust` to have a non-empty indoor `strong` pool, and
the same expectation will extend to underground in Task 6.

🔑 **OWNER RULING 2026-09-16: SILENCE IS THE DEFAULT UNDERGROUND.**
*"Most weather would be imperceptible inside a cave unless it is a really
strong storm."*

This inverts the authoring burden. Underground is not "indoor prose about
stone"; it is mostly **nothing**, with a short list of exceptions. Do not
author a pool to fill a slot. An empty pool is the correct, expected answer for
most of these types, and the store is built to render silence cleanly.

Note that the felt banding already carries half of this rule: `mild` plays
below `StrongFeltThreshold` and `strong` at or above it, so authoring
`strong` only means a type is felt underground **only at high intensity**,
which is exactly the ruling. `mild` stays empty underground for all nine, with
no exceptions.

| Type | Underground `strong` | What is actually felt |
|---|---|---|
| `storm` | **author, depth 6** | thunder transmitted through rock as pressure, dust shaken loose |
| `blizzard` | **author, depth 6** | the draught reversing, cold pouring down the passage |
| `dust` | **author, depth 6** | grit sifting from the ceiling, the air drying out |
| `rain` | **author, depth 6** | heavy rain only: seepage finding a seam, stone sweating |
| `frost` | **author, depth 6** | stone aching cold along the seep lines (owner ruled in 2026-09-16) |
| `heatwave` | **author, depth 6** | the cave staying cool, the contrast at the entrance (owner ruled in) |
| `snow` | **leave empty** | snow is silent, and a cave cannot hear it |
| `fog` | **leave empty** | fog does not enter stone |
| `overcast` | **leave empty** | there is nothing to perceive |

`rain.yaml` was authored in Task 1 before this ruling. Re-read its six lines
against it: they describe seepage and sweating stone, which is the heavy-rain
case, so they stand. If any line reads as gentle rain, replace it.

Where a pool is left empty, say so explicitly in the commit message so it reads
as a decision rather than an oversight. Guard 4 permits empty pools by design,
so nothing will flag these for you.

- [ ] **Step 2: Verify every file after each edit**

Run: `go test ./modules/weather/content/ -run TestShippedDogmudEmoteTables -v`

Expected: PASS after each file.

- [ ] **Step 3: Confirm all nine now carry the section**

Run: `grep -L "^underground:" _datafiles/world/dogmud/weather/emotes/*.yaml`

Expected: no output. `grep -L` lists files *without* a match, so any filename
printed is a file that was missed.

- [ ] **Step 4: Confirm the golden is still GREEN**

Run: `go test ./internal/narration/ -run 'TestSnapshotStores/weather_emotes'`

Expected: PASS, with no `-update`.

This is the check that proves the ordering is working. The underground prose is
authored but nothing resolves to it yet, so not one rendered line has changed.
A red golden here means class resolution leaked in early and caves may have
been silent between commits.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/weather/emotes/
git commit -m "content(weather): underground prose for the remaining eight types

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Classification maps and class resolution

The content now exists, so flipping resolution is safe.

**Files:**
- Modify: `modules/weather/content/emotes.go`
- Modify: `modules/weather/content/emotes_test.go`

- [ ] **Step 1: Write the failing test**

Add to `modules/weather/content/emotes_test.go`:

```go
func TestClassResolution(t *testing.T) {
	tables := Tables{
		"rain": {
			Weather: "rain",
			TableSection: TableSection{
				Outdoor: map[string][]string{"default": {"OUT"}},
				Indoor: map[string]IndoorPool{
					"default": {Mild: []string{"IN-MILD"}, Strong: []string{"IN-STRONG"}},
				},
				Underground: map[string]IndoorPool{
					"default": {Mild: nil, Strong: []string{"UNDER-STRONG"}},
				},
			},
		},
	}

	cases := []struct {
		name   string
		biome  string
		indoor bool
		felt   float64
		want   string
	}{
		{"outdoor ignores the biome", "forest", false, 1.0, "OUT"},
		{"house is surface indoor", "house", true, 1.0, "IN-STRONG"},
		{"fort is surface indoor", "fort", true, 1.0, "IN-STRONG"},
		{"cave is underground", "cave", true, 1.0, "UNDER-STRONG"},
		{"dungeon is underground", "dungeon", true, 1.0, "UNDER-STRONG"},
		{"spiderweb is NOT underground", "spiderweb", true, 1.0, "IN-STRONG"},
		{"indoor mild band below threshold", "house", true, 0.0, "IN-MILD"},
		{"underground mild is empty, so silence", "cave", true, 0.0, ""},
		{"unknown biome falls back to default", "nowhere", true, 1.0, "IN-STRONG"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := tables.Pick("rain", c.biome, c.indoor, c.felt, "", narration.FirstPicker)
			if got != c.want {
				t.Fatalf("want %q, got %q", c.want, got)
			}
		})
	}
}

// Underground must never borrow indoor's or outdoor's prose. A cave with no
// authored underground pool is SILENT, which is the store's standing rule:
// silence beats wrong prose.
func TestUndergroundNeverFallsBackToAnotherClass(t *testing.T) {
	tables := Tables{
		"rain": {
			Weather: "rain",
			TableSection: TableSection{
				Outdoor: map[string][]string{"default": {"OUT"}},
				Indoor: map[string]IndoorPool{
					"default": {Strong: []string{"IN-STRONG"}},
				},
				// Underground deliberately absent.
			},
		},
	}
	if got := tables.Pick("rain", "cave", true, 1.0, "", narration.FirstPicker); got != "" {
		t.Fatalf("underground with no pool must be silent, got %q", got)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./modules/weather/content/ -run 'TestClassResolution|TestUndergroundNeverFallsBack' -v`

Expected: FAIL. The `cave` and `dungeon` cases return `IN-STRONG` because
nothing classifies them yet. Confirm the failure names those cases
specifically; if `spiderweb` also fails, the test itself is wrong.

- [ ] **Step 3: Add the maps and the resolver**

In `modules/weather/content/emotes.go`, add above `bandedSectionLines`:

```go
// undergroundBiomes names the biomes whose weather is felt through STONE
// rather than through walls: seepage, draughts, transmitted sound, mineral
// cold. 124 of the game's 161 indoor rooms are one of these, which is why the
// class exists at all.
//
// 🔑 ADDING, RENAMING OR REMOVING A BIOME REQUIRES EDITING THIS MAP OR
// surfaceIndoorBiomes. biome_coupling_test.go fails the build otherwise; it is
// the only thing standing between a new indoor biome and silently inheriting
// prose about roofs and windowpanes.
var undergroundBiomes = map[string]bool{
	"cave":    true,
	"dungeon": true,
}

// surfaceIndoorBiomes names the sheltered-but-not-underground biomes: built
// structures, where rain on a roof and wind in the eaves are the right images.
//
// This map exists so classification is TOTAL. Without it a newly added indoor
// biome would fall through to this class silently, which is exactly the defect
// the underground split was written to fix.
//
// spiderweb is here rather than in undergroundBiomes deliberately: it is dark
// and sheltered, but its darkness is webbing, not stone, so stone prose would
// be wrong. It currently has ZERO rooms, so no prose is authored for it; if it
// is ever used it wants its own biome-keyed pool rather than either default.
var surfaceIndoorBiomes = map[string]bool{
	"house":     true,
	"fort":      true,
	"spiderweb": true,
}
```

Then replace `bandedSectionLines` (`emotes.go:129`):

```go
// bandedSectionLines resolves one prose class and then biome -> "default"
// within it. Outdoor is a flat list; Indoor and Underground are felt-banded
// (Mild below StrongFeltThreshold, else Strong).
//
// A class NEVER falls back to another class. An unauthored underground pool
// renders silence rather than borrowing house prose, which is the whole point
// of the split.
func bandedSectionLines(sec TableSection, biome string, useIndoor bool, felt float64) []string {
	if !useIndoor {
		lines := sec.Outdoor[biome]
		if len(lines) == 0 {
			lines = sec.Outdoor["default"]
		}
		return lines
	}

	pools := sec.Indoor
	if undergroundBiomes[biome] {
		pools = sec.Underground
	}

	pool, ok := pools[biome]
	if !ok || (len(pool.Mild) == 0 && len(pool.Strong) == 0) {
		pool = pools["default"]
	}
	if felt >= StrongFeltThreshold {
		return pool.Strong
	}
	return pool.Mild
}
```

Update the three call sites in `Tables.Pick` and `SeasonalTables.Pick` to pass
the whole section rather than two maps:

```go
	// in Tables.Pick, the seasonal-variant branch:
			lines = bandedSectionLines(v, biome, indoor, felt)
	// in Tables.Pick, the base branch:
		lines = bandedSectionLines(t.TableSection, biome, indoor, felt)
	// in SeasonalTables.Pick:
	return renderAmbient(bandedSectionLines(sec, biome, indoor, felt), pick)
```

- [ ] **Step 4: Run the tests**

Run: `go test ./modules/weather/content/ -run 'TestClassResolution|TestUndergroundNeverFallsBack' -v`

Expected: PASS, all nine subtests.

- [ ] **Step 5: Run the module**

Run: `go test ./modules/weather/...`

Expected: PASS.

- [ ] **Step 6: The golden must change in EXACTLY the cave and dungeon rows**

Run: `go test ./internal/narration/ -run 'TestSnapshotStores/weather_emotes' -v`

Expected: FAIL. This is the first legitimate golden change in item 9, and it is
the moment the whole slice is proved.

The builder needs no edit: it passes `(biome, indoor)` and lets `Pick` resolve
the class, and PR 1 deliberately swept a representative biome set so that
`cave` and `dungeon` rows already exist. They have been carrying house prose;
they must now carry the underground prose authored in Tasks 1 and 2.

Before updating, inspect the diff:

Run: `go test ./internal/narration/ -run 'TestSnapshotStores/weather_emotes' -update`

Run: `git diff internal/narration/testdata/stores/weather_emotes.golden`

Every changed row must match `sheltered|cave|` or `sheltered|dungeon|`.
Confirm nothing else moved:

Run: `git diff -U0 internal/narration/testdata/stores/weather_emotes.golden | grep '^[+-]' | grep -v '^[+-][+-]' | grep -vE 'sheltered\|(cave|dungeon)\|'`

Expected: no output. Run this one standalone: `grep` exits 1 on zero matches
and would break an `&&` chain, silently skipping whatever followed.

Any `outdoor`, `house`, `fort` or `spiderweb` row in that output means the
resolver touched a class it was not supposed to. Stop and fix it rather than
accepting the re-record.

- [ ] **Step 7: Confirm the harness**

Run: `go test ./internal/narration/`

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add modules/weather/content/emotes.go modules/weather/content/emotes_test.go internal/narration/testdata/stores/weather_emotes.golden
git commit -m "feat(weather): resolve prose class from the biome

cave and dungeon now read the underground section; house, fort and spiderweb
keep the built-structure prose; outdoor is unchanged. Class resolves from the
(biome, indoor) pair EmitAmbient already passes, so the emitter and the whole
sim side are untouched.

A class never falls back to another class: an unauthored underground pool is
silence, not borrowed house prose.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: The biome coupling guards

**Files:**
- Create: `modules/weather/content/biome_coupling_test.go`

- [ ] **Step 1: Write the guards**

Create `modules/weather/content/biome_coupling_test.go`:

```go
package content

import (
	"io/fs"
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
// just to read two fields. The trade is that a rename of the `biomeid` or
// `indoor` yaml key would slip past this test, which is acceptable because
// such a rename breaks room loading loudly and immediately.
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

	sectionKeys := func(sec TableSection) (out []string) {
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
	for k, sec := range seasonal {
		check(t, "ambience "+k.Track+"/"+k.Season, sectionKeys(sec))
	}

	// Guard the guard: fs.FS is used above only through LoadEmotes, so if the
	// data dir moved these loops would run zero times and pass vacuously.
	if len(tables) == 0 || len(seasonal) == 0 {
		t.Fatal("no tables loaded; this guard would pass vacuously")
	}
	_ = fs.ValidPath("weather/emotes")
}
```

- [ ] **Step 2: Run the guards**

Run: `go test ./modules/weather/content/ -run 'TestEveryIndoorBiomeIsClassified|TestClassificationMapsHaveNoDeadKeys|TestAuthoredBiomeKeysAreRealBiomes' -v`

Expected: guards 1 and 2 PASS. **Guard 3 FAILS**, naming
`monsoon_wet` and the key `jungle`. That is a real pre-existing defect, not a
bug in the guard: `jungle` is not a biome in either world and no room uses it,
so those two lines have never rendered. Task 5 fixes it.

- [ ] **Step 3: Prove guard 1 is capable of failing**

A guard that has never been seen to fail is not a guard.

```bash
cp _datafiles/world/dogmud/biomes/cave.yaml /tmp/crypt.yaml
```

Create `_datafiles/world/dogmud/biomes/crypt.yaml` with `biomeid: crypt`,
`name: Crypt`, `indoor: true` and the other fields copied from `cave.yaml`.

Run: `go test ./modules/weather/content/ -run TestEveryIndoorBiomeIsClassified -v`

Expected: FAIL, naming `crypt` and telling the reader which two maps to choose
between. Read the message and confirm it is actually actionable.

Then remove the file:

```bash
rm _datafiles/world/dogmud/biomes/crypt.yaml
```

Run: `go test ./modules/weather/content/ -run TestEveryIndoorBiomeIsClassified -v`

Expected: PASS. Confirm `git status --porcelain` shows no stray `crypt.yaml`.

- [ ] **Step 4: Commit guards 1 and 2 only**

Guard 3 is red until Task 5, so this commit would leave the tree failing.
Commit all three anyway **only if** Task 5 is done in the same sitting;
otherwise do Task 5 first and commit them together. Prefer the latter.

---

### Task 5: Fix the dead `jungle` pools

**Files:**
- Modify: `_datafiles/world/dogmud/weather/emotes/seasons/monsoon_wet.yaml`

- [ ] **Step 1: Read what is stranded**

Run: `grep -A3 "^  jungle:" _datafiles/world/dogmud/weather/emotes/seasons/monsoon_wet.yaml`

Expected, verbatim:

```yaml
  jungle:
    - "The canopy drips on long after the rain has stopped, a slow second rainfall."
    - "Mist coils up from the jungle floor wherever light leans through the leaves."
```

- [ ] **Step 2: Re-key to `forest` and adjust the one phrase**

`forest` is the nearest real biome, and both lines are about a canopy, which
`forest` has. The second line names "the jungle floor" and must change.

If `monsoon_wet.yaml` already authors a `forest:` key, merge into it rather
than creating a second one; a duplicate YAML key silently drops the earlier
entry.

```yaml
  forest:
    - "The canopy drips on long after the rain has stopped, a slow second rainfall."
    - "Mist coils up from the forest floor wherever light leans through the leaves."
```

- [ ] **Step 3: Verify guard 3 goes green**

Run: `go test ./modules/weather/content/ -run TestAuthoredBiomeKeysAreRealBiomes -v`

Expected: PASS.

- [ ] **Step 4: Commit the guards and the fix together**

```bash
git add modules/weather/content/biome_coupling_test.go _datafiles/world/dogmud/weather/emotes/seasons/monsoon_wet.yaml
git commit -m "test(weather): guard the biome coupling, and fix the dead jungle pools

Three shipped-data guards: every indoor biome is classified into exactly one
prose class, no classification key is dead, and every biome key authored in a
weather table is a real biome.

The third went red on the day it was written. monsoon_wet.yaml authored a
jungle: key; jungle is not a biome in either world and no room uses it, so
those two lines have never rendered and never could. Re-keyed to forest.

Guard 1 was proven capable of failing with a throwaway crypt biome before it
was trusted.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Rewrite indoor, pad outdoor, to depth 6

**Files:**
- Modify: all 9 files in `_datafiles/world/dogmud/weather/emotes/`
- Modify: all 6 files in `_datafiles/world/dogmud/weather/emotes/seasons/`

- [ ] **Step 1: Rewrite the indoor sections**

All 31 shipped indoor lines stay in the built-structure class, which is now
correct for them, but the owner's second finding still applies: several narrate
the weather **outside** rather than what is felt inside. Rewrite those and pad
each `strong` pool to 6.

The two the owner named, to be fixed for certain:

| File | Replace | Because |
|---|---|---|
| `snow.yaml` | `"Snow whispers against the windows, piling soft in the corners outside."` | narrates outside |
| `fog.yaml` | `"The world beyond the glass has simply dissolved into pale nothing."` | narrates outside |

- [ ] **Step 2: Pad the outdoor pools to depth 6**

Every non-empty outdoor pool, including the per-biome ones. The shallowest
today is `rain.yaml`'s `swamp` at a single line.

**Guard 4 is the worklist.** It is written in Task 7 Step 5, but write it
FIRST, here, and let it be red: its failure output names every
`type/section/biome/band` still under depth, which is exactly the list this
task works through. This is item 8's pattern, where the audit tool's output was
the red signal.

Run: `go test ./modules/weather/content/ -run TestShippedPoolsMeetMinimumDepth -v`

Expected at the start of this task: FAIL, listing every shallow pool. Expected
at the end: PASS. Do not commit while it is red; the commits in Step 4 come
after it is green, or the tree ships failing.

Note that only the TEST moves early. `ValidatePool` and `minPoolDepth` (Task 7
Steps 1 to 3) must exist for it to compile, so write those here too; what stays
in Task 7 is wiring validation into the loaders, which must come last.

No Python tool is needed. An earlier draft of this plan called for one; guard 4
reports the same thing from data the test already loads, and a Python tool that
touched these files would risk the `yaml.dump` trap item 8 recorded, where a
rewrite destroys quoting and comment headers.

- [ ] **Step 3: Verify after each file**

Run: `go test ./modules/weather/content/ -run TestShippedDogmudEmoteTables -v`

Expected: PASS after each file. This catches an over-80-character line
immediately, which is much cheaper than finding nine of them at the end.

- [ ] **Step 4: Commit per file or per small group**

```bash
git add _datafiles/world/dogmud/weather/emotes/<file>.yaml
git commit -m "content(weather): <type> indoor rewrite and outdoor padding to depth 6

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Wire the depth validator into the loaders

`ValidatePool`, `minPoolDepth` and guard 4 were written in Task 6, where guard
4's red output served as the padding worklist. What remains here is wiring
validation into the load path, which must come **last**: it fails every load
until the content pass is complete.

If Task 6 was executed as written, Steps 1 to 6 below are already done. Verify
each is green rather than re-writing it, then do Step 7.

**Files:**
- Modify: `modules/weather/content/emotes.go`
- Modify: `modules/weather/content/emotes_test.go`
- Modify: `modules/weather/content/biome_coupling_test.go`

- [ ] **Step 1: Write the failing test**

Add to `modules/weather/content/emotes_test.go`:

```go
func TestValidatePool(t *testing.T) {
	// Empty is LEGAL and means deliberate silence: light weather is inaudible
	// through walls and imperceptible through stone. This is the case that
	// stops a flat minimum being usable, and weather is the only store in the
	// arc that has it.
	if err := ValidatePool(nil); err != nil {
		t.Fatalf("empty pool must be legal: %v", err)
	}
	if err := ValidatePool([]string{}); err != nil {
		t.Fatalf("empty pool must be legal: %v", err)
	}
	for n := 1; n < minPoolDepth; n++ {
		lines := make([]string, n)
		for i := range lines {
			lines[i] = "line"
		}
		if err := ValidatePool(lines); err == nil {
			t.Errorf("a pool of %d must be rejected; the minimum is %d", n, minPoolDepth)
		}
	}
	deep := make([]string, minPoolDepth)
	for i := range deep {
		deep[i] = "line"
	}
	if err := ValidatePool(deep); err != nil {
		t.Fatalf("a pool of %d must be accepted: %v", minPoolDepth, err)
	}
	// A blank variant is rejected at any depth.
	deep[2] = ""
	if err := ValidatePool(deep); err == nil {
		t.Error("a blank variant must be rejected")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./modules/weather/content/ -run TestValidatePool -v`

Expected: FAIL to compile, `undefined: ValidatePool`.

- [ ] **Step 3: Implement**

Add to `modules/weather/content/emotes.go`:

```go
// minPoolDepth is the floor for a NON-EMPTY ambient pool.
//
// Ambient emotes fire every few rounds for a whole session in one zone, so
// repetition shows far faster here than in combat, where a given pool is drawn
// from only during a fight. Six is where a session stops feeling looped.
const minPoolDepth = 6

// ValidatePool enforces the depth contract for ONE pool.
//
// 🔑 AN EMPTY POOL IS LEGAL AND MEANS DELIBERATE SILENCE: light weather is
// inaudible through walls and imperceptible through stone, and `mild: []` is
// how an author says so. Weather is the only store in the arc where silence is
// an authored value, which is why this wraps narration.ValidateVariants rather
// than calling it directly: everything except the empty case is delegated.
func ValidatePool(lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	return narration.ValidateVariants(
		narration.Variants{Observer: lines}, minPoolDepth, narration.RoleObserver)
}
```

- [ ] **Step 4: Run the test**

Run: `go test ./modules/weather/content/ -run TestValidatePool -v`

Expected: PASS.

- [ ] **Step 5: Add guard 4 over the shipped data**

Add to `modules/weather/content/biome_coupling_test.go`:

```go
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
```

- [ ] **Step 6: Run guard 4**

Run: `go test ./modules/weather/content/ -run TestShippedPoolsMeetMinimumDepth -v`

Expected: PASS. Any failure names the exact `type/section/biome/band` that is
still shallow, which is a Task 6 miss. Go back and finish it.

- [ ] **Step 7: Wire validation into the loaders**

In `LoadEmotes`, after `ParseEmoteTable` succeeds for a file, validate every
pool in the table and return an error naming the file and the pool. Do the same
in `LoadSeasonalEmotes`.

**Keep the existing failure policy: the caller fails soft.** Weather is
ambient; a bad emote file must not stop the world booting. The loader returns
the error, the module logs it and runs with empty tables, which is silence. The
guards above are what actually hold the contract, because they fail the build
rather than the server.

- [ ] **Step 8: Run everything**

Run: `go test ./modules/weather/...`

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add modules/weather/content/emotes.go modules/weather/content/emotes_test.go modules/weather/content/biome_coupling_test.go
git commit -m "feat(weather): depth floor of 6 for every non-empty ambient pool

An empty pool stays legal and means deliberate silence, which is why this
wraps narration.ValidateVariants rather than calling it directly. Weather is
the only store in the arc where silence is an authored value.

Wired in last: shipped pools were 1 to 4 deep, so enforcing this before the
content pass would have failed every load. Loader failure policy is unchanged
and still fails soft, because ambient weather must not stop the world booting;
the shipped-data guard is what holds the contract.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Re-record the golden, once

**Files:**
- Modify: `internal/narration/testdata/stores/weather_emotes.golden`

- [ ] **Step 1: Look at the diff before updating**

Run: `go test ./internal/narration/ -run 'TestSnapshotStores/weather_emotes' -v`

Expected: FAIL, with rows changed by the content pass.

- [ ] **Step 2: Re-record**

Run: `go test ./internal/narration/ -run 'TestSnapshotStores/weather_emotes' -update`

- [ ] **Step 3: Verify the shape of the change**

Run: `git diff --stat internal/narration/testdata/stores/weather_emotes.golden`

The row count must GROW (padding adds biome keys and the `jungle` row becomes a
`forest` row). Confirm no row went from text to `""` except where a pool was
deliberately emptied, which for this slice is only a possible `overcast`
underground `strong`.

Run: `git diff internal/narration/testdata/stores/weather_emotes.golden | grep '^-' | grep -v '=> ""'`

Read every line this prints. Each is a line a player used to see and no longer
will. That is expected for the indoor rewrite, and must be nothing else.

- [ ] **Step 4: Confirm the whole harness**

Run: `go test ./internal/narration/`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/narration/testdata/stores/weather_emotes.golden
git commit -m "test(narration): re-record the weather golden after the content pass

The one legitimate golden change in item 9.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: Comments, context.md, and the M6 ledger

**Files:**
- Modify: `internal/rooms/biomes.go`
- Modify: `internal/rooms/context.md`
- Modify: `modules/weather/content/context.md`
- Modify: `docs/superpowers/audits/messaging-m6-content-ledger.md`

- [ ] **Step 1: Comment the `Indoor` field**

In `internal/rooms/biomes.go:25`, replace the field's trailing comment:

```go
	// Indoor marks a room as sheltered from weather; outdoor-only mutators
	// don't render here.
	//
	// 🔑 ADDING, RENAMING OR REMOVING A BIOME? An indoor biome must also be
	// classified as a weather PROSE CLASS, in one of the two maps in
	// modules/weather/content/emotes.go: undergroundBiomes (felt through
	// stone) or surfaceIndoorBiomes (a built structure). Without that a new
	// indoor biome silently serves prose about roofs and windowpanes inside
	// it. modules/weather/content/biome_coupling_test.go fails the build if
	// you forget, but it runs in that package, so `go test ./internal/rooms/...`
	// alone will NOT tell you.
	Indoor bool `yaml:"indoor,omitempty"`
```

The explicit file-and-map naming is deliberate. "See the weather module" would
not survive contact with someone who has never opened it.

- [ ] **Step 2: Record the coupling in both context.md files**

`internal/rooms/context.md`: a short Gotchas entry pointing at the weather
classification and the guard.

`modules/weather/content/context.md`: the three prose classes, the two maps,
that `spiderweb` is deliberately surface-indoor with zero rooms, the depth
floor, that an empty pool is deliberate silence, and the four guards.

**Verify before you document.** Every symbol named must exist:

Run: `Select-String -Path modules\weather\content\*.go -Pattern '^(func|type|const|var)\s'`

Run: `python tools/context_md_audit.py`

Expected: no findings for either package.

- [ ] **Step 3: Add item 9's rows to the M6 content ledger**

`docs/superpowers/audits/messaging-m6-content-ledger.md` sits at 29 rows; item
8 added rows 26 to 29. Every slice adds its deferred-text rows in the same
commit. Candidates from this slice:

- `spiderweb` has zero rooms and no authored prose; if it is ever used it needs
  its own biome-keyed pool rather than either default.
- `overcast` underground may ship an empty strong pool, so overcast is
  imperceptible underground by design.
- Whether underground wants a higher felt threshold than indoor, so deep stone
  muffles weather more than walls do.

- [ ] **Step 4: Commit**

```bash
git add internal/rooms/biomes.go internal/rooms/context.md modules/weather/content/context.md docs/superpowers/audits/messaging-m6-content-ledger.md
git commit -m "docs: the biome-to-weather-prose coupling, and item 9's ledger rows

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: Playtest gate

The arc ends every content slice with a playtest. This one's fixture problem is
unlike combat's: ambient emotes need **time in a zone**, not a fight that
survives rounds.

- [ ] **Step 1: Shorten the cadence locally**

In the local `_datafiles/config.yaml`, raise `EmoteEveryRounds` frequency and
both `EmoteMildChancePct` / `EmoteStrongChancePct` so lines come fast enough to
sample.

🪤 **`config.yaml` carries the git skip-worktree bit.** `git diff` reports it
clean because git is not comparing. Never `git add` it. Build any commit
touching it from the `git show HEAD:` blob. For a playtest-only change, revert
it by hand afterwards and confirm with `git ls-files -v` that the `S` flag is
still set.

🔧 Local `config.yaml` fixes are Claude's job, never a chore handed to the owner.

- [ ] **Step 2: Collect underground lines**

Stand in a cave zone with active weather. There are 123 cave rooms, so this is
the easy part. Collect until the pool is seen to vary.

Verify: no roofs, panes, shutters, eaves, rafters or floorboards appear
underground. That is the headline defect and it must be visibly gone.

- [ ] **Step 3: Collect surface-indoor lines**

Stand in a `house` or `fort` room (15 and 22 rooms respectively) and confirm
the built-structure prose still reads correctly and was not swept into the
underground rewrite.

- [ ] **Step 4: Collect outdoor lines**

Confirm the padded per-biome pools render and that a `forest` or `swamp` room
draws its own prose rather than `default`.

- [ ] **Step 5: Restore the config and confirm the skip-worktree bit**

Run: `git ls-files -v _datafiles/config.yaml`

Expected: a line beginning with `S`.

- [ ] **Step 6: Extract findings to memory**

Playtest reports are gitignored. Anything learned must be written to a memory
topic file in the same sitting or it is lost.

---

### Task 11: Gate and PR

- [ ] **Step 1: Format, vet, full suite**

Run: `gofmt -l internal modules`

Expected: no output.

Run: `go vet ./internal/... ./modules/...`

Expected: clean.

Run: `go test ./...`

Expected: PASS. Two known false reds, neither from this change:
`internal/playtestrun` under full-suite load, and `internal/rooms` zone
lifecycle tests on Windows under `DOGMUD_BOOT_SMOKE=1`.

- [ ] **Step 2: Confirm no stray files**

Run: `git status --porcelain`

Expected: clean. In particular no `crypt.yaml` left from the Task 4 probe and
no modification to `_datafiles/config.yaml`.

- [ ] **Step 3: Confirm all four guards are green**

Run: `go test ./modules/weather/content/ -run 'TestEveryIndoorBiomeIsClassified|TestClassificationMapsHaveNoDeadKeys|TestAuthoredBiomeKeysAreRealBiomes|TestShippedPoolsMeetMinimumDepth' -v`

Expected: PASS, all four.

- [ ] **Step 4: Push and open the PR**

```bash
git push -u origin feature/messaging-m3-item9-weather-three-way-split
gh pr create --repo pruuk/DOGMud --base master \
  --title "M3 item 9 PR 2: outdoors / indoors / underground" \
  --body-file <path to a body file>
```

**Every `gh` command carries `--repo pruuk/DOGMud`.**

The body should state: 124 of 161 indoor rooms were getting house prose and now
get their own; the `jungle` pools were dead content found by guard 3 on the day
it was written; classification is total and guarded, with guard 1 proven
capable of failing; the depth floor is 6 with empty meaning deliberate silence;
and the golden was re-recorded exactly once.

This closes M3. Next is M4, the flip.
