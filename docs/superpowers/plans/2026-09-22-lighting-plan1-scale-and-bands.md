# Graded Lighting Plan 1: The Scale and the Bands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the three-value room visibility with a graded light scale and the normal observer's band thresholds, migrating every consumer, with no player-visible change whatsoever.

**Architecture:** A new `Room.LightLevel()` computes a graded value on the same inputs `GetVisibility()` used, mapped so every shipped room keeps its exact current classification. `internal/messaging`'s predicates derive the sight tiers from thresholds instead of a hardcoded `>= 1`. `GetVisibility()` is then deleted so the compiler enumerates its consumers. A golden recorded BEFORE any change, over every shipped room and every observer kind, is the proof that nothing moved.

**Tech Stack:** Go, `internal/rooms`, `internal/messaging`, `internal/configs`, repo-root golden test.

---

## Facts verified against source

Read from the tree on 2026-09-22 at master `389f4ad99`.

| # | Fact | Evidence |
|---|------|----------|
| 1 | `GetVisibility()` returns 0, 1 or 2: base 2, night minus 1, dark biome minus 2, lit biome plus 1, mutator `LightMod` summed, any light source plus 1, clamped | `internal/rooms/rooms.go:145-190` |
| 2 | It has 14 non-test consumers | grep, listed in Task 5 |
| 3 | Sight asks only `>= 1` | `internal/messaging/predicates.go:28`, `roomIsLit` |
| 4 | 🔴 **The 1-versus-2 distinction is NOT dead.** `look.go:258` gates looking THROUGH an exit on `visibility < 2` | `internal/usercommands/look.go:258-265` |
| 5 | That matches the documented mutator semantics: "1 = can see this room. 2 = can see this room and all exits" | `internal/mutators/mutators.go:70` |
| 6 | `ParticipantSight` short-circuits: blinded first, then lit room, then the NightVision flag, then the InfraredVision flag | `internal/messaging/predicates.go:51-68` |
| 7 | So today a NightVision holder sees FULLY in a pitch dark room | fact 6 |
| 8 | `litRoom{}` is a stand-in `RoomVisibility` hardcoded to 1, used by the AsLit send path | `internal/rooms/rooms.go:358-361` |
| 9 | Balance knobs live in one struct with yaml tags and a documented default in the trailing comment | `internal/configs/config.balance.go`, e.g. `:301` |
| 10 | Four biomes set `darkarea: true`: cave, dungeon, swamp, unused spiderweb | `_datafiles/world/dogmud/biomes/` |
| 11 | 1387 shipped rooms, 140 with a dark biome, 43 of those permanently lit by static `lightmod: 2` mutators | measured 2026-09-22 |

### 🔴 The correction this plan makes to the spec

The spec's fact 2 says only `look.go` reads the integer and implies the value
is otherwise binary. Reading it properly shows `look.go` uses the integer
**twice**, and the second use (`< 2`, fact 4) is a real second threshold: it
decides whether you can see down an exit into the next room.

**So this plan preserves TWO thresholds, not one.** A single lit-or-not
threshold would have silently broken exit-peering in every room that is lit
but not brightly lit.

---

## The mapping

Chosen so every shipped room keeps its exact current classification.

| Today | New light | Why |
|---|---|---|
| visibility 0 | **0** | dark; normal observers blind, exactly as now |
| visibility 1 | **60** | lit enough to see the room, not down an exit |
| visibility 2 | **70** | lit enough to see the room and its exits |

With thresholds `LightBlindBelow: 25`, `LightDimBelow: 50` and
`LightExitsAbove: 65`:

- 0 is below 25, so a normal observer is blind. Unchanged.
- 60 and 70 are both in the perfect band (50 to 75), so both give `SightFull`.
  Unchanged.
- 60 is below 65 so exits stay hidden; 70 is above 65 so exits are visible.
  Unchanged.
- **Nothing maps into the dim band (25 to 50)**, so the new `SightShapes`
  branch for normal observers is unreachable in this plan. That is deliberate:
  the band is defined here and becomes reachable in plan 3, once ambient light
  varies.

---

## File structure

| File | Responsibility |
|---|---|
| `internal/configs/config.balance.go` | the four threshold knobs |
| `internal/rooms/lighting.go` (new) | `LightLevel()` and the mapping, kept out of the already large `rooms.go` |
| `internal/rooms/rooms.go` | `GetVisibility()` deleted; `litRoom{}` updated |
| `internal/messaging/predicates.go` | bands derived from thresholds |
| `lighting_parity_golden_test.go` (new, repo root) | the proof that nothing moved |

---

## Task 1: Record the parity golden BEFORE anything changes

This is the proof artifact for the whole plan and it must be recorded against
the UNMODIFIED tree. Recorded after a change, it proves nothing.

**Files:**
- Create: `lighting_parity_golden_test.go` (repo root, `package main`)
- Create: `testdata/lighting_parity.golden`

- [ ] **Step 1: Write the golden test**

Create `lighting_parity_golden_test.go`:

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

var updateLightingParity = flag.Bool("update-lighting-parity", false,
	"re-record testdata/lighting_parity.golden")

// TestLightingParityAcrossEveryShippedRoom is the behaviour-preservation
// proof for graded lighting plan 1. It records, for every shipped room, what
// each kind of observer can see and whether exits are visible.
//
// It is recorded against the UNMODIFIED tree and must come back byte
// identical after the scale swap. A golden recorded after the change would
// prove nothing, which is why Task 1 comes first.
//
// The four observer kinds are the ones ParticipantSight actually branches on
// (predicates.go:51-68): an ordinary character, one carrying NightVision, one
// carrying InfraredVision, and one that is Blinded.
func TestLightingParityAcrossEveryShippedRoom(t *testing.T) {
	configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles":  "_datafiles/world/dogmud",
		"Network.LogoutRounds": 3,
	})
	rooms.LoadBiomeDataFiles()
	conditions.LoadDataFiles()
	rooms.LoadDataFiles()

	ids := rooms.GetAllRoomIds()
	if len(ids) < 1000 {
		t.Fatalf("loaded only %d rooms: the walk is not seeing the world, so a green run proves nothing", len(ids))
	}
	sort.Ints(ids)

	observers := []struct {
		name        string
		conditionId int
	}{
		{"plain", 0},
		{"nightvision", 29},
		{"infrared", 85},
		{"blinded", 3},
	}

	var b strings.Builder
	for _, id := range ids {
		room := rooms.LoadRoom(id)
		if room == nil {
			continue
		}
		fmt.Fprintf(&b, "room %d biome=%s\n", id, room.Biome)
		for _, o := range observers {
			ch := characters.New()
			ch.Name = o.name
			if o.conditionId > 0 {
				if err := ch.AddCondition(o.conditionId, true); err != nil {
					t.Fatalf("AddCondition(%d): %v", o.conditionId, err)
				}
			}
			fmt.Fprintf(&b, "  %-12s sight=%v clear=%v shapes=%v\n",
				o.name,
				messaging.ParticipantSight(ch, room),
				messaging.CanSeeClearly(ch, room),
				messaging.CanSeeShapes(ch, room))
		}
	}

	goldenPath := filepath.Join("testdata", "lighting_parity.golden")
	if *updateLightingParity {
		if err := os.WriteFile(goldenPath, []byte(b.String()), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("re-recorded %s (%d bytes, %d rooms)", goldenPath, b.Len(), len(ids))
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v (record it with -update-lighting-parity)", err)
	}
	if b.String() != string(want) {
		t.Errorf("lighting parity changed. This plan is behaviour preserving, so a diff here is a DEFECT, not something to re-record.\nIf a change is genuinely intended, re-record with:\n  go test . -run TestLightingParityAcrossEveryShippedRoom -update-lighting-parity")
	}
}
```

⚠️ Verify `rooms.GetAllRoomIds` and `rooms.LoadDataFiles` exist with those
exact names before relying on them. If the room-id enumerator is named
differently, adapt and say so. Check with:

Run: `grep -nE '^func (GetAllRoomIds|LoadDataFiles)' internal/rooms/*.go`

🔴 **VERIFY THE BLINDED OBSERVER IS ACTUALLY BLIND, before trusting the
golden.** `ParticipantSight` tests `observer.Perception != nil &&
observer.Perception.State() == perception.Blinded`, which is the perception
machine, **not** a condition flag. Adding condition 3 may or may not drive it.
If it does not, that observer records as fully sighted and the golden's blind
coverage is fake while looking complete.

Prove it before recording:

```
go test ./internal/messaging/ -run TestParticipantSight -v
```

then check directly that a character built the way the golden builds its
`blinded` observer returns `SightNone` in a LIT room. If condition 3 does not
produce that, find what does (look at how `perception.Blinded` is actually
set) and build the fixture that way instead. **Report which mechanism you
used.** A null probe that cannot fail is the failure mode this whole task
exists to avoid.

- [ ] **Step 2: Record it**

Run: `go test . -run TestLightingParityAcrossEveryShippedRoom -update-lighting-parity -v`

Expected: PASS, with a log line naming the byte count and a room count at or
above 1387.

- [ ] **Step 3: READ the golden and sanity-check it**

Run: `head -20 testdata/lighting_parity.golden`

Expected: rooms with their biome, and four observer lines each. **Confirm by
eye that a `biome=cave` room shows `plain` blind and `infrared` seeing
shapes.** If every room looks identical regardless of biome, the walk is not
computing real lighting and everything downstream is worthless.

Run: `grep -c "biome=cave" testdata/lighting_parity.golden`

Expected: a count in the low hundreds, consistent with fact 11's 140 dark
biome rooms.

- [ ] **Step 4: Run it as a check**

Run: `go test . -run TestLightingParityAcrossEveryShippedRoom -v`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add lighting_parity_golden_test.go testdata/lighting_parity.golden
git commit -F - <<'EOF'
test(lighting): record the parity golden before the scale changes

Graded lighting plan 1 replaces the room visibility model outright while
promising no player-visible change. This is the artifact that proves it.

For every shipped room it records what each of the four observer kinds
ParticipantSight branches on can see: an ordinary character, one with
NightVision, one with InfraredVision, and a blinded one.

Recorded against the unmodified tree deliberately. A golden recorded
after the change would only prove the change is self-consistent, which
is not the question being asked.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 2: The threshold knobs

**Files:**
- Modify: `internal/configs/config.balance.go`

- [ ] **Step 1: Add the knobs**

Add to the balance config struct, following the file's existing style of a
yaml tag and a trailing comment naming the default:

```go
	// Graded room lighting thresholds, on the -100 to 100 light scale
	// introduced by the graded lighting arc. They define the normal
	// observer's bands: blind below LightBlindBelow, shapes from there to
	// LightDimBelow, and perfect above it. The dazzle threshold above the
	// perfect band arrives with the plan that gives dazzle an effect.
	//
	// LightExitsAbove is separate and older in spirit: it is the threshold
	// for seeing THROUGH an exit into the next room, which the previous
	// model expressed as visibility 2 rather than 1.
	LightBlindBelow ConfigInt `yaml:"LightBlindBelow"` // Below this a normal observer is blind (default 25)
	LightDimBelow   ConfigInt `yaml:"LightDimBelow"`   // Below this a normal observer reads shapes only (default 50)
	LightExitsAbove ConfigInt `yaml:"LightExitsAbove"` // At or above this, exits into adjacent rooms are visible (default 65)
```

⚠️ **Three knobs, not four. There is deliberately no `LightDazzleAbove`
here.** The dazzle band has no mechanical effect until the penalty exists,
which is plan 2, so adding its threshold now would ship a config knob that
nothing reads. This project has been bitten by exactly that: M5 PR 3 shipped a
type with two helper methods no production code called and had to strip them
before merge. Add the knob in the plan that uses it.

- [ ] **Step 2: Set the defaults**

Find how this file's other `ConfigInt` knobs get their defaults, and follow
exactly the same mechanism:

Run: `grep -n "DarknessCombatPenalty" internal/configs/*.go`

Set `LightBlindBelow: 25`, `LightDimBelow: 50` and `LightExitsAbove: 65` by
that mechanism.

- [ ] **Step 3: Build and test**

Run: `go build ./... && go test ./internal/configs/`

Expected: no build output, then `ok`.

- [ ] **Step 4: Commit**

```bash
git add internal/configs/config.balance.go
git commit -F - <<'EOF'
feat(config): the graded lighting thresholds

Four knobs defining the normal observer's bands on the new light scale,
plus the separate threshold for seeing through an exit.

That last one is not new behaviour. The previous model expressed it as
the difference between visibility 1 and 2, which look.go:258 reads to
decide whether you can see down a corridor. Carrying it as its own knob
keeps that distinction explicit rather than implied by an integer.

Balance numbers belong in config rather than Go, per the project rule.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 3: LightLevel, alongside the old model

`GetVisibility` stays for now so the tree keeps building. Task 5 deletes it.

**Files:**
- Create: `internal/rooms/lighting.go`
- Test: `internal/rooms/lighting_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/rooms/lighting_test.go`:

```go
package rooms

import "testing"

// TestLightLevelMapsTheOldModel pins the mapping that makes graded lighting
// plan 1 behaviour preserving. Every shipped room must land on one of three
// values, and each must sit on the correct side of every threshold.
//
// 0  dark, a normal observer is blind
// 60 lit enough to see the room, NOT down an exit
// 70 lit enough to see the room and its exits
func TestLightLevelMapsTheOldModel(t *testing.T) {
	tests := []struct {
		name string
		room *Room
		want int
	}{
		{"default biome, day", &Room{RoomId: 1}, 70},
		{"cave", &Room{RoomId: 2, Biome: "cave"}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.room.LightLevel(); got != tc.want {
				t.Errorf("LightLevel() = %d, want %d", got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/rooms/ -run TestLightLevelMapsTheOldModel -v`

Expected: build failure, `room.LightLevel undefined`.

- [ ] **Step 3: Implement it**

Create `internal/rooms/lighting.go`:

```go
package rooms

// Light levels on the graded scale. Plan 1 maps the previous three-value
// visibility model onto exactly these three points, chosen so that every
// shipped room keeps its current classification against every threshold.
//
// Later plans in this arc make the scale continuous. Nothing outside this
// file should assume a room's light is one of these three values.
const (
	// LightDark is a room with no light at all. A normal observer is blind.
	LightDark = 0
	// LightRoomOnly is lit enough to see the room but not down an exit.
	// It is what the old model called visibility 1.
	LightRoomOnly = 60
	// LightFull is lit enough to see the room and its exits. It is what the
	// old model called visibility 2.
	LightFull = 70
)

// LightLevel reports the room's light on the graded scale.
//
// Plan 1 deliberately computes this from the same inputs the previous
// GetVisibility used, then maps the result onto the three constants above.
// The point of this plan is the SCALE and its consumers, not new lighting
// behaviour, so a diff in what any room reports here is a defect.
func (r *Room) LightLevel() int {
	switch r.legacyVisibility() {
	case 0:
		return LightDark
	case 1:
		return LightRoomOnly
	default:
		return LightFull
	}
}
```

Then move the current body of `GetVisibility` into a new unexported
`legacyVisibility()` method in the same new file, verbatim, and have
`GetVisibility` in `rooms.go` call it:

```go
func (r *Room) GetVisibility() int { return r.legacyVisibility() }
```

⚠️ **Move the body verbatim.** Do not tidy it while moving it. A behaviour
change smuggled in during a move is exactly what the Task 1 golden exists to
catch, and you want it to catch nothing.

- [ ] **Step 4: Run the test**

Run: `go test ./internal/rooms/ -run TestLightLevelMapsTheOldModel -v`

Expected: PASS for both cases.

- [ ] **Step 5: Run the package and the parity golden**

Run: `go test ./internal/rooms/ && go test . -run TestLightingParityAcrossEveryShippedRoom`

Expected: `ok` for both. Nothing consumes `LightLevel` yet, so parity must be
untouched.

- [ ] **Step 6: Commit**

```bash
git add internal/rooms/lighting.go internal/rooms/lighting_test.go internal/rooms/rooms.go
git commit -F - <<'EOF'
feat(rooms): LightLevel on the graded scale, mapped from the old model

The first half of the swap. LightLevel computes from exactly the inputs
GetVisibility used and maps the result onto three points on the graded
scale, chosen so every shipped room keeps its classification against
every threshold, including the seeing-through-an-exit one.

GetVisibility's body moved verbatim into legacyVisibility and is now
called through, so the two cannot drift while both exist. Nothing
consumes LightLevel yet.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 4: The bands, in the predicates

**Files:**
- Modify: `internal/messaging/predicates.go`
- Test: `internal/messaging/predicates_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/messaging/predicates_test.go`:

```go
// TestParticipantSightReadsTheBands pins the normal observer's band model.
// The dim band is unreachable from shipped content in plan 1 (nothing maps
// between 25 and 50), so this test drives the thresholds directly through a
// stub room. It is what makes the band real before plan 3 starts producing
// dim rooms.
func TestParticipantSightReadsTheBands(t *testing.T) {
	tests := []struct {
		name  string
		light int
		want  SightDecision
	}{
		{"pitch dark", 0, SightNone},
		{"just below blind threshold", 24, SightNone},
		{"dim, bottom", 25, SightShapes},
		{"dim, top", 49, SightShapes},
		{"perfect, bottom", 50, SightFull},
		{"perfect, top", 75, SightFull},
		{"dazzled, still sees", 90, SightFull},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ch := characters.New()
			got := ParticipantSight(ch, fixedLightRoom(tc.light))
			if got != tc.want {
				t.Errorf("ParticipantSight at light %d = %v, want %v", tc.light, got, tc.want)
			}
		})
	}
}
```

You will need a `fixedLightRoom` stub implementing the room interface the
predicates take. Read `RoomVisibility` in `predicates.go` first and write the
stub to match whatever that interface requires after Task 5 renames its
method. If Task 5 has not run yet, write the stub against the current
interface and update it in Task 5.

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/messaging/ -run TestParticipantSightReadsTheBands -v`

Expected: FAIL. At light 25 and 49 it reports `SightFull` where the test wants
`SightShapes`, because no band logic exists yet.

- [ ] **Step 3: Implement the bands**

Replace `roomIsLit` with band helpers reading the config knobs, and rewrite
`ParticipantSight`'s middle section. **Keep the blinded check first and keep
the NightVision and InfraredVision flag shortcuts exactly as they are.**

```go
// ParticipantSight is THE optics primitive. [keep the existing doc comment,
// then add:]
//
// PLAN 1 NOTE. The NightVision and InfraredVision branches below are the
// pre-graded-lighting flag shortcuts, kept deliberately. Plan 2 of the graded
// lighting arc replaces them with the window model, where an ability shifts
// where the observer's usable band sits rather than granting sight outright.
// They are left alone here because the window model is a real behaviour
// change for those holders (today a NightVision holder sees fully in a pitch
// dark room, and under the window model they are blind below 1), and changing
// the scale and their behaviour in one plan would make this plan's
// behaviour-preservation guarantee impossible to assert.
func ParticipantSight(observer *characters.Character, room RoomVisibility) SightDecision {
	if observer == nil {
		return SightFull
	}
	if observer.Perception != nil && observer.Perception.State() == perception.Blinded {
		return SightNone
	}
	if room == nil {
		return SightFull
	}
	switch light := room.LightLevel(); {
	case light >= int(configs.GetBalanceConfig().LightDimBelow):
		return SightFull
	case light >= int(configs.GetBalanceConfig().LightBlindBelow):
		return SightShapes
	}
	if observer.HasFlagFromAnySource(conditions.NightVision) {
		return SightFull
	}
	if observer.HasFlagFromAnySource(conditions.InfraredVision) {
		return SightShapes
	}
	return SightNone
}
```

⚠️ Check whether `internal/messaging` already imports `internal/configs`. If
it does not, confirm that adding it creates no import cycle before proceeding,
and report the result either way.

- [ ] **Step 4: Run the test, the package, and the parity golden**

Run: `go test ./internal/messaging/ -run TestParticipantSightReadsTheBands -v`
Expected: PASS for all seven cases.

Run: `go test ./internal/messaging/`
Expected: `ok`.

Run: `go test . -run TestLightingParityAcrossEveryShippedRoom`
Expected: PASS. **This is the important one.** Nothing maps into the dim band,
so no shipped room may change.

- [ ] **Step 5: Commit**

```bash
git add internal/messaging/predicates.go internal/messaging/predicates_test.go
git commit -F - <<'EOF'
feat(messaging): sight tiers derive from light bands, not a lit flag

ParticipantSight stops asking whether the room is lit and starts asking
where its light sits against the normal observer's thresholds. The dim
band, which returns shapes, is unreachable from shipped content in this
plan because nothing maps between the blind and dim thresholds. It is
defined now so plan 3 has somewhere to put a dim room.

The NightVision and InfraredVision flag shortcuts are deliberately
untouched. Plan 2 replaces them with the window model, which is a real
behaviour change for those holders, and doing both at once would make
this plan's parity guarantee unassertable.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 5: Delete GetVisibility and migrate every consumer

Changing the name is what makes the compiler enumerate the consumers. This
project's refactoring idiom, and it found an unlisted call site in messaging
M5 PR 3.

**Files:**
- Modify: `internal/rooms/rooms.go` (delete `GetVisibility`, update `litRoom`)
- Modify: the 13 consumer sites below

- [ ] **Step 1: Delete it and let the compiler speak**

Delete `func (r *Room) GetVisibility() int` from `rooms.go`. Rename the
`RoomVisibility` interface method in `internal/messaging/predicates.go:14`
from `GetVisibility() int` to `LightLevel() int`, and update `litRoom` in
`rooms.go:358-361`:

```go
// litRoom is a messaging.RoomVisibility that is always lit. Any light source
// lifts a room to at least LightRoomOnly, which is all sight needs.
type litRoom struct{}

func (litRoom) LightLevel() int { return LightRoomOnly }
```

Run: `go build ./... 2>&1 | head -30`

Record the full list of errors. **That list is the enumeration** and every
entry must be handled below.

- [ ] **Step 2: Migrate the twelve binary consumers**

Each of these asks "is this room lit". Replace `room.GetVisibility() >= 1`
with `room.LightLevel() >= int(configs.GetBalanceConfig().LightBlindBelow)`,
and `< 1` with the negation. Add the `configs` import where needed.

```
internal/actions/skill_helpers.go:54
internal/hooks/Death_MobBroadcast.go:55
internal/hooks/Death_MobLoot.go:124
internal/hooks/MobIdle_HandleIdleMobs.go:115
internal/hooks/NewRound_DoCombat_helpers.go:422
internal/hooks/NewRound_UserRoundTick.go:211
internal/mobcommands/darkness.go:31
internal/usercommands/get.go:75
internal/usercommands/loot.go:24
```

⚠️ Line numbers drift as you edit. Work from the compiler's list, not this
one, and confirm each site's current text before changing it.

- [ ] **Step 3: Migrate look.go, which uses BOTH thresholds**

`internal/usercommands/look.go` reads the value twice. Replace:

```go
	visibility := room.GetVisibility()

	if visibility < 1 {
```

with:

```go
	light := room.LightLevel()
	balance := configs.GetBalanceConfig()

	if light < int(balance.LightBlindBelow) {
```

and replace the second use at `:258`:

```go
		if visibility < 2 {
```

with:

```go
		// Seeing THROUGH an exit needs more light than seeing the room you
		// are standing in. The old model expressed this as visibility 2
		// rather than 1; LightExitsAbove carries it explicitly.
		if light < int(balance.LightExitsAbove) {
```

🔴 **This is the site the spec got wrong.** Collapsing both uses onto one
threshold would silently let players see down corridors from rooms that are
merely lit, or block them from rooms that are brightly lit, depending which
threshold you picked. The parity golden does not cover exit-peering, so this
one is on you to get right by reading.

- [ ] **Step 4: Build and run everything**

Run: `go build ./...`
Expected: no output.

Run: `go test ./...`
Expected: zero `FAIL` lines. Report the `ok` count.

Run: `go test . -run TestLightingParityAcrossEveryShippedRoom -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -u
git commit -F - <<'EOF'
refactor(rooms): GetVisibility is deleted, consumers read LightLevel

Renaming rather than shimming, so the compiler enumerates every
consumer. That idiom found an unlisted call site in messaging M5 PR 3
and is cheap insurance here.

Twelve sites only ever asked whether the room was lit and now compare
against the blind threshold. look.go is the exception: it uses the value
twice, and its second use gates seeing THROUGH an exit, which the old
model expressed as the difference between visibility 1 and 2. That is
now LightExitsAbove, named rather than implied.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

⚠️ `git add -u` stages every tracked modification. Run `git status` first and
confirm the list is exactly the files this task touched. The project rule
against `git add -A` exists because of unrelated files being swept in; `-u`
has the same hazard on a dirty tree.

---

## Task 6: Prove the golden can fail

A parity golden that has never been seen red is not a proof.

**Files:** none committed; this task is a sabotage and a revert.

- [ ] **Step 1: Sabotage the mapping**

In `internal/rooms/lighting.go`, temporarily change `LightRoomOnly` from 60 to
20, which drops every visibility-1 room below the blind threshold.

- [ ] **Step 2: Run the golden and confirm it fails loudly**

Run: `go test . -run TestLightingParityAcrossEveryShippedRoom -v`

Expected: FAIL, reporting that lighting parity changed. Record the output.

- [ ] **Step 3: Revert and confirm green**

Restore 60. Run: `git diff internal/rooms/lighting.go`
Expected: empty.

Run: `go test . -run TestLightingParityAcrossEveryShippedRoom -v`
Expected: PASS.

- [ ] **Step 4: Sabotage the exit threshold instead**

This one the golden does NOT catch, and proving that is the point.
Temporarily set `LightExitsAbove` to 50 in the config default, so every lit
room would allow exit-peering.

Run: `go test . -run TestLightingParityAcrossEveryShippedRoom -v`
Expected: **PASS**, because the golden records sight tiers and not
exit-peering.

Revert it. Then record in the commit message that exit-peering is verified by
reading `look.go` and by the boot check, not by this golden. **A guard whose
blind spots are unknown is worse than one whose blind spots are written
down.**

- [ ] **Step 5: Commit the finding**

```bash
git commit --allow-empty -F - <<'EOF'
test(lighting): record what the parity golden does and does not catch

Sabotage results for the plan 1 parity golden.

Dropping the visibility-1 mapping below the blind threshold reddens it
immediately, so it genuinely covers the sight tiers for every shipped
room and every observer kind.

It does NOT catch a wrong exit threshold: setting LightExitsAbove to 50
leaves it green, because the golden records sight tiers and exit-peering
is not one. That site is verified by reading look.go and by the boot
check instead. Writing the blind spot down is the point; a guard whose
gaps are unknown is worse than one whose gaps are documented.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 7: Document the package

**Files:**
- Modify: `internal/rooms/context.md`
- Modify: `internal/messaging/context.md`

- [ ] **Step 1: Update both context files**

`internal/rooms/context.md` must describe `LightLevel`, the three mapping
constants, and that `GetVisibility` is gone. `internal/messaging/context.md`
must describe the band thresholds and that the NightVision and InfraredVision
branches are plan 1 leftovers scheduled for replacement in plan 2.

**Every symbol named must exist.** Extract the real surface first:

Run: `grep -nE '^(func|type|const|var) ' internal/rooms/lighting.go`

- [ ] **Step 2: Run the audit**

Run: `python tools/context_md_audit.py internal/rooms internal/messaging`

Expected: zero phantom symbols.

- [ ] **Step 3: No patch note**

This plan is behaviour preserving, so `docs/PATCH_NOTES.md` gets **no entry**.
Confirm you have not added one. The arc's first player-visible change is plan
3.

- [ ] **Step 4: Commit**

```bash
git add internal/rooms/context.md internal/messaging/context.md
git commit -F - <<'EOF'
docs(lighting): context.md describes the graded scale

Both packages now describe LightLevel, the mapping constants and the
band thresholds. The messaging file also records that its NightVision
and InfraredVision branches are plan 1 leftovers, so the next reader
does not mistake them for the intended end state.

No patch note: this plan changes nothing a player can observe.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Proof obligations

Before the PR opens:

- [ ] **`go test ./...` across all 121 packages, zero failures.** Not just the
      packages touched. A regression escaped a narrower run during M5.
- [ ] **`gofmt -l internal/ modules/` empty.**
- [ ] **The parity golden is byte identical to its Task 1 recording**, and has
      been seen red once (Task 6).
- [ ] **Boot check** in an isolated detached worktree: `Server Ready`, zero
      real panics, exit 124.
- [ ] **Exit-peering verified by hand**, because the golden cannot see it. In
      the booted server, stand in a lit room and look toward an exit, then do
      the same from a dark room, and confirm both behave as they do on master.
- [ ] **No entry in `docs/PATCH_NOTES.md`.**

## Risks

1. **The parity golden is the whole proof, and it has one known blind spot**
   (exit-peering, Task 6). That gap is covered by reading and by the manual
   boot check, and is written into the commit message rather than left to be
   rediscovered.
2. **`internal/messaging` may not import `internal/configs` today.** Task 4
   checks. If that import would create a cycle, the thresholds must be passed
   in rather than read, which changes the shape of `ParticipantSight`'s
   signature and every caller. Stop and report if so rather than working
   around it.
3. **Nothing exercises the dim band in this plan.** That is deliberate and
   stated, but it means the band's correctness rests on Task 4's unit test
   alone until plan 3 lands.
