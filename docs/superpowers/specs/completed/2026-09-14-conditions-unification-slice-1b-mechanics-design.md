# Conditions unification, slice 1b: the mechanics the owner ruled on

Date: 2026-09-14. Branch `feature/conditions-unification-slice-1b-mechanics`
off master `04134f7e5` (PR #129, slice 1, merged and not deployed). Slice 1b
sits between slice 1 (the model) and slice 2 (the Go and player-facing
rename). It carries every BEHAVIOUR change the owner ruled on after slice 1
so that slice 2 stays a pure rename with no behaviour in its diff, and the
rename then covers code that is already final.

This slice keeps today's Go names (`buffs`, `BuffSpec`, `Buff`). Slice 2
renames them.

## Owner rulings (2026-09-14, do not relitigate)

1. **Spell damage-over-time ticks every round.** "DOTs are pretty underused
   atm." Record 121 Poisoned goes from every third round to every round. The
   per-tick amount is unchanged, so a spell dot's total damage triples. That
   is accepted.
2. **Bleeds stack and ramp to an equilibrium.** Each application is its own
   stack with its own timer; stacks are longer and weaker than today's single
   bleed, so a lone attacker settles at about 2 to 3 live stacks and a group
   builds more. No hard cap: the equilibrium is how often stacks land times
   how long they last. Sources stay as today (rake, maul, hamstring, drain,
   throttle, the `apply_condition` item proc).
3. **The extra final tick slice 1 gave player tick buffs is the intent.** Keep
   it.
4. **Shields decaying once per round (lasting about twice as long in combat)
   is the intent.** Keep it.
5. **The Recovering penalty must bite for players.** The position penalties
   for being prone or supine already work (the owner sees them in the combat
   dashboard); the one-round `attacks_cap: 1` record added while standing up
   does not reach combat for a player. Make it.
6. **Secret records are left out of `Char.Conditions` entirely**, not shown as
   "Mysterious Affliction". **The in-game `conditions` command matches the web
   client** (spec review, 2026-09-14): hidden and secret records display in
   neither.
7. **Scope: this is slice 1b, before the rename** (approach A of three).
8. **Bleed data model: stacks inside the one Bleeding record** (approach A of
   three; several records per id and a shared-timer counter were rejected).

**Filed, not in this slice:** edged-weapon crits adding bleed stacks, with
matching effects for non-edged weapons for parity. The owner likes the idea
and wants it recorded as a future enhancement.

## Facts verified against source

Every claim below was read from the tree at `04134f7e5` on 2026-09-14.

| Fact | Where |
|---|---|
| Record 121 Poisoned ships `triggerrate: 3 rounds`, `triggercount: 10`, `tick_pool: health`, `tick_from_magnitude: true`, flags `poison`, `silent-start` | `_datafiles/world/dogmud/buffs/121-*.yaml` |
| Record 122 Bleeding ships `triggerrate: 3 rounds`, `triggercount: 1`, `tick_pool: health`, `tick_from_magnitude: true`, trigger line `Blood seeps from your wounds!`, end line `Your wounds stop bleeding.`, flags `bleeding`, `silent-start` | `_datafiles/world/dogmud/buffs/122-*.yaml` |
| `TickTriggers(rounds)` returns 1 below 3 rounds, else `rounds/3`. Non-test production callers: the two spell dot producers and the seven bleed producers, nothing else | `internal/buffs/ticks.go:8-13`; grep |
| Spell dot producers: `dotDuration := calcSpellDuration(...) / 3`, floored at 3, then `AddBuffMagnitude(BuffIdPoisoned, TickTriggers(dotDuration), -float64(dotAmount), "spell")` | `internal/hooks/spell_resolution.go:631-648` (player casting at a mob), `:1657-1674` (mob casting at a player) |
| Bleed producers, all on a landed hit, all `AddBuffMagnitude(BuffIdBleeding, TickTriggers(n), -mag, source)`: rake `Str/12` min 2, n=4; maul `Str/8` min 3, n=5; hamstring `Str/10` min 2, n=5; drain `Str/12` min 2, n=4 (single and area); throttle `Str/10` min 2, n=3; item proc `params.magnitude` (default 2 when < 1), `params.duration` (default 4 when < 1). Every divisor, floor and duration is a Go literal; no bleed knob exists in `config.balance.go` or `config.yaml` | `internal/actions/combat_rake.go:128-132`, `combat_maul.go:128-132`, `combat_hamstring.go:131-135`, `combat_drain.go:143-147,313-317`, `combat_throttle.go:140-144`; `internal/hooks/item_procs.go:200-218`; grep `bleed` in both config files returns nothing |
| So today a bleed of 3 to 5 rounds is **one trigger**: one hit of about 2 to 8 health, three rounds after the move | `TickTriggers` above |
| The five moves share one special-move cooldown (`SpecialMoveReady` / `ClaimSpecialMove`), shipped `SpecialMoveCooldown: 4` (Go default 5). One attacker lands at most one bleed move per 4 rounds | `combat_rake.go:88-97` (same shape in the other four); `config.yaml` blob line 747; `config.balance.combat.go:134-135` |
| Who can use them: rake needs a clawed species with no hands; maul and throttle a fanged species with no hands; drain a life-drain species; hamstring has no player command. Player command files exist for rake, maul, throttle, drain | `combat_rake.go:83`, `combat_maul.go:83`, `combat_throttle.go:94`, `combat_drain.go:95`; `internal/usercommands/{rake,maul,throttle,drain}.go`; `internal/mobcommands/hamstring.go` |
| Mobs get bleed moves two ways. The combat AI offers hamstring, rake, maul, drain and throttle to any mob whose species passes `CanUseHamstring` / `CanUseRake` / `CanUseMaul` / `CanUseDrain` / `CanUseThrottle`. Separately, 11 mob YAMLs list one explicitly: `hamstring` on 10 (the four `ironwind_steppe` wolves 205, 206, 215, 223; two `cascade_pass_road`; four `eastern_highlands`) and `drain` on the summoned vampire 304 | `internal/combat/ai.go:143-167,385,529,555,702,817`; grep `_datafiles/world/dogmud/mobs` |
| The only item carrying `apply_condition`: 40186 Thornwall Harness, `on_grapple`, chance 50, cooldown 5, `condition: 1`, `duration: 6`, `magnitude: 14` | `_datafiles/world/dogmud/items/materials-40000/40186-thornwall_harness.yaml:21-29` |
| `Buffs.AddBuffMagnitude` goes through `AddBuffScaled`, which on a held id **overwrites** `TriggersLeft` and resets `RoundCounter`; then it overwrites `Magnitude` and the `TickAmount` snapshot (`int(magnitude)`, a non-zero value that truncates to 0 floored to 1 in its sign). A second bleed replaces the first | `internal/buffs/buffs.go:249-290,306-337` |
| One record per id is structural: `Buffs.buffIds map[int]int` indexes the list; `HasBuff`, `RemoveBuff`, `TriggersLeft`, `RefreshBuff`, `Started`, `SetTickAmount` all read it | `buffs.go:49-53,108-120,235-246,349-366,531-535` |
| `Buff` instance fields: `BuffId`, `Source`, `OnStartWaiting`, `PermaBuff`, `RoundCounter`, `TriggersLeft`, `TickAmount`, `Magnitude`; all but `BuffId` carry `yaml` omitempty tags, and `Character.Buffs` persists | `buffs.go:14-32` |
| `Buffs.Trigger` increments `RoundCounter`, and on `RoundCounter % RoundInterval == 0` appends the buff to the result and decrements `TriggersLeft`; the caller reads `buff.TickAmount` afterwards | `buffs.go:415-464` |
| Player tick path applies `TickAmount` on every trigger including the expiring one (health harm also cancels craft/salvage, cancels `cancel-on-damage` records, stamps `LastTickCause`); the mob tick path mirrors it | `internal/hooks/NewRound_UserRoundTick.go:264-395`; `NewRound_MobRoundTick.go:218-294` |
| Expired records are pruned on the turn tick, not the round tick | `internal/hooks/NewTurn_PruneBuffs.go:40,97` |
| `GetDurations` reports `TriggersLeft*RoundInterval - RoundCounter%RoundInterval` rounds left | `buffs.go:555-571` |
| Flags are a closed list (`AllFlags`); `LoadDataFiles` rejects an unknown flag; `Quiet`, `SilentStart`, `Bleeding` are the most recent additions | `internal/buffs/buffspec.go:87-146` |
| `BuffSpec.Secret` exists; `VisibleNameDesc` returns "Mysterious Affliction" / "Unknown" for it. Both the `conditions` command and the `Char.Conditions` builder already skip `hidden`-flagged records and use `VisibleNameDesc` for the rest | `buffspec.go:158,215-220`; `internal/usercommands/conditions.go:45-53`; `modules/gmcp/gmcp.Char.go:669-686` |
| The web client labels a chip with `cond.name`, falling back to the map key | `_datafiles/html/public/webclient-pure.html:2157` |
| **Position penalties work.** Prone attack x `ProneAttackMultiplier`, vulnerability x `ProneVulnerabilityMultiplier`, damage x `ProneDamagePenalty`, dodge/parry/block x the three defence penalties, all read from the Position FSM | `internal/combat/combat_helpers.go:470-474,532-539,712-722`; `defence_multiplier.go:477-481` |
| **The Recovering record does not reach a player's combat.** `AttemptRecovery` adds 118 (`attacks_cap: 1`, one trigger) while prone or supine; `UserRoundTick` calls it at `:246` and then `Buffs.Trigger()` at `:264` in the same hook, so the record is already expired when `DoCombat` runs (`Effect` skips expired records). Mobs tick buffs first (`:131`) and recover after (`:165`), so their cap bites | `internal/characters/skills.go:54-107`; `NewRound_UserRoundTick.go:246,264`; `NewRound_MobRoundTick.go:131,165`; `internal/buffs/effects.go:108-113`; hook order `internal/hooks/hooks.go` (UserRoundTick, MobRoundTick, then DoCombat) |
| `calcSwingCount` clamps to `Buffs.Effect(EffectAttacksCap)` | `combat_helpers.go:233` |
| Death cause reads `HasBuff(BuffIdPoisoned)` then `HasBuff(BuffIdBleeding)`; `tickCauseFor` reads the `poison` / `bleeding` flag | `internal/hooks/Death_PlayerAnnouncement.go:199-202`; `internal/hooks/tick_cause.go` |
| The apply-path guard allowlists every bleed and dot producer by `file|line` | `buff_apply_path_guard_test.go:158-172` |

## Design

### 1. Stacking records (new capability in `internal/buffs`)

**Flag.** A new flag `stacking` joins `AllFlags`. `BuffSpec.Validate` rejects a
`stacking` record unless it has `tick_from_magnitude: true` and a one-round
`triggerrate`. Those two are what the tick below assumes; any other
combination is a data error caught at load, not a silent misbehaviour.

**Instance data.** `Buff` gains

```go
Stacks []Stack `yaml:"stacks,omitempty"`

type Stack struct {
	RoundsLeft int `yaml:"roundsleft"`
	Amount     int `yaml:"amount"` // signed per-round amount, negative harms
}
```

The field is empty for every non-stacking record, so existing saves and every
other record are unchanged. A saved bleed keeps its stacks across logout,
as slice 1 made every former condition do.

**Writer.** `Buffs.AddBuffMagnitude` keeps its signature. For a `stacking`
record it appends a stack instead of overwriting:

- `RoundsLeft` = `triggers`, or the spec's `triggercount` when `triggers` is 0.
- `Amount` = `int(magnitude)` with the existing floor (a non-zero value that
  truncates to 0 becomes 1 in its sign). One amount rule, shared with the
  non-stacking path through one helper.
- The record's `TriggersLeft` = the largest `RoundsLeft` among its stacks, so
  `Expired`, `GetDurations`, the prune pass, the `conditions` list and GMCP
  all see a record that lives as long as its longest stack.
- `TickAmount` = the sum of the live stacks' `Amount`; `Magnitude` = the same
  sum as a float. Readers that look at either see the whole bleed.
- The record is created on the first stack exactly as today (flags index,
  `Source`, synchronous, silent start).

**Tick.** `Buffs.Trigger` handles a `stacking` record in the same loop, before
it appends the record to the triggered list: set `TickAmount` to the sum of
the stacks, decrement every stack's `RoundsLeft`, drop stacks at zero, set
`TriggersLeft` to the largest remaining `RoundsLeft` (0 when none remain, so
the record is expired and pruned with its normal end line). Both round-tick
paths already read `buff.TickAmount` after `Trigger` returns, so neither the
player nor the mob tick path changes: one sum is one harm, one wake, one
cancel, one death-cause stamp, one trigger line per round however many stacks
are live. Every bleed producer runs inside combat, which is after that
round's tick (hook order `UserRoundTick`, `MobRoundTick`, `DoCombat`), so a
new stack first ticks on the next round's tick. `Trigger` itself does not
delay a fresh stack: one added and then ticked straight away ticks at once,
as any one-round record does. A stacking record that somehow holds no stacks
(nothing produces one; slice 1 is undeployed, so no save carries a
pre-stacking bleed) is not ticked: `Trigger` expires it without returning it,
so no zero-amount tick can reach the tick path's `tick_percent` fallback.

**Removal.** `RemoveBuff` expires the record and clears its stacks, so a cure
removes the whole bleed.

**Display.** One helper, `buffs.DisplayName(buff, spec)`, returns the visible
name and appends the stack count when more than one stack is live
("Bleeding (3)"). The `conditions` command and the `Char.Conditions` builder
both use it for the name, so neither the template nor the web client changes:
the client already labels the chip with `cond.name`. The GMCP map key stays
the plain visible name.

### 2. Bleed record and producers

- Record 122: `triggerrate: 1 round`, flags gain `stacking`. `triggercount`
  becomes the default stack length used only when a producer passes 0.
- Every bleed producer stops calling `TickTriggers` and passes its stack
  length in rounds.
- **The numbers move to `config.yaml`.** For each of rake, maul, hamstring,
  drain and throttle, three balance knobs in `config.balance.go` with defaults
  in `config.balance.combat.go`: `<Move>BleedRounds` (stack length),
  `<Move>BleedStrengthDivisor` (per-round amount = Strength `ValueAdj` /
  divisor) and `<Move>BleedMin` (per-round floor). Drain's single and area
  forms share drain's three. All fifteen ship in `config.yaml` under the
  special-moves section with a comment block explaining the equilibrium. Per
  the absent-key rule, each default guard backfills only a key that is absent
  or below its legal minimum (rounds and divisor below 1; the floor below 1),
  and a test pins each shipped value.
- **Starting values are chosen in the plan, not here,** by measurement: the
  plan reads the Strength `ValueAdj` of the mobs that can bleed (the eleven
  YAML listings plus the species the AI gates admit) and a
  representative player HP pool, and picks values that meet two targets:
  a stack's length is 2 to 3 times the shipped cooldown (so a lone attacker
  settles at 2 to 3 live stacks), and a stack's total damage is between 1.5
  and 3 times the single hit it replaces (bleeds were underused; this is a
  deliberate buff, bounded). The plan states the chosen numbers and the
  resulting steady-state damage per round before any code uses them.
- **Item proc.** `procApplyCondition` passes `params.duration` as rounds. Its
  in-Go fallbacks (4 and 2) stay as parameter defaults for a malformed item.
  The Thornwall Harness's `duration: 6`, `magnitude: 14` would become 84 total
  per stack (was 14); the plan retunes those two params in the item YAML to
  the same targets.

### 3. Spell dot every round

- Record 121: `triggerrate: 1 round`. Not stacking: a second cast refreshes,
  as today.
- Both spell dot producers pass `dotDuration` as the trigger count directly.
  `dotDuration`'s formula (`calcSpellDuration / 3`, floor 3) is unchanged: it
  belongs to the spell scaling arc.
- With no caller left, `TickTriggers` and `ticks.go` / `ticks_test.go` are
  deleted, and every comment and `context.md` passage that names it is
  rewritten.

### 4. Recovering bites for players

In `UserRoundTick`, the `AttemptRecovery` block moves to after the buff
`Trigger` block, the order `MobRoundTick` already uses. The record is then
live (one trigger left) when `DoCombat` reads `attacks_cap`, and the next
round's `Trigger` expires it before a new attempt re-adds it. The stand and
slip lines are unchanged; they now follow that round's tick lines rather than
preceding them. No mob change.

### 5. Secret records leave both lists

Today three records ship `secret: true`: 81 Respawn Grace, 85 InfraredVision,
99 Alt Character Mob. `VisibleNameDesc` has exactly two production callers,
the `conditions` command and the `Char.Conditions` builder.

- One predicate on the spec, `BuffSpec.Listed() bool`, is false for a
  `hidden`-flagged or `Secret` record. Both lists call it in place of their
  own `hidden` check, so the two can never disagree again.
- Both lists then use `spec.Name` / `spec.Description` (through
  `DisplayName` for the name). With no secret record reaching either list,
  `VisibleNameDesc` and its "Mysterious Affliction" branch have no reader:
  delete the method, its cases in `internal/buffs/buffspec_test.go`, and its
  passage in `internal/buffs/context.md:768`, rather than leave dead code.
- The comments at both skips say why: a hidden record would tell you that you
  are hidden; a secret record is engine bookkeeping or a state the player is
  not meant to know about.

### 6. Documentation

- `internal/buffs/context.md`: the `stacking` flag, `Stack`, the writer and
  tick rules, `DisplayName`; delete the `TickTriggers` passages (`:85`,
  `:416-424`, `:603`, file table `:1225`); record rulings 3 and 4 as intended
  behaviour rather than side effects.
- `internal/characters/context.md:936-945` and `internal/combat/context.md:623`:
  the Recovering record now bites for players; state the tick order that
  makes it.
- `internal/hooks/context.md:1621-1623`: the spell dot passes `dotDuration`
  directly.
- Stale comments naming `TickTriggers` or the one-trigger bleed:
  `internal/characters/buffs.go:155`, `internal/users/userrecord.go:459`,
  `internal/events/eventtypes.go:32`, `internal/hooks/tick_cause.go:18`,
  `internal/hooks/Death_PlayerAnnouncement.go:167`,
  `NewRound_UserRoundTick.go:281-304`, and the six producers' doc comments
  that state a duration and divisor.
- `docs/README.md` indexes this spec and its plan.

## Behaviour changes a player will notice

1. Spell poison hits every round; its total damage is three times what it was.
2. Bleeds from beasts, drain and the Thornwall Harness last longer, hit every
   round for less, and pile up. The list shows "Bleeding (3)". The trigger line
   still prints once per round, not once per stack.
3. A player standing up from prone or supine swings once that round, as mobs
   already do.
4. Secret records such as Respawn Grace no longer appear in the web client's
   status panel or the `conditions` command.

## Testing

Every new assertion is proven capable of failing (sabotage the line it pins,
confirm red, restore).

- **`internal/buffs`:** two adds make two stacks, not an overwrite; the record's
  `TriggersLeft` tracks the longest stack; `Trigger` sums, decrements and drops
  stacks and expires the record with the last one; a stacking record with no
  stacks expires without ticking; `RemoveBuff` clears stacks; `Validate` rejects
  `stacking` without `tick_from_magnitude` or with a longer interval; a
  non-stacking record still overwrites; stacks survive a YAML round trip;
  `DisplayName` shows the count only above one stack.
- **Equilibrium, two gates.** In `internal/configs`, a test loads the shipped
  `config.yaml` and asserts, for each move, stack length / `SpecialMoveCooldown`
  in [2, 3] and a stack's total at Strength 100 in [1.5, 3] times the old
  single hit. In `internal/buffs` (which cannot read balance knobs without the
  shipped file), a simulation adds one stack every 4 rounds for 40 rounds at
  stack lengths 8, 10 and 12 and asserts the live count after warm-up is
  exactly 2, 2 to 3, and 3.
- **Hooks:** a player and a mob holding three stacks take the summed harm once
  per round with one trigger line; death by bleed still names "bleeding out";
  the spell dot lands on consecutive rounds for `dotDuration` rounds.
- **Recovering:** a prone player's `calcSwingCount` is 1 in the round of the
  attempt, driven through `UserRoundTick` rather than by adding the record
  directly (the direct-add test already existed and passed while the real
  path was inert).
- **Both lists:** the two lists live in different packages, so each is
  extracted into a function that takes a character (`conditionEntries` in
  `internal/usercommands`, `buildConditionsPayload` in `modules/gmcp`) and
  each has a test with the SAME fixture (a plain, a hidden, a secret and a
  stacked record) and the SAME expected visible set (plain and stacked only,
  the stacked name carrying its count). `Listed()` has its own four-case test;
  it is the one predicate both call, which is what keeps them agreeing.
- **Config:** each of the fifteen knobs is present in the committed
  `config.yaml` blob, and its default guard does not overwrite a shipped value.
- **Guard:** the apply-path guard allowlist is re-keyed to the producers' new
  lines.
- **Playtest:** one lane where a tester fights a steppe wolf pack
  (`ironwind_steppe`, hamstring users; the plan confirms by a boot-time probe
  which bleed moves the AI actually picks for them) long enough to see the
  stack count rise and
  settle and the bleed end after the fight; one lane where a tester is knocked
  prone and the combat log shows a single swing in the recovery round; a caster
  lane confirming the spell dot's per-round line; after a death, both the web
  client status panel and the `conditions` command checked for Respawn Grace's
  absence.

## Out of scope

- Edged-weapon crits adding stacks, and matching non-edged effects (filed as a
  future enhancement).
- Spell dot duration and magnitude formulas (spell scaling arc).
- Any rename (slice 2) or YAML key and save migration (slice 3).
