# Lighting Plan 3b: Biome Vocabulary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give the world a biome vocabulary honest enough to use the light model plan 3a built: six new biomes, one deleted, one dead zone removed, and every one of the 117 orphan rooms resolved.

**Architecture:** Almost entirely data. `BiomeInfo` already carries `skylight` and `lamp` from 3a, and `Room` already carries per-room overrides, so no Go type changes. The Go edits are three small ones: two weather classification maps, a weather climate archetype binding, and the biome template.

**Tech Stack:** YAML data files under `_datafiles/world/dogmud/`, Go in `modules/weather/`, golden-file tests under `testdata/`.

**Spec:** `docs/superpowers/specs/2026-09-23-lighting-plan3b-biome-vocabulary-design.md`

**Branch:** `feature/lighting-plan3b-biome-vocabulary`, already created from master `d8562aa5b`.

---

## Traps. Read before Task 1.

🪤 **Go templates reach methods by REFLECTION, invisible to `go build`.**
`_datafiles/world/dogmud/templates/descriptions/biome.template` reads biome
methods on a `*BiomeInfo` handed to `templates.Process`. This broke twice in
3a with a green build and a green suite. **If you touch a template, RENDER it
in a test.**

🪤 **`internal/templates/templatesfunctions.go` shadows `lt`, `lte` and `gte`
as INT-ONLY.** A template comparing a float fails at render time with "wrong
type for value; expected int; got float64". `le` and `ge` are not shadowed.

🪤 **`biome_coupling_test.go` is your friend here, not an obstacle.** Every
`indoor: true` biome must appear in exactly one of `undergroundBiomes` or
`surfaceIndoorBiomes` (`modules/weather/content/emotes.go`), and a name in
either map that is not a real biome fails `TestClassificationMapsHaveNoDeadKeys`
with a message naming it. Deleting `house` without touching that map gives you
a clear red rather than a silent hole.

🪤 **`gametime`'s `roundDateCache` is keyed on the round with NO config
fingerprint.** A test that changes lighting config and reuses another test's
round silently gets that test's answer. Call `gametime.ClearDateCacheForTest()`
before sampling and on cleanup.

🪤 **`gofmt -l` false-positives on Windows** (CRLF copy, LF blob) but has been
genuinely right three times on this arc. Check `tr -cd '\r' < <path> | wc -c`
AND `gofmt -d <path>` before dismissing it.

🪤 **`grep -c` exits 1 on zero matches**, so an "expect zero" check breaks an
`&&` chain and silently skips everything after it. Run such checks standalone.

🪤 **Never `git add -A` or `git add .`.** Named paths only.

🪤 **Never edit a file with a Python read-modify-write.** `open(path, 'w')`
truncates before the write expression evaluates; this has destroyed files here
twice. Read fully into a variable first, or use the Edit tool.

🪤 **The boot-check recipe builds from `HEAD`, which does NOT contain your
uncommitted work.** Every task here boots BEFORE committing, so booting `HEAD`
would test the tree without your change and pass for the wrong reason. Snapshot
the working tree first:

```bash
SNAP=$(git stash create)
git worktree add --detach C:/tmp/dogmud-boot-check "$SNAP"
```

`git stash create` writes a commit object without touching your index, your
working tree or the stash list. Confirm the snapshot actually contains the
change with `git ls-tree` before booting, or you have proven nothing.

🚨 **Run `golangci-lint run --new-from-merge-base=origin/master` before pushing.**
It is step 3 of the pre-push SOP and it is the only local gate that sees what
CI's lint gate sees. `unconvert` and friends are in neither `go vet` nor `gofmt`.

---

## File Structure

**Created:**

| Path | Responsibility |
|---|---|
| `tools/lighting_golden_diff.py` | Read-only analyser that groups a golden's diff by section and biome. Six tasks in this plan move a golden and each must prove its shape; this is that proof, written once |
| `_datafiles/world/dogmud/biomes/sewer.yaml` | Underground, built, wet |
| `_datafiles/world/dogmud/biomes/interior.yaml` | Built indoor space; absorbs `house` |
| `_datafiles/world/dogmud/biomes/dense_forest.yaml` | Canopy thick enough to matter |
| `_datafiles/world/dogmud/biomes/plains.yaml` | Open grassland |
| `_datafiles/world/dogmud/biomes/river.yaml` | Flowing water |
| `_datafiles/world/dogmud/biomes/ether.yaml` | Outside the world; time-invariant |
| `_datafiles/world/dogmud/weather/climate/{sewer,interior,dense_forest,plains,river,ether}.yaml` | One climate file per new biome |

**Deleted:**

| Path | Why |
|---|---|
| `_datafiles/world/dogmud/rooms/a_dark_forest/` (82 files) | Spec facts 2 and 3: unreachable, unpopulated, stub text, not upstream's |
| `_datafiles/world/dogmud/biomes/house.yaml` | Folded into `interior` |
| `_datafiles/world/dogmud/weather/climate/house.yaml` | Follows its biome |

**Modified:** `modules/weather/content/emotes.go` (the two classification maps), `modules/weather/sim/climate.go` (archetype bindings), `_datafiles/world/dogmud/templates/descriptions/biome.template` if the `house` deletion touches it, ~130 room YAMLs, 4 zone-configs, 2 mutator YAMLs, both goldens.

---

### Task 1: The golden shape-prover

Six later tasks move a golden. In 3a every implementer wrote this script from
scratch and one of them reported a shape that was right for the wrong reason.
Write it once, commit it, use it six times.

**Files:**
- Create: `tools/lighting_golden_diff.py`
- Modify: `docs/README.md` (the tools table)

- [ ] **Step 1: Write the tool**

```python
#!/usr/bin/env python3
"""Group a lighting golden's diff by sample section and biome.

Read-only. Never writes a golden; it only explains one that moved.

    python tools/lighting_golden_diff.py <before> <after>

<before> may be a git ref-ish path, in which case pipe it in yourself:

    git show HEAD:testdata/lighting_daycycle.golden > before.tmp
    python tools/lighting_golden_diff.py before.tmp testdata/lighting_daycycle.golden

Every golden move in the graded lighting arc must be explained room-for-room
before it is re-recorded. A move you cannot account for is a defect, however
plausible the totals look.
"""
import collections
import io
import re
import sys

ROOM = re.compile(r"^room (-?\d+) biome=(\S+) light=(-?\d+)")


def parse(path):
    """Return {section: {roomid: (biome, light)}} and the section order."""
    sections, order, current = {}, [], None
    for line in io.open(path, encoding="utf-8"):
        line = line.rstrip("\n")
        if line.startswith("== "):
            current = line
            sections[current] = {}
            order.append(current)
        elif current:
            m = ROOM.match(line)
            if m:
                sections[current][int(m.group(1))] = (m.group(2), int(m.group(3)))
    return sections, order


def main(before_path, after_path):
    before, order = parse(before_path)
    after, after_order = parse(after_path)

    if len(before) != len(after):
        print(f"SECTION COUNT CHANGED: {len(before)} -> {len(after)}")

    total = 0
    for section in order:
        b = before.get(section, {})
        a = after.get(section, {})
        if section not in after:
            print(f"{section}: SECTION REMOVED")
            continue

        gone = sorted(set(b) - set(a))
        added = sorted(set(a) - set(b))
        moved = [(r, b[r], a[r]) for r in sorted(set(b) & set(a)) if b[r] != a[r]]
        total += len(moved)

        if not (gone or added or moved):
            print(f"{section}: unchanged")
            continue

        print(f"{section}: {len(moved)} moved, {len(gone)} removed, {len(added)} added")
        kinds = collections.Counter(
            (old[0], new[0], old[1], new[1]) for _, old, new in moved
        )
        for (ob, nb, ol, nl), count in kinds.most_common():
            arrow = f"{ob} -> {nb}" if ob != nb else ob
            print(f"      {arrow:<28} {ol:>4} -> {nl:<4} x{count}")
        if gone:
            gone_biomes = collections.Counter(b[r][0] for r in gone)
            print(f"      removed by biome: {dict(gone_biomes)}")
        if added:
            add_biomes = collections.Counter(a[r][0] for r in added)
            print(f"      added by biome:   {dict(add_biomes)}")

    print(f"\nTOTAL room-readings moved: {total}")
    rooms_before = len(before[order[0]]) if order else 0
    rooms_after = len(after[after_order[0]]) if after_order else 0
    if rooms_before != rooms_after:
        print(f"ROOM COUNT CHANGED: {rooms_before} -> {rooms_after}")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        print(__doc__)
        sys.exit(2)
    main(sys.argv[1], sys.argv[2])
```

- [ ] **Step 2: Prove it reports "unchanged" on an unchanged golden**

```bash
git show HEAD:testdata/lighting_daycycle.golden > before.tmp
python tools/lighting_golden_diff.py before.tmp testdata/lighting_daycycle.golden
```
Expected: every section `unchanged`, `TOTAL room-readings moved: 0`.

- [ ] **Step 3: Prove it can SEE a change**

A tool that reports "unchanged" for everything is useless. Perturb a copy and
confirm it is caught:

```bash
sed 's/^room 75 biome=default light=\([0-9]*\)/room 75 biome=default light=99/' testdata/lighting_daycycle.golden > after.tmp
python tools/lighting_golden_diff.py before.tmp after.tmp
```
Expected: 12 sections each reporting 1 moved, with `default` and `-> 99`.

Then: `rm -f before.tmp after.tmp` and confirm `git status --short` is clean
apart from the new tool.

- [ ] **Step 4: Add a row to `docs/README.md`'s tools table**

Match the existing format for `tools/context_md_audit.py` and the combat audit
scripts. One line: what it does and that it is read-only.

- [ ] **Step 5: Commit**

```bash
git add tools/lighting_golden_diff.py docs/README.md
git commit -m "tools(lighting): a read-only shape-prover for golden moves

Six tasks in plan 3b move a lighting golden, and every move in this arc must
be explained room-for-room before it is re-recorded. In 3a each implementer
wrote this analysis from scratch and one reported a shape that was right for
the wrong reason.

Groups a diff by sample section and by biome transition, and reports section
and room COUNT changes separately, because 3b deletes a zone and a shrinking
room count is a different event from a changing light value.

Proven both ways before committing: reports every section unchanged on an
unmoved golden, and catches a single perturbed room across all twelve
sections.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Delete `a_dark_forest`

81 rooms nothing can reach, zero mob spawns, stub descriptions, no reference
anywhere. Spec facts 2, 3 and 4.

**Files:**
- Delete: `_datafiles/world/dogmud/rooms/a_dark_forest/` (82 files)
- Modify: `testdata/lighting_daycycle.golden`, `testdata/lighting_parity.golden`

- [ ] **Step 1: Re-verify the three facts that justify deletion**

Do not take the spec's word. Run each standalone:

```bash
grep -rn "roomid: 10[0-8][0-9]" _datafiles/world/dogmud/rooms/ --include=*.yaml | grep -v "/a_dark_forest/"
```
Expected: no output, or only matches that are NOT exit targets. Inspect any hit.

```bash
grep -rln "A Dark Forest\|a_dark_forest" _datafiles/world/dogmud/ --include=*.yaml --include=*.js | grep -v "/rooms/a_dark_forest/"
```
Expected: no output.

```bash
grep -l "spawninfo" _datafiles/world/dogmud/rooms/a_dark_forest/*.yaml | wc -l
```
Expected: `0`.

⚠️ **If any of these three comes back non-empty, STOP and report.** The
deletion's whole justification is that the zone is inert.

- [ ] **Step 2: Delete the zone**

```bash
git rm -r _datafiles/world/dogmud/rooms/a_dark_forest/
```

- [ ] **Step 3: Boot the server**

Deleting 81 rooms can orphan an exit elsewhere, and only a real boot catches a
dangling room reference. Use the isolated detached worktree recipe from the
`dogmud-shipping` skill (step 6 of the Pre-Push SOP). Confirm `Server Ready`
and zero panics.

⚠️ **This is the one step in this plan that no test replaces.** Do not skip it.

- [ ] **Step 4: Re-record both goldens and prove the shape**

```bash
git show HEAD:testdata/lighting_daycycle.golden > before.tmp
go test . -run TestLightingDayCycleAcrossSampleRounds -update-lighting-daycycle
go test . -run TestLightingParityAcrossEveryShippedRoom -update-lighting-parity
python tools/lighting_golden_diff.py before.tmp testdata/lighting_daycycle.golden
rm -f before.tmp
```

Expected shape, and **nothing else**:
- `ROOM COUNT CHANGED: 1386 -> 1305`
- every section reporting **81 removed**, all of biome `default`
- **0 moved** in every section

🔴 **If any room MOVED, stop.** Deleting rooms must not change the light of a
room that remains. A move here means something else in the tree changed too.

- [ ] **Step 5: Verify and commit**

```bash
go build ./... && go test ./...
git add -u _datafiles/world/dogmud/rooms/ testdata/
git commit -m "content(world): delete a_dark_forest, 81 unreachable stub rooms

Nothing outside the zone exits into it, it has zero mob spawns, every
description is the stub 'This area has not yet been described', and no quest,
script or content file references it anywhere. All three re-verified before
deleting rather than taken from the spec.

It is DOGMud's own zone rather than upstream's: world/default has no copy, so
this cannot conflict on an upstream merge. It had been swept along by bulk
migrations without ever being authored.

Room ids 1002 to 1082 return to the pool.

Both goldens re-recorded. The shape is 81 rooms removed from every section,
all of biome default, and ZERO rooms moved: deleting rooms must not change the
light of a room that remains. Room count 1386 to 1305.

Booted the server in an isolated worktree to confirm no exit elsewhere
referenced a deleted room, which no test would have caught.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: The six new biomes

Add all six at once. They have no rooms yet, which is deliberate: this task is
purely additive and must move no golden at all.

**Files:**
- Create: six files in `_datafiles/world/dogmud/biomes/`
- Create: six files in `_datafiles/world/dogmud/weather/climate/`
- Modify: `modules/weather/content/emotes.go`
- Modify: `modules/weather/sim/climate.go`

- [ ] **Step 1: Write the six biome files**

`sewer.yaml`:
```yaml
biomeid: sewer
name: Sewer
symbol: ⌇
description: Brick vaults and channels beneath a city, carrying its water and
  its waste. Lightless except where a grate or a pried drain-cap lets a column
  of daylight down.
skylight: 0.0
requireditemid: 0
usesitem: false
burns: false
movementcost: 1.4
indoor: true
```

`interior.yaml`:
```yaml
biomeid: interior
name: Interior
symbol: ⌂
description: Inside a built structure, whether a house, a hall, a temple or the
  sealed corridors of something older. Lit by what its builders put there
  rather than by the sky.
skylight: 0.15
lamp: 50
requireditemid: 0
usesitem: false
burns: false
movementcost: 1.0
indoor: true
```

`dense_forest.yaml`:
```yaml
biomeid: dense_forest
name: Dense Forest
symbol: ♠
description: Old timber grown close, where the canopy closes over and the
  ground stays dim even at midday. Deeper and darker than ordinary woodland.
skylight: 0.25
requireditemid: 0
usesitem: false
burns: true
movementcost: 1.5
```

`plains.yaml`:
```yaml
biomeid: plains
name: Plains
symbol: ⁖
description: Open grassland running to the horizon, with nothing between the
  ground and the sky.
skylight: 1.0
requireditemid: 0
usesitem: false
burns: true
movementcost: 1.0
```

`river.yaml`:
```yaml
biomeid: river
name: River
symbol: ≋
description: Moving water with a current to it, unlike the still water of a
  lake or a pool.
skylight: 1.0
requireditemid: 0
usesitem: false
burns: false
movementcost: 2.0
```

`ether.yaml`:
```yaml
biomeid: ether
name: Ether
symbol: ∴
description: Not a place in the world at all. What holds you before you arrive,
  or after you have left, lit by nothing in particular and unchanged by any
  hour.
skylight: 0.0
lamp: 60
requireditemid: 0
usesitem: false
burns: false
movementcost: 1.0
indoor: true
```

⚠️ **Verify every symbol is distinct from the existing set** (`cave ⌬`,
`city •`, `cliffs ▼`, `desert *`, `dungeon •`, `farmland ,`, `forest ♣`,
`fort •`, `house ⌂`, `land •`, `mountains ⩕`, `road •`, `shore ~`, `snow ❄`,
`spiderweb 🕸`, `swamp ♨`, `water ≈`). `interior` deliberately takes `⌂` from
`house`, which Task 4 deletes. If a symbol does not render in your terminal,
pick another and say which.

⚠️ **`burns` and `movementcost` are not lighting fields.** Values above are
chosen to match each biome's nearest existing sibling; check them against
`forest.yaml`, `water.yaml` and `house.yaml` and adjust if a sibling disagrees.

- [ ] **Step 2: Write the six climate files**

All three indoor biomes take the sheltered archetype, which is what `cave.yaml`
and `house.yaml` both carry verbatim today:

```yaml
biome: sewer
weather: { clear: 1 }
influence: { intensityDelta: -0.04, moistureDelta: -0.04, movementResistance: 0.3 }
spawnWeight: 0.0
```

Identical for `interior.yaml` and `ether.yaml`, changing only the `biome:` key.

`dense_forest.yaml` follows `forest.yaml`, with a little more shelter:

```yaml
biome: dense_forest
weather: { clear: 3, overcast: 2, rain: 2, fog: 1, storm: 1 }
influence: { intensityDelta: 0.0, moistureDelta: 0.02, movementResistance: 0.1 }
spawnWeight: 0.9
track: temperate
```

`plains.yaml` follows `land.yaml`:

```yaml
biome: plains
weather: { clear: 4, overcast: 2, rain: 1, fog: 1 }
influence: { intensityDelta: 0.0, moistureDelta: 0.0, movementResistance: 0.0 }
spawnWeight: 0.8
track: temperate
```

`river.yaml` follows `water.yaml`:

```yaml
biome: river
weather: { clear: 2, overcast: 2, fog: 2, rain: 2, storm: 1 }
influence: { intensityDelta: 0.06, moistureDelta: 0.10, movementResistance: 0.0 }
spawnWeight: 1.4
track: temperate
```

⚠️ **Read each source file before copying** and report if it differs from the
above; these were read on 2026-09-23 and a later edit would make them stale.

- [ ] **Step 3: Classify the three new indoor biomes**

In `modules/weather/content/emotes.go`, add `sewer` to `undergroundBiomes`
(brick vaults under a city are felt through stone, not through a roof), and add
`interior` and `ether` to `surfaceIndoorBiomes`.

⚠️ `ether` is not really either, having no weather at all, but the
classification must be TOTAL or `TestEveryIndoorBiomeIsClassified` fails.
Surface-indoor is the right default for a place with no stone around it.
**Comment the choice**, since a reader will wonder.

- [ ] **Step 4: Bind climate archetypes**

Read `modules/weather/sim/climate.go` around the existing `land`, `city` and
`fort` bindings. Bind `plains` and `river` to the archetypes their siblings
use. Leave the three indoor biomes unbound if `fort`'s comment says indoor
biomes are deliberately unbound; follow whatever that file already does.

- [ ] **Step 5: Verify**

```bash
go build ./... && go test ./modules/weather/... && go test ./...
```
Expected: PASS. In particular `TestEveryIndoorBiomeIsClassified`,
`TestClassificationMapsHaveNoDeadKeys` and `TestAuthoredBiomeKeysAreRealBiomes`
must pass.

```bash
git show HEAD:testdata/lighting_daycycle.golden > before.tmp
python tools/lighting_golden_diff.py before.tmp testdata/lighting_daycycle.golden
rm -f before.tmp
```
Expected: every section `unchanged`. **Adding a biome no room uses must move
nothing.** If anything moved, a room is silently picking up a new biome.

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/biomes/ _datafiles/world/dogmud/weather/climate/ modules/weather/
git commit -m "content(biomes): add sewer, interior, dense_forest, plains, river, ether

Six new biomes with no rooms yet, so this commit is purely additive and moves
no golden. The rooms arrive in the following tasks.

sewer is underground, built and wet, which nothing in the existing set means,
and it joins undergroundBiomes so its weather prose is felt through stone
rather than through a roof.

interior is a built indoor space and takes house's map symbol, since Task 4
deletes house and folds its rooms in here.

dense_forest at skylight 0.25 against forest's 0.45. forest is already blind
at night, so a darker forest can only differentiate at midday: 0.25 puts
midwinter noon in the shapes band while equinox and midsummer stay clear.

plains and river change no light at all, both being open sky like land and
water. They are here on world-building grounds, so that four zone-configs stop
naming biomes that do not exist, and the spec says so plainly.

ether is not a place in the world. skylight 0 and lamp 60 make it
time-invariant by construction, which the character-creation rooms require:
look.go refuses a room description below the blind threshold, so an
antechamber that followed the sun would leave new players unable to read their
own introduction.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Delete `house`, fold into `interior`

**Files:**
- Delete: `_datafiles/world/dogmud/biomes/house.yaml`, `_datafiles/world/dogmud/weather/climate/house.yaml`
- Modify: `modules/weather/content/emotes.go`, 15 room YAMLs, both goldens

- [ ] **Step 1: Find every reference to `house` as a biome**

Run each standalone. The Go tree, the data tree, AND the templates:

```bash
grep -rn "biome: house" _datafiles/world/dogmud/rooms/
grep -rn '"house"' --include=*.go internal/ modules/
grep -rn "house" _datafiles/world/dogmud/templates/
```

Report all three results. The third matters most: **a template reference is
invisible to the compiler.**

- [ ] **Step 2: Move the 15 rooms**

Change `biome: house` to `biome: interior` in each. Expect: Chrysalis School
Hall, Cleric's Study, Coulee Provisions, Lantern Sleeping Loft, Ore Stall,
Strongbox House, The Barn, The Cook-Fire, The Drowned Lantern, The Longhouse,
The Mending Hut, The Patch-Tent, The Threshold House, The Warder's Dugout,
Wickerwork Cottage. **Verify the count is 15**; if it differs, report.

- [ ] **Step 3: Delete the biome and its climate file, and unclassify it**

```bash
git rm _datafiles/world/dogmud/biomes/house.yaml _datafiles/world/dogmud/weather/climate/house.yaml
```

Remove `"house": true` from `surfaceIndoorBiomes` in
`modules/weather/content/emotes.go`.

- [ ] **Step 4: Prove the coupling test would have caught you**

Before running the suite clean, confirm the guard works: temporarily leave
`"house"` in `surfaceIndoorBiomes` with the biome file deleted, and run:

```bash
go test ./modules/weather/content/ -run TestClassificationMapsHaveNoDeadKeys
```
Expected: FAIL naming `house`. Then remove the key and confirm PASS.

🔑 Record the failure text in your report. This guard is the reason a biome
deletion is safe here at all.

- [ ] **Step 5: Render the biome template**

`house` is gone, so anything the template said about it is now wrong, and a
template error is invisible to `go build` and `go test`. Write a TEMPORARY test
in `internal/templates` using that package's existing `TestMain` fixture, call
`templates.Process("descriptions/biome", &b)` for a `rooms.BiomeInfo`
representing `interior` (sky 0.15, lamp 50) and one representing an open biome,
assert neither output contains `TEMPLATE ERROR`, log both, then delete the test
file and prove `git status --short` is clean.

- [ ] **Step 6: Re-record and prove the shape**

Expected: **15 rooms**, all `house -> interior`. `house`'s sky was 0.15 with
lamp 50 and `interior` is the same, so the light values should be **identical**
and the only change is the biome label.

🔴 **If any of the 15 changed its light value, the two biomes disagree** and one
of them is authored wrong. Stop and report the numbers.

- [ ] **Step 7: Commit**

```bash
git add -u
git commit -m "content(biomes): delete house, fold its 15 rooms into interior

house meant a built indoor space and so does interior, which covers temples,
halls, archives and sealed corridors as well as dwellings. One biome rather
than two, and the map symbol carries over.

The biome is DELETED rather than aliased, so every consumer is enumerated
rather than silently redirected. Three greps ran before the change, over the Go
tree, the data tree and the templates, because a template reaches biome methods
by reflection and go build cannot see it.

The light values of all 15 rooms are unchanged: interior carries the same sky
fraction and lamp house did, so only the label moved.

biome_coupling_test.go was proven capable of catching the mistake before the
mistake was fixed: leaving house in surfaceIndoorBiomes with its file deleted
fails TestClassificationMapsHaveNoDeadKeys by name.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: The mislabelled rooms

Five groups labelled as what they resemble rather than what they are.

**Files:** ~69 room YAMLs, both goldens

- [ ] **Step 1: Move each group**

| Rooms | From | To | Identify by |
|---|---|---|---|
| 20 | `city` | `sewer` | every room in `rooms/new_plymouth_sewers/` |
| 31 | `cave` | `interior` | every room in `rooms/crash_site_interior/` |
| 12 | `cave` | `spiderweb` | every room in `rooms/the_foldweave/` |
| 5 | *none* | `interior` | every room in `rooms/instance_arena/` |
| ~13 | `city` | `interior` | **judged by reading**, see below |

For `new_plymouth_temple`, the outdoor rooms **stay `city`**: Temple Gate
Plaza, Temple Courtyard, Censer Court, Garden of Repose, Eastern Processional,
Cloister Walk, Sexton's Walk. The interiors move: Grand Temple Sanctuary, High
Altar, the three Keeper's House rooms, Warden's Cell, Healer's Chapel, Chapel
Recovery Room, Canon's Cell, Canon's Oratory, the two Seminary rooms, the two
Archive rooms, Restricted Collection, Bell Tower Base, Pilgrims' Rest, Almonry.

⚠️ **Read each description before moving it.** The list above is from titles.
A room whose intent is unclear from its description **stays where it is and is
reported**, not guessed.

⚠️ **Also check `new_plymouth_crafting` (27 rooms) and any other `city` zone**
for the same pattern. Report what you find; move only what is unambiguous.

- [ ] **Step 2: Re-record and prove the shape**

This is the largest move in the plan. Expected transitions:

| Transition | Rooms | Expected light change |
|---|---|---|
| `city -> sewer` | 20 | 42ish to **0** at night, and to 0 by day. A sewer has no sky and no lamp |
| `cave -> interior` | 31 | 58 (the lightmod bridge) to **50** by night, 57ish by day |
| `cave -> spiderweb` | 12 | 58 to **0**, until Task 9 gives spiderweb its lamp |
| `none -> interior` | 5 | open-sky default to interior's 50 |
| `city -> interior` | ~13 | 42ish at night to 50; 70 by day to 57ish |

🔴 **CORRECTED after this task ran: the `cave -> spiderweb` rooms do NOT move
here.** They stay at 58. The `lightmod: 2` bridge is additive on
`ActiveMutators` and independent of biome, so exchanging two zero-sky no-lamp
biomes changes nothing. Task 9 moves them, by deleting the mutator's
`lightmod`. Do not author `spiderweb`'s lamp early to force a change here.

⚠️ **Derive the expected numbers yourself from the current golden** rather than
trusting the table, which is approximate. Report predicted against measured.

- [ ] **Step 3: Verify and commit**

Run `go test ./...` and boot the server (a biome id typo in a room YAML panics
at startup, not at build). Commit with the per-group counts and the measured
shape in the message.

---

### Task 6: The orphans

**Files:** 35 room YAMLs, both goldens

- [ ] **Step 1: Move them**

| Rooms | Zone | To | Note |
|---|---|---|---|
| 21 | `endless_trashheap` | `land` | Open wasteland, all 21 titled The Wasteland |
| 6 | `newcomer_antechamber` | `ether` | Replace the explicit `biome: default` |
| 1 | `shadow_realm` (75, Waiting Room) | `ether` | |
| +1 | `shadow_realm` (-1, The Void) | `ether` | **Not a shipped room.** Biome it for consistency; it moves no golden |
| 3 | `instance_planar_oasis` | `ether` **+ room `lamp: 38`** | Its own sky, permanently at twilight |

For the oasis, add a per-room override to each of the three files:

```yaml
lamp: 38
```

🔑 **38 is not arbitrary.** It puts the oasis in the shapes band, where names
hide, and the room text already reads *"Shapes move in the heat haze, some are
mirages, some are not."* `ether`'s biome lamp of 60 would make it fully
readable and contradict its own description.

- [ ] **Step 2: Prove the antechamber can never go dark**

Add a permanent test. This is the one behaviour in 3b that a player would hit
on their very first minute and that nothing else guards.

Put it in `internal/rooms/onboarding_light_test.go`, following the fixture
pattern `lighting_test.go`'s `TestRoomIsLit` already uses (`SeedBiomesForTest`,
`configs.SetConfigForTest`, `gametime.ClearDateCacheForTest`,
`util.SetRoundCountForTest`):

```go
package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TestOnboardingRoomsAreNeverDark pins the constraint that made `ether` exist.
//
// 🔴 look.go refuses to print a room description below LightBlindBelow, and
// these six rooms are character creation: The Threshold, Knowing Yourself,
// What You Carry, The World Speaks, The Proving, The Landing. If they ever
// follow the sun, a share of every day's new players arrive unable to read
// their own introduction, and nothing else in the suite would say so.
//
// Sampled across the year AND the clock, because the failure mode is seasonal.
// A room can read fine at whatever round a test happens to pick and be black
// at midwinter midnight.
func TestOnboardingRoomsAreNeverDark(t *testing.T) {
	etherLamp := 60
	zeroSky := 0.0
	cleanupBiomes := SeedBiomesForTest(map[string]*BiomeInfo{
		"ether": {
			BiomeId: "ether", Name: "Ether", Symbol: "∴",
			SkyLight: &zeroSky, Lamp: &etherLamp, Indoor: true,
		},
	})
	t.Cleanup(cleanupBiomes)

	cfg := configs.GetConfig()
	cfg.Timing.RoundsPerDay = 900
	cfg.Timing.RoundSeconds = 4
	cfg.Timing.Validate()
	cfg.Balance.Validate() // shipped latitude, deliberately not overridden
	configs.SetConfigForTest(t, cfg)

	gametime.ClearDateCacheForTest()
	t.Cleanup(gametime.ClearDateCacheForTest)

	original := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCountForTest(original) })

	dim := configs.GetLightingConfig().DimBelow
	room := Room{Biome: "ether"}

	// Midwinter, equinox and midsummer, at midnight, dawn, noon and dusk.
	for _, doy := range []int{356, 81, 172} {
		for _, hour := range []float64{0, 6, 12, 18} {
			round := uint64(float64(doy-1)*900 + hour*37.5)
			util.SetRoundCountForTest(round)
			gametime.ClearDateCacheForTest()

			if got := room.LightLevel(); got < dim {
				t.Errorf("day %d hour %v: ether reads %d, below DimBelow %d. "+
					"A character-creation room that is not FULLY readable is a "+
					"new player who cannot read their own introduction.",
					doy, hour, got, dim)
			}
		}
	}
}
```

⚠️ **Verify `SeedBiomesForTest`'s signature and `util.SetRoundCountForTest`'s
name against the tree before trusting this**; both were read on 2026-09-23.
If the fixture helper differs, follow the tree and say so.

⚠️ This asserts on the BIOME, not on the six real rooms, because loading real
rooms needs the world fixture. That is deliberate: the biome is what guarantees
the property, and a room override that broke it would be caught by the golden.

- [ ] **Step 3: Prove that test can fail**

Temporarily set `ether`'s lamp to 0 and confirm the test reddens at the night
samples. Restore and confirm clean.

🔴 **If it does not redden, the test is not guarding the thing it claims** and
must be fixed before this task is done.

- [ ] **Step 4: Re-record, prove the shape, verify, commit**

Expected: 21 trashheap rooms take open-sky values; 10 rooms go to `ether`'s
flat 60 except the three oasis rooms at 38. Nothing else moves.

---

### Task 7: The forest split

**Files:** ~12 room YAMLs, both goldens

- [ ] **Step 1: Read all 61 forest room descriptions and split them**

Unambiguously dense, from the spec: Deep Woods, Hidden Grove, Overgrown Hollow,
The Deep Timber, Under the Old Trees, Grove Heart, Whispering Grove, Old Stand.

Unambiguously open: Forest Path, The Forest Edge, and every Camp, Track, Road,
Crossing, Meadow, Clearing, Glade and Pool.

⚠️ **The middle is a reading, not a pattern match.** Briar Tangle, Tangled
Bracken, Veil Hollow, Scrub Tangle, The Den, Den Approach and Lair Approach all
need their descriptions read. **Report your decision and one-line reason for
every room you move**, and leave anything genuinely ambiguous on `forest`.

- [ ] **Step 2: Re-record and prove the shape**

Expected: only `forest -> dense_forest` transitions, each dropping about 7
points (`8 * log2(0.25/0.45)` is about -6.8). **No other biome may appear in
the diff.**

Sanity check one room by hand: a dense_forest room at midwinter noon should
read about **47**, in the shapes band, where the same room read about 54 before.

- [ ] **Step 3: Verify and commit**, listing every moved room with its reason.

---

### Task 8: `plains`, `river`, and the four zone-configs

**Files:** 32 room YAMLs, 4 zone-configs, both goldens

- [ ] **Step 1: Move the rooms**

25 `land` rooms to `plains` across `dustwalk_road` (10), `marches_spur_road`
(8) and `stillwater` (7). 7 `water` rooms to `river` in `watchers_crossing`.

⚠️ **`marches_spur_road` and `stillwater` also contain farmland, cave, city and
swamp rooms.** Move only the `land` ones.

- [ ] **Step 2: Fix the zone-configs**

🔴 **ADDED after Task 5.** Three more zone-configs carry a `defaultbiome`
that is now stale: `new_plymouth_sewers` says `city`, and
`crash_site_interior` and `the_foldweave` say `cave`. All three are dead today,
because every room in those directories carries an explicit `biome:`, but they
are the same latent trap as `plains`: the moment someone adds a room without
one it inherits a wrong biome silently. Point them at `sewer`, `interior` and
`spiderweb` respectively.

Then the four this task was originally scoped for:

`dustwalk_road`, `marches_spur_road` and `stillwater` declare
`defaultbiome: plains`; `watchers_crossing` declares `river`. Those biomes now
exist, so the declarations become valid rather than dead.

Check `newcomer_antechamber`'s `defaultbiome: default` too: point it at `ether`.

- [ ] **Step 3: Re-record and prove the shape**

🔴 **Expected: NOTHING MOVES.** `plains` and `river` carry sky 1.0 with no
lamp, exactly like `land` and `water`. Every one of the 32 rooms should report
the same light under its new biome.

**This is the strongest check in the plan.** If any room's light changes, one
of the four biomes is authored wrong. Report the numbers and stop.

- [ ] **Step 4: Verify and commit**, saying plainly in the message that this
commit changes no light and exists to make the vocabulary honest.

---

### Task 9: Retire the `lightmod: 2` bridge

The 43 rooms held lit by a mutator get honest biome lamps instead.

**Files:** `_datafiles/world/dogmud/mutators/hull_suppression.yaml`,
`_datafiles/world/dogmud/mutators/foldweave_glow.yaml`,
`_datafiles/world/dogmud/biomes/spiderweb.yaml`, both goldens

- [ ] **Step 1: Give `spiderweb` its lamp**

Add `lamp: 45` to `_datafiles/world/dogmud/biomes/spiderweb.yaml`.

🔑 The Foldweave glows. 45 puts it below `interior`'s 50 and in the shapes
band, so a spider lair reads as dim and web-choked rather than as a lit room.

- [ ] **Step 2: Remove `lightmod` from both mutators**

Delete the `lightmod: 2` line from `hull_suppression.yaml` and
`foldweave_glow.yaml`. Leave everything else in those files alone; they carry
conditions and text that have nothing to do with light.

⚠️ **Check whether any OTHER mutator carries `lightmod`** and leave those
alone: the three weather mutators with negative values belong to plan 4.

- [ ] **Step 3: Re-record and prove the shape**

Expected:
- 31 Crash Site rooms: the `+2` bridge term (58) goes away, leaving
  `interior`'s lamp of 50 at night and about 57 by day.
- 12 Foldweave rooms: from **58** to `spiderweb`'s **45**.

  🔴 **CORRECTED 2026-09-23 after Task 5 ran.** This plan predicted they
  would be at 0 by now, because Task 5 moved them from `cave` to `spiderweb`.
  They did not move at all. The `lightmod: 2` bridge is a flat additive term
  computed from `ActiveMutators` in `internal/rooms/lighting.go`, **entirely
  independent of biome**, so swapping one zero-sky no-lamp biome for another
  cannot change the number. Deleting the mutator's `lightmod` is the only
  thing that moves them, which is this task.
- **Nothing else moves.**

- [ ] **Step 4: Confirm the bridge has no load-bearing consumers left**

```bash
grep -rn "lightmod" _datafiles/world/dogmud/mutators/
```
Expected: only the three weather mutators with NEGATIVE values. Record the
result; plan 4 inherits exactly that list.

- [ ] **Step 5: Verify and commit.**

---

### Task 10: Documentation and the patch note

**Files:** `internal/rooms/context.md`, `docs/PATCH_NOTES.md`, `docs/README.md`

- [ ] **Step 1: Update `internal/rooms/context.md`**

Record the new biome set and what each means, that `house` is gone, and the two
per-room override cases now in use (the oasis lamp, and whatever Task 5 needed).

⚠️ **Every symbol you name must exist.** Run `python tools/context_md_audit.py`
and confirm nothing for `rooms`. It has known false positives; read its findings
rather than trusting its exit code.

- [ ] **Step 2: Write the patch note**

Read the `dogmud-player-copy` skill first. 80-character hard wrap, no raw
numbers, ESL-clear, no em dashes or en dashes.

Convey: some woods are darker than others and the deep timber is dim even at
noon in winter; sewers and the inside of the wreck are their own kind of dark;
nothing else a player will notice.

🔴 **Do NOT mention** dazzle, lanterns, light spells, or town streets. The first
two are later plans, and town lighting does not change in 3b.

🔑 **Check the claim against the golden before writing it**, not against this
plan's prose. 3a's patch note shipped a sentence that measurement disproved.

- [ ] **Step 3: Add the patch note row to `docs/README.md`**, and the plan row.

- [ ] **Step 4: Final gate**

```bash
gofmt -l internal/ modules/
go vet ./...
go test ./...
golangci-lint run --new-from-merge-base=origin/master
```
All must be clean, the last reporting `0 issues`.

Boot the server one final time in an isolated worktree.

- [ ] **Step 5: Commit.**

---

## Before opening the PR

Every `gh` command carries `--repo pruuk/DOGMud`. This repo is a fork and `gh`
defaults to the parent.

The PR description must carry:
- The per-task golden shapes, predicted against measured.
- The `biome_coupling_test.go` failure text from Task 4 Step 4, as evidence the
  guard works.
- The sabotage result from Task 6 Step 3, proving the onboarding test can fail.
- The final `lightmod` grep from Task 9 Step 4, which is plan 4's inheritance.
- A statement that `plains` and `river` change no light.

## Known state this PR deliberately leaves behind

Not defects. Put them in the PR description so a reviewer does not file them.

- **Every `city` room still shares one lamp**, so main streets and back lanes
  are equally dim. **Plan 3c.**
- **`fort` still shares one sky fraction** between its open training yard and
  its buried vault. **Plan 3c.**
- **No transition notices** when light changes around a player. **Plan 3d.**
- **`LightMod` still exists** as a mutator field with three negative-value
  weather consumers. **Plan 4.**
- **Dazzle is still inert.** **Plan 5.**
- **The new biomes' `movementcost` matches their sibling exactly.**
  `dense_forest` costs the same to cross as `forest`, and `river` the same as
  `water`. Task 3 chose this deliberately over the plan's draft values: nothing
  justified the deltas, existing biomes always comment a non-default cost, and
  `movementcost` is stamina, so a lighting plan must not retune it silently.
  **Whether dense woods and moving water should cost more to cross is a real
  balance question, and it is open.**
- 📌 **`modules/weather/sim/climate.go` already held a generic `"plains"`
  archetype key** before this arc, which now shares a name with a real biome. It
  resolves correctly, because the per-biome YAML overlays it at load time and a
  second Go key would be a duplicate-key compile error, but the coincidence is
  worth a reviewer's eye.
