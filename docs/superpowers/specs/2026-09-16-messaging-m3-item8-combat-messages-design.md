# Messaging M3 item 8: combat-messages onto the narration core

The largest store in the arc, and the third and last live instance of the
defect the core exists to prevent. `combat-messages` narrates one swing to
three or four audiences by taking an independent random draw per audience, so
the attacker, the defender and the room are routinely told about different
events. Melee defence shipped this (PR #112), taunt shipped it (PR #115), and
this store still ships it today.

Unlike those two, it cannot be fixed by wiring alone. The core refuses to
render when role pools disagree in length, and 440 of the store's 534 per-tier
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
| Per-tier role equality | 94 groups equal, **440 unequal** | counted from YAML |
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

## What item 8 delivers

1. 984 authored lines bringing every dogmud role pool to per-tier equality.
2. A coordinated renderer on the attack store, replacing the per-role draws.
3. A real `Validate()`, the one the defence sibling has always had.
4. `ConsistentAttackMessages` repaired, still shipping `false`.
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
        pick narration.Picker, indexOverride ...int) narration.Roles

func (m SeparateMessages) Render(skillLevel int, tokens map[TokenName]string,
        pick narration.Picker, indexOverride ...int) narration.Roles
```

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
boot on 440 groups.

`coupdegrace` stays out of the required-intensity list. It is authored only in
`generic.yaml` and every other subtype reaches it through the existing Generic
fallback, which is deliberate and works.

### The knob

`ConsistentAttackMessages` becomes real. When it is on, the call sites pass
`weaponItemId % n` as `Render`'s `indexOverride`, where `n` is now the single
coordinated length shared by every role. That gives per-weapon consistency and
cross-audience coordination at once, which the old shared-seed approach could
never do: the same seed hit `% len` against four different lengths and landed
on four different indices.

The shipped value stays `false`. Turning it on is a feel decision, not a
refactor, and it belongs to the owner. `internal/configs/context.md:108` is
corrected to say `false`.

### The content pad

984 lines across 20 dogmud files, roughly 48 per file, written under
`dogmud-player-copy`: 80-character hard wrap, no raw numbers, ESL-clear
phrasing. Distribution by verb, heaviest first: heavy 206, wait 148, prepare
147, normal 145, miss 109, critical 95, weak 70, fumble 40, coupdegrace 6, plus
the 18 `shooting` lines below.

New lines are **appended** to the tier they belong to, never inserted, so every
existing line keeps its index and the pairing at low indices survives. Where a
pool is short, the new lines are written to pair with the lines already at
those indices in the sibling roles. This is what makes the coordination
meaningful rather than merely legal.

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

### The golden, re-shaped

`combat_messages.golden` moves from "index 0 of each pool, per role" to "every
coordinated variant, all roles on one row", keyed
`subtype|intensity|split|tier|index`. After the pad that is **2,280 rows
covering all 6,975 dogmud lines, each line appearing exactly once** (counted,
not estimated).

The key is per **tier**, not per skill level. Keying by skill level would
re-record every beginner line three times, since the unions are cumulative.

The gap this closes is real and was load-bearing in the design: today's golden
takes a fresh `SequencePicker` per tuple, so it only ever records index 0
(`snapshot_test.go:341`). A migration that dropped, duplicated or reordered any
non-zero-index line passes it green. With 984 lines being added by hand, that
is precisely the mistake most available, so the net has to target it. This is
the M2 lesson restated: a net must target what the refactor makes easier to get
wrong.

Rows stay keyed by the **authored** role name (`toattacker`, `todefender`,
`toroom`, `toattackerroom`, `todefenderroom`), not the core's vocabulary, for
the same reason `defense_messages.golden` does
(`narration/context.md:134-139`): it is what makes a swap of which authored
pool lands in which role visible.

### Recording order

PR 1 re-records the golden in its existing shape, so its diff is pure
additions and every pre-existing row is proven unchanged. PR 2 re-shapes it.
Two `-update` runs, each deliberate, each reviewed for what it is.

### Probes, each proven red before it is trusted

- **Coordination.** One index reaches all four roles. Sabotage: give the
  observer its own draw. Must go red.
- **Draw count.** One `util.Rand` draw per rendered message, not three or four.
  Sabotage: restore a per-role draw. Must go red.
- **Validator.** Each of the three new checks fails a boot on a fixture that
  violates only it. Three sabotages, three separate reds. A validator that
  cannot fail is not a validator.
- **Override.** With the knob on, the same weapon id yields the same
  coordinated index across all roles, and a different weapon id generally does
  not.

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

**PR 1, content.** The 984 lines, the `shooting` null fix, and a golden
re-record in the existing shape. No Go changes. Reviewable as prose.

**PR 2, mechanism.** `PoolFor` and the two `Render` methods, the validator, the
two call sites, the knob, the deletions, the default-tree comments, the
`configs/context.md` correction, the re-shaped golden and all four probes.

The order is forced: the validator fails the boot on 440 groups until the
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
- **Turning `ConsistentAttackMessages` on** is a feel call for the owner once
  the mechanism is correct.
- **M6 content ledger rows** are added in the same commit that defers them, per
  the ledger rule.

## Documentation

- `internal/items/context.md`: the new render surface, the validator's three
  checks, and the per-tier equality invariant with the reason it is exactly
  equivalent to union equality.
- `internal/narration/context.md`: combat-messages joins the consumer list, and
  the `combat_messages.golden` entry is rewritten for the new shape and its
  authored-role keying.
- `internal/configs/context.md:108`: `ConsistentAttackMessages` corrected to
  `false`.
- `docs/README.md`: this spec and its plan.
- The M6 content ledger gains item 8's deferred rows.
