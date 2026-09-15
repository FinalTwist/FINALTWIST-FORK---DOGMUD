# Conditions unification, slice 3: disk, wire and save migration

Date: 2026-09-15. Branch `feature/conditions-unification-slice-3-disk-wire`
off master `854ff488c` (PR #132 merged; #111 to #132 not deployed). Slice 3,
the last slice of the conditions unification arc. Slice 1 unified the model,
slice 1b changed the mechanics, and slice 2 renamed everything compiled or
shown. This slice renames every spelling a file on disk, a client or a
content author reads, and migrates existing saves on first boot.

**This slice changes no behaviour.** Every record, spell, mob and save means
exactly what it meant before; only spellings change, three dead config knobs
and three dead docs go, and the admin `buff` alias is removed.

## Owner rulings (do not relitigate)

From 2026-09-12 and 2026-09-14 (arc and slice 2): buffs absorb conditions;
the rename goes all the way; slice 3 is the disk/wire slice (YAML keys and
values, `buffs/` folders, config keys, the save keys, GMCP JSON field names,
messaging category strings, template function names); literal swap of Buff
to Condition keeping each name's shape, with `PermaBuff` becoming `Permanent`.

From 2026-09-15:
1. **Delete the three dead config knobs** (`AllowItemBuffRemoval`,
   `DeathsShadowBuffId`, `broken_limb_buff_duration`) rather than rename
   them. The broken limb duration is authored on record 83, so the knob adds
   nothing.
2. **Saves migrate by a startup sweep** in the existing versioned migration
   framework. Go knows only the new keys afterwards; no dual-read.
3. **`melee_self_buff` becomes `melee_self_empower`**, a one-off exception to
   the literal swap, like `Permanent`.
4. **Delete docs that describe systems that do not exist**:
   `SCRIPTING_BUFFS.md` and its README link, the `ActorObject` buff functions
   in `FUNCTIONS_ACTORS.md`, and the "Available triggers" line in
   `docs/schemas/mob.md` (which lists `has_buff:N` / `missing_buff:N`; no
   trigger on that line has a parser).
5. **Approach A**: one PR, one shared rename table feeding both the content
   rewrite and the save migration.

## Facts verified against source

Read from the tree at `854ff488c` on 2026-09-15. Content counts are tracked
files only (`git grep`), because this machine carries gitignored dev saves
under `_datafiles/world/dogmud/users/`.

| Fact | Where |
|---|---|
| On-disk keys, files per world (dogmud / default): `buffid` 110 / 42, `buffids` 74 / 27, `buff_ids` 19 / 0, `buff_id` (behaviour tree param) 4 / 0, `critbuffids` 4 / 4, `wornbuffids` 1 / 5, `trapbuffids` 7 / 0, `playerbuffids` 6 / 5, `mobbuffids` 1 / 1, `start_remove_buffs` 1 / 0, `buffs` 0 / 1, `permabuff` 0 / 1 | `git grep -liE '^\s*-?\s*<key>\s*:'` per world |
| Tagged in Go with no tracked data: `nativebuffids` (commented out in mutator files), `prizebuffids`, quest `apply_buff` and nested `buff` (read by the builder JS) | `internal/mutators/mutators.go:67`, `internal/mobs/mobs.go:56`, `internal/quests/triggers.go:58,139` |
| Every buff-spelled tag: `characters/character.go:145` `buffs`; `characters/shop.go:21` `buffid`; `conditions/conditions.go:15,18` `buffid`, `permabuff`; `conditions/conditionspec.go:163,192` `buffid`, `start_remove_buffs`; `configs/config.balance.go:214`; `gamelock/gamelock.go:16` `trapbuffids`; `items/itemspec.go:225,263,264` `critbuffids`, `buffids`, `wornbuffids`; `mobs/mobs.go:56,123` `prizebuffids`, `buffids`; `mutators/mutators.go:65-67`; `pets/pets.go:27` `buffids`; `quests/quests.go:40` `buffid` (yaml+json); `quests/triggers.go:58,139`; `rooms/spawninfo.go:23` `buffids` / json `buffIds`; `species/species.go:35` `buffids`; `spells/spells.go:41` `buff_ids`; `modules/gmcp/gmcp.Item.go:84,113` `buffIds`, `wornBuffIds`; `gmcp.Mob.go:133,165` `buffIds`, `buffs`; `gmcp.Quest.go:122` `buffs` | grep of `yaml:"…buff` / `json:"…buff` |
| `effect_type: buff` in 17 dogmud spell files, 0 default; read by `case "buff":` at `internal/hooks/spell_resolution.go:906,1093,1493,1740` and `internal/usercommands/spells.go:32` | grep |
| `melee_self_buff`: `behaviors/archetypes/melee_self_buff.yaml`, mobs `summons/304-vampire.yaml:4`, `summons/313-fire_elemental.yaml:4`; Go `internal/itemvalue/profiles.go:152`; test files `behaviortree/melee_self_buff_*_test.go`. Not an archetype shift target (`behaviortree/archetype_shift.go:86`) | grep |
| Behaviour tree registry keys `add_buff`, `remove_buff`, `mob_has_buff` and param `buff_id`: `internal/behaviortree/actions.go:52,68,151`, `conditions.go:29,53`, `actions_combat.go:131,146`; data in `archetypes/ambusher.yaml`, `boss_rhett.yaml`, `boss_sylara.yaml`, `thornwall_city/272-chrysalis_phantom.yaml` | grep |
| Folders `_datafiles/world/dogmud/buffs/` (109 files), `_datafiles/world/default/buffs/` (39); one production path builder `internal/conditions/conditionspec.go:403` (`` `/buffs` ``) | `git ls-files`, grep |
| Template functions `buffname`, `buffduration` registered at `internal/templates/templatesfunctions.go:150,157`; used in `dogmud/templates/descriptions/identify.template:21,35` and `help/species.template` in both worlds | grep |
| `fg="buff"`: `help/species.template` (both worlds), `internal/actions/buy.go:756,767` (event log lines; `UserRecord.EventLog` is `yaml:"-"`, `internal/users/userrecord.go:58`, so never saved). Aliases `buff: 147`, unused `buff-text: 14`, `buff-apply: 109`, `buff-expire: 109` in `dogmud/ansi-aliases.yaml:167,168,272,273`; default has `buff`, `buff-text` at `:148,149` | grep |
| Category strings `"buff-apply"` / `"buff-expire"` at `internal/messaging/messaging.go:225,227`; no web client reader; no saved per-player preference | grep of `_datafiles/html`, `tools/`, `internal/users` |
| Admin alias: `keywords.yaml` `setcondition: [buff]` help alias and `['buff']` command alias at dogmud `:299,322`, default `:162,184` | grep |
| Builder JS readers of the GMCP fields: `_datafiles/html/public/static/js/items.js`, `mobs.js`, `quests.js`, `public/build.html`. The player client `webclient-pure.html` reads no buff-spelled field | grep |
| Admin `.data.html` form names `buffids[]`, `critbuffids[]`, `wornbuffids[]`, `playerbuffids[]`, `nativebuffids[]`, `spawninfo[…].buffids[]` in `_datafiles/html/admin/{items,mobs,species,mutators,rooms}/`; GET-only pages, no POST parser | grep of `internal/web` |
| `Char.Conditions` entries carry nested `Mods map[string]int` `json:"affects"` | `modules/gmcp/gmcp.Char.go:708` |
| Dead config knobs: `AllowItemBuffRemoval` (`config.gameplay.go:4`, skipped at `:60`), `DeathsShadowBuffId` (`:53`, clamp `:120-121`), `BrokenLimbBuffDuration` (`config.balance.go:214`, clamp `config.balance.combat.go:291-292`); no other reader. Shipped at `config.yaml:281,305,1107` (HEAD blob). Guard allowlist `identifier_word_guard_test.go:40-42` | grep; `git show HEAD:_datafiles/config.yaml` |
| Record 83 carries its own duration (`triggerrate: 1 round`, `triggercount: 900`) and is applied by `applyBrokenLimbCondition` (`internal/combat/submission_outcome.go:320-327`) without the knob | read |
| Weather key `BuffsEnabled` read as `get("BuffsEnabled")` with fallback **true** at `modules/weather/weather_config.go:111`; shipped `BuffsEnabled: false` at `config.yaml:2323` | grep |
| **Loaders ignore unknown keys**: config (`internal/configs/configs.go:465-471`), overrides (`:526-530`, `:143-155`), content (`internal/fileloader/fileloader.go:95`; strict probe only in `boot_smoke_test.go`), user saves (`internal/users/users.go:498`, plain `yaml.Unmarshal`) | read |
| Versioned migration framework: `migration.Run` (`internal/migration/migration.go:102`) backs up DataFiles to a temp dir (`backup.go:13`), runs `doAllMigrations` gated on `IsOlderThan`, restores on error, then sets `Server.CurrentVersion`. Called from `main.go:209`, skipped on copyover. `VERSION = "0.16.0"` at `main.go:97`; last step is 0.16.0 | read |
| 🪤 **The 0.14.0 player sweep skips alts.** `reclassifyUsersInDir` globs `users/*.yaml`, which matches `<id>.alts.yaml`, unmarshals into a map, and on the list-shaped alts file warns and `continue`s | `internal/migration/0.14.0.go:54-80` |
| Saves carrying buff keys: `users/<id>.yaml` (`character.buffs[].buffid`, `permabuff`, `character.pet.buffids`); `users/<id>.alts.yaml` (a YAML list of characters, same shape; `internal/characters/alts.go:25`); `rooms.instances/**` (containers are instance-saved, `rooms/rooms.go:95`, and `rooms/container.go:9` carries a `gamelock.Lock` with `trapbuffids`; exits are `instance:"skip"`, `:96`; `SpawnInfo` is `instance:"skip"`, `:105`); `shops/**` (`ShopItem.ConditionId` `buffid`, none in stock today) | read |
| Mob instance saves carry nothing to migrate: `SaveMobInstance` marshals `MobInstanceData` (`internal/mobs/instance_save.go:22-56`), which has no condition field and a `behavior_archetype` that only holds shift targets | read |
| No JavaScript scripting layer exists: no `internal/scripting`, no goja/otto in `go.mod`, zero `.js` under `_datafiles/world` | `ls`, grep, `find` |
| Stale docs: `_datafiles/guides/building/scripting/SCRIPTING_BUFFS.md`, linked from that folder's `README.md:14-15`; `FUNCTIONS_ACTORS.md:39-43,327-361` (ActorObject buff functions pointing at the missing `internal/scripting/actor_func.go`); `docs/schemas/mob.md:200` "Available triggers" line (`health_below:N` through `has_buff:N`, `missing_buff:N`): no parser for any of them; the only similar name is the behaviour tree's `mob_health_below` (`internal/behaviortree/conditions.go:19`) | read, grep |
| Schema docs naming buff keys: `docs/schemas/buff.md` (280 lines), `item.md`, `mob.md`, `spell.md`, `room.md`, `behavior.md`, `pinnacle-items.md`; `docs/README.md:20` | grep |
| Freeze tests that pin today's spellings: root `wire_freeze_test.go` (5 tests), `modules/gmcp/gmcp_wire_freeze_test.go:14`, `internal/keywords/keywords_setcondition_alias_test.go` | read |

## Name map

**Keys**: `buffid` → `conditionid`; `buffids` → `conditionids`; `buff_ids` →
`condition_ids`; `buff_id` → `condition_id`; `critbuffids` →
`critconditionids`; `wornbuffids` → `wornconditionids`; `trapbuffids` →
`trapconditionids`; `playerbuffids` → `playerconditionids`; `mobbuffids` →
`mobconditionids`; `nativebuffids` → `nativeconditionids`; `prizebuffids` →
`prizeconditionids`; `start_remove_buffs` → `start_remove_conditions`;
`buffs` (character save) → `conditions`; `permabuff` → `permanent`; quest
`apply_buff` → `apply_condition` and its nested `buff` → `condition`.

**Values**: `effect_type: buff` → `effect_type: condition`;
`melee_self_buff` → `melee_self_empower`; behaviour tree `add_buff`,
`remove_buff`, `mob_has_buff` → `add_condition`, `remove_condition`,
`mob_has_condition`.

**Wire**: GMCP `buffIds` → `conditionIds`, `wornBuffIds` →
`wornConditionIds`, `buffs` → `conditions`, quest `buffid` → `conditionid`;
categories `buff-apply` / `buff-expire` → `condition-apply` /
`condition-expire`; colour alias `buff` → `condition` (and `buff-text`
deleted, unused); template functions `buffname` / `buffduration` →
`conditionname` / `conditionduration`.

**Files**: `buffs/` → `conditions/` (both worlds);
`melee_self_buff.yaml` → `melee_self_empower.yaml`; the two
`melee_self_buff_*_test.go` files follow; `docs/schemas/buff.md` →
`docs/schemas/condition.md`.

**Config**: weather `BuffsEnabled` → `ConditionsEnabled`. Three knobs
deleted.

**Not renamed**: the nested `affects` key in `Char.Conditions` (not buff
spelled; the player client reads it); history (completed and past specs and
plans, published patch notes, commit messages, dated playtest goal and
scenario files).

## Design

### 1. The rename table and the content rewrite

**Table.** New package `internal/conditionrename`, one Go file, importable by
`migration` and tests. Two lists, both from the name map: **keys** and
**values**. Each entry carries the context it applies in (the folders a key
occurs in, the key a value belongs to), so the bare quest `buff` key and the
save `buffs` key cannot be renamed anywhere else. The table is the single
source of truth; nothing else lists old spellings.

**Go changes, in the same commit as the data.** Every buff-spelled `yaml:` /
`json:` tag; `case "buff"` (5 sites); behaviour tree registry keys and the
`condition_id` param; `itemvalue` archetype case; the loader path
`/conditions`; template function registrations; the two category strings.

**Content rewrite.** A throwaway authoring tool (not committed) applies the
table as text edits anchored to each entry's context, so comments, quoting
and key order stay byte-identical apart from renamed tokens. `git mv` for
the folders and renamed files, in both worlds. Also rewritten: templates,
`ansi-aliases.yaml`, admin `.data.html` form names, builder JS, and the
tracked fixture `_datafiles/world/default/users/1.yaml`.

**One-time equivalence proof** (a plan verification step, result recorded
in the PR, not committed): decode every YAML file under both worlds at
`master` and at the branch, apply the table to the old tree, and require
deep equality. A missed key, a rename in the wrong context, or any other
data change fails it.

**Committed load test**: every shipped record type loads through its real
loader with a nonzero count per folder, in both worlds, and a sample of
known ids resolves its renamed fields (a spell's `condition_ids`, an item's
`wornconditionids`, a species' `conditionids`, a trapped room lock). This is
the silent-empty trap: a tag and a file that disagree load as zero values
with no error.

### 2. The startup migration (0.17.0)

`VERSION` becomes `0.17.0`; `doAllMigrations` gains
`if lastConfigVersion.IsOlderThan(version.New(0, 17, 0))` calling
`migrate_ConditionKeys(false)`. Backup, restore on error and
`Server.CurrentVersion` come from the framework.

**Targets**, under DataFiles, read and written with `gopkg.in/yaml.v2`
(what their owners use):

| Target | Shape | Key paths |
|---|---|---|
| `users/<id>.yaml` (skip `users.idx`) | map | `character.buffs` → `conditions`; each entry `buffid`, `permabuff`; `character.pet.buffids` |
| `users/<id>.alts.yaml` | **list** of characters | same paths per element |
| `rooms.instances/**/*.yaml` | map | `containers.*.lock.trapbuffids` |
| `shops/**/*.yaml` | map | `buffid` in stock entries |

- **Path-anchored**: each target renames only its listed paths, taken from
  the table. Nothing else in a file is touched.
- **Idempotent, no marker**: a file with no old key is not rewritten. A
  second run changes nothing, so the per-alt marker trap cannot arise.
- **Collision is an error**: an old key whose new key already exists in the
  same map returns an error naming the file; `Run` restores the backup and
  the server exits.
- **Unparseable is an error**, unlike 0.14.0's warn-and-skip: a skipped save
  would lose its conditions on next load, silently.
- **Logging**: one line per rewritten file, a count per target.
- **Dry run**: `migrate_ConditionKeys(true)` logs and writes nothing.

**Tests** on fixture directories (a user map with conditions and a pet, an
alts list, a room instance with a trapped container, a shop): before/after
content; second run is a no-op (file bytes and mtime unchanged); collision
errors; malformed file errors; a migrated user loads through `users.LoadUser`
and its alts through `characters.LoadAlts` with conditions, `Permanent` and
pet condition ids intact. Every assertion null-probed.

### 3. Wire, config, text and deletions

- **GMCP builder payloads** per the name map, with `items.js`, `mobs.js`,
  `quests.js`, `build.html` updated in the same commit, internal JS names
  (`buffBox`, `rBuff`, datalist ids) included. The quest action vocabulary
  entry in `gmcp.Quest.go` follows. `gmcp_wire_freeze_test.go` flips to the
  new names.
- **Config**: delete the three knobs (fields, clamps, `config.yaml` entries
  and comment blocks, guard allowlist entries), committing from the
  `git show HEAD:` blob because `config.yaml` is skip-worktree. Rename the
  weather key in `config.yaml` and `weather_config.go:111`.
- **Admin alias**: remove `buff` from both `keywords.yaml` alias lists in both
  worlds and the "buff still works" lines from both `setcondition` help
  templates; the alias tests flip to assert `buff` is not recognised.
- **Colours**: `condition`, `condition-apply`, `condition-expire` aliases land
  in the same commit as the Go strings; a test renders a condition start line
  and asserts the colour code is present (a missing alias drops colour
  silently).
- **Content prose**: comments in condition YAML that say buff or name old
  hooks (for example record 83's `Buff_ApplyBuffs`) are corrected.
- **Docs**: `docs/schemas/buff.md` → `condition.md` (keys, folder, filename
  formula, drop the "until slice 3" banner); keys updated in the other schema
  docs and `docs/README.md`; the three dead-doc deletions; `context.md` files
  and `.claude/skills` that quote disk spellings; `internal/conditions/context.md`
  loses its "slice 3 renames those" paragraph.
- **Patch note**: none; no player sees a change. The PR body lists the admin
  changes (alias gone, builder field names).

### 4. Guard, commit order, testing, rollout

**Guard.** `identifier_word_guard_test.go` extends from Go identifiers and
template field reads to: Go string literals and struct tag values
(`go/scanner` STRING tokens), and tracked `.yaml`, `.template`, `.html`,
`.js`, `.md` under `_datafiles/` and `docs/schemas/`. It fails on `(?i)buff`
not inside `buffer`. Short allowlist, each entry with a reason and required
to still match. Skips vanished files. Null-probed with a planted `buffid:`
and a planted tag.

**Commit order**, one PR, each commit building and passing the suite:

1. `internal/conditionrename` and its tests.
2. The atomic data commit: tags, Go strings, cases, registry, loader path,
   folder and file moves, content rewrite, templates, colour aliases,
   fixture save, flipped `wire_freeze_test.go`, load test. Equivalence proof
   run here.
3. GMCP builder fields and JS, flipped GMCP freeze test.
4. Migration 0.17.0 and tests.
5. Config deletions and weather key.
6. Admin alias removal, docs and deletions.
7. The guard.

**Testing.**
- `gofmt`, `go vet`, full `go test ./...` with `DOGMUD_BOOT_SMOKE=1`; the
  expected reds are only the two pre-existing `internal/rooms` zone lifecycle
  tests that fail locally on Windows (filed 2026-09-15, green in CI).
- Web client node tests; local `golangci-lint run --new-from-rev=master`
  (CI lint goes red on a >300-file PR from the diff API 406).
- **Migration rehearsal on real saves**: a detached worktree at the branch
  with this machine's gitignored dev saves copied in, booted at 0.17.0: the
  log lists rewritten files, no buff key survives under `users/`,
  `rooms.instances/`, `shops/`, a second boot rewrites nothing, and a
  spot-checked user loads with conditions intact.
- Boot check in a detached worktree, both worlds loading, zero panics.
- **Smoke playtest**, single agent, short: log in holding a condition; cast
  Iron Will (formerly `effect_type: buff`); take a bleed on the `early`
  profile at room 3015; admin `setcondition` works and `buff` is refused;
  summon a fire elemental (`melee_self_empower`).

**Rollout notes for the PR body** (the owner deploys):
- First boot on the droplet runs 0.17.0, which copies all datafiles to a temp
  dir first; check free disk on the droplet before deploying.
- On error the server restores the backup and exits; the log names the file.
- A manual snapshot of `users/`, `rooms.instances/` and `shops/` before the
  deploy is cheap insurance.
- A local skip-worktree `config.yaml` that still says `BuffsEnabled` turns
  weather conditions on (fallback true); update the local copy.

## Out of scope

- The nested `affects` key in `Char.Conditions`.
- Any behaviour change, including wiring the deleted knobs.
- The slice 1b owner product calls.
- The rest of `FUNCTIONS_ACTORS.md`: the whole ActorObject guide describes the
  missing scripting layer, but the owner ruling covers only its buff
  functions. Flagged for a docs cleanup.
- History: completed and past specs and plans, published patch notes, dated
  playtest goal and scenario files.
