# Messaging M3 item 8: combat-messages onto the narration core

The largest store in the arc, and the third and last live instance of the
defect the core exists to prevent. `combat-messages` narrates one swing to
three or four audiences by taking an independent random draw per audience, so
the attacker, the defender and the room are routinely told about different
events. Melee defence shipped this (PR #112), taunt shipped it (PR #115), and
this store still ships it today.

Unlike those two, it cannot be fixed by wiring alone. The core refuses to
render when role pools disagree in length, and 446 of the store's 534 per-tier
role groups disagree. So item 8 is a content slice and a plumbing slice in that
order: square the pools up, then coordinate them.

Owner rulings taken during design, recorded here because three of them narrowed
the slice:

- Pad the pools rather than trimming them or deferring. All existing prose is
  kept and the coordination lands now, not in M6.
- The delivery seam is **out**. Combat lines keep their `AttackResult` buffers
  and are filed for M4.
- `world/default` "has nothing to do with us now, this is a holdover from
  GoMud. Comment it and move on."
- **`ConsistentAttackMessages` is deleted, not repaired.** See the history
  below: it is an upstream mechanism this project deliberately superseded, and
  its only remaining effect is to reverse that decision.
- **Existing lines are REORDERED as well as padded.** Found while writing the
  first file and ruled on then: padding alone makes coordination structurally
  possible but leaves roughly a quarter of indices pairing the wrong moments,
  permanently. See "The drift under the gap" below.
- Two PRs: content first, mechanism second.

## Facts verified against source

Every row below was read from source on 2026-09-16, not recalled.

| Fact | Value | Source |
|---|---|---|
| Authored lines in the store | 6,629 (dogmud 5,991, default 638) | counted from YAML |
| The arc spec's figure | "20 files", counted in files not fields | `2026-08-31-messaging-unification-design.md:61` |
| Live selection | `allMessages[pick(len(allMessages))]`, one independent draw per role | `internal/items/attack_messages.go:144` |
| Roles drawn separately | 3 calls (together) or 4 (separate), one per audience | `internal/combat/combat.go:269-279`, `internal/combat/combat_helpers.go:1679-1687` |
| Tier selection | UNIONS beginner, then expert at skill >= 34, then master at >= 67 | `internal/items/attack_messages.go:119-132` |
| The seeded path | `allMessages[msgSeed[0]%len(allMessages)]`, only when `msgSeed != 0` | `internal/items/attack_messages.go:140-141` |
| `msgSeed` source | the weapon's `ItemId`, gated on a balance knob | `internal/combat/combat.go:236-239` |
| `ConsistentAttackMessages` ships | **`false`** (HEAD blob and disk agree) | `_datafiles/config.yaml:853` |
| `configs/context.md` documents it as | `true`. Drift. | `internal/configs/context.md:108` |
| So in production | `msgSeed` is always 0, the seeded branch is dead, every role draws independently | derived from the three rows above |
| Dead branch | `if seedNum[0] == 0 { return mo[0] }` is unreachable, guarded by line 78 | `internal/items/attack_messages.go:82-84` |
| `Validate()` checks | presence of 8 intensity keys. Nothing else. | `internal/items/attack_messages.go:153-164` |
| Defence sibling `Validate()` checks | min-5 variants, non-empty, and equal role lengths | `internal/items/defensive_messages.go:85-96` |
| Defence already renders coordinated | `DefenseOptions.RenderTriad` calls `narration.Render` | `internal/items/defensive_messages.go:143-149` |
| Attack store's narration use | imports `narration` for `Picker`/`DefaultPicker` only. Never calls `Render` or `ValidateVariants`. | `internal/items/attack_messages.go:7,71-74,113-116` |
| Core refuses unequal pools | `Variants.Len()` returns 0, and every caller renders nothing | `internal/narration/render.go:51-70` |
| The fourth role exists for this slice | "`ActeeObserver` ... exists for combat-messages' `separate` case" | `internal/narration/render.go:35-38` |
| Per-tier role equality | 88 groups equal, **446 unequal** | counted from YAML |
| Attacker vs defender alone | 299 groups disagree; only 4 groups have all three roles equal | counted from YAML |
| Lines to author for full equality | **984** (966 plus 18 for two nulled `shooting` roles) | counted from YAML |
| Split selection | `sourceChar.RoomId == targetChar.RoomId` | `internal/combat/combat.go:268`, `internal/combat/combat_helpers.go:1678` |
| `separate` present in | `generic.yaml` and `shooting.yaml` only, both trees | YAML |
| `coupdegrace` present in | `dogmud/generic.yaml` only, and absent from `Validate()`'s list, so every other subtype reaches it through the Generic fallback | `_datafiles/world/dogmud/combat-messages/generic.yaml:663`, `internal/items/attack_messages.go:156,170-190` |
| Golden coverage today | index 0 only, fresh `SequencePicker` per tuple | `internal/narration/snapshot_test.go:301,341` |
| Loader | `fileloader.LoadAllFlatFiles`, panics via the caller on a `Validate()` error | `internal/items/itemspec.go:786-791` |

### The default tree, measured

| Fact | Value | Source |
|---|---|---|
| `ValidateWorldFiles` compares | top-level **directory names** only. It never opens a file. | `internal/util/util.go:913-942` |
| Its only caller | `main.go:278`, passing `world/default` as the example tree | `main.go:278` |
| `world/default` combat-messages shape | flat lists under each role, no `beginner`/`expert`/`master` keys at all | YAML |
| `fumble:` in the 8 default files | **zero**, while `Validate()` requires it. Control: the same grep finds it in 20 of 20 dogmud files. | YAML + `internal/items/attack_messages.go:156` |
| Subsystems `world/default` lacks | 30+, including `defense-messages`, `taunt-messages`, `itemvoices`, `messaging/`, `recipes`, `tips.yaml`, `gossip_templates.yaml`, `mutations`, `factions`, `behaviors`, `weather`, `enchantments` | directory diff |
| YAML file counts | default 1,204 against dogmud 5,143 | `find` count |

So the tree cannot load, and combat-messages is nowhere near the first reason
why. A boot against it dies on many independent loaders. It survives only
because the directory-name check passes.

Two consequences worth recording even though this slice does not act on them.
`ValidateWorldFiles` iterates the **example** tree's 24 subfolders, so it can
only catch dogmud losing a directory that default also has; it is blind to the
32 dogmud directories default lacks. And if `DataFiles` ever resolved to the Go
zero value `_datafiles/world/default` (`internal/configs/config.filepaths.go:22-23`),
the boot would panic inside the combat-messages loader. `config.yaml:232` sets
it to `world/dogmud`, so that path is not taken today.

### What the defect looks like

`dogmud/combat-messages/slashing.yaml`, `critical`, `together`, `beginner`,
ansi stripped:

```
toattacker  [0] Your {itemname} CRITICALLY LACERATES {target}!
            [1] You deliver a CRITICAL STRIKE to {target} with your {itemname}!
            [2] Your {itemname} TEARS THROUGH {target}!
            [3] You land a DEVASTATING HIT on {target}!
todefender  [0] {source}'s {itemname} CRITICALLY LACERATES you!
            [1] {source}'s {itemname} delivers a CRITICAL STRIKE!
            [2] You are DEVASTATED by {source}'s {itemname}!
            [3] {source} lands a DEVASTATING HIT on you!
toroom      [0] {source}'s {itemname} CRITICALLY LACERATES {target}!
            [1] {source}'s {itemname} delivers a CRITICAL STRIKE to {target}!
            [2] {source} DEVASTATES {target} with their {itemname}!
```

The pools are authored as index-paired triads. Index 0 is the laceration, index
1 is the critical strike. But `toroom` is one line short, and production draws
each role independently, so a single swing can narrate as "Your sword
CRITICALLY LACERATES the goblin!" to the attacker, "You are DEVASTATED by
Sable's sword!" to the defender, and "Sable's sword delivers a CRITICAL STRIKE
to the goblin!" to the room. Three events, one swing.

This is why padding is the right answer rather than trimming: the authoring
intent is already coordinated, and the pools have simply drifted apart.

### The drift under the gap

The `critical` example above is the flattering case. Reading a whole file
showed the pools are not reliably index-paired even where they are the same
length. `slashing` / `prepare` / `together` / `beginner`:

```
        attacker                      defender                     room
[0]  prepare for mortal combat    prepares to fight you        prepares to attack {target}
[1]  grip your blade tightly      raises blade menacingly      grips blade tightly
[2]  raise your blade nervously   grips blade awkwardly        (missing)
```

Attacker and room agree at `[1]`. The defender's `[1]` and `[2]` are swapped
relative to them. Today that mismatch only lands sometimes, because the picks
are independent. **Under a coordinated index it lands every time.**

Measured across the store with a lexical proxy, comparing each sibling line
against the attacker line at the same index by content-word overlap:

| | Count | Share |
|---|---|---|
| Best match is the same index | 1,882 | 50% |
| Clearly belongs at a different index | **903** | **24%** |
| No clear match either way | 917 | 24% |
| Compared | 3,702 | |

The proxy is crude and 24% is an estimate, not a count, but the `slashing` case
is confirmed by reading and the direction is not in doubt.

So the pad alone would buy the structure and not the payoff, and would convert
occasional incoherence into permanent incoherence on about a quarter of
indices. The owner ruled that PR 1 therefore **reorders existing lines within
their own tier** as well as appending new ones, so that index N means the same
moment in every role.

Nothing is edited and nothing is deleted. That is what keeps the work provable,
and it is a weaker claim than "additions only", so it needs its own check:
`tools/combat_message_pad_check.py` compares each (file, verb, split, role,
tier) group's multiset of lines against a baseline git ref and fails on any
deletion or edit while allowing free reordering and additions. Both directions
were probed before it was trusted: an edited word reports `LOST`, and a pure
swap of two lines reports `OK ... 1 group(s) reordered`.

## What item 8 delivers

1. 984 authored lines bringing every dogmud role pool to per-tier equality.
2. A coordinated renderer on the attack store, replacing the per-role draws.
3. A real `Validate()`, the one the defence sibling has always had.
4. `ConsistentAttackMessages` and its whole seeded-pick apparatus deleted.
5. A golden that freezes every authored line rather than every pool's first line.
6. Comments marking the default tree as an unloadable GoMud holdover.

## Design

### Why per-tier equality is exactly the requirement

The store hands the core one already-unioned slice per role, because the union
is assembly and assembly stays in the store
(`internal/narration/context.md:55-70`). Those unioned slices must be equal in
length at every skill level. Since beginner is always included and expert and
master are added cumulatively, union equality at all three levels holds if and
only if each tier is role-equal on its own. That is the invariant the validator
enforces and the content pad establishes.

### The renderer

In `internal/items/attack_messages.go`, mirroring `DefenseOptions.RenderTriad`:

```go
// PoolFor returns the tier union for a skill level, the same union
// GetForSkillLevelWith built, as plain strings for the core.
func (stm SkillTieredMessages) PoolFor(skillLevel int) []string

func (m TogetherMessages) Render(skillLevel int, tokens map[TokenName]string,
        pick narration.Picker) narration.Roles

func (m SeparateMessages) Render(skillLevel int, tokens map[TokenName]string,
        pick narration.Picker) narration.Roles
```

No `indexOverride`. The defence sibling takes one, but nothing here needs it
once the knob is gone: production passes a nil picker for
`narration.DefaultPicker`, and the golden builder walks every coordinated
variant with a fresh `SequencePicker`, which yields 0, 1, 2 and so on across
`n` calls.

Role mapping, which is the mapping `render.go:35-38` was written for:

| | Actor | Actee | Observer | ActeeObserver |
|---|---|---|---|---|
| `Together` | `ToAttacker` | `ToDefender` | `ToRoom` | empty |
| `Separate` | `ToAttacker` | `ToDefender` | `ToAttackerRoom` | `ToDefenderRoom` |

`Together` leaving `ActeeObserver` empty is correct rather than missing: when
the participants share a room there is only one observer audience. The
validator is told so explicitly, per `narration/context.md:98-105`.

`GetForSkillLevel` and `GetForSkillLevelWith` are deleted at the moment the
call sites move, per M3's no-dual-maintenance rule. `MessageOptions.Get` and
`GetWith` survive only if something outside this store still uses them; the
plan checks that and deletes them too if not. The unreachable
`if seedNum[0] == 0 { return mo[0] }` goes with them.

### The two call sites

`GetWaitMessages` (`internal/combat/combat.go:268-279`) and
`buildAttackMessages` (`internal/combat/combat_helpers.go:1678-1687`) each
collapse from three or four `GetForSkillLevel` calls into one `Render` call on
the branch they already take. Nothing else about either function changes: the
`{exitname}`/`{entrancename}` resolution for the separate case
(`combat.go:282-300`, `combat_helpers.go:1690-1706`) and the `AttackResult`
buffering are untouched.

### The validator

`WeaponAttackMessageGroup.Validate()` keeps its 8-intensity presence check and
gains, for every intensity, split and tier:

- every declared role pool non-empty,
- all role pools equal in length,
- `narration.ValidateVariants` with the expected roles named, so a
  deliberately absent `ActeeObserver` on a `together` block is distinguishable
  from a `toroom` pool that went missing.

This runs at load through `fileloader.LoadAllFlatFiles`, so a content mistake
is a boot failure rather than silence in play. That is the whole reason the
content pad has to land first: a validator added before the padding fails the
boot on 446 groups.

`coupdegrace` stays out of the required-intensity list. It is authored only in
`generic.yaml` and every other subtype reaches it through the existing Generic
fallback, which is deliberate and works.

### The knob, and why it is deleted rather than repaired

The mechanism is upstream GoMud's. `2a1c51087` (Volte6, 2024-11-21, PR #165)
introduced the seeded pick `mo[seedNum[0]%len(mo)]`, the only commit that
string has ever appeared in, and it has not been touched since.

Its purpose was sound. Combat messages are authored per weapon **subtype**, so
every slashing weapon shares one pool. Seeding the index with the item's
`ItemId`, which is the spec key and not the instance
(`internal/items/items.go:325`), makes a Blackrazor and an iron sword each land
on a different but stable line. It buys per-weapon voice on top of shared
per-subtype pools without authoring any per-weapon text, and no per-item
message override exists anywhere in the codebase.

Upstream's version worked, because upstream keeps its pools equal. The
`default` tree is still 0 of 70 groups unequal today, which is the property the
mechanism depends on: equal pools plus one shared seed means the same index for
every role, so a single seed delivered a coordinated triad and per-weapon voice
at once.

This project moved away from it deliberately. `bcd08700b` (Stage 9.2,
2026-02-14) expanded `slashing`/`critical` from 3 lines per role to 15 and
turned the knob off in the same commit. That was correct: with a fixed index a
player would have seen 1 of the 15, so the expansion would have been invisible.

The side effect went unnoticed. Turning the knob off did not merely disable
consistency, it switched selection from one shared index to one independent
draw per role. **That commit is where the uncoordinated triad was born.**
`39878e436` (Stage 9.5, 2026-02-15) added the skill tiers the next day, pools
drifted unequal, and from then on flipping the knob back would not even have
coordinated.

So the knob is not broken, it is superseded, and its only remaining effect is
to reverse a deliberate design decision that still stands. Item 8 deletes the
config field (`internal/configs/config.balance.go:304`), its yaml key
(`_datafiles/config.yaml:853`), the comment at
`internal/configs/config.balance.combat.go:340`, the `config.gameplay.go:59`
ignore line, both call-site branches (`internal/combat/combat.go:236-239`,
`internal/combat/combat_helpers.go:498-507`), the `msgSeed` field and
parameter, the seeded branches in both getters, and the unreachable
`if seedNum[0] == 0 { return mo[0] }`.

The blast radius is fully contained: `msgSeed` and `seedNum` have no consumers
outside the three files item 8 already rewrites, and the knob has exactly two
live call sites, both of which are being replaced. Deleting it also removes the
`indexOverride` argument from the new `Render` calls, which then take a picker
and nothing else.

`config.yaml` carries the git skip-worktree bit, so that one-line removal is
built from the `git show HEAD:` blob and never from disk. The key is removed
from `internal/configs/context.md` rather than corrected, since the knob is
gone.

### The content pad

984 lines across 20 dogmud files, roughly 48 per file, written under
`dogmud-player-copy`: 80-character hard wrap, no raw numbers, ESL-clear
phrasing. Distribution by verb, heaviest first: heavy 206, wait 148, prepare
147, normal 145, miss 109, critical 95, weak 70, fumble 40, coupdegrace 6, plus
the 18 `shooting` lines below.

Within a tier, lines may be freely reordered and new ones placed wherever the
pairing requires. No line is edited and no line is deleted, and no line moves
between groups. `tools/combat_message_pad_check.py` enforces exactly that
against a baseline ref.

**This is the one part of the slice that nothing can check, and it is the part
that matters.** The validator checks length. The golden records whatever is
there. Neither can see meaning. So this passes every gate while defeating the
entire purpose of the work:

```
attacker[5]  You take a {stance} stance, {position}, preparing to face {target}!
defender[5]  {source} adopts a {stance} stance against you.
room[5]      {source} adopts a masterful combat stance.
```

Six on-theme, well-written, correctly wrapped room lines appended in file order
give `bite`/`prepare` equal pools, a green validator and a clean golden, and
still tell the room about a different moment than the two fighters. The
coordinated index makes that mismatch permanent rather than occasional, which
is arguably worse than today.

The rule that prevents it: **author each (verb, split, tier) group as a set
across all roles at once**, with the sibling lines visibly in front of the
writer, never as "fill role X's gap". One subagent per weapon file, each given
the full role set for every group it touches. The reviewable unit is the group,
not the line.

A worked contrast, same slot:

```
attacker[5]  You take a {stance} stance, {position}, preparing to face {target}!
defender[5]  {source} adopts a {stance} stance against you.
room[5]      {source} takes a {stance} stance against {target}.
```

One weapon file is written and reviewed first, and the remaining 19 start only
once its pairing has been accepted.

`shooting.yaml` sets `todefenderroom: null` on `prepare` and `wait`. That is a
gap, not a deliberate silence: `generic.yaml` authors that role for both verbs
(6 lines each). Shooting gets 9 lines for each, matching its other pools.

Twenty files is a clean parallel unit, one subagent per weapon, each with the
sibling roles in front of it so the pairing is written rather than guessed.

### The default tree

A comment banner at the top of each of the 8 files in
`_datafiles/world/default/combat-messages/`, recording that the tree is
upstream GoMud content, is not loaded by this fork, does not match the current
schema, and that the live store is `world/dogmud/combat-messages`. Nothing is
deleted and nothing is restructured. The broader finding is filed, not fixed.

## The net

### The golden, in two stages

Today's golden is **blind to the content pad**, which is why the widening moves
into PR 1 rather than riding along with the migration.

`buildCombatMessagesGolden` takes a fresh `SequencePicker` per tuple and so
records index 0 of each tier's pool (`snapshot_test.go:341`). The pad appends,
so index 0 does not move. Measured: a 984-line addition would change **6 of
roughly 1,560 rows**, and only because `shooting`'s six nulled
`todefenderroom` tiers go from empty to a real first line. A content PR whose
net moves 6 rows out of 1,560 is not a net.

So:

**PR 1 widens it.** Same per-role vocabulary as today, every index instead of
just the first, keyed `subtype|intensity|split|role|tier|index`, recorded
before any content moves.

Because PR 1 reorders as well as appends, that golden's diff is **not**
additions-only and cannot be read as the proof. It is still worth having: it
shows every line the player can see, which is what makes the pairing
reviewable. The proof that nothing was lost or silently rewritten is
`tools/combat_message_pad_check.py` against the pre-content commit.

**PR 2 re-keys it.** From per-role rows to coordinated rows, keyed
`subtype|intensity|split|tier|index` with all roles on one row: **2,280 rows
covering all 6,975 dogmud lines, each line appearing exactly once** (counted,
not estimated). The strings are the same; only the grouping changes, so the
diff shows the coordination and nothing else.

The key is per **tier**, not per skill level, in both stages. Keying by skill
level would re-record every beginner line three times, since the unions are
cumulative.

The gap both stages close is the same one: at index 0 only, a dropped,
duplicated or reordered non-zero-index line passes green. With 984 lines
arriving by hand and then being re-grouped by a refactor, that is the mistake
most available in each PR, so each PR's net has to target it. This is the M2
lesson restated: a net must target what the change makes easier to get wrong.

Rows stay keyed by the **authored** role name (`toattacker`, `todefender`,
`toroom`, `toattackerroom`, `todefenderroom`), not the core's vocabulary, for
the same reason `defense_messages.golden` does
(`narration/context.md:134-139`): it is what makes a swap of which authored
pool lands in which role visible.

### Recording order

Three `-update` runs, each deliberate and each reviewed for what it is.

1. **PR 1, before any content.** Widen the builder to all indices and record.
   This is the baseline the pad is measured against.
2. **PR 1, per weapon file.** Re-record as each file is padded and reordered.
   The diff is not additions-only, because lines move; it is the reviewable
   picture of the pairing. The mechanical proof is
   `tools/combat_message_pad_check.py` against the pre-content commit, which
   fails on any deletion or edit and permits reordering.
3. **PR 2.** Re-key to coordinated rows. That diff must contain no string that
   did not already appear in PR 1's final golden.

The third property is worth checking mechanically rather than by eye, since the
re-key moves every row: a sorted multiset of the message strings in the two
goldens must be identical.

### Probes, each proven red before it is trusted

- **Coordination.** One index reaches all four roles. Sabotage: give the
  observer its own draw. Must go red.
- **Draw count.** One `util.Rand` draw per rendered message, not three or four.
  Sabotage: restore a per-role draw. Must go red.
- **Validator.** Each of the three new checks fails a boot on a fixture that
  violates only it. Three sabotages, three separate reds. A validator that
  cannot fail is not a validator.
- **Knob removal.** A repo guard asserts no `ConsistentAttackMessages`,
  `msgSeed` or `seedNum` identifier survives anywhere. Probed by
  reintroducing one, which must fail the guard. This is the anti-backslide
  check the conditions slice-2 rename used.

Per the repo rule, every sabotage is confirmed to compile and confirmed to turn
the test red before the green run means anything. Where two branches are
byte-identical, the sabotage is by line number and the failure must name the
right line.

### Expected, not a regression

Coordination replaces three or four `util.Rand` draws per narrated swing with
one. That shifts the global random stream, so combat-determinism tests and
playtest outcomes will move. This is correct and unavoidable, and it is
recorded here so that nobody debugs it as a fault. The goldens are unaffected
because they drive an explicit picker.

The player-visible change is that the three viewpoints of a swing now describe
the same moment. Variety per audience is unchanged or slightly increased, since
no pool shrinks and 984 lines are added.

### Gate

The content SOP's adversarial playtest closes the slice. It must run a melee
fight to several rounds with a second player in the room, so all three
viewpoints of the same swings are captured together and can be read for
coherence, and a ranged fight across two rooms to exercise the `separate` split
and its fourth audience. A fixture that dies in one round comes back partial,
so the Sable arena at Rift Chamber 5000 is the right harness with gold set for
a long fight.

## Sequencing

**PR 1, content.** The 984 lines, the `shooting` null fix, a read-only pool
audit tool, and the golden builder widened from index 0 to every index in its
existing per-role vocabulary, then re-recorded. The only Go touched is the
snapshot builder; no production code changes. Reviewable as prose.

**PR 2, mechanism.** `PoolFor` and the two `Render` methods, the validator, the
two call sites, the full `ConsistentAttackMessages` and `msgSeed` deletion, the
default-tree comments, the `configs/context.md` removal, the re-shaped golden
and all four probes.

The order is forced: the validator fails the boot on 446 groups until the
padding lands.

## Out of scope, filed

- **Delivery.** Combat lines never touch `messaging.SendTrio`. They buffer into
  `AttackResult` `TaggedMessage`s (`internal/combat/attackresult.go:202-216`)
  and drain in `internal/hooks` through `SendText` and
  `SendTextVisualToUser` (`internal/hooks/combat_verbosity.go:307,313,349`),
  ending as raw `events.Message` (`internal/users/userrecord.go:496-499`). The
  buffers exist to honour combat verbosity filtering, so this is not a swap.
  Filed for M4's one-send-path goal.
- **`world/default` is vestigial** across 30+ subsystems, and
  `ValidateWorldFiles` is nearly inert in the direction that matters. Filed
  against the M5 `world/default` template-shadowing item.
- **Real per-weapon voice**, if it is ever wanted. The deleted knob faked it by
  fixing an index into the shared subtype pool. The genuine versions are
  authored per-item overrides for named weapons such as the Blackrazor, or an
  `ItemId`-derived offset that shifts which slice of a pool a weapon type draws
  from without collapsing it to one line. Both are features with content and
  balance consequences, not a config flip, and neither belongs in item 8.
- **M6 content ledger rows** are added in the same commit that defers them, per
  the ledger rule.

## Documentation

- `internal/items/context.md`: the new render surface, the validator's three
  checks, and the per-tier equality invariant with the reason it is exactly
  equivalent to union equality.
- `internal/narration/context.md`: combat-messages joins the consumer list, and
  the `combat_messages.golden` entry is rewritten for the new shape and its
  authored-role keying.
- `internal/configs/context.md:108`: the `ConsistentAttackMessages` line is
  removed, not corrected. It documented the knob as `true` while the shipped
  value was `false`, so the drift dies with the knob.
- `docs/README.md`: this spec and its plan.
- The M6 content ledger gains item 8's deferred rows.
