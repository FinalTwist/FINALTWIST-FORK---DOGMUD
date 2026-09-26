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
6. **The penalty is a ramp, not a band** (owner, 2026-09-26, spitballed and
   adopted): it grows with the distance from the observer's own comfort band,
   linearly, to a cap of 0.80 on both sides. Dazzle's cap equals dark's. The
   tiers keep governing what you can SEE (names, shapes, refusals); the ramp
   governs what you can DO.

---

## Design

### 1. One multiplier, a ramp on each side of comfort

Two functions replace `DarknessScoreMultiplier(SightDecision)`, split where
the knowledge lives:

**`messaging.ComfortDistance(observer, room) (dark, bright float64)`** owns
the geometry, beside `LightBand`. For the observer's shifted window (edges
`blind = LightBlindBelow - strength`, `dim = LightDimBelow - strength`,
`dazzle = LightDazzleAbove - strength`, top = 100):

```
width = dim - blind                                             (25 at the shipped knobs)
light in [dim, dazzle)   → (0, 0)                              comfortable
light <  dim             → dark   = min(1, (dim - light) / width)
light >= dazzle          → bright = min(1, (light - dazzle) / width)
Perception Blinded       → (1, 0)
```

Each is the fraction of the way from the comfort band to the cap: 0 at the
band's edge, 1 one width beyond it, and 1 past that. The bright ramp is as
wide as the dark one, so for normal eyes it reaches the cap at exactly the
top of the scale (75 + 25 = 100), and for a shifted window it reaches it at
`dazzle + 25` (76 for nightvision 24): a strong window is punished by excess
light as fast as it is helped by faint light. Exactly one of the two is
non-zero. Infra reach does not soften the ramp: heat-sense gives shapes to
SEE by, not steadiness to act by.

**`combat.SightScoreMultiplier(dark, bright float64, bal) float64`** owns the
knobs:

```
mult = 1 - dark * (1 - DarknessCombatPenalty) - bright * (1 - DazzleCap)
```

| Knob | Ships | Meaning |
|---|---|---|
| `DarknessCombatPenalty` | 0.80 (existing) | the multiplier at and below the blind edge |
| `DazzleCap` | 0.80 (new) | the multiplier at and above the top of the scale |
| `LightDazzleAbove` | 75 (new; today the constant `windowDazzleEdge`) | where the comfort band ends |
| `DarknessShapesCombatPenalty` | RETIRED | its 0.90 is now the ramp's midpoint |

`DarknessCombatPenalty` keeps its name for config compatibility; its doc says
it prices every roll. `DazzleCap` is validated in (0, 1]; `LightDazzleAbove`
above `LightDimBelow` and at most 100. The retired knob is deleted from
`config.balance.go`, its validator and `config.yaml` (grep the yaml TAG, per
`dogmud-balance-config`).

Worked values at the shipped knobs (normal eyes: dim 50, blind 25, dazzle 75;
nightvision 24: 26, 1, 51):

| Situation | Light | Multiplier |
|---|---|---|
| Normal eyes, dim street at night | 37 | 0.90 |
| Normal eyes, just above blind | 26 | 0.81 |
| Normal eyes, torch at midsummer noon | 75 | 1.00 |
| Normal eyes, endgame glow flash | 90 | 0.88 |
| Nightvision 24, equinox noon | 70 | 0.85 |
| Nightvision 24, endgame glow flash | 90 | 0.80 |
| Cat's Eye drinker at midsummer noon | 73 | 0.82 |

The tiers (`SightDecision`, `LightBand`) are unchanged and keep governing
what an observer can see and the notices they read.

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
`ComfortDistance` and `SightScoreMultiplier`. Applied at:

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
make out the goods well enough to deal.", placed beside the sleep gate. Above
it they work and the bartering discount is multiplied by `SightMult`, so a
dazzled haggler bargains worse in proportion to the glare.

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

`DazzleCap` and `LightDazzleAbove` declared in `config.balance.go`, defaulted
in `config.balance.lighting.go`, written into `config.yaml` beside the
darkness knobs with the "zero reverts to the default" line;
`DarknessShapesCombatPenalty` deleted from all three places and its pair
validator reduced to the single-knob rule. Committed from the `git show HEAD:`
blob per `dogmud-balance-config`.

---

## Consequences accepted

- A nightvision creature in daylight, and a Cat's Eye drinker after sunrise,
  now act at about 0.82 to 0.85 all day, worse than the flat 0.90 first
  proposed. That is the cost the plan 3 amendment quantified and the owner
  accepted ("say so on the tin"), now priced in proportion.
- A torch bearer under a midsummer noon sky is dazzled by a point or two and
  pays almost nothing, which is the "narrow consequence" ruled 09-25; a fresh
  endgame glow in a cave takes its night-eyed residents to the 0.80 cap,
  which is what "light is a weapon" meant.
- Cave residents at light 25 to 35 fight at 0.81 to 0.86 instead of a flat
  0.90: a small shift for plan 6 to retune if it shows.
- The graded light scale now produces graded outcomes instead of three; every
  point of light a lantern, a hood or a cloud moves is felt in a roll.

## Out of scope

Lockpicking (minigame). The buy/sell/list-while-crafting hole (filed).
Tuning any of the four multipliers (plan 6). The vision spells and potion
(5c). Darkness (5d). Mobs choosing to use light as a weapon (behaviour arc).

## Testing and gates

- Table tests for `ComfortDistance` (both edges, both caps, the shifted
  window at strength 24, Blinded, the top of the scale) and for
  `SightScoreMultiplier` (the worked-values table above, to two decimals),
  plus the validators for the new knobs and the deletion of the old one.
- The situational table test (`situational_test.go`) extended with the sight
  row for every channel, including the social N.
- A defence-side test: a defender at light 90 defends at 0.88 of the same
  defender's score at 60, on melee, ranged and spell; unchanged on rhetoric.
- Melee parity: at the old band midpoints (light 37 and 15) the new ramp
  gives 0.90 and 0.80, so the melee census tool's numbers move only where the
  ramp differs from the band; the plan records the census before and after.
- Stealth: a dazzled observer spots a fixed hider less often than a faces
  observer over a seeded run; the hider's score is unchanged.
- Shops: `list` refuses at shapes, works at faces and dazzled.
- The guard's sabotage proof.
- Goldens: the parity and day-cycle goldens sample light, not rolls, and must
  not move.
- Playtest: one scenario, three actors: a nightvision profile fighting and
  searching at noon under open sky (dazzled), the same at dusk (faces), and a
  torch bearer haggling with Trader Onna at noon; read every line.
