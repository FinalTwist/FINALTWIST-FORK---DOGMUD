package conditions

import (
	"fmt"
	"sort"
	"strconv"
)

// EffectKind is one of the closed set of mechanical effects a record may
// declare. Combat reads them through Conditions.Effect. The set is closed on
// purpose: a new kind is a code change with a reader, never a data change.
type EffectKind string

const (
	EffectDamageMult     EffectKind = "damage_mult"     // physical damage multiplier (warcry)
	EffectDefenseMult    EffectKind = "defense_mult"    // defense score multiplier (rally, grapple exposure)
	EffectDodgeMult      EffectKind = "dodge_mult"      // dodge score multiplier (no producer today; kept for parity with the reader)
	EffectRegenMult      EffectKind = "regen_mult"      // multiplier on base health regen (heal spells, corpse feeding)
	EffectMitigationFlat EffectKind = "mitigation_flat" // flat physical mitigation points (wards)
	EffectPoolMaxPct     EffectKind = "pool_max_pct"    // fraction taken off a pool maximum; the pool rides on Condition.Source
	EffectAttacksCap     EffectKind = "attacks_cap"     // upper bound on swings per round
	// EffectNightVisionStrength is how far DOWN the scale an observer's usable
	// light band shifts. Aggregated as MAX, not summed: two night-sight
	// sources do not stack into a wider window than the better one grants.
	EffectNightVisionStrength EffectKind = `nightvision_strength`
	// EffectInfraReach is how far BELOW the window floor heat-sensing still
	// reads shapes. Independent of strength: a creature can sense heat deeply
	// while being no better than anyone else at using faint light.
	EffectInfraReach EffectKind = `infra_reach`
)

// AllEffectKinds is the closed set, for validation and docs.
var AllEffectKinds = []EffectKind{
	EffectDamageMult, EffectDefenseMult, EffectDodgeMult, EffectRegenMult,
	EffectMitigationFlat, EffectPoolMaxPct, EffectAttacksCap,
	EffectNightVisionStrength, EffectInfraReach,
}

func (k EffectKind) isMultiplier() bool {
	return k == EffectDamageMult || k == EffectDefenseMult || k == EffectDodgeMult || k == EffectRegenMult
}

func (k EffectKind) isCap() bool { return k == EffectAttacksCap }

// isMax reports whether this kind aggregates by taking the strongest held
// value. Used by the vision window, where summing would let two abilities
// stack into a window wider than either one grants.
func (k EffectKind) isMax() bool {
	return k == EffectNightVisionStrength || k == EffectInfraReach
}

// EffectValue is either a literal number or the word "magnitude", meaning the
// instance's own Magnitude, which the applier set.
type EffectValue struct {
	Literal       float64
	UsesMagnitude bool
}

func (v *EffectValue) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err == nil {
		if s == "magnitude" {
			*v = EffectValue{UsesMagnitude: true}
			return nil
		}
		f, ferr := strconv.ParseFloat(s, 64)
		if ferr != nil {
			return fmt.Errorf("effect value %q is neither a number nor the word magnitude", s)
		}
		*v = EffectValue{Literal: f}
		return nil
	}
	var f float64
	if err := unmarshal(&f); err != nil {
		return fmt.Errorf("effect value must be a number or the word magnitude: %w", err)
	}
	*v = EffectValue{Literal: f}
	return nil
}

func (v EffectValue) MarshalYAML() (interface{}, error) {
	if v.UsesMagnitude {
		return "magnitude", nil
	}
	return v.Literal, nil
}

// validateEffects refuses an unknown key and a magnitude-bound tick without a
// pool. It is called from ConditionSpec.Validate.
func (b *ConditionSpec) validateEffects() error {
	keys := make([]string, 0, len(b.Effects))
	for k := range b.Effects {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	for _, k := range keys {
		known := false
		for _, ak := range AllEffectKinds {
			if EffectKind(k) == ak {
				known = true
				break
			}
		}
		if !known {
			return fmt.Errorf("conditionId %d (%s) declares unknown effect %q; see conditions.AllEffectKinds", b.ConditionId, b.Name, k)
		}
	}
	if b.TickFromMagnitude {
		if b.TickPool == "" {
			return fmt.Errorf("conditionId %d (%s) sets tick_from_magnitude without tick_pool", b.ConditionId, b.Name)
		}
		if b.TickPercent != 0 {
			return fmt.Errorf("conditionId %d (%s) sets both tick_from_magnitude and tick_percent; the applier's magnitude IS the per-round amount", b.ConditionId, b.Name)
		}
	}
	return nil
}

// Effect combines every held, unexpired record's contribution for one kind:
// multipliers multiply (identity 1, a zero magnitude contributes nothing),
// flats and pool fractions sum (identity 0), attacks_cap takes the minimum
// (0 meaning no cap), and a max kind takes the strongest held value (identity
// 0). This is the ONE door timed state's mechanical effects are read through;
// combat was its first reader, and optics (the vision window) is now another.
// It never calls HasFlag with expire=true, which mutates.
func (bs *Conditions) Effect(kind EffectKind) float64 {
	product := 1.0
	sum := 0.0
	capValue := 0.0
	maxValue := 0.0
	for _, b := range bs.List {
		if b.Expired() {
			continue
		}
		spec := GetConditionSpec(b.ConditionId)
		if spec == nil {
			continue
		}
		v, ok := spec.Effects[kind]
		if !ok {
			continue
		}
		val := v.Literal
		if v.UsesMagnitude {
			val = b.Magnitude
		}
		switch {
		case kind.isMultiplier():
			if val != 0 {
				product *= val
			}
		case kind.isCap():
			if val > 0 && (capValue == 0 || val < capValue) {
				capValue = val
			}
		case kind.isMax():
			if val > maxValue {
				maxValue = val
			}
		default:
			sum += val
		}
	}
	switch {
	case kind.isMultiplier():
		return product
	case kind.isCap():
		return capValue
	case kind.isMax():
		return maxValue
	default:
		return sum
	}
}

// HasEffect reports whether any held, unexpired record declares the kind.
func (bs *Conditions) HasEffect(kind EffectKind) bool {
	for _, b := range bs.List {
		if b.Expired() {
			continue
		}
		if spec := GetConditionSpec(b.ConditionId); spec != nil {
			if _, ok := spec.Effects[kind]; ok {
				return true
			}
		}
	}
	return false
}
