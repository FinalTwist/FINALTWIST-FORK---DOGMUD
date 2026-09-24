# Messaging M4d PR 3: The Remaining Paths

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The five narration paths that still name a party the reader cannot see
deliver through `SendTrio`, and the one-path guard widens to cover what they
close.

**Architecture:** Each of these was deferred with evidence, not overlooked. PR 1
moved every path whose lines named nobody but the reader and stopped at the ones
that did, because PR 1's contract was no text movement. PR 2 owned combat. This
PR finishes the list. Every change here has the same shape: a line that names a
second party stops naming them when the reader cannot see.

**Player-visible.** That is the point: these paths leak names in the dark today.

**Spec:** `docs/superpowers/specs/2026-09-20-messaging-m4d-send-path-design.md`,
the "PR 3, scoped 2026-09-20" section.

**NOT in this PR, by owner ruling:** the `canSeeInDark` migration. It rides with
M4e's first PR, which opens the same twenty files.

**Tech Stack:** Go, `internal/combat`, `internal/questengine`, `internal/hooks`,
`internal/usercommands`, `internal/messaging`, the repo-root AST guards.

---

## Facts verified against source (master `93d5795e9`, 2026-09-20)

| Fact | Value | Source |
|---|---|---|
| Ranged second room | `toDefenderRoomMsg` built from `roles.ActeeObserver`, sent by `SendToTargetRoom` | `internal/combat/combat.go:316,340-341` |
| Why it leaks | `SendToTargetRoom` sight-gates but never calls `HideNames`; it relies on tag-based `Anonymize`, whose docstring records that bare untagged names leak | `attackresult.go` -> `combat_verbosity.go` -> `rooms.go` |
| Its seat exists and is unused | `Trio.RemoteObserver`, `Audience.RemoteRoom`, added by PR 1 | `internal/messaging/trio.go:42` |
| Paired guard | `TestRemoteRoomIsPairedWithRemoteObserver` already enforces that setting `RemoteRoom` means naming `RemoteObserver` | `messaging_surface_guard_test.go` |
| Quest trigger narration | `(*GameBridge).Narrate(v narration.Variants)` | `internal/questengine/bridge.go:215` |
| Why it leaks | every trigger observer line carries `{actor}` by validation (`quests.RoomTextProblems`), and it delivers on `room.SendTextVisual` | its own docstring |
| Self-cast spell branches | the four that pair a caster line with a room line naming the caster: purge, heal, condition, shield | named in the comment at `spell_resolution.go:1217-1225` |
| Their room sender | `sendVisualRoomText` -> `room.SendTextVisual` | `NewRound_DoCombat_helpers.go:402-407` |
| The fifth branch | already on `SendTrio` (PR 1); it has no room line at all | `spell_resolution.go:1226-1236` |
| position_control senders | `fireStaminaWarningIfLow` -> `sendCharacterMsg`; `sendSubmissionTriple` | `internal/hooks/Position_Messaging.go:156,228,293` |
| Why they leak | the stamina room line names the warned character; the submission triples name the other grappler in the PERSONAL lines, not only the observer line | PR 1 Task 6 report, confirmed in `position_control.yaml` |
| Crafting instant narration | `case result.ImmediateComplete` and `completeCraft` | `internal/usercommands/craft.go:134,656` |
| Why it leaks | its observer line names the crafter via `{actor}` | `crafting/narration_test.go` |
| Guard's current scope | `sendTrioOnlyCategories = []string{"Kick", "Trip", "Bash"}`, allowlist EMPTY | `send_trio_only_guard_test.go` |
| 🪤 `SightDecision` ordering | best to worst: `SightFull` = 0, `SightNone` = 2. **Never compare with `>=`** | `internal/messaging/pipeline.go:34-38` |
| 🪤 Root guard | `condition_apply_path_guard_test.go` keys on `spell_resolution.go`; this PR edits that file, so expect a re-key | repo root |

---

## The proof strategy, and why it is not a golden

Each path here leaks a name to a reader who cannot see. So the natural proof is
not a recorded golden but a **red-first test per path**: a blind reader receives
the line, and the other party's name is absent. That test fails today, which is
the leak, and passes after the migration.

That is stronger than a golden for this shape of change, because a golden would
freeze the leak and then be re-recorded, which proves only that something moved.

**Every task below writes its test first and reports the RED output.**

---

## Task 1: The ranged second room

**Files:**
- Modify: `internal/combat/combat.go`
- Create: a test

- [ ] **Step 1: Prove the leak**

Write a test in which a ranged attack's defender-room line contains a BARE,
untagged name and a shapes-only observer stands in the defender's room. Assert
the name is absent from what that observer reads.

It should FAIL today: `SendToTargetRoom` runs tag-based `Anonymize`, which by its
own docstring does not touch bare names. Report the actual red output.

⚠️ If it unexpectedly PASSES, stop and report. That would mean the path is
already safe by some route this plan did not find, and the task is moot.

- [ ] **Step 2: Seat it**

Route the line through the `RemoteObserver` seat and `Audience.RemoteRoom`,
which PR 1 added and left unused. The paired guard
(`TestRemoteRoomIsPairedWithRemoteObserver`) will then start enforcing this call
site; confirm it still passes.

- [ ] **Step 3: Gates and commit**

```bash
go build ./...
go test ./internal/combat/ ./internal/messaging/ ./internal/narration/
go test .
git diff --stat internal/narration/testdata
gofmt -l $(git diff --name-only master..HEAD | grep '\.go$')
```

The goldens render with stand-ins rather than real names, so none should move.
If one does, that is a finding: report it.

---

## Task 2: Quest trigger narration

**Files:**
- Modify: `internal/questengine/bridge.go`
- Create: a test

- [ ] **Step 1: Prove the leak**

`Narrate` delivers a trigger's observer line, which by validation always
contains `{actor}`. Write a test: a blind observer in the room does not read the
triggering player's name. Report the RED output.

- [ ] **Step 2: Migrate to `SendTrio`**

The actor is the triggering player; there is no actee. Set `ActeeName` to
`messaging.NoName` so nothing spurious is hidden.

⚠️ Quest text is authored content with its own validators
(`quests.RoomTextProblems`, `Quest.validateRoomText`). Run the quest tests and
the shipped-data guards, not just the package's own.

- [ ] **Step 3: Gates and commit**, as Task 1.

---

## Task 3: The four self-cast spell branches

**Files:**
- Modify: `internal/hooks/spell_resolution.go`
- Create: a test

PR 1 left a comment at `spell_resolution.go:1217-1225` naming these four
precisely: purge, heal, condition and shield each pair a safe caster line with a
room line that names the caster through `sendVisualRoomText`. **Read that comment
first; it is the map for this task.**

- [ ] **Step 1: Prove the leak** for one of the four, with a blind observer in
the room. Report the RED output.

- [ ] **Step 2: Migrate all four**, one commit for the set, since they share a
shape and splitting them would make the diff harder to read rather than easier.

⚠️ Delete or rewrite PR 1's comment in the same commit. It describes these four
as deliberately left behind, and that stops being true here. A stale comment
explaining why something was NOT done, sitting above code that now does it, is
worse than no comment.

- [ ] **Step 3: The root guard will likely re-key.**
`condition_apply_path_guard_test.go` is a line-number allowlist over
`spell_resolution.go`, and this task moves lines in it. Run `go test .` at the
root, re-key if it reddens, and say so in the commit body.

- [ ] **Step 4: Gates and commit.**

---

## Task 4: position_control

**Files:**
- Modify: `internal/hooks/Position_Messaging.go`
- Create: a test

This is the richest of the four and the one most likely to surprise you.

- [ ] **Step 1: Read both senders before changing either.**
`fireStaminaWarningIfLow` -> `sendCharacterMsg` pairs a safe self line with a
room line naming the warned character. `sendSubmissionTriple` is different: per
PR 1's finding, its submission templates name the other grappler in the
**personal** lines, not only the observer line. Confirm that against
`_datafiles/world/dogmud/messaging/position_control.yaml` and report what you
find.

🪤 M4a recorded that this store's stamina warning is addressed to the CHARACTER
while the controller is a different person: the one asymmetric mapping in the
store. Whoever the warning is about is the `{actor}` of its room line. Get that
mapping right or the wrong person's name gets hidden.

- [ ] **Step 2: Prove the leak** for both senders separately. Two red tests.

- [ ] **Step 3: Migrate**, keeping the asymmetric mapping intact.

- [ ] **Step 4: Gates and commit.**

---

## Task 5: Crafting's instant-craft narration

**Files:**
- Modify: `internal/usercommands/craft.go`
- Create: a test

PR 1 moved crafting's command responses and explicitly left this one: the
`ImmediateComplete` case at `:134` and `completeCraft` at `:656`, whose observer
line names the crafter.

- [ ] **Step 1: Prove the leak.** Report RED.
- [ ] **Step 2: Migrate.** Note that PR 1's `craftDeliver` helper leaves `Room`
unset, which is why its text-identity was structural. This path DOES have a room
line, so it needs a real `Audience` with a room, not `craftDeliver`.
- [ ] **Step 3: Gates and commit.**

---

## Task 6: Widen the one-path guard

**Files:**
- Modify: `send_trio_only_guard_test.go`

The guard covers `Kick`, `Trip` and `Bash`, 3 of 60 categories, with an empty
allowlist. Every path this PR migrated may have made its categories fully
`SendTrio`-only.

- [ ] **Step 1: Re-run the survey** the guard's author did: for each category,
grep every raw `Send*`-family call carrying it and check whether any production
sender remains outside `SendTrio`. **Report the new count out of 60.**

- [ ] **Step 2: Add every newly-clean category** to `sendTrioOnlyCategories`.

⚠️ Do NOT add a category that still has a bypass just because this PR touched
its area. The guard's value is that it states something true. If the honest
answer is that only one or two more categories qualify, that is the answer;
report it and move on.

- [ ] **Step 3: Prove it still fails.** Add a raw `SendText` with a
newly-guarded category to a non-allowlisted file, confirm the guard names it,
revert, prove the revert.

- [ ] **Step 4: Commit.**

---

## Task 7: Docs, gates, targeted verification, PR

- [ ] **Step 1: context.md** for every package touched:
`internal/combat`, `internal/questengine`, `internal/hooks`,
`internal/usercommands`. Run `python tools/context_md_audit.py` and report.

- [ ] **Step 2: Patch note** in `docs/PATCH_NOTES.md`, newest entry FIRST (a
single running file; there is no patchnotes directory). One short entry: in the
dark you no longer read the names of people you cannot see, in a few remaining
places where you still did. Player-copy rules: 80 columns, ESL-clear, no raw
numbers, no em dashes.

- [ ] **Step 3: Full gate**, each standalone:

```bash
go build ./...
go test . ./...
golangci-lint run --new-from-rev=master
gofmt -l $(git diff --name-only master..HEAD | grep '\.go$')
```

- [ ] **Step 4: Boot check.** Build, run with the repo as working directory,
wait about 45 seconds, then grep the log STANDALONE for `Server Ready` and for
`PANIC`. 🪤 A failed boot exits 0. Kill by the PID you started; the owner runs
their own server on this machine.

- [ ] **Step 5: Targeted verification instead of a full playtest.**

**This is a judgement call, stated so it can be overridden.** M4c deferred its
playtest and PR 2 required one, so the precedent is not automatic. PR 3 carries
no balance change, no colour change, and no new prose: each change makes one
name disappear for a reader who cannot see, and each is proven by a red-first
test. A full three-seat harness run would mostly re-prove PR 2's ground.

So: one targeted check rather than a full gate, on **grappling in the dark**,
because `sendSubmissionTriple` is the only path here whose PERSONAL lines change
and grappling is interactive enough that a wrong name mapping would read badly.
Use the harness with an unlit room, quote the submission lines verbatim for both
grapplers, and confirm the right person is hidden from the right reader.

If the owner wants the full gate instead, this step expands; nothing else in the
plan changes.

- [ ] **Step 6: PR.**

```bash
git push -u origin feature/messaging-m4d3-remaining-paths
gh pr create --repo pruuk/DOGMud --base master --title "M4d PR 3: the remaining paths" --body-file <body>
```

🪤 Every `gh` command carries `--repo pruuk/DOGMud`; this repo is a fork.

The body states: which five paths moved and what each stopped leaking; the
red-first test per path; the guard's new category count out of 60; that the
`canSeeInDark` migration is deliberately NOT here and rides with M4e; and the
verification choice in Step 5 with its reasoning.

---

## Self-review

**Spec coverage.** The spec's PR 3 section lists three items: the ranged second
room (Task 1), the four non-combat paths (Tasks 2 to 5), and widening the guard
(Task 6). The `canSeeInDark` exclusion is stated in the header and repeated in
the PR body. Covered.

**Placeholders.** None. Each task's test is described by what it must assert
rather than given as code, deliberately: these are five different packages with
five different fixture shapes, and inventing five fixtures in a plan without
reading each package's existing tests is how a plan produces code that compiles
against nothing. Each task says to read the neighbouring tests first.

**Ordering.** Task 6 must come last, because its survey depends on what Tasks 1
to 5 actually migrated. Tasks 1 to 5 are independent of each other and may be
done in any order; each is its own commit so any one can be dropped without
unpicking the rest.

**Scope.** The risk here is Task 4. If `position_control`'s asymmetric mapping
turns out to be more tangled than PR 1's note suggests, it can be dropped to its
own PR without affecting the other four, and the guard survey in Task 6 simply
reports one fewer clean category.
