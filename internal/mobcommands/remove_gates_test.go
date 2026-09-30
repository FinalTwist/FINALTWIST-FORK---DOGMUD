package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A busy mob keeps its gear on, and a cursed piece stays on against `remove`
// and `remove all` (rulings 4 and 5).
func TestMobRemove_BusyAndCursedGates(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	mob, room := getTestMobAndRoom(t)
	mob.Character.Equipment.Ring = items.Item{ItemId: 96401, Spec: &items.ItemSpec{ItemId: 96401, Name: "hexed ring", Type: items.Ring, Subtype: items.Wearable, Cursed: true}}
	mob.Character.Equipment.Head = items.Item{ItemId: 96402, Spec: &items.ItemSpec{ItemId: 96402, Name: "cap", Type: items.Head, Subtype: items.Wearable}}

	Remove("hexed ring", mob, room)
	assert.Equal(t, 96401, mob.Character.Equipment.Ring.ItemId)

	mob.Character.Activity = activity.NewMachine()
	require.NoError(t, mob.Character.Activity.TransitionToCrafting(
		activity.CraftingData{RecipeId: "test", RoundsTotal: 3},
		state.TransitionReason{Trigger: activity.TriggerCraftBegin}))
	Remove("cap", mob, room)
	assert.Equal(t, 96402, mob.Character.Equipment.Head.ItemId, "busy")

	mob.Character.Activity = nil
	Remove("all", mob, room)
	assert.Equal(t, 96401, mob.Character.Equipment.Ring.ItemId)
	assert.Equal(t, 0, mob.Character.Equipment.Head.ItemId)
}
