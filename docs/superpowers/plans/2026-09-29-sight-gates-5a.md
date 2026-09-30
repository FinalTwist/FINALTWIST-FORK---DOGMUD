# Sight and Gates Parity (Slice 5a) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A mob gets, looks, removes, equips and crafts by the player's gates, through shared bodies in `internal/actions` and `internal/characters`, and one slot-choice helper decides every ring, wrist, weapon and shield placement for `Wear`, `WearInArm` and the upgrade scorer alike.

**Architecture:** Each gate becomes one exported predicate or shared body (`actions.TooDarkToGet`, `TakeFloorItem`, `ResolveLook`, `CursedHolds`, `RemoveAllEquipment`, `TooDarkToCraft`); the command wrappers keep only wording. The equip curse rule lives beside `Wear` as `Character.CursedRefusal`, checked after placement for every slot. `Character.ChooseWornSlot` builds an ordered candidate list over the arms and wrists the character actually has (up to 6 arms) and applies one rule: fill the first empty candidate, else swap the first uncursed one, else refuse. `Wear` becomes `wear(i, place)` so `WearInArm` (the player's `equip X armN`) shares every gate. `itemvalue` asks the same helper which slot and which displaced items to score.

**Tech Stack:** Go 1.25, `internal/actions` Actor seam, `internal/characters` worn slots, `internal/itemvalue`, repo-root guard tests.

**Spec (binding):** `docs/superpowers/specs/2026-09-29-sight-and-gates-parity-design.md`, the 5a section and owner rulings 1 to 14. 5b (speech and emotes) is a separate plan and PR.

**Branch:** `feature/sight-gates-5a`, cut from `master` AFTER the spec PR (`docs/sight-gates-spec`) merges. Work in a worktree:

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
git fetch origin
git worktree add -b feature/sight-gates-5a C:/tmp/dogmud-sight-5a origin/master
```

All paths below are relative to that worktree. Tests load Go config defaults, never `_datafiles/config.yaml`; this slice reads no balance knob (the Spellcasting 4 threshold is a literal in today's `remove.go:66` and stays one).

---

## Facts verified against source (2026-09-29, HEAD `3152749b0`)

Every row re-read today at `3152749b0` (branch `docs/sight-gates-spec`, whose changes since the spec's `7d6d4ac38` are docs only). Negative rows name the search and a positive hit that proves it could match. Config numbers from `git show HEAD:_datafiles/config.yaml`.

| # | Fact | Where |
|---|---|---|
| F1 | `Character.Wear(i items.Item) (returnItems []items.Item, newItemWorn bool, failureReason string)` at `:586-672`: `i.Validate()`, type gate `:592`, MinStrength `:599`, `HandsRequired > 2` `:603-606`, `beforeReserve`/`savedEquipment` `:627-628`, placement `:632-636`, reservation revert `:642-650`, ComponentBag sort, armour-path `reapplyPermanentConditions` `:655-660`, light reset `:661-670` | `internal/characters/worn.go` |
| F2 | `wearWeaponOrShield(i, spec, iHandsRequired int, canDualWield bool)` at `:373-465`: 2H `:377-403` (cursed checks `:385-390`), shield `:405-422` ("no room" `:412-414`, cursed `:415-417`), 1H fill with dual-wield detour `:424-451`, fallback over `Weapon` `:453-464` (cursed `:454-456`, unchecked stray `:457-460`). Only caller: `Wear` `:633` | `worn.go`; grep `wearWeaponOrShield` |
| F3 | `wearArmorSlot` Ring case `:504-515`, Wrist case `:516-535`; neither checks a curse; the fallback writes `Ring`/`Wrist1` even when disabled | `worn.go:472-584` |
| F4 | `HandsRequired` dereferences `species.GetSpecies(c.SpeciesId)` with no nil guard (`:292-299`); `GetSpecies` is a bare map read that returns nil for an unregistered id. `WeaponHands` is `= int` | `worn.go:278-302`; `internal/species/species.go:76-78`; `internal/items/itemspec.go:30` |
| F5 | `GetHandPairs`, `HandSlot{Label, ItemPtr}`, labels `wielded`, `offhand`, `extra arm 1` to `extra arm 4`; `Is2H`, `IsEmpty` (nil, `ItemId < 1` or disabled), `IsHalfPair`. `FindFirstEmptySlot`, `FindFirstFreePair`, `FindCheapestPairToDisplace`, `PairIsFree`, `PairOccupantCount` have no callers outside `hand_slots.go`/`worn.go` (grep; the same grep finds `IsHalfPair` at `usercommands/equip.go:146,159`) | `internal/characters/hand_slots.go:7-149` |
| F6 | `validateMutationSlots` sets `ExtraArms` from `Mutations["extra-arms"]` capped at 4 and disables `ExtraArmN`/`ExtraWristN` above it; called only from `Validate()` (`:716`). `Validate(true)` differs from `Validate()` only by `reapplyPermanentConditions` | `internal/characters/validate.go:513-566,591,716-720` |
| F7 | `CanDualWield()` is `GetSkillLevel(skills.WeaponCombat) > 0` | `validate.go:305-308` |
| F8 | `AllSlots()` keys: `weapon`, `offhand`, `extraarm1`..`extraarm4`, ..., `wrist1`, `wrist2`, `extrawrist1`..`extrawrist4`, `ring`, `ring2`, `light` | `worn.go:53-69` |
| F9 | `IsCursed()` is `GetSpec().Cursed && !i.Uncursed` (pointer receiver); `ItemDisabledSlot = Item{ItemId: -1}`; `IsDisabled()` value receiver; `HasAdjective` pointer receiver; `ItemSpec.Cursed` | `internal/items/items.go:25,205,124,398-400`; `itemspec.go:347` |
| F10 | `usercommands.Equip`: `refuseWhileBusy` `:52`, arm-suffix parse `:65-86`, fashionable `:105-111`, `beforeReservation` `:119`, arm branch `:122-273` (the four cursed checks `:172-196` with "is cursed and can't be removed!", `CancelConditionsWithFlag(Hidden)` `:212`, `StoreItem(old)` ignoring its result `:228`, `Validate(true)` `:247`), ordinary path `:276-342` | `internal/usercommands/equip.go` |
| F11 | `actions.EquipItem(actor, itemName) EquipItemResult{Item, DisplacedItems, Found, Equipped, FailureReason}` calls `char.Wear` at `:44`, stores displaced or drops to the floor `:61-67`, `Validate()` `:69`. `RemoveEquipment(actor, itemName) RemoveEquipResult{Item, Found, Removed, Err}` at `:99-131` | `internal/actions/remove_equip.go` |
| F12 | Callers: `EquipItem` at `usercommands/equip.go:277`, `mobcommands/equip.go:65`, `hooks/mob_equip_best_floor_item.go:60`; direct `Wear` at `actions/sell.go:404`, `bountyhunter/bountyhunter.go:91`, `rooms/rooms.go:1001`, `modules/aicompanion/cooking.go:159`; `RemoveEquipment` at `usercommands/remove.go:33,79`, `mobcommands/remove.go:27,42`; `InitiateCraft` at `usercommands/craft.go:124`, `mobcommands/craft.go:26` | grep |
| F13 | `usercommands.Remove`: `refuseWhileBusy` `:17`, `all` loop `:30-54` with aggregate `EquipmentChange` `:39-42`, single `FindOnBody` `:57`, cursed `:65-77` (`Spellcasting` `< 4`), `RemoveEquipment` `:79`. `refuseWhileBusy` text `<ansi fg="red">You can't %s while focused on your work. Finish or be interrupted first.</ansi>` | `internal/usercommands/remove.go`; `busy_refuse.go:18-25` |
| F14 | `busy_refuse_test.go:49` pins `Remove("sword", ...)` to the busy line for a user with NOTHING worn, so the busy gate must run before the body lookup | `internal/usercommands/busy_refuse_test.go:39-75` |
| F15 | `mobcommands.Remove`: `PermaGear` emote `:17-20`, `all` loop `:24-39` with aggregate event, single `:42-47` | `internal/mobcommands/remove.go` |
| F16 | `IsActing()` is nil-safe (`Activity == nil` is false). `users.NewTestUser` builds no `Activity` and no `Perception`; `characters.New()` builds both | `internal/characters/spells.go:119-124`; `internal/users/test_helpers.go:56-80`; `character.go:406-425` |
| F17 | `actions.GetItemFromFloor` `:27-53` (bauble gate `:38-40`, `TransferItemToBackpack` `:43-50`); `GetGoldFromFloor` `:57-59`. `TransferItemToBackpack` removes from the source first, rolls back with `AddItem` on a full pack, fires `ItemOwnership{UserId, MobInstanceId, Item, Gained}` | `internal/actions/get.go`; `transfer.go:44-64` |
| F18 | `usercommands.Get`: sight gate `:94-97`; sweep `getAllMatchingFromFloor` `:30-87` (exploding `:52`, `CancelConditionsWithFlag` `:56`, `StoreItem`+`RemoveItem`+`ItemOwnership` `:58-65`); single-get peek `:634-640`, `GetItemFromFloor` `:642`, bauble `:646`, stash auto-detect `:668-692`; gold `:600-627`; `takeableOnFloor` `:821` | `internal/usercommands/get.go` |
| F19 | `mobcommands.Get` `:16-95`: gold path cancels `Hidden` BEFORE the pickup `:47` | `internal/mobcommands/get.go` |
| F20 | `usercommands.Look`: `ParticipantSight` `:33-37`, `events.Looking` `:55-60`, empty `:63-77`, creature `:88-156`, crate `:159`, container `:166-249`, exit `:254-305` (`SeesThroughExit` `:285`, lock `:291`), direction alias no-exit `:309-312`, item noun `:319`, carried item `:331-383`, room noun `:388-436`, pet `:441-458`, corpse `:460-527`, floor item `:540-572`, shapes hint `:576-579`. **Drift:** the corpse and floor-item branches are not in the spec's L4 list | `internal/usercommands/look.go` |
| F21 | `mobcommands.Look` `:14-170`: empty, exit (lock only), backpack, `ResolveTargetActor(room, lookAt)` `:92` (no viewer), body, noun, pet; room lines via plain `SendTextVisual`; `lookRoom` `:172-199` | `internal/mobcommands/look.go` |
| F22 | `ResolveTargetOptions{Viewer *characters.Character}`; `Room.SendTextVisualHidingNames(cat, txt string, names []string, excludeUserIds ...int)` | `internal/actions/target_resolution.go:23-27`; `internal/rooms/rooms.go:276` |
| F23 | `messaging.ParticipantSight(observer, room) SightDecision` (Blinded perception is `SightNone` first); `SeesThroughExit` `:96`; `CanSeeClearly` = awake and `SightFull` `:136-138` | `internal/messaging/predicates.go:56-62` |
| F24 | `actions.InitiateCraft(actor, recipeName) CraftResult` `:131`, first gate `IsCrafting` `:136`; `CraftResult` fields `Initiated`, `ImmediateComplete`, `RecipeNotFound`, `RecipeNotKnown`, `SkillTooLow`, `WrongStation`, `MissingIngredients`, `ForeignComponent`, `AlreadyCrafting`, `AmbiguousRecipes` | `internal/actions/craft.go:17-45,131-138` |
| F25 | `usercommands.Craft` refuses at `!messaging.CanSeeClearly` `:92-95` ("You can't see well enough to work on anything here."), after `list`/`all` `:79-81`, before storage and enchanting | `internal/usercommands/craft.go:78-124` |
| F26 | `mobcommands.Craft` `:19-70`, silent on every refusal | `internal/mobcommands/craft.go` |
| F27 | Companion: `get` verb refuses a household bauble up front `:365-368`; `equip` `:402-406`; `remove` `:408-412`; judged by worn count `:791-799`; craft not-ok line `:871-877`; `craftableHere(mob, p, room)` `:58-88` has no sight test; `cannotSee(mob, room)` is `sightOf != SightFull` | `modules/aicompanion/actions.go`; `cooking.go`; `perception.go:314-316` |
| F28 | `itemvalue`: `compatibleSlotsFor(spec, char)` `:57-113` (reads `spec.Hands`, `extraArmsLevel` from mutations `:40-46`, only caller `:80`); `displacedItemsForSlot(char, targetSlot, candidateSpec)` `:231-255`; `ItemValueDelta` `:287-340`; `IsUpgrade` `score.go:47-49`. Callers: `mobcommands/gearup.go:34,79`, `mobs/crafter.go:445`, `hooks/mob_equip_best_floor_item.go:45`, `planners/shop_upgrade.go:58` | `internal/itemvalue/delta.go`; grep |
| F29 | `itemvalue` tests call `compatibleSlotsFor(spec, ...)` and `displacedItemsForSlot(char, slot, spec)` directly on bare `&characters.Character{Mutations: ...}` fixtures (`delta_test.go:25-145`); the wrist tests set `Mutations["extra-arms"]`, not `ExtraArms` | `internal/itemvalue/delta_test.go` |
| F30 | Lookup guard keys `internal/mobcommands/look.go|Look` `{plain: 1, why: whyMob}` (`:49`) and `internal/usercommands/look.go|Look` `{viewer: 1}` (`:71`); fails on stale | `lookup_viewer_guard_test.go` |
| F31 | Narration guard keys `file|first literal` (80 runes): `usercommands/equip.go|You equip your ... in your %s.` (`:1347`), `...|You remove your ... backpack.` (`:1348`), `...|You wear your` (`:1349`), `...|You wield your ... dangerous.` (`:1350`); get.go `:1351-1360`; look.go `:1376-1381`; `usercommands/remove.go|You remove your ...` (`:1392`); no `mobcommands/(get,look,remove,equip,craft).go` or `actions/(get,remove_equip,craft).go` keys | `messaging_surface_guard_test.go:752-765,1106-1124,1347-1392` |
| F32 | `bauble_finder_view_guard_test.go:76` counts 4 viewer-accessor calls in `internal/usercommands/look.go|Look`; the walk must read `usercommands/look.go` and `get.go` (`:393`) | root |
| F33 | `bauble_sweep_guard_test.go` `transientItemHolders` must list every struct that names `items.Item` directly (it lists `actions.EquipItemResult`, `GetItemResult`, `RemoveEquipResult`, `characters.HandSlot`, `WornSlot`). **Not in the spec's T rows** | `bauble_sweep_guard_test.go:36-58` |
| F34 | Re-fork guard templates: package `main`, `os.ReadFile`, one regexp per wrapper pair | `drink_wrapper_guard_test.go`; `flee_wrapper_guard_test.go` |
| F35 | Parity-table style: `internal/actions/drink_parity_test.go` builds a player (`users.UserRecord`) and a mob (`mobs.Mob{Character: *newParityChar()}`) alike and drives both through `NewUserActorInRoom`/`NewMobActorInRoom`. Its constants `parityUserId` etc. are package-level, so new names must differ | root of `internal/actions` |
| F36 | Sight fixtures already in use: `room.SkyLight, room.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(0)` for pitch dark (`stolen_bauble_test.go:334`); infrared condition with `EffectInfraReach: {Literal: 30}` for shapes (`cast_sight_test.go:62-68`); `Perception.TransitionTo(perception.Blinded, ...)` (`stolen_bauble_test.go:329`); `viewerTestHide(t, c)` (`target_viewer_test.go:15`); `newTestRoom()` is `Lamp 90` (`transfer_test.go:83`) | `internal/actions/*_test.go` |
| F37 | `events.DrainQueuedInputsForTest(instanceId) []string` reads what `mob.Command` queued; `DrainQueuedMessagesForTest(userId)` | `internal/events/events.go:314,361` |
| F38 | `crafting.RegisterRecipeForTest(*RecipeSpec)` / `UnregisterRecipeForTest(id)`; companion fixtures `harmWorld`, `strangerModule`, `performAction(c, mob, owner, sc, ActionProposal, stims, delay, round)` | `internal/crafting/crafting.go:613-626`; `modules/aicompanion/harm_test.go:20-51,331-345`; `aicompanion_test.go:2328` |
| F39 | Config (HEAD blob): `DataFiles: _datafiles/world/dogmud` (`:240`), `MutationMaxLevel: 4` (`:1807`); the live world ships no cursed item (spec E10) | `git show HEAD:_datafiles/config.yaml` |
| F40 | New names are free: grep for `ErrTooDark`, `ErrExploding`, `TooDarkToGet`, `TooDarkToCraft`, `CursedHolds`, `ResolveLook`, `ChooseWornSlot`, `CursedRefusal`, `WearInArm`, `EquipItemInArm`, `RemoveAllEquipment`, `SlotChoice`, `busyRefusalText` over `internal`/`modules` returns nothing (the same grep finds `RemoveEquipment`), and `internal/actions/look.go` does not exist | grep |

## Design decisions made while planning

1. **The golden test is a frozen oracle, not a data file.** Task 1 copies today's `wearWeaponOrShield`, the three `Find*` helpers and the ring and wrist cases into a test file verbatim, and diffs today's `Wear` against it over every non-cursed loadout at 2, 3, 4 and 6 arms (about 27,000 placements: 1,140 hand layouts, three species sizes, dual wield on and off, four candidates; plus every ring and wrist layout). It passes before any change, then pins the refactor. Two cases are carved out by a predicate because the owner changed them on purpose with nothing cursed: the legacy fallback that writes a disabled `Ring` or `Wrist1` (it hands back the `ItemId -1` marker), and ruling 13's shield beside a two-hander at 3 or more arms. Both get explicit tests.
2. **`ChooseWornSlot` does not mutate.** It returns pointers into `c.Equipment` wrapped as `WornSlot`s; `Wear` applies them. The scorer calls it freely.
3. **`HandsRequired` treats an unregistered species as Medium** (nil guard). The scorer now reaches it for every weapon candidate (F4), and several test fixtures build characters with no species registry; a nil species used to be a panic, never a behaviour.
4. **`ChooseWornSlot` refuses `HandsRequired > 2`** for hands, so the scorer never offers a slot `Wear` will refuse ("That requires too many hands."). `wear()` keeps its own check first, so the order players see is unchanged.
5. **`ResolveLook` keeps the player's order exactly.** The player checks a sealed crate and a room container between the creature and the exit, and the pet after carried items and room nouns. So `ResolveLook` returns `LookOther` for a crate or known container name before it tries exits, and reports a nameable pet as `PetUserId` on `LookOther` instead of a `LookPet` kind (a kind would move the pet ahead of items and nouns). See "Spec points that could not be implemented as written".
6. **`RemoveEquipment` checks busy before the body lookup** (F14), and `RemoveAllEquipment` removes each worn item by identity through a shared private `removeWorn`, not by name, so two same-named rings (one cursed) cannot make the loop test the cursed one twice.
7. **The shield candidate list follows the spec literally.** With no two-hander in `Weapon`, the list is `Offhand`, then the extra arms in order with a 2H pair offering its First (displacing the two-hander). With a two-hander in `Weapon` (ruling 13), the swap walks extra arms from the highest down and skips any pair whose First holds a two-hander.
8. **New item-holding structs are registered with the bauble sweep guard** (F33) in the commit that adds them: `characters.SlotChoice`, `characters.slotCandidate`, `actions.RemoveAllResult`.
9. **The parity harness is one file**, `internal/actions/sight_gates_parity_test.go`, created in Task 5 and extended by Tasks 7 to 10, so every row of the spec's 5a parity table that lives in `actions` runs through a `UserActor` and a `MobActor` in the same scene at four lights (lit, shapes, dark, blinded) whose verdicts the fixture asserts before any row runs.

## Spec points that could not be implemented as written

- **`LookPet` kind** (spec "Look"): the player resolves the pet AFTER carried items and room nouns (F20), so a kind returned from `ResolveLook` would reorder the player's look. Replaced by `LookResolution.PetUserId` (set only at `SightFull`), read by each wrapper at its own pet step.
- **"creature before exit" as the whole order**: the player also resolves a sealed crate and a room container between those two (F20). `ResolveLook` defers those names with `LookOther`.
- **"With nothing cursed the choice is today's, exactly"**: rulings 10 and 13 themselves change two nothing-cursed placements (a disabled `Ring`/`Wrist1` fallback, and the shield beside a two-hander). The golden oracle carves out exactly those two and explicit tests pin the new behaviour.
- **"`FindFirstEmptySlot`, `FindFirstFreePair` and `FindCheapestPairToDisplace` ... become the helper's internals"**: the helper does not need them (a stable sort by occupant count is `FindFirstFreePair` then `FindCheapestPairToDisplace`); they are deleted, with `PairIsFree` and `PairOccupantCount`, which only they called (F5).

## File map

| File | Change |
|---|---|
| `internal/characters/wear_slot_golden_test.go` | Create (Task 1): frozen oracle and the golden diff |
| `internal/characters/wear_slot.go` | Create (Task 2): `SlotChoice`, `CursedRefusal`, `ChooseWornSlot`, candidates, `apply` |
| `internal/characters/wear_slot_test.go` | Create (Task 2, 3, 4): helper rows, hand table across arm counts, arm-N rows |
| `internal/characters/wear_curse_test.go` | Create (Task 3): `Wear` curse rows per slot family |
| `internal/characters/worn.go` | Modify (Tasks 2 to 4): `HandsRequired` nil guard, `wear`, `Wear`, `WearInArm`, `wearChosen`, Ring/Wrist cases; delete `wearWeaponOrShield` |
| `internal/characters/hand_slots.go` | Modify (Tasks 3, 4): delete the three `Find*` and two `Pair*` helpers; add `ArmLabel` |
| `internal/actions/remove_equip.go` | Modify (Tasks 5, 9): `equipItem`, `EquipItemInArm`, `ArmLabel`; `CursedHolds`, busy and curse gates, `removeWorn`, `RemoveAllEquipment` |
| `internal/actions/sight_gates_parity_test.go` | Create (Task 5), extend (Tasks 7 to 10) |
| `internal/usercommands/equip.go` | Modify (Task 5): arm branch through `EquipItemInArm` |
| `internal/usercommands/equip_arm_test.go` | Create (Task 5) |
| `internal/itemvalue/delta.go` | Modify (Task 6): scorer through the helper |
| `internal/itemvalue/delta_test.go` | Modify (Task 6) |
| `internal/itemvalue/slot_agreement_test.go` | Create (Task 6) |
| `internal/mobcommands/gearup_curse_test.go` | Create (Task 6) |
| `internal/actions/get.go` | Modify (Task 7) |
| `internal/usercommands/get.go`, `internal/mobcommands/get.go` | Modify (Task 7) |
| `internal/usercommands/get_sight_gates_test.go`, `internal/mobcommands/get_sight_gates_test.go` | Create (Task 7) |
| `internal/actions/look.go` | Create (Task 8) |
| `internal/usercommands/look.go`, `internal/mobcommands/look.go` | Modify (Task 8) |
| `internal/mobcommands/look_sight_gates_test.go` | Create (Task 8) |
| `internal/usercommands/busy_refuse.go`, `remove.go`, `internal/mobcommands/remove.go` | Modify (Task 9) |
| `internal/usercommands/remove_curse_test.go`, `internal/mobcommands/remove_gates_test.go` | Create (Task 9) |
| `modules/aicompanion/actions.go`, `cooking.go` | Modify (Tasks 9, 10) |
| `modules/aicompanion/sight_gates_test.go` | Create (Task 9, extended in 10) |
| `internal/actions/craft.go`, `internal/usercommands/craft.go`, `internal/mobcommands/craft.go` | Modify (Task 10) |
| `sight_gates_wrapper_guard_test.go` | Create (Task 11) |
| `lookup_viewer_guard_test.go`, `bauble_sweep_guard_test.go`, `messaging_surface_guard_test.go` | Re-key in the task that moves the code (5, 2, 8, 9 as noted) |
| `internal/{actions,characters,itemvalue,mobcommands,usercommands}/context.md`, `modules/aicompanion/context.md`, `docs/PATCH_NOTES.md` | Modify (Task 12) |

---

### Task 1: The golden oracle pins today's placement

**Model:** sonnet.

**Files:**
- Create: `internal/characters/wear_slot_golden_test.go`

Nothing in production changes in this task. It must land (committed) before Task 3 touches `Wear`.

- [ ] **Step 1: Write the oracle and the golden diff**

Create `internal/characters/wear_slot_golden_test.go`:

```go
package characters

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
)

// The golden oracle for slot choice (sight and gates slice 5a, spec
// "Slot choice"). The legacy* functions below are VERBATIM copies of the
// placement code as it stood at 3152749b0: wearWeaponOrShield, the three
// Find* helpers, and wearArmorSlot's Ring and Wrist cases. The tests diff
// Character.Wear against them over every non-cursed loadout at 2, 3, 4 and 6
// arms, so a refactor of the slot choice provably moves nothing when nothing
// is cursed. Do not "fix" the legacy copies: they are the record.

func legacyFindFirstEmptySlot(c *Character, pairs []HandPair, isShield bool) *HandSlot {
	for pi := range pairs {
		p := &pairs[pi]
		if pi == 0 && isShield {
			if !p.IsHalfPair() && p.Second.IsEmpty() && !p.First.Is2H(c) {
				return &p.Second
			}
			continue
		}
		if p.First.Is2H(c) {
			continue
		}
		if p.First.IsEmpty() {
			return &p.First
		}
		if !p.IsHalfPair() && p.Second.IsEmpty() {
			return &p.Second
		}
	}
	return nil
}

func legacyFindFirstFreePair(pairs []HandPair) *HandPair {
	for i := range pairs {
		p := pairs[i]
		if !p.IsHalfPair() && p.First.IsEmpty() && p.Second.IsEmpty() {
			return &pairs[i]
		}
	}
	return nil
}

func legacyPairOccupantCount(p HandPair) int {
	count := 0
	if !p.First.IsEmpty() {
		count++
	}
	if !p.IsHalfPair() && !p.Second.IsEmpty() {
		count++
	}
	return count
}

func legacyFindCheapestPairToDisplace(pairs []HandPair) *HandPair {
	var best *HandPair
	bestCount := 3
	for i := range pairs {
		if pairs[i].IsHalfPair() {
			continue
		}
		count := legacyPairOccupantCount(pairs[i])
		if count < bestCount {
			bestCount = count
			best = &pairs[i]
		}
	}
	return best
}

func legacyWearWeaponOrShield(c *Character, i items.Item, spec items.ItemSpec, iHandsRequired int, canDualWield bool) (returnItems []items.Item, newItemWorn bool, failureReason string) {
	pairs := c.GetHandPairs()
	isShield := spec.Type == items.Offhand

	if iHandsRequired >= 2 {
		freePair := legacyFindFirstFreePair(pairs)
		if freePair == nil {
			freePair = legacyFindCheapestPairToDisplace(pairs)
		}
		if freePair == nil {
			return returnItems, false, `You have no free pair of hands for a two-handed weapon.`
		}
		if !freePair.First.IsEmpty() && freePair.First.ItemPtr.IsCursed() {
			return returnItems, false, `Your ` + freePair.First.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
		}
		if !freePair.Second.IsEmpty() && freePair.Second.ItemPtr.IsCursed() {
			return returnItems, false, `Your ` + freePair.Second.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
		}
		if !freePair.First.IsEmpty() {
			returnItems = append(returnItems, *freePair.First.ItemPtr)
		}
		if !freePair.IsHalfPair() && !freePair.Second.IsEmpty() {
			returnItems = append(returnItems, *freePair.Second.ItemPtr)
		}
		*freePair.First.ItemPtr = i
		if !freePair.IsHalfPair() {
			*freePair.Second.ItemPtr = items.Item{}
		}
		c.reapplyPermanentConditions()
		return returnItems, true, ``
	}

	if isShield {
		slot := legacyFindFirstEmptySlot(c, pairs, true)
		if slot != nil {
			*slot.ItemPtr = i
			c.reapplyPermanentConditions()
			return returnItems, true, ``
		}
		if pairs[0].First.Is2H(c) {
			return returnItems, false, `Your two-handed weapon leaves no room for a shield.`
		}
		if pairs[0].Second.ItemPtr.IsCursed() {
			return returnItems, false, `Your ` + pairs[0].Second.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
		}
		returnItems = append(returnItems, *pairs[0].Second.ItemPtr)
		*pairs[0].Second.ItemPtr = i
		c.reapplyPermanentConditions()
		return returnItems, true, ``
	}

	bothMartial := spec.Subtype == items.Claws && c.Equipment.Weapon.GetSpec().Subtype == items.Claws

	slot := legacyFindFirstEmptySlot(c, pairs, false)
	if slot != nil {
		if slot.Label == "offhand" && !canDualWield && !bothMartial {
			slot = nil
			for pi := 1; pi < len(pairs); pi++ {
				p := &pairs[pi]
				if p.First.Is2H(c) {
					continue
				}
				if p.First.IsEmpty() {
					slot = &p.First
					break
				}
				if !p.IsHalfPair() && p.Second.IsEmpty() {
					slot = &p.Second
					break
				}
			}
		}
		if slot != nil {
			*slot.ItemPtr = i
			c.reapplyPermanentConditions()
			return returnItems, true, ``
		}
	}

	if c.Equipment.Weapon.IsCursed() {
		return returnItems, false, `Your ` + c.Equipment.Weapon.DisplayName() + ` is cursed and prevents you from removing it.`
	}
	if pairs[0].First.Is2H(c) && !pairs[0].Second.IsEmpty() {
		returnItems = append(returnItems, *pairs[0].Second.ItemPtr)
		*pairs[0].Second.ItemPtr = items.Item{}
	}
	returnItems = append(returnItems, c.Equipment.Weapon)
	c.Equipment.Weapon = i
	c.reapplyPermanentConditions()
	return returnItems, true, ``
}

func legacyWearRingOrWrist(c *Character, i items.Item, spec items.ItemSpec) (returnItems []items.Item, newItemWorn bool, failureReason string) {
	switch spec.Type {
	case items.Ring:
		if c.Equipment.Ring.IsDisabled() && c.Equipment.Ring2.IsDisabled() {
			return returnItems, false, `You can't wear rings.`
		}
		if !c.Equipment.Ring.IsDisabled() && c.Equipment.Ring.ItemId == 0 {
			c.Equipment.Ring = i
		} else if !c.Equipment.Ring2.IsDisabled() && c.Equipment.Ring2.ItemId == 0 {
			c.Equipment.Ring2 = i
		} else {
			returnItems = append(returnItems, c.Equipment.Ring)
			c.Equipment.Ring = i
		}
	case items.Wrist:
		if c.Equipment.Wrist1.IsDisabled() && c.Equipment.Wrist2.IsDisabled() {
			return returnItems, false, `You can't wear things on your wrists.`
		}
		if !c.Equipment.Wrist1.IsDisabled() && c.Equipment.Wrist1.ItemId == 0 {
			c.Equipment.Wrist1 = i
		} else if !c.Equipment.Wrist2.IsDisabled() && c.Equipment.Wrist2.ItemId == 0 {
			c.Equipment.Wrist2 = i
		} else if c.ExtraArms >= 1 && !c.Equipment.ExtraWrist1.IsDisabled() && c.Equipment.ExtraWrist1.ItemId == 0 {
			c.Equipment.ExtraWrist1 = i
		} else if c.ExtraArms >= 2 && !c.Equipment.ExtraWrist2.IsDisabled() && c.Equipment.ExtraWrist2.ItemId == 0 {
			c.Equipment.ExtraWrist2 = i
		} else if c.ExtraArms >= 3 && !c.Equipment.ExtraWrist3.IsDisabled() && c.Equipment.ExtraWrist3.ItemId == 0 {
			c.Equipment.ExtraWrist3 = i
		} else if c.ExtraArms >= 4 && !c.Equipment.ExtraWrist4.IsDisabled() && c.Equipment.ExtraWrist4.ItemId == 0 {
			c.Equipment.ExtraWrist4 = i
		} else {
			returnItems = append(returnItems, c.Equipment.Wrist1)
			c.Equipment.Wrist1 = i
		}
	}
	return returnItems, true, ``
}

// legacyWear is Wear's preamble (the gates before placement that these
// fixtures can reach) in front of the legacy placement.
func legacyWear(c *Character, i items.Item) ([]items.Item, bool, string) {
	i.Validate()
	spec := i.GetSpec()
	hands := c.HandsRequired(i)
	if hands > 2 {
		return nil, false, `That requires too many hands.`
	}
	if spec.Type == items.Weapon || spec.Type == items.Offhand {
		return legacyWearWeaponOrShield(c, i, spec, hands, c.CanDualWield())
	}
	return legacyWearRingOrWrist(c, i, spec)
}

// ---- fixtures, shared with wear_slot_test.go and wear_curse_test.go ----

const (
	goldenMedium = 0
	goldenSmall  = 1
	goldenLarge  = 2
)

func seedGoldenSpecies(t *testing.T) {
	t.Helper()
	t.Cleanup(species.SeedSpeciesForTest(map[int]*species.Species{
		goldenMedium: {SpeciesId: goldenMedium, Name: "human", Size: species.Medium},
		goldenSmall:  {SpeciesId: goldenSmall, Name: "halfling", Size: species.Small},
		goldenLarge:  {SpeciesId: goldenLarge, Name: "ogre", Size: species.Large},
	}))
}

var (
	goldenSword  = items.ItemSpec{Name: "sword", Type: items.Weapon, Subtype: items.Slashing, Hands: items.OneHanded}
	goldenClaws  = items.ItemSpec{Name: "claws", Type: items.Weapon, Subtype: items.Claws, Hands: items.OneHanded}
	goldenGreat  = items.ItemSpec{Name: "greatsword", Type: items.Weapon, Subtype: items.Slashing, Hands: items.TwoHanded}
	goldenShield = items.ItemSpec{Name: "buckler", Type: items.Offhand, Subtype: items.Wearable, Hands: items.OneHanded}
	goldenRing   = items.ItemSpec{Name: "ring", Type: items.Ring, Subtype: items.Wearable}
	goldenBracer = items.ItemSpec{Name: "bracer", Type: items.Wrist, Subtype: items.Wearable}
)

// goldenItem is a distinct instance: every placed item gets its own id, so a
// comparison can tell two swords apart.
func goldenItem(id int, spec items.ItemSpec, cursed bool) items.Item {
	s := spec
	s.ItemId = id
	s.Cursed = cursed
	return items.Item{ItemId: id, Spec: &s}
}

// goldenChar is a fresh character with extraArms arms beyond two (the
// mutation slots enabled exactly as Validate enables them), a species and
// optionally dual wield.
func goldenChar(extraArms, speciesId int, dual bool) *Character {
	c := New()
	c.SpeciesId = speciesId
	c.Mutations = map[string]int{}
	if extraArms > 0 {
		c.Mutations["extra-arms"] = extraArms
	}
	c.validateMutationSlots()
	if dual {
		c.SetSkill(string(skills.WeaponCombat), 1)
	}
	return c
}

// handSlotsInArmOrder lists the hand slots the character has, arm 1 first.
func handSlotsInArmOrder(c *Character) []*items.Item {
	var out []*items.Item
	for _, p := range c.GetHandPairs() {
		out = append(out, p.First.ItemPtr)
		if !p.IsHalfPair() {
			out = append(out, p.Second.ItemPtr)
		}
	}
	return out
}

// goldenHandLayouts enumerates every non-cursed hand layout: one code per
// hand slot in arm order. e empty, s sword, c claws (main hand only), b
// buckler (never the main hand), g greatsword (a full pair's First; its
// Second is then e).
func goldenHandLayouts(c *Character) []string {
	layouts := []string{``}
	for pi, p := range c.GetHandPairs() {
		firsts := "esb"
		if pi == 0 {
			firsts = "esc"
		}
		var opts []string
		for _, f := range firsts {
			if p.IsHalfPair() {
				opts = append(opts, string(f))
				continue
			}
			for _, s := range "esb" {
				opts = append(opts, string(f)+string(s))
			}
		}
		if !p.IsHalfPair() {
			opts = append(opts, "ge")
		}
		var next []string
		for _, l := range layouts {
			for _, o := range opts {
				next = append(next, l+o)
			}
		}
		layouts = next
	}
	return layouts
}

var goldenCode = map[rune]items.ItemSpec{'s': goldenSword, 'c': goldenClaws, 'b': goldenShield, 'g': goldenGreat, 'r': goldenRing, 'w': goldenBracer}

// applyLayout writes one item per code into slots, numbering from *nextId.
// Upper case is the same item cursed; d writes the disabled-slot marker.
func applyLayout(slots []*items.Item, layout string, nextId *int) {
	for n, code := range layout {
		switch code {
		case 'e':
			continue
		case 'd':
			*slots[n] = items.ItemDisabledSlot
			continue
		}
		lower := []rune(strings.ToLower(string(code)))[0]
		*nextId++
		*slots[n] = goldenItem(*nextId, goldenCode[lower], lower != code)
	}
}

func goldenSnapshot(c *Character) string {
	var b strings.Builder
	for _, s := range c.Equipment.AllSlots() {
		fmt.Fprintf(&b, "%s=%d ", s.Key, s.Item.ItemId)
	}
	return b.String()
}

func goldenIds(list []items.Item) string {
	var b strings.Builder
	for _, it := range list {
		fmt.Fprintf(&b, "%d,", it.ItemId)
	}
	return b.String()
}

// sanctionedDivergence names the two placements the owner changed on purpose
// with nothing cursed: the legacy ring and wrist fallback that writes a
// disabled Ring or Wrist1 (it hands back the ItemId -1 marker; ruling 10
// never writes a disabled slot), and the shield beside a two-hander refused
// with every hand full at 3 or more arms (ruling 13 swaps the last available
// hand). Both have explicit tests in wear_slot_test.go.
func sanctionedDivergence(extraArms int, legacyReturned []items.Item, legacyWhy string) bool {
	for _, it := range legacyReturned {
		if it.ItemId < 0 {
			return true
		}
	}
	return extraArms > 0 && legacyWhy == `Your two-handed weapon leaves no room for a shield.`
}

// goldenCompare runs the legacy oracle and Wear on two identically built
// characters and reports any difference.
func goldenCompare(t *testing.T, name string, extraArms int, build func() *Character, cand items.ItemSpec, failures *int) (compared bool) {
	t.Helper()
	oracle, subject := build(), build()
	oRet, oWorn, oWhy := legacyWear(oracle, goldenItem(9000, cand, false))
	if sanctionedDivergence(extraArms, oRet, oWhy) {
		return false
	}
	sRet, sWorn, sWhy := subject.Wear(goldenItem(9000, cand, false))
	report := func(format string, args ...any) {
		*failures++
		if *failures <= 20 {
			t.Errorf("%s: "+format, append([]any{name}, args...)...)
		}
	}
	if oWorn != sWorn || oWhy != sWhy {
		report("worn/reason legacy (%v, %q), Wear (%v, %q)", oWorn, oWhy, sWorn, sWhy)
	}
	if goldenIds(oRet) != goldenIds(sRet) {
		report("returned legacy [%s], Wear [%s]", goldenIds(oRet), goldenIds(sRet))
	}
	if a, b := goldenSnapshot(oracle), goldenSnapshot(subject); a != b {
		report("equipment\n legacy %s\n Wear   %s", a, b)
	}
	return true
}

func TestWearSlotChoice_GoldenHands(t *testing.T) {
	seedGoldenSpecies(t)
	candidates := []items.ItemSpec{goldenSword, goldenClaws, goldenGreat, goldenShield}
	compared, skipped, failures := 0, 0, 0
	for _, extra := range []int{0, 1, 2, 4} {
		for _, sp := range []int{goldenMedium, goldenSmall, goldenLarge} {
			for _, dual := range []bool{false, true} {
				for _, layout := range goldenHandLayouts(goldenChar(extra, sp, dual)) {
					build := func() *Character {
						c := goldenChar(extra, sp, dual)
						id := 1000
						applyLayout(handSlotsInArmOrder(c), layout, &id)
						return c
					}
					for _, cand := range candidates {
						name := fmt.Sprintf("arms=%d species=%d dual=%v hands=%s item=%s", 2+extra, sp, dual, layout, cand.Name)
						if goldenCompare(t, name, extra, build, cand, &failures) {
							compared++
						} else {
							skipped++
						}
					}
				}
			}
		}
	}
	t.Logf("compared %d hand placements, skipped %d sanctioned", compared, skipped)
	if compared < 20000 {
		t.Fatalf("the sweep compared only %d placements: it is too small to pin anything", compared)
	}
	if failures > 0 {
		t.Fatalf("%d placement(s) differ from the legacy oracle", failures)
	}
}

func TestWearSlotChoice_GoldenRingsAndWrists(t *testing.T) {
	seedGoldenSpecies(t)
	compared, skipped, failures := 0, 0, 0
	for _, extra := range []int{0, 1, 2, 4} {
		// Rings: Ring and Ring2 each empty, full or disabled.
		for _, r1 := range "erd" {
			for _, r2 := range "erd" {
				layout := string(r1) + string(r2)
				build := func() *Character {
					c := goldenChar(extra, goldenMedium, false)
					id := 1000
					applyLayout([]*items.Item{&c.Equipment.Ring, &c.Equipment.Ring2}, layout, &id)
					return c
				}
				if goldenCompare(t, fmt.Sprintf("arms=%d rings=%s", 2+extra, layout), extra, build, goldenRing, &failures) {
					compared++
				} else {
					skipped++
				}
			}
		}
		// Wrists: Wrist1 and Wrist2 empty, full or disabled; each enabled
		// extra wrist empty or full.
		var layouts []string
		for _, w1 := range "ewd" {
			for _, w2 := range "ewd" {
				layouts = append(layouts, string(w1)+string(w2))
			}
		}
		for n := 0; n < extra; n++ {
			var next []string
			for _, l := range layouts {
				next = append(next, l+"e", l+"w")
			}
			layouts = next
		}
		for _, layout := range layouts {
			build := func() *Character {
				c := goldenChar(extra, goldenMedium, false)
				slots := []*items.Item{&c.Equipment.Wrist1, &c.Equipment.Wrist2,
					&c.Equipment.ExtraWrist1, &c.Equipment.ExtraWrist2, &c.Equipment.ExtraWrist3, &c.Equipment.ExtraWrist4}
				id := 1000
				applyLayout(slots[:len(layout)], layout, &id)
				return c
			}
			if goldenCompare(t, fmt.Sprintf("arms=%d wrists=%s", 2+extra, layout), extra, build, goldenBracer, &failures) {
				compared++
			} else {
				skipped++
			}
		}
	}
	t.Logf("compared %d ring and wrist placements, skipped %d sanctioned", compared, skipped)
	if skipped == 0 {
		t.Fatalf("no disabled-slot fallback was generated: the carve-out is untested")
	}
	if failures > 0 {
		t.Fatalf("%d placement(s) differ from the legacy oracle", failures)
	}
}
```

Note: `Item.DisplayName()` title-cases the spec name (`items.go:496-547`), so every expected refusal in this plan names the item as `Sword`, `Buckler`, `Ring` and so on.

- [ ] **Step 2: Run it against today's code**

Run: `go test ./internal/characters/ -run 'TestWearSlotChoice_Golden' -count=1 -v`
Expected: PASS for both, with the logged counts (hands above 20000). If a species fixture is missing, `HandsRequired` panics: the seeding helper above must run first in each test.

- [ ] **Step 3: Prove it can fail**

Temporarily change `legacyWearWeaponOrShield`'s final fallback to displace `Offhand` instead of `Weapon` (swap `c.Equipment.Weapon` for `c.Equipment.Offhand` in the last three statements). Run Step 2's command. Expected: FAIL with "differ from the legacy oracle". Revert the change and rerun: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/characters/wear_slot_golden_test.go
git commit -m "test(characters): golden oracle pins today's hand, ring and wrist placement"
```

---

### Task 2: `CursedRefusal` and `ChooseWornSlot`

**Model:** opus (the candidate rules are the slice's riskiest logic).

**Files:**
- Create: `internal/characters/wear_slot.go`
- Create: `internal/characters/wear_slot_test.go`
- Modify: `internal/characters/worn.go:292` (nil guard)
- Modify: `bauble_sweep_guard_test.go` (`transientItemHolders`)

- [ ] **Step 1: Write the failing helper tests**

Create `internal/characters/wear_slot_test.go`:

```go
package characters

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cursedLine = ` is cursed and prevents you from removing it.`

// fillHands puts a plain sword in every hand slot the character has, then
// applies overrides (arm number to code, as applyLayout reads it).
func fillHands(c *Character, overrides map[int]rune) {
	slots := handSlotsInArmOrder(c)
	layout := []rune(strings.Repeat("s", len(slots)))
	for arm, code := range overrides {
		layout[arm-1] = code
	}
	id := 2000
	applyLayout(slots, string(layout), &id)
}

func TestCursedRefusal(t *testing.T) {
	c := New()
	assert.Equal(t, ``, c.CursedRefusal(items.Item{}), "an empty slot never refuses")
	assert.Equal(t, ``, c.CursedRefusal(goldenItem(11, goldenRing, false)))
	assert.Equal(t, `Your Ring`+cursedLine, c.CursedRefusal(goldenItem(12, goldenRing, true)))
	lifted := goldenItem(13, goldenRing, true)
	lifted.Uncursed = true
	assert.Equal(t, ``, c.CursedRefusal(lifted), "an uncursed cursed item comes off")
}

func TestChooseWornSlot_RingsSkipACursedRing(t *testing.T) {
	seedGoldenSpecies(t)
	c := goldenChar(0, goldenMedium, false)
	id := 3000
	applyLayout([]*items.Item{&c.Equipment.Ring, &c.Equipment.Ring2}, "Rr", &id)
	choice, why := c.ChooseWornSlot(goldenItem(9000, goldenRing, false), 0)
	require.Equal(t, ``, why)
	require.Len(t, choice.Slots, 1)
	assert.Equal(t, `ring2`, choice.Slots[0].Key)
	assert.Equal(t, goldenIds([]items.Item{c.Equipment.Ring2}), goldenIds(choice.Displaced))

	applyLayout([]*items.Item{&c.Equipment.Ring, &c.Equipment.Ring2}, "RR", &id)
	_, why = c.ChooseWornSlot(goldenItem(9000, goldenRing, false), 0)
	assert.Equal(t, `Your Ring`+cursedLine, why, "all cursed: the shared line naming the Ring item")

	applyLayout([]*items.Item{&c.Equipment.Ring, &c.Equipment.Ring2}, "er", &id)
	choice, _ = c.ChooseWornSlot(goldenItem(9000, goldenRing, false), 0)
	assert.Equal(t, `ring`, choice.Slots[0].Key, "an empty slot is filled first, in today's order")
	assert.Empty(t, choice.Displaced)

	applyLayout([]*items.Item{&c.Equipment.Ring, &c.Equipment.Ring2}, "dr", &id)
	choice, why = c.ChooseWornSlot(goldenItem(9000, goldenRing, false), 0)
	require.Equal(t, ``, why)
	assert.Equal(t, `ring2`, choice.Slots[0].Key, "a disabled Ring is never written")
}

func TestChooseWornSlot_WristsSkipCursedWristsOnEveryArm(t *testing.T) {
	seedGoldenSpecies(t)
	wrists := func(c *Character) []*items.Item {
		return []*items.Item{&c.Equipment.Wrist1, &c.Equipment.Wrist2,
			&c.Equipment.ExtraWrist1, &c.Equipment.ExtraWrist2, &c.Equipment.ExtraWrist3, &c.Equipment.ExtraWrist4}
	}
	for _, tc := range []struct {
		extra  int
		layout string
		want   string
		why    string
	}{
		{0, "Ww", "wrist2", ""},
		{0, "WW", "", "Your Bracer" + cursedLine},
		{2, "WWww", "extrawrist1", ""},
		{2, "Wwww", "wrist2", ""},
		{4, "WWWWWw", "extrawrist4", ""},
		{4, "WWWWWW", "", "Your Bracer" + cursedLine},
	} {
		c := goldenChar(tc.extra, goldenMedium, false)
		id := 3100
		applyLayout(wrists(c)[:len(tc.layout)], tc.layout, &id)
		choice, why := c.ChooseWornSlot(goldenItem(9000, goldenBracer, false), 0)
		if tc.want == "" {
			assert.Equal(t, tc.why, why, "extra=%d %s", tc.extra, tc.layout)
			continue
		}
		require.Equal(t, ``, why, "extra=%d %s", tc.extra, tc.layout)
		assert.Equal(t, tc.want, choice.Slots[0].Key, "extra=%d %s", tc.extra, tc.layout)
	}
}

// handExpect is one arm count's outcome: the slot key that receives the item,
// or the refusal (substring) when slot is empty.
type handExpect struct {
	slot string
	why  string
}

// The hand table across arm counts (spec ruling 12 and 13). Every other hand
// holds a plain sword unless the row overrides it; arm numbers are 1 to 6.
func TestChooseWornSlot_HandsAcrossArmCounts(t *testing.T) {
	seedGoldenSpecies(t)
	rows := []struct {
		name      string
		dual      bool
		overrides map[int]rune
		item      items.ItemSpec
		want      map[int]handExpect // keyed by extra arms 0, 1, 2, 4
	}{
		{"cursed main hand, plain offhand, dual wielder", true, map[int]rune{1: 'S'}, goldenSword,
			map[int]handExpect{0: {"offhand", ""}, 1: {"offhand", ""}, 2: {"offhand", ""}, 4: {"offhand", ""}}},
		{"cursed main hand, empty offhand, dual wielder", true, map[int]rune{1: 'S', 2: 'e'}, goldenSword,
			map[int]handExpect{0: {"offhand", ""}, 1: {"offhand", ""}, 2: {"offhand", ""}, 4: {"offhand", ""}}},
		{"cursed main hand, no dual wield", false, map[int]rune{1: 'S'}, goldenSword,
			map[int]handExpect{0: {"", "Your Sword" + cursedLine}, 1: {"extraarm1", ""}, 2: {"extraarm1", ""}, 4: {"extraarm1", ""}}},
		{"every usable hand cursed", true, map[int]rune{1: 'S', 2: 'S', 3: 'S', 4: 'S', 5: 'S', 6: 'S'}, goldenSword,
			map[int]handExpect{0: {"", "Your Sword" + cursedLine}, 1: {"", "Your Sword" + cursedLine}, 2: {"", "Your Sword" + cursedLine}, 4: {"", "Your Sword" + cursedLine}}},
		{"cursed two-hander in the main hands", true, map[int]rune{1: 'G', 2: 'e'}, goldenSword,
			map[int]handExpect{0: {"", "Your Greatsword" + cursedLine}, 1: {"extraarm1", ""}, 2: {"extraarm1", ""}, 4: {"extraarm1", ""}}},
		{"shield over a cursed offhand item", false, map[int]rune{2: 'B'}, goldenShield,
			map[int]handExpect{0: {"", "Your Buckler" + cursedLine}, 1: {"extraarm1", ""}, 2: {"extraarm1", ""}, 4: {"extraarm1", ""}}},
		{"shield over a cursed offhand item, every extra arm cursed", false, map[int]rune{2: 'B', 3: 'S', 4: 'S', 5: 'S', 6: 'S'}, goldenShield,
			map[int]handExpect{0: {"", "Your Buckler" + cursedLine}, 1: {"", "Your Buckler" + cursedLine}, 2: {"", "Your Buckler" + cursedLine}, 4: {"", "Your Buckler" + cursedLine}}},
		{"shield beside a two-hander, every hand full", false, map[int]rune{1: 'g', 2: 'e'}, goldenShield,
			map[int]handExpect{0: {"", "no room for a shield"}, 1: {"extraarm1", ""}, 2: {"extraarm2", ""}, 4: {"extraarm4", ""}}},
		{"shield beside a two-hander, the last hand cursed", false, map[int]rune{1: 'g', 2: 'e', 4: 'S', 6: 'S'}, goldenShield,
			map[int]handExpect{0: {"", "no room for a shield"}, 1: {"extraarm1", ""}, 2: {"extraarm1", ""}, 4: {"extraarm3", ""}}},
		{"shield beside a two-hander, every candidate cursed", false, map[int]rune{1: 'g', 2: 'e', 3: 'S', 4: 'S', 5: 'S', 6: 'S'}, goldenShield,
			map[int]handExpect{0: {"", "no room for a shield"}, 1: {"", "Your Sword" + cursedLine}, 2: {"", "Your Sword" + cursedLine}, 4: {"", "Your Sword" + cursedLine}}},
		{"two-hander, cursed item in the cheapest pair", false, map[int]rune{1: 'S'}, goldenGreat,
			map[int]handExpect{0: {"", "Your Sword" + cursedLine}, 1: {"", "Your Sword" + cursedLine}, 2: {"extraarm1", ""}, 4: {"extraarm1", ""}}},
	}
	for _, row := range rows {
		for _, extra := range []int{0, 1, 2, 4} {
			c := goldenChar(extra, goldenMedium, row.dual)
			ov := map[int]rune{}
			for arm, code := range row.overrides {
				if arm <= 2+extra {
					ov[arm] = code
				}
			}
			fillHands(c, ov)
			before := goldenSnapshot(c)
			choice, why := c.ChooseWornSlot(goldenItem(9000, row.item, false), 0)
			want := row.want[extra]
			assert.Equal(t, before, goldenSnapshot(c), "%s at %d arms: ChooseWornSlot must not mutate", row.name, 2+extra)
			if want.slot == "" {
				assert.Contains(t, why, want.why, "%s at %d arms", row.name, 2+extra)
				assert.Empty(t, choice.Slots, "%s at %d arms", row.name, 2+extra)
				continue
			}
			if assert.Equal(t, ``, why, "%s at %d arms", row.name, 2+extra) {
				assert.Equal(t, want.slot, choice.Slots[0].Key, "%s at %d arms", row.name, 2+extra)
			}
		}
	}
}

// With a cursed main-hand stray behind a plain two-hander, today's 1H swap
// took the stray off unchecked (spec E2); now the curse holds it.
func TestChooseWornSlot_StrayBehindATwoHanderIsCheckedToo(t *testing.T) {
	seedGoldenSpecies(t)
	c := goldenChar(0, goldenMedium, false)
	id := 3200
	applyLayout(handSlotsInArmOrder(c), "gB", &id)
	_, why := c.ChooseWornSlot(goldenItem(9000, goldenSword, false), 0)
	assert.Equal(t, `Your Buckler`+cursedLine, why)
}

// Too many hands is refused by the helper as well as by Wear, so the scorer
// never offers a slot Wear will refuse.
func TestChooseWornSlot_TooManyHands(t *testing.T) {
	seedGoldenSpecies(t)
	c := goldenChar(4, goldenSmall, false)
	_, why := c.ChooseWornSlot(goldenItem(9000, goldenGreat, false), 0)
	assert.Equal(t, `That requires too many hands.`, why)
}

// An unregistered species is read as Medium rather than a nil dereference.
func TestHandsRequired_UnregisteredSpeciesIsMedium(t *testing.T) {
	c := &Character{SpeciesId: 424242}
	assert.Equal(t, 2, c.HandsRequired(goldenItem(9000, goldenGreat, false)))
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/characters/ -run 'TestCursedRefusal|TestChooseWornSlot|TestHandsRequired_Unregistered' -count=1`
Expected: build failure, `c.CursedRefusal undefined`, `c.ChooseWornSlot undefined`.

- [ ] **Step 3: Add the nil guard to `HandsRequired`**

In `internal/characters/worn.go`, replace

```go
	speciesInfo := species.GetSpecies(c.SpeciesId)
	if speciesInfo.Size == species.Large {
```

with

```go
	// An unregistered species reads as Medium. The upgrade scorer reaches
	// this for every weapon it weighs (slice 5a), and a nil here used to be a
	// panic, never a rule.
	speciesInfo := species.GetSpecies(c.SpeciesId)
	if speciesInfo == nil {
		return iSpec.Hands
	}
	if speciesInfo.Size == species.Large {
```

- [ ] **Step 4: Write the helper**

Create `internal/characters/wear_slot.go`:

```go
package characters

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// SlotChoice is where ChooseWornSlot puts an item: Slots[0] receives it and
// every later entry is cleared (a two-hander's partner slot, a stray behind
// a two-hander, or the two-hander an even arm breaks). Displaced is
// everything that comes off, in the order Wear hands it back.
type SlotChoice struct {
	Slots     []WornSlot
	Displaced []items.Item
}

// apply writes i into the chosen slot and clears the rest.
func (ch SlotChoice) apply(i items.Item) {
	for n, s := range ch.Slots {
		if n == 0 {
			*s.Item = i
			continue
		}
		*s.Item = items.Item{}
	}
}

// CursedRefusal is the one statement of the equip curse rule: a worn item
// that is cursed cannot be displaced by putting something else on. It has no
// Health and no Spellcasting condition, because player equip has honoured
// neither (spec ruling 8); remove keeps its own exception
// (actions.CursedHolds), so a Spellcasting-4 wearer frees the slot with
// `remove` first.
func (c *Character) CursedRefusal(it items.Item) string {
	if it.ItemId > 0 && it.IsCursed() {
		return `Your ` + it.DisplayName() + ` is cursed and prevents you from removing it.`
	}
	return ``
}

// slotCandidate is one place an item could go.
type slotCandidate struct {
	slots     []*items.Item // [0] receives the item; the rest are cleared
	displaced []items.Item  // what comes off, in Wear's return order
	guarded   []items.Item  // the same items in the order the curse check reads them
}

// occupant is a candidate that writes one slot and takes off what is in it.
func occupant(p *items.Item) slotCandidate {
	cand := slotCandidate{slots: []*items.Item{p}}
	if p.ItemId > 0 {
		cand.displaced = []items.Item{*p}
		cand.guarded = cand.displaced
	}
	return cand
}

// ChooseWornSlot decides where a ring, wrist, weapon or shield goes, over the
// arms and wrists this character actually has (2 to 6 arms). Wear, WearInArm
// and the itemvalue scorer all call it, so they agree for any arm count.
//
// arm is 0 for the automatic choice and 1 to 6 for `equip X armN`. The rule,
// over an ordered candidate list: fill the first empty candidate; else swap
// the first candidate none of whose displaced items CursedRefusal refuses;
// else refuse with CursedRefusal of the first candidate's first cursed item.
// The first candidate is always today's fallback, so with nothing cursed the
// choice is today's (the golden test pins it), apart from the two changes
// rulings 10 and 13 made on purpose.
//
// It never mutates. For any other item type it returns an empty choice and
// no refusal.
func (c *Character) ChooseWornSlot(i items.Item, arm int) (choice SlotChoice, refusal string) {
	spec := i.GetSpec()
	if arm > 0 {
		return c.chooseArm(i, spec, arm)
	}
	switch spec.Type {
	case items.Weapon, items.Offhand:
		if c.HandsRequired(i) > 2 {
			return SlotChoice{}, `That requires too many hands.`
		}
		fill, swap, none := c.handCandidates(i, spec)
		return c.chooseFrom(fill, swap, none)
	case items.Ring:
		e := &c.Equipment
		if e.Ring.IsDisabled() && e.Ring2.IsDisabled() {
			return SlotChoice{}, `You can't wear rings.`
		}
		var list []slotCandidate
		for _, p := range []*items.Item{&e.Ring, &e.Ring2} {
			if !p.IsDisabled() {
				list = append(list, occupant(p))
			}
		}
		return c.chooseFrom(list, list, `You can't wear rings.`)
	case items.Wrist:
		e := &c.Equipment
		if e.Wrist1.IsDisabled() && e.Wrist2.IsDisabled() {
			return SlotChoice{}, `You can't wear things on your wrists.`
		}
		wrists := []*items.Item{&e.Wrist1, &e.Wrist2}
		for n, p := range []*items.Item{&e.ExtraWrist1, &e.ExtraWrist2, &e.ExtraWrist3, &e.ExtraWrist4} {
			if c.ExtraArms >= n+1 {
				wrists = append(wrists, p)
			}
		}
		var list []slotCandidate
		for _, p := range wrists {
			if !p.IsDisabled() {
				list = append(list, occupant(p))
			}
		}
		return c.chooseFrom(list, list, `You can't wear things on your wrists.`)
	}
	return SlotChoice{}, ``
}

// chooseFrom applies the one rule. fill and swap are the same list except
// where today fills in one order and swaps in another (2H pairs, ruling 13).
func (c *Character) chooseFrom(fill, swap []slotCandidate, none string) (SlotChoice, string) {
	for _, cand := range fill {
		if len(cand.displaced) == 0 {
			return c.slotChoice(cand), ``
		}
	}
	for _, cand := range swap {
		if c.cursedAmong(cand.guarded) == `` {
			return c.slotChoice(cand), ``
		}
	}
	if len(swap) > 0 {
		return SlotChoice{}, c.cursedAmong(swap[0].guarded)
	}
	return SlotChoice{}, none
}

func (c *Character) cursedAmong(its []items.Item) string {
	for _, it := range its {
		if r := c.CursedRefusal(it); r != `` {
			return r
		}
	}
	return ``
}

// slotChoice names a candidate's slots by their AllSlots entries.
func (c *Character) slotChoice(cand slotCandidate) SlotChoice {
	ch := SlotChoice{Displaced: cand.displaced}
	all := c.Equipment.AllSlots()
	for _, p := range cand.slots {
		for _, s := range all {
			if s.Item == p {
				ch.Slots = append(ch.Slots, s)
				break
			}
		}
	}
	return ch
}

// handCandidates builds the weapon and shield lists (spec "Slot choice").
func (c *Character) handCandidates(i items.Item, spec items.ItemSpec) (fill, swap []slotCandidate, none string) {
	pairs := c.GetHandPairs()

	// Two-hander: whole pairs only, a free pair first, then fewest
	// occupants, the earlier pair on a tie (today's FindFirstFreePair then
	// FindCheapestPairToDisplace, as one stable sort).
	if c.HandsRequired(i) >= 2 {
		for _, p := range pairs {
			if p.IsHalfPair() {
				continue
			}
			cand := slotCandidate{slots: []*items.Item{p.First.ItemPtr, p.Second.ItemPtr}}
			if !p.First.IsEmpty() {
				cand.displaced = append(cand.displaced, *p.First.ItemPtr)
			}
			if !p.Second.IsEmpty() {
				cand.displaced = append(cand.displaced, *p.Second.ItemPtr)
			}
			cand.guarded = cand.displaced
			fill = append(fill, cand)
		}
		swap = append([]slotCandidate(nil), fill...)
		sort.SliceStable(swap, func(a, b int) bool { return len(swap[a].displaced) < len(swap[b].displaced) })
		return fill, swap, `You have no free pair of hands for a two-handed weapon.`
	}

	weapon2H := pairs[0].First.Is2H(c)

	// Shield: Offhand unless the main hands hold a two-hander, then the
	// extra arms in arm order.
	if spec.Type == items.Offhand {
		if !weapon2H {
			fill = append(fill, occupant(pairs[0].Second.ItemPtr))
		}
		fill = append(fill, c.handSlotCandidates(pairs, 1)...)
		if !weapon2H {
			return fill, fill, ``
		}
		// Ruling 13: beside a two-hander in the main hands, with no hand
		// empty, the shield takes the LAST available hand: the highest arm,
		// counting down, that is not part of a two-hander.
		for n := len(pairs) - 1; n >= 1; n-- {
			p := pairs[n]
			if p.First.Is2H(c) {
				continue
			}
			if !p.IsHalfPair() {
				swap = append(swap, occupant(p.Second.ItemPtr))
			}
			swap = append(swap, occupant(p.First.ItemPtr))
		}
		return fill, swap, `Your two-handed weapon leaves no room for a shield.`
	}

	// One-hander: arms in order. Offhand only for a dual wielder or claws
	// over claws; a pair holding a two-hander offers only its First.
	bothMartial := spec.Subtype == items.Claws && c.Equipment.Weapon.GetSpec().Subtype == items.Claws
	if weapon2H {
		fill = append(fill, c.twoHanderSlot(pairs[0]))
	} else {
		fill = append(fill, occupant(pairs[0].First.ItemPtr))
		if c.CanDualWield() || bothMartial {
			fill = append(fill, occupant(pairs[0].Second.ItemPtr))
		}
	}
	fill = append(fill, c.handSlotCandidates(pairs, 1)...)
	return fill, fill, `You have no free hand for that.`
}

// handSlotCandidates lists pairs[from:] in arm order. A pair whose First
// holds a two-hander offers only that First: its Second is consumed.
func (c *Character) handSlotCandidates(pairs []HandPair, from int) []slotCandidate {
	var out []slotCandidate
	for _, p := range pairs[from:] {
		if p.First.Is2H(c) {
			out = append(out, c.twoHanderSlot(p))
			continue
		}
		out = append(out, occupant(p.First.ItemPtr))
		if !p.IsHalfPair() {
			out = append(out, occupant(p.Second.ItemPtr))
		}
	}
	return out
}

// twoHanderSlot is the First of a pair holding a two-hander: writing it takes
// the two-hander off, and any stray left behind it in the Second too. The
// stray comes back first (today's order); the curse check reads the
// two-hander first (today's order, which never read the stray at all).
func (c *Character) twoHanderSlot(p HandPair) slotCandidate {
	cand := slotCandidate{slots: []*items.Item{p.First.ItemPtr}}
	var stray []items.Item
	if !p.IsHalfPair() && !p.Second.IsEmpty() {
		cand.slots = append(cand.slots, p.Second.ItemPtr)
		stray = []items.Item{*p.Second.ItemPtr}
	}
	cand.displaced = append(append([]items.Item(nil), stray...), *p.First.ItemPtr)
	cand.guarded = append([]items.Item{*p.First.ItemPtr}, stray...)
	return cand
}

// chooseArm is `equip X armN` (ruling 11 and 12): the player's shape
// refusals, then the one slot the arm names. A cursed item there refuses; it
// never falls through to another arm.
func (c *Character) chooseArm(i items.Item, spec items.ItemSpec, arm int) (SlotChoice, string) {
	if spec.Type != items.Weapon && spec.Type != items.Offhand {
		return SlotChoice{}, `You can only wield weapons or shields in arm slots.`
	}
	if spec.Type == items.Offhand && arm == 1 {
		return SlotChoice{}, `You can't put a shield in your primary weapon hand (arm 1).`
	}
	pairs := c.GetHandPairs()
	pairIdx, slotInPair := (arm-1)/2, (arm-1)%2
	if pairIdx >= len(pairs) || (slotInPair == 1 && pairs[pairIdx].IsHalfPair()) {
		return SlotChoice{}, fmt.Sprintf(`You don't have arm %d.`, arm)
	}
	if c.HandsRequired(i) > 2 {
		return SlotChoice{}, `That requires too many hands.`
	}
	pair := pairs[pairIdx]
	twoHanded := c.HandsRequired(i) >= 2
	if twoHanded {
		if slotInPair != 0 {
			return SlotChoice{}, `A two-handed weapon needs a pair of arms. Try arm 1, 3, or 5.`
		}
		if pair.IsHalfPair() {
			return SlotChoice{}, `That arm doesn't have a partner for a two-handed weapon.`
		}
	}
	target := pair.First
	if slotInPair == 1 {
		target = pair.Second
	}

	cand := slotCandidate{slots: []*items.Item{target.ItemPtr}}
	// The curse check reads what the arm branch read, in its order: the
	// target, a two-hander's second slot, the pair's First, then a two-handed
	// partner beside an even arm.
	if !target.IsEmpty() {
		cand.guarded = append(cand.guarded, *target.ItemPtr)
	}
	if twoHanded {
		if !pair.Second.IsEmpty() {
			cand.guarded = append(cand.guarded, *pair.Second.ItemPtr)
		}
		if !pair.First.IsEmpty() {
			cand.guarded = append(cand.guarded, *pair.First.ItemPtr)
		}
	}
	if slotInPair == 1 && pair.First.Is2H(c) {
		cand.guarded = append(cand.guarded, *pair.First.ItemPtr)
		cand.displaced = append(cand.displaced, *pair.First.ItemPtr)
		cand.slots = append(cand.slots, pair.First.ItemPtr)
	}
	if !target.IsEmpty() {
		cand.displaced = append(cand.displaced, *target.ItemPtr)
	}
	if twoHanded && !pair.Second.IsEmpty() {
		cand.displaced = append(cand.displaced, *pair.Second.ItemPtr)
		cand.slots = append(cand.slots, pair.Second.ItemPtr)
	}
	if r := c.cursedAmong(cand.guarded); r != `` {
		return SlotChoice{}, r
	}
	return c.slotChoice(cand), ``
}
```

- [ ] **Step 5: Register the new item holders**

In `bauble_sweep_guard_test.go`, add to `transientItemHolders` (keep the map sorted):

```go
	`internal/characters.SlotChoice`:             `a view: pointers into Worn plus the items one equip displaces, alive for one call`,
	`internal/characters.slotCandidate`:          `one equip's candidate slot, alive for one call`,
```

- [ ] **Step 6: Run the tests**

Run: `go test ./internal/characters/ -count=1` then `go test . -run 'TestBaubleSweep' -count=1`
Expected: PASS. The golden tests still pass (Wear is untouched). If a hand-table row fails, re-read the row against the spec's rule before touching the helper: the rows are the spec's own list.

- [ ] **Step 7: Commit**

```bash
git add internal/characters/wear_slot.go internal/characters/wear_slot_test.go internal/characters/worn.go bauble_sweep_guard_test.go
git commit -m "feat(characters): ChooseWornSlot and CursedRefusal, one slot choice over every arm"
```

---

### Task 3: `Wear` goes through the helper and checks the curse after placement

**Model:** opus.

**Files:**
- Modify: `internal/characters/worn.go` (`wearWeaponOrShield` deleted, Ring/Wrist cases, `Wear` split into `wear`)
- Modify: `internal/characters/hand_slots.go` (delete `FindFirstEmptySlot`, `FindFirstFreePair`, `FindCheapestPairToDisplace`, `PairIsFree`, `PairOccupantCount`)
- Create: `internal/characters/wear_curse_test.go`

- [ ] **Step 1: Write the failing `Wear` curse tests**

Create `internal/characters/wear_curse_test.go`:

```go
package characters

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func armourSpec(name string, t items.ItemType) items.ItemSpec {
	return items.ItemSpec{Name: name, Type: t, Subtype: items.Wearable}
}

// Every single-slot armour family refuses to displace a cursed piece, and a
// refusal leaves the body exactly as it was (spec ruling 8, E3).
func TestWear_RefusesToDisplaceCursedArmour(t *testing.T) {
	seedGoldenSpecies(t)
	for _, tc := range []struct {
		slot func(c *Character) *items.Item
		typ  items.ItemType
	}{
		{func(c *Character) *items.Item { return &c.Equipment.Head }, items.Head},
		{func(c *Character) *items.Item { return &c.Equipment.Neck }, items.Neck},
		{func(c *Character) *items.Item { return &c.Equipment.Body }, items.Body},
		{func(c *Character) *items.Item { return &c.Equipment.Belt }, items.Belt},
		{func(c *Character) *items.Item { return &c.Equipment.Gloves }, items.Gloves},
		{func(c *Character) *items.Item { return &c.Equipment.Back }, items.Back},
		{func(c *Character) *items.Item { return &c.Equipment.Shoulders }, items.Shoulders},
		{func(c *Character) *items.Item { return &c.Equipment.Legs }, items.Legs},
		{func(c *Character) *items.Item { return &c.Equipment.Feet }, items.Feet},
		{func(c *Character) *items.Item { return &c.Equipment.ComponentBag }, items.ComponentBag},
		{func(c *Character) *items.Item { return &c.Equipment.Light }, items.Light},
	} {
		c := goldenChar(0, goldenMedium, false)
		old := goldenItem(4001, armourSpec("old "+string(tc.typ), tc.typ), true)
		*tc.slot(c) = old
		before := goldenSnapshot(c)
		ret, worn, why := c.Wear(goldenItem(4002, armourSpec("new "+string(tc.typ), tc.typ), false))
		assert.False(t, worn, "%s", tc.typ)
		assert.Nil(t, ret, "%s", tc.typ)
		// DisplayName title-cases the name ("Old Head").
		assert.Equal(t, `Your `+old.DisplayName()+cursedLine, why, "%s", tc.typ)
		assert.Equal(t, before, goldenSnapshot(c), "%s: a refusal leaves the body as it was", tc.typ)
	}
}

func TestWear_AnUncursedItemSwapsAsToday(t *testing.T) {
	seedGoldenSpecies(t)
	c := goldenChar(0, goldenMedium, false)
	lifted := goldenItem(4101, armourSpec("hood", items.Head), true)
	lifted.Uncursed = true
	c.Equipment.Head = lifted
	ret, worn, why := c.Wear(goldenItem(4102, armourSpec("cap", items.Head), false))
	require.True(t, worn, why)
	assert.Equal(t, "4101,", goldenIds(ret))
	assert.Equal(t, 4102, c.Equipment.Head.ItemId)
}

// A cursed refusal reads as the curse even when the swap would also breach
// the reservation ceiling: the curse pass runs before the reservation check.
func TestWear_ACursedRefusalWinsOverAReservationRefusal(t *testing.T) {
	seedGoldenSpecies(t)
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		4201: {ItemId: 4201, Name: "hungry collar", Type: items.Neck, Subtype: items.Wearable, ReserveStaminaPct: 0.60},
		4202: {ItemId: 4202, Name: "cursed belt", Type: items.Belt, Subtype: items.Wearable, Cursed: true},
		4203: {ItemId: 4203, Name: "hungry belt", Type: items.Belt, Subtype: items.Wearable, ReserveStaminaPct: 0.30},
	})()
	c := New()
	c.StaminaMax.Base = 100
	c.Equipment.Neck = items.New(4201)
	c.Equipment.Belt = items.New(4202)
	c.Validate()
	_, worn, why := c.Wear(items.New(4203))
	assert.False(t, worn)
	assert.Equal(t, `Your Cursed Belt`+cursedLine, why)
	assert.False(t, strings.Contains(strings.ToLower(why), "reserve"))
}

// The 2-arm hand families, each with no uncursed alternative.
func TestWear_HandFamiliesAtTwoArms(t *testing.T) {
	seedGoldenSpecies(t)
	for _, tc := range []struct {
		name   string
		layout string
		item   items.ItemSpec
		why    string
	}{
		{"2H over a cursed main hand", "Ss", goldenGreat, "Your Sword" + cursedLine},
		{"shield over a cursed offhand item", "sB", goldenShield, "Your Buckler" + cursedLine},
		{"1H over a cursed main hand, no dual wield", "Ss", goldenSword, "Your Sword" + cursedLine},
		{"1H over a two-hander with a cursed stray behind it", "gB", goldenSword, "Your Buckler" + cursedLine},
	} {
		c := goldenChar(0, goldenMedium, false)
		id := 4300
		applyLayout(handSlotsInArmOrder(c), tc.layout, &id)
		before := goldenSnapshot(c)
		_, worn, why := c.Wear(goldenItem(9000, tc.item, false))
		assert.False(t, worn, tc.name)
		assert.Equal(t, tc.why, why, tc.name)
		assert.Equal(t, before, goldenSnapshot(c), tc.name)
	}
}

// Wear lands exactly where ChooseWornSlot said, for the whole hand table.
func TestWear_FollowsTheHelperAcrossArmCounts(t *testing.T) {
	seedGoldenSpecies(t)
	for _, extra := range []int{0, 1, 2, 4} {
		for _, item := range []items.ItemSpec{goldenSword, goldenShield, goldenGreat} {
			for _, cursedArm := range []int{0, 1, 2, 3, 4, 5, 6} {
				c := goldenChar(extra, goldenMedium, false)
				ov := map[int]rune{}
				if cursedArm > 0 && cursedArm <= 2+extra {
					ov[cursedArm] = 'S'
				}
				fillHands(c, ov)
				choice, refusal := c.ChooseWornSlot(goldenItem(9000, item, false), 0)
				ret, worn, why := c.Wear(goldenItem(9000, item, false))
				if refusal != `` {
					assert.False(t, worn)
					assert.Equal(t, refusal, why)
					continue
				}
				require.True(t, worn, why)
				assert.Equal(t, goldenIds(choice.Displaced), goldenIds(ret))
				assert.Equal(t, 9000, choice.Slots[0].Item.ItemId, "the chosen slot now holds the item")
			}
		}
	}
}

// Ruling 13's nothing-cursed change, and the disabled-Ring change, both
// carved out of the golden test.
func TestWear_SanctionedChangesWithNothingCursed(t *testing.T) {
	seedGoldenSpecies(t)
	c := goldenChar(1, goldenMedium, false)
	id := 4400
	applyLayout(handSlotsInArmOrder(c), "ges", &id)
	ret, worn, why := c.Wear(goldenItem(9000, goldenShield, false))
	require.True(t, worn, why)
	assert.Equal(t, 9000, c.Equipment.ExtraArm1.ItemId, "the shield takes arm 3")
	assert.Len(t, ret, 1)

	c = goldenChar(0, goldenMedium, false)
	applyLayout([]*items.Item{&c.Equipment.Ring, &c.Equipment.Ring2}, "dr", &id)
	_, worn, why = c.Wear(goldenItem(9001, goldenRing, false))
	require.True(t, worn, why)
	assert.Equal(t, 9001, c.Equipment.Ring2.ItemId)
	assert.True(t, c.Equipment.Ring.IsDisabled(), "a disabled Ring is never written")
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/characters/ -run 'TestWear_' -count=1`
Expected: FAIL (armour swaps over cursed pieces; the stray row and the sanctioned rows fail).

- [ ] **Step 3: Rewrite `Wear` as `wear(i, place)` with the curse pass**

In `internal/characters/worn.go`, delete `wearWeaponOrShield` (`:369-465`) and replace the whole `Wear` function with:

```go
// Wear puts an item on the body and returns what came off. Its callers
// (actions.EquipItem, the merchant upgrade, spawn and bounty-hunter loot, the
// companion's starting kit) see one refusal wording per gate.
func (c *Character) Wear(i items.Item) (returnItems []items.Item, newItemWorn bool, failureReason string) {
	return c.wear(i, func(i items.Item, spec items.ItemSpec) ([]items.Item, bool, string) {
		if spec.Type == items.Weapon || spec.Type == items.Offhand {
			return c.wearChosen(i, 0)
		}
		return c.wearArmorSlot(i, spec)
	})
}

// wearChosen places a weapon or shield where ChooseWornSlot says (arm 0 for
// the automatic choice, 1 to 6 for a named arm) and reapplies permanent
// conditions, as the hand path always has.
func (c *Character) wearChosen(i items.Item, arm int) ([]items.Item, bool, string) {
	choice, refusal := c.ChooseWornSlot(i, arm)
	if refusal != `` {
		return nil, false, refusal
	}
	choice.apply(i)
	c.reapplyPermanentConditions()
	return choice.Displaced, true, ``
}

// wear is Wear's body with the placement step passed in, so Wear and
// WearInArm share every gate: the type gate, MinStrength, hands over two, the
// reservation snapshot and revert, the curse pass and the success tail.
func (c *Character) wear(i items.Item, place func(items.Item, items.ItemSpec) ([]items.Item, bool, string)) (returnItems []items.Item, newItemWorn bool, failureReason string) {

	i.Validate()

	spec := i.GetSpec()

	if spec.Type != items.Weapon && spec.Subtype != items.Wearable {
		return returnItems, false, `That item cannot be equipped.`
	}

	// Min-Strength wield gate: heavy bows and arbalests require a minimum
	// Strength to operate. Checked before HandsRequired so the rejection is
	// immediate and consistent for all callers.
	if spec.MinStrength > 0 && c.Stats.Strength.ValueAdj < spec.MinStrength {
		return returnItems, false, `You aren't strong enough to handle ` + i.DisplayName() + `.`
	}

	if c.HandsRequired(i) > 2 {
		return returnItems, false, `That requires too many hands.`
	}

	// (Keep the existing U7b reservation comment block here verbatim, with
	// "wearWeaponOrShield" replaced by "wearChosen" in its last paragraph.)
	beforeReserve := c.ReservationOverages()
	savedEquipment := c.Equipment

	returnItems, newItemWorn, failureReason = place(i, spec)
	if !newItemWorn {
		return returnItems, newItemWorn, failureReason
	}

	// The equip curse rule, on what placement actually displaced (slice 5a,
	// ruling 8): no slot choice is copied here, so every single-slot armour
	// type and the light are covered. Hands, rings and wrists never reach it
	// with a cursed item, because ChooseWornSlot already skipped or refused
	// the slot. Checked BEFORE the reservation test so a cursed refusal reads
	// as the curse. The revert is the reservation check's own.
	for _, d := range returnItems {
		if reason := c.CursedRefusal(d); reason != `` {
			c.Equipment = savedEquipment
			c.reapplyPermanentConditions()
			return nil, false, reason
		}
	}

	if pool, worse := beforeReserve.Worsened(c.ReservationOverages()); worse {
		c.Equipment = savedEquipment
		c.reapplyPermanentConditions()
		// The item's own contribution is what "added" means here: the snapshot
		// delta measures OVERAGE growth, which is smaller than the demand
		// whenever the wearer was already over, and would misreport a
		// single-item-too-heavy refusal as a merely-full one.
		return nil, false, c.ReservationRefusal(pool, c.ItemReserveOnPool(i, pool))
	}

	if spec.Type == items.ComponentBag {
		c.SortComponentItems()
	}
	if spec.Type != items.Weapon && spec.Type != items.Offhand {
		// Preserved from the pre-U7b shape: permanent conditions are reapplied on the
		// armour path only (wearChosen does its own), and only on success.
		c.reapplyPermanentConditions()
	}
	if spec.Type == items.Light {
		// Equipping a light is its fresh start (lighting plan 5a): full
		// strength, hood open, even when the item it replaced shared its
		// condition and the refresh above kept that record.
		for _, id := range spec.WornConditionIds {
			for _, rec := range c.Conditions.GetConditions(id) {
				rec.ResetLight()
			}
		}
	}
	return returnItems, newItemWorn, failureReason
}
```

(The instruction in the comment placeholder means: move the existing 20-line `U7b reservation ceiling` comment from today's `Wear` into this spot unchanged except for that one function name. Do not leave the parenthetical in the code.)

- [ ] **Step 4: Ring and Wrist through the helper**

In `wearArmorSlot`, replace the `case items.Ring:` and `case items.Wrist:` blocks (`:504-535`) with:

```go
	case items.Ring, items.Wrist:
		// Slot choice lives in ChooseWornSlot (slice 5a, ruling 10): an empty
		// slot is filled first in today's order; with every slot full the
		// first uncursed one is swapped, and a disabled slot is never
		// written.
		choice, refusal := c.ChooseWornSlot(i, 0)
		if refusal != `` {
			return returnItems, false, refusal
		}
		choice.apply(i)
		returnItems = choice.Displaced
```

Update `wearArmorSlot`'s doc comment: it no longer handles ring and wrist choice itself.

- [ ] **Step 5: Delete the dead hand helpers**

In `internal/characters/hand_slots.go`, delete `PairIsFree`, `PairOccupantCount`, `FindFirstEmptySlot`, `FindFirstFreePair` and `FindCheapestPairToDisplace` (`:73-149`). Confirm nothing else named them:

Run: `grep -rn "FindFirstEmptySlot\|FindFirstFreePair\|FindCheapestPairToDisplace\|PairIsFree\|PairOccupantCount" internal modules --include=*.go`
Expected: hits only in `internal/characters/wear_slot_golden_test.go` (the `legacy*` copies do not use these names; if one does, the copy is wrong).

- [ ] **Step 6: Run the characters package**

Run: `go test ./internal/characters/ -count=1`
Expected: PASS, including both golden tests (the hand sweep still over 20000 compared). A golden failure here means the helper moved a non-cursed placement: fix the helper, never the oracle.

- [ ] **Step 7: Commit**

```bash
git add internal/characters/worn.go internal/characters/hand_slots.go internal/characters/wear_curse_test.go
git commit -m "feat(characters): Wear places through ChooseWornSlot and refuses to displace any cursed item"
```

---

### Task 4: `WearInArm` and `ArmLabel`

**Model:** sonnet.

**Files:**
- Modify: `internal/characters/worn.go` (add `WearInArm`)
- Modify: `internal/characters/hand_slots.go` (add `ArmLabel`)
- Modify: `internal/characters/wear_slot_test.go` (arm-N rows)

- [ ] **Step 1: Write the failing arm tests**

Append to `internal/characters/wear_slot_test.go`:

```go
func TestWearInArm(t *testing.T) {
	seedGoldenSpecies(t)

	// A cursed arm-5 item refuses the named arm, though arms 1 to 4 and 6
	// are plain, and nothing moves (ruling 12).
	c := goldenChar(4, goldenMedium, false)
	fillHands(c, map[int]rune{5: 'S'})
	before := goldenSnapshot(c)
	_, worn, why := c.WearInArm(goldenItem(9000, goldenSword, false), 5)
	assert.False(t, worn)
	assert.Equal(t, `Your Sword`+cursedLine, why)
	assert.Equal(t, before, goldenSnapshot(c))

	c = goldenChar(2, goldenMedium, false)
	_, _, why = c.WearInArm(goldenItem(9000, goldenSword, false), 5)
	assert.Equal(t, `You don't have arm 5.`, why)

	// The five shape refusals keep the player's wording.
	c = goldenChar(0, goldenMedium, false)
	_, _, why = c.WearInArm(goldenItem(9000, armourSpec("cap", items.Head), false), 1)
	assert.Equal(t, `You can only wield weapons or shields in arm slots.`, why)
	_, _, why = c.WearInArm(goldenItem(9000, goldenShield, false), 1)
	assert.Equal(t, `You can't put a shield in your primary weapon hand (arm 1).`, why)
	_, _, why = c.WearInArm(goldenItem(9000, goldenSword, false), 3)
	assert.Equal(t, `You don't have arm 3.`, why)
	_, _, why = c.WearInArm(goldenItem(9000, goldenGreat, false), 2)
	assert.Equal(t, `A two-handed weapon needs a pair of arms. Try arm 1, 3, or 5.`, why)
	c = goldenChar(1, goldenMedium, false)
	_, _, why = c.WearInArm(goldenItem(9000, goldenGreat, false), 3)
	assert.Equal(t, `That arm doesn't have a partner for a two-handed weapon.`, why)

	// MinStrength runs before the arm's shape refusals.
	c = goldenChar(0, goldenMedium, false)
	c.Stats.Strength.ValueAdj = 1
	heavy := goldenSword
	heavy.MinStrength = 500
	_, _, why = c.WearInArm(goldenItem(9000, heavy, false), 3)
	assert.Equal(t, `You aren't strong enough to handle Sword.`, why)

	// Arm 2 beside a two-hander takes the two-hander off.
	c = goldenChar(0, goldenMedium, false)
	id := 4500
	applyLayout(handSlotsInArmOrder(c), "ge", &id)
	ret, worn, why := c.WearInArm(goldenItem(9000, goldenShield, false), 2)
	require.True(t, worn, why)
	assert.Equal(t, "4501,", goldenIds(ret))
	assert.Equal(t, 0, c.Equipment.Weapon.ItemId)
	assert.Equal(t, 9000, c.Equipment.Offhand.ItemId)

	// A cursed two-hander beside arm 2, and a cursed second slot under a
	// two-hander named at arm 1, both refuse with the shared line.
	c = goldenChar(0, goldenMedium, false)
	applyLayout(handSlotsInArmOrder(c), "Ge", &id)
	_, _, why = c.WearInArm(goldenItem(9000, goldenShield, false), 2)
	assert.Equal(t, `Your Greatsword`+cursedLine, why)
	c = goldenChar(0, goldenMedium, false)
	applyLayout(handSlotsInArmOrder(c), "sB", &id)
	_, _, why = c.WearInArm(goldenItem(9000, goldenGreat, false), 1)
	assert.Equal(t, `Your Buckler`+cursedLine, why)
}

func TestArmLabel(t *testing.T) {
	seedGoldenSpecies(t)
	c := goldenChar(1, goldenMedium, false)
	assert.Equal(t, `wielded`, c.ArmLabel(1))
	assert.Equal(t, `offhand`, c.ArmLabel(2))
	assert.Equal(t, `extra arm 1`, c.ArmLabel(3))
	assert.Equal(t, ``, c.ArmLabel(4), "a half pair has no arm 4")
	assert.Equal(t, ``, c.ArmLabel(0))
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/characters/ -run 'TestWearInArm|TestArmLabel' -count=1`
Expected: build failure, `c.WearInArm undefined`, `c.ArmLabel undefined`.

- [ ] **Step 3: Implement**

In `internal/characters/worn.go`, after `Wear`:

```go
// WearInArm is `equip X armN` (spec ruling 11): Wear's gates (type,
// MinStrength, hands over two, reservation, curse) around a placement into
// the one arm named, 1 to 6. A cursed item in that arm refuses; it never
// moves the item to another arm (ruling 12).
func (c *Character) WearInArm(i items.Item, arm int) (returnItems []items.Item, newItemWorn bool, failureReason string) {
	return c.wear(i, func(i items.Item, _ items.ItemSpec) ([]items.Item, bool, string) {
		return c.wearChosen(i, arm)
	})
}
```

In `internal/characters/hand_slots.go`, after `GetHandPairs`:

```go
// ArmLabel is the label of arm N (1 to 6) as GetHandPairs names it
// ("wielded", "offhand", "extra arm 1" ...), or "" when the character has no
// such arm.
func (c *Character) ArmLabel(arm int) string {
	if arm < 1 {
		return ``
	}
	pairs := c.GetHandPairs()
	pairIdx, slotInPair := (arm-1)/2, (arm-1)%2
	if pairIdx >= len(pairs) {
		return ``
	}
	if slotInPair == 0 {
		return pairs[pairIdx].First.Label
	}
	if pairs[pairIdx].IsHalfPair() {
		return ``
	}
	return pairs[pairIdx].Second.Label
}
```

- [ ] **Step 4: Run the package**

Run: `go test ./internal/characters/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/worn.go internal/characters/hand_slots.go internal/characters/wear_slot_test.go
git commit -m "feat(characters): WearInArm, equip into a named arm through Wear's gates"
```

---

### Task 5: `actions.EquipItemInArm` and the player's arm branch

**Model:** sonnet.

**Files:**
- Modify: `internal/actions/remove_equip.go:10-84`
- Create: `internal/actions/sight_gates_parity_test.go`
- Modify: `internal/usercommands/equip.go:113-342`
- Create: `internal/usercommands/equip_arm_test.go`
- Modify (re-key if the guard asks): `messaging_surface_guard_test.go:1347-1350`

- [ ] **Step 1: Write the parity harness and the equip row**

Create `internal/actions/sight_gates_parity_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The 5a parity table (sight and gates slice 5a). One player and one mob,
// built alike, stand in one room at a pinned light; every row drives a
// shared body through each actor and asserts the same outcome. The fixture
// asserts both actors' sight before any row runs, so a verdict is exact,
// never rolled.

const (
	gateRoomId    = 7950
	gateUserId    = 7951
	gateMobId     = 97952
	gateInfraCond = 7953
)

type gateLight int

const (
	gateLit gateLight = iota
	gateShapes
	gateDark
	gateBlinded
)

func (l gateLight) String() string { return [...]string{"lit", "shapes", "dark", "blinded"}[l] }

var gateLights = []gateLight{gateLit, gateShapes, gateDark, gateBlinded}

type gateScene struct {
	room *rooms.Room
	user *users.UserRecord
	mob  *mobs.Mob
}

func (s gateScene) actor(who string) Actor {
	if who == "player" {
		return NewUserActorInRoom(s.user, s.room)
	}
	return NewMobActorInRoom(s.mob, s.room)
}

var gateWho = []string{"player", "mob"}

func newGateScene(t *testing.T, light gateLight) gateScene {
	t.Helper()
	t.Cleanup(species.SeedSpeciesForTest(map[int]*species.Species{
		0: {SpeciesId: 0, Name: "Human", Size: species.Medium},
	}))
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		gateInfraCond: {ConditionId: gateInfraCond, Name: "Test Heat Sight", RoundInterval: 1, TriggerCount: 10,
			Flags:   []conditions.Flag{conditions.InfraredVision},
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectInfraReach: {Literal: 30}}},
	}))

	room := &rooms.Room{RoomId: gateRoomId, SkyLight: rooms.SkyLightPtr(0), Lamp: rooms.LampPtr(90)}
	if light == gateShapes || light == gateDark {
		room.Lamp = rooms.LampPtr(0)
	}

	u := users.NewTestUser(gateUserId, "gatey", "Gatey", uint64(gateUserId))
	u.Character.RoomId = gateRoomId
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{gateUserId: u}))
	room.AddPlayer(gateUserId)

	mc := characters.New()
	mc.Name, mc.RoomId, mc.Health = "Gatemob", gateRoomId, 100
	mc.HealthMax.Value = 100
	m := &mobs.Mob{InstanceId: gateMobId, Character: *mc}
	m.Character.MobInstanceId = gateMobId
	mobs.SetInstanceForTest(gateMobId, m)
	t.Cleanup(func() { mobs.SetInstanceForTest(gateMobId, nil) })
	room.AddMob(gateMobId)

	want := map[gateLight]messaging.SightDecision{
		gateLit: messaging.SightFull, gateShapes: messaging.SightShapes,
		gateDark: messaging.SightNone, gateBlinded: messaging.SightNone,
	}[light]
	for who, c := range map[string]*characters.Character{"player": u.Character, "mob": &m.Character} {
		switch light {
		case gateShapes:
			require.NoError(t, c.AddCondition(gateInfraCond, true))
		case gateBlinded:
			c.Perception = characters.New().Perception
			require.NoError(t, c.Perception.TransitionTo(perception.Blinded, state.TransitionReason{Trigger: "test"}))
		}
		require.Equal(t, want, messaging.ParticipantSight(c, room), "fixture: the %s must sit at %s", who, light)
	}
	return gateScene{room: room, user: u, mob: m}
}

// gateItem is a distinct item with its spec inline.
func gateItem(id int, name string, t items.ItemType, cursed bool) items.Item {
	return items.Item{ItemId: id, Spec: &items.ItemSpec{ItemId: id, Name: name, Type: t, Subtype: items.Wearable, Cursed: cursed}}
}

// Equip over a cursed piece: refused with the shared line, the candidate
// stays in the pack, the cursed piece stays on (ruling 8).
func TestGateParity_EquipOverCursedArmour(t *testing.T) {
	for _, who := range gateWho {
		s := newGateScene(t, gateLit)
		a := s.actor(who)
		c := a.GetCharacter()
		c.Equipment.Head = gateItem(39601, "iron helm", items.Head, true)
		require.True(t, c.StoreItem(gateItem(39602, "leather cap", items.Head, false)))
		res := EquipItem(a, "leather cap")
		assert.True(t, res.Found, who)
		assert.False(t, res.Equipped, who)
		assert.Equal(t, `Your Iron Helm is cursed and prevents you from removing it.`, res.FailureReason, who)
		assert.Equal(t, 39601, c.Equipment.Head.ItemId, who)
		_, inPack := c.FindInBackpack("leather cap")
		assert.True(t, inPack, "%s: the candidate stays in the pack", who)
	}
}
```

If the `gateShapes` fixture check fails (infrared not reading shapes with `SkyLight 0, Lamp 0`), set `room.Biome = "cave"` and seed the `cave` biome exactly as `cast_sight_test.go:49-56` does; do not loosen the assertion.

- [ ] **Step 2: Run it**

Run: `go test ./internal/actions/ -run 'TestGateParity_' -count=1 -v`
Expected: PASS already (Task 3 put the curse in `Wear`), proving `EquipItem` carries it for both actors. This row is a regression pin, not a red step.

- [ ] **Step 3: Write the failing arm-equip wrapper tests**

Create `internal/usercommands/equip_arm_test.go`:

```go
package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// `equip X armN` goes through Wear (spec ruling 11): Wear's MinStrength,
// reservation and curse gates, EquipItem's full-pack floor rule, and the
// arm branch's own shape refusals and lines.

func armItem(id int, name string, t items.ItemType, hands int, cursed bool) items.Item {
	sub := items.Wearable
	if t == items.Weapon {
		sub = items.Slashing
	}
	return items.Item{ItemId: id, Spec: &items.ItemSpec{ItemId: id, Name: name, Type: t, Subtype: sub, Hands: hands, Cursed: cursed}}
}

func armUser(t *testing.T) (*users.UserRecord, *rooms.Room) {
	t.Helper()
	t.Cleanup(seedAllRegistries())
	user, room := getTestUserAndRoom(t)
	user.Character.Equipment.Weapon = items.Item{}
	user.Character.Equipment.Offhand = items.Item{}
	events.DrainQueuedMessagesForTest(user.UserId)
	return user, room
}

func armOut(t *testing.T, user *users.UserRecord, room *rooms.Room, rest string) string {
	t.Helper()
	handled, err := Equip(rest, user, room, 0)
	require.NoError(t, err)
	require.True(t, handled)
	return strings.Join(events.DrainQueuedMessagesForTest(user.UserId), "\n")
}

func TestEquipArm_TooWeakIsRefusedFirst(t *testing.T) {
	user, room := armUser(t)
	heavy := armItem(96001, "arbalest", items.Weapon, 1, false)
	heavy.Spec.MinStrength = 500
	require.True(t, user.Character.StoreItem(heavy))
	out := armOut(t, user, room, "arbalest arm3")
	assert.Contains(t, out, "You aren't strong enough to handle Arbalest.")
	_, inPack := user.Character.FindInBackpack("arbalest")
	assert.True(t, inPack)
}

func TestEquipArm_CursedItemInTheNamedArmRefuses(t *testing.T) {
	user, room := armUser(t)
	user.Character.Equipment.Offhand = armItem(96010, "hexed dagger", items.Weapon, 1, true)
	require.True(t, user.Character.StoreItem(armItem(96011, "knife", items.Weapon, 1, false)))
	out := armOut(t, user, room, "knife arm2")
	assert.Contains(t, out, "Your Hexed Dagger is cursed and prevents you from removing it.")
	assert.Equal(t, 96010, user.Character.Equipment.Offhand.ItemId)
	_, inPack := user.Character.FindInBackpack("knife")
	assert.True(t, inPack)
}

func TestEquipArm_CursedTwoHandedPartnerRefuses(t *testing.T) {
	user, room := armUser(t)
	user.Character.Equipment.Weapon = armItem(96020, "hexed maul", items.Weapon, 2, true)
	require.True(t, user.Character.StoreItem(armItem(96021, "knife", items.Weapon, 1, false)))
	out := armOut(t, user, room, "knife arm2")
	assert.Contains(t, out, "Your Hexed Maul is cursed and prevents you from removing it.")
	assert.Equal(t, 96020, user.Character.Equipment.Weapon.ItemId)
}

func TestEquipArm_CursedSecondSlotUnderATwoHanderRefuses(t *testing.T) {
	user, room := armUser(t)
	user.Character.Equipment.Offhand = armItem(96030, "hexed buckler", items.Offhand, 1, true)
	require.True(t, user.Character.StoreItem(armItem(96031, "maul", items.Weapon, 2, false)))
	out := armOut(t, user, room, "maul arm1")
	assert.Contains(t, out, "Your Hexed Buckler is cursed and prevents you from removing it.")
}

func TestEquipArm_ShapeRefusalsKeepTheirWording(t *testing.T) {
	user, room := armUser(t)
	require.True(t, user.Character.StoreItem(armItem(96040, "cap", items.Head, 0, false)))
	require.True(t, user.Character.StoreItem(armItem(96041, "buckler", items.Offhand, 1, false)))
	require.True(t, user.Character.StoreItem(armItem(96042, "knife", items.Weapon, 1, false)))
	require.True(t, user.Character.StoreItem(armItem(96043, "maul", items.Weapon, 2, false)))
	assert.Contains(t, armOut(t, user, room, "cap arm1"), "You can only wield weapons or shields in arm slots.")
	assert.Contains(t, armOut(t, user, room, "buckler arm1"), "You can't put a shield in your primary weapon hand (arm 1).")
	assert.Contains(t, armOut(t, user, room, "knife arm3"), "You don't have arm 3.")
	assert.Contains(t, armOut(t, user, room, "maul arm2"), "A two-handed weapon needs a pair of arms. Try arm 1, 3, or 5.")
}

func TestEquipArm_SuccessLines(t *testing.T) {
	user, room := armUser(t)
	require.True(t, user.Character.StoreItem(armItem(96050, "knife", items.Weapon, 1, false)))
	require.True(t, user.Character.StoreItem(armItem(96051, "buckler", items.Offhand, 1, false)))
	assert.Contains(t, armOut(t, user, room, "knife arm1"), "You wield your")
	assert.Equal(t, 96050, user.Character.Equipment.Weapon.ItemId)
	out := armOut(t, user, room, "buckler arm2")
	assert.Contains(t, out, "You equip your")
	assert.Contains(t, out, "in your offhand.")
	assert.Equal(t, 96051, user.Character.Equipment.Offhand.ItemId)
}

// An item knocked off a full pack lands on the floor instead of vanishing.
func TestEquipArm_DisplacedItemOnAFullPackLandsOnTheFloor(t *testing.T) {
	user, room := armUser(t)
	anvil := armItem(96060, "anvil sword", items.Weapon, 1, false)
	anvil.Spec.Weight = 5000
	user.Character.Equipment.Weapon = anvil
	require.True(t, user.Character.StoreItem(armItem(96061, "knife", items.Weapon, 1, false)))
	user.Character.Stats.Strength.ValueAdj = 1
	armOut(t, user, room, "knife arm1")
	require.Equal(t, 96061, user.Character.Equipment.Weapon.ItemId)
	_, onFloor := room.FindOnFloor("anvil sword", false)
	assert.True(t, onFloor, "the anvil sword must be on the floor, not lost")
}
```

Add a reservation row modelled on `internal/characters/wear_reservation_test.go:25-60`: set `user.Character.StaminaMax.Base = 100`, wear a Neck item with `ReserveStaminaPct: 0.60` (seed it with `items.SeedItemsForTest` like that test and call `user.Character.Validate()`), store a buckler with `ReserveStaminaPct: 0.30`, run `buckler arm2`, and assert the output contains `reserve` (case-insensitive), the Offhand is empty and the buckler is back in the pack.

- [ ] **Step 4: Run to verify they fail**

Run: `go test ./internal/usercommands/ -run 'TestEquipArm_' -count=1`
Expected: FAIL (the arm branch skips MinStrength, prints "can't be removed!", loses the displaced item).

- [ ] **Step 5: `equipItem` and `EquipItemInArm`**

In `internal/actions/remove_equip.go`, add `characters` to the imports, add `ArmLabel string` to `EquipItemResult` with the comment `// ArmLabel is the hand the item went into ("offhand", "extra arm 1"), set only by EquipItemInArm.`, and replace `EquipItem` with:

```go
// EquipItem takes a named item from the actor's backpack and equips it.
// Displaced items (swapped-out gear) are stored back to the backpack, or
// dropped to the floor if the backpack is full: no item loss allowed.
// The reveal, Validate(), and EquipmentChange are all handled here.
// Messaging, condition onStart triggers, and quest-engine notifications
// remain in the callers.
func EquipItem(actor Actor, itemName string) EquipItemResult {
	return equipItem(actor, itemName, (*characters.Character).Wear)
}

// EquipItemInArm is `equip X armN` (spec ruling 11): EquipItem with the
// placement confined to arm N through Character.WearInArm, so the arm path
// meets every gate Wear has. It has one caller, the player's equip.
func EquipItemInArm(actor Actor, itemName string, arm int) EquipItemResult {
	res := equipItem(actor, itemName, func(c *characters.Character, i items.Item) ([]items.Item, bool, string) {
		return c.WearInArm(i, arm)
	})
	if res.Equipped {
		res.ArmLabel = actor.GetCharacter().ArmLabel(arm)
	}
	return res
}

func equipItem(actor Actor, itemName string, wear func(*characters.Character, items.Item) ([]items.Item, bool, string)) EquipItemResult {
	char := actor.GetCharacter()

	// Equippable-first, unfiltered fallback: mirrors usercommands/equip.go
	// so mob equips and gearup get the same preference.
	matchItem, found := char.FindInBackpackWhere(itemName, func(it items.Item) bool {
		spec := it.GetSpec()
		return spec.Type == items.Weapon || spec.Subtype == items.Wearable
	})
	if !found {
		matchItem, found = char.FindInBackpack(itemName)
	}
	if !found {
		return EquipItemResult{Found: false}
	}

	// Tentatively remove from backpack so the wear step can place it on the body.
	char.RemoveItem(matchItem)

	displaced, newItemWorn, failureReason := wear(char, matchItem)
	if !newItemWorn {
		// Put it back: equip failed.
		char.StoreItem(matchItem)
		return EquipItemResult{
			Item:          matchItem,
			Found:         true,
			Equipped:      false,
			FailureReason: failureReason,
		}
	}

	char.Awareness.TransitionToRevealing(state.TransitionReason{
		Trigger: awareness.TriggerForceVisible,
	})

	// Return displaced gear to backpack; drop to floor on overflow.
	for _, di := range displaced {
		if di.ItemId != 0 {
			if !char.StoreItem(di) {
				actor.GetRoom().AddItem(di, false)
			}
		}
	}

	char.Validate()

	events.AddToQueue(events.EquipmentChange{
		UserId:        actor.GetUserId(),
		MobInstanceId: actor.GetMobInstanceId(),
		ItemsWorn:     []items.Item{matchItem},
		ItemsRemoved:  displaced,
	})

	return EquipItemResult{
		Item:           matchItem,
		DisplacedItems: displaced,
		Found:          true,
		Equipped:       true,
	}
}
```

- [ ] **Step 6: The player's arm branch goes through it**

In `internal/usercommands/equip.go`:

1. Replace the comment above `beforeReservation` (`:113-118`) with: `// Snapshot reservation BEFORE anything touches the equipment set, so the disclosure below can tell the player when the thing they just put on set more of them aside.`
2. Delete the whole arm branch (`:121-273`, from `// Handle direct arm slot equip (arms 1-6)` to its closing `}`).
3. Replace `actor := ...` and `result := actions.EquipItem(actor, rest)` (`:275-277`) with:

```go
		// One shared body for both spellings; a named arm only confines where
		// the item goes (spec ruling 11), so the arm path meets Wear's
		// MinStrength, reservation and curse gates like any other equip.
		actor := &actions.UserActor{User: user, Room: room}
		var result actions.EquipItemResult
		if targetArmSlot > 0 {
			result = actions.EquipItemInArm(actor, rest, targetArmSlot)
		} else {
			result = actions.EquipItem(actor, rest)
		}
```

4. Replace the wear/wield line block (`:293-309`) with:

```go
			if result.ArmLabel != `` {
				if result.Item.GetSpec().Type == items.Offhand {
					user.SendText(messaging.CategorySystem, fmt.Sprintf(`You equip your <ansi fg="item">%s</ansi> in your %s.`, result.Item.DisplayName(), result.ArmLabel))
				} else {
					user.SendText(messaging.CategorySystem, fmt.Sprintf(`You wield your <ansi fg="item">%s</ansi> in your %s.`, result.Item.DisplayName(), result.ArmLabel))
				}
				room.SendTextVisual(messaging.CategoryEquipment,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> equips their <ansi fg="item">%s</ansi>.`, user.Character.Name, result.Item.DisplayName()),
					user.UserId,
				)
			} else if result.Item.GetSpec().Subtype == items.Wearable {
				user.SendText(messaging.CategorySystem,
					fmt.Sprintf(`You wear your <ansi fg="item">%s</ansi>.`, result.Item.DisplayName()),
				)
				room.SendTextVisual(messaging.CategoryEquipment,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> puts on their <ansi fg="item">%s</ansi>.`, user.Character.Name, result.Item.DisplayName()),
					user.UserId,
				)
			} else {
				user.SendText(messaging.CategorySystem,
					fmt.Sprintf(`You wield your <ansi fg="item">%s</ansi>. You're feeling dangerous.`, result.Item.DisplayName()),
				)
				room.SendTextVisual(messaging.CategoryEquipment,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> wields their <ansi fg="item">%s</ansi>.`, user.Character.Name, result.Item.DisplayName()),
					user.UserId,
				)
			}
```

5. Remove the now-unused `conditions` import. Confirm the file no longer names `GetHandPairs`, `HandsRequired`, `ItemPtr`, `IsCursed`, `CancelConditionsWithFlag` or `StoreItem`:

Run: `grep -n "GetHandPairs\|HandsRequired\|ItemPtr\|IsCursed\|CancelConditionsWithFlag\|StoreItem" internal/usercommands/equip.go`
Expected: no output (run it standalone; `grep` exits 1 on no match).

- [ ] **Step 7: Run the tests and the narration guard**

Run: `go test ./internal/actions/ ./internal/usercommands/ -count=1` then `go test . -run 'TestNarrationSitesMatchViewpointAudit|TestEveryTextSurfaceIsRegistered' -count=1`
Expected: PASS. If the narration guard reports a `usercommands/equip.go|...` key as stale and a new key as unregistered (the arm lines now sit inside `if result.ArmLabel != ""`), move that registry entry to the new key with its verdict and reason unchanged, in this commit. Never delete an entry to pass.

- [ ] **Step 8: Commit**

```bash
git add internal/actions/remove_equip.go internal/actions/sight_gates_parity_test.go internal/usercommands/equip.go internal/usercommands/equip_arm_test.go
git commit -m "feat(equip): equip X armN goes through Wear via actions.EquipItemInArm"
```

(Add `messaging_surface_guard_test.go` by name if Step 7 re-keyed it.)

---

### Task 6: The scorer weighs the swap `Wear` makes

**Model:** sonnet.

**Files:**
- Modify: `internal/itemvalue/delta.go:35-113,226-255,287-340`
- Modify: `internal/itemvalue/delta_test.go:25-146`
- Create: `internal/itemvalue/slot_agreement_test.go`
- Create: `internal/mobcommands/gearup_curse_test.go`

- [ ] **Step 1: Write the failing agreement and curse tests**

Create `internal/itemvalue/slot_agreement_test.go`:

```go
package itemvalue

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The scorer and Wear make one slot choice (spec rulings 10 and 12): for any
// arm count and any mix of cursed and plain gear, ItemValueDelta reports the
// slot Wear fills and the items Wear displaces, or both refuse.

var (
	agSword  = items.ItemSpec{Name: "sword", Type: items.Weapon, Subtype: items.Slashing, Hands: items.OneHanded}
	agGreat  = items.ItemSpec{Name: "greatsword", Type: items.Weapon, Subtype: items.Slashing, Hands: items.TwoHanded}
	agShield = items.ItemSpec{Name: "buckler", Type: items.Offhand, Subtype: items.Wearable, Hands: items.OneHanded}
	agRing   = items.ItemSpec{Name: "ring", Type: items.Ring, Subtype: items.Wearable}
	agBracer = items.ItemSpec{Name: "bracer", Type: items.Wrist, Subtype: items.Wearable}
)

func agItem(id int, spec items.ItemSpec, cursed bool) items.Item {
	s := spec
	s.ItemId, s.Cursed = id, cursed
	return items.Item{ItemId: id, Spec: &s}
}

func seedAgreementSpecies(t *testing.T) {
	t.Helper()
	t.Cleanup(species.SeedSpeciesForTest(map[int]*species.Species{
		0: {SpeciesId: 0, Name: "human", Size: species.Medium},
		1: {SpeciesId: 1, Name: "halfling", Size: species.Small},
		2: {SpeciesId: 2, Name: "ogre", Size: species.Large},
	}))
}

// agreementChar builds the same random loadout for the same seed.
func agreementChar(extra int, seed int64) *characters.Character {
	rng := rand.New(rand.NewSource(seed))
	c := characters.New()
	c.SpeciesId = rng.Intn(3)
	c.Mutations = map[string]int{}
	if extra > 0 {
		c.Mutations["extra-arms"] = extra
	}
	c.Validate()
	if rng.Intn(2) == 1 {
		c.SetSkill("weapon-combat", 1)
	}
	id := 1000
	put := func(p *items.Item, spec items.ItemSpec) {
		id++
		*p = agItem(id, spec, rng.Intn(3) == 0)
	}
	for _, p := range c.GetHandPairs() {
		switch rng.Intn(5) {
		case 0:
		case 1:
			put(p.First.ItemPtr, agSword)
		case 2:
			put(p.First.ItemPtr, agShield)
		case 3:
			if !p.IsHalfPair() {
				put(p.First.ItemPtr, agGreat)
				continue
			}
		case 4:
			put(p.First.ItemPtr, agSword)
		}
		if !p.IsHalfPair() && rng.Intn(2) == 1 {
			put(p.Second.ItemPtr, [...]items.ItemSpec{agSword, agShield}[rng.Intn(2)])
		}
	}
	for _, p := range []*items.Item{&c.Equipment.Ring, &c.Equipment.Ring2} {
		if rng.Intn(3) > 0 {
			put(p, agRing)
		}
	}
	wrists := []*items.Item{&c.Equipment.Wrist1, &c.Equipment.Wrist2, &c.Equipment.ExtraWrist1, &c.Equipment.ExtraWrist2, &c.Equipment.ExtraWrist3, &c.Equipment.ExtraWrist4}
	for _, p := range wrists[:2+extra] {
		if rng.Intn(3) > 0 {
			put(p, agBracer)
		}
	}
	return c
}

func agIds(list []items.Item) string {
	s := ""
	for _, it := range list {
		s += fmt.Sprintf("%d,", it.ItemId)
	}
	return s
}

func TestScorerAndWearAgreeOnEverySlotChoice(t *testing.T) {
	seedAgreementSpecies(t)
	rng := rand.New(rand.NewSource(20260929))
	checked, refused := 0, 0
	for _, extra := range []int{0, 1, 2, 4} {
		for n := 0; n < 1500; n++ {
			seed := rng.Int63()
			for _, cs := range []items.ItemSpec{agSword, agGreat, agShield, agRing, agBracer} {
				delta := ItemValueDelta(agreementChar(extra, seed), PhysicalBruiser, agItem(9000, cs, false))
				worn := agreementChar(extra, seed)
				ret, ok, why := worn.Wear(agItem(9000, cs, false))
				name := fmt.Sprintf("arms=%d seed=%d item=%s", 2+extra, seed, cs.Name)
				checked++
				if !ok {
					refused++
					assert.Equal(t, SlotName(""), delta.Slot, "%s: Wear refused (%q) but the scorer offered %s", name, why, delta.Slot)
					continue
				}
				placed := SlotName("")
				for _, s := range worn.Equipment.AllSlots() {
					if s.Item.ItemId == 9000 {
						placed = chooserSlotName[s.Key]
					}
				}
				require.Equal(t, placed, delta.Slot, "%s", name)
				assert.Equal(t, agIds(ret), agIds(delta.Displaced), "%s", name)
			}
		}
	}
	t.Logf("checked %d choices, %d refused by both", checked, refused)
	require.Greater(t, refused, 0, "no refusal was generated: the curse side is untested")
}

// With both rings full and nothing cursed the scorer weighs the swap Wear
// makes (Ring), even when Ring2 holds the weaker ring; with Ring cursed it
// weighs Ring2; with both cursed it offers nothing (spec testing, 5a ring).
func TestItemValueDelta_RingFollowsTheHelper(t *testing.T) {
	c := &characters.Character{Mutations: map[string]int{}}
	strong := agItem(1, agRing, false)
	strong.Spec.StatMods = map[string]int{"strength": 10}
	c.Equipment.Ring = strong
	c.Equipment.Ring2 = agItem(2, agRing, false)
	cand := agItem(3, agRing, false)
	cand.Spec.StatMods = map[string]int{"strength": 5}

	assert.Equal(t, SlotRing, ItemValueDelta(c, PhysicalBruiser, cand).Slot)

	c.Equipment.Ring.Spec.Cursed = true
	assert.Equal(t, SlotRing2, ItemValueDelta(c, PhysicalBruiser, cand).Slot)

	c.Equipment.Ring2.Spec.Cursed = true
	assert.Equal(t, SwapDelta{}, ItemValueDelta(c, PhysicalBruiser, cand))
	assert.False(t, IsUpgrade(c, PhysicalBruiser, cand))
}

// A single-slot swap a curse would refuse is never an upgrade (E8).
func TestItemValueDelta_SkipsACursedSingleSlot(t *testing.T) {
	c := &characters.Character{Mutations: map[string]int{}}
	c.Equipment.Head = agItem(1, items.ItemSpec{Name: "hexed helm", Type: items.Head, Subtype: items.Wearable}, true)
	cand := agItem(2, items.ItemSpec{Name: "fine cap", Type: items.Head, Subtype: items.Wearable, StatMods: map[string]int{"strength": 5}}, false)
	assert.Equal(t, SwapDelta{}, ItemValueDelta(c, PhysicalBruiser, cand))
}
```

`ItemSpec.StatMods` is `statmods.StatMods`, declared `map[string]int` (`internal/statmods/statmods.go:8`), so the `map[string]int{...}` literals above assign to it directly.

Create `internal/mobcommands/gearup_curse_test.go`:

```go
package mobcommands

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gearRing(id int, name string, cursed bool, str int) items.Item {
	return items.Item{ItemId: id, Spec: &items.ItemSpec{ItemId: id, Name: name, Type: items.Ring, Subtype: items.Wearable,
		Cursed: cursed, StatMods: map[string]int{"strength": str}}}
}

// A mob handed a better ring with a cursed first ring and a plain second
// wears it over the second; with both cursed it tries nothing and says
// nothing (spec "Companion", 5a).
func TestGearup_SkipsACursedRing(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	mob, room := getTestMobAndRoom(t)
	mob.Character.Equipment.Ring = gearRing(96101, "hexed band", true, 0)
	mob.Character.Equipment.Ring2 = gearRing(96102, "plain band", false, 0)
	gift := gearRing(96103, "gold band", false, 10)
	require.True(t, mob.Character.StoreItem(gift))

	events.DrainQueuedInputsForTest(mob.InstanceId)
	_, err := Gearup(fmt.Sprintf("!%d", gift.ItemId), mob, room)
	require.NoError(t, err)
	assert.Contains(t, events.DrainQueuedInputsForTest(mob.InstanceId), fmt.Sprintf("wear !%d", gift.ItemId))

	_, err = Equip(fmt.Sprintf("!%d", gift.ItemId), mob, room)
	require.NoError(t, err)
	assert.Equal(t, 96103, mob.Character.Equipment.Ring2.ItemId, "the gift goes over the plain ring")
	assert.Equal(t, 96101, mob.Character.Equipment.Ring.ItemId, "the cursed ring stays on")

	mob.Character.Equipment.Ring2 = gearRing(96104, "second hex", true, 0)
	gift2 := gearRing(96105, "silver band", false, 10)
	require.True(t, mob.Character.StoreItem(gift2))
	events.DrainQueuedInputsForTest(mob.InstanceId)
	_, err = Gearup(fmt.Sprintf("!%d", gift2.ItemId), mob, room)
	require.NoError(t, err)
	assert.Empty(t, events.DrainQueuedInputsForTest(mob.InstanceId), "no upgrade when every ring is cursed")
}
```

(`DrainQueuedInputsForTest` returns each queued `Input.InputText` for the instance, `events.go:314-331`; `FindInBackpack` resolves `!<id>`, `items.go:694`. `CanEquipFromGive` refuses a species whose `DisabledSlots` holds `Weapon`: if the fixture mob's species 1 is seeded that way, set `mob.Character.SpeciesId` to an unregistered id for this test.)

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/itemvalue/ ./internal/mobcommands/ -run 'Agree|RingFollows|SkipsACursed|TestGearup_Skips' -count=1`
Expected: FAIL (build: `chooserSlotName` undefined; and the scorer still offers `Ring2`/`Offhand` choices of its own).

- [ ] **Step 3: Route the four chooser types through the helper**

In `internal/itemvalue/delta.go`:

1. Delete `extraArmsLevel` (`:35-46`); its only caller was the wrist case.
2. Add after `canonicalSlotOrder`:

```go
// chooserSlotName maps an AllSlots key to its SlotName, for the four item
// types whose slot Character.ChooseWornSlot picks.
var chooserSlotName = map[string]SlotName{
	`weapon`: SlotWeapon, `offhand`: SlotOffhand,
	`extraarm1`: SlotExtraArm1, `extraarm2`: SlotExtraArm2, `extraarm3`: SlotExtraArm3, `extraarm4`: SlotExtraArm4,
	`wrist1`: SlotWrist1, `wrist2`: SlotWrist2,
	`extrawrist1`: SlotExtraWrist1, `extrawrist2`: SlotExtraWrist2, `extrawrist3`: SlotExtraWrist3, `extrawrist4`: SlotExtraWrist4,
	`ring`: SlotRing, `ring2`: SlotRing2,
}

// usesChooser is true for the item types whose slot Wear picks through
// Character.ChooseWornSlot (slice 5a): the scorer asks the same helper, so a
// scored upgrade is exactly the swap Wear makes, for any arm count.
func usesChooser(t items.ItemType) bool {
	return t == items.Weapon || t == items.Offhand || t == items.Ring || t == items.Wrist
}
```

3. Change `compatibleSlotsFor` to take the candidate item and consult the helper first:

```go
// compatibleSlotsFor returns the list of SlotNames where the candidate could
// be placed. For weapons, shields, rings and wrists that is the one slot
// Character.ChooseWornSlot picks (nothing when it refuses); for every other
// type, the type's own slot. Returns nil when the item is not equippable.
func compatibleSlotsFor(candidate items.Item, char *characters.Character) []SlotName {
	spec := candidate.GetSpec()
	if usesChooser(spec.Type) {
		choice, refusal := char.ChooseWornSlot(candidate, 0)
		if refusal != `` || len(choice.Slots) == 0 {
			return nil
		}
		return []SlotName{chooserSlotName[choice.Slots[0].Key]}
	}
	switch spec.Type {
	case items.Head:
		return []SlotName{SlotHead}
	case items.Neck:
		return []SlotName{SlotNeck}
	case items.Shoulders:
		return []SlotName{SlotShoulders}
	case items.Body:
		return []SlotName{SlotBody}
	case items.Back:
		return []SlotName{SlotBack}
	case items.Belt:
		return []SlotName{SlotBelt}
	case items.Gloves:
		return []SlotName{SlotGloves}
	case items.Legs:
		return []SlotName{SlotLegs}
	case items.Feet:
		return []SlotName{SlotFeet}
	case items.Tail:
		if hasTailMutation(char) {
			return []SlotName{SlotTail}
		}
		return nil
	case items.ComponentBag:
		return []SlotName{SlotComponentBag}
	case items.Light:
		return []SlotName{SlotLight}
	}
	return nil
}
```

4. Replace `displacedItemsForSlot`:

```go
// displacedItemsForSlot returns the items that come off when candidate is
// placed at targetSlot: the helper's Displaced for the chooser types (so a
// two-hander, a stray behind one, or an extra arm is modelled exactly as
// Wear does it), else the slot's own occupant.
func displacedItemsForSlot(char *characters.Character, targetSlot SlotName, candidate items.Item) []items.Item {
	if usesChooser(candidate.GetSpec().Type) {
		choice, _ := char.ChooseWornSlot(candidate, 0)
		return choice.Displaced
	}
	current := itemInSlot(targetSlot, char)
	if current.ItemId > 0 {
		return []items.Item{current}
	}
	return nil
}
```

5. In `ItemValueDelta`, change `slots := compatibleSlotsFor(candidateSpec, char)` to `slots := compatibleSlotsFor(candidate, char)`, and in the loop replace `displaced := displacedItemsForSlot(char, slot, candidateSpec)` with:

```go
		displaced := displacedItemsForSlot(char, slot, candidate)
		// A swap a curse would refuse is no upgrade (slice 5a, E8): Wear
		// will not make it. The chooser types never get here with one,
		// because the helper already skipped or refused the slot.
		if cursedAmong(char, displaced) {
			continue
		}
```

and add:

```go
func cursedAmong(char *characters.Character, displaced []items.Item) bool {
	for _, d := range displaced {
		if char.CursedRefusal(d) != `` {
			return true
		}
	}
	return false
}
```

Update the `ItemValueDelta` doc: "Smart slot selection" becomes "the slot Wear itself would pick for weapons, shields, rings and wrists".

- [ ] **Step 4: Bring the old unit tests to the new contract**

In `internal/itemvalue/delta_test.go`:
- Every `compatibleSlotsFor(spec, c)` becomes `compatibleSlotsFor(items.Item{ItemId: 1, Spec: &spec}, c)`.
- `TestCompatibleSlotsFor_OneHandedWeapon` expects `[]SlotName{SlotWeapon}` (empty hands: Wear fills the main hand). Rename it `TestCompatibleSlotsFor_OneHandedWeaponIsTheSlotWearFills`.
- `TestCompatibleSlotsFor_Ring` expects `[]SlotName{SlotRing}`.
- `TestCompatibleSlotsFor_WristWithExtraArms`: set `char.ExtraArms = 2` (the helper reads the field `Validate` derives, not the mutation map), fill `Wrist1` and `Wrist2` with `items.Item{ItemId: 5}` and `{ItemId: 6}`, expect `[]SlotName{SlotExtraWrist1}`. `TestCompatibleSlotsFor_WristWithMaxExtraArms`: `char.ExtraArms = 4`, fill `Wrist1`, `Wrist2`, `ExtraWrist1` to `ExtraWrist3`, expect `[]SlotName{SlotExtraWrist4}`.
- Every `displacedItemsForSlot(char, slot, spec)` becomes `displacedItemsForSlot(char, slot, items.Item{ItemId: 99, Spec: &spec})`.
- Replace the skipped `TestDisplacedItemsForSlot_OffhandWithTwoHandedWeapon` with a real test: `char.Equipment.Weapon = items.Item{ItemId: 1, Spec: &items.ItemSpec{ItemId: 1, Type: items.Weapon, Hands: items.TwoHanded}}`, candidate a one-handed sword spec, expect exactly `[Weapon]` back (the helper swaps the two-hander in the main hand).

- [ ] **Step 5: Run the packages that score**

Run: `go test ./internal/itemvalue/ ./internal/mobcommands/ ./internal/hooks/ ./internal/planners/ ./internal/mobs/ -count=1`
Expected: PASS. A pre-existing test that expected the scorer to pick `Offhand` or `Ring2` for a non-dual-wielder or a weaker second ring is asserting the E12/E17 mismatch this slice removes: update its expectation to the helper's slot and say so in the commit message.

- [ ] **Step 6: Commit**

```bash
git add internal/itemvalue/delta.go internal/itemvalue/delta_test.go internal/itemvalue/slot_agreement_test.go internal/mobcommands/gearup_curse_test.go
git commit -m "feat(itemvalue): score the swap Wear makes, through ChooseWornSlot, and skip cursed swaps"
```

---

### Task 7: Get: `TooDarkToGet`, `TakeFloorItem`, gold

**Model:** sonnet.

**Files:**
- Modify: `internal/actions/get.go`
- Modify: `internal/usercommands/get.go:20-97,598-692`
- Modify: `internal/mobcommands/get.go:43-57`
- Modify: `internal/actions/sight_gates_parity_test.go`
- Create: `internal/usercommands/get_sight_gates_test.go`, `internal/mobcommands/get_sight_gates_test.go`

- [ ] **Step 1: Write the failing parity rows**

Append to `internal/actions/sight_gates_parity_test.go`:

```go
func TestGateParity_TakeFloorItem(t *testing.T) {
	want := map[gateLight]error{gateLit: nil, gateShapes: nil, gateDark: ErrTooDark, gateBlinded: ErrTooDark}
	for _, light := range gateLights {
		for _, who := range gateWho {
			s := newGateScene(t, light)
			a := s.actor(who)
			pebble := gateItem(39501, "pebble", items.Junk, false)
			s.room.Items = []items.Item{pebble}
			err := TakeFloorItem(a, pebble, false)
			assert.ErrorIs(t, err, want[light], "%s at %s", who, light)
			if want[light] == nil {
				assert.NoError(t, err, "%s at %s", who, light)
			}
			_, onFloor := s.room.FindOnFloor("pebble", false)
			assert.Equal(t, want[light] != nil, onFloor, "%s at %s: a refusal moves nothing", who, light)
		}
	}
}

func TestGateParity_ExplodingItemIsRefused(t *testing.T) {
	for _, who := range gateWho {
		s := newGateScene(t, gateLit)
		bomb := gateItem(39502, "bomb", items.Junk, false)
		bomb.Adjectives = []string{`exploding`}
		s.room.Items = []items.Item{bomb}
		assert.ErrorIs(t, TakeFloorItem(s.actor(who), bomb, false), ErrExploding, who)
		res := GetItemFromFloor(s.actor(who), "bomb", false)
		assert.True(t, res.Found, who)
		assert.ErrorIs(t, res.Err, ErrExploding, who)
	}
}

func TestGateParity_GetItemFromFloorInTheDarkFindsNothing(t *testing.T) {
	for _, who := range gateWho {
		s := newGateScene(t, gateDark)
		s.room.Items = []items.Item{gateItem(39503, "pebble", items.Junk, false)}
		res := GetItemFromFloor(s.actor(who), "pebble", false)
		assert.False(t, res.Found, "%s: the dark tells the actor nothing about the floor", who)
		assert.ErrorIs(t, res.Err, ErrTooDark, who)
	}
}

func TestGateParity_GoldPickup(t *testing.T) {
	for _, light := range gateLights {
		for _, who := range gateWho {
			s := newGateScene(t, light)
			s.room.Gold = 10
			err := GetGoldFromFloor(s.actor(who), 10)
			if light == gateDark || light == gateBlinded {
				assert.ErrorIs(t, err, ErrTooDark, "%s at %s", who, light)
				assert.Equal(t, 10, s.room.Gold, "%s at %s", who, light)
				continue
			}
			assert.NoError(t, err, "%s at %s", who, light)
			assert.Equal(t, 0, s.room.Gold, "%s at %s", who, light)
		}
	}
}
```

(`items.Junk` must be a real `ItemType`; confirm with `grep -n "Junk\s*ItemType\|Junk *=" internal/items/itemspec.go` and use whichever plain non-wearable type exists.)

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/actions/ -run 'TestGateParity_(Take|Exploding|GetItem|Gold)' -count=1`
Expected: build failure, `TakeFloorItem`, `ErrTooDark`, `ErrExploding` undefined.

- [ ] **Step 3: The shared get gates**

Replace the body of `internal/actions/get.go` after the `ErrHouseholdBauble` declaration with:

```go
// ErrTooDark refuses a pickup by an actor who sees nothing at all here
// (slice 5a). Shapes are enough to grope for an item, so only SightNone
// refuses.
var ErrTooDark = errors.New(`too dark to find anything`)

// ErrExploding refuses an item that is about to explode; a sweep stops on it.
var ErrExploding = errors.New(`it is about to explode`)

// TooDarkToGet is the one statement of the pickup sight rule, for both
// actors. The player's `get` also asks it first, so its container, corpse
// and bag branches stay refused in the dark.
func TooDarkToGet(actor Actor) bool {
	return messaging.ParticipantSight(actor.GetCharacter(), actor.GetRoom()) == messaging.SightNone
}

// GetItemResult is the result of a GetItemFromFloor call.
type GetItemResult struct {
	Item  items.Item
	Found bool
	Err   error
}

// TakeFloorItem moves an item already found on the floor (or in the stash)
// into the actor's backpack, through every pickup gate in the player's order:
// ErrTooDark, ErrExploding, ErrHouseholdBauble, then the transfer (which
// fires ItemOwnership, or rolls back on a full pack).
func TakeFloorItem(actor Actor, item items.Item, stash bool) error {
	if TooDarkToGet(actor) {
		return ErrTooDark
	}
	if item.HasAdjective(`exploding`) {
		return ErrExploding
	}
	room := actor.GetRoom()
	// A household's bauble is never picked up (it is only ever on the floor,
	// never in a stash).
	if !stash && item.BaubleBelongsTo(room.RoomId) {
		return ErrHouseholdBauble
	}
	return TransferItemToBackpack(
		item,
		actor.GetCharacter(),
		actor.GetUserId(),
		actor.GetMobInstanceId(),
		func(i items.Item) { room.RemoveItem(i, stash) },
		func(i items.Item) { room.AddItem(i, stash) },
	)
}

// GetItemFromFloor searches the room floor (or stash) for an item matching
// itemName and takes it through TakeFloorItem. In the dark it finds nothing
// (Found false, ErrTooDark): the actor learns nothing about the floor. Every
// other refusal returns the item found, Found, and the gate's error.
func GetItemFromFloor(actor Actor, itemName string, stash bool) GetItemResult {
	if TooDarkToGet(actor) {
		return GetItemResult{Found: false, Err: ErrTooDark}
	}
	matchItem, found := actor.GetRoom().FindOnFloor(itemName, stash)
	if !found {
		return GetItemResult{Found: false}
	}
	return GetItemResult{Item: matchItem, Found: true, Err: TakeFloorItem(actor, matchItem, stash)}
}

// GetGoldFromFloor moves gold on the room floor into the actor's wallet
// (FloorPickupGold validates the amount); refused with ErrTooDark when the
// actor sees nothing.
func GetGoldFromFloor(actor Actor, amount int) error {
	if TooDarkToGet(actor) {
		return ErrTooDark
	}
	return FloorPickupGold(amount, actor.GetRoom(), actor.GetCharacter())
}
```

Add `messaging` to the imports.

- [ ] **Step 4: Write the failing wrapper tests**

Create `internal/usercommands/get_sight_gates_test.go`:

```go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The player's get keeps today's lines through the shared gates (slice 5a).

func TestGet_ExplodingItemKeepsItsLine(t *testing.T) {
	user, room := seedDarknessGateRoom(t, 90)
	bomb := items.Item{ItemId: 96201, Spec: &items.ItemSpec{ItemId: 96201, Name: "bomb"}, Adjectives: []string{`exploding`}}
	room.Items = append(room.Items, bomb)
	out := runGate(t, user, func() (bool, error) { return Get("bomb", user, room, 0) })
	assert.Contains(t, out, "You can't pick that up, it's about to explode!")
	_, onFloor := room.FindOnFloor("bomb", false)
	assert.True(t, onFloor)
}

func TestGet_SweepStopsSilentlyOnAnExplodingItem(t *testing.T) {
	user, room := seedDarknessGateRoom(t, 90)
	room.Items = append(room.Items,
		items.Item{ItemId: 96202, Spec: &items.ItemSpec{ItemId: 96202, Name: "stone"}},
		items.Item{ItemId: 96203, Spec: &items.ItemSpec{ItemId: 96203, Name: "stone"}, Adjectives: []string{`exploding`}},
	)
	out := runGate(t, user, func() (bool, error) { return Get("all stone", user, room, 0) })
	assert.Contains(t, out, "You pick up 1 item(s).")
	_, carried := user.Character.FindInBackpack("stone")
	require.True(t, carried)
}

func TestGet_DarkRefusalUnchanged(t *testing.T) {
	user, room := seedDarknessGateRoom(t, 0)
	out := runGate(t, user, func() (bool, error) { return Get("all", user, room, 0) })
	assert.Contains(t, out, "You can't see anything to pick up!")
}
```

(`seedDarknessGateRoom` and `runGate` are in `darkness_gates_sight_test.go`; `FindOnFloor`'s name matcher orders two same-named stones by list position, so the plain one is taken first.)

Create `internal/mobcommands/get_sight_gates_test.go`:

```go
package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
)

// A mob in the dark picks up nothing, gold included, and stays hidden.
func TestMobGet_DarkRefusesSilently(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	mob, room := getTestMobAndRoom(t)
	room.SkyLight, room.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(0)
	room.Items = append(room.Items, items.Item{ItemId: 96210, Spec: &items.ItemSpec{ItemId: 96210, Name: "pebble"}})
	room.Gold = 7
	Get("pebble", mob, room)
	Get("gold", mob, room)
	_, carried := mob.Character.FindInBackpack("pebble")
	assert.False(t, carried)
	assert.Equal(t, 7, room.Gold)
}

func TestMobGet_ExplodingItemRefused(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	mob, room := getTestMobAndRoom(t)
	room.Lamp = rooms.LampPtr(90)
	room.Items = append(room.Items, items.Item{ItemId: 96211, Spec: &items.ItemSpec{ItemId: 96211, Name: "bomb"}, Adjectives: []string{`exploding`}})
	Get("bomb", mob, room)
	_, carried := mob.Character.FindInBackpack("bomb")
	assert.False(t, carried)
}
```

- [ ] **Step 5: The player wrapper**

In `internal/usercommands/get.go`:

1. First statement of `Get` (`:91-97`) becomes:

```go
	// Can't pick things up if you can't see anything at all (the shared
	// rule, actions.TooDarkToGet). Asked here, first, so the container,
	// corpse and bag branches below stay refused in the dark too.
	if actions.TooDarkToGet(&actions.UserActor{User: user, Room: room}) {
		user.SendText(messaging.CategorySystem, "You can't see anything to pick up!")
		return true, nil
	}
```

2. In `getAllMatchingFromFloor`, replace the loop body from `if matchItem.HasAdjective(...)` through the `else { ...overloaded...; break }` (`:52-73`) with:

```go
		err := actions.TakeFloorItem(&actions.UserActor{User: user, Room: room}, matchItem, false)
		if errors.Is(err, actions.ErrExploding) || errors.Is(err, actions.ErrTooDark) {
			break
		}
		user.Character.CancelConditionsWithFlag(conditions.Hidden)
		if err != nil {
			user.SendText(messaging.CategorySystem,
				fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi> - you're already overloaded!`, matchItem.DisplayName()),
			)
			break
		}
		picked++
```

and update the function's comment: "Stops early on an exploding item (never swept up by accident; `actions.TakeFloorItem` refuses it)". A full pack now rolls the item back to the END of the floor list (the transfer removes first); the sweep stops there either way.

3. Replace the single-get block (`:633-665`) with:

```go
		// First try the requested source (floor or stash). The shared pickup
		// runs every gate: dark, exploding, a household's bauble, capacity.
		{
			result := actions.GetItemFromFloor(&actions.UserActor{User: user, Room: room}, rest, getFromStash)
			if result.Found {
				matchItem = result.Item
				found = true
				switch {
				case errors.Is(result.Err, actions.ErrExploding):
					user.SendText(messaging.CategorySystem, `You can't pick that up, it's about to explode!`)
					return true, nil
				case errors.Is(result.Err, actions.ErrHouseholdBauble):
					// A bauble found in this household belongs to it.
					// `get` never commits a crime: the shared pickup
					// refuses it for every taker, and this names the
					// steal command, which is the theft.
					user.SendText(messaging.CategorySystem, fmt.Sprintf(
						`The <ansi fg="itemname">%s</ansi> belongs to this household. To take it anyway, <ansi fg="command">steal %s</ansi>.`,
						matchItem.DisplayName(), stealWord(matchItem)))
					return true, nil
				case result.Err != nil:
					// Capacity exceeded: item was rolled back to floor
					user.SendText(messaging.CategorySystem,
						fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi> - you're already overloaded!`, matchItem.DisplayName()),
					)
					return true, nil
				}
			}
		}
```

4. In the stash auto-detect block (`:672-681`), before the capacity branch add the same `errors.Is(result.Err, actions.ErrExploding)` case with today's exploding line.

- [ ] **Step 6: The mob wrapper**

In `internal/mobcommands/get.go`, the gold branch (`:45-54`) becomes:

```go
		if room.Gold > 0 {
			actor := &actions.MobActor{Mob: mob, Room: room}
			goldAmt := room.Gold
			if err := actions.GetGoldFromFloor(actor, goldAmt); err == nil {
				// Revealed by the pickup, not by trying one in the dark.
				mob.Character.CancelConditionsWithFlag(conditions.Hidden)
				room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> picks up <ansi fg="gold">%d gold</ansi>.`, mob.Character.Name, goldAmt))
			}
		}
```

The item path already speaks only on `result.Found && result.Err == nil`; every refusal stays silent.

- [ ] **Step 7: Run**

Run: `go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ -count=1` then `go test . -run 'TestNarrationSitesMatchViewpointAudit|TestFinderViewReachesOnlyItsReader' -count=1`
Expected: PASS. An older test that calls `GetItemFromFloor`, `GetGoldFromFloor` or a `Get` wrapper in an unlit fixture room now meets the sight gate: pin that fixture's light (`room.Lamp = rooms.LampPtr(90)`), never weaken the gate, and name the file in the commit message.

- [ ] **Step 8: Commit**

```bash
git add internal/actions/get.go internal/actions/sight_gates_parity_test.go internal/usercommands/get.go internal/usercommands/get_sight_gates_test.go internal/mobcommands/get.go internal/mobcommands/get_sight_gates_test.go
git commit -m "feat(get): one pickup body for both actors, refusing the dark and exploding items"
```

---

### Task 8: Look: `actions.ResolveLook`

**Model:** sonnet.

**Files:**
- Create: `internal/actions/look.go`
- Modify: `internal/actions/sight_gates_parity_test.go`
- Modify: `internal/usercommands/look.go:24-305,441-458`
- Modify: `internal/mobcommands/look.go`
- Create: `internal/mobcommands/look_sight_gates_test.go`
- Modify: `lookup_viewer_guard_test.go:49,71`

- [ ] **Step 1: Write the failing parity rows**

Append to `internal/actions/sight_gates_parity_test.go` (add the `exit` import `github.com/GoMudEngine/GoMud/internal/exit`):

```go
func TestGateParity_ResolveLook(t *testing.T) {
	// Each actor looks at the other one.
	other := map[string]string{"player": "gatemob", "mob": "gatey"}
	for _, light := range gateLights {
		for _, who := range gateWho {
			s := newGateScene(t, light)
			room := ResolveLook(s.actor(who), "")
			creature := ResolveLook(s.actor(who), other[who])
			switch light {
			case gateLit:
				assert.Equal(t, LookRoom, room.Kind, who)
				assert.Equal(t, LookCreature, creature.Kind, who)
			case gateShapes:
				assert.Equal(t, LookRoom, room.Kind, who)
				assert.Equal(t, LookOther, creature.Kind, "%s: at shapes a creature is not named", who)
				assert.False(t, creature.NamesCreatures, who)
			default:
				assert.Equal(t, LookDark, room.Kind, "%s at %s", who, light)
				assert.Equal(t, LookDark, creature.Kind, "%s at %s", who, light)
			}
		}
	}
}

func TestGateParity_LookCannotNameAHiddenCreature(t *testing.T) {
	other := map[string]string{"player": "gatemob", "mob": "gatey"}
	for _, who := range gateWho {
		s := newGateScene(t, gateLit)
		if who == "player" {
			viewerTestHide(t, &s.mob.Character)
		} else {
			viewerTestHide(t, s.user.Character)
		}
		assert.NotEqual(t, LookCreature, ResolveLook(s.actor(who), other[who]).Kind, who)
	}
}

func TestGateParity_LookThroughAnExitNeedsLight(t *testing.T) {
	for _, light := range []gateLight{gateLit, gateShapes} {
		for _, who := range gateWho {
			s := newGateScene(t, light)
			s.room.Exits = map[string]exit.RoomExit{"north": {RoomId: gateRoomId + 1}}
			res := ResolveLook(s.actor(who), "north")
			if light == gateLit {
				assert.Equal(t, LookExit, res.Kind, who)
				assert.Equal(t, gateRoomId+1, res.ExitRoomId, who)
				continue
			}
			assert.Equal(t, LookExitTooDark, res.Kind, "%s: heat shows shapes here, not in the next room", who)
		}
	}
}
```

(If `ResolveTargetActor` cannot find the mob by `"gatemob"` because it reads `mobs.GetInstance` names differently, check `target_viewer_test.go:24-40` for the fixture shape it expects and match it.)

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/actions/ -run 'TestGateParity_(ResolveLook|Look)' -count=1`
Expected: build failure, `ResolveLook` undefined.

- [ ] **Step 3: Write `ResolveLook`**

Create `internal/actions/look.go`:

```go
package actions

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/messaging"
)

// LookKind is what a look resolved to.
type LookKind int

const (
	LookDark        LookKind = iota // the looker sees nothing at all here
	LookRoom                        // no target: the room itself
	LookCreature                    // a creature the looker perceives, at clear sight
	LookExit                        // an exit the looker can see through
	LookExitTooDark                 // an exit, but too dark to see through it
	LookExitLocked                  // an exit that is locked
	LookOther                       // anything else: each wrapper's own objects, in its own order
)

// LookResolution carries every sight rule of both looks (slice 5a).
type LookResolution struct {
	Kind  LookKind
	Sight messaging.SightDecision
	// NamesCreatures is Sight == SightFull: only then is a creature or a pet
	// named, so a shapes-only looker cannot confirm who is standing there.
	NamesCreatures bool
	Target         Actor  // LookCreature
	LookAt         string // the target, after a direction alias resolved to an exit
	ExitName       string // the LookExit kinds
	ExitRoomId     int    // LookExit
	// PetUserId is the owner of a pet the looker may name here (clear sight
	// only), 0 otherwise. It is not a kind because the player resolves the
	// pet AFTER carried items and room nouns; each wrapper reads it at its
	// own pet step.
	PetUserId int
}

// ResolveLook is the shared look resolution for players and mobs. Order is
// the player's: sight, no target, creature (resolved with the looker as
// viewer, so a creature it does not perceive cannot be named), then a sealed
// crate or a known container defers to the wrapper (LookOther) before any
// exit does, then the exit (with direction aliases), through-sight, lock.
// It has no side effects.
func ResolveLook(actor Actor, lookAt string) LookResolution {
	char := actor.GetCharacter()
	room := actor.GetRoom()
	res := LookResolution{Sight: messaging.ParticipantSight(char, room), LookAt: lookAt}
	res.NamesCreatures = res.Sight == messaging.SightFull

	if res.Sight == messaging.SightNone {
		res.Kind = LookDark
		return res
	}
	if lookAt == `` {
		res.Kind = LookRoom
		return res
	}

	if res.NamesCreatures {
		if target, err := ResolveTargetActor(room, lookAt, ResolveTargetOptions{Viewer: char}); err == nil {
			res.Kind, res.Target = LookCreature, target
			return res
		}
		res.PetUserId = room.FindByPetName(lookAt)
		if res.PetUserId == 0 && lookAt == `pet` && actor.IsPlayer() && char.Pet.Exists() {
			res.PetUserId = actor.GetUserId()
		}
	}

	if lookNamesAnObject(actor, lookAt) {
		res.Kind = LookOther
		return res
	}

	exitName, exitRoomId := room.FindExitByName(lookAt)
	if exitName == `` {
		if alias := keywords.TryDirectionAlias(lookAt); alias != lookAt {
			if exitName, exitRoomId = room.FindExitByName(alias); exitName != `` {
				res.LookAt = alias
			}
		}
	}
	if exitName == `` {
		res.Kind = LookOther
		return res
	}
	res.ExitName, res.ExitRoomId = exitName, exitRoomId

	// Seeing THROUGH an exit needs more light than seeing the room you are
	// standing in (messaging.SeesThroughExit; infra reach does not help).
	if !messaging.SeesThroughExit(char, room) {
		res.Kind = LookExitTooDark
		return res
	}
	if info, _ := room.GetExitInfo(exitName); info.Lock.IsLocked() {
		res.Kind = LookExitLocked
		return res
	}
	res.Kind = LookExit
	return res
}

// lookNamesAnObject reports a sealed crate or a room container the looker
// knows of: the player resolves those between the creature and the exit.
func lookNamesAnObject(actor Actor, lookAt string) bool {
	room := actor.GetRoom()
	if room.MatchesSealedCrate(strings.ToLower(lookAt)) {
		return true
	}
	name := room.FindContainerByName(lookAt)
	if name == `` {
		return false
	}
	if c, ok := room.Containers[name]; ok && c.Hidden {
		return actor.GetCharacter().HasDiscovery(room.RoomId, name)
	}
	return true
}
```

- [ ] **Step 4: Re-key the lookup guard**

In `lookup_viewer_guard_test.go`: delete the `internal/mobcommands/look.go|Look` and `internal/usercommands/look.go|Look` entries and add, sorted:

```go
	"internal/actions/look.go|ResolveLook":                             {viewer: 1},
```

- [ ] **Step 5: The player wrapper**

In `internal/usercommands/look.go`, restructure `Look`'s head (`:24-305`) so it reads:

```go
func Look(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	secretLook := flags.Has(events.CmdSecretly)

	isSneaking := user.Character.IsHidden()

	// trim off some fluff
	if len(rest) > 2 {
		if rest[0:3] == `at ` {
			rest = rest[3:]
		}
	}
	if len(rest) > 3 {
		if rest[0:4] == `the ` {
			rest = rest[4:]
		}
	}

	lookAt := rest

	// Every sight rule of look lives in actions.ResolveLook, shared with the
	// mob look (slice 5a): the no-sight refusal, the creature named only at
	// clear sight and only if perceived, the exit's through-sight and lock,
	// the pet at clear sight. This function only words the answer.
	res := actions.ResolveLook(&actions.UserActor{User: user, Room: room}, lookAt)
	if res.Kind == actions.LookDark {
		user.SendText(messaging.CategorySystem, `You can't see anything!`)
		return true, nil
	}
	sight := res.Sight

	events.AddToQueue(events.Looking{
		UserId: user.UserId,
		RoomId: room.RoomId,
		Target: lookAt,
		Hidden: isSneaking,
	})

	switch res.Kind {
	case actions.LookRoom:
		// (today's `if len(lookAt) == 0 { ... }` body, unchanged)
	case actions.LookCreature:
		target := res.Target
		// (today's creature body from `if target.IsPlayer() {` to its
		// `return true, nil`, unchanged)
	case actions.LookExitTooDark:
		user.SendText(messaging.CategorySystem, `It's too dark to see anything in that direction.`)
		return true, nil
	case actions.LookExitLocked:
		user.SendText(messaging.CategorySystem, fmt.Sprintf("The %s exit is locked.", res.ExitName))
		return true, nil
	case actions.LookExit:
		user.SendText(messaging.CategorySystem, fmt.Sprintf("You peer toward the %s.", res.ExitName))
		if !isSneaking {
			// (today's peers-toward room line with exitName → res.ExitName)
		}
		lookRoom(user, res.ExitRoomId, secretLook || isSneaking)
		return true, nil
	}
```

Fill each parenthetical with the moved block verbatim (they are moves, not rewrites; the parentheticals must not remain). Then:
- Keep the sealed-crate and container branches where they are (they now run after the `switch`, on `LookOther`).
- Delete the exit block (`:251-305`) except the direction-alias no-exit check (`:307-312`), which stays next.
- In the pet branch, replace `petUserId := room.FindByPetName(rest)` through `if petUserId > 0 && sight == messaging.SightFull {` with `if petUserId := res.PetUserId; petUserId > 0 {`.
- `sight` stays for the shapes hint at the end.
- Remove the now-unused `keywords` import only if nothing else in the file uses it (the alias no-exit check does, so it stays).

Keep every branch inside `Look` (the bauble finder-view guard counts four viewer-accessor calls in `usercommands/look.go|Look`; do not move them into helpers).

Confirm: `grep -n "ParticipantSight\|SeesThroughExit\|CanSeeClearly\|ResolveTargetActor" internal/usercommands/look.go` prints nothing (run it standalone).

- [ ] **Step 6: Write the failing mob-look tests**

Create `internal/mobcommands/look_sight_gates_test.go`:

```go
package mobcommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

func hideForLook(t *testing.T, u *users.UserRecord) {
	t.Helper()
	reason := state.TransitionReason{Trigger: "look_sight_gates_test"}
	require.NoError(t, u.Character.Awareness.TransitionToConcealing(awareness.ConcealingData{}, reason))
	u.Character.Awareness.ResolveConcealment(true, reason)
	require.True(t, u.Character.IsHidden())
}

// A mob cannot name a hidden player: no "is looking at you", no room line
// (closes spec L6).
func TestMobLook_CannotNameAHiddenPlayer(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	mob, room := getTestMobAndRoom(t)
	room.Lamp = rooms.LampPtr(90)
	alice, bob := users.GetByUserId(1), users.GetByUserId(2)
	hideForLook(t, alice)
	events.DrainQueuedMessagesForTest(1)
	events.DrainQueuedMessagesForTest(2)
	Look("aliceia", mob, room)
	require.NotContains(t, strings.Join(events.DrainQueuedMessagesForTest(1), "\n"), "is looking at you")
	require.NotContains(t, strings.Join(events.DrainQueuedMessagesForTest(2), "\n"), alice.Character.Name)
	_ = bob
}

// In the dark a mob's look is silent.
func TestMobLook_DarkIsSilent(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	mob, room := getTestMobAndRoom(t)
	room.SkyLight, room.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(0)
	events.DrainQueuedMessagesForTest(1)
	Look("", mob, room)
	Look("aliceia", mob, room)
	Look("north", mob, room)
	require.Empty(t, events.DrainQueuedMessagesForTest(1))
}
```

- [ ] **Step 7: The mob wrapper**

Replace `mobcommands.Look` (`:14-170`) with:

```go
func Look(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	secretLook := false
	if strings.HasPrefix(rest, "secretly") {
		secretLook = true
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "secretly"))
	}

	isSneaking := mob.Character.IsHidden()

	// trim off some fluff
	if len(rest) > 2 {
		if rest[0:3] == `at ` {
			rest = rest[3:]
		}
	}
	if len(rest) > 3 {
		if rest[0:4] == `the ` {
			rest = rest[4:]
		}
	}

	name := mob.Character.Name

	// The player's sight rules, shared (actions.ResolveLook, slice 5a). A mob
	// is silent on every refusal; its room lines hide names by each
	// observer's sight, as the player's do.
	res := actions.ResolveLook(actions.NewMobActorInRoom(mob, room), rest)
	switch res.Kind {
	case actions.LookDark, actions.LookExitTooDark, actions.LookExitLocked:
		return true, nil

	case actions.LookRoom:
		if !secretLook && !isSneaking {
			room.SendTextVisualHidingNames(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking around.`, name), []string{name})
			// Make it a "secret looks" now because we don't want another look message sent out by the lookRoom() func
			secretLook = true
		}
		lookRoom(mob, room.RoomId, secretLook || isSneaking)
		return true, nil

	case actions.LookExit:
		if !isSneaking {
			room.SendTextVisualHidingNames(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> peers toward the %s.`, name, res.ExitName), []string{name})
		}
		lookRoom(mob, res.ExitRoomId, secretLook || isSneaking)
		return true, nil

	case actions.LookCreature:
		if isSneaking {
			return true, nil
		}
		if res.Target.IsPlayer() {
			u := res.Target.(*actions.UserActor).User
			u.SendText(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking at you.`, name))
			room.SendTextVisualHidingNames(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking at <ansi fg="username">%s</ansi>.`, name, u.Character.Name),
				[]string{name, u.Character.Name}, u.UserId)
			return true, nil
		}
		m := res.Target.(*actions.MobActor).Mob
		room.SendTextVisualHidingNames(messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking at %s.`, name, m.Character.GetMobName(0).String()),
			[]string{name})
		return true, nil
	}

	// LookOther: the mob's own objects, in its order.
	if lookItem, found := mob.Character.FindInBackpack(rest); found {
		if !isSneaking {
			room.SendTextVisualHidingNames(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is admiring their <ansi fg="item">%s</ansi>.`, name, lookItem.DisplayName()), []string{name})
		}
		return true, nil
	}
	if lookItem, found := mob.Character.FindOnBody(rest); found {
		if !isSneaking {
			room.SendTextVisualHidingNames(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is admiring their <ansi fg="item">%s</ansi>.`, name, lookItem.DisplayName()), []string{name})
		}
		return true, nil
	}
	if foundNoun, _ := room.FindNoun(rest); len(foundNoun) > 0 {
		if !isSneaking {
			room.SendTextVisualHidingNames(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> is examining the <ansi fg="noun">%s</ansi>.`, name, foundNoun), []string{name})
		}
		return true, nil
	}
	if res.PetUserId > 0 {
		if petUser := users.GetByUserId(res.PetUserId); petUser != nil {
			room.SendTextVisualHidingNames(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is looking at %s.`, name, petUser.Character.Pet.DisplayName()), []string{name})
		}
	}
	return true, nil
}
```

In `lookRoom` (mob), change both `room.SendTextVisual(...)` calls to `room.SendTextVisualHidingNames(..., []string{mob.Character.Name})` with the same text.

- [ ] **Step 8: Run**

Run: `go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ -count=1` then `go test . -run 'TestEveryCreatureLookupDeclaresItsViewer|TestNarrationSitesMatchViewpointAudit|TestFinderViewReachesOnlyItsReader' -count=1`
Expected: PASS. The narration guard's `usercommands/look.go|...` keys should survive (the literals and their actor-then-observer shape are unchanged); if one moves, re-key it in this commit.

- [ ] **Step 9: Commit**

```bash
git add internal/actions/look.go internal/actions/sight_gates_parity_test.go internal/usercommands/look.go internal/mobcommands/look.go internal/mobcommands/look_sight_gates_test.go lookup_viewer_guard_test.go
git commit -m "feat(look): actions.ResolveLook, so a mob looks by the player's sight rules"
```

---

### Task 9: Remove: busy, curse, `remove all`, the companion

**Model:** sonnet.

**Files:**
- Modify: `internal/actions/remove_equip.go:86-131`
- Modify: `internal/actions/sight_gates_parity_test.go`
- Modify: `internal/usercommands/busy_refuse.go`, `internal/usercommands/remove.go`
- Modify: `internal/mobcommands/remove.go`
- Modify: `modules/aicompanion/actions.go:408-412`
- Create: `internal/usercommands/remove_curse_test.go`, `internal/mobcommands/remove_gates_test.go`, `modules/aicompanion/sight_gates_test.go`
- Modify: `bauble_sweep_guard_test.go` (register `actions.RemoveAllResult`)

- [ ] **Step 1: Write the failing parity rows**

Append to `internal/actions/sight_gates_parity_test.go` (imports: `activity`, `skills`):

```go
func gateBusy(t *testing.T, c *characters.Character) {
	t.Helper()
	c.Activity = activity.NewMachine()
	require.NoError(t, c.Activity.TransitionToCrafting(
		activity.CraftingData{RecipeId: "test", RoundsTotal: 3},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin}))
}

func TestGateParity_RemoveWhileBusy(t *testing.T) {
	for _, who := range gateWho {
		s := newGateScene(t, gateLit)
		a := s.actor(who)
		a.GetCharacter().Equipment.Head = gateItem(39701, "cap", items.Head, false)
		gateBusy(t, a.GetCharacter())
		assert.True(t, RemoveEquipment(a, "cap").Busy, who)
		assert.True(t, RemoveAllEquipment(a).Busy, who)
		assert.Equal(t, 39701, a.GetCharacter().Equipment.Head.ItemId, who)
	}
}

func TestGateParity_RemoveCursed(t *testing.T) {
	for _, who := range gateWho {
		s := newGateScene(t, gateLit)
		a := s.actor(who)
		c := a.GetCharacter()
		c.Equipment.Ring = gateItem(39702, "hexed ring", items.Ring, true)
		res := RemoveEquipment(a, "hexed ring")
		assert.True(t, res.Cursed, who)
		assert.False(t, res.Removed, who)
		assert.Equal(t, 39702, c.Equipment.Ring.ItemId, who)

		c.SetSkill(string(skills.Spellcasting), 4)
		res = RemoveEquipment(a, "hexed ring")
		assert.True(t, res.Removed, who)
		assert.True(t, res.CursedOverridden, who)
	}
}

func TestGateParity_RemoveAllSkipsCursed(t *testing.T) {
	for _, who := range gateWho {
		s := newGateScene(t, gateLit)
		a := s.actor(who)
		c := a.GetCharacter()
		c.Equipment.Ring = gateItem(39703, "hexed ring", items.Ring, true)
		c.Equipment.Head = gateItem(39704, "cap", items.Head, false)
		res := RemoveAllEquipment(a)
		require.Len(t, res.Cursed, 1, who)
		assert.Equal(t, 39703, res.Cursed[0].ItemId, who)
		require.Len(t, res.Removed, 1, who)
		assert.Equal(t, 39704, res.Removed[0].ItemId, who)
		assert.Equal(t, 39703, c.Equipment.Ring.ItemId, who)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/actions/ -run 'TestGateParity_Remove' -count=1`
Expected: build failure (`Busy`, `Cursed`, `CursedOverridden`, `RemoveAllEquipment` undefined).

- [ ] **Step 3: The shared remove body**

In `internal/actions/remove_equip.go` (import `characters`, `skills`), replace from `// RemoveEquipResult is the result` to the end with:

```go
// RemoveEquipResult is the result of a RemoveEquipment call.
type RemoveEquipResult struct {
	Item    items.Item
	Found   bool
	Removed bool // true when RemoveFromBody succeeded and item was stored/dropped
	// Busy: refused because the actor is focused on work (IsActing), before
	// anything is looked up.
	Busy bool
	// Cursed: refused because the curse holds (CursedHolds).
	Cursed bool
	// CursedOverridden: the item was cursed but the actor's Spellcasting
	// lifted it; it came off.
	CursedOverridden bool
	Err              error
}

// CursedHolds is the one statement of the remove curse rule: a cursed item
// stays on a living wearer, unless their Spellcasting is 4 or more, which
// overrides it. Both remove wrappers and the AI companion ask it. (Equip has
// its own rule, characters.CursedRefusal, with no exception.)
func CursedHolds(char *characters.Character, item items.Item) (holds, overridden bool) {
	if !item.IsCursed() || char.Health <= 0 {
		return false, false
	}
	if char.GetSkillLevel(skills.Spellcasting) >= 4 {
		return false, true
	}
	return true, false
}

// RemoveEquipment takes a worn item off and stores it in the backpack
// (dropping it on the floor if the backpack is full). Gates, in order: Busy
// (before the lookup, as the player's command always refused first), not
// found, Cursed. It reveals the actor, fires EquipmentChange and calls
// Validate(). Messaging stays in the callers.
func RemoveEquipment(actor Actor, itemName string) RemoveEquipResult {
	char := actor.GetCharacter()
	if char.IsActing() {
		return RemoveEquipResult{Busy: true}
	}
	matchItem, found := char.FindOnBody(itemName)
	if !found || matchItem.ItemId < 1 {
		return RemoveEquipResult{Found: false}
	}
	return removeWorn(actor, matchItem)
}

// removeWorn takes one worn item off through the curse gate.
func removeWorn(actor Actor, matchItem items.Item) RemoveEquipResult {
	char := actor.GetCharacter()
	room := actor.GetRoom()

	holds, overridden := CursedHolds(char, matchItem)
	if holds {
		return RemoveEquipResult{Item: matchItem, Found: true, Cursed: true}
	}

	char.Awareness.TransitionToRevealing(state.TransitionReason{
		Trigger: awareness.TriggerForceVisible,
	})

	if !char.RemoveFromBody(matchItem) {
		// RemoveFromBody failed: item is still on body
		return RemoveEquipResult{Item: matchItem, Found: true, CursedOverridden: overridden}
	}

	if !char.StoreItem(matchItem) {
		// Backpack full: drop to floor as safety net
		room.AddItem(matchItem, false)
	}

	events.AddToQueue(events.EquipmentChange{
		UserId:        actor.GetUserId(),
		MobInstanceId: actor.GetMobInstanceId(),
		ItemsRemoved:  []items.Item{matchItem},
	})

	char.Validate()

	return RemoveEquipResult{Item: matchItem, Found: true, Removed: true, CursedOverridden: overridden}
}

// RemoveAllResult is the result of a RemoveAllEquipment call.
type RemoveAllResult struct {
	Busy    bool
	Removed []items.Item
	Cursed  []items.Item // left on: the curse holds
}

// RemoveAllEquipment is `remove all` for both actors: the busy gate once,
// then every worn item through removeWorn by identity (not by name, so two
// same-named pieces cannot confuse it). A cursed item is skipped and listed;
// the rest come off, one EquipmentChange each.
func RemoveAllEquipment(actor Actor) RemoveAllResult {
	char := actor.GetCharacter()
	if char.IsActing() {
		return RemoveAllResult{Busy: true}
	}
	var out RemoveAllResult
	for _, item := range char.Equipment.GetAllItems() {
		res := removeWorn(actor, item)
		switch {
		case res.Cursed:
			out.Cursed = append(out.Cursed, res.Item)
		case res.Removed:
			out.Removed = append(out.Removed, res.Item)
		}
	}
	return out
}
```

(`RemoveFromBody` matches by `Equals`, so identity removal works for the copy `GetAllItems` returns; confirm with `grep -n "func (w \*Worn) GetAllItems\|func (w Worn) GetAllItems" internal/characters/worn.go`.)

Register `internal/actions.RemoveAllResult` in `bauble_sweep_guard_test.go`'s `transientItemHolders` with `an action's result, alive for one call`.

- [ ] **Step 4: Write the failing wrapper tests**

Create `internal/usercommands/remove_curse_test.go`:

```go
package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hexed(id int, name string, t items.ItemType) items.Item {
	return items.Item{ItemId: id, Spec: &items.ItemSpec{ItemId: id, Name: name, Type: t, Subtype: items.Wearable, Cursed: true}}
}

func removeOut(t *testing.T, rest string) (string, func() *items.Item) {
	t.Helper()
	user, room := getTestUserAndRoom(t)
	events.DrainQueuedMessagesForTest(user.UserId)
	handled, err := Remove(rest, user, room, 0)
	require.NoError(t, err)
	require.True(t, handled)
	return strings.Join(events.DrainQueuedMessagesForTest(user.UserId), "\n"), func() *items.Item { return &user.Character.Equipment.Ring }
}

func TestRemove_CursedLinesUnchanged(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	user, _ := getTestUserAndRoom(t)
	user.Character.Equipment.Ring = hexed(96301, "hexed ring", items.Ring)
	out, ring := removeOut(t, "hexed ring")
	assert.Contains(t, out, "You can't seem to remove your")
	assert.Contains(t, out, "CURSED!")
	assert.Equal(t, 96301, ring().ItemId)

	user.Character.SetSkill("spellcasting", 4)
	out, ring = removeOut(t, "hexed ring")
	assert.Contains(t, out, "luckily your")
	assert.Equal(t, 0, ring().ItemId)
}

// `remove all` leaves cursed gear on, with the cursed line, and takes the
// rest off (it stripped cursed gear before).
func TestRemoveAll_SkipsCursedGear(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	user, _ := getTestUserAndRoom(t)
	user.Character.Equipment.Ring = hexed(96302, "hexed ring", items.Ring)
	user.Character.Equipment.Head = items.Item{ItemId: 96303, Spec: &items.ItemSpec{ItemId: 96303, Name: "cap", Type: items.Head, Subtype: items.Wearable}}
	out, ring := removeOut(t, "all")
	assert.Contains(t, out, "CURSED!")
	assert.Equal(t, 96302, ring().ItemId)
	assert.Equal(t, 0, user.Character.Equipment.Head.ItemId)
}
```

Create `internal/mobcommands/remove_gates_test.go`:

```go
package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A busy mob keeps its gear on, and a cursed piece stays on against `remove`
// and `remove all` (rulings 4 and 5).
func TestMobRemove_BusyAndCursedGates(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	mob, room := getTestMobAndRoom(t)
	mob.Character.Equipment.Ring = items.Item{ItemId: 96401, Spec: &items.ItemSpec{ItemId: 96401, Name: "hexed ring", Type: items.Ring, Subtype: items.Wearable, Cursed: true}}
	mob.Character.Equipment.Head = items.Item{ItemId: 96402, Spec: &items.ItemSpec{ItemId: 96402, Name: "cap", Type: items.Head, Subtype: items.Wearable}}

	Remove("hexed ring", mob, room)
	assert.Equal(t, 96401, mob.Character.Equipment.Ring.ItemId)

	mob.Character.Activity = activity.NewMachine()
	require.NoError(t, mob.Character.Activity.TransitionToCrafting(
		activity.CraftingData{RecipeId: "test", RoundsTotal: 3},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin}))
	Remove("cap", mob, room)
	assert.Equal(t, 96402, mob.Character.Equipment.Head.ItemId, "busy")

	mob.Character.Activity = nil
	Remove("all", mob, room)
	assert.Equal(t, 96401, mob.Character.Equipment.Ring.ItemId)
	assert.Equal(t, 0, mob.Character.Equipment.Head.ItemId)
}
```

Create `modules/aicompanion/sight_gates_test.go`:

```go
package aicompanion

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// Her `remove` refuses a cursed worn item up front, as `get` refuses a
// household's bauble, so she does not record a futile attempt (spec R8).
func TestCompanionRemoveRefusesACursedItem(t *testing.T) {
	owner, _, room, her := harmWorld(t, configs.PVPDisabled)
	ring := items.Item{ItemId: 96501, Spec: &items.ItemSpec{ItemId: 96501, Name: "hexed ring", Type: items.Ring, Subtype: items.Wearable, Cursed: true}}
	her.Character.Equipment.Ring = ring
	m, c, _ := strangerModule()
	sc := &scene{RoomId: room.RoomId, byRef: map[string]*thing{}}
	sc.byRef[`w1`] = &thing{Ref: `w1`, Kind: `worn`, Name: `Hexed Ring`, Item: ring, HasItem: true}
	out := m.performAction(c, her, owner, sc, ActionProposal{Verb: `remove`, Ref: `w1`},
		[]stimulus{{Kind: `heard`, FromOwner: true}}, 0, 0)
	if out.Issued || out.Refused != `it will not come off` {
		t.Fatalf("want a refusal before any command, got %+v", out)
	}
}
```

(If `performAction` needs `stillThere` to find the worn item, it reads `mob.Character.Equipment`; the ring is on her, so it is.)

- [ ] **Step 5: The wrappers and the companion**

`internal/usercommands/busy_refuse.go`: split the text out.

```go
// busyRefusalText is the focused-work refusal, shared by refuseWhileBusy and
// the wrappers whose shared body reports Busy itself (remove, slice 5a).
func busyRefusalText(verb string) string {
	return fmt.Sprintf(`<ansi fg="red">You can't %s while focused on your work. Finish or be interrupted first.</ansi>`, verb)
}

func refuseWhileBusy(user *users.UserRecord, verb string) bool {
	if user.Character.IsActing() {
		user.SendText(messaging.CategorySystem, busyRefusalText(verb))
		return true
	}
	return false
}
```

(keep the existing doc comment above `refuseWhileBusy`).

`internal/usercommands/remove.go`: replace the body of `Remove` with:

```go
func Remove(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	actor := &actions.UserActor{User: user, Room: room}

	// Snapshot reservation BEFORE anything leaves the body, so the disclosure
	// below can tell the player when taking that off gave capacity back.
	// Taken up here so the `all` branch is covered by the same snapshot.
	beforeReservation := user.Character.ReservationTotals()

	cursedLine := func(it items.Item) string {
		return fmt.Sprintf(`You can't seem to remove your <ansi fg="item">%s</ansi>... It's <ansi fg="red-bold">CURSED!</ansi>`, it.DisplayName())
	}

	if rest == "all" {
		// The busy, curse and per-item rules live in actions.RemoveAllEquipment
		// (slice 5a), which fires one EquipmentChange per item.
		res := actions.RemoveAllEquipment(actor)
		if res.Busy {
			user.SendText(messaging.CategorySystem, busyRefusalText(`change equipment`))
			return true, nil
		}
		for _, it := range res.Cursed {
			user.SendText(messaging.CategorySystem, cursedLine(it))
		}
		sendReservationReturnDisclosure(user, beforeReservation)
		for _, item := range res.Removed {
			if item.GetSpec().Type == items.Light {
				lightnotice.Check(user, lightnotice.TriggerCommand)
				break
			}
		}
		return true, nil
	}

	result := actions.RemoveEquipment(actor, rest)
	switch {
	case result.Busy:
		user.SendText(messaging.CategorySystem, busyRefusalText(`change equipment`))
	case !result.Found:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You don't appear to be using a "%s".`, rest))
	case result.Cursed:
		user.SendText(messaging.CategorySystem, cursedLine(result.Item))
	default:
		if result.CursedOverridden {
			user.SendText(messaging.CategorySystem,
				`It's <ansi fg="red-bold">CURSED</ansi> but luckily your <ansi fg="skillname">enchant</ansi> skill level allows you to remove it.`,
			)
		}
		if result.Removed {
			user.SendText(messaging.CategorySystem,
				fmt.Sprintf(`You remove your <ansi fg="item">%s</ansi> and return it to your backpack.`, result.Item.DisplayName()),
			)
			room.SendTextVisual(messaging.CategoryEquipment,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> removes their <ansi fg="item">%s</ansi> and stores it away.`, user.Character.Name, result.Item.DisplayName()),
				user.UserId,
			)
			sendReservationReturnDisclosure(user, beforeReservation)
			// Taking off a light changes the band at once; the notice rides
			// this command rather than waiting for the next one.
			if result.Item.GetSpec().Type == items.Light {
				lightnotice.Check(user, lightnotice.TriggerCommand)
			}
		} else {
			user.SendText(messaging.CategorySystem,
				fmt.Sprintf(`You can't seem to remove your <ansi fg="item">%s</ansi>.`, result.Item.DisplayName()),
			)
		}
	}
	return true, nil
}
```

Drop the `skills` import.

`internal/mobcommands/remove.go`: keep the `PermaGear` refusal first; replace the rest with:

```go
	actor := &actions.MobActor{Mob: mob, Room: room}

	// Busy and curse gates live in the shared bodies (slice 5a); a mob is
	// silent on every refusal.
	if rest == "all" {
		actions.RemoveAllEquipment(actor)
		return true, nil
	}

	result := actions.RemoveEquipment(actor, rest)
	if result.Removed {
		room.SendTextVisual(messaging.CategoryEquipment,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> removes their <ansi fg="item">%s</ansi> and stores it away.`, mob.Character.Name, result.Item.DisplayName()))
	}
	return true, nil
```

Drop the `events` and `items` imports if unused.

`modules/aicompanion/actions.go`, the `remove` case:

```go
	case `remove`:
		if t.Kind != `worn` {
			return actionOutcome{Refused: `not worn`}
		}
		// The shared curse rule (actions.CursedHolds): say so now rather than
		// issue a remove that changes nothing.
		if holds, _ := actions.CursedHolds(&mob.Character, t.Item); holds {
			return actionOutcome{Refused: `it will not come off`}
		}
		return m.issue(c, mob, `remove`, `remove `+target, t.Key, t.Name, ``, delay, round)
```

- [ ] **Step 6: Run**

Run: `go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ ./modules/aicompanion/ -count=1` then `go test . -run 'TestNarrationSitesMatchViewpointAudit|TestBaubleSweep' -count=1`
Expected: PASS, including `TestBusyCommands_RefuseWhileCrafting` (the busy line for `remove sword` with nothing worn) and `remove_reservation_disclosure_test.go`. If the narration guard moves `usercommands/remove.go|You remove your ...` (it now sits in a `switch` default), re-key it in this commit.

- [ ] **Step 7: Commit**

```bash
git add internal/actions/remove_equip.go internal/actions/sight_gates_parity_test.go internal/usercommands/busy_refuse.go internal/usercommands/remove.go internal/usercommands/remove_curse_test.go internal/mobcommands/remove.go internal/mobcommands/remove_gates_test.go modules/aicompanion/actions.go modules/aicompanion/sight_gates_test.go bauble_sweep_guard_test.go
git commit -m "feat(remove): busy and curse gates in the shared body; remove all leaves cursed gear on"
```

---

### Task 10: Craft: `TooDarkToCraft` and the companion

**Model:** haiku.

**Files:**
- Modify: `internal/actions/craft.go:17-45,131-138`
- Modify: `internal/usercommands/craft.go:83-95`, `internal/mobcommands/craft.go:28-47`
- Modify: `modules/aicompanion/cooking.go:58-60`
- Modify: `internal/actions/sight_gates_parity_test.go`, `modules/aicompanion/sight_gates_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/actions/sight_gates_parity_test.go`:

```go
func TestGateParity_CraftNeedsClearSight(t *testing.T) {
	for _, light := range gateLights {
		for _, who := range gateWho {
			s := newGateScene(t, light)
			a := s.actor(who)
			res := InitiateCraft(a, "no-such-recipe")
			if light == gateLit {
				assert.False(t, TooDarkToCraft(a), "%s at %s", who, light)
				assert.False(t, res.CannotSee, "%s at %s", who, light)
				continue
			}
			assert.True(t, TooDarkToCraft(a), "%s at %s: shapes are not enough for fine work", who, light)
			assert.Equal(t, CraftResult{CannotSee: true}, res, "%s at %s", who, light)
		}
	}
}
```

Append to `modules/aicompanion/sight_gates_test.go` (add imports `crafting`, `rooms`):

```go
// She neither offers nor starts a recipe she cannot see to make (spec C5).
func TestCraftableHereIsEmptyInTheDark(t *testing.T) {
	_, _, room, her := harmWorld(t, configs.PVPDisabled)
	crafting.RegisterRecipeForTest(&crafting.RecipeSpec{RecipeId: `sg-twine`, Name: `Twine`, Skill: `tailoring`})
	t.Cleanup(func() { crafting.UnregisterRecipeForTest(`sg-twine`) })
	her.Character.KnownRecipes = map[string]int{`sg-twine`: 1}
	p := &Profile{Crafts: []string{`tailoring`}}

	if got := craftableHere(her, p, room); len(got) != 1 {
		t.Fatalf("control: lit, she can make twine, got %+v", got)
	}
	room.SkyLight, room.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(0)
	if got := craftableHere(her, p, room); len(got) != 0 {
		t.Fatalf("dark: nothing is craftable here, got %+v", got)
	}
}
```

(Check `KnownRecipes`' map value type with `grep -n "KnownRecipes " internal/characters/character.go` and match it.)

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/actions/ ./modules/aicompanion/ -run 'TestGateParity_Craft|TestCraftableHere' -count=1`
Expected: build failure (`TooDarkToCraft`, `CannotSee` undefined), then the dark companion row fails.

- [ ] **Step 3: Implement**

`internal/actions/craft.go`: add to `CraftResult`

```go
	// CannotSee is true when the actor cannot see clearly enough to work
	// (TooDarkToCraft). Checked first.
	CannotSee bool
```

and before `InitiateCraft`:

```go
// TooDarkToCraft is the one statement of the craft sight rule for both
// actors (slice 5a): crafting is fine work, so shapes by infrared are not
// enough; it needs clear sight (awake, SightFull). The player's command also
// asks it before its storage pull and enchanting branch.
func TooDarkToCraft(actor Actor) bool {
	return !messaging.CanSeeClearly(actor.GetCharacter(), actor.GetRoom())
}
```

and as `InitiateCraft`'s first gate, before `IsCrafting`:

```go
	// ── Can the actor see to work? ────────────────────────────────────────────
	if TooDarkToCraft(actor) {
		return CraftResult{CannotSee: true}
	}
```

(add the `messaging` import if missing).

`internal/usercommands/craft.go:92`: replace `if !messaging.CanSeeClearly(user.Character, room) {` with `if actions.TooDarkToCraft(&actions.UserActor{User: user, Room: room}) {`, and in the comment above replace "CanSeeClearly, not CanSeeShapes" with "Clear sight, not shapes (actions.TooDarkToCraft)". In the `InitiateCraft` result switch add a first case `case result.CannotSee:` printing the same "You can't see well enough to work on anything here." line (unreachable today, but the result is part of the contract).

`internal/mobcommands/craft.go`: add `case result.CannotSee:` with the comment `// Cannot see to work; waits for light (no-op for mobs)` returning `true, nil`, first in the switch.

`modules/aicompanion/cooking.go`, `craftableHere`: after the nil/empty guard add:

```go
	// Crafting needs clear sight, for her as for anyone (slice 5a): offer
	// nothing in the dark, so autonomy never picks a craft that would not
	// start and the prompt lists no recipe she cannot see to make.
	if cannotSee(mob, room) {
		return nil
	}
```

- [ ] **Step 4: Run**

Run: `go test ./internal/actions/ ./internal/usercommands/ ./internal/mobcommands/ ./modules/aicompanion/ ./internal/planners/ -count=1`
Expected: PASS. A craft test whose fixture room is unlit now reads `CannotSee`: pin that fixture's lamp and name it in the commit message.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/craft.go internal/actions/sight_gates_parity_test.go internal/usercommands/craft.go internal/mobcommands/craft.go modules/aicompanion/cooking.go modules/aicompanion/sight_gates_test.go
git commit -m "feat(craft): crafting needs clear sight for mobs and the companion too"
```

---

### Task 11: The re-fork guard

**Model:** haiku.

**Files:**
- Create: `sight_gates_wrapper_guard_test.go`

- [ ] **Step 1: Write the guard**

Create `sight_gates_wrapper_guard_test.go`:

```go
package main

import (
	"bytes"
	"go/parser"
	"go/printer"
	"go/token"
	"regexp"
	"testing"
)

// TestSightGateWrappersDoNotReFork: the 5a gates live in shared bodies
// (actions.TooDarkToGet, TakeFloorItem, ResolveLook, CursedHolds,
// RemoveEquipment, RemoveAllEquipment, TooDarkToCraft; Character.Wear,
// WearInArm, CursedRefusal, ChooseWornSlot). If a command wrapper evaluates
// sight, a curse, busyness or a slot itself, the player and mob paths have
// forked again. Comments are dropped before matching (the files name these
// words in comments), by printing each file's AST parsed without them.
func TestSightGateWrappersDoNotReFork(t *testing.T) {
	rows := []struct {
		files     []string
		forbidden *regexp.Regexp
		fix       string
	}{
		{[]string{"internal/usercommands/get.go", "internal/mobcommands/get.go"},
			regexp.MustCompile("ParticipantSight|CanSeeShapes|CanSeeClearly|`exploding`"),
			"ask actions.TooDarkToGet or go through actions.TakeFloorItem"},
		{[]string{"internal/usercommands/look.go", "internal/mobcommands/look.go"},
			regexp.MustCompile(`ParticipantSight|SeesThroughExit|CanSeeClearly|ResolveTargetActor`),
			"go through actions.ResolveLook"},
		{[]string{"internal/usercommands/remove.go", "internal/mobcommands/remove.go"},
			regexp.MustCompile(`IsCursed|IsActing|refuseWhileBusy|Spellcasting`),
			"go through actions.RemoveEquipment / RemoveAllEquipment"},
		{[]string{"internal/usercommands/equip.go", "internal/mobcommands/equip.go", "internal/usercommands/gearup.go", "internal/mobcommands/gearup.go"},
			regexp.MustCompile(`IsCursed|Spellcasting|CursedRefusal|ChooseWornSlot|\.Wear\(`),
			"go through actions.EquipItem / EquipItemInArm"},
		{[]string{"internal/usercommands/equip.go"},
			regexp.MustCompile(`GetHandPairs|HandsRequired|ItemPtr`),
			"arm placement lives in Character.WearInArm"},
		{[]string{"internal/usercommands/craft.go", "internal/mobcommands/craft.go"},
			regexp.MustCompile(`CanSeeClearly|ParticipantSight`),
			"ask actions.TooDarkToCraft"},
	}
	for _, row := range rows {
		for _, path := range row.files {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0) // no ParseComments: comments dropped
			if err != nil {
				t.Fatalf("parse %s (test must run from the repo root): %v", path, err)
			}
			var code bytes.Buffer
			if err := printer.Fprint(&code, fset, file); err != nil {
				t.Fatalf("print %s: %v", path, err)
			}
			if loc := row.forbidden.FindIndex(code.Bytes()); loc != nil {
				t.Errorf("%s applies a 5a gate itself (%q); %s", path, code.Bytes()[loc[0]:loc[1]], row.fix)
			}
		}
	}
}
```

- [ ] **Step 2: Run it**

Run: `go test . -run TestSightGateWrappersDoNotReFork -count=1`
Expected: PASS.

- [ ] **Step 3: Prove every row can fail**

For each row, add one temporary line to the first file of that row, run Step 2's command, expect FAIL naming that file, then remove it:
- `internal/usercommands/get.go`, inside `Get`: `_ = messaging.ParticipantSight`
- `internal/mobcommands/look.go`, inside `Look`: `_ = actions.ResolveTargetActor`
- `internal/usercommands/remove.go`, inside `Remove`: `_ = user.Character.IsActing()`
- `internal/mobcommands/equip.go`, inside `Equip`: `_ = mob.Character.CursedRefusal`
- `internal/usercommands/equip.go`, inside `Equip`: `_ = user.Character.GetHandPairs`
- `internal/mobcommands/craft.go`, inside `Craft`: `_ = messaging.CanSeeClearly`

Also add `// ParticipantSight` as a comment line in `internal/usercommands/get.go` and confirm the guard still PASSES (comments are dropped). Remove it. After all probes, `git diff --stat` must show only the new guard file.

- [ ] **Step 4: Commit**

```bash
git add sight_gates_wrapper_guard_test.go
git commit -m "test(guard): the 5a wrappers cannot re-fork sight, curse, busy or slot rules"
```

---

### Task 12: Docs, full gate, boot check

**Model:** sonnet.

**Files:**
- Modify: `internal/actions/context.md`, `internal/characters/context.md`, `internal/itemvalue/context.md`, `internal/mobcommands/context.md`, `internal/usercommands/context.md`, `modules/aicompanion/context.md`, `docs/PATCH_NOTES.md`

- [ ] **Step 1: context.md**

Verify every symbol first: `Select-String -Path internal\<pkg>\*.go -Pattern '^(func|type|const|var)\s'` (PowerShell), per package. Then:
- `actions`: `get.go` (`ErrTooDark`, `ErrExploding`, `TooDarkToGet`, `TakeFloorItem`, the gate order, dark `GetItemFromFloor` finds nothing, `GetGoldFromFloor` refuses dark); `look.go` (`ResolveLook`, `LookResolution`, the `LookKind` values, why `PetUserId` is not a kind, the crate and container deferral); `remove_equip.go` (`CursedHolds`, `RemoveEquipResult.Busy/Cursed/CursedOverridden`, `RemoveAllEquipment`, `RemoveAllResult`, `EquipItemInArm`, `EquipItemResult.ArmLabel`; replace the stale "Cursed-item checks ... remain in the callers" line); `craft.go` (`TooDarkToCraft`, `CraftResult.CannotSee`). Name the guard `sight_gates_wrapper_guard_test.go`.
- `characters`: `wear_slot.go` (`SlotChoice`, `CursedRefusal`, `ChooseWornSlot` and its fill, swap, refuse rule over 2 to 6 arms; the candidate lists; the ruling 13 shield swap; the arm-N case), `Wear` as `wear(i, place)`, `WearInArm`, `ArmLabel`, the `HandsRequired` nil guard, the deleted `Find*`/`Pair*` helpers, the golden oracle test and its two carve-outs. Fix the stale `ExtraArms` "(0-2)" field comment note if the file mentions it (the real range is 0 to 4).
- `itemvalue`: `compatibleSlotsFor` and `displacedItemsForSlot` ask `ChooseWornSlot` for weapons, shields, rings and wrists; `ItemValueDelta` skips a swap a curse would refuse; `extraArmsLevel` is gone; the agreement test.
- `mobcommands`: get, look, remove and craft are wrappers over the shared bodies and silent on every refusal; look room lines hide names by sight.
- `usercommands`: `equip X armN` goes through `actions.EquipItemInArm`; `busyRefusalText`; get, look, remove and craft word the shared answers.
- `modules/aicompanion`: the `remove` verb refuses a cursed item up front; `craftableHere` returns nothing in the dark.

Run: `python tools/context_md_audit.py` and expect no phantom symbol for these six packages.

- [ ] **Step 2: Patch notes**

Add at the top of `docs/PATCH_NOTES.md` (player-facing, no numbers, no dashes, 80-column wrap):

```markdown
## 2026-09-29: Cursed gear and the dark

- Cursed gear now stays put. Putting on armour, a ring, a bracer or a
  light over a cursed piece fails, and taking everything off leaves your
  cursed pieces on unless your magic is strong enough to lift the curse.
- With every ring or wrist slot full, a new ring or bracer replaces the
  first one that is not cursed.
- Weapons and shields skip a hand held by a cursed item and go to the next
  free hand you can use, on every arm you have.
- A creature with extra arms can now take up a shield beside a two handed
  weapon even with every hand full: the shield swaps out the item in its
  last free hand.
- Equipping into a named arm now checks your strength and your reserves
  like any other equip, and an item it knocks off a full pack lands on the
  floor instead of being lost.
- Monsters now follow your rules. In the dark they cannot pick things up
  or craft, and a monster can no longer look at you while you are hidden.
  A busy monster cannot take its gear off, and a monster's cursed gear
  stays on.
```

Copy-check against `dogmud-player-copy` (no numbers, no dashes, 80 columns).

- [ ] **Step 3: Full gate**

```bash
gofmt -l internal/ modules/ .
go vet ./...
go build ./...
go test ./... -count=1
golangci-lint run --new-from-merge-base=origin/master
```

Expected: gofmt and vet print nothing; every package `ok`; lint `0 issues`. A root guard keyed by file and literal or file and function (`TestNarrationSitesMatchViewpointAudit`, `TestEveryCreatureLookupDeclaresItsViewer`, `TestFinderViewReachesOnlyItsReader`, the bauble sweep guard) that still reports a moved site is re-keyed here in the same commit, never deleted. A failure unrelated to 5a is checked on a detached `origin/master` worktree before it is called pre-existing.

- [ ] **Step 4: Boot check**

Per `dogmud-shipping` (the config is skip-worktree, so copy it from the main checkout):

```bash
git worktree add --detach C:/tmp/dogmud-boot-check HEAD
cp "C:/Users/Calabe Davis/workspace/DOGMud/_datafiles/config.yaml" C:/tmp/dogmud-boot-check/_datafiles/config.yaml
cd C:/tmp/dogmud-boot-check && go build -o boot-check.exe .
timeout 180 ./boot-check.exe > boot.log 2>&1
grep -cE "^panic:|goroutine [0-9]+ \[running\]|runtime error" boot.log
grep -c "Server Ready" boot.log
```

Expected: exit 124, `0`, `1`. Remove the worktree afterwards (`git worktree remove --force`; PowerShell `Remove-Item -Recurse -Force` if Windows holds a lock, then `git worktree prune`). Kill only a process you started, by PID.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/context.md internal/characters/context.md internal/itemvalue/context.md internal/mobcommands/context.md internal/usercommands/context.md modules/aicompanion/context.md docs/PATCH_NOTES.md
git commit -m "docs(5a): context.md and patch notes for object and action gate parity"
```

(Add by name any guard file re-keyed in Step 3.)

---

### Task 13: Playtest and PR

**Model:** opus (judgment on findings).

- [ ] **Step 1: Playtest**

Load `dogmud-playtesting` and follow it: ephemeral goals file, `--checkout` of `feature/sight-gates-5a`, never kill the owner's server. The live world ships no cursed item (F39), so the run first curses an item through the builder item editor (tick Cursed on an ordinary ring and an ordinary helmet on the ephemeral checkout), then spawns them. Use a low-perception profile (no night vision, no infrared) for the sight checks. Goals, each confirming nothing freezes or loops:
1. Wear the cursed helmet, then try to put another helmet on: "Your ... is cursed and prevents you from removing it." `remove all` leaves it on with the cursed line and takes the rest off.
2. Wear the cursed ring and a plain ring, then a third ring: it replaces the plain one.
3. `equip <sword> arm1` over a cursed main-hand weapon refuses with the shared line; a too-heavy weapon named into an arm is refused for strength.
4. In a dark room: `get`, `look`, `look <player>` and `craft` refuse with today's lines; a companion in the dark at a station offers no recipe and starts none.
5. Hand the AI companion a better helmet while she wears the cursed one: she considers it and keeps it in her pack, silently, once.
6. Hide, and have a mob `look` at you (admin `command`): you see no "is looking at you".

Extract findings to memory (reports are gitignored).

- [ ] **Step 2: Push and open the PR**

```bash
git push -u origin feature/sight-gates-5a
gh pr create --repo pruuk/DOGMud --base master --head feature/sight-gates-5a --title "feat(parity): mobs get, look, remove, equip and craft by the player's gates (slice 5a)" --body-file <scratchpad>/pr-body.md
```

Body: the 5a parity table, the slot rule (fill, swap the first uncursed, refuse) over 2 to 6 arms and the golden oracle with its two sanctioned carve-outs, the "Spec points that could not be implemented as written" list, the gate results with counts, the boot check, the playtest outcome, the spec and plan paths, and a line that CI minutes are exhausted this month so the local gate is the merge gate; end with the attribution line. Read back the URL `gh` prints and confirm it says `pruuk/DOGMud`. The owner deploys; do not deploy.

---

## Self-review

- **Spec coverage.** Get: `TooDarkToGet`, `TakeFloorItem` order, dark `GetItemFromFloor` finds nothing, gold, the sweep through the shared body, the player's first-statement refusal, silent mob (T7). Look: `ResolveLook` with every sight rule, player order, mob silent refusals and hidden-name room lines, L6 closed, lookup guard re-key (T8). Remove: `CursedHolds`, busy then curse, `RemoveAllEquipment`, aggregate events dropped, busy text helper, `PermaGear` first, companion refusal (T9). Equip: `CursedRefusal`, curse pass before reservation, every caller inherits (T3, parity row T5), `ChooseWornSlot` over every arm and wrist with rulings 10, 12, 13 (T2, T3), golden pin before the change (T1), `WearInArm` and `EquipItemInArm` with MinStrength, reservation, curse, floor overflow, reveal (T4, T5), the scorer through the helper and the single-slot curse skip (T6), mob `gearup` with a cursed ring (T6). Craft: `TooDarkToCraft`, `CannotSee`, wrappers, `craftableHere` (T10). Guard proven able to fail (T11). Docs, gate, boot (T12). Playtest with an editor-cursed item and PR (T13).
- **Beyond the spec, stated above:** the `HandsRequired` nil guard, the helper's too-many-hands refusal, busy before lookup, removal by identity, `PetUserId` instead of `LookPet`, the crate and container deferral, bauble sweep registrations.
- **Names used across tasks:** `SlotChoice{Slots []WornSlot; Displaced []items.Item}`, `(SlotChoice).apply`, `CursedRefusal(it) string`, `ChooseWornSlot(i, arm) (SlotChoice, string)`, `wear(i, place)`, `wearChosen(i, arm)`, `WearInArm(i, arm)`, `ArmLabel(arm) string`, `EquipItemInArm(actor, name, arm)`, `EquipItemResult.ArmLabel`, `chooserSlotName`, `usesChooser`, `cursedAmong` (itemvalue, a function; characters has a method of the same name on `*Character`, separate packages), `ErrTooDark`, `ErrExploding`, `TooDarkToGet`, `TakeFloorItem`, `ResolveLook`, `LookResolution{Kind, Sight, NamesCreatures, Target, LookAt, ExitName, ExitRoomId, PetUserId}`, `CursedHolds(char, item) (holds, overridden bool)`, `removeWorn`, `RemoveAllEquipment`, `RemoveAllResult{Busy, Removed, Cursed}`, `busyRefusalText`, `TooDarkToCraft`, `CraftResult.CannotSee`; test fixtures `goldenChar`, `goldenItem`, `applyLayout`, `handSlotsInArmOrder`, `fillHands`, `goldenSnapshot`, `goldenIds`, `newGateScene`, `gateItem`, `gateWho`, `gateLights`.
