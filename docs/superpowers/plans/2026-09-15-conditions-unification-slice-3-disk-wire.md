# Conditions Unification Slice 3 (Disk and Wire) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename every buff spelling that a file on disk, a client or a content author reads to condition, migrate existing saves on first boot, delete three dead config knobs and three dead docs, and extend the guard so the word cannot come back.

**Architecture:** One case-preserving word map in a new package `internal/conditionrename` is the single source of truth. A throwaway tool applies it to every tracked data, template, web and Go-string file in one atomic commit; a 0.17.0 startup migration uses the same map for the key names it renames along fixed paths in player saves, alts and room instances. A one-time equivalence proof and a committed key-binding test show nothing but spellings changed.

**Tech Stack:** Go 1.25, `gopkg.in/yaml.v2` (saves and loaders), `gopkg.in/yaml.v3` (proof only), `go/scanner`, the existing `internal/migration` framework.

**Spec:** `docs/superpowers/specs/2026-09-15-conditions-unification-slice-3-disk-wire-design.md` (owner-approved 2026-09-15). Task 0 corrects it with the planning findings below.

---

## Planning findings (verified against source 2026-09-15, not in the spec)

| Finding | Where |
|---|---|
| Shop living-state files (`shops/**`) carry NO buff key: they save `gold`, `inventory[].item_id` etc. `ShopItem.ConditionId` (`buffid`) lives on `Character.Shop`, i.e. mob templates and, in principle, a player save | `internal/characters/character.go:129`, `internal/characters/shop.go:21`; a shop file read directly |
| MiscData key `pinnacle_bandolier_buffs` is written into `Character.MiscData` (saved as `miscdata`) | `internal/hooks/pinnacle_tick.go:279,286,326,365,387`; `character.go:318` |
| Mutation effect types in data: `on_hit_buff` (3 files), `aura_ally_buff` (2), `aura_enemy_debuff` (3), `on_reflect_buff` (3); read at `internal/mutations/aura.go:13,31`, `describe.go:57,59,61,100`, `mutations.go:630,651`. **Owner: all become `*_condition`** | `git grep` |
| Behaviour tree category `buff_friendly` (`archetypes/support_caster.yaml:32`) matches no spell's `categories:`. **Owner: rename to `condition_friendly`, leave inert; wiring deferred to the behaviour arc** | `git grep` |
| Behaviour tree YAML writes nodes as `check: mob_has_buff`, `do: add_buff`, param `buff_id: 9` | `archetypes/ambusher.yaml:23-24` |
| Runtime-only strings: `awareness_buff_mirror` (`internal/hooks/Awareness_Cascades.go:44`), perception triggers `buff_applied`/`buff_expired` (`internal/state/perception/transitions.go:22-23`), dead `BuffOverrides` warning (`modules/weather/engine/apply.go:196`, `ApplyConditionOverrides` has no caller) | `git grep` |
| `tools/id_inventory.py:45` maps type `buffs` to folder `buffs`; `tools/casing_sweep/main.go:35` names `_datafiles/world/dogmud/buffs` | read |
| 482 Go string literals contain buff (not buffer) across 130 files; ~70 in non-test code, all enumerated by the scan in Task 3 | `go/scanner` STRING scan |
| Words that contain buff but are not the concept: `buffer`, `buffet(s)`/`buffeted` (weather prose), `buffed` (room 5203 "buffed to a dull shine"), `rebuff` (mob 357 comment), `Buffalo` (`internal/gametime/zodiac.go:47`) | `git grep` |
| The newcomer dialogue deliberately accepts the player keyword `"buff"` (`_datafiles/world/dogmud/dialogue/newcomer_antechamber/9491.yaml:37`); it stays | read |
| Admin alias lines: `keywords.yaml` dogmud `:299` (help) `:322` (command), default `:162`, `:184`; alias prose in `templates/help/setcondition.template:26` and `templates/admincommands/help/command.setcondition.template:18-19`, both worlds; alias tests `internal/keywords/keywords_setcondition_alias_test.go`, `internal/usercommands/usercommands_test.go:1972-2025, 3186-3220, 4414-4437` | read |
| `identifier_word_guard_test.go` `templateBuffFieldAllowlist` has two entries (`room.data.html|.buffids`, `build.html|.buffIds`) that go stale the moment the rewrite runs, and its stale check fails on them | `identifier_word_guard_test.go:174-177,344-353` |
| yaml.v2 decoding into `yaml.MapSlice` keeps nested mappings as `MapSlice` (order preserved) | `vendor/gopkg.in/yaml.v2/decode.go:693-720` |
| `internal/migration` may use `os.WriteFile` (durable-write guard allowlists it) | `durable_write_guard_test.go:28` |
| Room instance saves marshal `Room` with `containers:` at top level; container locks carry `trapbuffids` | `internal/rooms/save_and_load.go:422`, `rooms.go:95`, `container.go:9` |

## Where the work happens

The owner's main checkout (`C:/Users/Calabe Davis/workspace/DOGMud`) keeps `_datafiles/config.yaml` under skip-worktree with local edits, and the owner may run a server from it. **All implementation runs in a worktree at `C:/tmp/dogmud-slice3`** on `feature/conditions-unification-slice-3-disk-wire`. Its `config.yaml` is the committed version with no skip-worktree bit, so config edits commit normally. Never edit files in the main checkout during this plan.

## File structure

| File | Responsibility |
|---|---|
| Create `internal/conditionrename/rename.go` | The word map: `Apply`, `HasBuff` |
| Create `internal/conditionrename/rename_test.go` | Every mapping, protection and idempotency |
| Create `internal/conditionrename/context.md` | Package doc (house rule) |
| Create (throwaway, never committed) `tools/slice3rewrite/main.go` | Applies `Apply` to tracked data/web files and Go STRING tokens |
| Create (throwaway, never committed) `tools/slice3proof/main.go` | Master-versus-branch equivalence proof |
| Create `condition_keys_bind_test.go` (root) | Every shipped file that uses a renamed key binds it to a non-empty Go field |
| Rewrite `wire_freeze_test.go` (root) | Pins the new keys, values and strings; asserts the old ones are gone |
| Create `internal/migration/0.17.0.go`, `0.17.0_test.go` | Startup key migration |
| Modify `internal/migration/migration.go`, `main.go` | Wire 0.17.0, bump `VERSION` |
| Modify `identifier_word_guard_test.go` | Drop stale allowlists (Task 3, Task 5), add the string/data guard (Task 7) |
| Everything else | Mechanical rewrite (Task 3), hand edits listed per task |

---

### Task 0: Worktree and spec corrections

**Files:**
- Modify: `docs/superpowers/specs/2026-09-15-conditions-unification-slice-3-disk-wire-design.md`

- [ ] **Step 1: Move the main checkout off the branch without touching the owner's config**

The branch is checked out in the main checkout, so it cannot also be checked out in a worktree. Switching the main checkout to `master` needs the committed `config.yaml` on disk for a moment; the owner's copy is restored byte-identical afterwards.

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
S="C:/Users/CALABE~1/AppData/Local/Temp/claude/C--Users-Calabe-Davis-workspace-DOGMud/f7132cac-2960-469a-8284-b8060a3fcd36/scratchpad"
cp _datafiles/config.yaml "$S/config.yaml.local-backup"
sha256sum _datafiles/config.yaml
git update-index --no-skip-worktree _datafiles/config.yaml
git checkout -- _datafiles/config.yaml
git checkout master
cp "$S/config.yaml.local-backup" _datafiles/config.yaml
git update-index --skip-worktree _datafiles/config.yaml
sha256sum _datafiles/config.yaml
git ls-files -v _datafiles/config.yaml
git status --short
```

Expected: both `sha256sum` lines print the same hash; `ls-files -v` prints `S _datafiles/config.yaml`; `git status --short` prints nothing; branch is `master`.

- [ ] **Step 2: Create the worktree**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
git worktree add C:/tmp/dogmud-slice3 feature/conditions-unification-slice-3-disk-wire
cd C:/tmp/dogmud-slice3 && git log --oneline -2 && git status --short
```

Expected: HEAD is the spec commit on top of `a4c078116`; clean tree.

- [ ] **Step 3: Correct the spec**

In the spec, make these edits (all under "Design"):

1. Section 2 targets table: delete the `shops/**/*.yaml` row. Replace the `users/<id>.yaml` key paths cell with:
   `character.buffs` → `conditions`; each `conditions.list[]` entry `buffid`, `permabuff`; `character.pet.buffids`; `character.shop[].buffid`; `character.miscdata.pinnacle_bandolier_buffs`
2. Section 2 tests paragraph: replace "a room instance with a trapped container, a shop" with "a room instance with a trapped container".
3. Section 4 rehearsal bullet: replace "`rooms.instances/`, `shops/`" with "`rooms.instances/`".
4. Section 4 rollout bullet: replace "`users/`, `rooms.instances/` and `shops/`" with "`users/` and `rooms.instances/`".
5. Section 1 "Content rewrite": replace the sentence "applies the table as text edits anchored to each entry's context" with "applies the word map (`conditionrename.Apply`) to the text of every tracked data, template and web file and to every Go string literal; the map is case-preserving and protects words that contain buff but are not the concept (buffer, buffet, buffed, rebuff, Buffalo)". Delete the sentence "Each entry carries the context it applies in (...) so the bare quest `buff` key and the save `buffs` key cannot be renamed anywhere else." and replace "Two lists, both from the name map: **keys** and **values**." with "One case-preserving word map with explicit exceptions (`melee_self_buff`, `aura_enemy_debuff`, `permabuff`, `debuff`); the migration takes its new key names from the same map."
6. Name map, **Values** paragraph: append "; mutation effect types `on_hit_buff`, `aura_ally_buff`, `on_reflect_buff`, `aura_enemy_debuff` → `on_hit_condition`, `aura_ally_condition`, `on_reflect_condition`, `aura_enemy_condition`; behaviour tree category `buff_friendly` → `condition_friendly` (still matched by no spell; owner 2026-09-15 defers wiring it to the behaviour arc); MiscData key `pinnacle_bandolier_buffs` → `pinnacle_bandolier_conditions`".
7. Owner rulings list: add item 6: "**Mutation effect types all end in `_condition`**, including `aura_enemy_condition`; **`buff_friendly` becomes `condition_friendly` and stays inert** (2026-09-15, planning)."
8. Section 4 commit order: replace the list with: "1. `internal/conditionrename`. 2. Admin alias removal. 3. The atomic rewrite (data, Go strings, web, templates, folders, freeze tests, key-binding test, proof). 4. Migration 0.17.0. 5. Config deletions and weather key. 6. Go comments, docs and deletions. 7. The guard." and replace the Section 3 GMCP bullet's "with `items.js`, `mobs.js`, `quests.js`, `build.html` updated in the same commit" with "renamed by the Task 3 rewrite in the same commit as the builder JS".

- [ ] **Step 4: Commit**

```bash
cd C:/tmp/dogmud-slice3
git add docs/superpowers/specs/2026-09-15-conditions-unification-slice-3-disk-wire-design.md
git commit -m "docs(conditions): slice 3 spec corrections from planning (no shop target, word map, mutation effect types)"
```

---

### Task 1: The word map package

**Files:**
- Create: `internal/conditionrename/rename.go`
- Create: `internal/conditionrename/rename_test.go`
- Create: `internal/conditionrename/context.md`

- [ ] **Step 1: Write the failing test**

`internal/conditionrename/rename_test.go`:

```go
package conditionrename

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApply_Keys(t *testing.T) {
	cases := map[string]string{
		"buffid: 83":               "conditionid: 83",
		"buffids: [1, 2]":          "conditionids: [1, 2]",
		"buff_ids: [3]":            "condition_ids: [3]",
		"buff_id: 9":               "condition_id: 9",
		"critbuffids: [4]":         "critconditionids: [4]",
		"wornbuffids: [5]":         "wornconditionids: [5]",
		"trapbuffids: [6]":         "trapconditionids: [6]",
		"playerbuffids: [7]":       "playerconditionids: [7]",
		"mobbuffids: [8]":          "mobconditionids: [8]",
		"nativebuffids: []":        "nativeconditionids: []",
		"prizebuffids: []":         "prizeconditionids: []",
		"start_remove_buffs: [47]": "start_remove_conditions: [47]",
		"buffs:":                   "conditions:",
		"permabuff: true":          "permanent: true",
		"apply_buff:":              "apply_condition:",
		"  buff: 3":                "  condition: 3",
		`yaml:"buffid"`:            `yaml:"conditionid"`,
		`json:"wornBuffIds"`:       `json:"wornConditionIds"`,
	}
	for in, want := range cases {
		assert.Equal(t, want, Apply(in), "Apply(%q)", in)
	}
}

func TestApply_Values(t *testing.T) {
	cases := map[string]string{
		"effect_type: buff":                     "effect_type: condition",
		"behavior_archetype: melee_self_buff":   "behavior_archetype: melee_self_empower",
		"check: mob_has_buff":                   "check: mob_has_condition",
		"do: add_buff":                          "do: add_condition",
		"do: remove_buff":                       "do: remove_condition",
		"category: buff_friendly":               "category: condition_friendly",
		"- type: on_hit_buff":                   "- type: on_hit_condition",
		"- type: aura_ally_buff":                "- type: aura_ally_condition",
		"- type: on_reflect_buff":               "- type: on_reflect_condition",
		"- type: aura_enemy_debuff":             "- type: aura_enemy_condition",
		`<ansi fg="buff">`:                      `<ansi fg="condition">`,
		"buff-apply: 109":                       "condition-apply: 109",
		"{{ buffname $id }} {{ buffduration $id }}": "{{ conditionname $id }} {{ conditionduration $id }}",
		`"pinnacle_bandolier_buffs"`:            `"pinnacle_bandolier_conditions"`,
		"/buffs":                                "/conditions",
		"BuffsEnabled":                          "ConditionsEnabled",
	}
	for in, want := range cases {
		assert.Equal(t, want, Apply(in), "Apply(%q)", in)
	}
}

func TestApply_CasePreservingProse(t *testing.T) {
	assert.Equal(t, "add new condition", Apply("add new buff"))
	assert.Equal(t, "Condition does not exist", Apply("Buff does not exist"))
	assert.Equal(t, "CONDITION", Apply("BUFF"))
	assert.Equal(t, "a harmful condition", Apply("a debuff"))
	assert.Equal(t, "Harmful conditions fade", Apply("Debuffs fade"))
	assert.Equal(t, "Permanent", Apply("PermaBuff"))
}

func TestApply_ProtectedWords(t *testing.T) {
	for _, s := range []string{
		"bytes.Buffer", "a ring buffer", "WORLD_EVENT_BUFFER",
		"the wind buffets the cliff", "buffeted by gusts",
		"surfaces buffed to a dull shine", "rebuff like shopkeepers",
		"Buffalo",
	} {
		assert.Equal(t, s, Apply(s), "protected text must not change")
	}
}

func TestApply_Idempotent(t *testing.T) {
	in := "buffid: 3\neffect_type: buff\nmelee_self_buff\npermabuff: true\na debuff\n"
	once := Apply(in)
	assert.Equal(t, once, Apply(once))
}

func TestHasBuff(t *testing.T) {
	assert.True(t, HasBuff("buffid: 3"))
	assert.True(t, HasBuff("a Debuff"))
	assert.False(t, HasBuff("bytes.Buffer and a buffet and Buffalo"))
	assert.False(t, HasBuff(Apply("buffid: 3 permabuff melee_self_buff")))
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `cd C:/tmp/dogmud-slice3 && go test ./internal/conditionrename/`
Expected: FAIL, `undefined: Apply` (build failure).

- [ ] **Step 3: Write the implementation**

`internal/conditionrename/rename.go`:

```go
// Package conditionrename holds the one spelling map for conditions
// unification slice 3: every on-disk, wire and content spelling of "buff"
// becomes "condition". The Task 3 rewrite, the 0.17.0 save migration and the
// guard all read it, so the old and new spellings are listed nowhere else.
package conditionrename

import (
	"strconv"
	"strings"
)

// protected are case-sensitive words that contain "buff" but are not the
// concept. They are masked before any rename and restored after.
var protected = []string{
	"Buffer", "buffer", "BUFFER",
	"Buffet", "buffet",
	"buffed", "rebuff",
	"Buffalo",
}

// explicit are renames the generic case-preserving rule would get wrong,
// applied in order before it. Longer spellings come first so a shorter entry
// never splits a longer one.
var explicit = []struct{ old, new string }{
	{"melee_self_buff", "melee_self_empower"},
	{"aura_enemy_debuff", "aura_enemy_condition"},
	{"permabuff", "permanent"},
	{"PermaBuff", "Permanent"},
	{"Debuffs", "Harmful conditions"},
	{"debuffs", "harmful conditions"},
	{"Debuff", "Harmful condition"},
	{"debuff", "harmful condition"},
}

// generic is the case-preserving swap for everything else.
var generic = []struct{ old, new string }{
	{"BUFF", "CONDITION"},
	{"Buff", "Condition"},
	{"buff", "condition"},
}

const maskOpen, maskClose = "\x00", "\x01"

// Apply returns s with every buff spelling renamed. It is idempotent.
func Apply(s string) string {
	masked := s
	for i, word := range protected {
		masked = strings.ReplaceAll(masked, word, maskOpen+strconv.Itoa(i)+maskClose)
	}
	for _, r := range explicit {
		masked = strings.ReplaceAll(masked, r.old, r.new)
	}
	for _, r := range generic {
		masked = strings.ReplaceAll(masked, r.old, r.new)
	}
	for i, word := range protected {
		masked = strings.ReplaceAll(masked, maskOpen+strconv.Itoa(i)+maskClose, word)
	}
	return masked
}

// HasBuff reports whether s still contains a buff spelling outside the
// protected words.
func HasBuff(s string) bool {
	masked := s
	for _, word := range protected {
		masked = strings.ReplaceAll(masked, word, "")
	}
	return strings.Contains(strings.ToLower(masked), "buff")
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/conditionrename/ -count=1`
Expected: `ok`.

- [ ] **Step 5: Null-probe**

Delete the `{"melee_self_buff", "melee_self_empower"},` line, run the tests, confirm `TestApply_Values` fails on `melee_self_buff`, restore the line, rerun green. Then delete `"buffet",` from `protected`, confirm `TestApply_ProtectedWords` fails, restore.

- [ ] **Step 6: context.md**

`internal/conditionrename/context.md`:

```markdown
# internal/conditionrename

The single spelling map for conditions unification slice 3
(`docs/superpowers/specs/2026-09-15-conditions-unification-slice-3-disk-wire-design.md`).

- `Apply(s string) string` renames every buff spelling in `s` to its
  condition spelling. Case-preserving (`buff`/`Buff`/`BUFF`), with explicit
  exceptions (`melee_self_buff` → `melee_self_empower`, `aura_enemy_debuff` →
  `aura_enemy_condition`, `permabuff` → `permanent`, `debuff` → `harmful
  condition`) and protected words that are left alone (`buffer`, `buffet`,
  `buffed`, `rebuff`, `Buffalo`). Idempotent.
- `HasBuff(s string) bool` reports a remaining buff spelling outside the
  protected words. The root guard uses it.

Readers: `internal/migration/0.17.0.go` (new save key names), the root guard
`identifier_word_guard_test.go`. Do not add a second list of old spellings
anywhere else; extend this one.
```

- [ ] **Step 7: Commit**

```bash
git add internal/conditionrename/rename.go internal/conditionrename/rename_test.go internal/conditionrename/context.md
git commit -m "feat(conditions): slice 3 word map package (conditionrename.Apply, HasBuff)"
```

Add a row for `internal/conditionrename/context.md` to `docs/README.md` only if that file indexes package `context.md` files (check with `grep -n "context.md" docs/README.md`); if it does not, skip.

---

### Task 2: Remove the admin `buff` alias

The rewrite in Task 3 would turn the alias into a `condition` alias, one letter from the player command `conditions`. It goes first, by hand.

**Files:**
- Modify: `_datafiles/world/dogmud/keywords.yaml:299,322`
- Modify: `_datafiles/world/default/keywords.yaml:162,184`
- Modify: `_datafiles/world/{dogmud,default}/templates/help/setcondition.template`
- Modify: `_datafiles/world/{dogmud,default}/templates/admincommands/help/command.setcondition.template`
- Modify: `internal/keywords/keywords_setcondition_alias_test.go`
- Modify: `internal/usercommands/usercommands_test.go:1972-2025, 3186-3220, 4414-4437`

- [ ] **Step 1: Flip the real-keywords test first (it must fail)**

Replace the whole of `internal/keywords/keywords_setcondition_alias_test.go` with:

```go
package keywords

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/require"
)

// TestBuffIsNoLongerAnAlias loads each shipped world's real keywords.yaml and
// proves conditions unification slice 3 removed the `buff` alias that slice 2
// kept for `setcondition` (owner ruling 2026-09-14: alias until slice 3).
// TryCommandAlias and TryHelpAlias return their input unchanged when no alias
// matches.
func TestBuffIsNoLongerAnAlias(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)

	origKeywords := loadedKeywords
	defer func() { loadedKeywords = origKeywords }()

	for _, world := range []string{"dogmud", "default"} {
		t.Run(world, func(t *testing.T) {
			cfg := configs.GetConfig()
			cfg.FilePaths.DataFiles = configs.ConfigString(filepath.Join(filepath.Dir(here), "..", "..", "_datafiles", "world", world))
			configs.SetConfigForTest(t, cfg)

			LoadAliases()

			require.Equal(t, "buff", TryCommandAlias("buff"),
				"the %s world's keywords.yaml must not alias `buff` to any command", world)
			require.Equal(t, "buff", TryHelpAlias("buff"),
				"the %s world's keywords.yaml must not alias `help buff`", world)
			require.Equal(t, "setcondition", TryCommandAlias("setcondition"))
		})
	}
}
```

Run: `go test ./internal/keywords/ -run TestBuffIsNoLongerAnAlias -count=1`
Expected: FAIL (`expected "buff", actual "setcondition"`).

- [ ] **Step 2: Remove the alias lines**

In `_datafiles/world/dogmud/keywords.yaml` delete line 299 (`  setcondition:     [buff]`) and line 322 (`  setcondition:       ['buff'] # Admin only: alias until conditions slice 3`). In `_datafiles/world/default/keywords.yaml` delete line 162 (`  setcondition:     [buff]`) and line 184 (the `['buff']` line). Delete by content, not by number, if the lines have moved.

Run: `go test ./internal/keywords/ -count=1`
Expected: `ok`.

- [ ] **Step 3: Remove the alias prose from the help templates**

In both `templates/help/setcondition.template` files: delete the line
`<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">"buff" still works as an alias for this command.</ansi>`
and the blank line directly above it if that leaves two blank lines in a row. Change line 3's `applies a condition (a buff, a debuff, or` to `applies a condition (helpful, harmful, or`.

In both `templates/admincommands/help/command.setcondition.template` files: delete the two lines
`<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">"buff" still works as an alias for this command;</ansi>` and
`<ansi fg="magenta">it will be removed in a later update.</ansi>`
and a now-doubled blank line.

Run: `git grep -n -i "buff" -- '_datafiles/world/*/templates/help/setcondition.template' '_datafiles/world/*/templates/admincommands/help/command.setcondition.template' '_datafiles/world/*/keywords.yaml'`
Expected: no output (exit 1).

- [ ] **Step 4: Retarget the synthetic alias tests**

In `internal/usercommands/usercommands_test.go`:

a) `TestAdminSetCondition_AliasDispatchAndAdminGate` (around line 1972): replace the doc comment's first sentence block (from `// TestAdminSetCondition_AliasDispatchAndAdminGate exercises real alias` to the line before `func`) with:

```go
// TestAdminSetCondition_AliasDispatchAndAdminGate exercises real alias
// resolution through TryCommand (not a direct SetCondition call) with a
// seeded alias `sc`. The alias is resolved by keywords.TryCommandAlias before
// userCommands is ever indexed (internal/usercommands/usercommands.go
// TryCommand), so the AdminOnly gate on the `setcondition` entry applies
// identically no matter which spelling is typed; a non-admin typing either
// must be refused exactly the same way as any other admin-only command (see
// admin_command_as_non_admin in TestTryCommand).
```

and in the body replace `map[string][]string{"setcondition": {"buff"}}` with `map[string][]string{"setcondition": {"sc"}}`, the two subtest names `"admin reaches the handler via the buff alias"` / `"non-admin is refused the buff alias"` with `"admin reaches the handler via an alias"` / `"non-admin is refused the alias"`, and both `TryCommand("buff", "list", 1, events.CmdSkipScripts)` with `TryCommand("sc", "list", 1, events.CmdSkipScripts)`.

b) `TestGetCmdSuggestions_FiltersAdminOnlyAliases` (around line 3186): in the doc comment replace
`// `command`) and, since slice 2 of the conditions unification, `buff`` / `// (alias for the admin-only `setcondition`) to every player. Uses the real`
with
`// `command`) and `sc` (a seeded alias for the admin-only `setcondition`)` / `// to every player. Uses the real`;
in the body replace `"setcondition": {"buff"},` with `"setcondition": {"sc"},`, `assert.NotContains(t, results, "buff", "buff aliases the admin-only setcondition")` with `assert.NotContains(t, results, "sc", "sc aliases the admin-only setcondition")`, and `assert.Contains(t, results, "buff")` with `assert.Contains(t, results, "sc")`.

c) Delete `TestGetHelpContents_AliasMatchesSetCondition` entirely (its doc comment and function, around lines 4414-4437). It tested the `help buff` alias, which no longer exists; the real-keywords test above proves the removal.

- [ ] **Step 5: Run the affected packages**

Run: `go test ./internal/keywords/ ./internal/usercommands/ ./internal/templates/ -count=1`
Expected: all `ok`. If `internal/templates` has a help-template golden that includes the deleted line, update that expectation to the new text and rerun.

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/keywords.yaml _datafiles/world/default/keywords.yaml \
  _datafiles/world/dogmud/templates/help/setcondition.template _datafiles/world/default/templates/help/setcondition.template \
  _datafiles/world/dogmud/templates/admincommands/help/command.setcondition.template _datafiles/world/default/templates/admincommands/help/command.setcondition.template \
  internal/keywords/keywords_setcondition_alias_test.go internal/usercommands/usercommands_test.go
git commit -m "feat(conditions): remove the admin buff alias (slice 3)"
```

Stage any template golden you updated in Step 5 by its path too.

---

### Task 3: The atomic rewrite

Everything a loader, a client or a template reads changes in this one commit, or the game loads empty data with no error.

**Files:**
- Create (throwaway): `tools/slice3rewrite/main.go`, `tools/slice3proof/main.go`
- Move: `_datafiles/world/dogmud/buffs/` → `conditions/`; `_datafiles/world/default/buffs/` → `conditions/`; `_datafiles/world/dogmud/behaviors/archetypes/melee_self_buff.yaml` → `melee_self_empower.yaml`; `internal/behaviortree/melee_self_buff_archetype_integration_test.go` → `melee_self_empower_archetype_integration_test.go`; `internal/behaviortree/melee_self_buff_packmate_hurt_test.go` → `melee_self_empower_packmate_hurt_test.go`; `internal/narration/testdata/stores/buffs.golden` → `conditions.golden`
- Rewrite (by tool): every tracked `.yaml .yml .template .html .js .css` under `_datafiles/` and every tracked `.go` STRING token, with the exclusions in Step 3
- Rewrite (by hand): `wire_freeze_test.go`, `identifier_word_guard_test.go` allowlist, `_datafiles/world/{dogmud,default}/ansi-aliases.yaml` (`condition-text` line), `tools/id_inventory.py`
- Create: `condition_keys_bind_test.go`

- [ ] **Step 1: Write the new wire freeze test (it must fail)**

Replace the whole of `wire_freeze_test.go` with:

```go
package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Conditions unification slice 3 renamed every on-disk and wire spelling of
// buff to condition (docs/superpowers/specs/2026-09-15-conditions-unification-slice-3-disk-wire-design.md).
// These tests pin the new literal keys, values and strings, and that the old
// ones no longer bind: loaders ignore unknown keys, so a stale key would load
// as a silent zero value.

func TestWireFreeze_CharacterSaveKeys(t *testing.T) {
	defer conditions.SeedConditionRecordsForTest()()

	c := characters.Character{}
	c.Conditions.Validate(true)
	require.True(t, c.Conditions.AddConditionMagnitude(conditions.ConditionIdBleeding, 3, -2))
	require.True(t, c.Conditions.AddConditionMagnitude(conditions.ConditionIdBleeding, 5, -3))
	c.Conditions.List = append(c.Conditions.List, &conditions.Condition{ConditionId: conditions.ConditionIdWarcry, Permanent: true, TriggersLeft: 1})

	out, err := yaml.Marshal(&c)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))
	assert.NotContains(t, doc, "buffs")
	held, ok := doc["conditions"].(map[any]any)
	require.True(t, ok, "the character save must use the `conditions:` key; got:\n%s", out)
	list, ok := held["list"].([]any)
	require.True(t, ok, "the held records must stay under `list:`")
	require.Len(t, list, 2)

	bleed := list[0].(map[any]any)
	assert.EqualValues(t, conditions.ConditionIdBleeding, bleed["conditionid"], "a record's id must be under `conditionid:`")
	assert.NotContains(t, bleed, "buffid")
	assert.Contains(t, bleed, "triggersleft")
	assert.Contains(t, bleed, "stacks")
	stack := bleed["stacks"].([]any)[0].(map[any]any)
	assert.Contains(t, stack, "roundsleft")
	assert.Contains(t, stack, "amount")

	warcry := list[1].(map[any]any)
	assert.Equal(t, true, warcry["permanent"], "a permanent record must be under `permanent:`")
	assert.NotContains(t, warcry, "permabuff")

	var back characters.Character
	require.NoError(t, yaml.Unmarshal(out, &back))
	require.Len(t, back.Conditions.List, 2)
	assert.Equal(t, conditions.ConditionIdBleeding, back.Conditions.List[0].ConditionId)
	assert.Len(t, back.Conditions.List[0].Stacks, 2)
	assert.True(t, back.Conditions.List[1].Permanent)

	var stale characters.Character
	require.NoError(t, yaml.Unmarshal([]byte("buffs:\n  list:\n  - buffid: 3\n    permabuff: true\n"), &stale))
	assert.Empty(t, stale.Conditions.List, "the old `buffs:` key must no longer bind; saves are migrated by 0.17.0")
}

func TestWireFreeze_ConditionFileKeys(t *testing.T) {
	var s conditions.ConditionSpec
	doc := "conditionid: 950\nname: Probe\ntriggerrate: 1 round\ntriggercount: 2\n" +
		"start_remove_conditions: [3]\neffects:\n  damage_mult: magnitude\nflags:\n  - stacking\n" +
		"tick_pool: health\ntick_from_magnitude: true\n"
	require.NoError(t, yaml.Unmarshal([]byte(doc), &s))
	assert.Equal(t, 950, s.ConditionId, "`conditionid:` must name the record")
	assert.Equal(t, "Probe", s.Name)
	assert.Equal(t, 2, s.TriggerCount)
	assert.Equal(t, "health", s.TickPool)
	assert.True(t, s.TickFromMagnitude)
	assert.Equal(t, []int{3}, s.StartRemoveConditions, "`start_remove_conditions:` must parse")
	assert.True(t, s.Effects[conditions.EffectDamageMult].UsesMagnitude)
	assert.Equal(t, []conditions.Flag{conditions.Stacking}, s.Flags)

	var old conditions.ConditionSpec
	require.NoError(t, yaml.Unmarshal([]byte("buffid: 950\nstart_remove_buffs: [3]\n"), &old))
	assert.Zero(t, old.ConditionId, "`buffid:` must no longer bind")
	assert.Empty(t, old.StartRemoveConditions, "`start_remove_buffs:` must no longer bind")
}

func TestWireFreeze_ShippedConditionFilesLoadInBothWorlds(t *testing.T) {
	// Root package main has no TestMain; conditions.LoadDataFiles logs.
	mudlog.SetupLogger(nil, `LOW`, ``, false)

	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	for _, world := range []string{"dogmud", "default"} {
		t.Run(world, func(t *testing.T) {
			cfg := configs.GetConfig()
			cfg.FilePaths.DataFiles = configs.ConfigString(filepath.Join(filepath.Dir(here), "_datafiles", "world", world))
			cfg.Network.LogoutRounds = 3
			configs.SetConfigForTest(t, cfg)
			restore := conditions.SeedConditionsForTest(nil)
			defer restore()
			conditions.LoadDataFiles()
			assert.NotEmpty(t, conditions.GetAllConditionIds(), "the %s world's conditions/ folder must load", world)
		})
	}
}

func TestWireFreeze_SpeciesAndSpellValues(t *testing.T) {
	var sp species.Species
	require.NoError(t, yaml.Unmarshal([]byte("name: Probe\nconditionids: [7]\n"), &sp))
	assert.Equal(t, []int{7}, sp.ConditionIds, "species `conditionids:` must parse")

	var spell spells.SpellData
	require.NoError(t, yaml.Unmarshal([]byte("spellid: probe\nname: Probe\neffect_type: condition\ncondition_ids: [3]\n"), &spell))
	assert.Equal(t, "condition", spell.EffectType)
	assert.Equal(t, []int{3}, spell.ConditionIds, "`condition_ids:` must parse")
}

func TestWireFreeze_MessagingCategoryStrings(t *testing.T) {
	assert.Equal(t, "condition-apply", messaging.CategoryConditionApply.String())
	assert.Equal(t, "condition-expire", messaging.CategoryConditionExpire.String())
}
```

Run: `go test . -run TestWireFreeze -count=1`
Expected: FAIL (the character save still writes `buffs:`, etc.).

- [ ] **Step 2: Move the folders and files**

```bash
cd C:/tmp/dogmud-slice3
git mv _datafiles/world/dogmud/buffs _datafiles/world/dogmud/conditions
git mv _datafiles/world/default/buffs _datafiles/world/default/conditions
git mv _datafiles/world/dogmud/behaviors/archetypes/melee_self_buff.yaml _datafiles/world/dogmud/behaviors/archetypes/melee_self_empower.yaml
git mv internal/behaviortree/melee_self_buff_archetype_integration_test.go internal/behaviortree/melee_self_empower_archetype_integration_test.go
git mv internal/behaviortree/melee_self_buff_packmate_hurt_test.go internal/behaviortree/melee_self_empower_packmate_hurt_test.go
git mv internal/narration/testdata/stores/buffs.golden internal/narration/testdata/stores/conditions.golden
git ls-files | grep -i buff | grep -v "^docs/superpowers/\|^tools/playtest/"
```

Expected: the last command lists only `_datafiles/guides/building/scripting/SCRIPTING_BUFFS.md` and `docs/schemas/buff.md` (both handled in Task 6).

- [ ] **Step 3: Write the rewrite tool**

`tools/slice3rewrite/main.go` (never committed):

```go
// Throwaway: conditions slice 3 rewrite. Applies conditionrename.Apply to
// tracked data/web files and to Go STRING tokens. Delete before committing.
package main

import (
	"bytes"
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/conditionrename"
)

var dataExts = map[string]bool{".yaml": true, ".yml": true, ".template": true, ".html": true, ".js": true, ".css": true}

// Files left alone: their buff spellings are handled by hand in a later task,
// or are deliberate player-facing keywords.
var skip = map[string]bool{
	"_datafiles/world/dogmud/dialogue/newcomer_antechamber/9491.yaml": true, // player keyword "buff" stays
	"identifier_word_guard_test.go":                                  true, // allowlist emptied in Task 3 Step 5c; extended in Task 7
	"wire_freeze_test.go":                                            true, // rewritten in Step 1; asserts old keys no longer bind
	"internal/keywords/keywords_setcondition_alias_test.go":          true, // Task 2; asserts "buff" is not an alias
	"internal/conditionrename/rename.go":                             true, // the map itself
	"internal/conditionrename/rename_test.go":                        true,
	"internal/configs/config.gameplay.go":                            true, // Task 5 deletes the knobs
	"internal/configs/config.balance.go":                             true, // Task 5
	"modules/weather/weather_config.go":                              true, // Task 5, with config.yaml
	"modules/weather/weather_config_test.go":                         true, // Task 5
}

func tracked(pathspec ...string) []string {
	out, err := exec.Command("git", append([]string{"ls-files", "-z", "--"}, pathspec...)...).Output()
	if err != nil {
		panic(err)
	}
	var files []string
	for _, f := range strings.Split(string(out), "\x00") {
		if f != "" {
			files = append(files, filepath.ToSlash(f))
		}
	}
	return files
}

func rewriteData(path string) (bool, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	dst := conditionrename.Apply(string(src))
	if dst == string(src) {
		return false, nil
	}
	return true, os.WriteFile(path, []byte(dst), 0644)
}

func rewriteGo(path string) (bool, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	fset := token.NewFileSet()
	file := fset.AddFile(path, fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(file, src, nil, 0)
	var out bytes.Buffer
	last := 0
	changed := false
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok != token.STRING {
			continue
		}
		off := file.Offset(pos)
		// go/scanner strips carriage returns from a raw string's `lit`, and
		// this checkout has CRLF line endings, so len(lit) is not the
		// literal's length on disk. Take the raw literal's extent from the
		// source bytes instead; an interpreted string cannot contain a CR.
		end := off + len(lit)
		if src[off] == '`' {
			end = off + 1 + bytes.IndexByte(src[off+1:], '`') + 1
		}
		orig := string(src[off:end])
		renamed := conditionrename.Apply(orig)
		if renamed == orig {
			continue
		}
		out.Write(src[last:off])
		out.WriteString(renamed)
		last = end
		changed = true
	}
	if !changed {
		return false, nil
	}
	out.Write(src[last:])
	return true, os.WriteFile(path, out.Bytes(), 0644)
}

func main() {
	n := 0
	for _, f := range tracked("_datafiles") {
		if skip[f] || !dataExts[filepath.Ext(f)] {
			continue
		}
		ok, err := rewriteData(f)
		if err != nil {
			panic(err)
		}
		if ok {
			n++
			fmt.Println("data", f)
		}
	}
	for _, f := range tracked("*.go") {
		if skip[f] || strings.HasPrefix(f, "vendor/") || strings.HasPrefix(f, "tools/slice3") {
			continue
		}
		ok, err := rewriteGo(f)
		if err != nil {
			panic(err)
		}
		if ok {
			n++
			fmt.Println("go  ", f)
		}
	}
	for _, f := range tracked("internal/narration/testdata/stores/conditions.golden") {
		ok, err := rewriteData(f)
		if err != nil {
			panic(err)
		}
		if ok {
			n++
			fmt.Println("gold", f)
		}
	}
	fmt.Println("files rewritten:", n)
}
```

Note: the golden file is under `internal/`, not `_datafiles/`, so it is handled explicitly; `.golden` rows like `buff|83|...` come from Go strings in `internal/narration/snapshot_test.go`, which the Go pass renames the same way.

- [ ] **Step 4: Run it**

```bash
cd C:/tmp/dogmud-slice3
go run ./tools/slice3rewrite > C:/tmp/slice3-rewrite.log
tail -1 C:/tmp/slice3-rewrite.log
gofmt -l $(git diff --name-only -- '*.go')
```

Expected: `files rewritten:` followed by a count in the hundreds; `gofmt -l` prints nothing (string contents changed, not layout).

- [ ] **Step 5: Hand fixes the tool cannot make**

a) `_datafiles/world/dogmud/ansi-aliases.yaml` and `_datafiles/world/default/ansi-aliases.yaml`: delete the line `  condition-text: 14` (was the unused `buff-text` alias). Confirm first that nothing uses it: `git grep -n 'condition-text' -- _datafiles internal modules` must show only those two lines.

b) `tools/id_inventory.py`: in the docstring change `spells / buffs / quests` to `spells / conditions / quests`, and change the TYPES entry `"buffs":     ("buffs",     False),` to `"conditions": ("conditions", False),`.

c) `identifier_word_guard_test.go`: the two `templateBuffFieldAllowlist` entries no longer match anything (the rewrite renamed `.buffids` and `.buffIds`). Replace the map literal with an empty one, keeping its doc comment:

```go
var templateBuffFieldAllowlist = map[string]string{}
```

d) `internal/behaviortree/test_export.go` comment and `internal/mobs/mobs.go:159` comment still say `melee_self_buff`: change both to `melee_self_empower` (Task 6 sweeps other comments; these two name a file path that now exists under the new name).

- [ ] **Step 6: Build, vet and run the whole suite**

```bash
go build ./... && go vet ./...
DOGMUD_BOOT_SMOKE=1 go test ./... -count=1 2>&1 | grep -E "^(FAIL|--- FAIL|ok|panic)" | grep -v "^ok"
```

Expected failures to fix in this task:
- Only these two may stay red: `internal/rooms` `TestDeleteZone_RemovesEveryTree` and `TestRenameZone_MovesRewritesAndRekeys` (pre-existing on master, Windows-local, filed 2026-09-15). If they left `_datafiles/world/dogmud/rooms/ziggurat_test_zone/` or `rename_probe_zone/` behind, `rm -rf` those two untracked folders.
- Any other failure is a real slice 3 regression. Common causes and fixes: a test comparing against authored text read from a file (fix the test's expected literal only if the rewrite changed the authored file identically); a golden that embeds a now-renamed string (regenerate ONLY if the diff is exactly buff→condition spellings, verified with `git diff --word-diff`); a hardcoded path the tool renamed in a string but a file that moved elsewhere. Record each fix in the commit message.

- [ ] **Step 7: Write the key-binding test**

`condition_keys_bind_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/pets"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Every loader in this codebase ignores unknown YAML keys, so a Go tag and a
// data file that disagree load as a silent zero value. For every shipped
// file that SAYS a renamed key, this proves the key reaches a non-empty Go
// field through the real struct the loader decodes into.
type keyBinding struct {
	folder string
	key    string
	bound  func(t *testing.T, path string, data []byte) bool
}

func keyLine(key string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^\s*-?\s*` + regexp.QuoteMeta(key) + `\s*:`)
}

func decode[T any](t *testing.T, path string, data []byte) T {
	t.Helper()
	var v T
	require.NoError(t, yaml.Unmarshal(data, &v), "decode %s", path)
	return v
}

var conditionKeyBindings = []keyBinding{
	{"conditions", "conditionid", func(t *testing.T, p string, d []byte) bool {
		id, err := strconv.Atoi(strings.SplitN(filepath.Base(p), "-", 2)[0])
		require.NoError(t, err, "condition file %s must start with its id", p)
		return decode[conditions.ConditionSpec](t, p, d).ConditionId == id
	}},
	{"conditions", "start_remove_conditions", func(t *testing.T, p string, d []byte) bool {
		return len(decode[conditions.ConditionSpec](t, p, d).StartRemoveConditions) > 0
	}},
	{"items", "conditionids", func(t *testing.T, p string, d []byte) bool {
		return len(decode[items.ItemSpec](t, p, d).ConditionIds) > 0
	}},
	{"items", "wornconditionids", func(t *testing.T, p string, d []byte) bool {
		return len(decode[items.ItemSpec](t, p, d).WornConditionIds) > 0
	}},
	{"items", "critconditionids", func(t *testing.T, p string, d []byte) bool {
		return len(decode[items.ItemSpec](t, p, d).Damage.CritConditionIds) > 0
	}},
	{"species", "conditionids", func(t *testing.T, p string, d []byte) bool {
		return len(decode[species.Species](t, p, d).ConditionIds) > 0
	}},
	{"species", "critconditionids", func(t *testing.T, p string, d []byte) bool {
		return len(decode[species.Species](t, p, d).Damage.CritConditionIds) > 0
	}},
	{"mobs", "conditionids", func(t *testing.T, p string, d []byte) bool {
		return len(decode[mobs.Mob](t, p, d).ConditionIds) > 0
	}},
	{"mobs", "conditionid", func(t *testing.T, p string, d []byte) bool {
		for _, s := range decode[mobs.Mob](t, p, d).Character.Shop {
			if s.ConditionId > 0 {
				return true
			}
		}
		return false
	}},
	{"spells", "condition_ids", func(t *testing.T, p string, d []byte) bool {
		return len(decode[spells.SpellData](t, p, d).ConditionIds) > 0
	}},
	{"mutators", "playerconditionids", func(t *testing.T, p string, d []byte) bool {
		return len(decode[mutators.MutatorSpec](t, p, d).PlayerConditionIds) > 0
	}},
	{"mutators", "mobconditionids", func(t *testing.T, p string, d []byte) bool {
		return len(decode[mutators.MutatorSpec](t, p, d).MobConditionIds) > 0
	}},
	{"rooms", "trapconditionids", func(t *testing.T, p string, d []byte) bool {
		r := decode[rooms.Room](t, p, d)
		for _, e := range r.Exits {
			if len(e.Lock.TrapConditionIds) > 0 {
				return true
			}
		}
		for _, c := range r.Containers {
			if len(c.Lock.TrapConditionIds) > 0 {
				return true
			}
		}
		return false
	}},
	{"pets", "conditionids", func(t *testing.T, p string, d []byte) bool {
		return len(decode[pets.Pet](t, p, d).ConditionIds) > 0
	}},
	{"quests", "conditionid", func(t *testing.T, p string, d []byte) bool {
		return decode[quests.Quest](t, p, d).Rewards.ConditionId > 0
	}},
}

func TestShippedConditionKeysBind(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	checked := 0
	for _, world := range []string{"dogmud", "default"} {
		for _, b := range conditionKeyBindings {
			root := filepath.Join(filepath.Dir(here), "_datafiles", "world", world, b.folder)
			if _, err := os.Stat(root); os.IsNotExist(err) {
				continue
			}
			re := keyLine(b.key)
			require.NoError(t, filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
				if err != nil {
					if os.IsNotExist(err) {
						return nil
					}
					return err
				}
				if d.IsDir() || !strings.HasSuffix(p, ".yaml") {
					return nil
				}
				data, rerr := os.ReadFile(p)
				if rerr != nil {
					return rerr
				}
				if !re.Match(data) {
					return nil
				}
				checked++
				if !b.bound(t, p, data) {
					t.Errorf("%s says `%s:` but it does not reach its Go field (tag and file disagree)", p, b.key)
				}
				return nil
			}))
		}
	}
	require.Greater(t, checked, 300, "checked only %d files; the key patterns or folders are wrong", checked)
}

// A missing colour alias drops colour silently: the renderer prints the text
// uncoloured. The category strings a condition start or end line is sent with
// must be real alias keys, and the condition colour must exist in both worlds.
func TestConditionColourAliasesExist(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	colorsOf := func(world string) map[string]any {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(here), "_datafiles", "world", world, "ansi-aliases.yaml"))
		require.NoError(t, err)
		var doc struct {
			Colors map[string]any `yaml:"colors"`
		}
		require.NoError(t, yaml.Unmarshal(data, &doc))
		require.NotEmpty(t, doc.Colors, "%s ansi-aliases.yaml has no colors: map", world)
		return doc.Colors
	}

	dogmud := colorsOf("dogmud")
	for _, key := range []string{"condition", messaging.CategoryConditionApply.String(), messaging.CategoryConditionExpire.String()} {
		require.Contains(t, dogmud, key, "dogmud ansi-aliases.yaml must define %q", key)
	}
	require.Contains(t, colorsOf("default"), "condition", "default ansi-aliases.yaml must define condition")
}
```

Add `"github.com/GoMudEngine/GoMud/internal/messaging"` to that file's imports.

Run: `go test . -run 'TestShippedConditionKeysBind|TestConditionColourAliasesExist' -count=1 -v 2>&1 | tail -8`
Expected: PASS. If a binding fails for a file where the key sits under a different struct path than assumed (for example a species `conditionids` nested under a field), read that file and correct the `bound` function for that case; do not loosen the count.

- [ ] **Step 8: Null-probe the key-binding test and the freeze test**

1. In `internal/items/itemspec.go` change `yaml:"wornconditionids,omitempty"` back to `yaml:"wornbuffids,omitempty"`. Run `go test . -run 'TestShippedConditionKeysBind' -count=1`. Expected: FAIL naming files that say `wornconditionids:`. Restore.
2. In `internal/characters/character.go:145` change `yaml:"conditions,omitempty"` to `yaml:"buffs,omitempty"`. Run `go test . -run TestWireFreeze_CharacterSaveKeys -count=1`. Expected: FAIL. Restore.
3. In `_datafiles/world/dogmud/ansi-aliases.yaml` rename the `condition-apply:` key to `condition-applyx:`. Run `go test . -run TestConditionColourAliasesExist -count=1`. Expected: FAIL naming `condition-apply`. Restore.

- [ ] **Step 9: Write and run the equivalence proof**

`tools/slice3proof/main.go` (never committed):

```go
// Throwaway: conditions slice 3 equivalence proof. For every YAML file under
// _datafiles/world at master, decode it, apply conditionrename.Apply to every
// map key and string scalar, and require deep equality with the branch's file
// at the renamed path. Delete before committing.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/conditionrename"
	"gopkg.in/yaml.v3"
)

func renameTree(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[conditionrename.Apply(k)] = renameTree(val)
		}
		return out
	case map[any]any:
		out := make(map[any]any, len(x))
		for k, val := range x {
			if ks, ok := k.(string); ok {
				k = conditionrename.Apply(ks)
			}
			out[k] = renameTree(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = renameTree(val)
		}
		return out
	case string:
		return conditionrename.Apply(x)
	}
	return v
}

func main() {
	list, err := exec.Command("git", "ls-tree", "-r", "--name-only", "master", "--", "_datafiles/world").Output()
	if err != nil {
		panic(err)
	}
	checked, bad := 0, 0
	for _, oldPath := range strings.Split(strings.TrimSpace(string(list)), "\n") {
		if !strings.HasSuffix(oldPath, ".yaml") {
			continue
		}
		oldRaw, err := exec.Command("git", "show", "master:"+oldPath).Output()
		if err != nil {
			panic(err)
		}
		newPath := conditionrename.Apply(oldPath)
		newRaw, err := os.ReadFile(newPath)
		if err != nil {
			fmt.Printf("MISSING %s (from %s)\n", newPath, oldPath)
			bad++
			continue
		}
		var oldDoc, newDoc any
		if err := yaml.Unmarshal(oldRaw, &oldDoc); err != nil {
			fmt.Printf("UNPARSEABLE-OLD %s: %v\n", oldPath, err)
			bad++
			continue
		}
		if err := yaml.Unmarshal(newRaw, &newDoc); err != nil {
			fmt.Printf("UNPARSEABLE-NEW %s: %v\n", newPath, err)
			bad++
			continue
		}
		checked++
		if !reflect.DeepEqual(renameTree(oldDoc), newDoc) {
			fmt.Printf("DIFFERS %s\n", newPath)
			bad++
		}
	}
	fmt.Printf("checked %d files, %d differ\n", checked, bad)
}
```

Run: `go run ./tools/slice3proof | tee C:/tmp/slice3-proof.log | tail -20`

Expected: `checked N files` with N above 3000, and `DIFFERS` only for exactly these files, each explained by a hand edit in Task 2 or Task 3:
- `_datafiles/world/dogmud/keywords.yaml`, `_datafiles/world/default/keywords.yaml` (alias removed, Task 2)
- `_datafiles/world/dogmud/ansi-aliases.yaml`, `_datafiles/world/default/ansi-aliases.yaml` (`buff-text` deleted, Step 5a)
- `_datafiles/world/dogmud/dialogue/newcomer_antechamber/9491.yaml` (skipped on purpose; the proof's map renames its `"buff"` keyword, the file keeps it)

Any other `DIFFERS`, any `MISSING` or `UNPARSEABLE` line is a defect: diff that file against master and fix it before continuing. Paste the last line of the log into the commit message.

- [ ] **Step 10: Delete the throwaway tools and verify nothing buff-spelled is left in Go strings or data**

```bash
rm -rf tools/slice3rewrite tools/slice3proof
git status --short tools/
git grep -n -i "buff" -- '_datafiles/*.yaml' '_datafiles/*.template' '_datafiles/*.html' '_datafiles/*.js' '_datafiles/*.css' | grep -v -i "buffer\|buffet\|buffed\|rebuff"
```

Expected: `git status --short tools/` shows only the `tools/id_inventory.py` and `tools/casing_sweep/main.go` modifications; the grep shows exactly one line, `dialogue/newcomer_antechamber/9491.yaml:37` (the kept keyword).

- [ ] **Step 11: Commit**

```bash
git diff --name-only -z | xargs -0 git add --
git add condition_keys_bind_test.go
git status --short | grep -v "^[MRD] "
git commit -m "feat(conditions): slice 3 atomic rename of every disk, wire and content buff spelling"
```

The first line stages exactly the modified tracked paths by name (the `git mv` renames are already staged); never `git add -A` or `git add .`. The `git status` line must print nothing: no untracked file left behind, no unstaged modification. Commit message body: the `files rewritten:` count, the proof's last line, and each Step 6 test fix.

---

### Task 4: Migration 0.17.0

**Files:**
- Create: `internal/migration/0.17.0.go`
- Create: `internal/migration/0.17.0_test.go`
- Modify: `internal/migration/migration.go` (after the 0.16.0 block, before `return nil`)
- Modify: `main.go:97`
- Modify: `internal/migration/context.md`

- [ ] **Step 1: Write the failing tests**

`internal/migration/0.17.0_test.go`:

```go
package migration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const oldUserSave = `userid: 7
username: probe
character:
  name: Probe
  buffs:
    list:
    - buffid: 122
      triggersleft: 4
      stacks:
      - roundsleft: 3
        amount: -2
    - buffid: 79
      permabuff: true
  pet:
    type: dog
    buffids: [12, 13]
  shop:
  - itemid: 5
  - buffid: 4
    price: 10
  miscdata:
    pinnacle_bandolier_buffs: [51]
    unrelated: keep
`

const oldAltsSave = `- name: AltOne
  buffs:
    list:
    - buffid: 3
      permabuff: true
- name: AltTwo
  pet:
    type: cat
    buffids: [9]
`

const oldRoomInstance = `roomid: 1001
containers:
  chest:
    lock:
      difficulty: 3
      trapbuffids: [45]
    gold: 7
  crate:
    gold: 1
`

func writeFixture(t *testing.T, dir, rel, body string) string {
	t.Helper()
	p := filepath.Join(dir, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0755))
	require.NoError(t, os.WriteFile(p, []byte(body), 0644))
	return p
}

func TestConditionKeys_UserSave(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "users/7.yaml", oldUserSave)
	require.NoError(t, migrateConditionKeysIn(dir, false))

	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	s := string(raw)
	for _, gone := range []string{"buffs:", "buffid:", "permabuff:", "buffids:", "pinnacle_bandolier_buffs"} {
		assert.NotContains(t, s, gone)
	}

	var u users.UserRecord
	require.NoError(t, yaml.Unmarshal(raw, &u))
	require.Len(t, u.Character.Conditions.List, 2)
	assert.Equal(t, 122, u.Character.Conditions.List[0].ConditionId)
	assert.Len(t, u.Character.Conditions.List[0].Stacks, 1)
	assert.Equal(t, 79, u.Character.Conditions.List[1].ConditionId)
	assert.True(t, u.Character.Conditions.List[1].Permanent)
	assert.Equal(t, []int{12, 13}, u.Character.Pet.ConditionIds)
	require.Len(t, u.Character.Shop, 2)
	assert.Equal(t, 4, u.Character.Shop[1].ConditionId)
	assert.NotNil(t, u.Character.MiscData["pinnacle_bandolier_conditions"])
	assert.Equal(t, "keep", u.Character.MiscData["unrelated"])
}

func TestConditionKeys_AltsList(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "users/7.alts.yaml", oldAltsSave)
	require.NoError(t, migrateConditionKeysIn(dir, false))

	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	var alts []characters.Character
	require.NoError(t, yaml.Unmarshal(raw, &alts))
	require.Len(t, alts, 2)
	require.Len(t, alts[0].Conditions.List, 1)
	assert.Equal(t, 3, alts[0].Conditions.List[0].ConditionId)
	assert.True(t, alts[0].Conditions.List[0].Permanent)
	assert.Equal(t, []int{9}, alts[1].Pet.ConditionIds)
}

func TestConditionKeys_RoomInstance(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "rooms.instances/frostfang/1001.yaml", oldRoomInstance)
	require.NoError(t, migrateConditionKeysIn(dir, false))

	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	var r rooms.Room
	require.NoError(t, yaml.Unmarshal(raw, &r))
	assert.Equal(t, []int{45}, r.Containers["chest"].Lock.TrapConditionIds)
	assert.Equal(t, 1, r.Containers["crate"].Gold)
}

func TestConditionKeys_SecondRunIsANoOp(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "users/7.yaml", oldUserSave)
	require.NoError(t, migrateConditionKeysIn(dir, false))
	first, err := os.ReadFile(p)
	require.NoError(t, err)
	info1, err := os.Stat(p)
	require.NoError(t, err)

	time.Sleep(20 * time.Millisecond)
	require.NoError(t, migrateConditionKeysIn(dir, false))
	second, err := os.ReadFile(p)
	require.NoError(t, err)
	info2, err := os.Stat(p)
	require.NoError(t, err)
	assert.Equal(t, string(first), string(second))
	assert.Equal(t, info1.ModTime(), info2.ModTime(), "an already-migrated file must not be rewritten")
}

func TestConditionKeys_UntouchedFileNotRewritten(t *testing.T) {
	dir := t.TempDir()
	body := "userid: 8\nusername: clean\ncharacter:\n  name: Clean\n"
	p := writeFixture(t, dir, "users/8.yaml", body)
	require.NoError(t, migrateConditionKeysIn(dir, false))
	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, body, string(raw), "a save with nothing to rename must keep its exact bytes")
}

func TestConditionKeys_DryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	p := writeFixture(t, dir, "users/7.yaml", oldUserSave)
	require.NoError(t, migrateConditionKeysIn(dir, true))
	raw, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, oldUserSave, string(raw))
}

func TestConditionKeys_CollisionIsAnError(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "users/9.yaml", "character:\n  buffs:\n    list: []\n  conditions:\n    list: []\n")
	err := migrateConditionKeysIn(dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "9.yaml")
}

func TestConditionKeys_UnparseableIsAnError(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "users/10.yaml", "character: [unclosed\n")
	err := migrateConditionKeysIn(dir, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "10.yaml")
}

func TestConditionKeys_MissingFoldersAreNotErrors(t *testing.T) {
	require.NoError(t, migrateConditionKeysIn(t.TempDir(), false))
}
```

Run: `go test ./internal/migration/ -run TestConditionKeys -count=1`
Expected: FAIL, `undefined: migrateConditionKeysIn`.

If the import of `internal/users` or `internal/characters` from `internal/migration` tests creates a cycle, the build error will say so; in that case replace those decodes with the same structs' packages that do not cycle (check with `go list -deps ./internal/users | grep internal/migration`, which printed nothing on 2026-09-15, so no cycle is expected).

- [ ] **Step 2: Write the migration**

`internal/migration/0.17.0.go`:

```go
package migration

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/conditionrename"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// Description:
// Conditions unification slice 3 renamed every buff-spelled save key to its
// condition spelling (spec
// docs/superpowers/specs/2026-09-15-conditions-unification-slice-3-disk-wire-design.md).
// Every loader ignores unknown keys, so an unmigrated save would load with its
// conditions, pet condition ids and trapped locks silently empty.
//
// Path-anchored: only the listed key paths are renamed, and the new name comes
// from conditionrename.Apply, the one spelling map.
//
// Idempotent without a marker: a file with no old key is not rewritten, so a
// second run changes nothing. That also avoids the alts trap, where a
// character-scoped marker re-runs per alt.
//
// Unlike 0.14.0, alts files (<id>.alts.yaml, a YAML LIST of characters) are
// migrated, and an unparseable file or a collision (old and new key both
// present) is an error, so Run restores the backup rather than leaving a save
// that would lose its conditions on load.
func migrate_ConditionKeys(dryRun bool) error {
	return migrateConditionKeysIn(string(configs.GetConfig().FilePaths.DataFiles), dryRun)
}

// keyRename renames key `old` inside every mapping reached by `parent`.
// Path segments: a key name, "*" for every value of a mapping, "[]" for every
// element of a list.
type keyRename struct {
	parent []string
	old    string
}

// characterRenames are relative to one character mapping. Order matters: the
// later paths use the already-renamed `conditions`.
var characterRenames = []keyRename{
	{nil, "buffs"},
	{[]string{"conditions", "list", "[]"}, "buffid"},
	{[]string{"conditions", "list", "[]"}, "permabuff"},
	{[]string{"pet"}, "buffids"},
	{[]string{"shop", "[]"}, "buffid"},
	{[]string{"miscdata"}, "pinnacle_bandolier_buffs"},
}

var roomInstanceRenames = []keyRename{
	{[]string{"containers", "*", "lock"}, "trapbuffids"},
}

// applyRenames walks node along rename.parent and renames rename.old in each
// mapping it reaches. It reports whether anything changed.
func applyRenames(node any, renames []keyRename) (bool, error) {
	changed := false
	for _, r := range renames {
		c, err := renameAt(node, r.parent, r.old, conditionrename.Apply(r.old))
		if err != nil {
			return changed, err
		}
		changed = changed || c
	}
	return changed, nil
}

func renameAt(node any, path []string, oldKey, newKey string) (bool, error) {
	if len(path) == 0 {
		m, ok := node.(yaml.MapSlice)
		if !ok {
			return false, nil
		}
		oldIdx, hasNew := -1, false
		for i, item := range m {
			switch k, _ := item.Key.(string); k {
			case oldKey:
				oldIdx = i
			case newKey:
				hasNew = true
			}
		}
		if oldIdx < 0 {
			return false, nil
		}
		if hasNew {
			return false, fmt.Errorf("both %q and %q present", oldKey, newKey)
		}
		// MapSlice shares its backing array with the parent, so this
		// renames the key in the decoded document in place.
		m[oldIdx].Key = newKey
		return true, nil
	}
	seg, rest := path[0], path[1:]
	changed := false
	switch seg {
	case "[]":
		list, ok := node.([]any)
		if !ok {
			return false, nil
		}
		for _, el := range list {
			c, err := renameAt(el, rest, oldKey, newKey)
			if err != nil {
				return changed, err
			}
			changed = changed || c
		}
	case "*":
		m, ok := node.(yaml.MapSlice)
		if !ok {
			return false, nil
		}
		for _, item := range m {
			c, err := renameAt(item.Value, rest, oldKey, newKey)
			if err != nil {
				return changed, err
			}
			changed = changed || c
		}
	default:
		m, ok := node.(yaml.MapSlice)
		if !ok {
			return false, nil
		}
		for _, item := range m {
			if k, _ := item.Key.(string); k == seg {
				return renameAt(item.Value, rest, oldKey, newKey)
			}
		}
	}
	return changed, nil
}

// migrateConditionKeysIn is the testable core: dataDir is a DataFiles root.
func migrateConditionKeysIn(dataDir string, dryRun bool) error {
	mode := "APPLY"
	if dryRun {
		mode = "DRY-RUN"
	}
	mudlog.Info("Migration 0.17.0", "message", "Renaming buff save keys to condition keys", "mode", mode)

	counts := map[string]int{}

	usersDir := filepath.Join(dataDir, "users")
	userFiles, err := filepath.Glob(filepath.Join(usersDir, "*.yaml"))
	if err != nil {
		return err
	}
	for _, path := range userFiles {
		if strings.HasSuffix(path, ".alts.yaml") {
			if err := migrateFile(path, dryRun, migrateAltsDoc); err != nil {
				return err
			}
			counts["alts"]++
			continue
		}
		if err := migrateFile(path, dryRun, migrateUserDoc); err != nil {
			return err
		}
		counts["users"]++
	}

	roomsDir := filepath.Join(dataDir, "rooms.instances")
	err = filepath.WalkDir(roomsDir, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			if errors.Is(werr, fs.ErrNotExist) {
				return nil
			}
			return werr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		counts["rooms.instances"]++
		return migrateFile(path, dryRun, migrateRoomDoc)
	})
	if err != nil {
		return err
	}

	mudlog.Info("Migration 0.17.0", "users", counts["users"], "alts", counts["alts"], "rooms.instances", counts["rooms.instances"], "mode", mode)
	return nil
}

type docMigrator func(raw []byte) (out any, changed bool, err error)

func migrateFile(path string, dryRun bool, migrate docMigrator) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("migration 0.17.0: read %s: %w", path, err)
	}
	doc, changed, err := migrate(raw)
	if err != nil {
		return fmt.Errorf("migration 0.17.0: %s: %w", path, err)
	}
	if !changed {
		return nil
	}
	mudlog.Info("Migration 0.17.0", "file", path, "renamed", true)
	if dryRun {
		return nil
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("migration 0.17.0: marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("migration 0.17.0: write %s: %w", path, err)
	}
	return nil
}

func migrateUserDoc(raw []byte) (any, bool, error) {
	var doc yaml.MapSlice
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, false, err
	}
	for _, item := range doc {
		if k, _ := item.Key.(string); k == "character" {
			changed, err := applyRenames(item.Value, characterRenames)
			return doc, changed, err
		}
	}
	return doc, false, nil
}

func migrateAltsDoc(raw []byte) (any, bool, error) {
	var doc []yaml.MapSlice
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, false, err
	}
	changed := false
	for _, character := range doc {
		c, err := applyRenames(character, characterRenames)
		if err != nil {
			return nil, false, err
		}
		changed = changed || c
	}
	return doc, changed, nil
}

func migrateRoomDoc(raw []byte) (any, bool, error) {
	var doc yaml.MapSlice
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, false, err
	}
	changed, err := applyRenames(doc, roomInstanceRenames)
	return doc, changed, err
}
```

Note: `[]yaml.MapSlice` elements are `yaml.MapSlice`, but nested lists decode as `[]interface{}` whose elements are `yaml.MapSlice`, which `renameAt`'s `"[]"` case handles. The top-level alts elements are passed directly as `yaml.MapSlice`.

- [ ] **Step 3: Run the tests**

Run: `go test ./internal/migration/ -count=1`
Expected: `ok`.

- [ ] **Step 4: Null-probe**

1. Remove the `{[]string{"pet"}, "buffids"},` line; confirm `TestConditionKeys_UserSave` and `TestConditionKeys_AltsList` fail on `Pet.ConditionIds`; restore.
2. In `migrateConditionKeysIn`, change the alts branch to `continue` without migrating; confirm `TestConditionKeys_AltsList` fails; restore.
3. Make `migrateFile` ignore the `changed` flag (always write); confirm `TestConditionKeys_UntouchedFileNotRewritten` fails; restore.

- [ ] **Step 5: Wire it and bump the version**

In `internal/migration/migration.go`, after the 0.16.0 block and before `return nil`:

```go
	if lastConfigVersion.IsOlderThan(version.New(0, 17, 0)) {
		// Conditions unification slice 3: rename buff-spelled save keys in
		// users, alts and room instances. Datafiles are backed up by Run()
		// before this and restored on error.
		if err := migrate_ConditionKeys(false); err != nil {
			return err
		}
	}
```

In `main.go:97` change `const VERSION = "0.16.0"` to `const VERSION = "0.17.0"`.

Run: `go build ./... && go test ./internal/migration/ ./internal/playtestenv/ -count=1`
Expected: `ok` (playtestenv parses `VERSION` from `main.go`).

- [ ] **Step 6: Document**

In `internal/migration/context.md`, add an entry for 0.17.0 in the same format as the existing per-version entries (read the file first): purpose, the three targets, path-anchored renames from `conditionrename.Apply`, no marker, alts handled, errors on collision and unparseable files.

- [ ] **Step 7: Commit**

```bash
git add internal/migration/0.17.0.go internal/migration/0.17.0_test.go internal/migration/migration.go internal/migration/context.md main.go
git commit -m "feat(migration): 0.17.0 renames buff save keys in users, alts and room instances"
```

---

### Task 5: Delete the dead config knobs and rename the weather key

**Files:**
- Modify: `internal/configs/config.gameplay.go:4,53,60,120-122`
- Modify: `internal/configs/config.balance.go:214`
- Modify: `internal/configs/config.balance.combat.go:291-293`
- Modify: `_datafiles/config.yaml` (lines 275-281, 305, 1107, 2323 at `a4c078116`)
- Modify: `modules/weather/weather_config.go:17-20,111`, `modules/weather/weather_config_test.go`
- Modify: `modules/weather/context.md:53`
- Modify: `identifier_word_guard_test.go:36-43`

- [ ] **Step 1: Write the failing weather test**

`modules/weather/weather_config_test.go` `TestBuildConfigCoercionAndClamps` already feeds a false value and asserts `cfg.ConditionsEnabled` is false (the fallback is true, so an unread key would turn weather conditions on). Rename its key: line 53 `"BuffsEnabled":       false,` → `"ConditionsEnabled":  false,`. In `TestBuildConfigDefaults` change the message `"buff/persist/seed defaults wrong: %+v"` to `"conditions/persist/seed defaults wrong: %+v"`.

Run `go test ./modules/weather/ -run TestBuildConfigCoercionAndClamps -count=1`. Expected: FAIL (`bool overrides ignored`).

- [ ] **Step 2: Rename the weather key**

`modules/weather/weather_config.go:111`: `ConditionsEnabled:    boolOr(get("ConditionsEnabled"), true),`. Lines 17-20 comment: `(BuffsEnabled, not Buffs.Enabled)` → `(ConditionsEnabled, not Conditions.Enabled)`. `modules/weather/context.md:53`: `ConditionsEnabled (key `BuffsEnabled` until slice 3)` → `ConditionsEnabled`.

In `_datafiles/config.yaml` (this worktree's copy is the committed one), change `    BuffsEnabled: false` to `    ConditionsEnabled: false`.

Run: `go test ./modules/weather/ -count=1`. Expected: `ok`.

- [ ] **Step 3: Delete the three knobs**

`internal/configs/config.gameplay.go`: delete line 4 (`AllowItemBuffRemoval ConfigBool ...`), delete the `DeathsShadowBuffId` field line, delete `	// Ignore AllowItemBuffRemoval`, and delete the three-line block
```go
	if g.Death.DeathsShadowBuffId < 1 {
		g.Death.DeathsShadowBuffId = 25
	}
```
`internal/configs/config.balance.go`: delete the `BrokenLimbBuffDuration ConfigInt ...` line.
`internal/configs/config.balance.combat.go`: delete
```go
	if b.BrokenLimbBuffDuration <= 0 {
		b.BrokenLimbBuffDuration = 900
	}
```
`_datafiles/config.yaml`: delete the `# - AllowItemBuffRemoval -` comment block (6 lines) and `  AllowItemBuffRemoval: false`; delete `    DeathsShadowBuffId: 25     # Buff ID for Death's Shadow debuff`; delete `  broken_limb_buff_duration: 900 ...`.
`identifier_word_guard_test.go`: replace the allowlist block (the comment `// The three config fields that keep...` and the `identifierGuardAllowedNames` literal) with:

```go
	// Every buff-spelled Go identifier is gone (slice 3 deleted the last
	// three, dead config knobs). Kept empty so a future exception is a
	// reviewed one-line addition.
	identifierGuardAllowedNames = map[string]bool{}
```

Run: `gofmt -l internal/configs && go build ./... && go test ./internal/configs/ . -run 'Config|Identifier|Boot' -count=1`
Expected: build ok, tests `ok`. If a configs test enumerates every Go field against `config.yaml` keys, it now passes with both sides removed; if it fails, it names what is left.

Also check the config audit memory trap: `git grep -n "AllowItemBuffRemoval\|DeathsShadowBuffId\|broken_limb_buff_duration\|BrokenLimbBuffDuration\|BuffsEnabled"` must print nothing outside `docs/superpowers/` history.

- [ ] **Step 4: Null-probe the weather key**

Revert `weather_config.go:111` to `get("BuffsEnabled")`; `TestBuildConfigCoercionAndClamps` must fail; restore.

- [ ] **Step 5: Commit**

```bash
git add internal/configs/config.gameplay.go internal/configs/config.balance.go internal/configs/config.balance.combat.go _datafiles/config.yaml modules/weather/weather_config.go modules/weather/weather_config_test.go modules/weather/context.md identifier_word_guard_test.go
git commit -m "feat(conditions): delete three dead buff-named config knobs, rename weather ConditionsEnabled key"
```

---

### Task 6: Go comments, docs and deletions

**Files:**
- Modify: Go comments that quote an old on-disk spelling (list produced in Step 1)
- Move/rewrite: `docs/schemas/buff.md` → `docs/schemas/condition.md`
- Modify: `docs/schemas/{item,mob,spell,room,behavior,pinnacle-items,schedule}.md`, `docs/README.md`
- Delete: `_datafiles/guides/building/scripting/SCRIPTING_BUFFS.md`; its link in `_datafiles/guides/building/scripting/README.md:14-15`; the ActorObject buff functions in `_datafiles/guides/building/scripting/FUNCTIONS_ACTORS.md:39-43,327-361`; the "Available triggers" line `docs/schemas/mob.md:200`
- Modify: `internal/*/context.md`, `modules/*/context.md`, `.claude/skills/*/SKILL.md` that name old disk spellings

- [ ] **Step 1: List the comment and doc hits**

Four files name old spellings on purpose and are excluded here and in the Task 7 guard: `wire_freeze_test.go` and `internal/keywords/keywords_setcondition_alias_test.go` (assert the old spellings are gone), `internal/migration/0.17.0.go` and `internal/migration/0.17.0_test.go` (rename them). Only their comments may still need a look.

```bash
git grep -n -i "buff" -- '*.go' ':(exclude)vendor' ':(exclude)identifier_word_guard_test.go' ':(exclude)wire_freeze_test.go' ':(exclude)internal/keywords/keywords_setcondition_alias_test.go' ':(exclude)internal/migration/0.17.0.go' ':(exclude)internal/migration/0.17.0_test.go' ':(exclude)internal/conditionrename' | grep -v -i "buffer\|buffet\|buffed\|rebuff\|Buffalo" > C:/tmp/slice3-go-comments.txt
git grep -n -i "buff" -- '*.md' ':(exclude)docs/superpowers' ':(exclude)tools/playtest' ':(exclude)tools/_archive' ':(exclude)docs/PATCH_NOTES.md' | grep -v -i "buffer\|buffet" > C:/tmp/slice3-docs.txt
wc -l C:/tmp/slice3-go-comments.txt C:/tmp/slice3-docs.txt
```

Every remaining `.go` hit is a comment (Task 3 renamed strings; Task 1-5 removed identifiers). `docs/PATCH_NOTES.md` is published history and stays.

- [ ] **Step 2: Fix Go comments**

For each line in `C:/tmp/slice3-go-comments.txt`: a comment that quotes an on-disk or wire spelling (`buffid`, `buffids`, `buff_ids`, `permabuff`, `buffs/`, `effect_type: buff`, `add_buff`, `mob_has_buff`, `buff-apply`, `buffname`, `BuffsEnabled`, a mutation effect type, `melee_self_buff`) gets the new spelling from the name map. A comment that uses "buff" as prose for the concept gets "condition" (or "harmful condition" for debuff). A comment that says something is "kept until slice 3" or "renamed in slice 3" is rewritten to state the current fact without the slice history. Leave commit-history quotes inside `// Built <date> from PRE-migration code` style provenance lines alone only if they quote a golden file's header verbatim; otherwise rename.

Run after: the same `git grep` command as Step 1 (with its exclusions), without the redirect.
Expected: no output.

- [ ] **Step 3: The condition schema doc**

```bash
git mv docs/schemas/buff.md docs/schemas/condition.md
```

In `docs/schemas/condition.md`: delete the opening banner that says the YAML keys and folder keep the buff spelling until slice 3; retitle to "Condition YAML schema"; rename every key (`buffid` → `conditionid`, `start_remove_buffs` → `start_remove_conditions`), the folder (`_datafiles/world/*/buffs/` → `conditions/`) and the filename formula (`{buffid}-{name}.yaml` → `{conditionid}-{name}.yaml`); delete section 5's JavaScript scripting part (lines 245-258 at `a4c078116`: companion `.js` files and `GiveBuff`/`HasBuff` examples), because no scripting layer exists (no `internal/scripting`, zero `.js` under `_datafiles/world`).

`docs/schemas/item.md`, `mob.md`, `spell.md`, `room.md`, `behavior.md`, `pinnacle-items.md`, `schedule.md`: rename every key, value and node name per the name map (`buffids`, `wornbuffids`, `critbuffids`, `buff_ids`, `effect_type: buff`, `spawninfo.buffids`, `mob_has_buff`, `add_buff`, `remove_buff`, `melee_self_buff`, `pinnacle_bandolier_buffs`); prose "buff" becomes "condition". In `docs/schemas/mob.md`, delete the whole line 200 ("**Available triggers:** `combat_start`, ... `has_buff:N`, `missing_buff:N`"): no trigger on it has a parser (the only similar name is the behaviour tree's `mob_health_below`).

`docs/README.md:20`: `(room, mob, item, spell, buff, dialogue,` → `(room, mob, item, spell, condition, dialogue,`. Update any other `docs/README.md` row linking `schemas/buff.md`.

- [ ] **Step 4: Delete the dead scripting docs**

```bash
git rm _datafiles/guides/building/scripting/SCRIPTING_BUFFS.md
```

In `_datafiles/guides/building/scripting/README.md` delete the `# Buff Scripting` heading and its `See [Buff Scripting](SCRIPTING_BUFFS.md)` line (lines 14-15) and a now-doubled blank line. In `FUNCTIONS_ACTORS.md` delete the five table-of-contents lines 39-43 (`ActorObject.HasBuff` ... `ActorObject.RemoveBuff`) and the five sections from `## [ActorObject.HasBuff(buffId int) bool]` through the end of the `## [ActorObject.RemoveBuff(buffId int)]` section (lines 327-361 at `a4c078116`; delete by heading, not number).

- [ ] **Step 5: context.md files and skills**

For every `context.md` and `.claude/skills/*/SKILL.md` line in `C:/tmp/slice3-docs.txt`: rename disk/wire spellings; delete statements that say a spelling is kept "until slice 3"; in `internal/conditions/context.md` delete the paragraph (around lines 7-23) that lists what slice 2 left for slice 3 and replace it with one sentence: "Since conditions unification slice 3 (2026-09-15) every file, save key, GMCP field, category string, colour alias and template function also says condition; `internal/conditionrename` holds the spelling map and migration 0.17.0 converted old saves." In `.claude/skills/dogmud-authoring-content/SKILL.md`, the `tools/id_inventory.py` mention lists `conditions`, not `buffs`.

Then run `python tools/context_md_audit.py` (if it takes arguments, read its `--help` first) for the touched packages and fix any phantom symbol it reports.

- [ ] **Step 6: Verify the docs are clean and links resolve**

```bash
git grep -n -i "buff" -- '*.md' ':(exclude)docs/superpowers' ':(exclude)tools/playtest' ':(exclude)tools/_archive' ':(exclude)docs/PATCH_NOTES.md' | grep -v -i "buffer\|buffet"
```

Expected: no output. Then check relative links in the files you edited: for each `](...)` target in `docs/schemas/*.md`, `docs/README.md` and `_datafiles/guides/building/scripting/*.md`, confirm the file exists (for example with the link-check loop used for PR #133).

- [ ] **Step 7: Commit**

```bash
git diff --name-only -z | xargs -0 git add --
git status --short | grep -v "^[MRD] "
git commit -m "docs(conditions): slice 3 comments, schema docs and context files; delete docs for the nonexistent scripting layer"
```

The `git status` line must print nothing.

---

### Task 7: The guard

**Files:**
- Modify: `identifier_word_guard_test.go`

- [ ] **Step 1: Add the string and data guard**

Add `"github.com/GoMudEngine/GoMud/internal/conditionrename"` and `"os/exec"` to the imports, update the header comment's first paragraph to cite both slice 2 and slice 3 (`docs/superpowers/specs/2026-09-15-conditions-unification-slice-3-disk-wire-design.md`), and append:

```go
// stringDataAllowlist pardons a buff spelling that is deliberate. Keyed
// "relpath|exact line content (trimmed)"; each entry must still match.
var stringDataAllowlist = map[string]string{
	`_datafiles/world/dogmud/dialogue/newcomer_antechamber/9491.yaml|- keywords: ["condition", "conditions", "buff", "effect"]`: "player keyword: a new player who types 'buff' is still understood",
}

// stringGuardNamesOldSpellings are Go files whose job is to name the old
// spellings: tests that assert they no longer bind or resolve, and the
// migration that renames them in saves.
var stringGuardNamesOldSpellings = map[string]bool{
	"wire_freeze_test.go":                                   true,
	"internal/keywords/keywords_setcondition_alias_test.go": true,
	"internal/migration/0.17.0.go":                          true,
	"internal/migration/0.17.0_test.go":                     true,
}

var stringDataExts = map[string]bool{
	".yaml": true, ".yml": true, ".template": true, ".html": true, ".js": true, ".css": true, ".md": true, ".golden": true,
}

// TestNoStringOrDataSaysBuff is the slice 3 half of this guard: Go string
// literals and struct tags (go/scanner STRING tokens), and every tracked data,
// template, web, golden and doc file under _datafiles/, docs/schemas/ and
// internal/**/testdata, must not spell buff outside the protected words
// conditionrename.HasBuff ignores.
func TestNoStringOrDataSaysBuff(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	require := func(cond bool, format string, args ...any) {
		t.Helper()
		if !cond {
			t.Fatalf(format, args...)
		}
	}
	require(ok, "runtime.Caller(0) failed")
	root := filepath.Dir(here)
	guardFile := filepath.Base(here)

	out, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	require(err == nil, "git ls-files: %v", err)

	seen := map[string]bool{}
	scannedGo, scannedData := 0, 0
	for _, rel := range strings.Split(string(out), "\x00") {
		if rel == "" {
			continue
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "vendor/") || strings.HasPrefix(rel, "docs/superpowers/") ||
			strings.HasPrefix(rel, "tools/playtest/") || strings.HasPrefix(rel, "tools/_archive/") ||
			rel == "docs/PATCH_NOTES.md" || rel == guardFile || strings.HasPrefix(rel, "internal/conditionrename/") ||
			stringGuardNamesOldSpellings[rel] {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		src, rerr := os.ReadFile(path)
		if rerr != nil {
			if os.IsNotExist(rerr) {
				continue
			}
			t.Fatalf("read %s: %v", rel, rerr)
		}

		if strings.HasSuffix(rel, ".go") {
			scannedGo++
			fset := token.NewFileSet()
			tf := fset.AddFile(path, fset.Base(), len(src))
			var sc scanner.Scanner
			sc.Init(tf, src, nil, 0)
			for {
				pos, tok, lit := sc.Scan()
				if tok == token.EOF {
					break
				}
				if tok == token.STRING && conditionrename.HasBuff(lit) {
					t.Errorf("%s:%d: string literal %s still says buff (%s)", rel, fset.Position(pos).Line, lit, identifierGuardSpecPath)
				}
			}
			continue
		}

		inScope := strings.HasPrefix(rel, "_datafiles/") || strings.HasPrefix(rel, "docs/schemas/") ||
			(strings.HasPrefix(rel, "internal/") && strings.Contains(rel, "/testdata/"))
		if !inScope || !stringDataExts[filepath.Ext(rel)] {
			continue
		}
		scannedData++
		for i, line := range strings.Split(string(src), "\n") {
			if !conditionrename.HasBuff(line) {
				continue
			}
			key := rel + "|" + strings.TrimSpace(line)
			if _, ok := stringDataAllowlist[key]; ok {
				seen[key] = true
				continue
			}
			t.Errorf("%s:%d: still says buff: %s (%s)", rel, i+1, strings.TrimSpace(line), identifierGuardSpecPath)
		}
	}
	require(scannedGo > 500, "scanned only %d Go files", scannedGo)
	require(scannedData > 3000, "scanned only %d data files", scannedData)
	for key := range stringDataAllowlist {
		if !seen[key] {
			t.Errorf("stringDataAllowlist entry %q matched nothing; update or remove it", key)
		}
	}
}
```

Also change `const identifierGuardSpecPath` to the slice 3 spec path `docs/superpowers/specs/2026-09-15-conditions-unification-slice-3-disk-wire-design.md` and update the two existing `t.Errorf` messages from "slice 2 of the conditions unification renamed these" to "the conditions unification renamed these".

- [ ] **Step 2: Run it**

Run: `go test . -run 'TestNoStringOrDataSaysBuff|TestNoIdentifierSaysBuff|TestNoTemplateReadsABuffField' -count=1 -v 2>&1 | tail -20`
Expected: PASS. Every failure is a leftover: fix the file (not the allowlist) unless the spelling is a deliberate player keyword like the dialogue line, and then the allowlist entry needs a reason a reviewer would accept.

- [ ] **Step 3: Null-probe**

1. Append `# buffid: 1` to `_datafiles/world/dogmud/conditions/0-meditating.yaml`; confirm FAIL naming that file and line; revert with `git checkout -- _datafiles/world/dogmud/conditions/0-meditating.yaml` and then `git status --short` to confirm the checkout staged nothing.
2. In `internal/pets/pets.go:27` change `yaml:"conditionids,omitempty"` to `yaml:"buffids,omitempty"`; confirm FAIL naming `internal/pets/pets.go`; restore the same way.
3. Delete the dialogue allowlist entry; confirm FAIL on `9491.yaml:37`; restore.

- [ ] **Step 4: Commit**

```bash
git add identifier_word_guard_test.go
git commit -m "test(conditions): guard Go strings, tags, data, templates and docs against buff spellings"
```

---

### Task 8: Gates, rehearsal, playtest, PR

- [ ] **Step 1: Full local gates**

```bash
cd C:/tmp/dogmud-slice3
gofmt -l internal/ modules/ *.go
go vet ./...
DOGMUD_BOOT_SMOKE=1 go test ./... -count=1 2>&1 | grep -E "^(FAIL|--- FAIL|panic)"
golangci-lint run --new-from-rev=master
node --test tools/jstest/ 2>&1 | tail -3
for f in tools/webclient-tests/*.js; do node "$f" >/dev/null 2>&1 || echo "FAIL $f"; done
```

Expected: `gofmt` and the `go test` grep print only the two known `internal/rooms` zone tests; `golangci-lint` reports 0 issues; node tests pass except `tools/webclient-tests/quest-panel-refresh.js`, which fails on master too (`writeQuestsHTML is not defined`). Remove any `ziggurat_test_zone/` or `rename_probe_zone/` folder the rooms tests left under `_datafiles/world/dogmud/rooms/`.

- [ ] **Step 2: Migration rehearsal on real saves**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
git worktree add --detach C:/tmp/dogmud-slice3-rehearsal feature/conditions-unification-slice-3-disk-wire
cp _datafiles/world/dogmud/users/*.yaml C:/tmp/dogmud-slice3-rehearsal/_datafiles/world/dogmud/users/
[ -d _datafiles/world/dogmud/rooms.instances ] && cp -r _datafiles/world/dogmud/rooms.instances C:/tmp/dogmud-slice3-rehearsal/_datafiles/world/dogmud/
grep -l "buffs:\|buffid:" C:/tmp/dogmud-slice3-rehearsal/_datafiles/world/dogmud/users/*.yaml | wc -l
cd C:/tmp/dogmud-slice3-rehearsal
printf 'Server:\n  CurrentVersion: 0.16.0\n' > _datafiles/world/dogmud/config-overrides.yaml
go build -o boot-check.exe .
env -u CONFIG_PATH timeout 180 ./boot-check.exe > boot1.log 2>&1; echo "exit $?"
grep -c "Migration 0.17.0" boot1.log
grep -c "Migration 0.1[0-6]" boot1.log
grep -cE "^panic:|goroutine [0-9]+ \[running\]|runtime error" boot1.log
grep -c "Server Ready" boot1.log
grep -n "CurrentVersion" _datafiles/world/dogmud/config-overrides.yaml
grep -l "buffs:\|buffid:\|permabuff:\|buffids:\|trapbuffids:" _datafiles/world/dogmud/users/*.yaml; echo "left: $?"
```

The users copied are this machine's gitignored dev saves (the owner's). They are copied into the rehearsal worktree only; the main checkout's files are never modified. Why the overrides file: the version lives in `Server.CurrentVersion`, which `migration.Run` writes with `configs.SetVal` to `<DataFiles>/config-overrides.yaml` or `$CONFIG_PATH` (`internal/configs/configs.go:422-428`); shipped `config.yaml` has no value, and an absent value defaults to 0.9.0 (`internal/configs/config.server.go:57-58`), which would run every migration since 0.9.1. Setting 0.16.0 makes this boot run exactly 0.17.0, as the droplet will.

Expected: exit 124 (server stayed up); `Migration 0.17.0` lines present (start line, one per rewritten file, count line); `Migration 0.1[0-6]` count `0`; zero panics; one `Server Ready`; the overrides file now says `CurrentVersion: 0.17.0`; the final grep lists no file (`left: 1`). Then boot a second time:

```bash
env -u CONFIG_PATH timeout 120 ./boot-check.exe > boot2.log 2>&1; grep -c "Migration 0.17.0" boot2.log
```

Expected: `0` (version already 0.17.0, `Run` returns before any migration). Spot-check one migrated save that had `buffs:`: `grep -n -A4 "conditions:" _datafiles/world/dogmud/users/<id>.yaml`.

Stop and remove: the `timeout` already stopped the server; confirm with `tasklist | grep boot-check` (kill only that PID if listed), then `cd "C:/Users/Calabe Davis/workspace/DOGMud" && git worktree remove --force C:/tmp/dogmud-slice3-rehearsal`, and if Windows holds a lock, PowerShell `Remove-Item -Recurse -Force C:\tmp\dogmud-slice3-rehearsal` then `git worktree prune`.

- [ ] **Step 3: Boot check (both worlds load)**

The rehearsal boot already booted the dogmud world from the branch. Also run the root boot smoke test: `cd C:/tmp/dogmud-slice3 && DOGMUD_BOOT_SMOKE=1 go test . -run Boot -count=1`. Expected: `ok`.

- [ ] **Step 4: Smoke playtest**

Write `tools/playtest/goals/2026-09-15-conditions-slice-3-smoke.yaml` (goal files for local runs need `ephemeral:`; read `.claude/skills/dogmud-playtesting` guidance and `tools/playtest/goals/2026-09-14-conditions-slice-2-smoke-player.yaml` for the exact shape first):

```yaml
# Conditions unification slice 3 smoke test, PLAYER half. Every on-disk
# spelling changed from buff to condition; nothing may play differently.
# Quote output VERBATIM. 3 commands per round on the AI port.
ephemeral:
  profile: early
  start_room: 3015
  budgets:
    wall_clock: 15m

goals:
  - >-
    Type `conditions` and quote the full output.
  - >-
    Fight the hostiles in the room until you are bleeding. Quote the
    conditions list while "Bleeding" is shown, and quote one bleed line.
  - >-
    Type `help species` for your own species and quote the Conditions line
    (it uses a renamed template function and colour).
  - >-
    Type `spells`. If you know Iron Will (its spell file used `effect_type:
    buff`), cast it on yourself, quote the cast and effect lines, then type
    `conditions` and quote its row. If you do not know it, say so and skip.
  - >-
    Type `spells`. If you know a summoning spell for a fire elemental or a
    vampire (both use the renamed `melee_self_empower` behaviour), cast it,
    fight one hostile with it beside you, and quote any template error or
    silence from the summon. If you know neither, say so and skip.
  - >-
    Judge every line: quote template errors ("can't evaluate field", "function
    not defined"), raw internal names, the word "buff", or a colour code
    printed as text.
```

and `tools/playtest/goals/2026-09-15-conditions-slice-3-smoke-admin.yaml`:

```yaml
# Conditions unification slice 3 smoke test, ADMIN half.
ephemeral:
  profile: admin
  start_room: 5000
  budgets:
    wall_clock: 15m

goals:
  - >-
    Type `setcondition list` and quote the header and first five rows.
  - >-
    Type `buff list` and quote the result; the alias was removed, so it must
    not run the command.
  - >-
    Type `help setcondition` and quote it; it must not mention "buff".
  - >-
    Apply condition 1 to yourself with `setcondition 1`, type `conditions`,
    and quote the row.
  - >-
    Judge every line for template errors, raw internal names, or the word
    "buff".
```

Run each with the playtest skill (`/playtest local --checkout C:/tmp/dogmud-slice3 feature-tester <goals file>`), single agent. Tear down each container with `docker rm -f` by its run id after (`playtestrun stop` leaves it running). Extract findings into the arc memory file `project-conditions-unification-arc.md` (reports are gitignored).

Expected: PASS on every goal; any template error or "buff" in output is a defect to fix before the PR.

- [ ] **Step 5: Index, push, PR**

Add a row for this plan to `docs/README.md` directly after the slice 3 spec row, in the same style. Commit the two goals files and the README row:

```bash
git add tools/playtest/goals/2026-09-15-conditions-slice-3-smoke.yaml tools/playtest/goals/2026-09-15-conditions-slice-3-smoke-admin.yaml docs/README.md
git commit -m "test(conditions): slice 3 smoke playtest goals; index the plan"
git push -u origin feature/conditions-unification-slice-3-disk-wire
gh pr create --repo pruuk/DOGMud --base master --head feature/conditions-unification-slice-3-disk-wire --title "Conditions unification slice 3: disk, wire and save migration" --body-file <body>
```

PR body must include: what changed (summary of the name map), the proof's last line, the rehearsal result (files rewritten, zero left, no rerun on second boot), playtest run ids and results, the admin-facing changes (`buff` alias removed; builder GMCP field names `conditionIds`, `wornConditionIds`, `conditions`, `apply_condition`, `conditionid`), and the rollout notes:
- first boot on the droplet runs 0.17.0 and copies all datafiles to a temp dir first: check free disk before deploying;
- on error the server restores the backup and exits, and the log names the file;
- snapshot `users/` and `rooms.instances/` before deploying;
- a local skip-worktree `config.yaml` still saying `BuffsEnabled` turns weather conditions on (fallback true): update the local copy;
- CI `validate / lint` is expected red from the >300-file diff API 406; local `golangci-lint run --new-from-rev=master` is the real check.

Read the URL `gh` prints and confirm it says `pruuk/DOGMud`. Watch checks with `gh pr checks <n> --repo pruuk/DOGMud --watch`, confirm the expected runs with `gh run list --repo pruuk/DOGMud --branch feature/conditions-unification-slice-3-disk-wire`, and merge with `--merge --delete-branch` when green apart from the explained lint red. Do not deploy.
