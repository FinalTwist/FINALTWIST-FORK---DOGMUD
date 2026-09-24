package messaging

// The normal observer's band edges on the graded light scale, and the two
// numbers that bound how far an ability may move them.
//
// These are CONSTANTS, not config knobs, and that is deliberate. Plan 1's rule
// is that a config knob nothing reads does not ship. windowDazzleEdge is needed
// to compute the window's upper edge but carries no mechanical penalty in this
// plan, so exposing it as a balance lever would ship a knob an operator could
// turn with no observable effect. It becomes a knob in the plan that gives
// dazzle teeth.
//
// The lower two edges DO have config knobs already (LightBlindBelow and
// LightDimBelow, shipped by plan 1) and the window function takes them as
// arguments rather than reading config, so this file stays pure and testable.
const (
	// windowDazzleEdge is where the perfect band ends and too-bright begins.
	windowDazzleEdge = 75
	// windowShiftCap is the most any ability may move the window down.
	windowShiftCap = 24
	// windowFloor is the light below which a shifted window reads nothing,
	// no matter how strong. Only an infra reach sees past it.
	windowFloor = 1
)

// SightThroughWindow reports what an observer reads at a given light level.
//
// strength moves every band edge DOWN by that many points, capped at
// windowShiftCap and floored at zero, so an ability trades bright-light comfort
// for dark-light acuity rather than simply gaining sight. reach is the separate
// heat-sensing extension that operates only at or below windowFloor; light at
// or above the negation of reach reads shapes, below that reads nothing.
//
// It takes the two lower band edges as arguments rather than reading config, so
// it stays a pure function with no locks and no global state. Its caller owns
// the config read.
func SightThroughWindow(light, strength, reach int, blindBelow, dimBelow int) SightDecision {
	if strength < 0 {
		strength = 0
	}
	if strength > windowShiftCap {
		strength = windowShiftCap
	}
	if reach < 0 {
		reach = 0
	}

	shiftedBlind := blindBelow - strength
	shiftedDim := dimBelow - strength

	if light >= shiftedDim {
		// Perfect and too-bright both read fully. Dazzle has no mechanical
		// penalty in this plan, so the upper edge is not consulted yet; it is
		// declared above so the next plan has one place to add the penalty.
		return SightFull
	}
	if light >= shiftedBlind && light >= windowFloor {
		return SightShapes
	}
	// Below the shifted window. Reach only operates at or below windowFloor;
	// without this gate a large reach would read shapes for any light the
	// shifted window merely failed to cover, even in ordinary dim light far
	// above the floor, which is not what "heat-sensing in the dark" means.
	if light <= windowFloor && reach > 0 && light >= -reach {
		return SightShapes
	}
	return SightNone
}
