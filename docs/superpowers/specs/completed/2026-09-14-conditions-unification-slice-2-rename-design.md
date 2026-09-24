# Conditions unification, slice 2: the Go and player-facing rename

Date: 2026-09-14. Branch `feature/conditions-unification-slice-2-rename` off
master `6f6a64696` (PR #130, slice 1b, merged and not deployed). Slice 2 of
the conditions unification arc: slice 1 unified the model, slice 1b changed
the mechanics, this slice renames buffs to conditions in everything compiled
or shown, and slice 3 renames the wire format and migrates saves.

**This slice changes no behaviour and nothing on disk or on the wire.** A
freeze test written before the first rename proves it.

## Owner rulings (do not relitigate)

From 2026-09-12: buffs absorb conditions; the unified thing is called a
condition; the rename goes all the way; slice 2 is the Go and player-facing
rename, slice 3 the YAML keys and saves.

From 2026-09-14:
1. **The line between slice 2 and slice 3 is the disk/wire rule.** If a file
   on disk or a client reads it, it waits for slice 3: YAML keys AND values
   (spell `effect_type: buff`, the `melee_self_buff` behaviour category, the
   `<ansi fg="buff">` colour tag), the `buffs/` data folders, config keys,
   the save `buffs:` key, GMCP JSON field names, the messaging category
   strings, template function names. Everything compiled or shown to a
   player or admin is slice 2.
2. **Literal name swap**: replace Buff with Condition, keeping each name's
   shape, so every old name maps predictably to its new one. One exception:
   `PermaBuff` becomes `Permanent`.
3. **Approach A**: compiler-driven renames (`gopls rename`), in layers, one
   PR.

## Facts verified against source

Read from the tree at `6f6a64696` on 2026-09-14.

| Fact | Where |
|---|---|
| `internal/buffs` is imported by 128 non-test and 140 test Go files | grep of the import path |
| About 8,400 identifier occurrences contain `buff`/`Buff` across Go, including false positives such as `bytes.Buffer`, `eventBuffer`, `Buffer []byte`, `WorldEventBufferSize` | grep |
| Package files: `buffs.go`, `buffspec.go`, `effects.go`, `ids.go`, `narration.go`, `notice.go`, `stacks.go`, `test_helpers.go`, `tick.go`, plus `context.md` and tests | `ls internal/buffs` |
| Exported API: types `Buff`, `Buffs`, `BuffSpec`, `BuffMessage`, `BuffMessages`, `Flag`, `EffectKind`, `Stack`; funcs `New`, `GetBuffSpec`, `GetAllBuffIds`, `SearchBuffs`, `LoadDataFiles`, `HasSpec`, `ValidateLoadedFlags`, `GetDurations`, `DisplayName`, `SeedBuffsForTest`, `SeedConditionRecordsForTest`; methods `AddBuff`, `AddBuffScaled`, `AddBuffMagnitude`, `RefreshBuff`, `RemoveBuff`, `HasBuff`, `GetBuffs`, `GetBuffIdsWithFlag`, `Started`, `TriggersLeft`, `Trigger`, `Prune`, `SetTickAmount`, `HasFlag`, `ProgressMult`, `StatMod`, `Effect`, `HasEffect`, `Listed`, `IsStacking`; constants `BuffIdWarcry` to `BuffIdEnchantWithdrawal` | `internal/buffs/*.go` |
| 🪤 **Untagged fields that ARE on disk or on the wire.** yaml.v2 names an untagged field by lowercasing the Go name, so renaming it silently changes the key: `buffs.Buff.BuffId` (player saves, key `buffid`); `buffs.BuffSpec.BuffId` (every buff YAML file, key `buffid`); `species.Species.BuffIds` (species YAML, `buffids`). Also untagged: `events.Buff.BuffId`, `events.BuffsTriggered.BuffIds`, `combat.AttackResult.BuffSource/BuffTarget`, `actions` `TrackResult.BuffApplied`, `weather` `BuffsEnabled`, `usercommands` `conditionEntry.PermaBuff` (template data) | awk scan of struct fields |
| Tagged buff-named fields (15), e.g. `Character.Buffs` `yaml:"buffs"`, `PermaBuff` `yaml:"permabuff"`, `StartRemoveBuffs`, `CritBuffIds`, `WornBuffIds`, `TrapBuffIds`, `PrizeBuffIds`, `PlayerBuffIds`/`MobBuffIds`/`NativeBuffIds`, `ApplyBuff`, quest `BuffId`/`Buff`, `rooms.SpawnInfo.BuffIds`, spell `BuffIds` `yaml:"buff_ids"`, gmcp `WornBuffIds`/`BuffIds`/`Buffs` JSON fields | grep of tagged fields |
| 🪤 **Templates read Go field and method names at runtime** (no compile check): `.BuffIds` (help/species.template in both worlds, descriptions/identify.template), `.CritBuffIds` (identify), `.Buffs` and `.HasBuff` (character/status.template), `.PermaBuff` (character/conditions.template in both worlds). Template functions `buffname`, `buffduration` are registered by name | grep of `_datafiles/world/{dogmud,default}/templates`; `internal/templates/templatesfunctions.go:150,157` |
| Two data worlds carry templates and buff files: `_datafiles/world/dogmud` (109 buff files) and the upstream `_datafiles/world/default` (39) | `ls` |
| The admin command is registered as `` `buff`: {Buff, false, true, true} `` | `internal/usercommands/usercommands.go:80` |
| Messaging categories `CategoryBuffApply`/`CategoryBuffExpire` stringify to `"buff-apply"`/`"buff-expire"` | `internal/messaging/messaging.go:97-98,224-227` |
| Config fields named after their keys: `AllowItemBuffRemoval`, `DeathsShadowBuffId`, `BrokenLimbBuffDuration` (`yaml:"broken_limb_buff_duration"`) | `internal/configs/config.gameplay.go:4,53`; `config.balance.go:214` |
| Hook files: `Buff_ApplyBuffs.go`, `NewTurn_PruneBuffs.go`; root guards: `buff_apply_path_guard_test.go`, `buff_flag_guard_test.go`, `buff_notice_guard_test.go`; plus buff-named tests in hooks, buffs and behaviortree | `git ls-files` |
| Nothing in game scripts or content calls the Go buff names (no `.js` in `_datafiles/world` names them) | grep |
| No collision for the new names in the packages that matter; `bounties.Condition` and `inputhandlers` `Condition` fields live in other packages. The word "conditions" is also used by behaviour tree condition nodes (`internal/behaviortree/conditions_*.go`), quests and web client triggers | grep |
| 26 `context.md` files name old buff symbols | grep |
| `gopls` v0.21.1 is installed at `~/go/bin/gopls` | `gopls version` |

## Name map

**Package**: `internal/buffs` becomes `internal/conditions` (directory, package
clause, import path, `context.md`).

**Rule**: in a Go identifier, `Buff` becomes `Condition` and `buff` becomes
`condition`, keeping the rest of the name. Plural follows (`Buffs` becomes
`Conditions`). Examples:

| Old | New |
|---|---|
| `buffs.Buff`, `buffs.Buffs`, `buffs.BuffSpec` | `conditions.Condition`, `conditions.Conditions`, `conditions.ConditionSpec` |
| `BuffMessage(s)`, `Flag`, `EffectKind`, `Stack` | `ConditionMessage(s)`, unchanged, unchanged, unchanged |
| `BuffIdBleeding` (and the other eight) | `ConditionIdBleeding` |
| `GetBuffSpec`, `GetAllBuffIds`, `SearchBuffs` | `GetConditionSpec`, `GetAllConditionIds`, `SearchConditions` |
| `AddBuff`, `AddBuffScaled`, `AddBuffMagnitude`, `RefreshBuff`, `RemoveBuff`, `HasBuff`, `GetBuffs`, `GetBuffIdsWithFlag`, `HasBuffFlag`, `CancelBuffsWithFlag`, `CancelCombatBuffs` | `AddCondition`, `AddConditionScaled`, `AddConditionMagnitude`, `RefreshCondition`, `RemoveCondition`, `HasCondition`, `GetConditions`, `GetConditionIdsWithFlag`, `HasConditionFlag`, `CancelConditionsWithFlag`, `CancelCombatConditions` |
| `Character.Buffs` | `Character.Conditions` (tag stays `yaml:"buffs,omitempty"`) |
| `Buff.BuffId`, `BuffSpec.BuffId` | `ConditionId` (tags added: `yaml:"buffid"`, see Layer 0) |
| `PermaBuff` | `Permanent` (tag stays `yaml:"permabuff,omitempty"`) |
| `events.Buff`, `events.BuffsTriggered` | `events.Condition`, `events.ConditionsTriggered` |
| `messaging.CategoryBuffApply`, `CategoryBuffExpire` | `CategoryConditionApply`, `CategoryConditionExpire` (strings stay `buff-apply`, `buff-expire`) |
| `SeedBuffsForTest` | `SeedConditionsForTest` |
| hooks `Buff_ApplyBuffs.go` / `ApplyBuffs`, `NewTurn_PruneBuffs.go` / `PruneBuffs` | `Condition_ApplyConditions.go` / `ApplyConditions`, `NewTurn_PruneConditions.go` / `PruneConditions` |
| Other struct fields (`CritBuffIds`, `WornBuffIds`, `TrapBuffIds`, `PrizeBuffIds`, `PlayerBuffIds`, `MobBuffIds`, `NativeBuffIds`, `StartRemoveBuffs`, `ApplyBuff`, `BuffDef`, spell/room/item/mob/pet/species/shop/quest `BuffId(s)`) | `Condition` swap, tags unchanged |

**Not renamed in slice 2** (slice 3, disk/wire): every `yaml:`/`json:` tag
value; untagged-field keys (pinned by tags added in Layer 0); YAML string
values; `effect_type: buff` and its Go `case "buff"` string; the
`melee_self_buff` category and its file name; the `fg="buff"` colour tag;
`_datafiles/world/*/buffs/`; config fields and keys `AllowItemBuffRemoval`,
`DeathsShadowBuffId`, `BrokenLimbBuffDuration` (field and key rename
together in slice 3); GMCP JSON fields; `"buff-apply"`/`"buff-expire"`;
template function names `buffname`/`buffduration`; `docs/schemas/buff.md`
and `_datafiles/guides/building/scripting/SCRIPTING_BUFFS.md` (content
schema docs, renamed with the keys they describe).

**Never renamed**: history. Completed specs and plans under
`docs/superpowers/*/completed/`, past slice specs and plans, patch notes
already published, and commit messages.

**Not buff at all** (the guard's allowlist): `bytes.Buffer` and other
`Buffer`/`buffer` identifiers, `WorldEventBufferSize`, `eventBuffer`,
`connections` `Buffer`.

## Design: the layers

Each layer is one or more plan tasks, compiles and passes the full suite on
its own, and is reviewed before the next.

### Layer 0. Freeze the wire (before any rename)

1. **Pin untagged on-disk fields with explicit tags** carrying today's key:
   `Buff.BuffId` gets `yaml:"buffid"`, `BuffSpec.BuffId` gets `yaml:"buffid"`,
   `species` `BuffIds` gets `yaml:"buffids"`. Check each struct those three
   belong to for any other untagged field that could be renamed later in
   this slice and pin it the same way. `events.*` and `combat.AttackResult`
   are never marshaled (verify in the plan; tag them only if they are).
2. **Wire-freeze test** (root package, one file), green before the rename
   and green after every layer, asserting on literal bytes:
   - a character with a held condition (with stacks) marshals to YAML
     containing `buffs:`, `buffid:`, `permabuff:` where set, `stacks:`; and
     unmarshals back to the same record;
   - the shipped buff YAML files load through the loader (both worlds) and a
     literal buff YAML using `buffid:`, `triggerrate:`, `effects:`,
     `start_remove_buffs:` parses to the expected spec;
   - species YAML `buffids:` still parses;
   - a spell with `effect_type: buff` and `buff_ids:` still applies its
     condition path (the Go `case "buff"` is reached);
   - `messaging.CategoryBuffApply.String()` is `buff-apply` (and the renamed
     constant later still returns it);
   - the GMCP `Char.Conditions` payload and the item/mob GMCP payloads
     marshal with today's JSON field names (`buffIds`, `wornBuffIds`, `buffs`);
   - the rendered `conditions`, `status`, `identify` and species help
     templates produce the same text for a fixed character before and after
     (golden strings captured in Layer 0).
3. Null-probe each assertion.

### Layer 1. The package

`git mv internal/buffs internal/conditions`; package clause; rewrite the
import path everywhere (the path string is unambiguous, so a literal
replacement is exact; the qualifier `buffs.` is then renamed by
`gopls rename` of the package name, or by a literal replacement of the
qualifier limited to Go files that import the package, reviewed). Build and
test.

### Layer 2. Types, constants, functions, methods, fields

`gopls rename` per exported symbol in the name map, then per unexported
symbol that contains the word (local variables and parameters included),
package by package. `gopls rename` follows the type checker, so it never
touches a string literal, a struct tag or an unrelated `Buffer`. Interface
methods (`actions.Actor.AddBuff`) rename with all implementations. Templates
that read renamed fields or methods are updated in the same task as the
field (`.Buffs`, `.HasBuff`, `.BuffIds`, `.CritBuffIds`, `.PermaBuff`), and
the Layer 0 template goldens prove it.

### Layer 3. Files, tests, comments, guards

- Rename Go files and test files whose names carry the word (hooks, root
  guards, buffs tests, behaviortree tests that name the Go concept, not the
  `melee_self_buff` category).
- Rename test functions and fixture helpers.
- Comments and doc comments: every prose "buff" that means the concept
  becomes "condition"; prose that quotes a disk or wire spelling keeps it.
- The root guards' regexes (e.g. the apply-path guard's
  `\.(AddBuff(?:Scaled|Magnitude)?)\(`) move to the new names, keeping their
  compile-time signature pins; allowlist keys move with renamed files.
- The stale comment "Bleeding out = automatic concentration break"
  (`internal/hooks/NewRound_DoCombat_helpers.go:537`) is corrected: the
  check is `IsDisabled()` (`Health <= 0`); there is no bleeding-out state.

### Layer 4. Player- and admin-facing text

- The admin command becomes `setcondition` (owner ruling 2026-09-14, during
  planning: `condition` sat one letter from the player command `conditions`,
  whose aliases are `c`, `cond`, `conds`, in `keywords.yaml`
  `command-aliases`). `buff` stays registered as an alias to the same handler
  until slice 3. Its help template is renamed
  `command.setcondition.template` (both worlds), the admin help list in
  `keywords.yaml` names `setcondition`, and `buff` resolves to the same help
  through the help alias mechanism; the plan reads how aliases resolve help.
- Help templates and in-game text (29 `dogmud` templates, the matching
  `default` ones, Go-side messages and admin/log strings): "buff" becomes
  "condition", or plain English where "condition" reads badly ("rally buffs
  your allies" becomes "rally strengthens your allies"). Loaded with the
  player-copy rules. Quoted wire spellings stay.
- Web client visible text that says "buff" (not identifiers or GMCP fields)
  changes; the `renderguard.js` and HTML comments follow.

### Layer 5. Documentation

- `internal/conditions/context.md` (moved), rewritten to the new names, with
  a short section that separates these conditions from behaviour tree
  condition nodes, quest conditions and web client trigger conditions.
- The 26 `context.md` files that name old symbols; `tools/context_md_audit.py`
  clean for every touched package.
- `.claude/skills/*`, `CLAUDE.md`, and the memory pointers that name live
  symbols (not history).
- `docs/README.md` for renamed indexed files; a patch note only if a player
  sees a change (the admin command is admin-facing; help wording is player
  facing, so one short note).

### Layer 6. The anti-backslide guard

A root test walks non-test and test Go files and fails on any identifier
(Go token of kind IDENT, via `go/scanner`, so string literals, comments and
struct tags are never read) that contains `Buff` or `buff`, except an
allowlist of the "not buff at all" names above and the slice 3 holdouts that
are Go identifiers (the three config fields). Slice 3 deletes the config
entries. It must tolerate a file vanishing mid-walk (skip `os.IsNotExist`),
because behaviortree tests write temporary crate files under a gitignored
`internal/**/_datafiles/` while packages run in parallel (seen 2026-09-14).

## Testing

- Layer 0's freeze test and template goldens stay green from Layer 0 to the
  end; any red is a slice 2 regression by definition.
- Full `go test ./...`, `gofmt`, `go vet`, lint, the web client node tests,
  after every layer.
- The anti-backslide guard, null-probed by reintroducing one `buff`
  identifier.
- Boot check in a detached worktree (both worlds load; zero panics).
- Smoke playtest, single agent, short: `conditions` with a held condition,
  a bleed from a steppe wolf on the `early` profile, an admin applying and
  listing a condition with `setcondition` and with the `buff` alias (admin
  profile), `help` for a spell that used to say "buff". No mechanics
  expectations beyond "unchanged".

## Out of scope

- Every disk/wire spelling listed above (slice 3).
- Any behaviour change.
- The owner product calls from slice 1b (bleed cures, per-weapon Recovering
  text, Thornwall per-round bleed, and the rest).
