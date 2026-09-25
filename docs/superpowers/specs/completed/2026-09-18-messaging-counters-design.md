# Messaging M4, counters slice: the counter answers the defence that won

Status: owner-approved design, 2026-09-18. Spec awaiting owner review.
Follows M4b-2 (`2026-09-18-messaging-m4b2-axes-design.md`, PR #144), which
carried this slice as "Out of scope, carried": re-keying the counter pools to
the winning defence, the missing parry and block counter text, the area gate,
and one playtest.

## Facts verified against source

All read from master `f53b07d76` on 2026-09-18.

### The counter tier today

| Fact | Value | Source |
|---|---|---|
| The primitive | `ExecuteCounter(defender, attacker, shape combatvocab.Attack, sameRoom bool) CounterResult` | `internal/combat/counter.go:90` |
| The counter-swing | always `Melee(TargetSingle)`, strength plus the counterer's combat skill, `IsCounter: true`, priced by `CounterDamagePercent` (shipped 0.5, 0 = off) | `counter.go:113-127`, `_datafiles/config.yaml:649` |
| Pool selection | `counterPoolFor(shape)` keyed on the ORIGINAL attack's type: ranged and thrown to `counter-ranged`; spell with social damage to `counter-defy`; other spells to `counter-quell`; rhetoric to `counter-defy`; default `counter-melee` | `counter.go:162-175` |
| `CounterResult.Shape` | the original attack, "kept for pool selection" | `counter.go:24-26` |
| Narration render | `fillCounterMessages` calls `items.RenderDefenseMessage(counterPoolFor(result.Shape), ...)`; generic Go fallback when the pool is not loaded | `counter.go:195-212` |
| Counter-taunt | `executeCounterTaunt(counterer, target)` in actions, unexported; `counterTauntExit` wires it at taunt's defy-crit exit and dispatches with `CategoryTauntSuccess`; narration from `combat.BuildCounterTauntMessages` off `counter-defy` | `internal/actions/combat_counter.go:151,229`, `combat_taunt.go:339`, `counter.go:254` |
| Why the counter-taunt lives in actions | taunt resolution is in actions, which imports combat; combat cannot call it | `combat_counter.go:11-15` |
| hooks imports actions | yes, already (`SendCounterTrio`) | `internal/hooks/counter_tier.go:10` |
| Skill-move exit | `counterSkillMoveExit(actor, defender, move, shape, sameRoom)` refuses `!move.Defence.DefensiveCrit || move.IsCounter`, else calls the primitive | `combat_counter.go:56-63` |
| Spell exit | `fireSpellCounterTier(room, out, shape, defender, caster, defenderUser, casterUser)` refuses `!out.DefensiveCrit`, calls the primitive with `sameRoom=true`, dispatches via `actions.SendCounterTrio` | `counter_tier.go:35-58` |
| Skill-move exit call sites | 12: bash, drain (single, `:136`), drain area (`:310`), fire (`:418`, `!crossRoom`), gore, hamstring, kick, maul, pounce, rake, throttle, trip | grep `counterSkillMoveExit(` in `internal/actions` |
| Spell exit call sites | 4: `resolveAgainstMob` (`:461`), `resolveAgainstPlayer` (`:979`), `resolveMobSpellAgainstMob` (`:1547`), `resolveMobSpellAgainstPlayer` (`:1772`), each passing `spellData.Attack()` | `internal/hooks/spell_resolution.go` |
| Shapes at the exits | every special move `Melee(Single)`; fire `Ranged(Single)`; drain area `Spell(DamagePhysical, TargetArea)`; spells carry their authored targeting | same sites |
| Throw | `Thrown(TargetArea)`, resolves through `ResolveChannelAttack` directly, has no counter exit | `internal/usercommands/throw.go:328` |
| Basic melee swing | not the counter tier: `applyCritEffects` gives parry crit a riposte, dodge crit an auto-trip, block crit an auto-bash, all Go literals | `internal/hooks/combat_shared_helpers.go:253-263,292-330` |
| Drain area's counters | `DrainAreaPlayerResult.Counter` filled per victim (`:310`); consumed by one loop in the mob drain dispatcher | `combat_drain.go:202,310`; `spell_resolution.go:1451-1453` |
| Who dispatches skill-move counters | 10 usercommands call `actions.DispatchCounterMessages(actor, res.Counter)` after their own outcome text | `internal/usercommands/{bash,drain,gore,kick,maul,pounce,rake,shoot,throttle,trip}.go` |
| Where the winning defence is recorded | `ChannelDefenceResult.Defence` (`out.Defence = winner`, `:551`); `DefensiveCrit = DamageMultiplier == 0` set after `Defended = true` (`:611-614`); `SkillMoveResult.Defence` is that struct | `internal/combat/defence_multiplier.go:226,551,614`; `skill_moves.go:49-52` |
| Defensive crit rule | `DefenseContestCrit(-res.Margin, res.DefenseRoll)` against `DefenseCritBar()`; a floored save never crits | `defence_multiplier.go:640-652`; `crit_bar.go:54` |

### Eligibility and shipped content

| Fact | Value | Source |
|---|---|---|
| Eligibility rows | melee/physical: dodge, parry, block; ranged/physical: dodge, block; thrown/physical: dodge, block; spell/physical: dodge, block; spell/mental: quell; spell/social: defy; rhetoric/social: defy; none/non_harm: none | `internal/combatvocab/attack.go:41-48` |
| Targetings | self, single, multi, area | `combatvocab/vocab.go:55-58` |
| `DefenceNone` | `""` | `vocab.go:66` |
| Shipped spells by axes | none/non_harm: 4 area, 12 self, 21 single; spell/mental/single 9; spell/physical/area 7; spell/physical/single 5; spell/social/single 1 (charm) | `_datafiles/world/dogmud/spells/*.yaml`, counted |
| Store key | `items.DefencePool`; `DefencePoolFor(d combatvocab.Defence) DefencePool` is `DefencePool(d)`; four counter constants `CounterPoolMelee/Ranged/Quell/Defy` | `internal/items/defensive_messages.go:15-39` |
| Loader | `LoadAllFlatFiles[DefencePool, *DefenseMessageGroup](dataPath + "/defense-messages")`; no fixed file list, every file in the directory loads | `internal/items/itemspec.go:803` |
| Validator | each of weak/normal/heavy must carry actee, actor, observer lists of at least 5 non-empty lines of equal length | `defensive_messages.go:78-116` |
| Shipped pool files | block, counter-defy, counter-melee, counter-quell, counter-ranged, defy, dodge, parry, quell | `ls _datafiles/world/dogmud/defense-messages` |
| Lines per counter pool | 45 (3 bands x 3 roles x 5) | counted |
| `counter-melee` wording | attack-neutral ("the opening", "your guard", "failed attack"); its header already names dodge, parry and block crits as its triggers | `counter-melee.yaml:1-7` |
| `counter-ranged` wording | 27 of 45 lines name the shot, aim, weapon or shooter | grep, counted |
| `counter-quell` header | "a quell crit on a mental working, or a dodge or block crit on a physical one"; "Never a generic riposte line pasted under a spell (the owner's hard requirement)" | `counter-quell.yaml:1-5` |
| `counter-defy` wording | 11 lines say "taunt"; ledger row 5 already records this as misleading when the exchange was a charm | `counter-defy.yaml`; `docs/superpowers/audits/messaging-m6-content-ledger.md:89` |
| Test fixture | `MinimalDefenseMessageFixture` seeds the five defence pools plus the four counter constants | `internal/items/test_helpers_combat.go:55-61` |
| Golden | `internal/narration/testdata/stores/defense_messages.golden`, built from every yaml key in the directory; 9 `counter-melee|` rows, 9 `counter-ranged|` rows, plus `melee|<pool>|` rows for every pool; `-update` regenerates | `internal/narration/snapshot_test.go:113,483-486` |
| Tests that name the retiring constants | `combat/counter_social_pool_test.go` (two tests), `items/defence_pool_test.go:22`, `items/defensive_messages_newly_defendable_test.go:135`, `items/test_helpers_combat.go` | grep |
| Tests that pin the tier | `combat/counter_test.go` (reach gate, no recursion, countered-party economy, knob); `hooks/counter_tier_test.go` (riposte knob, melee trio unchanged, four spell quadrants, narration from counter-quell); `actions/counter_tier_test.go`, `actions/counter_trio_test.go`, `combat/counter_narration_test.go`, `hooks/counter_seam_wiring_test.go` | file list |
| Root line-number guard | `condition_apply_path_guard_test.go` at the repo root allowlists lines in `spell_resolution.go` and `combat_*.go`; any edit above an allowlisted line re-keys it | M4b-2 execution notes |
| Docs naming the pools | `internal/combat/context.md:1649`, `internal/items/context.md:419-428`; the counter source comments; `docs/superpowers/audits/messaging-m6-content-ledger.md` row 5 and its last row 33 | grep |
| Patch notes | `docs/PATCH_NOTES.md`, newest entry first, dated headings, prose without numbers | file head |

## Goal

A counter is narrated by the defence that earned it, not by the attack it
answered. Parry and block get their own counter text. Words answer words:
every defy crit counter-taunts. Area attacks earn no counter. One playtest
covers the whole slice.

## Rulings

Owner, 2026-09-17 (M4b-2 spec ruling 5) and 2026-09-18 (this session):

1. Counter pools are keyed by the defence that won. Five pools:
   `counter-dodge`, `counter-parry`, `counter-block`, `counter-quell`,
   `counter-defy`. `counter-ranged` retires.
2. Area attacks stop earning counters, with targeting carried on the attack
   so the gate cannot be bypassed by omission.
3. Every defy crit answers with a counter-taunt, charm included. The
   counter-defy text must read for a charm as well as a taunt: "you tried to
   charm me, I defied it and mocked the attempt", in generic wording.
4. The dodge and block pools are written attack-agnostic, because a dodge or
   block crit can answer a swing, a point-blank shot, or a hurled physical
   working. Parry answers steel only, and quell answers workings only, so
   those pools may be specific.

## The matrix this slice ships

| Attack | Eligible | Counter today | Counter after |
|---|---|---|---|
| Basic melee swing | dodge, parry, block | riposte / auto-trip / auto-bash (not the tier) | unchanged |
| 11 special moves, single | dodge, parry, block | swing, `counter-melee` | swing, `counter-dodge` / `counter-parry` / `counter-block` by winner |
| Fire, same room | dodge, block | swing, `counter-ranged` | swing, `counter-dodge` / `counter-block` |
| Fire, other room | dodge, block | none (reach gate) | unchanged |
| Throw | dodge, block | none (never wired) | none, now also by the area gate |
| Physical spell, single (5) | dodge, block | swing, `counter-quell` | swing, `counter-dodge` / `counter-block` |
| Physical spell, area (7), core-drain | dodge, block | swing per victim, `counter-quell` | none (area gate) |
| Mental spell, single (9) | quell | swing, `counter-quell` | unchanged |
| Charm | defy | swing, `counter-defy` | counter-taunt, `counter-defy` |
| Taunt | defy | counter-taunt, `counter-defy` | unchanged |
| Non-harm cast, `self` target | none | none | unchanged |
| `multi` targeting (0 shipped) | per pair | would counter | none (gate) |

## Design

### 1. Pools

Five files in `_datafiles/world/dogmud/defense-messages/`:

- `counter-dodge.yaml`: `counter-melee.yaml` renamed, 45 lines unchanged, new
  header. Its lines are already attack-neutral.
- `counter-parry.yaml`: new, 45 lines. Steel turning steel and striking back
  along it. Melee is the only attack parry may answer.
- `counter-block.yaml`: new, 45 lines. The blow, shot or working taken on the
  guard and answered from behind it. Attack-agnostic.
- `counter-quell.yaml`: unchanged. Its header loses the clause about dodge
  and block crits on a physical working, which no longer route here.
- `counter-defy.yaml`: re-toned in place, 45 lines, no line naming a taunt.
  Closes ledger row 5.
- `counter-ranged.yaml`: deleted. Its shot flavour is recorded in the ledger
  as a new M6 row (per-attack counter flavour under a defence-keyed pool).

Every line obeys the player-copy rules: 80-character wrap after ansi
stripping, no numbers, ESL-clear phrasing. Bands keep their counter meaning:
weak = the counter is turned aside, normal = it lands, heavy = it crits.

In `internal/items`, the four counter constants become five, named after the
defence, and one conversion joins `DefencePoolFor`:

```go
// CounterPoolFor names the pool that narrates the counter earned by a
// defensive crit on d. DefenceNone maps to the empty pool.
func CounterPoolFor(d combatvocab.Defence) DefencePool
```

It returns `"counter-" + d` for the five defences and `""` for none. The
constants remain so the fixture and the file-name test can enumerate them.

### 2. Re-key: the primitive takes the winning defence

```go
func ExecuteCounter(defender, attacker *characters.Character,
    shape combatvocab.Attack, defence combatvocab.Defence, sameRoom bool) CounterResult
```

`CounterResult` gains `Defence combatvocab.Defence` and keeps `Shape` for the
gate. `counterPoolFor` takes the defence and is `items.CounterPoolFor`; its
charm carve-out and the comment explaining it are deleted. Both exits pass
the defence they already hold: `move.Defence.Defence` at the skill-move exit,
`out.Defence` at the spell exit.

The primitive refuses `DefenceNone` (returns not countered). It cannot happen
today, because `DefensiveCrit` is set only after a winner is recorded, and a
test on `ResolveChannelAttack` pins that invariant so a future edit cannot
make the pool lookup silently empty.

### 3. The area gate, in the primitive

`ExecuteCounter` refuses any shape whose `Targeting != TargetSingle`. Area is
the ruling; multi rides along because a counter answers one deliberate attack
at one target, and no shipped spell is multi. The gate lives in the primitive
so no exit can bypass it: the four spell exits keep passing
`spellData.Attack()` and the seven area spells fall out.

The drain-area call site is deleted rather than left calling a gate that
always refuses: `DrainAreaPlayerResult.Counter` goes, along with the loop in
the mob drain dispatcher that reads it. An always-empty field is a
lie the next reader believes.

Throw stays as it is: it never had a counter exit, and now the gate says why.

### 4. Every defy crit counter-taunts

`ExecuteCounter` refuses `DefenceDefy` with a comment: words answer words,
and the answer lives in actions. The spell exit branches on the winning
defence before calling anything:

- defy: `actions.FireCounterTaunt(room, counterer, countered, countererUser,
  counteredUser)`, a new exported entry point in `combat_counter.go` that
  wraps `executeCounterTaunt` and the dispatch `counterTauntExit` does today.
  `counterTauntExit` is rewritten to call the same function, so taunt and
  charm share one dispatch.
- anything else: the swing, as today.

The skill-move exit needs no branch: no skill move is social. A test pins
that `counterSkillMoveExit` with a defy defence returns not countered rather
than swinging.

Charm's counter therefore becomes conviction damage with the RETORT prefix
and `CategoryTauntSuccess`, and never health damage. `counter_social_pool_test.go`
is replaced by a test at the spell exit: a defy crit against
`Spell(DamageSocial, TargetSingle)` fires the counter-taunt, drains
conviction, leaves health alone, and renders from `counter-defy`.

### 5. Parity and proof

- A table test over every reachable (attack, defence) cell of the matrix
  above pins the pool: melee x {dodge, parry, block}, ranged x {dodge, block},
  spell-physical x {dodge, block}, spell-mental x quell, and the two defy
  rows refusing the swing. Area and multi shapes refuse for every defence.
- The golden `defense_messages.golden` is regenerated once. A check script
  proves, against the master blob, that every `counter-melee|` row reappears
  as `counter-dodge|` with identical text, that only `counter-ranged|` rows
  vanished, that `counter-quell|` rows are byte-identical, and that the
  `counter-parry|`, `counter-block|` and re-toned `counter-defy|` rows are the
  only new content. The check is sabotaged once (a changed dodge line) and
  seen to fail before its pass is trusted.
- Every task's gate runs `go test . ./...` at the repo root, because the
  line-number guard covers `spell_resolution.go` and `combat_drain.go`.
- Boot check on the detached worktree: `Server Ready`, no `PANIC`, because the
  loader validates every pool at boot and a short band panics there.

### 6. Behaviour changes, each in its own flagged commit

1. **Area attacks earn no counter.** The seven physical area spells and
   core-drain stop giving each victim a free swing at the caster. Patch note.
2. **A defied charm is answered with words.** Charm's defy crit becomes a
   counter-taunt (conviction damage, RETORT), not a swing. Patch note.

Everything else in the slice is narration: the same counters fire, from a
pool named for the defence.

### 7. Docs in the same PR

`internal/combat/context.md:1649` and `internal/items/context.md:419-428`
describe the five pools and the new signature; the counter source comments
lose the "until the counters slice" clauses; `docs/PATCH_NOTES.md` gains the
two entries; the ledger closes row 5 and adds the ranged-flavour row;
`docs/README.md` lists this spec and its plan.

### 8. The playtest

One adversarial run at the end, on the harness with an ephemeral goals file
and a checkout of the branch. The fixture is a player whose defence stats
dwarf the attacker's, so defensive crits are common rather than a 1-in-40
wait, facing in turn: a mob special move (expect a dodge, parry or block
counter whose text matches the defence line just printed), a same-room shot
(dodge or block), a single-target physical spell (dodge or block), a mental
spell (quell), an area spell (defence text and NO counter), a taunt (RETORT),
and a charm (RETORT, conviction falls, health does not). The report must
quote the defence line and the counter line together for each case, because
the whole slice is that they agree. Findings go to memory before the report
is discarded.

## Out of scope, carried

- The basic melee swing's riposte, auto-trip and auto-bash literals: M4e.
- Per-attack counter flavour under a defence-keyed pool (the retired ranged
  lines, spell-flavoured dodge and block counters): M6, as ledger rows.
- Per-targeting defence text (six victims reading the single-target line):
  M6, already filed.
- Defence bands reading margin and crit on the melee path: M4c.
- A social counter that is mechanically social for attacks other than taunt
  and charm: none exist.
