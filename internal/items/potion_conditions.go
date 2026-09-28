package items

// PotionEffectConditionIds returns every condition id that only potions
// grant: named by a potion's ConditionIds and by no non-potion item's
// ConditionIds or WornConditionIds. The Purging Draught strips this set
// (lighting plan 5c), which replaced a hardcoded id block that shipped
// potions had already outgrown. It is computed on each call from the loaded
// specs, so an admin item reload cannot leave it stale.
func PotionEffectConditionIds() map[int]bool {
	potion := map[int]bool{}
	other := map[int]bool{}
	for _, spec := range GetAllItemSpecs() {
		if spec.Type == Potion {
			for _, id := range spec.ConditionIds {
				potion[id] = true
			}
			continue
		}
		for _, id := range spec.ConditionIds {
			other[id] = true
		}
		for _, id := range spec.WornConditionIds {
			other[id] = true
		}
	}
	for id := range other {
		delete(potion, id)
	}
	return potion
}
