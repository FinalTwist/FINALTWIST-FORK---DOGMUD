package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// spellTickScale is the one caster formula for a spell's heal- or
// damage-over-time (player/mob parity slice 2): the caster's
// spellcasting SkillMultiplier times the equipped weapon's spell
// multiplier, adjusted for gear effectiveness. It used to be computed
// three ways (player caster with the weapon, mob self-cast without it,
// mob on mob not at all).
func spellTickScale(caster *characters.Character) float64 {
	if caster == nil {
		return 1.0
	}
	scale := combat.SkillMultiplier(caster.GetSkillLevel(skills.Spellcasting))
	if caster.Equipment.Weapon.ItemId > 0 {
		if ws := items.GetItemSpec(caster.Equipment.Weapon.ItemId); ws != nil && ws.SpellDamageMultiplier > 0 {
			scale *= ws.SpellDamageMultiplier * mutations.GearEffectivenessMultiplier(caster.Mutations)
		}
	}
	return scale
}
