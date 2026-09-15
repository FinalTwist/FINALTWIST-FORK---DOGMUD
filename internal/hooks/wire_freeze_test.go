package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWireFreeze_EffectTypeConditionStillApplies is a slice 2 (conditions
// unification) freeze test. wire_freeze_test.go at the repo root pins that
// `effect_type: buff` still PARSES to the Go string "buff" — but nothing
// reads that string except four `case "buff":` literals in
// spell_resolution.go (~906, 1093, 1493, 1740) and internal/usercommands/
// spells.go:32. A later slice 2 task doing a text replace across the
// codebase could turn one of those case literals into `case "condition":`
// without the compiler noticing (a string literal, not an identifier) and
// without the parse-only freeze test going red. Each of these four subtests
// drives a REAL `effect_type: buff` spell through one of the four dispatch
// shapes and asserts the condition was actually queued for the target, so a
// broken case literal shows up here even though it can't show up in a
// yaml.Unmarshal-only test.
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
// existing tests that already drive applyMobEffect and
// resolveMobSpellAgainstPlayer this way. Condition id 100 ("Test Strength Buff")
// comes from seedAllRegistries and carries no TickPool, so the tick-snapshot
// branch inside each case is not exercised here — only that the dispatch
// queued the right condition for the right holder.
func TestWireFreeze_EffectTypeConditionStillApplies(t *testing.T) {

	// spell_resolution.go:906 — applyMobEffect's top-level switch, a
	// player's spell landing on a mob, dispatching to applyMobEffect_condition.
	t.Run("PlayerCastsOnMob", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		events.DrainQueuedConditionsForTest(0)

		u := users.GetByUserId(1)
		mob := mobs.GetInstance(100)
		room := rooms.LoadRoom(1)

		spell := &spells.SpellData{
			SpellId:      "test-wire-freeze-buff-mob",
			Name:         "Test Ward",
			Type:         spells.HelpSingle,
			EffectType:   "buff",
			ConditionIds: []int{100},
		}

		applyMobEffect(u, u.Character, mob, room, spell, 0, spellContestAttackWin())

		queued := events.DrainQueuedConditionsForTest(0)
		require.Len(t, queued, 1,
			"a player's effect_type: buff spell landing on a mob must queue exactly one buff (spell_resolution.go's applyMobEffect case \"buff\")")
		assert.Equal(t, 100, queued[0].ConditionId)
		assert.Equal(t, mob.InstanceId, queued[0].MobInstanceId)
	})

	// spell_resolution.go:1093 — applyPlayerEffect's switch (reached via
	// resolveAgainstPlayer), the player self/single condition path: caster and
	// target are the same UserRecord.
	t.Run("PlayerSelfCast", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		events.DrainQueuedConditionsForTest(0)
		original := runSpellChannelAttack
		runSpellChannelAttack = func(combat.AttackChannel, combat.AttackSide, *characters.Character, *characters.Character) combat.ChannelDefenceResult {
			return spellContestAttackWin()
		}
		t.Cleanup(func() { runSpellChannelAttack = original })

		u := users.GetByUserId(1)
		room := rooms.LoadRoom(1)

		spell := &spells.SpellData{
			SpellId:      "test-wire-freeze-buff-self",
			Name:         "Test Fortify",
			Type:         spells.HelpSingle,
			EffectType:   "buff",
			ConditionIds: []int{100},
		}

		resolveAgainstPlayer(u, u, room, spell, combat.AttackSide{}, 0)

		queued := events.DrainQueuedConditionsForTest(0)
		require.Len(t, queued, 1,
			"a player self-casting an effect_type: buff spell must queue exactly one buff (spell_resolution.go's applyPlayerEffect case \"buff\")")
		assert.Equal(t, 100, queued[0].ConditionId)
		assert.Equal(t, u.UserId, queued[0].UserId)
	})

	// spell_resolution.go:1493 — applyMobSelfEffect's switch, a mob's
	// special-move boosting itself.
	t.Run("MobSelfCast", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		events.DrainQueuedConditionsForTest(0)

		mob := mobs.GetInstance(100)
		room := rooms.LoadRoom(1)

		spell := &spells.SpellData{
			SpellId:      "test-wire-freeze-buff-mobself",
			Name:         "Test Rally Cry",
			EffectType:   "buff",
			ConditionIds: []int{100},
		}

		applyMobSelfEffect(mob, room, spell, 0)

		queued := events.DrainQueuedConditionsForTest(0)
		require.Len(t, queued, 1,
			"a mob self-casting an effect_type: buff spell must queue exactly one buff (spell_resolution.go's applyMobSelfEffect case \"buff\")")
		assert.Equal(t, 100, queued[0].ConditionId)
		assert.Equal(t, mob.InstanceId, queued[0].MobInstanceId)
	})

	// spell_resolution.go:1740 — resolveMobSpellAgainstPlayer's switch, a
	// mob's spell landing on a player.
	t.Run("MobCastsOnPlayer", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		events.DrainQueuedConditionsForTest(0)
		original := runSpellChannelAttack
		runSpellChannelAttack = func(combat.AttackChannel, combat.AttackSide, *characters.Character, *characters.Character) combat.ChannelDefenceResult {
			return spellContestAttackWin()
		}
		t.Cleanup(func() { runSpellChannelAttack = original })

		caster := mobs.GetInstance(100)
		target := users.GetByUserId(2)
		room := rooms.LoadRoom(1)

		spell := &spells.SpellData{
			SpellId:      "test-wire-freeze-buff-mobcast",
			Name:         "Test Hex Ward",
			Type:         spells.HelpSingle,
			EffectType:   "buff",
			ConditionIds: []int{100},
		}

		resolveMobSpellAgainstPlayer(caster, target, room, spell, combat.AttackSide{}, 0)

		queued := events.DrainQueuedConditionsForTest(0)
		require.Len(t, queued, 1,
			"a mob's effect_type: buff spell landing on a player must queue exactly one buff (spell_resolution.go's resolveMobSpellAgainstPlayer case \"buff\")")
		assert.Equal(t, 100, queued[0].ConditionId)
		assert.Equal(t, target.UserId, queued[0].UserId)
	})
}
