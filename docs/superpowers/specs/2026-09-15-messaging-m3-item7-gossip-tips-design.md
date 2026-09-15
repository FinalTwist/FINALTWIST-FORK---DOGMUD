# Messaging M3 item 7: gossip onto the narration core, tips get a store, conversations stay

Date: 2026-09-15. Branch `feature/messaging-m3-item7-conversations-gossip-hints`
off master `c7960ac39`. Item 7 of M3 in the messaging unification arc
(`2026-08-31-messaging-unification-design.md`, M3 table: "conversations,
gossip_templates, hints").

Owner rulings taken during this design (2026-09-15, do not relitigate):

- **Conversations are not migrated.** The arc spec planned to map speaker A/B
  onto Actor/Actee. From source, A and B are two NPCs saying consecutive lines
  through their own `say` command, one line per round: two speakers of a
  sequence, not two audiences of one moment. The exchange is already picked
  once, and each line already has exactly one audience (the room, through
  `say`). The package already has its own loader and validators. Forcing it
  through `narration.Render` would be abstraction for its own sake. Delivery
  through `say` remains M4's.
- **The periodic broadcast is renamed from hints to tips, all the way down**,
  including each player's saved on/off flag, by a data migration (0.18.0).
  The earlier ruling (2026-09-11) was to rename it so it stops colliding with
  the quest `hint` command.
- **Tips are not wrapped by the server, and item 7 leaves that alone.** It is
  recorded as ledger row 22 for M5's wrap decision.

## Facts verified against source

Every claim below was read from the tree at `c7960ac39` on 2026-09-15.

### Gossip

| Fact | Where |
|---|---|
| `_datafiles/world/dogmud/gossip_templates.yaml`: 34 top-level keys, each a list of 2 to 5 lines, none blank. The default world has no such file | `yaml.safe_load` count; `ls _datafiles/world/default` |
| Tokens in the file: `{desc}` 83 times, `{description}` 4 times. `{tag}` appears once but only inside a comment. No template contains a token twice | grep, and a count of `{desc}.*{desc}` lines (0) |
| Keys are `EventType-Significance[-Local|-Distant]` for world events, plus `fallback` and `fact-default`; `fact-{factId}` and `fact-{tag}` are looked up but none ships | `internal/hooks/MobIdle_HandleIdleMobs.go:380-541`; key list from the file |
| The loader is package globals in hooks: `gossipTemplates map[string][]string` and `gossipTemplatesOnce sync.Once`, loaded lazily by the first `buildGossipLine` | `MobIdle_HandleIdleMobs.go:375-378,399-418,429-430` |
| A missing file or a parse error is only logged, and gossip goes silently empty for the life of the process. There is no content validation | `MobIdle_HandleIdleMobs.go:402-414` |
| Pools are picked with `util.Rand(len(pool))`, one draw per pick. World-event templates substitute with `strings.Replace(tmpl, "{desc}", desc, 1)`; fact templates with `strings.ReplaceAll(tmpl, "{description}", desc)` | `MobIdle_HandleIdleMobs.go:538-540,547-560` |
| The final fallback when no template matches an event is a Go literal, `I heard that %s` | `MobIdle_HandleIdleMobs.go:534-536` |
| One production caller, the gossiper idle branch, which runs `mob.Command("say " + line)` | `MobIdle_HandleIdleMobs.go:227-230` |
| Three hooks tests reach into the globals (`gossipTemplatesOnce.Do(func() {})` then assign `gossipTemplates`): `TestBuildGossipLine_FallbackWhenNoEvents`, `_EmptyTemplatesEmptyEvents`, `_KnownFactUsedWhenNoEvents`. `TestBuildGossipLine_*` are known shuffle-order flakes | `internal/hooks/hooks_test.go:2968-3053`; memory of the 5b execution |
| `narration.Render(v, tokens, pick)` calls `pick(n)` exactly once per render; `DefaultPicker(n)` is `util.Rand(n)` | `internal/narration/render.go:92-121`; `picker.go:14` |
| Files allowed to call `narration.Render` are registered in `narrationRenderCallers` (five Kind A stores plus `textutil/narrate.go`) | `narration_render_callers_guard_test.go:19-26` |
| `loadAllDataFiles(isReload bool)` is the boot and reload entry point for store loaders (`crafting.LoadRecipeFiles`, `itemvoices.LoadDataFiles`, ...) | `main.go:1615-1645` |

### Tips (today "hints")

| Fact | Where |
|---|---|
| `_datafiles/world/dogmud/hints.yaml`: key `hints:`, 74 entries, all folded scalars (`>-`), none blank, no `{` | `yaml.safe_load` count |
| 56 of 74 exceed 80 characters with tags stripped (longest 211); the file header says each "should fit within 80 characters", which is false because a folded scalar joins its lines | same measurement |
| Exactly one reader: `internal/hooks/NewRound_BroadcastHints.go`, which loads the file lazily (`hintOnce sync.Once`), logs and stays empty on error, rotates `hintMessages[hintIndex%len]` every 75 rounds, prefixes `<ansi fg="cyan-bold">[Tip]</ansi> `, skips `Deafened` users and users whose `GetConfigOption("hints")` is `false`, sends on `messaging.CategoryTip`, then queues `events.Communication{CommType: "broadcast", Name: "Tip", Message: tip}` | `NewRound_BroadcastHints.go:15-95` |
| The arc spec's "reached from 15 files" counted the spelling `hints`, which is almost entirely dialogue (`internal/dialogue/types.go:76,99,112`, 1,266 `hints:` lines in dialogue YAML) | grep |
| Registered listeners: `events.RegisterListener(events.NewRound{}, BroadcastHints)` and `events.RegisterListener(events.Looking{}, HandleLookHints)` | `internal/hooks/hooks.go:56,91` |
| `HandleLookHints` is a separate one-shot contextual tip (the `list` tip near merchants), gated by `user.DidTip("list")` and sent on `CategoryTip`; it does not read `hints.yaml`. Four tests call it | `internal/hooks/Looking_HandleLookHints.go`; `hooks_test.go:2195-2219` |
| The player toggle: `set hints` calls `cmdSetToggle(user, "hints", "Hints", true)`; the `set` listing calls `displayBoolSetting(user, "hints", "hints")`. The value is stored in `UserRecord.ConfigOptions` (`yaml:"configoptions"`) under key `hints` | `internal/usercommands/set.go:40-41,98,139-190`; `internal/users/userrecord.go:47` |
| No other code reads the `hints` config option; the web client's `hints` references are the dialogue editor | grep of `internal`, `modules`, `_datafiles/html` |
| `UserRecord.TipsComplete` (`yaml:"tipscomplete"`) is the one-shot tip record, a different top-level key; no `tips` key exists in `configoptions` today | `userrecord.go:57,318-333` |
| `help set` does not document `set hints` at all (dogmud or default world) | `_datafiles/world/*/templates/help/set.template` |
| The M0 key registry's `hints` entry names `hints.yaml` as its one outlier file | `messaging_surface_guard_test.go:204` |

### Migrations

| Fact | Where |
|---|---|
| `main.go` `const VERSION = "0.17.0"`; `migration.doAllMigrations` runs each `IsOlderThan(version.New(...))` block in order | `main.go:97`; `internal/migration/migration.go:14-110` |
| `migration.Run` backs up all of DataFiles first and restores the backup if any migration returns an error, then sets `Server.CurrentVersion` | `migration.go:112-145` |
| Per-user migrations glob `DataFiles/users/*.yaml`, skip `users.idx`, parse into a generic map so every other field survives, and expose a `...InDir(dir, dryRun)` core for tests | `internal/migration/0.16.0.go:31-70` |
| `configoptions` lives on the USER record, not the character, so the alts trap (a character-scoped marker re-running per alt) does not apply | `userrecord.go:47` |

### Conversations (unchanged, for the ruling)

| Fact | Where |
|---|---|
| 5 type pools and 9 pair overrides; an exchange is an ordered list of `{speaker: A|B, text}` | `_datafiles/world/dogmud/conversations/{types,pairs}` |
| Lines are delivered one per round as `ConvCommand("say " + text)` from the speaking NPC | `internal/conversations/state.go:183,232` |
| Selection uses `util.Rand` once for the exchange; the package has its own loader and pool validators | `conversation.go:147`; `loader.go` |

## What item 7 delivers

- `internal/gossip`: the gossip store, loaded at boot and on reload, validated,
  rendering every pick through `narration.Render`.
- `internal/tips`: the tips store (`tips.yaml`), loaded at boot and on reload,
  validated, owning the rotation.
- The broadcast, its toggle, its help and its saved flag all say "tips".
- Migration 0.18.0 renames each player's saved flag.
- A ruling, in this spec and the arc spec, that conversations stay.

Gossip output is byte-identical on shipped data, with the same random draw
count. Tip text and order are byte-identical. The visible changes are the
command name (`set tips`, with `set hints` still accepted) and its line in
`set` and `help set`.

## Design

### Gossip store (`internal/gossip`)

```go
// Load reads DataFiles/gossip_templates.yaml. A missing file is an empty
// store (the default world has none). A parse error or a failed Validate
// panics, as a bad recipe or spell does.
func Load()

// Validate refuses a blank line, and a line that carries the same token
// twice (world-event lines substitute {desc} once; refusing repeats makes
// that difference from ReplaceAll unreachable).
func Validate(templates map[string][]string) error

// Pool returns the lines for key, nil if absent.
func Pool(key string) []string

// Render picks one line from pool and substitutes token -> value, drawing
// exactly once: it is renderWith(pool, token, value, narration.DefaultPicker).
func Render(pool []string, token, value string) string

// renderWith is the testable core; the draw-count test passes a counting
// picker.
func renderWith(pool []string, token, value string, pick narration.Picker) string

// SeedForTest replaces the store for one test and restores it on cleanup.
func SeedForTest(t testing.TB, templates map[string][]string)
```

`Render` builds `narration.Variants{Actor: pool}` (the gossiping NPC speaks;
there is no other role) and passes `map[string]string{token: value}`. A pool
of one line still draws, exactly as `util.Rand(1)` does today.

`buildGossipLine` and `renderFactGossip` stay in hooks with their filtering,
dedup and 70/30 event-versus-fact choice unchanged. Only three things change
inside them: the lazy `Once` load goes, template lookups call `gossip.Pool`,
and each `tmpls[util.Rand(len(tmpls))]` plus its `strings.Replace` becomes
`gossip.Render(tmpls, "{desc}", evt.Description)` or
`gossip.Render(tmpls, "{description}", kf.Fact.Description)`. The `I heard
that %s` literal fallback stays in hooks. The three hooks tests use
`gossip.SeedForTest`. `internal/gossip/gossip.go` is registered in
`narrationRenderCallers` as a Kind A store.

`loadAllDataFiles` calls `gossip.Load()` beside `itemvoices.LoadDataFiles()`.

### Tips store (`internal/tips`)

```go
// Load reads DataFiles/tips.yaml (key `tips:`). Missing file: empty store.
// Parse error or failed Validate: panic.
func Load()

// Validate refuses a blank tip.
func Validate(tips []string) error

// Next returns the next tip in rotation and advances, or "" when empty.
func Next() string

// SeedForTest replaces the store and resets the rotation for one test.
func SeedForTest(t testing.TB, tips []string)
```

No 80-column rule: 56 shipped tips would fail it, and wrapping is M5's.

`_datafiles/world/dogmud/hints.yaml` is renamed with `git mv` to `tips.yaml`,
its key changes from `hints:` to `tips:`, and its header comment is corrected
(tips are joined into one line and the client wraps them). Content unchanged.

`NewRound_BroadcastHints.go` becomes `NewRound_BroadcastTips.go` with
`BroadcastTips`; it calls `tips.Next()` and keeps the interval, prefix,
Deafened skip, opt-out, `CategoryTip` send and `Communication` event exactly.
Its config key constant becomes `tipsConfigKey = "tips"`.
`Looking_HandleLookHints.go` becomes `Looking_HandleLookTips.go` with
`HandleLookTips`, body unchanged. Both registrations in `hooks.go` and the
four `HandleLookHints` tests are renamed.

`loadAllDataFiles` calls `tips.Load()`.

### The toggle and help

`set.go`: `case "tips", "hints": return cmdSetToggle(user, "tips", "Tips", true)`
and `displayBoolSetting(user, "tips", "tips")`. `hints` stays a silent alias
so muscle memory and old guides still work. Both worlds' `help set` gain:

```
  set tips
  This toggles the periodic gameplay tips on or off.
```

(formatted like the neighbouring `set tinymap` entry, 80 columns).

### Migration 0.18.0

`main.go` `VERSION` becomes `"0.18.0"`. `doAllMigrations` gains:

```go
if lastConfigVersion.IsOlderThan(version.New(0, 18, 0)) {
	if err := migrate_TipsConfigOption(false); err != nil {
		return err
	}
}
```

`internal/migration/0.18.0.go`: `migrate_TipsConfigOption(dryRun)` calls
`renameTipsConfigOptionInDir(filepath.Join(DataFiles, "users"), dryRun)`,
following 0.16.0's shape: glob `*.yaml`, skip `users.idx`, parse each into a
generic map, and when `configoptions` holds `hints`, move its value to `tips`
and write the file back.

- **Idempotent without a marker**: a save with no `hints` option is not
  rewritten, so a second run changes nothing. The flag is on the user record,
  so there is no alts trap.
- **Refuse the ambiguous case**: a save whose `configoptions` holds both
  `hints` and `tips` returns an error, so `Run` restores the backup instead of
  guessing which the player meant.
- **Only this key moves.** A save with a top-level `tipscomplete` or dialogue
  data is untouched; only `configoptions.hints` is renamed.

## The net

### Goldens first, from pre-migration code

Under `internal/narration/testdata/stores/`, committed before any production
change:

- `gossip.golden`: every key sorted, every variant index, keyed
  `gossip|<key>|<index> => <line>`, with the stand-in description
  `<the stand-in event>`. The pre-migration builder parses the YAML itself and
  applies today's expression to each index: `strings.Replace(line, "{desc}",
  desc, 1)` for every key not starting `fact-`, `strings.ReplaceAll(line,
  "{description}", desc)` for `fact-` keys. Today's code picks with
  `util.Rand`, which no test can pin, so the golden freezes the per-index
  substitution rather than a pick; that is the honest limit, stated in the
  golden's header.
- `tips.golden`: every tip in file order as `tip|<index> => <text>`. The
  pre-migration builder parses `hints.yaml`'s `hints:` list.

After the migration the gossip builder reads `gossip.Pool(key)` and renders
each index `i` through `gossip.RenderWithForTest` with the fixed picker
`func(int) int { return i }`, and the tips builder
calls `tips.Load` then `tips.Next` 74 times. Both goldens must be
byte-identical. The row set must match too: a key or tip lost in the move
shows as a missing row.

Because `internal/narration`'s snapshot test is an external test package and
cannot call the unexported `renderWith`, `internal/gossip` exports
`RenderWithForTest(pool, token, value, pick)` in a `test_helpers.go`,
following `internal/spells/test_helpers.go`.

### Draw-count test

`internal/gossip`: `renderWith` with a counting picker over pools of 1, 2 and
5 lines calls the picker exactly once each, matching today's single
`util.Rand(len)`. A second assertion reads `gossip.go` and requires `Render`
to pass `narration.DefaultPicker`, so production cannot drift to
`FirstPicker` (which would silently remove a global random draw per gossip
line and shift later rolls).

### Tests

- Gossip: `Validate` refuses a blank line and a repeated token; `Load` of a
  missing file yields an empty store; `Render` substitutes the named token.
- Tips: `Validate` refuses a blank tip; `Next` rotates and wraps; empty store
  returns "".
- `set tips` and `set hints` both toggle the same `tips` option; the listing
  shows `tips`.
- Migration: fixtures in a temp dir cover rename, no-op on a save without the
  option, no-op on a second run, error on both keys, `users.idx` skipped, and
  every other field preserved (compare the parsed map minus the moved key).

### Sabotage probes, each proven red

1. Swap `{desc}` for `{description}` in `gossip.Render`'s call for world
   events: `gossip.golden` red.
2. Remove the `hints` alias from `set.go`: the alias test red.
3. Delete the "both keys" check: its migration test red.
4. Make the migration rewrite every file unconditionally: the no-op test red
   (compare modification bytes, not timestamps).

### Root guards

- `narrationRenderCallers` gains `internal/gossip/gossip.go`.
- A store-file guard: no production Go file outside `internal/gossip` names
  `gossip_templates.yaml`, and none outside `internal/tips` names `tips.yaml`;
  and no production Go file names `hints.yaml` at all.
- The M0 key registry's `hints` reason drops its outlier note (the broadcast
  no longer uses the spelling); `TestEveryTextSurfaceIsRegistered` stays green.

### Gate

Full suite, `go test .`, gofmt, vet, boot check (both loaders log their counts:
34 gossip keys, 74 tips), and one playtest lane of about 25 minutes (the tip
interval is the constant `hintIntervalRounds = 75`, about 5 minutes, so the
lane sees four or five broadcasts): the player quotes each `[Tip]` line and
checks it against `tips.golden`, runs `set tips` off and confirms the next
broadcast does not arrive, runs `set hints` to turn tips back on and confirms
the one after does, checks that `set` lists `tips`, and stands in a gossiper's
tavern long enough to quote a gossip line, which must be one of that key's
rows in `gossip.golden` with the real event description in place of the
stand-in. The migration is proven by its fixture tests, not the playtest: an
ephemeral playtest env's version state is not a reliable way to exercise a
one-time migration.

## Out of scope, filed

- Server-side wrap for tips: ledger row 22, M5's wrap decision.
- Conversations: unchanged by ruling. Dialogue hints (`CategoryDialogueHint`
  in `ask.go` and `talk.go`) are dialogue content and unchanged.
- The quest `hint` command: the quest mechanisms arc.
- One token vocabulary (`{desc}` / `{description}` versus `{source}`): M4.

## Deploy note (for the owner's deploy, recorded in the PR)

0.18.0 rewrites player saves on the droplet's first boot after deploy. `Run`
backs up all of DataFiles first, so check droplet disk before deploying, as
for 0.17.0.

## Documentation

`context.md` for the new `internal/gossip` and `internal/tips` packages,
`internal/hooks` (renamed listeners, gossip no longer loaded there),
`internal/migration` (0.18.0), `internal/narration` (new goldens and Kind A
caller). The arc spec's M3 table row 7 gains a pointer to this spec's
conversations ruling. Rows for this spec and its plan in `docs/README.md`.
Patch note: tips are now called tips, `set tips` (with `set hints` still
working), and your setting carries over.
