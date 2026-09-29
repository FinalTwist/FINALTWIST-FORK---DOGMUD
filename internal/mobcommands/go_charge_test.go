package mobcommands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Movement parity 4b, owner ruling 2: a mob pays the player's step price.
func TestMobGo_PaysForTheStep(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := getTestMobAndRoom(t)
	require.Contains(t, room.Exits, "north", "fixture premise: room 1 has a north exit")

	mob.Character.ActionPointsMax.Value = 200
	mob.Character.ActionPointsSettled = false // settle full on the charge
	mob.Character.StaminaMax.Value = 100
	mob.Character.Stamina = 100

	handled, err := Go("north", mob, room)
	require.True(t, handled)
	require.NoError(t, err)
	require.NotEqual(t, 1, mob.Character.RoomId, "the mob walked north")
	require.Equal(t, 190, mob.Character.ActionPoints, "the step cost 10 action points")

	mob.Character.RoomId = 1
	room.AddMob(mob.InstanceId)
}

// A mob that cannot pay stays put, silently.
func TestMobGo_TiredMobStays(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := getTestMobAndRoom(t)

	mob.Character.ActionPointsMax.Value = 5 // settles full at 5, below a 10-point step
	mob.Character.ActionPointsSettled = false
	mob.Character.StaminaMax.Value = 100
	mob.Character.Stamina = 100

	handled, err := Go("north", mob, room)
	require.True(t, handled)
	require.NoError(t, err)
	require.Equal(t, 1, mob.Character.RoomId, "a tired mob does not move")
}
