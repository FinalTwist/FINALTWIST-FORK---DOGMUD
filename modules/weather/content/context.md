# content Package Context

## Overview
`content` is the pure data-loading layer for the weather module. It parses two
kinds of module data files from an `fs.FS` (in DOGMud: the runtime world
datafiles tree — `weather/climate` and `weather/emotes` under the engine's
configured data path): climate profiles (YAML → `sim.Climate` merged over
`sim.DefaultClimate`) and ambient emote tables (YAML → `Tables` + `Pick`).
`arch_test.go` enforces purity with one narrow, allowlisted exception:
`internal/narration`, the messaging-unification arc's shared rendering core
(imports nothing but stdlib and `internal/util`'s low-level helpers — no game
state, no rooms/mobs/players). No other `internal/` import is permitted. The
only non-stdlib dependency otherwise is `gopkg.in/yaml.v2`, which the GoMud
engine itself uses.

## Key Components
### Core Files
- **climate.go**: `ParseClimate` (one file → biome id + `sim.ClimateProfile`);
  `LoadClimate` (walks a directory, returns `DefaultClimate` overlaid with every
  `*.yaml` found; a missing directory is not an error — pure defaults are
  returned; the first malformed file aborts with an error).
- **emotes.go**: `Table` (per-weather-type ambient lines keyed by biome, split
  into three PROSE CLASSES: outdoor, indoor, underground; see "Three prose
  classes" below); `Tables` (weather type → `Table`); `ParseEmoteTable`;
  `LoadEmotes` (walks a directory, empty tables for a missing directory);
  `(Tables).Pick` and `(SeasonalTables).Pick` (biome → "default" fallback; a
  class NEVER falls back to another class, because silence beats wrong prose)
  resolve a line pool and hand it to the unexported `renderAmbient`, which renders it
  through `narration.Render`. Weather is the arc's ACTORLESS store:
  `renderAmbient` populates only `narration.Variants.Observer` and
  deliberately never invents an Actor or Actee. An empty pool renders `""` at
  every layer — silence beats wrong prose. `narration.Picker`'s `[0,n)`
  contract is no longer policed here: a picker that violates it now PANICS
  (indexing the pool directly inside `narration.Render`) instead of being
  clamped to index 0. The rendered output is frozen by
  `internal/narration/testdata/stores/weather_emotes.golden`.
- **arch_test.go**: purity guardrail — fails if any file imports a
  `GoMudEngine/GoMud/internal` path NOT in the `allowedInternalImports`
  allowlist, which today holds exactly one entry, `internal/narration`.
- **biome_coupling_test.go**: four shipped-data guards that keep the biome
  classification total and the depth contract real. See "Four guards" below.
- **shipped_emotes_test.go**: validates the SHIPPED YAML files under
  `_datafiles/world/dogmud/weather/emotes`. For emote tables: parseable,
  8 tables (one per weather type), outdoor-default pools non-empty, severe
  types have a non-empty strong indoor pool, lines ≤80 chars. Mutator spec
  validation lives in `engine/shipped_specs_test.go`. For mutator specs:
  parseable, `mutatorid` is `weather-` namespaced, filename matches
  `fileNameFor(mutatorid)` (mirrors `util.ConvertForFilename`), `respawnrate`
  forbidden (would fight the orchestrator), `decayintoid` forbidden (upstream
  `MutatorList.Remove` resets `SpawnedRound` and runs `Update` whose decay
  branch has no liveness guard — the decay target is instantly resurrected),
  `decayrate` required (self-heal safety net).
- **doc.go**: package-level comment.

### Three prose classes

The store splits sheltered prose into three classes, not two, because
"sheltered from weather" covers two physically different experiences and one
indoor prose pool used to serve both:

| Class | Distinguishing feel | Biomes | Rooms |
|---|---|---|---|
| `outdoor` | weather in the open, never attenuated | every non-indoor biome | n/a |
| `indoor` (surface) | felt through a BUILT STRUCTURE: roofs, eaves, windows, shutters | `fort`, `house`, `spiderweb` | 22, 15, 0 |
| `underground` | felt through STONE: seepage, draughts, transmitted sound, mineral cold | `cave`, `dungeon` | 123, 1 |

The split exists because 124 of the game's 161 indoor rooms are underground,
and before this class existed every one of them rendered the surface-indoor
pool: prose about rain drumming on a roof, inside a cave with no roof. The
defect is now structurally impossible for any biome that is classified (see
Guard 1 below): there is no shared "indoor" pool left for a new biome to fall
into by accident.

Classification lives in two maps in `emotes.go`:

- `undergroundBiomes`: felt through stone.
- `surfaceIndoorBiomes`: a built structure.

Every biome with `indoor: true` (`rooms.BiomeInfo.Indoor`) must appear in
exactly one of the two; classification is TOTAL by construction, because a
biome that is `indoor: true` but in neither map is exactly the silent
fallback that caused the original defect. `spiderweb` is deliberately in
`surfaceIndoorBiomes`, not `undergroundBiomes`: it is dark and sheltered, but
its darkness is webbing rather than stone, so stone prose would be wrong. It
has zero rooms today, so no prose is authored for it; if it is ever used it
wants its own biome-keyed pool rather than either default (see the M6 ledger
row below).

Class resolves from the `(biome, indoor)` pair inside `bandedSectionLines`,
using the room's existing biome id and its existing `Indoor` flag, the same
two values `engine.EmitAmbient` already passed before the split existed. The
emitter and the whole sim side needed no change to gain the third class.

**A class never falls back to another class.** `bandedSectionLines` picks
`sec.Underground` when the biome is in `undergroundBiomes`, else `sec.Indoor`,
then falls through only `biome -> "default"` WITHIN that one map. An
unauthored underground pool renders `""` (silence) rather than borrowing the
surface-indoor or outdoor pool. Silence beats wrong prose is the same
principle the outdoor/indoor split already used; the three-way split applies
it one more time.

### Depth contract and deliberate silence

`ValidatePool` enforces the depth floor for one pool: empty is legal
(returns `nil` immediately), non-empty must meet `minPoolDepth` (6 lines) via
`narration.ValidateVariants`. An **empty pool is a legal authored value
meaning deliberate silence**, not a gap; weather is the only store in the
arc where silence is itself authored content. Two shapes of it ship today:

- **Mild indoor pools are empty for 6 of the 9 weather types** (`fog`,
  `frost`, `heatwave`, `overcast`, `rain`, `snow`): light weather below
  `StrongFeltThreshold` is inaudible through a wall, so there is nothing to
  say. `blizzard`, `dust` and `storm` do author a mild indoor pool.
- **Underground pools (both bands) are empty for `snow`, `fog` and
  `overcast`**: those three genuinely cannot be perceived through stone, so
  no pool is authored at any intensity for the underground class.

### Fail-soft loading vs. build-time guards

`LoadEmotes` and `LoadSeasonalEmotes` validate every pool at load time
(`validateTableSection`), but a validation failure only aborts THAT load;
the caller, `weatherModule.loadContent`, fails soft on purpose: it logs a
warning and keeps whatever tables loaded before the bad file, which means
silence for the rest of that run. Ambient weather must never stop the world
booting. This is why `biome_coupling_test.go` exists as a separate,
build-failing guard: the load-time path cannot be allowed to hard-fail, so
the depth and classification contract has to be held somewhere that CAN fail
loudly, which is the shipped-data test suite, not runtime.

### Four guards (`biome_coupling_test.go`)

1. **`TestEveryIndoorBiomeIsClassified`**: every `indoor: true` biome in the
   shipped biome files is in exactly one of `undergroundBiomes` /
   `surfaceIndoorBiomes`. The load-bearing guard; proven capable of failing
   with a throwaway `crypt` biome during development.
2. **`TestClassificationMapsHaveNoDeadKeys`**: every key in either map names
   a real, still-`indoor:true` biome. Catches a biome rename or removal that
   the maps were not updated for.
3. **`TestAuthoredBiomeKeysAreRealBiomes`**: every non-"default" biome key
   authored in a weather table (base, seasonal variant, or seasonal-ambience
   table) names a real biome. Found a live defect when written: a `jungle:`
   key authored against a biome that exists in neither world and is used by
   zero rooms, so those lines could never render.
4. **`TestShippedPoolsMeetMinimumDepth`**: every non-empty shipped pool (all
   three classes, base and seasonal) meets `minPoolDepth`. Found six pools
   the content pass had missed.

### Key Types
```go
type IndoorPool struct {
    Mild   []string // plays when felt intensity < StrongFeltThreshold (0.5)
    Strong []string // plays when felt intensity >= StrongFeltThreshold
}
// TableSection is one outdoor/indoor/underground set of biome-keyed lines.
type TableSection struct {
    Outdoor     map[string][]string   `yaml:"outdoor"`
    Indoor      map[string]IndoorPool `yaml:"indoor"`
    Underground map[string]IndoorPool `yaml:"underground"`
}
type Table struct {
    Weather      string `yaml:"weather"`
    TableSection `yaml:",inline"`
    Seasonal map[string]TableSection `yaml:"seasonal"` // keyed by season name
}
type Tables map[sim.WeatherType]Table

// SeasonalTables holds the persistent seasonal-ambience tables (calm-weather
// voice of a season), keyed by (track, season). seasonalEmoteFile mirrors the
// on-disk schema for seasons/*.yaml and MUST carry every class TableSection
// does: yaml.v2 silently drops unknown keys, so a class missing from
// seasonalEmoteFile means prose authored under that key on disk parses,
// is discarded, and is never reported: the same failure mode as an authored
// biome key that matches no known biome (Guard 3). This was a real gap,
// fixed in the PR that added the underground class: seasonalEmoteFile had
// not yet grown an Underground field alongside TableSection's.
type SeasonalKey struct{ Track, Season string }
type SeasonalTables map[SeasonalKey]TableSection
```

## Core Functions
- `ParseClimate([]byte) (string, sim.ClimateProfile, error)` — parse one climate
  YAML into its biome id and profile.
- `LoadClimate(fs.FS, dir string) (sim.Climate, error)` — `sim.DefaultClimate()`
  overlaid with every `*.yaml` under `dir`.
- `ParseEmoteTable([]byte) (Table, error)` — parse one emote table YAML.
- `LoadEmotes(fs.FS, dir string) (Tables, error)` — all emote tables under `dir`.
- `(Tables).Pick(weather sim.WeatherType, biome string, indoor bool, felt float64, season string, pick narration.Picker) string`
  — select one ambient line and render it through `narration.Render`.
  `(SeasonalTables).Pick(track, season, biome string, indoor bool, felt float64, pick narration.Picker) string`
  is the same picker contract for the persistent seasonal-ambience tables. Per
  `narration.Picker`'s contract, `pick(n)` must return `[0,n)`; pass the
  engine's `util.Rand` (or a stub/`narration.FirstPicker`/`SequencePicker` in
  tests) — NEVER the sim RNG, which must stay isolated from presentation
  randomness. A nil `pick` is production behaviour: `narration.Render`
  substitutes `narration.DefaultPicker`. A picker that returns an index
  outside `[0,n)` now panics; see the emotes.go bullet above.

## Dependencies
- `github.com/GoMudEngine/GoMud/modules/weather/sim` (types only).
- `github.com/GoMudEngine/GoMud/internal/narration` — the one allowlisted
  engine import; see `arch_test.go`.
- `gopkg.in/yaml.v2` — the engine's own dependency; the standalone `go.mod`
  carries it for tests; `go.mod`/`go.sum` never travel to checkouts.
- Standard library (`io/fs`, `path`, `strings`, `fmt`).

## Consumers
- Module root (`weather_tick.go`): calls `LoadClimate` and `LoadEmotes` at
  startup; results are stored on `weatherModule` and re-used each tick.
- `engine.EmitAmbient`: receives a `content.Tables` and a `content.SeasonalTables`
  and calls `Pick` with the engine's `util.Rand` as the picker.

## Testing
- `climate_test.go`: `ParseClimate`, reject-missing-biome, `LoadClimate` merges
  override over defaults, missing dir returns pure defaults.
- `emotes_test.go`: `ParseEmoteTable`, `LoadEmotes` missing dir, `Pick` biome
  selection, class resolution (`TestClassResolution`), underground never
  falling back to another class (`TestUndergroundNeverFallsBackToAnotherClass`),
  seasonal-ambience tables carrying an underground section
  (`TestLoadSeasonalEmotesCarriesUndergroundSection`), picker forwarding
  across the full `[0,n)` range, and the nil-picker-falls-back-to-
  `DefaultPicker` case that is the one behavioural proof the store actually
  renders through `narration.Render` rather than merely accepting an
  assignable parameter type.
- `shipped_climate_test.go` / `shipped_emotes_test.go`: validate shipped YAML
  (see Key Components above).
- `biome_coupling_test.go`: the four shipped-data guards; see "Four guards"
  above.
- `arch_test.go`: engine-import purity guardrail.

All tests run standalone: `go test ./content/...` (no checkout required).
