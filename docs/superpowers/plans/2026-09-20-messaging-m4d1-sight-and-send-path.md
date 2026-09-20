# Messaging M4d PR 1: One Sight Verdict, Four Audiences, One Path

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The sight verdict becomes the primitive that the three boolean
predicates are defined over, `Trio` gains its fourth audience, and every
narration path that can deliver through `SendTrio` does.

**Architecture:** `messaging.ParticipantSight` already computes the one verdict,
but it is defined ON TOP of `CanSeeSightImpairedOnly`, so a boolean is the
primitive and the richer verdict is derived from it. This plan inverts that: the
verdict computes optics directly, and the three booleans become one-line
policies that compose attention (sleep) over it. Combat stops borrowing a
messaging predicate for its damage-scoring rule and carries the verdict itself,
which is what lets PR 2 give seeing shapes its own reduced penalty. `Trio` gains
`RemoteObserver`, seating ranged combat's defender-room audience. The four
remaining non-`SendTrio` paths move onto it, each one first proven to change no
text.

**Player-visible change: NONE.** Every text-moving decision belongs to PR 2.
Any path in Task 6 that cannot be proven text-identical is DEFERRED to PR 2
rather than shipped here.

**Spec:** `docs/superpowers/specs/2026-09-20-messaging-m4d-send-path-design.md`

**Tech Stack:** Go, `internal/messaging` (predicates, Trio), `internal/combat`,
`internal/rooms`, `internal/usercommands`, `internal/hooks`, golden files under
`internal/narration/testdata/stores`, AST guards at the repo root.

---

## Facts verified against source (master `523ea3cc4`, 2026-09-20)

| Fact | Value | Source |
|---|---|---|
| Verdict producer | `ParticipantSight(observer, room) SightDecision` EXISTS | `internal/messaging/predicates.go:139-150` |
| Its dependency | built on `CanSeeSightImpairedOnly`, the inversion this plan fixes | `predicates.go:140` |
| Room wrapper | `(*Room).ParticipantSight(userId)`, returns `SightFull` for an unknown user | `internal/rooms/rooms.go:321-326` |
| `CanSeeClearly` | blind -> false; **asleep -> false**; lit -> true; else NightVision | `predicates.go:24-46` |
| `CanSeeSightImpairedOnly` | blind -> false; lit -> true; else NightVision. **No sleep test, no infrared** | `predicates.go:72-84` |
| `CanSeeShapes` | `CanSeeClearly` OR (not blind, **not asleep**, InfraredVision) | `predicates.go:92-110` |
| Predicate call sites | `CanSeeClearly` 12, `CanSeeSightImpairedOnly` 15, `CanSeeShapes` 4 (non-test) | grep |
| Combat's borrow | `combatContext.sourceCanSee` / `targetCanSee` from `CanSeeSightImpairedOnly` | `internal/combat/combat.go:55,56,106,107,150,151,199,200` |
| What it drives | `Balance.DarknessCombatPenalty` on attack and defence scores, nothing else | `internal/combat/combat_helpers.go:557,749` |
| The seam comment | `predicates.go` itself says M2/M4 should collapse these, "with combat naming the specific disadvantage it means rather than borrowing a sight predicate" | `predicates.go:96-101` |
| `Trio` | `struct{ Actor, Actee, Observer Line }` | `internal/messaging/trio.go:40` |
| Trio literal guard | `TestEveryTrioLiteralNamesAllThreeRoles`, roles list at the top of the function | `messaging_surface_guard_test.go:1635-1637` |
| `SendTrio` | 151 call sites, 32 files, non-test | grep |
| Ranged defender-room line | built as `toDefenderRoomMsg` from `roles.ActeeObserver`, sent via `SendToTargetRoom` | `internal/combat/combat.go:316,341`; `combat_helpers.go:1722` |
| Crafting delivery | `user.SendText(messaging.CategorySystem, ...)`, self-only | `internal/usercommands/craft.go:73,79,85,89` |
| position_control delivery | `u.SendText` plus `r.SendTextVisual` | `internal/hooks/Position_Messaging.go:233,249,305` |
| Raw-message guard | `TestNoRawEventsMessageOutsidePipeline` | `raw_events_message_guard_test.go:27` |
| 🪤 `SightDecision` ordering | **best to worst**: `SightFull = iota` (0), `SightShapes` (1), `SightNone` (2). Ordered comparisons read BACKWARDS; use equality | `internal/messaging/pipeline.go:34-38` |

---

## File Structure

**Created**
- `internal/messaging/optics_pin_test.go`: the truth table that pins today's answers.
- `internal/messaging/sleep_policy_test.go`: the sleeping-observer guard.
- `internal/combat/darkness_penalty_verdict_test.go`: pins that the verdict drives the penalty exactly as the old boolean did, infrared still penalised.

**Modified**
- `internal/messaging/predicates.go`: dependency inverted; three one-line policies.
- `internal/messaging/trio.go`: `RemoteObserver` seat.
- `messaging_surface_guard_test.go`: guard moves to four roles.
- `internal/combat/combat.go`, `internal/combat/combat_helpers.go`: the context carries `SightDecision`; the ranged seat.
- Whichever of `internal/usercommands/craft.go`, the quest sender, the caster-only spell effect path and `internal/hooks/Position_Messaging.go` pass Task 6's text-identity check.
- `internal/messaging/context.md`, `internal/combat/context.md`.

---

## Task 1: Pin today's optics before touching them

**Why first:** this task's output is the net for every later task. It must pin
the two things the spec's first draft got wrong from reading names instead of
bodies: that `CanSeeSightImpairedOnly` ignores infrared, and that two of the
three predicates test sleep while the third deliberately does not.

**Files:**
- Create: `internal/messaging/optics_pin_test.go`

- [ ] **Step 1: Write the truth table**

```go
package messaging

import "testing"

// opticsCase is one observer state crossed with one room state.
type opticsCase struct {
	name                 string
	blind, asleep        bool
	nightVision          bool
	infraredVision       bool
	lit                  bool
	wantClearly          bool
	wantImpairedOnly     bool
	wantShapes           bool
}

// TestOpticsTruthTable pins what the three sight predicates answer TODAY, for
// every combination that distinguishes them. It is the net for M4d's
// dependency inversion: the predicates are about to be redefined over
// ParticipantSight, and byte-identical goldens cannot prove a boolean's
// semantics.
//
// Two rows here are the ones worth reading slowly, because an earlier draft of
// the M4d spec got both wrong by reasoning from the function NAMES:
//
//   - infrared in the dark: CanSeeSightImpairedOnly is FALSE. It consults
//     NightVision only. Widening it to "shapes or better" would silently
//     remove DarknessCombatPenalty from every infrared character.
//   - asleep in a lit room: CanSeeSightImpairedOnly is TRUE. It has no
//     attention test on purpose, so combat does not double a sleeper's
//     disadvantage (they are already auto-crit).
func TestOpticsTruthTable(t *testing.T) {
	cases := []opticsCase{
		{name: "lit, ordinary", lit: true,
			wantClearly: true, wantImpairedOnly: true, wantShapes: true},
		{name: "dark, no vision",
			wantClearly: false, wantImpairedOnly: false, wantShapes: false},
		{name: "dark, nightvision", nightVision: true,
			wantClearly: true, wantImpairedOnly: true, wantShapes: true},
		{name: "dark, infrared only", infraredVision: true,
			wantClearly: false, wantImpairedOnly: false, wantShapes: true},
		{name: "blind in a lit room", blind: true, lit: true,
			wantClearly: false, wantImpairedOnly: false, wantShapes: false},
		{name: "blind with infrared", blind: true, infraredVision: true,
			wantClearly: false, wantImpairedOnly: false, wantShapes: false},
		{name: "asleep in a lit room", asleep: true, lit: true,
			wantClearly: false, wantImpairedOnly: true, wantShapes: false},
		{name: "asleep with infrared in the dark", asleep: true, infraredVision: true,
			wantClearly: false, wantImpairedOnly: false, wantShapes: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obs := newOpticsObserver(t, tc)
			room := newOpticsRoom(t, tc.lit)

			if got := CanSeeClearly(obs, room); got != tc.wantClearly {
				t.Errorf("CanSeeClearly = %v, want %v", got, tc.wantClearly)
			}
			if got := CanSeeSightImpairedOnly(obs, room); got != tc.wantImpairedOnly {
				t.Errorf("CanSeeSightImpairedOnly = %v, want %v", got, tc.wantImpairedOnly)
			}
			if got := CanSeeShapes(obs, room); got != tc.wantShapes {
				t.Errorf("CanSeeShapes = %v, want %v", got, tc.wantShapes)
			}
		})
	}
}
```

⚠️ `newOpticsObserver` and `newOpticsRoom` are helpers you must write against
the REAL types. `CanSeeClearly` takes `*characters.Character` and a
`RoomVisibility`. Read `internal/messaging/predicates.go` for the interface and
find an existing test in this package that builds a character with a condition
flag and a room with a visibility; copy that construction rather than inventing
one. `internal/rooms/participant_sight_test.go` builds both for the sibling
function and is the closest model. If blindness is set through
`observer.Perception` and a state machine, set it the way that test does.

- [ ] **Step 2: Run it and confirm every row passes**

Run: `go test ./internal/messaging/ -run TestOpticsTruthTable -v`

Expected: PASS, eight subtests. **If any row fails, STOP.** A failing row means
this plan's reading of the predicate bodies is wrong, and the whole design rests
on that reading. Report which row and what it actually returned; do not adjust
the expectation to make it green.

- [ ] **Step 3: Prove the table can fail**

Temporarily change `CanSeeSightImpairedOnly` to also accept
`conditions.InfraredVision`. Run the same command.

Expected: **FAIL** on `dark, infrared only`, naming
`CanSeeSightImpairedOnly = true, want false`. That is exactly the balance change
the spec warns about, and this row is the thing standing between it and
production. Revert, rerun, expect PASS, and confirm
`git diff internal/messaging/predicates.go` prints nothing.

- [ ] **Step 4: Commit**

```bash
git add internal/messaging/optics_pin_test.go
git commit -m "test(messaging): pin what the three sight predicates answer today

The net for M4d's dependency inversion. Two rows exist because an earlier draft
of the spec reasoned from the function names: infrared does not satisfy
CanSeeSightImpairedOnly, and a sleeper in a lit room does satisfy it.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: The sleeping-observer guard

**Why separate from Task 1:** Task 1 pins the predicates as functions. This
pins the CONSEQUENCE that no golden covers, because no golden drives a sleeping
observer: a sleeper must not read narration.

**Files:**
- Create: `internal/messaging/sleep_policy_test.go`

- [ ] **Step 1: Write the guard**

```go
package messaging

import "testing"

// TestSleeperReadsNothingFromTheClearSightPolicies is the consequence guard for
// M4d's inversion. Sleep is about to move from being written inside each
// predicate to being composed over a shared verdict, and the failure mode is
// silent: a predicate that loses its attention test starts narrating to
// sleepers, and no golden covers that, because no golden drives a sleeping
// observer.
//
// CanSeeSightImpairedOnly is deliberately NOT in this list. It has no attention
// test today and must not gain one: it feeds DarknessCombatPenalty, and a
// sleeping defender is already auto-crit, so doubling their disadvantage was
// never asked for. See predicates.go's own comment.
func TestSleeperReadsNothingFromTheClearSightPolicies(t *testing.T) {
	sleeper := newOpticsObserver(t, opticsCase{asleep: true, lit: true})
	room := newOpticsRoom(t, true)

	if CanSeeClearly(sleeper, room) {
		t.Error("a sleeper must not pass CanSeeClearly, even in a lit room")
	}
	if CanSeeShapes(sleeper, room) {
		t.Error("a sleeper must not pass CanSeeShapes, even in a lit room")
	}
	if !CanSeeSightImpairedOnly(sleeper, room) {
		t.Error("CanSeeSightImpairedOnly must IGNORE sleep: it feeds the combat darkness penalty")
	}
}
```

- [ ] **Step 2: Prove it can fail, in the direction that matters**

Temporarily delete the `HasConditionFlag(conditions.Sleeping)` check from
`CanSeeClearly`. Run:

`go test ./internal/messaging/ -run TestSleeperReadsNothing -v`

Expected: **FAIL** with "a sleeper must not pass CanSeeClearly". Revert and
confirm `git diff internal/messaging/predicates.go` prints nothing.

- [ ] **Step 3: Commit**

```bash
git add internal/messaging/sleep_policy_test.go
git commit -m "test(messaging): a sleeper reads nothing from the clear-sight policies

No golden drives a sleeping observer, so the inversion's worst failure mode is
invisible to every existing test.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Invert the dependency

**Files:**
- Modify: `internal/messaging/predicates.go`

- [ ] **Step 1: Make the verdict compute optics directly**

Rewrite `ParticipantSight` so it consults blindness, room light, NightVision and
InfraredVision itself, and depends on no boolean predicate:

```go
// ParticipantSight is THE optics primitive. It answers what an observer can
// make out, and nothing else: blindness, room light, NightVision,
// InfraredVision.
//
// It does NOT consult sleep. Sleep is an attention property, not an optical
// one -- a sleeping character's eyes work, they are simply not reading -- and
// the policies below compose it where it belongs. Conflating the two is what
// left three predicates each carrying a comment explaining the split.
func ParticipantSight(observer *characters.Character, room RoomVisibility) SightDecision {
	if observer == nil {
		return SightFull
	}
	if observer.Perception != nil && observer.Perception.State() == perception.Blinded {
		return SightNone
	}
	if room == nil || roomIsLit(room) {
		return SightFull
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

- [ ] **Step 2: Redefine the three policies over it**

```go
// awake reports attention. Kept separate from optics on purpose; see
// ParticipantSight.
func awake(observer *characters.Character) bool {
	return observer == nil || !observer.HasConditionFlag(conditions.Sleeping)
}

func CanSeeClearly(observer *characters.Character, room RoomVisibility) bool {
	return awake(observer) && ParticipantSight(observer, room) == SightFull
}

// CanSeeSightImpairedOnly is the optics question with NO attention test, and it
// is SightFull specifically: infrared does not satisfy it. It feeds
// Balance.DarknessCombatPenalty, so widening it would hand every infrared
// character a silent balance change.
func CanSeeSightImpairedOnly(observer *characters.Character, room RoomVisibility) bool {
	return ParticipantSight(observer, room) == SightFull
}

// CanSeeShapes is "full sight OR shapes", written as two equalities on
// purpose. See the ordering note below: a comparison would read backwards.
func CanSeeShapes(observer *characters.Character, room RoomVisibility) bool {
	if !awake(observer) {
		return false
	}
	d := ParticipantSight(observer, room)
	return d == SightFull || d == SightShapes
}
```

🪤 **The constants run BEST-TO-WORST, not worst-to-best.**
`internal/messaging/pipeline.go:34-38` declares `SightFull = iota`, so
`SightFull = 0`, `SightShapes = 1`, `SightNone = 2`. An earlier draft of this
plan wrote `>= SightShapes` meaning "shapes or better"; with this ordering that
expression also matches `SightNone` and would have made a blind character see
shapes. **Do not use ordered comparisons on `SightDecision` anywhere in this
plan.** Use explicit equality, as above.

Keep the long WHY comments that currently sit on `CanSeeSightImpairedOnly`. They
record a real incident (a sleep gate silently applying a darkness penalty to a
sleeping defender in a lit room, corrupting combat analytics). Trim the
now-obsolete "THIS IS A TEMPORARY SEAM" paragraph, since this task is that seam
being closed, and replace it with one line saying M4d closed it.

- [ ] **Step 3: Run the nets**

```bash
go test ./internal/messaging/
go test ./internal/rooms/ ./internal/combat/ ./internal/narration/
git diff --stat internal/narration/testdata
```

Expected: PASS, and the `git diff --stat` prints NOTHING. Task 1's truth table
is the real proof here: all eight rows must still pass unchanged.

- [ ] **Step 4: Commit**

```bash
git add internal/messaging/predicates.go
git commit -m "refactor(messaging): the sight verdict is the primitive, not the boolean

ParticipantSight was defined on top of CanSeeSightImpairedOnly, so a boolean
was the primitive and the richer verdict was derived from it. That is why the
optics were written out three times. Now the verdict computes optics directly
and the three predicates are one-line policies over it, each composing its own
attention rule.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Combat carries the verdict instead of a boolean

**Owner ruling 6 (2026-09-20) sets this task's shape.** Infrared characters are
to take a REDUCED darkness combat penalty rather than the full one they take
today. A boolean cannot express three states, so combat does not get a renamed
boolean: it carries the `SightDecision`.

**This task is still byte-identical.** The mapping stays "full penalty unless
`SightFull`", which is exactly today's behaviour, infrared included. PR 2
changes the mapping. Getting the plumbing in now is what makes that a table
lookup rather than a new predicate.

**Files:**
- Modify: `internal/combat/combat.go` (8 sites), `internal/combat/combat_helpers.go` (struct plus 2 sites)
- Create: `internal/combat/darkness_penalty_verdict_test.go`

- [ ] **Step 1: Carry the verdict on the context**

In `internal/combat/combat_helpers.go`, replace the two `combatContext` fields:

```go
	// sourceSight and targetSight are the OPTICS VERDICT, not a narration gate.
	// They drive Balance.DarknessCombatPenalty on attack and defence scores and
	// nothing else. They carry the full SightDecision rather than a bool
	// because seeing shapes is not the same as seeing nothing: M4d PR 2 gives
	// the shapes case its own reduced penalty, and a bool cannot say that.
	sourceSight messaging.SightDecision
	targetSight messaging.SightDecision
```

At the 8 sites in `internal/combat/combat.go` (`:55,56,106,107,150,151,199,200`),
set them from `messaging.ParticipantSight(char, room)` instead of
`messaging.CanSeeSightImpairedOnly(char, room)`.

⚠️ `ParticipantSight` has no attention test, which is what these sites need and
is why they used `CanSeeSightImpairedOnly` rather than `CanSeeClearly`. The long
comment in `predicates.go` explains the incident behind that (a sleep gate
silently applying a darkness penalty to a sleeping defender in a lit room, and
corrupting combat analytics). Do not reintroduce a sleep test here.

- [ ] **Step 2: Read them at the two scoring sites, identically to today**

`combat_helpers.go:557` and `:749` currently test `!ctx.sourceCanSee` and
`!ctx.targetCanSee`. They become:

```go
	// PR 1 keeps today's rule exactly: any verdict short of SightFull takes the
	// full penalty, infrared included. PR 2 replaces this test with
	// DarknessScoreMultiplier, which gives SightShapes its own reduced value.
	if ctx.sourceSight != messaging.SightFull {
		attackScore *= float64(bal.DarknessCombatPenalty)
	}
```

Update every test that constructs a `combatContext` (there are several; the
compiler will name them all) to set the verdict instead of the bool.
`sourceCanSee: true` becomes `sourceSight: messaging.SightFull`.

- [ ] **Step 3: Write the test that pins the equivalence**

Create `internal/combat/darkness_penalty_verdict_test.go` with a test asserting
that the new field produces the same penalty decision as the old boolean for
every one of the eight observer/room states in
`internal/messaging/optics_pin_test.go`. The row that matters most is **infrared
in the dark: penalty STILL APPLIES in PR 1.** That row is what proves this task
did not quietly deliver ruling 6 early.

Write it against the package's real fixtures. If `internal/combat` cannot build
those states without heavy fixtures, assert the mapping at the level of
`messaging.ParticipantSight(...) != SightFull` in `internal/messaging` instead,
and say so in your report.

- [ ] **Step 4: Confirm the sweep**

```bash
go build ./...
go test ./internal/combat/
grep -rn "CanSeeSightImpairedOnly\|sourceCanSee\|targetCanSee" --include=*.go internal/combat/
```

Expected: build clean, tests pass, and the grep prints nothing. Run the grep
standalone (`grep -c` exits 1 on zero matches).

- [ ] **Step 5: Verify the remaining callers elsewhere are genuinely narration**

```bash
grep -rn "CanSeeSightImpairedOnly" --include=*.go internal/ modules/ | grep -v _test
```

Every remaining site should be a narration or perception decision, not a scoring
one. List them in your report with a word each on which it is. A scoring site
found here wanted the verdict too, and this task missed it.

- [ ] **Step 4: Verify the remaining callers are genuinely narration**

```bash
grep -rn "CanSeeSightImpairedOnly" --include=*.go internal/ modules/ | grep -v _test
```

Every remaining site should be a narration or perception decision, not a scoring
one. List them in your report with a word each on which it is. If one is a
scoring site, it wanted the verdict too, and this task missed it.

- [ ] **Step 5: Commit**

```bash
git add internal/combat/combat.go internal/combat/combat_helpers.go internal/combat/darkness_penalty_verdict_test.go
git commit -m "refactor(combat): the combat context carries the sight verdict

combat borrowed a messaging sight predicate for a scoring rule, which is why
sourceCanSee and targetCanSee looked like narration flags and were not. They
now carry the SightDecision itself, because owner ruling 6 gives seeing shapes
its own reduced penalty in PR 2 and a bool cannot say that.

Byte-identical: the scoring sites still apply the full penalty for any verdict
short of SightFull, infrared included. Only the mapping moves in PR 2.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

⚠️ Adjust that `git add` to the exact files you changed. Never `git add -A`.

---

## Task 5: The fourth audience

**Files:**
- Modify: `internal/messaging/trio.go`
- Modify: `messaging_surface_guard_test.go`
- Modify: `internal/combat/combat.go` (the ranged seat)

- [ ] **Step 1: Census the literals before changing the guard**

```bash
grep -rn "messaging.Trio{\|Trio{" --include=*.go internal/ modules/ | grep -v _test | wc -l
```

**Report the number.** The plan's assumption is that adding a required fourth
role touches roughly this many literals. If it is above 60, STOP and report:
that is a large mechanical diff for a field most events leave empty, and the
owner should choose between a required fourth role and an optional one before
you spend it.

- [ ] **Step 2: Add the seat**

```go
// Trio is the four audiences of one narrated event. RemoteObserver is the
// SECOND room: ranged combat narrates to the attacker's room and the
// defender's room, and until M4d that second audience had no seat here at all,
// so it travelled outside the pipeline.
type Trio struct{ Actor, Actee, Observer, RemoteObserver Line }
```

Add the delivery in `SendTrio`, mirroring the observer branch, gated on a
`RemoteRoom` field added to `Audience`. A nil `RemoteRoom` sends nothing, which
is every event except ranged combat.

- [ ] **Step 3: Move the guard to four roles**

In `messaging_surface_guard_test.go:1636`, extend the roles list to include
`"RemoteObserver"`. Run the guard; it will name every literal missing the field.
Add `RemoteObserver: {}` (or the zero `Line`) to each.

- [ ] **Step 4: Seat the ranged audience**

`internal/combat/combat.go:316,341` builds `toDefenderRoomMsg` from
`roles.ActeeObserver` and sends it with `SendToTargetRoom`. Route it through the
new `RemoteObserver` seat so it is sight-judged like every other audience.

⚠️ This is the one place in PR 1 where behaviour could move: the defender's room
currently receives that line through a path that may not hide names the way
`SendTrio` does. **Check before you change it.** If `SendToTargetRoom` already
sight-gates and hides names, this is plumbing. If it does not, then seating it
changes what a bystander in the defender's room reads in the dark, which is a PR
2 decision: leave `SendToTargetRoom` in place, add the seat unused, and report.

- [ ] **Step 5: Gates and commit**

```bash
go build ./...
go test . ./...
git diff --stat internal/narration/testdata
```

Expected: PASS, and no golden movement.

```bash
git commit -m "feat(messaging): Trio seats its fourth audience

RemoteObserver is ranged combat's defender-room audience, which M4b-1 named in
the surface registry but never gave a seat, so it travelled outside the
pipeline. The literal guard now requires all four roles.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: The remaining paths, one at a time, each proven text-identical

**This task is four independent sub-tasks. Do them one at a time and commit
separately**, so that a path which turns out to change text can be dropped
without unpicking the others.

For EACH of: crafting (`internal/usercommands/craft.go`), quests (find the
sender; start from `internal/quests` and the `send_text` reward path),
caster-only spell effects (`internal/hooks/spell_resolution.go`, the
`applyPlayerEffect` self-only lines), and `position_control`
(`internal/hooks/Position_Messaging.go`):

- [ ] **Step A: Prove the path carries no other party's name**

`SendTrio` hides the OTHER party's name from each participant by their sight.
So the move is text-identical if and only if the path's lines never contain a
name that `SendTrio` would hide.

Read every line the path sends. For each, answer in your report: does it name
anyone other than the reader? Crafting's lines are `CategorySystem` and
self-only, so the expected answer is no. `position_control`'s stamina warning is
addressed to the CHARACTER while the controller is a different person, which
M4a flagged as the one asymmetric mapping in that store, so it needs care.

- [ ] **Step B: If yes, STOP for that path.** Do not move it. Record it in your
report as deferred to PR 2 with the line that would change. Move to the next
path. This is a success, not a failure: PR 1's contract is no text movement.

- [ ] **Step C: If no, move it to `SendTrio`** with an `Audience` whose `Actee`
and `ActeeName` are unset where there is no second party.

- [ ] **Step D: Prove it**

```bash
go test ./internal/... ./modules/...
git diff --stat internal/narration/testdata
```

Expected: PASS and no golden movement. Commit that path alone:

```
refactor(<pkg>): <path> delivers through SendTrio

Text-identical: its lines name no one but the reader, so the per-reader name
hiding SendTrio adds has nothing to hide.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
```

---

## Task 7: The one-path guard

**Files:**
- Modify or create: a guard beside `raw_events_message_guard_test.go`

- [ ] **Step 1: Write the guard**

An AST guard asserting that narration categories leave only through `SendTrio`:
a `SendText` call whose category argument is a narration category, outside the
allowed files, is a failure. Model it on
`TestNoRawEventsMessageOutsidePipeline` (`raw_events_message_guard_test.go:27`),
including its allowlist shape.

⚠️ **The allowlist is the whole design.** Any path Task 6 deferred to PR 2 must
be ON it, with a comment naming PR 2 and why. A guard that cannot fail because
everything is allowlisted is worse than no guard: read the allowlist back and
confirm each entry has a reason.

- [ ] **Step 2: Prove it can fail**

Add a `SendText` with a narration category to a non-allowlisted file. Confirm
the guard reddens and names that file. Remove it.

- [ ] **Step 3: Commit**

```bash
git add <the guard file>
git commit -m "test: narration categories leave only through SendTrio

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: Docs, gates, PR

- [ ] **Step 1: context.md**

`internal/messaging/context.md`: the verdict is the primitive; the three
predicates are policies over it; the combat context carries the verdict as its scoring input
and not a narration gate; `Trio` has four roles.
`internal/combat/context.md`: the renamed context fields and what they drive.

```bash
python tools/context_md_audit.py
```

- [ ] **Step 2: Sweep by meaning**

Run each standalone:

```bash
grep -rn "TEMPORARY SEAM\|collapse back into one" --include=*.go internal/
grep -rni "three predicates\|borrows a sight predicate\|sourceCanSee" --include=*.md internal/ docs/
```

Every claim that the seam is temporary, or that combat borrows a sight
predicate, is now false.

- [ ] **Step 3: Full gate**

```bash
go build ./...
go test . ./...
golangci-lint run --new-from-rev=master
gofmt -l $(git diff --name-only master..HEAD | grep '\.go$')
```

Expected: build clean, tests pass, 0 new lint issues, and `gofmt -l` prints
NOTHING. Run the gofmt line standalone: CI's `validate / test` job runs a gofmt
gate separate from golangci-lint that kills the whole job in about 30 seconds.

- [ ] **Step 4: Boot check**

```bash
go build -o /c/tmp/dogmud_m4d.exe .
```

Start it with the working directory set to the repo, wait about 45 seconds, then
check the log STANDALONE for `Server Ready` and for `PANIC`. 🪤 A failed boot
exits 0, so the log is the check, never `$?`. Kill the server by the PID you
started; the owner runs their own server on this machine.

- [ ] **Step 5: PR**

```bash
git push -u origin feature/messaging-m4d1-sight-and-send-path
gh pr create --repo pruuk/DOGMud --base master --title "M4d PR 1: one sight verdict, four audiences, one path" --body-file <body>
```

🪤 Every `gh` command carries `--repo pruuk/DOGMud`; this repo is a fork.

The body must state: that no player-visible text moves and how that was proven
per path; which paths (if any) Task 6 deferred to PR 2 and why; the Trio literal
census from Task 5 Step 1; and that `CanSeeSightImpairedOnly` remains
`== SightFull` deliberately, with the infrared row of the truth table as the
reason.

---

## Self-review

**Spec coverage.** The spec's PR 1 has four parts: one sight producer (Tasks 1
to 3), combat's own named predicate (Task 4), the fourth audience (Task 5), and
one path plus its guard (Tasks 6 and 7). The spec's stated risk, that sleep
composition is invisible to goldens, is Task 2, which comes before the
inversion. Covered.

**Placeholders.** One deliberate: Task 4 Step 2 states what the test must
assert rather than giving a body, because the fixture shape depends on what
`internal/combat` can construct, and guessing it would produce code that
compiles against nothing. It is marked, with a fallback location.

**Type consistency.** `ParticipantSight` keeps its existing name and signature
throughout. `awake` is defined in Task 3 Step 2 and used only there.
`sourceSight` and `targetSight` are defined in Task 4 Step 1 and read in Step 2.
`opticsCase`, `newOpticsObserver` and `newOpticsRoom` are defined in Task 1 and
reused by name in Task 2.

**Scope.** Task 6 is the one that can grow. Its structure is per-path with an
explicit stop rule, so a path that turns out to move text leaves PR 1 rather
than expanding it.
