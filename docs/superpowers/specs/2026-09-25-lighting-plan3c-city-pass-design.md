# Graded room lighting, plan 3c: the city pass

Owner-approved design, 2026-09-25. Parent spec:
`docs/superpowers/specs/2026-09-22-graded-room-lighting-design.md`. Follows
plan 3b (`docs/superpowers/specs/2026-09-23-lighting-plan3b-biome-vocabulary-design.md`),
which handed this plan three items in its "Out of scope" section: the
main-street versus back-lane split, roughly 50 mis-biomed city interiors, and
`fort`'s split between open and roofed rooms.

## Facts verified against source

Read from master `5fab94896` on 2026-09-25.

| # | Fact | Source |
|---|---|---|
| 1 | **424 rooms declare `biome: city`** across 15 zones, not the 477 the 3b handoff quoted (3b moved 53 out). Per zone: `the_confluence` 128, `greenford` 42, `thornwall_city` 32, `new_plymouth_docks` 30, `stillwater` 29, `new_plymouth_merchant` 26, `hartcharn` 26, `new_plymouth_common` 25, `new_plymouth_old_quarter` 20, `new_plymouth_noble` 20, `new_plymouth_outskirts` 16, `new_plymouth_crafting` 12, `kilnreach_works` 9, `new_plymouth_temple` 7, `pothole_coulee` 2 | grep `^biome: city` under `_datafiles/world/dogmud/rooms` |
| 2 | **12 zones set `defaultbiome: city`**, but no room in them inherits it: every room declares its biome explicitly. The defaults still matter for rooms authored later | `*/zone-config.yaml`; `internal/rooms/roommanager.go:774` |
| 3 | `city` biome: `skylight: 0.95`, `lamp: 35`, `movementcost: 0.7`, `symbol: •`, `burns: false`, no `indoor` | `_datafiles/world/dogmud/biomes/city.yaml` |
| 4 | `fort` biome: `skylight: 0.35`, no lamp, `indoor: true`, `movementcost: 1.0`. **22 rooms**: 17 in `pothole_coulee`, 5 in `test_arena` | `biomes/fort.yaml`; grep |
| 5 | **Rooms can already override `skylight` and `lamp`** (`rooms.go:103-104`), and 3 oasis rooms use `lamp: 38` | `internal/rooms/rooms.go`; `instance_planar_oasis/5003-5005.yaml` |
| 6 | **`indoor` exists on the biome only** (`biomes.go:66`); no room-level override | `internal/rooms/biomes.go` |
| 7 | Bands for a normal observer: blind below 25, shapes below 50, faces at 50, exits visible at 65. Go defaults; `config.yaml` sets none of the three | `internal/configs/config.balance.go:1099-1101` |
| 8 | Every `city` room reads 38 to 42 at night, so shapes only | lighting arc memory, measured in 3a |
| 9 | **Light terms combine on a log scale and the result is never below the brightest term.** Weather occlusion attenuates the SKY term only; a lamp is untouched by weather | `internal/rooms/lighting.go:53-111` |
| 10 | The only Go code keyed on the biome id `city` is the weather climate table, `modules/weather/sim/climate.go:187`. **`fort` is deliberately left unbound there** because it is indoor | grep; `climate.go` comment at the `fort` entry |
| 11 | **The upstream `default` world has its own `biomes/city.yaml`** and uses the `city` climate key. It is out of scope and must keep working | `_datafiles/world/default/biomes/` |
| 12 | A title search for ruin words finds 3 `fort` rooms plus a handful elsewhere (`Ruined Barn`, `The Ruined Waypoint`, `The Crumbling Watchtower`, `Old Chapel Ruin`). Titles are only a first filter | grep of `^title:` |
| 13 | CI's lint gate inverts above **300 files** or 20,000 diff lines | `dogmud-shipping` skill; PR #163 |
| 14 | Crime witnessing reads sight since M5: a witness who sees shapes only records `PerpUnknown` | messaging M5 |

## What changes

### 1. The city splits into two tiers, and `city` is deleted

| Biome | skylight | lamp | Holds |
|---|---|---|---|
| `city_thoroughfare` | 0.95 | **52** | main streets, squares, market places, gates, bridges |
| `city_backstreet` | 0.95 | 35 | side streets, lanes, alleys, courts, yards, slums |

- **Night:** a thoroughfare never reads below 52 (fact 9), so faces are
  readable at any hour and in any weather. A backstreet stays at 38 to 42:
  shapes. By day a thoroughfare gains about 2 points from its lamp, which
  moves no band. Plan 6 owns final values.
- **Everything else is copied from `city` unchanged:** `movementcost: 0.7`,
  `symbol: •`, `burns: false`, no `indoor`. A lighting plan does not retune
  stamina or the map as a side effect.
- **Descriptions** carry `city`'s law-enforcement sentence and add an explicit
  statement of the lighting consequence, written to `dogmud-player-copy`
  (80 columns, no raw numbers). Intent:
  - `city_thoroughfare`: lamplit through the night; faces can be read at any
    hour, which makes it the safest place to be seen and the worst place to go
    unseen.
  - `city_backstreet`: the lamps do not reach here; after dark a normal eye
    makes out shapes but not faces, which is why thieves, and those who hunt
    them, prefer it.
- **`city` is deleted from the `dogmud` world** (`biomes/city.yaml`), as 3b
  deleted `house`. Deleting rather than aliasing is what makes a missed room
  visible.
- **The 12 zone defaults become `city_backstreet`**, so a room authored later
  without thought lands dim, not bright.
- **Climate:** add `city_thoroughfare` and `city_backstreet` to
  `climate.go`, each a copy of the `city` entry. **The `city` key stays**,
  because the `default` world still uses it (fact 11).

### 2. A new `ruins` biome, and `fort` narrows to roofed rooms

| Biome | skylight | lamp | indoor | movementcost | Holds |
|---|---|---|---|---|---|
| `ruins` | 0.75 | none | **false** | 1.0 | built places whose roof is gone |
| `fort` | 0.35 | none | true | 1.0 | unchanged; fortified rooms still roofed |

- A ruin follows the sky, a little shaded by its standing walls, and at night
  is as dark as the country around it. Because it is not indoor, weather
  reaches it. This avoids inventing a room-level `indoor` override (fact 6).
- Description intent: whatever roof it had is gone; it takes the weather and
  the sky as open ground does, a little shaded by what walls still stand, and
  after dark it is as black as the country around it.
- **Climate:** `ruins` is outdoor, so it needs a `climate.go` entry. Use
  `road`'s shape (temperate, travelled ground) unless the plan finds a better
  sibling; it must not be left unbound the way `fort` is.
- `fort` keeps its niche: an unlit roofed stronghold, distinct from the
  lamplit `interior`.
- **Accepted side effect (owner, 2026-09-25):** rooms moving to `ruins` from
  `land`, `plains` or another cheaper biome become slower to cross. Rubble is
  slower. The PR lists every affected room with its old and new cost.

### 3. The ~50 city interiors move to `interior`

3b counted about 50 city rooms that are inside buildings (offices, chapels,
cells, halls, workshops, lofts) and never listed them. This plan finds them
again during the sorting pass below and moves them to the existing `interior`
biome (`skylight: 0.15`, `lamp: 50`, `indoor: true`). No new mechanism.

## The sorting pass

Every one of the 424 `city` rooms and all 22 `fort` rooms is read and
assigned exactly one of `city_thoroughfare`, `city_backstreet`, `interior`,
`ruins` or (for `fort` rooms only) `fort`. Ruin candidates outside those sets
come from a keyword pre-filter over room DESCRIPTIONS (roofless, open to the
sky, collapsed roof, burnt-out, and similar), not titles, and each candidate is
decided by reading it.

- **The rubric lives in the plan,** with worked examples from real rooms for
  each class, including the hard cases: a covered market, a gatehouse passage,
  a courtyard inside a compound, a bridge, a shop front.
- **Read the room, not the title.** Description and exits both count. A
  merely abandoned building that keeps its roof is not a ruin.
- **Unsure means `city_backstreet`, flagged.** A guessed interior or a guessed
  thoroughfare is worse than a known-dim street.
- **The classification ledger comes before any YAML edit.** Each zone's calls
  go into a table (room id, title, old biome, new biome, one-line reason) in
  `docs/superpowers/audits/`, and the owner can spot-check it first.
- **Connectivity check:** a thoroughfare should connect to other
  thoroughfares. An isolated lit room inside backstreets gets a second read.

## Slicing

One PR would touch about 475 files, past CI's 300-file inversion (fact 13).
Two PRs, each under it:

- **3c-1.** Add `city_thoroughfare`, `city_backstreet` and `ruins` with their
  climate entries. Sort the 8 New Plymouth zones (156 rooms), all 22 `fort`
  rooms and the ruin candidates outside the city zones. `city` survives this
  PR. About 200 files.
- **3c-2.** Sort the remaining zones (`the_confluence`, `greenford`,
  `thornwall_city`, `stillwater`, `hartcharn`, `kilnreach_works`, and the 2
  `pothole_coulee` city rooms), change the 12 zone defaults, then **delete
  `biomes/city.yaml`**. About 280 files.

## Verification

Each test below must be shown to fail before it is trusted.

- **No reference to `city` remains in the `dogmud` world:** no room biome, no
  zone default. Scoped to `_datafiles/world/dogmud` so the `default` world and
  the `city` climate key are untouched. Proven by leaving one room on `city`.
  Lands in 3c-2.
- **Thoroughfares stay faces-readable:** every `city_thoroughfare` room reads
  at or above `LightDimBelow` sampled across the year and the clock, in the
  pattern of `internal/rooms/onboarding_light_test.go`. Proven by setting the
  biome's lamp to 35.
- **The split is real:** at least one `city_backstreet` room reads below
  `LightDimBelow` at midnight, so the tiers cannot silently collapse to one.
- **`ruins` is outdoor and follows the sky:** not indoor, and a ruin at noon
  sits within one band of open ground.
- **Climate coverage:** `modules/weather/content/shipped_climate_test.go`
  covers the three new keys.
- **Existing guards stay green,** including the permanent
  `onboarding_light_test.go`.
- **Boot check** on a committed tree: the server starts with no unknown-biome
  warnings.

## Carried traps

- Edit room YAML with the Edit tool, never `sed`: `sed` silently converts CRLF
  files to LF after the first insertion. Confirm CR count equals line count.
- The boot check builds from `HEAD`, which does not include uncommitted work.
- Run `golangci-lint run --new-from-merge-base=origin/master` locally before
  pushing.
- **The AI companion rule:** if PR #161 has merged before this lands, boot
  with the companion module on and confirm its perception still works. 3c
  changes no sight API, so no call-site migration is expected.

## Player-visible consequences

- A patch note names which streets stay lit at night in each town, and says
  that backstreets do not.
- After dark a backstreet witness sees shapes, so a crime there records an
  unknown perpetrator (fact 14). That is the gameplay point of the split, and
  plan 6's playtest exercises it.
- Ruins become weather-exposed and night-dark; roofless rooms that read as
  indoor before now read as outdoors.

## Out of scope

- Transition notices when light changes around a player: plan 3d.
- Retiring `LightMod`: plan 4.
- Carried and cast light sources, dazzle: plan 5.
- Final values for every lamp and sky fraction: plan 6.
- A room-level `indoor` override: not needed, since `ruins` covers the case.
