# Drink path unification: mobs drink through the player's path

Date: 2026-09-28. Owner-ordered followup after lighting 5c (#176).
Owner: "put mobs on the player one."

## Facts verified against source (2026-09-28, master `8c6561c5a`)

| # | Fact | Where |
|---|---|---|
| 1 | The player drink body is 425 lines; the mob one is 49 | `internal/usercommands/drink.go`, `internal/mobcommands/drink.go` |
| 2 | The mob path has a grapple gate, a drinkable check, `CancelConditionsWithFlag(Hidden)`, `UseItem`, one room line, and per condition either `items.PotionMagnitudeApplication` (5c) or plain `mob.AddCondition`. It has no busy gate, bandolier lookup, toxicity gate or accrual, aging or potency, spoilage, tick snapshots, quest notify, or special items | `internal/mobcommands/drink.go` |
| 3 | The player path's special items: Purging Draught 30052 (`applyPurgeEffects`, `purgeableConditionIds`), Ysolde's Purge 40109, Bloom Wafer 40108, Catalyst of Unmaking 30067, Phial of Second Birth 40181 | `drink.go:25-122`, `:326-425` |
| 4 | `applyPurgeEffects` takes a `*users.UserRecord` | `drink.go:99` |
| 5 | Every mechanic the player path uses is character-level: `FindInPotions`, `UseItemFromPotions`, `FindInBackpackWhere`, `GetToxicityMax`, `AddToxicity`, `AddBloomAddiction`, `BloomSeedNewMutation`, `BloomAdvanceMutation`, `ScourMutations`, `GrantRandomMutationRare`, `Conditions.SetTickAmount` | `internal/characters/` |
| 6 | Player-only pieces: `refuseWhileBusy` (checks `Character.IsActing()`, a character-level test), private `user.SendText` lines, and `questengine.Notify("command", ...)` with a `UserId` | `usercommands/busy_refuse.go:18`; `drink.go:248-253` |
| 7 | `actions.Actor` has `SendText` (a no-op for mobs), `AddCondition`, `GetUserId`, `IsPlayer`, `GetRoom`, `GetCharacter`; it has no scaled or magnitude condition door | `internal/actions/actor.go` |
| 8 | `UserRecord` has `AddCondition`, `AddConditionScaled`, `AddConditionMagnitude`; `Mob` has `AddCondition` and `AddConditionMagnitude` but no `AddConditionScaled` | `internal/users/userrecord.go:422,437,462`; `internal/mobs/mobs.go:859,873` |
| 9 | Precedent: `actions.Buy(buyer Actor, opts)` and `actions.InitiateCast(actor, ...)` are shared bodies with thin command wrappers; `actions` already imports `questengine` (`buy.go`), `mutations`, `items`, `conditions`, and `questengine` does not import `actions` (no cycle) | `internal/actions/`; `go list -deps` |
| 10 | Mob drink callers: `mobcommands.go:42` registers it; the AI companion issues `drink <target>` (`modules/aicompanion/actions.go:418`, `combat.go:463`); the survival planner returns `drink <potion>` (`internal/planners/survival.go:51`) | grep |
| 11 | Existing drink tests: `usercommands/drink_purge_test.go`, `drink_purge_shipped_test.go`, `drink_scour_test.go`, `mobcommands/drink_magnitude_test.go` | `ls` |
| 12 | The repo-root guard `condition_apply_path_guard_test.go` allowlists condition-apply call sites by `file|line`; moving drink moves its keys | repo root |
| 13 | A condition applied through the event queue starts with `TickAmount` 0 (the drink path's snapshot is a no-op on first application because the record is not in the list yet). The PLAYER tick fills a zero amount for any `tick_pool` condition (`ComputeTickAmount(maxPool, TickPercent, TickVariance, TickMin, 1.0)` then `SetTickAmount`); the MOB tick only acts when `TickAmount != 0`, so a mob's heal-over-time never heals (potions 5, 6, 7, 47, 50; spells 32, 33; throttle 89; room hazards) | `internal/hooks/NewRound_UserRoundTick.go:312-325`; `internal/hooks/NewRound_MobRoundTick.go:222` |

## Owner rulings (2026-09-28)

1. Mobs drink through the player's path. One body, no second copy.
2. **Every special potion applies fully to a mob, AI companion included**
   (Wafer, Catalyst, Phial, Ysolde's, the Purging Draught). "These pots are
   expensive, so if someone wants to take the time to use them on a
   companion, go for it."
3. **The mob round tick fills a zero tick amount exactly as the player tick
   does** (fact 13), so the merge delivers potions that actually heal. The
   proper fix, computing the amount when the condition event is applied for
   both sides (which also restores first-cast caster scaling for players and
   so changes balance), is its own "ticks" slice.

## Design

**Shape.** `actions.Drink(actor Actor, rest string) DrinkResult` holds today's
whole player body with its rules unchanged: busy and grapple gates, the
bandolier-then-backpack lookup (drinkable-first, unfiltered fallback), the
toxicity gate and its detox bypass, spoilage, aging and crafter potency,
magnitude potions, tick snapshots, the quest `command` notification (player
actors only, `GetUserId() > 0`), and every special item. The special-item
helpers and constants move with it; `applyPurgeEffects` and
`purgeableConditionIds` take the actor.

`usercommands.Drink` and `mobcommands.Drink` become thin wrappers: build the
actor, call `actions.Drink`, return. Neither touches a condition, toxicity or
an item.

**Condition doors.** `Actor` gains `AddConditionScaled(id, mult, source)` and
`AddConditionMagnitude(id, triggers, magnitude, source)`. `Mob` gains
`AddConditionScaled`, the event-door twin of the user one (queues
`events.Condition` with `DurationMult`), so a mob's potion start lines narrate
through `Condition_ApplyConditions` the way a player's do.

**Result.** `DrinkResult` reports what happened: `Drank bool`, the item, and a
refusal reason when it did not (`RefuseBusy`, `RefuseGrappled`, `RefuseNotFound`,
`RefuseNotDrinkable`, `RefuseToxicity`). Wrappers ignore it today; the survival
planner and the AI companion can read it later instead of the silent failure
they get now.

**Narration.** The drinker's private lines go through `actor.SendText`, a no-op
for mobs, so a mob gets no "you feel" text. Room lines stay the sight-aware
`SendTextVisual` calls they are today, with the actor's name in the right
colour (`username` for a player, `mobname` for a mob), chosen from
`IsPlayer()`.

**What changes for mobs and the AI companion:** toxicity accrues and can
refuse a drink; an aged or crafted potion scales by potency; a spoiled one
brings nausea and triple toxicity; the busy gate applies; bandolier potions are
found first; the purge strips potion effects; the Wafer, Catalyst and Phial
change mutations and Bloom addiction. Nothing changes for players.

**Mob tick parity (ruling 3).** The zero-amount fill-in moves out of
`NewRound_UserRoundTick` into one helper both ticks call, taking the
character and the condition: for a `tick_pool` condition whose `TickAmount`
is 0, compute it from the holder's pool max with scaling 1.0 and cache it.
`tickMobConditions` calls it before its `TickAmount != 0` test. The player
tick's behaviour is unchanged. Effect: every mob heal-over-time, damage over
time and room hazard now ticks (potions 5, 6, 7, 47, 50; Vital Surge and
Chrysalis Regeneration on a mob or pet; throttle 89).

**Guard against re-forking.** A repo-root test reads both wrapper files and
fails if either calls a condition door (`AddCondition*`), `AddToxicity`,
`UseItem*`, `ScourMutations`, or `AddBloomAddiction`. Only `actions` may.
Proven able to fail by a temporary violation.

## Testing

- One table drives the same potion through a `UserActor` and a `MobActor`
  and asserts identical mechanics: toxicity after, conditions queued with
  their multiplier or magnitude, item consumed, refusal reason. Rows:
  healing salve, an aged crafted potion, a magnitude potion (Pitsense),
  a spoiled potion, one past toxicity tolerance, the Purging Draught,
  Ysolde's, the Wafer, the Catalyst, the Phial.
- The four existing drink tests move to `internal/actions` beside the body,
  unchanged in what they assert.
- `Mob.AddConditionScaled` queues an event carrying the multiplier.
- Mob tick parity: a mob holding a `tick_pool` heal condition with
  `TickAmount` 0 gains health on its next trigger, and a damage-over-time one
  loses health; red before the fix. The player tick's existing tests pass
  unchanged.
- The re-fork guard, proven able to fail.
- Guard keys in `condition_apply_path_guard_test.go` re-keyed to the moved
  call sites.
- Gate: gofmt, vet, build, `go test ./...`, golangci-lint new-from-merge-base,
  boot check. A short playtest: an AI companion given a healing potion past
  its toxicity tolerance, and a Pitsense Tincture.

## Out of scope

- `eat`, `use`, `throw` and any other player/mob split: the concurrent
  "looks unified but isn't" audit ranks them; each confirmed one gets its
  own slice.
- Teaching the survival planner or the companion to react to `DrinkResult`.
