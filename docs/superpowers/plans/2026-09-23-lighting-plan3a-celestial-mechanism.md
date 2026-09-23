# Lighting Plan 3a: Celestial Mechanism Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `legacyVisibility` with a real light model: sun and three moons derived from `WorldLatitude`, combined with a per-biome sky fraction and lamp value on one logarithmic operator, so that day, night, season and moon phase change what a player can see.

**Architecture:** A new leaf package `internal/lightscale` holds the pure arithmetic of the scale with no config and no globals, the way `SightThroughWindow` did for plan 2. `internal/gametime` gains the celestial term, memoised once per round. `internal/rooms` composes ambient, lamp and mutators into `LightLevel()`. Biome `darkarea`/`litarea` booleans are deleted so the compiler enumerates every consumer, and `configs` gains one small lighting accessor that replaces a 424-field struct copy at fifteen call sites.

**Tech Stack:** Go, `gopkg.in/yaml.v2` via `internal/fileloader`, golden-file tests under `testdata/`.

**Spec:** `docs/superpowers/specs/2026-09-23-graded-room-lighting-amendment-celestial.md`, which amends `2026-09-22-graded-room-lighting-design.md`.

**Scope:** This is the mechanism only. Three sibling plans follow and are NOT in this PR:
- **3b** biome vocabulary (`sewer`, `interior` absorbing `house`, `plains`, `river`) and the 117 orphan rooms
- **3c** the city main/side/indoor pass over 477 rooms
- **3d** transition notices

⚠️ **Do not start 3b or 3c inside this PR.** Memory's standing trap: CI lint goes red on any PR over 300 files because the diff API 406s and kills `only-new-issues`.

---

## Traps that will bite you

Read these before Task 1. Each one has cost this project a debugging session.

🪤 **`NightHours` defaults to 0 in a Go test binary.** A test binary never loads
`_datafiles/config.yaml`; it gets the Go defaults from `Validate()`. `IsNight()`
is therefore ALWAYS false unless you pin `Timing` with `configs.SetConfigForTest`.
Any night test that does not pin it is fake. This applies to **Task 1's baseline
recording** above all: an unpinned baseline records a world with no night at all.

🪤 **`configs.GetBalanceConfig()` copies a 424-field struct under two read locks**
and measured 99.75 ns. Never call it twice in one function or inside a per-entity
loop. This plan adds `GetLightingConfig()` precisely so nothing has to.

🪤 **`grep -c` exits 1 when it finds zero matches**, so an "expect zero" check
breaks an `&&` chain and silently skips everything after it. Run such checks
standalone.

🪤 **`gofmt -l` false-positives on Windows** (CRLF working copy, LF blob). Use
`tr -cd '\r' | wc -c` or `file` to check line endings, never `grep -c $'\r'`.

🪤 **`_datafiles/config.yaml` carries the git skip-worktree bit** and desyncs in
both directions. This plan does NOT touch it. If you think you need to, build the
commit from `git show HEAD:_datafiles/config.yaml`, never from disk.

🪤 **Use `go test ./...`, not `go test .`.**

🪤 **Three goldens are already in play**: `testdata/lighting_parity.golden`,
`internal/narration/testdata/stores/conditions.golden`, and
`internal/hooks/darkness_narration.golden`. This plan adds a fourth and retires
the first one's guarantee.

---

## File Structure

**Created:**

| Path | Responsibility |
|---|---|
| `internal/lightscale/lightscale.go` | Pure scale arithmetic: `Combine`, `Attenuate`, `Absent`. No config, no globals, no locks |
| `internal/lightscale/lightscale_test.go` | Unit tests for the above |
| `internal/lightscale/context.md` | Package doc, required by the project convention |
| `internal/gametime/celestial.go` | Declination, solar altitude, day length, moon light, `CelestialLight()` with its round memo |
| `internal/gametime/celestial_test.go` | Unit tests, all pinning `Timing` |
| `lighting_daycycle_golden_test.go` | The new proof artifact: sight tier per room per sample round |
| `testdata/lighting_daycycle.golden` | Recorded in Task 1 against the UNMODIFIED tree |

**Modified:**

| Path | Change |
|---|---|
| `internal/configs/config.balance.go:1099-1125` | Eight new lighting knobs beside the existing four |
| `internal/configs/config.balance.lighting.go` | Validation for the new knobs |
| `internal/configs/config.lighting_accessor.go` (new) | `Lighting` struct and `GetLightingConfig()` |
| `internal/gametime/gametime.go:196-245` | `ReCalculate` derives night length from latitude; day-of-year moves ahead of the night block |
| `internal/gametime/gametime.go:506-560` | `GetLastPeriod` sunrise/sunset use the same derivation |
| `internal/rooms/biomes.go:14-45` | `DarkArea`/`LitArea` deleted; `SkyLight`/`Lamp` added |
| `internal/rooms/lighting.go` | `legacyVisibility` deleted; `LightLevel` composes the real model; `IsLit` added |
| `internal/rooms/rooms.go:90` | Room-level `SkyLight`/`Lamp` overrides |
| `_datafiles/world/dogmud/biomes/*.yaml` (17 files) | `darkarea`/`litarea` replaced by `skylight`/`lamp` |
| 15 call sites | Hand-rolled lit checks collapsed onto `Room.IsLit()` |

---

### Task 1: Record the day-cycle golden against the unmodified tree

Plan 1's lesson, applied correctly. `testdata/lighting_parity.golden` is about to
move on nearly every room, so it stops being a guard. Its replacement must be
recorded **before** any behaviour changes, or it only proves the new code is
self-consistent with itself.

🔑 The baseline will be almost flat, because today light barely varies by round.
**That flatness is the point**: the diff in Task 10 is then a direct readout of
what the celestial model did.

**Files:**
- Create: `lighting_daycycle_golden_test.go`
- Create: `testdata/lighting_daycycle.golden`

- [ ] **Step 1: Write the golden test**

Model it on the existing `lighting_parity_golden_test.go` (same directory, same
flag idiom). Sample rounds are chosen to hit midwinter/equinox/midsummer at
midnight/dawn/noon/dusk.

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var updateDaycycle = flag.Bool("update-lighting-daycycle", false,
	"re-record testdata/lighting_daycycle.golden")

// sampleRounds picks 12 rounds spanning the year and the day. RoundsPerDay is
// pinned to 900 below, so round = (dayOfYear-1)*900 + hour*37.5.
//
// Recorded BEFORE the celestial model exists, so the baseline shows what the
// old model did at each of these moments. A diff here after Task 8 is the
// readout of the new model, not a regression.
func sampleRounds() []struct {
	Label string
	Round uint64
} {
	type s = struct {
		Label string
		Round uint64
	}
	out := []s{}
	for _, d := range []struct {
		name string
		doy  int
	}{{"midwinter", 356}, {"equinox", 81}, {"midsummer", 172}} {
		for _, h := range []struct {
			name string
			hour float64
		}{{"midnight", 0}, {"dawn", 6}, {"noon", 12}, {"dusk", 18}} {
			r := uint64(float64(d.doy-1)*900 + h.hour*37.5)
			out = append(out, s{Label: d.name + "-" + h.name, Round: r})
		}
	}
	return out
}

func TestLightingDayCycleAcrossSampleRounds(t *testing.T) {
	// TRAP: a test binary loads Go defaults, where NightHours is 0 and
	// IsNight() can never be true. Without this pin the baseline records a
	// world with no night in it at all.
	cfg := configs.GetConfig()
	cfg.Timing.RoundsPerDay = 900
	cfg.Timing.NightHours = 8
	cfg.Timing.RoundSeconds = 4
	cfg.Timing.Validate()
	configs.SetConfigForTest(t, cfg)

	loadWorldForGoldenTest(t)

	originalRound := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(originalRound) })

	ids := rooms.GetAllRoomIds()
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	var b strings.Builder
	for _, s := range sampleRounds() {
		util.SetRoundCount(s.Round)
		fmt.Fprintf(&b, "== %s (round %d)\n", s.Label, s.Round)
		for _, id := range ids {
			r := rooms.LoadRoom(id)
			if r == nil {
				continue
			}
			fmt.Fprintf(&b, "room %d biome=%s light=%d\n", id, r.Biome, r.LightLevel())
		}
	}
	got := b.String()

	const path = "testdata/lighting_daycycle.golden"
	if *updateDaycycle {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("recorded %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v (record it with -update-lighting-daycycle)", err)
	}
	if got != string(want) {
		t.Errorf("day-cycle golden moved. If intended, re-record with:\n" +
			"  go test . -run TestLightingDayCycleAcrossSampleRounds -update-lighting-daycycle -v")
	}
}
```

⚠️ `loadWorldForGoldenTest` and `rooms.GetAllRoomIds` must match whatever
`lighting_parity_golden_test.go` already uses in this package. **Read that file
first and reuse its helper verbatim rather than inventing one.** If it inlines
the world load, inline the same code here.

- [ ] **Step 2: Verify the test fails with no golden present**

Run: `go test . -run TestLightingDayCycleAcrossSampleRounds -v`
Expected: FAIL, "read golden: ... no such file"

- [ ] **Step 3: Record the baseline**

Run: `go test . -run TestLightingDayCycleAcrossSampleRounds -update-lighting-daycycle -v`
Expected: PASS, logs "recorded testdata/lighting_daycycle.golden"

- [ ] **Step 4: Prove the baseline is not fake**

The pin in Step 1 exists because of a known trap, and a baseline recorded
without it describes a world that has no night in it. Prove the pin took effect.

🔴 **CORRECTED 2026-09-23 after this task ran.** The original check counted
`light=0` rooms and expected "more than 100". **That probe cannot work.**
`legacyVisibility` floors a dark-biome room at 0 in daylight too
(`2 − 2 = 0`, and night's extra `−1` only clamps harder to the same floor), so
the `light=0` count is **invariant to the time of day** and proves nothing about
night. Its true value is exactly **97**, which is 140 dark-biome rooms minus the
43 held lit by a static `lightmod: 2` (31 Crash Site Interior, 12 Foldweave) —
the same 97 plan 2's golden landed on. Use the probe below instead.

Run: `grep -c "^== midwinter-midnight" testdata/lighting_daycycle.golden`
Expected: `1`

Run standalone, counting the rooms that sit at `LightRoomOnly`:

```bash
awk '/^== midwinter-midnight/{f=1;next} /^== /{f=0} f' testdata/lighting_daycycle.golden | grep -c "light=60"
awk '/^== equinox-noon/{f=1;next} /^== /{f=0} f' testdata/lighting_daycycle.golden | grep -c "light=60"
```

Expected: **391** at midnight and **0** at noon.

🔑 **Why this probe works where the other did not.** 60 is `LightRoomOnly`,
which `legacyVisibility` reaches only via `2 − 1 = 1`, and that `−1` is applied
**only** by `gametime.IsNight()`. A neutral biome — one that is neither dark nor
lit — therefore reads 70 by day and 60 by night, and nothing else in the old
model can produce 60. So a non-zero count at midnight is a direct assertion that
`IsNight()` was genuinely true, which is exactly what the pin buys.

391 is independently checkable: it is the count of neutral-biome rooms, farmland
100 + cliffs 97 + water 91 + forest 61 + mountains 27 + shore 14 + desert 1.

If the midnight count is **0**, `NightHours` did not take and the baseline is
worthless — fix the pin and re-record. If it is non-zero but not 391, the biome
histogram has drifted since this plan was written; investigate before accepting.

- [ ] **Step 5: Verify it passes on a re-run**

Run: `go test . -run TestLightingDayCycleAcrossSampleRounds -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add lighting_daycycle_golden_test.go testdata/lighting_daycycle.golden
git commit -m "test(lighting): record the day-cycle baseline before the model changes

The parity golden is about to move on nearly every room, so it stops being a
guard. This records sight-relevant light for every shipped room at twelve
moments spanning the year and the day, against the unmodified tree, so the
diff after the celestial model lands is a readout rather than a regression.

Pins Timing explicitly: a test binary loads Go defaults where NightHours is 0
and IsNight() can never be true, so an unpinned baseline would record a world
with no night in it.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: The `internal/lightscale` package

A pure leaf package. `gametime` needs `Combine` for sun plus moons, `rooms` needs
it for ambient plus lamp, and plans 4 and 5 need it for weather and darkness.
Making it a leaf means nothing has to import `gametime` just to add two lights.

**Files:**
- Create: `internal/lightscale/lightscale.go`
- Create: `internal/lightscale/lightscale_test.go`
- Create: `internal/lightscale/context.md`

- [ ] **Step 1: Write the failing tests**

```go
package lightscale

import (
	"math"
	"testing"
)

func TestCombineOfNothingIsAbsent(t *testing.T) {
	if got := Combine(8, Absent(), Absent()); !math.IsInf(got, -1) {
		t.Fatalf("want -Inf, got %v", got)
	}
}

func TestCombineOfOneTermIsThatTerm(t *testing.T) {
	if got := Combine(8, 35, Absent()); got != 35 {
		t.Fatalf("want 35, got %v", got)
	}
}

// Two equal sources read exactly one doubling step brighter. This is the
// definition of the step, so it is the load-bearing test of the whole scale.
func TestTwoEqualSourcesAddOneStep(t *testing.T) {
	got := Combine(8, 35, 35)
	if math.Abs(got-43) > 1e-9 {
		t.Fatalf("want 43, got %v", got)
	}
}

func TestFourEqualSourcesAddTwoSteps(t *testing.T) {
	got := Combine(8, 35, 35, 35, 35)
	if math.Abs(got-51) > 1e-9 {
		t.Fatalf("want 51, got %v", got)
	}
}

// A source five doublings weaker than the brightest is negligible: it
// contributes something, but under half a point. This is what stops a lantern
// mattering at noon, which the halving rule the spec originally used could not
// express.
//
// 🔴 CORRECTED 2026-09-23 after this task ran. The original bound here was
// "above 70 and no more than 70.05", which is wrong by about sevenfold. The
// exact value is 70 + 8*log2(1 + 2^((30-70)/8)) = 70.35515295486763. The
// assertion is deliberately a PROPERTY rather than that exact number, because
// exact equality here would only re-test that log2 works, which
// TestTwoEqualSourcesAddOneStep already covers.
func TestFarWeakerSourceBarelyContributes(t *testing.T) {
	got := Combine(8, 70, 30)
	if got <= 70 {
		t.Fatalf("weaker source contributed nothing: got %v, want above 70", got)
	}
	if got-70 >= 0.5 {
		t.Fatalf("weaker source contributed too much: got %v, want within 0.5 of 70", got)
	}
}

func TestCombineIsOrderIndependent(t *testing.T) {
	a := Combine(8, 12, 55, 31)
	b := Combine(8, 55, 31, 12)
	if math.Abs(a-b) > 1e-9 {
		t.Fatalf("%v != %v", a, b)
	}
}

func TestAttenuateHalfIsMinusOneStep(t *testing.T) {
	got := Attenuate(8, 60, 0.5)
	if math.Abs(got-52) > 1e-9 {
		t.Fatalf("want 52, got %v", got)
	}
}

func TestAttenuateByOneIsIdentity(t *testing.T) {
	if got := Attenuate(8, 60, 1); got != 60 {
		t.Fatalf("want 60, got %v", got)
	}
}

// A sky fraction of zero is not "contributes zero", it is "there is no sky
// here". A cave must not receive a term at all.
func TestAttenuateByZeroIsAbsent(t *testing.T) {
	if got := Attenuate(8, 60, 0); !math.IsInf(got, -1) {
		t.Fatalf("want -Inf, got %v", got)
	}
}

func TestAttenuateOfAbsentIsAbsent(t *testing.T) {
	if got := Attenuate(8, Absent(), 0.5); !math.IsInf(got, -1) {
		t.Fatalf("want -Inf, got %v", got)
	}
}

func TestNonPositiveStepDoesNotPanicOrNaN(t *testing.T) {
	if got := Combine(0, 10, 10); math.IsNaN(got) {
		t.Fatalf("NaN from zero step")
	}
	if got := Attenuate(-4, 10, 0.5); math.IsNaN(got) {
		t.Fatalf("NaN from negative step")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/lightscale/...`
Expected: FAIL to build, "undefined: Combine"

- [ ] **Step 3: Write the implementation**

```go
// Package lightscale holds the arithmetic of DOGMud's graded light scale.
//
// The scale runs -100 to 100 and is PERCEPTUAL, not linear. Zero is the darkest
// light that naturally occurs, roughly an unlit cave; negative is magical
// darkness, light actively removed. Because the scale is logarithmic, one
// constant relates it to physical light: the doubling step, which is how many
// scale points twice as much light is worth.
//
// 🔑 The same step governs three things, which is why it is ONE config knob:
// combining sources, applying a sky fraction, and the shape of the daylight
// curve. See docs/superpowers/specs/2026-09-23-graded-room-lighting-amendment-celestial.md.
//
// This package is deliberately pure: no config reads, no globals, no locks. Its
// callers own the config read, the same discipline messaging.SightThroughWindow
// follows for the band edges.
package lightscale

import "math"

// Absent is a light term that is not present at all, as distinct from a term
// that is present and dark.
//
// 🔑 The distinction is load-bearing. A cave has no sky, which is not the same
// as a sky contributing zero: on a logarithmic scale two terms at zero
// legitimately combine to something brighter than one, so a "zero" sky would
// make a cave brighter for having a sky it does not have.
func Absent() float64 { return math.Inf(-1) }

// present reports whether a term should take part in a combination. Only finite
// values do; -Inf means absent and NaN means a caller made an arithmetic
// mistake, which must not silently poison the whole room.
func present(v float64) bool { return !math.IsInf(v, 0) && !math.IsNaN(v) }

// Combine returns the light produced by every present term together.
//
//	combined = brightest + step * log2( sum over i of 2^((s_i - brightest)/step) )
//
// Two equal terms read one step brighter than one; four read two steps brighter;
// a term far below the brightest contributes almost nothing. Absent terms are
// skipped, and a combination of nothing is Absent.
//
// It is computed relative to the brightest term rather than from an absolute
// origin so that large scale values cannot overflow the exponential.
func Combine(step float64, terms ...float64) float64 {
	if !(step > 0) {
		step = 1
	}
	best := math.Inf(-1)
	for _, t := range terms {
		if present(t) && t > best {
			best = t
		}
	}
	if math.IsInf(best, -1) {
		return Absent()
	}
	sum := 0.0
	for _, t := range terms {
		if present(t) {
			sum += math.Exp2((t - best) / step)
		}
	}
	if !(sum > 0) {
		return best
	}
	return best + step*math.Log2(sum)
}

// Attenuate applies a transmission fraction to one light term: the share of the
// light that gets through a canopy, a roof, a drain-cap or a blizzard.
//
// 🔑 On a logarithmic scale a MULTIPLIER is a SUBTRACTION. Letting half the
// light through is minus one doubling step, whatever the light was. That is why
// canopy, roof and weather are one operator rather than three, and why a
// blizzard is devastating at midnight and merely gloomy at noon: it removes the
// same number of points in both cases, but the sight bands are absolute.
//
// A fraction at or below zero returns Absent, not a very dark value, because "no
// sky reaches here" is a different statement from "very little does".
func Attenuate(step, light, fraction float64) float64 {
	if !present(light) || !(fraction > 0) {
		return Absent()
	}
	if fraction >= 1 {
		return light
	}
	if !(step > 0) {
		step = 1
	}
	return light + step*math.Log2(fraction)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/lightscale/... -v`
Expected: PASS, 11 tests

- [ ] **Step 5: Write `internal/lightscale/context.md`**

```markdown
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

## Traps

- **Absent is not zero.** A cave has no sky; a sky contributing zero would make
  the cave brighter, because two terms at zero combine to one step above zero.
  `Attenuate` with fraction 0 returns `Absent()` for this reason.
- **A multiplier is a subtraction here.** Half the light is minus one step.
- `Combine` skips NaN as well as -Inf, so one bad caller cannot poison a room.
- Both functions coerce a non-positive step to 1 rather than dividing by zero.

## Who uses it

`internal/gametime` (sun plus moons), `internal/rooms` (ambient plus lamp plus
mutators). Plans 4 and 5 add weather occlusion and darkness sources on the same
two functions.
```

- [ ] **Step 6: ~~Add the package to `docs/README.md`~~ — DELETED**

🔴 **CORRECTED 2026-09-23 after this task ran. Do not do this step.** The plan
told the implementer to add a row to `docs/README.md`'s package table. **There is
no package table.** Verified: 137 `context.md` files exist under `internal/` and
`modules/`, and not one of them is individually listed in `docs/README.md`; the
eight `context.md` mentions in that file are references to
`tools/context_md_audit.py` and incidental mentions inside spec descriptions.

The convention is the one `docs/README.md` states itself: *"Per-package developer
notes live beside the code, as `context.md` in each `internal/` and `modules/`
package."* A new package's documentation home is its own `context.md`, which
Step 5 already writes. `CLAUDE.md`'s "new files go in `docs/README.md`" rule
governs files under `docs/`, not Go packages.

**The same correction applies to every later task in this plan** that mentions
adding a package to `docs/README.md`. Plan and spec documents under `docs/` still
get a row; packages do not.

- [ ] **Step 7: Commit**

```bash
git add internal/lightscale
git commit -m "feat(lighting): pure arithmetic for the graded light scale

Combine and Attenuate on a logarithmic scale, with one doubling step relating
scale points to physical light. Two equal sources read one step brighter, and
a far weaker source contributes almost nothing, which is what the spec's
original halving rule could not express: it gave a lantern-lit street at noon
a light of 125.

Absent is deliberately distinct from a dark term. A cave has no sky, and a sky
contributing zero would make the cave brighter, since two terms at zero
combine to one step above zero.

Pure by design, no config and no globals, the same discipline
messaging.SightThroughWindow follows for the band edges.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Lighting config knobs and a narrow accessor

Fifteen call sites copy a 424-field struct to read one int (99.75 ns measured).
This adds the knobs the model needs and one accessor that replaces that copy.

🔑 **Free to do now.** None of plan 1's or plan 2's lighting knobs appear in
`_datafiles/config.yaml`; they run on Go defaults, so no operator config can
break on the change.

**Files:**
- Modify: `internal/configs/config.balance.go` (after line 1125, inside `Balance`)
- Modify: `internal/configs/config.balance.lighting.go` (`validateLighting`)
- Create: `internal/configs/config.lighting_accessor.go`
- Create: `internal/configs/config.lighting_accessor_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package configs

import "testing"

func TestLightingDefaultsAreTheShippedCalibration(t *testing.T) {
	var b Balance
	b.Validate()
	if b.LightDoublingStep != 8 {
		t.Errorf("LightDoublingStep = %v, want 8", b.LightDoublingStep)
	}
	if b.WorldLatitude != 46.5 {
		t.Errorf("WorldLatitude = %v, want 46.5", b.WorldLatitude)
	}
	if b.LightEquinoxNoon != 70 {
		t.Errorf("LightEquinoxNoon = %v, want 70", b.LightEquinoxNoon)
	}
	if b.LightMoonsFull != 35 {
		t.Errorf("LightMoonsFull = %v, want 35", b.LightMoonsFull)
	}
	if b.LightStarlight != 10 {
		t.Errorf("LightStarlight = %v, want 10", b.LightStarlight)
	}
	if b.LightMoonWeightSwiftmoon != 4 || b.LightMoonWeightWanderer != 1 || b.LightMoonWeightEye != 0.5 {
		t.Errorf("moon weights = %v/%v/%v, want 4/1/0.5",
			b.LightMoonWeightSwiftmoon, b.LightMoonWeightWanderer, b.LightMoonWeightEye)
	}
}

func TestDoublingStepRejectsNonPositive(t *testing.T) {
	for _, v := range []ConfigFloat{0, -3} {
		var b Balance
		b.LightDoublingStep = v
		b.Validate()
		if b.LightDoublingStep != 8 {
			t.Errorf("step %v survived validation as %v", v, b.LightDoublingStep)
		}
	}
}

// Latitude beyond the polar circles produces days with no sunrise or no sunset,
// which the model handles but which is almost never intended. Out of range
// reverts; zero is HONOURED and means "no latitude, fall back to NightHours".
func TestWorldLatitudeRangeAndZero(t *testing.T) {
	var b Balance
	b.WorldLatitude = 0
	b.Validate()
	if b.WorldLatitude != 0 {
		t.Errorf("zero latitude was coerced to %v; zero must be honoured", b.WorldLatitude)
	}

	for _, v := range []ConfigFloat{-91, 91} {
		var b2 Balance
		b2.WorldLatitude = v
		b2.Validate()
		if b2.WorldLatitude != 46.5 {
			t.Errorf("latitude %v survived as %v", v, b2.WorldLatitude)
		}
	}
}

// Starlight must sit below the all-moons-full value or the moon curve inverts
// and a full moon reads darker than a new one. Validated as a pair, the
// LightBlindBelow/LightDimBelow precedent.
func TestMoonRangeIsValidatedAsAPair(t *testing.T) {
	var b Balance
	b.LightStarlight = 40
	b.LightMoonsFull = 20
	b.Validate()
	if b.LightStarlight != 10 || b.LightMoonsFull != 35 {
		t.Errorf("inverted pair survived as %v/%v", b.LightStarlight, b.LightMoonsFull)
	}
}

func TestMoonWeightsRejectAllZero(t *testing.T) {
	var b Balance
	b.LightMoonWeightSwiftmoon = 0
	b.LightMoonWeightWanderer = 0
	b.LightMoonWeightEye = 0
	b.Validate()
	if b.LightMoonWeightSwiftmoon != 4 {
		t.Errorf("all-zero weights survived as %v", b.LightMoonWeightSwiftmoon)
	}
}

func TestGetLightingConfigMirrorsBalance(t *testing.T) {
	c := GetConfig()
	c.Balance.LightBlindBelow = 30
	c.Balance.LightDoublingStep = 11
	c.Balance.WorldLatitude = 12.5
	SetConfigForTest(t, c)

	got := GetLightingConfig()
	if got.BlindBelow != 30 {
		t.Errorf("BlindBelow = %d, want 30", got.BlindBelow)
	}
	if got.DoublingStep != 11 {
		t.Errorf("DoublingStep = %v, want 11", got.DoublingStep)
	}
	if got.WorldLatitude != 12.5 {
		t.Errorf("WorldLatitude = %v, want 12.5", got.WorldLatitude)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/configs/... -run 'Lighting|Doubling|Latitude|Moon'`
Expected: FAIL to build, "b.LightDoublingStep undefined"

- [ ] **Step 3: Add the struct fields**

In `internal/configs/config.balance.go`, immediately after
`LightDefaultVisionStrength` (line 1125) and before the closing `}` of `Balance`:

```go
	// LightDoublingStep is how many points on the -100..100 light scale are
	// worth TWICE as much physical light. It is the single constant relating
	// the perceptual scale to real light, and it governs three things at once:
	// combining sources (two equal lamps read one step brighter), applying a
	// sky fraction (half the light is minus one step) and the shape of the
	// daylight curve.
	//
	// Eight means the 100-point span covers about 12.5 doublings, and the
	// 25-point sight bands are about three doublings wide, so climbing from
	// shapes to full sight takes roughly six times more light.
	//
	// Non-positive is coerced rather than honoured: zero would make every
	// source identical and divide by zero in the combine.
	LightDoublingStep ConfigFloat `yaml:"LightDoublingStep"` // Scale points per doubling of light (default 8)

	// WorldLatitude is the latitude, in degrees, that the whole world sits at.
	// It is the ONLY seasonal input: declination, day length, sunrise, sunset
	// and noon height all derive from it, which is why no seasonal noon-peak
	// table exists. DOGMud ships 46.5, mirroring Washington State.
	//
	// 🔑 ZERO IS HONOURED and means "this world has no latitude": night length
	// falls back to the Timing.NightHours knob, preserving upstream GoMud
	// behaviour for anyone who has not set a latitude. Out-of-range reverts.
	//
	// ⚠️ Beyond about 66 degrees this produces days with no sunrise and days
	// with no sunset. The model handles both (the half-day angle clamps), but
	// it is almost certainly not what an operator intended.
	WorldLatitude ConfigFloat `yaml:"WorldLatitude"` // Degrees north; 0 disables latitude and falls back to NightHours (default 46.5)

	// LightEquinoxNoon calibrates the sun: it is the light at noon on an
	// equinox, which is the one moment the geometry pins exactly, because
	// declination is zero there and sin(altitude) is exactly cos(latitude).
	// Every other moment of every other day is derived from it.
	//
	// At 46.5 degrees and the shipped 70, midsummer noon reads 73 and
	// midwinter noon 62. Note that 73 is UNDER the dazzle edge of 75, so
	// natural daylight never dazzles; an operator wanting midsummer midday to
	// dazzle must raise this to about 72.
	LightEquinoxNoon ConfigFloat `yaml:"LightEquinoxNoon"` // Light at noon on an equinox (default 70)

	// LightStarlight and LightMoonsFull are the two anchors of the moon curve:
	// the light of a sky with every moon new, and with every moon full. They
	// are validated as a PAIR, the LightBlindBelow/LightDimBelow precedent,
	// because starlight at or above the full value inverts the curve and makes
	// a full moon darker than a new one.
	//
	// The shipped 10 and 35 both sit BELOW LightDimBelow, so a normal observer
	// without a lamp never reads full sight outdoors at night, any night of the
	// year. The moons' whole mechanical job is deciding blind against shapes.
	LightStarlight ConfigFloat `yaml:"LightStarlight"` // Light of a sky with every moon new (default 10)
	LightMoonsFull ConfigFloat `yaml:"LightMoonsFull"` // Light of a sky with every moon full (default 35)

	// The three moons' relative light at full, with The Wanderer as 1.0.
	//
	// Swiftmoon and The Wanderer are DERIVED from world.md:52-56: brightness
	// goes as angular AREA, so Swiftmoon at twice Luna's apparent size is four
	// times the area.
	//
	// ⚠️ The Eye's 0.5 is NOT derived. The lore says only "Small, bright",
	// which this reads as about a quarter the area at roughly twice the albedo.
	// It is a reading of prose and a balance pass may move it.
	//
	// Albedo is inside each weight rather than separate from it, deliberately:
	// from the ground a large dull moon and a small bright one are
	// indistinguishable, so only the product is observable and splitting them
	// would be two knobs with one effect.
	LightMoonWeightSwiftmoon ConfigFloat `yaml:"LightMoonWeightSwiftmoon"` // Relative light at full (default 4.0, derived: 2x Luna's size is 4x the area)
	LightMoonWeightWanderer  ConfigFloat `yaml:"LightMoonWeightWanderer"`  // Relative light at full (default 1.0, the baseline)
	LightMoonWeightEye       ConfigFloat `yaml:"LightMoonWeightEye"`       // Relative light at full (default 0.5, a reading of "Small, bright", not derived)
```

- [ ] **Step 4: Add validation**

In `internal/configs/config.balance.lighting.go`, append inside `validateLighting()`:

```go
	// LightDoublingStep: non-positive is coerced, not honoured. Zero divides
	// by zero in the combine and makes every source identical.
	if !(b.LightDoublingStep > 0) {
		b.LightDoublingStep = 8
	}

	// WorldLatitude: zero is HONOURED and means "no latitude, use NightHours".
	// Only genuinely impossible values revert. This is the opposite convention
	// from LightDefaultVisionStrength, where zero means "unset", and the
	// difference is deliberate: an equatorial world is a real thing to want,
	// and it happens to be exactly what falling back to a flat NightHours
	// produces.
	if b.WorldLatitude < -90 || b.WorldLatitude > 90 {
		b.WorldLatitude = 46.5
	}

	// LightEquinoxNoon must sit on the scale. Zero is coerced: a world whose
	// equinox noon is as dark as an unlit cave is not a calibration, it is an
	// unset field.
	if b.LightEquinoxNoon <= -100 || b.LightEquinoxNoon > 100 || b.LightEquinoxNoon == 0 {
		b.LightEquinoxNoon = 70
	}

	// The moon anchors are validated as a PAIR, following
	// DarknessShapesCombatPenalty and the LightBlindBelow/LightDimBelow pair
	// above: starlight at or above the full-moon value inverts the curve, so
	// an invalid pair reverts BOTH rather than leaving one correct.
	starOK := b.LightStarlight >= -100 && b.LightStarlight <= 100
	fullOK := b.LightMoonsFull >= -100 && b.LightMoonsFull <= 100
	if !starOK || !fullOK || b.LightStarlight >= b.LightMoonsFull {
		b.LightStarlight = 10
		b.LightMoonsFull = 35
	}

	// Moon weights: a negative weight would make a waxing moon darken the sky,
	// so negatives are floored at zero. All three at zero leaves the moon curve
	// with no span at all, so that reverts the whole set rather than leaving a
	// sky that never changes.
	if b.LightMoonWeightSwiftmoon < 0 {
		b.LightMoonWeightSwiftmoon = 0
	}
	if b.LightMoonWeightWanderer < 0 {
		b.LightMoonWeightWanderer = 0
	}
	if b.LightMoonWeightEye < 0 {
		b.LightMoonWeightEye = 0
	}
	if b.LightMoonWeightSwiftmoon+b.LightMoonWeightWanderer+b.LightMoonWeightEye <= 0 {
		b.LightMoonWeightSwiftmoon = 4.0
		b.LightMoonWeightWanderer = 1.0
		b.LightMoonWeightEye = 0.5
	}
```

- [ ] **Step 5: Write the accessor**

Create `internal/configs/config.lighting_accessor.go`:

```go
package configs

// Lighting is every knob the graded light model reads, in one small struct.
//
// 🔑 This exists for a measured reason. GetBalanceConfig() copies a 424-field
// struct under two read locks and benchmarks at 99.75 ns, against 8.23 ns for
// the small GetTimingConfig(). Fifteen call sites were paying that copy to read
// a single int, and Room.LightLevel() is called from per-round loops. The knobs
// stay declared on Balance, so the yaml schema is unchanged; only the read path
// is narrowed.
type Lighting struct {
	BlindBelow            int
	DimBelow              int
	ExitsAbove            int
	DefaultVisionStrength int

	DoublingStep  float64
	WorldLatitude float64
	EquinoxNoon   float64
	Starlight     float64
	MoonsFull     float64

	MoonWeightSwiftmoon float64
	MoonWeightWanderer  float64
	MoonWeightEye       float64
}

// GetLightingConfig returns the lighting knobs without copying Balance.
func GetLightingConfig() Lighting {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()

	b := &configData.Balance
	return Lighting{
		BlindBelow:            int(b.LightBlindBelow),
		DimBelow:              int(b.LightDimBelow),
		ExitsAbove:            int(b.LightExitsAbove),
		DefaultVisionStrength: int(b.LightDefaultVisionStrength),

		DoublingStep:  float64(b.LightDoublingStep),
		WorldLatitude: float64(b.WorldLatitude),
		EquinoxNoon:   float64(b.LightEquinoxNoon),
		Starlight:     float64(b.LightStarlight),
		MoonsFull:     float64(b.LightMoonsFull),

		MoonWeightSwiftmoon: float64(b.LightMoonWeightSwiftmoon),
		MoonWeightWanderer:  float64(b.LightMoonWeightWanderer),
		MoonWeightEye:       float64(b.LightMoonWeightEye),
	}
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/configs/... -v -run 'Lighting|Doubling|Latitude|Moon'`
Expected: PASS, 6 tests

- [ ] **Step 7: Prove the accessor is actually cheaper**

Create a temporary benchmark, run it, then delete the file.

```go
// internal/configs/zz_accessor_bench_test.go — DELETE AFTER RUNNING
package configs

import "testing"

func BenchmarkGetBalanceConfigLighting(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetBalanceConfig().LightBlindBelow
	}
}
func BenchmarkGetLightingConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetLightingConfig().BlindBelow
	}
}
```

Run: `go test ./internal/configs/ -run XXX -bench 'GetBalanceConfigLighting|GetLightingConfig' -benchtime=200000x`
Expected: `GetLightingConfig` at least **five times** faster than
`GetBalanceConfigLighting`. If it is not, the struct is too large or a copy crept
in — fix it before continuing.

Then: `rm internal/configs/zz_accessor_bench_test.go`

- [ ] **Step 8: Commit**

```bash
git add internal/configs/
git commit -m "feat(configs): lighting knobs for the celestial model, and a narrow accessor

Eight knobs: the doubling step that relates the perceptual scale to physical
light, the world latitude that is the model's only seasonal input, the equinox
noon calibration, the two moon anchors and the three moon weights.

WorldLatitude honours zero deliberately, unlike LightDefaultVisionStrength
where zero means unset: an equatorial world is a real thing to want and is
exactly what falling back to a flat NightHours produces, so zero keeps
upstream GoMud behaviour for anyone who has not set a latitude.

Starlight and moons-full validate as a pair, the LightBlindBelow/LightDimBelow
precedent, because starlight at or above the full value inverts the curve.

GetLightingConfig replaces a measured 99.75 ns copy of a 424-field struct with
a small read, for fifteen call sites and a per-round loop. Free to add now
because none of these knobs appear in config.yaml yet, so no operator config
can break on the move.

The Eye's weight is documented as NOT derived: the lore says only
'Small, bright' and 0.5 is a reading of that prose.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: The celestial term

**Files:**
- Create: `internal/gametime/celestial.go`
- Create: `internal/gametime/celestial_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package gametime

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func lightingForTest() configs.Lighting {
	return configs.Lighting{
		BlindBelow: 25, DimBelow: 50, ExitsAbove: 65,
		DoublingStep: 8, WorldLatitude: 46.5, EquinoxNoon: 70,
		Starlight: 10, MoonsFull: 35,
		MoonWeightSwiftmoon: 4, MoonWeightWanderer: 1, MoonWeightEye: 0.5,
	}
}

// Declination is zero at the equinoxes and hits the axial tilt at the
// solstices. Everything else in the model rests on this.
func TestDeclinationAtSolsticesAndEquinoxes(t *testing.T) {
	if d := declinationDegrees(356); math.Abs(d+23.44) > 0.2 {
		t.Errorf("midwinter declination %v, want about -23.44", d)
	}
	if d := declinationDegrees(172); math.Abs(d-23.44) > 0.2 {
		t.Errorf("midsummer declination %v, want about 23.44", d)
	}
	if d := declinationDegrees(81); math.Abs(d) > 0.5 {
		t.Errorf("equinox declination %v, want about 0", d)
	}
}

// At 46.5 degrees, real geometry gives 8h23m of night at midsummer and 15h37m
// at midwinter. Seattle at 47.6 publishes 15h59m of daylight; the small
// difference is refraction and the solar disc, both deliberately omitted.
func TestNightHoursAtLatitude46Point5(t *testing.T) {
	if n := NightHoursAt(46.5, 172); math.Abs(n-8.374) > 0.02 {
		t.Errorf("midsummer night %v, want about 8.374", n)
	}
	if n := NightHoursAt(46.5, 356); math.Abs(n-15.626) > 0.02 {
		t.Errorf("midwinter night %v, want about 15.626", n)
	}
	if n := NightHoursAt(46.5, 81); math.Abs(n-12) > 0.05 {
		t.Errorf("equinox night %v, want about 12", n)
	}
}

func TestNightHoursAtEquatorIsTwelveAllYear(t *testing.T) {
	for _, doy := range []int{1, 81, 172, 356} {
		if n := NightHoursAt(0, doy); math.Abs(n-12) > 1e-6 {
			t.Errorf("day %d at the equator: night %v, want 12", doy, n)
		}
	}
}

// Beyond the polar circles the half-day angle has no solution. It must clamp
// to a full day or a full night rather than returning NaN.
func TestPolarLatitudesClampInsteadOfNaN(t *testing.T) {
	if n := NightHoursAt(80, 172); n != 0 {
		t.Errorf("polar midsummer night %v, want 0", n)
	}
	if n := NightHoursAt(80, 356); n != 24 {
		t.Errorf("polar midwinter night %v, want 24", n)
	}
}

// The calibration anchor: equinox noon reads exactly EquinoxNoon, because
// declination is zero there and sin(altitude) is exactly cos(latitude).
func TestEquinoxNoonHitsTheCalibrationExactly(t *testing.T) {
	cfg := lightingForTest()
	if got := SunLight(cfg, 81, 12); math.Abs(got-70) > 0.25 {
		t.Errorf("equinox noon %v, want 70", got)
	}
}

func TestSolsticeNoonsAreDerivedNotInvented(t *testing.T) {
	cfg := lightingForTest()
	if got := SunLight(cfg, 172, 12); math.Abs(got-73) > 0.6 {
		t.Errorf("midsummer noon %v, want about 73", got)
	}
	if got := SunLight(cfg, 356, 12); math.Abs(got-62) > 0.6 {
		t.Errorf("midwinter noon %v, want about 62", got)
	}
}

// 🔑 The sun is ABSENT below the horizon, not zero. A zero term would combine
// with moonlight and make a moonlit night brighter for having a sun that has
// set.
func TestSunIsAbsentBelowTheHorizon(t *testing.T) {
	cfg := lightingForTest()
	if got := SunLight(cfg, 356, 0); !math.IsInf(got, -1) {
		t.Errorf("midwinter midnight sun %v, want -Inf", got)
	}
}

// Natural daylight must never reach the dazzle edge of 75 at this calibration.
// If this fails, the spec's claim that dazzle is purely a play mechanic is
// broken and plan 5's design rests on a false premise.
func TestNaturalDaylightNeverDazzles(t *testing.T) {
	cfg := lightingForTest()
	for doy := 1; doy <= 365; doy++ {
		for h := 0.0; h < 24; h += 0.25 {
			if v := SunLight(cfg, doy, h); v >= 75 {
				t.Fatalf("day %d hour %v reached %v, at or above the dazzle edge", doy, h, v)
			}
		}
	}
}

func TestMoonLightHitsBothAnchors(t *testing.T) {
	cfg := lightingForTest()
	if got := MoonLight(cfg, 0, 0, 0); math.Abs(got-10) > 1e-6 {
		t.Errorf("all new %v, want 10", got)
	}
	if got := MoonLight(cfg, 1, 1, 1); math.Abs(got-35) > 1e-6 {
		t.Errorf("all full %v, want 35", got)
	}
}

func TestMoonLightIsMonotonic(t *testing.T) {
	cfg := lightingForTest()
	prev := math.Inf(-1)
	for f := 0.0; f <= 1.0; f += 0.05 {
		got := MoonLight(cfg, f, f, f)
		if got < prev {
			t.Fatalf("moonlight fell from %v to %v at fullness %v", prev, got, f)
		}
		prev = got
	}
}

// Swiftmoon carries four times the Wanderer's weight, so it must move the sky
// further on its own.
func TestSwiftmoonOutweighsTheWanderer(t *testing.T) {
	cfg := lightingForTest()
	swift := MoonLight(cfg, 1, 0, 0)
	wander := MoonLight(cfg, 0, 1, 0)
	if swift <= wander {
		t.Errorf("Swiftmoon %v did not outweigh the Wanderer %v", swift, wander)
	}
}

func TestCelestialLightIsMemoisedPerRound(t *testing.T) {
	c := configs.GetConfig()
	c.Timing.RoundsPerDay = 900
	c.Timing.NightHours = 8
	c.Timing.RoundSeconds = 4
	c.Timing.Validate()
	configs.SetConfigForTest(t, c)

	original := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(original) })

	util.SetRoundCount(1000)
	first := CelestialLight()
	second := CelestialLight()
	if first != second {
		t.Fatalf("same round returned %v then %v", first, second)
	}

	util.SetRoundCount(1000 + 450) // twelve hours later
	if third := CelestialLight(); third == first {
		t.Fatalf("half a day later still returned %v; the memo is not keyed on the round", third)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/gametime/... -run 'Declination|NightHoursAt|Polar|Equinox|Solstice|Sun|Moon|Celestial'`
Expected: FAIL to build, "undefined: declinationDegrees"

- [ ] **Step 3: Write the implementation**

Create `internal/gametime/celestial.go`:

```go
package gametime

import (
	"math"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/lightscale"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const (
	// axialTiltDegrees is the planet's obliquity, which sets how far
	// declination swings across the year and therefore how long the longest
	// night is. Earth's value; world.md gives no other.
	axialTiltDegrees = 23.44

	// daysPerYear matches GameDate.ReCalculate, which divides the year into
	// 365 days. If that ever changes, this must change with it.
	daysPerYear = 365.0

	// moonReferenceIntensity is the floor the moon curve is measured from: the
	// light of a sky with every moon new, in the same arbitrary intensity units
	// the moon weights use. It shapes how fast moonlight rises off the
	// starlight anchor, and the two anchors themselves are config.
	//
	// It is a constant rather than a knob because nothing yet needs to turn it
	// independently of the anchors. Plan 6's balance pass may promote it.
	moonReferenceIntensity = 0.923
)

// declinationDegrees is the sun's declination on a given day of the year:
// zero at the equinoxes, plus or minus the axial tilt at the solstices.
//
// The plus-ten offset puts midwinter near day 356, matching the northern
// calendar the world's seasons are described in.
func declinationDegrees(dayOfYear int) float64 {
	return -axialTiltDegrees * math.Cos(2*math.Pi*(float64(dayOfYear)+10)/daysPerYear)
}

// halfDayHours is the time from local noon to sunset, in hours.
//
// Beyond the polar circles the arccosine has no solution, which is a real
// physical state and not an error: the sun either never sets or never rises. It
// clamps to 12 and 0 respectively rather than returning NaN, which would
// poison every light in the world.
func halfDayHours(latitudeDegrees float64, dayOfYear int) float64 {
	phi := latitudeDegrees * math.Pi / 180
	dec := declinationDegrees(dayOfYear) * math.Pi / 180
	c := -math.Tan(phi) * math.Tan(dec)
	switch {
	case c <= -1:
		return 12 // polar day: the sun never sets
	case c >= 1:
		return 0 // polar night: the sun never rises
	}
	return math.Acos(c) * 180 / math.Pi / 15
}

// NightHoursAt reports how many hours of night a given latitude sees on a given
// day of the year. At the equator it is twelve every day; at 46.5 degrees it
// runs from 8h23m at midsummer to 15h37m at midwinter.
func NightHoursAt(latitudeDegrees float64, dayOfYear int) float64 {
	return 24 - 2*halfDayHours(latitudeDegrees, dayOfYear)
}

// solarSinAltitude is the sine of the sun's altitude above the horizon, which
// is also the share of its light falling on level ground. Negative means below
// the horizon.
func solarSinAltitude(latitudeDegrees float64, dayOfYear int, hour float64) float64 {
	phi := latitudeDegrees * math.Pi / 180
	dec := declinationDegrees(dayOfYear) * math.Pi / 180
	return math.Sin(phi)*math.Sin(dec) +
		math.Cos(phi)*math.Cos(dec)*math.Cos(2*math.Pi*(hour-12)/24)
}

// SunLight is the sun's contribution to the sky at a given moment.
//
//	sun = sunFull + step * log2( sin altitude )
//
// 🔑 This has no "night" case. Below the horizon sin(altitude) is non-positive,
// the logarithm is undefined, and the term is simply Absent. The flat-topped
// middle of the day also comes for free, because sine is flat near its peak,
// which is why no flatness exponent exists.
//
// 🔑 The calibration anchor is exact rather than sampled. At an equinox the
// declination is zero, so sin(altitude) at noon is exactly cos(latitude); no
// magic day-of-year number is needed to find it.
//
// Deliberately omitted: atmospheric extinction, which would take a low winter
// sun further down than sin(altitude) alone, and the lore's 10-15% difference
// in seasonal lengths from orbital eccentricity (world.md:44). Neither changes
// a sight band at the shipped calibration.
func SunLight(cfg configs.Lighting, dayOfYear int, hour float64) float64 {
	s := solarSinAltitude(cfg.WorldLatitude, dayOfYear, hour)
	if s <= 0 {
		return lightscale.Absent()
	}
	reference := math.Cos(cfg.WorldLatitude * math.Pi / 180)
	if reference <= 0 {
		// A pole. There is no equinox noon to calibrate against, so the model
		// has nothing to say; treat the sun as absent rather than dividing by
		// a logarithm of zero.
		return lightscale.Absent()
	}
	step := cfg.DoublingStep
	if !(step > 0) {
		step = 1
	}
	sunFull := cfg.EquinoxNoon - step*math.Log2(reference)
	return sunFull + step*math.Log2(s)
}

// MoonLight is the three moons' combined contribution, given each moon's
// fullness in [0,1] as the existing phase functions report it.
//
// The weights are relative light at full, with albedo folded in: from the
// ground a large dull moon and a small bright one are indistinguishable, so
// only the product is observable.
//
// The result is interpolated between the two configured anchors on a
// logarithmic intensity axis, so the curve rises fast off starlight and
// flattens as the sky fills, which is how light actually reads.
func MoonLight(cfg configs.Lighting, swiftmoon, wanderer, eye float64) float64 {
	weightTotal := cfg.MoonWeightSwiftmoon + cfg.MoonWeightWanderer + cfg.MoonWeightEye
	if weightTotal <= 0 {
		return cfg.Starlight
	}
	intensity := moonReferenceIntensity +
		cfg.MoonWeightSwiftmoon*clamp01(swiftmoon) +
		cfg.MoonWeightWanderer*clamp01(wanderer) +
		cfg.MoonWeightEye*clamp01(eye)

	span := math.Log2((moonReferenceIntensity + weightTotal) / moonReferenceIntensity)
	if span <= 0 {
		return cfg.Starlight
	}
	position := math.Log2(intensity/moonReferenceIntensity) / span
	return cfg.Starlight + (cfg.MoonsFull-cfg.Starlight)*position
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	}
	return v
}

// The celestial memo. One value for the WHOLE WORLD per round, not one per room.
//
// 🔑 This is NOT a performance measure and must not grow into a per-room cache.
// Measured, the uncached computation is 332 ns; at 500 LightLevel calls in a
// four-second round that is 0.005% of the round, and about 0.03% scaled for the
// single-CPU production droplet. The memo exists only because it is three lines.
// See the amendment spec's Performance section before "optimising" anything here.
//
// A config change mid-round is reflected on the next round, which is acceptable
// for a value that is constant within a round by construction.
var (
	celestialMu    sync.Mutex
	celestialRound uint64
	celestialValue float64
	celestialKnown bool
)

// CelestialLight is the sky's light at the current round: sun and moons
// combined, before any sky fraction, lamp or weather is applied.
func CelestialLight() float64 {
	round := util.GetRoundCount()

	celestialMu.Lock()
	defer celestialMu.Unlock()
	if celestialKnown && celestialRound == round {
		return celestialValue
	}

	cfg := configs.GetLightingConfig()
	gd := GetDate(round)
	hour := float64(gd.Hour24) + gd.MinuteFloat/60

	swift, wander, eye := GetAllPhases()

	celestialValue = lightscale.Combine(cfg.DoublingStep,
		SunLight(cfg, gd.Day, hour),
		MoonLight(cfg, swift, wander, eye),
	)
	celestialRound = round
	celestialKnown = true
	return celestialValue
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/gametime/... -v -run 'Declination|NightHoursAt|Polar|Equinox|Solstice|Sun|Moon|Celestial'`
Expected: PASS, 12 tests

- [ ] **Step 5: Run the whole gametime package**

Run: `go test ./internal/gametime/...`
Expected: PASS. If `integration_moons_test.go` or `moonflavor_test.go` fail,
nothing above changed the phase functions, so investigate rather than adjusting
their expectations.

- [ ] **Step 6: Commit**

```bash
git add internal/gametime/celestial.go internal/gametime/celestial_test.go
git commit -m "feat(gametime): the sky's light, derived from one latitude

Declination, day length, solar altitude and the three moons' contribution,
with WorldLatitude as the only seasonal input. At 46.5 degrees this gives
8h23m of night at midsummer and 15h37m at midwinter, and noon of 73 / 70 / 62
across midsummer, equinox and midwinter, derived rather than tabulated.

The sun has no night case: below the horizon the logarithm of sin(altitude) is
undefined and the term is simply absent, which is the same thing a cave's sky
fraction of zero produces. The flat-topped middle of the day comes free from
the shape of sine, so no flatness exponent exists.

The calibration anchor is exact rather than sampled. At an equinox declination
is zero, so sin(altitude) at noon is exactly cos(latitude), and no magic
day-of-year constant is needed.

A regression test asserts natural daylight never reaches the dazzle edge at
any hour of any day, because plan 5's design rests on dazzle being reachable
only by a shifted window or an artificial source.

Polar latitudes clamp to a full day or a full night rather than returning NaN
from an arccosine with no solution.

The memo is one value for the whole world per round and is explicitly not a
performance measure: uncached is 0.005% of a round. The comment says so, so
nobody grows it into a per-room cache later.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Night length follows the latitude

`ReCalculate` currently splits `NightHoursPerDay` in half around midnight. It
must instead derive night length from the latitude on the current day of the
year, falling back to `NightHours` when the latitude is zero.

⚠️ **`ReCalculate` computes `day` AFTER the night block today.** The night block
now needs the day of the year, so the day computation moves up. Do this carefully
and read the whole function before editing.

**Files:**
- Modify: `internal/gametime/gametime.go:196-245` (`ReCalculate`)
- Modify: `internal/gametime/gametime.go:506-560` (`GetLastPeriod`)
- Create: `internal/gametime/nightlength_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package gametime

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func pinTiming(t *testing.T, latitude float64) {
	t.Helper()
	c := configs.GetConfig()
	c.Timing.RoundsPerDay = 900
	c.Timing.NightHours = 8
	c.Timing.RoundSeconds = 4
	c.Timing.Validate()
	c.Balance.WorldLatitude = configs.ConfigFloat(latitude)
	c.Balance.Validate()
	configs.SetConfigForTest(t, c)
}

// roundFor returns the round number for a given day of the year and hour, at
// the pinned RoundsPerDay of 900.
func roundFor(dayOfYear int, hour float64) uint64 {
	return uint64(float64(dayOfYear-1)*900 + hour*37.5)
}

// Midwinter night is 15h37m, so 05:00 is still night; midsummer night is
// 8h23m, so 05:00 is broad day. Under the old flat model both were day,
// because night ended at 04:00 all year.
func TestNightLengthVariesWithTheSeason(t *testing.T) {
	pinTiming(t, 46.5)
	original := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(original) })

	if gd := GetDate(roundFor(356, 5)); !gd.Night {
		t.Errorf("midwinter 05:00 reported day; at 46.5 degrees night runs to about 07:49")
	}
	if gd := GetDate(roundFor(172, 5)); gd.Night {
		t.Errorf("midsummer 05:00 reported night; at 46.5 degrees night ends about 04:11")
	}
}

// Latitude zero must reproduce the shipped flat model exactly, so an operator
// who has not set a latitude sees no change at all.
func TestZeroLatitudeFallsBackToNightHours(t *testing.T) {
	pinTiming(t, 0)
	original := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(original) })

	for _, doy := range []int{1, 81, 172, 356} {
		if gd := GetDate(roundFor(doy, 3)); !gd.Night {
			t.Errorf("day %d 03:00 reported day; NightHours 8 puts night from 20:00 to 04:00", doy)
		}
		if gd := GetDate(roundFor(doy, 5)); gd.Night {
			t.Errorf("day %d 05:00 reported night under the flat fallback", doy)
		}
	}
}

// Midnight is night and midday is day at every latitude on every day. If this
// ever fails the hour-of-day arithmetic has drifted.
func TestMidnightIsAlwaysNightAndNoonAlwaysDay(t *testing.T) {
	pinTiming(t, 46.5)
	original := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(original) })

	for doy := 1; doy <= 365; doy += 7 {
		if gd := GetDate(roundFor(doy, 0)); !gd.Night {
			t.Fatalf("day %d midnight reported day", doy)
		}
		if gd := GetDate(roundFor(doy, 12)); gd.Night {
			t.Fatalf("day %d noon reported night", doy)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/gametime/... -run 'NightLength|ZeroLatitude|MidnightIsAlways' -v`
Expected: FAIL. `TestNightLengthVariesWithTheSeason` fails both assertions,
because night is a fixed 20:00 to 04:00 today.

- [ ] **Step 3: Rewrite the night block in `ReCalculate`**

Replace the existing block, which currently reads:

```go
	night := false
	halfNight := int(math.Floor(float64(g.NightHoursPerDay) / 2))
	nightStart := 24 - halfNight
	nightEnd := int(g.NightHoursPerDay) - halfNight
	if hour >= nightStart || hour < nightEnd {
		night = true
	}
```

with this, and **move the `day` / `year` computation (currently below) above it**,
because the night length now depends on the day of the year:

```go
	// Day of the year must be known before night length, because night length
	// is seasonal. This block was BELOW the night block before the celestial
	// model landed; it moved up for that reason and must stay here.
	day := math.Floor(float64(currentRoundAdjusted)/float64(g.RoundsPerDay)) + 1
	year := math.Ceil(day / 365)
	if year > 1 {
		day -= math.Floor((year - 1) * 365)
	}

	// Night length is derived from the world's latitude on this day of the
	// year. A latitude of zero means "this world has no latitude", and falls
	// back to the flat Timing.NightHours knob, which is upstream GoMud's model
	// and what an operator who has set no latitude still gets.
	//
	// The hour used here is FRACTIONAL, unlike the integer `hour` above,
	// because a latitude-derived night boundary lands at 07:49, not 08:00.
	// Comparing an integer hour against it would round the boundary to the
	// nearest hour and lose up to half an hour of night at each end.
	hourOfDay := float64(roundOfDay) / float64(g.RoundsPerDay) * 24

	nightHours := float64(g.NightHoursPerDay)
	if latitude := configs.GetLightingConfig().WorldLatitude; latitude != 0 {
		nightHours = NightHoursAt(latitude, int(day))
	}
	halfNight := nightHours / 2
	nightStartHour := 24 - halfNight
	nightEndHour := halfNight

	night := hourOfDay >= nightStartHour || hourOfDay < nightEndHour
```

Then further down, where `g.Day` and `g.Year` were assigned from the old
computation, use the values computed above rather than recomputing them, and
set the two display fields from the fractional boundaries:

```go
	// NightStart and DayStart are display values (GameDate.String's dusk check
	// and the time command), so they round to the nearest hour. The Night
	// boolean above is computed from the unrounded boundaries.
	g.NightStart = int(math.Round(nightStartHour))
	g.DayStart = int(math.Round(nightEndHour))
```

⚠️ The old code computed `nightEnd` as `NightHoursPerDay - halfNight`, which for
an odd `NightHours` differs from `halfNight`. The new code is symmetric. At the
shipped `NightHours: 8` both give 4, so the fallback path is unchanged; for odd
values the boundary moves by half an hour. That is a deliberate simplification.

⚠️ `internal/gametime` now imports `internal/configs` inside `ReCalculate`.
It already imports `configs` at the top of the file, so no new import edge is
created. Confirm with `go build ./...` that no cycle appears.

- [ ] **Step 4: Update `GetLastPeriod` to match**

In `GetLastPeriod`, the `sunrise` and `sunset` arms currently use
`nightHoursPerDay` read straight from `Timing`. They must use the same
derivation, or "last sunrise" disagrees with `IsNight()`.

Replace the `nightHoursPerDay := uint64(c.NightHours)` line and the two arms
that use it:

```go
	// Night length here must match ReCalculate's, or "the last sunrise" will
	// disagree with whether it is currently night. Fact from the spec: no
	// shipped data uses a sunrise or sunset decayrate today, so this path is
	// exercised only by the admin time-jump commands and the time command.
	nightHoursPerDay := float64(c.NightHours)
	if latitude := configs.GetLightingConfig().WorldLatitude; latitude != 0 {
		dayOfYear := int(roundNumber/roundsPerDay)%365 + 1
		nightHoursPerDay = NightHoursAt(latitude, dayOfYear)
	}
```

and in the two arms replace `float64(nightHoursPerDay)` with `nightHoursPerDay`.

⚠️ `nightHoursPerDay` was `uint64` before this change and is now `float64`, so
every arithmetic expression using it in this function must be checked. The two
sunrise and sunset arms already wrap it in `float64(...)`, which becomes a
redundant conversion the compiler will reject; remove those conversions rather
than leaving them.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/gametime/... -v`
Expected: PASS, including the pre-existing gametime tests.

⚠️ If a pre-existing test fails, check whether it pins `Timing`. An unpinned test
was asserting on a world with `NightHours: 0`, where `IsNight()` was always false
and now still is, since latitude 0 is also the Balance default in a bare
`Balance{}`. If a test fails because it *did* pin Timing but not latitude, it is
now seeing a seasonal night: pin `WorldLatitude` to 0 in that test to keep its
old meaning, and say so in a comment.

- [ ] **Step 6: Run the full suite**

Run: `go test ./...`
Expected: PASS. Any failure outside `gametime` is a real consumer of `IsNight()`
whose fixture assumed a fixed boundary. There are 14 non-test consumers; fix the
fixture, not the model.

- [ ] **Step 7: Commit**

```bash
git add internal/gametime/
git commit -m "feat(gametime): night length follows the world's latitude

ReCalculate derives night length from WorldLatitude on the current day of the
year instead of splitting a flat NightHours around midnight. At 46.5 degrees
midwinter night is 15h37m and midsummer 8h23m, against a flat 8h all year.

Latitude zero falls back to NightHours exactly, so an operator who has set no
latitude sees no change and upstream GoMud behaviour is preserved.

The day-of-year computation moved ABOVE the night block, because night length
now depends on it. The Night boolean is computed from fractional hours rather
than the integer hour, because a derived boundary lands at 07:49 and comparing
an integer hour would lose up to half an hour of night at each end.
NightStart and DayStart stay integers: they are display values.

GetLastPeriod's sunrise and sunset arms use the same derivation, or the last
sunrise would disagree with whether it is currently night. No shipped data
uses a sunrise or sunset decayrate, so that path is exercised only by the
admin time-jump commands.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Biomes declare a sky fraction and a lamp

Delete `DarkArea` and `LitArea` so the compiler enumerates every consumer. This
is the project's established refactoring idiom and it found a missed call site
in M5 PR 3.

**Files:**
- Modify: `internal/rooms/biomes.go:14-45` and `:61-67` and `Validate`
- Modify: `_datafiles/world/dogmud/biomes/*.yaml` (17 files)
- Create: `internal/rooms/biomes_light_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package rooms

import "testing"

func TestBiomeSkyLightDefaultsToFullyOpen(t *testing.T) {
	b := BiomeInfo{BiomeId: "x", Name: "X", Symbol: "."}
	if got := b.SkyLightFraction(); got != 1.0 {
		t.Errorf("unset skylight = %v, want 1.0", got)
	}
}

// 🔑 Zero must be HONOURED, not treated as unset. A cave's sky fraction is
// genuinely zero, and coercing it to the default would make every cave as
// bright as an open field.
func TestBiomeSkyLightZeroIsHonoured(t *testing.T) {
	zero := 0.0
	b := BiomeInfo{BiomeId: "cave", Name: "Cave", Symbol: ".", SkyLight: &zero}
	if got := b.SkyLightFraction(); got != 0 {
		t.Errorf("authored zero skylight = %v, want 0", got)
	}
}

func TestBiomeLampDefaultsToNone(t *testing.T) {
	b := BiomeInfo{BiomeId: "x", Name: "X", Symbol: "."}
	if got, ok := b.LampValue(); ok {
		t.Errorf("unset lamp reported %v, want none", got)
	}
}

func TestBiomeLampZeroIsHonoured(t *testing.T) {
	zero := 0
	b := BiomeInfo{BiomeId: "x", Name: "X", Symbol: ".", Lamp: &zero}
	got, ok := b.LampValue()
	if !ok || got != 0 {
		t.Errorf("authored zero lamp = (%v,%v), want (0,true)", got, ok)
	}
}

func TestBiomeValidateRejectsSkyLightOutOfRange(t *testing.T) {
	for _, v := range []float64{-0.1, 1.1} {
		f := v
		b := BiomeInfo{BiomeId: "x", Name: "X", Symbol: ".", SkyLight: &f}
		if err := b.Validate(); err == nil {
			t.Errorf("skylight %v passed validation", v)
		}
	}
}

func TestBiomeValidateRejectsLampOffScale(t *testing.T) {
	for _, v := range []int{-101, 101} {
		n := v
		b := BiomeInfo{BiomeId: "x", Name: "X", Symbol: ".", Lamp: &n}
		if err := b.Validate(); err == nil {
			t.Errorf("lamp %v passed validation", v)
		}
	}
}

// Every shipped biome must declare a sky fraction. An unset one silently reads
// as fully open sky, which is right for a meadow and catastrophically wrong for
// a cave, so the shipped set is required to be explicit.
func TestEveryShippedBiomeDeclaresASkyFraction(t *testing.T) {
	loadBiomesForTest(t)
	for _, b := range GetAllBiomes() {
		if b.BiomeId == "default" {
			continue // synthetic Go fallback, not authored; plan 3b removes its users
		}
		if b.SkyLight == nil {
			t.Errorf("biome %q declares no skylight", b.BiomeId)
		}
	}
}
```

⚠️ `loadBiomesForTest` must match whatever helper the existing tests in
`internal/rooms` use to load `_datafiles`. **Read the package's existing tests and
reuse theirs.** If none exists, call `LoadBiomeDataFiles()` after pointing
`configs` at the datafiles path the way the package's other tests do.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/rooms/... -run 'Biome(SkyLight|Lamp|Validate)|EveryShippedBiome'`
Expected: FAIL to build, "unknown field SkyLight"

- [ ] **Step 3: Change the struct**

In `internal/rooms/biomes.go`, **delete** these two fields from `BiomeInfo`:

```go
	DarkArea       bool    `yaml:"darkarea"`
	LitArea        bool    `yaml:"litarea"`
```

and **delete** these two methods entirely:

```go
func (bi *BiomeInfo) IsLit() bool  { return bi.LitArea && !bi.DarkArea }
func (bi *BiomeInfo) IsDark() bool { return !bi.LitArea && bi.DarkArea }
```

Add in their place:

```go
	// SkyLight is the fraction of the open sky's light that reaches this
	// biome's floor: 1.0 for a desert dune, about 0.45 under forest canopy,
	// 0.0 at the back of a cave. It is applied as an attenuation on the
	// logarithmic light scale, so it is the SAME operator weather occlusion
	// uses in plan 4. Canopy, roof, drain-cap and blizzard are one idea.
	//
	// 🔑 It is a POINTER because zero is meaningful. A cave's sky fraction is
	// genuinely zero, which must be distinguishable from the field being
	// absent; an unset fraction reads as fully open sky.
	//
	// A room may override this; see Room.SkyLight.
	SkyLight *float64 `yaml:"skylight,omitempty"`

	// Lamp is a permanent light source belonging to the place itself: street
	// lanterns, a hearth, a cave's bioluminescence. It joins the room's light
	// on the same combine as the sky and any carried source, rather than
	// acting as a floor, so a lantern-lit tavern plus a carried torch does not
	// double-count.
	//
	// 🔑 Also a POINTER, for the same reason: a lamp of zero is a place that
	// has a light source producing nothing, which an unset field does not mean.
	//
	// Guidance: a value below LightDimBelow leaves a normal observer reading
	// shapes with names hidden, which is what a back lane should do; a value
	// above it means full sight all night, which is what a main street or an
	// inn should do.
	Lamp *int `yaml:"lamp,omitempty"`
```

Add the two accessors:

```go
// SkyLightFraction is the biome's sky fraction, defaulting to a fully open sky
// when unset.
func (bi *BiomeInfo) SkyLightFraction() float64 {
	if bi.SkyLight == nil {
		return 1.0
	}
	return *bi.SkyLight
}

// LampValue is the biome's own light source and whether it declares one at all.
func (bi *BiomeInfo) LampValue() (int, bool) {
	if bi.Lamp == nil {
		return 0, false
	}
	return *bi.Lamp, true
}
```

In `Validate()`, **delete**:

```go
	if bi.DarkArea && bi.LitArea {
		return fmt.Errorf("biome '%s' cannot be both dark and lit", bi.BiomeId)
	}
```

and add:

```go
	if bi.SkyLight != nil && (*bi.SkyLight < 0 || *bi.SkyLight > 1) {
		return fmt.Errorf("biome '%s' skylight %v is outside 0.0 to 1.0", bi.BiomeId, *bi.SkyLight)
	}
	if bi.Lamp != nil && (*bi.Lamp < -100 || *bi.Lamp > 100) {
		return fmt.Errorf("biome '%s' lamp %d is off the -100 to 100 light scale", bi.BiomeId, *bi.Lamp)
	}
```

- [ ] **Step 4: Let the compiler enumerate the consumers**

Run: `go build ./... 2>&1 | tee /tmp/darkarea-consumers.txt`
Expected: FAIL, listing every use of `IsDark`, `IsLit`, `DarkArea`, `LitArea`.

**Do not fix them yet beyond `lighting.go`**, which Task 8 rewrites wholesale.
Record the list; if anything outside `internal/rooms/lighting.go` and the
synthetic default biome in `biomes.go` appears, stop and read it, because the
spec's consumer survey did not predict it.

The two known sites are `internal/rooms/lighting.go:76-80` (inside
`legacyVisibility`, deleted in Task 8) and the two synthetic `default` biome
literals in `LoadBiomeDataFiles`, which set `LitArea: true`. Replace both
literals' `LitArea: true` with nothing — the synthetic default then has an unset
`SkyLight`, which reads as fully open sky, and plan 3b gives its 117 rooms real
biomes.

To keep the tree compiling until Task 8, temporarily replace the two lines in
`legacyVisibility` with a `skylight`-based equivalent; simpler, do Task 6 and
Task 8 in one working session and commit them together if the tree will not
build in between. **The commit below assumes Task 8 is done.** If you are
committing Task 6 alone, first stub `legacyVisibility`'s biome branch as:

```go
	// TEMPORARY, replaced wholesale in Task 8.
	if biome.SkyLightFraction() <= 0 {
		visibility -= 2
		if visibility < 0 {
			visibility = 0
		}
	}
```

- [ ] **Step 5: Author the 17 biome YAMLs**

Replace `darkarea:` and `litarea:` in every file under
`_datafiles/world/dogmud/biomes/` with `skylight:` and, where appropriate,
`lamp:`. `indoor:` is unchanged.

| File | skylight | lamp | Why |
|---|---|---|---|
| `cave.yaml` | `0.0` | — | No sky at all. The 31 lit Crash Site rooms keep their light from their `lightmod: 2` mutator, not the biome |
| `dungeon.yaml` | `0.0` | — | As cave |
| `spiderweb.yaml` | `0.0` | — | As cave. Zero shipped rooms |
| `swamp.yaml` | `0.35` | — | Outdoor but heavily overhung. Dark today by `darkarea`, and 0.35 keeps it blind all night |
| `forest.yaml` | `0.45` | — | Canopy. This is the value that makes the trees, not the sky, produce darkness |
| `house.yaml` | `0.15` | `50` | Built interior with windows and a hearth. Plan 3b folds this biome into `interior` |
| `fort.yaml` | `0.35` | — | 🔴 **Loses its lit status.** Its rooms are a ruined tower "winding up into the dark" AND an open training yard, so the biome default is a middling shade and plan 3c authors room overrides |
| `city.yaml` | `0.95` | `35` | Open street between buildings. The lamp is the SIDE-LANE value; plan 3c raises main streets to 55 |
| `land.yaml` | `1.0` | — | 🔴 **Loses its lit status.** 224 rooms of open countryside that were flagged as lantern-lit |
| `farmland.yaml` | `1.0` | — | Open field |
| `cliffs.yaml` | `1.0` | — | Open |
| `desert.yaml` | `1.0` | — | Open |
| `mountains.yaml` | `1.0` | — | Open |
| `road.yaml` | `1.0` | — | Open |
| `shore.yaml` | `1.0` | — | Open |
| `snow.yaml` | `1.0` | — | Open. Glare is a plan 4 or 5 concern, not a biome default |
| `water.yaml` | `1.0` | — | Open |

Example, `_datafiles/world/dogmud/biomes/cave.yaml` — replace the two boolean
lines with one:

```yaml
biomeid: cave
name: Cave
symbol: "•"
description: ...
skylight: 0.0
indoor: true
requireditemid: 0
usesitem: false
burns: false
movementcost: 1.0
```

⚠️ **No semicolons and no unquoted colons in YAML values.** An unquoted colon in
a description silently breaks the parse.

- [ ] **Step 6: Verify no boolean survives**

Run these **standalone**, not in an `&&` chain, because `grep -c` exits 1 on zero
matches and would silently skip the rest of a chain:

```bash
grep -rn "darkarea\|litarea" _datafiles/world/dogmud/biomes/
```
Expected: no output.

```bash
grep -rn "DarkArea\|LitArea\|IsDark()\|IsLit()" --include=*.go internal/ modules/
```
Expected: no output. Note `Room.IsLit()` from Task 9 does not match `IsLit()`
here because it is `r.IsLit()`; if it does appear, it is the new method and is
fine.

- [ ] **Step 7: Run tests**

Run: `go test ./internal/rooms/... -v -run 'Biome|EveryShippedBiome'`
Expected: PASS, 7 tests

- [ ] **Step 8: Commit** (with Task 8 if the tree will not build alone)

```bash
git add internal/rooms/biomes.go internal/rooms/biomes_light_test.go _datafiles/world/dogmud/biomes/
git commit -m "feat(rooms): biomes declare a sky fraction and a lamp

Deletes darkarea and litarea so the compiler enumerates every consumer, the
idiom that found a missed call site in M5 PR 3. They are replaced by skylight,
the fraction of the open sky that reaches the floor, and lamp, a permanent
light source belonging to the place.

Both are pointers because zero is meaningful for both. A cave's sky fraction
is genuinely zero and must be distinguishable from the field being absent;
coercing it to the default would make every cave as bright as an open field.

skylight is applied as an attenuation on the logarithmic scale, which is the
same operator plan 4 uses for weather. Canopy, roof, drain-cap and blizzard
become one idea rather than two mechanisms.

land and fort lose their lit status. land is 224 rooms of open countryside
that were flagged as lantern-lit like a street; fort covers both an open
training yard and a buried vault, so it takes a middling default and plan 3c
authors the room overrides.

city ships the SIDE LANE lamp of 35, which leaves a normal observer reading
shapes with names hidden all night. Plan 3c raises main streets to 55.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Room-level sky and lamp overrides

`fort` is the proof: *Training Yard* is open sky and *Buried Vault* is not, in
the same biome. Rather than split biomes, a room may override either value.

**Files:**
- Modify: `internal/rooms/rooms.go` (near line 90, beside `Biome`)
- Create: `internal/rooms/room_light_override_test.go`

- [ ] **Step 1: Write the failing test**

```go
package rooms

import "testing"

func TestRoomOverridesBiomeSkyAndLamp(t *testing.T) {
	open := 1.0
	lamp := 55

	r := Room{Biome: "cave"}
	if got := r.skyLightFraction(); got != 0 {
		t.Errorf("cave room without override = %v, want 0", got)
	}

	r.SkyLight = &open
	if got := r.skyLightFraction(); got != 1.0 {
		t.Errorf("overridden skylight = %v, want 1.0", got)
	}

	if got, ok := r.lampValue(); ok {
		t.Errorf("cave room lamp = (%v,%v), want none", got, ok)
	}
	r.Lamp = &lamp
	got, ok := r.lampValue()
	if !ok || got != 55 {
		t.Errorf("overridden lamp = (%v,%v), want (55,true)", got, ok)
	}
}

// An override of zero must survive. A buried vault inside an otherwise sunlit
// fort is exactly this case.
func TestRoomZeroOverridesAreHonoured(t *testing.T) {
	zero := 0.0
	r := Room{Biome: "land", SkyLight: &zero}
	if got := r.skyLightFraction(); got != 0 {
		t.Errorf("zero override = %v, want 0", got)
	}
}
```

⚠️ This test calls `GetBiome()` indirectly, which needs biomes loaded. Use the
package's existing load helper, as in Task 6.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rooms/... -run 'RoomOverrides|RoomZeroOverrides'`
Expected: FAIL to build, "unknown field SkyLight in struct literal"

- [ ] **Step 3: Add the fields and accessors**

In `internal/rooms/rooms.go`, immediately after the `Biome` field (line 90):

```go
	// SkyLight and Lamp override this room's biome defaults. Both are
	// pointers, and nil means "use the biome", which is distinct from an
	// authored zero.
	//
	// 🔑 These exist because a biome is not always granular enough. `fort`
	// holds both the open Training Yard and the buried Tower Base, and
	// `new_plymouth_sewers` is a brick vault lit by one drain-cap. Splitting
	// a biome for every such room would multiply the weather prose classes
	// that key on biome name; an override does not.
	//
	// They carry instance:"skip" to match Biome: an ephemeral instance of a
	// room inherits its template's lighting rather than persisting its own.
	SkyLight *float64 `yaml:"skylight,omitempty" instance:"skip"`
	Lamp     *int     `yaml:"lamp,omitempty" instance:"skip"`
```

At the bottom of `internal/rooms/lighting.go`:

```go
// skyLightFraction is this room's sky fraction: its own override if it has one,
// otherwise its biome's.
//
// ⚠️ GetBiome can return nil when the biome registry has not been loaded, which
// is the normal state in a unit test that does not read _datafiles. A nil check
// here is not defensive padding: without it every table-driven lighting test
// must load the whole world first, and a nil dereference in LightLevel would
// take down a live room read.
func (r *Room) skyLightFraction() float64 {
	if r.SkyLight != nil {
		return *r.SkyLight
	}
	if b := r.GetBiome(); b != nil {
		return b.SkyLightFraction()
	}
	return 1.0
}

// lampValue is this room's own light source, and whether it has one at all.
// Nil-safe for the same reason as skyLightFraction.
func (r *Room) lampValue() (int, bool) {
	if r.Lamp != nil {
		return *r.Lamp, true
	}
	if b := r.GetBiome(); b != nil {
		return b.LampValue()
	}
	return 0, false
}
```

⚠️ Confirm `GetBiome`'s nil behaviour before relying on it. `rooms.go:2904-2916`
falls back to `GetBiome("")`, which returns `nil, false` when the registry is
empty, and the existing code assigns that nil straight through. Read that
function and, if it cannot return nil in practice, keep the guards anyway and say
so in a comment — the cost is two branches and the alternative is a crash in a
per-round path.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/rooms/... -run 'RoomOverrides|RoomZeroOverrides' -v`
Expected: PASS, 2 tests

- [ ] **Step 5: Commit**

```bash
git add internal/rooms/rooms.go internal/rooms/lighting.go internal/rooms/room_light_override_test.go
git commit -m "feat(rooms): a room may override its biome's sky fraction and lamp

fort is the proof: Training Yard is open sky and Buried Vault is not, in the
same biome. Splitting a biome for every such room would multiply the weather
prose classes that key on biome name; an override does not.

Both are pointers so an authored zero survives, which is exactly the buried
vault inside an otherwise sunlit fort. They carry instance:\"skip\" to match
Biome, so an ephemeral instance inherits its template's lighting.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: `LightLevel` composes the real model

Delete `legacyVisibility`. This is the task that changes what players see.

**Files:**
- Modify: `internal/rooms/lighting.go` (wholesale)
- Create: `internal/rooms/lighting_model_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/lightscale"
)

func modelCfg() configs.Lighting {
	return configs.Lighting{
		BlindBelow: 25, DimBelow: 50, ExitsAbove: 65,
		DoublingStep: 8, WorldLatitude: 46.5, EquinoxNoon: 70,
		Starlight: 10, MoonsFull: 35,
		MoonWeightSwiftmoon: 4, MoonWeightWanderer: 1, MoonWeightEye: 0.5,
	}
}

// A room with no sky and no lamp is light 0: the darkest natural light, which
// the scale defines as an unlit cave. It is NOT negative; negative is magical
// darkness, which plan 5 introduces.
func TestUnlitCaveIsZeroNotNegative(t *testing.T) {
	zero := 0.0
	r := Room{SkyLight: &zero}
	if got := r.lightLevel(modelCfg(), 70); got != 0 {
		t.Errorf("unlit cave = %d, want 0", got)
	}
}

// The sky fraction is an attenuation, so 0.5 costs exactly one doubling step.
func TestSkyFractionCostsOneStepPerHalving(t *testing.T) {
	half := 0.5
	r := Room{SkyLight: &half}
	if got := r.lightLevel(modelCfg(), 60); got != 52 {
		t.Errorf("half sky under a 60 sky = %d, want 52", got)
	}
}

// A lamp joins the same combine as the sky rather than acting as a floor, so a
// lamp equal to the ambient reads one step above it, not double.
func TestLampJoinsTheCombineRatherThanFlooring(t *testing.T) {
	open := 1.0
	lamp := 36
	r := Room{SkyLight: &open, Lamp: &lamp}
	if got := r.lightLevel(modelCfg(), 36); got != 44 {
		t.Errorf("lamp 36 under a 36 sky = %d, want 44", got)
	}
}

// 🔑 The property that made the log combine necessary: a lantern is nearly
// irrelevant at noon. Under the spec's original halving rule this read 125.
func TestLanternIsNearlyIrrelevantAtNoon(t *testing.T) {
	open := 1.0
	lamp := 55
	r := Room{SkyLight: &open, Lamp: &lamp}
	got := r.lightLevel(modelCfg(), 70)
	if got < 70 || got > 74 {
		t.Errorf("lantern 55 at noon 70 = %d, want 70 to 74", got)
	}
}

// A positive lightmod mutator must still make a dark room readable: the 31
// Crash Site Interior rooms and 12 Foldweave rooms depend on it.
func TestPositiveLightModBridgeKeepsLitCavesLit(t *testing.T) {
	zero := 0.0
	r := Room{SkyLight: &zero}
	got := r.lightLevelWithMutatorBridge(modelCfg(), lightscale.Absent(), 2, 0)
	if got < modelCfg().DimBelow {
		t.Errorf("lightmod +2 in a cave = %d, want at or above DimBelow (%d)",
			got, modelCfg().DimBelow)
	}
}

// A negative lightmod attenuates the SKY only. A blizzard does not dim your
// lantern.
func TestNegativeLightModAttenuatesSkyNotLamp(t *testing.T) {
	open := 1.0
	r := Room{SkyLight: &open}
	clear := r.lightLevelWithMutatorBridge(modelCfg(), 60, 0, 0)
	stormy := r.lightLevelWithMutatorBridge(modelCfg(), 60, 0, 1)
	if clear-stormy != 8 {
		t.Errorf("one step of occlusion moved light by %d, want 8", clear-stormy)
	}
}

func TestLightIsClampedToTheScale(t *testing.T) {
	open := 1.0
	lamp := 100
	r := Room{SkyLight: &open, Lamp: &lamp}
	if got := r.lightLevel(modelCfg(), 100); got > 100 {
		t.Errorf("light = %d, above the scale ceiling", got)
	}
}
```

⚠️ **These tests build bare `Room{}` literals**, whose maps and slices are nil.
`lightLevelWithMutatorBridge` calls `r.GetMobs(FindHasLight)` and
`r.GetPlayers(FindHasLight)`. Before writing the tests, check those two methods
tolerate a zero-valued `Room`; ranging a nil slice is fine, but indexing a nil
map for writing is not. If either panics, construct the fixture with whatever
helper `internal/rooms`' existing tests use rather than a bare literal, and use
that same helper in every test in this task.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/rooms/... -run 'UnlitCave|SkyFraction|LampJoins|Lantern|LightMod|Clamped'`
Expected: FAIL to build, "r.lightLevel undefined"

- [ ] **Step 3: Rewrite `internal/rooms/lighting.go`**

Replace the whole file body below the imports. **Delete `legacyVisibility`,
`LightDark`, `LightRoomOnly` and `LightFull`** — the three constants existed only
to map the old three-value model and nothing should assume them now.

```go
package rooms

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/lightscale"
)

// LightLevel reports the room's light on the graded -100 to 100 scale.
//
// Three terms compose it, all on one logarithmic operator:
//
//  1. The sky, which is the celestial term attenuated by this room's sky
//     fraction. A room with no sky receives no term at all, which is not the
//     same as receiving a term of zero.
//  2. The room's own lamp, if it has one, joining the combine rather than
//     acting as a floor, so a lantern-lit tavern plus a carried torch does not
//     double-count.
//  3. Anyone in the room carrying a light.
//
// Weather and mutators attenuate the SKY only: a blizzard does not dim a
// lantern. Plan 4 gives them a real occlusion fraction; until then the old
// -2 to 2 LightMod vocabulary is bridged, one point per doubling step.
func (r *Room) LightLevel() int {
	return r.lightLevel(configs.GetLightingConfig(), gametime.CelestialLight())
}

// IsLit reports whether a normal observer can see anything at all here.
//
// 🔑 This is the predicate plan 1's design promised and never built. Fifteen
// call sites were hand-rolling `LightLevel() >= GetBalanceConfig().LightBlindBelow`,
// each copying a 424-field struct (99.75 ns measured) to read one int. This
// reads the narrow lighting config once.
func (r *Room) IsLit() bool {
	cfg := configs.GetLightingConfig()
	return r.lightLevel(cfg, gametime.CelestialLight()) >= cfg.BlindBelow
}

// lightLevel is LightLevel with its two reads injected, so it is testable
// without global state and so a caller holding both can avoid reading twice.
func (r *Room) lightLevel(cfg configs.Lighting, celestial float64) int {
	lightMod, occlusionSteps := r.mutatorLightTerms()
	return r.lightLevelWithMutatorBridge(cfg, celestial, lightMod, occlusionSteps)
}

// lightLevelWithMutatorBridge is the composition itself, with the mutator
// contribution already summarised, so tests can drive it directly.
//
// lightMod is the total POSITIVE LightMod across active mutators, and
// occlusionSteps the total NEGATIVE, expressed as doubling steps of sky removed.
func (r *Room) lightLevelWithMutatorBridge(cfg configs.Lighting, celestial float64, lightMod, occlusionSteps int) int {
	step := cfg.DoublingStep
	if !(step > 0) {
		step = 1
	}

	terms := make([]float64, 0, 4)

	// 1. The sky, attenuated by this room's fraction and then by any weather
	// blocking it. Attenuate returns Absent for a fraction of zero, so a cave
	// contributes no term rather than a term of zero.
	sky := r.skyLightFraction()
	if occlusionSteps > 0 {
		sky *= math.Exp2(-float64(occlusionSteps))
	}
	terms = append(terms, lightscale.Attenuate(step, celestial, sky))

	// 2. The room's own lamp.
	if lamp, ok := r.lampValue(); ok {
		terms = append(terms, float64(lamp))
	}

	// 3. The positive LightMod bridge. A +1 mutator lands exactly on
	// LightDimBelow and +2 one step above it, so the 31 Crash Site Interior
	// rooms and 12 Foldweave rooms that a static `lightmod: 2` holds lit today
	// stay fully visible. Plan 4 replaces this with an authored lamp value.
	if lightMod > 0 {
		terms = append(terms, float64(cfg.DimBelow)+float64(lightMod-1)*step)
	}

	// 4. Anyone carrying a light. Plan 5 gives carried sources real magnitudes
	// that scale from stat and skill; until then any light source lifts the
	// room to the bottom of the perfect band, which is what the old model's
	// "someone has light, cancel the darkness" rule effectively did.
	if len(r.GetMobs(FindHasLight)) > 0 || len(r.GetPlayers(FindHasLight)) > 0 {
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
	return n
}

// mutatorLightTerms sums the active mutators' LightMod into a positive
// contribution and a count of negative doubling steps.
//
// ⚠️ This is a BRIDGE. The -2 to 2 LightMod vocabulary predates the graded
// scale and plan 4 retires it in favour of an authored occlusion fraction and
// lamp value. Do not extend it.
func (r *Room) mutatorLightTerms() (lightMod, occlusionSteps int) {
	for mut := range r.ActiveMutators {
		spec := mut.GetSpec()
		if spec == nil || spec.LightMod == 0 {
			continue
		}
		if spec.LightMod > 0 {
			lightMod += spec.LightMod
		} else {
			occlusionSteps += -spec.LightMod
		}
	}
	return lightMod, occlusionSteps
}
```

⚠️ `internal/rooms/rooms.go:312` has `func (litRoom) LightLevel() int { return LightRoomOnly }`.
`LightRoomOnly` is deleted, so change it to `return 60` with a comment saying it
is a test stand-in that reports a lit room, not a scale constant.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/rooms/... -v -run 'UnlitCave|SkyFraction|LampJoins|Lantern|LightMod|Clamped'`
Expected: PASS, 7 tests

- [ ] **Step 5: Build the whole tree**

Run: `go build ./...`
Expected: SUCCESS. Any failure is a consumer of a deleted constant.

- [ ] **Step 6: Commit**

```bash
git add internal/rooms/lighting.go internal/rooms/lighting_model_test.go internal/rooms/rooms.go
git commit -m "feat(rooms): LightLevel composes the real light model

Deletes legacyVisibility and the three constants that mapped the old
three-value model. Room light is now the sky attenuated by this room's
fraction, plus the room's lamp, plus anyone carrying a light, all on one
logarithmic combine.

A room with no sky receives no sky term, which is not the same as a term of
zero: on a log scale two terms at zero combine to one step above zero, so a
zero sky would make a cave brighter for having a sky it does not have.

Weather attenuates the SKY only, because a blizzard does not dim a lantern.

Adds Room.IsLit, the predicate plan 1's design promised and never built.
Fifteen sites were hand-rolling the comparison, each copying a 424-field
struct at a measured 99.75 ns to read one int.

The -2 to 2 LightMod vocabulary is bridged rather than retired here: a +1
mutator lands exactly on LightDimBelow and +2 one step above, so the 31 Crash
Site Interior and 12 Foldweave rooms a static lightmod: 2 holds lit today stay
fully visible. Plan 4 retires the vocabulary.

Carried lights lift the room to the bottom of the perfect band, matching what
the old model's cancel-the-darkness rule effectively did. Plan 5 gives them
real magnitudes that scale from stat and skill.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: Collapse the fifteen hand-rolled lit checks

**Files:**
- Modify: 15 call sites listed below

- [ ] **Step 1: List them**

Run standalone:

```bash
grep -rn "LightLevel() >= int(configs.GetBalanceConfig().LightBlindBelow)\|LightLevel() < int(configs.GetBalanceConfig().LightBlindBelow)" --include=*.go internal/ modules/
```

Expected, 15 lines across: `actions/plant.go:341`, `actions/search.go:237`,
`actions/shadow.go:164`, `actions/sneak.go:78`, `actions/steal.go:444`,
`actions/track.go:275`, `hooks/Death_MobBroadcast.go:56`,
`hooks/Death_MobLoot.go:124`, `hooks/MobIdle_HandleIdleMobs.go:115`,
`hooks/NewRound_DoCombat_helpers.go:422`, `hooks/NewRound_UserRoundTick.go:211`,
`mobcommands/darkness.go:32`, `usercommands/get.go:76`, `usercommands/go.go:550`,
`usercommands/loot.go:25`, `usercommands/skill.skullduggery.shadow.go:136`.

⚠️ `usercommands/get.go:76` and `usercommands/loot.go:25` additionally test
`!user.Character.HasFlagFromAnySource(conditions.NightVision)`. **Keep that
clause.** It is a separate condition and this task only replaces the light half.

⚠️ `actions/sneak.go:78` reads `cfg.LightBlindBelow` from a `cfg` it already
holds. Replace the comparison but check whether `cfg` is still used afterwards;
if not, remove the now-dead read.

- [ ] **Step 2: Replace each**

`roomLit := room.LightLevel() >= int(configs.GetBalanceConfig().LightBlindBelow)`
becomes
`roomLit := room.IsLit()`

and
`if room.LightLevel() < int(configs.GetBalanceConfig().LightBlindBelow) {`
becomes
`if !room.IsLit() {`

Remove any `configs` import left unused. `go build ./...` will tell you.

- [ ] **Step 3: Verify none survive**

Run standalone:

```bash
grep -rn "LightLevel() >= int(configs.GetBalanceConfig()\|LightLevel() < int(configs.GetBalanceConfig()" --include=*.go internal/ modules/
```
Expected: no output.

- [ ] **Step 4: Run the full suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add -u
git commit -m "refactor(lighting): fifteen hand-rolled lit checks collapse onto Room.IsLit

Each site was copying a 424-field struct at a measured 99.75 ns to read one
int, and several sit inside per-round loops. Plan 1's design said these
consumers would migrate to a named predicate that says what they actually
mean; they migrated to a hand-rolled pair instead. This finishes that.

get.go and loot.go keep their separate NightVision clause, which is not part
of the light half.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: Prove the diff's shape before re-recording the golden

Plan 2's discipline: a golden that moves must have its diff's **shape** proven
before it is re-recorded. A blanket re-record proves nothing.

**Files:**
- Modify: `testdata/lighting_daycycle.golden` (re-recorded)
- Modify: `testdata/lighting_parity.golden` (re-recorded)

- [ ] **Step 1: Capture the current failure**

Run: `go test . -run 'TestLightingDayCycleAcrossSampleRounds' -v 2>&1 | head -40`
Expected: FAIL, the golden moved. This is intended.

- [ ] **Step 2: Produce the diff and read its shape**

```bash
cp testdata/lighting_daycycle.golden /tmp/daycycle.before
go test . -run TestLightingDayCycleAcrossSampleRounds -update-lighting-daycycle
diff /tmp/daycycle.before testdata/lighting_daycycle.golden > /tmp/daycycle.diff
wc -l /tmp/daycycle.diff
```

- [ ] **Step 3: Assert four properties of the diff before accepting it**

Run each standalone and record the answer in the PR description.

**(a) Midwinter midnight got darker in open country, not brighter.** Extract the
`land` rooms at midwinter-midnight from both files and confirm every one fell.

```bash
awk '/^== midwinter-midnight/{f=1;next} /^== /{f=0} f && /biome=land/' /tmp/daycycle.before | head -3
awk '/^== midwinter-midnight/{f=1;next} /^== /{f=0} f && /biome=land/' testdata/lighting_daycycle.golden | head -3
```
Expected: before shows `light=70` (lit biome, no night effect), after shows a
value **below 25**. If any `land` room is still at 70 at midwinter midnight, the
biome edit did not take.

**(b) Caves did not change.** A cave has no sky in either model.

```bash
awk '/^== equinox-noon/{f=1;next} /^== /{f=0} f && /biome=cave/' /tmp/daycycle.before | sort > /tmp/cave.before
awk '/^== equinox-noon/{f=1;next} /^== /{f=0} f && /biome=cave/' testdata/lighting_daycycle.golden | sort > /tmp/cave.after
diff /tmp/cave.before /tmp/cave.after | head
```
Expected: the 31 Crash Site rooms and 12 Foldweave rooms move from 70 to their
bridged value (58); the remaining ~77 cave rooms stay at 0. **If any cave room
rose above zero without a `lightmod` mutator, the sky fraction is not being
honoured.**

**(c) Noon varies by season and midnight varies by nothing else.**

```bash
for m in midwinter equinox midsummer; do
  printf "%s noon: " $m
  awk -v m="$m" '$0 ~ "^== "m"-noon"{f=1;next} /^== /{f=0} f && /biome=farmland/' testdata/lighting_daycycle.golden | head -1
done
```
Expected: roughly `light=62`, `light=70`, `light=73`. These are the derived noon
values; if they are equal, latitude is not reaching `SunLight`.

**(d) Nothing reached the dazzle band.**

```bash
grep -c "light=7[5-9]\|light=[89][0-9]\|light=100" testdata/lighting_daycycle.golden
```
Expected: `0`. Run standalone; `grep -c` exits 1 on zero matches.

- [ ] **Step 4: Re-record the parity golden and state plainly why it no longer guards**

Run: `go test . -run TestLightingParityAcrossEveryShippedRoom -update-lighting-parity`

Then add a comment at the top of `lighting_parity_golden_test.go` explaining that
this golden's guarantee is retired: it was plan 1's behaviour-preservation proof
and plan 3 is a deliberate behaviour change, so `lighting_daycycle.golden` is now
the guard and this one records the post-change state only.

- [ ] **Step 5: Verify both pass**

Run: `go test . -run 'TestLighting' -v`
Expected: PASS, both goldens.

- [ ] **Step 6: Commit**

```bash
git add testdata/ lighting_parity_golden_test.go
git commit -m "test(lighting): re-record both goldens, with the diff's shape proven first

Plan 2's discipline: a golden that moves has its diff's shape proven before it
is re-recorded, because a blanket re-record proves nothing.

Four properties checked and recorded in the PR description: open country falls
below the blind edge at midwinter midnight where it read 70 before; caves are
unchanged except the 43 rooms a static lightmod holds lit, which land on their
bridged value; noon reads about 62 / 70 / 73 across midwinter, equinox and
midsummer, which is latitude reaching SunLight; and nothing anywhere reaches
the dazzle band.

The parity golden's guarantee is explicitly retired. It was plan 1's
behaviour-preservation proof and plan 3 is a deliberate behaviour change, so
the day-cycle golden is now the guard and the comment at the top of its test
says so.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 11: Prove the new golden can fail

A golden that cannot fail is worse than no golden, because it reads as a
guarantee. Plan 1 sabotaged in both directions; do the same.

**Files:** none committed. This task produces evidence only.

- [ ] **Step 1: Sabotage the latitude**

Temporarily change the `WorldLatitude` default in
`internal/configs/config.balance.lighting.go` from `46.5` to `0`.

Run: `go test . -run TestLightingDayCycleAcrossSampleRounds`
Expected: **FAIL**, with seasonal noon values collapsing to one number.

Revert the change.

- [ ] **Step 2: Sabotage a sky fraction**

Temporarily change `_datafiles/world/dogmud/biomes/cave.yaml` `skylight` from
`0.0` to `1.0`.

Run: `go test . -run TestLightingDayCycleAcrossSampleRounds`
Expected: **FAIL**, with ~120 cave rooms changing.

Revert.

- [ ] **Step 3: Sabotage the doubling step**

Temporarily change `LightDoublingStep`'s default from `8` to `16`.

Run: `go test . -run TestLightingDayCycleAcrossSampleRounds`
Expected: **FAIL**.

Revert.

- [ ] **Step 4: Confirm the tree is clean and green**

Run standalone: `git status --short`
Expected: no modifications.

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 5: Record the evidence**

Write the three sabotage results into the PR description under a "Null probe"
heading. **If any sabotage did not redden the golden, the golden is not guarding
what this plan claims and must be fixed before merge.**

---

### Task 12: Documentation and the patch note

**Files:**
- Modify: `internal/rooms/context.md`
- Modify: `internal/gametime/context.md`
- Modify: `internal/configs/context.md`
- Create: a patch note, following the format plan 2's patch note used
- Modify: `docs/README.md`

- [ ] **Step 1: Update `internal/gametime/context.md`**

Add a section describing `celestial.go`: that `WorldLatitude` is the only
seasonal input, that `NightHoursAt`, `SunLight`, `MoonLight` and `CelestialLight`
are the surface, that the sun is Absent below the horizon rather than zero, and
the trap that **a test binary has `NightHours: 0` and a bare `Balance` has
`WorldLatitude: 0`, so any test touching night must pin both.**

- [ ] **Step 2: Update `internal/rooms/context.md`**

Replace any description of `darkarea` / `litarea` with `skylight` and `lamp`,
note that both are pointers because zero is meaningful, note the room-level
override, and record that `legacyVisibility` is gone.

- [ ] **Step 3: Update `internal/configs/context.md`**

Add `GetLightingConfig` and say why it exists: 99.75 ns against 8.23 ns, fifteen
call sites, per-round loops.

- [ ] **Step 4: Verify every symbol named in those files exists**

Run: `python tools/context_md_audit.py`
Expected: no findings for `rooms`, `gametime`, `configs`, `lightscale`.

- [ ] **Step 5: Write the patch note**

This plan is player-visible and owes one. It must say, in the project's player
copy style (80-character wrap, no raw numbers for durations or damage, ESL-clear):

- Night is now longer in winter and shorter in summer.
- Moonlight matters: a bright night in the open is navigable, a moonless one is not.
- Forests, swamps and caves are dark in a way open country is not.
- Town streets stay usable at night, but back lanes hide faces.
- Nothing about the light *scale*, the doubling step or latitude. Players do not
  read config knobs.

⚠️ **Do not mention dazzle.** It has no mechanical effect until plan 5, and the
amendment spec rules that describing a penalty the game does not apply is a lie
to the player.

- [ ] **Step 6: Add the plan and patch note to `docs/README.md`**

- [ ] **Step 7: Full verification before the PR**

Run: `gofmt -l internal/ modules/`
Expected: no output. If a file is listed, check it is not the Windows CRLF false
positive: `file <path>` and `tr -cd '\r' < <path> | wc -c`.

Run: `go vet ./...`
Expected: no output.

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add -u
git add docs/
git commit -m "docs(lighting): context.md for the celestial model, and the patch note

Records the trap that will bite the next test author: a Go test binary has
NightHours 0 and a bare Balance has WorldLatitude 0, so any test touching
night must pin BOTH or it is asserting on a world that has no night and no
seasons.

The patch note says what a player experiences and nothing about the scale, the
doubling step or the latitude, and deliberately does not mention dazzle, which
has no mechanical effect until plan 5.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Before opening the PR

Every `gh` command carries `--repo pruuk/DOGMud`. This repo is a fork and `gh`
defaults to the parent; a bare `gh pr create` once opened a PR on upstream.

Never `git add -A` or `git add .`. Named paths only.

The PR description must carry:
- The four diff-shape properties from Task 10 Step 3, with their measured values.
- The three sabotage results from Task 11.
- A plain statement that the parity golden's guarantee is retired and why.
- A note that plans 3b, 3c and 3d follow, and that the 117 orphan rooms and the
  477 city rooms are deliberately untouched here.

## Known state this PR deliberately leaves behind

These are not defects and must be in the PR description so a reviewer does not
file them:

- **The 117 orphan rooms are still on the synthetic `default` biome**, which has
  no sky fraction and therefore reads as fully open sky. They will darken at
  night, which is already better than being permanently lit, but 81 of them are
  a zone called `a_dark_forest` that should be under canopy. **Plan 3b.**
- **Every `city` room uses the side-lane lamp of 35**, so main streets and squares
  are dimmer than intended and the New Plymouth sewers are lit like a street.
  **Plan 3c.**
- **`fort` uses one middling sky fraction** for both its training yard and its
  buried vault. **Plan 3c.**
- **No transition notice fires** when a player walks between rooms of different
  light or stands still while dawn breaks. **Plan 3d.**
- **`house` is still its own biome** rather than folded into `interior`, and no
  `sewer`, `plains` or `river` biome exists. **Plan 3b.**
