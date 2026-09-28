package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWireFreeze_EffectTypeConditionStillApplies is a freeze test for the
// spell effect type `effect_type: condition`. Nothing reads that string
// except the `case "condition":` literal in applySpellEffect
// (spell_effects.go) and the spell-sorting switch in
// internal/usercommands/spells.go. A text replace across the codebase could
// change one of those literals without the compiler noticing (a string
// literal, not an identifier) and without a parse-only freeze test going
// red. Each of these four subtests drives a REAL `effect_type: condition`
// spell through one caster-target pairing and asserts the condition was
// actually queued for the target, so a broken case literal shows up here
// even though it can't show up in a yaml.Unmarshal-only test.
//
// Condition application queues an events.Condition and narrates on drain (see
// events.DrainQueuedConditionsForTest's doc comment) rather than landing
// synchronously, so — following the pattern that comment prescribes and that
// condition_after_death_test.go / condition_notice_test.go already use —
// these assert on the queued event, not on Character.Conditions.HasCondition.
//
// Fixture pattern and the runSpellChannelAttack override for the two
// contested paths follow TestDotProducerRecordsNegativeHarm_MobTarget and
// TestDotProducerRecordsNegativeHarm_PlayerTarget (hooks_test.go), the
// existing tests that already drive applySpellEffect and
// resolveMobSpellAgainstPlayer this way. Condition id 100 ("Test Strength Condition")
// comes from seedAllRegistries and carries no TickPool, so the tick-snapshot
// branch inside each case is not exercised here — only that the dispatch
// queued the right condition for the right holder.
func TestWireFreeze_EffectTypeConditionStillApplies(t *testing.T) {

	// PM: a player's spell landing on a mob.
	t.Run("PlayerCastsOnMob", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		events.DrainQueuedConditionsForTest(0)

		u := users.GetByUserId(1)
		mob := mobs.GetInstance(100)
		room := rooms.LoadRoom(1)

		spell := &spells.SpellData{
			SpellId:    "test-wire-freeze-condition-mob",
			Name:       "Test Ward",
			AttackType: combatvocab.AttackNone, DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle,
			EffectType:   "condition",
			ConditionIds: []int{100},
		}

		applySpellEffect(newSpellEffectCtx(u.Character, actions.NewUserActorInRoom(u, room),
			actions.NewMobActorInRoom(mob, room), room, spell, 0, spellContestAttackWin()))

		queued := events.DrainQueuedConditionsForTest(0)
		require.Len(t, queued, 1,
			"a player's effect_type: condition spell landing on a mob must queue exactly one condition (applySpellConditionEffect)")
		assert.Equal(t, 100, queued[0].ConditionId)
		assert.Equal(t, mob.InstanceId, queued[0].MobInstanceId)
	})

	// PP self: resolveAgainstPlayer with the caster as its own target.
	t.Run("PlayerSelfCast", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		events.DrainQueuedConditionsForTest(0)
		original := runSpellChannelAttack
		runSpellChannelAttack = func(messaging.RoomVisibility, combatvocab.Attack, combat.AttackSide, *characters.Character, *characters.Character) combat.ChannelDefenceResult {
			return spellContestAttackWin()
		}
		t.Cleanup(func() { runSpellChannelAttack = original })

		u := users.GetByUserId(1)
		room := rooms.LoadRoom(1)

		spell := &spells.SpellData{
			SpellId:    "test-wire-freeze-condition-self",
			Name:       "Test Fortify",
			AttackType: combatvocab.AttackNone, DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle,
			EffectType:   "condition",
			ConditionIds: []int{100},
		}

		resolveAgainstPlayer(u, u, room, spell, combat.AttackSide{}, 0)

		queued := events.DrainQueuedConditionsForTest(0)
		require.Len(t, queued, 1,
			"a player self-casting an effect_type: condition spell must queue exactly one condition (applySpellConditionEffect)")
		assert.Equal(t, 100, queued[0].ConditionId)
		assert.Equal(t, u.UserId, queued[0].UserId)
	})

	// MS: a mob's spell on itself.
	t.Run("MobSelfCast", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		events.DrainQueuedConditionsForTest(0)

		mob := mobs.GetInstance(100)
		room := rooms.LoadRoom(1)

		spell := &spells.SpellData{
			SpellId:      "test-wire-freeze-condition-mobself",
			Name:         "Test Rally Cry",
			EffectType:   "condition",
			ConditionIds: []int{100},
		}

		applySpellEffect(newSpellEffectCtx(&mob.Character, actions.NewMobActorInRoom(mob, room),
			actions.NewMobActorInRoom(mob, room), room, spell, 0, uncontestedSpellResult()))

		queued := events.DrainQueuedConditionsForTest(0)
		require.Len(t, queued, 1,
			"a mob self-casting an effect_type: condition spell must queue exactly one condition (applySpellConditionEffect)")
		assert.Equal(t, 100, queued[0].ConditionId)
		assert.Equal(t, mob.InstanceId, queued[0].MobInstanceId)
	})

	// MP: a mob's spell landing on a player.
	t.Run("MobCastsOnPlayer", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		events.DrainQueuedConditionsForTest(0)
		original := runSpellChannelAttack
		runSpellChannelAttack = func(messaging.RoomVisibility, combatvocab.Attack, combat.AttackSide, *characters.Character, *characters.Character) combat.ChannelDefenceResult {
			return spellContestAttackWin()
		}
		t.Cleanup(func() { runSpellChannelAttack = original })

		caster := mobs.GetInstance(100)
		target := users.GetByUserId(2)
		room := rooms.LoadRoom(1)

		spell := &spells.SpellData{
			SpellId:    "test-wire-freeze-condition-mobcast",
			Name:       "Test Hex Ward",
			AttackType: combatvocab.AttackNone, DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle,
			EffectType:   "condition",
			ConditionIds: []int{100},
		}

		resolveMobSpellAgainstPlayer(caster, target, room, spell, combat.AttackSide{}, 0)

		queued := events.DrainQueuedConditionsForTest(0)
		require.Len(t, queued, 1,
			"a mob's effect_type: condition spell landing on a player must queue exactly one condition (applySpellConditionEffect)")
		assert.Equal(t, 100, queued[0].ConditionId)
		assert.Equal(t, target.UserId, queued[0].UserId)
	})

	// Lighting plan 5a: the same four dispatch shapes carrying a
	// magnitude-driven light. The queued event must hold the CASTER-scaled
	// strength and duration (willpower 100, spellcasting 0: 50 for 4 triggers
	// at the shipped knobs), not the spec's authored trigger count and no
	// magnitude. The registries are released through t.Cleanup, not defer, so
	// they unwind after the glow seed that replaced their condition map.
	lightSpell := func(id string) *spells.SpellData {
		return &spells.SpellData{
			SpellId:    id,
			Name:       "Test Glow",
			AttackType: combatvocab.AttackNone, DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle,
			EffectType:   "condition",
			PrimaryStat:  "willpower",
			ConditionIds: []int{testGlowConditionId},
		}
	}
	newCaster := func(c *characters.Character) {
		c.Stats.Willpower.ValueAdj = 100
		c.SetSkill("spellcasting", 0)
	}
	seedLight := func(t *testing.T) {
		t.Cleanup(seedAllRegistries())
		configs.SetConfigForTest(t, configs.GetConfig())
		seedTestGlowCondition(t)
		events.DrainQueuedConditionsForTest(0)
	}
	winContests := func(t *testing.T) {
		original := runSpellChannelAttack
		runSpellChannelAttack = func(messaging.RoomVisibility, combatvocab.Attack, combat.AttackSide, *characters.Character, *characters.Character) combat.ChannelDefenceResult {
			return spellContestAttackWin()
		}
		t.Cleanup(func() { runSpellChannelAttack = original })
	}
	assertScaledGlow := func(t *testing.T, queued []events.Condition) {
		t.Helper()
		require.Len(t, queued, 1, "a light spell must queue exactly one condition")
		assert.Equal(t, testGlowConditionId, queued[0].ConditionId)
		assert.Equal(t, 50.0, queued[0].Magnitude, "strength 40 + 100/10 + 0/2")
		assert.Equal(t, 4, queued[0].Triggers, "duration 2 + 100/50 + 0/20")
	}

	t.Run("PlayerCastsOnMob_ScaledLight", func(t *testing.T) {
		seedLight(t)
		u := users.GetByUserId(1)
		newCaster(u.Character)
		mob := mobs.GetInstance(100)

		r := rooms.LoadRoom(1)
		applySpellEffect(newSpellEffectCtx(u.Character, actions.NewUserActorInRoom(u, r),
			actions.NewMobActorInRoom(mob, r), r, lightSpell("test-glow-mob"), 0, spellContestAttackWin()))

		queued := events.DrainQueuedConditionsForTest(0)
		assertScaledGlow(t, queued)
		assert.Equal(t, mob.InstanceId, queued[0].MobInstanceId)
	})

	t.Run("PlayerSelfCast_ScaledLight", func(t *testing.T) {
		seedLight(t)
		winContests(t)
		u := users.GetByUserId(1)
		newCaster(u.Character)

		resolveAgainstPlayer(u, u, rooms.LoadRoom(1), lightSpell("test-glow-self"), combat.AttackSide{}, 0)

		queued := events.DrainQueuedConditionsForTest(0)
		assertScaledGlow(t, queued)
		assert.Equal(t, u.UserId, queued[0].UserId)
	})

	t.Run("MobSelfCast_ScaledLight", func(t *testing.T) {
		seedLight(t)
		mob := mobs.GetInstance(100)
		newCaster(&mob.Character)

		r := rooms.LoadRoom(1)
		applySpellEffect(newSpellEffectCtx(&mob.Character, actions.NewMobActorInRoom(mob, r),
			actions.NewMobActorInRoom(mob, r), r, lightSpell("test-glow-mobself"), 0, uncontestedSpellResult()))

		queued := events.DrainQueuedConditionsForTest(0)
		assertScaledGlow(t, queued)
		assert.Equal(t, mob.InstanceId, queued[0].MobInstanceId)
	})

	t.Run("MobCastsOnPlayer_ScaledLight", func(t *testing.T) {
		seedLight(t)
		winContests(t)
		caster := mobs.GetInstance(100)
		newCaster(&caster.Character)
		target := users.GetByUserId(2)

		resolveMobSpellAgainstPlayer(caster, target, rooms.LoadRoom(1), lightSpell("test-glow-mobcast"), combat.AttackSide{}, 0)

		queued := events.DrainQueuedConditionsForTest(0)
		assertScaledGlow(t, queued)
		assert.Equal(t, target.UserId, queued[0].UserId)
	})
}
