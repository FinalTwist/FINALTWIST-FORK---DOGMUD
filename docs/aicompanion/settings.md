# AI Companion Settings

Every setting the module has, with its built-in default and what it does.
Change any of them in `_datafiles/config.yaml` under `Modules: aicompanion:`,
or in `_datafiles/config-overrides.yaml`, which wins over both.

The module ships no `data-overlays/config.yaml` of its own, deliberately: a
plugin overlay is pushed into the live config after `config.yaml` is read and
overwrites it (only `config-overrides.yaml` is protected), so an overlay would
silently ignore the file operators actually edit. The defaults below live in
`buildConfig` instead.

```yaml
# Defaults for Modules.aicompanion.*. Override in _datafiles/config.yaml.
#
# OFF BY DEFAULT. Set Modules.aicompanion.Enabled: true in _datafiles/config.yaml
# to use it; while it is false this module loads nothing and the server behaves
# as though it were not built in.
#
# Switched on, nothing else needs configuring to use OpenAI: set the
# OPENAI_API_KEY environment variable (the one OpenAI's own tools read) before
# starting the server, and the module picks suitable models for each tier by
# itself. Without a key, companions still appear, follow, fight and answer with
# short authored lines.
Enabled: false
# The main model. Empty = automatic: the first of gpt-5.4-mini, gpt-5-mini,
# gpt-4.1-mini, gpt-4o-mini that the key can use.
Model: ""
# Model tiers. Model is the main tier (conversation and relationship: someone
# speaks, asks, gives, hurts or heals; greetings and goodbyes). FastModel is a
# small, low-latency model for moments that need an answer quickly (combat
# plans, noticing things, quiet moments, follow-ups). DeepModel is the
# strongest model, used only for the private reflection after a session.
# An empty tier chooses automatically: fast from gpt-5.4-nano, gpt-5-nano,
# gpt-4.1-nano, gpt-4o-mini; deep from gpt-5.5, gpt-5.4, gpt-5, gpt-4.1,
# gpt-4o. A model the API refuses is skipped and the next one used.
FastModel: ""
DeepModel: "gpt-4.1-mini"
# Reasoning effort per tier: none, minimal, low, medium, high or xhigh. Empty
# = automatic (none for fast and main, which keeps replies quick and is
# required for function tools on current models; medium for deep). Adjusted
# to what each model family accepts, and not sent to models without it.
FastReasoningEffort: ""
MainReasoningEffort: ""
DeepReasoningEffort: ""
# Token and time limits for the fast and deep tiers (main uses
# MaxCompletionTokens and RequestTimeoutSeconds).
FastMaxCompletionTokens: 500
DeepMaxCompletionTokens: 2000
FastTimeoutSeconds: 12
BaseURL: "https://api.openai.com/v1"
# Environment variable holding the API key.
APIKeyEnv: "OPENAI_API_KEY"
# Alternatively, the key itself. Convenient on a private, single-player
# server; anyone who can read the config file can read the key. The
# environment variable wins when both are set. The key is never logged.
APIKey: ""
RequestTimeoutSeconds: 25
# Upper bound on generated tokens per decision (reasoning models count their
# reasoning against this, so do not set it too low).
MaxCompletionTokens: 900
# 0 = do not send a temperature (required for some reasoning models).
Temperature: 0
# Minimum real seconds between two model calls for one companion.
MinSecondsBetweenCalls: 2
# Total tokens per UTC day across all companions. 0 = unlimited.
DailyTokenBudget: 2000000
# Rounds a fallen bonded companion takes to recover and rejoin mid-session.
RecoveryRounds: 40
# How many recent lines of conversation the companion keeps word for word.
WorkingMemoryLines: 40
# How many lines of that are sent with each decision.
PromptMemoryLines: 24
# Long-term memories retrieved into each decision.
PromptMemories: 8
# Long-term memory store size. The least important, least recent memories are
# forgotten first; importance 8 and above are never forgotten.
MaxMemories: 300
# Facts kept about the owner.
MaxFacts: 40
# Session summaries kept.
MaxSummaries: 20
# Speak up when the owner logs in.
GreetOnLogin: true
# A mood the model set drifts back to calm after this many real minutes
# without being set again. 0 = moods never fade on their own.
MoodDecayMinutes: 20
# Log one line per decision: trigger kinds, the model's private intent,
# lines spoken, tokens and latency. Player text itself is never logged.
LogDecisions: true
# Reflect privately on the session when the owner logs out (one extra call).
ReflectOnLogout: true
# Skip the reflection when less than this much happened.
MinSessionLinesForReflection: 4
# Minutes of quiet before the companion considers starting a conversation.
# Whether it does depends on the profile's talkativeness. 0 = never.
InitiativeMinutes: 5
# How interesting (0..1, local score) something must be for the companion to
# notice it aloud when it walks in or when it appears.
NoticeThreshold: 0.6
# Minimum seconds between two "you notice" moments.
NoticeCooldownSeconds: 45
# Minutes between chances to deal with something nearby on its own when
# nothing else is happening. 0 = never acts unprompted.
AutonomyMinutes: 3
# Minutes between small idle gestures (authored, no model call). 0 = none.
IdleEmoteMinutes: 6
# How many room things are listed, with refs, in each decision.
OptionsInPrompt: 10
# May the companion go off on its own (errands, exploring one step) while
# its owner stays put? The owner moving always calls it back.
AllowErrands: true
# Longest route, in steps, the companion will take on an errand.
MaxErrandSteps: 15
# Rounds the companion lingers at an errand's end before heading back.
ErrandLingerRounds: 8
# Rounds apart after which a companion that knows no way back rejoins its
# owner as though it had followed.
LostRounds: 40
# Nearby known places with something in them, listed in each decision.
NearbyPlacesInPrompt: 6
# Rooms the companion remembers; the least recently seen are forgotten first.
MaxKnownRooms: 5000
# When a reply to someone speaking to the companion takes longer than this
# many seconds, it makes a small "thinking" gesture. 0 = never.
ThinkingSeconds: 4
# Rounds between two of the companion's own combat moves on top of the
# engine's fighting (switching target, taunting, drinking, fleeing). 1 = at
# most one a round, never in the round something happened.
CombatReactionRounds: 1
# Retry a call once, after a short pause, on a rate limit, server error or
# timeout.
RetryTransient: true
# Circuit breaker: after this many failures in a row, stop calling the model
# for BreakerSeconds and use fallback lines.
BreakerErrors: 5
BreakerSeconds: 60
# Tokens per UTC day for any one companion. 0 = only the server budget.
DailyTokensPerCompanion: 300000
# Check the companion's speech with the OpenAI moderation endpoint before it
# is said. Adds a short delay to each reply. Server key only: a call on a
# player's own key is not moderated (see the tiers below).
ModerateOutput: true
ModerationModel: "omni-moderation-latest"
# Keep a rotating backup of each mind every N sessions (three are kept).
# 0 = no backups.
BackupEverySessions: 1
# Rounds between copies of the companion's gear, gold and skills into its
# owner's saved record, so a restart or crash loses little. It is also
# copied right after any trade or pickup.
SnapshotRounds: 25
# How many times, in one decision, the model may ask the game for more
# information (look closer at something, recall memories, search its map)
# before it must answer. Main tier only. 0 = never.
ToolRounds: 2
# Meeting a companion. A new character meets one right after character
# creation, once it has stood in its first real room for MeetDelayRounds
# (not in the void, a skipped zone, a fight, or quitting). With
# AutoBondExisting, a character that has never had one meets one on its
# next login. If the player sends the companion away it does not come back
# on its own (an admin can still use aicompanion grant).
AutoBond: true
AutoBondProfile: "mara"
AutoBondExisting: false
MeetDelayRounds: 3
MeetSkipZones:
  - "Newcomer Antechamber"
# Seconds the owner has to confirm with `companion-part` after a companion
# asks to leave. The bond never ends on the model's word alone.
LeaveConfirmSeconds: 120
# Treat anything the owner says as spoken to the companion when no other
# player is in the room. With this off, the companion answers only when it
# is named or asked (`ask <name> ...`). Everything a player says in a room
# with the companion is sent to the API either way, as remembered context.
RespondWhenAlone: true
# How much a companion's nerve wanders from one fight to the next (0..1):
# it shifts when they decide to run and how readily they throw themselves in
# front of their owner, so the same companion is not a machine. 0 = exact.
CombatVariance: 0.25
# Minutes between the small things a companion does with itself when
# nothing is happening: a gesture, a search of the room, a look through the
# undergrowth. Chosen at random from whatever makes sense there, with no
# model call. Searching and foraging are also how those skills grow.
IdlePastimeMinutes: 4
# The chance she does anything at all when the moment comes round. At 0.5
# and four minutes, something happens every eight minutes on average, and
# never on a predictable beat.
IdlePastimeChance: 0.5
# Remember speech in the room that was not addressed to the companion. It is
# what lets it overhear and react later; it also means other players' words
# reach the API as context. Turn it off on a shared server.
RecordBystanderSpeech: false
# Allow BaseURL to point somewhere other than OpenAI (an OpenAI-compatible
# provider). Off, any other host is refused and the official endpoint used,
# so a mistyped URL cannot send the API key and player text elsewhere.
AllowCustomEndpoint: false
# Talk is remembered as a whole, not line by line: an exchange in one room
# becomes a single memory in her own words when it ends ("we talked about
# bread, and he told me his mother baked it"). A talk ends when it has gone
# quiet for ConversationGapSeconds, when she moves, when a fight starts, or
# at logout. Shorter exchanges than MinConversationExchanges leave nothing
# unless something notable was said.
ConversationSummaries: true
ConversationGapSeconds: 240
MinConversationExchanges: 3
# While the owner is sneaking, the companion stays where it is and keeps
# still: it does not follow, speak, notice things aloud or run errands. A
# second set of footsteps is what gets people caught.
HoldWhenSneaking: true
# After a fight, a bonded companion walks the ground and works some of its
# loosed arrows free: a random share between these two bounds, never all of
# them. This is the companion only; the player's arrows are the engine's
# business and are untouched.
RecoverArrows: true
RecoverArrowsMin: 0.1
RecoverArrowsMax: 0.8
# How many of the world's recent notable happenings reach her when she is
# somewhere with people to have heard them from. 0 = she hears no rumours.
RoadTalkLines: 3
# She walks after her owner rather than arriving with them: a moment's
# pause, then she takes the same way out herself, so the room sees her leave
# and enter like anyone else. Only when they actually walked; a portal, a
# ferry or being pulled out of a fight still moves her the engine's way.
FollowOnFoot: true
FollowDelayMin: 0.15
FollowDelayMax: 0.45
# A player is asked, in as many words, before anything they say is sent to
# OpenAI. Until they answer "i agree" their companion behaves as a plain one.
# Turning this off means players are never told; think hard before you do.
RequireConsent: true
# What a passer-by can ask of somebody else's companion: one question per
# this many seconds each, and this many tokens a day each. Their asking is
# charged to them, not to the companion's owner, and it never moves how the
# companion feels about its owner.
StrangerAskSeconds: 30
StrangerDailyTokens: 50000
# What passers-by, all of them together, may spend of one owner's companion
# in a UTC day, on either key: many strangers each within their own
# allowance could otherwise spend one owner's key without end. A fight a
# passer-by started counts as theirs too. 0 is no cap.
StrangerTokensPerOwner: 100000
# Player keys (tier 2): a player runs their own companion on their OWN key,
# from the web client. The key stays in their browser, on a relay page served
# from RelayOrigin, and never reaches this server. Off by default here; the
# shipped _datafiles/config.yaml turns it on. Nothing is offered unless
# RelayOrigin is also set.
PlayerKeys: false
# The relay page's origin: "https://" plus a host on its OWN subdomain, for
# example "https://keys.example.org". It must be https and must not be the
# game's own host (FilePaths.WebDomain); anything else offers nothing.
RelayOrigin: ""
# Seconds a call waits for the player's browser (which includes the
# provider's own answer) before she falls back on set lines for that turn.
# Values under 5 are raised to 5.
RelayTimeoutSeconds: 30

```

## Who pays for a call: the three tiers

Each model call is paid for by the first of these that is available for the
companion's OWNER, even when a passer-by is the one talking to her:

1. **The owner's own key (tier 2).** The owner has the web client open with a
   key set up and unlocked (`Companion.Relay.Ready` received this session).
2. **The server's key (tier 3).** `APIKeyEnv` or `APIKey` holds a key.
3. **Nobody (tier 1).** She follows, fights and answers with authored lines.

What changes on the owner's own key:

- The server's `DailyTokenBudget` and `DailyTokensPerCompanion` are not
  charged (the player pays).
- Passers-by prompt NO calls on the owner's key until the owner says
  `companion-ai strangers on`; she hears them and answers with set lines.
  An owner who has never used the command is off on their own key and on
  for the server's key; `companion-ai strangers off` stops them on both.
  Once on, `StrangerAskSeconds`, `StrangerDailyTokens` and
  `StrangerTokensPerOwner` limit what passers-by can spend of the owner's
  key, and a count the browser reports is never trusted past what was
  reserved.
- `ModerateOutput` does not apply: the player's provider may have no
  moderation endpoint, and a reply from a browser could be forged anyway.
  Each line she says that way is logged at Info against the owner instead,
  and a muted owner's companion says nothing on any tier.
- The model is the one the player chose, for every tier of call (fast, main,
  deep); reasoning effort is not sent.
- Failures count against that owner's own breaker (`BreakerErrors`,
  `BreakerSeconds`), never the global one.
- Her private reflection at logout cannot reach a closed browser, so it
  runs at the owner's next login once their key is ready.
- The owner can read every prompt their browser carries, so those prompts
  leave out what belongs to other players: speech she only overheard from
  someone else (`RecordBystanderSpeech`), and what another player looks like
  or carries (a closer look at them tells how they are and what kind they
  are, nothing more). Their names, and what they did or said to her, stay.

`companion-ai` tells a player which tier is answering; `aicompanion status`
shows it per companion (`tier=relay|server|none`) for an admin.

### Deploying player keys

1. A DNS record for the relay subdomain (for example `keys.example.org`)
   pointing at the same server as the game.
2. A site block in the reverse proxy (Caddy on our droplet) for that host,
   proxying to this server's web port with the `Host` header kept as sent.
   The Go server tells the relay apart from the game by `Host` alone.
3. `RelayOrigin: "https://keys.example.org"` and `PlayerKeys: true` in the
   production config.
4. HSTS on both hosts in the proxy (`Strict-Transport-Security:
   max-age=31536000; includeSubDomains`, a `header` line in each Caddy site
   block). The relay refuses to run outside https, but a player's first
   visit over plain http can be intercepted before any redirect; HSTS
   closes that for every later visit.

### What the relay protects, and what it does not

The key lives only on the relay origin: typed in the relay's own window,
kept in the relay frame's memory (or, when remembered, encrypted in the
relay origin's storage), and sent only to the endpoint the player stored.
No script on the game page can READ it: the browser keeps another origin's
memory and storage out of reach, and the relay answers only with the
provider's reply, never a header.

A script running on the game page (an injected script, a hostile browser
extension, a cross-site scripting bug in the web client) CAN still SPEND
the key: it can post requests to the relay frame exactly as the game page
does, and the relay cannot tell them apart. What bounds that is the relay's
own cap (at most 2 requests in flight and 30 a minute, whatever asks) and
the spending cap the setup text tells every player to set with their
provider. That is the boundary to state to players: the key cannot be
stolen from the game page, but while their game page is compromised it can
be used, within those caps, until they close it or forget the key.

Three operator traps:

- **The key is typed in a pop-up window on the relay host.** The relay
  frame opens it from a click inside the frame, so browsers allow it, but
  a player who has blocked pop-ups for the relay host sees a note asking
  them to allow it. The window's address bar is the player's proof of where
  the key is going: the setup text tells them to check it.
- **Players must reach the game on exactly the `FilePaths.WebDomain` host.**
  The relay page allows only `https://` plus `WebDomain` to frame it
  (`frame-ancestors`), so a player on `www.example.org` when `WebDomain` is
  `example.org` (or the other way round) sees no key setup at all. Redirect
  the other name to `WebDomain`.
- **A local model needs to allow the relay origin.** The browser posts to it
  from the relay page, so Ollama must be started with `OLLAMA_ORIGINS` set to
  the relay origin (for example `OLLAMA_ORIGINS=https://keys.example.org`);
  the setup panel tells the player the same.
