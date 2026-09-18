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
