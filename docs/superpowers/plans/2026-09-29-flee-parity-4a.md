# Flee Parity (Slice 4a) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A mob flees by the player's rules: same gates, same cost, same one-round `Disengaging` delay, same blocker contest, through one shared body in `internal/actions`.

**Architecture:** The flee command half becomes `actions.BeginFlee` and the round half becomes `actions.ResolveFlee`; both take an `Actor`. The command-to-round handoff moves from the player's temp data onto `Character`. Thin wrappers stay per actor for narration: `usercommands.Flee`, `mobcommands.Flee`, `hooks.handlePlayerFlee` and a new `hooks.handleMobFlee` in the mob round pass. The mob's relocation tail is lifted into `actions.RelocateMob` so a fleeing mob moves without going through the (4b: paid) `go` command.

**Tech Stack:** Go 1.25, the `combatphase` state machine, `internal/actions` Actor seam, repo-root guard tests.

**Spec:** `docs/superpowers/specs/2026-09-28-flee-and-movement-parity-design.md` (4a section and the owner rulings at its foot are binding).

**Branch:** `feature/flee-parity-4a`, cut from `docs/flee-movement-spec` (which carries the spec and both plans and is already merged with master `961995259`). Work in a worktree:

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
git worktree add -b feature/flee-parity-4a C:/tmp/dogmud-flee-4a docs/flee-movement-spec
```

All paths below are relative to that worktree. 4b (`2026-09-29-movement-parity-4b.md`) starts from master AFTER this PR merges.

---

## Facts verified against source (2026-09-29, HEAD `c061ca58f`)

Every row re-read today. Config numbers are Go defaults (tests load those, not `config.yaml`).

| # | Fact | Where |
|---|---|---|
| V1 | `usercommands.Flee` clears a stale admission when not disengaging, then refuses: `NoMovement` (line 72), `NoFlee` (79), `IsDisengaging` (88), `!IsInCombat` (98); publishes a pending `fleeAdmission{}` (106); nil phase refuses (107); `TransitionToDisengaging{Trigger: TriggerFleeCommand, Actor: {UserId}}` (112); on error cancels and picks: not in combat, grappled, not standing, "can't break away just yet" (116-133); quotes `ActionCostRequest{ActionFlee, PoolStamina, FleeStaminaCost, FlightFleeStaminaMult if flying, Units 1}` and `CommitCost(quote, CostPartial)` (138-150); ready admission `includeSkill: !Short()` (151); shortage line (155); "You attempt to flee..." (159) | `internal/usercommands/flee.go:61-162` |
| V2 | `TakeFleeAdmission(user)` peeks, returns false on a pending (not ready) handoff and leaves it, else takes it; `CancelFleeAdmission(user)` takes either. Temp key `flee-include-skill` | `internal/usercommands/flee.go:17-59` |
| V3 | Admission callers outside `flee.go`: `CombatPhase_FleeCancellation.go:25`, `NewRound_DoCombat_helpers.go:853,859`, `hooks/flee_cost_test.go:133,158,181,265`, `usercommands/flee_cost_test.go:41,45,78,118,156,193,201,209-221,251,261` | grep |
| V4 | The world loop runs commands and rounds under `util.LockMud()` on one goroutine, so a Character field needs no mutex; the pending/ready split is kept for reentrancy, as today | `world.go:785-880` |
| V5 | `Character` has no mutex; runtime fields are unexported with no yaml tag (`costCarry` comment, "an unexported field is already invisible"); mob templates are shallow-copied, so a VALUE field is safe where a pointer or map would be shared | `internal/characters/character.go:155-178` |
| V6 | `TransitionToDisengaging` vetoes `TriggerFleeCommand` when `positionSelf()` is false; the check is `c.IsStanding()`, registered by `hooks.wireCombatPhaseVetoes` through `OnCharacterCreated`. So in any test package that does not link `hooks` there is NO position veto | `combatphase.go:404-419`; `hooks/CombatPhase_Vetoes.go:33-38,102-104` |
| V7 | `position.Machine.IsStanding()` is `State() == Standing`, so a clinch is not standing; grapple and prone both already fail the veto | `internal/state/position/position.go:132` |
| V8 | `ResolveFlee(success)`: no-op unless `Disengaging`; success `ForceIdle(TriggerFleeSuccess)`; failure back to `Engaged` on `LastTarget` | `combatphase.go:464-484` |
| V9 | `IsInCombat()` is true for Engaging, Engaged and Disengaging | `combatphase.go:230-234`; `character.go:761` |
| V10 | `handlePlayerFlee`: orphan take when not disengaging (849-854), take (859-862), grapple refusal + `ResolveFlee(false)` (871-877), `ResolveFleeBlockers(char, room, includeSkill)` (885), award `contested && includeSkill` (897-900), blocker lines + `ResolveFlee(false)` (901-922), no exit (925-934), success lines (936-939), `targeting.Release` + `ResolveFlee(true)` (943-946), `MoveToRoom`, charmed mobs follow, `Look`, `room_enter` (948-973) | `internal/hooks/NewRound_DoCombat_helpers.go:838-976` |
| V11 | Round loop: `if handlePlayerFlee(user, uRoom, userId) { continue }` | `internal/hooks/NewRound_DoCombat.go:173` |
| V12 | Mob pass in-combat block: `CancelCombatConditions` (326), `handleMobFoldCasting` (328), `ValidateAggro`/`RetargetOrEnd` (333-337), block closes at 338; `mob_combat_round` fires at 392-398 | `internal/hooks/NewRound_DoCombat.go:323-398` |
| V13 | `mobcommands.Flee`: non-combatant (23), `NoFlee` (29), grapple room line on `CategoryGrappleFlow` (40-44), `ResolveFleeBlockers(&mob.Character, room, true)` (50), award on `contested` (58-61), blocked line `SendTextVisualHidingNames(CategoryRoomExit, ...)` (69-72), `targeting.Release` before the exit pick (76-78), cornered line `CategoryMobEmote` (86-88), "X flees!" `CategoryRoomExit` (92-93), `Go(exitName, ...)` (96), `mob_flee` (98-101) | `internal/mobcommands/flee.go` |
| V14 | `combat.FleeBlocker{Name, UserId, MobInstanceId}`, `IsPlayer()`; `ResolveFleeBlockers(fleer *characters.Character, room *rooms.Room, includeSkill bool) (*FleeBlocker, bool)` | `internal/combat/flee.go:14-59` |
| V15 | `room.GetRandomExit() (string, int)` skips secret and locked exits, including mutator exits; `room.GetExitInfo(name) (exit.RoomExit, bool)` | `internal/rooms/rooms.go:1306,1323` |
| V16 | `mobcommands.Go` relocation tail: `RemoveMob`, `clearRoomAggroOnDeparture`, `AddMob` (205-207), exit line + audio (212-216), entry line + audio (219-223), `SendTextToExits` (225), sounds (227-228), NPC party pull (244-266), waypoint `noop` (269-274). Far-side lock check at 185-200. `sendMovementMessage` is `room.SendTextVisualWithAudio` | `internal/mobcommands/go.go:102-280` |
| V17 | `clearRoomAggroOnDeparture(room, departingInstanceId)` is unexported in `mobcommands/go.go:22-86`; called at `go.go:137,206` and by `go_retarget_notice_test.go:81,154`; `actions/retarget_notice.go:28` names it in a comment | grep |
| V18 | `actions` imports `combat`, `targeting`, `parties`, `mobs`, `users`, `state`; `behaviortree` depends on `actions` (so `actions` cannot fire `mob_flee` or `room_enter`; those stay in `hooks`) | `go list` |
| V19 | `Actor` has `GetCharacter`, `GetUserId`, `GetMobInstanceId`, `AwardResolved(won, cands...)` (user passes its UserId, mob 0); `NewUserActorInRoom`, `NewMobActorInRoom`; `MobActor{Mob, Room}` | `internal/actions/actor.go:14-69`; `actor_mob.go:12-31`; `actor_user.go:12-32` |
| V20 | `actKeepDistance` issues `go <dir>` from `pickRetreatExit` (unlocked, home first) after refreshing `CombatMemory`; `actFlee` issues `flee` | `internal/behaviortree/actions_archer.go:237-281,287`; `actions_combat.go:89-96` |
| V21 | Authored `do: flee` fires on `mob_hurt` in 15 archetypes, on `mob_combat_round` (Sullen Garrow), under `mob_in_combat` (Chrysalis Phantom), and **on `mob_idle` after `try_steal` in `archetypes/thief.yaml:78-90`**: an out-of-combat escape that a refusing out-of-combat flee would strand | `_datafiles/world/dogmud/behaviors/` |
| V22 | `PackFlee` issues `flee` to every same-MobId or same-species, uncharmed, combatant, non-immune mob in the room, then prints the scatter line when `fleeCount > 0` | `internal/hooks/MobDeath_PackFlee.go:56-100` |
| V23 | Tests pinned to today's shapes: `actions_archer_test.go:272` expects `"go north"` from keep_distance; `archer_kiting_test.go` seeds a post-retreat archer (unaffected) | grep |
| V24 | Guards: `contest_site_guard_test.go:60` keys `internal/combat/flee.go:ResolveFleeBlockers` (unchanged by 4a); no guard keys a flee or mob-go literal by file (grep for `flees!`, `tries to flee but`, `leaves towards`, `mobcommands/go.go` in `*_test.go` is empty; the same grep finds `blocks you from fleeing` in `hooks/flee_cost_test.go`, so it can match) | grep |
| V25 | `drink_wrapper_guard_test.go` is the re-fork guard template (package `main`, `os.ReadFile`, one regexp) | repo root |
| V26 | `FleeStaminaCost` Go default 10 (pinned by `configs/config.balance.flee_test.go`), `FlightFleeStaminaMult` default 0.5 | `internal/configs/config.balance.combat.go:202-207` |
| V27 | `hooks` fixture `seedAllRegistries()`: users 1 and 2 and mob instance 100 ("Skeleton", MobId 1, HomeRoomId 1) in room 1; room 1 `north` -> room 2, room 2 `south` -> room 1; zone active | `internal/hooks/hooks_test.go:57-175` |

## Design decisions made while planning

1. **`BeginFlee` pre-checks grapple and standing before the transition** (`FleeGate`). V6: outside `hooks` the position veto is not registered, so without the pre-check the parity table could not prove the prone row, and the result would depend on which package is linked. V7 shows the pre-check refuses exactly what the veto refuses, in the same order the player's error branch reports, so no player line changes. The transition-error mapping stays as a fallback.
2. **`actFlee` walks when the mob is not in combat.** V21: the thief steals out of combat and flees. A flee out of combat now refuses (ruling 1, parity table row "Out of combat refuses"), so `actFlee` issues `go <pickRetreatExit>` instead when the mob is not fighting, which is what a player would do. `PackFlee` issues `flee` directly, not through `actFlee`, and gets its own in-combat filter (owner ruling 1 on open question 1).
3. **The NPC party pull moves into `RelocateMob`.** A fleeing party leader dragged its idle members along today (flee went through `Go`); keeping that avoids a silent behaviour change. The waypoint `noop` is path-walking only and stays in `Go`.
4. **A fleeing mob does not check the far-side lock.** A fleeing player does not either (`rooms.MoveToRoom`). Walking keeps the check in `Go`.
5. **A mob grappled between command and round** gets today's command-time room line ("tries to break free but you've got them locked down!") at resolution too, mirroring the player's round-time grapple line.
6. **Known, accepted:** a mob mid-fold-cast is skipped by `handleMobFoldCasting` before `handleMobFlee`, so its flee resolves when the cast ends, the same shape as the player loop (V12, V10 run after fold casting). A disengaging mob in a zone that goes idle waits for the zone to wake.

## File map

| File | Change |
|---|---|
| `internal/characters/flee_admission.go` | Create: `FleeAdmission`, the runtime field's type, `PublishFleeAdmission`, `TakeFleeAdmission`, `CancelFleeAdmission` |
| `internal/characters/flee_admission_test.go` | Create |
| `internal/characters/character.go` | Add the `fleeHandoff` field |
| `internal/actions/flee.go` | Create: `FleeRefusal`, `FleeGate`, `FleeBegin`, `BeginFlee`, `FleeOutcome`, `ResolveFlee`, `fleeExit` |
| `internal/actions/flee_parity_test.go` | Create: the 4a parity table |
| `internal/actions/relocate_mob.go` | Create: `RelocateMob`, `ClearRoomAggroOnDeparture` (moved) |
| `internal/actions/relocate_mob_test.go` | Create |
| `internal/actions/retarget_notice.go` | Comment names the moved function |
| `internal/usercommands/flee.go` | Thin wrapper; forwarders deleted |
| `internal/usercommands/flee_cost_test.go` | Admission calls move to the Character |
| `internal/mobcommands/flee.go` | Thin wrapper |
| `internal/mobcommands/go.go` | Calls `actions.RelocateMob`; `clearRoomAggroOnDeparture` removed |
| `internal/mobcommands/go_retarget_notice_test.go` | Calls `actions.ClearRoomAggroOnDeparture` |
| `internal/hooks/NewRound_DoCombat_helpers.go` | `handlePlayerFlee` on `ResolveFlee`; new `handleMobFlee` |
| `internal/hooks/NewRound_DoCombat.go` | Calls `handleMobFlee` |
| `internal/hooks/CombatPhase_FleeCancellation.go` | Cancels for any character |
| `internal/hooks/MobDeath_PackFlee.go` | Only fighting packmates flee |
| `internal/hooks/mob_flee_round_test.go` | Create |
| `internal/hooks/flee_cost_test.go` | Admission calls move to the Character |
| `internal/behaviortree/actions_combat.go` | `actFlee` walks out of combat |
| `internal/behaviortree/actions_archer.go` | `actKeepDistance` issues a flee |
| `internal/behaviortree/actions_archer_test.go` | Expects `flee north` |
| `internal/behaviortree/actions_flee_test.go` | Create |
| `flee_wrapper_guard_test.go` | Create (repo root) |
| `context.md` in `internal/actions`, `internal/characters`, `internal/mobcommands`, `internal/usercommands`, `internal/hooks`, `internal/behaviortree` | Update |
| `docs/PATCH_NOTES.md` | Dated entry |

---

### Task 1: The flee admission lives on the Character

**Model:** sonnet.

**Files:**
- Create: `internal/characters/flee_admission.go`, `internal/characters/flee_admission_test.go`
- Modify: `internal/characters/character.go` (field beside `costCarry`, around line 178)
- Modify: `internal/usercommands/flee.go`, `internal/usercommands/flee_cost_test.go`, `internal/hooks/NewRound_DoCombat_helpers.go`, `internal/hooks/CombatPhase_FleeCancellation.go`, `internal/hooks/flee_cost_test.go`

- [ ] **Step 1: Write the failing test**

`internal/characters/flee_admission_test.go`:

```go
package characters

import "testing"

// The flee command publishes a PENDING admission before its state transition
// and a READY one after it has settled the cost. The round resolver may only
// take a ready one; terminal cancellation may retract either. This is the
// contract usercommands kept in the player's temp data until slice 4a moved
// it here so a mob could flee through the same handoff.
func TestFleeAdmission_PendingWaitsForReadyOrCancel(t *testing.T) {
	c := New()
	c.PublishFleeAdmission(FleeAdmission{})
	if _, ok := c.TakeFleeAdmission(); ok {
		t.Fatal("round resolver took a pending admission")
	}
	if !c.CancelFleeAdmission() {
		t.Fatal("cancel could not retract a pending admission")
	}
	if c.CancelFleeAdmission() {
		t.Fatal("an admission was cancelled twice")
	}
}

func TestFleeAdmission_ReadyIsTakenOnce(t *testing.T) {
	c := New()
	c.PublishFleeAdmission(FleeAdmission{IncludeSkill: true, Ready: true, PreferredExit: "east"})
	got, ok := c.TakeFleeAdmission()
	if !ok || !got.IncludeSkill || got.PreferredExit != "east" {
		t.Fatalf("take = %+v, %v; want the published ready admission", got, ok)
	}
	if _, ok := c.TakeFleeAdmission(); ok {
		t.Fatal("a ready admission was taken twice")
	}
}

func TestFleeAdmission_NilCharacterIsSafe(t *testing.T) {
	var c *Character
	if _, ok := c.TakeFleeAdmission(); ok {
		t.Fatal("nil character yielded an admission")
	}
	if c.CancelFleeAdmission() {
		t.Fatal("nil character cancelled an admission")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/characters/ -run TestFleeAdmission -count=1`
Expected: FAIL to compile, `c.PublishFleeAdmission undefined`.

- [ ] **Step 3: Implement**

`internal/characters/flee_admission.go`:

```go
package characters

// FleeAdmission is the flee command's handoff to the round that resolves the
// flee. The command publishes a pending one (Ready false) before it asks
// CombatPhase for Disengaging, and a ready one once the cost is settled.
// IncludeSkill is false when the fleer could not pay in full and so brings no
// Skullduggery to the blocker contest. PreferredExit, when set and still
// passable at resolution, is the exit the flee takes instead of a random one
// (a kiting archer falls back toward home).
type FleeAdmission struct {
	IncludeSkill  bool
	Ready         bool
	PreferredExit string
}

// fleeHandoff holds at most one admission. It is a value, not a pointer, so a
// mob template's shallow copy cannot share one between instances.
type fleeHandoff struct {
	set       bool
	admission FleeAdmission
}

// PublishFleeAdmission replaces any admission with a.
func (c *Character) PublishFleeAdmission(a FleeAdmission) {
	if c == nil {
		return
	}
	c.fleeHandoff = fleeHandoff{set: true, admission: a}
}

// TakeFleeAdmission consumes a READY admission. A pending one is left in
// place for the command to finish, and the second result is false, so a round
// that observes Disengaging inside the command's handoff window cannot
// consume an attempt whose cost is not decided yet.
func (c *Character) TakeFleeAdmission() (FleeAdmission, bool) {
	if c == nil || !c.fleeHandoff.set || !c.fleeHandoff.admission.Ready {
		return FleeAdmission{}, false
	}
	a := c.fleeHandoff.admission
	c.fleeHandoff = fleeHandoff{}
	return a, true
}

// CancelFleeAdmission retracts a pending or ready admission and reports
// whether there was one. CombatPhase terminal hooks use it when combat ends
// before the flee round can resolve.
func (c *Character) CancelFleeAdmission() bool {
	if c == nil || !c.fleeHandoff.set {
		return false
	}
	c.fleeHandoff = fleeHandoff{}
	return true
}
```

In `internal/characters/character.go`, directly after the `costCarry map[Pool]float64` line:

```go
	// fleeHandoff is the flee command's admission for the round resolver
	// (flee_admission.go). Runtime only; see costCarry for why there is no
	// yaml tag.
	fleeHandoff fleeHandoff
```

- [ ] **Step 4: Run it to verify it passes**

Run: `go test ./internal/characters/ -run TestFleeAdmission -count=1`
Expected: PASS.

- [ ] **Step 5: Move every caller onto the Character and delete the forwarders**

In `internal/usercommands/flee.go` delete `fleeIncludeSkillTempKey`, the `fleeAdmission` type, `TakeFleeAdmission` and `CancelFleeAdmission` (lines 17-59, keeping `fleeShortageText` in its own `const`). Replace inside `Flee`:
- `user.SetTempData(fleeIncludeSkillTempKey, nil)` -> `user.Character.CancelFleeAdmission()`
- `user.SetTempData(fleeIncludeSkillTempKey, fleeAdmission{})` -> `user.Character.PublishFleeAdmission(characters.FleeAdmission{})`
- `CancelFleeAdmission(user)` -> `user.Character.CancelFleeAdmission()`
- the ready publish -> `user.Character.PublishFleeAdmission(characters.FleeAdmission{IncludeSkill: !costResult.Short(), Ready: true})`

In `internal/hooks/NewRound_DoCombat_helpers.go`:
- line 853 `usercommands.TakeFleeAdmission(user)` -> `user.Character.TakeFleeAdmission()`
- lines 859-862 become:

```go
	admission, admitted := user.Character.TakeFleeAdmission()
	if !admitted {
		return true
	}
	includeSkill := admission.IncludeSkill
```

In `internal/hooks/CombatPhase_FleeCancellation.go` the body becomes (it now cancels for ANY character and only speaks to a player):

```go
			if !c.CancelFleeAdmission() {
				return
			}
			u := users.GetByUserId(c.GetUserId())
			if u == nil || u.Character != c {
				return
			}
			u.SendText(messaging.CategorySystem, `The fight ends before you need to flee.`)
```

Drop the `usercommands` import there. Rename the hook label from `"player_flee_terminal_cancellation"` to `"flee_terminal_cancellation"` and update the doc comment's first sentence to "closes an admitted flee, player or mob, when combat terminates ...".

Tests, mechanically:
- `TakeFleeAdmission(u)` / `usercommands.TakeFleeAdmission(u)` returning `(includeSkill, admitted)` -> `adm, admitted := u.Character.TakeFleeAdmission()` and read `adm.IncludeSkill` where the old code read `includeSkill`. Where only `admitted` was read, use `_, admitted :=`.
- `usercommands/flee_cost_test.go:193` `u.SetTempData(fleeIncludeSkillTempKey, fleeAdmission{includeSkill: false})` -> `u.Character.PublishFleeAdmission(characters.FleeAdmission{})`.
- Delete `TestTakeFleeAdmission_PendingHandoffWaitsForCommandOrCancellation` (`usercommands/flee_cost_test.go:209-222`); Step 1's first test is its replacement.

Run: `go build ./... && go vet ./internal/usercommands/ ./internal/hooks/ ./internal/characters/`
Expected: no output. If the compiler names another caller, apply the same mapping.

- [ ] **Step 6: Run the moved tests**

Run: `go test ./internal/characters/ ./internal/usercommands/ ./internal/hooks/ -run 'Flee' -count=1`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
gofmt -l internal/
git add internal/characters/flee_admission.go internal/characters/flee_admission_test.go internal/characters/character.go internal/usercommands/flee.go internal/usercommands/flee_cost_test.go internal/hooks/NewRound_DoCombat_helpers.go internal/hooks/CombatPhase_FleeCancellation.go internal/hooks/flee_cost_test.go
git commit -m "refactor(flee): the flee admission lives on the Character"
```

(`gofmt -l` must print nothing before the commit.)

---

### Task 2: `actions.BeginFlee` and the player command on it

**Model:** sonnet.

**Files:**
- Create: `internal/actions/flee.go`, `internal/actions/flee_parity_test.go`
- Modify: `internal/usercommands/flee.go`

- [ ] **Step 1: Write the failing parity test (begin half)**

`internal/actions/flee_parity_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/exit"
)

// The flee parity table (slice 4a). A player and a mob, built alike, begin
// and resolve a flee through actions.BeginFlee and actions.ResolveFlee; every
// row asserts the two come out the same. Before 4a a mob's flee was free,
// instant, and ignored roots, standing and whether it was even fighting.

const (
	fleeParityUserId   = 7401
	fleeParityMobId    = 98401
	fleeParityRoomId   = 99401
	fleeParityNorthId  = 99402
	fleeParityEastId   = 99403
	fleeRootCondId     = 9401
	fleeNoFleeCondId   = 9402
	fleeParityFlightId = "flee-parity-flight"
)

// fleeSide is one fleer after a row ran.
type fleeSide struct {
	begin      FleeBegin
	second     FleeBegin
	outcome    FleeOutcome
	state      combatphase.State
	staminaUse int
	admission  characters.FleeAdmission
	admitted   bool
}

func newFleeParityChar() *characters.Character {
	c := characters.New()
	c.Name = "Fleer"
	c.RoomId = fleeParityRoomId
	c.Mutations = map[string]int{}
	c.Validate()
	c.StaminaMax.Base = 500
	c.StaminaMax.Recalculate()
	c.Stamina = 500
	return c
}

func engageForFlee(t *testing.T, c *characters.Character) {
	t.Helper()
	if err := c.CombatPhase.TransitionToEngaging(
		combatphase.EngagingData{Target: state.ActorRef{MobInstanceId: 4242}},
		state.TransitionReason{Trigger: combatphase.TriggerAttackCommand},
	); err != nil {
		t.Fatalf("could not enter combat: %v", err)
	}
	c.CombatPhase.OnRoundTick()
}

type fleeRow struct {
	name      string
	engage    bool
	setup     func(t *testing.T, c *characters.Character)
	preferred string
	beginTwice bool
	// resolve, when set, runs ResolveFlee after an accepted begin in a room
	// with the given exits; between runs mid-flee setup (e.g. a grapple).
	resolve    bool
	exits      map[string]exit.RoomExit
	between    func(t *testing.T, c *characters.Character)
}

// fleeBoth runs one row for a player and for a mob and returns both sides.
func fleeBoth(t *testing.T, row fleeRow) (fleeSide, fleeSide) {
	t.Helper()
	run := func(actorFor func(c *characters.Character, room *rooms.Room) Actor) fleeSide {
		c := newFleeParityChar()
		room := &rooms.Room{RoomId: fleeParityRoomId, Exits: row.exits}
		actor := actorFor(c, room)
		c = actor.GetCharacter() // the mob copies the struct; use its own
		if row.setup != nil {
			row.setup(t, c)
		}
		if row.engage {
			engageForFlee(t, c)
		}
		before := c.Stamina
		side := fleeSide{begin: BeginFlee(actor, row.preferred)}
		side.staminaUse = before - c.Stamina
		if row.beginTwice {
			side.second = BeginFlee(actor, row.preferred)
		}
		if row.resolve && side.begin.Accepted {
			if row.between != nil {
				row.between(t, c)
			}
			side.outcome = ResolveFlee(actor, room)
		} else {
			side.admission, side.admitted = c.TakeFleeAdmission()
		}
		side.state = c.CombatPhase.State()
		return side
	}
	player := run(func(c *characters.Character, room *rooms.Room) Actor {
		return NewUserActorInRoom(&users.UserRecord{UserId: fleeParityUserId, Character: c}, room)
	})
	mob := run(func(c *characters.Character, room *rooms.Room) Actor {
		m := &mobs.Mob{InstanceId: fleeParityMobId, Character: *c}
		return NewMobActorInRoom(m, room)
	})
	return player, mob
}

func seedFleeConditions(t *testing.T) {
	t.Helper()
	cleanup := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		fleeRootCondId:   {ConditionId: fleeRootCondId, Name: "Rooted", TriggerCount: 1, Flags: []conditions.Flag{conditions.NoMovement}},
		fleeNoFleeCondId: {ConditionId: fleeNoFleeCondId, Name: "Frenzied", TriggerCount: 1, Flags: []conditions.Flag{conditions.NoFlee}},
	})
	t.Cleanup(cleanup)
	cleanupMut := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		fleeParityFlightId: {
			MutationId: fleeParityFlightId, Name: "Flight", Rarity: 8,
			Pros: []mutations.MutationEffect{{Type: "flag", Target: "flying"}},
		},
	})
	t.Cleanup(cleanupMut)
}

func addFleeCond(id int) func(t *testing.T, c *characters.Character) {
	return func(t *testing.T, c *characters.Character) {
		t.Helper()
		if !c.Conditions.AddCondition(id, false) {
			t.Fatalf("could not apply condition %d", id)
		}
	}
}

func knockDown(t *testing.T, c *characters.Character) {
	t.Helper()
	if c.Position == nil {
		c.Position = position.NewMachine()
	}
	r := state.TransitionReason{Trigger: "test_setup"}
	c.Position.ForceStanding(r)
	if err := c.Position.TransitionToProne(position.ProneData{}, r); err != nil {
		t.Fatalf("knock down: %v", err)
	}
}

func clinch(t *testing.T, c *characters.Character) {
	t.Helper()
	if c.Position == nil {
		c.Position = position.NewMachine()
	}
	c.Position.ForceStanding(state.TransitionReason{Trigger: "test_setup"})
	if err := c.Position.TransitionToClinch(
		position.GrappleData{Partner: state.ActorRef{UserId: 1}},
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	); err != nil {
		t.Fatalf("clinch: %v", err)
	}
}

func TestFleeParity_Begin(t *testing.T) {
	seedFleeConditions(t)
	rows := []struct {
		row      fleeRow
		accepted bool
		refusal  FleeRefusal
		short    bool
	}{
		{fleeRow{name: "rooted refuses", engage: true, setup: addFleeCond(fleeRootCondId)}, false, FleeRefuseRooted, false},
		{fleeRow{name: "no-flee refuses", engage: true, setup: addFleeCond(fleeNoFleeCondId)}, false, FleeRefuseNoFlee, false},
		{fleeRow{name: "out of combat refuses", engage: false}, false, FleeRefuseNotInCombat, false},
		{fleeRow{name: "prone refuses", engage: true, setup: knockDown}, false, FleeRefuseProne, false},
		{fleeRow{name: "grappled refuses", engage: true, setup: clinch}, false, FleeRefuseGrappled, false},
		{fleeRow{name: "paid in full is accepted with skill", engage: true}, true, FleeOK, false},
		{fleeRow{name: "spent is accepted without skill", engage: true, setup: func(t *testing.T, c *characters.Character) { c.Stamina = 0 }}, true, FleeOK, true},
	}
	for _, tc := range rows {
		t.Run(tc.row.name, func(t *testing.T) {
			p, m := fleeBoth(t, tc.row)
			for who, s := range map[string]fleeSide{"player": p, "mob": m} {
				if s.begin.Accepted != tc.accepted || s.begin.Refusal != tc.refusal || s.begin.Short != tc.short {
					t.Errorf("%s begin = %+v, want accepted=%v refusal=%v short=%v", who, s.begin, tc.accepted, tc.refusal, tc.short)
				}
				if tc.accepted {
					if s.state != combatphase.Disengaging {
						t.Errorf("%s state = %v, want Disengaging", who, s.state)
					}
					if !s.admitted || !s.admission.Ready || s.admission.IncludeSkill == tc.short {
						t.Errorf("%s admission = %+v (%v), want ready with IncludeSkill=%v", who, s.admission, s.admitted, !tc.short)
					}
				} else if s.admitted {
					t.Errorf("%s refused flee left an admission", who)
				}
			}
			if p.staminaUse != m.staminaUse {
				t.Errorf("stamina paid: player %d, mob %d; want equal", p.staminaUse, m.staminaUse)
			}
			if tc.accepted && !tc.short && p.staminaUse <= 0 {
				t.Errorf("an accepted flee paid %d stamina, want > 0", p.staminaUse)
			}
		})
	}
}

func TestFleeParity_AlreadyDisengagingRefuses(t *testing.T) {
	seedFleeConditions(t)
	p, m := fleeBoth(t, fleeRow{name: "twice", engage: true, beginTwice: true})
	for who, s := range map[string]fleeSide{"player": p, "mob": m} {
		if !s.begin.Accepted || s.second.Refusal != FleeRefuseAlready {
			t.Errorf("%s: first %+v second %+v, want accepted then FleeRefuseAlready", who, s.begin, s.second)
		}
	}
}

func TestFleeParity_FlightHalvesTheCost(t *testing.T) {
	seedFleeConditions(t)
	_, ground := fleeBoth(t, fleeRow{name: "ground", engage: true})
	fp, fm := fleeBoth(t, fleeRow{name: "flying", engage: true, setup: func(t *testing.T, c *characters.Character) {
		c.Mutations = map[string]int{fleeParityFlightId: 1}
	}})
	if fp.staminaUse != fm.staminaUse {
		t.Fatalf("flying stamina paid: player %d, mob %d; want equal", fp.staminaUse, fm.staminaUse)
	}
	if fm.staminaUse >= ground.staminaUse {
		t.Fatalf("flying paid %d, grounded %d; flight must pay less", fm.staminaUse, ground.staminaUse)
	}
}
```

If `exit` is not the package that defines `RoomExit`, use the import path `rooms` fixtures use (`grep -rn "exit.RoomExit" internal/hooks/hooks_test.go` shows `github.com/GoMudEngine/GoMud/internal/exit`); keep the imports gofmt-sorted.

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/actions/ -run TestFleeParity -count=1`
Expected: FAIL to compile, `undefined: BeginFlee`.

- [ ] **Step 3: Implement `BeginFlee`**

`internal/actions/flee.go`:

```go
package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/costs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
)

// FleeRefusal says why a flee did not begin.
type FleeRefusal int

const (
	FleeOK FleeRefusal = iota
	FleeRefuseRooted
	FleeRefuseNoFlee
	FleeRefuseAlready
	FleeRefuseNotInCombat
	FleeRefuseGrappled
	FleeRefuseProne
	FleeRefuseNotReady
)

// FleeBegin reports one flee command. Accepted means the fleer is now
// Disengaging and the round will resolve the escape; Short means it could
// not pay in full and brings no Skullduggery to the blocker contest.
type FleeBegin struct {
	Accepted bool
	Short    bool
	Refusal  FleeRefusal
}

// FleeGate is every refusal a flee command can know before it transitions,
// in the order the player has always been told them. It is exported so a
// behaviour-tree action can decide to fire instead of kiting when a flee
// could not begin. Grapple and standing duplicate CombatPhase's position veto
// on purpose: the veto is registered by the hooks package, so without these
// two checks a flee's outcome would depend on which packages are linked.
func FleeGate(c *characters.Character) FleeRefusal {
	switch {
	case c.HasConditionFlag(conditions.NoMovement):
		return FleeRefuseRooted
	case c.HasConditionFlag(conditions.NoFlee):
		return FleeRefuseNoFlee
	case c.IsDisengaging():
		return FleeRefuseAlready
	case !c.IsInCombat():
		return FleeRefuseNotInCombat
	case c.IsStandingGrapple() || c.IsGroundGrapple():
		return FleeRefuseGrappled
	case !c.IsStanding():
		return FleeRefuseProne
	}
	return FleeOK
}

// BeginFlee is the flee command, shared by players and mobs: the gates, the
// pending admission, the Disengaging transition, then one quote and partial
// commit of the flee cost. Shortage never refuses a flee (it is the only way
// out of a fight); it drops Skullduggery from the contest instead. The escape
// itself happens on the next round, in ResolveFlee. preferredExit is carried
// to that round; "" means a random passable exit.
func BeginFlee(actor Actor, preferredExit string) FleeBegin {
	c := actor.GetCharacter()

	// Any command that is not the attempt already in flight owns no pending
	// admission, so retract an orphan before any refusal returns.
	if !c.IsDisengaging() {
		c.CancelFleeAdmission()
	}
	if r := FleeGate(c); r != FleeOK {
		return FleeBegin{Refusal: r}
	}

	// Publish a pending handoff before the transition. Cost belongs only to an
	// accepted Disengaging transition.
	c.PublishFleeAdmission(characters.FleeAdmission{PreferredExit: preferredExit})
	if c.CombatPhase == nil {
		c.CancelFleeAdmission()
		return FleeBegin{Refusal: FleeRefuseNotReady}
	}
	if err := c.CombatPhase.TransitionToDisengaging(state.TransitionReason{
		Trigger: combatphase.TriggerFleeCommand,
		Actor:   state.ActorRef{UserId: actor.GetUserId(), MobInstanceId: actor.GetMobInstanceId()},
	}); err != nil {
		c.CancelFleeAdmission()
		return FleeBegin{Refusal: fleeVetoReason(c)}
	}

	bal := configs.GetBalanceConfig()
	modifier := 1.0
	if mutations.IsFlying(c.Mutations) {
		modifier = float64(bal.FlightFleeStaminaMult)
	}
	quote := c.QuoteActionCost(characters.ActionCostRequest{
		Action:   costs.ActionFlee,
		Pool:     characters.PoolStamina,
		Base:     float64(bal.FleeStaminaCost),
		Modifier: modifier,
		Units:    1,
	})
	short := c.CommitCost(quote, characters.CostPartial).Short()
	c.PublishFleeAdmission(characters.FleeAdmission{
		IncludeSkill:  !short,
		Ready:         true,
		PreferredExit: preferredExit,
	})
	return FleeBegin{Accepted: true, Short: short}
}

// fleeVetoReason names a transition the machine refused after FleeGate let it
// through (a veto registered elsewhere, or combat ending in between).
func fleeVetoReason(c *characters.Character) FleeRefusal {
	switch {
	case !c.IsInCombat():
		return FleeRefuseNotInCombat
	case c.IsStandingGrapple() || c.IsGroundGrapple():
		return FleeRefuseGrappled
	case !c.IsStanding():
		return FleeRefuseProne
	}
	return FleeRefuseNotReady
}
```

Also add a stub so the test compiles until Task 3 (Task 3 replaces it):

```go
// FleeOutcome reports one flee resolution (Task 3 fills it in).
type FleeOutcome struct{}

// ResolveFlee resolves a flee on its round (Task 3).
func ResolveFlee(actor Actor, room *rooms.Room) FleeOutcome { return FleeOutcome{} }
```

(with the `rooms` import). Delete the stub in Task 3.

- [ ] **Step 4: Run it to verify it passes**

Run: `go test ./internal/actions/ -run TestFleeParity -count=1`
Expected: PASS.

- [ ] **Step 5: Put the player command on `BeginFlee`**

`internal/usercommands/flee.go`, whole file:

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const fleeShortageText = "You break away on instinct rather than technique, too spent to use your training."

// fleeRefusalText is what the player is told for each refusal. Every line is
// unchanged from before slice 4a moved the rules into actions.BeginFlee.
var fleeRefusalText = map[actions.FleeRefusal]string{
	// A no-go root (a Jailed holding cell, 5.1c) pins the player; flee must
	// honour it or it becomes a jail-escape hole (smoke BUG-02).
	actions.FleeRefuseRooted: `You're locked in — there's nowhere to flee to.`,
	// Blood Frenzy, hamstrung, winded, tackled: you can fight, not retreat.
	actions.FleeRefuseNoFlee: `You can't break off to flee right now — you can only fight.`,
	// A second flee while the first resolves used to print nothing at all.
	actions.FleeRefuseAlready: `You're already trying to break away. Give it a moment.`,
	// Also rejects a stale queued flee after a lethal round respawned you.
	actions.FleeRefuseNotInCombat: `You're not in combat; there's nothing to flee from.`,
	actions.FleeRefuseGrappled:    `<ansi fg="red">You can't flee while grappled!</ansi>`,
	// Knockdown is common and is exactly when a player wants to run, so say
	// that standing up is what unblocks it.
	actions.FleeRefuseProne:    `<ansi fg="red">You can't flee from the ground. Stand up first!</ansi>`,
	actions.FleeRefuseNotReady: `You can't break away just yet.`,
}

func Flee(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	begin := actions.BeginFlee(actions.NewUserActorInRoom(user, room), "")
	if !begin.Accepted {
		user.SendText(messaging.CategorySystem, fleeRefusalText[begin.Refusal])
		return true, nil
	}
	if begin.Short {
		user.SendText(messaging.CategorySystem, fleeShortageText)
	}
	user.SendText(messaging.CategorySystem, `You attempt to flee...`)
	return true, nil
}
```

Before replacing, copy each refusal literal from the current file byte for byte (they contain em dashes that are player text, not prose, and must not change); the map above must match them exactly.

- [ ] **Step 6: Run the player flee tests**

Run: `go test ./internal/usercommands/ ./internal/hooks/ -run 'Flee' -count=1`
Expected: PASS (`flee_refusal_copy_test.go`, `flee_cost_test.go` in both packages unchanged in what they assert).

- [ ] **Step 7: Commit**

```bash
gofmt -l internal/
git add internal/actions/flee.go internal/actions/flee_parity_test.go internal/usercommands/flee.go
git commit -m "refactor(flee): one BeginFlee for players and mobs, player command on it"
```

---

### Task 3: `actions.ResolveFlee` and the player round on it

**Model:** sonnet.

**Files:**
- Modify: `internal/actions/flee.go`, `internal/actions/flee_parity_test.go`, `internal/hooks/NewRound_DoCombat_helpers.go`

- [ ] **Step 1: Write the failing resolve rows**

Append to `internal/actions/flee_parity_test.go`:

```go
func TestFleeParity_Resolve(t *testing.T) {
	seedFleeConditions(t)
	north := map[string]exit.RoomExit{"north": {RoomId: fleeParityNorthId}}
	two := map[string]exit.RoomExit{"north": {RoomId: fleeParityNorthId}, "east": {RoomId: fleeParityEastId}}
	rows := []struct {
		row      fleeRow
		escaped  bool
		exitName string
		grappled bool
		noExit   bool
		state    combatphase.State
	}{
		{fleeRow{name: "escapes through the only exit", engage: true, resolve: true, exits: north}, true, "north", false, false, combatphase.Idle},
		{fleeRow{name: "takes the preferred exit", engage: true, resolve: true, exits: two, preferred: "east"}, true, "east", false, false, combatphase.Idle},
		{fleeRow{name: "no exit returns to the fight", engage: true, resolve: true}, false, "", false, true, combatphase.Engaged},
		{fleeRow{name: "grappled mid-flee returns to the fight", engage: true, resolve: true, exits: north, between: clinch}, false, "", true, false, combatphase.Engaged},
	}
	for _, tc := range rows {
		t.Run(tc.row.name, func(t *testing.T) {
			p, m := fleeBoth(t, tc.row)
			for who, s := range map[string]fleeSide{"player": p, "mob": m} {
				o := s.outcome
				if !o.Fleeing || !o.Resolved {
					t.Fatalf("%s outcome %+v, want a resolved flee", who, o)
				}
				if o.Escaped() != tc.escaped || o.ExitName != tc.exitName || o.Grappled != tc.grappled || o.NoExit != tc.noExit {
					t.Errorf("%s outcome = %+v, want escaped=%v exit=%q grappled=%v noExit=%v", who, o, tc.escaped, tc.exitName, tc.grappled, tc.noExit)
				}
				if s.state != tc.state {
					t.Errorf("%s state = %v, want %v", who, s.state, tc.state)
				}
			}
		})
	}
}

func TestResolveFlee_NotDisengagingIsNotAFlee(t *testing.T) {
	c := newFleeParityChar()
	engageForFlee(t, c)
	room := &rooms.Room{RoomId: fleeParityRoomId}
	u := &users.UserRecord{UserId: fleeParityUserId, Character: c}
	if o := ResolveFlee(NewUserActorInRoom(u, room), room); o.Fleeing || o.Resolved {
		t.Fatalf("an engaged fighter resolved a flee: %+v", o)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/actions/ -run 'TestFleeParity_Resolve|TestResolveFlee' -count=1`
Expected: FAIL to compile (`o.Fleeing undefined`).

- [ ] **Step 3: Implement `ResolveFlee`** (replace the Task 2 stub)

Add to `internal/actions/flee.go` imports `combat`, `rooms`, `skills`, `targeting`, then:

```go
// FleeOutcome reports one flee round. Fleeing is false when the actor was not
// Disengaging, so the round goes on as normal. Fleeing with Resolved false
// means another resolver already owns this attempt: skip the round and say
// nothing. A resolved flee ends one of four ways: Grappled, blocked (Blocker
// set), NoExit, or escaped through ExitName to ExitRoomId.
type FleeOutcome struct {
	Fleeing    bool
	Resolved   bool
	Grappled   bool
	Blocker    *combat.FleeBlocker
	NoExit     bool
	ExitName   string
	ExitRoomId int
}

// Escaped reports a flee that got away.
func (o FleeOutcome) Escaped() bool {
	return o.Resolved && !o.Grappled && o.Blocker == nil && !o.NoExit
}

// ResolveFlee is the flee round, shared by players and mobs: consume the
// admission, refuse a fleer grappled since the command, run the blocker
// contest, practise Skullduggery only when a contest happened AND the fleer
// paid in full, pick the exit, and settle CombatPhase (back to Engaged on any
// failure, Idle on success). It does not move the fleer; the wrapper does,
// because the player and mob moves differ (Look, charmed followers and room
// entry events for one; RelocateMob and mob_flee for the other).
func ResolveFlee(actor Actor, room *rooms.Room) FleeOutcome {
	c := actor.GetCharacter()
	if !c.IsDisengaging() {
		// A terminal transition can end Disengaging before this round runs.
		// Retract that orphan; an absent handoff is a harmless no-op.
		c.TakeFleeAdmission()
		return FleeOutcome{}
	}
	admission, admitted := c.TakeFleeAdmission()
	if !admitted {
		return FleeOutcome{Fleeing: true}
	}
	out := FleeOutcome{Fleeing: true, Resolved: true}

	if c.IsStandingGrapple() || c.IsGroundGrapple() {
		out.Grappled = true
		settleFlee(c, false)
		return out
	}

	blocker, contested := combat.ResolveFleeBlockers(c, room, admission.IncludeSkill)
	// Practise only when an opposed roll happened and the fleer brought the
	// skill to it (a short payment leaves it out). Won is "got away".
	if contested && admission.IncludeSkill {
		actor.AwardResolved(blocker == nil, c.CandidateFor(string(skills.Skullduggery)))
	}
	if blocker != nil {
		out.Blocker = blocker
		settleFlee(c, false)
		return out
	}

	exitName, exitRoomId := fleeExit(room, admission.PreferredExit)
	if exitName == `` {
		out.NoExit = true
		settleFlee(c, false)
		return out
	}
	out.ExitName, out.ExitRoomId = exitName, exitRoomId

	targeting.Release(c, targeting.ReasonDisengage)
	settleFlee(c, true)
	return out
}

func settleFlee(c *characters.Character, success bool) {
	if c.CombatPhase != nil {
		c.CombatPhase.ResolveFlee(success)
	}
}

// fleeExit is the preferred exit when it is still there and unlocked, else a
// random passable one (secret and locked exits excluded).
func fleeExit(room *rooms.Room, preferred string) (string, int) {
	if preferred != `` {
		if info, ok := room.GetExitInfo(preferred); ok && !info.Lock.IsLocked() {
			return preferred, info.RoomId
		}
	}
	return room.GetRandomExit()
}
```

- [ ] **Step 4: Run it to verify it passes**

Run: `go test ./internal/actions/ -run 'TestFleeParity|TestResolveFlee' -count=1`
Expected: PASS.

- [ ] **Step 5: Put `handlePlayerFlee` on `ResolveFlee`**

Replace the body of `handlePlayerFlee` (`internal/hooks/NewRound_DoCombat_helpers.go:836-976`) with:

```go
// handlePlayerFlee resolves a player's flee on its round through the shared
// actions.ResolveFlee and renders the player's lines. Returns true when the
// player is fleeing and should skip combat this round.
func handlePlayerFlee(user *users.UserRecord, uRoom *rooms.Room, userId int) bool {
	out := actions.ResolveFlee(actions.NewUserActorInRoom(user, uRoom), uRoom)
	if !out.Fleeing {
		return false
	}
	if !out.Resolved {
		return true
	}

	switch {
	case out.Grappled:
		user.SendText(messaging.CategorySystem, `<ansi fg="red">You can't flee while grappled!</ansi>`)
		return true
	case out.Blocker != nil:
		blocker := out.Blocker
		targetTag := "mobname"
		if blocker.IsPlayer() {
			targetTag = "username"
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red-bold"><ansi fg="%s">%s</ansi> blocks you from fleeing!</ansi>`, targetTag, blocker.Name))
		excludes := []int{user.UserId}
		if blocker.IsPlayer() {
			excludes = append(excludes, blocker.UserId)
		}
		// Visual (owner ruling 2026-09-21): a reader who cannot see does not
		// witness a blocked escape at all, and one who makes out shapes reads
		// neither name.
		uRoom.SendTextVisualHidingNames(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> is blocked from fleeing by <ansi fg="%s">%s</ansi>!`, user.Character.Name, targetTag, blocker.Name), []string{user.Character.Name, blocker.Name}, excludes...)
		return true
	case out.NoExit:
		user.SendText(messaging.CategorySystem, `You can't find an exit!`)
		return true
	}

	user.SendText(messaging.CategoryRoomExit, fmt.Sprintf(`You flee to the <ansi fg="exit">%s</ansi> exit!`, out.ExitName))
	// Visual (owner ruling 2026-09-21): seeing someone break away and which
	// way they went is sight, so a reader who cannot see learns nothing.
	uRoom.SendTextVisualHidingNames(messaging.CategoryRoomExit, fmt.Sprintf(`<ansi fg="username">%s</ansi> flees to the <ansi fg="exit">%s</ansi> exit!`, user.Character.Name, out.ExitName), []string{user.Character.Name}, user.UserId)

	if err := rooms.MoveToRoom(user.UserId, out.ExitRoomId); err == nil {
		for _, instId := range uRoom.GetMobs(rooms.FindCharmed) {
			if mob := mobs.GetInstance(instId); mob != nil {
				if mob.Character.IsCharmed(userId) {
					mob.Command(out.ExitName)
				}
			}
		}

		newRoom := rooms.LoadRoom(out.ExitRoomId)
		usercommands.Look(``, user, newRoom, events.CmdSecretly)

		// Entering a room by any means triggers its entry hooks; without this
		// fleeing INTO the newcomer tutorial's final room (6467) stranded the
		// player (2026-07-17 playtest).
		behaviortree.TryRoomBehavior(out.ExitRoomId, behaviortree.EventContext{
			EventType: "room_enter",
			UserId:    user.UserId,
			RoomId:    out.ExitRoomId,
		})
	}

	return true
}
```

Copy the four player literals from the current body byte for byte before replacing (compare with `git diff` afterwards: only structure may change, no literal). Remove imports the file no longer uses (`skills`, `targeting`, `combat` if unused; `go build` names them).

- [ ] **Step 6: Run the player flee tests**

Run: `go test ./internal/hooks/ ./internal/usercommands/ -run 'Flee' -count=1`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
gofmt -l internal/
git add internal/actions/flee.go internal/actions/flee_parity_test.go internal/hooks/NewRound_DoCombat_helpers.go
git commit -m "refactor(flee): one ResolveFlee for players and mobs, player round on it"
```

---

### Task 4: `actions.RelocateMob`

**Model:** sonnet.

**Files:**
- Create: `internal/actions/relocate_mob.go`, `internal/actions/relocate_mob_test.go`
- Modify: `internal/mobcommands/go.go`, `internal/mobcommands/go_retarget_notice_test.go`, `internal/actions/retarget_notice.go` (comment)

- [ ] **Step 1: Write the failing test**

`internal/actions/relocate_mob_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// RelocateMob is the mob's move with no gate and no charge: walking (after
// its gates and, from 4b, its charge) and a successful flee both end in it.
func TestRelocateMob_MovesTheMobBetweenRooms(t *testing.T) {
	const from, to, instId = 99411, 99412, 98411
	cleanup := rooms.SeedRoomsForTest(map[int]*rooms.Room{
		from: {RoomId: from, Zone: "test", Exits: map[string]exit.RoomExit{"north": {RoomId: to}}},
		to:   {RoomId: to, Zone: "test", Exits: map[string]exit.RoomExit{"south": {RoomId: from}}},
	}, map[string]*rooms.ZoneConfig{})
	defer cleanup()

	m := &mobs.Mob{InstanceId: instId, Character: *characters.New()}
	m.Character.Name = "Walker"
	m.Character.RoomId = from
	mobs.SetInstanceForTest(instId, m)
	defer mobs.SetInstanceForTest(instId, nil)

	fromRoom, toRoom := rooms.LoadRoom(from), rooms.LoadRoom(to)
	fromRoom.AddMob(instId)

	RelocateMob(m, fromRoom, "north", toRoom)

	if contains := func(ids []int) bool {
		for _, id := range ids {
			if id == instId {
				return true
			}
		}
		return false
	}; contains(fromRoom.GetMobs(rooms.FindAll)) || !contains(toRoom.GetMobs(rooms.FindAll)) {
		t.Fatalf("mob not moved: from=%v to=%v", fromRoom.GetMobs(rooms.FindAll), toRoom.GetMobs(rooms.FindAll))
	}
	if m.Character.RoomId != to {
		t.Fatalf("mob RoomId = %d, want %d", m.Character.RoomId, to)
	}
}
```

`Room.AddMob` sets `Character.RoomId` and `Zone` (`internal/rooms/rooms.go:1150-1167`), so the last assertion holds.

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/actions/ -run TestRelocateMob -count=1`
Expected: FAIL to compile, `undefined: RelocateMob`.

- [ ] **Step 3: Implement**

Create `internal/actions/relocate_mob.go`. Move `clearRoomAggroOnDeparture` from `internal/mobcommands/go.go:19-86` into it verbatim, renamed `ClearRoomAggroOnDeparture` (its references to `actions.RetargetNotice` become `RetargetNotice`), then add:

```go
// RelocateMob moves a mob from one room to the room through exitName, with no
// gate and no charge: the caller has already decided the move happens.
// Walking (mobcommands.Go, after its lock checks) and a successful flee
// (hooks.handleMobFlee) both end here. It drops aggro the old room held on the
// mob, narrates the exit and the entry (sight-gated, with a sound fallback),
// plays the movement sounds, and pulls an NPC party's idle members after
// their leader.
func RelocateMob(mob *mobs.Mob, from *rooms.Room, exitName string, dest *rooms.Room) {
	enterFrom := `somewhere`
	if back := dest.FindExitTo(from.RoomId); back != `` {
		enterFrom = fmt.Sprintf(`the <ansi fg="exit">%s</ansi>`, back)
	}

	from.RemoveMob(mob.InstanceId)
	ClearRoomAggroOnDeparture(from, mob.InstanceId)
	dest.AddMob(mob.InstanceId)

	c := configs.GetTextFormatsConfig()

	from.SendTextVisualWithAudio(messaging.CategoryRoomExit,
		fmt.Sprintf(string(c.ExitRoomMessageWrapper),
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> leaves towards the <ansi fg="exit">%s</ansi> exit.`, mob.Character.Name, exitName),
		),
		`You hear footsteps moving away.`)

	dest.SendTextVisualWithAudio(messaging.CategoryRoomEntry,
		fmt.Sprintf(string(c.EnterRoomMessageWrapper),
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> enters from %s.`, mob.Character.Name, enterFrom),
		),
		`You hear footsteps approaching.`)

	dest.SendTextToExits(`You hear someone moving around.`, true, from.GetPlayers(rooms.FindAll)...)

	from.PlaySound(`room-exit`, `movement`)
	dest.PlaySound(`room-enter`, `movement`)

	pullMobPartyThrough(mob, from, exitName)
}
```

and move the party block (`go.go:230-266`, with its comment) into:

```go
// pullMobPartyThrough re-issues the leader's exit on every idle NPC party
// member still in the old room. In-combat members stay in their fight.
func pullMobPartyThrough(leader *mobs.Mob, from *rooms.Room, exitName string) {
	// body: the moved block, with mob -> leader and room -> from
}
```

(write the moved block out in full, not the placeholder comment).

In `internal/mobcommands/go.go`:
- delete `clearRoomAggroOnDeparture`; the teleport path (line 137) calls `actions.ClearRoomAggroOnDeparture(room, mob.InstanceId)`.
- replace lines 184-266 (the far-side check through the party block) with:

```go
		// Entering through the far side of a locked door would unlock it; for
		// now mobs do not do that.
		if back := destRoom.FindExitTo(room.RoomId); back != `` {
			if backInfo, _ := destRoom.GetExitInfo(back); backInfo.Lock.IsLocked() {
				return true, nil
			}
		}

		actions.RelocateMob(mob, room, exitName, destRoom)
```

keeping the waypoint `noop` block after it. `sendMovementMessage` stays (the teleport uses it). Remove imports `go build` reports unused (`parties`, `targeting`, `users`, `state` likely).

In `internal/mobcommands/go_retarget_notice_test.go` lines 81 and 154: `clearRoomAggroOnDeparture(` -> `actions.ClearRoomAggroOnDeparture(` (add the import); update the two comments naming it. In `internal/actions/retarget_notice.go:28` change `mobcommands.clearRoomAggroOnDeparture` to `ClearRoomAggroOnDeparture (relocate_mob.go)`.

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/actions/ ./internal/mobcommands/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
gofmt -l internal/
git add internal/actions/relocate_mob.go internal/actions/relocate_mob_test.go internal/actions/retarget_notice.go internal/mobcommands/go.go internal/mobcommands/go_retarget_notice_test.go
git commit -m "refactor(mobs): lift the mob move tail into actions.RelocateMob"
```

---

### Task 5: The mob flees through the shared bodies

**Model:** sonnet (opus if the hooks round test flakes; the contest must be forced, not retried).

**Files:**
- Modify: `internal/mobcommands/flee.go`, `internal/hooks/NewRound_DoCombat_helpers.go`, `internal/hooks/NewRound_DoCombat.go`
- Create: `internal/hooks/mob_flee_round_test.go`

- [ ] **Step 1: Write the failing hooks tests**

`internal/hooks/mob_flee_round_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/targeting"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// Slice 4a: a mob's flee used to resolve inside the command, free, ignoring
// standing. It now enters Disengaging and escapes on the next round through
// the same resolver as a player.

// fleeingMob readies mob 100 (seedAllRegistries) to fight player 1 in room 1.
func fleeingMob(t *testing.T) (*mobs.Mob, *rooms.Room) {
	t.Helper()
	m := mobs.GetInstance(100)
	require.NotNil(t, m)
	require.NoError(t, m.Character.Validate())
	m.Character.StaminaMax.Value = 1000
	m.Character.Stamina = 1000
	m.Character.SetAggro(1, 0, characters.DefaultAttack)
	m.Character.CombatPhase.OnRoundTick()
	require.True(t, m.Character.IsInCombat(), "fixture: mob must be fighting")
	return m, rooms.LoadRoom(1)
}

func mobInRoom(room *rooms.Room, instId int) bool {
	for _, id := range room.GetMobs(rooms.FindAll) {
		if id == instId {
			return true
		}
	}
	return false
}

func TestMobFlee_EscapesOnTheNextRoundNotInTheCommand(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	m, room := fleeingMob(t)

	begin := actions.BeginFlee(actions.NewMobActorInRoom(m, room), "")
	require.True(t, begin.Accepted, "begin = %+v", begin)
	require.True(t, m.Character.IsDisengaging())
	require.True(t, mobInRoom(room, m.InstanceId), "the command itself must not move the mob")
	require.Less(t, m.Character.Stamina, 1000, "a mob's flee costs stamina")

	require.True(t, handleMobFlee(m, room))
	require.False(t, mobInRoom(room, m.InstanceId), "the round did not move the mob out")
	require.True(t, mobInRoom(rooms.LoadRoom(2), m.InstanceId), "the mob is not in the room north")
	require.False(t, m.Character.IsInCombat())
}

func TestMobFlee_BlockedMobReturnsToTheFight(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	cfg := configs.GetConfig()
	cfg.Balance.ContestFloor = 0
	configs.SetConfigForTest(t, cfg)

	m, room := fleeingMob(t)
	m.Character.Stats.Dexterity.ValueAdj = 1
	u := users.GetByUserId(1)
	require.NoError(t, u.Character.Validate())
	u.Character.Stats.Dexterity.ValueAdj = 100
	u.Character.SetAggro(0, m.InstanceId, characters.DefaultAttack)

	require.True(t, actions.BeginFlee(actions.NewMobActorInRoom(m, room), "").Accepted)
	events.DrainQueuedMessagesForTest(1)
	require.True(t, handleMobFlee(m, room))
	require.True(t, mobInRoom(room, m.InstanceId), "a blocked mob left the room")
	require.Equal(t, combatphase.Engaged, m.Character.CombatPhase.State())
}

func TestMobFlee_KnockedDownMobCannotFlee(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	m, room := fleeingMob(t)
	setCombatPositionParallel(&m.Character, position.Prone)

	begin := actions.BeginFlee(actions.NewMobActorInRoom(m, room), "")
	require.Equal(t, actions.FleeRefuseProne, begin.Refusal)
	require.False(t, m.Character.IsDisengaging())
}

func TestMobFlee_CorneredMobStaysInTheFight(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	m, room := fleeingMob(t)
	room.Exits = nil

	require.True(t, actions.BeginFlee(actions.NewMobActorInRoom(m, room), "").Accepted)
	require.True(t, handleMobFlee(m, room))
	require.True(t, m.Character.IsInCombat(), "a cornered mob dropped its fight")
	require.Equal(t, combatphase.Engaged, m.Character.CombatPhase.State())
}

func TestMobFlee_CombatEndingRetractsTheAdmission(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	m, room := fleeingMob(t)

	require.True(t, actions.BeginFlee(actions.NewMobActorInRoom(m, room), "").Accepted)
	targeting.Release(&m.Character, targeting.ReasonDisengage)
	require.False(t, m.Character.IsInCombat())
	_, admitted := m.Character.TakeFleeAdmission()
	require.False(t, admitted, "combat ended but the mob's flee admission survived")
}

func TestHandleMobCombat_DisengagingMobEscapes(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	m, room := fleeingMob(t)

	require.True(t, actions.BeginFlee(actions.NewMobActorInRoom(m, room), "").Accepted)
	handleMobCombat(events.NewRound{RoundNumber: 7})
	require.False(t, mobInRoom(room, m.InstanceId), "the mob round pass did not resolve the flee")
}
```

`TestMobFlee_CombatEndingRetractsTheAdmission` already passes after Task 1 (the hook cancels for any character); it stays as the regression. Confirm it could fail: temporarily restore the `users.GetByUserId` early return before the cancel in `CombatPhase_FleeCancellation.go`, see it fail, revert.

- [ ] **Step 2: Run to verify the rest fail**

Run: `go test ./internal/hooks/ -run 'TestMobFlee|TestHandleMobCombat_Disengaging' -count=1`
Expected: FAIL to compile, `undefined: handleMobFlee`.

- [ ] **Step 3: Implement `handleMobFlee` and call it**

Append to `internal/hooks/NewRound_DoCombat_helpers.go`, after `handlePlayerFlee`:

```go
// handleMobFlee is the mob twin of handlePlayerFlee: the same
// actions.ResolveFlee, the mob's own room lines, and on success an uncharged
// actions.RelocateMob (a player's flee pays no movement cost either) and the
// mob_flee behaviour event. Returns true when the mob is fleeing and should
// skip combat this round.
func handleMobFlee(mob *mobs.Mob, room *rooms.Room) bool {
	out := actions.ResolveFlee(actions.NewMobActorInRoom(mob, room), room)
	if !out.Fleeing {
		return false
	}
	if !out.Resolved {
		return true
	}

	name := mob.Character.Name
	switch {
	case out.Grappled:
		// The grappler sees the hold working, as at command time.
		room.SendTextVisual(messaging.CategoryGrappleFlow,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to break free but you've got them locked down!`, name))
		return true
	case out.Blocker != nil:
		// Passing the NAME matters: SendTextVisual alone falls back to the
		// tag-based "a figure", uncapitalised at a sentence start.
		room.SendTextVisualHidingNames(messaging.CategoryRoomExit,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to flee but is blocked!`, name),
			[]string{name})
		return true
	case out.NoExit:
		// Cornered: the mob stays in the fight, and the room sees it try.
		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> looks around frantically for an escape but finds none!`, name))
		return true
	}

	room.SendTextVisual(messaging.CategoryRoomExit,
		fmt.Sprintf(`<ansi fg="mobname">%s</ansi> flees!`, name))
	if dest := rooms.LoadRoom(out.ExitRoomId); dest != nil {
		actions.RelocateMob(mob, room, out.ExitName, dest)
	}
	behaviortree.TryMobBehavior(mob.InstanceId, behaviortree.EventContext{
		EventType: "mob_flee",
		RoomId:    mob.Character.RoomId,
	})
	return true
}
```

In `internal/hooks/NewRound_DoCombat.go`, inside the `if mob.Character.IsInCombat() {` block, after the `ValidateAggro` block and before its closing brace (line 338):

```go
			// A mob flees by the player's rules (slice 4a): its flee command
			// only entered Disengaging, and the escape resolves here, a round
			// later, at the same point as handlePlayerFlee in the player pass.
			if handleMobFlee(mob, mobRoom) {
				continue
			}
```

- [ ] **Step 4: Put `mobcommands.Flee` on `BeginFlee`**

`internal/mobcommands/flee.go`, whole file:

```go
package mobcommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Flee starts a mob's flee through the player's rules (actions.BeginFlee):
// it must be fighting, standing, unrooted and not frenzied, it pays the flee
// cost, and it enters Disengaging. The escape resolves on the next round in
// hooks.handleMobFlee. rest, when given, is the exit the mob would rather
// take (a kiting archer passes the exit toward home).
func Flee(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	// Non-combatant mobs never flee (they should never be in combat).
	if mob.IsNonCombatant() {
		return true, nil
	}

	begin := actions.BeginFlee(actions.NewMobActorInRoom(mob, room), strings.TrimSpace(rest))

	// Every refusal is silent for a mob except the grapple: the player holding
	// it needs to see the hold working.
	if begin.Refusal == actions.FleeRefuseGrappled {
		room.SendTextVisual(messaging.CategoryGrappleFlow,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to break free but you've got them locked down!`, mob.Character.Name))
	}
	return true, nil
}
```

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/hooks/ ./internal/mobcommands/ ./internal/actions/ -count=1`
Expected: PASS. If `TestMobFlee_BlockedMobReturnsToTheFight` fails because the player was not a blocker, read `combat.ResolveFleeBlockers` (`internal/combat/flee.go:59-120`) for the blocker filter and satisfy it in the fixture; never loosen the assertion or retry.

- [ ] **Step 6: Commit**

```bash
gofmt -l internal/
git add internal/mobcommands/flee.go internal/hooks/NewRound_DoCombat_helpers.go internal/hooks/NewRound_DoCombat.go internal/hooks/mob_flee_round_test.go
git commit -m "feat(flee): mobs flee by the player's rules, a round later"
```

---

### Task 6: Flee callers: pack scatter, out-of-combat escapes, kiting

**Model:** sonnet.

**Files:**
- Modify: `internal/hooks/MobDeath_PackFlee.go`, `internal/behaviortree/actions_combat.go`, `internal/behaviortree/actions_archer.go`, `internal/behaviortree/actions_archer_test.go`
- Create: `internal/behaviortree/actions_flee_test.go`
- Test: `internal/hooks/predator_hooks_test.go` (add one test)

- [ ] **Step 1: Write the failing tests**

Append to `internal/hooks/predator_hooks_test.go`:

```go
// Owner ruling (2026-09-28, open question 1): when a packmate dies, only
// packmates already fighting flee; idle ones stay put and are not counted in
// the scatter line.
func TestPackFlee_IdlePackmateStaysPut(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob := mobs.GetInstance(100)
	require.NotNil(t, mob)
	require.False(t, mob.Character.IsInCombat(), "fixture: mob 100 starts idle")
	events.DrainQueuedInputsForTest(mob.InstanceId)

	PackFlee(events.MobDeath{MobId: 1, InstanceId: 999, RoomId: 1, CharacterName: "Skeleton"})

	for _, cmd := range events.DrainQueuedInputsForTest(mob.InstanceId) {
		if cmd == "flee" {
			t.Fatal("an idle packmate was told to flee")
		}
	}
}
```

(If `DrainQueuedInputsForTest` returns a different shape, match `archer_kiting_test.go`'s use of it.)

`internal/behaviortree/actions_flee_test.go`:

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// A flee out of combat is refused by the player's rules (slice 4a), but
// authored trees flee out of combat: the thief steals on mob_idle and then
// flees. actFlee walks away instead when the mob is not fighting.
func TestActFlee_OutOfCombatWalksAway(t *testing.T) {
	const here, there = 12, 13
	cleanup := rooms.SeedRoomsForTest(map[int]*rooms.Room{
		here:  {RoomId: here, Zone: "test", Exits: map[string]exit.RoomExit{"north": {RoomId: there}}},
		there: {RoomId: there, Zone: "test", Exits: map[string]exit.RoomExit{"south": {RoomId: here}}},
	}, map[string]*rooms.ZoneConfig{})
	defer cleanup()

	mob := newTestMob(t)
	mob.Character.RoomId = here
	queuedCmds(mob.InstanceId)

	if got := LookupAction("flee")(nil, &EvalContext{InstanceId: mob.InstanceId, RoomId: here}); got != Success {
		t.Fatalf("flee = %v, want Success", got)
	}
	if cmds := queuedCmds(mob.InstanceId); !contains(cmds, "go north") {
		t.Errorf("out-of-combat flee queued %v, want 'go north'", cmds)
	}
}

func TestActFlee_InCombatFlees(t *testing.T) {
	const here = 14
	cleanup := rooms.SeedRoomsForTest(map[int]*rooms.Room{
		here: {RoomId: here, Zone: "test"},
	}, map[string]*rooms.ZoneConfig{})
	defer cleanup()

	target := seedTargetMob(t, 406, here)
	mob := newTestMob(t)
	mob.Character.RoomId = here
	mob.Character.SetAggro(0, target.InstanceId, characters.DefaultAttack)
	queuedCmds(mob.InstanceId)

	if got := LookupAction("flee")(nil, &EvalContext{InstanceId: mob.InstanceId, RoomId: here}); got != Success {
		t.Fatalf("flee = %v, want Success", got)
	}
	if cmds := queuedCmds(mob.InstanceId); !contains(cmds, "flee") {
		t.Errorf("in-combat flee queued %v, want 'flee'", cmds)
	}
}
```

In `internal/behaviortree/actions_archer_test.go:272-273` change the expectation to `"flee north"` and the message to `"expected 'flee north' queued, got %v"`; add above the `LookupAction` call a comment: `// Owner ruling 4 (2026-09-28): kiting out of melee is a flee, toward home.`

`newTestMob` (`conditions_test.go:52`) builds a bare mob with no `CombatPhase`, so it is out of combat; `SetAggro` creates the phase lazily (`characters/engagement_storage.go:141`), which is what puts the keep_distance and in-combat mobs into combat.

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/behaviortree/ -run 'TestActFlee|TestActKeepDistance' -count=1` and `go test ./internal/hooks/ -run TestPackFlee_IdlePackmateStaysPut -count=1`
Expected: FAIL (`flee` queued instead of `go north`; `go north` instead of `flee north`; the idle packmate was told to flee).

- [ ] **Step 3: Implement**

`internal/hooks/MobDeath_PackFlee.go`, after the `PackFleeImmune` check:

```go
		// Owner ruling (2026-09-28): only packmates already fighting flee. An
		// idle one stays put; a flee out of combat is refused anyway (slice
		// 4a), so counting it would print a scatter nobody performs.
		if !mob.Character.IsInCombat() {
			continue
		}
```

`internal/behaviortree/actions_combat.go`, `actFlee`:

```go
// actFlee flees a fight through the player's rules (slice 4a). Out of combat
// there is nothing to flee and the flee command refuses, so the mob walks
// away instead, toward home when it can: the thief's steal-and-run and a
// skittish animal's bolt still leave the room.
func actFlee(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if !mob.Character.IsInCombat() {
		room := rooms.LoadRoom(mob.Character.RoomId)
		if room == nil {
			return Failure
		}
		dir := pickRetreatExit(mob, room)
		if dir == "" {
			return Failure
		}
		mob.Command("go " + dir)
		return Success
	}
	mob.Command("flee")
	return Success
}
```

`internal/behaviortree/actions_archer.go`, `actKeepDistance`: after the `health_percent` check add

```go
	// Kiting out of melee is a flee (owner ruling 4, 2026-09-28): the blocker
	// contest, the flee cost and a round disengaging. When a flee could not
	// even begin (rooted, frenzied, knocked down, grappled), fail so the
	// selector falls through to fighting.
	if actions.FleeGate(&mob.Character) != actions.FleeOK {
		return Failure
	}
```

and replace `mob.Command("go " + dir)` with `mob.Command("flee " + dir)`. Update the doc comment: "picks a passable exit, preferring the one toward home, and flees through it (a flee since slice 4a: it can be blocked, costs stamina and takes a round)", and "stepping out of the room" becomes "escaping the room". Add the `actions` import.

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/behaviortree/ ./internal/hooks/ -count=1`
Expected: PASS, including `archer_kiting_test.go`.

- [ ] **Step 5: Commit**

```bash
gofmt -l internal/
git add internal/hooks/MobDeath_PackFlee.go internal/hooks/predator_hooks_test.go internal/behaviortree/actions_combat.go internal/behaviortree/actions_archer.go internal/behaviortree/actions_archer_test.go internal/behaviortree/actions_flee_test.go
git commit -m "feat(flee): pack scatter, out-of-combat escapes and kiting follow the flee rules"
```

---

### Task 7: The re-fork guard

**Model:** haiku.

**Files:**
- Create: `flee_wrapper_guard_test.go` (repo root)

- [ ] **Step 1: Write the guard**

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

// TestFleeWrappersDoNotReFork: the flee rules live in actions.BeginFlee and
// actions.ResolveFlee. If a command wrapper transitions CombatPhase, prices
// or charges the flee, or runs the blocker contest itself, the player and mob
// flees have forked again (until slice 4a the mob's was free, instant and
// ignored standing).
func TestFleeWrappersDoNotReFork(t *testing.T) {
	forbidden := regexp.MustCompile(`TransitionToDisengaging|QuoteActionCost|CommitCost|FleeStaminaCost|ResolveFleeBlockers`)
	for _, path := range []string{"internal/usercommands/flee.go", "internal/mobcommands/flee.go"} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if loc := forbidden.FindIndex(b); loc != nil {
			t.Errorf("%s applies flee rules itself (%q); call actions.BeginFlee instead", path, b[loc[0]:loc[1]])
		}
	}
}

// TestFleeResolutionStaysInActions: only actions (and the packages that own
// the primitives) may run the blocker contest or settle a flee's phase.
func TestFleeResolutionStaysInActions(t *testing.T) {
	resolve := regexp.MustCompile(`ResolveFleeBlockers\(|CombatPhase\.ResolveFlee\(`)
	allowed := []string{"internal/actions/", "internal/combat/", "internal/state/combatphase/"}
	checked := 0
	for _, root := range []string{"internal", "modules"} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			p := filepath.ToSlash(path)
			if d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			for _, a := range allowed {
				if strings.HasPrefix(p, a) {
					return nil
				}
			}
			checked++
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if loc := resolve.FindIndex(b); loc != nil {
				t.Errorf("%s resolves a flee itself (%q); call actions.ResolveFlee instead", p, b[loc[0]:loc[1]])
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	if checked == 0 {
		t.Fatal("walked no files; the guard is broken, not the code")
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test . -run 'TestFleeWrappersDoNotReFork|TestFleeResolutionStaysInActions' -count=1`
Expected: PASS.

- [ ] **Step 3: Prove each half can fail**

Append `// user.Character.CombatPhase.ResolveFlee(false)` as the last line of `internal/hooks/MobDeath_PackFlee.go`, and `// FleeStaminaCost` as the last line of `internal/mobcommands/flee.go`. Run Step 2's command: expect two failures naming those files. Remove both lines with the Edit tool, rerun, expect PASS.

- [ ] **Step 4: Commit**

```bash
git add flee_wrapper_guard_test.go
git commit -m "test(flee): guard the flee wrappers against re-forking"
```

---

### Task 8: Docs, full gate, boot check

**Model:** sonnet.

**Files:**
- Modify: `internal/actions/context.md`, `internal/characters/context.md`, `internal/mobcommands/context.md`, `internal/usercommands/context.md`, `internal/hooks/context.md`, `internal/behaviortree/context.md`, `docs/PATCH_NOTES.md`

- [ ] **Step 1: context.md**

Verify every symbol with `Select-String -Path internal\<pkg>\*.go -Pattern '^(func|type|const|var)\s'` before naming it.
- `actions`: `flee.go` (`FleeGate`, `BeginFlee`, `ResolveFlee`, `FleeBegin`, `FleeOutcome`, `FleeRefusal` values) and `relocate_mob.go` (`RelocateMob`, `ClearRoomAggroOnDeparture`); note the wrappers must not re-fork (guard name).
- `characters`: `FleeAdmission`, `PublishFleeAdmission`, `TakeFleeAdmission`, `CancelFleeAdmission`; pending versus ready.
- `mobcommands`: `Flee` only begins; `Go` relocates through `actions.RelocateMob`; `clearRoomAggroOnDeparture` is gone.
- `usercommands`: `Flee` is a wrapper; `TakeFleeAdmission`/`CancelFleeAdmission` are gone.
- `hooks`: `handleMobFlee` in the mob pass; `mob_disengaging` now fires for mobs; the cancellation hook covers mobs; `PackFlee` moves only fighting packmates.
- `behaviortree`: `flee` walks out of combat; `keep_distance` flees toward home.
Run `python tools/context_md_audit.py` and expect no new phantom symbols for these six packages.

- [ ] **Step 2: Patch notes**

Add a dated `2026-09-29` entry to `docs/PATCH_NOTES.md`, player-facing, no numbers, no em dashes:

```markdown
- Monsters now flee by the same rules you do. A monster that turns to run
  takes a moment to break away, and you get that moment to block it. A
  monster knocked off its feet cannot flee until it stands, running tires it
  out, and a cornered one keeps fighting instead of giving up the fight.
- When a pack animal falls, only its packmates already in the fight scatter.
- Archers pinned in melee now have to flee to get clear, so they can be
  blocked while they try.
```

- [ ] **Step 3: Full gate**

```bash
gofmt -l internal/ modules/
go vet ./...
go build ./...
go test ./... -count=1
golangci-lint run --new-from-merge-base=origin/master
```

Expected: gofmt and vet print nothing; every package `ok`; lint `0 issues`. A root guard that keys narration sites or functions by file (`TestNarrationSitesMatchViewpointAudit`, `TestEveryTrioLiteralNamesAllThreeRoles`, `lookup_viewer_guard_test.go`, `move_narration_migration_guard_test.go`) may now find a moved line under its new file: re-key that entry to the new `file|...` in the same commit, never delete it. A pre-existing failure unrelated to flee is reported, not fixed here; check with `git stash`-free comparison by running the same test on a detached master worktree.

- [ ] **Step 4: Boot check**

Per `dogmud-shipping`:

```bash
git worktree add --detach C:/tmp/dogmud-boot-check HEAD
cp "C:/Users/Calabe Davis/workspace/DOGMud/_datafiles/config.yaml" C:/tmp/dogmud-boot-check/_datafiles/config.yaml
cd C:/tmp/dogmud-boot-check && go build -o boot-check.exe .
timeout 180 ./boot-check.exe > boot.log 2>&1
grep -cE "^panic:|goroutine [0-9]+ \[running\]|runtime error" boot.log
grep -c "Server Ready" boot.log
```

Expected: exit 124, `0`, `1`. Remove the worktree afterwards (PowerShell `Remove-Item -Recurse -Force` if Windows holds a lock, then `git worktree prune`).

- [ ] **Step 5: Commit**

```bash
git add internal/actions/context.md internal/characters/context.md internal/mobcommands/context.md internal/usercommands/context.md internal/hooks/context.md internal/behaviortree/context.md docs/PATCH_NOTES.md
git commit -m "docs(flee): context.md and patch notes for mob flee parity"
```

(Add any guard file re-keyed in Step 3 to this commit by name.)

---

### Task 9: Playtest and PR

**Model:** opus (judgment on findings).

- [ ] **Step 1: Playtest**

Load `dogmud-playtesting` and follow it (ephemeral goals file, `--checkout` of this branch, never kill the owner's server). Goals, each checking nothing freezes or loops:
1. Fight a generic-fighter mob below a quarter health: it announces nothing new, stays a round, then flees; a second run with a blocker present sees "tries to flee but is blocked!" and the mob keeps fighting.
2. Trip or bash a hurt mob: it cannot flee until it stands.
3. Kill one wolf of a pack while a second fights you and a third idles: the fighting one flees, the idle one stays.
4. Fight an archer-archetype mob in melee: it tries to flee toward home, can be blocked, and fires from the next room after escaping.
5. Let a thief mob steal: it walks out of the room afterwards.
6. A player flee still works unchanged (one round, the same lines).

Extract findings to memory (reports are gitignored).

- [ ] **Step 2: Push and open the PR**

```bash
git push -u origin feature/flee-parity-4a
gh pr create --repo pruuk/DOGMud --base master --head feature/flee-parity-4a --title "feat(flee): mobs flee by the player's rules (parity slice 4a)" --body-file <scratchpad>/pr-body.md
```

Body: summary of the 4a parity table, the three planning decisions (pre-check, out-of-combat `flee` walks, party pull kept), the gate results with counts, the boot check, the playtest outcome, and the spec and plan paths, ending with the attribution line. Read back the URL `gh` prints and confirm it says `pruuk/DOGMud`. CI minutes are exhausted this month, so the local gate is the merge gate; say so in the body.

---

## Self-review

- Spec coverage: `BeginFlee` (T2), `ResolveFlee` (T3), admission on the Character (T1), cancellation for any character (T1, regression in T5), both command wrappers (T2, T5), `handlePlayerFlee` (T3), `handleMobFlee` at the F8 point (T5), `RelocateMob` (T4), per-actor narration kept (T3, T5), guard proven able to fail (T7), every parity-table row (T2 begin rows, T3 resolve rows, T5 blocked, next-round, prone and cornered), PackFlee ruling 1 (T6), `keep_distance` ruling 4 with its exit preference and CombatMemory refresh kept (T6), `context.md` for the six packages (T8), gate and playtest (T8, T9).
- Beyond the spec, stated in "Design decisions": the `FleeGate` pre-check, `actFlee` walking out of combat (thief), the party pull in `RelocateMob`.
- Names used across tasks: `FleeAdmission{IncludeSkill, Ready, PreferredExit}`, `PublishFleeAdmission`, `TakeFleeAdmission() (FleeAdmission, bool)`, `CancelFleeAdmission() bool`, `FleeGate`, `BeginFlee(actor, preferredExit) FleeBegin`, `ResolveFlee(actor, room) FleeOutcome`, `FleeOutcome.Escaped()`, `RelocateMob(mob, from, exitName, dest)`, `ClearRoomAggroOnDeparture(room, instId)`, `handleMobFlee(mob, room) bool`.
