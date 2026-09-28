package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func healSpellForParityTest() *spells.SpellData {
	return &spells.SpellData{
		SpellId: "test-mend", Name: "Mend", AttackType: combatvocab.AttackNone,
		DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle,
		EffectType: "heal", EffectMagnitude: 3, BaseFolds: 4, PrimaryStat: "willpower",
		Schools: []string{spells.SchoolVital},
	}
}

// parityHealRounds is the heal's duration on the fixture's equalised caster
// (base folds 4, skill 3, willpower 100): half the universal duration,
// floored at six. 4 x (10 + 5 + 1.5) = 66, so 33.
func parityHealRounds() int {
	rounds := calcSpellDuration(4, 3, 100) / 2
	if rounds < 6 {
		rounds = 6
	}
	return rounds
}

// regenRecord returns c's one Regenerating record, or fails the test.
func regenRecord(t *testing.T, c *characters.Character) *conditions.Condition {
	t.Helper()
	recs := c.GetConditions(conditions.ConditionIdRegenerating)
	require.Len(t, recs, 1, "the heal must leave one Regenerating record")
	return recs[0]
}

// Slice 3b (audit row 3): a creature's heal on a player was contested, fell
// to the default arm and applied nothing.
func TestSpellHeal_ACreatureHealsAPlayer(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := healSpellForParityTest()

	resolveMobSpellAgainstPlayer(f.casterMob, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), spell.EffectMagnitude)

	rec := regenRecord(t, f.targetUser.Character)
	assert.Equal(t, 3.0, rec.Magnitude)
	assert.Equal(t, parityHealRounds(), rec.TriggersLeft)
	assert.Equal(t, 1, countContaining(drainPlain(2), "Mend envelops you in healing energy."))
	assert.Equal(t, 1, countContaining(drainPlain(3), "Mend envelops Bobrick in healing light."))
}

// Owner ruling 3: help spells do not crit. The player-to-player heal doubled
// the part of its multiplier above 1x on a crit no cast could reach.
func TestSpellHeal_ACritChangesNothing(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	caster := actions.NewUserActorInRoom(f.casterUser, f.room)
	target := actions.NewUserActorInRoom(f.targetUser, f.room)

	applySpellEffect(newSpellEffectCtx(f.casterUser.Character, caster, target, f.room,
		healSpellForParityTest(), 3, spellContestAttackCrit()))

	assert.Equal(t, 3.0, regenRecord(t, f.targetUser.Character).Magnitude, "a crit must not boost a heal")
}

// A player healing a mob queues events.Healed for the AI companion; nothing
// else does, because the event names a player healer and a mob.
func TestSpellHeal_OnlyAPlayerHealingAMobQueuesHealed(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := healSpellForParityTest()
	events.DrainQueuedHealedForTest(0)

	resolveAgainstMob(f.casterUser, f.targetMob, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), spell.EffectMagnitude)
	healed := events.DrainQueuedHealedForTest(0)
	require.Len(t, healed, 1)
	assert.Equal(t, events.Healed{HealerUserId: 1, MobInstanceId: 101}, healed[0])

	resolveMobSpellAgainstMob(f.casterMob, f.targetMob, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), spell.EffectMagnitude)
	resolveAgainstPlayer(f.casterUser, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), spell.EffectMagnitude)
	assert.Empty(t, events.DrainQueuedHealedForTest(0), "only a player healing a mob is tended")
}
