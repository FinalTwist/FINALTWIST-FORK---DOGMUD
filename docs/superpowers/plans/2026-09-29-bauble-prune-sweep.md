# Bauble Catalog Prune Sweep Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the bauble catalog's time-only prune with an owner-ruled periodic sweep that collects every bauble id any item still points at (live world and every save file) and prunes only the records nothing has pointed at for two sweeps and the whole keep window.

**Architecture:** Every store of items gets a `WalkItems(fn func(*items.Item))` method. The main package registers live walks (users, rooms, mobs, shops, guilds) and the auction module registers its own, with `baubles.RegisterLiveSource`. A sweeper goroutine runs at boot and every `Balance.BaubleSweepHours`: it walks the live world under the mud lock, reads every `.yaml` and `.plugin.dat` under DataFiles off the lock (skipping `baubles/` and `economy/`), then updates `LastSeenAt` / `UnseenSweeps` on each record and writes each changed shard without its prunable records before they leave memory. Any error anywhere fails the sweep closed. Two repo-root guards fail when a store of items is not walked.

**Tech Stack:** Go 1.25.7, `gopkg.in/yaml.v3` (`yaml.Node` walk), `reflect` + `go/ast` in the guards, PowerShell for the boot smoke, Git Bash for git.

**Branch / worktree:** `feature/bauble-prune-sweep` in `C:/tmp/dogmud-bauble-sweep`, cut from `origin/master` at `3bd6ccaa3` (Merge PR #184). All paths below are relative to that worktree. Every task commits on this branch. Do not push until Task 14.

---

## Facts verified against source (2026-09-29, `3bd6ccaa3`)

| # | Fact | Where |
|---|------|-------|
| 1 | `Prune(now)` / `prunable` / `goneAt` exist; callers are `loadFrom` (`catalog.go:115`), `SaveAll` (`catalog.go:266`) and `prune_test.go:56` only (repo grep `Prune(` for baubles) | `internal/baubles/catalog.go:338,354,367` |
| 2 | Today's rule: prunable iff `goneAt` (later of `SoldAt`, `VanishedAt`) is set, `ReturnCreditAt` is zero, and `now - gone >= keep`. A record never sold or vanished is kept forever | `catalog.go:354-361` |
| 3 | `persistShard` snapshots under `mu.RLock`, writes through the `shardWriter` var outside `mu`, ordered by `writeMu`; a failed write marks `dirty[shard]` for `SaveAll` | `catalog.go:274-304` |
| 4 | Every record is sellable, a sold one included; the sale only refuses an UNKNOWN record (`baubleSayUnknown`) | `internal/baubles/sales.go:12-15`, `internal/actions/sell_bauble.go:109-111` |
| 5 | `MarkSold` sets `Status=sold`, `SoldAt=now`, `SoldValue`; `StatusSold` is read only by `CatalogStats`, `Restore`, `SalesSince`, and `bauble show/stats` | `sales.go:20-30`, repo grep `StatusSold` |
| 6 | `internal/baubles/context.md:310` names a `Sellable` consumer symbol; no such function exists anywhere (grep `func Sellable` finds nothing) | context.md |
| 7 | `BaubleCatalogKeepDays` default 30, floor 7 (`config.balance.baubles.go:46-47,226-231`), declared `config.balance.go:940`, shipped `30` (`git show HEAD:_datafiles/config.yaml`, line 1536) | configs |
| 8 | `BaubleUntakenHours` default 24, shipped 24 (config.yaml:1476); `UntakenLimit()` | `internal/baubles/theft.go:82` |
| 9 | Item keys: `Bauble` yaml `bauble`, `BaubleLeftAt` yaml `baubleleftat` (Unix seconds). `yaml:"bauble"` is the only such tag in the repo | `internal/items/items.go:65,68` |
| 10 | `items.Item` holds no nested items (its fields are scalars, `*ItemSpec`, `*SpecBaseline`, a base64 `Blob` string and an unexported `map[string]any`) | `items.go` `type Item struct` |
| 11 | `(*Item).BaubleUntakenFor(now)` returns `(age, true)` only for a bauble with `BaubleLeftAt > 0` | `internal/items/bauble_placement.go:44` |
| 12 | `removeUntakenBaubles` runs only from `Room.Prepare` (`rooms.go:853`) and `RoundTick` (`rooms.go:2694`). `LoadRoom`/`LoadRoomInstance` (`save_and_load.go:79,124`) never prepare; `LoadRoom` is `LoadRoomInstance`'s only caller | `internal/rooms` |
| 13 | AST scan (every top-level struct in non-test `.go` files with a field whose type mentions `items.Item`) finds 34 types; list in the Store list below | scratch AST scan |
| 14 | `Character`: `Items`, `ComponentItems`, `PotionItems` (`character.go:142-144`), `Equipment Worn` (`:146`), `Pet pets.Pet` (`:328`, `Pet.Items` `pets.go:29`), `Companions []CompanionInfo` (`:329`; `CompanionInfo.Items`/`Equipment` `companions.go:82-83`). `Worn.AllSlots()` is the single slot list, guarded by `worn_allslots_test.go` | `internal/characters` |
| 15 | `UserRecord`: `Character *characters.Character` (`:45`), `ItemStorage Storage` (`:46`; `Slots[].Item` and legacy `Items`, `AllItemPtrs` `storage.go:156`), `Inbox` (`:48`; `Message.Item *items.Item` `inbox.go:16`) | `internal/users` |
| 16 | `Room`: `Items`, `Stash` (`rooms.go:113-114`), `Containers map[string]Container` (`:107`, `Container.Items` `container.go:10`), `Corpses []Corpse` `yaml:"-"` (`:115`; `Corpse.Character`, `Corpse.Loot Container`), `SealedCrate *sealedcrate.Crate` `yaml:"-"` (`:125`; unexported `items` guarded by `mu`) | `internal/rooms` |
| 17 | `Mob.Character` (`mobs.go:118`). A mob instance file persists only `Equipment` (`MobInstanceData`, `instance_save.go:22-54`): a mob's pack exists only in memory | `internal/mobs` |
| 18 | `ShopInventory.AffixedStock []AffixedStockEntry` (`shopinventory.go:93`); `Guild.Vault` (`guilds.go:53`); `AuctionManager.ActiveAuction *AuctionItem` (`auctions.go:748`), `SeizedQueue []SeizedLot` (`:752`); `PastAuctionItem` keeps names only (`:782`) | shops, guilds, auctions |
| 19 | Registries: `users.GetAllActiveUsers` SKIPS zombies (`users.go:132-144`); `roomManager.rooms` has no exported iterator (`LoadedRoomCount` `roommanager.go:719`); `mobs.GetAllMobInstanceIds` + `GetInstance` (`mobs.go:818,809`); `shops.AllShops` (`persistence.go:229`); `guilds.All` (`registry.go:79`) | |
| 20 | `roomManager.rooms` can be written while only `RLockMud` is held (`GetAutoComplete` -> `LoadRoom`, `roommanager.go:52-60`), so the live walk must take the full `util.LockMud` | |
| 21 | `util.LockMud` / `UnlockMud` | `internal/util/util.go:92,96` |
| 22 | Copyover runs holding the mud lock (`copyover.go:33-48`), so it must not wait for a sweep that may be waiting on that lock | |
| 23 | Shutdown calls `baubles.SaveAll()` at `main.go:611`, before `close(workerShutdownChan)` at `main.go:633`; MainWorker is still ticking then | `main.go` |
| 24 | Background ticker precedent: the economy snapshot goroutine | `main.go:530-564` |
| 25 | Every item store marshals with `gopkg.in/yaml.v2` (`users.go:24`, `save_and_load.go:18`, `shops/persistence.go:16`, `guilds/persistence.go:13`, `mobs/instance_save.go:16`, `plugins.go:21`) | |
| 26 | Plugin data lives at `<DataFiles>/plugin-data/<name>-v<ver>/<id>.plugin.dat` (`plugins.go:519,402-407`); the auction house's id is `auctionhistory` (`auctions.go:135`) | |
| 27 | Sealed crates: `<DataFiles>/crates/*.yaml` (`main.go:1926`), on-disk shape `cratePayload` (`sealedcrate/persistence.go:13`) | |
| 28 | Precedent for the disk half: migration 0.17.0 scans every `.yaml` and `.plugin.dat` under DataFiles with a cheap prefilter, because "a path list missed most of them" | `internal/migration/0.17.0.go:27-33,137-183` |
| 29 | `users.SearchOfflineUsers` skips `.alts.yaml` and discards `filepath.Walk`'s error, so it cannot fail closed and is not reused | `users.go:593-640` |
| 30 | Quarantined files are `<path>.corrupt-<stamp>` (`livingstate.go:100`); `util.SafeSave` writes `<path>.new` (`util.go:710`). Neither ends in `.yaml` or `.plugin.dat` | `internal/util` |
| 31 | Local DataFiles: 5,295 `.yaml`/`.plugin.dat` files, 58 MB; `economy/` is 392 metric snapshot files (41 MB, shop stock counts, no items); excluding `economy/` and `baubles/`: 4,903 files, 8.7 MB read; users: 105 saves + 1 alts file | measured |
| 32 | A Go read-plus-regex pass over those 4,903 files took 0.57 to 0.69 s warm on this Windows machine | scratch timing program |
| 33 | `internal/baubles` has a `TestMain` that sets up the logger (`catalog_test.go:17`); the repo root package has no `TestMain`, so root tests must not call code that logs | |
| 34 | Test helpers: `setBaubleConfig` (`find_test.go:12`), `withCatalog` (`catalog_test.go:28`); rooms: `seedRegistry` (`rooms_test.go:16`) and the `LoadRoomInstance` file fixture pattern (`save_and_load_test.go:73-165`); actions: `seedBaubleSale`, `newBauble`, `seedSellRoom`, `seedSellMerchant`, `newSellerActor` (`sell_bauble_test.go`) | |
| 35 | `items.ItemDisabledSlot` is `ItemId: -1` (`items.go:25`); `Worn.GetAllItemPtrs` walks `ItemId > 0` (`worn.go:171`) | |
| 36 | `characters.MigrateDetunedRangedWeapons` hand-lists Items, ComponentItems, PotionItems, Pet.Items and Equipment and does NOT reach companions' gear | `internal/characters/migrate_detuned_bows.go:63-86` |
| 37 | In this worktree `git ls-files -v _datafiles/config.yaml` prints `H` (no skip-worktree bit), so `config.yaml` is edited and committed normally here. The main checkout's copy carries `S` | |
| 38 | `golangci-lint` 2.12.2 at `~/go/bin/golangci-lint`; `compose.test.yml` service `test` | |
| 39 | New names are unused: grep for `WalkItems`, `WalkSlice`, `LoadedRooms`, `GetAllLoadedUsers`, `RegisterLiveSource`, `DiskRefs`, `BaubleSweepHours` finds nothing | |
| 40 | Module path `github.com/GoMudEngine/GoMud` | `go.mod:1` |

## Every place an item can live (the store list)

Live (in memory, walked under the mud lock):

| Store | Root | Reached through |
|---|---|---|
| Online and link-dead players: backpack, component bag, potion bandolier, 26 equipment slots, pet pack, each companion's saved pack and gear, bank slots and legacy bank list, inbox attachments | `users.UserRecord` | `GetAllLoadedUsers` (new, includes zombies) |
| Loaded rooms (ephemeral included): floor, stash, every container, every corpse (the dead character's gear and its loot container), the sealed crate | `rooms.Room` | `LoadedRooms` (new) |
| Every live mob: its character's gear and pack (packs are never saved: fact 17), charmed companions | `mobs.Mob` | `GetAllMobInstanceIds` + `GetInstance` |
| Shops: unique resale stock (`AffixedStock`) | `shops.ShopInventory` | `AllShops` |
| Guild vaults | `guilds.Guild` | `All` |
| Auction house: the lot on the block and seized lots | `auctions.AuctionManager` | registered by the module |

On disk (read off the lock, whole DataFiles tree minus `baubles/` and `economy/`): `users/<id>.yaml` (character, bank, inbox), `users/<id>.alts.yaml` (offline alts), `rooms.instances/<zone>/<id>.yaml` (floors, stashes, containers of rooms not loaded), `mobs.instances/**` (mob equipment), `shops/<zone>/<mob>-room<room>.yaml` (affixed stock), `guilds/<tag>.yaml` (vaults), `crates/<room>-<label>.yaml` (crates whose room is missing or not yet attached), `plugin-data/auctions-v1.0/auctionhistory.plugin.dat` (auction house between saves), and any future store that writes items under DataFiles.

Transient (alive for one call or one tick; listed with reasons in `transientItemHolders`, Task 4): `actions.DropItemResult`, `EquipItemResult`, `GetItemResult`, `GiveItemResult`, `RemoveEquipResult`, `StealOptions`; `characters.HandSlot`, `WornSlot` (views into `Worn`); `combat.DisarmResult`, `weaponSetup`; `events.EquipmentChange`, `ItemOwnership`, `StorageItemSeized`; `hooks.WeaponBreakResult`, `plannedSeizure`; `itemvalue.SwapDelta`; `parser.Match`; `sealedcrate.cratePayload` (disk shape); `usercommands.enchantSlotCandidate`; `modules/aicompanion.thing`. A find being delivered is minted and handed over inside one lock hold (`search_bauble.go:273-320`), so it is never in flight between stores. Checked and holding no items: warehouses (`warehouse.Entry` is `ItemId` + count), caravans, ferries, `economy/snapshots`, `PastAuctionItem`.

Nesting: items do not nest (fact 10). Containers exist only as room containers and corpse loot, both reached by `Room.WalkItems`.

## Decisions, with reasons

1. **When and where it runs.** A dedicated sweeper goroutine started from `main.go` just before `Server Ready`: one sweep at once, then every `BaubleSweepHours` (new knob, default 6, floor 1). Not at `SaveAll`: copyover calls `SaveAll` holding the mud lock (fact 22). Shutdown calls `StopSweeper()` before `SaveAll()` so a sweep in progress finishes its writes (bounded wait of 30 s); copyover does not wait (deadlock), which is safe because every catalog write is atomic.
2. **Locks.** Live half: full `util.LockMud` (fact 20), walking only in-memory values, then released. Disk half: no mud lock, no catalog lock; plain file reads. Apply half: catalog `mu` for the field updates, then per-shard `persistShardPruning` under `writeMu`. The catalog lock is never held across a disk write, as today.
3. **Two-phase, and why a record needs two unseen sweeps AND the keep window.** A record gets `LastSeenAt=now, UnseenSweeps=0` when referenced, otherwise `UnseenSweeps++` (capped at 2). It is prunable when `UnseenSweeps >= 2` and `now - lastEvidence >= keep`, where `lastEvidence` is the latest of `FoundAt`, `StolenAt`, `RecognizedAt`, `ReturnedAt`, `SoldAt`, `VanishedAt`, `LastSeenAt`. An item moving between stores while one sweep looks can be missed once; nothing is lost unless that repeats for the whole window. Two sweeps also stop a single boot sweep after a long downtime from pruning on one look. A store the sweep NEVER sees would repeat forever; the guards (Tasks 2 and 4) are what prevent that. `ReturnCreditAt` records are kept forever, as today. Records found after the sweep began are left alone.
4. **The only pruner is a successful sweep.** `Load` and `SaveAll` stop pruning. A time-only prune would delete a rolled-back sold record whenever sweeps keep failing closed.
5. **Rolled-back sold baubles.** A sold record that an item points at again is simply seen, so it survives; its `Status`/`SoldAt` are left as they were. It is already sellable (fact 4), and a re-sale overwrites `SoldAt`. Rewriting the sale on sweep evidence would be wrong: a user save on disk lags a sale by up to one autosave, so a sweep in that window would "revive" and erase a real sale.
6. **Fail closed.** No live source registered, a live source that panics, a data folder that cannot be listed, a file that cannot be read (other than one deleted since the listing), a file that names a bauble and does not parse, or a document nested deeper than 100 levels: the sweep records the error, logs at ERROR, and applies NOTHING (no `LastSeenAt`, no `UnseenSweeps`, no prune).
7. **Disk scope.** Whole DataFiles, like migration 0.17.0 (fact 28), because a folder list rots the day someone adds a store. Skipped: `baubles/` (the catalog itself) and `economy/` (metric snapshots, 70% of the bytes, no items). A file is parsed only if it contains a `bauble` key (bare or JSON-quoted).
8. **Untaken finds in never-revisited rooms.** In `rooms.instances/` only, a find with `baubleleftat` older than `UntakenLimit()` is not counted as a reference (the owner's gap "room never revisited"). To make that safe, `LoadRoomInstance` now removes such finds (and marks them vanished) whenever a room is loaded from its instance file, so a pruned find can never be picked up by a mob that wanders into an unprepared room. The live walk counts everything it sees.
9. **Persistence.** `util.Save` through the existing `writeShard`. Persist before publish for removals: a shard is written without its prunable records first, and only after that write succeeds are they taken out of memory, still under `writeMu`; a record that changed meanwhile and is no longer prunable is kept and the shard marked dirty. `LastSeenAt`/`UnseenSweeps` follow the catalog's write-through rule (set, then written); losing one of those writes only makes a record look seen slightly earlier, which the keep window absorbs.
10. **Admin visibility.** `bauble status` gains one `Sweep:` line (never run / failed with its error / catalog empty / counts and timings). Every sweep logs one INFO line (`action=sweep records referenced pruned files parsed live disk`), and a failed one an ERROR line.
11. **Scope widened (flaw found):** `MigrateDetunedRangedWeapons` misses companions' gear (fact 36). It now walks `Character.WalkItems` (Task 3).

## Cost estimate

- Disk half, off the lock: about 4,900 files and 9 MB read per sweep locally, 0.6 to 0.7 s warm on Windows (fact 32); the droplet's Linux file reads are cheaper per file. It grows by about 7 KB per user save. Every 6 hours, that is under 3 s of background I/O a day.
- Live half, under the mud lock: a walk over loaded users, loaded rooms (hundreds), live mobs (hundreds to low thousands) and shops, each character about 40 slot checks: on the order of 100,000 pointer checks, estimated at 1 to 5 ms. The boot smoke (Task 13) records the real `live=` figure; if it exceeds 50 ms, stop and report before shipping.
- Apply half: one shard write (500 records) per shard that holds a changed record. With every held record's `LastSeenAt` refreshed, that is every shard holding a held bauble, each sweep: at 10,000 records about 20 writes of a few hundred KB, outside the catalog lock.
- An empty catalog skips the sweep entirely (no disk read).

## File structure

| File | Change | Responsibility |
|---|---|---|
| `internal/items/walk.go` | create | `WalkSlice`, the one slice walker every store uses |
| `internal/characters/walk_items.go` | create | `(*Worn).WalkItems`, `(*Character).WalkItems` |
| `internal/users/walk_items.go` | create | `(*UserRecord).WalkItems`, `GetAllLoadedUsers` |
| `internal/rooms/walk_items.go` | create | `(*Room).WalkItems`, `LoadedRooms` |
| `internal/sealedcrate/sealedcrate.go` | modify | `(*Crate).WalkItems` |
| `internal/mobs/walk_items.go` | create | `(*Mob).WalkItems` |
| `internal/shops/walk_items.go` | create | `(*ShopInventory).WalkItems` |
| `internal/guilds/walk_items.go` | create | `(*Guild).WalkItems` |
| `modules/auctions/walk_items.go` | create | `(*AuctionManager).WalkItems` |
| `modules/auctions/auctions.go` | modify | register the `auctions` live source in `init` |
| `internal/characters/migrate_detuned_bows.go` | modify | walk through `Character.WalkItems` |
| `internal/configs/config.balance.go`, `config.balance.baubles.go`, `_datafiles/config.yaml` | modify | `BaubleSweepHours` knob |
| `internal/baubles/record.go` | modify | `LastSeenAt`, `UnseenSweeps` |
| `internal/baubles/catalog.go` | modify | drop `Prune`, add `lastEvidence`, `prunableAt`, `persistShardPruning` |
| `internal/baubles/sweep.go` | create | registry, `applySweep`, `runSweep`, status, sweeper loop |
| `internal/baubles/sweep_disk.go` | create | `DiskRefs`, the DataFiles scan |
| `internal/rooms/save_and_load.go` | modify | remove expired untaken finds on instance load |
| `bauble_sweep.go` (repo root) | create | `registerBaubleSweepSources` |
| `main.go`, `copyover.go` | modify | start/stop the sweeper; copyover comment |
| `internal/usercommands/admin.bauble.go` | modify | `Sweep:` line in `bauble status` |
| `item_walker_guard_test.go`, `bauble_sweep_guard_test.go`, `bauble_sweep_test.go` (repo root) | create | the two guards, registration and real-type disk tests |
| tests in each package | create/modify | as listed per task |
| `internal/baubles/context.md` and eight other `context.md` files, `docs/PATCH_NOTES.md`, `docs/README.md` | modify | docs |

---

### Task 1: `items.WalkSlice`

**Files:**
- Create: `internal/items/walk.go`
- Test: `internal/items/walk_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/items/walk_test.go`:

```go
package items

import "testing"

// WalkSlice visits real items only (an empty or disabled slot is skipped),
// in order, through pointers into the slice itself.
func TestWalkSliceVisitsRealItemsInPlace(t *testing.T) {
	s := []Item{{ItemId: 5}, {}, ItemDisabledSlot, {ItemId: 7}}
	seen := []int{}
	WalkSlice(s, func(it *Item) {
		seen = append(seen, it.ItemId)
		it.Uses = 3
	})
	if len(seen) != 2 || seen[0] != 5 || seen[1] != 7 {
		t.Fatalf("visited %v, want [5 7]", seen)
	}
	if s[0].Uses != 3 || s[3].Uses != 3 {
		t.Fatal("the pointers are into the slice itself")
	}
	WalkSlice(nil, func(*Item) { t.Fatal("nothing to visit in nil") })
}
```

- [ ] **Step 2: Run it to see it fail**

Run: `go test ./internal/items/ -run TestWalkSliceVisitsRealItemsInPlace`
Expected: FAIL to compile, `undefined: WalkSlice`.

- [ ] **Step 3: Implement**

Create `internal/items/walk.go`:

```go
package items

// WalkSlice calls fn with a pointer to each real item in s (ItemId above
// zero: an empty slot or ItemDisabledSlot is skipped), in order. The
// pointers are into s itself, so fn may change an item in place. Every
// store's item walker (Character.WalkItems, Room.WalkItems and the rest)
// is built on it; the bauble catalog sweep reads every live item through
// them.
func WalkSlice(s []Item, fn func(*Item)) {
	for i := range s {
		if s[i].ItemId > 0 {
			fn(&s[i])
		}
	}
}
```

- [ ] **Step 4: Run it to see it pass**

Run: `go test ./internal/items/ -run TestWalkSliceVisitsRealItemsInPlace`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git -C C:/tmp/dogmud-bauble-sweep add internal/items/walk.go internal/items/walk_test.go
git -C C:/tmp/dogmud-bauble-sweep commit -m "feat(items): WalkSlice, the shared item slice walker

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: A `WalkItems` for every store, and the field-planting guard

**Files:**
- Create: `item_walker_guard_test.go` (repo root)
- Create: `internal/characters/walk_items.go`, `internal/users/walk_items.go`, `internal/users/walk_items_test.go`, `internal/rooms/walk_items.go`, `internal/mobs/walk_items.go`, `internal/shops/walk_items.go`, `internal/guilds/walk_items.go`, `modules/auctions/walk_items.go`
- Modify: `internal/sealedcrate/sealedcrate.go`, `internal/sealedcrate/sealedcrate_test.go`

- [ ] **Step 1: Write the failing guard**

Create `item_walker_guard_test.go`:

```go
package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unsafe"

	"github.com/GoMudEngine/GoMud/internal/guilds"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/modules/auctions"
)

// The bauble catalog sweep (internal/baubles/sweep.go) prunes a record once
// nothing in the world points at it, and it finds live items only through
// these roots and their WalkItems methods. A store it misses loses its
// baubles' names. Two guards keep that complete:
//
//   - TestItemWalkersVisitEveryItemField (here) builds each root with an
//     item planted in EVERY items.Item field reachable from it by
//     reflection, unexported ones included, and fails naming the path of
//     any planted item WalkItems does not visit.
//   - TestEveryItemHolderIsASweepRootOrTransient (bauble_sweep_guard_test.go)
//     finds every struct in the repo that holds an items.Item and fails for
//     one no root reaches, unless it is listed as transient with a reason.
//
// Known limit: an item held behind an interface (any) is invisible to both.
// None is today: Mob.BTreeState, Room.LongTermDataStore and the
// tempDataStore maps hold no items.

type sweepRoot struct {
	name string
	typ  reflect.Type
	walk func(root reflect.Value, fn func(*items.Item))
}

// sweepRoots are the live stores registerBaubleSweepSources (bauble_sweep.go)
// and modules/auctions register, one per source name.
func sweepRoots() []sweepRoot {
	return []sweepRoot{
		{`auctions`, reflect.TypeOf((*auctions.AuctionManager)(nil)).Elem(), func(v reflect.Value, fn func(*items.Item)) {
			v.Addr().Interface().(*auctions.AuctionManager).WalkItems(fn)
		}},
		{`guilds`, reflect.TypeOf((*guilds.Guild)(nil)).Elem(), func(v reflect.Value, fn func(*items.Item)) {
			v.Addr().Interface().(*guilds.Guild).WalkItems(fn)
		}},
		{`mobs`, reflect.TypeOf((*mobs.Mob)(nil)).Elem(), func(v reflect.Value, fn func(*items.Item)) {
			v.Addr().Interface().(*mobs.Mob).WalkItems(fn)
		}},
		{`rooms`, reflect.TypeOf((*rooms.Room)(nil)).Elem(), func(v reflect.Value, fn func(*items.Item)) {
			v.Addr().Interface().(*rooms.Room).WalkItems(fn)
		}},
		{`shops`, reflect.TypeOf((*shops.ShopInventory)(nil)).Elem(), func(v reflect.Value, fn func(*items.Item)) {
			v.Addr().Interface().(*shops.ShopInventory).WalkItems(fn)
		}},
		{`users`, reflect.TypeOf((*users.UserRecord)(nil)).Elem(), func(v reflect.Value, fn func(*items.Item)) {
			v.Addr().Interface().(*users.UserRecord).WalkItems(fn)
		}},
	}
}

var walkGuardItemType = reflect.TypeOf(items.Item{})

var walkGuardReach = map[reflect.Type]bool{}

// walkGuardReaches reports whether a value of type t can hold an items.Item
// anywhere inside it (through struct fields, pointers, slices, arrays and
// map keys or values; not through interfaces).
func walkGuardReaches(t reflect.Type) bool {
	if r, ok := walkGuardReach[t]; ok {
		return r
	}
	r := walkGuardSearch(t, map[reflect.Type]bool{})
	walkGuardReach[t] = r
	return r
}

func walkGuardSearch(t reflect.Type, seen map[reflect.Type]bool) bool {
	if t == walkGuardItemType {
		return true
	}
	if seen[t] {
		return false
	}
	seen[t] = true
	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			if walkGuardSearch(t.Field(i).Type, seen) {
				return true
			}
		}
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return walkGuardSearch(t.Elem(), seen)
	case reflect.Map:
		return walkGuardSearch(t.Key(), seen) || walkGuardSearch(t.Elem(), seen)
	}
	return false
}

// walkGuardPlanter puts a distinct bauble in every items.Item field it can
// reach, allocating pointers, one-element slices and one-entry maps on the
// way. A type may appear at most twice on one path, so a recursive type is
// planted one level deep instead of forever.
type walkGuardPlanter struct {
	next    int
	planted map[string]string // bauble id -> field path
	onPath  map[reflect.Type]int
}

func (p *walkGuardPlanter) plant(v reflect.Value, path string) {
	t := v.Type()
	if !walkGuardReaches(t) || p.onPath[t] >= 2 {
		return
	}
	v = walkGuardSettable(v)
	if t == walkGuardItemType {
		p.next++
		id := fmt.Sprintf(`B%07d`, p.next)
		v.Set(reflect.ValueOf(items.Item{ItemId: items.BaubleItemId, Bauble: id}))
		p.planted[id] = path
		return
	}
	p.onPath[t]++
	defer func() { p.onPath[t]-- }()
	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			p.plant(v.Field(i), path+`.`+t.Field(i).Name)
		}
	case reflect.Pointer:
		if v.IsNil() {
			v.Set(reflect.New(t.Elem()))
		}
		p.plant(v.Elem(), path)
	case reflect.Slice:
		v.Set(reflect.MakeSlice(t, 1, 1))
		p.plant(v.Index(0), path+`[0]`)
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			p.plant(v.Index(i), fmt.Sprintf(`%s[%d]`, path, i))
		}
	case reflect.Map:
		elem := reflect.New(t.Elem()).Elem()
		p.plant(elem, path+`[k]`)
		m := reflect.MakeMap(t)
		m.SetMapIndex(reflect.New(t.Key()).Elem(), elem)
		v.Set(m)
	}
}

// walkGuardSettable makes an unexported field settable. Test-only.
func walkGuardSettable(v reflect.Value) reflect.Value {
	if v.CanSet() {
		return v
	}
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}

func TestItemWalkersVisitEveryItemField(t *testing.T) {
	for _, root := range sweepRoots() {
		t.Run(root.name, func(t *testing.T) {
			p := &walkGuardPlanter{planted: map[string]string{}, onPath: map[reflect.Type]int{}}
			v := reflect.New(root.typ).Elem()
			p.plant(v, root.typ.Name())
			if len(p.planted) == 0 {
				t.Fatalf("planted nothing in %s: the guard cannot fail", root.typ)
			}
			seen := map[string]bool{}
			root.walk(v, func(it *items.Item) { seen[it.Bauble] = true })
			missing := []string{}
			for id, path := range p.planted {
				if !seen[id] {
					missing = append(missing, path)
				}
			}
			sort.Strings(missing)
			if len(missing) > 0 {
				t.Errorf("%s.WalkItems misses %d item field(s); walk them, or the bauble sweep prunes the records of what they hold:\n  %s",
					root.typ, len(missing), strings.Join(missing, "\n  "))
			}
		})
	}
}
```

- [ ] **Step 2: Run it to see it fail**

Run: `go test . -run TestItemWalkersVisitEveryItemField`
Expected: FAIL to compile: `...WalkItems undefined` for each of the six root types.

- [ ] **Step 3: Implement the walkers**

Create `internal/characters/walk_items.go`:

```go
package characters

import "github.com/GoMudEngine/GoMud/internal/items"

// WalkItems calls fn with a pointer to the item in every filled equipment
// slot, in AllSlots order. The pointers are the real slot fields.
func (w *Worn) WalkItems(fn func(*items.Item)) {
	for _, s := range w.AllSlots() {
		if s.Item.ItemId > 0 {
			fn(s.Item)
		}
	}
}

// WalkItems calls fn with a pointer to every item this character holds:
// backpack, component bag, potion bandolier, every equipment slot, the
// pet's pack, and each companion's saved pack and gear. The pointers are
// live, so a migration may change an item in place.
//
// The bauble catalog sweep reads every character in the world through
// this; an item it does not see can have its record pruned.
// TestItemWalkersVisitEveryItemField (repo root) plants an item in every
// items.Item field reachable from a character and fails naming any this
// does not walk.
func (c *Character) WalkItems(fn func(*items.Item)) {
	items.WalkSlice(c.Items, fn)
	items.WalkSlice(c.ComponentItems, fn)
	items.WalkSlice(c.PotionItems, fn)
	c.Equipment.WalkItems(fn)
	items.WalkSlice(c.Pet.Items, fn)
	for i := range c.Companions {
		items.WalkSlice(c.Companions[i].Items, fn)
		c.Companions[i].Equipment.WalkItems(fn)
	}
}
```

Create `internal/users/walk_items.go`:

```go
package users

import "github.com/GoMudEngine/GoMud/internal/items"

// WalkItems calls fn with a pointer to every item this account holds: its
// active character's (Character.WalkItems), the bank (the slots and the
// legacy list), and any item waiting in the inbox. Alts are not in memory:
// they live only in <userid>.alts.yaml, which the bauble sweep reads from
// disk.
func (u *UserRecord) WalkItems(fn func(*items.Item)) {
	if u.Character != nil {
		u.Character.WalkItems(fn)
	}
	for _, it := range u.ItemStorage.AllItemPtrs() {
		if it.ItemId > 0 {
			fn(it)
		}
	}
	for i := range u.Inbox {
		if it := u.Inbox[i].Item; it != nil && it.ItemId > 0 {
			fn(it)
		}
	}
}

// GetAllLoadedUsers returns every user record in memory, link-dead
// (zombie) users included: their characters are still in the world.
// GetAllActiveUsers leaves zombies out.
func GetAllLoadedUsers() []*UserRecord {
	userManager.mu.RLock()
	defer userManager.mu.RUnlock()

	ret := make([]*UserRecord, 0, len(userManager.Users))
	for _, u := range userManager.Users {
		ret = append(ret, u)
	}
	return ret
}
```

Create `internal/users/walk_items_test.go`:

```go
package users

import "testing"

// A link-dead player's character is still in the world, so the bauble
// sweep must see it: GetAllLoadedUsers includes zombies.
func TestGetAllLoadedUsersIncludesZombies(t *testing.T) {
	userManager.mu.Lock()
	userManager.Users[990001] = &UserRecord{UserId: 990001}
	userManager.Users[990002] = &UserRecord{UserId: 990002, isZombie: true}
	userManager.mu.Unlock()
	t.Cleanup(func() {
		userManager.mu.Lock()
		delete(userManager.Users, 990001)
		delete(userManager.Users, 990002)
		userManager.mu.Unlock()
	})

	found := map[int]bool{}
	for _, u := range GetAllLoadedUsers() {
		found[u.UserId] = true
	}
	if !found[990001] || !found[990002] {
		t.Fatalf("loaded users %v, want both the active and the zombie", found)
	}
	for _, u := range GetAllActiveUsers() {
		if u.UserId == 990002 {
			t.Fatal("GetAllActiveUsers still skips zombies")
		}
	}
}
```

Create `internal/rooms/walk_items.go`:

```go
package rooms

import "github.com/GoMudEngine/GoMud/internal/items"

// WalkItems calls fn with a pointer to every item in the room: on the
// floor, in the stash, in each container, on each corpse (the dead
// character's own gear and the loot beside it), and in the sealed crate.
// A container's items are reached through the map value's slice, which
// shares its backing array with the map, so those pointers are live too.
func (r *Room) WalkItems(fn func(*items.Item)) {
	items.WalkSlice(r.Items, fn)
	items.WalkSlice(r.Stash, fn)
	for name := range r.Containers {
		items.WalkSlice(r.Containers[name].Items, fn)
	}
	for i := range r.Corpses {
		r.Corpses[i].Character.WalkItems(fn)
		items.WalkSlice(r.Corpses[i].Loot.Items, fn)
	}
	if r.SealedCrate != nil {
		r.SealedCrate.WalkItems(fn)
	}
}

// LoadedRooms returns every room in memory, ephemeral rooms included. The
// caller holds the mud lock (util.LockMud), which guards the room map.
func LoadedRooms() []*Room {
	out := make([]*Room, 0, len(roomManager.rooms))
	for _, r := range roomManager.rooms {
		out = append(out, r)
	}
	return out
}
```

In `internal/sealedcrate/sealedcrate.go`, after `func (c *Crate) Snapshot() []items.Item { ... }`, add:

```go
// WalkItems calls fn with a pointer to each item in the crate, holding the
// crate's lock. fn must not call back into the crate.
func (c *Crate) WalkItems(fn func(*items.Item)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	items.WalkSlice(c.items, fn)
}
```

Append to `internal/sealedcrate/sealedcrate_test.go` (it is `package sealedcrate`; add `"github.com/GoMudEngine/GoMud/internal/items"` to its imports if absent):

```go
func TestCrateWalkItems(t *testing.T) {
	c := New(1, 5)
	c.Add(items.Item{ItemId: 900, Bauble: `B0000001`})
	c.Add(items.Item{ItemId: 12})
	seen := []int{}
	c.WalkItems(func(it *items.Item) { seen = append(seen, it.ItemId) })
	if len(seen) != 2 || seen[0] != 900 || seen[1] != 12 {
		t.Fatalf("walked %v, want [900 12]", seen)
	}
}
```

Create `internal/mobs/walk_items.go`:

```go
package mobs

import "github.com/GoMudEngine/GoMud/internal/items"

// WalkItems calls fn with a pointer to every item the mob holds (its
// character's, Character.WalkItems). Nothing else on a Mob holds items;
// TestItemWalkersVisitEveryItemField (repo root) fails if that changes.
func (m *Mob) WalkItems(fn func(*items.Item)) {
	m.Character.WalkItems(fn)
}
```

Create `internal/shops/walk_items.go`:

```go
package shops

import "github.com/GoMudEngine/GoMud/internal/items"

// WalkItems calls fn with a pointer to each unique item the shop resells
// (AffixedStock). Stock entries are counts of an item id, not items.
func (si *ShopInventory) WalkItems(fn func(*items.Item)) {
	for i := range si.AffixedStock {
		if si.AffixedStock[i].Item.ItemId > 0 {
			fn(&si.AffixedStock[i].Item)
		}
	}
}
```

Create `internal/guilds/walk_items.go`:

```go
package guilds

import "github.com/GoMudEngine/GoMud/internal/items"

// WalkItems calls fn with a pointer to each item in the guild vault.
func (g *Guild) WalkItems(fn func(*items.Item)) {
	items.WalkSlice(g.Vault, fn)
}
```

Create `modules/auctions/walk_items.go`:

```go
package auctions

import "github.com/GoMudEngine/GoMud/internal/items"

// WalkItems calls fn with a pointer to every item the auction house holds:
// the lot on the block and each seized lot waiting for one. Past auctions
// keep only names.
func (am *AuctionManager) WalkItems(fn func(*items.Item)) {
	if am.ActiveAuction != nil && am.ActiveAuction.ItemData.ItemId > 0 {
		fn(&am.ActiveAuction.ItemData)
	}
	for i := range am.SeizedQueue {
		if am.SeizedQueue[i].Item.ItemId > 0 {
			fn(&am.SeizedQueue[i].Item)
		}
	}
}
```

- [ ] **Step 4: Run the guard and the package tests**

Run: `go test . -run TestItemWalkersVisitEveryItemField -v`
Expected: PASS, six subtests (`auctions`, `guilds`, `mobs`, `rooms`, `shops`, `users`). If a subtest names a path these walkers do not cover, the guard is doing its job: walk that field in the owning type's `WalkItems` and rerun; do not change the guard.

Run: `go test ./internal/users/ -run TestGetAllLoadedUsersIncludesZombies` and `go test ./internal/sealedcrate/ -run TestCrateWalkItems`
Expected: `ok` for both.

- [ ] **Step 5: Prove the guard can fail (null probe)**

In `internal/characters/walk_items.go`, comment out `items.WalkSlice(c.Pet.Items, fn)`. Run `go test . -run TestItemWalkersVisitEveryItemField`.
Expected: FAIL in the `users`, `mobs` and `rooms` subtests, each naming a path ending in `.Pet.Items[0]` (for example `UserRecord.Character.Pet.Items[0]`). Restore the line and rerun: PASS.

- [ ] **Step 6: Commit**

```bash
git -C C:/tmp/dogmud-bauble-sweep add item_walker_guard_test.go internal/characters/walk_items.go internal/users/walk_items.go internal/users/walk_items_test.go internal/rooms/walk_items.go internal/sealedcrate/sealedcrate.go internal/sealedcrate/sealedcrate_test.go internal/mobs/walk_items.go internal/shops/walk_items.go internal/guilds/walk_items.go modules/auctions/walk_items.go
git -C C:/tmp/dogmud-bauble-sweep commit -m "feat: WalkItems on every store of items, with a field-planting guard

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: The bow detune migration reaches companions' gear

**Files:**
- Modify: `internal/characters/migrate_detuned_bows.go:63-86`
- Test: `internal/characters/migrate_detuned_bows_test.go` (`TestMigrateDetunedRangedWeapons_ReachesEveryCarriedCollection`)

- [ ] **Step 1: Extend the reach test**

In `TestMigrateDetunedRangedWeapons_ReachesEveryCarriedCollection`, change the character literal and the lines after it to:

```go
	c := &Character{
		Name:           "TestArcher",
		Items:          []items.Item{preDetuneBow()},
		ComponentItems: []items.Item{preDetuneBow()},
		PotionItems:    []items.Item{preDetuneBow()},
		Pet:            pets.Pet{Type: "packmule", Capacity: 4, Items: []items.Item{preDetuneBow()}},
		Companions:     []CompanionInfo{{Items: []items.Item{preDetuneBow()}}},
	}
	weapon := preDetuneBow()
	c.Equipment.Weapon = weapon
	c.Companions[0].Equipment.Weapon = preDetuneBow()
```

and add two rows to the table after `{"equipped weapon", ...}`:

```go
		{"companion pack", c.Companions[0].Items[0].Spec.DamageMultiplier},
		{"companion weapon", c.Companions[0].Equipment.Weapon.Spec.DamageMultiplier},
```

- [ ] **Step 2: Run it to see it fail**

Run: `go test ./internal/characters/ -run TestMigrateDetunedRangedWeapons_ReachesEveryCarriedCollection`
Expected: FAIL with `companion pack bow = 7.5000, want 2.7500 -- this collection is not in the sweep` and the same for `companion weapon`.

- [ ] **Step 3: Walk through `Character.WalkItems`**

In `internal/characters/migrate_detuned_bows.go`, replace the body of `func (c *Character) MigrateDetunedRangedWeapons()` from `ptrs := make(...)` through `ptrs = append(ptrs, c.Equipment.GetAllItemPtrs()...)` with:

```go
	// Every item the character holds, companions' gear included, through the
	// one walker the bauble sweep also relies on (walk_items.go).
	ptrs := []*items.Item{}
	c.WalkItems(func(it *items.Item) { ptrs = append(ptrs, it) })
```

leaving `updated := items.MigrateDetunedRangedWeapons(ptrs)` and the log line as they are.

- [ ] **Step 4: Run the migration tests**

Run: `go test ./internal/characters/ -run TestMigrateDetunedRangedWeapons`
Expected: `ok` (all three tests).

- [ ] **Step 5: Commit**

```bash
git -C C:/tmp/dogmud-bauble-sweep add internal/characters/migrate_detuned_bows.go internal/characters/migrate_detuned_bows_test.go
git -C C:/tmp/dogmud-bauble-sweep commit -m "fix(characters): the bow detune migration reaches companions' gear

It hand-listed the character's collections and missed CompanionInfo's
pack and equipment. It now walks Character.WalkItems.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: The item-holder guard

**Files:**
- Create: `bauble_sweep_guard_test.go` (repo root)

- [ ] **Step 1: Confirm each transient reason by reading the type**

Run each and confirm the type is an argument, result, event, view or disk shape that does not outlive its call or tick:

```bash
cd C:/tmp/dogmud-bauble-sweep
grep -rn "type \(DropItemResult\|EquipItemResult\|GetItemResult\|GiveItemResult\|RemoveEquipResult\|StealOptions\)" internal/actions
grep -rn "type \(DisarmResult\|weaponSetup\)" internal/combat
grep -rn "type \(WeaponBreakResult\|plannedSeizure\)" internal/hooks
grep -rn "type \(SwapDelta\)" internal/itemvalue; grep -rn "type Match " internal/parser
grep -rn "type enchantSlotCandidate" internal/usercommands; grep -rn "type thing " modules/aicompanion
```

If any of them is stored in a package-level variable or a long-lived struct, stop: it is a store, and it needs a sweep root instead of an allowlist entry.

- [ ] **Step 2: Write the guard**

Create `bauble_sweep_guard_test.go`:

```go
package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// sweepModulePath is go.mod's module line.
const sweepModulePath = `github.com/GoMudEngine/GoMud`

// transientItemHolders are structs that hold an items.Item only for the
// length of one call or one tick, so the bauble sweep has nothing to find
// in them: the item they carry lives in a store the sweep walks, or is on
// its way there within the same lock hold. Keyed "<dir>.<Type>".
var transientItemHolders = map[string]string{
	`internal/actions.DropItemResult`:            `an action's result, alive for one call`,
	`internal/actions.EquipItemResult`:           `an action's result, alive for one call`,
	`internal/actions.GetItemResult`:             `an action's result, alive for one call`,
	`internal/actions.GiveItemResult`:            `an action's result, alive for one call`,
	`internal/actions.RemoveEquipResult`:         `an action's result, alive for one call`,
	`internal/actions.StealOptions`:              `a steal's arguments, alive for one call`,
	`internal/characters.HandSlot`:               `a view: a pointer into Worn, which Character.WalkItems walks`,
	`internal/characters.WornSlot`:               `a view: a pointer into Worn (AllSlots), which Worn.WalkItems walks`,
	`internal/combat.DisarmResult`:               `an attack's result, alive for one call`,
	`internal/combat.weaponSetup`:                `one attack's weapon, alive for one call`,
	`internal/events.EquipmentChange`:            `an event carrying a copy of an item that lives in a store, handled within the tick`,
	`internal/events.ItemOwnership`:              `an event carrying a copy of an item that lives in a store, handled within the tick`,
	`internal/events.StorageItemSeized`:          `an event moving a seized bank item to the auction queue within the tick`,
	`internal/hooks.WeaponBreakResult`:           `a weapon break's result, alive for one call`,
	`internal/hooks.plannedSeizure`:              `a storage fee's plan, alive for one call`,
	`internal/itemvalue.SwapDelta`:               `an item comparison, alive for one call`,
	`internal/parser.Match`:                      `a parsed command's target, alive for one command`,
	`internal/sealedcrate.cratePayload`:          `the crate file's on-disk shape; the sweep reads crates/ from disk`,
	`internal/usercommands.enchantSlotCandidate`: `one command's choice, alive for one call`,
	`modules/aicompanion.thing`:                  `one prompt's description of the room, alive for one call`,
}

// itemHolderTypes lists "<dir>.<Type>" for every top-level struct in the
// repo's non-test Go files with a field whose type mentions items.Item.
func itemHolderTypes(t *testing.T) []string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot find the repo root")
	}
	root := filepath.Dir(here)
	out := []string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case `.git`, `_datafiles`, `node_modules`, `vendor`:
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, `.go`) || strings.HasSuffix(p, `_test.go`) {
			return nil
		}
		f, perr := parser.ParseFile(token.NewFileSet(), p, nil, parser.SkipObjectResolution)
		if perr != nil {
			return perr
		}
		inItems := f.Name.Name == `items`
		rel, _ := filepath.Rel(root, filepath.Dir(p))
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts := spec.(*ast.TypeSpec)
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, fld := range st.Fields.List {
					if mentionsItem(fld.Type, inItems) {
						out = append(out, filepath.ToSlash(rel)+`.`+ts.Name.Name)
						break
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

// mentionsItem reports whether a field type spells items.Item (or Item,
// inside package items), not counting function or interface types.
func mentionsItem(e ast.Expr, inItems bool) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok && id.Name == `items` && x.Sel.Name == `Item` {
				found = true
			}
			return false
		case *ast.Ident:
			if inItems && x.Name == `Item` {
				found = true
			}
		case *ast.FuncType, *ast.InterfaceType:
			return false
		}
		return true
	})
	return found
}

// sweepReachableTypes lists "<dir>.<Type>" for every named type reachable
// from a sweep root through fields, pointers, slices, arrays and maps.
func sweepReachableTypes() map[string]bool {
	out := map[string]bool{}
	seen := map[reflect.Type]bool{}
	var visit func(t reflect.Type)
	visit = func(t reflect.Type) {
		if seen[t] {
			return
		}
		seen[t] = true
		if t.Name() != `` && strings.HasPrefix(t.PkgPath(), sweepModulePath+`/`) {
			out[strings.TrimPrefix(t.PkgPath(), sweepModulePath+`/`)+`.`+t.Name()] = true
		}
		switch t.Kind() {
		case reflect.Struct:
			for i := 0; i < t.NumField(); i++ {
				visit(t.Field(i).Type)
			}
		case reflect.Pointer, reflect.Slice, reflect.Array:
			visit(t.Elem())
		case reflect.Map:
			visit(t.Key())
			visit(t.Elem())
		}
	}
	for _, r := range sweepRoots() {
		visit(r.typ)
	}
	return out
}

// Every struct that holds an items.Item is either reached from a bauble
// sweep root (and so walked: TestItemWalkersVisitEveryItemField) or listed
// as transient with its reason. A new store of items that is neither would
// have its baubles' records pruned while they still exist.
func TestEveryItemHolderIsASweepRootOrTransient(t *testing.T) {
	holders := itemHolderTypes(t)
	if len(holders) < 20 {
		t.Fatalf("found only %d item-holding types: the scan is broken, not the repo", len(holders))
	}
	reach := sweepReachableTypes()
	isHolder := map[string]bool{}
	for _, h := range holders {
		isHolder[h] = true
		_, transient := transientItemHolders[h]
		switch {
		case reach[h] && transient:
			t.Errorf("%s is reached from a bauble sweep root; take it off transientItemHolders", h)
		case !reach[h] && !transient:
			t.Errorf("%s holds an items.Item but no bauble sweep root reaches it. If it outlives one call, give it a WalkItems and a live source (bauble_sweep.go) and a root in item_walker_guard_test.go; if it is transient, add it to transientItemHolders with the reason", h)
		}
	}
	for h := range transientItemHolders {
		if !isHolder[h] {
			t.Errorf("transientItemHolders lists %s, which no longer holds an item; remove it", h)
		}
	}
}
```

- [ ] **Step 3: Run it**

Run: `go test . -run TestEveryItemHolderIsASweepRootOrTransient -v`
Expected: PASS. (The 14 reachable holders are `characters.Character`, `CompanionInfo`, `Worn`, `guilds.Guild`, `pets.Pet`, `rooms.Container`, `rooms.Room`, `sealedcrate.Crate`, `shops.AffixedStockEntry`, `users.Message`, `users.Storage`, `users.StorageSlot`, `auctions.AuctionItem`, `auctions.SeizedLot`.)

- [ ] **Step 4: Prove it can fail (null probe)**

Delete the `internal/parser.Match` line from `transientItemHolders` and rerun: expected FAIL naming `internal/parser.Match holds an items.Item but no bauble sweep root reaches it`. Restore it. Then add `` `internal/parser.Nope`: `x`, `` and rerun: expected FAIL `transientItemHolders lists internal/parser.Nope, which no longer holds an item`. Remove it and rerun: PASS.

- [ ] **Step 5: Commit**

```bash
git -C C:/tmp/dogmud-bauble-sweep add bauble_sweep_guard_test.go
git -C C:/tmp/dogmud-bauble-sweep commit -m "test: every item-holding struct is a bauble sweep root or listed transient

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: The `BaubleSweepHours` knob

**Files:**
- Modify: `internal/configs/config.balance.go:940`, `internal/configs/config.balance.baubles.go:46-47` and `:226-231`, `_datafiles/config.yaml` (after `BaubleCatalogKeepDays: 30`)
- Test: `internal/configs/config.balance.baubles_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/configs/config.balance.baubles_test.go`:

```go
// The catalog sweep runs every BaubleSweepHours real hours: 6 by default,
// never less than 1.
func TestBaubleSweepHours(t *testing.T) {
	b := &Balance{}
	b.validateBaubles()
	if b.BaubleSweepHours != 6 {
		t.Fatalf("default %d, want 6", b.BaubleSweepHours)
	}
	b.BaubleSweepHours = -2
	b.validateBaubles()
	if b.BaubleSweepHours != 6 {
		t.Fatalf("a negative value resets to %d, want 6", b.BaubleSweepHours)
	}
	b.BaubleSweepHours = 1
	b.validateBaubles()
	if b.BaubleSweepHours != 1 {
		t.Fatalf("1 is allowed, got %d", b.BaubleSweepHours)
	}
}
```

and in `TestBaubleShippedConfigMatchesDefaults`, after the line `shipped.BaubleCatalogKeepDays != defaults.BaubleCatalogKeepDays ||` add:

```go
		shipped.BaubleSweepHours != defaults.BaubleSweepHours ||
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/configs/ -run "TestBaubleSweepHours|TestBaubleShippedConfigMatchesDefaults"`
Expected: FAIL to compile, `b.BaubleSweepHours undefined`.

- [ ] **Step 3: Declare, default and ship the knob**

In `internal/configs/config.balance.go`, replace the line

```go
	BaubleCatalogKeepDays ConfigInt           `yaml:"BaubleCatalogKeepDays"` // Real days a record whose bauble was sold or vanished is kept before it is pruned (default 30, at least 7)
```

with

```go
	BaubleCatalogKeepDays ConfigInt           `yaml:"BaubleCatalogKeepDays"` // Real days a record no item points at any more is kept before the catalog sweep prunes it, counted from the last sign of the bauble (default 30, at least 7)
	BaubleSweepHours      ConfigInt           `yaml:"BaubleSweepHours"`      // Real hours between catalog sweeps, which look for every item still pointing at a record; one also runs at boot (default 6, at least 1)
```

In `internal/configs/config.balance.baubles.go`, after `minBaubleCatalogKeepDays     = 7 // sales stats read the last seven days` add:

```go
	defaultBaubleSweepHours      = 6
```

and after the block

```go
	if b.BaubleCatalogKeepDays < minBaubleCatalogKeepDays {
		b.BaubleCatalogKeepDays = minBaubleCatalogKeepDays
	}
```

add:

```go
	if b.BaubleSweepHours <= 0 {
		b.BaubleSweepHours = defaultBaubleSweepHours
	}
```

In `_datafiles/config.yaml` (this worktree's copy has no skip-worktree bit, fact 37), replace

```yaml
  # A catalog record whose bauble was sold, or vanished untaken, is pruned
  # this many REAL days later (at least 7; sales stats read the last week).
  # Records that earned a thief return credit are always kept.
  BaubleCatalogKeepDays: 30
```

with

```yaml
  # A catalog record nothing in the world points at any more (the bauble
  # was sold, vanished, junked, or lost with whoever held it) is pruned once
  # this many REAL days have passed since the last sign of it (at least 7;
  # sales stats read the last week). Records that earned a thief return
  # credit are always kept.
  BaubleCatalogKeepDays: 30
  # The catalog sweep runs at boot and then every this many REAL hours: it
  # looks for every item still pointing at a record, in the live world and
  # in every save file, and prunes the records nothing has pointed at for
  # two sweeps and BaubleCatalogKeepDays (at least 1).
  BaubleSweepHours: 6
```

- [ ] **Step 4: Run the tests, then gofmt**

Run: `gofmt -l internal/configs/` (expected: nothing; if it prints `config.balance.go`, run `gofmt -w internal/configs/config.balance.go`).
Run: `go test ./internal/configs/ -run "TestBaubleSweepHours|TestBaubleShippedConfigMatchesDefaults|TestBaubleDefaults"`
Expected: `ok`.

- [ ] **Step 5: Null probe**

Temporarily delete the `BaubleSweepHours: 6` line from `_datafiles/config.yaml`; `TestBaubleShippedConfigMatchesDefaults` must FAIL (`shipped bauble config differs`). Restore it; PASS.

- [ ] **Step 6: Commit**

```bash
git -C C:/tmp/dogmud-bauble-sweep add internal/configs/config.balance.go internal/configs/config.balance.baubles.go internal/configs/config.balance.baubles_test.go _datafiles/config.yaml
git -C C:/tmp/dogmud-bauble-sweep commit -m "feat(configs): BaubleSweepHours, how often the catalog sweep runs

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Seen and unseen records; the apply half of the sweep

**Files:**
- Modify: `internal/baubles/record.go` (after `SoldValue`), `internal/baubles/catalog.go` (lines 115-117, 249-270, 274-304, 324-394)
- Create: `internal/baubles/sweep.go`
- Modify: `internal/baubles/prune_test.go` (remove `TestPruneRemovesOnlyWhatIsGone`)
- Test: `internal/baubles/sweep_apply_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/baubles/sweep_apply_test.go`:

```go
package baubles

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// sweepRecord creates a record and applies change to it.
func sweepRecord(t *testing.T, name string, change func(r *Record)) string {
	t.Helper()
	r, err := Create(Record{Name: name, NameSimple: `thing`, Tier: TierCheap, Value: 3, Status: StatusReady})
	if err != nil {
		t.Fatal(err)
	}
	if change != nil {
		Update(r.Id, change)
	}
	return r.Id
}

// readShards returns the text of every catalog shard in dir.
func readShards(t *testing.T, dir string) string {
	t.Helper()
	shards, _ := filepath.Glob(filepath.Join(dir, `catalog-*.yaml`))
	out := ``
	for _, f := range shards {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		out += string(data)
	}
	return out
}

// A record goes once two complete sweeps in a row found nothing pointing at
// it and the keep window has passed since the last sign of it. Sold,
// vanished, retired or simply lost makes no difference; a held record stays
// (a sold one included, its sale left as it was); a credited return always
// stays; only catalog shards are rewritten.
func TestApplySweepPrunesOnlyWhatNothingHolds(t *testing.T) {
	dir := t.TempDir()
	SetDirForTest(dir)
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	setBaubleConfig(t, func(b *configs.Balance) { b.BaubleCatalogKeepDays = 30 })
	overlay := filepath.Join(dir, `corpus.promoted.yaml`)
	if err := os.WriteFile(overlay, []byte("entries: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	old, recent := now.Add(-31*24*time.Hour), now.Add(-2*24*time.Hour)
	lost := sweepRecord(t, `Lost Long Ago`, func(r *Record) { r.FoundAt = old })
	lostRecent := sweepRecord(t, `Lost Lately`, func(r *Record) { r.FoundAt = recent })
	soldOld := sweepRecord(t, `Sold Long Ago`, func(r *Record) { r.FoundAt, r.Status, r.SoldAt, r.SoldValue = old, StatusSold, old, 2 })
	soldRecent := sweepRecord(t, `Sold Lately`, func(r *Record) { r.FoundAt, r.Status, r.SoldAt, r.SoldValue = old, StatusSold, recent, 2 })
	vanished := sweepRecord(t, `Vanished Long Ago`, func(r *Record) { r.FoundAt, r.VanishedAt = old, old })
	retired := sweepRecord(t, `Retired And Lost`, func(r *Record) { r.FoundAt, r.Status = old, StatusRetired })
	credited := sweepRecord(t, `Credited Return`, func(r *Record) {
		r.FoundAt, r.Status, r.SoldAt, r.SoldValue = old, StatusSold, old, 2
		r.ReturnCreditUserId, r.ReturnCreditFactions, r.ReturnCreditAt = 7, []string{`town`}, old
	})
	held := sweepRecord(t, `Still Held`, func(r *Record) { r.FoundAt = old })
	soldHeld := sweepRecord(t, `Sold Then Rolled Back`, func(r *Record) { r.FoundAt, r.Status, r.SoldAt, r.SoldValue = old, StatusSold, old, 2 })
	refs := map[string]bool{held: true, soldHeld: true}
	keep := KeepDuration()

	if ref, n := applySweep(now, refs, keep); ref != 2 || n != 0 {
		t.Fatalf("first sweep: %d referenced, %d pruned; want 2 and 0 (one unseen sweep is not enough)", ref, n)
	}
	later := now.Add(time.Hour)
	if _, n := applySweep(later, refs, keep); n != 4 {
		t.Fatalf("second sweep pruned %d, want 4 (lost, sold, vanished, retired: all long unseen)", n)
	}
	check := func(when string) {
		t.Helper()
		for _, id := range []string{lost, soldOld, vanished, retired} {
			if _, ok := Get(id); ok {
				t.Errorf("%s: %s should be pruned", when, id)
			}
		}
		for _, id := range []string{lostRecent, soldRecent, credited, held, soldHeld} {
			if _, ok := Get(id); !ok {
				t.Errorf("%s: %s should be kept", when, id)
			}
		}
		if n := ReturnCredits(7, `town`, 0); n != 1 {
			t.Errorf("%s: the credited record still counts (index): %d", when, n)
		}
		if r, _ := Get(soldHeld); r.Status != StatusSold || !r.LastSeenAt.Equal(later) || r.UnseenSweeps != 0 {
			t.Errorf("%s: a held sold record is seen, its sale left as it was: %+v", when, r)
		}
	}
	check(`after the sweep`)

	onDisk := readShards(t, dir)
	if strings.Contains(onDisk, `Lost Long Ago`) || !strings.Contains(onDisk, `Still Held`) || !strings.Contains(onDisk, `last_seen_at`) {
		t.Fatalf("the shards are rewritten without the pruned records and with the seen times:\n%s", onDisk)
	}
	if err := loadFrom(dir); err != nil {
		t.Fatal(err)
	}
	check(`after reload`)
	if data, err := os.ReadFile(overlay); err != nil || string(data) != "entries: {}\n" {
		t.Fatalf("a file that is not a catalog shard is untouched: %q %v", data, err)
	}
}

// One second inside the keep window a record stays; at exactly the window
// it goes.
func TestApplySweepKeepWindowBoundary(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	setBaubleConfig(t, func(b *configs.Balance) { b.BaubleCatalogKeepDays = 30 })
	id := sweepRecord(t, `Boundary Pebble`, nil)
	base := time.Now().UTC()
	keep := KeepDuration()

	applySweep(base, map[string]bool{id: true}, keep) // last seen at base
	if _, n := applySweep(base.Add(time.Hour), nil, keep); n != 0 {
		t.Fatalf("pruned %d after one unseen sweep", n)
	}
	if _, n := applySweep(base.Add(keep-time.Second), nil, keep); n != 0 {
		t.Fatal("one second inside the keep window it stays")
	}
	if _, ok := Get(id); !ok {
		t.Fatal("still there inside the window")
	}
	if _, n := applySweep(base.Add(keep), nil, keep); n != 1 {
		t.Fatal("at exactly the keep window it goes")
	}
}

// Persist before publish: a shard write that fails takes nothing out of
// memory, and the next sweep prunes it.
func TestApplySweepWriteFailurePrunesNothing(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	now := time.Now().UTC()
	id := sweepRecord(t, `Unlucky Button`, func(r *Record) {
		r.FoundAt, r.UnseenSweeps = now.Add(-40*24*time.Hour), minUnseenSweeps
	})

	orig := shardWriter
	shardWriter = func(string, int, []*Record) error { return errors.New(`disk full`) }
	t.Cleanup(func() { shardWriter = orig })
	if _, n := applySweep(now, nil, KeepDuration()); n != 0 {
		t.Fatalf("pruned %d with every write failing", n)
	}
	if _, ok := Get(id); !ok {
		t.Fatal("a failed write takes nothing out of memory")
	}
	shardWriter = orig
	if _, n := applySweep(now, nil, KeepDuration()); n != 1 {
		t.Fatalf("pruned %d once writes work again, want 1", n)
	}
}

// A record found after the sweep began is left alone.
func TestApplySweepLeavesNewerRecordsAlone(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	now := time.Now().UTC()
	id := sweepRecord(t, `Fresh Find`, func(r *Record) { r.FoundAt = now.Add(time.Minute) })
	applySweep(now, nil, KeepDuration())
	if r, _ := Get(id); r.UnseenSweeps != 0 {
		t.Fatalf("a find newer than the sweep was counted unseen: %+v", r)
	}
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/baubles/ -run TestApplySweep`
Expected: FAIL to compile: `undefined: applySweep`, `r.LastSeenAt undefined`, `undefined: minUnseenSweeps`.

- [ ] **Step 3: Add the record fields**

In `internal/baubles/record.go`, after

```go
	SoldAt    time.Time `yaml:"sold_at,omitempty"`
	SoldValue int       `yaml:"sold_value,omitempty"`
```

add:

```go

	// Set by the catalog sweep (sweep.go). LastSeenAt is when a sweep last
	// found an item pointing at this record; UnseenSweeps counts the
	// complete sweeps since then that found none (it stops at
	// minUnseenSweeps). Both can only keep a record longer.
	LastSeenAt   time.Time `yaml:"last_seen_at,omitempty"`
	UnseenSweeps int       `yaml:"unseen_sweeps,omitempty"`
```

- [ ] **Step 4: Change the catalog**

In `internal/baubles/catalog.go`:

(a) In `loadFrom`, delete

```go
	if n := Prune(time.Now()); n > 0 {
		mudlog.Info(`baubles.Load`, `action`, `pruned records`, `count`, n)
	}
```

(b) Replace the `SaveAll` doc comment line `// Everything else is already on disk. Call at shutdown and copyover.` with

```go
// Everything else is already on disk. Call at shutdown and copyover.
// Pruning is the catalog sweep's alone (sweep.go).
```

and delete from `SaveAll` the block

```go
	if n := Prune(time.Now()); n > 0 {
		mudlog.Info(`baubles`, `action`, `pruned records`, `count`, n)
	}
```

(c) Replace the whole of `func (c *catalog) persistShard(shard int) error { ... }` (its doc comment kept) with:

```go
// persistShard writes every record of one shard, with mu free while it
// marshals and writes (see the catalog comment). Never call it holding mu.
func (c *catalog) persistShard(shard int) error {
	_, err := c.persistShardPruning(shard, nil)
	return err
}

// persistShardPruning writes one shard as persistShard does, leaving out
// every record prune reports true for, and only once that write has
// succeeded takes those records out of memory: persist before publish, so
// a failed write prunes nothing. It holds writeMu from the snapshot to the
// removal, so no other write of the shard can land in between. A record
// that changed since the snapshot and is no longer prunable is kept, and
// the shard is marked dirty so the next write puts it back on disk. prune
// runs under the catalog lock: keep it to reading fields. It returns how
// many records it removed.
func (c *catalog) persistShardPruning(shard int, prune func(r *Record) bool) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.mu.RLock()
	dir := c.dir
	recs := []*Record{}
	left := []string{}
	for id, r := range c.records {
		seq, ok := seqOf(id)
		if !ok || shardOf(seq) != shard {
			continue
		}
		if prune != nil && prune(r) {
			left = append(left, id)
			continue
		}
		cp := *r
		recs = append(recs, &cp)
	}
	c.mu.RUnlock()

	err := shardWriter(dir, shard, recs)

	removed := 0
	c.mu.Lock()
	if err != nil {
		c.dirty[shard] = true
	} else {
		delete(c.dirty, shard)
		for _, id := range left {
			r, ok := c.records[id]
			if !ok {
				continue
			}
			if !prune(r) {
				c.dirty[shard] = true
				continue
			}
			delete(c.records, id)
			c.indexCreditLocked(&Record{Id: id})
			removed++
		}
	}
	c.mu.Unlock()
	if err != nil {
		mudlog.Error(`baubles`, `action`, `write shard`, `shard`, shard, `error`, err)
	}
	return removed, err
}
```

(d) Replace everything from the `// KeepDuration is how long a record whose bauble is gone from the world` comment to the end of the file (`KeepDuration`, `goneAt`, `prunable`, `Prune`) with:

```go
// KeepDuration is how long a record nothing in the world points at any
// more is kept before the catalog sweep prunes it
// (Balance.BaubleCatalogKeepDays), counted from the last sign of its
// bauble (Record.lastEvidence). Sales stats read the last seven days, so it
// is never shorter than a week.
func KeepDuration() time.Duration {
	return time.Duration(configs.GetBalanceConfig().BaubleCatalogKeepDays) * 24 * time.Hour
}

// minUnseenSweeps is how many complete sweeps in a row must find no item
// pointing at a record before it can be pruned: one sweep can miss an item
// that moves between stores while it looks (sweep.go).
const minUnseenSweeps = 2

// lastEvidence is the latest time anything showed this record's bauble
// existed: found, stolen, recognised, returned, sold, vanished, or seen by a
// sweep.
func (r Record) lastEvidence() time.Time {
	last := r.FoundAt
	for _, t := range []time.Time{r.StolenAt, r.RecognizedAt, r.ReturnedAt, r.SoldAt, r.VanishedAt, r.LastSeenAt} {
		if t.After(last) {
			last = t
		}
	}
	return last
}

// prunableAt reports whether the sweep may remove r at now: at least
// minUnseenSweeps complete sweeps in a row found nothing pointing at it AND
// keep has passed since its lastEvidence. Sold, vanished and retired
// records are no exception either way: a sold bauble a crash put back in a
// pack is seen, and kept, like any other. A record whose return earned its
// thief reputation (ReturnCreditAt) always stays: the credit history lives
// only there, and dropping it would let the bauble earn credit again.
func (r Record) prunableAt(now time.Time, keep time.Duration) bool {
	if !r.ReturnCreditAt.IsZero() || r.UnseenSweeps < minUnseenSweeps {
		return false
	}
	return now.Sub(r.lastEvidence()) >= keep
}
```

- [ ] **Step 5: Create `sweep.go` with the apply half**

Create `internal/baubles/sweep.go`:

```go
package baubles

import (
	"time"
)

// The catalog sweep. A record is only useful while some item points at it,
// and items leave the world in many ways the catalog never hears about
// (junked, eaten by a script, left on a corpse that decayed, on a character
// that was deleted, in a room file that was wiped). So rather than hook
// every one of them, a sweep periodically collects every bauble id any item
// still points at, in the live world and in every save file, and prunes the
// records nothing has pointed at for KeepDuration.
//
// Two phases. Collect (runSweep): the live world under the mud lock,
// through the sources registered with RegisterLiveSource (users, rooms,
// mobs, shops, guilds, the auction house), then every save file under
// DataFiles off the lock (sweep_disk.go). Apply (applySweep): under the
// catalog lock, a record something points at gets LastSeenAt = now and
// UnseenSweeps = 0, any other counts one more unseen sweep; then each
// changed shard is written without the records now prunable (prunableAt),
// before they leave memory (persistShardPruning).
//
// Fail closed: a collection that went wrong anywhere (a live source that
// panicked, a file that could not be read, or that names a bauble and does
// not parse) applies nothing at all, so a store it could not see never
// looks empty.
//
// Why a record must stay unseen for minUnseenSweeps sweeps AND
// KeepDuration: an item moving between stores while a sweep looks (from a
// room file into a live room, say) can be missed by that one sweep. Nothing
// is lost to a miss unless it repeats for the whole keep window. A store
// the sweep never looks at would repeat forever; the repo-root guards
// (TestItemWalkersVisitEveryItemField,
// TestEveryItemHolderIsASweepRootOrTransient) are what stop that.
//
// A sold record that something points at again (a crash rolled the
// seller's save back past the sale) is simply seen, so it stays. Its status
// is left as sold: every record is sellable already (sales.go), a save file
// on disk can lag a sale by one autosave, and rewriting the sale on that
// evidence would erase real ones.

// applySweep folds one complete collection (refs: every record id some item
// points at) into the catalog at now, and returns how many records were
// referenced and how many it pruned. Records found after now are left
// alone.
func applySweep(now time.Time, refs map[string]bool, keep time.Duration) (referenced int, pruned int) {
	shards := map[int]bool{}

	cat.mu.Lock()
	if cat.dir == `` {
		cat.mu.Unlock()
		return 0, 0
	}
	for id, r := range cat.records {
		seq, ok := seqOf(id)
		if !ok || r.FoundAt.After(now) {
			continue
		}
		shard := shardOf(seq)
		if refs[id] {
			referenced++
			r.LastSeenAt, r.UnseenSweeps = now, 0
			shards[shard] = true
			continue
		}
		if r.UnseenSweeps < minUnseenSweeps {
			r.UnseenSweeps++
			shards[shard] = true
		}
		if r.prunableAt(now, keep) {
			shards[shard] = true
		}
	}
	cat.mu.Unlock()

	prune := func(r *Record) bool { return r.prunableAt(now, keep) }
	for shard := range shards {
		n, _ := cat.persistShardPruning(shard, prune)
		pruned += n
	}
	return referenced, pruned
}
```

- [ ] **Step 6: Remove the old prune test**

In `internal/baubles/prune_test.go`, delete from the comment line `// Prune (the owner's fix round, item 3): a record whose bauble is gone from the world` through the closing `}` of `TestPruneRemovesOnlyWhatIsGone` (the line before `// The keep period is at least a week: sales stats read the last seven days.`). Its cases now live in `TestApplySweepPrunesOnlyWhatNothingHolds`. Replace the file's import block with:

```go
import (
	"sync"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)
```

- [ ] **Step 7: Run the package tests**

Run: `go build ./... && go test ./internal/baubles/`
Expected: `ok`. (`go build ./...` confirms nothing else called `Prune`.)

- [ ] **Step 8: Null probes**

(a) In `prunableAt`, change `r.UnseenSweeps < minUnseenSweeps` to `r.UnseenSweeps < 1`: `TestApplySweepPrunesOnlyWhatNothingHolds` must FAIL at `first sweep: ... want 2 and 0`. Restore.
(b) In `persistShardPruning`, move the `delete(c.records, id)` loop above `err := shardWriter(...)` (publish before persist): `TestApplySweepWriteFailurePrunesNothing` must FAIL with `a failed write takes nothing out of memory`. Restore. Rerun: `ok`.

- [ ] **Step 9: Commit**

```bash
git -C C:/tmp/dogmud-bauble-sweep add internal/baubles/record.go internal/baubles/catalog.go internal/baubles/sweep.go internal/baubles/sweep_apply_test.go internal/baubles/prune_test.go
git -C C:/tmp/dogmud-bauble-sweep commit -m "feat(baubles): prune only records unseen by two sweeps for the keep window

Load and SaveAll no longer prune. Records gain LastSeenAt and
UnseenSweeps; a shard is written without its prunable records before they
leave memory.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: The disk half: `DiskRefs`

**Files:**
- Create: `internal/baubles/sweep_disk.go`
- Test: `internal/baubles/sweep_disk_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/baubles/sweep_disk_test.go`:

```go
package baubles

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// writeDataFiles writes each file (slash path relative to root).
func writeDataFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func sortedIds(refs map[string]bool) []string {
	out := make([]string, 0, len(refs))
	for id := range refs {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Items are found in every save shape, as YAML or JSON, in any document of
// a file; the catalog and the economy snapshots are skipped, and
// quarantined and half-written files are never read.
func TestDiskRefsFindsItemsInEverySaveShape(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	young := now.Add(-time.Hour).Unix()
	writeDataFiles(t, root, map[string]string{
		`users/7.yaml`:      "userid: 7\ncharacter:\n  items:\n  - itemid: 900\n    bauble: B0000001\nitemstorage:\n  slots:\n  - item:\n      itemid: 900\n      bauble: B0000002\n    count: 1\ninbox:\n- item:\n    itemid: 900\n    bauble: B0000003\n",
		`users/7.alts.yaml`: "- name: Alt\n  items:\n  - itemid: 900\n    bauble: B0000004\n",
		`rooms.instances/ashwick/4023.yaml`: fmt.Sprintf("items:\n- itemid: 900\n  bauble: B0000005\n  baubleleftat: %d\ncontainers:\n  chest:\n    items:\n    - itemid: 900\n      bauble: B0000006\n", young),
		`plugin-data/auctions-v1.0/auctionhistory.plugin.dat`: "ActiveAuction:\n  ItemData:\n    itemid: 900\n    bauble: B0000007\n",
		`plugin-data/other-v1.0/state.plugin.dat`:             `{"lot": {"itemid": 900, "bauble": "B0000008"}}`,
		`shops/town/5-room1.yaml`:                             "---\naffixed_stock:\n- item:\n    itemid: 900\n    bauble: B0000009\n---\naffixed_stock:\n- item:\n    itemid: 900\n    bauble: B0000010\n",
		`economy/snapshots/1.yaml`:                            "item:\n  bauble: B0000011\n",
		`baubles/catalog-0000.yaml`:                           "records:\n- id: B0000012\n  bauble: B0000012\n",
		`users/7.yaml.corrupt-20260929T000000.000000000Z`:     "character:\n  items:\n  - bauble: B0000013\n",
		`users/8.yaml.new`:                                    "character:\n  items:\n  - bauble: B0000014\n",
		`rooms.instances/ashwick/4024.yaml`:                   "items:\n- itemid: 12\n  baublespot: on the shelf\n",
		`users/9.yaml`:                                        "character:\n  description: \"a bauble: of no account\"\n  items:\n  - itemid: 900\n    bauble: not-an-id\n",
	})

	refs, files, parsed, err := DiskRefs(root, now)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{`B0000001`, `B0000002`, `B0000003`, `B0000004`, `B0000005`, `B0000006`, `B0000007`, `B0000008`, `B0000009`, `B0000010`}
	if got := sortedIds(refs); !reflect.DeepEqual(got, want) {
		t.Fatalf("refs %v, want %v", got, want)
	}
	if files != 8 || parsed != 7 {
		t.Fatalf("read %d files and parsed %d, want 8 and 7 (4024 names no bauble key)", files, parsed)
	}
}

// A find lying untaken past the limit is not a reference, but only in
// rooms.instances, where finds lie on floors (loading the room removes it:
// rooms.LoadRoomInstance). Anywhere else the same shape still counts.
func TestDiskRefsSkipsExpiredUntakenFindsOnlyOnFloors(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	expired := now.Add(-UntakenLimit() - time.Minute).Unix()
	young := now.Add(-time.Minute).Unix()
	writeDataFiles(t, root, map[string]string{
		`rooms.instances/ashwick/4023.yaml`: fmt.Sprintf("items:\n- itemid: 900\n  bauble: B0000001\n  baubleleftat: %d\n- itemid: 900\n  bauble: B0000002\n  baubleleftat: %d\n", expired, young),
		`users/7.yaml`:                      fmt.Sprintf("character:\n  items:\n  - itemid: 900\n    bauble: B0000003\n    baubleleftat: %d\n", expired),
	})
	refs, _, _, err := DiskRefs(root, now)
	if err != nil {
		t.Fatal(err)
	}
	if got := sortedIds(refs); !reflect.DeepEqual(got, []string{`B0000002`, `B0000003`}) {
		t.Fatalf("refs %v, want [B0000002 B0000003]", got)
	}
}

// Anything that leaves the scan incomplete is an error; a broken file that
// names no bauble, or one deleted mid-scan, is not.
func TestDiskRefsFailsClosed(t *testing.T) {
	now := time.Now().UTC()
	t.Run(`a file that names a bauble must parse`, func(t *testing.T) {
		root := t.TempDir()
		writeDataFiles(t, root, map[string]string{`users/5.yaml`: "character:\n  items:\n  - bauble: B0000001\n   bad: [\n"})
		if _, _, _, err := DiskRefs(root, now); err == nil || !strings.Contains(err.Error(), `parse users/5.yaml`) {
			t.Fatalf("err %v, want a parse error naming users/5.yaml", err)
		}
	})
	t.Run(`a broken file that names no bauble is not parsed`, func(t *testing.T) {
		root := t.TempDir()
		writeDataFiles(t, root, map[string]string{`weather/broken.yaml`: "a: [\n"})
		if _, files, parsed, err := DiskRefs(root, now); err != nil || files != 1 || parsed != 0 {
			t.Fatalf("files %d parsed %d err %v, want 1, 0, nil", files, parsed, err)
		}
	})
	t.Run(`an unreadable file`, func(t *testing.T) {
		root := t.TempDir()
		writeDataFiles(t, root, map[string]string{`users/6.yaml`: "userid: 6\n"})
		orig := sweepReadFile
		sweepReadFile = func(string) ([]byte, error) { return nil, errors.New(`permission denied`) }
		t.Cleanup(func() { sweepReadFile = orig })
		if _, _, _, err := DiskRefs(root, now); err == nil || !strings.Contains(err.Error(), `read users/6.yaml`) {
			t.Fatalf("err %v, want a read error naming users/6.yaml", err)
		}
	})
	t.Run(`a file deleted since the listing`, func(t *testing.T) {
		root := t.TempDir()
		writeDataFiles(t, root, map[string]string{`users/6.yaml`: "userid: 6\n"})
		orig := sweepReadFile
		sweepReadFile = func(p string) ([]byte, error) { return nil, &fs.PathError{Op: `open`, Path: p, Err: fs.ErrNotExist} }
		t.Cleanup(func() { sweepReadFile = orig })
		if _, files, _, err := DiskRefs(root, now); err != nil || files != 0 {
			t.Fatalf("files %d err %v, want 0 and nil", files, err)
		}
	})
	t.Run(`a missing data folder`, func(t *testing.T) {
		if _, _, _, err := DiskRefs(filepath.Join(t.TempDir(), `nope`), now); err == nil {
			t.Fatal("a data folder that is not there must fail the scan")
		}
	})
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/baubles/ -run TestDiskRefs`
Expected: FAIL to compile, `undefined: DiskRefs`, `undefined: sweepReadFile`.

- [ ] **Step 3: Implement**

Create `internal/baubles/sweep_disk.go`:

```go
package baubles

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/items"
	"gopkg.in/yaml.v3"
)

// The sweep's disk half. Every item that is not in memory is in a save
// file under DataFiles: users and their alts (the bank and inbox inside
// them), rooms.instances, mobs.instances, shops, guilds, crates, and
// plugin-data for the auction house. Like migration 0.17.0 it reads the
// whole tree rather than a list of store folders, because a list goes stale
// the day someone adds a store. Two folders are skipped: the catalog itself
// (baubles/) and economy/, the dashboard's metric snapshots, which hold no
// items and are most of the tree's bytes.
//
// Cheap: a file with no `bauble` key is read but never parsed. Fail closed:
// a file that cannot be read, or that names a bauble and does not parse,
// fails the whole sweep, because skipping it would prune what it holds.
// Quarantined files (`.corrupt-...`) and util.Save's `.new` temp files do
// not end in .yaml or .plugin.dat, so they are never read.

// sweepSkipDirs are top-level folders under DataFiles the scan skips.
var sweepSkipDirs = map[string]bool{`baubles`: true, `economy`: true}

// untakenDir is the one folder where a find can lie untaken on a floor.
const untakenDir = `rooms.instances`

// baubleKeyRe finds a `bauble` key, bare or quoted as JSON writes it. Not
// `baublespot` or `baubleleftat`: the name must end at the colon.
var baubleKeyRe = regexp.MustCompile(`["']?\bbauble["']?\s*:`)

// sweepReadFile reads one data file. A variable so a test can make a read
// fail.
var sweepReadFile = os.ReadFile

// maxYAMLDepth bounds the walk of one document. Saves nest a dozen levels;
// anything deeper fails the sweep rather than being walked partway.
const maxYAMLDepth = 100

// DiskRefs reads every data file under root and returns the record ids the
// items in them point at, how many files it read, and how many of those it
// parsed. A find in rooms.instances/ that has lain untaken past
// UntakenLimit is not counted: loading its room removes it before anything
// can take it (rooms.LoadRoomInstance). Any error means the result is
// incomplete and must not be used to prune.
func DiskRefs(root string, now time.Time) (refs map[string]bool, files int, parsed int, err error) {
	refs = map[string]bool{}
	files, parsed, err = scanDisk(root, now, func(id string) { refs[id] = true })
	return refs, files, parsed, err
}

func scanDisk(root string, now time.Time, add func(id string)) (files int, parsed int, err error) {
	if _, serr := os.Stat(root); serr != nil {
		return 0, 0, fmt.Errorf(`data files: %w`, serr)
	}
	limit := UntakenLimit()
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			if errors.Is(werr, fs.ErrNotExist) {
				return nil
			}
			return werr
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if sweepSkipDirs[rel] {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, `.yaml`) && !strings.HasSuffix(name, `.plugin.dat`) {
			return nil
		}
		raw, ferr := sweepReadFile(path)
		if ferr != nil {
			if errors.Is(ferr, fs.ErrNotExist) {
				return nil // removed since the listing, and whatever it held with it
			}
			return fmt.Errorf(`read %s: %w`, rel, ferr)
		}
		files++
		if !bytes.Contains(raw, []byte(`bauble`)) || !baubleKeyRe.Match(raw) {
			return nil
		}
		parsed++
		untaken := time.Duration(0)
		if strings.HasPrefix(rel, untakenDir+`/`) {
			untaken = limit
		}
		if perr := refsInYAML(raw, now, untaken, add); perr != nil {
			return fmt.Errorf(`parse %s: %w`, rel, perr)
		}
		return nil
	})
	return files, parsed, err
}

// refsInYAML adds the id of every item in raw, every document of it.
// untaken is the untaken limit where finds lie on floors, 0 elsewhere.
func refsInYAML(raw []byte, now time.Time, untaken time.Duration, add func(id string)) error {
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	for {
		var doc yaml.Node
		if err := dec.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if err := walkYAML(&doc, 0, now, untaken, add); err != nil {
			return err
		}
	}
}

func walkYAML(n *yaml.Node, depth int, now time.Time, untaken time.Duration, add func(id string)) error {
	if n == nil {
		return nil
	}
	if depth > maxYAMLDepth {
		return fmt.Errorf(`nested deeper than %d levels`, maxYAMLDepth)
	}
	switch n.Kind {
	case yaml.AliasNode:
		return walkYAML(n.Alias, depth+1, now, untaken, add)
	case yaml.MappingNode:
		if id, ok := itemRefIn(n, now, untaken); ok {
			add(id)
		}
	}
	for _, c := range n.Content {
		if err := walkYAML(c, depth+1, now, untaken, add); err != nil {
			return err
		}
	}
	return nil
}

// itemRefIn reads a mapping as an item: its `bauble` id, unless it is a
// find left lying untaken past untaken (when untaken is not 0).
func itemRefIn(m *yaml.Node, now time.Time, untaken time.Duration) (string, bool) {
	it := items.Item{}
	for i := 0; i+1 < len(m.Content); i += 2 {
		k, v := m.Content[i], m.Content[i+1]
		if v.Kind != yaml.ScalarNode {
			continue
		}
		switch k.Value {
		case `bauble`:
			it.Bauble = v.Value
		case `baubleleftat`:
			it.BaubleLeftAt, _ = strconv.ParseInt(v.Value, 10, 64)
		}
	}
	if _, ok := seqOf(it.Bauble); !ok {
		return ``, false
	}
	if untaken > 0 {
		if age, lying := it.BaubleUntakenFor(now); lying && age >= untaken {
			return ``, false
		}
	}
	return it.Bauble, true
}
```

- [ ] **Step 4: Run them to see them pass**

Run: `go test ./internal/baubles/ -run TestDiskRefs -v`
Expected: PASS, all subtests.

- [ ] **Step 5: Null probes**

(a) Change `sweepSkipDirs` to `map[string]bool{`baubles`: true}`: `TestDiskRefsFindsItemsInEverySaveShape` must FAIL listing `B0000011`. Restore.
(b) In `scanDisk`, change `return fmt.Errorf(`parse %s: %w`, rel, perr)` to `return nil`: the `a file that names a bauble must parse` subtest must FAIL. Restore. Rerun: PASS.

- [ ] **Step 6: Commit**

```bash
git -C C:/tmp/dogmud-bauble-sweep add internal/baubles/sweep_disk.go internal/baubles/sweep_disk_test.go
git -C C:/tmp/dogmud-bauble-sweep commit -m "feat(baubles): DiskRefs, every bauble reference in the save files

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 8: Live sources, `runSweep`, status and the sweeper

**Files:**
- Modify: `internal/baubles/sweep.go`
- Test: `internal/baubles/sweep_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/baubles/sweep_test.go`:

```go
package baubles

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// withLiveSources replaces the registered live sources for one test.
func withLiveSources(t *testing.T, sources map[string]LiveWalk) {
	t.Helper()
	liveMu.Lock()
	saved := liveSources
	liveSources = map[string]LiveWalk{}
	for name, walk := range sources {
		liveSources[name] = walk
	}
	liveMu.Unlock()
	t.Cleanup(func() {
		liveMu.Lock()
		liveSources = saved
		liveMu.Unlock()
	})
}

// holding is a live source whose items point at ids.
func holding(ids ...string) LiveWalk {
	return func(visit func(*items.Item)) {
		for _, id := range ids {
			it := items.Item{ItemId: items.BaubleItemId, Bauble: id}
			visit(&it)
		}
	}
}

// A sweep that could not see everything applies nothing.
func TestRunSweepFailsClosed(t *testing.T) {
	for name, tc := range map[string]struct {
		sources map[string]LiveWalk
		files   map[string]string
		readErr bool
		want    string
	}{
		`no live sources`: {want: `no live item sources`},
		`a live source panics`: {
			sources: map[string]LiveWalk{`rooms`: func(func(*items.Item)) { panic(`boom`) }},
			want:    `live source rooms: panic: boom`,
		},
		`a save names a bauble and does not parse`: {
			sources: map[string]LiveWalk{`rooms`: holding()},
			files:   map[string]string{`users/5.yaml`: "character:\n  items:\n  - bauble: B0000001\n   bad: [\n"},
			want:    `parse users/5.yaml`,
		},
		`a save cannot be read`: {
			sources: map[string]LiveWalk{`rooms`: holding()},
			files:   map[string]string{`users/6.yaml`: "userid: 6\n"},
			readErr: true,
			want:    `read users/6.yaml`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			SetDirForTest(t.TempDir())
			t.Cleanup(func() { items.SetBaubleResolver(nil) })
			withLiveSources(t, tc.sources)
			root := t.TempDir()
			writeDataFiles(t, root, tc.files)
			if tc.readErr {
				orig := sweepReadFile
				sweepReadFile = func(string) ([]byte, error) { return nil, errors.New(`permission denied`) }
				t.Cleanup(func() { sweepReadFile = orig })
			}
			now := time.Now().UTC()
			id := sweepRecord(t, `Doomed If Pruned`, func(r *Record) {
				r.FoundAt, r.UnseenSweeps = now.Add(-400*24*time.Hour), 1
			})

			st := runSweep(now, root)
			if st.OK || !strings.Contains(st.Err, tc.want) {
				t.Fatalf("status %+v, want a failure containing %q", st, tc.want)
			}
			if r, ok := Get(id); !ok || r.UnseenSweeps != 1 {
				t.Fatalf("a failed sweep applies nothing: %+v %v", r, ok)
			}
			if got := LastSweep(); got.OK || got.Err != st.Err {
				t.Fatalf("LastSweep %+v, want the failure", got)
			}
		})
	}
}

// Live and disk references both keep a record; a record neither holds goes
// on the second sweep. The status counts what happened.
func TestRunSweepSeesLiveAndDisk(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	setBaubleConfig(t, func(b *configs.Balance) { b.BaubleCatalogKeepDays = 30 })
	now := time.Now().UTC()
	old := now.Add(-60 * 24 * time.Hour)
	live := sweepRecord(t, `Held In Hand`, func(r *Record) { r.FoundAt = old })
	disk := sweepRecord(t, `Kept In A Bank`, func(r *Record) { r.FoundAt = old })
	gone := sweepRecord(t, `Junked Long Ago`, func(r *Record) { r.FoundAt = old })
	withLiveSources(t, map[string]LiveWalk{`users`: holding(live)})
	root := t.TempDir()
	writeDataFiles(t, root, map[string]string{
		`users/9.yaml`: "itemstorage:\n  slots:\n  - item:\n      itemid: 900\n      bauble: " + disk + "\n    count: 1\n",
	})

	if st := runSweep(now, root); !st.OK || st.Referenced != 2 || st.Pruned != 0 {
		t.Fatalf("first sweep %+v", st)
	}
	st := runSweep(now.Add(time.Hour), root)
	if !st.OK || st.Referenced != 2 || st.Pruned != 1 || st.Records != 2 || st.Files != 1 || st.Parsed != 1 {
		t.Fatalf("second sweep %+v, want 2 referenced, 1 pruned, 2 left, 1 file read and parsed", st)
	}
	if _, ok := Get(gone); ok {
		t.Fatal("the record nothing holds is pruned")
	}
	if got := LastSweep(); !got.OK || got.Pruned != 1 {
		t.Fatalf("LastSweep %+v", got)
	}
}

// A crash rolled the seller's save back past the sale: the bauble is in the
// pack again. Its record stays, still marked sold, for as long as it is
// held; once it is gone for good it goes like any other.
func TestRunSweepKeepsARolledBackSale(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	setBaubleConfig(t, func(b *configs.Balance) { b.BaubleCatalogKeepDays = 30 })
	now := time.Now().UTC()
	longAgo := now.Add(-40 * 24 * time.Hour)
	id := sweepRecord(t, `Painted Wooden Horse`, func(r *Record) {
		r.FoundAt, r.Status, r.SoldAt, r.SoldValue = longAgo.Add(-24*time.Hour), StatusSold, longAgo, 6
	})
	withLiveSources(t, map[string]LiveWalk{`users`: holding()})
	root := t.TempDir()
	writeDataFiles(t, root, map[string]string{`users/3.yaml`: "character:\n  items:\n  - itemid: 900\n    bauble: " + id + "\n"})

	for i := 0; i < 3; i++ {
		if st := runSweep(now.Add(time.Duration(i)*time.Hour), root); !st.OK || st.Pruned != 0 {
			t.Fatalf("sweep %d: %+v", i, st)
		}
	}
	r, ok := Get(id)
	if !ok || r.Status != StatusSold || r.UnseenSweeps != 0 {
		t.Fatalf("a held sold record is kept and left sold: %+v %v", r, ok)
	}

	writeDataFiles(t, root, map[string]string{`users/3.yaml`: "character:\n  items: []\n"})
	runSweep(r.LastSeenAt.Add(time.Hour), root)
	runSweep(r.LastSeenAt.Add(KeepDuration()), root)
	if _, ok := Get(id); ok {
		t.Fatal("gone for the keep window after its last sighting, it is pruned")
	}
}

// An empty catalog has nothing to look for: no scan.
func TestRunSweepSkipsAnEmptyCatalog(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	withLiveSources(t, nil)
	if st := runSweep(time.Now().UTC(), t.TempDir()); !st.OK || !st.Skipped || st.Files != 0 {
		t.Fatalf("status %+v, want OK and skipped", st)
	}
}

// Sweeps and ordinary catalog writes at once: run under -race.
func TestSweepRacesWithCatalogWrites(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	now := time.Now().UTC()
	ids := []string{}
	for i := 0; i < 20; i++ {
		ids = append(ids, sweepRecord(t, `Glass Marble`, func(r *Record) { r.FoundAt = now.Add(-60 * 24 * time.Hour) }))
	}
	withLiveSources(t, map[string]LiveWalk{`users`: holding(ids[:10]...)})
	root := t.TempDir()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			runSweep(now.Add(time.Duration(i)*time.Hour), root)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			v := i%5 + 1
			Update(ids[i%20], func(r *Record) { r.Value = v })
			Get(ids[(i+7)%20])
		}
	}()
	wg.Wait()
	for _, id := range ids[:10] {
		if _, ok := Get(id); !ok {
			t.Fatalf("held record %s was pruned", id)
		}
	}
}

func TestSweepIntervalFromConfig(t *testing.T) {
	setBaubleConfig(t, func(b *configs.Balance) { b.BaubleSweepHours = 3 })
	if got := SweepInterval(); got != 3*time.Hour {
		t.Fatalf("interval %v, want 3h", got)
	}
}

// The loop runs a sweep at once, then again every interval, until stopped.
func TestSweepLoopRunsUntilStopped(t *testing.T) {
	stop := make(chan struct{})
	runs := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		sweepLoop(stop, func() time.Duration { return time.Millisecond }, func() {
			select {
			case runs <- struct{}{}:
			default:
			}
		})
		close(done)
	}()
	for i := 0; i < 3; i++ {
		select {
		case <-runs:
		case <-time.After(2 * time.Second):
			t.Fatal("the loop stopped running sweeps")
		}
	}
	close(stop)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the loop did not stop")
	}
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test ./internal/baubles/ -run "TestRunSweep|TestSweep"`
Expected: FAIL to compile: `undefined: LiveWalk`, `liveMu`, `liveSources`, `runSweep`, `LastSweep`, `SweepInterval`, `sweepLoop`.

- [ ] **Step 3: Implement**

In `internal/baubles/sweep.go`, replace the import block with:

```go
import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)
```

and append to the end of the file:

```go
// LiveWalk visits every item one live store holds. It runs under the mud
// lock and must only read.
type LiveWalk func(visit func(*items.Item))

var (
	liveMu      sync.Mutex
	liveSources = map[string]LiveWalk{}
)

// RegisterLiveSource names a store of live items for the sweep; the same
// name replaces the earlier walk. The core stores are registered by the
// main package (bauble_sweep.go), the auction house by its module.
func RegisterLiveSource(name string, walk LiveWalk) {
	liveMu.Lock()
	defer liveMu.Unlock()
	liveSources[name] = walk
}

// LiveSourceNames lists the registered live stores, sorted.
func LiveSourceNames() []string {
	liveMu.Lock()
	defer liveMu.Unlock()
	out := make([]string, 0, len(liveSources))
	for name := range liveSources {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

type namedWalk struct {
	name string
	walk LiveWalk
}

func liveSourceList() []namedWalk {
	liveMu.Lock()
	defer liveMu.Unlock()
	out := make([]namedWalk, 0, len(liveSources))
	for name, walk := range liveSources {
		out = append(out, namedWalk{name: name, walk: walk})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].name < out[b].name })
	return out
}

// sweepLock and sweepUnlock hold the world still while the live stores are
// walked.
var sweepLock, sweepUnlock = util.LockMud, util.UnlockMud

var errNoLiveSources = errors.New(`no live item sources are registered, so the live world cannot be seen`)

// collectLive walks every live store under the mud lock.
func collectLive(add func(id string)) error {
	sources := liveSourceList()
	if len(sources) == 0 {
		return errNoLiveSources
	}
	sweepLock()
	defer sweepUnlock()
	for _, s := range sources {
		if err := walkLiveSource(s, add); err != nil {
			return err
		}
	}
	return nil
}

func walkLiveSource(s namedWalk, add func(id string)) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf(`live source %s: panic: %v`, s.name, r)
		}
	}()
	s.walk(func(it *items.Item) {
		if it != nil && it.Bauble != `` {
			add(it.Bauble)
		}
	})
	return nil
}

// SweepStatus is what the last sweep did, for `bauble status` and the log.
type SweepStatus struct {
	At         time.Time     // when it ran; zero if no sweep has run yet
	OK         bool          // false: it failed closed and applied nothing
	Err        string        // why it failed
	Skipped    bool          // the catalog was empty: nothing to look for
	Records    int           // records in the catalog afterwards
	Referenced int           // records something still points at
	Pruned     int           // records it removed
	Files      int           // data files it read
	Parsed     int           // of them, files that name a bauble
	Live       time.Duration // time holding the mud lock
	Disk       time.Duration // time reading data files
}

var (
	statusMu   sync.Mutex
	lastSweep  SweepStatus
	sweepRunMu sync.Mutex
)

// LastSweep is what the most recent sweep did.
func LastSweep() SweepStatus {
	statusMu.Lock()
	defer statusMu.Unlock()
	return lastSweep
}

// SweepInterval is the time between sweeps (Balance.BaubleSweepHours).
func SweepInterval() time.Duration {
	h := int(configs.GetBalanceConfig().BaubleSweepHours)
	if h < 1 {
		h = 1
	}
	return time.Duration(h) * time.Hour
}

// RunSweep runs one complete sweep now against DataFiles and returns what
// it did. It takes the mud lock for the live half, so never call it while
// holding that lock (a command handler does): StartSweeper runs it on its
// own goroutine.
func RunSweep(now time.Time) SweepStatus {
	return runSweep(now, configs.GetFilePathsConfig().DataFiles.String())
}

func runSweep(now time.Time, root string) (st SweepStatus) {
	sweepRunMu.Lock()
	defer sweepRunMu.Unlock()
	st.At = now
	defer func() {
		if r := recover(); r != nil {
			st.OK = false
			st.Err = fmt.Sprintf(`panic: %v`, r)
		}
		statusMu.Lock()
		lastSweep = st
		statusMu.Unlock()
		logSweep(st)
	}()

	st.Records = Count()
	if st.Records == 0 {
		st.OK, st.Skipped = true, true
		return st
	}
	refs := map[string]bool{}
	add := func(id string) { refs[id] = true }

	start := time.Now()
	err := collectLive(add)
	st.Live = time.Since(start)
	if err != nil {
		st.Err = err.Error()
		return st
	}
	start = time.Now()
	st.Files, st.Parsed, err = scanDisk(root, now, add)
	st.Disk = time.Since(start)
	if err != nil {
		st.Err = err.Error()
		return st
	}
	st.Referenced, st.Pruned = applySweep(now, refs, KeepDuration())
	st.Records = Count()
	st.OK = true
	return st
}

func logSweep(st SweepStatus) {
	switch {
	case !st.OK:
		mudlog.Error(`baubles`, `action`, `sweep`, `result`, `failed; nothing pruned`, `error`, st.Err)
	case st.Skipped:
		mudlog.Info(`baubles`, `action`, `sweep`, `result`, `catalog empty`)
	default:
		mudlog.Info(`baubles`, `action`, `sweep`, `records`, st.Records, `referenced`, st.Referenced, `pruned`, st.Pruned,
			`files`, st.Files, `parsed`, st.Parsed, `live`, st.Live.String(), `disk`, st.Disk.String())
	}
}

// sweepLoop runs a sweep at once, then again every interval, until stop
// closes.
func sweepLoop(stop <-chan struct{}, interval func() time.Duration, run func()) {
	for {
		run()
		t := time.NewTimer(interval())
		select {
		case <-stop:
			t.Stop()
			return
		case <-t.C:
		}
	}
}

var (
	sweeperMu   sync.Mutex
	sweeperStop chan struct{}
	sweeperDone chan struct{}
)

// StartSweeper runs the catalog sweep at once and then every
// SweepInterval, on its own goroutine. Call once at boot, after the world
// is loaded and the live sources are registered. A second call does
// nothing.
func StartSweeper() {
	sweeperMu.Lock()
	defer sweeperMu.Unlock()
	if sweeperStop != nil {
		return
	}
	stop, done := make(chan struct{}), make(chan struct{})
	sweeperStop, sweeperDone = stop, done
	go func() {
		defer close(done)
		sweepLoop(stop, SweepInterval, func() { RunSweep(time.Now().UTC()) })
	}()
}

// StopSweeper stops the sweeper and waits (up to 30 seconds) for a sweep in
// progress to finish its writes. Call at shutdown before SaveAll, never
// while holding the mud lock (a sweep in progress may be waiting on it).
func StopSweeper() {
	sweeperMu.Lock()
	stop, done := sweeperStop, sweeperDone
	sweeperStop, sweeperDone = nil, nil
	sweeperMu.Unlock()
	if stop == nil {
		return
	}
	close(stop)
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		mudlog.Warn(`baubles`, `action`, `stop sweeper`, `result`, `a sweep was still running after 30s; not waiting`)
	}
}
```

- [ ] **Step 4: Run the package tests**

Run: `go test ./internal/baubles/`
Expected: `ok`.

- [ ] **Step 5: Null probes**

(a) In `runSweep`, change the disk-error branch `st.Err = err.Error(); return st` to fall through (delete the `return st` line after `st.Err = err.Error()` in the disk branch): `TestRunSweepFailsClosed/a_save_names_a_bauble_and_does_not_parse` must FAIL (`a failed sweep applies nothing`, UnseenSweeps 2). Restore.
(b) In `walkLiveSource`, delete the `defer func() { ... recover ... }()`: the `a live source panics` subtest must FAIL by panicking. Restore. Rerun: `ok`.

- [ ] **Step 6: Commit**

```bash
git -C C:/tmp/dogmud-bauble-sweep add internal/baubles/sweep.go internal/baubles/sweep_test.go
git -C C:/tmp/dogmud-bauble-sweep commit -m "feat(baubles): the catalog sweep: live sources, fail-closed runs, status, sweeper

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 9: Loading a room removes expired untaken finds

**Files:**
- Modify: `internal/rooms/save_and_load.go` (end of `LoadRoomInstance`)
- Test: `internal/rooms/baubles_untaken_load_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/rooms/baubles_untaken_load_test.go`:

```go
package rooms

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"gopkg.in/yaml.v2"
)

// A find that has lain untaken past its limit is gone the moment its room
// is loaded from its instance file, before anything (a mob wandering in,
// which never prepares the room) can pick it up. The bauble sweep stops
// counting such a find as a reference and may prune its record, so this
// must hold.
func TestLoadRoomInstanceRemovesExpiredUntakenFinds(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	tempDir := t.TempDir()
	prev := configs.GetFilePathsConfig()
	if err := configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": tempDir}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": prev.DataFiles.String()})
	}()

	template := &Room{RoomId: 90103, Zone: "test_zone", Title: "Untaken Test", Description: "A test room.", Exits: map[string]exit.RoomExit{}}
	templateYAML, err := yaml.Marshal(template)
	if err != nil {
		t.Fatal(err)
	}
	templatePath := filepath.Join(tempDir, "rooms", "test_zone", "90103.yaml")
	if err := os.MkdirAll(filepath.Dir(templatePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(templatePath, templateYAML, 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	instance := map[string]any{
		"items": []map[string]any{
			{"itemid": items.BaubleItemId, "bauble": "B0000001", "baubleleftat": now.Add(-25 * time.Hour).Unix()},
			{"itemid": items.BaubleItemId, "bauble": "B0000002", "baubleleftat": now.Add(-time.Hour).Unix()},
		},
	}
	instanceYAML, err := yaml.Marshal(instance)
	if err != nil {
		t.Fatal(err)
	}
	instancePath := filepath.Join(tempDir, "rooms.instances", "test_zone", "90103.yaml")
	if err := os.MkdirAll(filepath.Dir(instancePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(instancePath, instanceYAML, 0o644); err != nil {
		t.Fatal(err)
	}
	roomManager.setCachedFilePath(90103, "test_zone/90103.yaml")

	loaded := LoadRoomInstance(90103)
	if loaded == nil {
		t.Fatal("room did not load")
	}
	if len(loaded.Items) != 1 || loaded.Items[0].Bauble != "B0000002" {
		t.Fatalf("items %+v, want only the young find B0000002", loaded.Items)
	}
}
```

- [ ] **Step 2: Run it to see it fail**

Run: `go test ./internal/rooms/ -run TestLoadRoomInstanceRemovesExpiredUntakenFinds`
Expected: FAIL: `items [...] want only the young find B0000002` (both finds present).

- [ ] **Step 3: Implement**

In `internal/rooms/save_and_load.go`, at the end of `LoadRoomInstance`, replace

```go
	room.applyDefusedExits()

	return room
}
```

with

```go
	room.applyDefusedExits()

	// A find left lying untaken past its limit is gone before anything can
	// touch the room, not only before a visitor sees it (Prepare): a mob
	// wandering in loads a room without preparing it. The bauble catalog
	// sweep stops counting such a find in a room file as a reference
	// (internal/baubles/sweep_disk.go), so its record may already be pruned.
	room.removeUntakenBaubles(time.Now())

	return room
}
```

(`time` is already imported by this file; `SaveAllRooms` uses it.)

- [ ] **Step 4: Run the rooms tests**

Run: `go test ./internal/rooms/ -run "TestLoadRoomInstance|TestRemoveUntakenBaubles"`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git -C C:/tmp/dogmud-bauble-sweep add internal/rooms/save_and_load.go internal/rooms/baubles_untaken_load_test.go
git -C C:/tmp/dogmud-bauble-sweep commit -m "fix(rooms): a room loaded from its save drops expired untaken finds at once

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 10: Wire the sweep in; registration and real-type disk tests

**Files:**
- Create: `bauble_sweep.go`, `bauble_sweep_test.go` (repo root)
- Modify: `modules/auctions/auctions.go` (`init`), `main.go` (before `mudlog.Info("Server Ready"...)`, and `main.go:611`), `copyover.go` (before `baubles.SaveAll()`)

- [ ] **Step 1: Write the failing tests**

Create `bauble_sweep_test.go`:

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/guilds"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/sealedcrate"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/modules/auctions"
	"gopkg.in/yaml.v2"
)

// Every live root the guards check is registered as a sweep source, and
// nothing else is.
func TestBaubleSweepSourcesMatchTheGuardedRoots(t *testing.T) {
	registerBaubleSweepSources() // the auction house registers itself in its init
	want := []string{}
	for _, r := range sweepRoots() {
		want = append(want, r.name)
	}
	sort.Strings(want)
	if got := baubles.LiveSourceNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("live sources %v, want %v", got, want)
	}
}

func sweepBauble(n int) items.Item {
	return items.Item{ItemId: items.BaubleItemId, Bauble: fmt.Sprintf(`B%07d`, n)}
}

func writeSweepFile(t *testing.T, root string, rel string, v any) {
	t.Helper()
	data, err := yaml.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// Each store's real save shape, written with the library and layout its
// store uses, is read by the sweep's disk half.
func TestBaubleSweepReadsEveryStoreFromDisk(t *testing.T) {
	root := t.TempDir()
	inboxItem := sweepBauble(4)
	writeSweepFile(t, root, `users/7.yaml`, &users.UserRecord{
		UserId: 7, Username: `sweeper`,
		Character: &characters.Character{
			Name:      `Sweeper`,
			Items:     []items.Item{sweepBauble(1)},
			Equipment: characters.Worn{Neck: sweepBauble(2)},
		},
		ItemStorage: users.Storage{Slots: []users.StorageSlot{{Item: sweepBauble(3), Count: 1}}},
		Inbox:       users.Inbox{{FromName: `a friend`, Item: &inboxItem}},
	})
	writeSweepFile(t, root, `users/7.alts.yaml`, []characters.Character{{Name: `Alt`, Items: []items.Item{sweepBauble(5)}}})
	writeSweepFile(t, root, `rooms.instances/test_zone/1.yaml`, &rooms.Room{
		RoomId:     1,
		Items:      []items.Item{sweepBauble(6)},
		Stash:      []items.Item{sweepBauble(7)},
		Containers: map[string]rooms.Container{`chest`: {Items: []items.Item{sweepBauble(8)}}},
	})
	writeSweepFile(t, root, `mobs.instances/test_zone/1-guard-1.yaml`, &mobs.MobInstanceData{Equipment: &characters.Worn{Weapon: sweepBauble(9)}})
	writeSweepFile(t, root, `shops/test_zone/5-room1.yaml`, &shops.ShopInventory{AffixedStock: []shops.AffixedStockEntry{{Item: sweepBauble(10), Price: 5}}})
	writeSweepFile(t, root, `guilds/tst.yaml`, &guilds.Guild{Tag: `TST`, Name: `Testers`, Vault: []items.Item{sweepBauble(11)}})
	crate := sealedcrate.New(1, 5)
	crate.Add(sweepBauble(12))
	if err := sealedcrate.SaveTo(filepath.Join(root, `crates`, `1-test.yaml`), crate); err != nil {
		t.Fatal(err)
	}
	writeSweepFile(t, root, `plugin-data/auctions-v1.0/auctionhistory.plugin.dat`, &auctions.AuctionManager{
		ActiveAuction: &auctions.AuctionItem{ItemData: sweepBauble(13)},
		SeizedQueue:   []auctions.SeizedLot{{Item: sweepBauble(14), Count: 1}},
	})

	refs, _, _, err := baubles.DiskRefs(root, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 14; n++ {
		if id := fmt.Sprintf(`B%07d`, n); !refs[id] {
			t.Errorf("%s not found on disk", id)
		}
	}
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test . -run "TestBaubleSweepSourcesMatchTheGuardedRoots|TestBaubleSweepReadsEveryStoreFromDisk"`
Expected: FAIL to compile, `undefined: registerBaubleSweepSources`.

- [ ] **Step 3: Register the sources and start the sweeper**

Create `bauble_sweep.go`:

```go
package main

import (
	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/guilds"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// registerBaubleSweepSources tells the bauble catalog sweep
// (internal/baubles/sweep.go) where the live world keeps items. Each walk
// runs under the mud lock and only reads. The auction house registers its
// own in modules/auctions. Every store here has a WalkItems kept complete by
// TestItemWalkersVisitEveryItemField; a new store of items needs a source
// here and a root in item_walker_guard_test.go, or
// TestEveryItemHolderIsASweepRootOrTransient fails.
func registerBaubleSweepSources() {
	baubles.RegisterLiveSource(`users`, func(visit func(*items.Item)) {
		for _, u := range users.GetAllLoadedUsers() {
			u.WalkItems(visit)
		}
	})
	baubles.RegisterLiveSource(`rooms`, func(visit func(*items.Item)) {
		for _, r := range rooms.LoadedRooms() {
			r.WalkItems(visit)
		}
	})
	baubles.RegisterLiveSource(`mobs`, func(visit func(*items.Item)) {
		for _, id := range mobs.GetAllMobInstanceIds() {
			if m := mobs.GetInstance(id); m != nil {
				m.WalkItems(visit)
			}
		}
	})
	baubles.RegisterLiveSource(`shops`, func(visit func(*items.Item)) {
		for _, s := range shops.AllShops() {
			s.WalkItems(visit)
		}
	})
	baubles.RegisterLiveSource(`guilds`, func(visit func(*items.Item)) {
		for _, g := range guilds.All() {
			g.WalkItems(visit)
		}
	})
}
```

In `modules/auctions/auctions.go` `init()`, after

```go
	events.RegisterListener(events.StorageItemSeized{}, a.storageSeizedHandler)
```

add:

```go

	// The bauble catalog sweep reads the lot on the block and the seized lots
	// through this (internal/baubles/sweep.go).
	baubles.RegisterLiveSource(`auctions`, func(visit func(*items.Item)) {
		a.auctionMgr.WalkItems(visit)
	})
```

(`baubles` and `items` are already imported there, fact: `auctions.go:12,14`.)

In `main.go`, directly before `mudlog.Info("Server Ready", "Time Taken", time.Since(serverStartTime))`, add:

```go
	// Bauble catalog sweep: once now, then every Balance.BaubleSweepHours,
	// prune the records no item points at any more (internal/baubles/sweep.go).
	registerBaubleSweepSources()
	baubles.StartSweeper()

```

and replace

```go
	baubles.SaveAll()         // retries any catalog write that failed; the rest is already on disk
```

with

```go
	baubles.StopSweeper()     // a sweep in progress finishes its writes first
	baubles.SaveAll()         // retries any catalog write that failed; the rest is already on disk
```

In `copyover.go`, replace

```go
	warehouse.SaveAll()
	baubles.SaveAll()
	apiframework.SaveBudget()
```

with

```go
	warehouse.SaveAll()
	// The bauble sweeper is not stopped here: copyover holds the mud lock a
	// sweep may be waiting on, so waiting for it would deadlock. Its catalog
	// writes are atomic (util.Save), so the re-exec cutting a sweep off loses
	// nothing; the next boot sweeps again.
	baubles.SaveAll()
	apiframework.SaveBudget()
```

- [ ] **Step 4: Run gofmt, build and the tests**

Run: `gofmt -l main.go copyover.go bauble_sweep.go bauble_sweep_test.go modules/auctions/` (expected: nothing; if `main.go` is listed, run `gofmt -w main.go`).
Run: `go build ./... && go test . -run "TestBaubleSweep|TestItemWalkersVisitEveryItemField|TestEveryItemHolderIsASweepRootOrTransient" -v`
Expected: PASS.

- [ ] **Step 5: Null probe**

Comment out the `guilds` registration in `bauble_sweep.go`: `TestBaubleSweepSourcesMatchTheGuardedRoots` must FAIL (`live sources [auctions mobs rooms shops users], want [auctions guilds mobs rooms shops users]`). Restore; PASS.

- [ ] **Step 6: Commit**

```bash
git -C C:/tmp/dogmud-bauble-sweep add bauble_sweep.go bauble_sweep_test.go modules/auctions/auctions.go main.go copyover.go
git -C C:/tmp/dogmud-bauble-sweep commit -m "feat: run the bauble catalog sweep at boot and on its interval

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 11: `bauble status` shows the sweep; a sold bauble sells again

**Files:**
- Modify: `internal/usercommands/admin.bauble.go` (`baubleStatus`, line 158)
- Test: `internal/usercommands/admin.bauble_test.go`, `internal/actions/sell_bauble_test.go`

- [ ] **Step 1: Write the failing test for the status line**

Append to `internal/usercommands/admin.bauble_test.go` (add `"time"` to its imports):

```go
// `bauble status` says when the catalog sweep last ran and what it did, or
// why it pruned nothing.
func TestBaubleSweepLine(t *testing.T) {
	every := 6 * time.Hour
	at := time.Date(2026, 9, 29, 14, 2, 0, 0, time.UTC)
	for name, tc := range map[string]struct {
		st   baubles.SweepStatus
		want []string
	}{
		`never`:  {baubles.SweepStatus{}, []string{`not run yet`, `every 6 hours`}},
		`failed`: {baubles.SweepStatus{At: at, Err: `parse users/5.yaml: bad`}, []string{`failed`, `2026-09-29 14:02 UTC`, `nothing was pruned`, `parse users/5.yaml: bad`}},
		`empty`:  {baubles.SweepStatus{At: at, OK: true, Skipped: true}, []string{`catalog was empty`}},
		`ran`: {baubles.SweepStatus{At: at, OK: true, Records: 40, Referenced: 31, Pruned: 3, Files: 412, Parsed: 17,
			Disk: 180 * time.Millisecond, Live: 4 * time.Millisecond}, []string{`40 records`, `31 still held`, `3 pruned`, `412 files`, `17 name a bauble`, `180ms`, `4ms`, `Every 6 hours`}},
	} {
		line := baubleSweepLine(tc.st, every)
		for _, w := range tc.want {
			assert.Contains(t, line, w, name)
		}
	}
}
```

- [ ] **Step 2: Run it to see it fail**

Run: `go test ./internal/usercommands/ -run TestBaubleSweepLine`
Expected: FAIL to compile, `undefined: baubleSweepLine`.

- [ ] **Step 3: Implement**

In `internal/usercommands/admin.bauble.go`, replace

```go
	fmt.Fprintf(&b, "Catalog: %d records.\r\n", baubles.Count())
	user.SendText(messaging.CategorySystem, b.String())
	return true, nil
}
```

with

```go
	fmt.Fprintf(&b, "Catalog: %d records.\r\n", baubles.Count())
	b.WriteString(baubleSweepLine(baubles.LastSweep(), baubles.SweepInterval()))
	user.SendText(messaging.CategorySystem, b.String())
	return true, nil
}

// baubleSweepLine is `bauble status`'s line about the catalog sweep: when it
// last ran and what it did, or why it pruned nothing.
func baubleSweepLine(st baubles.SweepStatus, every time.Duration) string {
	hours := int(every.Hours())
	when := st.At.UTC().Format(`2006-01-02 15:04 MST`)
	switch {
	case st.At.IsZero():
		return fmt.Sprintf("Sweep: not run yet; it runs at boot and every %d hours.\r\n", hours)
	case !st.OK:
		return fmt.Sprintf("Sweep: <ansi fg=\"red\">failed</ansi> at %s, so nothing was pruned: %s\r\n", when, st.Err)
	case st.Skipped:
		return fmt.Sprintf("Sweep: %s, the catalog was empty. Every %d hours.\r\n", when, hours)
	default:
		return fmt.Sprintf("Sweep: %s, %d records, %d still held somewhere, %d pruned. Read %d files (%d name a bauble) in %s; held the world %s. Every %d hours.\r\n",
			when, st.Records, st.Referenced, st.Pruned, st.Files, st.Parsed,
			st.Disk.Round(time.Millisecond), st.Live.Round(time.Millisecond), hours)
	}
}
```

(`time` is already imported by `admin.bauble.go`.)

- [ ] **Step 4: Pin the resale**

Append to `internal/actions/sell_bauble_test.go` (add `"time"` to its imports):

```go
// A crash can roll a seller's save back past a sale: the bauble is in the
// pack again while its record says sold. It sells again like any bauble, and
// the new sale is recorded (the catalog sweep keeps the record while it is
// held: internal/baubles/sweep.go).
func TestSell_Bauble_ASoldRecordSellsAgain(t *testing.T) {
	seedBaubleSale(t)
	defer seedSellRoom(t)()
	defer seedSellMerchant(t, 1000)()

	seller := newSellerActor(t, true)
	char := seller.GetCharacter()
	b := newBauble(t, "Painted Wooden Horse", "horse", 12, baubles.StatusFallback)
	longAgo := time.Now().UTC().Add(-40 * 24 * time.Hour)
	baubles.Update(b.Bauble, func(r *baubles.Record) {
		r.Status, r.SoldAt, r.SoldValue = baubles.StatusSold, longAgo, 6
	})
	require.True(t, char.StoreItem(b))

	res := Sell(seller, SellOptions{ItemName: "horse", Quantity: 1})

	require.Equal(t, SellStopSoldAll, res.Reason, "res=%+v", res)
	assert.Equal(t, 6, char.Gold, "paid from the catalog, as for any bauble")
	rec, _ := baubles.Get(b.Bauble)
	assert.True(t, rec.SoldAt.After(longAgo), "the new sale is recorded")
}
```

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/usercommands/ -run "TestBaubleSweepLine|TestAdminBauble" && go test ./internal/actions/ -run TestSell_Bauble`
Expected: `ok` for both. `TestSell_Bauble_ASoldRecordSellsAgain` pins existing behaviour, so it passes on first run.

- [ ] **Step 6: Null probe for the resale pin**

In `internal/actions/sell_bauble.go` `baubleOfferFor`, directly after `rec, ok := baubles.Get(item.Bauble)` and its `!ok` check, add `if rec.Status == baubles.StatusSold { return BaubleOffer{Refusal: baubleSayUnknown} }`: `TestSell_Bauble_ASoldRecordSellsAgain` must FAIL. Remove it; PASS.

- [ ] **Step 7: Commit**

```bash
git -C C:/tmp/dogmud-bauble-sweep add internal/usercommands/admin.bauble.go internal/usercommands/admin.bauble_test.go internal/actions/sell_bauble_test.go
git -C C:/tmp/dogmud-bauble-sweep commit -m "feat(usercommands): bauble status reports the catalog sweep; pin resale of a rolled-back sale

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 12: Docs

**Files:**
- Modify: `internal/baubles/context.md`, `internal/items/context.md`, `internal/characters/context.md`, `internal/users/context.md`, `internal/rooms/context.md`, `internal/mobs/context.md`, `internal/shops/context.md`, `internal/guilds/context.md`, `internal/sealedcrate/context.md`, `modules/auctions/context.md`, `docs/PATCH_NOTES.md`

- [ ] **Step 1: `internal/baubles/context.md`**

Replace the `catalog.go` bullet (from `- **catalog.go**: the in-memory catalog, write-through to disk, and the` through `` `corpus.promoted.yaml` survives. ``) with:

```markdown
- **catalog.go**: the in-memory catalog, write-through to disk, and the
  resolver installed into `internal/items`. Disk writes (`persistShard`,
  `persistMeta`) snapshot under the read lock and write outside it, ordered
  by a separate write mutex, so `Get` never waits on the disk. A per-user
  index of return credits backs `ReturnCredits`. `persistShardPruning`
  writes a shard without the records a predicate marks and only then takes
  them out of memory (persist before publish). `Record.prunableAt(now, keep)`:
  at least `minUnseenSweeps` (2) complete sweeps in a row found nothing
  pointing at the record AND `KeepDuration()` (`Balance.BaubleCatalogKeepDays`,
  30, at least 7 because the sales stats read a week) has passed since
  `Record.lastEvidence` (found, stolen, recognised, returned, sold, vanished,
  or last seen by a sweep); a record with a return credit is never pruned.
  `Load` and `SaveAll` do not prune: only a successful sweep does.
- **sweep.go**: the catalog sweep. `RegisterLiveSource(name, LiveWalk)`
  (the main package registers `users`, `rooms`, `mobs`, `shops`, `guilds` in
  `bauble_sweep.go`; `modules/auctions` registers `auctions`). `runSweep`
  walks every live source under `util.LockMud`, then `scanDisk` off the lock,
  then `applySweep`: a referenced record gets `LastSeenAt=now,
  UnseenSweeps=0`, any other one more unseen sweep, and each changed shard is
  written without its prunable records. Any error (no live source, a source
  that panics, an unreadable file, a file that names a bauble and does not
  parse) fails closed: nothing is applied. `StartSweeper` runs it at boot and
  every `SweepInterval()` (`Balance.BaubleSweepHours`, 6); `StopSweeper` at
  shutdown. `LastSweep()` feeds `bauble status`.
- **sweep_disk.go**: `DiskRefs(root, now)`: every `.yaml` and `.plugin.dat`
  under DataFiles except `baubles/` and `economy/`; a file is parsed
  (`yaml.Node`) only if it has a `bauble` key. In `rooms.instances/` a find
  untaken past `UntakenLimit()` is not a reference; `rooms.LoadRoomInstance`
  removes such finds on load.
```

In the API block, replace the line `func Prune(now time.Time) int` with:

```go
type LiveWalk func(visit func(*items.Item))
func RegisterLiveSource(name string, walk LiveWalk)
func LiveSourceNames() []string
type SweepStatus struct { /* At, OK, Err, Skipped, Records, Referenced, Pruned, Files, Parsed, Live, Disk */ }
func RunSweep(now time.Time) SweepStatus // takes the mud lock: never call holding it
func LastSweep() SweepStatus
func SweepInterval() time.Duration
func StartSweeper()
func StopSweeper()
func DiskRefs(root string, now time.Time) (refs map[string]bool, files int, parsed int, err error)
```

After the Rules bullet that starts `- Every change is written through to disk before the call returns`, add:

```markdown
- A record lives as long as something points at it. The sweep is the only
  pruner and it fails closed. A sold record held again (a crash rolled the
  seller back) is seen and kept, and its sale is left as it was: every
  record is sellable, and a save on disk can lag a real sale by one
  autosave. `TestItemWalkersVisitEveryItemField` and
  `TestEveryItemHolderIsASweepRootOrTransient` (repo root) fail when a store
  of items is not walked; a new store needs a `WalkItems`, a live source in
  `bauble_sweep.go` and a root in `item_walker_guard_test.go`.
```

In Consumers, replace

```markdown
- `main.go` (`Load` at boot, `SaveAll` at shutdown), `copyover.go`
  (`SaveAll`).
```

with

```markdown
- `main.go` (`Load` at boot, `StartSweeper` before Server Ready,
  `StopSweeper` then `SaveAll` at shutdown), `bauble_sweep.go`
  (`RegisterLiveSource`), `copyover.go` (`SaveAll`; the sweeper is not
  stopped there, see the comment).
- `modules/auctions` (`RegisterLiveSource` for the auction house).
```

and replace `- \`internal/actions/sell_bauble.go\` (\`Get\`, \`Sellable\`, \`MarkSold\`).` with `` - `internal/actions/sell_bauble.go` (`Get`, `MarkSold`). `` (there is no `Sellable`). In the `admin.bauble.go` consumer line, append `; \`bauble status\` reads \`LastSweep\`, \`SweepInterval\``.

- [ ] **Step 2: The other package docs**

Append a section at the end of each file:

`internal/items/context.md`:

```markdown

## Item walkers

`WalkSlice(s []Item, fn func(*Item))` (walk.go) calls fn with a pointer
into s for each item with ItemId above zero. Every store's `WalkItems` is
built on it: `characters.Character` and `Worn`, `users.UserRecord`,
`rooms.Room`, `mobs.Mob`, `shops.ShopInventory`, `guilds.Guild`,
`sealedcrate.Crate`, `modules/auctions.AuctionManager`. The bauble catalog
sweep (`internal/baubles/sweep.go`) reads every live item through them.
```

`internal/characters/context.md`:

```markdown

## WalkItems (walk_items.go)

`(*Worn).WalkItems` and `(*Character).WalkItems` call fn with a live pointer
to every item a character holds: backpack, component bag, potion
bandolier, every equipment slot, the pet's pack, and each companion's saved
pack and gear. The bauble catalog sweep reads characters through them, and
`MigrateDetunedRangedWeapons` walks with them (it used to miss companions'
gear). A new item field on Character, Worn, Pet or CompanionInfo must be
walked here: `TestItemWalkersVisitEveryItemField` (repo root) fails naming
it otherwise.
```

`internal/users/context.md`:

```markdown

## WalkItems and GetAllLoadedUsers (walk_items.go)

`(*UserRecord).WalkItems` walks the active character, the bank (slots and
the legacy list) and inbox attachments. Alts are not in memory; the bauble
sweep reads `<userid>.alts.yaml` from disk. `GetAllLoadedUsers` returns
every user in memory, zombies included (`GetAllActiveUsers` skips them).
```

`internal/rooms/context.md`:

```markdown

## WalkItems, LoadedRooms, and untaken finds on load

`(*Room).WalkItems` (walk_items.go) walks the floor, the stash, every
container, every corpse (the dead character's gear and its loot) and the
sealed crate. `LoadedRooms()` returns every room in memory, ephemeral ones
included; the caller holds the mud lock. `LoadRoomInstance` removes finds
left untaken past `baubles.UntakenLimit()` as soon as a room is loaded from
its instance file, because the bauble sweep no longer counts them and may
prune their records.
```

`internal/mobs/context.md`:

```markdown

## WalkItems (walk_items.go)

`(*Mob).WalkItems` walks the mob's character (`Character.WalkItems`). A
mob's pack exists only in memory (instance files keep equipment only), so
the bauble sweep sees it through this live walk alone.
```

`internal/shops/context.md`:

```markdown

## WalkItems (walk_items.go)

`(*ShopInventory).WalkItems` walks `AffixedStock`, the only per-instance
items a shop holds; stock entries are counts of an item id.
```

`internal/guilds/context.md`:

```markdown

## WalkItems (walk_items.go)

`(*Guild).WalkItems` walks the vault. The bauble catalog sweep registers
it as the `guilds` live source (`bauble_sweep.go`).
```

`internal/sealedcrate/context.md`:

```markdown

## WalkItems

`(*Crate).WalkItems` walks the crate's items holding its lock; fn must not
call back into the crate. The bauble sweep reaches a crate through
`Room.WalkItems`, and a crate whose room is not attached through its file
under `crates/`.
```

`modules/auctions/context.md`:

```markdown

## WalkItems and the bauble sweep

`(*AuctionManager).WalkItems` (walk_items.go) walks the lot on the block and
the seized lots; past auctions keep names only. `init` registers it with
`baubles.RegisterLiveSource("auctions", ...)`, so the bauble catalog sweep
sees items held by the auction house between saves.
```

- [ ] **Step 3: Patch notes**

In `docs/PATCH_NOTES.md`, insert after the `# DOGMud Patch Notes` line and its blank line:

```markdown
## 2026-09-29: Trinkets that come back

If the server ever has to roll your character back to an earlier save, a
trinket you had already sold can turn up in your pack again. It now keeps
its name and still sells, instead of turning into a nameless Curious
Trinket nobody will buy.

Behind the scenes, the game also tidies away its records of trinkets that
no longer exist anywhere in the world.

```

- [ ] **Step 4: Check the docs name only real symbols**

Run: `python tools/context_md_audit.py internal/baubles internal/items internal/characters internal/users internal/rooms internal/mobs internal/shops internal/guilds internal/sealedcrate modules/auctions`
Expected: nothing reported for the symbols this plan added or named (`WalkSlice`, `WalkItems`, `GetAllLoadedUsers`, `LoadedRooms`, `RegisterLiveSource`, `LiveSourceNames`, `RunSweep`, `LastSweep`, `SweepInterval`, `StartSweeper`, `StopSweeper`, `DiskRefs`, `LiveWalk`, `SweepStatus`). A pre-existing finding in a file this plan did not change is not this task's; note it in the PR body.

Run this check on its own line (it prints matches; `grep -c` would exit 1 on zero and break a chain): `git -C C:/tmp/dogmud-bauble-sweep diff origin/master -- '*.md' | grep '^+' | grep -n $'\u2014\|\u2013'`
Expected: no output (no em or en dash in any added doc line).

- [ ] **Step 5: Commit**

```bash
git -C C:/tmp/dogmud-bauble-sweep add internal/baubles/context.md internal/items/context.md internal/characters/context.md internal/users/context.md internal/rooms/context.md internal/mobs/context.md internal/shops/context.md internal/guilds/context.md internal/sealedcrate/context.md modules/auctions/context.md docs/PATCH_NOTES.md
git -C C:/tmp/dogmud-bauble-sweep commit -m "docs: the bauble catalog sweep and the item walkers

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 13: Full local gate and boot smoke

CI is out of minutes until 10-01, so every gate runs here and its output goes into the PR body.

- [ ] **Step 1: gofmt**

Run (Git Bash): `cd C:/tmp/dogmud-bauble-sweep && gofmt -l internal/ modules/ *.go`
Expected: no output. (Fact: `gofmt -l` can false-positive on Windows for a CRLF working copy of an LF blob; if a file you did not touch is listed, confirm with `git diff --stat` that it is untouched.)

- [ ] **Step 2: build and vet**

Run: `go build ./... && go vet . ./internal/baubles/... ./internal/items/... ./internal/characters/... ./internal/users/... ./internal/rooms/... ./internal/mobs/... ./internal/shops/... ./internal/guilds/... ./internal/sealedcrate/... ./internal/configs/... ./internal/usercommands/... ./internal/actions/... ./modules/auctions/...`
Expected: no output, exit 0.

- [ ] **Step 3: Targeted tests, the repo root included**

Run: `go test . ./internal/baubles/ ./internal/items/ ./internal/characters/ ./internal/users/ ./internal/rooms/ ./internal/mobs/ ./internal/shops/ ./internal/guilds/ ./internal/sealedcrate/ ./internal/configs/ ./internal/actions/ ./internal/usercommands/ ./modules/auctions/`
Expected: every line `ok`.

- [ ] **Step 4: Full suite**

Run: `go test ./... 2>&1 | tail -60` (timeout 20 minutes; `internal/combat` is the slow one).
Expected: no `FAIL`. A failure in a package this branch did not touch: rerun that package alone on `origin/master` in a detached worktree before deciding it is not ours.

- [ ] **Step 5: Lint**

Run: `golangci-lint run --new-from-rev=$(git -C C:/tmp/dogmud-bauble-sweep merge-base HEAD origin/master)`
Expected: `0 issues.`

- [ ] **Step 6: Race on the catalog**

Run: `docker compose -f compose.test.yml run --build --rm test go test -race ./internal/baubles/...`
Expected: `ok  github.com/GoMudEngine/GoMud/internal/baubles`, no `WARNING: DATA RACE`.

- [ ] **Step 7: Boot smoke on private ports**

Create a detached boot worktree (Git Bash):

```bash
git -C C:/tmp/dogmud-bauble-sweep worktree add --detach C:/tmp/dogmud-bauble-boot HEAD
```

With the Write tool, create `C:/tmp/dogmud-bauble-boot/smoke-overrides.yaml`:

```yaml
Network:
  TelnetPort: [33533]
  LocalPort: 9899
  HttpPort: 8391
  HttpsPort: 0
  AIPort: 0
```

`C:/tmp/dogmud-bauble-boot/_datafiles/world/dogmud/baubles/meta.yaml`:

```yaml
schema: 1
next_seq: 3
```

`C:/tmp/dogmud-bauble-boot/_datafiles/world/dogmud/baubles/catalog-0000.yaml` (B0000001: found 90 days ago, already unseen twice, held nowhere: must be pruned; B0000002: held in a room file: must be kept and seen):

```yaml
schema: 1
records:
- id: B0000001
  status: fallback
  name: Smoke Test Pebble
  name_simple: pebble
  description: A smooth grey pebble.
  tier: cheap
  value: 2
  weight_lbs: 0.1
  source: admin
  found_at: 2026-07-01T00:00:00Z
  generator: local
  unseen_sweeps: 2
- id: B0000002
  status: fallback
  name: Smoke Test Button
  name_simple: button
  description: A brass button with a worn crest.
  tier: cheap
  value: 3
  weight_lbs: 0.1
  source: admin
  found_at: 2026-07-01T00:00:00Z
  generator: local
```

`C:/tmp/dogmud-bauble-boot/_datafiles/world/dogmud/rooms.instances/ashwick/4023.yaml` (room 4023 is `_datafiles/world/dogmud/rooms/ashwick/4023.yaml`, Maren's Cottage):

```yaml
items:
- itemid: 900
  bauble: B0000002
```

Build and boot (PowerShell; kill only this PID):

```powershell
Set-Location C:\tmp\dogmud-bauble-boot
go build -o boot-check.exe .
$env:CONFIG_PATH = 'C:\tmp\dogmud-bauble-boot\smoke-overrides.yaml'
$p = Start-Process -FilePath 'C:\tmp\dogmud-bauble-boot\boot-check.exe' -WorkingDirectory 'C:\tmp\dogmud-bauble-boot' -RedirectStandardOutput 'C:\tmp\dogmud-bauble-boot\boot.log' -RedirectStandardError 'C:\tmp\dogmud-bauble-boot\boot.err' -NoNewWindow -PassThru
$deadline = (Get-Date).AddSeconds(180)
while ((Get-Date) -lt $deadline -and -not (Select-String -Path 'C:\tmp\dogmud-bauble-boot\boot.log' -Pattern 'action.{1,4}sweep' -Quiet)) { Start-Sleep -Seconds 2 }
Stop-Process -Id $p.Id
Remove-Item Env:CONFIG_PATH
```

Check (PowerShell):

```powershell
Select-String -Path C:\tmp\dogmud-bauble-boot\boot.log -Pattern 'Starting http server','bind:'
(Select-String -Path C:\tmp\dogmud-bauble-boot\boot.log -Pattern 'Server Ready').Count
(Select-String -Path C:\tmp\dogmud-bauble-boot\boot.log, C:\tmp\dogmud-bauble-boot\boot.err -Pattern '^panic:','goroutine [0-9]+ \[running\]','runtime error').Count
Select-String -Path C:\tmp\dogmud-bauble-boot\boot.log -Pattern 'action.{1,4}sweep'
Select-String -Path C:\tmp\dogmud-bauble-boot\_datafiles\world\dogmud\baubles\catalog-0000.yaml -Pattern 'B0000001','B0000002','last_seen_at'
```

Expected: the http line names port 8391 and there is no `bind:` error (the server really bound its own ports); `Server Ready` count 1; panic count 0; one sweep line with `records` 1, `referenced` 1, `pruned` 1 and a `live` duration under 50 ms (stop and report if it is higher); the shard holds `B0000002` and `last_seen_at`, and no `B0000001`. Do not grep for the bare word `panic` (`GamePlay.MapConsistencyEnforce: panic` is a real config value).

Tear down (PowerShell, since Windows can hold the exe): `Remove-Item -Recurse -Force C:\tmp\dogmud-bauble-boot`, then (Git Bash) `git -C C:/tmp/dogmud-bauble-sweep worktree prune`.

- [ ] **Step 8: Record the evidence**

Save the outputs of Steps 1 to 7 (the `ok` lines, `0 issues.`, the race result, and the sweep log line) to `C:\Users\CALABE~1\AppData\Local\Temp\claude\C--Users-Calabe-Davis-workspace-DOGMud\eada5c36-5618-4580-a525-ba54f2e95ea4\scratchpad\bauble-sweep-gate.txt` for the PR body.

---

### Task 14: PR and merge

- [ ] **Step 1: Confirm the branch is clean and small**

Run (Git Bash): `git -C C:/tmp/dogmud-bauble-sweep status --short` (expected: nothing) and `git -C C:/tmp/dogmud-bauble-sweep diff --stat origin/master...HEAD | tail -1` (expected: about 40 files, well under CI's 300-file lint inversion).

- [ ] **Step 2: Push and open the PR on the fork**

```bash
git -C C:/tmp/dogmud-bauble-sweep push -u origin feature/bauble-prune-sweep
gh pr create --repo pruuk/DOGMud --base master --head feature/bauble-prune-sweep --title "feat(baubles): periodic catalog sweep prunes records no item points at" --body-file C:/Users/CALABE~1/AppData/Local/Temp/claude/C--Users-Calabe-Davis-workspace-DOGMud/eada5c36-5618-4580-a525-ba54f2e95ea4/scratchpad/bauble-sweep-pr.md
```

Before running it, write `bauble-sweep-pr.md` in the scratchpad with: a summary (the owner-ruled sweep; the store list; fail closed; the two guards; `BaubleSweepHours`; the rolled-back sale decision; the untaken-find removal on room load; the detune migration now reaching companions), the plan path `docs/superpowers/plans/2026-09-29-bauble-prune-sweep.md`, a "Local gate (CI out of minutes until 10-01)" section pasting `bauble-sweep-gate.txt`, "No deploy.", and the closing line `🤖 Generated with [Claude Code](https://claude.com/claude-code)`. Read the URL `gh` prints and confirm it says `pruuk/DOGMud`.

- [ ] **Step 3: Merge past red CI**

```bash
gh pr merge <n> --repo pruuk/DOGMud --merge --delete-branch
```

If GitHub refuses because required checks did not pass, rerun with `--admin` (the owner authorised merging past red CI on local gate evidence for this PR). Then `git -C C:/tmp/dogmud-bauble-sweep ls-remote --tags origin master` and, if a stray `refs/tags/master` appears, delete it with `git -C C:/tmp/dogmud-bauble-sweep push origin :refs/tags/master`.

- [ ] **Step 4: Hand-off notes (no deploy)**

Report: the PR number and merge SHA; that nothing was deployed; that the owner's main checkout `_datafiles/config.yaml` (skip-worktree `S`) lacks `BaubleSweepHours` until the EOD re-sync from the HEAD blob (the Go default 6 equals the shipped value, so nothing behaves differently meanwhile); and that the first sweeps after a deploy need two runs (about 6 hours) before anything can be pruned.

---

## Self-review

- **Spec coverage.** Research (every store, how to read it off the live world, alts, bank, inbox, rooms and `rooms.instances`, containers, corpses, mobs and `mobs.instances`, shops with `AffixedStock`, auctions, guild vaults, crates; warehouses, caravans, ferries and housing checked and holding no items; nesting): facts table and store list. When and where: Decision 1, Tasks 8 and 10. Locks and game-loop cost: Decision 2, cost estimate, Task 13 Step 7 threshold. Two-phase safety and `last seen`: Decision 3, Task 6. `ReturnCreditAt`: Task 6 test. Rolled-back sale: Decision 5, `TestRunSweepKeepsARolledBackSale`, `TestSell_Bauble_ASoldRecordSellsAgain`. Fail closed: Decision 6, `TestRunSweepFailsClosed`, `TestDiskRefsFailsClosed`. Persistence (`util.Save`, living state, persist before publish): Decision 9, `persistShardPruning`, `TestApplySweepWriteFailurePrunesNothing`. Admin line and log line: Tasks 8 and 11. Config knob per dogmud-balance-config with Go default equal to the shipped value and a test: Task 5. Tests: guard for a missed store (Tasks 2 and 4), per-store (Task 2 subtests per root, Task 10 per store file), fail closed, rollback, keep-window boundary (`TestApplySweepKeepWindowBoundary`). Gate and delivery: Tasks 13 and 14.
- **Placeholder scan.** Every code step carries its code. The only values filled at execution are the PR number and the pasted gate output, which are run results, not design.
- **Type consistency.** `LiveWalk func(visit func(*items.Item))` is used by `RegisterLiveSource`, `withLiveSources`, `holding`, `bauble_sweep.go` and the auctions `init`. `applySweep(now, refs, keep) (referenced, pruned int)` matches its tests and `runSweep`. `scanDisk(root, now, add) (files, parsed, err)` matches `DiskRefs` and `runSweep`. `SweepStatus` fields (`At, OK, Err, Skipped, Records, Referenced, Pruned, Files, Parsed, Live, Disk`) match `baubleSweepLine` and the tests. `persistShardPruning(shard, prune) (int, error)` matches `persistShard` and `applySweep`. `minUnseenSweeps`, `prunableAt`, `lastEvidence` are defined in Task 6 before Task 8 uses them; `writeDataFiles` (Task 7) and `sweepRecord` (Task 6) are defined before Task 8 reuses them.
