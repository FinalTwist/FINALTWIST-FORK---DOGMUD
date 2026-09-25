# Graded room lighting, plan 4: weather occlusion

Owner-approved design, 2026-09-25. Parent specs:
`docs/superpowers/specs/2026-09-22-graded-room-lighting-design.md` (plan 4:
"Weather mutators become multiplicative blockers, and the `-2 to 2` `LightMod`
vocabulary is retired in favour of the new scale") and the celestial amendment
`docs/superpowers/specs/2026-09-23-graded-room-lighting-amendment-celestial.md`
("a multiplier is a subtraction on a log scale", so occlusion is a term plan 3
already built).

## Facts verified against source

Read from master `e4d5b8985` on 2026-09-25.

| # | Fact | Source |
|---|---|---|
| 1 | `MutatorSpec.LightMod int` (`yaml:"lightmod"`), commented "-2 to 2"; no validation of its range | `internal/mutators/mutators.go:70`, `Validate` at `:321-336` |
| 2 | Exactly four shipped mutators carry `lightmod`: `weather_blizzard`, `weather_dust`, `weather_storm` at -1, `bioluminescent_caves` at +2 | `_datafiles/world/dogmud/mutators/*.yaml` |
| 3 | `bioluminescent-caves` is referenced by nothing: no room, zone, script or Go file names it | grep of `_datafiles`, `internal`, `modules` |
| 4 | `mutatorLightTerms()` sums positive LightMod into a bridge term and negative LightMod into doubling steps of sky removed; `composeLight(cfg, celestial, lightMod, occlusionSteps)` applies `sky *= 2^-occlusionSteps` and adds the bridge as `DimBelow + (lightMod-1)*step` | `internal/rooms/lighting.go:85-170` |
| 5 | `LightTerms` exposes `OcclusionSteps int` and `LightMod int`; `lightnotice.attribute` compares both (`LightMod` under the lamp cause, `OcclusionSteps` as the weather cause) | `internal/rooms/lighting.go:60-76`, `internal/lightnotice/tracker.go:146-165` |
| 6 | Weather reaches a room as zone mutators (`zc.Mutators`), which `ActiveMutators` yields together with room mutators, skipping `outdooronly` specs in an indoor biome | `modules/weather/engine/apply.go:93-105`, `internal/rooms/rooms.go:2935-2960` |
| 7 | All eight weather mutators are `outdooronly: true`: blizzard, dust, fog, heatwave, overcast, rain, snow, storm | those files, line 2 |
| 8 | Indoor biomes: cave, dungeon, ether, fort, interior, sewer, spiderweb. Only `fort` (skylight 0.35) and `interior` (0.15) have any sky, so they are the only rooms where keeping weather outdoors changes a light value | `_datafiles/world/dogmud/biomes/*.yaml` |
| 9 | The mutator loader is `fileloader.LoadAllFlatFiles`, non-strict YAML: a deleted field's key is silently ignored, not rejected | `internal/mutators/mutators.go:339-344` |
| 10 | `mutator.data.html` renders `$mutator.LightMod` in a -2..+2 select. Templates reach fields by reflection, so deleting the Go field breaks it at render time, not compile time | `_datafiles/html/admin/mutators/mutator.data.html:200-208` |
| 11 | Measured sky (`gametime.CelestialLight`, shipped config): noon 62.8 (day 356), 70.2 (day 81), 73.7 (day 172); moonlit midnights up to 34.3; moonless midnights 11.4 to 13.5. One doubling step is 8 points | scratch measurement over days 172, 81, 356 |
| 12 | Tests that call the bridge signature or read the two fields: `internal/rooms/lighting_model_test.go:64-84`, `internal/rooms/light_terms_test.go`, `internal/lightnotice/tracker_test.go`; comments naming the bridge in `internal/usercommands/look_exit_visibility_test.go:39` and `lighting_parity_golden_test.go:91` | grep |

## What the player gets

Weather now dims the sky by kind, and only the sky: lamps and carried light are
never touched, and roofed rooms are not reached.

| Weather | Sky fraction | Effect on the sky term |
|---|---|---|
| fog, overcast, rain, snow | 0.7 | about 4 points darker |
| storm, dust, blizzard | 0.5 | 8 points darker (one step) |
| heatwave | none | unchanged |

What that means outdoors, from fact 11:

- **Light cloud** leaves a moonlit night at 26 to 30, just above blind, and
  never hides faces by day.
- **Heavy weather** takes a moonlit night to 22 to 26, blind on all but the
  brightest moons, blinds every moonless night, and keeps faces at every noon
  (55 at winter noon, 66 in summer).
- Every player carries `chrysalis-glow`, so darkness is always answerable.

Several active mutators multiply: fog under a storm is `0.7 x 0.5 = 0.35`.

## Architecture

### Data

- `internal/mutators`: **delete `LightMod`**; add `SkyLight *float64`
  (`yaml:"skylight,omitempty"`). Nil means the mutator does not touch the sky.
  `Validate` rejects a value outside 0 to 1. The name and meaning match the
  biome and room `skylight` field: a fraction of the sky that gets through.
- Weather files: `lightmod: -1` becomes `skylight: 0.5` on storm, dust,
  blizzard; `skylight: 0.7` is added to fog, overcast, rain, snow.
- **Delete `bioluminescent_caves.yaml`** (fact 3). It is the last positive
  `lightmod`, and with it goes the positive bridge.

### Rooms

- `mutatorLightTerms` becomes `mutatorSkyFilter() float64`: the product of the
  active mutators' `SkyLight`, 1.0 when none apply.
- `composeLight(cfg, celestial, skyFilter)` multiplies the room's sky fraction
  by the filter. The positive bridge term is deleted; the combine has three
  terms (sky, lamp, carried).
- `LightTerms`: `OcclusionSteps` and `LightMod` are replaced by
  `SkyFilter float64` (1.0 when clear).
- `lightLevelWithMutatorBridge` is renamed to fit, since there is no bridge.
- Indoor rooms stay as they are (fact 8): `outdooronly` keeps weather out of
  `fort` and `interior`. The change there would be about 2 points behind a lamp
  of 50, not worth a special case (approved with this design).

### Light notices

- Weather cause: `SkyFilter` changed. Lamp cause: `HasLamp` or `Lamp` changed.
  The `LightMod` comparison goes.
- Comments and `internal/lightnotice/context.md` drop the bridge; `lamp.yaml`'s
  header comment is corrected (no mutator reaches the lamp lines any more).

### Admin page

`mutator.data.html` replaces the -2..+2 select with a read-only sky-light
display (blank when nil). Every template under `_datafiles/html` is grepped for
`LightMod`, because the compiler cannot find a reflection use (fact 10).

### Method

Delete the Go field first and let the compiler enumerate its consumers
(`dogmud-refactoring`), then the template grep, then the YAML.

## Verification

Each test is shown able to fail before it is trusted.

- **Model table** on `composeLight`: filters 1.0, 0.7, 0.5 and 0.35 remove 0,
  about 4, 8 and about 12 points from the sky term; a lamp and a carried light
  are unchanged by any filter; a room with no sky is unchanged.
- **Filter product**: two active mutators multiply; an `outdooronly` mutator in
  an indoor biome contributes nothing.
- **Shipped data**, days 172, 81 and 356, a room with open sky:
  - under fog, a moonlit midnight is not blind for a normal observer;
  - under a storm, no noon hides faces;
  - under a storm, a moonless midnight is blind.
- **Guards**:
  - no shipped mutator file carries a `lightmod:` key (the loader cannot reject
    it, fact 9, so a file scan does);
  - every shipped `skylight` on a mutator is within 0 to 1, and `Validate`
    refuses 1.5 and -0.1.
- **Light notices**: attribution names weather when only the filter changed,
  and lamp when only the lamp changed; the existing golden does not move.
- **Goldens**: the lighting goldens load mutators but no weather is active in
  them, so none should move. Any move is accounted for row by row.
- **AI companion**: no sight API changes; the full suite, lint and a boot check
  with the companion enabled.

## Out of scope

- Retuning any other light value: plan 6.
- Weather prose written for a house but shown in caves: filed separately.
- Mutators that add light, such as festival lanterns: plan 5's scheduled
  sources, which will choose their own vocabulary.
