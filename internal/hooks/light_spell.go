package hooks

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// lightSpellApplication reports how a condition from a spell should be applied
// when it is a magnitude-driven light: at a strength and a duration scaled
// from the CASTER's primary stat and spellcasting skill (lighting plan 5a).
// ok is false for any other condition, which keeps its authored application.
// The record then trims to its HOLDER's eyes, who may not be the caster.
func lightSpellApplication(spellData *spells.SpellData, caster *characters.Character, conditionId int) (magnitude float64, triggers int, ok bool) {
	if spellData == nil || caster == nil {
		return 0, 0, false
	}
	spec := conditions.GetConditionSpec(conditionId)
	if spec == nil {
		return 0, 0, false
	}
	if v, isLight := spec.Effects[conditions.EffectLightStrength]; !isLight || !v.UsesMagnitude {
		return 0, 0, false
	}
	cfg := configs.GetLightingConfig()
	stat := float64(spellData.CasterStatValue(caster.Stats))
	skill := float64(caster.GetSkillLevel(skills.Spellcasting))
	magnitude = cfg.SpellStrengthBase + stat/cfg.SpellStrengthStatDivisor + skill/cfg.SpellStrengthSkillDivisor
	triggers = int(math.Round(cfg.SpellDurationBase + stat/cfg.SpellDurationStatDivisor + skill/cfg.SpellDurationSkillDivisor))
	if triggers < 1 {
		triggers = 1
	}
	return magnitude, triggers, true
}
