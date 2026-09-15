package mutations

// GetAllyAuraConditions returns the condition ids that owned mutations project onto
// nearby allies (effect type "aura_ally_condition", Value = condition id).
func GetAllyAuraConditions(owned map[string]int) []int {
	var out []int
	for id := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, p := range spec.Pros {
			if p.Type == "aura_ally_condition" && p.Value > 0 {
				out = append(out, int(p.Value))
			}
		}
	}
	return out
}

// GetEnemyAuraConditions returns the harmful condition ids that owned mutations project onto
// nearby enemies (effect type "aura_enemy_condition", Value = condition id).
func GetEnemyAuraConditions(owned map[string]int) []int {
	var out []int
	for id := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, p := range spec.Pros {
			if p.Type == "aura_enemy_condition" && p.Value > 0 {
				out = append(out, int(p.Value))
			}
		}
	}
	return out
}
