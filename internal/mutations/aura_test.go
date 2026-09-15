package mutations

import "testing"

func TestGetAllyAuraConditions(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"commanding-presence": {MutationId: "commanding-presence", Name: "Commanding Presence", Rarity: 4,
			Pros: []MutationEffect{{Type: "aura_ally_buff", Value: 101}}},
		"plain": {MutationId: "plain", Name: "Plain", Rarity: 2,
			Pros: []MutationEffect{{Type: "stat_flat", Target: "charisma", Value: 5}}},
	})
	defer cleanup()

	got := GetAllyAuraConditions(map[string]int{"commanding-presence": 1, "plain": 1})
	if len(got) != 1 || got[0] != 101 {
		t.Fatalf("GetAllyAuraConditions = %v, want [101]", got)
	}
	if len(GetAllyAuraConditions(map[string]int{})) != 0 {
		t.Fatal("no mutations → no auras")
	}
}

func TestDescribeEffect_AuraAllyCondition(t *testing.T) {
	if DescribeEffect(MutationEffect{Type: "aura_ally_buff", Value: 101}) == "" {
		t.Fatal("aura_ally_buff must have a non-empty description")
	}
}
