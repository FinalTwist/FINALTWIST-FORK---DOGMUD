package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

const (
	hoodTestAdjustableCond = 9751
	hoodTestFixedCond      = 9752
	hoodTestLanternItem    = 999960
	hoodTestTorchItem      = 999961
)

// hoodFixture seeds an adjustable lantern and a fixed (non-adjustable) light,
// a species for Wear to dereference, and returns user 1 standing in room 2.
func hoodFixture(t *testing.T) (*users.UserRecord, *rooms.Room) {
	t.Helper()
	t.Cleanup(seedAllRegistries())
	t.Cleanup(species.SeedSpeciesForTest(map[int]*species.Species{
		0: {SpeciesId: 0, Name: "human", Size: species.Medium},
	}))
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		hoodTestAdjustableCond: {ConditionId: hoodTestAdjustableCond, Name: "Test Hooded Lantern", Secret: true, TriggerCount: 1, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {Literal: 54}},
			Flags:   []conditions.Flag{conditions.Adjustable}},
		hoodTestFixedCond: {ConditionId: hoodTestFixedCond, Name: "Test Fixed Light", Secret: true, TriggerCount: 1, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {Literal: 56}}},
	}))
	t.Cleanup(items.SeedItemsForTest(map[int]*items.ItemSpec{
		hoodTestLanternItem: {ItemId: hoodTestLanternItem, Name: "test hooded lantern", Type: items.Light, Subtype: items.Wearable, WornConditionIds: []int{hoodTestAdjustableCond}},
		hoodTestTorchItem:   {ItemId: hoodTestTorchItem, Name: "test fixed light", Type: items.Light, Subtype: items.Wearable, WornConditionIds: []int{hoodTestFixedCond}},
	}))

	user := users.GetByUserId(1)
	require.NotNil(t, user)
	user.Character.SpeciesId = 0
	user.Character.Stats.Strength.ValueAdj = 100

	room := rooms.LoadRoom(2)
	require.NotNil(t, room)
	user.Character.RoomId = 2
	room.AddPlayer(user.UserId)
	return user, room
}

func hoodTestText(userId int) string {
	var b strings.Builder
	for _, line := range events.DrainQueuedMessagesForTest(userId) {
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func TestHoodAndUnhood(t *testing.T) {
	user, room := hoodFixture(t)
	_, ok, why := user.Character.Wear(items.New(hoodTestLanternItem))
	require.True(t, ok, why)
	rec := user.Character.Conditions.LightSources()[0]
	rec.SetLightOutput(30)

	_, err := Hood("", user, room, 0)
	require.NoError(t, err)
	require.True(t, rec.Hooded, "hood did not close the hood")
	require.False(t, user.Character.EmitsLight(), "a hooded lantern still sheds light")

	_, err = Unhood("", user, room, 0)
	require.NoError(t, err)
	require.False(t, rec.Hooded)
	require.Equal(t, conditions.LightFull, rec.LightTrim, "unhood must return the lantern to full strength")
	require.True(t, user.Character.EmitsLight(), "an unhooded lantern sheds no light")

	// A nil room is tolerated.
	_, err = Hood("", user, nil, 0)
	require.NoError(t, err)
	require.True(t, rec.Hooded)
	_, err = Unhood("", user, nil, 0)
	require.NoError(t, err)
	require.False(t, rec.Hooded)
}

func TestHood_NoLightWornRefuses(t *testing.T) {
	user, room := hoodFixture(t)
	require.Less(t, user.Character.Equipment.Light.ItemId, 1, "fixture must start with an empty light slot")
	events.DrainQueuedMessagesForTest(user.UserId)

	handled, err := Hood("", user, room, 0)
	require.True(t, handled)
	require.NoError(t, err)
	require.Contains(t, hoodTestText(user.UserId), "no hooded light")

	handled, err = Unhood("", user, room, 0)
	require.True(t, handled)
	require.NoError(t, err)
	require.Contains(t, hoodTestText(user.UserId), "no hooded light")
}

func TestHood_NonAdjustableLightRefuses(t *testing.T) {
	user, room := hoodFixture(t)
	_, ok, why := user.Character.Wear(items.New(hoodTestTorchItem))
	require.True(t, ok, why)
	recs := user.Character.Conditions.LightSources()
	require.Len(t, recs, 1)
	rec := recs[0]
	require.True(t, user.Character.EmitsLight())
	events.DrainQueuedMessagesForTest(user.UserId)

	_, err := Hood("", user, room, 0)
	require.NoError(t, err)
	require.Contains(t, hoodTestText(user.UserId), "no hooded light")
	require.False(t, rec.Hooded, "a non-adjustable light was hooded")
	require.True(t, user.Character.EmitsLight(), "refusing to hood must leave the light shining")
}

func TestHood_TwiceSaysAlreadyHooded(t *testing.T) {
	user, room := hoodFixture(t)
	_, ok, why := user.Character.Wear(items.New(hoodTestLanternItem))
	require.True(t, ok, why)
	rec := user.Character.Conditions.LightSources()[0]

	_, err := Hood("", user, room, 0)
	require.NoError(t, err)
	require.True(t, rec.Hooded)
	events.DrainQueuedMessagesForTest(user.UserId)

	_, err = Hood("", user, room, 0)
	require.NoError(t, err)
	require.Contains(t, hoodTestText(user.UserId), "already hooded")
	require.True(t, rec.Hooded)

	_, err = Unhood("", user, room, 0)
	require.NoError(t, err)
	events.DrainQueuedMessagesForTest(user.UserId)
	_, err = Unhood("", user, room, 0)
	require.NoError(t, err)
	require.Contains(t, hoodTestText(user.UserId), "already open")
}
