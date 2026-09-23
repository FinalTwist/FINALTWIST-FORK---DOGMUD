# Graded Lighting Plan 2: Vision Windows Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the NightVision and InfraredVision boolean flag shortcuts with the window model, where an ability moves where an observer's usable band sits on the light scale instead of granting sight outright.

**Architecture:** A pure function maps (light, nightvision strength, infra reach) to a `SightDecision`. Strength rides on the existing condition `Effects` vocabulary (which already supports per-instance magnitude, so plan 5 can scale it from stat and skill) and on `MutationEffect.Value` (which already exists and is ignored for flag effects today). `ParticipantSight` reads both through one character accessor that takes the strongest held value.

**Tech Stack:** Go, `internal/messaging`, `internal/conditions`, `internal/mutations`, `internal/characters`, `internal/configs`, condition and mob YAML.

---

## Facts verified against source

Read from the tree on 2026-09-22 at master `7e200aa47`.

| # | Fact | Evidence |
|---|------|----------|
| 1 | Plan 1 shipped. `Room.LightLevel()` returns only `LightDark` 0, `LightRoomOnly` 60 or `LightFull` 70. Bands are `LightBlindBelow` 25, `LightDimBelow` 50, `LightExitsAbove` 65 | `internal/rooms/lighting.go`, `internal/configs/config.balance.lighting.go` |
| 2 | `ParticipantSight`'s two flag shortcuts sit AFTER the band switch, so they are only reached when light is below `LightBlindBelow` | `internal/messaging/predicates.go` |
| 3 | 🔑 `nightvision` has exactly ONE content path: condition 65, from item 30047 Cat's Eye Draught. 500 rounds, `strength: -10`, also grants `see-hidden`, alchemy `skill_minimum: 12`, sold by nobody | `items/consumables-30000/30047-cats_eye_draught.yaml:12-13`, `conditions/65-cats_eye_draught.yaml`, `recipes/alchemy/cats-eye-draught.yaml` |
| 4 | 🔴 Condition 29 "Night Vision" is granted by NOTHING. `infraredvision` (condition 85) is granted by NOTHING. No item, spell, mob, species or mutation | exhaustive grep of items, spells, mobs, mutations, species |
| 5 | `conditions.Flag` is a plain `string`. Flags are boolean, with no magnitude | `internal/conditions/conditionspec.go:29` |
| 6 | `HasFlagFromAnySource` checks conditions AND mutations, in that order | `internal/characters/conditions.go:27` |
| 7 | 🔑 `ConditionSpec.Effects` is a closed `map[EffectKind]EffectValue`. `Conditions.Effect(kind)` aggregates: multipliers multiply, `isCap()` kinds take the minimum, everything else SUMS. **There is no MAX mode** | `internal/conditions/effects.go:108-148` |
| 8 | 🔑 `EffectValue.UsesMagnitude` makes `Effect` read the RECORD's `Magnitude`, not the spec's literal, so per-instance scaling already works | `internal/conditions/effects.go:126-129` |
| 9 | 🔑 `ConditionSpec.ProgressMult` is the precedent for a numeric spec field: "0 means the default (2.0, the historic literal). The strongest held value wins." | `internal/conditions/conditionspec.go` |
| 10 | 🔑 `MutationEffect` is `{Type string, Target string, Value float64}`. `HasMutationFlag` matches `Type == "flag" && Target == flag` and **ignores `Value` entirely** | `internal/mutations/mutations.go:22-27`, `HasMutationFlag` |
| 11 | No dazzle threshold knob exists. Plan 1 omitted it deliberately because nothing would read it | grep `internal/configs` for `Dazzle`, empty |
| 12 | The existing `testdata/lighting_parity.golden` records 1386 rooms by 4 observer kinds (plain, nightvision via condition 29, infrared via condition 85, blinded via condition 3) | `lighting_parity_golden_test.go` |
| 13 | `AllEffectKinds` and `validateEffects` both enumerate the vocabulary, so a new kind must be added to both or validation silently misses it | `internal/conditions/effects.go:25`, `:74` |

### 🔴 The consequence this plan accepts, stated precisely

Because `LightLevel()` returns only 0, 60 and 70 (fact 1), the window model changes
**exactly one cell** of observable behaviour in shipped content:

| Light | Observer | Before this plan | After this plan |
|---|---|---|---|
| 0 | nightvision | `SightFull` | **`SightNone`** |
| 0 | infrared | `SightShapes` | `SightShapes` (unchanged) |
| 0 | plain, blinded | `SightNone` | `SightNone` (unchanged) |
| 60, 70 | every kind | as today | unchanged |

So a nightvision holder stops seeing in an unlit cave. That is the point of a
window that moves rather than widens, and the owner has accepted it. Nothing
deploys until the messaging arc and this whole arc are finished, so no player
experiences the gap before plan 3 makes dim rooms exist and plan 5 makes true
dark sight obtainable.

⚠️ **This means the plan 1 parity golden WILL move, deliberately, for the first
time.** Task 6 re-records it once and proves the diff's SHAPE: only
`nightvision` lines in dark rooms may change, and every `plain`, `blinded` and
`infrared` line must be byte identical. A blanket re-record without that proof
would discard the arc's only behaviour-preservation instrument.

---

## The window, precisely

From the spec's "Vision is a window, not a bonus". A normal observer:

| Light | Band | Result |
|---|---|---|
| at or above `LightDazzleAbove` (75) | too bright | `SightFull` |
| `LightDimBelow` (50) to 75 | perfect | `SightFull` |
| `LightBlindBelow` (25) to 50 | too dim | `SightShapes` |
| below 25 | blind | `SightNone` |

An ability **shifts every one of those edges down by its strength**, capped at
`LightWindowShiftCap` (24), and is blind below `LightWindowFloor` (1). So
strength 24 gives: perfect 26 to 51, shapes 1 to 26, blind below 1.

InfraredVision is nightvision plus a downward extension carrying a second
number, its **infra reach**. At or below `LightWindowFloor` the reach decides how
far down it still reads `SightShapes`: light at or above `-reach` reads shapes,
below that reads nothing. The two numbers are independent.

🔑 **Magical darkness gets its floor here, and it is emergent rather than a
knob.** The spec defines it as "any light below the reach of the strongest
infravision", so it is not a separate threshold to author: it falls out of the
reach comparison. Light below the negation of the best reach in the room reads
`SightNone` for everyone, which is the only state in which nothing sees at all.
Task 1's table pins it with the `infra reach bottoms out` row. Nothing in
shipped content can reach a negative light value until plan 5 adds a darkness
spell, so this is groundwork with unit coverage only, which is deliberate.

⚠️ **Dazzle carries NO mechanical penalty in this plan and NO
`LightDazzleAbove` knob ships as a balance lever.** It is needed as a band edge
to compute the window, so it ships as a constant in `internal/messaging`, not as
config. Plan 1's rule stands: a config knob that nothing reads does not ship.
When dazzle earns a penalty, the threshold becomes a knob in that plan.

---

## File structure

| File | Responsibility |
|---|---|
| `internal/messaging/window.go` (new) | the pure window function and its band constants. No config reads, no character reads |
| `internal/messaging/window_test.go` (new) | the ladder table |
| `internal/conditions/effects.go` | two new `EffectKind`s and the `isMax()` aggregation mode |
| `internal/mutations/mutations.go` | `FlagValue`, reading the `Value` that `HasMutationFlag` ignores |
| `internal/characters/vision.go` (new) | `NightVisionStrength()` and `InfraReach()`, taking the strongest across conditions and mutations |
| `internal/configs/config.balance.lighting.go` | the default-strength knob and the shift cap |
| `internal/messaging/predicates.go` | `ParticipantSight` calls the window instead of the flag shortcuts |
| `_datafiles/world/dogmud/conditions/65-cats_eye_draught.yaml` | reworded, and given an explicit strength |
| `_datafiles/world/dogmud/conditions/29-night_vision.yaml` | given an explicit strength |
| `_datafiles/world/dogmud/conditions/85-infraredvision.yaml` | given a strength and a reach |

---

## Task 1: The window as a pure function

Start here because everything else is wiring. This task reads no config and no
character, so its test is a plain table with no fixtures.

**Files:**
- Create: `internal/messaging/window.go`
- Create: `internal/messaging/window_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/messaging/window_test.go`:

```go
package messaging

import "testing"

// TestSightThroughWindow is the whole window model in one table. Every row is
// a light value and an observer's two numbers; the expected value is the tier
// that observer reads.
//
// The rows are taken from the spec's ladder and from the band edges, because a
// band model is wrong at its edges long before it is wrong in the middle.
func TestSightThroughWindow(t *testing.T) {
	// A normal observer: no shift, no reach.
	const normal, noReach = 0, 0

	tests := []struct {
		name     string
		light    int
		strength int
		reach    int
		want     SightDecision
	}{
		// Normal observer, unshifted bands 25 / 50 / 75.
		{"normal, pitch dark", 0, normal, noReach, SightNone},
		{"normal, just below blind", 24, normal, noReach, SightNone},
		{"normal, bottom of dim", 25, normal, noReach, SightShapes},
		{"normal, top of dim", 49, normal, noReach, SightShapes},
		{"normal, bottom of perfect", 50, normal, noReach, SightFull},
		{"normal, top of perfect", 74, normal, noReach, SightFull},
		{"normal, dazzled still sees", 75, normal, noReach, SightFull},
		{"normal, dazzled hard", 100, normal, noReach, SightFull},

		// Max nightvision: every edge drops 24, floor at 1.
		{"nv24, pitch dark is blind", 0, 24, noReach, SightNone},
		{"nv24, at the floor reads shapes", 1, 24, noReach, SightShapes},
		{"nv24, top of shifted dim", 25, 24, noReach, SightShapes},
		{"nv24, bottom of shifted perfect", 26, 24, noReach, SightFull},
		{"nv24, top of shifted perfect", 50, 24, noReach, SightFull},
		{"nv24, dazzled in daylight", 70, 24, noReach, SightFull},

		// The cap: strength above the cap behaves as the cap.
		{"strength above cap clamps", 26, 99, noReach, SightFull},

		// Negative strength is nonsense and must not widen the window.
		{"negative strength behaves as zero", 25, -50, noReach, SightShapes},

		// Infra reach only matters at or below the floor.
		{"infra reach reads an unlit cave", 0, 24, 30, SightShapes},
		{"infra reach reads shallow magical dark", -30, 24, 30, SightShapes},
		{"infra reach bottoms out", -31, 24, 30, SightNone},
		{"reach does not help above the floor", 24, 0, 30, SightNone},
		{"reach without strength still reads dark", 0, 0, 10, SightShapes},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SightThroughWindow(tc.light, tc.strength, tc.reach)
			if got != tc.want {
				t.Errorf("SightThroughWindow(light=%d, strength=%d, reach=%d) = %v, want %v",
					tc.light, tc.strength, tc.reach, got, tc.want)
			}
		})
	}
}
```

⚠️ **The last row is the one to think about.** A creature with reach but no
nightvision strength still senses heat in the dark. Fact: the spec says the two
numbers are independent, so reach must not require strength.

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/messaging/ -run TestSightThroughWindow -v`

Expected: build failure, `undefined: SightThroughWindow`.

- [ ] **Step 3: Implement it**

Create `internal/messaging/window.go`:

```go
package messaging

// The normal observer's band edges on the graded light scale, and the two
// numbers that bound how far an ability may move them.
//
// These are CONSTANTS, not config knobs, and that is deliberate. Plan 1's rule
// is that a config knob nothing reads does not ship. windowDazzleEdge is needed
// to compute the window's upper edge but carries no mechanical penalty in this
// plan, so exposing it as a balance lever would ship a knob an operator could
// turn with no observable effect. It becomes a knob in the plan that gives
// dazzle teeth.
//
// The lower two edges DO have config knobs already (LightBlindBelow and
// LightDimBelow, shipped by plan 1) and the window function takes them as
// arguments rather than reading config, so this file stays pure and testable.
const (
	// windowDazzleEdge is where the perfect band ends and too-bright begins.
	windowDazzleEdge = 75
	// windowShiftCap is the most any ability may move the window down.
	windowShiftCap = 24
	// windowFloor is the light below which a shifted window reads nothing,
	// no matter how strong. Only an infra reach sees past it.
	windowFloor = 1
)

// SightThroughWindow reports what an observer reads at a given light level.
//
// strength moves every band edge DOWN by that many points, capped at
// windowShiftCap and floored at zero, so an ability trades bright-light comfort
// for dark-light acuity rather than simply gaining sight. reach is the separate
// heat-sensing extension that operates only at or below windowFloor; light at
// or above the negation of reach reads shapes, below that reads nothing.
//
// It takes the two lower band edges as arguments rather than reading config, so
// it stays a pure function with no locks and no global state. Its caller owns
// the config read.
func SightThroughWindow(light, strength, reach int, blindBelow, dimBelow int) SightDecision {
	if strength < 0 {
		strength = 0
	}
	if strength > windowShiftCap {
		strength = windowShiftCap
	}
	if reach < 0 {
		reach = 0
	}

	shiftedBlind := blindBelow - strength
	shiftedDim := dimBelow - strength

	if light >= shiftedDim {
		// Perfect and too-bright both read fully. Dazzle has no mechanical
		// penalty in this plan, so the upper edge is not consulted yet; it is
		// declared above so the next plan has one place to add the penalty.
		return SightFull
	}
	if light >= shiftedBlind && light >= windowFloor {
		return SightShapes
	}
	// Below the shifted window, and at or below the floor. Only heat-sensing
	// reaches further down.
	//
	// The floor gate is load-bearing. Without it, a DIM room (light 24,
	// strength 0, so the shape check fails at the unshifted blind edge of 25)
	// would fall through to `light >= -reach`, which any positive reach
	// satisfies, and a heat sense would wrongly grant shapes in ordinary
	// gloom. The spec is explicit: "at or below LightWindowFloor the reach
	// decides".
	if light <= windowFloor && reach > 0 && light >= -reach {
		return SightShapes
	}
	return SightNone
}
```

⚠️ **This floor gate was MISSING from this plan's first draft and the bug was
caught during Task 1's execution.** The row `{"reach does not help above the
floor", 24, 0, 30, SightNone}` is the row that catches it: without the gate the
function returns `SightShapes` there. The row was always correct; the reference
code above was not. Corrected here so plans 3 and 5 do not copy it.

⚠️ **The test in Step 1 calls `SightThroughWindow` with three arguments and this
signature takes five.** That is deliberate: write the test first, watch it fail,
then reconcile. Update the test's call to pass the band edges explicitly:

```go
got := SightThroughWindow(tc.light, tc.strength, tc.reach, 25, 50)
```

Pass the literals 25 and 50 in the test, NOT `configs.GetBalanceConfig()`. The
point of a pure function is that its test does not depend on shipped config.

- [ ] **Step 4: Run the test**

Run: `go test ./internal/messaging/ -run TestSightThroughWindow -v`

Expected: PASS on all rows. If `{"nv24, at the floor reads shapes", 1, 24, ...}`
fails, check the `light >= windowFloor` term: at strength 24 the shifted blind
edge is 1, so light 1 must satisfy both conditions.

- [ ] **Step 5: Commit**

```bash
git add internal/messaging/window.go internal/messaging/window_test.go
git commit -F - <<'EOF'
feat(messaging): the vision window, as a pure function

An ability does not add light to a room. It moves where the observer's
usable band sits on the scale. This is that function and nothing else:
no config reads, no character reads, no locks, so its test is a plain
table with no fixtures.

The band edges arrive as arguments rather than being read from config,
which keeps the model testable at its edges. A band model is wrong at
its edges long before it is wrong in the middle, so the table pins every
edge and both clamps.

The dazzle edge is a constant here, not a config knob. It is needed to
compute the window but carries no penalty in this plan, and plan 1's
rule is that a knob nothing reads does not ship.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 2: A MAX aggregation mode for effects

The window needs the STRONGEST held strength, not the sum. Fact 7 says
`Conditions.Effect` sums by default, which would make two nightvision sources
stack into a wider window than either grants.

**Files:**
- Modify: `internal/conditions/effects.go`
- Test: `internal/conditions/effects_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/conditions/effects_test.go`:

```go
// TestEffectMaxKindTakesStrongestNotSum pins the aggregation mode the vision
// window needs. Two nightvision sources must not stack into a wider window
// than the better one grants, which is what the default summing behaviour
// would do.
func TestEffectMaxKindTakesStrongestNotSum(t *testing.T) {
	if !EffectNightVisionStrength.isMax() {
		t.Fatalf("EffectNightVisionStrength must aggregate as MAX, or two sources would stack")
	}
	if !EffectInfraReach.isMax() {
		t.Fatalf("EffectInfraReach must aggregate as MAX")
	}
	// A summing kind must not have become a max kind by accident.
	if EffectMitigationFlat.isMax() {
		t.Fatalf("EffectMitigationFlat must keep summing")
	}
}
```

⚠️ `isMax` is unexported, so this test must live in package `conditions`, not
`conditions_test`. Check which package the existing `effects_test.go` declares
and match it. If it is an external test package, add this test to a file that
declares the internal package instead and say which you chose.

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/conditions/ -run TestEffectMaxKindTakesStrongestNotSum -v`

Expected: build failure, `undefined: EffectNightVisionStrength`.

- [ ] **Step 3: Add the kinds and the mode**

In `internal/conditions/effects.go`, add to the `EffectKind` const block:

```go
	// EffectNightVisionStrength is how far DOWN the scale an observer's usable
	// light band shifts. Aggregated as MAX, not summed: two night-sight
	// sources do not stack into a wider window than the better one grants.
	EffectNightVisionStrength EffectKind = `nightvision_strength`
	// EffectInfraReach is how far BELOW the window floor heat-sensing still
	// reads shapes. Independent of strength: a creature can sense heat deeply
	// while being no better than anyone else at using faint light.
	EffectInfraReach EffectKind = `infra_reach`
```

Add both to `AllEffectKinds` (fact 13: `validateEffects` enumerates it, so a
kind missing from this slice is silently unvalidated).

Add the mode, beside `isCap`:

```go
// isMax reports whether this kind aggregates by taking the strongest held
// value. Used by the vision window, where summing would let two abilities
// stack into a window wider than either one grants.
func (k EffectKind) isMax() bool {
	return k == EffectNightVisionStrength || k == EffectInfraReach
}
```

Then add the branch to `Effect`. Insert the max accumulator beside the existing
three, and a case BEFORE the `default:` that currently sums:

```go
	maxValue := 0.0
```

```go
		case kind.isMax():
			if val > maxValue {
				maxValue = val
			}
```

and in the trailing switch:

```go
	case kind.isMax():
		return maxValue
```

⚠️ **Read the existing `Effect` body before editing it and place the new case
inside the SAME `switch` the existing cases use.** Adding a second switch would
leave the `default: sum += val` arm reachable for the new kinds, which would
sum and max simultaneously.

- [ ] **Step 4: Run the test and the package**

Run: `go test ./internal/conditions/ -run TestEffectMaxKindTakesStrongestNotSum -v`
Expected: PASS.

Run: `go test ./internal/conditions/`
Expected: `ok`. If `validateEffects` tests fail, the new kinds are missing from
`AllEffectKinds`.

- [ ] **Step 5: Prove the max actually maxes**

The test above checks the mode flag, not the arithmetic. Add a second test that
builds a `Conditions` holding two records declaring the same kind at different
values and asserts `Effect` returns the larger, not the sum. Read how existing
tests in this file construct a `Conditions` with held records and follow that
pattern exactly rather than inventing a fixture.

Then sabotage it: change `isMax` to return false for
`EffectNightVisionStrength`, confirm the arithmetic test goes red reporting a
summed value, and restore.

Paste the red output in your report.

- [ ] **Step 6: Commit**

```bash
git add internal/conditions/effects.go internal/conditions/effects_test.go
git commit -F - <<'EOF'
feat(conditions): effects can aggregate by strongest held value

The vision window needs the best night-sight an observer holds, not the
sum of every source. Summing would let two abilities stack into a window
wider than either one grants, which is the opposite of a window that
moves rather than widens.

Two kinds join the vocabulary: nightvision_strength, how far down the
scale the usable band shifts, and infra_reach, how far below the floor
heat-sensing still reads shapes. They are independent numbers on purpose.

Riding the existing Effects vocabulary rather than adding new spec fields
buys per-instance magnitude for free, which is what lets plan 5 scale a
spell's strength from stat and skill.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 3: Mutations can carry a strength too

Fact 6 says `HasFlagFromAnySource` reads mutations as well as conditions. Fact
10 says `MutationEffect` already has a `Value float64` that `HasMutationFlag`
ignores. If only conditions could carry strength, a mutation could grant the
flag but no window shift, which is the sibling-inconsistency this project's
rules forbid.

**Files:**
- Modify: `internal/mutations/mutations.go`
- Test: `internal/mutations/mutations_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/mutations/mutations_test.go`:

```go
// TestFlagValueReadsTheValueHasMutationFlagIgnores pins that a mutation
// granting a vision flag can also declare how strong it is. HasMutationFlag
// answers only yes or no; the window model needs a number, and MutationEffect
// has carried an unused Value field all along.
func TestFlagValueReadsTheValueHasMutationFlagIgnores(t *testing.T) {
	// Build the owned map and specs the way this package's existing tests do.
	// Read them first and match the fixture style rather than inventing one.
	owned := map[string]int{"testmut-nightsight": 1}

	got := FlagValue(owned, "nightvision")
	if got != 18 {
		t.Errorf("FlagValue = %v, want 18 (the Value on the flag effect)", got)
	}

	// A flag the mutation does not grant reads zero, not a default.
	if v := FlagValue(owned, "infraredvision"); v != 0 {
		t.Errorf("FlagValue for an ungranted flag = %v, want 0", v)
	}
}
```

⚠️ **This test needs a registered mutation spec carrying
`{Type: "flag", Target: "nightvision", Value: 18}`.** Read how
`internal/mutations`' existing tests register or seed specs (look for a seed
helper, a test-only registry setter, or a `LoadDataFiles` call) and use that
mechanism. **Do NOT author a new YAML file under `_datafiles/` for a unit
test.** If there is no seam for registering a spec in a test, say so and report
it rather than inventing one.

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/mutations/ -run TestFlagValueReadsTheValue -v`

Expected: build failure, `undefined: FlagValue`.

- [ ] **Step 3: Implement it**

Add to `internal/mutations/mutations.go`, beside `HasMutationFlag`:

```go
// FlagValue returns the strongest Value declared on any owned mutation's flag
// effect matching flag, or 0 if no owned mutation grants it.
//
// HasMutationFlag answers whether the flag is present. This answers how
// strongly, which the vision window needs: a mutation that grants night sight
// declares how far it shifts the observer's band on the same effect entry that
// grants the flag. MutationEffect has carried Value since it was written and
// flag effects have always ignored it.
//
// Strongest rather than summed, matching the MAX aggregation the condition side
// uses, so two mutations granting the same flag do not stack.
func FlagValue(owned map[string]int, flag string) float64 {
	best := 0.0
	for id := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, e := range append(append([]MutationEffect{}, spec.Pros...), spec.Cons...) {
			if e.Type == "flag" && e.Target == flag && e.Value > best {
				best = e.Value
			}
		}
	}
	return best
}
```

⚠️ **Read `HasMutationFlag` and match how it walks `Pros` then `Cons`.** If it
iterates them as two separate loops, do the same rather than concatenating, to
keep the two functions obviously parallel. The concatenation above allocates on
every call and this is reached from a hot path; prefer two loops.

- [ ] **Step 4: Run the test and the package**

Run: `go test ./internal/mutations/ -run TestFlagValueReadsTheValue -v`
Expected: PASS both assertions.

Run: `go test ./internal/mutations/`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/mutations/mutations.go internal/mutations/mutations_test.go
git commit -F - <<'EOF'
feat(mutations): a flag effect's Value is no longer ignored

HasMutationFlag answers whether a mutation grants a flag. The vision
window needs to know how strongly, so FlagValue reads the Value field
MutationEffect has carried all along and flag effects have always
discarded.

Strongest wins rather than summing, matching the condition side, so two
mutations granting night sight do not stack into a wider window than
either grants.

Without this a mutation could grant the flag but no window shift, and
the two grant paths would disagree about what the flag means.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 4: The default strength knob

Fact 4 says condition 29 grants the flag with no strength. Under Tasks 2 and 3
that reads as strength 0, which shifts the window by nothing, so the flag would
mean literally nothing. Fact 9 gives the established answer: a numeric field
where 0 means "use the default".

**Files:**
- Modify: `internal/configs/config.balance.lighting.go`
- Modify: `internal/configs/config.balance.go`
- Test: `internal/configs/config_lighting_thresholds_test.go`

- [ ] **Step 1: Add the knob**

Add one `ConfigInt` beside the plan 1 lighting knobs in
`internal/configs/config.balance.go`, following that block's existing comment
style:

```go
	LightDefaultVisionStrength ConfigInt `yaml:"LightDefaultVisionStrength"` // Window shift for a vision flag that declares no strength of its own (default 12)
```

Document WHY in the struct comment: a flag with no declared strength must still
mean something, and 12 is deliberately mid-range so an authored source can be
clearly better or clearly worse than a bare flag.

- [ ] **Step 2: Default and validate it**

In `validateLighting()`, default it to 12 and clamp it to `[0, 24]`, since a
shift above the window cap is meaningless and a negative shift is nonsense.
**Read the existing function first** and follow the zero-coercion reasoning
already written there, including the comment explaining why zero coerces.

⚠️ Do NOT reuse the hardcoded-fallback pattern that had to be fixed in
`0a3274dfc`. If the clamp interacts with another knob, re-check the final value
after assignment rather than trusting a literal.

- [ ] **Step 3: Write the tests**

Follow `config_lighting_thresholds_test.go`'s existing style exactly: a zero
defaults correctly case, an authored valid value survives case, and both clamp
boundaries (0, 24 accepted; -1 and 25 rejected). Prove each can fail by flipping
the comparison and watching the right case redden.

- [ ] **Step 4: Run**

Run: `go test ./internal/configs/ -v -run Lighting`
Expected: every case passes, including the plan 1 cases.

Run: `go build ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.lighting.go internal/configs/config_lighting_thresholds_test.go
git commit -F - <<'EOF'
feat(config): a default window shift for a bare vision flag

Condition 29 grants the nightvision flag and declares no strength. Read
literally that is a shift of zero, which would make the flag mean
nothing at all. This knob is what a bare flag falls back to.

Twelve is deliberately mid-range so an authored source can be clearly
better or clearly worse than an unadorned flag, rather than every source
being identical or every bare flag being useless.

Follows ProgressMult's established idiom: a numeric field where zero
means the default rather than zero.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 5: One accessor per observer

**Files:**
- Create: `internal/characters/vision.go`
- Test: `internal/characters/vision_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/characters/vision_test.go` with a test that covers, for a
character built by `characters.New()`:

1. No vision flag at all reads strength 0 and reach 0.
2. A flag present with no declared strength reads the config default (pin the
   config with `configs.SetConfigForTest`, following
   `internal/rooms/lighting_test.go`'s precedent, and do NOT rely on ambient
   defaults).
3. A condition declaring an explicit strength reads that strength.
4. A condition and a mutation both granting it read the STRONGER of the two,
   not the sum.

Write the fixtures the way this package's existing condition tests do. Read them
first. Condition 3 blinding a character via `AddCondition` is the proven pattern
from plan 1's Task 1.

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/characters/ -run TestVision -v`

Expected: build failure, `c.NightVisionStrength undefined`.

- [ ] **Step 3: Implement it**

Create `internal/characters/vision.go`:

```go
package characters

import (
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// NightVisionStrength reports how far DOWN the light scale this character's
// usable band shifts, in scale points.
//
// Zero means no night sight at all. A character holding a vision flag that
// declares no strength of its own falls back to the configured default, so a
// bare flag still means something; see LightDefaultVisionStrength.
//
// The strongest source wins across conditions and mutations alike. It is never
// summed: the window MOVES rather than widening, so two abilities cannot
// combine into a window wider than the better one grants.
func (c *Character) NightVisionStrength() int {
	return c.bestVisionNumber(conditions.EffectNightVisionStrength, conditions.NightVision, true)
}

// InfraReach reports how far BELOW the window floor this character still reads
// shapes by sensing heat. Zero means not at all.
//
// Independent of NightVisionStrength on purpose: a creature can sense heat
// deeply while being no better than anyone else at using faint light.
func (c *Character) InfraReach() int {
	return c.bestVisionNumber(conditions.EffectInfraReach, conditions.InfraredVision, false)
}

// bestVisionNumber is the shared body. effectKind is the numeric channel,
// flag is the boolean channel, and defaultOnBareFlag says whether a flag
// carrying no number falls back to the configured default.
//
// Only nightvision defaults on a bare flag. A bare infrared flag reads reach 0,
// because "senses heat" with no stated range has no sensible fallback, whereas
// "sees in the dark" plainly means at least a little.
func (c *Character) bestVisionNumber(effectKind conditions.EffectKind, flag conditions.Flag, defaultOnBareFlag bool) int {
	best := int(c.Conditions.Effect(effectKind))

	if m := int(mutations.FlagValue(c.Mutations, string(flag))); m > best {
		best = m
	}

	if best == 0 && defaultOnBareFlag && c.HasFlagFromAnySource(flag) {
		best = int(configs.GetBalanceConfig().LightDefaultVisionStrength)
	}

	return best
}
```

⚠️ **Verify `c.Conditions` and `c.Mutations` are the real field names and types
before relying on them**, and confirm `internal/characters` may import
`internal/configs` without a cycle. Plan 1 proved `internal/messaging` can; this
is a different package. Run `go list -deps ./internal/configs` and confirm it
does not reach `internal/characters`. Report the result either way.

- [ ] **Step 4: Run the test and the package**

Run: `go test ./internal/characters/ -run TestVision -v`
Expected: PASS all four cases.

Run: `go test ./internal/characters/`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/vision.go internal/characters/vision_test.go
git commit -F - <<'EOF'
feat(characters): one door for an observer's two vision numbers

NightVisionStrength and InfraReach are what the window model asks of a
character. Both take the strongest value across conditions and mutations
rather than summing, because the window moves rather than widening.

Only nightvision falls back to a default on a bare flag. A bare infrared
flag reads reach zero: "senses heat" with no stated range has no sensible
fallback, while "sees in the dark" plainly means at least a little.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## 🔴 ORDERING CORRECTION, found during execution

**Task 7 must run BEFORE Task 6.** The plan as first written had the switch
flipped before the strengths were authored, which produces TWO golden
re-records instead of one and makes Task 6's shape check false.

Why: condition 85 (InfraredVision) has no `infra_reach` until Task 7 authors it.
If Task 6 runs first, infrared's reach is 0, so at light 0 the window returns
`SightNone` where the old flag shortcut returned `SightShapes`. That is a
regression Task 7 would then revert, so the golden would move twice and Task 6's
claim that "only nightvision lines change" would be wrong.

Authoring first is safe precisely because the authored numbers are INERT until
`ParticipantSight` reads them, which only happens in Task 6. So Task 7 moves
nothing, and Task 6 then produces exactly one re-record in which only
`nightvision` lines change.

Execute in the order: 1, 2, 3, 4, 5, **7, 6**, 8, 9.

With condition 29 authored at `nightvision_strength: 18` and condition 85 at
`nightvision_strength: 12` plus `infra_reach: 30`, the expected Task 6 diff is:

| Light | Observer | Before | After |
|---|---|---|---|
| 0 | nightvision (strength 18, shifted blind 7) | `SightFull` | **`SightNone`** |
| 0 | infrared (reach 30) | `SightShapes` | `SightShapes` (unchanged) |
| 0 | plain, blinded | `SightNone` | `SightNone` (unchanged) |
| 60, 70 | every kind | as today | unchanged |

---

## Task 6: Replace the flag shortcuts, and re-record the golden ONCE

This is the behaviour change. Everything before it was inert.

**Files:**
- Modify: `internal/messaging/predicates.go`
- Modify: `testdata/lighting_parity.golden` (re-recorded, deliberately)

- [ ] **Step 1: Replace the shortcuts**

In `ParticipantSight`, delete the two flag-shortcut blocks and the band switch,
and call the window instead. Keep the nil-observer check, the blinded check and
the nil-room guard exactly as they are, in that order. Keep the single hoisted
`configs.GetBalanceConfig()` call; do not reintroduce a second fetch.

```go
	balance := configs.GetBalanceConfig()
	return SightThroughWindow(
		room.LightLevel(),
		observer.NightVisionStrength(),
		observer.InfraReach(),
		int(balance.LightBlindBelow),
		int(balance.LightDimBelow),
	)
```

Replace the plan 1 note in the doc comment. It currently says the flag
shortcuts are deliberate leftovers that plan 2 replaces. They are now replaced,
so that paragraph is stale and must go. Write in its place what the window model
means for a reader: an ability moves the band rather than granting sight, and
the consequence is that night sight costs bright-light comfort.

- [ ] **Step 2: Run the messaging package**

Run: `go test ./internal/messaging/`

Expected: `ok`. ⚠️ **Existing tests in this package assert the OLD flag
behaviour.** `TestParticipantSight` and `TestOpticsTruthTable` almost certainly
have rows asserting that a nightvision holder sees fully in an unlit room. Those
rows are now wrong. **Update them to the new expectation and say in your report
exactly which rows you changed and why each one's new value is correct.** Do not
delete a row to make a suite pass.

- [ ] **Step 3: Run the parity golden and EXPECT it to fail**

Run: `go test . -run TestLightingParityAcrossEveryShippedRoom -v`

Expected: **FAIL.** This is the first time in the arc that a golden diff is
correct rather than a defect. Record the reported first-difference offset and
context.

- [ ] **Step 4: Prove the diff's SHAPE before re-recording**

🔴 **Do not re-record until this check passes.** The golden is the arc's only
behaviour-preservation instrument and a blanket re-record discards it.

Save the current golden, re-record, and diff the two. Then prove three things:

1. Every changed line is a `nightvision` line.
2. Every changed `nightvision` line belongs to a room whose other lines show
   `plain sight=none`, in other words a dark room.
3. The change is always `sight=full` to `sight=none`, with `clear` and `shapes`
   following.

Do this with a script or a filtered diff, not by eye over 6930 lines. Paste the
counts: how many lines changed, how many rooms, and the complete set of distinct
before-and-after pairs observed. If ANY `plain`, `blinded` or `infrared` line
moved, STOP and report; that means something other than the window changed.

Cross-check the room count against the arc's known figures: 140 rooms carry a
dark biome and 43 of those are permanently lit by static `lightmod: 2` mutators,
so roughly 97 rooms should be genuinely dark. The changed-room count should land
near that, and a wildly different number means the fixture or the model is wrong.

- [ ] **Step 5: Re-record, with the reasoning in the commit**

Run: `go test . -run TestLightingParityAcrossEveryShippedRoom -update-lighting-parity -v`

Then run it again without the flag and confirm PASS.

- [ ] **Step 6: Run everything**

Run: `go test ./...`
Expected: zero `FAIL`. Report the `ok` count. ⚠️ **`internal/combat` is the one
to watch**: `CanSeeSightImpairedOnly` feeds `DarknessCombatPenalty`, so a
nightvision holder in a dark room now takes the blind penalty where they
previously took none. If `darkness_penalty_verdict_test.go` fails, that is a
real behaviour change to reason about and report, not a fixture to silence.

- [ ] **Step 7: Commit**

```bash
git add internal/messaging/predicates.go internal/messaging/predicates_test.go testdata/lighting_parity.golden
git commit -F - <<'EOF'
feat(messaging): sight reads the window, and the flag shortcuts are gone

An ability no longer grants sight outright. It moves where the
observer's usable band sits on the scale, so night sight is bought with
bright-light comfort rather than being free.

THE PARITY GOLDEN MOVED, deliberately, for the first time in this arc.
One cell of shipped behaviour changes: a nightvision holder in an unlit
room goes from seeing fully to seeing nothing, because a shifted window
is still blind below its floor. Every other observer kind and every lit
room is byte identical, and that shape was proven by a filtered diff
before re-recording rather than assumed.

Infrared is unchanged at light 0 because its reach, not its shift, is
what reads an unlit cave.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 7: Author the strengths, and reword the draught

The mechanism exists but no content declares a number, so every holder is on
the default. This task makes the three shipped conditions say what they mean.

**Files:**
- Modify: `_datafiles/world/dogmud/conditions/29-night_vision.yaml`
- Modify: `_datafiles/world/dogmud/conditions/65-cats_eye_draught.yaml`
- Modify: `_datafiles/world/dogmud/conditions/85-infraredvision.yaml`

- [ ] **Step 1: Give each condition its numbers**

Add an `effects:` block to each, using the keys Task 2 added
(`nightvision_strength`, `infra_reach`). **Read an existing condition that
already declares `effects:` and copy its exact YAML shape** rather than guessing
the schema.

Values, and the reasoning to write into a comment in each file:

- **29 Night Vision**: `nightvision_strength: 18`. Granted by nothing today
  (fact 4), so this is the reference value an admin-granted flag reads. Above
  the default of 12 and below the cap of 24.
- **65 Cat's Eye Draught**: `nightvision_strength: 24`, the cap. It is a crafted,
  costed, temporary potion and should be the best night sight obtainable, which
  is the whole reason to pay 10 strength for it. **No `infra_reach`.**
- **85 InfraredVision**: `nightvision_strength: 12` and `infra_reach: 30`.
  Fact: the spec says the two are independent, and 30 makes shallow magical
  darkness readable while deep darkness is not.

- [ ] **Step 2: Reword the draught**

🔴 **This is the owner's ruling and it is player-visible.** Under the window
model the draught no longer makes an unlit cave transparent, so its text must
stop promising that. Rewrite `description`, `start_actee`, `end_actee`,
`start_observer` and `end_observer` so the potion reads as a **dim-light and
moonlight** tool, not a pitch-dark one.

Keep: `see-hidden`, `strength: -10`, 500 rounds, the cat imagery.
Change: anything claiming darkness becomes transparent or that it works in total
dark.

Apply `dogmud-player-copy`: 80-character wrap, no raw numbers, ESL-clear
phrasing, no em dashes or en dashes.

The item's own `description` in
`_datafiles/world/dogmud/items/consumables-30000/30047-cats_eye_draught.yaml`
also says "the darkness becomes transparent". **Fix that too.** A reworded
condition with an unchanged item description is exactly the sibling
inconsistency this project's rules forbid.

- [ ] **Step 3: Boot the server**

Run the detached-worktree boot check from `dogmud-shipping`. Condition YAML
panics at STARTUP on a schema error, and no unit test catches that.

Expected: `Server Ready`, zero panics, exit 124.

- [ ] **Step 4: Re-run the golden and the suite**

Run: `go test . -run TestLightingParityAcrossEveryShippedRoom -v`

⚠️ **This will move AGAIN**, because the golden's `nightvision` observer uses
condition 29, which now declares 18 instead of falling back to 12. Work out
whether 18 versus 12 changes any verdict at the three reachable light values
BEFORE you re-record. At light 0 both are blind, and at 60 and 70 both read
full, so **it should NOT move.** If it does move, something is wrong; stop and
report rather than re-recording.

Run: `go test ./...`
Expected: zero FAIL.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/conditions/29-night_vision.yaml _datafiles/world/dogmud/conditions/65-cats_eye_draught.yaml _datafiles/world/dogmud/conditions/85-infraredvision.yaml _datafiles/world/dogmud/items/consumables-30000/30047-cats_eye_draught.yaml
git commit -F - <<'EOF'
content(lighting): vision sources declare their strength, and the draught tells the truth

Three conditions stop relying on the bare-flag default and say how good
they actually are. The Cat's Eye Draught takes the cap, because it is
crafted, costed and temporary, and being the best night sight obtainable
is the reason to pay ten strength for it.

Its text is rewritten. Under the window model it no longer makes an
unlit cave transparent, and a potion whose description promises that
would be lying to the player. It is now plainly a dim-light and
moonlight tool. The item description said the same thing and is fixed
with it.

Seeing in TRUE dark becomes obtainable in plan 5, which ships a
nightvision spell, an infravision spell and an infravision potion.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 8: Give infravision a live consumer

Fact 4: nothing grants infraredvision, so Tasks 2, 3 and 5 built a mechanism
with no content exercising it. That is the unread-surface trap that had to be
stripped from M5 PR 3 before merge. The owner's ruling is to grant it to mobs.

**Files:**
- Modify: mob YAML for the chosen mobs
- Test: a guard that the grant is real

- [ ] **Step 1: Choose the mobs by reading, not by guessing**

Find mobs that live in dark-biome rooms. Start from the biomes that set
`darkarea: true` (cave, dungeon, swamp) and the zones whose rooms use them, then
look at which mobs those rooms spawn.

**Report the list you chose and why each one earns it.** A creature that hunts
in a lightless cave has an obvious reason to sense heat; a creature that wandered
in does not. Pick a small set, four to eight, not every mob in every cave.

⚠️ Check `tools/id_inventory.py` conventions and `dogmud-authoring-content`
before editing mob YAML. The filename must match the name field, and an
unquoted colon in a text field is a known panic.

- [ ] **Step 2: Grant the condition**

The mechanism already exists and is verified: **22 shipped mobs carry a
`conditionids:` key**, authored as an inline list. Two worked examples:

```
_datafiles/world/dogmud/mobs/ironwind_steppe/217-goblin_scout.yaml:7:  conditionids: [9]
_datafiles/world/dogmud/mobs/ironwind_steppe/225-pale_lurker.yaml:5:   conditionids: [9]
```

So the edit per mob is to add `85` to that list, creating the key if the mob has
none. Match the inline `[...]` style those files use.

⚠️ **Confirm condition 85 actually persists on a mob before trusting this.**
Fact: condition 85 was born expired until 2026-09-21 because it had no
`triggercount`, and its current comment explains that a pure flag condition never
ticks so any count at or above 1 persists for the holder's life. Verify that the
mob apply path honours that the same way the player path does, by booting and
checking a granted mob actually reports a nonzero reach, not by assuming.

- [ ] **Step 3: Write the guard**

Write a test asserting that at least one shipped mob resolves to a nonzero
`InfraReach()`. Its purpose is to catch a future edit that silently removes the
only consumer and returns the mechanism to being dead surface.

Name it so its intent is unmistakable, for example
`TestSomeShippedMobSensesHeat`. Prove it can fail by removing the grant from one
mob at a time until it reddens, then restore.

- [ ] **Step 4: Boot, then run everything**

Boot check per `dogmud-shipping`: mob YAML panics at startup.

Run: `go test ./...`
Expected: zero FAIL.

Run: `go test . -run TestLightingParityAcrossEveryShippedRoom -v`
Expected: PASS unchanged. The golden records PLAYER observer fixtures, not mobs,
so granting a mob a condition must not move it. If it does, something else
changed.

- [ ] **Step 5: Commit**

```bash
git add <the mob files you touched> <the guard test>
git commit -F - <<'EOF'
content(lighting): cave dwellers sense heat

Nothing in the game granted infraredvision, so the reach half of the
window model had no consumer at all. Shipping a mechanism nothing reads
is the trap that had to be stripped from M5 PR 3 before merge.

These creatures hunt where there is no light to use, so heat is how they
find anything. It also makes "mobs perceive darkness" real rather than
notional, which the behaviour work will want.

A guard asserts at least one shipped mob still resolves to a nonzero
reach, so a future edit cannot quietly return this to dead surface.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 9: Document, and the patch note this plan DOES owe

Plan 1 owed no patch note because nothing player-visible changed. **This plan
does.** A nightvision holder loses cave sight and the draught's text changes.

**Files:**
- Modify: `internal/messaging/context.md`
- Modify: `internal/characters/context.md`
- Modify: `internal/conditions/context.md`
- Modify: `internal/mutations/context.md`
- Modify: `docs/PATCH_NOTES.md`

- [ ] **Step 1: Update every context.md whose package changed**

Four packages changed their API or data model. Plan 1's Task 7 found that the
plan named only two context files while three packages had changed, so check all
four rather than trusting this list:

- `internal/messaging`: the window replaces the flag shortcuts. **Delete the
  plan 1 leftover paragraph**; it now describes code that is gone.
- `internal/characters`: `NightVisionStrength()` and `InfraReach()`.
- `internal/conditions`: the two new effect kinds and the MAX aggregation mode.
- `internal/mutations`: `FlagValue` and that flag effects now read `Value`.

**Every symbol you name must exist.** Extract the real surface first:

Run: `grep -nE '^(func|type|const|var) ' internal/messaging/window.go internal/characters/vision.go`

🔴 **This arc's reviews have caught SIX drifted or fabricated line citations.**
Open every file and line you cite and confirm it says what you claim.

- [ ] **Step 2: Run the audit**

Run: `python tools/context_md_audit.py internal/messaging`

Repeat per package. ⚠️ The tool takes ONE root path and walks it recursively; it
does not accept several package arguments. `internal/configs` has three
pre-existing phantoms unrelated to this work (`isEditAllowed`, `server_Config`,
`Get`); do not try to fix them here.

- [ ] **Step 3: Write the patch note**

Add a dated entry to `docs/PATCH_NOTES.md`. Apply `dogmud-player-copy`: player
framing, no raw numbers, no em dashes or en dashes, 80-character wrap.

It must tell a player two true things: night sight now trades away comfort in
bright light rather than being purely a gain, and the Cat's Eye Draught helps in
dim and moonlit places rather than in total darkness. **Do not mention scale
points, thresholds, strengths or config knobs.**

- [ ] **Step 4: Commit**

```bash
git add internal/messaging/context.md internal/characters/context.md internal/conditions/context.md internal/mutations/context.md docs/PATCH_NOTES.md
git commit -F - <<'EOF'
docs(lighting): context.md describes the window, and the patch note players need

Four packages changed shape and now say so. The messaging file loses its
plan 1 leftover paragraph, which described the flag shortcuts that this
plan deleted.

Unlike plan 1 this plan owes a patch note, because it changes what a
player experiences: night sight is now a trade rather than a pure gain,
and the Cat's Eye Draught is a dim-light tool.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Proof obligations

Before the PR opens:

- [ ] **`go test ./...` across every package, zero failures.** Not just the ones
      touched. A regression escaped a narrower run during M5.
- [ ] **`gofmt -l internal/ modules/`** clean, allowing for the documented
      Windows CRLF false positive (extract the committed blob with `git show`
      and check that, not the working copy).
- [ ] **`golangci-lint run --new-from-merge-base=master ./...`** reports 0
      issues. Run it locally; a red lint check on a large PR can be meaningless.
- [ ] **The parity golden moved EXACTLY ONCE**, in Task 6, and the shape of the
      diff was proven before re-recording: only `nightvision` lines, only in
      dark rooms, only `full` to `none`.
- [ ] **The window's own test covers every band edge and both clamps.**
- [ ] **Boot check** in an isolated detached worktree: `Server Ready`, zero
      panics, exit 124. Mandatory because this plan edits condition and mob
      YAML, which panics at startup rather than failing a test.
- [ ] **`internal/combat` reasoned about explicitly.** A nightvision holder in a
      dark room now takes `DarknessCombatPenalty` where they previously took
      none. Confirm that is intended and that
      `darkness_penalty_verdict_test.go` reflects it.
- [ ] **A patch note EXISTS**, unlike plan 1.
- [ ] **At least one shipped mob resolves to a nonzero `InfraReach()`**, guarded
      by a test.

## Risks

1. **The golden re-record is the one irreversible-feeling step.** Mitigated by
   proving the diff's shape first. If the shape check shows anything other than
   nightvision lines in dark rooms, the model is wrong and re-recording would
   hide it.
2. **`internal/characters` importing `internal/configs` may cycle.** Task 5
   checks. Plan 1 proved `internal/messaging` can import it, but this is a
   different package. If it cycles, the default-strength lookup must move to the
   caller and `bestVisionNumber` takes the default as an argument.
3. **The dim band is STILL unreachable** after this plan, because `LightLevel()`
   returns only 0, 60 and 70. The window's shapes tier is therefore exercised
   only by unit tests and by infrared reach until plan 3 lands. That is expected
   and is why plan 3 follows immediately.
4. **Mob permanent conditions may have no existing mechanism.** Task 8 reports
   rather than inventing one. If none exists, that is a finding for the owner,
   not a thing to improvise.
