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
  `bauble status` (routes and the shared budget), `onSave`
  (`apiframework.SaveBudget`).
- **config.go**: `Config`, `buildConfig` (defaults and bounds).
- **prompt.go**: `PromptVersion` (2: a targeted search sends what was
  searched and its authored description, `searched_description`; 3: in wild
  places whose description shows only nature, the find is usually a natural
  curiosity, a crystal, rough gem, fossil, shell, amber or a piece of bone,
  antler or tooth, never a crafting material, with a specific keyword; 4: a
  pickpocketed find is lifted from a person, `taken_from` the NPC's authored
  name (`GenRequest.Victim`), and is POCKET-SIZED, `size_rule` naming the
  weight limit the catalog enforces, `sizeRule`), the system prompt,
  `buildMessages`. The model judges "only nature" from the room text and
  terrain it is sent; no flag is computed. The thief is never named.
- **generate.go**: `generate` (the GeneratorFunc), `name` (the route),
  `viaPlayer`, `viaServer`, `moderate`.

## How a call goes

1. A delivery goroutine (no mud lock) calls `baubles.Generate`, which calls
   `generate` with a context capped at `baubles.MaxGenerateTime`.
2. A slot is taken (`MaxConcurrent`); none free means an error at once, not
   a queue.
3. `name` picks the route. With `UsePlayerKeys`, and a finder
   (`GenRequest.FinderUserId`) whose Companion key page has "Also name things
   I find while searching" ticked, `apiframework.PlayerRelay().Model(userId,
   PurposeFinds)` answers and `viaPlayer` sends the request through that
   player's browser relay, on their key. It costs the server nothing. A
   pickpocket's find (`SourcePickpocket`) takes the same route on the
   thief's key; its naming starts the moment the roll succeeds, so a thief
   reading their browser's network traffic can see a success before the
   reveal (accepted by the owner: every find follows one key order). Its
   outcome (any failure, a reply that does not decode included) feeds that
   player's FINDS breaker only, never their companion's.
4. Otherwise, or when that fails, `viaServer`: the server key
   (`apiframework.Server`); leave from the breakers (`apiframework.Allow` for
   `ConsumerBaubles`: baubles' own and the provider's; a half-open breaker
   lets one probe through); a hold on the one daily budget; `apiframework.Post`
   with the strict `baubles.ReplySchema()`, retried once when
   `RetryTransient`; `Charged` and `Settle`. Each route returns `report`,
   and `generate` gives it ONE outcome for the whole find, retries included,
   once the answer has been parsed and checked (`ParseReply`, `CleanReply`),
   so a model that keeps ignoring the schema pauses baubles' own breaker.
   A find given up on (`context.Canceled`, a copyover's flush) is released
   unjudged; one that runs out of time counts, as the provider not
   answering.
   A model or request the provider refuses (400, 404) pauses baubles' own
   breaker only; only provider-wide failures reach the companion's.
5. Neither route: `errNoRoute`, a generic trinket.
6. `baubles.ParseReply`; then, with `ModerateOutput`, the moderation endpoint
   on the name and description, through the server key (`moderate`). A flag
   always refuses. A check that cannot be made (no server key, the provider
   breaker open, an error) refuses a server-key find, and accepts a
   player-key find unmoderated (`Moderated` false in its record). The check
   is free and feeds no breaker.
7. The engine (`baubles.Generate`) then runs `CleanReply` and
   `ApplyLimits`; any failure anywhere is a generic trinket.

## Config (`Modules.baubles` in config.yaml)

`Enabled` (false), `UsePlayerKeys` (true), `Model` (gpt-5-nano),
`ReasoningEffort` (minimal; not sent on a player's key, whose model the
player chose), `Temperature` (0, not sent), `TimeoutSeconds` (15, 3 to 30),
`MaxCompletionTokens` (800), `RetryTransient` (false), `MaxConcurrent` (4),
`ModerateOutput` (true), `ModerationModel` (omni-moderation-latest),
`LogRequests` (false). `Model`, `MaxCompletionTokens`, `MaxConcurrent`,
`UsePlayerKeys`, `ModerateOutput` and `ModerationModel` are hard-locked
(`configs.hardLocked`): only config.yaml sets them.

The key, endpoint, daily budget and breaker are not here: they are the
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
- **Moderation on a player's key.** It uses the server key. When the check
  cannot be made (no server key, the provider breaker open, the endpoint
  failing), a find named on the player's key is accepted unmoderated (the
  same rule the companion follows on a player's key), so a working player
  key never turns into a trinket over the server's route; a server-named
  find is refused. A flag always refuses. Pinned by
  `TestModerationOutageNeverSpoilsAPlayerKeyFind`.
- **The key** is sent only in the Authorization header, never logged, never
  in a request body, never printed by a test (`apiframework`'s key guard reads
  this package's tests).
- **No game state is touched here.** Everything the goroutines share is
  behind `mu`; `configure` runs on the game loop at load.
- Markup is stripped from everything sent (`baubles.PlainText`).
- The package is named `baubles`, like the engine package it imports. The
  tests import that one as `eng`.

## Dependencies

`internal/apiframework`, `internal/baubles`, `internal/mudlog`,
`internal/plugins`.
