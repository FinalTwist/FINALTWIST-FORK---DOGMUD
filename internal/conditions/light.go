package conditions

import "math"

// LightTrim is where an adjustable light record's output stands. It is an
// explicit state rather than a stored -Inf, so a save never has to encode an
// infinity.
type LightTrim string

const (
	LightFull    LightTrim = ""        // untrimmed: full strength
	LightTrimmed LightTrim = "trimmed" // running at LightOutput
	LightOff     LightTrim = "off"     // trimmed to nothing: the room was bright enough
)

// LightMax is the record's full strength: the applier's magnitude for a spell
// source, the authored number for an item. 0 when spec declares no light. A
// magnitude light added with no magnitude (an admin setcondition) is 0 and
// sheds nothing; cast the spell instead.
func (b *Condition) LightMax(spec *ConditionSpec) float64 {
	if spec == nil {
		return 0
	}
	v, ok := spec.Effects[EffectLightStrength]
	if !ok {
		return 0
	}
	if v.UsesMagnitude {
		return b.Magnitude
	}
	return v.Literal
}

// LightNow is the term this record adds to its room's light right now, and
// false when it adds none: expired, hooded, trimmed to nothing, or strengthless.
func (b *Condition) LightNow(spec *ConditionSpec) (float64, bool) {
	if b.Expired() || b.Hooded {
		return 0, false
	}
	max := b.LightMax(spec)
	if max <= 0 {
		return 0, false
	}
	switch b.LightTrim {
	case LightOff:
		return 0, false
	case LightTrimmed:
		return b.LightOutput, true
	}
	return max, true
}

// SetLightOutput records a trim result from lightscale.Trim. A non-finite
// output (lightscale.Absent) means the room needs nothing from this source.
func (b *Condition) SetLightOutput(out float64) {
	if math.IsInf(out, 0) || math.IsNaN(out) {
		b.LightTrim, b.LightOutput = LightOff, 0
		return
	}
	b.LightTrim, b.LightOutput = LightTrimmed, out
}

// ResetLight returns the record to full strength with any hood open: a fresh
// cast, a fresh equip, or unhood.
func (b *Condition) ResetLight() {
	b.LightTrim, b.LightOutput, b.Hooded = LightFull, 0, false
}

// LightSources returns every held, unexpired record whose spec declares a
// light strength, in held order. The order is stable, so a bearer's several
// sources always trim in the same sequence.
func (bs *Conditions) LightSources() []*Condition {
	var out []*Condition
	for _, b := range bs.List {
		if b.Expired() {
			continue
		}
		if spec := GetConditionSpec(b.ConditionId); spec != nil && spec.IsLightSource() {
			out = append(out, b)
		}
	}
	return out
}
