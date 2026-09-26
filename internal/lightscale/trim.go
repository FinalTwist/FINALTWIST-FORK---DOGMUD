package lightscale

import "math"

// Polarity says which way an adjustable source pushes its room: a light
// raises it, a darkness lowers it. It is the only difference between the two
// kinds of adjustable source, so one Trim serves both (owner ruling,
// 2026-09-26: "the same function, just inverted").
type Polarity int

const (
	Brightens Polarity = 1
	Darkens   Polarity = -1
)

// Trim returns the output an adjustable source should run at: the smallest
// cut from its full strength that keeps the room on the bearer's side of
// target.
//
// others is the room's light with this source left out, Absent when nothing
// else lights it.
//
// For a light, max and the result are light-scale terms fed to Combine. The
// result solves Combine(others, out) == target exactly, capped at max; it is
// Absent when the room already reaches target without this source. A linear
// "target - others" is wrong here: on a log scale adding a source does not add
// its value.
//
// For a darkness, max and the result are points subtracted from the combined
// light. The result is the cut that lands the room on target, capped at max,
// and 0 when the room is already at or below target. An Absent room counts as
// 0, the darkest light that occurs naturally.
func Trim(step, others, max, target float64, p Polarity) float64 {
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
	return math.Min(need, max)
}
