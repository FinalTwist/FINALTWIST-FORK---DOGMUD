# M4e PR 1b: Player Special Moves to YAML

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task.
> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the twelve `internal/usercommands` special-move files' narration
into the store PR 1a built, squaring every ragged variant pool so the three
audiences of one swing describe the same moment.

**Architecture:** The store, the call-site helper pattern and the guards all
exist already (`internal/movenarration`, PR #152, master `2df46c681`). This
slice adds the player-side `actor` role to the same thirteen YAML files and
retires the last twelve entries from `m2FrozenFiles` / `m2RoutingFiles`.

**Tech Stack:** Go, `internal/movenarration`, `internal/narration`
(`Variants`/`Render`/`ValidateVariants`), `internal/messaging` (`SendTrio`).

---

## Facts verified against source, 2026-09-21, master `2df46c681`

| Fact | Value | Source |
|---|---|---|
| Files in scope | **12** `internal/usercommands` special-move files | `m2FrozenFiles`, `m2RoutingFiles` (12 entries each remain) |
| Files WITH variant pools | **7**: drain, gore, kick, maul, pounce, rake, throttle | `grep '\[\]string{' ` + `util.Rand`, read per file |
| Files WITHOUT pools | **5**: bash, trip, grapple, shoot, throw. Already 1/1/1 per branch | same grep, **zero** matches; verified, not assumed |
| Pool variables | 92 across 30 events | counted by reading |
| Naming convention | actor `<label>Msgs`, actee `<label>TargetMsgs`, observer `<label>RoomMsgs` | uniform across all 7 |
| Events needing squaring | **30** | census |
| New lines needed | **59** total: **41** parallel-squaring + **18** union-authoring | census |
| Divergent events | **6, all `miss`**: drain, gore, kick(knee), maul, pounce, rake | census |
| Store already carries | 13 verbs, 67 events, `actor` role EMPTY on every mob event | `_datafiles/world/dogmud/narration/special-moves/` |

### Corrections the census made to the earlier (PR 1a) survey

Read from source; the earlier figures were incomplete.

- **`gore` and `pounce` each have a SECOND hit pool** (the non-knockdown
  branch, 4/3/2), never previously listed.
- **`kick` has a THIRD sub-moveset**, knee (6/6/5), beyond stomp and standard.
- "every `partial` is 2/2/1" is **false**: kick's stomp and knee partials are
  1/1/1.
- "every `miss` is 3/2/1" is **false**: kick-standard's is 3/2/2.
- `kick` is not one verb with three outcomes; it is three sub-movesets
  selected by `res.Variant`, each with its own pools, and only Standard has a
  knockdown branch. It is also the only one of the twelve firing a
  `questengine` notification.

### Out of scope, verified

- **`drain`'s lifesteal lines** (`healMsgs`) send `Actee` and `Observer` as
  `messaging.NoLine` BY DESIGN: private information, per the detail-line
  ruling. A legitimately absent role, not a gap. Do not square it.
- **`throttle`'s spell-interruption triad** is a hardcoded 1/1/1 that routes
  its observer line through `CategorySpellDisruption` rather than
  `CategorySystem`. Looks like a missed pool; is not one.
- **Defended-outright branches** in all seven call the shared
  `moveDefenceLines()` helper. Not local prose.

### Owner rulings (do not relitigate)

1. **Squaring is a UNION, not a truncation and not a choice between
   readings.** Where roles disagree, each distinct feel becomes its own
   complete trio. Confirmed by the census: only 6 events actually diverge.
2. **The 18 union lines were reviewed and approved on 2026-09-21**, with one
   change: `kick`'s knee miss actor line is REPLACED (see Task 3).
3. **Em dashes in `gore` and `pounce`'s existing actor lines are FILED, not
   fixed** (M6 ledger row 55). Rewriting one would break byte-identity.
4. ESL-opaque idioms stay filed for M6 (rows 48-53).

---

## 🔴 CORRECTED 2026-09-21, after a first attempt got both of these wrong

### 1. The twins keep SEPARATE event keys (owner ruling, option B)

The original plan assumed the player side just fills in the empty `actor` role
on the existing events. **It cannot.** The twins shipped DIFFERENT prose for
the same event, and each twin's byte-identity net pins its own wording to
specific pool indices, so they collide. Measured, `drain/hit` actee role:

```
mob    (in the store):  "{actor} plunges into you, sapping your vitality and
                         leeching your life-force!"
player (still in Go):   4 unrelated lines, "latches onto you and drinks deep",
                         "tears into you and feeds", ...
```

**Player events take `player_` prefixed keys** (`player_hit`,
`player_standard_miss`, `player_knee_miss`). Mob events are left EXACTLY as
PR #152 shipped them. The M4e spec's "the twins share one event file" is still
satisfied: one file per verb, two sets of keys.

Merging the two prose sets into one pool is **option A, filed as M6 ledger
row 56**. It is the bigger content win and costs roughly 24 more authored
lines, because a mob event has no actor line to pair with its actee and
observer entries. It does not belong inside a slice whose proof is that
nothing changed.

### 2. Authoring and migration land in ONE commit per file

`TestMoveEventKeysAgree` fails on any authored event no Go call site names, so
a `player_*` event **cannot** ship before the code that references it. The
original Task 2 / Task 3 / Task 4 split is therefore impossible, and the first
attempt hit exactly this wall on `drain`'s lifesteal events.

**Tasks 2, 3 and 4 below are superseded by Task 4b**: per file, one commit that
authors that verb's squared `player_*` events AND migrates its call site, with
the net dropping by that file's row count each time.

## The shape of the work

🔑 **41 of the 59 lines are not invention.** In a PARALLEL event, index N of
each role already describes the same moment and the shorter pools were simply
truncated, so each missing line has a sibling that dictates its content.
Measured example, `maul/hit`: actor 0 is "fangs savage, wounds weep blood",
actee 0 is "savages you, tearing bleeding wounds", observer 0 is "savages with
vicious fangs".

🔴 **The remaining 18 are genuine authoring**, in the six divergent `miss`
events, and are already written and approved in Task 3.

🔑 **Squaring fixes a live defect, not just a data shape.** Each role draws its
own `util.Rand` today, so one swing is narrated to three people as three
different moments. This is the same defect M3 item 8 fixed for core combat when
it deleted `ConsistentAttackMessages`; the special moves never got that pass.

---

## Task 1: Extend the net BEFORE touching any file

The 118-row byte-identity net covers the mob files only. It must cover the
player files before a single literal moves, exactly as in PR 1a.

**Files:** Modify `tools/move_narration_net.py`,
`internal/mobcommands/testdata/pre_migration_literals.json` (or add a sibling
fixture for the player side)

- [ ] **Step 1: Extend the extractor to the twelve player files**

The player files differ from the mob files in three ways the extractor must
handle: they have an `actor` role (the mob files never did), they use POOLS
(`fmt.Sprintf(pool[util.Rand(len(pool))], ...)`) rather than single literals,
and `kick` selects a sub-moveset before choosing a pool.

Record EVERY pool entry with its index, not just index 0. A net that checks
one entry per pool cannot fail on the other five.

- [ ] **Step 2: Prove it goes RED against the unmigrated tree**

Mis-author one line in `kick.yaml`, run the net, confirm it fails naming that
verb/event/role/index, restore, confirm green. Report both verbatim.

🔴 A null probe must be proven capable of failing. PR 1a's net went red on 7
rows and caught two real defects before a single production file moved.

- [ ] **Step 3: Commit**

```bash
git add tools/move_narration_net.py internal/mobcommands/testdata/pre_migration_literals.json
git commit -m "test(m4e-1b): extend the byte-identity net to the player pools

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Square the 24 parallel events (41 lines)

**Files:** Modify `_datafiles/world/dogmud/narration/special-moves/{drain,gore,kick,maul,pounce,rake,throttle}.yaml`

- [ ] **Step 1: For each parallel event, author the missing lines from their siblings**

Per-file new-line counts: drain 5, gore 9, kick 15, maul 7, pounce 9, rake 7,
throttle 7 (59 total, of which these 41 are the parallel share).

Method, and it is not free-writing: take the longest role's entry at index N,
read what moment it describes, and write the shorter roles' index N to describe
THAT moment from their own seat. Match the sibling's verb and imagery, shift
person (you / they), and drop the damage token from observer lines, since the
room does not read damage today.

- [ ] **Step 2: Preserve every existing line byte for byte**

The net proves this. Existing entries keep their exact wording, including the
em dashes in `gore` and `pounce` (ruling 3) and the idioms filed in rows 48-53.

- [ ] **Step 3: Gate and commit per file**

```bash
go test ./internal/movenarration/ ./internal/narration/
go test .
```

---

## Task 3: The 18 union lines, approved 2026-09-21

**Files:** Modify the six divergent verbs' YAML

All six share one shape: the actor pool carries 3 distinct feels, the actee 2,
the observer 1. Each therefore needs **1 new actee + 2 new observer** lines.

- [ ] **Step 1: Author exactly these lines**

**drain** (feels: plain whiff / target slips away / lunge glances off)
- actee: `{actor}'s draining lunge glances off you harmlessly!`
- observer: `{actor} reaches for {actee}, who slips away!`
- observer: `{actor}'s draining lunge glances off {actee} harmlessly!`

**gore** (sidestep / dodge / carries past)
- actee: `{actor}'s horned charge carries them past you as you step aside!`
- observer: `{actor} thunders toward {actee}, who dodges the gore!`
- observer: `{actor}'s horned charge carries them past {actee}, who steps aside!`

**maul** (bite misses / snap and dodge / lunge glances off)
- actee: `{actor}'s mauling lunge glances off you harmlessly!`
- observer: `{actor} snaps at {actee}, who dodges the fangs!`
- observer: `{actor}'s mauling lunge glances off {actee} harmlessly!`

**pounce** (sidestep / dodge / carries past)
- actee: `{actor}'s pounce carries them past you as you step aside!`
- observer: `{actor} springs at {actee}, who dodges the pounce!`
- observer: `{actor}'s pounce carries them past {actee}, who steps aside!`

**rake** (plain miss / dodge / glances off)
- actee: `{actor}'s rake glances off you harmlessly!`
- observer: `{actor} swipes at {actee}, who dodges the claws!`
- observer: `{actor}'s rake glances off {actee} harmlessly!`

**kick, knee variant** (turns away / glances off / target blocks)
- actee: `{actor} tries to knee you, but you turn your body away!`
- observer: `{actor}'s knee strike glances off {actee} in the grapple!`
- observer: `{actor} tries to knee {actee} in the grapple, but they block it!`

- [ ] **Step 2: 🔴 The ONE existing line this slice deliberately changes**

Owner ruling 2, 2026-09-21. `kick`'s knee miss actor line is REPLACED:

```
was:  You try to knee %s, but can't find the angle!
now:  You try to knee {actee}, but they turn their body away!
```

The feel shifts from "the attacker could not find an angle" to "the target
turned away", which is why the new actee line mirrors it as "you turn your
body away". **This one line is NOT byte-identical and the net must be updated
to expect the new text**, in the SAME commit, with the change called out.

Every other line in this slice stays byte-identical. Do not let this exception
become a habit.

- [ ] **Step 3: Commit, flagged**

```bash
git commit -m "content(m4e-1b): the 18 union variants, and one reworded knee line

The six divergent miss events get a complete trio per distinct feel, per the
owner's union ruling. One EXISTING line changes with owner approval: kick's
knee miss actor line moves from 'can't find the angle' to 'they turn their
body away', so the new actee line can mirror it. Everything else is
byte-identical and the net proves it.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 4b: Per file, author AND migrate in one commit (SUPERSEDES Tasks 2-4)

For each verb, in one commit:

1. **Author its `player_*` events**, squared. Copy every existing line from the
   Go pool BYTE FOR BYTE, replacing each `%s` with the token its argument
   supplied and letting the token replace the whole ansi-tagged span. Fill the
   parallel gaps from the sibling at the same index (see "the shape of the
   work" above); take the six divergent `miss` events' new lines verbatim from
   Task 3, which is still the approved wording.
2. **Migrate the call site** onto a `sendMoveEvent` equivalent in
   `internal/usercommands`, naming those keys as LITERALS so the key-agreement
   guard can see them.
3. **Gate**: the net's skip/fail counts for that verb go to zero, plus
   `go test .` at the ROOT and the package tests.

Order, easiest shape first:

- [ ] **Step 1: the five unpooled files** (bash, trip, grapple, shoot, throw).
      One line per role per branch; no squaring at all.
- [ ] **Step 2: drain, maul, rake, throttle.** One hit pool each. `drain` also
      gets its actor-only `player_hit_lifesteal` / `player_partial_lifesteal`
      events, whose actee and observer are absent BY DESIGN.
- [ ] **Step 3: gore, pounce.** Two hit pools each, knockdown and not.
- [ ] **Step 4: kick.** Three sub-movesets, the `questengine` notification, and
      the one deliberately reworded line (Task 3, Step 2).

## Task 4 (SUPERSEDED, see Task 4b): Migrate the twelve files, grouped by shape

Follow `internal/mobcommands/kick.go` and `move_narration.go` from PR #152 as
the template. The player side needs its own `sendMoveEvent` equivalent in
`internal/usercommands`, because the mob one is unexported in another package.

🪤 **The identity tags belong to the CALL SITE.** `kick` tags its target
`<ansi fg="username">`; `shoot` tags its target `<ansi fg="mobname">`.

🪤 **Pass token values BARE** unless the Go original wrapped them at the call
site. The YAML already bakes `<ansi fg="damage">` around `{damage}`.

🪤 **Keep every literal event key inline** at the call site. A key built by
concatenation is invisible to the root key-agreement guard, which finds
referenced events by matching literal call sites.

- [ ] **Step 1: the five unpooled files** (bash, trip, grapple, shoot, throw)
- [ ] **Step 2: drain, maul, rake, throttle** (single hit pool each)
- [ ] **Step 3: gore, pounce** (two hit pools each: knockdown and not)
- [ ] **Step 4: kick** — three sub-movesets, and the `questengine` notification

Each step ends with the net, the package tests, `go test .` at the ROOT, and a
commit.

---

## Task 5: Retire the last frozen entries

- [ ] **Step 1: Empty `m2FrozenFiles` and `m2RoutingFiles`**

The twelve player files are the last entries. Remove them together (a test
enforces the two lists agree) and re-record `testdata/m2-routing.golden`,
confirming the diff removes ONLY those files' rows.

- [ ] **Step 2: Delete `TestM2LiteralsAreFrozen`**

The arc spec says it is deleted as its files move. With both lists empty it
guards nothing, and the literal-free guard from PR 1a is the stronger
replacement. Extend that guard's file list to all 25 files.

- [ ] **Step 3: Prove the widened guard still fails**

Add a compiling prose literal back to one player file, confirm RED naming it,
remove, confirm GREEN.

---

## Task 6: Docs, ledger, boot check, playtest

- [ ] Update `internal/movenarration/context.md`: the `actor` role is now
      filled, and the store serves both twins from one file per verb
- [ ] Patch note, player-facing framing, no raw numbers, no dashes, wrap at 80
- [ ] Ledger: close any row this slice resolves; add rows for anything deferred
- [ ] Boot check in an isolated detached worktree. 🪤 A failed boot EXITS 0:
      grep for `Server Ready` and for a real panic line, never `$?`
- [ ] Playtest: a lit room this time. The point is that the three audiences now
      describe ONE moment, which darkness would hide

---

## How we know it worked

1. The net passes for every pool entry at every index, and was proven red first
2. `go test .` at the root: key agreement in both directions, role agreement,
   and no narration literal left in any of the 25 files
3. The store golden renders every event at every variant index
4. One deliberate wording change (Task 3 Step 2), called out and expected
5. Boot check clean in an isolated worktree

## Risks

| Risk | Answer |
|---|---|
| A parallel event is not actually parallel, and squaring invents a wrong moment | Task 2 reads the sibling before writing. The census classified all 30; the 6 divergent ones are handled separately in Task 3 |
| The net checks one pool entry and misses the rest | Task 1 Step 1 records every index; Step 2 proves red on a specific index |
| `kick`'s three sub-movesets collide on one event key | Keys are `<variant>_<outcome>` already, established in PR 1a |
| The reworded knee line spreads into an excuse for other edits | It is the single flagged exception, in its own commit, and the net holds every other line |
