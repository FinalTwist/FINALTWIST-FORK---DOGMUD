# M4e PR 1a: Mob Special Moves to YAML + the canSeeInDark Migration

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task.
> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the thirteen `internal/mobcommands` special-move files' narration
wording out of Go string literals into a shipped YAML store, and retire
`canSeeInDark` across all twenty files that reference it in favour of the
`ParticipantSight` + `HideNames` pair the pipeline already runs.

**Architecture:** A new event-keyed narration store
(`_datafiles/world/dogmud/narration/special-moves/<verb>.yaml`) loaded at boot
by a new leaf package `internal/movenarration`, modelled exactly on the taunt
store (`internal/combat/taunt_messages.go`). Go keeps every branching decision
and names an event by typed key; Go keeps no wording. The darkness branches in
the mob files are **deleted rather than migrated**, because `messaging.SendTrio`
already hides each party's name per reader by that reader's sight verdict.

**Tech Stack:** Go 1.x, `gopkg.in/yaml.v3` via `internal/fileloader`,
`internal/narration` (`Variants`/`Render`/`ValidateVariants`), `internal/messaging`
(`SendTrio`/`ParticipantSight`/`HideNames`).

---

## Facts verified against source, 2026-09-21, master `18c9124a8`

Every row was read from source today. The command that produced each is named.
Nothing here is recalled from a prior session.

| Fact | Value | Source |
|---|---|---|
| `canSeeInDark` definition | `func canSeeInDark(u *users.UserRecord, room *rooms.Room) bool { return room.GetVisibility() >= 1 \|\| u.Character.HasFlagFromAnySource(conditions.NightVision) }` | `internal/mobcommands/darkness.go:43-46`, `cat` |
| Second definition, not an import | byte-identical unexported twin | `internal/usercommands/skill_move_defence.go:93-95` |
| Total references / files | **30 refs across 20 files** (19 `mobcommands` + 1 `usercommands`) | `grep -rn "canSeeInDark" --include=*.go . \| wc -l` |
| Executable call sites | **24** (30 minus 2 definitions and 4 comments) | same grep, classified by reading; corrected from 25 by the Task 1 inventory |
| Third hand-rolled darkness check | `sendAudioRoomText`, its own comment says it should collapse, names **M5** not M4 | `internal/mobcommands/darkness.go:19-42`, `internal/mobcommands/go.go:96-98` |
| `messaging.ParticipantSight` | `func ParticipantSight(observer *characters.Character, room RoomVisibility) SightDecision` | `internal/messaging/predicates.go:51` |
| `messaging.HideNames` | `func HideNames(text string, names []string, d SightDecision) string` | `internal/messaging/hidenames.go:38` |
| `SightDecision` order | `SightFull = iota` (0), `SightShapes` (1), `SightNone` (2) — **best to worst, never compare with `>=`** | `internal/messaging/pipeline.go:32-38` |
| `HideNames` on a tagged name | consumes the identity tag with the name, emits `<ansi fg="combat-anon">something\|a figure</ansi>`, capitalized at sentence start | `internal/messaging/hidenames.go:58-90` |
| 🔑 `SendTrio` already hides names | actor line hides `ActeeName`, actee line hides `ActorName`, both judged by that reader's `ParticipantSight`; observer and remote-observer lines hide both, judged per observer by the room | `internal/messaging/trio.go:108-135` |
| 🔑 Mob files already pass the names | `aud := messaging.Audience{ActorName: mobName, ..., ActeeName: target.Name, Room: room}` | `internal/mobcommands/kick.go:52-59` |
| Mob special-move files | **13**: attack, bash, charge, drain, gore, grapple, hamstring, kick, maul, pounce, rake, shoot, throttle, trip (minus attack — see note) | `m2FrozenFiles`, `messaging_surface_guard_test.go:1508` |
| Frozen-list files, both packages | **25** (12 `usercommands` + 13 `mobcommands`), two lists agreeing 1:1 | `messaging_surface_guard_test.go:1508`, `m2_routing_guard_test.go:70` |
| Mob backtick literals | **184** raw, **182** real narration rows (2 are an empty-string comparison and a backtick inside a comment) | Task 1 inventory, `docs/superpowers/audits/2026-09-21-m4e1-site-inventory.md` |
| Mob variant pools | **zero** — no `util.Rand`, no `[]string{}` narration pool in any mob file | `grep -l 'util.Rand\|\[\]string{' internal/mobcommands/<13 files>` → no hits |
| Mob Trio shape | `Actor` is always `messaging.NoLine` (a mob has no client); `Actee` + `Observer` carry the text; `shoot.go` additionally uses `RemoteObserver` | read across the 13 files |
| Format verbs | `%s` only. **`%d` appears zero times** in all 25 files; damage is pre-formatted by `combat.GetDamageDescription()` | `grep -o '%d'` → 0 hits |
| `narration.Variants` contract | every non-empty role must hold the **same** number of variants | `internal/narration/render.go:18-27` |
| `Variants.Len()` on ragged pools | returns **0**, and every caller treats 0 as "render nothing" | `internal/narration/render.go:50-60` |
| `ValidateVariants` on ragged pools | returns an error naming both counts; store loaders panic on it | `internal/narration/render.go:240-255` |
| `narration.Render` | `func Render(v Variants, tokens map[string]string, pick Picker, indexOverride ...int) Roles` | `internal/narration/render.go` |
| The one substitution entry point | `narration.Substitute(s string, tokens map[string]string) string` | `internal/narration/render.go` |
| Canonical tokens | `TokenActor`, `TokenActee`, `TokenActorPlain`, `TokenActeePlain` | `internal/narration/render.go:128-133` |
| Model store loader | `func LoadTauntMessageFiles()` at `internal/combat/taunt_messages.go:106`, called at **boot** from `main.go:1945`, **panics** on failure (event tier) | `cat`, `grep -n LoadTauntMessageFiles main.go` |
| Model store path idiom | `dir := string(configs.GetFilePathsConfig().DataFiles) + "/taunt-messages"` then `fileloader.LoadAllFlatFiles[...]` | `internal/combat/taunt_messages.go:109-110` |
| Shipped-data guard | `TestShippedNarrationDataValidates`, hardcodes `shippedWorldRoot = "_datafiles/world/dogmud"` because a **test binary never reads `config.yaml`** | `shipped_narration_data_guard_test.go:79` |
| Golden walker | `TestSnapshotStores`, one subtest per store | `internal/narration/snapshot_test.go:809-855` |
| Golden regeneration | `go test ./internal/narration/... -run TestSnapshotStores -update` | `internal/narration/snapshot_test.go:113,203` |
| One-path guard | `TestNoRawEventsMessageOutsidePipeline` + `rawEventsMessageAllowed` (5 entries) | `raw_events_message_guard_test.go:14-26` |
| `internal/actions` grapple prose | `DisarmResult.Message/TargetMsg/RoomMessage`, `CritFailure.Message/TargetMessage/RoomMessage` consumed by both grapple twins | `internal/usercommands/grapple.go:138-165`, `internal/mobcommands/grapple.go:79-112` |
| Shared defence lines | `moveDefenceLines(...)` in `skill_move_defence.go` feeds `defence.ToRoom/ToAttacker/ToDefender` in 12 of 13 mob files | read across the 13 files |
| 🐛 `usercommands/shoot.go:426` | anonymizes on `result.IsSneaking` **only**, never consults darkness; the mob twin ORs in `!canSeeInDark` | `internal/usercommands/shoot.go:426-429` vs `internal/mobcommands/shoot.go:85` |

**Note on `attack.go`:** `internal/mobcommands/attack.go:94` calls `canSeeInDark`
for an engagement notice but is not one of the 13 frozen special-move files. It
is in scope for the **sight** half of this PR (Task 9) and out of scope for the
**wording** half.

### Owner rulings, 2026-09-21 (do not relitigate)

1. **Ragged pools are squared by authoring the missing lines**, not by padding
   or truncating. That work lands in **PR 1b**, not here: every mob file has
   zero pools, so PR 1a is unaffected.
2. **`usercommands/shoot.go`'s missing darkness check is a defect and gets
   fixed**, in its own commit, flagged as a behaviour change (Task 10).
3. **Grapple's prose in `internal/actions` is pulled in** rather than deferred,
   so one verb's wording does not sit half in YAML and half in Go (Task 8).
4. **PR 1a is mob-first**: the 13 mob files plus the **entire** `canSeeInDark`
   migration including both `skill_move_defence.go` twins. PR 1b takes the 12
   player files and their authoring. This honours the "open each sight file
   once" ruling, because 1a carries all of the sight work.

### The central finding this plan is built on

🔑 **The mob files' darkness branches are a redundant reimplementation of what
the pipeline already does, and they are worse at it.**

`internal/mobcommands/kick.go:121-135` is the pattern, verbatim:

```go
		default: // KickStandard
			hitActee := messaging.NoLine
			if targetUser != nil {
				if canSee {
					hitActee = messaging.Say(messaging.CategoryKick, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> kicks you hard! (<ansi fg="damage">%s</ansi>)`, mobName, dmgDesc))
				} else {
					hitActee = messaging.Say(messaging.CategoryKick, fmt.Sprintf(`Something kicks you hard! (<ansi fg="damage">%s</ansi>)`, dmgDesc))
				}
			}
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.NoLine,
				Actee: hitActee,
				Observer: messaging.Say(messaging.CategoryKick,
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> kicks <ansi fg="username">%s</ansi>!`, mobName, target.Name)),
			}, aud)
```

The `Audience` at `:52-59` already carries `ActorName: mobName`, and
`SendTrio` already runs `HideNames(text, []string{aud.ActorName}, sight)` on the
actee line. So the `canSee` branch produces by hand, at one tier, what the
pipeline produces for free at three tiers.

**Consequences for this plan:**

- The `canSee` local and the `Something ...` twin literal are **deleted** in all
  eleven files that use that shape. Only the named line is authored into YAML.
- This is a **deliberate, improving, player-visible diff**, exactly the M4d PR 2
  shape. An infrared reader who reads `Something kicks you hard!` today will
  read `A figure kicks you hard!` after. A fully blind reader reads
  `Something kicks you hard!` as before, now inside an `<ansi fg="combat-anon">`
  tag. **Accepted side effect, same as PR 2's:** the anonymised word renders in
  the anon identity colour for everyone.
- The observer line was **never** affected by `canSee` and needs no change of
  behaviour: `SendTrio` already routes it through
  `SendTextVisualHidingNames`.

### Risks

| Risk | Answer |
|---|---|
| A verb's YAML event key is referenced but absent, so a move narrates nothing | Task 12's `TestMoveEventKeysAgree` asserts referenced-set == authored-set in **both** directions, and is sabotage-proven to fail each way |
| A golden covers the store but not the path production calls (the M4a/M4c lesson, hit three times) | Task 6's net renders through the **call-site helper** each file uses, not through the store API, and Task 7 sabotage-proves it |
| The net passes because it compares a thing to itself | Task 6 step 2 requires the net to go RED against the unmigrated tree before any file is migrated |
| A migrated file still holds prose | Task 12's per-file literal ban replaces the shrinking `m2FrozenFiles` entries |
| The deliberate darkness diff is mistaken for a regression in review | Task 11 records before/after lines verbatim for all three sight tiers |
| `-race` cannot run locally (no cgo on this machine) | Gate runs without `-race`; CI runs it under the 900s cap raised by #150 |

---

## File structure

**Create:**
- `internal/movenarration/store.go` — the store type, loader, validation
- `internal/movenarration/render.go` — the lookup/render API call sites use
- `internal/movenarration/context.md` — required for every new package
- `internal/movenarration/store_test.go` — unit tests for loader and render
- `_datafiles/world/dogmud/narration/special-moves/<verb>.yaml` — 13 files
- `internal/narration/testdata/stores/special_moves.golden` — the store golden
- `tools/move_narration_net.py` — the byte-identity net's literal extractor
- `internal/mobcommands/special_move_net_test.go` — the net's table test

**Modify:**
- `internal/mobcommands/*.go` — 13 special-move files plus `attack.go`,
  `darkness.go`, `go.go`
- `internal/usercommands/skill_move_defence.go`, `internal/usercommands/shoot.go`
- `internal/actions/` — the grapple result-struct literals
- `main.go` — boot-time loader call
- `messaging_surface_guard_test.go`, `m2_routing_guard_test.go` — shrink the
  frozen lists
- `shipped_narration_data_guard_test.go` — add the store's subtest
- `internal/narration/snapshot_test.go` — add the golden subtest
- `docs/superpowers/audits/messaging-m6-content-ledger.md` — deferred rows
- `docs/README.md` — index the new files

---

## Task 1: Re-derive the site list and freeze it as data

The spec requires Task 1 to re-derive the surface with a broader search than the
audit's candidate definition, because a grep can only find the name you guessed.

**Files:**
- Create: `docs/superpowers/audits/2026-09-21-m4e1-site-inventory.md`

- [ ] **Step 1: Enumerate every narration literal in the 13 mob files**

Run, from the repo root:

```bash
for f in attack bash charge drain gore grapple hamstring kick maul pounce rake shoot throttle trip; do
  p="internal/mobcommands/$f.go"
  [ -f "$p" ] || continue
  printf '=== %s: %s backtick literals, %s SendTrio calls, %s canSeeInDark refs\n' \
    "$p" \
    "$(grep -oE '`[^`]*`' "$p" | wc -l)" \
    "$(grep -c 'messaging.SendTrio' "$p")" \
    "$(grep -c 'canSeeInDark' "$p")"
done
```

Note: `grep -c` exits 1 on zero matches. These run inside `$( )` in a `printf`,
never in an `&&` chain, so a zero count prints `0` and does not abort the loop.

- [ ] **Step 2: Enumerate the sight surface independently**

```bash
grep -rn "canSeeInDark" --include=*.go . | tee /tmp/sight-sites.txt | wc -l
grep -rn "messaging.Anonymize(" --include=*.go internal/mobcommands internal/usercommands
grep -rn "sendAudioRoomText(" --include=*.go internal/mobcommands | wc -l
```

Expected: 30 `canSeeInDark` references (2 definitions, 3 comments, 25 call
sites). If the number differs from 30, **stop and reconcile before continuing** —
the plan's scope is wrong and the remaining tasks inherit the error.

- [ ] **Step 3: Verify the negative**

The claim "no mob special-move file uses a variant pool" must be proven capable
of failing. Run the search, then run it again against a file that *does* pool:

```bash
grep -l 'util.Rand' internal/mobcommands/kick.go internal/mobcommands/gore.go ; echo "exit=$?"
grep -l 'util.Rand' internal/usercommands/kick.go ; echo "exit=$?"
```

Expected: first command prints nothing, `exit=1`. Second prints
`internal/usercommands/kick.go`, `exit=0`. The second proves the first could
have found something.

- [ ] **Step 4: Write the inventory document**

Record one row per literal: file, line, outcome branch, role (actee/observer/
remote_observer), the format string verbatim, and the argument list. This table
is the input to Task 6's net and the authority for the YAML in Task 3.

- [ ] **Step 5: Commit**

```bash
git add docs/superpowers/audits/2026-09-21-m4e1-site-inventory.md
git commit -m "docs(m4e-1): inventory the mob special-move narration and sight surface

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: The store package, test first

**Files:**
- Create: `internal/movenarration/store.go`, `internal/movenarration/store_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/movenarration/store_test.go`:

```go
package movenarration

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/narration"
)

func TestEventVariantsRoundTrip(t *testing.T) {
	g := &MoveNarrationGroup{
		MoveId: "kick",
		Events: map[EventKey]*EventMessages{
			"hit": {
				Actor:    []string{`You kick {actee} hard! ({damage})`},
				Actee:    []string{`{actor} kicks you hard! ({damage})`},
				Observer: []string{`{actor} kicks {actee}!`},
			},
		},
	}
	if err := g.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	v, ok := g.Variants("hit")
	if !ok {
		t.Fatal(`Variants("hit") not found`)
	}
	if v.Len() != 1 {
		t.Fatalf("Len = %d, want 1", v.Len())
	}
	roles := narration.Render(v, map[string]string{
		narration.TokenActor: `<ansi fg="mobname">Goblin</ansi>`,
		narration.TokenActee: `<ansi fg="username">Kesh</ansi>`,
		TokenDamage:          `<ansi fg="damage">a solid hit</ansi>`,
	}, narration.FirstPicker)
	want := `<ansi fg="mobname">Goblin</ansi> kicks you hard! (<ansi fg="damage">a solid hit</ansi>)`
	if roles.Actee != want {
		t.Errorf("Actee =\n%q\nwant\n%q", roles.Actee, want)
	}
}

func TestValidateRejectsRaggedPools(t *testing.T) {
	g := &MoveNarrationGroup{
		MoveId: "kick",
		Events: map[EventKey]*EventMessages{
			"hit": {
				Actee:    []string{`a`, `b`},
				Observer: []string{`c`},
			},
		},
	}
	if err := g.Validate(); err == nil {
		t.Fatal("Validate accepted ragged pools; it must reject them")
	}
}

func TestVariantsMissingEventReportsNotFound(t *testing.T) {
	g := &MoveNarrationGroup{MoveId: "kick", Events: map[EventKey]*EventMessages{}}
	if _, ok := g.Variants("nope"); ok {
		t.Fatal(`Variants("nope") reported found for an absent event`)
	}
}
```

- [ ] **Step 2: Run it to confirm it fails**

```bash
go test ./internal/movenarration/ -run TestEventVariantsRoundTrip -v
```

Expected: FAIL — the package does not compile, `undefined: MoveNarrationGroup`.

- [ ] **Step 3: Write the store**

Create `internal/movenarration/store.go`:

```go
// Package movenarration holds the shipped wording for special-move narration:
// the player-side verbs (kick, bash, trip, ...) and their mob twins.
//
// Go decides WHICH event fires and names it by key. Go holds no wording. The
// three audiences of one event are variant-paired by index, which is why
// Validate refuses pools of unequal length: variant N of each role describes
// the same moment.
package movenarration

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/narration"
	"github.com/pkg/errors"
)

// TokenDamage is this store's one event token. Damage always arrives
// pre-formatted as prose from combat.GetDamageDescription; there is no numeric
// verb anywhere in the special-move surface.
const TokenDamage = "{damage}"

// EventKey names one outcome branch of one verb, such as "hit" or "miss".
type EventKey string

// EventMessages is one event as each audience is told it. The yaml keys are the
// arc's canonical role keys.
type EventMessages struct {
	Actor          []string `yaml:"actor"`
	Actee          []string `yaml:"actee"`
	Observer       []string `yaml:"observer"`
	RemoteObserver []string `yaml:"remote_observer"`
}

// MoveNarrationGroup is one verb's file.
type MoveNarrationGroup struct {
	MoveId string                        `yaml:"moveid"`
	Events map[EventKey]*EventMessages   `yaml:"events"`
}

func (g *MoveNarrationGroup) Id() string { return g.MoveId }

func (g *MoveNarrationGroup) Filepath() string {
	return fmt.Sprintf("%s.yaml", g.MoveId)
}

// Validate fails the boot on any malformed event. This store is EVENT tier
// under the arc's two-tier loader policy: silence here means a move lands with
// no narration at all, so it must never be survivable.
func (g *MoveNarrationGroup) Validate() error {
	if g.MoveId == "" {
		return errors.New("moveid is empty")
	}
	if len(g.Events) == 0 {
		return errors.Errorf("move %q declares no events", g.MoveId)
	}
	for key, ev := range g.Events {
		if ev == nil {
			return errors.Errorf("move %q event %q is empty", g.MoveId, key)
		}
		if err := narration.ValidateVariants(ev.variants(), 1); err != nil {
			return errors.Wrapf(err, "move %q event %q", g.MoveId, key)
		}
	}
	return nil
}

func (e *EventMessages) variants() narration.Variants {
	return narration.Variants{
		Actor:         e.Actor,
		Actee:         e.Actee,
		Observer:      e.Observer,
		ActeeObserver: e.RemoteObserver,
	}
}

// Variants returns the event's pools, or ok=false if the verb does not declare
// that event.
func (g *MoveNarrationGroup) Variants(key EventKey) (narration.Variants, bool) {
	ev, ok := g.Events[key]
	if !ok || ev == nil {
		return narration.Variants{}, false
	}
	return ev.variants(), true
}

var loadedMoves map[string]*MoveNarrationGroup

// LoadMoveNarrationFiles loads the store at boot. It panics on any failure,
// matching combat.LoadTauntMessageFiles: this is event narration, not ambient.
func LoadMoveNarrationFiles() {
	dir := string(configs.GetFilePathsConfig().DataFiles) + `/narration/special-moves`
	loaded, err := fileloader.LoadAllFlatFiles[string, *MoveNarrationGroup](dir)
	if err != nil {
		panic(errors.Wrap(err, "loading special-move narration"))
	}
	loadedMoves = loaded
}

// GetMove returns one verb's group, or nil if the store is unloaded or the verb
// is absent.
func GetMove(moveId string) *MoveNarrationGroup {
	if loadedMoves == nil {
		return nil
	}
	return loadedMoves[moveId]
}
```

Note the `ActeeObserver` mapping: `narration.Variants` spells the fourth role
`ActeeObserver` while `messaging.Trio` spells it `RemoteObserver`. They are the
same audience — the defender's room in ranged combat. The yaml key is the arc's
canonical `remote_observer`.

- [ ] **Step 4: Run the tests to confirm they pass**

```bash
go test ./internal/movenarration/ -v
```

Expected: PASS, three tests.

- [ ] **Step 5: Commit**

```bash
git add internal/movenarration/store.go internal/movenarration/store_test.go
git commit -m "feat(m4e-1): the special-move narration store

Event-keyed, role-paired by index, event tier: Validate refuses ragged pools
and the loader panics, because a silent miss here means a move lands with no
narration at all.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Author the thirteen YAML files from the inventory

**Files:**
- Create: `_datafiles/world/dogmud/narration/special-moves/{attack,bash,charge,drain,gore,grapple,hamstring,kick,maul,pounce,rake,shoot,throttle,trip}.yaml`

- [ ] **Step 1: Write `kick.yaml` as the worked example**

Create `_datafiles/world/dogmud/narration/special-moves/kick.yaml`. Each line is
the mob literal from Task 1's inventory with `%s` replaced by the canonical
token that supplied it. The `Actor` role is left empty for now: mobs have no
actor line, and PR 1b authors the player side into the same file.

```yaml
moveid: kick
events:
  standard_hit:
    actee:
      - '{actor} kicks you hard! ({damage})'
    observer:
      - '{actor} kicks {actee}!'
  standard_knockdown:
    actee:
      - '{actor} kicks you hard, knocking you down! ({damage})'
    observer:
      - '{actor} kicks {actee} to the ground!'
  standard_miss:
    actee:
      - '{actor} kicks at you and misses!'
    observer:
      - '{actor} kicks at {actee} and misses!'
```

🔴 **The variant axis goes in the event key.** Task 1 found that `kick` and
`trip` carry an orthogonal variant dimension the outcome key alone does not
capture: kick resolves as stomp, knee or standard, and trip as tailsweep or
trip. A key of `hit` alone would make three different moves collide on one
pool and silently narrate a stomp as a kick.

Spell these `<variant>_<outcome>`: `stomp_hit`, `knee_miss`,
`standard_knockdown`, `tailsweep_hit`. Every other verb has a single variant
and keeps the bare outcome key (`hit`, `partial`, `miss`), so the shared keys
stay shared where they genuinely mean the same moment.

Task 12's bidirectional key guard is what enforces this: a variant whose key
nothing references, or a reference with no authored key, fails the build.

⚠️ **Authoring rules for every file in this task:**
- Copy each line **byte for byte** from the inventory. Task 6's net fails on a
  single changed character. This is a migration, not an edit pass.
- Wrap every value in **single quotes**. A YAML value containing `: ` breaks an
  unquoted scalar, and several lines contain colons.
- Do **not** author the `Something ...` dark twin. It is deleted, not migrated.
- Only the damage token is an event token; it is spelled `{damage}`.

- [ ] **Step 2: Author the remaining twelve files the same way**

One file per verb, keys drawn from that verb's outcome branches in the
inventory. `shoot.yaml` is the only file with a `remote_observer` role.

- [ ] **Step 3: Verify every file parses and validates**

```bash
go run ./tools/validate_move_narration 2>/dev/null || \
  go test ./internal/movenarration/ -run TestShipped -v
```

If neither target exists yet, this step is satisfied by Task 5, which adds the
shipped-data subtest. Run it there and return here only if it fails.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/narration/special-moves/
git commit -m "content(m4e-1): the thirteen mob special-move event files

Wording copied byte for byte from the Go literals inventoried in Task 1. The
darkness twins are deliberately absent: the pipeline hides names per reader.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Wire the loader into boot

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Find where the sibling store loads**

```bash
grep -n "LoadTauntMessageFiles" main.go
```

Expected: one hit around `main.go:1945`.

- [ ] **Step 2: Add the call immediately after it**

```go
	combat.LoadTauntMessageFiles()
	movenarration.LoadMoveNarrationFiles()
```

Add the import `"github.com/GoMudEngine/GoMud/internal/movenarration"` to
`main.go`'s import block.

- [ ] **Step 3: Prove the boot loads it**

```bash
go build -o /tmp/dogmud-boot . && echo BUILD_OK
```

Expected: `BUILD_OK`.

🪤 **A failed boot exits 0.** `main()` recovers, logs `PANIC` plus a stack, and
returns normally. Never check `$?`. The boot check in Task 13 greps for
`Server Ready` and for the `PANIC` line.

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "feat(m4e-1): load the special-move narration store at boot

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: The shipped-data guard and the golden

**Files:**
- Modify: `shipped_narration_data_guard_test.go`, `internal/narration/snapshot_test.go`
- Create: `internal/narration/testdata/stores/special_moves.golden`

- [ ] **Step 1: Add the shipped-data subtest**

In `shipped_narration_data_guard_test.go`, inside
`TestShippedNarrationDataValidates`, add a subtest beside the existing ones:

```go
	t.Run("special-moves", func(t *testing.T) {
		dir := filepath.Join(shippedWorldRoot, "narration", "special-moves")
		groups, err := fileloader.LoadAllFlatFiles[string, *movenarration.MoveNarrationGroup](dir)
		if err != nil {
			t.Fatalf("loading %s: %v", dir, err)
		}
		if len(groups) == 0 {
			t.Fatalf("no special-move files loaded from %s", dir)
		}
		for id, g := range groups {
			if err := g.Validate(); err != nil {
				t.Errorf("move %q: %v", id, err)
			}
		}
	})
```

⚠️ This reads `shippedWorldRoot` directly, **not** `configs.GetFilePathsConfig()`.
A test binary never loads `config.yaml`, so the config would hand back
`_datafiles/world/default`, which does not carry this store.

- [ ] **Step 2: Prove the guard can fail**

Temporarily break one shipped file — make `kick.yaml`'s `hit` event ragged by
deleting one of its two role lines — then run:

```bash
go test . -run TestShippedNarrationDataValidates -v
```

Expected: FAIL naming `move "kick" event "hit"` and both pool counts. **Restore
the file and re-run to confirm it goes green.** A guard not proven capable of
failing is not a guard; this plan has three such proofs and every one is
mandatory.

- [ ] **Step 3: Confirm `fileloader` actually calls `Validate`**

The store's whole failure policy rests on this. Verify it rather than assuming:

```bash
grep -n "Validate()" internal/fileloader/*.go
```

Expected: `LoadAllFlatFiles` invokes `Validate()` on each loaded item and
propagates the error. If it does **not**, `LoadMoveNarrationFiles` must call
`Validate()` itself in a loop over `loaded` before assigning `loadedMoves`, or
every guarantee in Task 2 is vacuous.

- [ ] **Step 4: Add the golden subtest**

In `internal/narration/snapshot_test.go`, beside the existing store subtests in
`TestSnapshotStores`:

```go
	t.Run("special_moves", func(t *testing.T) {
		checkGolden(t, "special_moves.golden", buildSpecialMovesGolden(t))
	})
```

Write `buildSpecialMovesGolden` in the same file:

```go
func buildSpecialMovesGolden(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(shippedWorldRootForSnapshot, "narration", "special-moves")
	groups, err := fileloader.LoadAllFlatFiles[string, *movenarration.MoveNarrationGroup](dir)
	if err != nil {
		t.Fatalf("loading %s: %v", dir, err)
	}
	verbs := make([]string, 0, len(groups))
	for id := range groups {
		verbs = append(verbs, id)
	}
	sort.Strings(verbs)

	tokens := map[string]string{
		narration.TokenActor:      `<ansi fg="mobname">ACTOR</ansi>`,
		narration.TokenActee:      `<ansi fg="username">ACTEE</ansi>`,
		narration.TokenActorPlain: `ACTOR`,
		narration.TokenActeePlain: `ACTEE`,
		movenarration.TokenDamage: `DAMAGE`,
	}

	var b strings.Builder
	for _, verb := range verbs {
		g := groups[verb]
		keys := make([]string, 0, len(g.Events))
		for k := range g.Events {
			keys = append(keys, string(k))
		}
		sort.Strings(keys)
		for _, key := range keys {
			v, ok := g.Variants(movenarration.EventKey(key))
			if !ok {
				t.Fatalf("verb %q lost event %q between listing and lookup", verb, key)
			}
			// Every index, not just index 0: a golden that renders one index
			// cannot fail on a variant added or reordered behind it.
			for i := 0; i < v.Len(); i++ {
				r := narration.Render(v, tokens, narration.DefaultPicker, i)
				fmt.Fprintf(&b, "%s|%s|%d|actor= %s\n", verb, key, i, r.Actor)
				fmt.Fprintf(&b, "%s|%s|%d|actee= %s\n", verb, key, i, r.Actee)
				fmt.Fprintf(&b, "%s|%s|%d|observer= %s\n", verb, key, i, r.Observer)
				fmt.Fprintf(&b, "%s|%s|%d|remote_observer= %s\n", verb, key, i, r.ActeeObserver)
			}
		}
	}
	return b.String()
}
```

⚠️ The `indexOverride` argument is what makes this golden able to fail on a
variant change. A builder that called `Render` without it would render one
picked index and go green on a pool whose other entries had been rewritten.

- [ ] **Step 5: Record the golden**

```bash
go test ./internal/narration/... -run TestSnapshotStores -update
go test ./internal/narration/... -run TestSnapshotStores
```

Expected: second run PASSES with no diff. This is the **one** legitimate
`-update` in the plan: the store is new, so there is no prior recording to
preserve. `-update` is forbidden in every later task.

- [ ] **Step 6: Commit**

```bash
git add shipped_narration_data_guard_test.go internal/narration/snapshot_test.go internal/narration/testdata/stores/special_moves.golden
git commit -m "test(m4e-1): shipped-data guard and golden for the special-move store

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: The byte-identity net, built BEFORE any file is migrated

This is the task that makes the migration provable. It must go red against the
unmigrated tree, or it is measuring nothing.

**Files:**
- Create: `tools/move_narration_net.py`, `internal/mobcommands/special_move_net_test.go`

- [ ] **Step 1: Extract every old literal to a fixture**

Write `tools/move_narration_net.py`. It reads the 13 mob files, and for each
narration literal records: verb, event key, role, the Go format string, and the
ordered argument names. It writes
`internal/mobcommands/testdata/pre_migration_literals.json`.

⚠️ Decode explicitly as UTF-8. `subprocess(text=True)` and bare `open()` decode
with the **Windows locale codec** on this machine, which mojibakes a dash into a
false difference. Use `open(path, encoding="utf-8")` everywhere, and never
`open(path, "w")` on a file you are also reading — that truncates before the
read expression evaluates.

- [ ] **Step 2: Write the table test**

Create `internal/mobcommands/special_move_net_test.go`. For every row in the
fixture, it renders the YAML event with stand-in values and asserts equality
with `fmt.Sprintf(oldFormat, sameValues)`:

```go
func TestMigratedWordingIsByteIdentical(t *testing.T) {
	rows := loadPreMigrationLiterals(t)
	for _, row := range rows {
		t.Run(row.Verb+"/"+row.Event+"/"+row.Role, func(t *testing.T) {
			g := movenarration.GetMove(row.Verb)
			if g == nil {
				t.Fatalf("verb %q absent from the store", row.Verb)
			}
			v, ok := g.Variants(movenarration.EventKey(row.Event))
			if !ok {
				t.Fatalf("verb %q has no event %q", row.Verb, row.Event)
			}
			got := roleText(narration.Render(v, standInTokens, narration.FirstPicker), row.Role)
			want := fmt.Sprintf(row.Format, standInArgs(row.Args)...)
			if got != want {
				t.Errorf("wording moved:\n got %q\nwant %q", got, want)
			}
		})
	}
}
```

- [ ] **Step 3: Prove the net goes RED before any migration**

Deliberately mis-author one line in `kick.yaml` — change `kicks you hard` to
`kicks you HARD` — and run:

```bash
go test ./internal/mobcommands/ -run TestMigratedWordingIsByteIdentical -v
```

Expected: FAIL on exactly `kick/hit/actee`, printing both strings. **Restore the
line and confirm green.**

🔴 A null probe must be proven capable of failing. This plan's three proofs are
Task 5 step 2, this step, and Task 12 step 3. None is optional. When two
branches are byte-identical, sabotage by line number and confirm the failure
names the right one.

- [ ] **Step 4: Commit**

```bash
git add tools/move_narration_net.py internal/mobcommands/testdata/pre_migration_literals.json internal/mobcommands/special_move_net_test.go
git commit -m "test(m4e-1): the byte-identity net, proven red before migrating

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Migrate `kick.go`, the worked file

Every later migration task repeats these steps exactly. Read this task in full
even if you are executing Task 8 or later; the steps are not repeated there.

**Files:**
- Modify: `internal/mobcommands/kick.go`

- [ ] **Step 1: Replace the branch with a store lookup**

Replace `internal/mobcommands/kick.go:121-135` (quoted in full in the "central
finding" section above) with:

```go
		default: // KickStandard
			sendMoveEvent(`kick`, `standard_hit`, aud, messaging.CategoryKick, map[string]string{
				movenarration.TokenDamage: fmt.Sprintf(`<ansi fg="damage">%s</ansi>`, dmgDesc),
			})
```

The `canSee` local at `:45` and the `targetUser` lookup that feeds **only** it
are now unused. Delete them. Keep any `targetUser` use that still has a
consumer; the compiler names what is left.

- [ ] **Step 2: Write the shared call-site helper**

Create it once, in `internal/mobcommands/move_narration.go`:

```go
// sendMoveEvent renders one special-move event from the store and delivers it.
//
// Identity tokens are filled from the Audience, so the wording never names a
// party the Audience does not also declare -- which is what lets SendTrio hide
// each name by its reader's sight.
func sendMoveEvent(verb string, event movenarration.EventKey, aud messaging.Audience, cat messaging.Category, extra map[string]string) {
	g := movenarration.GetMove(verb)
	if g == nil {
		return
	}
	v, ok := g.Variants(event)
	if !ok {
		return
	}
	tokens := map[string]string{
		narration.TokenActor:      taggedMobName(aud.ActorName),
		narration.TokenActee:      taggedUserName(aud.ActeeName),
		narration.TokenActorPlain: aud.ActorName,
		narration.TokenActeePlain: aud.ActeeName,
	}
	for k, val := range extra {
		tokens[k] = val
	}
	roles := narration.Render(v, tokens, narration.DefaultPicker)
	messaging.SendTrio(messaging.Trio{
		Actor:          messaging.NoLine,
		Actee:          messaging.Say(cat, roles.Actee),
		Observer:       messaging.Say(cat, roles.Observer),
		RemoteObserver: remoteLine(cat, roles.ActeeObserver),
	}, aud)
}
```

`taggedMobName`, `taggedUserName` and `remoteLine` are three-line helpers in the
same file: the first two wrap a name in `<ansi fg="mobname">` /
`<ansi fg="username">`, and `remoteLine` returns `messaging.NoLine` for empty
text so a verb without a second room sends nothing.

- [ ] **Step 3: Run the net and the package tests**

```bash
go test ./internal/mobcommands/ -run TestMigratedWordingIsByteIdentical -v
go test ./internal/mobcommands/ ./internal/movenarration/ ./internal/narration/
```

Expected: PASS. The net proves the wording did not move; the golden proves the
store did not move.

- [ ] **Step 4: Run the root gate**

```bash
go test .
```

Expected: PASS. 🔑 Every task gate includes `go test .` at the **root**: the
surface and line-number allowlist guards live in package `main`, not under
`internal/`, and a package-scoped run cannot see them.

- [ ] **Step 5: Commit**

```bash
git add internal/mobcommands/kick.go internal/mobcommands/move_narration.go
git commit -m "refactor(m4e-1): kick reads its wording from the store

The canSee branch and its 'Something kicks you hard!' twin are deleted, not
migrated: SendTrio already hides the mob's name by each reader's sight, and at
three tiers rather than one.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: Migrate the remaining twelve mob files, in four commits

Repeat Task 7's five steps for each file. Group the commits so a reviewer reads
related verbs together, and so a bisect lands on a small diff.

- [ ] **Step 1: bash, gore, maul, rake** — the plain hit/miss/partial shape
- [ ] **Step 2: drain, throttle, hamstring, pounce** — same shape, verb-specific categories
- [ ] **Step 3: charge, trip** — these have a **second** darkness branch each
      (`charge.go:186`, `trip.go:280`, the "overruns you while prone" lines).
      Both delete the same way as `kick`'s. Confirm with `grep -c canSeeInDark`
      that each file reaches **0** before committing.
- [ ] **Step 4: grapple, shoot** — the two files with prose sourced from outside
      the file.

For `grapple`, Task 4's owner ruling applies: the literals behind
`DisarmResult.Message/TargetMsg/RoomMessage` and
`CritFailure.Message/TargetMessage/RoomMessage` in `internal/actions` move into
`grapple.yaml` in this same commit, and the result structs carry the event key
instead of the prose. Both grapple twins consume them, so the player-side file
reads the migrated wording immediately even though its own literals wait for
PR 1b.

For `shoot`, two specifics, both read from `internal/mobcommands/shoot.go:84-100`:

🪤 **`shoot`'s lines take `{actor_plain}`, not `{actor}`.** The existing code
substitutes `shooter := mobName` **bare**, with no `<ansi>` tag:
`fmt.Sprintf(`%s's shot strikes you!`, shooter)`. Authoring `{actor}` would wrap
it in an identity tag and add colour the line never had, and the net would
correctly report it as a migration defect. Use `{actor_plain}` for every
`shoot` line that names the shooter this way. `HideNames` matches bare names as
whole words, so the pipeline still hides it.

**Preserve the `IsSneaking` disjunction.** The darkness half of
`anonymous := result.IsSneaking || !canSeeInDark(u, room)` is deleted in favour
of the pipeline; the stealth half is not darkness and stays. Implement it by
filling the token, not by touching the `Audience`:

```go
	// Stealth is not darkness. The pipeline hides the shooter's name by the
	// reader's sight; a sneaking shooter is hidden from everyone regardless,
	// so the name never enters the text in the first place.
	actorPlain := mobName
	if result.IsSneaking {
		actorPlain = `Someone`
	}
```

Do **not** clear `aud.ActorName`. `hideForReader` returns the text unchanged
when `otherName == NoName`, so clearing it would *disable* the sight hiding
rather than strengthen it. Leaving `ActorName` as the real name is correct and
harmless in the sneaking case: `HideNames` simply finds nothing to replace,
because the name is no longer in the text.

Write the test for the sneaking-and-dark combination before the code. The four
cases it must pin: seen/not-sneaking (real name), dark/not-sneaking (hidden by
tier), lit/sneaking (`Someone`), dark/sneaking (`Someone`).

Each of the four steps ends with the full Task 7 gate: the net, the package
tests, `go test .`, and a commit.

---

## Task 9: Retire `canSeeInDark` in `mobcommands`

**Files:**
- Modify: `internal/mobcommands/attack.go`, `howl.go`, `taunt.go`,
  `skill_move_defence.go`, `darkness.go`, `go.go`

By this point the eleven special-move call sites are gone. What remains is
`attack.go:94`, `howl.go:54,77`, `taunt.go:71,97,151,184`,
`skill_move_defence.go:78`, and the audio helper.

- [ ] **Step 1: Confirm what is left**

```bash
grep -rn "canSeeInDark" --include=*.go internal/mobcommands
```

Expected: the definition, plus exactly the sites listed above. If a special-move
file still appears, Task 8 is incomplete.

- [ ] **Step 2: Apply the delivery test to each site BEFORE deleting anything**

🔴 **Deleting a darkness branch is only safe where `SendTrio` delivers the line
and the `Audience` declares the name.** Task 7's deletion is safe precisely
because `kick.go` satisfies both. Several sites here satisfy neither, and
deleting their branch would leak the mob's name to a blind player.

For each remaining site, answer both questions from source before touching it:

1. Is the line delivered by `messaging.SendTrio`?
2. Does the `Audience` carry the name in `ActorName` / `ActeeName`?

**Both yes** → delete the branch, author the named sentence only (the Task 7
move).

**Either no** → keep an explicit substitution at the site:

```go
	sight := messaging.ParticipantSight(reader.Character, room)
	text = messaging.HideNames(text, []string{mobName}, sight)
```

Known answers, from the Task 1 inventory:

- **`attack.go:94` makes ZERO `SendTrio` calls.** Its engagement notice is
  outside the Actor/Actee/Observer shape entirely. It takes the explicit
  substitution, **not** the deletion. Its wording is **not** migrated to YAML in
  this PR: putting it on `SendTrio` first is a separate change and belongs with
  PR 4's remaining-paths work. `attack.go` is a sight-only file here.
- **`howl.go:54,77` and `taunt.go:71,97`** — verify each individually. Where the
  line rides the audio channel it is Step 4's problem, not this step's.

Record the verdict per site in the commit message, so a reviewer can check the
test was applied rather than assumed.

- [ ] **Step 3: Migrate the pre-anonymised sites onto the three-tier pair**

`skill_move_defence.go:78` and `taunt.go:151,184` deliberately hand
`SendTrio` text they have already anonymised (`trio.go`'s own docstring names
this file as the reason that is allowed). Replace `messaging.Anonymize(text)`
with the pair:

```go
	sight := messaging.ParticipantSight(targetUser.Character, room)
	text = messaging.HideNames(text, []string{sourceName}, sight)
```

🪤 `SightDecision` runs **best to worst** (`SightFull` = 0, `SightNone` = 2).
Never write `>= SightShapes`; it also matches `SightNone` and would let a blind
reader see shapes. `HideNames` already does the tier mapping — do not
reimplement it with a comparison.

This is the fix for the defect PR 2's playtest found: a fully blind player
reads `a figure` on these lines today, because `Anonymize` knows only that one
word.

- [ ] **Step 4: Handle the audio-channel siblings**

`howl.go:59,82` and `taunt.go:158,190` anonymise unconditionally because the
**audio channel bypasses the sight gate entirely** — the same shape PR 3 found
in `position_control`. Route them through `HideNames` with the listener's own
sight so an infrared listener hears `a figure` and a blind one `something`,
instead of everyone getting one word.

`sendAudioRoomText` (`darkness.go:19-42`) is the third hand-rolled darkness
check; its own comment at `go.go:96-98` says it should collapse, naming M5.
Collapse it here rather than opening these files a third time: replace its
`HasFlagFromAnySource(conditions.NightVision)` per-player branch with
`messaging.ParticipantSight` + `HideNames`.

- [ ] **Step 5: Delete the predicate**

```bash
grep -rn "canSeeInDark" --include=*.go internal/mobcommands
```

Expected: **no output** apart from the definition. Delete the definition from
`darkness.go`. Run `go build ./...`; the compiler is the dead-code sweep and
will name any consumer left.

- [ ] **Step 6: Gate and commit**

```bash
go test ./internal/mobcommands/ ./internal/messaging/ && go test .
```

```bash
git add internal/mobcommands/
git commit -m "refactor(m4e-1): retire canSeeInDark in mobcommands

The binary predicate collapsed SightShapes and SightNone into one word, so a
fully blind player read 'a figure'. Every site now reads ParticipantSight and
substitutes through HideNames, which knows all three tiers. sendAudioRoomText
collapses with them rather than being opened a third time.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: The `usercommands` twin and the shoot defect

**Files:**
- Modify: `internal/usercommands/skill_move_defence.go`, `internal/usercommands/shoot.go`

- [ ] **Step 1: Migrate the twin predicate**

`skill_move_defence.go:114` is the same pre-anonymised shape as its mob twin.
Apply the Task 9 step 3 change, then delete the twin definition at `:93-95` and
the comment at `:86-92` that describes the collapse this task performs.

- [ ] **Step 2: Write the failing test for the shoot defect**

Owner ruling 2 makes the player shoot path darkness-sensitive like the mob path.
Write the test first, in `internal/usercommands/shoot_darkness_test.go`: a
shooter in an unlit room, a victim with no night vision, asserting the victim's
personal line does **not** contain the shooter's name.

- [ ] **Step 3: Run it to confirm it fails**

```bash
go test ./internal/usercommands/ -run TestShootHidesShooterInDark -v
```

Expected: FAIL — the line contains the shooter's real name. This failure **is**
the bug: a player who shoots from a dark room hands the victim their identity.

- [ ] **Step 4: Fix it**

Fill `narration.TokenActor` from the reader's sight the same way the mob twin
now does, keeping the `IsSneaking` disjunction separate.

- [ ] **Step 5: Confirm green, then gate**

```bash
go test ./internal/usercommands/ && go test .
```

- [ ] **Step 6: Commit, flagged**

```bash
git add internal/usercommands/skill_move_defence.go internal/usercommands/shoot.go internal/usercommands/shoot_darkness_test.go
git commit -m "fix(m4e-1): a player shooting from the dark no longer names themself

BEHAVIOUR CHANGE, flagged. usercommands/shoot.go anonymised on IsSneaking only
and never consulted darkness, while the mob twin ORed in !canSeeInDark. The
player path now matches the mob path.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 11: Record the deliberate diff, verbatim, for all three tiers

The wording half of this PR is byte-identical and the net proves it. The sight
half is a deliberate player-visible diff and needs a record a reviewer can read
without running the server.

**Files:**
- Create: `docs/superpowers/audits/2026-09-21-m4e1-darkness-diff.md`

- [ ] **Step 1: Write a rendering harness test**

A test that renders one representative event (`kick/standard_hit`) at each of
`SightFull`, `SightShapes`, `SightNone`, printing the actee line, and does the
same against the pre-migration literal path.

- [ ] **Step 2: Record the table**

| Reader | Before | After |
|---|---|---|
| Clear sight | `<ansi fg="mobname">Goblin</ansi> kicks you hard! (...)` | unchanged |
| Infrared (shapes) | `Something kicks you hard! (...)` | `<ansi fg="combat-anon">A figure</ansi> kicks you hard! (...)` |
| Blind | `Something kicks you hard! (...)` | `<ansi fg="combat-anon">Something</ansi> kicks you hard! (...)` |

The infrared row is the improvement this PR exists for; the blind row's only
change is the colour tag, the accepted side effect carried over from PR 2.

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/audits/2026-09-21-m4e1-darkness-diff.md
git commit -m "docs(m4e-1): the darkness diff, all three sight tiers, verbatim

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 12: Shrink the frozen lists and replace them with a stronger guard

**Files:**
- Modify: `messaging_surface_guard_test.go`, `m2_routing_guard_test.go`

- [ ] **Step 1: Remove the 13 mob entries**

Delete the thirteen `internal/mobcommands/*.go` rows from `m2FrozenFiles`
(`messaging_surface_guard_test.go:1508`) and from `m2RoutingFiles`
(`m2_routing_guard_test.go:70`). The twelve `usercommands` rows stay; PR 1b
removes them. `TestM2FreezeListsAgree` must still pass, proving the two lists
stayed 1:1.

- [ ] **Step 2: Add the forward guard**

A frozen fingerprint says "this file's literals have not changed". The stronger
claim now available is "this file holds no narration literal at all". Add
`TestMigratedFilesHoldNoNarrationLiterals` with an explicit list of the thirteen
migrated files, asserting each holds no backtick prose literal outside imports.

Keep the allowlist **empty**. A guard with a populated allowlist on day one is a
guard that never bites.

- [ ] **Step 3: Prove the new guard can fail**

Add a literal back to `internal/mobcommands/kick.go` — a plain
`` `You feel kicked.` `` assigned to an unused local — and run:

```bash
go test . -run TestMigratedFilesHoldNoNarrationLiterals -v
```

Expected: FAIL naming `internal/mobcommands/kick.go`. Remove the literal and
confirm green. The sabotage must **compile**; a guard proven only against a
build failure is proven against nothing.

- [ ] **Step 4: Add the key-agreement guard, in BOTH directions**

The spec requires that every event key Go references exists in YAML, **and**
that every YAML event is referenced by Go. One direction catches a typo that
silences a move; the other catches wording that ships but can never render.

Add `TestMoveEventKeysAgree` to the same root test file:

```go
func TestMoveEventKeysAgree(t *testing.T) {
	// Referenced: every sendMoveEvent(`verb`, `event`, ...) in mobcommands.
	call := regexp.MustCompile("sendMoveEvent\\(`([a-z_]+)`,\\s*`([a-z_]+)`")
	referenced := map[string]bool{}
	for _, path := range migratedSpecialMoveFiles {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		for _, m := range call.FindAllStringSubmatch(string(src), -1) {
			referenced[m[1]+"/"+m[2]] = true
		}
	}
	if len(referenced) == 0 {
		t.Fatal("no sendMoveEvent call sites found; the regex cannot succeed, so this test proves nothing")
	}

	authored := map[string]bool{}
	dir := filepath.Join(shippedWorldRoot, "narration", "special-moves")
	groups, err := fileloader.LoadAllFlatFiles[string, *movenarration.MoveNarrationGroup](dir)
	if err != nil {
		t.Fatalf("loading %s: %v", dir, err)
	}
	for verb, g := range groups {
		for key := range g.Events {
			authored[verb+"/"+string(key)] = true
		}
	}

	for k := range referenced {
		if !authored[k] {
			t.Errorf("Go references event %q but no YAML authors it: the move would narrate nothing", k)
		}
	}
	for k := range authored {
		if !referenced[k] {
			t.Errorf("YAML authors event %q but no Go call site reaches it: dead wording", k)
		}
	}
}
```

⚠️ The `len(referenced) == 0` check is load-bearing. A regex with a typo, or one
that stops matching after a refactor changes the call shape, would otherwise
produce an empty set that satisfies the first loop vacuously — the same
"vacuous pass" defect that shipped an empty pair list during M4b-1. **"I grepped
and found nothing" is only evidence if the grep could have found something.**

- [ ] **Step 5: Prove that guard can fail, both ways**

Rename one event key in `kick.yaml` (`standard_hit` to `standard_hits`) and run:

```bash
go test . -run TestMoveEventKeysAgree -v
```

Expected: **two** failures, one from each direction — `kick/standard_hit` referenced but
not authored, `kick/standard_hits` authored but not referenced. Restore and confirm
green. A one-sided failure means one of the two loops is not wired.

- [ ] **Step 6: Commit**

```bash
git add messaging_surface_guard_test.go m2_routing_guard_test.go
git commit -m "test(m4e-1): the mob files graduate from frozen to literal-free

Adds the bidirectional key-agreement guard: a referenced key with no YAML
narrates nothing, and authored YAML no call site reaches is dead wording.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 13: Docs, ledger, boot check, and the pre-push gate

**Files:**
- Create: `internal/movenarration/context.md`
- Modify: `internal/mobcommands/context.md`, `internal/messaging/context.md`,
  `docs/README.md`, `docs/superpowers/audits/messaging-m6-content-ledger.md`

- [ ] **Step 1: Write `internal/movenarration/context.md`**

Required for every new package. **Verify before you document:** every symbol
named must exist. Extract the real surface rather than recalling it:

```bash
grep -nE '^(func|type|const|var)\s' internal/movenarration/*.go
```

- [ ] **Step 2: Update the two touched packages' `context.md`**

`internal/mobcommands` loses `canSeeInDark` and gains a store dependency;
`internal/messaging` gains nothing but should note that the mob commands are now
full pipeline citizens.

- [ ] **Step 3: Add the M6 ledger rows**

Every slice adds its deferred-text rows in the same commit. This slice defers:
the twelve player-side files and their ~55 unwritten variants (PR 1b); the
`Actor` role left empty in all thirteen YAML files; and whether mob and player
wording should converge now that they share one event file.

- [ ] **Step 4: Index the new files in `docs/README.md`**

State the full path of each new file. Required by project convention.

- [ ] **Step 5: Boot check in an isolated worktree**

🪤 A branch switch is **blocked** whenever `_datafiles/config.yaml` differs
between branches: it carries the git skip-worktree bit, so git will not
overwrite the local copy, which holds local-only HttpPort, LogLevel and Playtest
settings. Do **not** clear the bit. Use a worktree, and remove it from the main
checkout's directory, never from inside the worktree.

```bash
git worktree add ../dogmud-m4e1-boot HEAD
```

Boot there and check the output for `Server Ready` and for the absence of
`PANIC`. Never check `$?`: a failed boot exits 0.

- [ ] **Step 6: Run the full pre-push gate**

```bash
gofmt -l $(git diff --name-only master..HEAD | grep '\.go$')
```

Expected: **no output**. 🪤 CI's `validate / test` runs a gofmt gate **separate**
from golangci-lint, and it fails the whole job in about 30 seconds before a
single test runs. A green `golangci-lint run --new-from-rev=master` says nothing
about it.

```bash
golangci-lint run --new-from-rev=master
go test ./...
go test .
```

🪤 `-race` cannot run on this machine: there is no gcc, so cgo is unavailable
and `go test -race` fails instantly. CI runs it, under the 900s per-binary cap
raised by #150. Treat a combat timeout in CI as real, not as a flake.

- [ ] **Step 7: Commit and open the PR**

```bash
git add internal/movenarration/context.md internal/mobcommands/context.md internal/messaging/context.md docs/README.md docs/superpowers/audits/messaging-m6-content-ledger.md
git commit -m "docs(m4e-1): context files, ledger rows, and the docs index

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

🚨 Every `gh` command carries `--repo pruuk/DOGMud`. This repo is a fork of
`GoMudEngine/GoMud` and `gh` defaults to the **parent**; a bare `gh pr create`
once opened a PR on upstream.

```bash
gh pr create --repo pruuk/DOGMud --base master \
  --title "M4e PR 1a: mob special moves to YAML, and canSeeInDark retired" \
  --body-file <(cat docs/superpowers/audits/2026-09-21-m4e1-darkness-diff.md)
```

🪤 Never `git add -A` or `git add .`. Named paths only, as every commit above
does.

---

## Task 14: The playtest gate

**Files:**
- Create: `tools/playtest/goals/2026-09-21-m4e1-mob-moves-dark.yaml`

- [ ] **Step 1: Write a goals file that survives**

🔴 A combat fixture must **survive rounds** or the playtest comes back partial —
PR 3's grapple check staged only partially because the character fought two mobs
and died in under fifteen rounds. Use **one** mob, and prefer Sable's arena
(`ask sable arena <gold>` in the Rift Chamber, room 5000); gold is the
difficulty dial.

The fixture needs a mob that actually uses special moves, in an unlit room, with
the player lacking night vision, held long enough for several verbs to fire.

- [ ] **Step 2: Run it and quote the lines verbatim**

Capture the actee lines for at least four distinct verbs. Quote them in the PR
thread.

- [ ] **Step 3: Record what the playtest could NOT prove**

Two things, both known in advance:
- **The shapes tier will not exercise itself.** PR 2's run failed to load
  condition 85 at runtime and got the blind tier twice; its guard covers
  template loading only, not materialize-to-login, and it had already failed the
  same way on 2026-09-11. If condition 85 is not confirmed live in the session,
  say the shapes tier is unverified rather than implying it passed.
- **Colour is unverifiable by the harness at all** — the AI port strips ANSI.
  Use telnet on 33333 if the `combat-anon` tag must be seen.

- [ ] **Step 4: Extract findings to memory**

🧹 Playtest reports are gitignored. Anything learned must be written to a memory
file in the same session or it is lost.

- [ ] **Step 5: Commit the fixture**

```bash
git add tools/playtest/goals/2026-09-21-m4e1-mob-moves-dark.yaml
git commit -m "test(m4e-1): dark special-move playtest fixture

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## How we know it worked

1. **Wording did not move:** Task 6's net, proven red first, passes for every
   literal in the inventory.
2. **The store is sound:** the shipped-data guard and the golden both pass, and
   both were proven capable of failing.
3. **The sight change is the intended one:** Task 11's table, and the playtest's
   verbatim lines for at least four verbs.
4. **Nothing was missed:** `grep -rn canSeeInDark --include=*.go .` returns
   nothing, and Task 12's literal-free guard passes with an empty allowlist.
5. **It boots:** `Server Ready` in an isolated worktree, no `PANIC` line.

## What PR 1b inherits

- The twelve `usercommands` files and their ~55 unwritten variants, to square
  every ragged pool at its widest.
- The `Actor` role, empty in all thirteen YAML files, which the player side
  fills.
- The twelve `m2FrozenFiles` / `m2RoutingFiles` rows that remain, and the
  deletion of `TestM2LiteralsAreFrozen` once the last file graduates.
- The open question of whether mob and player wording should converge, now that
  one event file serves both.
