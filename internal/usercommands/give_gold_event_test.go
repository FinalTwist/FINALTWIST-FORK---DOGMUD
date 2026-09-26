package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/require"
)

// Giving gold to a mob names the giver in a GoldGiven event: the mob's
// purse alone cannot say who it came from, and a bonded companion must
// thank (and charge) the right person, not whoever stands nearby.
func TestGiveGoldToMobNamesTheGiver(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)
	_, mobInstanceId := room.FindByName("skeleton")
	require.NotZero(t, mobInstanceId, "test mob 'skeleton' must be in the room")
	mob := mobs.GetInstance(mobInstanceId)
	require.NotNil(t, mob)

	events.DrainQueuedGoldGivenForTest(0)
	user.Character.Gold = 100
	before := mob.Character.Gold

	handled, err := Give("30 gold skeleton", user, room, 0)
	require.True(t, handled)
	require.NoError(t, err)
	require.Equal(t, before+30, mob.Character.Gold, "fixture: the gold changed hands")

	require.Equal(t, []events.GoldGiven{{UserId: user.UserId, MobInstanceId: mobInstanceId, Amount: 30}},
		events.DrainQueuedGoldGivenForTest(0))
}
