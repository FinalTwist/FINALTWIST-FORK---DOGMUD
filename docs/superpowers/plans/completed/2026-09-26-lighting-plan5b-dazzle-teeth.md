# Lighting plan 5b: dazzle's teeth, Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every opposed or difficulty roll pays a sight penalty that ramps with the distance from the roller's own comfort band, capped at 0.80 on the dark and the bright side, except the voice contests; shops refuse below faces.

**Architecture:** `messaging.ComfortDistance` turns a room and an observer into two fractions (dark, bright); `messaging.SightScoreMultiplier` turns them into a multiplier with two cap knobs, and `messaging.SightMult` composes the two. The character-aware channel funnel applies it through the situational table (a sight row per channel, social excluded) and a new defence-side twin; every score-only site applies `SightMult` to the party who needs to see; a repo-root guard requires each roll site to do one or the other, or to be exempt with a reason.

**Tech Stack:** Go (DOGMud fork of GoMud), YAML content under `_datafiles/world/dogmud`, `_datafiles/config.yaml` (skip-worktree in the main checkout).

**Spec:** `docs/superpowers/specs/completed/2026-09-26-lighting-plan5b-dazzle-teeth-design.md`. Read it first. The spec was amended 2026-09-26 to put both helpers in `messaging` (fact 19 below says why).

---

## Before you start

- Work in a worktree on `feature/lighting-plan5b-dazzle-teeth`, branched from `docs/lighting-plan5b-spec` so the spec and plan ship in the PR. A fresh worktree has the plain `HEAD` `config.yaml`, which Task 1 edits.
- `-race` cannot run here; plain `go test`. Never `git add -A`. Edit YAML and Go with the Edit tool (never `sed -i`, which rewrites a CRLF file to LF wholesale). `grep -c` exits 1 on zero matches: run "expect zero" checks alone.
- Every commit must build and pass the packages it touches. Melee's numbers move only where the ramp differs from the old band; Task 3 records the melee census before and after.

## Facts verified against source (2026-09-26, master `49bacf2d7`)

| # | Fact | Where |
|---|---|---|
| 1 | `DarknessScoreMultiplier(sight messaging.SightDecision, bal) float64`: full 1.0, shapes `DarknessShapesCombatPenalty`, else `DarknessCombatPenalty`. Applied at `attackScore *= ...(ctx.sourceSight, bal)` and `defenseScore *= ...(ctx.targetSight, bal)`. | `internal/combat/combat_helpers.go:517,580,772` |
| 2 | `combatContext{ sourceSight, targetSight messaging.SightDecision; ... }` set from `messaging.ParticipantSight(x, room)` at four sites. **The two fields also drive narration**: `combat.go:721,728` hide names by them, and about forty test files build `combatContext{sourceSight: messaging.SightFull, ...}` literals. They STAY; the ramp adds fields beside them. | `combat_helpers.go:35-42`, `combat.go:55,106,150,199,721,728`, grep |
| 3 | Knobs: `DarknessCombatPenalty` (`config.balance.go:301`), `DarknessShapesCombatPenalty` (`:313`), validated as a pair in `validateCombat` (`config.balance.combat.go:335-342`), shipped at `config.yaml:913-928` (0.80, 0.90). Three comment references in `config.balance.go:1079`, `config.balance.lighting.go:14,41,122`. | source |
| 4 | `windowDazzleEdge = 75`, `windowShiftCap = 24`, `windowFloor = 1` are constants; `BandThroughWindow(light, strength, reach, blindBelow, dimBelow) Band` reads the edge; `LightTrimTarget(strength int) float64 = windowDazzleEdge - clampShift(strength) - 1`. | `internal/messaging/window.go:14-24,78-85`, `band.go:46` |
| 5 | `LightBand(observer, room RoomVisibility) Band`: nil → faces; `observer.Perception.State() == perception.Blinded` → dark; else `BandThroughWindow(room.LightLevel(), NightVisionStrength(), InfraReach(), cfg.BlindBelow, cfg.DimBelow)`. `RoomVisibility` is the interface `LightLevel() int`; `*rooms.Room` (`rooms/lighting.go:29`) and the test type `litRoom` (`rooms/rooms.go:324`, light 60) satisfy it. | `band.go:62-83`, `predicates.go:21` |
| 6 | `configs.Lighting` accessor has `BlindBelow`, `DimBelow`, `ExitsAbove`, `DefaultVisionStrength`, the celestial knobs and the six `Spell*` knobs; no dazzle edge. `GetLightingConfig()` (`:32`) takes no argument; there is no `From(Balance)` variant. Test seam: `configs.SetConfigForTest(t, c Config)` (`testing_support.go:30`). | `internal/configs/config.lighting_accessor.go:11-32` |
| 7 | `TrimLightFor` calls `messaging.LightTrimTarget(c.NightVisionStrength())` (5a). | `internal/rooms/light_trim.go` |
| 8 | `SituationalAttackMult(attacker, shape combatvocab.Attack) float64`: prone and stamina for `AttackMelee/Ranged/Thrown`; a declared table in its doc (`situational.go:11-31`). 16 production callers build `AttackSide.Mult` with it: `actions/combat_{bash,drain×2,fire,gore,hamstring,kick,maul,pounce,rake,taunt,throttle,trip}.go`, `hooks/combat_shared_helpers.go:348,407`, `hooks/spell_resolution.go:353`. | `internal/combat/situational.go:36-52`, grep |
| 9 | `combatvocab`: `Melee(t)`, `Ranged(t)`, `Thrown(t)`, `Rhetoric(t)`, `Spell(d DamageType, t Targeting)` constructors; types `AttackMelee/Ranged/Thrown/Spell/Rhetoric/None`. | `internal/combatvocab/attack.go:89-97`, `vocab.go:22-25,75` |
| 10 | `ResolveChannelAttack(shape, side AttackSide, attacker, defender) ChannelDefenceResult` → `resolveChannelAttackWithRunner(..., runner)`; the defence loop (`defence_multiplier.go:460-492`) builds `score := defender.GetDefenseScoreFor(d, includeSkill) * defenceEffectiveness(d)` then prone penalties, then `contest.Entry{Name, Score}`. No room in scope. Production callers: `executeCounterTaunt` (`actions/combat_counter.go:158`), `Throw` (`usercommands/throw.go:168`), and the runner-injected path from `hooks/spell_resolution.go`. | `defence_multiplier.go:415-492`, `codegraph_callers` |
| 11 | `RunContest(atkScore, entries)` (`combat/run_contest.go:23`) production callers: `actions/defuse.go:56`, `combat/submission.go:75`, `combat/skill_moves.go:172`, `combat/grapple.go:76`, `actions/plant.go:132,277,357`, `actions/shadow.go:129`, `actions/steal.go:147,366,460`, `actions/sneak.go:52`, `usercommands/skill.skullduggery.shadow.go:135`, `actions/search.go:44` (`spotsHider`), `actions/track.go:79`, `usercommands/go.go:576,595`, `tools/melee_band_census/main.go:32`. Every `actions` site has `room := actor.GetRoom()` in the same function (steal.go:88,204,463; plant.go:58,163,360; shadow.go:108,149; defuse.go:75; track.go:83; sneak.go:68). | `codegraph_callers RunContest`, grep |
| 12 | `CalcDetectionScore(c) = Perception.ValueAdj + Search × SkillWeight`, no light. Callers: `plant.go:340`, `search.go:46`, `shadow.go:166`, `sneak.go:124,149`, `steal.go:443`, `track.go:276`, `go.go:575,594,625`, `skill.skullduggery.shadow.go:138`. `CalcSneakScoreVsObserver(sneaker, observer, roomLit)` (`:60`) already folds per-observer light into the hider. | `internal/actions/skill_helpers.go:60-77`, grep |
| 13 | `AgainstDifficulty` callers: `actions/search.go:96` (`Search`, has `room`), `forager/forage_core.go:114` (`ForageCore(a ForageAttempt) ForageResult`, pure; its one caller `actions/forage.go:112` builds `ForageAttempt{Biome, SearchScore: searchScore, AtNight}` at `:103`), `actions/track.go:344` (`resolveTrailDetail(searchScore)`, called from `Track` which has `room`). | source |
| 14 | `RunCraftContest`/`RunSalvageContest` are called BARE inside `crafting/salvage.go:42,67` (`RollSalvageReturns`, `RollSalvageReturnsFromSpec`, both taking `score`); the score arrives from `actions/salvage.go:178,350,352`. Craft: `hooks/NewRound_UserRoundTick.go:591`, `hooks/NewRound_MobRoundTick.go:578` (both hold `craftScore` and a room), `mobs/crafter.go:533,597` (`executeCraft(mob, recipe, shopInv)`, `executeCraftLegacy(mob, recipe)`, reached only via `TickMobCraft(mob)` from `hooks/MobIdle_HandleIdleMobs.go:103`). `RunConcentrationContest` callers: `actions/combat_throttle.go:168` (`hold, grip`), `hooks/combat_shared_helpers.go:146,614` (`concentrationScore(ch)`). | grep |
| 15 | `contest_floor_guard_test.go` (repo root): `guardedRollFuncs map[pkg]map[func]bool` (line 30), `guardedRollExemptions map[pkg]map[fileOrDir]reason` (line 54), `TestOpposedContestsAreFloored` (line 155) walks production Go files with `go/ast`, matches `pkg.Sel` SELECTOR calls only (a bare in-package call is invisible to it), and fails any file not exempt. | source |
| 16 | Shops: `List` (`usercommands/list.go:26`) gated by `actions.ShopClosedForSleep(room)` at `:31`; entry points `actions.Buy(buyer Actor, opts)` (`buy.go:289`) and `actions.Sell(seller Actor, opts)` (`sell.go:55`); bartering `barterSkill := char.GetSkillLevel(skills.Bartering)` feeds `discount := float64(barterSkill)/50.0*0.15` at `buy.go:533-537` and `:556-558`, and `bonus` at `sell.go:283-289`. | source |
| 17 | `blindCombatNoticeText` and the verbosity membership read `CanSeeSightImpairedOnly` (non-full sight), not the multiplier. Dazzle has its own 3d notices. | `internal/hooks/combat_verbosity.go:414-460` |
| 18 | Vision grants: `conditions/29-night_vision.yaml` (strength 18), `65-cats_eye_draught.yaml` (24, `secret: false`), `85-infraredvision.yaml` (12, secret). | source |
| 19 | Import graph (`go list -deps`): `messaging` depends on `characters`, `conditions`, `configs`, `mutations`, `state`, `perception`, `combatvocab` and transitively `crafting` (through `characters`), and NOT on `mobs`, `rooms`, `combat` or `actions`. `combat` imports `mobs` and `messaging`. `mobs` depends on none of `messaging`, `rooms`, `combat`. `crafting` depends on neither `messaging` nor `characters`. So: `mobs` MAY import `messaging`; `crafting` may NOT (cycle through `characters`); `combat` and `actions` may not host a helper `mobs` needs. Both helpers therefore live in `messaging`. | `go list -deps` |
| 20 | No test file in `combat`, `actions` or `messaging` defines a `near` float helper. | grep |

## File structure

| File | Responsibility |
|---|---|
| `internal/configs/config.balance.go`, `config.balance.combat.go`, `config.balance.lighting.go`, `config.lighting_accessor.go`, `_datafiles/config.yaml` | `DazzleCap`, `LightDazzleAbove`; retire `DarknessShapesCombatPenalty` |
| `internal/messaging/window.go`, `band.go`, new `comfort.go`, new `sight_mult.go` | the edge as a parameter; `ComfortDistance`; `SightScoreMultiplier`; `SightMult` |
| `internal/combat/combat_helpers.go`, `combat.go` | melee onto the ramp |
| `internal/combat/situational.go`, `defence_multiplier.go` | sight row; `SituationalDefenceMult`; `ResolveChannelAttack(room, ...)` |
| `internal/actions/skill_helpers.go` and each score-only site; `hooks` round ticks; `mobs/crafter.go` + `hooks/MobIdle_HandleIdleMobs.go` | `CalcDetectionScore(c, room)`; `SightMult` at every actor-side score |
| `internal/actions/shop_sight.go` (new), `usercommands/list.go`, `actions/buy.go`, `sell.go` | the shop gate and the dazzled discount |
| `sight_penalty_guard_test.go` (repo root) | the guard |
| conditions 29/65, help `light`/`hood`/`chrysalis-glow`, `docs/PATCH_NOTES.md`, `context.md` ×6 | text |

---

### Task 1: The knobs

**Files:**
- Modify: `internal/configs/config.balance.go:301-313`, `config.balance.combat.go:335-342`, `config.balance.lighting.go` (`validateLighting`), `config.lighting_accessor.go`, `_datafiles/config.yaml:913-928`, `internal/combat/combat_helpers.go:517-525` (a shim only)
- Test: `internal/configs/sight_knobs_test.go`

- [ ] **Step 1: Write the failing test**

```go
package configs

import "testing"

func TestSightKnobDefaultsAndValidation(t *testing.T) {
	var b Balance
	b.Validate()
	if b.DarknessCombatPenalty != 0.80 || b.DazzleCap != 0.80 || b.LightDazzleAbove != 75 {
		t.Fatalf("defaults: dark %v dazzle %v edge %v", b.DarknessCombatPenalty, b.DazzleCap, b.LightDazzleAbove)
	}
	b.DazzleCap = 1.5
	b.LightDazzleAbove = 40 // not above LightDimBelow (50)
	b.Validate()
	if b.DazzleCap != 0.80 || b.LightDazzleAbove != 75 {
		t.Errorf("out of range did not revert: dazzle %v edge %v", b.DazzleCap, b.LightDazzleAbove)
	}
	b.DazzleCap = 0.6
	b.LightDazzleAbove = 80
	b.Validate()
	if b.DazzleCap != 0.6 || b.LightDazzleAbove != 80 {
		t.Errorf("legal values were changed: dazzle %v edge %v", b.DazzleCap, b.LightDazzleAbove)
	}
	cfg := GetConfig()
	cfg.Balance = b
	SetConfigForTest(t, cfg)
	if got := GetLightingConfig().DazzleAbove; got != 80 {
		t.Errorf("accessor DazzleAbove = %d, want 80", got)
	}
}
```
(`Balance.Validate` may be named differently; read `config.balance.go` for the validator entry point and use it.)

- [ ] **Step 2: Run and watch it fail.** `go test ./internal/configs/ -run SightKnob`. Expected: `b.DazzleCap undefined`.

- [ ] **Step 3: Declare.** In `config.balance.go`, delete the `DarknessShapesCombatPenalty` field and its doc block (`:303-313`); rewrite `DarknessCombatPenalty`'s comment:

```go
	// DarknessCombatPenalty is the multiplier at and below the observer's BLIND
	// edge: the dark-side CAP of the sight ramp (lighting plan 5b). Between the
	// dim edge and the blind edge the penalty ramps linearly from 1.0 to this.
	// It prices EVERY opposed or difficulty roll, not only combat; the name is
	// kept for config compatibility.
	DarknessCombatPenalty ConfigFloat `yaml:"DarknessCombatPenalty"` // default 0.80
	// DazzleCap is the multiplier one ramp-width above the observer's dazzle
	// edge and beyond: the bright-side cap. The bright ramp is as wide as the
	// dark one (LightDimBelow - LightBlindBelow), so for normal eyes it reaches
	// the cap at the top of the scale.
	DazzleCap ConfigFloat `yaml:"DazzleCap"` // default 0.80
```
Beside `LightDimBelow` add:
```go
	// LightDazzleAbove is where the comfortable band ends and too-bright
	// begins for a normal observer; a vision ability moves it down by its
	// strength. It was the constant windowDazzleEdge until plan 5b gave dazzle
	// a mechanical cost. Must sit above LightDimBelow and at most 100.
	LightDazzleAbove ConfigInt `yaml:"LightDazzleAbove"` // default 75
```

- [ ] **Step 4: Validate.** In `validateCombat` replace lines 335-342 with:
```go
	if b.DarknessCombatPenalty <= 0 || b.DarknessCombatPenalty > 1.0 {
		b.DarknessCombatPenalty = 0.80
	}
	if b.DazzleCap <= 0 || b.DazzleCap > 1.0 {
		b.DazzleCap = 0.80
	}
```
In `validateLighting`, after the blind/dim pair block:
```go
	// LightDazzleAbove: must sit above the dim edge and within the scale. Zero
	// means unset (a test binary never loads config.yaml).
	if b.LightDazzleAbove <= b.LightDimBelow || b.LightDazzleAbove > 100 {
		b.LightDazzleAbove = 75
	}
```
Fix the three comments that cite the retired knob (`config.balance.go:1079`, `config.balance.lighting.go:14,41,122`): they describe pair validation as a precedent; point them at the `LightBlindBelow/LightDimBelow` pair instead.

- [ ] **Step 5: Accessor.** Add `DazzleAbove int` to `configs.Lighting` after `DimBelow` and `DazzleAbove: int(b.LightDazzleAbove),` in `GetLightingConfig`.

- [ ] **Step 6: config.yaml.** Replace lines 917-928 (the `DarknessShapesCombatPenalty` comment and key) with:
```yaml
  # DazzleCap: the same cap on the BRIGHT side. Plan 5b made the penalty a
  #   ramp: from 1.0 at the edge of your comfortable band to DarknessCombatPenalty
  #   at the blind edge, and to DazzleCap one ramp-width above the dazzle edge.
  #   Both price every opposed or difficulty roll. (0, 1]; zero reverts to 0.80.
  DazzleCap: 0.80
```
and update the DARKNESS block comment above `DarknessCombatPenalty` to say "cap of the sight ramp". Add `LightDazzleAbove: 75` beside the `LightSpell*` block with a two-line comment.

- [ ] **Step 7: Keep the build green.** `DarknessScoreMultiplier`'s shapes case (`combat_helpers.go:522`) reads the deleted field. Make it return the ramp's midpoint, `1.0 - (1.0-float64(bal.DarknessCombatPenalty))/2`, with the comment `// Task 3 of plan 5b deletes this function.` Then `go build ./... && go test ./internal/configs/ ./internal/combat/`. `darkness_penalty_verdict_test.go` may pin the shapes value at 0.90; the midpoint IS 0.90 at the shipped cap, so it should pass; if it reads the deleted knob, point it at the midpoint expression.

- [ ] **Step 8: Commit.** `git add internal/configs/config.balance.go internal/configs/config.balance.combat.go internal/configs/config.balance.lighting.go internal/configs/config.lighting_accessor.go internal/configs/sight_knobs_test.go internal/combat/combat_helpers.go _datafiles/config.yaml`, message `feat(configs): DazzleCap and LightDazzleAbove; retire DarknessShapesCombatPenalty`.

---

### Task 2: `ComfortDistance`, `SightScoreMultiplier`, `SightMult`; the dazzle edge as a parameter

**Files:**
- Create: `internal/messaging/comfort.go`, `comfort_test.go`, `sight_mult.go`, `sight_mult_test.go`
- Modify: `internal/messaging/window.go`, `band.go`, `window_trim_target_test.go`, `internal/rooms/light_trim.go`, plus any other `BandThroughWindow` caller `go build` names

- [ ] **Step 1: Write the failing tests**

`comfort_test.go`:
```go
package messaging

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// Normal eyes: blind 25, dim 50, dazzle 75, width 25.
func TestComfortDistanceNormalEyes(t *testing.T) {
	cases := []struct {
		light        int
		dark, bright float64
	}{
		{60, 0, 0}, {50, 0, 0}, {74, 0, 0},
		{37, 0.52, 0}, {26, 0.96, 0}, {25, 1, 0}, {0, 1, 0},
		{75, 0, 0}, {90, 0, 0.6}, {100, 0, 1},
	}
	for _, c := range cases {
		d, b := comfortDistance(c.light, 0, 25, 50, 75)
		if !near(d, c.dark) || !near(b, c.bright) {
			t.Errorf("light %d: (%v, %v), want (%v, %v)", c.light, d, b, c.dark, c.bright)
		}
	}
}

// Nightvision 24: edges 1, 26, 51; the bright ramp is still 25 wide, so the
// cap arrives at 76, well inside daylight.
func TestComfortDistanceShiftedWindow(t *testing.T) {
	cases := []struct {
		light        int
		dark, bright float64
	}{
		{30, 0, 0}, {20, 0.24, 0}, {1, 1, 0},
		{70, 0, 0.76}, {73, 0, 0.88}, {76, 0, 1}, {90, 0, 1},
	}
	for _, c := range cases {
		d, b := comfortDistance(c.light, 24, 25, 50, 75)
		if !near(d, c.dark) || !near(b, c.bright) {
			t.Errorf("light %d: (%v, %v), want (%v, %v)", c.light, d, b, c.dark, c.bright)
		}
	}
}

func TestComfortDistanceStrengthIsClamped(t *testing.T) {
	d1, b1 := comfortDistance(70, 40, 25, 50, 75)
	d2, b2 := comfortDistance(70, 24, 25, 50, 75)
	if d1 != d2 || b1 != b2 {
		t.Error("a strength above the cap must clamp exactly as the window does")
	}
}
```

`sight_mult_test.go`:
```go
package messaging

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
)

type fixedLight int

func (f fixedLight) LightLevel() int { return int(f) }

func TestSightScoreMultiplierWorkedValues(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.Validate() // dark 0.80, dazzle 0.80
	bal := cfg.Balance
	cases := []struct {
		dark, bright, want float64
	}{
		{0, 0, 1.0},
		{0.52, 0, 0.896}, // normal eyes at 37
		{0.96, 0, 0.808}, // normal eyes at 26
		{1, 0, 0.80},
		{0, 0.6, 0.88},   // normal eyes at 90
		{0, 0.76, 0.848}, // nightvision 24 at 70
		{0, 0.88, 0.824}, // Cat's Eye at 73
		{0, 1, 0.80},
	}
	for _, c := range cases {
		if got := SightScoreMultiplier(c.dark, c.bright, bal); !near(got, c.want) {
			t.Errorf("(%v, %v) = %v, want %v", c.dark, c.bright, got, c.want)
		}
	}
	bal.DazzleCap = 0.5
	if got := SightScoreMultiplier(0, 1, bal); got != 0.5 {
		t.Errorf("DazzleCap not read: %v", got)
	}
}

func TestSightMultFollowsTheRamp(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.Validate()
	configs.SetConfigForTest(t, cfg)
	c := characters.New()
	for _, tc := range []struct {
		light int
		want  float64
	}{{60, 1.0}, {37, 0.896}, {90, 0.88}, {0, 0.80}} {
		if got := SightMult(c, fixedLight(tc.light)); !near(got, tc.want) {
			t.Errorf("light %d: %v, want %v", tc.light, got, tc.want)
		}
	}
	if got := SightMult(c, nil); got != 1.0 {
		t.Error("nil room must be unity")
	}
	if got := SightMult(nil, fixedLight(0)); got != 1.0 {
		t.Error("nil observer must be unity")
	}
}
```
(If `characters.New()` needs arguments or seeded registries, copy what `band_test.go` does to build an observer.) In `window_trim_target_test.go`, change `LightTrimTarget(c.strength)` to `LightTrimTarget(c.strength, 75)` and `BandThroughWindow(..., 25, 50)` to `BandThroughWindow(..., 25, 50, 75)`.

- [ ] **Step 2: Run and watch it fail.** `go test ./internal/messaging/ -run 'Comfort|Sight'`.

- [ ] **Step 3: Delete the constant and thread the edge.** In `window.go` delete `windowDazzleEdge`; change `BandThroughWindow(light, strength, reach, blindBelow, dimBelow int)` to `BandThroughWindow(light, strength, reach, blindBelow, dimBelow, dazzleAbove int)` and use `dazzleAbove` where the constant was; `LightTrimTarget(strength, dazzleAbove int) float64 { return float64(dazzleAbove - clampShift(strength) - 1) }`. Update the file's header comment (the edge is a knob now). `go build ./...` enumerates the callers: `band.go` (`LightBand` passes `cfg.DazzleAbove`), `rooms/light_trim.go` (`messaging.LightTrimTarget(c.NightVisionStrength(), cfg.DazzleAbove)`), tests.

- [ ] **Step 4: Create `comfort.go`.**

```go
package messaging

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// ComfortDistance reports how far a room's light sits outside the observer's
// own comfortable band, as two fractions of the way to the cap: dark (below
// the dim edge, 1 at the blind edge) and bright (above the dazzle edge, 1 one
// ramp-width beyond it). At most one is non-zero. It is the geometry behind
// the sight penalty (lighting plan 5b); SightScoreMultiplier turns it into a
// number. A nil observer or room is comfortable; a Blinded observer is fully
// dark.
func ComfortDistance(observer *characters.Character, room RoomVisibility) (dark, bright float64) {
	if observer == nil || room == nil {
		return 0, 0
	}
	if observer.Perception != nil && observer.Perception.State() == perception.Blinded {
		return 1, 0
	}
	cfg := configs.GetLightingConfig()
	return comfortDistance(room.LightLevel(), observer.NightVisionStrength(), cfg.BlindBelow, cfg.DimBelow, cfg.DazzleAbove)
}

// comfortDistance is the pure form. The bright ramp is as wide as the dark
// one, so a strong window is punished by excess light as fast as it is helped
// by faint light; infra reach does not soften either side.
func comfortDistance(light, strength, blindBelow, dimBelow, dazzleAbove int) (dark, bright float64) {
	s := clampShift(strength)
	blind, dim, dazzle := blindBelow-s, dimBelow-s, dazzleAbove-s
	width := float64(dim - blind)
	if width <= 0 {
		width = 1
	}
	switch {
	case light < dim:
		dark = float64(dim-light) / width
		if dark > 1 {
			dark = 1
		}
	case light >= dazzle:
		bright = float64(light-dazzle) / width
		if bright > 1 {
			bright = 1
		}
	}
	return dark, bright
}
```
`nil` check on `room`: `RoomVisibility` is an interface, so also guard a typed nil `*rooms.Room` the way `LightBand` does (read `band.go:68-75` and copy its nil handling exactly).

- [ ] **Step 5: Create `sight_mult.go`.**

```go
package messaging

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
)

// SightScoreMultiplier is the ONE place the sight ramp turns into a number
// (lighting plan 5b). dark and bright are ComfortDistance's fractions; the
// multiplier runs from 1.0 at the edge of the comfortable band to
// Balance.DarknessCombatPenalty at the blind edge and to Balance.DazzleCap one
// ramp-width above the dazzle edge. It prices every opposed or difficulty
// roll, not only combat; the voice contests are exempt by never asking.
//
// It lives here, not in combat, because mobs (the shopkeeper crafter) and the
// crafting round ticks need the same body and sit below combat in the import
// graph.
func SightScoreMultiplier(dark, bright float64, bal configs.Balance) float64 {
	mult := 1.0 - dark*(1.0-float64(bal.DarknessCombatPenalty)) - bright*(1.0-float64(bal.DazzleCap))
	if mult < 0 {
		return 0
	}
	return mult
}

// SightMult is the sight ramp for one roller in one room: apply it to the
// score of whoever needs to SEE for the roll. In a detection roll that is the
// observer; in a theft the thief; in a search, track, defuse, forage, craft or
// concentration roll the actor. The voice contests never call it. It composes
// ComfortDistance and SightScoreMultiplier so no site can pair them
// differently.
func SightMult(c *characters.Character, room RoomVisibility) float64 {
	dark, bright := ComfortDistance(c, room)
	return SightScoreMultiplier(dark, bright, configs.GetBalanceConfig())
}
```

- [ ] **Step 6: Run.** `go build ./... && go test ./internal/messaging/ ./internal/rooms/ ./internal/lightnotice/ .` (the goldens sample light, not rolls: unmoved).

- [ ] **Step 7: Commit.** Named paths; message `feat(messaging): ComfortDistance, SightScoreMultiplier, SightMult; the dazzle edge is a knob`.

---

### Task 3: Melee onto the ramp

**Files:**
- Modify: `internal/combat/combat_helpers.go` (delete `DarknessScoreMultiplier` and its shim; `combatContext`; lines 580 and 772), `internal/combat/combat.go:55,106,150,199`, `internal/combat/darkness_penalty_verdict_test.go`

- [ ] **Step 1: Record the melee census BEFORE.** `go run ./tools/melee_band_census > "$SCRATCH/census-before.txt"` (read the tool's usage first; keep its output in the scratchpad).

- [ ] **Step 2: Write the failing test.** In `darkness_penalty_verdict_test.go`, replace the three-verdict test at `:209-211` with one that builds `combatContext{sourceSight: messaging.SightFull, sourceDark: 0.52}` and expects `calcAttackScore` to be the clean score × 0.896, and `sourceBright: 0.6` → × 0.88 (read the file's existing fixture for `observer`, `target`; keep its other tests, which pin that `sourceSight` still carries the verdict). Run it: `sourceDark` undefined.

- [ ] **Step 3: Add the comfort fields.** In `combatContext`, KEEP `sourceSight`/`targetSight` (they drive name hiding at `combat.go:721,728`) and add after them:

```go
	// sourceDark/sourceBright and targetDark/targetBright are
	// messaging.ComfortDistance for each side: the sight RAMP's input (plan
	// 5b). The verdict fields above stay the narration gate; these two pairs
	// are the only thing the scoring reads.
	sourceDark, sourceBright float64
	targetDark, targetBright float64
```
At `combat.go:55,106,150,199` keep the two `ParticipantSight` lines and add four fields from `messaging.ComfortDistance(X, room)` / `(Y, room)` computed just above each literal. Delete `DarknessScoreMultiplier`; at `:580` write `attackScore *= messaging.SightScoreMultiplier(ctx.sourceDark, ctx.sourceBright, bal)` and at `:772` `defenseScore *= messaging.SightScoreMultiplier(ctx.targetDark, ctx.targetBright, bal)`; update both comments (the ramp replaces the band; a shapes-only attacker used to get a flat 0.90). The forty test literals that set only the verdict fields keep compiling and now mean "comfortable", which is what they meant.

- [ ] **Step 4: Run and census AFTER.** `go test ./internal/combat/ ./internal/hooks/ ./internal/actions/`; `go run ./tools/melee_band_census > "$SCRATCH/census-after.txt"`; `diff` the two and put the differing rows in the commit body: they must be only rows whose light sits off the old band midpoints.

- [ ] **Step 5: Commit.** Message `feat(combat): melee attack and defence on the sight ramp`, body with the census diff.

---

### Task 4: The channel funnel: a sight row and a defence side

**Files:**
- Modify: `internal/combat/situational.go`, `situational_test.go`, `defence_multiplier.go` (`ResolveChannelAttack`, `resolveChannelAttackWithRunner`), the 16 `SituationalAttackMult` callers (fact 8), the `ResolveChannelAttack` callers (fact 10)
- Test: `internal/combat/situational_sight_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package combat

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/configs"
)

type fixedLight int

func (f fixedLight) LightLevel() int { return int(f) }

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// A dazzled attacker (light 90, normal eyes) pays 0.88 on melee, ranged,
// thrown and spell, and nothing on rhetoric; a comfortable one pays nothing.
func TestSituationalAttackMultSightRow(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.Validate()
	configs.SetConfigForTest(t, cfg)
	c := characters.New()
	c.Stats.Dexterity.ValueAdj = 100
	c.Stamina, c.StaminaMax.Value = 100, 100
	shapes := map[string]combatvocab.Attack{
		"melee":    combatvocab.Melee(combatvocab.TargetSingle),
		"ranged":   combatvocab.Ranged(combatvocab.TargetSingle),
		"thrown":   combatvocab.Thrown(combatvocab.TargetSingle),
		"spell":    combatvocab.Spell(combatvocab.DamagePhysical, combatvocab.TargetSingle),
		"rhetoric": combatvocab.Rhetoric(combatvocab.TargetSingle),
	}
	for name, shape := range shapes {
		comfortable := SituationalAttackMult(c, fixedLight(60), shape)
		dazzled := SituationalAttackMult(c, fixedLight(90), shape)
		want := 0.88
		if name == "rhetoric" {
			want = 1.0
		}
		if got := dazzled / comfortable; !near(got, want) {
			t.Errorf("%s: dazzled/comfortable = %v, want %v", name, got, want)
		}
	}
}

func TestSituationalDefenceMultSightRow(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.Validate()
	configs.SetConfigForTest(t, cfg)
	d := characters.New()
	for name, shape := range map[string]combatvocab.Attack{
		"melee":    combatvocab.Melee(combatvocab.TargetSingle),
		"spell":    combatvocab.Spell(combatvocab.DamagePhysical, combatvocab.TargetSingle),
		"rhetoric": combatvocab.Rhetoric(combatvocab.TargetSingle),
	} {
		got := SituationalDefenceMult(d, fixedLight(90), shape)
		want := 0.88
		if name == "rhetoric" {
			want = 1.0
		}
		if !near(got, want) {
			t.Errorf("%s: %v, want %v", name, got, want)
		}
	}
	if got := SituationalDefenceMult(nil, fixedLight(90), combatvocab.Melee(combatvocab.TargetSingle)); got != 1.0 {
		t.Error("nil defender must be unity")
	}
}
```
(Stamina field names: read how `situational_test.go` sets a full-stamina attacker and copy it.)

- [ ] **Step 2: Run and watch it fail.** `go test ./internal/combat/ -run SituationalAttackMultSightRow`: `too many arguments`.

- [ ] **Step 3: The attack side.** `SituationalAttackMult(attacker *characters.Character, room messaging.RoomVisibility, shape combatvocab.Attack) float64`; after the existing switch add:

```go
	// Sight (lighting plan 5b): every channel but social. A taunt needs no
	// eyes; a swing, a shot, a throw and a cast all do.
	switch shape.Type {
	case combatvocab.AttackMelee, combatvocab.AttackRanged, combatvocab.AttackThrown, combatvocab.AttackSpell:
		mult *= messaging.SightMult(attacker, room)
	}
```
Add the `sight` row to the declared table in the doc comment: `Y Y Y Y N`. A nil `room` yields unity through `ComfortDistance`.

- [ ] **Step 4: The defence side.** In `situational.go`:

```go
// SituationalDefenceMult composes the defender-side situational multipliers.
// Today it carries one row, sight (lighting plan 5b): a defender must see a
// swing, a shot, a throw or a cast coming, and a defy needs no eyes. Prone
// defence penalties stay per-defence in the channel funnel and melee, where
// they always were.
func SituationalDefenceMult(defender *characters.Character, room messaging.RoomVisibility, shape combatvocab.Attack) float64 {
	if defender == nil {
		return 1.0
	}
	switch shape.Type {
	case combatvocab.AttackMelee, combatvocab.AttackRanged, combatvocab.AttackThrown, combatvocab.AttackSpell:
		return messaging.SightMult(defender, room)
	}
	return 1.0
}
```
Change `ResolveChannelAttack(room messaging.RoomVisibility, shape, side, attacker, defender)` and `resolveChannelAttackWithRunner(room, ...)`; in the defence loop after the prone switch: `score *= SituationalDefenceMult(defender, room, shape)`. `go build ./...` enumerates every caller of both functions and of `SituationalAttackMult`: each already holds a `room` (`*rooms.Room` satisfies `RoomVisibility`); where a special move holds only the character, use `rooms.LoadRoom(char.RoomId)` once, in the same function, and say so in a comment. The spell path (`hooks/spell_resolution.go:353` and its runner-injected `resolveChannelAttackWithRunner` use) passes the room it already has. Test callers of `resolveChannelAttackWithRunner` pass `nil` for the room unless they test sight.

- [ ] **Step 5: Run.** `go test ./internal/combat/ ./internal/actions/ ./internal/hooks/ ./internal/usercommands/ .`; the wire-freeze and counter tests exercise the funnel.

- [ ] **Step 6: Commit.** Message `feat(combat): sight row on the situational table; a defence-side twin; ResolveChannelAttack takes the room`.

---

### Task 5: Score-only sites: `SightMult` on the party who needs to see

**Files:**
- Modify: `internal/actions/skill_helpers.go` (`CalcDetectionScore(c, room)`) and every site in the table below
- Test: `internal/actions/detection_sight_test.go`

- [ ] **Step 1: Write the failing test**

```go
package actions

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
)

type fixedLight int

func (f fixedLight) LightLevel() int { return int(f) }

// The observer's eyes count in detection; the hider's already did.
func TestDetectionScoreTakesTheObserversEyes(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.Validate()
	configs.SetConfigForTest(t, cfg)
	o := characters.New()
	o.Stats.Perception.ValueAdj = 100
	comfortable := CalcDetectionScore(o, fixedLight(60))
	dazzled := CalcDetectionScore(o, fixedLight(90))
	if math.Abs(dazzled/comfortable-0.88) > 1e-9 {
		t.Errorf("dazzled/comfortable = %v, want 0.88", dazzled/comfortable)
	}
}
```

- [ ] **Step 2: Run and watch it fail.**

- [ ] **Step 3: `CalcDetectionScore(c *characters.Character, room messaging.RoomVisibility) float64`** multiplies its result by `messaging.SightMult(c, room)`; doc comment says whose eyes. `go build` enumerates the eleven callers (fact 12): each passes the room it holds (`room`, `destRoom`, `rooms.LoadRoom(target.Character.RoomId)`).

- [ ] **Step 4: The actor-side sites.** At each, multiply the actor's score by `messaging.SightMult(actor, room)` once, with a one-line comment `// sight ramp (plan 5b): the <thief|searcher|...> needs to see`. Every `actions` function below already has `room := actor.GetRoom()` (fact 11).

| Site | Score to multiply |
|---|---|
| `steal.go:147,366,460` | the thief's attack score |
| `plant.go:132,277,357` | the planter's attack score |
| `shadow.go:129`, `usercommands/skill.skullduggery.shadow.go:135` | the shadower's score |
| `defuse.go:56` | the defuser's score |
| `search.go:96` (static tiers) | the searcher's score handed to `AgainstDifficulty` |
| `track.go:79` and the `searchScore` passed into `resolveTrailDetail` from `Track` | the tracker's score |
| `forage.go:103` | `SearchScore: searchScore * messaging.SightMult(char, room)` (`ForageCore` stays pure) |
| `salvage.go:178,350,352` | `score` (the salvager) before the `crafting.RollSalvageReturns*` calls; multiply once where `score` is computed if both sites share it |
| `hooks/NewRound_UserRoundTick.go:591`, `hooks/NewRound_MobRoundTick.go:578` | `craftScore` (the crafter; both hold a room) |
| `mobs/crafter.go:533,597` | `craftScore`; `TickMobCraft(mob *Mob, room messaging.RoomVisibility)` threads the room into `executeCraft` and `executeCraftLegacy`; `hooks/MobIdle_HandleIdleMobs.go:103` passes the room it holds (`mobs` may import `messaging`, fact 19) |
| `combat/submission.go:75`, `combat/grapple.go:76` | attacker AND defender scores (both must see); both take a room parameter from their callers, who hold one |
| `hooks/combat_shared_helpers.go:146,614`, `actions/combat_throttle.go:168` | the caster's concentration score (`hold` in throttle) |
| `combat/skill_moves.go:172` | already covered: its callers fold `SituationalAttackMult` (Task 4) into the params; confirm by reading, and note it in the guard's exemption table |

- [ ] **Step 5: Run.** `go build ./... && go test ./internal/actions/ ./internal/combat/ ./internal/hooks/ ./internal/usercommands/ ./internal/crafting/ ./internal/mobs/ ./internal/forager/`.

- [ ] **Step 6: Commit.** Message `feat(actions): SightMult on every score-only roll, on the party who needs to see`.

---

### Task 6: Shops

**Files:**
- Create: `internal/actions/shop_sight.go`, `internal/usercommands/list_sight_test.go`
- Modify: `internal/usercommands/list.go:31`, `internal/actions/buy.go:289` (top of `Buy`) and `:533-537,556-558`, `internal/actions/sell.go:55` (top of `Sell`) and `:283-289`

- [ ] **Step 1: Write the failing test** (usercommands fixture: `seedAllRegistries()`, user 1, `rooms.LoadRoom(2)` with `room.Biome = "cave"` for dark and a lit biome for faces; a merchant mob in the room: read the existing `list`/`buy` tests for how a shop is seeded): `List` in a dark room prints "You can't make out the goods well enough to deal." and lists nothing; in a lit room it lists.

- [ ] **Step 2: `shop_sight.go`.**

```go
// ShopSightRefusal is the sight gate on dealing (lighting plan 5b): below the
// faces band you cannot make out the goods, so list, buy and sell refuse. It
// mirrors ShopClosedForSleep beside it. Above faces the deal goes through and
// the bartering discount takes SightMult, so a dazzled haggler bargains worse.
func ShopSightRefusal(c *characters.Character, room *rooms.Room) bool {
	return messaging.LightBand(c, room) < messaging.BandFaces
}
```
In `list.go` after the sleep gate: `if actions.ShopSightRefusal(user.Character, room) { user.SendText(messaging.CategorySystem, "You can't make out the goods well enough to deal."); return true, nil }`. Same gate at the top of `Buy` and `Sell` (return the result type's refusal form with the same reason; read how `ShopClosedForSleep` is surfaced there and mirror it). At `buy.go:533-537`, `:556-558` and `sell.go:283-289` multiply the barter discount by `messaging.SightMult(char, room)` before it is applied.

- [ ] **Step 3: Run, commit.** `go test ./internal/usercommands/ ./internal/actions/ ./internal/shops/`; message `feat(shops): list, buy and sell need faces; a dazzled haggler bargains worse`.

---

### Task 7: The guard

**Files:**
- Create: `sight_penalty_guard_test.go` (repo root), modelled on `contest_floor_guard_test.go`

- [ ] **Step 1: Write it.** Walk production Go files with `go/ast` exactly as `TestOpposedContestsAreFloored` does. For every SELECTOR call in `guardedSightFuncs`:

```go
var guardedSightFuncs = map[string]map[string]bool{
	"combat":   {"RunContest": true, "ResolveChannelAttack": true, "RunConcentrationContest": true},
	"contest":  {"AgainstDifficulty": true},
	"crafting": {"RunCraftContest": true, "RunSalvageContest": true, "RollSalvageReturns": true, "RollSalvageReturnsFromSpec": true},
	"forager":  {"ForageCore": true},
}
```
find the enclosing `*ast.FuncDecl` and require that its body contains a call whose selector name is one of `SightMult`, `SightScoreMultiplier`, `ComfortDistance`, `SituationalAttackMult`, `SituationalDefenceMult` or `CalcDetectionScore`, OR that `file|func` appears in `sightExemptSites`:

```go
var sightExemptSites = map[string]string{
	"internal/combat/skill_moves.go|executeSkillMoveWithRunner": "callers fold SituationalAttackMult into SkillMoveParams; verified by reading",
	"internal/actions/combat_counter.go|executeCounterTaunt":    "a defy counter-taunt is voice: exempt by ruling (2026-09-26)",
	"internal/combat/run_contest.go|RunContest":                 "defines the funnel",
	"tools/melee_band_census/main.go|main":                       "an analysis tool, not the game",
}
```
Fail with a message that names the file, the function, the two ways to comply, and the ruling. Add the sabotage check as a second test: parse a small in-memory Go source containing a `combat.RunContest` call with no helper and assert the checker reports it (factor the checker into a function taking `[]*ast.File` so this is cheap). Note in the file header that a bare in-package call (`crafting/salvage.go`'s own `RunSalvageContest`) is invisible to a selector walk, which is why `RollSalvageReturns*` are guarded instead.

- [ ] **Step 2: Run.** `go test . -run SightPenalty -v`. Every real site from Tasks 4 to 6 must pass; any it flags is a site the earlier tasks missed: fix the site, do not exempt it. The existing `contest_floor_guard_test.go` must still pass untouched.

- [ ] **Step 3: Commit.** Message `test(guard): every roll site applies the sight ramp or is exempt by ruling`.

---

### Task 8: Text, patch note, context

**Files:**
- Modify: `_datafiles/world/dogmud/conditions/29-night_vision.yaml`, `65-cats_eye_draught.yaml` (descriptions), `templates/help/light.template`, `hood.template`, `chrysalis-glow.template`, `docs/PATCH_NOTES.md`, `context.md` in `configs`, `messaging`, `combat`, `actions`, `mobs`, `rooms`

- [ ] **Step 1: Conditions.** Append one sentence to each description (80 columns, no numbers, no dashes): condition 65: "In daylight the same sharpened eyes are dazzled, and everything you do by sight suffers until dusk or until the draught wears off." Condition 29: "Daylight dazzles these eyes; everything done by sight suffers until dark." Condition 85 is secret: nothing. The narration store golden (`internal/narration/testdata/stores/conditions.golden`) freezes condition TEXT: it will move; explain the diff (two description lines) and re-record with the tool the store test names.

- [ ] **Step 2: Help.** `light.template`: after the dazzle sentence in "Carrying a light", add "Being dazzled costs you: your aim, your guard and your eye for detail all suffer, more the brighter it gets, and the same is true of dim light as it fades toward dark. Talking is unaffected." `hood.template`: one sentence that opening the hood in a cave can dazzle creatures made for the dark, and cost them. `chrysalis-glow.template`: "A fresh glow at full strength can dazzle your own party until you move." Width check: strip tags, no line over 80.

- [ ] **Step 3: Patch note** in `docs/PATCH_NOTES.md`'s format, player terms: light now affects every roll in proportion; too bright hurts as much as too dark; shouting, rallying and taunting are unaffected; shops need light to deal.

- [ ] **Step 4: context.md.** Name every new symbol (`ComfortDistance`, `SightScoreMultiplier`, `SightMult` in `messaging`; `SituationalDefenceMult` and the `ResolveChannelAttack` room parameter in `combat`; `ShopSightRefusal` and `CalcDetectionScore(c, room)` in `actions`; `TickMobCraft(mob, room)` in `mobs`; the two knobs and the retired one in `configs`) in the package that owns it; delete `DarknessScoreMultiplier` and `windowDazzleEdge` wherever named (`combat/context.md:1056-1071` at least). `python tools/context_md_audit.py` must add no phantom.

- [ ] **Step 5: Commit.** Message `docs(lighting): dazzle's cost on the tin; help, patch note, context`.

---

### Task 9: Full gate, playtest, PR

- [ ] **Step 1: Gate.** `gofmt -l internal/ modules/` (nothing), `go build ./... && go vet ./... && go test ./...`, `golangci-lint run --new-from-merge-base=origin/master` (0 issues), goldens: only the conditions store golden moved (Task 8), explained.

- [ ] **Step 2: Boot check** per `dogmud-shipping` in a detached worktree (build to `boot-check.exe`, exit 124 is success, kill by PID).

- [ ] **Step 3: Playtest** (`playtest-scenario`, three actors, `bug-finder`): a Cat's Eye drinker (profile with `grant_items: [30047]`, drink at start) who fights Sable's arena mob (`ask sable arena <gold>`, room 5000) at noon under open sky and again after dusk, reporting hit rates over several rounds and the `blindCombatNoticeText` line; a normal-eyed torch bearer who haggles with Trader Onna (room 5207) at noon (dazzled by a point or two: the discount barely moves) and in a dark room (`list` refused); a third actor who searches and tracks in a dim street at night. Every line read as prose; report anything showing a raw number. Extract findings to memory.

- [ ] **Step 4: PR.** `gh pr create --repo pruuk/DOGMud --base master --head feature/lighting-plan5b-dazzle-teeth`, body listing the rulings honoured, the census diff, the golden move, and the owner hand-checks. The owner merges.
