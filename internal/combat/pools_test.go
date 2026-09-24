package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// The scale channel is what CalcRawDamage and DamageScale key on, and it is
// ALSO the toughen channel. Pinned against master 612b85d54:
//   - calcSpellDamageForCharacter always passes ChannelMagical
//     (hooks/combat_shared_helpers.go:52), physical spells included;
//   - the old flattened-channel mapping (deleted, defence_multiplier.go:769)
//     answered "physical" for melee and ranged, "magical" for BOTH spell
//     damage types, "conviction" for social.
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

// ToughenName crosses into characters.ToughenStatFor (progression.go:511),
// which characters cannot spell from this side because it cannot import
// combat. The round trip is the contract: each pool must name the stat it
// toughens, and an unknown pool must name none.
func TestToughenNameRoundTripsThroughCharactersToughenStatFor(t *testing.T) {
	cases := map[DamageChannel]string{
		ChannelPhysical:   "vitality",
		ChannelMagical:    "willpower",
		ChannelConviction: "charisma",
	}
	for ch, wantStat := range cases {
		if got := characters.ToughenStatFor(ch.ToughenName()); got != wantStat {
			t.Errorf("ToughenStatFor(%v.ToughenName()=%q) = %q, want %q", ch, ch.ToughenName(), got, wantStat)
		}
	}
	if got := characters.ToughenStatFor(DamageChannel(99).ToughenName()); got != "" {
		t.Errorf("an unknown pool must toughen nothing, got %q", got)
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

// Master's channelDamageChannel, restated through the shapes each channel
// became. ChannelSocial covered taunt AND charm, so both shapes must toughen
// conviction; the scale pool alone would send charm to magical.
func TestToughenChannelForMatchesMasterChannelDamageChannel(t *testing.T) {
	cases := []struct {
		shape combatvocab.Attack
		want  DamageChannel
	}{
		{combatvocab.Melee(combatvocab.TargetSingle), ChannelPhysical},
		{combatvocab.Ranged(combatvocab.TargetSingle), ChannelPhysical},
		{combatvocab.Thrown(combatvocab.TargetArea), ChannelPhysical},
		{combatvocab.Spell(combatvocab.DamagePhysical, combatvocab.TargetSingle), ChannelMagical},
		{combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetSingle), ChannelMagical},
		{combatvocab.Spell(combatvocab.DamageSocial, combatvocab.TargetSingle), ChannelConviction},
		{combatvocab.Rhetoric(combatvocab.TargetSingle), ChannelConviction},
	}
	for _, tc := range cases {
		if got := ToughenChannelFor(tc.shape); got != tc.want {
			t.Errorf("ToughenChannelFor(%v) = %v, want %v", tc.shape, got, tc.want)
		}
	}
}
