package aicompanion

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func withStepQuote(t *testing.T, q actions.MoveCharge) {
	t.Helper()
	prev := stepQuote
	stepQuote = func(*mobs.Mob, string) actions.MoveCharge { return q }
	t.Cleanup(func() { stepQuote = prev })
}

// withStepAffordable pins every step as payable now, or as too tired to pay
// now but payable once rested.
func withStepAffordable(t *testing.T, ok bool) {
	t.Helper()
	if ok {
		withStepQuote(t, actions.MoveCharge{})
		return
	}
	withStepQuote(t, actions.MoveCharge{Refusal: actions.MoveRefuseExhausted})
}

func tiredLines(c *controller) int {
	n := 0
	for _, l := range c.mind.RecentLines {
		if strings.Contains(l.Text, `too tired to go on`) {
			n++
		}
	}
	return n
}

// Movement parity 4b: a tired companion does not issue the step, does not
// start the step clock, writes one line in its mind (not one a round), and
// takes the step once rested.
func TestTiredCompanionRestsOnItsTrip(t *testing.T) {
	w := newConsentWorld(t, true)
	here := w.her.Character.RoomId
	w.c.travel = &travelPlan{Dest: 9, DestName: `the mill`, Purpose: `errand`,
		Steps: []step{{Exit: `north`, To: 9}}, FromRoom: here}
	t.Cleanup(func() { events.DrainQueuedInputsForTest(w.her.InstanceId) })

	withStepAffordable(t, false)
	w.m.advanceTravel(w.c, w.her, w.owner, 50)
	w.m.advanceTravel(w.c, w.her, w.owner, 51)

	if w.c.travel == nil || w.c.travel.Expect != 0 {
		t.Fatalf("a tired companion must not start the step clock: %+v", w.c.travel)
	}
	if got := events.InspectQueuedInputForTest(w.her.InstanceId, `go `); got != `` {
		t.Fatalf("a tired companion issued %q", got)
	}
	if n := tiredLines(w.c); n != 1 {
		t.Fatalf("expected exactly one resting line in her mind, got %d", n)
	}

	withStepAffordable(t, true)
	w.m.advanceTravel(w.c, w.her, w.owner, 52)
	if got := events.InspectQueuedInputForTest(w.her.InstanceId, `go `); got == `` {
		t.Fatal("rested, she takes the step")
	}
}

// A step issued while she could pay, that then timed out because she could
// not (stamina changed in between), is a rest, not a bad exit: no Fails.
func TestTimedOutTiredStepDoesNotMarkTheExit(t *testing.T) {
	w := newConsentWorld(t, true)
	here := w.her.Character.RoomId
	w.c.mind.Map = map[int]*RoomRecord{here: {Title: `A Lane`, Exits: map[string]*ExitRecord{`north`: {To: 9}}}}
	w.c.travel = &travelPlan{Dest: 9, DestName: `the mill`, Purpose: `errand`,
		Steps: []step{{Exit: `north`, To: 9}}, FromRoom: here, Expect: 9, Issued: 10}

	withStepAffordable(t, false)
	w.m.advanceTravel(w.c, w.her, w.owner, 10+stepTimeoutRounds)

	if f := w.c.mind.Map[here].Exits[`north`].Fails; f != 0 {
		t.Fatalf("exhaustion wrote %d failure(s) against the exit", f)
	}
	if w.c.travel == nil || w.c.travel.Expect != 0 {
		t.Fatalf("the step must be re-quoted next round: %+v", w.c.travel)
	}
}

// A step she could not pay for even fully rested (the quote says Never) ends
// the trip the way any failed trip ends (endTravel, with its mind line),
// instead of resting forever. Both where a step is issued and where an issued
// step timed out.
func TestNeverAffordableStepEndsTheTrip(t *testing.T) {
	for _, timedOut := range []bool{false, true} {
		name := "before issuing"
		if timedOut {
			name = "after a timed-out step"
		}
		t.Run(name, func(t *testing.T) {
			w := newConsentWorld(t, true)
			here := w.her.Character.RoomId
			w.c.mind.Map = map[int]*RoomRecord{here: {Title: `A Lane`, Exits: map[string]*ExitRecord{`north`: {To: 9}}}}
			w.c.travel = &travelPlan{Dest: 9, DestName: `the mill`, Purpose: `errand`,
				Steps: []step{{Exit: `north`, To: 9}}, FromRoom: here}
			round := uint64(50)
			if timedOut {
				w.c.travel.Expect, w.c.travel.Issued = 9, 10
				round = 10 + stepTimeoutRounds
			}
			t.Cleanup(func() { events.DrainQueuedInputsForTest(w.her.InstanceId) })

			withStepQuote(t, actions.MoveCharge{Refusal: actions.MoveRefuseExhausted, Never: true})
			w.m.advanceTravel(w.c, w.her, w.owner, round)

			if w.c.travel != nil {
				t.Fatalf("a step she can never pay for must end the trip: %+v", w.c.travel)
			}
			if got := events.InspectQueuedInputForTest(w.her.InstanceId, `go `); got != `` {
				t.Fatalf("she issued %q", got)
			}
			if f := w.c.mind.Map[here].Exits[`north`].Fails; f != 0 {
				t.Fatalf("exhaustion wrote %d failure(s) against the exit", f)
			}
			last := w.c.mind.RecentLines[len(w.c.mind.RecentLines)-1]
			if !strings.Contains(last.Text, `the mill`) {
				t.Fatalf("expected a trip-ending line naming the destination, got %q", last.Text)
			}
		})
	}
}
