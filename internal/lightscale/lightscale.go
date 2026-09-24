// Package lightscale holds the arithmetic of DOGMud's graded light scale.
//
// The scale runs -100 to 100 and is PERCEPTUAL, not linear. Zero is the darkest
// light that naturally occurs, roughly an unlit cave; negative is magical
// darkness, light actively removed. Because the scale is logarithmic, one
// constant relates it to physical light: the doubling step, which is how many
// scale points twice as much light is worth.
//
// 🔑 The same step governs three things, which is why it is ONE config knob:
// combining sources, applying a sky fraction, and the shape of the daylight
// curve. See docs/superpowers/specs/2026-09-23-graded-room-lighting-amendment-celestial.md.
//
// This package is deliberately pure: no config reads, no globals, no locks. Its
// callers own the config read, the same discipline messaging.SightThroughWindow
// follows for the band edges.
package lightscale

import "math"

// Absent is a light term that is not present at all, as distinct from a term
// that is present and dark.
//
// 🔑 The distinction is load-bearing. A cave has no sky, which is not the same
// as a sky contributing zero: on a logarithmic scale two terms at zero
// legitimately combine to something brighter than one, so a "zero" sky would
// make a cave brighter for having a sky it does not have.
func Absent() float64 { return math.Inf(-1) }

// present reports whether a term should take part in a combination. Only finite
// values do; -Inf means absent and NaN means a caller made an arithmetic
// mistake, which must not silently poison the whole room.
//
// ⚠️ +Inf is folded into "absent" too, which is deliberate but is NOT the same
// judgement as the other two. -Inf is a legitimate value this package produces
// on purpose, and NaN is excluded so one bad term cannot poison a whole room.
// +Inf is neither: no caller can currently produce it, and if one ever does it
// is a bug upstream, most likely a division by zero. Skipping it means that bug
// would vanish without trace rather than reddening a test. It is folded in here
// only because a term of infinite brightness has no sane reading on a bounded
// -100..100 scale, so there is nothing better to do with it locally. If plans 4
// or 5 add a source whose magnitude is computed by division, give +Inf its own
// branch and make it loud.
func present(v float64) bool { return !math.IsInf(v, 0) && !math.IsNaN(v) }

// Combine returns the light produced by every present term together.
//
//	combined = brightest + step * log2( sum over i of 2^((s_i - brightest)/step) )
//
// Two equal terms read one step brighter than one; four read two steps brighter;
// a term far below the brightest contributes almost nothing. Absent terms are
// skipped, and a combination of nothing is Absent.
//
// It is computed relative to the brightest term rather than from an absolute
// origin so that large scale values cannot overflow the exponential.
func Combine(step float64, terms ...float64) float64 {
	if !(step > 0) {
		step = 1
	}
	best := math.Inf(-1)
	for _, t := range terms {
		if present(t) && t > best {
			best = t
		}
	}
	if math.IsInf(best, -1) {
		return Absent()
	}
	sum := 0.0
	for _, t := range terms {
		if present(t) {
			sum += math.Exp2((t - best) / step)
		}
	}
	// Unreachable as written, and kept as a backstop rather than removed. Once
	// best is finite, the very term that set it is re-encountered by this loop
	// and contributes Exp2(0/step) = 1 exactly, so sum is always at least 1. It
	// stays because the guarantee depends on the two loops agreeing about which
	// terms are present: change the best-selection loop without changing the
	// summation loop and sum could reach here as zero, whose Log2 is -Inf.
	if !(sum > 0) {
		return best
	}
	return best + step*math.Log2(sum)
}

// Attenuate applies a transmission fraction to one light term: the share of the
// light that gets through a canopy, a roof, a drain-cap or a blizzard.
//
// 🔑 On a logarithmic scale a MULTIPLIER is a SUBTRACTION. Letting half the
// light through is minus one doubling step, whatever the light was. That is why
// canopy, roof and weather are one operator rather than three, and why a
// blizzard is devastating at midnight and merely gloomy at noon: it removes the
// same number of points in both cases, but the sight bands are absolute.
//
// A fraction at or below zero returns Absent, not a very dark value, because "no
// sky reaches here" is a different statement from "very little does".
func Attenuate(step, light, fraction float64) float64 {
	if !present(light) || !(fraction > 0) {
		return Absent()
	}
	if fraction >= 1 {
		return light
	}
	if !(step > 0) {
		step = 1
	}
	return light + step*math.Log2(fraction)
}
