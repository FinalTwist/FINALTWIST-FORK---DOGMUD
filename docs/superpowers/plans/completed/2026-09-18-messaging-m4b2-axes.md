# Messaging M4b-2: the four axes

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Four authored axes (`AttackType`, `DamageType`, `Targeting`, `Defence`), each declared once in a new leaf package `internal/combatvocab`, replace `combat.AttackChannel`, `combat.DefenseType`, `characters.Defense*`, `spells.SpellType` and `SpellData.TargetDefenseType`; defence eligibility is one table keyed by the attack-and-damage pair; every golden stays byte-identical, and the two behaviour changes (core-drain, non-harm casts at mobs) land last in their own flagged commits.

**Architecture:** The leaf package lands first and is imported by `characters`, `items`, `combat`, `spells`, `templates` and `hooks`. `combatvocab.Attack{Type, Damage, Targeting}` is the value that travels through the seam in place of the flattened channel enum; a field holding one is named `Shape` because `SkillMoveParams.Attack` is already the attacker's `AttackSide`. `DamageChannel` (Physical, Magical, Conviction) is not an axis: it stays in `combat` and is DERIVED from the axes by two functions that replace three hand-rolled switches. Spells flip in three commits (additive Go, YAML gains the new keys beside the old, consumers flip and the old keys are stripped) so every commit compiles and boots. Untyped string constants in the 13 test files that spell `"dodge"` keep compiling against a `Defence` parameter, so that sweep is the compiler's, not a script's.

**Tech Stack:** Go 1.23, `gopkg.in/yaml.v3` (unknown keys ignored, hence the shipped-data grep), snapshot goldens in `internal/narration/testdata/stores/`, Python 3 line-edit tool with temp-file swap (never a yaml round trip), AST and literal-scan guards in package tests.

**Spec:** `docs/superpowers/specs/2026-09-18-messaging-m4b2-axes-design.md`. Every ruling and every line number below comes from it; the spec's facts table was verified against master `612b85d54` on 2026-09-18.

**Scope note:** The counters slice (re-keying counter pools to the winning defence, the parry and block counter text, the area gate, the playtest) is the NEXT plan. This plan carries `Targeting` on the attack so that slice only has to read it.

---

## Facts verified against source (master `612b85d54`, 2026-09-18)

| Fact | Value | Source |
|---|---|---|
| `SkillMoveParams` already has a field named `Attack` | `Attack AttackSide` | `internal/combat/skill_moves.go:60-70` |
| `ChannelDefenceResult.DefenceType` | `string`, set from `res.Winner` | `internal/combat/defence_multiplier.go:225,550` |
| The unchecked bridge | `items.DefenseType(out.DefenceType)` | `defence_multiplier.go:298` |
| `runSpellChannelAttack` | package var aliasing `combat.ResolveChannelAttack` | `internal/hooks/spell_resolution.go:324` |
| `spellAttackChannel` | switch on `TargetDefenseType`: physical, social, default mental | `spell_resolution.go:1241-1262` |
| Spell mitigation switch | physical: physical mitigation; mental: magical; default: 0, cap 0.75 | `internal/hooks/combat_shared_helpers.go:87-96` |
| `channelDamageChannel` | melee, ranged "physical"; both spell channels "magical"; social "conviction" | `internal/combat/defence_multiplier.go:769-778` |
| `SituationalAttackMult` | penalties on melee and ranged only | `internal/combat/situational.go:35-51` |
| `counterPoolFor` | ranged; spell-physical and spell-mental to counter-quell; social; default melee | `internal/combat/counter.go:161-172` |
| `DefenceSetFor` | 5 rows, default nil | `internal/combat/defence_sets.go:51-65` |
| `defenceTypesUsed` iterates | dodge, parry, block, in that order | `internal/hooks/NewRound_DoCombat_helpers.go:322-335` |
| `seedTestSpell` | `(spellId string, spellType spells.SpellType, baseFolds int)` | `internal/actions/cast_test.go:32` |
| Test files naming `spells.Harm*`/`Help*`/`Neutral` or `TargetDefenseType` | 33 files, 135 lines; largest `behaviortree/action_cast_best_in_category_test.go` (22), `hooks/hooks_test.go` (18) | grep |
| Test files naming `Channel*` constants | 29 files | grep |
| `defensename` template func | `templatesfunctions.go:100-113`; tests `templates/defensename_test.go` | read |
| `spell.template` rows | `.Type.HelpOrHarmString` (line 7), `.Type.TargetTypeString` (8), `defensename .TargetDefenseType` (12) | read |
| Shipped `effect_type: damage|dot|knockdown` spells all carry physical or mental | 11 damage, 2 dot, 1 knockdown; none absent | spec table |
| The six `calcSpellDamageForCharacter` production callers | the `damage`, `dot` and `knockdown` effect arms only (`spell_resolution.go:573,680,994,1607,1688` plus the mob applier) | grep |
| Boot check | detached worktree, `boot-check.exe`, exit 124 is success, judge by `Server Ready` and the panic patterns | `.claude/skills/dogmud-shipping/SKILL.md:129-150` |

---

## File structure

**Create**
- `internal/combatvocab/vocab.go`: the four enums, their constants, `Valid()`.
- `internal/combatvocab/attack.go`: `Attack`, constructors, `EligibleDefences`, `Pairs`.
- `internal/combatvocab/vocab_test.go`, `attack_test.go`, `one_declaration_guard_test.go`.
- `internal/combatvocab/context.md`.
- `internal/combat/pools.go`: `ScaleChannelFor`, `MitigationChannelFor`, `DamageChannel.ToughenName`.
- `internal/combat/pools_test.go`.
- `internal/spells/axes.go`: `Attack()`, `IsHarm()`, display methods, `DefenceNames()`, `validateAxes`.
- `internal/spells/axes_test.go`, `internal/spells/shipped_axes_test.go`.
- `tools/spell_axes_rewrite.py`.
- `internal/hooks/nonharm_mob_shortcut_test.go`.

**Modify** (the compiler enumerates the rest once a type is deleted)
- `internal/items/defensive_messages.go`, `test_helpers_combat.go`: `DefenseType` becomes `DefencePool`.
- `internal/characters/character.go:727-741` (delete consts), `combat.go:279,341`, `resources.go:140-297`.
- `internal/combat/attackresult.go`, `defence_sets.go`, `defence_multiplier.go`, `skill_moves.go`, `situational.go`, `counter.go`, `combat.go`, `combat_helpers.go`, `calculations.go`, `surprise_narration.go`.
- `internal/actions/combat_*.go` (12 files), `cast.go`, `cast_admission.go`.
- `internal/hooks/spell_resolution.go`, `combat_shared_helpers.go`, `counter_tier.go`, `charm_spell.go`, `NewRound_DoCombat_helpers.go`, `NewRound_DoCombat_unified.go`.
- `internal/usercommands/throw.go`, `spells.go`, `skill.cast.go`; `internal/mobcommands/cast.go`.
- `internal/spells/spells.go`; `internal/templates/templatesfunctions.go`.
- `_datafiles/world/dogmud/spells/*.yaml` (59), `_datafiles/world/default/spells/*.yaml` (8), `_datafiles/world/dogmud/templates/help/spell.template`, `_datafiles/world/default/templates/help/spell.template`.
- Docs: eight `context.md`, `docs/schemas/spell.md`, `docs/README.md`, `docs/PATCH_NOTES.md`, `.claude/skills/dogmud-combat/SKILL.md`.

**Delete**
- `combat.AttackChannel` and its five constants; `combat.DefenseType` and its four constants; `combat.DefenceSetFor`; `combat.channelDamageChannel`; `characters.Defense*` six constants; `spells.SpellType` and its seven constants plus both display methods on it; `SpellData.Type`, `SpellData.TargetDefenseType`; `hooks.spellAttackChannel`; the `defensename` template func.

---

## Task 0: Branch and baseline

**Files:** none

- [ ] **Step 1: Branch from the design branch so the spec rides along**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
git checkout feature/messaging-m4b2-design
git checkout -b feature/messaging-m4b2-axes
git log --oneline -1   # expect 40b1e088f docs(messaging): M4b-2 spec, the four axes
```

- [ ] **Step 2: Record the golden baseline from master**

Every golden must equal master's at the end. Save master's copies now so the final diff is against a fixed target, not against a working tree that later tasks touch.

```bash
mkdir -p C:/tmp/m4b2-goldens
for f in internal/narration/testdata/stores/*.golden; do git show master:"$f" > "C:/tmp/m4b2-goldens/$(basename "$f")"; done
ls C:/tmp/m4b2-goldens | wc -l   # expect 15
```

- [ ] **Step 3: Prove the suite is green before anything changes**

```bash
go build ./... && go test ./... 2>&1 | tail -5
```
Expected: every package `ok` (the `internal/rooms` zone lifecycle tests fail on Windows under `DOGMUD_BOOT_SMOKE=1` only; do not set it).

---

## Task 1: The leaf package `internal/combatvocab`

**Files:**
- Create: `internal/combatvocab/vocab.go`
- Create: `internal/combatvocab/attack.go`
- Create: `internal/combatvocab/vocab_test.go`
- Create: `internal/combatvocab/attack_test.go`
- Create: `internal/combatvocab/context.md`

- [ ] **Step 1: Write the failing tests**

`internal/combatvocab/vocab_test.go`:

```go
package combatvocab

import "testing"

func TestZeroValuesAreInvalid(t *testing.T) {
	if AttackType("").Valid() || DamageType("").Valid() || Targeting("").Valid() || Defence("").Valid() {
		t.Fatal("the zero value of every axis must be invalid, so an unset field cannot pass silently")
	}
}

func TestEveryDeclaredValueIsValid(t *testing.T) {
	for _, a := range AttackTypes() {
		if !a.Valid() {
			t.Errorf("AttackType %q declared but not Valid", a)
		}
	}
	for _, d := range DamageTypes() {
		if !d.Valid() {
			t.Errorf("DamageType %q declared but not Valid", d)
		}
	}
	for _, tg := range Targetings() {
		if !tg.Valid() {
			t.Errorf("Targeting %q declared but not Valid", tg)
		}
	}
	for _, d := range Defences() {
		if !d.Valid() {
			t.Errorf("Defence %q declared but not Valid", d)
		}
	}
}

func TestDeclaredValueCountsAreTheSpecsCounts(t *testing.T) {
	if got := len(AttackTypes()); got != 6 {
		t.Errorf("AttackTypes: %d, spec says 6 (melee, ranged, thrown, spell, rhetoric, none)", got)
	}
	if got := len(DamageTypes()); got != 4 {
		t.Errorf("DamageTypes: %d, spec says 4", got)
	}
	if got := len(Targetings()); got != 4 {
		t.Errorf("Targetings: %d, spec says 4", got)
	}
	if got := len(Defences()); got != 5 {
		t.Errorf("Defences: %d, spec says 5", got)
	}
}

func TestParseRejectsUnknownAndAcceptsKnown(t *testing.T) {
	if _, err := ParseAttackType("sword"); err == nil {
		t.Error("ParseAttackType accepted an unknown value")
	}
	if got, err := ParseAttackType("thrown"); err != nil || got != AttackThrown {
		t.Errorf("ParseAttackType(thrown) = %q, %v", got, err)
	}
	if _, err := ParseDamageType("non-harm"); err == nil {
		t.Error("the hyphen spelling from the M4 spec is NOT the canonical one; it must be rejected")
	}
	if got, err := ParseDamageType("non_harm"); err != nil || got != DamageNonHarm {
		t.Errorf("ParseDamageType(non_harm) = %q, %v", got, err)
	}
	if _, err := ParseTargeting("group"); err == nil {
		t.Error("ParseTargeting accepted the display word instead of the key")
	}
	if _, err := ParseDefence("resist"); err == nil {
		t.Error("ParseDefence accepted the pre-U6 name")
	}
}
```

`internal/combatvocab/attack_test.go`:

```go
package combatvocab

import (
	"reflect"
	"testing"
)

// The eligibility table, pinned as literals. The first five rows ARE the old
// combat.DefenceSetFor table (defence_sets.go:51-65 on master 612b85d54); the
// thrown and spell/social rows are the spec's two additions, and the last row
// is the uncontested pair. If this test and attack.go disagree, the code is
// wrong, not the test.
func TestEligibilityTableIsTheSpecsTable(t *testing.T) {
	cases := []struct {
		name string
		a    Attack
		want []Defence
	}{
		{"melee physical", Melee(TargetSingle), []Defence{DefenceDodge, DefenceParry, DefenceBlock}},
		{"ranged physical", Ranged(TargetSingle), []Defence{DefenceDodge, DefenceBlock}},
		{"thrown physical", Thrown(TargetArea), []Defence{DefenceDodge, DefenceBlock}},
		{"spell physical", Spell(DamagePhysical, TargetSingle), []Defence{DefenceDodge, DefenceBlock}},
		{"spell mental", Spell(DamageMental, TargetSingle), []Defence{DefenceQuell}},
		{"spell social", Spell(DamageSocial, TargetSingle), []Defence{DefenceDefy}},
		{"rhetoric social", Rhetoric(TargetSingle), []Defence{DefenceDefy}},
		{"none non_harm", NonHarm(TargetSelf), []Defence{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := EligibleDefences(tc.a)
			if !ok {
				t.Fatalf("%v is not in the table", tc.a)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("EligibleDefences(%v) = %v, want %v", tc.a, got, tc.want)
			}
		})
	}
}

func TestTargetingDoesNotChangeEligibility(t *testing.T) {
	for _, tg := range Targetings() {
		got, _ := EligibleDefences(Melee(tg))
		want := []Defence{DefenceDodge, DefenceParry, DefenceBlock}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Melee(%s) = %v; targeting must not change what may roll (spec ruling 4)", tg, got)
		}
	}
}

func TestUnknownPairIsNotInTheTable(t *testing.T) {
	bad := Attack{Type: AttackMelee, Damage: DamageMental, Targeting: TargetSingle}
	if set, ok := EligibleDefences(bad); ok || set != nil {
		t.Errorf("melee/mental must be absent from the table, got %v, %v", set, ok)
	}
	if _, ok := EligibleDefences(Attack{}); ok {
		t.Error("the zero Attack must not be in the table")
	}
}

func TestEveryConstructorBuildsATablePair(t *testing.T) {
	for _, a := range []Attack{
		Melee(TargetSingle), Ranged(TargetSingle), Thrown(TargetArea),
		Spell(DamagePhysical, TargetArea), Spell(DamageMental, TargetSingle), Spell(DamageSocial, TargetSingle),
		Rhetoric(TargetSingle), NonHarm(TargetSelf),
	} {
		if !a.Valid() {
			t.Errorf("constructor produced an invalid attack %v", a)
		}
	}
	// Spell with a non-harm damage type is the one pair a constructor could be
	// asked for that the table refuses: a cast that harms nobody is NonHarm.
	if Spell(DamageNonHarm, TargetSingle).Valid() {
		t.Error("Spell(DamageNonHarm) must be invalid; use NonHarm")
	}
}

func TestNoneAndNonHarmAreBoundTogether(t *testing.T) {
	if (Attack{Type: AttackNone, Damage: DamagePhysical, Targeting: TargetSingle}).Valid() {
		t.Error("attack_type none with a harm damage type must be invalid (ruling 10)")
	}
	if (Attack{Type: AttackSpell, Damage: DamageNonHarm, Targeting: TargetSingle}).Valid() {
		t.Error("damage_type non_harm with an attack type other than none must be invalid (ruling 10)")
	}
}

func TestPairsListsExactlyTheTable(t *testing.T) {
	if got := len(Pairs()); got != 8 {
		t.Errorf("Pairs() has %d rows, the spec table has 8", got)
	}
	for _, p := range Pairs() {
		if _, ok := EligibleDefences(Attack{Type: p.Type, Damage: p.Damage, Targeting: TargetSingle}); !ok {
			t.Errorf("Pairs() lists %v but EligibleDefences does not know it", p)
		}
	}
}
```

- [ ] **Step 2: Run them to verify they fail**

```bash
go test ./internal/combatvocab/ 2>&1 | head -5
```
Expected: build failure, `undefined: AttackType` (the package does not exist yet).

- [ ] **Step 3: Write `vocab.go`**

```go
// Package combatvocab is the one declaration of the four combat axes:
// what kind of attack it is, what kind of harm it does, how many it reaches,
// and which defences may answer it. Every other package imports this one; it
// imports nothing but the standard library.
//
// M4b-2 of the messaging arc (docs/superpowers/specs/2026-09-18-messaging-m4b2-axes-design.md).
// Before it, the same five defence names were declared three times, the
// attack and damage axes were flattened into one five-value enum, and the
// three damage types were spelt differently in three places.
package combatvocab

import "fmt"

// AttackType is HOW the attack is delivered. Parry is gated on it: you cannot
// parry a bolt, a flask or a working.
type AttackType string

const (
	AttackMelee    AttackType = "melee"
	AttackRanged   AttackType = "ranged"
	AttackThrown   AttackType = "thrown"
	AttackSpell    AttackType = "spell"
	AttackRhetoric AttackType = "rhetoric"
	// AttackNone is the attack type of a cast that harms nobody: a heal is not
	// an attack. It is bound to DamageNonHarm by Attack.Valid (owner ruling
	// 2026-09-18) and is the uncontested pair.
	AttackNone AttackType = "none"
)

// DamageType is WHAT the attack does to its target. Quell and defy are gated
// on it. It is not the damage pipeline's pool (combat.DamageChannel); a
// physical spell is dodged but still scales magically. That pool is derived
// from these axes, never authored.
type DamageType string

const (
	DamagePhysical DamageType = "physical"
	DamageMental   DamageType = "mental"
	DamageSocial   DamageType = "social"
	DamageNonHarm  DamageType = "non_harm"
)

// Targeting is HOW MANY the attack reaches. It never changes eligibility
// (owner ruling 2026-09-17): a cleave is still parryable. It changes how many
// contests happen, the defence text, and whether a counter is earned.
type Targeting string

const (
	// TargetSelf means NO target is resolved and the argument text passes
	// through: summons and identify. It is not "defaults to the caster";
	// that is TargetSingle with the caster as the default.
	TargetSelf   Targeting = "self"
	TargetSingle Targeting = "single"
	TargetMulti  Targeting = "multi"
	TargetArea   Targeting = "area"
)

// Defence is one of the five things a defender can do. "" is DefenceNone and
// is the zero value: a swing nobody defended.
type Defence string

const (
	DefenceNone  Defence = ""
	DefenceDodge Defence = "dodge"
	DefenceParry Defence = "parry"
	DefenceBlock Defence = "block"
	DefenceQuell Defence = "quell"
	DefenceDefy  Defence = "defy"
)

var (
	attackTypes = []AttackType{AttackMelee, AttackRanged, AttackThrown, AttackSpell, AttackRhetoric, AttackNone}
	damageTypes = []DamageType{DamagePhysical, DamageMental, DamageSocial, DamageNonHarm}
	targetings  = []Targeting{TargetSelf, TargetSingle, TargetMulti, TargetArea}
	defences    = []Defence{DefenceDodge, DefenceParry, DefenceBlock, DefenceQuell, DefenceDefy}
)

// AttackTypes returns every declared value, in declaration order.
func AttackTypes() []AttackType { return append([]AttackType(nil), attackTypes...) }

// DamageTypes returns every declared value, in declaration order.
func DamageTypes() []DamageType { return append([]DamageType(nil), damageTypes...) }

// Targetings returns every declared value, in declaration order.
func Targetings() []Targeting { return append([]Targeting(nil), targetings...) }

// Defences returns the five real defences; DefenceNone is not one.
func Defences() []Defence { return append([]Defence(nil), defences...) }

func (a AttackType) Valid() bool {
	for _, v := range attackTypes {
		if a == v {
			return true
		}
	}
	return false
}

func (d DamageType) Valid() bool {
	for _, v := range damageTypes {
		if d == v {
			return true
		}
	}
	return false
}

func (t Targeting) Valid() bool {
	for _, v := range targetings {
		if t == v {
			return true
		}
	}
	return false
}

// Valid reports a real defence. DefenceNone is not valid: it is the absence
// of one.
func (d Defence) Valid() bool {
	for _, v := range defences {
		if d == v {
			return true
		}
	}
	return false
}

// IsHarm is the harm-versus-help question in one place.
func (d DamageType) IsHarm() bool { return d.Valid() && d != DamageNonHarm }

func ParseAttackType(s string) (AttackType, error) {
	if a := AttackType(s); a.Valid() {
		return a, nil
	}
	return "", fmt.Errorf("attack_type %q is not one of %v", s, attackTypes)
}

func ParseDamageType(s string) (DamageType, error) {
	if d := DamageType(s); d.Valid() {
		return d, nil
	}
	return "", fmt.Errorf("damage_type %q is not one of %v", s, damageTypes)
}

func ParseTargeting(s string) (Targeting, error) {
	if t := Targeting(s); t.Valid() {
		return t, nil
	}
	return "", fmt.Errorf("targeting %q is not one of %v", s, targetings)
}

func ParseDefence(s string) (Defence, error) {
	if d := Defence(s); d.Valid() {
		return d, nil
	}
	return "", fmt.Errorf("defence %q is not one of %v", s, defences)
}
```

- [ ] **Step 4: Write `attack.go`**

```go
package combatvocab

import "fmt"

// Attack is the value that travels with every attack through the contest
// seam: the three authored axes together. It replaced combat.AttackChannel,
// which flattened the first two into one enum (which is why melee-physical
// allowed parry and ranged-physical did not: parry is gated by Type, quell and
// defy by Damage).
//
// Build one with a constructor. A struct literal compiles, but only the
// constructors and the spell validator are guaranteed to produce a pair the
// eligibility table knows.
type Attack struct {
	Type      AttackType
	Damage    DamageType
	Targeting Targeting
}

func (a Attack) String() string {
	return fmt.Sprintf("%s/%s/%s", a.Type, a.Damage, a.Targeting)
}

// Pair is one row key of the eligibility table.
type Pair struct {
	Type   AttackType
	Damage DamageType
}

// eligibility is THE table. Adding a row here is the whole change for a new
// pairing, subject to the equipment gate in combat.DefenceEntriesFor and the
// three things a new DEFENCE must carry (see that function's comment).
//
// Rows are in the spec's order. The (none, non_harm) row is empty on purpose:
// it is the uncontested pair, and its presence in the table is what lets a
// non-harm cast be told apart from a pair nobody declared.
var eligibility = []struct {
	pair Pair
	set  []Defence
}{
	{Pair{AttackMelee, DamagePhysical}, []Defence{DefenceDodge, DefenceParry, DefenceBlock}},
	{Pair{AttackRanged, DamagePhysical}, []Defence{DefenceDodge, DefenceBlock}},
	{Pair{AttackThrown, DamagePhysical}, []Defence{DefenceDodge, DefenceBlock}},
	{Pair{AttackSpell, DamagePhysical}, []Defence{DefenceDodge, DefenceBlock}},
	{Pair{AttackSpell, DamageMental}, []Defence{DefenceQuell}},
	{Pair{AttackSpell, DamageSocial}, []Defence{DefenceDefy}},
	{Pair{AttackRhetoric, DamageSocial}, []Defence{DefenceDefy}},
	{Pair{AttackNone, DamageNonHarm}, []Defence{}},
}

// EligibleDefences returns the defences that may answer a, in table order,
// and whether the pair is in the table at all. A copy is returned: callers
// append to the set (dual-wield parry entries) and must not touch the table.
//
// ok == false is a programming error or bad data, never a legitimate
// outcome. The seam logs it and resolves uncontested, which is what the old
// DefenceSetFor default arm did silently.
func EligibleDefences(a Attack) ([]Defence, bool) {
	for _, row := range eligibility {
		if row.pair.Type == a.Type && row.pair.Damage == a.Damage {
			return append([]Defence{}, row.set...), true
		}
	}
	return nil, false
}

// Pairs returns the table's row keys, in order, for the guards.
func Pairs() []Pair {
	out := make([]Pair, 0, len(eligibility))
	for _, row := range eligibility {
		out = append(out, row.pair)
	}
	return out
}

// Valid reports an attack the table knows with a valid targeting. Because
// (none, non_harm) is the only row carrying either of those values, it also
// enforces the owner's rule that they come together.
func (a Attack) Valid() bool {
	if !a.Targeting.Valid() {
		return false
	}
	_, ok := EligibleDefences(a)
	return ok
}

// Constructors. Each produces one of the table's rows and nothing else.

func Melee(t Targeting) Attack    { return Attack{AttackMelee, DamagePhysical, t} }
func Ranged(t Targeting) Attack   { return Attack{AttackRanged, DamagePhysical, t} }
func Thrown(t Targeting) Attack   { return Attack{AttackThrown, DamagePhysical, t} }
func Rhetoric(t Targeting) Attack { return Attack{AttackRhetoric, DamageSocial, t} }
func NonHarm(t Targeting) Attack  { return Attack{AttackNone, DamageNonHarm, t} }

// Spell builds a harmful cast. Spell(DamageNonHarm, t) is deliberately NOT a
// table row: a cast that harms nobody is NonHarm(t), and Valid() says so.
func Spell(d DamageType, t Targeting) Attack { return Attack{AttackSpell, d, t} }
```

- [ ] **Step 5: Run the tests**

```bash
go test ./internal/combatvocab/ -v 2>&1 | grep -E "^(=== RUN|--- (PASS|FAIL)|ok|FAIL)" | head -30
```
Expected: every test PASS, `ok`.

- [ ] **Step 6: Write `internal/combatvocab/context.md`**

```markdown
# internal/combatvocab

## Purpose

The one declaration of the four combat axes and the defence eligibility
table. Introduced by messaging M4b-2 (2026-09-18) to replace three
declarations of the five defence names, the flattened `combat.AttackChannel`,
`spells.SpellType` and `SpellData.TargetDefenseType`.

It imports nothing but the standard library, so `characters`, `items`,
`combat`, `spells`, `templates` and `hooks` can all import it. It does not
know what a Character is, does not score, cost or narrate anything, and does
not own the damage pipeline's pool (`combat.DamageChannel`), which is DERIVED
from these axes by `combat.ScaleChannelFor` and `combat.MitigationChannelFor`.

## Files

| File | Purpose |
|------|---------|
| `vocab.go` | `AttackType`, `DamageType`, `Targeting`, `Defence`; their constants; `Valid`, `Parse*`, the `*s()` listers, `DamageType.IsHarm`. |
| `attack.go` | `Attack{Type, Damage, Targeting}`, the constructors, `EligibleDefences`, `Pairs`, `Attack.Valid`. |
| `one_declaration_guard_test.go` | Fails the build if a defence name is declared as a Go string literal anywhere but here and `internal/actionspec`. |

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
and one or more rows in `eligibility`. The derived pools in
`internal/combat/pools.go` must also learn it, and their parity tests will say
so. A new `Defence` additionally needs the three things
`combat.DefenceEntriesFor`'s comment lists.
```

- [ ] **Step 7: Commit**

```bash
git add internal/combatvocab/vocab.go internal/combatvocab/attack.go internal/combatvocab/vocab_test.go internal/combatvocab/attack_test.go internal/combatvocab/context.md
git commit -m "feat(combatvocab): the four combat axes and the eligibility table

One declaration each of AttackType, DamageType, Targeting and Defence, and
the defence eligibility table keyed by the attack-and-damage pair. Nothing
consumes it yet.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

## Task 2: Derived pools in `combat`

`DamageChannel` stays. Two functions derive it from the axes; parity tests pin them against the three switches they will replace in Task 5.

**Files:**
- Create: `internal/combat/pools.go`
- Create: `internal/combat/pools_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// The scale channel is what CalcRawDamage and DamageScale key on, and it is
// ALSO the toughen channel. Pinned against master 612b85d54:
//   - calcSpellDamageForCharacter always passes ChannelMagical
//     (hooks/combat_shared_helpers.go:52), physical spells included;
//   - channelDamageChannel (defence_multiplier.go:769) answered "physical"
//     for melee and ranged, "magical" for BOTH spell channels, "conviction"
//     for social.
func TestScaleChannelForMatchesTheOldSwitches(t *testing.T) {
	cases := map[combatvocab.AttackType]DamageChannel{
		combatvocab.AttackMelee:    ChannelPhysical,
		combatvocab.AttackRanged:   ChannelPhysical,
		combatvocab.AttackThrown:   ChannelPhysical,
		combatvocab.AttackSpell:    ChannelMagical,
		combatvocab.AttackRhetoric: ChannelConviction,
	}
	for at, want := range cases {
		if got := ScaleChannelFor(at); got != want {
			t.Errorf("ScaleChannelFor(%s) = %v, want %v", at, got, want)
		}
	}
}

func TestToughenNameMatchesCharactersToughenStatForInputs(t *testing.T) {
	cases := map[DamageChannel]string{
		ChannelPhysical:   "physical",
		ChannelMagical:    "magical",
		ChannelConviction: "conviction",
	}
	for ch, want := range cases {
		if got := ch.ToughenName(); got != want {
			t.Errorf("%v.ToughenName() = %q, want %q", ch, got, want)
		}
	}
}

// Mitigation is keyed by the DAMAGE type. Pinned against the switch at
// hooks/combat_shared_helpers.go:87-96 on master: physical -> physical
// mitigation, mental -> magical mitigation. Social answers conviction, the
// pool taunt already mitigates on (actions/combat_taunt.go:251); no shipped
// social spell deals damage, so this is a rule for the next one, not a
// change for any of today's.
func TestMitigationChannelForIsKeyedByDamageType(t *testing.T) {
	cases := map[combatvocab.DamageType]DamageChannel{
		combatvocab.DamagePhysical: ChannelPhysical,
		combatvocab.DamageMental:   ChannelMagical,
		combatvocab.DamageSocial:   ChannelConviction,
	}
	for dt, want := range cases {
		got, ok := MitigationChannelFor(dt)
		if !ok || got != want {
			t.Errorf("MitigationChannelFor(%s) = %v, %v; want %v, true", dt, got, ok, want)
		}
	}
	if _, ok := MitigationChannelFor(combatvocab.DamageNonHarm); ok {
		t.Error("non_harm has no mitigation pool; it must never reach the damage pipeline")
	}
}

// Every declared attack type except none must scale somewhere, and none must
// not scale at all: a non-harm cast never reaches CalcRawDamage.
func TestEveryHarmAttackTypeHasAScalePool(t *testing.T) {
	for _, at := range combatvocab.AttackTypes() {
		got, ok := scaleChannelFor(at)
		if at == combatvocab.AttackNone {
			if ok {
				t.Errorf("AttackNone must have no scale pool, got %v", got)
			}
			continue
		}
		if !ok {
			t.Errorf("AttackType %s has no scale pool; add it to pools.go", at)
		}
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/combat/ -run 'ScaleChannelFor|ToughenName|MitigationChannelFor|HarmAttackType' 2>&1 | head -5
```
Expected: `undefined: ScaleChannelFor`.

- [ ] **Step 3: Write `pools.go`**

```go
package combat

import (
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// DamageChannel is NOT one of the four authored axes. It is the damage
// pipeline's pool: which scale knob, which mitigation cap, which stat
// toughens on a defensive crit. Nothing declares it in data; these two
// functions derive it from combatvocab (owner ruling 2026-09-18), replacing
// three switches that used to encode the same facts by hand.

// scaleChannelFor is the table behind ScaleChannelFor, with the ok the guard
// test wants.
func scaleChannelFor(at combatvocab.AttackType) (DamageChannel, bool) {
	switch at {
	case combatvocab.AttackMelee, combatvocab.AttackRanged, combatvocab.AttackThrown:
		return ChannelPhysical, true
	case combatvocab.AttackSpell:
		return ChannelMagical, true
	case combatvocab.AttackRhetoric:
		return ChannelConviction, true
	}
	return ChannelPhysical, false
}

// ScaleChannelFor returns the pool an attack SCALES on (CalcRawDamage,
// DamageScale) and the pool whose stat TOUGHENS on a defensive crit against
// it. Both are properties of how the attack is delivered, not of what it
// does: a physical spell is dodged but is still cast off willpower, so it
// scales magically and toughens willpower. (The old channelDamageChannel's
// comment warned that mapping spell-physical to "physical" would toughen the
// wrong stat; this keeps that mapping.)
//
// AttackNone never reaches the pipeline. It answers Physical here only so a
// caller that does reach it cannot divide by a zero scale; the error log is
// the tell.
func ScaleChannelFor(at combatvocab.AttackType) DamageChannel {
	ch, ok := scaleChannelFor(at)
	if !ok {
		mudlog.Error("ScaleChannelFor", "attack_type", string(at), "error", "no scale pool; a non-harm attack reached the damage pipeline")
	}
	return ch
}

// MitigationChannelFor returns the pool that MITIGATES a damage type: the
// defender's physical mitigation against physical harm, magical against
// mental, conviction against social. non_harm has none and returns false;
// a non-harm cast takes the uncontested path before any damage helper runs.
func MitigationChannelFor(dt combatvocab.DamageType) (DamageChannel, bool) {
	switch dt {
	case combatvocab.DamagePhysical:
		return ChannelPhysical, true
	case combatvocab.DamageMental:
		return ChannelMagical, true
	case combatvocab.DamageSocial:
		return ChannelConviction, true
	}
	return ChannelPhysical, false
}

// ToughenName is the string characters.ToughenStatFor expects. characters
// cannot import combat, so the string crosses that boundary; this is the one
// place it is spelt on this side.
func (c DamageChannel) ToughenName() string {
	switch c {
	case ChannelPhysical:
		return "physical"
	case ChannelMagical:
		return "magical"
	case ChannelConviction:
		return "conviction"
	}
	return ""
}
```

- [ ] **Step 4: Run the tests**

```bash
go test ./internal/combat/ -run 'ScaleChannelFor|ToughenName|MitigationChannelFor|HarmAttackType' -v 2>&1 | grep -E "^(--- |ok|FAIL)"
```
Expected: 4 PASS, `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/combat/pools.go internal/combat/pools_test.go
git commit -m "feat(combat): derive the damage pool from the axes

ScaleChannelFor and MitigationChannelFor, pinned against the three switches
they will replace. DamageChannel stays: it is the pipeline's pool, not an
authored axis.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

## Task 3: `items.DefenseType` becomes `items.DefencePool`

The message store's key type keeps only pool names. The five defence pools are named from `combatvocab.Defence`; the four counter pools are its own constants.

**Files:**
- Modify: `internal/items/defensive_messages.go:10-34,67,115,204`
- Modify: `internal/items/test_helpers_combat.go:29-58`
- Modify: `internal/combat/defence_multiplier.go:298`, `combat_helpers.go:1276-1310`, `counter.go:161-172,252`
- Modify: `internal/items/*_test.go` and `internal/combat/*_test.go` that name `items.Defense*`

- [ ] **Step 1: Write the failing test** (`internal/items/defence_pool_test.go`)

```go
package items

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// The five defence pools are keyed by the defence's own name, so the YAML
// files under defense-messages/ (dodge.yaml ...) do not move.
func TestDefencePoolForIsTheDefenceName(t *testing.T) {
	for _, d := range combatvocab.Defences() {
		if got := DefencePoolFor(d); string(got) != string(d) {
			t.Errorf("DefencePoolFor(%s) = %q", d, got)
		}
	}
	if got := DefencePoolFor(combatvocab.DefenceNone); got != "" {
		t.Errorf("DefencePoolFor(none) = %q, want empty", got)
	}
}

func TestCounterPoolNamesAreTheShippedFileNames(t *testing.T) {
	want := map[DefencePool]bool{"counter-melee": true, "counter-ranged": true, "counter-quell": true, "counter-defy": true}
	for _, p := range []DefencePool{CounterPoolMelee, CounterPoolRanged, CounterPoolQuell, CounterPoolDefy} {
		if !want[p] {
			t.Errorf("counter pool %q is not a shipped file name", p)
		}
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/items/ -run DefencePool 2>&1 | head -3
```
Expected: `undefined: DefencePoolFor`.

- [ ] **Step 3: Replace lines 10-34 of `defensive_messages.go`**

```go
var (
	defenseMessages map[DefencePool]*DefenseMessageGroup = map[DefencePool]*DefenseMessageGroup{}
)

// DefencePool is the KEY of the defense-messages/ store: the five defence
// pools, named from combatvocab.Defence so the files do not move, plus the
// four counter pools. It is not a defence type; that vocabulary lives in
// internal/combatvocab and this package only converts INTO its key.
type DefencePool string

// DefencePoolFor names the pool that narrates a defence. DefenceNone maps to
// the empty pool, which RenderDefenseMessage answers with an empty triad.
func DefencePoolFor(d combatvocab.Defence) DefencePool {
	return DefencePool(d)
}

const (
	// Counter-narration pools (U6b Task 11). Not defences: each is the
	// narration for the counter EARNED by a defensive crit. They ride the same
	// loader, shape, and validator as the defence pools. Band semantics
	// differ: weak = the counter is turned aside (no damage), normal = the
	// counter lands, heavy = the counter crits.
	//
	// Keyed by the ORIGINAL attack's type until the counters slice re-keys
	// them to the defence that won.
	CounterPoolMelee  DefencePool = "counter-melee"
	CounterPoolRanged DefencePool = "counter-ranged"
	CounterPoolQuell  DefencePool = "counter-quell"
	CounterPoolDefy   DefencePool = "counter-defy"
)
```

Add `"github.com/GoMudEngine/GoMud/internal/combatvocab"` to the imports. Then, in the same file, rename every remaining `DefenseType` to `DefencePool`: the `OptionId` field type (line 37), `Id()` return type (line 67), the `RenderDefenseMessage` and `GetDefenseMessage` parameters (lines 115, 204). `sed` does it safely because the identifier appears nowhere else in the file:

```bash
sed -i 's/\bDefenseType\b/DefencePool/g' internal/items/defensive_messages.go
grep -n "DefenseType" internal/items/defensive_messages.go   # expect nothing
```

- [ ] **Step 4: Fix `test_helpers_combat.go`**

`SeedDefenseMessagesForTest` takes `map[DefencePool]*DefenseMessageGroup`; `MinimalDefenseMessageFixture` returns the same and its `all` slice becomes:

```go
	all := []DefencePool{
		DefencePoolFor(combatvocab.DefenceDodge), DefencePoolFor(combatvocab.DefenceParry),
		DefencePoolFor(combatvocab.DefenceBlock), DefencePoolFor(combatvocab.DefenceQuell),
		DefencePoolFor(combatvocab.DefenceDefy),
		CounterPoolMelee, CounterPoolRanged, CounterPoolQuell, CounterPoolDefy,
	}
```

- [ ] **Step 5: Let the compiler enumerate the rest**

```bash
go build ./... 2>&1 | grep -v "^#" | head -40
```
Fix each reported site. Known sites and their replacements:

| Site | Old | New |
|---|---|---|
| `combat/defence_multiplier.go:298` | `items.DefenseType(out.DefenceType)` | `items.DefencePool(out.DefenceType)` (Task 4 makes this `items.DefencePoolFor(out.Defence)`) |
| `combat/combat_helpers.go:1276` | `var itemsDefenseType items.DefenseType` | `var itemsDefenseType items.DefencePool` |
| `combat/combat_helpers.go:1280,1283,1286` | `items.DefenseDodge` etc. | `items.DefencePoolFor(combatvocab.DefenceDodge)` etc. |
| `combat/counter.go:161-172` | `items.DefenseType` return, `items.DefenseCounter*` | `items.DefencePool`, `items.CounterPool*` |
| `combat/counter.go:252` | `items.DefenseCounterDefy` | `items.CounterPoolDefy` |
| any `items.DefenseDodge`-style constant elsewhere | | `items.DefencePoolFor(combatvocab.DefenceX)` |

Then the tests:

```bash
go vet ./... 2>&1 | grep -v "^#" | head -40
```
Apply the same table to every `_test.go` it names (`combat/counter_social_pool_test.go:26-28` uses `items.DefenseCounter*`; `items` tests use `DefenseDodge`; `combat/channel_defence_messages_test.go` seeds the fixture map). `sed` handles the mechanical part:

```bash
grep -rl --include=*_test.go -E "items\.Defense(Type|Dodge|Parry|Block|Quell|Defy|Counter)" internal | xargs sed -i \
  -e 's/items\.DefenseType/items.DefencePool/g' \
  -e 's/items\.DefenseCounterMelee/items.CounterPoolMelee/g' \
  -e 's/items\.DefenseCounterRanged/items.CounterPoolRanged/g' \
  -e 's/items\.DefenseCounterQuell/items.CounterPoolQuell/g' \
  -e 's/items\.DefenseCounterDefy/items.CounterPoolDefy/g' \
  -e 's/items\.DefenseDodge/items.DefencePoolFor(combatvocab.DefenceDodge)/g' \
  -e 's/items\.DefenseParry/items.DefencePoolFor(combatvocab.DefenceParry)/g' \
  -e 's/items\.DefenseBlock/items.DefencePoolFor(combatvocab.DefenceBlock)/g' \
  -e 's/items\.DefenseQuell/items.DefencePoolFor(combatvocab.DefenceQuell)/g' \
  -e 's/items\.DefenseDefy/items.DefencePoolFor(combatvocab.DefenceDefy)/g'
```
Inside package `items` itself the constants are unqualified (`DefenseDodge`); apply the same substitutions without the `items.` prefix to `internal/items/*_test.go`, then add the `combatvocab` import wherever `go vet` says it is missing (`goimports -w` on the named files does it).

- [ ] **Step 6: Run everything**

```bash
go build ./... && go vet ./... && go test ./internal/items/ ./internal/combat/ ./internal/hooks/ 2>&1 | tail -5
```
Expected: `ok` for all three.

- [ ] **Step 7: Commit**

```bash
git add -u internal/items internal/combat internal/hooks
git add internal/items/defence_pool_test.go
git commit -m "refactor(items): DefenseType becomes DefencePool, the store key only

The five defence pools are named from combatvocab.Defence; the four counter
pools keep their file names. Nothing player-visible changes.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

(`git add -u <dir>` stages only tracked files under the directory, which satisfies the named-paths rule; never `git add -A` or `git add .`.)

---

## Task 4: One `Defence` type

Delete `characters.Defense*` and `combat.DefenseType`; every `string`-typed defence parameter and field takes `combatvocab.Defence`.

**Files:**
- Modify: `internal/characters/character.go:727-741`, `combat.go:279,341`, `resources.go:140,178,201,246,297`
- Modify: `internal/combat/attackresult.go:8-15,88,118,180-181`, `defence_multiplier.go:135,190,225,298,550,677,726`, `defence_sets.go` (return types only; the table itself flips in Task 5), `combat.go:566`, `combat_helpers.go:100,656,710-732,1250-1254,1273-1286,1472-1481`, `calculations.go:101-103`, `surprise_narration.go:56-63`
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go:322-365`, `NewRound_DoCombat_unified.go:150,178`
- Modify: tests that name `characters.Defense*` or `combat.Defense*`

- [ ] **Step 1: Write the failing test** (`internal/characters/defence_type_test.go`)

```go
package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// Every real defence must score, cost and pool through the ONE vocabulary.
// GetDefenseScoreFor's default arm returns 0 and defenseCostRequest's returns
// false, so a defence the switches forgot would silently enter every contest
// at 0 and cost nothing. Only this assertion catches it.
func TestEveryDefenceScoresAndCosts(t *testing.T) {
	c := New()
	c.Stats.Dexterity.ValueAdj = 100
	c.Stats.Strength.ValueAdj = 100
	c.Stats.Willpower.ValueAdj = 100
	for _, d := range combatvocab.Defences() {
		if c.GetDefenseScore(d) <= 0 {
			t.Errorf("GetDefenseScore(%s) = 0: the switch has no arm for it", d)
		}
		if _, ok := defenseCostRequest(d); !ok {
			t.Errorf("defenseCostRequest(%s) unknown: the switch has no arm for it", d)
		}
	}
	if _, ok := defenseCostRequest(combatvocab.DefenceNone); ok {
		t.Error("DefenceNone must not price as anything")
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/characters/ -run TestEveryDefenceScoresAndCosts 2>&1 | head -3
```
Expected: `cannot use d (variable of type combatvocab.Defence) as string value`.

- [ ] **Step 3: Delete the six constants in `character.go:727-741`** (the whole `const (...)` block under "Stage 7.1: Segmented Defense Helper Methods", comment included) and retype the characters functions:

```go
func (c *Character) GetDefenseScoreFor(defenseType combatvocab.Defence, includeSkill bool) float64 {
	...
	switch defenseType {
	case combatvocab.DefenceDodge:
	...
	case combatvocab.DefenceParry:
	...
	case combatvocab.DefenceBlock:
	...
	case combatvocab.DefenceQuell:
	...
	case combatvocab.DefenceDefy:
```

```go
func (c *Character) GetDefenseScore(defenseType combatvocab.Defence) float64
func defenseCostRequest(defenseType combatvocab.Defence) (ActionCostRequest, bool)   // cases combatvocab.DefenceDodge ... DefenceDefy
func (c *Character) QuoteDefenseCost(defenseType combatvocab.Defence) (CostQuote, bool)
func DefensePool(defenseType combatvocab.Defence) Pool
func (c *Character) GetDefenseCostFloat(defenseType combatvocab.Defence) float64
func (c *Character) GetDefenseCost(defenseType combatvocab.Defence) int
```

Add the `combatvocab` import to `combat.go` and `resources.go`.

- [ ] **Step 4: Delete `combat.DefenseType` (`attackresult.go:8-15`) and retype its fields**

```go
	DefenseUsed   combatvocab.Defence      // SwingEvent, line 88
	Defence combatvocab.Defence            // SwingDefence, line 118
	DefenseUsed             combatvocab.Defence     // AttackResult, line 180
	DefenseAttempts         []combatvocab.Defence   // line 181
```

- [ ] **Step 5: Retype the combat package by the compiler's list**

```bash
go build ./internal/combat/ 2>&1 | head -60
```
Work through every line. The complete list on master and what each becomes:

| File:line | Old | New |
|---|---|---|
| `defence_sets.go:51` | `func DefenceSetFor(channel AttackChannel) []string` | returns `[]combatvocab.Defence`; body uses `combatvocab.DefenceDodge` etc. |
| `defence_sets.go:106` | `equipmentGatedMeleeDefences(...) []string` | `[]combatvocab.Defence` |
| `defence_sets.go:125` | `thirdPartyGrappleDefences(defSeq []string) []string` | `[]combatvocab.Defence` |
| `defence_sets.go:151` | `DefenceEntriesFor(...) []string` | `[]combatvocab.Defence`; the inner `switch name { case characters.DefenseDodge, ...` uses `combatvocab.DefenceDodge, DefenceParry, DefenceBlock` |
| `defence_sets.go:187` | `countDefenceName(set []string, name string)` | both `combatvocab.Defence` |
| `defence_multiplier.go:135` | `DefenceSkillAndStat(defenceType string)` | `combatvocab.Defence`; cases `combatvocab.DefenceDodge` ... |
| `defence_multiplier.go:190` | `AwardDefenceProgression(c, userId, defenceType string, won bool)` | `combatvocab.Defence` |
| `defence_multiplier.go:203` | `if defenceType == characters.DefenseParry` | `combatvocab.DefenceParry` |
| `defence_multiplier.go:225` | `DefenceType string` | `Defence combatvocab.Defence` |
| `defence_multiplier.go:298` | `items.DefencePool(out.DefenceType)` | `items.DefencePoolFor(out.Defence)` |
| `defence_multiplier.go:305` | `logMissingDefencePool(out.DefenceType)` | `logMissingDefencePool(string(out.Defence))` (or retype the helper) |
| `defence_multiplier.go:446-460` | `for _, d := range defences { defender.QuoteDefenseCost(d) ... GetDefenseScoreFor(d, ...) * defenceEffectiveness(d)` | `d` is now `combatvocab.Defence`; `defenceEffectiveness` and the contest `Entry{Name: string(d)}` need the explicit `string(d)` |
| `defence_multiplier.go:550` | `out.DefenceType = res.Winner` | `out.Defence = combatvocab.Defence(res.Winner)` |
| `defence_multiplier.go:677` | `defenderRankOf(defender, winner string)` | takes `combatvocab.Defence`; call sites pass `combatvocab.Defence(res.Winner)` |
| `defence_multiplier.go:726` | `DefenceSkillAndStat(res.Winner)` | `DefenceSkillAndStat(combatvocab.Defence(res.Winner))` |
| `combat_helpers.go:100` | `defenseType string` | `defenseType combatvocab.Defence` |
| `combat_helpers.go:648` | `defSeq []string` | `[]combatvocab.Defence` |
| `combat_helpers.go:656` | `DefenseType(defenseType)` | `defenseType` |
| `combat_helpers.go:710-732` | `characters.DefenseDodge` etc. (9 lines) | `combatvocab.DefenceDodge` etc. |
| `combat_helpers.go:1250-1254` | `characters.DefenseParry/Dodge/Block` | `combatvocab.DefenceParry/Dodge/Block` |
| `combat_helpers.go:1273` | `result.DefenseUsed = DefenseType(best.defenseType)` | `= best.defenseType` |
| `combat_helpers.go:1278-1286` | `case characters.DefenseDodge: ... itemsDefenseType = items.DefencePoolFor(combatvocab.DefenceDodge)` | the switch collapses: `itemsDefenseType := items.DefencePoolFor(best.defenseType)` and `defenseVerb = string(best.defenseType)`, keeping the `"counter"` fallback for `DefenceNone` |
| `combat_helpers.go:1472` | `deflectedSwingLines(defense DefenseType, ...)` | `combatvocab.Defence`; cases `combatvocab.DefenceDodge` etc. |
| `combat.go:566` | `Defence: DefenseType(best.defenseType)` | `Defence: best.defenseType` |
| `combat.go:465` | `defenseSequence := DefenceEntriesFor(...)` | unchanged, type flows |
| `calculations.go:101-103` | `characters.DefenseDodge` etc. | `combatvocab.DefenceDodge` etc. |
| `surprise_narration.go:56` | `openingStrikeDefendedLines(defense DefenseType, ...)` | `combatvocab.Defence`; cases retyped |
| `counter.go` | no defence-name switch | unchanged |

Any site the table missed, the compiler names. Where `res.Winner` (a `string` from `internal/contest`) meets a `Defence`, convert with `combatvocab.Defence(...)`; where an `Entry.Name` needs a string, `string(d)`.

- [ ] **Step 6: Retype hooks**

`NewRound_DoCombat_helpers.go:322-365`: every `combat.DefenseType` becomes `combatvocab.Defence`, `combat.DefenseNone` becomes `combatvocab.DefenceNone`, the fixed order slice becomes `[]combatvocab.Defence{combatvocab.DefenceDodge, combatvocab.DefenceParry, combatvocab.DefenceBlock}`, and `defenceSkillFor`/`defenceStatFor` pass `used[0]` without the `string(...)` cast.

`NewRound_DoCombat_unified.go:150,178`: `combat.DefenseDodge` etc. become `combatvocab.DefenceDodge` etc.

```bash
go build ./... 2>&1 | head -40
```
Expected: clean.

- [ ] **Step 7: Sweep the tests**

```bash
grep -rl --include=*_test.go -E "characters\.Defense(None|Dodge|Parry|Block|Quell|Defy)\b|combat\.Defense(Type|None|Dodge|Parry|Block)\b" internal | xargs sed -i \
  -e 's/characters\.DefenseNone/combatvocab.DefenceNone/g' \
  -e 's/characters\.DefenseDodge/combatvocab.DefenceDodge/g' \
  -e 's/characters\.DefenseParry/combatvocab.DefenceParry/g' \
  -e 's/characters\.DefenseBlock/combatvocab.DefenceBlock/g' \
  -e 's/characters\.DefenseQuell/combatvocab.DefenceQuell/g' \
  -e 's/characters\.DefenseDefy/combatvocab.DefenceDefy/g' \
  -e 's/combat\.DefenseType/combatvocab.Defence/g' \
  -e 's/combat\.DefenseNone/combatvocab.DefenceNone/g' \
  -e 's/combat\.DefenseDodge/combatvocab.DefenceDodge/g' \
  -e 's/combat\.DefenseParry/combatvocab.DefenceParry/g' \
  -e 's/combat\.DefenseBlock/combatvocab.DefenceBlock/g'
```
Inside packages `characters` and `combat` the constants are unqualified; apply the unprefixed forms to `internal/characters/*_test.go` and `internal/combat/*_test.go` (`DefenseDodge` -> `combatvocab.DefenceDodge`, `DefenseType` -> `combatvocab.Defence`, `DefenseNone` -> `combatvocab.DefenceNone`). Then:

```bash
goimports -w $(grep -rl --include=*_test.go "combatvocab\." internal)
go vet ./... 2>&1 | grep -v "^#" | head -40
```
Fix what remains by hand: a test that compares a `Defence` to a `string` variable needs `string(...)`; a test building `[]string{"dodge"}` for a `[]combatvocab.Defence` parameter becomes `[]combatvocab.Defence{"dodge"}` (untyped constants convert, a `[]string` does not).

- [ ] **Step 8: Run the suite**

```bash
go build ./... && go vet ./... && go test ./... 2>&1 | grep -v "^ok" | head -20
```
Expected: no `FAIL` lines (the `TestDefenceSetForReturnsKnownDefenceNames` test's `known` map now keys on `combatvocab.Defence` and still passes).

- [ ] **Step 9: Commit**

```bash
git add -u internal
git add internal/characters/defence_type_test.go
git commit -m "refactor: one Defence type, from combatvocab

characters.Defense* and combat.DefenseType are deleted. Every defence
parameter and field is combatvocab.Defence; untyped test literals still
compile. Nothing player-visible changes.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

## Task 5: `combatvocab.Attack` replaces `AttackChannel` through the seam

The big routing flip. Spells keep their legacy fields until Task 7; `spellAttackChannel` is rewritten to return an `Attack` built from them, so this task changes no data.

**Files:**
- Modify: `internal/combat/defence_sets.go` (delete `AttackChannel`, the constants and `DefenceSetFor`; `DefenceEntriesFor` reads the table)
- Modify: `internal/combat/defence_multiplier.go:415-460,733,753-778`, `skill_moves.go:60-70,189`, `situational.go:35-51`, `counter.go:16-30,89-90,161-172,191-193`
- Modify: `internal/actions/combat_bash.go`, `combat_drain.go`, `combat_fire.go`, `combat_gore.go`, `combat_hamstring.go`, `combat_kick.go`, `combat_maul.go`, `combat_pounce.go`, `combat_rake.go`, `combat_throttle.go`, `combat_trip.go`, `combat_taunt.go`, `combat_counter.go`
- Modify: `internal/hooks/spell_resolution.go:334-356,1241-1262` and the four `fireSpellCounterTier` calls, `combat_shared_helpers.go:335,394`, `counter_tier.go:34-36`
- Modify: `internal/usercommands/throw.go:327`
- Modify: 29 test files naming `Channel*`

- [ ] **Step 1: Write the failing seam test** (`internal/combat/defence_entries_shape_test.go`)

```go
package combat

import (
	"reflect"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// DefenceEntriesFor reads the combatvocab table. For an unarmed, shieldless
// defender the equipment gate leaves dodge on the physical rows and the
// ungated defences everywhere else, so the result IS the table minus parry
// and block. Pinned literally so a wrong table row shows here, not in play.
func TestDefenceEntriesForReadsTheEligibilityTable(t *testing.T) {
	bare := characters.New()
	cases := []struct {
		shape combatvocab.Attack
		want  []combatvocab.Defence
	}{
		{combatvocab.Melee(combatvocab.TargetSingle), []combatvocab.Defence{combatvocab.DefenceDodge}},
		{combatvocab.Ranged(combatvocab.TargetSingle), []combatvocab.Defence{combatvocab.DefenceDodge}},
		{combatvocab.Thrown(combatvocab.TargetArea), []combatvocab.Defence{combatvocab.DefenceDodge}},
		{combatvocab.Spell(combatvocab.DamagePhysical, combatvocab.TargetSingle), []combatvocab.Defence{combatvocab.DefenceDodge}},
		{combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetSingle), []combatvocab.Defence{combatvocab.DefenceQuell}},
		{combatvocab.Spell(combatvocab.DamageSocial, combatvocab.TargetSingle), []combatvocab.Defence{combatvocab.DefenceDefy}},
		{combatvocab.Rhetoric(combatvocab.TargetSingle), []combatvocab.Defence{combatvocab.DefenceDefy}},
		{combatvocab.NonHarm(combatvocab.TargetSingle), []combatvocab.Defence{}},
	}
	for _, tc := range cases {
		got := DefenceEntriesFor(tc.shape, bare, DefenceEntryOpts{})
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("DefenceEntriesFor(%v) = %v, want %v", tc.shape, got, tc.want)
		}
	}
}

// An unknown pair resolves uncontested, exactly as DefenceSetFor's default
// arm did, but it is no longer silent.
func TestDefenceEntriesForUnknownPairIsEmptyNotPanic(t *testing.T) {
	bare := characters.New()
	bad := combatvocab.Attack{Type: combatvocab.AttackMelee, Damage: combatvocab.DamageMental, Targeting: combatvocab.TargetSingle}
	if got := DefenceEntriesFor(bad, bare, DefenceEntryOpts{}); len(got) != 0 {
		t.Errorf("unknown pair produced defences %v", got)
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/combat/ -run 'DefenceEntriesForReads|UnknownPairIsEmpty' 2>&1 | head -3
```
Expected: `cannot use combatvocab.Melee(...) (value of type combatvocab.Attack) as AttackChannel value`.

- [ ] **Step 3: Rewrite `defence_sets.go`**

Delete `AttackChannel`, its five constants and `DefenceSetFor` (lines 5-65). Keep the long comment above `DefenceSetFor` about the three things a new defence must carry: move it above `DefenceEntriesFor`, reworded to say the table lives in `combatvocab.EligibleDefences`. `DefenceEntriesFor` becomes:

```go
func DefenceEntriesFor(shape combatvocab.Attack, defender *characters.Character, opts DefenceEntryOpts) []combatvocab.Defence {
	if defender == nil {
		return nil
	}

	set, ok := combatvocab.EligibleDefences(shape)
	if !ok {
		// A pair the table does not know. The constructors cannot build one
		// and the spell validator refuses one, so this is a struct literal
		// somewhere. Uncontested is what the old default arm did; the log is
		// what it did not.
		mudlog.Error("DefenceEntriesFor", "shape", shape.String(), "error", "attack pair is not in the eligibility table; resolving uncontested")
		return []combatvocab.Defence{}
	}

	gated := equipmentGatedMeleeDefences(defender)

	entries := []combatvocab.Defence{}
	for _, name := range set {
		switch name {
		case combatvocab.DefenceDodge, combatvocab.DefenceParry, combatvocab.DefenceBlock:
			for i := 0; i < countDefenceName(gated, name); i++ {
				entries = append(entries, name)
			}
		default:
			entries = append(entries, name)
		}
	}

	if opts.ThirdPartyVsGrappler {
		entries = thirdPartyGrappleDefences(entries)
	}

	return entries
}
```

Imports: add `combatvocab` and `mudlog`.

- [ ] **Step 4: Rewrite the rest of `combat`**

`defence_multiplier.go`:
- `ResolveChannelAttack(shape combatvocab.Attack, side AttackSide, attacker, defender *characters.Character)` and `resolveChannelAttackWithRunner(shape combatvocab.Attack, ...)`; the `DefenceEntriesFor(channel, ...)` call passes `shape`.
- `awardChannelDefenceBonus`'s `channel AttackChannel` parameter becomes `shape combatvocab.Attack`; line 733 becomes `ToughenStat: characters.ToughenStatFor(ScaleChannelFor(shape.Type).ToughenName()),`.
- Delete `channelDamageChannel` (lines 753-778) and its comment.
- Every other `channel AttackChannel` parameter in this file (grep `AttackChannel` for the list; 12 lines on master) becomes `shape combatvocab.Attack`.

`skill_moves.go:60-70`:

```go
	// Shape selects the defence set through combatvocab.EligibleDefences
	// (Melee(TargetSingle) for the physical moves, Ranged for fire) and
	// carries the targeting the counters slice reads. Required: every caller
	// sets Shape + Attack. It is not named Attack because that field is the
	// attacker's AttackSide.
	Shape combatvocab.Attack
```
and line 189: `resolveChannelAttackWithRunner(p.Shape, p.Attack, p.Attacker, p.Defender, runner)`.

`situational.go:35-51`:

```go
func SituationalAttackMult(attacker *characters.Character, shape combatvocab.Attack) float64 {
	if attacker == nil {
		return 1.0
	}
	mult := 1.0
	switch shape.Type {
	case combatvocab.AttackMelee, combatvocab.AttackRanged, combatvocab.AttackThrown:
```

`counter.go`:
- `CounterResult.Channel AttackChannel` (line 24-26) becomes `Shape combatvocab.Attack` with the comment "the ORIGINAL attack, kept for pool selection".
- `ExecuteCounter(defender, attacker *characters.Character, shape combatvocab.Attack, sameRoom bool)`; line 90 `result := CounterResult{Shape: shape}`; the counter-swing's params gain `Shape: combatvocab.Melee(combatvocab.TargetSingle),` in place of `Channel: ChannelMelee,`.
- `counterPoolFor`:

```go
// counterPoolFor maps the ORIGINAL attack's type to its counter-narration
// pool. Keyed by attack type until the counters slice re-keys the pools to
// the defence that won (spec ruling 5). Thrown shares ranged's pool; both
// spell damage types share the put-the-working-down pool as before.
func counterPoolFor(shape combatvocab.Attack) items.DefencePool {
	switch shape.Type {
	case combatvocab.AttackRanged, combatvocab.AttackThrown:
		return items.CounterPoolRanged
	case combatvocab.AttackSpell:
		return items.CounterPoolQuell
	case combatvocab.AttackRhetoric:
		return items.CounterPoolDefy
	default:
		return items.CounterPoolMelee
	}
}
```
- `fillCounterMessages` line 193: `counterPoolFor(result.Shape)`.

```bash
go build ./internal/combat/ 2>&1 | head
```
Expected: clean. (`combat/analytics.go`'s `AttackType string` field is the per-swing weapon label and is untouched.)

- [ ] **Step 5: Rewrite the 17 call sites**

Each `Channel: combat.ChannelMelee,` in `internal/actions/combat_{bash,drain,gore,hamstring,kick,maul,pounce,rake,throttle,trip}.go` and `internal/hooks/combat_shared_helpers.go:335,394` becomes `Shape: combatvocab.Melee(combatvocab.TargetSingle),`, EXCEPT `combat_drain.go:289` (inside `ExecuteDrainArea`'s loop), which becomes `Shape: combatvocab.Melee(combatvocab.TargetArea),` (Task 9 moves it again). Every `combat.SituationalAttackMult(char, combat.ChannelMelee)` on those lines becomes `combat.SituationalAttackMult(char, combatvocab.Melee(combatvocab.TargetSingle))` (area for the drain loop).

`combat_fire.go:396,400,417`: `Shape: combatvocab.Ranged(combatvocab.TargetSingle)`, the same in `SituationalAttackMult`, and `counterSkillMoveExit(actor, defChar, result.MoveResult, combatvocab.Ranged(combatvocab.TargetSingle), !crossRoom)`.

`combat_drain.go:302`: `counterSkillMoveExit(actor, target.Character, moveResult, combatvocab.Melee(combatvocab.TargetArea), true)`.

`combat_taunt.go:173,176`: `combatvocab.Rhetoric(combatvocab.TargetSingle)` in both.

`combat_counter.go:56`: `counterSkillMoveExit(actor Actor, defender *characters.Character, move combat.SkillMoveResult, shape combatvocab.Attack, sameRoom bool)`; line 183: `combat.ResolveChannelAttack(combatvocab.Rhetoric(combatvocab.TargetSingle), side, counterer, target)`.

`usercommands/throw.go:327`: `combat.ResolveChannelAttack(combatvocab.Thrown(combatvocab.TargetArea), side, user.Character, &mob.Character)` (ruling 9: the throw command passes `thrown`; the throw is an area effect by this file's own comment at line 283).

A sed handles the two commonest lines; the rest are by hand from the list above:

```bash
grep -rl "combat\.ChannelMelee" internal/actions internal/hooks | xargs sed -i \
  -e 's/Channel:  combat\.ChannelMelee,/Shape: combatvocab.Melee(combatvocab.TargetSingle),/' \
  -e 's/combat\.SituationalAttackMult(char, combat\.ChannelMelee)/combat.SituationalAttackMult(char, combatvocab.Melee(combatvocab.TargetSingle))/'
```
then fix `combat_drain.go:289,293` to `TargetArea` by hand, and `goimports -w` the touched files.

- [ ] **Step 6: Rewrite hooks**

`spell_resolution.go:1241-1262`, transitional until Task 7 (the legacy fields still drive routing, so this is the old switch restated as an `Attack`):

```go
// spellAttackShape builds the attack the seam resolves from the spell's
// legacy fields. TRANSITIONAL: Task 7 of the M4b-2 plan replaces it with
// SpellData.Attack() reading attack_type/damage_type/targeting.
func spellAttackShape(spellData *spells.SpellData) combatvocab.Attack {
	targeting := combatvocab.TargetSingle
	if spellData != nil {
		switch spellData.Type {
		case spells.HarmArea, spells.HelpArea:
			targeting = combatvocab.TargetArea
		case spells.HarmMulti, spells.HelpMulti:
			targeting = combatvocab.TargetMulti
		case spells.Neutral:
			targeting = combatvocab.TargetSelf
		}
	}
	if spellData == nil {
		return combatvocab.Spell(combatvocab.DamageMental, targeting)
	}
	switch spellData.TargetDefenseType {
	case "physical":
		return combatvocab.Spell(combatvocab.DamagePhysical, targeting)
	case "social":
		return combatvocab.Spell(combatvocab.DamageSocial, targeting)
	}
	// An absent target_defense_type is the DEFAULT, not an escape from
	// routing: every unclassified spell resolves as a mental attack.
	return combatvocab.Spell(combatvocab.DamageMental, targeting)
}
```
Rename every `spellAttackChannel(` call to `spellAttackShape(` (`:354,406,445,963,1566,1791` and the mob resolvers).

`counter_tier.go:34-36`: `channel combat.AttackChannel` becomes `shape combatvocab.Attack`; `combat.ExecuteCounter(defender, caster, shape, true)`.

`charm_spell.go` and `cast.go` only mention the channel in comments; update the words (`ChannelSocial` -> "the spell/social pair").

```bash
go build ./... 2>&1 | head -20
```
Expected: clean.

- [ ] **Step 7: Sweep the 29 test files**

```bash
grep -rl --include=*_test.go -E "Channel(Melee|Ranged|SpellPhysical|SpellMental|Social)\b|AttackChannel" internal | xargs sed -i \
  -e 's/combat\.ChannelMelee/combatvocab.Melee(combatvocab.TargetSingle)/g' \
  -e 's/combat\.ChannelRanged/combatvocab.Ranged(combatvocab.TargetSingle)/g' \
  -e 's/combat\.ChannelSpellPhysical/combatvocab.Spell(combatvocab.DamagePhysical, combatvocab.TargetSingle)/g' \
  -e 's/combat\.ChannelSpellMental/combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetSingle)/g' \
  -e 's/combat\.ChannelSocial/combatvocab.Rhetoric(combatvocab.TargetSingle)/g' \
  -e 's/\bChannelMelee\b/combatvocab.Melee(combatvocab.TargetSingle)/g' \
  -e 's/\bChannelRanged\b/combatvocab.Ranged(combatvocab.TargetSingle)/g' \
  -e 's/\bChannelSpellPhysical\b/combatvocab.Spell(combatvocab.DamagePhysical, combatvocab.TargetSingle)/g' \
  -e 's/\bChannelSpellMental\b/combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetSingle)/g' \
  -e 's/\bChannelSocial\b/combatvocab.Rhetoric(combatvocab.TargetSingle)/g' \
  -e 's/combat\.AttackChannel/combatvocab.Attack/g' \
  -e 's/\bAttackChannel\b/combatvocab.Attack/g' \
  -e 's/\bChannel:\(\s*\)combatvocab\./Shape:\1combatvocab./g'
goimports -w $(grep -rl --include=*_test.go "combatvocab\." internal)
go vet ./... 2>&1 | grep -v "^#" | head -40
```
By hand afterwards:
- `internal/combat/defence_sets_test.go`: delete `TestDefenceSetFor`, `TestDefenceSetForUnknownChannel` and `TestDefenceSetForReturnsKnownDefenceNames` (their job is now `combatvocab.TestEligibilityTableIsTheSpecsTable`, `TestUnknownPairIsNotInTheTable` and `characters.TestEveryDefenceScoresAndCosts`; say so in the commit message).
- `internal/combat/counter_social_pool_test.go`: the map keys become `combatvocab.Attack` values; add a `Thrown` row expecting `CounterPoolRanged`.
- `internal/hooks/charm_channel_test.go` (`TestSpellAttackChannel_Routing`): rename to `TestSpellAttackShape_Routing`, expect `combatvocab.Spell(combatvocab.DamageSocial, combatvocab.TargetSingle)` for social, `DamagePhysical` for physical, `DamageMental` for mental, `""` and `"none"`.
- `internal/combat/contest_sign_test.go:199` switches on the shape's `Damage`: `case combatvocab.DamageMental, combatvocab.DamagePhysical:` on `shape.Damage`, guarded by `shape.Type == combatvocab.AttackSpell`.
- Any test that built `AttackChannel("not-a-channel")` is testing the deleted default arm; delete it.

```bash
go vet ./... && go test ./... 2>&1 | grep -v "^ok" | head -20
```
Expected: no FAIL.

- [ ] **Step 8: The goldens are untouched**

```bash
for f in internal/narration/testdata/stores/*.golden; do cmp -s "$f" "C:/tmp/m4b2-goldens/$(basename "$f")" || echo "DIFF $f"; done
```
Expected: no output.

- [ ] **Step 9: Commit**

```bash
git add -u internal
git add internal/combat/defence_entries_shape_test.go
git commit -m "refactor(combat): combatvocab.Attack replaces AttackChannel through the seam

The flattened channel enum is deleted. Every attack carries its type,
damage and targeting; eligibility comes from the combatvocab table; the
scale, mitigation and toughen pools derive from the axes. Spells still
route from their legacy fields through a transitional shape builder.
Three DefenceSetFor tests are deleted because their job moved to
combatvocab and characters. Nothing player-visible changes.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

## Task 6: Spells learn the axes (additive)

`SpellData` gains the three fields and their methods. YAML gains the three keys BESIDE the old two, via the tool. Nothing reads the new fields in production yet except `Attack()`, which prefers them when present.

**Files:**
- Modify: `internal/spells/spells.go:20-45` (three fields)
- Create: `internal/spells/axes.go`, `internal/spells/axes_test.go`
- Create: `tools/spell_axes_rewrite.py`
- Modify: `_datafiles/world/dogmud/spells/*.yaml`, `_datafiles/world/default/spells/*.yaml`

- [ ] **Step 1: Write the failing tests** (`internal/spells/axes_test.go`)

```go
package spells

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// The seven legacy SpellType values, mapped through the rewrite table, must
// print EXACTLY what SpellType's two display methods printed on master
// 612b85d54 (spells.go:103-150). Pinned as literals: the `spells` listing
// and the help template are player-visible.
func TestDisplayStringsMatchTheLegacyTypes(t *testing.T) {
	cases := []struct {
		legacy    string
		attack    combatvocab.AttackType
		damage    combatvocab.DamageType
		targeting combatvocab.Targeting
		helpOrHarm, short, long string
	}{
		{"neutral", combatvocab.AttackNone, combatvocab.DamageNonHarm, combatvocab.TargetSelf, "Neutral", "Self", "Self"},
		{"harmsingle", combatvocab.AttackSpell, combatvocab.DamageMental, combatvocab.TargetSingle, "Harmful", "Single", "Single Target"},
		{"harmmulti", combatvocab.AttackSpell, combatvocab.DamageMental, combatvocab.TargetMulti, "Harmful", "Group", "Group Target"},
		{"helpsingle", combatvocab.AttackNone, combatvocab.DamageNonHarm, combatvocab.TargetSingle, "Helpful", "Single", "Single Target"},
		{"helpmulti", combatvocab.AttackNone, combatvocab.DamageNonHarm, combatvocab.TargetMulti, "Helpful", "Group", "Group Target"},
		{"harmarea", combatvocab.AttackSpell, combatvocab.DamagePhysical, combatvocab.TargetArea, "Harmful", "Area", "Area Target"},
		{"helparea", combatvocab.AttackNone, combatvocab.DamageNonHarm, combatvocab.TargetArea, "Helpful", "Area", "Area Target"},
	}
	for _, tc := range cases {
		s := &SpellData{AttackType: tc.attack, DamageType: tc.damage, Targeting: tc.targeting}
		if got := s.HelpOrHarmString(); got != tc.helpOrHarm {
			t.Errorf("%s: HelpOrHarmString = %q, want %q", tc.legacy, got, tc.helpOrHarm)
		}
		if got := s.TargetTypeString(true); got != tc.short {
			t.Errorf("%s: TargetTypeString(true) = %q, want %q", tc.legacy, got, tc.short)
		}
		if got := s.TargetTypeString(); got != tc.long {
			t.Errorf("%s: TargetTypeString() = %q, want %q", tc.legacy, got, tc.long)
		}
	}
}

// DefenceNames replaces the template's defensename helper and must print
// what it printed (templates/templatesfunctions.go:100-113 on master).
func TestDefenceNamesMatchTheOldTemplateHelper(t *testing.T) {
	cases := map[combatvocab.DamageType]string{
		combatvocab.DamagePhysical: "dodge or block",
		combatvocab.DamageMental:   "quell",
		combatvocab.DamageSocial:   "defy",
	}
	for dt, want := range cases {
		s := &SpellData{AttackType: combatvocab.AttackSpell, DamageType: dt, Targeting: combatvocab.TargetSingle}
		if got := s.DefenceNames(); got != want {
			t.Errorf("DefenceNames(%s) = %q, want %q", dt, got, want)
		}
	}
	nonHarm := &SpellData{AttackType: combatvocab.AttackNone, DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle}
	if got := nonHarm.DefenceNames(); got != "" {
		t.Errorf("a non-harm spell must print no Resisted-by line, got %q", got)
	}
}

func TestIsHarmAndAttack(t *testing.T) {
	harm := &SpellData{AttackType: combatvocab.AttackSpell, DamageType: combatvocab.DamagePhysical, Targeting: combatvocab.TargetArea}
	if !harm.IsHarm() {
		t.Error("physical spell must be harm")
	}
	if got := harm.Attack(); got != combatvocab.Spell(combatvocab.DamagePhysical, combatvocab.TargetArea) {
		t.Errorf("Attack() = %v", got)
	}
	help := &SpellData{AttackType: combatvocab.AttackNone, DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSelf}
	if help.IsHarm() {
		t.Error("non_harm must not be harm")
	}
}

func TestValidateAxesRefusesBadData(t *testing.T) {
	bad := []SpellData{
		{SpellId: "missing", PrimaryStat: "willpower"},
		{SpellId: "pair", PrimaryStat: "willpower", AttackType: combatvocab.AttackMelee, DamageType: combatvocab.DamageMental, Targeting: combatvocab.TargetSingle},
		{SpellId: "none-harm", PrimaryStat: "willpower", AttackType: combatvocab.AttackNone, DamageType: combatvocab.DamagePhysical, Targeting: combatvocab.TargetSingle},
		{SpellId: "harm-nonharm", PrimaryStat: "willpower", AttackType: combatvocab.AttackSpell, DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle},
		{SpellId: "targeting", PrimaryStat: "willpower", AttackType: combatvocab.AttackSpell, DamageType: combatvocab.DamageMental, Targeting: "group"},
	}
	for i := range bad {
		if err := bad[i].validateAxes(); err == nil {
			t.Errorf("%s: validateAxes accepted bad axes", bad[i].SpellId)
		}
	}
	good := SpellData{SpellId: "ok", PrimaryStat: "willpower", AttackType: combatvocab.AttackSpell, DamageType: combatvocab.DamageSocial, Targeting: combatvocab.TargetArea}
	if err := good.validateAxes(); err != nil {
		t.Errorf("validateAxes refused a table pair: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/spells/ -run 'DisplayStrings|DefenceNames|IsHarmAndAttack|ValidateAxes' 2>&1 | head -3
```
Expected: `unknown field AttackType in struct literal`.

- [ ] **Step 3: Add the fields to `SpellData`** (after `Description`, before `Type`)

```go
	// The three authored axes (messaging M4b-2). All required; validated
	// against the combatvocab eligibility table at load, so a pair the table
	// does not know fails the boot. attack_type none pairs with damage_type
	// non_harm and is the uncontested cast (a heal is not an attack).
	// targeting self means NO target is resolved and the argument passes
	// through (summons, identify); single defaults to the caster.
	AttackType combatvocab.AttackType `yaml:"attack_type,omitempty"`
	DamageType combatvocab.DamageType `yaml:"damage_type,omitempty"`
	Targeting  combatvocab.Targeting  `yaml:"targeting,omitempty"`
```

- [ ] **Step 4: Write `internal/spells/axes.go`**

```go
package spells

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// Attack is the attack the seam resolves for this spell, built from the
// three axes.
func (s *SpellData) Attack() combatvocab.Attack {
	return combatvocab.Attack{Type: s.AttackType, Damage: s.DamageType, Targeting: s.Targeting}
}

// IsHarm is the harm-versus-help question, in one place. It replaced every
// `Type == HarmSingle || Type == HarmArea || Type == HarmMulti`.
func (s *SpellData) IsHarm() bool {
	return s.DamageType.IsHarm()
}

// HelpOrHarmString is the word the `spells` listing and the help template
// print. Derived, and pinned by TestDisplayStringsMatchTheLegacyTypes to
// what SpellType.HelpOrHarmString printed: a non-harm cast that resolves no
// target (self) is "Neutral", any other non-harm cast is "Helpful".
func (s *SpellData) HelpOrHarmString() string {
	switch {
	case s.IsHarm():
		return `Harmful`
	case s.DamageType == combatvocab.DamageNonHarm && s.Targeting == combatvocab.TargetSelf:
		return `Neutral`
	case s.DamageType == combatvocab.DamageNonHarm:
		return `Helpful`
	}
	return `Unknown`
}

// TargetTypeString is the targeting word the listing (short) and the help
// template (long) print, pinned to SpellType.TargetTypeString's output.
func (s *SpellData) TargetTypeString(short ...bool) string {
	isShort := len(short) > 0 && short[0]
	switch s.Targeting {
	case combatvocab.TargetSelf:
		return `Self`
	case combatvocab.TargetSingle:
		if isShort {
			return `Single`
		}
		return `Single Target`
	case combatvocab.TargetMulti:
		if isShort {
			return `Group`
		}
		return `Group Target`
	case combatvocab.TargetArea:
		if isShort {
			return `Area`
		}
		return `Area Target`
	}
	return `Unknown`
}

// DefenceNames is the help template's "Resisted by" value, derived from the
// eligibility table so it can never disagree with what actually rolls. Empty
// for a non-harm cast, which suppresses the line.
func (s *SpellData) DefenceNames() string {
	set, ok := combatvocab.EligibleDefences(s.Attack())
	if !ok || len(set) == 0 {
		return ""
	}
	names := make([]string, 0, len(set))
	for _, d := range set {
		names = append(names, string(d))
	}
	return strings.Join(names, " or ")
}

// validateAxes is called from Validate. The three keys are required, the
// pair must be in the table (which also binds none to non_harm), and the
// targeting must be one of the four.
func (s *SpellData) validateAxes() error {
	if s.AttackType == "" || s.DamageType == "" || s.Targeting == "" {
		return fmt.Errorf("spell %q: attack_type, damage_type and targeting are all required", s.SpellId)
	}
	if _, err := combatvocab.ParseAttackType(string(s.AttackType)); err != nil {
		return fmt.Errorf("spell %q: %w", s.SpellId, err)
	}
	if _, err := combatvocab.ParseDamageType(string(s.DamageType)); err != nil {
		return fmt.Errorf("spell %q: %w", s.SpellId, err)
	}
	if _, err := combatvocab.ParseTargeting(string(s.Targeting)); err != nil {
		return fmt.Errorf("spell %q: %w", s.SpellId, err)
	}
	if !s.Attack().Valid() {
		return fmt.Errorf("spell %q: attack_type %s with damage_type %s is not a pairing the eligibility table knows (none pairs only with non_harm)", s.SpellId, s.AttackType, s.DamageType)
	}
	return nil
}
```

Do NOT wire `validateAxes` into `Validate()` yet: the shipped YAML gains the keys in Step 7 and the boot must stay green between commits. Task 7 wires it.

- [ ] **Step 5: Run the tests**

```bash
go test ./internal/spells/ -run 'DisplayStrings|DefenceNames|IsHarmAndAttack|ValidateAxes' -v 2>&1 | grep -E "^(--- |ok|FAIL)"
```
Expected: 4 PASS.

- [ ] **Step 6: Write `tools/spell_axes_rewrite.py`**

```python
#!/usr/bin/env python3
"""Rewrite spell YAML from `type:` + `target_defense_type:` to the three axes
(messaging M4b-2), and rewrite Go TEST fixtures that build SpellData with the
retiring fields.

Text-line edits only. NEVER yaml.load/yaml.dump: a round trip destroys
quoting and comment headers. Writes to <file>.tmp then os.replace, so a
crash cannot truncate a shipped file.

Modes:
  --add        insert attack_type/damage_type/targeting after the `type:` line,
               derived from the two legacy keys; keeps both legacy keys.
  --strip      delete the `type:` and `target_defense_type:` lines.
  --check      non-zero if any spell file lacks one of the three keys or still
               carries a legacy key.
  --go-tests   rewrite Go test files: `Type: spells.X,` struct fields (and a
               following `TargetDefenseType: "...",` line) become the three
               axis fields; bare `spells.X` arguments become constructor calls.
  --dry-run    with any of the above: print what would change, write nothing.

The mapping is the spec's table (docs/superpowers/specs/2026-09-18-messaging-m4b2-axes-design.md,
section 6). core-drain is the one special case (owner ruling: physical).
"""
import argparse
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SPELL_DIRS = [
    os.path.join(ROOT, "_datafiles", "world", "dogmud", "spells"),
    os.path.join(ROOT, "_datafiles", "world", "default", "spells"),
]
LEGACY_KEYS = ("type", "target_defense_type")
NEW_KEYS = ("attack_type", "damage_type", "targeting")

TARGETING = {
    "harmsingle": "single", "helpsingle": "single",
    "harmmulti": "multi", "helpmulti": "multi",
    "harmarea": "area", "helparea": "area",
    "neutral": "self",
}
HARM = {"harmsingle", "harmmulti", "harmarea"}
SPECIAL = {"core-drain": ("spell", "physical", "area")}


def derive(spell_id, legacy_type, legacy_def):
    """Return (attack_type, damage_type, targeting) for one spell."""
    if spell_id in SPECIAL:
        return SPECIAL[spell_id]
    if legacy_type not in TARGETING:
        raise ValueError(f"{spell_id}: unknown type {legacy_type!r}")
    targeting = TARGETING[legacy_type]
    if legacy_type in HARM:
        damage = legacy_def if legacy_def in ("physical", "mental", "social") else "mental"
        return ("spell", damage, targeting)
    return ("none", "non_harm", targeting)


KEY_RE = re.compile(r"^(?P<key>[a-z_]+):\s*(?P<val>[^#\n]*?)\s*(#.*)?$")


def read_key(lines, key):
    for line in lines:
        m = KEY_RE.match(line)
        if m and m.group("key") == key:
            return m.group("val").strip().strip('"').strip("'")
    return None


def write_swap(path, lines):
    tmp = path + ".tmp"
    with open(tmp, "w", encoding="utf-8", newline="") as f:
        f.writelines(lines)
    os.replace(tmp, path)


def spell_files():
    for d in SPELL_DIRS:
        for name in sorted(os.listdir(d)):
            if name.endswith(".yaml"):
                yield os.path.join(d, name)


def do_add(dry):
    changed = 0
    for path in spell_files():
        with open(path, encoding="utf-8", newline="") as f:
            lines = f.readlines()
        spell_id = read_key(lines, "spellid") or os.path.basename(path)[:-5]
        if read_key(lines, "attack_type") is not None:
            continue
        legacy_type = read_key(lines, "type")
        legacy_def = read_key(lines, "target_defense_type")
        attack, damage, targeting = derive(spell_id, legacy_type, legacy_def)
        out = []
        inserted = False
        for line in lines:
            out.append(line)
            m = KEY_RE.match(line)
            if m and m.group("key") == "type" and not inserted:
                nl = "\r\n" if line.endswith("\r\n") else "\n"
                out.append(f"attack_type: {attack}{nl}")
                out.append(f"damage_type: {damage}{nl}")
                out.append(f"targeting: {targeting}{nl}")
                inserted = True
        if not inserted:
            raise ValueError(f"{path}: no type: line to anchor on")
        changed += 1
        print(f"{'would add' if dry else 'add'} {spell_id}: {attack}/{damage}/{targeting}")
        if not dry:
            write_swap(path, out)
    print(f"{changed} files")


def do_strip(dry):
    changed = 0
    for path in spell_files():
        with open(path, encoding="utf-8", newline="") as f:
            lines = f.readlines()
        out = [l for l in lines if not (KEY_RE.match(l) and KEY_RE.match(l).group("key") in LEGACY_KEYS)]
        if len(out) != len(lines):
            changed += 1
            print(f"{'would strip' if dry else 'strip'} {os.path.basename(path)}: {len(lines) - len(out)} lines")
            if not dry:
                write_swap(path, out)
    print(f"{changed} files")


def do_check():
    bad = 0
    seen = 0
    for path in spell_files():
        seen += 1
        with open(path, encoding="utf-8", newline="") as f:
            lines = f.readlines()
        for k in NEW_KEYS:
            if read_key(lines, k) is None:
                print(f"MISSING {k}: {path}")
                bad += 1
        for k in LEGACY_KEYS:
            if read_key(lines, k) is not None:
                print(f"LEGACY {k}: {path}")
                bad += 1
    print(f"checked {seen} files, {bad} problems")
    if seen == 0:
        print("checked nothing: the spell directories are wrong")
        return 2
    return 1 if bad else 0


# Go test fixtures.
CTOR = {
    "Neutral": "combatvocab.NonHarm(combatvocab.TargetSelf)",
    "HelpSingle": "combatvocab.NonHarm(combatvocab.TargetSingle)",
    "HelpMulti": "combatvocab.NonHarm(combatvocab.TargetMulti)",
    "HelpArea": "combatvocab.NonHarm(combatvocab.TargetArea)",
    "HarmSingle": "combatvocab.Spell(combatvocab.Damage{D}, combatvocab.TargetSingle)",
    "HarmMulti": "combatvocab.Spell(combatvocab.Damage{D}, combatvocab.TargetMulti)",
    "HarmArea": "combatvocab.Spell(combatvocab.Damage{D}, combatvocab.TargetArea)",
}
FIELDS = {
    "Neutral": ("AttackNone", "DamageNonHarm", "TargetSelf"),
    "HelpSingle": ("AttackNone", "DamageNonHarm", "TargetSingle"),
    "HelpMulti": ("AttackNone", "DamageNonHarm", "TargetMulti"),
    "HelpArea": ("AttackNone", "DamageNonHarm", "TargetArea"),
    "HarmSingle": ("AttackSpell", "Damage{D}", "TargetSingle"),
    "HarmMulti": ("AttackSpell", "Damage{D}", "TargetMulti"),
    "HarmArea": ("AttackSpell", "Damage{D}", "TargetArea"),
}
TYPE_FIELD_RE = re.compile(r"^(?P<indent>\s*)Type:(?P<sp>\s*)spells\.(?P<t>\w+),(?P<rest>.*)$")
INLINE_TYPE_RE = re.compile(r"\bType:\s*spells\.(?P<t>\w+),")
DEF_FIELD_RE = re.compile(r'^\s*TargetDefenseType:\s*"(?P<d>\w*)",\s*$')
INLINE_DEF_RE = re.compile(r'\s*TargetDefenseType:\s*"(?P<d>\w*)",')
BARE_RE = re.compile(r"\bspells\.(Neutral|HelpSingle|HelpMulti|HelpArea|HarmSingle|HarmMulti|HarmArea)\b")


def damage_word(d):
    return {"physical": "Physical", "mental": "Mental", "social": "Social"}.get(d, "Mental")


def fields_for(t, d):
    a, dm, tg = FIELDS[t]
    dm = dm.replace("{D}", damage_word(d))
    return f"AttackType: combatvocab.{a}, DamageType: combatvocab.{dm}, Targeting: combatvocab.{tg},"


def rewrite_go(path, dry):
    with open(path, encoding="utf-8", newline="") as f:
        lines = f.readlines()
    out = []
    i = 0
    changed = False
    while i < len(lines):
        line = lines[i]
        m = TYPE_FIELD_RE.match(line)
        if m:
            # Look ahead (same literal, within 12 lines) for a TargetDefenseType line.
            d = ""
            for j in range(i + 1, min(i + 13, len(lines))):
                dm = DEF_FIELD_RE.match(lines[j])
                if dm:
                    d = dm.group("d")
                    del lines[j]
                    break
                if lines[j].strip() in ("}", "})", "},"):
                    break
            out.append(f"{m.group('indent')}{fields_for(m.group('t'), d)}{m.group('rest')}\n" if line.endswith("\n") else f"{m.group('indent')}{fields_for(m.group('t'), d)}{m.group('rest')}")
            changed = True
            i += 1
            continue
        im = INLINE_TYPE_RE.search(line)
        if im:
            d = ""
            dm = INLINE_DEF_RE.search(line)
            if dm:
                d = dm.group("d")
                line = line[:dm.start()] + line[dm.end():]
            line = INLINE_TYPE_RE.sub(lambda mm: fields_for(mm.group("t"), d), line, count=1)
            changed = True
        if BARE_RE.search(line):
            line = BARE_RE.sub(lambda mm: CTOR[mm.group(1)].replace("{D}", "Mental"), line)
            changed = True
        out.append(line)
        i += 1
    if changed:
        print(f"{'would rewrite' if dry else 'rewrite'} {os.path.relpath(path, ROOT)}")
        if not dry:
            write_swap(path, out)
    return changed


def do_go_tests(dry):
    n = 0
    for base in ("internal", "modules"):
        for dirpath, _, names in os.walk(os.path.join(ROOT, base)):
            for name in names:
                if name.endswith("_test.go"):
                    n += rewrite_go(os.path.join(dirpath, name), dry)
    print(f"{n} test files")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--add", action="store_true")
    ap.add_argument("--strip", action="store_true")
    ap.add_argument("--check", action="store_true")
    ap.add_argument("--go-tests", action="store_true")
    ap.add_argument("--dry-run", action="store_true")
    args = ap.parse_args()
    if args.add:
        do_add(args.dry_run)
    elif args.strip:
        do_strip(args.dry_run)
    elif args.go_tests:
        do_go_tests(args.dry_run)
    elif args.check:
        sys.exit(do_check())
    else:
        ap.print_help()
        sys.exit(2)


if __name__ == "__main__":
    main()
```

- [ ] **Step 7: Dry-run, then add the keys to both worlds**

```bash
python tools/spell_axes_rewrite.py --add --dry-run | tail -70
```
Expected: 67 lines of `would add <id>: ...` (59 dogmud + 8 default) and `67 files`. Spot-check by eye: `charm: spell/social/single`, `core-drain: spell/physical/area`, `conjure-fire: none/non_harm/self`, `heal: none/non_harm/single`, `summon-hive-swarm: none/non_harm/single`, `sparks: spell/physical/area` (dogmud) and default's `sparks: spell/mental/multi`, `mind-spike: spell/mental/single`, `cleansing-wave: none/non_harm/area`.

```bash
python tools/spell_axes_rewrite.py --add
git diff --stat -- _datafiles/world/dogmud/spells _datafiles/world/default/spells | tail -1   # 67 files changed, 201 insertions
grep -rh -E "^(attack_type|damage_type|targeting):" _datafiles/world/dogmud/spells | sort | uniq -c
```
Expected counts for dogmud: `attack_type: none` 37, `attack_type: spell` 22, `damage_type: non_harm` 37, `damage_type: physical` 12 (11 plus core-drain), `damage_type: mental` 9, `damage_type: social` 1, `targeting: single` 36, `targeting: area` 11, `targeting: self` 12. (`multi` 0.)

- [ ] **Step 8: Prove the loader still boots and the goldens are untouched**

```bash
go test ./internal/spells/ ./internal/narration/ 2>&1 | tail -3
for f in internal/narration/testdata/stores/*.golden; do cmp -s "$f" "C:/tmp/m4b2-goldens/$(basename "$f")" || echo "DIFF $f"; done
```
Expected: `ok`, no DIFF (the spells golden renders narration only).

- [ ] **Step 9: Commit**

```bash
git add internal/spells/spells.go internal/spells/axes.go internal/spells/axes_test.go tools/spell_axes_rewrite.py
git add _datafiles/world/dogmud/spells _datafiles/world/default/spells
git commit -m "feat(spells): the three axes, beside the legacy keys

SpellData gains attack_type, damage_type and targeting with derived display
strings pinned to what SpellType printed. Every shipped spell in both worlds
carries the new keys next to the old two; nothing reads them yet.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

## Task 7: Spells consumers flip; the legacy vocabulary dies

**Files:**
- Modify: `internal/spells/spells.go` (delete `SpellType`, the seven constants, both display methods, `Type`, `TargetDefenseType`; wire `validateAxes`)
- Create: `internal/spells/shipped_axes_test.go`
- Modify: `internal/actions/cast.go:111-308`, `cast_admission.go:39-50`
- Modify: `internal/hooks/spell_resolution.go` (11 lines + `spellAttackShape` deleted), `combat_shared_helpers.go:87-96,655`, `NewRound_DoCombat_helpers.go:631,638`
- Modify: `internal/mobcommands/cast.go:179`, `internal/usercommands/skill.cast.go:246-248`, `internal/usercommands/spells.go:42-61,81-83`
- Modify: `internal/templates/templatesfunctions.go:93-113`, `internal/templates/defensename_test.go`
- Modify: both worlds' `templates/help/spell.template:7,8,12`
- Modify: 33 test files (tool), `internal/actions/cast_test.go:32`
- Modify: `_datafiles/world/*/spells/*.yaml` (strip)

- [ ] **Step 1: Write the failing shipped-data guard** (`internal/spells/shipped_axes_test.go`)

```go
package spells

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

// Both shipped worlds, every spell file: the three keys present, no legacy
// key. yaml.v3 ignores unknown keys, so a leftover `type:` would load
// silently; this is the only thing that catches it.
func TestShippedSpellsCarryTheAxesAndNoLegacyKeys(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	root := filepath.Join(filepath.Dir(here), "..", "..")
	legacy := regexp.MustCompile(`(?m)^(type|target_defense_type):`)
	required := []*regexp.Regexp{
		regexp.MustCompile(`(?m)^attack_type:\s*\S`),
		regexp.MustCompile(`(?m)^damage_type:\s*\S`),
		regexp.MustCompile(`(?m)^targeting:\s*\S`),
	}
	seen := 0
	for _, world := range []string{"dogmud", "default"} {
		dir := filepath.Join(root, "_datafiles", "world", world, "spells")
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			if filepath.Ext(e.Name()) != ".yaml" {
				continue
			}
			seen++
			raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if loc := legacy.FindIndex(raw); loc != nil {
				t.Errorf("%s/%s still carries a legacy key: %q", world, e.Name(), raw[loc[0]:loc[1]])
			}
			for _, re := range required {
				if !re.Match(raw) {
					t.Errorf("%s/%s lacks %s", world, e.Name(), re.String())
				}
			}
		}
	}
	if seen < 60 {
		t.Fatalf("scanned only %d spell files; the guard is not looking at the shipped worlds", seen)
	}
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/spells/ -run TestShippedSpellsCarryTheAxesAndNoLegacyKeys 2>&1 | grep -c "still carries a legacy key"
```
Expected: 101 (67 `type:` lines plus 34 `target_defense_type:` lines).

- [ ] **Step 3: Delete the legacy vocabulary in `spells.go`**

Delete `type SpellType string` (line 19), the `Type` and `TargetDefenseType` fields, the seven `SpellType` constants (lines 82-88, leave the school constants), and both `func (s SpellType) ...` methods (lines 103-150). In `Validate()`, after `validatePrimaryStat`:

```go
	if err := s.validateAxes(); err != nil {
		return err
	}
```

- [ ] **Step 4: Rewrite `actions/cast.go`**

The target-resolution switch at line 111 becomes a switch on the pair. Every arm's BODY is unchanged; only the arm heads change:

```go
	switch {

	case spellInfo.IsHarm() && spellInfo.Targeting == combatvocab.TargetSingle:
		// (body of the old HarmSingle arm, verbatim)

	case spellInfo.IsHarm() && spellInfo.Targeting == combatvocab.TargetMulti:
		// (body of the old HarmMulti arm, verbatim)

	case !spellInfo.IsHarm() && spellInfo.Targeting == combatvocab.TargetSingle:
		// (body of the old HelpSingle arm, verbatim)

	case !spellInfo.IsHarm() && spellInfo.Targeting == combatvocab.TargetMulti:
		// (body of the old HelpMulti arm, verbatim)

	case spellInfo.IsHarm() && spellInfo.Targeting == combatvocab.TargetArea:
		// (body of the old HarmArea arm, verbatim)

	case !spellInfo.IsHarm() && spellInfo.Targeting == combatvocab.TargetArea:
		// (body of the old HelpArea arm, verbatim)

	case spellInfo.Targeting == combatvocab.TargetSelf:
		spellRest = targetName
	}
```
Line 306: `if spellInfo.IsHarm() {`. The comment at line 172-180 mentioning `target_defense_type` and `ChannelSocial` is reworded to "charm declares damage_type social".

`cast_admission.go:39-50`:

```go
	switch {
	case spellInfo.Targeting == combatvocab.TargetSingle:
	case spellInfo.IsHarm() && spellInfo.Targeting == combatvocab.TargetMulti:
	default:
		return aim, false
	}
	...
	if !spellInfo.IsHarm() && spellInfo.Targeting == combatvocab.TargetSingle && castsAtSelf(actor, targetName) {
```
(The old set was harmsingle, harmmulti, helpsingle: single of either kind, plus harm multi.)

- [ ] **Step 5: Rewrite hooks**

`spell_resolution.go`:
- `playerHarmTargetPermitted(spellData *spells.SpellData, mob *mobs.Mob) bool`: `if spellData.IsHarm() { return !mobs.CheckPlayerHarm(mob).Blocked() }; return true`. Its two callers pass `spellData`.
- `:95` `if spellData.IsHarm() && spellData.Targeting == combatvocab.TargetArea {`
- `:109` `if !spellData.IsHarm() && spellData.Targeting == combatvocab.TargetArea {`
- `:165` `if targetUser.Character.Health < 1 && spellData.IsHarm() {`
- `:168` `if spellData.AttackType == combatvocab.AttackNone {` (comment: "Non-harm cast: uncontested, an attack win by construction")
- `:760, :795, :953, :1746, :1757` every three-way `Type ==` disjunction becomes `spellData.IsHarm()`.
- `:1329` `if spellData.IsHarm() && spellData.Targeting == combatvocab.TargetArea {`
- Delete `spellAttackShape` (Task 5's transitional builder) and its comment; every `spellAttackShape(spellData)` becomes `spellData.Attack()`. The `spellData == nil` guard it carried is dead: every caller dereferences `spellData` first.

`combat_shared_helpers.go:87-96`:

```go
		// Mitigation is keyed by the DAMAGE type (combat.MitigationChannelFor):
		// physical harm meets physical mitigation, mental meets magical,
		// social meets conviction. A non-harm cast never reaches this helper.
		var mitigPct, cap float64
		if ch, ok := combat.MitigationChannelFor(spellData.DamageType); ok {
			switch ch {
			case combat.ChannelPhysical:
				mitigPct = target.GetPhysicalMitigation()
			case combat.ChannelMagical:
				mitigPct = target.GetMagicalMitigation()
			case combat.ChannelConviction:
				mitigPct = target.GetConvictionMitigation()
			}
			cap = combat.MitigationCap(ch)
		} else {
			mudlog.Error("calcSpellDamageForCharacter", "spell", spellData.SpellId, "error", "non-harm spell reached the damage pipeline")
			cap = 0.75
		}
```
(`GetConvictionMitigation` exists: `actions/combat_taunt.go:250` calls it.)

`combat_shared_helpers.go:655`: `(spellData.Type == ...)` becomes `spellData.IsHarm()`.

`NewRound_DoCombat_helpers.go:631`: `if !spellData.IsHarm() && spellData.Targeting == combatvocab.TargetSingle &&`; `:638`: `if spellData.IsHarm() && (spellData.Targeting == combatvocab.TargetArea || spellData.Targeting == combatvocab.TargetMulti) &&`.

- [ ] **Step 6: Rewrite the commands and the listing**

`mobcommands/cast.go:178-179`: `if spellInfo.IsHarm() {` in place of the switch.

`usercommands/skill.cast.go:246-248`: `if !spellInfo.IsHarm() && spellInfo.Targeting == combatvocab.TargetSingle {` ... `} else if spellInfo.IsHarm() && spellInfo.Targeting == combatvocab.TargetArea {`.

`usercommands/spells.go`:
- `:42` `if !sp.IsHarm() && sp.Targeting == combatvocab.TargetSelf {` (was Neutral); `:46` `if sp.IsHarm() {`.
- `targetRank` (`:53-63`):
```go
	switch sp.Targeting {
	case combatvocab.TargetSelf:
		return 0
	case combatvocab.TargetSingle:
		return 1
	case combatvocab.TargetMulti:
		return 2
	case combatvocab.TargetArea:
		return 3
	}
	return 0
```
- `:81,83` `sp.HelpOrHarmString()` and `sp.TargetTypeString(true)`.

- [ ] **Step 7: The template and its helper**

Delete the `"defensename"` entry from the func map (`templatesfunctions.go:93-113`, comment included). In BOTH `_datafiles/world/dogmud/templates/help/spell.template` and `_datafiles/world/default/templates/help/spell.template`:

```
<ansi fg="yellow">Type:        </ansi> <ansi fg="spell-{{ lowercase .HelpOrHarmString }}">{{ .HelpOrHarmString }}</ansi>
<ansi fg="yellow">Target:      </ansi> <ansi fg="white-bold">{{ .TargetTypeString }}</ansi>
...
{{ with .DefenceNames -}}
<ansi fg="yellow">Resisted by: </ansi> <ansi fg="white-bold">{{ . }}</ansi>
{{ end -}}
```

Rewrite `internal/templates/defensename_test.go`: delete `TestDefenseName`; `TestSpellTemplateDefenceRow` renders the fragment `{{ with .DefenceNames -}}Resisted by: {{ . }}\n{{ end -}}` against real `*spells.SpellData` values (import `spells` and `combatvocab`): social `Resisted by: defy\n`, mental `quell`, physical `dodge or block`, and a `NonHarm` spell renders `""`. `TestSpellTemplatesParseWithTheRealFuncMap` is unchanged and now proves the real templates parse without `defensename`.

- [ ] **Step 8: The test-fixture sweep**

```bash
python tools/spell_axes_rewrite.py --go-tests --dry-run | tail -5    # expect "33 test files" or close
python tools/spell_axes_rewrite.py --go-tests
```
Then `internal/actions/cast_test.go:32` by hand:

```go
func seedTestSpell(spellId string, shape combatvocab.Attack, baseFolds int) (*spells.SpellData, func()) {
	sd := &spells.SpellData{
		SpellId:    spellId,
		Name:       "Test " + spellId,
		AttackType: shape.Type,
		DamageType: shape.Damage,
		Targeting:  shape.Targeting,
		BaseFolds:  baseFolds,
		Cost:       5,
	}
```
(the tool already rewrote its callers' `spells.HelpSingle` arguments to constructor calls.)

```bash
goimports -w $(grep -rl --include=*_test.go "combatvocab\." internal)
go vet ./... 2>&1 | grep -v "^#" | head -40
```
Fix by hand whatever the tool could not see: a `Type:` field split across lines, a comparison `sd.Type == spells.X` (becomes `sd.Targeting == ...`/`sd.IsHarm()`), a `TargetDefenseType` set in a separate statement (`sd.TargetDefenseType = "physical"` becomes `sd.DamageType = combatvocab.DamagePhysical`). `hooks/hooks_test.go:629,651` set `TargetDefenseType: "mental"` in literals WITHOUT a `Type:` line; the tool leaves those, and they become `DamageType: combatvocab.DamageMental` with `AttackType: combatvocab.AttackSpell, Targeting: combatvocab.TargetSingle` added.

- [ ] **Step 9: Strip the legacy keys**

```bash
python tools/spell_axes_rewrite.py --strip --dry-run | tail -3   # 67 files
python tools/spell_axes_rewrite.py --strip
python tools/spell_axes_rewrite.py --check                       # "checked 67 files, 0 problems", exit 0
```

Prove `--check` can fail: re-add one legacy line and one missing key by hand, run `--check`, expect exit 1 with two lines, then `git checkout -- <that file>` is NOT safe (it stages nothing here since the file is unstaged, but the tripwire says never use the pathspec form; use `git restore --worktree -- _datafiles/world/dogmud/spells/<file>`).

- [ ] **Step 10: Build, vet, test, goldens, and the mitigation-arm trace**

```bash
go build ./... && go vet ./... && go test ./... 2>&1 | grep -v "^ok" | head -20
for f in internal/narration/testdata/stores/*.golden; do cmp -s "$f" "C:/tmp/m4b2-goldens/$(basename "$f")" || echo "DIFF $f"; done
```
Expected: no FAIL, no DIFF.

The spec's one open trace: can any shipped spell reach the mitigation helper with `non_harm`? The six production callers of `calcSpellDamageForCharacter` are the `damage`, `dot` and `knockdown` effect arms. Prove every such spell has a real damage type:

```bash
for f in _datafiles/world/dogmud/spells/*.yaml; do e=$(grep -m1 '^effect_type:' "$f" | awk '{print $2}'); d=$(grep -m1 '^damage_type:' "$f" | awk '{print $2}'); case "$e" in damage|dot|knockdown) echo "$(basename $f .yaml) $e $d";; esac; done | grep -c non_harm
```
Expected: `0` (run standalone; `grep -c` exits 1 on zero matches). Record the result in the commit message.

- [ ] **Step 11: Commit**

```bash
git add -u internal _datafiles/world/dogmud/spells _datafiles/world/default/spells _datafiles/world/dogmud/templates/help/spell.template _datafiles/world/default/templates/help/spell.template
git add internal/spells/shipped_axes_test.go
git commit -m "refactor(spells): the axes replace type and target_defense_type

SpellType, SpellData.Type and TargetDefenseType are deleted; every consumer
reads IsHarm(), Targeting and Attack(). Validation requires the three keys
and a pairing the eligibility table knows. The help template and the spells
listing print exactly what they printed, from derived strings. Mitigation
is keyed by damage type; no shipped damage, dot or knockdown spell is
non_harm (traced, 0). Nothing player-visible changes.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

## Task 8: Guards that fail the build

**Files:**
- Create: `internal/combatvocab/one_declaration_guard_test.go`

- [ ] **Step 1: Write the guard**

```go
package combatvocab

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// A defence name declared as a Go string literal outside this package is
// the drift this package exists to end: three declarations of "dodge" is how
// M4b-2 started. actionspec's cost-action keys share the spelling and are a
// different namespace, so that one file is allowed.
var defenceLiteral = regexp.MustCompile(`=\s*"(dodge|parry|block|quell|defy)"`)

var allowedFiles = map[string]bool{
	"internal/combatvocab/vocab.go":   true,
	"internal/actionspec/action.go":   true,
}

func repoRoot(t *testing.T) string {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	return filepath.Join(filepath.Dir(here), "..", "..")
}

func scanForDefenceLiterals(t *testing.T, root string) (hits []string, scanned int) {
	for _, base := range []string{"internal", "modules"} {
		err := filepath.WalkDir(filepath.Join(root, base), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel := filepath.ToSlash(strings.TrimPrefix(path, root+string(os.PathSeparator)))
			if allowedFiles[rel] {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			scanned++
			if loc := defenceLiteral.FindIndex(raw); loc != nil {
				hits = append(hits, rel+": "+string(raw[loc[0]:loc[1]]))
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return hits, scanned
}

func TestDefenceNamesAreDeclaredOnce(t *testing.T) {
	hits, scanned := scanForDefenceLiterals(t, repoRoot(t))
	if scanned < 500 {
		t.Fatalf("scanned only %d files; the walk is not looking at the repo", scanned)
	}
	for _, h := range hits {
		t.Errorf("defence name declared outside combatvocab: %s", h)
	}
}

// The guard must be able to fail. A copy of a real declaration line, planted
// in a temp tree with the same shape, must be found.
func TestDefenceNamesGuardIsNotVacuous(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "somewhere")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte("package somewhere\n\nconst DefenseDodge string = \"dodge\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	hits, _ := scanForDefenceLiterals(t, root)
	if len(hits) != 1 {
		t.Fatalf("planted declaration not found: hits = %v", hits)
	}
}
```

- [ ] **Step 2: Run it, expecting the vacuity test to pass and the real scan to pass**

```bash
go test ./internal/combatvocab/ -run 'DeclaredOnce|NotVacuous' -v 2>&1 | grep -E "^(--- |ok|FAIL)"
```
If `TestDefenceNamesAreDeclaredOnce` reports a hit, it is a real leftover; fix the site (it should be none after Task 4: `combat_helpers.go:1278-1286`'s `defenseVerb = "dodge"` literals were collapsed to `string(best.defenseType)`, and `deflectedSwingLines`'s `verbYou, verbThey = "dodge", "dodges"` assigns TWO strings, which the regex does not match; `surprise_narration.go:59-63` likewise).

- [ ] **Step 3: Prove it capable of failing against the real tree**

Add `const sabotage = "dodge"` to `internal/combat/pools.go`, run the test, expect one `defence name declared outside combatvocab: internal/combat/pools.go` failure, then remove the line and rerun green.

- [ ] **Step 4: Commit**

```bash
git add internal/combatvocab/one_declaration_guard_test.go
git commit -m "test(combatvocab): fail the build on a second declaration of a defence name

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

## Task 9: Behaviour commit A: core-drain drops parry (ruling 7)

**Files:**
- Modify: `internal/actions/combat_drain.go:286-302`
- Create: `internal/actions/drain_area_shape_test.go`

- [ ] **Step 1: Write the failing test**

```go
package actions

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// core-drain (spell) resolves through ExecuteDrainArea, which used to
// hardcode the melee set so a construct's room-wide drain could be PARRIED.
// Owner ruling (M4 spec 9, carried to M4b-2): it is a physical spell, so its
// victims dodge or block. Pinned at the source: the loop's Shape must be
// Spell(DamagePhysical, TargetArea), and no Melee(...) may remain in
// ExecuteDrainArea.
func TestExecuteDrainAreaIsAPhysicalAreaSpell(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(filepath.Dir(here), "combat_drain.go"), nil, 0)
	require.NoError(t, err)

	var fn *ast.FuncDecl
	for _, d := range parsed.Decls {
		if f, ok := d.(*ast.FuncDecl); ok && f.Name.Name == "ExecuteDrainArea" {
			fn = f
		}
	}
	require.NotNil(t, fn, "ExecuteDrainArea not found")

	spellPhysicalArea, melee := 0, 0
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "combatvocab" {
			return true
		}
		switch sel.Sel.Name {
		case "Melee":
			melee++
		case "Spell":
			if len(call.Args) == 2 && isSel(call.Args[0], "DamagePhysical") && isSel(call.Args[1], "TargetArea") {
				spellPhysicalArea++
			}
		}
		return true
	})
	require.Zero(t, melee, "ExecuteDrainArea still builds a melee shape; core-drain must not be parryable")
	require.GreaterOrEqual(t, spellPhysicalArea, 2, "the skill move AND the counter exit must both carry Spell(DamagePhysical, TargetArea)")
}

func isSel(e ast.Expr, name string) bool {
	s, ok := e.(*ast.SelectorExpr)
	return ok && s.Sel.Name == name
}
```

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/actions/ -run TestExecuteDrainAreaIsAPhysicalAreaSpell 2>&1 | grep -E "still builds|FAIL|ok"
```
Expected: FAIL, "still builds a melee shape".

- [ ] **Step 3: Change the three sites in `ExecuteDrainArea`**

Lines 289, 293 and 302: `combatvocab.Melee(combatvocab.TargetArea)` becomes `combatvocab.Spell(combatvocab.DamagePhysical, combatvocab.TargetArea)`. Note that `SituationalAttackMult` now returns 1.0 for the drain (spells take no prone or stamina penalty), which is part of the same ruling: the drain is a cast. Add above the loop:

```go
		// M4b-2 (owner ruling, M4 spec 9): core-drain is a PHYSICAL SPELL. Its
		// victims dodge or block; nobody parries a room. This also drops the
		// melee prone/stamina accuracy penalty, because a cast pays neither.
		// A balance change, in its own commit.
```

- [ ] **Step 4: Run the test and the actions and hooks suites**

```bash
go test ./internal/actions/ ./internal/hooks/ 2>&1 | tail -3
```
Expected: `ok`. If `internal/hooks/spell_drainarea_test.go` pinned parry in the drain's defence set, update its expectation and say so in the commit.

- [ ] **Step 5: Commit**

```bash
git add internal/actions/combat_drain.go internal/actions/drain_area_shape_test.go
git add -u internal/hooks
git commit -m "balance(combat): core-drain is a physical area spell, not a melee sweep

BEHAVIOUR CHANGE (owner ruling, M4 spec 9). ExecuteDrainArea resolves as
Spell(DamagePhysical, TargetArea): victims dodge or block and no longer
parry, and the caster pays no melee accuracy penalty. Lands on the Core
Guardian's drain fights.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

## Task 10: Behaviour commit B: non-harm casts at mobs are uncontested (finding 2)

**Files:**
- Modify: `internal/hooks/spell_resolution.go:136-150` (mob-target loop), `:1546-1550` (`resolveMobSpellAgainstMob`)
- Create: `internal/hooks/nonharm_mob_shortcut_test.go`

- [ ] **Step 1: Write the failing call-site test**

The M4a lesson: a golden covers the store, not the path production calls. This test drives `resolveAgainstMob` itself through the swappable seam and counts contests.

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/contest"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// A heal cast at your own companion used to run a quell contest: the
// companion could "defend" the heal, a fumble backfired on you, and a
// defensive crit handed the companion a counter-swing at its owner
// (spell_resolution.go:406 on master 612b85d54 ran the seam unconditionally).
// Non-harm casts take the uncontested path now, for mob targets as they
// always did for player targets.
func TestNonHarmCastAtAMobRunsNoContest(t *testing.T) {
	contests := 0
	restore := combat.SetChannelAttackContestRunnerForTest(func(atk float64, entries []contest.Entry) contest.Result {
		contests++
		return contest.Result{}
	})
	t.Cleanup(restore)

	const roomId = 8830
	room, cleanupRoom := seedHookRoom(t, roomId)          // MobDeath_FactionRep_test.go:51
	defer cleanupRoom()
	user := users.NewTestUser(1, "caster", "Cala", 1001)  // as channel_defence_routing_test.go:162
	user.Character.RoomId = roomId
	restoreUsers := users.SeedUsersForTest(map[int]*users.UserRecord{1: user})
	defer restoreUsers()
	room.AddPlayer(user.UserId)
	mob := charmTestMob(t, 8831, roomId)                  // charm_effect_test.go:25
	mobs.SetInstanceForTest(mob.InstanceId, mob)
	defer mobs.SetInstanceForTest(mob.InstanceId, nil)
	room.AddMob(mob.InstanceId)
	heal := &spells.SpellData{
		SpellId: "test-heal", Name: "Test Heal", PrimaryStat: "willpower",
		AttackType: combatvocab.AttackNone, DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle,
		EffectType: "heal", EffectMagnitude: 10,
	}
	before := mob.Character.Health
	mob.Character.Health = before / 2

	fumbled, landed := resolveAgainstMob(user, mob, room, heal, spellAttackSideFor(heal, user.Character), heal.EffectMagnitude)

	if contests != 0 {
		t.Fatalf("a non-harm cast ran %d contest(s) against a mob", contests)
	}
	if fumbled || !landed {
		t.Errorf("fumbled=%v landed=%v; an uncontested cast lands and cannot fumble", fumbled, landed)
	}
	if mob.Character.Health <= before/2 {
		t.Error("the heal did not apply")
	}
}

// The harm path must still contest, or the test above proves nothing.
func TestHarmCastAtAMobStillRunsOneContest(t *testing.T) {
	contests := 0
	restore := combat.SetChannelAttackContestRunnerForTest(func(atk float64, entries []contest.Entry) contest.Result {
		contests++
		return contest.Result{Success: true}
	})
	t.Cleanup(restore)

	const roomId = 8832
	room, cleanupRoom := seedHookRoom(t, roomId)
	defer cleanupRoom()
	user := users.NewTestUser(1, "caster", "Cala", 1001)
	user.Character.RoomId = roomId
	restoreUsers := users.SeedUsersForTest(map[int]*users.UserRecord{1: user})
	defer restoreUsers()
	room.AddPlayer(user.UserId)
	mob := charmTestMob(t, 8833, roomId)
	mobs.SetInstanceForTest(mob.InstanceId, mob)
	defer mobs.SetInstanceForTest(mob.InstanceId, nil)
	room.AddMob(mob.InstanceId)
	bolt := &spells.SpellData{
		SpellId: "test-bolt", Name: "Test Bolt", PrimaryStat: "willpower",
		AttackType: combatvocab.AttackSpell, DamageType: combatvocab.DamageMental, Targeting: combatvocab.TargetSingle,
		EffectType: "damage", DamageMultiplier: 1,
	}
	resolveAgainstMob(user, mob, room, bolt, spellAttackSideFor(bolt, user.Character), 0)
	if contests != 1 {
		t.Fatalf("a harm cast ran %d contest(s), want exactly 1", contests)
	}
}
```
The three helpers are the package's own: `seedHookRoom` (`MobDeath_FactionRep_test.go:51`), `charmTestMob` (`charm_effect_test.go:25`) and `users.NewTestUser` with `users.SeedUsersForTest` (`channel_defence_routing_test.go:162-166`). The companion case does not need the mob charmed: `resolveAgainstMob` is reached after target resolution, and the shortcut keys on the spell, not the mob. `room.AddMob` exists if `room.AddPlayer` does; check `internal/rooms` and drop the call if the resolver never consults room membership.

- [ ] **Step 2: Run to verify the first test fails and the second passes**

```bash
go test ./internal/hooks/ -run 'NonHarmCastAtAMob|HarmCastAtAMobStill' -v 2>&1 | grep -E "^(--- |ok|FAIL)"
```
Expected: `--- FAIL: TestNonHarmCastAtAMobRunsNoContest` ("ran 1 contest(s)"), `--- PASS: TestHarmCastAtAMobStillRunsOneContest`.

- [ ] **Step 3: Add the shortcut at the top of `resolveAgainstMob`** (before the charm block at line 393)

```go
	// Non-harm cast at a mob (a heal on your companion, an area mend over
	// allies): uncontested, exactly as the player-target loop has always
	// treated it. Until M4b-2 this ran a quell contest, so a companion could
	// "defend" its own heal, a fumble backfired on the caster, and a
	// defensive crit earned the companion a counter-swing at its owner.
	// BEHAVIOUR CHANGE, own commit.
	if spellData.AttackType == combatvocab.AttackNone {
		applyMobEffect(user, user.Character, mob, room, spellData, magnitude, combat.ChannelDefenceResult{DamageMultiplier: 1})
		combat.RecordSpell(combat.User, combat.Mob, true, false, false, false, 0, 0, user.Character, &mob.Character, util.GetRoundCount())
		return false, true
	}
```
Check `applyMobEffect`'s signature at its declaration before pasting; match it exactly.

And in `resolveMobSpellAgainstMob` (`:1546`), widen the heal-only shortcut:

```go
	if spellData.AttackType == combatvocab.AttackNone {
```
with the comment updated to say every non-harm cast, not only `heal`, is cooperative.

- [ ] **Step 4: Run the tests**

```bash
go test ./internal/hooks/ 2>&1 | tail -3
```
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/spell_resolution.go internal/hooks/nonharm_mob_shortcut_test.go
git commit -m "fix(spells): a non-harm cast at a mob is uncontested

BEHAVIOUR CHANGE. Healing or buffing a companion (or an ally mob under an
area mend) no longer runs a quell contest: it cannot be defended, cannot
backfire, and cannot earn the target a counter-swing at its owner. The
mob-to-mob path's heal-only shortcut widens to every non-harm cast. The M4
spec asked for this trace; the path was reachable.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

## Task 11: Docs

**Files:**
- Modify: `internal/combat/context.md:225-244,392-404,426,490-499,1629,1953`, `internal/items/context.md:412-422`, `internal/characters/context.md:687-715`, `internal/spells/context.md:44-56,105-119`, `internal/actions/context.md` (grep hits), `internal/conditions/context.md:1117-1127`, `internal/hooks/context.md:1233`
- Modify: `docs/schemas/spell.md:35,43,169-179` and every example block with `type:`
- Modify: `.claude/skills/dogmud-combat/SKILL.md:180-182`
- Modify: `docs/README.md`, `docs/PATCH_NOTES.md`

- [ ] **Step 1: `context.md` files**

For each file, replace the named region so it describes the code as it now is. Verify every symbol named exists (`grep -n "func \|type " internal/combatvocab/*.go`). Content to carry:

- `combat/context.md`: the defence-set section becomes "Defence sets come from `combatvocab.EligibleDefences`, keyed by the attack-and-damage pair" with the eight-row table; `ResolveChannelAttack(shape combatvocab.Attack, ...)`; the derived pools (`pools.go`) with the fact that spells scale magically and mitigate by damage type; the counter pool keyed by attack type until the counters slice; the files table rows for `defence_sets.go` (now the equipment gate only) and the new `pools.go`.
- `items/context.md`: `DefencePool`, `DefencePoolFor`, the four `CounterPool*` constants.
- `characters/context.md`: "there are FIVE defences, declared once in `combatvocab`"; the signatures now take `combatvocab.Defence`.
- `spells/context.md`: the struct listing gains the three fields and loses two; the "Spell Types" block is replaced by the axes table and the sentence about `self` versus `single`; `Attack()`, `IsHarm()`, `HelpOrHarmString()`, `TargetTypeString()`, `DefenceNames()`.
- `actions/context.md`: every `ChannelMelee`/`ChannelRanged`/`ChannelSocial` mention becomes the constructor call; note `Shape` on `SkillMoveParams`.
- `conditions/context.md:1117-1127`: the two shield spells are `attack_type: none`, and the branch is `AttackType == combatvocab.AttackNone`.
- `hooks/context.md:1233`: "the `(spell, social)` pairing" for `ChannelSocial`.

Run the audit: `python tools/context_md_audit.py` and expect no phantom symbol in the eight files.

- [ ] **Step 2: `docs/schemas/spell.md`**

Replace the `type` row with three rows:

```
| `attack_type` | string | **yes** | `spell` for a harmful cast, `none` for anything that harms nobody. |
| `damage_type` | string | **yes** | `physical`, `mental`, `social`, or `non_harm`. `non_harm` requires `attack_type: none`. Decides which defence answers the cast (physical: dodge or block; mental: quell; social: defy) and which mitigation the target uses. |
| `targeting` | string | **yes** | `self` (no target is resolved; summons, identify), `single` (one target, the caster by default for a non-harm cast), `multi`, `area`. |
```
Delete the `target_defense_type` row. Replace the "Valid SpellType Values" table with an "Axes" table showing the seven old values mapped to the new triples, headed "Migrated 2026-09-18; `tools/spell_axes_rewrite.py --check` refuses the old keys". Update every example block (`type: helpsingle` at lines 86, 252; `type: neutral` at 102, 113, 130) to the three keys.

- [ ] **Step 3: `.claude/skills/dogmud-combat/SKILL.md:180-182`**

The sentence quoting `spellAttackChannel` becomes: "`SpellData.Attack()` (`internal/spells/axes.go`) builds the attack from the three authored axes and the seam resolves it against `combatvocab.EligibleDefences`; the defender's score comes from `GetDefenseScoreFor` inside the seam."

- [ ] **Step 4: `docs/README.md` and `docs/PATCH_NOTES.md`**

README: add rows for this plan and for `tools/spell_axes_rewrite.py` (the tools table already lists `messaging_token_rewrite.py`; match its row shape). Mark the notes row as superseded.

PATCH_NOTES, at the top, two entries:

```markdown
## 2026-09-18: Mending your companion no longer picks a fight

Healing or shielding your own companion used to be resolved as if you were
attacking it. The companion could shrug the mending off, a badly cast one
could rebound on you, and a lucky companion could even answer your kindness
with a swing. A cast that harms nobody is now simply received. Nothing else
about spells has changed.

## 2026-09-18: The Core Guardian's drain is a working, not a swing

The Core Guardian's room-wide drain was being treated as a sweep of its
arms, so a raised blade could turn it aside. It is a working of its cold
fire and is now answered the way a bolt of force is: you get out of its way
or you take it on a shield. Nothing else about the fight has changed.
```

- [ ] **Step 5: Commit**

```bash
git add internal/combat/context.md internal/items/context.md internal/characters/context.md internal/spells/context.md internal/actions/context.md internal/conditions/context.md internal/hooks/context.md docs/schemas/spell.md .claude/skills/dogmud-combat/SKILL.md docs/README.md docs/PATCH_NOTES.md
git commit -m "docs: the four axes in every context.md, the spell schema and the patch notes

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

## Task 12: Final verification and the PR

- [ ] **Step 1: The sweep grep, proven capable first**

```bash
git grep -n -E "AttackChannel|combat\.DefenseType|characters\.Defense(None|Dodge|Parry|Block|Quell|Defy)|TargetDefenseType|spells\.SpellType|spellAttackChannel|channelDamageChannel|DefenceSetFor" master -- 'internal/*.go' | wc -l   # a positive number: the pattern can match
git grep -n -E "AttackChannel|combat\.DefenseType|characters\.Defense(None|Dodge|Parry|Block|Quell|Defy)|TargetDefenseType|spells\.SpellType|spellAttackChannel|channelDamageChannel|DefenceSetFor" HEAD -- 'internal/*.go' 'modules/*.go' ':!*_test.go'
```
Expected: the first prints a count well above zero; the second prints nothing. Run each standalone, not in an `&&` chain.

- [ ] **Step 2: Full suite, lint, goldens, tool check**

```bash
go build ./... && go vet ./... && go test ./... 2>&1 | grep -v "^ok" | head
golangci-lint run --new-from-rev=master 2>&1 | tail -3
for f in internal/narration/testdata/stores/*.golden; do cmp -s "$f" "C:/tmp/m4b2-goldens/$(basename "$f")" || echo "DIFF $f"; done
python tools/spell_axes_rewrite.py --check
```
Expected: no FAIL, `0 issues`, no DIFF, `0 problems`.

- [ ] **Step 3: Boot check** (from `dogmud-shipping`)

```bash
git worktree add --detach C:/tmp/dogmud-boot-check HEAD
cp _datafiles/config.yaml C:/tmp/dogmud-boot-check/_datafiles/config.yaml
cd C:/tmp/dogmud-boot-check && go build -o boot-check.exe .
timeout 180 ./boot-check.exe > boot.log 2>&1
grep -cE "^panic:|goroutine [0-9]+ \[running\]|runtime error" boot.log   # want 0 (standalone: exits 1 on zero)
grep -c "Server Ready" boot.log                                          # want 1
grep -c "attack pair is not in the eligibility table" boot.log           # want 0
cd "C:/Users/Calabe Davis/workspace/DOGMud" && git worktree remove --force C:/tmp/dogmud-boot-check
```
Exit code 124 from `timeout` is the success case. Judge by the three greps, never by `$?` of the binary.

- [ ] **Step 4: Push and open the PR**

```bash
git push -u origin feature/messaging-m4b2-axes
gh pr create --repo pruuk/DOGMud --base master --head feature/messaging-m4b2-axes --title "Messaging M4b-2: the four axes" --body-file - <<'EOF'
Spec: docs/superpowers/specs/2026-09-18-messaging-m4b2-axes-design.md
Plan: docs/superpowers/plans/2026-09-18-messaging-m4b2-axes.md

One declaration each of AttackType, DamageType, Targeting and Defence in the new leaf package internal/combatvocab. combatvocab.Attack replaces AttackChannel through the seam; eligibility is one table keyed by the attack-and-damage pair; the damage pool is derived from the axes. Spell type and target_defense_type retire for attack_type, damage_type and targeting, rewritten in both worlds by tools/spell_axes_rewrite.py.

Every golden is byte-identical to master. The spells listing and help template print exactly what they printed.

TWO BEHAVIOUR COMMITS, last on the branch:
- core-drain resolves as a physical area spell: dodge or block, no parry (owner ruling, M4 spec 9).
- a non-harm cast at a mob is uncontested: healing your companion can no longer be defended, backfire, or earn the companion a counter-swing (found while verifying the spec; the path was reachable).

Guards added, each proven capable of failing: eligibility parity, pool parity, display parity, shipped-data axes, one declaration of the defence names, and a call-site test on the companion-heal path.

CI lint will go red if the diff exceeds 300 files; local `golangci-lint run --new-from-rev=master` is the check that counts.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
```

---

## Self-review against the spec

- **Section 1, leaf package:** Task 1. Constructors, table, `Pairs`, `Valid`, zero-value invalid. `multi` kept.
- **Section 2, derived pools:** Task 2 defines, Task 5 wires (`awardChannelDefenceBonus`, `SituationalAttackMult`), Task 7 wires the mitigation switch. `ReflectChannel` and analytics untouched.
- **Section 3, seam and callers:** Task 5, all 17 sites named, throw passes `Thrown(TargetArea)`, counter pool keyed by type.
- **Section 4, one Defence:** Task 4, every listed parameter and field.
- **Section 5, store keeps pool keys:** Task 3.
- **Sections 6 and 7, spells data and code:** Tasks 6 and 7, including the template, the listing, `playerHarmTargetPermitted`, both shortcuts (Task 10 for the mob ones).
- **Section 8, guards:** 1 and 2 in Tasks 1 and 2; 3 in Task 6; 4 in Task 7; 5 in Task 8; 6 in Task 10; 7 in Task 12.
- **Section 9, docs:** Task 11.
- **Behaviour commits:** Tasks 9 and 10, last before docs, flagged in the PR body.
- **The open trace** (mitigation default arm): Task 7 Step 10.
- **Field-name collision** the spec did not foresee (`SkillMoveParams.Attack` already exists): resolved as `Shape` in Task 5 and stated in the plan header.
