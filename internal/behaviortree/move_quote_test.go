package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/require"
)

func tiredStepWorld(t *testing.T, apMax int) *mobs.Mob {
	t.Helper()
	// QuoteMobStep resolves the exit through FindExitByName, which reads the
	// direction aliases; a test binary has none loaded (same gap Task 7's
	// walkerWorld hit).
	t.Cleanup(keywords.SeedKeywordsForTest())
	room1 := &rooms.Room{RoomId: 1, Zone: "TestZone"}
	room2 := &rooms.Room{RoomId: 2, Zone: "TestZone", Exits: map[string]exit.RoomExit{"south": {RoomId: 1}}}
	t.Cleanup(rooms.SeedRoomsForTest(map[int]*rooms.Room{1: room1, 2: room2}, nil))

	caller := &mobs.Mob{InstanceId: 100, Character: characters.Character{RoomId: 1, Health: 50}}
	self := &mobs.Mob{InstanceId: 91101}
	self.Character.Name = "stepper"
	self.Character.RoomId = 2
	self.Character.Stamina = 100
	self.Character.StaminaMax.Value = 100
	self.Character.ActionPointsMax.Value = apMax
	self.Character.Conditions = conditions.New()
	t.Cleanup(mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{100: caller, 91101: self}))
	t.Cleanup(func() { events.DrainQueuedInputsForTest(91101) })
	return self
}

// Movement parity 4b: a single-step node quotes first and FAILS when the mob
// cannot pay, so a selector falls through to its next child.
func TestSingleStepNodesFailWhenTired(t *testing.T) {
	cases := []struct {
		name string
		run  func(ctx *EvalContext) Result
	}{
		{"go_to_caller_room", func(ctx *EvalContext) Result { return actGoToCallerRoom(nil, ctx) }},
		{"move", func(ctx *EvalContext) Result {
			return actMove(map[string]any{"direction": "south"}, ctx)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name+" tired", func(t *testing.T) {
			tiredStepWorld(t, 5)
			ctx := &EvalContext{InstanceId: 91101, Event: EventContext{MobId: 100}}
			require.Equal(t, Failure, tc.run(ctx))
			require.Empty(t, events.InspectQueuedInputForTest(91101, "go "))
		})
		t.Run(tc.name+" rested", func(t *testing.T) {
			tiredStepWorld(t, 200)
			ctx := &EvalContext{InstanceId: 91101, Event: EventContext{MobId: 100}}
			require.Equal(t, Success, tc.run(ctx))
			require.NotEmpty(t, events.InspectQueuedInputForTest(91101, "go "))
		})
	}
}
