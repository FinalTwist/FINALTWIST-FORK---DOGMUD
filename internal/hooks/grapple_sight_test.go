package hooks

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/contest"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Lighting plan 5b: the per-round grapple drift is an opposed roll where both
// grapplers need to see, so each side's score pays its own sight ramp in the
// pair's room. The injected runner records the scores the contest receives.
func TestGrappleDriftScoresTakeBothGrapplersEyes(t *testing.T) {
	setGrappleCostConfig(t)

	scoresAt := func(lamp int, ids int) (float64, float64) {
		const roomId = 99550
		cleanup := rooms.SeedRoomsForTest(map[int]*rooms.Room{
			roomId: {RoomId: roomId, Lamp: rooms.LampPtr(lamp)},
		}, map[string]*rooms.ZoneConfig{})
		defer cleanup()

		controller, controlled := grappleCostPair(t, ids, ids+1)
		controller.RoomId = roomId
		controlled.RoomId = roomId
		var ctrl, cd float64
		processGrapplePairWithContest(controller, controlled,
			func(attack float64, defenses []contest.Entry) contest.Result {
				ctrl, cd = attack, defenses[0].Score
				return fixedGrappleContest(0)(attack, defenses)
			})
		return ctrl, cd
	}

	ctrlComfortable, cdComfortable := scoresAt(60, 1700)
	ctrlDazzled, cdDazzled := scoresAt(90, 1702)
	if r := ctrlDazzled / ctrlComfortable; math.Abs(r-0.88) > 1e-9 {
		t.Errorf("controller dazzled/comfortable = %v, want 0.88", r)
	}
	if r := cdDazzled / cdComfortable; math.Abs(r-0.88) > 1e-9 {
		t.Errorf("controlled dazzled/comfortable = %v, want 0.88", r)
	}
}
