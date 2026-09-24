# Messaging M4d PR 2: Combat On The Path

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Dark combat stops being narrated by twelve hardcoded sentences and
starts rendering from the store with identities hidden, a per-round notice
explains why, seeing shapes becomes worth something in a fight, and GMCP stops
handing the answer back.

**Architecture:** PR 1 made the sight verdict the primitive and put it on
`combatContext` per side. So this PR hides names **at composition inside
`internal/combat`**, where both the verdict and both names are already in hand,
and the verbosity drain stays dumb. `replaceDarknessMessages` and its twelve Go
literals are deleted. `DarknessScoreMultiplier` replaces a boolean test so
`SightShapes` can take its own reduced penalty. `Char.Enemies` mirrors the
status prompt.

**PLAYER-VISIBLE. This PR carries the playtest gate deferred from M4c.**

**Spec:** `docs/superpowers/specs/2026-09-20-messaging-m4d-send-path-design.md`
(read its "What PR 1 changed about PR 2" section first).

**Tech Stack:** Go, `internal/combat`, `internal/hooks`, `internal/messaging`,
`internal/configs`, `modules/gmcp`, the playtest harness under `tools/playtest`.

---

## Facts verified against source (master `904a45ac0`, 2026-09-20, after #148)

| Fact | Value | Source |
|---|---|---|
| Per-side verdict on the context | `combatContext.sourceSight` / `targetSight`, `messaging.SightDecision` | `internal/combat/combat_helpers.go:36-42` |
| Scoring sites | two, both `*= float64(bal.DarknessCombatPenalty)` | `combat_helpers.go:566,761` |
| Today's rule | full penalty for ANY verdict short of `SightFull`, infrared included | same |
| Shipped penalty | `DarknessCombatPenalty: 0.80`, a flat 20% to hit and defend | `_datafiles/config.yaml`; `config.balance.go:301` |
| Its validation | `<= 0 || > 1.0` reverts to `0.80` | `config.balance.combat.go:335-336` |
| Darkness substitution | `replaceDarknessMessages(result, sourceCanSee, targetCanSee)` | `internal/hooks/NewRound_DoCombat_helpers.go:438` |
| Its twelve literals | six attacker-side (`:450`), six defender-side (`:478`) | same file |
| Its only caller | gated on `CanSeeSightImpairedOnly` per side | `NewRound_DoCombat_unified.go:549-558` |
| Participant drain | `drainParticipantLines`, bare `u.SendText`, verbosity only | `internal/hooks/combat_verbosity.go:304-315` |
| Spectator drain | `room.SendTextVisualToUser` | `combat_verbosity.go:349` |
| Per-round seam | `flushCombatTallies()`, "called once at the end of DoCombat each round", already iterates viewers | `combat_verbosity.go:396-411` |
| Name hiding wording | `something` at `SightNone`, `a figure` at `SightShapes` | `internal/messaging/hidenames.go:42-45` |
| GMCP enemies payload | `Name`, `DisplayHealth()`, `MaxHp`, **no sight check** | `modules/gmcp/gmcp.Char.go:431-459` |
| Prompt's rule to mirror | `canSeeInRoomFn`, wired to `messaging.CanSeeClearly(c, room)` | `internal/users/userrecord.prompt.go:51-63`; `main.go:327-336` |
| Prompt's behaviour | `{target}` prints `an unseen foe`; `{targethealth}` and `{targetpos}` suppress | `userrecord.prompt.go:530-576` |
| Ranged second room | `toDefenderRoomMsg` sent via `SendToTargetRoom`; seat exists but unused | `internal/combat/combat.go:316,341`; `internal/messaging/trio.go:42` |
| 🪤 `SendToTargetRoom` | sight-gates, but never calls `HideNames`; relies on tag-based `Anonymize` | `internal/combat/attackresult.go:209` -> `combat_verbosity.go:332` -> `rooms.go:463` |
| One-path guard | 3 guarded categories, allowlist EMPTY | `send_trio_only_guard_test.go` |
| 🪤 `SightDecision` ordering | best to worst: `SightFull` = 0, `SightNone` = 2. **Never compare with `>=`** | `internal/messaging/pipeline.go:34-38` |
| 🪤 Root guard | `condition_apply_path_guard_test.go` keys on `spell_resolution.go` and `combat_drain.go`; it re-keyed once in PR 1 | repo root |

---

## Task 1: Record the twelve literals before deleting them

**Why first:** the arc's standing rule is that a slice adds its deferred-text
rows in the same commit that retires the text. Doing it first means the ledger
cannot be forgotten once the code is gone.

**Files:**
- Modify: `docs/superpowers/audits/messaging-m6-content-ledger.md`

- [ ] **Step 1: Extract them verbatim**

```bash
sed -n '438,510p' internal/hooks/NewRound_DoCombat_helpers.go | grep -o 'Text: `[^`]*`'
```

Expected: twelve lines. Count them and report the number; if it is not twelve,
the plan's census is wrong and you should say so rather than proceeding.

- [ ] **Step 2: Add the rows**

Read the ledger's existing row format first and match it exactly. Each row
records: the text, that it was combat's dark-room substitute, that M4d PR 2
retired it, and what replaced it (the store line with the identity hidden). The
point of the row is that an author in M6 can decide whether dark combat deserves
purpose-written prose again, so preserve the flavour distinction: these lines
named the CONDITION ("in the darkness", "blindly", "You hear"), which the
anonymised store line will not.

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/audits/messaging-m6-content-ledger.md
git commit -m "docs(ledger): the twelve dark-combat literals M4d PR 2 retires

Recorded before deletion, per the arc rule that a slice files its deferred text
in the same breath as retiring it. They named the condition explicitly, which
the anonymised store line will not, and that is the M6 question.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Record the dark-versus-lit matrix BEFORE the change

**Why:** this is the M4c pattern that worked. A golden recorded through the
production path, before the flip, so the diff IS the review.

**Files:**
- Create: `internal/hooks/darkness_narration_golden_test.go`
- Create: `internal/hooks/testdata/darkness_narration.golden` (generated)

- [ ] **Step 1: Write the golden**

The grid crosses, at minimum:
- attacker sight: `SightFull`, `SightShapes`, `SightNone`
- defender sight: the same three
- outcome: a clean hit, a defended swing, a defensive crit, a miss, a fumble

For each cell, record what the ATTACKER reads, what the DEFENDER reads, and what
a spectator reads. Use a label fixture the way
`internal/combat/melee_defence_band_golden_test.go` does (one variant per pool
naming itself) so the golden records WHICH LINE was selected rather than
authored prose, and production randomness cannot move it.

⚠️ Drive the real path. The value of this golden is that it goes through
whatever `NewRound_DoCombat_unified.go` and the verbosity drain actually do,
including `replaceDarknessMessages`. If that requires standing up more fixture
than `internal/hooks` tests usually do, look at how the existing
`NewRound_DoCombat_*_test.go` files build their world and copy it.

- [ ] **Step 2: Record it and census the rows**

```bash
go test ./internal/hooks/ -run TestDarknessNarrationGolden -update-darkness -v
grep -v "^#" internal/hooks/testdata/darkness_narration.golden | grep -c "=>"
```

Report the row count. Then read the file and confirm it shows today's
behaviour: any cell where a participant's sight is not `SightFull` should show
one of the twelve hardcoded sentences rather than a pool line.

- [ ] **Step 3: Prove it can fail**

Temporarily change one of the twelve literals in `replaceDarknessMessages`. The
golden must redden and name that cell. Revert, confirm
`git diff internal/hooks/NewRound_DoCombat_helpers.go` prints nothing, rerun
green. Report the actual failure output.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/darkness_narration_golden_test.go internal/hooks/testdata/darkness_narration.golden
git commit -m "test(hooks): freeze dark-versus-lit combat narration before M4d PR 2

Recorded through the production path, so the flip's diff is the review.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Hide identities at composition, delete the substitution

**This is the task.** Everything else in this PR is smaller.

**Files:**
- Modify: `internal/combat/combat_helpers.go` (composition)
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go` (delete `replaceDarknessMessages`)
- Modify: `internal/hooks/NewRound_DoCombat_unified.go` (delete its call and the `srcCanSee`/`tgtCanSee` plumbing)
- Modify: `internal/hooks/testdata/darkness_narration.golden` (regenerated ONCE)

- [ ] **Step 1: Hide at composition**

`internal/combat` already has, per side, the verdict (`ctx.sourceSight`,
`ctx.targetSight`) and both names. Apply `messaging.HideNames` to each
participant's PERSONAL line as it is composed, hiding the OTHER party's name by
THAT reader's verdict:

- the attacker's line hides the defender's name by `ctx.sourceSight`
- the defender's line hides the attacker's name by `ctx.targetSight`

Room and remote-room lines are already sight-judged by their senders; do not
double-hide them here.

⚠️ **Do not hide the reader's OWN name.** `SendTrio` hides only the other
party, and a line reading "Something swings at something" is a bug, not
atmosphere. The golden's `SightNone`-versus-`SightNone` cell is where that shows
up; read it specifically.

⚠️ Apply this to the composed pool lines, NOT to the deflected-swing composite
or the opening-strike lines if those are built elsewhere; find every place a
participant's personal line is produced and report the list. `grep -n
"SendToSource\|SendToTarget" internal/combat/*.go` is a starting point, not the
answer.

- [ ] **Step 2: Delete `replaceDarknessMessages` and its call**

Remove the function (`NewRound_DoCombat_helpers.go:438` through its end) and its
call site, plus the now-unused `srcCanSee` / `tgtCanSee` locals at
`NewRound_DoCombat_unified.go:549-558`.

⚠️ Read `:586-600` before deleting: the tally recording is gated on `srcCanSee`
so that a blind attacker's tally does not reintroduce names their prose hid.
That gate must survive in some form, now expressed against the verdict. If you
delete it, a blind attacker gets a named summary of a fight they cannot see,
which is precisely the leak this PR exists to close. Report how you preserved it.

- [ ] **Step 3: Regenerate the golden ONCE and review the diff**

```bash
go test ./internal/hooks/ -run TestDarknessNarrationGolden -update-darkness
git diff internal/hooks/testdata/darkness_narration.golden
```

Check the diff against these expectations and state each explicitly:
- every `SightFull` / `SightFull` cell is UNCHANGED
- every cell where a participant cannot see now shows a pool line with the other
  party's identity replaced by `something` or `a figure`, not one of the twelve
- no cell shows the reader's own name replaced
- the spectator column is unchanged (its sender already judged sight)

If any expectation fails, STOP and report rather than adjusting the golden.

- [ ] **Step 4: Gates**

```bash
go build ./...
go test ./internal/combat/ ./internal/hooks/ ./internal/messaging/ ./internal/narration/
go test .
gofmt -l $(git diff --name-only master..HEAD | grep '\.go$')
```

🪤 `go test .` at the root is not optional: deleting a function shifts lines in
`NewRound_DoCombat_*.go`, and the root allowlist guard already re-keyed once in
PR 1. The gofmt line must print nothing; run it standalone.

- [ ] **Step 5: Commit**

```
feat(combat)!: dark fights render from the store with identities hidden

replaceDarknessMessages threw the composed line away and substituted one of
twelve hardcoded sentences, so a dark fight lost weapon and species flavour,
the defence that won, and M4c's bands: a defensive crit and a barely-scraped
parry both read "Your attack is turned aside by something!". Names are now
hidden at composition instead, by each reader's own sight verdict, which
internal/combat already carries per side since PR 1.

PLAYER-VISIBLE. The twelve literals are retired; their text is in the M6
content ledger.
```

---

## Task 4: The per-round blind notice

**Files:**
- Modify: `internal/hooks/combat_verbosity.go`
- Modify: `internal/messaging` category list (a new category)
- Create: a test

- [ ] **Step 1: Draft the copy**

Load the `dogmud-player-copy` skill's rules: 80 column hard wrap, ESL-clear, no
raw numbers, no em dashes. One short line. It must say that the player cannot
see and that it is costing them, without numbers. Put two or three candidates in
your report and pick one, saying why.

- [ ] **Step 2: Send it once per round per blind participant**

`flushCombatTallies` (`combat_verbosity.go:398`) already runs once per round and
iterates viewers. Send the notice from there or immediately beside it, to a
viewer whose verdict is not `SightFull`, and only when that viewer actually
fought this round.

⚠️ **Only when they fought.** A player standing in a dark room who is not in
combat must not read a combat notice every round. Say in your report how you
established "fought this round" and from what data.

Give it its own `messaging.Category` so verbosity can suppress it. It is NOT
floor-protected.

- [ ] **Step 3: Test it**

A test proving: fires once for a blind combatant in a round with several swings;
does not fire for a sighted combatant; does not fire for a blind non-combatant.
Prove it red first for at least the "fires once, not per swing" case.

- [ ] **Step 4: Gates and commit**

---

## Task 5: Infrared takes a reduced darkness penalty (flagged balance commit)

**Files:**
- Modify: `internal/configs/config.balance.go`, `internal/configs/config.balance.combat.go`
- Create: `internal/configs/config_darkness_shapes_test.go`
- Modify: `internal/combat/combat_helpers.go`
- **`_datafiles/config.yaml` is the CONTROLLER's job.** Do not touch it.

- [ ] **Step 1: The knob**

```go
	// DarknessShapesCombatPenalty is the multiplier for a combatant who makes
	// out SHAPES but not detail: infrared in an unlit room. Seeing shapes is
	// worth something in a fight and is not worth everything, so it sits
	// between DarknessCombatPenalty and no penalty at all.
	//
	// Validated as a PAIR with DarknessCombatPenalty: it must be at or above
	// the blind penalty and at or below 1.0. An inverted or out-of-range pair
	// reverts BOTH, so a typo cannot ship a world where seeing shapes is worse
	// than seeing nothing. Zero is rejected: test binaries never load
	// config.yaml.
	DarknessShapesCombatPenalty ConfigFloat `yaml:"DarknessShapesCombatPenalty"` // Multiplier when the combatant sees shapes only (default 0.90)
```

Validation, beside the existing `DarknessCombatPenalty` block:

```go
	if b.DarknessShapesCombatPenalty <= 0 || b.DarknessShapesCombatPenalty > 1.0 ||
		b.DarknessShapesCombatPenalty < b.DarknessCombatPenalty {
		b.DarknessCombatPenalty = 0.80
		b.DarknessShapesCombatPenalty = 0.90
	}
```

**0.90 is a STARTING POINT, not a considered balance value.** It is the midpoint
of the blind penalty and no penalty. Say so in the PR body; it is a config knob
and retuning costs nothing.

- [ ] **Step 2: The multiplier function**

```go
// DarknessScoreMultiplier is the sight-based multiplier on an attack or defence
// score. It is the ONE place the three verdicts turn into a number.
func DarknessScoreMultiplier(sight messaging.SightDecision, bal configs.Balance) float64 {
	switch sight {
	case messaging.SightFull:
		return 1.0
	case messaging.SightShapes:
		return float64(bal.DarknessShapesCombatPenalty)
	default:
		return float64(bal.DarknessCombatPenalty)
	}
}
```

Replace both scoring sites (`combat_helpers.go:566,761`) with it.

⚠️ Confirm `internal/combat` can take `configs.Balance` as a parameter here
without an import cycle; both sites already have `bal` in hand. If the signature
is awkward, read the knob inside the function instead and say so.

- [ ] **Step 3: Tests**

Validation tests (zero rejected, inverted pair reverts both, authored pair
survives), plus a test that an infrared combatant's score is strictly between a
blind one's and a fully-sighted one's. **That last test is the ruling**; prove
it red against the pre-change code first.

⚠️ PR 1 left a test asserting the penalty STILL applies to infrared
(`internal/combat/darkness_penalty_verdict_test.go`). That test is now WRONG by
design. Update it in this commit and say so: it was correct for PR 1's contract
and this commit is where the contract changes.

- [ ] **Step 4: Gates and commit**, message beginning `balance(combat)!:` and
stating plainly that infrared characters now land and defend more often in the
dark than before, and that the value is a starting point.

---

## Task 6: `Char.Enemies` stops leaking

**Files:**
- Modify: `modules/gmcp/gmcp.Char.go`
- Create: a test

- [ ] **Step 1: Gate it**

At `gmcp.Char.go:445-451`, when the viewing user cannot see, send `an unseen
foe` for `Name` and suppress the HP fields, mirroring
`userrecord.prompt.go:530-576`.

Use the same predicate the prompt uses: `messaging.CanSeeClearly(character,
room)`. Confirm `modules/gmcp` can import `internal/messaging`; if a boundary
forbids it, use the same injection pattern `users.SetCanSeeInRoomCheck` uses
(`main.go:327`) and say so.

⚠️ Keep the enemy ROW. Dropping the row entirely tells a scripted client the
fight ended. The row stays; its identity and health are what go.

- [ ] **Step 2: Test**

A blind viewer's payload names no mob and carries no HP; a sighted viewer's is
unchanged. Prove it red first.

- [ ] **Step 3: Gates and commit**

---

## Task 7: The ranged second room, and the deferred non-combat paths

**Scope rule: these are independent of each other and of Tasks 3 to 6. If PR 2
is already large when you reach this task, STOP and report which of these you
did not do.** They can become a PR 3 without blocking the playtest, because the
playtest's subject is combat darkness.

- [ ] **Step 1: The ranged second room**

PR 1 seated `RemoteObserver` but left `SendToTargetRoom` in place, because that
sender sight-gates yet never runs `HideNames`, relying on tag-based `Anonymize`
which leaks bare untagged names. Route the defender-room line through the
`RemoteObserver` seat so it is hidden the same way every other audience is.

Prove the leak is closed: a test with a BARE (untagged) name in a
`remote_observer` line, read by a shapes-only observer in the defender's room.
It must be hidden after, and you should confirm it leaks before.

- [ ] **Step 2: The four deferred paths, one at a time, each its own commit**

Quest trigger actions (`internal/questengine/bridge.go`), the four
`applyPlayerEffect` self-cast branches, `position_control` (both senders), and
crafting's instant-craft narration. Each moves to `SendTrio`, and each CHANGES
TEXT in the dark, which is why PR 1 deferred them. That is expected here.

For each, record in your report what a reader in the dark now sees instead.

- [ ] **Step 3: Widen the one-path guard**

Each path migrated means its categories may now be fully on `SendTrio`. Re-run
the survey the guard's author did, add every newly-migrated category to
`sendTrioOnlyCategories`, and report the new count out of 60. This is how the
guard earns its keep rather than ossifying at three.

---

## Task 8: Docs, patch note, gates

- [ ] **Step 1: context.md** for `internal/combat`, `internal/hooks`,
`internal/messaging` and `modules/gmcp` as each applies. Run
`python tools/context_md_audit.py` and report.

- [ ] **Step 2: Patch note** in `docs/PATCH_NOTES.md` (a single running file,
newest entry FIRST; there is no patchnotes directory). Player-facing, 80 column
wrap, no raw numbers, no em dashes. It must cover both things a player will
notice: fighting blind now describes the fight rather than only the darkness,
and characters who see heat fight better in the dark than those who see nothing.

- [ ] **Step 3: Sweep by meaning.** Run each standalone:

```bash
grep -rn "replaceDarknessMessages" --include=*.go --include=*.md .
grep -rni "darkness substitution\|dark-room substitute\|hardcoded dark" --include=*.md internal/ docs/
```

- [ ] **Step 4: Full gate**: `go build ./...`, `go test . ./...`,
`golangci-lint run --new-from-rev=master`, and `gofmt -l` over the branch diff,
each standalone.

- [ ] **Step 5: Boot check.** Build, start with the repo as working directory,
wait about 45 seconds, then grep the log STANDALONE for `Server Ready` and for
`PANIC`. 🪤 A failed boot exits 0. Kill by the PID you started; the owner runs
their own server on this machine.

---

## Task 9: The playtest gate

**This is the gate M4c deferred. It is not optional and it is not a formality.**

- [ ] **Step 1: Build the fixture**

A dark room, a combat that survives several rounds, and three seats: attacker,
defender, and a witness who is not fighting. Model the fixture on
`tools/playtest/profiles/counters.yaml`, which solved the same "must survive
rounds" problem.

🔑 A combat fixture must SURVIVE ROUNDS or the run comes back partial. The
counters slice learned this the hard way.

- [ ] **Step 2: Run it from BOTH bridges**

Telnet and GMCP. The GMCP seat is what proves Task 6 landed; without it the
report cannot honestly say darkness works.

- [ ] **Step 3: Quote lines verbatim** in the report, for each seat, covering:
a hit, a defended swing, a defensive crit, the per-round notice, and what the
witness saw. Include an infrared character's seat if the fixture can carry one,
since that is the only way to observe Task 5 in play rather than in a test.

- [ ] **Step 4: Answer the question this PR exists to ask.**

Does the anonymised store line read BETTER or WORSE than the twelve sentences it
replaced? The spec names the fallback plainly: if it reads worse, keep the
bespoke prose and PR 2 shrinks to the `HideNames` seam plus the `Char.Enemies`
gate. **Recommending the fallback is a successful playtest**, not a failed one.

🧹 Playtest reports are gitignored. Extract findings to memory in the same
session or they are lost.

---

## Self-review

**Spec coverage.** The spec's PR 2 has five parts: delete the substitution
(Task 3), dark lines from the store (Task 3), the per-round notice (Task 4), the
infrared penalty (Task 5), and the `Char.Enemies` gate (Task 6). Its two proofs
are the before-recorded matrix (Task 2) and the playtest (Task 9). The ledger
rule is Task 1. PR 1's deferred paths and the ranged leak are Task 7, explicitly
droppable. Covered.

**Placeholders.** Task 4's copy is drafted in the task rather than fixed here,
deliberately, because player copy is reviewed against the copy skill and
proposing final wording in a plan invites rubber-stamping. Task 2's fixture
shape is described rather than coded because the `internal/hooks` world fixture
is heavier than anything this plan can safely guess at; the task says to copy an
existing one and name which.

**Type consistency.** `DarknessScoreMultiplier` is defined in Task 5 Step 2 and
replaces both sites named in the facts table.
`DarknessShapesCombatPenalty` is declared in Step 1 and read in Step 2.
`ctx.sourceSight` / `ctx.targetSight` are PR 1's fields, used in Tasks 3 and 5.

**Scope.** Task 7 is the pressure valve. If it is dropped whole, PR 2 is still a
complete, coherent, playtestable slice, and PR 3 is small and pre-specified.
