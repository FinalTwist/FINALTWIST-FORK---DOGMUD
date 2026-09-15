# Migration Context

## Purpose

`internal/migration` upgrades on-disk data files when the server binary is
newer than the version recorded in config. It runs once at start-up, **before**
the main data-load block, and it rewrites YAML in place — player saves, room
files, and config.

Everything is guarded by a whole-datafiles backup with automatic restore on
failure, because these migrations edit files the game cannot regenerate.

## Files

- **migration.go** — `Run` (entry point) and `doAllMigrations` (the ordered
  version ladder).
- **backup.go** — `datafilesBackup`, `copyDir`, `copyFile`.
- **classify.go** — `PlayerSignals` and `ClassifyPlayer`.
- **grant.go** — `SeedForCluster`.
- **0.9.1.go … 0.17.0.go** — one file per version step, named for the version
  it upgrades *to*.

## Control flow

```go
func Run(lastConfigVersion, serverVersion version.Version) error
```

1. If `lastConfigVersion == serverVersion`, return immediately — no backup, no
   work.
2. Back up the entire datafiles tree; `defer os.RemoveAll(backupFolder)`.
3. Run `doAllMigrations(lastConfigVersion)`.
4. **On error, copy the backup back over datafiles** and return the error.
5. On success, write `Server.CurrentVersion = serverVersion` to config.

`doAllMigrations` is a flat list of `if lastConfigVersion.IsOlderThan(...)`
blocks in ascending order. A server three versions behind runs all three steps
in sequence.

## The version ladder

| To | Migration | What it does |
|----|-----------|--------------|
| 0.9.1  | `migrate_RoomZoneConfig` | introduces per-zone config files |
| 0.10.0 | `migrate_UserStatsRename` | renames stat keys in user saves |
| 0.11.0 | `migrate_RollCharacterStats` | rerolls stats onto the 100-baseline model |
| 0.12.0 | `migrate_RaceToSpecies` | `race:` → `species:` |
| 0.13.0 | `migrate_SeedWarrenRepFromQuestToken` | seeds faction rep from a legacy quest token |
| 0.14.0 | `migrate_ReclassifyPlayerMutations` | wipes retired mutation 41 and reclassifies every save onto the cluster graph |
| 0.15.0 | `migrate_BackfillCoords` | crawls exit deltas to backfill authored x/y/z/plane on every non-instance room |
| 0.16.0 | `migrate_FreezeExploitedVitality` | freezes fyttyn's vitality TOTAL at 280, the soft-cap-compressed value actually in play, rather than handing back the raw 411 |
| 0.17.0 | `migrate_ConditionKeys` | renames buff-spelled keys to their condition spelling in every .yaml and .plugin.dat file under DataFiles, plus the config overrides file |

The newest four (0.14.0 to 0.17.0) take a `dryRun bool` so they can be
exercised without writing.

## Mutation reclassification (0.14.0)

```go
type PlayerSignals struct { /* play-pattern counters read from the save */ }
func ClassifyPlayer(s PlayerSignals) string        // → cluster name
func SeedForCluster(cluster string) map[string]int // → mutation grants
```

`extractSignals` reads raw play signals out of the YAML map, `ClassifyPlayer`
picks the cluster that best fits, and `SeedForCluster` returns the mutations to
grant.

**`SeedForCluster("admin")` grants 11 keystone mutations and freezes drift.**
That is why an admin character is useless for evaluating mutation pacing — it
was seeded, not grown. Any playtest of drift or apex pacing must use a
non-admin character.

## Coordinate backfill (0.15.0)

Walks each rooms directory, groups rooms into connected components by spatial
exits, and assigns coordinates by crawling deltas from an arbitrary origin per
component (`crawlComponent`). Zones marked `non_cartesian` are read from
`loadNonCartesianZones` and skipped for collision purposes.
`writeCoordsPreservingOrder` re-emits the YAML with the new fields inserted
without reordering the rest of the file — important, because these are
hand-authored files people read in diffs.

`countCollisions` reports how many rooms landed on an occupied cell; a non-zero
count means the world was not Cartesian-consistent at migration time.

## Save key renames (0.17.0)

Conditions unification slice 3 renamed every buff-spelled Go yaml tag to its
condition spelling. Every loader ignores unknown keys, so an existing save
that still carries the old key would load with its conditions, pet condition
ids, trapped locks and item condition ids silently empty.
`migrate_ConditionKeys` fixes that by renaming the old keys in place.

**Why the whole DataFiles tree, not path lists.** An item saves its full spec
copy under `overrides:` (`items.Item.Spec`) once it is enchanted, affixed or
renamed, plus `enchantbaseline:`, and `GetSpec()` then never reads the
template again. A missed `wornbuffids` empties that list for good. Items are
saved in character `items`, `componentitems`, `potionitems`, `equipment.*`,
`pet.items` and `companions[].items`; user `itemstorage`; room instance
`items`, `stash` and container items; `mobs.instances` equipment; shop
affixed stock; guild vaults; sealed crates; and auctions plugin data. The
first version walked three path lists and missed most of these.

How it works (`renameConditionKeysUnder`):

- **Which files.** Every regular `*.yaml` or `*.plugin.dat` file under
  DataFiles. A file with no old spelling (`conditionrename.ContainsOldSpelling`:
  `buff` in any case, outside protected words like `buffer`) is skipped
  without parsing, so content files, JSON plugin data (weather) and unrelated
  corrupt files are never parsed, rewritten or reported.
- **Parse.** `decodeOrdered` reads the root kind first, then decodes a mapping
  root as `yaml.MapSlice` and a list of mappings (alts, both `<id>.alts.yaml`
  and legacy `<name>-alts.yaml`) as `[]yaml.MapSlice`, so key order survives.
  Reading the kind first matters: yaml.v2 will decode a list of mappings into
  a `MapSlice` without error and lose the data.
- **Rename.** `renameKeys` recurses into every mapping and list. Each key in
  `renamedKeys` (the complete set of distinctive buff-spelled save, plugin and
  config keys) becomes `conditionrename.Apply(key)` wherever it sits. The
  generic `buffs` becomes `conditions` only when its value is a mapping with
  a `list` key, the conditions record shape. Values are never touched.
- **Write.** Only when something was renamed, with `yaml.Marshal`. One log line
  per rewritten file and a final scanned / parsed / rewritten count. A dry run
  logs and writes nothing.
- **Config overrides.** When `CONFIG_PATH` points outside DataFiles that file
  is migrated too. Whenever the overrides file is rewritten,
  `configs.ReloadConfig` runs: config was loaded before migrations, and `Run`'s
  closing `SetVal` marshals the in-memory overrides map back to disk, which
  would otherwise restore `BuffsEnabled`.

There is no migration marker. A file with no old key is left byte-for-byte
unchanged, so a second run is a no-op and there is nothing to desync per-alt.
These are hard errors, so `Run` restores the backup rather than shipping a
broken save: a file with an old spelling that fails to parse; a mapping with
both an old key and its new name (a collision); and a list root with
non-mapping elements that carries an old key (no store saves that shape, and
it cannot be rewritten order-preserving).

## Gotchas

- **Migrations run before data loading.** Nothing is in memory yet — no rooms,
  no mobs, no factions. The 0.13.0 step has to call
  `factions.LoadAllDefinitions()` itself for exactly this reason, and any new
  migration that needs loaded data must do the same.
- **The backup is deleted by `defer` even on the error path** — but only after
  the restore has already run, so ordering is correct. Do not add an early
  return between the restore and the deferred cleanup.
- **A partially-applied migration is restored wholesale, not rolled back
  step-by-step.** Any migration that writes outside the datafiles tree escapes
  the safety net.
- **Version comparison is `IsOlderThan`, not equality.** Steps are cumulative;
  never write a migration that assumes the previous one just ran in this
  process.
- **`Run` only writes `Server.CurrentVersion` on full success.** A failed
  migration leaves the version untouched, so the next boot retries from the
  same point.

## Dependencies

`configs`, `version`, `factions` (0.13.0 only), `conditionrename` and `mudlog`
(0.17.0 only), plus direct YAML and filesystem access. Deliberately minimal —
this code must work before the engine is up.

## Consumers

`main.go` only, at start-up.
