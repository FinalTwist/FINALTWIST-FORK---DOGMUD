package messaging

import "testing"

// TestSleeperReadsNothingFromTheClearSightPolicies is the consequence guard for
// M4d's inversion. Sleep is about to move from being written inside each
// predicate to being composed over a shared verdict, and the failure mode is
// silent: a predicate that loses its attention test starts narrating to
// sleepers, and no golden covers that, because no golden drives a sleeping
// observer.
//
// CanSeeSightImpairedOnly is deliberately NOT in this list. It has no attention
// test today and must not gain one: it feeds DarknessCombatPenalty, and a
// sleeping defender is already auto-crit, so doubling their disadvantage was
// never asked for. See predicates.go's own comment.
func TestSleeperReadsNothingFromTheClearSightPolicies(t *testing.T) {
	sleeper := newOpticsObserver(t, opticsCase{asleep: true, lit: true})
	room := newOpticsRoom(t, true)

	if CanSeeClearly(sleeper, room) {
		t.Error("a sleeper must not pass CanSeeClearly, even in a lit room")
	}
	if CanSeeShapes(sleeper, room) {
		t.Error("a sleeper must not pass CanSeeShapes, even in a lit room")
	}
	if !CanSeeSightImpairedOnly(sleeper, room) {
		t.Error("CanSeeSightImpairedOnly must IGNORE sleep: it feeds the combat darkness penalty")
	}
}
