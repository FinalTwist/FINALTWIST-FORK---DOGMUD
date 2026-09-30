package mobcommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

func hideForLook(t *testing.T, u *users.UserRecord) {
	t.Helper()
	reason := state.TransitionReason{Trigger: "look_sight_gates_test"}
	require.NoError(t, u.Character.Awareness.TransitionToConcealing(awareness.ConcealingData{}, reason))
	u.Character.Awareness.ResolveConcealment(true, reason)
	require.True(t, u.Character.IsHidden())
}

// A mob cannot name a hidden player: no "is looking at you", no room line
// (closes spec L6).
func TestMobLook_CannotNameAHiddenPlayer(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	mob, room := getTestMobAndRoom(t)
	room.Lamp = rooms.LampPtr(90)
	alice, bob := users.GetByUserId(1), users.GetByUserId(2)
	hideForLook(t, alice)
	events.DrainQueuedMessagesForTest(1)
	events.DrainQueuedMessagesForTest(2)
	Look("aliceia", mob, room)
	require.NotContains(t, strings.Join(events.DrainQueuedMessagesForTest(1), "\n"), "is looking at you")
	require.NotContains(t, strings.Join(events.DrainQueuedMessagesForTest(2), "\n"), alice.Character.Name)
	_ = bob
}

// In the dark a mob's look is silent.
func TestMobLook_DarkIsSilent(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	mob, room := getTestMobAndRoom(t)
	room.SkyLight, room.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(0)
	events.DrainQueuedMessagesForTest(1)
	Look("", mob, room)
	Look("aliceia", mob, room)
	Look("north", mob, room)
	require.Empty(t, events.DrainQueuedMessagesForTest(1))
}
