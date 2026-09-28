# Slice M: Config Locks and Redaction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** In-game config writes can never change a security setting, and no
view of the config a person can read (boot log, `server set` listing,
`server config` menu, `/viewconfig`) ever prints a secret.

**Architecture:** `configs.SetVal` becomes the operator write and refuses any
key that `configs.IsLocked` names (a Go hard list plus `Server.Locked`); a new
`configs.SetEngineVal` carries the two engine-owned writes that
`Server.Locked` must not stop. A new `Config.DisplayConfigData` wraps
`AllConfigData` and redacts every `ConfigSecret` and every leaf named
`apikey`, `secret` or `password`; every human-facing reader switches to it and
a root guard test keeps it that way.

**Tech Stack:** Go 1.x, `text/template` (web pages), `gopkg.in/yaml.v2`,
`testing` plus `testify/require` where the package already uses it.

**Source spec:** `docs/superpowers/specs/2026-09-28-baubles-hardening-and-corpus-design.md`,
section "Slice M" (M1, M2, Tests), and its "Owner rulings" 11 and 13.
**Every task executes in the master worktree `C:/tmp/dogmud-config-locks`**
on branch `fix/config-locks-redaction`, branched fresh from master.

**Delivery (ruling 11).** Slice M is its own PR to master, opened and merged
per `dogmud-shipping` (every `gh` call carries `--repo pruuk/DOGMud`). PR #175
merges first; our commits never rebase or push onto FinalTwist's branch. The
owner deploys, never Claude. Every diff, gofmt, lint and context.md check
compares against `$BASE`, the master commit Task 0 records, and the gate
keeps `git diff --shortstat $BASE..HEAD` under 20k lines and 300 files (the
CI lint inversion). This plan document reaches master through the separate
docs-only PR for the spec and plans, so this plan adds no `docs/README.md`
row of its own. Slice H later branches from master after this merges and adds
`Modules.baubles.*` (including the baubles `ModerateOutput` and
`ModerationModel`) to the hard list.

---

## Facts verified against master (`8c6561c5a`)

Read with `git show master:<path>` and `git grep ... master` on 2026-09-28.
Line numbers are master's.

| Fact | Where (master) |
|---|---|
| `ErrLockedConfig = errors.New("config name is locked")` is declared and used nowhere | `internal/configs/configs.go:40` |
| `func (c *Config) DotPaths() map[string]any` (pointer receiver) walks structs by yaml tag and walks maps, so `Modules.<name>.<key>` leaves appear | `configs.go:157-163`, `buildDotPaths` `:165-220` (map branch `:200-215`) |
| `func (c Config) AllConfigData(excludeStrings ...string) map[string]any` (value receiver); exclusions are `util.StringWildcardMatch` against the WHOLE lowercased dotted path | `configs.go:347-370`, `internal/util/util.go:883-911` |
| `func SetVal(propertyPath string, newVal string) error`: `FindFullPath`, refuse on empty type, merge into `overrides`, `util.Save` to `overridePathFor(...)`, overlay, `Validate`. No lock check | `configs.go:372-411` |
| `ReloadConfig` builds `keyLookups`/`typeLookups` inline from `configData.AllConfigData()`, i.e. the config BEFORE this load (on first boot, `newUnloadedConfig()`, whose `Modules` is nil) | `configs.go:497-520` |
| `func FindFullPath(inputKey string) (properKey string, typeName string)`; unknown keys come back unchanged with type `""` | `configs.go:572-578` |
| `overridePathFor` honours `CONFIG_PATH` first | `configs.go:432-439` |
| `type ConfigSecret string`; `func (c ConfigSecret) String() string { return `*** REDACTED ***` }` | `internal/configs/config_types.go:11, 95-97` |
| Only two `ConfigSecret` fields: `Server.Seed`, `Integrations.Discord.WebhookUrl` | `config.server.go:6`, `config.integrations.go:8` |
| `Server.Locked ConfigSliceString`; only yaml key ending in `Locked` in the whole config | `config.server.go:11` |
| `FilePaths.WebDomain ConfigString` exists | `config.filepaths.go:4` |
| `type Modules map[string]any` | `config.modules.go:3` |
| `APIFramework` does NOT exist on master (`git grep APIFramework master -- internal/configs` is empty) | n/a |
| `SetConfigForTest(t *testing.T, c Config)` exported, restores via `t.Cleanup` | `internal/configs/testing_support.go:30-41` |
| `GetServerConfig() Server` | `config.server.go:105` |
| Shipped `Server.Locked`: `FilePaths`, `Server.CurrentVersion`, `Server.NextRoomId`, `Server.Seed`, `Server.OnLoginCommands`, `Server.BannedNames`; comment block above it | `_datafiles/config.yaml:94-98` (comment), `:99-105` (list) |
| `Modules.aicompanion` keys in `config.yaml`: `Enabled`, `APIKey`, `Model`, `FastModel`, `DeepModel`, budgets, `PlayerKeys`, `RelayOrigin`, ... | `_datafiles/config.yaml:2401-2470` |
| `buildConfig` also reads `APIKeyEnv`, `BaseURL`, `AllowCustomEndpoint`, `ModerationModel` (absent from `config.yaml`) | `modules/aicompanion/config.go:187-310` |
| `ModerateOutput` ships in `config.yaml` (`ModerateOutput: true`) and `buildConfig` reads it; `ModerationModel` is read only by `buildConfig` (`get("ModerationModel")`, defaulted to `omni-moderation-latest`), never in `config.yaml` | `_datafiles/config.yaml:2438`; `modules/aicompanion/config.go:71-72, 248-249, 304-305, 490-491` |
| `configDataLock sync.RWMutex`; `SetConfigForTest` takes it for the swap and the restore. `AllConfigData`/`DotPaths` take no lock, so lookups can be built before locking | `configs.go:38`; `testing_support.go:30-41`; `configs.go:157-163, 347-370` |
| `user.SendText` runs `messaging.RenderForRecipient`, whose normalize stage capitalises the first letter and appends end punctuation; `CategorySystem` is not wrapped. `server set motd hello` therefore prints `Config changed: motd=hello.` | `internal/users/userrecord.go:486`; `internal/messaging/pipeline.go:53-83`, `normalize.go:159, 186`; `admin.server.go:118` |
| `server set` reports a refusal as `config change error: %s=%s (%s)` with the error text mid-sentence | `admin.server.go:113-115` |
| No Go file outside `internal/configs` calls `DotPaths` or `GetOverrides`; no template mentions `.CONFIG.Modules`, `DotPaths` or `GetOverrides` | `git grep` on master, empty |
| Templates can reach the whole config through the `getconfig` template func | `internal/web/template_func.go:102-104` |
| `rooms.SetNextRoomId` callers besides `roommanager.go`: `rooms.go:147`, `save_and_load.go:267, 510`, and `internal/devtools/gridgen.go:55` | as listed |
| `const VERSION = "0.18.0"`; `Server.CurrentVersion` is absent from `config.yaml` values (only in `Locked`) and `Validate` defaults it to `0.9.0`, so a fresh boot runs every migration then records `0.18.0` through `CONFIG_PATH` | `main.go:101, 202-217`; `config.server.go:57-58`; `migration.go:122-151` |
| Migration 0.18.0 always logs `Migration 0.18.0` when it runs | `internal/migration/0.18.0.go:43` |
| HTTP bind logs `"stage", "Starting http server", "port", <HttpPort>`; a bind failure logs `Error starting web server` | `internal/web/web.go:535, 549` |
| aicompanion ships no data-overlay config, so nothing calls `AddOverlayOverrides` for its keys | `config.yaml:2394` comment; only caller `internal/plugins/plugins.go:575` |
| **Every `SetVal` caller:** `server set` `admin.server.go:113`; `server config` menu `admin.server.go:336`; `setmotd` `admin.setmotd.go:28` (`server.motd`); `PluginConfig.Set` `internal/plugins/pluginconfig.go:15` (no module calls it: `git grep 'Config\.Set(' master -- '*.go'` outside tests is empty); **migration** `internal/migration/migration.go:151` (`Server.CurrentVersion`); **rooms** `internal/rooms/roommanager.go:230` (`Server.NextRoomId`). Both engine callers ignore the error | as listed |
| `isEditAllowed`: lowercase, `HasSuffix(.., "locked")`, then lowercase prefix match over `Server.Locked`; called at `:323, :366, :386` | `internal/usercommands/admin.server.go:416-432` |
| **Every `AllConfigData` use outside `internal/configs`**, all DISPLAY, none a type lookup: `server set` listing `admin.server.go:55`; `server config` prompt default `:320`; confirmation `:338`; prompt default `:392`; `getConfigOptions` menu descriptions `:440`; boot log `main.go:245`; `/viewconfig` `_datafiles/html/public/viewconfig.html:11` | as listed |
| Boot log: `cfgData := c.AllConfigData()`, sort with `slices.Sort`, `mudlog.Info("Config", "name", k, "value", cfgData[k])`; `slices` used nowhere else in `main.go` | `main.go:245-256`, import `:12` |
| `/viewconfig` is `serveTemplate` on the unauthenticated `/` route; `"CONFIG": configs.GetConfig()`; pages parse with `text/template` | `internal/web/web.go:66, 159, 308`, import `"text/template"` |
| Web tests swap `httpRoot` directly and resolve repo paths with `runtime.Caller(0)` | `internal/web/companion_relay_test.go:18-29`, `admin_condition_templates_test.go:32` |
| usercommands tests capture output with `events.DrainQueuedMessagesForTest(userId int) []string`; fixtures `seedAllRegistries()`, `getTestUserAndRoom(t)` | `internal/events/events.go:338`, `usercommands_test.go:107, 355` |
| usercommands `TestMain` calls `configs.AddOverlayOverrides` for `FilePaths.DataFiles` | `usercommands_test.go:68-85` |
| Root guard pattern: `package main`, `filepath.Abs(".")`, `WalkDir`, skip dot-dirs, AST scan | `durable_write_guard_test.go:98-165` |
| `getConfigOptions(input string) ([]templates.NameDescription, bool)` | `admin.server.go:434` |
| `version.Version.String()` is `%d.%d.%d` | `internal/version/version.go:21-23` |
| `migration` package already imports `mudlog` elsewhere (`0.10.0.go`); `rooms/roommanager.go` imports `mudlog` | as listed |

### Spec statements found false or incomplete against master

1. **`SetVal` cannot simply enforce `Server.Locked`.** Two engine writers go
   through `SetVal` to keys in the SHIPPED lock list: `migration.go:151`
   (`Server.CurrentVersion`) and `roommanager.go:230` (`Server.NextRoomId`).
   Both ignore the error, so enforcement would silently stop recording the
   migrated version (every migration re-runs every boot, the #171 bug class)
   and stop persisting room ids. This plan adds `SetEngineVal` (skips
   `Server.Locked`, still refuses the hard list) and moves both callers to it.
2. **The spec's `AllConfigData` caller list misses `admin.server.go:440`**
   (`getConfigOptions`), which prints every leaf value in the `server config`
   menu.
3. **On master the aicompanion keys are not writable through `server set`
   today**: `ReloadConfig` builds the lookups from the pre-load config
   (`configs.go:500`), and aicompanion registers no overlay, so
   `Modules.aicompanion.*` resolves to type `""` and `SetVal` refuses it as an
   invalid property. The hard lock is defence in depth there (and lives the
   moment the lookups include module keys). The LIVE master leak is M2:
   `Modules.aicompanion.APIKey` is a plain `string` in the untyped `Modules`
   map, so the boot log, `server set` listing, `server config` menu (and its
   prompt default) and `/viewconfig` all print it raw. This plan does NOT fix
   the lookup build; it is out of Slice M's scope and fixing it would newly
   make module keys writable.
4. **Master line numbers differ** from the spec's PR-branch table:
   `configs.go` "Nothing to do with Locked" is `:262` (spec `:267`), `SetVal`
   `:372` (spec `:377`), lookup build `:497-520` (spec `:505-521`),
   `FindFullPath` `:572` (spec `:577`), dump walk `:165-220` (spec
   `:169-223`), boot log `main.go:245` (spec `:247`).
5. **"A sentinel key set in both key locations"**: master has one module key
   location (`Modules.aicompanion.APIKey`); `APIFramework` does not exist. The
   tests use `Modules.aicompanion.APIKey` plus the `ConfigSecret`-typed
   `Integrations.Discord.WebhookUrl` as the second location.
6. **`/viewconfig`'s existing exclusions mostly do not fire**: they match the
   whole dotted path, so `"seed*"` never matches `server.seed` and
   `"bannednames"` never matches a nested key. Prefix patterns `"modules*"`
   and `"apiframework*"` DO work, so the added exclusions are real.

### Decisions this plan takes

- The `APIFramework.*` entries ARE in the master hard list: it is a plain
  `[]string` compared by exact lowercase path, an absent key matches nothing
  and costs nothing, and slice H inherits them.
- The lock check runs BEFORE the unknown-key check, so a hard-locked key is
  refused as locked even where the lookups cannot resolve it (point 3).
- `SetVal` also refuses the redaction marker as a value
  (`ErrRedactedValue`): the `server config` prompt offers the displayed
  value as its default, so after M2 pressing Enter on any redacted leaf would
  otherwise write `*** REDACTED ***` over the real secret.
- `isEditAllowed`'s `HasSuffix("locked")` rule is kept inside `IsLocked` for
  parity (only `Server.Locked` ends in `Locked` today).
- Ruling 13: `Modules.aicompanion.ModerateOutput` and
  `Modules.aicompanion.ModerationModel` join the hard list. `ModerationModel`
  is read only by `buildConfig` and never resolves through the lookups; the
  lock-first order refuses it as locked all the same.

---

## File map

All paths are in the worktree `C:\tmp\dogmud-config-locks`.

| File | Change | Responsibility |
|---|---|---|
| `internal/configs/configs.go` | Modify | `buildKeyLookups` extracted from `ReloadConfig`; `SetVal` / `SetEngineVal` over one `setVal` |
| `internal/configs/config_locks.go` | Create | `hardLocked`, `isHardLocked`, `IsLocked` |
| `internal/configs/config_display.go` | Create | `RedactedValue`, `ErrRedactedValue`, `DisplayConfigData`, `isSecretConfigValue` |
| `internal/configs/config_types.go` | Modify | `ConfigSecret.String` returns `RedactedValue` |
| `internal/configs/testing_support.go` | Modify | `SetConfigWithLookupsForTest` |
| `internal/configs/config_locks_test.go` | Create | lock, engine-write and helper tests |
| `internal/configs/config_display_test.go` | Create | redaction tests |
| `internal/migration/migration.go` | Modify | `recordMigratedVersion` via `SetEngineVal` |
| `internal/migration/record_version_test.go` | Create | engine write survives `Server.Locked` |
| `internal/rooms/roommanager.go` | Modify | `SetNextRoomId` via `SetEngineVal` |
| `internal/rooms/next_room_id_test.go` | Create | engine write survives `Server.Locked` |
| `internal/usercommands/admin.server.go` | Modify | `isEditAllowed` delegates; five reads to `DisplayConfigData` |
| `internal/usercommands/admin.server_locks_test.go` | Create | `server set` refusals, listing and menu redaction |
| `boot_config_log.go` | Create | `logBootConfig`, testable boot log |
| `boot_config_log_test.go` | Create | boot log redaction |
| `main.go` | Modify | call `logBootConfig`, drop `slices` import |
| `_datafiles/html/public/viewconfig.html` | Modify | `DisplayConfigData`, exclude `modules*`, `apiframework*` |
| `internal/web/viewconfig_redaction_test.go` | Create | rendered `/viewconfig` carries no secret |
| `config_display_guard_test.go` | Create | root guard: no human-facing reader calls `AllConfigData`, `DotPaths`, `GetOverrides` or reaches `Modules` from a template |
| `internal/configs/context.md` | Modify | document locks, engine writes, display view |
| `_datafiles/config.yaml` | Modify | `Locked` comment |
| `docs/PATCH_NOTES.md` | Modify | dated entry (Task 12) |

No file here is a new document, so `docs/README.md` gains no row; the plan
itself is indexed by the docs-only PR.

---

### Task 0: Worktree and baseline

**Files:** none changed.

- [ ] **Step 1: Create the worktree off master (run against the main checkout)**

```bash
git -C "C:/Users/Calabe Davis/workspace/DOGMud" worktree add -b fix/config-locks-redaction C:/tmp/dogmud-config-locks master
```

Expected: `Preparing worktree (new branch 'fix/config-locks-redaction')` and
`HEAD is now at 8c6561c5a` (or master's current tip). Do NOT touch the main
checkout's branch or files.

- [ ] **Step 1b: Record `BASE`, the commit every later check diffs from**

Shell state does not survive between steps, so the value goes to a file
outside the worktree (inside it, it would dirty `git status`):

```bash
git -C C:/tmp/dogmud-config-locks rev-parse HEAD > C:/tmp/dogmud-config-locks.base && cat C:/tmp/dogmud-config-locks.base
```

Expected: one full SHA, master's tip at branch time. Every later step that
names `$BASE` starts with `BASE=$(cat C:/tmp/dogmud-config-locks.base)`. Do
not substitute `master` or `origin/master`: master moves while this work is
in flight (PR #175 and others merge), and a diff against a moved master
counts their files as ours.

- [ ] **Step 2: Confirm config.yaml in the new worktree is the HEAD blob**

```bash
cd C:/tmp/dogmud-config-locks && git ls-files -v _datafiles/config.yaml && git diff --quiet HEAD -- _datafiles/config.yaml; echo "diff-exit=$?"
```

Expected: `H _datafiles/config.yaml` (a fresh worktree index carries no
skip-worktree bit) and `diff-exit=0`. If it shows `S`, stop and report: every
later edit to that file must then be built from `git show HEAD:_datafiles/config.yaml`.

- [ ] **Step 3: Baseline build and the packages this plan touches**

```bash
cd C:/tmp/dogmud-config-locks && go build ./... && go test ./internal/configs/ ./internal/usercommands/ ./internal/web/ ./internal/rooms/ ./internal/migration/ ./internal/plugins/ . 2>&1 | tail -15
```

Expected: build succeeds; every package `ok`. Record any pre-existing failure
verbatim before continuing; do not attribute it to this work later.

---

### Task 1: Extract the key lookups and add a test helper that installs them

A test binary never runs `ReloadConfig`, so `FindFullPath` has no lookups and
`SetVal` refuses everything as an unknown key. Any lock test built on that
would pass for the wrong reason. This task gives tests real lookups.

**Files:**
- Modify: `internal/configs/configs.go:497-520` (inline lookup build in `ReloadConfig`), add `buildKeyLookups` after `FindFullPath` (`:572-578`)
- Modify: `internal/configs/testing_support.go` (append)
- Test: `internal/configs/config_locks_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `C:\tmp\dogmud-config-locks\internal\configs\config_locks_test.go`:

```go
package configs

import (
	"testing"
)

// TestSetConfigWithLookupsForTestResolvesKeys proves the helper gives
// FindFullPath real lookups (a test binary never runs ReloadConfig) and puts
// the previous lookups back when the test ends.
func TestSetConfigWithLookupsForTestResolvesKeys(t *testing.T) {
	beforePath, beforeType := FindFullPath(`apikey`)

	t.Run(`installed`, func(t *testing.T) {
		c := GetConfig()
		c.Modules = Modules{`aicompanion`: map[string]any{`APIKey`: ``}}
		SetConfigWithLookupsForTest(t, c)

		if p, typ := FindFullPath(`seed`); p != `Server.Seed` || typ != `configs.ConfigSecret` {
			t.Errorf(`FindFullPath("seed") = (%q, %q), want ("Server.Seed", "configs.ConfigSecret")`, p, typ)
		}
		if p, typ := FindFullPath(`apikey`); p != `Modules.aicompanion.APIKey` || typ != `string` {
			t.Errorf(`FindFullPath("apikey") = (%q, %q), want ("Modules.aicompanion.APIKey", "string")`, p, typ)
		}
	})

	if p, typ := FindFullPath(`apikey`); p != beforePath || typ != beforeType {
		t.Errorf(`lookups not restored: FindFullPath("apikey") = (%q, %q), was (%q, %q)`, p, typ, beforePath, beforeType)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/configs/ -run TestSetConfigWithLookupsForTestResolvesKeys -count=1
```

Expected: FAIL to compile, `undefined: SetConfigWithLookupsForTest`.

- [ ] **Step 3: Extract `buildKeyLookups` in configs.go**

In `C:\tmp\dogmud-config-locks\internal\configs\configs.go`, replace the
inline block in `ReloadConfig` (master `:497-520`):

```go
	// Build a special lookup to attempt to match old data or even some minor typos
	keyLookups = map[string]string{}
	typeLookups = map[string]string{}
	for k, v := range configData.AllConfigData() {

		if strings.Index(k, `.`) != -1 {

			parts := strings.Split(k, `.`)

			for i := len(parts) - 1; i >= 0; i-- {
				tmpKey := strings.Join(parts[i:], `.`)
				keyLookups[strings.ToLower(tmpKey)] = k

				tmpKey = strings.Join(parts[i:], ``)
				keyLookups[strings.ToLower(tmpKey)] = k

			}

		} else {
			keyLookups[strings.ToLower(k)] = k
		}

		typeLookups[k] = reflect.TypeOf(v).String()
	}
```

with:

```go
	// Build a special lookup to attempt to match old data or even some minor typos
	keyLookups, typeLookups = buildKeyLookups(configData)
```

(This keeps reading `configData`, the pre-load config, exactly as before. Do
not change it to `tmpConfigData`: see "Spec statements" point 3.)

Then add, directly after `FindFullPath`:

```go
// buildKeyLookups builds the tables FindFullPath reads from c's dot paths.
// Every key is reachable by its full path and by each dotted and undotted
// suffix, lowercased, which is how `server set seed 1` finds Server.Seed. A
// suffix shared by two keys resolves to whichever was walked last.
func buildKeyLookups(c Config) (keys map[string]string, types map[string]string) {
	keys = map[string]string{}
	types = map[string]string{}
	for k, v := range c.AllConfigData() {
		if strings.Contains(k, `.`) {
			parts := strings.Split(k, `.`)
			for i := len(parts) - 1; i >= 0; i-- {
				keys[strings.ToLower(strings.Join(parts[i:], `.`))] = k
				keys[strings.ToLower(strings.Join(parts[i:], ``))] = k
			}
		} else {
			keys[strings.ToLower(k)] = k
		}
		types[k] = reflect.TypeOf(v).String()
	}
	return keys, types
}
```

`reflect` stays imported (`AddOverlayOverrides` and `setEnvAssignments` use it).

- [ ] **Step 4: Add the helper to testing_support.go**

In `C:\tmp\dogmud-config-locks\internal\configs\testing_support.go`, change
the import line `import "testing"` to:

```go
import (
	"path/filepath"
	"testing"
)
```

and append:

```go
// SetConfigWithLookupsForTest installs c as SetConfigForTest does and also
// builds the key and type lookups FindFullPath reads from c, so SetVal
// resolves keys in a test binary that never ran ReloadConfig. It snapshots and
// restores the lookups, the overrides union and the module overlay ledger,
// all of which SetVal mutates, and points CONFIG_PATH at a scratch file so a
// SetVal that succeeds never writes a real config-overrides.yaml. It returns
// that scratch path. Not for parallel tests (it uses t.Setenv). The swap and
// the restore both hold configDataLock, as SetConfigForTest does.
func SetConfigWithLookupsForTest(t *testing.T, c Config) string {
	t.Helper()
	overridePath := filepath.Join(t.TempDir(), `config-overrides.yaml`)
	t.Setenv(`CONFIG_PATH`, overridePath)
	SetConfigForTest(t, c)

	// Built before locking: AllConfigData takes no lock of its own.
	newKeys, newTypes := buildKeyLookups(c)

	configDataLock.Lock()
	prevKeys, prevTypes := keyLookups, typeLookups
	prevOverrides, prevOwned := overrides, moduleOverlayKeys
	keyLookups, typeLookups = newKeys, newTypes
	overrides = map[string]any{}
	moduleOverlayKeys = map[string]struct{}{}
	configDataLock.Unlock()

	t.Cleanup(func() {
		configDataLock.Lock()
		defer configDataLock.Unlock()
		keyLookups, typeLookups = prevKeys, prevTypes
		overrides, moduleOverlayKeys = prevOverrides, prevOwned
	})
	return overridePath
}
```

- [ ] **Step 5: Run the test to verify it passes, plus the package**

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/configs/ -run TestSetConfigWithLookupsForTestResolvesKeys -count=1 -v && go test ./internal/configs/ -count=1
```

Expected: PASS, then `ok`.

- [ ] **Step 6: Commit**

```bash
cd C:/tmp/dogmud-config-locks && git add internal/configs/configs.go internal/configs/testing_support.go internal/configs/config_locks_test.go && git commit -m "$(cat <<'EOF'
refactor(configs): extract buildKeyLookups, add SetConfigWithLookupsForTest

A test binary never runs ReloadConfig, so FindFullPath had no lookups and a
lock test would pass because every key was unknown, not because it was
locked. The helper installs real lookups and a scratch CONFIG_PATH.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: `SetVal` enforces locks; `SetEngineVal` for engine-owned keys

**Files:**
- Create: `internal/configs/config_locks.go`
- Modify: `internal/configs/configs.go:372-411` (`SetVal`)
- Test: `internal/configs/config_locks_test.go` (append)

- [ ] **Step 1: Write the failing tests**

Append to `C:\tmp\dogmud-config-locks\internal\configs\config_locks_test.go`
(and extend its import block to `errors`, `os`, `strings`, `testing`):

```go
// lockTestConfig is the shipped Server.Locked list plus an aicompanion block,
// so the module keys resolve through FindFullPath. Modules is a fresh map:
// never mutate a map GetConfig handed back, another test may share it.
func lockTestConfig() Config {
	c := GetConfig()
	c.Server.Locked = ConfigSliceString{`FilePaths`, `Server.CurrentVersion`, `Server.NextRoomId`, `Server.Seed`, `Server.OnLoginCommands`, `Server.BannedNames`}
	c.Modules = Modules{`aicompanion`: map[string]any{`APIKey`: ``, `Model`: ``, `RelayOrigin`: ``, `ModerateOutput`: true, `DailyTokenBudget`: 2000000}}
	return c
}

func TestSetValRefusesLockedKeys(t *testing.T) {
	overridePath := SetConfigWithLookupsForTest(t, lockTestConfig())

	// Each case carries its reason as a field, not a trailing comment, so the
	// table stays gofmt-stable when a row is added.
	cases := []struct{ key, resolved, why string }{
		{`Server.Seed`, `Server.Seed`, `Server.Locked, exact`},
		{`seed`, `Server.Seed`, `bare suffix key: the RESOLVED path is checked`},
		{`FilePaths.DataFiles`, `FilePaths.DataFiles`, `Server.Locked prefix`},
		{`Server.Locked`, `Server.Locked`, `hard list`},
		{`locked`, `Server.Locked`, `hard list through a suffix key`},
		{`FilePaths.WebDomain`, `FilePaths.WebDomain`, `hard list`},
		{`Modules.aicompanion.APIKey`, `Modules.aicompanion.APIKey`, `hard list, module key`},
		{`apikey`, `Modules.aicompanion.APIKey`, `hard list through a suffix key`},
		{`Modules.aicompanion.RelayOrigin`, `Modules.aicompanion.RelayOrigin`, `hard list`},
		{`Modules.aicompanion.Model`, `Modules.aicompanion.Model`, `hard list`},
		{`Modules.aicompanion.ModerateOutput`, `Modules.aicompanion.ModerateOutput`, `hard list (ruling 13), resolves through the lookups`},
		{`moderateoutput`, `Modules.aicompanion.ModerateOutput`, `hard list (ruling 13) through a suffix key`},
		{`Modules.aicompanion.ModerationModel`, `Modules.aicompanion.ModerationModel`, `hard list (ruling 13), not in the lookups: refused as LOCKED, not as unknown`},
		{`Modules.aicompanion.BaseURL`, `Modules.aicompanion.BaseURL`, `not in the lookups: refused as LOCKED, not as unknown`},
		{`APIFramework.APIKey`, `APIFramework.APIKey`, `section absent on master: the entry costs nothing and still binds`},
	}
	for _, tc := range cases {
		err := SetVal(tc.key, `x`)
		if !errors.Is(err, ErrLockedConfig) {
			t.Errorf(`SetVal(%q) = %v, want ErrLockedConfig (%s)`, tc.key, err, tc.why)
			continue
		}
		if !strings.Contains(err.Error(), tc.resolved) {
			t.Errorf(`SetVal(%q) error %q does not name the resolved path %q (%s)`, tc.key, err, tc.resolved, tc.why)
		}
	}

	if got := string(GetServerConfig().Seed); got == `x` {
		t.Errorf(`Server.Seed changed to %q through a refused SetVal`, got)
	}
	if _, err := os.Stat(overridePath); !os.IsNotExist(err) {
		t.Errorf(`a refused SetVal wrote %s (stat err %v)`, overridePath, err)
	}
}

// TestSetValStillWritesAnUnlockedKey is the positive control: without it the
// refusals above could come from a SetVal that refuses everything.
func TestSetValStillWritesAnUnlockedKey(t *testing.T) {
	overridePath := SetConfigWithLookupsForTest(t, lockTestConfig())

	if err := SetVal(`motd`, `hello from the lock test`); err != nil {
		t.Fatalf(`SetVal("motd") = %v, want nil`, err)
	}
	if got := string(GetServerConfig().Motd); got != `hello from the lock test` {
		t.Errorf(`Server.Motd = %q after SetVal`, got)
	}
	written, err := os.ReadFile(overridePath)
	if err != nil {
		t.Fatalf(`read %s: %v`, overridePath, err)
	}
	if !strings.Contains(string(written), `hello from the lock test`) {
		t.Errorf(`override file does not carry the new value:\n%s`, written)
	}
}

func TestSetEngineValSkipsServerLockedButNotTheHardList(t *testing.T) {
	SetConfigWithLookupsForTest(t, lockTestConfig())

	if err := SetVal(`Server.NextRoomId`, `4321`); !errors.Is(err, ErrLockedConfig) {
		t.Fatalf(`operator SetVal("Server.NextRoomId") = %v, want ErrLockedConfig`, err)
	}
	if err := SetEngineVal(`Server.NextRoomId`, `4321`); err != nil {
		t.Fatalf(`SetEngineVal("Server.NextRoomId") = %v, want nil`, err)
	}
	if got := int(GetServerConfig().NextRoomId); got != 4321 {
		t.Errorf(`Server.NextRoomId = %d after SetEngineVal, want 4321`, got)
	}
	for _, key := range []string{`Server.Locked`, `Modules.aicompanion.APIKey`, `Modules.aicompanion.ModerateOutput`, `FilePaths.WebDomain`} {
		if err := SetEngineVal(key, `x`); !errors.Is(err, ErrLockedConfig) {
			t.Errorf(`SetEngineVal(%q) = %v, want ErrLockedConfig`, key, err)
		}
	}
}

func TestIsLocked(t *testing.T) {
	SetConfigWithLookupsForTest(t, lockTestConfig())

	for _, p := range []string{`filepaths`, `FilePaths.DataFiles`, `server.seed`, `Server.Locked`,
		`modules.aicompanion.apikey`, `MODULES.AICOMPANION.PLAYERKEYS`, `APIFramework.BaseURL`,
		`modules.aicompanion.moderateoutput`, `Modules.aicompanion.ModerationModel`} {
		if !IsLocked(p) {
			t.Errorf(`IsLocked(%q) = false, want true`, p)
		}
	}
	// A partial path must stay open: the server config menu browses through
	// modules.aicompanion to reach its unlocked budget knobs.
	for _, p := range []string{`Server.Motd`, `Modules.aicompanion.DailyTokenBudget`, `modules.aicompanion`, `Network.HttpPort`} {
		if IsLocked(p) {
			t.Errorf(`IsLocked(%q) = true, want false`, p)
		}
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/configs/ -run 'TestSetValRefusesLockedKeys|TestSetValStillWritesAnUnlockedKey|TestSetEngineVal|TestIsLocked' -count=1
```

Expected: FAIL to compile, `undefined: SetEngineVal` and `undefined: IsLocked`.

- [ ] **Step 3: Create config_locks.go**

Create `C:\tmp\dogmud-config-locks\internal\configs\config_locks.go`:

```go
package configs

import "strings"

// hardLocked names the config keys no in-game command may change, whatever
// Server.Locked says: credentials, where they are sent, which models they pay
// for, whether and how generated text is moderated, the game's own web
// domain, and the lock list itself. Exact paths, compared lowercase against
// the path FindFullPath resolved.
//
// The APIFramework entries name a section master does not have yet (the
// baubles PR adds it). They are plain strings: an absent key matches nothing
// and costs nothing, and they bind the day the section lands.
var hardLocked = []string{
	`APIFramework.APIKey`,
	`APIFramework.APIKeyEnv`,
	`APIFramework.BaseURL`,
	`APIFramework.AllowCustomEndpoint`,
	`Modules.aicompanion.APIKey`,
	`Modules.aicompanion.APIKeyEnv`,
	`Modules.aicompanion.BaseURL`,
	`Modules.aicompanion.AllowCustomEndpoint`,
	`Modules.aicompanion.RelayOrigin`,
	`Modules.aicompanion.PlayerKeys`,
	`Modules.aicompanion.Model`,
	`Modules.aicompanion.FastModel`,
	`Modules.aicompanion.DeepModel`,
	`Modules.aicompanion.ModerateOutput`,
	`Modules.aicompanion.ModerationModel`,
	`FilePaths.WebDomain`,
	`Server.Locked`,
}

// isHardLocked reports whether configPath is on the hard list.
func isHardLocked(configPath string) bool {
	lower := strings.ToLower(configPath)
	for _, h := range hardLocked {
		if lower == strings.ToLower(h) {
			return true
		}
	}
	return false
}

// IsLocked reports whether an operator may not change configPath in game:
// it is on the hard list, it ends in "locked", or it starts with (lowercase)
// an entry of Server.Locked, so "FilePaths" locks every FilePaths key. The
// server config menu also passes partial paths while browsing; a partial path
// is locked only by a Server.Locked prefix, never by the exact hard list.
func IsLocked(configPath string) bool {
	if isHardLocked(configPath) {
		return true
	}
	lower := strings.ToLower(configPath)
	if strings.HasSuffix(lower, `locked`) {
		return true
	}
	for _, v := range GetServerConfig().Locked {
		if strings.HasPrefix(lower, strings.ToLower(v)) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Split SetVal in configs.go**

In `C:\tmp\dogmud-config-locks\internal\configs\configs.go`, replace the head
of `SetVal` (master `:372-377`):

```go
func SetVal(propertyPath string, newVal string) error {

	propertyPath, propertyType := FindFullPath(propertyPath)
	if propertyType == `` {
		return errors.New(`invalid property name: ` + propertyPath)
	}
```

with:

```go
// SetVal is the operator's write: `server set`, the `server config` menu,
// `setmotd` and module config setters (plugins.PluginConfig.Set). It refuses
// any key IsLocked names with ErrLockedConfig, checked on the RESOLVED path so
// a bare suffix key ("seed") cannot slip past a full-path lock.
func SetVal(propertyPath string, newVal string) error {
	return setVal(propertyPath, newVal, true)
}

// SetEngineVal is the engine's own write, for values the server maintains
// itself that Server.Locked keeps from operators: Server.CurrentVersion
// (internal/migration) and Server.NextRoomId (internal/rooms). It skips the
// Server.Locked list but still refuses the hard list, because no engine write
// needs a security key.
func SetEngineVal(propertyPath string, newVal string) error {
	return setVal(propertyPath, newVal, false)
}

func setVal(propertyPath string, newVal string, operator bool) error {

	propertyPath, propertyType := FindFullPath(propertyPath)

	// Locks first, so a hard-locked key is refused as locked even where the
	// lookups cannot resolve it.
	if isHardLocked(propertyPath) || (operator && IsLocked(propertyPath)) {
		return fmt.Errorf(`%w: %s`, ErrLockedConfig, propertyPath)
	}

	if propertyType == `` {
		return errors.New(`invalid property name: ` + propertyPath)
	}
```

The rest of the old `SetVal` body (from `quickMap := make(map[string]any)`
to the closing `return nil`) stays unchanged as the rest of `setVal`. `fmt`
is already imported.

- [ ] **Step 5: Run the tests to verify they pass, plus the package**

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/configs/ -run 'TestSetValRefusesLockedKeys|TestSetValStillWritesAnUnlockedKey|TestSetEngineVal|TestIsLocked' -count=1 -v && go test ./internal/configs/ -count=1
```

Expected: all PASS; package `ok`.

- [ ] **Step 6: Prove the refusal test can fail**

Temporarily change the lock line in `setVal` to `if false {` and run:

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/configs/ -run 'TestSetValRefusesLockedKeys' -count=1
```

Expected: FAIL naming `SetVal("Server.Seed") = <nil>, want ErrLockedConfig`
(and for `Modules.aicompanion.BaseURL` and
`Modules.aicompanion.ModerationModel`, `invalid property name`, proving the
lock check order matters). Restore the line, rerun, confirm PASS, and confirm
`git diff internal/configs/configs.go` shows only the intended change.

- [ ] **Step 7: Commit**

```bash
cd C:/tmp/dogmud-config-locks && git add internal/configs/config_locks.go internal/configs/configs.go internal/configs/config_locks_test.go && git commit -m "$(cat <<'EOF'
fix(configs): SetVal enforces Server.Locked and a hard Go lock list

server set called SetVal directly and SetVal never checked Locked; only the
server config menu did. SetVal now refuses, on the resolved path, any key on
the hard list (API keys, endpoints, companion models, moderation switch and
model, relay origin, FilePaths.WebDomain, Server.Locked) or under a
Server.Locked entry.

SetEngineVal carries the engine's own writes to Server.CurrentVersion and
Server.NextRoomId, which the shipped Locked list names; it still refuses the
hard list.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Move the two engine writers to `SetEngineVal`

Without this, Task 2 silently stops recording the migrated version (every
migration re-runs every boot) and stops persisting `NextRoomId`.

**Files:**
- Modify: `internal/migration/migration.go:148-151`
- Modify: `internal/rooms/roommanager.go:229-231`
- Test: `internal/migration/record_version_test.go` (create), `internal/rooms/next_room_id_test.go` (create)

- [ ] **Step 1: Write the failing tests**

Create `C:\tmp\dogmud-config-locks\internal\migration\record_version_test.go`:

```go
package migration

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/version"
)

// TestRecordMigratedVersionWritesThroughServerLocked: the shipped config
// locks Server.CurrentVersion against operators. The migration's own write
// must still land, or every migration re-runs on every boot.
func TestRecordMigratedVersionWritesThroughServerLocked(t *testing.T) {
	c := configs.GetConfig()
	c.Server.Locked = configs.ConfigSliceString{`Server.CurrentVersion`}
	c.Server.CurrentVersion = `0.1.0`
	configs.SetConfigWithLookupsForTest(t, c)

	recordMigratedVersion(version.New(0, 99, 0))

	if got := string(configs.GetServerConfig().CurrentVersion); got != `0.99.0` {
		t.Fatalf(`Server.CurrentVersion = %q after recordMigratedVersion(0.99.0), want "0.99.0"`, got)
	}
}
```

Create `C:\tmp\dogmud-config-locks\internal\rooms\next_room_id_test.go`:

```go
package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// TestSetNextRoomIdWritesThroughServerLocked: the shipped config locks
// Server.NextRoomId against operators; room creation's own write must land.
func TestSetNextRoomIdWritesThroughServerLocked(t *testing.T) {
	c := configs.GetConfig()
	c.Server.Locked = configs.ConfigSliceString{`Server.NextRoomId`}
	c.Server.NextRoomId = 1002
	configs.SetConfigWithLookupsForTest(t, c)

	SetNextRoomId(5150)

	if got := GetNextRoomId(); got != 5150 {
		t.Fatalf(`GetNextRoomId() = %d after SetNextRoomId(5150), want 5150: the engine write was refused as an operator write`, got)
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/migration/ -run TestRecordMigratedVersion -count=1; go test ./internal/rooms/ -run TestSetNextRoomIdWritesThroughServerLocked -count=1
```

Expected: migration FAILS to compile (`undefined: recordMigratedVersion`);
rooms FAILS with `GetNextRoomId() = 1002 after SetNextRoomId(5150)`.

- [ ] **Step 3: Implement**

In `C:\tmp\dogmud-config-locks\internal\migration\migration.go`, replace:

```go
	//
	// Finally, since successful, update to the version this migration is for
	//
	configs.SetVal(`Server.CurrentVersion`, serverVersion.String())
```

with:

```go
	//
	// Finally, since successful, update to the version this migration is for
	//
	recordMigratedVersion(serverVersion)
```

and add at the end of the file:

```go
// recordMigratedVersion stores the version the data files now match. It goes
// through configs.SetEngineVal because the shipped Server.Locked names
// Server.CurrentVersion, and SetVal honours that list; a refused write would
// leave the old version on disk and re-run every migration on the next boot.
// A failed write is logged, not returned, so a read-only config directory
// still boots as it did before.
func recordMigratedVersion(v version.Version) {
	if err := configs.SetEngineVal(`Server.CurrentVersion`, v.String()); err != nil {
		mudlog.Error(`migration`, `action`, `record version`, `version`, v.String(), `error`, err)
	}
}
```

and add `"github.com/GoMudEngine/GoMud/internal/mudlog"` to its import block
(between `factions` and `version`).

In `C:\tmp\dogmud-config-locks\internal\rooms\roommanager.go`, replace:

```go
func SetNextRoomId(nextRoomId int) {
	configs.SetVal(`Server.NextRoomId`, strconv.Itoa(nextRoomId))
}
```

with:

```go
// SetNextRoomId persists the next room id. It uses configs.SetEngineVal
// because the shipped Server.Locked names Server.NextRoomId, and SetVal
// honours that list.
func SetNextRoomId(nextRoomId int) {
	if err := configs.SetEngineVal(`Server.NextRoomId`, strconv.Itoa(nextRoomId)); err != nil {
		mudlog.Error(`SetNextRoomId`, `nextRoomId`, nextRoomId, `error`, err)
	}
}
```

(`mudlog` is already imported there.)

- [ ] **Step 4: Run to verify they pass, plus both packages**

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/migration/ ./internal/rooms/ -run 'TestRecordMigratedVersion|TestSetNextRoomIdWritesThroughServerLocked' -count=1 -v && go test ./internal/migration/ ./internal/rooms/ -count=1
```

Expected: PASS; both packages `ok`. (If `internal/rooms` shows
`TestDeleteZone_RemovesEveryTree` / `TestRenameZone_MovesRewritesAndRekeys`
failing, check you did not set `DOGMUD_BOOT_SMOKE=1`; they fail on Windows on
master too, see `dogmud-writing-tests`.)

- [ ] **Step 5: Prove the rooms test can fail**

Temporarily change `configs.SetEngineVal` back to `configs.SetVal` in
`SetNextRoomId`, run the rooms test, expect FAIL `GetNextRoomId() = 1002`,
restore, rerun PASS.

- [ ] **Step 6: Commit**

```bash
cd C:/tmp/dogmud-config-locks && git add internal/migration/migration.go internal/migration/record_version_test.go internal/rooms/roommanager.go internal/rooms/next_room_id_test.go && git commit -m "$(cat <<'EOF'
fix(configs): engine writes to locked keys go through SetEngineVal

Migration records Server.CurrentVersion and room creation records
Server.NextRoomId, both in the shipped Server.Locked list. Now that SetVal
honours that list they use SetEngineVal, and log a failed write instead of
dropping it.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: `server set` refusals and `isEditAllowed` delegation

**Files:**
- Modify: `internal/usercommands/admin.server.go:416-432` (`isEditAllowed`)
- Test: `internal/usercommands/admin.server_locks_test.go` (create)

- [ ] **Step 1: Write the tests**

Create `C:\tmp\dogmud-config-locks\internal\usercommands\admin.server_locks_test.go`:

```go
package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/require"
)

const serverLockSentinel = `sk-server-listing-sentinel-2d8e`

var shippedLockList = configs.ConfigSliceString{`FilePaths`, `Server.CurrentVersion`, `Server.NextRoomId`, `Server.Seed`, `Server.OnLoginCommands`, `Server.BannedNames`}

// serverLockConfig installs the shipped Server.Locked list and a sentinel
// secret in both key locations master has (a module map leaf and a
// ConfigSecret field), with lookups, so `server set` resolves keys the way a
// live server does. Modules is a fresh map, never one GetConfig returned.
func serverLockConfig(t *testing.T) {
	t.Helper()
	c := configs.GetConfig()
	c.Server.Locked = shippedLockList
	c.Server.Seed = `ShippedSeed`
	c.Integrations.Discord.WebhookUrl = configs.ConfigSecret(serverLockSentinel)
	c.Modules = configs.Modules{`aicompanion`: map[string]any{`APIKey`: serverLockSentinel, `Model`: `gpt-test`}}
	configs.SetConfigWithLookupsForTest(t, c)
}

func serverOutput(userId int) string {
	return strings.Join(events.DrainQueuedMessagesForTest(userId), "\n")
}

func TestServerSetRefusesLockedKeys(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	serverLockConfig(t)
	user, room := getTestUserAndRoom(t)
	events.DrainQueuedMessagesForTest(user.UserId)

	for _, cmd := range []string{`set Server.Seed 1`, `set seed 1`, `set Server.Locked x`, `set Modules.aicompanion.APIKey x`,
		`set Modules.aicompanion.ModerateOutput false`, `set Modules.aicompanion.ModerationModel x`} {
		_, err := Server(cmd, user, room, 0)
		require.NoError(t, err, cmd)
		require.Contains(t, serverOutput(user.UserId), `config name is locked`, cmd)
	}
	require.Equal(t, `ShippedSeed`, string(configs.GetServerConfig().Seed))
	require.Equal(t, shippedLockList, configs.GetServerConfig().Locked)

	// Positive control: an unlocked key still changes. SendText's normalize
	// stage capitalises the line and appends a period, so the real output is
	// "Config changed: motd=hello." and the match skips the first letter.
	_, err := Server(`set motd hello`, user, room, 0)
	require.NoError(t, err)
	require.Contains(t, serverOutput(user.UserId), `onfig changed: motd=hello`)
	require.Equal(t, `hello`, string(configs.GetServerConfig().Motd))
}

func TestIsEditAllowedDelegatesToConfigsIsLocked(t *testing.T) {
	serverLockConfig(t)
	require.False(t, isEditAllowed(`filepaths`), `Server.Locked prefix`)
	require.False(t, isEditAllowed(`server.seed`), `Server.Locked entry`)
	require.False(t, isEditAllowed(`modules.aicompanion.apikey`), `hard list`)
	require.False(t, isEditAllowed(`filepaths.webdomain`), `hard list`)
	require.False(t, isEditAllowed(`modules.aicompanion.moderateoutput`), `hard list (ruling 13)`)
	require.True(t, isEditAllowed(`modules.aicompanion`), `a partial path stays browsable`)
	require.True(t, isEditAllowed(`server.motd`))
}
```

- [ ] **Step 2: Run them**

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/usercommands/ -run 'TestServerSetRefusesLockedKeys|TestIsEditAllowedDelegatesToConfigsIsLocked' -count=1 -v
```

Expected: `TestServerSetRefusesLockedKeys` PASSES already (Task 2 put the
check in `SetVal`, which `server set` calls; Step 5 proves it can fail). The
refusal text sits mid-sentence (`Config change error: seed=1 (config name is
locked: Server.Seed).`), so normalize's capital and period never touch the
`config name is locked` match; the positive control matches
`onfig changed: motd=hello` because the rendered line starts with a capital
and ends with a period, and it also reads `Server.Motd` back. If the
positive control fails on the text alone, print `serverOutput` before
changing the assertion: it must be the rendering, not the write.
`TestIsEditAllowedDelegatesToConfigsIsLocked` FAILS with the message
`hard list` on `modules.aicompanion.apikey`, which the old `isEditAllowed`
allows. (`filepaths.webdomain` already fails closed through the shipped
`FilePaths` prefix.)

- [ ] **Step 3: Delegate isEditAllowed**

In `C:\tmp\dogmud-config-locks\internal\usercommands\admin.server.go`, replace
the whole function (master `:416-432`):

```go
func isEditAllowed(configPath string) bool {

	configPath = strings.ToLower(configPath)

	if strings.HasSuffix(configPath, "locked") {
		return false
	}

	sc := configs.GetServerConfig()
	for _, v := range sc.Locked {
		if strings.HasPrefix(configPath, strings.ToLower(v)) {
			return false
		}
	}

	return true
}
```

with:

```go
// isEditAllowed asks configs.IsLocked, the same rule SetVal enforces, so the
// menu and `server set` can never disagree about what is locked.
func isEditAllowed(configPath string) bool {
	return !configs.IsLocked(configPath)
}
```

- [ ] **Step 4: Run to verify both pass, plus the package**

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/usercommands/ -run 'TestServerSetRefusesLockedKeys|TestIsEditAllowedDelegatesToConfigsIsLocked' -count=1 -v && go build ./internal/usercommands/
```

Expected: PASS; build clean (`strings` is still used elsewhere in the file).

- [ ] **Step 5: Prove the `server set` test can fail**

Temporarily set the lock line in `internal/configs/configs.go` `setVal` to
`if false {`, run `TestServerSetRefusesLockedKeys`, expect FAIL at
`set Server.Seed 1` with the output missing `config name is locked`; restore
and rerun PASS. (A sabotaged run writes only to the scratch `CONFIG_PATH`.)

- [ ] **Step 6: Commit**

```bash
cd C:/tmp/dogmud-config-locks && git add internal/usercommands/admin.server.go internal/usercommands/admin.server_locks_test.go && git commit -m "$(cat <<'EOF'
fix(usercommands): server config menu and server set share one lock rule

isEditAllowed now delegates to configs.IsLocked, so the hard lock list
covers the menu too. Tests pin that server set refuses Server.Seed, the bare
seed suffix key, Server.Locked, the companion key and the companion's
moderation switch and model.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: `DisplayConfigData`, `RedactedValue`, and refusing the marker as a value

**Files:**
- Create: `internal/configs/config_display.go`
- Modify: `internal/configs/config_types.go:95-97`
- Modify: `internal/configs/configs.go` (`setVal`, after the lock check)
- Test: `internal/configs/config_display_test.go` (create)

- [ ] **Step 1: Write the failing tests**

Create `C:\tmp\dogmud-config-locks\internal\configs\config_display_test.go`:

```go
package configs

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

const displaySentinel = `sk-display-sentinel-51c9`

func displayTestConfig() Config {
	c := GetConfig()
	c.Integrations.Discord.WebhookUrl = ConfigSecret(displaySentinel)
	c.Modules = Modules{
		`aicompanion`: map[string]any{`APIKey`: displaySentinel, `APIKeyEnv`: `OPENAI_API_KEY`, `Model`: `gpt-test`},
		`othermod`:    map[string]any{`password`: displaySentinel, `Secret`: displaySentinel},
	}
	return c
}

func TestDisplayConfigDataRedactsSecrets(t *testing.T) {
	c := displayTestConfig()
	shown := c.DisplayConfigData()

	// %#v never calls String(), so a raw ConfigSecret would show here too.
	for name, value := range shown {
		if strings.Contains(fmt.Sprintf(`%#v`, value), displaySentinel) {
			t.Errorf(`DisplayConfigData shows the secret at %s`, name)
		}
	}
	for _, name := range []string{`Integrations.Discord.WebhookUrl`, `Server.Seed`,
		`Modules.aicompanion.APIKey`, `Modules.othermod.password`, `Modules.othermod.Secret`} {
		if shown[name] != RedactedValue {
			t.Errorf(`DisplayConfigData[%q] = %#v, want RedactedValue`, name, shown[name])
		}
	}
	// The leaf match is exact: an environment variable NAME is not a secret.
	if shown[`Modules.aicompanion.APIKeyEnv`] != `OPENAI_API_KEY` {
		t.Errorf(`APIKeyEnv = %#v, want it shown`, shown[`Modules.aicompanion.APIKeyEnv`])
	}
	if shown[`Modules.aicompanion.Model`] != `gpt-test` {
		t.Errorf(`Model = %#v, want it shown`, shown[`Modules.aicompanion.Model`])
	}

	raw := c.AllConfigData()
	if raw[`Modules.aicompanion.APIKey`] != displaySentinel {
		t.Errorf(`AllConfigData must stay raw (the lookups read it), got %#v`, raw[`Modules.aicompanion.APIKey`])
	}
	if len(shown) != len(raw) {
		t.Errorf(`DisplayConfigData has %d keys, AllConfigData %d: redaction must not drop keys`, len(shown), len(raw))
	}
}

func TestDisplayConfigDataKeepsExclusions(t *testing.T) {
	for name := range displayTestConfig().DisplayConfigData(`modules*`) {
		if strings.HasPrefix(strings.ToLower(name), `modules`) {
			t.Errorf(`excluded key %s still present`, name)
		}
	}
}

func TestConfigSecretStringIsRedactedValue(t *testing.T) {
	if got := ConfigSecret(`x`).String(); got != RedactedValue {
		t.Errorf(`ConfigSecret.String() = %q, want RedactedValue`, got)
	}
}

// TestSetValRefusesTheRedactionMarker: the server config prompt offers the
// displayed value as its default, so Enter on a redacted leaf would otherwise
// write the marker over the real secret.
func TestSetValRefusesTheRedactionMarker(t *testing.T) {
	overridePath := SetConfigWithLookupsForTest(t, displayTestConfig())

	err := SetVal(`Integrations.Discord.WebhookUrl`, RedactedValue)
	if !errors.Is(err, ErrRedactedValue) {
		t.Fatalf(`SetVal(webhook, RedactedValue) = %v, want ErrRedactedValue`, err)
	}
	if got := string(GetIntegrationsConfig().Discord.WebhookUrl); got != displaySentinel {
		t.Errorf(`WebhookUrl = %q, want the original secret kept`, got)
	}
	if _, statErr := os.Stat(overridePath); !os.IsNotExist(statErr) {
		t.Errorf(`a refused SetVal wrote %s`, overridePath)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/configs/ -run 'TestDisplayConfigData|TestConfigSecretStringIsRedactedValue|TestSetValRefusesTheRedactionMarker' -count=1
```

Expected: FAIL to compile, `undefined: RedactedValue`, `DisplayConfigData`,
`ErrRedactedValue`.

- [ ] **Step 3: Create config_display.go**

Create `C:\tmp\dogmud-config-locks\internal\configs\config_display.go`:

```go
package configs

import (
	"errors"
	"regexp"
	"strings"
)

// RedactedValue is what every human-facing view of the config prints in
// place of a secret. ConfigSecret.String returns it too.
const RedactedValue = `*** REDACTED ***`

// ErrRedactedValue refuses the redaction marker as a new value: a prompt that
// offers the displayed value as its default must not write the marker over
// the real secret.
var ErrRedactedValue = errors.New("value is the redaction marker, not a real value")

// secretLeafName matches the last path element of a key whose value is a
// credential even when it is not typed ConfigSecret. Module config lives in
// the untyped Modules map, so Modules.aicompanion.APIKey arrives as a plain
// string and only its name marks it.
var secretLeafName = regexp.MustCompile(`(?i)^(apikey|secret|password)$`)

// DisplayConfigData is AllConfigData for anything a person reads: the boot
// log, the server set listing, the server config menu and /viewconfig. Every
// ConfigSecret, and every leaf secretLeafName matches, reads RedactedValue.
// AllConfigData stays raw because the key and type lookups are built from it.
func (c Config) DisplayConfigData(excludeStrings ...string) map[string]any {
	out := c.AllConfigData(excludeStrings...)
	for name, value := range out {
		if isSecretConfigValue(name, value) {
			out[name] = RedactedValue
		}
	}
	return out
}

func isSecretConfigValue(name string, value any) bool {
	if _, ok := value.(ConfigSecret); ok {
		return true
	}
	leaf := name
	if i := strings.LastIndex(name, `.`); i >= 0 {
		leaf = name[i+1:]
	}
	return secretLeafName.MatchString(leaf)
}
```

- [ ] **Step 4: Point ConfigSecret.String at the constant**

In `C:\tmp\dogmud-config-locks\internal\configs\config_types.go` replace:

```go
func (c ConfigSecret) String() string {
	return `*** REDACTED ***`
}
```

with:

```go
func (c ConfigSecret) String() string {
	return RedactedValue
}
```

- [ ] **Step 5: Refuse the marker in setVal**

In `C:\tmp\dogmud-config-locks\internal\configs\configs.go`, in `setVal`,
directly after the lock check block

```go
	if isHardLocked(propertyPath) || (operator && IsLocked(propertyPath)) {
		return fmt.Errorf(`%w: %s`, ErrLockedConfig, propertyPath)
	}
```

insert:

```go

	if newVal == RedactedValue {
		return fmt.Errorf(`%w: %s`, ErrRedactedValue, propertyPath)
	}
```

- [ ] **Step 6: Run to verify they pass, plus the package**

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/configs/ -run 'TestDisplayConfigData|TestConfigSecretStringIsRedactedValue|TestSetValRefusesTheRedactionMarker' -count=1 -v && go test ./internal/configs/ -count=1
```

Expected: PASS; package `ok`.

- [ ] **Step 7: Prove the redaction test can fail**

Temporarily make `DisplayConfigData` `return c.AllConfigData(excludeStrings...)`,
run `TestDisplayConfigDataRedactsSecrets`, expect FAIL naming
`Modules.aicompanion.APIKey` (and `Integrations.Discord.WebhookUrl` through
`%#v`); restore, rerun PASS.

- [ ] **Step 8: Commit**

```bash
cd C:/tmp/dogmud-config-locks && git add internal/configs/config_display.go internal/configs/config_types.go internal/configs/configs.go internal/configs/config_display_test.go && git commit -m "$(cat <<'EOF'
feat(configs): DisplayConfigData, one redacted view of the config

Module settings live in the untyped Modules map, so the companion's APIKey
is a plain string that ConfigSecret redaction never touched. The new view
redacts every ConfigSecret and every leaf named apikey, secret or password.
SetVal refuses the redaction marker as a value, so a prompt that offers the
displayed value as its default cannot overwrite a secret with it.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Boot log through a testable function

**Files:**
- Create: `boot_config_log.go`
- Modify: `main.go:12` (import), `main.go:245-256`
- Test: `boot_config_log_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `C:\tmp\dogmud-config-locks\boot_config_log_test.go`:

```go
package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestBootConfigLogRedactsSecrets(t *testing.T) {
	const sentinel = `sk-bootlog-sentinel-7f3a`
	c := configs.GetConfig()
	c.Integrations.Discord.WebhookUrl = configs.ConfigSecret(sentinel)
	c.Modules = configs.Modules{`aicompanion`: map[string]any{`APIKey`: sentinel, `Model`: `gpt-test`}}

	logged := map[string]any{}
	var order []string
	logBootConfig(c, func(name string, value any) {
		logged[name] = value
		order = append(order, name)
	})

	// %#v never calls String(): a raw ConfigSecret shows here, as it would
	// under a JSON log handler.
	for name, value := range logged {
		if strings.Contains(fmt.Sprintf(`%#v`, value), sentinel) {
			t.Errorf(`boot log prints the secret at %s`, name)
		}
	}
	for _, name := range []string{`Modules.aicompanion.APIKey`, `Integrations.Discord.WebhookUrl`} {
		if logged[name] != configs.RedactedValue {
			t.Errorf(`boot log %s = %#v, want RedactedValue`, name, logged[name])
		}
	}
	if logged[`Modules.aicompanion.Model`] != `gpt-test` {
		t.Errorf(`a non-secret module value was redacted or dropped: %#v`, logged[`Modules.aicompanion.Model`])
	}
	for i := 1; i < len(order); i++ {
		if order[i-1] > order[i] {
			t.Fatalf(`boot log is not sorted: %q before %q`, order[i-1], order[i])
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
cd C:/tmp/dogmud-config-locks && go test . -run TestBootConfigLogRedactsSecrets -count=1
```

Expected: FAIL to compile, `undefined: logBootConfig`.

- [ ] **Step 3: Create boot_config_log.go**

Create `C:\tmp\dogmud-config-locks\boot_config_log.go`:

```go
package main

import (
	"slices"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// logBootConfig hands every config value to emit in sorted key order, read
// through DisplayConfigData so no secret reaches the boot log. It is split out
// of main() so a test can read exactly what the boot log would print.
func logBootConfig(c configs.Config, emit func(name string, value any)) {
	cfgData := c.DisplayConfigData()
	cfgKeys := make([]string, 0, len(cfgData))
	for k := range cfgData {
		cfgKeys = append(cfgKeys, k)
	}
	slices.Sort(cfgKeys)
	for _, k := range cfgKeys {
		emit(k, cfgData[k])
	}
}
```

- [ ] **Step 4: Call it from main.go**

In `C:\tmp\dogmud-config-locks\main.go` replace (master `:245-255`):

```go
	cfgData := c.AllConfigData()
	cfgKeys := make([]string, 0, len(cfgData))
	for k := range cfgData {
		cfgKeys = append(cfgKeys, k)
	}

	// sort the keys
	slices.Sort(cfgKeys)
	for _, k := range cfgKeys {
		mudlog.Info("Config", "name", k, "value", cfgData[k])
	}
```

with:

```go
	logBootConfig(c, func(name string, value any) {
		mudlog.Info("Config", "name", name, "value", value)
	})
```

and delete the now-unused `"slices"` line from the import block (master
`:12`). Confirm nothing else needs it:

```bash
cd C:/tmp/dogmud-config-locks && grep -n "slices\." main.go
```

Expected: no output (exit 1 is correct here, run it standalone).

- [ ] **Step 5: Run to verify it passes, and build**

```bash
cd C:/tmp/dogmud-config-locks && go build . && go test . -run TestBootConfigLogRedactsSecrets -count=1 -v
```

Expected: build clean; PASS.

- [ ] **Step 6: Prove it can fail**

Temporarily change `c.DisplayConfigData()` to `c.AllConfigData()` in
`boot_config_log.go`, run the test, expect FAIL `boot log prints the secret
at Modules.aicompanion.APIKey`; restore, rerun PASS.

- [ ] **Step 7: Commit**

```bash
cd C:/tmp/dogmud-config-locks && git add boot_config_log.go boot_config_log_test.go main.go && git commit -m "$(cat <<'EOF'
fix(boot): the boot config log prints the redacted view

The boot log printed every config value raw, including the companion's
APIKey. It moves into logBootConfig, which reads DisplayConfigData and which
a test can call.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: `server set` listing and `server config` menu read the redacted view

**Files:**
- Modify: `internal/usercommands/admin.server.go:55, 320, 338, 392, 440`
- Test: `internal/usercommands/admin.server_locks_test.go` (append)

- [ ] **Step 1: Write the failing test**

Append to `C:\tmp\dogmud-config-locks\internal\usercommands\admin.server_locks_test.go`:

```go
func TestServerListingsRedactSecrets(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	serverLockConfig(t)
	user, room := getTestUserAndRoom(t)
	events.DrainQueuedMessagesForTest(user.UserId)

	// `server set` with no key lists every value.
	_, err := Server(`set`, user, room, 0)
	require.NoError(t, err)
	listing := serverOutput(user.UserId)
	require.Contains(t, listing, `aicompanion.APIKey`, `the listing must still name the key`)
	require.Contains(t, listing, configs.RedactedValue)
	require.NotContains(t, listing, serverLockSentinel)

	// The server config menu: section listings and single leaves.
	for _, path := range []string{`modules.aicompanion`, `modules.aicompanion.apikey`, `integrations.discord`, `integrations.discord.webhookurl`} {
		opts, ok := getConfigOptions(path)
		require.True(t, ok, path)
		require.NotEmpty(t, opts, path)
		for _, o := range opts {
			require.NotContains(t, o.Description, serverLockSentinel, path)
		}
	}
	opts, _ := getConfigOptions(`modules.aicompanion.apikey`)
	require.Equal(t, configs.RedactedValue, opts[0].Description)
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/usercommands/ -run TestServerListingsRedactSecrets -count=1
```

Expected: FAIL, the listing contains `sk-server-listing-sentinel-2d8e`.

- [ ] **Step 3: Switch all five reads**

In `C:\tmp\dogmud-config-locks\internal\usercommands\admin.server.go`, use
the Edit tool with `replace_all: true`, replacing
`configs.GetConfig().AllConfigData()` with
`configs.GetConfig().DisplayConfigData()`. Then confirm the count:

```bash
cd C:/tmp/dogmud-config-locks && grep -c "DisplayConfigData()" internal/usercommands/admin.server.go
```

Expected: `5` (master lines 55, 320, 338, 392, 440). And, standalone:

```bash
cd C:/tmp/dogmud-config-locks && grep -n "AllConfigData" internal/usercommands/admin.server.go
```

Expected: no output (exit 1 is correct).

These five are all display (listing, prompt default, confirmation, menu
descriptions); `getConfigOptions` uses only the KEYS for path lookup, and the
key set is identical in both views.

- [ ] **Step 4: Run to verify it passes, plus the package**

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/usercommands/ -run 'TestServerListingsRedactSecrets|TestServerSetRefusesLockedKeys|TestIsEditAllowed' -count=1 -v && go test ./internal/usercommands/ -count=1
```

Expected: PASS; package `ok`.

- [ ] **Step 5: Commit**

```bash
cd C:/tmp/dogmud-config-locks && git add internal/usercommands/admin.server.go internal/usercommands/admin.server_locks_test.go && git commit -m "$(cat <<'EOF'
fix(usercommands): server listings and the config menu show secrets redacted

server set's listing, the server config menu descriptions, its prompt
default and its confirmation all read DisplayConfigData. The prompt default
is the redaction marker for a secret, which SetVal now refuses.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 8: `/viewconfig` reads the redacted view and drops module and API settings

**Files:**
- Modify: `_datafiles/html/public/viewconfig.html:11`
- Test: `internal/web/viewconfig_redaction_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `C:\tmp\dogmud-config-locks\internal\web\viewconfig_redaction_test.go`:

```go
package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/require"
)

// publicHtmlDir resolves the shipped _datafiles/html/public from this file's
// own path; the test binary's CWD is not reliable (auth_test.go chdirs).
func publicHtmlDir(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(here), "..", "..", "_datafiles", "html", "public")
}

// TestViewConfigShowsNoSecret renders the real viewconfig.html through
// serveTemplate, the unauthenticated "/" handler.
func TestViewConfigShowsNoSecret(t *testing.T) {
	const sentinel = `sk-viewconfig-sentinel-93b1`
	c := configs.GetConfig()
	c.Integrations.Discord.WebhookUrl = configs.ConfigSecret(sentinel)
	c.Modules = configs.Modules{`aicompanion`: map[string]any{`APIKey`: sentinel, `RelayOrigin`: `https://keys.example.org`}}
	configs.SetConfigForTest(t, c)

	prev := httpRoot
	httpRoot = publicHtmlDir(t)
	t.Cleanup(func() { httpRoot = prev })

	req := httptest.NewRequest(http.MethodGet, `/viewconfig`, nil)
	rec := httptest.NewRecorder()
	serveTemplate(rec, req)

	body := rec.Body.String()
	require.Equal(t, http.StatusOK, rec.Code, body)
	require.Contains(t, body, `Server.MudName`, `the page must still render config rows`)
	require.NotContains(t, body, sentinel)
	require.NotContains(t, body, `Modules.aicompanion`, `/viewconfig leaves module settings out`)
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/web/ -run TestViewConfigShowsNoSecret -count=1
```

Expected: FAIL, body contains `sk-viewconfig-sentinel-93b1` (the
`Modules.aicompanion.APIKey` row). This red run is also the null probe for
the test.

- [ ] **Step 3: Change the template**

In `C:\tmp\dogmud-config-locks\_datafiles\html\public\viewconfig.html`
replace line 11:

```
            {{range $name, $value := (.CONFIG.AllConfigData "bannednames" "*startroom*" "script*"  "onlogin*" "nextroomid*" "mob*" "*port" "seed*" "folder*" "file*" "seedint" "carefulsavefiles" "lootgoblin*" "maxcpucores" "*roommessagewrapper" "logintervalroundcount" ) }}
```

with:

```
            {{range $name, $value := (.CONFIG.DisplayConfigData "bannednames" "*startroom*" "script*"  "onlogin*" "nextroomid*" "mob*" "*port" "seed*" "folder*" "file*" "seedint" "carefulsavefiles" "lootgoblin*" "maxcpucores" "*roommessagewrapper" "logintervalroundcount" "modules*" "apiframework*" ) }}
```

- [ ] **Step 4: Run to verify it passes, plus the package**

```bash
cd C:/tmp/dogmud-config-locks && go test ./internal/web/ -run TestViewConfigShowsNoSecret -count=1 -v && go test ./internal/web/ -count=1
```

Expected: PASS; package `ok` (`TestDiagramArtifactsHaveNoTemplateDelimiters`
is unaffected: it scans `diagrams/` only).

- [ ] **Step 5: Commit**

```bash
cd C:/tmp/dogmud-config-locks && git add _datafiles/html/public/viewconfig.html internal/web/viewconfig_redaction_test.go && git commit -m "$(cat <<'EOF'
fix(web): /viewconfig shows the redacted config, without module settings

The public, unauthenticated /viewconfig page printed the companion's APIKey.
It reads DisplayConfigData and excludes modules* and apiframework*.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 9: Root guard, no human-facing reader reads the raw config

**Files:**
- Test: `config_display_guard_test.go` (create)

- [ ] **Step 1: Write the guard**

Create `C:\tmp\dogmud-config-locks\config_display_guard_test.go`:

```go
package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// allConfigDataCallers lists repo-relative files outside internal/configs
// allowed to call a raw config reader, each with its reason. AllConfigData,
// DotPaths and GetOverrides return secrets raw; they exist because
// internal/configs builds the key and type lookups and the override file
// from them. Anything a person reads (logs, command output, web pages) goes
// through DisplayConfigData. A caller that needs only a key's TYPE may be
// added here with its reason. Empty on master: every outside caller was a
// display.
var allConfigDataCallers = map[string]string{}

// rawConfigSelectors are the Go selectors that return config values raw.
var rawConfigSelectors = map[string]bool{
	"AllConfigData": true,
	"DotPaths":      true,
	"GetOverrides":  true,
}

// rawConfigInTemplate matches a page template reaching raw config values:
// the three raw readers by name, the untyped Modules map through the page's
// CONFIG, or through the getconfig template func
// (internal/web/template_func.go) inside one {{ }} action.
var rawConfigInTemplate = regexp.MustCompile(`AllConfigData|DotPaths|GetOverrides|\.CONFIG\.Modules\b|\{\{[^}]*getconfig[^}]*\.Modules\b`)

// TestNoDisplayReadsRawConfig fails when a Go file outside internal/configs,
// or any page template, reads the raw config. Slice M found four displays
// (boot log, server set listing, server config menu, /viewconfig) printing
// the companion's APIKey raw because each read the raw view.
func TestNoDisplayReadsRawConfig(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	var offenders []string
	scanned := map[string]bool{}
	fset := token.NewFileSet()

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			// Dot-dirs include agent worktrees under .claude/, a second copy of the tree.
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return filepath.SkipDir
			}
			switch d.Name() {
			case "vendor", "node_modules", "docs", "testdata":
				return filepath.SkipDir
			}
			if rel == "internal/configs" {
				return filepath.SkipDir
			}
			return nil
		}
		if _, ok := allConfigDataCallers[rel]; ok {
			return nil
		}

		switch {
		case strings.HasSuffix(rel, ".go"):
			if strings.HasSuffix(rel, "_test.go") {
				return nil
			}
			file, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if perr != nil {
				// A syntax error is the compiler's to report.
				return nil
			}
			scanned[rel] = true
			ast.Inspect(file, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok && rawConfigSelectors[sel.Sel.Name] {
					offenders = append(offenders, rel+":"+strconv.Itoa(fset.Position(sel.Pos()).Line)+" "+sel.Sel.Name)
				}
				return true
			})
		case strings.HasSuffix(rel, ".html"), strings.HasSuffix(rel, ".template"), strings.HasSuffix(rel, ".tmpl"):
			body, rerr := os.ReadFile(path)
			if rerr != nil {
				return nil
			}
			scanned[rel] = true
			for _, m := range rawConfigInTemplate.FindAllString(string(body), -1) {
				offenders = append(offenders, rel+" "+m)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo: %v", err)
	}

	// Verify the negative: the walk must have read the files that held the
	// four displays, or an empty offender list proves nothing.
	for _, must := range []string{"main.go", "boot_config_log.go",
		"internal/usercommands/admin.server.go", "_datafiles/html/public/viewconfig.html"} {
		if !scanned[must] {
			t.Errorf("guard never scanned %s; the walk cannot see what it guards", must)
		}
	}

	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("%d raw config read(s) outside internal/configs. AllConfigData, DotPaths, GetOverrides\n"+
			"and the Modules map return secrets raw.\n"+
			"Use DisplayConfigData for anything a person reads, or add the file to\n"+
			"allConfigDataCallers in this file with the reason it needs only key types.\n  %s",
			len(offenders), strings.Join(offenders, "\n  "))
	}
}
```

- [ ] **Step 2: Run it**

```bash
cd C:/tmp/dogmud-config-locks && go test . -run TestNoDisplayReadsRawConfig -count=1 -v
```

Expected: PASS (Tasks 6 to 8 removed every outside call).

- [ ] **Step 3: Prove it can fail, in both file kinds and on every pattern**

Make these temporary edits (Edit tool), all at once:

1. In `internal/usercommands/admin.server.go`, change one
   `DisplayConfigData()` back to `AllConfigData()`.
2. In the same file, directly under that line, add three lines:
   `probeCfg := configs.GetConfig()`, `_ = probeCfg.DotPaths()` and
   `_ = configs.GetOverrides()`. (`DotPaths` has a pointer receiver, so it
   needs the addressable local; `go test .` compiles `usercommands` because
   `main` imports it, so the probe must compile.)
3. In `_datafiles/html/public/viewconfig.html`, change `DisplayConfigData`
   back to `AllConfigData` on line 11, and add a line after it:
   `{{ .CONFIG.Modules }}{{ (getconfig).Modules }}`.

Run the guard:

```bash
cd C:/tmp/dogmud-config-locks && go test . -run TestNoDisplayReadsRawConfig -count=1
```

Expected: FAIL listing six offenders:
`internal/usercommands/admin.server.go:<line> AllConfigData`,
`... GetOverrides`, `... DotPaths`, and
`_datafiles/html/public/viewconfig.html AllConfigData`,
`... .CONFIG.Modules`, `... {{ (getconfig).Modules`. If the run fails to
compile instead, the probe lines are wrong, not the guard: fix them and
rerun. Restore all three edits, rerun PASS, and
confirm `git status --short` shows only `config_display_guard_test.go` as
new.

- [ ] **Step 4: Commit**

```bash
cd C:/tmp/dogmud-config-locks && git add config_display_guard_test.go && git commit -m "$(cat <<'EOF'
test(configs): guard that no display reads the raw config

Fails when any Go file outside internal/configs, or any page template,
calls AllConfigData, DotPaths or GetOverrides, or a template reaches the
Modules map through .CONFIG or getconfig. Asserts the walk actually scanned
the four files that held the leaking displays.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 10: Docs, `internal/configs/context.md` and the `config.yaml` Locked comment

**Files:**
- Modify: `internal/configs/context.md` (Security Features list, Dot-Notation block, "Locked Configuration Properties" block, Runtime Updates block, Files table)
- Modify: `_datafiles/config.yaml:94-98`

- [ ] **Step 1: Verify every symbol you are about to document exists**

```bash
cd C:/tmp/dogmud-config-locks && grep -n -E "^func (\(c Config\) )?(IsLocked|SetVal|SetEngineVal|DisplayConfigData|SetConfigWithLookupsForTest|buildKeyLookups|isHardLocked)\(|^var (hardLocked|ErrRedactedValue|secretLeafName)|^const RedactedValue" internal/configs/*.go
```

Expected: one hit each for `IsLocked`, `SetVal`, `SetEngineVal`,
`DisplayConfigData`, `SetConfigWithLookupsForTest`, `buildKeyLookups`,
`isHardLocked`, `hardLocked`, `ErrRedactedValue`, `secretLeafName`,
`RedactedValue`.

- [ ] **Step 2: Edit context.md (Edit tool only)**

In `C:\tmp\dogmud-config-locks\internal\configs\context.md`:

(a) Replace

```
### 4. **Security Features**
- `ConfigSecret` type automatically redacts sensitive values in logs and output
- Environment variable support for secure credential injection
- Locked configuration properties to prevent unauthorized changes
- Validation of user input against banned patterns
```

with

```
### 4. **Security Features**
- `ConfigSecret` type automatically redacts sensitive values in logs and output
- `Config.DisplayConfigData` is the only view a person may read (see "Locks and the redacted view")
- Environment variable support for secure credential injection
- `SetVal` refuses locked keys: `Server.Locked` plus the Go hard list `hardLocked`
- Validation of user input against banned patterns
```

(b) Replace

```
### Locked Configuration Properties
```go
// Some properties cannot be changed at runtime
func isEditAllowed(configPath string) bool {
    serverConfig := configs.GetServerConfig()
    for _, lockedPath := range serverConfig.Locked {
        if configPath == lockedPath {
            return false
        }
    }
    return true
}
```
```

with

```
### Locks and the redacted view

**Locks (`config_locks.go`).** `IsLocked(path)` is true when the path is on
`hardLocked` (exact, lowercase), ends in `locked`, or starts (lowercase) with
an entry of `Server.Locked`. `hardLocked` holds the security keys no in-game
command may change whatever `Server.Locked` says: `APIFramework.APIKey`,
`.APIKeyEnv`, `.BaseURL`, `.AllowCustomEndpoint` (the section is absent on
master; the entries cost nothing), `Modules.aicompanion.APIKey`, `.APIKeyEnv`,
`.BaseURL`, `.AllowCustomEndpoint`, `.RelayOrigin`, `.PlayerKeys`, `.Model`,
`.FastModel`, `.DeepModel`, `.ModerateOutput`, `.ModerationModel`,
`FilePaths.WebDomain`, `Server.Locked`.

- `SetVal` is the OPERATOR write (`server set`, the `server config` menu,
  `setmotd`, `plugins.PluginConfig.Set`). It resolves the key with
  `FindFullPath`, then refuses with `ErrLockedConfig` when `IsLocked` names the
  RESOLVED path, before the unknown-key check. A bare suffix key (`seed`)
  cannot slip past a full-path lock.
- `SetEngineVal` is the ENGINE write for values the server maintains itself
  that `Server.Locked` keeps from operators: `Server.CurrentVersion`
  (`internal/migration`) and `Server.NextRoomId` (`internal/rooms`). It skips
  `Server.Locked` and still refuses `hardLocked`. A new engine-owned key in
  the shipped lock list must use it, or its write is silently refused.
- Both refuse `RedactedValue` as a value (`ErrRedactedValue`).
- `usercommands.isEditAllowed` delegates to `IsLocked`.

**Redacted view (`config_display.go`).** `AllConfigData` returns values raw
and is for the lookups only (`buildKeyLookups`). `DisplayConfigData` takes the
same exclusion patterns and replaces with `RedactedValue` every `ConfigSecret`
and every leaf whose last path element matches `(?i)^(apikey|secret|password)$`.
Module settings are untyped (`Modules map[string]any`), so a module's key is a
plain string and only its name marks it. The boot log (`logBootConfig` in
`boot_config_log.go`), the `server set` listing, the `server config` menu and
`/viewconfig` all read `DisplayConfigData`; the root test
`config_display_guard_test.go` fails on any `AllConfigData`, `DotPaths` or
`GetOverrides` call outside this package, and on any page template that
reaches `Modules` through `.CONFIG` or `getconfig`.

**Tests.** `SetConfigWithLookupsForTest(t, c) string` installs `c` with real
lookups (a test binary never runs `ReloadConfig`, so without it every key is
"unknown" and a lock test passes for the wrong reason), snapshots `overrides`,
and points `CONFIG_PATH` at a scratch file whose path it returns.

**Known gap, not fixed here.** `ReloadConfig` builds the lookups from the
config BEFORE the load, so on a single boot module keys without a data
overlay (the aicompanion's) do not resolve and `SetVal` refuses them as
unknown.
```

(c) In the Dot-Notation block replace

```
// All configuration paths support dot notation
allConfig := config.AllConfigData()
```

with

```
// All configuration paths support dot notation. AllConfigData is RAW (lookups
// only); anything a person reads uses DisplayConfigData.
allConfig := config.DisplayConfigData()
```

(d) In "Runtime Updates" replace

```
// Configuration changes are immediately persisted
err := configs.SetVal("Server.MudName", "New Name")
```

with

```
// Configuration changes are immediately persisted. SetVal refuses locked keys
// (ErrLockedConfig); engine-owned locked keys use SetEngineVal.
err := configs.SetVal("Server.MudName", "New Name")
```

(e) In the Files table, after the `config_types.go` row add:

```
| `config_locks.go` | `hardLocked`, `IsLocked`: which keys `SetVal` refuses |
| `config_display.go` | `DisplayConfigData`, `RedactedValue`: the redacted view |
```

- [ ] **Step 3: Edit the Locked comment in config.yaml (Edit tool only)**

Task 0 Step 2 confirmed the worktree's copy IS the HEAD blob, so edit it
directly. In `C:\tmp\dogmud-config-locks\_datafiles\config.yaml` replace:

```
  # - Locked -
  #   All config names defined here are immutable to the `server set` admin
  #   command. They can only be changed by editing the config file directly.
  #   It is a good idea to lock configs related to folder/file paths to prevent
  #   accidental changes that could break the game.
```

with:

```
  # - Locked -
  #   All config names defined here are immutable to the `server set` admin
  #   command and the `server config` menu. They can only be changed by
  #   editing the config file directly. A name locks itself and everything
  #   under it: FilePaths locks every FilePaths setting.
  #   It is a good idea to lock configs related to folder/file paths to prevent
  #   accidental changes that could break the game.
  #   The server still updates CurrentVersion and NextRoomId itself.
  #   Separately, the server always locks its security settings whatever this
  #   list says: API keys and key variable names, provider addresses, the
  #   companion's models, moderation switch and moderation model,
  #   player-key switch and relay origin, FilePaths.WebDomain, and this list
  #   itself (internal/configs/config_locks.go).
```

The block grows from 5 lines to 13, so every `config.yaml` line after it
(master `:99` onward) shifts down by 8. Informational only: slice H keys its
`config.yaml` edits by text, not by line number, but any line citation into
`config.yaml` made against master (this plan's facts table included) is 8
low after this commit.

Then verify only the comment moved:

```bash
BASE=$(cat C:/tmp/dogmud-config-locks.base); cd C:/tmp/dogmud-config-locks && git diff --stat $BASE -- _datafiles/config.yaml && git diff $BASE -- _datafiles/config.yaml | grep -E "^[+-][^+-]" | grep -v -E "^[+-]  #"
```

Expected: `1 file changed`, with exactly 8 more insertions than deletions;
the second command prints nothing (every changed line
is a comment; exit 1 from `grep` is correct).

- [ ] **Step 4: Scan the new prose for dashes**

```bash
BASE=$(cat C:/tmp/dogmud-config-locks.base); cd C:/tmp/dogmud-config-locks && git diff $BASE -- internal/configs/context.md _datafiles/config.yaml | grep -n -P "\x{2013}|\x{2014}"
```

Expected: no output (exit 1 is correct).

- [ ] **Step 5: Commit**

```bash
cd C:/tmp/dogmud-config-locks && git add internal/configs/context.md _datafiles/config.yaml && git commit -m "$(cat <<'EOF'
docs(configs): document the lock rule, engine writes and the redacted view

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 11: Gate

**Files:** none changed (fix and recommit if any step fails).

- [ ] **Step 0: Size, against `$BASE` (ruling 11, the CI lint inversion)**

```bash
BASE=$(cat C:/tmp/dogmud-config-locks.base); cd C:/tmp/dogmud-config-locks && git diff --shortstat $BASE..HEAD && git diff --name-only $BASE..HEAD | wc -l
```

Expected: the shortstat's insertions plus deletions well under 20000 (this
slice is about 1k) and the file count under 300 (this slice is about 20). A
larger number means `$BASE` is wrong or something unrelated was committed;
stop and find out which before going on.

- [ ] **Step 1: gofmt, on every Go file changed since `$BASE`, plus the tree**

```bash
BASE=$(cat C:/tmp/dogmud-config-locks.base); cd C:/tmp/dogmud-config-locks && git diff --name-only --diff-filter=d $BASE..HEAD -- '*.go' | xargs -r gofmt -l && gofmt -l internal/ modules/
```

Expected: no output from either. If a file is listed, check whether it is
the Windows CRLF false positive before touching it:

```bash
cd C:/tmp/dogmud-config-locks && git show HEAD:<listed/path.go> | gofmt -l
```

Empty output means the committed blob is formatted and the hit came from a
CRLF working copy; anything else is real, fix with `gofmt -w <file>` and
commit.

- [ ] **Step 2: Build and vet**

```bash
cd C:/tmp/dogmud-config-locks && go build ./... && go vet . ./internal/configs/ ./internal/usercommands/ ./internal/web/ ./internal/rooms/ ./internal/migration/ ./internal/plugins/
```

Expected: no output.

- [ ] **Step 3: Targeted tests**

```bash
cd C:/tmp/dogmud-config-locks && go test -count=1 -run 'TestSetConfigWithLookupsForTest|TestSetVal|TestSetEngineVal|TestIsLocked|TestDisplayConfigData|TestConfigSecretStringIsRedactedValue' ./internal/configs/ && go test -count=1 -run 'TestServerSetRefusesLockedKeys|TestIsEditAllowed|TestServerListingsRedactSecrets' ./internal/usercommands/ && go test -count=1 -run TestViewConfigShowsNoSecret ./internal/web/ && go test -count=1 -run TestRecordMigratedVersion ./internal/migration/ && go test -count=1 -run TestSetNextRoomIdWritesThroughServerLocked ./internal/rooms/ && go test -count=1 -run 'TestBootConfigLogRedactsSecrets|TestNoDisplayReadsRawConfig' .
```

Expected: every line `ok`.

- [ ] **Step 4: Full packages, the root package and the callers**

```bash
cd C:/tmp/dogmud-config-locks && go test -count=1 ./internal/configs/... ./internal/usercommands/... ./internal/web/... && go test -count=1 . ./internal/rooms/... ./internal/migration/... ./internal/plugins/...
```

Expected: all `ok`. The root package is mandatory
(`condition_apply_path_guard_test.go` is a line-number allowlist; this plan
touches none of its files, but run it anyway). Compare any failure against
the Task 0 baseline before treating it as new.

- [ ] **Step 4b: The whole suite, once**

`SetVal` is global and now refuses what it used to write, so a caller this
plan never opened can change behaviour. `rooms.SetNextRoomId`, for one, is
also called from `internal/devtools/gridgen.go:55`. Run everything once:

```bash
cd C:/tmp/dogmud-config-locks && go test -count=1 ./... 2>&1 | grep -v -E "^(ok|\?) " ; echo "done"
```

Expected: only `done` (every package `ok` or `no test files`). This takes
several minutes; `internal/combat` alone runs for minutes, so run it in the
background if the harness times out at two. Compare any failure against the
Task 0 baseline (rerun the failing package on `$BASE` in a scratch detached
worktree if the baseline did not cover it) before treating it as new.

- [ ] **Step 5: Lint, from `$BASE`**

```bash
BASE=$(cat C:/tmp/dogmud-config-locks.base); cd C:/tmp/dogmud-config-locks && golangci-lint run --new-from-rev=$BASE
```

Expected: `0 issues.`

- [ ] **Step 5b: context.md audit**

```bash
BASE=$(cat C:/tmp/dogmud-config-locks.base); cd C:/tmp/dogmud-config-locks && git diff --name-only $BASE..HEAD -- internal/configs/context.md && python tools/context_md_audit.py internal/configs
```

Expected: the first command prints `internal/configs/context.md` (the
package API changed, so its context.md must be in the diff); the audit
reports no symbol this plan's context.md text introduced. (Task 10 (b)
removes the old `func isEditAllowed` go block, which named a
`usercommands` symbol.)

- [ ] **Step 6: Boot check in a detached worktree, on private ports**

Build to a fixed path, never `go run .`. The override file moves every port
off the owner's server and absorbs any engine write; the worktree absorbs any
migration. `timeout` stops the process, nothing is killed by name.

```bash
git -C C:/tmp/dogmud-config-locks worktree add --detach C:/tmp/dogmud-boot-check-m HEAD
cd C:/tmp/dogmud-boot-check-m && cat > boot-overrides.yaml <<'EOF'
Network:
  TelnetPort: ["33533"]
  LocalPort: 9899
  HttpPort: 8391
  HttpsPort: 0
  AIPort: 0
EOF
go build -o boot-check.exe . && CONFIG_PATH=boot-overrides.yaml LOG_NOCOLOR=1 timeout 180 ./boot-check.exe > boot.log 2>&1; echo "exit=$?"
```

Expected: `exit=124` (the server stayed up until the timeout). Then, each
standalone:

```bash
cd C:/tmp/dogmud-boot-check-m && grep -cE "^panic:|goroutine [0-9]+ \[running\]|runtime error" boot.log
```

Expected: `0`.

```bash
cd C:/tmp/dogmud-boot-check-m && grep -c "Server Ready" boot.log
```

Expected: `1`.

```bash
cd C:/tmp/dogmud-boot-check-m && grep "Modules.aicompanion.APIKey" boot.log
```

Expected: one line whose value reads `*** REDACTED ***`.

```bash
cd C:/tmp/dogmud-boot-check-m && grep -E "Starting http server.*port=8391" boot.log
```

Expected: one line (the http server started on the private port, not the
owner's). A bare `grep 8391` proves nothing: the boot config dump prints the
port whatever happens to the bind. Then, standalone:

```bash
cd C:/tmp/dogmud-boot-check-m && grep -c "Error starting web server" boot.log
```

Expected: `0` (exit 1 is correct).

**The migration version persists (the #171 bug class).** A fresh worktree's
config carries no `Server.CurrentVersion` value, so `Validate` falls back to
`0.9.0` and this first boot ran every migration, then recorded the version
through `SetEngineVal` into `CONFIG_PATH`. Each standalone:

```bash
cd C:/tmp/dogmud-boot-check-m && grep -c "Migration 0.18.0" boot.log
```

Expected: at least `1` (the migrations ran on the first boot; if `0`, the
check below proves nothing, so stop and find out why).

```bash
cd C:/tmp/dogmud-boot-check-m && grep CurrentVersion boot-overrides.yaml
```

Expected: one line carrying `0.18.0` (`main.go`'s `VERSION`; if master's
`VERSION` has moved, that value). No line means the engine write was refused
and every migration would re-run on every boot.

Boot a second time on the same override file:

```bash
cd C:/tmp/dogmud-boot-check-m && CONFIG_PATH=boot-overrides.yaml LOG_NOCOLOR=1 timeout 180 ./boot-check.exe > boot2.log 2>&1; echo "exit=$?"
```

Expected: `exit=124`. Then, each standalone:

```bash
cd C:/tmp/dogmud-boot-check-m && grep -c "Server Ready" boot2.log
```

Expected: `1`.

```bash
cd C:/tmp/dogmud-boot-check-m && grep -c "Migration 0" boot2.log
```

Expected: `0` (exit 1 is correct): the second boot read `0.18.0` back and
ran no migration. Then tear down:

```bash
git -C C:/tmp/dogmud-config-locks worktree remove --force C:/tmp/dogmud-boot-check-m
```

If Windows holds a lock on the exe, use PowerShell
`Remove-Item -Recurse -Force C:\tmp\dogmud-boot-check-m` and then
`git -C C:/tmp/dogmud-config-locks worktree prune`.

- [ ] **Step 7: Branch state**

```bash
BASE=$(cat C:/tmp/dogmud-config-locks.base); cd C:/tmp/dogmud-config-locks && git status --short && git log --oneline $BASE..HEAD
```

Expected: clean tree; ten commits (Tasks 1 to 10).

---

### Task 12: Delivery, slice M's own PR to master (ruling 11, `dogmud-shipping`)

**Files:**
- Modify: `docs/PATCH_NOTES.md` (new entry at the top)

- [ ] **Step 1: Patch notes entry (Edit tool)**

The pre-push SOP asks for a dated entry. In
`C:\tmp\dogmud-config-locks\docs\PATCH_NOTES.md`, directly under the
`# DOGMud Patch Notes` heading line and its following blank line, insert
the block below. If you commit on a later day than 2026-09-28, put that day
(`date +%F`) in the heading instead.

```
## 2026-09-28: Server secrets stay secret

The public server settings page no longer lists module settings, and no
page, log or admin listing shows a secret setting any more. Security
settings, such as service keys and where they are sent, can no longer be
changed from inside the game; they change only in the config file.

```

Then scan it for dashes and commit:

```bash
BASE=$(cat C:/tmp/dogmud-config-locks.base); cd C:/tmp/dogmud-config-locks && git diff $BASE -- docs/PATCH_NOTES.md | grep -n -P "\x{2013}|\x{2014}"
```

Expected: no output (exit 1 is correct).

```bash
cd C:/tmp/dogmud-config-locks && git add docs/PATCH_NOTES.md && git commit -m "$(cat <<'EOF'
docs(patch-notes): server secrets stay secret

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 2: Re-run the size check with the final commit**

```bash
BASE=$(cat C:/tmp/dogmud-config-locks.base); cd C:/tmp/dogmud-config-locks && git diff --shortstat $BASE..HEAD && git diff --name-only $BASE..HEAD | wc -l
```

Expected: under 20000 lines changed and under 300 files, as in Task 11
Step 0.

- [ ] **Step 3: Confirm PR #175 has merged first**

```bash
gh pr view 175 --repo pruuk/DOGMud --json state --jq .state
```

Expected: `MERGED`. If it prints `OPEN`, stop here and report: ruling 11
merges #175 first. Pushing and opening this PR may wait with it; merging may
not.

- [ ] **Step 4: Push and open the PR against pruuk/DOGMud**

```bash
cd C:/tmp/dogmud-config-locks && git push -u origin fix/config-locks-redaction
```

```bash
cd C:/tmp/dogmud-config-locks && gh pr create --repo pruuk/DOGMud --base master --head fix/config-locks-redaction --title "fix(configs): config locks and redaction (slice M)" --body "$(cat <<'EOF'
Slice M of docs/superpowers/specs/2026-09-28-baubles-hardening-and-corpus-design.md.

- SetVal (operator writes: server set, the server config menu, setmotd, module setters) refuses any key under Server.Locked or on a hard Go list: API keys and key variable names, endpoints, the companion's models, moderation switch and moderation model, relay origin, player keys, FilePaths.WebDomain and Server.Locked itself. The check runs on the resolved path, so a bare suffix key cannot slip past.
- SetEngineVal carries the engine's own writes to Server.CurrentVersion (migration) and Server.NextRoomId (rooms), which the shipped Locked list names; without it every migration would re-run on every boot.
- DisplayConfigData redacts every ConfigSecret and every leaf named apikey, secret or password. The boot log, the server set listing, the server config menu and /viewconfig read it; /viewconfig also drops module settings. SetVal refuses the redaction marker as a value.
- A root guard fails on any raw config read (AllConfigData, DotPaths, GetOverrides, or Modules from a template) outside internal/configs.

Verified: targeted tests, go test ./..., golangci-lint from the branch point, and a two-boot check on private ports (secret redacted in the boot log; CurrentVersion recorded on boot one; no migration on boot two).

Not deployed. The owner deploys.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

Expected: `gh` prints a URL starting `https://github.com/pruuk/DOGMud/pull/`.
If it names `GoMudEngine/GoMud`, close that PR at once and report.

- [ ] **Step 5: Watch the checks, read failures, merge**

```bash
gh pr checks <n> --repo pruuk/DOGMud --watch
```

Expected: every check passes. A green summary is not proof: for any run
that emitted annotations, read `gh run view <id> --repo pruuk/DOGMud
--log-failed`. With #175 confirmed merged (Step 3) and checks green:

```bash
gh pr merge <n> --repo pruuk/DOGMud --merge --delete-branch
```

`<n>` is the number Step 4 printed. `--merge`, never `--squash`. Do NOT
deploy: the owner runs every deploy. After merge, the owner's main checkout
carries `config.yaml` with the skip-worktree bit, so its disk copy will lag
this comment change until the EOD re-sync builds it from the HEAD blob.
Then remove the worktree and the base file:

```bash
git -C "C:/Users/Calabe Davis/workspace/DOGMud" worktree remove C:/tmp/dogmud-config-locks && rm C:/tmp/dogmud-config-locks.base
```

---

## Self-review

- **Spec coverage.** M1: `SetVal` refuses on the resolved path (Task 2);
  `Server.Locked` prefix, lowercase (Task 2 `IsLocked`); `hardLocked` with the
  spec's list, APIFramework entries included by decision (Task 2);
  `isEditAllowed` delegates (Task 4); `Server.Locked` hard-locked (Task 2);
  module setters inherit through `PluginConfig.Set` (facts table, no code
  needed); every `SetVal` caller grepped, and the two engine callers the spec
  missed are moved (Task 3). M2: `DisplayConfigData` with `ConfigSecret` and
  the leaf regex (Task 5); `AllConfigData` stays raw (Task 5 test); boot log
  moved to a testable function (Task 6); both `server` listings (Task 7);
  `/viewconfig` plus `modules*`, `apiframework*` (Task 8); guard with a named
  allowlist (Task 9, empty on master with the reason). Tests: `server set
  Server.Seed 1`, `server set seed 1`, `server set Server.Locked x` refused
  (Tasks 2 and 4); a sentinel in both master key locations absent from the
  boot-log function (Task 6), both listings (Task 7) and a rendered
  `/viewconfig` (Task 8). Docs: `internal/configs/context.md` and the
  `config.yaml` Locked comment (Task 10).
- **Rulings.** 11: own master PR after #175 (Task 12 Step 3), `$BASE` from
  Task 0 for every diff, gofmt, lint and context.md check, the size check in
  Task 11 Step 0 and Task 12 Step 2, `--repo pruuk/DOGMud` on every `gh`
  call, no deploy. 13: companion `ModerateOutput` and `ModerationModel` in
  `hardLocked` with test rows in Tasks 2 and 4; the baubles pair is slice H's.
- **Placeholder scan.** Every code step carries the code; the only `<...>`
  are the gofmt recheck (the path gofmt printed) and the PR number `<n>` that
  Task 12 Step 4 prints.
- **Type consistency.** `SetConfigWithLookupsForTest(t, c) string`,
  `SetEngineVal(string, string) error`, `IsLocked(string) bool`,
  `DisplayConfigData(...string) map[string]any`, `RedactedValue`,
  `ErrRedactedValue`, `logBootConfig(configs.Config, func(string, any))`,
  `recordMigratedVersion(version.Version)` are used with the same shapes in
  every task.
