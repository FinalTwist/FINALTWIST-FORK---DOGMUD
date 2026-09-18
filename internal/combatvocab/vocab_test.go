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
