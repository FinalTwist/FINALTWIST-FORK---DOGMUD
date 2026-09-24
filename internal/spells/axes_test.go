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
		legacy                  string
		attack                  combatvocab.AttackType
		damage                  combatvocab.DamageType
		targeting               combatvocab.Targeting
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
		{SpellId: "harm-self", PrimaryStat: "willpower", AttackType: combatvocab.AttackSpell, DamageType: combatvocab.DamagePhysical, Targeting: combatvocab.TargetSelf},
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
