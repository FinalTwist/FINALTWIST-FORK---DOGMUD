# Sight and gates parity: mobs get, look, remove, craft and speak by the player's rules

Date: 2026-09-29. Player/mob parity slice 5, split into **5a object and
action gates** (get, look, remove, craft) and **5b speech in the dark** (say,
shout, rally, warcry). Source: the owner-ordered parity audit (2026-09-28)
and the owner's decisions of 2026-09-29. Each sub-slice ships as its own PR,
5a first.

## Facts verified against source (2026-09-29, master `7d6d4ac38`)

Every row below was read at `7d6d4ac38` in a fresh worktree. Negative rows
("no callers", "zero") name the search and a positive hit that proves the
same search could match.

### Get

| # | Fact | Where |
|---|---|---|
| G1 | `usercommands.Get` refuses at `messaging.ParticipantSight(user.Character, room) == messaging.SightNone` ("You can't see anything to pick up!") as its first statement, so the refusal covers every branch: floor, gold, stash, containers, corpses, component bag and bandolier | `internal/usercommands/get.go:94-97` |
| G2 | Single get peeks `room.FindOnFloor(rest, getFromStash)` and refuses an `exploding` item ("You can't pick that up, it's about to explode!") BEFORE calling `actions.GetItemFromFloor`, so the exploding check runs before the household-bauble check | `get.go:635-640,642` |
| G3 | The filtered sweep `getAllMatchingFromFloor` (for `get all <name>` and `get all.<name>`) does NOT call `GetItemFromFloor`: it loops `takeableOnFloor`, stops silently on an `exploding` item, and moves each item with `StoreItem` plus its own `ItemOwnership` event | `get.go:30-87` (exploding at `:52`); callers `:222,:251` |
| G4 | Unfiltered `get all` recurses `Get(item.Name(), ...)` per item, so each item meets G2 | `get.go:226-242` |
| G5 | `actions.GetItemFromFloor(actor Actor, itemName string, stash bool) GetItemResult` gates only the household bauble (`ErrHouseholdBauble`), then `TransferItemToBackpack`, which fires the same `ItemOwnership` event G3 fires by hand. No sight gate, no exploding gate | `internal/actions/get.go:27-53`; `internal/actions/transfer.go:44-64` |
| G6 | `actions.GetGoldFromFloor(actor, amount)` has no sight gate; the player reaches it only past G1 | `actions/get.go:57-59`; `usercommands/get.go:610` |
| G7 | `mobcommands.Get` supports `all`, `gold`, floor and stash, with no sight and no exploding gate | `internal/mobcommands/get.go:16-95` |
| G8 | The only Go issuer of a mob `get` is the AI companion (`get gold`, `get <item ref>`). Search `(Command\|issue)\(...get` over `internal/` and `modules/` also hit `actions.go:373`; mob YAML `idlecommands` carry no `- get` (the same pattern finds 415 `- say` lines) | `modules/aicompanion/actions.go:359-375` |
| G9 | The companion lists floor items and gold only when `!sc.Dark`, and `sc.Dark = cannotSee(mob, room)`, which is `sightOf != SightFull`. So it never issues a `get` for a floor item it does not see clearly | `modules/aicompanion/scene.go:134,141-168`; `perception.go:304-316` |

### Look

| # | Fact | Where |
|---|---|---|
| L1 | `usercommands.Look` refuses at `SightNone` ("You can't see anything!") before anything else and keeps the verdict in `sight` | `internal/usercommands/look.go:33-37` |
| L2 | It resolves a creature with `ResolveTargetOptions{Viewer: user.Character}` and only acts on it at `sight == SightFull`; a pet likewise needs `SightFull` (`:446`); at `SightShapes` an unmatched name ends in "You can only make out shapes here." (`:576-579`) | `look.go:88-89,446,576` |
| L3 | Looking through an exit refuses when `!messaging.SeesThroughExit(user.Character, room)` ("It's too dark to see anything in that direction."), then refuses a locked exit | `look.go:285-294` |
| L4 | Player resolution order: sight gate, `events.Looking`, no target, creature, sealed crate, container, exit, direction alias, carried-item noun, carried item, room noun, pet, shapes hint | `look.go:33-579` |
| L5 | `mobcommands.Look` has no sight gate at all. Order: no target, exit (locked refuses silently; no through-sight check), backpack, creature, body, room noun, pet | `internal/mobcommands/look.go:14-170` |
| L6 | It resolves the creature with a bare `actions.ResolveTargetActor(room, lookAt)` (no viewer), so it resolves a hidden player, tells them "X is looking at you." and tells the room "X is looking at <hidden player>." through plain `SendTextVisual` | `mobcommands/look.go:92,95-107` |
| L7 | `ResolveTargetOptions.Viewer`: "a creature it does not perceive ... cannot be named". Mob callers that already pass their own character: `FindAttackTarget(..., viewer)` and `castViewer(actor)`, which returns the mob's character since slice F | `internal/actions/target_resolution.go:23-27,79-90`; `combat_attack.go:32,119`; `cast_sight.go:11-13` |
| L8 | The repo-root lookup guard registers `internal/mobcommands/look.go\|Look` as `{plain: 1, why: whyMob}` and `internal/usercommands/look.go\|Look` as `{viewer: 1}`; it fails on a stale entry | `lookup_viewer_guard_test.go:49,71,233-236` |
| L9 | No production caller issues a mob `look`. Searched `Command\(...look` in `internal/`, `modules/` and `_datafiles/**/*.js` (the same search finds `mob.Command(\`lookforaid\`)` at `hooks/MobIdle_HandleIdleMobs.go:276`), `- look` in mob and schedule YAML (0; the pattern finds 415 `- say`), and behaviour `cmds:` (0). The companion's `look_at` emits its own emote and never issues `look` | grep; `modules/aicompanion/actions.go:353-354,625-655` |

### Remove

| # | Fact | Where |
|---|---|---|
| R1 | `usercommands.Remove` first calls `refuseWhileBusy(user, "change equipment")`, which refuses on `Character.IsActing()` | `internal/usercommands/remove.go:17-19`; `busy_refuse.go:18-25` |
| R2 | Single remove refuses a cursed item when `matchItem.IsCursed() && user.Character.Health > 0` and `GetSkillLevel(skills.Spellcasting) < 4`; at 4 or above it prints "It's CURSED but luckily your enchant skill level allows you to remove it." and proceeds | `remove.go:63-77` |
| R3 | `remove all` loops `Equipment.GetAllItems()` into `actions.RemoveEquipment` with no cursed check, so it strips cursed gear. It prints no per-item line | `remove.go:30-54` |
| R4 | `actions.RemoveEquipment(actor Actor, itemName string) RemoveEquipResult`; its doc says "Cursed-item checks, messaging, and all-remove loops remain in the callers". It already fires one `EquipmentChange` per item | `internal/actions/remove_equip.go:94-131` |
| R5 | `mobcommands.Remove` refuses under `PermaGear` (an emote), then runs the same `all` loop or a single remove. No busy gate, no cursed gate | `internal/mobcommands/remove.go:15-50` |
| R6 | Both `remove all` wrappers queue an aggregate `EquipmentChange` on top of R4's per-item events. The only readers of `ItemsRemoved` are the GMCP refresh and the light recompute, so the duplicate is harmless | `usercommands/remove.go:39-42`; `mobcommands/remove.go:33-36`; `modules/gmcp/gmcp.Char.go:249`; `hooks/Awareness_LightChange.go:115` |
| R7 | `RemoveEquipment` callers: the two wrappers only. The companion issues `remove <worn ref>` | grep `RemoveEquipment(`; `modules/aicompanion/actions.go:408-412` |
| R8 | The companion judges a remove by whether its worn count changed; a refused remove records "You tried to change your gear (X), but nothing changed." | `modules/aicompanion/actions.go:791-799` |

### Craft

| # | Fact | Where |
|---|---|---|
| C1 | `usercommands.Craft` refuses an attempt when `!messaging.CanSeeClearly(user.Character, room)` ("You can't see well enough to work on anything here."), after `craft`/`craft list` and BEFORE the storage pull and the enchanting dispatch | `internal/usercommands/craft.go:83-95,113-120` |
| C2 | Enchanting never reaches `actions.InitiateCraft`; storage pulls and `storageAwareMissingTag` stay in the command layer because storage hangs off the user record | `craft.go:113-124,713-715`; `internal/actions/craft.go:124-130` |
| C3 | `actions.InitiateCraft(actor Actor, recipeName string) CraftResult` has no sight gate; its first gate is `IsCrafting` | `actions/craft.go:131-138` |
| C4 | Callers: the two wrappers only. Mob `craft` is issued by the goal planners (`craft_item.go:70`, `mastery_skill.go:49`) and the companion (`actions.go:314-319`, `autonomy.go:369-375`). The shop crafter restocks shelves directly and never issues `craft` | grep `InitiateCraft(`, `"craft "`; `internal/mobs/crafter.go:250-300` |
| C5 | The companion does NOT skip crafting in the dark: `craftableHere` checks recipe, station and ingredients only; autonomy offers `craft` when `craftableHere` is non-empty (`:301`); the prompt lists the same (`runtime.go:521`) | `modules/aicompanion/cooking.go:58-88`; `autonomy.go:301-303,369-375` |
| C6 | A companion craft that did nothing is recorded as "You set about making X." (the not-ok branch assumes a multi-round craft) | `modules/aicompanion/actions.go:871-877` |
| C7 | `CanSeeClearly` = awake and `ParticipantSight == SightFull`; `ParticipantSight` returns `SightNone` for a blinded observer before reading light | `internal/messaging/predicates.go:56-62,136-138` |

### Speech

| # | Fact | Where |
|---|---|---|
| S1 | `actions.Say(actor, text)` reveals a hidden speaker, echoes "You hear someone talking." to exits (quiet), queues `events.Communication`, and returns `IsSneaking`. The room line is left to callers | `internal/actions/say.go:18-48` |
| S2 | `TransitionToRevealing` passes through `Revealing` to `Visible` in the same call, so after S1 `IsHidden()` is false and every "someone says/shouts" hidden branch is unreachable unless the transition errors | `internal/state/awareness/awareness.go:190-214` |
| S3 | Player say sends `FormatSayText(name, ...)` through `room.SendTextCommunication`, with no darkness handling: a player's name reaches every listener in any light | `internal/usercommands/say.go:32-39` |
| S4 | Player shout reveals, uppercases, drunkifies, escapes, then sends the named line through `SendTextCommunication`; adjacent rooms get `Someone shouts from the <exit> direction, "<WORDS>"` as a communication; it wakes sleepers except the shouter | `internal/usercommands/shout.go:25-85` |
| S5 | Mob say returns early when `room.PlayerCt() < 1` (so no `Communication` event and no exit echo), then calls `actions.Say` and `sendAudioRoomText` | `internal/mobcommands/say.go:15-30` |
| S6 | Mob shout does NOT reveal a hidden mob; it sends through `sendAudioRoomText`; adjacent rooms get `Someone is shouting from the <exit> direction.` with NO words, through plain `SendText`; it wakes sleepers except itself | `internal/mobcommands/shout.go:14-58` |
| S7 | `sendAudioRoomText` is TWO-tier, not three: in a lit room (`room.IsLit()`) it sends the named line to everyone, a blinded listener included; in the dark `SightFull` hears the name and everyone else the anonymous string | `internal/mobcommands/darkness.go:25-52` (lit shortcut `:30-32`) |
| S8 | `sendAudioRoomTextHidingNames` is the three-tier one: per listener, `messaging.HideNames(text, names, ParticipantSight(...))`, delivered with `u.SendText` (audio, no deafen filter) | `darkness.go:66-82` |
| S9 | `HideNames` writes "a figure" at `SightShapes` and "something" (capitalised at sentence start) at `SightNone`, not "someone" | `internal/messaging/hidenames.go:30-35,55-72` |
| S10 | `sendAudioRoomText` callers: `say.go:29`, `shout.go:24`, `rally.go:25`, `warcry.go:25`, `howl.go:47,68`, `taunt.go:54,75,86,99` (10). `sendAudioRoomTextHidingNames` callers: `howl.go:58,79`, `taunt.go:153,189` (4). All in `internal/mobcommands` | grep |
| S11 | Player rally and warcry send their room line through `SendTextVisual` (sight-gated: a listener who cannot see gets nothing); mob rally and warcry use `sendAudioRoomText`. Player taunt is visual too | `internal/usercommands/rally.go:42,73`; `warcry.go:44,77`; `taunt.go:165,192,225`; `mobcommands/rally.go:25`; `warcry.go:25` |
| S12 | `Room.SendTextCommunication` queues one RoomId-keyed `events.Message{IsCommunication: true}` with no category. Its doc: "deafen mutes player chatter only. NPC/merchant speech must NOT use this; it goes through SendText / SendTextVisual unfiltered so moderated players still hear quest content. Audited 2026-07-10." | `internal/rooms/rooms.go:218-236` |
| S13 | The deafen filter is on BOTH branches of the listener: per-user (`:29`) and room (`:83`). `Deafened` is the admin moderation flag ("Cannot HEAR custom communications from anyone but admin/mods"). `IsQuiet` lines pass only to a `SuperHearing` listener. The hook's comment says `IsQuiet` has zero emitters; that is stale, since `SendTextToExits(txt, true)` sets it for say's exit echo. But no dogmud condition grants `superhearing` (the same search finds `world/default/conditions/28-superior_hearing.yaml`), so those lines reach nobody, and speech lines never set `IsQuiet` | `internal/hooks/Message_SendMessages.go:29,44-52,83-92`; `internal/rooms/rooms.go:516-543`; `internal/users/userrecord.go:50`; `usercommands/admin.deafen.go:31` |
| S14 | Mob `say` carries quest dialogue (`npc_say` builds `say <text>`), shopkeeper replies, dialogue trees and the companion's speech | `internal/questengine/bridge.go:415-420,437,439`; `internal/actions/buy.go:169`; `internal/behaviortree/actions_dialogue.go:69,128`; `modules/aicompanion/runtime.go:1115-1123` |
| S15 | `Actor.SendRoomCommunication(msg, excludeSelf)` already encodes a split: `UserActor` sends a communication, `MobActor` sends `SendTextVisual` ("Mobs do not respect client-side mute/deafen settings"). It has no production caller (grep finds only the interface, the two methods and nine test fakes) | `internal/actions/actor.go:27-31`; `actor_user.go:46-52`; `actor_mob.go:45-51` |
| S16 | Mob shout is live: the human species' `angrycommands` hold three `shout` lines. No Go code and no mob YAML `idlecommands` issue `shout` | `_datafiles/world/dogmud/species/1-human.yaml:10-13`; grep |
| S17 | Mob rally and warcry are issued by behaviour archetypes (`leader`, `guard_captain`, `tank_taunter`, `boss_soren`) and the companion's combat moves | `_datafiles/world/dogmud/behaviors/archetypes/*.yaml`; `modules/aicompanion/decision.go:53` |
| S18 | Whisper is a remote, name-addressed tell (`users.GetByCharacterName`) with its own deafen refusal; it is not room-bound, so darkness does not apply. Mob `sayto`/`replyto` send the mob's and the target's names to the room with plain `SendText` and no sight gate, so they leak names in the dark | `internal/usercommands/whisper.go:31-60`; `internal/mobcommands/sayto.go:36-51,65-68,142-158` |

### Guards and tests that key on this code

| # | Item | Where |
|---|---|---|
| T1 | Re-fork guard pattern to follow: a regex over the wrapper files | `drink_wrapper_guard_test.go`; `flee_wrapper_guard_test.go` |
| T2 | Lookup guard entries L8 go stale when look resolution moves | `lookup_viewer_guard_test.go:49,71` |
| T3 | `messaging_surface_guard_test.go` matches EXACT room-send method names for the observer viewpoint; a new Room sender must be added | `messaging_surface_guard_test.go:896-897` |
| T4 | `bauble_finder_view_guard_test.go` lists beyond-reader senders by name, `SendRoomCommunication` among them | `bauble_finder_view_guard_test.go:95-98` |
| T5 | `m2_routing_guard_test.go` still recognises `sendAudioRoomText` calls although `m2RoutingFiles` is empty; `send_trio_only_guard_test.go` names it in comments | `m2_routing_guard_test.go:70,261`; `send_trio_only_guard_test.go:20,45` |
| T6 | Say and shout tests assert only `handled`/`err`: `usercommands_test.go:610,617,663-680,1271,7179`; `mobcommands_test.go:385,391,408-414,1219`. `FormatSayText` text is pinned by `internal/actions/actions_test.go:67-126`. `internal/mobcommands/audio_room_text_sight_test.go` pins `sendAudioRoomText` | grep |
| T7 | 5a tests that touch the moved gates: `darkness_gates_sight_test.go` (calls `Get("", ...)` and `Look` in the dark), `look_exit_visibility_test.go`, `shapes_roster_test.go`, `household_bauble_test.go:96` (the sweep), `actions/get_household_test.go`, `actions/economy_test.go:279-340`, `usercommands_test.go:1097` (`TestRemove`), `mobcommands_test.go:430,604` (`TestLook`, `TestRemoveMob`), `remove_reservation_disclosure_test.go` | grep |

## Owner decisions (binding, 2026-09-29)

1. **One spec, two PRs.** 5a object and action gates first, then 5b speech.
2. **Approach A.** Every rule moves into the shared body both actors already
   call; the command wrappers keep only their wording. A repo-root re-fork
   guard pins it. No per-wrapper copies, no rule-table mechanism.
3. **Dark speech is three-tier for player and mob speakers.** Clear sight
   hears the name, shapes hears "A figure", no sight hears "Someone". The
   words are always heard.
4. **Mob remove gets the busy gate** (the player's `refuseWhileBusy`).
5. **Riders:** shout reveals a hidden mob speaker as it does a player; the
   Spellcasting-4 cursed-removal exception applies to mobs; the mob-only
   `PermaGear` refusal stays.
6. **NPC speech stays unfiltered by deafen** (owner, 2026-09-29, reversing the
   first draft's rider). Deafen is the upstream child-safety tool: it shields a
   player from other players' free text, and NPC lines are authored content,
   so a deafened player keeps hearing quest givers, merchants, dialogue and
   their companion (S12, S14). Mob speech takes the three-tier names through
   `SendTextHidingNames`; player speech keeps the deafen filter through
   `SendCommunicationHidingNames`. That one call is the only difference.

## 5a: Object and action gates

**Shape.** Where a rule guards a branch only the player has (containers,
corpses, storage pulls, enchanting), the wrapper still needs the refusal
before that branch runs. So each gate is ONE exported predicate in
`internal/actions`, called by the shared body for both actors and, where a
player-only branch runs first, by the player wrapper too. The wrapper never
evaluates sight, curses or busyness itself; it asks the shared predicate and
words the answer.

### Get

- `actions.TooDarkToGet(actor Actor) bool`: `ParticipantSight == SightNone`
  (G1, unchanged predicate; shapes are enough to grope).
- `actions.TakeFloorItem(actor Actor, item items.Item, stash bool) error`: the
  gated transfer of an item already found. Order, matching the player today
  (G1, G2, G5): `ErrTooDark`, `ErrExploding`, `ErrHouseholdBauble`, then
  `TransferItemToBackpack`.
- `GetItemFromFloor` becomes find plus `TakeFloorItem`. `ErrTooDark` returns
  `Found: false` (the actor learns nothing about the floor). `GetGoldFromFloor`
  refuses with `ErrTooDark` too (G6).
- The sweep (G3) finds with `takeableOnFloor` as now and moves each item
  through `TakeFloorItem`; `ErrExploding` stops the sweep silently, as today.
  Its hand-built `StoreItem` and `ItemOwnership` go (G5 fires the same event).
- `usercommands.Get` keeps its first-statement refusal by calling
  `TooDarkToGet`, so containers, corpses and the bags stay refused in the
  dark. It maps `ErrExploding` to today's line. Single get drops its peek.
- `mobcommands.Get`: silent on every refusal.

**Companion.** No change in behaviour (G9): it never issues a floor `get` it
cannot see clearly. A companion that issued a `get` just before the room went
dark now records "did not manage it" (G8, the existing not-ok line).

### Look

- New `actions.ResolveLook(actor Actor, lookAt string) LookResolution` holds
  every sight rule of both looks: the `SightNone` refusal (L1), the creature
  resolved with `Viewer: actor.GetCharacter()` and acted on only at
  `SightFull` (L2, L7), the exit with `SeesThroughExit` and the lock (L3), and
  the pet at `SightFull`. `LookResolution` carries `Sight`, a kind
  (`LookDark`, `LookRoom`, `LookCreature`, `LookExit`, `LookExitTooDark`,
  `LookExitLocked`, `LookPet`, `LookOther`), the target `Actor`, the exit
  name and room id, and `NamesCreatures bool` for the wrappers' shapes hint.
- Resolution order is the player's (L4): creature before exit. The mob's
  backpack-before-creature order (L5) goes; with no callers (L9) nothing
  observable depends on it.
- `usercommands.Look`: calls `ResolveLook` and renders each kind with
  today's lines; its sealed-crate, container, noun and item branches run on
  `LookOther` exactly as now. `events.Looking` stays player-side.
- `mobcommands.Look`: calls `ResolveLook`; silent on `LookDark`,
  `LookExitTooDark` and `LookExitLocked`. Its room lines keep their wording
  and move to `SendTextVisualHidingNames` with the same names the player's
  lines hide (looker, and the looked-at player).
- Closes L6: a mob can no longer name a hidden player, tell them it is
  looking, or tell the room.
- The lookup guard (T2): both `Look` entries are removed and
  `internal/actions/look.go|ResolveLook` is registered `{viewer: 1}`.

### Remove

- `actions.CursedHolds(char *characters.Character, item items.Item) (holds,
  overridden bool)`: `IsCursed() && Health > 0`, overridden when
  `GetSkillLevel(skills.Spellcasting) >= 4` (R2). The one statement of the
  rule; the companion calls it too.
- `RemoveEquipment` gains, in order: `Busy` when `IsActing()` (R1), then
  `Cursed` when the curse holds, else `CursedOverridden` set and the item
  comes off. Its doc comment (R4) is corrected.
- New `actions.RemoveAllEquipment(actor Actor) RemoveAllResult`: the busy
  gate once, then each worn item through the same per-item gates. A cursed
  item is skipped and listed in `Cursed`; the rest come off. The two copied
  loops (R3, R5) go, and so do the wrappers' aggregate `EquipmentChange`
  events (R6), since the shared body already fires one per item.
- `usercommands.Remove`: renders `Busy` with `refuseWhileBusy`'s text for
  "change equipment" (the text moves to a helper both use; the wrapper no
  longer calls `IsActing`), `Cursed` with today's cursed line, and
  `CursedOverridden` with today's "luckily" line. In `remove all` each skipped
  item gets the cursed line; the overridden ones come off as silently as every
  other item there.
- `mobcommands.Remove`: `PermaGear` stays first (ruling 5); silent on `Busy`
  and `Cursed`.
- Companion: the `remove` verb refuses up front when `CursedHolds` says the
  item holds ("it will not come off"), the way `get` refuses a household
  bauble at `actions.go:366-368`, so it does not record a futile attempt (R8).

### Craft

- `actions.TooDarkToCraft(actor Actor) bool`: `!CanSeeClearly` (C1, C7).
- `InitiateCraft` checks it first and returns `CraftResult{CannotSee: true}`.
- `usercommands.Craft` keeps its refusal where it is, before the storage pull
  and enchanting (C1, C2), by calling `TooDarkToCraft`; storage, quest notify
  and enchanting stay player-side as documented (C2).
- `mobcommands.Craft`: `CannotSee` is a silent no-op like its other refusals.
- Companion (C5, C6): `craftableHere` returns nothing when `cannotSee(mob,
  room)`, so autonomy never picks `craft` in the dark and the prompt offers no
  recipe there. Without this the companion would record "You set about making
  X." for a craft that never started.
- Planners (C4): a planner crafter at a station in the dark issues `craft` and
  is refused silently, then retries at its own cadence (slice 4, R3). No hot
  loop; the mob waits for light.

### Guard (5a)

Repo-root `sight_gates_wrapper_guard_test.go`, the `drink_wrapper_guard`
shape, fails if any of these files matches its forbidden set:

| Files (both actors) | Forbidden |
|---|---|
| `usercommands/get.go`, `mobcommands/get.go` | `ParticipantSight`, `CanSeeShapes`, `CanSeeClearly`, `` `exploding` `` |
| `usercommands/look.go`, `mobcommands/look.go` | `ParticipantSight`, `SeesThroughExit`, `CanSeeClearly`, `ResolveTargetActor` |
| `usercommands/remove.go`, `mobcommands/remove.go` | `IsCursed`, `IsActing`, `refuseWhileBusy`, `Spellcasting` |
| `usercommands/craft.go`, `mobcommands/craft.go` | `CanSeeClearly`, `ParticipantSight` |

The guard matches code, not comments: it drops `//` comment text before
matching, because `get.go:28`, `get.go:635`, `look.go:284` and `craft.go:88`
name these words in comments today. Outside the moved gates the files use
none of them in code (grep: `ParticipantSight` at `look.go:33` only,
`SeesThroughExit` at `look.go:285` only). Proven able to fail by a temporary
violation in each row.

### Parity table (5a)

| Rule | Player today | Mob today | Both after |
|---|---|---|---|
| Get refused at no sight | yes | no | yes |
| Gold get refused at no sight | yes | no | yes |
| Exploding item refused (single) | yes | no | yes |
| Exploding item stops a sweep | yes | no sweep | yes (player sweep through shared body) |
| Household bauble refused | yes | yes | yes |
| Look refused at no sight | yes | no | yes (mob silent) |
| Creature named only when perceived | yes | no (hidden player named) | yes |
| Creature named only at clear sight | yes | no | yes |
| Pet named only at clear sight | yes | no | yes |
| Look through an exit needs `SeesThroughExit` | yes | no | yes |
| Remove refused while busy | yes | no | yes |
| Cursed item holds (alive, Spellcasting < 4) | single only | no | single and `all` |
| Spellcasting 4 removes a cursed item | yes | n/a | yes |
| `remove all` skips cursed, removes the rest | no (strips all) | no | yes |
| `PermaGear` refuses | n/a | yes | unchanged (mob only) |
| Craft refused below clear sight | yes | no | yes |

## 5b: Speech in the dark

**Shape.** Two Room senders beside `SendTextVisualHidingNames`, both
three-tier through a hiding function, per listener, never shortcutting on a
lit room (which is what leaks S7 to a blinded listener):

- `Room.SendCommunicationHidingNames(cat messaging.Category, text string,
  names []string, excludeUserIds ...int)`: per listener, hides `names` by
  that listener's `ParticipantSight`, renders through `RenderForRecipient` on
  the audio channel (so the category's colour and wrap apply, as mob speech
  gets today), and queues a per-user `events.Message{IsCommunication: true}`,
  which the per-user deafen check (S13, `:29`) filters. The owner's sketch had
  no category; one is added because mob lines carry `CategorySpeech`,
  `CategoryShout`, `CategoryRally` and `CategoryWarcry` today.
- `Room.SendTextHidingNames(cat, text, names, excludeUserIds...)`: the same,
  unfiltered, for authored sounds that are not chatter (S12's audited line).
  It replaces `sendAudioRoomTextHidingNames`.

**The unseen word.** `HideNames` says "something" at no sight (S9); ruling 3
wants "Someone" for a speaker. `messaging` gains `HideSpeakerNames(text,
names, d)`, the same matcher with "a figure" and "someone". Speech lines use
it; sounds (rally, warcry, howl, taunt) keep `HideNames`, whose "Something
lets out a rallying roar!" is today's anonymous mob line word for word.

**Shared bodies.**

- `actions.Say(actor, text)` now also sends the room line:
  `FormatSayText` with the actor's colours (from `IsPlayer()`), with the
  speaker's name hidden by each listener's sight, excluding a speaking player.
  A player speaker sends through `SendCommunicationHidingNames` (deafen
  applies); a mob speaker through `SendTextHidingNames` (authored NPC speech,
  deafen does not apply, ruling 6). The wrappers keep: mute, drunk, escaping, the self line,
  and the mob's `PlayerCt() < 1` early return (S5, a cost shortcut, left).
- New `actions.Shout(actor, text) ShoutResult`: reveal (rider 5, S6), the room
  line through the same per-speaker sender as `Say` (ruling 6), the
  adjacent-room line, and
  waking sleepers except the shouter. The player wrapper keeps mute,
  uppercase, drunk, escaping and the self line; the mob wrapper keeps nothing
  but the call. If a speaker is somehow still hidden after the reveal (S2),
  every listener reads the no-sight form, so a hidden name never leaks.
- **Adjacent rooms** keep hearing an anonymous shout, and now one line for
  both: the player's `Someone shouts from the <exit> direction, "<words>"`
  (ruling 3, words always heard). Mob shouts next door gain their words (S6).
- **Rally and warcry.** `ExecuteRally` and `ExecuteWarcry` are already
  shared. The room line (wording per side, unchanged) goes through one
  shared sender, `actions.SendHeard(actor, cat, text)`, which calls
  `SendTextHidingNames` with the actor's name and excludes a player actor.
  The player's lines, including the Resonant Larynx fold lines
  (`rally.go:73`, `warcry.go:77`), stop being visual (S11): a roar is heard in
  the dark, name hidden by sight. They are authored text, not chatter, so
  deafen does not apply (S12).
- **Howl and taunt** (mob only, S10) move mechanically from the two mob
  helpers to `SendTextHidingNames`: each `(anon, full)` pair becomes the full
  line with `[mob name, target name]`. Player taunt stays visual (out of
  scope).
- `sendAudioRoomText` and `sendAudioRoomTextHidingNames` are deleted with
  `darkness.go`, and `audio_room_text_sight_test.go` is ported to the Room
  senders. `Actor.SendRoomCommunication` (S15), dead and now contradicting the
  shipped rule, is deleted with its nine test fakes.

**Guard (5b).** Repo-root `speech_wrapper_guard_test.go` fails if
`usercommands/{say,shout,rally,warcry}.go` or
`mobcommands/{say,shout,rally,warcry}.go` matches `SendTextCommunication`,
`sendAudioRoomText`, `HideNames`, `HideSpeakerNames`, `ParticipantSight`,
`TransitionToRevealing`, `ForEachAdjacentRoom`, `OnSleeperWoken`,
`room\.SendText\(` or `room\.SendTextVisual\(`, and if any file outside
`internal/rooms` and `internal/actions` calls
`SendCommunicationHidingNames`. Comments are dropped before matching, as in
5a. Party member lines (`memberUser.SendText`) do not match. Proven able to fail. T3, T4 add the two new senders; T5 drops
the dead recogniser and updates the comments.

### Parity table (5b)

| Rule | Player today | Mob today | Both after |
|---|---|---|---|
| Say: name by listener sight, three tiers | no (always named) | two tiers | yes |
| Say: blinded listener in a lit room hears no name | no | no (lit shortcut) | yes |
| Say: words always heard | yes | yes | yes |
| Say: reveals a hidden speaker | yes | yes | yes |
| Say: deafened listener filtered | yes | no | unchanged: players yes, NPCs no (ruling 6) |
| Shout: name by listener sight, three tiers | no | two tiers | yes |
| Shout: reveals a hidden speaker | yes | no | yes |
| Shout: adjacent rooms hear an anonymous line with the words | yes | no words | yes |
| Shout: wakes sleepers in the room | yes | yes | yes |
| Shout: deafened listener filtered | yes | no | unchanged: players yes, NPCs no (ruling 6) |
| Rally/warcry heard, name by sight | no (visual) | two tiers | yes |
| Rally/warcry deafen-filtered | no | no | no |
| Unseen speaker reads "Someone" | n/a | "someone" (2 tiers) | "A figure" / "Someone" |

## What changes in play

**Players see, 5a.** Nothing new is refused. `remove all` no longer strips
cursed gear: each cursed item stays on with the cursed line, unless the
player has Spellcasting 4. `get all <name>` behaves exactly as before.

**Players see, 5b.** In a dark room a speaker's name now follows the
listener's sight: "A figure says, ..." at shapes, "Someone says, ..." in
blackness, and a blinded listener in a lit room hears "Someone" too. This
applies to other players (named to everyone today) and to NPCs (two tiers
today). Mob shouts next door now carry their words. A player's rally or
warcry is heard in the dark instead of vanishing.

**Mobs, 5a.** A mob cannot pick up anything, or gold, when it sees nothing,
nor an exploding item. A mob cannot look at, or be seen to look at, a hidden
player. A busy mob cannot take gear off, and a mob's cursed gear stays on
(Spellcasting 4 aside). Crafters stop in the dark: a planner crafter at a
station after the lamps fail waits for light, and the AI companion neither
offers nor starts a recipe it cannot see to make.

**Mobs, 5b.** A hidden human mob that shouts "it's time to die!" on entering
combat now reveals itself. The AI companion's speech, quest NPC lines and
shopkeepers follow the three-tier rule, and a deafened player still hears
them (ruling 6).

## Testing and gates

- 5a: a table drives each shared body through a `UserActor` and a `MobActor`
  and asserts the same outcome for every row of the 5a parity table
  (`TakeFloorItem`, `GetGoldFromFloor`, `ResolveLook`, `RemoveEquipment`,
  `RemoveAllEquipment`, `InitiateCraft`), with light set by `lamp` and
  blindness by the perception machine so the verdicts are exact, not rolled.
  Wrapper tests pin today's wording for each player refusal and the silent
  mob path. The companion's `craftableHere` returns nothing in the dark and
  its `remove` refuses a cursed item.
- 5b: a per-listener table for say, shout, rally and warcry, each with a
  player speaker and a mob speaker, over five listeners in the speaker's room:
  clear sight (named), shapes ("A figure"), dark ("Someone"), blinded in a lit
  room ("Someone"), deafened (a player speaker's say and shout: nothing; a
  mob speaker's say and shout, and rally and warcry from either: heard, per
  ruling 6). Every row asserts the words arrive.
  Adjacent rooms get the one anonymous line with the words. A hidden mob that
  shouts is revealed.
- The T6 and T7 tests move with the code, unchanged in what they assert;
  `FormatSayText`'s tests stay.
- Both re-fork guards, each proven able to fail; the lookup guard re-key;
  T3 to T5.
- `context.md` updated for `internal/actions`, `internal/rooms`,
  `internal/messaging`, `internal/mobcommands`, `internal/usercommands` and
  `modules/aicompanion`.
- Gate per PR: gofmt, vet, build, `go test ./...`, golangci-lint
  new-from-merge-base, boot check, playtest (5a: companion in a dark room
  with a station and cursed gear; 5b: say and shout between two players and
  an NPC across light, shapes, dark and blinded). **5a ships first; 5b
  follows as a second PR.**

## Out of scope

- Whisper and reply: remote tells, not room speech (S18).
- Mob `sayto` and `replyto`, which name both parties to the room with no
  sight gate (S18). A finding, filed, not fixed here.
- Player emote and `ask` room lines, and player taunt's visual channel.
- Equip displacing a cursed item: the check lives in `usercommands/equip.go`
  (`:173-193`) and `Character.Wear`; mob equip and `gearup` are a separate
  row.
- Disarm and forced unequips (`combat/criteffects.go:61`,
  `hooks/combat_shared_helpers.go:243`), which bypass `RemoveEquipment`.
- The mob `say` early return with no players present (S5).
- Retuning any sight threshold.

