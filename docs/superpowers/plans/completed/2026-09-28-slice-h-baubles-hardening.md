# Slice H: Baubles Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land sections S1, S2, S3, S4, U2 and U4 of the approved spec
`docs/superpowers/specs/2026-09-28-baubles-hardening-and-corpus-design.md`
on master, as its own PR (owner ruling 11), plus the four owner rulings of
2026-09-28/29 (amendment below): the server key is a secret, it only goes
to OpenAI, text named on a player's key is moderated (everyone sees it) or,
where the server cannot moderate, kept to its finder (everyone else sees
the generic trinket), every reply is cleaned of invisible Unicode and
links, the bauble search chance pays the sight ramp, the look, search and
stolen-bauble observer lines hide names by sight, a failed pickpocket roll
is caught however the pause ends, the household-bauble refusal holds for
every taker, and nobody may pickpocket a companion.

**Architecture:** Engine-side rules (`internal/baubles`, `internal/items`,
`internal/apiframework`, `internal/actions`) carry every invariant, so a
future generator or the corpus (slice C) cannot skip them; the module
(`modules/baubles`) adds the moderation policy, a per-finder slot and the
allowlist fallback. Per-viewer naming is fail-safe: every viewer-agnostic
item accessor shows a finder-only bauble as the generic trinket, and a
short, guarded list of single-reader sites asks for the finder's view.
S5 (per-user allowances) and slice C (corpus) are out of scope. Slice M
and PR #175 are both on master.

**Tech Stack:** Go 1.25, `golang.org/x/text/unicode/norm` (already a
direct requirement at v0.36.0), the repo-root AST guards.

**Worktree:** `C:\tmp\dogmud-baubles-h`, branch `fix/baubles-hardening-h`,
created from `origin/master` at `3bd6ccaa3` (the amendment commit is the
branch's first commit). All paths below are relative to it. Run git from
Bash, Windows process work from PowerShell. Every diff and gate compares
against `$BASE`, the master commit Task 0 records in
`/c/tmp/dogmud-baubles-h.base` (outside the worktree), never
`master...HEAD`: master may move while the slice is in flight.

**CI is out of minutes until 2026-10-01.** Each PR's gate (Tasks 5b, 11b,
14e) is fully local: `-race` runs in the Linux test container (`docker compose -f
compose.test.yml run --build --rm test go test -race ...`), and the real
boot uses private ports through a `CONFIG_PATH` override file, never the
main checkout's config, and stops only its own PID.

---

## Amendment 2026-09-29: facts verified against source at `3bd6ccaa3`

Everything this amendment adds or changes was read at `origin/master`
`3bd6ccaa3` (PR #175 merged, the cleanup #184, slice M #179). The original
table further down was read at `e711ee9de`; its 64 quoted anchors were
re-grepped at `3bd6ccaa3` and 63 hold. The one that moved is row "Stolen-bauble
observer lines" (FinalTwist's recognition sight gate rewrote `ownerRecognizes`),
corrected below and in Task 14.

| Fact | Where (at `3bd6ccaa3`) |
|---|---|
| Slice M is on master: `var hardLocked = []string{...}` holds the four `APIFramework.*` paths, eleven `Modules.aicompanion.*` rows ending `ModerateOutput`, `ModerationModel` (after a `// ruling 13` comment), no `Modules.baubles.*`; `isHardLocked` and `IsLocked` beside it | `internal/configs/config_locks.go:15-36, 39, 54` |
| `func (c Config) DisplayConfigData(excludeStrings ...string) map[string]any`; redacts a `ConfigSecret` and any path segment matching `(?i)(apikey\|api_key\|secret\|password\|token\|webhookurl\|secretkey\|privatekey)$`; `APIKeyEnv` does not match, so Task 1's probe still goes red | `internal/configs/config_display.go:35, 47` |
| S1, S2, S4, U2 are still unfixed: `APIKey ConfigString`; `EndpointAllowed` accepts any `.openai.com`/`.azure.com`; `cleanLine` has no NFKC; no `SightPenalty` in `find.go`, `search_bauble.go`, `search_feature.go`; no `"baubles"`/`"actions"` rows in the sight guard | `config.apiframework.go:25, 54`; `settings.go:32`; `validate.go:60-71`; greps empty |
| S3 is still unfixed: `moderate` accepts an unmoderated player-key reply when there is no server key, the provider breaker is open or the check fails (`unavailable`); moderates `Name` and `Description` only; `generate` discards `CleanReply`'s result | `modules/baubles/generate.go:248-276, 73-76` |
| `Record` has `FoundByUserId int` (`found_by_user_id`), `PlayerKey`, `Moderated`; no `FinderOnly`. `Record.View()` returns the record's text, or `retiredName`/`retiredDescription` when retired | `internal/baubles/record.go:63, 104-105, 123-137` |
| `GenResult` fields end `Moderated`, `PlayerKey`; `Generate` runs `CleanReply` then `TooBigFor` | `internal/baubles/generate.go:53-61, 112-153` |
| `genericName = "Trinket"`, `genericNameSimple = "trinket"`, four `genericDescriptions`; `fallback.go` has no imports | `internal/baubles/fallback.go:13-29` |
| `Mint` copies `PlayerKey: res.PlayerKey` into the record and never reads a viewer | `internal/baubles/mint.go:87-114` (`PlayerKey` at `:111`) |
| `ApplyRegenerated` sets `Moderated`, never `PlayerKey`; a regeneration request never carries `FinderUserId` (`BaubleRequestForRecord`), so a regen is always server-key | `internal/baubles/admin.go:207-239`; `internal/actions/bauble_admin.go:45-64` |
| An item's text is viewer-agnostic: `GetSpec()` overlays the resolver's `BaubleView` (`baubleSpec`); `DisplayName`, `Name`, `NameSimple`, `NameComplex`, `GetLongDescription` all read `GetSpec()` | `internal/items/items.go:238-240, 325-336, 492-538, 540-570, 574-594`; `internal/items/bauble.go:26-33, 62-89` |
| `DisplayName` calls `i.GetSpec()` twice (`QuestToken` at `:502`, `spec :=` at `:522`); `items.go` has a second `spec := i.GetSpec()` at `:215` | `internal/items/items.go` |
| `NameMatch` word-matches a bauble through `baubleWordMatch(input, simpleName, displayName, withoutPossessives(i.Name()))` (partial only); `matchStrength` builds `names := []string{...i.Name()..., ...i.NameSimple()..., withoutPossessives(i.Name())}` | `internal/items/items.go:629`; `internal/items/bauble.go:197` |
| Render sites that can show a bauble's name to someone other than its finder, counted by a scan for room sends with an item name within five lines: about 156 room-broadcast sites in about 60 files (`get.go` 17, `equip.go` 8, `steal.go` 7, `auctions.go` 6, `storage.go` 6, `loot.go` 6, `look.go` 6, `give.go` 5, ...), plus templates (`character/inventory-look` shows another player's inventory, 18 templates render items), the GMCP room payload (`gmcp.Room.go:257`), shop and auction listings, the room ground listing. Routing each through a viewer-aware name is not a contained change; this plan inverts the default instead (ruling 2 below) | scan run 2026-09-29 |
| Single-reader sites that show a bauble to the person who holds or finds it: the find lines (`search_bauble.go:322` `name := itm.DisplayName()`, `BaubleDelivery.deliver`), the pickpocket success line (`steal.go:314, 325, 332` in `takeFromMob`), the inventory list (`inventory.go:196, 274-279`), `look` at a carried item (`look.go:335, 350`) and a floor item (`look.go:534, 546`), the room listing (`look.go:768, 789`, `lookRoom`), the bauble appraisal (`appraise.go:107-121`, `appraiseBauble`, which also prints `rec.Material`), the web client backpack (`gmcp.Char.go:529` in `GMCPCharModule.GetCharNode`, `newInventory_Item` sets `Name: itm.Name()` at `:897`) | as listed |
| The admin `bauble show` prints `rec.Name`, `rec.Material`, `rec.Description` straight from the record (staff; allowed real text) | `internal/usercommands/admin.bauble.go:170-199` |
| `lookupFuncName(fd *ast.FuncDecl) string` (root package `main`) names a method `Type.Method`; the lookup guard keys `"path|function"` | `lookup_viewer_guard_test.go:133-147` |
| Room send methods: `SendTextCommunication`, `SendText`, `SendTextVisual`, `SendTextVisualHidingNames`, `SendTextVisualAsLit`, `SendTextVisualAsLitHidingNames`, `SendTextVisualWithAudio`, `SendTextToExits`; `messaging.SendTrio` | `internal/rooms/rooms.go:226-516`; `internal/messaging/trio.go:108` |
| Breakers: `Allow` admits BOTH the consumer's and the provider's breaker; `Record` records both (the provider's only on `ProviderFailure`); `consumerBreaker(consumer)` and `breaker.record(ticket, failed, now, limit, cooldown)` exist; nothing records a consumer alone. `Blocked(consumer, now)` is provider OR consumer; `Shared()` returns the `*Books`; `SetConsumerBreakerForTest`, `ConsumerFailures`, `NewBooksForTest` exist | `internal/apiframework/breaker.go:91-122, 234-246, 258-299, 304, 347, 390`; `books.go:28` |
| `apiframework.Moderate(ep, model, timeout, texts, carries, admit)` feeds no breaker | `internal/apiframework/transport.go:137-166` |
| Pickpocket: `resolve` treats a thief offline, elsewhere, in combat or attacked, or a mark gone, dead or moved, as "chance lost" whatever the roll; only then `AwardResolved`, and `if !p.success { return caughtByMob(thief, m, room) }`. `pocketAttempt` keeps `roomId`, not the room | `internal/actions/steal_pocket.go:54-80, 272-330` |
| `caughtByMob(actor, m, room)` sends the actor line and the room line, then `thiefCaught`; `thiefCaught(actor, m, room)` = awareness reveal, wake a sleeper, the faction crime block (all by `actor.GetUserId()`: `crimes.IdentifiedPerp`, `factions.BumpRep`, `justice.MaybeDeclareBounty`, knowledge), then `m.Command("attack @id")`. Every call in the crime block takes a user id and works for an offline player | `internal/actions/steal.go:358-379, 620-680`; `factions.go:102`; `bounty.go:88`; `crimes.go:270` |
| Tests resolve a pickpocket in line (`resolvePocketInLine`) with `pocketThief` returning `p.actor, p.actor != nil`; `TestPickpocketAwardsAndCatchesAtTheReveal`'s second half pins today's "walks off after a failed roll: no award, not caught" | `internal/actions/bauble_testinit_test.go:19-34`; `internal/actions/pickpocket_test.go:410-436` |
| `GetItemFromFloor(actor, itemName, stash)` = `FindOnFloor` then `TransferItemToBackpack`; no household check. Callers: `usercommands/get.go:650, 670` and `mobcommands/get.go:85` (every mob, companions and scavengers included; its `get all` calls `Get(item.Name(), ...)` per floor item) | `internal/actions/get.go:18-37` |
| The player-only household refusal: `usercommands/get.go:640-648` (a peek before `GetItemFromFloor`); `get all` and `get all <name>` skip via `BaubleBelongsTo` (`:30-42`, `:231-236`, `takeableOnFloor` `:818-833`) | `internal/usercommands/get.go` |
| `hooks.EquipBestFloorItem` removes a floor item without `GetItemFromFloor`, but only an item that scores as an equipment upgrade; a bauble (no slot) never does | `internal/hooks/mob_equip_best_floor_item.go:28-57` |
| `stealFromMob` refuses `IsNonCombatant() \|\| PlayerAttackImmune` and says, above it, "Deliberately NOT mobs.CheckPlayerHarm: ... stealing from a companion is currently allowed" | `internal/actions/steal.go:190-202` |
| `mobs.CheckPlayerHarm(m)`: charmed first (`HarmBlockedCompanion`, "any player's companion is off-limits, not only the actor's own"), then non-combatant, then attack-immune | `internal/mobs/harm_authorization.go:41-55` |
| An AI companion is `companionai.IsBondedCompanion(instanceId)` (the aicompanion module's controller), which need not be charmed; `steal_pocket.go` already imports `companionai` | `internal/companionai/companionai.go:73`; `modules/aicompanion/aicompanion.go:561` |
| Observer lines after FinalTwist's recognition gate: `ownerRecognizes` now builds `who` ("a figure" unless the OWNER sees clearly) and sends `points at %s. "That's mine! Thief!"` at `stolen_bauble.go:253-255`; "looks overjoyed" at `:325`; `steal.go` room lines at `:364` (gets caught trying to steal), `:537` (is caught stealing from), `:802` (is caught trying to pocket) | `internal/actions/stolen_bauble.go`, `internal/actions/steal.go` |
| Local test tooling: `compose.test.yml` builds `provisioning/Dockerfile` target `test` (`golang:1.25.0-bookworm`, `CGO_ENABLED=1`, `WORKDIR /src`); `CONFIG_PATH` names the override file (`configs.overridePathFor`); shipped ports `TelnetPort: [33333, 44444]`, `LocalPort: 9999`, `HttpPort: 80`, `HttpsPort: 0`, `AIPort: 55555` | `compose.test.yml`; `provisioning/Dockerfile`; `internal/configs/configs.go:465-471`; `config.yaml:2258-2341` |

### Owner rulings added to this slice (2026-09-28/29) and the calls this plan makes

1. **Pickpocket walk-out (Task 14c).** A FAILED roll is caught however the
   pause ends: the thief left, logged out, started a fight, or a copyover
   or shutdown flushed it. The naming request still starts at the roll. A
   successful roll whose thief left stays "chance lost". Decided here: a
   mark that is gone or dead catches nobody (nobody felt the hand); a mark
   that moved still catches (it felt the hand). Beside the mark it is the
   ordinary catch in the act; anywhere else, or offline, the mark cries
   thief in its own room and the crime is recorded against the thief, but
   nobody is attacked, since the mark cannot reach them. An online thief
   is told and trained on the loss.
2. **Player-key text (Tasks 7, 9a, 9b, 10).** With a server key and
   moderation available, player-key text is moderated as S3 plans and is
   everyone's. Where moderation is impossible the text is FINDER-ONLY
   (`Record.KeptToFinder`, derived as `PlayerKey && !Moderated`, so records
   the pre-slice `moderate` accepted unmoderated are covered too): the
   finder reads it, everyone else the generic
   trinket. Decided here: "impossible" is no server key, `ModerateOutput`
   off, or the provider breaker or the moderation breaker open
   (`moderationPossible`). A check that IS made and fails still refuses
   the find (S3). Every check made, on either route, is recorded on the
   moderation check's OWN breaker (`moderationBreaker`, via
   `apiframework.RecordConsumer`), never the provider's and never the
   naming breaker, so a run of failures turns later player-key finds
   finder-only instead of refusing them. The allowlist, the
   value re-roll, the `RecentNames` exclusion and never-promotable (slice
   C promotes only `PlayerKey` false and `Moderated` true) all still apply.
   **Per-viewer naming is inverted, not routed:** the owner's brief was to
   route every non-finder render site through a viewer-aware accessor; the
   count above (about 156 room sends plus templates, GMCP, shops, auctions)
   makes that neither contained nor safe, since one missed site leaks. So
   the catalog's viewer-agnostic view of a finder-only record IS the
   generic trinket, and ten single-reader functions ask for the finder's
   view (`GetSpecFor`, `DisplayNameFor`, `NameFor`, `LongDescriptionFor`,
   `Record.MaterialFor`). The guard the owner asked for therefore takes two
   halves: `TestFinderOnlyBaubleIsGenericToEveryoneButItsFinder` (items)
   proves every viewer-agnostic accessor shows the generic trinket, so no
   render path, old or new, can use them to leak; and the root guard
   `TestFinderViewReachesOnlyItsReader` fails when a finder-view call
   appears outside the listed functions, when a listed function's call
   count changes, or when one sits inside anything that sends beyond its
   one reader. Accepted limits: the finder's action echoes
   (get, drop, give, sell, put) print the generic name; a stranger who
   guesses a word of a finder-only name can match the trinket with it
   (matching shows no text; the finder needs it to type what they read).
3. **Household guard (Task 14a).** The refusal moves into
   `actions.GetItemFromFloor` (`ErrHouseholdBauble`), so mobs, companions
   and scavengers obey it; the player-only peek is deleted; `get all` is
   unchanged. The function keeps one early-return gate block, ready for the
   parity session's darkness gate.
4. **Companion theft (Task 14b).** Decided: refuse EVERY companion, the
   thief's own included, charmed (`IsCharmed`, the predicate
   `mobs.CheckPlayerHarm` uses) or bonded to the AI companion
   (`companionai.IsBondedCompanion`). `CheckPlayerHarm` itself refuses any
   player's companion "not only the actor's own", and stealing from one's
   own companion gains nothing a `give` or order does not; an owner check
   would add a branch for no play value. It applies to mob thieves too, as
   the non-combatant and attack-immune refusals beside it already do.
5. **Point 8 (the pickpocket reveal on a `NewTurn` listener): SKIPPED.**
   Not small or contained: it replaces the pause scheduler (a goroutine per
   attempt with `time.After`, then `util.LockMud`) with listener-driven
   deadline bookkeeping, and reshapes `FlushPocketAttempts`, the
   `runPocketAttempt`/`paused`/`waitSettled` test harness that ten
   pickpocket tests run on, and the reveal timing. It needs its own slice.

### Tasks this amendment adds or changes

- Task 0: no wait for #175 or slice M (both merged); anchors refreshed and extended.
- Task 4: the rows go after `Modules.aicompanion.ModerationModel`.
- Task 7: `GenResult.FinderOnly`; `Generate` accepts unmoderated player-key text only as finder-only.
- Task 9a (new): `Record.KeptToFinder` (derived), its generic view and `MaterialFor`; the item layer's viewer-aware accessors and finder matching; baubles never enchanted.
- Task 9b (new): the ten single-reader sites (four of them per-user GMCP payloads) and the root finder-view guard, which pins a call count per function and fails on any finder-view call inside a room send, a `Command`, `merchantSay`, an `events.Message` or a `SendText` to anyone but the viewer.
- Task 9c (new): the AI companion tells its model a player-key bauble as its carrier (`ModelName`, `ModelDescription`).
- Task 9d (new): `bauble show` prints player key, moderated and finder only; `bauble list` filters.
- Task 10: the policy above replaces `playerRouteOpen`; moderation has its own breaker (`moderationBreaker`, fed on both routes by `apiframework.RecordConsumer`).
- Tasks 6, 11: `-race` in the test container.
- Task 14: `ownerRecognizes` row re-keyed to the new text.
- Tasks 14a, 14b, 14c (new): household guard (with the AI companion refused up front), companion theft, pickpocket walk-out (with an away mode for witnesses).
- Delivery: three PRs, H1 (Tasks 0 to 5, docs 5a, gate 5b), H2 (H2-0, 6 to 11, docs 11a, gate 11b), H3 (H3-0, 12 to 14c, docs 14d, gate 14e); the single docs Task 15 and gate Task 16 are gone, split into those six.

### Review fixes folded in (blind review of `7c90163af`, verified against source first)

| Finding | Verified | Where fixed |
|---|---|---|
| a: records the pre-slice `moderate` accepted unmoderated stay public | true: `generate.go:248-276` accepted them with `Moderated` false; a stored flag would never cover them | 9a: finder-only is DERIVED (`Record.KeptToFinder` = `PlayerKey && !Moderated`), no stored field; old-record case in `TestFinderOnlyRecordView` |
| b: an away catch finds bystanders in the mark's room as witnesses, with `RecordMet` | true: `steal.go:638, 664`; `crimes.go:229-263` | 14c: `theftWitnesses` away mode (the mark alone, no external witness, no meeting), `theftCrime(..., away)`, crime recorded in the theft's room; `TestTheftWitnessesAwayAreTheMarkAlone` with a real faction and a second same-faction mob |
| c: guard gaps (other `SendText` receivers, `merchantSay`, `Command`, a function-level allowlist) | true: 13 `xxxRoom.SendText` sites, `who.room.SendText`, `offer.go:59, 79`, `look.go:95` | 9b: guard rewritten (per-function call counts; beyond-reader calls; `SendText` receiver must root at the viewer) and its probe covers each shape |
| d: `RecordConsumer` on `ConsumerBaubles` never opens (naming success resets it) | true: `generate.go:73-77` reports before moderation; `breaker.go:100-101` resets | 10: `moderationBreaker`, both routes, checked in `moderationPossible` and `moderate`; `TestFailedChecksOpenTheModerationBreakerAlone` (server, player, server) |
| e: admin views and per-user GMCP | true: `admin.bauble.go:170-223`; `gmcp.Room.go:257`; `gmcp.Char.go:994, 1016` | 9d; 9b (three more GMCP sites) |
| f: AI companion prompts carry item names | true, at the six sites named and three more (`actions.go:508, 624`; `scene.go:278, 288`) | 9c |
| g: boot path and log line | true: dogmud-shipping `SKILL.md:150-178`; `main.go:1549` | 5b, 11b, 14e Step 6 |
| h: steal help | true: `steal.template:26-32` says walking away loses the chance | 14d |
| i: guards in 14b and 14c | partly: `TestEveryTextSurfaceIsRegistered` walks YAML keys only; the Go-line guard is `TestNarrationSitesMatchViewpointAudit`, whose candidates need an actor line plus exactly one of actee or observer (`:1106-1124`) | 14b Step 4 and 14c Step 5 run both; rows prepared for every line the guard can report (14b's refusal is actor-only and cannot be one) |
| LOW | each checked | typed-word match noted in `internal/items/context.md` (11a); baubles never enchanted (9a Step 8b; `enchantments.go:203`); companion get refused up front (14a Step 7b); file map test names; 14b cooldown noted (`steal.go:136`) and Step 2 claim corrected; 14c keeps `roomId`, loads rooms at the reveal, and a roomless online thief is away; `moderate` reads `Server()` and the clock once; `RecordConsumer`'s no-probe behaviour documented; Task 5 Step 2 expected output and the `PlainText` note; Task 11 returns before `m.count` on `errSlotsBusy`; Task 2 replaces the whole comment; Task 13 anchor `:55`; Task 12 case off the threshold; config.yaml moderation comment and `baubles.go:141` reworded |
- Removed: nothing in S1 to S4, U2 or U4 was redundant with FinalTwist's round (lint, catalog prune and lock, matching, SkillMultiplier, the recognition sight gate, the return window, best offer, honest-to-stolen, fences as shopkeepers); each section's code was re-read unfixed at `3bd6ccaa3` (table above). The Task 0 steps that waited for #175 and slice M are gone.

---

## Facts verified against source (original plan, at `e711ee9de`)

Superseded where the amendment table above says otherwise (the
`hardLocked`/`DisplayConfigData` row: both now exist; the stolen-bauble
observer row: re-keyed). Code read at `e711ee9de` (PR #175's head; the planning worktree
`C:\tmp\pr175-ours` is rebased onto it and its later commits are docs only)
unless marked master (`8c6561c5a`). Line numbers WILL shift again:
FinalTwist's pending fix round lands on `modules/baubles/generate.go`
(`name`, `viaServer`, `viaPlayer`), the `FindOpts` blocks in
`search_bauble.go` and `search_feature.go`, `steal_pocket.go` and
`sight_penalty_guard_test.go`, and slice M lands before this slice starts.
Every edit below is keyed by quoted text, not by line, and Task 0 Step 4
greps every quoted anchor against the merged code before Task 1.

| Fact | Where |
|---|---|
| `APIFramework.APIKey` is `ConfigString`; `Validate` rebuilds it as `ConfigString` | `internal/configs/config.apiframework.go:25, 54` |
| `ConfigSecret.String()` returns `*** REDACTED ***`; `ConfigSecret` is a `string` type, so `string(c.APIKey)` in `resolveServer` compiles unchanged | `internal/configs/config_types.go:11, 95-97`; `internal/apiframework/settings.go:216` |
| `configs.GetAPIFrameworkConfig` is read only by `apiframework.RefreshServer` | `settings.go:198`, `config.apiframework.go:67` |
| `Config.AllConfigData(excludeStrings ...string) map[string]any` is a value-receiver method; `Modules` is `map[string]any` | `internal/configs/configs.go:352`, `config.modules.go:3` |
| `hardLocked` and `DisplayConfigData` do NOT exist on this branch or on master yet. Slice M's plan (written beside this one, `docs/superpowers/plans/2026-09-28-slice-m-config-locks-redaction.md`) creates `internal/configs/config_locks.go` with `var hardLocked = []string{...}` (the four `APIFramework.*` paths included, no `Modules.baubles.*`), `isHardLocked(configPath string) bool`, `IsLocked(configPath string) bool`, and `internal/configs/config_display.go` with `func (c Config) DisplayConfigData(excludeStrings ...string) map[string]any`, which redacts any `ConfigSecret` and any leaf named `apikey`, `secret` or `password` | `git grep` on `HEAD` and `master`, both empty; slice M plan lines 127-128, 510-545, 1136 |
| `EndpointAllowed` (exported, not `endpointAllowed`) accepts `api.openai.com`, any `*.openai.com`, any `*.azure.com` | `internal/apiframework/settings.go:20-33` |
| `TestEndpointAllowed` asserts `x.openai.azure.com` allowed and three bad hosts refused | `internal/apiframework/apiframework_test.go:470-482` |
| `DecodeChat` keeps up to 300 bytes of a non-200 body as `StatusError.Detail`, which `Error()` prints | `internal/apiframework/wire.go:162-171`, `breaker.go:172-182` |
| `DecodeChat` callers: companion `decodeChatResponse`, bauble `viaPlayer`, bauble `viaServer` | `modules/aicompanion/openai.go:310`, `modules/baubles/generate.go:157, 201` |
| On the relay route `viaPlayer` never calls `DecodeChat` for a non-200: it builds `StatusError{Status}` with no Detail | `modules/baubles/generate.go:149-153` |
| `Charged(reported int, sent bool, status int, prompt int, maxTokens int, relayed bool) (tokens int, estimated bool)` clamps a relayed count to `prompt+maxTokens` | `internal/apiframework/wire.go:229-241` |
| `viaPlayer` returns `reply.Tokens` raw | `modules/baubles/generate.go:157-158` |
| `generate` takes one shared slot for the whole find (model call AND moderation) before choosing a route | `modules/baubles/generate.go:36-44` |
| `m.slots` is `chan struct{}`, rebuilt by `configure` with `cfg.MaxConcurrent` | `modules/baubles/baubles.go:35, 75` |
| `name(ctx, cfg, req, chat) (content string, tokens int, model string, playerKey bool, report func(error), err error)`; the route choice is the `if cfg.UsePlayerKeys && req.FinderUserId > 0` at line 108 | `modules/baubles/generate.go:103-124` |
| `generate` discards `CleanReply`'s cleaned reply (`_, err = baubles.CleanReply(reply)`) and moderates the RAW reply | `modules/baubles/generate.go:73-82` |
| `moderate` sends `[]string{reply.Name, reply.Description}`; `unavailable` accepts a player-key reply unmoderated when there is no server key, the breaker is open, or the call fails | `modules/baubles/generate.go:248-276` |
| `apiframework.Moderate` errors when the result count differs from the input count | `internal/apiframework/transport.go:157-159` |
| The module test fake answers `/moderations` with exactly TWO results, whatever it was sent | `modules/baubles/baubles_test.go:76-81` |
| `testModule` sets `ModerateOutput = false` | `modules/baubles/baubles_test.go:109-127` |
| `apiframework.SetBreakerForTest(consecutive int, until time.Time)` opens the shared provider breaker | `internal/apiframework/breaker.go:374` |
| `cleanLine` maps `unicode.IsControl` to space, collapses ASCII `\s` (`whitespaceRE`); name, description and material lengths are `len()` bytes | `internal/baubles/validate.go:31, 60-71, 87, 93, 96` |
| `CleanReply` error messages print the whole offending name with `%q` | `internal/baubles/validate.go:88, 91` |
| `baubles.Generate` logs `CleanReply` errors at Warn | `internal/baubles/generate.go:136-139` |
| `authoredKeyword` is a package var over `items.AuthoredKeyword`, swapped by one test | `internal/baubles/validate.go:125`, `generate_test.go:204-207` |
| `authoredWords atomic.Pointer[map[string]bool]`, built by `rebuildAuthoredKeywords`, which skips `BaubleItemId`; seven call sites rebuild it | `internal/items/itemspec.go:536-575`; `itemspec.go:854, 880`, `newitemfile.go:45`, `save.go:66, 81`, `test_helpers.go:9, 12` |
| `golang.org/x/text v0.36.0` is a DIRECT require (no `// indirect`), `go.sum` has its `h1:` line, `unicode/norm` is in the module cache | `go.mod:8`, `go.sum:38-39` |
| `Mint` builds `limited := ApplyLimitsFor(res.Reply, tier, source)`; `ValueProposed` is the reply's own value | `internal/baubles/mint.go:86-95` |
| `ValueTier.RollValue(randn func(n int) int) int`; nil randn gives the midpoint | `internal/baubles/tiers.go:107-113` |
| `RecentNames` filters `r.Zone == zone && r.Generator == GeneratorOpenAI` | `internal/baubles/generate.go:157-176` |
| `BaubleRequestForRecord` always appends `rec.Name` to `RecentNames` | `internal/actions/bauble_admin.go:62` |
| `ApplyRegenerated` sets `Moderated` from the result, never `PlayerKey` | `internal/baubles/admin.go:207-239` |
| `GenResult.PlayerKey` and `Record.PlayerKey` (`yaml:"player_key,omitempty"`) exist | `internal/baubles/generate.go:60`, `record.go:85` |
| `FindOpts` fields: `Place`, `UserId`, `SkillFactor`, `Feature`, `Household`, `Randn`, `Now` | `internal/baubles/find.go:161-186` |
| `RollFind`: `chance := s.chanceFor(...)`, `if chance <= 0` returns WITHOUT spending a window roll, then `takeRoll`, then `rollChance` | `internal/baubles/find.go:214-222` |
| `messaging.SightMult(c *characters.Character, room RoomVisibility) float64`, floors at 0, 1.0 at best; shipped caps `DarknessCombatPenalty: 0.80`, `DazzleCap: 0.80` (Go defaults also 0.80) | `internal/messaging/sight_mult.go:18-35`; `config.yaml` HEAD blob `:925, :931`; `config.balance.combat.go:358-359` |
| `var searchBaubleRoll = func(o baubles.FindOpts) (baubles.ValueTier, bool) { return baubles.RollFind(o) }`; its only production callers are `searchForBauble` and `searchFeatureForBauble` | `internal/actions/search_bauble.go:55-57, 521-526`; `search_feature.go:169-175` |
| `baubles.RollFind` has exactly one production caller, the `searchBaubleRoll` initialiser | `grep -rn "RollFind" --include=*.go`, non-test |
| Sight guard: `guardedSightFuncs` keyed by PACKAGE NAME; a package-level initialiser is checked with `complies=false`, so it is reported ONLY if it contains a call to a guarded name; a stale `sightExemptSites` row fails the test | `sight_penalty_guard_test.go:66-71, 94, 276-294, 391-409` |
| Package names: only `internal/actions` is `package actions`; both `internal/baubles` and `modules/baubles` are `package baubles`; neither calls a bare `RollFind` in production | `grep -rl "^package ..."` |
| `SendTextVisualHidingNames(cat messaging.Category, txt string, names []string, excludeUserIds ...int)`; callers pass the parties' plain names, e.g. `[]string{user.Character.Name}` | `internal/rooms/rooms.go:276`; `internal/hooks/NewRound_DoCombat_helpers.go:919`, `internal/usercommands/target.go:209` |
| `RenderForRecipient` runs `Anonymize` on every `SightShapes` visual line; `Anonymize` replaces `<ansi fg="username">NAME</ansi>` with "a figure" | `internal/messaging/pipeline.go:60-67`, `anonymize.go:21-60` |
| Worktree observer lines of the U4 shape: `search.go:164, 170`; `look.go:67, 95, 122, 287, 333, 412, 437, 460, 525, 567, 572`. Every one wraps the actor in `<ansi fg="username">%s</ansi>` | `internal/actions/search.go`, `internal/usercommands/look.go` |
| Master has the same lines at `search.go:158` and `look.go:66, 100, 127, 295, 341, 420, 446, 469, 542, 547` (no feature branch, no floor-item line) | `git show master:...` |
| The messaging surface guard treats `SendTextVisualHidingNames` on a `room` receiver as an observer, same as `SendTextVisual` | `messaging_surface_guard_test.go:896-914` |
| `TestNoTestPrintsAKey` flags, in `internal/apiframework` and `modules/baubles` tests, any `t.Error*`/`t.Fatal*`/`t.Log*`/`t.Skip*` message whose text contains "key" (case-insensitive, so "keys" and "player-key" count) and has a `%q` or `%s` verb ANYWHERE in it, whatever that verb prints; and, in those two plus `modules/aicompanion`, any print argument containing `APIKey` and the like. Each finding is a `t.Error` at `:102`. So a message mentioning a key uses `%v` or `%d` only | `internal/apiframework/key_guard_test.go:31-35, 66-76, 86-110` |
| `config.yaml` route header is HEAD blob `:2644-2647`, the `MaxConcurrent` comment `:2658`, the player-key moderation comment `:2659-2661`; `ModerationModel` is not in `config.yaml` at all (the module defaults it). A fresh `git worktree add` has its own index, so the skip-worktree bit is NOT set there (`git ls-files -v` prints `H`) | HEAD blob at `e711ee9de` |
| `PromptVersion` is `const PromptVersion = 4` with a version history comment above it; no test pins its value (`baubles_test.go:160` compares against the constant; `pickpocket_test.go:75` and others set their own literal in a fake `GenResult`). `systemPrompt` is a `strings.Join` of lines, the last `` `Do not repeat or closely copy any name in avoid_names.`, `` | `modules/baubles/prompt.go:13-22, 41-53` |
| `Reply` fields `Name`, `NameSimple`, `Description`, `Material`, `WeightLbs`, `Value`. After `CleanReply`, `NameSimple` always matches `nameSimpleRE` (`^[a-z]{2,20}$`) or is a word of the name that does, or `trinket` | `internal/baubles/reply.go:16-23`; `validate.go:32, 106-121` |
| NFKC folds U+2026 to `...` but leaves U+2018, U+2019, U+201C, U+201D, U+2013, U+2014 and U+3002 unchanged (checked with `norm.NFKC.String` from `golang.org/x/text v0.36.0`) | probe run on 2026-09-28 |
| `Relay.Result(userId, err)` only feeds the player's finds breaker (`relayFor.Result` calls `findsResult`); nothing else needs one Result per Send, so not calling it records nothing | `internal/apiframework/relay.go:35-36`; `modules/aicompanion/relayfor.go:49-62` |
| Module config keys are read by name `get(`Model`)`, `MaxCompletionTokens`, `MaxConcurrent`, `UsePlayerKeys`, `ModerateOutput`, `ModerationModel` | `modules/baubles/config.go:121-146` |
| Stolen-bauble observer lines: `stolen_bauble.go:234` (`ownerRecognizes`: "points at `<ansi fg="username">`carrier"), `:294` (the owner "looks overjoyed", names only the mob and the item); `steal.go:801` (`stealHouseholdBauble`, "is caught trying to pocket", PR code, the actor in a username tag); `steal.go:364, 537` are master's crime lines (master `steal.go:303, 583` at `8c6561c5a`) | `internal/actions/stolen_bauble.go`, `internal/actions/steal.go`; `git show master:internal/actions/steal.go` |
| `docs/aicompanion/settings.md:30` (endpoint comment) and `:341-346` (bauble relay paragraph) describe the old rules; `docs/baubles/implementation-plan.md:598-600` says player-key finds are accepted unmoderated | as listed |

### Existing tests each change touches

| Changed function | Existing tests | Must change? |
|---|---|---|
| `configs.APIFramework.Validate` | none direct; `apiframework_test.go:490-600` builds `configs.APIFramework{APIKey: ...}` with untyped constants | No (constants convert to `ConfigSecret`) |
| `EndpointAllowed` | `TestEndpointAllowed` | Extend with the new refusals |
| `DecodeChat` | `TestDecodeChat`, `status()` helper, `ProviderFailure` tests (`apiframework_test.go:58-80, 281-306`) | No; new test added |
| `cleanLine`, `CleanReply` | `TestCleanReply`, `TestCleanReplyKeywords`, `TestCleanReplyKeepsOffLoadedItemsKeywords`, `TestKeywordFallbackAvoidsRealItemsNameWords`, `TestPickpocketFindsArePocketSized` | No; new table tests added |
| `rebuildAuthoredKeywords`, `AuthoredKeyword` | `TestAuthoredKeyword`, `TestAuthoredKeywordIsSafeWhileItemsAreWritten` (`internal/items/bauble_placement_test.go:45-104`) | Extend the race test to read `AuthoredName` too |
| `baubles.Generate` | `generate_test.go:30-89` | No; new cases added |
| `Mint` | `TestMintFromAModelResult`, `TestMintMakesAUniqueCatalogBackedItem`, `TestMintHoldsAPickpocketFindToThePocket` | No |
| `RecentNames` | `TestRecentNames` | No; new test added |
| `ApplyRegenerated` | `TestApplyRegenerated` | No; new assertion in a new test |
| `BaubleRequestForRecord` | `TestRegenerateBauble` (`internal/actions/bauble_admin_test.go:23`) | No; new test added |
| `modules/baubles` `generate`, `name`, `viaPlayer`, `moderate` | every test in `modules/baubles/baubles_test.go` that uses a relay or moderation | YES: `TestModerationOutageNeverSpoilsAPlayerKeyFind` inverts and is renamed; `TestNoKeyAtAll` second half inverts; `TestFindersOwnKeyNamesTheirFind`, `TestFindersKeyOnlyWhenAllowed`, `TestFindersKeyFailingFallsBackToTheServer`, `TestFindersUnusableReplyIsReportedForFindsOnly`, `TestPickpocketUsesTheThiefsKeyFirst` must turn `ModerateOutput` on; the fake's `/moderations` must answer one result per input |
| `RollFind` | `TestRollFindWithinTheWindow`, `TestRollFindUsesTheRoomAndTheSearcher`, `TestRollFindRespectsTheSwitchAndExcludedZones`, `TestFeatureSearchedOncePerWindow` | No (zero `SightPenalty` is none) |
| `searchForBauble`, `searchFeatureForBauble` | `search_bauble_test.go`, `search_feature_test.go` (they stub `searchBaubleRoll`) | No |
| sight guard maps | `TestEveryRollSiteAppliesTheSightPenalty`, `TestSightPenaltyGuardCatchesAnOmission`, `TestSightPenaltyGuardSeesTheSpellAlias` | No; new probe test added |
| look/search observer sends, `ownerRecognizes`, `stealHouseholdBauble` | messaging surface guard; `internal/usercommands` look tests; `internal/actions` steal and stolen-bauble tests | No |
| `PromptVersion`, `systemPrompt` | the module test at `baubles_test.go:160` compares `res.PromptVersion` with the constant | No; new test added |
| `DisplayName`, `GetLongDescription`, `NameMatch`, `matchStrength`, `baubleSpec` (Task 9a) | every items test; `internal/items/bauble_test.go` (matching) | No: for any item that is not finder-only each returns what it did |
| `Record.View`, `Mint`, `ApplyRegenerated` (Task 9a) | `TestMint*`, `TestApplyRegenerated`, the retire tests | No |
| `moderate` signature (Task 10) | its only caller is `generate` | No other caller |
| `GetItemFromFloor` (Task 14a) | `TestGetItemFromFloor_Happy`, `_NotFound` (`economy_test.go:283, 306`); `usercommands` household tests | No |
| `stealFromMob` (Task 14b) | `steal_test.go` (no companion mob in any of them) | No |
| `resolve`, `thiefCaught` (Task 14c) | `TestPickpocketAwardsAndCatchesAtTheReveal` second half pins "walked off after a failed roll: not caught" | YES: inverted in Task 14c |

### Spec statements found false or imprecise against source

1. **U2 exemption.** The spec says to add `"actions": {"searchBaubleRoll": true}` and exempt the `var searchBaubleRoll` initialiser. With only that entry the initialiser contains no call to a guarded name (it calls `baubles.RollFind`, which is not guarded), so it is never reported, and an exemption row for it is STALE: `TestEveryRollSiteAppliesTheSightPenalty` fails on a stale row. The exemption is correct only if `baubles.RollFind` is ALSO guarded. This plan guards both, which also catches a future caller that bypasses the seam.
2. **U4 premise.** The spec calls these "bare-name" lines. None is: every one wraps the actor in `<ansi fg="username">`, which the pipeline's `Anonymize` already turns into "a figure" for a shapes-only reader. The switch to `SendTextVisualHidingNames` changes no output today; it is defence in depth. The same shape sits at seven more `look.go` lines the spec did not list (`67, 95, 122, 287, 333, 412, 437`); this plan moves all of them so `look.go` stays consistent (CLAUDE.md: finish sibling paths you made inconsistent).
3. **"Master" line numbers.** `look.go 460/567/572` and `search.go 170` are this worktree's numbers for lines that also exist on master; on master itself they are `469/542/547` and `158`.
4. **S2 "on both server and relay routes".** For baubles the relay route never keeps a provider body (it builds `StatusError` with no Detail); the scrub in `DecodeChat` covers the server route and the companion's relay path. The scrub lives in `DecodeChat`, so every caller is covered either way.
5. **S2 name.** The function is `EndpointAllowed`, exported.
6. **U2 "before the window roll".** Applying the penalty before `RollFind`'s `chance <= 0` guard would let a full penalty skip the window roll, the opposite of the spec's intent. The plan applies it after that guard (which is still after `chanceFor` and before `takeRoll`).
7. **The allowlist gains `"` and ellipses (ruling 15).** Ruling 15 folds U+201C and U+201D to `"` and U+2026 to `...` before the check. With the spec's set (`' - , . ! ?`, a period only before a space or at the end) both folds would only turn one refusal into another. The plan adds `"`: it is plain ASCII, carries no markup once `cleanLine` has stripped tags and `<` `>`, cannot form a link, and a quoted inscription ("To Mara") is ordinary bauble text; `NameSimple`, the only field a player types, stays `[a-z]`. The period rule becomes "every run of periods is followed by a space, a `"`, or the end", so `horse... its` and `"Mine."` pass while `horse.Its` and `a...b` are refused. `:` and `;` stay refused; the system prompt now says so (PromptVersion 5).
8. **Where the module checks the allowlist (ruling 15).** An allowlist refusal must fall back to the server route and must not feed the player's breaker, but `generate` runs `report(err)` on every parse or clean failure after `name` has already chosen the route. The check therefore runs inside `name`'s player branch (`refusedByAllowlist`), before the route is final; `generate` does not repeat it, and the engine's `Generate` (Task 7) stays the backstop for any generator.

---

## File map

| File | Change | Responsibility |
|---|---|---|
| `internal/configs/config.apiframework.go` | Modify | `APIKey` becomes `ConfigSecret` |
| `internal/configs/config.apiframework_test.go` | Create | S1 Validate test; S2 hard-lock membership test |
| `internal/configs/config_locks.go` (slice M's) | Modify | S2 hard-locked baubles paths |
| `internal/apiframework/key_guard_test.go` | Modify | S1 display-path cases |
| `internal/apiframework/settings.go` | Modify | S2 exact-host allowlist |
| `internal/apiframework/wire.go` | Modify | S2 key scrub in `DecodeChat` |
| `internal/apiframework/apiframework_test.go` | Modify | S2 tests |
| `internal/items/itemspec.go` | Modify | S3 `AuthoredName` in the same snapshot |
| `internal/items/bauble_placement_test.go` | Modify | `AuthoredName` tests |
| `internal/baubles/validate.go` | Modify | S4 cleaning, rune lengths, links, quoting; S3 authored-name refusal |
| `internal/baubles/validate_test.go` | Create | S4 tables |
| `internal/baubles/playerkey.go` | Create | S3 `CheckPlayerKeyText` |
| `internal/baubles/playerkey_test.go` | Create | allowlist tests |
| `internal/baubles/generate.go` | Modify | engine player-key invariants; `RecentNames` skip |
| `internal/baubles/generate_test.go` | Modify | engine tests |
| `internal/baubles/mint.go` | Modify | player-key value roll |
| `internal/baubles/admin.go` | Modify | `ApplyRegenerated` sets `PlayerKey` |
| `internal/baubles/admin_test.go` | Modify | test |
| `internal/baubles/find.go` | Modify | `SightPenalty` |
| `internal/baubles/find_test.go` | Modify | test |
| `internal/actions/bauble_admin.go` | Modify | omit a player-key name |
| `internal/actions/bauble_admin_test.go` | Modify | test |
| `internal/actions/search_bauble.go`, `search_feature.go` | Modify | pass `SightPenalty` |
| `internal/actions/search_bauble_sight_test.go` | Create | U2 caller tests |
| `sight_penalty_guard_test.go` | Modify | guard entries, exemption, probe |
| `modules/baubles/generate.go` | Modify | pre-check, slots, moderation policy, tokens |
| `modules/baubles/baubles.go` | Modify | per-finder slot map, `info` text |
| `modules/baubles/baubles_test.go` | Modify | fake and test rewrites, new tests |
| `modules/baubles/prompt.go` | Modify | the allowed characters in the system prompt; `PromptVersion` 5 |
| `internal/actions/search.go`, `internal/usercommands/look.go` | Modify | U4 |
| `internal/actions/stolen_bauble.go`, `internal/actions/steal.go` | Modify | U4 siblings: the two bauble theft observer lines that name a player |
| `internal/rooms/participant_sight_test.go` | Modify | U4 characterization test |
| `internal/baubles/record.go`, `internal/baubles/fallback.go` | Modify | Task 9a: `FinderOnly`, the generic view, `MaterialFor`, `genericDescriptionFor` |
| `internal/baubles/record_test.go` | Create | Task 9a tests |
| `internal/items/bauble.go`, `internal/items/items.go` | Modify | Task 9a: `BaubleView.Finder`, `baubleSpecFor`, finder matching, `displayNameFrom`, `longDescriptionFrom` |
| `internal/items/bauble_viewer.go`, `internal/items/bauble_viewer_test.go` | Create | Task 9a: `GetSpecFor`, `DisplayNameFor`, `NameFor`, `LongDescriptionFor` |
| `internal/usercommands/inventory.go`, `look.go`, `appraise.go`, `modules/gmcp/gmcp.Char.go` | Modify | Task 9b: the finder's own view |
| `bauble_finder_view_guard_test.go` | Create | Task 9b: root guard |
| `internal/apiframework/breaker.go` | Modify | Task 10: `RecordConsumer` |
| `internal/actions/get.go`, `internal/usercommands/get.go` | Modify | Task 14a: `ErrHouseholdBauble` |
| `internal/actions/get_household_test.go`, `internal/usercommands/get_household_test.go` | Create | Task 14a tests |
| `modules/aicompanion/actions.go`, `modules/aicompanion/harm_test.go` | Modify | Task 14a: the companion is refused a household's bauble up front |
| `modules/aicompanion/perception.go`, `scene.go`, `actions.go`; `internal/items/bauble_model.go`; `modules/aicompanion/bauble_prompt_test.go` | Modify / Create | Task 9c |
| `internal/usercommands/admin.bauble.go`, `admin.bauble_test.go`, `command.bauble.template` | Modify | Task 9d |
| `internal/enchantments/enchantments.go`, `internal/enchantments/bauble_test.go` | Modify / Create | Task 9a: no bauble is enchanted |
| `modules/gmcp/gmcp.Room.go` | Modify | Task 9b |
| `_datafiles/world/dogmud/templates/help/steal.template` | Modify | PR H3 docs |
| `internal/actions/steal_crime_test.go` | Create | Task 14c: witnesses away from the theft |
| `internal/actions/steal.go`, `internal/actions/steal_test.go` | Modify | Task 14b (companion refusal); Task 14c (`theftCrime`) |
| `internal/actions/steal_pocket.go`, `internal/actions/pickpocket_test.go` | Modify | Task 14c |
| docs (`context.md` in every touched package, `docs/aicompanion/settings.md`, `docs/baubles/implementation-plan.md`) | Modify | Tasks 5a, 11a, 14d, one share per PR |
| `docs/README.md` | Modify | this plan's row (the amendment commit) |

New non-code files: none (every created file is Go). This plan is
already indexed in `docs/README.md`; the amendment commit updated its row.

Every commit ends with:

```
Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
```

Use a heredoc for every message (bash command-substitutes backticks inside `-m`). Stage named paths only; never `git add -A` or `git add .`.

---

## Delivery: three PRs, in order (review, 2026-09-29)

Slice H ships as three PRs, each with its own share of the docs and its
own local gate, each branched from `origin/master` after the one before it
has merged. Smaller PRs keep each review and each CI-less local gate to
one concern.

| PR | Branch | Tasks | Content |
|---|---|---|---|
| H1 | `fix/baubles-hardening-h` (this branch; its first commit is this plan) | 0, 1 to 5, 5a docs, 5b gate | S1 key secrecy, S2 endpoints, error-text scrub and hard locks, S4 output cleaning |
| H2 | `fix/baubles-hardening-h2`, from `origin/master` after H1 merges | H2-0, 6 to 11 (with 9a to 9d), 11a docs, 11b gate | S3 and finder-only text, with review fixes a, c, d, e, f |
| H3 | `fix/baubles-hardening-h3`, from `origin/master` after H2 merges | H3-0, 12 to 14c, 14d docs, 14e gate | U2, U4, the household guard, companion theft, the pickpocket walk-out, with review fixes b, h, i |

All three reuse the worktree `C:\tmp\dogmud-baubles-h`, so every command
below stays as written; each PR's first task switches it to that PR's
branch and records that PR's base in `/c/tmp/dogmud-baubles-h.base`. The
owner (or the coordinator on the owner's word) pushes, opens and merges
each PR; a gate task ends by handing the branch over, never by pushing.

## PR H1: S1, S2, S4

---

### Task 0: Preconditions

**Files:** none in the repo. Records `/c/tmp/dogmud-baubles-h.base`.

Delivery (owner ruling 11): PR #175 and slice M (#179) are both on master.
The branch `fix/baubles-hardening-h` was created from `origin/master` at
`3bd6ccaa3` by the planning session, in the worktree
`C:\tmp\dogmud-baubles-h`, and this plan is its first commit. Never
rebase onto, commit to or push to FinalTwist's branch.

- [ ] **Step 1: Confirm the worktree and record the base**

Run:
```bash
cd /c/tmp/dogmud-baubles-h && git status --short && git branch --show-current && git fetch -q origin && BASE=$(git merge-base HEAD origin/master) && echo "$BASE" > /c/tmp/dogmud-baubles-h.base && echo "BASE=$BASE" && git log --oneline -1 "$BASE"
```
Expected: a clean tree, `fix/baubles-hardening-h`, and `BASE=3bd6ccaa3...` (`Merge pull request #184`). The merge base stays `3bd6ccaa3` however far `origin/master` moves, because the branch does not contain anything newer. Every later step that needs it runs `BASE=$(cat /c/tmp/dogmud-baubles-h.base)` first (shell state does not persist between calls).

If the worktree is missing, recreate it from the main checkout without changing that checkout's branch: `git -C "/c/Users/Calabe Davis/workspace/DOGMud" worktree add C:/tmp/dogmud-baubles-h fix/baubles-hardening-h`.

- [ ] **Step 2: Confirm slice M's names**

Run:
```bash
cd /c/tmp/dogmud-baubles-h && grep -n "Modules.aicompanion.ModerationModel\|^func isHardLocked\|^func IsLocked" internal/configs/config_locks.go && grep -n "^func (c Config) DisplayConfigData" internal/configs/config_display.go
```
Expected: four lines (`config_locks.go` `:31`, `:39`, `:54`; `config_display.go:47` at `3bd6ccaa3`). Tasks 1 and 4 are written against these names.

- [ ] **Step 3: The local race runner works**

CI is out of minutes until 2026-10-01, so `-race` runs in the Linux test container (Tasks 6, 11 and 16). Build it once now, so a broken Docker setup shows up before any task depends on it. Run (Bash):
```bash
cd /c/tmp/dogmud-baubles-h && docker compose -f compose.test.yml build test 2>&1 | tail -3
```
Expected: the build finishes (its last lines name the image). If the Docker daemon is down, start Docker Desktop and rerun; if it cannot run at all, STOP and report: the gate cannot be run locally without it.

- [ ] **Step 4: Every quoted anchor this plan edits exists in the merged code**

Checked against `3bd6ccaa3` on 2026-09-29: every anchor below is present. Rerun it in case master or the branch moved. Run:
```bash
cd /c/tmp/dogmud-baubles-h && anchors=$(mktemp) && cat > "$anchors" <<'EOF'
internal/configs/config.apiframework.go|APIKey ConfigString `yaml:"APIKey"`
internal/configs/config.apiframework.go|a.APIKey = ConfigString(strings.TrimSpace(string(a.APIKey)))
internal/configs/config.apiframework.go|Never logged.
internal/configs/config_locks.go|Modules.aicompanion.DeepModel
internal/apiframework/settings.go|return host == `api.openai.com` || strings.HasSuffix(host, `.openai.com`) || strings.HasSuffix(host, `.azure.com`)
internal/apiframework/apiframework_test.go|func TestEndpointAllowed(t *testing.T) {
internal/apiframework/wire.go|snippet := strings.TrimSpace(string(raw))
internal/baubles/validate.go|whitespaceRE = regexp.MustCompile(`\s+`)
internal/baubles/validate.go|func cleanLine(s string) string {
internal/baubles/validate.go|if r.Name == `` || len(r.Name) > maxNameLen || len(words) < minNameWords || len(words) > maxNameWords {
internal/baubles/validate.go|if len(r.Material) > maxMaterialLen {
internal/baubles/validate.go|var authoredKeyword = items.AuthoredKeyword
internal/items/itemspec.go|var authoredWords atomic.Pointer[map[string]bool]
internal/items/itemspec.go|func AuthoredKeyword(word string) bool {
internal/items/bauble_placement_test.go|_ = AuthoredKeyword(`lantern`)
internal/items/bauble_placement_test.go|func TestAuthoredKeywordIsSafeWhileItemsAreWritten(
internal/baubles/generate.go|cleaned, err := CleanReply(res.Reply)
internal/baubles/generate.go|if r.Zone == zone && r.Generator == GeneratorOpenAI {
internal/baubles/mint.go|limited := ApplyLimitsFor(res.Reply, tier, source)
internal/baubles/admin.go|r.Moderated = res.Moderated
internal/baubles/find.go|chance := s.chanceFor(o.Place.Biome, o.SkillFactor)
internal/baubles/find.go|SkillFactor float64
internal/baubles/find.go|mudlog.Info(`baubles`, `action`, `found`, `tier`, string(tier), `household`, o.Household, `chancePct`, chance,
internal/actions/bauble_admin.go|req.RecentNames = append(req.RecentNames, rec.Name)
internal/actions/search_bauble.go|var searchBaubleRoll = func(o baubles.FindOpts) (baubles.ValueTier, bool) {
internal/actions/search_bauble.go|SkillFactor: BaubleSkillFactor(actor.GetCharacter()),
internal/actions/search_feature.go|Feature:     feature.WindowName(),
internal/actions/stolen_bauble.go|`<ansi fg="mobname">%s</ansi> points at %s. "That's mine! Thief!"`,
internal/actions/steal.go|is caught trying to pocket the <ansi fg="itemname">%s</ansi>!`,
internal/actions/search.go|is searching the %s.
internal/actions/search.go|is snooping around.
internal/usercommands/look.go|is looking around.
internal/usercommands/look.go|peers toward the %s.
internal/usercommands/look.go|is admiring their
internal/usercommands/look.go|is examining the <ansi fg="noun">%s</ansi>.
internal/usercommands/look.go|is looking into the room from somewhere...
internal/usercommands/look.go|is looking into the room from the <ansi fg="exit">%s</ansi> exit
modules/baubles/generate.go|errBreakerOpen = errors.New(`the server key's breaker is open`)
modules/baubles/generate.go|// A fixed number of calls at once.
modules/baubles/generate.go|_, err = baubles.CleanReply(reply)
modules/baubles/generate.go|if cfg.UsePlayerKeys && req.FinderUserId > 0 {
modules/baubles/generate.go|content, tokens, report, err = viaPlayer(ctx, r, req.FinderUserId, relayModel, chat)
modules/baubles/generate.go|content, tokens, report, err = viaServer(ctx, cfg, chat)
modules/baubles/generate.go|reply := apiframework.DecodeChat(status, raw)
modules/baubles/generate.go|func (m *BaublesModule) moderate(cfg Config, reply baubles.Reply, playerKey bool) (bool, error) {
modules/baubles/baubles.go|slots chan struct{}
modules/baubles/baubles.go|detail += ` No server key: only finders who allowed their own key get named finds.`
modules/baubles/prompt.go|const PromptVersion = 4
modules/baubles/prompt.go|`Do not repeat or closely copy any name in avoid_names.`,
modules/baubles/baubles_test.go|fmt.Fprintf(w, `{"results":[{"flagged":%t},{"flagged":false}]}`, f.flagged)
modules/baubles/baubles_test.go|func TestModerationOutageNeverSpoilsAPlayerKeyFind(
modules/baubles/baubles_test.go|// A finder's own key still names it;
modules/baubles/baubles_test.go|m := testModule(t, serverSide, nil)
modules/baubles/baubles_test.go|m2 := testModule(t, serverSide, func(c *Config) { c.UsePlayerKeys = false })
modules/baubles/baubles_test.go|if !res.PlayerKey || res.Model != `player-model` ||
sight_penalty_guard_test.go|"forager":  {"ForageCore": true},
sight_penalty_guard_test.go|executeCounterTaunt
sight_penalty_guard_test.go|func TestSightPenaltyGuardSeesTheSpellAlias(
_datafiles/config.yaml|# else through the server's key (APIFramework: its one daily budget and
_datafiles/config.yaml|MaxConcurrent: 4           # more finds at once than this are generic
_datafiles/config.yaml|# key of its own is not moderated, as with the AI companion.
docs/aicompanion/settings.md|Off: any other host is refused
docs/aicompanion/settings.md|The key page also has a box, "Also name things I find while searching
docs/baubles/implementation-plan.md|- **Moderation** of a player-key find: a flag refuses; a check that cannot be
internal/configs/config_locks.go|`Modules.aicompanion.ModerationModel`,
internal/items/items.go|func (i *Item) GetLongDescription() string {
internal/items/items.go|if i.GetSpec().QuestToken != `` {
internal/items/items.go|wordPart, wordFull := baubleWordMatch(input, simpleName, displayName, withoutPossessives(i.Name()))
internal/items/bauble.go|func baubleSpec(base ItemSpec, id string) ItemSpec {
internal/items/bauble.go|names := []string{util.NormalizeForMatch(i.Name()), util.NormalizeForMatch(i.NameSimple()), withoutPossessives(i.Name())}
internal/baubles/record.go|PlayerKey     bool      `yaml:"player_key,omitempty"` // named through the finder's own key
internal/baubles/record.go|func (r Record) View() items.BaubleView {
internal/baubles/mint.go|PlayerKey:      res.PlayerKey,
internal/baubles/generate.go|PlayerKey     bool // named through the finder's own key, not the server's
internal/actions/search_bauble.go|name := itm.DisplayName()
internal/actions/steal.go|fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, b.DisplayName()))
internal/actions/steal.go|fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, itemStolen.DisplayName()))
internal/usercommands/inventory.go|iNameFormatted := fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, item.Name())
internal/usercommands/inventory.go|for _, part := range util.BreakIntoParts(item.Name()) {
internal/usercommands/look.go|itemDesc := lookItem.GetLongDescription()
internal/usercommands/look.go|util.SplitStringNL(floorItem.GetLongDescription(), 80),
internal/usercommands/look.go|groundStacks[key] = &groundStack{name: item.DisplayName() + item.BaubleSpotSuffix(), count: 1}
internal/usercommands/look.go|name := item.DisplayName() + ` <ansi fg="item-stashed">(stashed)</ansi>`
internal/usercommands/look.go|fmt.Sprintf(`You look at the <ansi fg="item">%s</ansi> %s:`, lookItem.DisplayName(), lookDestination),
internal/usercommands/look.go|fmt.Sprintf(`You look at the <ansi fg="item">%s</ansi> %s:`, floorItem.DisplayName(), where),
internal/usercommands/appraise.go|if rec.Material != `` {
internal/usercommands/appraise.go|spec := item.GetSpec()
internal/usercommands/admin.bauble.go|fmt.Fprintf(&b, "  generator:   %s %s (prompt v%d, %d tokens)\r\n", rec.Generator, rec.Model, rec.PromptVersion, rec.Tokens)
modules/gmcp/gmcp.Char.go|payload.Inventory.Backpack.Items = append(payload.Inventory.Backpack.Items, newInventory_Item(itm))
internal/apiframework/breaker.go|func (k *Books) Release(consumer string, t Ticket) {
internal/actions/get.go|matchItem, found := room.FindOnFloor(itemName, stash)
internal/usercommands/get.go|if peekFound && !getFromStash && peekItem.BaubleBelongsTo(room.RoomId) {
internal/actions/steal.go|// Deliberately NOT mobs.CheckPlayerHarm: that policy also blocks charmed
internal/actions/steal.go|func thiefCaught(actor Actor, m *mobs.Mob, room *rooms.Room) {
internal/actions/steal_pocket.go|if !p.success {
internal/actions/steal_pocket.go|roomId        int
internal/actions/pickpocket_test.go|h2.thief.room = newSearchTestRoom(9698) // walks off
EOF
missing=0; while IFS= read -r line; do f=${line%%|*}; a=${line#*|}; grep -qF -- "$a" "$f" || { echo "MISSING in $f: $a"; missing=1; }; done < "$anchors"; rm -f "$anchors"; echo "missing=$missing"
```
Expected: `missing=0` (every anchor held at `3bd6ccaa3` on 2026-09-29; the original list was also run at `e711ee9de`, where the `config_locks.go` anchor was missing, so the loop is proven able to report a miss). For every `MISSING` line, STOP before the task that edits it: read the code, rewrite that task's quoted old text (and anything depending on it, such as Task 11's replacement of Task 10's block) against it, and report the change. Do not guess at a shape.

Then confirm FinalTwist's round did not already do part of U2:
```bash
cd /c/tmp/dogmud-baubles-h && grep -n "SightPenalty" internal/baubles/find.go internal/actions/search_bauble.go internal/actions/search_feature.go
```
```bash
cd /c/tmp/dogmud-baubles-h && grep -n '"baubles":\|"actions":\|searchBaubleRoll\|RollFind' sight_penalty_guard_test.go
```
Expected: no output from either (each exits 1; both were empty at `e711ee9de`, and the second pattern does match the guard file's own `guardedSightFuncs` rows once Task 13 adds them, so it can succeed). Any hit means the merged code already carries part of Tasks 12 and 13: adapt those tasks to extend it rather than duplicate it.

- [ ] **Step 5: Baseline green**

Run:
```bash
cd /c/tmp/dogmud-baubles-h && go build ./... && go test ./internal/baubles/ ./internal/items/ ./internal/apiframework/ ./internal/configs/ ./modules/baubles/ ./modules/gmcp/ ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ ./internal/rooms/ . 2>&1 | tail -20
```
Expected: every package `ok`. A failure here is pre-existing: record it and do not attribute it to this slice.

---

### Task 1: S1, the server key is a `ConfigSecret`

**Files:**
- Modify: `internal/configs/config.apiframework.go` (the `APIKey` field and the `Validate` line for it)
- Create: `internal/configs/config.apiframework_test.go`
- Modify: `internal/apiframework/key_guard_test.go`

- [ ] **Step 1: Write the failing configs test**

Create `internal/configs/config.apiframework_test.go`:

```go
package configs

import (
	"fmt"
	"strings"
	"testing"
)

// The server's key is a secret wherever the config is printed: a boot log, a
// server listing, /viewconfig (spec S1). Validate keeps it one.
func TestAPIFrameworkKeyIsASecret(t *testing.T) {
	a := APIFramework{APIKey: `  sk-sentinel-s1-0001  `}
	a.Validate()
	if string(a.APIKey) != `sk-sentinel-s1-0001` {
		t.Fatalf("Validate trims the value: got %d bytes", len(string(a.APIKey)))
	}
	if got := fmt.Sprint(a.APIKey); strings.Contains(got, `sentinel`) || got != `*** REDACTED ***` {
		t.Fatalf("printed, the value must be redacted, got %d bytes", len(got))
	}
	if got := fmt.Sprintf(`%v`, a); strings.Contains(got, `sentinel`) {
		t.Fatal("printing the whole section must not show the value")
	}
}
```

Note: failure messages print lengths, never the value, so `TestNoTestPrintsAKey` (which scans only `internal/apiframework`, `modules/baubles`, `modules/aicompanion`) would pass them anyway.

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/configs/ -run TestAPIFrameworkKeyIsASecret -v`
Expected: FAIL with "printed, the value must be redacted".

- [ ] **Step 3: Make the key a secret**

In `internal/configs/config.apiframework.go`, replace

```go
	APIKey ConfigString `yaml:"APIKey"`
```
with
```go
	APIKey ConfigSecret `yaml:"APIKey"`
```

and in `Validate` replace

```go
	a.APIKey = ConfigString(strings.TrimSpace(string(a.APIKey)))
```
with
```go
	a.APIKey = ConfigSecret(strings.TrimSpace(string(a.APIKey)))
```

Update the field comment's last sentence from `Never logged.` to `A ConfigSecret: every config listing prints it redacted. Never logged.`

- [ ] **Step 4: Run it to verify it passes, and that nothing else broke**

Run: `go test ./internal/configs/ -run TestAPIFrameworkKeyIsASecret -v && go build ./... && go test ./internal/apiframework/`
Expected: PASS, build clean, `ok`.

- [ ] **Step 5: Write the display-path key guard cases**

Append to `internal/apiframework/key_guard_test.go` (add `"fmt"` and `"github.com/GoMudEngine/GoMud/internal/configs"` to its imports):

```go
// Neither place a server key can be configured shows through the one
// redacted view of the config (slice M's DisplayConfigData, which the boot
// log, both server listings and /viewconfig render): the typed
// APIFramework.APIKey (a ConfigSecret) and the companion's old module-map
// key (a plain map value, redacted by its leaf name).
func TestNoServerKeyReachesTheConfigDisplay(t *testing.T) {
	var c configs.Config
	c.APIFramework.APIKey = `sk-sentinel-typed-0001`
	c.Modules = configs.Modules{`aicompanion`: map[string]any{`APIKey`: `sk-sentinel-module-0002`}}
	shown := 0
	for path, v := range c.DisplayConfigData() {
		shown++
		if strings.Contains(fmt.Sprint(v), `sk-sentinel`) {
			t.Errorf("the config display leaks the sentinel at %v", path)
		}
	}
	if shown < 10 {
		t.Fatalf("the display walked only %d entries: this test could not have found a leak", shown)
	}
}
```

Slice M's plan declares `func (c Config) DisplayConfigData(excludeStrings ...string) map[string]any`, so the call above takes no arguments.

- [ ] **Step 6: Run the guard and prove it can fail**

Run: `go test ./internal/apiframework/ -run "TestNoServerKeyReachesTheConfigDisplay|TestNoTestPrintsAKey" -v`
Expected: PASS.

Probe: slice M redacts any leaf named `APIKey` as well as any `ConfigSecret`, so reverting Step 3 alone leaves this test green (the configs test in Step 1 is what pins the type). Prove the display test can fail instead: temporarily add `c.APIFramework.APIKeyEnv = `sk-sentinel-env-0003`` to it (`APIKeyEnv` is neither a secret type nor a secret leaf name) and rerun. Expected: FAIL "the config display leaks the sentinel at APIFramework.APIKeyEnv". Remove the line, rerun, PASS.

- [ ] **Step 7: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/configs/config.apiframework.go internal/configs/config.apiframework_test.go internal/apiframework/key_guard_test.go && git commit -F - <<'EOF'
fix(apiframework): the server key is a ConfigSecret

APIFramework.APIKey printed in full through every config listing. It is
now a ConfigSecret, and a guard proves neither key location shows through
the redacted config display.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 2: S2, the key only goes to OpenAI

**Files:**
- Modify: `internal/apiframework/settings.go` (`EndpointAllowed`)
- Modify: `internal/apiframework/apiframework_test.go` (`TestEndpointAllowed`)

- [ ] **Step 1: Extend the failing test**

In `internal/apiframework/apiframework_test.go`, replace the whole of `TestEndpointAllowed` with:

```go
func TestEndpointAllowed(t *testing.T) {
	for _, good := range []string{`https://api.openai.com/v1`, `https://x.openai.azure.com/v1`, `https://API.OpenAI.com/v1`} {
		if !EndpointAllowed(good, false) {
			t.Errorf("%q is OpenAI or Azure OpenAI over https", good)
		}
	}
	// Exactly api.openai.com and *.openai.azure.com (spec S2): any other
	// openai.com or azure.com host is somebody else's server.
	for _, bad := range []string{
		`http://api.openai.com/v1`, `https://evil.example.com/v1`, `notaurl`,
		`https://files.openai.com/v1`, `https://evil.azure.com/v1`, `https://openai.azure.com.evil.example/v1`,
		`https://xopenai.azure.com/v1`, `https://api.openai.com.evil.example/v1`,
		// Azure AI Services hosts need AllowCustomEndpoint (documented).
		`https://x.cognitiveservices.azure.com/v1`, `https://x.services.ai.azure.com/v1`,
	} {
		if EndpointAllowed(bad, false) {
			t.Errorf("%q must be refused without AllowCustomEndpoint", bad)
		}
	}
	if !EndpointAllowed(`https://llm.internal.example/v1`, true) || EndpointAllowed(`http://llm.internal.example/v1`, true) {
		t.Fatal("custom endpoints: https only, and only when allowed")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/apiframework/ -run TestEndpointAllowed -v`
Expected: FAIL naming `https://files.openai.com/v1`, `https://evil.azure.com/v1`, `https://xopenai.azure.com/v1`, `https://x.cognitiveservices.azure.com/v1` and `https://x.services.ai.azure.com/v1` (all five end in `.openai.com` or `.azure.com`, which today's rule accepts).

- [ ] **Step 3: Tighten the allowlist**

In `internal/apiframework/settings.go`, replace

```go
	host := strings.ToLower(u.Hostname())
	return host == `api.openai.com` || strings.HasSuffix(host, `.openai.com`) || strings.HasSuffix(host, `.azure.com`)
```
with
```go
	// Exactly OpenAI's API host, or an Azure OpenAI resource
	// (<resource>.openai.azure.com). Any other openai.com or azure.com host
	// is not where the key belongs. Azure's newer AI Services hosts
	// (*.cognitiveservices.azure.com, *.services.ai.azure.com) are refused
	// too; an operator who uses one sets AllowCustomEndpoint, which is
	// hard-locked, so only config.yaml can.
	host := strings.ToLower(u.Hostname())
	return host == `api.openai.com` || strings.HasSuffix(host, `.openai.azure.com`)
```

and replace the whole doc comment above `EndpointAllowed`

```go
// EndpointAllowed keeps the key and whatever is sent going where they are
// meant to: https, and an OpenAI (or Azure OpenAI) host, unless the operator
// has deliberately allowed another provider.
```
with
```go
// EndpointAllowed keeps the key and whatever is sent going where they are
// meant to: https, and exactly api.openai.com or an Azure OpenAI resource
// (*.openai.azure.com), unless the operator has deliberately allowed
// another provider (AllowCustomEndpoint, hard-locked).
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/apiframework/ -v -run "TestEndpointAllowed|TestServer"`
Expected: PASS (the `TestServer*` tests use `https://api.openai.com/v1` and a custom host with `AllowCustomEndpoint`, both still accepted).

- [ ] **Step 5: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/apiframework/settings.go internal/apiframework/apiframework_test.go && git commit -F - <<'EOF'
fix(apiframework): send the key only to api.openai.com or Azure OpenAI

EndpointAllowed accepted any *.openai.com or *.azure.com host. It now
accepts exactly api.openai.com and *.openai.azure.com unless
AllowCustomEndpoint.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 3: S2, provider error text never keeps a key

**Files:**
- Modify: `internal/apiframework/wire.go` (`DecodeChat`, new `keyTextRE`)
- Modify: `internal/apiframework/apiframework_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/apiframework/apiframework_test.go`:

```go
// A provider's error text can quote the key it was sent ("Incorrect API key
// provided: sk-proj-...") and DecodeChat keeps that text in the error, which
// callers log. Anything shaped like an OpenAI key is scrubbed first, so no
// part of one survives, even one the 300-byte cut would have split (spec S2).
func TestDecodeChatScrubsKeysFromErrorText(t *testing.T) {
	cases := map[string]string{
		`quoted`:   `{"error":{"message":"Incorrect API key provided: sk-proj-abc123SECRETwxyz. You can find your API key at https://platform.openai.com"}}`,
		`masked`:   `{"error":{"message":"Incorrect API key provided: sk-proj-****wxyz."}}`,
		`plain`:    `bad key sk-abcdefSECRET0123456789`,
		`boundary`: strings.Repeat(`x`, 290) + ` sk-SECRETSECRETSECRETSECRETSECRET`,
	}
	for name, body := range cases {
		err := DecodeChat(401, []byte(body)).Err
		if err == nil {
			t.Fatalf("%s: a 401 is an error", name)
		}
		got := err.Error()
		if strings.Contains(got, `SECRET`) || strings.Contains(got, `abc123`) || strings.Contains(got, `wxyz`) {
			t.Errorf("%v: key material survived in %d bytes of error text", name, len(got))
		}
		if !strings.Contains(got, `401`) {
			t.Errorf("%s: the status is still reported", name)
		}
	}
	// Ordinary words that merely contain "sk-" are left alone.
	if got := DecodeChat(500, []byte(`task-queue risk-free`)).Err.Error(); !strings.Contains(got, `task-queue risk-free`) {
		t.Errorf("words ending in sk- are not keys, got %d bytes", len(got))
	}
}
```

`TestNoTestPrintsAKey` scans this package's tests and flags any message that mentions "key" (case-insensitive, so "keys" counts) and has a `%s` or `%q` verb anywhere in it, whatever the verb prints. So the case name in "key material survived" is printed with `%v`, and "words ending in sk- are not keys" prints only `%d`; the two messages with `%s` ("a 401 is an error", "the status is still reported") never say "key". Step 4 runs the guard.

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/apiframework/ -run TestDecodeChatScrubsKeysFromErrorText -v`
Expected: FAIL for `quoted`, `masked`, `plain` and `boundary` ("key material survived"; `masked` fails on its `wxyz`).

- [ ] **Step 3: Scrub before the cut**

In `internal/apiframework/wire.go`, add `"regexp"` to the imports and, above `DecodeChat`, add:

```go
// keyTextRE matches an OpenAI key in provider text: sk-..., sk-proj-...,
// and the masked sk-proj-****abcd form, from the key's start to the next
// space. \b keeps it off words that merely end in "sk" (task-, risk-).
var keyTextRE = regexp.MustCompile(`\bsk-\S*`)
```

Replace in `DecodeChat`

```go
		snippet := strings.TrimSpace(string(raw))
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
```
with
```go
		// Scrub any key BEFORE the cut, so a key the cut would split
		// leaves no fragment behind.
		snippet := strings.TrimSpace(keyTextRE.ReplaceAllString(string(raw), `sk-[redacted]`))
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/apiframework/ -v -run "TestDecodeChat|ProviderFailure|TestNoTestPrintsAKey" && go test ./internal/apiframework/ ./modules/aicompanion/`
Expected: PASS; both packages `ok`.

- [ ] **Step 5: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/apiframework/wire.go internal/apiframework/apiframework_test.go && git commit -F - <<'EOF'
fix(apiframework): scrub keys from the provider error text DecodeChat keeps

A 401 body quotes the key it was sent, and DecodeChat kept 300 bytes of
it in an error callers log. Key-shaped text is replaced before the cut.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 4: S2, the baubles model, spend and moderation settings are hard-locked

Slice M's plan (`docs/superpowers/plans/2026-09-28-slice-m-config-locks-redaction.md`)
declares `var hardLocked = []string{...}` in `internal/configs/config_locks.go`
with `isHardLocked(configPath string) bool` and the exported
`IsLocked(configPath string) bool`, and already lists the four
`APIFramework.*` paths and the companion's rows (including its
`ModerateOutput` and `ModerationModel`, ruling 13); it leaves
`Modules.baubles.*` to this slice. Task 0 Step 2 confirms what actually
landed; if a name differs, use the landed one. The six key names below are
the ones `modules/baubles/config.go` reads (`get(`Model`)`,
`MaxCompletionTokens`, `MaxConcurrent`, `UsePlayerKeys`, `ModerateOutput`,
`ModerationModel`, lines 121-146 at `e711ee9de`); recheck them with
`grep -n 'get(`' modules/baubles/config.go` before Step 3.

**Files:**
- Modify: `internal/configs/config.apiframework_test.go`
- Modify: `internal/configs/config_locks.go` (`hardLocked`)

- [ ] **Step 1: Write the failing test**

Append to `internal/configs/config.apiframework_test.go`:

```go
// The server key's destination, what spends it and whether its text is
// moderated cannot be changed from inside the game (spec M1 and S2, ruling
// 13): an admin who could set BaseURL or the model could send the key, or
// its budget, anywhere, and one who could turn moderation off would let
// unchecked text into the world. IsLocked is what SetVal and the server
// config menu both ask.
func TestKeyAndBaublesSpendSettingsAreHardLocked(t *testing.T) {
	for _, p := range []string{
		`APIFramework.APIKey`, `APIFramework.APIKeyEnv`, `APIFramework.BaseURL`, `APIFramework.AllowCustomEndpoint`,
		`Modules.baubles.Model`, `Modules.baubles.MaxCompletionTokens`, `Modules.baubles.MaxConcurrent`, `Modules.baubles.UsePlayerKeys`,
		`Modules.baubles.ModerateOutput`, `Modules.baubles.ModerationModel`,
		`modules.baubles.model`,
	} {
		if !isHardLocked(p) || !IsLocked(p) {
			t.Errorf("%v is not hard-locked", p)
		}
	}
	// A setting that only bounds how long a find waits is not the key's
	// destination or its spend: TimeoutSeconds stays tunable in game.
	if isHardLocked(`Modules.baubles.TimeoutSeconds`) {
		t.Error("Modules.baubles.TimeoutSeconds bounds a wait, not the spend, and stays tunable")
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test ./internal/configs/ -run TestKeyAndBaublesSpendSettingsAreHardLocked -v`
Expected: FAIL naming the six `Modules.baubles.*` paths and `modules.baubles.model`.

- [ ] **Step 3: Add the rows**

In `internal/configs/config_locks.go`, in `hardLocked`, after the `Modules.aicompanion.ModerationModel` row (the companion's last, below its `// ruling 13` comment, `:31` at `3bd6ccaa3`), add:

```go
	// Bauble naming spends the server's key: its model, how much one find
	// may spend and run at once, and whether and how its text is moderated
	// are the owner's call, not an admin's (spec S2, ruling 13). The daily
	// budget knobs stay tunable in game.
	`Modules.baubles.Model`,
	`Modules.baubles.MaxCompletionTokens`,
	`Modules.baubles.MaxConcurrent`,
	`Modules.baubles.UsePlayerKeys`,
	`Modules.baubles.ModerateOutput`,
	`Modules.baubles.ModerationModel`,
```

- [ ] **Step 4: Run to verify it passes, and slice M's own lock tests still pass**

Run: `go test ./internal/configs/ ./internal/usercommands/ -count=1 2>&1 | tail -5`
Expected: both `ok`.

- [ ] **Step 5: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/configs/config.apiframework_test.go internal/configs/config_locks.go && git commit -F - <<'EOF'
fix(configs): hard-lock the baubles model, spend and moderation settings

Modules.baubles.Model, MaxCompletionTokens, MaxConcurrent, UsePlayerKeys,
ModerateOutput and ModerationModel join the hard lock list beside the
key's own settings, so no admin can redirect or enlarge what bauble
naming spends or switch off the check on its text.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 5: S4, output cleaning on every route

**Files:**
- Modify: `internal/baubles/validate.go` (`cleanLine`, `CleanReply`, new `linkRE`, `cleanRune`, `quoteShort`)
- Create: `internal/baubles/validate_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/baubles/validate_test.go`:

```go
package baubles

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

// Every code point the v10 review showed surviving CleanReply, plus the
// others the spec names (S4): each is dropped or turned into a plain space,
// never kept.
func TestCleanReplyStripsInvisibleAndFormatCharacters(t *testing.T) {
	cases := map[string]rune{
		`zero width space`:   '\u200B',
		`right-to-left mark`: '\u202E',
		`byte order mark`:    '\uFEFF',
		`line separator`:     '\u2028',
		`paragraph sep`:      '\u2029',
		`hangul filler`:      '\u3164',
		`hangul choseong f`:  '\u115F',
		`hangul jungseong f`: '\u1160',
		`halfwidth filler`:   '\uFFA0',
		`private use`:        '\uE000',
		`combining stroke`:   '\u0336',
		`ideographic space`:  '\u3000',
	}
	for name, r := range cases {
		reply := goodReply()
		reply.Name = `Painted` + string(r) + ` Wooden Horse`
		reply.Description = `A child's toy` + string(r) + ` horse, its red paint flaking from the mane.`
		got, err := CleanReply(reply)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if strings.ContainsRune(got.Name, r) || strings.ContainsRune(got.Description, r) {
			t.Errorf("%s (U+%04X) survived", name, r)
		}
		if strings.Contains(got.Name, `  `) {
			t.Errorf("%s: spaces not collapsed in %q", name, got.Name)
		}
	}

	// NBSP and the other spaces become an ordinary space.
	reply := goodReply()
	reply.Name = "Painted\u00a0Wooden\u2003Horse"
	if got, err := CleanReply(reply); err != nil || got.Name != `Painted Wooden Horse` {
		t.Fatalf("NBSP and em space read as spaces: %q %v", got.Name, err)
	}

	// An OSC sequence (not the CSI colour codes ansiRE knows) leaves no
	// escape or bell behind. It goes in the description: its "8" would
	// refuse a name for its digit before the escape was ever looked at.
	reply = goodReply()
	reply.Description = "A child's toy horse\x1b]8;;evil\x07, its red paint flaking from the mane."
	got, err := CleanReply(reply)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(got.Description, "\x1b\x07") {
		t.Fatalf("OSC control bytes survived: %q", got.Description)
	}

	// Fullwidth markup is folded by NFKC before the markup is stripped.
	reply = goodReply()
	reply.Name = "Painted \uFF1Cb\uFF1EWooden Horse"
	if got, err := CleanReply(reply); err != nil || strings.ContainsAny(got.Name, "<>\uFF1C\uFF1E") {
		t.Fatalf("fullwidth angle brackets: %q %v", got.Name, err)
	}
}

// Text that reads as a link is refused in every field (spec S4).
func TestCleanReplyRefusesLinks(t *testing.T) {
	cases := map[string]func(*Reply){
		`scheme`:      func(r *Reply) { r.Description = `A toy horse. Details at http://example.org today.` },
		`www`:         func(r *Reply) { r.Description = `A toy horse from WWW.shop, still boxed in paper.` },
		`bare domain`: func(r *Reply) { r.Name = `Evil.com Horse` },
		`path`:        func(r *Reply) { r.Description = `A toy horse with a tag reading evil.co/x on its belly.` },
		`fullwidth`:   func(r *Reply) { r.Description = "A toy horse stamped \uFF57\uFF57\uFF57\uFF0Eshop in red." },
		`material`:    func(r *Reply) { r.Material = `pine.io` },
	}
	for name, change := range cases {
		r := goodReply()
		change(&r)
		if _, err := CleanReply(r); !errors.Is(err, ErrUnusableReply) {
			t.Errorf("%s must be refused, got %v", name, err)
		}
	}
	// Ordinary sentences are not links.
	r := goodReply()
	r.Description = `A cup. It is old, and chipped at the rim! Who left it here?`
	if _, err := CleanReply(r); err != nil {
		t.Fatalf("plain sentences pass: %v", err)
	}
}

// Lengths are runes, not bytes (spec S4).
func TestCleanReplyCountsRunes(t *testing.T) {
	r := goodReply()
	r.Name = strings.Repeat(`é`, maxNameLen-2) + ` A` // 40 runes, 78 bytes
	if _, err := CleanReply(r); err != nil {
		t.Fatalf("a %d-rune name fits: %v", utf8.RuneCountInString(r.Name), err)
	}
	r.Name = strings.Repeat(`é`, maxNameLen-1) + ` A` // 41 runes
	if _, err := CleanReply(r); !errors.Is(err, ErrUnusableReply) {
		t.Fatal("a 41-rune name is too long")
	}
	r = goodReply()
	r.Description = strings.Repeat(`é`, minDescriptionLen-1) // 19 runes, 38 bytes
	if _, err := CleanReply(r); !errors.Is(err, ErrUnusableReply) {
		t.Fatal("19 runes is too short, however many bytes")
	}
}

// A refusal quotes at most 60 runes of the offending text: a reply can be
// a megabyte, and the error is logged at Warn.
func TestCleanReplyErrorsQuoteLittle(t *testing.T) {
	r := goodReply()
	r.Name = strings.Repeat(`Long `, 100)
	_, err := CleanReply(r)
	if err == nil {
		t.Fatal("refused")
	}
	if strings.Contains(err.Error(), strings.Repeat(`Long `, 13)) {
		t.Fatalf("the error quotes more than 60 runes: %d bytes", len(err.Error()))
	}
}

// Curly quotes, en and em dashes and the ellipsis fold to ASCII (owner
// ruling 15), so ordinary typography from a model reads the same on every
// route and can pass the player-key allowlist (Task 7).
func TestCleanReplyFoldsTypography(t *testing.T) {
	r := goodReply()
	r.Name = "Mara\U00002019s Wooden Horse"
	r.Description = "A child\U00002018s toy horse \U00002014 its paint flaking \U00002013 marked \U0000201CMara\U0000201D\U00002026 still loved."
	got, err := CleanReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != `Mara's Wooden Horse` {
		t.Errorf("name folded: got %q", got.Name)
	}
	if want := `A child's toy horse - its paint flaking - marked "Mara"... still loved.`; got.Description != want {
		t.Errorf("description folded:\n got %q\nwant %q", got.Description, want)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/baubles/ -run "TestCleanReplyStrips|TestCleanReplyRefusesLinks|TestCleanReplyCountsRunes|TestCleanReplyErrorsQuoteLittle|TestCleanReplyFoldsTypography" -v`
Expected: FAIL in each of the five. In `TestCleanReplyStripsInvisibleAndFormatCharacters`, "survived" errors for the zero width space, RLO, BOM and the other runes, then the test stops at its first `Fatalf`, "NBSP and em space read as spaces" (today's `whitespaceRE` is ASCII `\s`, so NBSP survives), so the OSC and fullwidth checks below it do not run yet; the link cases "must be refused"; the 41-rune and 19-rune cases; the long quote; both folds (the curly quotes and dashes survive today, and the ellipsis too, since today's `cleanLine` does no NFKC).

Note for Step 3: `PlainText` (`validate.go:129`, the room text sent in the prompt) is `cleanLine`, so NFKC, the typography fold and the dropped format characters apply to prompt text too. That is intended (the model gets plain text); record it in the context.md step.

- [ ] **Step 3: Implement the cleaning**

In `internal/baubles/validate.go`, set the imports to:

```go
import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"github.com/GoMudEngine/GoMud/internal/items"
)
```

In the `var (...)` block, delete the `whitespaceRE` line and add:

```go
	// linkRE is text that reads as a link: a scheme, a www., or a
	// domain-shaped word (evil.com, shop.co/x). Matched on lower-cased text.
	linkRE = regexp.MustCompile(`://|www\.|[a-z0-9-]+\.[a-z]{2,6}(/|\b)`)
```

Replace the whole of `cleanLine` with:

```go
// typographyFold turns the typography a model reaches for into ASCII
// (owner ruling 15): curly quotes to ' and ", en and em dashes to -, the
// ellipsis to ... (NFKC already does the ellipsis; it is listed so the rule
// reads whole). It also drops < and > (markup leftovers) and turns a
// backtick into '.
var typographyFold = strings.NewReplacer(
	`<`, ``, `>`, ``, "`", `'`,
	"\U00002018", `'`, "\U00002019", `'`, "\U0000201C", `"`, "\U0000201D", `"`,
	"\U00002013", `-`, "\U00002014", `-`, "\U00002026", `...`,
)

// cleanLine folds compatibility forms (NFKC, so fullwidth markup and
// letters become plain ones before anything else looks), strips markup,
// turns control characters and every Unicode space into a plain space,
// drops format, private-use, surrogate and combining characters, the line
// and paragraph separators and the Hangul fillers (which render as
// nothing), folds curly quotes, dashes and the ellipsis to ASCII
// (typographyFold), and collapses whitespace.
func cleanLine(s string) string {
	s = norm.NFKC.String(s)
	s = ansiRE.ReplaceAllString(s, ``)
	s = tagRE.ReplaceAllString(s, ``)
	s = strings.Map(cleanRune, s)
	s = typographyFold.Replace(s)
	return strings.Join(strings.Fields(s), ` `)
}

// cleanRune is cleanLine's per-character rule: -1 drops it.
func cleanRune(r rune) rune {
	switch {
	case r == '\u2028', r == '\u2029', r == '\u115F', r == '\u1160', r == '\u3164', r == '\uFFA0':
		return -1
	case unicode.IsControl(r), unicode.Is(unicode.Zs, r):
		return ' '
	case unicode.In(r, unicode.Cf, unicode.Co, unicode.Cs, unicode.Mn):
		return -1
	}
	return r
}

// quoteShort quotes at most 60 runes of s for an error that may be logged.
func quoteShort(s string) string {
	const most = 60
	if utf8.RuneCountInString(s) > most {
		s = string([]rune(s)[:most]) + `...`
	}
	return strconv.Quote(s)
}
```

In `CleanReply`, replace

```go
	words := strings.Fields(r.Name)
	if r.Name == `` || len(r.Name) > maxNameLen || len(words) < minNameWords || len(words) > maxNameWords {
		return bad(`name %q`, r.Name)
	}
	if strings.IndexFunc(r.Name, unicode.IsDigit) >= 0 {
		return bad(`name has digits: %q`, r.Name)
	}
	if len(r.Description) < minDescriptionLen || len(r.Description) > maxDescriptionLen {
		return bad(`description length %d`, len(r.Description))
	}
	if len(r.Material) > maxMaterialLen {
		r.Material = ``
	}
```
with
```go
	words := strings.Fields(r.Name)
	if r.Name == `` || utf8.RuneCountInString(r.Name) > maxNameLen || len(words) < minNameWords || len(words) > maxNameWords {
		return bad(`name %s`, quoteShort(r.Name))
	}
	if strings.IndexFunc(r.Name, unicode.IsDigit) >= 0 {
		return bad(`name has digits: %s`, quoteShort(r.Name))
	}
	if n := utf8.RuneCountInString(r.Description); n < minDescriptionLen || n > maxDescriptionLen {
		return bad(`description length %d`, n)
	}
	if utf8.RuneCountInString(r.Material) > maxMaterialLen {
		r.Material = ``
	}
	for _, f := range [...]struct{ field, text string }{{`name`, r.Name}, {`description`, r.Description}, {`material`, r.Material}} {
		if linkRE.MatchString(strings.ToLower(f.text)) {
			return bad(`%s reads as a link: %s`, f.field, quoteShort(f.text))
		}
	}
```

Known limits of `linkRE`, accepted (owner review 2026-09-28) and written into the context.md in Task 5a:

- It misses a top-level domain longer than six letters (`evil.example`, `shop.museum` is six and caught) and a domain written with the ideographic full stop U+3002 (`evil。com`; NFKC leaves U+3002 alone). On the server route the text is moderated and the server's own key wrote it; on a player's key the ASCII allowlist (Task 7) refuses U+3002 outright, and a long TLD still needs a period glued to letters, which the allowlist's period rule refuses.
- It refuses a model's missing-space typo such as `horse.Its` (it reads as `horse.its`, domain-shaped) on every route. The find falls back as any unusable reply does; the prompt (Task 10) asks for a space after every sentence.

- [ ] **Step 4: Run to verify they pass, and every older CleanReply test**

Run: `go test ./internal/baubles/ -v -run "CleanReply|Keyword|Pickpocket|Generate|Mint" 2>&1 | tail -30`
Expected: PASS.

- [ ] **Step 5: Confirm go.mod is unchanged**

Run: `cd /c/tmp/dogmud-baubles-h && go build ./... && go mod tidy && git diff --exit-code go.mod go.sum`
Expected: exit 0, no diff. `golang.org/x/text` is already a direct requirement, so `go mod tidy` changes neither file. If it does show a diff, restore both files with `git checkout -- go.mod go.sum` and report: the diff is unrelated drift, not this task's.

- [ ] **Step 6: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/baubles/validate.go internal/baubles/validate_test.go && git commit -F - <<'EOF'
fix(baubles): strip invisible Unicode and links from every reply

cleanLine folds NFKC and curly quotes, dashes and the ellipsis to ASCII,
maps every Unicode space to a space and drops format, private-use,
combining and filler characters; lengths count runes; link-shaped text is
refused; errors quote at most 60 runes.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 5a: PR H1 docs

(Text quoted from a `context.md` here is shown unwrapped; the file wraps it, so match it by its words and rewrap the replacement to the file's width.)

**Files:**
- Modify: `internal/baubles/context.md`, `internal/apiframework/context.md`, `internal/configs/context.md`, `modules/baubles/context.md`
- Modify: `_datafiles/config.yaml` (comments only), `docs/aicompanion/settings.md`

- [ ] **Step 0: Verify every symbol these docs name**

```bash
cd /c/tmp/dogmud-baubles-h && grep -rn "var keyTextRE\|var typographyFold\|func cleanRune\|func quoteShort\|var linkRE\|linkRE = \|APIKey ConfigSecret\|Modules.baubles.ModerationModel" internal modules
```
Expected: at least one hit for each pattern except `var linkRE` (`linkRE` is declared inside a `var (...)` block, so `linkRE = ` finds it instead). A symbol with no hit is not in the code: fix the doc text, never the grep.

- [ ] **Step 1: `internal/baubles/context.md`**

- In the **validate.go** file bullet, describe `CleanReply` as: `` `CleanReply` (the text checks the schema cannot make: NFKC, curly quotes, en and em dashes and the ellipsis folded to ASCII (`typographyFold`), invisible and format characters dropped (`cleanRune`), rune lengths, link-shaped text refused (`linkRE`), errors quoting at most 60 runes (`quoteShort`)) and `PlainText`, which is `cleanLine`, so the room text in a prompt is folded the same way. ``
- Add a gotcha: `` - **`linkRE` is a heuristic.** It misses a top-level domain longer than six letters and a domain written with U+3002 (NFKC keeps it); server-key text is moderated, and player-key text is held to the ASCII allowlist, which refuses both. It also refuses a missing-space typo such as `horse.Its` on every route; that find falls back like any unusable reply. Accepted by the owner, 2026-09-28. ``

- [ ] **Step 2: `internal/apiframework/context.md`**

- In the **wire.go** bullet change `` `Reply` and `DecodeChat`; `` to `` `Reply` and `DecodeChat` (a non-200's kept text has key-shaped strings scrubbed by `keyTextRE` before the 300-byte cut); ``.
- In the config paragraph, after the `AllowCustomEndpoint` description add: `` Without it `EndpointAllowed` accepts exactly `api.openai.com` and `*.openai.azure.com` over https; Azure's AI Services hosts (`*.cognitiveservices.azure.com`, `*.services.ai.azure.com`) and every other host need `AllowCustomEndpoint`, which, like the other three, is hard-locked (`configs.hardLocked`), so only config.yaml sets it. `APIKey` is a `configs.ConfigSecret`. ``

- [ ] **Step 3: `internal/configs/context.md`**

Where the `APIFramework` section is shown (the block around `APIKeyEnv: "OPENAI_API_KEY"`), add below it: `` `APIFramework.APIKey` is a `ConfigSecret`. `hardLocked` also holds `Modules.baubles.Model`, `.MaxCompletionTokens`, `.MaxConcurrent`, `.UsePlayerKeys`, `.ModerateOutput` and `.ModerationModel`. `` (If the file already lists `hardLocked`'s rows, add the six baubles rows to that list instead.)

- [ ] **Step 4: `modules/baubles/context.md`**

In the Config paragraph, add after the list: `` `Model`, `MaxCompletionTokens`, `MaxConcurrent`, `UsePlayerKeys`, `ModerateOutput` and `ModerationModel` are hard-locked (`configs.hardLocked`): only config.yaml sets them. ``

- [ ] **Step 5: `_datafiles/config.yaml` comments**

First confirm the disk copy equals HEAD and check the skip-worktree bit:
```bash
cd /c/tmp/dogmud-baubles-h && git diff --quiet HEAD -- _datafiles/config.yaml; echo "differs=$?"; git ls-files -v _datafiles/config.yaml
```
If `differs=0`: edit the disk copy with the Edit tool. If `differs=1`: STOP editing disk; write `git show HEAD:_datafiles/config.yaml` to the scratchpad, make the edit there with the Edit tool, then stage it with `blob=$(git hash-object -w <scratch copy>) && git update-index --cacheinfo 100644,$blob,_datafiles/config.yaml`, and if `git ls-files -v` then prints `H` where it printed `S`, restore the bit with `git update-index --skip-worktree _datafiles/config.yaml`.

Replace (a comment; no value changes)
```yaml
  # A find is named through the finder's OWN key when they ticked "Also name
  # things I find while searching" on the Companion key page (UsePlayerKeys),
  # else through the server's key (APIFramework: its one daily budget and
  # breaker, shared with the AI companion), else it is a generic "Trinket".
```
with
```yaml
  # A find is named through the finder's OWN key when they ticked "Also name
  # things I find while searching" on the Companion key page (UsePlayerKeys),
  # else through the server's key (APIFramework: its one daily budget and
  # breaker, shared with the AI companion), else it is a generic "Trinket".
  # Model, MaxCompletionTokens, MaxConcurrent, UsePlayerKeys, ModerateOutput
  # and ModerationModel are hard-locked: only this file changes them.
```

- [ ] **Step 6: `docs/aicompanion/settings.md`**

- Line 30's comment: change `Off: any other host is refused` to `Off: only api.openai.com and *.openai.azure.com are accepted (Azure AI Services hosts need it on)`.
- Directly after the paragraph that ends `page refuses the bauble request shape from a key whose box is not ticked.` (line 346), add a blank line and:

```markdown
`AllowCustomEndpoint` off accepts exactly `api.openai.com` and Azure OpenAI
resources (`*.openai.azure.com`). Azure's AI Services hosts
(`*.cognitiveservices.azure.com`, `*.services.ai.azure.com`) need it on, and
it is hard-locked: only the config file changes it.
```

- [ ] **Step 7: The context.md audit, and commit**

Run: `cd /c/tmp/dogmud-baubles-h && python tools/context_md_audit.py 2>&1 | grep -E "baubles|apiframework|configs" ; echo "exit=$?"`
Expected: no phantom symbols in these packages (grep exit 1 means none listed). The tool only reads files.

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/baubles/context.md internal/apiframework/context.md internal/configs/context.md modules/baubles/context.md _datafiles/config.yaml docs/aicompanion/settings.md && git commit -F - <<'EOF'
docs(baubles): PR H1 key secrecy, endpoints, locks and cleaning

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```
If Step 5 staged `config.yaml` through `update-index`, leave it out of `git add` (it is already staged) and confirm with `git diff --cached --stat` before committing.

---

### Task 5b: PR H1 gate

**Files:** none new. Every diff runs from `$BASE` (`/c/tmp/dogmud-baubles-h.base`, Task 0 Step 1), never `master...HEAD`.

- [ ] **Step 1: gofmt on the committed blobs (the Windows CRLF false positive)**

```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && for f in $(git diff --name-only $BASE..HEAD -- '*.go'); do out=$(git show HEAD:"$f" | gofmt -l); [ -n "$out" ] && echo "UNFORMATTED: $f"; done; echo done
```
Expected: only `done`. For a listed file, run `gofmt -w <file>`, check `git diff` shows only whitespace, and commit it as a fixup.

- [ ] **Step 2: Size under the CI lint inversion**

```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && git diff --shortstat $BASE..HEAD
```
Expected: under 300 files and under 20,000 lines.

- [ ] **Step 3: Build, vet, tests**

```bash
cd /c/tmp/dogmud-baubles-h && go build ./... && go vet ./internal/configs/ ./internal/apiframework/ ./internal/baubles/ ./internal/usercommands/ ./modules/baubles/ ./modules/aicompanion/ . && go test ./internal/configs/ ./internal/apiframework/ ./internal/baubles/ ./internal/usercommands/ ./modules/baubles/ ./modules/aicompanion/ . 2>&1 | tail -12
```
Expected: no vet output, every package `ok`.

- [ ] **Step 4: The guards this PR leans on, by name**

```bash
cd /c/tmp/dogmud-baubles-h && go test . -count=1 -run "TestNoDisplayReadsRawConfig|TestFindRawConfigInTemplateMatcher" -v 2>&1 | grep -E "^(--- FAIL|--- PASS|FAIL|ok)"
```
```bash
cd /c/tmp/dogmud-baubles-h && go test ./internal/apiframework/ -count=1 -run "TestNoTestPrintsAKey|TestNoServerKeyReachesTheConfigDisplay|TestEndpointAllowed|TestDecodeChatScrubsKeysFromErrorText" -v 2>&1 | grep -E "^(--- FAIL|--- PASS|FAIL|ok)"
```
Expected: every named test `--- PASS`. A name that does not appear has changed on master: find it with `grep -n "^func Test"` before calling the gate green.

- [ ] **Step 5: Boot smoke in the test process (no port)**

```bash
cd /c/tmp/dogmud-baubles-h && DOGMUD_BOOT_SMOKE=1 go test . -count=1 -run TestSmoke_ServerBootsCleanWithRealData -timeout 600s 2>&1 | tail -5
```
Expected: `ok`.

- [ ] **Step 6: A real boot on private ports, stopped by its own PID**

The owner runs their own server on this machine. This boot uses dogmud-shipping's fixed path `C:\tmp\dogmud-boot-check\boot-check.exe` (its firewall rule is keyed on that path, so no "Windows Security Alert" pops up), private ports through a `CONFIG_PATH` override file (never the main checkout's config), and is stopped by the PID it started, never by name or port.

First check the path is free (another session may be using it):
```bash
git -C /c/tmp/dogmud-baubles-h worktree list | grep -F "dogmud-boot-check"; ls /c/tmp/dogmud-boot-check 2>/dev/null | head -3
```
Expected: no output from either (each exits 1). If the directory exists, STOP and ask: it may be another session's boot check.

Build (Bash):
```bash
git -C /c/tmp/dogmud-baubles-h worktree add --detach /c/tmp/dogmud-boot-check HEAD && cd /c/tmp/dogmud-boot-check && go build -o boot-check.exe . && cat > /c/tmp/dogmud-boot-check.overrides.yaml <<'EOF'
Network:
  TelnetPort: [33533]
  LocalPort: 9899
  HttpPort: 8391
  HttpsPort: 0
  AIPort: 0
EOF
echo built
```
Expected: `built`. The worktree's `_datafiles/config.yaml` is the committed blob; the override file lives outside every checkout.

Run (PowerShell), wait for `Server Ready` or 180 seconds, and stop exactly that PID:
```powershell
$env:CONFIG_PATH = 'C:\tmp\dogmud-boot-check.overrides.yaml'
$p = Start-Process -FilePath 'C:\tmp\dogmud-boot-check\boot-check.exe' -WorkingDirectory 'C:\tmp\dogmud-boot-check' -RedirectStandardOutput 'C:\tmp\dogmud-boot-check.log' -RedirectStandardError 'C:\tmp\dogmud-boot-check.err' -WindowStyle Hidden -PassThru
$deadline = (Get-Date).AddSeconds(180)
while ((Get-Date) -lt $deadline -and -not $p.HasExited -and -not (Select-String -Path 'C:\tmp\dogmud-boot-check.log','C:\tmp\dogmud-boot-check.err' -Pattern 'Server Ready' -Quiet)) { Start-Sleep -Seconds 2 }
"pid=$($p.Id) exited=$($p.HasExited)"
Select-String -Path 'C:\tmp\dogmud-boot-check.log','C:\tmp\dogmud-boot-check.err' -Pattern 'Server Ready|status=listening|^panic:|goroutine [0-9]+ \[running\]|runtime error|bind:' | Select-Object -First 20
if (-not $p.HasExited) { Stop-Process -Id $p.Id -Force -Confirm:$false }
Remove-Item Env:\CONFIG_PATH
```
Expected: `exited=False` (it stayed up), a `Server Ready` line, a `Telnet status=listening port=33533` line (`main.go:1549` logs `"Telnet", "status", "listening", "port", ...`) and none naming 33333 or 44444, and no `panic:`, `goroutine ... [running]`, `runtime error` or `bind:` line (a bind error means a collision with a port someone else holds). Do not grep for the bare word `panic`: `GamePlay.MapConsistencyEnforce` has the value `panic`.

Clean up (PowerShell; `Remove-Item` succeeds where `git worktree remove` can fail on a just-stopped exe):
```powershell
Remove-Item -Recurse -Force 'C:\tmp\dogmud-boot-check'; Remove-Item -Force 'C:\tmp\dogmud-boot-check.overrides.yaml','C:\tmp\dogmud-boot-check.log','C:\tmp\dogmud-boot-check.err'
```
then `git -C /c/tmp/dogmud-baubles-h worktree prune`.

- [ ] **Step 7: Race, in the Linux test container**

```bash
cd /c/tmp/dogmud-baubles-h && docker compose -f compose.test.yml run --build --rm test go test -race -count=1 ./internal/apiframework/ ./internal/baubles/ ./internal/configs/ 2>&1 | tail -8
```
Expected: `ok` for all three, no `WARNING: DATA RACE`. CI is out of minutes until 2026-10-01; this run IS the race gate.

- [ ] **Step 8: Lint, go.mod, docs, tree**

```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && golangci-lint run --new-from-merge-base=$BASE
```
Expected: `0 issues` (a finding on a line this PR did not write is pre-existing; check `git log -L`).
```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && git diff --exit-code $BASE..HEAD -- go.mod go.sum; echo "exit=$?"; git diff --name-only $BASE..HEAD -- '*context.md' _datafiles/config.yaml docs/; git status --short
```
Expected: `exit=0`; the four `context.md` files of Task 5a, `_datafiles/config.yaml`, `docs/aicompanion/settings.md`, this plan and `docs/README.md` listed; a clean tree.

- [ ] **Step 9: Hand over**

No push from here. Report the branch, `$BASE`, the commit list (`git log --oneline $BASE..HEAD`) and each gate result. When the owner says to ship it: `git push -u origin fix/baubles-hardening-h`, then `gh pr create --repo pruuk/DOGMud --base master --head fix/baubles-hardening-h --title "Slice H1: key secrecy, OpenAI-only endpoints, hard locks, output cleaning"`, and read back the URL `gh` prints: it must say `pruuk/DOGMud`. PR H2 starts only after this PR has merged.

---

## PR H2: S3 and finder-only text

---

### Task H2-0: Branch PR H2 from master

**Files:** none in the repo.

- [ ] **Step 1: PR H1 is merged; switch the worktree**

```bash
cd /c/tmp/dogmud-baubles-h && gh pr list --repo pruuk/DOGMud --state merged --head fix/baubles-hardening-h --json number,mergedAt && git status --short && git fetch -q origin && git switch -c fix/baubles-hardening-h2 origin/master && BASE=$(git rev-parse HEAD) && echo "$BASE" > /c/tmp/dogmud-baubles-h.base && echo "BASE=$BASE"
```
Expected: one merged PR listed, a clean tree, and a new `BASE`. If the list is empty, STOP: H2 waits for H1. If `git switch` refuses because `_datafiles/config.yaml` differs, the worktree has a local edit: restore it from `git show HEAD:_datafiles/config.yaml` first.

- [ ] **Step 2: H2's anchors exist**

```bash
cd /c/tmp/dogmud-baubles-h && anchors=$(mktemp) && cat > "$anchors" <<'EOF'
internal/baubles/validate.go|var authoredKeyword = items.AuthoredKeyword
internal/items/itemspec.go|var authoredWords atomic.Pointer[map[string]bool]
internal/items/itemspec.go|func AuthoredKeyword(word string) bool {
internal/items/bauble_placement_test.go|func TestAuthoredKeywordIsSafeWhileItemsAreWritten(
internal/baubles/generate.go|cleaned, err := CleanReply(res.Reply)
internal/baubles/generate.go|if r.Zone == zone && r.Generator == GeneratorOpenAI {
internal/baubles/generate.go|PlayerKey     bool // named through the finder's own key, not the server's
internal/baubles/mint.go|limited := ApplyLimitsFor(res.Reply, tier, source)
internal/baubles/admin.go|r.Moderated = res.Moderated
internal/baubles/record.go|func (r Record) View() items.BaubleView {
internal/actions/bauble_admin.go|req.RecentNames = append(req.RecentNames, rec.Name)
internal/items/items.go|func (i *Item) GetLongDescription() string {
internal/items/items.go|if i.GetSpec().QuestToken != `` {
internal/items/bauble.go|func baubleSpec(base ItemSpec, id string) ItemSpec {
internal/actions/search_bauble.go|name := itm.DisplayName()
internal/actions/steal.go|fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, b.DisplayName()))
internal/usercommands/inventory.go|iNameFormatted := fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, item.Name())
internal/usercommands/look.go|itemDesc := lookItem.GetLongDescription()
internal/usercommands/appraise.go|if rec.Material != `` {
internal/usercommands/admin.bauble.go|fmt.Fprintf(&b, "  generator:   %s %s (prompt v%d, %d tokens)\r\n", rec.Generator, rec.Model, rec.PromptVersion, rec.Tokens)
modules/gmcp/gmcp.Char.go|payload.Inventory.Backpack.Items = append(payload.Inventory.Backpack.Items, newInventory_Item(itm))
modules/gmcp/gmcp.Room.go|Name:      itm.Name(),
internal/apiframework/breaker.go|func (k *Books) Release(consumer string, t Ticket) {
internal/enchantments/enchantments.go|func ApplyTier(item *items.Item, def *EnchantmentDef, tier int) {
modules/aicompanion/perception.go|carried = append(carried, it.Name())
modules/aicompanion/actions.go|inside = append(inside, ct.Items[i].Name())
modules/baubles/generate.go|_, err = baubles.CleanReply(reply)
modules/baubles/generate.go|if cfg.UsePlayerKeys && req.FinderUserId > 0 {
modules/baubles/generate.go|func (m *BaublesModule) moderate(cfg Config, reply baubles.Reply, playerKey bool) (bool, error) {
modules/baubles/baubles.go|slots chan struct{}
modules/baubles/prompt.go|const PromptVersion = 4
modules/baubles/baubles_test.go|func TestModerationOutageNeverSpoilsAPlayerKeyFind(
_datafiles/config.yaml|MaxConcurrent: 4           # more finds at once than this are generic
docs/aicompanion/settings.md|The key page also has a box, "Also name things I find while searching
docs/baubles/implementation-plan.md|- **Moderation** of a player-key find: a flag refuses; a check that cannot be
EOF
missing=0; while IFS= read -r line; do f=${line%%|*}; a=${line#*|}; grep -qF -- "$a" "$f" || { echo "MISSING in $f: $a"; missing=1; }; done < "$anchors"; rm -f "$anchors"; echo "missing=$missing"
```
Expected: `missing=0`. For a `MISSING` line, STOP before the task that edits it and re-read the code, as Task 0 Step 4 says.

- [ ] **Step 3: Baseline green**

```bash
cd /c/tmp/dogmud-baubles-h && go build ./... && go test ./internal/baubles/ ./internal/items/ ./internal/apiframework/ ./internal/actions/ ./internal/usercommands/ ./internal/enchantments/ ./modules/baubles/ ./modules/aicompanion/ ./modules/gmcp/ . 2>&1 | tail -12
```
Expected: every package `ok`; a failure here is pre-existing, recorded, not this PR's.

---

### Task 6: S3, a bauble never takes an authored item's name

**Files:**
- Modify: `internal/items/itemspec.go` (`authoredWords`, `rebuildAuthoredKeywords`, `AuthoredKeyword`, new `AuthoredName`, `normalizeItemName`)
- Modify: `internal/items/bauble_placement_test.go`
- Modify: `internal/baubles/validate.go` (`CleanReply`, new `authoredName` var)
- Modify: `internal/baubles/validate_test.go`

- [ ] **Step 1: Write the failing items test**

In `internal/items/bauble_placement_test.go`, add after `TestAuthoredKeyword`:

```go
// A loaded item's whole name, compared after NFKC, lower case and collapsed
// spaces, so a model cannot pass a real item's name off by case, spacing or
// fullwidth letters (spec S3). The bauble carrier does not count.
func TestAuthoredName(t *testing.T) {
	restore := SeedItemsForTest(map[int]*ItemSpec{
		10:           {ItemId: 10, Name: `Hooded Lantern`, NameSimple: `lamp`},
		BaubleItemId: {ItemId: BaubleItemId, Name: `Curious Trinket`, NameSimple: `trinket`},
	})
	defer restore()
	for name, want := range map[string]bool{
		`Hooded Lantern`:               true,
		`hooded lantern`:               true,
		"  Hooded\u00a0\t Lantern ":    true,
		"\uFF28ooded \uFF2Cantern":      true, // fullwidth H and L
		`Hooded Lanterns`:              false,
		`Lantern`:                      false,
		`Curious Trinket`:              false,
	} {
		if AuthoredName(name) != want {
			t.Errorf("AuthoredName(%q) = %v, want %v", name, !want, want)
		}
	}
}
```

And in `TestAuthoredKeywordIsSafeWhileItemsAreWritten`, change the reader loop body from `_ = AuthoredKeyword(`lantern`)` to:

```go
			_ = AuthoredKeyword(`lantern`)
			_ = AuthoredName(`Hooded Lantern`)
```

and its final check to:

```go
	if !AuthoredKeyword(`candle`) || !AuthoredKeyword(`lantern`) || !AuthoredName(`test candle`) {
		t.Fatal("the snapshot follows every write")
	}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/items/ -run "TestAuthoredName|TestAuthoredKeyword" -v`
Expected: build failure `undefined: AuthoredName`.

- [ ] **Step 3: One snapshot, words and names together**

In `internal/items/itemspec.go`, add `"golang.org/x/text/unicode/norm"` to the imports. Replace from the `authoredWords` comment through the end of `AuthoredKeyword` with:

```go
// authored is what loaded, authored items answer to: every keyword and word
// of a name (words), and every whole name, normalised (names). It is
// rebuilt by whatever writes the items map (a load, an item saved, created
// or deleted), on the game loop, as ONE snapshot so the two sets always
// agree. AuthoredKeyword and AuthoredName are read from goroutines off the
// mud lock (a bauble being named), and a range over the live items map
// there, racing one of those writes, is a fatal "concurrent map iteration
// and map write"; this snapshot is read instead.
type authoredSnapshot struct {
	words map[string]bool
	names map[string]bool
}

var authored atomic.Pointer[authoredSnapshot]

// rebuildAuthoredKeywords takes the snapshot. Call after every change to
// the items map, where the change is made.
func rebuildAuthoredKeywords() {
	snap := &authoredSnapshot{
		words: make(map[string]bool, len(items)*2),
		names: make(map[string]bool, len(items)),
	}
	for id, spec := range items {
		if id == BaubleItemId || spec == nil {
			continue
		}
		if w := strings.ToLower(strings.TrimSpace(spec.NameSimple)); w != `` {
			snap.words[w] = true
		}
		// Every word of the name, not only its head noun: a bauble keyed
		// "silver" would fully match `get silver` and take it over a real
		// "Silver Dagger", which that word only partly matches.
		for _, f := range strings.Fields(spec.Name) {
			if w := headNoun(f); w != `` {
				snap.words[w] = true
			}
		}
		if n := normalizeItemName(spec.Name); n != `` {
			snap.names[n] = true
		}
	}
	authored.Store(snap)
}

// AuthoredKeyword reports whether word is what a real, authored item
// answers to: its keyword (NameSimple), or any word of its name ("lantern"
// and "hooded" for "Hooded Lantern"). The bauble carrier is not counted. Bauble keywords are kept off these so `get lantern` never picks
// up a model-named trinket instead of the lantern. Safe from any goroutine:
// it reads the snapshot (authored), never the items map.
func AuthoredKeyword(word string) bool {
	snap := authored.Load()
	if snap == nil {
		return false
	}
	return snap.words[strings.ToLower(strings.TrimSpace(word))]
}

// AuthoredName reports whether name is a real, authored item's whole name,
// compared after normalizeItemName. The bauble carrier is not counted. A
// bauble may not take one (baubles.CleanReply). Safe from any goroutine.
func AuthoredName(name string) bool {
	snap := authored.Load()
	if snap == nil {
		return false
	}
	return snap.names[normalizeItemName(name)]
}

// normalizeItemName is a name as AuthoredName compares it: NFKC (fullwidth
// and other compatibility letters folded), lower case, every run of
// Unicode space one plain space, trimmed.
func normalizeItemName(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(norm.NFKC.String(s))), ` `)
}
```

- [ ] **Step 4: Run the items tests under -race**

Run (Bash; `-race` needs cgo, so it runs in the Linux test container, CI being out of minutes):
```bash
cd /c/tmp/dogmud-baubles-h && docker compose -f compose.test.yml run --build --rm test go test -race ./internal/items/ -run "TestAuthored" -v 2>&1 | tail -15
```
Expected: PASS, no `WARNING: DATA RACE`.

- [ ] **Step 5: Write the failing CleanReply test**

Append to `internal/baubles/validate_test.go` (add `"github.com/GoMudEngine/GoMud/internal/items"` to its imports):

```go
// A bauble may not carry a real item's whole name, however it is cased or
// spaced (spec S3): "Hooded Lantern" on a trinket would pass for the real
// one in a shop list or a trade.
func TestCleanReplyRefusesAnAuthoredItemsName(t *testing.T) {
	restore := items.SeedItemsForTest(map[int]*items.ItemSpec{
		10: {ItemId: 10, Name: `Painted Wooden Horse`, NameSimple: `toyhorse`},
	})
	defer restore()
	for _, name := range []string{`Painted Wooden Horse`, `painted  wooden horse`, "Painted\u00a0Wooden Horse"} {
		r := goodReply()
		r.Name = name
		if _, err := CleanReply(r); !errors.Is(err, ErrUnusableReply) {
			t.Errorf("%q is a real item's name: refused, got %v", name, err)
		}
	}
	r := goodReply()
	r.Name = `Painted Wooden Horses`
	if _, err := CleanReply(r); err != nil {
		t.Fatalf("a name that is not a real item's passes: %v", err)
	}
}
```

- [ ] **Step 6: Run to verify it fails**

Run: `go test ./internal/baubles/ -run TestCleanReplyRefusesAnAuthoredItemsName -v`
Expected: FAIL "is a real item's name: refused, got <nil>".

- [ ] **Step 7: Refuse the name**

In `internal/baubles/validate.go`, below `var authoredKeyword = items.AuthoredKeyword` add:

```go
// authoredName is items.AuthoredName. A variable for tests.
var authoredName = items.AuthoredName
```

In `CleanReply`, directly after the `name has digits` check, add:

```go
	if authoredName(r.Name) {
		return bad(`name is a real item's: %s`, quoteShort(r.Name))
	}
```

- [ ] **Step 8: Run to verify it passes, with every baubles and items test**

Run: `go test ./internal/baubles/ ./internal/items/`
Expected: `ok` for both.

- [ ] **Step 9: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/items/itemspec.go internal/items/bauble_placement_test.go internal/baubles/validate.go internal/baubles/validate_test.go && git commit -F - <<'EOF'
fix(baubles): refuse a name that is an authored item's

items.AuthoredName reads the same atomic snapshot as AuthoredKeyword,
comparing after NFKC, lower case and collapsed spaces. CleanReply refuses
a bauble name that matches.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 7: S3, the player-key text allowlist and the engine's invariants

**Files:**
- Create: `internal/baubles/playerkey.go`, `internal/baubles/playerkey_test.go`
- Modify: `internal/baubles/generate.go` (`Generate`)
- Modify: `internal/baubles/generate_test.go`

- [ ] **Step 1: Write the failing allowlist test**

Create `internal/baubles/playerkey_test.go`:

```go
package baubles

import (
	"errors"
	"testing"
)

// Text a player's own key wrote reaches other players, so it is held to
// ASCII letters, spaces and ' " - , . ! ? with every run of periods followed
// by a space, a " or the end (spec S3, ruling 15). Cyrillic lookalikes pass
// CleanReply (they are real letters) and are refused here. The check takes
// CLEANED text: an em dash is refused here and folded to - by CleanReply
// first (TestPlayerKeyTextIsFoldedBeforeTheCheck).
func TestCheckPlayerKeyText(t *testing.T) {
	if err := CheckPlayerKeyText(goodReply()); err != nil {
		t.Fatalf("the good reply is plain text: %v", err)
	}
	cases := map[string]func(*Reply){
		`cyrillic a in the name`: func(r *Reply) { r.Name = "P\u0430inted Wooden Horse" },
		`accent`:                 func(r *Reply) { r.Description = `A café toy horse, its red paint flaking from the mane.` },
		`digit`:                  func(r *Reply) { r.Description = `A toy horse, 3 legs left, its red paint flaking away.` },
		`colon`:                  func(r *Reply) { r.Description = `A toy horse: its red paint is flaking from the mane.` },
		`semicolon`:              func(r *Reply) { r.Description = `A toy horse; its red paint is flaking from the mane.` },
		`uncleaned em dash`:      func(r *Reply) { r.Description = "A toy horse \u2014 its red paint flaking from the mane." },
		`inner period`:           func(r *Reply) { r.Description = `A toy horse.Its red paint is flaking from the mane.` },
		`ellipsis glued on`:      func(r *Reply) { r.Description = `A toy horse...its red paint flaking from the mane.` },
		`material`:               func(r *Reply) { r.Material = `pine/oak` },
		`name simple`:            func(r *Reply) { r.NameSimple = "h\U000000F6rse" },
	}
	for name, change := range cases {
		r := goodReply()
		change(&r)
		if err := CheckPlayerKeyText(r); !errors.Is(err, ErrUnusableReply) {
			t.Errorf("%s must be refused, got %v", name, err)
		}
	}
	r := goodReply()
	r.Description = `Old, chipped - and loved! Whose was it? Nobody's now... "Mine." it says.`
	if err := CheckPlayerKeyText(r); err != nil {
		t.Fatalf("every allowed mark together passes: %v", err)
	}
	// A homoglyph name passes CleanReply: the defence is this check.
	r = goodReply()
	r.Name = "P\u0430inted Wooden Horse"
	if _, err := CleanReply(r); err != nil {
		t.Fatalf("a Cyrillic letter is a letter to CleanReply: %v", err)
	}
}

// Curly quotes, an em dash, an en dash and the ellipsis pass once
// CleanReply has folded them (owner ruling 15): a model's ordinary
// typography does not cost a player-key find its name.
func TestPlayerKeyTextIsFoldedBeforeTheCheck(t *testing.T) {
	r := goodReply()
	r.Name = "Mara\U00002019s Wooden Horse"
	r.Description = "A child\U00002018s toy horse \U00002014 its paint flaking \U00002013 marked \U0000201CMara\U0000201D\U00002026 still loved."
	cleaned, err := CleanReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckPlayerKeyText(cleaned); err != nil {
		t.Fatalf("folded typography passes the allowlist: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/baubles/ -run "TestCheckPlayerKeyText|TestPlayerKeyTextIsFoldedBeforeTheCheck" -v`
Expected: build failure `undefined: CheckPlayerKeyText`.

- [ ] **Step 3: Implement it**

Create `internal/baubles/playerkey.go`:

```go
package baubles

import (
	"fmt"
	"regexp"
)

// playerKeyTextRE is every character text named on a player's own key may
// hold: ASCII letters, the space, and ' " - , . ! ? (spec S3, ruling 15).
// Anything else (digits, accents, lookalike letters from other scripts,
// colons, semicolons, slashes, any other symbol) refuses the reply. The
// text is CLEANED first, so curly quotes, en and em dashes and the ellipsis
// have already been folded to ' " - and ... (cleanLine). " is allowed
// because the fold produces it: it carries no markup once cleanLine has
// stripped tags and < >, and it cannot form a link.
var playerKeyTextRE = regexp.MustCompile(`^[A-Za-z ',.!?"-]*$`)

// CheckPlayerKeyText refuses a CLEANED reply (CleanReply) named on a
// player's own key whose name, keyword, description or material strays
// outside playerKeyTextRE, or has a run of periods followed by anything but
// a space, a " or the end. That text is written by a key the server does
// not control and read by other players, so it is held to plain words as
// well as moderated.
func CheckPlayerKeyText(r Reply) error {
	fields := [...]struct{ field, text string }{
		{`name`, r.Name}, {`name_simple`, r.NameSimple}, {`description`, r.Description}, {`material`, r.Material},
	}
	for _, f := range fields {
		if !playerKeyTextRE.MatchString(f.text) || !periodsEndSentences(f.text) {
			return fmt.Errorf(`%w: player-key %s is not plain text: %s`, ErrUnusableReply, f.field, quoteShort(f.text))
		}
	}
	return nil
}

// periodsEndSentences reports whether every run of periods in s (one, or an
// ellipsis) is followed by a space, a closing " or the end of s, so no
// period sits inside a word the way a domain's does (evil.com, a...b).
func periodsEndSentences(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '.' {
			continue
		}
		j := i
		for j < len(s) && s[j] == '.' {
			j++
		}
		if j < len(s) && s[j] != ' ' && s[j] != '"' {
			return false
		}
		i = j
	}
	return true
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/baubles/ -run "TestCheckPlayerKeyText|TestPlayerKeyTextIsFoldedBeforeTheCheck" -v`
Expected: PASS for both (the second relies on Task 5's `typographyFold`).

- [ ] **Step 5: Write the failing engine test**

Append to `internal/baubles/generate_test.go`:

```go
// The engine holds every generator to the player-key rules, whatever the
// module did (spec S3; owner ruling 2026-09-29): player-key text is plain,
// and either moderated (everyone reads it) or kept to its finder
// (FinderOnly, which needs a finder). Nothing else is ever finder-only.
func TestGenerateHoldsPlayerKeyTextToItsRules(t *testing.T) {
	odd := goodReply()
	odd.Name = "P\u0430inted Wooden Horse"
	refused := map[string]struct {
		res    GenResult
		finder int
	}{
		`unmoderated, not kept to the finder`: {GenResult{Reply: goodReply(), PlayerKey: true}, 7},
		`not plain`:                           {GenResult{Reply: odd, PlayerKey: true, Moderated: true}, 7},
		`not plain, finder-only`:              {GenResult{Reply: odd, PlayerKey: true, FinderOnly: true}, 7},
		`finder-only with no finder`:          {GenResult{Reply: goodReply(), PlayerKey: true, FinderOnly: true}, 0},
		`finder-only on the server's key`:     {GenResult{Reply: goodReply(), FinderOnly: true}, 7},
	}
	for name, c := range refused {
		c := c
		installGenerator(t, func(ctx context.Context, req GenRequest) (GenResult, error) { return c.res, nil })
		if got := Generate(context.Background(), GenRequest{Tier: TierAverage, FinderUserId: c.finder}, nil); got.Generator != GeneratorLocal {
			t.Errorf("%v: a generic trinket, got %+v", name, got)
		}
	}
	kept := map[string]GenResult{
		`moderated`:   {Reply: goodReply(), PlayerKey: true, Moderated: true},
		`finder-only`: {Reply: goodReply(), PlayerKey: true, FinderOnly: true},
	}
	for name, res := range kept {
		res := res
		installGenerator(t, func(ctx context.Context, req GenRequest) (GenResult, error) { return res, nil })
		got := Generate(context.Background(), GenRequest{Tier: TierAverage, FinderUserId: 7}, nil)
		if got.Generator != GeneratorOpenAI || !got.PlayerKey || got.FinderOnly != res.FinderOnly {
			t.Errorf("%v: used as it came: %+v", name, got)
		}
	}
}
```

- [ ] **Step 6: Run to verify it fails**

Run: `go test ./internal/baubles/ -run TestGenerateHoldsPlayerKeyTextToItsRules -v`
Expected: build failure `unknown field FinderOnly in struct literal of type GenResult`.

- [ ] **Step 7: Enforce it in Generate**

In `internal/baubles/generate.go`, in `GenResult`, replace

```go
	PlayerKey     bool // named through the finder's own key, not the server's
}
```
with
```go
	PlayerKey     bool // named through the finder's own key, not the server's

	// FinderOnly is player-key text the server could not moderate: its
	// finder reads it, everyone else the generic trinket (the record derives
	// it: Record.KeptToFinder; owner ruling 2026-09-29). Only ever with
	// PlayerKey and a finder.
	FinderOnly bool
}
```

In `Generate`, replace

```go
	cleaned, err := CleanReply(res.Reply)
	if err != nil {
		mudlog.Warn(`baubles`, `action`, `generate`, `result`, `generic trinket`, `error`, err)
		return generic()
	}
```
with
```go
	cleaned, err := CleanReply(res.Reply)
	if err == nil && (res.PlayerKey || res.FinderOnly) {
		// Text a player's own key wrote is held to plain words, and is
		// either moderated (everyone reads it) or kept to its finder
		// (FinderOnly: the server could not moderate it; owner ruling
		// 2026-09-29). Nothing else is ever finder-only (spec S3).
		switch {
		case !res.PlayerKey:
			err = errors.New(`only text a player's own key wrote is kept to its finder`)
		case res.Moderated:
			res.FinderOnly = false // moderated text is everyone's
			err = CheckPlayerKeyText(cleaned)
		case res.FinderOnly && req.FinderUserId > 0:
			err = CheckPlayerKeyText(cleaned)
		default:
			err = errors.New(`player-key text was neither moderated nor kept to its finder`)
		}
	}
	if err != nil {
		mudlog.Warn(`baubles`, `action`, `generate`, `result`, `generic trinket`, `error`, err)
		return generic()
	}
```
and add `"errors"` to the file's imports.

- [ ] **Step 8: Run all baubles tests, and prove the finder rule can fail**

Run: `go test ./internal/baubles/`
Expected: `ok`.

Probe: temporarily change `case res.FinderOnly && req.FinderUserId > 0:` to `case res.FinderOnly:` and rerun `go test ./internal/baubles/ -run TestGenerateHoldsPlayerKeyTextToItsRules -v`. Expected: FAIL naming `finder-only with no finder`. Restore, rerun, PASS.

- [ ] **Step 9: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/baubles/playerkey.go internal/baubles/playerkey_test.go internal/baubles/generate.go internal/baubles/generate_test.go && git commit -F - <<'EOF'
fix(baubles): player-key text is plain, and moderated or finder-only

CheckPlayerKeyText holds player-key text to ASCII letters and simple
punctuation. baubles.Generate refuses any player-key result that fails it,
or is neither moderated nor kept to a real finder (GenResult.FinderOnly),
whatever the generator did.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 8: S3, the server rolls a player-key find's value

**Files:**
- Modify: `internal/baubles/mint.go`
- Modify: `internal/baubles/generate_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/baubles/generate_test.go`:

```go
// A player's own key proposes a value the server does not trust, even
// clamped: Mint rolls it in the tier instead, keeping the proposal for the
// record (spec S3). A server-key value is kept, clamped.
func TestMintRollsAPlayerKeyFindsValue(t *testing.T) {
	withCatalog(t)
	r := goodReply()
	r.Value = 14 // inside average (10 to 15), so a clamp alone would keep it
	res := GenResult{Reply: r, Generator: GeneratorOpenAI, Moderated: true, PlayerKey: true}
	_, rec, err := Mint(MintOpts{Source: SourceSearch, Place: NewPlace(1, `z`, ``, `city`), Tier: TierAverage, Result: &res, Randn: first})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Value != TierAverage.Range().Min || rec.ValueProposed != 14 || !rec.PlayerKey {
		t.Fatalf("rolled by the server (first die: the tier's minimum), proposal kept: %+v", rec)
	}

	res.PlayerKey = false
	_, rec, err = Mint(MintOpts{Source: SourceSearch, Place: NewPlace(1, `z`, ``, `city`), Tier: TierAverage, Result: &res, Randn: first})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Value != 14 {
		t.Fatalf("a server-key value stands: %+v", rec)
	}
}
```

(`first` is `catalog_test.go:50`, returning 0; `TierAverage` is 10 to 15 per `TestGenerateOnTheServersKey`'s `10 to 15` prompt check.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/baubles/ -run TestMintRollsAPlayerKeyFindsValue -v`
Expected: FAIL "rolled by the server".

- [ ] **Step 3: Roll it**

In `internal/baubles/mint.go`, replace

```go
	limited := ApplyLimitsFor(res.Reply, tier, source)
```
with
```go
	limited := ApplyLimitsFor(res.Reply, tier, source)
	if res.PlayerKey {
		// A value a player's own key proposed is not trusted, even clamped
		// into the tier: the server rolls it. ValueProposed keeps what the
		// key said, for the record (spec S3).
		limited.Reply.Value = tier.RollValue(randn)
	}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/baubles/`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/baubles/mint.go internal/baubles/generate_test.go && git commit -F - <<'EOF'
fix(baubles): the server rolls a player-key find's value

Mint ignores a player key's proposed value and rolls one in the tier,
keeping the proposal in ValueProposed.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 9: S3, player-key names never reach another prompt

**Files:**
- Modify: `internal/baubles/generate.go` (`RecentNames`), `internal/baubles/admin.go` (`ApplyRegenerated`)
- Modify: `internal/baubles/generate_test.go`, `internal/baubles/admin_test.go`
- Modify: `internal/actions/bauble_admin.go` (`BaubleRequestForRecord`), `internal/actions/bauble_admin_test.go`

- [ ] **Step 1: Write the failing engine tests**

Append to `internal/baubles/generate_test.go`:

```go
// A name a player's own key wrote never goes into another find's prompt
// (spec S3): RecentNames is sent to the model for every later find in the
// zone, whoever's key names it.
func TestRecentNamesSkipsPlayerKeyNames(t *testing.T) {
	withCatalog(t)
	_, _ = Create(Record{Name: `Old Cup`, Zone: `ashwick`, Generator: GeneratorOpenAI})
	_, _ = Create(Record{Name: `Player Written`, Zone: `ashwick`, Generator: GeneratorOpenAI, PlayerKey: true})
	got := RecentNames(`ashwick`, 5)
	if len(got) != 1 || got[0] != `Old Cup` {
		t.Fatalf("server-key names only: %v", got)
	}
}
```

Append to `internal/baubles/admin_test.go`:

```go
// Regenerating takes the new result's key, both ways (spec S3).
func TestApplyRegeneratedSetsPlayerKey(t *testing.T) {
	withCatalog(t)
	r := seedRecord(t, Record{Name: `Trinket`, NameSimple: `trinket`, Tier: TierAverage, Value: 11, Status: StatusReady, Generator: GeneratorOpenAI, PlayerKey: true})
	got, err := ApplyRegenerated(r.Id, GenResult{Reply: goodReply(), Generator: GeneratorOpenAI, Moderated: true}, `Admin`)
	if err != nil {
		t.Fatal(err)
	}
	if got.PlayerKey {
		t.Fatal("named again on the server's key: no longer a player-key record")
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/baubles/ -run "TestRecentNamesSkipsPlayerKeyNames|TestApplyRegeneratedSetsPlayerKey" -v`
Expected: FAIL for both.

- [ ] **Step 3: Implement**

In `internal/baubles/generate.go` `RecentNames`, replace

```go
		if r.Zone == zone && r.Generator == GeneratorOpenAI {
```
with
```go
		// Never a name a player's own key wrote: this list goes into
		// every later find's prompt, on anyone's key (spec S3).
		if r.Zone == zone && r.Generator == GeneratorOpenAI && !r.PlayerKey {
```

and update its doc comment's first line to `RecentNames returns up to n names of server-key model-named baubles found in the zone,`.

In `internal/baubles/admin.go` `ApplyRegenerated`, after `r.Moderated = res.Moderated` add:

```go
		r.PlayerKey = res.PlayerKey
```

- [ ] **Step 4: Run to verify they pass**

Run: `go test ./internal/baubles/`
Expected: `ok`.

- [ ] **Step 5: Write the failing actions test**

Append to `internal/actions/bauble_admin_test.go`:

```go
// A player-key record's name is not sent back to the model as a name to
// avoid when it is regenerated (spec S3): it is a player's text.
func TestBaubleRequestForRecordOmitsAPlayerKeyName(t *testing.T) {
	seedBaubleSale(t)
	rec, err := baubles.Create(baubles.Record{
		Name: "Player Written Cup", NameSimple: "cup", Tier: baubles.TierAverage, Value: 12,
		Description: "A cup a player's own key described.", Status: baubles.StatusReady,
		Generator: baubles.GeneratorOpenAI, PlayerKey: true, RoomId: 424243, Zone: "nowhere",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range BaubleRequestForRecord(rec).RecentNames {
		if n == rec.Name {
			t.Fatalf("a player-key name went into the request: %v", BaubleRequestForRecord(rec).RecentNames)
		}
	}
}
```

- [ ] **Step 6: Run to verify it fails**

Run: `go test ./internal/actions/ -run TestBaubleRequestForRecordOmitsAPlayerKeyName -v`
Expected: FAIL "a player-key name went into the request".

- [ ] **Step 7: Implement**

In `internal/actions/bauble_admin.go`, replace

```go
	req.RecentNames = append(req.RecentNames, rec.Name)
	return req
```
with
```go
	// A name a player's own key wrote is never sent to a model (spec S3).
	if !rec.PlayerKey {
		req.RecentNames = append(req.RecentNames, rec.Name)
	}
	return req
```

and change the doc comment's third sentence to `The record's own name is added to the names to avoid, so a regeneration asks for something new, unless a player's own key wrote it.`

- [ ] **Step 8: Run to verify it passes**

Run: `go test ./internal/actions/ -run "Bauble" -v 2>&1 | tail -20`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/baubles/generate.go internal/baubles/generate_test.go internal/baubles/admin.go internal/baubles/admin_test.go internal/actions/bauble_admin.go internal/actions/bauble_admin_test.go && git commit -F - <<'EOF'
fix(baubles): player-key names never reach another prompt

RecentNames skips player-key records, a regeneration request omits a
player-key record's name, and ApplyRegenerated takes the new result's
PlayerKey.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 9a: Finder-only text in the record, the catalog view and the item layer

Owner ruling 2026-09-29 (amendment, ruling 2): player-key text the server
could not moderate is kept to its finder. This task makes that fail-safe:
the catalog's viewer-agnostic view of such a record IS the generic trinket,
so every existing render path (about 156 room sends, the templates, GMCP,
shops, auctions) shows "Trinket" with no change, and four new item
accessors give the finder's view to whoever asks for one viewer.

**Files:**
- Create: `internal/items/bauble_viewer.go`, `internal/items/bauble_viewer_test.go`
- Modify: `internal/items/bauble.go` (`BaubleView`, `baubleSpec`, new `baubleSpecFor`, new `baubleFinderNames`, `matchStrength`)
- Modify: `internal/items/items.go` (`DisplayName`, `GetLongDescription`, `NameMatch`)
- Create: `internal/baubles/record_test.go`
- Modify: `internal/baubles/record.go` (new `KeptToFinder`, `View`, new `MaterialFor`), `internal/baubles/fallback.go` (new `genericDescriptionFor`), `internal/baubles/generate_test.go`
- Modify: `internal/enchantments/enchantments.go` (`ApplyTier`); Create: `internal/enchantments/bauble_test.go`

- [ ] **Step 1: Write the failing item-layer tests**

Create `internal/items/bauble_viewer_test.go`:

```go
package items

import (
	"strings"
	"testing"
)

// A finder-only bauble (text a player's own key wrote that the server could
// not moderate; owner ruling 2026-09-29) reads as the generic trinket
// through every viewer-agnostic accessor, so no render path built on them,
// today's or a new one, can show its text to anyone. Only the viewer-aware
// accessors (bauble_viewer.go) show it, and only to its finder.
func TestFinderOnlyBaubleIsGenericToEveryoneButItsFinder(t *testing.T) {
	restore := SeedItemsForTest(map[int]*ItemSpec{
		BaubleItemId: {ItemId: BaubleItemId, Name: `Curious Trinket`, NameSimple: `trinket`, Type: Object, Subtype: Mundane},
	})
	defer restore()
	SetBaubleResolver(func(id string) (BaubleView, bool) {
		return BaubleView{
			Name: `Trinket`, NameSimple: `trinket`, Description: `A small trinket of no particular make.`, Value: 12, WeightLbs: 0.2,
			FinderUserId: 7,
			Finder: &BaubleView{Name: `Painted Wooden Horse`, NameSimple: `horse`,
				Description: `A child's toy horse, its red paint flaking.`, Value: 12, WeightLbs: 0.2},
		}, id == `b1`
	})
	defer SetBaubleResolver(nil)

	itm := New(BaubleItemId)
	itm.Bauble = `b1`
	spec := itm.GetSpec()
	for accessor, text := range map[string]string{
		`GetSpec().Name`: spec.Name, `GetSpec().NameSimple`: spec.NameSimple, `GetSpec().Description`: spec.Description,
		`Name`: itm.Name(), `NameSimple`: itm.NameSimple(), `DisplayName`: itm.DisplayName(),
		`NameComplex`: itm.NameComplex(), `GetLongDescription`: itm.GetLongDescription(),
	} {
		if strings.Contains(strings.ToLower(text), `horse`) {
			t.Errorf("%s shows the finder's own text to everyone: %q", accessor, text)
		}
	}
	for _, viewer := range []int{0, 8} {
		if got := itm.DisplayNameFor(viewer); got != `Trinket` {
			t.Errorf("viewer %d reads the generic name, got %q", viewer, got)
		}
		if got := itm.NameFor(viewer) + itm.LongDescriptionFor(viewer) + itm.GetSpecFor(viewer).NameSimple; strings.Contains(strings.ToLower(got), `horse`) {
			t.Errorf("viewer %d reads the finder's text: %q", viewer, got)
		}
	}
	if itm.DisplayNameFor(7) != `Painted Wooden Horse` || itm.NameFor(7) != `Painted Wooden Horse` ||
		itm.GetSpecFor(7).NameSimple != `horse` || !strings.Contains(itm.LongDescriptionFor(7), `toy horse`) {
		t.Fatalf("the finder reads their own text: %q %q", itm.DisplayNameFor(7), itm.LongDescriptionFor(7))
	}
	// The finder types the words they read; matching shows nobody any text.
	if part, _ := itm.NameMatch(`horse`, true); !part {
		t.Error("the finder's word matches the trinket")
	}
	if _, full := itm.NameMatch(`trinket`, true); !full {
		t.Error("the generic keyword still names it in full")
	}
}

// For anything that is not a finder-only bauble, each viewer-aware accessor
// is its viewer-agnostic twin.
func TestViewerAccessorsAreTheirTwinsForAnyOtherItem(t *testing.T) {
	restore := SeedItemsForTest(map[int]*ItemSpec{
		10: {ItemId: 10, Name: `Hooded Lantern`, NameSimple: `lamp`, Description: `A lantern with a hood.`, Type: Object, Subtype: Mundane},
	})
	defer restore()
	itm := New(10)
	if itm.DisplayNameFor(7) != itm.DisplayName() || itm.NameFor(7) != itm.Name() ||
		itm.LongDescriptionFor(7) != itm.GetLongDescription() || itm.GetSpecFor(7).Name != itm.GetSpec().Name {
		t.Fatal("an ordinary item reads the same to every viewer")
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/items/ -run "TestFinderOnlyBaubleIsGenericToEveryoneButItsFinder|TestViewerAccessorsAreTheirTwinsForAnyOtherItem" -v`
Expected: build failure `unknown field FinderUserId in struct literal of type BaubleView` (and `itm.DisplayNameFor undefined`).

- [ ] **Step 3: The finder's view in the item layer**

In `internal/items/bauble.go`, replace the `BaubleView` struct

```go
type BaubleView struct {
	Name        string
	NameSimple  string
	DisplayName string
	Description string
	Value       int
	WeightLbs   float64
}
```
with
```go
type BaubleView struct {
	Name        string
	NameSimple  string
	DisplayName string
	Description string
	Value       int
	WeightLbs   float64

	// FinderUserId and Finder make a finder-only view (internal/baubles
	// Record.KeptToFinder, owner ruling 2026-09-29): the fields above are what
	// everyone sees, Finder what player FinderUserId sees, and only through
	// the viewer-aware accessors (bauble_viewer.go). Nil Finder: everyone
	// sees the fields above.
	FinderUserId int
	Finder       *BaubleView
}
```

Replace the head of `baubleSpec`

```go
func baubleSpec(base ItemSpec, id string) ItemSpec {
	p := baubleResolver.Load()
	if p == nil || *p == nil {
		return base
	}
	v, ok := (*p)(id)
	if !ok {
		return base
	}
	if v.Name != `` {
```
with
```go
func baubleSpec(base ItemSpec, id string) ItemSpec {
	return baubleSpecFor(base, id, 0)
}

// baubleSpecFor is baubleSpec as viewerUserId sees it: a finder-only
// bauble's own text for its finder, the generic view for everyone else.
// Viewer 0 is nobody, so baubleSpec is always the generic view.
func baubleSpecFor(base ItemSpec, id string, viewerUserId int) ItemSpec {
	p := baubleResolver.Load()
	if p == nil || *p == nil {
		return base
	}
	v, ok := (*p)(id)
	if !ok {
		return base
	}
	if v.Finder != nil && viewerUserId > 0 && viewerUserId == v.FinderUserId {
		v = *v.Finder
	}
	if v.Name != `` {
```

In `matchStrength`, directly after

```go
	names := []string{util.NormalizeForMatch(i.Name()), util.NormalizeForMatch(i.NameSimple()), withoutPossessives(i.Name())}
```
add
```go
	names = append(names, i.baubleFinderNames()...)
```

Add, directly above `// anyBauble reports whether any item in the list is a bauble.`:

```go
// baubleFinderNames is a finder-only bauble's own name and keyword,
// normalised for matching, so its finder can type the words they read
// (`drop horse`). Matching shows nobody any text. Nil for any other item.
func (i *Item) baubleFinderNames() []string {
	if i.Bauble == `` {
		return nil
	}
	p := baubleResolver.Load()
	if p == nil || *p == nil {
		return nil
	}
	v, ok := (*p)(i.Bauble)
	if !ok || v.Finder == nil {
		return nil
	}
	return []string{util.NormalizeForMatch(v.Finder.Name), util.NormalizeForMatch(v.Finder.NameSimple), withoutPossessives(v.Finder.Name)}
}

```

In `internal/items/items.go`, `NameMatch`, replace

```go
		wordPart, wordFull := baubleWordMatch(input, simpleName, displayName, withoutPossessives(i.Name()))
```
with
```go
		// A finder-only bauble also answers to the words its finder reads.
		wordPart, wordFull := baubleWordMatch(input, append([]string{simpleName, displayName, withoutPossessives(i.Name())}, i.baubleFinderNames()...)...)
```

Replace the head of `GetLongDescription`

```go
func (i *Item) GetLongDescription() string {

	iSpec := i.GetSpec()
```
with
```go
func (i *Item) GetLongDescription() string {
	return i.longDescriptionFrom(i.GetSpec())
}

// longDescriptionFrom is GetLongDescription over a spec already resolved
// (GetSpec, or GetSpecFor for one viewer).
func (i *Item) longDescriptionFrom(iSpec ItemSpec) string {
```

Replace the head of `DisplayName`

```go
func (i *Item) DisplayName() string {
	if i.ItemId < 1 { // Used to represent item slots that are disabled
```
with
```go
func (i *Item) DisplayName() string {
	return i.displayNameFrom(i.GetSpec())
}

// displayNameFrom is DisplayName over a spec already resolved (GetSpec, or
// GetSpecFor for one viewer).
func (i *Item) displayNameFrom(spec ItemSpec) string {
	if i.ItemId < 1 { // Used to represent item slots that are disabled
```
and, in the same function, replace `	if i.GetSpec().QuestToken != `` {` with `	if spec.QuestToken != `` {`, and delete the line `	spec := i.GetSpec()` that sits directly above the comment `// Normalize the bare template name to canonical smart Title case.` (the other `spec := i.GetSpec()` in `items.go`, near line 215, stays).

Create `internal/items/bauble_viewer.go`:

```go
package items

// Viewer-aware accessors (owner ruling 2026-09-29). A bauble named on a
// player's own key while the server could not moderate it is finder-only:
// every viewer-agnostic accessor (GetSpec, Name, NameSimple, DisplayName,
// NameComplex, GetLongDescription, and so every template and GMCP payload
// built on them) shows the generic trinket, and these show its own text to
// its finder alone. Call them only where the output reaches that one
// viewer: the repo-root guard (bauble_finder_view_guard_test.go) lists
// every caller. For any other item each is its viewer-agnostic twin.

// GetSpecFor is GetSpec as viewerUserId sees it.
func (i *Item) GetSpecFor(viewerUserId int) ItemSpec {
	if i.Spec != nil || i.Bauble == `` {
		return i.GetSpec()
	}
	iSpec := GetItemSpec(i.ItemId)
	if iSpec == nil {
		iSpec = &ItemSpec{}
	}
	return baubleSpecFor(*iSpec, i.Bauble, viewerUserId)
}

// DisplayNameFor is DisplayName as viewerUserId sees it.
func (i *Item) DisplayNameFor(viewerUserId int) string {
	if i.Bauble == `` {
		return i.DisplayName()
	}
	return i.displayNameFrom(i.GetSpecFor(viewerUserId))
}

// NameFor is Name as viewerUserId sees it.
func (i *Item) NameFor(viewerUserId int) string {
	if i.Bauble == `` || i.ItemId < 1 {
		return i.Name()
	}
	return i.GetSpecFor(viewerUserId).Name
}

// LongDescriptionFor is GetLongDescription as viewerUserId sees it.
func (i *Item) LongDescriptionFor(viewerUserId int) string {
	if i.Bauble == `` {
		return i.GetLongDescription()
	}
	return i.longDescriptionFrom(i.GetSpecFor(viewerUserId))
}
```

- [ ] **Step 4: Run to verify they pass, and prove the agnostic half can fail**

Run: `go test ./internal/items/`
Expected: `ok`.

Probe: in `baubleSpecFor`, temporarily change `if v.Finder != nil && viewerUserId > 0 && viewerUserId == v.FinderUserId {` to `if v.Finder != nil {` and rerun `go test ./internal/items/ -run TestFinderOnlyBaubleIsGenericToEveryoneButItsFinder -v`. Expected: FAIL listing `GetSpec().Name`, `Name`, `DisplayName`, `NameComplex`, `GetLongDescription` and the others as showing the finder's text to everyone. Restore, rerun, PASS.

- [ ] **Step 5: Write the failing record tests**

Create `internal/baubles/record_test.go`:

```go
package baubles

import (
	"strings"
	"testing"
)

// A finder-only record (owner ruling 2026-09-29) shows the generic trinket
// to everyone and its own text to its finder alone; its value and weight
// are the item's for everyone. Retired text wins over both. Finder-only is
// DERIVED (KeptToFinder: PlayerKey and not Moderated), never stored, so a
// record written before this slice, which has no such field, is kept to its
// finder too (review finding a).
func TestFinderOnlyRecordView(t *testing.T) {
	// Exactly the shape the pre-slice moderate left: a player's own key,
	// accepted unmoderated.
	r := Record{Id: `b0000001`, Status: StatusReady, Name: `Painted Wooden Horse`, NameSimple: `horse`,
		Description: `A child's toy horse, its red paint flaking.`, Material: `pine`, Value: 12, WeightLbs: 0.3,
		FoundByUserId: 7, PlayerKey: true, Moderated: false}
	if !r.KeptToFinder() {
		t.Fatal("unmoderated player-key text is its finder's alone")
	}
	v := r.View()
	if !v.PlayerText {
		t.Fatal("player-key text is marked, so no model prompt carries it (Task 9c)")
	}
	if v.Name != genericName || v.NameSimple != genericNameSimple || strings.Contains(v.Description, `horse`) {
		t.Fatalf("everyone sees the generic trinket: %+v", v)
	}
	if v.Value != 12 || v.WeightLbs != 0.3 {
		t.Fatalf("the numbers are the item's for everyone: %+v", v)
	}
	if v.FinderUserId != 7 || v.Finder == nil || v.Finder.Name != r.Name || v.Finder.Description != r.Description || v.Finder.Finder != nil {
		t.Fatalf("the finder's own view: %+v", v.Finder)
	}
	if again := r.View(); again.Description != v.Description {
		t.Fatal("the generic description is the same every time")
	}
	if r.MaterialFor(7) != `pine` || r.MaterialFor(8) != `` || r.MaterialFor(0) != `` {
		t.Fatal("the material is the finder's alone")
	}

	r.FoundByUserId = 0 // nobody to keep it for: nobody reads it
	if v := r.View(); v.Finder != nil || v.Name != genericName {
		t.Fatalf("no finder, no finder view: %+v", v)
	}
	r.FoundByUserId, r.Status = 7, StatusRetired
	if v := r.View(); v.Finder != nil || v.Name != retiredName {
		t.Fatalf("retired text wins: %+v", v)
	}
	r.Status, r.Moderated = StatusReady, true
	if v := r.View(); v.Name != r.Name || v.Finder != nil || r.MaterialFor(8) != `pine` || !v.PlayerText {
		t.Fatalf("moderated player-key text is everyone's, and still never a model's: %+v", v)
	}
	r.PlayerKey = false
	if v := r.View(); v.Name != r.Name || v.Finder != nil || v.PlayerText || r.KeptToFinder() {
		t.Fatalf("server-key text is everyone's: %+v", v)
	}
}
```

Append to `internal/baubles/generate_test.go`:

```go
// A finder-only result reaches the record kept to the finder Mint records
// (owner ruling 2026-09-29), whatever GenResult.FinderOnly said: the record
// derives it from PlayerKey and Moderated. A regeneration, always on the
// server's key, makes it everyone's.
func TestFinderOnlyReachesTheRecordAndRegenClearsIt(t *testing.T) {
	withCatalog(t)
	// FinderOnly deliberately left false: the record must not trust it.
	res := GenResult{Reply: goodReply(), Generator: GeneratorOpenAI, PlayerKey: true, Moderated: false}
	_, rec, err := Mint(MintOpts{Source: SourceSearch, Place: NewPlace(1, `z`, ``, `city`), FinderUserId: 7, Tier: TierAverage, Result: &res, Randn: first})
	if err != nil {
		t.Fatal(err)
	}
	if !rec.KeptToFinder() || rec.FoundByUserId != 7 || rec.View().Finder == nil {
		t.Fatalf("finder-only, kept to user 7: %+v", rec)
	}
	got, err := ApplyRegenerated(rec.Id, GenResult{Reply: goodReply(), Generator: GeneratorOpenAI, Moderated: true}, `Admin`)
	if err != nil {
		t.Fatal(err)
	}
	if got.KeptToFinder() || got.PlayerKey || got.View().Finder != nil {
		t.Fatalf("named again on the server's key: everyone's: %+v", got)
	}
}
```

- [ ] **Step 6: Run to verify they fail**

Run: `go test ./internal/baubles/ -run "TestFinderOnlyRecordView|TestFinderOnlyReachesTheRecordAndRegenClearsIt" -v`
Expected: build failure `r.KeptToFinder undefined` (and `r.MaterialFor undefined`, `v.PlayerText undefined`).

- [ ] **Step 7: The record**

Finder-only is derived, not stored (review finding a): a stored flag would leave every record the pre-slice `moderate` accepted unmoderated (`PlayerKey` true, `Moderated` false, and no new field) public, and would trust whatever a generator put in `GenResult.FinderOnly`. `Mint` and `ApplyRegenerated` already copy `PlayerKey` and `Moderated`, so they need no change for it.

In `internal/items/bauble.go` `BaubleView`, add below the `Finder` field Step 3 added:

```go
	// PlayerText: the record's text was written by a player's own key,
	// moderated or not. Such text never goes into any language model's
	// prompt (spec S3): ModelName and ModelDescription (Task 9c) show the
	// carrier's own name instead.
	PlayerText bool
```

In `internal/baubles/record.go`, directly above `// Text shown for a bauble whose text an admin has withdrawn.`, add:

```go
// KeptToFinder reports whether the record's text is its finder's alone:
// text a player's own key wrote that was never moderated (owner ruling
// 2026-09-29). Derived, never stored, so a record named before the rule
// (PlayerKey, Moderated false) is kept to its finder too. Never promotable
// to the corpus either way (slice C takes only server-key moderated text).
func (r Record) KeptToFinder() bool {
	return r.PlayerKey && !r.Moderated
}

```

In `View`, replace

```go
		v.Description = retiredDescription
	}
	return v
}
```
with
```go
		v.Description = retiredDescription
		return v
	}
	v.PlayerText = r.PlayerKey
	if r.KeptToFinder() {
		// Text a player's own key wrote that was never moderated (owner
		// ruling 2026-09-29): its finder reads it through the item layer's
		// viewer-aware accessors, everyone else the generic trinket.
		own := v
		v.Name, v.NameSimple, v.Description = genericName, genericNameSimple, genericDescriptionFor(r.Id)
		if r.FoundByUserId > 0 {
			v.FinderUserId, v.Finder = r.FoundByUserId, &own
		}
	}
	return v
}

// MaterialFor is the material as viewerUserId may read it (appraise): a
// finder-only record's is its finder's alone, like its name (View).
func (r Record) MaterialFor(viewerUserId int) string {
	if r.KeptToFinder() && (viewerUserId <= 0 || viewerUserId != r.FoundByUserId) {
		return ``
	}
	return r.Material
}
```

`own` is copied after `PlayerText` is set, so the finder's view carries it too (`v.Finder.Finder` stays nil).

In `internal/baubles/fallback.go`, replace

```go
package baubles

// The generic trinket: what every bauble is when it is not named by the
```
with
```go
package baubles

import "hash/fnv"

// The generic trinket: what every bauble is when it is not named by the
```
and append at the end of the file:

```go

// genericDescriptionFor is one of genericDescriptions, the same one every
// time for the same record id: what everyone but its finder reads for a
// finder-only bauble (Record.View).
func genericDescriptionFor(id string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	return genericDescriptions[h.Sum32()%uint32(len(genericDescriptions))]
}
```

`Mint` copies `PlayerKey` and `Moderated` from the result and `ApplyRegenerated` takes both (Task 9 added `PlayerKey` there), so neither needs a line for finder-only: the record derives it. A regen is always server-key (`BaubleRequestForRecord` carries no finder), so it makes the text everyone's.

- [ ] **Step 8: Run to verify they pass, and prove the derivation**

Run: `go test ./internal/baubles/ ./internal/items/`
Expected: `ok` for both.

Probe: temporarily make `KeptToFinder` return `false` and rerun `go test ./internal/baubles/ -run "TestFinderOnlyRecordView|TestFinderOnlyReachesTheRecordAndRegenClearsIt" -v`. Expected: FAIL "unmoderated player-key text is its finder's alone" and "finder-only, kept to user 7". Restore, rerun, PASS.

- [ ] **Step 8b: No bauble is ever enchanted (its text would be baked)**

`enchantments.ApplyTier` writes `item.Spec = &newSpec` built from `GetSpec()` (`internal/enchantments/enchantments.go:203`), and an item with a `Spec` never consults the catalog again (`items.go:325-327`), so an enchanted bauble would carry its text baked in, past a retire, and past the finder-only view. Decided: baubles are never enchantment targets. Today no path offers one (`TargetType` is a weapon or armour slot, and a bauble is an `Object`), so this is a guard, not a behaviour change.

Create `internal/enchantments/bauble_test.go`:

```go
package enchantments

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// A bauble is never enchanted: ApplyTier would bake its catalog text into
// Item.Spec, past a retire and past the finder-only view (slice H).
func TestApplyTier_NeverTouchesABauble(t *testing.T) {
	def := &EnchantmentDef{EnchantId: "test-edge", Tiers: []TierDef{{Tier: 0, Adjective: "gleaming"}}}
	itm := items.Item{ItemId: items.BaubleItemId, Bauble: "b0000001"}
	ApplyTier(&itm, def, 0)
	if itm.Spec != nil || len(itm.Adjectives) != 0 {
		t.Fatalf("the bauble is untouched: spec %v adjectives %v", itm.Spec != nil, itm.Adjectives)
	}
}
```

Run: `go test ./internal/enchantments/ -run TestApplyTier_NeverTouchesABauble -v`
Expected: FAIL "the bauble is untouched: spec true".

In `internal/enchantments/enchantments.go` `ApplyTier`, replace

```go
func ApplyTier(item *items.Item, def *EnchantmentDef, tier int) {
	if tier < 0 || tier >= len(def.Tiers) {
		return
	}
```
with
```go
func ApplyTier(item *items.Item, def *EnchantmentDef, tier int) {
	if tier < 0 || tier >= len(def.Tiers) {
		return
	}
	// A bauble's text lives in the catalog and may be one viewer's alone;
	// a baked Spec would outlive both (slice H). Never enchanted.
	if item.IsBauble() {
		return
	}
```

Run: `go test ./internal/enchantments/`
Expected: `ok`.

- [ ] **Step 9: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/items/bauble.go internal/items/items.go internal/items/bauble_viewer.go internal/items/bauble_viewer_test.go internal/baubles/record.go internal/baubles/record_test.go internal/baubles/fallback.go internal/baubles/generate_test.go internal/enchantments/enchantments.go internal/enchantments/bauble_test.go && git commit -F - <<'EOF'
feat(baubles): finder-only text reads as a trinket to everyone else

Record.KeptToFinder (PlayerKey and not Moderated, derived, so records
named before this slice are covered) keeps unmoderated player-key text to
its finder: the catalog's view of such a record is the generic trinket,
with the finder's own view beside it, so every viewer-agnostic item
accessor, and every render path built on one, shows "Trinket".
GetSpecFor, DisplayNameFor, NameFor and LongDescriptionFor give one
viewer's view; the finder can still type the words they read.
BaubleView.PlayerText marks player-key text for Task 9c. Baubles are never
enchanted, so their text is never baked into Item.Spec.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 9b: The finder's own view at the single-reader sites, and the guard

Ten functions show a bauble to the one player who holds, finds or stands
over it: the find lines, the pickpocket success line, inventory, look,
the room listing, appraise, and four per-user GMCP payloads (the backpack,
the bandolier, the component bag, `Room.Info.Contents.Items`). Each asks
for that player's view; every other render path keeps the viewer-agnostic
accessors, which Task 9a made show the generic trinket.

**Files:**
- Modify: `internal/actions/search_bauble.go` (`BaubleDelivery.deliver`), `internal/actions/steal.go` (`takeFromMob`)
- Modify: `internal/usercommands/inventory.go` (`Inventory`), `internal/usercommands/look.go` (`Look`, `lookRoom`), `internal/usercommands/appraise.go` (`appraiseBauble`)
- Modify: `modules/gmcp/gmcp.Char.go` (`GMCPCharModule.GetCharNode`, `buildBandolierContainer`, `buildComponentBagContainer`), `modules/gmcp/gmcp.Room.go` (`GMCPRoomModule.GetRoomNode`)
- Create: `bauble_finder_view_guard_test.go`, `internal/usercommands/finder_only_bauble_test.go`

- [ ] **Step 1: Write the failing guard and its probe**

Create `bauble_finder_view_guard_test.go`:

```go
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// finderViewSites is every function allowed to read a bauble's text as one
// viewer sees it (items.Item GetSpecFor, DisplayNameFor, NameFor,
// LongDescriptionFor; baubles.Record MaterialFor), keyed "path|function"
// as lookupFuncName names it, each with why its output reaches that viewer
// alone. A finder-only bauble's own text (owner ruling 2026-09-29) is read
// through these and nowhere else: every viewer-agnostic accessor shows the
// generic trinket (internal/items
// TestFinderOnlyBaubleIsGenericToEveryoneButItsFinder), so a new render
// path that uses them cannot leak it, and this guard stops a new caller of
// the finder's view from reaching anyone else.
//
// Two rules, so neither a new function nor a new line inside a listed one
// slips past (review finding c):
//   - Each listed function makes exactly `calls` finder-view calls. A new
//     call in Look (which also talks to another player) changes the count
//     and fails until someone reads it and updates the row.
//   - No finder-view call may sit inside anything that sends beyond its one
//     reader: a room send (roomSendSelectors), a mob's Command (a `say`),
//     merchantSay, an events.Message literal, or a SendText whose receiver
//     is not provably the viewer (the receiver's root identifier must be
//     the viewer argument's: user.SendText(..., x.NameFor(user.UserId))).
//
// KNOWN LIMITS (each is a way a real leak could pass):
//   - A finder-view value kept in a variable and later sent is not traced;
//     the call count makes every such variable a reviewed line.
//   - The five method names are matched by name, not by type, and only
//     inside function bodies (not package-level initialisers).
//   - internal/items and internal/baubles define the accessors and are not
//     scanned.
type finderViewSite struct {
	calls int
	why   string
}

var finderViewSites = map[string]finderViewSite{
	"internal/actions/search_bauble.go|BaubleDelivery.deliver":    {1, "the find's own lines, sent to the finder alone (who.send); the room line names no item"},
	"internal/actions/steal.go|takeFromMob":                       {3, "the thief's own success line (actor.SendText); the room is not told what was taken"},
	"internal/usercommands/appraise.go|appraiseBauble":            {4, "the appraisal, sent to the player who asked for it (user.SendText); the room line names no item"},
	"internal/usercommands/inventory.go|Inventory":                {2, "the player's own inventory listing"},
	"internal/usercommands/look.go|Look":                          {4, "what the looker reads about an item they carry or one on the floor; the room lines beside them keep DisplayName"},
	"internal/usercommands/look.go|lookRoom":                      {2, "the looker's own view of the room's floor and their own stash"},
	"modules/gmcp/gmcp.Char.go|GMCPCharModule.GetCharNode":        {1, "the player's own Char.Inventory backpack"},
	"modules/gmcp/gmcp.Char.go|buildBandolierContainer":           {1, "the player's own bandolier payload (GetCharNode passes user.UserId)"},
	"modules/gmcp/gmcp.Char.go|buildComponentBagContainer":        {1, "the player's own component bag payload (GetCharNode passes user.UserId)"},
	"modules/gmcp/gmcp.Room.go|GMCPRoomModule.GetRoomNode":        {1, "Room.Info.Contents.Items, built for one user and sent to that user"},
}

// finderViewSelectors are the viewer-aware accessors.
var finderViewSelectors = map[string]bool{
	"GetSpecFor": true, "DisplayNameFor": true, "NameFor": true, "LongDescriptionFor": true, "MaterialFor": true,
}

// beyondReaderCalls send text beyond one reader, by callee name, whatever
// the receiver: the room sends, a mob's Command (a `say`), merchantSay.
var beyondReaderCalls = map[string]bool{
	"SendTextCommunication": true, "SendTextVisual": true, "SendTextVisualHidingNames": true,
	"SendTextVisualAsLit": true, "SendTextVisualAsLitHidingNames": true, "SendTextVisualWithAudio": true,
	"SendTextToExits": true, "SendRoomCommunication": true, "SendTrio": true, "Command": true, "merchantSay": true,
}

// rootIdent is the identifier an expression hangs from: user for
// user.UserId, actor for actor.GetUserId(), "" when there is none.
func rootIdent(e ast.Expr) string {
	for {
		switch x := e.(type) {
		case *ast.Ident:
			return x.Name
		case *ast.SelectorExpr:
			e = x.X
		case *ast.CallExpr:
			e = x.Fun
		case *ast.ParenExpr:
			e = x.X
		case *ast.StarExpr:
			e = x.X
		default:
			return ""
		}
	}
}

// calleeName is a call's function name: Sel for a selector, the identifier
// for a plain call.
func calleeName(call *ast.CallExpr) (string, *ast.SelectorExpr) {
	switch f := call.Fun.(type) {
	case *ast.SelectorExpr:
		return f.Sel.Name, f
	case *ast.Ident:
		return f.Name, nil
	}
	return "", nil
}

// finderViewCallsIn returns every finder-view call within n.
func finderViewCallsIn(n ast.Node) []*ast.CallExpr {
	var out []*ast.CallExpr
	ast.Inspect(n, func(m ast.Node) bool {
		if c, ok := m.(*ast.CallExpr); ok {
			if name, _ := calleeName(c); finderViewSelectors[name] {
				out = append(out, c)
			}
		}
		return true
	})
	return out
}

type finderViewFile struct {
	rel  string
	file *ast.File
}

// scanFinderView returns how many finder-view calls each function makes,
// and every such call that sits inside a send beyond its one reader.
func scanFinderView(fset *token.FileSet, files []finderViewFile) (calls map[string]int, leaks []string) {
	calls = map[string]int{}
	for _, f := range files {
		for _, decl := range f.file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			key := f.rel + "|" + lookupFuncName(fd)
			report := func(c *ast.CallExpr, via string) {
				name, _ := calleeName(c)
				leaks = append(leaks, fmt.Sprintf("%s:%d in %s: %s inside %s",
					f.rel, fset.Position(c.Pos()).Line, key, name, via))
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.CompositeLit:
					// events.Message{UserId: ..., Text: ...}: queued to some
					// user, not provably the viewer.
					if sel, ok := x.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Message" {
						for _, c := range finderViewCallsIn(x) {
							report(c, "events.Message")
						}
					}
				case *ast.CallExpr:
					name, sel := calleeName(x)
					if finderViewSelectors[name] {
						calls[key]++
					}
					switch {
					case beyondReaderCalls[name]:
						for _, arg := range x.Args {
							for _, c := range finderViewCallsIn(arg) {
								report(c, name)
							}
						}
					case name == "SendText" && sel != nil:
						receiver := rootIdent(sel.X)
						if inner, ok := sel.X.(*ast.CallExpr); ok {
							if n, _ := calleeName(inner); n == "GetRoom" {
								receiver = "" // a room reached through its reader is still a room
							}
						}
						if s, ok := sel.X.(*ast.SelectorExpr); ok && strings.HasSuffix(strings.ToLower(s.Sel.Name), "room") {
							receiver = "" // who.room.SendText, any xxxRoom field
						}
						if id, ok := sel.X.(*ast.Ident); ok && strings.HasSuffix(strings.ToLower(id.Name), "room") {
							receiver = "" // room, fromRoom, markRoom: a room variable
						}
						for _, arg := range x.Args {
							for _, c := range finderViewCallsIn(arg) {
								if len(c.Args) == 0 || receiver == "" || rootIdent(c.Args[0]) != receiver {
									report(c, receiver+".SendText")
								}
							}
						}
					}
				}
				return true
			})
		}
	}
	return calls, leaks
}

// TestFinderViewReachesOnlyItsReader fails when a function not in
// finderViewSites reads a bauble as one viewer sees it, when a listed
// function no longer does (stale), or when any finder-view call sits inside
// a room send's arguments. If you are here for a new single-reader site
// (the output goes to one player, and only them), add it with the reason.
// If the output reaches anyone else, use the viewer-agnostic accessor,
// which shows a finder-only bauble as the generic trinket.
func TestFinderViewReachesOnlyItsReader(t *testing.T) {
	fset := token.NewFileSet()
	var files []finderViewFile
	scanned := map[string]bool{}
	for _, root := range []string{"internal", "modules"} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			rel := filepath.ToSlash(path)
			if d.IsDir() {
				if rel == "internal/items" || rel == "internal/baubles" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
				return nil
			}
			file, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if perr != nil {
				return nil
			}
			scanned[rel] = true
			files = append(files, finderViewFile{rel: rel, file: file})
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s (test must run from the repo root): %v", root, err)
		}
	}
	for _, must := range []string{"internal/usercommands/look.go", "internal/usercommands/get.go", "modules/gmcp/gmcp.Char.go", "modules/auctions/auctions.go"} {
		if !scanned[must] {
			t.Fatalf("the walk never read %s: it cannot see what it guards", must)
		}
	}

	calls, leaks := scanFinderView(fset, files)
	var problems []string
	for key, got := range calls {
		want, ok := finderViewSites[key]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("%s: makes %d finder-view call(s), and is not in finderViewSites", key, got))
		case got != want.calls:
			problems = append(problems, fmt.Sprintf("%s: makes %d finder-view call(s), finderViewSites says %d: read each one before changing the row", key, got, want.calls))
		}
	}
	for key := range finderViewSites {
		if calls[key] == 0 {
			problems = append(problems, key+": in finderViewSites but reads no finder view (stale)")
		}
	}
	problems = append(problems, leaks...)
	sort.Strings(problems)
	if len(problems) > 0 {
		t.Errorf("%d finder-view problem(s):\n  %s", len(problems), strings.Join(problems, "\n  "))
	}
}

// TestFinderViewGuardCatchesALeak proves the scan can fail on every shape
// it claims to catch, and passes the one private shape.
func TestFinderViewGuardCatchesALeak(t *testing.T) {
	fset := token.NewFileSet()
	src := `package probe

func Shout(room *rooms.Room, itm items.Item, uid int) {
	room.SendTextVisual(1, fmt.Sprintf("%s", itm.DisplayNameFor(uid)))
}

func Mine(user *users.UserRecord, itm items.Item) { user.SendText(1, itm.NameFor(user.UserId)) }

func ToAnother(user *users.UserRecord, u *users.UserRecord, itm items.Item) {
	u.SendText(1, itm.NameFor(user.UserId))
}

func Chained(user *users.UserRecord, itm items.Item) { user.GetRoom().SendText(1, itm.NameFor(user.UserId)) }

func Says(room *rooms.Room, mob *mobs.Mob, itm items.Item, uid int) {
	merchantSay(room, mob, itm.NameFor(uid))
	mob.Command("say " + itm.NameFor(uid))
}

func Queued(itm items.Item, uid int) {
	events.AddToQueue(events.Message{UserId: 2, Text: itm.NameFor(uid)})
}
`
	file, err := parser.ParseFile(fset, "probe.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	calls, leaks := scanFinderView(fset, []finderViewFile{{rel: "probe.go", file: file}})
	if calls["probe.go|Says"] != 2 || calls["probe.go|Mine"] != 1 || len(calls) != 6 {
		t.Fatalf("every call counted, per function: %v", calls)
	}
	want := []string{"Shout", "ToAnother", "Chained", "Says", "Says", "Queued"}
	if len(leaks) != len(want) {
		t.Fatalf("want %d leaks (%v), got:\n  %s", len(want), want, strings.Join(leaks, "\n  "))
	}
	for _, l := range leaks {
		if strings.Contains(l, "|Mine:") || strings.Contains(l, " in probe.go|Mine") {
			t.Fatalf("the player's own SendText is private: %v", l)
		}
	}
}
```

`user.GetRoom().SendText` roots at `user`, so the receiver check alone would pass it; the `GetRoom` and `...room` checks in the `SendText` branch catch it, `who.room.SendText`, and every `xxxRoom.SendText` (13 such sites at `3bd6ccaa3`).

Run: `go test . -run "TestFinderViewReachesOnlyItsReader|TestFinderViewGuardCatchesALeak" -v`
Expected: `TestFinderViewGuardCatchesALeak` PASS (the scan works on the probe); `TestFinderViewReachesOnlyItsReader` FAIL listing all ten `finderViewSites` keys as stale (no site calls a finder view yet).

- [ ] **Step 2: Write the failing command test**

Create `internal/usercommands/finder_only_bauble_test.go`:

```go
package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/require"
)

// A finder-only bauble (owner ruling 2026-09-29): its finder reads its own
// text in `inventory` and `look`; anyone else carrying it reads "Trinket".
func TestFinderOnlyBaubleReadsByViewer(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	// Midsummer noon, so look is not refused as blind (look_item_noun_test.go).
	cfg := configs.GetConfig()
	cfg.Timing.RoundsPerDay = 20
	configs.SetConfigForTest(t, cfg)
	gametime.ClearDateCacheForTest()
	t.Cleanup(gametime.ClearDateCacheForTest)
	util.SetRoundCountForTest(uint64(3430))
	t.Cleanup(util.ResetRoundCountForTest)

	restoreItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		items.BaubleItemId: {ItemId: items.BaubleItemId, Name: "Curious Trinket", NameSimple: "trinket",
			Type: items.Object, Subtype: items.Mundane, Weight: 0.2, Value: 1, NotSalable: true},
	})
	defer restoreItems()
	baubles.SetDirForTest(t.TempDir())
	defer items.SetBaubleResolver(nil)

	user, room := getTestUserAndRoom(t)
	origItems := user.Character.Items
	defer func() { user.Character.Items = origItems }()

	for _, finder := range []int{user.UserId, user.UserId + 1000} {
		rec, err := baubles.Create(baubles.Record{Name: "Painted Wooden Horse", NameSimple: "horse", Tier: baubles.TierCheap,
			Value: 3, WeightLbs: 0.5, Description: "A child's toy horse, its red paint flaking.", Status: baubles.StatusReady,
			PlayerKey: true, Moderated: false, FoundByUserId: finder})
		require.NoError(t, err)
		itm := items.New(items.BaubleItemId)
		itm.Bauble = rec.Id
		user.Character.Items = []items.Item{itm}

		events.DrainQueuedMessagesForTest(user.UserId)
		_, _ = Inventory("", user, room, 0)
		_, _ = Look("trinket", user, room, 0)
		out := strings.Join(events.DrainQueuedMessagesForTest(user.UserId), "\n")
		mine := finder == user.UserId
		if strings.Contains(out, "Painted Wooden Horse") != mine || strings.Contains(out, "toy horse") != mine {
			t.Errorf("found by %d, read by %d: the horse shows %v, want %v:\n%s", finder, user.UserId, !mine, mine, out)
		}
		require.Contains(t, out, "You look at the", "look reached the carried trinket")
	}
}
```

Run: `go test ./internal/usercommands/ -run TestFinderOnlyBaubleReadsByViewer -v`
Expected: FAIL on the finder's pass ("found by 1, read by 1: the horse shows false, want true"): Task 9a made every existing path show "Trinket", including to the finder. The stranger's pass already holds. If `You look at the` is missing, the room is not lit or `look trinket` did not reach the item: fix the fixture before Step 3.

- [ ] **Step 3: Give each single-reader site the finder's view**

`internal/actions/search_bauble.go`, `BaubleDelivery.deliver`: replace

```go
	name := itm.DisplayName()
```
with
```go
	// Every line naming it goes to the finder alone (who.send): their own
	// view of a finder-only bauble. The room line below names no item.
	name := itm.DisplayNameFor(d.UserId)
```

`internal/actions/steal.go`, `takeFromMob`: replace

```go
			stolenStuff = append(stolenStuff,
				fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, itemStolen.DisplayName()))
```
with
```go
			stolenStuff = append(stolenStuff,
				fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, itemStolen.DisplayNameFor(actor.GetUserId())))
```
and replace
```go
					fmt.Sprintf(`<ansi fg="itemname">%s</ansi> (too much to carry: it falls at your feet)`, b.DisplayName()))
```
with
```go
					fmt.Sprintf(`<ansi fg="itemname">%s</ansi> (too much to carry: it falls at your feet)`, b.DisplayNameFor(actor.GetUserId())))
```
and replace
```go
			stolenStuff = append(stolenStuff,
				fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, b.DisplayName()))
```
with
```go
			stolenStuff = append(stolenStuff,
				fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, b.DisplayNameFor(actor.GetUserId())))
```
(`stolenStuff` goes only into the thief's own success line; `result.StoleItemName` keeps `DisplayName`.)

`internal/usercommands/inventory.go`, `Inventory`: replace `for _, part := range util.BreakIntoParts(item.Name()) {` with `for _, part := range util.BreakIntoParts(item.NameFor(user.UserId)) {`, and replace

```go
		iName := item.Name()
		iNameFormatted := fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, item.Name())

		if isSpoiled {
			iName = fmt.Sprintf(`%s (turned)`, item.Name())
			iNameFormatted = fmt.Sprintf(`<ansi fg="8">%s (turned)</ansi>`, item.Name())
```
with
```go
		baseName := item.NameFor(user.UserId) // their own view of a finder-only bauble
		iName := baseName
		iNameFormatted := fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, baseName)

		if isSpoiled {
			iName = fmt.Sprintf(`%s (turned)`, baseName)
			iNameFormatted = fmt.Sprintf(`<ansi fg="8">%s (turned)</ansi>`, baseName)
```

`internal/usercommands/look.go`, `Look`: replace

```go
			fmt.Sprintf(`You look at the <ansi fg="item">%s</ansi> %s:`, lookItem.DisplayName(), lookDestination),
```
with
```go
			fmt.Sprintf(`You look at the <ansi fg="item">%s</ansi> %s:`, lookItem.DisplayNameFor(user.UserId), lookDestination),
```
replace `		itemDesc := lookItem.GetLongDescription()` with `		itemDesc := lookItem.LongDescriptionFor(user.UserId)`; replace

```go
			fmt.Sprintf(`You look at the <ansi fg="item">%s</ansi> %s:`, floorItem.DisplayName(), where),
```
with
```go
			fmt.Sprintf(`You look at the <ansi fg="item">%s</ansi> %s:`, floorItem.DisplayNameFor(user.UserId), where),
```
and replace `			util.SplitStringNL(floorItem.GetLongDescription(), 80),` with `			util.SplitStringNL(floorItem.LongDescriptionFor(user.UserId), 80),`. The two room lines beside them (`is admiring their`, `is looking at the <ansi fg="item">%s</ansi> %s.`) keep `DisplayName()`.

`internal/usercommands/look.go`, `lookRoom`: replace

```go
			groundStacks[key] = &groundStack{name: item.DisplayName() + item.BaubleSpotSuffix(), count: 1}
```
with
```go
			groundStacks[key] = &groundStack{name: item.DisplayNameFor(user.UserId) + item.BaubleSpotSuffix(), count: 1}
```
and replace
```go
		name := item.DisplayName() + ` <ansi fg="item-stashed">(stashed)</ansi>`
```
with
```go
		name := item.DisplayNameFor(user.UserId) + ` <ansi fg="item-stashed">(stashed)</ansi>`
```

`internal/usercommands/appraise.go`, `appraiseBauble`: replace `	spec := item.GetSpec()` with `	spec := item.GetSpecFor(user.UserId) // the appraisal reaches this player alone`; replace both `item.DisplayName()` calls in the function (the "turns ... over in their hands" line and the `fmt.Fprintf(&b, "<ansi fg=\"itemname\">%s</ansi>\r\n", ...)` line) with `item.DisplayNameFor(user.UserId)`; and replace

```go
	if rec.Material != `` {
		fmt.Fprintf(&b, "  Made of:  %s\r\n", rec.Material)
	}
```
with
```go
	if material := rec.MaterialFor(user.UserId); material != `` {
		fmt.Fprintf(&b, "  Made of:  %s\r\n", material)
	}
```

`modules/gmcp/gmcp.Char.go`, `GetCharNode`: replace

```go
			payload.Inventory.Backpack.Items = append(payload.Inventory.Backpack.Items, newInventory_Item(itm))
```
with
```go
			d := newInventory_Item(itm)
			d.Name = itm.NameFor(user.UserId) // their own view of a finder-only bauble
			payload.Inventory.Backpack.Items = append(payload.Inventory.Backpack.Items, d)
```

and, in the same function, replace

```go
			Bandolier:    buildBandolierContainer(user.Character),
			ComponentBag: buildComponentBagContainer(user.Character),
```
with
```go
			Bandolier:    buildBandolierContainer(user.Character, user.UserId),
			ComponentBag: buildComponentBagContainer(user.Character, user.UserId),
```

Change `func buildBandolierContainer(c *characters.Character) *GMCPCharModule_Payload_Container {` to `func buildBandolierContainer(c *characters.Character, viewerUserId int) *GMCPCharModule_Payload_Container {` and its loop body `cont.Items = append(cont.Items, newInventory_Item(itm))` (the one over `c.PotionItems`) to:

```go
		d := newInventory_Item(itm)
		d.Name = itm.NameFor(viewerUserId) // the wearer's own view
		cont.Items = append(cont.Items, d)
```
and do the same for `buildComponentBagContainer` (signature with `viewerUserId int`, the loop over `contents`). Those are their only callers (`gmcp.Char.go:523-524`; no test calls them).

`modules/gmcp/gmcp.Room.go`, `GMCPRoomModule.GetRoomNode` (built for one `user`, sent to that user): replace

```go
				Name:      itm.Name(),
				QuestFlag: itm.GetSpec().QuestToken != ``,
```
with
```go
				Name:      itm.NameFor(user.UserId), // their own view of a finder-only bauble
				QuestFlag: itm.GetSpec().QuestToken != ``,
```

- [ ] **Step 4: Run to verify both tests pass**

Run: `go build ./... && go test . -run "TestFinderViewReachesOnlyItsReader|TestFinderViewGuardCatchesALeak" -v && go test ./internal/usercommands/ -run TestFinderOnlyBaubleReadsByViewer -v`
Expected: PASS, PASS, PASS.

- [ ] **Step 5: Prove the guard fails on a real leak**

In `internal/usercommands/look.go`, temporarily change the floor-item room line's `floorItem.DisplayName()` (in `is looking at the <ansi fg="item">%s</ansi> %s.`) to `floorItem.DisplayNameFor(user.UserId)`. Run `go test . -run TestFinderViewReachesOnlyItsReader -v`. Expected: FAIL naming both `internal/usercommands/look.go|Look: makes 5 finder-view call(s), finderViewSites says 4` and `internal/usercommands/look.go:<line> in internal/usercommands/look.go|Look: DisplayNameFor inside SendTextVisual`. Then also try `u.SendText(messaging.CategoryMobEmote, lookItem.NameFor(user.UserId))` inside the look-at-a-player branch (`u` is the looked-at player, `look.go:95`): expected FAIL `NameFor inside u.SendText`. Restore (`git diff internal/usercommands/look.go` shows only Step 3's lines), rerun, PASS.

- [ ] **Step 6: Run the touched packages and the root guards**

Run: `go test ./internal/actions/ ./internal/usercommands/ ./modules/gmcp/ . 2>&1 | tail -8`
Expected: all `ok`.

- [ ] **Step 7: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add bauble_finder_view_guard_test.go internal/usercommands/finder_only_bauble_test.go internal/actions/search_bauble.go internal/actions/steal.go internal/usercommands/inventory.go internal/usercommands/look.go internal/usercommands/appraise.go modules/gmcp/gmcp.Char.go modules/gmcp/gmcp.Room.go && git commit -F - <<'EOF'
feat(baubles): a finder reads their own finder-only bauble

The find lines, the pickpocket success line, inventory, look, the room
listing, the appraisal and four per-user GMCP payloads ask for the
reader's own view. A root guard pins the finder-view call count of each of
those ten functions and fails on any other caller, or on a finder-view
call inside a room send, a Command, merchantSay, an events.Message, or a
SendText to anyone but the viewer.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 9c: Player-key text never reaches the AI companion's model

Review finding f. The AI companion describes what it perceives to its
model: its own gear and pack (`perception.go:184, 194`), what lies in the
room (`perception.go:213`, `scene.go:149`), its owner's weapon
(`perception.go:293`), its carried and worn things (`scene.go:278, 288`),
a take-from line (`actions.go:508`), a looked-at item's description
(`actions.go:624`) and a container's contents (`actions.go:661`). A
finder-only bauble already reads "Trinket" there (Task 9a), but MODERATED
player-key text is everyone's, and spec S3 keeps player-key text out of
every model prompt, not only the baubles module's. So these sites use two
model-safe accessors that show any player-key bauble as its carrier's own
name ("Curious Trinket") and description.

**Files:**
- Create: `internal/items/bauble_model.go`
- Modify: `internal/items/bauble_viewer_test.go`
- Modify: `modules/aicompanion/perception.go`, `modules/aicompanion/scene.go`, `modules/aicompanion/actions.go`
- Create: `modules/aicompanion/bauble_prompt_test.go`

- [ ] **Step 1: Write the failing item test**

Append to `internal/items/bauble_viewer_test.go`:

```go
// Player-key text, moderated or not, never goes into a language model's
// prompt (spec S3): ModelName and ModelDescription show such a bauble as
// its carrier ("Curious Trinket"), and are Name and GetLongDescription for
// anything else.
func TestModelAccessorsKeepPlayerKeyTextOut(t *testing.T) {
	restore := SeedItemsForTest(map[int]*ItemSpec{
		BaubleItemId: {ItemId: BaubleItemId, Name: `Curious Trinket`, NameSimple: `trinket`,
			Description: `A small curiosity.`, Type: Object, Subtype: Mundane},
	})
	defer restore()
	views := map[string]BaubleView{
		`player`: {Name: `Painted Wooden Horse`, NameSimple: `horse`, Description: `A child's toy horse.`, PlayerText: true},
		`server`: {Name: `Tin Soldier`, NameSimple: `soldier`, Description: `A dented tin soldier.`},
	}
	SetBaubleResolver(func(id string) (BaubleView, bool) { v, ok := views[id]; return v, ok })
	defer SetBaubleResolver(nil)

	mine := New(BaubleItemId)
	mine.Bauble = `player`
	if mine.ModelName() != `Curious Trinket` || strings.Contains(mine.ModelDescription(), `horse`) {
		t.Fatalf("player-key text stays out of the prompt: %q %q", mine.ModelName(), mine.ModelDescription())
	}
	if mine.Name() != `Painted Wooden Horse` {
		t.Fatal("players still read moderated player-key text")
	}
	server := New(BaubleItemId)
	server.Bauble = `server`
	if server.ModelName() != `Tin Soldier` || !strings.Contains(server.ModelDescription(), `tin soldier`) {
		t.Fatalf("server-key text is the model's to read: %q", server.ModelName())
	}
}
```

Run: `go test ./internal/items/ -run TestModelAccessorsKeepPlayerKeyTextOut -v`
Expected: build failure `mine.ModelName undefined`.

- [ ] **Step 2: The accessors**

Create `internal/items/bauble_model.go`:

```go
package items

// Model-safe accessors. Text a player's own key wrote, moderated or not,
// never goes into any language model's prompt (spec S3; the baubles
// module's RecentNames and regeneration keep it out of theirs, and these
// keep it out of the AI companion's). Such a bauble reads as its carrier
// item's own spec ("Curious Trinket"). Use these, never Name or
// GetLongDescription, for anything a model is told.

// ModelName is Name for a model's prompt.
func (i *Item) ModelName() string {
	if spec, ok := i.playerTextCarrier(); ok {
		return spec.Name
	}
	return i.Name()
}

// ModelDescription is GetLongDescription for a model's prompt.
func (i *Item) ModelDescription() string {
	if spec, ok := i.playerTextCarrier(); ok {
		return i.longDescriptionFrom(spec)
	}
	return i.GetLongDescription()
}

// playerTextCarrier is the carrier's own spec when this is a bauble whose
// text a player's own key wrote.
func (i *Item) playerTextCarrier() (ItemSpec, bool) {
	if i.Bauble == `` {
		return ItemSpec{}, false
	}
	p := baubleResolver.Load()
	if p == nil || *p == nil {
		return ItemSpec{}, false
	}
	v, ok := (*p)(i.Bauble)
	if !ok || !v.PlayerText {
		return ItemSpec{}, false
	}
	spec := GetItemSpec(i.ItemId)
	if spec == nil {
		return ItemSpec{}, false
	}
	return *spec, true
}
```

Run: `go test ./internal/items/`
Expected: `ok`.

- [ ] **Step 3: Write the failing companion test**

Create `modules/aicompanion/bauble_prompt_test.go`:

```go
package aicompanion

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// What the companion is told about a room never carries text a player's
// own key wrote (spec S3; review finding f): a player-key bauble lying
// there reads as its carrier.
func TestRoomThingsKeepPlayerKeyTextOut(t *testing.T) {
	t.Cleanup(items.SeedItemsForTest(map[int]*items.ItemSpec{
		items.BaubleItemId: {ItemId: items.BaubleItemId, Name: `Curious Trinket`, NameSimple: `trinket`,
			Type: items.Object, Subtype: items.Mundane},
	}))
	items.SetBaubleResolver(func(id string) (items.BaubleView, bool) {
		return items.BaubleView{Name: `Painted Wooden Horse`, NameSimple: `horse`, PlayerText: true}, true
	})
	t.Cleanup(func() { items.SetBaubleResolver(nil) })

	b := items.New(items.BaubleItemId)
	b.Bauble = `b0000001`
	room := &rooms.Room{RoomId: 1, Items: []items.Item{b}}
	got := strings.Join(roomThings(room), `, `)
	if strings.Contains(got, `Horse`) || !strings.Contains(got, `Curious Trinket`) {
		t.Fatalf("the companion reads the carrier, never the player's text: %q", got)
	}
}
```

Run: `go test ./modules/aicompanion/ -run TestRoomThingsKeepPlayerKeyTextOut -v`
Expected: FAIL "the companion reads the carrier, never the player's text: \"Painted Wooden Horse\"".

- [ ] **Step 4: Route every model-facing item name**

Replace `.Name()` with `.ModelName()` at each of these (the receiver varies; the expression is otherwise unchanged):

| File | Text today |
|---|---|
| `modules/aicompanion/perception.go` | `worn = append(worn, it.Name())` |
| `modules/aicompanion/perception.go` | `carried = append(carried, it.Name())` |
| `modules/aicompanion/perception.go` | `out = append(out, it.Name())` (in `roomThings`) |
| `modules/aicompanion/perception.go` | ``parts = append(parts, `carrying `+w.Name())`` |
| `modules/aicompanion/scene.go` | ``Kind: `item`, Name: item.Name(), Class: `lying here`,`` |
| `modules/aicompanion/scene.go` | ``sc.Carried = append(sc.Carried, thing{Ref: fmt.Sprintf(`p%d`, i+1), Kind: `carried`, Name: item.Name(),`` |
| `modules/aicompanion/scene.go` | ``sc.Worn = append(sc.Worn, thing{Ref: fmt.Sprintf(`w%d`, wi), Kind: `worn`, Name: item.Name(),`` |
| `modules/aicompanion/actions.go` | `t.Key, it.Name(), t.Name, delay, round)` (the `take_from` issue) |
| `modules/aicompanion/actions.go` | `inside = append(inside, ct.Items[i].Name())` |

and in `actions.go` `lookAt` replace `		desc = plainText(item.GetLongDescription())` with `		desc = plainText(item.ModelDescription())`.

Then list what is left, to read each by eye:
```bash
cd /c/tmp/dogmud-baubles-h && grep -n "\.Name()\|\.DisplayName()\|GetLongDescription()" modules/aicompanion/*.go | grep -v _test
```
Expected at `3bd6ccaa3` plus this step: only `combat.go:650` (a weapon's name: a bauble is never a weapon), `economy.go:98, 113` (shop wares: shops never stock a bauble, `sell_bauble.go`), `inventory.go:32` (keyword matching, not sent) and the `profile.go` directory entries (`e.Name()`, files). Anything else that names an item to the model: route it the same way, and say so in the commit.

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./modules/aicompanion/ ./internal/items/`
Expected: `ok` for both.

Probe: temporarily make `playerTextCarrier` return `ItemSpec{}, false` and rerun both new tests. Expected: FAIL in each. Restore, rerun, PASS.

- [ ] **Step 6: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/items/bauble_model.go internal/items/bauble_viewer_test.go modules/aicompanion/perception.go modules/aicompanion/scene.go modules/aicompanion/actions.go modules/aicompanion/bauble_prompt_test.go && git commit -F - <<'EOF'
fix(aicompanion): player-key bauble text never reaches the model

items.Item ModelName and ModelDescription show a bauble whose text a
player's own key wrote, moderated or not, as its carrier; every item name
and description the AI companion tells its model goes through them.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 9d: `bauble show` and `bauble list` tell an admin whose text it is

Review finding e. `bauble show` (`admin.bauble.go:170-199`) prints no
`PlayerKey`, `Moderated` or finder-only state, and `bauble list`
(`:203-223`) cannot find the records an admin most needs to review.

**Files:**
- Modify: `internal/usercommands/admin.bauble.go` (`baubleShow`'s builder, `baubleList`)
- Modify: `_datafiles/world/dogmud/templates/admincommands/help/command.bauble.template`
- Modify: `internal/usercommands/admin.bauble_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/usercommands/admin.bauble_test.go` (it imports `strings`, `baubles`, `items`, `assert`, `require`; add `"github.com/GoMudEngine/GoMud/internal/events"`):

```go
// An admin sees whose text a bauble carries (review finding e), and can
// list only the player-key, the unmoderated or the finder-only ones.
func TestAdminBauble_ShowsAndFiltersPlayerKeyText(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	baubles.SetDirForTest(t.TempDir())
	defer items.SetBaubleResolver(nil)
	admin, room := getTestUserAndRoom(t)

	mine, err := baubles.Create(baubles.Record{Name: "Painted Wooden Horse", NameSimple: "horse", Tier: baubles.TierCheap,
		Value: 3, WeightLbs: 0.5, Description: "A toy horse.", Status: baubles.StatusReady,
		Generator: baubles.GeneratorOpenAI, PlayerKey: true, FoundByUserId: 7})
	require.NoError(t, err)
	_, err = baubles.Create(baubles.Record{Name: "Tin Soldier", NameSimple: "soldier", Tier: baubles.TierCheap,
		Value: 3, WeightLbs: 0.5, Description: "A tin soldier.", Status: baubles.StatusReady,
		Generator: baubles.GeneratorOpenAI, Moderated: true})
	require.NoError(t, err)

	events.DrainQueuedMessagesForTest(admin.UserId)
	_, _ = Bauble("show "+mine.Id, admin, room, 0)
	out := strings.Join(events.DrainQueuedMessagesForTest(admin.UserId), "\n")
	assert.Contains(t, out, "player key: yes")
	assert.Contains(t, out, "moderated: no")
	assert.Contains(t, out, "finder only: yes (user 7)")

	for filter, wantHorse := range map[string]bool{"playerkey": true, "unmoderated": true, "finderonly": true, "": true} {
		_, _ = Bauble(strings.TrimSpace("list 10 "+filter), admin, room, 0)
		out = strings.Join(events.DrainQueuedMessagesForTest(admin.UserId), "\n")
		assert.Equal(t, wantHorse, strings.Contains(out, "Painted Wooden Horse"), filter)
		assert.Equal(t, filter == "", strings.Contains(out, "Tin Soldier"), filter)
	}
}
```

Run: `go test ./internal/usercommands/ -run TestAdminBauble_ShowsAndFiltersPlayerKeyText -v`
Expected: FAIL on "player key: yes" and on the filtered lists still showing the Tin Soldier.

- [ ] **Step 2: Show and filter**

In `internal/usercommands/admin.bauble.go`, replace

```go
	fmt.Fprintf(&b, "  generator:   %s %s (prompt v%d, %d tokens)\r\n", rec.Generator, rec.Model, rec.PromptVersion, rec.Tokens)
```
with
```go
	fmt.Fprintf(&b, "  generator:   %s %s (prompt v%d, %d tokens)\r\n", rec.Generator, rec.Model, rec.PromptVersion, rec.Tokens)
	fmt.Fprintf(&b, "  player key: %s, moderated: %s\r\n", yesNo(rec.PlayerKey), yesNo(rec.Moderated))
	if rec.KeptToFinder() {
		fmt.Fprintf(&b, "  finder only: yes (user %d); everyone else sees a plain Trinket\r\n", rec.FoundByUserId)
	}
```
and add at the end of the file:

```go

// yesNo is a flag for the admin bauble views.
func yesNo(v bool) string {
	if v {
		return `yes`
	}
	return `no`
}
```
(no `yesNo` exists in `internal/usercommands` at `3bd6ccaa3`.)

In `baubleList`, replace

```go
	n := 10
	if len(args) > 0 {
		if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
			n = v
		}
	}
	recent := baubles.Recent(n)
```
with
```go
	n := 10
	filter := ``
	for _, a := range args {
		if v, err := strconv.Atoi(a); err == nil && v > 0 {
			n = v
			continue
		}
		filter = strings.ToLower(a)
	}
	keep := func(r baubles.Record) bool {
		switch filter {
		case `playerkey`:
			return r.PlayerKey
		case `unmoderated`:
			return !r.Moderated && r.Generator == baubles.GeneratorOpenAI
		case `finderonly`:
			return r.KeptToFinder()
		}
		return true
	}
	recent := make([]baubles.Record, 0, n)
	for _, r := range baubles.Recent(baubles.Count()) {
		if len(recent) >= n {
			break
		}
		if keep(r) {
			recent = append(recent, r)
		}
	}
```
(`baubles.Recent(n int) []Record` and `baubles.Count() int`, `catalog.go:226, 233`; the loop below the old line, `for _, r := range recent`, is unchanged.)

In `command.bauble.template`, change the `bauble list [n]` line to `<ansi fg="command">bauble list [n] [playerkey|unmoderated|finderonly]</ansi>` and add under its description: `  Only baubles named on a player's own key, those never moderated, or those shown to their finder alone.`

- [ ] **Step 3: Run to verify it passes**

Run: `go test ./internal/usercommands/ -run "TestAdminBauble" -v 2>&1 | grep -E "^(--- FAIL|--- PASS|FAIL|ok)"`
Expected: PASS for both admin bauble tests.

- [ ] **Step 4: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/usercommands/admin.bauble.go internal/usercommands/admin.bauble_test.go _datafiles/world/dogmud/templates/admincommands/help/command.bauble.template && git commit -F - <<'EOF'
feat(bauble): admins see and filter player-key and finder-only text

bauble show prints whose key named the text, whether it was moderated and
whether it is kept to its finder; bauble list takes a playerkey,
unmoderated or finderonly filter.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 10: S3, the module's moderation policy, allowlist fallback and finder-only text

Task 0 Step 4 confirmed every block quoted below still reads as quoted at
`3bd6ccaa3`. This task changes `name`'s player branch, `moderate`'s body
and signature, the lines of `generate` around `CleanReply` and
`moderate`, the status text and the system prompt, and adds one breaker
function to `apiframework`.

The policy (spec S3 as amended by owner ruling 2026-09-29, amendment
ruling 2). The finder's own key is asked whenever they allowed it; there
is no pre-check closing that route any more (the original plan's
`playerRouteOpen` is dropped). After the answer:

- Moderation possible (`moderationPossible`: `ModerateOutput` on, a server
  key, neither the provider breaker nor the moderation breaker open):
  the name, keyword, description and material are moderated. Clean:
  `Moderated`, everyone reads it. Flagged: refused. The check made and
  failing: refused. Every check made, on either route, is recorded on the
  moderation breaker (`moderationBreaker`, `apiframework.RecordConsumer`),
  never the provider's and never the naming breaker `ConsumerBaubles`; a
  clean or flagged answer counts as a success there.
- Moderation impossible: no call is made, and the find is `FinderOnly`:
  the finder reads it, everyone else the generic trinket (Task 9a).
- A server-key find is unchanged: moderated when `ModerateOutput` is on
  (a check that cannot be made refuses it), unchecked when it is off.

Ruling 15 decides the allowlist's place: a player-key reply whose text
parses and cleans but fails `CheckPlayerKeyText` is NOT the player's key
failing. Their breaker hears nothing, and the find goes on to the server's
route (when that is closed, a generic trinket; slice C's corpus later).
So the check runs inside `name`, before the route is final
(`refusedByAllowlist`), and not in `generate`, which feeds every parse or
clean failure to the route's breaker. It applies to finder-only text too.

**Files:**
- Modify: `internal/apiframework/breaker.go` (new `RecordConsumer`), `internal/apiframework/apiframework_test.go`
- Modify: `modules/baubles/generate.go` (`generate`, `name`, `moderate`, new `moderationPossible`, new `refusedByAllowlist`)
- Modify: `modules/baubles/prompt.go` (`PromptVersion`, `systemPrompt`)
- Modify: `modules/baubles/baubles.go` (`info`)
- Modify: `modules/baubles/baubles_test.go`

- [ ] **Step 0: A check can feed one consumer's breaker alone**

Append to `internal/apiframework/apiframework_test.go`:

```go
// A check that belongs to one feature but is not a model call on the
// server's key (baubles' moderation of text a player's own key wrote)
// counts against that feature's own breaker alone, never the provider's
// the companion shares (owner ruling 2026-09-29).
func TestRecordConsumerFeedsOnlyItsOwnBreaker(t *testing.T) {
	k := NewBooksForTest()
	now := time.Unix(1000, 0)
	k.RecordConsumer(ConsumerBaubles, status(500), now)
	k.RecordConsumer(ConsumerBaubles, status(500), now)
	if !k.Blocked(ConsumerBaubles, now) {
		t.Fatal("two failures open baubles' own breaker (BreakerErrors 2)")
	}
	if k.BreakerOpen(now) || k.BreakerFailures() != 0 || k.Blocked(ConsumerCompanion, now) {
		t.Fatal("a provider-shaped failure (500) still never reaches the provider breaker or the companion")
	}
	k.ResetBreaker()
	k.RecordConsumer(ConsumerBaubles, status(500), now)
	k.RecordConsumer(ConsumerBaubles, nil, now)
	if k.ConsumerFailures(ConsumerBaubles) != 0 {
		t.Fatal("a success resets the run")
	}
}
```

Run: `go test ./internal/apiframework/ -run TestRecordConsumerFeedsOnlyItsOwnBreaker -v`
Expected: build failure `k.RecordConsumer undefined`.

In `internal/apiframework/breaker.go`, directly above `// Release hands a ticket back unjudged`, add:

```go
// RecordConsumer counts one outcome against consumer's OWN breaker alone,
// never the provider's: for a check that belongs to one feature but is not
// a model call on the server's key (baubles' moderation of text a
// player's own key wrote; owner ruling 2026-09-29). err nil is a success.
// There is no ticket: the check never asked Allow for leave, so it is
// never the half-open probe. Once an opened breaker's cooldown is up,
// Blocked lets every such check through again (no probe is out), a
// success resets the run without clearing the trip, and BreakerErrors
// failures in a row open it for another cooldown. That is the right
// shape for a free check that cannot overload anyone; a caller that needs
// one-probe semantics uses Allow and Record.
func RecordConsumer(consumer string, err error, now time.Time) {
	shared.RecordConsumer(consumer, err, now)
}

// RecordConsumer on these books.
func (k *Books) RecordConsumer(consumer string, err error, now time.Time) {
	limit, cooldown := breakerSettings()
	k.consumerBreaker(consumer).record(0, err != nil, now, limit, cooldown)
}

```

Run: `go test ./internal/apiframework/ -run TestRecordConsumerFeedsOnlyItsOwnBreaker -v`
Expected: PASS. Probe: temporarily make the method body `k.Record(consumer, Ticket{}, err, now)` and rerun. Expected: FAIL "a provider-shaped failure (500) still never reaches the provider breaker". Restore, rerun, PASS.

- [ ] **Step 1: Make the fake answer one result per input**

In `modules/baubles/baubles_test.go`, add two fields to `fakeOpenAI` after `modStatus`:

```go
	flagWord       string       // when set, any moderation input containing it is flagged
	lastModeration atomic.Value // the last moderation inputs, newline-joined
	moderations    int32        // moderation calls made, status answered or not
```

and, as the first line of the `/moderations` case, add `			atomic.AddInt32(&f.moderations, 1)`.

and replace the `/moderations` case body

```go
			if f.modStatus != 0 {
				w.WriteHeader(f.modStatus)
				return
			}
			fmt.Fprintf(w, `{"results":[{"flagged":%t},{"flagged":false}]}`, f.flagged)
```
with
```go
			if f.modStatus != 0 {
				w.WriteHeader(f.modStatus)
				return
			}
			// One result per input, as the real endpoint answers (and as
			// apiframework.Moderate requires).
			var in struct {
				Input []string `json:"input"`
			}
			_ = json.Unmarshal(body, &in)
			f.lastModeration.Store(strings.Join(in.Input, "\n"))
			results := make([]string, len(in.Input))
			for i, text := range in.Input {
				flag := (i == 0 && f.flagged) || (f.flagWord != `` && strings.Contains(text, f.flagWord))
				results[i] = fmt.Sprintf(`{"flagged":%t}`, flag)
			}
			fmt.Fprintf(w, `{"results":[%s]}`, strings.Join(results, `,`))
```

Run: `go test ./modules/baubles/`
Expected: `ok` (the fake still answers two inputs with two results).

- [ ] **Step 2: Rewrite the tests whose rule changes**

In `modules/baubles/baubles_test.go`:

(a) Replace the whole of `TestModerationOutageNeverSpoilsAPlayerKeyFind` (and its comment) with:

```go
// Moderation policy (spec S3; owner ruling 2026-09-29): a flag always keeps
// a find out, and a check that is made and fails keeps out a find on EITHER
// key. Every failed check is held against the moderation breaker alone,
// never the naming breaker and never the provider's the companion shares.
func TestModerationOutageRefusesAPlayerKeyFind(t *testing.T) {
	f := newFakeOpenAI(t)
	f.modStatus = 500
	m := testModule(t, f, func(c *Config) { c.ModerateOutput = true })
	if _, err := m.generate(context.Background(), request()); err == nil {
		t.Fatal("the server's key: no check, no name")
	}
	relay := &fakeRelay{allowed: map[int]bool{7: true}, model: `player-model`, provider: f}
	apiframework.SetRelay(relay)
	if _, err := m.generate(context.Background(), request()); err == nil || relay.sends != 1 {
		t.Fatalf("the finder's key named it, the check failed: refused (sends=%d)", relay.sends)
	}
	// Both failed checks, the server-key one and the player-key one, are on
	// the moderation breaker; the naming breaker and the provider's hold none.
	if n := apiframework.Shared().ConsumerFailures(moderationBreaker); n != 2 ||
		apiframework.Shared().ConsumerFailures(apiframework.ConsumerBaubles) != 0 || apiframework.BreakerFailures() != 0 {
		t.Fatalf("the moderation breaker holds both failed checks (%d), the naming breaker and the provider's none", n)
	}
	f.modStatus, f.flagged = 0, true
	if _, err := m.generate(context.Background(), request()); err == nil {
		t.Fatal("a flag keeps a player-key find out")
	}
	f.flagged = false
	if res, err := m.generate(context.Background(), request()); err != nil || !res.PlayerKey || !res.Moderated || res.FinderOnly {
		t.Fatalf("a clean check: named on the finder's key, moderated, everyone's: %+v %v", res, err)
	}
}
```

(b) In `TestNoKeyAtAll`, replace everything from the comment `// A finder's own key still names it;` to the end of the function with:

```go
	// With no server key nothing can moderate a finder's own key's text, so
	// it is named there and kept to its finder (owner ruling 2026-09-29):
	// FinderOnly, not Moderated, and no moderation call is made.
	relay := &fakeRelay{allowed: map[int]bool{7: true}, model: `player-model`, provider: f}
	apiframework.SetRelay(relay)
	res, err := m.generate(context.Background(), request())
	if err != nil || !res.PlayerKey || res.Moderated || !res.FinderOnly || relay.sends != 1 {
		t.Fatalf("finder's key, kept to the finder: %+v %v sends=%d", res, err, relay.sends)
	}
	if got := f.lastModeration.Load(); got != nil {
		t.Fatalf("nothing was sent to moderation: %v", got)
	}
}
```

(c) In `TestFindersOwnKeyNamesTheirFind`, `TestFindersKeyFailingFallsBackToTheServer`, `TestFindersUnusableReplyIsReportedForFindsOnly` and `TestPickpocketUsesTheThiefsKeyFirst`, change `m := testModule(t, serverSide, nil)` to:

```go
	m := testModule(t, serverSide, func(c *Config) { c.ModerateOutput = true })
```

and in `TestFindersOwnKeyNamesTheirFind` change `if !res.PlayerKey || res.Model != `player-model` ||` to `if !res.PlayerKey || !res.Moderated || res.Model != `player-model` ||`.

(d) In `TestFindersKeyOnlyWhenAllowed`, change `m := testModule(t, serverSide, nil)` to `m := testModule(t, serverSide, func(c *Config) { c.ModerateOutput = true })` and `m2 := testModule(t, serverSide, func(c *Config) { c.UsePlayerKeys = false })` to `m2 := testModule(t, serverSide, func(c *Config) { c.UsePlayerKeys, c.ModerateOutput = false, true })`.

- [ ] **Step 3: Write the new failing tests**

Append to `modules/baubles/baubles_test.go`:

```go
// Where the server cannot moderate (no server key, ModerateOutput off, the
// provider breaker or the moderation breaker open), a finder's own key still names
// the find, and its text is kept to that finder (owner ruling 2026-09-29):
// FinderOnly, not Moderated, and no moderation call is made.
func TestPlayerKeyTextThatCannotBeModeratedIsFinderOnly(t *testing.T) {
	cases := map[string]func(t *testing.T, f *fakeOpenAI) *BaublesModule{
		`no server key`: func(t *testing.T, f *fakeOpenAI) *BaublesModule {
			m := testModule(t, f, func(c *Config) { c.ModerateOutput = true })
			server(t, f.srv.URL, ``, 2000000, 3)
			return m
		},
		`moderation off`: func(t *testing.T, f *fakeOpenAI) *BaublesModule {
			return testModule(t, f, nil)
		},
		`provider breaker open`: func(t *testing.T, f *fakeOpenAI) *BaublesModule {
			m := testModule(t, f, func(c *Config) { c.ModerateOutput = true })
			apiframework.SetBreakerForTest(5, time.Now().Add(time.Minute))
			return m
		},
		`moderation breaker open`: func(t *testing.T, f *fakeOpenAI) *BaublesModule {
			m := testModule(t, f, func(c *Config) { c.ModerateOutput = true })
			apiframework.Shared().SetConsumerBreakerForTest(moderationBreaker, 0, time.Now().Add(time.Minute))
			return m
		},
	}
	for name, build := range cases {
		f := newFakeOpenAI(t)
		m := build(t, f)
		relay := &fakeRelay{allowed: map[int]bool{7: true}, model: `player-model`, provider: newFakeOpenAI(t)}
		apiframework.SetRelay(relay)
		res, err := m.generate(context.Background(), request())
		if err != nil || !res.PlayerKey || res.Moderated || !res.FinderOnly || relay.sends != 1 {
			t.Errorf("%v: named on the finder's key, kept to the finder: %+v %v sends=%d", name, res, err, relay.sends)
		}
		if got := f.lastModeration.Load(); got != nil {
			t.Errorf("%v: no moderation call is made: %v", name, got)
		}
	}
}

// The moderation breaker is its own (review finding d): a server-key find
// whose naming succeeds does not reset a run of failed checks, so checks
// failing on both routes open it, and then a finder's own key's text is
// kept to the finder with no check tried, while the naming breaker, which
// saw only successes, stays closed. Sequence: server find, player find,
// server find, each with the check failing (BreakerErrors 3).
func TestFailedChecksOpenTheModerationBreakerAlone(t *testing.T) {
	f := newFakeOpenAI(t)
	f.modStatus = 500
	m := testModule(t, f, func(c *Config) { c.ModerateOutput = true })
	relay := &fakeRelay{allowed: map[int]bool{}, model: `player-model`, provider: newFakeOpenAI(t)}
	apiframework.SetRelay(relay)

	for i, playerKey := range []bool{false, true, false} {
		relay.allowed[7] = playerKey
		if _, err := m.generate(context.Background(), request()); err == nil {
			t.Fatalf("find %d: the check failed, so it is refused", i)
		}
	}
	if !apiframework.Blocked(moderationBreaker, time.Now()) {
		t.Fatal("three failed checks in a row, whatever the route, open the moderation breaker")
	}
	if apiframework.Blocked(apiframework.ConsumerBaubles, time.Now()) || apiframework.BreakerOpen(time.Now()) {
		t.Fatal("every naming call succeeded: the naming and provider breakers stay closed")
	}

	f.modStatus = 0
	checks := atomic.LoadInt32(&f.moderations)
	relay.allowed[7] = true
	res, err := m.generate(context.Background(), request())
	if err != nil || !res.FinderOnly || res.Moderated || atomic.LoadInt32(&f.moderations) != checks {
		t.Fatalf("with the breaker open, the finder's text is kept to them, unchecked: %+v %v", res, err)
	}
	relay.allowed[7] = false
	if _, err := m.generate(context.Background(), request()); !errors.Is(err, errBreakerOpen) {
		t.Fatalf("and a server-key find is refused, unchecked: %v", err)
	}
}

// Player-key text outside the allowlist (ruling 15) is not the finder's key
// failing: their finds breaker hears nothing, and the find goes on to the
// server's key, which names it.
func TestPlayerKeyTextOutsideTheAllowlistFallsBackToTheServer(t *testing.T) {
	serverSide := newFakeOpenAI(t)
	m := testModule(t, serverSide, func(c *Config) { c.ModerateOutput = true })
	odd := strings.Replace(goodContent, `Painted Wooden Horse`, `Painted Wooden H\u00f6rse`, 1)
	relay := &fakeRelay{allowed: map[int]bool{7: true}, model: `player-model`, reply: chatBody(odd, 240), provider: newFakeOpenAI(t)}
	apiframework.SetRelay(relay)
	res, err := m.generate(context.Background(), request())
	if err != nil || res.PlayerKey || res.Reply.Name != `Painted Wooden Horse` || relay.sends != 1 || atomic.LoadInt32(&serverSide.chats) != 1 {
		t.Fatalf("refused on the finder's key, named on the server's: %+v %v sends=%d server=%d", res, err, relay.sends, serverSide.chats)
	}
	if len(relay.results) != 0 {
		t.Fatalf("an allowlist refusal is not the finder's key failing: %v", relay.results)
	}
}

// Curly quotes and dashes from a finder's own key are folded to ASCII
// before the allowlist looks (ruling 15), so the find keeps its route.
func TestPlayerKeyTypographyIsFoldedNotRefused(t *testing.T) {
	serverSide := newFakeOpenAI(t)
	m := testModule(t, serverSide, func(c *Config) { c.ModerateOutput = true })
	curly := strings.Replace(goodContent, `A child's toy horse, its red`, `A child\u2019s toy horse \u2014 its red`, 1)
	relay := &fakeRelay{allowed: map[int]bool{7: true}, model: `player-model`, reply: chatBody(curly, 240), provider: newFakeOpenAI(t)}
	apiframework.SetRelay(relay)
	res, err := m.generate(context.Background(), request())
	if err != nil || !res.PlayerKey || atomic.LoadInt32(&serverSide.chats) != 0 {
		t.Fatalf("named on the finder's key: %+v %v server=%d", res, err, serverSide.chats)
	}
	if want := `A child's toy horse - its red paint flaking from the mane.`; res.Reply.Description != want {
		t.Fatalf("the cleaned, folded text is what is kept:\n got %v\nwant %v", res.Reply.Description, want)
	}
}

// Every text field is moderated, on every route (spec S3, ruling 15): the
// name, the keyword players type, the description, and the material that
// appraise shows.
func TestEveryTextFieldIsModerated(t *testing.T) {
	f := newFakeOpenAI(t)
	f.flagWord = `pine`
	m := testModule(t, f, func(c *Config) { c.ModerateOutput = true })
	if _, err := m.generate(context.Background(), request()); err == nil {
		t.Fatal("a flagged material refuses the find")
	}
	got, _ := f.lastModeration.Load().(string)
	want := "Painted Wooden Horse\nhorse\nA child's toy horse, its red paint flaking from the mane.\npine"
	if got != want {
		t.Fatalf("name, keyword, description and material, in that order:\n got %v\nwant %v", got, want)
	}
}

// The system prompt states the characters the player-key allowlist accepts
// (ruling 15), so a model on either key is asked for text that passes.
func TestSystemPromptStatesTheAllowedCharacters(t *testing.T) {
	if PromptVersion < 5 {
		t.Fatalf("the prompt changed: PromptVersion must be at least 5, is %d", PromptVersion)
	}
	for _, want := range []string{`plain ASCII`, `' " - , . ! ?`, `no colons`} {
		if !strings.Contains(systemPrompt, want) {
			t.Errorf("the system prompt does not say %v", want)
		}
	}
}

// chatBody is a provider's chat completions answer with this content.
func chatBody(content string, tokens int) string {
	b, _ := json.Marshal(map[string]any{
		`choices`: []any{map[string]any{`finish_reason`: `stop`, `message`: map[string]any{`content`: content}}},
		`usage`:   map[string]any{`total_tokens`: tokens},
	})
	return string(b)
}
```

`goodContent` is `{"name":"Painted Wooden Horse","name_simple":"horse","description":"A child's toy horse, its red paint flaking from the mane.","material":"pine",...}` (`baubles_test.go:38`); `\u00f6` inside a Go raw string stays the six characters `\u00f6`, which JSON decodes to `ö`. The other escapes in the new tests' raw strings work the same way. The no-server-key case is covered by `TestNoKeyAtAll` above. Every new message that mentions a key prints its values with `%v` or `%d`, never `%s` or `%q`: `TestNoTestPrintsAKey` scans this package, and "keyword" contains "key" too.

- [ ] **Step 4: Run to verify the new and rewritten tests fail**

Run: `go test ./modules/baubles/ -v 2>&1 | grep -E "^(=== RUN|--- FAIL|--- PASS|FAIL|ok)" | grep -E "FAIL|ok"`
Expected: build failure `undefined: moderationBreaker`. To see the assertions fail too, add Step 5's `const moderationBreaker` line alone and rerun: FAIL for `TestModerationOutageRefusesAPlayerKeyFind` (the finder's key is accepted unmoderated today, and nothing feeds a moderation breaker), `TestFailedChecksOpenTheModerationBreakerAlone` (never opens), `TestNoKeyAtAll` and `TestPlayerKeyTextThatCannotBeModeratedIsFinderOnly` (named, but never `FinderOnly`), `TestPlayerKeyTextOutsideTheAllowlistFallsBackToTheServer` (the accented name is used on the finder's key today), `TestPlayerKeyTypographyIsFoldedNotRefused` (today `generate` keeps the RAW reply, so the description still carries the curly apostrophe and the em dash), `TestEveryTextFieldIsModerated` (two inputs sent, not four) and `TestSystemPromptStatesTheAllowedCharacters`.

- [ ] **Step 5: The moderation test and the allowlist fallback**

In `modules/baubles/generate.go`, add, directly above `name`:

```go
// moderationBreaker is the moderation check's OWN consumer breaker, apart
// from ConsumerBaubles (review finding d): a naming call's success on the
// server's key reports to ConsumerBaubles before moderation runs
// (generate), and a breaker shared by both would have every good naming
// call reset the run of failed checks, so it could never open; and failed
// naming calls would count as moderation failures. It is fed only by
// moderate, on both routes (apiframework.RecordConsumer), and read by
// moderationPossible and moderate.
const moderationBreaker = `baubles-moderation`

// moderationPossible reports whether the server can moderate a reply now:
// ModerateOutput on, a server key, and neither the provider breaker nor
// the moderation breaker open (apiframework.Blocked reads both). s and now
// are read once by the caller and passed in. Player-key text named while
// it is not is kept to its finder (FinderOnly; owner ruling 2026-09-29),
// never shown to anyone else unmoderated.
func moderationPossible(cfg Config, s apiframework.ServerSettings, now time.Time) bool {
	return cfg.ModerateOutput && s.HasKey() && !apiframework.Blocked(moderationBreaker, now)
}
```

And directly below it:

```go
// refusedByAllowlist reports whether content, from a finder's own key, is
// a usable answer (it parses and passes CleanReply) whose cleaned text
// baubles.CheckPlayerKeyText refuses (ruling 15). That is not the key
// failing: the caller tells its breaker nothing and goes on to the server's
// route. An answer that does not parse or clean is not this; it goes on to
// generate, which reports it to the player's breaker as before.
func refusedByAllowlist(content string) bool {
	reply, err := baubles.ParseReply(content)
	if err != nil {
		return false
	}
	if reply, err = baubles.CleanReply(reply); err != nil {
		return false
	}
	return baubles.CheckPlayerKeyText(reply) != nil
}
```

In `name`, replace the whole player branch

```go
	if cfg.UsePlayerKeys && req.FinderUserId > 0 {
		if r := apiframework.PlayerRelay(); r != nil {
			if relayModel, ok := r.Model(req.FinderUserId, apiframework.PurposeFinds); ok {
				content, tokens, report, err = viaPlayer(ctx, r, req.FinderUserId, relayModel, chat)
				if err == nil {
					return content, tokens, relayModel, true, report, nil
				}
				report(err)
				if ctx.Err() != nil {
					return ``, 0, relayModel, true, func(error) {}, err
				}
			}
		}
	}
```
with
```go
	if cfg.UsePlayerKeys && req.FinderUserId > 0 {
		if r := apiframework.PlayerRelay(); r != nil {
			if relayModel, ok := r.Model(req.FinderUserId, apiframework.PurposeFinds); ok {
				content, tokens, report, err = viaPlayer(ctx, r, req.FinderUserId, relayModel, chat)
				switch {
				case err == nil && refusedByAllowlist(content):
					// Plain enough for the model, not for other players
					// (ruling 15): not the key's failure, so its breaker
					// hears nothing, and the server's key names the find.
				case err == nil:
					return content, tokens, relayModel, true, report, nil
				default:
					report(err)
					if ctx.Err() != nil {
						return ``, 0, relayModel, true, func(error) {}, err
					}
				}
			}
		}
	}
```

- [ ] **Step 6: Moderate the cleaned text, every field included**

In `generate`, replace

```go
	reply, err := baubles.ParseReply(content)
	if err == nil {
		_, err = baubles.CleanReply(reply)
	}
	report(err)
```
with
```go
	reply, err := baubles.ParseReply(content)
	if err == nil {
		// Keep the cleaned text: it is what is moderated and what the
		// world shows. A player-key reply already passed the allowlist in
		// name (refusedByAllowlist); it is not checked again here, where a
		// refusal would feed the player's breaker. baubles.Generate holds
		// every generator to it all the same.
		reply, err = baubles.CleanReply(reply)
	}
	report(err)
```

and replace

```go
	moderated, err := m.moderate(cfg, reply, playerKey)
	if err != nil {
		return baubles.GenResult{}, err
	}
```
with
```go
	moderated, finderOnly, err := m.moderate(cfg, reply, playerKey)
	if err != nil {
		return baubles.GenResult{}, err
	}
```
and, in the returned `baubles.GenResult`, after `PlayerKey:     playerKey,` add `FinderOnly:    finderOnly,`.

Replace the whole of `moderate` and its doc comment with:

```go
// moderate checks the name, keyword (NameSimple), description and material
// through the server's key (a player's key page reaches no moderation
// endpoint). The policy, decided and pinned by test (spec S3, ruling 15,
// owner ruling 2026-09-29):
//
//   - A flag always keeps the text out of the world: a generic trinket.
//   - Server-key text: checked when ModerateOutput is on, and kept out when
//     the check cannot be made or fails; not checked when it is off.
//   - Player-key text: checked whenever the server can
//     (moderationPossible), and then a failed check keeps it out too.
//     When the server cannot check it, no call is made and it is kept to
//     its finder (finderOnly: everyone else reads the generic trinket).
//   - Every check made, on either route, is recorded on the moderation
//     breaker alone (apiframework.RecordConsumer), never the provider's
//     and never the naming breaker. A flag is the check working.
//
// The check is free and is not a model call, so it reserves nothing. The
// server settings and the clock are read once here and passed on.
func (m *BaublesModule) moderate(cfg Config, reply baubles.Reply, playerKey bool) (moderated bool, finderOnly bool, err error) {
	now := time.Now()
	s := apiframework.Server()
	if playerKey && !moderationPossible(cfg, s, now) {
		return false, true, nil
	}
	if !cfg.ModerateOutput {
		return false, false, nil
	}
	if !s.HasKey() {
		return false, false, errNoRoute
	}
	if apiframework.Blocked(moderationBreaker, now) {
		return false, false, errBreakerOpen
	}
	// Every field a player reads or types. CleanReply always leaves a
	// keyword (the model's, a word of the name, or "trinket"); a material
	// it found too long is empty and not sent.
	texts := []string{reply.Name, reply.NameSimple, reply.Description}
	if reply.Material != `` {
		texts = append(texts, reply.Material)
	}
	flags, err := apiframework.Moderate(s.Endpoint, cfg.ModerationModel, time.Duration(cfg.TimeoutSeconds)*time.Second,
		texts, apiframework.CarriesNoPlayerData, nil)
	// Enough failed checks in a row, on either route, open the moderation
	// breaker: later player-key finds are then kept to their finders, and
	// server-key finds refused, until it closes. A flag is the check working.
	apiframework.RecordConsumer(moderationBreaker, err, now)
	if err != nil {
		return false, false, fmt.Errorf(`moderation: %w`, err)
	}
	for _, f := range flags {
		if f {
			return false, false, errors.New(`moderation flagged the reply`)
		}
	}
	return true, false, nil
}
```

Update `generate`'s doc comment route paragraph to: `The route: the finder's own key first, when they allowed it on the key page (apiframework.PurposeFinds); then the server's key, reserved against the one daily budget every feature shares; else no name. Player-key text the server cannot moderate is kept to its finder (moderate).`

- [ ] **Step 7: Correct the status text**

In `modules/baubles/baubles.go` `info`, replace

```go
	if !s.HasKey() {
		detail += ` No server key: only finders who allowed their own key get named finds.`
	}
```
with
```go
	if !s.HasKey() {
		detail += ` No server key: finds named on a finder's own key are shown to that finder alone (nothing can moderate them); every other find is a generic trinket.`
	} else if cfg.UsePlayerKeys && !cfg.ModerateOutput {
		detail += ` ModerateOutput is off: finds named on finders' own keys are shown to those finders alone.`
	}
	if now := time.Now(); cfg.ModerateOutput && apiframework.Blocked(moderationBreaker, now) && !apiframework.BreakerOpen(now) {
		detail += ` The moderation breaker is open (the moderation endpoint keeps failing): server-key finds are generic and finders' own keys' finds are shown to those finders alone until ` +
			apiframework.BreakerUntil(moderationBreaker).Format(`15:04:05`) + `.`
	}
```

The existing naming-breaker line (`baubles.go:141`, "Baubles' own breaker is open (the provider is fine; check Modules.baubles.Model)") now describes the NAMING breaker only; reword it to ` Baubles' naming breaker is open (the provider is fine; check Modules.baubles.Model)`.

- [ ] **Step 8: The system prompt states the allowed characters (PromptVersion 5)**

In `modules/baubles/prompt.go`, replace

```go
// Version 4: a pickpocketed find is lifted from a person (taken_from, the
// NPC's authored name) and is pocket-sized (size_rule).
const PromptVersion = 4
```
with
```go
// Version 4: a pickpocketed find is lifted from a person (taken_from, the
// NPC's authored name) and is pocket-sized (size_rule).
// Version 5: every text field is asked for in plain ASCII letters and
// ' " - , . ! ? with a space after each sentence, the characters the
// player-key allowlist accepts (baubles.CheckPlayerKeyText, ruling 15).
const PromptVersion = 5
```

and in `systemPrompt`, replace

```go
	`Do not repeat or closely copy any name in avoid_names.`,
}, "\n\n")
```
with
```go
	`Do not repeat or closely copy any name in avoid_names.`,
	`Characters: write name, name_simple, description and material in plain ASCII only, using the letters A to Z and a to z, spaces, and the marks ' " - , . ! ? and nothing else: no digits, no accented or non-Latin letters, no colons, semicolons, slashes or other symbols, and no curly quotes or long dashes (use ' " and - instead). Put a space after every sentence, never a full stop joined straight to the next word.`,
}, "\n\n")
```

No test pins `PromptVersion`'s value: `baubles_test.go:160` compares a result with the constant, and the literals in `pickpocket_test.go`, `search_bauble_test.go`, `admin_test.go` and `generate_test.go` sit in fake `GenResult`s. `TestSystemPromptStatesTheAllowedCharacters` (Step 3) pins the new line.

- [ ] **Step 9: Run to verify every module test passes**

Run: `go test ./modules/baubles/ -v 2>&1 | grep -E "^(--- FAIL|FAIL|ok)"`
Expected: only `ok`.

- [ ] **Step 10: Probe the policy and the allowlist fallback**

Temporarily change `moderationPossible`'s body to `return cfg.ModerateOutput && s.HasKey()` (dropping the breaker check), run `go test ./modules/baubles/ -run TestPlayerKeyTextThatCannotBeModeratedIsFinderOnly -v`. Expected: FAIL naming `provider breaker open` and `moderation breaker open` (both refused by `errBreakerOpen` instead of kept to the finder). Restore, rerun, PASS.

Temporarily delete the `apiframework.RecordConsumer(moderationBreaker, err, now)` line from `moderate`, run `go test ./modules/baubles/ -run "TestModerationOutageRefusesAPlayerKeyFind|TestFailedChecksOpenTheModerationBreakerAlone" -v`. Expected: FAIL "the moderation breaker holds both failed checks (0)" and "three failed checks in a row, whatever the route, open the moderation breaker". Then, instead, change it to `apiframework.RecordConsumer(apiframework.ConsumerBaubles, err, now)` (the shared breaker the review found) and rerun the second test: expected FAIL, since each good naming call resets the run. Restore, rerun, PASS.

Temporarily change `refusedByAllowlist`'s last line to `return false`, run `go test ./modules/baubles/ -run TestPlayerKeyTextOutsideTheAllowlistFallsBackToTheServer -v`. Expected: FAIL "refused on the finder's key, named on the server's" (the accented name is kept on the player's key; `baubles.Generate` would refuse it later, but this test calls the module directly). Restore, rerun, PASS.

- [ ] **Step 11: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/apiframework/breaker.go internal/apiframework/apiframework_test.go modules/baubles/generate.go modules/baubles/prompt.go modules/baubles/baubles.go modules/baubles/baubles_test.go && git commit -F - <<'EOF'
fix(baubles): player-key text is moderated, or kept to its finder

Where the server can moderate, a finder's own key's text is checked like
the server's: a flag or a failed check refuses it. Every check made, on
either route, feeds the moderation check's own breaker
(apiframework.RecordConsumer), never the naming breaker or the
provider's. Where it cannot (no server key, ModerateOutput off, a breaker
open), the find is FinderOnly. Text outside the plain-text allowlist falls back to
the server's key without touching the player's breaker. The keyword and
material are moderated with the name and description on every route, and
the system prompt (version 5) states the allowed characters.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 11: S3, relay tokens are clamped and relay calls take a per-finder slot

**Files:**
- Modify: `modules/baubles/generate.go` (`generate`, `name`, `viaPlayer`)
- Modify: `modules/baubles/baubles.go` (`BaublesModule`, new `takeFinderSlot`, `takeServerSlot`)
- Modify: `modules/baubles/baubles_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `modules/baubles/baubles_test.go`:

```go
// A token count relayed through a player's browser is theirs to write: it
// is held to what one request could cost before it reaches the record or
// the statistics (spec S3).
func TestARelayedTokenCountIsClamped(t *testing.T) {
	serverSide := newFakeOpenAI(t)
	m := testModule(t, serverSide, func(c *Config) { c.ModerateOutput = true })
	relay := &fakeRelay{allowed: map[int]bool{7: true}, model: `player-model`, reply: chatBody(goodContent, 999999), provider: newFakeOpenAI(t)}
	apiframework.SetRelay(relay)
	res, err := m.generate(context.Background(), request())
	if err != nil || !res.PlayerKey {
		t.Fatalf("named on the finder's key: %+v %v", res, err)
	}
	most := apiframework.EstimateTokens(buildMessages(request())) + schemaOverhead + m.snapshot().MaxCompletionTokens
	if res.Tokens != most {
		t.Fatalf("clamped to the most one request costs (%d), got %d", most, res.Tokens)
	}
}

// A relay call takes the finder's own slot (one in flight per finder), not
// one of the server's shared slots (spec S3).
func TestAFindersOwnKeyTakesTheirOwnSlot(t *testing.T) {
	serverSide := newFakeOpenAI(t)
	m := testModule(t, serverSide, func(c *Config) { c.MaxConcurrent, c.ModerateOutput = 1, true })
	relay := &fakeRelay{allowed: map[int]bool{7: true}, model: `player-model`, provider: newFakeOpenAI(t)}
	apiframework.SetRelay(relay)

	m.slots <- struct{}{} // every server slot is busy
	res, err := m.generate(context.Background(), request())
	<-m.slots
	if err != nil || !res.PlayerKey || relay.sends != 1 {
		t.Fatalf("the finder's key names it with the server's slots full: %+v %v sends=%d", res, err, relay.sends)
	}

	release, ok := m.takeFinderSlot(7)
	if !ok {
		t.Fatal("the finder's slot is free again")
	}
	res, err = m.generate(context.Background(), request())
	release()
	if err != nil || res.PlayerKey || relay.sends != 1 || atomic.LoadInt32(&serverSide.chats) != 1 {
		t.Fatalf("with their own call in flight, the server's key names it: %+v %v sends=%d", res, err, relay.sends)
	}
	if _, ok := m.takeFinderSlot(7); !ok {
		t.Fatal("released")
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./modules/baubles/ -run "TestARelayedTokenCountIsClamped|TestAFindersOwnKeyTakesTheirOwnSlot" -v`
Expected: build failure `m.takeFinderSlot undefined`. (After Step 3 adds the methods but before Step 4, the first test fails with the raw 999999 and the second with "names it with the server's slots full".)

- [ ] **Step 3: Add the slots**

In `modules/baubles/baubles.go`, add a field to `BaublesModule` after `slots chan struct{}`:

```go
	// finders holds the users with a find in flight on their own key: one
	// at a time each, and never one of the server's slots (spec S3).
	finders map[int]bool
```

and add, after `snapshot`:

```go
// takeServerSlot takes one of the server key's MaxConcurrent slots, or
// reports none free. A find beyond them is not queued: it is a generic
// trinket.
func (m *BaublesModule) takeServerSlot() (release func(), ok bool) {
	m.mu.Lock()
	slots := m.slots
	m.mu.Unlock()
	select {
	case slots <- struct{}{}:
		return func() { <-slots }, true
	default:
		return nil, false
	}
}

// takeFinderSlot takes the finder's own slot: one call on their key at a
// time.
func (m *BaublesModule) takeFinderSlot(userId int) (release func(), ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.finders[userId] {
		return nil, false
	}
	if m.finders == nil {
		m.finders = map[int]bool{}
	}
	m.finders[userId] = true
	return func() {
		m.mu.Lock()
		delete(m.finders, userId)
		m.mu.Unlock()
	}, true
}
```

- [ ] **Step 4: Route through the slots and clamp the relay count**

In `modules/baubles/generate.go`:

(a) Add `errSlotsBusy = errors.New(`all generation slots busy`)` to the error `var` block.

(b) In `generate`, delete the whole block from `// A fixed number of calls at once.` through the closing `}` of the `select` (the `m.mu.Lock()`, `slots := m.slots`, `m.mu.Unlock()` and the `select`). Then, directly after `	content, tokens, model, playerKey, report, err := m.name(ctx, cfg, req, chat)`, add:

```go
	if errors.Is(err, errSlotsBusy) {
		// Refused at the door, not a call that failed: it counts in no
		// statistic (m.count) and feeds no breaker, as before this slice.
		return baubles.GenResult{}, err
	}
```
(`m.count` at `baubles.go:101` would otherwise count a busy refusal as a failed call, which the old slot check, returning before `m.count`, never did.)

(c) In `name`, replace Task 10's relay call and its `switch`

```go
				content, tokens, report, err = viaPlayer(ctx, r, req.FinderUserId, relayModel, chat)
				switch {
				case err == nil && refusedByAllowlist(content):
					// Plain enough for the model, not for other players
					// (ruling 15): not the key's failure, so its breaker
					// hears nothing, and the server's key names the find.
				case err == nil:
					return content, tokens, relayModel, true, report, nil
				default:
					report(err)
					if ctx.Err() != nil {
						return ``, 0, relayModel, true, func(error) {}, err
					}
				}
```
with
```go
				// The finder's own slot, never one of the server's: a busy
				// one (their last find still naming) goes to the server.
				if release, free := m.takeFinderSlot(req.FinderUserId); free {
					content, tokens, report, err = viaPlayer(ctx, r, req.FinderUserId, relayModel, chat)
					release()
					switch {
					case err == nil && refusedByAllowlist(content):
						// Plain enough for the model, not for other players
						// (ruling 15): not the key's failure, so its breaker
						// hears nothing, and the server's key names the find.
					case err == nil:
						return content, tokens, relayModel, true, report, nil
					default:
						report(err)
						if ctx.Err() != nil {
							return ``, 0, relayModel, true, func(error) {}, err
						}
					}
				}
```

and replace

```go
	content, tokens, report, err = viaServer(ctx, cfg, chat)
	return content, tokens, cfg.Model, false, report, err
```
with
```go
	// The server key's model calls, MaxConcurrent at once. A slot covers
	// the call only; moderation afterwards is free and not a model call.
	release, free := m.takeServerSlot()
	if !free {
		return ``, 0, cfg.Model, false, func(error) {}, errSlotsBusy
	}
	defer release()
	content, tokens, report, err = viaServer(ctx, cfg, chat)
	return content, tokens, cfg.Model, false, report, err
```

(d) In `viaPlayer`, replace

```go
	reply := apiframework.DecodeChat(status, raw)
	return reply.Content, reply.Tokens, report, reply.Err
```
with
```go
	reply := apiframework.DecodeChat(status, raw)
	// The count came through the player's browser, which they can write:
	// held to what one request could cost before it is recorded (spec S3).
	prompt := apiframework.EstimateTokens(chat.Messages) + schemaOverhead
	tokens, _ := apiframework.Charged(reply.Tokens, true, status, prompt, chat.MaxTokens, true)
	return reply.Content, tokens, report, reply.Err
```

- [ ] **Step 5: Run every module test**

Run: `go test ./modules/baubles/ -v 2>&1 | grep -E "^(--- FAIL|FAIL|ok)"`
Expected: only `ok`. `TestBusySlotsAreRefusedNotQueued` still passes (no relay: the server route finds its one slot taken).

- [ ] **Step 6: Race check**

Run (Bash):
```bash
cd /c/tmp/dogmud-baubles-h && docker compose -f compose.test.yml run --build --rm test go test -race ./modules/baubles/ 2>&1 | tail -5
```
Expected: `ok`, no `WARNING: DATA RACE`.

- [ ] **Step 7: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add modules/baubles/generate.go modules/baubles/baubles.go modules/baubles/baubles_test.go && git commit -F - <<'EOF'
fix(baubles): clamp relayed token counts, one relay call per finder

viaPlayer passes the relay's count through Charged(relayed=true). Relay
calls take a per-finder slot instead of one of the server's
MaxConcurrent slots, which now cover the server's model call only.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 11a: PR H2 docs

(Text quoted from a `context.md` here is shown unwrapped; match it by its words and rewrap the replacement to the file's width.)

**Files:**
- Modify: `internal/baubles/context.md`, `modules/baubles/context.md`, `internal/items/context.md`, `internal/apiframework/context.md`, `internal/actions/context.md`, `internal/usercommands/context.md`, `modules/gmcp/context.md`, `modules/aicompanion/context.md`, `internal/enchantments/context.md`
- Modify: `_datafiles/config.yaml` (comments only), `docs/aicompanion/settings.md`, `docs/baubles/implementation-plan.md`

- [ ] **Step 0: Verify every symbol these docs name**

```bash
cd /c/tmp/dogmud-baubles-h && grep -rn "func CheckPlayerKeyText\|func AuthoredName\|func moderationPossible\|func refusedByAllowlist\|func (m \*BaublesModule) takeFinderSlot\|func (m \*BaublesModule) takeServerSlot\|const PromptVersion = 5\|const moderationBreaker\|func (r Record) KeptToFinder\|func (r Record) MaterialFor\|func genericDescriptionFor\|func (i \*Item) GetSpecFor\|func (i \*Item) DisplayNameFor\|func (i \*Item) NameFor\|func (i \*Item) LongDescriptionFor\|func (i \*Item) ModelName\|func (i \*Item) ModelDescription\|func RecordConsumer\|PlayerText bool\|FinderOnly bool" internal modules
```
Expected: at least one hit for each pattern (`FinderOnly bool` is `GenResult`'s; there is no stored `Record.FinderOnly`). Fix the doc text, never the grep.

- [ ] **Step 1: `internal/baubles/context.md`**

- In the **validate.go** bullet add: `` An authored item's whole name is refused (`items.AuthoredName`). ``
- Add a file bullet: `` - **playerkey.go**: `CheckPlayerKeyText`, the plain-text allowlist for text a player's own key wrote, on the CLEANED name, keyword, description and material: ASCII letters, space, `' " - , . ! ?`, and every run of periods followed by a space, a `"` or the end. ``
- In the **generate.go** bullet add: `` `Generate` refuses a `PlayerKey` result that fails `CheckPlayerKeyText`, or is neither `Moderated` nor `FinderOnly` with a finder, and any `FinderOnly` result that is not `PlayerKey`; `RecentNames` skips `PlayerKey` records. ``
- Change the `GenResult` signature line to `/* Reply, Generator, Model, PromptVersion, Tokens, Moderated, PlayerKey, FinderOnly */`.
- In the mint section add: `` `Mint` rolls a `PlayerKey` find's value with `tier.RollValue`, keeping the key's proposal in `ValueProposed`. `ApplyRegenerated` takes the new result's `PlayerKey` (a regen is always server-key). ``
- In the **record.go** bullet add: `` `KeptToFinder()` is `PlayerKey && !Moderated`: DERIVED, never stored, so records named before slice H are covered. Such a record's `View` is the generic trinket (`genericName`, `genericNameSimple`, `genericDescriptionFor(id)`, stable per id) with its own text in `BaubleView.Finder` for `FoundByUserId` (none when that is 0); retired text wins over both. `View` sets `BaubleView.PlayerText` for any `PlayerKey` record, so no model prompt carries it (`items.Item.ModelName`). `MaterialFor(viewerUserId)` is the material for the finder alone. Never promotable (slice C takes only server-key moderated text). ``
- Add a gotcha: `` - **Finder-only text is fail-safe, not routed.** The catalog's viewer-agnostic view of a finder-only record IS the generic trinket, so every render path shows "Trinket" unless it asks the item layer for one viewer's view (`items.Item.GetSpecFor` and kin), which only the single-reader functions listed in the repo-root `bauble_finder_view_guard_test.go` do, each with a pinned call count. Never read a record's `Name`, `Description` or `Material` for display outside the admin command; use the item accessors or `MaterialFor`. ``

- [ ] **Step 2: `modules/baubles/context.md`**

- Replace "How a call goes" step 2 with: `` 2. `name` picks the route; a slot is taken there (see 3 and 4). A busy server slot returns `errSlotsBusy` before `generate` counts anything. ``
- In step 3, after "on their key." insert `` It takes the finder's own slot (`takeFinderSlot`, one in flight per finder), never a server slot; a busy one goes to the server. The relay's token count passes through `Charged(..., relayed=true)`. An answer that parses and cleans but fails `baubles.CheckPlayerKeyText` (`refusedByAllowlist`) is not the key's failure: its breaker hears nothing and the find goes on to the server's key (ruling 15). ``
- In step 4, after "`viaServer`:" insert `` one of the `MaxConcurrent` server slots (`takeServerSlot`; none free is `errSlotsBusy` at once, not a queue; it covers the model call only); ``
- Replace step 6 with: `` 6. `baubles.ParseReply` and `CleanReply` (the cleaned text is kept). Then `moderate`, on the name, keyword, description and material, through the server key, reading the server settings and the clock once. Server-key text: checked when `ModerateOutput` is on (a flag, or a check that cannot be made or fails, refuses), unchecked when off. Player-key text: checked whenever `moderationPossible` (`ModerateOutput` on, a server key, neither the provider breaker nor the moderation breaker open); a flag or a failed check refuses. Every check made, on either route, is recorded on the moderation check's OWN breaker, `moderationBreaker` (`apiframework.RecordConsumer`), never on the naming breaker `ConsumerBaubles` (whose run a good naming call resets) and never on the provider's. When moderation is not possible no call is made and the find is `FinderOnly`: its finder reads it, everyone else the generic trinket (owner ruling 2026-09-29). ``
- Replace the "Moderation on a player's key" gotcha with: `` - **Moderation on a player's key.** Moderated when the server can, and then refused on a flag or a failed check; kept to its finder when it cannot, never shown to anyone else unmoderated. Also held to plain ASCII after folding (`CheckPlayerKeyText`; text outside it falls back to the server's key without touching the player's breaker), its value re-rolled by the server (`Mint`), its name never in another prompt (`RecentNames`), and never in the AI companion's prompts either (`items.Item.ModelName`). Pinned by `TestModerationOutageRefusesAPlayerKeyFind`, `TestFailedChecksOpenTheModerationBreakerAlone`, `TestPlayerKeyTextThatCannotBeModeratedIsFinderOnly`, `TestNoKeyAtAll`, `TestPlayerKeyTextOutsideTheAllowlistFallsBackToTheServer`, `TestPlayerKeyTypographyIsFoldedNotRefused`. ``
- Add a gotcha: `` - **The moderation breaker has no probe.** `RecordConsumer` carries no ticket, so once its cooldown is up every check is let through again, and `BreakerErrors` failures in a row reopen it. Fine for a free check; do not copy it for a model call. ``
- In the Config paragraph change `` `MaxConcurrent` (4) `` to `` `MaxConcurrent` (4, server-key calls only) ``.
- In the **prompt.go** bullet (the `PromptVersion` history) add: `` 5: the system prompt states the characters the player-key allowlist accepts. ``
- In the **generate.go** bullet list `moderationPossible`, `moderationBreaker` and `refusedByAllowlist` beside `name`; add `takeServerSlot` and `takeFinderSlot` to the **baubles.go** bullet, and note that `info` reports the naming breaker and the moderation breaker separately.

- [ ] **Step 3: `internal/items/context.md`**

- After the `AuthoredKeyword(word)` sentence add: `` `AuthoredName(name)` is whether a loaded item's whole name matches after NFKC, lower case and collapsed spaces (`normalizeItemName`); `baubles.CleanReply` refuses such a name. Both read one snapshot (`authored`, an atomic pointer to words and names together). `` and change `(`authoredWords`, an atomic pointer)` to `(`authored`)`.
- In the "Baubles: one carrier, catalog-backed identity" section, add: `` - **Finder-only baubles and viewer-aware accessors** (`bauble_viewer.go`; owner ruling 2026-09-29). `BaubleView.FinderUserId` and `Finder` carry a finder-only record's own text beside the generic view. `GetSpec` and everything built on it (`Name`, `NameSimple`, `DisplayName`, `NameComplex`, `GetLongDescription`, templates, GMCP) show the generic view to everyone; `GetSpecFor(viewerUserId)`, `DisplayNameFor`, `NameFor` and `LongDescriptionFor` show the finder their own text, and are each other item's viewer-agnostic twin. Call them only where the output reaches that one viewer (the repo-root guard pins every caller and its call count). `displayNameFrom` and `longDescriptionFrom` are the spec-taking bodies of `DisplayName` and `GetLongDescription`. ``
- Add: `` - **Matching a finder-only bauble** (accepted, owner review 2026-09-29): `NameMatch` and `matchStrength` also match its own words (`baubleFinderNames`) so its finder can type what they read. A stranger who guesses a word of that hidden name can match the trinket with it too; no text is ever shown to them, and the item still reads "Trinket". ``
- Add: `` - **Model-safe accessors** (`bauble_model.go`): `ModelName` and `ModelDescription` show any bauble whose text a player's own key wrote (`BaubleView.PlayerText`), moderated or not, as its carrier's own spec ("Curious Trinket"). Anything a language model is told uses these (the AI companion's perception, scene and actions). ``
- Add: `` - A bauble is never enchanted (`enchantments.ApplyTier` returns at once): an item with a `Spec` never consults the catalog again, so a baked bauble would outlive a retire and the finder-only view. ``

- [ ] **Step 4: `internal/apiframework/context.md`**

In the **breaker.go** bullet, after `Allow`, `Record`, `Release` add `` `RecordConsumer` (one outcome against a consumer's own breaker alone, no ticket and so never the half-open probe: for a feature's check that is not a model call, such as baubles' moderation, which uses its own consumer name `baubles-moderation`) ``.

- [ ] **Step 5: `internal/actions/context.md`, `internal/usercommands/context.md`, `modules/gmcp/context.md`, `modules/aicompanion/context.md`, `internal/enchantments/context.md`**

- `internal/actions/context.md`, in the bauble search paragraph: `` `BaubleRequestForRecord` omits a `PlayerKey` record's name. The find lines (`BaubleDelivery.deliver`) and the pickpocket success line (`takeFromMob`) name a bauble as their one reader sees it (`items.Item.DisplayNameFor`), so a finder reads their own finder-only bauble; everywhere else it is "Trinket". ``
- `internal/usercommands/context.md`, beside the bauble look and appraise notes: `` A finder-only bauble (unmoderated player-key text) reads as "Trinket" to everyone but its finder. `inventory`, `look` (a carried or floor item, and the room's floor and stash listing) and the bauble `appraise` ask for the reader's own view (`DisplayNameFor`, `NameFor`, `LongDescriptionFor`, `GetSpecFor`, `baubles.Record.MaterialFor`); every room line and every other command keeps the viewer-agnostic name. `bauble show` prints `player key`, `moderated` and `finder only`; `bauble list [n] [playerkey|unmoderated|finderonly]` filters. ``
- `modules/gmcp/context.md`, beside `Char.Inventory` and `Room.Info`: `` Item names in the backpack, the bandolier, the component bag and `Room.Info.Contents.Items` are the recipient's own view (`items.Item.NameFor(user.UserId)`), so a finder reads their own finder-only bauble; every other payload uses the viewer-agnostic name. ``
- `modules/aicompanion/context.md`, in its perception or prompt section: `` Every item name and description told to the model goes through `items.Item.ModelName` and `ModelDescription`: text a player's own key wrote, moderated or not, never reaches a model prompt (spec S3). ``
- `internal/enchantments/context.md`, at `ApplyTier`: `` It refuses a bauble (`items.Item.IsBauble`): a baked `Spec` would outlive the catalog's text. ``

- [ ] **Step 6: `_datafiles/config.yaml` comments**

Confirm the disk copy and the bit exactly as Task 5a Step 5 does (`git diff --quiet HEAD -- _datafiles/config.yaml; echo "differs=$?"; git ls-files -v _datafiles/config.yaml`), and follow the same `differs=1` procedure if needed. Replace
```yaml
    MaxConcurrent: 4           # more finds at once than this are generic
    # Check names with the moderation endpoint (needs the server's key);
    # fails closed. A find named on a player's own key on a server with no
    # key of its own is not moderated, as with the AI companion.
```
with
```yaml
    MaxConcurrent: 4           # server-key calls at once; more are generic
    # Check the name, keyword, description and material with the moderation
    # endpoint (needs the server's key). A flag, or a check that is made and
    # fails, keeps the find out on either key. Server-key text that cannot
    # be checked is kept out. Text a finder's own key wrote that cannot be
    # checked (no server key, this off, a breaker open) is shown to its
    # finder alone, a plain "Trinket" to everyone else. Either way it must
    # be plain ASCII letters and ' " - , . ! ? (curly quotes and dashes are
    # folded first); text that is not goes to the server's key.
```

- [ ] **Step 7: `docs/aicompanion/settings.md` and `docs/baubles/implementation-plan.md`**

In `docs/aicompanion/settings.md`, replace the paragraph beginning `The key page also has a box, "Also name things I find while searching` (through `page refuses the bauble request shape from a key whose box is not ticked.`) with:

```markdown
The key page also has a box, "Also name things I find while searching (uses
this key)", off unless the player ticks it. Ticked, and with
`Modules.baubles.Enabled` and `UsePlayerKeys` on, the baubles that player
finds are named on their own key through the same relay, with nothing of
theirs in the request (only the room's authored text). Unlike the
companion's speech, a bauble's text is shown to other players, so while
the server can moderate it (`Modules.baubles.ModerateOutput` on, a server
key, no breaker open) it is moderated, and a flag or a failed check refuses
it. When the server cannot, the find is kept to its finder: they read its
name and description, and everyone else sees a plain "Trinket". Either
way it is held to plain ASCII letters and simple punctuation (text that is
not goes to the server's key instead, without counting against the
player's key), never passed to another find's prompt or the companion's,
and its value is rolled by the server. Unticked, their finds use the
server's key, or stay generic trinkets when there is none. The relay page
refuses the bauble request shape from a key whose box is not ticked.
```

In `docs/baubles/implementation-plan.md`, replace the bullet
```markdown
- **Moderation** of a player-key find: a flag refuses; a check that cannot be
  made accepts it unmoderated (a server-key find is refused), so a working
  player key never becomes a trinket over the server's route.
```
with
```markdown
- **Moderation** of a player-key find (hardening, 2026-09-28/29): moderated
  whenever the server can (a flag or a failed check refuses on either key,
  and every check feeds the moderation breaker, not the naming breaker);
  when it cannot (no server key, moderation off, a breaker open), the find
  is FINDER-ONLY: its finder reads it, everyone else sees a plain Trinket.
  Player-key text must also pass `baubles.CheckPlayerKeyText` after curly
  quotes and dashes are folded, and text that does not goes to the
  server's key without counting against the player's.
```

- [ ] **Step 8: The context.md audit, and commit**

Run: `cd /c/tmp/dogmud-baubles-h && python tools/context_md_audit.py 2>&1 | grep -E "baubles|items|apiframework|actions|usercommands|gmcp|aicompanion|enchantments" ; echo "exit=$?"`
Expected: no phantom symbols in these packages.

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/baubles/context.md modules/baubles/context.md internal/items/context.md internal/apiframework/context.md internal/actions/context.md internal/usercommands/context.md modules/gmcp/context.md modules/aicompanion/context.md internal/enchantments/context.md _datafiles/config.yaml docs/aicompanion/settings.md docs/baubles/implementation-plan.md && git commit -F - <<'EOF'
docs(baubles): PR H2 moderation, finder-only text and model-safe names

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```
If Step 6 staged `config.yaml` through `update-index`, leave it out of `git add` and confirm with `git diff --cached --stat`.

---

### Task 11b: PR H2 gate

**Files:** none new. Every diff runs from `$BASE` (Task H2-0 Step 1).

- [ ] **Step 1: gofmt on the committed blobs**

```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && for f in $(git diff --name-only $BASE..HEAD -- '*.go'); do out=$(git show HEAD:"$f" | gofmt -l); [ -n "$out" ] && echo "UNFORMATTED: $f"; done; echo done
```
Expected: only `done`.

- [ ] **Step 2: Size**

```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && git diff --shortstat $BASE..HEAD
```
Expected: under 300 files and under 20,000 lines.

- [ ] **Step 3: Build, vet, tests**

```bash
cd /c/tmp/dogmud-baubles-h && go build ./... && go vet ./internal/baubles/ ./internal/items/ ./internal/apiframework/ ./internal/actions/ ./internal/usercommands/ ./internal/enchantments/ ./modules/baubles/ ./modules/aicompanion/ ./modules/gmcp/ . && go test ./internal/baubles/ ./internal/items/ ./internal/apiframework/ ./internal/actions/ ./internal/usercommands/ ./internal/enchantments/ ./modules/baubles/ ./modules/aicompanion/ ./modules/gmcp/ . 2>&1 | tail -14
```
Expected: no vet output, every package `ok`.

- [ ] **Step 4: The guards this PR leans on, by name**

```bash
cd /c/tmp/dogmud-baubles-h && go test . -count=1 -run "TestFinderViewReachesOnlyItsReader|TestFinderViewGuardCatchesALeak|TestNarrationSitesMatchViewpointAudit|TestEveryTextSurfaceIsRegistered|TestEveryCreatureLookupDeclaresItsViewer|TestNoDisplayReadsRawConfig" -v 2>&1 | grep -E "^(--- FAIL|--- PASS|FAIL|ok)"
```
```bash
cd /c/tmp/dogmud-baubles-h && go test ./internal/apiframework/ ./internal/items/ -count=1 -run "TestNoTestPrintsAKey|TestRecordConsumerFeedsOnlyItsOwnBreaker|TestFinderOnlyBaubleIsGenericToEveryoneButItsFinder|TestModelAccessorsKeepPlayerKeyTextOut" -v 2>&1 | grep -E "^(--- FAIL|--- PASS|FAIL|ok)"
```
Expected: every named test `--- PASS`.

- [ ] **Step 5: Boot smoke in the test process (no port)**

```bash
cd /c/tmp/dogmud-baubles-h && DOGMUD_BOOT_SMOKE=1 go test . -count=1 -run TestSmoke_ServerBootsCleanWithRealData -timeout 600s 2>&1 | tail -5
```
Expected: `ok`.

- [ ] **Step 6: A real boot on private ports, stopped by its own PID**

Check the fixed boot path is free:
```bash
git -C /c/tmp/dogmud-baubles-h worktree list | grep -F "dogmud-boot-check"; ls /c/tmp/dogmud-boot-check 2>/dev/null | head -3
```
Expected: no output (each exits 1); if the directory exists, STOP and ask.

Build (Bash):
```bash
git -C /c/tmp/dogmud-baubles-h worktree add --detach /c/tmp/dogmud-boot-check HEAD && cd /c/tmp/dogmud-boot-check && go build -o boot-check.exe . && cat > /c/tmp/dogmud-boot-check.overrides.yaml <<'EOF'
Network:
  TelnetPort: [33533]
  LocalPort: 9899
  HttpPort: 8391
  HttpsPort: 0
  AIPort: 0
EOF
echo built
```

Run (PowerShell):
```powershell
$env:CONFIG_PATH = 'C:\tmp\dogmud-boot-check.overrides.yaml'
$p = Start-Process -FilePath 'C:\tmp\dogmud-boot-check\boot-check.exe' -WorkingDirectory 'C:\tmp\dogmud-boot-check' -RedirectStandardOutput 'C:\tmp\dogmud-boot-check.log' -RedirectStandardError 'C:\tmp\dogmud-boot-check.err' -WindowStyle Hidden -PassThru
$deadline = (Get-Date).AddSeconds(180)
while ((Get-Date) -lt $deadline -and -not $p.HasExited -and -not (Select-String -Path 'C:\tmp\dogmud-boot-check.log','C:\tmp\dogmud-boot-check.err' -Pattern 'Server Ready' -Quiet)) { Start-Sleep -Seconds 2 }
"pid=$($p.Id) exited=$($p.HasExited)"
Select-String -Path 'C:\tmp\dogmud-boot-check.log','C:\tmp\dogmud-boot-check.err' -Pattern 'Server Ready|status=listening|^panic:|goroutine [0-9]+ \[running\]|runtime error|bind:' | Select-Object -First 20
if (-not $p.HasExited) { Stop-Process -Id $p.Id -Force -Confirm:$false }
Remove-Item Env:\CONFIG_PATH
```
Expected: `exited=False`, `Server Ready`, `Telnet status=listening port=33533` (and no 33333 or 44444), no `panic:`, `goroutine ... [running]`, `runtime error` or `bind:` line. Never grep the bare word `panic`.

Clean up (PowerShell), then `git -C /c/tmp/dogmud-baubles-h worktree prune`:
```powershell
Remove-Item -Recurse -Force 'C:\tmp\dogmud-boot-check'; Remove-Item -Force 'C:\tmp\dogmud-boot-check.overrides.yaml','C:\tmp\dogmud-boot-check.log','C:\tmp\dogmud-boot-check.err'
```

- [ ] **Step 7: Race, in the Linux test container**

```bash
cd /c/tmp/dogmud-baubles-h && docker compose -f compose.test.yml run --build --rm test go test -race -count=1 ./internal/items/ ./internal/baubles/ ./internal/apiframework/ ./internal/actions/ ./modules/baubles/ ./modules/aicompanion/ 2>&1 | tail -10
```
Expected: `ok` for all six, no `WARNING: DATA RACE`. This run IS the race gate until CI has minutes again.

- [ ] **Step 8: Lint, go.mod, docs, tree**

```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && golangci-lint run --new-from-merge-base=$BASE
```
Expected: `0 issues`.
```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && git diff --exit-code $BASE..HEAD -- go.mod go.sum; echo "exit=$?"; git diff --name-only $BASE..HEAD | sed -n 's#^\(internal/[^/]*\|modules/[^/]*\)/.*#\1#p' | sort -u; git diff --name-only $BASE..HEAD -- '*context.md' _datafiles/config.yaml docs/ _datafiles/world/dogmud/templates/; git status --short
```
Expected: `exit=0`; every package in the first list has its `context.md` in the second; `_datafiles/config.yaml`, both docs and `command.bauble.template` present; a clean tree.

- [ ] **Step 9: Hand over**

No push from here. Report the branch, `$BASE`, `git log --oneline $BASE..HEAD` and each gate result. When the owner says to ship: `git push -u origin fix/baubles-hardening-h2`, `gh pr create --repo pruuk/DOGMud --base master --head fix/baubles-hardening-h2 --title "Slice H2: player-key moderation, finder-only text, model-safe names"`, and read back that the URL says `pruuk/DOGMud`. PR H3 starts only after this one has merged.

---

## PR H3: U2, U4 and the steal work

---

### Task H3-0: Branch PR H3 from master

**Files:** none in the repo.

- [ ] **Step 1: PR H2 is merged; switch the worktree**

```bash
cd /c/tmp/dogmud-baubles-h && gh pr list --repo pruuk/DOGMud --state merged --head fix/baubles-hardening-h2 --json number,mergedAt && git status --short && git fetch -q origin && git switch -c fix/baubles-hardening-h3 origin/master && BASE=$(git rev-parse HEAD) && echo "$BASE" > /c/tmp/dogmud-baubles-h.base && echo "BASE=$BASE"
```
Expected: one merged PR, a clean tree, a new `BASE`. An empty list: STOP, H3 waits for H2.

- [ ] **Step 2: H3's anchors exist**

```bash
cd /c/tmp/dogmud-baubles-h && anchors=$(mktemp) && cat > "$anchors" <<'EOF'
internal/baubles/find.go|chance := s.chanceFor(o.Place.Biome, o.SkillFactor)
internal/baubles/find.go|SkillFactor float64
internal/actions/search_bauble.go|var searchBaubleRoll = func(o baubles.FindOpts) (baubles.ValueTier, bool) {
internal/actions/search_bauble.go|SkillFactor: BaubleSkillFactor(actor.GetCharacter()),
internal/actions/search_feature.go|Feature:     feature.WindowName(),
sight_penalty_guard_test.go|"forager":  {"ForageCore": true},
sight_penalty_guard_test.go|func TestSightPenaltyGuardSeesTheSpellAlias(
internal/actions/stolen_bauble.go|`<ansi fg="mobname">%s</ansi> points at %s. "That's mine! Thief!"`,
internal/actions/steal.go|is caught trying to pocket the <ansi fg="itemname">%s</ansi>!`,
internal/actions/search.go|is snooping around.
internal/usercommands/look.go|is looking around.
internal/usercommands/look.go|is looking into the room from somewhere...
internal/actions/get.go|matchItem, found := room.FindOnFloor(itemName, stash)
internal/usercommands/get.go|if peekFound && !getFromStash && peekItem.BaubleBelongsTo(room.RoomId) {
internal/actions/steal.go|// Deliberately NOT mobs.CheckPlayerHarm: that policy also blocks charmed
internal/actions/steal.go|func thiefCaught(actor Actor, m *mobs.Mob, room *rooms.Room) {
internal/actions/steal.go|witnesses := crimes.WitnessesInRoom(factionIds, room, 0)
internal/actions/steal_pocket.go|if !p.success {
internal/actions/pickpocket_test.go|h2.thief.room = newSearchTestRoom(9698) // walks off
modules/aicompanion/actions.go|return actionOutcome{Refused: `cannot pick that up`}
_datafiles/world/dogmud/templates/help/steal.template|fight before it is done, and you lose your chance. Now and then
EOF
missing=0; while IFS= read -r line; do f=${line%%|*}; a=${line#*|}; grep -qF -- "$a" "$f" || { echo "MISSING in $f: $a"; missing=1; }; done < "$anchors"; rm -f "$anchors"; echo "missing=$missing"
```
Expected: `missing=0`. A `MISSING` line: STOP before the task that edits it.

- [ ] **Step 3: Baseline green**

```bash
cd /c/tmp/dogmud-baubles-h && go build ./... && go test ./internal/baubles/ ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ ./internal/rooms/ ./modules/aicompanion/ . 2>&1 | tail -10
```
Expected: every package `ok`.

---

### Task 12: U2, `RollFind` pays a sight penalty

**Files:**
- Modify: `internal/baubles/find.go` (`FindOpts`, `RollFind`, new `clampUnit`)
- Modify: `internal/baubles/find_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/baubles/find_test.go`:

```go
// The searcher's sight costs the chance (owner ruling 2026-09-28, lighting
// plan 5b's ramp): FindOpts.SightPenalty is 1 - messaging.SightMult, 0 is
// none. A search in the dark still spends its window roll, exactly as one
// in the light does.
func TestRollFindPaysTheSightPenalty(t *testing.T) {
	resetAllWindowsForTest()
	t0 := time.Unix(4_000_000, 0)

	// 40000 in a million is 4%: under an unskilled indoor search's 5%...
	if _, found := RollFind(FindOpts{Place: NewPlace(601, `town`, ``, `interior`), Randn: always(40000), Now: t0}); !found {
		t.Fatal("4% hits 5% in the light")
	}
	// ...and not under 5% x (1 - 0.3) = 3.5%, well clear of the 4% roll
	// (a case exactly on the threshold would hang on float rounding).
	if _, found := RollFind(FindOpts{Place: NewPlace(602, `town`, ``, `interior`), SightPenalty: 0.3, Randn: always(40000), Now: t0}); found {
		t.Fatal("the dark costs the chance")
	}
	if used, _, _, open := WindowState(602, 0, t0); !open || used != 1 {
		t.Fatalf("a search in the dark spends its window roll: used %d open %v", used, open)
	}

	// Out of range is clamped: below 0 is none, above 1 is all.
	if _, found := RollFind(FindOpts{Place: NewPlace(603, `town`, ``, `interior`), SightPenalty: -3, Randn: always(40000), Now: t0}); !found {
		t.Fatal("a negative penalty is none")
	}
	if _, found := RollFind(FindOpts{Place: NewPlace(604, `town`, ``, `interior`), SightPenalty: 7, Randn: always(0), Now: t0}); found {
		t.Fatal("a penalty above 1 leaves no chance")
	}
	if used, _, _, open := WindowState(604, 0, t0); !open || used != 1 {
		t.Fatalf("even a hopeless search spends its window roll: used %d open %v", used, open)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/baubles/ -run TestRollFindPaysTheSightPenalty -v`
Expected: build failure `unknown field SightPenalty in struct literal`.

- [ ] **Step 3: Implement**

In `internal/baubles/find.go` `FindOpts`, after `SkillFactor float64` add:

```go

	// SightPenalty is what the searcher's sight costs the chance, from 0
	// (none: the zero value, so a FindOpts without it behaves as before) to
	// 1 (nothing can be found). The caller passes 1 - messaging.SightMult
	// for the searcher in the room (lighting plan 5b's ramp).
	SightPenalty float64
```

In `RollFind`, replace

```go
	chance := s.chanceFor(o.Place.Biome, o.SkillFactor)
	if chance <= 0 {
		return ``, false
	}
	if o.Feature == `` && !takeRoll(o.Place.RoomId, o.UserId, s.perPlayer, s.rolls, s.window, now) {
		return ``, false
	}
```
with
```go
	chance := s.chanceFor(o.Place.Biome, o.SkillFactor)
	if chance <= 0 {
		return ``, false
	}
	// The searcher's sight, after the check above: a search in the dark
	// still spends its window roll exactly as one in the light does, so
	// only a place where nothing is ever found opens no window.
	chance *= 1 - clampUnit(o.SightPenalty)
	if o.Feature == `` && !takeRoll(o.Place.RoomId, o.UserId, s.perPlayer, s.rolls, s.window, now) {
		return ``, false
	}
```

and replace the `mudlog.Info` line with:

```go
	mudlog.Info(`baubles`, `action`, `found`, `tier`, string(tier), `household`, o.Household, `chancePct`, chance, `sightPenalty`, o.SightPenalty, `biome`, o.Place.Biome, `feature`, o.Feature, `roomId`, o.Place.RoomId, `zone`, o.Place.Zone, `userId`, o.UserId)
```

Add at the end of the file:

```go
// clampUnit holds v to 0..1.
func clampUnit(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
```

- [ ] **Step 4: Run all baubles tests**

Run: `go test ./internal/baubles/`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/baubles/find.go internal/baubles/find_test.go && git commit -F - <<'EOF'
feat(baubles): the bauble roll takes a sight penalty

FindOpts.SightPenalty (0 is none) multiplies the chance after the
nothing-here check, so a search in the dark still spends its window roll.
The logged chance is the post-sight one.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 13: U2, both bauble searches pay the sight ramp, and the guard sees them

**Files:**
- Modify: `internal/actions/search_bauble.go` (`searchForBauble`), `internal/actions/search_feature.go` (`searchFeatureForBauble`)
- Create: `internal/actions/search_bauble_sight_test.go`
- Modify: `sight_penalty_guard_test.go`

- [ ] **Step 1: Write the failing actions tests**

Create `internal/actions/search_bauble_sight_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// darkBaubleRoom is a feature room in an unlit cave: a searcher with no
// vision of their own pays the sight ramp there.
func darkBaubleRoom(t *testing.T, roomId int) *rooms.Room {
	t.Helper()
	t.Cleanup(rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"cave":    {BiomeId: "cave", SkyLight: rooms.SkyLightPtr(0.0)},
		"default": {BiomeId: "default"},
	}))
	r := featureRoom(roomId)
	r.Biome = "cave"
	return r
}

// Both bauble rolls, the room's and a feature's, pass the searcher's sight
// as FindOpts.SightPenalty (owner ruling 2026-09-28).
func TestSearch_Bauble_RollsPayTheSightRamp(t *testing.T) {
	pinConfigForTest(t)
	room := darkBaubleRoom(t, 9631)
	actor := newSearchFakeActor("Groper", room, true, 7631)
	stubBaubleSearch(t, false, actor)
	var penalties []float64
	searchBaubleRoll = func(o baubles.FindOpts) (baubles.ValueTier, bool) {
		penalties = append(penalties, o.SightPenalty)
		return ``, false
	}

	want := 1 - messaging.SightMult(actor.char, room)
	if want <= 0 {
		t.Fatalf("fixture: an unlit cave must cost sight, SightMult %v", 1-want)
	}

	Search(actor, SearchOptions{})
	actor.char.Cooldowns = nil
	Search(actor, SearchOptions{Feature: "hearth"})

	if len(penalties) != 2 || penalties[0] != want || penalties[1] != want {
		t.Fatalf("room then feature roll, each at penalty %v: got %v", want, penalties)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/actions/ -run TestSearch_Bauble_RollsPayTheSightRamp -v`
Expected: FAIL "got [0 0]". (`Search` has no darkness refusal, so both rolls are reached in the cave; if the output shows fewer than two penalties, the fixture is not reaching a roll and must be fixed before Step 3.)

- [ ] **Step 3: Pass the penalty from both callers**

In `internal/actions/search_bauble.go` `searchForBauble`, replace

```go
	tier, found := searchBaubleRoll(baubles.FindOpts{
		Place:       BaublePlace(room),
		UserId:      actor.GetUserId(),
		SkillFactor: BaubleSkillFactor(actor.GetCharacter()),
		Household:   household,
	})
```
with
```go
	tier, found := searchBaubleRoll(baubles.FindOpts{
		Place:       BaublePlace(room),
		UserId:      actor.GetUserId(),
		SkillFactor: BaubleSkillFactor(actor.GetCharacter()),
		// sight ramp (plan 5b): the searcher needs to see what glints.
		SightPenalty: 1 - messaging.SightMult(actor.GetCharacter(), room),
		Household:    household,
	})
```

In `internal/actions/search_feature.go` `searchFeatureForBauble`, replace

```go
	tier, found := searchBaubleRoll(baubles.FindOpts{
		Place:       BaublePlace(room),
		UserId:      actor.GetUserId(),
		SkillFactor: BaubleSkillFactor(actor.GetCharacter()),
		Feature:     feature.WindowName(),
		Household:   household,
	})
```
with
```go
	tier, found := searchBaubleRoll(baubles.FindOpts{
		Place:       BaublePlace(room),
		UserId:      actor.GetUserId(),
		SkillFactor: BaubleSkillFactor(actor.GetCharacter()),
		// sight ramp (plan 5b): the searcher needs to see what glints.
		SightPenalty: 1 - messaging.SightMult(actor.GetCharacter(), room),
		Feature:      feature.WindowName(),
		Household:    household,
	})
```

(`search_feature.go` already imports `messaging`; run `go build ./internal/actions/` to confirm, and add the import if the build says otherwise.)

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/actions/ -run "TestSearch_Bauble|TestSearchFeature" -v 2>&1 | grep -E "^(--- FAIL|FAIL|ok)"`
Expected: only `ok`.

- [ ] **Step 5: Write the failing guard probe**

In `sight_penalty_guard_test.go`, append after `TestSightPenaltyGuardSeesTheSpellAlias`:

```go
// TestSightPenaltyGuardSeesTheBaubleRollSeam proves the guard sees the
// bauble roll through its test seam: a caller of searchBaubleRoll with no
// sight helper is reported, one that calls SightMult is not, and the seam's
// initialiser (which forwards to baubles.RollFind) is reported until
// exempted by its file|var key.
func TestSightPenaltyGuardSeesTheBaubleRollSeam(t *testing.T) {
	fset := token.NewFileSet()
	f := parseSightSource(t, fset, "internal/actions/search_bauble.go", `package actions

var searchBaubleRoll = func(o baubles.FindOpts) (baubles.ValueTier, bool) { return baubles.RollFind(o) }

func dark(a Actor, r *rooms.Room) { searchBaubleRoll(baubles.FindOpts{}) }

func lit(a Actor, r *rooms.Room) {
	searchBaubleRoll(baubles.FindOpts{SightPenalty: 1 - messaging.SightMult(a.GetCharacter(), r)})
}
`)
	got := findUnpenalisedRollSites(fset, []sightFile{f}, nil)
	if len(got) != 2 || got[0].Func != "var searchBaubleRoll" || got[0].Call != "baubles.RollFind" ||
		got[1].Func != "dark" || got[1].Call != "actions.searchBaubleRoll" {
		t.Fatalf("want the seam's initialiser and dark reported, got:\n  %s", formatSightSites(got))
	}
	got = findUnpenalisedRollSites(fset, []sightFile{f}, map[string]string{"internal/actions/search_bauble.go|var searchBaubleRoll": "test"})
	if len(got) != 1 || got[0].Func != "dark" {
		t.Fatalf("exempting the seam leaves only dark, got:\n  %s", formatSightSites(got))
	}
}
```

Run: `go test . -run TestSightPenaltyGuardSeesTheBaubleRollSeam -v`
Expected: FAIL "want the seam's initialiser and dark reported, got:" with nothing listed (neither name is guarded yet).

- [ ] **Step 6: Guard the roll and exempt the seam**

In `sight_penalty_guard_test.go`, in `guardedSightFuncs`, after the `"forager"` row add:

```go
	// The bauble search roll pays the sight ramp through SightMult (owner
	// ruling 2026-09-28). RollFind is the roll; searchBaubleRoll is the
	// actions seam both bauble searches call it through, so both are
	// guarded: a new caller of either must pass SightPenalty.
	"baubles": {"RollFind": true},
	"actions": {"searchBaubleRoll": true},
```

In `sightExemptSites`, after the `executeCounterTaunt` row add:

```go
	// The bauble roll seam: a package-level initialiser that only forwards
	// to baubles.RollFind. Its two callers are guarded as
	// actions.searchBaubleRoll and each passes SightPenalty from
	// messaging.SightMult once.
	"internal/actions/search_bauble.go|var searchBaubleRoll": "test seam forwarding to baubles.RollFind; searchForBauble and searchFeatureForBauble pass SightPenalty from SightMult once each",
```

Update the file's header note on `guardedSightFuncs`: add `baubles.RollFind` and `actions.searchBaubleRoll` to the sentence listing what is watched, if one exists; the map comment above suffices otherwise.

- [ ] **Step 7: Run the whole guard**

Run: `go test . -run "TestSightPenaltyGuard|TestEveryRollSiteAppliesTheSightPenalty" -v`
Expected: PASS (the probe, the real-tree walk, and no stale row).

- [ ] **Step 8: Probe the real tree**

Temporarily delete the `SightPenalty:` line (and its comment) from `searchForBauble` in `internal/actions/search_bauble.go`. Run `go test . -run TestEveryRollSiteAppliesTheSightPenalty -v`. Expected: FAIL listing `internal/actions/search_bauble.go:<line> in searchForBauble: actions.searchBaubleRoll`. Restore the line (`git diff internal/actions/search_bauble.go` must show only Step 3's change), rerun, PASS.

Also temporarily delete the new `sightExemptSites` row and rerun: expected FAIL listing `internal/actions/search_bauble.go:55 in var searchBaubleRoll: baubles.RollFind` (55 at `3bd6ccaa3`; the line shifts if the file shifted). Restore, rerun, PASS.

- [ ] **Step 9: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/actions/search_bauble.go internal/actions/search_feature.go internal/actions/search_bauble_sight_test.go sight_penalty_guard_test.go && git commit -F - <<'EOF'
feat(search): the bauble search chance pays the sight ramp

Both bauble searches pass 1 - messaging.SightMult as SightPenalty. The
sight guard now watches baubles.RollFind and actions.searchBaubleRoll,
exempting only the seam's forwarding initialiser; both probes went red.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 14: U4, look, search and stolen-bauble observer lines hide names by sight

Finding recorded in the facts table: each of these lines already tags the
actor as `<ansi fg="username">`, which the pipeline's `Anonymize` hides from
a shapes-only reader, so this changes no output today. It makes each line
carry its names explicitly, as every other name-bearing observer line does,
so a lost or changed tag cannot leak the name.

Sibling lines (CLAUDE.md: finish sibling paths): PR #175's bauble theft
code has two observer lines of the same shape that name a player, and both
move too: `ownerRecognizes` in `stolen_bauble.go` ("points at" the
carrier) and `stealHouseholdBauble` in `steal.go` ("is caught trying to
pocket"). Out of scope, recorded: `stolen_bauble.go`'s "looks overjoyed"
line (about `:294`) names only the mob and the item, so it has no player
name to hide; and master's two crime lines in `steal.go` ("gets caught
trying to steal", "is caught stealing from", master `steal.go:303, 583` at
`8c6561c5a`, `:364, 537` at `e711ee9de`) predate the PR, are not bauble
code, and already hide the thief through the same username tag. They
belong to the messaging arc's remaining slices, not this one.

**Files:**
- Modify: `internal/rooms/participant_sight_test.go`
- Modify: `internal/actions/search.go`, `internal/usercommands/look.go`, `internal/actions/stolen_bauble.go`, `internal/actions/steal.go`

- [ ] **Step 1: Write the characterization test**

Append to `internal/rooms/participant_sight_test.go`:

```go
// The look and search observer lines tag the actor as a username, so a
// shapes-only reader already reads "a figure" through the pipeline's
// Anonymize; SendTextVisualHidingNames with the actor's name keeps that
// true when the name reaches the line without its tag.
func TestLookAndSearchObserverLinesHideTheActor(t *testing.T) {
	r := sightTestRoom(t, "cave")
	if !users.GetByUserId(7413).Character.Conditions.AddCondition(sightTestInfraredConditionId, true) {
		t.Fatal("precondition: the observer should now carry infrared")
	}
	tagged := `<ansi fg="username">Aliceia</ansi> is snooping around.`
	r.SendTextVisual(messaging.CategoryMobEmote, tagged, 7411)
	r.SendTextVisualHidingNames(messaging.CategoryMobEmote, tagged, []string{"Aliceia"}, 7411)
	r.SendTextVisualHidingNames(messaging.CategoryMobEmote, `Aliceia is snooping around.`, []string{"Aliceia"}, 7411)

	got := sightTestPlain(events.DrainQueuedMessagesForTest(7413))
	want := "A figure is snooping around."
	if len(got) != 3 || got[0] != want || got[1] != want || got[2] != want {
		t.Fatalf("infrared observer read %q, want %q three times", got, want)
	}
}
```

- [ ] **Step 2: Run it, then prove it can fail**

Run: `go test ./internal/rooms/ -run TestLookAndSearchObserverLinesHideTheActor -v`
Expected: PASS (it pins today's behaviour; the finding above).

Plainly: this test pins the room method's behaviour (a tagged line is hidden either way, a bare name is hidden only when named). It does NOT guard the conversion: it passes before Step 3 and after it, and it would still pass if Step 3 missed a call. Step 4's grep is the conversion check.

Probe: change the third call's names argument to `nil`, rerun. Expected: FAIL showing `"Aliceia is snooping around."` as the third line. Restore, rerun, PASS.

- [ ] **Step 3: Move every observer line in the two files**

The transformation is the same for each call: the method becomes `SendTextVisualHidingNames`, and a names argument goes in immediately before the excluded user id. Multi-line shape, before and after:

```go
			room.SendTextVisual(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> is looking around.`, user.Character.Name),
				user.UserId,
			)
```
```go
			room.SendTextVisualHidingNames(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> is looking around.`, user.Character.Name),
				[]string{user.Character.Name},
				user.UserId,
			)
```

Single-line shape, before and after:

```go
			room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(`<ansi fg="username">%s</ansi> peers toward the %s.`, user.Character.Name, exitName), user.UserId)
```
```go
			room.SendTextVisualHidingNames(messaging.CategoryMobEmote, fmt.Sprintf(`<ansi fg="username">%s</ansi> peers toward the %s.`, user.Character.Name, exitName), []string{user.Character.Name}, user.UserId)
```

Apply it with the Edit tool (never a script) to every call below, identified by its format string:

| File | Format string (identifies the call) | names argument |
|---|---|---|
| `internal/actions/search.go` | `is searching the %s.` | `[]string{char.Name}` |
| `internal/actions/search.go` | `is snooping around.` | `[]string{char.Name}` |
| `internal/usercommands/look.go` | `is looking around.` | `[]string{user.Character.Name}` |
| `internal/usercommands/look.go` | `is looking at <ansi fg="username">%s</ansi>.` | `[]string{user.Character.Name, u.Character.Name}` |
| `internal/usercommands/look.go` | `is looking at %s.`, the MOB branch (`targetName`) | `[]string{user.Character.Name}` |
| `internal/usercommands/look.go` | `peers toward the %s.` | `[]string{user.Character.Name}` |
| `internal/usercommands/look.go` | `is admiring their` | `[]string{user.Character.Name}` |
| `internal/usercommands/look.go` | `is examining the <ansi fg="noun">%s</ansi>.` | `[]string{user.Character.Name}` |
| `internal/usercommands/look.go` | `is looking at %s.`, the PET branch (`petUser.Character.Pet.DisplayName()`) | `[]string{user.Character.Name}` |
| `internal/usercommands/look.go` | `is looking at the <ansi fg="%s">%s</ansi>.` (corpse) | `[]string{user.Character.Name}` |
| `internal/usercommands/look.go` | `is looking at the <ansi fg="item">%s</ansi> %s.` (floor item) | `[]string{user.Character.Name}` |
| `internal/usercommands/look.go` | `is looking into the room from somewhere...` | `[]string{user.Character.Name}` |
| `internal/usercommands/look.go` | `is looking into the room from the <ansi fg="exit">%s</ansi> exit` | `[]string{user.Character.Name}` |
| `internal/actions/stolen_bauble.go` | `points at %s. "That's mine! Thief!"` (`ownerRecognizes`; `%s` is `who`, the tagged name or "a figure") | `[]string{carrier.GetCharacter().Name}` |
| `internal/actions/steal.go` | `is caught trying to pocket the <ansi fg="itemname">%s</ansi>!` (`stealHouseholdBauble`) | `[]string{actor.GetName()}` |

In `search.go` and `steal.go` the excluded id is `actor.GetUserId(),`; the names argument goes on its own line before it. In `stolen_bauble.go` (after FinalTwist's recognition gate, `:253-255` at `3bd6ccaa3`) the call ends `m.Character.Name, who), carrier.GetUserId())`; the result is:

```go
	room.SendTextVisualHidingNames(messaging.CategoryMobEmote, fmt.Sprintf(
		`<ansi fg="mobname">%s</ansi> points at %s. "That's mine! Thief!"`,
		m.Character.Name, who), []string{carrier.GetCharacter().Name}, carrier.GetUserId())
```

`who` is the carrier's tagged name when the OWNER sees clearly, else "a figure"; hiding the carrier's name from each observer by that observer's own sight is still needed, since the owner seeing clearly says nothing about a bystander in the dark.

The names are the values each format string prints in its username tag (`actor.GetName()` in `steal.go`, `carrier.GetCharacter().Name` in `stolen_bauble.go`), so what is hidden is exactly what is shown.

- [ ] **Step 4: Confirm none is left and the build is clean**

Run (each check standalone, since `grep -c` exits 1 on zero):
```bash
cd /c/tmp/dogmud-baubles-h && grep -n "SendTextVisual(" internal/usercommands/look.go internal/actions/search.go
```
Expected: no output (exit 1 is the success case here).

```bash
cd /c/tmp/dogmud-baubles-h && grep -n -A2 "SendTextVisual(" internal/actions/stolen_bauble.go internal/actions/steal.go
```
Expected: exactly three calls, none of them this task's: the "looks overjoyed" line in `stolen_bauble.go`, and in `steal.go` the "gets caught trying to steal" and "is caught stealing from" lines (master's, out of scope above). If "points at" or "caught trying to pocket" appears in the output, that call was missed.

```bash
cd /c/tmp/dogmud-baubles-h && go build ./... && go vet ./internal/usercommands/ ./internal/actions/
```
Expected: clean.

- [ ] **Step 5: Run the affected packages and the root guards**

Run: `go test ./internal/usercommands/ ./internal/actions/ ./internal/rooms/ . 2>&1 | tail -10`
Expected: all `ok`. The messaging surface guard reads `SendTextVisualHidingNames` on a `room` receiver as an observer, so its registry needs no change; if it does fail, read the failure and register nothing without first checking the claim against source.

- [ ] **Step 6: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/rooms/participant_sight_test.go internal/actions/search.go internal/usercommands/look.go internal/actions/stolen_bauble.go internal/actions/steal.go && git commit -F - <<'EOF'
fix(look,search,baubles): observer lines hide the actor's name by sight

Every look and search observer line, and the two bauble theft lines that
name a player, now name their parties to SendTextVisualHidingNames. The
username tag already hid the actor from a shapes-only reader; this keeps
it hidden if the tag is ever lost.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 14a: The household-bauble refusal holds for every taker

Owner ruling 2026-09-29 (amendment, ruling 3). The refusal lived only in a
player's `get` (a peek before the shared pickup), so a mob's `get`, a
companion's or a scavenger's took a household's bauble without a word.
It moves into `actions.GetItemFromFloor`, the one floor pickup every
`get` calls. `get all` is unchanged for players (it already skips them).
`hooks.EquipBestFloorItem` removes floor items without it, but only an
equipment upgrade, which a bauble never is (facts table). Keep the gate
block readable: the parity session's sight slice adds a darkness gate to
the same function later.

**Files:**
- Modify: `internal/actions/get.go` (new `ErrHouseholdBauble`, `GetItemFromFloor`)
- Modify: `internal/usercommands/get.go` (the explicit `get <name>` branch)
- Create: `internal/actions/get_household_test.go`, `internal/usercommands/get_household_test.go`

- [ ] **Step 1: Write the failing actions test**

Create `internal/actions/get_household_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A household's bauble on the floor is refused to every taker, not only a
// player's `get` (owner ruling 2026-09-29): mobs, companions and scavengers
// all pick up through GetItemFromFloor. Nothing moves. A bauble that is no
// household's is taken as ever.
func TestGetItemFromFloor_RefusesAHouseholdsBaubleToEveryTaker(t *testing.T) {
	seedBaubleSale(t)
	char := newTestChar()
	room := newTestRoom()
	room.RoomId = 9701
	actor := newStubActor(char, room) // not a player: a mob's, companion's or scavenger's get

	theirs := newBauble(t, "Small Child's Doll", "doll", 3, baubles.StatusReady)
	theirs.BaubleHousehold = room.RoomId
	room.Items = append(room.Items, theirs)

	result := GetItemFromFloor(actor, "doll", false)
	require.True(t, result.Found, "the doll is found")
	require.ErrorIs(t, result.Err, ErrHouseholdBauble)
	assert.Equal(t, 0, countCharItems(char), "nothing taken")
	assert.Equal(t, 1, countFloorItems(room), "the doll stays where it lies")

	room.Items[0].BaubleHousehold = 0 // nobody's household's now
	result = GetItemFromFloor(actor, "doll", false)
	require.True(t, result.Found)
	require.NoError(t, result.Err)
	assert.Equal(t, 1, countCharItems(char), "taken as ever")
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/actions/ -run TestGetItemFromFloor_RefusesAHouseholdsBaubleToEveryTaker -v`
Expected: build failure `undefined: ErrHouseholdBauble`.

- [ ] **Step 3: The gate in the shared pickup**

In `internal/actions/get.go`, replace

```go
import (
	"github.com/GoMudEngine/GoMud/internal/items"
)
```
with
```go
import (
	"errors"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// ErrHouseholdBauble refuses taking a household's bauble off the floor: it
// belongs to the house, and taking it is theft, which only `steal`
// attempts (stealHouseholdBauble). Every taker is held to it, a player's
// `get`, a mob's, a companion's or a scavenger's (owner ruling 2026-09-29).
var ErrHouseholdBauble = errors.New(`that belongs to this household`)
```

and in `GetItemFromFloor` replace

```go
	matchItem, found := room.FindOnFloor(itemName, stash)
	if !found {
		return GetItemResult{Found: false}
	}

	char := actor.GetCharacter()
```
with
```go
	matchItem, found := room.FindOnFloor(itemName, stash)
	if !found {
		return GetItemResult{Found: false}
	}

	// Gates: each refuses with the item it found and an error, and moves
	// nothing; the caller words the refusal. A household's bauble is never
	// picked up (it is only ever on the floor, never in a stash).
	if !stash && matchItem.BaubleBelongsTo(room.RoomId) {
		return GetItemResult{Item: matchItem, Found: true, Err: ErrHouseholdBauble}
	}

	char := actor.GetCharacter()
```

Add to `GetItemFromFloor`'s doc comment: `A household's bauble is refused with ErrHouseholdBauble (Found, nothing moved).`

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/actions/ -run "TestGetItemFromFloor" -v`
Expected: PASS for the new test and `TestGetItemFromFloor_Happy`, `TestGetItemFromFloor_NotFound`.

- [ ] **Step 5: Write the player characterization test**

A player's `get doll` must still refuse with the same line once the player-only peek is gone. Create `internal/usercommands/get_household_test.go`:

```go
package usercommands

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A plain `get` of a household's bauble is refused, with the steal command
// named, and nothing is taken, now through the shared floor pickup
// (actions.ErrHouseholdBauble; owner ruling 2026-09-29).
func TestHouseholdBauble_GetRefusesThroughTheSharedPickup(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	// Midsummer noon, so get is not refused as blind (look_item_noun_test.go).
	cfg := configs.GetConfig()
	cfg.Timing.RoundsPerDay = 20
	configs.SetConfigForTest(t, cfg)
	gametime.ClearDateCacheForTest()
	t.Cleanup(gametime.ClearDateCacheForTest)
	util.SetRoundCountForTest(uint64(3430))
	t.Cleanup(util.ResetRoundCountForTest)

	restoreItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		items.BaubleItemId: {ItemId: items.BaubleItemId, Name: "Curious Trinket", NameSimple: "trinket",
			Type: items.Object, Subtype: items.Mundane, Weight: 0.2, Value: 1, NotSalable: true},
	})
	defer restoreItems()
	baubles.SetDirForTest(t.TempDir())
	defer items.SetBaubleResolver(nil)

	user, room := getTestUserAndRoom(t)
	user.Character.Stats.Strength.ValueAdj = 50
	origItems := user.Character.Items
	user.Character.Items = nil
	defer func() { user.Character.Items = origItems }()

	rec, err := baubles.Create(baubles.Record{Name: "Small Child's Doll", NameSimple: "doll", Tier: baubles.TierCheap,
		Value: 3, WeightLbs: 0.5, Description: "A rag doll with one button eye.", Status: baubles.StatusReady})
	require.NoError(t, err)
	theirs := items.New(items.BaubleItemId)
	theirs.Bauble = rec.Id
	theirs.LeaveBaubleAt("on the shelf", room.RoomId, time.Now())
	room.AddItem(theirs, false)
	defer room.RemoveItem(theirs, false)

	events.DrainQueuedMessagesForTest(user.UserId)
	handled, err := Get("doll", user, room, 0)
	require.True(t, handled)
	require.NoError(t, err)
	out := strings.Join(events.DrainQueuedMessagesForTest(user.UserId), "\n")
	assert.Contains(t, out, "belongs to this household. To take it anyway")
	assert.Empty(t, user.Character.Items, "nothing taken")
}
```

Run: `go test ./internal/usercommands/ -run TestHouseholdBauble_GetRefusesThroughTheSharedPickup -v`
Expected: PASS (it pins today's player behaviour, through the peek). If `belongs to this household` is missing, read the output first: a darkness refusal means the fixture is not lit, not a regression.

- [ ] **Step 6: Delete the player-only copy, and word the shared refusal**

In `internal/usercommands/get.go`, delete the peek block

```go
			// A bauble found in this household belongs to it. `get` never
			// commits a crime: it refuses and names the steal command, which
			// is the theft (actions/steal.go, stealHouseholdBauble).
			if peekFound && !getFromStash && peekItem.BaubleBelongsTo(room.RoomId) {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(
					`The <ansi fg="itemname">%s</ansi> belongs to this household. To take it anyway, <ansi fg="command">steal %s</ansi>.`,
					peekItem.DisplayName(), stealWord(peekItem)))
				return true, nil
			}
```

and, directly below it in the `if peekFound {` branch, replace

```go
					matchItem = result.Item
					found = true
					if result.Err != nil {
```
with
```go
					matchItem = result.Item
					found = true
					if errors.Is(result.Err, actions.ErrHouseholdBauble) {
						// A bauble found in this household belongs to it.
						// `get` never commits a crime: the shared pickup
						// refuses it for every taker, and this names the
						// steal command, which is the theft.
						user.SendText(messaging.CategorySystem, fmt.Sprintf(
							`The <ansi fg="itemname">%s</ansi> belongs to this household. To take it anyway, <ansi fg="command">steal %s</ansi>.`,
							matchItem.DisplayName(), stealWord(matchItem)))
						return true, nil
					}
					if result.Err != nil {
```
(That three-line anchor is unique: the stash branch below sets `found = true` before `matchItem`.) Add `"errors"` to the file's imports. The refusal still returns before the pickup's later lines, so a hidden player stays hidden, as before.

- [ ] **Step 7: Run both tests and the get packages, and prove the move**

Run: `go build ./... && go test ./internal/actions/ -run "TestGetItemFromFloor" -v && go test ./internal/usercommands/ -run "TestHouseholdBauble|TestGet" -v 2>&1 | grep -E "^(--- FAIL|--- PASS|FAIL|ok)" && go test ./internal/mobcommands/`
Expected: every named test PASS, `internal/mobcommands` `ok`.

Probe: temporarily delete the `if !stash && matchItem.BaubleBelongsTo(room.RoomId) {...}` gate from `GetItemFromFloor` and rerun both tests. Expected: FAIL for BOTH `TestGetItemFromFloor_RefusesAHouseholdsBaubleToEveryTaker` (the mob took it) and `TestHouseholdBauble_GetRefusesThroughTheSharedPickup` (the player took it: the player-only copy is gone). Restore, rerun, PASS.

- [ ] **Step 7b: The AI companion is refused up front, not silently**

A mob's `get` of a household's bauble now does nothing and says nothing. The AI companion issues `get` as a command (`modules/aicompanion/actions.go:360-369`, `m.issue(... "get "+target ...)`): the command would quietly fail, the trinket would still lie there, and she could keep choosing it. So `performAction` refuses it before issuing, with a reason she is told.

Append to `modules/aicompanion/harm_test.go`:

```go
// A household's bauble is not hers to take (slice H, 14a): the get is
// refused before any command is issued, with a reason she is told, so she
// does not keep trying a pickup that would quietly fail.
func TestHouseholdBaubleIsRefusedUpFront(t *testing.T) {
	owner, _, room, her := harmWorld(t, configs.PVPDisabled)
	b := items.Item{ItemId: items.BaubleItemId, Bauble: `b0000001`, BaubleHousehold: room.RoomId}
	room.Items = append(room.Items, b)
	m, c, _ := strangerModule()
	sc := &scene{RoomId: room.RoomId, byRef: map[string]*thing{}}
	sc.byRef[`t1`] = &thing{Ref: `t1`, Kind: `item`, Name: `Trinket`, Item: b, HasItem: true}

	out := m.performAction(c, her, owner, sc, ActionProposal{Verb: `get`, Ref: `t1`},
		[]stimulus{{Kind: `heard`, FromOwner: true}}, 0, 0)
	if out.Issued || out.Refused != `it belongs to the household here` {
		t.Fatalf("refused up front, with the reason: %+v", out)
	}
}
```
and add `"github.com/GoMudEngine/GoMud/internal/items"` to that file's imports.

Run: `go test ./modules/aicompanion/ -run TestHouseholdBaubleIsRefusedUpFront -v`
Expected: FAIL (the get is issued, or refused for another reason).

In `modules/aicompanion/actions.go` `performAction`, replace

```go
	case `get`:
		if t.Kind != `item` && t.Kind != `gold` {
			return actionOutcome{Refused: `cannot pick that up`}
		}
```
with
```go
	case `get`:
		if t.Kind != `item` && t.Kind != `gold` {
			return actionOutcome{Refused: `cannot pick that up`}
		}
		// actions.GetItemFromFloor refuses a household's bauble to every
		// taker (ErrHouseholdBauble); say so now, so she does not keep
		// issuing a get that quietly does nothing.
		if t.Kind == `item` && t.Item.BaubleBelongsTo(room.RoomId) {
			return actionOutcome{Refused: `it belongs to the household here`}
		}
```

Run: `go test ./modules/aicompanion/`
Expected: `ok`.

- [ ] **Step 8: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/actions/get.go internal/actions/get_household_test.go internal/usercommands/get.go internal/usercommands/get_household_test.go modules/aicompanion/actions.go modules/aicompanion/harm_test.go && git commit -F - <<'EOF'
fix(baubles): every taker leaves a household's bauble where it lies

The household-bauble refusal moves from the player's get into
actions.GetItemFromFloor (ErrHouseholdBauble), so a mob's, companion's or
scavenger's get obeys it too. The AI companion is told up front instead
of issuing a get that quietly fails. The player's line is unchanged; get
all is unchanged.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 14b: Nobody pickpockets a companion

Owner ruling 2026-09-29 (amendment, ruling 4, with the decision recorded
there: every companion, the thief's own included, for every thief).

**Files:**
- Modify: `internal/actions/steal.go` (`stealFromMob`)
- Create: `internal/actions/steal_companion_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/actions/steal_companion_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/companionai"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// Nobody may pickpocket a companion, the thief's own included: a charmed
// one (the predicate mobs.CheckPlayerHarm refuses first) or one bonded to
// the AI companion (owner ruling 2026-09-29). Nothing is rolled, taken or
// trained.
func TestSteal_RefusesAnyCompanion(t *testing.T) {
	actor := newStealPlayerActor(200, 8)
	cases := map[string]func(m *mobs.Mob){
		`someone's charmed companion`:       func(m *mobs.Mob) { m.Character.Charmed = characters.NewCharm(4242, 10, ``) },
		`the thief's own charmed companion`: func(m *mobs.Mob) { m.Character.Charmed = characters.NewCharm(actor.GetUserId(), 10, ``) },
		`an AI companion`: func(m *mobs.Mob) {
			companionai.SetBondedCheck(func(id int) bool { return id == testMobInstId })
		},
	}
	for name, setup := range cases {
		target := newStealTestMob(testMobInstId, 50, 1)
		mobs.SetInstanceForTest(testMobInstId, target)
		setup(target)
		delete(actor.char.Cooldowns, skills.Skullduggery.String("steal"))
		awards := len(actor.awards)

		result := Steal(actor, StealOptions{TargetMobInstanceId: testMobInstId})

		companionai.SetBondedCheck(nil)
		mobs.SetInstanceForTest(testMobInstId, nil)
		if result.Reason != "companion" || result.Succeeded || result.Pending || target.Character.Gold != 50 || len(actor.awards) != awards {
			t.Errorf("%v: refused, nothing taken or trained: %+v gold=%d", name, result, target.Character.Gold)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/actions/ -run TestSteal_RefusesAnyCompanion -v`
Expected: FAIL for all three cases with "refused, nothing taken or trained": today the roll is made, and in this package's tests the pickpocket resolves in line (`runPocketAttempt = resolvePocketInLine`, `bauble_testinit_test.go`), so the result is a success or `detected`, never `companion`, and the thief is trained either way.

Note, recorded rather than changed: `Steal` spends the steal cooldown (`TryCooldown`, `steal.go:136`) before `stealFromMob` looks at the target, so a refused companion costs the cooldown, exactly as the non-combatant and attack-immune refusals already do. The test deletes the cooldown each case for that reason. Moving every target refusal ahead of the cooldown is a separate behaviour change, left alone here.

- [ ] **Step 3: Refuse them**

In `internal/actions/steal.go` `stealFromMob`, replace

```go
	// Deliberately NOT mobs.CheckPlayerHarm: that policy also blocks charmed
	// companions, and stealing from a companion is currently allowed. Widening
	// it here would be a gameplay change, not a finding-3 fix. Keep the two
	// protections that do apply.
	if m.IsNonCombatant() || m.PlayerAttackImmune {
```
with
```go
	// Any companion is off-limits to theft, the thief's own included: a
	// charmed one (IsCharmed, the predicate mobs.CheckPlayerHarm refuses
	// first) or one bonded to the AI companion, which need not be charmed.
	// Its pocket is its owner's (owner ruling 2026-09-29). This holds for a
	// mob thief as well, as the two protections below do.
	if m.Character.IsCharmed() || companionai.IsBondedCompanion(m.InstanceId) {
		actor.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> is someone's companion. You can't steal from them.`,
			m.Character.Name))
		return StealResult{
			DefenderName: m.Character.Name,
			Reason:       "companion",
		}
	}

	// The rest of mobs.CheckPlayerHarm's policy.
	if block := mobs.CheckPlayerHarm(m); block.Blocked() {
```
and add `"github.com/GoMudEngine/GoMud/internal/companionai"` to the file's imports (`steal_pocket.go`, same package, already imports it, so there is no cycle). `CheckPlayerHarm` returns `HarmBlockedNonCombatant` or `HarmBlockedAttackImmune` here, exactly the two conditions the old line tested, since the companion case has already returned.

`internal/mobs/harm_authorization.go` needs no change: its doc already lists theft among the actions it authorises. `pocketBaubleAllowed` (`steal_pocket.go`) keeps its charmed and bonded checks: they are now defence in depth for a charmed mark, and its `EverCharmed` check still matters, since a former companion may be pickpocketed but its name must not reach the model. `TestNoBaubleFromACompanionsPocket` calls `startPocketAttempt` directly, below this refusal, so it is unaffected.

- [ ] **Step 4: Run to verify it passes, with every steal test**

Run: `go test ./internal/actions/ -run "TestSteal|TestPickpocket|TestPocket" -v 2>&1 | grep -E "^(--- FAIL|FAIL|ok)"`
Expected: only `ok`.

Probe: temporarily drop `|| companionai.IsBondedCompanion(m.InstanceId)` and rerun `-run TestSteal_RefusesAnyCompanion`. Expected: FAIL naming `an AI companion`. Restore, rerun, PASS.

Then the root text guards (review finding i):
```bash
cd /c/tmp/dogmud-baubles-h && go test . -count=1 -run "TestEveryTextSurfaceIsRegistered|TestNarrationSitesMatchViewpointAudit" -v 2>&1 | grep -E "^(--- FAIL|--- PASS|FAIL|ok)|steal.go"
```
Expected: both PASS. `TestEveryTextSurfaceIsRegistered` walks YAML keys only, so no Go line can change it; it runs here because the review asked, and must stay green. `TestNarrationSitesMatchViewpointAudit` treats an event as a candidate only when it has an actor line and exactly one of an actee or an observer line (`narrationCandidateEvent`, `messaging_surface_guard_test.go:1106-1124`); the refusal is an actor line alone, so it is not one. If the guard reports it all the same, register the key it prints, `actions/steal.go|<ansi fg="mobname">%s</ansi> is someone's companion. You can't steal from them.`, in `narrationViewpointRegistry` beside the other bauble rows as `{verdictCorrect, true, false, false, "a refused pickpocket of anyone's companion (slice H, owner ruling 2026-09-29): a private refusal to the thief, as the non-combatant refusal above it; nobody else is told."}` and add `messaging_surface_guard_test.go` to the commit.

- [ ] **Step 5: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/actions/steal.go internal/actions/steal_companion_test.go && git commit -F - <<'EOF'
fix(steal): nobody pickpockets a companion

stealFromMob refuses any charmed companion, the thief's own included, and
any AI companion, before the roll; the non-combatant and attack-immune
refusals now come from mobs.CheckPlayerHarm.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 14c: A failed pickpocket roll is caught however the pause ends

Owner ruling 2026-09-29 (amendment, ruling 1). The naming request still
starts at the roll. A successful roll whose thief left stays "chance
lost". A failed roll is caught wherever the thief is: beside the mark it
is the ordinary catch in the act; anywhere else, or offline, the mark
cries thief in its own room and the crime is recorded against the thief
(the faction block of `thiefCaught`, which goes by user id), but nobody is
attacked. A mark that is gone or dead catches nobody. The flush at
copyover and shutdown runs the same `resolve`, so it follows the same
rule.

Review finding b: the faction block finds its witnesses with
`crimes.WitnessesInRoom` in the room it is given, and every identifying
witness gets `knowledge.RecordMet(..., SourceWitnessed)` (`steal.go:638,
664`; `crimes.go:229-263`). Away from the theft, the mark's room now holds
bystanders who saw nothing, so an AWAY catch has one witness, the mark,
by its own sight of the room the theft happened in; `hadExternal` is
false; and nobody records meeting the thief (the mark felt a hand, it did
not meet anyone). `theftWitnesses` decides the witnesses for both modes.

The attempt keeps the theft room's ID (`roomId`, as today), not a
`*rooms.Room` across the pause: the reveal loads the room again
(`rooms.LoadRoom`). An online thief whose room cannot be loaded is treated
by `pocketThief` as offline (it returns `false`, `steal_pocket.go:104-115`,
unchanged), and `caught` treats a nil room as away: both land in the away
catch, which needs no thief room.

**Files:**
- Modify: `internal/actions/steal.go` (`thiefCaught` split: new `theftCrime`, new `theftWitnesses`)
- Modify: `internal/actions/steal_pocket.go` (`resolve`, new `caught`, new `pocketCrime`, the file comment)
- Modify: `internal/actions/pickpocket_test.go`
- Create: `internal/actions/steal_crime_test.go`

- [ ] **Step 1: Write the failing tests**

In `internal/actions/pickpocket_test.go`, replace the doc comment of `TestPickpocketAwardsAndCatchesAtTheReveal`

```go
// Training comes with the reveal, never before it (a skill-up line then
// would give the roll away), and never for a chance lost (walking off to
// dodge being caught trains nothing). A failed attempt is caught at the
// reveal, not before.
```
with
```go
// Training comes with the reveal, never before it (a skill-up line then
// would give the roll away). A failed attempt is caught at the reveal, not
// before, and walking off does not dodge it (owner ruling 2026-09-29).
```

and replace its second half

```go
	h2 := setupPocket(t, 9611, 7611)
	paused(100*time.Millisecond, 0)
	util.LockMud()
	startPocketAttempt(h2.thief, h2.mark, false)
	h2.thief.room = newSearchTestRoom(9698) // walks off
	util.UnlockMud()
	waitSettled(t)
	if len(h2.thief.awards) != 0 || said(h2.thief, "catches you in the act") != 0 || said(h2.thief, "lose your chance") != 1 {
		t.Fatalf("a chance lost trains nothing and is caught by nobody: %+v %q", h2.thief.awards, h2.thief.sent)
	}
}
```
with
```go
	h2 := setupPocket(t, 9611, 7611)
	seedPocketRooms(t, h2.room)
	crimes := stubPocketCrime(t)
	paused(100*time.Millisecond, 0)
	util.LockMud()
	startPocketAttempt(h2.thief, h2.mark, false)
	h2.thief.room = newSearchTestRoom(9698) // walks off
	util.UnlockMud()
	waitSettled(t)
	if len(h2.thief.awards) != 1 || h2.thief.awards[0].won || said(h2.thief, "felt your hand") != 1 ||
		said(h2.thief, "lose your chance") != 0 || len(*crimes) != 1 {
		t.Fatalf("walked off, still caught, trained on the loss: %+v %q crimes=%v", h2.thief.awards, h2.thief.sent, *crimes)
	}
}

// stubPocketCrime records the crimes a pickpocket caught away from the mark
// raises (pocketCrime), instead of reaching the faction books.
func stubPocketCrime(t *testing.T) *[]int {
	t.Helper()
	got := &[]int{}
	orig := pocketCrime
	pocketCrime = func(userId int, m *mobs.Mob, room *rooms.Room) { *got = append(*got, userId) }
	t.Cleanup(func() { pocketCrime = orig })
	return got
}

// seedPocketRooms makes rooms loadable, as real ones are: the reveal loads
// the theft room and the mark's room by ID (rooms.LoadRoom).
func seedPocketRooms(t *testing.T, rs ...*rooms.Room) {
	t.Helper()
	m := map[int]*rooms.Room{}
	for _, r := range rs {
		m[r.RoomId] = r
	}
	t.Cleanup(rooms.SeedRoomsForTest(m, nil))
}

// A failed roll is caught however the pause ends (owner ruling 2026-09-29).
// A thief who walked off or logged out, or whose mark moved, is caught away
// from the mark: the mark cries thief in its own room and the crime is
// recorded, and nobody is attacked. One still beside it is caught in the
// act. A mark that has gone catches nobody.
func TestPickpocketFailedRollIsCaughtHoweverThePauseEnds(t *testing.T) {
	cases := map[string]struct {
		before func(h *pocketHarness, p *pocketAttempt)
		caught bool
		crimes int    // raised away from the mark (pocketCrime)
		told   string // a line the thief is told exactly once
	}{
		`walked off`: {func(h *pocketHarness, p *pocketAttempt) { h.thief.room = newSearchTestRoom(9696) }, true, 1, "felt your hand"},
		`logged out`: {func(h *pocketHarness, p *pocketAttempt) { p.actor = nil }, true, 1, "You attempt to pick"},
		`mark moved`: {func(h *pocketHarness, p *pocketAttempt) { h.mark.Character.RoomId = 9695 }, true, 1, "felt your hand"},
		`still here`: {func(h *pocketHarness, p *pocketAttempt) {}, true, 0, "catches you in the act"},
		`mark gone`:  {func(h *pocketHarness, p *pocketAttempt) { mobs.SetInstanceForTest(h.mark.InstanceId, nil) }, false, 0, "lose your chance"},
	}
	for name, c := range cases {
		c := c
		h := setupPocket(t, 9617, 7617)
		seedPocketRooms(t, h.room, newSearchTestRoom(9695))
		crimes := stubPocketCrime(t)
		runPocketAttempt = func(p *pocketAttempt) StealResult {
			c.before(h, p)
			return resolvePocketInLine(p)
		}
		res := startPocketAttempt(h.thief, h.mark, false)
		if res.Detected != c.caught || len(*crimes) != c.crimes || said(h.thief, c.told) != 1 {
			t.Errorf("%v: detected %v (want %v), crimes %v (want %d), told %q", name, res.Detected, c.caught, *crimes, c.crimes, h.thief.sent)
		}
		if name == `logged out` && (len(h.thief.sent) != 1 || len(h.thief.awards) != 0) {
			t.Errorf("logged out: told and trained nothing after the attempt line: %q %+v", h.thief.sent, h.thief.awards)
		}
	}
}
```

`pickpocket_test.go` already imports `rooms` and `mobs`.

Run: `go test ./internal/actions/ -run "TestPickpocketAwardsAndCatchesAtTheReveal|TestPickpocketFailedRollIsCaughtHoweverThePauseEnds" -v`
Expected: build failure `undefined: pocketCrime`.

- [ ] **Step 1b: Write the failing witness test (review finding b)**

`theftCrime` writes to the faction, crime, bounty and knowledge stores, and knowledge writes under `DataFiles/knowledge` (`knowledge/persistence.go:20`), which a test binary would put in the real data tree. So the test drives the one function `theftCrime` takes its witnesses from, with a real faction definition and a real second same-faction mob. Create `internal/actions/steal_crime_test.go`:

```go
package actions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// A pickpocket caught AWAY from the theft (the mark felt a hand after the
// thief had gone; slice H review finding b) has one witness, the mark: a
// same-faction bystander in the mark's room saw nothing, so it neither
// identifies the thief nor counts as an external witness. In the act,
// that bystander does both, as ever.
func TestTheftWitnessesAwayAreTheMarkAlone(t *testing.T) {
	// Registered before t.Setenv, so it runs after the environment is put
	// back: the registry reloads from wherever it loaded before.
	t.Cleanup(func() { _ = factions.LoadAllDefinitions() })
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", dir)
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	body := "faction_id: thornwall_citizens\ndisplay_name: \"Thornwall Citizenry\"\ndescription: \"x\"\ndefault_rep: 0\nallies: []\nenemies: []\n"
	if err := os.WriteFile(filepath.Join(dir, "thornwall_citizens.yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	if err := factions.LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{"default": {BiomeId: "default"}}))

	room := &rooms.Room{RoomId: 9702, Lamp: rooms.LampPtr(90)}
	mark := newStealTestMob(9921, 0, 100)
	mark.Character.RoomId = room.RoomId
	mark.Groups = []string{"thornwall_citizens"}
	bystander := newStealTestMob(9922, 0, 100)
	bystander.Character.RoomId = room.RoomId
	bystander.Groups = []string{"thornwall_citizens"}
	for _, m := range []*mobs.Mob{mark, bystander} {
		mobs.SetInstanceForTest(m.InstanceId, m)
		room.AddMob(m.InstanceId)
	}
	t.Cleanup(func() {
		mobs.SetInstanceForTest(mark.InstanceId, nil)
		mobs.SetInstanceForTest(bystander.InstanceId, nil)
	})
	factionIds := factions.FactionsForMob(mark)
	if len(factionIds) != 1 {
		t.Fatalf("fixture: the mark is of one faction, got %v", factionIds)
	}

	inAct, externalInAct := theftWitnesses(factionIds, mark, room, false)
	if len(inAct.Identifying) != 2 || !externalInAct {
		t.Fatalf("in the act, the bystander identifies the thief too: %+v external %v", inAct, externalInAct)
	}
	away, externalAway := theftWitnesses(factionIds, mark, room, true)
	if len(away.Identifying) != 1 || away.Identifying[0] != mark.InstanceId || len(away.ShapesOnly) != 0 || externalAway {
		t.Fatalf("away, the mark alone: %+v external %v", away, externalAway)
	}
}
```

Run: `go test ./internal/actions/ -run TestTheftWitnessesAwayAreTheMarkAlone -v`
Expected: build failure `undefined: theftWitnesses`. (If, once Step 2 exists, the in-act half finds fewer than two witnesses, the fixture's room is not lit for these mobs: fix the fixture, not the rule. `Groups` is `mobs.Mob.Groups`, read by `factions.FactionsForMob`.)

- [ ] **Step 2: Split the mark's side out of `thiefCaught`**

In `internal/actions/steal.go`, replace

```go
	// Harmless if it fails (already revealed); combat_fire.go does the same.
	_ = actor.GetCharacter().Awareness.TransitionToRevealing(state.TransitionReason{
		Trigger: awareness.TriggerSkullduggeryFailed,
	})

	// Chunk 3.3: failed theft wakes a sleeping victim.
```
with
```go
	// Harmless if it fails (already revealed); combat_fire.go does the same.
	_ = actor.GetCharacter().Awareness.TransitionToRevealing(state.TransitionReason{
		Trigger: awareness.TriggerSkullduggeryFailed,
	})

	theftCrime(actor.GetUserId(), m, room, false)

	// A victim that cannot be fought (a non-combatant shopkeeper, a
	// player-attack-immune NPC) does not attack; it has already raised the
	// crime above. stealFromMob never reaches here with one (it refuses to
	// steal from them), so for `steal` this changes nothing.
	if !m.IsNonCombatant() && !m.PlayerAttackImmune {
		m.Command(fmt.Sprintf(`attack @%d`, actor.GetUserId()))
	}
}

// theftWitnesses is who witnessed userId's theft from m in room, and
// whether anyone but m identified the thief. In the act: every faction mob
// in room that saw it, m included, by its sight (crimes.WitnessesInRoom).
// Away (a pickpocket's failed roll revealed after the thief had gone; slice
// H review finding b): m alone, by its own sight of room, the room the
// theft happened in, which felt the hand; bystanders saw nothing, so
// hadExternal is false.
func theftWitnesses(factionIds []string, m *mobs.Mob, room *rooms.Room, away bool) (crimes.Witnesses, bool) {
	if !away {
		// All witnesses including the victim (excludeInstanceId=0), and the
		// external ones (excluding the victim) for HadExternalWitness, which
		// asks whether the theft was identified by someone other than the
		// victim, not merely noticed, so it reads Identifying.
		witnesses := crimes.WitnessesInRoom(factionIds, room, 0)
		external := crimes.WitnessesInRoom(factionIds, room, m.InstanceId)
		return witnesses, len(external.Identifying) > 0
	}
	var w crimes.Witnesses
	switch {
	case messaging.CanSeeClearly(&m.Character, room):
		w.Identifying = []int{m.InstanceId}
	case messaging.CanSeeShapes(&m.Character, room):
		w.ShapesOnly = []int{m.InstanceId}
	}
	return w, false
}

// theftCrime is the mark's side of a caught theft by userId in room (the
// room the theft happened in): a sleeping m wakes, and the theft is
// recorded as a crime against m's factions (reputation, bounty, witnesses'
// knowledge). Every part of it goes by user id, so it holds for a thief
// who has left or logged out. away is a pickpocket's failed roll revealed
// after the thief walked away (steal_pocket.go, pocketCrime): the mark is
// the only witness (theftWitnesses) and nobody records meeting the thief.
// thiefCaught runs it in the act.
func theftCrime(userId int, m *mobs.Mob, room *rooms.Room, away bool) {
	// Chunk 3.3: failed theft wakes a sleeping victim.
```

and replace the old tail of the function (now the tail of `theftCrime`)

```go
	// A victim that cannot be fought (a non-combatant shopkeeper, a
	// player-attack-immune NPC) does not attack; it has already raised the
	// crime above. stealFromMob never reaches here with one (it refuses to
	// steal from them), so for `steal` this changes nothing.
	if !m.IsNonCombatant() && !m.PlayerAttackImmune {
		m.Command(fmt.Sprintf(`attack @%d`, actor.GetUserId()))
	}
}

// stealObserverPass is the theft observer contest:
```
with
```go
}

// stealObserverPass is the theft observer contest:
```
(the second occurrence of that attack block, the one directly above `// stealObserverPass`; the first, which the previous replacement created, stays).

Inside `theftCrime`, replace the witness block

```go
		// All witnesses including the victim (excludeInstanceId=0).
		witnesses := crimes.WitnessesInRoom(factionIds, room, 0)
		perp := crimes.IdentifiedPerp(actor.GetUserId(), witnesses)
		// External witnesses (excluding victim) for HadExternalWitness.
		externalWitnesses := crimes.WitnessesInRoom(factionIds, room, m.InstanceId)
		// HadExternalWitness asks whether the theft was identified by
		// someone other than the victim, not merely noticed, so it reads
		// Identifying.
		hadExternal := len(externalWitnesses.Identifying) > 0
```
with
```go
		witnesses, hadExternal := theftWitnesses(factionIds, m, room, away)
		perp := crimes.IdentifiedPerp(userId, witnesses)
```
and the meeting

```go
					knowledge.RecordMet(int(w.MobId), subject, room.RoomId,
						knowledge.SourceWitnessed)
```
with
```go
					// Away, the mark felt a hand; it met nobody.
					if !away {
						knowledge.RecordMet(int(w.MobId), subject, room.RoomId,
							knowledge.SourceWitnessed)
					}
```
Then replace each remaining `actor.GetUserId()` in `theftCrime` with `userId`: three, in `factions.BumpRep`, `justice.MaybeDeclareBounty` and `knowledge.PlayerSubject`. `go build ./internal/actions/` names any one missed as `undefined: actor`. `steal.go` already imports `messaging`.

Run: `go test ./internal/actions/ -run TestTheftWitnessesAwayAreTheMarkAlone -v`
Expected: PASS. Probe: temporarily make `theftWitnesses` ignore `away` (delete the `if !away {` line and its closing `}`, so both modes use the room walk) and rerun: expected FAIL "away, the mark alone". Restore, rerun, PASS.

- [ ] **Step 3: The catch away from the mark**

In `internal/actions/steal_pocket.go`:

(a), (b): nothing. The attempt keeps only `roomId` across the pause, as today; the reveal loads rooms by ID.

(c) In `resolve`, replace

```go
	var room *rooms.Room
	if online {
		room = thief.GetRoom()
	}
	if !online || room == nil || room.RoomId != p.roomId || thief.GetCharacter().IsInCombat() ||
```
with
```go
	var room *rooms.Room
	if online {
		room = thief.GetRoom()
	}
	// A failed roll is caught however the pause ends (owner ruling
	// 2026-09-29): walking off, logging out, starting a fight or a
	// copyover's flush does not undo what the mark already felt. Only a
	// mark that has gone or died catches nobody.
	if !p.success && m != nil && !m.Character.IsDead() {
		return p.caught(thief, online, m)
	}
	if !online || room == nil || room.RoomId != p.roomId || thief.GetCharacter().IsInCombat() ||
```
replace the comment
```go
		// The chance is gone: nothing taken, nothing caught, nothing
		// trained. A bauble named for it stays in the mark's pocket, to be
		// found by whoever tries next.
```
with
```go
		// A successful roll's chance is gone (a failed one was caught
		// above): nothing taken, nothing trained. A bauble named for it
		// stays in the mark's pocket, to be found by whoever tries next.
```
and delete
```go
	if !p.success {
		return caughtByMob(thief, m, room)
	}
```
(every failed roll has returned by then: caught above, or, with the mark gone, lost).

(d) Add, directly below `resolve`:

```go
// pocketCrime is the mark's side of a catch away from it: theftCrime in
// the room the theft happened in, in away mode (the mark the only witness,
// no meeting recorded). A variable so tests can see it raised without the
// faction books.
var pocketCrime = func(userId int, m *mobs.Mob, theftRoom *rooms.Room) {
	theftCrime(userId, m, theftRoom, true)
}

// caught is a failed roll's reveal, wherever the thief is by now (owner
// ruling 2026-09-29). Beside the mark it is the ordinary catch in the act
// (caughtByMob: the room sees it, the crime, the attack). Anywhere else,
// or offline, the mark felt the hand all the same: it cries thief in its
// own room, and the theft is recorded against the thief in the room it
// happened in (pocketCrime), but it attacks nobody, since the thief is not
// there. An online thief is told and trained on the loss. An online thief
// with no room (GetRoom nil) is away.
func (p *pocketAttempt) caught(thief Actor, online bool, m *mobs.Mob) StealResult {
	if online {
		thief.AwardResolved(false, thief.GetCharacter().CandidateFor(string(skills.Skullduggery)))
		if here := thief.GetRoom(); here != nil && here.RoomId == m.Character.RoomId {
			return caughtByMob(thief, m, here)
		}
		thief.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> felt your hand in their pocket. A cry of "Thief!" follows you.`, p.mobName))
	}
	markRoom := rooms.LoadRoom(m.Character.RoomId)
	if markRoom != nil {
		markRoom.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> pats a pocket and cries, "Thief!"`, m.Character.Name), p.userId)
	}
	theftRoom := rooms.LoadRoom(p.roomId)
	if theftRoom == nil {
		theftRoom = markRoom // the theft's room is gone: the crime is recorded where the mark is
	}
	if theftRoom != nil {
		pocketCrime(p.userId, m, theftRoom)
	}
	return StealResult{Detected: true, DefenderName: p.mobName, Reason: `detected`}
}
```

(e) In the file's top comment, replace

```go
// A thief who has left the room, logged off or started fighting by then, or
// a mark who has gone, loses the chance: nothing is taken, nothing is
// caught.
```
with
```go
// A failed roll is caught however the pause ends (owner ruling
// 2026-09-29): beside the mark, in the act; away from it or offline, the
// mark cries thief in its room and the crime is recorded, with no attack
// (caught). A successful roll whose thief has left the room, logged off or
// started fighting by then, or whose mark has gone, loses the chance:
// nothing is taken.
```

- [ ] **Step 4: Run to verify they pass, with every pickpocket and steal test**

Run: `go build ./... && go test ./internal/actions/ -run "TestPickpocket|TestPocket|TestFlush|TestOnePickpocket|TestNoBauble|TestSteal|TestStolen|TestHousehold" -v 2>&1 | grep -E "^(--- FAIL|FAIL|ok)"`
Expected: only `ok`.

Probe: temporarily change the new guard in `resolve` to `if !p.success && m != nil && !m.Character.IsDead() && online && room != nil && room.RoomId == p.roomId {` (caught only when still present, today's rule) and rerun `-run TestPickpocketFailedRollIsCaughtHoweverThePauseEnds`. Expected: FAIL naming `walked off` and `logged out` (both fall to "chance lost"). Restore, rerun, PASS.

- [ ] **Step 5: The root guards**

Run: `go test . -count=1 -run "TestEveryTextSurfaceIsRegistered|TestNarrationSitesMatchViewpointAudit|TestEveryRollSiteAppliesTheSightPenalty|TestEveryCreatureLookupDeclaresItsViewer" -v 2>&1 | grep -E "^(--- FAIL|--- PASS|FAIL|ok)|steal_pocket"`

Expected: `TestEveryTextSurfaceIsRegistered` PASS (it walks YAML keys only; run because the review asked). `TestNarrationSitesMatchViewpointAudit` should report `caught`'s event, an actor line ("felt your hand") and an observer line ("pats a pocket") with no actee, which is exactly a candidate (`narrationCandidateEvent`: actor plus exactly one of actee or observer). The guard keys an event by its first literal, stripped of quoting and cut to 80 runes (`narrationSiteKey`, `narrationLiteralMaxRunes`). Register every key it prints, using these rows in `narrationViewpointRegistry`, beside the other bauble rows (keys computed from the literals above; use the printed key if it differs):

```go
	`actions/steal_pocket.go|<ansi fg="mobname">%s</ansi> felt your hand in their pocket. A cry of "Thief!" f`: {verdictCorrect, true, false, true, "a failed pickpocket revealed after the thief walked away (owner ruling 2026-09-29): the away sibling of the audited catch in the act; the actee is a mob, and the mark's own room is told."},
	`actions/steal_pocket.go|<ansi fg="mobname">%s</ansi> pats a pocket and cries, "Thief!"`: {verdictCorrect, false, false, true, "the mark's room hears it cry thief after a failed pickpocket, with the thief gone or offline (owner ruling 2026-09-29); the thief's line, when online, is the row above."},
```
Add only the rows the guard actually reports (an unused row is stale and fails the same test). The 14b refusal line is actor-only and never a candidate (Task 14b Step 4).

- [ ] **Step 6: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/actions/steal.go internal/actions/steal_pocket.go internal/actions/pickpocket_test.go internal/actions/steal_crime_test.go && git commit -F - <<'EOF'
fix(steal): a failed pickpocket roll is caught however the pause ends

Walking off, logging out, starting a fight or a copyover's flush no longer
turns a failed roll into a lost chance. Beside the mark it is the catch in
the act; away from it, the mark cries thief in its room and the crime is
recorded against the thief in the room of the theft (theftCrime, split
from thiefCaught), with the mark the only witness, no meeting recorded
and no attack. A successful roll's lost chance is unchanged.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```
(Add `messaging_surface_guard_test.go` to `git add` if Step 5 registered the line.)

---

### Task 14d: PR H3 docs

**Files:**
- Modify: `internal/baubles/context.md`, `internal/actions/context.md`, `internal/usercommands/context.md`, `internal/mobcommands/context.md`, `modules/aicompanion/context.md`
- Modify: `_datafiles/world/dogmud/templates/help/steal.template`

No new non-code file (every file this slice creates is Go), so no `docs/README.md` row.

- [ ] **Step 0: Verify every symbol the docs will name**

```bash
cd /c/tmp/dogmud-baubles-h && grep -rn "SightPenalty float64\|var ErrHouseholdBauble\|func theftCrime\|func theftWitnesses\|var pocketCrime\|func (p \*pocketAttempt) caught\|func clampUnit" internal modules
```
Expected: at least one hit for each pattern. Fix the doc text, never the grep.

- [ ] **Step 1: `internal/baubles/context.md`**

- Change the `FindOpts` line to `type FindOpts struct{ Place Place; UserId int; SkillFactor float64; SightPenalty float64; Feature string; Household bool; Randn func(n int) int; Now time.Time }`.
- Add to the find section: `` `FindOpts.SightPenalty` (0 is none, clamped to 0..1 by `clampUnit`) multiplies the chance by `1 - SightPenalty` after the nothing-here check and before the window roll, so a search in the dark spends its roll as one in the light does; callers pass `1 - messaging.SightMult`. ``

- [ ] **Step 1b: `modules/aicompanion/context.md`**

Beside the actions description: `` A `get` of a household's bauble is refused before any command is issued ("it belongs to the household here"): `actions.GetItemFromFloor` would refuse it anyway, silently, and she would keep trying. ``

- [ ] **Step 1c: `_datafiles/world/dogmud/templates/help/steal.template` (review finding h)**

Replace
```
Being hidden improves your chances. On failure, your cover
is blown and the target may attack. Cannot be used on
other players.
```
with
```
Being hidden improves your chances. On failure, your cover
is blown and the target may attack. Cannot be used on
other players, or on anyone's companion, your own included.
```
and replace
```
(<ansi fg="yellow">Dexterity</ansi>) make the moment shorter. Walk away or get into a
fight before it is done, and you lose your chance. Now and then
```
with
```
(<ansi fg="yellow">Dexterity</ansi>) make the moment shorter. Walk away or get into a
fight before it is done, and you lose your chance, but only if
your hands were quick enough: if the mark felt them, walking away
does not help, and they cry thief all the same. Now and then
```
Keep every line within 80 columns (dogmud-player-copy); none of the new lines is longer than the ones around it.

- [ ] **Step 2: `internal/actions/context.md`**

In the bauble search paragraph (the one naming `baubles.RollFind`), add: `` Both bauble rolls pass `SightPenalty: 1 - messaging.SightMult(char, room)`; the root sight guard watches `baubles.RollFind` and `actions.searchBaubleRoll`. ``

In the stolen-baubles paragraph (the one beginning `**Stolen baubles after the theft (`stolen_bauble.go`), add: `` The owner's "points at" line and the household "caught trying to pocket" line (`steal.go`) go through `SendTextVisualHidingNames` with the player's name. ``

(Throughout this task, text quoted from a `context.md` is shown unwrapped; the file wraps it over several lines, so match it by its words and rewrap the replacement to the file's width.)

In the Mob pickpocket path (the numbered `**Three paths:**` list, item 1):
- Change the first bullet to: `` - Refused for any companion (charmed, the thief's own included, or bonded to the AI companion: "X is someone's companion. You can't steal from them.", reason `companion`), then for non-combatant or player-attack-immune mobs (`mobs.CheckPlayerHarm`), and below skullduggery rank 2. ``
- Replace the sentences from `A thief who has left the room, gone offline, started fighting or come under attack by then,` through `walking off only ever forfeits.` with: `` A FAILED roll is caught however the pause ends (owner ruling 2026-09-29; `caught`): beside the mark, `caughtByMob` as below; away from it or offline, the mark cries thief in its own room and `theftCrime` records the crime against the thief (through the `pocketCrime` seam), with no attack; an online thief is told and trained on the loss. A mark that is gone or dead catches nobody. A SUCCESSFUL roll whose thief has left the room, gone offline, started fighting or come under attack by then, or whose mark has moved, died or gone, loses the chance ("You lose your chance at X's pocket."): nothing taken, nothing trained. ``
- Change the Failure bullet to: `` - Failure (`caughtByMob`): "X catches you in the act!", the room sees it, then `thiefCaught` (revealed, then `theftCrime` in the act: a sleeper wakes, the crime with every faction witness in the room; then the attack). Away, `theftCrime` runs in the theft's room with the mark its only witness (`theftWitnesses`), no external witness and no meeting recorded. ``

Add to the Get section (or, if none, beside `GetItemFromFloor` in the file table): `` `GetItemFromFloor` refuses a household's bauble (`BaubleBelongsTo`) with `ErrHouseholdBauble`, the item found and nothing moved, for every taker: a player's `get`, a mob's, a companion's, a scavenger's (owner ruling 2026-09-29). Its gates sit in one early-return block. ``

- [ ] **Step 3: `internal/usercommands/context.md`**

Run:
```bash
cd /c/tmp/dogmud-baubles-h && grep -n "SendTextVisual\|observer\|room line" internal/usercommands/context.md
```
Only if that prints a line describing how `look.go` tells the room (at `e711ee9de` it printed only the crafting observer paragraph, lines 466-479, which is not about `look`; the file describes what `look` shows the looker, not the room lines), add beside it: `` Every observer line in `look.go` goes through `SendTextVisualHidingNames` with the looker's name. ``

Always, in the **Household baubles** section (`get.go`), replace `` An explicit `get <name>` refuses with "The X belongs to this household. To take it anyway, steal <word>." before the ordinary pickup (which would end the player's hiding); `` with `` An explicit `get <name>` is refused by the shared floor pickup (`actions.GetItemFromFloor` returns `actions.ErrHouseholdBauble` for every taker, mobs included), and `get` words it: "The X belongs to this household. To take it anyway, steal <word>.", before anything that would end the player's hiding; ``.

- [ ] **Step 4: `internal/mobcommands/context.md`**

Where `get` is described (`grep -n "get" internal/mobcommands/context.md` finds the command list), add: `` A mob's `get` never takes a household's bauble: `actions.GetItemFromFloor` refuses it (`ErrHouseholdBauble`) and the mob says nothing. ``

- [ ] **Step 5: Run the context.md audit**

Run: `cd /c/tmp/dogmud-baubles-h && python tools/context_md_audit.py 2>&1 | grep -E "baubles|actions|usercommands|mobcommands|aicompanion" ; echo "exit=$?"`
Expected: no phantom symbols in the touched packages (grep exit 1 means none listed). The tool only reads files; it writes nothing.

- [ ] **Step 6: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/baubles/context.md internal/actions/context.md internal/usercommands/context.md internal/mobcommands/context.md modules/aicompanion/context.md _datafiles/world/dogmud/templates/help/steal.template && git commit -F - <<'EOF'
docs(baubles): PR H3 sight ramp, observer lines and the steal work

The steal help says a failed pocket-pick is caught even if you walk away,
and that nobody's companion can be stolen from.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 14e: PR H3 gate

**Files:** none new. Every diff runs from `$BASE` (Task H3-0 Step 1).

- [ ] **Step 1: gofmt on the committed blobs**

```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && for f in $(git diff --name-only $BASE..HEAD -- '*.go'); do out=$(git show HEAD:"$f" | gofmt -l); [ -n "$out" ] && echo "UNFORMATTED: $f"; done; echo done
```
Expected: only `done`.

- [ ] **Step 2: Size**

```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && git diff --shortstat $BASE..HEAD
```
Expected: under 300 files and under 20,000 lines.

- [ ] **Step 3: Build, vet, tests**

```bash
cd /c/tmp/dogmud-baubles-h && go build ./... && go vet ./internal/baubles/ ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ ./internal/rooms/ ./modules/aicompanion/ . && go test ./internal/baubles/ ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ ./internal/rooms/ ./internal/messaging/ ./modules/aicompanion/ . 2>&1 | tail -12
```
Expected: no vet output, every package `ok`. `internal/rooms`' two zone lifecycle tests fail on Windows only under `DOGMUD_BOOT_SMOKE=1` and skip otherwise; this run does not set it.

- [ ] **Step 4: The guards this PR leans on, by name**

```bash
cd /c/tmp/dogmud-baubles-h && go test . -count=1 -run "TestEveryRollSiteAppliesTheSightPenalty|TestSightPenaltyGuard|TestEveryTextSurfaceIsRegistered|TestNarrationSitesMatchViewpointAudit|TestOpposedContestsAreFloored|TestEveryCreatureLookupDeclaresItsViewer|TestFinderViewReachesOnlyItsReader" -v 2>&1 | grep -E "^(--- FAIL|--- PASS|FAIL|ok)"
```
Expected: every named test `--- PASS` (the sight guard, its probes and the bauble seam probe; the text surface registry and the viewpoint audit; the contest floor guard; the lookup guard; H2's finder-view guard, since Task 14 edits `look.go` and `steal.go`). A name that does not appear has changed on master: find it with `grep -n "^func Test"`.

- [ ] **Step 5: Boot smoke in the test process (no port)**

```bash
cd /c/tmp/dogmud-baubles-h && DOGMUD_BOOT_SMOKE=1 go test . -count=1 -run TestSmoke_ServerBootsCleanWithRealData -timeout 600s 2>&1 | tail -5
```
Expected: `ok`.

- [ ] **Step 6: A real boot on private ports, stopped by its own PID**

Check the fixed boot path is free:
```bash
git -C /c/tmp/dogmud-baubles-h worktree list | grep -F "dogmud-boot-check"; ls /c/tmp/dogmud-boot-check 2>/dev/null | head -3
```
Expected: no output (each exits 1); if the directory exists, STOP and ask.

Build (Bash):
```bash
git -C /c/tmp/dogmud-baubles-h worktree add --detach /c/tmp/dogmud-boot-check HEAD && cd /c/tmp/dogmud-boot-check && go build -o boot-check.exe . && cat > /c/tmp/dogmud-boot-check.overrides.yaml <<'EOF'
Network:
  TelnetPort: [33533]
  LocalPort: 9899
  HttpPort: 8391
  HttpsPort: 0
  AIPort: 0
EOF
echo built
```

Run (PowerShell):
```powershell
$env:CONFIG_PATH = 'C:\tmp\dogmud-boot-check.overrides.yaml'
$p = Start-Process -FilePath 'C:\tmp\dogmud-boot-check\boot-check.exe' -WorkingDirectory 'C:\tmp\dogmud-boot-check' -RedirectStandardOutput 'C:\tmp\dogmud-boot-check.log' -RedirectStandardError 'C:\tmp\dogmud-boot-check.err' -WindowStyle Hidden -PassThru
$deadline = (Get-Date).AddSeconds(180)
while ((Get-Date) -lt $deadline -and -not $p.HasExited -and -not (Select-String -Path 'C:\tmp\dogmud-boot-check.log','C:\tmp\dogmud-boot-check.err' -Pattern 'Server Ready' -Quiet)) { Start-Sleep -Seconds 2 }
"pid=$($p.Id) exited=$($p.HasExited)"
Select-String -Path 'C:\tmp\dogmud-boot-check.log','C:\tmp\dogmud-boot-check.err' -Pattern 'Server Ready|status=listening|^panic:|goroutine [0-9]+ \[running\]|runtime error|bind:' | Select-Object -First 20
if (-not $p.HasExited) { Stop-Process -Id $p.Id -Force -Confirm:$false }
Remove-Item Env:\CONFIG_PATH
```
Expected: `exited=False`, `Server Ready`, `Telnet status=listening port=33533` (and no 33333 or 44444), no `panic:`, `goroutine ... [running]`, `runtime error` or `bind:` line. Never grep the bare word `panic`.

Clean up (PowerShell), then `git -C /c/tmp/dogmud-baubles-h worktree prune`:
```powershell
Remove-Item -Recurse -Force 'C:\tmp\dogmud-boot-check'; Remove-Item -Force 'C:\tmp\dogmud-boot-check.overrides.yaml','C:\tmp\dogmud-boot-check.log','C:\tmp\dogmud-boot-check.err'
```

- [ ] **Step 7: Race, in the Linux test container**

```bash
cd /c/tmp/dogmud-baubles-h && docker compose -f compose.test.yml run --build --rm test go test -race -count=1 ./internal/actions/ ./internal/baubles/ ./modules/aicompanion/ 2>&1 | tail -8
```
Expected: `ok` for all three, no `WARNING: DATA RACE` (the pickpocket pause runs on its own goroutine; `TestPickpocketAwardsAndCatchesAtTheReveal` exercises it).

- [ ] **Step 8: Lint, go.mod, docs, tree**

```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && golangci-lint run --new-from-merge-base=$BASE
```
Expected: `0 issues`.
```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && git diff --exit-code $BASE..HEAD -- go.mod go.sum; echo "exit=$?"; git diff --name-only $BASE..HEAD | sed -n 's#^\(internal/[^/]*\|modules/[^/]*\)/.*#\1#p' | sort -u; git diff --name-only $BASE..HEAD -- '*context.md' _datafiles/world/dogmud/templates/; git status --short
```
Expected: `exit=0`; every package in the first list has its `context.md` in the second, except `internal/rooms` (a test-only change); `steal.template` listed; a clean tree.

- [ ] **Step 9: Hand over**

No push from here. Report the branch, `$BASE`, `git log --oneline $BASE..HEAD` and each gate result. When the owner says to ship: `git push -u origin fix/baubles-hardening-h3`, `gh pr create --repo pruuk/DOGMud --base master --head fix/baubles-hardening-h3 --title "Slice H3: sight ramp, hidden observer names, household guard, companion theft, pickpocket walk-out"`, and read back that the URL says `pruuk/DOGMud`. No deploy: the owner runs deploys.

---

## Self-review

**Spec coverage.** S1: Task 1 (`ConfigSecret`, `Validate`, key guard through `DisplayConfigData`). S2: Task 2 (exact hosts), Task 3 (`DecodeChat` scrub, both key forms), Task 4 (six baubles rows in `hardLocked`, `ModerateOutput` and `ModerationModel` included per ruling 13). S3 as amended (owner ruling 2026-09-29): moderated where possible and refused on a flag or failed check, finder-only where not (Task 10 `moderationPossible`, `moderate`; Task 7 engine rule; Task 9a record, catalog view and item accessors; Task 9b single-reader sites and root guard), every check on the moderation breaker alone (Task 10 `moderationBreaker`, `RecordConsumer`), player-key text out of every model prompt (Task 9c), admin views (Task 9d), allowlist (Task 7 engine; Task 10 `refusedByAllowlist` in `name`, falling back to the server route without feeding the player's breaker, ruling 15), typography folded before the allowlist (Task 5 `typographyFold`, pinned in Tasks 5, 7 and 10), the allowed characters in the prompt (Task 10, `PromptVersion` 5), `NameSimple` moderated and allowlisted (Tasks 7 and 10), material moderated (Task 10), value (Task 8), tokens (Task 11), slots (Task 11), recent names and regen (Task 9), authored names (Task 6), log quoting (Task 5 `quoteShort`, used by Tasks 6 and 7). S4: Task 5 (NFKC, Zs, Cf/Co/Cs/Mn, U+2028/2029, Hangul fillers, links, runes, code-point table, OSC, NBSP; `linkRE`'s accepted limits documented) and the homoglyph pin (Task 7). U2: Tasks 12 and 13 (field, order, log, callers, guard entry, exemption, both probes). U4: Task 14 (the six spec lines, their seven `look.go` siblings, and the two PR bauble theft lines; the characterization test pins behaviour and the grep checks the conversion). Pickpocket walk-out: Task 14c (failed roll caught however the pause ends; success unchanged; mark gone catches nobody; flush follows `resolve`; away witnesses the mark alone). Household guard: Task 14a (in `GetItemFromFloor`, player copy deleted, `get all` unchanged, the AI companion refused up front, both probes). Companion theft: Task 14b (every companion, the thief's own included, charmed or AI-bonded; the stale comment replaced). Point 8: recorded as skipped (amendment, ruling 5). Docs: Tasks 5a, 11a, 14d, one share per PR. Gates: Tasks 5b, 11b, 14e (diffs from that PR's `$BASE`, size under 20k lines and 300 files, lint from `$BASE`, the named root guards and the key guard, the boot smoke, a real boot at `C:\tmp\dogmud-boot-check` on private ports stopped by PID, `-race` in the Linux test container). Delivery: three PRs in order (Task 0, H2-0, H3-0 each branch from `origin/master`, record the base and grep that PR's anchors). Review fixes a to i and the LOW list: see the table under "Tasks this amendment adds or changes".

**Placeholder scan.** Tasks 1 and 4 use slice M's landed names (`config_locks.go`, `hardLocked`, `isHardLocked`, `IsLocked`, `DisplayConfigData`), which Task 0 Step 2 confirms. Every quoted anchor was grepped at `3bd6ccaa3`; Task 0 Step 4 re-checks all of them for H1, and H2-0 Step 2 and H3-0 Step 2 re-check that PR's own after the earlier PRs merge (H1's own anchors are gone by then, by design). The conditional steps name their exact entries: Task 14b Step 4 and Task 14c Step 5 (register only the narration rows the guard reports).

**Type consistency.** `CheckPlayerKeyText(r Reply) error`, `AuthoredName(name string) bool`, `moderationPossible(cfg Config, s apiframework.ServerSettings, now time.Time) bool`, `moderationBreaker` (a `string` const), `(r Record) KeptToFinder() bool`, `BaubleView.PlayerText bool`, `(i *Item) ModelName() string`, `(i *Item) ModelDescription() string`, `theftWitnesses(factionIds []string, m *mobs.Mob, room *rooms.Room, away bool) (crimes.Witnesses, bool)`, `finderViewSite{calls int; why string}`, `(m *BaublesModule) moderate(cfg Config, reply baubles.Reply, playerKey bool) (moderated bool, finderOnly bool, err error)`, `refusedByAllowlist(content string) bool`, `typographyFold *strings.Replacer`, `takeServerSlot() (release func(), ok bool)`, `takeFinderSlot(userId int) (release func(), ok bool)`, `FindOpts.SightPenalty float64`, `quoteShort(s string) string`, `errSlotsBusy`, `chatBody(content string, tokens int) string`, `GenResult.FinderOnly bool` (there is no `Record.FinderOnly`), `(r Record) MaterialFor(viewerUserId int) string`, `genericDescriptionFor(id string) string`, `BaubleView.FinderUserId int`, `BaubleView.Finder *BaubleView`, `baubleSpecFor(base ItemSpec, id string, viewerUserId int) ItemSpec`, `(i *Item) GetSpecFor/DisplayNameFor/NameFor/LongDescriptionFor(viewerUserId int)`, `(i *Item) baubleFinderNames() []string`, `displayNameFrom(spec ItemSpec) string`, `longDescriptionFrom(iSpec ItemSpec) string`, `apiframework.RecordConsumer(consumer string, err error, now time.Time)`, `actions.ErrHouseholdBauble`, `theftCrime(userId int, m *mobs.Mob, room *rooms.Room, away bool)`, `pocketCrime func(userId int, m *mobs.Mob, theftRoom *rooms.Room)`, `(p *pocketAttempt) caught(thief Actor, online bool, m *mobs.Mob) StealResult`, `scanFinderView(fset *token.FileSet, files []finderViewFile) (map[string]int, []string)`, `buildBandolierContainer(c *characters.Character, viewerUserId int)`, `buildComponentBagContainer(c *characters.Character, viewerUserId int)`, `yesNo(v bool) string` are used with those signatures everywhere. Gone, and no task defines or calls them: the original plan's `playerRouteOpen` and `errUnmoderated`, the first amendment's `Record.FinderOnly` field and `pocketAttempt.room`.
