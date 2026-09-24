package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Finding 3 / chunk 5.2 — the resolution-time half.
//
// Spells fold over several rounds, so the target set chosen at cast time is
// stale by the time the spell lands. A mob can become a companion, or a
// builder can flag it non_combatant, between initiation and resolution. The
// authorization policy therefore has to be re-checked when the spell lands,
// not only when it is aimed.
// ---------------------------------------------------------------------------

// spellFor builds the minimal *spells.SpellData playerHarmTargetPermitted
// needs from an attack shape.
func spellFor(shape combatvocab.Attack) *spells.SpellData {
	return &spells.SpellData{AttackType: shape.Type, DamageType: shape.Damage, Targeting: shape.Targeting}
}

func TestPlayerHarmTargetPermitted_HarmfulSpellsRespectPolicy(t *testing.T) {
	harmful := []combatvocab.Attack{combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetSingle), combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetMulti), combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetArea)}

	for _, shape := range harmful {
		st := spellFor(shape)
		ordinary := &mobs.Mob{Character: characters.Character{Name: "Bandit"}}
		assert.True(t, playerHarmTargetPermitted(st, ordinary),
			"%s must still land on an ordinary mob", shape)

		immune := &mobs.Mob{
			PlayerAttackImmune: true,
			Character:          characters.Character{Name: "Caravan Guard"},
		}
		assert.False(t, playerHarmTargetPermitted(st, immune),
			"%s must not land on an attack-immune mob", shape)

		nonCombatant := &mobs.Mob{
			NonCombatant: true,
			Character:    characters.Character{Name: "Barkeep"},
		}
		assert.False(t, playerHarmTargetPermitted(st, nonCombatant),
			"%s must not land on a non-combatant", shape)

		companion := &mobs.Mob{Character: characters.Character{Name: "Wolf"}}
		companion.Character.Charm(7, 100, "")
		assert.False(t, playerHarmTargetPermitted(st, companion),
			"%s must not land on a companion", shape)
	}
}

// Help spells legitimately target companions, so the harm policy must not be
// applied to them.
func TestPlayerHarmTargetPermitted_HelpSpellsAreUnaffected(t *testing.T) {
	companion := &mobs.Mob{Character: characters.Character{Name: "Wolf"}}
	companion.Character.Charm(7, 100, "")

	for _, shape := range []combatvocab.Attack{combatvocab.NonHarm(combatvocab.TargetSingle), combatvocab.NonHarm(combatvocab.TargetMulti), combatvocab.NonHarm(combatvocab.TargetArea), combatvocab.NonHarm(combatvocab.TargetSelf)} {
		assert.True(t, playerHarmTargetPermitted(spellFor(shape), companion),
			"%s must still reach a companion", shape)
	}
}

// A mob that gained protection while the spell was folding must be dropped at
// resolution even though it passed the check at cast time.
func TestPlayerHarmTargetPermitted_ProtectionGainedMidCast(t *testing.T) {
	m := &mobs.Mob{Character: characters.Character{Name: "Stray Dog"}}
	st := spellFor(combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetSingle))

	assert.True(t, playerHarmTargetPermitted(st, m),
		"target was legal when the cast started")

	// The player charms it with a second spell before the first one lands.
	m.Character.Charm(7, 100, "")

	assert.False(t, playerHarmTargetPermitted(st, m),
		"target became a companion mid-cast and must be spared")
}
