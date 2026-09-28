package mobcommands

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/require"
)

// Lighting plan 5c final review, finding 2: a mob's drink applied a potion's
// conditions with plain AddCondition, so a magnitude-scaled condition landed
// at magnitude 0. The AI companion and the survival planner drink through
// this path, so a companion drinking the Pitsense Tincture got reach 0. It
// now applies through items.PotionMagnitudeApplication at durationMult 1,
// the same helper the player's drink uses.
func TestMobDrink_AppliesPotionMagnitude(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	const heatId, plainId, tinctureId, brewId = 9811, 9812, 39811, 39812
	restoreConditions := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		heatId: {ConditionId: heatId, Name: "Test Tincture", RoundInterval: 1, TriggerCount: 400,
			Flags: []conditions.Flag{conditions.InfraredVision},
			Effects: map[conditions.EffectKind]conditions.EffectValue{
				conditions.EffectNightVisionStrength: {Literal: 12}, conditions.EffectInfraReach: {UsesMagnitude: true}}},
		plainId: {ConditionId: plainId, Name: "Test Brew", RoundInterval: 1, TriggerCount: 50},
	})
	defer restoreConditions()
	restoreItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		tinctureId: {ItemId: tinctureId, Name: "test tincture", Type: items.Potion, Subtype: items.Drinkable,
			Uses: 1, Magnitude: 20, ConditionIds: []int{heatId}},
		brewId: {ItemId: brewId, Name: "test brew", Type: items.Potion, Subtype: items.Drinkable,
			Uses: 1, ConditionIds: []int{plainId}},
	})
	defer restoreItems()

	mob := mobs.GetInstance(100)
	require.NotNil(t, mob)
	room := rooms.LoadRoom(1)
	require.NotNil(t, room)

	for _, id := range []int{tinctureId, brewId} {
		require.True(t, mob.Character.StoreItem(items.New(id)))
	}

	events.DrainQueuedConditionsForTest(0)
	_, err := Drink("tincture", mob, room)
	require.NoError(t, err)
	_, err = Drink("brew", mob, room)
	require.NoError(t, err)

	byId := map[int]events.Condition{}
	for _, c := range events.DrainQueuedConditionsForTest(0) {
		if c.MobInstanceId == mob.InstanceId {
			byId[c.ConditionId] = c
		}
	}
	heat, ok := byId[heatId]
	require.True(t, ok, "the tincture's condition must be queued")
	require.InDelta(t, 20.0, heat.Magnitude, 1e-9, "the tincture's reach must ride on the event, not 0")
	require.Equal(t, 400, heat.Triggers)

	brew, ok := byId[plainId]
	require.True(t, ok, "an unscaled potion still applies")
	require.True(t, math.Abs(brew.Magnitude) < 1e-9, "an unscaled potion carries no magnitude")
	require.Zero(t, brew.Triggers, "an unscaled potion keeps its authored duration")
}
