# internal/lightscale

The arithmetic of the graded light scale. Pure: no config, no globals, no locks.

## What it is for

The scale runs -100 to 100 and is perceptual, not linear. Zero is the darkest
naturally occurring light (an unlit cave). Negative is magical darkness.

One constant relates the scale to physical light: the **doubling step**, how
many points twice as much light is worth. It is `LightDoublingStep` in config,
shipped at 8, and it is passed in rather than read here.

## Surface

| Symbol | Purpose |
|---|---|
| `Absent() float64` | A term that is not present at all, distinct from a dark term |
| `Combine(step float64, terms ...float64) float64` | Every present term together |
| `Attenuate(step, light, fraction float64) float64` | A transmission fraction applied to one term |
| `Polarity` (`Brightens`, `Darkens`) | Which way an adjustable source pushes its room (`trim.go`, lighting plan 5a) |
| `Trim(step, others, max, target float64, p Polarity) float64` | The one adjustment function: the output an adjustable source runs at so the room lands on `target` (`trim.go`) |

## Traps

- **Absent is not zero.** A cave has no sky; a sky contributing zero would make
  the cave brighter, because two terms at zero combine to one step above zero.
  `Attenuate` with fraction 0 returns `Absent()` for this reason.
- **A multiplier is a subtraction here.** Half the light is minus one step.
- `Combine` skips NaN as well as -Inf, so one bad caller cannot poison a room.
- Both functions coerce a non-positive step to 1 rather than dividing by zero.
  `Trim` does the same.
- **`Trim` for a light solves the combine exactly; it is not `target - others`.**
  It returns `Absent()` when the room already reaches `target` without the
  source, or when the needed term would fall below 0 (a light never runs at a
  negative term). A NaN `target` or `max` returns `Absent()` for a light and 0
  for a darkness.
- **`Darkens` is the inverted subtraction** (points cut from the combined
  light, an Absent room counting as 0). Plan 5a ships only the light half;
  darkness sources wire it in a later slice.

## Who uses it

`internal/gametime` (sun plus moons), `internal/rooms` (ambient plus lamp plus
carried light; `Trim` from `light_trim.go`). Plan 4 added weather occlusion;
later plan 5 slices add darkness sources on the same functions.
