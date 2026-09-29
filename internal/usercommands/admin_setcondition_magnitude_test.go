package usercommands

import (
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// Lighting plan 5c final review, finding 3: `setcondition 129` (or 130, or any
// condition reading one of conditions.ScaledKinds from its magnitude) queued a
// plain add, so the record landed at magnitude 0 (reach 0, a useless heat
// sight) while the command reported success. It now applies such a condition
// at a new character's spell value for its kind (stat 100, skill 0, capped
// like the spell) and the condition's authored trigger count. A test for the
// held record's magnitude lives in the hooks apply path; this pins what the
// command queues.
func TestAdminSetCondition_ScaledConditionLandsAtNewCharacterSpellValue(t *testing.T) {
	const infraId, nightId, lightId, plainId = 9711, 9712, 9713, 9714
	restoreConditions := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		infraId: {ConditionId: infraId, Name: "Test Heat Sight", TriggerRate: "3 real minutes", TriggerCount: 1,
			Flags: []conditions.Flag{conditions.InfraredVision},
			Effects: map[conditions.EffectKind]conditions.EffectValue{
				conditions.EffectNightVisionStrength: {Literal: 12}, conditions.EffectInfraReach: {UsesMagnitude: true}}},
		nightId: {ConditionId: nightId, Name: "Test Night Sight", TriggerRate: "3 real minutes", TriggerCount: 1,
			Flags:   []conditions.Flag{conditions.NightVision},
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectNightVisionStrength: {UsesMagnitude: true}}},
		lightId: {ConditionId: lightId, Name: "Test Glow", RoundInterval: 1, TriggerCount: 40,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {UsesMagnitude: true}}},
		plainId: {ConditionId: plainId, Name: "Test Plain", RoundInterval: 1, TriggerCount: 5},
	})
	defer restoreConditions()

	admin := users.NewTestUser(9741, "magadmin", "Magadmin", uint64(9741))
	admin.Role = users.RoleAdmin
	admin.Character.RoomId = 1
	cleanupUsers := users.SeedUsersForTest(map[int]*users.UserRecord{9741: admin})
	defer cleanupUsers()
	room := &rooms.Room{RoomId: 1}
	cleanupRooms := rooms.SeedRoomsForTest(map[int]*rooms.Room{1: room}, nil)
	defer cleanupRooms()
	mob := &mobs.Mob{MobId: 1, InstanceId: 9742, HomeRoomId: 1,
		Character: characters.Character{Name: "Magratling", RoomId: 1, Health: 10, Conditions: conditions.New()}}
	cleanupMobs := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{9742: mob})
	defer cleanupMobs()
	room.AddMob(9742)
	defer room.RemoveMob(9742)

	// Triggers 0 on the event means the spec's own triggercount
	// (Conditions.AddConditionMagnitude), which is what a plain add applies.
	cfg := configs.GetLightingConfig()
	want := map[int]struct {
		mag      float64
		triggers int
	}{
		infraId: {min(cfg.InfraSpellBase+100/cfg.InfraSpellStatDivisor, float64(cfg.InfraReachCap)), 0},
		nightId: {cfg.NightVisionSpellBase + 100/cfg.NightVisionSpellStatDivisor, 0},
		lightId: {cfg.SpellStrengthBase + 100/cfg.SpellStrengthStatDivisor, 0},
	}

	for _, target := range []string{"", "magratling "} {
		for id, w := range want {
			events.DrainQueuedConditionsForTest(0)
			events.DrainQueuedMessagesForTest(admin.UserId)
			handled, err := SetCondition(target+strconv.Itoa(id), admin, room, 0)
			require.NoError(t, err)
			require.True(t, handled)
			queued := events.DrainQueuedConditionsForTest(0)
			require.Len(t, queued, 1, "target %q condition %d", target, id)
			require.Greater(t, w.mag, 0.0, "the expected value must be real or this row proves nothing")
			require.InDelta(t, w.mag, queued[0].Magnitude, 1e-9, "target %q condition %d magnitude", target, id)
			require.Equal(t, w.triggers, queued[0].Triggers, "target %q condition %d triggers", target, id)
			msgs := strings.Join(events.DrainQueuedMessagesForTest(admin.UserId), "\n")
			require.Contains(t, msgs, "applied to")
		}

		events.DrainQueuedConditionsForTest(0)
		_, err := SetCondition(target+strconv.Itoa(plainId), admin, room, 0)
		require.NoError(t, err)
		queued := events.DrainQueuedConditionsForTest(0)
		require.Len(t, queued, 1)
		require.Zero(t, queued[0].Magnitude, "an unscaled condition keeps its authored application")
		require.Zero(t, queued[0].Triggers)
	}
}
