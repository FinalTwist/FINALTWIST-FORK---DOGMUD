# Movement Parity (Slice 4b) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Mobs pay the player's movement costs (action points, terrain stamina, encumbrance, flight, hidden) and roll the player's hidden-detection contests on entry, through one shared body in `internal/actions/move.go`, without any mob freezing, looping or falling back home because it is tired.

**Architecture:** Three shared bodies (`ChargeMove`, `QuoteMove`, `EntryDetection`) plus the rare Search roll live in `internal/actions/move.go`; `usercommands.Go` and `mobcommands.Go` become thin wrappers that call them after their own lock gates. Mob action points settle lazily from a turn stamp on `Character`, so no 20 Hz loop over mob instances is added. Every caller that issues a mob step quotes first and waits (or fails a behaviour-tree node) instead of issuing a step it cannot pay for.

**Tech Stack:** Go 1.25, testify, the repo's seed-for-test helpers (`rooms.SeedRoomsForTest`, `users.SeedUsersForTest`, `mobs.SetInstanceForTest`, `configs.SetConfigForTest`).

**Spec:** `docs/superpowers/specs/2026-09-28-flee-and-movement-parity-design.md` (section "4b: Movement", owner rulings 2 and 3, open-question rulings 2 and 3).

**Branching:** 4b ships as its own PR AFTER 4a (flee) has merged. Branch from `origin/master` once 4a is in: `git fetch origin && git checkout -b feature/movement-parity-4b origin/master`. Task 0 verifies the 4a contract before anything is written.

---

## Facts verified against source (2026-09-29, `docs/flee-movement-spec` at `c061ca58f` = master `961995259` plus the spec)

Config values are read from `git show HEAD:_datafiles/config.yaml`, never from disk (skip-worktree). Rows marked **moved** differ from the spec's table; rows marked **new** were not in the spec.

| # | Fact | Where |
|---|---|---|
| V1 | Player walking refuses in combat before any cost | `internal/usercommands/go.go:125-137` |
| V2 | Player AP: `actionCost := 10`, `50` when `GetCarriedWeight() > CarryCapacity()`, `DeductActionPoints` refuses with "too encumbered" / "too tired" (plus a `mudlog.Debug`) | `go.go:191-208` |
| V3 | Player stamina: `destRoom := rooms.LoadRoom(goRoomId)` at `:212`; `rooms.GetBiome(destRoom.Biome)` (NOT `destRoom.GetBiome()`), `GetMovementCost()`, `GetMovementStaminaCost(terrain)`, times `FlightMoveStaminaMult` when `mutations.IsFlying`; `ApplyCostFloatOrRefuse(PoolStamina, cost)`; refusal refunds `ActionPoints += actionCost`; winded line when `Stamina < EffectivePoolMax(PoolStamina)/4` | `go.go:212-259` |
| V4 | Charge happens BEFORE the lock block (`:265-328`) and the exit-message requeue (`:332-336`); the requeue re-runs the whole body, so an exit message charges twice. Only `_datafiles/world/default/rooms/frostfang/432.yaml` authors `exitmessage`; no dogmud room does | `go.go:191-336`; grep |
| V5 | **moved** Player detection: `destRoomLight` at `:550`, sneaking-mover block `:553-621` (party members excluded from player observers; mob observers silent), newcomer block `:623-725` (Search awarded on both outcomes via `AwardResolved(user.UserId, success, ...)`) | `go.go:550-725` |
| V6 | The rare Search roll: `movementTrainsSearch()` at `go.go:73-82`, called at `:398-401` inside the `MoveToRoom` success branch | `go.go` |
| V7 | `CalcSneakScoreVsObserver(sneaker, observer *characters.Character, room messaging.RoomVisibility)` and `CalcDetectionScore(c *characters.Character, room messaging.RoomVisibility)` | `internal/actions/skill_helpers.go:65,87` |
| V8 | `DeductActionPoints(amount int) bool` is called in exactly one production file, `usercommands/go.go:198` | `internal/characters/resources.go:14`; grep |
| V9 | `GetMovementStaminaCost(terrain)` = `costs.Calc(base * terrain, encumbrance, inverse Search)`, times mutation speed, times `HiddenMoveStaminaMultiplier` when `IsHidden()`, capped at `MovementMaxStaminaCost` | `resources.go:91-123` |
| V10 | `ApplyCostFloatOrRefuse` decides on a CANDIDATE carry (`fractionalCost(carry, amount)`) and writes nothing on refusal; `costCarry` is unexported, so no read-only float check exists outside `characters` | `internal/characters/pools.go:103,529-549` |
| V11 | `QuoteActionCost`/`CommitCost` cannot price a step: the move price caps AFTER the mutation and hidden multipliers, which `costs.Calc` cannot express | `pools.go:130-240`; `resources.go:108-121` |
| V12 | `ActionPoints int` has no yaml tag (so it persists for players under `actionpoints`); `ActionPointsMax` is `yaml:"-"`, `Mods = 200`, floored at 50 | `internal/characters/character.go:124,150`; `validate.go:114,151-153` |
| V13 | AP regen: `hooks.ActionPoints` adds 1 per `NewTurn` for `users.GetAllActiveUsers()` only | `internal/hooks/NewTurn_ActionPoints.go:23-28`; `hooks.go:75` |
| V14 | **moved** The turn counter: `util.IncrementTurnCount()` at `world.go:868` (spec said 870); `util.GetTurnCount() uint64` | `world.go:868`; `internal/util/util.go:130-136` |
| V15 | Mob spawn fills Health, Stamina, Conviction and never ActionPoints | `internal/mobs/mobs.go:670-672` |
| V16 | Only mob YAML setting `actionpoints` is the loot goblin, to 0 | `_datafiles/world/dogmud/mobs/endless_trashheap/13-loot_goblin.yaml:59` |
| V17 | Knobs (HEAD blob): `TurnMs: 50`, `RoundSeconds: 4`, `MovementBaseStaminaCost: 0.5`, `MovementMaxStaminaCost: 20.0`, `HiddenMoveStaminaMultiplier: 3.0`, `MovementSearchTrainChance: 0.005`, `MobStaminaRegenPct: 0.02`, `StaminaBase: 5`, `StaminaPerVitality: 3`, `StaminaPerWillpower: 1`; `FlightMoveStaminaMult` and `ScheduleMaxPathRetries` absent (Go defaults 0.5 and 20) | `config.yaml:189,193,1208,1210,900,1218,1193,1249-1251`; `config.balance.combat.go:199-200`; `config.balance.mobs.go:27-28` |
| V18 | **moved** Mob Search training is bounded by `MobSkillTrainingCap: 25` (`MobSkillCap: 3` is legacy, superseded) | `config.yaml:1500`; `config.balance.go:476` |
| V19 | `mobcommands.Go` (pre-4a): `NoMovement` gate; numeric `rest` not adjacent teleports; `home` queues `pathto home`; near-side lock emotes; far-side lock refuses silently; relocation tail; NPC party pull; waypoint `noop`. No cost, no detection | `internal/mobcommands/go.go:106-280` |
| V20 | `PathQueue` has `Len`, `Clear`, `Current`, `Next` (advances), `Waypoints`, `SetPath`; no `Peek` | `internal/mobs/mobs_path.go:9-55` |
| V21 | The path walker calls `Path.Next()` BEFORE issuing the step; a mob not standing in `Current().RoomId()` re-paths through its remaining waypoints | `internal/hooks/NewRound_IdleMobs.go:140-196` |
| V22 | Schedule and patrol executors run BEFORE the walker in the same tick; patrol `WantsPath` adds one to `patrol_path_fail_count` every tick; schedule counts only a new `pathto` | `NewRound_IdleMobs.go:88-113`; `NewRound_IdleMobs_patrol.go:174-181`; `NewRound_IdleMobs_schedule.go:183-192` |
| V23 | `Wander` increments `WanderCount` and queues `go <exit>` before knowing the result, then calls `mobs.MovePackFollowers(mob, exitName, preMoveRoomMobs)`, whose followers each `WanderCount++` and queue `go`. A follower not in its alpha's room has `PackAlphaId` cleared next tick | `internal/mobcommands/wander.go:93-110`; `internal/mobs/pack_roaming.go:95-103,256-294` |
| V24 | `internal/mobs` does not import `internal/actions` (and cannot: `actions` imports `mobs`) | grep |
| V25 | BT single steps: `actGoToCallerRoom` `actions_mob.go:293-318`, `actMove` `:348-359`, `actMoveTowardTracked` `actions_scout.go:169-213`; `behaviortree` imports `actions` | grep |
| V26 | AI companion: `advanceTravel` issues `go <exit>` at `travel.go:193`; timeout `stepTimeoutRounds = 3` (`:39`) adds `er.Fails++` (`:144-148`); `Fails >= 3` drops an exit (`worldmap.go:278,318`); mind lines via `c.mind.addLine(Line{Kind: "event", Text: ...}, m.cfg.WorkingMemoryLines)` | `modules/aicompanion/` |
| V27 | **new** Other step issuers, all silent on refusal with no retry logic to fix: `aicompanion/loot.go:254` (`mobCompanionFollow`, one shot), `NewRound_DoCombat_unified.go:925` (defender walks to attacker), `seeders/witness_response.go:91`, `modules/follow/follow.go:301`, `mobcommands/callforhelp.go` (`go <roomId>`), the NPC party pull in `mobcommands/go.go`, charmed mobs following a walking player (`usercommands/go.go:477-490`). The ferry factor walks by `pathto` and has a silent stuck-recovery teleport (`internal/ferry/factor.go:311-330`) that a long rest can trip, harmlessly | grep |
| V28 | G1: `contestSiteOwners["internal/usercommands/go.go:Go"]` and `legacyLiteralFiles` includes `go.go`; the guard fails on stale rows | `internal/combat/contest_site_guard_test.go:73,368,230-265` |
| V29 | G2: `TestHiddenDetectionNeverNamesWhatYouCannotSee` parses `go.go`, counts name-tagged literals containing "shadows", requires at least 3, all inside a `CanSeeClearly` branch or a `SendTextVisual*` call | `internal/usercommands/dark_name_leak_guard_test.go:56-139` |
| V30 | G3: `TestGo_HiddenDetectionFiresThroughTheSeamOnBothOutcomes` requires exactly 2 `AwardResolved(user.UserId, success,` in `go.go`; `TestGo_TheSneakingWrapperSurvives` requires `if !isSneaking {` | `internal/usercommands/go_test.go:83-99,67-70` |
| V31 | G4: `narrationViewpointRegistry` holds two `usercommands/go.go` keys (walls, unlock), both staying in `go.go`; the walk fails on new unregistered keys and stale ones | `messaging_surface_guard_test.go:1364-1365,1446-1515` |
| V32 | **new** `sight_penalty_guard_test.go`: a `combat.RunContest` call must sit in a FUNCTION that also calls a sight helper (`CalcDetectionScore` counts). Each new detection function must call `CalcDetectionScore` itself | `sight_penalty_guard_test.go:70-88,428-475` |
| V33 | **new** `movementTrainsSearch` has its own test file, `internal/usercommands/movement_search_test.go`, whose helpers are used nowhere else | grep |
| V34 | **new** Existing tests that will need a mob with action points: `internal/behaviortree/heard_callforhelp_test.go` (two tests assert a queued `go`), fixtures with `ActionPointsMax.Value` 0 | `heard_callforhelp_test.go:34-72,110-130` |
| V35 | Contests: `combat.RunContest` = `contest.RunWithFloors(compressContestGap(...), ContestFloor)`; a test pins `ContestFloor = 0` and `ContestGapSaturation = 0` for determinism, as `combat_fire_seam_test.go:44-54` does | `internal/combat/run_contest.go` |
| V36 | Admin readout `mob_Schedule` returns early for a mob with no schedule (so a patrol-only mob shows nothing today) | `internal/usercommands/admin.mob.go:224-339` |
| V37 | Package `main` tests can call `loadAllDataFiles(false)` and `mapper.PreCacheMaps()`; the boot smoke test gates on `DOGMUD_BOOT_SMOKE` | `boot_smoke_test.go:31-60`; `main.go:398-401,1610` |

### Contract from 4a this plan relies on (verify in Task 0)

- `actions.RelocateMob(mob *mobs.Mob, from *rooms.Room, exitName string, dest *rooms.Room)` performs the relocation tail lifted from `mobcommands.Go` (remove, aggro cleanup, add, exit and entry lines, `SendTextToExits`, sounds, and the NPC party pull, which re-issues the exit as each idle member's own command, so in 4b every member pays its own step), computes its own entry-side exit name, and charges and detects NOTHING. `mobcommands.Go` calls it after its far-side lock gate.
- `actKeepDistance` is already a flee (4b does not touch it).

### Spec deviations this plan makes (each small, each stated)

1. The spec names both a function and its result type `EntryDetection`. Go forbids that; the result type is `EntryDetectionResult`.
2. A mob mover excludes its own NPC party from its observers, the parity twin of the player mover excluding its party (V5).
3. `MoveCharge.Never` is added so a walker can tell "tired, wait" from "cannot ever pay, give up" (spec 4b caller item 1).
4. `actions.MovePrice` is exported so the shipped-route measurement test reads the same arithmetic instead of recomputing it.

---

## File map

| File | Change | Responsibility |
|---|---|---|
| `internal/characters/character.go` | modify | two runtime AP settlement fields |
| `internal/characters/action_points.go` | create | `SettleActionPoints` |
| `internal/characters/action_points_test.go` | create | settlement tests |
| `internal/characters/pools.go` | modify | read-only `CanAffordCostFloat` |
| `internal/characters/pools_afford_float_test.go` | create | its test |
| `internal/mobs/mobs.go` | modify | spawn settles AP full |
| `internal/mobs/spawn_action_points_test.go` | create | spawn test |
| `internal/mobs/mobs_path.go` | modify | `PathQueue.Peek` |
| `internal/mobs/mobs_path_test.go` | create | Peek test |
| `internal/mobs/pack_roaming.go` | modify | `MovePackFollowers` takes a `canStep` gate |
| `internal/mobs/pack_roaming_test.go` | modify | new arg, tired follower test |
| `internal/actions/move.go` | create | `MovePrice`, `QuoteMove`, `ChargeMove`, `QuoteMobStep`, `TrainSearchOnMove`, `EntryDetection` |
| `internal/actions/move_test.go` | create | cost parity table |
| `internal/actions/move_detection_test.go` | create | four pairings, both directions |
| `internal/actions/move_search_test.go` | moved from `internal/usercommands/movement_search_test.go` | Search gate rate |
| `internal/actions/move_dark_name_leak_guard_test.go` | moved from `internal/usercommands/dark_name_leak_guard_test.go` | G2 on `move.go` |
| `internal/actions/move_seam_test.go` | create | G3 twin |
| `internal/usercommands/go.go` | modify | wrapper: charge after gates, shared detection |
| `internal/usercommands/go_test.go` | modify | G3 now asserts the move out |
| `internal/usercommands/go_charge_order_test.go` | create | M6 fix |
| `internal/mobcommands/go.go` | modify | wrapper: charge, detection, Search roll |
| `internal/mobcommands/go_charge_test.go` | create | mob pays, tired mob stays |
| `internal/mobcommands/wander.go` | modify | quote before counting, followers gated |
| `internal/mobcommands/wander_tired_test.go` | create | tired wanderer |
| `internal/hooks/NewRound_IdleMobs.go` | modify | walker extracted to `advanceMobPath`, waits when tired |
| `internal/hooks/NewRound_IdleMobs_patrol.go` | modify | a waiting tick is not a failure |
| `internal/hooks/mob_path_wait_test.go` | create | walker and patrol tests |
| `internal/behaviortree/actions_mob.go`, `actions_scout.go` | modify | quote, `Failure` when tired |
| `internal/behaviortree/move_quote_test.go` | create | BT tests |
| `internal/behaviortree/heard_callforhelp_test.go` | modify | fixture AP max |
| `modules/aicompanion/travel.go` | modify | quote before a step, no `Fails` for exhaustion, one mind line |
| `modules/aicompanion/travel_tired_test.go` | create | companion tests |
| `internal/usercommands/admin.mob.go` | modify | vitals line |
| `internal/usercommands/admin_mob_vitals_test.go` | create | its test |
| `internal/combat/contest_site_guard_test.go` | modify | G1 re-key |
| `messaging_surface_guard_test.go` | modify only if Task 6 step reports keys | G4 |
| `move_wrapper_guard_test.go` | create (repo root) | re-fork guard |
| `move_route_stamina_test.go` | create (repo root) | shipped-route measurement |
| `context.md` in `internal/{actions,characters,mobcommands,usercommands,hooks,mobs,behaviortree}` and `modules/aicompanion` | modify | docs |
| `docs/PATCH_NOTES.md`, `docs/README.md` | modify | patch note, new files |

---

## Drift found at execution (2026-09-29, master `3d9a44f5f`)

Master gained 4a (flee parity) and the baubles PR #175 (FinalTwist) since this
plan's facts were verified at `c061ca58f`. Every row of the facts table was
re-checked against `3d9a44f5f`. Signatures, call sites and behaviour all match;
only citations drifted. Nothing below breaks a task's design.

**Contract from 4a (Task 0 Steps 2-3): confirmed exactly.**
`actions.RelocateMob(mob *mobs.Mob, from *rooms.Room, exitName string, dest *rooms.Room)`
is defined once, at `internal/actions/relocate_mob.go:84`; `mobcommands/go.go`
calls it once, at line 127, after the far-side lock gate. `grep -n
"ChargeMove\|DeductActionPoints\|EntryDetection" internal/actions/relocate*.go
internal/actions/flee*.go` returns nothing (exit 1): `RelocateMob` charges and
detects nothing, as the contract requires. `mobcommands/go.go` is now 141
lines total (down from the pre-4a 106-280 range V19 describes): the NPC party
pull V19 placed inline is gone from this file, consistent with the contract's
own claim that `RelocateMob` absorbed it.

**Line-number-only drift (values, signatures and behaviour unchanged):**
- V13: `hooks.go:75` -> `:76` (`RegisterListener(events.NewTurn{}, ActionPoints)`).
- V14: `world.go:868` -> `:873` (`util.IncrementTurnCount()` call site).
- V17: config.yaml block shifted roughly +7 lines starting after
  `HiddenMoveStaminaMultiplier` (still `:900`, unchanged): `MobStaminaRegenPct`
  `:1193`->`:1200`, `MovementBaseStaminaCost` `:1208`->`:1215`,
  `MovementMaxStaminaCost` `:1210`->`:1217`, `MovementSearchTrainChance`
  `:1218`->`:1225`, `StaminaBase` `:1249`->`:1256`, `StaminaPerVitality`
  `:1250`->`:1257`, `StaminaPerWillpower` `:1251`->`:1258`. All values
  unchanged. `FlightMoveStaminaMult` Go default: `config.balance.combat.go`
  `:199-200` -> `:221-222`, value still 0.5.
- V18: `MobSkillTrainingCap` in config.yaml `:1500` -> `:1621`, value still 25;
  its declaration in `config.balance.go` `:476` -> `:484` (`MobSkillCap` legacy
  row now at `:479`).
- V25: `actGoToCallerRoom` `:293` -> `:294`; `actMoveTowardTracked` unchanged
  at `:169`; `actMove` unchanged at `:348`.
- V26: `advanceTravel`'s `go <exit>` issue `:193` -> `:192`; `stepTimeoutRounds`,
  the two `Fails >= 3` sites and the mind-line pattern are all at the exact
  cited lines, unchanged.
- V27: `aicompanion/loot.go`'s `mobCompanionFollow` `:254` -> `:230`;
  `witness_response.go:91` and `follow.go:301` unchanged exactly;
  `usercommands/go.go`'s charmed-mob-follow loop `:477-490` -> `:485-502`;
  ferry factor's stuck-recovery teleport (`moveFactorSilently`) `:311-330` ->
  starts `:326`, called from `issueWalkIfDue` at `:320`. `seeders/` in the
  plan's shorthand is `internal/seeders/` on disk, unchanged content.
- V31: `narrationViewpointRegistry`'s two `usercommands/go.go` keys
  `:1364-1365` -> `:1365-1366`; the walk-guard block `:1446-1515` ->
  `:1470-1477` (guard logic condensed, same two failure modes: unregistered
  new key, stale registered key).
- V37: `boot_smoke_test.go`'s `loadAllDataFiles` call cited at `:31-60` is
  actually at `:87` (and again at `:317`, `:361`); `main.go` `:398-401,1610`
  -> `:397,403,1614`.
- V9 (`GetMovementStaminaCost`, `resources.go:91-123`), V12 (`character.go:124`
  `ActionPoints int`, `:150` `ActionPointsMax`), and V20
  (`mobs_path.go` `PathQueue` methods, no `Peek`) verified at the exact cited
  lines: no drift.

**New since the facts table was written, not breaking any task:**
- `internal/hooks/NewRound_IdleMobs.go` gained an NPC-NPC idle-conversation
  chunk ("Chunk 3.6") between the patrol executor and the path-walker check
  (now around `:118-140`); the walker's own `mob.Path.Next()` call is still
  inside V21's cited `:140-196` range, now at `:165`. Task 6, which extracts
  the walker into `advanceMobPath`, needs to route around this block rather
  than through it; noted for that task, not acted on here.
- `internal/mobcommands/wander.go` now early-returns before the wander body
  for `mob.ScatterRounds > 0` and `mob.PackAlphaId > 0 && !mob.IsPackAlpha`
  (pack-roaming state, Stage 42.8) before reaching the `WanderCount++` /
  `mob.Command("go "+exit)` pair V23 describes, which is now at roughly
  `:100-114` instead of `:93-110`. `pack_roaming.go`'s `MovePackFollowers`
  (`:256-294`) is unchanged at the cited lines.

---

### Task 0: Branch and verify the 4a contract

**Files:** none

- [ ] **Step 1: Branch from master after 4a merged**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
git fetch origin
git checkout -b feature/movement-parity-4b origin/master
```

- [ ] **Step 2: Confirm the contract exists**

Run: `grep -n "func RelocateMob" internal/actions/*.go` and `grep -n "RelocateMob(" internal/mobcommands/go.go`
Expected: one definition with signature `func RelocateMob(mob *mobs.Mob, from *rooms.Room, exitName string, dest *rooms.Room)`, and one call in `mobcommands/go.go`.

Run: `grep -n "ChargeMove\|DeductActionPoints\|EntryDetection" internal/actions/relocate*.go internal/actions/flee*.go`
Expected: no output (grep exits 1). `RelocateMob` must not charge or detect; if it does, stop and report.

- [ ] **Step 3: Read the post-4a `mobcommands.Go` adjacent-exit block in full** (`internal/mobcommands/go.go`, from `if exitName != `` {` to its `return true, nil`). Task 7 edits it; note the exact lines around the far-side lock gate and the `actions.RelocateMob` call.

- [ ] **Step 4: Re-sync the skip-worktree config** per `dogmud-balance-config` if `_datafiles/config.yaml` changed on master since the checkout was last synced. Nothing in this plan edits it.

---

### Task 1: Character AP settlement and a read-only float affordability check

**Files:**
- Modify: `internal/characters/character.go:124`
- Create: `internal/characters/action_points.go`
- Create: `internal/characters/action_points_test.go`
- Modify: `internal/characters/pools.go` (after `ApplyCostFloatOrRefuse`, ends `:549`)
- Create: `internal/characters/pools_afford_float_test.go`

- [ ] **Step 1: Write the failing settlement tests**

`internal/characters/action_points_test.go`:

```go
package characters

import "testing"

func apChar(max int) *Character {
	c := &Character{}
	c.ActionPointsMax.Value = max
	return c
}

// A character never settled starts full: a fresh spawn, or a mob loaded from
// an instance file, must not start frozen at 0 (every live mob did before
// movement parity 4b).
func TestSettleActionPoints_FirstSettleFills(t *testing.T) {
	c := apChar(200)
	c.SettleActionPoints(1000)
	if c.ActionPoints != 200 {
		t.Fatalf("first settle: ActionPoints = %d, want 200", c.ActionPoints)
	}
	if !c.ActionPointsSettled || c.ActionPointsSettledTurn != 1000 {
		t.Fatalf("first settle must stamp the turn: settled=%v turn=%d", c.ActionPointsSettled, c.ActionPointsSettledTurn)
	}
}

// One point per elapsed turn, the player rate (hooks.ActionPoints).
func TestSettleActionPoints_AddsElapsedTurns(t *testing.T) {
	c := apChar(200)
	c.SettleActionPoints(1000)
	c.ActionPoints = 50
	c.SettleActionPoints(1030)
	if c.ActionPoints != 80 {
		t.Fatalf("30 turns later: ActionPoints = %d, want 80", c.ActionPoints)
	}
}

func TestSettleActionPoints_CapsAtMax(t *testing.T) {
	c := apChar(200)
	c.SettleActionPoints(10)
	c.ActionPoints = 190
	c.SettleActionPoints(1_000_000)
	if c.ActionPoints != 200 {
		t.Fatalf("long gap: ActionPoints = %d, want the cap 200", c.ActionPoints)
	}
}

// Settling twice at the same turn adds nothing: a quote followed by a charge
// in the same turn must not mint points.
func TestSettleActionPoints_SameTurnIsIdempotent(t *testing.T) {
	c := apChar(200)
	c.SettleActionPoints(500)
	c.ActionPoints = 40
	c.SettleActionPoints(500)
	c.SettleActionPoints(500)
	if c.ActionPoints != 40 {
		t.Fatalf("same turn: ActionPoints = %d, want 40", c.ActionPoints)
	}
}

// The turn counter restarts at 0 on a reboot. A stamp from the future is not
// a debt: the character refills.
func TestSettleActionPoints_CounterRestartRefills(t *testing.T) {
	c := apChar(200)
	c.SettleActionPoints(9000)
	c.ActionPoints = 3
	c.SettleActionPoints(5)
	if c.ActionPoints != 200 || c.ActionPointsSettledTurn != 5 {
		t.Fatalf("restart: ActionPoints = %d turn = %d, want 200 and 5", c.ActionPoints, c.ActionPointsSettledTurn)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/characters/ -run TestSettleActionPoints -count=1`
Expected: FAIL to compile, `c.SettleActionPoints undefined`.

- [ ] **Step 3: Add the fields and the method**

In `internal/characters/character.go`, directly after line 124 (`ActionPoints        int ...`), add:

```go
	ActionPointsSettled     bool   `yaml:"-"` // runtime: SettleActionPoints has run at least once (mobs only; movement parity 4b)
	ActionPointsSettledTurn uint64 `yaml:"-"` // runtime: the turn ActionPoints was last settled to
```

Create `internal/characters/action_points.go`:

```go
package characters

// SettleActionPoints brings a MOB's action points up to date at turn, lazily.
//
// Players regain one point per turn in hooks.ActionPoints, which loops the
// online users every turn so the {ap} prompt token stays live. Mobs never
// regained anything and nothing set them at spawn, so every live mob held 0
// (movement parity 4b, fact V15). Looping every mob instance at 20 turns a
// second to fix that would cost far more than it buys, so a mob instead
// records the turn it was last settled and adds the elapsed turns, at the
// player rate, whenever something is about to read or spend its points.
//
// Never settled, or a stamp from the future (the turn counter restarts at 0
// on a reboot), means full.
//
// NEVER call this for a player: the per-turn hook already credits them, and a
// settle would credit the same turns twice. actions.ChargeMove and
// actions.QuoteMove settle only non-player actors.
func (c *Character) SettleActionPoints(turn uint64) {
	max := c.ActionPointsMax.Value
	if !c.ActionPointsSettled || turn < c.ActionPointsSettledTurn {
		c.ActionPoints = max
	} else if elapsed := turn - c.ActionPointsSettledTurn; elapsed > 0 {
		if elapsed > uint64(max) {
			elapsed = uint64(max)
		}
		c.ActionPoints += int(elapsed)
	}
	if c.ActionPoints > max {
		c.ActionPoints = max
	}
	c.ActionPointsSettled = true
	c.ActionPointsSettledTurn = turn
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/characters/ -run TestSettleActionPoints -count=1`
Expected: PASS.

- [ ] **Step 5: Write the failing read-only affordability test**

`internal/characters/pools_afford_float_test.go`:

```go
package characters

import "testing"

// CanAffordCostFloat must give exactly ApplyCostFloatOrRefuse's verdict and
// change nothing. Movement parity 4b quotes a step before issuing it; a quote
// that disagreed with the charge would issue steps the charge then refuses.
func TestCanAffordCostFloat_AgreesWithTheChargeAndMutatesNothing(t *testing.T) {
	cases := []struct {
		name    string
		stamina int
		carry   float64
		amount  float64
	}{
		{"sub-one step on an empty pool banks, so it is affordable", 0, 0, 0.55},
		{"a whole point on an empty pool is not", 0, 0, 1.4},
		{"a whole point with one in the pool is", 1, 0, 1.4},
		{"carry tips a sub-one step over a whole point", 0, 0.6, 0.55},
		{"NaN is free", 0, 0, nanForTest()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			build := func() *Character {
				c := &Character{Stamina: tc.stamina}
				c.StaminaMax.Value = 100
				if tc.carry > 0 {
					c.costCarry = map[Pool]float64{PoolStamina: tc.carry}
				}
				return c
			}
			quoted := build()
			got := quoted.CanAffordCostFloat(PoolStamina, tc.amount)
			if quoted.Stamina != tc.stamina || quoted.costCarry[PoolStamina] != tc.carry {
				t.Fatalf("the quote mutated the character: stamina %d carry %v", quoted.Stamina, quoted.costCarry[PoolStamina])
			}
			want := build().ApplyCostFloatOrRefuse(PoolStamina, tc.amount)
			if got != want {
				t.Fatalf("CanAffordCostFloat = %v, ApplyCostFloatOrRefuse = %v", got, want)
			}
		})
	}
}

func nanForTest() float64 {
	zero := 0.0
	return zero / zero
}
```

- [ ] **Step 6: Run to verify it fails**

Run: `go test ./internal/characters/ -run TestCanAffordCostFloat -count=1`
Expected: FAIL to compile, `CanAffordCostFloat undefined`.

- [ ] **Step 7: Implement it**

In `internal/characters/pools.go`, directly after `ApplyCostFloatOrRefuse` (ends at `:549`), add:

```go
// CanAffordCostFloat reports, without changing anything, whether
// ApplyCostFloatOrRefuse would pay amount from pool right now.
//
// It exists because the carry is private: a caller deciding whether to ISSUE
// an action it will charge later (a mob walker choosing whether to take its
// next step, movement parity 4b) needs the charge's own verdict, and
// QuoteActionCost cannot price a step, whose cap applies after the hidden and
// mutation multipliers.
func (c *Character) CanAffordCostFloat(pool Pool, amount float64) bool {
	carry := 0.0
	if c.costCarry != nil {
		carry = c.costCarry[pool]
	}
	whole, _, valid := fractionalCost(carry, amount)
	if !valid || whole <= 0 {
		return true
	}
	return c.CanAfford(pool, whole)
}
```

- [ ] **Step 8: Run to verify it passes, then the package**

Run: `go test ./internal/characters/ -count=1`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
gofmt -l internal/characters/
git add internal/characters/character.go internal/characters/action_points.go internal/characters/action_points_test.go internal/characters/pools.go internal/characters/pools_afford_float_test.go
git commit -m "feat(characters): lazy mob action point settlement and a read-only float affordability check

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```
`gofmt -l` must print nothing before the commit.

---

### Task 2: Mob spawn starts full, and `PathQueue.Peek`

**Files:**
- Modify: `internal/mobs/mobs.go:670-672`
- Create: `internal/mobs/spawn_action_points_test.go`
- Modify: `internal/mobs/mobs_path.go`
- Create: `internal/mobs/mobs_path_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/mobs/spawn_action_points_test.go`:

```go
package mobs

import "testing"

// Every live mob held 0 action points (movement parity 4b, fact V15). A spawn
// now starts full and settled, so charging a step does not freeze it.
func TestSpawnStartsWithFullActionPoints(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	m := NewMobByIdFresh(MobId(1), 4242)
	if m == nil {
		t.Fatal("spawn returned nil")
	}
	defer DestroyInstance(m.InstanceId)

	if m.Character.ActionPointsMax.Value < 50 {
		t.Fatalf("fixture premise: ActionPointsMax = %d, Validate floors it at 50", m.Character.ActionPointsMax.Value)
	}
	if m.Character.ActionPoints != m.Character.ActionPointsMax.Value {
		t.Fatalf("spawn ActionPoints = %d, want full %d", m.Character.ActionPoints, m.Character.ActionPointsMax.Value)
	}
	if !m.Character.ActionPointsSettled {
		t.Fatal("spawn must stamp the settlement so the first charge does not refill twice")
	}
}
```

`internal/mobs/mobs_path_test.go`:

```go
package mobs

import "testing"

type peekStep struct {
	exit string
	room int
}

func (s peekStep) ExitName() string { return s.exit }
func (s peekStep) RoomId() int      { return s.room }
func (s peekStep) Waypoint() bool   { return false }

// Peek shows the next step and does NOT advance: a tired walker must be able
// to look at a step it will not take yet (movement parity 4b).
func TestPathQueuePeekDoesNotAdvance(t *testing.T) {
	var p PathQueue
	if p.Peek() != nil {
		t.Fatal("an empty queue peeks nil")
	}
	p.SetPath([]PathRoom{peekStep{"north", 2}, peekStep{"east", 3}})
	if got := p.Peek(); got == nil || got.RoomId() != 2 {
		t.Fatalf("Peek = %v, want the north step", got)
	}
	if p.Len() != 2 || p.Current() != nil {
		t.Fatalf("Peek advanced the queue: Len %d Current %v", p.Len(), p.Current())
	}
	if got := p.Next(); got.RoomId() != 2 {
		t.Fatalf("Next after Peek = %v, want the same step", got)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/mobs/ -run "TestSpawnStartsWithFullActionPoints|TestPathQueuePeekDoesNotAdvance" -count=1`
Expected: FAIL to compile, `p.Peek undefined` (fix that first, then the spawn test fails with `spawn ActionPoints = 0`).

- [ ] **Step 3: Implement**

`internal/mobs/mobs_path.go`, after `Next`:

```go
// Peek returns the next step without advancing the queue, or nil when the
// queue is empty. The path walker quotes a step before taking it, and a step
// it cannot afford yet must stay queued (movement parity 4b).
func (p PathQueue) Peek() PathRoom {
	if len(p.roomQueue) == 0 {
		return nil
	}
	return p.roomQueue[0]
}
```

`internal/mobs/mobs.go`, after line 672 (`mob.Character.Conviction = mob.Character.ConvictionMax.Value`):

```go
		// Movement parity 4b: a spawn starts with full action points, settled
		// at the current turn. Nothing else ever set a mob's points, so every
		// live mob held 0.
		mob.Character.SettleActionPoints(util.GetTurnCount())
```

- [ ] **Step 4: Run to verify they pass, then the package**

Run: `go test ./internal/mobs/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -l internal/mobs/
git add internal/mobs/mobs.go internal/mobs/spawn_action_points_test.go internal/mobs/mobs_path.go internal/mobs/mobs_path_test.go
git commit -m "feat(mobs): spawn with full action points; PathQueue.Peek

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: `actions/move.go` costs: `MovePrice`, `QuoteMove`, `ChargeMove`, `QuoteMobStep`, `TrainSearchOnMove`

**Files:**
- Create: `internal/actions/move.go`
- Create: `internal/actions/move_test.go`
- Move: `internal/usercommands/movement_search_test.go` to `internal/actions/move_search_test.go`

- [ ] **Step 1: Write the failing cost tests**

`internal/actions/move_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

const moveCliffBiome = "movetestcliff"

func seedMoveBiomes(t *testing.T) {
	t.Helper()
	t.Cleanup(rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		moveCliffBiome: {BiomeId: moveCliffBiome, MovementCost: 2.5},
	}))
}

// setupMover shapes a character the same way for both actor kinds.
type moverSetup struct {
	stamina, staminaMax, apMax int
	hidden                     bool
	overloaded                 bool
}

func (s moverSetup) apply(t *testing.T, c *characters.Character) {
	t.Helper()
	c.Stamina = s.stamina
	c.StaminaMax.Value = s.staminaMax
	c.ActionPointsMax.Value = s.apMax
	c.Stats.Strength.ValueAdj = 10
	if s.overloaded {
		c.Items = []items.Item{{ItemId: 99101, Spec: &items.ItemSpec{ItemId: 99101, Name: "anvil", Weight: 100000}}}
	}
	if s.hidden {
		c.Awareness = awareness.NewMachine()
		reason := state.TransitionReason{Trigger: "move_test"}
		require.NoError(t, c.Awareness.TransitionToConcealing(awareness.ConcealingData{}, reason))
		c.Awareness.ResolveConcealment(true, reason)
		require.True(t, c.IsHidden())
	}
}

func newMoveUser(t *testing.T, s moverSetup) Actor {
	t.Helper()
	u := users.NewTestUser(9601, "mover", "Mover", 0)
	u.Character = characters.New()
	s.apply(t, u.Character)
	u.Character.ActionPoints = s.apMax // players are credited by the per-turn hook, never settled
	return NewUserActor(u)
}

func newMoveMob(t *testing.T, s moverSetup) Actor {
	t.Helper()
	m := &mobs.Mob{InstanceId: 9602}
	m.Character = *characters.New()
	s.apply(t, &m.Character)
	return NewMobActor(m) // never settled: the first quote or charge fills it
}

// The parity table: the same character pays the same price whether a player
// or a mob is walking (movement parity 4b, owner ruling 2).
func TestChargeMove_PlayerAndMobPayTheSame(t *testing.T) {
	seedMoveBiomes(t)
	road := &rooms.Room{RoomId: 9610}
	cliff := &rooms.Room{RoomId: 9611, Biome: moveCliffBiome}

	cases := []struct {
		name        string
		setup       moverSetup
		dest        *rooms.Room
		wantRefusal MoveRefusal
		wantAP      int
	}{
		{"plain step", moverSetup{stamina: 100, staminaMax: 100, apMax: 200}, road, MoveOK, 10},
		{"cliff step", moverSetup{stamina: 100, staminaMax: 100, apMax: 200}, cliff, MoveOK, 10},
		{"hidden step", moverSetup{stamina: 100, staminaMax: 100, apMax: 200, hidden: true}, road, MoveOK, 10},
		{"over capacity costs 50 points", moverSetup{stamina: 100, staminaMax: 100, apMax: 200, overloaded: true}, road, MoveOK, 50},
		{"too tired", moverSetup{stamina: 100, staminaMax: 100, apMax: 5}, road, MoveRefuseTired, 10},
		{"too encumbered", moverSetup{stamina: 100, staminaMax: 100, apMax: 40, overloaded: true}, road, MoveRefuseEncumbered, 50},
		{"exhausted", moverSetup{stamina: 0, staminaMax: 100, apMax: 200}, cliff, MoveRefuseExhausted, 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			player := newMoveUser(t, tc.setup)
			mob := newMoveMob(t, tc.setup)

			pc := ChargeMove(player, tc.dest)
			mc := ChargeMove(mob, tc.dest)

			require.Equal(t, tc.wantRefusal, pc.Refusal, "player refusal")
			require.Equal(t, pc.Refusal, mc.Refusal, "mob refusal must match the player's")
			require.Equal(t, tc.wantAP, pc.ActionCost)
			require.Equal(t, pc.ActionCost, mc.ActionCost)
			require.InDelta(t, pc.StaminaCost, mc.StaminaCost, 1e-12)

			_, wantStamina := MovePrice(player.GetCharacter(), tc.dest)
			require.InDelta(t, wantStamina, pc.StaminaCost, 1e-12, "ChargeMove must price through MovePrice")

			if tc.wantRefusal == MoveOK {
				require.Equal(t, tc.setup.apMax-tc.wantAP, player.GetCharacter().ActionPoints)
				require.Equal(t, tc.setup.apMax-tc.wantAP, mob.GetCharacter().ActionPoints)
			}
			if tc.wantRefusal == MoveRefuseExhausted {
				require.Equal(t, tc.setup.apMax, player.GetCharacter().ActionPoints, "a stamina refusal refunds the points")
				require.Equal(t, tc.setup.apMax, mob.GetCharacter().ActionPoints, "a stamina refusal refunds the points")
			}
		})
	}
}

// Terrain and hidden multiply the stamina price (V9); a cliff costs five road
// steps and hidden triples it, until the 20-point cap.
func TestMovePrice_TerrainAndHidden(t *testing.T) {
	seedMoveBiomes(t)
	road := &rooms.Room{RoomId: 9610}
	cliff := &rooms.Room{RoomId: 9611, Biome: moveCliffBiome}
	plain := newMoveUser(t, moverSetup{stamina: 100, staminaMax: 100, apMax: 200})
	hidden := newMoveUser(t, moverSetup{stamina: 100, staminaMax: 100, apMax: 200, hidden: true})

	_, r := MovePrice(plain.GetCharacter(), road)
	_, c := MovePrice(plain.GetCharacter(), cliff)
	_, h := MovePrice(hidden.GetCharacter(), road)
	require.InDelta(t, 2.5, c/r, 1e-9, "a 2.5 biome costs 2.5 road steps")
	require.InDelta(t, 3.0, h/r, 1e-9, "hidden triples the price")
}

// A step the character can never pay, even rested, reports Never so a walker
// gives up instead of waiting forever.
func TestQuoteMove_NeverWhenThePriceExceedsTheWholePool(t *testing.T) {
	seedMoveBiomes(t)
	cliff := &rooms.Room{RoomId: 9611, Biome: moveCliffBiome}

	tired := newMoveMob(t, moverSetup{stamina: 0, staminaMax: 100, apMax: 200})
	q := QuoteMove(tired, cliff)
	require.Equal(t, MoveRefuseExhausted, q.Refusal)
	require.False(t, q.Never, "a rested mob could pay this step")

	// Stamina 0 so the quote refuses (at 1 the whole point of a 1.37 step is
	// affordable); a pool of 1 so no rest can ever cover it.
	frail := newMoveMob(t, moverSetup{stamina: 0, staminaMax: 1, apMax: 200})
	q = QuoteMove(frail, cliff)
	require.Equal(t, MoveRefuseExhausted, q.Refusal)
	require.True(t, q.Never, "a 1-stamina mob can never pay a cliff step")
}

// A quote spends nothing.
func TestQuoteMove_SpendsNothing(t *testing.T) {
	road := &rooms.Room{RoomId: 9610}
	mob := newMoveMob(t, moverSetup{stamina: 100, staminaMax: 100, apMax: 200})
	q := QuoteMove(mob, road)
	require.True(t, q.OK())
	require.Equal(t, 200, mob.GetCharacter().ActionPoints, "the quote settled the mob full and spent nothing")
	require.Equal(t, 100, mob.GetCharacter().Stamina)
}

// Mob points refill lazily at one per turn; a player's are never settled by
// the charge (the per-turn hook credits them).
func TestChargeMove_SettlesMobsOnly(t *testing.T) {
	road := &rooms.Room{RoomId: 9610}

	// The turn counter only moves in world.go, so it may sit at 0 in a test
	// binary; push it past 20 so "20 turns ago" is a real turn.
	for util.GetTurnCount() < 100 {
		util.IncrementTurnCount()
	}
	mob := newMoveMob(t, moverSetup{stamina: 100, staminaMax: 100, apMax: 200})
	mc := mob.GetCharacter()
	mc.SettleActionPoints(util.GetTurnCount() - 20)
	mc.ActionPoints = 0
	require.Equal(t, MoveOK, ChargeMove(mob, road).Refusal, "20 elapsed turns buy a 10-point step")
	require.Equal(t, 10, mc.ActionPoints)

	player := newMoveUser(t, moverSetup{stamina: 100, staminaMax: 100, apMax: 200})
	pc := player.GetCharacter()
	pc.ActionPoints = 5
	require.Equal(t, MoveRefuseTired, ChargeMove(player, road).Refusal, "a player is not settled by the charge")
	require.False(t, pc.ActionPointsSettled)
}
```

Add `"github.com/GoMudEngine/GoMud/internal/util"` to the test file's imports.

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/actions/ -run "TestChargeMove|TestMovePrice|TestQuoteMove" -count=1`
Expected: FAIL to compile, `undefined: ChargeMove`.

- [ ] **Step 3: Implement the cost half of `internal/actions/move.go`**

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Movement parity 4b. The price of one room step, and the hidden-detection
// contests on arrival, are shared by players and mobs here. The command
// wrappers (usercommands.Go, mobcommands.Go) keep their own gates, lock
// handling and narration and call these once, right before relocating.

// MoveRefusal says why a step was not paid for.
type MoveRefusal int

const (
	MoveOK MoveRefusal = iota
	MoveRefuseEncumbered
	MoveRefuseTired
	MoveRefuseExhausted
)

// MoveCharge reports one priced step.
type MoveCharge struct {
	Refusal     MoveRefusal
	ActionCost  int     // 10, or 50 over carry capacity
	StaminaCost float64 // fractional stamina price, banked through the carry
	Winded      bool    // paid, and stamina is now under a quarter of its reachable max
	Never       bool    // refused, and this actor could not pay the step even fully rested
}

// OK reports whether the step was (or, for a quote, would be) paid.
func (m MoveCharge) OK() bool { return m.Refusal == MoveOK }

// The hardcoded step prices (fact V2). Not knobs; the spec keeps them as they
// are.
const (
	moveActionCost           = 10
	moveEncumberedActionCost = 50
)

// MovePrice is the price of one step into dest for c: action points, and the
// fractional stamina cost. It reads the destination biome by NAME
// (rooms.GetBiome(dest.Biome)), exactly as the player path always has.
func MovePrice(c *characters.Character, dest *rooms.Room) (int, float64) {
	actionCost := moveActionCost
	if c.GetCarriedWeight() > c.CarryCapacity() {
		actionCost = moveEncumberedActionCost
	}
	terrain := 1.0
	if biome, _ := rooms.GetBiome(dest.Biome); biome != nil {
		terrain = biome.GetMovementCost()
	}
	stamina := c.GetMovementStaminaCost(terrain)
	if mutations.IsFlying(c.Mutations) {
		// Winged Flight glides over terrain: movement barely tires you.
		stamina *= float64(configs.GetBalanceConfig().FlightMoveStaminaMult)
	}
	return actionCost, stamina
}

// settleMoverActionPoints brings a mob's points up to date. Players are
// credited per turn by hooks.ActionPoints and must never be settled here.
func settleMoverActionPoints(actor Actor) {
	if actor.IsPlayer() {
		return
	}
	actor.GetCharacter().SettleActionPoints(util.GetTurnCount())
}

func apRefusal(actionCost int) MoveRefusal {
	if actionCost == moveEncumberedActionCost {
		return MoveRefuseEncumbered
	}
	return MoveRefuseTired
}

// QuoteMove prices a step without paying for it, for callers that must decide
// before issuing one (the path walker, wander, behaviour-tree steps, the AI
// companion). Settling a mob's points is bookkeeping, not a charge.
func QuoteMove(actor Actor, dest *rooms.Room) MoveCharge {
	c := actor.GetCharacter()
	settleMoverActionPoints(actor)
	ap, st := MovePrice(c, dest)
	q := MoveCharge{ActionCost: ap, StaminaCost: st}
	switch {
	case c.ActionPoints < ap:
		q.Refusal = apRefusal(ap)
		q.Never = ap > c.ActionPointsMax.Value
	case !c.CanAffordCostFloat(characters.PoolStamina, st):
		q.Refusal = MoveRefuseExhausted
		q.Never = st > float64(c.EffectivePoolMax(characters.PoolStamina))
	}
	return q
}

// ChargeMove pays for one step into dest: action points first, then stamina,
// refunding the points when stamina refuses. Wrappers call it AFTER their
// lock checks and any exit-message requeue, so a door that stays locked costs
// nothing and a requeued step is charged once (fact V4).
func ChargeMove(actor Actor, dest *rooms.Room) MoveCharge {
	c := actor.GetCharacter()
	settleMoverActionPoints(actor)
	ap, st := MovePrice(c, dest)
	m := MoveCharge{ActionCost: ap, StaminaCost: st}
	if !c.DeductActionPoints(ap) {
		m.Refusal = apRefusal(ap)
		m.Never = ap > c.ActionPointsMax.Value
		return m
	}
	if !c.ApplyCostFloatOrRefuse(characters.PoolStamina, st) {
		c.ActionPoints += ap
		m.Refusal = MoveRefuseExhausted
		m.Never = st > float64(c.EffectivePoolMax(characters.PoolStamina))
		return m
	}
	// EffectivePoolMax, not the raw max: current stamina is reserve-clamped,
	// so a raw denominator nags a reserved character at a full pool.
	m.Winded = c.Stamina < c.EffectivePoolMax(characters.PoolStamina)/4
	return m
}

// QuoteMobStep quotes the step a mob would take through exitName from the room
// it stands in. A step that does not resolve (no room, no such exit, no
// destination, or `home`) quotes as affordable, so the caller's existing
// handling of a bad step is unchanged.
func QuoteMobStep(mob *mobs.Mob, exitName string) MoveCharge {
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return MoveCharge{}
	}
	ex := FindExit(room, exitName)
	if !ex.Found {
		return MoveCharge{}
	}
	dest := rooms.LoadRoom(ex.RoomId)
	if dest == nil {
		return MoveCharge{}
	}
	return QuoteMove(NewMobActorInRoom(mob, room), dest)
}

// movementTrainsSearch reports whether this move should record a search use.
// Moved unchanged from usercommands/go.go by movement parity 4b; the long
// rationale lives in internal/actions/context.md ("Movement").
//
// A zero or negative MovementSearchTrainChance switches the feature off.
func movementTrainsSearch() bool {
	chance := float64(configs.GetBalanceConfig().MovementSearchTrainChance)
	if chance <= 0 {
		return false
	}
	// Resolving against 100,000 keeps a knob as small as 0.00001 meaningful.
	const resolution = 100000
	return util.Rand(resolution) < int(chance*resolution)
}

// TrainSearchOnMove is the rare Search training a completed step earns. Walking
// is not a contest, so won is always true; the rarity gate is the rule.
func TrainSearchOnMove(actor Actor) {
	if movementTrainsSearch() {
		actor.AwardResolved(true, actor.GetCharacter().CandidateFor(string(skills.Search)))
	}
}
```

Before deleting it in Task 5, copy the full doc comment of `usercommands.movementTrainsSearch` (`go.go:35-72`) into `internal/actions/context.md` under a new "Movement" heading in Task 10; the shortened comment above points there.

- [ ] **Step 4: Move the Search-gate test**

```bash
git mv internal/usercommands/movement_search_test.go internal/actions/move_search_test.go
```

Edit its first line from `package usercommands` to `package actions`. Nothing else changes: it calls `movementTrainsSearch()` and `configs.SetConfigForTest`, both available in `actions`.

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./internal/actions/ -run "TestChargeMove|TestMovePrice|TestQuoteMove|TestMovementTrainsSearch" -count=1`
Expected: PASS. `go build ./...` must also pass (usercommands still has its own `movementTrainsSearch` until Task 5).

- [ ] **Step 6: Commit**

```bash
gofmt -l internal/actions/
git add internal/actions/move.go internal/actions/move_test.go internal/actions/move_search_test.go internal/usercommands/movement_search_test.go
git commit -m "feat(actions): shared step price, quote and charge for players and mobs

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: `actions.EntryDetection`, with the G2 and G3 twins

**Files:**
- Modify: `internal/actions/move.go`
- Create: `internal/actions/move_detection_test.go`
- Move: `internal/usercommands/dark_name_leak_guard_test.go` to `internal/actions/move_dark_name_leak_guard_test.go`
- Create: `internal/actions/move_seam_test.go`

- [ ] **Step 1: Write the failing detection tests**

`internal/actions/move_detection_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/progression"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// pinDetectionKnobs removes the contest floor and gap compression so scores
// 1000 apart decide every roll (fact V35).
func pinDetectionKnobs(t *testing.T) {
	t.Helper()
	c := configs.GetConfig()
	c.Balance.ContestFloor = 0
	c.Balance.ContestGapSaturation = 0
	configs.SetConfigForTest(t, c)
}

func hide(t *testing.T, c *characters.Character) {
	t.Helper()
	if c.Awareness == nil {
		c.Awareness = awareness.NewMachine()
	}
	reason := state.TransitionReason{Trigger: "move_detection_test"}
	require.NoError(t, c.Awareness.TransitionToConcealing(awareness.ConcealingData{}, reason))
	c.Awareness.ResolveConcealment(true, reason)
	require.True(t, c.IsHidden())
}

// recordingMover wraps an Actor and records AwardResolved instead of paying it.
type recordingMover struct {
	Actor
	rec awardRecorder
}

func (r *recordingMover) AwardResolved(won bool, cands ...progression.Candidate) {
	r.rec.AwardResolved(won, cands...)
}

type detectWorld struct {
	dest *rooms.Room
}

func newDetectWorld(t *testing.T) *detectWorld {
	t.Helper()
	pinDetectionKnobs(t)
	dest := &rooms.Room{RoomId: 9800, Zone: "MoveDetect", Lamp: rooms.LampPtr(80)}
	t.Cleanup(rooms.SeedRoomsForTest(map[int]*rooms.Room{9800: dest}, map[string]*rooms.ZoneConfig{}))
	return &detectWorld{dest: dest}
}

// place puts a character of the given kind into dest and returns its actor
// and character. Player ids start at 9810, mob instances at 9850.
func (w *detectWorld) place(t *testing.T, kind string, id int, name string) (Actor, *characters.Character) {
	t.Helper()
	if kind == "player" {
		u := users.NewTestUser(id, name, name, 0)
		u.Character.RoomId = w.dest.RoomId
		existing := map[int]*users.UserRecord{}
		for _, pid := range w.dest.GetPlayers() {
			if p := users.GetByUserId(pid); p != nil {
				existing[pid] = p
			}
		}
		existing[id] = u
		t.Cleanup(users.SeedUsersForTest(existing))
		w.dest.AddPlayer(id)
		return NewUserActorInRoom(u, w.dest), u.Character
	}
	m := &mobs.Mob{InstanceId: id}
	m.Character = *characters.New()
	m.Character.Name = name
	m.Character.RoomId = w.dest.RoomId
	mobs.SetInstanceForTest(id, m)
	t.Cleanup(func() { mobs.SetInstanceForTest(id, nil) })
	w.dest.AddMob(id)
	return NewMobActorInRoom(m, w.dest), &m.Character
}

var detectPairings = []struct{ mover, other string }{
	{"player", "player"}, {"player", "mob"}, {"mob", "player"}, {"mob", "mob"},
}

func otherId(kind string) int {
	if kind == "player" {
		return 9811
	}
	return 9851
}

func moverId(kind string) int {
	if kind == "player" {
		return 9810
	}
	return 9850
}

// A clumsy sneaking mover walking in on a sharp observer is spotted: it stops
// sneaking and is no longer hidden. All four mover and observer pairings.
func TestEntryDetection_SneakingMoverIsSpotted(t *testing.T) {
	for _, p := range detectPairings {
		t.Run(p.mover+" sneaks past a "+p.other, func(t *testing.T) {
			w := newDetectWorld(t)
			_, obs := w.place(t, p.other, otherId(p.other), "Watcher")
			obs.Stats.Perception.ValueAdj = 1000
			mover, mc := w.place(t, p.mover, moverId(p.mover), "Sneak")
			mc.Stats.Dexterity.ValueAdj = 0
			hide(t, mc)
			mc.SetMiscData(`sneaking`, true)

			got := EntryDetection(mover, w.dest, true)

			require.False(t, got.StillSneaking)
			require.False(t, mc.IsHidden(), "a spotted sneaker is revealed")
			require.Nil(t, mc.GetMiscData(`sneaking`))
		})
	}
}

// A skilled sneaker walking in on a blind observer stays hidden.
func TestEntryDetection_SkilledSneakerSlipsIn(t *testing.T) {
	for _, p := range detectPairings {
		t.Run(p.mover+" sneaks past a "+p.other, func(t *testing.T) {
			w := newDetectWorld(t)
			_, obs := w.place(t, p.other, otherId(p.other), "Watcher")
			obs.Stats.Perception.ValueAdj = 0
			mover, mc := w.place(t, p.mover, moverId(p.mover), "Sneak")
			mc.Stats.Dexterity.ValueAdj = 1000
			hide(t, mc)

			got := EntryDetection(mover, w.dest, true)

			require.True(t, got.StillSneaking)
			require.True(t, mc.IsHidden())
		})
	}
}

// A sharp newcomer spots a clumsy hider and earns a winning Search award; a
// blind one misses a skilled hider and still earns a losing award (U10b-2:
// both outcomes fire).
func TestEntryDetection_NewcomerRollsForHiders(t *testing.T) {
	for _, p := range detectPairings {
		for _, sharp := range []bool{true, false} {
			name := p.mover + " walks in on a hidden " + p.other
			if !sharp {
				name += " and misses it"
			}
			t.Run(name, func(t *testing.T) {
				w := newDetectWorld(t)
				_, hc := w.place(t, p.other, otherId(p.other), "Lurker")
				hide(t, hc)
				inner, mc := w.place(t, p.mover, moverId(p.mover), "Newcomer")
				if sharp {
					mc.Stats.Perception.ValueAdj = 1000
					hc.Stats.Dexterity.ValueAdj = 0
				} else {
					mc.Stats.Perception.ValueAdj = 0
					hc.Stats.Dexterity.ValueAdj = 1000
				}
				mover := &recordingMover{Actor: inner}

				got := EntryDetection(mover, w.dest, false)

				require.False(t, got.StillSneaking)
				require.Equal(t, !sharp, hc.IsHidden(), "the hider is revealed exactly when spotted")
				require.Len(t, mover.rec.awards, 1, "one Search award per hidden occupant")
				require.Equal(t, sharp, mover.rec.awards[0].won)
				require.Equal(t, string(skills.Search), mover.rec.awards[0].cands[0].Skill)
			})
		}
	}
}

// The mover never rolls against itself: a hidden mob walking in with no one
// else present awards nothing.
func TestEntryDetection_SkipsTheMover(t *testing.T) {
	w := newDetectWorld(t)
	inner, mc := w.place(t, "mob", 9850, "Loner")
	hide(t, mc)
	mover := &recordingMover{Actor: inner}
	EntryDetection(mover, w.dest, false)
	require.Empty(t, mover.rec.awards)
	require.True(t, mc.IsHidden())
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/actions/ -run TestEntryDetection -count=1`
Expected: FAIL to compile, `undefined: EntryDetection`.

- [ ] **Step 3: Implement the detection half of `move.go`**

Add to the imports of `internal/actions/move.go`: `"fmt"`, `"github.com/GoMudEngine/GoMud/internal/combat"`, `"github.com/GoMudEngine/GoMud/internal/contest"`, `"github.com/GoMudEngine/GoMud/internal/messaging"`, `"github.com/GoMudEngine/GoMud/internal/parties"`, `"github.com/GoMudEngine/GoMud/internal/state"`, `"github.com/GoMudEngine/GoMud/internal/state/awareness"`, `"github.com/GoMudEngine/GoMud/internal/users"`. Then append:

```go
// EntryDetectionResult reports what arrival detection left behind.
type EntryDetectionResult struct {
	// StillSneaking is false once a sneaking mover has been spotted.
	StillSneaking bool
}

// EntryDetection runs the hidden-detection contests for a mover that has just
// arrived in dest, the same for a player or a mob (movement parity 4b, owner
// ruling 3: symmetric both ways).
//
// A sneaking mover is rolled against every observer in dest, players first
// (the one who spots it is told) and then mobs (silent), skipping the mover's
// own party. Once it is not sneaking, whether it never was or was just
// spotted, the mover rolls to spot every hidden player and mob in dest and
// earns a Search award on BOTH outcomes (U10b-2).
//
// Moved from usercommands.Go. The contests keep their shape exactly; lines to
// players go through each player's own actor and room lines through
// SendTextVisual, so a mob mover's lines differ only in its name colour.
func EntryDetection(mover Actor, dest *rooms.Room, sneaking bool) EntryDetectionResult {
	// The light is invariant across every occupant, so it is composed once.
	light := messaging.FixedLight(dest.LightLevel())

	if sneaking && sneakerSpotted(mover, dest, light) {
		mc := mover.GetCharacter()
		// Drive the Awareness FSM out of Hidden; the mirror cascade in
		// Awareness_Cascades.go clears the Hidden condition. Silent to the
		// mover: the condition's own end text is the signal.
		_ = mc.Awareness.TransitionToRevealing(
			state.TransitionReason{Trigger: awareness.TriggerObserverSearch})
		mc.SetMiscData(`sneaking`, nil)
		sneaking = false
	}

	if !sneaking {
		newcomerSpots(mover, dest, light)
	}

	return EntryDetectionResult{StillSneaking: sneaking}
}

// moverName is the mover's name in its identity colour.
func moverName(mover Actor) string {
	if mover.IsPlayer() {
		return fmt.Sprintf(`<ansi fg="username">%s</ansi>`, mover.GetName())
	}
	return fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, mover.GetName())
}

// moverAllies is the mover's party: allies do not expose a sneaker.
type moverAllies struct {
	users map[int]bool
	mobs  map[int]bool
}

func alliesOf(mover Actor) moverAllies {
	a := moverAllies{users: map[int]bool{}, mobs: map[int]bool{}}
	if mover.IsPlayer() {
		if p := parties.Get(mover.GetUserId()); p != nil {
			for _, uid := range p.GetMembers() {
				a.users[uid] = true
			}
		}
		return a
	}
	if p := parties.GetByMobInstanceId(mover.GetMobInstanceId()); p != nil {
		for _, member := range p.Members {
			if id := member.GetMobInstanceId(); id != 0 {
				a.mobs[id] = true
			}
		}
	}
	return a
}

// sneakerSpotted rolls a sneaking mover against dest's observers. The sneak
// score is computed per observer so a nightvision observer applies the right
// light modifier.
func sneakerSpotted(mover Actor, dest *rooms.Room, light messaging.RoomVisibility) bool {
	mc := mover.GetCharacter()
	allies := alliesOf(mover)

	for _, pId := range dest.GetPlayers() {
		if pId == mover.GetUserId() || allies.users[pId] {
			continue
		}
		p := users.GetByUserId(pId)
		if p == nil {
			continue
		}
		sneakScore := CalcSneakScoreVsObserver(mc, p.Character, light)
		observerScore := CalcDetectionScore(p.Character, dest)
		if !combat.RunContest(sneakScore, []contest.Entry{{Score: observerScore}}).Success {
			NewUserActor(p).SendText(messaging.CategorySystem, fmt.Sprintf(
				`%s slips into the room but you notice them.`, moverName(mover)))
			return true
		}
	}

	for _, mId := range dest.GetMobs() {
		if mId == mover.GetMobInstanceId() || allies.mobs[mId] {
			continue
		}
		m := mobs.GetInstance(mId)
		if m == nil {
			continue
		}
		sneakScore := CalcSneakScoreVsObserver(mc, &m.Character, light)
		observerScore := CalcDetectionScore(&m.Character, dest)
		if !combat.RunContest(sneakScore, []contest.Entry{{Score: observerScore}}).Success {
			return true
		}
	}

	return false
}

// newcomerSpots rolls the arriving mover to spot every hidden player and mob
// in dest. Neither side learns a name it cannot see.
func newcomerSpots(mover Actor, dest *rooms.Room, light messaging.RoomVisibility) {
	mc := mover.GetCharacter()
	// The newcomer now stands in dest: that is the light their eyes meet.
	observerScore := CalcDetectionScore(mc, dest)

	for _, pId := range dest.GetPlayers() {
		if pId == mover.GetUserId() {
			continue
		}
		hiddenP := users.GetByUserId(pId)
		if hiddenP == nil || !hiddenP.Character.IsHidden() {
			continue
		}
		hiddenScore := CalcSneakScoreVsObserver(hiddenP.Character, mc, light)
		success := combat.RunContest(observerScore, []contest.Entry{{Score: hiddenScore}}).Success
		if success {
			_ = hiddenP.Character.Awareness.TransitionToRevealing(
				state.TransitionReason{Trigger: awareness.TriggerObserverSearch})
			hiddenP.Character.SetMiscData(`sneaking`, nil)
			hider := NewUserActor(hiddenP)
			if messaging.CanSeeClearly(hiddenP.Character, dest) {
				hider.SendText(messaging.CategorySystem, fmt.Sprintf(
					`%s enters the room and notices you!`, moverName(mover)))
			} else {
				hider.SendText(messaging.CategorySystem,
					`Someone enters the room and notices you!`)
			}
			if messaging.CanSeeClearly(mc, dest) {
				mover.SendText(messaging.CategorySystem, fmt.Sprintf(
					`You notice <ansi fg="username">%s</ansi> lurking in the shadows.`,
					hiddenP.Character.Name))
			} else {
				mover.SendText(messaging.CategorySystem,
					`You notice someone lurking in the shadows.`)
			}
		}
		// U10b-2: the Search award fires on BOTH outcomes, full on a win and
		// partial on a resolved loss. Outside the success branch on purpose.
		mover.AwardResolved(success, mc.CandidateFor(string(skills.Search)))
	}

	for _, mId := range dest.GetMobs(rooms.FindAll) {
		if mId == mover.GetMobInstanceId() {
			continue
		}
		m := mobs.GetInstance(mId)
		if m == nil || !m.Character.IsHidden() {
			continue
		}
		hiddenScore := CalcSneakScoreVsObserver(&m.Character, mc, light)
		success := combat.RunContest(observerScore, []contest.Entry{{Score: hiddenScore}}).Success
		if success {
			_ = m.Character.Awareness.TransitionToRevealing(
				state.TransitionReason{Trigger: awareness.TriggerObserverSearch})
			// Spotting something is not the same as identifying it.
			if messaging.CanSeeClearly(mc, dest) {
				mover.SendText(messaging.CategorySystem, fmt.Sprintf(
					`You notice <ansi fg="mobname">%s</ansi> lurking in the shadows!`,
					m.Character.Name))
			} else {
				mover.SendText(messaging.CategorySystem,
					`You notice something lurking in the shadows!`)
			}
			// SendTextVisual: a sight event; each bystander gets the version
			// their eyes allow, or nothing.
			dest.SendTextVisual(messaging.CategorySystem, fmt.Sprintf(
				`%s spots <ansi fg="mobname">%s</ansi> hiding in the shadows!`,
				moverName(mover), m.Character.Name),
				mover.GetUserId())
		}
		mover.AwardResolved(success, mc.CandidateFor(string(skills.Search)))
	}
}
```

Both `sneakerSpotted` and `newcomerSpots` call `CalcDetectionScore` in their own bodies; keep it that way or `sight_penalty_guard_test.go` fails (fact V32).

- [ ] **Step 4: Run to verify they pass**

Run: `go test ./internal/actions/ -run TestEntryDetection -count=5`
Expected: PASS all five runs (the knobs make each roll deterministic in practice).

- [ ] **Step 5: Move G2 to `move.go`**

```bash
git mv internal/usercommands/dark_name_leak_guard_test.go internal/actions/move_dark_name_leak_guard_test.go
```

In the moved file make exactly these edits:
1. `package usercommands` becomes `package actions`.
2. `parser.ParseFile(fset, "go.go", nil, parser.ParseComments)` becomes `parser.ParseFile(fset, "move.go", nil, parser.ParseComments)`, and `require.NoError(t, err, "go.go must parse")` becomes `require.NoError(t, err, "move.go must parse")`.
3. The final message `"expected at least 3 name-tagged stealth lines in go.go, found %d — "` becomes `"expected at least 3 name-tagged stealth lines in move.go, found %d; "` (the lines moved with EntryDetection in movement parity 4b).
4. Rename the test to `TestEntryDetectionNeverNamesWhatYouCannotSee`.

If the moved file declares `nameTagged` or `posRange` and a name collides with an existing `actions` identifier, `go vet ./internal/actions/` says so; prefix the colliding helper with `move`.

- [ ] **Step 6: Add the G3 twin**

`internal/actions/move_seam_test.go`:

```go
package actions

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The hidden-detection Search awards moved from usercommands/go.go to
// EntryDetection (movement parity 4b). They must still fire through the
// progression seam on BOTH outcomes, passing the contest result, never a
// literal true and never OnSkillUse directly.
func TestEntryDetectionFiresSearchThroughTheSeamOnBothOutcomes(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	b, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "move.go"))
	require.NoError(t, err)
	src := string(b)

	require.NotContains(t, src, "OnSkillUse(string(skills.Search)",
		"move.go calls the OnSkillUse primitive directly; detection must fire through AwardResolved")
	require.Equal(t, 2, strings.Count(src, "mover.AwardResolved(success,"),
		"expected both hidden-detection sites (players and mobs) to pass the contest result as the won argument")
}
```

- [ ] **Step 7: Run the actions package**

Run: `go test ./internal/actions/ -count=1`
Expected: PASS. Then `go build ./... && go vet ./internal/actions/`.

- [ ] **Step 8: Commit**

```bash
gofmt -l internal/actions/
git add internal/actions/move.go internal/actions/move_detection_test.go internal/actions/move_dark_name_leak_guard_test.go internal/actions/move_seam_test.go internal/usercommands/dark_name_leak_guard_test.go
git commit -m "feat(actions): EntryDetection shared by player and mob movers

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: The player wrapper, the M6 fix, and the G1 and G3 re-keys

**Files:**
- Modify: `internal/usercommands/go.go`
- Create: `internal/usercommands/go_charge_order_test.go`
- Modify: `internal/usercommands/go_test.go:72-99`
- Modify: `internal/combat/contest_site_guard_test.go:73,359-372`

- [ ] **Step 1: Write the failing M6 test**

`internal/usercommands/go_charge_order_test.go`:

```go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/stretchr/testify/require"
)

// Movement parity 4b, owner ruling 2 on the open questions: walking into a
// locked door you cannot open used to cost the step's action points and
// stamina anyway, because the charge ran before the lock check. It now costs
// nothing.
func TestGo_ALockedDoorCostsNothing(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)

	orig, ok := room.Exits["north"]
	require.True(t, ok, "fixture premise: room 1 has a north exit")
	locked := orig
	locked.Lock = gamelock.Lock{Difficulty: 10}
	room.Exits["north"] = locked
	t.Cleanup(func() { room.Exits["north"] = orig })
	_ = exit.RoomExit{} // keep the import honest if the fixture type changes

	user.Character.ActionPoints = 100
	stamina := user.Character.Stamina

	handled, err := Go("north", user, room, 0)
	require.True(t, handled)
	require.NoError(t, err)
	require.Equal(t, 1, user.Character.RoomId, "the door stayed locked")
	require.Equal(t, 100, user.Character.ActionPoints, "a locked door costs no action points")
	require.Equal(t, stamina, user.Character.Stamina, "a locked door costs no stamina")

	user.Character.ActionPoints = 5
}
```

Run: `go test ./internal/usercommands/ -run TestGo_ALockedDoorCostsNothing -count=1`
Expected: FAIL, `a locked door costs no action points` (90 != 100).

- [ ] **Step 2: Rewrite the charge in `usercommands.Go`**

Replace `go.go:191-259` (from `actionCost := 10` through the closing brace of the winded `if`) with only the destination load:

```go
		destRoom := rooms.LoadRoom(goRoomId)
		if destRoom == nil {
			return false, fmt.Errorf(`room %d not found`, goRoomId)
		}
```

Then, directly after the exit-message requeue block (`if exitInfo.ExitMessage != `` && !flags.Has(events.CmdIsRequeue) { ... }`), insert:

```go
		// Movement parity 4b: the step is paid here, after the lock and the
		// exit-message requeue, so a door that stays locked costs nothing and
		// a requeued step is charged once, not twice. The price and the charge
		// are actions.ChargeMove, shared with mobs.
		charge := actions.ChargeMove(actions.NewUserActorInRoom(user, room), destRoom)
		switch charge.Refusal {
		case actions.MoveRefuseEncumbered:
			user.SendText(messaging.CategorySystem, "You're too encumbered to move (<ansi fg=\"command\">help encumbrance</ansi>)!")
			return true, nil
		case actions.MoveRefuseTired:
			user.SendText(messaging.CategorySystem, "You're too tired to move (slow down)!")
			mudlog.Debug("No ActionPoints", "AP", user.Character.ActionPoints, "Needed", charge.ActionCost)
			return true, nil
		case actions.MoveRefuseExhausted:
			user.SendText(messaging.CategorySystem, "You're too exhausted to move! Rest and recover your stamina.")
			return true, nil
		}
		if charge.Winded {
			user.SendText(messaging.CategorySystem, "<ansi fg=\"yellow\">You're feeling winded. Consider resting to recover your stamina.</ansi>")
		}
```

Delete the now-stale comment `// destRoom already loaded above for stamina calculation`.

- [ ] **Step 3: Replace the Search roll and the detection blocks**

Replace the `if movementTrainsSearch() { ... }` block (was `go.go:398-401`, with its U7/U10b comment above it) with:

```go
			// U7 Task 10: a completed move rarely trains search. Inside the
			// MoveToRoom success branch on purpose: a refused or locked move
			// never reaches here. Shared with mobs (movement parity 4b).
			actions.TrainSearchOnMove(actions.NewUserActorInRoom(user, destRoom))
```

Replace everything from the comment above `destRoomLight := messaging.FixedLight(destRoom.LightLevel())` (was `:545`) through the closing brace of the newcomer block (`if !isSneaking { ... }` ending at `:725`) with:

```go
			// Hidden detection on room entry, both directions: the sneaking
			// mover against the room, then the newcomer against the room's
			// hiders. Shared with mobs (movement parity 4b).
			isSneaking = actions.EntryDetection(actions.NewUserActorInRoom(user, destRoom), destRoom, isSneaking).StillSneaking
```

The following `if !isSneaking {` (room_enter, player_enter, ambush) is untouched.

- [ ] **Step 4: Delete the local Search gate**

Delete `movementTrainsSearch` and its doc comment (`go.go:35-82`). Run `go build ./internal/usercommands/` and remove each import the compiler reports unused (expected: `combat`, `contest`, `mutations`, `awareness`, possibly `characters`, `skills`, `state`, `util`; keep any the compiler does not name).

- [ ] **Step 5: Re-key G3**

In `internal/usercommands/go_test.go`, replace `TestGo_HiddenDetectionFiresThroughTheSeamOnBothOutcomes` (`:72-99`) with:

```go
// Movement parity 4b moved hidden detection and the movement Search roll to
// internal/actions/move.go (EntryDetection, TrainSearchOnMove). Their seam
// assertions live in actions/move_seam_test.go now. This pins that go.go did
// not grow a second copy: any Search award or detection contest here is a
// re-fork.
func TestGo_DetectionAndSearchLiveInActions(t *testing.T) {
	src := goSource(t)
	require.NotContains(t, src, "OnSkillUse(string(skills.Search)")
	require.NotContains(t, src, "AwardResolved(",
		"go.go awards progression itself again; detection and the movement Search roll live in actions")
	require.Contains(t, src, "actions.EntryDetection(",
		"go.go no longer calls the shared detection")
}
```

- [ ] **Step 6: Re-key G1**

In `internal/combat/contest_site_guard_test.go`, replace the row

```go
	"internal/usercommands/go.go:Go":                                         "U6b task 16 (hidden detection on room entry, four sites)",
```

with

```go
	"internal/actions/move.go:sneakerSpotted":                                "U6b task 16 (hidden detection on room entry: a sneaking mover against the room's players and mobs; moved from usercommands/go.go by movement parity 4b)",
	"internal/actions/move.go:newcomerSpots":                                 "U6b task 16 (hidden detection on room entry: the newcomer against hidden players and mobs; moved from usercommands/go.go by movement parity 4b)",
```

and add `"internal/actions/move.go",` to `legacyLiteralFiles` directly after `"internal/actions/shadow.go",`. Keep the `go.go` entry there (the file still exists and must stay free of the old literal).

- [ ] **Step 7: Run everything touched**

Run: `go test ./internal/usercommands/ ./internal/combat/ ./internal/actions/ -count=1`
Expected: PASS, including `TestGo_ALockedDoorCostsNothing`, `TestGo_TheSneakingWrapperSurvives`, `TestEveryContestSiteIsOwned`, `TestNoLegacySkillWeightLiteralSurvives`.

Run from the repo root: `go test -run "TestNarrationSitesMatchViewpointAudit|TestEveryRollSiteAppliesTheSightPenalty" -count=1 .`
Expected: PASS. If `TestNarrationSitesMatchViewpointAudit` reports unregistered `actions/move.go|...` keys, each is a line that moved verbatim from `go.go`: add one `narrationViewpointRegistry` row per key, verdict `verdictCorrect`, the viewpoints the report prints, reason `"moved from usercommands/go.go by movement parity 4b (EntryDetection); same event, same audiences"`. If it reports stale `usercommands/go.go|...` keys, confirm with `grep` that the literal left `go.go` in this task, then delete the row. Re-run until PASS.

- [ ] **Step 8: Commit**

```bash
gofmt -l internal/ modules/
git add internal/usercommands/go.go internal/usercommands/go_charge_order_test.go internal/usercommands/go_test.go internal/combat/contest_site_guard_test.go
git commit -m "refactor(usercommands): go charges after the lock gate and uses the shared move bodies

A locked door no longer costs a step, and an exit message no longer
charges twice.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```
Add `messaging_surface_guard_test.go` to the `git add` only if Step 7 changed it.

---

### Task 6: The mob wrapper pays and detects

**Files:**
- Modify: `internal/mobcommands/go.go` (post-4a adjacent-exit block)
- Create: `internal/mobcommands/go_charge_test.go`

- [ ] **Step 1: Write the failing test**

`internal/mobcommands/go_charge_test.go`:

```go
package mobcommands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Movement parity 4b, owner ruling 2: a mob pays the player's step price.
func TestMobGo_PaysForTheStep(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := getTestMobAndRoom(t)
	require.Contains(t, room.Exits, "north", "fixture premise: room 1 has a north exit")

	mob.Character.ActionPointsMax.Value = 200
	mob.Character.ActionPointsSettled = false // settle full on the charge
	mob.Character.StaminaMax.Value = 100
	mob.Character.Stamina = 100

	handled, err := Go("north", mob, room)
	require.True(t, handled)
	require.NoError(t, err)
	require.NotEqual(t, 1, mob.Character.RoomId, "the mob walked north")
	require.Equal(t, 190, mob.Character.ActionPoints, "the step cost 10 action points")

	mob.Character.RoomId = 1
	room.AddMob(mob.InstanceId)
}

// A mob that cannot pay stays put, silently.
func TestMobGo_TiredMobStays(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := getTestMobAndRoom(t)

	mob.Character.ActionPointsMax.Value = 5 // settles full at 5, below a 10-point step
	mob.Character.ActionPointsSettled = false
	mob.Character.StaminaMax.Value = 100
	mob.Character.Stamina = 100

	handled, err := Go("north", mob, room)
	require.True(t, handled)
	require.NoError(t, err)
	require.Equal(t, 1, mob.Character.RoomId, "a tired mob does not move")
}
```

Run: `go test ./internal/mobcommands/ -run TestMobGo_ -count=1`
Expected: FAIL: `TestMobGo_PaysForTheStep` reports ActionPoints 0 (nothing charged, nothing settled) and `TestMobGo_TiredMobStays` reports the mob moved.

- [ ] **Step 2: Wire the charge, the Search roll and detection**

In the post-4a `mobcommands.Go` adjacent-exit block, between the far-side lock gate (the `return true, nil` for a locked `enterFromExit`) and the `actions.RelocateMob(mob, room, exitName, destRoom)` call, insert:

```go
		// Movement parity 4b, owner ruling 2: a mob pays the player's step
		// price, after the lock gates, silently on refusal (a mob has no one
		// to tell).
		mover := actions.NewMobActorInRoom(mob, room)
		if !actions.ChargeMove(mover, destRoom).OK() {
			return true, nil
		}

		sneaking := mob.Character.IsHidden()
		if flag, ok := mob.Character.GetMiscData(`sneaking`).(bool); ok && flag {
			sneaking = true
		}
```

Directly after the `actions.RelocateMob(...)` call, insert:

```go
		// The rare Search roll and hidden detection on entry, both ways
		// (owner ruling 3), shared with players.
		arrived := actions.NewMobActorInRoom(mob, destRoom)
		actions.TrainSearchOnMove(arrived)
		actions.EntryDetection(arrived, destRoom, sneaking)
```

The numeric teleport (`go <roomId>` to a non-adjacent room) is NOT touched (owner ruling 3; follow-up filed). The `home` branch is not touched.

- [ ] **Step 3: Run**

Run: `go test ./internal/mobcommands/ -count=1`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
gofmt -l internal/mobcommands/
git add internal/mobcommands/go.go internal/mobcommands/go_charge_test.go
git commit -m "feat(mobcommands): mobs pay for each step and roll hidden detection on entry

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: The path walker waits, and a waiting patrol is not failing

**Files:**
- Modify: `internal/hooks/NewRound_IdleMobs.go:138-196`
- Modify: `internal/hooks/NewRound_IdleMobs_patrol.go:174-181`
- Create: `internal/hooks/mob_path_wait_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/hooks/mob_path_wait_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/require"
)

type walkerTestStep struct {
	exit string
	room int
}

func (s walkerTestStep) ExitName() string { return s.exit }
func (s walkerTestStep) RoomId() int      { return s.room }
func (s walkerTestStep) Waypoint() bool   { return false }

const walkerCliff = "walkertestcliff"

// walkerWorld: room 7101 with a north exit to 7102, a 2.5 cliff.
func walkerWorld(t *testing.T, stamina, staminaMax int) *mobs.Mob {
	t.Helper()
	t.Cleanup(rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		walkerCliff: {BiomeId: walkerCliff, MovementCost: 2.5},
	}))
	from := &rooms.Room{RoomId: 7101, Exits: map[string]exit.RoomExit{"north": {RoomId: 7102}}}
	to := &rooms.Room{RoomId: 7102, Biome: walkerCliff}
	t.Cleanup(rooms.SeedRoomsForTest(map[int]*rooms.Room{7101: from, 7102: to}, map[string]*rooms.ZoneConfig{}))

	m := &mobs.Mob{InstanceId: 7150}
	m.Character = *characters.New()
	m.Character.RoomId = 7101
	m.Character.ActionPointsMax.Value = 200
	m.Character.StaminaMax.Value = staminaMax
	m.Character.Stamina = stamina
	m.Path.SetPath([]mobs.PathRoom{walkerTestStep{"north", 7102}})
	t.Cleanup(func() { events.DrainQueuedInputsForTest(7150) })
	return m
}

// A tired mob keeps its path and issues nothing; it tries again next round.
func TestAdvanceMobPath_TiredMobWaits(t *testing.T) {
	m := walkerWorld(t, 0, 100)
	require.True(t, advanceMobPath(m), "a waiting mob is busy with its path")
	require.Equal(t, 1, m.Path.Len(), "the step stays queued")
	require.Nil(t, m.Path.Current(), "Next was not called")
	require.Empty(t, events.InspectQueuedInputForTest(7150, "north"))
	require.True(t, mobPathStepWaiting(m))
}

// A rested mob takes the step.
func TestAdvanceMobPath_RestedMobSteps(t *testing.T) {
	m := walkerWorld(t, 100, 100)
	require.True(t, advanceMobPath(m))
	require.Equal(t, 0, m.Path.Len())
	require.NotEmpty(t, events.InspectQueuedInputForTest(7150, "north"))
	require.False(t, mobPathStepWaiting(m))
}

// A step the mob could never pay clears the path, handing the mob to the
// schedule and patrol fallbacks instead of parking it forever.
func TestAdvanceMobPath_NeverAffordableClears(t *testing.T) {
	m := walkerWorld(t, 0, 1) // empty, and a pool of 1 can never cover a 1.37 cliff step
	require.False(t, advanceMobPath(m), "a cleared path leaves the mob idle")
	require.Equal(t, 0, m.Path.Len())
	require.Empty(t, events.InspectQueuedInputForTest(7150, "north"))
}

// A patrol mob resting mid-path does not count a failed path attempt, so a
// long rest never trips the home fallback.
func TestApplyPatrolPlan_WaitingIsNotFailing(t *testing.T) {
	registerTestPatrol(t)
	m := walkerWorld(t, 0, 100)
	plan := patrolTickPlan(m, "test_patrol")
	require.True(t, plan.WantsPath, "fixture premise: 7101 is not a waypoint")
	applyPatrolPlan(m, plan, "test_patrol")
	if got := m.Character.GetMiscData("patrol_path_fail_count"); got != nil {
		require.Equal(t, 0, got.(int))
	}
}
```

Run: `go test ./internal/hooks/ -run "TestAdvanceMobPath|TestApplyPatrolPlan_WaitingIsNotFailing" -count=1`
Expected: FAIL to compile, `undefined: advanceMobPath`.

- [ ] **Step 2: Extract and change the walker**

In `internal/hooks/NewRound_IdleMobs.go`, replace the whole block from `// Check whether they are currently in the middle of a path, or have one waiting to start.` (`:138`) through the closing brace before `events.AddToQueue(events.MobIdle{...})` (`:194`) with:

```go
		// Check whether they are currently in the middle of a path, or have one waiting to start.
		// This comes after checks for whether they are currently in a conversation, or in combat, etc.
		if advanceMobPath(mob) {
			continue
		}
```

Append to the same file:

```go
// advanceMobPath moves a mob one step along its path. It returns true when the
// mob is busy with its path this round (a step issued, a re-path queued, or a
// tired wait) and the caller must not treat it as idle.
//
// Movement parity 4b: the next step is quoted before it is taken. A mob that
// cannot pay yet keeps its path and waits; before, Next() advanced the queue
// first, so a step the mob never took forced a re-path on the next tick. A
// step it could never pay clears the path, which hands the mob to the
// schedule and patrol fallbacks rather than parking it forever.
func advanceMobPath(mob *mobs.Mob) bool {
	currentStep := mob.Path.Current()
	if currentStep == nil && mob.Path.Len() == 0 {
		return false
	}

	// If their currentStep isn't actually the room they are in, they've
	// somehow been moved. Recalculate a new path.
	if currentStep != nil && currentStep.RoomId() != mob.Character.RoomId {
		reDoWaypoints := mob.Path.Waypoints()
		if len(reDoWaypoints) > 0 {
			newCommand := `pathto`
			for _, wpInt := range reDoWaypoints {
				newCommand += ` ` + strconv.Itoa(wpInt)
			}
			mob.Command(newCommand)
			return true
		}
		// if we were unable to come up with a new path, send them home.
		mob.Command(`pathto home`)
		return true
	}

	if nextStep := mob.Path.Peek(); nextStep != nil {
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			if exitInfo, ok := room.Exits[nextStep.ExitName()]; ok && exitInfo.RoomId == nextStep.RoomId() {
				quote := actions.QuoteMobStep(mob, nextStep.ExitName())
				if !quote.OK() && !quote.Never {
					return true // tired: keep the path, try again next round
				}
				if quote.OK() {
					mob.Path.Next()
					mob.Command(nextStep.ExitName())
					// Stage 2 caravan: pace caravan crews a shade slower than
					// default mob walking. The noop pushes lastCommandTurn
					// forward so the next path step waits ~1.5s.
					for _, g := range mob.Groups {
						if g == "caravan" {
							mob.Command("noop", 1.5)
							break
						}
					}
					return true
				}
			}
		}
	}

	mob.Path.Clear()
	if mob.HomeRoomId == mob.Character.RoomId {
		mob.WanderCount = 0
	}
	return false
}

// mobPathStepWaiting reports whether the mob's next path step is one it
// cannot afford yet but could once rested: the walker waits on it. The patrol
// executor reads it so a rest does not count as a failed path.
func mobPathStepWaiting(mob *mobs.Mob) bool {
	next := mob.Path.Peek()
	if next == nil {
		return false
	}
	q := actions.QuoteMobStep(mob, next.ExitName())
	return !q.OK() && !q.Never
}
```

Add `"github.com/GoMudEngine/GoMud/internal/actions"` to the file's imports.

- [ ] **Step 3: Patrol counter**

In `internal/hooks/NewRound_IdleMobs_patrol.go`, in `case plan.WantsPath:`, after the `if mob.Path.Len() == 0 && mob.Path.Current() == nil { ... }` block and before `fails := ...`, insert:

```go
		// Movement parity 4b: a mob resting mid-path because it cannot pay
		// its next step has not failed to path. Counting the rest would trip
		// the home fallback on any long enough rest.
		if mobPathStepWaiting(mob) {
			return
		}
```

Schedules need no change: they count only a newly queued `pathto` (fact V22), and a waiting mob has a path in flight.

- [ ] **Step 4: Run**

Run: `go test ./internal/hooks/ -count=1`
Expected: PASS (existing patrol tests included: their mobs have no path, so `Peek` is nil and nothing waits).

- [ ] **Step 5: Commit**

```bash
gofmt -l internal/hooks/
git add internal/hooks/NewRound_IdleMobs.go internal/hooks/NewRound_IdleMobs_patrol.go internal/hooks/mob_path_wait_test.go
git commit -m "feat(hooks): a tired mob waits on its path instead of re-pathing or failing its patrol

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 8: Wander, pack followers, behaviour-tree steps

**Files:**
- Modify: `internal/mobs/pack_roaming.go:253-295`
- Modify: `internal/mobs/pack_roaming_test.go` (every `MovePackFollowers(` call)
- Modify: `internal/mobcommands/wander.go:93-110`
- Create: `internal/mobcommands/wander_tired_test.go`
- Modify: `internal/behaviortree/actions_mob.go:293-318,348-359`, `internal/behaviortree/actions_scout.go:169-213`
- Create: `internal/behaviortree/move_quote_test.go`
- Modify: `internal/behaviortree/heard_callforhelp_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/mobs/pack_roaming_test.go`:

```go
// Movement parity 4b: each follower pays its own step. One that cannot stays
// behind and leaves the pack, rather than counting a wander it never walked.
func TestMovePackFollowers_TiredFollowerStaysBehind(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	alpha := &Mob{
		MobId: 1, InstanceId: 200, IsPackAlpha: true,
		Groups:    []string{"canine"},
		Character: characters.Character{Name: "Alpha Wolf", RoomId: 10},
	}
	follower := &Mob{
		MobId: 1, InstanceId: 201, PackAlphaId: 200,
		MaxWander: -1, WanderCount: 0,
		Groups:    []string{"canine"},
		Character: characters.Character{Name: "Tired Wolf", RoomId: 10},
	}
	mobInstances[200] = alpha
	mobInstances[201] = follower

	MovePackFollowers(alpha, "north", []int{200, 201}, func(*Mob) bool { return false })

	assert.Equal(t, 0, follower.PackAlphaId, "a follower that cannot step leaves the pack")
	assert.Equal(t, 0, follower.WanderCount, "a step never taken is not counted")
}
```

and add `, nil` as the fourth argument to every existing `MovePackFollowers(` call in that file.

`internal/mobcommands/wander_tired_test.go`:

```go
package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/require"
)

// Movement parity 4b: a tired wanderer does not count a wander it cannot
// walk, so it is not sent home early for steps it never took. The rested
// control proves the fixture reaches the counting branch at all (Wander only
// counts a step into the mob's own zone).
func TestWander_TiredMobDoesNotCount(t *testing.T) {
	for _, tc := range []struct {
		name      string
		apMax     int
		wantCount int
	}{
		{"rested control counts the step", 200, 1},
		{"tired mob does not", 5, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := seedAllRegistries()
			defer cleanup()
			mob, room := getTestMobAndRoom(t)

			north, ok := room.Exits["north"]
			require.True(t, ok, "fixture premise: room 1 has a north exit")
			dest := rooms.LoadRoom(north.RoomId)
			require.NotNil(t, dest)
			origExits := room.Exits
			room.Exits = map[string]exit.RoomExit{"north": north}
			t.Cleanup(func() { room.Exits = origExits })
			mob.Character.Zone = dest.Zone

			mob.MaxWander = -1
			mob.WanderCount = 0
			mob.Character.ActionPointsMax.Value = tc.apMax
			mob.Character.ActionPointsSettled = false
			mob.Character.StaminaMax.Value = 100
			mob.Character.Stamina = 100

			handled, err := Wander("", mob, room)
			require.True(t, handled)
			require.NoError(t, err)
			require.Equal(t, tc.wantCount, mob.WanderCount)
		})
	}
}
```

`internal/behaviortree/move_quote_test.go`:

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/require"
)

func tiredStepWorld(t *testing.T, apMax int) *mobs.Mob {
	t.Helper()
	room1 := &rooms.Room{RoomId: 1, Zone: "TestZone"}
	room2 := &rooms.Room{RoomId: 2, Zone: "TestZone", Exits: map[string]exit.RoomExit{"south": {RoomId: 1}}}
	t.Cleanup(rooms.SeedRoomsForTest(map[int]*rooms.Room{1: room1, 2: room2}, nil))

	caller := &mobs.Mob{InstanceId: 100, Character: characters.Character{RoomId: 1, Health: 50}}
	self := &mobs.Mob{InstanceId: 91101}
	self.Character.Name = "stepper"
	self.Character.RoomId = 2
	self.Character.Stamina = 100
	self.Character.StaminaMax.Value = 100
	self.Character.ActionPointsMax.Value = apMax
	self.Character.Conditions = conditions.New()
	t.Cleanup(mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{100: caller, 91101: self}))
	t.Cleanup(func() { events.DrainQueuedInputsForTest(91101) })
	return self
}

// Movement parity 4b: a single-step node quotes first and FAILS when the mob
// cannot pay, so a selector falls through to its next child.
func TestSingleStepNodesFailWhenTired(t *testing.T) {
	cases := []struct {
		name string
		run  func(ctx *EvalContext) Result
	}{
		{"go_to_caller_room", func(ctx *EvalContext) Result { return actGoToCallerRoom(nil, ctx) }},
		{"move", func(ctx *EvalContext) Result {
			return actMove(map[string]any{"direction": "south"}, ctx)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name+" tired", func(t *testing.T) {
			tiredStepWorld(t, 5)
			ctx := &EvalContext{InstanceId: 91101, Event: EventContext{MobId: 100}}
			require.Equal(t, Failure, tc.run(ctx))
			require.Empty(t, events.InspectQueuedInputForTest(91101, "go "))
		})
		t.Run(tc.name+" rested", func(t *testing.T) {
			tiredStepWorld(t, 200)
			ctx := &EvalContext{InstanceId: 91101, Event: EventContext{MobId: 100}}
			require.Equal(t, Success, tc.run(ctx))
			require.NotEmpty(t, events.InspectQueuedInputForTest(91101, "go "))
		})
	}
}
```

Before running, confirm `EvalContext` field names with `grep -n "type EvalContext struct" -A12 internal/behaviortree/*.go`; if `Event` or `InstanceId` differ, use the real names (the action bodies read `ctx.InstanceId` and `ctx.Event.MobId`, so they should match).

Run: `go test ./internal/mobs/ ./internal/mobcommands/ ./internal/behaviortree/ -run "TestMovePackFollowers|TestWander_TiredMobDoesNotCount|TestSingleStepNodesFailWhenTired" -count=1`
Expected: FAIL (compile error on the new `MovePackFollowers` argument; then the wander and BT tired cases fail).

- [ ] **Step 2: `MovePackFollowers` gains a step gate**

In `internal/mobs/pack_roaming.go`, change the signature and doc:

```go
// MovePackFollowers moves all followers of the given alpha mob through
// the specified exit. Called after the alpha successfully moves.
// Caller must check PackRoamingEnabled() before calling.
//
// canStep reports whether a follower can pay for the step (movement parity
// 4b). It is a parameter because this package cannot import internal/actions,
// which prices steps. nil means every follower can. A follower that cannot
// stays behind and leaves the pack, exactly as one over its wander budget
// does.
func MovePackFollowers(alphaMob *Mob, exitName string, oldRoomMobIds []int, canStep func(*Mob) bool) {
```

and immediately before `mob.WanderCount++` at the end of the loop body insert:

```go
		if canStep != nil && !canStep(mob) {
			mob.PackAlphaId = 0
			continue
		}
```

- [ ] **Step 3: `Wander` quotes before counting**

In `internal/mobcommands/wander.go`, replace the body of `if !restrictZone || r.Zone == mob.Character.Zone {` with:

```go
				// Movement parity 4b: a step the wanderer cannot pay is not
				// taken and not counted, and drags no followers.
				if !actions.QuoteMobStep(mob, exitName).OK() {
					return true, nil
				}

				// Stage 42.8: Capture mob list before alpha moves (for follower movement)
				var preMoveRoomMobs []int
				if mob.IsPackAlpha && mobs.PackRoamingEnabled() {
					preMoveRoomMobs = room.GetMobs(rooms.FindAll)
				}

				mob.WanderCount++
				mob.Command(fmt.Sprintf("go %s", exitName))

				// Stage 42.8: Move pack followers through the same exit. Each
				// pays its own step.
				if mob.IsPackAlpha && len(preMoveRoomMobs) > 0 {
					mobs.MovePackFollowers(mob, exitName, preMoveRoomMobs, func(f *mobs.Mob) bool {
						return actions.QuoteMobStep(f, exitName).OK()
					})
				}
```

Add `"github.com/GoMudEngine/GoMud/internal/actions"` to its imports.

- [ ] **Step 4: Behaviour-tree single steps**

In `actGoToCallerRoom` (`actions_mob.go`), replace

```go
		if info.RoomId == caller.Character.RoomId {
			self.Command(fmt.Sprintf("go %s", exitName))
			return Success
		}
```

with

```go
		if info.RoomId == caller.Character.RoomId {
			// Movement parity 4b: a tired mob fails the node so a selector
			// falls through, rather than issuing a step it cannot pay.
			if !actions.QuoteMobStep(self, exitName).OK() {
				return Failure
			}
			self.Command(fmt.Sprintf("go %s", exitName))
			return Success
		}
```

In `actMove`, before `mob.Command("go " + direction)` insert:

```go
	// Movement parity 4b: fail when the mob cannot pay for the step.
	if !actions.QuoteMobStep(mob, direction).OK() {
		return Failure
	}
```

In `actMoveTowardTracked` (`actions_scout.go`), before `mob.Command("go " + dir)` insert:

```go
	// Movement parity 4b: fail when the mob cannot pay for the step.
	if !actions.QuoteMobStep(mob, dir).OK() {
		return Failure
	}
```

Both files already import `actions` (fact V25); if either does not, add it.

- [ ] **Step 5: Give the call-for-help fixtures action points**

In `internal/behaviortree/heard_callforhelp_test.go`, in both tests, after `responder.Character.Conviction = 500` add:

```go
	responder.Character.ActionPointsMax.Value = 200 // movement parity 4b: steps cost action points
```

- [ ] **Step 6: Run**

Run: `go test ./internal/mobs/ ./internal/mobcommands/ ./internal/behaviortree/ -count=1`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
gofmt -l internal/
git add internal/mobs/pack_roaming.go internal/mobs/pack_roaming_test.go internal/mobcommands/wander.go internal/mobcommands/wander_tired_test.go internal/behaviortree/actions_mob.go internal/behaviortree/actions_scout.go internal/behaviortree/move_quote_test.go internal/behaviortree/heard_callforhelp_test.go
git commit -m "feat(mobs): wanderers, pack followers and single-step behaviours quote a step before taking it

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 9: AI companion travel rests instead of poisoning its map

**Files:**
- Modify: `modules/aicompanion/travel.go`
- Create: `modules/aicompanion/travel_tired_test.go`

- [ ] **Step 1: Write the failing test**

`modules/aicompanion/travel_tired_test.go`:

```go
package aicompanion

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func withStepAffordable(t *testing.T, ok bool) {
	t.Helper()
	prev := stepAffordable
	stepAffordable = func(*mobs.Mob, string) bool { return ok }
	t.Cleanup(func() { stepAffordable = prev })
}

func tiredLines(c *controller) int {
	n := 0
	for _, l := range c.mind.RecentLines {
		if strings.Contains(l.Text, `too tired to go on`) {
			n++
		}
	}
	return n
}

// Movement parity 4b: a tired companion does not issue the step, does not
// start the step clock, writes one line in its mind (not one a round), and
// takes the step once rested.
func TestTiredCompanionRestsOnItsTrip(t *testing.T) {
	w := newConsentWorld(t, true)
	here := w.her.Character.RoomId
	w.c.travel = &travelPlan{Dest: 9, DestName: `the mill`, Purpose: `errand`,
		Steps: []step{{Exit: `north`, To: 9}}, FromRoom: here}
	t.Cleanup(func() { events.DrainQueuedInputsForTest(w.her.InstanceId) })

	withStepAffordable(t, false)
	w.m.advanceTravel(w.c, w.her, w.owner, 50)
	w.m.advanceTravel(w.c, w.her, w.owner, 51)

	if w.c.travel == nil || w.c.travel.Expect != 0 {
		t.Fatalf("a tired companion must not start the step clock: %+v", w.c.travel)
	}
	if got := events.InspectQueuedInputForTest(w.her.InstanceId, `go `); got != `` {
		t.Fatalf("a tired companion issued %q", got)
	}
	if n := tiredLines(w.c); n != 1 {
		t.Fatalf("expected exactly one resting line in her mind, got %d", n)
	}

	withStepAffordable(t, true)
	w.m.advanceTravel(w.c, w.her, w.owner, 52)
	if got := events.InspectQueuedInputForTest(w.her.InstanceId, `go `); got == `` {
		t.Fatal("rested, she takes the step")
	}
}

// A step issued while she could pay, that then timed out because she could
// not (stamina changed in between), is a rest, not a bad exit: no Fails.
func TestTimedOutTiredStepDoesNotMarkTheExit(t *testing.T) {
	w := newConsentWorld(t, true)
	here := w.her.Character.RoomId
	w.c.mind.Map = map[int]*RoomRecord{here: {Title: `A Lane`, Exits: map[string]*ExitRecord{`north`: {To: 9}}}}
	w.c.travel = &travelPlan{Dest: 9, DestName: `the mill`, Purpose: `errand`,
		Steps: []step{{Exit: `north`, To: 9}}, FromRoom: here, Expect: 9, Issued: 10}

	withStepAffordable(t, false)
	w.m.advanceTravel(w.c, w.her, w.owner, 10+stepTimeoutRounds)

	if f := w.c.mind.Map[here].Exits[`north`].Fails; f != 0 {
		t.Fatalf("exhaustion wrote %d failure(s) against the exit", f)
	}
	if w.c.travel == nil || w.c.travel.Expect != 0 {
		t.Fatalf("the step must be re-quoted next round: %+v", w.c.travel)
	}
}
```

Run: `go test ./modules/aicompanion/ -run "TestTiredCompanion|TestTimedOutTiredStep" -count=1`
Expected: FAIL to compile, `undefined: stepAffordable`.

- [ ] **Step 2: Implement**

In `modules/aicompanion/travel.go`:

Add `Resting bool // already told her mind she stopped to rest on this step` to `travelPlan` after `Errand`.

Add after the `const` block:

```go
// tooTiredLine is what a companion remembers when it stops to rest mid-trip.
const tooTiredLine = `You are too tired to go on, and stop to catch your breath.`

// stepAffordable reports whether the companion can pay for a step through
// exitName right now (movement parity 4b). A variable so tests can pin it.
var stepAffordable = func(mob *mobs.Mob, exitName string) bool {
	return actions.QuoteMobStep(mob, exitName).OK()
}
```

and add `"github.com/GoMudEngine/GoMud/internal/actions"` to the imports.

At the top of `case round >= p.Issued+stepTimeoutRounds:`, before the `tryUnlock` check, insert:

```go
			// Movement parity 4b: a step that did not move because she
			// could not pay for it is a rest, not a bad exit. Re-quote next
			// round; never charge exhaustion to the map.
			if !stepAffordable(mob, p.Steps[p.Next].Exit) {
				p.Expect = 0
				return
			}
```

Replace the final issue (`mob.Command(`go ` + util.EscapeAnsiTags(st.Exit))` and the four lines after it) with:

```go
	// Movement parity 4b: quote before issuing. Tired, she waits without
	// starting the step clock and tells her mind once.
	if !stepAffordable(mob, st.Exit) {
		if !p.Resting {
			p.Resting = true
			c.mind.addLine(Line{Kind: `event`, Text: tooTiredLine}, m.cfg.WorkingMemoryLines)
			c.dirty = true
		}
		return
	}
	p.Resting = false
	mob.Command(`go ` + util.EscapeAnsiTags(st.Exit))
	p.FromRoom = cur
	p.Issued = round
	p.Expect = st.To
	if st.To == 0 {
		p.Expect = -1
	}
```

- [ ] **Step 3: Run**

Run: `go test ./modules/aicompanion/ -count=1`
Expected: PASS (including `TestNothingNamingAnyoneIsWrittenBeforeConsent`: the resting line names no one).

- [ ] **Step 4: Commit**

```bash
gofmt -l modules/
git add modules/aicompanion/travel.go modules/aicompanion/travel_tired_test.go
git commit -m "feat(aicompanion): a tired companion rests on a trip without marking the exit bad

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 10: Admin readout, re-fork guard, shipped-route measurement

**Files:**
- Modify: `internal/usercommands/admin.mob.go:224-245`
- Create: `internal/usercommands/admin_mob_vitals_test.go`
- Create: `move_wrapper_guard_test.go` (repo root)
- Create: `move_route_stamina_test.go` (repo root)

- [ ] **Step 1: Failing vitals test**

`internal/usercommands/admin_mob_vitals_test.go`:

```go
package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/require"
)

// Movement parity 4b: a playtester watches a mob tire through `mob schedule`.
func TestMobVitalsLine(t *testing.T) {
	m := &mobs.Mob{InstanceId: 5}
	m.Character.Stamina = 37
	m.Character.StaminaMax.Value = 120
	m.Character.ActionPointsMax.Value = 200
	line := mobVitalsLine(m, 10)
	require.True(t, strings.Contains(line, "stamina:         37 / 120"), line)
	require.True(t, strings.Contains(line, "action points:   200 / 200"), line)
	require.True(t, m.Character.ActionPointsSettled, "the readout shows settled points")
}
```

Run: `go test ./internal/usercommands/ -run TestMobVitalsLine -count=1` - Expected: FAIL, `undefined: mobVitalsLine`.

- [ ] **Step 2: Implement the readout**

In `internal/usercommands/admin.mob.go`, add:

```go
// mobVitalsLine is the stamina and action point readout for `mob schedule`
// (movement parity 4b). The mob's points are settled first, so the number is
// what its next step would see.
func mobVitalsLine(m *mobs.Mob, turn uint64) string {
	m.Character.SettleActionPoints(turn)
	return fmt.Sprintf(
		"  stamina:         %d / %d\n"+
			"  action points:   %d / %d",
		m.Character.Stamina, m.Character.EffectivePoolMax(characters.PoolStamina),
		m.Character.ActionPoints, m.Character.ActionPointsMax.Value)
}
```

In `mob_Schedule`, directly after the `if m == nil { ... }` block (before `if m.ScheduleId == ""`), insert:

```go
	user.SendText(messaging.CategorySystem, fmt.Sprintf("Vitals for %s (instance %d):\n%s",
		m.Character.Name, instId, mobVitalsLine(m, util.GetTurnCount())))
```

so a patrol-only mob shows its vitals too (fact V36). Add the `characters` and `util` imports if the compiler asks.

Run the test again: PASS.

- [ ] **Step 3: Re-fork guard**

`move_wrapper_guard_test.go` (repo root):

```go
package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestMoveWrappersDoNotReFork: the step price and the arrival detection live
// in internal/actions/move.go (movement parity 4b). If a command wrapper
// prices, charges or rolls detection itself, the player and mob paths have
// forked again (mobs walked free and never rolled detection until 4b).
func TestMoveWrappersDoNotReFork(t *testing.T) {
	forbidden := regexp.MustCompile(`DeductActionPoints|GetMovementStaminaCost|ApplyCostFloatOrRefuse|CalcSneakScoreVsObserver|CalcDetectionScore|RunContest`)
	for _, path := range []string{"internal/usercommands/go.go", "internal/mobcommands/go.go"} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if loc := forbidden.FindIndex(b); loc != nil {
			t.Errorf("%s prices or detects a step itself (%q); call actions.ChargeMove / actions.EntryDetection instead", path, b[loc[0]:loc[1]])
		}
	}
}

// TestOnlyActionsSpendActionPoints: action points are spent by
// actions.ChargeMove alone. Any other production caller is a second price.
func TestOnlyActionsSpendActionPoints(t *testing.T) {
	scanned := 0
	for _, root := range []string{"internal", "modules"} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			scanned++
			slash := filepath.ToSlash(path)
			if strings.HasPrefix(slash, "internal/actions/") || strings.HasPrefix(slash, "internal/characters/") {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(b), "DeductActionPoints(") {
				t.Errorf("%s spends action points itself; only actions.ChargeMove may", slash)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	if scanned < 500 {
		t.Fatalf("scanned only %d files; the walk is not seeing the tree, so a pass proves nothing", scanned)
	}
}
```

Run: `go test -run "TestMoveWrappersDoNotReFork|TestOnlyActionsSpendActionPoints" -count=1 .`
Expected: PASS.

- [ ] **Step 4: Prove the guard can fail**

Add one line inside `usercommands.Go` right after `charge := actions.ChargeMove(...)`: `_ = user.Character.DeductActionPoints(0)`. Run the command from Step 3.
Expected: FAIL naming `internal/usercommands/go.go` in BOTH tests. Remove the line with the Edit tool, run again, expect PASS, and confirm `git diff --stat internal/usercommands/go.go` shows no change. Record the failing output in the commit message body.

- [ ] **Step 5: Shipped-route measurement**

`move_route_stamina_test.go` (repo root):

```go
package main

import (
	"math"
	"os"
	"sort"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// routeStops lists a routed mob's stops in walking order: its schedule's
// target rooms by start hour (a patrol segment contributes its waypoints),
// then its standalone patrol, each loop closed back to its start.
func routeStops(m *mobs.Mob) []int {
	var stops []int
	addPatrol := func(id string) {
		p := mobs.GetPatrol(id)
		if p == nil {
			return
		}
		for _, w := range p.Waypoints {
			stops = append(stops, w.Room)
		}
		if len(p.Waypoints) > 1 && p.LoopShape != "oneshot" {
			stops = append(stops, p.Waypoints[0].Room)
		}
	}
	if m.ScheduleId != "" {
		if s := mobs.GetSchedule(m.ScheduleId); s != nil {
			segs := append([]mobs.ScheduleSegment(nil), s.Segments...)
			sort.Slice(segs, func(i, j int) bool { return segs[i].Start < segs[j].Start })
			first := len(stops)
			for _, seg := range segs {
				if seg.Activity == "patrol" {
					addPatrol(seg.PatrolId)
					continue
				}
				if seg.TargetRoom > 0 {
					stops = append(stops, seg.TargetRoom)
				}
			}
			if len(stops)-first > 1 {
				stops = append(stops, stops[first])
			}
		}
	}
	if m.PatrolId != "" {
		addPatrol(m.PatrolId)
	}
	return stops
}

// TestShippedRoutedMobsCanPayTheirWorstStep measures, instead of inferring,
// whether movement parity 4b can strand a scheduled or patrolling mob: for
// every shipped mob with a route, the dearest single step on the paths between
// its stops, priced by actions.MovePrice for the mob as spawned (its real
// carried load), against its reachable stamina pool. A mob whose plain worst
// step exceeds its whole pool can never walk its route and FAILS the test.
// Hidden steps (x3, capped) that exceed the pool are LOGGED: they bite only a
// mob that sneaks on that route.
//
// Loads the whole world, so it shares the boot smoke test's opt-in.
func TestShippedRoutedMobsCanPayTheirWorstStep(t *testing.T) {
	if os.Getenv(bootSmokeEnvVar) == `` {
		t.Skipf("set %s=1 to run the shipped-route stamina measurement (~40s)", bootSmokeEnvVar)
	}
	mudlog.SetupLogger(nil, `LOW`, ``, false)
	configs.SetConfigForTest(t, configs.GetConfig())
	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}
	loadAllDataFiles(false)
	mapper.PreCacheMaps()

	b := configs.GetBalanceConfig()
	hiddenMult := float64(b.HiddenMoveStaminaMultiplier)
	capCost := float64(b.MovementMaxStaminaCost)

	type row struct {
		name              string
		mobId             int
		pool              int
		worst, hidden     float64
		worstRoom, rooms  int
	}
	var rows []row
	for _, spec := range mobs.AllMobTemplates() {
		stops := routeStops(spec)
		if len(stops) < 2 {
			continue
		}
		route := map[int]bool{stops[0]: true}
		for i := 1; i < len(stops); i++ {
			steps, err := mapper.GetPath(stops[i-1], stops[i])
			if err != nil {
				t.Logf("mob %d %s: no path %d -> %d (%v); the boot validator owns that", spec.MobId, spec.Character.Name, stops[i-1], stops[i], err)
				continue
			}
			for _, s := range steps {
				route[s.RoomId()] = true
			}
		}

		// Stat pools are randomised per spawn: take the weakest of five.
		pool := math.MaxInt
		var worst float64
		worstRoom := 0
		for i := 0; i < 5; i++ {
			inst := mobs.NewMobByIdFresh(spec.MobId, stops[0])
			if inst == nil {
				t.Fatalf("mob %d did not spawn", spec.MobId)
			}
			if p := inst.Character.EffectivePoolMax(characters.PoolStamina); p < pool {
				pool = p
			}
			for rid := range route {
				r := rooms.LoadRoom(rid)
				if r == nil {
					continue
				}
				if _, st := actions.MovePrice(&inst.Character, r); st > worst {
					worst, worstRoom = st, rid
				}
			}
			mobs.DestroyInstance(inst.InstanceId)
		}
		rows = append(rows, row{spec.Character.Name, int(spec.MobId), pool, worst,
			math.Min(worst*hiddenMult, capCost), worstRoom, len(route)})
	}

	if len(rows) < 10 {
		t.Fatalf("measured only %d routed mobs; the world did not load, so a pass proves nothing", len(rows))
	}
	sort.Slice(rows, func(i, j int) bool {
		return float64(rows[i].pool)-rows[i].hidden < float64(rows[j].pool)-rows[j].hidden
	})
	for i, r := range rows {
		if i < 15 {
			t.Logf("%-28s mob %5d pool %4d worst %.2f (room %d) hidden %.2f over %d rooms",
				r.name, r.mobId, r.pool, r.worst, r.worstRoom, r.hidden, r.rooms)
		}
		if r.worst > float64(r.pool) {
			t.Errorf("%s (mob %d) can never pay its worst step: %.2f stamina (room %d) against a %d pool",
				r.name, r.mobId, r.worst, r.worstRoom, r.pool)
		} else if r.hidden > float64(r.pool) {
			t.Logf("HIDDEN ONLY: %s (mob %d) cannot pay a hidden step of %.2f against %d", r.name, r.mobId, r.hidden, r.pool)
		}
	}
	t.Logf("measured %d routed mobs", len(rows))
}
```

Confirm `AllMobTemplates` returns `[]*Mob` (`grep -n "func AllMobTemplates" internal/mobs/mobs.go`, fact: `mobs.go:234`) and that `mapper.GetPath`'s steps expose `RoomId()` (they implement `mobs.PathRoom`).

Run (PowerShell, so no console window spawns): `$env:DOGMUD_BOOT_SMOKE='1'; go test -run TestShippedRoutedMobsCanPayTheirWorstStep -count=1 -v . ; Remove-Item Env:DOGMUD_BOOT_SMOKE`
Expected: PASS with a logged table. **If it FAILS, stop and report the named mobs to the owner before merge**: a failing row is a shipped mob 4b would strand, and the fix (a knob, a stat, a route) is the owner's call, not this plan's.

- [ ] **Step 6: Commit**

```bash
gofmt -l internal/ modules/
gofmt -l move_wrapper_guard_test.go move_route_stamina_test.go
git add internal/usercommands/admin.mob.go internal/usercommands/admin_mob_vitals_test.go move_wrapper_guard_test.go move_route_stamina_test.go
git commit -m "test: move re-fork guard, shipped-route stamina measurement, mob vitals readout

Guard proven able to fail: <paste the two failing lines from Task 10 Step 4>

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 11: Docs

**Files:** the eight `context.md` files, `docs/PATCH_NOTES.md`, `docs/README.md`

- [ ] **Step 1: `internal/actions/context.md`**: add a "Movement (`move.go`)" section: `MovePrice`, `QuoteMove`, `ChargeMove`, `MoveCharge` (`OK`, `Never`), `MoveRefusal`, `QuoteMobStep`, `TrainSearchOnMove`, `EntryDetection`/`EntryDetectionResult`; the rule that wrappers charge after their lock gates; mobs are settled, players never; the full rationale paragraph moved from `usercommands.movementTrainsSearch` (copied verbatim from `go.go:35-72` at Task 3).
- [ ] **Step 2: `internal/characters/context.md`**: replace the "ActionPoints is a fourth pool" gotcha (`:654-656`) with: mobs settle lazily through `SettleActionPoints` (never call it for players), `CanAffordCostFloat` is the read-only twin of `ApplyCostFloatOrRefuse`, and only `actions.ChargeMove` spends action points (guarded at the repo root).
- [ ] **Step 3: `internal/mobs/context.md`**: `PathQueue.Peek`; spawn settles AP full; `MovePackFollowers`'s `canStep`.
- [ ] **Step 4: `internal/hooks/context.md`**: `advanceMobPath` waits on an unaffordable step; `mobPathStepWaiting` and the patrol counter.
- [ ] **Step 5: `internal/mobcommands/context.md`**, **`internal/usercommands/context.md`**, **`internal/behaviortree/context.md`**, **`modules/aicompanion/context.md`**: one paragraph each on the wrapper or caller change in this plan.
- [ ] **Step 6: Verify every named symbol exists**: `python tools/context_md_audit.py` must report no new phantom symbols for these eight files.
- [ ] **Step 7: `docs/PATCH_NOTES.md`**: dated entry, player-facing, no numbers: creatures now tire as they travel and rest when spent; a guard walking in can find you hiding, and a thief creeping in can be spotted; walking into a locked door no longer tires you.
- [ ] **Step 8: `docs/README.md`**: rows for this plan (if not already added when it was committed) and for the new repo-root files `move_wrapper_guard_test.go` and `move_route_stamina_test.go`.
- [ ] **Step 9: Commit**

```bash
git add internal/actions/context.md internal/characters/context.md internal/mobs/context.md internal/hooks/context.md internal/mobcommands/context.md internal/usercommands/context.md internal/behaviortree/context.md modules/aicompanion/context.md docs/PATCH_NOTES.md docs/README.md
git commit -m "docs: movement parity 4b context, patch notes and index

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 12: Gate, boot check, playtest, PR

- [ ] **Step 1: Local gate** (CI minutes are exhausted this month; this IS the gate)

```bash
gofmt -l internal/ modules/          # expect no output
go vet ./...
go build ./...
go test ./... -count=1
golangci-lint run --new-from-merge-base=origin/master   # expect 0 issues
```

Run the `grep -c`-style "expect zero" checks standalone, never in an `&&` chain.

- [ ] **Step 2: Boot check in the detached worktree** per `dogmud-shipping` ("Boot check in an isolated worktree"): build to `boot-check.exe`, expect 0 panics and exactly 1 `Server Ready`, then `git worktree remove --force`.

- [ ] **Step 3: Playtest** (load `dogmud-playtesting`; local runs need an ephemeral goals file and `--checkout`; never kill the owner's server, kill by your own PID only). Scenarios, each checking that nothing freezes, loops or falls back home unexpectedly:
  1. Shadow a patrolling guard for a full loop with `mob schedule <inst>` open; watch stamina and action points move.
  2. Shadow a scheduled shopkeeper across a segment change.
  3. Overload an AI companion, walk it across rough terrain, send it on an errand; confirm it rests with the mind line and resumes, and its map shows no new failed exits.
  4. Hide in a room a guard patrols through; confirm the guard can find you.
  5. Let a thief mob sneak into your room; confirm it can be spotted.
  6. Walk into a locked door you cannot open; confirm `status` shows no action point or stamina change.

  Reports are gitignored: file findings as a memory before closing the session.

- [ ] **Step 4: PR** (named paths were committed task by task; nothing to add here)

```bash
git push -u origin feature/movement-parity-4b
gh pr create --repo pruuk/DOGMud --base master --head feature/movement-parity-4b --title "feat: movement parity (slice 4b): mobs pay for steps and roll hidden detection" --body "$(cat <<'EOF'
Player/mob parity slice 4b. Spec: docs/superpowers/specs/2026-09-28-flee-and-movement-parity-design.md. Plan: docs/superpowers/plans/2026-09-29-movement-parity-4b.md.

- One step price and charge (actions.ChargeMove/QuoteMove) and one arrival detection (actions.EntryDetection) for players and mobs.
- Mob action points settle lazily and a spawn starts full (every live mob held 0).
- The charge runs after the lock gate: a locked door costs nothing, an exit message no longer charges twice.
- Tired mobs wait: the path walker keeps its path, a resting patrol does not count failures, wanderers and pack followers do not count steps they never take, single-step behaviours fail, and the AI companion rests without marking exits bad.
- Guards: move re-fork guard (proven able to fail), G1 to G4 re-keyed, shipped-route stamina measurement.

CI minutes are exhausted for the month: the local gate (gofmt, vet, build, go test ./..., golangci-lint new-from-merge-base) and the boot check are the gate.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

Read back the URL `gh` prints and confirm it says `pruuk/DOGMud`. Merge with `gh pr merge <n> --repo pruuk/DOGMud --merge --delete-branch` once the owner approves; never deploy.

---

## Self-review

**Spec coverage.** ChargeMove/QuoteMove/EntryDetection in `move.go`: Tasks 3, 4. Rare Search roll beside them: Task 3, wired Tasks 5, 6. Charge after locks and requeue (M6): Task 5 with its test. Lazy mob AP, spawn full: Tasks 1, 2. Both wrappers: Tasks 5, 6. Numeric teleport untouched: Task 6 Step 2 says so. Caller 1 (Peek, wait, clear never-affordable): Tasks 2, 7. Caller 2 (patrol counter): Task 7. Caller 3 (Wander, pack): Task 8. Caller 4 (companion, mind line, no Fails): Task 9. Caller 5 (BT single steps, `keep_distance` left to 4a): Task 8. Move re-fork guard proven able to fail: Task 10. G1: Task 5. G2: Task 4 (moved). G3: Tasks 4 (twin) and 5. G4: Task 5 Step 7. Shipped-route measurement: Task 10. Admin readout: Task 10. context.md: Task 11. Gate, boot, playtest, PR: Task 12. Parity table rows: AP 10/50 (Task 3 table), regen (Task 1), terrain/encumbrance/Search discount (MovePrice through `GetMovementStaminaCost`, Task 3), flight/hidden/cap (Task 3, flight by code path), refund (Task 3), charged after locks (Task 5), four detection pairings both directions with Search awards (Task 4), rare Search per move (Task 3), walking in combat unchanged (no task touches it).

**Placeholder scan.** Two steps depend on source not yet written (post-4a `mobcommands.Go`) or on a guard's output (G4 keys); each gives the exact insertion points or the exact row template. The commit message in Task 10 Step 6 asks for the pasted failing output from Step 4.

**Type consistency.** `MoveCharge{Refusal, ActionCost, StaminaCost, Winded, Never}` and `OK()` are used identically in Tasks 3 to 10; `QuoteMobStep(*mobs.Mob, string) MoveCharge`; `EntryDetection(Actor, *rooms.Room, bool) EntryDetectionResult` with `StillSneaking`; `SettleActionPoints(uint64)`; `CanAffordCostFloat(Pool, float64) bool`; `MovePackFollowers(*Mob, string, []int, func(*Mob) bool)`; `advanceMobPath(*mobs.Mob) bool`; `mobPathStepWaiting(*mobs.Mob) bool`; `mobVitalsLine(*mobs.Mob, uint64) string`; `stepAffordable func(*mobs.Mob, string) bool`.
