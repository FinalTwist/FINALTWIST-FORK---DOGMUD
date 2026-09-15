package mutations

import "testing"

func TestGetEnemyAuraConditions(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"dissonance-organ": {MutationId: "dissonance-organ", Name: "Dissonance Organ", Rarity: 5,
			Pros: []MutationEffect{{Type: "aura_enemy_condition", Value: 102}}},
	})
	defer cleanup()
	got := GetEnemyAuraConditions(map[string]int{"dissonance-organ": 1})
	if len(got) != 1 || got[0] != 102 {
		t.Fatalf("GetEnemyAuraConditions = %v, want [102]", got)
	}
	if len(GetEnemyAuraConditions(map[string]int{})) != 0 {
		t.Fatal("no mutations → no enemy auras")
	}
}

func TestDescribeEffect_AuraEnemyHarmfulCondition(t *testing.T) {
	if DescribeEffect(MutationEffect{Type: "aura_enemy_condition", Value: 102}) == "" {
		t.Fatal("aura_enemy_condition must have a non-empty description")
	}
}
