# Slice H: Baubles Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land sections S1, S2, S3, S4, U2 and U4 of the approved spec
`docs/superpowers/specs/2026-09-28-baubles-hardening-and-corpus-design.md`
on master, as its own PR, after PR #175 and slice M have both merged
(owner ruling 11): the server key is a secret, it only goes to OpenAI,
text named on a player's key is moderated or refused, every reply is
cleaned of invisible Unicode and links, the bauble search chance pays the
sight ramp, and the look, search and stolen-bauble observer lines hide
names by sight.

**Architecture:** Engine-side rules (`internal/baubles`, `internal/items`,
`internal/apiframework`) carry every invariant, so a future generator or
the corpus (slice C) cannot skip them; the module (`modules/baubles`) adds
one route pre-check function, a per-finder slot and the moderation policy.
S5 (per-user allowances), slice M (config locks and redaction) and slice C
(corpus) are out of scope. Slice H branches FRESH from master once #175
(after FinalTwist's fix round) and slice M have merged; our commits never
rebase or push onto FinalTwist's branch.

**Tech Stack:** Go 1.25, `golang.org/x/text/unicode/norm` (already a
direct requirement at v0.36.0), the repo-root AST guards.

**Worktree:** `C:\tmp\dogmud-baubles-h`, branch `fix/baubles-hardening-h`,
created from master by Task 0. All paths below are relative to it. Run git
from Bash, Windows process work from PowerShell. Every diff and gate
compares against `$BASE`, the master commit Task 0 records in
`/c/tmp/dogmud-baubles-h.base` (outside the worktree), never
`master...HEAD`: master may move while the slice is in flight.

---

## Facts verified against source

Code read at `e711ee9de` (PR #175's head; the planning worktree
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
| `_datafiles/config.yaml` | Modify | comments only |
| docs (`context.md` x7, `docs/aicompanion/settings.md`, `docs/baubles/implementation-plan.md`) | Modify | Task 15 |

No `docs/README.md` row: this plan reaches master through the separate
docs-only PR (ruling 11), which carries its row, and the slice creates no
new non-code file (`docs/README.md` indexes everything that is not code).

Every commit ends with:

```
Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
```

Use a heredoc for every message (bash command-substitutes backticks inside `-m`). Stage named paths only; never `git add -A` or `git add .`.

---

### Task 0: Preconditions

**Files:** none in the repo. Creates the worktree and `/c/tmp/dogmud-baubles-h.base`.

Delivery (owner ruling 11): PR #175 merges to master first, after
FinalTwist's fix round, and slice M merges as its own PR. Slice H then
branches fresh from master. Never rebase onto, commit to or push to
FinalTwist's branch.

- [ ] **Step 1: Confirm #175 and slice M are on master**

Run (Bash, from the main checkout):
```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud" && gh pr view 175 --repo pruuk/DOGMud --json state --jq .state && git fetch origin && git fetch origin master:master; git log --oneline -1 master && git log --oneline -1 origin/master
```
Expected: `MERGED`, and `master` at the same commit as `origin/master`. `git fetch origin master:master` refuses when some worktree has `master` checked out; then fast-forward it there (`git -C <that worktree> merge --ff-only origin/master`) and rerun. If the state is not `MERGED`, STOP and report: this slice waits for #175.

Then:
```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud" && git grep -n "hardLocked" master -- internal/configs | head -5
```
```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud" && git grep -n "DisplayConfigData" master -- internal/configs | head -5
```
Expected: both print at least one line (run them separately: an empty `git grep` exits 1). If either is empty, STOP: slice M has not merged. Report and wait.

- [ ] **Step 2: Create the worktree from master and record the base**

Run:
```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud" && git worktree add -b fix/baubles-hardening-h C:/tmp/dogmud-baubles-h master && cd /c/tmp/dogmud-baubles-h && BASE=$(git rev-parse HEAD) && echo "$BASE" > /c/tmp/dogmud-baubles-h.base && echo "BASE=$BASE"
```
Expected: the worktree is created and `BASE=<sha>` prints. Every later step that needs it runs `BASE=$(cat /c/tmp/dogmud-baubles-h.base)` first (shell state does not persist between calls).

- [ ] **Step 3: Record the shape of `hardLocked` and `DisplayConfigData`**

Run:
```bash
cd /c/tmp/dogmud-baubles-h && git grep -n -A40 "hardLocked = " -- internal/configs | head -50 && git grep -n "DisplayConfigData(" -- internal/configs
```
Confirm it matches what slice M's plan declares (see the facts table): `hardLocked` a `[]string` in `internal/configs/config_locks.go`, `isHardLocked` and `IsLocked` beside it, the four `APIFramework.*` paths and the `Modules.aicompanion.*` rows present (slice M also adds the companion's `ModerateOutput` and `ModerationModel`, ruling 13), no `Modules.baubles.*`, and `func (c Config) DisplayConfigData(excludeStrings ...string) map[string]any`. Tasks 1 and 4 are written against those names; if the landed code differs, adapt those two tasks to the landed names before starting them.

- [ ] **Step 4: Every quoted anchor this plan edits exists in the merged code**

FinalTwist's fix round rewrites parts of `modules/baubles/generate.go` (`name`, `viaServer`, `viaPlayer`), the `FindOpts` blocks in `search_bauble.go` and `search_feature.go`, `steal_pocket.go` and `sight_penalty_guard_test.go`. Run:
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
internal/actions/stolen_bauble.go|points at <ansi fg="username">%s</ansi>. "That's mine! Thief!"`,
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
EOF
missing=0; while IFS= read -r line; do f=${line%%|*}; a=${line#*|}; grep -qF -- "$a" "$f" || { echo "MISSING in $f: $a"; missing=1; }; done < "$anchors"; rm -f "$anchors"; echo "missing=$missing"
```
Expected: `missing=0` (checked against `e711ee9de` on 2026-09-28: every anchor but the `config_locks.go` one, which slice M creates, was present, so the loop can report a hit and a miss). For every `MISSING` line, STOP before the task that edits it: read the merged code, rewrite that task's quoted old text (and anything depending on it, such as Task 11's replacement of Task 10's block) against it, and report the change. Do not guess at a merged shape.

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
cd /c/tmp/dogmud-baubles-h && go build ./... && go test ./internal/baubles/ ./internal/items/ ./internal/apiframework/ ./internal/configs/ ./modules/baubles/ ./internal/actions/ ./internal/rooms/ . 2>&1 | tail -20
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

and change the doc comment's second sentence to: `https, and exactly api.openai.com or an Azure OpenAI host (*.openai.azure.com), unless the operator has deliberately allowed another provider.`

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
`Modules.baubles.*` to this slice. Task 0 Step 3 confirms what actually
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

In `internal/configs/config_locks.go`, in `hardLocked`, after the `Modules.aicompanion.DeepModel` row (or after the companion's last row, if slice M ordered them differently), add:

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
Expected: FAIL: zero width space, RLO, BOM and others "survived"; link cases "must be refused"; the 41-rune and 19-rune cases; the long quote; both folds (the curly quotes and dashes survive today, and the ellipsis too, since today's `cleanLine` does no NFKC).

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

Known limits of `linkRE`, accepted (owner review 2026-09-28) and written into the context.md in Task 15:

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

Run: `go test -race ./internal/items/ -run "TestAuthored" -v`
Expected: PASS. (`-race` needs cgo; if the local toolchain refuses `-race`, run without it and note that CI's Linux job runs it.)

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
// module did: player-key text must be moderated and plain (spec S3).
func TestGenerateHoldsPlayerKeyTextToItsRules(t *testing.T) {
	cases := map[string]GenResult{
		`unmoderated`: {Reply: goodReply(), PlayerKey: true, Moderated: false},
		`not plain`: {Reply: func() Reply {
			r := goodReply()
			r.Name = "P\u0430inted Wooden Horse"
			return r
		}(), PlayerKey: true, Moderated: true},
	}
	for name, res := range cases {
		res := res
		installGenerator(t, func(ctx context.Context, req GenRequest) (GenResult, error) { return res, nil })
		if got := Generate(context.Background(), GenRequest{Tier: TierAverage}, nil); got.Generator != GeneratorLocal {
			t.Errorf("%s: a generic trinket, got %+v", name, got)
		}
	}
	ok := GenResult{Reply: goodReply(), PlayerKey: true, Moderated: true}
	installGenerator(t, func(ctx context.Context, req GenRequest) (GenResult, error) { return ok, nil })
	if got := Generate(context.Background(), GenRequest{Tier: TierAverage}, nil); got.Generator != GeneratorOpenAI || !got.PlayerKey {
		t.Fatalf("moderated plain player-key text is used: %+v", got)
	}
}
```

- [ ] **Step 6: Run to verify it fails**

Run: `go test ./internal/baubles/ -run TestGenerateHoldsPlayerKeyTextToItsRules -v`
Expected: FAIL for both `unmoderated` and `not plain`.

- [ ] **Step 7: Enforce it in Generate**

In `internal/baubles/generate.go` `Generate`, replace

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
	if err == nil && res.PlayerKey {
		// Text a player's own key wrote reaches other players: it is
		// moderated or refused, and held to plain words (spec S3).
		if !res.Moderated {
			err = errors.New(`player-key text was not moderated`)
		} else {
			err = CheckPlayerKeyText(cleaned)
		}
	}
	if err != nil {
		mudlog.Warn(`baubles`, `action`, `generate`, `result`, `generic trinket`, `error`, err)
		return generic()
	}
```
and add `"errors"` to the file's imports.

- [ ] **Step 8: Run all baubles tests**

Run: `go test ./internal/baubles/`
Expected: `ok`.

- [ ] **Step 9: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/baubles/playerkey.go internal/baubles/playerkey_test.go internal/baubles/generate.go internal/baubles/generate_test.go && git commit -F - <<'EOF'
fix(baubles): player-key text is moderated and plain, or a trinket

CheckPlayerKeyText holds player-key text to ASCII letters and simple
punctuation. baubles.Generate refuses any player-key result that is
unmoderated or fails it, whatever the generator did.

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

### Task 10: S3, the module's route pre-check, allowlist fallback and moderation policy

The pre-check is its own function, called in the route condition and
nowhere else. FinalTwist's fix round (a "moderate before paying" reorder of
`generate` and `viaServer`, among others) has merged before this slice
starts; Task 0 Step 4 confirmed every block quoted below still reads as
quoted. This task changes `name`'s player branch, `moderate`'s body, the
lines of `generate` around `CleanReply`, and the system prompt.

Ruling 15 decides the allowlist's place: a player-key reply whose text
parses and cleans but fails `CheckPlayerKeyText` is NOT the player's key
failing. Their breaker hears nothing, and the find goes on to the server's
route (when that is closed, a generic trinket; slice C's corpus later).
So the check runs inside `name`, before the route is final
(`refusedByAllowlist`), and not in `generate`, which feeds every parse or
clean failure to the route's breaker.

**Files:**
- Modify: `modules/baubles/generate.go` (`generate`, `name`, `moderate`, new `playerRouteOpen`, new `refusedByAllowlist`, new errors)
- Modify: `modules/baubles/prompt.go` (`PromptVersion`, `systemPrompt`)
- Modify: `modules/baubles/baubles.go` (`info`)
- Modify: `modules/baubles/baubles_test.go`

- [ ] **Step 1: Make the fake answer one result per input**

In `modules/baubles/baubles_test.go`, add two fields to `fakeOpenAI` after `modStatus`:

```go
	flagWord       string       // when set, any moderation input containing it is flagged
	lastModeration atomic.Value // the last moderation inputs, newline-joined
```

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
// Moderation policy (spec S3): a flag always keeps a find out, and a check
// that cannot be made keeps out a find on EITHER key. Text a player's own
// key wrote reaches other players, so it is moderated or refused.
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
	f.modStatus, f.flagged = 0, true
	if _, err := m.generate(context.Background(), request()); err == nil {
		t.Fatal("a flag keeps a player-key find out")
	}
	f.flagged = false
	if res, err := m.generate(context.Background(), request()); err != nil || !res.PlayerKey || !res.Moderated {
		t.Fatalf("a clean check: named on the finder's key, moderated: %+v %v", res, err)
	}
}
```

(b) In `TestNoKeyAtAll`, replace everything from the comment `// A finder's own key still names it;` to the end of the function with:

```go
	// With no server key there is no moderation, so a finder's own key is
	// not used at all (spec S3): still no name.
	relay := &fakeRelay{allowed: map[int]bool{7: true}, model: `player-model`, provider: f}
	apiframework.SetRelay(relay)
	if _, err := m.generate(context.Background(), request()); !errors.Is(err, errNoRoute) || relay.sends != 0 {
		t.Fatalf("no server key: the finder's key is not asked either: %v sends=%d", err, relay.sends)
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
// A finder's own key is asked only when its text can be moderated: moderation
// on, a server key, the provider breaker closed (spec S3). Otherwise the
// find goes straight to the server's route, and the player's key is never
// spent on text that would be refused.
func TestPlayerRouteNeedsTheServersModeration(t *testing.T) {
	cases := map[string]func(t *testing.T, f *fakeOpenAI) *BaublesModule{
		`moderation off`: func(t *testing.T, f *fakeOpenAI) *BaublesModule {
			return testModule(t, f, nil)
		},
		`breaker open`: func(t *testing.T, f *fakeOpenAI) *BaublesModule {
			m := testModule(t, f, func(c *Config) { c.ModerateOutput = true })
			apiframework.SetBreakerForTest(5, time.Now().Add(time.Minute))
			return m
		},
	}
	for name, build := range cases {
		f := newFakeOpenAI(t)
		m := build(t, f)
		relay := &fakeRelay{allowed: map[int]bool{7: true}, model: `player-model`, provider: f}
		apiframework.SetRelay(relay)
		res, _ := m.generate(context.Background(), request())
		if relay.sends != 0 || res.PlayerKey {
			t.Errorf("%v: the finder's key must not be asked (sends=%d)", name, relay.sends)
		}
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
Expected: FAIL for `TestModerationOutageRefusesAPlayerKeyFind`, `TestNoKeyAtAll`, `TestPlayerRouteNeedsTheServersModeration`, `TestPlayerKeyTextOutsideTheAllowlistFallsBackToTheServer` (the accented name is used on the finder's key today), `TestPlayerKeyTypographyIsFoldedNotRefused` (today `generate` keeps the RAW reply, so the description still carries the curly apostrophe and the em dash), `TestEveryTextFieldIsModerated` (two inputs sent, not four) and `TestSystemPromptStatesTheAllowedCharacters`.

- [ ] **Step 5: Implement the pre-check**

In `modules/baubles/generate.go`, extend the error block:

```go
var (
	errNoRoute     = errors.New(`no key to name it with`)
	errBreakerOpen = errors.New(`the server key's breaker is open`)
	errUnmoderated = errors.New(`player-key text needs moderation, which is off`)
)
```

Add, directly above `name`:

```go
// playerRouteOpen reports whether a find may be named on the finder's own
// key at all: only when its text can be moderated afterwards, which needs
// ModerateOutput on, a server key, and the provider breaker closed (spec
// S3). Checked BEFORE the relay is asked, so a player's key is never spent
// on text that would be refused, and kept apart from the route itself so a
// reorder of the server route leaves it alone.
func playerRouteOpen(cfg Config, now time.Time) bool {
	return cfg.ModerateOutput && apiframework.Server().HasKey() && !apiframework.BreakerOpen(now)
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
	if cfg.UsePlayerKeys && req.FinderUserId > 0 && playerRouteOpen(cfg, time.Now()) {
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

Replace the whole of `moderate` and its doc comment with:

```go
// moderate checks the name, keyword (NameSimple), description and material
// when ModerateOutput is on, through the server's key (a player's key page
// reaches no moderation endpoint). The policy, decided and pinned by test
// (spec S3, ruling 15):
//
//   - A flag always keeps the text out of the world: a generic trinket.
//   - A check that cannot be made (no server key, the provider breaker
//     open, the check failing) keeps the text out too, on EITHER key. Text a
//     player's own key wrote reaches other players, so it is moderated or
//     refused; playerRouteOpen keeps the player route closed whenever the
//     check could not be made, so this refusal is the rare race.
//   - With ModerateOutput off, a server-key find is not checked; a
//     player-key find is refused (playerRouteOpen never lets one through).
//
// The check is free and is not a model call, so it reserves nothing and
// feeds no breaker; it does not try while the provider breaker is open.
func (m *BaublesModule) moderate(cfg Config, reply baubles.Reply, playerKey bool) (bool, error) {
	if !cfg.ModerateOutput {
		if playerKey {
			return false, errUnmoderated
		}
		return false, nil
	}
	s := apiframework.Server()
	if !s.HasKey() {
		return false, errNoRoute
	}
	if apiframework.BreakerOpen(time.Now()) {
		return false, errBreakerOpen
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
	if err != nil {
		return false, fmt.Errorf(`moderation: %w`, err)
	}
	for _, f := range flags {
		if f {
			return false, errors.New(`moderation flagged the reply`)
		}
	}
	return true, nil
}
```

Update `generate`'s doc comment route paragraph to: `The route: the finder's own key first, when they allowed it on the key page (apiframework.PurposeFinds) and its text can be moderated (playerRouteOpen); then the server's key, reserved against the one daily budget every feature shares; else no name.`

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
		detail += ` No server key: every find is a generic trinket (a finder's own key needs the server's moderation).`
	} else if cfg.UsePlayerKeys && !cfg.ModerateOutput {
		detail += ` Finders' own keys are not used while ModerateOutput is off.`
	}
```

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

- [ ] **Step 10: Probe the pre-check and the allowlist fallback**

Temporarily change `playerRouteOpen`'s body to `return true`, run `go test ./modules/baubles/ -run "TestPlayerRouteNeedsTheServersModeration|TestNoKeyAtAll" -v`. Expected: FAIL naming `moderation off` and `breaker open` ("must not be asked") and `TestNoKeyAtAll`. Restore, rerun, PASS.

Temporarily change `refusedByAllowlist`'s last line to `return false`, run `go test ./modules/baubles/ -run TestPlayerKeyTextOutsideTheAllowlistFallsBackToTheServer -v`. Expected: FAIL "refused on the finder's key, named on the server's" (the accented name is kept on the player's key; `baubles.Generate` would refuse it later, but this test calls the module directly). Restore, rerun, PASS.

- [ ] **Step 11: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add modules/baubles/generate.go modules/baubles/prompt.go modules/baubles/baubles.go modules/baubles/baubles_test.go && git commit -F - <<'EOF'
fix(baubles): player-key text is moderated or refused

A finder's own key is asked only when moderation is on, the server has a
key and the provider breaker is closed. Any moderation failure refuses a
player-key reply. Text outside the plain-text allowlist falls back to the
server's key without touching the player's breaker. The keyword and
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

(b) In `generate`, delete the whole block from `// A fixed number of calls at once.` through the closing `}` of the `select` (the `m.mu.Lock()`, `slots := m.slots`, `m.mu.Unlock()` and the `select`).

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

Run: `go test -race ./modules/baubles/`
Expected: `ok` (or the toolchain refuses `-race`; note it and rely on CI).

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
	// ...and not under 5% x (1 - 0.2) = 4%.
	if _, found := RollFind(FindOpts{Place: NewPlace(602, `town`, ``, `interior`), SightPenalty: 0.2, Randn: always(40000), Now: t0}); found {
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

Also temporarily delete the new `sightExemptSites` row and rerun: expected FAIL listing `internal/actions/search_bauble.go:56 in var searchBaubleRoll: baubles.RollFind` (56 at `e711ee9de`; the line shifts if the file shifted). Restore, rerun, PASS.

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
| `internal/actions/stolen_bauble.go` | `points at <ansi fg="username">%s</ansi>. "That's mine! Thief!"` (`ownerRecognizes`) | `[]string{carrier.GetCharacter().Name}` |
| `internal/actions/steal.go` | `is caught trying to pocket the <ansi fg="itemname">%s</ansi>!` (`stealHouseholdBauble`) | `[]string{actor.GetName()}` |

In `search.go` and `steal.go` the excluded id is `actor.GetUserId(),`; the names argument goes on its own line before it. In `stolen_bauble.go` the call ends `m.Character.Name, carrier.GetCharacter().Name), carrier.GetUserId())`; the result is:

```go
	room.SendTextVisualHidingNames(messaging.CategoryMobEmote, fmt.Sprintf(
		`<ansi fg="mobname">%s</ansi> points at <ansi fg="username">%s</ansi>. "That's mine! Thief!"`,
		m.Character.Name, carrier.GetCharacter().Name), []string{carrier.GetCharacter().Name}, carrier.GetUserId())
```

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

### Task 15: Docs and the config comment

**Files:**
- Modify: `internal/baubles/context.md`, `modules/baubles/context.md`, `internal/items/context.md`, `internal/apiframework/context.md`, `internal/configs/context.md`, `internal/actions/context.md`, `internal/usercommands/context.md` (only if it describes look's room lines; check with `grep -n "SendTextVisual" internal/usercommands/context.md`)
- Modify: `_datafiles/config.yaml` (comments only)
- Modify: `docs/aicompanion/settings.md`, `docs/baubles/implementation-plan.md`

No `docs/README.md` edit: this plan's row reaches master through the separate docs-only PR (ruling 11), and the slice adds no new non-code file.

- [ ] **Step 0: Verify every symbol the docs will name**

```bash
cd /c/tmp/dogmud-baubles-h && grep -n "func CheckPlayerKeyText\|func AuthoredName\|SightPenalty float64\|func playerRouteOpen\|func refusedByAllowlist\|func (m \*BaublesModule) takeFinderSlot\|func (m \*BaublesModule) takeServerSlot\|var keyTextRE\|var typographyFold\|const PromptVersion = 5" -r internal modules
```
Expected: ten hits.

- [ ] **Step 1: `internal/baubles/context.md`**

- In the **validate.go** file bullet, change to: `` `CleanReply` (the text checks the schema cannot make: NFKC, curly quotes, en and em dashes and the ellipsis folded to ASCII (`typographyFold`), invisible and format characters dropped, rune lengths, link-shaped text refused, an authored item's whole name refused through `items.AuthoredName`, errors quoting at most 60 runes) and `PlainText`. ``
- Add a gotcha: `` - **`linkRE` is a heuristic.** It misses a top-level domain longer than six letters and a domain written with U+3002 (NFKC keeps it); server-key text is moderated, and player-key text is held to the ASCII allowlist, which refuses both. It also refuses a missing-space typo such as `horse.Its` on every route; that find falls back like any unusable reply. Accepted by the owner, 2026-09-28. ``
- Add a file bullet: `` - **playerkey.go**: `CheckPlayerKeyText`, the plain-text allowlist for text a player's own key wrote, on the CLEANED name, keyword, description and material: ASCII letters, space, `' " - , . ! ?`, and every run of periods followed by a space, a `"` or the end. ``
- In the **generate.go** bullet add: `` `Generate` refuses a `PlayerKey` result that is not `Moderated` or fails `CheckPlayerKeyText`; `RecentNames` skips `PlayerKey` records. ``
- Change the `GenResult` signature line to `/* Reply, Generator, Model, PromptVersion, Tokens, Moderated, PlayerKey */` and the `FindOpts` line to `type FindOpts struct{ Place Place; UserId int; SkillFactor float64; SightPenalty float64; Feature string; Household bool; Randn func(n int) int; Now time.Time }`.
- Add to the find section: `` `FindOpts.SightPenalty` (0 is none) multiplies the chance by `1 - SightPenalty` after the nothing-here check and before the window roll, so a search in the dark spends its roll as one in the light does; callers pass `1 - messaging.SightMult`. ``
- In the mint section add: `` `Mint` rolls a `PlayerKey` find's value with `tier.RollValue`, keeping the key's proposal in `ValueProposed`. `ApplyRegenerated` takes the new result's `PlayerKey`. ``

- [ ] **Step 2: `modules/baubles/context.md`**

- Replace "How a call goes" step 2 with: `` 2. `name` picks the route; a slot is taken there (see 3 and 4). ``
- In step 3, after "`PurposeFinds)` answers" insert `` and `playerRouteOpen` holds (`ModerateOutput` on, a server key, the provider breaker closed: player-key text must be moderated, so the player's key is never spent on text that would be refused) ``, and after "on their key." insert `` It takes the finder's own slot (`takeFinderSlot`, one in flight per finder), never a server slot; a busy one goes to the server. The relay's token count passes through `Charged(..., relayed=true)`. An answer that parses and cleans but fails `baubles.CheckPlayerKeyText` (`refusedByAllowlist`) is not the key's failure: its breaker hears nothing and the find goes on to the server's key (ruling 15). ``
- In step 4, after "`viaServer`:" insert `` one of the `MaxConcurrent` server slots (`takeServerSlot`; none free is `errSlotsBusy` at once, not a queue; it covers the model call only); ``
- Replace step 6 with: `` 6. `baubles.ParseReply` and `CleanReply` (the cleaned text is kept; a player-key reply already passed `baubles.CheckPlayerKeyText` in `name`). Then, with `ModerateOutput`, the moderation endpoint on the name, keyword (`NameSimple`), description and material, through the server key (`moderate`). A flag refuses. A check that cannot be made (no server key, the provider breaker open, an error) refuses on EITHER key; with `ModerateOutput` off a player-key reply is refused. The check is free and feeds no breaker. ``
- Replace the "Moderation on a player's key" gotcha with: `` - **Moderation on a player's key.** Mandatory. The route is closed unless the check can be made (`playerRouteOpen`), and a check that fails after the call refuses the reply. Its text is also held to plain ASCII after folding (`CheckPlayerKeyText`; text outside it falls back to the server's key without touching the player's breaker), its value is re-rolled by the server (`Mint`), and its name never enters another prompt (`RecentNames`). Pinned by `TestModerationOutageRefusesAPlayerKeyFind`, `TestPlayerRouteNeedsTheServersModeration`, `TestNoKeyAtAll`, `TestPlayerKeyTextOutsideTheAllowlistFallsBackToTheServer`, `TestPlayerKeyTypographyIsFoldedNotRefused`. ``
- In the Config paragraph change `` `MaxConcurrent` (4) `` to `` `MaxConcurrent` (4, server-key calls only) `` and add after the list: `` `Model`, `MaxCompletionTokens`, `MaxConcurrent`, `UsePlayerKeys`, `ModerateOutput` and `ModerationModel` are hard-locked (`configs.hardLocked`): only config.yaml sets them. ``
- In the **prompt.go** bullet (it lists the `PromptVersion` history), add: `` 5: the system prompt states the characters the player-key allowlist accepts. ``
- In the **generate.go** bullet list `playerRouteOpen` and `refusedByAllowlist` beside `name`; add a **baubles.go** mention of `takeServerSlot` and `takeFinderSlot` in the file list.

- [ ] **Step 3: `internal/items/context.md`**

After the `AuthoredKeyword(word)` sentence add: `` `AuthoredName(name)` is whether a loaded item's whole name matches after NFKC, lower case and collapsed spaces (`normalizeItemName`); `baubles.CleanReply` refuses such a name. Both read one snapshot (`authored`, an atomic pointer to words and names together). `` and change `(`authoredWords`, an atomic pointer)` to `(`authored`)`.

- [ ] **Step 4: `internal/apiframework/context.md`**

- In the **wire.go** bullet change `` `Reply` and `DecodeChat`; `` to `` `Reply` and `DecodeChat` (a non-200's kept text has key-shaped strings scrubbed by `keyTextRE` before the 300-byte cut); ``.
- In the config paragraph, after the `AllowCustomEndpoint` description add: `` Without it `EndpointAllowed` accepts exactly `api.openai.com` and `*.openai.azure.com` over https; Azure's AI Services hosts (`*.cognitiveservices.azure.com`, `*.services.ai.azure.com`) and every other host need `AllowCustomEndpoint`, which, like the other three, is hard-locked (`configs.hardLocked`), so only config.yaml sets it. `APIKey` is a `configs.ConfigSecret`. ``

- [ ] **Step 5: `internal/configs/context.md`**

Where the `APIFramework` section is shown (the block around `APIKeyEnv: "OPENAI_API_KEY"`), add below it: `` `APIFramework.APIKey` is a `ConfigSecret`. `hardLocked` also holds `Modules.baubles.Model`, `.MaxCompletionTokens`, `.MaxConcurrent`, `.UsePlayerKeys`, `.ModerateOutput` and `.ModerationModel`. `` (If slice M's context.md text already lists `hardLocked`'s rows, add the six baubles rows to that list instead.)

- [ ] **Step 6: `internal/actions/context.md`**

In the bauble search paragraph (the one naming `baubles.RollFind`), add: `` Both bauble rolls pass `SightPenalty: 1 - messaging.SightMult(char, room)`; the root sight guard watches `baubles.RollFind` and `actions.searchBaubleRoll`. `BaubleRequestForRecord` omits a `PlayerKey` record's name. ``

In the stolen-baubles paragraph (the one beginning `**Stolen baubles after the theft (`stolen_bauble.go`), add: `` The owner's "points at" line and the household "caught trying to pocket" line (`steal.go`) go through `SendTextVisualHidingNames` with the player's name. ``

- [ ] **Step 7: `internal/usercommands/context.md`**

Run:
```bash
cd /c/tmp/dogmud-baubles-h && grep -n "SendTextVisual\|observer\|room line" internal/usercommands/context.md
```
Only if that prints a line describing how `look.go` tells the room (at `e711ee9de` it printed only the crafting observer paragraph, lines 466-479, which is not about `look`; the file describes what `look` shows the looker, not the room lines), add beside it: `` Every observer line in `look.go` goes through `SendTextVisualHidingNames` with the looker's name. `` Otherwise leave the file alone.

- [ ] **Step 8: `_datafiles/config.yaml` comments**

First confirm the disk copy equals HEAD and check the skip-worktree bit:
```bash
cd /c/tmp/dogmud-baubles-h && git diff --quiet HEAD -- _datafiles/config.yaml; echo "differs=$?"; git ls-files -v _datafiles/config.yaml
```
If `differs=0`: edit the disk copy with the Edit tool. If `differs=1`: STOP editing disk; write `git show HEAD:_datafiles/config.yaml` to the scratchpad, make the edits there with the Edit tool, then stage it with `blob=$(git hash-object -w <scratch copy>) && git update-index --cacheinfo 100644,$blob,_datafiles/config.yaml`, and if `git ls-files -v` then prints `H` where it printed `S`, restore the bit with `git update-index --skip-worktree _datafiles/config.yaml`.

Edits (all comments; no value changes):

Replace
```yaml
  # A find is named through the finder's OWN key when they ticked "Also name
  # things I find while searching" on the Companion key page (UsePlayerKeys),
  # else through the server's key (APIFramework: its one daily budget and
  # breaker, shared with the AI companion), else it is a generic "Trinket".
```
with
```yaml
  # A find is named through the finder's OWN key when they ticked "Also name
  # things I find while searching" on the Companion key page (UsePlayerKeys)
  # and the server can moderate what it writes (ModerateOutput on, a server
  # key, its breaker closed), else through the server's key (APIFramework:
  # its one daily budget and breaker, shared with the AI companion), else it
  # is a generic "Trinket". Model, MaxCompletionTokens, MaxConcurrent,
  # UsePlayerKeys, ModerateOutput and ModerationModel are hard-locked: only
  # this file changes them.
```

Replace
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
    # endpoint (needs the server's key); fails closed on EITHER key. Text a
    # finder's own key wrote reaches other players, so it is moderated or
    # refused, and must also be plain ASCII letters and ' " - , . ! ? (curly
    # quotes and dashes are folded first); text that is not goes to the
    # server's key instead. Off: finders' own keys are not used at all.
```

- [ ] **Step 9: `docs/aicompanion/settings.md`**

- Line 30's comment: change `Off: any other host is refused` to `Off: only api.openai.com and *.openai.azure.com are accepted (Azure AI Services hosts need it on)`.
- Replace the bauble paragraph beginning `The key page also has a box, "Also name things I find while searching` with:

```markdown
The key page also has a box, "Also name things I find while searching (uses
this key)", off unless the player ticks it. Ticked, and with
`Modules.baubles.Enabled` and `UsePlayerKeys` on, the baubles that player
finds are named on their own key through the same relay, with nothing of
theirs in the request (only the room's authored text), but only while the
server can moderate the answer: `Modules.baubles.ModerateOutput` on, a
server key, and the provider breaker closed. Unlike the companion's speech,
a bauble's text is shown to other players, so it is always moderated (any
check that fails refuses it), held to plain ASCII letters and simple
punctuation (text that is not goes to the server's key instead, without
counting against the player's key), never passed to another find's prompt,
and its value is rolled by the server. Unticked, or when moderation is
unavailable, their finds use the server's key, or stay generic trinkets
when there is none. The relay page refuses the bauble request shape from a
key whose box is not ticked.

`AllowCustomEndpoint` off accepts exactly `api.openai.com` and Azure OpenAI
resources (`*.openai.azure.com`). Azure's AI Services hosts
(`*.cognitiveservices.azure.com`, `*.services.ai.azure.com`) need it on, and
it is hard-locked: only the config file changes it.
```

- [ ] **Step 10: `docs/baubles/implementation-plan.md`**

Replace the bullet
```markdown
- **Moderation** of a player-key find: a flag refuses; a check that cannot be
  made accepts it unmoderated (a server-key find is refused), so a working
  player key never becomes a trinket over the server's route.
```
with
```markdown
- **Moderation** of a player-key find (hardening, 2026-09-28): mandatory. The
  player route is used only while the check can be made; a flag or a failed
  check refuses on either key; player-key text must also pass
  `baubles.CheckPlayerKeyText` after curly quotes and dashes are folded,
  and text that does not goes to the server's key without counting against
  the player's.
```

- [ ] **Step 11: Run the context.md audit**

Run: `cd /c/tmp/dogmud-baubles-h && python tools/context_md_audit.py 2>&1 | grep -E "baubles|items|apiframework|configs|actions|usercommands" ; echo "exit=$?"`
Expected: no phantom symbols in the touched packages (grep exit 1 means none listed). The tool only reads files; it writes nothing.

- [ ] **Step 12: Commit**

```bash
cd /c/tmp/dogmud-baubles-h && git add internal/baubles/context.md modules/baubles/context.md internal/items/context.md internal/apiframework/context.md internal/configs/context.md internal/actions/context.md _datafiles/config.yaml docs/aicompanion/settings.md docs/baubles/implementation-plan.md && git commit -F - <<'EOF'
docs(baubles): hardening in context.md, settings and config comments

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```
Add `internal/usercommands/context.md` to the `git add` only if Step 7 changed it. If Step 8 staged `config.yaml` through `update-index`, leave it out of `git add` (it is already staged) and confirm with `git diff --cached --stat` before committing.

---

### Task 16: Gate

**Files:** none new.

Every diff below runs from `$BASE`, the master commit Task 0 Step 2 recorded, never `master...HEAD`: master can move while the slice is in flight, and the PR's diff is what CI and the reviewers see.

- [ ] **Step 1: gofmt, avoiding the Windows CRLF false positive**

`gofmt -l` on a Windows checkout can list a file whose committed LF blob is fine (the disk copy has CRLF). Check the committed blobs of every Go file this slice changed:

```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && for f in $(git diff --name-only $BASE..HEAD -- '*.go'); do out=$(git show HEAD:"$f" | gofmt -l); [ -n "$out" ] && echo "UNFORMATTED: $f"; done; echo done
```
Expected: only `done`. For any listed file, run `gofmt -w <file>`, check `git diff` shows only whitespace, and amend it into a fixup commit.

- [ ] **Step 1b: Size stays under the CI lint inversion**

Run:
```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && git diff --shortstat $BASE..HEAD
```
Expected: fewer than 300 files changed and fewer than 20,000 lines (insertions plus deletions). CI's lint gate inverts past either (owner ruling 11); at or over, STOP and report, and split the slice rather than push it.

- [ ] **Step 2: Build and vet**

Run: `cd /c/tmp/dogmud-baubles-h && go build ./... && go vet ./internal/baubles/ ./internal/items/ ./internal/apiframework/ ./internal/configs/ ./internal/actions/ ./internal/usercommands/ ./internal/rooms/ ./modules/baubles/ ./modules/aicompanion/ .`
Expected: no output.

- [ ] **Step 3: Targeted tests**

Run:
```bash
cd /c/tmp/dogmud-baubles-h && go test ./internal/baubles/ ./modules/baubles/ ./internal/apiframework/ -run "CleanReply|PlayerKey|Authored|RollFind|Sight|Moderat|Route|Slot|Relayed|Endpoint|DecodeChat|Key|Mint|RecentNames|Regenerat|Typography|TextField|SystemPrompt" -v 2>&1 | grep -E "^(--- FAIL|FAIL|ok)"
```
Expected: only `ok` lines.

- [ ] **Step 4: Full test run for every touched package (not all of internal/combat)**

Run:
```bash
cd /c/tmp/dogmud-baubles-h && go test ./internal/baubles/ ./internal/items/ ./internal/apiframework/ ./internal/configs/ ./internal/actions/ ./internal/usercommands/ ./internal/rooms/ ./internal/messaging/ ./modules/baubles/ ./modules/aicompanion/ . 2>&1 | tail -20
```
Expected: every line `ok`. The repo-root `.` carries the sight guard, the messaging surface guard, the contest floor guard and the line-number allowlist guards. `internal/rooms`' `TestDeleteZone_RemovesEveryTree` and `TestRenameZone_MovesRewritesAndRekeys` fail on Windows only under `DOGMUD_BOOT_SMOKE=1` and skip otherwise; do not set that variable for this run (Step 4c sets it for the root package alone).

- [ ] **Step 4b: The root guards and the key guard, by name**

Run:
```bash
cd /c/tmp/dogmud-baubles-h && go test . -count=1 -run "TestEveryRollSiteAppliesTheSightPenalty|TestSightPenaltyGuard|TestEveryTextSurfaceIsRegistered|TestNarrationSitesMatchViewpointAudit|TestOpposedContestsAreFloored" -v 2>&1 | grep -E "^(--- FAIL|--- PASS|FAIL|ok)"
```
```bash
cd /c/tmp/dogmud-baubles-h && go test ./internal/apiframework/ -count=1 -run TestNoTestPrintsAKey -v 2>&1 | grep -E "^(--- FAIL|--- PASS|FAIL|ok)" && go test ./internal/items/ ./internal/apiframework/ -count=1
```
Expected: every named test `--- PASS` (the sight guard, its three probes and the bauble seam probe; the messaging surface registry and viewpoint audit; the contest floor guard; the key guard), then `ok` for `internal/items` and `internal/apiframework`. A named test that does not appear at all means its name changed on master: find it with `grep -n "^func Test" *_test.go` before calling the gate green.

- [ ] **Step 4c: Boot smoke (loads every data file; starts no server)**

Run:
```bash
cd /c/tmp/dogmud-baubles-h && DOGMUD_BOOT_SMOKE=1 go test . -count=1 -run TestSmoke_ServerBootsCleanWithRealData -timeout 600s 2>&1 | tail -5
```
Expected: `ok`. This is `boot_smoke_test.go`, the automated form of dogmud-shipping's boot check: it calls the real `loadAllDataFiles` in the test process and opens no port, so it cannot touch the owner's running server. If a real boot is wanted as well, use dogmud-shipping's detached-worktree `boot-check.exe` recipe, and stop only that process, by its own PID; never kill by process name or port.

- [ ] **Step 5: Race run on the concurrency-touched packages**

Run: `cd /c/tmp/dogmud-baubles-h && go test -race ./internal/items/ ./modules/baubles/ ./internal/baubles/`
Expected: `ok` (if the toolchain cannot run `-race` locally, record that; CI runs it).

- [ ] **Step 6: Lint what CI's lint gate sees**

Run: `cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && golangci-lint run --new-from-merge-base=$BASE`
Expected: `0 issues`. dogmud-shipping's documented form is `--new-from-merge-base=origin/master`; `$BASE` is that merge base at branch time and stays right if master moves. A finding on a line this slice did not write is pre-existing; check `git log -L` before hunting.

- [ ] **Step 7: go.mod untouched**

Run: `cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && git diff --exit-code $BASE..HEAD -- go.mod go.sum; echo "exit=$?"`
Expected: `exit=0`.

- [ ] **Step 8: Every touched package's context.md was updated**

Run:
```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && git diff --name-only $BASE..HEAD | sed -n 's#^\(internal/[^/]*\|modules/[^/]*\)/.*#\1#p' | sort -u
```
```bash
cd /c/tmp/dogmud-baubles-h && BASE=$(cat /c/tmp/dogmud-baubles-h.base) && git diff --name-only $BASE..HEAD -- '*context.md' _datafiles/config.yaml docs/
```
For each package the first command prints, confirm `<pkg>/context.md` is in the second's output, except `internal/rooms` (test-only change) and `internal/usercommands` (only if Task 15 Step 7 found nothing to say). Also confirm `_datafiles/config.yaml`, `docs/aicompanion/settings.md` and `docs/baubles/implementation-plan.md` are there, and that `docs/README.md` is NOT (this plan's row travels in the docs-only PR).

- [ ] **Step 9: Nothing staged by accident, config bit intact**

Run: `cd /c/tmp/dogmud-baubles-h && git status --short && git ls-files -v _datafiles/config.yaml`
Expected: a clean tree; the config flag letter is what Task 15 Step 8 found it to be before editing.

No push, no PR, no deploy here: the owner runs deploys, and pushing is a separate instruction. When the owner asks to push, every `gh` command carries `--repo pruuk/DOGMud`.

---

## Self-review

**Spec coverage.** S1: Task 1 (`ConfigSecret`, `Validate`, key guard through `DisplayConfigData`). S2: Task 2 (exact hosts), Task 3 (`DecodeChat` scrub, both key forms), Task 4 (six baubles rows in `hardLocked`, `ModerateOutput` and `ModerationModel` included per ruling 13). S3: pre-check (Task 10 `playerRouteOpen`), post-refusal on any failure (Task 10 `moderate`), allowlist (Task 7 engine; Task 10 `refusedByAllowlist` in `name`, falling back to the server route without feeding the player's breaker, ruling 15), typography folded before the allowlist (Task 5 `typographyFold`, pinned in Tasks 5, 7 and 10), the allowed characters in the prompt (Task 10, `PromptVersion` 5), `NameSimple` moderated and allowlisted (Tasks 7 and 10), material moderated (Task 10), value (Task 8), tokens (Task 11), slots (Task 11), recent names and regen (Task 9), authored names (Task 6), log quoting (Task 5 `quoteShort`, used by Tasks 6 and 7). S4: Task 5 (NFKC, Zs, Cf/Co/Cs/Mn, U+2028/2029, Hangul fillers, links, runes, code-point table, OSC, NBSP; `linkRE`'s accepted limits documented) and the homoglyph pin (Task 7). U2: Tasks 12 and 13 (field, order, log, callers, guard entry, exemption, both probes). U4: Task 14 (the six spec lines, their seven `look.go` siblings, and the two PR bauble theft lines; the characterization test pins behaviour and the grep checks the conversion). Docs: Task 15. Gate: Task 16 (diffs from `$BASE`, size under 20k lines and 300 files, lint from `$BASE`, the named root guards and the key guard, the boot smoke). Delivery: Task 0 (fresh branch from master after #175 and slice M, every quoted anchor grepped).

**Placeholder scan.** Tasks 1 and 4 are written against slice M's planned names (`config_locks.go`, `hardLocked`, `isHardLocked`, `IsLocked`, `DisplayConfigData`); Task 0 Step 3 checks the landed code matches before either task starts. Every other quoted anchor was read at `e711ee9de` and is re-checked against merged master by Task 0 Step 4, which stops the task that owns a missing anchor.

**Type consistency.** `CheckPlayerKeyText(r Reply) error`, `AuthoredName(name string) bool`, `playerRouteOpen(cfg Config, now time.Time) bool`, `refusedByAllowlist(content string) bool`, `typographyFold *strings.Replacer`, `takeServerSlot() (release func(), ok bool)`, `takeFinderSlot(userId int) (release func(), ok bool)`, `FindOpts.SightPenalty float64`, `quoteShort(s string) string`, `errUnmoderated`, `errSlotsBusy`, `chatBody(content string, tokens int) string` are used with those signatures everywhere.
