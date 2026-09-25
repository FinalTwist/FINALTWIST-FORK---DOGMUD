# Lighting plan 3d: transition notices, implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Tell a player, in world terms, when the light they can see by crosses a band (dark, shapes, faces, dazzled), checked on moves, combat rounds and commands.

**Architecture:** A new `internal/lightnotice` package holds per-player in-memory state, a pure `decide` function (trigger rules and cause attribution), and a YAML store of lines under `_datafiles/world/dogmud/narration/light-notices/`. Two additive changes feed it: `messaging.LightBand` / `BandThroughWindow` and `(*rooms.Room).LightTerms()`. `ParticipantSight` and `SightThroughWindow` do not change, so no sight consumer (the AI companion's perception included) moves. Call sites are `TryCommand`, `handlePlayerCombat`, and four listeners in one hooks file.

**Tech Stack:** Go, YAML via `internal/fileloader`, `internal/narration` picker core, `internal/events` listeners.

**Spec:** `docs/superpowers/specs/completed/2026-09-25-lighting-plan3d-transition-notices-design.md` (owner approved 2026-09-25). Owner rulings live there; do not re-ask them.

---

## Facts verified against source

Read on branch `feature/lighting-plan3d-transition-notices` (`04a52626b`, master `3887ed1dc` plus the spec) on 2026-09-25.

| # | Fact | Source |
|---|---|---|
| 1 | `ParticipantSight` reads `configs.GetBalanceConfig()` `LightBlindBelow`/`LightDimBelow`; `configs.GetLightingConfig()` exposes the same two as `BlindBelow`/`DimBelow` | `internal/messaging/predicates.go:56-89`, `internal/configs/config.lighting_accessor.go:29-52` |
| 2 | `windowDazzleEdge = 75`, `windowShiftCap = 24`, `windowFloor = 1`; `SightThroughWindow` never reads the dazzle edge | `internal/messaging/window.go:16-68` |
| 3 | Shipped `config.yaml` carries no `LightBlindBelow`/`LightDimBelow`, so the Go defaults 25/50 are live | `grep` of `_datafiles/config.yaml` (empty), `config.balance.go:1099-1100` |
| 4 | Room light is composed in `lightLevelWithMutatorBridge(cfg, celestial, lightMod, occlusionSteps)`: sky (`Attenuate(step, celestial, skyFraction * 2^-occlusion)`), lamp (`lampValue()`), positive LightMod bridge, carried light (`FindHasLight` on mobs and players) | `internal/rooms/lighting.go:53-111` |
| 5 | Negative LightMod (occlusion) comes only from `weather_blizzard`, `weather_dust`, `weather_storm` mutators (`lightmod: -1`) | `_datafiles/world/dogmud/mutators/weather_*.yaml:15` |
| 6 | Tests call `lightLevelWithMutatorBridge` directly; its signature must survive | `internal/rooms/lighting_model_test.go:64-84` |
| 7 | Indoor is `(*BiomeInfo).Indoor`, read via `room.GetBiome()` (nil-safe needed) | `internal/rooms/biomes.go:66`, `rooms.go:2918` |
| 8 | `CategoryTimeOfDay` appears in `shouldWrap` (`pipeline.go:138`), `skipStages` (`normalize.go:28`), `String()` (`messaging.go:221`), `wrapAllowlist` (`pipeline_test.go:108`); the allowlist count is pinned at 44 (`pipeline_test.go:136`); alias `time-of-day: 179` in `_datafiles/world/dogmud/ansi-aliases.yaml:267` | those lines |
| 9 | `TestCategoryStringRoundTrip` walks `CategoryDefault..categoryMax` and needs a unique `String()` per value | `internal/messaging/messaging_test.go:11-23` |
| 10 | `narration.Render(Variants, tokens, pick)` picks one index; `ValidateVariants(v, min, expected...)`; `narration.RoleActor` exists | `internal/narration/render.go:92,213,172-176` |
| 11 | `fileloader.LoadAllFlatFiles[K, T Loadable[K]](dir)`; `Loadable` needs `Id() K`, `Validate() error`, `Filepath() string` | `internal/fileloader/fileloader.go:31-39,165` |
| 12 | Boot loads narration stores at `main.go:1946-1947` (`combat.LoadTauntMessageFiles()`, `movenarration.LoadMoveNarrationFiles()`) | `main.go` |
| 13 | `TryCommand(cmd, rest string, userId int, flags events.EventFlag)`; user nil check is its first statement; its only caller is `world.go:1035` | `internal/usercommands/usercommands.go:316-322` |
| 14 | `handlePlayerCombat` loops online users; the `ValidateAggro` block ends at line 158, then `user.Character.CancelCombatConditions()` at 160 | `internal/hooks/NewRound_DoCombat.go:111-160` |
| 15 | `RoomChange{UserId, MobInstanceId, FromRoomId, ToRoomId, Unseen}` is QUEUED (`events.AddToQueue`) after `user.Character.RoomId` is set | `internal/rooms/roommanager.go:460-472`, `internal/events/eventtypes.go:180-188` |
| 16 | Listener registration lives in `internal/hooks/hooks.go` (RoomChange 35-40, NewRound 43-67, PlayerSpawn 84, PlayerDespawn 86-89 with `HandleLeave` registered `events.Last`) | `internal/hooks/hooks.go` |
| 17 | **There is no single waking seam and no condition-removed event.** Sleep ends at 11 hand-rolled `CancelConditionsWithFlag` sites or by expiry; `events.Condition` is apply-only | `internal/characters/conditions.go:34,178`, `internal/events/eventtypes.go:20-42` |
| 18 | `user.SendText(cat, txt)` renders and queues an `events.Message{UserId, Text}` | `internal/users/userrecord.go:486-499` |
| 19 | Test seams: `users.NewTestUser`, `users.SeedUsersForTest`, `rooms.SeedRoomsForTest`, `rooms.SeedBiomesForTest`, `rooms.SkyLightPtr`, `rooms.LampPtr`, `rooms.LoadBiomeDataFiles`, `util.SetRoundCountForTest`, `gametime.ClearDateCacheForTest`, `gametime.ClearCelestialMemoForTest`, `events.RegisterListener`/`UnregisterListener`/`ProcessEvents` | `internal/users/test_helpers.go`, `internal/rooms/test_helpers.go`, `internal/rooms/biomes.go:163`, `internal/util/util.go:147`, `internal/gametime/*.go` |
| 20 | A condition granting nightvision strength N in a test: `SeedConditionsForTest` with `Flags: []conditions.Flag{conditions.NightVision}` and `Effects: {conditions.EffectNightVisionStrength: {Literal: N}}`, then `c.AddCondition(id, true)` | `internal/characters/vision_test.go:100-115` |
| 21 | `city_backstreet` reads shapes at midnight on days 356, 81, 172; `city_thoroughfare` reads at least `DimBelow` at every hour; the clock helper is `SetRoundCountForTest(uint64(float64(doy-1)*900 + hour*37.5))` under `RoundsPerDay 900` | `internal/rooms/city_tier_light_test.go:15-95` |
| 22 | Neither `internal/hooks` nor `internal/usercommands` loads a lightnotice store today; an UNLOADED store must mean silence so the other ~thousands of tests in those packages see no new messages | design rule for this plan |

### One seam the spec named that does not exist

The spec's call-site table lists "waking" as a `TriggerQuiet` seam (fact 17 shows there is none). This plan keeps the spec's BEHAVIOUR (waking and the end of blindness are recorded silently) with a cheaper seam: a `NewRound` listener calls `lightnotice.NoteAttention(user)` for every online player, which only reads two flags and marks a sleeping or blinded player's record `quiet`. The next check after they wake or see again records silently. No light is computed and no text is sent per round, so the owner's "not every round" ruling holds.

---

## File map

| Path | Status | Responsibility |
|---|---|---|
| `internal/messaging/band.go` | create | `Band`, `BandThroughWindow`, `LightBand` |
| `internal/messaging/band_test.go` | create | band edges, parity with `ParticipantSight` |
| `internal/messaging/window.go` | modify | comment only: the dazzle edge is now read by `BandThroughWindow` |
| `internal/messaging/messaging.go` | modify | `CategoryLight` |
| `internal/messaging/pipeline.go`, `normalize.go` | modify | treat `CategoryLight` as `CategoryTimeOfDay` |
| `internal/messaging/pipeline_test.go` | modify | allowlist + count 45 |
| `internal/messaging/category_light_test.go` | create | pins "treated exactly as time of day" |
| `_datafiles/world/dogmud/ansi-aliases.yaml` | modify | `light: 179` |
| `internal/rooms/lighting.go` | modify | `LightTerms`, `composeLight` |
| `internal/rooms/light_terms_test.go` | create | terms tests |
| `internal/lightnotice/store.go` | create | store types, validation, load, lookup |
| `internal/lightnotice/store_test.go` | create | validation and load tests |
| `internal/lightnotice/tracker.go` | create | `Trigger`, `decide`, attribution, state, `Check`, `Forget`, `NoteAttention`, `ResetForTest` |
| `internal/lightnotice/tracker_test.go` | create | pure decide tests |
| `internal/lightnotice/check_test.go` | create | `Check` against seeded users and rooms |
| `internal/lightnotice/integration_test.go` | create | shipped biomes + clock scenarios |
| `internal/lightnotice/main_test.go` | create | `TestMain` logger setup |
| `internal/lightnotice/context.md` | create | package doc |
| `_datafiles/world/dogmud/narration/light-notices/{movement,carried,lamp,weather,sky,eyes}.yaml` | create | the lines |
| `internal/hooks/LightNotice_Triggers.go` | create | RoomChange, PlayerSpawn, PlayerDespawn, NewRound listeners |
| `internal/hooks/LightNotice_Triggers_test.go` | create | listener and combat wiring tests |
| `internal/hooks/hooks.go` | modify | register the four listeners |
| `internal/hooks/NewRound_DoCombat.go` | modify | `TriggerCombatRound` call |
| `internal/usercommands/usercommands.go` | modify | `TriggerCommand` call |
| `internal/usercommands/light_notice_wiring_test.go` | create | notice precedes command output |
| `main.go` | modify | boot loader |
| `internal/narration/snapshot_test.go` | modify | `light_notices` golden |
| `internal/narration/testdata/stores/light_notices.golden` | create | generated |
| `internal/messaging/context.md`, `internal/rooms/context.md`, `internal/narration/context.md` | modify | document new surface |
| `docs/README.md` | modify | index this plan |

---

### Task 1: `messaging.Band`, `BandThroughWindow`, `LightBand`

**Files:**
- Create: `internal/messaging/band.go`
- Create: `internal/messaging/band_test.go`
- Modify: `internal/messaging/window.go` (comment inside `SightThroughWindow`)

- [ ] **Step 1: Write the failing test**

`internal/messaging/band_test.go`:

```go
package messaging

import "testing"

// TestBandThroughWindowEdges pins every band edge at strength 0 and at the
// capped strength 24, plus infrared reach. Edges use the shipped 25/50.
func TestBandThroughWindowEdges(t *testing.T) {
	cases := []struct {
		name     string
		light    int
		strength int
		reach    int
		want     Band
	}{
		{"s0 pitch dark", 0, 0, 0, BandDark},
		{"s0 just below blind", 24, 0, 0, BandDark},
		{"s0 at blind edge", 25, 0, 0, BandShapes},
		{"s0 just below dim", 49, 0, 0, BandShapes},
		{"s0 at dim edge", 50, 0, 0, BandFaces},
		{"s0 just below dazzle", 74, 0, 0, BandFaces},
		{"s0 at dazzle edge", 75, 0, 0, BandDazzled},
		{"s24 below floor", 0, 24, 0, BandDark},
		{"s24 at floor", 1, 24, 0, BandShapes},
		{"s24 at shifted dim", 26, 24, 0, BandFaces},
		{"s24 just below shifted dazzle", 50, 24, 0, BandFaces},
		{"s24 at shifted dazzle", 51, 24, 0, BandDazzled},
		{"s30 caps at 24", 51, 30, 0, BandDazzled},
		{"negative strength reads as 0", 74, -5, 0, BandFaces},
		{"reach reads shapes in the dark", 0, 0, 10, BandShapes},
		{"reach has a limit", -11, 0, 10, BandDark},
	}
	for _, c := range cases {
		if got := BandThroughWindow(c.light, c.strength, c.reach, 25, 50); got != c.want {
			t.Errorf("%s: BandThroughWindow(%d, s=%d, r=%d) = %v, want %v",
				c.name, c.light, c.strength, c.reach, got, c.want)
		}
	}
}

// TestLightBandAgreesWithParticipantSight is the guard that keeps the two
// optics answers from drifting: LightBand only ever SPLITS ParticipantSight's
// full tier into faces and dazzled, never moves an edge.
func TestLightBandAgreesWithParticipantSight(t *testing.T) {
	c := newChar(t)
	for light := -30; light <= 100; light++ {
		room := sightLight(light)
		band := LightBand(c, room)
		sight := ParticipantSight(c, room)
		var want SightDecision
		switch band {
		case BandDark:
			want = SightNone
		case BandShapes:
			want = SightShapes
		default:
			want = SightFull
		}
		if sight != want {
			t.Errorf("light %d: LightBand %v but ParticipantSight %v", light, band, sight)
		}
	}
}

func TestLightBandBlindedIsDark(t *testing.T) {
	c := newChar(t)
	setBlinded(t, c)
	if got := LightBand(c, sightLight(100)); got != BandDark {
		t.Fatalf("a Blinded observer in a blazing room = %v, want dark", got)
	}
}

func TestLightBandNilsReadFaces(t *testing.T) {
	if got := LightBand(nil, sightLight(0)); got != BandFaces {
		t.Errorf("nil observer = %v, want faces", got)
	}
	if got := LightBand(newChar(t), nil); got != BandFaces {
		t.Errorf("nil room = %v, want faces", got)
	}
}

func TestBandString(t *testing.T) {
	for b, want := range map[Band]string{BandDark: "dark", BandShapes: "shapes", BandFaces: "faces", BandDazzled: "dazzled"} {
		if b.String() != want {
			t.Errorf("Band(%d).String() = %q, want %q", b, b.String(), want)
		}
	}
}
```

`newChar`, `setBlinded` live in `predicates_test.go`; `sightLight` in `participant_sight_test.go`. All three are in package `messaging`.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/messaging/ -run 'TestBand|TestLightBand' -count=1`
Expected: build failure, `undefined: Band`.

- [ ] **Step 3: Implement**

`internal/messaging/band.go`:

```go
package messaging

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// Band is what an observer can make out at a room's light, one step finer than
// SightDecision: it splits full sight into reading faces and being dazzled.
//
// The constants run DARKEST TO BRIGHTEST, the opposite of SightDecision's
// best-to-worst order, because the one consumer (internal/lightnotice) asks
// "did it get darker?" and an ordered comparison should read that way.
//
// Dazzled carries no penalty yet: an observer there reads fully. Plan 5 gives
// it teeth. It exists now so a notice can tell a player the light stabs at
// their eyes.
type Band uint8

const (
	BandDark Band = iota
	BandShapes
	BandFaces
	BandDazzled
)

func (b Band) String() string {
	switch b {
	case BandDark:
		return "dark"
	case BandShapes:
		return "shapes"
	case BandFaces:
		return "faces"
	case BandDazzled:
		return "dazzled"
	}
	return "unknown"
}

// BandThroughWindow is SightThroughWindow with the full tier split at the
// observer's shifted dazzle edge (windowDazzleEdge minus strength, strength
// clamped exactly as SightThroughWindow clamps it). It never moves a lower
// edge: dark, shapes and faces-or-dazzled are SightThroughWindow's answers.
func BandThroughWindow(light, strength, reach, blindBelow, dimBelow int) Band {
	switch SightThroughWindow(light, strength, reach, blindBelow, dimBelow) {
	case SightNone:
		return BandDark
	case SightShapes:
		return BandShapes
	}
	if strength < 0 {
		strength = 0
	}
	if strength > windowShiftCap {
		strength = windowShiftCap
	}
	if light >= windowDazzleEdge-strength {
		return BandDazzled
	}
	return BandFaces
}

// LightBand is ParticipantSight's band-grained twin, for a caller that needs
// to know about dazzle. It is optics only, exactly like ParticipantSight: it
// does not consult sleep. A Blinded observer is dark; a nil observer or a nil
// room reads faces, matching ParticipantSight's defensive defaults.
//
// It reads the narrow lighting config rather than the 400-field Balance copy
// ParticipantSight takes; both carry the same two edges.
func LightBand(observer *characters.Character, room RoomVisibility) Band {
	if observer == nil || room == nil {
		return BandFaces
	}
	if observer.Perception != nil && observer.Perception.State() == perception.Blinded {
		return BandDark
	}
	cfg := configs.GetLightingConfig()
	return BandThroughWindow(
		room.LightLevel(),
		observer.NightVisionStrength(),
		observer.InfraReach(),
		cfg.BlindBelow,
		cfg.DimBelow,
	)
}
```

In `internal/messaging/window.go`, replace the comment inside the `if light >= shiftedDim {` branch:

```go
		// Perfect and too-bright both read fully. Dazzle has no mechanical
		// penalty yet; BandThroughWindow (band.go) is the one reader of the
		// upper edge, and only to tell a player the light hurts.
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/messaging/ -count=1`
Expected: `ok`.

- [ ] **Step 5: Prove the parity guard can fail**

Temporarily change `return BandShapes` in `BandThroughWindow` to `return BandFaces`, run `go test ./internal/messaging/ -run TestLightBandAgreesWithParticipantSight -count=1`, confirm FAIL, restore, re-run, confirm PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/messaging/band.go internal/messaging/band_test.go internal/messaging/window.go
git commit -F - <<'EOF'
feat(messaging): LightBand splits full sight into faces and dazzled

Additive. ParticipantSight and SightThroughWindow are unchanged, so no
sight consumer moves; a parity test pins that LightBand only splits the
full tier.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 2: `CategoryLight`

**Files:**
- Modify: `internal/messaging/messaging.go`, `internal/messaging/pipeline.go:138`, `internal/messaging/normalize.go:28`, `internal/messaging/pipeline_test.go:57-137`
- Modify: `_datafiles/world/dogmud/ansi-aliases.yaml:267`
- Create: `internal/messaging/category_light_test.go`

- [ ] **Step 1: Write the failing test**

`internal/messaging/category_light_test.go`:

```go
package messaging

import "testing"

// TestCategoryLightIsTreatedAsTimeOfDay pins the spec's rule: light notices are
// prose-wrapped, skip every normalisation stage, and no verbosity tier drops
// them.
func TestCategoryLightIsTreatedAsTimeOfDay(t *testing.T) {
	if CategoryLight.String() != "light" {
		t.Fatalf("CategoryLight.String() = %q, want light", CategoryLight.String())
	}
	if shouldWrap(CategoryLight) != shouldWrap(CategoryTimeOfDay) {
		t.Error("CategoryLight must wrap exactly as CategoryTimeOfDay")
	}
	if skipStages(CategoryLight) != skipStages(CategoryTimeOfDay) {
		t.Error("CategoryLight must skip the same normalisation stages as CategoryTimeOfDay")
	}
	for _, v := range []Verbosity{VerbosityFull, VerbosityMedium, VerbosityLight} {
		if v.Suppresses(CategoryLight) {
			t.Errorf("verbosity %v suppresses CategoryLight; no tier may", v)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/messaging/ -run TestCategoryLight -count=1`
Expected: `undefined: CategoryLight`.

- [ ] **Step 3: Implement**

In `messaging.go`, append after `CategoryToxin` (end of the `// Other.` group, before `categoryMax`). Appended, not grouped under Environment, so no existing category value shifts:

```go
	CategoryToxin

	// Environment, appended rather than grouped with CategoryTimeOfDay so no
	// existing Category value shifts. Lighting plan 3d's band-change notices.
	CategoryLight
```

In `String()`, after the `CategoryToxin` case:

```go
	case CategoryLight:
		return "light"
```

In `pipeline.go` `shouldWrap`, change `CategoryWeather, CategoryTimeOfDay,` to `CategoryWeather, CategoryTimeOfDay, CategoryLight,`.

In `normalize.go` `skipStages`, change `CategoryWeather, CategoryTimeOfDay, CategorySplash,` to `CategoryWeather, CategoryTimeOfDay, CategoryLight, CategorySplash,`.

In `pipeline_test.go` `wrapAllowlist`, after `CategoryTimeOfDay:    true,` add `CategoryLight:        true,` (gofmt aligns). Change `if admitted != 44 {` to `if admitted != 45 {` and its message `"expected exactly 44 categories` to `"expected exactly 45 categories`.

In `_datafiles/world/dogmud/ansi-aliases.yaml`, after `  time-of-day: 179` add:

```yaml
  light: 179
```

- [ ] **Step 4: Run to verify**

Run: `go test ./internal/messaging/ -count=1`
Expected: `ok` (includes `TestCategoryStringRoundTrip` and `TestShouldWrapMatchesPinnedAllowlist`).

- [ ] **Step 5: Commit**

```bash
git add internal/messaging/messaging.go internal/messaging/pipeline.go internal/messaging/normalize.go internal/messaging/pipeline_test.go internal/messaging/category_light_test.go _datafiles/world/dogmud/ansi-aliases.yaml
git commit -F - <<'EOF'
feat(messaging): CategoryLight, treated exactly as time of day

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 3: `(*Room).LightTerms()`

**Files:**
- Modify: `internal/rooms/lighting.go:47-111`
- Create: `internal/rooms/light_terms_test.go`

- [ ] **Step 1: Write the failing test**

`internal/rooms/light_terms_test.go`:

```go
package rooms

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/lightscale"
)

// Level must be exactly what LightLevel reports: the two share one computation.
func TestLightTermsLevelMatchesLightLevel(t *testing.T) {
	cfg := modelCfg()
	zero, half, open := 0.0, 0.5, 1.0
	lamp36, lamp55 := 36, 55
	for _, r := range []Room{
		{SkyLight: &zero},
		{SkyLight: &half},
		{SkyLight: &open, Lamp: &lamp36},
		{SkyLight: &open, Lamp: &lamp55},
		{SkyLight: &zero, Lamp: &lamp55},
	} {
		for _, celestial := range []float64{lightscale.Absent(), 0, 36, 60, 70} {
			for _, occ := range []int{0, 1, 2} {
				for _, mod := range []int{0, 1, 2} {
					want := r.lightLevelWithMutatorBridge(cfg, celestial, mod, occ)
					got := r.composeLight(cfg, celestial, mod, occ).Level
					if got != want {
						t.Errorf("sky=%v lamp=%v celestial=%v occ=%d mod=%d: terms.Level %d, LightLevel %d",
							r.SkyLight, r.Lamp, celestial, occ, mod, got, want)
					}
				}
			}
		}
	}
}

func TestLightTermsReportEachTerm(t *testing.T) {
	cfg := modelCfg()
	open, zero := 1.0, 0.0
	lamp := 40

	lit := Room{SkyLight: &open, Lamp: &lamp}
	got := lit.composeLight(cfg, 60, 1, 2)
	if !got.HasLamp || got.Lamp != 40 {
		t.Errorf("lamp = (%v, %d), want (true, 40)", got.HasLamp, got.Lamp)
	}
	if got.OcclusionSteps != 2 {
		t.Errorf("OcclusionSteps = %d, want 2", got.OcclusionSteps)
	}
	if got.LightMod != 1 {
		t.Errorf("LightMod = %d, want 1", got.LightMod)
	}
	wantSky := lightscale.Attenuate(cfg.DoublingStep, 60, 0.25)
	if math.Abs(got.Sky-wantSky) > 1e-9 {
		t.Errorf("Sky = %v, want %v (sky after two occlusion steps)", got.Sky, wantSky)
	}
	if got.Carried {
		t.Error("Carried = true with nobody in the room")
	}

	cave := Room{SkyLight: &zero}
	if c := cave.composeLight(cfg, 60, 0, 0); !math.IsInf(c.Sky, -1) || c.HasLamp {
		t.Errorf("a cave has no sky term and no lamp, got Sky=%v HasLamp=%v", c.Sky, c.HasLamp)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/rooms/ -run TestLightTerms -count=1`
Expected: `r.composeLight undefined`.

- [ ] **Step 3: Implement**

In `internal/rooms/lighting.go`, add after `IsLit` (before `lightLevel`):

```go
// LightTerms is the room's light broken into the terms LightLevel combines,
// for a caller that needs to know WHY the light is what it is.
// internal/lightnotice names the cause of a band change from them.
type LightTerms struct {
	// Level is exactly LightLevel(): both come from composeLight.
	Level int
	// Sky is the sky term after the sky fraction and weather occlusion, in
	// light-scale units; lightscale.Absent() when the room has no sky.
	Sky float64
	// OcclusionSteps is the doubling steps of sky removed by weather mutators.
	OcclusionSteps int
	// Lamp is the room's own lamp; 0 when HasLamp is false.
	Lamp    int
	HasLamp bool
	// LightMod is the positive LightMod bridge total.
	LightMod int
	// Carried reports that someone in the room carries a light.
	Carried bool
}

// LightTerms reports the terms behind LightLevel, from the same single
// computation.
func (r *Room) LightTerms() LightTerms {
	lightMod, occlusionSteps := r.mutatorLightTerms()
	return r.composeLight(configs.GetLightingConfig(), gametime.CelestialLight(), lightMod, occlusionSteps)
}
```

Replace the body of `lightLevelWithMutatorBridge` so it delegates, and move the old body into `composeLight`, recording each term. The full replacement for everything from the `lightLevelWithMutatorBridge` doc comment through its closing brace:

```go
// lightLevelWithMutatorBridge is the composition itself, with the mutator
// contribution already summarised, so tests can drive it directly.
//
// lightMod is the total POSITIVE LightMod across active mutators, and
// occlusionSteps the total NEGATIVE, expressed as doubling steps of sky removed.
func (r *Room) lightLevelWithMutatorBridge(cfg configs.Lighting, celestial float64, lightMod, occlusionSteps int) int {
	return r.composeLight(cfg, celestial, lightMod, occlusionSteps).Level
}

// composeLight is the one computation behind LightLevel and LightTerms.
func (r *Room) composeLight(cfg configs.Lighting, celestial float64, lightMod, occlusionSteps int) LightTerms {
	step := cfg.DoublingStep
	if !(step > 0) {
		step = 1
	}

	out := LightTerms{OcclusionSteps: occlusionSteps}
	terms := make([]float64, 0, 4)

	// 1. The sky, attenuated by this room's fraction and then by any weather
	// blocking it. Attenuate returns Absent for a fraction of zero, so a cave
	// contributes no term rather than a term of zero.
	sky := r.skyLightFraction()
	if occlusionSteps > 0 {
		sky *= math.Exp2(-float64(occlusionSteps))
	}
	out.Sky = lightscale.Attenuate(step, celestial, sky)
	terms = append(terms, out.Sky)

	// 2. The room's own lamp.
	if lamp, ok := r.lampValue(); ok {
		out.Lamp, out.HasLamp = lamp, true
		terms = append(terms, float64(lamp))
	}

	// 3. The positive LightMod bridge. A +1 mutator lands exactly on
	// LightDimBelow and +2 one step above it, so the 31 Crash Site Interior
	// rooms and 12 Foldweave rooms that a static `lightmod: 2` holds lit today
	// stay fully visible. Plan 4 replaces this with an authored lamp value.
	if lightMod > 0 {
		out.LightMod = lightMod
		terms = append(terms, float64(cfg.DimBelow)+float64(lightMod-1)*step)
	}

	// 4. Anyone carrying a light. Plan 5 gives carried sources real magnitudes
	// that scale from stat and skill; until then any light source lifts the
	// room to the bottom of the perfect band, which is what the old model's
	// "someone has light, cancel the darkness" rule effectively did.
	if len(r.GetMobs(FindHasLight)) > 0 || len(r.GetPlayers(FindHasLight)) > 0 {
		out.Carried = true
		terms = append(terms, float64(cfg.DimBelow))
	}

	v := lightscale.Combine(step, terms...)
	if math.IsInf(v, -1) {
		// No light of any kind. Zero is the darkest light that NATURALLY
		// occurs, which is what an unlit cave is. Magical darkness goes below
		// this and arrives in plan 5.
		v = 0
	}

	n := int(math.Round(v))
	if n < -100 {
		n = -100
	} else if n > 100 {
		n = 100
	}
	out.Level = n
	return out
}
```

- [ ] **Step 4: Run to verify**

Run: `go test ./internal/rooms/ -count=1`
Expected: `ok`. Every existing lighting test must still pass unchanged; this is a pure refactor.

- [ ] **Step 5: Commit**

```bash
git add internal/rooms/lighting.go internal/rooms/light_terms_test.go
git commit -F - <<'EOF'
feat(rooms): LightTerms reports the terms behind LightLevel

LightLevel and LightTerms share composeLight, one computation, so the
two cannot disagree.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 4: The light-notice store (code)

**Files:**
- Create: `internal/lightnotice/store.go`
- Create: `internal/lightnotice/store_test.go`
- Create: `internal/lightnotice/main_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/lightnotice/main_test.go`:

```go
package lightnotice

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// shippedDir is the store as a booted dogmud world reads it. A test binary
// never reads config.yaml, so tests load it explicitly.
const shippedDir = "../../_datafiles/world/dogmud/narration/light-notices"

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}
```

`internal/lightnotice/store_test.go`:

```go
package lightnotice

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func anyPools(lines ...string) *Pools { return &Pools{Any: lines} }

func validGroup(c Cause) *CauseGroup {
	g := &CauseGroup{Cause: c, Transitions: map[Transition]*Pools{}}
	for _, tr := range Transitions() {
		g.Transitions[tr] = anyPools("The light changes around you.", "Your view of things changes.")
	}
	return g
}

func TestValidateAcceptsACompleteGroup(t *testing.T) {
	if err := validGroup(CauseSky).Validate(); err != nil {
		t.Fatalf("valid group refused: %v", err)
	}
}

func TestValidateRefuses(t *testing.T) {
	cases := map[string]func(g *CauseGroup){
		"unknown cause":        func(g *CauseGroup) { g.Cause = "moonbeam" },
		"missing transition":   func(g *CauseGroup) { delete(g.Transitions, DarkerDark) },
		"unknown transition":   func(g *CauseGroup) { g.Transitions["sideways"] = anyPools("a", "b") },
		"one variant":          func(g *CauseGroup) { g.Transitions[DarkerDark] = anyPools("Only one line here.") },
		"blank line":           func(g *CauseGroup) { g.Transitions[DarkerDark] = anyPools("Fine.", "  ") },
		"over eighty columns":  func(g *CauseGroup) { g.Transitions[DarkerDark] = anyPools("Fine.", strings.Repeat("a", 81)) },
		"a number":             func(g *CauseGroup) { g.Transitions[DarkerDark] = anyPools("Fine.", "It is 3 times darker.") },
		"an em dash":           func(g *CauseGroup) { g.Transitions[DarkerDark] = anyPools("Fine.", "Dark \u2014 very.") },
		"an en dash":           func(g *CauseGroup) { g.Transitions[DarkerDark] = anyPools("Fine.", "Dark \u2013 very.") },
		"a token":              func(g *CauseGroup) { g.Transitions[DarkerDark] = anyPools("Fine.", "{actor} dims.") },
		"any plus split":       func(g *CauseGroup) { g.Transitions[DarkerDark].Outdoor = []string{"a.", "b."} },
		"split missing indoor": func(g *CauseGroup) { g.Transitions[DarkerDark] = &Pools{Outdoor: []string{"a.", "b."}} },
		"empty pools":          func(g *CauseGroup) { g.Transitions[DarkerDark] = &Pools{} },
	}
	for name, mutate := range cases {
		g := validGroup(CauseSky)
		mutate(g)
		if err := g.Validate(); err == nil {
			t.Errorf("%s: Validate accepted it", name)
		}
	}
}

func writeStore(t *testing.T, causes []Cause) string {
	t.Helper()
	dir := t.TempDir()
	for _, c := range causes {
		var b strings.Builder
		fmt.Fprintf(&b, "cause: %s\ntransitions:\n", c)
		for _, tr := range Transitions() {
			fmt.Fprintf(&b, "  %s:\n    any:\n      - 'The light changes around you.'\n      - 'Your view of things changes.'\n", tr)
		}
		if err := os.WriteFile(filepath.Join(dir, string(c)+".yaml"), []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestLoadFromNeedsEveryCause(t *testing.T) {
	t.Cleanup(ResetForTest)
	if err := LoadFrom(writeStore(t, Causes())); err != nil {
		t.Fatalf("complete store refused: %v", err)
	}
	if err := LoadFrom(writeStore(t, Causes()[1:])); err == nil {
		t.Fatal("a store missing a cause file loaded")
	}
}

func TestPoolPicksSetting(t *testing.T) {
	t.Cleanup(ResetForTest)
	g := validGroup(CauseSky)
	g.Transitions[DarkerShapes] = &Pools{Outdoor: []string{"Out one.", "Out two."}, Indoor: []string{"In one.", "In two."}}
	setStoreForTest(map[string]*CauseGroup{string(CauseSky): g})

	if got := Pool(CauseSky, DarkerShapes, false); len(got) != 2 || got[0] != "Out one." {
		t.Errorf("outdoor pool = %v", got)
	}
	if got := Pool(CauseSky, DarkerShapes, true); len(got) != 2 || got[0] != "In one." {
		t.Errorf("indoor pool = %v", got)
	}
	if got := Pool(CauseSky, DarkerDark, true); len(got) != 2 {
		t.Errorf("an any pool serves both settings, got %v", got)
	}
	if got := Pool(CauseLamp, DarkerDark, true); got != nil {
		t.Errorf("absent cause = %v, want nil", got)
	}
}

func TestUnloadedStoreIsSilent(t *testing.T) {
	ResetForTest()
	if _, ok := line(CauseSky, DarkerDark, false, nil); ok {
		t.Fatal("an unloaded store produced a line")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/lightnotice/ -count=1`
Expected: build failure (package has no non-test files).

- [ ] **Step 3: Implement**

`internal/lightnotice/store.go`:

```go
// Package lightnotice tells a player, in world terms, when the light they can
// see by crosses a band: dark, shapes, faces or dazzled.
//
// Go decides WHETHER a notice fires and names its cause and transition. Go
// holds no wording: every line lives in
// _datafiles/world/dogmud/narration/light-notices/<cause>.yaml.
package lightnotice

import (
	"strings"
	"unicode/utf8"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/narration"
	"github.com/pkg/errors"
)

// Cause names what changed the light, as the player is told it.
type Cause string

const (
	CauseMovement Cause = "movement" // the player is in a different room
	CauseCarried  Cause = "carried"  // a carried light arrived or left
	CauseLamp     Cause = "lamp"     // the room's own light (lamp or LightMod bridge)
	CauseWeather  Cause = "weather"  // weather occlusion of the sky
	CauseSky      Cause = "sky"      // the sky itself: dusk, dawn, moons
	CauseEyes     Cause = "eyes"     // no light term explains it; the observer's sight changed
)

// Transition names the band change a line narrates.
type Transition string

const (
	DarkerFaces   Transition = "darker_faces"   // dazzled down to faces
	DarkerShapes  Transition = "darker_shapes"  // down to shapes
	DarkerDark    Transition = "darker_dark"    // down to nothing
	LighterShapes Transition = "lighter_shapes" // dark up to shapes
	LighterFaces  Transition = "lighter_faces"  // up to faces
	IntoDazzle    Transition = "dazzled"        // up into dazzle
)

var allCauses = []Cause{CauseMovement, CauseCarried, CauseLamp, CauseWeather, CauseSky, CauseEyes}

var allTransitions = []Transition{DarkerFaces, DarkerShapes, DarkerDark, LighterShapes, LighterFaces, IntoDazzle}

// Causes and Transitions return copies of the full vocabularies, in a fixed
// order, for tests and the narration snapshot harness.
func Causes() []Cause           { return append([]Cause(nil), allCauses...) }
func Transitions() []Transition { return append([]Transition(nil), allTransitions...) }

// MinVariants is the fewest lines any one pool may hold, so a player who sees
// the same crossing twice need not read the same sentence.
const MinVariants = 2

// maxLineColumns is dogmud-player-copy's hard wrap. A notice is one line.
const maxLineColumns = 80

// Pools is one transition's lines: either one pool for any setting, or an
// outdoor and an indoor pool when the wording needs the difference.
type Pools struct {
	Any     []string `yaml:"any,omitempty"`
	Outdoor []string `yaml:"outdoor,omitempty"`
	Indoor  []string `yaml:"indoor,omitempty"`
}

// CauseGroup is one cause's file.
type CauseGroup struct {
	Cause       Cause                  `yaml:"cause"`
	Transitions map[Transition]*Pools `yaml:"transitions"`
}

func (g *CauseGroup) Id() string       { return string(g.Cause) }
func (g *CauseGroup) Filepath() string { return string(g.Cause) + ".yaml" }

// Validate fails the boot on any malformed file. Every cause must author every
// transition, because the trigger rules can reach all of them: a missing pool
// would be silence in play rather than a boot failure.
func (g *CauseGroup) Validate() error {
	known := false
	for _, c := range allCauses {
		if g.Cause == c {
			known = true
		}
	}
	if !known {
		return errors.Errorf("unknown cause %q", g.Cause)
	}
	for tr := range g.Transitions {
		if !knownTransition(tr) {
			return errors.Errorf("cause %q declares unknown transition %q", g.Cause, tr)
		}
	}
	for _, tr := range allTransitions {
		p := g.Transitions[tr]
		if p == nil {
			return errors.Errorf("cause %q is missing transition %q", g.Cause, tr)
		}
		if err := p.validate(); err != nil {
			return errors.Wrapf(err, "cause %q transition %q", g.Cause, tr)
		}
	}
	return nil
}

func knownTransition(tr Transition) bool {
	for _, t := range allTransitions {
		if t == tr {
			return true
		}
	}
	return false
}

func (p *Pools) validate() error {
	split := len(p.Outdoor) > 0 || len(p.Indoor) > 0
	switch {
	case len(p.Any) > 0 && split:
		return errors.New("declares both any and outdoor/indoor; author one or the other")
	case split && (len(p.Outdoor) == 0 || len(p.Indoor) == 0):
		return errors.New("an outdoor/indoor split must author both settings")
	case !split && len(p.Any) == 0:
		return errors.New("holds no lines")
	}
	named := []struct {
		name string
		pool []string
	}{{"any", p.Any}, {"outdoor", p.Outdoor}, {"indoor", p.Indoor}}
	for _, n := range named {
		if len(n.pool) == 0 {
			continue
		}
		if err := narration.ValidateVariants(narration.Variants{Actor: n.pool}, MinVariants, narration.RoleActor); err != nil {
			return errors.Wrap(err, n.name)
		}
		for i, text := range n.pool {
			if err := validateLine(text); err != nil {
				return errors.Wrapf(err, "%s variant %d", n.name, i)
			}
		}
	}
	return nil
}

// validateLine enforces dogmud-player-copy on a line this store alone owns: one
// line of at most 80 columns, no numbers, no dashes, and no {tokens} (this
// store fills none, so a token would render its own braces).
func validateLine(text string) error {
	if n := utf8.RuneCountInString(text); n > maxLineColumns {
		return errors.Errorf("is %d columns, over %d", n, maxLineColumns)
	}
	if strings.ContainsAny(text, "0123456789") {
		return errors.New("shows a number")
	}
	if strings.ContainsAny(text, "\u2014\u2013") {
		return errors.New("uses an em or en dash")
	}
	if strings.ContainsAny(text, "{}") {
		return errors.New("carries a token; this store fills none")
	}
	return nil
}

// pool returns the lines for a setting.
func (p *Pools) pool(indoor bool) []string {
	if len(p.Any) > 0 {
		return p.Any
	}
	if indoor {
		return p.Indoor
	}
	return p.Outdoor
}

var loaded map[string]*CauseGroup

// LoadFrom loads the store from an explicit directory and returns the error.
// It exists for tests: a test binary never reads config.yaml, so the
// configured path would resolve to _datafiles/world/default.
func LoadFrom(dir string) error {
	got, err := fileloader.LoadAllFlatFiles[string, *CauseGroup](dir)
	if err != nil {
		return errors.Wrap(err, "loading light notices")
	}
	for _, c := range allCauses {
		if got[string(c)] == nil {
			return errors.Errorf("light notices: no file for cause %q", c)
		}
	}
	loaded = got
	return nil
}

// LoadLightNoticeFiles loads the store at boot and panics on any failure,
// matching movenarration.LoadMoveNarrationFiles.
func LoadLightNoticeFiles() {
	dir := string(configs.GetFilePathsConfig().DataFiles) + `/narration/light-notices`
	if err := LoadFrom(dir); err != nil {
		panic(err)
	}
}

// Pool returns the lines one notice draws from, or nil when the store is
// unloaded or the entry is absent.
func Pool(c Cause, tr Transition, indoor bool) []string {
	g := loaded[string(c)]
	if g == nil {
		return nil
	}
	p := g.Transitions[tr]
	if p == nil {
		return nil
	}
	return p.pool(indoor)
}

// line renders one notice through the narration core. ok is false when there
// is nothing to say, which is always the case while the store is unloaded:
// that is what keeps every test package that never loads it silent.
func line(c Cause, tr Transition, indoor bool, pick narration.Picker) (string, bool) {
	pool := Pool(c, tr, indoor)
	if len(pool) == 0 {
		return "", false
	}
	return narration.Render(narration.Variants{Actor: pool}, nil, pick).Actor, true
}

// setStoreForTest installs a store directly. Test-only by name.
func setStoreForTest(m map[string]*CauseGroup) { loaded = m }
```

`ResetForTest` is defined in Task 7's `tracker.go`. For this task, add a temporary definition at the bottom of `store.go` so the package compiles:

```go
// ResetForTest unloads the store. Task 7 moves this into tracker.go and
// widens it to clear per-player records too.
func ResetForTest() { loaded = nil }
```

- [ ] **Step 4: Run to verify**

Run: `go test ./internal/lightnotice/ -count=1`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/lightnotice/store.go internal/lightnotice/store_test.go internal/lightnotice/main_test.go
git commit -F - <<'EOF'
feat(lightnotice): the light-notice store and its load-time rules

Every cause authors every transition, at least two lines per pool, one
line of at most 80 columns, no numbers, dashes or tokens. An unloaded
store is silent.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 5: The shipped lines

**Files:**
- Create: `_datafiles/world/dogmud/narration/light-notices/movement.yaml`, `carried.yaml`, `lamp.yaml`, `weather.yaml`, `sky.yaml`, `eyes.yaml`
- Modify: `internal/lightnotice/store_test.go` (add one test)

Every line obeys `dogmud-player-copy` (read the skill first). Lines use single-quoted YAML scalars and carry no apostrophes. "Widened eyes" is deliberate in dazzle lines: dazzle is reachable only with night sight (spec fact 3).

- [ ] **Step 1: Write the failing test**

Append to `store_test.go`:

```go
// TestShippedStoreLoads is the boot check for the real files.
func TestShippedStoreLoads(t *testing.T) {
	t.Cleanup(ResetForTest)
	if err := LoadFrom(shippedDir); err != nil {
		t.Fatalf("shipped light notices refused: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/lightnotice/ -run TestShippedStoreLoads -count=1`
Expected: FAIL (directory missing).

- [ ] **Step 3: Author the files**

`movement.yaml`:

```yaml
cause: movement
transitions:
  darker_faces:
    any:
      - 'You step out of the glare, and your eyes ease.'
      - 'The glare falls behind you; faces are easy to read again.'
  darker_shapes:
    any:
      - 'You step into dimmer light; faces blur into shapes.'
      - 'The light thins here; you can make out shapes, not faces.'
  darker_dark:
    any:
      - 'You step into darkness.'
      - 'The light falls away behind you, and you can see nothing.'
  lighter_shapes:
    any:
      - 'You step into faint light; shapes come out of the dark.'
      - 'There is a little light here; you can make out shapes.'
  lighter_faces:
    any:
      - 'You step into the light; faces are clear again.'
      - 'The light is better here; you can make out faces.'
  dazzled:
    any:
      - 'You step into light so bright it stabs at your widened eyes.'
      - 'The glare here is too much for your widened eyes.'
```

`carried.yaml`:

```yaml
cause: carried
transitions:
  darker_faces:
    any:
      - 'The glare of the carried light is gone, and your eyes ease.'
      - 'With the bright light gone, your eyes stop aching.'
  darker_shapes:
    any:
      - 'The carried light is gone, and faces blur into shapes.'
      - 'Without the carried light, you can make out shapes, not faces.'
  darker_dark:
    any:
      - 'The carried light is gone, and darkness closes in.'
      - 'Without the carried light, you can see nothing.'
  lighter_shapes:
    any:
      - 'A light flares up close; shapes come out of the dark.'
      - 'Light spills from a carried flame; you can make out shapes.'
  lighter_faces:
    any:
      - 'A carried light spreads around you; faces are clear again.'
      - 'By the carried light, you can make out faces again.'
  dazzled:
    any:
      - 'A carried light blazes up close and stabs at your widened eyes.'
      - 'The carried light is too bright for your widened eyes.'
```

`lamp.yaml` (unreachable today: no lamp changes at runtime; authored so plan 5 needs no store change):

```yaml
cause: lamp
transitions:
  darker_faces:
    any:
      - 'The lamps dim, and the glare eases from your eyes.'
      - 'The lamplight softens; your eyes stop aching.'
  darker_shapes:
    any:
      - 'The lamps gutter low; faces blur into shapes.'
      - 'The lamplight fades, and you can make out shapes, not faces.'
  darker_dark:
    any:
      - 'The lamps go out, and darkness closes in.'
      - 'The last lamplight dies; you can see nothing.'
  lighter_shapes:
    any:
      - 'A lamp flickers to life; shapes come out of the dark.'
      - 'Weak lamplight spreads; you can make out shapes.'
  lighter_faces:
    any:
      - 'The lamps burn brighter; faces are clear again.'
      - 'The lamplight grows; you can make out faces again.'
  dazzled:
    any:
      - 'The lamplight stabs at your widened eyes.'
      - 'The lamps flare too bright for your widened eyes.'
```

`weather.yaml`:

```yaml
cause: weather
transitions:
  darker_faces:
    any:
      - 'Clouds cut the glare, and your eyes ease.'
      - 'The weather dims the glare; your eyes stop aching.'
  darker_shapes:
    any:
      - 'The weather dims the light; faces blur into shapes.'
      - 'Heavy cloud steals the light; you can make out shapes, not faces.'
  darker_dark:
    any:
      - 'The weather swallows the last light; you can see nothing.'
      - 'Heavy cloud blots out the sky, and darkness closes in.'
  lighter_shapes:
    any:
      - 'The weather thins, and shapes come out of the dark.'
      - 'The cloud breaks a little; you can make out shapes.'
  lighter_faces:
    any:
      - 'The weather clears, and faces are plain again.'
      - 'The sky opens; you can make out faces again.'
  dazzled:
    any:
      - 'The cloud breaks, and the bright sky stabs at your widened eyes.'
      - 'The weather clears into a glare too bright for your widened eyes.'
```

`sky.yaml` (outdoor and indoor differ; indoor lines say "from outside", never "windows", because not every roofed room has one):

```yaml
cause: sky
transitions:
  darker_faces:
    outdoor:
      - 'The sun sinks lower, and the glare eases from your eyes.'
      - 'The day softens; your eyes stop aching.'
    indoor:
      - 'The light from outside softens; your eyes ease.'
      - 'The glare from outside fades; your eyes stop aching.'
  darker_shapes:
    outdoor:
      - 'Dusk settles around you; faces blur into shapes.'
      - 'The sky darkens; you can make out shapes, not faces.'
    indoor:
      - 'The light from outside fades; faces blur into shapes.'
      - 'Dusk dims the room; you can make out shapes, not faces.'
  darker_dark:
    outdoor:
      - 'Night closes in, and you can see nothing.'
      - 'The last light leaves the sky; darkness closes in.'
    indoor:
      - 'The last light from outside fades; you can see nothing.'
      - 'Night falls outside, and darkness fills the room.'
  lighter_shapes:
    outdoor:
      - 'The sky pales; shapes come out of the dark.'
      - 'First light creeps in; you can make out shapes.'
    indoor:
      - 'Grey light reaches in from outside; shapes appear.'
      - 'A little light comes in from outside; you can make out shapes.'
  lighter_faces:
    outdoor:
      - 'Daylight grows; you can make out faces again.'
      - 'The sky brightens, and faces are clear again.'
    indoor:
      - 'Daylight fills the room; faces are clear again.'
      - 'The light from outside grows; you can make out faces.'
  dazzled:
    outdoor:
      - 'The bright sky stabs at your widened eyes.'
      - 'Daylight floods in, too bright for your widened eyes.'
    indoor:
      - 'Daylight pours in, too bright for your widened eyes.'
      - 'The light from outside stabs at your widened eyes.'
```

`eyes.yaml` (no light term explains the change, for example a night-sight draught wearing off):

```yaml
cause: eyes
transitions:
  darker_faces:
    any:
      - 'Your eyes settle, and the glare no longer hurts.'
      - 'The ache fades from your eyes; the light is bearable now.'
  darker_shapes:
    any:
      - 'Your eyes lose their edge; faces blur into shapes.'
      - 'Your sight dulls; you can make out shapes, not faces.'
  darker_dark:
    any:
      - 'Your eyes can no longer use this light; you can see nothing.'
      - 'Your sight fails in the gloom, and you can see nothing.'
  lighter_shapes:
    any:
      - 'Your eyes adjust; shapes come out of the dark.'
      - 'Your sight sharpens; you can make out shapes.'
  lighter_faces:
    any:
      - 'Your eyes adjust; you can make out faces again.'
      - 'Your sight sharpens, and faces are clear.'
  dazzled:
    any:
      - 'Your widened eyes ache in this light.'
      - 'The light here is suddenly too bright for your eyes.'
```

- [ ] **Step 4: Run to verify**

Run: `go test ./internal/lightnotice/ -count=1`
Expected: `ok`. If a line trips a rule, fix the LINE, not the rule.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/narration/light-notices/movement.yaml _datafiles/world/dogmud/narration/light-notices/carried.yaml _datafiles/world/dogmud/narration/light-notices/lamp.yaml _datafiles/world/dogmud/narration/light-notices/weather.yaml _datafiles/world/dogmud/narration/light-notices/sky.yaml _datafiles/world/dogmud/narration/light-notices/eyes.yaml internal/lightnotice/store_test.go
git commit -F - <<'EOF'
content(lightnotice): the shipped light-notice lines

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 6: `decide`, the pure trigger and attribution rules

**Files:**
- Create: `internal/lightnotice/tracker.go` (pure half)
- Create: `internal/lightnotice/tracker_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/lightnotice/tracker_test.go`:

```go
package lightnotice

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/lightscale"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

var (
	termsLit    = rooms.LightTerms{Level: 60, Sky: 60}
	termsDim    = rooms.LightTerms{Level: 40, Sky: 40}
	termsBright = rooms.LightTerms{Level: 80, Sky: 80}
)

func rec(room int, b messaging.Band, terms rooms.LightTerms) record {
	return record{roomId: room, band: b, terms: terms}
}

func obs(room int, b messaging.Band, terms rooms.LightTerms) observation {
	return observation{roomId: room, band: b, terms: terms}
}

func TestDecideTriggerRules(t *testing.T) {
	cases := []struct {
		name  string
		prev  record
		now   observation
		trig  Trigger
		speak bool
		tr    Transition
	}{
		{"move to darker speaks", rec(1, messaging.BandFaces, termsLit), obs(2, messaging.BandShapes, termsDim), TriggerMove, true, DarkerShapes},
		{"move to lighter is silent", rec(1, messaging.BandShapes, termsDim), obs(2, messaging.BandFaces, termsLit), TriggerMove, false, ""},
		{"move into dazzle speaks", rec(1, messaging.BandFaces, termsLit), obs(2, messaging.BandDazzled, termsBright), TriggerMove, true, IntoDazzle},
		{"combat darker speaks", rec(1, messaging.BandFaces, termsLit), obs(1, messaging.BandShapes, termsDim), TriggerCombatRound, true, DarkerShapes},
		{"combat lighter speaks", rec(1, messaging.BandShapes, termsDim), obs(1, messaging.BandFaces, termsLit), TriggerCombatRound, true, LighterFaces},
		{"command darker speaks", rec(1, messaging.BandFaces, termsLit), obs(1, messaging.BandShapes, termsDim), TriggerCommand, true, DarkerShapes},
		{"command lighter speaks", rec(1, messaging.BandShapes, termsDim), obs(1, messaging.BandFaces, termsLit), TriggerCommand, true, LighterFaces},
		{"quiet never speaks", rec(1, messaging.BandFaces, termsLit), obs(1, messaging.BandShapes, termsDim), TriggerQuiet, false, ""},
		{"same band is silent", rec(1, messaging.BandFaces, termsLit), obs(1, messaging.BandFaces, termsDim), TriggerCommand, false, ""},
	}
	for _, c := range cases {
		n, speak, next := decide(c.prev, true, c.now, c.trig)
		if speak != c.speak {
			t.Errorf("%s: speak = %v, want %v", c.name, speak, c.speak)
		}
		if speak && n.transition != c.tr {
			t.Errorf("%s: transition = %q, want %q", c.name, n.transition, c.tr)
		}
		if next.band != c.now.band || next.roomId != c.now.roomId || next.quiet {
			t.Errorf("%s: the observed band must always be recorded, got %+v", c.name, next)
		}
	}
}

func TestDecideSilentRecording(t *testing.T) {
	now := obs(1, messaging.BandShapes, termsDim)

	if _, speak, next := decide(record{}, false, now, TriggerCommand); speak || next.band != messaging.BandShapes {
		t.Errorf("first check must record silently, speak=%v next=%+v", speak, next)
	}

	asleep := now
	asleep.asleep = true
	_, speak, next := decide(rec(1, messaging.BandFaces, termsLit), true, asleep, TriggerCommand)
	if speak || !next.quiet {
		t.Errorf("a sleeper gets no notice and is marked quiet, speak=%v next=%+v", speak, next)
	}

	blind := now
	blind.blinded = true
	_, speak, next = decide(rec(1, messaging.BandFaces, termsLit), true, blind, TriggerCombatRound)
	if speak || !next.quiet {
		t.Errorf("a blinded player gets no notice and is marked quiet, speak=%v next=%+v", speak, next)
	}

	quiet := rec(1, messaging.BandFaces, termsLit)
	quiet.quiet = true
	_, speak, next = decide(quiet, true, now, TriggerCommand)
	if speak || next.quiet || next.band != messaging.BandShapes {
		t.Errorf("the first check after quiet records silently and clears quiet, speak=%v next=%+v", speak, next)
	}

	_, speak, _ = decide(next, true, obs(1, messaging.BandFaces, termsLit), TriggerCommand)
	if !speak {
		t.Error("the check after a silent resync must speak on a real change")
	}
}

func TestTransitionOf(t *testing.T) {
	cases := []struct {
		from, to messaging.Band
		want     Transition
	}{
		{messaging.BandDark, messaging.BandShapes, LighterShapes},
		{messaging.BandDark, messaging.BandFaces, LighterFaces},
		{messaging.BandShapes, messaging.BandFaces, LighterFaces},
		{messaging.BandDark, messaging.BandDazzled, IntoDazzle},
		{messaging.BandFaces, messaging.BandDazzled, IntoDazzle},
		{messaging.BandDazzled, messaging.BandFaces, DarkerFaces},
		{messaging.BandDazzled, messaging.BandShapes, DarkerShapes},
		{messaging.BandFaces, messaging.BandShapes, DarkerShapes},
		{messaging.BandFaces, messaging.BandDark, DarkerDark},
		{messaging.BandDazzled, messaging.BandDark, DarkerDark},
	}
	for _, c := range cases {
		if got := transitionOf(c.from, c.to); got != c.want {
			t.Errorf("%v -> %v = %q, want %q", c.from, c.to, got, c.want)
		}
	}
}

func TestAttribution(t *testing.T) {
	base := rooms.LightTerms{Level: 60, Sky: 55, Lamp: 40, HasLamp: true}
	with := func(f func(*rooms.LightTerms)) rooms.LightTerms { t2 := base; f(&t2); return t2 }

	cases := []struct {
		name string
		prev record
		now  observation
		want Cause
	}{
		{"a different room is movement",
			rec(1, messaging.BandFaces, base), obs(2, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 40 })), CauseMovement},
		{"carried light leaving",
			rec(1, messaging.BandFaces, with(func(x *rooms.LightTerms) { x.Carried = true })), obs(1, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 40 })), CauseCarried},
		{"lamp value changing",
			rec(1, messaging.BandFaces, base), obs(1, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 40; x.Lamp = 20 })), CauseLamp},
		{"LightMod bridge changing is the lamp",
			rec(1, messaging.BandFaces, with(func(x *rooms.LightTerms) { x.LightMod = 2 })), obs(1, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 40 })), CauseLamp},
		{"weather occlusion",
			rec(1, messaging.BandFaces, base), obs(1, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 40; x.OcclusionSteps = 1; x.Sky = 30 })), CauseWeather},
		{"the sky alone",
			rec(1, messaging.BandFaces, base), obs(1, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 40; x.Sky = 30 })), CauseSky},
		{"sky gone entirely counts as the sky",
			rec(1, messaging.BandFaces, base), obs(1, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 40; x.Sky = lightscale.Absent() })), CauseSky},
		{"level unchanged is the eyes",
			rec(1, messaging.BandFaces, base), obs(1, messaging.BandShapes, base), CauseEyes},
		{"level rose but the band fell is the eyes, not the sky",
			rec(1, messaging.BandFaces, base), obs(1, messaging.BandShapes, with(func(x *rooms.LightTerms) { x.Level = 61; x.Sky = 56 })), CauseEyes},
	}
	for _, c := range cases {
		tr := transitionOf(c.prev.band, c.now.band)
		if got := attribute(c.prev, c.now, tr); got != c.want {
			t.Errorf("%s: cause = %q, want %q", c.name, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/lightnotice/ -run 'TestDecide|TestTransitionOf|TestAttribution' -count=1`
Expected: `undefined: record`.

- [ ] **Step 3: Implement**

`internal/lightnotice/tracker.go` (pure half; Task 7 appends the stateful half):

```go
package lightnotice

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Trigger names why a check is running, which decides what it may announce.
type Trigger uint8

const (
	// TriggerMove runs after a player arrives in a room. It announces only a
	// darker band or dazzle: walking into light needs no notice, the room
	// description already says it.
	TriggerMove Trigger = iota
	// TriggerCombatRound runs once per combat round the player is fighting in.
	TriggerCombatRound
	// TriggerCommand runs before every command the player issues, so the
	// notice lands before the command's own output.
	TriggerCommand
	// TriggerQuiet records the current band and never speaks: login.
	TriggerQuiet
)

// observation is one moment of one player's light, gathered by Check.
type observation struct {
	roomId  int
	band    messaging.Band
	terms   rooms.LightTerms
	indoor  bool
	asleep  bool
	blinded bool
}

// record is what was last announced (or silently recorded) to a player.
type record struct {
	roomId int
	band   messaging.Band
	terms  rooms.LightTerms
	// quiet says the player was asleep or blinded since the last record, so
	// the next attentive check re-records silently. Their end must never read
	// as the light changing.
	quiet bool
}

// notice is what decide asks Check to say.
type notice struct {
	cause      Cause
	transition Transition
	indoor     bool
}

// decide applies the spec's trigger rules. It returns the notice to send, if
// any, and the record to store. It is pure so every rule is table-testable.
func decide(prev record, known bool, now observation, trigger Trigger) (notice, bool, record) {
	if now.asleep || now.blinded {
		prev.quiet = true
		return notice{}, false, prev
	}
	next := record{roomId: now.roomId, band: now.band, terms: now.terms}
	if !known || prev.quiet || trigger == TriggerQuiet || now.band == prev.band {
		return notice{}, false, next
	}
	tr := transitionOf(prev.band, now.band)
	if trigger == TriggerMove && (tr == LighterShapes || tr == LighterFaces) {
		return notice{}, false, next
	}
	return notice{cause: attribute(prev, now, tr), transition: tr, indoor: now.indoor}, true, next
}

// transitionOf names a band change. Bands run darkest to brightest.
func transitionOf(from, to messaging.Band) Transition {
	switch {
	case to == messaging.BandDazzled:
		return IntoDazzle
	case to > from && to == messaging.BandFaces:
		return LighterFaces
	case to > from:
		return LighterShapes
	case to == messaging.BandFaces:
		return DarkerFaces
	case to == messaging.BandShapes:
		return DarkerShapes
	}
	return DarkerDark
}

// attribute names the likeliest cause of a band change.
//
// A room change is movement. Otherwise, if the light LEVEL did not move in the
// band's direction, no light term explains the change and the observer's own
// sight did (a draught wearing off): that is checked BEFORE the terms, because
// the sky drifts a little almost every round and would otherwise take the
// blame for everything. Then the first term that moved, in the order carried
// light, the room's own light, weather, sky.
func attribute(prev record, now observation, tr Transition) Cause {
	if prev.roomId != now.roomId {
		return CauseMovement
	}
	a, b := prev.terms, now.terms
	darker := tr == DarkerFaces || tr == DarkerShapes || tr == DarkerDark
	if (darker && b.Level >= a.Level) || (!darker && b.Level <= a.Level) {
		return CauseEyes
	}
	switch {
	case a.Carried != b.Carried:
		return CauseCarried
	case a.HasLamp != b.HasLamp || a.Lamp != b.Lamp || a.LightMod != b.LightMod:
		return CauseLamp
	case a.OcclusionSteps != b.OcclusionSteps:
		return CauseWeather
	case skyMoved(a.Sky, b.Sky):
		return CauseSky
	}
	return CauseEyes
}

func skyMoved(a, b float64) bool {
	aAbsent, bAbsent := math.IsInf(a, -1), math.IsInf(b, -1)
	if aAbsent || bAbsent {
		return aAbsent != bAbsent
	}
	return math.Abs(a-b) > 1e-9
}
```

- [ ] **Step 4: Run to verify**

Run: `go test ./internal/lightnotice/ -count=1`
Expected: `ok`.

- [ ] **Step 5: Prove the move rule can fail**

Temporarily delete the `if trigger == TriggerMove && ...` block, run `go test ./internal/lightnotice/ -run TestDecideTriggerRules -count=1`, confirm "move to lighter is silent" FAILS, restore.

- [ ] **Step 6: Commit**

```bash
git add internal/lightnotice/tracker.go internal/lightnotice/tracker_test.go
git commit -F - <<'EOF'
feat(lightnotice): trigger rules and cause attribution, as a pure function

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 7: `Check`, `Forget`, `NoteAttention`, `ResetForTest`

**Files:**
- Modify: `internal/lightnotice/tracker.go` (append), `internal/lightnotice/store.go` (delete the temporary `ResetForTest`)
- Create: `internal/lightnotice/check_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/lightnotice/check_test.go`:

```go
package lightnotice

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// seedLampWorld builds two sky-less rooms whose light is exactly their lamp, so
// bands are fixed without a clock: 60 faces, 30 shapes, 80 dazzled.
func seedLampWorld(t *testing.T) (*users.UserRecord, *rooms.Room, *rooms.Room) {
	t.Helper()
	r1 := &rooms.Room{RoomId: 1, Zone: "LightZone", SkyLight: rooms.SkyLightPtr(0), Lamp: rooms.LampPtr(60)}
	r2 := &rooms.Room{RoomId: 2, Zone: "LightZone", SkyLight: rooms.SkyLightPtr(0), Lamp: rooms.LampPtr(30)}
	t.Cleanup(rooms.SeedRoomsForTest(
		map[int]*rooms.Room{1: r1, 2: r2},
		map[string]*rooms.ZoneConfig{"LightZone": {Name: "LightZone", RoomId: 1, RoomIds: map[int]struct{}{1: {}, 2: {}}}},
	))
	t.Cleanup(rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{}))
	u := users.NewTestUser(1, "alice", "Aliceia", 1001)
	u.Character.RoomId = 1
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{1: u}))
	if err := LoadFrom(shippedDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ResetForTest)
	return u, r1, r2
}

// captureFor returns a drain function yielding every message text sent to one
// user since the capture began.
func captureFor(t *testing.T, userId int) func() []string {
	t.Helper()
	events.ProcessEvents() // drop anything queued before the capture
	var got []string
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if m, ok := e.(events.Message); ok && m.UserId == userId {
			got = append(got, strings.ReplaceAll(m.Text, "\n", " "))
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return func() []string {
		events.ProcessEvents()
		out := got
		got = nil
		return out
	}
}

func containsAny(text string, pool []string) bool {
	for _, l := range pool {
		if strings.Contains(text, l) {
			return true
		}
	}
	return false
}

func TestCheckSpeaksOnceThenFallsSilent(t *testing.T) {
	u, r1, _ := seedLampWorld(t)
	drain := captureFor(t, 1)

	Check(u, TriggerQuiet)
	r1.Lamp = rooms.LampPtr(30)
	Check(u, TriggerCommand)
	got := drain()
	if len(got) != 1 || !containsAny(got[0], Pool(CauseLamp, DarkerShapes, false)) {
		t.Fatalf("want exactly one lamp darker_shapes line, got %q", got)
	}

	Check(u, TriggerCommand)
	if got := drain(); len(got) != 0 {
		t.Fatalf("a second check in the same band must be silent, got %q", got)
	}
}

func TestCheckMoveUsesMovementCause(t *testing.T) {
	u, _, _ := seedLampWorld(t)
	drain := captureFor(t, 1)

	Check(u, TriggerQuiet)
	u.Character.RoomId = 2
	Check(u, TriggerMove)
	got := drain()
	if len(got) != 1 || !containsAny(got[0], Pool(CauseMovement, DarkerShapes, false)) {
		t.Fatalf("want one movement darker_shapes line, got %q", got)
	}

	u.Character.RoomId = 1
	Check(u, TriggerMove)
	if got := drain(); len(got) != 0 {
		t.Fatalf("moving into better light must be silent, got %q", got)
	}
}

func TestBlindnessEndingIsSilent(t *testing.T) {
	u, r1, _ := seedLampWorld(t)
	drain := captureFor(t, 1)
	u.Character.Perception = perception.NewMachine()

	Check(u, TriggerQuiet)
	if err := u.Character.Perception.TransitionTo(perception.Blinded, state.TransitionReason{Trigger: "test"}); err != nil {
		t.Fatal(err)
	}
	NoteAttention(u)
	r1.Lamp = rooms.LampPtr(30)
	if err := u.Character.Perception.TransitionTo(perception.Sighted, state.TransitionReason{Trigger: "test"}); err != nil {
		t.Fatal(err)
	}
	Check(u, TriggerCommand)
	if got := drain(); len(got) != 0 {
		t.Fatalf("the first check after blindness must record silently, got %q", got)
	}

	r1.Lamp = rooms.LampPtr(60)
	Check(u, TriggerCommand)
	if got := drain(); len(got) != 1 {
		t.Fatalf("after the silent resync a real change must speak, got %q", got)
	}
}

func TestForgetMakesTheNextCheckSilent(t *testing.T) {
	u, r1, _ := seedLampWorld(t)
	drain := captureFor(t, 1)

	Check(u, TriggerQuiet)
	Forget(u.UserId)
	r1.Lamp = rooms.LampPtr(30)
	Check(u, TriggerCommand)
	if got := drain(); len(got) != 0 {
		t.Fatalf("after Forget the first check records silently, got %q", got)
	}
}

func TestUnloadedStoreNeverSpeaks(t *testing.T) {
	u, r1, _ := seedLampWorld(t)
	ResetForTest()
	drain := captureFor(t, 1)

	Check(u, TriggerQuiet)
	r1.Lamp = rooms.LampPtr(30)
	Check(u, TriggerCommand)
	if got := drain(); len(got) != 0 {
		t.Fatalf("an unloaded store must be silent, got %q", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/lightnotice/ -run 'TestCheck|TestBlindness|TestForget|TestUnloaded' -count=1`
Expected: `undefined: Check`.

- [ ] **Step 3: Implement**

Delete the temporary `ResetForTest` at the bottom of `store.go`. Replace the `tracker.go` import block with:

```go
import (
	"math"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
	"github.com/GoMudEngine/GoMud/internal/users"
)
```

Append to `tracker.go`:

```go
// Per-player state, in memory only: cleared on logout, never saved.
var (
	mu      sync.Mutex
	records = map[int]record{}
)

// Check compares the player's current band with the last one recorded for
// them and, if the trigger's rule allows, sends one notice.
func Check(user *users.UserRecord, trigger Trigger) {
	if user == nil || user.Character == nil {
		return
	}
	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return
	}
	now := observe(user.Character, room)

	mu.Lock()
	prev, known := records[user.UserId]
	n, speak, next := decide(prev, known, now, trigger)
	records[user.UserId] = next
	mu.Unlock()

	if !speak {
		return
	}
	if text, ok := line(n.cause, n.transition, n.indoor, nil); ok {
		user.SendText(messaging.CategoryLight, text)
	}
}

// NoteAttention marks a sleeping or blinded player so the first check after
// they wake or see again records silently. It computes no light and sends no
// text, so it is cheap enough to run for every player every round. It is the
// seam for waking: sleep ends at many hand-rolled sites and by expiry, and no
// event announces it.
func NoteAttention(user *users.UserRecord) {
	if user == nil || user.Character == nil || !inattentive(user.Character) {
		return
	}
	mu.Lock()
	r := records[user.UserId]
	r.quiet = true
	records[user.UserId] = r
	mu.Unlock()
}

// Forget drops a player's record, at logout.
func Forget(userId int) {
	mu.Lock()
	delete(records, userId)
	mu.Unlock()
}

// ResetForTest unloads the store and clears every record.
func ResetForTest() {
	mu.Lock()
	records = map[int]record{}
	mu.Unlock()
	loaded = nil
}

func inattentive(c *characters.Character) bool {
	return c.HasConditionFlag(conditions.Sleeping) ||
		(c.Perception != nil && c.Perception.State() == perception.Blinded)
}

// fixedLight hands LightBand the level LightTerms already computed, so band
// and terms come from one computation rather than two.
type fixedLight int

func (l fixedLight) LightLevel() int { return int(l) }

func observe(c *characters.Character, room *rooms.Room) observation {
	terms := room.LightTerms()
	indoor := false
	if b := room.GetBiome(); b != nil {
		indoor = b.Indoor
	}
	return observation{
		roomId:  room.RoomId,
		band:    messaging.LightBand(c, fixedLight(terms.Level)),
		terms:   terms,
		indoor:  indoor,
		asleep:  c.HasConditionFlag(conditions.Sleeping),
		blinded: c.Perception != nil && c.Perception.State() == perception.Blinded,
	}
}
```

- [ ] **Step 4: Run to verify**

Run: `go test ./internal/lightnotice/ -count=1 -race`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/lightnotice/tracker.go internal/lightnotice/store.go internal/lightnotice/check_test.go
git commit -F - <<'EOF'
feat(lightnotice): Check, NoteAttention and Forget over in-memory records

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 8: Listeners and the boot loader

**Files:**
- Create: `internal/hooks/LightNotice_Triggers.go`
- Create: `internal/hooks/LightNotice_Triggers_test.go`
- Modify: `internal/hooks/hooks.go` (registrations), `main.go:1947`

- [ ] **Step 1: Write the failing tests**

`internal/hooks/LightNotice_Triggers_test.go`:

```go
package hooks

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/lightnotice"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

const lightNoticeDir = "../../_datafiles/world/dogmud/narration/light-notices"

// lampRooms pins rooms 1 and 2 of the seeded registries to sky-less lamp light.
func lampRooms(t *testing.T, lamp1, lamp2 int) (*rooms.Room, *rooms.Room) {
	t.Helper()
	r1, r2 := rooms.LoadRoom(1), rooms.LoadRoom(2)
	require.NotNil(t, r1)
	require.NotNil(t, r2)
	r1.SkyLight, r1.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(lamp1)
	r2.SkyLight, r2.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(lamp2)
	return r1, r2
}

func loadLightNotices(t *testing.T) {
	t.Helper()
	require.NoError(t, lightnotice.LoadFrom(lightNoticeDir))
	t.Cleanup(lightnotice.ResetForTest)
}

func lightLines(msgs []events.Message, userId int) []string {
	var pool []string
	for _, c := range lightnotice.Causes() {
		for _, tr := range lightnotice.Transitions() {
			pool = append(pool, lightnotice.Pool(c, tr, false)...)
			pool = append(pool, lightnotice.Pool(c, tr, true)...)
		}
	}
	var out []string
	for _, m := range msgs {
		if m.UserId != userId {
			continue
		}
		text := strings.ReplaceAll(m.Text, "\n", " ")
		for _, l := range pool {
			if strings.Contains(text, l) {
				out = append(out, l)
				break
			}
		}
	}
	return out
}

func TestLightNoticeOnMove(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	loadLightNotices(t)
	lampRooms(t, 60, 30)

	u := users.GetByUserId(1)
	require.NotNil(t, u)
	u.Character.RoomId = 1
	lightnotice.Check(u, lightnotice.TriggerQuiet)

	captured, capCleanup := captureMessages(t)
	u.Character.RoomId = 2
	LightNoticeOnMove(events.RoomChange{UserId: 1, FromRoomId: 1, ToRoomId: 2})
	u.Character.RoomId = 1
	LightNoticeOnMove(events.RoomChange{UserId: 1, FromRoomId: 2, ToRoomId: 1})
	capCleanup()

	got := lightLines(*captured, 1)
	require.Len(t, got, 1, "darker move speaks once, lighter move is silent")
	require.Contains(t, lightnotice.Pool(lightnotice.CauseMovement, lightnotice.DarkerShapes, false), got[0])
}

func TestLightNoticeOnMoveIgnoresMobs(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	loadLightNotices(t)
	require.Equal(t, events.Continue, LightNoticeOnMove(events.RoomChange{MobInstanceId: 100, FromRoomId: 1, ToRoomId: 2}))
}

func TestLightNoticeDespawnForgets(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	loadLightNotices(t)
	r1, _ := lampRooms(t, 60, 30)

	u := users.GetByUserId(1)
	u.Character.RoomId = 1
	LightNoticeOnSpawn(events.PlayerSpawn{UserId: 1, RoomId: 1})
	LightNoticeOnDespawn(events.PlayerDespawn{UserId: 1, RoomId: 1})

	captured, capCleanup := captureMessages(t)
	r1.Lamp = rooms.LampPtr(30)
	lightnotice.Check(u, lightnotice.TriggerCommand)
	capCleanup()
	require.Empty(t, lightLines(*captured, 1), "after despawn the next check records silently")
}

// TestLightNoticeCombatRound drives the real combat pass: a fighting player
// whose light changed gets the notice from handlePlayerCombat.
func TestLightNoticeCombatRound(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	loadLightNotices(t)
	r1, _ := lampRooms(t, 60, 30)

	u := users.GetByUserId(1)
	u.Character.RoomId = 1
	lightnotice.Check(u, lightnotice.TriggerQuiet)
	u.Character.SetAggro(0, 100, characters.DefaultAttack)
	require.True(t, u.Character.IsInCombat())

	r1.Lamp = rooms.LampPtr(30)
	captured, capCleanup := captureMessages(t)
	handlePlayerCombat(events.NewRound{})
	capCleanup()

	got := lightLines(*captured, 1)
	require.Len(t, got, 1, "one notice per crossing during a combat round")
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/hooks/ -run TestLightNotice -count=1`
Expected: `undefined: LightNoticeOnMove`.

- [ ] **Step 3: Implement the listeners**

`internal/hooks/LightNotice_Triggers.go`:

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/lightnotice"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Lighting plan 3d's event seams into internal/lightnotice. The command and
// combat-round seams are direct calls in usercommands.TryCommand and
// handlePlayerCombat; these four ride events.

// LightNoticeOnMove checks a player after they arrive in a room. RoomChange is
// queued, so this runs after the move command's own output; the command's
// TryCommand check ran BEFORE the move, in the old room, so the two never
// announce the same crossing.
func LightNoticeOnMove(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok || evt.UserId == 0 || evt.MobInstanceId != 0 {
		return events.Continue
	}
	lightnotice.Check(users.GetByUserId(evt.UserId), lightnotice.TriggerMove)
	return events.Continue
}

// LightNoticeOnSpawn records a logging-in player's band silently.
func LightNoticeOnSpawn(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.PlayerSpawn); ok {
		lightnotice.Check(users.GetByUserId(evt.UserId), lightnotice.TriggerQuiet)
	}
	return events.Continue
}

// LightNoticeOnDespawn drops a leaving player's record.
func LightNoticeOnDespawn(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.PlayerDespawn); ok {
		lightnotice.Forget(evt.UserId)
	}
	return events.Continue
}

// LightNoticeAttention marks every sleeping or blinded player so waking or
// seeing again records silently. Two flag reads per player, no light maths.
func LightNoticeAttention(e events.Event) events.ListenerReturn {
	for _, userId := range users.GetOnlineUserIds() {
		lightnotice.NoteAttention(users.GetByUserId(userId))
	}
	return events.Continue
}
```

In `internal/hooks/hooks.go`, add after line 40 (`PresencePlayerEntry`):

```go
	events.RegisterListener(events.RoomChange{}, LightNoticeOnMove)
```

After line 64 (`IdleMobs`) inside the NewRound block:

```go
	events.RegisterListener(events.NewRound{}, LightNoticeAttention)
```

After line 84 (`HandleJoin`):

```go
	events.RegisterListener(events.PlayerSpawn{}, LightNoticeOnSpawn)
```

Before line 89 (`HandleLeave`, which is `events.Last`):

```go
	events.RegisterListener(events.PlayerDespawn{}, LightNoticeOnDespawn)
```

Read the file around each anchor first; insert beside the named line, not at a guessed number.

- [ ] **Step 4: Implement the combat-round call**

In `internal/hooks/NewRound_DoCombat.go` `handlePlayerCombat`, between the end of the `if !ValidateAggro(user.Character) { ... }` block and `user.Character.CancelCombatConditions()`:

```go
		// Lighting plan 3d: a fighting player hears about a light crossing
		// each round, in both directions, before this round's combat text.
		lightnotice.Check(user, lightnotice.TriggerCombatRound)

```

Add `"github.com/GoMudEngine/GoMud/internal/lightnotice"` to that file's imports.

- [ ] **Step 5: Boot loader**

In `main.go`, after `movenarration.LoadMoveNarrationFiles()`:

```go
	lightnotice.LoadLightNoticeFiles()
```

Add the import `"github.com/GoMudEngine/GoMud/internal/lightnotice"`.

- [ ] **Step 6: Run to verify**

Run: `go test ./internal/hooks/ -run TestLightNotice -count=1` then `go build ./...`
Expected: `ok`, clean build.

- [ ] **Step 7: Prove the combat test can fail**

Comment out the `lightnotice.Check(user, lightnotice.TriggerCombatRound)` line, run `go test ./internal/hooks/ -run TestLightNoticeCombatRound -count=1`, confirm FAIL, restore.

- [ ] **Step 8: Run the whole hooks package**

Run: `go test ./internal/hooks/ -count=1`
Expected: `ok`. Nothing else loads the store, so no other test sees a new message (fact 22). If a test does see one, it means the store leaked through `ResetForTest` not running; fix the leak, do not weaken the test.

- [ ] **Step 9: Commit**

```bash
git add internal/hooks/LightNotice_Triggers.go internal/hooks/LightNotice_Triggers_test.go internal/hooks/hooks.go internal/hooks/NewRound_DoCombat.go main.go
git commit -F - <<'EOF'
feat(hooks): wire light notices to moves, combat rounds, login and logout

Waking and the end of blindness have no single seam, so a NewRound
sweep marks sleeping and blinded players quiet with two flag reads each;
no light is computed per round.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 9: The command seam

**Files:**
- Modify: `internal/usercommands/usercommands.go:316-322`
- Create: `internal/usercommands/light_notice_wiring_test.go`

- [ ] **Step 1: Write the failing test**

`internal/usercommands/light_notice_wiring_test.go`:

```go
package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/lightnotice"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// TestLightNoticeArrivesBeforeTheCommandOutput pins the spec's ordering: the
// notice is delivered before the command's own output, and only once.
func TestLightNoticeArrivesBeforeTheCommandOutput(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	require.NoError(t, lightnotice.LoadFrom("../../_datafiles/world/dogmud/narration/light-notices"))
	t.Cleanup(lightnotice.ResetForTest)

	room1 := rooms.LoadRoom(1)
	require.NotNil(t, room1)
	room1.SkyLight, room1.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(30)
	u := users.GetByUserId(1)
	require.NotNil(t, u)
	lightnotice.Check(u, lightnotice.TriggerQuiet)
	room1.Lamp = rooms.LampPtr(60)

	pool := lightnotice.Pool(lightnotice.CauseLamp, lightnotice.LighterFaces, false)
	run := func() (noticeAt, titleAt, notices int) {
		events.ProcessEvents()
		var texts []string
		id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
			if m, ok := e.(events.Message); ok && m.UserId == 1 {
				texts = append(texts, strings.ReplaceAll(m.Text, "\n", " "))
			}
			return events.Continue
		})
		defer events.UnregisterListener(events.Message{}, id)
		_, err := TryCommand("look", "", 1, 0)
		require.NoError(t, err)
		events.ProcessEvents()
		noticeAt, titleAt = -1, -1
		for i, text := range texts {
			for _, l := range pool {
				if strings.Contains(text, l) {
					notices++
					if noticeAt < 0 {
						noticeAt = i
					}
				}
			}
			if titleAt < 0 && strings.Contains(text, "Town Square") {
				titleAt = i
			}
		}
		return
	}

	noticeAt, titleAt, notices := run()
	require.Equal(t, 1, notices, "exactly one notice")
	require.GreaterOrEqual(t, titleAt, 0, "look must print the room title")
	require.Less(t, noticeAt, titleAt, "the notice must precede the room description")

	_, _, notices = run()
	require.Equal(t, 0, notices, "a second look in the same band says nothing")
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/usercommands/ -run TestLightNoticeArrivesBeforeTheCommandOutput -count=1`
Expected: FAIL, `exactly one notice` (0 notices).

- [ ] **Step 3: Implement**

In `TryCommand`, immediately after the `if user == nil { ... }` block:

```go
	// Lighting plan 3d: tell the player about any light crossing since they
	// last acted, before this command's own output. It runs before scripts,
	// behaviour trees and quest intercepts so an intercepted command still
	// gets it.
	lightnotice.Check(user, lightnotice.TriggerCommand)
```

Add the import `"github.com/GoMudEngine/GoMud/internal/lightnotice"`.

- [ ] **Step 4: Run to verify**

Run: `go test ./internal/usercommands/ -count=1`
Expected: `ok` for the whole package.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/usercommands.go internal/usercommands/light_notice_wiring_test.go
git commit -F - <<'EOF'
feat(usercommands): a light notice precedes the command that reveals it

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 10: The narration snapshot golden

**Files:**
- Modify: `internal/narration/snapshot_test.go`
- Create: `internal/narration/testdata/stores/light_notices.golden` (generated)

- [ ] **Step 1: Add the builder and subtest**

In `TestSnapshotStores`, after the `special_moves` subtest:

```go
	t.Run("light_notices", func(t *testing.T) {
		checkGolden(t, "light_notices.golden", buildLightNoticesGolden(t))
	})
```

Add near `buildSpecialMovesGolden`:

```go
// ---------------------------------------------------------------------
// Store: light notices (internal/lightnotice), lighting plan 3d
// ---------------------------------------------------------------------

// buildLightNoticesGolden freezes every line of every cause, transition and
// setting, every index, so a dropped, reordered or reworded line goes red.
func buildLightNoticesGolden(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(dogmudDataDir(t), "narration", "light-notices")
	groups, err := fileloader.LoadAllFlatFiles[string, *lightnotice.CauseGroup](dir)
	if err != nil {
		t.Fatalf("loading %s: %v", dir, err)
	}
	if len(groups) == 0 {
		t.Fatal("no light-notice files loaded; the golden would be vacuous")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# light-notices store snapshot (lighting plan 3d)\n")
	fmt.Fprintf(&b, "# dimensions: cause x transition x setting(any/outdoor/indoor) x index\n\n")
	for _, c := range lightnotice.Causes() {
		g := groups[string(c)]
		if g == nil {
			t.Fatalf("cause %q has no file", c)
		}
		for _, tr := range lightnotice.Transitions() {
			p := g.Transitions[tr]
			if p == nil {
				t.Fatalf("cause %q lost transition %q", c, tr)
			}
			for _, set := range []struct {
				name string
				pool []string
			}{{"any", p.Any}, {"outdoor", p.Outdoor}, {"indoor", p.Indoor}} {
				for i, text := range set.pool {
					fmt.Fprintf(&b, "%s|%s|%s|%d => %s\n", c, tr, set.name, i, text)
				}
			}
		}
	}
	return b.String()
}
```

Add `"github.com/GoMudEngine/GoMud/internal/lightnotice"` to the imports.

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/narration/ -run TestSnapshotStores/light_notices -count=1`
Expected: FAIL, "read golden ... light_notices.golden".

- [ ] **Step 3: Generate, then read it**

Run: `go test ./internal/narration/ -run TestSnapshotStores/light_notices -count=1 -update`
Open the golden and confirm it has 84 rows (30 `any` pools and 6 `sky` split pools, 2 lines each: 30*2 + 6*2*2 = 84).

- [ ] **Step 4: Prove it can fail**

Change one word in `_datafiles/world/dogmud/narration/light-notices/eyes.yaml`, run without `-update`, confirm FAIL, restore, confirm PASS.

- [ ] **Step 5: Confirm no other golden moved**

Run: `go test ./internal/narration/ -count=1`
Expected: `ok`, and `git status --short internal/narration/testdata` lists only the new golden.

- [ ] **Step 6: Commit**

```bash
git add internal/narration/snapshot_test.go internal/narration/testdata/stores/light_notices.golden
git commit -F - <<'EOF'
test(narration): snapshot the light-notice store

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 11: Integration against shipped biomes and the clock

**Files:**
- Create: `internal/lightnotice/integration_test.go`

- [ ] **Step 1: Write the tests**

```go
package lightnotice

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// withShippedWorld loads the REAL biome files and pins the clock, following
// internal/rooms/city_tier_light_test.go's withShippedBiomesAndClock.
func withShippedWorld(t *testing.T) {
	t.Helper()
	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = `../../_datafiles/world/dogmud`
	cfg.Timing.RoundsPerDay = 900
	cfg.Timing.RoundSeconds = 4
	cfg.Timing.Validate()
	cfg.Balance.Validate()
	configs.SetConfigForTest(t, cfg)

	t.Cleanup(rooms.SeedBiomesForTest(nil))
	rooms.LoadBiomeDataFiles()

	original := util.GetRoundCount()
	t.Cleanup(func() {
		util.SetRoundCountForTest(original)
		gametime.ClearDateCacheForTest()
		gametime.ClearCelestialMemoForTest()
	})

	if err := LoadFrom(shippedDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ResetForTest)
}

// setClock moves the world to a day of year and an hour (900 rounds a day).
func setClock(doy int, hour float64) {
	util.SetRoundCountForTest(uint64(float64(doy-1)*900 + hour*37.5))
	gametime.ClearDateCacheForTest()
	gametime.ClearCelestialMemoForTest()
}

func seedCity(t *testing.T) {
	t.Helper()
	rs := map[int]*rooms.Room{
		1: {RoomId: 1, Zone: "City", Biome: "city_backstreet"},
		2: {RoomId: 2, Zone: "City", Biome: "city_thoroughfare"},
		3: {RoomId: 3, Zone: "City", Biome: "cave"},
	}
	t.Cleanup(rooms.SeedRoomsForTest(rs, map[string]*rooms.ZoneConfig{
		"City": {Name: "City", RoomId: 1, RoomIds: map[int]struct{}{1: {}, 2: {}, 3: {}}},
	}))
}

// A backstreet loses faces between noon and midnight on midsummer day; a
// thoroughfare never does. Exactly one notice each time the band changes.
func TestDuskInTheBackstreetButNotOnTheThoroughfare(t *testing.T) {
	withShippedWorld(t)
	seedCity(t)
	lane := users.NewTestUser(1, "alice", "Aliceia", 1001)
	lane.Character.RoomId = 1
	street := users.NewTestUser(2, "bob", "Bobrick", 1002)
	street.Character.RoomId = 2
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{1: lane, 2: street}))

	drainLane := captureFor(t, 1)
	drainMain := captureFor(t, 2)

	setClock(172, 12)
	Check(lane, TriggerQuiet)
	Check(street, TriggerQuiet)

	setClock(172, 0)
	Check(lane, TriggerCommand)
	Check(street, TriggerCommand)

	got := drainLane()
	if len(got) != 1 || !containsAny(got[0], Pool(CauseSky, DarkerShapes, false)) {
		t.Fatalf("backstreet at midnight: want one sky darker_shapes line, got %q", got)
	}
	if got := drainMain(); len(got) != 0 {
		t.Fatalf("a thoroughfare keeps faces all night, got %q", got)
	}

	Check(lane, TriggerCommand)
	if got := drainLane(); len(got) != 0 {
		t.Fatalf("a second check says nothing, got %q", got)
	}
}

// A night-sight 24 player walking out of a cave into a lamplit thoroughfare is
// dazzled: the shifted dazzle edge is 51 and the street lamp alone is 52.
func TestNightSightIsDazzledByTheThoroughfare(t *testing.T) {
	withShippedWorld(t)
	seedCity(t)
	const catEye = 99024
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		catEye: {
			ConditionId: catEye,
			Name:        "Test Cat Eye",
			Flags:       []conditions.Flag{conditions.NightVision},
			Effects:     map[conditions.EffectKind]conditions.EffectValue{conditions.EffectNightVisionStrength: {Literal: 24}},
		},
	}))
	u := users.NewTestUser(1, "alice", "Aliceia", 1001)
	u.Character.RoomId = 3
	if err := u.Character.AddCondition(catEye, true); err != nil {
		t.Fatal(err)
	}
	if got := u.Character.NightVisionStrength(); got != 24 {
		t.Fatalf("fixture strength = %d, want 24", got)
	}
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{1: u}))
	drain := captureFor(t, 1)

	setClock(172, 0)
	Check(u, TriggerQuiet)
	u.Character.RoomId = 2
	Check(u, TriggerMove)

	got := drain()
	if len(got) != 1 || !containsAny(got[0], Pool(CauseMovement, IntoDazzle, false)) {
		t.Fatalf("want one movement dazzle line, got %q", got)
	}
}
```

- [ ] **Step 2: Run**

Run: `go test ./internal/lightnotice/ -count=1 -race`
Expected: `ok`. If the backstreet's noon band is not faces or the thoroughfare's midnight level is below 51 on day 172, the test is measuring a different world than fact 21 says: STOP and report the measured levels rather than moving the hour.

- [ ] **Step 3: Prove the dazzle test can fail**

Temporarily change `windowDazzleEdge` to `76` in `internal/messaging/window.go`, run `go test ./internal/lightnotice/ -run TestNightSightIsDazzled -count=1`, confirm FAIL, restore.

- [ ] **Step 4: Commit**

```bash
git add internal/lightnotice/integration_test.go
git commit -F - <<'EOF'
test(lightnotice): dusk in a backstreet, dazzle on a thoroughfare

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 12: Documentation

**Files:**
- Create: `internal/lightnotice/context.md`
- Modify: `internal/messaging/context.md`, `internal/rooms/context.md`, `internal/narration/context.md`, `docs/README.md`

- [ ] **Step 1: Extract the real surface before writing a word**

Run: `grep -nE '^(func|type|const|var)\s|^\t(Cause|Trigger|Darker|Lighter|IntoDazzle|Band)' internal/lightnotice/*.go internal/messaging/band.go`
Document only what that prints.

- [ ] **Step 2: Write `internal/lightnotice/context.md`**

Sections, following `internal/movenarration/context.md`'s shape:
1. **What it is**: band-change notices for players (lighting plan 3d); Go decides whether and why, YAML holds the words.
2. **Bands and triggers**: the spec's two tables (bands; move = darker or dazzle only, combat round and command = both directions; login quiet; sleeping and blinded silent, their end recorded silently; mobs never).
3. **Attribution order**: movement, then "the level did not move the band's way" is the eyes, then carried, lamp (lamp value or LightMod bridge), weather occlusion, sky, else eyes. Say why the eyes check comes before the terms (the sky drifts every round).
4. **Seams**: `TryCommand` top, `handlePlayerCombat` after aggro validation, and the four listeners in `internal/hooks/LightNotice_Triggers.go`; why waking uses `NoteAttention` (no single waking seam).
5. **Data**: path, file per cause, `any` or `outdoor`+`indoor`, `MinVariants`, the line rules; `lamp.yaml` is unreachable until plan 5 schedules lamps.
6. **Public surface** table, verified by Step 1's grep, with the grep command and date.
7. **Testing**: an unloaded store is silent (why other packages see nothing), `LoadFrom` + `t.Cleanup(ResetForTest)`, the golden in `internal/narration`.

- [ ] **Step 3: Update the three existing `context.md` files**

- `internal/messaging/context.md`: beside `SightThroughWindow`, add `Band`, `BandThroughWindow`, `LightBand` (band-grained, reads the lighting config, parity-tested against `ParticipantSight`, does not consult sleep). Add `CategoryLight` wherever `CategoryTimeOfDay`'s treatment is described.
- `internal/rooms/context.md`: after the `IsLit` paragraph, add `Room.LightTerms() LightTerms` and that it shares `composeLight` with `LightLevel`. Update the `lighting.go` row of the file table.
- `internal/narration/context.md`: add the light-notices store to its list of stores and goldens.

- [ ] **Step 4: `docs/README.md`**

This plan's row already sits after the 3d spec row (added with the plan). Append to it any defect or deviation execution found, in the same style as the completed plan rows around it.

- [ ] **Step 5: Audit**

Run: `python tools/context_md_audit.py`
Expected: no new finding for `internal/lightnotice`, `internal/messaging`, `internal/rooms`.

- [ ] **Step 6: Commit**

```bash
git add internal/lightnotice/context.md internal/messaging/context.md internal/rooms/context.md internal/narration/context.md docs/README.md
git commit -F - <<'EOF'
docs(lighting): 3d context.md for lightnotice, messaging, rooms, narration

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 13: Full verification

Load `dogmud-shipping` before this task and follow its pre-push gate order where it differs from the steps below.

- [ ] **Step 1: Full suite**

Run: `go test ./... -count=1` (run in the background; `internal/combat` is slow).
Expected: all `ok`. Report any failure with its output; do not re-run hoping for green.

- [ ] **Step 2: No lighting golden moved**

Run: `git diff --stat master -- '*.golden'`
Expected: only `internal/narration/testdata/stores/light_notices.golden`. Any other golden in the list is a defect in this plan's work (no light value changed); investigate it.

- [ ] **Step 3: Lint and format**

Run: `gofmt -l internal/lightnotice internal/messaging internal/rooms internal/hooks internal/usercommands main.go` (remember the Windows CRLF false positive; confirm any hit against the blob) and `golangci-lint run --new-from-merge-base=origin/master`.
Expected: clean.

- [ ] **Step 4: Boot check with the AI companion enabled**

Follow `dogmud-shipping`'s detached-worktree boot check. Boot once with the AI companion module enabled. Confirm: the log shows no panic from `LoadLightNoticeFiles`; a login, a `look`, and a walk from a lit room into a dark one produce exactly one "You step into darkness."-style line; a companion's perception reads as before (this plan changes no sight API). Kill only the server PID you started (`dogmud-playtesting`).

- [ ] **Step 5: Report**

State plainly what passed and anything that did not. The owner opens the PR decision; follow `dogmud-shipping` (`--repo pruuk/DOGMud` on every `gh` command) only when asked.
