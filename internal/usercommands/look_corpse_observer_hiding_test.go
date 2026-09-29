package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// File: look_corpse_observer_hiding_test.go
//
// A player corpse's display name is "<name> corpse" inside a user-corpse
// tag, which messaging.Anonymize does not strip (it is not an identity tag),
// so the observer line of `look <corpse>` must pass the dead player's name
// to SendTextVisualHidingNames or a shapes-only bystander reads it.
//
// Aliceia (user 1) looks at the corpse; Bobrick (user 2) watches. Both carry
// infrared in an unlit cave, so both are at shapes: the looker may still
// look (only SightNone is refused), and the watcher cannot tell who is who.
func TestLookCorpse_ShapesOnlyObserverDoesNotReadTheDeadPlayersName(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restoreCond := seedCraftHidingCondition()
	defer restoreCond()
	darkenCraftRoom(t, 1)

	looker, watcher := users.GetByUserId(1), users.GetByUserId(2)
	require.True(t, looker.Character.Conditions.AddCondition(craftHidingInfraredConditionId, true))
	require.True(t, watcher.Character.Conditions.AddCondition(craftHidingInfraredConditionId, true))

	room := rooms.LoadRoom(1)
	origCorpses := room.Corpses
	defer func() { room.Corpses = origCorpses }()
	corpse := rooms.Corpse{UserId: 777}
	corpse.Character.Name = "Deadric"
	room.Corpses = append([]rooms.Corpse{}, corpse)

	craftPlainLines(1)
	craftPlainLines(2)

	handled, err := Look("deadric corpse", looker, room, 0)
	require.True(t, handled)
	require.NoError(t, err)

	lookerLines, watcherLines := craftPlainLines(1), craftPlainLines(2)
	require.Equal(t, 1, craftCountContaining(lookerLines, "You look at the Deadric corpse."),
		"precondition: the looker must reach the corpse branch: got %v", lookerLines)
	require.Equal(t, 1, craftCountContaining(watcherLines, "is looking at the"),
		"the watcher must get the observer line: got %v", watcherLines)
	require.Equal(t, 0, craftCountContaining(watcherLines, "Deadric"),
		"a shapes-only observer must not read the dead player's name: got %v", watcherLines)
	require.Equal(t, []string{"A figure is looking at the corpse of a figure."}, watcherLines,
		"the hidden line must still read as English")
}
