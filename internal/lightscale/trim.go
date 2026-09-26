package lightscale

import "math"

// Polarity says which way an adjustable source pushes its room: a light
// raises it, a darkness lowers it. It is the only difference between the two
// kinds of adjustable source, so one Trim serves both (owner ruling,
// 2026-09-26: "the same function, just inverted"). Any value other than
// Darkens, including the zero value, is treated as a light.
type Polarity int

const (
	// Brightens marks an adjustable light source: Trim raises the room.
	Brightens Polarity = 1
	// Darkens marks an adjustable darkness source: Trim lowers the room.
	Darkens Polarity = -1
)

// Trim returns the output an adjustable source should run at: the smallest
// cut from its full strength that keeps the room on the bearer's side of
// target.
//
// others is the room's light with this source left out, Absent when nothing
// else lights it.
//
// For a light, max and the result are light-scale terms fed to Combine. The
// result solves Combine(others, out) == target analytically; in floating
// point to rounding, capped at max. A non-positive max is passed through
// unchanged, because 0 (or below) is a legitimate light term and the caller,
// not Trim, is responsible for refusing a strengthless source. The result is
// Absent when the room already reaches target without this source, or when
// the term the arithmetic needs would fall below 0, the darkest light that
// occurs naturally: such a source would have to be darker than an unlit cave
// to matter, so it is not needed at all rather than "lit" at a meaningless
// negative value. A linear "target - others" is wrong here: on a log scale
// adding a source does not add its value.
//
// For a darkness, max and the result are points subtracted from the combined
// light. The result is the cut that lands the room on target, capped at max,
// and 0 when the room is already at or below target or when max is
// non-positive. An Absent room counts as 0, the darkest light that occurs
// naturally, and is cut like any other level.
//
// A NaN target or max cannot produce a meaningful term; Trim returns Absent
// for a light and 0 for a darkness rather than propagate the NaN.
//
// The "never below 0" floor on a light assumes a target well above 0 (5a's
// run 50 to 74). At a target at or below 0, a present room would always
// return Absent while an Absent room still returns min(target, max): a
// caller with a low target must apply the same floor to that branch's result
// before using it, which is 5d's problem to solve, not this function's.
func Trim(step, others, max, target float64, p Polarity) float64 {
	if math.IsNaN(target) || math.IsNaN(max) {
		if p == Darkens {
			return 0
		}
		return Absent()
	}
	if !(step > 0) {
		step = 1
	}
	if p == Darkens {
		level := others
		if !present(level) {
			level = 0
		}
		cut := level - target
		if cut <= 0 || max <= 0 {
			return 0
		}
		return math.Min(cut, max)
	}
	if !present(others) {
		return math.Min(target, max)
	}
	if others >= target {
		return Absent()
	}
	need := target + step*math.Log2(1-math.Exp2((others-target)/step))
	if need < 0 {
		return Absent()
	}
	return math.Min(need, max)
}
