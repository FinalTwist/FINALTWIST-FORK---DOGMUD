# apiframework Context

## Purpose

The one mechanism every model-backed feature uses to reach an
OpenAI-compatible API: today the AI companion (`modules/aicompanion`) and
bauble naming (`modules/baubles`). It owns the wire format, the HTTP
transport and its consent door, the server key and endpoint, one daily token
budget shared by every feature, one circuit breaker for the server key, and
the registry through which a player's own key (the companion's browser relay)
can be lent to another feature.

A feature brings its own prompts, models, timeouts and policy. It does not
keep its own key, budget or breaker.

## Files

- **wire.go**: `Message`, `ToolCall`, `ToolFunction`, `ToolSpec`, `ToolDef`;
  `Chat` and `Chat.Body()` (the chat-completions body, with an optional strict
  JSON schema and reasoning effort); `Reply` and `DecodeChat` (a non-200's
  kept text has key-shaped strings scrubbed by `keyTextRE` before the
  300-byte cut);
  `MaxToolCallsPerReply`; `EstimateTokens`; `Charged` (what a call costs the
  budget).
- **transport.go**: `HTTPClient`, `Endpoint` (redacts its key in `String`
  and `GoString`), `Carries` (`CarriesPlayerData`, the zero value, and
  `CarriesNoPlayerData`), `Admit`, `ErrNotAdmitted`, `Exchange`, `Post`,
  `NeverConnected`, `Moderate`, `ListModels`.
- **settings.go**: `DefaultBaseURL`, `DefaultKeyEnv`,
  `DefaultDailyTokenBudget`, `DefaultBreakerErrors`, `DefaultBreakerSeconds`,
  `EndpointAllowed`, `ResolveKey`, `ServerSettings` (`HasKey`, `Legacy`),
  `Server`, `RefreshServer`, `SetServerForTest`.
- **budget.go**: the shared daily ledger. `ConsumerCompanion`,
  `ConsumerBaubles`, `Hold`, `Reserve`, `Settle`, `HasRoom`, `Usage`,
  `ConsumerUsage`, `Today`, `SeedTokens`, `SaveBudget`, `ErrOverBudget`, and
  the test helpers `ResetBudgetForTest`, `SetSpentForTest`, `SetClockForTest`.
- **breaker.go**: the server key's breakers. `Allow`, `Record`, `Release`,
  `RecordConsumer` (one outcome against a consumer's own breaker alone, no
  ticket and so never the half-open probe: for a feature's check that is
  not a model call, such as baubles' moderation, which uses its own
  consumer name `baubles-moderation`) and `Ticket` (`Probing`); `Blocked`, `BreakerOpen`, `BreakerUntil`,
  `BreakerFailures`, `ConsumerFailures`, `ResetBreaker`; `ProviderFailure` and
  `StatusError` (`KeyOrProvider`), which `DecodeChat` returns for a non-200
  reply; test helpers `SetBreakerForTest`, `SetConsumerBreakerForTest`.
- **books.go**: `Books`, the budget and breaker together; `Shared` (the
  server's one set, which every package-level budget and breaker function
  uses) and `NewBooksForTest` (an isolated set). Each of those functions is
  also a `*Books` method.
- **relay.go**: `Relay`, `PurposeFinds`, `SetRelay`, `PlayerRelay`.

## Config (`APIFramework` in config.yaml)

`APIKeyEnv`, `APIKey`, `BaseURL`, `AllowCustomEndpoint` (a string: "true",
"false", or empty to inherit the companion's old setting, so an explicit
false overrides an old true; anything unreadable is false),
`DailyTokenBudget` (a negative value is no cap), `BreakerErrors`,
`BreakerSeconds`. Declared in
`internal/configs/config.apiframework.go`, whose `Validate` sets no numeric
defaults on purpose. Without it `EndpointAllowed` accepts exactly
`api.openai.com` and `*.openai.azure.com` over https; Azure's AI Services
hosts (`*.cognitiveservices.azure.com`, `*.services.ai.azure.com`) and every
other host need `AllowCustomEndpoint`, which, like the other three, is
hard-locked (`configs.hardLocked`), so only config.yaml sets it. `APIKey` is
a `configs.ConfigSecret`.

`Server()` returns a snapshot, safe from any goroutine; `RefreshServer()`
reads the config into it (`resolveServer`) and must only run on the game
loop, because the engine writes the config there with no lock of its own
(`configs.SetVal`) and a map read racing that write is a fatal error. The
companion and baubles modules refresh at load, every round and in their
admin views, so a `config set` takes effect within a round. Each setting the
section leaves empty or 0 is read from where the AI companion kept it
before, `Modules.aicompanion` (same names; `companionBlock`), and failing
that the companion's old default (`DefaultKeyEnv` OPENAI_API_KEY,
`DefaultBaseURL`, `DefaultDailyTokenBudget`, `DefaultBreakerErrors`,
`DefaultBreakerSeconds`). So a config.yaml written before the section
existed, which a patch never updates on a server, behaves exactly as the
companion did: the key from OPENAI_API_KEY (which wins over a configured
key), its BaseURL and AllowCustomEndpoint, its DailyTokenBudget (0 there is
no cap, as it always was), its breaker (floors 1 and 5 seconds).
`ServerSettings.Legacy` names the settings read from the old place; the
companion logs them once at load (names only). A `BaseURL` refused by
`EndpointAllowed` is replaced by `DefaultBaseURL` and reported in
`RejectedBaseURL`. `TestServerReadsAnOldConfigAsTheCompanionDid` pins every
case.

## How a server-key call goes

1. `Server()`; no key (`HasKey`) means the feature has no server route.
   `Blocked(consumer, now)` answers "would a call be let through?" without
   taking anything (for deciding whether to try at all).
2. `Allow(consumer, now)` immediately before the first send: a `Ticket`, or
   no (a breaker open, or half-open with its one probe already out).
3. `Reserve(consumer, worstCase)` holds tokens against the one budget.
4. `Post(ctx, ep, path, body, carries, admit)` sends it off the mud lock
   (retries of the same logical call reuse the ticket).
5. `Charged` turns the reply into what it cost; `Settle` releases the hold
   and books the real use under the consumer.
6. ONE `Record(consumer, ticket, err, now)` for the logical call, retries
   included; or `Release(consumer, ticket)` when it was given up on or never
   left (nobody's failure).

## Breakers

Two levels. The PROVIDER breaker is every feature's and counts only
`ProviderFailure` (no answer, a timeout, 401, 403, 408, 429, 5xx); a reply
the provider did give (a 400 or 404 for one feature's model or schema, a
403 that only refuses a model, `ModelRefusal`, a refusal, bad content) is
the provider answering and closes its run. Each
CONSUMER has its own breaker, fed by every failure of its calls. So a bauble
model the provider does not offer pauses baubles only, while a real outage
seen by any feature pauses them all. `BreakerErrors` failures in a row open
a breaker for `BreakerSeconds`; then it is half-open and exactly one caller
gets through as the probe (its ticket is `Probing`). The probe's success
closes it, its failure opens it again for the whole cooldown, and `Release`
frees a probe given up on. Releasing a ticket already recorded does
nothing (only the live probe's own ticket frees it), so callers can always
release last. `probeTTL` (five minutes, or the cooldown if longer; longer
than any logical call, a companion decision's retry and tool rounds
included) frees a probe whose caller never returns at all.

A player's own key goes through `PlayerRelay()` instead: no server budget,
no server breaker. The relay's owner (the companion) keeps its own per-player
counters and consent, and a breaker per purpose: another feature's results
(`Relay.Result`) never touch the one the companion runs on.

## Gotchas

- **Fail closed on player data.** `Post` and `Moderate` refuse a request
  classed `CarriesPlayerData` that has no `Admit` hook (`ErrNotAdmitted`).
  The zero value of `Carries` is the guarded kind, so an unclassified request
  is treated as carrying a player's words. The door runs after the request
  is built and immediately before it leaves. Only authored game text may be
  sent as `CarriesNoPlayerData`.
- **The key** goes only in the Authorization header. It is never logged,
  returned or put in a body. `Endpoint` prints as `[redacted]`, and
  `key_guard_test.go` reads the test files of this package, the companion and
  the baubles module and fails any test that could print a key.
- **The environment variable wins.** `ResolveKey` reads the variable it is
  given, then the configured value; `Server` gives it OPENAI_API_KEY when no
  config names one. Tests in the three packages clear OPENAI_API_KEY in
  `TestMain` and fix `Server()` with `SetServerForTest`.
- **One budget.** `Reserve` checks the whole day against
  `DailyTokenBudget`; the per-consumer figures in `Today().ByConsumer` are for
  display only, not separate caps. The companion keeps its own per-player
  caps on top.
- **Midnight.** Holds still in flight at the UTC rollover carry into the new
  day, and are settled there.
- **Living state.** The ledger is saved to
  `<DataFiles>/apiframework/budget.yaml` (`SaveBudget`, called from the
  modules' save hooks, `main.go` and `copyover.go`) and loaded lazily. A
  corrupt file is quarantined and the day starts fresh. `SeedTokens` migrates
  the companion's old saved total once, on a fresh day only. The directory is
  git- and Docker-ignored; keep it on the server like `shops/`.
- **Relays are lent per purpose.** `Relay.Model(userId, PurposeFinds)` only
  answers when the player ticked "Also name things I find while searching" on
  the key page; the companion enforces this, and the browser relay only
  accepts the bauble schema from a key with that box ticked.
- **Global state in tests.** The shared books and the settings override are
  package globals. A caller that can leave calls in flight between tests
  should spend from its own `*Books` in tests (the AI companion gives every
  test module its own, `isolateBooks`), or a finished test's call settles
  into the next test's figures. Tests that use the shared set reset it
  (`ResetBreaker`, `ResetBudgetForTest`, `SetServerForTest`) and must not run
  in parallel.

## Dependencies

`internal/configs`, `internal/util`, `internal/mudlog`. Imported by
`modules/aicompanion` and `modules/baubles`.
