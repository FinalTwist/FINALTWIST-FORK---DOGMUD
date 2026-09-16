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
  outdoor/indoor); `Tables` (weather type → `Table`); `ParseEmoteTable`; `LoadEmotes`
  (walks a directory, empty tables for a missing directory); `(Tables).Pick`
  and `(SeasonalTables).Pick` (biome → "default" fallback; indoor NEVER falls
  back to outdoor — silence beats wrong prose) resolve a line pool and hand it
  to the unexported `renderAmbient`, which renders it through
  `narration.Render`. Weather is the arc's ACTORLESS store: `renderAmbient`
  populates only `narration.Variants.Observer` and deliberately never invents
  an Actor or Actee. An empty pool renders `""` at every layer — silence beats
  wrong prose. `narration.Picker`'s `[0,n)` contract is no longer policed here:
  a picker that violates it now PANICS (indexing the pool directly inside
  `narration.Render`) instead of being clamped to index 0. The rendered output
  is frozen by `internal/narration/testdata/stores/weather_emotes.golden`.
- **arch_test.go**: purity guardrail — fails if any file imports a
  `GoMudEngine/GoMud/internal` path NOT in the `allowedInternalImports`
  allowlist, which today holds exactly one entry, `internal/narration`.
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

### Key Types
```go
type IndoorPool struct {
    Mild   []string // plays when felt intensity < StrongFeltThreshold (0.5)
    Strong []string // plays when felt intensity >= StrongFeltThreshold
}
type Table struct {
    Weather string                `yaml:"weather"`
    Outdoor map[string][]string   // biome -> lines (outdoor section)
    Indoor  map[string]IndoorPool // biome -> intensity-banded indoor pool
}
type Tables map[sim.WeatherType]Table
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
  selection, indoor-never-falls-back-to-outdoor, picker forwarding across the
  full `[0,n)` range, and the nil-picker-falls-back-to-`DefaultPicker` case
  that is the one behavioural proof the store actually renders through
  `narration.Render` rather than merely accepting an assignable parameter
  type.
- `shipped_climate_test.go` / `shipped_emotes_test.go`: validate shipped YAML
  (see Key Components above).
- `arch_test.go`: engine-import purity guardrail.

All tests run standalone: `go test ./content/...` (no checkout required).
