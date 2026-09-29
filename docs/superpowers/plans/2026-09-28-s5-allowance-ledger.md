# S5: per-user allowances in apiframework, companion migrated. Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move every per-user daily token allowance (the AI companion's owner, passer-by and per-owner passer-by caps, and a new per-finder bauble cap) onto the one `apiframework` ledger, with per-feature shares of the server budget, and migrate the companion onto it without losing a single pinned rule.

**Architecture:** The ledger (`internal/apiframework/budget.go`) grows a `Charge` (one per-user allowance, carrying its limit), a `Reserve(consumer, tokens, spendServer, charges...)` that checks the global budget, the consumer's share and every charge under its one lock, all or nothing, a `Settle` that adjusts every counter the hold touched, and `by_user` persistence. The companion keeps no allowance state of its own: `reserveRoute` and `settleRoute` become thin calls onto the ledger, the read-only checks call `Allowance`, and the ledger's clock is the only day. Baubles charge `baubles.finder` on both routes.

**Tech Stack:** Go 1.25, `gopkg.in/yaml.v3` (ledger), `gopkg.in/yaml.v2` (companion tests), the repo's own `util.Save` / `util.ReadLivingState` living-state helpers.

**Spec:** `docs/superpowers/specs/2026-09-28-baubles-hardening-and-corpus-design.md`, section "S5. Allowances in `apiframework`, with the companion migrated", and owner rulings 11 and 12. Owner ruling 4 (full migration) and ruling 12 (the S5 defaults) are settled; this plan does not re-argue them.

**Where and when (ruling 11):** PR #175 merges first; then slices M, H, S5 and C each reach master as their own PR. S5 branches FRESH from master after slice H's PR has merged: worktree `C:/tmp/dogmud-baubles-s5`, branch `fix/baubles-allowance-ledger`, created from the main checkout in Task 1. S5 never rebases or pushes onto FinalTwist's branch. Task 1 records the base commit as `BASE`; every diff, format check and size check below runs from `$BASE`. Every file:line below was read at `5b1b0221a` in `C:/tmp/pr175-ours` (its Go files under `internal/apiframework`, `modules/aicompanion`, `modules/baubles` and `internal/configs/config.apiframework.go` are byte-identical to `cfb531437`, where the tables were first built). Slice H edits `internal/apiframework/settings.go`, `internal/configs/config.apiframework.go`, `modules/baubles/generate.go` (FinalTwist's moderation reorder also lands there, in `viaServer`, `name` and `viaPlayer`) and the docs, so Task 1 re-verifies every row and every quoted anchor against merged master before any code moves. Task 1 ran at `09964d50f` (origin/master after slice H: H1 #187, the sweep #188, H2 #190, H3 #191): every file:line in the facts table, the rule table and tables T1 and T2, and every task's line references and quoted anchors, now read at `09964d50f`. The local `master` in the main checkout is stale; wherever this plan says `master` for a base or a merge-base, it means `origin/master`.

---

## Facts verified against source (read at `5b1b0221a`; re-verified at `09964d50f`)

Rows whose file slice H touched carry their `09964d50f` line numbers. Files slice H did not touch (`budget.go`, `books.go`, `relay.go`, `aicompanion.go`, `models.go`, `tiers.go`, `runtime.go`, `commands.go`, `autonomy.go`, `conversation.go`, `corememory.go`, `reflect.go`, `money_test.go`, `tiers_test.go`, `relayfor_test.go`, `frameworktest_helpers_test.go`, `modules/baubles/config.go`) are byte-identical to `5b1b0221a`, so their rows stand as read.

| Fact | Where |
|---|---|
| `Hold` is `{Consumer, Tokens, Day}`; no user, no spendServer, no charges | `internal/apiframework/budget.go:47-51` |
| `ledgerState` holds `Day, Tokens, Calls, Failures, ByConsumer, CallsBy` | `budget.go:53-60` |
| `rollLocked` starts a new day at `Tokens: outstanding` and re-makes the two maps | `budget.go:77-88` |
| `reserve(consumer, tokens, limit)` checks only `Tokens+tokens > limit` when `limit > 0` | `budget.go:142-160` |
| `settle` does not clamp `used`; overage is charged; earlier-day hold gives the consumer share no refund; floors total and share at 0 | `budget.go:175-202` |
| `SeedTokens` applies once, today only, fresh day only | `budget.go:267-283` |
| `SaveBudget` copies `ByConsumer` and `CallsBy` under the lock, saves outside it, re-dirties on failure; it saves only the shared ledger `budget` | `budget.go:286-319` |
| `Books{l *ledger; b *breaker; ...}`, `Shared()`, `NewBooksForTest()` (loaded, no disk) | `internal/apiframework/books.go:15-41` |
| `internal/apiframework/relay.go` is 57 lines and holds only the `Relay` interface | `relay.go:1-57` |
| `ServerSettings{Endpoint, RejectedBaseURL, DailyTokenBudget, BreakerErrors, BreakerSeconds, Legacy}` | `internal/apiframework/settings.go:68-80` (re-verified at `09964d50f`) |
| `resolveServer` maps `DailyTokenBudget` > 0 as cap, < 0 as no cap, 0 as legacy or default | `settings.go:257-269` (re-verified at `09964d50f`) |
| `configs.APIFramework` has no share fields; `Validate` sets no numeric defaults (slice H made `APIKey` a `ConfigSecret`) | `internal/configs/config.apiframework.go:17-64` (re-verified at `09964d50f`) |
| Non-test `Reserve` callers: the companion (`models.go:567`) and baubles (`generate.go:258`); test callers `apiframework_test.go:182,186,189,206,219,237,654,665`, `baubles_test.go:259` (re-verified at `09964d50f`) | grep `\.Reserve(\|budget\.reserve` |
| Companion per-user state: `budgetDay` (`aicompanion.go:161`), `ownerTokens`, `strangerTokens`, `strangersFor`, `noticesToday` (`aicompanion.go:172-175`); `callsToday`, `errorsToday` (`:162-163`) | `modules/aicompanion/aicompanion.go` |
| `rollDay` resets calls, errors, the three allowance maps and notices on `time.Now()` UTC | `aicompanion.go:403-414` |
| 15 production `rollDay()` call sites | `aicompanion.go:445`, `autonomy.go:181`, `commands.go:93`, `conversation.go:348`, `corememory.go:206`, `listeners.go:569,577` (re-verified at `09964d50f`), `models.go:559,598,614`, `reflect.go:276`, `runtime.go:725`, `tiers.go:252,298` |
| `tryReserveTokens`, `tryReserveFor`, `settleTokens`, `settleFor`, `settleForDay` have NO production caller; only tests call them | grep, `models.go:537-604` |
| Production reservations go only through `reserveRoute` (4 sites) and settle only through `settleRoute` (8 sites) | `conversation.go:302,328,347`, `corememory.go:162,187,205`, `reflect.go:233,257,275`, `runtime.go:620,674,709` |
| Tests build the module with no plugin (`m.plug` nil); `loadBudget` would nil-deref | `aicompanion_test.go:2051-2067` (re-verified at `09964d50f`), `models.go:658` |
| `m.fw()` gives each test module its own isolated `Books` (`isolateBooks`) | `aicompanion.go:367-381`, `frameworktest_helpers_test.go:20-31` |
| `listeners.go` does not import `apiframework` today | `listeners.go` imports |
| Map and `budgetDay` test references: **64 lines** (25 in `aicompanion_test.go`, 24 in `tiers_test.go`, 15 in `money_test.go`), not 61 | grep `ownerTokens\|strangerTokens\|strangersFor\|budgetDay` |
| Calls to the helpers this plan deletes or renames: 49 test lines | grep, table T2 below |
| `viaServer(ctx, cfg, chat)` has no finder id; `viaPlayer(ctx, r, userId, model, chat)` reserves nothing | `modules/baubles/generate.go:200, 239, 258` (re-verified at `09964d50f`) |
| `schemaOverhead = 300` | `generate.go:17` |
| Slice H as merged: `viaPlayer` passes the relayed count through `Charged(..., relayed=true)` (`generate.go:226-227`); `name` takes the finder's own slot (`takeFinderSlot`, `:161`) around `viaPlayer` and a server slot (`takeServerSlot`, `:183-187`) around `viaServer`, and none free is `errSlotsBusy`, which `generate` returns before `m.count` (`:48-52`). There is NO `playerRouteOpen`: the relay route opens whenever `UsePlayerKeys`, a finder, the relay's `Model(finder, PurposeFinds)` and a free finder slot allow it (`:156-161`); moderation is decided after the naming (`moderate`, `moderationPossible`, `:118-120, 323-363`), and player-key text it cannot moderate is `FinderOnly`, not refused. A usable relay answer that fails the plain-words allowlist (`refusedByAllowlist`, `:135-144`) tells the relay breaker nothing and goes on to the server's key (`:165-168`) | `modules/baubles/generate.go` at `09964d50f` |
| Admin regen builds its request with no `FinderUserId` | `internal/actions/bauble_admin.go:45-66` (re-verified at `09964d50f`) |
| Baubles `Config` has no per-user knob; `buildConfig` bounds via `clampInt` | `modules/baubles/config.go:14-35, 113-149` |
| `APIFramework:` block and `Modules.baubles:` block in the shipped config | `_datafiles/config.yaml:2149-2171`, `:2688-2708` (HEAD blob at `09964d50f`, re-verified; the main checkout carries `S`, the S5 worktree `H` and equal to HEAD) |
| `strangerFits` checks `StrangerTokensPerOwner` even when `ownerId` is 0 (`strangersFor[0]` is never charged, so the check is `tokens <= StrangerTokensPerOwner`) | `models.go:245`, `:256-258` |
| The ledger's settle floors the server total at 0 | `budget.go:186-188` |
| A corrupt `budget.yaml` is quarantined and the day starts from nothing (`loadLocked`, `quarantine`) | `budget.go:102-129` |
| Today a `budget.yaml` quarantine loses no companion allowance: they live in the companion's own `budget-state` file | `models.go:638-708` |
| `generate` counts every error from `name` but `errSlotsBusy` as a failure in the display stats (`m.count(playerKey, err != nil)`, line 53, after the `errSlotsBusy` early return at 48-52); the breakers are fed only through `report`, which is the no-op `none` when `viaServer`'s `Reserve` refuses (the ticket is released) | `modules/baubles/generate.go:47-61, 254-262`, `baubles.go:138-154` (re-verified at `09964d50f`) |
| `baubles.Generate` logs every error from the module's generator at Warn, with the error text, before it falls back to a generic trinket | `internal/baubles/generate.go:138-141` (re-verified at `09964d50f`) |
| `logBudgetRefusal(ownerId, askerId, wanted)` logs one payer's spend and cap, not which counter refused; its one caller is the dispatch refusal | `runtime.go:1549-1565`, `:630` |
| Shipped-config tests read `_datafiles/config.yaml` by repo path and decode with `gopkg.in/yaml.v2` (the loader's library) | `internal/configs/config.balance.baubles_test.go:175-183` (re-verified at `09964d50f`), `internal/configs/configs.go:13` |
| `m.books` is an `atomic.Pointer[apiframework.Books]`; `m.fw()` reads it, so a test can replace it | `aicompanion.go:188, 367-381` |
| This machine: `CGO_ENABLED=0`, no `gcc`; `-race` cannot run locally | `go env CGO_ENABLED`, `which gcc` |
| CI runs `go test -timeout 900s -race ./...`; the Docker test image sets `CGO_ENABLED=1` | `.github/actions/codegen-and-test/action.yml:69`, `provisioning/Dockerfile:17-27`, `compose.test.yml` |
| Settings doc guard: every companion `buildConfig` key must appear in `docs/aicompanion/settings.md` | `tiers_test.go:871-890` |

## Spec statements corrected against source

1. **"`internal/apiframework/relay.go:411-412`"** does not exist: the file is 57 lines. The claim it supports (a passer-by call on the owner's key never touches the server ledger) is true, at `modules/aicompanion/tiers.go:259-268`.
2. **"61 companion test references"** is 64 lines at `cfb531437` (25, 24, 15). Table T1 lists every one.
3. **"`used` is clamped to 0..`h.Tokens` (the relay clamp, now for every caller)"** would delete a live rule: on the server's key, usage past the reservation is charged (`budget.go:184-188`; `models.go:594-596` "What it used past its reservation is still charged"). A provider-reported count is trusted; only a browser-relayed one is not. This plan floors `used` at 0 for every hold and clamps the top only when `SpendServer` is false (rule R15 keeps a new test). Owner ruling 12 decided it: clamp relayed counts only; server-key overage is still charged.
4. **"`Allowance(dim, userId) (spent, limit int)`"**: the ledger has no source for a module's limit (limits are `Modules.aicompanion.*` and `Modules.baubles.*`, which `internal/` cannot read, and tests change `m.cfg` directly). The limit therefore rides on each `Charge` (`Charge{Dim, UserId, Limit}`), read from live config at the call, and `Allowance(dim, userId) int` returns the spend; callers already hold their limit.
5. **"The companion's `rollDay` and `budgetDay` go"**: true for allowances only. `callsToday`, `errorsToday` and `noticesToday` (the `NoticeCallsPerDay` cap, `autonomy.go:181-192`) share that day. They stay in the module, rolled by `rollCounters` on the ledger's clock (`Books.Day()`), in a field renamed `countersDay` so the compiler finds every old use.
6. **Owner id 0** (not in the spec): a server-key owner call with `ownerId` 0 was charged to key 0 and never settled back (`models.go:630` `case ownerId > 0`). The ledger settles every charge it made, so key 0 is now refunded like any other. No test pinned the old leak. Owner ruling 12: refund owner-less key-0 holds.
7. **`viaPlayer` charges `baubles.finder`** (as the spec says): a finder's own key is then bounded by a server-set allowance. Owner ruling 12: charged on the finder's own key too.
8. **Share percent 0**: the spec gives `BaublesSharePercent` 25 and no companion value. Owner ruling 12: `0` (and absent) means the default (companion 100, baubles 25); `-1` or `100` (and above) means no share cap; no `DailyTokenBudget` means no share cap.
9. **"Corruption keeps today's quarantine behaviour; the day's per-user spends restart with it, which is the same loss the file already accepts for totals"** is wrong for the companion. Today the companion's per-user counts live in its own `budget-state` file (`models.go:638-708`), so a quarantined `budget.yaml` loses none of them; moving them into `budget.yaml` alone would make a quarantine hand every owner and passer-by a fresh allowance, a loss that does not exist today. Owner ruling 12 closes it: the companion keeps writing its three maps to its own file as a backup (read from the ledger through a new `Allowances(dim) map[int]int`), and after a quarantine the ledger re-seeds from that backup. Seed marks are one per dimension per day, saved in `budget.yaml`: a normal restart finds the marks and skips the seed; a quarantine loses the marks with the counts, so the next boot re-seeds. `baubles.finder` has no backup; a quarantine restarts it, which loses nothing that exists today (the dimension is new).
10. **`strangerFits` with owner id 0** (not in the spec): today a passer-by's call with no owner is still refused when it alone exceeds `StrangerTokensPerOwner` (`models.go:245` reads `strangersFor[0]`, which is never charged). The ledger drops this on purpose: `allowanceCharges` passes no `companion.strangersfor` charge without an owner, because that cap protects one owner's companion and there is none; the passer-by's own `StrangerDailyTokens` still applies. Rule R42 records it.

---

## File map

| File | Change | Responsibility |
|---|---|---|
| `internal/apiframework/budget.go` | modify | `Charge`, dimensions, `Hold` fields, `RefusalError`/`RefusedBy`, `reserve`/`settle` rules, `Allowance`, `Allowances`, `SeedAllowances`, `Day`, `SetAllowanceForTest`, `by_user`/`seeded` state, `SaveBudget` deep copy |
| `internal/apiframework/settings.go` | modify | `ServerSettings.CompanionSharePercent`, `BaublesSharePercent`, `SharePercent`, `sharePercent`, share defaults |
| `internal/apiframework/allowance_test.go` | create | every new ledger rule, the `-race` `SaveBudget` test, the shipped share knobs |
| `internal/apiframework/apiframework_test.go` | modify | mechanical `reserve`/`Reserve` call updates only |
| `internal/configs/config.apiframework.go` | modify | two `ConfigInt` share knobs |
| `modules/aicompanion/tiers.go` | modify | `allowanceCharges`, `reserveRoute`, `hold`, `settleRoute` on the ledger |
| `modules/aicompanion/models.go` | modify | delete the eleven allowance helpers; `ownerBudgetLeft` reads the ledger; `restoreBudget` (seed from the backup), `budgetStateToSave` (write the backup) |
| `modules/aicompanion/aicompanion.go` | modify | `rollCounters`, `countersDay`, `countCall`; delete `rollDay`, `budgetDay`, the three maps |
| `modules/aicompanion/listeners.go`, `commands.go`, `runtime.go`, `autonomy.go`, `conversation.go`, `corememory.go`, `reflect.go` | modify | call sites; `logBudgetRefusal` names the counter that refused |
| `modules/aicompanion/frameworktest_helpers_test.go` | modify | `ownerSpent`/`setOwnerSpent` and siblings |
| `modules/aicompanion/allowance_test.go` | create | `allowanceCharges` table, counters roll, quarantine re-seed and restart tests |
| `modules/aicompanion/aicompanion_test.go`, `tiers_test.go`, `money_test.go`, `relayfor_test.go` | modify | the 64 + 49 references (tables T1, T2) |
| `modules/baubles/generate.go`, `config.go`, `baubles_test.go` | modify | finder charge on both routes, `DailyTokensPerUser`, a refusal is not a failure in the stats, the shipped `DailyTokensPerUser` |
| `_datafiles/config.yaml` | modify (from the HEAD blob) | three knobs and their comments |
| `internal/apiframework/context.md`, `modules/aicompanion/context.md`, `modules/baubles/context.md`, `docs/aicompanion/settings.md` | modify | docs (this plan's own `docs/README.md` row ships with the docs-only PR, not here) |

---

## Rule table (Task 1 deliverable)

Every charge, check, settle, clamp, floor and day rule of the current allowances. "Test" is what pins it today; "After" is how the ledger expresses it and what test pins it after S5. A rule with "none" in Test had no pin before; the plan adds one where marked NEW.

Re-verified at `09964d50f`: slice H added four import lines to `aicompanion_test.go` (every line below them moved down 4; its new tests sit after `TestGoneMindStillRefundsItsOwner`), eleven lines to `listeners.go` before `strangerMayAsk` (+11), and reshaped `modules/baubles/generate.go` and `baubles_test.go`. The line numbers in these tables and in T1 and T2 are the `09964d50f` ones; `money_test.go`, `tiers_test.go`, `relayfor_test.go`, `apiframework_test.go` below its line 471, and every companion source file but `listeners.go` did not move.

### Checks and charges at reservation

| # | Rule | Source | Test today | After (and its test) |
|---|---|---|---|---|
| R1 | An owner's call on the server key is refused when `ownerTokens[o] + t > DailyTokensPerCompanion` (cap > 0; 0 is no cap) | `models.go:564-566` | `TestTokenReservationSettles` (`aicompanion_test.go:1246`) | `Charge{DimCompanionOwner, o, DailyTokensPerCompanion}`; same test rewritten; `TestReserveChargesEveryAllowanceOrNone` |
| R2 | That call is charged to the owner's counter, key 0 included | `models.go:573-575, 215-220` | `TestTokenReservationSettles` | ledger charges the owner charge; same test |
| R3 | A passer-by's call is refused when `strangerTokens[a] + t > StrangerDailyTokens` (cap > 0), on both routes | `models.go:241-244`, called `:561`, `tiers.go:263` | `TestStrangerCallsAreReservedAgainstTheStranger` (`aicompanion_test.go:2396`), `TestStrangerRelayCallsStopAtTheStrangerCap` (`tiers_test.go:251`) | `Charge{DimCompanionStranger, a, StrangerDailyTokens}`; both tests rewritten |
| R4 | A passer-by's call is refused when `strangersFor[o] + t > StrangerTokensPerOwner` (cap > 0), on both routes | `models.go:245` | `TestStrangerTokensPerOwnerCapsThemTogether` (`money_test.go:491`) | `Charge{DimCompanionStrangersFor, o, StrangerTokensPerOwner}`; test rewritten |
| R5 | `strangersFor` is charged only when `ownerId > 0`; the stranger counter never for `askerId <= 0` | `models.go:256-258, 226-228` | none | `allowanceCharges` omits the charge; `TestAllowanceChargesMapOneToOne` NEW |
| R6 | A server-key passer-by call never charges the owner's allowance, and the owner's spent allowance does not refuse it | `models.go:560-575` | `TestStrangerCallsAreReservedAgainstTheStranger` (2403-2411), `TestStrangerTalkSummaryIsTheStrangersToPayFor` (2576, part 3) | no owner charge on a stranger call; both rewritten; `TestAllowanceChargesMapOneToOne` |
| R7 | The server budget refuses when `Tokens + t > DailyTokenBudget` (limit > 0); the hold counts per consumer and as a call | `budget.go:147-159`, `models.go:567` | `TestBudgetReserveSettleAndShares` (`apiframework_test.go:180`), `TestTheBudgetIsShared` (`baubles_test.go:254`) | unchanged in `reserve`; tests kept |
| R8 | All or nothing: an allowance refusal holds nothing of the server's; a server refusal charges no allowance | `models.go:559-576` | first half: `TestStrangerTalkSummaryIsTheStrangersToPayFor` part 2; second half: none | checked under one lock before any add; `TestReserveChargesEveryAllowanceOrNone` NEW |
| R9 | Check and hold are one step: concurrent reservations never slip past a cap together | `models.go:537-548`, `budget.go:142-160` | `TestStrangerReservationsCannotSlipPastTheCapTogether` (`aicompanion_test.go:2439`), `TestStrangerRelayReservationsCannotSlipPastTheCapTogether` (`tiers_test.go:278`) | ledger lock; both rewritten |
| R10 | Owner's own key, owner's call: nothing checked, nothing held | `tiers.go:260-262` | `TestRelayCallsReserveNothingOfTheServers` (`tiers_test.go:229`), `TestRelayReflectionAndCoreMemorySpendNothingOfTheServers` (`:359`) | `reserveRoute` returns before the ledger; both rewritten |
| R11 | Owner's own key, passer-by: R3 and R4 checked and charged; the server ledger untouched | `tiers.go:259-268` | `TestStrangerRelayCallsStopAtTheStrangerCap`, `TestRelaySummaryChargesOnlyThePasserBy` (`tiers_test.go:305`) | `Reserve(..., spendServer=false, stranger, strangersfor)`; rewritten; `TestARelayReserveLeavesTheServerAlone` NEW |
| R12 | No route: refused, nothing held | `tiers.go:269` | `TestRelayCallsReserveNothingOfTheServers` (246-248) | unchanged |
| R41 | A limit of 0 (or less) is no cap | `budget.go:150` | `TestBudgetReserveSettleAndShares` (206-208) | same for charges; `TestReserveChargesEveryAllowanceOrNone` |
| R42 | A passer-by's call with no owner (`ownerId` 0) is refused when it alone exceeds `StrangerTokensPerOwner` (`strangersFor[0]` is never charged) | `models.go:245`, `:256-258` | none | INTENTIONALLY DROPPED (correction 10): no `companion.strangersfor` charge without an owner, since that cap protects one owner's companion and there is none; the asker's own `StrangerDailyTokens` still applies. `TestAllowanceChargesMapOneToOne` ("a passer-by, no owner") pins the new shape |
| R43 | A refused reservation says which counter refused it: `global` (the day's budget), `share` (the consumer's share), or the per-user dimension (`Charge.Dim`) | NEW (today the companion's refusal log names no counter, `runtime.go:1549-1565`) | none | `RefusalError`, `RefusedBy`; `logBudgetRefusal` logs `refusedBy`; `TestARefusalNamesItsCounter` NEW (Tasks 2, 4), `TestAFinderOverTheirAllowanceGetsNoName` checks the bauble stats (Task 10) |

### Settlement

| # | Rule | Source | Test today | After |
|---|---|---|---|---|
| R13 | Same-day settle: each payer counter moves by `used - reserved` | `models.go:622-635` | `TestTokenReservationSettles` (1258-1265), `TestStrangerCallsAreReservedAgainstTheStranger` (2419-2428), `TestStrangerTokensPerOwnerCapsThemTogether` (508-514), `TestAHoldAcrossMidnightRefundsNothing` control (912-917) | `settle` loops `h.Charges`; all rewritten |
| R14 | No counter goes below 0 | owner `models.go:632-634`, stranger `:253-255`, per-owner `:263-265`, server `budget.go:181-188, 195-197` | `TestTokenReservationSettles` (1266-1269), `TestStrangerCallsAreReservedAgainstTheStranger` (2433-2436) | floors in `settle`; rewritten; `TestSettleClampsAndFloors` NEW |
| R15 | Server key: usage past the reservation is charged to the total, the consumer and the payer | `models.go:594-596, 623`, `budget.go:184-188` | none | kept (see correction 3); `TestSettleClampsAndFloors` NEW |
| R16 | Owner-less server call: charged to key 0 on reserve, never refunded | `models.go:573-575, 630` | none | CHANGES (owner ruling 12): key 0 settles like any key (correction 6); `TestAnOwnerlessHoldIsRefunded` NEW (Task 7) |
| R17 | Server-key hold from an earlier day: payer counter gets no refund; overage still charged | `models.go:590-596, 624-626` | `TestAHoldAcrossMidnightRefundsNothing` (`money_test.go:881`) | ledger `earlier` rule for charges; rewritten on the ledger clock; `TestAHoldFromYesterdayRefundsNoAllowance` NEW |
| R18 | Ledger across midnight: the new day starts at what is still held; the hold settles against today; the consumer share gets no refund from an earlier day's hold | `budget.go:77-81, 175-197` | `TestBudgetRollsOverWithCallsInFlight` (`apiframework_test.go:214`), `TestBudgetCountsTheWholeRequest` (`aicompanion_test.go:1766`) | unchanged; both kept (the second rewritten on the ledger clock) |
| R19 | The server hold settled is the ledger's own, not one rebuilt from the module's day | `models.go:606-621`, `tiers.go:289` | `TestSettlementReturnsTheLedgersOwnHold` (`relayfor_test.go:138`) | `hold.fw` is the only hold; rewritten |
| R20 | A relayed count is held between 0 and the reservation | `tiers.go:293-296` (and `wire.go` `Charged` with `relayed`) | `TestRelayUsageIsNeverTrusted` (`money_test.go:284`, 309-322) | `settle` clamps when `!SpendServer`; rewritten; `TestSettleClampsAndFloors` |
| R21 | Owner's key: a passer-by's hold from an earlier day gives nothing back | `tiers.go:298-301` | `TestAHoldAcrossMidnightRefundsNothing` (887, 901, 905) | ledger `earlier` rule; rewritten |
| R22 | Owner's key, owner's hold: settling does nothing | `tiers.go:291-293` | `TestRelayCallsReserveNothingOfTheServers` (242-245) | empty `hold.fw` is a no-op; rewritten |
| R23 | Each reservation settles exactly once, even when the mind is gone or the goroutine panics | `conversation.go:320-330, 347`, `corememory.go:180-189, 205`, `reflect.go:250-259, 275`, `runtime.go:667-709` | `TestGoneMindStillRefundsItsOwner` (`aicompanion_test.go:2636`), `TestPanickedBackgroundCallsSettle` (`money_test.go:334`) | call sites untouched; both rewritten |

### Read-only checks and display

| # | Rule | Source | Test today | After |
|---|---|---|---|---|
| R24 | `ownerBudgetLeft`: cap <= 0 is true, else `spent < cap` (strict) | `models.go:208-213` | `TestBreakerAndBudgets` (`aicompanion_test.go:943`, 959-966) | reads `Allowance`; rewritten |
| R25 | `modelReadyFor`: relay true; none false; server needs the breaker closed, `HasRoom`, and for an owner call with an owner, `ownerBudgetLeft` | `aicompanion.go:435-452` | `TestModelReadyFollowsTheRoute` (`tiers_test.go:184`) | unchanged logic; rewritten |
| R26 | `strangerMayAsk` refuses when the asker's spend `>= StrangerDailyTokens` (cap > 0, strangers not off) | `listeners.go:568-573` | `TestStrangerDailyCapStopsTheirPrompts` (`aicompanion_test.go:2372`) | reads `Allowance`; rewritten |
| R27 | `strangerMayAsk` refuses when passers-by's spend of this owner `>= StrangerTokensPerOwner` | `listeners.go:576-581` | `TestStrangerTokensPerOwnerCapsThemTogether` (517-527) | reads `Allowance`; rewritten |
| R28 | An allowance refusal does not spend the stranger cooldown | `listeners.go:561-589` (order) | `TestStrangerDailyCapStopsTheirPrompts` (2386-2393) | order unchanged; rewritten |
| R29 | A dispatch refusal marks `budgetSpent` for an owner call, not a passer-by's; success clears it | `runtime.go:621-637` | none | untouched code |
| R30 | `logBudgetRefusal`, once a minute, logs the payer's spend and cap | `runtime.go:1549-1565` | none | reads `Allowance`, and logs `refusedBy` (R43) |
| R31 | `aicompanion status` shows `spentToday` per owner and the cap line | `commands.go:116-119, 157` | none | reads `Allowance` |
| R34 | `HasRoom` is `limit <= 0 || Tokens < limit` | `budget.go:210-217` | `TestModelReadyFollowsTheRoute` (203-206) | unchanged |

### Day

| # | Rule | Source | Test today | After |
|---|---|---|---|---|
| R32 | The module's day rolls on `time.Now()` UTC and resets calls, errors, the three allowance maps and notices; it runs before every check and settle | `aicompanion.go:403-414` + 15 sites | `TestBudgetCountsTheWholeRequest`, `TestAHoldAcrossMidnightRefundsNothing` | allowances roll with the ledger (`rollLocked`); calls, errors, notices roll in `rollCounters` on `Books.Day()`; `TestCountersRollOnTheLedgersClock` NEW |
| R33 | At most `NoticeCallsPerDay` "you notice" moments per owner per day | `autonomy.go:181-192` | `TestNoticedIsCappedAndSparesTheOwnersKey` (`money_test.go:713`) | `rollCounters`; test unchanged |

### Persistence

| # | Rule | Source | Test today | After |
|---|---|---|---|---|
| R35 | `loadBudget` restores only the same UTC day: Day, Calls, Owners, Strangers, StrangersFor, Notices; nil maps become empty | `models.go:656-685` | `TestStrangersForIsKeptWithTheBudget` (`money_test.go:531`, the yaml tag only) | `restoreBudget` seeds the three through `SeedAllowances`, once per dimension per day (the mark is saved in `budget.yaml`); `TestFirstBootSeedsTheOldAllowancesOnce` NEW keeps the yaml check |
| R36 | The old save's Tokens seed the ledger once, on a fresh day only | `models.go:667`, `budget.go:267-283` | `TestSeedTokensOnlyOnceAndOnlyToday` (`apiframework_test.go:266`) | unchanged; `TestFirstBootSeedsTheOldAllowancesOnce` |
| R37 | `saveBudget` saves the ledger first, then Day, Calls, the maps, Notices, and Tokens as the companion's ledger share when the days match | `models.go:687-708` | none | KEPT (owner ruling 12): the three maps are still written, now read from the ledger (`Allowances`), as the backup a quarantine re-seeds from; `TestFirstBootSeedsTheOldAllowancesOnce` checks `budgetStateToSave` |
| R38 | `SaveBudget` copies every map under the lock, clears dirty, re-dirties on a failed write | `budget.go:286-319` | `TestBudgetSavesAndLoads` (`apiframework_test.go:233`; not the copy) | `ByUser` and `Seeded` copied too; `TestSaveBudgetCopiesEveryMapUnderTheLock` NEW (`-race`) |
| R39 | Ledger load: a stale day is a new day; a corrupt file is quarantined and the day starts fresh | `budget.go:102-129` | `TestBudgetSavesAndLoads` | per-user spends and seed marks restart with it, so the next seed applies; `TestAllowancesSaveLoadAndSeedOnce` NEW; the companion's counts come back from its backup (correction 9): `TestAQuarantinedLedgerReseedsFromTheCompanionsBackup`, `TestASameDayRestartDoesNotSeedTwice` NEW (Task 9) |

### Baubles

| # | Rule | Source | Test today | After |
|---|---|---|---|---|
| R40 | A server-key naming reserves prompt + max (doubled with `RetryTransient`) against the server budget and settles `Charged`; a finder's own key reserves nothing | `generate.go:239-294, 200-229` (and `name`, `:151-190`) | `TestGenerateOnTheServersKey` (`baubles_test.go:168`), `TestTheBudgetIsShared` (254), `TestFindersOwnKeyNamesTheirFind` (545) | server route adds the finder charge; own key reserves with `spendServer` false and the finder charge; `TestAFindIsChargedToItsFinder`, `TestAFinderOverTheirAllowanceGetsNoName`, `TestBaublesOverTheirShareFallBack`, `TestAdminRegenChargesNoFinder` NEW; a refused reservation feeds no breaker (as today) and is no longer counted as a failure in `bauble status` (R43) |

**43 rules** (R1 to R43; R42 is dropped on purpose, R43 is new). Every "Test today" test is rewritten, not deleted, except `TestStrangersForIsKeptWithTheBudget`, whose one assertion (the `strangers_for` yaml tag still reads) moves into `TestFirstBootSeedsTheOldAllowancesOnce`.

### T1. The 64 map and `budgetDay` test lines, and what each becomes

Helpers (Task 6): `ownerSpent(m, id)`, `strangerSpent(m, id)`, `strangersForSpent(m, owner)` read; `setOwnerSpent`, `setStrangerSpent`, `setStrangersForSpent` write. A read `m.ownerTokens[X]` becomes `ownerSpent(m, X)`; a write `m.ownerTokens[X] = V` becomes `setOwnerSpent(m, X, V)`; the same for the other two.

| Line | Test | Kind | Becomes |
|---|---|---|---|
| `aicompanion_test.go:1263, 1264, 1267` | TestTokenReservationSettles | read owner[3] | `ownerSpent(m, 3)` (whole test in Task 7 step 6) |
| `aicompanion_test.go:1783` | TestBudgetCountsTheWholeRequest | write `budgetDay` | ledger clock to tomorrow (Task 7 step 6) |
| `aicompanion_test.go:2376` | TestStrangerDailyCapStopsTheirPrompts | write stranger[2] = cap | `setStrangerSpent(m, 2, m.cfg.StrangerDailyTokens)` |
| `aicompanion_test.go:2389` | same | write stranger[2] = 0 | `setStrangerSpent(m, 2, 0)` |
| `aicompanion_test.go:2403, 2404, 2421, 2422, 2426, 2427, 2434` | TestStrangerCallsAreReservedAgainstTheStranger | read owner[1], stranger[2], stranger[3] | `ownerSpent`/`strangerSpent` (whole test in Task 7 step 6) |
| `aicompanion_test.go:2458, 2459` | TestStrangerReservationsCannotSlipPastTheCapTogether | read stranger[2] | `strangerSpent(m, 2)` |
| `aicompanion_test.go:2587, 2588, 2600, 2601, 2628, 2629` | TestStrangerTalkSummaryIsTheStrangersToPayFor | read owner[1], stranger[2] | `ownerSpent(m, 1)`, `strangerSpent(m, 2)` |
| `aicompanion_test.go:2596` | same | write stranger[2] = cap | `setStrangerSpent(m, 2, m.cfg.StrangerDailyTokens)` |
| `aicompanion_test.go:2611` | same | write owner[1] = cap | `setOwnerSpent(m, 1, m.cfg.DailyTokensPerCompanion)` |
| `aicompanion_test.go:2652, 2653` | TestGoneMindStillRefundsItsOwner | read owner[1] | `ownerSpent(m, 1)` |
| `tiers_test.go:188` | TestModelReadyFollowsTheRoute | write owner[5] = cap | `setOwnerSpent(m, 5, m.cfg.DailyTokensPerCompanion)` |
| `tiers_test.go:208` | same | write owner[6] = cap | `setOwnerSpent(m, 6, m.cfg.DailyTokensPerCompanion)` |
| `tiers_test.go:234` | TestRelayCallsReserveNothingOfTheServers | write owner[5] = cap | `setOwnerSpent(m, 5, m.cfg.DailyTokensPerCompanion)` |
| `tiers_test.go:239, 240, 243, 244` | same | read owner[5] | `ownerSpent(m, 5)` |
| `tiers_test.go:261, 263, 266, 267, 273, 274` | TestStrangerRelayCallsStopAtTheStrangerCap | read stranger[2], stranger[3], owner[5] | `strangerSpent`, `ownerSpent` |
| `tiers_test.go:296, 297` | TestStrangerRelayReservationsCannotSlipPastTheCapTogether | read stranger[2] | `strangerSpent(m, 2)` |
| `tiers_test.go:317` | TestRelaySummaryChargesOnlyThePasserBy | write owner[1] = cap | `setOwnerSpent(m, 1, m.cfg.DailyTokensPerCompanion)` |
| `tiers_test.go:321, 334, 347, 348` | same | read stranger[2], owner[1] | `strangerSpent(m, 2)`, `ownerSpent(m, 1)` |
| `tiers_test.go:371` | TestRelayReflectionAndCoreMemorySpendNothingOfTheServers | write owner[1] = cap | `setOwnerSpent(m, 1, m.cfg.DailyTokensPerCompanion)` |
| `tiers_test.go:404, 405` | same | read owner[1] | `ownerSpent(m, 1)` |
| `tiers_test.go:868` | `settleToday` helper | read `budgetDay` | rebuilt `apiframework.Hold` with `Day: m.fw().Day()` (Task 7 step 5) |
| `money_test.go:313, 314, 320, 321` | TestRelayUsageIsNeverTrusted | read stranger[2], stranger[3] | `strangerSpent` |
| `money_test.go:371` | TestPanickedBackgroundCallsSettle | read owner[1] | `ownerSpent(m, 1)` |
| `money_test.go:509, 510` | TestStrangerTokensPerOwnerCapsThemTogether | read strangersFor[5] | `strangersForSpent(m, 5)` |
| `money_test.go:522` | same | write strangersFor[5] = 1500 | `setStrangersForSpent(m, 5, 1500)` |
| `money_test.go:893` | TestAHoldAcrossMidnightRefundsNothing | write `budgetDay` | ledger clock (whole test in Task 7 step 6) |
| `money_test.go:902, 903, 905, 906, 915, 916` | same | read owner[5], stranger[2], strangersFor[5] | `ownerSpent`, `strangerSpent`, `strangersForSpent` |

Count: 25 + 24 + 15 = 64.

### T2. The 49 test lines calling helpers this plan deletes or renames

| Line(s) | Test | Call | Becomes |
|---|---|---|---|
| `aicompanion_test.go:959, 963` | TestBreakerAndBudgets | `m.chargeOwner(7, n)` | `setOwnerSpent(m, 7, 90)` then `setOwnerSpent(m, 7, 110)` |
| `aicompanion_test.go:960, 964` | same | `m.ownerBudgetLeft` | unchanged (kept, reads the ledger) |
| `aicompanion_test.go:1249, 1252, 1255, 1259` | TestTokenReservationSettles | `m.tryReserveTokens` | `tryRoute(m, server, ...)` |
| `aicompanion_test.go:1258, 1262, 1266` | same | `m.settleTokens` | `settleToday(m, server, ...)` |
| `aicompanion_test.go:1782, 1788` | TestBudgetCountsTheWholeRequest | `tryReserveTokens`, `settleTokens` | `reserveRoute` and `settleRoute` with the real hold |
| `aicompanion_test.go:1784` | same | `m.rollDay()` | deleted (ledger clock) |
| `aicompanion_test.go:2375, 2595, 2610` | stranger tests | `m.rollDay()` | deleted |
| `aicompanion_test.go:2400, 2406, 2412, 2448` | stranger reservation tests | `m.tryReserveFor` | `tryRoute(m, server, ...)` |
| `aicompanion_test.go:2409` | same | `m.tryReserveTokens` | `tryRoute(m, server, 1, 0, 900)` |
| `aicompanion_test.go:2420, 2425, 2433` | same | `m.settleFor` | `settleToday(m, server, ...)` |
| `aicompanion_test.go:2429` | same | `m.settleTokens` | `settleToday(m, server, 1, 0, 900, 900)` |
| `aicompanion_test.go:2647` | TestGoneMindStillRefundsItsOwner | `m.reserveRoute` | unchanged |
| `money_test.go:521` | TestStrangerTokensPerOwnerCapsThemTogether | `m.rollDay()` | deleted |
| `money_test.go:532, 537` | TestStrangersForIsKeptWithTheBudget | `budgetState` | test replaced by `TestFirstBootSeedsTheOldAllowancesOnce` (Task 9) |
| `money_test.go:745` | TestNoticedIsCappedAndSparesTheOwnersKey | `m.noticesToday = nil` | unchanged |
| `money_test.go:886, 887, 913` | TestAHoldAcrossMidnightRefundsNothing | `m.reserveRoute` | unchanged |
| `money_test.go:894` | same | `m.rollDay()` | ledger clock |
| `money_test.go:897, 898` | same | `chargeOwner`, `chargeStrangerFor` | `setOwnerSpent`, `setStrangerSpent`, `setStrangersForSpent` |
| `money_test.go:900, 901, 914` | same | `m.settleRoute` | unchanged |
| `relayfor_test.go:140` | TestSettlementReturnsTheLedgersOwnHold | `m.rollDay()` | deleted |
| `relayfor_test.go:143, 147` | same | `reserveRoute`, `settleRoute` | unchanged; assertion on `h.day` replaced |
| `tiers_test.go:186, 232, 315, 369` | four relay tests | `m.rollDay()` | deleted |
| `tiers_test.go:860, 862` | `tryRoute` helper | `reserveRoute` | unchanged |
| `tiers_test.go:868` | `settleToday` helper | `settleRoute(hold{...})` | rebuilt hold (Task 7 step 5) |

---

## Task 1: Re-verify the rule table and commit the plan

**Files:**
- Modify: `docs/superpowers/plans/2026-09-28-s5-allowance-ledger.md` (this file, only where a row drifted; it is on master through the docs-only PR)

- [ ] **Step 1: Confirm slice H is merged on master**

Run (Bash), from the main checkout:
```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git fetch origin
git log --oneline -1 master; git log --oneline -1 origin/master
git grep -n 'func playerRouteOpen\|func (m \*BaublesModule) takeFinderSlot' origin/master -- modules/baubles/
git grep -n 'hardLocked' origin/master -- internal/configs/ | head -3
```
Expected: both `git grep`s print hits (slice H's `playerRouteOpen` and `takeFinderSlot`, slice M's `hardLocked`). No hit means slice H (or M) has not merged: stop and report; S5 does not start. If `master` is behind `origin/master`, fast-forward it with `git fetch origin master:master` (it refuses when master is checked out in some worktree; then run `git pull --ff-only` in that worktree instead) and re-run the two log lines until they print the same commit.

Result at `09964d50f`: slice H merged (H1 #187, the sweep #188, H2 #190, H3 #191), and `takeFinderSlot` and `hardLocked` hit. `playerRouteOpen` did NOT land under that or any name (the relay route's shape is in the facts table's slice H row); the `takeFinderSlot` hit is the evidence. The main checkout's local `master` was stale, so the worktree was cut from `origin/master` directly.

- [ ] **Step 2: Create the S5 worktree and record the base**

Run (Bash), from the main checkout:
```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git worktree add -b fix/baubles-allowance-ledger C:/tmp/dogmud-baubles-s5 origin/master
cd /c/tmp/dogmud-baubles-s5
BASE=$(git rev-parse HEAD); echo "BASE=$BASE"
git ls-files -v _datafiles/config.yaml
git status --short
```
Expected: the worktree is created, `BASE=<sha>` prints (write the SHA into the task list you are tracking; shell variables do not survive between tool calls), `config.yaml` shows `H` (a fresh worktree carries no skip-worktree bit), and the tree is clean. Every later command that names `$BASE` sets it first with `BASE=$(git merge-base HEAD origin/master)` (never the main checkout's local `master`, which can be stale), which prints this same SHA for as long as the branch is never rebased; if it ever prints a different SHA, stop and report. Every later command runs in `/c/tmp/dogmud-baubles-s5`.

Then confirm the base is green:
```bash
cd /c/tmp/dogmud-baubles-s5 && go build ./... && go test ./internal/apiframework/ ./modules/aicompanion/ ./modules/baubles/ ./internal/baubles/ 2>&1 | tail -5
```
Expected: four `ok` lines. Stop and report if any fails: S5 starts from green.

Result: `BASE=09964d50f91d3e86f91ad58dd72e908ef36b7f64`, `config.yaml` `H` and equal to the HEAD blob, tree clean; build clean and four `ok` (`internal/apiframework`, `modules/aicompanion`, `modules/baubles`, `internal/baubles`).

- [ ] **Step 3: Re-grep every row whose file slice H may have touched**

Run each (Bash), from `/c/tmp/dogmud-baubles-s5`:
```bash
grep -n 'func (l \*ledger) reserve\|func (l \*ledger) settle\|func SaveBudget\|func (k \*Books) SeedTokens\|type Hold struct\|type ledgerState struct\|func (l \*ledger) rollLocked' internal/apiframework/budget.go
grep -n 'type ServerSettings struct\|switch budget := int' internal/apiframework/settings.go
grep -n 'DailyTokenBudget ConfigInt\|func (a \*APIFramework) Validate' internal/configs/config.apiframework.go
grep -n 'func viaServer\|func viaPlayer\|apiframework.Reserve\|apiframework.Settle\|apiframework.Charged\|func (m \*BaublesModule) name\|takeFinderSlot\|takeServerSlot\|playerRouteOpen' modules/baubles/generate.go
grep -n 'func (m \*AICompanionModule) \(rollDay\|modelReadyFor\|ownerBudgetLeft\|chargeOwner\|chargeStranger\|strangerFits\|chargeStrangerFor\|tryReserveTokens\|reserveFor\|settleHeld\|loadBudget\|saveBudget\|reserveRoute\|settleRoute\|strangerMayAsk\|logBudgetRefusal\|cmdStatus\|notice\)(' modules/aicompanion/*.go
grep -n 'rollDay()' modules/aicompanion/*.go
```
Expected: every function exists; line numbers match the rule table within slice H's shifts. For each shifted row, correct its line number in this file with the Edit tool (never a script). Run the count separately (it may legitimately differ):
```bash
grep -n 'ownerTokens\|strangerTokens\|strangersFor\|budgetDay' modules/aicompanion/*_test.go | wc -l
```
Expected: `64`. If not 64, update table T1 to the real lines before going on.

Then re-find the config blocks (the pre-H blob had `APIFramework:` at 2111 and `baubles:` at 2648):
```bash
grep -n '^APIFramework:\|^  baubles:\|  BreakerSeconds: 0\|MaxConcurrent: 4\|fresh allowance' _datafiles/config.yaml
```

Result at `09964d50f`: every function exists. `budget.go`, `models.go`, `tiers.go`, `runtime.go`, `commands.go`, `aicompanion.go` and the other companion sources but `listeners.go` did not move. Shifted and corrected in this file: `settings.go` (+7: `ServerSettings` 68-80, `resolveServer`'s switch 257-269), `config.apiframework.go` (+1 after line 24), `listeners.go` (+11 after line 251: `strangerMayAsk` at 561, `rollDay()` at 569 and 577), `aicompanion_test.go` (+4 throughout the lines T1 and T2 name), `apiframework_test.go` (`Reserve` callers 654 and 665), `baubles_test.go` (`Reserve` caller 259, tests at 168, 254, 545), `modules/baubles/generate.go` (reshaped: `name` 151, `viaPlayer` 200, `viaServer` 239, `Reserve` 258, `Settle` 282, `m.count` 53). The count is `64`. Config: `APIFramework:` 2149 (`BreakerSeconds: 0` at 2171), `fresh allowance` 2595, `baubles:` 2688 (`MaxConcurrent: 4` at 2698, block ends 2708).

- [ ] **Step 4: Confirm every quoted anchor this plan edits against exists in the merged code**

Each later step replaces or inserts after an exact string. Slice H and FinalTwist's moderation reorder (which lands in `modules/baubles/generate.go`, in `viaServer`, `name` and `viaPlayer`) may have changed some. Run (Bash):
```bash
cd /c/tmp/dogmud-baubles-s5
while IFS='|' read -r f a; do grep -qF -- "$a" "$f" && echo "ok   $f" || echo "MISS $f: $a"; done <<'EOF'
internal/apiframework/budget.go|var ErrOverBudget = errors.New(`daily token budget spent`)
internal/apiframework/budget.go|st.ByConsumer, st.CallsBy = byC, callsBy
internal/apiframework/budget.go|if l.st.CallsBy == nil {
internal/apiframework/apiframework_test.go|budget.reserve(ConsumerCompanion, 600, 1000)
internal/apiframework/apiframework_test.go|own.Reserve(ConsumerCompanion, 300)
internal/apiframework/settings.go|BreakerSeconds   int
internal/configs/config.apiframework.go|BreakerSeconds
modules/aicompanion/models.go|m.fw().Reserve(apiframework.ConsumerCompanion, tokens)
modules/aicompanion/models.go|apiframework.Hold{Consumer: apiframework.ConsumerCompanion, Tokens: reserved, Day: holdDay}
modules/aicompanion/models.go|m.budgetDay = st.Day
modules/aicompanion/commands.go|m.ownerTokens[c.ownerUserId]
modules/aicompanion/runtime.go|m.logBudgetRefusal(ownerId, asker, reserved)
modules/aicompanion/runtime.go|`askerSpentToday`, m.strangerTokens[askerId]
modules/aicompanion/runtime.go|`ownerSpentToday`, m.ownerTokens[ownerId]
modules/baubles/generate.go|m.count(playerKey, err != nil)
modules/baubles/generate.go|apiframework.Reserve(apiframework.ConsumerBaubles, reserve)
modules/baubles/generate.go|viaServer(ctx, cfg, chat)
modules/baubles/generate.go|req.FinderUserId, relayModel, chat)
modules/baubles/baubles_test.go|apiframework.Reserve(apiframework.ConsumerCompanion, 2500)
modules/baubles/config.go|MaxConcurrent
_datafiles/config.yaml|  BreakerSeconds: 0
_datafiles/config.yaml|    MaxConcurrent: 4           # server-key calls at once; more are generic
_datafiles/config.yaml|    # not hand out a fresh allowance.
internal/apiframework/context.md|3. `Reserve(consumer, worstCase)` holds tokens against the one budget.
internal/apiframework/context.md|   and books the real use under the consumer.
internal/apiframework/context.md|  display only, not separate caps. The companion keeps its own per-player
internal/apiframework/context.md|  the companion's old saved total once, on a fresh day only. The directory is
internal/apiframework/context.md|a `configs.ConfigSecret`.
modules/aicompanion/context.md|module's own budget file keeps the per-owner and passer-by counts, and an
modules/aicompanion/context.md|  of that owner's companion (`StrangerTokensPerOwner`, `strangersFor`,
modules/aicompanion/context.md|  kept in the budget file; `strangerFits`, `chargeStrangerFor`) and, on
modules/baubles/context.md|`MaxCompletionTokens` (800), `RetryTransient` (false), `MaxConcurrent` (4,
modules/baubles/context.md|server-key calls only), `ModerateOutput` (true), `ModerationModel` (omni-moderation-latest),
modules/baubles/context.md|   player's browser relay, on their key. It costs the server nothing. A
modules/baubles/context.md|   lets one probe through); a hold on the one daily budget; `apiframework.Post`
modules/baubles/context.md|The key, endpoint, daily budget and breaker are not here: they are the
docs/aicompanion/settings.md|server-key total, the limit and the companion's share of it. The spend is
docs/aicompanion/settings.md|kept in `_datafiles/apiframework/budget.yaml` across restarts.
EOF
```
Expected: every line `ok`. For each `MISS`, find the string's new form (`grep -n` a distinctive part of it), and correct the step that quotes it in this plan with the Edit tool before going on. A `MISS` in `generate.go` means FinalTwist's reorder or slice H reshaped the route; Task 10 Step 4 says how to apply its three pieces to the shape that landed.

Result at `09964d50f`: two anchors had drifted, and the list above now quotes their current form. `internal/apiframework/context.md`'s `defaults on purpose.` is no longer a line of its own (slice H continued the paragraph with the endpoint allowlist and `ConfigSecret`), so Task 14 now appends the share sentence after `a \`configs.ConfigSecret\`.`. `modules/baubles/context.md`'s Config list re-wrapped `` `MaxConcurrent` (4, `` / `` server-key calls only), `` across two lines, so Task 14 now quotes both lines; the two route-step anchors in that file are new (Task 14 also says the finder allowance in the route description slice H wrote). Every `generate.go` anchor still matched, but the code around them was reshaped: Task 10 was rewritten against the merged `name`, `viaPlayer`, `viaServer` and `generate`.

- [ ] **Step 5: Commit any drift corrections**

Only if Steps 3 or 4 changed this plan:
```bash
cd /c/tmp/dogmud-baubles-s5
git add docs/superpowers/plans/2026-09-28-s5-allowance-ledger.md
git commit -F - <<'EOF'
docs(plan): S5 plan rows re-verified against master after slice H

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```
No `docs/README.md` change: the plan's row reached master with the docs-only PR.

---

## Task 2: Ledger charges, reserve with spendServer, Allowance, Day

**Files:**
- Modify: `internal/apiframework/budget.go:36-60, 76-88, 131-160`
- Create: `internal/apiframework/allowance_test.go`
- Modify: `internal/apiframework/apiframework_test.go:182,186,189,206,219,237,654,665`
- Modify: `modules/aicompanion/models.go:567`, `modules/baubles/generate.go:258`, `modules/baubles/baubles_test.go:259`

- [ ] **Step 1: Write the failing tests**

Create `internal/apiframework/allowance_test.go`:
```go
package apiframework

import (
	"errors"
	"testing"
)

// R1, R3, R4, R8, R41: every charge is checked, and a refusal anywhere
// holds nothing anywhere.
func TestReserveChargesEveryAllowanceOrNone(t *testing.T) {
	ResetBudgetForTest(``)
	stranger := Charge{Dim: DimCompanionStranger, UserId: 2, Limit: 1000}
	perOwner := Charge{Dim: DimCompanionStrangersFor, UserId: 5, Limit: 1500}
	if _, err := budget.reserve(ConsumerCompanion, 900, 0, 0, true, []Charge{stranger, perOwner}); err != nil {
		t.Fatal(err)
	}
	other := Charge{Dim: DimCompanionStranger, UserId: 3, Limit: 1000}
	if _, err := budget.reserve(ConsumerCompanion, 700, 0, 0, true, []Charge{other, perOwner}); !errors.Is(err, ErrOverAllowance) {
		t.Fatalf("the second charge refuses the whole reservation: %v", err)
	}
	if Allowance(DimCompanionStranger, 3) != 0 || Allowance(DimCompanionStrangersFor, 5) != 900 || Today().Tokens != 900 {
		t.Fatalf("a refusal holds nothing anywhere: other=%d perOwner=%d server=%d",
			Allowance(DimCompanionStranger, 3), Allowance(DimCompanionStrangersFor, 5), Today().Tokens)
	}
	// The server's budget refusing charges no allowance either.
	if _, err := budget.reserve(ConsumerCompanion, 200, 1000, 0, true, []Charge{other}); !errors.Is(err, ErrOverBudget) {
		t.Fatalf("over the server's budget: %v", err)
	}
	if Allowance(DimCompanionStranger, 3) != 0 {
		t.Fatal("a server refusal leaves the allowance untouched")
	}
	// A limit of 0 is no cap, and the spend is still counted.
	if _, err := budget.reserve(ConsumerCompanion, 5000, 0, 0, true, []Charge{{Dim: DimCompanionOwner, UserId: 7}}); err != nil {
		t.Fatal(err)
	}
	if Allowance(DimCompanionOwner, 7) != 5000 {
		t.Fatalf("counted with no cap: %d", Allowance(DimCompanionOwner, 7))
	}
}

// R11: a relayed call (the player's own key) is charged to its allowances
// and to nothing of the server's, and the server's budget cannot refuse it.
func TestARelayReserveLeavesTheServerAlone(t *testing.T) {
	ResetBudgetForTest(``)
	c := Charge{Dim: DimCompanionStranger, UserId: 2, Limit: 1000}
	h, err := budget.reserve(ConsumerCompanion, 400, 1000, 0, false, []Charge{c})
	if err != nil {
		t.Fatal(err)
	}
	if h.SpendServer || len(h.Charges) != 1 || h.Tokens != 400 || h.Day != Today().Day {
		t.Fatalf("the hold records what it touched: %+v", h)
	}
	if u := Today(); u.Tokens != 0 || u.Outstanding != 0 || u.Calls != 0 || len(u.ByConsumer) != 0 {
		t.Fatalf("nothing of the server's: %+v", u)
	}
	SetSpentForTest(1000, 0)
	if _, err := budget.reserve(ConsumerCompanion, 400, 1000, 0, false, []Charge{c}); err != nil {
		t.Fatal("the server's spent budget does not refuse the player's own key")
	}
	if Allowance(DimCompanionStranger, 2) != 800 {
		t.Fatalf("both charged to the allowance: %d", Allowance(DimCompanionStranger, 2))
	}
}

// The ledger's clock is the only day.
func TestDayIsTheLedgersClock(t *testing.T) {
	k := NewBooksForTest()
	if k.Day() != Today().Day {
		t.Fatalf("today: %s vs %s", k.Day(), Today().Day)
	}
}

// SetAllowanceForTest sets one counter, for tests that start mid-day.
func TestSetAllowanceForTest(t *testing.T) {
	k := NewBooksForTest()
	k.SetAllowanceForTest(DimCompanionOwner, 5, 1234)
	if k.Allowance(DimCompanionOwner, 5) != 1234 || k.Allowance(DimCompanionOwner, 6) != 0 {
		t.Fatal("one counter, by dimension and user")
	}
}

// R43: a refusal says which counter refused it, so a log line can tell a
// spent server day from one player's spent allowance.
func TestARefusalNamesItsCounter(t *testing.T) {
	ResetBudgetForTest(``)
	_, err := budget.reserve(ConsumerCompanion, 200, 100, 0, true, nil)
	if !errors.Is(err, ErrOverBudget) || RefusedBy(err) != RefusedGlobal {
		t.Fatalf("the day's budget: %v (%q)", err, RefusedBy(err))
	}
	c := Charge{Dim: DimCompanionStrangersFor, UserId: 5, Limit: 100}
	_, err = budget.reserve(ConsumerCompanion, 200, 0, 0, true, []Charge{{Dim: DimCompanionStranger, UserId: 2}, c})
	if !errors.Is(err, ErrOverAllowance) || RefusedBy(err) != DimCompanionStrangersFor {
		t.Fatalf("the charge that refused: %v (%q)", err, RefusedBy(err))
	}
	if RefusedBy(nil) != `` || RefusedBy(errors.New(`other`)) != `` {
		t.Fatal("anything else names no counter")
	}
}
```

- [ ] **Step 2: Run to confirm it fails to compile**

Run: `cd /c/tmp/dogmud-baubles-s5 && go test ./internal/apiframework/ -run 'TestReserveChargesEveryAllowanceOrNone|TestARelayReserveLeavesTheServerAlone|TestDayIsTheLedgersClock|TestSetAllowanceForTest|TestARefusalNamesItsCounter' 2>&1 | head`
Expected: build failure naming `Charge`, `DimCompanionStranger`, `Allowance`, `ErrOverAllowance`, `RefusedBy`.

- [ ] **Step 3: Implement in `budget.go`**

Add `"strconv"` to the imports. After `ErrOverBudget` (line 37) add:
```go
// ErrOverAllowance is a reservation refused because one of its per-user
// allowances (a Charge) is spent.
var ErrOverAllowance = errors.New(`daily allowance spent`)

// ErrOverShare is a reservation refused because its consumer has held all
// of its share of the day's budget (ServerSettings.SharePercent).
var ErrOverShare = errors.New(`consumer's share of the daily token budget spent`)

// Per-user allowance dimensions. Each is one daily counter per user id,
// kept by the ledger beside the server's totals.
const (
	DimCompanionOwner        = `companion.owner`        // an owner's own companion calls
	DimCompanionStranger     = `companion.stranger`     // one passer-by, any companion
	DimCompanionStrangersFor = `companion.strangersfor` // all passers-by, one owner's companion
	DimBaublesFinder         = `baubles.finder`         // one finder's namings
)

// Charge names one per-user daily allowance a reservation also counts
// against. Limit is that allowance's size, read by the caller from its own
// config at the moment of the call (the ledger cannot read a module's
// config); 0 or less is no cap, and the spend is still counted.
type Charge struct {
	Dim    string
	UserId int
	Limit  int
}

func allowanceKey(dim string, userId int) string { return dim + `:` + strconv.Itoa(userId) }

func (c Charge) key() string { return allowanceKey(c.Dim, c.UserId) }

// The counters a reservation can be refused by, besides a Charge's own
// dimension (RefusedBy).
const (
	RefusedGlobal = `global` // the day's budget (ErrOverBudget)
	RefusedShare  = `share`  // the consumer's share of it (ErrOverShare)
)

// RefusalError is a reservation Reserve refused, naming the counter that
// refused it: RefusedGlobal, RefusedShare, or a Charge's Dim. errors.Is
// still matches ErrOverBudget, ErrOverShare and ErrOverAllowance.
type RefusalError struct {
	Counter string
	err     error
}

func (e *RefusalError) Error() string { return e.err.Error() + ` (` + e.Counter + `)` }
func (e *RefusalError) Unwrap() error { return e.err }

func refused(counter string, err error) error { return &RefusalError{Counter: counter, err: err} }

// RefusedBy is the counter that refused err's reservation, or "" when err
// is not a refusal (nil, or any other failure).
func RefusedBy(err error) string {
	var r *RefusalError
	if errors.As(err, &r) {
		return r.Counter
	}
	return ``
}
```
Replace the `Hold` struct (lines 45-51) with:
```go
// Hold is one call's reservation, returned by Reserve and given back to
// Settle. It records every counter it touched: the server's (SpendServer)
// and each Charge.
type Hold struct {
	Consumer    string
	Tokens      int
	Day         string
	SpendServer bool
	Charges     []Charge
}
```
In `ledgerState` add after `CallsBy`:
```go
	// ByUser is each per-user allowance's spend today, keyed
	// "<dim>:<userId>".
	ByUser map[string]int `yaml:"by_user,omitempty"`
	// Seeded marks each dimension SeedAllowances has already seeded today,
	// one mark per dimension (a new day starts with none).
	Seeded map[string]bool `yaml:"seeded,omitempty"`
```
In `rollLocked`, after the `CallsBy` nil check, add:
```go
	if l.st.ByUser == nil {
		l.st.ByUser = map[string]int{}
	}
	if l.st.Seeded == nil {
		l.st.Seeded = map[string]bool{}
	}
```
Replace lines 131-160 (`Reserve`, `Books.Reserve`, `ledger.reserve`) with:
```go
// Reserve holds tokens for consumer, all or nothing, under the ledger's one
// lock: when spendServer, against today's server budget
// (Server().DailyTokenBudget); and against every per-user allowance in
// charges. spendServer false is a player's own key: its allowances are
// charged and nothing of the server's is. It refuses with a *RefusalError
// wrapping ErrOverBudget, ErrOverShare or ErrOverAllowance and naming the
// counter (RefusedBy), holding nothing.
func Reserve(consumer string, tokens int, spendServer bool, charges ...Charge) (Hold, error) {
	return shared.Reserve(consumer, tokens, spendServer, charges...)
}

// Reserve on these books.
func (k *Books) Reserve(consumer string, tokens int, spendServer bool, charges ...Charge) (Hold, error) {
	s := Server()
	return k.l.reserve(consumer, tokens, s.DailyTokenBudget, 0, spendServer, charges)
}

// reserve checks everything before it adds anything, so a refusal holds
// nothing anywhere. sharePct caps consumer's part of limit (0 is no share
// cap).
func (l *ledger) reserve(consumer string, tokens int, limit int, sharePct int, spendServer bool, charges []Charge) (Hold, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.loadLocked()
	l.rollLocked()
	if tokens < 0 {
		tokens = 0
	}
	if spendServer && limit > 0 && l.st.Tokens+tokens > limit {
		return Hold{}, refused(RefusedGlobal, ErrOverBudget)
	}
	for _, c := range charges {
		if c.Limit > 0 && l.st.ByUser[c.key()]+tokens > c.Limit {
			return Hold{}, refused(c.Dim, ErrOverAllowance)
		}
	}
	if spendServer {
		l.st.Tokens += tokens
		l.st.ByConsumer[consumer] += tokens
		l.st.Calls++
		l.st.CallsBy[consumer]++
		l.outstanding += tokens
	}
	for _, c := range charges {
		l.st.ByUser[c.key()] += tokens
	}
	l.dirty = true
	return Hold{Consumer: consumer, Tokens: tokens, Day: l.st.Day, SpendServer: spendServer,
		Charges: append([]Charge(nil), charges...)}, nil
}
```
The `sharePct` parameter is checked in Task 4; this task passes 0 from `Books.Reserve` and the tests.

After `HasRoom` (line 217) add:
```go
// Allowance is one per-user allowance's spend today (Charge.Dim and
// UserId), for read-only checks and the admin views.
func Allowance(dim string, userId int) int {
	return shared.Allowance(dim, userId)
}

// Allowance on these books.
func (k *Books) Allowance(dim string, userId int) int {
	k.l.mu.Lock()
	defer k.l.mu.Unlock()
	k.l.loadLocked()
	k.l.rollLocked()
	return k.l.st.ByUser[allowanceKey(dim, userId)]
}

// Day is the ledger's day: the UTC date on its clock. Every daily count a
// feature keeps rolls on this, so no feature keeps a clock of its own.
func (k *Books) Day() string {
	k.l.mu.Lock()
	defer k.l.mu.Unlock()
	k.l.loadLocked()
	k.l.rollLocked()
	return k.l.st.Day
}
```
After `SetSpentForTest` (line 348) add:
```go
// SetAllowanceForTest sets one per-user allowance's spend today.
func (k *Books) SetAllowanceForTest(dim string, userId int, tokens int) {
	k.l.mu.Lock()
	defer k.l.mu.Unlock()
	k.l.loadLocked()
	k.l.rollLocked()
	k.l.st.ByUser[allowanceKey(dim, userId)] = tokens
}
```

- [ ] **Step 4: Update every caller of the old signature**

With the Edit tool, exactly:
- `internal/apiframework/apiframework_test.go`: `budget.reserve(ConsumerCompanion, 600, 1000)` becomes `budget.reserve(ConsumerCompanion, 600, 1000, 0, true, nil)`; likewise `(ConsumerBaubles, 500, 1000)`, `(ConsumerBaubles, 300, 1000)`, `(ConsumerBaubles, 1000, 0)`, `(ConsumerBaubles, 1000, 5000)`, `(ConsumerCompanion, 500, 0)` each gain `, 0, true, nil` before the closing parenthesis; `own.Reserve(ConsumerCompanion, 300)` becomes `own.Reserve(ConsumerCompanion, 300, true)`; `Reserve(ConsumerBaubles, 50)` becomes `Reserve(ConsumerBaubles, 50, true)`.
- `modules/aicompanion/models.go:567`: `m.fw().Reserve(apiframework.ConsumerCompanion, tokens)` becomes `m.fw().Reserve(apiframework.ConsumerCompanion, tokens, true)`.
- `modules/baubles/generate.go:258` (in `viaServer`): `apiframework.Reserve(apiframework.ConsumerBaubles, reserve)` becomes `apiframework.Reserve(apiframework.ConsumerBaubles, reserve, true)`.
- `modules/baubles/baubles_test.go:259`: `apiframework.Reserve(apiframework.ConsumerCompanion, 2500)` becomes `apiframework.Reserve(apiframework.ConsumerCompanion, 2500, true)`.

Then confirm nothing was missed: `go build ./... && go vet ./internal/apiframework/ ./modules/aicompanion/ ./modules/baubles/`
Expected: no output. Then confirm no caller compares a refusal by identity (a `*RefusalError` is never `==` the sentinel); run standalone, zero matches (exit 1) is the pass:
```bash
grep -rn '== apiframework.ErrOver\|== ErrOver\|!= apiframework.ErrOver\|!= ErrOver' --include=*.go .
```

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/apiframework/ ./modules/aicompanion/ ./modules/baubles/ 2>&1 | tail -5`
Expected: three `ok`.

- [ ] **Step 6: Prove the all-or-nothing test can fail**

Temporarily move the `for _, c := range charges { if c.Limit > 0 ...` check block in `reserve` below the `if spendServer { l.st.Tokens += tokens ...` block. Run `go test ./internal/apiframework/ -run TestReserveChargesEveryAllowanceOrNone`. Expected: FAIL with "a refusal holds nothing anywhere". Restore the order and re-run: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/apiframework/budget.go internal/apiframework/allowance_test.go internal/apiframework/apiframework_test.go modules/aicompanion/models.go modules/baubles/generate.go modules/baubles/baubles_test.go
git commit -F - <<'EOF'
feat(apiframework): per-user charges on the ledger, reserve all or nothing

Reserve takes spendServer and any number of Charges and checks them all
under the one lock before it adds anything. Allowance reads a counter and
Books.Day is the ledger's day.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

## Task 3: Settle every counter the hold touched

**Files:**
- Modify: `internal/apiframework/budget.go:175-202`
- Modify: `internal/apiframework/allowance_test.go`
- Modify: `modules/aicompanion/models.go:603, 619`

- [ ] **Step 1: Write the failing tests**

Append to `internal/apiframework/allowance_test.go` (add `"time"` to its imports):
```go
// R14, R15, R20: a relayed count is held between nothing and its hold; a
// server-key count is trusted, overage included; no counter goes negative.
func TestSettleClampsAndFloors(t *testing.T) {
	ResetBudgetForTest(``)
	c := Charge{Dim: DimCompanionStranger, UserId: 2, Limit: 10000}
	relay, _ := budget.reserve(ConsumerCompanion, 400, 0, 0, false, []Charge{c})
	budget.settle(relay, 5000, false)
	if Allowance(DimCompanionStranger, 2) != 400 {
		t.Fatalf("a relayed count never charges past its hold: %d", Allowance(DimCompanionStranger, 2))
	}
	relay2, _ := budget.reserve(ConsumerCompanion, 400, 0, 0, false, []Charge{c})
	budget.settle(relay2, -900, false)
	if Allowance(DimCompanionStranger, 2) != 400 {
		t.Fatalf("nor below nothing: %d", Allowance(DimCompanionStranger, 2))
	}
	if u := Today(); u.Tokens != 0 || u.Outstanding != 0 {
		t.Fatalf("a relayed settlement touches nothing of the server's: %+v", u)
	}

	owner := Charge{Dim: DimCompanionOwner, UserId: 5}
	h, _ := budget.reserve(ConsumerCompanion, 300, 0, 0, true, []Charge{owner})
	budget.settle(h, 450, false)
	share := 0
	for _, cu := range Today().ByConsumer {
		if cu.Consumer == ConsumerCompanion {
			share = cu.Tokens
		}
	}
	if Today().Tokens != 450 || share != 450 || Allowance(DimCompanionOwner, 5) != 450 || Today().Outstanding != 0 {
		t.Fatalf("server-key overage is charged everywhere: total=%d share=%d owner=%d", Today().Tokens, share, Allowance(DimCompanionOwner, 5))
	}

	// Both counters set below the hold, so the refund would take them
	// negative: the allowance and the server total (budget.go:186) floor.
	h2, _ := budget.reserve(ConsumerCompanion, 300, 0, 0, true, []Charge{owner})
	shared.SetAllowanceForTest(DimCompanionOwner, 5, 100)
	SetSpentForTest(100, 300)
	budget.settle(h2, 0, false)
	if Allowance(DimCompanionOwner, 5) != 0 {
		t.Fatalf("an allowance floors at nothing: %d", Allowance(DimCompanionOwner, 5))
	}
	if u := Today(); u.Tokens != 0 || u.Outstanding != 0 {
		t.Fatalf("the server total floors at nothing: %+v", u)
	}
}

// R17, R21: a hold made yesterday gives nothing back to today's
// allowances, and its overage is still charged.
func TestAHoldFromYesterdayRefundsNoAllowance(t *testing.T) {
	ResetBudgetForTest(``)
	day1 := time.Date(2026, 9, 26, 23, 59, 0, 0, time.UTC)
	SetClockForTest(func() time.Time { return day1 })
	t.Cleanup(func() { SetClockForTest(time.Now) })
	owner := Charge{Dim: DimCompanionOwner, UserId: 5}
	stranger := Charge{Dim: DimCompanionStranger, UserId: 2}
	hs, _ := budget.reserve(ConsumerCompanion, 900, 0, 0, true, []Charge{owner})
	hs2, _ := budget.reserve(ConsumerCompanion, 100, 0, 0, true, []Charge{owner})
	hr, _ := budget.reserve(ConsumerCompanion, 400, 0, 0, false, []Charge{stranger})

	SetClockForTest(func() time.Time { return day1.Add(2 * time.Minute) })
	if Allowance(DimCompanionOwner, 5) != 0 || Allowance(DimCompanionStranger, 2) != 0 {
		t.Fatal("a new day's allowances start at nothing")
	}
	shared.SetAllowanceForTest(DimCompanionOwner, 5, 500)
	shared.SetAllowanceForTest(DimCompanionStranger, 2, 300)
	budget.settle(hs, 100, false)
	budget.settle(hs2, 250, false)
	budget.settle(hr, 0, false)
	if Allowance(DimCompanionOwner, 5) != 650 {
		t.Fatalf("no refund from yesterday, overage still charged: %d", Allowance(DimCompanionOwner, 5))
	}
	if Allowance(DimCompanionStranger, 2) != 300 {
		t.Fatalf("the relayed hold gives nothing back: %d", Allowance(DimCompanionStranger, 2))
	}
	if u := Today(); u.Outstanding != 0 || u.Tokens != 350 {
		t.Fatalf("the server settles as before: %+v", u)
	}
}
```

- [ ] **Step 2: Run to confirm they fail**

Run: `go test ./internal/apiframework/ -run 'TestSettleClampsAndFloors|TestAHoldFromYesterdayRefundsNoAllowance' 2>&1 | head -20`
Expected: FAIL. `TestSettleClampsAndFloors` fails with "nor below nothing: 800" (today's settle never touches a charge, so the first clamp check passes by accident and the refund check is the one that catches it); `TestAHoldFromYesterdayRefundsNoAllowance` fails with "no refund from yesterday, overage still charged".

- [ ] **Step 3: Replace `ledger.settle` (lines 175-202)**

```go
func (l *ledger) settle(h Hold, used int, failed bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.loadLocked()
	l.rollLocked()
	if used < 0 {
		used = 0
	}
	// A count relayed through a player's browser, which that player can
	// write, may lower a charge below its hold, never raise it past it. The
	// provider's own count on the server's key is trusted: usage past the
	// hold is charged.
	if !h.SpendServer && used > h.Tokens {
		used = h.Tokens
	}
	diff := used - h.Tokens
	earlier := h.Day != l.st.Day
	if h.SpendServer {
		l.outstanding -= h.Tokens
		if l.outstanding < 0 {
			l.outstanding = 0
		}
		l.st.Tokens += diff
		if l.st.Tokens < 0 {
			l.st.Tokens = 0
		}
		// What a consumer is shown is what it spent today; a hold from an
		// earlier day gives nothing back to today's share.
		share := diff
		if earlier && share < 0 {
			share = 0
		}
		l.st.ByConsumer[h.Consumer] += share
		if l.st.ByConsumer[h.Consumer] < 0 {
			l.st.ByConsumer[h.Consumer] = 0
		}
		if failed {
			l.st.Failures++
		}
	}
	// Each allowance started the new day at nothing, so a hold from an
	// earlier day gives it nothing back; usage past the hold is still
	// charged.
	each := diff
	if earlier && each < 0 {
		each = 0
	}
	for _, c := range h.Charges {
		k := c.key()
		l.st.ByUser[k] += each
		if l.st.ByUser[k] < 0 {
			l.st.ByUser[k] = 0
		}
	}
	l.dirty = true
}
```
Update the `Settle` doc comment (line 162) to: `// Settle replaces a reservation with what the call really used (use Charged to work that out), on every counter the hold touched. used below 0 is 0; on a player's own key (SpendServer false) it is at most the hold. failed counts a failed server-key call in the day's figures. A hold from an earlier day settles the server's total against today, since today started at what was still held, and gives no refund to a consumer share or an allowance.`

- [ ] **Step 4: Mark the companion's rebuilt holds as server holds**

The companion still rebuilds holds until Task 7; without `SpendServer` they would no longer settle the server. In `modules/aicompanion/models.go`, change both literals `apiframework.Hold{Consumer: apiframework.ConsumerCompanion, Tokens: reserved, Day: holdDay}` (in `settleForDay` and `settleHeld`) to `apiframework.Hold{Consumer: apiframework.ConsumerCompanion, Tokens: reserved, Day: holdDay, SpendServer: true}`.

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/apiframework/ ./modules/aicompanion/ ./modules/baubles/ 2>&1 | tail -5`
Expected: three `ok`.

- [ ] **Step 6: Prove the clamp test can fail**

Delete the line `if !h.SpendServer && used > h.Tokens {` block temporarily; run `go test ./internal/apiframework/ -run TestSettleClampsAndFloors`. Expected: FAIL "a relayed count never charges past its hold". Change it to `if used > h.Tokens {` (the spec's clamp-for-everyone); expected: FAIL "server-key overage is charged everywhere". Restore. Then delete the server total's floor (`if l.st.Tokens < 0 { l.st.Tokens = 0 }`); expected: FAIL "the server total floors at nothing". Restore; PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/apiframework/budget.go internal/apiframework/allowance_test.go modules/aicompanion/models.go
git commit -F - <<'EOF'
feat(apiframework): settle every counter a hold touched

A relayed count is clamped to its hold, a server-key count keeps its
overage, every counter floors at nothing, and a hold from an earlier day
refunds no allowance.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

## Task 4: Consumer shares of the server budget

**Files:**
- Modify: `internal/apiframework/settings.go:68-83` (`ServerSettings` and `HasKey`, at `09964d50f`)
- Modify: `internal/apiframework/budget.go` (`Books.Reserve`, `ledger.reserve`)
- Modify: `internal/apiframework/allowance_test.go`

- [ ] **Step 1: Write the failing test**

Append to `allowance_test.go`:
```go
// A consumer holds at most its share of the day's budget; with no global
// cap there is no share cap; a player's own key is never held to one.
func TestAConsumerIsHeldToItsShare(t *testing.T) {
	ResetBudgetForTest(``)
	restore := SetServerForTest(ServerSettings{Endpoint: Endpoint{BaseURL: DefaultBaseURL},
		DailyTokenBudget: 1000, BaublesSharePercent: 25, BreakerErrors: 2, BreakerSeconds: 60})
	t.Cleanup(func() { restore(); ResetBudgetForTest(``) })

	h, err := Reserve(ConsumerBaubles, 200, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Reserve(ConsumerBaubles, 100, true); !errors.Is(err, ErrOverShare) || RefusedBy(err) != RefusedShare {
		t.Fatalf("over a 250-token share, and it says so: %v (%q)", err, RefusedBy(err))
	}
	if _, err := Reserve(ConsumerCompanion, 700, true); err != nil {
		t.Fatal("a consumer with no share cap spends the rest")
	}
	Settle(h, 50, false)
	if _, err := Reserve(ConsumerBaubles, 200, true); err != nil {
		t.Fatal("settling frees share")
	}
	if _, err := Reserve(ConsumerBaubles, 5000, false); err != nil {
		t.Fatal("a player's own key is held to no share")
	}

	ResetBudgetForTest(``)
	noCap := SetServerForTest(ServerSettings{Endpoint: Endpoint{BaseURL: DefaultBaseURL},
		DailyTokenBudget: 0, BaublesSharePercent: 25, BreakerErrors: 2, BreakerSeconds: 60})
	defer noCap()
	if _, err := Reserve(ConsumerBaubles, 100000, true); err != nil {
		t.Fatal("no global cap is no share cap")
	}
}

func TestSharePercentByConsumer(t *testing.T) {
	s := ServerSettings{CompanionSharePercent: 60, BaublesSharePercent: 25}
	if s.SharePercent(ConsumerCompanion) != 60 || s.SharePercent(ConsumerBaubles) != 25 || s.SharePercent(`other`) != 0 {
		t.Fatal("each consumer's own share; an unknown one has none")
	}
}
```

- [ ] **Step 2: Run to confirm it fails to compile**

Run: `go test ./internal/apiframework/ -run 'TestAConsumerIsHeldToItsShare|TestSharePercentByConsumer' 2>&1 | head`
Expected: build failure on `BaublesSharePercent` and `SharePercent`.

- [ ] **Step 3: Implement**

In `settings.go`, inside `ServerSettings` after `BreakerSeconds int` add:
```go
	// CompanionSharePercent and BaublesSharePercent cap what each feature
	// may hold of DailyTokenBudget in a day, as a percentage (1 to 99). 0
	// is no share cap, and so is no DailyTokenBudget.
	CompanionSharePercent int
	BaublesSharePercent   int
```
After `HasKey` add:
```go
// SharePercent is consumer's share of the day's budget (0: no share cap).
func (s ServerSettings) SharePercent(consumer string) int {
	switch consumer {
	case ConsumerCompanion:
		return s.CompanionSharePercent
	case ConsumerBaubles:
		return s.BaublesSharePercent
	}
	return 0
}
```
In `budget.go` `Books.Reserve`, replace `k.l.reserve(consumer, tokens, s.DailyTokenBudget, 0, spendServer, charges)` with `k.l.reserve(consumer, tokens, s.DailyTokenBudget, s.SharePercent(consumer), spendServer, charges)`.
In `ledger.reserve`, directly after the `ErrOverBudget` check add:
```go
	if spendServer && limit > 0 && sharePct > 0 && sharePct < 100 &&
		l.st.ByConsumer[consumer]+tokens > limit*sharePct/100 {
		return Hold{}, refused(RefusedShare, ErrOverShare)
	}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/apiframework/ ./modules/aicompanion/ ./modules/baubles/ 2>&1 | tail -5`
Expected: three `ok` (every existing test sets no share, which is no cap).

- [ ] **Step 5: Prove it can fail**

Change `> limit*sharePct/100` to `> limit` temporarily; run `-run TestAConsumerIsHeldToItsShare`; expected FAIL "over a 250-token share, and it says so". Restore. Change `refused(RefusedShare, ErrOverShare)` to `refused(RefusedGlobal, ErrOverShare)`; same run, same FAIL (the counter is named wrong). Restore; PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/apiframework/settings.go internal/apiframework/budget.go internal/apiframework/allowance_test.go
git commit -F - <<'EOF'
feat(apiframework): each consumer holds at most its share of the day

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

## Task 5: by_user persistence, SeedAllowances, Allowances, SaveBudget deep copy

Owner ruling 12 (correction 9): the companion keeps its own copy of its three allowance maps as a backup, and a quarantined `budget.yaml` re-seeds from it. The ledger's half is here: one seed mark per dimension per day, saved with the day, so a normal restart skips the seed and a quarantine (which loses the marks with the counts) lets the next boot seed again; and `Allowances(dim)`, the read the companion writes its backup from.

**Files:**
- Modify: `internal/apiframework/budget.go` (`SeedAllowances`, `Allowances`, `SaveBudget`, imports)
- Modify: `internal/apiframework/allowance_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `allowance_test.go` (imports gain `"fmt"`, `"os"`, `"path/filepath"`, `"reflect"`, `"strings"`, `"sync"`):
```go
// R38, R39: allowances and seed marks are living state with the rest of
// the day. A dimension is seeded once a day, today only; a same-day restart
// finds the mark and seeds nothing; a quarantine loses the counts and the
// marks together, so the next seed applies again.
func TestAllowancesSaveLoadAndSeedOnce(t *testing.T) {
	dir := t.TempDir()
	ResetBudgetForTest(dir)
	t.Cleanup(func() { ResetBudgetForTest(``) })
	h, err := Reserve(ConsumerBaubles, 500, true, Charge{Dim: DimBaublesFinder, UserId: 7, Limit: 20000})
	if err != nil {
		t.Fatal(err)
	}
	Settle(h, 123, false)
	day := Today().Day
	backup := map[int]int{5: 777, 8: 0}

	SeedAllowances(DimCompanionOwner, `1999-01-01`, map[int]int{6: 50})
	if Allowance(DimCompanionOwner, 6) != 0 {
		t.Fatal("a stale day seeds nothing, and leaves no mark")
	}
	SeedAllowances(DimCompanionOwner, day, backup)
	SeedAllowances(DimCompanionOwner, day, backup)
	if Allowance(DimCompanionOwner, 5) != 777 || Allowance(DimCompanionOwner, 8) != 0 {
		t.Fatalf("seeded once a day per dimension: %d", Allowance(DimCompanionOwner, 5))
	}
	if got := Allowances(DimCompanionOwner); !reflect.DeepEqual(got, map[int]int{5: 777}) {
		t.Fatalf("one dimension's spends, nothing spent left out: %v", got)
	}
	if got := Allowances(DimCompanionStranger); len(got) != 0 {
		t.Fatalf("a dimension is its own: %v", got)
	}
	SaveBudget()

	ResetBudgetForTest(dir) // a same-day restart
	if Allowance(DimBaublesFinder, 7) != 123 || Allowance(DimCompanionOwner, 5) != 777 {
		t.Fatalf("a restart keeps the day's allowances: finder=%d owner=%d", Allowance(DimBaublesFinder, 7), Allowance(DimCompanionOwner, 5))
	}
	SeedAllowances(DimCompanionOwner, day, backup)
	if Allowance(DimCompanionOwner, 5) != 777 {
		t.Fatalf("the mark is saved too: a same-day restart seeds nothing: %d", Allowance(DimCompanionOwner, 5))
	}
	raw, err := os.ReadFile(filepath.Join(dir, `budget.yaml`))
	if err != nil || !strings.Contains(string(raw), `by_user:`) || !strings.Contains(string(raw), `baubles.finder:7`) ||
		!strings.Contains(string(raw), `seeded:`) {
		t.Fatalf("by_user and seeded in budget.yaml: %s", raw)
	}

	if err := os.WriteFile(filepath.Join(dir, `budget.yaml`), []byte("day: [unclosed"), 0644); err != nil {
		t.Fatal(err)
	}
	ResetBudgetForTest(dir) // a boot that finds the file corrupt
	if Allowance(DimBaublesFinder, 7) != 0 || Allowance(DimCompanionOwner, 5) != 0 {
		t.Fatal("a quarantine restarts the day's allowances with its totals")
	}
	SeedAllowances(DimCompanionOwner, day, backup)
	if Allowance(DimCompanionOwner, 5) != 777 {
		t.Fatalf("the marks went with the counts, so the backup seeds again: %d", Allowance(DimCompanionOwner, 5))
	}
}

// R38: SaveBudget marshals copies, never the live maps. Only -race can see
// the shared map; run it in CI or the Docker test image (see the gate). The
// key space is bounded (ten users and ten seed dimensions per goroutine) and
// the loop capped, so the maps stay small and it runs in seconds under -race.
func TestSaveBudgetCopiesEveryMapUnderTheLock(t *testing.T) {
	dir := t.TempDir()
	ResetBudgetForTest(dir)
	restore := SetServerForTest(ServerSettings{Endpoint: Endpoint{BaseURL: DefaultBaseURL}, BreakerErrors: 2, BreakerSeconds: 60})
	t.Cleanup(func() { restore(); ResetBudgetForTest(``) })
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 20000; i++ {
				// The work comes first, so goroutine 0 writes key 0 (read
				// back below) before it can see stop.
				c := Charge{Dim: DimBaublesFinder, UserId: g*10 + i%10}
				if h, err := Reserve(ConsumerBaubles, 1, true, c); err == nil {
					Settle(h, 1, false)
				}
				SeedAllowances(fmt.Sprintf(`test.seed%d`, g*10+i%10), Today().Day, map[int]int{1: 1})
				select {
				case <-stop:
					return
				default:
				}
			}
		}(g)
	}
	for i := 0; i < 200; i++ {
		SaveBudget()
	}
	close(stop)
	wg.Wait()
	SaveBudget()
	want := Allowance(DimBaublesFinder, 0)
	ResetBudgetForTest(dir)
	if want < 1 || Allowance(DimBaublesFinder, 0) != want {
		t.Fatalf("what was saved reads back: saved %d, read %d", want, Allowance(DimBaublesFinder, 0))
	}
}
```

- [ ] **Step 2: Run to confirm they fail to compile**

Run: `go test ./internal/apiframework/ -run 'TestAllowancesSaveLoadAndSeedOnce|TestSaveBudgetCopiesEveryMapUnderTheLock' 2>&1 | head`
Expected: build failure on `SeedAllowances` and `Allowances`.

- [ ] **Step 3: Implement**

Add `"strings"` to `budget.go`'s imports. After `Books.SeedTokens` add:
```go
// SeedAllowances hands the ledger one dimension's per-user spends from
// today, kept by a feature before the ledger kept them or as its own backup
// (the AI companion's saved day), so neither the move to the ledger nor a
// quarantined budget.yaml hands out a second allowance. It applies once per
// dimension per day: the mark is saved with the day, so a normal restart
// seeds nothing, and a quarantine, which loses the marks with the counts,
// lets the next boot seed again. A stale day seeds nothing.
func SeedAllowances(dim string, day string, spent map[int]int) {
	shared.SeedAllowances(dim, day, spent)
}

// SeedAllowances on these books.
func (k *Books) SeedAllowances(dim string, day string, spent map[int]int) {
	k.l.mu.Lock()
	defer k.l.mu.Unlock()
	k.l.loadLocked()
	k.l.rollLocked()
	if day != k.l.st.Day || k.l.st.Seeded[dim] {
		return
	}
	for userId, tokens := range spent {
		if tokens > 0 {
			k.l.st.ByUser[allowanceKey(dim, userId)] += tokens
		}
	}
	k.l.st.Seeded[dim] = true
	k.l.dirty = true
}

// Allowances is every user's spend today in one dimension, by user id: a
// copy, with nothing-spent users left out. A feature that keeps its own
// backup of its allowances (the AI companion) writes it from this.
func Allowances(dim string) map[int]int {
	return shared.Allowances(dim)
}

// Allowances on these books.
func (k *Books) Allowances(dim string) map[int]int {
	k.l.mu.Lock()
	defer k.l.mu.Unlock()
	k.l.loadLocked()
	k.l.rollLocked()
	prefix := dim + `:`
	out := map[int]int{}
	for key, tokens := range k.l.st.ByUser {
		if tokens <= 0 || !strings.HasPrefix(key, prefix) {
			continue
		}
		if userId, err := strconv.Atoi(key[len(prefix):]); err == nil {
			out[userId] = tokens
		}
	}
	return out
}
```
(`companion.stranger:` is not a prefix of `companion.strangersfor:5`: the colon ends each dimension's name.)
In `SaveBudget`, replace `st.ByConsumer, st.CallsBy = byC, callsBy` with:
```go
	byUser := make(map[string]int, len(st.ByUser))
	for k, v := range st.ByUser {
		byUser[k] = v
	}
	seeded := make(map[string]bool, len(st.Seeded))
	for k, v := range st.Seeded {
		seeded[k] = v
	}
	st.ByConsumer, st.CallsBy, st.ByUser, st.Seeded = byC, callsBy, byUser, seeded
```

- [ ] **Step 4: Run the tests (no race locally)**

Run: `go test ./internal/apiframework/ 2>&1 | tail -3`
Expected: `ok`.

- [ ] **Step 5: Run the race test where CGO exists**

This machine has `CGO_ENABLED=0` and no gcc, so `-race` cannot run here. Run it in the repo's Docker test image (Docker Desktop is installed; this starts no game server):
```bash
cd /c/tmp/dogmud-baubles-s5 && docker compose -f compose.test.yml run --build --rm test go test -race -count=1 -run 'TestSaveBudgetCopiesEveryMapUnderTheLock|TestAllowancesSaveLoadAndSeedOnce' ./internal/apiframework/
```
Expected: `ok`. If Docker is unavailable, record "race: CI only" in the commit body; CI's `go test -timeout 900s -race ./...` runs it on the PR.

- [ ] **Step 6: Prove the race test can fail (Docker only)**

Temporarily change the final assignment back to `st.ByConsumer, st.CallsBy = byC, callsBy` (no `ByUser`/`Seeded` copy). Re-run Step 5's command. Expected: `WARNING: DATA RACE` naming `SaveBudget` and `reserve`. Restore; re-run; `ok`.

- [ ] **Step 7: Commit**

```bash
git add internal/apiframework/budget.go internal/apiframework/allowance_test.go
git commit -F - <<'EOF'
feat(apiframework): allowances persist in budget.yaml, seeded once a day

by_user and the per-dimension seed marks are copied under the lock in
SaveBudget. A quarantine loses the marks with the counts, so a feature's
own backup (Allowances) seeds the day again. The race test needs CGO: run
in the Docker test image or CI.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

## Task 6: The companion's charges and test helpers

**Files:**
- Modify: `modules/aicompanion/tiers.go` (after `applyRoute`, line 239)
- Modify: `modules/aicompanion/frameworktest_helpers_test.go`
- Create: `modules/aicompanion/allowance_test.go`

- [ ] **Step 1: Write the failing test**

Create `modules/aicompanion/allowance_test.go`:
```go
package aicompanion

import (
	"reflect"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/apiframework"
)

// R5, R6: who pays, one to one with the old rules. A passer-by pays from
// their own allowance and from what passers-by together may spend of this
// owner's companion (only when there is an owner), never from the owner's
// allowance; everything else is the owner's.
func TestAllowanceChargesMapOneToOne(t *testing.T) {
	m := &AICompanionModule{cfg: Config{DailyTokensPerCompanion: 300, StrangerDailyTokens: 50, StrangerTokensPerOwner: 100}}
	owner := func(id int) apiframework.Charge {
		return apiframework.Charge{Dim: apiframework.DimCompanionOwner, UserId: id, Limit: 300}
	}
	stranger := apiframework.Charge{Dim: apiframework.DimCompanionStranger, UserId: 2, Limit: 50}
	perOwner := apiframework.Charge{Dim: apiframework.DimCompanionStrangersFor, UserId: 5, Limit: 100}
	for _, tc := range []struct {
		name         string
		owner, asker int
		want         []apiframework.Charge
	}{
		{`the owner's own call`, 5, 0, []apiframework.Charge{owner(5)}},
		{`a passer-by`, 5, 2, []apiframework.Charge{stranger, perOwner}},
		{`a passer-by, no owner`, 0, 2, []apiframework.Charge{stranger}},
		{`no owner, no asker`, 0, 0, []apiframework.Charge{owner(0)}},
	} {
		if got := m.allowanceCharges(tc.owner, tc.asker); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %+v, want %+v", tc.name, got, tc.want)
		}
	}
}
```

- [ ] **Step 2: Run to confirm it fails**

Run: `go test ./modules/aicompanion/ -run TestAllowanceChargesMapOneToOne 2>&1 | head`
Expected: build failure, `m.allowanceCharges undefined`.

- [ ] **Step 3: Implement**

In `tiers.go`, after `applyRoute`:
```go
// allowanceCharges names the per-user allowances a call counts against
// (apiframework.Charge), each with its limit from the config now. A call a
// passer-by prompted is theirs: their own StrangerDailyTokens, and what
// passers-by together may spend of this owner's companion
// (StrangerTokensPerOwner, when there is an owner); never the owner's
// allowance. Every other call is the owner's (DailyTokensPerCompanion). 0
// is no cap for any of them.
func (m *AICompanionModule) allowanceCharges(ownerId int, askerId int) []apiframework.Charge {
	if askerId > 0 {
		cs := []apiframework.Charge{{Dim: apiframework.DimCompanionStranger, UserId: askerId, Limit: m.cfg.StrangerDailyTokens}}
		if ownerId > 0 {
			cs = append(cs, apiframework.Charge{Dim: apiframework.DimCompanionStrangersFor, UserId: ownerId, Limit: m.cfg.StrangerTokensPerOwner})
		}
		return cs
	}
	return []apiframework.Charge{{Dim: apiframework.DimCompanionOwner, UserId: ownerId, Limit: m.cfg.DailyTokensPerCompanion}}
}
```
Append to `frameworktest_helpers_test.go`:
```go
// The day's per-user allowances, as the ledger keeps them.
func ownerSpent(m *AICompanionModule, ownerId int) int {
	return m.fw().Allowance(apiframework.DimCompanionOwner, ownerId)
}
func strangerSpent(m *AICompanionModule, askerId int) int {
	return m.fw().Allowance(apiframework.DimCompanionStranger, askerId)
}
func strangersForSpent(m *AICompanionModule, ownerId int) int {
	return m.fw().Allowance(apiframework.DimCompanionStrangersFor, ownerId)
}
func setOwnerSpent(m *AICompanionModule, ownerId int, tokens int) {
	m.fw().SetAllowanceForTest(apiframework.DimCompanionOwner, ownerId, tokens)
}
func setStrangerSpent(m *AICompanionModule, askerId int, tokens int) {
	m.fw().SetAllowanceForTest(apiframework.DimCompanionStranger, askerId, tokens)
}
func setStrangersForSpent(m *AICompanionModule, ownerId int, tokens int) {
	m.fw().SetAllowanceForTest(apiframework.DimCompanionStrangersFor, ownerId, tokens)
}
```

- [ ] **Step 4: Run**

Run: `go test ./modules/aicompanion/ -run TestAllowanceChargesMapOneToOne -v 2>&1 | tail -3 && go vet ./modules/aicompanion/`
Expected: PASS, vet silent.

- [ ] **Step 5: Commit**

```bash
git add modules/aicompanion/tiers.go modules/aicompanion/frameworktest_helpers_test.go modules/aicompanion/allowance_test.go
git commit -F - <<'EOF'
feat(aicompanion): name each call's allowances as ledger charges

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

## Task 7: The companion reserves, settles and checks on the ledger

One commit: the writes and the reads move together, with the tests that pin them, so no commit has a check reading a counter nothing writes.

**Files:**
- Modify: `modules/aicompanion/tiers.go:241-304`
- Modify: `modules/aicompanion/models.go:205-265, 537-636`
- Modify: `modules/aicompanion/listeners.go:561-589` (and imports)
- Modify: `modules/aicompanion/commands.go:157`, `runtime.go:630, 1546-1565`
- Modify: `modules/aicompanion/aicompanion_test.go`, `tiers_test.go`, `money_test.go`, `relayfor_test.go`, `allowance_test.go`

- [ ] **Step 1: Rewrite `reserveRoute`, `hold`, `settleRoute` in `tiers.go` (lines 241-304)**

```go
// reserveRoute holds a call's worst case against whoever pays for it, in
// one check-and-hold step on the ledger (apiframework Reserve: all or
// nothing, under its own lock). The server's key is held against the
// server's budget and the payer's allowances (allowanceCharges). A player's
// own key spends nothing of the server's, so it is held against nothing,
// except that a passer-by's question is still held against their
// StrangerDailyTokens and the owner's StrangerTokensPerOwner: the owner's
// key is not theirs to spend without end.
//
// The hold it returns is what settleRoute takes back.
func (m *AICompanionModule) reserveRoute(r route, ownerId int, askerId int, tokens int) (hold, bool) {
	h := hold{r: r}
	spendServer := false
	switch r.kind {
	case routeServer:
		spendServer = true
	case routeRelay:
		if askerId <= 0 {
			return h, true // the owner's own key, for the owner: nothing to hold
		}
	default:
		return h, false
	}
	fh, err := m.fw().Reserve(apiframework.ConsumerCompanion, tokens, spendServer, m.allowanceCharges(ownerId, askerId)...)
	if err != nil {
		h.refusal = err
		return h, false
	}
	h.fw = fh
	return h, true
}

// hold is one call's reservation, as reserveRoute made it: the route, the
// ledger's own hold (empty when nothing was held), and, when reserveRoute
// said no, the ledger's refusal (apiframework.RefusedBy names its counter;
// nil when there was no route at all).
type hold struct {
	r       route
	fw      apiframework.Hold
	refusal error
}

// settleRoute settles a reservation made by reserveRoute, exactly once. The
// ledger applies every rule: a count relayed through the owner's browser
// is held between nothing and the reservation, no counter goes below
// nothing, and a hold from an earlier day gives nothing back to today's.
func (m *AICompanionModule) settleRoute(h hold, used int) {
	if h.fw.Consumer == `` {
		return
	}
	m.fw().Settle(h.fw, used, false)
}
```

- [ ] **Step 2: Delete the old helpers in `models.go`**

Delete `chargeOwner`, `chargeStranger`, `strangerFits`, `chargeStrangerFor` (lines 215-265) and `tryReserveTokens`, `tryReserveFor`, `reserveFor`, `settleTokens`, `settleFor`, `settleForDay`, `settleHeld` with their comments (lines 537-636). Replace `ownerBudgetLeft` (lines 205-213) with:
```go
// ownerBudgetLeft reports whether one companion has any daily tokens left
// at all (the ledger's DimCompanionOwner). Admission of a particular call
// goes through reserveRoute, which weighs that call's worst case.
func (m *AICompanionModule) ownerBudgetLeft(ownerId int) bool {
	if m.cfg.DailyTokensPerCompanion <= 0 {
		return true
	}
	return m.fw().Allowance(apiframework.DimCompanionOwner, ownerId) < m.cfg.DailyTokensPerCompanion
}
```

- [ ] **Step 3: Read checks and display**

In `listeners.go`, add `"github.com/GoMudEngine/GoMud/internal/apiframework"` to the imports and replace the two allowance blocks in `strangerMayAsk` (lines 568-581) with:
```go
	if m.cfg.StrangerDailyTokens > 0 && !m.strangersOff(c.ownerUserId) &&
		m.fw().Allowance(apiframework.DimCompanionStranger, u.UserId) >= m.cfg.StrangerDailyTokens {
		return false
	}
	// Nor when passers-by together have spent all they may of this owner's
	// companion today (StrangerTokensPerOwner).
	if m.cfg.StrangerTokensPerOwner > 0 && !m.strangersOff(c.ownerUserId) &&
		m.fw().Allowance(apiframework.DimCompanionStrangersFor, c.ownerUserId) >= m.cfg.StrangerTokensPerOwner {
		return false
	}
```
In `commands.go:157` replace `m.ownerTokens[c.ownerUserId]` with `m.fw().Allowance(apiframework.DimCompanionOwner, c.ownerUserId)`.
In `runtime.go`, the dispatch refusal (line 630) `m.logBudgetRefusal(ownerId, asker, reserved)` becomes `m.logBudgetRefusal(ownerId, asker, reserved, held.refusal)`, and `logBudgetRefusal` (lines 1546-1565) becomes:
```go
// logBudgetRefusal notes a decision the budgets would not pay for, at most
// once a minute per server, so a spent allowance is visible in the log
// rather than silently turning a companion into a set of stock phrases.
// refusedBy names the counter that said no (apiframework.RefusedBy: the
// day's budget "global", the companion's "share", or one allowance's
// dimension), so a spent share is not mistaken for a spent allowance.
func (m *AICompanionModule) logBudgetRefusal(ownerId int, askerId int, wanted int, why error) {
	now := time.Now()
	if now.Sub(m.lastBudgetLog) < time.Minute {
		return
	}
	m.lastBudgetLog = now
	server := m.fw().Today() // the one budget every feature shares
	refusedBy := apiframework.RefusedBy(why)
	if askerId > 0 {
		mudlog.Warn(`aicompanion`, `action`, `budgetRefused`, `refusedBy`, refusedBy, `owner`, ownerId, `asker`, askerId, `wanted`, wanted,
			`askerSpentToday`, m.fw().Allowance(apiframework.DimCompanionStranger, askerId), `askerCap`, m.cfg.StrangerDailyTokens,
			`serverSpentToday`, server.Tokens, `serverCap`, server.Limit)
		return
	}
	mudlog.Warn(`aicompanion`, `action`, `budgetRefused`, `refusedBy`, refusedBy, `owner`, ownerId, `wanted`, wanted,
		`ownerSpentToday`, m.fw().Allowance(apiframework.DimCompanionOwner, ownerId), `ownerCap`, m.cfg.DailyTokensPerCompanion,
		`serverSpentToday`, server.Tokens, `serverCap`, server.Limit)
}
```

- [ ] **Step 4: Let the compiler list the test sites**

Run: `go vet ./modules/aicompanion/ 2>&1 | head -60`
Expected: errors at the T2 lines (`tryReserveTokens`, `tryReserveFor`, `settleTokens`, `settleFor`, `chargeOwner`, `chargeStrangerFor` undefined; `hold` has no field `owner`/`tokens`/`day`). That list is the work list; fix every one in Steps 5 and 6.

- [ ] **Step 5: Rewrite the test helpers in `tiers_test.go` (lines 860-869)**

```go
// tryRoute is reserveRoute for a test that settles the same day.
func tryRoute(m *AICompanionModule, r route, ownerId int, askerId int, tokens int) bool {
	_, ok := m.reserveRoute(r, ownerId, askerId, tokens)
	return ok
}

// settleToday settles a reservation tryRoute made today: the hold the
// ledger would have given, rebuilt from the same route and payer.
func settleToday(m *AICompanionModule, r route, ownerId int, askerId int, reserved int, used int) {
	h := hold{r: r}
	if r.kind == routeServer || (r.kind == routeRelay && askerId > 0) {
		h.fw = apiframework.Hold{Consumer: apiframework.ConsumerCompanion, Tokens: reserved, Day: m.fw().Day(),
			SpendServer: r.kind == routeServer, Charges: m.allowanceCharges(ownerId, askerId)}
	}
	m.settleRoute(h, used)
}
```

- [ ] **Step 6: Rewrite the tests whose shape changes**

`aicompanion_test.go`, `TestBreakerAndBudgets` lines 959-966 become:
```go
	setOwnerSpent(m, 7, 90)
	if !m.ownerBudgetLeft(7) {
		t.Fatal("budget left")
	}
	setOwnerSpent(m, 7, 110)
	if m.ownerBudgetLeft(7) || !m.ownerBudgetLeft(8) {
		t.Fatal("per-companion budget")
	}
```
`TestTokenReservationSettles` lines 1246-1269 become:
```go
func TestTokenReservationSettles(t *testing.T) {
	freshServer(t, 5000, 5, 60)
	m := &AICompanionModule{cfg: Config{DailyTokensPerCompanion: 1000}}
	server := route{kind: routeServer}
	if !tryRoute(m, server, 3, 0, 900) {
		t.Fatal("a call that fits must be admitted")
	}
	if tryRoute(m, server, 3, 0, 900) {
		t.Fatal("a second call that would overshoot the budget must be refused, not merely counted")
	}
	if tryRoute(m, server, 4, 0, 900) != true {
		t.Fatal("another companion has its own budget")
	}
	settleToday(m, server, 3, 0, 900, 120)
	if !tryRoute(m, server, 3, 0, 800) {
		t.Fatal("settling a call frees what it did not use")
	}
	settleToday(m, server, 3, 0, 800, 0)
	if ownerSpent(m, 3) != 120 {
		t.Fatalf("owner tokens after settlement: %d", ownerSpent(m, 3))
	}
	// Both counters set below a hold, so its refund would take them
	// negative (a -500 settlement of a 0 hold is floored to 0 first and
	// could never test this): each floors at nothing.
	setOwnerSpent(m, 3, 100)
	m.fw().SetSpentForTest(100, 900)
	settleToday(m, server, 3, 0, 900, 0)
	if serverSpent(m) != 0 || ownerSpent(m, 3) != 0 {
		t.Fatalf("counters floor at nothing: server=%d owner=%d", serverSpent(m), ownerSpent(m, 3))
	}
```
(the rest of the function, from the `// The whole worst case is held` comment on, is unchanged).

`TestBudgetCountsTheWholeRequest`, lines 1778-1791 become:
```go
	// A reservation outstanding over the day boundary is not credited back
	// against the new day.
	freshServer(t, 1000, 5, 60)
	m := &AICompanionModule{cfg: Config{}}
	h, ok := m.reserveRoute(route{kind: routeServer}, 1, 0, 600)
	if !ok {
		t.Fatal("fixture: the hold fits")
	}
	tomorrow := time.Now().UTC().Add(24 * time.Hour)
	m.fw().SetClockForTest(func() time.Time { return tomorrow })
	if serverSpent(m) != 600 {
		t.Fatalf("outstanding reservations carry over the rollover, got %d", serverSpent(m))
	}
	m.settleRoute(h, 100)
	if serverSpent(m) != 100 || serverHeld(m) != 0 {
		t.Fatalf("settlement after a rollover: today=%d outstanding=%d", serverSpent(m), serverHeld(m))
	}
}
```
`TestStrangerCallsAreReservedAgainstTheStranger` (2396-2437) becomes:
```go
func TestStrangerCallsAreReservedAgainstTheStranger(t *testing.T) {
	freshServer(t, 5000, 5, 60)
	m := &AICompanionModule{cfg: Config{DailyTokensPerCompanion: 1000, StrangerDailyTokens: 1000}}
	server := route{kind: routeServer}

	if !tryRoute(m, server, 1, 2, 900) {
		t.Fatal("a passer-by's question that fits their allowance is admitted")
	}
	if ownerSpent(m, 1) != 0 || strangerSpent(m, 2) != 900 {
		t.Fatalf("it is held against the passer-by, not her owner: owner=%d stranger=%d", ownerSpent(m, 1), strangerSpent(m, 2))
	}
	if tryRoute(m, server, 1, 2, 900) {
		t.Fatal("a second question that would overshoot their allowance is refused while the first is held")
	}
	if !tryRoute(m, server, 1, 0, 900) {
		t.Fatal("her owner's own allowance is untouched by a stranger's questions")
	}
	if !tryRoute(m, server, 1, 3, 900) {
		t.Fatal("another passer-by has an allowance of their own")
	}
	if serverSpent(m) != 2700 || serverHeld(m) != 2700 {
		t.Fatalf("the server's budget holds all three: today=%d outstanding=%d", serverSpent(m), serverHeld(m))
	}

	// Settled against the same payer: what was not used goes back to them.
	settleToday(m, server, 1, 2, 900, 100)
	if strangerSpent(m, 2) != 100 || ownerSpent(m, 1) != 900 {
		t.Fatalf("settlement: stranger=%d owner=%d", strangerSpent(m, 2), ownerSpent(m, 1))
	}
	// A call that failed refunds all of it, to the passer-by.
	settleToday(m, server, 1, 3, 900, 0)
	if strangerSpent(m, 3) != 0 || ownerSpent(m, 1) != 900 {
		t.Fatalf("refund: stranger=%d owner=%d", strangerSpent(m, 3), ownerSpent(m, 1))
	}
	settleToday(m, server, 1, 0, 900, 900)
	if serverSpent(m) != 1000 || serverHeld(m) != 0 {
		t.Fatalf("after settling everything: today=%d outstanding=%d", serverSpent(m), serverHeld(m))
	}
	// Set below a hold, so the refund would take it negative: it floors.
	setStrangerSpent(m, 2, 50)
	setStrangersForSpent(m, 1, 50)
	settleToday(m, server, 1, 2, 900, 0)
	if strangerSpent(m, 2) != 0 || strangersForSpent(m, 1) != 0 {
		t.Fatalf("a stranger's counts floor at nothing: %d, %d", strangerSpent(m, 2), strangersForSpent(m, 1))
	}
}
```
`TestStrangerReservationsCannotSlipPastTheCapTogether` line 2448: `m.tryReserveFor(1, 2, 400)` becomes `tryRoute(m, route{kind: routeServer}, 1, 2, 400)`; lines 2458-2459 per T1.

`TestStrangerDailyCapStopsTheirPrompts`: delete line 2375 (`m.rollDay()`); lines 2376 and 2389 per T1.
`TestStrangerTalkSummaryIsTheStrangersToPayFor`: delete lines 2595 and 2610 (`m.rollDay()`); every other line per T1.
`TestGoneMindStillRefundsItsOwner`: lines 2652-2653 per T1.

`money_test.go`, `TestAHoldAcrossMidnightRefundsNothing` (881-918) becomes:
```go
func TestAHoldAcrossMidnightRefundsNothing(t *testing.T) {
	m := relayModule(t)
	m.cfg.DailyTokensPerCompanion, m.cfg.StrangerDailyTokens, m.cfg.StrangerTokensPerOwner = 100000, 100000, 100000
	server, relay := route{kind: routeServer}, route{kind: routeRelay, model: `player-model`}
	today := time.Now().UTC()
	m.fw().SetClockForTest(func() time.Time { return today })

	ownerHold, ok1 := m.reserveRoute(server, 5, 0, 900)
	strangerHold, ok2 := m.reserveRoute(relay, 5, 2, 400)
	if !ok1 || !ok2 {
		t.Fatal("fixture: both holds fit")
	}
	// They were held yesterday, and the day turns.
	m.fw().SetClockForTest(func() time.Time { return today.Add(24 * time.Hour) })
	// Today's own spending.
	setOwnerSpent(m, 5, 500)
	setStrangerSpent(m, 2, 300)
	setStrangersForSpent(m, 5, 300)

	m.settleRoute(ownerHold, 100)
	m.settleRoute(strangerHold, 0)
	if ownerSpent(m, 5) != 500 {
		t.Fatalf("the owner's count today is untouched by yesterday's hold: %d", ownerSpent(m, 5))
	}
	if strangerSpent(m, 2) != 300 || strangersForSpent(m, 5) != 300 {
		t.Fatalf("so is the passer-by's: %d, %d", strangerSpent(m, 2), strangersForSpent(m, 5))
	}
	if serverHeld(m) != 0 {
		t.Fatalf("and nothing is left held: %d", serverHeld(m))
	}

	// Control: the same holds settled on their own day do give back.
	h, _ := m.reserveRoute(server, 5, 0, 900)
	m.settleRoute(h, 100)
	if ownerSpent(m, 5) != 600 {
		t.Fatalf("a same-day hold settles at what was used: %d", ownerSpent(m, 5))
	}
}
```
`TestStrangerTokensPerOwnerCapsThemTogether`: delete line 521 (`m.rollDay()`); lines 509-510 and 522 per T1.
`TestRelayUsageIsNeverTrusted` 313-321 and `TestPanickedBackgroundCallsSettle` 371 per T1.

`tiers_test.go`: delete lines 186, 232, 315, 369 (`m.rollDay()`); every map line per T1.

`relayfor_test.go`, `TestSettlementReturnsTheLedgersOwnHold` (138-158) becomes:
```go
// Analysis item 8: the server budget is given back the very hold it gave
// out. The ledger's day is the only day, so the hold carries it.
func TestSettlementReturnsTheLedgersOwnHold(t *testing.T) {
	m := &AICompanionModule{cfg: Config{}}
	tomorrow := time.Now().UTC().Add(24 * time.Hour)
	m.fw().SetClockForTest(func() time.Time { return tomorrow })
	h, ok := m.reserveRoute(route{kind: routeServer}, 1, 0, 900)
	if !ok || h.fw.Tokens != 900 || h.fw.Day != tomorrow.Format(`2006-01-02`) || !h.fw.SpendServer {
		t.Fatalf("held on the ledger's day: %+v", h)
	}
	m.settleRoute(h, 100)
	share := 0
	for _, c := range m.fw().Today().ByConsumer {
		if c.Consumer == apiframework.ConsumerCompanion {
			share = c.Tokens
		}
	}
	if share != 100 || m.fw().Today().Tokens != 100 || m.fw().Today().Outstanding != 0 {
		t.Fatalf("settled to what it used on the ledger's day: share=%d total=%d held=%d",
			share, m.fw().Today().Tokens, m.fw().Today().Outstanding)
	}
}
```
Append to `modules/aicompanion/allowance_test.go`:
```go
// R16 (owner ruling 12): an owner-less server-key call is charged to key 0
// and, like any other, settled back to what it used.
func TestAnOwnerlessHoldIsRefunded(t *testing.T) {
	freshServer(t, 5000, 5, 60)
	m := &AICompanionModule{cfg: Config{DailyTokensPerCompanion: 1000}}
	h, ok := m.reserveRoute(route{kind: routeServer}, 0, 0, 900)
	if !ok || ownerSpent(m, 0) != 900 {
		t.Fatalf("fixture: held against key 0: %v %d", ok, ownerSpent(m, 0))
	}
	m.settleRoute(h, 100)
	if ownerSpent(m, 0) != 100 {
		t.Fatalf("key 0 is refunded what it did not use: %d", ownerSpent(m, 0))
	}
}

// R43: a refused reservation keeps the ledger's reason, so the refusal log
// can say which counter said no.
func TestARefusedRouteKeepsTheLedgersReason(t *testing.T) {
	freshServer(t, 5000, 5, 60)
	m := &AICompanionModule{cfg: Config{StrangerDailyTokens: 100}}
	h, ok := m.reserveRoute(route{kind: routeServer}, 1, 2, 500)
	if ok || apiframework.RefusedBy(h.refusal) != apiframework.DimCompanionStranger {
		t.Fatalf("refused by the passer-by's allowance: %v %v", ok, h.refusal)
	}
	h, ok = m.reserveRoute(route{kind: routeServer}, 1, 0, 6000)
	if ok || apiframework.RefusedBy(h.refusal) != apiframework.RefusedGlobal {
		t.Fatalf("refused by the day's budget: %v %v", ok, h.refusal)
	}
	if h, ok := m.reserveRoute(route{kind: routeNone}, 1, 0, 10); ok || h.refusal != nil {
		t.Fatal("no route is no refusal of the ledger's")
	}
}
```

- [ ] **Step 7: Confirm no map is written outside load, save and the day roll**

Run standalone (a zero count exits 1, which is the pass):
```bash
grep -n 'ownerTokens\|strangerTokens\|strangersFor' modules/aicompanion/*.go | grep -v 'aicompanion.go:17[2-4]\|aicompanion.go:4[01][0-9]\|models.go:6[4-9][0-9]\|models.go:70[0-9]'
```
Expected: no output (only the declarations, `rollDay` and `loadBudget`/`saveBudget` remain; line numbers shift after Step 2, so read what prints and accept only those four places).

- [ ] **Step 8: Run the full package and the root gate**

Run: `go vet ./modules/aicompanion/ && go test ./modules/aicompanion/ ./internal/apiframework/ ./modules/baubles/ 2>&1 | tail -5 && go test . 2>&1 | tail -3`
Expected: all `ok`.

- [ ] **Step 9: Prove two rewritten tests can fail**

(a) In `allowanceCharges`, temporarily append an owner charge to the passer-by branch (`cs = append(cs, apiframework.Charge{Dim: apiframework.DimCompanionOwner, UserId: ownerId, Limit: m.cfg.DailyTokensPerCompanion})`). Run `go test ./modules/aicompanion/ -run 'TestStrangerCallsAreReservedAgainstTheStranger|TestAllowanceChargesMapOneToOne'`. Expected: FAIL "it is held against the passer-by, not her owner". Restore.
(b) In `strangerMayAsk`, change `>= m.cfg.StrangerDailyTokens` to `> m.cfg.StrangerDailyTokens`. Run `-run TestStrangerDailyCapStopsTheirPrompts`. Expected: FAIL "a passer-by with nothing left today prompts nothing". Restore.
(c) In `reserveRoute`, delete `h.refusal = err`. Run `-run TestARefusedRouteKeepsTheLedgersReason`. Expected: FAIL "refused by the passer-by's allowance". Restore; all three PASS.

- [ ] **Step 10: Commit**

```bash
git add modules/aicompanion/tiers.go modules/aicompanion/models.go modules/aicompanion/listeners.go modules/aicompanion/commands.go modules/aicompanion/runtime.go modules/aicompanion/aicompanion_test.go modules/aicompanion/tiers_test.go modules/aicompanion/money_test.go modules/aicompanion/relayfor_test.go modules/aicompanion/allowance_test.go
git commit -F - <<'EOF'
refactor(aicompanion): allowances reserved, settled and read on the ledger

reserveRoute and settleRoute call apiframework Reserve and Settle with the
call's charges; ownerBudgetLeft, strangerMayAsk, the status line and the
refusal log read Allowance, and the refusal log names the counter that
refused. Eleven module helpers deleted; every test that pinned them
asserts the same rule through the ledger. An owner-less hold is refunded
like any other (owner ruling 12).

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

## Task 8: One clock: the companion's counters roll on the ledger's day

**Files:**
- Modify: `modules/aicompanion/aicompanion.go:158-163, 400-414, 445`
- Modify: `autonomy.go:181`, `commands.go:93`, `conversation.go:306, 348`, `corememory.go:168, 206`, `reflect.go:237, 276`, `runtime.go:650, 725`, `models.go` (`loadBudget`, `saveBudget`)
- Modify: `modules/aicompanion/allowance_test.go`

- [ ] **Step 1: Write the failing test**

Append to `modules/aicompanion/allowance_test.go` (imports gain `"time"`):
```go
// R32, R33: the companion's own counts (calls, errors, "you notice"
// moments) roll when the ledger's day turns, on the ledger's clock.
func TestCountersRollOnTheLedgersClock(t *testing.T) {
	m := &AICompanionModule{cfg: Config{NoticeCallsPerDay: 1}}
	m.rollCounters()
	m.callsToday, m.errorsToday = 3, 2
	m.noticesToday[5] = 1
	m.rollCounters()
	if m.callsToday != 3 || m.noticesToday[5] != 1 {
		t.Fatal("the same day keeps its counts")
	}
	tomorrow := time.Now().UTC().Add(24 * time.Hour)
	m.fw().SetClockForTest(func() time.Time { return tomorrow })
	m.rollCounters()
	if m.callsToday != 0 || m.errorsToday != 0 || m.noticesToday[5] != 0 || m.countersDay != tomorrow.Format(`2006-01-02`) {
		t.Fatalf("a new ledger day starts them afresh: calls=%d errors=%d notices=%d day=%s",
			m.callsToday, m.errorsToday, m.noticesToday[5], m.countersDay)
	}
}
```

- [ ] **Step 2: Run to confirm it fails**

Run: `go test ./modules/aicompanion/ -run TestCountersRollOnTheLedgersClock 2>&1 | head`
Expected: build failure, `m.rollCounters undefined`.

- [ ] **Step 3: Replace the day**

In `aicompanion.go` replace the `budgetDay` comment and field (158-161) with:
```go
	// countersDay is the day of the companion's own daily counts (calls,
	// errors, "you notice" moments), on the ledger's clock: the ledger's
	// day is the only day. The allowances and the server's token budget
	// are the ledger's own (apiframework).
	countersDay string
```
Replace `rollDay` (400-414) with:
```go
// rollCounters starts the companion's own daily counts afresh when the
// ledger's day has turned. Allowances roll with the ledger itself.
func (m *AICompanionModule) rollCounters() {
	if day := m.fw().Day(); day != m.countersDay {
		m.countersDay = day
		m.callsToday = 0
		m.errorsToday = 0
		m.noticesToday = map[int]int{}
	}
}

// countCall counts one model call started today.
func (m *AICompanionModule) countCall() {
	m.rollCounters()
	m.callsToday++
}
```
Then:
- `aicompanion.go:445` (in `modelReadyFor`): delete `m.rollDay()` (it guarded only the old maps; `HasRoom` and `ownerBudgetLeft` read the ledger).
- `autonomy.go:181`, `commands.go:93`, `conversation.go:348`, `corememory.go:206`, `reflect.go:276`, `runtime.go:725`: `m.rollDay()` becomes `m.rollCounters()`.
- `conversation.go:306`, `corememory.go:168`, `reflect.go:237`, `runtime.go:650`: `m.callsToday++` becomes `m.countCall()` (reserveRoute no longer rolls the day before these).
- `models.go` `loadBudget`: `m.budgetDay = st.Day` becomes `m.countersDay = st.Day`, and `if st.Day != time.Now().UTC().Format(`2006-01-02`)` becomes `if st.Day != m.fw().Day()`.
- `models.go` `saveBudget`: `Day: m.budgetDay` becomes `Day: m.countersDay`, and `u.Day == m.budgetDay` becomes `u.Day == m.countersDay`; add `m.rollCounters()` as the first line after the `Enabled` check.

- [ ] **Step 4: Let the compiler confirm nothing else read the old day**

Run: `go vet ./modules/aicompanion/ 2>&1 | head -20`
Expected: no output. Then, standalone: `grep -n 'rollDay\|budgetDay' modules/aicompanion/*.go` Expected: no output (exit 1).

- [ ] **Step 5: Run**

Run: `go test ./modules/aicompanion/ 2>&1 | tail -3`
Expected: `ok` (`TestNoticedIsCappedAndSparesTheOwnersKey` passes unchanged).

- [ ] **Step 6: Commit**

```bash
git add modules/aicompanion/aicompanion.go modules/aicompanion/autonomy.go modules/aicompanion/commands.go modules/aicompanion/conversation.go modules/aicompanion/corememory.go modules/aicompanion/reflect.go modules/aicompanion/runtime.go modules/aicompanion/models.go modules/aicompanion/allowance_test.go
git commit -F - <<'EOF'
refactor(aicompanion): one clock, the ledger's; rollDay becomes rollCounters

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

## Task 9: The companion's file is the allowances' backup; boots seed from it once

Owner ruling 12 (correction 9): the companion keeps writing its three maps, now read from the ledger (`Allowances`), and every boot hands them to `SeedAllowances`. The ledger's per-dimension marks decide: the first boot after the move and a boot after a quarantine seed; a normal restart does not.

**Files:**
- Modify: `modules/aicompanion/models.go:638-708` (`budgetState`, `loadBudget`, `saveBudget`)
- Modify: `modules/aicompanion/money_test.go:529-545` (replace `TestStrangersForIsKeptWithTheBudget`)
- Modify: `modules/aicompanion/allowance_test.go` (the quarantine and restart tests)

- [ ] **Step 1: Write the failing tests**

In `money_test.go`, replace `TestStrangersForIsKeptWithTheBudget` (its comment and body, lines 529-545) with:
```go
// R35, R36, R37: the first boot after the move hands the old save's
// allowances to the ledger once; the save still writes them, from the
// ledger, as the backup a quarantined budget.yaml re-seeds from.
func TestFirstBootSeedsTheOldAllowancesOnce(t *testing.T) {
	m := &AICompanionModule{cfg: Config{}}
	day := m.fw().Day()
	old := budgetState{Day: day, Tokens: 900, Calls: 4,
		Owners: map[int]int{5: 1200}, Strangers: map[int]int{2: 300}, StrangersFor: map[int]int{5: 300},
		Notices: map[int]int{5: 2}}
	for boot := 0; boot < 2; boot++ { // a second boot reading the same old file seeds nothing more
		m.restoreBudget(old)
		if ownerSpent(m, 5) != 1200 || strangerSpent(m, 2) != 300 || strangersForSpent(m, 5) != 300 || serverSpent(m) != 900 {
			t.Fatalf("boot %d: owner=%d stranger=%d perOwner=%d server=%d",
				boot, ownerSpent(m, 5), strangerSpent(m, 2), strangersForSpent(m, 5), serverSpent(m))
		}
	}
	if m.callsToday != 4 || m.noticesToday[5] != 2 || m.countersDay != day {
		t.Fatalf("the module's own counts: calls=%d notices=%d day=%s", m.callsToday, m.noticesToday[5], m.countersDay)
	}
	saved := m.budgetStateToSave()
	if saved.Owners[5] != 1200 || saved.Strangers[2] != 300 || saved.StrangersFor[5] != 300 {
		t.Fatalf("the backup is written from the ledger's counts: %+v", saved)
	}
	setOwnerSpent(m, 5, 1500)
	if m.budgetStateToSave().Owners[5] != 1500 {
		t.Fatal("the backup follows the ledger, not the file it was seeded from")
	}
	if saved.Day != day || saved.Calls != 4 || saved.Notices[5] != 2 || saved.Tokens != 900 {
		t.Fatalf("the rest is still written: %+v", saved)
	}

	stale := &AICompanionModule{cfg: Config{}}
	stale.restoreBudget(budgetState{Day: `1999-01-01`, Owners: map[int]int{5: 1}})
	if ownerSpent(stale, 5) != 0 {
		t.Fatal("a stale day seeds nothing")
	}

	// The old file's keys still read (the save before the move wrote them).
	var back budgetState
	if err := yaml.Unmarshal([]byte("day: \"2026-09-25\"\nowners:\n  5: 7\nstrangers:\n  2: 8\nstrangers_for:\n  5: 1234\n"), &back); err != nil {
		t.Fatal(err)
	}
	if back.Owners[5] != 7 || back.Strangers[2] != 8 || back.StrangersFor[5] != 1234 {
		t.Fatalf("old keys read: %+v", back)
	}
}
```
Append to `modules/aicompanion/allowance_test.go` (imports gain `"os"` and `"path/filepath"`). Both tests use the real, disk-backed shared ledger (`apiframework.ResetBudgetForTest(dir)`, and `m.books` pointed at `apiframework.Shared()`), because only a real file can be quarantined:
```go
// Correction 9, owner ruling 12: the companion's own file is the backup of
// its allowances. A quarantined budget.yaml loses the ledger's counts and
// its seed marks together, so the next boot seeds them again from that
// backup: a corrupt file hands nobody a fresh allowance.
func TestAQuarantinedLedgerReseedsFromTheCompanionsBackup(t *testing.T) {
	dir := t.TempDir()
	apiframework.ResetBudgetForTest(dir)
	t.Cleanup(func() { apiframework.ResetBudgetForTest(``) })
	cfg := Config{DailyTokensPerCompanion: 100000, StrangerDailyTokens: 100000, StrangerTokensPerOwner: 100000}
	m := &AICompanionModule{cfg: cfg}
	m.books.Store(apiframework.Shared()) // the real ledger, on disk in dir
	m.restoreBudget(budgetState{Day: m.fw().Day()}) // this boot's seed: it marks all three dimensions
	server := route{kind: routeServer}
	h1, ok1 := m.reserveRoute(server, 5, 0, 900)
	h2, ok2 := m.reserveRoute(server, 5, 2, 900)
	if !ok1 || !ok2 {
		t.Fatal("fixture: both holds fit")
	}
	m.settleRoute(h1, 400)
	m.settleRoute(h2, 300)
	backup := m.budgetStateToSave() // what saveBudget writes to the companion's own file
	if backup.Owners[5] != 400 || backup.Strangers[2] != 300 || backup.StrangersFor[5] != 300 {
		t.Fatalf("the backup is the ledger's counts: %+v", backup)
	}
	apiframework.SaveBudget()

	if err := os.WriteFile(filepath.Join(dir, `budget.yaml`), []byte("day: [unclosed"), 0644); err != nil {
		t.Fatal(err)
	}
	apiframework.ResetBudgetForTest(dir) // the next boot finds budget.yaml corrupt
	booted := &AICompanionModule{cfg: cfg}
	booted.books.Store(apiframework.Shared())
	booted.restoreBudget(backup)
	if ownerSpent(booted, 5) != 400 || strangerSpent(booted, 2) != 300 || strangersForSpent(booted, 5) != 300 {
		t.Fatalf("re-seeded from the backup: owner=%d stranger=%d perOwner=%d",
			ownerSpent(booted, 5), strangerSpent(booted, 2), strangersForSpent(booted, 5))
	}
	if serverSpent(booted) != 700 {
		t.Fatalf("and the companion's share of the day, as SeedTokens always did: %d", serverSpent(booted))
	}
}

// A normal same-day restart finds budget.yaml whole, with its seed marks,
// so the backup is not added a second time.
func TestASameDayRestartDoesNotSeedTwice(t *testing.T) {
	dir := t.TempDir()
	apiframework.ResetBudgetForTest(dir)
	t.Cleanup(func() { apiframework.ResetBudgetForTest(``) })
	cfg := Config{DailyTokensPerCompanion: 100000, StrangerDailyTokens: 100000, StrangerTokensPerOwner: 100000}
	m := &AICompanionModule{cfg: cfg}
	m.books.Store(apiframework.Shared())
	m.restoreBudget(budgetState{Day: m.fw().Day(), // the first boot after the move
		Owners: map[int]int{5: 1200}, Strangers: map[int]int{2: 300}, StrangersFor: map[int]int{5: 300}})
	h, ok := m.reserveRoute(route{kind: routeServer}, 5, 0, 900)
	if !ok {
		t.Fatal("fixture: the hold fits")
	}
	m.settleRoute(h, 100)
	backup := m.budgetStateToSave()
	apiframework.SaveBudget()

	apiframework.ResetBudgetForTest(dir) // a normal restart: budget.yaml reads back
	booted := &AICompanionModule{cfg: cfg}
	booted.books.Store(apiframework.Shared())
	booted.restoreBudget(backup)
	if ownerSpent(booted, 5) != 1300 || strangerSpent(booted, 2) != 300 || strangersForSpent(booted, 5) != 300 {
		t.Fatalf("seeded once, not twice: owner=%d stranger=%d perOwner=%d",
			ownerSpent(booted, 5), strangerSpent(booted, 2), strangersForSpent(booted, 5))
	}
}
```

- [ ] **Step 2: Run to confirm they fail**

Run: `go test ./modules/aicompanion/ -run 'TestFirstBootSeedsTheOldAllowancesOnce|TestAQuarantinedLedgerReseedsFromTheCompanionsBackup|TestASameDayRestartDoesNotSeedTwice' 2>&1 | head`
Expected: build failure, `restoreBudget` and `budgetStateToSave` undefined.

- [ ] **Step 3: Implement in `models.go`**

Replace the `budgetState` comment with: `// budgetState is the companion's own day on disk: its calls and "you notice" moments. Tokens is the companion's share of the server key's day, still written so a server rolled back to the code before the framework resumes the day where it was. Owners, Strangers and StrangersFor are the ledger's allowances (apiframework Allowances), written here as a backup: every boot hands them to SeedAllowances, which applies once per dimension per day, so they seed the first boot after the move and a boot after budget.yaml was quarantined, and nothing on a normal restart.`
Replace `loadBudget` and `saveBudget` (656-708) with:
```go
func (m *AICompanionModule) loadBudget() {
	var st budgetState
	if err := m.plug.ReadIntoStruct(budgetStateId, &st); err != nil {
		return // nothing recorded yet, or unreadable: start the day fresh
	}
	m.restoreBudget(st)
}

// restoreBudget takes up a saved day, when it is today on the ledger's
// clock. Its server total and its allowances are handed to the ledger
// (SeedTokens, SeedAllowances), which takes them only when it has no day of
// its own to go on: the first boot after the move to the ledger, or a boot
// after budget.yaml was quarantined. On a normal restart the ledger's seed
// marks turn them away, so nothing is counted twice and a corrupt file
// hands out no second allowance.
func (m *AICompanionModule) restoreBudget(st budgetState) {
	if st.Day != m.fw().Day() {
		return // a stale day is simply a new day
	}
	m.countersDay = st.Day
	m.fw().SeedTokens(apiframework.ConsumerCompanion, st.Day, st.Tokens)
	m.fw().SeedAllowances(apiframework.DimCompanionOwner, st.Day, st.Owners)
	m.fw().SeedAllowances(apiframework.DimCompanionStranger, st.Day, st.Strangers)
	m.fw().SeedAllowances(apiframework.DimCompanionStrangersFor, st.Day, st.StrangersFor)
	m.callsToday = st.Calls
	m.noticesToday = st.Notices
	if m.noticesToday == nil {
		m.noticesToday = map[int]int{}
	}
}

func (m *AICompanionModule) saveBudget() {
	if !m.cfg.Enabled {
		return
	}
	apiframework.SaveBudget()
	st := m.budgetStateToSave()
	if err := m.plug.WriteStruct(budgetStateId, &st); err != nil {
		mudlog.Error(`aicompanion`, `action`, `saveBudget`, `error`, err)
	}
}

// budgetStateToSave is the companion's own day as saveBudget writes it,
// with the ledger's allowances copied in as the backup restoreBudget seeds
// from.
func (m *AICompanionModule) budgetStateToSave() budgetState {
	m.rollCounters()
	st := budgetState{Day: m.countersDay, Calls: m.callsToday, Notices: m.noticesToday,
		Owners:       m.fw().Allowances(apiframework.DimCompanionOwner),
		Strangers:    m.fw().Allowances(apiframework.DimCompanionStranger),
		StrangersFor: m.fw().Allowances(apiframework.DimCompanionStrangersFor)}
	if u := m.fw().Today(); u.Day == m.countersDay {
		for _, c := range u.ByConsumer {
			if c.Consumer == apiframework.ConsumerCompanion {
				st.Tokens = c.Tokens
			}
		}
	}
	return st
}
```
If `time` becomes unused in `models.go`, `go vet` will say so; remove the import only then.

- [ ] **Step 4: Run**

Run: `go vet ./modules/aicompanion/ && go test ./modules/aicompanion/ 2>&1 | tail -3`
Expected: `ok`.

- [ ] **Step 5: Prove the seed tests can fail**

(a) In `Books.SeedAllowances` (`budget.go`), temporarily delete `|| k.l.st.Seeded[dim]`. Run `go test ./modules/aicompanion/ -run 'TestFirstBootSeedsTheOldAllowancesOnce|TestASameDayRestartDoesNotSeedTwice'`. Expected: FAIL "boot 1: owner=2400" and "seeded once, not twice". Restore.
(b) In `restoreBudget`, temporarily delete the three `SeedAllowances` lines. Run `-run TestAQuarantinedLedgerReseedsFromTheCompanionsBackup`. Expected: FAIL "re-seeded from the backup". Restore.
(c) In `budgetStateToSave`, temporarily drop the three `Allowances` fields. Same run. Expected: FAIL "the backup is the ledger's counts". Restore; all PASS.

- [ ] **Step 6: Commit**

```bash
git add modules/aicompanion/models.go modules/aicompanion/money_test.go modules/aicompanion/allowance_test.go
git commit -F - <<'EOF'
feat(aicompanion): the companion's file backs up the ledger's allowances

Every boot hands the saved allowances to SeedAllowances; the ledger's
per-dimension marks take them on the first boot after the move and after
a budget.yaml quarantine, and turn them away on a normal restart.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

## Task 10: Baubles charge baubles.finder on both routes

**Files:**
- Modify: `modules/baubles/config.go` (`Config`, `buildConfig`)
- Modify: `modules/baubles/generate.go:53, 151-294` (at `09964d50f`: `m.count` 53, `name` 151-190, `viaPlayer` 196-229, `viaServer` 231-294)
- Modify: `modules/baubles/baubles_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `modules/baubles/baubles_test.go`:
```go
func finderSpent(id int) int { return apiframework.Allowance(apiframework.DimBaublesFinder, id) }

// A find is charged to its finder, on the server's key and on their own.
// Moderation is on, as in slice H's player-key tests, so the finder's find
// is moderated and everyone's (it is charged the same either way).
func TestAFindIsChargedToItsFinder(t *testing.T) {
	f := newFakeOpenAI(t)
	m := testModule(t, f, func(c *Config) { c.ModerateOutput = true })
	if _, err := m.generate(context.Background(), request()); err != nil {
		t.Fatal(err)
	}
	if finderSpent(7) != 240 {
		t.Fatalf("the server's key: the finder is charged what it cost: %d", finderSpent(7))
	}
	relay := &fakeRelay{allowed: map[int]bool{7: true}, model: `player-model`, provider: newFakeOpenAI(t)}
	apiframework.SetRelay(relay)
	before := apiframework.Today().Tokens
	if res, err := m.generate(context.Background(), request()); err != nil || !res.PlayerKey {
		t.Fatalf("fixture: named on the finder's key: %+v %v", res, err)
	}
	if finderSpent(7) != 480 || apiframework.Today().Tokens != before {
		t.Fatalf("their own key: charged to them, nothing of the server's: finder=%d server %d->%d",
			finderSpent(7), before, apiframework.Today().Tokens)
	}
}

// Over their allowance, a finder's find is a generic trinket: no call on
// either key, and the relay's breaker is not fed.
func TestAFinderOverTheirAllowanceGetsNoName(t *testing.T) {
	f := newFakeOpenAI(t)
	// The relay route opens with or without moderation (slice H keeps text
	// it cannot moderate to its finder): only the allowance stops it. Its
	// refusal falls through to the server's key, which the same allowance
	// refuses too.
	m := testModule(t, f, nil)
	relay := &fakeRelay{allowed: map[int]bool{7: true}, model: `player-model`, provider: newFakeOpenAI(t)}
	apiframework.SetRelay(relay)
	apiframework.Shared().SetAllowanceForTest(apiframework.DimBaublesFinder, 7, m.snapshot().DailyTokensPerUser)
	for i := 0; i < 4; i++ { // more refusals in a row than testModule's BreakerErrors (3)
		_, err := m.generate(context.Background(), request())
		if !errors.Is(err, apiframework.ErrOverAllowance) || apiframework.RefusedBy(err) != apiframework.DimBaublesFinder {
			t.Fatalf("over the allowance, and it says whose: %v", err)
		}
	}
	if relay.sends != 0 || atomic.LoadInt32(&f.chats) != 0 || len(relay.results) != 0 {
		t.Fatalf("no call anywhere, the relay's breaker unfed: sends=%d chats=%d results=%v", relay.sends, f.chats, relay.results)
	}
	if apiframework.Blocked(apiframework.ConsumerBaubles, time.Now()) {
		t.Fatal("a refusal feeds no breaker of the server's")
	}
	m.mu.Lock()
	server, player, failures := m.stats.server, m.stats.player, m.stats.failures
	m.mu.Unlock()
	if server != 0 || player != 0 || failures != 0 {
		t.Fatalf("a refusal is no naming and no failure in bauble status: server=%d player=%d failed=%d", server, player, failures)
	}
}

// Baubles hold at most their share of the day's budget; the companion
// still has the rest.
func TestBaublesOverTheirShareFallBack(t *testing.T) {
	f := newFakeOpenAI(t)
	m := testModule(t, f, nil)
	restore := apiframework.SetServerForTest(apiframework.ServerSettings{
		Endpoint:         apiframework.Endpoint{BaseURL: f.srv.URL, APIKey: `sk-test`},
		DailyTokenBudget: 10000, BaublesSharePercent: 25, BreakerErrors: 3, BreakerSeconds: 60,
	})
	t.Cleanup(restore)
	if _, err := apiframework.Reserve(apiframework.ConsumerBaubles, 2400, true); err != nil {
		t.Fatal(err)
	}
	if _, err := m.generate(context.Background(), request()); !errors.Is(err, apiframework.ErrOverShare) ||
		apiframework.RefusedBy(err) != apiframework.RefusedShare {
		t.Fatalf("over a 2500-token share, and it says so: %v", err)
	}
	if atomic.LoadInt32(&f.chats) != 0 {
		t.Fatal("a refused reservation makes no call")
	}
	if _, err := apiframework.Reserve(apiframework.ConsumerCompanion, 5000, true); err != nil {
		t.Fatal("the companion still has the rest of the day")
	}
}

// An admin's regeneration has no finder: it charges no allowance and
// counts under the baubles share.
func TestAdminRegenChargesNoFinder(t *testing.T) {
	f := newFakeOpenAI(t)
	m := testModule(t, f, nil)
	req := request()
	req.FinderUserId = 0
	if _, err := m.generate(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if finderSpent(0) != 0 || baublesTokens() != 240 {
		t.Fatalf("no finder charged, the share counts it: finder0=%d share=%d", finderSpent(0), baublesTokens())
	}
}
```
In `TestBuildConfigDefaultsAndBounds`, add after the first `if`:
```go
	if c.DailyTokensPerUser != 20000 {
		t.Fatalf("default DailyTokensPerUser: %d", c.DailyTokensPerUser)
	}
	if buildConfig(func(k string) any {
		if k == `DailyTokensPerUser` {
			return -5
		}
		return nil
	}).DailyTokensPerUser != 0 {
		t.Fatal("a negative allowance is no cap")
	}
```

Slice H has no named moderation-on helper: its tests pass `func(c *Config) { c.ModerateOutput = true }` to `testModule` inline (`baubles_test.go:222` and on), and the tests above do the same. The fake's `/moderations` answer is slice H's (one result per input, `baubles_test.go:92`). `fakeRelay` (`:504-543`) has the `allowed`, `model`, `sends`, `results` and `provider` fields these tests use; `m.stats` has `server`, `player` and `failures` (`baubles.go:42-47`).

- [ ] **Step 2: Run to confirm they fail**

Run: `go test ./modules/baubles/ 2>&1 | head`
Expected: build failure, `DailyTokensPerUser` undefined.

- [ ] **Step 3: Config**

In `config.go` `Config`, after `MaxConcurrent int` add:
```go

	// DailyTokensPerUser is what one finder's finds may spend in a UTC day,
	// on the server's key or their own (apiframework's baubles.finder
	// allowance). 0 is no cap.
	DailyTokensPerUser int
```
In `buildConfig`'s literal, after `MaxConcurrent:` add `DailyTokensPerUser:  max(asInt(get(`DailyTokensPerUser`), 20000), 0),` (gofmt aligns the column).

- [ ] **Step 4: Routes**

In `generate.go`, add after the `var (...)` block:
```go
// finderCharges is the finder's own allowance, when there is a finder (an
// admin's regeneration has none and counts only under the baubles share).
func finderCharges(cfg Config, finderId int) []apiframework.Charge {
	if finderId <= 0 {
		return nil
	}
	return []apiframework.Charge{{Dim: apiframework.DimBaublesFinder, UserId: finderId, Limit: cfg.DailyTokensPerUser}}
}
```
In `name` (slice H wrapped these calls in `takeFinderSlot` / `takeServerSlot`; only the calls change), `viaPlayer(ctx, r, req.FinderUserId, relayModel, chat)` (`generate.go:162`) becomes `viaPlayer(ctx, cfg, r, req.FinderUserId, relayModel, chat)` and `viaServer(ctx, cfg, chat)` (`:188`) becomes `viaServer(ctx, cfg, chat, req.FinderUserId)`. Nothing else in `name` changes: a refused `viaPlayer` returns the no-op report, so its `default:` branch (`:171-175`) feeds the relay's breaker nothing and, the context being alive, goes on to the server's key, whose reservation carries the same finder charge.
`viaPlayer` as merged at `09964d50f` (`generate.go:200-229`) computes `prompt` and a clamped `tokens` through `Charged(..., relayed=true)` after `DecodeChat` (`:226-227`), and returns `tokens`. Keep all of that; add the reservation around the send. The block below is the merged function with the NEW pieces marked (read against `09964d50f`; the merged body error return is `func(error) {}`, spelled `none` here):
```go
func viaPlayer(ctx context.Context, cfg Config, r apiframework.Relay, userId int, model string, chat apiframework.Chat) (string, int, func(error), error) {
	none := func(error) {} // NEW
	// The outcome is held against the finder's key for finds only (the
	// relay keeps a breaker per purpose, and their companion's is never
	// touched), so a provider that cannot serve finds stops being asked.
	report := func(err error) {
		if !canceled(ctx) {
			r.Result(userId, err)
		}
	}
	chat.Model, chat.Effort = model, ``
	body, err := chat.Body()
	if err != nil {
		return ``, 0, none, err
	}
	prompt := apiframework.EstimateTokens(chat.Messages) + schemaOverhead // NEW position: moved up from after DecodeChat
	// NEW: held against the finder's allowance only. A refusal is nobody's
	// failure, so the relay is neither asked nor fed.
	hold, err := apiframework.Reserve(apiframework.ConsumerBaubles, prompt+chat.MaxTokens, false, finderCharges(cfg, userId)...)
	if err != nil {
		return ``, 0, none, err
	}
	status, raw, sent, err := r.Send(ctx, userId, body, apiframework.CarriesNoPlayerData) // NEW: sent named (merged: _)
	if err == nil && status != http.StatusOK {
		// The body is the provider's own text about the player's own
		// account: neither kept nor logged. The status says enough.
		err = &apiframework.StatusError{Status: status}
	}
	if err != nil {
		// NEW: a request that may have left is charged as the provider may
		// have billed it; one that never left costs nothing.
		n, _ := apiframework.Charged(0, sent, status, prompt, chat.MaxTokens, true)
		apiframework.Settle(hold, n, false)
		return ``, 0, report, err
	}
	reply := apiframework.DecodeChat(status, raw)
	// The count came through the player's browser, which they can write:
	// held to what one request could cost before it is recorded (spec S3).
	tokens, _ := apiframework.Charged(reply.Tokens, true, status, prompt, chat.MaxTokens, true)
	apiframework.Settle(hold, tokens, false) // NEW
	return reply.Content, tokens, report, reply.Err
}
```
Its doc comment's last sentence, "Nothing is reserved against the server's budget; the player's finds breaker is fed." (`generate.go:198-199`), becomes "Nothing of the server's is reserved; the finder's own allowance is, and the player's finds breaker is fed."
In `viaServer`, the signature (`generate.go:239`) becomes `func viaServer(ctx context.Context, cfg Config, chat apiframework.Chat, finderId int) (string, int, func(error), error)` and the reserve line becomes `hold, err := apiframework.Reserve(apiframework.ConsumerBaubles, reserve, true, finderCharges(cfg, finderId)...)`.

A refusal and the breakers, as the code stands at `09964d50f` (`generate.go:47-61` and `:254-262`): a refused `Reserve` in `viaServer` releases its breaker ticket and returns the no-op `none` as `report`, and `viaPlayer` above returns `none` too, so `report(err)` in `generate` (`:59`) and in `name`'s relay `default:` branch (`:172`) feeds no breaker, the provider's, baubles' or the relay's. That stays. What does count it is line 53, `m.count(playerKey, err != nil)`: a refusal shows in `bauble status` as a find "named on the server's key" that "failed". Slice H already keeps one kind of refusal out of the stats: `errSlotsBusy` returns before `m.count` (`:48-52`, "Refused at the door, not a call that failed"). A refused reservation is the same kind of thing, neither a naming nor a failure; the reason is logged at Warn by `baubles.Generate` (`internal/baubles/generate.go:140`) with the error text, which names the counter (`RefusalError.Error()`, e.g. "consumer's share of the daily token budget spent (share)"). It is not folded into the `errSlotsBusy` early return because that return also skips the `LogRequests` line and `report`, and a refusal still logs its `model call` line when `LogRequests` is on. Replace (line 53, after the `errSlotsBusy` block)
```go
	m.count(playerKey, err != nil)
```
with
```go
	// A refused reservation (a spent day, share or allowance) made no call:
	// it is neither a naming nor a failure in `bauble status`, and it feeds
	// no breaker (its report is a no-op). baubles.Generate logs the refusal,
	// naming the counter (apiframework.RefusedBy).
	if apiframework.RefusedBy(err) == `` {
		m.count(playerKey, err != nil)
	}
```
This is the one `m.count` call in `generate.go` at `09964d50f` (it follows `m.name` and the `errSlotsBusy` return).

- [ ] **Step 5: Run**

Run: `go vet ./modules/baubles/ && go test ./modules/baubles/ ./internal/baubles/ ./internal/actions/ 2>&1 | tail -4`
Expected: all `ok` (the old player-key tests still pass: a relay hold touches nothing of the server's).

- [ ] **Step 6: Prove the finder test can fail**

Make `finderCharges` return `nil` always; run `-run 'TestAFindIsChargedToItsFinder|TestAFinderOverTheirAllowanceGetsNoName'`; expected FAIL on both. Restore. Then drop the `RefusedBy` guard around `m.count`; run `-run TestAFinderOverTheirAllowanceGetsNoName`; expected FAIL "a refusal is no naming and no failure in bauble status". Restore; PASS.

- [ ] **Step 7: Commit**

```bash
git add modules/baubles/config.go modules/baubles/generate.go modules/baubles/baubles_test.go
git commit -F - <<'EOF'
feat(baubles): each finder's namings draw on their own daily allowance

FinderUserId reaches viaServer and viaPlayer; both charge baubles.finder,
the finder's own key included (owner ruling 12). An admin regeneration has
no finder and counts only under the share. A refused reservation is neither
a naming nor a failure in bauble status and feeds no breaker.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

## Task 11: The knobs: shares and the finder allowance in config

**Files:**
- Modify: `internal/configs/config.apiframework.go:37-47`
- Modify: `internal/apiframework/settings.go` (`resolveServer`, consts)
- Modify: `internal/apiframework/allowance_test.go`
- Modify: `modules/baubles/baubles_test.go` (the shipped `DailyTokensPerUser`)
- Modify: `_datafiles/config.yaml` (APIFramework block, `Modules.baubles` block), built from the HEAD blob

- [ ] **Step 1: Write the failing tests**

Append to `internal/apiframework/allowance_test.go` (imports gain `"github.com/GoMudEngine/GoMud/internal/configs"` and `"gopkg.in/yaml.v2"`, the library `internal/configs` loads config.yaml with, `configs.go:13`):
```go
func TestSharePercentsResolve(t *testing.T) {
	s := resolveServer(configs.APIFramework{}, legacyConfig{})
	if s.CompanionSharePercent != 0 || s.BaublesSharePercent != 25 {
		t.Fatalf("absent: the companion uncapped, baubles 25: %d/%d", s.CompanionSharePercent, s.BaublesSharePercent)
	}
	s = resolveServer(configs.APIFramework{CompanionSharePercent: 60, BaublesSharePercent: -1}, legacyConfig{})
	if s.CompanionSharePercent != 60 || s.BaublesSharePercent != 0 {
		t.Fatalf("set, and -1 is no cap: %d/%d", s.CompanionSharePercent, s.BaublesSharePercent)
	}
	if s := resolveServer(configs.APIFramework{BaublesSharePercent: 100}, legacyConfig{}); s.BaublesSharePercent != 0 {
		t.Fatal("100 is no cap")
	}
}

// The yaml tags decode: a tag on the wrong field is a silent no-op.
func TestShareKnobsDecode(t *testing.T) {
	var a configs.APIFramework
	if err := yaml.Unmarshal([]byte("CompanionSharePercent: 60\nBaublesSharePercent: 30\n"), &a); err != nil {
		t.Fatal(err)
	}
	if a.CompanionSharePercent != 60 || a.BaublesSharePercent != 30 {
		t.Fatalf("decoded: %+v", a)
	}
}

// The committed config.yaml, read by repo path as the shipped-config tests
// in internal/configs do (TestBaubleShippedConfigMatchesDefaults), resolves
// to the shipped shares: a knob in the wrong block or with a typo is a
// silent no-op otherwise. CI checks out the HEAD blob; locally, run it with
// the disk copy equal to HEAD (Task 11 Step 5).
func TestTheShippedShareKnobs(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(`..`, `..`, `_datafiles`, `config.yaml`))
	if err != nil {
		t.Fatalf("read shipped config: %v", err)
	}
	var shipped struct {
		APIFramework configs.APIFramework `yaml:"APIFramework"`
	}
	if err := yaml.Unmarshal(data, &shipped); err != nil {
		t.Fatalf("decode shipped config: %v", err)
	}
	if shipped.APIFramework.BaublesSharePercent != 25 || shipped.APIFramework.CompanionSharePercent != 0 {
		t.Fatalf("shipped knobs: %+v", shipped.APIFramework)
	}
	s := resolveServer(shipped.APIFramework, legacyConfig{})
	if s.BaublesSharePercent != 25 || s.CompanionSharePercent != 0 {
		t.Fatalf("shipped shares resolve to baubles 25, the companion uncapped: %d/%d", s.BaublesSharePercent, s.CompanionSharePercent)
	}
}
```
Append to `modules/baubles/baubles_test.go` (imports gain `"path/filepath"` and `"gopkg.in/yaml.v2"`):
```go
// The committed config.yaml's Modules.baubles block, read by repo path and
// put through buildConfig, gives each finder the shipped 20000-token day.
func TestTheShippedFinderAllowance(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(`..`, `..`, `_datafiles`, `config.yaml`))
	if err != nil {
		t.Fatalf("read shipped config: %v", err)
	}
	var shipped struct {
		Modules struct {
			Baubles map[string]any `yaml:"baubles"`
		} `yaml:"Modules"`
	}
	if err := yaml.Unmarshal(data, &shipped); err != nil {
		t.Fatalf("decode shipped config: %v", err)
	}
	if v, ok := shipped.Modules.Baubles[`DailyTokensPerUser`]; !ok || v != 20000 {
		t.Fatalf("Modules.baubles.DailyTokensPerUser in config.yaml: %v (%v)", v, ok)
	}
	c := buildConfig(func(k string) any { return shipped.Modules.Baubles[k] })
	if c.DailyTokensPerUser != 20000 {
		t.Fatalf("shipped DailyTokensPerUser through buildConfig: %d", c.DailyTokensPerUser)
	}
}
```
At `09964d50f` (re-verified), `baubles_test.go` still imports `os` and neither `path/filepath` nor a yaml package, so both are added. `configs.APIFramework`'s field types (`ConfigInt`, `ConfigString`, `ConfigSecret`) are plain named types with no `UnmarshalYAML` (`internal/configs/config_types.go:8-11`), so a direct decode matches the loader's.

- [ ] **Step 2: Run to confirm they fail**

Run: `go test ./internal/apiframework/ -run 'TestSharePercentsResolve|TestShareKnobsDecode|TestTheShippedShareKnobs' 2>&1 | head`
Expected: build failure, unknown field `CompanionSharePercent` in `configs.APIFramework`. (`TestTheShippedFinderAllowance` already compiles after Task 10 and fails until Step 5: run `go test ./modules/baubles/ -run TestTheShippedFinderAllowance` and expect FAIL "Modules.baubles.DailyTokensPerUser in config.yaml: <nil> (false)".)

- [ ] **Step 3: Implement**

In `config.apiframework.go`, after `BreakerSeconds ConfigInt ...` add:
```go
	// CompanionSharePercent and BaublesSharePercent cap what each feature
	// may hold of DailyTokenBudget in a UTC day, as a percentage, so one
	// feature cannot spend the day for the others. 0 or absent: the
	// default (the companion 100, baubles 25). -1, or 100 and above: no
	// share cap. With no DailyTokenBudget there is no share cap either.
	CompanionSharePercent ConfigInt `yaml:"CompanionSharePercent"`
	BaublesSharePercent   ConfigInt `yaml:"BaublesSharePercent"`
```
None of the three new knobs goes on slice M's hard-lock list (`internal/configs/config_locks.go:15-46`): its own comment keeps the daily budget knobs tunable in game (`:32-35`, "The daily budget knobs stay tunable in game"), and `DailyTokenBudget` itself is not on it. The `Modules.baubles` comment in config.yaml that names the hard-locked keys (`config.yaml:2686-2687`) therefore stays as it is.

In `settings.go`, add to the defaults `const` block (`settings.go:49-53` at `09964d50f`):
```go
	DefaultCompanionSharePercent = 100
	DefaultBaublesSharePercent   = 25
```
In `resolveServer` (`settings.go:212`), before its `return s` (`:275`; the `return s` at `:207` belongs to the function before it):
```go
	s.CompanionSharePercent = sharePercent(int(c.CompanionSharePercent), DefaultCompanionSharePercent)
	s.BaublesSharePercent = sharePercent(int(c.BaublesSharePercent), DefaultBaublesSharePercent)
```
After `pick`:
```go
// sharePercent is a consumer's share as ServerSettings keeps it: 0 or
// absent takes def; below 0, or 100 and above, is no share cap (0).
func sharePercent(v int, def int) int {
	if v == 0 {
		v = def
	}
	if v < 0 || v >= 100 {
		return 0
	}
	return v
}
```

- [ ] **Step 4: Run**

Run: `go test ./internal/apiframework/ -run 'TestSharePercentsResolve|TestShareKnobsDecode' && go test ./internal/configs/ 2>&1 | tail -3`
Expected: `ok` twice. (`TestTheShippedShareKnobs` still fails: config.yaml has no knob until Step 5.)

- [ ] **Step 5: Add the knobs to `config.yaml` from the HEAD blob**

In the S5 worktree the file is not skip-worktree (`H`) and equals HEAD (Task 1 Step 2); the checks below confirm it rather than assume it. Run (Bash):
```bash
cd /c/tmp/dogmud-baubles-s5
git ls-files -v _datafiles/config.yaml
git show HEAD:_datafiles/config.yaml > "$TEMP/config.head.yaml"
cmp -s "$TEMP/config.head.yaml" _datafiles/config.yaml && echo SAME || echo DIFFERS
```
If `DIFFERS`: `cp _datafiles/config.yaml "$TEMP/config.local.yaml"` (backup), then `cp "$TEMP/config.head.yaml" _datafiles/config.yaml` so the edit starts from the committed blob. If the first line printed `S`, run `git update-index --no-skip-worktree _datafiles/config.yaml` before staging and restore it in Step 6.

With the Edit tool, in the `APIFramework:` block replace
```yaml
  BreakerErrors: 0
  BreakerSeconds: 0
```
with
```yaml
  BreakerErrors: 0
  BreakerSeconds: 0
  # What each feature may hold of DailyTokenBudget in a UTC day, as a
  # percentage, so one feature cannot spend the day for the others. 0: the
  # default (the companion 100, baubles 25). -1 or 100: no share cap. With
  # no DailyTokenBudget there is no share cap either.
  CompanionSharePercent: 0
  BaublesSharePercent: 25
```
and in the `Modules.baubles` block replace the line slice H left (its Task 15 Step 8 rewrote the comment; Task 1 Step 4 confirmed this exact text)
```yaml
    MaxConcurrent: 4           # server-key calls at once; more are generic
```
with
```yaml
    MaxConcurrent: 4           # server-key calls at once; more are generic
    # Tokens per UTC day one finder's finds may spend, on the server's key
    # or their own (about 8 to 10 names). Over it, a find is a generic
    # trinket. An admin's regeneration charges nobody. 0 is no cap.
    DailyTokensPerUser: 20000
```
Also update the companion's allowance comment (the block before `DailyTokensPerCompanion: 300000`): replace `    # not hand out a fresh allowance.` with `    # not hand out a fresh allowance (APIFramework's budget.yaml keeps it,`, followed by a new line `    # and the companion's own budget file backs it up).`

- [ ] **Step 6: Check, commit, restore any local edits**

Run (each standalone):
```bash
go test -count=1 -run TestTheShippedShareKnobs ./internal/apiframework/
go test -count=1 -run TestTheShippedFinderAllowance ./modules/baubles/
DOGMUD_BOOT_SMOKE=1 go test -run 'TestSmoke_NoNewSilentlyIgnoredYAMLKeys' -timeout 300s . 2>&1 | tail -3
```
Expected: `ok` three times. Prove the shipped tests can fail: temporarily change `BaublesSharePercent: 25` to `BaublesSharePercent: 30` and `DailyTokensPerUser: 20000` to `DailyTokensPerUser: 2000` in the disk copy; re-run the first two; expected FAIL "shipped knobs" and "Modules.baubles.DailyTokensPerUser in config.yaml". Restore both values with the Edit tool and confirm `git diff --stat -- _datafiles/config.yaml` shows only this task's additions.
```bash
git add internal/configs/config.apiframework.go internal/apiframework/settings.go internal/apiframework/allowance_test.go modules/baubles/baubles_test.go _datafiles/config.yaml
git commit -F - <<'EOF'
feat(config): consumer share knobs and the per-finder bauble allowance

APIFramework.CompanionSharePercent (0: the default 100, no cap) and
BaublesSharePercent (25); Modules.baubles.DailyTokensPerUser (20000).
Tests read the committed config.yaml and resolve both.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
git ls-files -v _datafiles/config.yaml
```
If a backup was made in Step 5, re-apply only its local-only differences (compare `$TEMP/config.local.yaml` against `git show HEAD~1:_datafiles/config.yaml`) with the Edit tool, then, if the bit was `S` before, `git update-index --skip-worktree _datafiles/config.yaml` and confirm `git ls-files -v _datafiles/config.yaml` prints `S`.

---

## Task 12: Remove the companion's old maps

**Files:**
- Modify: `modules/aicompanion/aicompanion.go:172-174`

- [ ] **Step 1: Delete the declarations and let the compiler enumerate**

Delete the three lines declaring `ownerTokens`, `strangerTokens`, `strangersFor` (keep `noticesToday`). Run: `go build ./... && go vet ./modules/aicompanion/ 2>&1 | head -30`
Expected: no errors (Tasks 7 to 9 removed every consumer). Any error printed is a consumer the tasks missed: fix it the way its task says, then rebuild.

- [ ] **Step 2: Sweep the non-Go surfaces**

Run each standalone (zero matches exit 1, which is the pass):
```bash
grep -rn 'ownerTokens\|strangerTokens\|strangersFor\b\|budgetDay\|rollDay\|strangerFits\|chargeStrangerFor\|chargeOwner\|tryReserveFor\|tryReserveTokens\|settleForDay\|settleHeld' modules/ internal/ docs/aicompanion/
```
Expected: matches only in `modules/aicompanion/context.md` and `docs/` prose, which Task 14 rewrites. Record the list for Task 14. `StrangersFor` (capital, the `budgetState` field, still written as the backup) is expected in `models.go`, `money_test.go` and `allowance_test.go` only.

- [ ] **Step 3: Run and commit**

Run: `go test ./modules/aicompanion/ 2>&1 | tail -3` (expected `ok`)
```bash
git add modules/aicompanion/aicompanion.go
git commit -F - <<'EOF'
refactor(aicompanion): drop the old allowance maps; the ledger keeps them

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

## Task 13: S5 code review checkpoint

S5's own review, before the gate. The owner's rulings on corrections 3, 6, 7, 8 and 9 are settled (ruling 12); the review checks the code honours them, it does not reopen them.

**Files:** none changed unless the review finds something.

- [ ] **Step 1: Dispatch an independent reviewer**

Use `superpowers:requesting-code-review` with a fresh subagent (sonnet or better). Give it: the S5 spec section and owner rulings 11 and 12, this plan's rule table and corrections, and the diff from Task 1's base:
```bash
cd /c/tmp/dogmud-baubles-s5
BASE=$(git merge-base HEAD origin/master); echo "BASE=$BASE"
git diff $BASE..HEAD -- internal/apiframework modules/aicompanion modules/baubles internal/configs _datafiles/config.yaml
```
(`BASE` must print the SHA Task 1 Step 2 recorded.) Ask it to confirm, row by row, that each of R1 to R43 is pinned by a test in the new tree (name the test and line; R42 is pinned by its dropped-on-purpose shape), that no rule lost a test, and to check specifically: all-or-nothing under one lock; the relay clamp versus the kept server overage (correction 3, ruling 12); key 0 refunded (correction 6); the finder charge on the own-key route (correction 7); share 0 as the default and -1 or 100 as no cap (correction 8); a refusal naming its counter in the companion log and in the bauble error, and never feeding a breaker or the bauble failure count (R43); `SaveBudget` copying every map; the seed marks one per dimension per day, a same-day restart not seeding twice, and a quarantine re-seeding from the companion's backup (correction 9).

- [ ] **Step 2: Fix what it finds**

Fix every reviewer finding in delivered work (do not revert it), each fix with its own test and commit, and re-run that task's prove-it-can-fail step for any test the fix touches. A finding that the plan itself got a fact wrong is corrected in this plan (Edit tool) in the same commit as the code fix.

---

## Task 14: Gate and docs

**Files:**
- Modify: `internal/apiframework/context.md`, `modules/aicompanion/context.md`, `modules/baubles/context.md`, `docs/aicompanion/settings.md`

- [ ] **Step 1: Docs**

Every anchor below is quoted exactly as it stands at `09964d50f`, line breaks included (the Edit tool's `old_string` spans them). Task 1 Step 4 confirmed each one against merged master (two had drifted with slice H and are corrected here); if a paragraph was re-wrapped since, re-grep a distinctive phrase (`grep -n`) and quote the lines as they now stand. Symbols named here exist by Task 12 (verify with `grep -n 'func Allowances\|func SeedAllowances\|func RefusedBy\|func (m \*AICompanionModule) restoreBudget\|func (m \*AICompanionModule) allowanceCharges' -r internal/apiframework modules/aicompanion`, expecting five hits).

`internal/apiframework/context.md`, with the Edit tool:
- "How a server-key call goes" step 3: replace the one line
  ```
  3. `Reserve(consumer, worstCase)` holds tokens against the one budget.
  ```
  with
  ```
  3. `Reserve(consumer, worstCase, spendServer, charges...)` holds tokens, all
     or nothing, against the one budget, the consumer's share (`SharePercent`)
     and each per-user `Charge` (`DimCompanionOwner`, `DimCompanionStranger`,
     `DimCompanionStrangersFor`, `DimBaublesFinder`, each with its `Limit`).
     `spendServer` false is a player's own key: allowances only. A refusal is
     a `*RefusalError`; `RefusedBy` names the counter (`global`, `share`, or
     the dimension).
  ```
- Step 5: replace the line
  ```
     and books the real use under the consumer.
  ```
  with
  ```
     and books the real use under the consumer, on every counter the hold
     touched. It clamps a relayed count to its hold, keeps a server-key
     overage, floors every counter at 0, and refunds no allowance from an
     earlier day's hold.
  ```
- "Config": slice H continued the paragraph that ended `defaults on purpose.` with the endpoint allowlist; it now ends with the line `` a `configs.ConfigSecret`. `` (`context.md:64` at `09964d50f`, the only match). Replace that line with
  ```
  a `configs.ConfigSecret`. `CompanionSharePercent` (0: the default, 100, no
  cap) and `BaublesSharePercent` (0: the default, 25) cap each feature's part
  of `DailyTokenBudget`; -1 or 100 is no share cap, and so is no budget.
  ```
- "Gotchas", the "One budget" bullet: replace
  ```
  - **One budget.** `Reserve` checks the whole day against
    `DailyTokenBudget`; the per-consumer figures in `Today().ByConsumer` are for
    display only, not separate caps. The companion keeps its own per-player
    caps on top.
  ```
  with
  ```
  - **One budget, with shares.** `Reserve` checks the whole day against
    `DailyTokenBudget` and each consumer's `ByConsumer` figure against its
    share (`SharePercent`). Per-user allowances are the ledger's too
    (`Allowance`, `Allowances`, `by_user` in budget.yaml), each call's limit
    riding on its `Charge`; `Books.Day()` is the only day.
  ```
- The "Living state" bullet: replace
  ```
    corrupt file is quarantined and the day starts fresh. `SeedTokens` migrates
    the companion's old saved total once, on a fresh day only. The directory is
  ```
  with
  ```
    corrupt file is quarantined and the day starts fresh. `SeedTokens` migrates
    the companion's old saved total once, on a fresh day only.
    `SeedAllowances` takes a feature's own copy of one dimension once per day
    (one `seeded` mark per dimension, saved with the day): the companion
    writes its allowances to its own file as a backup (`Allowances`), so a
    normal restart seeds nothing and a quarantine, which loses the marks with
    the counts, re-seeds from that backup. The directory is
  ```

`modules/aicompanion/context.md`, with the Edit tool:
- "Persistence": replace
  ```
  module's own budget file keeps the per-owner and passer-by counts, and an
  old file's server total seeds the ledger once (`SeedTokens`). A corrupt file is quarantined by
  ```
  with
  ```
  per-owner and passer-by allowances are the ledger's too (`companion.owner`,
  `companion.stranger`, `companion.strangersfor`). The module's own budget
  file keeps its calls and notices and a backup of those allowances
  (`budgetStateToSave`, from `Allowances`); every boot hands it to
  `restoreBudget`, which seeds the ledger (`SeedTokens`, `SeedAllowances`)
  only on the first boot after the move or after a quarantined budget.yaml.
  A corrupt file is quarantined by
  ```
  (the line before it ends `(`<DataFiles>/apiframework/budget.yaml`); the`, which still reads correctly before "per-owner").
- "Model tiers": replace
  ```
    of that owner's companion (`StrangerTokensPerOwner`, `strangersFor`,
    kept in the budget file; `strangerFits`, `chargeStrangerFor`) and, on
  ```
  with
  ```
    of that owner's companion (`StrangerTokensPerOwner`; `allowanceCharges`,
    reserved on the ledger by `reserveRoute`) and, on
  ```
- Fix every other match Task 12 Step 2 listed the same way.

`modules/baubles/context.md`, with the Edit tool (at `09964d50f` the Config list wraps `MaxConcurrent`'s entry across two lines, `context.md:104-105`):
- Replace the two lines
  ```
  `MaxCompletionTokens` (800), `RetryTransient` (false), `MaxConcurrent` (4,
  server-key calls only), `ModerateOutput` (true), `ModerationModel` (omni-moderation-latest),
  ```
  with
  ```
  `MaxCompletionTokens` (800), `RetryTransient` (false), `MaxConcurrent` (4,
  server-key calls only), `DailyTokensPerUser` (20000; each finder's
  `baubles.finder` allowance, on either key; 0 is no cap; not hard-locked),
  `ModerateOutput` (true), `ModerationModel` (omni-moderation-latest),
  ```
- "How a call goes" step 3 (the relay route slice H described): replace `` It costs the server nothing. A `` (in the line `   player's browser relay, on their key. It costs the server nothing. A`) with `` It costs the server nothing but is held against the finder's own `baubles.finder` allowance (`finderCharges`); a refusal is no call and feeds no breaker, and the find goes on to the server's key, where the same allowance refuses it. A ``, then re-wrap that paragraph to 80 columns.
- "How a call goes" step 4 (`viaServer`): replace `` a hold on the one daily budget; `` (in the line ``   lets one probe through); a hold on the one daily budget; `apiframework.Post` ``) with `` a hold on the one daily budget, the baubles share and the finder's allowance (none for an admin's regeneration, which has no finder); ``, then re-wrap that paragraph to 80 columns.
- Replace the line `The key, endpoint, daily budget and breaker are not here: they are the` with `The key, endpoint, daily budget, the baubles share (`APIFramework.BaublesSharePercent`) and breaker are not here: they are the`.
- Add to "Gotchas": `` - **A refused reservation is not a failure.** A spent day, share or finder allowance (`apiframework.RefusedBy`) makes no call, feeds no breaker and is not counted in `bauble status`; `baubles.Generate` logs it with the counter's name. ``

`docs/aicompanion/settings.md`, with the Edit tool: replace
```
server-key total, the limit and the companion's share of it. The spend is
kept in `_datafiles/apiframework/budget.yaml` across restarts.
```
with
```
server-key total, the limit and the companion's share of it. The spend is
kept in `_datafiles/apiframework/budget.yaml` across restarts. So are each
companion's and each passer-by's allowances (`DailyTokensPerCompanion`,
`StrangerDailyTokens`, `StrangerTokensPerOwner`), backed up in the
companion's own budget file, and the companion's share of the day
(`APIFramework.CompanionSharePercent`, no cap by default).
```

- [ ] **Step 2: Format (Windows CRLF caveat)**

`gofmt -l` on this checkout can report false positives (CRLF working copy, LF blob). Check the committed blobs instead (Bash):
```bash
cd /c/tmp/dogmud-baubles-s5
BASE=$(git merge-base HEAD origin/master); echo "BASE=$BASE"
for f in $(git diff --name-only $BASE..HEAD -- '*.go'); do out=$(git show HEAD:"$f" | gofmt -l); [ -n "$out" ] && echo "UNFORMATTED $f"; done; echo done
```
Expected: `BASE` is Task 1's SHA, then only `done`.

- [ ] **Step 3: Size**

```bash
cd /c/tmp/dogmud-baubles-s5
BASE=$(git merge-base HEAD origin/master)
git diff --shortstat $BASE..HEAD
```
Expected: under 20,000 changed lines and under 300 files (CI's lint gate inverts past either, ruling 11). S5 is about 25 files; a number near either limit means something unintended is in the branch: stop and report.

- [ ] **Step 4: Vet, tests and root guards**

```bash
cd /c/tmp/dogmud-baubles-s5
go build ./...
go vet ./internal/apiframework/ ./modules/aicompanion/ ./modules/baubles/ ./internal/baubles/ ./internal/configs/
go test -count=1 ./internal/apiframework/ ./modules/aicompanion/ ./modules/baubles/ ./internal/baubles/ ./internal/configs/ ./internal/actions/
go test -count=1 .
```
Expected: vet silent, every package `ok`. The last line is the root guards (the repo-root test package).

- [ ] **Step 5: Lint (dogmud-shipping, pre-push step 3)**

```bash
cd /c/tmp/dogmud-baubles-s5
git fetch origin
~/go/bin/golangci-lint --version
~/go/bin/golangci-lint run --new-from-merge-base=origin/master
```
Expected: `0 issues`. Check the version matches `.github/workflows/` first, as the skill says. A finding on a line S5 did not write may still be S5's to fix if S5 re-aligned it (gofmt re-tabbing): check `git log -L` before assuming.

- [ ] **Step 6: Boot smoke**

```bash
cd /c/tmp/dogmud-baubles-s5
DOGMUD_BOOT_SMOKE=1 go test -run 'TestSmoke_ServerBootsCleanWithRealData|TestSmoke_NoNewSilentlyIgnoredYAMLKeys' -timeout 300s .
```
Expected: `ok`. The boot smoke test is a Go test, not a server this session keeps running; it touches no server the owner runs.

- [ ] **Step 7: Race**

`-race` needs CGO, absent here. Run the S5 packages in the Docker test image:
```bash
cd /c/tmp/dogmud-baubles-s5
docker compose -f compose.test.yml run --build --rm test go test -race -count=1 ./internal/apiframework/ ./modules/aicompanion/ ./modules/baubles/
```
Expected: three `ok`. Without Docker, CI's `go test -timeout 900s -race ./...` on the PR is the race gate; say so in the handoff.

- [ ] **Step 8: Commit the docs**

```bash
cd /c/tmp/dogmud-baubles-s5
git add internal/apiframework/context.md modules/aicompanion/context.md modules/baubles/context.md docs/aicompanion/settings.md
git commit -F - <<'EOF'
docs(apiframework): per-user allowances and shares live in the ledger

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```
Then re-run Step 3's size check. S5 reaches master as its own PR from `fix/baubles-allowance-ledger` (ruling 11), opened per `dogmud-shipping` with `--repo pruuk/DOGMud`; never pushed onto FinalTwist's branch. No deploy: the owner runs every deploy.

---

## Self-review

- **Spec coverage.** API: `Charge` (Task 2), `Reserve(consumer, tokens, spendServer, charges...)` (2, 4), `Hold` fields (2), `Settle` clamp and floor (3), `Allowance` (2), `SeedAllowances` and `Allowances` (5), refusals naming their counter (2, 4, 7, 10). Day: ledger clock only (8). Persistence: `by_user` deep copy (5), seed marks one per dimension per day (5), the companion's file as the allowances' backup, re-seeding after a quarantine and not on a normal restart (9). Config: shares and `DailyTokensPerUser` (4, 10, 11). Callers: baubles with finder (10); companion one to one (6, 7). Tests: rule table first (1); 64 references rewritten (7, T1); two-charge refusal (2); relay leaves server total (2); settle clamp and floor (3); cross-midnight (3, 7); `-race` `SaveBudget` (5); first-boot seed, quarantine re-seed, same-day restart (9); baubles over share and over per-user limit, a refusal not counted as a failure (10); the committed config.yaml resolving to the shipped knobs (11). Review (13). Old maps removed last (12). Gate from `$BASE`: format, size, vet and tests, root guards, lint, boot smoke, Docker `-race` (14).
- **Deviations from the spec text** are the ten corrections above; 3, 6, 7, 8 and 9 were settled by owner ruling 12, and 10 is a rule dropped on purpose (R42). None is an open question.
- **Names used across tasks:** `Charge{Dim, UserId, Limit}`, `Hold{Consumer, Tokens, Day, SpendServer, Charges}`, `DimCompanionOwner`, `DimCompanionStranger`, `DimCompanionStrangersFor`, `DimBaublesFinder`, `ErrOverAllowance`, `ErrOverShare`, `RefusalError`, `RefusedBy`, `RefusedGlobal`, `RefusedShare`, `Books.Allowance`, `Books.Allowances`, `Books.Day`, `Books.SetAllowanceForTest`, `SeedAllowances`, `ServerSettings.SharePercent`, `hold.refusal`, `allowanceCharges`, `rollCounters`, `countersDay`, `countCall`, `restoreBudget`, `budgetStateToSave`, `finderCharges`, `ownerSpent`/`setOwnerSpent` and siblings: each defined once, in the task that first uses it.
