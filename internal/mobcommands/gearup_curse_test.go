package mobcommands

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gearRing(id int, name string, cursed bool, str int) items.Item {
	return items.Item{ItemId: id, Spec: &items.ItemSpec{ItemId: id, Name: name, Type: items.Ring, Subtype: items.Wearable,
		Cursed: cursed, StatMods: map[string]int{"strength": str}}}
}

// A mob handed a better ring with a cursed first ring and a plain second
// wears it over the second; with both cursed it tries nothing and says
// nothing (spec "Companion", 5a).
func TestGearup_SkipsACursedRing(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	mob, room := getTestMobAndRoom(t)
	mob.Character.Equipment.Ring = gearRing(96101, "hexed band", true, 0)
	mob.Character.Equipment.Ring2 = gearRing(96102, "plain band", false, 0)
	gift := gearRing(96103, "gold band", false, 10)
	require.True(t, mob.Character.StoreItem(gift))

	events.DrainQueuedInputsForTest(mob.InstanceId)
	_, err := Gearup(fmt.Sprintf("!%d", gift.ItemId), mob, room)
	require.NoError(t, err)
	assert.Contains(t, events.DrainQueuedInputsForTest(mob.InstanceId), fmt.Sprintf("wear !%d", gift.ItemId))

	_, err = Equip(fmt.Sprintf("!%d", gift.ItemId), mob, room)
	require.NoError(t, err)
	assert.Equal(t, 96103, mob.Character.Equipment.Ring2.ItemId, "the gift goes over the plain ring")
	assert.Equal(t, 96101, mob.Character.Equipment.Ring.ItemId, "the cursed ring stays on")

	mob.Character.Equipment.Ring2 = gearRing(96104, "second hex", true, 0)
	gift2 := gearRing(96105, "silver band", false, 10)
	require.True(t, mob.Character.StoreItem(gift2))
	events.DrainQueuedInputsForTest(mob.InstanceId)
	_, err = Gearup(fmt.Sprintf("!%d", gift2.ItemId), mob, room)
	require.NoError(t, err)
	assert.Empty(t, events.DrainQueuedInputsForTest(mob.InstanceId), "no upgrade when every ring is cursed")
}
