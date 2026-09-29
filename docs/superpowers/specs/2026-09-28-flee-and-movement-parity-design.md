# Flee and movement parity: mobs flee and walk by the player's rules

Date: 2026-09-28. Player/mob parity slice 4, split into **4a flee** and
**4b movement**. Source: the owner-ordered parity audit (2026-09-28), rows 5
and 8. Each sub-slice ships as its own PR, 4a first.

## Facts verified against source (2026-09-28, master `62f9027a2`)

Config values are read from the `git show HEAD:_datafiles/config.yaml` blob,
not from disk (the file carries skip-worktree). "Absent" means the key does
not appear in the blob, so the Go default is the live value.

### Flee, player side

| # | Fact | Where |
|---|---|---|
| F1 | `usercommands.Flee` gates in this order: clears a stale admission unless already disengaging; refuses under `NoMovement` ("locked in"); under `NoFlee`; when already `IsDisengaging()`; when `!IsInCombat()`; when `CombatPhase == nil` | `internal/usercommands/flee.go:61-111` |
| F2 | It publishes a pending admission (`fleeAdmission{}` in the user's temp data under `flee-include-skill`) BEFORE the transition, then calls `CombatPhase.TransitionToDisengaging(TriggerFleeCommand)`; on error it cancels the admission and picks a line: not in combat, grappled, "You can't flee from the ground. Stand up first!", or "You can't break away just yet." | `flee.go:106-134` |
| F3 | Cost is quoted and committed once, only after an accepted transition: `QuoteActionCost{Action: costs.ActionFlee, Pool: PoolStamina, Base: FleeStaminaCost, Modifier: FlightFleeStaminaMult if mutations.IsFlying else 1.0, Units: 1}` then `CommitCost(quote, CostPartial)`. Shortage never refuses; it sets `includeSkill = !Short()` on the ready admission and prints `fleeShortageText` | `flee.go:136-157` |
| F4 | `TakeFleeAdmission` / `CancelFleeAdmission` read the admission from `UserRecord` temp data; `Mob` has `SetTempData`/`GetTempData` but no `TakeTempData` | `flee.go:31-59`; `internal/users/userrecord.go:545,561,580`; `internal/mobs/mobs.go:1007,1020` |
| F5 | `TransitionToDisengaging` vetoes a `TriggerFleeCommand` when `vetoes.positionSelf()` is false; the check registered for every character is `c.IsStanding()`. It is wired through `characters.OnCharacterCreated`, which `Character.Validate` fires once per character, mobs included (`mobs.go:659` calls it at spawn) | `internal/state/combatphase/combatphase.go:404-419`; `internal/hooks/CombatPhase_Vetoes.go:33-38,102-104`; `internal/characters/validate.go:647-650` |
| F6 | Legal edges: `Engaging -> Disengaging`, `Engaged -> Disengaging`, `Disengaging -> {Idle, Engaged}`. There is no `Idle -> Disengaging`, so an out-of-combat flee cannot enter the phase | `internal/state/combatphase/transitions.go:30-37` |
| F7 | `OnRoundTick` does nothing in `Disengaging`; resolution is external through `ResolveFlee(success)`: success force-idles, failure restores `Engaged` on `DisengagingData.LastTarget` | `combatphase.go:440-441,464-484` |
| F8 | The round loop resolves a player's flee in `handlePlayerCombat`, after `ValidateAggro`/retarget (`:147`) and `CancelCombatConditions` (`:166`), at `if handlePlayerFlee(...) { continue }` | `internal/hooks/NewRound_DoCombat.go:173` |
| F9 | `handlePlayerFlee`: consumes the admission (`:849-862`); refuses a grappled fleer with `ResolveFlee(false)` (`:871-877`); calls `combat.ResolveFleeBlockers(char, room, includeSkill)` (`:885`); awards Skullduggery only when `contested && includeSkill` (`:897-900`); on a blocker prints private and room lines and `ResolveFlee(false)` (`:901-922`); no exit is a failure (`:925-934`); success prints "flees to the X exit", `targeting.Release`, `ResolveFlee(true)` (`:936-946`), then `rooms.MoveToRoom` (no movement cost), charmed mobs follow, `Look`, and the room behaviour `room_enter` (`:948-972`) | `internal/hooks/NewRound_DoCombat_helpers.go:838-976` |
| F10 | `combat.ResolveFleeBlockers(fleer *characters.Character, room *rooms.Room, includeSkill bool) (*FleeBlocker, bool)` already serves players and mobs; it applies the prone penalty and the sight ramp and awards nothing | `internal/combat/flee.go:59-120` |
| F11 | The delay: the command only enters `Disengaging`; the escape happens on the next `NewRound`. `RoundSeconds: 4`, `TurnMs: 50` | `config.yaml:193,189` |
| F12 | `wireFleeCancellationMessage` retracts an orphaned admission when `Disengaging -> Idle` happens for a reason other than flee success or death; it looks the character up as a USER only | `internal/hooks/CombatPhase_FleeCancellation.go:16-34` |
| F13 | Knobs: `FleeStaminaCost: 10` (shipped); `FlightFleeStaminaMult` absent, Go default 0.5 | `config.yaml:755`; `internal/configs/config.balance.combat.go:202-207` |

### Flee, mob side

| # | Fact | Where |
|---|---|---|
| F14 | `mobcommands.Flee` gates on `IsNonCombatant`, `NoFlee`, and grapple (room line "tries to break free but you've got them locked down!"). It has no `NoMovement`, standing, in-combat or already-fleeing gate and charges nothing | `internal/mobcommands/flee.go:20-44` |
| F15 | It resolves in the same command: `ResolveFleeBlockers(&mob.Character, room, true)` (skill always included), awards on `contested` alone, prints "tries to flee but is blocked!" and returns without touching `CombatPhase` | `flee.go:50-73` |
| F16 | Otherwise it calls `targeting.Release` BEFORE looking for an exit, so a cornered mob ("looks around frantically") has already dropped its fight; on an exit it prints "X flees!", moves through `mobcommands.Go`, and fires the `mob_flee` behaviour event | `flee.go:75-103` |
| F17 | `handleMobCombat` ticks the mob's `CombatPhase` (`:301-304`) and has no `Disengaging` branch anywhere in the mob pass | `internal/hooks/NewRound_DoCombat.go:241-436` |
| F18 | `mob_disengaging` is emitted on `Engaged -> Disengaging` for mobs, which nothing reaches today because no mob path transitions | `internal/hooks/CombatPhase_BtreeEvents.go:38-39` |
| F19 | Mob flee callers: `actFlee` (`internal/behaviortree/actions_combat.go:89-96`, 17 behaviour files author `do: flee`); `PackFlee` issues `flee` to every same-species packmate in the room whether or not it is fighting (`internal/hooks/MobDeath_PackFlee.go:56-95`); the AI companion's flee stance (`modules/aicompanion/combat.go:449-455`) | grep |

### Movement, player side

| # | Fact | Where |
|---|---|---|
| M1 | Walking is refused in combat | `internal/usercommands/go.go:125-137` |
| M2 | Action points: `actionCost := 10`, or `50` when `GetCarriedWeight() > CarryCapacity()`; `DeductActionPoints` refuses ("too encumbered" / "too tired"). Both numbers are hardcoded, not knobs | `go.go:191-208`; `internal/characters/resources.go:14-24` |
| M3 | Terrain stamina: destination biome `GetMovementCost()` (shipped 0.5 road to 2.5 cliffs), `GetMovementStaminaCost(terrain)`, times `FlightMoveStaminaMult` when flying, charged by `ApplyCostFloatOrRefuse(PoolStamina, cost)`; refusal refunds the action points. A "winded" warning prints under a quarter of `EffectivePoolMax` | `go.go:212-259`; `_datafiles/world/dogmud/biomes/*.yaml` |
| M4 | `GetMovementStaminaCost` = `costs.Calc(Base: MovementBaseStaminaCost * terrain, encumbrance from carried/capacity, inverse Search skill)`, times the mutation speed modifier, times `HiddenMoveStaminaMultiplier` when hidden, capped at `MovementMaxStaminaCost` | `internal/characters/resources.go:91-123` |
| M5 | Knobs: `MovementBaseStaminaCost: 0.5`, `MovementMaxStaminaCost: 20.0`, `HiddenMoveStaminaMultiplier: 3.0`, `MovementSearchTrainChance: 0.005`; `FlightMoveStaminaMult` absent, default 0.5 | `config.yaml:1208,1210,900,1218`; `config.balance.combat.go:199-201` |
| M6 | The charge happens BEFORE the lock check (`:263-328`) and before the exit-message requeue (`:332-336`), so a player pays for a door that stays locked, and an exit with an `ExitMessage` charges twice (the requeued command runs the whole body again). The second is latent: no dogmud room authors an exit message (1 file world-wide) | `go.go:191-336`; grep `exitmessage` |
| M7 | Action points regenerate +1 per turn for `users.GetAllActiveUsers()` only, capped at `ActionPointsMax` (Mods 200, floor 50). At 50 ms turns that is 80 per round, 8 plain steps a round, a burst of 20 | `internal/hooks/NewTurn_ActionPoints.go:23-28`; `internal/hooks/hooks.go:75`; `world.go:870` (repo root); `internal/characters/validate.go:114,151-153` |
| M8 | Player hidden detection, sneaking mover: per observer in the destination, `CalcSneakScoreVsObserver(mover, observer, destLight)` against `CalcDetectionScore(observer, destRoom)` through `combat.RunContest`; player observers first (the spotter is told), then mob observers (silent); spotted drives `Awareness.TransitionToRevealing` and clears `sneaking` | `go.go:550-621` |
| M9 | Player hidden detection, newcomer: for each hidden player and hidden mob in the destination, the newcomer's detection score against the hider's sneak score; a spot reveals the hider with sight-gated lines to both and a room visual line for a mob; the newcomer's Search is awarded on BOTH outcomes | `go.go:623-725` |
| M10 | The detection helpers already live in `actions`: `CalcSneakScoreVsObserver(sneaker, observer, room)`, `CalcDetectionScore(c, room)` | `internal/actions/skill_helpers.go:65,87` |
| M11 | A completed move rarely trains Search (`movementTrainsSearch`) | `go.go:73-82,398-401` |

### Movement, mob side

| # | Fact | Where |
|---|---|---|
| M12 | `mobcommands.Go` checks `NoMovement`, then: a numeric `rest` that is not an adjacent exit teleports the mob ("runs off suddenly"); `home` queues `pathto home`; a locked exit emotes; a far-side lock refuses; otherwise relocates (`RemoveMob`, `clearRoomAggroOnDeparture`, `AddMob`), narrates, pulls NPC party members through, and pauses at a waypoint. No action points, stamina, encumbrance, combat gate or detection roll | `internal/mobcommands/go.go:106-280` |
| M13 | Every mob movement is a command string: `go` is registered at `mobcommands.go:59`, and a bare exit word falls through `TryCommand` to `Go` | `internal/mobcommands/mobcommands.go:59,180-191` |
| M14 | Callers: `Wander` (increments `WanderCount` and issues `go` before knowing the result; drags pack followers) `wander.go:93-110`; the path walker `NewRound_IdleMobs.go:140-196`; schedules `NewRound_IdleMobs_schedule.go:183-192`; patrols `NewRound_IdleMobs_patrol.go:174-181`; pack followers `internal/mobs/pack_roaming.go:292-293`; behaviour-tree actions `go_to_caller_room` (`actions_mob.go:313`), `move` (`:357`), `keep_distance` (in combat, `actions_archer.go:279`), scout (`actions_scout.go:211`), `pathto` from party and forager actions; `callforhelp` responders (`go <roomId>`, `callforhelp.go:103-109`); a defender walking to its attacker (`NewRound_DoCombat_unified.go:925`); charmed mobs following a walking player (`usercommands/go.go:485-500`) or a fleeing one (`NewRound_DoCombat_helpers.go:950-956`); `modules/follow/follow.go:301`; `seeders/witness_response.go:91`; ferry factors (`ferry/factor.go:320`); mob flee (`mobcommands/flee.go:96`) | grep |
| M15 | AI companion: `goTo` (`modules/aicompanion/actions.go:941`) and `explore` (`:975`) start a trip; `advanceTravel` issues one `go <exit>` per step (`travel.go:193`). A step that has not moved after `stepTimeoutRounds = 3` gets `Fails++` on that exit (`travel.go:39,129-148`), and an exit with `Fails >= 3` is dropped from route planning (`worldmap.go:278,318`) | `modules/aicompanion/` |
| M16 | The path walker calls `Path.Next()` (which advances the queue) BEFORE issuing the step; on the next tick a mob not standing in `Current().RoomId()` re-paths through its remaining waypoints. `PathQueue` has no peek | `NewRound_IdleMobs.go:144-170`; `internal/mobs/mobs_path.go:27-34` |
| M17 | A patrol mob not at its waypoint adds one to `patrol_path_fail_count` EVERY tick (path in flight or not) and falls back to `pathto home` at `ScheduleMaxPathRetries`; a schedule only counts when it queues a new `pathto`. `ScheduleMaxPathRetries` is absent, default 20 | `NewRound_IdleMobs_patrol.go:78-93,174-181`; `NewRound_IdleMobs_schedule.go:125-142,183-192`; `config.balance.mobs.go:27-28` |
| M18 | Mob `ActionPoints` is never set at spawn (`Health`, `Stamina`, `Conviction` are filled; AP is not) and never regenerates (M7 loops users only). `ActionPoints` has no yaml tag; the only mob YAML that sets it is the loot goblin, to 0. So every live mob holds 0 action points | `internal/mobs/mobs.go:670-672`; `internal/characters/character.go:124`; `_datafiles/world/dogmud/mobs/endless_trashheap/13-loot_goblin.yaml:59` |
| M19 | Mob stamina regenerates in `AutoHeal`, every third round: `StaminaPerRound()` (`MobStaminaRegenPct: 0.02` of raw `StaminaMax`, floor 1), a quarter in combat, times the room regen multiplier. Players get the same rate (`PlayerStaminaRegenPct: 0.02`) | `internal/hooks/NewRound_AutoHeal.go:35,238-251,339-349`; `resources.go:403-429`; `config.yaml:1190,1193` |
| M20 | `StaminaMax` = `StaminaBase 5 + Vitality*3 + Willpower*1 + Strength*0`, so a stat-100 mob holds about 405 and regains about 8 every three rounds | `validate.go:104-108`; `config.yaml:1249-1252` |

### What a mob does today when it cannot afford an action

| # | Fact | Where |
|---|---|---|
| R1 | `CommandIsReady`, the behaviour tree's gate for `command_best_of`, checks activity, the special-move timer and per-command state. It never checks cost | `internal/actions/command_readiness.go:24-134` |
| R2 | Shared actions return a structured cost refusal; player wrappers render `CostRefusalText`, mob wrappers stay silent. Example: mob `sneak` returns quietly on `CostRefused` | `internal/actions/action_cost.go:22-42`; `internal/mobcommands/sneak.go` |
| R3 | So a refused mob action is a silent no-op, retried at the caller's own cadence. Nothing in trees, planners or executors reacts to a refusal. Mob commands are queued (`Mob.Command`), so no caller learns the result synchronously | `internal/mobs/mobs.go:938-963` |

### Guards that key on the code this slice moves

| # | Guard | Where |
|---|---|---|
| G1 | Contest sites registered by `file:function`: `internal/combat/flee.go:ResolveFleeBlockers` and `internal/usercommands/go.go:Go` (the four detection contests) | `internal/combat/contest_site_guard_test.go:60,73,359,368` |
| G2 | Parses `go.go` and expects at least three name-tagged stealth lines | `internal/usercommands/dark_name_leak_guard_test.go:58,139` |
| G3 | Asserts `go.go` does not call `OnSkillUse` directly | `internal/usercommands/go_test.go:87` |
| G4 | Messaging surface verdicts keyed `file|literal`, including `go.go` lines | `messaging_surface_guard_test.go:1364-1365` |
| G5 | Existing flee tests: `internal/usercommands/flee_cost_test.go`, `internal/hooks/flee_cost_test.go`, `internal/combat/flee_test.go`, `internal/hooks/hooks_test.go:2403-2415` | `ls`, grep |

## Owner rulings (binding)

1. **Flee: full parity.** A mob enters the same `Disengaging` phase, pays the
   same flee cost (flight modifier included), is refused when not standing,
   and escapes through the same round resolution as a player.
2. **Movement: A, full parity.** Mobs pay the same action point, terrain
   stamina and encumbrance costs per step as players, idle wandering included.
3. **Hidden detection on movement: symmetric both ways.** A mob walking in
   rolls to spot hidden occupants; a sneaking mob is rolled against.

## 4a: Flee

**Shape.** Two shared bodies in `internal/actions/flee.go`, following
`actions.Drink` and `actions.InitiateCast`:

- `actions.BeginFlee(actor Actor) FleeBegin` is today's command half (F1 to
  F3): the `NoMovement`, `NoFlee`, already-disengaging, not-in-combat and
  nil-phase gates; the pending admission; `TransitionToDisengaging` with the
  actor's `ActorRef`; the veto reason (grappled, not standing, other); the
  quote and partial commit with the flight modifier; the ready admission.
  `FleeBegin` carries `Accepted bool`, `Short bool` and a refusal enum
  (`FleeRefuseRooted`, `FleeRefuseNoFlee`, `FleeRefuseAlready`,
  `FleeRefuseNotInCombat`, `FleeRefuseNotReady`, `FleeRefuseGrappled`,
  `FleeRefuseProne`).
- `actions.ResolveFlee(actor Actor, room *rooms.Room) FleeOutcome` is today's
  round half without the room move (F9 up to the move): consume the
  admission, grapple refusal, `ResolveFleeBlockers`, the
  `contested && includeSkill` award through `actor.AwardResolved`, the exit
  pick, `ResolveFlee(false)` on block or no exit, and on success
  `targeting.Release` plus `ResolveFlee(true)`. `FleeOutcome` reports
  `Resolved`, `Grappled`, the blocker, `NoExit`, and the exit name and room.

**Admission moves to the Character.** The handoff is character state, and
`Mob` lacks an atomic take (F4). A runtime (`yaml:"-"`) field on `Character`
behind three methods (`PublishFleeAdmission`, `TakeFleeAdmission`,
`CancelFleeAdmission`) replaces the user temp-data key, keeping the
pending-then-ready semantics exactly. `wireFleeCancellationMessage` then
cancels for any character and only speaks when the character is a player.
The usercommands forwarders are deleted and the compiler enumerates their
callers.

**Wrappers.**

- `usercommands.Flee`: build the actor, call `BeginFlee`, map the refusal to
  today's line (every line unchanged), print the shortage line and "You
  attempt to flee...". Nothing else.
- `mobcommands.Flee`: call `BeginFlee`. Silent on every refusal except
  grappled, which keeps today's room line so the grappling player sees the
  hold working (F14). Nothing else happens in the command; the escape waits
  for the round.
- `hooks.handlePlayerFlee`: call `ResolveFlee`, render today's private and
  room lines, and on success do the player move exactly as today.
- New `hooks.handleMobFlee(mob, room)`, called in `handleMobCombat` inside
  the in-combat block after `ValidateAggro` and before `mob_combat_round`, the
  same point as F8: call `ResolveFlee`; on a blocker print today's
  "tries to flee but is blocked!" line; on no exit today's "looks around
  frantically" line; on success "X flees!" then an uncharged relocation and
  the `mob_flee` event.

**Uncharged mob relocation.** A player's flee success moves through
`rooms.MoveToRoom` and pays no movement cost (F9). The mob twin must not
route through `mobcommands.Go`, which 4b makes paid, and `hooks` does not
import `mobcommands`. So 4a lifts the relocation tail of `mobcommands.Go`
(remove, `clearRoomAggroOnDeparture`, add, exit and entry lines, sounds) into
`actions.RelocateMob(mob, from, exitName, dest)`, used by `mobcommands.Go` and
`handleMobFlee`. No behaviour change for walking.

**Per-actor narration that stays different.** Refusal and shortage text is
player-only (a mob has no one to tell, R2). The mob keeps its grapple room
line, its "cornered" line and its destination entry line; a fleeing player
has no entry line at the destination today, and this slice does not add one.
Name colours come from `IsPlayer()`.

**Guard.** A repo-root `flee_wrapper_guard_test.go` fails if
`usercommands/flee.go` or `mobcommands/flee.go` calls
`TransitionToDisengaging`, `QuoteActionCost`, `CommitCost` or reads
`FleeStaminaCost`, and if any production file outside `internal/actions` and
`internal/combat` calls `ResolveFleeBlockers` or `CombatPhase.ResolveFlee`.
Proven able to fail by a temporary violation.

**Parity table (4a).**

| Rule | Player today | Mob today | Both after |
|---|---|---|---|
| Rooted (`NoMovement`) refuses | yes | no | yes |
| `NoFlee` refuses | yes | yes | yes |
| Already disengaging refuses | yes | no gate | yes |
| Out of combat refuses | yes | no | yes |
| Not standing refuses (phase veto) | yes | no | yes |
| Grappled refuses | yes | yes | yes |
| `FleeStaminaCost`, partial, flight 0.5 | yes | free | yes |
| Shortage drops Skullduggery from the contest | yes | skill always in | yes |
| Enters `Disengaging`, `mob_disengaging` fires | yes | no | yes |
| Escape on the next round | yes | same command | yes |
| Award only when contested and paid in full | yes | contested only | yes |
| Blocked or no exit returns to `Engaged` | yes | blocked stays, cornered drops the fight | yes |
| Success move pays no movement cost | yes | free today | yes |

## 4b: Movement

**Shape.** Moving the whole 800-line player `Go` is not the job: most of it
is player-only (quest notify, fog-of-war, greetings, parties, `Look`, GMCP,
unlocking with keys). The ruling covers costs and detection, so those become
the shared bodies, in `internal/actions/move.go`:

- `actions.ChargeMove(actor Actor, dest *rooms.Room) MoveCharge`: settles a
  mob's action points (below), deducts 10 or 50 by encumbrance, prices the
  step with `GetMovementStaminaCost(dest biome)` and the flight multiplier,
  charges through `ApplyCostFloatOrRefuse`, refunds the action points on a
  stamina refusal. `MoveCharge` carries a refusal (`MoveRefuseEncumbered`,
  `MoveRefuseTired`, `MoveRefuseExhausted`) and `Winded`.
- `actions.QuoteMove(actor, dest) MoveCharge`: the same arithmetic, read
  only, for callers that must decide before issuing a step.
- `actions.EntryDetection(mover Actor, dest *rooms.Room, sneaking bool)
  EntryDetection`: both blocks of M8 and M9 over the destination's players
  and mobs, with the same contests, reveals and Search awards (through
  `AwardResolved`, so a mob passes 0). Lines go to each player recipient
  through its own `UserActor.SendText` and to the room through
  `SendTextVisual`, with the mover's name coloured by `IsPlayer()`. It
  reports whether the mover is still sneaking.
- The rare post-move Search roll (M11) moves beside them, since a mob's step
  price reads its Search rank.

**Charge after the gates, once.** Both wrappers call `ChargeMove` after the
lock and far-side-lock checks and after the exit-message requeue, right
before relocation. This fixes M6 for players: a locked door and a requeued
step stop charging.

**Mob action points.** The ruling is not implementable without this: every
mob holds 0 action points and nothing refills them (M18), so charging them
freezes every mob in the world. Mobs regain action points at the player
rate, 1 per turn up to `ActionPointsMax`, settled lazily: `Character`
records the turn it last settled, and `ChargeMove` and `QuoteMove` add the
elapsed turns before reading. A spawn starts full. This avoids a 20 Hz loop
over every mob instance; players keep the per-turn hook, whose `{ap}` prompt
token needs a live number.

**Wrappers.** `usercommands.Go` keeps every line it prints today and calls
`ChargeMove` and `EntryDetection` where it now computes them inline.
`mobcommands.Go` calls `ChargeMove` on the adjacent-exit path (silent on
refusal, R2), relocates through `actions.RelocateMob`, then calls
`EntryDetection`. The non-adjacent numeric teleport (M12) charges one step
priced at the destination (open question 3).

**When a mob cannot afford a step.** Refusal stays silent and no caller
retries faster than it does today (R3), so there is no hot loop. The risk is
callers that misread a refused step as something else. The smallest safe
handling, caller by caller:

1. **Path walker.** Add `PathQueue.Peek()`. The walker quotes the next step
   first; if unaffordable it leaves the path intact and waits a round,
   instead of `Next()` advancing a step the mob never takes and forcing a
   re-path (M16). If the step can never be afforded (its price exceeds the
   mob's reachable stamina, possible only hidden or overloaded on rough
   terrain), the walker clears the path, which hands the mob to the existing
   schedule and patrol fallbacks rather than parking it forever.
2. **Patrol and schedule counters.** A tick the walker spent waiting does not
   count toward `patrol_path_fail_count` (M17), so resting does not trigger
   the home fallback. Schedules already count only new `pathto`s.
3. **Wander and pack movement.** `Wander` quotes the picked exit before
   issuing it, and only then counts `WanderCount` and drags pack followers.
   Each follower pays its own step; one that cannot stays behind.
4. **AI companion travel.** `advanceTravel` quotes before issuing a step. When
   tired it does not issue, does not start the step clock, and does not add
   `Fails` to the exit, so exhaustion never poisons the companion's map
   (M15). It records one line in the companion's mind ("You are too tired to
   go on, and stop to catch your breath.") so its next decision knows why.
   Under ruling 2 the companion's walking pays like anyone's, `goTo` and
   `explore` included.
5. **Behaviour-tree single steps** (`move`, `go_to_caller_room`,
   `keep_distance`, scout): quote first and return `Failure` when
   unaffordable, so a selector falls through (an archer that cannot kite
   fires instead).

**Per-actor narration that stays different.** Refusal and "winded" text is
player-only. Unlock lines, bumping into walls and the locked-exit emote stay
in their wrappers. Detection lines are shared and differ only in the name
colour.

**Guard.** A repo-root `move_wrapper_guard_test.go` fails if
`usercommands/go.go` or `mobcommands/go.go` calls `DeductActionPoints`,
`GetMovementStaminaCost`, `ApplyCostFloatOrRefuse`,
`CalcSneakScoreVsObserver`, `CalcDetectionScore` or `RunContest`, and if any
production file outside `internal/actions` and `internal/characters` calls
`DeductActionPoints`. Proven able to fail. G1 re-keys to the new
`actions/move.go` function, G2 and G4 follow the lines to their new file, G3
gains an `actions` twin.

**Parity table (4b).**

| Rule | Player today | Mob today | Both after |
|---|---|---|---|
| 10 action points a step, 50 over capacity | yes | free | yes |
| Action points regenerate 1 a turn | yes | never | yes |
| Terrain stamina with encumbrance and Search discount | yes | free | yes |
| Flight 0.5, hidden 3.0, cap 20 | yes | free | yes |
| Refused stamina refunds action points | yes | n/a | yes |
| Charged after locks and requeue | no (charged first) | n/a | yes |
| Sneaking mover rolled against players | yes | no | yes |
| Sneaking mover rolled against mobs | yes | no | yes |
| Newcomer rolls for hidden players | yes | no | yes |
| Newcomer rolls for hidden mobs | yes | no | yes |
| Search awarded on the newcomer's rolls | yes | no | yes |
| Rare Search training per move | yes | no | yes |
| Walking refused in combat | yes | no | unchanged (open question 4) |

## What changes in play

**4a.** A fleeing mob announces nothing new but now takes a round to get
away, during which it neither attacks nor escapes, and players get that
round to block it. A tripped or bashed mob cannot flee until it stands, which
makes knockdown a real answer to a fleeing boss. Fleeing costs a mob stamina,
and a spent mob flees without its Skullduggery. A cornered mob stays in the
fight instead of silently dropping it. Jailing-style roots now hold mobs too.
The risk is `PackFlee` (open question 1).

**4b.** Mobs now tire. Arithmetic from shipped knobs, not a measurement: a
stat-100 mob regains about 2.7 stamina a round and an unloaded forest step
costs about 0.55, so ordinary patrols, schedules and wandering are
unaffected. The risks are the edges:

- **Chases and kiting.** In-combat regen is a quarter, so a kiting archer or
  a pursuing mob runs down over a long chase, as a player does.
- **A mob stopping.** Hidden (x3), overloaded (up to x5) and rough terrain (up
  to x2.5) compound to the 20 cap; a low-Vitality mob can find a step it
  cannot pay and wait, and one step above its whole pool clears its path.
- **Patrol and schedule timing.** A tired mob arrives late; a shopkeeper may
  open late after a long walk.
- **Companions.** A loaded companion can fall behind its owner or stop
  mid-errand.
- **Detection.** Hidden players can now be found by a guard walking in, and a
  thief mob sneaking in can be spotted. Both train Search on mobs, bounded by
  `MobSkillCap`.

**How the playtest watches.** Before merge, the plan measures instead of
inferring: a test over every shipped mob spec that has a patrol or schedule
computes its `StaminaMax` against the worst single step on its route (hidden
and carried load included) and lists any that can never pay. The admin
`mob schedule` readout gains stamina and action points so a playtester can
watch a mob tire. 4a playtest: a generic fighter below 25% health flees a
round late, can be blocked, is refused while tripped, and a pack death
scatters (or not) as ruled. 4b playtest: shadow a patrolling guard and a
scheduled shopkeeper for a full loop; walk an overloaded AI companion across
rough terrain and send it on an errand; hide in a room a guard patrols
through; let a thief mob sneak in. Each checks that nothing freezes, loops,
or falls back home unexpectedly.

## Testing and gates

- 4a: a table drives `BeginFlee` and `ResolveFlee` through a `UserActor` and
  a `MobActor` and asserts identical outcomes for every row of the 4a parity
  table. A hooks test proves a mob's flee resolves on the next round, not in
  the command, and that a blocked mob returns to `Engaged`. The existing flee
  tests (G5) move with the code, unchanged in what they assert.
- 4b: `ChargeMove` table across actors, terrain, flight, hidden and load; mob
  action point settlement; the walker waits without re-pathing and clears a
  never-affordable path; a waiting patrol does not count failures; a tired
  companion does not add `Fails`; `EntryDetection` both directions for all
  four mover and hider pairings with scores forced far apart so the contest
  cannot flake; the M6 fix for a player.
- Both re-fork guards, each proven able to fail; the G1 to G4 re-keys.
- `context.md` updated for `internal/actions`, `internal/characters`,
  `internal/mobcommands`, `internal/usercommands`, `internal/hooks` and
  `internal/mobs`.
- Gate per PR: gofmt, vet, build, `go test ./...`, golangci-lint
  new-from-merge-base, boot check, playtest. **4a ships as its own PR first;
  4b follows as a second PR.**

## Out of scope

- Walking while in combat for mobs (`keep_distance`) beyond paying for it.
- The player flee's missing arrival line at the destination.
- Shadow following (audit slice 6) and every other audit row.
- Pursuit behaviour, which stays authored per U10b-1.
- Retuning any movement or flee knob; the hardcoded 10 and 50 stay as they
  are.

## Open questions for the owner

1. **`PackFlee` scatters bystanders that are not fighting** (F19). Under
   ruling 1 an out-of-combat flee is refused (F6), so those packmates would
   stop scattering and only fighting packmates would flee, a round later.
   Proposed: `PackFlee` issues `flee` to fighting packmates and a paid random
   step to idle ones. Or keep only the fighters.
2. **Charging after the lock and requeue** (M6) changes a player cost: a
   locked door stops costing a step. Proposed: fix it, as designed.
3. **The mob numeric teleport** (`go <roomId>` to a non-adjacent room, used
   by `callforhelp` responders) has no player equivalent. Proposed: charge
   one step priced at the destination. Or exempt it as a scripted move.
4. **Mobs walk in combat; players cannot** (M1). `keep_distance` kiting is a
   walk out of melee, which for a player would have to be a flee. Ruling 2
   covers costs only, so this spec leaves the gate alone. Should kiting
   become a flee in a later slice?
