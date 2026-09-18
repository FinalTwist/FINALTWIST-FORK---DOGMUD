package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// The (spell, social) pairing was unreachable in counterPoolFor because
// taunt, the only Rhetoric attack, short-circuits its defy-crit into a
// counter-taunt at the call site. U10c made charm a social SPELL, and
// fireSpellCounterTier has no such carve-out, so a defy-crit against a charm
// now arrives here.
//
// Without a case it would fall through to the physical melee pool and a mob
// would answer a charm with a sword-swing narration.
func TestCounterPoolFor_SocialUsesTheDefyPool(t *testing.T) {
	if got := counterPoolFor(combatvocab.Spell(combatvocab.DamageSocial, combatvocab.TargetSingle)); got != items.CounterPoolDefy {
		t.Errorf("counterPoolFor(charm) = %v, want %v (not the physical fallthrough)",
			got, items.CounterPoolDefy)
	}
}

func TestCounterPoolFor_OtherChannelsUnchanged(t *testing.T) {
	cases := map[combatvocab.Attack]items.DefencePool{
		combatvocab.Ranged(combatvocab.TargetSingle):                            items.CounterPoolRanged,
		combatvocab.Thrown(combatvocab.TargetArea):                              items.CounterPoolRanged,
		combatvocab.Spell(combatvocab.DamagePhysical, combatvocab.TargetSingle): items.CounterPoolQuell,
		combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetSingle):   items.CounterPoolQuell,
		combatvocab.Rhetoric(combatvocab.TargetSingle):                          items.CounterPoolDefy,
		combatvocab.Melee(combatvocab.TargetSingle):                             items.CounterPoolMelee,
	}
	for shape, want := range cases {
		if got := counterPoolFor(shape); got != want {
			t.Errorf("counterPoolFor(%v) = %v, want %v", shape, got, want)
		}
	}
}
