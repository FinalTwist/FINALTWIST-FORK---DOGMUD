package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// File: combat_blind_notice_cost_test.go
//
// Lighting plan 5c playtest finding 6. The blind-combat notice says "your
// attacks and defense are weaker", and it fired every round for a Heat Sight
// caster whose sight multiplier was about 0.97, on the verdict alone (not
// SightFull). At infra reach 50 the multiplier in a dark room is exactly 1.0
// and the line would be false. The notice now also needs the player's real
// messaging.SightMult in the room to be below 1.0.

const blindCostDeepHeatConditionId = 9651

func blindCostRound(t *testing.T, condition int) (int, float64, messaging.SightDecision) {
	t.Helper()
	cleanup := seedAllRegistries()
	t.Cleanup(cleanup)
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		blindCostDeepHeatConditionId: {ConditionId: blindCostDeepHeatConditionId, Name: "Test Deep Heat",
			Flags:   []conditions.Flag{conditions.InfraredVision},
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectInfraReach: {Literal: 50}}},
	}))
	roundBlindCombatants = map[int]bool{}

	darken(t, 2)
	room2 := rooms.LoadRoom(2)
	u1 := users.GetByUserId(1)
	require.NotNil(t, u1)
	u1.Character.RoomId = 2
	rooms.LoadRoom(1).RemovePlayer(1)
	room2.AddPlayer(1)
	if condition != 0 {
		require.True(t, u1.Character.Conditions.AddCondition(condition, true))
	}

	mob := mobs.GetInstance(100)
	require.NotNil(t, mob)
	atk := actions.NewUserActorInRoom(u1, room2)
	def := actions.NewMobActorInRoom(mob, room2)

	captured, capCleanup := captureMessages(t)
	t.Cleanup(capCleanup)

	dispatchCritAndMessaging(atk, def, vbLandingResult())
	flushBlindCombatNotices()
	events.ProcessEvents()

	return countContaining(textsForUser(*captured, 1), blindNoticeNeedle),
		messaging.SightMult(u1.Character, room2),
		messaging.ParticipantSight(u1.Character, room2)
}

func TestBlindCombatNotice_FollowsTheRealCost(t *testing.T) {
	t.Run("infra reach 50 in the dark pays nothing and is not told it does", func(t *testing.T) {
		notices, mult, sight := blindCostRound(t, blindCostDeepHeatConditionId)
		require.Equal(t, messaging.SightShapes, sight, "precondition: heat sight reads shapes in the dark")
		require.Equal(t, 1.0, mult, "precondition: reach 50 at light 0 must cost nothing")
		assert.Equal(t, 0, notices, "a combatant whose sight costs nothing must not read the blind notice")
	})
	t.Run("a normal eye in the dark pays and is told", func(t *testing.T) {
		notices, mult, sight := blindCostRound(t, 0)
		require.Equal(t, messaging.SightNone, sight)
		require.Less(t, mult, 1.0, "precondition: a normal eye in the dark pays the ramp")
		assert.Equal(t, 1, notices)
	})
}
