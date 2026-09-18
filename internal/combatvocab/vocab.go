// Package combatvocab is the one declaration of the four combat axes:
// what kind of attack it is, what kind of harm it does, how many it reaches,
// and which defences may answer it. Every other package imports this one; it
// imports nothing but the standard library.
//
// M4b-2 of the messaging arc (docs/superpowers/specs/2026-09-18-messaging-m4b2-axes-design.md).
// Before it, the same five defence names were declared three times, the
// attack and damage axes were flattened into one five-value enum, and the
// three damage types were spelt differently in three places.
package combatvocab

import "fmt"

// AttackType is HOW the attack is delivered. Parry is gated on it: you cannot
// parry a bolt, a flask or a working.
type AttackType string

const (
	AttackMelee    AttackType = "melee"
	AttackRanged   AttackType = "ranged"
	AttackThrown   AttackType = "thrown"
	AttackSpell    AttackType = "spell"
	AttackRhetoric AttackType = "rhetoric"
	// AttackNone is the attack type of a cast that harms nobody: a heal is not
	// an attack. It is bound to DamageNonHarm by Attack.Valid (owner ruling
	// 2026-09-18) and is the uncontested pair.
	AttackNone AttackType = "none"
)

// DamageType is WHAT the attack does to its target. Quell and defy are gated
// on it. It is not the damage pipeline's pool (combat.DamageChannel); a
// physical spell is dodged but still scales magically. That pool is derived
// from these axes, never authored.
type DamageType string

const (
	DamagePhysical DamageType = "physical"
	DamageMental   DamageType = "mental"
	DamageSocial   DamageType = "social"
	DamageNonHarm  DamageType = "non_harm"
)

// Targeting is HOW MANY the attack reaches. It never changes eligibility
// (owner ruling 2026-09-17): a cleave is still parryable. It changes how many
// contests happen, the defence text, and whether a counter is earned.
type Targeting string

const (
	// TargetSelf means NO target is resolved and the argument text passes
	// through: summons and identify. It is not "defaults to the caster";
	// that is TargetSingle with the caster as the default.
	TargetSelf   Targeting = "self"
	TargetSingle Targeting = "single"
	TargetMulti  Targeting = "multi"
	TargetArea   Targeting = "area"
)

// Defence is one of the five things a defender can do. "" is DefenceNone and
// is the zero value: a swing nobody defended.
type Defence string

const (
	DefenceNone  Defence = ""
	DefenceDodge Defence = "dodge"
	DefenceParry Defence = "parry"
	DefenceBlock Defence = "block"
	DefenceQuell Defence = "quell"
	DefenceDefy  Defence = "defy"
)

var (
	attackTypes = []AttackType{AttackMelee, AttackRanged, AttackThrown, AttackSpell, AttackRhetoric, AttackNone}
	damageTypes = []DamageType{DamagePhysical, DamageMental, DamageSocial, DamageNonHarm}
	targetings  = []Targeting{TargetSelf, TargetSingle, TargetMulti, TargetArea}
	defences    = []Defence{DefenceDodge, DefenceParry, DefenceBlock, DefenceQuell, DefenceDefy}
)

// AttackTypes returns every declared value, in declaration order.
func AttackTypes() []AttackType { return append([]AttackType(nil), attackTypes...) }

// DamageTypes returns every declared value, in declaration order.
func DamageTypes() []DamageType { return append([]DamageType(nil), damageTypes...) }

// Targetings returns every declared value, in declaration order.
func Targetings() []Targeting { return append([]Targeting(nil), targetings...) }

// Defences returns the five real defences; DefenceNone is not one.
func Defences() []Defence { return append([]Defence(nil), defences...) }

func (a AttackType) Valid() bool {
	for _, v := range attackTypes {
		if a == v {
			return true
		}
	}
	return false
}

func (d DamageType) Valid() bool {
	for _, v := range damageTypes {
		if d == v {
			return true
		}
	}
	return false
}

func (t Targeting) Valid() bool {
	for _, v := range targetings {
		if t == v {
			return true
		}
	}
	return false
}

// Valid reports a real defence. DefenceNone is not valid: it is the absence
// of one.
func (d Defence) Valid() bool {
	for _, v := range defences {
		if d == v {
			return true
		}
	}
	return false
}

// IsHarm is the harm-versus-help question in one place.
func (d DamageType) IsHarm() bool { return d.Valid() && d != DamageNonHarm }

func ParseAttackType(s string) (AttackType, error) {
	if a := AttackType(s); a.Valid() {
		return a, nil
	}
	return "", fmt.Errorf("attack_type %q is not one of %v", s, attackTypes)
}

func ParseDamageType(s string) (DamageType, error) {
	if d := DamageType(s); d.Valid() {
		return d, nil
	}
	return "", fmt.Errorf("damage_type %q is not one of %v", s, damageTypes)
}

func ParseTargeting(s string) (Targeting, error) {
	if t := Targeting(s); t.Valid() {
		return t, nil
	}
	return "", fmt.Errorf("targeting %q is not one of %v", s, targetings)
}

func ParseDefence(s string) (Defence, error) {
	if d := Defence(s); d.Valid() {
		return d, nil
	}
	return "", fmt.Errorf("defence %q is not one of %v", s, defences)
}
