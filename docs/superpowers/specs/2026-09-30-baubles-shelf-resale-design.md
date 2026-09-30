# Baubles slice D: shelf resale (design)

Date: 2026-09-30. Owner-approved design, specced against `origin/master`
6b6ff7ddf. Parent spec:
`docs/superpowers/specs/2026-09-28-baubles-hardening-and-corpus-design.md`
(slice D row). The owner ruled on four points the code raised
(2026-09-30); they are folded into the sections below and recorded under
"Rulings" at the end.

## Facts verified against source

Every row was read from the tree at 6b6ff7ddf on 2026-09-30. Balance values
are from `_datafiles/config.yaml` (`git show HEAD:`).

| # | Fact | Where |
|---|------|-------|
| F1 | `sellBaubleToMerchant(seller, item, room, mob, shopInv, awardProgression)` prices via `baubleOfferFor`, removes the item, saves a living shop, then calls `baubles.MarkSold(item.Bauble, sellValue, seller.GetUserId())`. Nothing is stocked; the comment says "it leaves the world" | `internal/actions/sell_bauble.go:198-257`, comment `:240-241`, `MarkSold` `:248` |
| F2 | `baubleOfferFor(item, shopInv, fence, zone)`: fence and `StolenGoods()` pays `FencePrice`; fence otherwise pays `BaublePrice`; a living shop not in `BaubleBuyerCraftSupports` refuses; `HotIn(zone, now)` refuses; else `BaublePrice`. Reserve check for living shops | `sell_bauble.go:108-138` |
| F3 | Test clock for heat at a sale: `var baubleNowForSale = time.Now` | `sell_bauble.go:61` |
| F4 | Legacy merchant (nil `shopInv`) pays from `mob.Character.Gold` | `sell_bauble.go:143-148`, `:223-227` |
| F5 | `(*Mob).GetSellPrice` returns 0 for a bauble | `internal/mobs/mobs.go:1093-1095` |
| F6 | `(*Mob).IsFence` matches `m.Groups` against `BaubleFenceGroups` | `mobs.go:1000-1013` |
| F7 | `BaubleFenceGroups: [fence]`, `BaubleFenceBuyPct: 60`, `BaubleStolenHeatHours: 72`, `BaubleBuyerCraftSupports: [general, jewelcrafting]`, `ShopBuyRatio: 0.50`, `BarterMaxDiscount: 0.15`, `BaubleCheapMinValue: 1` | `config.yaml:1531, 1530, 1519, 1554-1556, 1418, 1434, 1546` |
| F8 | Nine fence mobs: 104 (general), 250 (none), 9172 (none), 9185 (general), 9209 (general), 9213 (general), 9215 (none), 9323 (none), 9428 (cooking). Each has shop entries | `_datafiles/world/dogmud/mobs/**` `groups:` and `craft_support:` |
| F9 | Every mob with any `Character.Shop` entry, or a crafter with materials or recipes, gets a `ShopInventory` via `RegisterMobShop` | `internal/mobs/crafter.go:44-107` |
| F10 | `ValidateShopMobTags` lets only a fence omit `craft_support` | `internal/shops/validation.go:53-56` |
| F11 | `AffixedStockEntry{Item, Price, AddedRound uint64 "added_round,omitempty"}` | `internal/shops/shopinventory.go:72-76` |
| F12 | `AddAffixedStock(item, price, cap int)` appends with `AddedRound: util.GetRoundCount()` and drops index 0 while `len > cap` (cap <= 0 means none) | `shopinventory.go:141-152` |
| F13 | `RemoveAffixedStock(idx) (items.Item, bool)` | `shopinventory.go:155-162` |
| F14 | `AddedRound` is written at `:145` and read nowhere | repo-wide grep for `AddedRound` |
| F15 | `ShopAffixedStockCap` is NOT in `config.yaml`; the live value is the Go default 8 | `internal/configs/config.balance.go:950`, `config.balance.misc.go:312-313`; grep of `config.yaml` finds nothing |
| F16 | `AddAffixedStock` callers: sell of affixed loot `sell.go:392-393` (price `item.GetSpec().Value`), auction win `modules/auctions/npc_buyers.go:296-297`, buy rollback `buy.go:639` (cap 0, barter-discounted `matched.price`) | grep |
| F17 | The auction shopkeeper bids only where `EvaluateBuyRules` offers; that refuses an item whose spec has no `VendorCategories`, and carrier item 900 has none, so auctions never shelve a bauble today | `npc_buyers.go:216-237`, `internal/shops/buyrules.go:50-52`, `_datafiles/world/*/items/*/900-curious_trinket.yaml` |
| F18 | `list`: `buildShopStockFromInventory` reads only `shopInv.Stock`; `AffixedStock` is never shown. An empty listing makes the mob `say I have nothing to sell` | `internal/usercommands/list.go:125-149`, `:68-70` |
| F19 | `renderShopTable` sends one table to the lister alone with `user.SendText` | `list.go:433-438` |
| F20 | `tryPurchaseFromInventory` adds every `AffixedStock` entry to the match list by `e.Item.GetSpec().Name`, fancy names by `e.Item.DisplayName()`, price `e.Price` less barter | `internal/actions/buy.go:566-582` |
| F21 | No match: the mob says "Any interest in this X?" naming a random fancy name to the room (`shopMob.Command`) | `buy.go:588-596` |
| F22 | Affixed purchase: gold check, `RemoveAffixedStock`, fresh UUID, `StoreItem`, rollback on failure, `SaveShop`, "You buy the X" to buyer, room line with `DisplayName()` | `buy.go:628-664` |
| F23 | For a bauble, `GetSpec()` resolves through the catalog: `Value` is the record's `Value`; name and text are the generic view (viewer 0) | `internal/items/items.go:329-341`, `internal/items/bauble.go:76-105` |
| F24 | Viewer-aware accessors: `GetSpecFor`, `DisplayNameFor`, `NameFor`, `LongDescriptionFor` (items), `MaterialFor` (baubles). A finder-only record shows `Trinket` to everyone else | `internal/items/bauble_viewer.go:13-46`, `internal/baubles/record.go:155-181`, `fallback.go:16` |
| F25 | Root guard lists every finder-view call site as `"path\|Func"` with a call count; a reference in an unlisted function fails; sends beyond one reader are refused | `bauble_finder_view_guard_test.go:71-82`, naming `lookup_viewer_guard_test.go:132` |
| F26 | Heat: `HeatDuration()`, `Record.Hot(now)` (stolen goods and `now < StolenAt + HeatDuration()`), `StolenGoods()`, `HotIn(zone, now)`, `HeatArea`, and the item helper `ItemIsHotIn(itm, zone, now)` | `internal/baubles/theft.go:88-151` |
| F27 | Statuses `ready`, `fallback`, `sold`, `retired` | `record.go:14-19` |
| F28 | `MarkSold` sets `Status=sold`, `SoldAt=now UTC`, `SoldValue` | `internal/baubles/sales.go:20-30` |
| F29 | `SalesSince(t)` counts `Status == sold && !SoldAt.Before(t)` | `sales.go:35-45` (check at `:39`) |
| F30 | "Every record is sellable, a sold one included" rule and its crash rationale | `sales.go:13-16`; test `TestSell_Bauble_ASoldRecordSellsAgain` `sell_bauble_test.go:233` |
| F31 | Status readers: `CatalogStats` unsold count `admin.go:70`; `Restore` `admin.go:113-121` (sold when `SoldValue > 0`); `ApplyRegenerated` `admin.go:262`; retired checks `record.go:163`, `corpus.go:380`, `corpus_admin.go:139`; admin command `admin.bauble.go:248, 292, 396`; `SalesSince` | grep for `Status` in `internal/baubles`, `internal/usercommands` |
| F32 | `View()` shows the real text unless `Status == retired` | `record.go:163-168` |
| F33 | Sweep: live source `shops` walks `shops.AllShops()` with `WalkItems`, which visits every `AffixedStock` item; the disk scan reads `shops/` files too | `bauble_sweep.go:49-52`, `internal/shops/walk_items.go:7-13`, `bauble_sweep_test.go:92` |
| F34 | `prunableAt` needs 2 unseen sweeps and `BaubleCatalogKeepDays` since last evidence; status plays no part | `internal/baubles/catalog.go:374-397` |
| F35 | Commands run in `processInput` (`world.go:962`, `TryCommand` `:1040`), reached from the `Input` listener inside `EventLoop`, which runs under `util.LockMud()` (`world.go:864-866`). The auction `NewRound` listener (`auctions.go:72`) runs there too | `world.go` |
| F36 | `ShopInventory` has no lock of its own; `shopCacheMu` guards only the cache map. `SaveShop` marshals the live struct | `internal/shops/persistence.go:19-22, 151-183` |
| F37 | Catalog: `cat.mu` taken briefly, never held across a call out; disk writes outside it. Sweep takes the mud lock for the live walk, then `cat.mu` in `applySweep` | `catalog.go:20-27`, `sweep.go:212, 228-229, 59-63` |
| F38 | `ShopSnapshot` has `CraftSupport` (yaml and json `craft_support`); `captureShops` covers every cached shop; `lookupShopMobName` falls back to `mobs.GetMobSpec` | `internal/economy/health/snapshot.go:60-88`, `capture.go:42-111, 484-499` |
| F39 | `CraftSupport` consumers: `PerCraftSupportScores` keys on it (`scoring.go:151`), `ShopScoreRow.CraftSupport` (`scoring.go:825`); page groups `s.craft_support \|\| "(uncategorized)"` (`index.html:314`), score lookup `PerCraftSupport[disc]` (`:324`), per-shop cell `row.CraftSupport` with a dash fallback (`:351`) | files named |
| F40 | Test to invert: `assert.Len(t, si.AffixedStock, 0, "baubles are not resold like affixed loot")` in `TestSell_Bauble_LivingShopByCraftSupport` | `sell_bauble_test.go:160-192`, line `:191` |
| F41 | `saleProgression` awards Bartering once per `sell` command (`awardProgression` is true for the first sale only); `postSuccessBookkeeping` awards Bartering once per `buy` command, living-shop path included | `sell.go:292-295, 474-477`, `buy.go:431-435, 809-822` |
| F42 | `Restore` picks the unsold status inline: `fallback`, or `ready` when `r.Generator.Named()`; then `sold` when `SoldValue > 0` | `internal/baubles/admin.go:111-128` (rule `:116-119`) |
| F43 | The award path `AwardResolved` then `ApplyProgression` then `OnSkillUseScaled` then `CheckSkillProgression` has no per-trade or time-based limit on ordinary events: the chance is `ProgressionChanceForSkill`, keyed on skill level (plus mob gates). The only per-round claims are `claimBonusProgression` (bonus events) and `DriftFromCombat` | `internal/characters/progression_award_resolved.go:32-82`, `progression.go:115-140, 178-216, 384, 824, 884-935`; grep of `internal/progression`, `internal/characters` for cooldown, diminish, throttle, repeat |

Brief corrections: `ApplyRegenerated` (`admin.go:262`) also reads status;
F17 confirms auctions shelve no bauble today; F15 shows the cap knob has no
`config.yaml` key to raise.

## Summary

Baubles sold to a living shop go on its shelf (`AffixedStock`) at catalog
value instead of leaving the world. `list` finally shows the shelf, per
viewer. A bauble still hot when shelved is held off the shelf until its heat
ends. A retired bauble is never shelved. Held entries sit outside the cap;
the cap (listed entries) rises from 8 to 12. Buying one back returns its
record to the status it had unsold. The dashboard types a fence's shop as
`fence`.

## 1. Shelf entry and helpers (`internal/shops`)

`AffixedStockEntry` gains two fields; `AddedRound` stays as is.

```go
AddedAt   time.Time `yaml:"added_at,omitempty"`   // wall clock when shelved
HoldUntil time.Time `yaml:"hold_until,omitempty"` // zero: listed at once
```

Why a wall-clock `AddedAt`: heat is real time (`StolenAt`, F26) and
`AddedRound` is game rounds (F12), which do not advance while the server is
down, so the two cannot be ordered against each other. Both ends of the
comparison become `time.Time`.

- `(e AffixedStockEntry) Held(now time.Time) bool` is `now.Before(e.HoldUntil)`.
- `(e AffixedStockEntry) ListedAt() time.Time` is `HoldUntil` when non-zero,
  else `AddedAt`. Entries saved before this change have both zero and so
  sort earliest.
- `AddAffixedStock(item items.Item, price, cap int, holdUntil, now time.Time)`
  appends `{Item, Price, AddedRound: util.GetRoundCount(), AddedAt: now,
  HoldUntil: holdUntil}` and then calls `EnforceAffixedCap(cap, now)`. The
  signature change is deliberate: the compiler lists every caller (F16).
- `EnforceAffixedCap(cap int, now time.Time) int`: while the count of
  entries with `!Held(now)` exceeds `cap`, remove the non-held entry with the
  earliest `ListedAt()`, ties to the lowest index. Returns how many it
  removed. `cap <= 0` removes nothing. A removed item is gone; a bauble's
  record keeps the sale that shelved it.
- `RestoreAffixedStock(idx int, e AffixedStockEntry)` reinserts an entry at
  `idx` (clamped to the length), for `buy`'s rollback.

`baubles.ShelfHoldUntil(itm items.Item, now time.Time) time.Time`, beside
`ItemIsHotIn` (F26): for a bauble whose record is `Hot(now)` (global, not
`HotIn`), `rec.StolenAt.Add(HeatDuration())`; otherwise zero. Every shelving
caller uses it, so non-baubles always get zero.

## 2. Shelving at the sale (`internal/actions/sell_bauble.go`)

Inside the existing `if shopInv != nil` block (F1), before `SaveShop`:

```go
if rec, ok := baubles.Get(item.Bauble); ok && rec.Status != baubles.StatusRetired {
    now := baubleNowForSale()
    shopInv.AddAffixedStock(item, item.GetSpec().Value,
        int(configs.GetBalanceConfig().ShopAffixedStockCap),
        baubles.ShelfHoldUntil(item, now), now)
}
```

Price is the catalog value (F23), the same `GetSpec().Value` rule as affixed
loot (F16). Every living shop that buys a bauble shelves it: fences, and the
`general` and `jewelcrafting` buyers (F2, F7, F9). Two cases still destroy
it as today: a legacy merchant (nil `shopInv`), and a retired record (ruling
1), whose withdrawn text would otherwise be listed under its real name once
`MarkSold` sets it sold (F28, F32). `MarkSold` is unchanged and still runs
after the save for every sale. The header comment (`:18-25`) and the `:240`
comment are rewritten to say which baubles are shelved.

## 3. `list` (`internal/usercommands/list.go`)

For every mob merchant with a `ShopInventory`:

1. `now := time.Now()`; `EnforceAffixedCap(cap, now)`; if it removed
   anything, `shops.SaveShop`.
2. Render the regular stock as today.
3. Render the non-held shelf entries as a second table titled
   `On the shelf`, columns Name, Type, Price, help line
   `To buy something, type: buy [name]`, through `renderShopTable` (F19).
   Name is `e.Item.DisplayNameFor(user.UserId)` (F24), so a finder-only
   bauble reads `Trinket` to everyone but its finder. Price is `e.Price`
   (before barter, like the stock table).
4. The "nothing to sell" say (F18) fires only when both tables are empty.

The new rows are built in one function, `buildShelfRows(shelf
[]shops.AffixedStockEntry, viewerUserId int, now time.Time) [][]string`,
registered in `finderViewSites` (F25):
`"internal/usercommands/list.go|buildShelfRows": {1, "the lister's own shop
listing; renderShopTable sends it to that user alone"}`.

This also fixes the existing gap where sold-on affixed loot could be bought
only by guessing its name (F18, F20).

## 4. `buy` (`internal/actions/buy.go`)

In `tryPurchaseFromInventory`:

- Before building the lists: `now := baubleNowForSale()`;
  `EnforceAffixedCap(cap, now)`; remember whether it removed anything.
- Match a shelf bauble by the buyer's own view (ruling 4): its entry's
  `plainName` and its `itemNames` element are
  `e.Item.NameFor(buyer.GetUserId())` (F24), the same view `list` showed
  that buyer, so a finder buys by their own name and everyone else by
  `Trinket`. A mob buyer has user id 0 and so gets the generic view.
  `itemNamesFancy` (sent to the room by the mob's say, F21), the room line
  and the "You buy the X" line keep `DisplayName()`. For every other entry
  `plainName` stays `GetSpec().Name`. The one new reference is registered
  in `finderViewSites` (F25):
  `"internal/actions/buy.go|tryPurchaseFromInventory": {1, "a match key
  only: the buyer's own view of a shelf bauble's name, compared with what
  the buyer typed and never sent"}`.
- Skip held entries: they join neither `itemNames` nor `itemNamesFancy`
  (F20). A request that matches only a held bauble therefore takes the
  existing no-match path (F21) or the existing close-match rule, and the
  mob's "Any interest in this X?" can never name a held item.
- Affixed purchase (F22): take a copy of the entry before
  `RemoveAffixedStock`; on `StoreItem` failure, `RestoreAffixedStock(idx,
  copy)`. This replaces the `AddAffixedStock(bought, matched.price, 0)`
  rollback, which stamped a new listing time and stored the barter-discounted
  price as the relist price.
- After a successful purchase of a bauble: `baubles.MarkBought(bought.Bauble,
  buyer.GetUserId())` (section 5).
- If `EnforceAffixedCap` removed anything and no purchase saved the shop,
  save it before returning.

## 5. Record (`internal/baubles/sales.go`)

- New `(r Record) unsoldStatus() Status` in `admin.go`: `StatusReady` when
  `r.Generator.Named()`, else `StatusFallback`. `Restore` (F42) is rewritten
  to call it in place of its inline lines `:116-119`, so the rule lives once
  (ruling 2).
- New `MarkBought(id string, buyerUserId int) bool`: `Update` that sets
  `Status = r.unsoldStatus()` when the status is `sold`, and changes nothing
  otherwise, so a record an admin retired while it sat on the shelf stays
  retired. Logs `action bought` like `MarkSold` (F28). `SoldAt`,
  `SoldValue` and every theft field are kept. A stolen bauble bought back
  is therefore cold (its hold outlasted `Hot`) but still `StolenGoods()`:
  honest shops buy it anywhere and a fence pays `FencePrice`, 60% (F2, F7).
- `SalesSince` counts `!r.SoldAt.IsZero() && !r.SoldAt.Before(t)` and drops
  the status test (F29), so a buyback does not erase a past sale. A record
  sold twice holds only its latest `SoldAt` and `SoldValue`, so it counts
  once, at its latest sale.
- No new status (owner ruling).
- The "a sold record can sell again" rule (F30) stays: the code path is
  unchanged. Its comment is rewritten: a record now also returns to a pack
  legitimately, through a buyback, and the crash case is the only way one
  still marked sold does.

## 6. Dashboard (`internal/economy/health`, admin page)

`ShopSnapshot` gains `Fence bool` with `yaml:"fence,omitempty"
json:"fence,omitempty"`, set in `captureShops` from the mob template:
`t := mobs.GetMobSpec(mobs.MobId(inv.MobId)); ss.Fence = t != nil &&
t.IsFence()` (the template is always loaded, F38; instance groups can gain
bounty tags). `CraftSupport` keeps the shop's real tag.

A flag rather than a derived type string: saved snapshots decode with
`Fence` false and render exactly as before, and the real `craft_support` of
fences 104 and 9428 (F8) is not lost from history.

`func (s ShopSnapshot) Type() string` returns `"fence"` when `Fence`, else
`CraftSupport`. `PerCraftSupportScores` keys on `s.Type()` (`scoring.go:151`)
and `ShopScoreRow.CraftSupport` takes `s.Type()` (`:825`). The page groups on
`s.fence ? "fence" : (s.craft_support || "(uncategorized)")` (`index.html:314`);
the per-shop cell already reads `row.CraftSupport`. Nothing else on the page
changes.

## 7. Config

- `ShopAffixedStockCap: 12` is added to `config.yaml` under
  `SHOP ECONOMY`, after `BarterMaxBonus` (`:1435`), commented as the cap on
  listed shelf entries, held ones excluded.
- Go default 8 becomes 12 (`config.balance.misc.go:313`) and the field
  comment (`config.balance.go:950`) says the same.
- Commit `config.yaml` from the `git show HEAD:` blob (CLAUDE.md tripwire).

## 8. Concurrency and locks

`list`, `buy`, `sell` and the auction win all mutate a `ShopInventory` from
the event loop under the mud lock (F35); the sweep's live walk takes the
same lock (F37), so a bauble moving shelf to pack is seen in one of the two.
`SaveShop` marshals the struct and so relies on the same lock (F36); the new
saves in `list` and `buy` sit inside it. `MarkBought` and `ShelfHoldUntil`
take `cat.mu` briefly and never while holding a shop or cache lock, the
existing order (mud lock, then `cat.mu`, F37). `list` resolving names through
the catalog is the same read `look` already makes.

## 9. Persistence and migration

Shop files are living state. `AddedAt` and `HoldUntil` are `omitempty`, so
an unheld entry written after this change differs from today only by
`added_at`, and an old file loads with both zero: listed, and earliest in
eviction order. No migration: the zero values are the correct meaning, no
bauble is on any shelf before this ships (F1, F17), and nothing is renamed or
removed. Snapshots: see section 6.

## 10. Other `AddAffixedStock` callers

- `sell.go:393` (affixed loot): passes `baubles.ShelfHoldUntil(item, now)`
  (always zero there) and `now`; behaviour is the new cap rule, 12 listed.
- `npc_buyers.go:297` (auction win): same call shape. Auctions never win a
  bauble today (F17); if a later change lets them, a hot one is held.
- `buy.go:639`: replaced by `RestoreAffixedStock` (section 4).

## 11. Tests to pin

1. A held entry is absent from `list` and cannot be bought by its name.
2. A hold expires and the entry lists and sells.
3. Held entries do not count toward the cap: cap listed entries plus any
   number held.
4. Over the cap, the entry with the earliest `ListedAt()` goes: an entry
   shelved first but held until later outlives an unheld entry shelved
   after it.
5. Enforcement after a hold expires happens lazily on `list`, on `buy`, and
   on an add.
6. A buyback sets a named record ready and a generic one fallback,
   leaves a retired one retired, and `SalesSince` still counts the sale;
   `Restore` still passes its existing tests through `unsoldStatus`.
7. `list` shows a finder-only bauble's own name to its finder and `Trinket`
   to another player; `buy` matches the finder's own name for the finder
   and `Trinket` for anyone else; the guard passes with the two new rows
   and fails without either.
8. The dashboard types a fence's shop `fence` in the discipline rollup and
   the per-shop row; an old snapshot without `fence` renders as before.
9. A shelved bauble survives a catalog sweep (live and disk).
10. Invert F40: a general store's sale leaves one shelf entry at catalog
    value; a hot bauble's entry carries `HoldUntil = StolenAt + 72h`; a
    retired bauble's sale leaves the shelf empty and the record sold.
11. `buy` rollback restores the original entry and price.
12. Existing `TestAffixedStock_CapEvictsOldest` and
    `TestSell_Bauble_ASoldRecordSellsAgain` still pass.

Tests load Go defaults, not `config.yaml` (dogmud-writing-tests), so the
cap tests pass the cap explicitly.

## 12. Risks

- Duplication on crash: the shop is saved at the sale, the seller's save
  later. A crash between leaves the bauble in the restored pack and on the
  shelf, two items on one record. Affixed loot has had the same window since
  it was shelved (F16). Mirror case: a crash after a buy loses the bauble.
- Held entries are uncapped. A fence could hold many hot baubles for up to
  72 hours; `list`, `buy` and the sweep walk them all. Bounded by theft rate.
- `HoldUntil` is fixed at shelving; a later change to
  `BaubleStolenHeatHours` does not move existing holds.
- A value-1 bauble sells for 1 (F2) and buys back for 1 after barter (F7),
  so a sell and buy loop costs nothing and awards Bartering twice per round
  trip, once per command (F41). No limit on repeat awards from one trade
  was found: the award path's only damping is the level-keyed chance, and
  its per-round claims cover bonus events and combat drift only (F43). The
  plan must re-run that grep (`internal/progression`,
  `internal/characters/progression*.go`, `sell.go`, `buy.go` for a
  cooldown, a per-merchant or per-item memory, or diminishing returns),
  cite any limit it finds, and otherwise record the loop as an accepted
  exposure or put a fix to the owner.
- Accepted limitation (ruling 3): retiring and then restoring a bauble that
  was bought back labels it `sold` while a player holds it, because
  `Restore` still reads `SoldValue > 0` (F42). No `BoughtAt` field is added;
  the label feeds admin reporting only.
- The general-store discipline score changes when four fences (104, 9185,
  9209, 9213) leave the `general` group for `fence`.

## Rulings

Owner rulings of 2026-09-30 on the four points the code raised against the
approved design.

1. **Retired baubles are not shelved.** `MarkSold` overwrites `retired`
   with `sold` (F28) and `View()` hides text only for `retired` (F32), so a
   shelved retired bauble would be listed under its withdrawn text. Ruling:
   selling a retired bauble destroys it as today, `MarkSold` is unchanged,
   and only records that are not retired go on the shelf (section 2).
2. **Buyback status uses `Restore`'s rule.** A literal "ready" would
   mislabel a generic trinket. Ruling: `ready` when `Generator.Named()`,
   otherwise `fallback`, from one shared helper that `Restore` also calls
   (section 5).
3. **`Restore` after a buyback shows `sold`: accepted.** No `BoughtAt`
   field. Recorded under Risks as an accepted limitation.
4. **`buy` matches a shelf bauble by the buyer's own view,** the view
   `list` used, and the new site is registered with the finder-view guard
   (section 4).

## Out of scope

Salvaging baubles into materials, heat stepping by zone, and any other
dashboard change (no panel, no shelf counts).

## Docs to update in the implementation

`internal/shops/context.md` (entry fields, new methods, cap rule),
`internal/baubles/context.md` (`MarkBought`, `ShelfHoldUntil`, `SalesSince`
rule), `internal/economy/health/context.md` (`Fence`, `Type()`),
`internal/actions/context.md` (its Baubles section says "Never stocked,
never resold" at `:1369`; replace with the shelf and the buyback), and
`internal/usercommands/context.md` (`list` shows the shelf).
