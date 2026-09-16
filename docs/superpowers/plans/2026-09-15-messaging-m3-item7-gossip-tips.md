# Messaging M3 item 7: gossip store, tips rename and migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Gossip templates become an `internal/gossip` store rendering through the narration core; the hints broadcast becomes an `internal/tips` store named tips everywhere, including a 0.18.0 migration of each player's saved on/off flag.

**Architecture:** Spec: `docs/superpowers/specs/2026-09-15-messaging-m3-item7-gossip-tips-design.md` (owner-approved 2026-09-15; read its rulings and facts table first). Gossip copies the Kind A single-role store shape of `internal/itemvoices` (a pool, `narration.Render` with `DefaultPicker`). Tips is a plain store with rotation; the narration core adds nothing there. Conversations are not touched.

**Tech Stack:** Go, `gopkg.in/yaml.v2`, `internal/narration`, `internal/migration`, root-package guard tests.

**Branch:** `feature/messaging-m3-item7-conversations-gossip-hints` (spec committed at `5225680d8`).

**One deliberate difference from the spec's sketch:** the spec wrote `SeedForTest(t testing.TB, ...)`. This plan uses the repo's existing seed style instead, `SeedForTest(...) func()` returning a restore func (as `itemvoices.SeedVoicesForTest` and `spells.SeedSpellsForTest` do), so production packages do not import `testing`.

---

## Rules for every implementer (read first)

- **Edit tool only** for changing files, except `git mv` where a task says so. Never a Python read-modify-write.
- **Never `git add -A` or `git add .`**; add named paths only. **Never `git reset`**, never `git checkout -- <path>` (it stages the revert). One implementer at a time.
- `grep -c` that finds zero exits 1 and breaks `&&` chains: run "expect zero" checks standalone.
- Run `go test .` (repo root guards) in every task that touches Go code.
- No em dashes or en dashes anywhere you write.
- `-update` on narration goldens is allowed ONLY in Task 0.
- `internal/hooks` has known shuffle-order flakes (`TestWaitRound_*`, `TestBuildGossipLine_*`); rerun once if one fails and report both runs. After Task 2 the gossip ones should stop flaking; say so if they still do.
- Commit via heredoc (`git commit -F - <<'EOF'`); end the message with the Co-Authored-By line your session's system reminder specifies.

## File map

| File | Responsibility | Task |
|---|---|---|
| `internal/narration/snapshot_test.go`, `testdata/stores/gossip.golden`, `tips.golden` | frozen gossip substitution and tip text | 0, 2, 3 |
| `internal/gossip/gossip.go` (new) | load, validate, pool lookup, render | 1 |
| `internal/gossip/test_helpers.go`, `gossip_test.go` (new) | seeds and tests | 1 |
| `narration_render_callers_guard_test.go` | register gossip as a Kind A caller | 1 |
| `internal/hooks/MobIdle_HandleIdleMobs.go`, `hooks_test.go` | gossip reads the store | 2 |
| `main.go` | `gossip.Load()`, `tips.Load()`, `VERSION` | 2, 3, 5 |
| `internal/tips/tips.go`, `test_helpers.go`, `tips_test.go` (new) | tips store | 3 |
| `_datafiles/world/dogmud/hints.yaml` -> `tips.yaml` | the tip text | 3 |
| `internal/hooks/NewRound_BroadcastHints.go` -> `NewRound_BroadcastTips.go`; `Looking_HandleLookHints.go` -> `Looking_HandleLookTips.go`; `hooks.go`; `hooks_test.go`; new `broadcast_tips_test.go` | renamed listeners | 3 |
| `internal/usercommands/set.go`, new `set_tips_test.go`; both worlds' `templates/help/set.template` | `set tips` | 4 |
| `internal/migration/0.18.0.go`, `0.18.0_test.go`, `migration.go` | the saved-flag migration | 5 |
| `store_data_files_guard_test.go` (new, root); `messaging_surface_guard_test.go`; `internal/hooks/spell_resolution.go` comment | guards | 6 |
| `context.md` x5, arc spec, `docs/PATCH_NOTES.md`, `docs/README.md` | docs | 7 |

---

### Task 0: Record `gossip.golden` and `tips.golden` from pre-migration data

Today's gossip picks with `util.Rand`, which no test can pin, so the golden freezes each variant's substitution exactly as today's code performs it. Tips are frozen in file order.

**Files:**
- Modify: `internal/narration/snapshot_test.go`
- Create (generated): `internal/narration/testdata/stores/gossip.golden`, `tips.golden`

- [ ] **Step 1: Imports**

Add to the import block (keep it sorted): `"gopkg.in/yaml.v2"` at the end of the third-party group. `os`, `path/filepath`, `sort`, `strings` are already imported.

- [ ] **Step 2: Subtests**

In `TestSnapshotStores`, after the `crafting` subtest and before `post_pipeline`:

```go
	t.Run("gossip", func(t *testing.T) {
		checkGolden(t, "gossip.golden", buildGossipGolden(t))
	})
	t.Run("tips", func(t *testing.T) {
		checkGolden(t, "tips.golden", buildTipsGolden(t))
	})
```

- [ ] **Step 3: Pre-migration builders at the end of the file**

```go
// Store 12: gossip templates (_datafiles/world/dogmud/gossip_templates.yaml)
//
// Built from PRE-migration data and code: the YAML is parsed here and each
// variant is substituted exactly as internal/hooks does it today. Today's code
// picks with util.Rand, which no test can pin, so this golden freezes every
// variant's substitution, not a pick. M3 item 7 Task 2 switches the builder to
// the gossip store; the rows, their order and the header must not change.
const gossipStandIn = "<the stand-in event>"

func buildGossipGolden(t *testing.T) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(dogmudDataDir(t), "gossip_templates.yaml"))
	if err != nil {
		t.Fatalf("read gossip_templates.yaml: %v", err)
	}
	templates := map[string][]string{}
	if err := yaml.Unmarshal(raw, &templates); err != nil {
		t.Fatalf("parse gossip_templates.yaml: %v", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# gossip store snapshot\n")
	fmt.Fprintf(&b, "# Built 2026-09-15 from PRE-migration code. Every variant of every key, substituted\n")
	fmt.Fprintf(&b, "# as the gossiper sends it: event keys replace {desc} once, fact- keys replace every\n")
	fmt.Fprintf(&b, "# {description}, fallback lines are sent as written. The stand-in description is\n")
	fmt.Fprintf(&b, "# %q. The pick itself (util.Rand) is not frozen; no test can pin it.\n", gossipStandIn)
	fmt.Fprintf(&b, "# dimensions: key x variant index\n\n")

	keys := make([]string, 0, len(templates))
	for k := range templates {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		t.Fatal("no gossip keys parsed")
	}
	for _, key := range keys {
		for i, line := range templates[key] {
			var rendered string
			switch {
			case key == "fallback":
				rendered = line
			case strings.HasPrefix(key, "fact-"):
				rendered = strings.ReplaceAll(line, "{description}", gossipStandIn)
			default:
				rendered = strings.Replace(line, "{desc}", gossipStandIn, 1)
			}
			fmt.Fprintf(&b, "gossip|%s|%d => %s\n", key, i, rendered)
		}
	}
	return b.String()
}

// Store 13: tips (the periodic broadcast; hints.yaml before M3 item 7)
//
// Built from PRE-migration data: the `hints:` list of hints.yaml in file order,
// which is the broadcast's rotation order. M3 item 7 Task 3 renames the file
// and switches this builder to the tips store; rows and header must not change.
func buildTipsGolden(t *testing.T) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(dogmudDataDir(t), "hints.yaml"))
	if err != nil {
		t.Fatalf("read hints.yaml: %v", err)
	}
	var file struct {
		Hints []string `yaml:"hints"`
	}
	if err := yaml.Unmarshal(raw, &file); err != nil {
		t.Fatalf("parse hints.yaml: %v", err)
	}
	if len(file.Hints) == 0 {
		t.Fatal("no tips parsed")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# tips store snapshot\n")
	fmt.Fprintf(&b, "# Built 2026-09-15 from PRE-migration data. Every tip in rotation order, as the text\n")
	fmt.Fprintf(&b, "# after the [Tip] prefix. dimensions: rotation index\n\n")
	for i, tip := range file.Hints {
		fmt.Fprintf(&b, "tip|%d => %s\n", i, tip)
	}
	return b.String()
}
```

- [ ] **Step 4: Record (the only `-update` in this plan)**

Run: `go test ./internal/narration/ -run 'TestSnapshotStores/(gossip|tips)$' -update`
Expected: `ok`.

- [ ] **Step 5: Verify**

Run: `go test ./internal/narration/ -run TestSnapshotStores` -> `ok` (13 subtests).
Run standalone: `grep -c "^gossip|" internal/narration/testdata/stores/gossip.golden` -> the total variant count (between 68 and 170; record it). Run standalone: `grep -c "^tip|" internal/narration/testdata/stores/tips.golden` -> `74`.
Run: `git status --short` -> only `snapshot_test.go` modified and the two new goldens. Any other golden changed means `-update` ran too broadly: STOP and report.

- [ ] **Step 6: Commit** `internal/narration/snapshot_test.go` and both goldens:

```
test(narration): gossip.golden and tips.golden recorded from pre-migration data
```

---

### Task 1: The gossip store

**Files:**
- Create: `internal/gossip/gossip.go`, `internal/gossip/test_helpers.go`, `internal/gossip/gossip_test.go`
- Modify: `narration_render_callers_guard_test.go:19-26`

- [ ] **Step 1: Failing tests**: `internal/gossip/gossip_test.go`:

```go
package gossip

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name      string
		templates map[string][]string
		wantErr   string
	}{
		{"shipped shape passes", map[string][]string{
			"fallback":                {"Quiet day.", "Nothing to report."},
			"MobCraftedRare-Regional": {"I heard {desc}"},
			"fact-default":            {"They say {description}"},
		}, ""},
		{"empty pool refused", map[string][]string{"fallback": {}}, "has no lines"},
		{"blank line refused", map[string][]string{"fallback": {"ok", "   "}}, "is blank"},
		{"repeated desc refused", map[string][]string{"X-Local": {"{desc} and {desc}"}}, "{desc} more than once"},
		{"repeated description refused", map[string][]string{"fact-default": {"{description}, {description}"}}, "{description} more than once"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.templates)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want an error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestRenderWith_OneDrawPerLine(t *testing.T) {
	for _, n := range []int{1, 2, 5} {
		pool := make([]string, n)
		for i := range pool {
			pool[i] = "line {desc}"
		}
		calls := 0
		counting := func(size int) int {
			calls++
			if size != n {
				t.Errorf("picker got size %d, want %d", size, n)
			}
			return 0
		}
		if got := renderWith(pool, "{desc}", "X", counting); got != "line X" {
			t.Errorf("pool of %d rendered %q, want %q", n, got, "line X")
		}
		if calls != 1 {
			t.Errorf("pool of %d drew %d times, want exactly 1 (today's util.Rand(len))", n, calls)
		}
	}
}

func TestRenderWith_EmptyPoolDrawsNothing(t *testing.T) {
	calls := 0
	if got := renderWith(nil, "{desc}", "X", func(int) int { calls++; return 0 }); got != "" || calls != 0 {
		t.Errorf("empty pool rendered %q with %d draws, want \"\" and 0", got, calls)
	}
}

func TestRenderWith_NoTokenLeavesLineAlone(t *testing.T) {
	if got := renderWith([]string{"Quiet {desc} day."}, "", "", func(int) int { return 0 }); got != "Quiet {desc} day." {
		t.Errorf("token-less render changed the line: %q", got)
	}
}

func TestRender_UsesTheDefaultPicker(t *testing.T) {
	src, err := os.ReadFile("gossip.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "renderWith(pool, token, value, narration.DefaultPicker)") {
		t.Error("Render must pass narration.DefaultPicker: gossip consumed one util.Rand draw per line before the migration, and FirstPicker would silently remove it and shift every later roll")
	}
}

func TestLoad_MissingFileIsAnEmptyStore(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = configs.ConfigString(t.TempDir())
	configs.SetConfigForTest(t, cfg)
	defer SeedForTest(map[string][]string{"fallback": {"stale"}})()

	Load()

	if got := Pool("fallback"); got != nil {
		t.Errorf("Pool after loading a world with no gossip file = %q, want nil", got)
	}
}

func TestLoad_ReadsAndValidatesTheFile(t *testing.T) {
	dir := t.TempDir()
	body := "fallback:\n  - \"Quiet day.\"\nX-Local:\n  - \"Near: {desc}\"\n"
	if err := os.WriteFile(filepath.Join(dir, "gossip_templates.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = configs.ConfigString(dir)
	configs.SetConfigForTest(t, cfg)
	defer SeedForTest(nil)()

	Load()

	if got := Render(Pool("X-Local"), "{desc}", "a bridge fell"); got != "Near: a bridge fell" {
		t.Errorf("rendered %q", got)
	}
	if keys := Keys(); len(keys) != 2 || keys[0] != "X-Local" || keys[1] != "fallback" {
		t.Errorf("Keys() = %q, want sorted [X-Local fallback]", keys)
	}
}
```

Run: `go test ./internal/gossip/` -> build FAIL (undefined `Validate`, `renderWith`, ...).

- [ ] **Step 2: The store**: `internal/gossip/gossip.go`:

```go
// Package gossip is the gossip template store: pools of lines a gossiping
// NPC says about recent world events and known facts, keyed
// "EventType-Significance[-Local|-Distant]", "fact-<id|tag|default>" and
// "fallback", loaded from DataFiles/gossip_templates.yaml.
//
// Which key and which token apply is decided by the gossiper in
// internal/hooks (buildGossipLine); this package owns the text, its
// validation and the pick.
package gossip

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/narration"
	"gopkg.in/yaml.v2"
)

const fileName = "gossip_templates.yaml"

// tokens a template may carry; each at most once per line.
var tokens = []string{"{desc}", "{description}"}

var templates map[string][]string

// Load reads DataFiles/gossip_templates.yaml. A world with no such file (the
// default world) gets an empty store. A read error other than a missing file,
// a parse error, or a Validate failure panics, as a bad recipe or spell does:
// before the store existed these were logged and gossip went silently empty
// for the life of the process.
func Load() {
	start := time.Now()
	path := string(configs.GetFilePathsConfig().DataFiles) + "/" + fileName

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			templates = map[string][]string{}
			mudlog.Info("gossip.Load()", "loadedKeys", 0, "note", "this world has no "+fileName)
			return
		}
		panic(fmt.Errorf("gossip: read %s: %w", path, err))
	}

	loaded := map[string][]string{}
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		panic(fmt.Errorf("gossip: parse %s: %w", path, err))
	}
	if err := Validate(loaded); err != nil {
		panic(fmt.Errorf("gossip: %s: %w", path, err))
	}

	templates = loaded
	mudlog.Info("gossip.Load()", "loadedKeys", len(templates), "Time Taken", time.Since(start))
}

// Validate refuses an empty pool, a blank line, and a line that carries the
// same token twice. The last rule is what keeps rendering byte-identical to
// the pre-store code, which replaced {desc} only once: with no repeats, once
// and every-occurrence substitution produce the same line.
func Validate(t map[string][]string) error {
	keys := make([]string, 0, len(t))
	for k := range t {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		pool := t[key]
		if len(pool) == 0 {
			return fmt.Errorf("key %q has no lines", key)
		}
		for i, line := range pool {
			if strings.TrimSpace(line) == "" {
				return fmt.Errorf("key %q line %d is blank", key, i)
			}
			for _, tok := range tokens {
				if strings.Count(line, tok) > 1 {
					return fmt.Errorf("key %q line %d uses %s more than once", key, i, tok)
				}
			}
		}
	}
	return nil
}

// Pool returns the lines for key, or nil.
func Pool(key string) []string {
	return templates[key]
}

// Keys returns every loaded key, sorted.
func Keys() []string {
	out := make([]string, 0, len(templates))
	for k := range templates {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Render picks one line from pool and substitutes token with value. It draws
// exactly once, through narration.DefaultPicker (util.Rand), as the pre-store
// code did with util.Rand(len(pool)). An empty token substitutes nothing.
func Render(pool []string, token, value string) string {
	return renderWith(pool, token, value, narration.DefaultPicker)
}

// renderWith is Render with an explicit picker. A gossiping NPC is the only
// speaker, so the pool is the Actor role and nothing else is authored.
func renderWith(pool []string, token, value string, pick narration.Picker) string {
	if len(pool) == 0 {
		return ""
	}
	var subs map[string]string
	if token != "" {
		subs = map[string]string{token: value}
	}
	return narration.Render(narration.Variants{Actor: pool}, subs, pick).Actor
}
```

`internal/gossip/test_helpers.go`:

```go
package gossip

import "github.com/GoMudEngine/GoMud/internal/narration"

// SeedForTest replaces the store and returns a restore func to defer.
// Intended for cross-package tests (hooks).
func SeedForTest(m map[string][]string) func() {
	old := templates
	templates = m
	return func() { templates = old }
}

// RenderWithForTest exposes the picker seam to the narration snapshot harness,
// an external test package.
func RenderWithForTest(pool []string, token, value string, pick narration.Picker) string {
	return renderWith(pool, token, value, pick)
}
```

- [ ] **Step 3: Register the caller**: in `narration_render_callers_guard_test.go`'s `narrationRenderCallers`, add (keep alphabetical):

```go
	"internal/gossip/gossip.go":            "Kind A: gossip template pools, single role (the gossiping NPC)",
```

- [ ] **Step 4: Test**

Run: `go test ./internal/gossip/ && go test . -run TestNarrationRenderCallers`. If the root test name differs, run `go test .`. Expected: `ok`.

- [ ] **Step 5: Probe**: temporarily change `Render` to pass `narration.FirstPicker`; `go test ./internal/gossip/ -run TestRender_UsesTheDefaultPicker` must FAIL; revert with Edit and re-read the line to confirm it passes `narration.DefaultPicker` again (the file is untracked, so `git diff` cannot show the revert); rerun, expect `ok`.

- [ ] **Step 6: Commit** the three gossip files and the guard file:

```
feat(gossip): gossip template store rendering through the narration core
```

---

### Task 2: Gossip reads the store

**Files:**
- Modify: `internal/hooks/MobIdle_HandleIdleMobs.go:375-418,429-560`, `internal/hooks/hooks_test.go:2968-3053`, `main.go` (after `itemvoices.LoadDataFiles()`), `internal/narration/snapshot_test.go`

- [ ] **Step 1: Delete the hook loader**

Delete the `gossipTemplates` / `gossipTemplatesOnce` var block and the whole `loadGossipTemplates` function. Change the comment above `eventTypeKey` from "used in gossip_templates.yaml" to "used in gossip store keys (internal/gossip)". Remove the now-unused imports the compiler names (`os`, `sync`, `gopkg.in/yaml.v2`) and add `"github.com/GoMudEngine/GoMud/internal/gossip"`.

- [ ] **Step 2: `buildGossipLine`**

Delete its first line, `gossipTemplatesOnce.Do(loadGossipTemplates)`. Replace:

```go
		if fallbacks, ok := gossipTemplates["fallback"]; ok && len(fallbacks) > 0 {
			return fallbacks[util.Rand(len(fallbacks))]
		}
```
with
```go
		if fallbacks := gossip.Pool("fallback"); len(fallbacks) > 0 {
			return gossip.Render(fallbacks, "", "")
		}
```

Replace the template lookup block from `templates, found := gossipTemplates[baseKey+"-"+distance]` through the final `return strings.Replace(tmpl, "{desc}", evt.Description, 1)` with:

```go
	templates := gossip.Pool(baseKey + "-" + distance)
	if len(templates) == 0 {
		// Fall back to base key without distance suffix
		templates = gossip.Pool(baseKey)
	}
	if len(templates) == 0 {
		// Try without significance
		for _, s := range []string{"Global", "Regional", "Local"} {
			if templates = gossip.Pool(typeStr + "-" + s); len(templates) > 0 {
				break
			}
		}
	}

	if len(templates) == 0 {
		// Final fallback: just say the description
		return fmt.Sprintf("I heard that %s", evt.Description)
	}

	return gossip.Render(templates, "{desc}", evt.Description)
```

Read the original block before replacing and confirm the lookup order is identical (distance key, base key, then `Global`, `Regional`, `Local`).

- [ ] **Step 3: `renderFactGossip`**: replace its body with:

```go
	if tmpls := gossip.Pool("fact-" + kf.Fact.Id); len(tmpls) > 0 {
		return gossip.Render(tmpls, "{description}", kf.Fact.Description)
	}
	for _, tag := range kf.Fact.Tags {
		if tmpls := gossip.Pool("fact-" + tag); len(tmpls) > 0 {
			return gossip.Render(tmpls, "{description}", kf.Fact.Description)
		}
	}
	if tmpls := gossip.Pool("fact-default"); len(tmpls) > 0 {
		return gossip.Render(tmpls, "{description}", kf.Fact.Description)
	}
	return ""
```

Every `util.Rand` draw that remains in `buildGossipLine` (fact choice, the 70/30 roll, event choice) is untouched.

- [ ] **Step 4: Hooks tests**: in each of the three `TestBuildGossipLine_*` tests, replace the three-line seeding pattern

```go
	gossipTemplatesOnce.Do(func() {})
	gossipTemplates = map[string][]string{ ... }
	defer func() { gossipTemplates = nil }()
```
with
```go
	defer gossip.SeedForTest(map[string][]string{ ... })()
```
(keeping each test's map contents exactly; for `_EmptyTemplatesEmptyEvents` the map is `map[string][]string{}`). Add the `gossip` import to `hooks_test.go`.

- [ ] **Step 5: Boot load**: in `main.go`'s `loadAllDataFiles`, after `itemvoices.LoadDataFiles()`:

```go
	// Messaging M3 item 7: gossip template store (loaded here, not lazily by
	// the first gossiping NPC, so a broken file fails boot instead of silencing
	// gossip).
	gossip.Load()
```
and add the import.

- [ ] **Step 6: Golden reads the store**: in `snapshot_test.go`: add `gossip.Load()` to `setupRealStores` after `crafting.LoadRecipeFiles()` and import `internal/gossip`. Replace the body of `buildGossipGolden` between the header `Fprintf`s and `return` (the YAML read and the substitution switch) so it reads:

```go
	keys := gossip.Keys()
	if len(keys) == 0 {
		t.Fatal("no gossip keys loaded; setupRealStores must call gossip.Load()")
	}
	for _, key := range keys {
		token, value := "{desc}", gossipStandIn
		switch {
		case key == "fallback":
			token, value = "", ""
		case strings.HasPrefix(key, "fact-"):
			token = "{description}"
		}
		pool := gossip.Pool(key)
		for i := range pool {
			index := i
			rendered := gossip.RenderWithForTest(pool, token, value, func(int) int { return index })
			fmt.Fprintf(&b, "gossip|%s|%d => %s\n", key, i, rendered)
		}
	}
```
and update the builder's doc comment's last sentence to: "Since M3 item 7 Task 2 it reads through the gossip store; rows, order and header are unchanged from the pre-migration recording, which is the byte-identity proof." Leave the printed header lines untouched. The `yaml` import may now be unused in this file only if Task 0's tips builder is also gone, which happens in Task 3; do not remove it here.

- [ ] **Step 7: Verify**

Run: `go build ./... && go test ./internal/hooks/ ./internal/gossip/ ./internal/narration/ . `
Expected: all `ok`, all goldens unchanged.

Run standalone: `grep -rn "gossipTemplates" --include=*.go internal` -> no output.

- [ ] **Step 8: Probe 1**: temporarily change the event-key call in `buildGossipGolden`'s post-migration switch default to use `"{description}"`; `go test ./internal/narration/ -run TestSnapshotStores/gossip` must FAIL on an event-key row; revert with Edit; rerun `ok`.

- [ ] **Step 9: Commit** `MobIdle_HandleIdleMobs.go`, `hooks_test.go`, `main.go`, `snapshot_test.go`:

```
refactor(gossip): gossiping NPCs read the gossip store; loaded at boot
```

---

### Task 3: The tips store, the file rename, the listener renames

**Files:**
- Create: `internal/tips/tips.go`, `internal/tips/test_helpers.go`, `internal/tips/tips_test.go`, `internal/hooks/broadcast_tips_test.go`
- Rename: `_datafiles/world/dogmud/hints.yaml` -> `tips.yaml`; `internal/hooks/NewRound_BroadcastHints.go` -> `NewRound_BroadcastTips.go`; `internal/hooks/Looking_HandleLookHints.go` -> `Looking_HandleLookTips.go`
- Modify: `internal/hooks/hooks.go:56,91`, `internal/hooks/hooks_test.go:2195-2219`, `main.go`, `internal/narration/snapshot_test.go`

- [ ] **Step 1: Failing store tests**: `internal/tips/tips_test.go`:

```go
package tips

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

func TestValidate(t *testing.T) {
	if err := Validate([]string{"Try help.", "Rest to heal."}); err != nil {
		t.Errorf("valid tips refused: %v", err)
	}
	if err := Validate([]string{"Try help.", "  "}); err == nil {
		t.Error("a blank tip was accepted")
	}
}

func TestNext_RotatesInOrderAndWraps(t *testing.T) {
	defer SeedForTest([]string{"a", "b", "c"})()
	var got []string
	for i := 0; i < 5; i++ {
		got = append(got, Next())
	}
	want := []string{"a", "b", "c", "a", "b"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rotation = %q, want %q", got, want)
		}
	}
}

func TestNext_EmptyStoreIsBlank(t *testing.T) {
	defer SeedForTest(nil)()
	if got := Next(); got != "" {
		t.Errorf("Next() on an empty store = %q, want \"\"", got)
	}
}

func TestLoad_MissingFileIsAnEmptyStore(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = configs.ConfigString(t.TempDir())
	configs.SetConfigForTest(t, cfg)
	defer SeedForTest([]string{"stale"})()

	Load()

	if Count() != 0 {
		t.Errorf("Count() = %d after loading a world with no tips file, want 0", Count())
	}
}

func TestLoad_ReadsTheTipsKey(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tips.yaml"), []byte("tips:\n  - first tip\n  - second tip\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = configs.ConfigString(dir)
	configs.SetConfigForTest(t, cfg)
	defer SeedForTest(nil)()

	Load()

	if all := All(); len(all) != 2 || all[0] != "first tip" || all[1] != "second tip" {
		t.Errorf("All() = %q", all)
	}
}
```

Run: `go test ./internal/tips/` -> build FAIL.

- [ ] **Step 2: The store**: `internal/tips/tips.go`:

```go
// Package tips is the periodic gameplay tip store: short pieces of advice
// broadcast to every player who has not turned them off (`set tips`), one per
// interval, in file order. Loaded from DataFiles/tips.yaml. Before messaging
// M3 item 7 these were called hints, a name that collided with the quest
// `hint` command and with dialogue hints.
//
// Tips are sent as one line each with no server-side wrap; see the messaging
// M6 content ledger, row 22.
package tips

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

const fileName = "tips.yaml"

var (
	all  []string
	next int
)

// Load reads DataFiles/tips.yaml (key `tips:`). A world with no such file gets
// an empty store. Any other read error, a parse error or a Validate failure
// panics. The rotation position survives a reload.
func Load() {
	start := time.Now()
	path := string(configs.GetFilePathsConfig().DataFiles) + "/" + fileName

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			all = nil
			mudlog.Info("tips.Load()", "loadedCount", 0, "note", "this world has no "+fileName)
			return
		}
		panic(fmt.Errorf("tips: read %s: %w", path, err))
	}

	var file struct {
		Tips []string `yaml:"tips"`
	}
	if err := yaml.Unmarshal(data, &file); err != nil {
		panic(fmt.Errorf("tips: parse %s: %w", path, err))
	}
	if err := Validate(file.Tips); err != nil {
		panic(fmt.Errorf("tips: %s: %w", path, err))
	}

	all = file.Tips
	mudlog.Info("tips.Load()", "loadedCount", len(all), "Time Taken", time.Since(start))
}

// Validate refuses a blank tip. There is deliberately no length rule: most
// shipped tips run past 80 characters, and wrapping is the messaging arc's M5
// decision, not an authoring rule this store can enforce.
func Validate(t []string) error {
	for i, tip := range t {
		if strings.TrimSpace(tip) == "" {
			return fmt.Errorf("tip %d is blank", i)
		}
	}
	return nil
}

// Next returns the next tip in rotation and advances, or "" for an empty store.
func Next() string {
	if len(all) == 0 {
		return ""
	}
	tip := all[next%len(all)]
	next++
	return tip
}

// Count is the number of loaded tips.
func Count() int { return len(all) }

// All returns a copy of the loaded tips in rotation order.
func All() []string {
	return append([]string(nil), all...)
}
```

`internal/tips/test_helpers.go`:

```go
package tips

// SeedForTest replaces the store, resets the rotation, and returns a restore
// func to defer. Intended for cross-package tests (hooks, narration).
func SeedForTest(t []string) func() {
	oldAll, oldNext := all, next
	all, next = t, 0
	return func() { all, next = oldAll, oldNext }
}
```

Run: `go test ./internal/tips/` -> `ok`.

- [ ] **Step 3: Rename the data file**

Run: `git mv _datafiles/world/dogmud/hints.yaml _datafiles/world/dogmud/tips.yaml`

Then with Edit, replace the file's header (the comment lines and `hints:` key, everything above the first `  - >-`) with:

```yaml
# Periodic gameplay tips, broadcast to every player who has not turned them
# off with `set tips`, one every few minutes, in the order listed here.
#
# Each entry is a folded scalar (>-): its line breaks become spaces when the
# file loads, so a tip is sent as ONE line and the player's client wraps it.
# Keep tips short, but do not rely on the line breaks below for layout.
# Use <ansi fg="command">command</ansi> to highlight commands.

tips:
```

Tip entries are unchanged.

- [ ] **Step 4: Rename the listeners**

Run: `git mv internal/hooks/NewRound_BroadcastHints.go internal/hooks/NewRound_BroadcastTips.go` and `git mv internal/hooks/Looking_HandleLookHints.go internal/hooks/Looking_HandleLookTips.go`.

Replace the entire content of `NewRound_BroadcastTips.go` with:

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/tips"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// BroadcastTips sends the next gameplay tip (internal/tips) to every active
// player who is not deafened and has not turned tips off with `set tips`,
// every tipIntervalRounds rounds (~5 minutes at 4-second rounds).

const (
	tipIntervalRounds = 75
	tipsConfigKey     = `tips`
)

func BroadcastTips(e events.Event) events.ListenerReturn {
	evt := e.(events.NewRound)

	if evt.RoundNumber%tipIntervalRounds != 0 {
		return events.Continue
	}

	tip := tips.Next()
	if tip == `` {
		return events.Continue
	}

	fullText := `<ansi fg="cyan-bold">[Tip]</ansi> ` + tip

	for _, u := range users.GetAllActiveUsers() {
		if u.Deafened {
			continue
		}
		// Tips default to ON (nil = on).
		if on, ok := u.GetConfigOption(tipsConfigKey).(bool); ok && !on {
			continue
		}
		u.SendText(messaging.CategoryTip, fullText)
	}

	// Also fire a Communication event for the web client's comms window.
	events.AddToQueue(events.Communication{
		CommType: "broadcast",
		Name:     "Tip",
		Message:  tip,
	})

	return events.Continue
}
```

In `Looking_HandleLookTips.go`, rename the function `HandleLookHints` to `HandleLookTips`; nothing else changes. In `hooks.go` change the two registrations to `BroadcastTips` and `HandleLookTips`. In `hooks_test.go`, rename the section comment and the four tests `TestHandleLookHints_*` to `TestHandleLookTips_*` and their calls to `HandleLookTips`.

- [ ] **Step 5: Broadcast test**: `internal/hooks/broadcast_tips_test.go`:

```go
package hooks

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/tips"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func TestBroadcastTips(t *testing.T) {
	on := users.NewTestUser(9301, "tipson", "Tipson", 19301)
	off := users.NewTestUser(9302, "tipsoff", "Tipsoff", 19302)
	off.SetConfigOption("tips", false)
	deaf := users.NewTestUser(9303, "tipsdeaf", "Tipsdeaf", 19303)
	deaf.Deafened = true
	defer users.SeedUsersForTest(map[int]*users.UserRecord{9301: on, 9302: off, 9303: deaf})()
	defer tips.SeedForTest([]string{"First tip.", "Second tip."})()
	for _, id := range []int{9301, 9302, 9303} {
		events.DrainQueuedMessagesForTest(id)
	}

	BroadcastTips(events.NewRound{RoundNumber: tipIntervalRounds - 1})
	if got := events.DrainQueuedMessagesForTest(9301); len(got) != 0 {
		t.Fatalf("off-interval round sent %q", got)
	}

	BroadcastTips(events.NewRound{RoundNumber: tipIntervalRounds})
	got := events.DrainQueuedMessagesForTest(9301)
	if len(got) != 1 || !strings.Contains(got[0], "[Tip]") || !strings.Contains(got[0], "First tip.") {
		t.Fatalf("player with tips on got %q, want one [Tip] First tip. line", got)
	}
	if got := events.DrainQueuedMessagesForTest(9302); len(got) != 0 {
		t.Errorf("player with tips off got %q", got)
	}
	if got := events.DrainQueuedMessagesForTest(9303); len(got) != 0 {
		t.Errorf("deafened player got %q", got)
	}

	BroadcastTips(events.NewRound{RoundNumber: 2 * tipIntervalRounds})
	if got := events.DrainQueuedMessagesForTest(9301); len(got) != 1 || !strings.Contains(got[0], "Second tip.") {
		t.Errorf("second broadcast = %q, want Second tip.", got)
	}
}
```

- [ ] **Step 6: Boot load and golden**

`main.go`, right after `gossip.Load()`:

```go
	tips.Load() // Messaging M3 item 7: the periodic tip broadcast's store
```
with the import.

`snapshot_test.go`: add `tips.Load()` to `setupRealStores` after `gossip.Load()`, import `internal/tips`, and replace the body of `buildTipsGolden` between the header and `return` with:

```go
	defer tips.SeedForTest(tips.All())()
	n := tips.Count()
	if n == 0 {
		t.Fatal("no tips loaded; setupRealStores must call tips.Load()")
	}
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "tip|%d => %s\n", i, tips.Next())
	}
```

Move the header `Fprintf` lines above that block if needed so their text and order stay exactly as recorded, delete the `hints.yaml` read and parse, update the doc comment's last sentence to say it reads through the tips store since Task 3, and remove the `gopkg.in/yaml.v2` import if nothing else in the file uses it.

- [ ] **Step 7: Verify**

Run: `go build ./... && go test ./internal/tips/ ./internal/hooks/ ./internal/narration/ .` -> all `ok`, goldens unchanged.
Run standalone: `grep -rn "hints.yaml\|BroadcastHints\|HandleLookHints\|hintMessages" --include=*.go internal modules main.go` -> only `internal/hooks/spell_resolution.go`'s comment (Task 6 fixes it). Anything else: fix it here.

- [ ] **Step 8: Commit** the new tips files, the renamed files (`git add` both old and new paths as `git mv` staged them), `hooks.go`, `hooks_test.go`, `broadcast_tips_test.go`, `main.go`, `snapshot_test.go`:

```
feat(tips): the hints broadcast becomes the tips store (tips.yaml, BroadcastTips)
```

---

### Task 4: `set tips`

**Files:**
- Modify: `internal/usercommands/set.go:40-41,98`; `_datafiles/world/dogmud/templates/help/set.template`; `_datafiles/world/default/templates/help/set.template`
- Create: `internal/usercommands/set_tips_test.go`

- [ ] **Step 1: Failing test**

```go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func TestSetTips_HintsIsAnAliasForTheSameSetting(t *testing.T) {
	u := users.NewTestUser(8801, "tipper", "Tipper", 18801)

	if _, err := Set("tips", u, nil, events.EventFlag(0)); err != nil {
		t.Fatal(err)
	}
	if got, ok := u.GetConfigOption("tips").(bool); !ok || got {
		t.Fatalf("after `set tips` from the default (on), tips = %v, want false", u.GetConfigOption("tips"))
	}

	if _, err := Set("hints", u, nil, events.EventFlag(0)); err != nil {
		t.Fatal(err)
	}
	if got, ok := u.GetConfigOption("tips").(bool); !ok || !got {
		t.Fatalf("after `set hints`, tips = %v, want true", u.GetConfigOption("tips"))
	}
	if v := u.GetConfigOption("hints"); v != nil {
		t.Errorf("`set hints` wrote a hints option (%v); the alias must write tips", v)
	}
	events.DrainQueuedMessagesForTest(8801)
}
```

Run: `go test ./internal/usercommands/ -run TestSetTips` -> FAIL (`set tips` unknown, `tips` stays nil).

- [ ] **Step 2: Implement**: in `set.go` replace

```go
	case `hints`:
		return cmdSetToggle(user, `hints`, `Hints`, true)
```
with
```go
	case `tips`, `hints`: // `hints` kept as an alias: the setting was renamed in 0.18.0
		return cmdSetToggle(user, `tips`, `Tips`, true)
```
and `displayBoolSetting(user, `hints`, `hints`)` with `displayBoolSetting(user, `tips`, `tips`)`.

- [ ] **Step 3: Help**: in BOTH `set.template` files, after the `set tinymap` entry's text line and its blank line, add:

```
  <ansi fg="command">set tips</ansi>
  This toggles the periodic gameplay tips on or off.

```

- [ ] **Step 4: Verify**: `go test ./internal/usercommands/ -run TestSetTips && go test .` -> `ok`.

- [ ] **Step 5: Probe 2**: delete `, `hints`` from the case; the test must FAIL; restore with Edit; `ok`.

- [ ] **Step 6: Commit** the four files:

```
feat(tips): `set tips` (with `set hints` as an alias) and its help entry
```

---

### Task 5: Migration 0.18.0

**Files:**
- Create: `internal/migration/0.18.0.go`, `internal/migration/0.18.0_test.go`
- Modify: `internal/migration/migration.go` (after the 0.17.0 block), `main.go:97`

- [ ] **Step 1: Failing tests**: `internal/migration/0.18.0_test.go`:

```go
package migration

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

func writeUserSave(t *testing.T, dir, name, body string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readBytes(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Deliberately NOT in yaml.v2's sorted key order, so a needless rewrite
// changes the bytes and the no-op tests can see it.
const saveWithHintsOff = "username: alice\nuserid: 1\nconfigoptions:\n  tinymap: true\n  hints: false\ncharacter:\n  name: Aliceia\ntipscomplete:\n  list: true\n"

func TestRenameTipsConfigOption_MovesTheFlag(t *testing.T) {
	dir := t.TempDir()
	path := writeUserSave(t, dir, "1.yaml", saveWithHintsOff)

	if err := renameTipsConfigOptionInDir(dir, false); err != nil {
		t.Fatal(err)
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(readBytes(t, path), &doc); err != nil {
		t.Fatal(err)
	}
	opts := doc["configoptions"].(map[interface{}]interface{})
	if v, ok := opts["tips"]; !ok || v != false {
		t.Errorf("configoptions.tips = %v (present %v), want false", v, ok)
	}
	if _, ok := opts["hints"]; ok {
		t.Error("configoptions.hints is still present")
	}
	if opts["tinymap"] != true || doc["username"] != "alice" || doc["userid"] != 1 {
		t.Errorf("other fields changed: %v", doc)
	}
	if doc["character"].(map[interface{}]interface{})["name"] != "Aliceia" {
		t.Error("character data changed")
	}
	if doc["tipscomplete"].(map[interface{}]interface{})["list"] != true {
		t.Error("tipscomplete, a different key, changed")
	}
}

func TestRenameTipsConfigOption_LeavesOtherSavesByteIdentical(t *testing.T) {
	dir := t.TempDir()
	body := "username: bob\nuserid: 2\nconfigoptions:\n  tinymap: false\n"
	path := writeUserSave(t, dir, "2.yaml", body)

	if err := renameTipsConfigOptionInDir(dir, false); err != nil {
		t.Fatal(err)
	}
	if got := readBytes(t, path); !bytes.Equal(got, []byte(body)) {
		t.Errorf("a save without the hints option was rewritten:\n%s", got)
	}
}

func TestRenameTipsConfigOption_SecondRunIsANoOp(t *testing.T) {
	dir := t.TempDir()
	path := writeUserSave(t, dir, "1.yaml", saveWithHintsOff)

	if err := renameTipsConfigOptionInDir(dir, false); err != nil {
		t.Fatal(err)
	}
	first := readBytes(t, path)
	if err := renameTipsConfigOptionInDir(dir, false); err != nil {
		t.Fatal(err)
	}
	if second := readBytes(t, path); !bytes.Equal(first, second) {
		t.Error("a second run rewrote an already-migrated save")
	}
}

func TestRenameTipsConfigOption_RefusesBothKeys(t *testing.T) {
	dir := t.TempDir()
	body := "username: carol\nconfigoptions:\n  hints: false\n  tips: true\n"
	path := writeUserSave(t, dir, "3.yaml", body)

	err := renameTipsConfigOptionInDir(dir, false)
	if err == nil || !strings.Contains(err.Error(), "both hints and tips") {
		t.Fatalf("err = %v, want a refusal naming both keys", err)
	}
	if got := readBytes(t, path); !bytes.Equal(got, []byte(body)) {
		t.Error("the ambiguous save was modified")
	}
}

func TestRenameTipsConfigOption_DryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	path := writeUserSave(t, dir, "1.yaml", saveWithHintsOff)

	if err := renameTipsConfigOptionInDir(dir, true); err != nil {
		t.Fatal(err)
	}
	if got := readBytes(t, path); !bytes.Equal(got, []byte(saveWithHintsOff)) {
		t.Error("dry run wrote the file")
	}
}

func TestRenameTipsConfigOption_MissingUsersDirIsFine(t *testing.T) {
	if err := renameTipsConfigOptionInDir(filepath.Join(t.TempDir(), "users"), false); err != nil {
		t.Errorf("a tree with no users directory returned %v", err)
	}
}
```

Run: `go test ./internal/migration/ -run TestRenameTipsConfigOption` -> build FAIL.

- [ ] **Step 2: Implement**: `internal/migration/0.18.0.go`:

```go
package migration

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// Description:
// Messaging M3 item 7 renamed the periodic gameplay broadcast from hints to
// tips, including the player's on/off setting: `set tips` now reads and writes
// configoptions.tips. A save still holding configoptions.hints would silently
// turn tips back on for a player who had turned them off, so the value moves.
//
// The setting lives on the USER record, not the character, so there is no
// alts trap. Idempotent without a marker: a save with no hints option is not
// rewritten, so a second run changes nothing. A save holding both hints and
// tips is an error, so Run restores the datafiles backup instead of guessing
// which the player meant. A save that fails to parse is logged and skipped,
// as in 0.16.0: it would not load as a player either.
func migrate_TipsConfigOption(dryRun bool) error {
	c := configs.GetConfig()
	return renameTipsConfigOptionInDir(filepath.Join(string(c.FilePaths.DataFiles), "users"), dryRun)
}

// renameTipsConfigOptionInDir is the testable core.
func renameTipsConfigOptionInDir(usersDir string, dryRun bool) error {
	mode := "APPLY"
	if dryRun {
		mode = "DRY-RUN"
	}

	matches, err := filepath.Glob(filepath.Join(usersDir, "*.yaml"))
	if err != nil {
		return err
	}

	mudlog.Info("Migration 0.18.0", "message", "Renaming the hints setting to tips", "mode", mode, "files", len(matches))

	renamed := 0
	for _, path := range matches {
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		var userMap map[string]interface{}
		if err := yaml.Unmarshal(raw, &userMap); err != nil {
			mudlog.Warn("Migration 0.18.0", "file", filepath.Base(path), "error", err)
			continue
		}

		opts, ok := userMap["configoptions"].(map[interface{}]interface{})
		if !ok {
			continue
		}
		value, hasHints := opts["hints"]
		if !hasHints {
			continue
		}
		if _, hasTips := opts["tips"]; hasTips {
			return fmt.Errorf("%s: configoptions carries both hints and tips; refusing to guess which the player meant", path)
		}

		renamed++
		if dryRun {
			continue
		}

		opts["tips"] = value
		delete(opts, "hints")

		out, err := yaml.Marshal(userMap)
		if err != nil {
			return fmt.Errorf("failed to marshal %s: %w", path, err)
		}
		if err := os.WriteFile(path, out, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}

	mudlog.Info("Migration 0.18.0", "message", "hints setting renamed", "saves", renamed, "mode", mode)
	return nil
}
```

`migration.go`, after the 0.17.0 block:

```go
	if lastConfigVersion.IsOlderThan(version.New(0, 18, 0)) {
		// Rename each player's configoptions.hints to configoptions.tips (the
		// broadcast was renamed in messaging M3 item 7).
		if err := migrate_TipsConfigOption(false); err != nil {
			return err
		}
	}
```

`main.go:97`: `const VERSION = "0.18.0"`.

- [ ] **Step 3: Verify**: `go build ./... && go test ./internal/migration/ ./internal/playtestenv/ .` -> `ok`. Run standalone `grep -rn '0\.17\.0' --include=*.go . | grep -v internal/migration` and report any hit that is a version assertion rather than a history comment.

- [ ] **Step 4: Probes 3 and 4**: (3) delete the `hasTips` refusal: `TestRenameTipsConfigOption_RefusesBothKeys` must FAIL; restore. (4) move the `!hasHints` `continue` so every parsed save is marshalled and written: `TestRenameTipsConfigOption_LeavesOtherSavesByteIdentical` must FAIL; restore. `git diff internal/migration/0.18.0.go` prints nothing relative to your Step 2 content after each restore (the file is new: re-read it).

- [ ] **Step 5: Commit** the two new files, `migration.go`, `main.go`:

```
feat(migration): 0.18.0 renames each player's hints setting to tips
```

---

### Task 6: Guards and stray references

**Files:**
- Create: `store_data_files_guard_test.go` (repo root, `package main`)
- Modify: `messaging_surface_guard_test.go:204` (`hints` reason), `internal/hooks/spell_resolution.go:1849` (comment)

- [ ] **Step 1: The guard**

```go
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Each data file has exactly one Go owner that names it. M3 item 7 moved the
// gossip templates out of hook globals and renamed the hints broadcast to tips;
// a second reader would reintroduce the silent, unvalidated load both stores
// replaced, and a stray hints.yaml would read a file that no longer exists.
var storeDataFileOwners = []struct{ file, owner string }{
	{"gossip_templates.yaml", "internal/gossip/"},
	{"tips.yaml", "internal/tips/"},
}

func TestStoreDataFilesAreNamedOnlyByTheirStore(t *testing.T) {
	inside := map[string]int{}
	var problems []string

	for _, root := range messagingSurfaceGoRoots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			rel := filepath.ToSlash(path)
			for i, line := range strings.Split(string(src), "\n") {
				if strings.Contains(line, "hints.yaml") {
					problems = append(problems, fmt.Sprintf("%s:%d names hints.yaml, which was renamed to tips.yaml: %s", rel, i+1, strings.TrimSpace(line)))
				}
				for _, o := range storeDataFileOwners {
					if !strings.Contains(line, o.file) {
						continue
					}
					if strings.HasPrefix(rel, o.owner) {
						inside[o.file]++
						continue
					}
					problems = append(problems, fmt.Sprintf("%s:%d names %s outside %s: %s", rel, i+1, o.file, o.owner, strings.TrimSpace(line)))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}

	for _, o := range storeDataFileOwners {
		if inside[o.file] == 0 {
			t.Errorf("%s is named nowhere inside %s; the guard is blind, not the tree clean", o.file, o.owner)
		}
	}
	sort.Strings(problems)
	for _, p := range problems {
		t.Error(p)
	}
}
```

Run: `go test . -run TestStoreDataFilesAreNamedOnlyByTheirStore` -> FAIL on `internal/hooks/spell_resolution.go:1849` (the `hints.yaml` comment). That red is expected and proves the `hints.yaml` branch can fire.

- [ ] **Step 2: Fix the stray comment**: in `spell_resolution.go`, change `charm.yaml, charm.template and hints.yaml all went on` to `charm.yaml, charm.template and a gameplay tip all went on`. Rerun the guard: `ok`.

- [ ] **Step 3: Registry reason**: replace the `hints` entry's reason in `messaging_surface_guard_test.go` with:

```go
	"hints":               {content, "internal/dialogue/types.go's Hints field -- narrator-perspective text describing dialogue options (see CLAUDE.md Dialogue Voice & Trigger Discoverability), read on request when a player enters a dialogue node. The periodic broadcast that once shared this spelling was renamed to tips.yaml in messaging M3 item 7, so every remaining use is dialogue."},
```

Keep the column alignment gofmt produces.

- [ ] **Step 4: Probe**: temporarily add a comment naming `gossip_templates.yaml` to `internal/hooks/MobIdle_HandleIdleMobs.go`; the guard must FAIL naming that file; remove it; `ok`.

- [ ] **Step 5: Verify and commit**: `go test .` -> `ok`. Commit the three files:

```
test(messaging): gossip and tips data files have one Go owner each
```

---

### Task 7: Documentation

**Files:** `internal/gossip/context.md` (new), `internal/tips/context.md` (new), `internal/hooks/context.md`, `internal/migration/context.md`, `internal/narration/context.md`, `docs/superpowers/specs/2026-08-31-messaging-unification-design.md`, `docs/PATCH_NOTES.md`, `docs/README.md`

Verify every symbol you name exists (`Select-String -Path internal\gossip\*.go -Pattern '^(func|type|const|var)\s'`).

- [ ] **Step 1: `internal/gossip/context.md`**: Purpose (the store, key shapes, who picks the key), API (`Load`, `Validate`, `Pool`, `Keys`, `Render`; test helpers `SeedForTest`, `RenderWithForTest`), the rules (`Load` panics on a bad file, missing file is empty; `Render` draws exactly once through `DefaultPicker`, and why that matters; tokens `{desc}` and `{description}` at most once per line), consumers (`internal/hooks` `buildGossipLine`, `renderFactGossip`), and that `gossip.golden` freezes it.

- [ ] **Step 2: `internal/tips/context.md`**: Purpose (formerly hints; why renamed), API (`Load`, `Validate`, `Next`, `Count`, `All`, `SeedForTest`), the no-length-rule decision and ledger row 22, consumer `hooks.BroadcastTips`, the player setting `configoptions.tips` via `set tips` (alias `set hints`) and migration 0.18.0.

- [ ] **Step 3: Existing context files**: `internal/hooks/context.md`: every mention of `BroadcastHints`, `HandleLookHints`, `hints.yaml` or the gossip loader updated to the new names and stores (grep the file first). `internal/migration/context.md`: add 0.18.0 in the file's existing per-version format. `internal/narration/context.md`: add `internal/gossip/gossip.go` to the Kind A callers and `gossip.golden`, `tips.golden` to the goldens list.

- [ ] **Step 4: Arc spec**: in `2026-08-31-messaging-unification-design.md`, append to M3 table row 7's "What it proves" cell: ` **Ruled 2026-09-15:** conversations are not migrated (speakers of a sequence through say, not audiences of one moment); gossip joins the core; hints becomes tips. See [the item 7 spec](2026-09-15-messaging-m3-item7-gossip-tips-design.md).`

- [ ] **Step 5: Patch note**: at the top of `docs/PATCH_NOTES.md` under the title:

```markdown
## 2026-09-15: Hints are now tips

The gameplay advice that appears every few minutes is now called a tip, so it
is no longer confused with a quest hint. Turn tips off or on with set tips.
If you had turned hints off, tips stay off for you. The old set hints still
works.
```

- [ ] **Step 6: README**: the spec and this plan are already indexed in `docs/README.md`; update either row only if a task changed what it describes.

- [ ] **Step 7: Audit and commit**: `python tools/context_md_audit.py`; no phantom symbols in `gossip`, `tips`, `hooks`, `migration`, `narration`. Commit all the files above:

```
docs(messaging): M3 item 7 context.md, arc ruling, patch note
```

---

### Task 8: Gate, playtest, PR

Load `dogmud-shipping` and `dogmud-playtesting` first.

- [ ] **Step 1:** `gofmt -l internal modules *.go` (nothing), `go vet ./...` (nothing), full `go test ./...` (every package `ok`), `go test .`.
- [ ] **Step 2: Boot check** per `dogmud-shipping` (detached worktree, `boot-check.exe`, 180s): exit 124, 0 panics, `Server Ready`, and the boot log shows `gossip.Load()` with `loadedKeys=34` and `tips.Load()` with `loadedCount=74`. Check whether `Migration 0.18.0` ran (it runs only if the copied config's `CurrentVersion` is older); report either way.
- [ ] **Step 3: Whole-branch review** over `git diff master...HEAD` against the spec, reviewer runs `go test .` and the touched packages.
- [ ] **Step 4: Playtest lane (about 25 minutes)** per `dogmud-playtesting`, scenario `tools/playtest/scenarios/m3-item7-tips-gossip.yaml` with goals under `tools/playtest/goals/scenarios/m3-item7-tips-gossip/`. One player, starting in a tavern room that holds an NPC in the `gossiper` group (find one: `grep -rln "gossiper" _datafiles/world/dogmud/mobs` and its spawn room). The player:
  1. quotes every `[Tip]` line; each must equal a `tip|<n>` row of `tips.golden`, in consecutive order;
  2. after the first tip, runs `set tips` (expects "Tips toggled OFF") and confirms the next interval brings no tip;
  3. runs `set hints` (expects "Tips toggled ON") and confirms the following interval brings a tip;
  4. runs `set` and confirms the listing shows `tips:` and not `hints:`;
  5. quotes any gossip line the NPC says; it must match a row of that key family in `gossip.golden` with a real event or fact description in place of the stand-in (fallback lines match exactly).
  Extract findings to memory; tear down the container by ID.
- [ ] **Step 5: PR**: push; `gh pr create --repo pruuk/DOGMud --base master --head feature/messaging-m3-item7-conversations-gossip-hints ...`; body lists the rulings, the evidence (goldens unchanged, probes red, suite, boot, playtest), the deploy note (0.18.0 rewrites player saves on first boot after deploy; check droplet disk for the DataFiles backup), and ends with the Claude Code attribution line. Merge with `--merge --delete-branch` on green, confirming every workflow ran. Do NOT deploy.
