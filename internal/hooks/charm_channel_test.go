package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// Charm declares target_defense_type: social so the cast it ALREADY makes is
// answered by defy instead of quell. An absent value must keep meaning
// combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetSingle) -- it
// is the default for every unclassified spell, not an escape from routing,
// and reading it as one is what produced two rejected plans for this slice.
func TestSpellAttackShape_Routing(t *testing.T) {
	cases := []struct {
		name string
		tdt  string
		want combatvocab.Attack
	}{
		{"social routes to social", "social", combatvocab.Spell(combatvocab.DamageSocial, combatvocab.TargetSingle)},
		{"physical routes to physical", "physical", combatvocab.Spell(combatvocab.DamagePhysical, combatvocab.TargetSingle)},
		{"absent stays mental", "", combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetSingle)},
		{"unknown stays mental", "nonsense", combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetSingle)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := spellAttackShape(&spells.SpellData{TargetDefenseType: c.tdt}); got != c.want {
				t.Errorf("spellAttackShape(%q) = %v, want %v", c.tdt, got, c.want)
			}
		})
	}

	if got := spellAttackShape(nil); got != combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetSingle) {
		t.Errorf("spellAttackShape(nil) = %v, want combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetSingle)", got)
	}
}
