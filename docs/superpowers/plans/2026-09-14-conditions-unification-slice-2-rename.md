# Conditions Unification Slice 2 (Rename) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename buffs to conditions in every Go identifier, file, comment, test and player- or admin-facing text, with no behaviour change and nothing changed on disk or on the wire.

**Architecture:** Freeze the wire first (explicit tags on untagged on-disk fields, byte-level freeze tests, template render tests). Move the package with `gopls rename` plus `git mv`. Rename every identifier with a scan-and-rename loop that drives `gopls rename` by byte offset, so only type-checked references change. Then files, comments, text, docs, and an identifier guard that stops the word coming back.

**Tech Stack:** Go, `gopls` v0.21.1 (`~/go/bin/gopls`), `go/parser` and `go/scanner`, yaml.v2, testify.

**Spec:** `docs/superpowers/specs/2026-09-14-conditions-unification-slice-2-rename-design.md` (read the rulings, the facts table and the name map first).

**Branch:** `feature/conditions-unification-slice-2-rename` off master `6f6a64696`.

---

## Facts verified for this plan (2026-09-14)

| Fact | How verified |
|---|---|
| `gopls rename -w <file>:#<offset> <NewName>` takes a **0-based byte offset** (the same as `go/token.Position.Offset`); an offset anywhere inside the identifier works, one byte before it does not | probe on `internal/buffs/ids.go` `BuffIdRecovering` at offsets 458/459/460 |
| One `gopls rename` call takes about 4 to 6 seconds on this repo | timed |
| `gopls rename` refuses a rename that would shadow: renaming the package clause to `conditions` failed because `modules/gmcp/gmcp.Char.go:736` declares a local `conditions` in `buildConditionsPayload`. That is the only identifier named `conditions` in a file importing `internal/buffs` | probe; grep |
| Package rename on Windows: after renaming that local, `gopls rename -w internal/buffs/ids.go:1:9 conditions` rewrote the package clause in every package file and every import path (290 files) but failed to move the directory ("open ...\internal\conditions\buffs.go: The system cannot find the path specified"). `git mv internal/buffs internal/conditions` afterwards gave a clean `go build ./...` and `go vet ./...` | probe in a throwaway worktree |
| After that, `store_text_fields_guard_test.go:27` still names the path `"internal/buffs"` and fails (its pattern matched nothing; reads reported as outside the store); descriptive strings in `messaging_surface_guard_test.go:73-80` and comments name `internal/buffs/...` | probe `go test .` |
| Untagged fields whose yaml key comes from the Go name: `buffs.Buff.BuffId` (`internal/buffs/buffs.go:15`), `buffs.BuffSpec.BuffId` (`buffspec.go:163`), `species.Species.BuffIds` (`internal/species/species.go:35`) | awk scan |
| GMCP payload structs holding buff JSON fields are unexported: `itemUpdateReq` (`gmcp.Item.go:84,113`), `mobUpdateReq` (`gmcp.Mob.go:133`), `mobEnums` (`:165`), `questEnums` (`gmcp.Quest.go:122`) | awk |
| `messaging.Category` has `String()` (`internal/messaging/messaging.go:110`); `CategoryBuffApply` returns `"buff-apply"`, `CategoryBuffExpire` `"buff-expire"` | read |
| Templates reading Go names: `character/status.template:19-20` (`.Character.Buffs.HasBuff 83`, `.Character.Buffs.TriggersLeft 83`), `descriptions/identify.template:19-20,33-34` (`$spec.BuffIds`, `$spec.Damage.CritBuffIds`), `help/species.template:30` (`$speciesInfo.BuffIds`), `character/conditions.template:3` (`.PermaBuff`), in `_datafiles/world/dogmud` and the same species/conditions templates in `_datafiles/world/default` | grep |
| Render data: `status` passes `*users.UserRecord`; `appraise` passes `struct{Item *items.Item; ItemSpec *items.ItemSpec}`; species help passes `[]species.Species` (`getSpeciesOptions`); `conditions` passes `[]conditionEntry` | `internal/usercommands/status.go:13`, `appraise.go:48-56,78`, `help.go:108`, `conditions.go:30` |
| Admin command: `` `buff`: {Buff, false, true, true} `` (`usercommands.go:80`), handler `func Buff` in `admin.buff.go:25`; `keywords.yaml` lists `buff` under `help: admin: all:` (dogmud `:200`, default `:107`) and has `help-aliases:` (`:233`) and `command-aliases:` (`:304`, e.g. `conditions: ['c', 'cond', 'conds']`) | read |
| Identifiers that need a hand-picked name, not the literal swap: `FindBuffed` (`internal/rooms`), `TestDefensiveCaster_FullHP_Unbuffed_CastsCocoonFirst`, and the `Perma[Bb]uff` family (`PermaBuff`, `permaBuffIds`, `reapplyPermabuffs`, `recalcPermaBuffs`, `refreshTestPermaBuffId`, `RemovePermaBuff`, `SetPermaBuffs`, `Permabuffs`); `debuff`/`rebuff` hits are prose or content strings | grep |
| Config fields that keep their names until slice 3: `AllowItemBuffRemoval`, `DeathsShadowBuffId`, `BrokenLimbBuffDuration`; not buff at all: identifiers containing `Buffer`/`buffer` | spec |
| behaviortree tests write temp crate files under gitignored `internal/**/_datafiles/`, which can vanish mid-walk while packages test in parallel | flake seen 2026-09-14 |

## File map

| Path | Change |
|---|---|
| `internal/buffs/` | moves to `internal/conditions/` (Task 1) |
| `internal/buffs/buffs.go`, `buffspec.go`, `internal/species/species.go` | explicit yaml tags (Task 0) |
| `wire_freeze_test.go` (repo root) | NEW (Task 0) |
| `modules/gmcp/gmcp_wire_freeze_test.go` | NEW (Task 0) |
| `internal/usercommands/template_freeze_test.go` | NEW (Task 0) |
| every Go file naming a buff identifier | renamed identifiers (Task 2) |
| buff-named Go files and tests, guard regexes and path strings, comments | Task 3 |
| `internal/usercommands/admin.buff.go`, `usercommands.go`, help templates, `keywords.yaml` (both worlds), web client text | Task 4 |
| `context.md` files, skills, `CLAUDE.md`, `docs/README.md`, `docs/PATCH_NOTES.md` | Task 5 |
| `identifier_word_guard_test.go` (repo root) | NEW (Task 6) |
| `C:/tmp/condrename/` | scratch tool, never committed (Task 2) |

## Conventions for every task

- Bash (Git Bash) for go, gopls and git, from the repo root.
- After every task: `gofmt -l internal/ modules/ .` prints nothing for touched files, `go build ./...`, `go vet ./...`, and `go test ./... -count=1` pass. The Task 0 freeze tests are green at the end of every task; a red freeze test is a slice 2 regression, fixed before continuing.
- `git add` named paths, or for a rename task the exact list from `git status --short` reviewed before staging; never `git add -A` or `git add .`. `git mv` for moves.
- Commits end with `Co-Authored-By:` for your model. No em or en dashes in anything written.
- Null-probe every new assertion (break it, see red for the named reason, restore).
- Never edit anything under `docs/superpowers/*/completed/`, earlier slice specs and plans, or published patch note entries: history keeps its words.

---

### Task 0: Freeze the wire

**Files:**
- Modify: `internal/buffs/buffs.go:15`, `internal/buffs/buffspec.go:163`, `internal/species/species.go:35`
- Create: `wire_freeze_test.go`, `modules/gmcp/gmcp_wire_freeze_test.go`, `internal/usercommands/template_freeze_test.go`

- [ ] **Step 1: Pin the three untagged on-disk fields**

`internal/buffs/buffs.go`, the `Buff` struct: `BuffId         int    // Which buff template does it refer to?` becomes
```go
	BuffId         int    `yaml:"buffid"` // Which buff template does it refer to? The tag pins the save key through the slice 2 rename.
```
`internal/buffs/buffspec.go`, the `BuffSpec` struct: `BuffId        int               // Unique identifier for this buff spec` becomes
```go
	BuffId        int               `yaml:"buffid"` // Unique identifier for this buff spec. The tag pins the file key through the slice 2 rename.
```
`internal/species/species.go:35`: `BuffIds ... // Permabuffs this species always has` gains `` `yaml:"buffids"` `` the same way (read the line; keep its type).

Then read each of those three structs whole and list any other UNTAGGED exported field whose Go name contains `Buff`/`buff` (none expected beyond these). If one exists and its struct is marshaled, pin it the same way and say so in the commit body.

- [ ] **Step 2: Root freeze test**

Create `wire_freeze_test.go` (package `main`, repo root):

```go
package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Conditions unification slice 2 renames buffs to conditions in Go and in
// player-facing text, and must change NOTHING on disk or on the wire (owner
// disk/wire rule, 2026-09-14). These tests pin the literal keys, values and
// strings. They were written before the first rename and must stay green
// through the slice; slice 3 is the one that deliberately changes them.

func TestWireFreeze_CharacterSaveKeys(t *testing.T) {
	defer buffs.SeedConditionRecordsForTest()()

	c := characters.Character{}
	c.Buffs.Validate(true)
	require.True(t, c.Buffs.AddBuffMagnitude(buffs.BuffIdBleeding, 3, -2))
	require.True(t, c.Buffs.AddBuffMagnitude(buffs.BuffIdBleeding, 5, -3))
	c.Buffs.List = append(c.Buffs.List, &buffs.Buff{BuffId: buffs.BuffIdWarcry, PermaBuff: true, TriggersLeft: 1})

	out, err := yaml.Marshal(&c)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))
	held, ok := doc["buffs"].(map[any]any)
	require.True(t, ok, "the character save must keep the `buffs:` key; got top-level keys from:\n%s", out)
	list, ok := held["list"].([]any)
	require.True(t, ok, "the held records must stay under `list:`")
	require.Len(t, list, 2)

	bleed := list[0].(map[any]any)
	assert.EqualValues(t, buffs.BuffIdBleeding, bleed["buffid"], "a record's id must stay under `buffid:`")
	assert.Contains(t, bleed, "triggersleft")
	assert.Contains(t, bleed, "stacks", "stacks must stay under `stacks:`")
	stack := bleed["stacks"].([]any)[0].(map[any]any)
	assert.Contains(t, stack, "roundsleft")
	assert.Contains(t, stack, "amount")

	warcry := list[1].(map[any]any)
	assert.Equal(t, true, warcry["permabuff"], "a permanent record must stay under `permabuff:`")

	var back characters.Character
	require.NoError(t, yaml.Unmarshal(out, &back))
	require.Len(t, back.Buffs.List, 2)
	assert.Equal(t, buffs.BuffIdBleeding, back.Buffs.List[0].BuffId)
	assert.Len(t, back.Buffs.List[0].Stacks, 2)
	assert.True(t, back.Buffs.List[1].PermaBuff)
}

func TestWireFreeze_ConditionFileKeys(t *testing.T) {
	var s buffs.BuffSpec
	doc := "buffid: 950\nname: Probe\ntriggerrate: 1 round\ntriggercount: 2\n" +
		"start_remove_buffs: [3]\neffects:\n  damage_mult: magnitude\nflags:\n  - stacking\n" +
		"tick_pool: health\ntick_from_magnitude: true\n"
	require.NoError(t, yaml.Unmarshal([]byte(doc), &s))
	assert.Equal(t, 950, s.BuffId, "`buffid:` must still name the record")
	assert.Equal(t, []int{3}, s.StartRemoveBuffs, "`start_remove_buffs:` must still parse")
	assert.True(t, s.Effects[buffs.EffectDamageMult].UsesMagnitude)
	assert.Equal(t, []buffs.Flag{buffs.Stacking}, s.Flags)
}

func TestWireFreeze_ShippedConditionFilesLoadInBothWorlds(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	for _, world := range []string{"dogmud", "default"} {
		t.Run(world, func(t *testing.T) {
			cfg := configs.GetConfig()
			cfg.FilePaths.DataFiles = configs.ConfigString(filepath.Join(filepath.Dir(here), "_datafiles", "world", world))
			cfg.Network.LogoutRounds = 3
			configs.SetConfigForTest(t, cfg)
			restore := buffs.SeedBuffsForTest(nil)
			defer restore()
			buffs.LoadDataFiles()
			assert.NotEmpty(t, buffs.GetAllBuffIds(), "the %s world's condition files must load under today's keys", world)
		})
	}
}

func TestWireFreeze_SpeciesAndSpellValues(t *testing.T) {
	var sp species.Species
	require.NoError(t, yaml.Unmarshal([]byte("name: Probe\nbuffids: [7]\n"), &sp))
	assert.Equal(t, []int{7}, sp.BuffIds, "species `buffids:` must still parse")

	var spell spells.SpellData
	require.NoError(t, yaml.Unmarshal([]byte("spellid: probe\nname: Probe\neffect_type: buff\nbuff_ids: [3]\n"), &spell))
	assert.Equal(t, "buff", spell.EffectType, "the `effect_type: buff` value is wire and stays")
	assert.Equal(t, []int{3}, spell.BuffIds, "`buff_ids:` must still parse")
}

func TestWireFreeze_MessagingCategoryStrings(t *testing.T) {
	assert.Equal(t, "buff-apply", messaging.CategoryBuffApply.String())
	assert.Equal(t, "buff-expire", messaging.CategoryBuffExpire.String())
}
```

Before running, read and confirm: `buffs.SeedBuffsForTest(nil)` restores correctly when `LoadDataFiles` replaces the map (if `LoadDataFiles` assigns the package map, the returned restore puts the original back; if not, save and restore with the same helper the records test uses); `species.Species` has a `Name` field and `BuffIds` (adjust the literal YAML to the fields that exist); `spells.SpellData` field names `EffectType` and `BuffIds`. Adapt the literal to what the code declares; the assertions on keys and values must not change.

- [ ] **Step 3: GMCP freeze test**

Create `modules/gmcp/gmcp_wire_freeze_test.go`:

```go
package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Slice 2 of the conditions unification must not change any GMCP JSON field
// name; the web client and Mudlet packages read them. See wire_freeze_test.go
// at the repo root.
func TestWireFreeze_GMCPJSONFieldNames(t *testing.T) {
	keysOf := func(v any) map[string]any {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		m := map[string]any{}
		require.NoError(t, json.Unmarshal(b, &m))
		return m
	}
	assert.Contains(t, keysOf(itemUpdateReq{}), "buffIds")
	assert.Contains(t, keysOf(itemUpdateReq{}), "wornBuffIds")
	assert.Contains(t, keysOf(mobUpdateReq{}), "buffIds")
	assert.Contains(t, keysOf(mobEnums{}), "buffs")
	assert.Contains(t, keysOf(questEnums{}), "buffs")
	assert.Contains(t, keysOf(GMCPCondition{}), "duration_cur")
}
```

If any of these types contain fields that fail to marshal at the zero value (e.g. channels), marshal a one-field copy instead; do not drop an assertion.

- [ ] **Step 4: Template render freeze test**

Create `internal/usercommands/template_freeze_test.go`. It renders the four templates that read Go names, through `templates.Process` with the real DOGMud template files, and asserts `err == nil` and a substring that only appears when the Go names resolved:

```go
package usercommands

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Templates read Go field and method names at runtime, so a rename the
// compiler accepts can still break a template silently. These renders pin the
// four templates that read condition names (slice 2 of the conditions
// unification).

func useDogmudTemplates(t *testing.T) {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = configs.ConfigString(filepath.Join(filepath.Dir(here), "..", "..", "_datafiles", "world", "dogmud"))
	configs.SetConfigForTest(t, cfg)
}

func TestTemplateFreeze_ConditionsListReadsPermanent(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer buffs.SeedConditionRecordsForTest()()
	useDogmudTemplates(t)

	out, err := templates.Process("character/conditions", []conditionEntry{
		{Name: "Bleeding (2)", Description: "Wounds seeping blood.", RoundsLeft: 4},
		{Name: "Stoneskin", Description: "Skin like rock.", PermaBuff: true},
	}, 0)
	require.NoError(t, err)
	assert.Contains(t, out, "Bleeding (2)")
	assert.Contains(t, out, "Stoneskin")
}

func TestTemplateFreeze_StatusReadsTheBrokenLimbRecord(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	useDogmudTemplates(t)

	user, _ := getTestUserAndRoom(t)
	restore := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		83: {BuffId: 83, Name: "Broken Limb", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 10},
	})
	defer restore()
	user.Character.Buffs.Validate(true)
	require.True(t, user.Character.Buffs.AddBuff(83, false))

	out, err := templates.Process("character/status", user, user.UserId)
	require.NoError(t, err)
	assert.Contains(t, out, "Broken limb", "status reads .Character.Buffs.HasBuff and .TriggersLeft")
}

func TestTemplateFreeze_IdentifyReadsConditionIds(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	useDogmudTemplates(t)
	restore := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		940: {BuffId: 940, Name: "Probe Glow", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 5},
		941: {BuffId: 941, Name: "Probe Rend", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 5},
	})
	defer restore()

	spec := items.ItemSpec{ItemId: 99940, Name: "probe", BuffIds: []int{940}}
	spec.Damage.CritBuffIds = []int{941}
	item := items.Item{ItemId: 99940}
	out, err := templates.Process("descriptions/identify", struct {
		Item     *items.Item
		ItemSpec *items.ItemSpec
	}{&item, &spec}, 0)
	require.NoError(t, err)
	assert.Contains(t, out, "Probe Glow", "identify reads $spec.BuffIds")
	assert.Contains(t, out, "Probe Rend", "identify reads $spec.Damage.CritBuffIds")
}

func TestTemplateFreeze_SpeciesHelpReadsConditionIds(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	useDogmudTemplates(t)
	restore := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		942: {BuffId: 942, Name: "Probe Hide", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 5},
	})
	defer restore()

	out, err := templates.Process("help/species", []species.Species{{Name: "Probe", BuffIds: []int{942}}}, 0)
	require.NoError(t, err)
	assert.Contains(t, out, "Probe Hide", "species help reads $speciesInfo.BuffIds")
}
```

Before running, read and adapt to reality without weakening the assertions: how `templates.Process` resolves files in a usercommands test binary (if a registered default-world filesystem masks the DOGMud file, the `Broken limb` and `Applies`/`Crits Apply` rows prove which template rendered; if they are missing, find the override the templates package tests use, `u8_help_test.go:useU8DataFilesAt`, and apply the equivalent here); whether `identify` needs more fields on the spec or item to reach the `BuffIds` rows (read the template); whether `species.Species` needs slots or a size for the template to render. A template that renders but prints nothing for the probe id is a FAIL, not a fixture problem to paper over.

- [ ] **Step 5: Run, then null-probe**

Run: `go test . -run TestWireFreeze -count=1`, `go test ./modules/gmcp/ -run TestWireFreeze -count=1`, `go test ./internal/usercommands/ -run TestTemplateFreeze -count=1`.
Expected: PASS.

Probes (restore each): change the `Buff.BuffId` tag to `yaml:"conditionid"`: `TestWireFreeze_CharacterSaveKeys` and `ConditionFileKeys` red. Rename the json tag `buffIds` on `itemUpdateReq`: the GMCP test red. In `status.template` change `.Character.Buffs.HasBuff 83` to `.Character.Buffs.HasBuffX 83`: the status render red (error or missing text).

- [ ] **Step 6: Full gate and commit**

Full gate per conventions. Commit the three tags and three test files:
```bash
git add internal/buffs/buffs.go internal/buffs/buffspec.go internal/species/species.go wire_freeze_test.go modules/gmcp/gmcp_wire_freeze_test.go internal/usercommands/template_freeze_test.go
git commit -F - <<'EOF'
test(conditions): freeze the wire before the slice 2 rename (explicit tags on three untagged on-disk fields, byte-level key tests, GMCP JSON names, template renders)

Probes: <record>

Co-Authored-By: <your model> <noreply@anthropic.com>
EOF
```

---

### Task 1: Move the package

**Files:** `modules/gmcp/gmcp.Char.go:736`; `internal/buffs/` to `internal/conditions/`; every importer; path strings in Go.

- [ ] **Step 1: Remove the one shadowing local**

```bash
off=$(grep -bo 'conditions := make(map\[string\]GMCPCondition)' modules/gmcp/gmcp.Char.go | head -1 | cut -d: -f1)
gopls rename -w "modules/gmcp/gmcp.Char.go:#$off" held
go build ./modules/gmcp/
```
Expected: builds; the local is `held` everywhere in `buildConditionsPayload`.

- [ ] **Step 2: Rename the package clause (rewrites clauses and import paths)**

```bash
off=$(grep -bo 'package buffs' internal/buffs/ids.go | head -1 | cut -d: -f1)
gopls rename -w "internal/buffs/ids.go:#$((off+8))" conditions 2>&1 | tail -3
```
Expected on Windows: it reports it could not open `internal\conditions\...` (it cannot move the directory). Confirm every file in `internal/buffs/*.go` now starts `package conditions`: `grep -L '^package conditions' internal/buffs/*.go` prints nothing. If gopls refuses for a shadowing reason, rename that local as in Step 1 and repeat.

- [ ] **Step 3: Move the directory**

```bash
git mv internal/buffs internal/conditions
go build ./... && go vet ./...
```
Expected: clean.

- [ ] **Step 4: Path strings**

`grep -rn 'internal/buffs' --include=*.go .` and update every hit that is a PATH (e.g. `store_text_fields_guard_test.go:27` `{"internal/buffs", ...}`, descriptive strings in `messaging_surface_guard_test.go`, comments naming files) to `internal/conditions`; keep the guard's human label text consistent. Also `grep -rn 'internal/buffs' --include=*.md internal modules .claude CLAUDE.md` is Task 5's; leave it.

- [ ] **Step 5: Gate and commit**

Full gate (including the Task 0 freeze tests). Stage from `git status --short` (the `git mv` renames, the gopls-edited importers, the path strings), review the list, commit `refactor(conditions): move internal/buffs to internal/conditions`.

---

### Task 2: Rename every identifier

**Files:** every Go file declaring or referencing an identifier that contains `buff`/`Buff`; the four templates.

- [ ] **Step 1: Build the scratch scanner (not committed)**

Create `C:/tmp/condrename/main.go`:

```go
// condrename lists every Go DECLARATION whose name contains "buff" (any case),
// excluding "buffer", a fixed keep list, and skipped positions, as
// "<path>:#<offset> <name>", one per line, sorted. gopls rename on a
// declaration renames every reference, so renaming the first line and
// rescanning converges.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	word    = regexp.MustCompile(`(?i)buff`)
	notBuff = regexp.MustCompile(`(?i)buffer`)
	keep    = map[string]bool{
		// config fields rename with their keys in slice 3
		"AllowItemBuffRemoval": true, "DeathsShadowBuffId": true, "BrokenLimbBuffDuration": true,
	}
)

func main() {
	root := flag.String("root", ".", "repo root")
	skipPath := flag.String("skip", "", "file of skipped '<path>:#<offset>' keys")
	flag.Parse()

	skip := map[string]bool{}
	if *skipPath != "" {
		if f, err := os.Open(*skipPath); err == nil {
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				if fields := strings.Fields(sc.Text()); len(fields) > 0 {
					skip[fields[0]] = true
				}
			}
			f.Close()
		}
	}

	type hit struct{ key, name string }
	var hits []hit
	fset := token.NewFileSet()
	_ = filepath.WalkDir(*root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			n := d.Name()
			if path != *root && (n == ".git" || n == "node_modules" || n == "_datafiles" || strings.HasPrefix(n, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil
		}
		add := func(id *ast.Ident) {
			if id == nil || id.Name == "_" || !word.MatchString(id.Name) || notBuff.MatchString(id.Name) || keep[id.Name] {
				return
			}
			key := fmt.Sprintf("%s:#%d", filepath.ToSlash(path), fset.Position(id.Pos()).Offset)
			if !skip[key] {
				hits = append(hits, hit{key, id.Name})
			}
		}
		addLHS := func(exprs []ast.Expr) {
			for _, e := range exprs {
				if id, ok := e.(*ast.Ident); ok {
					add(id)
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncDecl:
				add(x.Name)
			case *ast.TypeSpec:
				add(x.Name)
			case *ast.ValueSpec:
				for _, id := range x.Names {
					add(id)
				}
			case *ast.Field:
				for _, id := range x.Names {
					add(id)
				}
			case *ast.AssignStmt:
				if x.Tok == token.DEFINE {
					addLHS(x.Lhs)
				}
			case *ast.RangeStmt:
				if x.Tok == token.DEFINE {
					addLHS([]ast.Expr{x.Key, x.Value})
				}
			case *ast.LabeledStmt:
				add(x.Label)
			}
			return true
		})
		return nil
	})
	sort.Slice(hits, func(i, j int) bool { return hits[i].key < hits[j].key })
	for _, h := range hits {
		fmt.Println(h.key, h.name)
	}
}
```

Build it outside the module: `cd C:/tmp/condrename && go mod init condrename && go build -o condrename.exe . && cd -`. Run `C:/tmp/condrename/condrename.exe -root . | wc -l` and record the starting count.

- [ ] **Step 2: The name function**

```bash
newname() {
  case "$1" in
    PermaBuff) echo Permanent; return;;
    FindBuffed) echo FindWithConditions; return;;
    TestDefensiveCaster_FullHP_Unbuffed_CastsCocoonFirst) echo TestDefensiveCaster_FullHP_WithoutConditions_CastsCocoonFirst; return;;
  esac
  echo "$1" | sed -e 's/Perma[Bb]uff/PermanentCondition/g; s/perma[Bb]uff/permanentCondition/g; s/BUFF/CONDITION/g; s/Buff/Condition/g; s/buff/condition/g'
}
```
Any other name matching `(?i)debuff|buffed|rebuff|unbuff` that the scanner lists is added to the `case` with a hand-chosen name (a debuff is a harmful condition: `debuff` becomes `harmfulCondition`); record each in the commit body.

- [ ] **Step 3: The rename loop**

```bash
SKIP=C:/tmp/condrename/skip.txt; : > "$SKIP"; LOG=C:/tmp/condrename/failures.log; : > "$LOG"
n=0
while line=$(C:/tmp/condrename/condrename.exe -root . -skip "$SKIP" | head -1); [ -n "$line" ]; do
  pos=${line% *}; name=${line##* }; new=$(newname "$name")
  if [ "$new" = "$name" ]; then echo "$pos $name (no mapping)" >> "$SKIP"; continue; fi
  if ! out=$(gopls rename -w "$pos" "$new" 2>&1); then
    echo "$pos $name -> $new : $out" >> "$LOG"; echo "$pos $name" >> "$SKIP"
  fi
  n=$((n+1)); if [ $((n % 40)) -eq 0 ]; then go build ./... || break; fi
done
go build ./... && go vet ./...
```
Run it with the Bash tool's `run_in_background: true` (it takes a while) and wait for completion. Expected at the end: the scanner prints nothing except skipped entries; `failures.log` lists refusals. A refused identifier can reappear at a new offset after a later rename edits its file and be refused again; that only duplicates log lines (the loop still terminates, because every successful rename removes a name for good). If the loop runs more than twice the starting scanner count, stop it and resolve the logged refusals (Step 4) before restarting.

- [ ] **Step 4: Resolve every refusal by hand**

For each line in `failures.log`: read the reason (usually a shadowing or duplicate declaration, e.g. a local `condition` already exists, or `conditions` shadows the package). Choose the nearest clear name (`held`, `record`, `spec`, `heldConditions`), run `gopls rename -w <pos> <chosen>`, and note it. Rerun the loop (Step 3) until the scanner output is empty. Offsets in `skip.txt` go stale after other renames; clear `skip.txt` before each rerun.

- [ ] **Step 5: Templates**

Update the Go names templates read, in `_datafiles/world/dogmud/templates` and `_datafiles/world/default/templates`: `.Character.Buffs.HasBuff` to `.Character.Conditions.HasCondition`, `.Buffs.TriggersLeft` to `.Conditions.TriggersLeft`, `.BuffIds` to `.ConditionIds`, `.Damage.CritBuffIds` to `.Damage.CritConditionIds`, `.PermaBuff` to `.Permanent`. Confirm each new name exists in Go by grep. Template FUNCTION names `buffname`/`buffduration` do not change (disk/wire rule). Then `grep -rnoE '\.[A-Za-z]*[Bb]uff[A-Za-z]*' _datafiles/world/*/templates` must print nothing (run standalone).

- [ ] **Step 6: Verify nothing on the wire moved**

Full gate. The Task 0 freeze tests must pass unchanged in meaning (gopls will have renamed their Go references; read the diff of the three freeze test files and confirm no literal key, string or JSON name changed). `git diff --stat` must show no `_datafiles` change other than the template field references.

- [ ] **Step 7: Commit**

Review `git status --short` (expect a few hundred Go files and the templates), stage them by the listed paths, commit `refactor(conditions): rename every buff identifier to condition (gopls, type-checked)` with the starting scanner count, the refusals and chosen names, and the special-case names in the body.

---

### Task 3: Files, tests, comments, guards

- [ ] **Step 1: Rename Go files**

`git ls-files '*.go' | grep -iE 'buff'` lists them (expect `internal/conditions/buffs.go`, `buffspec.go`, their tests, `internal/hooks/Buff_ApplyBuffs.go`, `NewTurn_PruneBuffs.go`, `internal/hooks/*buff*_test.go`, root `buff_*_guard_test.go`, `internal/characters/buffs.go`, `internal/characters/*buff*_test.go`, behaviortree `*buff*` tests). `git mv` each with the literal swap (`buffs.go` becomes `conditions.go`, `buffspec.go` becomes `conditionspec.go`, `Buff_ApplyBuffs.go` becomes `Condition_ApplyConditions.go`, `NewTurn_PruneBuffs.go` becomes `NewTurn_PruneConditions.go`, `buff_apply_path_guard_test.go` becomes `condition_apply_path_guard_test.go`). EXCEPT files named after the `melee_self_buff` behaviour category (`internal/behaviortree/melee_self_buff_*_test.go`): the category string is wire; keep those names. If a target name already exists (e.g. `internal/conditions/conditions.go` from another file), pick a descriptive one and say so.

- [ ] **Step 2: Guards that name things by text**

Root guards match names with regexes and key allowlists by `file|line`:
- the apply-path guard's `buffAddCallPattern` (`\.(AddBuff(?:Scaled|Magnitude)?)\(`) becomes `\.(AddCondition(?:Scaled|Magnitude)?)\(`, and every message and comment in that guard follows;
- every allowlist key naming a renamed file (e.g. `internal/hooks/Buff_ApplyBuffs.go|104`) follows the file rename; run the guard and re-key lines that moved;
- grep every `*_guard_test.go` and `*_test.go` for string literals naming old identifiers or files (`grep -rnE '"[^"]*(Buff|buffs\.)[^"]*"' --include=*_test.go .`) and update those that name Go symbols or Go files (not wire strings such as `buff-apply`, `effect_type`, YAML keys).

- [ ] **Step 3: Comments and doc comments**

`grep -rniE '\bbuffs?\b|\bBuff[A-Za-z]*\b' --include=*.go .` (comments and strings remain). Rewrite prose "buff"/"buffs" meaning the concept to "condition"/"conditions"; references to renamed symbols to their new names; keep quoted wire spellings (`buffs:`, `buffid`, `effect_type: buff`, `fg="buff"`, `buff-apply`, `start_remove_buffs`, `melee_self_buff`, `_datafiles/.../buffs/`). Fix `internal/hooks/NewRound_DoCombat_helpers.go` "Bleeding out = automatic concentration break" to "A character at zero health (IsDisabled) breaks concentration automatically; there is no bleeding-out state."

- [ ] **Step 4: Gate and commit**

Full gate. Commit `refactor(conditions): rename buff files, guard patterns and comments`.

---

### Task 4: Player- and admin-facing text

Load `.claude/skills/dogmud-player-copy/SKILL.md`.

- [ ] **Step 1: The admin command**

Rename the handler (already `Condition` after Task 2? if the scanner renamed `func Buff` to `func Condition`, rename it to `SetCondition` with gopls) and its file `admin.buff.go` to `admin.setcondition.go`. In `internal/usercommands/usercommands.go` the registration becomes `` `setcondition`: {SetCondition, false, true, true}, // Admin only `` and a second entry `` `buff`: {SetCondition, false, true, true}, // Admin only: alias until conditions slice 3 `` (read how the map and `command-aliases` in `keywords.yaml` interact; if aliases for admin commands are expressed in `keywords.yaml` `command-aliases:`, use that instead of a second map entry, and verify with a test that `buff` dispatches to the same handler). Its user-visible strings ("put buff on target", "search for buff", table headers, "Buff %d (%s) applied to %s.", the stacking refusal) say "condition". Rename `command.buff.template` to `command.setcondition.template` in both worlds, rewrite its text, list `setcondition` instead of `buff` under `help: admin: all:` in both `keywords.yaml`, and make `help buff` resolve to it through `help-aliases:`. Update `TestAdminBuff_*` tests (renamed in Task 2/3) to drive `setcondition` and one to drive the `buff` alias.

- [ ] **Step 2: Help and other templates**

For every template under `_datafiles/world/{dogmud,default}/templates` containing the word `buff` in player-visible text (`grep -rliE '\bbuff' ...`, 29 in dogmud at spec time): rewrite to "condition" or plain English that reads well ("rally strengthens your allies"), no raw numbers, 80-column wrap. Keep `buffname`/`buffduration` function calls and `fg="buff"` colour tags (wire). Keep the help file names that are commands or spells.

- [ ] **Step 3: Go-side messages and the web client**

`grep -rniE '"[^"]*\bbuffs?\b[^"]*"|`[^`]*\bbuffs?\b[^`]*`' --include=*.go internal modules` for strings shown to players or admins (messages, admin output, log lines an admin reads) and rewrite the wording; keep wire strings. In `_datafiles/html`, change visible text that says buff (not identifiers, GMCP field names, or CSS classes the markup depends on).

- [ ] **Step 4: Gate and commit**

Full gate plus `node tools/webclient-tests/*.js`. One patch note entry (Task 5 writes it). Commit `feat(conditions): player and admin text say condition; admin command setcondition (buff kept as alias)`.

---

### Task 5: Documentation

- [ ] **Step 1: The package doc**

`internal/conditions/context.md`: every symbol and file name updated (verify each with grep); add a short section "Not these conditions": behaviour tree condition nodes (`internal/behaviortree/conditions_*.go`), quest conditions, web client trigger conditions, bounty `Condition`. Keep the wire spellings where the doc describes YAML or saves, and say slice 3 renames them.

- [ ] **Step 2: Everything else that names live symbols**

`grep -rlE '\b(buffs\.|BuffSpec|AddBuff[A-Za-z]*|HasBuff[A-Za-z]*|RemoveBuff|CancelBuffsWithFlag|BuffId[A-Za-z]*|events\.Buff|internal/buffs|PermaBuff|Buff_ApplyBuffs|PruneBuffs)\b' --include=*.md internal modules .claude CLAUDE.md docs/guides docs/schemas` and update each (not history under `docs/superpowers`). Run `python tools/context_md_audit.py`; no new phantoms in touched packages. Update `docs/README.md` rows for renamed indexed files, and add the Task 6 guard and this plan if not present. Add a `docs/PATCH_NOTES.md` entry: help and messages now call these effects conditions; nothing about them plays differently.

- [ ] **Step 3: Commit**

`docs(conditions): documentation speaks conditions`.

---

### Task 6: The identifier guard

- [ ] **Step 1: Write the guard**

Create `identifier_word_guard_test.go` (repo root). It walks every `.go` file under the repo (skipping `.git`, `_datafiles`, `node_modules`, dot directories), tokenizes with `go/scanner` (so comments and string literals, including struct tags, are never IDENT tokens), and fails on any `token.IDENT` containing `buff` in any case, except names matching `(?i)buffer` and the allowlist `AllowItemBuffRemoval`, `DeathsShadowBuffId`, `BrokenLimbBuffDuration` (config fields renamed with their keys in slice 3). A file that vanishes during the walk (`os.IsNotExist`) is skipped, because behaviortree tests write temp files under gitignored `internal/**/_datafiles/`. It must also fail if it scanned zero files. Failure message: `<file>:<line>: identifier <name> still says buff; slice 2 of the conditions unification renamed these (docs/superpowers/specs/2026-09-14-conditions-unification-slice-2-rename-design.md)`.

- [ ] **Step 2: Probe and commit**

Run: `go test . -run TestNoIdentifierSaysBuff -count=1` PASS. Probe: add `var probeBuff = 1` to a non-test file, see red naming it, remove. Commit `test(conditions): guard that no Go identifier says buff`.

---

### Task 7: Gate and ship

Load `dogmud-shipping` and `dogmud-playtesting`.

- [ ] **Step 1:** Whole-branch review (superpowers:requesting-code-review) against `6f6a64696..HEAD` with the spec; the reviewer diffs `git diff 6f6a64696..HEAD -- _datafiles` to confirm only templates, help text and `keywords.yaml` changed there, and `git show HEAD:_datafiles/config.yaml` equals master's. Fix every finding.
- [ ] **Step 2:** Full local gate; boot check in a detached worktree (both worlds' condition files load, zero panics, Server Ready).
- [ ] **Step 3:** Smoke playtest, single agent per `/playtest local` with an ephemeral goals file under `tools/playtest/goals/`: `early` profile at room 3015 (a bleed, `conditions` shows it), then an `admin` profile run: `setcondition list`, `setcondition search bleed`, `buff list` (alias), `help setcondition`, `help buff`, `help rally`. Expected: identical behaviour, new wording, no template errors. Extract findings to memory.
- [ ] **Step 4:** Push, PR with `--repo pruuk/DOGMud`, body noting no behaviour or wire change, the freeze tests, the guard, the admin command rename and alias, and slice 3 as next. Merge `--merge` when CI is green and every workflow ran. Update memory.
