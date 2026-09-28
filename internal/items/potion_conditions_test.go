package items

import "testing"

func TestPotionEffectConditionIds(t *testing.T) {
	t.Cleanup(SeedItemsForTest(map[int]*ItemSpec{
		1: {ItemId: 1, Type: Potion, ConditionIds: []int{7, 5}},
		2: {ItemId: 2, Type: Potion, ConditionIds: []int{130}},
		3: {ItemId: 3, Type: Food, ConditionIds: []int{5}},
		4: {ItemId: 4, Type: Legs, WornConditionIds: []int{7}},
	}))
	got := PotionEffectConditionIds()
	if !got[130] {
		t.Error("130 is named only by a potion; it must be in the set")
	}
	if got[5] {
		t.Error("5 is also granted by food; stripping it would undo a meal")
	}
	if got[7] {
		t.Error("7 is also granted while worn; stripping it would undo armour")
	}
}
