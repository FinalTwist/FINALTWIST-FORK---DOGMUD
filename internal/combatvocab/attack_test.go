package combatvocab

import (
	"reflect"
	"testing"
)

// The eligibility table, pinned as literals. The first five rows ARE the old
// combat channel-to-defence table (defence_sets.go:51-65 on master
// 612b85d54, since deleted); the thrown and spell/social rows are the spec's
// two additions, and the last row is the uncontested pair. If this test and
// attack.go disagree, the code is wrong, not the test.
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
