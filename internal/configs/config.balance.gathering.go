package configs

// validateGathering sets defaults for the wilderness-trades gathering knobs:
// the gathering roll, tool tier multipliers and material grade sell values.
//
// Every knob uses the <=0 idiom. None of them has a meaningful zero: a zero
// skill weight is the only candidate, and "skill does nothing at all" is a
// design change that should be made by editing this file, not by a config
// typo that silently erases it.
func (b *Balance) validateGathering() {
	if b.GatherSkillWeight <= 0 {
		b.GatherSkillWeight = 1.5
	}
	if b.GatherBaseDifficulty <= 0 {
		b.GatherBaseDifficulty = 100
	}
	if b.GatherGradeStepSigma <= 0 {
		b.GatherGradeStepSigma = 1.0
	}

	if b.ToolMultCrude <= 0 {
		b.ToolMultCrude = 0.8
	}
	if b.ToolMultIron <= 0 {
		b.ToolMultIron = 1.0
	}
	if b.ToolMultSteel <= 0 {
		b.ToolMultSteel = 1.15
	}
	if b.ToolMultMasterwork <= 0 {
		b.ToolMultMasterwork = 1.3
	}

	if b.QualityValueCrude <= 0 {
		b.QualityValueCrude = 0.4
	}
	if b.QualityValueStandard <= 0 {
		b.QualityValueStandard = 1.0
	}
	if b.QualityValueFine <= 0 {
		b.QualityValueFine = 1.6
	}
	if b.QualityValueSuperb <= 0 {
		b.QualityValueSuperb = 2.5
	}
	if b.QualityValuePristine <= 0 {
		b.QualityValuePristine = 4.0
	}

	if b.GatherStatPoolDifficulty <= 0 {
		b.GatherStatPoolDifficulty = 0.2
	}
	if b.GatherCarcassEase <= 0 {
		b.GatherCarcassEase = 20
	}
	if b.GatherSizeDifficultyMedium <= 0 {
		b.GatherSizeDifficultyMedium = 5
	}
	if b.GatherSizeDifficultyLarge <= 0 {
		b.GatherSizeDifficultyLarge = 15
	}
	if b.GatherTargetedDifficulty <= 0 {
		b.GatherTargetedDifficulty = 15
	}
	if b.GatherJobRoundsSmall <= 0 {
		b.GatherJobRoundsSmall = 2
	}
	if b.GatherJobRoundsMedium <= 0 {
		b.GatherJobRoundsMedium = 4
	}
	if b.GatherJobRoundsLarge <= 0 {
		b.GatherJobRoundsLarge = 6
	}
	if b.GatherRareBaseChance <= 0 {
		b.GatherRareBaseChance = 0.15
	}
	if b.GatherStatPerBonusUnit <= 0 {
		b.GatherStatPerBonusUnit = 50
	}
	if b.CorpseStaleGradeAt <= 0 {
		b.CorpseStaleGradeAt = 0.5
	}
	if b.CorpseMeatLostAt <= 0 {
		b.CorpseMeatLostAt = 0.75
	}
	if b.ShopWalkInDevaluePerUnit <= 0 {
		b.ShopWalkInDevaluePerUnit = 0.02
	}
}
