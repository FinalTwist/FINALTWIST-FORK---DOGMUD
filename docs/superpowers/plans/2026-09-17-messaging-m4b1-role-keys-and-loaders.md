# Messaging M4b-1: role keys, shipped-data guards, loader policy

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every narration store spells its roles `actor`, `actee`, `observer`, `remote_observer` in shipped YAML and Go struct tags; every store is validated against its shipped data by a test that fails the build; and every event store fails the boot on bad data, loaded from `main.go` through the configured data path.

**Architecture:** The guards land BEFORE the renames, because a rename that silently empties a store is exactly what the guards exist to catch. Each rename is one store group per commit, with a mechanical proof: the previous golden, with its labels passed through the same translation table, must equal the new golden byte for byte. Re-recording a golden is forbidden.

**Tech Stack:** Go 1.23, `gopkg.in/yaml.v3`, snapshot goldens in `internal/narration/testdata/stores/`, AST and data guards at repo root (package `main`).

**Spec:** `docs/superpowers/specs/2026-09-17-messaging-m4-flip-design.md` (M4b section).

**Scope note:** M4b's other half (one `DefenseType`, the spell `target_defense_type` convention, core-drain's channel) is a SECOND PR with its own plan, written when this one lands. This plan is self-contained and ships on its own.

---

## Facts verified against source (master `7874b0364`, 2026-09-17, after M4a merged)

### Role keys today

| Store | Struct : line | Authored keys | Roles they map to | Mapper |
|---|---|---|---|---|
| defence | `items/defensive_messages.go:47-51` | `toattacker`, `todefender`, `toroom` | actor, actee, observer | `RenderTriad` `:143-160` |
| combat together | `items/attack_messages.go:34-38` | `toattacker`, `todefender`, `toroom` | actor, actee, observer | `TogetherMessages.Render` `:130-140` |
| combat separate | `items/attack_messages.go:40-45` | `toattacker`, `todefender`, `toattackerroom`, `todefenderroom` | actor, actee, observer, remote_observer | `SeparateMessages.Render` `:152-163` |
| taunt | `combat/taunt_messages.go:24-28` | `toattacker`, `todefender`, `toroom` | actor, actee, observer | `variants` `:80-86` |
| grapple triads | `grapplemessaging/loader.go:22-26` | `controller`, `controlled`, `observers` | actor, actee, observer | `RenderTriad` `render.go:158-170` |
| grapple gradients | `grapplemessaging/loader.go:35-39` | `self`, `partner`, `observers` | actor, actee, observer | `RenderGradient` `render.go:211-223` |
| conditions | `conditions/conditionspec.go:180-185` | `start/trigger/end_user_text`, `..._room_text` | **actee**, observer | `Narration` `narration.go:28-39` |
| spells | `spells/spells.go` | `cast/wait/magic_user_text`, `..._room_text` | actor, observer | `Narration` `narration.go:24-35` |
| quests actions | `quests/triggers.go` | `send_text`, `room_text` | actor, observer | `ActionDef.Narration` `narration.go:14-16` |
| quests rewards | `quests/quests.go` | `playermessage`, `roommessage` | actor, observer | `QuestReward.Narration` `narration.go:27-29` |
| crafting | `crafting/crafting.go` | `success_message`, `success_room_message`, `failure_message`, `failure_room_message` | actor, observer | `RecipeSpec.Narration` `narration.go:24-33` |
| position_control | `hooks/Position_Messaging.go:37-41` | `attacker`, `target`, `room`; `self`, `room` | actor, actee, observer | `sendSubmissionTriple` `:229-270` |
| itemvoices, casting, gossip, tips, weather | - | no role keys (single-role or actorless stores) | - | - |

### Loaders today

| Store | Failure policy | Boot-loaded |
|---|---|---|
| defence, combat, itemvoices, casting, conditions, spells, quests, crafting | panic | yes (`main.go:1631,1634,1642,1643,1646,1844,1938`) |
| **taunt** | logs, empty map, continues | yes, `main.go:1937` |
| **grapple** | logs, empty library, continues; completeness only warns | **no, lazy `sync.Once`** at `hooks/Position_GrappleTick.go:60-84`, hardcoded path |
| **position_control** | warns, degrades to silent templates | **no, lazy `sync.Once`** at `hooks/Position_Messaging.go:70`, hardcoded path |
| gossip, tips | missing file: empty + info log; malformed: panic | yes, `main.go:1650,1651` |
| weather | never panics, by documented intent (`modules/weather/content/emotes.go:118-126`) | module init |

### Golden labels, and where they live in a row

Two different positions, so the translation must handle both:

| Golden | Row shape | Authored labels |
|---|---|---|
| `combat_messages.golden` | `bite\|prepare\|together\|beginner\|0 => toattacker=... todefender=... toroom=...` | INSIDE the value |
| `defense_messages.golden` | `block\|weak\|todefender => ...` | in the row KEY |
| `taunt_messages.golden` | `rhetoric\|hit\|toattacker => ...` | row key |
| `messaging_grapple.golden` | `advancements\|<key>\|controller => ...` | row key |
| `conditions.golden` | `condition\|0\|start_user_text => ...` | row key |
| `quests.golden` | `quest\|2\|rewards\|playermessage => ...` | row key |
| `crafting.golden` | `recipe\|<id>\|success_message => ...` | row key |
| `position_control.golden` | `gradient\|controlled\|<key>\|self => ...`, `...\|attacker =>` | row key |
| `itemvoices`, `casting_messages`, `spells`, `weather_emotes`, `gossip`, `tips` | no role label | none |

Row keys are built in `internal/narration/snapshot_test.go` at `:422,442` (combat), `:515-517,546-548` (defence), `:591-593` (taunt), `:686-688` (grapple), `:1011-1019` (conditions), `:1136-1139` (quests), `:1179-1183` (crafting), `:1539-1545` (position_control).

### The M0 surface registry

`textSurfaceRegistry` (`messaging_surface_guard_test.go:64`) keys entries by YAML spelling: `toattacker` `:142`, `todefender` `:143`, `toroom` `:144`, `controller` `:159`, `observers` `:161`, `start_user_text` `:76`, `success_message` `:90`, `playermessage` `:110`, `send_text` `:117` and more. A second set at `:242-243` lists role spellings. **Renaming a key without updating both fails the guard**, which is the intended behaviour: the guard is the reminder.

### Corrections to earlier surveys

- **`SpellData.TargetDefenseType` HAS live readers**, contrary to a survey that reported only a comment: `internal/hooks/spell_resolution.go:168` (the `== ""` uncontested shortcut), `:1245` (`spellAttackChannel`'s switch), and `internal/hooks/combat_shared_helpers.go:87` (mitigation switch). That work belongs to M4b-2; it is recorded here so the false negative is not inherited.
- Shipped `target_defense_type` under `_datafiles/world/dogmud/spells`: physical 11, mental 9, social 1, `none` 13, absent 25 of 59 files. All 8 default-world spells omit it.

---

## Task 1: Shipped-data guards, before any rename

A guard here catches what the boot cannot: three stores log and continue, so today their data can be wrong and nothing says so.

**Files:**
- Create: `shipped_narration_data_guard_test.go` (repo root, package `main`)

- [ ] **Step 1: Write the guard**

```go
// TestShippedNarrationDataValidates loads every narration store from the
// shipped data path and fails the BUILD when one does not validate.
//
// Why this exists: three stores log and continue rather than panicking
// (taunt, grapple, position_control), so bad data in them reaches players as
// silence with only a log line. A build-time check over shipped data is the
// only thing that catches that today, and it is the pattern weather already
// uses (modules/weather/content/biome_coupling_test.go).
//
// It loads from _datafiles/world/dogmud explicitly rather than through the
// config, because a test binary does not read config.yaml: it would get the
// Go default (_datafiles/world/default), which is a different, vestigial
// world (internal/configs/config.filepaths.go:23).
func TestShippedNarrationDataValidates(t *testing.T) {
	const root = "_datafiles/world/dogmud"
	t.Run("taunt", func(t *testing.T) {
		groups, err := loadTauntForGuard(root)
		if err != nil {
			t.Fatalf("taunt-messages: %v", err)
		}
		if len(groups) == 0 {
			t.Fatal("taunt-messages loaded zero groups: the store is shipped, so zero means the load failed silently")
		}
		for id, g := range groups {
			if err := g.Validate(); err != nil {
				t.Errorf("taunt group %q: %v", id, err)
			}
		}
	})
	// one subtest per store, same shape
}
```

Write one subtest per store: taunt, grapple outcomes, position_control, defence, combat-messages, itemvoices, casting, conditions, spells, quests, crafting, gossip, tips. Each asserts the load succeeds, the store is non-empty, and every record's `Validate` (where one exists) returns nil. For a store whose loader already panics, the subtest still earns its place: it names the store in the failure instead of a boot stack trace.

- [ ] **Step 2: Run it**

Run: `go test . -run TestShippedNarrationDataValidates -v`
Expected: PASS, with one line per store subtest.

If a subtest FAILS, that is a find. Report the store and the record before changing anything: shipped data that has been quietly invalid is exactly what this task exists to surface.

- [ ] **Step 3: Prove each subtest can fail**

For three stores (one that panics, one that logs and continues, one ambient), temporarily break the shipped file, run the guard, confirm it goes red naming that store, then restore. Use `git checkout -- <path>` to restore and confirm with `git status --short` that the tree is clean afterwards.

⚠️ Restore by `git checkout -- <path>` only for files you did NOT create in this session; never sabotage an uncommitted file, because that command discards it.

- [ ] **Step 4: Commit**

```bash
git add shipped_narration_data_guard_test.go
git commit -m "test: validate every narration store against shipped data at build time

Three stores log and continue rather than panicking, so bad data in them
reaches players as silence with only a log line. This is the check that
fails the build instead, and it lands before M4b's key renames so a
rename that empties a store cannot pass unnoticed.

Proven capable of failing for a panicking store, a logging store and an
ambient store.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: The two-tier loader policy

**Files:**
- Modify: `internal/combat/taunt_messages.go:91-109`
- Modify: `internal/grapplemessaging/loader.go:68`, `internal/hooks/Position_GrappleTick.go:60-84`
- Modify: `internal/hooks/Position_Messaging.go:70`
- Move: `_datafiles/messages/position_control.yaml` -> `_datafiles/world/dogmud/messaging/position_control.yaml`
- Copy: both messaging files into `_datafiles/world/default/messaging/`
- Modify: `main.go`

- [ ] **Step 1: Move position_control under the world tree**

```bash
mkdir -p _datafiles/world/dogmud/messaging
git mv _datafiles/messages/position_control.yaml _datafiles/world/dogmud/messaging/position_control.yaml
```

`_datafiles/messages/` is then empty and the directory goes away. Grep for the old path and fix every hit (`internal/hooks/Position_Messaging.go:73`, `internal/narration/snapshot_test.go`, any context.md).

- [ ] **Step 2: Give the default world both messaging files**

The event tier fails the boot on a missing file, and `_datafiles/world/default` is what a config with an empty `DataFiles` resolves to (`internal/configs/config.filepaths.go:23`). Copy both files so that world still boots:

```bash
mkdir -p _datafiles/world/default/messaging
cp _datafiles/world/dogmud/messaging/position_control.yaml _datafiles/world/default/messaging/position_control.yaml
cp _datafiles/world/dogmud/messaging/grapple_outcomes.yaml _datafiles/world/default/messaging/grapple_outcomes.yaml
```

- [ ] **Step 3: Both loaders read the configured path and are called from main.go**

Replace the hardcoded path in `internal/hooks/Position_GrappleTick.go:61` and in `loadPositionMessages`:

```go
	path := filepath.Join(configs.GetFilePathsConfig().DataFiles, "messaging", "grapple_outcomes.yaml")
```

Add exported load functions (`grapplemessaging.LoadFromDataFiles()` and a hooks-side equivalent) and call them from `main.go` beside the other store loads, near `main.go:1650`. Keep the `sync.Once` so a test that reaches the store without booting still loads it once, but the boot path no longer depends on a grapple tick happening first.

- [ ] **Step 4: Event tier fails the boot**

Taunt (`combat/taunt_messages.go:99-104`), grapple (`grapplemessaging.Load`'s caller) and position_control now panic on a load or validation error, in the style of `internal/items/itemspec.go:788`. Keep the log line; add the panic after it so the operator still sees which file.

Ambient stores are untouched: gossip and tips keep their missing-file tolerance, weather keeps its fail-soft loader and its "do not fix this into a panic" comment.

- [ ] **Step 5: Document the policy where a reader will find it**

Add to `internal/narration/context.md` a short section naming the two tiers, which stores are in each, and the rule that decides: an event store's silence misleads a player mid-action, an ambient store's silence costs nothing.

- [ ] **Step 6: Boot check**

```bash
git worktree add --detach C:/tmp/dogmud-boot-check HEAD
cp _datafiles/config.yaml C:/tmp/dogmud-boot-check/_datafiles/config.yaml
cd C:/tmp/dogmud-boot-check && go build -o boot-check.exe .
timeout 180 ./boot-check.exe > boot.log 2>&1
grep -cE "^panic:|goroutine [0-9]+ \[running\]|runtime error" boot.log   # want 0
grep -c "Server Ready" boot.log                                          # want 1
```

Exit 124 is success. Then prove the new policy bites: break `grapple_outcomes.yaml` in the worktree only, rebuild, boot, and confirm it panics naming the file. Restore by deleting the worktree.

- [ ] **Step 7: Commit**

```bash
git add internal/combat/taunt_messages.go internal/grapplemessaging internal/hooks/Position_GrappleTick.go internal/hooks/Position_Messaging.go internal/narration/snapshot_test.go internal/narration/context.md main.go _datafiles/world/dogmud/messaging _datafiles/world/default/messaging
git commit -m "feat(messaging): two-tier loader policy, decided by the cost of silence

Event narration fails the boot on bad data: a silent fight or a silent
grapple misleads a player mid-action. Ambient narration (weather, tips,
gossip) keeps warning and going quiet, because a missing weather line
costs nothing.

Taunt, grapple and position_control move into the first tier. Grapple and
position_control also stop loading lazily from a hardcoded dogmud path and
load at boot from the configured data path, which is what made their
failure invisible in the first place.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: The rename proof, built before the first rename

**Files:**
- Create: `tools/messaging_role_key_check.py`

A rename legitimately changes golden labels, so byte-identity cannot be the proof. The proof is: **the old golden, with labels translated, equals the new golden.**

- [ ] **Step 1: Write the checker**

```python
#!/usr/bin/env python3
"""Prove a role-key rename changed labels and nothing else (M4b-1).

Compares each golden at a base git ref against the working tree copy,
after applying the same label translation the rename applied. Any
difference that is not a label is a real change and is reported.

Usage:
  python tools/messaging_role_key_check.py --base HEAD --group combat
"""
import argparse
import re
import subprocess
import sys

GOLDEN_DIR = "internal/narration/testdata/stores"

# Per store group: golden file -> ordered (old, new) label pairs. Longest
# first, so toattackerroom is rewritten before toattacker.
GROUPS = {
    "combat": {
        "combat_messages.golden": [
            ("toattackerroom", "observer"), ("todefenderroom", "remote_observer"),
            ("toattacker", "actor"), ("todefender", "actee"), ("toroom", "observer"),
        ],
        "defense_messages.golden": [
            ("toattacker", "actor"), ("todefender", "actee"), ("toroom", "observer"),
        ],
        "taunt_messages.golden": [
            ("toattacker", "actor"), ("todefender", "actee"), ("toroom", "observer"),
        ],
    },
    "grapple": {
        "messaging_grapple.golden": [
            ("controller", "actor"), ("controlled", "actee"), ("observers", "observer"),
            ("self", "actor"), ("partner", "actee"),
        ],
    },
    "kindb": {
        "conditions.golden": [
            ("start_user_text", "start_actee"), ("start_room_text", "start_observer"),
            ("trigger_user_text", "trigger_actee"), ("trigger_room_text", "trigger_observer"),
            ("end_user_text", "end_actee"), ("end_room_text", "end_observer"),
        ],
        "spells.golden": [],
        "quests.golden": [
            ("playermessage", "actor"), ("roommessage", "observer"),
            ("send_text", "actor"), ("room_text", "observer"),
        ],
        "crafting.golden": [
            ("success_room_message", "success_observer"), ("failure_room_message", "failure_observer"),
            ("success_message", "success_actor"), ("failure_message", "failure_actor"),
        ],
    },
    "position": {
        "position_control.golden": [
            ("attacker", "actor"), ("target", "actee"), ("room", "observer"),
            ("self", "actor"),
        ],
    },
}


def golden_at(ref, name):
    out = subprocess.run(["git", "show", "%s:%s/%s" % (ref, GOLDEN_DIR, name)],
                         capture_output=True, text=True)
    if out.returncode != 0:
        raise SystemExit("cannot read %s at %s: %s" % (name, ref, out.stderr.strip()))
    return out.stdout


def translate(text, pairs):
    for old, new in pairs:
        # Labels appear in the row key ("|todefender =>") and inside values
        # ("todefender="). Both are word-bounded.
        text = re.sub(r"\b%s\b" % re.escape(old), new, text)
    return text


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", required=True, help="git ref holding the pre-rename goldens")
    ap.add_argument("--group", required=True, choices=sorted(GROUPS))
    args = ap.parse_args()

    failures = 0
    for name, pairs in GROUPS[args.group].items():
        old = translate(golden_at(args.base, name), pairs)
        with open("%s/%s" % (GOLDEN_DIR, name), "r", encoding="utf-8", newline="") as fh:
            new = fh.read()
        if old == new:
            print("%s: identical after label translation" % name)
            continue
        failures += 1
        old_lines, new_lines = old.splitlines(), new.splitlines()
        for i in range(max(len(old_lines), len(new_lines))):
            o = old_lines[i] if i < len(old_lines) else "<missing>"
            n = new_lines[i] if i < len(new_lines) else "<missing>"
            if o != n:
                print("%s: first difference at line %d" % (name, i + 1))
                print("  translated old: %s" % o)
                print("  new:            %s" % n)
                break
    print("goldens differing beyond labels: %d" % failures)
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
```

- [ ] **Step 2: Prove the checker can fail**

Run it now, before any rename, with an empty translation for one group:

Run: `python tools/messaging_role_key_check.py --base HEAD --group grapple`
Expected: it reports `messaging_grapple.golden: first difference` for every `controller` row, because the working tree has NOT been renamed yet, so translating the old file makes it differ. That proves the comparison is live rather than vacuously equal.

Then, after Task 5's rename, the same command must report `identical after label translation`.

- [ ] **Step 3: Commit**

```bash
git add tools/messaging_role_key_check.py docs/README.md
git commit -m "chore(messaging): checker that a role-key rename touched labels only

Byte-identity cannot prove a rename, because a rename legitimately changes
golden labels. This translates the pre-rename golden's labels and demands
equality with the new one, so a swapped role still shows as a diff. The
alternative, re-recording the goldens, would bake a swap in invisibly.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

(Index the tool in `docs/README.md` with the other live scripts, per the new-files rule.)

---

## Task 4: Rename the pooled combat stores

**Files:**
- Modify: `internal/items/defensive_messages.go:47-51`, `internal/items/attack_messages.go:34-45`
- Modify: `internal/combat/taunt_messages.go:24-28`
- Data: `_datafiles/world/dogmud/{defense-messages,combat-messages,taunt-messages}`, `_datafiles/world/default/combat-messages`
- Modify: `messaging_surface_guard_test.go:142-144,242-243`

- [ ] **Step 1: Record the base ref**

```bash
git rev-parse HEAD > /tmp/m4b-base.txt
```

Every proof in this task compares against that ref.

- [ ] **Step 2: Extend the rewrite tool with role-key groups**

`tools/messaging_token_rewrite.py` already rewrites text tokens. Add a parallel `KEY_GROUPS` table and a `--keys <group>` mode that rewrites YAML KEYS only, anchored so it cannot touch prose: the pattern is a line whose trimmed form starts with `<key>:` or `- <key>:`.

```python
KEY_GROUPS = {
    "combat": [
        (os.path.join(W, "defense-messages"), [
            ("toattacker", "actor"), ("todefender", "actee"), ("toroom", "observer"),
        ]),
        (os.path.join(W, "combat-messages"), [
            ("toattackerroom", "observer"), ("todefenderroom", "remote_observer"),
            ("toattacker", "actor"), ("todefender", "actee"), ("toroom", "observer"),
        ]),
        (os.path.join(DEFAULT_W, "combat-messages"), [
            ("toattackerroom", "observer"), ("todefenderroom", "remote_observer"),
            ("toattacker", "actor"), ("todefender", "actee"), ("toroom", "observer"),
        ]),
        (os.path.join(W, "taunt-messages"), [
            ("toattacker", "actor"), ("todefender", "actee"), ("toroom", "observer"),
        ]),
    ],
}
```

🔴 **Order matters and is load-bearing:** `toattackerroom` must be rewritten before `toattacker`, or it becomes `actorroom`. The list is ordered longest-first and the tool must preserve that order, not sort the table.

⚠️ **`observer` appears twice in the combat-messages table** (`toattackerroom` and `toroom` both map to it) because a `together` group's room line and a `separate` group's attacker-room line are the same audience. That is correct, not a typo.

- [ ] **Step 3: Dry run, then read the diff before writing**

Run: `python tools/messaging_token_rewrite.py --keys combat --dry-run | tail -1`
Expected: a file and key count. Spot-read three files from the list and confirm only key lines would change.

- [ ] **Step 4: Change the struct tags**

```go
type DefenseTogetherMessages struct {
	ToAttacker MessageOptions `yaml:"actor,omitempty"`
	ToDefender MessageOptions `yaml:"actee,omitempty"`
	ToRoom     MessageOptions `yaml:"observer,omitempty"`
}
```

Same for `TogetherMessages`, `SeparateMessages` (whose `ToAttackerRoom` becomes `observer` and `ToDefenderRoom` becomes `remote_observer`) and `TauntMessages`. **Go field names stay as they are in this task**; renaming fields as well would bury the tag change in hundreds of lines. Field renames are optional follow-up work, not part of the proof.

- [ ] **Step 5: Rewrite the data and run the proof**

```bash
python tools/messaging_token_rewrite.py --keys combat
go test ./internal/narration/ -run TestSnapshotStores
python tools/messaging_role_key_check.py --base $(cat /tmp/m4b-base.txt) --group combat
```

Expected: the snapshot test PASSES (the goldens are regenerated only if the test writes them, which it does not: it compares), and the checker prints `identical after label translation` for all three goldens and `goldens differing beyond labels: 0`.

🪤 If `TestSnapshotStores` fails with a label mismatch, that is expected until the snapshot builder emits the new labels. The builder reads the authored key names from the struct tags; confirm where each row key string comes from (`snapshot_test.go:422,442,515-517,546-548,591-593`) and update those literals to the new spellings. The checker is what proves those edits are label-only.

- [ ] **Step 6: Update the surface registry**

`messaging_surface_guard_test.go` entries at `:142-144` and the role set at `:242-243` name the old spellings. Rename the keys, keep each `Reason` string, and add to each a note that the spelling changed in M4b with the old name, so the history is not lost.

Run: `go test . -run TestEveryTextSurfaceIsRegistered`
Expected: PASS.

- [ ] **Step 7: Full check and commit**

```bash
go test ./...
gofmt -l internal/ modules/
git add internal/items internal/combat/taunt_messages.go messaging_surface_guard_test.go internal/narration/snapshot_test.go internal/narration/testdata/stores tools/messaging_token_rewrite.py _datafiles/world/dogmud/defense-messages _datafiles/world/dogmud/combat-messages _datafiles/world/dogmud/taunt-messages _datafiles/world/default/combat-messages
git commit -m "refactor(messaging): combat, defence and taunt on the canonical role keys

toattacker/todefender/toroom become actor/actee/observer, and the ranged
separate shape's second room line becomes remote_observer, which is what
the core has always called that audience.

Proved with tools/messaging_role_key_check.py: the pre-rename goldens,
with labels translated, equal the new ones byte for byte. Re-recording
them would have hidden a swapped role.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Rename the grapple store

**Files:**
- Modify: `internal/grapplemessaging/loader.go:22-26,35-39`
- Data: `_datafiles/world/{dogmud,default}/messaging/grapple_outcomes.yaml`
- Modify: `messaging_surface_guard_test.go:159-161`

Grapple is the store where two authored vocabularies collapse into one: outcome triads say `controller`/`controlled`/`observers`, gradients say `self`/`partner`/`observers`, and both mean actor/actee/observer.

- [ ] **Step 1: Add the grapple key group to the tool**

```python
    "grapple": [
        (os.path.join(W, "messaging"), [
            ("controller", "actor"), ("controlled", "actee"), ("observers", "observer"),
            ("self", "actor"), ("partner", "actee"),
        ]),
        (os.path.join(DEFAULT_W, "messaging"), [
            ("controller", "actor"), ("controlled", "actee"), ("observers", "observer"),
            ("self", "actor"), ("partner", "actee"),
        ]),
    ],
```

⚠️ **Check the prefix relationships before trusting the order.** `controller` and `controlled` share ten characters but neither is a prefix of the other, so their order is free. Keep the table longest-first anyway, because the next table added to it may not be so lucky.

⚠️ **The key rewrite must be anchored to key lines.** `self` and `room` are common English words that appear in grapple prose ("you control the position yourself"). An unanchored replace would corrupt authored text. This is why Task 4 Step 2 anchors on `<key>:` at the start of a trimmed line.

- [ ] **Step 2: Struct tags**

```go
type TemplateTriad struct {
	Controller []string `yaml:"actor"`
	Controlled []string `yaml:"actee"`
	Observers  []string `yaml:"observer"`
}

type GradientTriad struct {
	Self      []string `yaml:"actor"`
	Partner   []string `yaml:"actee"`
	Observers []string `yaml:"observer"`
}
```

Both structs now carry identical tags, which is the point: one vocabulary, two authored shapes collapsed.

- [ ] **Step 3: Rewrite, test, prove**

```bash
python tools/messaging_token_rewrite.py --keys grapple
go test ./internal/narration/ -run TestSnapshotStores ./internal/grapplemessaging/ ./internal/hooks/
python tools/messaging_role_key_check.py --base $(cat /tmp/m4b-base.txt) --group grapple
```

Expected: tests PASS, checker prints `identical after label translation`.

- [ ] **Step 4: Update the registry and commit**

Rename the `controller`, `controlled` and `observers` entries at `messaging_surface_guard_test.go:159-161`, keeping their Reason text plus the old spelling.

```bash
go test .
git add internal/grapplemessaging messaging_surface_guard_test.go internal/narration/snapshot_test.go internal/narration/testdata/stores/messaging_grapple.golden _datafiles/world/dogmud/messaging _datafiles/world/default/messaging
git commit -m "refactor(messaging): grapple on the canonical role keys

Its two authored vocabularies, controller/controlled/observers for
outcomes and self/partner/observers for gradients, were always the same
three audiences. Both structs now carry the same tags.

Proved by label translation against the pre-rename golden.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Rename the Kind B stores

**Files:**
- Modify: `internal/conditions/conditionspec.go:180-185`, `internal/spells/spells.go`, `internal/quests/triggers.go`, `internal/quests/quests.go`, `internal/crafting/crafting.go`
- Data: `_datafiles/world/dogmud/{conditions,spells,quests,recipes}`, `_datafiles/world/default/{conditions,spells,quests}`
- Modify: `messaging_surface_guard_test.go` entries at `:76`, `:90`, `:110`, `:117` and their siblings

These stores key by PHASE and role together (`start_user_text`), so the canonical form keeps the phase and replaces the role half:

| Store | Old | New |
|---|---|---|
| conditions | `start_user_text`, `start_room_text` | `start_actee`, `start_observer` |
| conditions | `trigger_user_text`, `trigger_room_text` | `trigger_actee`, `trigger_observer` |
| conditions | `end_user_text`, `end_room_text` | `end_actee`, `end_observer` |
| spells | `cast_user_text`, `cast_room_text` | `cast_actor`, `cast_observer` |
| spells | `wait_user_text`, `wait_room_text` | `wait_actor`, `wait_observer` |
| spells | `magic_user_text`, `magic_room_text` | `magic_actor`, `magic_observer` |
| quests actions | `send_text`, `room_text` | `actor`, `observer` |
| quests rewards | `playermessage`, `roommessage` | `actor`, `observer` |
| crafting | `success_message`, `success_room_message` | `success_actor`, `success_observer` |
| crafting | `failure_message`, `failure_room_message` | `failure_actor`, `failure_observer` |

🔑 **Conditions takes `actee`, not `actor`**, because the holder is the one a condition happens to. That asymmetry is the whole reason this store had an inverted token before M4a, and the key rename must not quietly "fix" it into `actor`.

- [ ] **Step 1: Add the kindb key group, ordered longest-first**

`success_room_message` before `success_message`, `failure_room_message` before `failure_message`, `start_room_text` before `start_user_text` (not a prefix, but keep the habit), and `room_text` before `send_text` has no ordering interaction. Verify each pair for prefix relationships before running.

- [ ] **Step 2: Struct tags, one store at a time, building after each**

```go
	StartUserText  string `yaml:"start_actee,omitempty"`
	StartRoomText  string `yaml:"start_observer,omitempty"`
```

and the equivalents. Run `go build ./...` after each store so a mistake is attributed to one file.

- [ ] **Step 3: Rewrite, test, prove**

```bash
python tools/messaging_token_rewrite.py --keys kindb
go test ./internal/narration/ -run TestSnapshotStores
python tools/messaging_role_key_check.py --base $(cat /tmp/m4b-base.txt) --group kindb
```

Expected: PASS and `goldens differing beyond labels: 0`.

- [ ] **Step 4: Check the OTHER readers of these keys**

Quest YAML keys are also read by the quest editor and by GMCP. Grep for each old spelling across Go and any JSON or JS under `modules/` and `_datafiles/html`, and fix every hit:

Run: `grep -rn "playermessage\|roommessage\|send_text\|room_text\|start_user_text\|success_room_message" --include=*.go --include=*.js --include=*.json . | grep -v _test`
Expected after fixes: only historical mentions in comments and context.md, each one deliberate.

🔴 **This is the step most likely to find something the plan did not predict.** M4a found three such cases. Treat a hit in a quest editor or a GMCP payload as a real consumer, not as documentation.

- [ ] **Step 5: Registry, full suite, commit**

```bash
go test ./...
git add internal/conditions internal/spells internal/quests internal/crafting messaging_surface_guard_test.go internal/narration/snapshot_test.go internal/narration/testdata/stores _datafiles/world/dogmud/conditions _datafiles/world/dogmud/spells _datafiles/world/dogmud/quests _datafiles/world/dogmud/recipes _datafiles/world/default/conditions _datafiles/world/default/spells _datafiles/world/default/quests
git commit -m "refactor(messaging): conditions, spells, quests and crafting on canonical role keys

Phase-plus-role keys keep their phase and canonicalise their role half:
start_user_text becomes start_actee, success_room_message becomes
success_observer, and so on.

Conditions takes actee rather than actor, because a condition happens to
its holder. That is the same asymmetry M4a's token flip recorded, and it
is deliberate.

Proved by label translation against the pre-rename goldens.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Rename position_control

**Files:**
- Modify: `internal/hooks/Position_Messaging.go:37-41` and the gradient and stamina structs
- Data: `_datafiles/world/{dogmud,default}/messaging/position_control.yaml`

- [ ] **Step 1: Decide the two dead blocks, and say which you chose**

`gradient_messages` and `transition_messages` are authored text no Go struct reads (found during M4a). Now that the file sits under the world tree, pick one:

- **A (recommended): keep and rename them with the rest.** They are authored prose, the live gradient store next door has the same shape, and deleting authored text in a refactor slice is the kind of loss the arc exists to prevent.
- **B: delete them**, and record the deletion in the M6 content ledger so the text can be re-authored deliberately.

Do NOT leave them half-renamed. Whichever you choose, state it in the commit body.

- [ ] **Step 2: Rename the keys and tags**

`attacker` -> `actor`, `target` -> `actee`, `room` -> `observer`, and the stamina block's `self` -> `actor`, `room` -> `observer`.

⚠️ The anchored key rewrite matters most here: `room` and `self` appear inside this file's prose.

- [ ] **Step 3: Prove and commit**

```bash
python tools/messaging_token_rewrite.py --keys position
go test ./internal/narration/ -run TestSnapshotStores ./internal/hooks/
python tools/messaging_role_key_check.py --base $(cat /tmp/m4b-base.txt) --group position
go test ./...
git add internal/hooks/Position_Messaging.go internal/narration/snapshot_test.go internal/narration/testdata/stores/position_control.golden _datafiles/world/dogmud/messaging/position_control.yaml _datafiles/world/default/messaging/position_control.yaml
git commit -m "refactor(messaging): position_control on the canonical role keys

The tenth store's three authored role vocabularies finish collapsing:
attacker/target/room and the stamina block's self/room all become
actor/actee/observer.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: The no-old-spelling guard

**Files:**
- Modify: `shipped_narration_data_guard_test.go`

- [ ] **Step 1: Add the guard**

```go
// TestNoLegacyRoleKeysInShippedData fails when a narration YAML file still
// spells a role the old way. The renames of M4b-1 are only durable if a new
// authored file cannot reintroduce the old vocabulary, and a store whose
// struct no longer declares the tag would load that file SILENTLY EMPTY.
func TestNoLegacyRoleKeysInShippedData(t *testing.T) {
	legacy := []string{
		"toattacker", "todefender", "toroom", "toattackerroom", "todefenderroom",
		"controller", "controlled", "observers", "partner",
		"start_user_text", "start_room_text", "trigger_user_text", "trigger_room_text",
		"end_user_text", "end_room_text",
		"cast_user_text", "cast_room_text", "wait_user_text", "wait_room_text",
		"magic_user_text", "magic_room_text",
		"playermessage", "roommessage", "send_text",
		"success_message", "success_room_message", "failure_message", "failure_room_message",
	}
	// walk _datafiles/world/{dogmud,default}, read each .yaml, and report any
	// line whose trimmed form starts with "<legacy>:" or "- <legacy>:"
}
```

⚠️ `room_text` is deliberately NOT in that list: quest YAML uses `room_text` as an action key, and M4b-2 may still be reading it. Confirm its state before adding it, rather than assuming.

- [ ] **Step 2: Prove it fails**

Reintroduce `toattacker:` into one shipped combat-messages file, run the guard, confirm it names the file and line, then `git checkout -- <path>`.

- [ ] **Step 3: Commit**

```bash
git add shipped_narration_data_guard_test.go
git commit -m "test: fail the build on a legacy role key in shipped data

A store whose struct no longer declares the old tag loads such a file
silently empty, which is the failure this slice exists to make
impossible. Proven capable of failing.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: Docs, gate, PR

- [ ] **Step 1: context.md sweep**

Every package whose keys changed: `internal/items`, `internal/combat`, `internal/grapplemessaging`, `internal/conditions`, `internal/spells`, `internal/quests`, `internal/crafting`, `internal/hooks`, `internal/narration`. Name the new keys, record the old ones as history, and verify every symbol exists.

Run: `python tools/context_md_audit.py`

- [ ] **Step 2: Schema docs**

`docs/schemas/` documents authorable YAML for builders. Grep it for every old role key and update, because these files are what a human writes new content against.

Run: `grep -rn "toattacker\|todefender\|toroom\|start_user_text\|playermessage\|success_message" docs/schemas/`

- [ ] **Step 3: PATCH_NOTES**

One short entry, player-facing framing, no raw numbers, no dashes. This slice changes no wording: say so plainly.

- [ ] **Step 4: Full gate**

```bash
gofmt -l internal/ modules/
go build ./...
go test ./...
python tools/messaging_token_rewrite.py --check
```

Run the last one standalone.

- [ ] **Step 5: Boot check**

The recipe from Task 2 Step 6. Exit 124, zero panics, one `Server Ready`.

- [ ] **Step 6: Hand back to the controller**

Do NOT push or open the PR. Report the branch diff size (`git diff --stat master...HEAD | tail -1`) and the full gate output.

---

## Self-review against the spec

| Spec item (M4b, this PR's half) | Task |
|---|---|
| Shipped-data tests land FIRST | 1 |
| Two-tier loader policy by cost of silence | 2 |
| Grapple and position_control load at boot from the configured path | 2 |
| position_control moves under the world tree | 2 |
| Canonical role keys in YAML and struct tags | 4, 5, 6, 7 |
| `remote_observer` for ranged combat's second room | 4 |
| Conditions keeps actee | 6 |
| Proof is label-translated equality, never `-update` | 3, and every rename task |
| M0 surface registry updated | 4, 5, 6 |
| A legacy key cannot come back | 8 |
| One `DefenseType`, spell defence convention, core-drain | **M4b-2, separate plan** |
| The M0 guard walking `_datafiles/messages` | dissolved by Task 2: the file moves under the tree the guard already walks |
