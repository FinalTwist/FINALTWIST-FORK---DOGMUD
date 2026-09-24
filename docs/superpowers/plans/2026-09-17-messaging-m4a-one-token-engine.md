# Messaging M4a: one token engine, one token vocabulary

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Leave exactly one token substitution engine (`narration.Substitute`) and one name-token vocabulary (`{actor}`, `{actee}`, `{actor_plain}`, `{actee_plain}`) across every store, with every golden byte-identical.

**Architecture:** Each store's Go token map and its shipped YAML flip in the SAME commit, because a map that emits `{actor}` against YAML that still says `{source}` renders the literal token. Stores are grouped by which Go vocabulary they share: Kind B stores share `textutil.TokenContext`; combat, defence and taunt share `items.TokenName`; grapple and position_control each have their own. The three leftover engines are deleted as their store flips.

**Tech Stack:** Go 1.23, `gopkg.in/yaml.v3`, snapshot goldens under `internal/narration/testdata/stores/`, AST guards at repo root (package `main`).

**Spec:** `docs/superpowers/specs/2026-09-17-messaging-m4-flip-design.md` (M4a section).

---

## Facts this plan was written against (master `ee8477064`, 2026-09-17)

| Fact | Value | Source |
|---|---|---|
| Core engine | unexported `substitute(s, tokens)` | `internal/narration/render.go:135` |
| Kind B vocabulary | `TokenContext{SourceName, SourcePlainName, TargetName, TargetPlainName}` -> 4 keys | `internal/textutil/tokens.go:9-27` |
| Kind B call sites | 23 `textutil.TokenContext{` literals in 18 files | grep, non-test |
| Item vocabulary | `items.TokenName`, 16 constants | `internal/items/itemspec.go:200-216` |
| Leftover engine 1 | `ItemMessage.SetTokenValue` | `internal/items/attack_messages.go:57`, called `internal/combat/combat_helpers.go:1724-1748` |
| Leftover engine 2 | `grapplemessaging.RenderTemplate` | `internal/grapplemessaging/render.go:14-16`, called `internal/hooks/Position_GrappleTick.go:559` |
| Leftover engine 3 | `hooks.substitute` | `internal/hooks/Position_Messaging.go:142-152` |
| Taunt alias map | `{source}`/`{target}` aliased in Go, file unchanged | `internal/combat/taunt_messages.go:147-157` |
| Defence role mapping | `ToAttacker`->Actor, `ToDefender`->Actee, `ToRoom`->Observer | `internal/items/defensive_messages.go:143-160` |
| Conditions inversion | holder line is the Actee; `{source}` is the holder | `internal/conditions/narration.go:22-38` |
| Goldens | 14 files | `internal/narration/testdata/stores/` |
| position_control | 3 name vocabularies: `{attacker}`/`{target}` (submission), `{Controller}`/`{Controlled}` (gradient, transition), `{Character}` (stamina) | `_datafiles/messages/position_control.yaml` |
| position_control golden | DOES NOT EXIST | `testdata/stores/` listing |
| Prompt tokens | `{target}` also exists in the PROMPT engine (regex + switch), player-configured, unrelated | `internal/users/userrecord.prompt.go:32,530` |
| Shipped counts | `{source}` 4540, `{target}` 2871, `{attacker}` 186, `{defender}` 414, `{controllerName}` 239, `{controlledName}` 172, `{source_plain}` 17 | `grep -rho` under `_datafiles/world/dogmud` |

⚠️ **Do NOT rewrite `_datafiles/world/dogmud/templates/`, `_datafiles/world/dogmud/users/`, or any prompt string.** Their `{target}` is the prompt vocabulary, not narration.

⚠️ **The rewrite script must never use `yaml.dump` or any load-and-dump round trip.** It edits text lines only, writes to a temp file and `os.replace`s it. A dump destroys quoting and comment headers (M3 item 9 trap).

---

## File structure

| File | Responsibility | Task |
|---|---|---|
| `tools/messaging_token_rewrite.py` | the one rewrite tool: per-store table, `--dry-run`, `--check` | 1 |
| `internal/narration/render.go` | exported `Substitute`, canonical token constants | 2 |
| `internal/textutil/tokens.go` | Kind B vocabulary, renamed fields | 3 |
| `internal/items/itemspec.go` | `TokenName` constants renamed | 4 |
| `internal/items/attack_messages.go` | `SetTokenValue` deleted | 4 |
| `internal/combat/combat_helpers.go` | fallback and feint paths use `narration.Substitute` | 4 |
| `internal/combat/taunt_messages.go` | alias map deleted | 4 |
| `internal/grapplemessaging/render.go` | `RenderTemplate` deleted | 5 |
| `internal/hooks/Position_Messaging.go` | local `substitute` deleted | 6 |
| `internal/narration/snapshot_test.go` | position_control golden | 6 |
| `token_engine_guard_test.go` (repo root) | one-engine AST guard | 7 |

---

## Task 1: The rewrite tool, with a dry run

**Files:**
- Create: `tools/messaging_token_rewrite.py`

- [ ] **Step 1: Confirm the suite is green before touching anything**

Run: `go test ./internal/narration/... ./internal/items/... ./internal/combat/... ./internal/conditions/... ./internal/spells/... ./internal/quests/... ./internal/crafting/... ./internal/grapplemessaging/... ./internal/hooks/...`
Expected: all PASS. If anything is red at HEAD, stop and report; do not start on a red tree.

- [ ] **Step 2: Write the tool**

```python
#!/usr/bin/env python3
"""Rewrite shipped narration name tokens to the canonical vocabulary (M4a).

Text-line edits only. NEVER yaml.load/yaml.dump: a round trip destroys
quoting and comment headers. Writes to <file>.tmp then os.replace, so a
crash cannot truncate a shipped file.

Usage:
  python tools/messaging_token_rewrite.py --group kindb --dry-run
  python tools/messaging_token_rewrite.py --group kindb
  python tools/messaging_token_rewrite.py --check      # non-zero if any old token remains
"""
import argparse
import os
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
W = os.path.join(ROOT, "_datafiles", "world", "dogmud")
MSG = os.path.join(ROOT, "_datafiles", "messages")

# Per (store, key), never global: the same spelling means different roles in
# different stores. conditions' {source} is the HOLDER, which is the actee.
GROUPS = {
    "kindb": [
        (os.path.join(W, "conditions"), {
            "{source}": "{actee}", "{source_plain}": "{actee_plain}",
            "{target}": "{actor}", "{target_plain}": "{actor_plain}",
        }),
        (os.path.join(W, "spells"), {
            "{source}": "{actor}", "{source_plain}": "{actor_plain}",
            "{target}": "{actee}", "{target_plain}": "{actee_plain}",
        }),
        (os.path.join(W, "quests"), {
            "{source}": "{actor}", "{source_plain}": "{actor_plain}",
        }),
        (os.path.join(W, "recipes"), {
            "{source}": "{actor}", "{source_plain}": "{actor_plain}",
        }),
    ],
    "items": [
        (os.path.join(W, "combat-messages"), {
            "{source}": "{actor}", "{target}": "{actee}",
            "{sourcetype}": "{actortype}", "{targettype}": "{acteetype}",
        }),
        (os.path.join(W, "defense-messages"), {
            "{attacker}": "{actor}", "{defender}": "{actee}",
        }),
        (os.path.join(W, "taunt-messages"), {
            "{source}": "{actor}", "{target}": "{actee}",
            "{sourcetype}": "{actortype}", "{targettype}": "{acteetype}",
        }),
    ],
    "grapple": [
        (os.path.join(W, "messaging"), {
            "{controllerName}": "{actor}", "{controlledName}": "{actee}",
        }),
    ],
    "position": [
        (MSG, {
            "{attacker}": "{actor}", "{target}": "{actee}",
            "{Controller}": "{actor}", "{Controlled}": "{actee}",
            "{Character}": "{actor}",
        }),
    ],
}


def files_under(path):
    if os.path.isfile(path):
        return [path]
    out = []
    for dirpath, _, names in os.walk(path):
        for n in sorted(names):
            if n.endswith(".yaml") or n.endswith(".yml"):
                out.append(os.path.join(dirpath, n))
    return sorted(out)


def rewrite(path, table, dry_run):
    with open(path, "r", encoding="utf-8", newline="") as fh:
        original = fh.read()
    updated = original
    for old, new in sorted(table.items(), key=lambda kv: -len(kv[0])):
        updated = updated.replace(old, new)
    if updated == original:
        return 0
    hits = sum(original.count(old) for old in table)
    if not dry_run:
        tmp = path + ".tmp"
        with open(tmp, "w", encoding="utf-8", newline="") as fh:
            fh.write(updated)
        os.replace(tmp, path)
    return hits


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--group", choices=sorted(GROUPS))
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--check", action="store_true")
    args = ap.parse_args()

    if args.check:
        stale = []
        for group in GROUPS.values():
            for path, table in group:
                for f in files_under(path):
                    with open(f, "r", encoding="utf-8", newline="") as fh:
                        body = fh.read()
                    for old in table:
                        if old in body:
                            stale.append("%s: %s x%d" % (f, old, body.count(old)))
        for row in stale:
            print(row)
        print("stale token occurrences: %d" % len(stale))
        return 1 if stale else 0

    if not args.group:
        ap.error("--group is required unless --check is given")
    total, touched = 0, 0
    for path, table in GROUPS[args.group]:
        for f in files_under(path):
            hits = rewrite(f, table, args.dry_run)
            if hits:
                touched += 1
                total += hits
                print("%s%s: %d" % ("[dry-run] " if args.dry_run else "", f, hits))
    print("files touched: %d, tokens rewritten: %d" % (touched, total))
    return 0


if __name__ == "__main__":
    sys.exit(main())
```

- [ ] **Step 3: Dry-run every group and record the counts**

Run:
```bash
for g in kindb items grapple position; do python tools/messaging_token_rewrite.py --group $g --dry-run | tail -1; done
```
Expected: four `files touched: N, tokens rewritten: M` lines, nothing written. The `items` group should report the largest count (combat-messages alone holds 4368 `{source}` and 2812 `{target}`).

- [ ] **Step 4: Confirm the tool cannot reach the excluded trees**

Run: `python tools/messaging_token_rewrite.py --group items --dry-run | grep -c "templates/\|/users/"`
Expected: `0`. Those hold prompt tokens, not narration.

- [ ] **Step 5: Commit**

```bash
git add tools/messaging_token_rewrite.py
git commit -m "chore(messaging): M4a token rewrite tool, dry-run only

Text-line rewrites with a per-store table, because the same spelling
means different roles in different stores: conditions' {source} is the
holder, which is the actee, while every other store's {source} is the
actor. No yaml round trip, so quoting and comment headers survive.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Export the core engine and name the vocabulary

**Files:**
- Modify: `internal/narration/render.go:135`
- Test: `internal/narration/render_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/narration/render_test.go`:

```go
func TestSubstituteIsExportedAndOnePass(t *testing.T) {
	got := Substitute("{actor} hits {actee}", map[string]string{
		TokenActor: "Alice",
		TokenActee: "{actor}",
	})
	if got != "Alice hits {actor}" {
		t.Fatalf("Substitute() = %q, want %q", got, "Alice hits {actor}")
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/narration/ -run TestSubstituteIsExportedAndOnePass`
Expected: FAIL, `undefined: Substitute`.

- [ ] **Step 3: Export the function and add the constants**

In `internal/narration/render.go`, rename `substitute` to `Substitute`, keep its doc comment, and add above it:

```go
// The canonical name-token vocabulary (messaging arc M4a). Every store's
// shipped YAML spells names these four ways and no other. Event tokens
// ({weapon}, {bodypart}, {itemname} and the rest) stay store-specific: they
// name what happened, not who it happened to.
const (
	TokenActor      = "{actor}"
	TokenActee      = "{actee}"
	TokenActorPlain = "{actor_plain}"
	TokenActeePlain = "{actee_plain}"
)
```

Update the one internal caller at `render.go:113` (`return substitute(...)` becomes `return Substitute(...)`).

- [ ] **Step 4: Run the test and the package**

Run: `go test ./internal/narration/`
Expected: PASS, goldens included.

- [ ] **Step 5: Commit**

```bash
git add internal/narration/render.go internal/narration/render_test.go
git commit -m "refactor(narration): export Substitute and name the canonical tokens

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Kind B stores flip (conditions, spells, quests, crafting)

**Files:**
- Modify: `internal/textutil/tokens.go:9-27`
- Modify: the 23 `textutil.TokenContext{` literals (compiler lists them)
- Data: `_datafiles/world/dogmud/{conditions,spells,quests,recipes}`

⚠️ **The Go change and the YAML rewrite are ONE commit.** Splitting them renders a literal `{source}` to players in between.

- [ ] **Step 1: Rename the struct fields and emit canonical keys**

In `internal/textutil/tokens.go`:

```go
// TokenContext holds the names for substitution in YAML text fields.
//
// Actor is the one acting; Actee is the one acted upon. The conditions store
// passes the HOLDER as the actee, because a condition happens to its holder
// (see internal/conditions/narration.go).
type TokenContext struct {
	ActorName      string // ANSI-tagged display name
	ActorPlainName string // Plain name (for possessives)
	ActeeName      string // ANSI-tagged display name (empty if none)
	ActeePlainName string // Plain name (empty if none)
}

// Tokens is the vocabulary as the narration core takes it. All four keys are
// always present, so an absent actee substitutes to an empty string.
func (ctx TokenContext) Tokens() map[string]string {
	return map[string]string{
		narration.TokenActor:      ctx.ActorName,
		narration.TokenActee:      ctx.ActeeName,
		narration.TokenActorPlain: ctx.ActorPlainName,
		narration.TokenActeePlain: ctx.ActeePlainName,
	}
}
```

- [ ] **Step 2: Let the compiler enumerate the call sites**

Run: `go build ./... 2>&1 | head -40`
Expected: errors at each of the 23 literals naming `SourceName`, `SourcePlainName`, `TargetName`, `TargetPlainName`.

Fix 22 of the 23 mechanically: `SourceName` -> `ActorName`, `SourcePlainName` -> `ActorPlainName`, `TargetName` -> `ActeeName`, `TargetPlainName` -> `ActeePlainName`.

**One site is not mechanical.** `internal/hooks/Condition_ApplyConditions.go:150-153` passes the condition HOLDER as `SourceName`, and the holder is the actee (its own comment at `:151` says so, and `internal/conditions/narration.go:35` puts the holder's line in the Actee slot). That site becomes:

```go
			roles := conditionInfo.Narrate(conditions.PhaseStart, charName, charPlainName)
```

after Step 2a changes that method's signature, which is what stops any call site choosing the wrong slot.

Every other conditions call site the compiler names (`NewTurn_PruneConditions.go`, `NewRound_MobRoundTick.go`, `NewRound_UserRoundTick.go`, `internal/actions/sleep.go`) takes the same two-string form.

🔴 **Why the slot cannot be left to callers.** `TestSnapshotStores` calls each store's `Narration`/`Narrate` directly with its own token context; it never runs a hook call site. A site that fills `ActorName` where the store reads `{actee}` renders an EMPTY name in play and a perfectly green golden. Step 2a removes the choice rather than testing for it.

- [ ] **Step 2a: Make the mistake impossible instead of testing for it**

A test cannot cheaply cover 23 call sites, so the slot stops being a caller's choice. `ConditionSpec.Narrate` already knows the holder is the actee; let it own the mapping. In `internal/conditions/narration.go`:

```go
// Narrate renders one phase for its audiences. It takes the HOLDER, not a
// token context: the holder is always the actee (a condition happens to them),
// and building the context here means no call site can put the name in the
// wrong slot. Actor stays empty until events.Condition carries a caster (M6).
func (b *ConditionSpec) Narrate(p Phase, holderName, holderPlainName string) narration.Roles {
	return textutil.Narrate(b.Narration(p), textutil.TokenContext{
		ActeeName:      holderName,
		ActeePlainName: holderPlainName,
	})
}
```

Every conditions call site then passes two strings and cannot choose a slot. The compiler finds them all. Leave the spells, quests and crafting signatures alone: their name is the actor, which is `TokenContext`'s obvious default, and their call sites already read that way.

- [ ] **Step 3: Rewrite the shipped YAML**

Run: `python tools/messaging_token_rewrite.py --group kindb`
Expected: a per-file list ending in `files touched: N, tokens rewritten: M`, matching Task 1 Step 3's dry-run numbers.

- [ ] **Step 4: Run the goldens**

Run: `go test ./internal/narration/ -run TestSnapshotStores`
Expected: PASS with no diff. `conditions.golden`, `spells.golden`, `quests.golden` and `crafting.golden` must be byte-identical.

If a golden diffs, the cause is a mismatched pair (a YAML file rewritten while its Go field was not, or the reverse). **Do not run `-update`.**

- [ ] **Step 5: Confirm nothing else changed**

Run: `git status --short _datafiles | grep -cv "conditions/\|spells/\|quests/\|recipes/"`
Expected: `0`.

- [ ] **Step 6: Commit**

```bash
git add internal/textutil/tokens.go internal/hooks internal/actions internal/behaviortree internal/crafting internal/justice internal/mobcommands internal/questengine internal/usercommands _datafiles/world/dogmud/conditions _datafiles/world/dogmud/spells _datafiles/world/dogmud/quests _datafiles/world/dogmud/recipes
git commit -m "refactor(messaging): Kind B stores on the canonical token vocabulary

TokenContext fields and shipped YAML flip together: a Go map emitting
{actor} against YAML that still says {source} renders the literal token.
Conditions' holder keeps its Actee slot and its token is now spelled
{actee}, ending the one store where {source} named the acted-upon.

Goldens byte-identical; -update forbidden in this slice.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Items vocabulary flip and the first engine deleted

**Files:**
- Modify: `internal/items/itemspec.go:200-216`
- Modify: `internal/items/attack_messages.go:57` (delete `SetTokenValue`)
- Modify: `internal/combat/combat_helpers.go:1724-1748`, `:1912-1930` (feint literals)
- Modify: `internal/combat/taunt_messages.go:147-157` (delete the alias map)
- Data: `_datafiles/world/dogmud/{combat-messages,defense-messages,taunt-messages}`

- [ ] **Step 1: Rename the token constants**

In `internal/items/itemspec.go`, rename the four name tokens and their type tokens, leaving every event token alone:

```go
	TokenActor     TokenName = `{actor}`
	TokenActee     TokenName = `{actee}`
	TokenActorType TokenName = `{actortype}`
	TokenActeeType TokenName = `{acteetype}`
```

Delete `TokenSource`, `TokenTarget`, `TokenSourceType`, `TokenTargetType`, `TokenAttacker`, `TokenDefender`. The defence store's `{attacker}`/`{defender}` become `TokenActor`/`TokenActee`: same two people, one spelling.

- [ ] **Step 2: Fix the call sites the compiler names**

Run: `go build ./... 2>&1 | head -40`

Map each: `TokenSource`/`TokenAttacker` -> `TokenActor`; `TokenTarget`/`TokenDefender` -> `TokenActee`; `TokenSourceType` -> `TokenActorType`; `TokenTargetType` -> `TokenActeeType`. The combat token map at `internal/combat/combat_helpers.go:1627-1630` and the defence map at `:1321-1330` are the two large ones.

- [ ] **Step 3: Replace the second engine at its two call sites**

In `internal/combat/combat_helpers.go`, the fallback loop at `:1724-1748` becomes one call each. Delete the per-token loop and write:

```go
	if !rendered {
		tokens := items.TokenStrings(tokenReplacements)
		toAttackerMsg = items.ItemMessage(narration.Substitute(string(toAttackerMsg), tokens))
		toDefenderMsg = items.ItemMessage(narration.Substitute(string(toDefenderMsg), tokens))
		toAttackerRoomMsg = items.ItemMessage(narration.Substitute(string(toAttackerRoomMsg), tokens))
		if len(string(toDefenderRoomMsg)) > 0 {
			toDefenderRoomMsg = items.ItemMessage(narration.Substitute(string(toDefenderRoomMsg), tokens))
		}
	}
```

`internal/items/defensive_messages.go:175` already has a private `tokenStrings(map[TokenName]string) map[string]string`. Export it as `items.TokenStrings` (rename in place, update its two in-package callers) rather than writing a second copy.

Do the same for the feint block that follows in the same function: build the token map once, call `narration.Substitute` once per line, instead of eight `SetTokenValue` calls.

- [ ] **Step 4: Delete `SetTokenValue`**

Remove the method at `internal/items/attack_messages.go:57-59`. `go build ./...` must then report no remaining callers.

- [ ] **Step 5: Rewrite the feint literals in Go**

At `internal/combat/combat_helpers.go:1912-1930`, the feint strings are Go literals holding `{target}`, `{targettype}`, `{source}`, `{sourcetype}`. Rewrite them to `{actee}`, `{acteetype}`, `{actor}`, `{actortype}`. Update the doc comment at `:1912` to name the canonical tokens. (These literals move to YAML in M4e; here they only change vocabulary.)

- [ ] **Step 6: Delete the taunt alias map**

In `internal/combat/taunt_messages.go:147-157`, the map becomes canonical keys and the comment explaining the aliasing goes away:

```go
	roles := narration.Render(msgs.variants(), map[string]string{
		narration.TokenActor: source,
		narration.TokenActee: target,
		`{actortype}`:        sourceType,
		`{acteetype}`:        targetType,
		`{damage}`:           damageDesc,
	}, pick)
```

- [ ] **Step 7: Rewrite the shipped YAML**

Run: `python tools/messaging_token_rewrite.py --group items`
Expected: the counts from Task 1's dry run.

- [ ] **Step 8: Run the goldens and the combat suite**

Run: `go test ./internal/narration/ ./internal/items/ ./internal/combat/`
Expected: PASS. `combat_messages.golden`, `defense_messages.golden`, `taunt_messages.golden` byte-identical.

🪤 `internal/combat` tests are flaky at roughly 2.3% through the attack fumble path. A single red run naming a fumble is not a regression; re-run the named test before investigating.

- [ ] **Step 9: Commit**

```bash
git add internal/items internal/combat _datafiles/world/dogmud/combat-messages _datafiles/world/dogmud/defense-messages _datafiles/world/dogmud/taunt-messages
git commit -m "refactor(messaging): items vocabulary canonical, SetTokenValue deleted

The second token engine is gone: combat's fallback and feint paths now
substitute through narration.Substitute, the same door every store uses.
Taunt's alias map is deleted because rhetoric.yaml now says {actor} and
{actee} itself.

Goldens byte-identical.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Grapple flip and the second engine deleted

**Files:**
- Modify: `internal/grapplemessaging/render.go:10-17`
- Modify: `internal/hooks/Position_GrappleTick.go:559`
- Data: `_datafiles/world/dogmud/messaging/grapple_outcomes.yaml`

- [ ] **Step 1: Point the one caller at the core**

`internal/hooks/Position_GrappleTick.go:559` calls `grapplemessaging.RenderTemplate(tpl, controllerName, controlledName)`. Replace with:

```go
	line := narration.Substitute(tpl, map[string]string{
		narration.TokenActor: controllerName,
		narration.TokenActee: controlledName,
	})
```

- [ ] **Step 2: Delete `RenderTemplate`**

Remove `internal/grapplemessaging/render.go:10-17`. Run `go build ./...`; expected: no remaining callers.

- [ ] **Step 3: Rewrite the shipped YAML**

Run: `python tools/messaging_token_rewrite.py --group grapple`
Expected: 411 tokens across `grapple_outcomes.yaml`.

- [ ] **Step 4: Run the golden**

Run: `go test ./internal/narration/ -run TestSnapshotStores ./internal/grapplemessaging/ ./internal/hooks/`
Expected: PASS, `messaging_grapple.golden` byte-identical.

⚠️ `messaging_grapple.golden`'s header says it renders "with fixed stand-ins Controller/Controlled" through `RenderTemplate` (`internal/narration/snapshot_test.go:630`). Update that comment to name `narration.Substitute`, and re-run: a comment line inside the golden is generated, so the file content changes by exactly that line. Record the one-line diff in the commit body so a reviewer is not surprised by a golden that is not byte-identical.

- [ ] **Step 5: Commit**

```bash
git add internal/grapplemessaging internal/hooks/Position_GrappleTick.go internal/narration/snapshot_test.go internal/narration/testdata/stores/messaging_grapple.golden _datafiles/world/dogmud/messaging/grapple_outcomes.yaml
git commit -m "refactor(messaging): grapple on the canonical tokens, RenderTemplate deleted

Third engine gone. The golden's generated header line changes to name
narration.Substitute; every rendered row is byte-identical.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: position_control, netted first, then flipped

This store has no golden and three name vocabularies. The net comes first, per the arc's rule that the previous slice's net never reaches the next slice's target.

**Files:**
- Modify: `internal/narration/snapshot_test.go`
- Create: `internal/narration/testdata/stores/position_control.golden`
- Modify: `internal/hooks/Position_Messaging.go:120-152`
- Data: `_datafiles/messages/position_control.yaml`

- [ ] **Step 1: Record the golden from PRE-change code**

Add a section to `internal/narration/snapshot_test.go` that loads `_datafiles/messages/position_control.yaml` and renders every line with fixed stand-ins, keyed by block and key:

```go
// position_control is the tenth store and the only one outside the world
// tree. Recorded from PRE-migration code (M4a), so the flip has something to
// be identical against.
func writePositionControlSnapshot(t *testing.T, b *strings.Builder) {
	t.Helper()
	tpl := loadPositionControlForSnapshot(t) // reads the YAML directly
	subs := map[string]string{
		"position": "side control", "Character": "Actorius",
		"Controller": "Actorius", "Controlled": "Acteeus",
		"attacker": "Actorius", "target": "Acteeus",
		"old_position": "guard", "new_position": "side control",
	}
	for _, row := range positionControlRows(tpl) { // block|key|role => text
		fmt.Fprintf(b, "%s => %s\n", row.Key, renderPre(row.Text, subs))
	}
}
```

`renderPre` is the CURRENT `hooks.substitute` logic copied into the test (a `{key}` loop), because the golden must record what production does today, not what the core will do tomorrow.

- [ ] **Step 2: Generate it and read it**

Run: `go test ./internal/narration/ -run TestSnapshotStores -update`
Expected: `position_control.golden` created, roughly 57 rows. **This is the one place `-update` is allowed in M4a**, because the file does not exist yet. Read the generated file and confirm the rows look like real sentences with the stand-ins filled in.

- [ ] **Step 3: Prove the golden can fail**

Temporarily change one line in `_datafiles/messages/position_control.yaml` ("You settle into a dominating {position}." -> "You settle into a dominating {position}!"). Run: `go test ./internal/narration/ -run TestSnapshotStores`. Expected: FAIL naming that row. Revert the YAML edit and re-run: PASS.

- [ ] **Step 4: Commit the net**

```bash
git add internal/narration/snapshot_test.go internal/narration/testdata/stores/position_control.golden
git commit -m "test(narration): golden for position_control, the tenth store

Recorded from pre-migration code. The arc never listed this store: it
lives outside _datafiles/world/dogmud, which is the only tree the M0
surface guard walks, so it had no snapshot and no guard.

Proven capable of failing.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 5: Flip the store**

In `internal/hooks/Position_Messaging.go`, `substitutionsForCharacter` returns canonical keys:

```go
func substitutionsForCharacter(c *characters.Character) map[string]string {
	partner := resolvePartner(c)
	partnerName := ""
	if partner != nil {
		partnerName = partner.Name
	}
	// The controller is the actor and the controlled is the actee, whichever
	// of the two this character is: the templates name the sides, not the
	// reader. {Character}'s one template (the stamina warning) is about this
	// character, so it is the actor there.
	actor, actee := c.Name, partnerName
	if !c.IsController() {
		actor, actee = partnerName, c.Name
	}
	return map[string]string{
		"{position}":           c.Position.State().String(),
		narration.TokenActor:   actor,
		narration.TokenActee:   actee,
	}
}
```

⚠️ **The stamina warning is the one asymmetric case.** Its room line is `{Character} looks exhausted in the {position}.` and `{Character}` is always `c`, even when `c` is the controlled side. Under the mapping above a controlled `c` would render the CONTROLLER's name there. So the stamina warning must pass its own map with the actor set to `c`:

```go
	staminaSubs := substitutionsForCharacter(c)
	staminaSubs[narration.TokenActor] = c.Name
```

The golden proves this: if the mapping is wrong, the stamina row changes.

Delete the local `substitute` (`:142-152`) and call `narration.Substitute` at the three sites (`:112-113` and the submission senders).

- [ ] **Step 6: Rewrite the shipped YAML**

Run: `python tools/messaging_token_rewrite.py --group position`
Expected: every `{attacker}`, `{target}`, `{Controller}`, `{Controlled}`, `{Character}` replaced; `{position}`, `{old_position}`, `{new_position}` untouched.

- [ ] **Step 7: Re-render the golden without `-update`**

Update the test's `renderPre` to call `narration.Substitute` and its stand-in map to canonical keys, then run: `go test ./internal/narration/ -run TestSnapshotStores ./internal/hooks/`
Expected: PASS, `position_control.golden` byte-identical to the file committed in Step 4.

- [ ] **Step 8: Commit**

```bash
git add internal/hooks/Position_Messaging.go internal/narration/snapshot_test.go _datafiles/messages/position_control.yaml
git commit -m "refactor(messaging): position_control on the canonical tokens

Its three name vocabularies ({attacker}/{target}, {Controller}/
{Controlled}, {Character}) collapse to {actor}/{actee}. The stamina
warning keeps its own actor because that line is about the character it
is sent to, not about the controller; the golden pins it.

Golden byte-identical against the pre-migration recording.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Unknown tokens fail at boot for event stores

**Files:**
- Modify: `internal/conditions/conditionspec.go:296-300`, `internal/spells/spells.go:326-330`
- Modify: `internal/crafting/crafting.go` (gains validation)
- Test: `internal/crafting/crafting_test.go`

Ambient stores (weather, gossip, tips) keep warning until M4b sets the two-tier loader policy.

- [ ] **Step 1: Write the failing test**

In `internal/crafting/crafting_test.go`:

```go
func TestRecipeValidateRejectsUnknownToken(t *testing.T) {
	r := RecipeSpec{RecipeId: "test-recipe", SuccessMessage: "You forge {sorce} a blade."}
	problems := r.ValidateNarrationTokens()
	if len(problems) == 0 {
		t.Fatal("expected {sorce} to be reported as an unknown token")
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test ./internal/crafting/ -run TestRecipeValidateRejectsUnknownToken`
Expected: FAIL, `undefined: ValidateNarrationTokens`.

- [ ] **Step 3: Implement**

```go
// ValidateNarrationTokens reports unknown tokens in this recipe's four
// narration fields. Crafting was the only store with no token check at all
// (messaging arc M4a), so a typo shipped silently and rendered raw to the
// player.
func (r RecipeSpec) ValidateNarrationTokens() []string {
	var problems []string
	for _, text := range []string{
		r.SuccessMessage, r.SuccessRoomMessage,
		r.FailureMessage, r.FailureRoomMessage,
	} {
		problems = append(problems, textutil.ValidateTokens(text)...)
	}
	return problems
}
```

Call it from the recipe loader's existing validation and return an error, which the loader already turns into a boot panic.

- [ ] **Step 4: Promote conditions and spells from warn to fail**

At `internal/conditions/conditionspec.go:298` and `internal/spells/spells.go:328`, the `textutil.ValidateTokens` results are logged through `mudlog.Warn`. Return them as validation errors instead, so the existing loader panics. Keep the message text; only the severity changes.

- [ ] **Step 5: Run the boot check**

```bash
git worktree add --detach C:/tmp/dogmud-boot-check HEAD
cp _datafiles/config.yaml C:/tmp/dogmud-boot-check/_datafiles/config.yaml
cd C:/tmp/dogmud-boot-check && go build -o boot-check.exe .
timeout 180 ./boot-check.exe > boot.log 2>&1
grep -cE "^panic:|goroutine [0-9]+ \[running\]|runtime error" boot.log   # want 0
grep -c "Server Ready" boot.log                                          # want 1
```

Exit code 124 is success. Then `git worktree remove --force C:/tmp/dogmud-boot-check`.

If the boot panics, shipped data has an unknown token that was being warned about and ignored. **That is a real find, not a blocker to route around:** fix the data, and record the token in the commit body.

- [ ] **Step 6: Commit**

```bash
git add internal/crafting internal/conditions/conditionspec.go internal/spells/spells.go
git commit -m "feat(messaging): unknown tokens fail the boot for event stores

Crafting had no token validation at all, so a typo rendered raw to the
player. Conditions and spells warned and carried on. Ambient stores keep
warning until M4b sets the two-tier loader policy.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: The one-engine guard

**Files:**
- Create: `token_engine_guard_test.go` (repo root, package `main`)

- [ ] **Step 1: Write the guard**

```go
// TestNoSecondTokenEngine fails when any file outside internal/narration
// substitutes a brace token by hand. The messaging arc promised this guard in
// its "How we know it worked" list and never built it; three engines survived
// M0 to M3 as a result (items.SetTokenValue, grapplemessaging.RenderTemplate,
// hooks.substitute), each rendering the same stores through different code.
//
// It matches a call to strings.Replace, strings.ReplaceAll or
// strings.NewReplacer where any argument is a string literal containing "{".
// The prompt engine (internal/users/userrecord.prompt.go) is NOT matched: it
// is a regexp plus a switch over a different vocabulary, and it is not
// narration.
func TestNoSecondTokenEngine(t *testing.T) {
	var offenders []string
	for _, root := range messagingSurfaceGoRoots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel := filepath.ToSlash(path)
			if strings.HasPrefix(rel, "internal/narration/") {
				return nil
			}
			fset := token.NewFileSet()
			file, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				return perr
			}
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok || pkg.Name != "strings" {
					return true
				}
				switch sel.Sel.Name {
				case "Replace", "ReplaceAll", "NewReplacer":
				default:
					return true
				}
				for _, arg := range call.Args {
					lit, ok := arg.(*ast.BasicLit)
					if ok && lit.Kind == token.STRING && strings.Contains(lit.Value, "{") {
						offenders = append(offenders, fmt.Sprintf("%s:%d: %s.%s with a brace-token literal",
							rel, fset.Position(lit.Pos()).Line, pkg.Name, sel.Sel.Name))
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	if len(offenders) > 0 {
		t.Errorf("token substitution outside internal/narration (use narration.Substitute):\n  %s",
			strings.Join(offenders, "\n  "))
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test . -run TestNoSecondTokenEngine`
Expected: PASS, because Tasks 4, 5 and 6 deleted all three engines.

- [ ] **Step 3: Prove it can fail**

Add to a scratch file `internal/hooks/zz_sabotage.go`:

```go
package hooks

import "strings"

func sabotageTokenEngine(s, name string) string {
	return strings.ReplaceAll(s, "{actor}", name)
}
```

Run: `go vet ./internal/hooks/` (must compile: a sabotage that does not compile proves nothing), then `go test . -run TestNoSecondTokenEngine`.
Expected: FAIL naming `internal/hooks/zz_sabotage.go` and its line. Delete the file and re-run: PASS.

- [ ] **Step 4: Commit**

```bash
git add token_engine_guard_test.go
git commit -m "test: guard that narration.Substitute is the only token engine

Proven capable of failing with a sabotage verified to compile first.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: Docs, then ship

**Files:**
- Modify: `internal/narration/context.md`, `internal/textutil/context.md` (if present), `internal/items/context.md`, `internal/hooks/context.md`
- Modify: `docs/PATCH_NOTES.md`

- [ ] **Step 1: Update each context.md**

Name `narration.Substitute` and the four canonical tokens; delete every mention of `SetTokenValue`, `RenderTemplate` and the local `substitute`; record that `position_control.yaml` is a store with a golden. Verify every symbol you name exists:

Run: `Select-String -Path internal\narration\*.go -Pattern '^(func|type|const|var)\s'`

- [ ] **Step 2: Add a PATCH_NOTES entry**

Under today's date, one line, player-facing framing, no raw numbers, no em dashes. This slice changes no wording, so the entry says so: internal messaging cleanup with no visible change.

- [ ] **Step 3: Full gate**

```bash
gofmt -l internal/ modules/          # must print nothing
go build ./...
go test ./...
python tools/messaging_token_rewrite.py --check    # must print "stale token occurrences: 0"
```

🪤 Run the `--check` line on its own, not chained with `&&`: a zero-match grep-style check exits non-zero and would silently skip whatever followed.

- [ ] **Step 4: Boot check in an isolated worktree**

Same recipe as Task 7 Step 5. Exit code 124 is success.

- [ ] **Step 5: Push and open the PR**

```bash
git push -u origin feature/messaging-m4a-token-engine
gh pr create --repo pruuk/DOGMud --base master --head feature/messaging-m4a-token-engine \
  --title "Messaging M4a: one token engine, one token vocabulary" --fill
gh pr checks <n> --repo pruuk/DOGMud --watch
```

Confirm the URL printed says `pruuk/DOGMud`. The PR body must state: every golden byte-identical except `messaging_grapple.golden`'s generated header line and the new `position_control.golden`; `-update` used exactly once, to create a golden that did not exist.

---

## Self-review against the spec

| Spec item (M4a) | Task |
|---|---|
| `narration.Substitute` the only engine | 2, 4, 5, 6 |
| Three leftover engines deleted | 4 (`SetTokenValue`), 5 (`RenderTemplate`), 6 (`hooks.substitute`) |
| position_control joins the engine | 6 |
| Shipped YAML rewritten by a committed script with a per-store table | 1, 3, 4, 5, 6 |
| conditions `{source}` -> `{actee}` | 1 (table), 3 |
| `_plain` variants follow their base | 1 (table), 3 |
| `TokenContext` fields renamed | 3 |
| Unknown tokens fail at boot for event stores | 7 |
| AST guard, no second engine | 8 |
| M0 surface guard sees `_datafiles/messages` | **deferred to M4b**, where position_control moves under the world tree; the golden added in Task 6 is this slice's cover |
| Goldens byte-identical, `-update` forbidden | 3, 4, 5, 6 (one documented exception: creating `position_control.golden`) |
