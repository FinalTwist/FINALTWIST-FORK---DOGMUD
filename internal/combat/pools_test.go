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
