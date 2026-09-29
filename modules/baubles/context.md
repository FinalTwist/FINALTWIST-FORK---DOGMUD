# baubles module Context

## Purpose

The model side of bauble loot (docs/baubles/implementation-plan.md, Phases 4
and 5e). The engine side, the catalog, the search roll and delivery, is
`internal/baubles` and `internal/actions/search_bauble.go`; this module only
installs the namer with `baubles.SetGenerator`. Every request goes through
`internal/apiframework`, the one mechanism the AI companion uses too.

Off by default twice over: `Balance.BaublesEnabled` (no finds at all) and
`Modules.baubles.Enabled` (no naming). With naming off, or no key to name a
find with, every find is a generic "Trinket" (value and weight random within
the tier). With naming on, each find is named, described, weighed and priced
from the room it was found in.

## Files

- **baubles.go**: module registration, `onLoad`, `configure` (installs the
  generator whenever `Enabled`), `onNewRound` (refreshes the server key's
  settings on the game loop: `apiframework.RefreshServer`, since a find is
  named off it and may only read the snapshot), `count` (stats by route), `info` for
  `bauble status` (routes and the shared budget; reports the naming breaker
  and the moderation breaker separately), `onSave`
  (`apiframework.SaveBudget`), `takeServerSlot`, `takeFinderSlot`.
- **config.go**: `Config`, `buildConfig` (defaults and bounds).
- **prompt.go**: `PromptVersion` (2: a targeted search sends what was
  searched and its authored description, `searched_description`; 3: in wild
  places whose description shows only nature, the find is usually a natural
  curiosity, a crystal, rough gem, fossil, shell, amber or a piece of bone,
  antler or tooth, never a crafting material, with a specific keyword; 4: a
  pickpocketed find is lifted from a person, `taken_from` the NPC's authored
  name (`GenRequest.Victim`), and is POCKET-SIZED, `size_rule` naming the
  weight limit the catalog enforces, `sizeRule`; 5: the system prompt states
  the characters the player-key allowlist accepts), the system prompt,
  `buildMessages`. The model judges "only nature" from the room text and
  terrain it is sent; no flag is computed. The thief is never named.
- **generate.go**: `generate` (the GeneratorFunc), `name` (the route),
  `viaPlayer`, `viaServer`, `moderate`, `moderationPossible`,
  `moderationBreaker`, `refusedByAllowlist`.

## How a call goes

1. A delivery goroutine (no mud lock) calls `baubles.Generate`, which calls
   `generate` with a context capped at `baubles.MaxGenerateTime`.
2. `name` picks the route; a slot is taken there (see 3 and 4). A busy
   server slot returns `errSlotsBusy` before `generate` counts anything.
3. `name` picks the route. With `UsePlayerKeys`, and a finder
   (`GenRequest.FinderUserId`) whose Companion key page has "Also name things
   I find while searching" ticked, `apiframework.PlayerRelay().Model(userId,
   PurposeFinds)` answers and `viaPlayer` sends the request through that
   player's browser relay, on their key. It costs the server nothing but is
   held against the finder's own `baubles.finder` allowance
   (`finderCharges`); a refusal is no call and feeds no breaker, and the
   find goes on to the server's key, where the same allowance refuses it. A
   pickpocket's find (`SourcePickpocket`) takes the same route on the
   thief's key; its naming starts the moment the roll succeeds, so a thief
   reading their browser's network traffic can see a success before the
   reveal (accepted by the owner: every find follows one key order). Its
   outcome (any failure, a reply that does not decode included) feeds that
   player's FINDS breaker only, never their companion's. It takes the
   finder's own slot (`takeFinderSlot`, one in flight per finder), never a
   server slot; a busy one goes to the server. The relay's token count
   passes through `Charged(..., relayed=true)`. An answer that parses and
   cleans but fails `baubles.CheckPlayerKeyText` (`refusedByAllowlist`) is
   not the key's failure: its breaker hears nothing and the find goes on to
   the server's key (ruling 15).
4. Otherwise, or when that fails, `viaServer`: one of the `MaxConcurrent`
   server slots (`takeServerSlot`; none free is `errSlotsBusy` at once, not
   a queue; it covers the model call only); the server key
   (`apiframework.Server`); leave from the breakers (`apiframework.Allow` for
   `ConsumerBaubles`: baubles' own and the provider's; a half-open breaker
   lets one probe through); a hold on the one daily budget, the baubles
   share and the finder's allowance (none for an admin's regeneration,
   which has no finder); `apiframework.Post` with the strict
   `baubles.ReplySchema()`, retried once when `RetryTransient`; `Charged`
   and `Settle`. Each route returns `report`,
   and `generate` gives it ONE outcome for the whole find, retries included,
   once the answer has been parsed and checked (`ParseReply`, `CleanReply`),
   so a model that keeps ignoring the schema pauses baubles' own breaker.
   A find given up on (`context.Canceled`, a copyover's flush) is released
   unjudged; one that runs out of time counts, as the provider not
   answering.
   A model or request the provider refuses (400, 404) pauses baubles' own
   breaker only; only provider-wide failures reach the companion's.
5. Neither route: `errNoRoute`, a generic trinket.
6. `baubles.ParseReply` and `CleanReply` (the cleaned text is kept). Then
   `moderate`, on the name, keyword, description and material, through the
   server key, reading the server settings and the clock once. Server-key
   text: checked when `ModerateOutput` is on (a flag, or a check that
   cannot be made or fails, refuses), unchecked when off. Player-key text:
   checked whenever `moderationPossible` (`ModerateOutput` on, a server
   key, neither the provider breaker nor the moderation breaker open); a
   flag or a failed check refuses. Every check made, on either route, is
   recorded on the moderation check's OWN breaker, `moderationBreaker`
   (`apiframework.RecordConsumer`), never on the naming breaker
   `ConsumerBaubles` (whose run a good naming call resets) and never on the
   provider's. When moderation is not possible no call is made and the
   find is `FinderOnly`: its finder reads it, everyone else the generic
   trinket (owner ruling 2026-09-29).
7. The engine (`baubles.Generate`) then runs `CleanReply` and
   `ApplyLimits`; any failure anywhere is a generic trinket.

## Config (`Modules.baubles` in config.yaml)

`Enabled` (false), `UsePlayerKeys` (true), `Model` (gpt-5-nano),
`ReasoningEffort` (minimal; not sent on a player's key, whose model the
player chose), `Temperature` (0, not sent), `TimeoutSeconds` (15, 3 to 30),
`MaxCompletionTokens` (800), `RetryTransient` (false), `MaxConcurrent` (4,
server-key calls only), `DailyTokensPerUser` (20000; each finder's
`baubles.finder` allowance, on either key; 0 is no cap; not hard-locked),
`ModerateOutput` (true), `ModerationModel` (omni-moderation-latest),
`LogRequests` (false). `Model`, `MaxCompletionTokens`, `MaxConcurrent`,
`UsePlayerKeys`, `ModerateOutput` and `ModerationModel` are hard-locked
(`configs.hardLocked`): only config.yaml sets them.

The key, endpoint, daily budget, the baubles share
(`APIFramework.BaublesSharePercent`) and breaker are not here: they are the
`APIFramework` section's, shared with the companion.

## Gotchas

- **No player data.** Every request is `apiframework.CarriesNoPlayerData`:
  only authored room text, region and terrain, the time of day, what was
  searched and recent bauble names. Nothing a player wrote, said or is
  called. `FinderUserId` picks the relay and is never put in the prompt.
  Keep it that way: a request that carried player data would need the
  companion's consent door.
- **A player's key only by their choice.** The relay is lent only for
  `PurposeFinds`, which the player opts into on the key page; the browser
  relay itself refuses the bauble schema otherwise.
- **Moderation on a player's key.** Moderated when the server can, and then
  refused on a flag or a failed check; kept to its finder when it cannot,
  never shown to anyone else unmoderated. Also held to plain ASCII after
  folding (`CheckPlayerKeyText`; text outside it falls back to the server's
  key without touching the player's breaker), its value re-rolled by the
  server (`Mint`), its name never in another prompt (`RecentNames`), and
  never in the AI companion's prompts either (`items.Item.ModelName`).
  Pinned by `TestModerationOutageRefusesAPlayerKeyFind`,
  `TestFailedChecksOpenTheModerationBreakerAlone`,
  `TestPlayerKeyTextThatCannotBeModeratedIsFinderOnly`, `TestNoKeyAtAll`,
  `TestPlayerKeyTextOutsideTheAllowlistFallsBackToTheServer`,
  `TestPlayerKeyTypographyIsFoldedNotRefused`.
- **The moderation breaker has no probe.** `RecordConsumer` carries no
  ticket, so once its cooldown is up every check is let through again, and
  `BreakerErrors` failures in a row reopen it. Fine for a free check; do
  not copy it for a model call.
- **The key** is sent only in the Authorization header, never logged, never
  in a request body, never printed by a test (`apiframework`'s key guard reads
  this package's tests).
- **No game state is touched here.** Everything the goroutines share is
  behind `mu`; `configure` runs on the game loop at load.
- **A refused reservation is not a failure.** A spent day, share or finder
  allowance (`apiframework.RefusedBy`) makes no call, feeds no breaker and
  is not counted in `bauble status`; `baubles.Generate` logs it with the
  counter's name.
- **`bauble status` counts roll on the ledger's clock.** `count` and `info`
  read `apiframework.Shared().Day()`, not a clock of their own; a stats day
  that does not match it is stale and shown as zero, so the figures reset
  at the ledger's midnight, not at boot or at the module's own rollover.
- Markup is stripped from everything sent (`baubles.PlainText`).
- The package is named `baubles`, like the engine package it imports. The
  tests import that one as `eng`.

## Dependencies

`internal/apiframework`, `internal/baubles`, `internal/mudlog`,
`internal/plugins`.
