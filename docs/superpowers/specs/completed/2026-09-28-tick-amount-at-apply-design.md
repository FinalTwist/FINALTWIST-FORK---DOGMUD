# Tick amounts computed when a condition lands

Date: 2026-09-28. Player/mob parity slice 2 of 6 (audit
`project-player-mob-parity-audit-2026-09-28`, rows 1, 2, and the caster-strength
note). Follows the drink unification (#177), which added the zero-amount
fallback to the mob tick.

## Facts verified against source (2026-09-28, master `0ec151356`)

| # | Fact | Where |
|---|---|---|
| 1 | Every tick amount is computed AFTER the condition is queued: `SetTickAmount` (`conditions.go:594`) finds nothing on a first application, so the record arrives with `TickAmount` 0 and the round tick's fallback computes it at scaling 1.0 and caches it | `internal/actions/drink.go:359-379`; `internal/hooks/spell_resolution.go:789-811`, `:1141-1163`, `:1507-1522`; `internal/hooks/condition_tick_amount.go` |
| 2 | On a REFRESH (target already holds the condition) the same `SetTickAmount` lands, so a recast heals with caster scaling while a fresh cast heals at 1.0 | same |
| 3 | Caster scaling today: player caster `combat.SkillMultiplier(spellcasting) * weapon.SpellDamageMultiplier * mutations.GearEffectivenessMultiplier`; mob self-cast `SkillMultiplier` only (no weapon); mob caster on a mob: none (the block runs only `if user != nil`) | `spell_resolution.go:792-800`, `:1144-1152`, `:1511-1512` |
| 4 | `SkillMultiplier` ships 1.0 (`SkillMultiplierBase`) to 3.0 (`SkillMultiplierMax`) | `internal/combat/damage_pipeline.go:23`; `config.yaml:1061-1062` |
| 5 | Only three spells reach a ticking condition: `vital-surge` (condition 32, 5%), `chrysalis-regeneration` and `mass-mend` (33, 8%). The other 15 `tick_pool` conditions come from potions, hazards and moves, all at 1.0 | `conditions/*.yaml` `tick_pool`; `spells/*.yaml` |
| 6 | `applySpellCondition(target spellConditionTarget, ...)`: the target interface has `AddCondition` and `AddConditionMagnitude`; four callers in `spell_resolution.go` | `internal/hooks/light_spell.go` |
| 7 | `events.Condition` carries `DurationMult`, `Magnitude`, `Triggers`; `Condition_ApplyConditions` applies it through `AddConditionMagnitude`, `AddConditionScaled` or `AddCondition` and returns early on a refusal | `internal/events/eventtypes.go:20-40`; `internal/hooks/Condition_ApplyConditions.go:100-110` |
| 8 | Synchronous character-level adds (`Character.AddConditionMagnitude` for former combat conditions such as throttle 89) bypass the event and keep relying on the round-tick fallback | `internal/actions/combat_throttle.go:149`; `condition_tick_amount.go` |

## Owner ruling (2026-09-28)

**A: caster scaling on every cast.** A skilled caster's heal-over-time is as
strong on a first cast as a recast is today, and mob casters scale by the same
formula as players, weapon included. Up to 3x plus the weapon for the three
spells in fact 5.

## Design

**The event carries the scale.** `events.Condition` gains `TickScale float64`;
0 means 1.0.

**The amount is computed where the condition lands.** In
`Condition_ApplyConditions`, after a successful add (fresh or refresh), a
`tick_pool` condition gets
`ComputeTickAmount(holder pool max, TickPercent, TickVariance, TickMin, scale)`
and `SetTickAmount`, with the holder's pool read by one shared helper (the pool
switch now copied at five sites collapses into it). Potions and hazards queue
no scale and land at 1.0, exactly as today.

**One caster formula.** `spellTickScale(caster *characters.Character) float64`
= `SkillMultiplier(spellcasting)` times the equipped weapon's
`SpellDamageMultiplier` times `GearEffectivenessMultiplier(caster.Mutations)`
(when the weapon has one), for player and mob casters alike.

**The spell hook passes it.** `spellConditionTarget` gains
`AddConditionTickScaled(conditionId, scale, source)`, implemented on
`UserRecord` and `Mob` as event doors. `applySpellCondition` sends a
magnitude-scaled light or sight through `AddConditionMagnitude` (unchanged),
a `tick_pool` condition through `AddConditionTickScaled` with
`spellTickScale(caster)`, and anything else through `AddCondition`.

**Dead code goes.** The three `SetTickAmount` blocks in `spell_resolution.go`
and the snapshot block in `actions/drink.go` are deleted.
`fillZeroTickAmount` stays as the fallback for synchronous adds (fact 8),
with its comment updated to say so.

**What changes in play:** Vital Surge, Chrysalis Regeneration and Mass Mend
heal at caster strength on every cast; a mob casting them on itself gains its
weapon multiplier; a mob casting them on another mob now scales at all.
Potions, hazards and moves are unchanged.

## Testing

- Apply-time: a queued `tick_pool` condition with `TickScale` 2.0 lands with
  the amount computed at 2.0 on a FRESH application (red today: 0 until the
  tick), and on a refresh.
- `spellTickScale`: skill 0 and no weapon is 1.0; a weapon with a spell
  multiplier multiplies; player and mob casters with the same skill and
  weapon give the same number.
- Spell path: casting `vital-surge` (seeded) at a fresh target queues an
  event with the caster's scale; the holder's `TickAmount` after apply equals
  the expected amount. Rows for player-on-player, player-on-mob,
  mob-on-self, mob-on-mob.
- Potions: a healing potion still lands at scale 1.0.
- The existing player and mob tick tests pass; the synchronous throttle path
  still ticks via the fallback.
- Repo-root apply-path guard re-keyed for the new door calls. Gate: gofmt, vet,
  build, `go test ./...`, golangci-lint, boot check.

## Out of scope

Mob heal and shield ON a player (audit row 3), damage-over-time length
(row 9), help-area targeting (row 13), shield on a pet (row 15): the spells
slice, next.
