# internal/combatvocab

## Purpose

The one declaration of the four combat axes and the defence eligibility
table. Introduced by messaging M4b-2 (2026-09-18) to replace three
declarations of the five defence names, `combat`'s old flattened
attack-channel enum (deleted by Task 5), `spells.SpellType` and
`SpellData.TargetDefenseType`.

It imports nothing but the standard library, so `characters`, `items`,
`combat`, `spells`, `templates` and `hooks` can all import it. It does not
know what a Character is, does not score, cost or narrate anything, and does
not own the damage pipeline's pool (`combat.DamageChannel`), which will be
DERIVED from these axes by `combat.ScaleChannelFor` and
`combat.MitigationChannelFor` (`internal/combat/pools.go`, added later in the
same plan).

## Files

| File | Purpose |
|------|---------|
| `vocab.go` | `AttackType`, `DamageType`, `Targeting`, `Defence`; their constants; `Valid`, `Parse*`, the `*s()` listers, `DamageType.IsHarm`. |
| `attack.go` | `Attack{Type, Damage, Targeting}`, the constructors, `EligibleDefences`, `Pairs`, `Attack.Valid`. |
| `one_declaration_guard_test.go` (added by a later task in the M4b-2 plan) | Fails the build if a defence name is declared as a Go string literal anywhere but here and `internal/actionspec`. |

## The axes

| Axis | Values | Gates |
|---|---|---|
| `AttackType` | melee, ranged, thrown, spell, rhetoric, none | parry (reach) |
| `DamageType` | physical, mental, social, non_harm | quell, defy |
| `Targeting` | self, single, multi, area | nothing in eligibility; contest count, text, counters |
| `Defence` | dodge, parry, block, quell, defy ("" is none) | |

`self` means no target is resolved and the argument passes through (summons,
identify). `single` defaults to the caster for a non-harm cast.

## The eligibility table

| attack | damage | defences |
|---|---|---|
| melee | physical | dodge, parry, block |
| ranged | physical | dodge, block |
| thrown | physical | dodge, block |
| spell | physical | dodge, block |
| spell | mental | quell |
| spell | social | defy |
| rhetoric | social | defy |
| none | non_harm | (uncontested) |

Any other pair is absent: `EligibleDefences` returns `ok == false`, the
constructors cannot build it, and `spells.SpellData.Validate` refuses it.
`(none, non_harm)` is the only row carrying either value, which is how
`Attack.Valid` enforces that they come together.

## Adding a value

A new `AttackType` or `DamageType` is one constant, one entry in its lister,
and one or more rows in `eligibility`. The derived pools that will land in
`internal/combat/pools.go` (added later in the same plan) must also learn it,
and their parity tests will say so. A new `Defence` additionally needs the
three things `combat.DefenceEntriesFor`'s comment lists.
