# Grapple Messaging Context

## Purpose

`internal/grapplemessaging` holds the prose library for the grappling system —
the per-position, per-transition lines describing what two entangled fighters
are doing — and picks a line without repeating itself.

Grappling has many states and many transitions between them, so the message
count is large enough that (a) the pool must be validated for completeness at
load, and (b) selection must avoid immediate repeats or the fight reads like a
stuck record.

## Files

- **loader.go** — `Library`, `TemplateTriad`, `GradientTriad`, `Load`,
  `LoadFromDataFiles`, `DataFilesPath`, `ValidateCompleteness`.
- **render.go** — `PickTemplate`, `PickIndex`, `RenderTriad`, `RenderGradient`.

## Types

```go
type TemplateTriad struct  { /* the three viewpoints of one event */ }
type GradientTriad struct  { /* intensity-graded variants */ }
type Library struct        { /* all pools, keyed by position/transition */ }
```

A **triad** is the same event told three ways — to the controller, to the
controlled, and to the room. Storing them together is what stops the three
drifting apart when someone edits one.

## API

```go
func Load(path string) (*Library, error)
func LoadFromDataFiles() (*Library, error)
func DataFilesPath() string
func ValidateCompleteness(lib *Library) []error

func PickTemplate(pool []string, cooldowns map[string]bool, keyPrefix string, picker ...narration.Picker) string
func PickIndex(n int, cooldowns map[string]bool, keyPrefix string, picker ...narration.Picker) int
func RenderTriad(tri TemplateTriad, controllerName, controlledName string, cooldowns map[string]bool, keyPrefix string, picker ...narration.Picker) RenderedTriad
func RenderGradient(tri GradientTriad, selfName, partnerName string, cooldowns map[string]bool, keyPrefix string, picker ...narration.Picker) RenderedGradient
```

`LoadFromDataFiles` and `DataFilesPath` resolve the store under the CONFIGURED
world (`FilePaths.DataFiles`), which M4b-1 introduced to replace a hardcoded
`_datafiles/world/dogmud/...` literal in `internal/hooks`. `Load` stays exported
for tests that supply their own file. This store is EVENT-tier narration:
`main.go` calls `hooks.LoadGrappleMessaging` at boot, which loads, runs
`ValidateCompleteness`, and panics on either failing. See the two-tier loader
policy in `internal/narration/context.md`.

`RenderTriad` and `RenderGradient` are the coordinated renderers: one `PickIndex`
draw serves all three roles, and each substitutes names through
`narration.Substitute` keyed on the core's canonical `narration.TokenActor` /
`narration.TokenActee` (M4a). The store's own single-line renderer,
`RenderTemplate`, was the second of the messaging arc's three leftover token
engines and is gone; its one caller (`internal/hooks`, the mount-strike flavor
line) now substitutes through `narration.Substitute` directly, and
`grapple_outcomes.yaml` spells its two name tokens `{actor}`/`{actee}` rather
than `{controllerName}`/`{controlledName}`.

## Gotchas

- **`ValidateCompleteness` returns a slice of errors, not one.** Run it at load
  and report all of them; a partially-covered library produces silent blanks
  mid-fight, which is far harder to diagnose than a startup complaint.
- **`PickTemplate` needs the caller to own the cooldown map.** It reads and
  marks; it does not age entries. A caller that never clears the map eventually
  runs out of eligible lines.
- **`keyPrefix` namespaces the cooldowns.** Two pools sharing a prefix will
  suppress each other's lines.
- **Both names are substituted positionally** — `RenderTriad`/`RenderGradient`
  do not know which is the player. Getting controller and controlled the wrong
  way round produces text that is grammatical and completely wrong.
- **Maintenance-shortage text is participant-private.** The grapple tick uses
  its existing `sendToCharacter` grapple-flow route once for each short player.
  It is not a template triad: partners and observers must not receive it, and
  the route deliberately no-ops for NPCs. Reversal and submission messaging
  must not replay the maintenance warning.
- **The message follows independent admission.** Controller and controlled
  participants receive separate role-adjusted maintenance quotes before drift.
  Either may be short and lose only their own Unarmed Combat term; do not infer
  one participant's status from the other or from the eventual contest result.

## Dependencies

`configs`, `mudlog`, plus YAML.

## Consumers

`internal/combat` and `internal/hooks` on the grapple resolution path.
