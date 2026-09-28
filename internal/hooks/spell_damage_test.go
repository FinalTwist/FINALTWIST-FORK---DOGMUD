package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Slice 3a (audit row 7): a mob's damage spell at another mob used to land in
// silence and start no fight, because every line and both aggro commits were
// gated on a player caster.
func TestSpellDamage_MobOnMobIsSeenAndStartsAFight(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := physicalHarmSpellForCollapseTest()

	resolveMobSpellAgainstMob(f.casterMob, f.targetMob, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), spell.EffectMagnitude)

	assert.Less(t, f.targetMob.Character.Health, 1000, "the spell must land")
	assert.Equal(t, state.ActorRef{MobInstanceId: 100}, f.targetMob.Character.CurrentCombatTarget(),
		"the target must turn on its caster")
	assert.Equal(t, 1, countContaining(drainPlain(3), "Stone Lash strikes Ghoul!"),
		"a watcher must see a mob's spell land on another mob")
}

// A defensive crit negates the damage but the spell was still an attack. The
// mob-on-player arm used to break out before its aggro commit.
func TestSpellDamage_MobOnPlayerDefensiveCritStillStartsAFight(t *testing.T) {
	pinCounterTierKnobs(t, 0) // 0 is the documented off switch: no counter-swing muddies the read
	f := newSpellParityFixture(t, combat.ChannelDefenceResult{
		Defended: true, DefensiveCrit: true, Defence: combatvocab.DefenceQuell})
	spell := physicalHarmSpellForCollapseTest()

	resolveMobSpellAgainstPlayer(f.casterMob, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), spell.EffectMagnitude)

	assert.Equal(t, 1000, f.targetUser.Character.Health, "a defensive crit negates the damage")
	assert.Equal(t, state.ActorRef{MobInstanceId: 100}, f.targetUser.Character.CurrentCombatTarget(),
		"the target must still turn on its caster")
}

// Owner ruling 2 (2026-09-28): a player's harmful spell on a mob is an
// assault, recorded through actions.SeedAggression exactly as throw records
// one. The first cast is fresh aggression and records the crime; a second
// cast in the same fight is not fresh, so it records no new crime, but it
// still counts as aggression for the revenge and opinion seeders.
func TestSpellHarmOnAFactionMobIsACrime(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	// Registered BEFORE the Setenv calls so it runs after they restore the
	// environment: the reload then finds no definitions and leaves an empty
	// registry for the rest of the package.
	t.Cleanup(func() {
		_ = factions.LoadAllDefinitions()
		factions.ClearCache()
		crimes.ClearCache()
		opinions.ClearCache()
	})
	definitions := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", definitions)
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", t.TempDir())
	require.NoError(t, os.WriteFile(filepath.Join(definitions, "thornwall_citizens.yaml"), []byte(`faction_id: thornwall_citizens
display_name: "Citizens"
description: "test faction"
default_rep: 0
allies: []
enemies: []
`), 0644))
	require.NoError(t, factions.LoadAllDefinitions())
	factions.ClearCache()
	crimes.ClearCache()
	opinions.ClearCache()
	f.targetMob.Groups = []string{"thornwall_citizens"}

	spell := physicalHarmSpellForCollapseTest()
	cast := func() {
		resolveAgainstMob(f.casterUser, f.targetMob, f.room, spell,
			spellAttackSideFor(spell, f.casterUser.Character, nil), spell.EffectMagnitude)
	}

	cast()
	recorded := crimes.AllForFaction("thornwall_citizens", false)
	require.Len(t, recorded, 1, "the first harmful cast is an assault")
	assert.Equal(t, crimes.KindAssault, recorded[0].Kind)
	// If only this line fails, the victim could not see the caster: check
	// messaging.CanSeeClearly for a speciesless fixture mob before touching
	// production code. The crime row itself is the ruling under test.
	assert.Equal(t, crimes.PerpPlayer, recorded[0].Perpetrator.Type)
	assert.Len(t, events.DrainQueuedPlayerAttackedMobsForTest(1), 1)

	cast()
	assert.Len(t, crimes.AllForFaction("thornwall_citizens", false), 1,
		"a second cast in the same fight is not a new assault")
	assert.Len(t, events.DrainQueuedPlayerAttackedMobsForTest(1), 1,
		"every harmful cast still counts as aggression")
}
