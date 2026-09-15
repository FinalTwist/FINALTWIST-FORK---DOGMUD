package behaviortree

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// archetypeYAML is the path to the melee_self_buff archetype YAML.
// Go tests run with cwd = package directory (internal/behaviortree/).
const archetypeYAML = "../../_datafiles/world/dogmud/behaviors/archetypes/melee_self_buff.yaml"

// Integration tests for the melee_self_buff archetype.
//
// The TestMeleeSelfCondition_* names below say Condition (slice 2 of the
// conditions unification renames every Go identifier); the archetype and its
// YAML stay melee_self_buff, because the behaviour-category string is wire
// and this slice does not touch it.
//
// All three tests use the full end-to-end pipeline:
//  1. LoadArchetypeForTest loads the real melee_self_buff.yaml
//  2. A mob with BehaviorArchetype:"melee_self_buff" is seeded
//  3. TryMobBehavior fires mob_combat_round → selector evaluates children
//  4. cast_best_in_category runs synchronously (not in delayedActions),
//     so mob.Command fires inline → "cast X" is queued in events
//  5. events.InspectQueuedInputForTest verifies the cast command
//
// Test 1: fresh mob with an offense spell → casts self_offense
// Test 2: surge condition already active, only defense spells → selector
//         falls through offense (Failure) → casts self_defense
// Test 3: defense-only mob → offense always Failure → casts self_defense

// seedArchetypeSpells installs the four real spells used by melee_self_buff.
// Returns a cleanup function.
func seedArchetypeSpells(t *testing.T) func() {
	t.Helper()
	return spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"conviction-surge": {
			SpellId: "conviction-surge", Name: "Conviction Surge",
			Type: spells.HelpSingle, Cost: 35, BaseFolds: 4,
			EffectType: "buff", ConditionIds: []int{26},
			Categories: []string{"self_offense"},
		},
		"iron-will": {
			SpellId: "iron-will", Name: "Iron Will",
			Type: spells.HelpSingle, Cost: 45, BaseFolds: 6,
			EffectType: "buff", ConditionIds: []int{27},
			Categories: []string{"self_defense"},
		},
		"conviction-ward": {
			SpellId: "conviction-ward", Name: "Conviction Ward",
			Type: spells.HelpSingle, Cost: 30, BaseFolds: 4,
			EffectType: "shield",
			Categories: []string{"self_defense"},
		},
		"conviction-armor": {
			SpellId: "conviction-armor", Name: "Conviction Armor",
			Type: spells.HelpSingle, Cost: 50, BaseFolds: 6,
			EffectType: "buff", ConditionIds: []int{38},
			Categories: []string{"self_defense"},
		},
	})
}

// seedArchetypeMob seeds a mob with BehaviorArchetype set to "melee_self_buff"
// and the provided spellbook. Returns the mob pointer and a cleanup function.
func seedArchetypeMob(t *testing.T, instanceId int, spellbook map[string]int) (*mobs.Mob, func()) {
	t.Helper()
	m := &mobs.Mob{
		MobId:             mobs.MobId(300 + instanceId),
		InstanceId:        instanceId,
		BehaviorArchetype: "melee_self_buff",
	}
	m.Character.Name = "testmob"
	m.Character.Conviction = 500
	m.Character.SpellBook = spellbook
	m.Character.Conditions = conditions.New()
	cleanup := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{300 + instanceId: m},
		map[int]*mobs.Mob{instanceId: m},
	)
	return m, cleanup
}

// seedConditionOnChar seeds a condition as active on the character, using the same
// pattern as action_cast_best_in_category_test.go: directly set Conditions.List
// then call Validate(true) to rebuild the conditionIds index. Also seeds the condition
// spec so the conditions package doesn't panic on lookup.
func seedConditionOnChar(t *testing.T, char *characters.Character, conditionId int) func() {
	t.Helper()
	cleanupCondition := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		conditionId: {ConditionId: conditionId, Name: "TestBuff"},
	})
	char.Conditions.List = append(char.Conditions.List, &conditions.Condition{
		ConditionId:  conditionId,
		TriggersLeft: 5,
	})
	char.Conditions.Validate(true)
	if !char.HasCondition(conditionId) {
		t.Fatalf("seedBuffOnChar: HasBuff(%d) false after seeding — setup broken", conditionId)
	}
	return cleanupCondition
}

// TestMeleeSelfCondition_FreshMobCastsSelfOffense is a full end-to-end pipeline
// test: real YAML loaded, TryMobBehavior fires, delayed action drained, and
// the queued "cast conviction-surge" is verified.
func TestMeleeSelfCondition_FreshMobCastsSelfOffense(t *testing.T) {
	defer seedArchetypeSpells(t)()
	LoadArchetypeForTest(t, "melee_self_buff", archetypeYAML)

	mob, cleanup := seedArchetypeMob(t, 90001, map[string]int{
		"conviction-surge": 3, // self_offense — only offense spell
		"conviction-ward":  4, // self_defense — fallback (never reached this round)
		"iron-will":        4, // self_defense — fallback (never reached this round)
	})
	defer cleanup()
	defer events.DrainQueuedInputsForTest(mob.InstanceId)

	ok := TryMobBehavior(mob.InstanceId, EventContext{EventType: "mob_combat_round"})
	if !ok {
		t.Fatalf("TryMobBehavior: expected Success (tree handled event), got false")
	}
	// Flush the perception-delay queue so mob.Command fires synchronously.
	DrainAllDelayedActionsForTest(t)

	cmd := events.InspectQueuedInputForTest(mob.InstanceId, "cast ")
	if !strings.HasPrefix(cmd, "cast conviction-surge") {
		t.Fatalf("fresh mob should cast conviction-surge (only self_offense), got %q", cmd)
	}
}

// TestMeleeSelfCondition_WithSurgeActiveCastsIronWill verifies the full selector
// fallthrough: when the offense condition (surge, condition 26) is already active, the
// offense child returns Failure and the selector falls through to defense.
// The defense action picks iron-will (score 6×45=270) over conviction-ward
// (score 4×30=120).
func TestMeleeSelfCondition_WithSurgeActiveCastsIronWill(t *testing.T) {
	defer seedArchetypeSpells(t)()
	LoadArchetypeForTest(t, "melee_self_buff", archetypeYAML)

	mob, cleanup := seedArchetypeMob(t, 90002, map[string]int{
		"conviction-ward": 4, // self_defense, score 120
		"iron-will":       4, // self_defense, score 270 — should win
	})
	defer cleanup()
	defer events.DrainQueuedInputsForTest(mob.InstanceId)

	// Mark surge condition 26 as active so the offense child finds no eligible
	// spell and returns Failure, forcing the selector to try defense.
	cleanupCondition := seedConditionOnChar(t, &mob.Character, 26)
	defer cleanupCondition()

	ok := TryMobBehavior(mob.InstanceId, EventContext{EventType: "mob_combat_round"})
	if !ok {
		t.Fatalf("TryMobBehavior: expected Success (tree handled event), got false")
	}

	cmd := events.InspectQueuedInputForTest(mob.InstanceId, "cast ")
	if !strings.HasPrefix(cmd, "cast iron-will") {
		t.Fatalf("highest-scoring defense (iron-will, score 270) should be cast, got %q", cmd)
	}
}

// TestMeleeSelfCondition_FireElementalCastsDefenseOnly verifies that a mob with
// only self_defense spells correctly falls through the offense child (Failure)
// and casts the highest-scoring defense spell (conviction-armor, 300 > ward, 120).
// This covers the fire elemental archetype case (stays on melee_self_buff).
func TestMeleeSelfCondition_FireElementalCastsDefenseOnly(t *testing.T) {
	defer seedArchetypeSpells(t)()
	LoadArchetypeForTest(t, "melee_self_buff", archetypeYAML)

	mob, cleanup := seedArchetypeMob(t, 90004, map[string]int{
		"conviction-armor": 3, // self_defense, score 6×50=300
		"conviction-ward":  3, // self_defense, score 4×30=120
	})
	defer cleanup()
	defer events.DrainQueuedInputsForTest(mob.InstanceId)

	ok := TryMobBehavior(mob.InstanceId, EventContext{EventType: "mob_combat_round"})
	if !ok {
		t.Fatalf("TryMobBehavior: expected Success (tree handled event), got false")
	}

	cmd := events.InspectQueuedInputForTest(mob.InstanceId, "cast ")
	if !strings.HasPrefix(cmd, "cast conviction-armor") {
		t.Fatalf("defense-only mob should cast conviction-armor (score 300 > ward 120), got %q", cmd)
	}
}
