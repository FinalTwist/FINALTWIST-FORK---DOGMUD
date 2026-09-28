package hooks

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// magnitudeSpellApplication reports how a spell's condition should be applied
// when it reads one of conditions.ScaledKinds from its magnitude: at a value
// and a duration scaled from the CASTER's primary stat and spellcasting skill
// (lighting plan 5a for light, 5c for nightvision and infra reach). Each kind
// has its own base + stat/D1 + skill/D2 trio; all three share the light
// duration trio. Infra reach is capped at LightInfraReachCap here so the
// record holds the value it acts at; nightvision is left to the window's own
// clamp. ok is false for any other condition, which keeps its authored
// application. A light then trims to its HOLDER's eyes, who may not be the
// caster.
func magnitudeSpellApplication(spellData *spells.SpellData, caster *characters.Character, conditionId int) (magnitude float64, triggers int, ok bool) {
	if spellData == nil || caster == nil {
		return 0, 0, false
	}
	spec := conditions.GetConditionSpec(conditionId)
	if spec == nil {
		return 0, 0, false
	}
	kind, scaled := spec.ScaledKind()
	if !scaled {
		return 0, 0, false
	}
	cfg := configs.GetLightingConfig()
	base, statDiv, skillDiv := cfg.SpellStrengthBase, cfg.SpellStrengthStatDivisor, cfg.SpellStrengthSkillDivisor
	switch kind {
	case conditions.EffectNightVisionStrength:
		base, statDiv, skillDiv = cfg.NightVisionSpellBase, cfg.NightVisionSpellStatDivisor, cfg.NightVisionSpellSkillDivisor
	case conditions.EffectInfraReach:
		base, statDiv, skillDiv = cfg.InfraSpellBase, cfg.InfraSpellStatDivisor, cfg.InfraSpellSkillDivisor
	}
	stat := float64(spellData.CasterStatValue(caster.Stats))
	skill := float64(caster.GetSkillLevel(skills.Spellcasting))
	magnitude = base + stat/statDiv + skill/skillDiv
	if kind == conditions.EffectInfraReach {
		if limit := float64(cfg.InfraReachCap); magnitude > limit {
			magnitude = limit
		}
	}
	triggers = int(math.Round(cfg.SpellDurationBase + stat/cfg.SpellDurationStatDivisor + skill/cfg.SpellDurationSkillDivisor))
	if triggers < 1 {
		triggers = 1
	}
	return magnitude, triggers, true
}

// spellConditionTarget is what a spell condition lands on: a player record or
// a mob, both of which queue the narrating events.Condition.
type spellConditionTarget interface {
	AddCondition(conditionId int, source string)
	AddConditionMagnitude(conditionId int, triggers int, magnitude float64, source string)
}

// applySpellCondition applies one of a spell's conditions to its target: a
// magnitude-scaled light or sight at the caster's scaled value and duration,
// anything else at its authored values. Both doors queue events.Condition, so
// the holder reads the start notice either way.
func applySpellCondition(target spellConditionTarget, spellData *spells.SpellData, caster *characters.Character, conditionId int) {
	if mag, trig, ok := magnitudeSpellApplication(spellData, caster, conditionId); ok {
		target.AddConditionMagnitude(conditionId, trig, mag, "spell")
		return
	}
	target.AddCondition(conditionId, "spell")
}
