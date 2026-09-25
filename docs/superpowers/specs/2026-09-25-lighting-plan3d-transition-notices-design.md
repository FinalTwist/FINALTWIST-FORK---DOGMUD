# Graded room lighting, plan 3d: transition notices

Owner-approved design, 2026-09-25. Parent specs:
`docs/superpowers/specs/2026-09-22-graded-room-lighting-design.md` and its
celestial amendment `docs/superpowers/specs/2026-09-23-graded-room-lighting-amendment-celestial.md`,
whose "Transition notices" section (line 310) is the owner requirement this
plan implements: player-facing text when light changes, on two triggers
(moving between rooms whose light differs, and a room's light crossing a band
around a standing player), firing on a BAND change for the observer, not on
every numeric change, with text going through `internal/narration`.

## Facts verified against source

Read from master `3887ed1dc` on 2026-09-25.

| # | Fact | Source |
|---|---|---|
| 1 | Sight for a party is `messaging.ParticipantSight(observer, room RoomVisibility) SightDecision`: Blinded perception returns none; otherwise `SightThroughWindow(room.LightLevel(), observer.NightVisionStrength(), observer.InfraReach(), LightBlindBelow, LightDimBelow)` | `internal/messaging/predicates.go:56-89` |
| 2 | `SightThroughWindow` returns full/shapes/none; the dazzle edge `windowDazzleEdge = 75` is declared but NOT read ("Perfect and too-bright both read fully"). Strength shifts every edge down, capped at 24 | `internal/messaging/window.go:17-68` |
| 3 | A normal observer is never dazzled today: natural light peaks at 73 and the brightest authored lamp is 52. A Cat's Eye Draught drinker (strength 24) has a dazzle edge of 51, so daylight and every `city_thoroughfare` (lamp 52) dazzle them | `window.go`; `config.balance.go:1174`; `biomes/city_thoroughfare.yaml` |
| 4 | Sleep is `c.HasConditionFlag(conditions.Sleeping)`; ParticipantSight deliberately ignores it, the `awake` helper composes it | `predicates.go:91-95` |
| 5 | Room light is computed in `(*Room).lightLevelWithMutatorBridge` from four terms: sky (celestial × sky fraction, attenuated by weather occlusion steps), the room lamp, the positive `LightMod` bridge, and any carried light (`FindHasLight`) | `internal/rooms/lighting.go:53-111` |
| 6 | A global sunrise/sunset splash reaches EVERY online player, indoors and underground included, as art plus a caption, on the `DayNightCycle` event | `internal/hooks/DayNightCycle_NotifySunriseSunset.go`, `Splash_Deliver.go:34-38` |
| 7 | `Awareness_LightChange.go` subscribes to `RoomChange` and `EquipmentChange`; both bodies are stealth re-roll stubs | the hook |
| 8 | User commands pass through `usercommands.TryCommand(cmd, rest, userId, flags)`; combat rounds through `NewRound_DoCombat`; login fires `events.PlayerSpawn` | `internal/usercommands/usercommands.go:316`, `internal/hooks/`, `internal/events/eventtypes.go:285` |
| 9 | `CategoryTimeOfDay` wraps as prose (`pipeline.go:138`), skips name normalisation (`normalize.go:28`), is suppressed by no verbosity tier (`verbosity.go:54-70`), and its only sender is the `time` command (`modules/time/time.go:69`) | those files |
| 10 | Narration stores follow `internal/movenarration`: data in `_datafiles/world/dogmud/narration/<store>/`, `LoadFrom(dir)` for tests, a boot loader that panics on failure | `internal/movenarration/store.go:157-173` |
| 11 | Every player starts with `chrysalis-glow`, so darkness is always answerable | `internal/characters/character.go:367` |

## What the player gets

### Bands

Each notice is about a change in the observer's own **band**, computed through
their own vision window:

| Band | Meaning |
|---|---|
| `dark` | reads nothing |
| `shapes` | reads shapes, not faces |
| `faces` | reads fully |
| `dazzled` | light at or above the observer's shifted dazzle edge (`75 − strength`); reads fully today, the penalty arrives in plan 5 |

### When a notice fires

| Trigger | Announces |
|---|---|
| Moving into a new room | only a change to a **darker** band, or into **dazzled** |
| Each combat round the player is in | any band change, **both directions** |
| Any command the player issues | any band change, **both directions**, delivered before the command's own output |

- **Owner rationale for not checking every round:** an idle player is not
  reading; they learn of a change the next time they act, and the global
  splash (fact 6) already gives everyone background flavour.
- **Silent recording** (no notice, band just stored): login, waking, the end
  of blindness, and the first check after any of them.
- **Sleeping or blinded players get no notices.** Blindness is not light; its
  end is recorded silently so it never reads as dawn breaking.
- **Mobs get no text.**

### What a notice says

**World terms that state the consequence** (owner choice): what changed and
what the player can now make out, never a number or a mechanic's name.

| Cause | Example |
|---|---|
| Sky (dusk, dawn) | "Dusk settles over the street; faces blur into shapes." / "Daylight grows; you can make out faces again." |
| Carried light arriving or leaving | "The lantern-light leaves with its bearer, and the room sinks to shapes." |
| Room light (lamp term) | "The lamps gutter low; faces blur." Unreachable today (no lamp changes at runtime); authored now so plan 5's scheduled lamps need no store change |
| Weather | "The storm dims the day; faces blur." |
| Movement | "You step out of the lamplight." / "You step into darkness." |
| Dazzle | "The lamplight stabs at your widened eyes." |

Outdoor and indoor variants where the wording needs them. All text follows
`dogmud-player-copy` (80 columns, no numbers, no dashes) and is authored as
data, not Go.

**The global sunrise and sunset splash stays** as the world's clock. Band
crossings happen when light actually passes a threshold, not at the Night
flag's flip, so the two do not stack.

## Architecture

### `internal/lightnotice` (new package)

```go
type Trigger uint8
const (TriggerMove Trigger = iota; TriggerCombatRound; TriggerCommand; TriggerQuiet)

// Check compares the player's current band with the last one announced to
// them and, if the trigger's rule allows, sends one notice.
func Check(user *users.UserRecord, trigger Trigger)
```

- Holds per-player state in a mutex-guarded map, **in memory only**: the last
  announced band, the room it was recorded in, and that room's light terms at
  the time. Cleared on logout.
- **Cause attribution** compares the stored terms with the current ones: a
  room change is movement; otherwise the first term that moved in the order
  carried light, lamp, weather occlusion, sky names the cause. Nothing moved
  (for example, a strength change from a potion) falls back to a generic line.
- Imports `messaging`, `rooms`, `users`, `narration`, `configs`; never
  `usercommands` or `hooks`, which call it. Same pattern as
  `internal/companionai`.
- Ships a `context.md`.

### Two additive changes to existing packages

1. **`messaging.LightBand(observer, room RoomVisibility) Band`** returning
   `BandDark | BandShapes | BandFaces | BandDazzled`, built from the same inputs
   as `ParticipantSight` plus the dazzle edge `windowDazzleEdge − strength`.
   **`ParticipantSight` and `SightThroughWindow` are unchanged**, so no
   sight consumer moves, including the AI companion's `perception.go`.
   `windowDazzleEdge` stays a constant: plan 1's rule makes it a knob in the
   plan that gives dazzle a penalty.
2. **`(*Room).LightTerms()`** returning the four terms `lightLevelWithMutatorBridge`
   already computes (sky after occlusion, occlusion steps, lamp, carried
   light present), refactored so `LightLevel` and `LightTerms` share one
   computation rather than duplicating it.

### Call sites

| Seam | Trigger |
|---|---|
| Top of `usercommands.TryCommand`, for a real user | `TriggerCommand` |
| The per-player step of `NewRound_DoCombat` | `TriggerCombatRound` |
| A `RoomChange` listener, for a player mover, after the move | `TriggerMove` |
| `PlayerSpawn`, and waking | `TriggerQuiet` |

A move command passes `TryCommand` first (recording, in the OLD room, any
standing change since the last command) and then fires `RoomChange`, so the
two checks never announce the same crossing twice.

### The store

`_datafiles/world/dogmud/narration/light-notices/`, loaded like
`internal/movenarration` (`LoadFrom(dir)` for tests; boot panics on a bad
file). Keyed by cause × direction (`darker`, `lighter`, `dazzled`) × setting
(`outdoor`, `indoor`), with a minimum number of variants per key and the
80-column rule validated at load. Added to the narration snapshot harness
(`internal/narration/snapshot_test.go`) as a new golden.

### Category

A new **`CategoryLight`**, treated exactly as `CategoryTimeOfDay` (fact 9):
prose-wrapped, name-normalisation skipped, not verbosity-suppressible.
Movement notices are not time of day, and M6/M7 and client filters will want
to address light on its own.

## Verification

Each test is shown able to fail before it is trusted.

- **`LightBand` table:** every band edge for strength 0, a shifted window at
  24 (dazzle at 51), infrared reach, and a Blinded observer.
- **Trigger rules:** move to darker notifies; move to lighter does not; move
  into dazzled notifies; combat and command triggers notify in both
  directions.
- **Once only:** after a notice, a second check in the same band is silent;
  `TriggerQuiet` never speaks; sleeping and blinded players never receive one.
- **Cause attribution:** sky versus carried light versus lamp versus weather,
  on a constructed room.
- **Integration:** a player standing in a `city_backstreet` as the clock
  crosses dusk (faces to shapes; a thoroughfare never leaves `faces`) issues
  `look` and receives exactly one notice, before the room description; a
  second `look` receives none. A nightvision-24 player walking into a thoroughfare
  receives the dazzle notice.
- **Store:** load validation, and a golden in the snapshot harness.
- **Goldens:** no lighting golden should move (no light value changes). If
  one does, that is a defect.
- The full suite, lint, and a boot check. If the AI companion module is
  merged (it is), boot once with it enabled and confirm a companion's
  perception is unaffected: this plan changes no sight API.

## Out of scope

- Dazzle penalties, scheduled and pulsing light sources: plan 5.
- Retuning any light value, and the infrared-reach oddity recorded on
  2026-09-25: plan 6.
- Notices for mobs, and for players watching OTHER players' light change
  (for example "Bram's lantern goes out"): not requested.
