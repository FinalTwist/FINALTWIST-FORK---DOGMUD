package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// File: look_exit_visibility_test.go
//
// Graded lighting arc, plan 1 Task 5: look.go now reads two different
// thresholds off the same LightLevel() value. LightBlindBelow gates seeing
// the room at all; LightExitsAbove separately gates seeing THROUGH an exit.
// The parity golden (lighting_parity_golden_test.go) only covers sight
// tiers, not exit-peering, so this is the one automated check for the
// look.go:258 site.
//
// TestLookExit_RoomOnlyLightBlocksExitPeering pins the band the old model
// could not name explicitly: light enough to see the room (LightRoomOnly,
// 60, which is >= LightBlindBelow's 25) but not enough to see down an exit
// (60 < LightExitsAbove's 65). The old model expressed this exact band as
// visibility 1: enough to pass the "can't see anything" gate but not the
// "too dark to see anything in that direction" gate. Getting a room to land
// on LightRoomOnly requires a dark biome (which alone clamps to LightDark)
// plus a light source in the room (which then adds exactly one step); see
// internal/rooms/lighting_test.go's TestLightLevelMapsTheOldModel and
// internal/rooms/lighting.go's legacyVisibility for why that combination is
// the only cheap way to reach the middle constant.
func TestLookExit_RoomOnlyLightBlocksExitPeering(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	const torchConditionId = 9101
	restoreConditions := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		torchConditionId: {
			ConditionId:   torchConditionId,
			Name:          "Test Torch",
			RoundInterval: 1,
			TriggerCount:  1,
			Flags:         []conditions.Flag{conditions.EmitsLight},
		},
	})
	defer restoreConditions()

	user := users.GetByUserId(1)
	require.NotNil(t, user)

	room := rooms.LoadRoom(2)
	require.NotNil(t, room)
	room.Biome = "cave"

	user.Character.RoomId = 2
	room.AddPlayer(user.UserId)
	require.NoError(t, user.Character.AddCondition(torchConditionId, false))

	require.Equal(t, rooms.LightRoomOnly, room.LightLevel(),
		"fixture must land on the room-only band (dark biome + a light source), or this test proves nothing")

	events.DrainQueuedMessagesForTest(user.UserId)

	handled, err := Look("south", user, room, 0)
	require.True(t, handled)
	require.NoError(t, err)

	lines := events.DrainQueuedMessagesForTest(user.UserId)
	require.NotEmpty(t, lines, "look must still respond, since the room itself is lit enough to see")
	joined := strings.Join(lines, "\n")
	require.Contains(t, joined, "too dark to see anything in that direction",
		"room-only light must still block seeing THROUGH an exit")
}

// TestLookExit_FullLightAllowsExitPeering is the sibling positive case: a
// room lit up to LightFull (>= LightExitsAbove) must NOT be refused. Without
// this, a defect that always blocks exit-peering would still pass the test
// above (it only checks the blocked case).
func TestLookExit_FullLightAllowsExitPeering(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)
	require.Equal(t, rooms.LightFull, room.LightLevel(),
		"fixture room must be at full light, or this test proves nothing")

	events.DrainQueuedMessagesForTest(user.UserId)

	handled, err := Look("north", user, room, 0)
	require.True(t, handled)
	require.NoError(t, err)

	lines := events.DrainQueuedMessagesForTest(user.UserId)
	joined := strings.Join(lines, "\n")
	require.NotContains(t, joined, "too dark to see anything in that direction",
		"full light must not block seeing THROUGH an exit")
}
