package conditions

import "github.com/GoMudEngine/GoMud/internal/configs"

// SpellScaledMagnitude is the magnitude a spell applies a condition reading
// kind (one of ScaledKinds) at, for a caster with the given primary-stat value
// and spellcasting skill: base + stat/statDivisor + skill/skillDivisor, from
// that kind's knob trio in configs.GetLightingConfig() (LightSpellStrength* for
// light, LightNightVisionSpell* for nightvision, LightInfraSpell* for infra
// reach), capped by CapScaledMagnitude so the record holds the value it acts
// at.
//
// internal/hooks' spell path calls it with the caster's numbers; the admin
// setcondition command calls it at a new character's (stat 100, skill 0), so a
// magnitude-scaled condition applied by hand lands at a real strength rather
// than zero.
func SpellScaledMagnitude(kind EffectKind, stat, skill float64) float64 {
	cfg := configs.GetLightingConfig()
	base, statDiv, skillDiv := cfg.SpellStrengthBase, cfg.SpellStrengthStatDivisor, cfg.SpellStrengthSkillDivisor
	switch kind {
	case EffectNightVisionStrength:
		base, statDiv, skillDiv = cfg.NightVisionSpellBase, cfg.NightVisionSpellStatDivisor, cfg.NightVisionSpellSkillDivisor
	case EffectInfraReach:
		base, statDiv, skillDiv = cfg.InfraSpellBase, cfg.InfraSpellStatDivisor, cfg.InfraSpellSkillDivisor
	}
	return CapScaledMagnitude(kind, base+stat/statDiv+skill/skillDiv)
}

// CapScaledMagnitude bounds a scaled kind's magnitude to the most it can act
// at: infra reach at LightInfraReachCap, nightvision strength at
// configs.LightWindowShiftCap (the window model clamps any strength there
// anyway, so a record above it would only misreport itself). Light is not
// capped here; the light scale clamps the room's composed light instead. The
// spell and potion paths both apply it.
func CapScaledMagnitude(kind EffectKind, magnitude float64) float64 {
	limit := -1.0
	switch kind {
	case EffectInfraReach:
		limit = float64(configs.GetLightingConfig().InfraReachCap)
	case EffectNightVisionStrength:
		limit = configs.LightWindowShiftCap
	}
	if limit >= 0 && magnitude > limit {
		return limit
	}
	return magnitude
}

// NewCharacterSpellStat and NewCharacterSpellSkill are the caster numbers
// SpellScaledMagnitude is evaluated at when no caster exists, such as the
// admin setcondition command: a new character's stat (stats centre on 100)
// and an untrained skill.
const (
	NewCharacterSpellStat  = 100.0
	NewCharacterSpellSkill = 0.0
)
