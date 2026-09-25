# aicompanion Module Context

## Purpose

`modules/aicompanion` drives **bonded companions**
(`characters.CompanionBonded`): persistent companions that talk, remember and
change mood through a language model (OpenAI chat completions with a strict
JSON schema). They ride on the existing companion system, so they appear with
their owner at login, leave at logout, follow, fight with the normal mob
combat AI, and progress skills by use exactly like any other companion.

The model never touches game state. The only things it can cause are
ordinary `say` and `emote` mob commands (phase 1). Later phases add more
ordinary mob commands; the rule stays the same.

Roadmap and phase plan: `docs/aicompanion/`.

## Files

- **aicompanion.go**: module registration, `controller`, the mind cache
  (`getMind`), config/profile loading, save callback, budget counters.
- **runtime.go**: the round loop (`sync`): attach/detach, session greeting,
  farewell during `quit`, fall and recovery, initiative after quiet spells,
  mood decay; `dispatch`, `applyResult` (speech, memory, facts, promises,
  bounded opinion change), the decision trace, the fallback path.
- **listeners.go**: `say`, emotes (`events.Emote`), gifts (`GiftAccepted`),
  attacks (`PlayerAttackedMob`), healing (`events.Healed`) and the `ask`
  hook; an owner speaking to the companion interrupts an errand. They record what the
  companion perceived and queue stimuli; they never call the model. The
  per-companion halves (`hearSaid`, `seeEmote`, `hearAsked`) read the
  owner's literal "i agree"/"i decline" first, and before consent write
  nothing into the mind while still queueing the stimulus, so dispatch
  answers with set lines. Anything a passer-by aims at her (speech to
  her, `ask`, an emote, a gift, healing) is heard and remembered as usual
  but queued only if `strangerMayAsk` passes (their daily allowance, then
  a per-companion cooldown that trying spends), called once per thing,
  just before the push. Their stimuli carry `AskerUserId`. With the
  owner's `companion-ai strangers off` (`bondRecord.StrangersOff`,
  `strangersOff`) she still hears them and answers with set lines, paced
  by the same cooldown, but `strangerMayPrompt` stops any call they would
  prompt: dispatch falls back, and a talk with passers-by alone is not
  summed up.
- **scene.go**: `buildScene`, the structured, locally scored picture of what
  the companion can see and carry, with the refs ("t2", "p1", "w1") the
  model uses in actions; item and NPC classification; novelty from the
  interaction memory.
- **actions.go**: `performAction` validates a proposed verb and ref against
  the scene and the live world, applies the loot arrangement, answers
  `look_at` and `consider` itself (the same information a player gets), or
  issues one ordinary mob command; `verifyPending` judges the outcome from
  what changed.
- **autonomy.go**: `perceive` (runs each round: settles the last action,
  notices new rooms and new things, offers quiet moments to act),
  `handleIdle` (owns the idle tick: first aid, idle gestures), impressions of
  NPCs and places.
- **worldmap.go**: the companion's own map (`Mind.Map`): rooms it has
  stood in, exits it has walked, a clearly marked guess at the way back,
  features and people seen, danger; the Dijkstra route finder over that map
  only; nearby places; place search; hearsay and its confirmation.
- **travel.go**: trips. `go_to`, `explore` and the automatic return walk a
  route one ordinary `go <exit>` at a time, waiting for each arrival,
  recording blocked exits, re-planning from wherever the companion ends up,
  and giving up cleanly.
- **archetype.go**: archetype validation against the game's skills, gear
  fit, skill rank words and rank-change detection.
- **inventory.go**: supplies and shortfalls, carrying load, protected items.
- **economy.go**: `browse` (priced as `list` prices), shop and price
  memory, the money rules for buying.
- **loot.go**: the module's own mob commands `companion-loot` (owner's loot
  rights only) and `companion-takeout` (unhidden, unlocked containers).
- **harm.go**: `harmAllowed` and `areaHarmAllowed`, the one gate on
  everything she starts: the engine's own player harm rules
  (`mobs.CheckPlayerHarm` for a creature; `(*Room).CanPvp` plus the party
  check for a person) asked with her OWNER as the one acting.
- **goals.go**: goals, checks against live state, restock goals, the
  session agenda, and the model's goal proposals.
- **combat.go**: the fight from the companion's side: tracking, the
  model's plan (stance, target, flee point, style), local reflexes at most
  once a round, hold-back with the owner's auto-assist restored, authored
  battle lines, and the summary afterwards.
- **meeting.go**: meeting a companion after character creation (or at the
  next login), the persisted bond state, parting ways, and consent:
  `consented`, `answerConsent`, and `consentLedger`, the copy of who has
  agreed that the model door reads off the mud lock (rebuilt by
  `syncConsent` on every bond load and save).
- **tools.go**: the read-only questions the model may ask the game before
  answering (look closer, size up, wares, recall, find a place), answered
  under the mud lock from player-visible information only.
- **models.go**: model tiers (fast, main, deep) and their routing, the
  circuit breaker, per-companion budgets, per-tier metrics, decision traces.
- **tiers.go**: who pays for a call. `route` picks the owner's own key
  through their browser (`routeRelay`, tier 2, only when
  `playerKeysOffered`: `PlayerKeys` on and `validRelayOrigin`), else the
  server's key (`routeServer`, tier 3), else nothing (`routeNone`, set
  lines). `applyRoute` stamps the route on every `modelCall` at build
  time; `reserveRoute`, `settleRoute` and `routeResult` read that one
  route, so a call settles once, to the ledger it was held against, and
  its outcome reaches the breaker of whoever paid (`relayTable` keeps a
  per-owner breaker; the global one is the server key's alone).
- **conversation.go**: talk gathered into one conversation per exchange
  (`noteConversation`, which also notes whether her owner spoke and the
  last passer-by who did), held mid-talk notes, and `closeConversation`,
  which sums the whole talk up as one memory on the fast tier
  (`summariseConversation`) or keeps the best note when there is no model
  or nobody's allowance to pay. A talk with passers-by alone is paid for
  by the last of them (`conversation.payer`); one her owner took part in is
  the owner's.
- **reflect.go**: the private end-of-session reflection (summary,
  conclusions, facts), run in the background after logout
  (`detachReflection`). An owner whose own key was live this session
  (`controller.relaySeen`) cannot be reached at logout, so their
  reflection is copied (`deferReflection`, one per owner, the newest) and
  started by the round tick (`startDueReflection` in `sync`, under the
  lock) once they are online with their relay up (`dueReflection`), never
  on the server's key and never from the connection goroutine. The launch
  asks for consent again.
- **memory.go**: retrieval (importance, recency, relevance by words, place
  and people) and forgetting.
- **opinion.go**: `Opinion`, per-trigger envelopes, `boundDelta`
  (envelope, personality sensitivity, diminishing returns on praise),
  the audited `applyOpinion`, and `opinionWords` (the only form the model
  ever sees).
- **openai.go**: `callModel`, the blocking call (goroutine only), with a
  per-call strict JSON schema. Every request passes `admit`, the one door,
  which refuses (`errNoConsent`) any request whose `OwnerUserId` has not
  agreed, or is 0; only `listModels` (key, no player data) is exempt, as
  `carriesNoPlayerData`. There are two ways out and both call `admit`
  first: `send` (HTTP, the server's key) and `sendRelay` (the owner's
  browser, always guarded). Both transports decode through one
  `decodeChatResponse`. A relay call waits `RelayTimeoutSeconds`, not the
  tier's timeout, since it includes the browser's round trip.
- **relay.go**: the relay transport (tier 2). `pendingRelays.do` sends
  `Companion.Relay.Request {id, body}` (a random 128-bit hex id and the
  chat completions body, nothing else) and waits for
  `Companion.Relay.Response {id, status, body}`; `deliver` accepts a reply
  only from the owner it went to, for a pending id, once (refusals are
  counted in `dropped`). A reply over 1 MiB (`errRelayTooLarge`) or that
  `looksLikeAKey` (`errRelayKeyShaped`, logged once a minute without the
  text) is a failed call. `onRelayInbound` (installed as
  `companionai.SetRelayInbound` while the module is on) handles `Ready`
  (model name checked by `relayModelOK`: at most 100 printable,
  space-free characters, not key-shaped), `Gone` and `Response`, and does
  nothing while `playerKeysOffered` is false. `relayGone` (on `Gone` and
  on `PlayerDespawn`) takes the relay down and fails pending calls at once
  (`errRelayGone`). Unsent, gone, timed-out (`errRelayTimeout`),
  key-shaped and oversized failures are never retried (`relayFinal`).
  The first failure the provider causes in a relay session (not a relay
  that went away, not a cancelled call) tells the owner once, in plain
  words (`noticeFallback`, `relayOwner.noticeSent`, reset by `ready`).
- **decision.go**: the decision schema, `parseDecision`,
  `sanitizeDecision` and `cleanText`, which enforce everything the schema
  cannot.
- **prompt.go**: `buildMessages` and pure helpers (`isAddressed`,
  `humanizeElapsed`). Player text only ever appears quoted in the user
  message. Trust-gated backstory is withheld from the model until earned.
- **perception.go**: `describeSituation`, what the companion can currently
  perceive, as words (no numbers, no hidden creatures, no secret exits, no
  hidden containers, no container contents).
- **mind.go**: `Mind` (schema 2: memories, facts, promises, summaries,
  opinion and its audit log), migration from schema 1, save/load.
- **profile.go**, **profiles/*.yaml**: authored companion definitions,
  embedded in the binary.
- **commands.go**: the admin-only `aicompanion` command, including
  `aicompanion mind <character>` and a status line per companion with its
  `tier=relay|server|none`; the player's `companion-ai` (consent, the
  `strangers on|off` toggle, and which tier is answering, `tierWords`).
  Its lines are wrapped at 80 columns before sending, since the system
  category is never wrapped for the reader.
- **primer.txt**: the common-knowledge world primer given to the model.
- **config.go**: every setting and its default (`buildConfig`). The module
  ships no data-overlay: a plugin overlay overwrites `_datafiles/config.yaml`
  rather than filling in behind it. See docs/aicompanion/settings.md.

## Locking

- Listeners, the ask handler and admin commands run under the mud lock and
  must never take it.
- `dispatch` builds the whole request under the lock, then starts a goroutine
  that holds no game pointers. That goroutine takes `util.LockMud()` only
  around `applyResult`.
- Reflections run after the controller is gone. They find their Mind through
  the `minds` cache, which keeps one pointer per mind for the life of the
  process, so a reflection and a new session can never hold two copies.
- `controller.seq` is bumped on every dispatch, detach, fall and pause. A
  reply whose seq no longer matches is dropped.
- `Mind` is only touched under the lock. There is no other mutex.

## Persistence

One mind file per owner and companion, through the plugin store
(`WriteStruct`, durable and autosave-queued): identifier
`mind-<ownerUserId>-<mobId>`. A corrupt file is quarantined by
`ReadIntoStruct` and a fresh mind is used. Mechanical state (items, gold,
skills, health) is never in the mind file; it lives on the owner's
`CompanionInfo` and on the live mob.

## Model tiers

- With no model configured, `modelChooser` picks per tier from
  `modelPreferences`, using the API's model list when it could be read
  (`probeModels` at load), and moves on from any model the API refuses
  (`modelRefused`). `effortFor` adapts reasoning effort to each family.

- `tierFor` picks fast, main or deep from what triggered the call;
  `Config.settingsFor` resolves the model, token limit, timeout and
  reasoning effort, falling back to `Model`.
- Parsing and optional moderation happen on the model goroutine; the result
  carries the parsed decision into `applyResult`.
- `aicompanion prompt <character>` shows the exact last request. See
  `docs/aicompanion/testing-and-prompts.md` for what each section is for.

## Combat rules

- The engine's combat AI does the fighting; the module never blocks it.
- Plans come from the model in the background; reflexes run every round
  with `CombatReactionRounds` between moves, using only `attack #id`,
  `fire #id`, `taunt`, `flee`, `drink` and `aid @id`.
- She harms only what her owner could harm (`harmAllowed`): the `attack`
  verb, a harmful `cast`, the plan's chosen target
  (`applyCombatProposal`), the reflex strike at whatever is hurting the
  owner, and a special move at her current foe all pass it (`mayStrike`,
  `mayStrikeCurrent`). The engine gates harm by a PLAYER actor and never
  a mob one, so without this a bonded companion could reach what her owner
  may not. There is no self-defence exception, because a player gets none:
  a protected creature that turns on her is fought by the engine's round,
  not by anything she chooses. Ordinary companions are not gated; making
  "a mob acting for a player is gated as that player" engine-wide is a
  separate call.
- `refusesToFight` is personality only: shopkeepers and the profile's
  `refuse` words. What nobody may attack is the engine's to say.
- Holding back switches the owner's `AutoAssist` off for that fight only.
  The previous value is kept in `Mind.AssistToRestore` and put back when
  the fight ends, when the companion falls, when the session ends, or at
  the next attach.

## Safety rules

- The bond ends only through `companion-part` (the owner's command), never
  on the model's word: `leave` sets a request that lapses after
  `LeaveConfirmSeconds`. The exception is `collapsed()`.
- Every dispatch captures `controller.worldRev`; `answerTools` and
  `applyResult` drop everything if it has changed.
- `controller.cancelInFlight` cancels the HTTP call on logout, pause, reset
  and death. Budgets are reserved at dispatch in one check-and-hold step
  (`reserveRoute`, which is `tryReserveFor` on the server's key) and
  settled on return against the same payer and route (`settleRoute`),
  exactly once. A call a passer-by prompted (`strangerBehind`) is held
  against their `StrangerDailyTokens` and, on the server's key, the server
  budget, never the owner's allowance, and runs with no tool rounds so its
  worst case fits. On the owner's own key (tier 2) nothing of the server's
  is held: only a passer-by's allowance. `modelReadyFor(owner, asker)`
  routes by the owner even when a passer-by asks.
- `nextBatch` gives each decision one prompter: the owner's words (and an
  errand the owner sent her on) and each passer-by's are decided
  separately, with the world's stimuli going to the first. The owner-only
  verbs (`ownerPrompted`) and "ask first" (`ownerAskedNow`) are refused
  whenever any passer-by's stimulus is in the batch, whoever else spoke.
- What she says is her owner's to answer for, on every tier: a muted owner
  (`UserRecord.Muted`) silences her say, emote and `sayto` alike
  (`spokenLines`, in `speak` and the `sayto` action). On the owner's own
  key nothing moderates her words, so each line she says that way is
  logged at Info against the owner (`speechLogLine`, `logSpeech`).
- Item commands use `itemRef` (`!<itemId>:<uuid>`), never a display name.
- Sight is `messaging.ParticipantSight`, the engine's own rule.

## Acting rules

- The model never writes a command. It picks a verb from a closed list and a
  ref from the scene it was shown; code builds the command.
- A `cast` is owner-driven when the spell harms (`SpellData.IsHarm`, the
  engine's answer): `castHarm` refuses it in any batch a passer-by
  prompted, and aims it only at a creature or person `harmAllowed` passes;
  one that lands on the room (`areaHarmAllowed`) must pass for everyone it
  could catch. A helpful cast (a mending, a ward) stays her own judgement.
- One non-perception action at a time: a new one is refused until the last
  one's outcome has been judged (two rounds after issue).
- `look_at` and `consider` are answered by the module from what a player
  would read, and earn exactly one follow-up decision so the companion can
  react. A follow-up never earns another.
- Loot arrangement (`ask_first` by default): with `ask_first`, `get` is
  allowed only in a decision triggered by the owner speaking to the
  companion; `leave_it` refuses all pickups; `take_freely` allows them.
  The arrangement changes only when the model reports a new agreement.
- Travel: the companion only knows rooms it has stood in. `mapper.GetPath`
  is never used, because it searches the whole world. Errands start only
  from beside the owner, who is not fighting; the owner moving pulls the
  companion back through the engine's companion transport, which asks
  `holdFollow` per companion; only the bonded companion is ever held (a
  sneaking owner, or her walking in on foot), never another companion of
  the same owner. After
  `ErrandLingerRounds` apart it walks back by its own map; after
  `LostRounds` with no known way it rejoins through `companionai.Rejoin`.
- Exit names that are numbers or `home` are never walked: the mob `go`
  command would treat them as a teleport to a room id or as its own
  pathing.
- Buying and selling go through the ordinary mob `buy` and `sell`; a
  purchase must pass `purchaseCheck` (purse, reserve, size for the
  personality, unless a need or the owner asked). Protected items (gifts
  from the owner) cannot be sold, dropped, stored or given to anyone else.
- Checkable goals are completed only by `checkGoals`, never by the model.
- Not possible: repair (no item condition in the game), banking, skill
  training, trading item for item with the owner.

## Opinion rules

- The model proposes a change; code applies at most the envelope of what
  triggered it (talk: 2 per dimension; an emote: 3; a gift: up to 5
  affection; an attack by the owner: up to 15, and never a gain).
- Only stimuli caused by the owner can move the owner opinion.
- Three warm conversational gains within an hour and further warm words
  stop counting until deeds follow.
- Rules that apply whatever the model says: an owner attack costs 5 trust
  and 5 affection; a gift from the owner adds 1 affection (three a day at
  most); a promise the owner keeps adds 4 trust, one they break costs 6;
  a session of ten minutes or more adds a point of trust and affection
  while they are still modest.
- Every change is recorded in `OpinionLog` with its trigger and reason.

## Adding a companion

1. Author a mob template (see `_datafiles/world/dogmud/mobs/summons/9800-mara_venn.yaml`).
   It must carry **no items, equipment or gold**: a bonded companion that dies
   loses its gear to its corpse and respawns from the template, so template
   gear would be free gear on every death.
2. Author `profiles/<id>.yaml` with that `mob_id`.
3. Rebuild. `aicompanion profiles` lists what loaded; `aicompanion grant
   <character> <id>` bonds it to an online player.

## Gotchas

- **Bonded companions cannot be dismissed or renamed by command**, cost no
  Conviction reserve, and recover from death (see
  `internal/hooks/companion_bonded.go`).
- **Switched off, the module leaves bonded records alone.** Login still
  fields a bonded companion from the owner's record (the engine's
  `respawnCompanions` does not filter by source type), and nothing drives
  it. Because the bonded check (`companionai.SetBondedCheck`) is installed
  only while the module is on, `companionai.DrivesBonded` is false and the
  engine's `dismiss` lets the owner part with it peacefully. Nothing is
  deleted at boot, so switching the module back on picks the bond up again.
- **Mob commands split on `;`.** `cleanText` replaces it; never bypass it.
- **All model text is escaped** with `util.EscapeAnsiTags` before it reaches
  a mob command.
- **The companion list shows the source type**, which is why the stored value
  is `bonded`, not `ai`.
- **The farewell rides the `quit` meditation.** `quit` applies condition 0
  and the owner leaves when it expires; `sync` sees the condition and
  dispatches a goodbye immediately. A dropped connection gets no farewell.
- **Two listeners are registered even when the module is off.** The engine
  raises `events.Emote` and `events.Healed` for this module alone, so
  `onLoad` registers `onEmote` and `onHealed` before the `Enabled` check
  (both return at once while off); otherwise every emote and every heal
  cast at a creature is logged as "no listener for event". The events
  package has no notion of an event that may go unheard.
- **Config is read through the plugin config bag**, so a mistyped key is a
  silent default. `aicompanion status` shows what is actually in effect.

## Dependencies

`actions`, `characters`, `companionai`, `events`, `items`, `messaging`,
`mobs`, `mudlog`, `parties`, `plugins`, `rooms`, `spells`, `targeting`,
`users`, `util`;
`net/http` and `gopkg.in/yaml.v3`.

## Opinion

A bonded companion's feelings live in `Mind.Opinion` (three axes, audited,
no decay: only the owner's own deeds move it, and only the owner can bring
it back up). The engine's per-template `internal/opinions` score is
deliberately not moved for bonded companions (`companionai.IsBondedCompanion`
guards `actions.aggression` and the gift seeder), because that store is keyed
by mob template and every owner of the same profile would share one slot.
So `admin.opinion` does not describe a bonded companion; `aicompanion mind
<character>` does.
