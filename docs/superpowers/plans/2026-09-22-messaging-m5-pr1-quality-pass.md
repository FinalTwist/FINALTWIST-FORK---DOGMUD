# M5 PR 1: The Quality Pass Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the last bare-name leaks in condition and throw narration, widen the guard that catches the whole class, point the templates test suite at the world players actually read, and retire the last raw `CategorySubmission` sender.

**Architecture:** Six independent tasks against existing seams. Nothing new is invented: the name-hiding path (`sendTextVisualJudgedBy`), the identity guard (`TestObserverIdentityTagsAreAnonymizable`) and the SendTrio-only guard all exist and are extended in place. Tasks 1 and 2 fix code; Task 3 records those fixes as proofs in the guard's existing safe map, so it must land after them.

**Tech Stack:** Go, `internal/hooks`, `internal/rooms`, `internal/messaging`, `internal/usercommands`, `internal/templates`, root-package guard tests.

---

## Facts verified against source

Read from the tree on 2026-09-22 at master `a5613be95`, plus two corrections made while writing this plan.

| # | Fact | Evidence |
|---|------|----------|
| 1 | `sendConditionEndRoomText` passes no names, so `HideNames` never runs on an end line | `internal/hooks/NewTurn_PruneConditions.go:130-138` |
| 2 | It routes light conditions to `SendTextVisualAsLit` and everything else to `SendTextVisual`. Neither takes names | same, `:132` and `:137` |
| 3 | `SendTextVisualHidingNames` exists; there is no `SendTextVisualAsLitHidingNames` | `internal/rooms/rooms.go:315`, `:338` |
| 4 | The shared path already takes names, so the missing method is a one-line wrapper | `internal/rooms/rooms.go:353` |
| 5 | Only a `SightShapes` reader is hidden from. `SightFull` returns the text untouched | `internal/rooms/rooms.go:370`, `internal/messaging/hidenames.go:39` |
| 6 | 🔴 **CORRECTION.** `SightShapes` is structurally unreachable on the `AsLit` path, so condition 1 cannot leak. `litRoom{}.GetVisibility()` is always 1, and `ParticipantSight` only returns `SightShapes` when the room is NOT lit | `internal/rooms/rooms.go:345`, `internal/messaging/predicates.go:51-68` |
| 7 | So exactly ONE authored end line can leak today: `9-hidden.yaml:15`, `"{actee_plain} emerges from the shadows."`, which has no light flag | `_datafiles/world/dogmud/conditions/9-hidden.yaml:15` |
| 8 | `1-illumination.yaml:12` is safe by construction, not by accident. Nothing pins that today | `_datafiles/world/dogmud/conditions/1-illumination.yaml:12` |
| 9 | The start and trigger phases already pass the holder's plain name | `Condition_ApplyConditions.go:170`, `NewRound_UserRoundTick.go:302`, `NewRound_MobRoundTick.go:287` |
| 10 | `throw.go` builds ONE Audience with `ActeeName: messaging.NoName` | `internal/usercommands/throw.go:289-295` |
| 11 | `player_cast_interrupt` renders a real mob name into `{actee_plain}` | `throw.go:363-366`; `_datafiles/world/dogmud/narration/special-moves/throw.yaml:26,28` |
| 12 | `NoName` means "nobody to hide", at both the participant and room hide sites | `internal/messaging/trio.go:127`, `internal/messaging/hidenames.go:48` |
| 13 | `sendMoveEvent` takes the Audience BY VALUE, so a per-event copy is safe | `internal/usercommands/move_narration.go:97` |
| 14 | The identity guard walks only two stores | `shipped_narration_data_guard_test.go:764-767` |
| 15 | Its pattern cannot match a `_plain` token | `shipped_narration_data_guard_test.go:837`, `\{actor\}\|\{actee\}` |
| 16 | The safe map is the established idiom for "safe by a runtime guarantee content cannot see" | `shipped_narration_data_guard_test.go:769-800` |
| 17 | Only 16 shipped files use a `_plain` token: 14 conditions plus `grapple.yaml` and `throw.yaml` | exhaustive grep over `_datafiles` |
| 18 | The templates test binary reads `world/default` | `internal/templates/process_test.go:27`, used at `:34`, `:40`, `:366`, `:418` |
| 19 | Flipping it breaks exactly two tests | probed 2026-09-22, both failures reproduced |
| 20 | `item_procs.go` is the last raw `CategorySubmission` sender | `internal/hooks/item_procs.go:274-275` |
| 21 | The SendTrio-only guard covers three categories | `send_trio_only_guard_test.go`, `sendTrioOnlyCategories` |
| 22 | `context.md`'s numbered list has 7 stages, its file table lists 6 and omits the sight gate | `internal/messaging/context.md:8-23` and `:259` |

### 🔴 The correction this plan makes to the spec

The spec (fact 19 in its own table) says **two** End-phase lines leak. Reading
the delivery path proves only **one** can.

`SendTextVisualAsLit` judges sight against `litRoom{}`, whose `GetVisibility()`
is always 1. `ParticipantSight` returns `SightShapes` only for an unblinded
observer in a room that is NOT lit. Under `litRoom{}` that branch is
unreachable, so an `AsLit` reader is always either `SightFull` (gets the line
whole, correctly, because the light really was there) or `SightNone` (gets
nothing). `HideNames` is only consulted at `SightShapes`.

So `1-illumination.yaml`'s bare `{actee_plain}` is safe by construction.
**Nothing pins that, and it is exactly the kind of accidental safety that a
later change to `SendTextVisualAsLit` would silently remove.** Task 1 still
threads names through both paths, and adds a test that pins the structural
reason, so a future shapes tier on the lit path fails loudly instead of leaking.

---

## Task 1: the End-phase condition leak

**Files:**
- Modify: `internal/rooms/rooms.go` (add `SendTextVisualAsLitHidingNames` beside `SendTextVisualAsLit` at `:338`)
- Modify: `internal/hooks/NewTurn_PruneConditions.go:130-138` and its two call sites at `:51` and `:110`
- Test: `internal/hooks/condition_room_text_test.go`
- Test helper: `internal/hooks/narration_testhelpers_test.go`

- [ ] **Step 1: Add the two test conditions the new tests need**

In `internal/hooks/narration_testhelpers_test.go`, extend the const block
(currently ending at `dozeConditionId = 7007`):

```go
	dozeConditionId      = 7007 // puts the bearer to sleep; RoundInterval 0, so it never ticks
	shadeConditionId     = 7008 // end_observer with a BARE {actee_plain}, mirrors shipped condition 9
	emberConditionId     = 7009 // a light source whose end_observer has a BARE {actee_plain}, mirrors shipped condition 1
)
```

Then add both to the map inside `seedNarrationConditions`, after the
`dozeConditionId` entry:

```go
		shadeConditionId: {ConditionId: shadeConditionId, Name: "Test Shade", RoundInterval: 5, TriggerCount: 3,
			EndRoomText: "{actee_plain} emerges from the shadows."},
		emberConditionId: {ConditionId: emberConditionId, Name: "Test Ember", RoundInterval: 5, TriggerCount: 3,
			Flags: []conditions.Flag{conditions.EmitsLight},
			EndRoomText: "The glow surrounding {actee_plain} fades away."},
```

Both strings are the shipped lines verbatim, from `9-hidden.yaml:15` and
`1-illumination.yaml:12`.

- [ ] **Step 2: Write the failing test for the plain path**

Append to `internal/hooks/condition_room_text_test.go`:

```go
// TestConditionEndRoomText_InfraredObserverDoesNotReadABareHolderName is the
// End-phase twin of the start and trigger fixes in 39b75fe07. A bare
// {actee_plain} is invisible to tag-based Anonymize, so unless the sender
// passes the holder's plain name into HideNames, a shapes-only observer reads
// the real name. Shipped condition 9 authors exactly this line.
func TestConditionEndRoomText_InfraredObserverDoesNotReadABareHolderName(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	darken(t, 1)
	holder := users.GetByUserId(1)
	require.True(t, holder.Character.Conditions.AddCondition(shadeConditionId, false))
	require.True(t, users.GetByUserId(2).Character.Conditions.AddCondition(heatEyesConditionId, true))
	expire(t, holder.Character.Conditions.List, shadeConditionId)
	drainPlain(2)

	PruneConditions(events.NewTurn{TurnNumber: 1})

	lines := drainPlain(2)
	require.Equal(t, 1, countContaining(lines, "emerges from the shadows"),
		"the shapes observer must still receive the line, or this test proves nothing: %v", lines)
	assert.Zero(t, countContaining(lines, "Aliceia"),
		"an infrared-only observer read the holder's bare name: %v", lines)
}
```

- [ ] **Step 3: Run it and watch it fail**

Run: `go test ./internal/hooks/ -run TestConditionEndRoomText_InfraredObserverDoesNotReadABareHolderName -v`

Expected: FAIL, on the second assertion, with `an infrared-only observer read
the holder's bare name:` followed by a line containing `Aliceia emerges from
the shadows.` The first assertion must PASS, which is what proves the line is
being delivered and the test is measuring hiding rather than silence.

- [ ] **Step 4: Add the missing room method**

In `internal/rooms/rooms.go`, immediately after `SendTextVisualAsLit` (which
ends at `:340`), add:

```go
// SendTextVisualAsLitHidingNames is SendTextVisualAsLit for a line that names
// the parties to an event.
//
// 🔑 NO READER OF THIS PATH IS EVER AT SightShapes, so names is never
// consulted today: litRoom{} reports the room lit, and ParticipantSight only
// returns SightShapes for an unblinded observer in an UNLIT room. It exists so
// that the light and non-light end-text paths are threaded identically, and so
// that if SendTextVisualAsLit ever grows a shapes tier, the names are already
// there rather than newly missing. TestConditionEndRoomText_LightPathHasNoShapesTier
// pins the reason.
func (r *Room) SendTextVisualAsLitHidingNames(cat messaging.Category, txt string, names []string, excludeUserIds ...int) {
	r.sendTextVisualJudgedBy(litRoom{}, cat, txt, names, excludeUserIds...)
}
```

- [ ] **Step 5: Thread names through the sender**

In `internal/hooks/NewTurn_PruneConditions.go`, replace `sendConditionEndRoomText`
entirely:

```go
// sendConditionEndRoomText sends a condition's end room line on the visual channel. A
// light condition's line is judged as if the room were still lit, because its light
// went out when the condition expired, a round before this prune: see
// Room.SendTextVisualAsLit. Every other end line is judged by the room as it is.
//
// names is the holder's PLAIN name. An end line may author a bare
// {actee_plain} (shipped conditions 1 and 9 both do), which tag-based
// Anonymize cannot see, so the name must be handed to HideNames explicitly.
// This is the End-phase twin of the start and trigger senders.
func sendConditionEndRoomText(r *rooms.Room, spec *conditions.ConditionSpec, msg string, names []string, skip ...int) {
	for _, flag := range spec.Flags {
		if flag == conditions.EmitsLight {
			r.SendTextVisualAsLitHidingNames(messaging.CategoryConditionExpire, msg, names, skip...)
			return
		}
	}
	r.SendTextVisualHidingNames(messaging.CategoryConditionExpire, msg, names, skip...)
}
```

- [ ] **Step 6: Pass the plain name at the player call site**

In the same file, at the player branch (currently `:51`), the plain name is
already computed one line above for `Narrate`. Replace:

```go
								sendConditionEndRoomText(r, endConditionSpec, roles.Observer, user.UserId)
```

with:

```go
								sendConditionEndRoomText(r, endConditionSpec, roles.Observer,
									[]string{user.Character.GetCharacterName(false)}, user.UserId)
```

- [ ] **Step 7: Pass the plain name at the mob call site**

At the mob branch (currently `:110`), replace:

```go
							sendConditionEndRoomText(r, endConditionSpec, roles.Observer)
```

with:

```go
							sendConditionEndRoomText(r, endConditionSpec, roles.Observer,
								[]string{mob.Character.GetCharacterName(false)})
```

- [ ] **Step 8: Run the test and watch it pass**

Run: `go test ./internal/hooks/ -run TestConditionEndRoomText_InfraredObserverDoesNotReadABareHolderName -v`

Expected: PASS. The delivered line now reads `A figure emerges from the shadows.`

- [ ] **Step 9: Pin why the light path is safe**

Append to `internal/hooks/condition_room_text_test.go`:

```go
// TestConditionEndRoomText_LightPathHasNoShapesTier pins the structural reason
// shipped condition 1 ("The glow surrounding {actee_plain} fades away.") does
// not leak despite authoring a bare name: SendTextVisualAsLit judges sight
// against litRoom{}, and ParticipantSight returns SightShapes only for an
// UNLIT room, so no reader of that path is ever at the one tier HideNames acts
// on. An infrared observer therefore reads the real name here, correctly, and
// a blind or sleeping one reads nothing.
//
// If this test ever fails, SendTextVisualAsLit has grown a shapes tier and
// every light condition authoring a bare _plain token became a live leak.
func TestConditionEndRoomText_LightPathHasNoShapesTier(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	darken(t, 1)
	holder := users.GetByUserId(1)
	require.True(t, holder.Character.Conditions.AddCondition(emberConditionId, false))
	require.True(t, users.GetByUserId(2).Character.Conditions.AddCondition(heatEyesConditionId, true))
	expire(t, holder.Character.Conditions.List, emberConditionId)
	drainPlain(2)

	PruneConditions(events.NewTurn{TurnNumber: 1})

	lines := drainPlain(2)
	require.Equal(t, 1, countContaining(lines, "fades away"),
		"the light line must reach the observer, or this test proves nothing: %v", lines)
	assert.Equal(t, 1, countContaining(lines, "Aliceia"),
		"the lit path has no shapes tier, so the name is expected here: %v", lines)
}
```

- [ ] **Step 10: Run the whole condition room-text suite**

Run: `go test ./internal/hooks/ -run TestConditionEndRoomText -v`

Expected: PASS, every test in the family, including the three pre-existing
light-condition tests, which must be unaffected.

- [ ] **Step 11: Run the package**

Run: `go test ./internal/hooks/ ./internal/rooms/`

Expected: `ok` for both.

- [ ] **Step 12: Commit**

```bash
git add internal/rooms/rooms.go internal/hooks/NewTurn_PruneConditions.go internal/hooks/condition_room_text_test.go internal/hooks/narration_testhelpers_test.go
git commit -F - <<'EOF'
fix(conditions): the end phase hides a bare holder name like its siblings

The start and trigger phases learned to pass the holder's plain name into
HideNames on 2026-09-21 (39b75fe07). The end phase never did, so shipped
condition 9's "{actee_plain} emerges from the shadows." reached a
shapes-only observer as the real name. Tag-based Anonymize cannot see a
bare name; only HideNames can, and only when the sender hands it one.

Condition 1's light line turns out to be safe by construction rather than
by accident: SendTextVisualAsLit judges against litRoom{}, and
ParticipantSight yields SightShapes only for an UNLIT room, so no reader
of that path is ever at the tier HideNames acts on. Both paths are
threaded identically anyway, and a new test pins the structural reason so
that a future shapes tier on the lit path fails loudly instead of leaking
every light condition that authors a bare token.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 2: the `throw.go` interrupt leak

**Files:**
- Modify: `internal/usercommands/throw.go:284-295` (the comment and the Audience) and `:363-366` (the send)
- Test: `internal/usercommands/throw_interrupt_hiding_test.go` (create)

- [ ] **Step 1: Read the current site**

Run: `sed -n '284,296p;360,368p' internal/usercommands/throw.go`

Expected: the Audience literal carrying `ActeeName: messaging.NoName`, and the
`sendMoveEvent("throw", "player_cast_interrupt", ...)` call passing `aud`.
Confirm both before editing.

- [ ] **Step 2: Write the failing test**

Create `internal/usercommands/throw_interrupt_hiding_test.go`:

```go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestThrowInterruptAudienceNamesTheActee proves the cast-interrupt event's
// Audience carries the mob's name, so SendTrio can hide it from a reader who
// cannot see.
//
// throw is area-effect and genuinely actee-less, so it builds ONE Audience with
// ActeeName: NoName and reuses it for every event. player_cast_interrupt is the
// exception: it renders a single mob's name into throw.yaml's {actee_plain}.
// NoName means "nobody to hide" at trio.go:127 and hidenames.go:48, so before
// this fix the mob's name reached every reader at every sight tier.
func TestThrowInterruptAudienceNamesTheActee(t *testing.T) {
	base := messaging.Audience{
		ActorName: "Aliceia",
		ActeeName: messaging.NoName,
	}

	got := throwInterruptAudience(base, "Cave Crawler")

	require.Equal(t, messaging.NoName, base.ActeeName,
		"the base Audience must not be mutated: every other throw event depends on it staying actee-less")
	assert.Equal(t, "Cave Crawler", got.ActeeName,
		"the interrupt event must name the mob so SendTrio can hide it")
	assert.Equal(t, "Aliceia", got.ActorName,
		"the copy must keep every other field")
}
```

- [ ] **Step 3: Run it and watch it fail**

Run: `go test ./internal/usercommands/ -run TestThrowInterruptAudienceNamesTheActee -v`

Expected: FAIL to compile, with `undefined: throwInterruptAudience`.

- [ ] **Step 4: Add the helper and use it**

In `internal/usercommands/throw.go`, add immediately above the `Throw`
function's Audience literal:

```go
// throwInterruptAudience returns a copy of throw's actee-less Audience that
// names one mob, for the single event that has an actee.
//
// It returns a copy rather than mutating: every other throw event in the same
// loop reuses the base Audience and must stay actee-less.
func throwInterruptAudience(base messaging.Audience, mobName string) messaging.Audience {
	base.ActeeName = mobName
	return base
}
```

Then replace the comment above the Audience literal (currently at `:284-287`)
so it no longer contradicts the code:

```go
	// Throw is an AREA effect against mobs, so it has no actee at all and
	// every Trio below writes Actee: messaging.NoLine. That is correct, not an
	// oversight: the M1 audit ruled this file's 17 actor sends and zero actee
	// sends as the one genuinely actee-less member of the special-move family.
	//
	// ONE EXCEPTION: player_cast_interrupt names a single mob, because it
	// reports what happened to that mob's cast. It takes a named copy of this
	// Audience via throwInterruptAudience so SendTrio can hide that name from a
	// reader who cannot see; the base Audience below stays actee-less for
	// every other event.
```

- [ ] **Step 5: Pass the named copy at the send site**

Replace the send at `:363-366`:

```go
		if maybeInterruptOnThrow(mob, matchItem.ItemId, state.ActorRef{UserId: user.UserId}) {
			sendMoveEvent("throw", "player_cast_interrupt",
				moveIdentities{ActeePlain: mob.Character.Name},
				throwInterruptAudience(aud, mob.Character.Name),
				sameMoveCategory(messaging.CategorySpellDisruption), nil)
		}
```

- [ ] **Step 6: Run the test and watch it pass**

Run: `go test ./internal/usercommands/ -run TestThrowInterruptAudienceNamesTheActee -v`

Expected: PASS.

- [ ] **Step 7: Run the package**

Run: `go test ./internal/usercommands/`

Expected: `ok`. The special-move byte-identity net lives here and must stay
green: this task changes no authored wording, only which name the hide list
carries.

- [ ] **Step 8: Commit**

```bash
git add internal/usercommands/throw.go internal/usercommands/throw_interrupt_hiding_test.go
git commit -F - <<'EOF'
fix(throw): the cast-interrupt event names the mob it is about

throw is area-effect and genuinely actee-less, so it builds one Audience
with ActeeName: NoName and reuses it for all seventeen sends. One event
breaks that rule: player_cast_interrupt renders a single mob's name into
throw.yaml's {actee_plain}, because it reports what happened to that
mob's cast.

NoName means "nobody to hide" at both trio.go:127 and hidenames.go:48, so
that mob's name was reaching every reader at every sight tier, including a
blind one. The fix is a per-event copy of the Audience rather than a
mutation, because every other event in the same loop depends on the base
staying actee-less. The file's comment claimed throw has no actee at all;
it now records the exception instead of contradicting the code.

Found while designing M5. It is in no prior document.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 3: widen the identity guard to cover the whole class

**Files:**
- Modify: `shipped_narration_data_guard_test.go` (`observerIdentityGuardRoots` at `:764`, `observerIdentityGuardContentSafeViaCode` at `:769`, `identityPlaceholderPattern` at `:837`, and the test's file-count floor at `:895`)

**This task MUST land after Tasks 1 and 2**, because its safe-map entries assert
the fixes those tasks make.

- [ ] **Step 1: Widen the walk roots**

Replace `observerIdentityGuardRoots`:

```go
var observerIdentityGuardRoots = []string{
	shippedWorldRoot + "/combat-messages",
	shippedWorldRoot + "/defense-messages",
	shippedWorldRoot + "/conditions",
	shippedWorldRoot + "/narration/special-moves",
}
```

- [ ] **Step 2: Widen the placeholder pattern**

Replace `identityPlaceholderPattern` and its comment:

```go
// identityPlaceholderPattern finds a bare {actor}, {actee}, {actor_plain} or
// {actee_plain} token in an authored line. The two _plain variants are the
// dangerous ones: they are untagged by definition, so messaging.Anonymize,
// which strips identity TAGS only, cannot see them at all. They are safe only
// when the delivery path hands the name to HideNames, which is a runtime
// property this content-only guard cannot verify and which
// observerIdentityGuardContentSafeViaCode therefore records by hand.
//
// It does not match inside {actortype}/{acteetype}: those do not end in `}`
// immediately after "actor"/"actee", so the exact-token anchor is enough.
var identityPlaceholderPattern = regexp.MustCompile(`\{actor\}|\{actee\}|\{actor_plain\}|\{actee_plain\}`)
```

- [ ] **Step 3: Run the guard and watch it go red**

Run: `go test . -run TestObserverIdentityTagsAreAnonymizable -v`

Expected: FAIL, with one error per bare `_plain` token across the 16 newly
walked files, each reading `is not anonymizable: it does not sit inside a
closed ansi tag whose alias messaging.Anonymize recognises`. Expect 17
condition-line errors plus the `grapple.yaml` and `throw.yaml` lines. **This
red run is the proof the widened pattern and roots actually reach new
content.** Record the exact count before continuing.

- [ ] **Step 4: Record each store's proof in the existing safe map**

Append to `observerIdentityGuardContentSafeViaCode`, keeping its existing
"name the file and the renderer that makes it safe" shape. Add above the map a
paragraph to its doc comment:

```go
// A SECOND safety route exists for the _plain tokens this guard also matches.
// A bare _plain name is never anonymizable by content, so no ansi alias can
// rescue it; it is safe only when its DELIVERY PATH passes that name into
// messaging.HideNames. That is a runtime guarantee, invisible to a
// content-only walk, so each store is named here with the sender that provides
// it:
//
//	conditions/*.yaml            -> all three phases pass the holder's plain
//	                                name: start Condition_ApplyConditions.go,
//	                                trigger NewRound_{User,Mob}RoundTick.go,
//	                                end sendConditionEndRoomText (M5 PR 1
//	                                Task 1). 1-illumination.yaml is safe for a
//	                                second reason as well: its light flag routes
//	                                it through SendTextVisualAsLit, which has no
//	                                SightShapes tier at all.
//	narration/special-moves/*.yaml -> messaging.SendTrio hides Audience
//	                                ActorName and ActeeName from every reader by
//	                                that reader's ParticipantSight. throw.yaml
//	                                relies on M5 PR 1 Task 2, which stopped its
//	                                interrupt event passing NoName.
```

Then add the sixteen entries themselves, in the map's existing
`"basename.yaml": "reason",` style. The exhaustive set, confirmed by
`grep -rln "actee_plain\|actor_plain" _datafiles/world/dogmud/`:

```go
	// Conditions: all three narration phases pass the holder's plain name
	// into HideNames (start Condition_ApplyConditions.go:170, trigger
	// NewRound_UserRoundTick.go:302 and NewRound_MobRoundTick.go:287, end
	// sendConditionEndRoomText, M5 PR 1 Task 1).
	"1-illumination.yaml":     "condition end text; also routed through SendTextVisualAsLit, which has no SightShapes tier",
	"2-stunned.yaml":          "condition observer text; holder plain name passed to HideNames",
	"3-blinded.yaml":          "condition observer text; holder plain name passed to HideNames",
	"9-hidden.yaml":           "condition observer text; holder plain name passed to HideNames",
	"106-searing_backlash.yaml": "condition observer text; holder plain name passed to HideNames",
	"107-rimefrost.yaml":      "condition observer text; holder plain name passed to HideNames",
	"108-static_shock.yaml":   "condition observer text; holder plain name passed to HideNames",
	"109-reeling.yaml":        "condition observer text; holder plain name passed to HideNames",
	"110-mired.yaml":          "condition observer text; holder plain name passed to HideNames",
	"111-ensnared.yaml":       "condition observer text; holder plain name passed to HideNames",
	"112-paralysed.yaml":      "condition observer text; holder plain name passed to HideNames",
	"114-cursed.yaml":         "condition observer text; holder plain name passed to HideNames",
	"115-rending_bleed.yaml":  "condition observer text; holder plain name passed to HideNames",
	"116-terrified.yaml":      "condition observer text; holder plain name passed to HideNames",
	// Special moves: messaging.SendTrio hides Audience ActorName and ActeeName
	// from every reader by that reader's ParticipantSight.
	"grapple.yaml":            "SendTrio hides both Audience names by reader sight",
	"throw.yaml":              "SendTrio hides both Audience names by reader sight; the interrupt event stopped passing NoName in M5 PR 1 Task 2",
```

🪤 **Cross-check this list against the red run's output from Step 3.** If the
run named a file that is not here, the grep and the guard disagree and the
guard is right: investigate before adding it. If a file here was NOT named, it
authors no observer-role `_plain` token and must NOT be added, because a stale
allowlist entry is exactly what the map's own `seen` check exists to catch.

- [ ] **Step 5: Run the guard and watch it pass**

Run: `go test . -run TestObserverIdentityTagsAreAnonymizable -v`

Expected: PASS, with the log line reporting a file count and placeholder count
both substantially higher than before, and a non-zero exempt count naming the
newly listed files.

- [ ] **Step 6: Raise the file-count floor**

The test asserts `filesInspected < 10` is an error (`:895`). The walk now
covers four roots and far more files. Raise the floor to match reality, and
keep it a floor rather than a pin:

```go
	// A floor, not a pin. Content volume moves; a walk collapsing to a
	// handful of files does not happen for a legitimate reason. Four roots
	// now: combat-messages, defense-messages, conditions, special-moves.
	if filesInspected < 40 {
		t.Errorf("inspected only %d files across %d walk roots: expected the whole combat, defense, condition and special-move message tree", filesInspected, len(observerIdentityGuardRoots))
	}
```

Before committing the number, confirm the real count from Step 5's log line and
set the floor comfortably below it.

- [ ] **Step 7: SABOTAGE. Prove the widened guard can still fail**

A guard that cannot fail is not a guard, and this project has been bitten by
that repeatedly. Author a bare token into a file that is NOT in the safe map:

```bash
cp _datafiles/world/dogmud/conditions/15-sleeping.yaml /tmp/15-sleeping.bak
```

Add a new observer line to a condition file that has none, or temporarily
remove one file's entry from the safe map. Then run:

Run: `go test . -run TestObserverIdentityTagsAreAnonymizable -v`

Expected: FAIL, naming that exact file and line.

- [ ] **Step 8: Revert the sabotage and confirm green**

```bash
cp /tmp/15-sleeping.bak _datafiles/world/dogmud/conditions/15-sleeping.yaml
rm /tmp/15-sleeping.bak
git status --short
```

Expected: no modification to any `_datafiles` path. Then run:

Run: `go test . -run TestObserverIdentityTagsAreAnonymizable -v`

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add shipped_narration_data_guard_test.go
git commit -F - <<'EOF'
test(guard): the identity guard covers bare _plain tokens and two more stores

The guard already asked the right question, of the wrong surface. It
walked combat-messages and defense-messages only, and its pattern
`{actor}|{actee}` could not match a _plain token at all, which is the
variant that actually leaks: a _plain name is untagged by definition, so
Anonymize cannot see it under any alias.

Widened to four roots and four tokens. The 16 shipped files that use a
_plain token are recorded in the existing content-safe-via-code map with
the sender that makes each safe, because "the delivery path passes this
name to HideNames" is a runtime guarantee a content-only walk cannot
verify. That is the same idiom the map already used for the ten
defense-messages files.

Proven capable of failing by sabotage before being trusted, and the
file-count floor raised so the wider walk cannot silently collapse.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 4: point the templates tests at the world players read

**Files:**
- Modify: `internal/templates/process_test.go:27`
- Modify: `internal/templates/process_test.go` (the `TestProcess_CharacterSkillsTemplate` fixture, around `:170`)
- Modify: `internal/templates/u8_help_test.go` (around `:360`)

`_datafiles/world/default` itself is NOT touched. The owner ruled on
2026-09-17 that the vestigial default world is left alone.

- [ ] **Step 1: Capture the baseline**

Run: `go test ./internal/templates/`

Expected: `ok`. This is the green-against-the-wrong-world baseline.

- [ ] **Step 2: Flip the const**

In `internal/templates/process_test.go:26-27`:

```go
// dataFilesRoot is the relative path from this package dir to the LIVE world's
// data files. It was `world/default` until 2026-09-22, which meant this whole
// test binary asserted against templates no player ever reads: `help/attack`
// resolved to default's attack.md stub, printing raw formulas at the player,
// while dogmud's real attack.template had zero coverage. Tests that genuinely
// want the default tree register it explicitly.
const dataFilesRoot = `../../_datafiles/world/dogmud`
```

- [ ] **Step 3: Run and confirm exactly two failures**

Run: `go test ./internal/templates/ 2>&1 | tail -30`

Expected: FAIL, exactly two tests:
- `TestProcess_CharacterSkillsTemplate`, with `template: character/skills:4:7: executing "character/skills" at <index $blurbs $skillName>: error calling index: index of untyped nil`
- `TestU8CrossReferenceValidationRejectsMissingDOGMudOnlyTopic`, with `mutation control requires the registered default-world shoot template`

If any OTHER test fails, stop and report: the probe measured these two and only
these two.

- [ ] **Step 4: Give the skills test real fixture data**

Dogmud's template opens with
`{{ $cooldowns := .SkillCooldowns }}{{ $blurbs := .SkillBlurbs }}` and then
does `{{ if index $blurbs $skillName }}`. The fixture supplies `SkillList` and
`SkillCooldowns` but no `SkillBlurbs`, so `$blurbs` is a nil `any` and `index`
errors. Add the missing key to the `data` map in
`TestProcess_CharacterSkillsTemplate`, right after `"SkillCooldowns"`:

```go
		"SkillCooldowns": map[string]int{
			"melee":   0,
			"ranged":  0,
			"stealth": 0,
		},
		// The live template indexes SkillBlurbs per skill. Default's stub did
		// not, which is why this fixture never had the key and why the test
		// never exercised the template a player is actually served.
		"SkillBlurbs": map[string]string{
			"melee":   "Striking with a held weapon.",
			"ranged":  "Loosing a shot at a distance.",
			"stealth": "Moving without being noticed.",
		},
		"TrainingPoints": 3,
	}
```

Then add an assertion that only the real template can satisfy, below the
existing ones:

```go
	assert.Contains(t, result, "Striking with a held weapon.",
		"the live template renders a skill blurb; default's stub has no blurb line at all")
```

- [ ] **Step 5: Make the U8 test register the default world explicitly**

`TestU8CrossReferenceValidationRejectsMissingDOGMudOnlyTopic` opens by scanning
the package-level `fileSystems` for `templates/help/shoot.template`, which only
the default world ships (dogmud renamed it to `fire`). It inherited that
registration from `TestMain`, which no longer provides it.

Register the default tree for this one test, immediately before the existing
`registeredDefaultShoot := false` line, and restore afterwards:

```go
func TestU8CrossReferenceValidationRejectsMissingDOGMudOnlyTopic(t *testing.T) {
	// This test needs the DEFAULT world's shoot.template, which only that tree
	// ships: dogmud renamed the command to `fire`. TestMain points at dogmud
	// (the live world every other test should assert against), so this one
	// registers default explicitly rather than depending on a global that is
	// wrong for everyone else.
	saved := fileSystems
	RegisterFS(os.DirFS(`../../_datafiles/world/default`).(fs.ReadFileFS))
	t.Cleanup(func() { fileSystems = saved })

	registeredDefaultShoot := false
```

`os` and `io/fs` are already imported by this package's tests; confirm with
`head -20 internal/templates/u8_help_test.go` and add either import if the
file itself lacks it.

- [ ] **Step 6: Run the package**

Run: `go test ./internal/templates/ -v 2>&1 | tail -20`

Expected: `ok`, all tests passing against the live world.

- [ ] **Step 7: Prove the coverage actually moved**

Run: `go test ./internal/templates/ -run TestProcess_StaticTemplates -v`

Expected: PASS, and it is now rendering dogmud's `help/attack.template`. Confirm
by temporarily asserting on a phrase that exists only in the dogmud template
and not in default's stub, watching it pass, then removing the temporary
assertion. A flip that changed no coverage would be a silent no-op.

- [ ] **Step 8: Commit**

```bash
git add internal/templates/process_test.go internal/templates/u8_help_test.go
git commit -F - <<'EOF'
test(templates): assert against the world players actually read

The package's TestMain pointed the whole test binary at
_datafiles/world/default, so every test that did not override it asserted
against templates nobody is served. `help/attack` is the clearest case:
default ships both attack.md and attack.template, the lookup tries .md
first, and that stub prints raw formulas at the player. Dogmud's real
attack.template, a rewritten file with no raw numbers, had zero coverage.

Flipping the const broke exactly two tests, and both breaks were the
defect confessing. The skills test's fixture was written for default's
simpler stub and never supplied the blurbs map the real template indexes.
The U8 cross-reference test genuinely wants the default tree, so it now
registers it explicitly instead of inheriting it.

_datafiles/world/default is untouched, per the 2026-09-17 ruling.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 5: retire the last raw `CategorySubmission` sender

**Files:**
- Modify: `internal/hooks/item_procs.go:272-276`
- Modify: `send_trio_only_guard_test.go` (`sendTrioOnlyCategories`)

- [ ] **Step 1: Confirm it is still the last one**

Run: `grep -rn "CategorySubmission" --include=*.go internal/ | grep -v _test.go`

Expected: the `item_procs.go` send, plus `sendSubmissionTriple`'s SendTrio use
and the enum declaration. If a second raw sender has appeared, stop and report.

- [ ] **Step 2: Replace the raw send with the observer-only Trio**

In `internal/hooks/item_procs.go`, replace:

```go
	// Room-wide narration, no raw numbers (project rule). CategorySubmission
	// matches condition 84's own submission-stagger flavor.
	room.SendTextVisual(messaging.CategorySubmission,
		`<ansi fg="yellow">A jarring shockwave ripples outward, staggering the hostile creatures nearby!</ansi>`)
```

with:

```go
	// Room-wide narration, no raw numbers (project rule). CategorySubmission
	// matches condition 84's own submission-stagger flavor.
	//
	// Observer-only SendTrio, not a raw SendTextVisual. The line names nobody,
	// so this is not a leak fix: it is what lets CategorySubmission join
	// sendTrioOnlyCategories, since this was the category's last raw sender.
	// Both names are NoName because there is no actor and no actee to hide.
	messaging.SendTrio(messaging.Trio{
		Observer: messaging.Say(messaging.CategorySubmission,
			`<ansi fg="yellow">A jarring shockwave ripples outward, staggering the hostile creatures nearby!</ansi>`),
	}, messaging.Audience{
		ActorName: messaging.NoName,
		ActeeName: messaging.NoName,
		Room:      room,
	})
```

- [ ] **Step 3: Run the hooks package**

Run: `go test ./internal/hooks/`

Expected: `ok`. The line's text is unchanged, so any test asserting on it still
passes; only the delivery path moved.

- [ ] **Step 4: Widen the guard**

In `send_trio_only_guard_test.go`:

```go
var sendTrioOnlyCategories = []string{"Kick", "Trip", "Bash", "Submission"}
```

- [ ] **Step 5: Run the guard**

Run: `go test . -run TestNarrationTrioOnlyCategoriesLeaveOnlyThroughSendTrio -v`

Expected: PASS. If it fails, it will name the file still referencing
`messaging.CategorySubmission` outside a producer shape, which is the guard
doing its job: fix that site or record it in `sendTrioOnlyAllowed` with the
reason and the PR that closes it.

- [ ] **Step 6: SABOTAGE. Prove the widened guard sees the new category**

Temporarily reintroduce a raw send in `internal/hooks/item_procs.go`:

```go
	room.SendTextVisual(messaging.CategorySubmission, `sabotage`)
```

Run: `go test . -run TestNarrationTrioOnlyCategoriesLeaveOnlyThroughSendTrio -v`

Expected: FAIL, naming `internal/hooks/item_procs.go` and the line number.
Remove the sabotage line, rerun, expect PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/hooks/item_procs.go send_trio_only_guard_test.go
git commit -F - <<'EOF'
refactor(submission): the last raw CategorySubmission sender joins SendTrio

item_procs.go's stagger shockwave was the only remaining raw sender of
CategorySubmission once sendSubmissionTriple moved to SendTrio in
633576fc6. The line names nobody, so nothing a player reads was ever
wrong; what the raw send blocked was the guard.

With it migrated, sendTrioOnlyCategories widens from three categories to
four. One line of production code for a quarter more guard coverage is the
best ratio available anywhere in the arc right now, which is why M4d's
design filed it.

Proven by sabotage that the guard sees the new category before trusting
the green run.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 6: correct `context.md`'s stage count

**Files:**
- Modify: `internal/messaging/context.md:259`

**Scope note:** this task fixes ONLY the six-versus-seven stage inconsistency.
`context.md` also documents the wrap stage as live, which it is not. **That
claim is NOT corrected here.** It is corrected in M5 PR 2 Task 2c, so the
document and the behavior change on the same day. Correcting it now would
leave the repo briefly documenting a wrap policy that does not yet exist.

- [ ] **Step 1: Confirm the inconsistency**

Run: `sed -n '8,23p;259p' internal/messaging/context.md`

Expected: the numbered list shows 7 stages including `3. **Sight gate**`, while
the file table row for `pipeline.go` lists 6 and omits it.

- [ ] **Step 2: Fix the file table row**

Replace line 259:

```
| `pipeline.go` | Stage ordering: compose, normalize, sight gate, anonymize, color, wrap, deliver |
```

This adds the missing sight gate so the row agrees with the numbered list
above it, and drops the em dash in line with the project's prose rule.

- [ ] **Step 3: Verify no symbol claim changed**

Run: `python tools/context_md_audit.py internal/messaging`

Expected: `packages checked: 1, packages with phantom symbols: 0, total phantom
symbols: 0, All documented symbols resolve.` This task edits prose only, so the
audit's answer must be unchanged.

- [ ] **Step 4: Commit**

```bash
git add internal/messaging/context.md
git commit -F - <<'EOF'
docs(messaging): the file table lists the sight gate it was missing

context.md's numbered pipeline list has seven stages; its file table row
for pipeline.go listed six and silently dropped the sight gate, which is
the stage that decides whether a visual line is delivered at all.

The file's other wrap-stage claim is deliberately left alone here. It is
corrected in M5 PR 2 alongside the behavior change that makes it true,
so the document and the code stop disagreeing on the same day rather than
trading which one is wrong.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Proof obligations

Before the PR opens, all of the following must be true and stated in the PR
body with the evidence:

- [ ] **Full suite green.** `go test ./...` from the repo root, every package.
- [ ] **Root gate green.** `go test .` for the root-package guards, which is
      where both widened guards live.
- [ ] **The Task 3 sabotage is done and reverted**, with the red output
      recorded in the commit message or the PR body. A widened guard that was
      never proven capable of failing on its new surface proves nothing about
      that surface.
- [ ] **The Task 5 sabotage is done and reverted**, same reason.
- [ ] **The Task 4 coverage move is proven**, not assumed: a temporary
      assertion on dogmud-only template text passed before being removed.
- [ ] **Boot check** in an isolated detached worktree. 🪤 A failed boot EXITS 0:
      `main()` recovers, logs `PANIC` and a stack, and returns normally. Grep
      the log for `Server Ready` and for a real panic line; never check `$?`.
      🪤 Two `PANIC` hits in a boot log are the documented false positive:
      `GamePlay.MapConsistencyEnforce` has the literal value `panic`.
- [ ] **`git status --short` clean of `_datafiles`**, proving no sabotage
      content survived.
- [ ] **No playtest is required for this PR.** Every change is invisible to a
      sighted player: one bare name stops leaking to shapes-only readers, one
      mob name starts being hidden, and the rest is tests and docs. PR 2 and
      PR 3 carry the playtests.
