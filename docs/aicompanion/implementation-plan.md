# AI Companion: Implementation Plan

Status: Phases 0 to 7 written as one cumulative patch (unverified: not yet
compiled or tested). What the model is told, and the test plan:
[`testing-and-prompts.md`](testing-and-prompts.md).
Feature spec: [`feature-spec-v2.txt`](feature-spec-v2.txt). Feature ids below
(F1.1, F7.3, ...) refer to it.

## 1. Decision: build on the existing companion system

DOGMud already has persistent companions: mob instances owned by a player,
recorded in `Character.Companions` (`internal/characters/companions.go`).
They already:

- respawn beside the owner at login (`respawnCompanions`,
  `internal/hooks/PlayerSpawn_HandleJoin.go`) and are saved then destroyed at
  logout (`saveCompanionState`, `internal/hooks/PlayerDespawn_HandleLeave.go`);
- persist skills, skill-use counts, mutations, spellbook, carried items and
  worn equipment, and progress by use;
- follow the owner room to room (`internal/hooks/companion_follow.go`);
- fight through the normal mob combat AI, and show vitals in the web client.

The AI companion is a new companion kind, `CompanionBonded`, on top of that
system. It is not a player account on the AI port. Reasons: the AI port and
`IsAI` flag belong to the playtest harness (admin `who` shows `[AI]`,
leaderboards exclude it), and building on the companion system keeps the
engine changes small, which matters because this repository moves fast.

Consequences accepted:

- The companion acts through **mob commands** (about 74) rather than player
  commands (about 186). Gaps are closed by adding mob wrappers over the
  shared `internal/actions` functions, phase by phase.
- **Mobs receive no text** (`MobActor.SendText` is a no-op). Perception is
  built from room state, filtered to what a player could see, and outcomes
  are verified from state changes and existing events that carry
  `MobInstanceId` (`ItemOwnership`, `EquipmentChange`, `RoomChange`).
- **The owner moving pulls the companion along** (`TransportCompanions`).
  Independent errands are possible only while the owner stays put. This is
  the right default for a companion and needs no change.

## 2. Ground rules and where the code enforces them

| Rule (spec Part 0) | Enforcement |
|---|---|
| R1 fairness | Bonded companions are ordinary charmed mobs. No new stats, no protections. Death is handled like any mob death; only the record survives. |
| R2 ordinary commands only | The model's output is data. `runtime.go` turns it into `mob.Command("say ...")` / `mob.Command("emote ...")` only. Later phases add an allowlist of mob commands, validated per call. |
| R3 perception parity | `perception.go` reads only the companion's own room, skips hidden characters and secret exits, and reports condition in words. Future structured inputs must pass the same filter. |
| R4 own knowledge | Mind file (memories, notes), later the explored-room map, owned by the module. |
| R5 DOGMud is truth | Mechanical state is never in the mind file. Gold, items, skills live on `CompanionInfo` and the live mob. |
| R6 never break character | `sanitizeDecision` drops lines that mention AI, models, prompts; fallback lines are authored in character; errors are never spoken. |
| R7 fail safe | Model calls run on a goroutine outside the mud lock; failures and budget exhaustion fall back to authored emotes. The module is off by default. |

## 3. Engine touch list (all phases)

Kept deliberately short. Everything else is new files.

| File | Phase | Change |
|---|---|---|
| `internal/characters/companions.go` | 0 | `CompanionBonded` source type; `Gold` field on `CompanionInfo` |
| `internal/hooks/PlayerDespawn_HandleLeave.go` | 0 | save bonded companion gold |
| `internal/hooks/PlayerSpawn_HandleJoin.go` | 0 | restore bonded companion gold |
| `internal/hooks/companion_reserve_backfill.go` | 0 | bonded companions hold no Conviction reserve |
| `internal/hooks/MobDeath_CompanionCleanup.go` | 0 | bonded companions fall instead of being deleted |
| `internal/hooks/companion_bonded.go` (new) | 0, 4 | fall handling, progression snapshot, mid-session respawn; rejoin through companion transport (phase 4) |
| `internal/hooks/hooks.go` | 0, 4 | installs the respawner and the rejoiner into `companionai` |
| `internal/companionai/` (new) | 0 | the engine to module seam (nil-safe hooks) |
| `internal/usercommands/ask.go` | 1 | `companionai.RouteAsk` before the companion refusal |
| `internal/usercommands/dismiss.go` | 0 | refuse to dismiss a bonded companion |
| `internal/usercommands/companion.go` | 0 | refuse to rename a bonded companion |
| `pool_mutation_guard_test.go` | 0 | exemption for `companion_bonded.go` (respawn sets pools to half) |
| `modules/all-modules.go`, `modules/context.md` | 1 | module import and index row |
| `internal/usercommands/emote.go` + `internal/events/eventtypes.go` | 2 | `events.Emote`, fired after a player emote, so companions can react to emotes |
| `internal/hooks/spell_resolution.go` + `internal/events/eventtypes.go` | 2 | `events.Healed`, fired when a player casts a healing spell on a mob, so a companion knows who tended it |
| `internal/hooks/MobIdle_HandleIdleMobs.go` | 3 | `companionai.RouteIdle` early return after the sleeping check, so default idle behaviour (floor-loot grabs, behaviour-tree idle, idle emotes) does not fight the module |
| (none in `internal/mobcommands/`) | 5 | The two missing verbs, looting a body and taking from a container, are mob commands registered by the module through `plugins.AddMobCommand` (`companion-loot`, `companion-takeout`), so no engine file changes. Shop browsing is a perception verb in the module. |

## 4. Module architecture (`modules/aicompanion`)

```
NewRound ─► sync()  attach/detach controllers, fall + recovery, session greeting
            dispatchAll() ─► dispatch(c)  [under mud lock]
                               build perception + prompt (plain data)
                               go callModel()  [no lock, no game pointers]
                                    └─► util.LockMud(); applyResult(); unlock
                                              sanitize ─► mob.Command(say/emote)
                                              update mind (mood, notes, lines)
Communication(say) ─► record line; if addressed ─► push stimulus
ask <companion> ─► companionai.RouteAsk ─► handleAsk ─► push stimulus
PlayerDespawn ─► detach (save mind)
plugin OnSave ─► save dirty minds (durable, autosave-queued)
```

- One `controller` per online owner; one decision in flight per companion;
  `seq` drops stale replies (logout, fall, pause, newer call).
- Rate limits: `MinSecondsBetweenCalls` per companion, `DailyTokenBudget`
  across the server, bounded pending queue (6).
- The response is a strict JSON schema (`decision.go`); `sanitizeDecision`
  enforces the limits the schema cannot (lengths, line count, `;`, control
  characters, name prefixes on emotes, character breaks).

## 5. Phases

Each phase is one commit on the `aicompanion` branch and adds to the single
cumulative patch. Every phase ends with the verification gate in section 6.

### Phase 0: Bonded companion kind (engine) (written)

Spec: F1.1, F1.2, F1.7, F1.9, F14.1.

- `CompanionBonded` (`"bonded"`; the companion list prints this value, so it
  names the relationship, not the mechanism) and persisted `Gold`.
- No Conviction reserve; cannot be dismissed or renamed.
- Death: progression snapshotted from the live mob; items, equipment and gold
  cleared from the record because the loot path puts all of them in the
  corpse (`itemdropchance: 0` means the default 100%, not 0%). Keeping them
  would duplicate gear. `PermaGear` keeps everything, matching the loot rule.
- `RespawnBondedCompanion` for mid-session recovery at half pools.
- First template, `9800-mara_venn.yaml`, deliberately with no gear or gold.

Acceptance: a granted bonded companion appears at login, leaves at logout,
keeps gold across sessions, refuses dismiss/rename, and after dying appears
again (hurt) without duplicating the corpse's contents.

### Phase 1: Present and talking (written)

Spec: F1.5, F1.8, F2.1, F2.3, F2.10, F2.11 (partial), F2.14, F2.17, F3.1,
F3.2 (notes), F4.1, F4.2, F4.3, F6.1 (basic), F6.2, F13.1, F13.2, F13.7,
F13.9, F15.x (budget), F17.x (admin).

- Module skeleton, config (off by default), embedded profiles, world primer.
- Mind file: recent lines, long-term notes, mood, session count, first met,
  last seen, deaths.
- Listeners: `NewRound`, `PlayerDespawn`, `Communication` (say).
- `ask <companion> <text>` routed to the module.
- Greeting at session start (first meeting vs returning, with elapsed time),
  recovery line after a fall, and a farewell during the `quit` meditation
  (condition 0), the only window in which the companion still exists to say
  it.
- Perception includes what is lying in the room (floor items, bodies,
  unhidden containers, loose coin), not only people and exits.
- Mood fades back to calm after `MoodDecayMinutes` without renewal.
- Decision trace: one log line per decision with trigger kinds, the
  model's private intent, lines spoken, tokens and latency. Player text is
  never logged.
- OpenAI chat completions with strict JSON schema; fallback to authored
  emotes when there is no key, no model, an error, or no budget.
- Admin `aicompanion status|profiles|grant|revoke|pause|resume`.
- A small authored "thinking" gesture when a reply to someone speaking to
  the companion takes longer than `ThinkingSeconds` (F2.11).
- Unit tests for config, profile loading, sanitizing, parsing, addressing,
  prompt quoting and mind caps.

Acceptance: with the module enabled and a key set, a granted companion
greets its owner at login, answers `say` and `ask`, stays in character when
provoked, remembers notes across restarts, and keeps talking in fallback mode
when the key is removed. `aicompanion status` shows tokens and errors.

Known limitation: a player who disconnects without `quit` gets no
farewell; there is no moment in which the companion could say it.

### Phase 2: Remembers and cares (written)

Spec: sections 2, 3 and 5; F1.6; F4.3.

- Mind schema 2, migrated from schema 1 on load: episodic memories
  (importance, emotion, people, place), facts about the owner (source and
  confidence), promises (by whom, open/kept/broken), session summaries,
  opinion of the owner (trust, respect, affection, -100..100) and an audit
  log of every opinion change.
- Retrieval: each decision gets the `PromptMemories` highest-scoring
  memories by importance, recency and relevance (shared words, the same
  place, the same people). Reflections and the last two session summaries
  are always shown. Old unimportant memories are shown as vague; the store
  forgets the least important first and never forgets importance 8+.
- Decision schema 2: memory (with importance and emotion), facts, opinion
  proposal with reason, promise made/kept/broken.
- Opinion is bounded in code per trigger, scaled by the profile's
  sensitivity, with diminishing returns on repeated warm words, and deeds
  applied as rules whatever the model says (see the module context.md).
  Only the owner's own actions move the owner opinion. The model sees the
  opinion only as words.
- Gradual self-disclosure: backstory entries are withheld from the model
  until trust reaches their threshold; the model knows only that there are
  things it keeps private.
- Curiosity: the profile lists what the companion wants to learn; known
  facts are shown so it does not re-ask.
- Reactions to emotes (new `events.Emote`), gifts (`GiftAccepted`), attacks
  (`PlayerAttackedMob`, deduplicated per attacker), healing spells (new
  `events.Healed`; a small rule-based gain when the owner heals it, three a
  day at most) and the owner's party changing (polled each round).
- Avoiding repetition: the companion's own last twelve lines are kept
  across sessions and shown to the model as things not to repeat.
- Initiative: after `InitiativeMinutes` of quiet, with the owner present and
  nobody fighting, a roll against the profile's talkativeness may give the
  companion a chance to speak up; the model may still stay silent.
- End-of-session reflection after logout: a short private summary, up to two
  conclusions, missed facts. Stored even though the owner is gone.
- Admin `aicompanion mind <character>` shows scores, memories, facts,
  promises and recent opinion changes.

Acceptance: over several sessions the companion recalls shared events when
relevant, asks about things it does not know and not about things it does,
opens up as trust grows, reacts to being hit, praised or given things, and
the admin view shows opinion moving by small, explained amounts.

Not in phase 2: embeddings for retrieval (keyword relevance only), merging
old session summaries into longer periods, opinions of NPCs and places
(phase 3), physical needs such as hunger feeding mood (phase 5).

### Phase 3: Notices and interacts (written)

Spec: sections 6 and 7; F4.5; F3.9; F5.9; F14.2, F14.5, F14.6.

- The module owns a bonded companion's idle tick (`companionai.RouteIdle`),
  keeping charmed first aid (`lookforaid`) and adding authored idle
  gestures (`idle_emotes`) chosen locally.
- Scene model: loose items, coin, bodies, unhidden containers, room
  fixtures (nouns), NPCs classified (fighting, hostile, merchant, local,
  someone's companion, stranger), other players, and the companion's own
  pack and gear, each with a ref. Scored locally by kind, the profile's
  `interests`, value, junk, hostility, and novelty from interaction memory.
- Noticing: walking into a room, or something notable appearing, produces a
  rate-limited "you notice" moment with the best few things. Quiet moments
  every `AutonomyMinutes` let it deal with something nearby.
- Actions: `look_at`, `consider`, `get`, `drop`, `give`, `show`, `put`
  (into a visible container), `sayto` (words to one person), `equip`,
  `remove`, `eat`, `drink`, `forage`, `search`. Perception verbs are answered
  by the module with what a player would read; the rest are single ordinary
  mob commands, validated against the scene and the live world, and judged
  two rounds later by what changed. Failures are remembered; a thing that
  failed twice is ignored for a day.
- What the owner is doing counts: while the owner is talking to someone
  else, the companion does not notice things aloud or act on its own; when
  the owner has been idle at the keyboard, quiet chances to act come twice
  as often.
- Loot arrangement agreed in conversation (`ask_first` default,
  `take_freely`, `leave_it`) and enforced in code.
- Impressions of NPCs (by template) and places (by room), with visit
  counts, shown back to the model as words.
- Mind schema 3 (migrated automatically).

Acceptance: walking into a room with something that suits her (a bow, a
hostile creature, loose coin), she notices it; with `ask_first` she points
things out and picks them up when told; she can look at and size up people
and things, equip better gear she was given, eat and drink from her pack,
and hand things to the owner, all visible to the room as ordinary actions.

Not in phase 3: looting bodies and containers, shop browsing (`list`),
learning from NPC dialogue (NPC dialogue answers players, not mobs), and
telling who owns an item lying on the floor. These move to phase 5 with the
mob wrappers they need.

### Phase 4: Knows its way around (written)

Spec: section 11; F7.10.

- Mind schema 4: the companion's own map. Every room it stands in is
  recorded with title, zone, visible exits (with doors), features (merchants,
  fixtures, containers), the people it saw there, visit counts and a danger
  score from fights (rate limited) and falls. The exit it just used is
  recorded as walked; the opposite exit of the new room is marked as a
  guessed way back until walked. Oldest rooms are forgotten beyond
  `MaxKnownRooms`.
- Route finding: Dijkstra over known, walked or guessed exits only, costed
  by distance, remembered danger, doors and guesses; exits that failed three
  times are skipped. `mapper.GetPath` is not used.
- New actions: `find_place` (search its own map and hearsay; the answer
  comes back as a follow-up), `go_to` a known place (`r123`) or `owner`,
  and `explore` one step through a way out it has never taken.
- Trips walk one `go <exit>` at a time, wait for arrival, record blocked
  exits, re-plan up to twice, and end with an "arrived" moment. An errand
  the owner asked for carries that permission, so a `get` on arrival passes
  the ask-first rule.
- An errand stops when the owner speaks to, emotes at, or asks the
  companion something; the walk back to the owner is never interrupted.
- Exploring outward: the prompt lists nearby known rooms that still have
  untaken exits, skipping rooms where it has had trouble (F11.8).
- Errands only from beside a non-fighting owner; the owner moving calls the
  companion back. Apart for `ErrandLingerRounds`, it walks back; with no
  known way for `LostRounds`, it rejoins through the engine's companion
  transport (`companionai.Rejoin`).
- Prompt: nearby known places with steps and staleness, unexplored exits
  here, hearsay, and the trip in progress. Hearsay (`place_tip`) is
  confirmed automatically when the companion reaches a room that matches it.

Acceptance: asked "where's the smith?", she remembers from her own map and
answers correctly with how far it is; asked to fetch something two rooms
away while the owner waits, she walks there by known exits, picks it up,
and walks back; she never walks toward a room she has not been to.

Not in phase 4: exploration journeys pursued as goals (phase 5), and
exploring through locked doors (no unlock yet).

### Phase 5: Purpose, skills and economy (written)

Spec: sections 8, 9, 10 and 12; the items moved here from phase 3.

- Archetype: primary and secondary role, weights over the game's sixteen
  real skills (validated at profile load against `internal/skills`), gear
  it favours and gear it will not use. Gear fit raises or lowers how much an
  item catches its eye. Its favoured skills are shown in the game's own
  rank words (novice, apprentice, journeyman...), never numbers, and a rank
  that changes during play is noticed and remembered.
- Inventory: supplies it likes to keep (by item type or name word, with a
  minimum and a target), carrying load in words, and protected items. A
  gift from the owner is protected: it will not be sold, dropped, stored or
  given to anyone but the owner. Protection is by item id and count, because
  item UUIDs are not saved.
- Shops: `browse` reads what every awake merchant in the room sells, priced
  exactly as `list` prices it, and remembers it. Wares then appear with
  `[s]` refs for half an hour. `buy` (with a quantity) and `sell` use the
  ordinary mob commands and are checked afterwards against purse and pack;
  what a shop paid is remembered. The prompt names the cheapest known place
  for anything it is short of.
- Money rules: never more than the purse; nothing from the emergency
  reserve, and nothing large for its personality (thrifty, ordinary, free),
  unless it meets a need or the owner asked.
- Bodies and containers: `loot` takes from a body exactly what the owner is
  entitled to (kill ownership and party loot mode); `take_from` takes a
  named item from an unhidden, unlocked container after looking inside. Both
  obey the loot arrangement.
- Goals: long-term ambitions seeded from the profile, restock goals created
  automatically from shortfalls, goals the model takes on or is given.
  Goals that can be checked (carrying enough, gold, a skill rank, having
  visited a room) are completed by code only; the rest by the model. A
  session agenda of up to three goals; stale goals pause after a week. A
  finished goal is remembered and gives her a moment to mention it.
- Autonomy level agreed in conversation (close, normal, free): close stops
  errands unless asked and halves quiet chances to act; free doubles them.
- Mind schema 5.

Acceptance: running low on arrows, she knows it, remembers where they are
cheapest, and on a quiet moment in town (or when asked) goes to buy them
within her means; she will not sell the owner's gift; she talks about her
goals and what she has achieved; she can loot a kill she and her owner
made, and take things from an open chest when the arrangement allows.

Not in phase 5: repairing gear (no item condition exists in the game to
track), banking (the bank belongs to player accounts), skill training (skills
grow only by use in DOGMud), haggling beyond what `buy` and `sell` already
apply, learning from NPC dialogue (NPCs answer players only), and knowing
who dropped an item on the floor.

### Phase 6: Fights well (written)

Spec: section 13.

- The engine's mob combat AI stays the reflex layer: it attacks, uses
  special moves and auto-assists the owner as before. No engine files
  change in this phase.
- A fight is tracked from the companion's side: who is fighting it or its
  owner (or being fought by them), the worst health each side reached,
  who fell, whether it ran.
- Plan (F13.1, F13.2): when a fight starts, when someone joins, or when the
  companion or its owner drops into a worse health band, the model is asked
  (at most every three rounds, jumping the queue, never blocking the fight)
  for a stance (fight, protect, hold_back, flee), a target by `[e]` ref,
  when to run, and melee or ranged. The prompt shows each enemy's condition,
  who it is fighting and the odds, in words.
- Reflexes (F13.4, F13.5): at most one ordinary command a round, never in
  the round something happened: `flee` at the chosen threshold; `drink` a
  potion when badly hurt; when the owner is in danger and the companion is
  willing (trust, affection and the profile's bravery, or a protect
  stance), `attack #id` or `fire #id` the thing hurting them, then `taunt`
  to draw it off; switch to the chosen target; `aid @owner` when the owner
  is down and the room is calm.
- Hold back (F2.15, F13.6): the companion can refuse to join a fight, for
  example one its owner started with a merchant or a child (the profile's
  `refuse` words). It switches the engine's auto-assist off for that fight
  and puts the owner's setting back afterwards; the setting is also kept in
  the mind so it is restored even if the session ends mid-fight.
- Battle lines (F13.7): authored lines for the start, being hurt, the owner
  being hurt, an enemy falling, victory and running, used locally with a
  cooldown and a chance, no model call.
- After the fight (F13.8): a summary is remembered (more important if
  anyone nearly died or it was long), the room is marked dangerous, and the
  companion gets a moment to check on its owner and say something. The
  owner's death is remembered as one of the most important things that can
  happen.

Acceptance: in a fight she keeps shooting, switches to whatever is hurting
her owner when she cares enough, draws it off, drinks a potion when badly
hurt, runs when she said she would, refuses to join an attack on a merchant,
speaks a few words at the right moments, and afterwards asks after her
owner and remembers the fight.

Not in phase 6: casting spells as part of a combat plan (the engine's own
combat AI already casts what the companion knows), and choosing special
moves (left to the engine's combat AI).

### Phase 7: Hardened (written)

Spec: sections 15 to 17 and 19, Part 4.

- Model tiers: fast (fights, noticing, quiet moments, follow-ups), main
  (conversation and relationship) and deep (reflection), each with its own
  model, token limit, timeout and optional reasoning effort, all falling
  back to `Model`. A batch with any conversation goes main; any fight goes
  fast.
- Resilience: one retry on transient failures (not for the fast tier), a
  circuit breaker after repeated errors, a per-companion daily token budget
  alongside the server budget.
- Output moderation (optional): speech and `sayto` lines are checked with
  the OpenAI moderation endpoint on the model goroutine; flagged lines are
  dropped, and if nothing is left an authored reply is used. A moderation
  failure never silences the companion.
- Richer context for replies: time of day and month, how long ago older
  lines were said, which travellers present it already knows, what the
  owner looks like and carries, what it was doing and meant to do, the
  rooms it has just walked through, and an explicit rule to answer in the
  speaker's language and to what was actually said.
- Asking the game: on the main tier the model may call read-only tools
  (`look_closer`, `size_up`, `check_wares`, `recall`, `find_place`) for up
  to `ToolRounds` rounds before it answers; answers are built from what a
  player could see, the companion's own memories and its own map.
- Visibility audit: hostility and item value are no longer used (players
  cannot see them); floor items are known by name until looked at; the
  companion's own senses (perception of hidden characters, night vision)
  decide what it sees.
- Mid-session snapshots: the bonded companion's gear, gold and skills are
  copied into the owner's record every `SnapshotRounds` rounds and after
  any trade or pickup (`companionai.Snapshot`, installed by hooks), so a
  restart does not lose a session's purchases and loot.
- Zero configuration: the module is on by default, reads `OPENAI_API_KEY`
  (or `APIKey` in the config), lists the models the key can use and picks
  per tier from built-in preference lists, skips any model the API refuses,
  and sets reasoning effort per model family (none when tools are offered).
- Meeting at character creation: a new character meets its companion in
  its first real room; an existing character with none meets one at its
  next login; the companion may part ways when clearly sent away (and then
  stays gone), or when trust and affection have collapsed.
- Mind backups: three rotating copies, written at the end of every
  `BackupEverySessions`-th session; a damaged mind is restored from the
  newest readable backup.
- Admin: `aicompanion trace` (recent decisions with tier, model, intent),
  `aicompanion prompt` (the full last request), `aicompanion models`
  (routing and per-tier metrics), `aicompanion breaker reset`; the status
  view shows an open breaker. Help templates for both commands.
- Owner: `companion-unstick` clears a stuck companion without touching its
  mind (F18.2).
- `testing-and-prompts.md`: what the model is told, the tier table, the test
  plan (restart, copyover, outage, soak) and the lifelikeness checklist.

### Review fixes (after an external code review)

- Parting ways is now a two-step: the model may ask, the owner confirms
  with `companion-part`, and an unconfirmed request lapses. Only a wholly
  collapsed relationship ends the bond without confirmation.
- Corpse gold a companion loots goes to the owner's party pool when they
  are in a party, as the player loot path does.
- Sight comes from the engine's `messaging.ParticipantSight`: blindness,
  light, night vision and shapes-only. Without full sight the companion is
  not told the room name, its contents, who is present or its owner's
  state, and it does not map the room.
- Every request carries a world revision; a reply or tool answer built on a
  stale picture is discarded whole.
- Model calls are cancelled on logout, pause, reset and death; token
  budgets are reserved before a call and settled after it.
- Information tools no longer emote or count as the companion's action.
- Item commands name the exact item instance (`!<itemId>:<uuid>`), so two
  identical items cannot be confused by the command parser.
- Facts carry the owner's own words as their source and are held at lower
  confidence without them; a promise the model judges kept or broken moves
  trust gently, where a deed the game saw moves it more.
- The capability list in the prompt is generated from what is enabled, so
  it can no longer contradict itself.
- Whether the owner's speech counts as spoken to the companion when nobody
  else is present is now a setting (`RespondWhenAlone`, on by default).
- Not adopted: defaulting the module and auto-bonding to off. On a private
  single-player server that would mean the feature does not exist until it
  is configured, and with no API key set nothing is called or spent. For an
  upstream release, set `Enabled` and `AutoBondExisting` to false.

## 6. Patch process

The uploaded zip (2026-09-22) is the base commit. Work happens on branch
`aicompanion`, one commit per phase.

There is exactly one deliverable: `DOGMud_AI_Companion.patch`, always the
full diff from the base (`git diff --full-index base..aicompanion`). Each
phase replaces the previous file with a larger one. Apply it to a clean
checkout of the base commit, never on top of an earlier version of itself.

Applying on a checkout at the same commit as the zip:

```
git checkout -b aicompanion
git apply --3way --whitespace=nowarn DOGMud_AI_Companion.patch
go generate ./...          # only if module-imports.go should regenerate all-modules.go
gofmt -w internal/companionai internal/hooks internal/usercommands modules/aicompanion  # the patch was hand-formatted
go vet ./internal/companionai ./internal/hooks ./internal/usercommands ./modules/aicompanion
go test . ./internal/... ./modules/...
```

The root package tests (`go test .`) run the repository guard tests. Those
most likely to react to this work: pool mutation (already registered),
lookup viewer (the module never looks creatures up by name), raw
`events.Message` (the module only uses `SendText` and room senders), durable
writes (the module only writes through the plugin store), messaging surface
(profiles are embedded, not under `_datafiles/world/dogmud`).

If upstream has moved, `--3way` resolves most drift because the patch's
blob ids match the real repository. Conflicts are most likely in `ask.go`,
`dismiss.go` and the two companion save/restore hooks; each change there is a
few lines and self-contained.

Errors from any step go back into the next patch revision; the cumulative
patch is always regenerated from the branch, never hand-edited.

## 7. Operating it

```yaml
# _datafiles/config.yaml (note the skip-worktree bit on this file)
Modules:
  aicompanion:
    Enabled: true
    Model: "<current OpenAI chat model>"
    DailyTokenBudget: 400000
```

```
export OPENAI_API_KEY=...        # never in a file
aicompanion profiles
aicompanion grant <character> mara
aicompanion status
```

## 8. Open questions

- Should a bonded companion cost the owner anything (gold wage, a quest to
  recruit) instead of being admin-granted? Phase 5 can add a hiring flow.
- Should death be softer (knocked out, keeps gear) rather than following the
  mob loot rules? Current choice is the fair one.
- Copyover: companions are saved and respawned by the existing login path;
  confirm a bonded companion's mind survives a copyover in a live test.
