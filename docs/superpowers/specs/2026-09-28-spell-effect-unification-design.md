# Spell effect unification (parity slices 3a and 3b)

Date: 2026-09-28. Player/mob parity slices 3a (harmful) and 3b (helpful) of
seven; audit rows 3, 9, 13, 15 (`project-player-mob-parity-audit-2026-09-28`).
One spec, two plans, two PRs. 3a ships first.

## Facts verified against source (2026-09-28, master `441495b32`)

Pairings: **PM** player to mob, **PP** player to player, **MS** mob to self,
**MM** mob to mob, **MP** mob to player. All in `internal/hooks/spell_resolution.go`
unless stated.

| # | Fact | Where |
|---|---|---|
| 1 | Effects are applied by three switches plus one self path: `applyMobEffect` (909, serves PM and MM, `user` nil for MM), `applyPlayerEffect` (999, PP), an inline switch in `resolveMobSpellAgainstPlayer` (1616, MP), and `applyMobSelfEffect` (1489, MS) | file |
| 2 | Arms present: damage in PM/MM, PP, MP (not MS); dot in PM/MM, MP (PP falls to default); knockdown in PM/MM, MP (PP default); condition in all; heal in PM/MM, PP, MS (MP default); shield in PP, MS only; purge in PP only; charm in PM only | per switch |
| 3 | Help spells skip the defence contest in PM (401), PP (via `resolveSpell` 168) and MM (1559); **MP has no skip**, so a mob buffing a player is contested, then heal and shield fall to default and apply nothing (audit row 3) | lines |
| 4 | MM dot reads its duration from a nil caster: skill 0, stat 100 (646-651) (audit row 9) | `applyMobEffect_dot` |
| 5 | Mob HelpArea target filling is MISSING (player side 108-124); player HelpArea takes any charmed mob (`IsCharmed()`, no owner check) | 108-124 |
| 6 | No shield arm on a mob target (PM/MM), so a shield on a charmed pet applies nothing (audit row 15) | `applyMobEffect` |
| 7 | A landed MM damage/dot/knockdown/condition is silent (every line gated on `user != nil`); MM has no aggro; PP and MM never call `RecordSpell`; only PM calls `maybeInterruptSpellOnMob`; MP knockdown skips `cancelDamageConditions`; MP damage skips its aggro commit on a defensive crit (1625) | lines |
| 8 | No spell path calls `RecordAssaultCrime(user, mob, room)` or `SeedAggression(user, mob, room, freshAggro)`; melee (`attack.go:246`, `melee_target.go:223`), `shoot.go:214`, `target.go:37` and `throw.go:144` do | `internal/actions/aggression.go:25,94` |
| 9 | PP's heal crit boost and shield x1.5 crit are unreachable: help spells never enter the contest, the only source of a crit | `applyPlayerEffect` 1091-1256 |
| 10 | Shared helpers: `calcSpellDamageForCharacter` (`combat_shared_helpers.go:37`), `scaleSpellDamageByDefence` (366), `calcSpellDuration` (37), `applySpellCondition` / `spellTickScale` (tick slice #178), `spellAttackSideFor` (336), `runSpellChannelAttack` (323), `sendSpellChannelDefenceMessages` (519), `spellAudience` (`spell_audience.go:16`), `mobDisplayName`, `fireSpellCounterTier`, `dispatchItemProcs`, `cancelDamageConditions`, `mobAreaHarmTargets` (`mob_area_harm.go:32`) | files |
| 11 | `hooks` imports `actions`; the helpers live in `hooks`, so the appliers live in `hooks` and take `actions.Actor` targets | `go list` |
| 12 | Guards pinning this file: `condition_apply_path_guard_test.go` keys `spell_resolution.go|<line>` rows; `sight_penalty_guard_test.go:117-120` exempts the four resolvers by `file|func`; `channel_defence_routing_test.go:58` AST-parses this file and requires exactly the four named resolvers to call `runSpellChannelAttack` once each; `messaging_surface_guard_test.go:1286-1293` holds `hooks/spell_resolution.go|<literal>` rows with exact set equality; 64 direct test calls in 11 hooks test files name these functions | repo |

## Owner rulings (2026-09-28)

1. **Full unification** (option B): one applier per effect type, fed by thin
   resolvers; split into 3a harmful and 3b helpful.
2. **Harmful spells are crimes.** A player's harmful spell on a mob records the
   assault and seeds aggression exactly as melee does, through
   `RecordAssaultCrime` and `SeedAggression`.
3. **The dead heal and shield crit boosts are deleted.** Help spells do not
   crit; nothing a player sees changes.

## Architecture

**One context, one dispatcher.** Each resolver, after its contest step, builds
a `spellEffectCtx` per target: caster `*characters.Character`, caster
`actions.Actor` (nil-safe for narration), target `actions.Actor`, room, spell,
magnitude and `out` (the contest result, zero for uncontested help spells).
`applySpellEffect(ctx)` switches on the effect type and calls one applier per
type. `applyMobEffect`, `applyPlayerEffect`, the MP inline switch and
`applyMobSelfEffect` become calls to it (MS: target is the caster).

**The resolvers stay where they are.** `resolveAgainstMob`,
`resolveAgainstPlayer`, `resolveMobSpellAgainstMob` and
`resolveMobSpellAgainstPlayer` keep their names and file, each calling
`runSpellChannelAttack` once, so `channel_defence_routing_test.go` and the
sight exemptions hold. What they share moves into the ctx path: help-spell
contest skip (all four, fixing MP), backfire narration (caster line and room
line for every caster kind), `RecordSpell`, the counter tier, and
`maybeInterruptSpellOnMob` generalised to any target that is casting.

**Narration per audience, not per caster kind.** Every applier narrates
through `spellAudience`/`SendTrio`, which already sets `NoLine` for whichever
side is a mob, so a landed MM spell is visible to players in the room. Mob
names go through `mobDisplayName`.

**Aggro and crime in one place.** A harmful applier commits target-to-caster
aggro for every pairing (including MM and the MP defensive-crit case) and, for
a player caster on a mob target, calls `SeedAggression` (which records the
crime on a fresh engagement) exactly as `throw` does.

**Old names survive as thin wrappers** until each test caller is migrated, so
every commit stays green; the last commit of each PR deletes the wrappers that
no test still names.

## 3a harmful: damage, dot, knockdown, drain

- **damage:** one applier from PM's arm; PP and MP arms deleted.
  `cancelDamageConditions`, `on_spell_hit` procs, aggro (including on a
  defensive crit), crime for player-on-mob.
- **dot:** one applier; duration from the caster's own cast skill and stat via
  `GetSpellStatAndSkill` (Manifestation-aware, as MP does today), so MM stops
  reading skill 0 and stat 100 (row 9); PP gains the arm it lacked.
- **knockdown:** one applier; PP gains it; MP gains `cancelDamageConditions`.
- **drain_area:** stays in `resolveMobDrainArea` (its own `ExecuteSkillMove`
  contest); only its narration moves to the shared helpers.
- **Parity table:** each harmful effect driven through PM, PP (PvP enabled in
  the fixture), MM and MP with the same caster stats and spell asserts the
  same damage or duration, aggro committed, `RecordSpell` called, a room line
  delivered, and (PM only) the crime and aggression seeded.

## 3b helpful: condition, heal, shield, area help, purge

- **condition:** one applier over `applySpellCondition` (which already scales
  ticks by `spellTickScale`); MP gains nothing new here except going through it.
- **heal:** one applier; MP gains it (row 3); the dead PP crit deleted
  (ruling 3); `events.Healed` fires when the healer is a player.
- **shield:** one applier; PM, MM and MP gain it (rows 3, 15); the dead PP crit
  deleted.
- **Help skip on MP:** a mob's help spell on a player is uncontested, as on
  every other pairing (row 3).
- **Area help:** one target filler shared by player and mob casters (row 13):
  players in the room plus mobs charmed BY THE CASTER (or, for a mob caster,
  its allies by the same rule the mob's own AI uses; the plan reads that rule
  from source), fixing "any charmed mob" (fact 5).
- **purge:** one applier over the existing purge body; mob targets gain it.
- **Parity table:** each helpful effect through PM, PP, MS, MM, MP asserts the
  same heal amount, shield value, duration and condition queued; MP is
  uncontested.

## Testing and gates

Parity tables per PR (above), written red first where a pairing lacked the
branch. Guard rows in fact 12 re-keyed or collapsed as sites merge; the
messaging-surface rows follow the narration change and are re-derived, not
hand-edited. Every commit green. Full gate (gofmt, vet, build,
`go test ./...`, golangci-lint new-from-merge-base), boot check, and a
playtest per PR: 3a a player casting harm in a guarded town (crime reaction)
and a mob caster against a player; 3b a player healing a party member and a
pet, and a mob healer.

## Out of scope

Mob casters casting on themselves with purge-affliction (Go hook differs),
charm for mob casters (needs an owner), identify and summon (player-only by
nature), and any new crit mechanic for help spells.
