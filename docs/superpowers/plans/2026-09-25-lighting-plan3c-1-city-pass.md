# Lighting plan 3c-1: the city pass, first half

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `city_thoroughfare`, `city_backstreet` and `ruins`, then sort the 8 New Plymouth zones, all 22 `fort` rooms and the world's roofless ruins into them, so a main street is readable at night and a back lane is not.

**Architecture:** Pure data plus tests. Three biome YAMLs, three climate YAMLs, a yaml.v2 anchor that shares each `city:` weather-emote pool with both new tiers, and a one-line `biome:` change per sorted room. Every room call is recorded in a classification ledger the owner spot-checks before any room file changes. `city` survives this PR; 3c-2 deletes it.

**Tech Stack:** Go tests (`internal/rooms`, `modules/weather/content`), YAML content, a read-only Python checker.

**Spec:** `docs/superpowers/specs/2026-09-25-lighting-plan3c-city-pass-design.md`. Read its "Facts verified against source" table first. **Branch:** `feature/lighting-plan3c-city-pass` (spec commits already on it).

**Every commit** ends with:
```
Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
```
(Subagents: use your own model name in that trailer.)

---

## Traps. Read before Task 1.

1. **Test binaries run on Go config defaults, not `config.yaml`**
   (`dogmud-writing-tests`). The light tests below set timing explicitly and
   load the SHIPPED biome YAMLs from disk, so they can fail when the data is
   wrong. A test that seeds a biome in Go (as `onboarding_light_test.go` does)
   cannot catch a bad data file.
2. **Every room, biome and weather YAML is stored LF** (`git ls-files --eol`
   shows `i/lf` for all 1,354 room files and all 61 weather/biome files).
   Git Bash `sed -i` strips CRs from the working copy; that is harmless here
   because `core.autocrlf` restores them and the commit is LF either way.
   **The check that matters:** after an edit pass, `git diff --numstat` shows
   exactly `1 1` for every room file touched. Anything else is a bug.
3. **Never `git add -A` or `git add .`** Named paths only; for a zone,
   `git add _datafiles/world/dogmud/rooms/<zone>/` is a named path.
4. **`grep -c` exits 1 on zero matches** and silently breaks an `&&` chain.
   Run expect-zero checks standalone.
5. **`go test ./...`, never `go test .`** for a full run.
6. **The boot check builds from `HEAD`.** Commit before booting.
7. **CI lint inverts above 300 files.** Count before pushing:
   `git diff --name-only origin/master...HEAD | wc -l` must be under 300.
8. **A guessed interior is worse than a known-dim street.** Unsure means
   `city_backstreet`, flagged in the ledger.

## File structure

| Path | Change | Responsibility |
|---|---|---|
| `internal/rooms/city_tier_light_test.go` | create | Thoroughfare readable all year; backstreet dim at midnight; ruins outdoor and sky-following. Reads shipped biomes |
| `_datafiles/world/dogmud/biomes/city_thoroughfare.yaml` | create | Main-street tier, lamp 52 |
| `_datafiles/world/dogmud/biomes/city_backstreet.yaml` | create | Everything-else tier, lamp 35 |
| `_datafiles/world/dogmud/biomes/ruins.yaml` | create | Roofless built places, outdoor, skylight 0.75 |
| `_datafiles/world/dogmud/weather/climate/{city_thoroughfare,city_backstreet,ruins}.yaml` | create | Weather simulation profiles |
| `modules/weather/content/shipped_climate_test.go` | modify | Cover the three new climate keys |
| `modules/weather/content/city_tier_emotes_test.go` | create | Both tiers carry the urban emote lines |
| `_datafiles/world/dogmud/weather/emotes/{fog,frost,heatwave,rain,storm}.yaml`, `emotes/seasons/temperate_winter.yaml` | modify | Anchor each `city:` pool, alias it to both tiers |
| `tools/city_tier_ledger.py` | create | Read-only: builds candidate rows, checks connectivity, applies nothing |
| `docs/superpowers/audits/2026-09-25-lighting-plan3c-ledger.md` | create | Every room call with a reason |
| `_datafiles/world/dogmud/rooms/<8 NP zones>/*.yaml`, `pothole_coulee`, `test_arena`, ruin candidates | modify | One `biome:` line each |
| `internal/rooms/context.md`, `docs/PATCH_NOTES.md`, `docs/README.md` | modify | Vocabulary, patch note, index |

---

### Task 1: The light tests, failing first

**Files:**
- Create: `internal/rooms/city_tier_light_test.go`

- [ ] **Step 1: Write the tests**

```go
package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// withShippedBiomesAndClock loads the REAL biome YAMLs from
// _datafiles/world/dogmud/biomes (so a wrong data file fails these tests) and
// pins the clock config the light model needs. It follows loadBiomesForTest's
// rule of never calling ReloadConfig, for the reason documented there.
func withShippedBiomesAndClock(t *testing.T) {
	t.Helper()

	origBiomes := biomes
	t.Cleanup(func() { biomes = origBiomes })

	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = `../../_datafiles/world/dogmud`
	cfg.Timing.RoundsPerDay = 900
	cfg.Timing.RoundSeconds = 4
	cfg.Timing.Validate()
	cfg.Balance.Validate() // shipped latitude, deliberately not overridden
	configs.SetConfigForTest(t, cfg)

	LoadBiomeDataFiles()

	gametime.ClearDateCacheForTest()
	t.Cleanup(gametime.ClearDateCacheForTest)

	original := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCountForTest(original) })
}

// setClock moves the world to a day of year and an hour. 900 rounds per day,
// so one hour is 37.5 rounds.
func setClock(doy int, hour float64) {
	util.SetRoundCountForTest(uint64(float64(doy-1)*900 + hour*37.5))
	gametime.ClearDateCacheForTest()
}

func requireBiome(t *testing.T, id string) {
	t.Helper()
	if _, ok := GetBiome(id); !ok {
		t.Fatalf("biome %q is not shipped in _datafiles/world/dogmud/biomes", id)
	}
}

// TestThoroughfaresAreFaceReadableAllYear is the promise the patch note makes:
// a main street can be walked after dark with faces readable. Sampled across
// the year and the clock, because the failure mode is midwinter midnight.
func TestThoroughfaresAreFaceReadableAllYear(t *testing.T) {
	withShippedBiomesAndClock(t)
	requireBiome(t, "city_thoroughfare")

	dim := configs.GetLightingConfig().DimBelow
	room := Room{Biome: "city_thoroughfare"}
	for _, doy := range []int{356, 81, 172} {
		for _, hour := range []float64{0, 3, 6, 12, 18, 21} {
			setClock(doy, hour)
			if got := room.LightLevel(); got < dim {
				t.Errorf("day %d hour %v: city_thoroughfare reads %d, below DimBelow %d; "+
					"a main street must show faces at any hour", doy, hour, got, dim)
			}
		}
	}
}

// TestBackstreetsHideFacesAtMidnight keeps the two tiers from collapsing into
// one. If someone raises the backstreet lamp to match, this goes red.
func TestBackstreetsHideFacesAtMidnight(t *testing.T) {
	withShippedBiomesAndClock(t)
	requireBiome(t, "city_backstreet")

	cfg := configs.GetLightingConfig()
	room := Room{Biome: "city_backstreet"}
	for _, doy := range []int{356, 81, 172} {
		setClock(doy, 0)
		got := room.LightLevel()
		if got >= cfg.DimBelow {
			t.Errorf("day %d midnight: city_backstreet reads %d, at or above DimBelow %d; "+
				"a back lane must hide faces after dark", doy, got, cfg.DimBelow)
		}
		if got < cfg.BlindBelow {
			t.Errorf("day %d midnight: city_backstreet reads %d, below BlindBelow %d; "+
				"a lane is dim, not blind", doy, got, cfg.BlindBelow)
		}
	}
}

// TestRuinsAreOutdoorAndFollowTheSky pins what makes a ruin a ruin: weather
// reaches it, faces read at noon, and nothing lights it at night.
func TestRuinsAreOutdoorAndFollowTheSky(t *testing.T) {
	withShippedBiomesAndClock(t)
	requireBiome(t, "ruins")

	b, _ := GetBiome("ruins")
	if b.Indoor {
		t.Fatal("ruins is indoor; a roofless room must take the weather")
	}
	if _, ok := b.LampValue(); ok {
		t.Fatal("ruins declares a lamp; nothing lights a ruin")
	}

	dim := configs.GetLightingConfig().DimBelow
	room := Room{Biome: "ruins"}
	for _, doy := range []int{356, 81, 172} {
		setClock(doy, 12)
		if got := room.LightLevel(); got < dim {
			t.Errorf("day %d noon: ruins reads %d, below DimBelow %d; a roofless room is lit by day",
				doy, got, dim)
		}
		setClock(doy, 0)
		if got := room.LightLevel(); got >= dim {
			t.Errorf("day %d midnight: ruins reads %d, at or above DimBelow %d; a ruin is dark at night",
				doy, got, dim)
		}
	}
}
```

- [ ] **Step 2: Signatures this test relies on** (verified 2026-09-25 on master `5fab94896`)

`GetBiome(name string) (*BiomeInfo, bool)` (`internal/rooms/biomes.go:201`) ·
`(*BiomeInfo).LampValue() (int, bool)` (`:97`) · `BiomeInfo.Indoor bool`
(field, no method) · `configs.GetLightingConfig() Lighting` with `BlindBelow`
and `DimBelow` ints (`internal/configs/config.lighting_accessor.go:11-29`) ·
`util.GetRoundCount() uint64` / `util.SetRoundCountForTest(uint64)`
(`internal/util/util.go:142-147`). If any has changed since, adapt the test,
not the production code.

- [ ] **Step 3: Run and confirm they fail for the RIGHT reason**

Run: `go test -count=1 -run 'Thoroughfares|Backstreets|RuinsAre' ./internal/rooms/`
Expected: three FAILs, each `biome "<id>" is not shipped`. A failure for any
other reason (config, clock, panic) means the helper is wrong; fix it first.

- [ ] **Step 4: Commit**

```bash
git add internal/rooms/city_tier_light_test.go
git commit -m "test(lighting): pin the city tiers and ruins against the shipped biomes"
```

---

### Task 2: The three biomes

**Files:**
- Create: `_datafiles/world/dogmud/biomes/city_thoroughfare.yaml`, `city_backstreet.yaml`, `ruins.yaml`

Descriptions follow `dogmud-player-copy`: wrap at 80 columns, no raw numbers.
Everything except id, name, description and the light fields is copied from
the source biome unchanged (spec: no stamina or map retune as a side effect).

- [ ] **Step 1: `city_thoroughfare.yaml`**

```yaml
biomeid: city_thoroughfare
name: City Thoroughfare
symbol: •
description: The main ways through a city, its squares, markets, gates and
  bridges. Law enforcement will attempt to subdue those who murder or steal.
  Lamps burn here all night, so faces can be read at any hour. That makes a
  thoroughfare the safest place in a city to be seen, and the worst place to
  go unseen.
skylight: 0.95
lamp: 52
requireditemid: 0
usesitem: false
burns: false
movementcost: 0.7
```

- [ ] **Step 2: `city_backstreet.yaml`**

```yaml
biomeid: city_backstreet
name: City Backstreet
symbol: •
description: The side streets, lanes, alleys, courts and yards off a city's
  main ways. Law enforcement will attempt to subdue those who murder or steal,
  when it can see them. The lamps do not reach here. After dark a normal eye
  makes out shapes but not faces, which is why thieves prefer backstreets, and
  why those who hunt thieves do too.
skylight: 0.95
lamp: 35
requireditemid: 0
usesitem: false
burns: false
movementcost: 0.7
```

- [ ] **Step 3: `ruins.yaml`**

```yaml
biomeid: ruins
name: Ruins
symbol: •
description: A built place whose roof is gone, a fallen barracks, a cracked
  dome, a burnt-out barn. It takes the weather and the sky as open ground
  does, a little shaded by what walls still stand. Nothing lights it, and
  after dark it is as black as the country around it. Rubble makes it slow
  going.
skylight: 0.75
requireditemid: 0
usesitem: false
burns: false
movementcost: 1.0
```

- [ ] **Step 4: Run the light tests**

Run: `go test -count=1 -run 'Thoroughfares|Backstreets|RuinsAre|EveryShippedBiome' ./internal/rooms/`
Expected: PASS. If `TestRuinsAreOutdoorAndFollowTheSky` fails at noon on day
356, read the actual value before touching anything: midwinter noon at
latitude 46.5 attenuated to 0.75 may sit near 50. Report the number; the fix
is a skylight value the owner approves, not a loosened test.

- [ ] **Step 5: Prove the thoroughfare test can fail**

Temporarily set `lamp: 35` in `city_thoroughfare.yaml`, rerun, confirm
`TestThoroughfaresAreFaceReadableAllYear` FAILS at hour 0, restore `52`,
rerun, confirm PASS. Record the failing output in the commit message body.

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/biomes/city_thoroughfare.yaml _datafiles/world/dogmud/biomes/city_backstreet.yaml _datafiles/world/dogmud/biomes/ruins.yaml
git commit -m "feat(lighting): city_thoroughfare, city_backstreet and ruins biomes"
```

---

### Task 3: Climate profiles

**Files:**
- Modify: `modules/weather/content/shipped_climate_test.go:22-24`
- Create: `_datafiles/world/dogmud/weather/climate/city_thoroughfare.yaml`, `city_backstreet.yaml`, `ruins.yaml`

- [ ] **Step 1: Extend the test's biome list**

In `TestShippedDogmudClimateProfiles`, change the list to:

```go
	biomes := []string{"water", "shore", "cliffs", "desert", "snow", "mountains",
		"swamp", "forest", "farmland", "land", "road", "city",
		"city_thoroughfare", "city_backstreet", "ruins",
		"cave", "dungeon", "fort", "spiderweb"}
```

and add, after the indoor loop, a pin that the tiers copied `city` exactly:

```go
	// The two city tiers are a LIGHTING split, not a weather one: they must
	// carry city's climate exactly (plan 3c).
	for _, tier := range []string{"city_thoroughfare", "city_backstreet"} {
		if climate[tier].SpawnWeight != climate["city"].SpawnWeight ||
			len(climate[tier].Weather) != len(climate["city"].Weather) {
			t.Errorf("%s: climate differs from city's; the tiers split light only", tier)
		}
	}
```

(3c-2 deletes `city`; that plan replaces this comparison with literal values.)

- [ ] **Step 2: Run, expect FAIL**

Run: `go test -count=1 -run TestShippedDogmudClimateProfiles ./modules/weather/content/`
Expected: FAIL, `missing climate profile for biome "city_thoroughfare"` (and the other two).

- [ ] **Step 3: Create the three files**

`city_thoroughfare.yaml` and `city_backstreet.yaml` are `city.yaml` with only the
`biome:` line changed:

```yaml
biome: city_thoroughfare
weather: { clear: 4, overcast: 2, rain: 2, fog: 1 }
influence: { intensityDelta: -0.01, moistureDelta: 0.0, movementResistance: 0.05 }
spawnWeight: 0.7
track: temperate
```

```yaml
biome: city_backstreet
weather: { clear: 4, overcast: 2, rain: 2, fog: 1 }
influence: { intensityDelta: -0.01, moistureDelta: 0.0, movementResistance: 0.05 }
spawnWeight: 0.7
track: temperate
```

`ruins.yaml` is `road.yaml` with the `biome:` line changed:

```yaml
biome: ruins
weather: { clear: 4, overcast: 2, rain: 1 }
influence: { intensityDelta: 0.0, moistureDelta: 0.0, movementResistance: 0.0 }
spawnWeight: 0.6
track: temperate
```

Before saving, `diff` each against its source to confirm only `biome:` differs.

- [ ] **Step 4: Run, expect PASS**, plus the coupling guard

Run: `go test -count=1 ./modules/weather/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add modules/weather/content/shipped_climate_test.go _datafiles/world/dogmud/weather/climate/city_thoroughfare.yaml _datafiles/world/dogmud/weather/climate/city_backstreet.yaml _datafiles/world/dogmud/weather/climate/ruins.yaml
git commit -m "feat(weather): climate for the city tiers and ruins"
```

---

### Task 4: The urban emote lines follow both tiers

**Files:**
- Create: `modules/weather/content/city_tier_emotes_test.go`
- Modify: `_datafiles/world/dogmud/weather/emotes/{fog,frost,heatwave,rain,storm}.yaml`, `emotes/seasons/temperate_winter.yaml`

Six files carry an outdoor `city:` pool (spec fact 10). Without this task every
sorted room loses its urban weather lines and falls back to `default`.

- [ ] **Step 1: Write the test**

```go
package content

import (
	"os"
	"reflect"
	"testing"
)

// TestCityTiersShareTheCityPools: the city split is about light, so both tiers
// must say the same urban things about the weather. Holds before and after
// 3c-2 deletes the city key: wherever any of the three keys appears, both
// tiers must be present and identical, and equal to city while city exists.
func TestCityTiersShareTheCityPools(t *testing.T) {
	root := os.DirFS("../../../_datafiles/world/dogmud")

	check := func(where string, sec TableSection) bool {
		city, hasCity := sec.Outdoor["city"]
		th, hasTh := sec.Outdoor["city_thoroughfare"]
		bk, hasBk := sec.Outdoor["city_backstreet"]
		if !hasCity && !hasTh && !hasBk {
			return false
		}
		if !hasTh || !hasBk {
			t.Errorf("%s: an urban pool exists but city_thoroughfare=%v city_backstreet=%v",
				where, hasTh, hasBk)
			return true
		}
		if !reflect.DeepEqual(th, bk) {
			t.Errorf("%s: city_thoroughfare and city_backstreet pools differ", where)
		}
		if hasCity && !reflect.DeepEqual(city, th) {
			t.Errorf("%s: the tiers' pool differs from city's", where)
		}
		return true
	}

	found := 0
	tables, err := LoadEmotes(root, "weather/emotes")
	if err != nil {
		t.Fatalf("LoadEmotes: %v", err)
	}
	for wt, tbl := range tables {
		if check(string(wt), tbl.TableSection) {
			found++
		}
		for season, sec := range tbl.Seasonal {
			if check(string(wt)+" season:"+season, sec) {
				found++
			}
		}
	}
	seasonal, err := LoadSeasonalEmotes(root, "weather/emotes/seasons")
	if err != nil {
		t.Fatalf("LoadSeasonalEmotes: %v", err)
	}
	for k, sec := range seasonal {
		if check("ambience "+k.Track+"/"+k.Season, sec) {
			found++
		}
	}
	// Six pools shipped when this was written; fewer means one was lost.
	if found < 6 {
		t.Errorf("found %d urban pools, want at least 6", found)
	}
}
```

`SeasonalTables` is `map[SeasonalKey]TableSection` with
`SeasonalKey struct{ Track, Season string }` (`emotes.go:309-314`), hence the
`k.Track+"/"+k.Season` label.

- [ ] **Step 2: Run, expect FAIL**

Run: `go test -count=1 -run TestCityTiersShareTheCityPools ./modules/weather/content/`
Expected: six errors, `an urban pool exists but city_thoroughfare=false`.

- [ ] **Step 3: Anchor and alias each pool**

In each of the six files, turn the `city:` key into an anchor and add two
aliases directly beneath the pool, at the same indent. For `fog.yaml`:

```yaml
  city: &city_fog
    - "Lamplight and doorways become dim halos in the drifting murk."
    # ...the existing lines, unchanged...
  city_thoroughfare: *city_fog
  city_backstreet: *city_fog
```

Anchor names: `&city_fog`, `&city_frost`, `&city_heatwave`, `&city_rain`,
`&city_storm`, `&city_winter`. Change nothing else in the files.

- [ ] **Step 4: Run, expect PASS**, plus every weather guard

Run: `go test -count=1 ./modules/weather/...`
Expected: PASS, including `biome_coupling_test.go`
(`TestAuthoredBiomeKeysAreRealBiomes` accepts the new keys because Task 2
shipped the biomes; `TestShippedPoolsMeetMinimumDepth` sees full pools).

- [ ] **Step 5: Prove the test can fail**

Delete one alias line (say `city_backstreet: *city_rain`), rerun, confirm FAIL
naming `rain`, restore it, rerun, confirm PASS.

- [ ] **Step 6: Commit**

```bash
git add modules/weather/content/city_tier_emotes_test.go _datafiles/world/dogmud/weather/emotes/fog.yaml _datafiles/world/dogmud/weather/emotes/frost.yaml _datafiles/world/dogmud/weather/emotes/heatwave.yaml _datafiles/world/dogmud/weather/emotes/rain.yaml _datafiles/world/dogmud/weather/emotes/storm.yaml _datafiles/world/dogmud/weather/emotes/seasons/temperate_winter.yaml
git commit -m "feat(weather): both city tiers keep the urban weather lines"
```

---

### Task 5: The ledger tool

**Files:**
- Create: `tools/city_tier_ledger.py`

A READ-ONLY tool. It writes nothing but stdout. Two subcommands:

- `candidates <zone>...` prints a markdown table of every `biome: city` or
  `biome: fort` room in the named zones: id, title, current biome, exits
  (direction → room id), and the first 240 characters of the description, one
  row per room, for the classifier to read alongside the full file.
- `ruins-scan` prints every room in the world, on any biome, whose
  DESCRIPTION (not title) matches
  `roofless|open to the sky|roof (has |had )?(fallen|collapsed|gone)|collapsed roof|burnt[- ]out|burned[- ]out|no roof|sky (shows|shines) through|caved[- ]in roof`
  (case-insensitive), with id, zone, title, biome and the matching sentence.
- `check <ledger.md>` reads the ledger's tables and, for every row classed
  `city_thoroughfare`, reports it if none of its exits leads to another room
  classed `city_thoroughfare` (by the ledger, or by the room's current biome
  when the neighbour is not in the ledger). It also reports any ledger row
  whose room id does not exist, or appears twice.

Parse room YAML with `yaml.safe_load`. Descriptions sit under `description:`;
exits under `exits: {<dir>: {roomid: N}}`. Zone folder names are the zone
keys used on the command line.

- [ ] **Step 1: Write the tool** following the description above.
- [ ] **Step 2: Smoke it**: `python tools/city_tier_ledger.py candidates new_plymouth_temple` prints 7 rows; `python tools/city_tier_ledger.py ruins-scan` prints a table and exits 0.
- [ ] **Step 3: Commit**

```bash
git add tools/city_tier_ledger.py
git commit -m "tools: read-only ledger helper for the lighting 3c city pass"
```

---

### Task 6: Classify New Plymouth, `fort` and the ruins into the ledger

**Files:**
- Create: `docs/superpowers/audits/2026-09-25-lighting-plan3c-ledger.md`

This is judgement work. Dispatch one subagent per zone group, in parallel,
each READ-ONLY on room files and writing only its own section of the ledger
(the controller assembles the sections):

| Group | Zones / source | Rooms |
|---|---|---|
| A | `new_plymouth_docks`, `new_plymouth_crafting`, `new_plymouth_temple` | 49 |
| B | `new_plymouth_merchant`, `new_plymouth_noble` | 46 |
| C | `new_plymouth_common`, `new_plymouth_old_quarter`, `new_plymouth_outskirts` | 61 |
| D | all 22 `fort` rooms (`pothole_coulee`, `test_arena`) + `ruins-scan` output | 22 + scan |

**The rubric** (copy it verbatim into every subagent prompt):

| Class | It is this when | Worked examples |
|---|---|---|
| `city_thoroughfare` | A named main way through the city, a square or plaza, an open market, a city or district gate, a bridge, a quay a crowd would use | `5802` The Central Square · `5800` The Long Market, Merchant Gate · `5510` Dock Street · `5601` Lower Common Way |
| `city_backstreet` | Any outdoor city space that is not a main way: side streets, lanes, alleys, rows, courts, yards, steps, slums. Also the default when unsure | `5515` Cutter's Lane · `5822` Coin Alley · `5615` The Back Court · `5617` The Tanner's Yard |
| `interior` | Inside a roofed building: shop, office, tavern, taproom, bathhouse, hall, chapel, cell, loft, cellar, rooms above a shop | `5507` The Harbormaster's Office · `5512` The Salt Cellar Taproom · `5518` Old Sable's Pawnshop · `5609` The Bathhouse |
| `ruins` | Built, and its roof is GONE, so the sky shows | `Collapsed Barracks`, `Cracked Dome` (fort) — verify by reading |
| `fort` | (group D only) A fortified room that still has its roof | `Tower Stair`, `The Watch Room`, `Buried Vault` — verify by reading |

Rules for every call:
1. Read the whole room file: description AND exits AND nouns. The title is a
   hint, not evidence. "Market Stalls" in a square is outdoor; a "covered
   market" hall is `interior`.
2. A shop FRONT the player stands outside of, on the street, is the street's
   class. A room whose description puts the player inside is `interior`.
3. "Abandoned" is not "ruined": a roofed empty building is `interior` (or
   `fort` for fort rooms).
4. A gatehouse PASSAGE under a roof is `interior`; the gate's open approach
   is `city_thoroughfare`.
5. Unsure means `city_backstreet` for city rooms and `fort` for fort rooms,
   with `UNSURE:` at the start of the reason.
6. Ruin-scan rooms not in group D keep their current biome unless the
   description clearly says the roof is gone; record every scanned room either
   way, with the decision.

**Ledger format** (one table per zone, then one for fort, one for the scan):

```markdown
## new_plymouth_docks (30)

| Room | Title | Old | New | Reason |
|---|---|---|---|---|
| 5500 | River Road Landing | city | city_thoroughfare | The river road's landing, where the main road meets the quays |
```

Close the ledger with a summary table: count per new class per zone, the
number of `UNSURE` rows, and every room moving to `ruins` with its old
biome's `movementcost` and the new 1.0 (spec: owner accepted the cost
change; the PR lists it).

- [ ] **Step 1: Dispatch groups A to D** with the rubric, the rules, the tool
  commands, and the ledger format.
- [ ] **Step 2: Assemble** the sections into the ledger file.
- [ ] **Step 3: Run the checker**: `python tools/city_tier_ledger.py check docs/superpowers/audits/2026-09-25-lighting-plan3c-ledger.md`. Every isolated thoroughfare gets a second read; record the outcome in its reason. Zero missing or duplicate ids.
- [ ] **Step 4: Commit the ledger**

```bash
git add docs/superpowers/audits/2026-09-25-lighting-plan3c-ledger.md
git commit -m "docs(lighting): 3c-1 classification ledger for New Plymouth, fort and ruins"
```

- [ ] **Step 5: 🛑 OWNER CHECKPOINT.** Stop. Tell the owner the ledger is
  ready, with the summary table, the `UNSURE` rows and every `ruins` row
  inline. Do not touch a room file until the owner has spot-checked it and
  any corrections are made in the ledger.

---

### Task 7: Apply the ledger

**Files:**
- Modify: one `biome:` line in every ledger row whose `New` differs from `Old`.

- [ ] **Step 1: Apply per zone.** For each row, replace the room file's
  `biome: <old>` line with `biome: <new>`, for example:

```bash
f=_datafiles/world/dogmud/rooms/new_plymouth_docks/5507.yaml
grep -c '^biome: city$' "$f"    # must print 1 before editing
sed -i 's/^biome: city$/biome: interior/' "$f"
```

  Drive it from the ledger rows; do not hand-type 150 commands.

- [ ] **Step 2: Verify each zone before committing it**

```bash
git diff --numstat -- _datafiles/world/dogmud/rooms/new_plymouth_docks/
```

  Every line must read `1	1	<path>`. Then confirm the zone's remaining
  `biome: city` count equals the ledger's rows that stay `city` for it (none
  should, in the 8 NP zones):

```bash
grep -l '^biome: city$' _datafiles/world/dogmud/rooms/new_plymouth_docks/*.yaml
```

  Expect no output (run it standalone; trap 4).

- [ ] **Step 3: Commit one zone at a time**, named paths:

```bash
git add _datafiles/world/dogmud/rooms/new_plymouth_docks/
git commit -m "content(lighting): sort new_plymouth_docks into the city tiers"
```

  Repeat for the other seven NP zones, then `fort` (both zones' fort rooms),
  then the ruin-scan rooms as one commit.

- [ ] **Step 4: Full suite**

Run: `go test -count=1 ./...`
Expected: PASS. A room-data test that fails names a real problem; read it
before touching the test.

---

### Task 8: Documentation and the patch note

**Files:**
- Modify: `internal/rooms/context.md`, `docs/PATCH_NOTES.md`, `docs/README.md`

- [ ] **Step 1: `internal/rooms/context.md`.** In the biome vocabulary section
  3b wrote, add the three biomes with sky, lamp and one line each, and note
  that `city` is being retired (3c-2 deletes it). Verify every symbol you
  name exists (`tools/context_md_audit.py`).
- [ ] **Step 2: `docs/PATCH_NOTES.md`.** A player-facing entry per
  `dogmud-player-copy`: New Plymouth's main streets, squares and gates now
  stay lamplit through the night; its lanes and alleys do not, and faces
  cannot be made out there after dark; many shops and halls that were lit
  like streets are now proper interiors; roofless ruins are open to the
  weather and dark at night. Name New Plymouth's lit ways from the ledger.
  No numbers.
- [ ] **Step 3: `docs/README.md`.** Index this plan and the ledger in the
  audits/plans tables, one row each, in the style of the rows beside them.
- [ ] **Step 4: Commit**

```bash
git add internal/rooms/context.md docs/PATCH_NOTES.md docs/README.md
git commit -m "docs(lighting): 3c-1 vocabulary, patch note and index"
```

---

## Before opening the PR

1. `go test -count=1 ./...` green.
2. `golangci-lint run --new-from-merge-base=origin/master` clean.
3. `git diff --name-only origin/master...HEAD | wc -l` under 300.
4. Boot check on the committed tree (`dogmud-shipping`): the server starts,
   and the log has no unknown-biome warning.
5. If the AI companion (PR #161) has merged, boot with it enabled and confirm
   a companion still perceives a sorted room. 3c changes no sight API.
6. PR body: the ledger link, the summary table, the `ruins` movement-cost
   list, and the owner's checkpoint noted. `gh ... --repo pruuk/DOGMud`.

## What 3c-2 does (its own plan, written after 3c-1 merges)

The remaining city zones (`the_confluence` 128, `greenford` 42,
`thornwall_city` 32, `stillwater` 29, `hartcharn` 26, `kilnreach_works` 9,
`pothole_coulee` 2) through the same tool, rubric, ledger and owner
checkpoint; the 12 zone `defaultbiome: city` lines to `city_backstreet`; then
delete `biomes/city.yaml` and `weather/climate/city.yaml`, move each emote
anchor onto `city_thoroughfare` and drop the `city:` keys (the coupling guard
proves none remain), replace Task 3's `city` comparison with literal values,
and add a guard that no `dogmud` room or zone names `city`, proven by leaving
one room on it. Written after 3c-1 so it inherits whatever the first ledger
teaches.
