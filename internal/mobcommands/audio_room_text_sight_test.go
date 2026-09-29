package mobcommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// Lighting plan 5c final review, finding 1: sendAudioRoomText gave the named
// line to any listener holding the nightvision FLAG in a dark room. Since
// plan 2 night vision shifts a window rather than granting sight, and below
// LightBlindBelow no shift reaches the faces band, so a nightvision holder in
// a dark room makes out shapes at best and must hear the anonymous line. The
// named line now goes to a listener whose sight is SightFull.
func TestSendAudioRoomText_NamedLineFollowsSight(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restoreBiomes := rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"cave": {BiomeId: "cave", Name: "Cave", Symbol: ".", SkyLight: rooms.SkyLightPtr(0.0), MovementCost: 1},
	})
	defer restoreBiomes()
	const nightId = 9631
	restoreConditions := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		nightId: {ConditionId: nightId, Name: "Test Night Sight", RoundInterval: 1, TriggerCount: 10,
			Flags:   []conditions.Flag{conditions.NightVision},
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectNightVisionStrength: {Literal: 24}}},
	})
	defer restoreConditions()

	mob := mobs.GetInstance(100)
	require.NotNil(t, mob)

	for _, lamp := range []int{0, 10, 24} {
		room := rooms.LoadRoom(2)
		require.NotNil(t, room)
		room.Biome = "cave"
		room.Lamp = rooms.LampPtr(lamp)
		u := users.GetByUserId(1)
		require.NotNil(t, u)
		rooms.LoadRoom(1).RemovePlayer(1)
		u.Character.RoomId = 2
		room.AddPlayer(1)
		if !u.Character.HasCondition(nightId) {
			require.True(t, u.Character.Conditions.AddCondition(nightId, true))
		}
		require.NotEqual(t, messaging.SightFull, messaging.ParticipantSight(u.Character, room), "light %d", lamp)

		events.DrainQueuedMessagesForTest(1)
		sendAudioRoomText(room, mob, messaging.CategorySpeech, "Someone growls.", "Skeleton growls.")
		got := strings.Join(events.DrainQueuedMessagesForTest(1), "\n")
		require.Contains(t, got, "Someone growls.", "light %d", lamp)
		require.NotContains(t, got, "Skeleton", "light %d: a nightvision holder in a dark room cannot tell who", lamp)
	}
}
