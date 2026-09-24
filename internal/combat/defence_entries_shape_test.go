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

// An unknown pair resolves uncontested, exactly as the old flattened
// channel's default arm did, but it is no longer silent.
func TestDefenceEntriesForUnknownPairIsEmptyNotPanic(t *testing.T) {
	bare := characters.New()
	bad := combatvocab.Attack{Type: combatvocab.AttackMelee, Damage: combatvocab.DamageMental, Targeting: combatvocab.TargetSingle}
	if got := DefenceEntriesFor(bad, bare, DefenceEntryOpts{}); len(got) != 0 {
		t.Errorf("unknown pair produced defences %v", got)
	}
}
