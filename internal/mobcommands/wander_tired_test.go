package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/require"
)

// Movement parity 4b: a tired wanderer does not count a wander it cannot
// walk, so it is not sent home early for steps it never took. The rested
// control proves the fixture reaches the counting branch at all (Wander only
// counts a step into the mob's own zone).
func TestWander_TiredMobDoesNotCount(t *testing.T) {
	for _, tc := range []struct {
		name      string
		apMax     int
		wantCount int
	}{
		{"rested control counts the step", 200, 1},
		{"tired mob does not", 5, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := seedAllRegistries()
			defer cleanup()
			mob, room := getTestMobAndRoom(t)

			north, ok := room.Exits["north"]
			require.True(t, ok, "fixture premise: room 1 has a north exit")
			dest := rooms.LoadRoom(north.RoomId)
			require.NotNil(t, dest)
			origExits := room.Exits
			room.Exits = map[string]exit.RoomExit{"north": north}
			t.Cleanup(func() { room.Exits = origExits })
			mob.Character.Zone = dest.Zone

			mob.MaxWander = -1
			mob.WanderCount = 0
			mob.Character.ActionPointsMax.Value = tc.apMax
			mob.Character.ActionPointsSettled = false
			mob.Character.StaminaMax.Value = 100
			mob.Character.Stamina = 100

			handled, err := Wander("", mob, room)
			require.True(t, handled)
			require.NoError(t, err)
			require.Equal(t, tc.wantCount, mob.WanderCount)
		})
	}
}
