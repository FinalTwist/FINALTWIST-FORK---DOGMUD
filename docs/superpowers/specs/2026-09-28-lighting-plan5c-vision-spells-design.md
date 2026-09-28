# Lighting plan 5c: vision spells, infravision potion, infravision fixed

Date: 2026-09-28. Arc: graded room lighting, plan 5 ("light as play"), slice
5c. Parent spec: `docs/superpowers/specs/2026-09-26-lighting-plan5-light-as-play-design.md`.
Depends on 5b (merged #173).

## Facts verified against source (2026-09-28, master `cf5af4c55`)

| # | Fact | Where |
|---|---|---|
| 1 | Infra reads shapes only at `light <= windowFloor` (1) and `light >= -reach`; that gate is the faint-room bug (light 2 to 12 reads dark to infravision) | `internal/messaging/window.go:55`; `windowFloor = 1` at :14, `windowShiftCap = 24` at :11 |
| 2 | `ComfortDistance(observer, room) (dark, bright)` ignores reach entirely; pure form `comfortDistance(light, strength, blindBelow, dimBelow, dazzleAbove)` | `internal/messaging/comfort.go:19`, `:36` |
| 3 | `SightScoreMultiplier(dark, bright, bal)` = `1 - dark*(1-DarknessCombatPenalty) - bright*(1-DazzleCap)`; `SightMult` composes it with `ComfortDistance` | `internal/messaging/sight_mult.go:18`, `:32` |
| 4 | Melee calls `ComfortDistance` directly, not `SightMult`: 8 calls in 4 functions; `SightMult` has 34 non-test callers | `internal/combat/combat.go:54,55,111,112,161,162,216,217` |
| 5 | Shipped `DarknessCombatPenalty: 0.80`, `DazzleCap: 0.80`; `LightDoublingStep` absent from `config.yaml`, so the Go default 8 is live | `config.yaml` HEAD blob :918, :924; `internal/configs/config.balance.lighting.go:109-110` |
| 6 | `InfraReach()` and `NightVisionStrength()` both return the BEST source: `Conditions.Effect` aggregates `isMax` kinds by max, `mutations.FlagValue` returns the largest rank-scaled value | `internal/characters/vision.go:29-39,63`; `internal/conditions/effects.go:55`; `internal/mutations/mutations.go:603` |
| 7 | `InfraReach()` readers: `messaging.LightBand` (band.go:79), `ParticipantSight` (predicates.go:84), `lightnotice` tracker (tracker.go:260) | grep `InfraReach()` |
| 8 | Glow scaling: `lightSpellApplication` scales only a condition whose `light_strength` is `magnitude`: `base + stat/D1 + skill/D2`, triggers `base + stat/D + skill/D` (knobs 40/10/2 and 2/50/20); `applySpellCondition` routes it to `AddConditionMagnitude` | `internal/hooks/light_spell.go:18`, `:51`; `config.yaml` :942-947 |
| 9 | The condition event already carries `Magnitude` and `Triggers`; either non-zero routes to `AddConditionMagnitude` | `internal/events/eventtypes.go:34-35`; `internal/hooks/Condition_ApplyConditions.go:102-103` |
| 10 | The drink path computes `durationMult = potencyMult * (1 + CraftSkill/100)` (aging phase: fresh 1.0, fermented 1.15, peak 1.30, declining to 0.5) and applies it to duration only, via `AddConditionScaled` | `internal/usercommands/drink.go:255-265`; `internal/items/aging.go:31` |
| 11 | Condition 29 Night Vision: literal `nightvision_strength: 18`, triggerrate 3 real minutes, count 1; no content grants it (admin `setcondition` only) | `_datafiles/world/dogmud/conditions/29-night_vision.yaml` |
| 12 | Condition 65 Cat's Eye Draught: `nightvision_strength: 24` (the cap), item 30047, recipe `cats-eye-draught` (alchemy min 12, 2 Moonpetal + 1 Dustwalk Herb + bottle), stocked by three apothecaries (98, 9592, 9601) | conditions, items, recipes, shops |
| 13 | Condition 85 InfraredVision: literal strength 12, `infra_reach: 30`, secret; granted by `conditionids` to 15 mobs; no player path | `conditions/85-infraredvision.yaml`; grep |
| 14 | Spell discovery gates on `difficulty` (required spellcasting = difficulty at `SpellDiscoverySkillPerDifficulty` 1.0) and draws weighted by difficulty from the elemental/enhancement/mental/vital pool; `cost` is CP | `internal/spells/spells.go:378,411`; `internal/hooks/NewRound_DoCombat_helpers.go:605-623` |
| 15 | Glow is mental 0/35, veil-sight mental 5/35, iron-will 15/45, neural-stun 35/50, sensory-overload 40/80 (difficulty/cost) | `spells/*.yaml` |
| 16 | Rarity tier reads LOW = RARE: Moonpetal 30, Dustwalk Herb 40, Stillwater Black Pearl 20 | `items/materials-40000/` |
| 17 | Pale Lurker (225) carries no items; Blind Stalker (227) drops Serpent Venom Sac 40048; both Ironwind Steppe cave hunters holding condition 85 | `mobs/ironwind_steppe/` |
| 18 | The Purging Draught strips potion effects by a hardcoded id block 54 to 75; shipped potions already sit OUTSIDE it (first condition ids 5, 7, 44, 47, 48, 49, 51, 82) | `internal/usercommands/drink.go:44-47,76` |
| 19 | Free ids: conditions 128, 129; items 30068 (consumables), 40233 (materials) | `tools/id_inventory.py --alloc` |
| 20 | Help page `light` lives in `templates/help/light.template` | `_datafiles/world/dogmud/templates/help/` |

## Owner rulings (2026-09-26 call 3, 2026-09-28 brainstorm; do not relitigate)

1. Infravision sees SHAPES in any light down to `-reach`, and nothing below.
   Natural sight wins wherever it reads better (faces beat shapes).
2. Infravision helps only on the DARK side. A dazzling room costs an
   infravision creature the full dazzle ramp; light stays a weapon against
   cave dwellers.
3. Reach is capped at 50 (`InfraReachCap`), whatever the source.
4. The infra penalty is a linear ramp: `InfraPenaltyFloor + (1 - floor) *
   reach / InfraReachCap`, floor 0.90. Reach 25 reads 0.95, reach 50 reads 1.0
   (no penalty, still shapes: names stay hidden). Ships in 5c.
5. Nightvision sources: strongest wins (unchanged, plan 2's ruling). Infra
   reach sources combine with `lightscale.Combine(LightDoublingStep, ...)`
   and then cap.
6. Both spells scale from the caster (stat plus spellcasting) like glow;
   potions scale by the drink path's existing potency; mutations by rank.
7. Spells are learned by DISCOVERY, never taught by an NPC. Infravision
   costs more CP and needs more spellcasting than nightvision.
8. The infravision potion takes rarer ingredients than the Cat's Eye.

## 1. The infravision model (`internal/messaging`, `internal/characters`)

**Sight decision.** `SightThroughWindow` drops the `light <= windowFloor`
clause. Order stays best to worst: full (shifted dim edge) first, then
natural shapes, then infra shapes when `reach > 0 && light >= -reach`, else
none. Faint rooms (2 to 12) now read shapes to infravision by construction.
`BandThroughWindow` inherits it.

**Sight cost.** Infra lives INSIDE `ComfortDistance`, because melee reads
`ComfortDistance` directly (fact 4) and must not diverge from `SightMult`.
The infra multiplier is expressed as a cap on the dark fraction:

```
infraMult = InfraPenaltyFloor + (1 - InfraPenaltyFloor) * min(reach, cap) / cap
infraDark = (1 - infraMult) / (1 - DarknessCombatPenalty)   // skipped when the dark cap is >= 1 (no dark penalty exists)
dark      = min(dark, infraDark)   when reach > 0 and light >= -reach
```

`SightScoreMultiplier` then yields exactly `max(natural ramp, infraMult)` with
no change to it or to any caller. At shipped knobs reach 1 gives
`infraDark` 0.49, reach 25 gives 0.25, reach 50 gives 0. `bright` is never
touched (ruling 2). The pure form gains `reach`, `infraFloor`, `infraCap`
and `darkCap` arguments; the wrapper reads them from config.

**Reach aggregation.** `InfraReach()` stops taking the best source:
`Conditions.EffectValues(EffectInfraReach) []float64` (new, every held
unexpired record's value, magnitude-aware exactly as `Effect` reads it) and
`mutations.FlagValues(owned, flag) []float64` (new, each mutation's
rank-scaled value) feed `lightscale.Combine(LightDoublingStep, ...)`, then
`min(InfraReachCap)`, then one `math.Round`. `Effect(EffectInfraReach)` keeps
its `isMax` for any other reader. Nightvision strength is unchanged.
Example: mutation 40 + spell 30 + potion 25 reads about 46.

**Knobs** (new, in `config.balance.lighting.go` with `config.yaml` entries):
`InfraPenaltyFloor` 0.90, `InfraReachCap` 50.

**Consequences, intended:** the 15 condition-85 mobs now read shapes in
faint rooms (slice F's hostile scans gate on sight, so they will pick
targets there), and at reach 30 pay 0.96 instead of up to 0.80 in the dark. The
darkness-parity goldens move only where an infravision observer sits in a
room between light 2 and the natural blind edge; each move is proven by a
filtered diff before re-recording.

## 2. One magnitude hook for spells

`lightSpellApplication` becomes `magnitudeSpellApplication`: it finds the
condition's one `UsesMagnitude` effect kind and scales it with that kind's
knob trio, capped by the kind's cap. Duration uses glow's existing duration
trio for all three kinds. A condition with two magnitude kinds is refused at
load (one magnitude per record).

| Kind | Formula (new knobs) | Cap | (100,0) | (130,30) | (175,65) |
|---|---|---|---|---|---|
| `light_strength` | 40 + stat/10 + skill/2 (existing) | none | 50 | 68 | 90 |
| `nightvision_strength` | 4 + stat/12.5 + skill/6.5 | 24 | 12 | 19 | 24 |
| `infra_reach` | 5 + stat/7 + skill/3 | 50 | 19 | 34 | 50 |

Duration at 3-minute triggers: 4 triggers (12 min) for a new character, 9
(27 min) at endgame.

## 3. Content

**Night Vision spell** (`spells/night-vision.yaml`): mental, difficulty 15,
cost 45, single target, `effect_type: condition`, `condition_ids: [29]`.
Condition 29 switches to `nightvision_strength: magnitude`; an admin grant
with no magnitude falls back to the bare-flag default 12.

**Heat Sight spell** (`spells/heat-sight.yaml`, alias `infravision`):
mental, difficulty 35, cost 80, single target, condition 128. Condition 128
"Heat Sight": literal `nightvision_strength: 12`, `infra_reach: magnitude`,
flag `infraredvision`, not secret, triggerrate 3 real minutes like 29 (the
spell sets its trigger count). Condition 85 stays literal for its mobs
(a mob grant carries no magnitude).

**Heat-Pit Organ** (material 40233): the heat-sensing pit of a blind hunter.
Rarity tier 20, component tag `heat-pit`, alchemy vendor category. Drops at
15% from Pale Lurker 225 and Blind Stalker 227.

**Pitsense Tincture** (potion 30068, condition 129): recipe `pitsense-tincture`,
alchemy minimum 30, 1 Heat-Pit Organ + 2 Moonpetal + 1 bottle. Toxicity 25,
value 60, Cat's Eye aging thresholds. Condition 129: literal strength 12,
`infra_reach: magnitude`, triggerrate 1 round, triggercount 400 (Cat's Eye
is 500: stronger, so shorter). Not stocked by any shop in 5c (see Deferred).

**Drink path magnitude.** New item field `magnitude` (the base). When a
potion condition has a magnitude effect, the drink path queues it with
`Magnitude = magnitude * durationMult` and `Triggers = round(triggercount *
durationMult)`, both through the event door (fact 9). Pitsense authors
`magnitude: 20`: fresh at alchemy 30 reads 26, peak at 50 reads 39, peak at
100 caps at 50. A potion carrying a magnitude condition with no `magnitude`
fails load.

**Purge.** Condition 129 must be stripped by the Purging Draught. The purge
moves from the id block to `isPotionEffectCondition(id)`: the block plus an
explicit list naming 129. **Owner call:** the pre-existing leak (fact 18:
potions at 44 to 51 and 82 already escape the purge) can be closed in the
same list now, or filed. Recommendation: close it now, since it is the same
line.

**Text.** Each new condition carries the 5b daylight sentence; the heat ones
add that heat shows a shape, never a face. Help `light` gains an
infravision paragraph (sees heat in any darkness down to its reach, costs
nothing at full strength, does not soften glare) and names the two spells
and the tincture. 80-column wrap, no raw numbers.

## Testing

- Pure tables for `SightThroughWindow` (faint room now shapes; below
  `-reach` none; natural faces still win) and `comfortDistance` (infra
  never touches `bright`; reach 0, 1, 25, 50; dark cap 1.0 guard).
- `InfraReach` combine and cap; `EffectValues` and `FlagValues`.
- `magnitudeSpellApplication` for all three kinds at the three reference
  characters; the two-magnitude-kinds load refusal.
- Drink path magnitude and triggers; missing-`magnitude` load failure;
  purge strips 129.
- Each new null test proven capable of failing before trusting a green.
- Goldens: filtered diff first, then re-record.
- Playtest gate: one scenario. An infravision caster and a nightvision
  caster in a faint room, a pitch-dark cave and a lit tavern, fighting a
  condition-85 mob. Checks shapes versus faces, the lower penalty, and that
  glare still bites.

## Deferred

- **Rare-ingredient restocking** (owner, 2026-09-28): shops gain a small
  chance to restock rarer materials alongside the common baseline, with
  some materials (group-instance drops) gated to a very low chance or none.
  Its own design; the tincture and its Heat-Pit Organ are candidates.
- Echolocation, if ever added, follows the infra model.
- Foragers and shopkeepers carrying light (5b follow-ups, behaviour arc).
- 5d darkness trims to the bottom of the caster's usable band, which is now
  `-reach` for an infravision caster (up to -50).
