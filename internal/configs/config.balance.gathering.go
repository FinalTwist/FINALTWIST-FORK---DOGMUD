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
}
