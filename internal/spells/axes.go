package spells

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// Attack is the attack the seam resolves for this spell, built from the
// three axes.
func (s *SpellData) Attack() combatvocab.Attack {
	return combatvocab.Attack{Type: s.AttackType, Damage: s.DamageType, Targeting: s.Targeting}
}

// IsHarm is the harm-versus-help question, in one place. It replaced every
// `Type == HarmSingle || Type == HarmArea || Type == HarmMulti`.
func (s *SpellData) IsHarm() bool {
	return s.DamageType.IsHarm()
}

// HelpOrHarmString is the word the `spells` listing and the help template
// print. Derived, and pinned by TestDisplayStringsMatchTheLegacyTypes to
// what SpellType.HelpOrHarmString printed: a non-harm cast that resolves no
// target (self) is "Neutral", any other non-harm cast is "Helpful".
func (s *SpellData) HelpOrHarmString() string {
	switch {
	case s.IsHarm():
		return `Harmful`
	case s.DamageType == combatvocab.DamageNonHarm && s.Targeting == combatvocab.TargetSelf:
		return `Neutral`
	case s.DamageType == combatvocab.DamageNonHarm:
		return `Helpful`
	}
	return `Unknown`
}

// TargetTypeString is the targeting word the listing (short) and the help
// template (long) print, pinned to SpellType.TargetTypeString's output.
func (s *SpellData) TargetTypeString(short ...bool) string {
	isShort := len(short) > 0 && short[0]
	switch s.Targeting {
	case combatvocab.TargetSelf:
		return `Self`
	case combatvocab.TargetSingle:
		if isShort {
			return `Single`
		}
		return `Single Target`
	case combatvocab.TargetMulti:
		if isShort {
			return `Group`
		}
		return `Group Target`
	case combatvocab.TargetArea:
		if isShort {
			return `Area`
		}
		return `Area Target`
	}
	return `Unknown`
}

// DefenceNames is the help template's "Resisted by" value, derived from the
// eligibility table so it can never disagree with what actually rolls. Empty
// for a non-harm cast, which suppresses the line.
func (s *SpellData) DefenceNames() string {
	set, ok := combatvocab.EligibleDefences(s.Attack())
	if !ok || len(set) == 0 {
		return ""
	}
	names := make([]string, 0, len(set))
	for _, d := range set {
		names = append(names, string(d))
	}
	return strings.Join(names, " or ")
}

// validateAxes is called from Validate. The three keys are required,
// the pair must be in the table (which also binds none to non_harm), and
// the targeting must be one of the four.
func (s *SpellData) validateAxes() error {
	if s.AttackType == "" || s.DamageType == "" || s.Targeting == "" {
		return fmt.Errorf("spell %q: attack_type, damage_type and targeting are all required", s.SpellId)
	}
	if _, err := combatvocab.ParseAttackType(string(s.AttackType)); err != nil {
		return fmt.Errorf("spell %q: %w", s.SpellId, err)
	}
	if _, err := combatvocab.ParseDamageType(string(s.DamageType)); err != nil {
		return fmt.Errorf("spell %q: %w", s.SpellId, err)
	}
	if _, err := combatvocab.ParseTargeting(string(s.Targeting)); err != nil {
		return fmt.Errorf("spell %q: %w", s.SpellId, err)
	}
	if !s.Attack().Valid() {
		return fmt.Errorf("spell %q: attack_type %s with damage_type %s is not a pairing the eligibility table knows (none pairs only with non_harm)", s.SpellId, s.AttackType, s.DamageType)
	}
	return nil
}
