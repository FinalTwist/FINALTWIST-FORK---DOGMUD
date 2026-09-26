# Lighting plan 5b: dazzle's teeth

Design for slice 5b of the plan 5 spec
(`2026-09-26-lighting-plan5-light-as-play-design.md`). 5a shipped in #170 and
this slice gives the too-bright band a cost, and gives the dim band the same
cost everywhere it was missing.

Brainstormed with the owner 2026-09-26. The load-bearing ruling:

> Dazzle and dim (shapes) should hurt ANY roll that is opposed or against a
> difficulty. Only voice and sound contests are exempt. Bartering requires
> full sight; `list` should not work without it. Lockpicking is a minigame
> with no roll, so nothing changes there.

---

## Facts verified against source (2026-09-26, master `49bacf2d7`)

| # | Fact | Where |
|---|---|---|
| 1 | Every opposed roll funnels through `combat.RunContest(atkScore, entries)`, which compresses the defence gap and calls `contest.RunWithFloors` with `Balance.ContestFloor`. A repo-root guard forbids any other production caller of the primitives. | `internal/combat/run_contest.go:23`, `contest_floor_guard_test.go` |
| 2 | `internal/contest` deliberately knows nothing about characters: scores arrive fully modified, and it reads no config. | `internal/contest/context.md` |
| 3 | Production callers of `RunContest` (18): `Defuse` (`actions/defuse.go:56`), `RollSubmissionAttempt` (`combat/submission.go:75`), `executeSkillMoveWithRunner` (`combat/skill_moves.go:172`), `AttemptGrapple` (`combat/grapple.go:76`), `plantOnMob`/`plantOnPlayer`/`plantInContainer` (`actions/plant.go:132,277,357`), `shadowPlayer` (`actions/shadow.go:129`), `stealFromMob`/`stealFromPlayer`/`stealFromContainer` (`actions/steal.go:147,366,460`), `Sneak` (`actions/sneak.go:52`), `shadowDetectionRoll` (`usercommands/skill.skullduggery.shadow.go:135`), `spotsHider` (`actions/search.go:44`), `Track` (`actions/track.go:79`), `Go` (`usercommands/go.go:576,595,625`), and the channel funnel below. | `codegraph_callers RunContest` |
| 4 | `AgainstDifficulty` has three sanctioned callers: `Search` (`actions/search.go:96`, the static tiers), `ForageCore` (`forager/forage_core.go:114`), `resolveTrailDetail` (`actions/track.go:344`). | guard exemption table |
| 5 | Crafting rolls through `crafting.RunCraftContest(score, difficulty)` and `RunSalvageContest` (`internal/crafting/difficulty.go:144,152`), the one readers of `CraftFloor`/`SalvageFloor`. | source |
| 6 | The character-aware funnel is `combat.ResolveChannelAttack(shape combatvocab.Attack, side AttackSide, attacker, defender)` → `resolveChannelAttackWithRunner` (`combat/defence_multiplier.go:415,434`). Spells, rhetoric (`actions/combat_taunt.go:177`), counters and `throw` use it. | source |
| 7 | `combat.SituationalAttackMult(attacker, shape)` (`combat/situational.go:36`) is a DECLARED per-channel table (`situational.go:11-31`): prone and resource depletion apply to melee/ranged/specials, not to spell or social. It has no sight row and no defence-side twin. Sixteen special moves call it. | source |
| 8 | Only melee applies a sight multiplier: `DarknessScoreMultiplier(sight messaging.SightDecision, bal)` (`combat/combat_helpers.go:517`): full 1.0, shapes `DarknessShapesCombatPenalty` (0.90), none `DarknessCombatPenalty` (0.80); applied to the attack score (`:580`, `ctx.sourceSight`) and the defence score (`:772`, `ctx.targetSight`). Both knobs ship in `config.yaml:913-928`. | source |
| 9 | Dazzle is a band, not a `SightDecision`: `messaging.LightBand(observer, room) Band` (`band.go:68`) returns `BandDark`, `BandShapes`, `BandFaces`, `BandDazzled`; `BandThroughWindow` (`band.go:46`) splits full sight at `windowDazzleEdge - strength`. `windowDazzleEdge = 75` is a constant (`window.go:18`), deliberately not a knob until something reads it. | source |
| 10 | Stealth contests are built per observer: `CalcSneakScoreVsObserver(sneaker, observer, roomLit)` (`actions/skill_helpers.go:60`) folds the observer's night vision into the hider's score; `CalcDetectionScore(c)` (`:73`) is perception + search and reads NO light. | source |
| 11 | `RunConcentrationContest(casterScore, disruption)` (`combat/run_concentration_contest.go:17`) resolves a caster holding a spell under damage; callers `ExecuteThrottle`, `checkConcentrationBreak`, `processFoldRound`. | source |
| 12 | Bartering is not a roll: `buy.go:532-556` and `sell.go:283` turn the `Bartering` skill into a discount. `List` (`usercommands/list.go:26`) is gated only by `actions.ShopClosedForSleep(room)` (`:31`, defined `actions/sleeping_target.go:78`). No shop command reads sight. | source |
| 13 | The three shipped vision grants: `conditions/29-night_vision.yaml`, `65-cats_eye_draught.yaml`, `85-infraredvision.yaml`; their dazzle edges are 57, 51, 63 (plan 3 amendment table). | source |
| 14 | `Cancel`, `hood`, `unhood` and light equips run `lightnotice.Check` after their action (5a); the notice store has dazzle lines that read true for any observer. | `internal/lightnotice`, 5a |
| 15 | Lockpicking is a minigame (`usercommands/picklock.go`), no contest call. Bartering, `list` and crafting today all work while an activity is in progress (a separate hole, filed). | grep |

---

## Rulings (owner, 2026-09-26; do not relitigate)

1. **Every roll that is opposed or against a difficulty takes the sight penalty**, built into the roller rather than at each site.
2. **Exempt: voice and nerve only.** Taunt, demoralize, rally, warcry, defy. Not exempt: spells, concentration, crafting, salvage, forage, search, track, defuse, steal, plant, special moves, grapple, submission, melee.
3. **The party who needs to see pays.** In stealth contests that is the observer; in a theft the thief, and the victim as observer; in an attack both sides.
4. **Bartering requires full sight**; `list`, `buy` and `sell` refuse below the faces band.
5. Lockpicking is a minigame with no roll; nothing changes.
6. Dazzle ships at the same cost as shapes (0.90) until plan 6 tunes it.

---

## Design

### 1. One multiplier, four bands

`DarknessScoreMultiplier(SightDecision)` is replaced by
`combat.SightScoreMultiplier(band messaging.Band, bal configs.Balance) float64`:

| Band | Multiplier | Knob |
|---|---|---|
| dark | 0.80 | `DarknessCombatPenalty` (existing) |
| shapes | 0.90 | `DarknessShapesCombatPenalty` (existing) |
| faces | 1.0 | |
| dazzled | 0.90 | `DazzlePenalty` (new; validated with the pair: at or above the shapes penalty is NOT required, but within (0, 1]) |

The two existing knob names keep their `Combat` suffix for config compatibility;
their doc comments say they now price every roll. `windowDazzleEdge` becomes
the knob `LightDazzleAbove` (75), read through `configs.Lighting`, validated
above `LightDimBelow` and at most 100, because something reads it now.

The band comes from `messaging.LightBand(c, room)` (plan 3d), which already
composes light, night-vision strength, infra reach and blindness.

### 2. The channel funnel: a sight row and a defence side

`SituationalAttackMult(attacker, shape)` gains a `sight` row in its declared
table: Y for melee, ranged, specials and spell; N for social. It needs the
room, so the signature becomes `SituationalAttackMult(attacker, room, shape)`;
the sixteen special-move callers and taunt pass the room they already hold.

A defence-side twin, `SituationalDefenceMult(defender, room, shape)`, carries
the same sight row (a defender must see a swing, a shot or a spell coming; a
defy against a taunt needs no eyes) and nothing else for now (prone-defender
vulnerability stays melee-only where it is). `resolveChannelAttackWithRunner`
applies it to the defence entries it builds.

Melee's two hand-placed multiplications in `calcAttackScore` and the defence
score path move onto the same two functions, so `combatContext.sourceSight`
and `targetSight` become bands and there is one place a band turns into a
number.

### 3. Score-only sites: one helper, applied to the party who needs to see

`actions.SightMult(c *characters.Character, room *rooms.Room) float64` wraps
`LightBand` and `SightScoreMultiplier`. Applied at:

| Site | Whose eyes | Where the multiplier lands |
|---|---|---|
| `Go`, `Sneak`, `shadowDetectionRoll`, `spotsHider` | the observer | `CalcDetectionScore(c, room)` gains the room and applies it |
| steal ×3, plant ×3 | the thief | the attack score; the victim's notice roll goes through `CalcDetectionScore` |
| `shadowPlayer` | the shadower | the attack score |
| `Search` static tiers, `Track`, `resolveTrailDetail`, `Defuse` | the actor | the score handed to `AgainstDifficulty`/`RunContest` |
| `ForageCore` | the forager | the score; `ForageCore` is pure, so the caller applies it and passes the modified score |
| craft, salvage | the crafter | the score handed to `RunCraftContest`/`RunSalvageContest` |
| `RunConcentrationContest` callers | the caster | the caster score |
| grapple, submission | both | attacker and defender scores |

The hider's side of stealth is unchanged: `CalcSneakScoreVsObserver` already
folds per-observer light into the hider's score.

### 4. Shops

`list`, `buy` and `sell` refuse below the faces band with one line, "You can't
make out the goods well enough to deal.", placed beside the sleep gate. At the
dazzled band they work and the bartering discount is multiplied by
`SightScoreMultiplier(dazzled)`, so a dazzled haggler bargains worse.

### 5. The guard

`contest_floor_guard_test.go` (or a sibling `sight_penalty_guard_test.go`)
gains a rule: every production call of `RunContest`, `AgainstDifficulty`,
`RunCraftContest`, `RunSalvageContest` and `RunConcentrationContest` must sit
in either the sight-applied table (naming the helper call in the same
function) or the sight-exempt table with a reason. The exempt table ships
with the five voice entries. A sabotage step in the plan proves the guard
fails when a site is dropped from both tables.

### 6. Text

- The daylight-cost sentence on conditions 29, 65 and 85 (plan 3 amendment
  convention): the Cat's Eye Draught's text says its dazzle reaches into every
  daylight hour; Night Vision's says nearly all of daylight; InfraredVision is
  secret and gets none.
- `help light` and `help hood` say what dazzle costs (in words: "your aim,
  your guard and your eye for detail all suffer"), and `help chrysalis-glow`
  gains one sentence that a fresh glow can dazzle the caster's own party.
- The dazzle notices in `internal/lightnotice` already read true; unchanged.

### 7. Config

`DazzlePenalty` and `LightDazzleAbove` declared in `config.balance.go`,
defaulted in `config.balance.lighting.go`, written into `config.yaml` beside
the darkness knobs with the "zero reverts to the default" line, committed from
the `git show HEAD:` blob per `dogmud-balance-config`.

---

## Consequences accepted

- A nightvision creature in daylight, and a Cat's Eye drinker after sunrise,
  now fight, search and haggle at 0.90 all day. That is the cost the plan 3
  amendment quantified and the owner accepted ("say so on the tin").
- A torch bearer under a midsummer noon sky is dazzled and pays the same 0.90
  in every roll. The hooded lantern is the answer, as ruled 09-25.
- Overloading a cave with a fresh endgame glow now blinds its residents'
  rolls, which is what "light is a weapon" meant.

## Out of scope

Lockpicking (minigame). The buy/sell/list-while-crafting hole (filed).
Tuning any of the four multipliers (plan 6). The vision spells and potion
(5c). Darkness (5d). Mobs choosing to use light as a weapon (behaviour arc).

## Testing and gates

- Table tests for `SightScoreMultiplier` over all four bands, and the pair
  validator for the new knobs.
- The situational table test (`situational_test.go`) extended with the sight
  row for every channel, including the social N.
- A defence-side test: a dazzled defender's defence score is 0.90 of the
  same defender's at faces, on melee, ranged and spell; unchanged on rhetoric.
- Stealth: a dazzled observer spots a fixed hider less often than a faces
  observer over a seeded run; the hider's score is unchanged.
- Shops: `list` refuses at shapes, works at faces and dazzled.
- The guard's sabotage proof.
- Goldens: the parity and day-cycle goldens sample light, not rolls, and must
  not move.
- Playtest: one scenario, three actors: a nightvision profile fighting and
  searching at noon under open sky (dazzled), the same at dusk (faces), and a
  torch bearer haggling with Trader Onna at noon; read every line.
