# Drink Path Unification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Mobs, AI companion included, drink through the player's drink path, and a mob's heal-over-time actually heals.

**Architecture:** Move the body of `usercommands.Drink` verbatim into `actions.Drink(actor DrinkActor, rest string) DrinkResult`, the established shared-action pattern (`actions.Buy`, `actions.InitiateCast`). Both command files become thin wrappers. A shared tick helper gives the mob round tick the zero-amount fill-in the player tick already has. A repo-root guard stops the wrappers from re-forking.

**Tech Stack:** Go, `go test`.

**Spec:** `docs/superpowers/specs/2026-09-28-drink-path-unification-design.md` (owner-approved 2026-09-28). Its 13-row facts table is authoritative.

## Extra facts for this plan (verified 2026-09-28, master `8c6561c5a`)

| # | Fact | Where |
|---|---|---|
| P1 | `UserActor{User, Room}` and `MobActor` are the two real `Actor`s; eight test fakes also implement `Actor` (`internal/actions/{consider,economy,forage,salvage,scan,search,sleep,steal}_test.go`), so `Actor` itself must NOT grow | `internal/actions/actor_user.go`, `actor_mob.go` |
| P2 | `questengine.NewGameBridge(user *users.UserRecord, roomId int)` needs a user record; `actions/buy.go` already imports `questengine` | `internal/questengine/bridge.go:35` |
| P3 | The player body's room lines: spoiled `drinks something and immediately gags` (line ~200) and `drinks <item>` (line ~257), both `room.SendTextVisual(CategoryMobEmote, ..., user.UserId)` with `<ansi fg="username">`; the mob line uses `<ansi fg="mobname">` and excludes no one | `internal/usercommands/drink.go`, `internal/mobcommands/drink.go:33` |
| P4 | `refuseWhileBusy(user, verb)` is `if user.Character.IsActing() { SendText(red "You can't %s while focused on your work. Finish or be interrupted first."); return true }` | `internal/usercommands/busy_refuse.go:18-25` |
| P5 | The player tick's zero-amount fill-in (inline) is at `NewRound_UserRoundTick.go:312-325`; the mob tick's `if condition.TickAmount != 0` is at `NewRound_MobRoundTick.go:222` | `internal/hooks/` |
| P6 | Player drink strings contain em dashes (grapple refusal line ~131, toxicity refusal ~227, aging lines ~265-271); house style forbids them in player text | `internal/usercommands/drink.go` |

## File map

| File | Change |
|---|---|
| `internal/mobs/mobs.go` | `Mob.AddConditionScaled` |
| `internal/actions/actor_user.go`, `actor_mob.go` | the two doors |
| `internal/actions/drink.go` (new) | `DrinkActor`, `DrinkResult`, `Drink`, moved helpers and constants |
| `internal/actions/drink_*_test.go` (new/moved) | moved and new tests |
| `internal/usercommands/drink.go`, `internal/mobcommands/drink.go` | thin wrappers |
| `internal/hooks/condition_tick_amount.go` (new), `NewRound_UserRoundTick.go`, `NewRound_MobRoundTick.go` | shared fill-in |
| `drink_wrapper_guard_test.go` (new, repo root) | re-fork guard |
| `condition_apply_path_guard_test.go` | re-keyed sites |
| context.md for actions, usercommands, mobcommands, hooks, mobs; `docs/PATCH_NOTES.md`; `docs/README.md` | docs |

Models: Tasks 1, 2, 6 sonnet; Tasks 3, 4, 5 opus.

---

### Task 0: Worktree

- [ ] `git worktree add ../DOGMud-drink -b feature/drink-path-unification docs/drink-unification-spec` from the main checkout; then in the worktree `go build ./... && go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ ./internal/hooks/ -count=1` must pass.

---

### Task 1: Condition doors for both actors

**Files:** Modify `internal/mobs/mobs.go` (after `AddCondition`, ~line 869), `internal/actions/actor_user.go`, `internal/actions/actor_mob.go`. Test: `internal/mobs/mob_condition_scaled_test.go`, `internal/actions/drink_actor_test.go`.

- [ ] **Step 1: Failing tests.**

```go
// internal/mobs/mob_condition_scaled_test.go
package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestMobAddConditionScaledQueuesTheMultiplier(t *testing.T) {
	m := &Mob{InstanceId: 88101}
	events.DrainQueuedConditionsForTest(0) // clear
	m.AddConditionScaled(55, 1.3, "drink")
	var got *events.Condition
	for _, e := range events.DrainQueuedMobConditionsForTest(88101) {
		e := e
		got = &e
	}
	if got == nil || got.ConditionId != 55 || got.DurationMult != 1.3 || got.Source != "drink" {
		t.Fatalf("queued %+v, want condition 55 at DurationMult 1.3 from drink", got)
	}
}
```

Before writing it, grep `internal/events` for the test drain helpers (`DrainQueuedConditionsForTest` exists and is keyed by user id). If no mob-keyed drain exists, add `DrainQueuedMobConditionsForTest(mobInstanceId int) []events.Condition` beside the user one, mirroring it exactly, and say so in the commit.

```go
// internal/actions/drink_actor_test.go
package actions

var (
	_ DrinkActor = (*UserActor)(nil)
	_ DrinkActor = (*MobActor)(nil)
)
```

- [ ] **Step 2:** `go test ./internal/mobs/ ./internal/actions/ -run 'TestMobAddConditionScaled' -count=1` fails to compile.

- [ ] **Step 3: Implement.** `mobs.go`:

```go
// AddConditionScaled queues a condition whose duration is scaled by
// durationMult through the event path, the mob twin of
// UserRecord.AddConditionScaled (drink path unification). A non-positive
// multiplier means the authored duration.
func (m *Mob) AddConditionScaled(conditionId int, durationMult float64, source string) {
	if durationMult <= 0 {
		durationMult = 1.0
	}
	events.AddToQueue(events.Condition{
		MobInstanceId: m.InstanceId,
		ConditionId:   conditionId,
		Source:        source,
		DurationMult:  durationMult,
		LifeEpoch:     m.Character.LifeEpoch,
	})
}
```

`actions/drink.go` (new file, first content):

```go
package actions

// DrinkActor is an Actor that can receive a potion's conditions at a scaled
// duration or an exact magnitude. It is its own interface, not two new Actor
// methods, because eight test fakes implement Actor and none of them drinks.
type DrinkActor interface {
	Actor
	AddConditionScaled(conditionId int, durationMult float64, source string)
	AddConditionMagnitude(conditionId int, triggers int, magnitude float64, source string)
}
```

`actor_user.go`:

```go
func (a *UserActor) AddConditionScaled(conditionId int, durationMult float64, source string) {
	a.User.AddConditionScaled(conditionId, durationMult, source)
}

func (a *UserActor) AddConditionMagnitude(conditionId int, triggers int, magnitude float64, source string) {
	a.User.AddConditionMagnitude(conditionId, triggers, magnitude, source)
}
```

`actor_mob.go` (read its struct field name first; it wraps the `*mobs.Mob`):

```go
func (a *MobActor) AddConditionScaled(conditionId int, durationMult float64, source string) {
	a.Mob.AddConditionScaled(conditionId, durationMult, source)
}

func (a *MobActor) AddConditionMagnitude(conditionId int, triggers int, magnitude float64, source string) {
	a.Mob.AddConditionMagnitude(conditionId, triggers, magnitude, source)
}
```

- [ ] **Step 4:** tests pass; `go build ./...`.
- [ ] **Step 5: Commit** (`feat(actions): scaled and magnitude condition doors for both actors`). The apply-path guard (`go test . -run TestPlayerConditionsTravelTheEventPath -count=1`) may need the new door call sites allowlisted as EVENT doors, the way `light_spell.go` is; add them with that reason.

---

### Task 2: The mob tick fills a zero tick amount

**Files:** Create `internal/hooks/condition_tick_amount.go`, test `internal/hooks/condition_tick_amount_test.go`. Modify `NewRound_UserRoundTick.go:312-325`, `NewRound_MobRoundTick.go:~220-222`.

- [ ] **Step 1: Failing test.** Build a mob character holding a seeded `tick_pool: health`, `tick_percent: 10` condition added through `Conditions.AddCondition` (so `TickAmount` is 0), with `Health` below `HealthMax`; run `tickMobConditions(mob, id)` for one trigger; assert health rose. Add a `tick_percent: -10` (damage) row asserting health fell. Read `internal/hooks` tests for an existing mob-tick fixture (grep `tickMobConditions(`) and follow it; seed conditions with `conditions.SeedConditionsForTest`.
- [ ] **Step 2:** run it; it fails (health unchanged).
- [ ] **Step 3: Implement** the helper:

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
)

// fillZeroTickAmount computes and caches the per-trigger amount for a
// tick_pool condition whose TickAmount is still 0. A condition applied
// through the event queue is not in the list when its applier would snapshot
// the amount, so it arrives at 0. Shared by the player and mob round ticks
// (drink path unification); before it, only the player tick filled the
// amount and every mob heal-over-time and damage-over-time was inert.
// Scaling is 1.0: the applier's caster scaling is already lost by then (the
// "ticks" parity slice moves the computation to apply time).
func fillZeroTickAmount(c *characters.Character, cond *conditions.Condition, spec *conditions.ConditionSpec) int {
	if spec == nil || spec.TickPool == "" || cond.TickAmount != 0 {
		return cond.TickAmount
	}
	var maxPool int
	switch spec.TickPool {
	case "health":
		maxPool = c.HealthMax.Value
	case "stamina":
		maxPool = c.StaminaMax.Value
	case "conviction":
		maxPool = c.ConvictionMax.Value
	}
	amt := conditions.ComputeTickAmount(maxPool, spec.TickPercent, spec.TickVariance, spec.TickMin, 1.0)
	c.Conditions.SetTickAmount(cond.ConditionId, amt)
	return amt
}
```

Check the concrete type the ticks iterate (`Trigger()` return element type) and the `TickAmount` field type, and adapt the signature to them exactly. Replace the player tick's inline block (P5) with a call; in `tickMobConditions`, call it before the `TickAmount != 0` test and use its return. The trigger loop may iterate copies: make sure the mob branch reads the RETURNED amount, not the stale copy.

- [ ] **Step 4:** the new test passes; `go test ./internal/hooks/ -count=1` passes (the player tick's tests unchanged).
- [ ] **Step 5: Commit** (`fix(ticks): the mob round tick fills a zero tick amount like the player tick`).

---

### Task 3: `actions.Drink` holds the player body

**Files:** `internal/actions/drink.go`; move `internal/usercommands/drink_purge_test.go`, `drink_purge_shipped_test.go`, `drink_scour_test.go`, and `internal/mobcommands/drink_magnitude_test.go` into `internal/actions/` (rename to `drink_*_test.go`, package `actions`).

- [ ] **Step 1: Result type** (in `drink.go`):

```go
// DrinkRefusal says why a drink did not happen.
type DrinkRefusal int

const (
	DrinkOK DrinkRefusal = iota
	DrinkRefuseBusy
	DrinkRefuseGrappled
	DrinkRefuseNotFound
	DrinkRefuseNotDrinkable
	DrinkRefuseToxicity
)

// DrinkResult reports one drink. Drank is true when the item was consumed,
// including a spoiled potion.
type DrinkResult struct {
	Drank   bool
	Spoiled bool
	ItemId  int
	Refusal DrinkRefusal
}
```

- [ ] **Step 2: Move the body.** Copy `usercommands.Drink`'s body (from the busy check to the end), the constants (`bloomWaferItemId`, `ysoldesPurgeItemId`, `purgingDraughtItemId`, the potion-condition block, `catalystOfUnmakingItemId`, `scourRerollCharges`, `phialOfSecondBirthItemId`, `phialRarityFloor`), and `bypassesToxicityGate`, `purgeableConditionIds`, `applyPurgeEffects` into `actions/drink.go` as `func Drink(actor DrinkActor, rest string) DrinkResult`. Apply exactly these substitutions and nothing else:

| Player body | Shared body |
|---|---|
| `refuseWhileBusy(user, "drink")` | `if char.IsActing() { actor.SendText(CategorySystem, <same red line with "drink">); return DrinkResult{Refusal: DrinkRefuseBusy} }` |
| `user.Character` | `char := actor.GetCharacter()` |
| `user.SendText(...)` | `actor.SendText(...)` |
| `user.AddCondition(...)`, `AddConditionScaled`, `AddConditionMagnitude` | the `actor.` methods |
| `room` | `room := actor.GetRoom()`; if nil, `rooms.LoadRoom(char.RoomId)` |
| `user.UserId` as a room exclude id | exclude the player's id for a player; for a mob, send to all (as the mob path does today) |
| `<ansi fg="username">%s</ansi>` in room lines | `username` for `actor.IsPlayer()`, else `mobname` |
| quest notify block | only when `ua, ok := actor.(*UserActor); ok`, using `ua.User` |
| each `return true, nil` | the matching `DrinkResult` (refusals set `Refusal`; success sets `Drank`, `ItemId`; spoiled sets `Spoiled` too) |
| `applyPurgeEffects(u *users.UserRecord)` | `applyPurgeEffects(actor DrinkActor)` |
| em dashes in player strings (P6) | a colon or a new sentence; nothing else about the text changes |

The logic, order, numbers and conditions are unchanged. After moving, `usercommands/drink.go` still holds the old body until Task 4; do not delete it yet.

- [ ] **Step 3: Move the tests.** Move the four test files into `internal/actions/`, switching them to call `Drink(NewUserActorInRoom(u, room), ...)` or `applyPurgeEffects(NewUserActor(u))` and the moved helpers. The magnitude test moves as-is (it calls `items.PotionMagnitudeApplication`). The shipped-world purge test uses `t.Chdir` to the repo root; the actions package is also two levels down, so the same `t.Chdir` works; confirm.
- [ ] **Step 4:** `go test ./internal/actions/ -count=1` passes.
- [ ] **Step 5: Commit** (`refactor(actions): the drink body moves to actions.Drink`).

---

### Task 4: Thin wrappers, the parity table, the re-fork guard

**Files:** `internal/usercommands/drink.go`, `internal/mobcommands/drink.go`, `internal/actions/drink_parity_test.go` (new), `drink_wrapper_guard_test.go` (new, repo root), `condition_apply_path_guard_test.go`.

- [ ] **Step 1: Parity table (failing first).** For each row, build one player (`UserActor` over a `users.UserRecord` with a character) and one mob (`MobActor` over a `mobs.Mob`), each holding the same potion, drink, and assert identical: `DrinkResult` (minus actor ids), toxicity after, item consumed, and the queued condition events (id, `DurationMult`, `Triggers`, `Magnitude`), read with the events drain helpers. Rows: a plain healing potion; the same potion with `CraftedRound`/`CraftSkill` set to reach peak; the Pitsense-shaped magnitude potion; a spoiled potion; one past toxicity tolerance (`DrinkRefuseToxicity`); the Purging Draught (strips a held potion condition, clears toxicity, queues 76); Ysolde's Purge (Bloom addiction down); the Catalyst (scour); the Phial (scour then grant; assert a mutation changed); the Bloom Wafer (addiction up, condition 90 queued). Seed items and conditions with `items.SeedItemsForTest` / `conditions.SeedConditionsForTest`; pin randomness where the body rolls (Wafer, spoiled recipe discovery) by asserting only the deterministic parts. Written against the Task 3 body, rows involving the MOB fail until Step 3 routes the mob wrapper, which is the point.
- [ ] **Step 2: Re-fork guard** at the repo root:

```go
package main

import (
	"os"
	"regexp"
	"testing"
)

// TestDrinkWrappersDoNotReFork: the drink rules live in actions.Drink. If a
// command wrapper applies a condition, toxicity, an item use, or a mutation
// change itself, the player and mob paths have forked again (the mob path
// was a 49-line copy missing toxicity, potency and every special potion
// until 2026-09-28).
func TestDrinkWrappersDoNotReFork(t *testing.T) {
	forbidden := regexp.MustCompile(`AddCondition|AddToxicity|UseItem|ScourMutations|AddBloomAddiction|GrantRandomMutation|BloomSeedNewMutation|BloomAdvanceMutation|SetTickAmount`)
	for _, path := range []string{"internal/usercommands/drink.go", "internal/mobcommands/drink.go"} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if loc := forbidden.FindIndex(b); loc != nil {
			t.Errorf("%s applies drink rules itself (%q); call actions.Drink instead", path, b[loc[0]:loc[1]])
		}
	}
}
```

- [ ] **Step 3: Wrappers.**

```go
// internal/usercommands/drink.go
// Drink is the player command; the rules live in actions.Drink (drink path
// unification), shared with mobs and the AI companion.
func Drink(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	actions.Drink(actions.NewUserActorInRoom(user, room).(actions.DrinkActor), rest)
	return true, nil
}
```

```go
// internal/mobcommands/drink.go
// Drink is the mob command; the rules live in actions.Drink, the same body a
// player drinks through.
func Drink(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actions.Drink(actions.NewMobActorInRoom(mob, room).(actions.DrinkActor), rest)
	return true, nil
}
```

If the constructors return the concrete pointer types, drop the assertion. Delete everything else from both files, including the moved helpers and constants, and the now-orphaned test files left behind in `usercommands`/`mobcommands`.

- [ ] **Step 4: Guard keys.** Run `go test . -run TestPlayerConditionsTravelTheEventPath -count=1`; re-key the drink entries to their new `internal/actions/drink.go|<line>` sites with the same EVENT-door reasons; remove keys for deleted sites.
- [ ] **Step 5: Prove the re-fork guard can fail** by temporarily adding `mob.AddCondition(1, "x")` to the mob wrapper; see it fail; revert.
- [ ] **Step 6:** `go build ./... && go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ . -count=1` passes.
- [ ] **Step 7: Commit** (`refactor(drink): players and mobs drink through actions.Drink; guard the wrappers`).

---

### Task 5: Gate

- [ ] `gofmt -l internal/ modules/ *.go`; `go vet ./...`; `go build ./...`; `go test ./... -count=1` (background, 10 min); `~/go/bin/golangci-lint run --new-from-merge-base=origin/master`. Any golden that moves: filtered diff proving only intended rows. Boot check per `dogmud-shipping` (detached worktree at `C:/tmp/dogmud-boot-check`, `CONFIG_PATH` port overrides `TelnetPort [33334]`, `LocalPort 9998`, `HttpPort 8091`, `HttpsPort 0`, `AIPort 0`, kill by PID).

---

### Task 6: Docs, playtest, PR

- [ ] **Docs:** `internal/actions/context.md` (new `Drink`, `DrinkActor`, `DrinkResult`), `internal/usercommands/context.md` and `internal/mobcommands/context.md` (drink is a wrapper), `internal/hooks/context.md` (`fillZeroTickAmount`, the mob tick now heals), `internal/mobs/context.md` (`AddConditionScaled`). Verify every symbol exists; run `python tools/context_md_audit.py`. `docs/PATCH_NOTES.md` entry (80 columns, no numbers, no dashes): companions and other creatures now drink by the same rules as players (toxicity, freshness, the special potions), and healing that works over time now heals creatures too. `docs/README.md`: this plan's row.
- [ ] **Playtest** (`dogmud-playtesting`, one actor, 10m): hire or summon an AI companion if the environment allows, otherwise use a charmed pet; hand it healing potions; confirm it heals over time and eventually refuses on toxicity. If no companion is reachable in the harness, state it and rely on the parity table.
- [ ] **PR:** `gh pr create --repo pruuk/DOGMud --base master`, body lists the mob-side behaviour changes and the audit follow-on slices.
