# Messaging M3 item 8, PR 1: combat-message content pad

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring every `combat-messages` role pool in the dogmud world to
per-tier equality by authoring 984 new lines, so PR 2 can render all audiences
from one coordinated index.

**Architecture:** No production Go changes. A read-only Python audit tool
reports pool gaps and is the red/green signal. The snapshot golden is widened
from index 0 to every index first, so it captures every existing line as a
baseline; each weapon file is then padded and the golden re-recorded, making
each commit's golden diff the reviewable artifact for that file.

**Tech Stack:** YAML content under `_datafiles/world/dogmud/combat-messages/`,
Python 3 with PyYAML for the read-only audit, Go test goldens under
`internal/narration/testdata/stores/`.

Spec: `docs/superpowers/specs/2026-09-16-messaging-m3-item8-combat-messages-design.md`

---

## Read this before touching anything

### Why the pools must be equal

PR 2 renders one swing to all audiences from a **single coordinated index**, so
variant N must describe the same moment to the attacker, the defender and the
room. `narration.Variants.Len()` returns 0 when the non-empty role pools
disagree in length, and every caller treats 0 as "render nothing"
(`internal/narration/render.go:51-70`). Today 446 of 534 per-tier role groups
disagree, so without this PR the migration would silence most of combat.

Equality is required **per tier**, not just on totals. The store unions tiers
cumulatively (beginner, plus expert at skill >= 34, plus master at >= 67:
`internal/items/attack_messages.go:119-132`), so equal totals with unequal
tiers would still put an expert line opposite a master line at the same index.
Per-tier equality is exactly equivalent to union equality at all three skill
levels.

### The part no tool can check

The audit tool checks **length**. The golden records **whatever is there**.
Neither can see meaning. This passes every gate in this plan and still defeats
the purpose of the work:

```
attacker[5]  You take a {stance} stance, {position}, preparing to face {target}!
defender[5]  {source} adopts a {stance} stance against you.
room[5]      {source} adopts a masterful combat stance.
```

Six well-written, correctly wrapped, on-theme room lines appended in file order
give equal pools, a green audit and a clean golden, while telling the room
about a different moment than the two fighters. The coordinated index makes
that mismatch **permanent** rather than occasional, which is worse than today.

The correct version of the same slot:

```
attacker[5]  You take a {stance} stance, {position}, preparing to face {target}!
defender[5]  {source} adopts a {stance} stance against you.
room[5]      {source} takes a {stance} stance against {target}.
```

**The rule: author each (verb, split, tier) group as a set across all roles at
once, with the sibling lines in front of you. Never "fill role X's gap".** New
lines are always **appended** to their tier, never inserted, so existing
indices do not move.

### Tripwires

- **Never rewrite a YAML file with Python.** `yaml.safe_load` plus `yaml.dump`
  destroys the token comment header at the top of every file, reflows all
  quoting and reorders keys. Every content edit in this plan uses the Edit
  tool. The audit tool is **read-only** and must stay that way.
- **Never `git add -A` or `git add .`.** Named paths only.
- `grep` exits 1 when it finds zero matches, so any "expect zero" check must be
  run **standalone**, never in an `&&` chain, or a passing check silently skips
  everything after it.
- Player-facing text follows `dogmud-player-copy`: hard wrap at 80 characters,
  no raw numbers for damage or duration, ESL-clear phrasing.
- No em dashes or en dashes anywhere.

### Token and markup vocabulary

Every combat-message file documents its tokens in a comment header. The live
set (`internal/items/itemspec.go:200-216`):

`{itemname}` `{source}` `{sourcetype}` `{target}` `{targettype}` `{damage}`
`{exitname}` `{entrancename}` `{stance}` `{position}` `{momentum}`

Names are wrapped in ansi tags keyed by the type token, and items by a literal
class:

```yaml
- 'Your <ansi fg="item">{itemname}</ansi> bites into <ansi fg="{targettype}">{target}</ansi>!'
- '<ansi fg="{sourcetype}">{source}</ansi> lunges at <ansi fg="{targettype}">{target}</ansi>.'
```

Match the surrounding lines in the file you are editing. A `toattacker` line
addresses the player as "you" and never wraps `{source}`; a `todefender` line
wraps `{source}` and addresses the reader as "you"; a `toroom` line wraps both
and addresses neither.

### Role vocabulary

| Split | Roles present | When used |
|---|---|---|
| `together` | `toattacker`, `todefender`, `toroom` | attacker and defender share a room |
| `separate` | `toattacker`, `todefender`, `toattackerroom`, `todefenderroom` | ranged, the two are in different rooms |

Only `generic.yaml` and `shooting.yaml` have a `separate` block.

---

## File structure

**Create:**
- `tools/combat_message_pool_audit.py`: read-only pool-gap reporter, exit 1 on any gap

**Modify (content, 20 files):**
- `_datafiles/world/dogmud/combat-messages/*.yaml`

**Modify (proof):**
- `internal/narration/snapshot_test.go`: widen `buildCombatMessagesGolden` to every index, and record `coupdegrace` for `generic`
- `internal/narration/testdata/stores/combat_messages.golden`: re-recorded, 1,632 store rows to 6,975

**Modify (docs):**
- `docs/README.md`: the new tool
- `docs/superpowers/audits/messaging-m6-content-ledger.md`: item 8 rows

---

### Task 0: The pool audit tool

**Files:**
- Create: `tools/combat_message_pool_audit.py`

- [ ] **Step 1: Write the tool**

```python
"""Read-only audit of combat-message role-pool equality.

PR 2 of messaging M3 item 8 renders every audience of one swing from a single
coordinated index, which requires the role pools in each (verb, split, tier)
group to be equal in length. This reports every group where they are not.

READ ONLY. Never write YAML from Python: yaml.dump would destroy the token
comment header, the quoting and the key order of every file it touched.

Usage:
    python tools/combat_message_pool_audit.py            # whole store
    python tools/combat_message_pool_audit.py slashing   # one subtype

Exit 0 when every group is equal, 1 when any gap remains.
"""
import glob
import os
import sys

import yaml

STORE = os.path.join("_datafiles", "world", "dogmud", "combat-messages")
TIERS = ["beginner", "expert", "master"]


def groups(path):
    """Yield (verb, split, tier, {role: count}) for one file."""
    with open(path, "r", encoding="utf-8") as fh:
        doc = yaml.safe_load(fh)
    for verb, verb_body in ((doc or {}).get("options") or {}).items():
        for split, split_body in (verb_body or {}).items():
            if not split_body:
                continue
            # An explicitly-null role is a present role with an empty pool,
            # not an absent one: shooting.yaml nulls todefenderroom.
            roles = {
                role: (body if isinstance(body, dict) else {})
                for role, body in split_body.items()
            }
            for tier in TIERS:
                counts = {r: len(b.get(tier) or []) for r, b in roles.items()}
                if any(counts.values()):
                    yield verb, split, tier, counts


def main():
    only = sys.argv[1] if len(sys.argv) > 1 else None
    pattern = f"{only}.yaml" if only else "*.yaml"
    paths = sorted(glob.glob(os.path.join(STORE, pattern)))
    if not paths:
        print(f"no files matched {pattern} under {STORE}")
        return 1

    total_lines = 0
    total_groups = 0
    for path in paths:
        name = os.path.basename(path)
        rows = []
        for verb, split, tier, counts in groups(path):
            widest = max(counts.values())
            need = sum(widest - n for n in counts.values())
            if need:
                detail = ", ".join(
                    f"{r}={n}" + ("" if n == widest else f" (+{widest - n})")
                    for r, n in sorted(counts.items())
                )
                rows.append(f"    {verb}/{split}/{tier}: {detail}")
                total_lines += need
                total_groups += 1
        if rows:
            print(f"{name}")
            for row in rows:
                print(row)

    if total_lines:
        print(f"\nGAPS: {total_lines} lines across {total_groups} groups")
        return 1
    print(f"OK: every role pool is equal per tier ({len(paths)} file(s))")
    return 0


if __name__ == "__main__":
    sys.exit(main())
```

- [ ] **Step 2: Run it and confirm it reports the known gap (this is the RED)**

Run: `python tools/combat_message_pool_audit.py`

Expected: a per-file listing ending with exactly

```
GAPS: 984 lines across 446 groups
```

Exit code 1. If the total is not 984, stop and reconcile against the spec
before going further: the spec's per-file table is the reference.

- [ ] **Step 3: Confirm it scopes to a single file**

Run: `python tools/combat_message_pool_audit.py throttle`

Expected: `throttle.yaml` gaps listed, `GAPS: 28 lines across 21 groups`,
exit 1.

The tool's ability to report a **gap** is proven here, which is the direction
that matters right now. Its ability to report **success** is proven the first
time a file is padded, at Task 2 Step 4, and the success path is
sabotage-tested at Task 4 Step 4 before the whole-store green is trusted. Do
not treat any `OK` before Task 4 Step 4 as proof of anything.

- [ ] **Step 4: Verify per-file totals match the spec**

Run: `for f in bite bludgeoning claws cleaving drain generic gore maul pounce sceptre shooting slam slashing stabbing staff sting throttle unarmed wand whipping; do printf "%-14s " "$f"; python tools/combat_message_pool_audit.py "$f" | tail -1; done`

Expected, matching the spec's table exactly:

| File | Lines | Groups | | File | Lines | Groups |
|---|---|---|---|---|---|---|
| `bite` | 41 | 21 | | `slam` | 41 | 21 |
| `bludgeoning` | 56 | 21 | | `slashing` | 60 | 21 |
| `claws` | 45 | 21 | | `stabbing` | 56 | 21 |
| `cleaving` | 40 | 21 | | `staff` | 55 | 24 |
| `drain` | 36 | 21 | | `sting` | 55 | 21 |
| `generic` | 61 | 31 | | `throttle` | 28 | 21 |
| `gore` | 41 | 21 | | `unarmed` | 63 | 24 |
| `maul` | 41 | 21 | | `wand` | 51 | 24 |
| `pounce` | 41 | 21 | | `whipping` | 46 | 21 |
| `sceptre` | 54 | 24 | | `shooting` | 73 | 25 |

- [ ] **Step 5: Commit**

```bash
git add tools/combat_message_pool_audit.py
git commit -m "tools(combat): read-only audit of combat-message role-pool equality

Reports every (verb, split, tier) group whose role pools differ in length,
which is what blocks the coordinated index in M3 item 8 PR 2. Baseline is
984 lines across 446 groups. Read only: writing YAML from Python would
destroy the token comment header and the quoting in every file.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 1: Widen the golden to every index, and record the baseline

This must happen **before** any content is added. Recording the widened golden
first is what makes every later commit provably additions-only.

**Files:**
- Modify: `internal/narration/snapshot_test.go:292-357`
- Modify: `internal/narration/testdata/stores/combat_messages.golden`

- [ ] **Step 1: Confirm today's golden is blind to the pad**

Read `internal/narration/snapshot_test.go:341`. It calls
`mo.GetWith(narration.SequencePicker())` with a fresh picker per tuple, so it
records index 0 of each tier only. Appending lines cannot change index 0, so a
984-line pad would move 6 of the 1,632 store rows. That is the gap this
task closes.

- [ ] **Step 2: Replace the two render loops in `buildCombatMessagesGolden`**

In `internal/narration/snapshot_test.go`, replace the `together` loop
(currently lines 337-344) with:

```go
			for _, role := range togetherRoles {
				stm := role.get(opts.Together)
				for _, tier := range tiers {
					mo := tier.get(stm)
					for idx := 0; idx < len(mo); idx++ {
						text := substituteTokens(string(mo[idx]))
						fmt.Fprintf(&b, "%s|%s|together|%s|%s|%d => %s\n",
							subtype, intensity, role.name, tier.name, idx, text)
					}
				}
			}
```

and the `separate` loop (currently lines 346-355) with:

```go
			if hasSeparate[subtype] {
				for _, role := range separateRoles {
					stm := role.get(opts.Separate)
					for _, tier := range tiers {
						mo := tier.get(stm)
						for idx := 0; idx < len(mo); idx++ {
							text := substituteTokens(string(mo[idx]))
							fmt.Fprintf(&b, "%s|%s|separate|%s|%s|%d => %s\n",
								subtype, intensity, role.name, tier.name, idx, text)
						}
					}
				}
			}
```

Indexing `mo[idx]` directly rather than going through a picker is deliberate:
the golden is enumerating the pool, not sampling it, and a picker would consume
draws for no reason.

- [ ] **Step 3: Record `coupdegrace`, which the golden has never covered**

The builder's intensity list has 8 entries and omits `items.CoupDeGrace`
(`snapshot_test.go:304`), so `generic`'s `coupdegrace` pools are invisible to
the golden today and 6 of this PR's 984 lines would not appear in any diff.

`coupdegrace` is authored only in `generic.yaml`, and
`GetPreAttackMessage` falls back to `Generic` for a missing intensity
(`internal/items/attack_messages.go:170-190`), so looping it over all 20
subtypes would record generic's lines 19 extra times. Scope it to `generic`,
mirroring how `hasSeparate` is already handled.

Immediately inside `for _, subtype := range subtypes {`, add:

```go
		intensityList := intensities
		if subtype == "generic" {
			intensityList = append(append([]items.Intensity{}, intensities...), items.CoupDeGrace)
		}
```

and change the inner loop header from `for _, intensity := range intensities {`
to:

```go
		for _, intensity := range intensityList {
```

- [ ] **Step 4: Update the header lines the builder writes**

Replace the `dimensions` and `separate section` header lines (currently lines
301-302) with:

```go
	fmt.Fprintf(&b, "# dimensions: subtype x intensity x section x role x tier x index, every authored line exactly once\n")
	fmt.Fprintf(&b, "# separate section present only for: generic, shooting (per source at time of writing)\n")
	fmt.Fprintf(&b, "# PR 1 of M3 item 8 widened this from index 0 only; PR 2 re-keys it to coordinated rows\n\n")
```

- [ ] **Step 5: Re-record the golden**

Run: `go test ./internal/narration/... -run TestSnapshotStores -update`

Expected: PASS, and `git status` shows `combat_messages.golden` modified.

- [ ] **Step 6: Confirm the baseline covers every existing line exactly once**

Run: `grep -c " => " internal/narration/testdata/stores/combat_messages.golden`

Expected: **`5994`**. That is 5,991 store rows, which is every authored line in
the dogmud tree counted independently from the YAML, plus the 3
derived-selection rows at the foot of the file.

For reference, the pre-task golden had 1,635 such rows (1,632 store rows at
index 0 only, plus the same 3). If the new count is not 5994, the builder is
either missing pools or recording some twice. The two most likely causes are
the `coupdegrace` scoping in Step 3 being applied to all subtypes rather than
`generic` alone, which would add 19 duplicate sets, or a role loop not being
converted to index enumeration.

- [ ] **Step 7: Run the full narration suite**

Run: `go test ./internal/narration/...`

Expected: PASS.

- [ ] **Step 8: Commit and tag the baseline**

```bash
git add internal/narration/snapshot_test.go internal/narration/testdata/stores/combat_messages.golden
git commit -m "test(narration): widen combat-messages golden from index 0 to every index

The builder took a fresh SequencePicker per tuple and so recorded only
index 0 of each tier. Appending lines cannot change index 0, so the
984-line pad in this PR would have moved 6 of 1,632 store rows: the
golden was effectively blind to the content work it is meant to guard.

Also records coupdegrace, which the 8-entry intensity list omitted
entirely, scoped to generic since that is the only file authoring it and
every other subtype reaches it through the Generic fallback.

Now enumerates every authored line exactly once, keyed
subtype|intensity|split|role|tier|index: 5991 store rows, up from 1632.
Recorded before any content lands so each later commit's diff is
provably additions only.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"

git tag m3-item8-golden-baseline
```

The tag is what Task 4 diffs against to prove no pre-existing line changed
across the whole PR. Delete it after the PR merges: `git tag -d m3-item8-golden-baseline`.

---

### Task 2: Pad `slashing.yaml`, the reviewed sample

`slashing` is first and is a **hard checkpoint**. The other 19 files do not
start until the owner has accepted the pairing in this one.

**Files:**
- Modify: `_datafiles/world/dogmud/combat-messages/slashing.yaml`
- Modify: `internal/narration/testdata/stores/combat_messages.golden`

- [ ] **Step 1: List the exact gaps**

Run: `python tools/combat_message_pool_audit.py slashing`

Expected: 21 groups listed, ending `GAPS: 60 lines across 21 groups`.

- [ ] **Step 2: Read every group you are about to touch, all roles together**

For each group the tool named, open the file and read `toattacker`,
`todefender` and `toroom` for that verb and tier side by side. You are writing
the missing entries of a table, not filling a list.

Worked example, `critical` / `together` / `beginner`, which today is 4/4/3:

```
toattacker [0] Your {itemname} CRITICALLY LACERATES {target}!
           [1] You deliver a CRITICAL STRIKE to {target} with your {itemname}!
           [2] Your {itemname} TEARS THROUGH {target}!
           [3] You land a DEVASTATING HIT on {target}!
todefender [0] {source}'s {itemname} CRITICALLY LACERATES you!
           [1] {source}'s {itemname} delivers a CRITICAL STRIKE!
           [2] You are DEVASTATED by {source}'s {itemname}!
           [3] {source} lands a DEVASTATING HIT on you!
toroom     [0] {source}'s {itemname} CRITICALLY LACERATES {target}!
           [1] {source}'s {itemname} delivers a CRITICAL STRIKE to {target}!
           [2] {source} DEVASTATES {target} with their {itemname}!
```

`toroom` is missing index 3, and index 3 in the other two roles is the
DEVASTATING HIT moment. So the line to append is the room's view of that same
moment:

```yaml
        - '<ansi fg="{sourcetype}">{source}</ansi> lands a DEVASTATING HIT on <ansi fg="{targettype}">{target}</ansi>!'
```

Not a new idea about criticals. The same event, from the third seat.

- [ ] **Step 3: Append the missing lines with the Edit tool**

Append only. Never insert, never reorder, never touch an existing line. Use the
Edit tool on the YAML directly; do not round-trip the file through Python.

Keep each line at or under 80 characters of rendered text, ignoring the ansi
tags, which are markup rather than visible width.

- [ ] **Step 4: Verify the file is now equal**

Run: `python tools/combat_message_pool_audit.py slashing`

Expected: `OK: every role pool is equal per tier (1 file(s))`, exit 0.

- [ ] **Step 5: Verify the file still parses and the store still loads**

Run: `go test ./internal/items/... -run TestLoad`

Expected: PASS. A YAML syntax error or a mis-indented block fails here.

- [ ] **Step 6: Re-record the golden and prove the diff is additions only**

Run: `go test ./internal/narration/... -run TestSnapshotStores -update`

Then, as a **standalone** command (not chained, because grep exits 1 on no
match):

```bash
git diff -- internal/narration/testdata/stores/combat_messages.golden | grep '^-' | grep -v '^---'
```

Expected: **no output**. Any line here means an existing golden row changed,
which means an existing message line was edited, reordered or deleted. Fix the
YAML rather than re-recording over it.

Then confirm the additions are the right size:

```bash
git diff --numstat -- internal/narration/testdata/stores/combat_messages.golden
```

Expected: `60	0	internal/narration/testdata/stores/combat_messages.golden`

- [ ] **Step 7: Read the added lines back as a player would**

Run: `git diff -- internal/narration/testdata/stores/combat_messages.golden | grep '^+' | grep -v '^+++'`

Read the 60 rows. Each should sit beside its siblings at the same index and
describe the same moment. This is the review that matters and no command
performs it.

- [ ] **Step 8: Commit**

```bash
git add _datafiles/world/dogmud/combat-messages/slashing.yaml internal/narration/testdata/stores/combat_messages.golden
git commit -m "content(combat): pad slashing role pools to per-tier equality

60 lines across 21 groups. Each added line is the missing seat at an
index the other roles already describe, so the coordinated index in PR 2
narrates one moment to all three audiences.

Golden diff is additions only: no existing line moved.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 9: STOP. Owner review checkpoint.**

Show the owner the 60 added golden rows grouped by verb and tier. Do not begin
Task 3 until the pairing has been accepted. If the owner wants a different
register or pairing convention, it is corrected here once rather than in 20
files.

---

### Task 3: Pad the remaining 19 files

This is one procedure applied per file, stated in full once because the steps
are identical and the only thing that varies is the filename and the expected
counts. Run it independently for each row of the table. Under
subagent-driven-development, dispatch one subagent per row, each given this
task's full text plus its row.

**Files, per run:**
- Modify: `_datafiles/world/dogmud/combat-messages/<FILE>.yaml`
- Modify: `internal/narration/testdata/stores/combat_messages.golden`

| Order | `<FILE>` | Expected lines | Expected groups | Note |
|---|---|---|---|---|
| 1 | `generic` | 61 | 31 | fallback for every subtype; has `separate` and the only `coupdegrace` |
| 2 | `shooting` | 73 | 25 | has `separate`; includes the nulled-role fix below |
| 3 | `unarmed` | 63 | 24 | |
| 4 | `bludgeoning` | 56 | 21 | |
| 5 | `stabbing` | 56 | 21 | |
| 6 | `sting` | 55 | 21 | |
| 7 | `staff` | 55 | 24 | |
| 8 | `sceptre` | 54 | 24 | |
| 9 | `wand` | 51 | 24 | |
| 10 | `whipping` | 46 | 21 | |
| 11 | `claws` | 45 | 21 | |
| 12 | `bite` | 41 | 21 | |
| 13 | `gore` | 41 | 21 | |
| 14 | `maul` | 41 | 21 | |
| 15 | `pounce` | 41 | 21 | |
| 16 | `slam` | 41 | 21 | |
| 17 | `cleaving` | 40 | 21 | |
| 18 | `drain` | 36 | 21 | |
| 19 | `throttle` | 28 | 21 | |

`generic` goes first because `GetPreAttackMessage` falls back to it for any
subtype missing an intensity (`internal/items/attack_messages.go:170-190`), so
its lines are the ones most players actually see.

- [ ] **Step 1: List the exact gaps for this file**

Run: `python tools/combat_message_pool_audit.py <FILE>`

Expected: the group listing, ending with
`GAPS: <expected lines> lines across <expected groups> groups` from the table.
If the numbers differ from the table, stop: either an earlier task touched this
file or the table is stale.

- [ ] **Step 2: Read every named group across all roles together**

For each group the tool named, read every role for that verb, split and tier
side by side before writing anything. Identify, for each missing index, what
moment the other roles already put at that index. Write that moment from the
missing seat.

For a `separate` block (`generic`, `shooting`) there are four seats, not three:
`toattacker`, `todefender`, `toattackerroom` and `todefenderroom`. The two room
roles are different audiences in different rooms: `toattackerroom` sees the
shooter, `todefenderroom` sees the target being hit. They are not
interchangeable and must not be copies of each other.

- [ ] **Step 3 (`shooting` only): fix the two nulled roles**

`shooting.yaml` sets `todefenderroom: null` on `prepare` and `wait`. That is a
gap rather than a deliberate silence: `generic.yaml` authors that role for both
verbs. Replace each `null` with a full tiered block matching the sibling
shape, so both verbs end at 9 lines like shooting's other pools.

The tier split to match the siblings is 4 beginner, 3 expert, 2 master for each
of the two verbs, giving 9 per verb and 18 lines in total. These are what the
defender's room sees while a shooter in another room takes aim: the target
reacting, not the shooter acting.

```yaml
      todefenderroom:
        beginner:
        - '<ansi fg="{targettype}">{target}</ansi> spots an attacker taking aim from the <ansi fg="exit">{entrancename}</ansi>.'
```

Append the remaining lines in the same shape, reading the `toattackerroom`
block for the same verb and tier to find the moment each index describes.

- [ ] **Step 4: Append the missing lines with the Edit tool**

Append only. Never insert, never reorder, never touch an existing line. Edit
the YAML directly; do not round-trip it through Python, which would destroy the
token comment header and the quoting.

Keep each line at or under 80 characters of rendered text, ignoring ansi tags.
Match the weapon's voice: `bite`, `claws`, `gore`, `maul`, `pounce`, `sting`
and `throttle` are natural attacks and should read as an animal, not a
swordsman.

- [ ] **Step 5: Verify the file is now equal**

Run: `python tools/combat_message_pool_audit.py <FILE>`

Expected: `OK: every role pool is equal per tier (1 file(s))`, exit 0.

- [ ] **Step 6: Verify the store still loads**

Run: `go test ./internal/items/... -run TestLoad`

Expected: PASS.

- [ ] **Step 7: Re-record the golden and prove the diff is additions only**

Run: `go test ./internal/narration/... -run TestSnapshotStores -update`

Then, as a **standalone** command:

```bash
git diff -- internal/narration/testdata/stores/combat_messages.golden | grep '^-' | grep -v '^---'
```

Expected: **no output**, with one exception. For `shooting` only, expect
exactly 6 removed rows, the six previously-empty `todefenderroom` rows for
`prepare` and `wait` across the three tiers, each replaced by a real line.

Then:

```bash
git diff --numstat -- internal/narration/testdata/stores/combat_messages.golden
```

Expected: `<expected lines>	0` for every file except `shooting`, which is
`79	6` (73 additions plus the 6 replaced rows).

- [ ] **Step 8: Read the added lines back as a player would**

Run: `git diff -- internal/narration/testdata/stores/combat_messages.golden | grep '^+' | grep -v '^+++'`

Read them grouped by verb and tier, checking each new line against its siblings
at the same index.

- [ ] **Step 9: Commit**

```bash
git add _datafiles/world/dogmud/combat-messages/<FILE>.yaml internal/narration/testdata/stores/combat_messages.golden
git commit -m "content(combat): pad <FILE> role pools to per-tier equality

<N> lines across <G> groups. Each added line is the missing seat at an
index the other roles already describe.

Golden diff is additions only: no existing line moved.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Whole-store verification

**Files:** none modified; this task only verifies.

- [ ] **Step 1: The store is fully equal**

Run: `python tools/combat_message_pool_audit.py`

Expected: `OK: every role pool is equal per tier (20 file(s))`, exit 0.

- [ ] **Step 2: The line count moved by exactly 984**

Run: `grep -c " => " internal/narration/testdata/stores/combat_messages.golden`

Expected: **`6978`**, which is the 5994 baseline from Task 1 Step 6 plus the
984 padded lines. Independently: 6,975 store rows plus the 3
derived-selection rows.

- [ ] **Step 3: No existing line changed across the whole PR**

Run, standalone:

```bash
git diff m3-item8-golden-baseline -- internal/narration/testdata/stores/combat_messages.golden | grep '^-' | grep -v '^---'
```

Expected: exactly 6 lines, all `shooting` `todefenderroom` rows for `prepare`
and `wait`, which go from empty to a real first line. Anything else is an
edited, reordered or deleted message line.

The baseline must be the Task 1 tag, not the branch point. Task 1 re-keyed
every row in the golden, so diffing against anything earlier shows the whole
file as changed and proves nothing.

- [ ] **Step 4: Prove the audit tool would still fail if the store regressed**

The tool has been green for a while by this point, so prove it can still go
red before trusting it.

Temporarily delete one line from any tier pool in `throttle.yaml`, then run
`python tools/combat_message_pool_audit.py throttle` and confirm it reports a
gap and exits 1. Restore the line with `git checkout -- <path>` and confirm the
tool returns to exit 0.

Note: `git checkout <ref> -- <path>` **stages** the revert. Check `git status`
afterwards and unstage if needed.

- [ ] **Step 5: Full test suite**

Run: `go test ./...`

Expected: PASS. Combat tests carry a roughly 2.3% attack-fumble flake
(`dogmud-writing-tests`); re-run any single combat failure alone before
treating it as real.

- [ ] **Step 6: Boot the server**

Follow `dogmud-shipping`'s detached-worktree boot check. The store now loads
984 more lines; a mis-indented block that survived the YAML parse but broke a
tier would surface here.

---

### Task 5: Documentation and the content ledger

**Files:**
- Modify: `docs/README.md`
- Modify: `docs/superpowers/audits/messaging-m6-content-ledger.md`

- [ ] **Step 1: Add the tool to the docs index**

Add a row to the tools table in `docs/README.md`:

```markdown
| [`../tools/combat_message_pool_audit.py`](../tools/combat_message_pool_audit.py) | Read-only check that every combat-message role pool is equal per tier, which is what the coordinated narration index in M3 item 8 requires. Exits 1 with a per-group listing when any pool is short |
```

- [ ] **Step 2: Add item 8's rows to the M6 content ledger**

Append rows to `docs/superpowers/audits/messaging-m6-content-ledger.md` for
anything this PR deferred rather than authored. At minimum:

- the `world/default/combat-messages` tree is upstream-shaped and unloadable,
  commented in PR 2 rather than fixed
- `coupdegrace` is authored only in `generic.yaml`, so every other subtype
  reaches it through the Generic fallback rather than a weapon-specific line

If the padding surfaced further deferred text, add a row per item in this same
commit, per the ledger rule.

- [ ] **Step 3: Commit**

```bash
git add docs/README.md docs/superpowers/audits/messaging-m6-content-ledger.md
git commit -m "docs(combat): pool audit tool and item 8 content ledger rows

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Pre-push gate and PR

- [ ] **Step 1: Run the pre-push gate**

Follow `dogmud-shipping`'s gate in its stated order. Do not skip hooks.

- [ ] **Step 2: Open the PR**

```bash
gh pr create --repo pruuk/DOGMud \
  --base master \
  --head feature/messaging-m3-item8-combat-messages \
  --title "content(combat): pad combat-message role pools to per-tier equality (M3 item 8, PR 1)" \
  --body-file <path to a written body>
```

`--repo pruuk/DOGMud` is mandatory. This repo is a fork of
`GoMudEngine/GoMud` and `gh` defaults to the parent; a bare `gh pr create` has
already opened a PR on upstream once.

The body must state: 984 lines across 446 groups, why equality is required per
tier, that the golden was widened first so every content commit is provably
additions-only, and that PR 2 carries the migration and cannot start until this
merges.

End the body with:

```
🤖 Generated with [Claude Code](https://claude.com/claude-code)
```

- [ ] **Step 3: Do not deploy**

The owner runs all deploys. This PR merges into the existing undeployed stack.
