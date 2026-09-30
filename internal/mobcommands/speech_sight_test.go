package mobcommands

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

var mobSpeechTag = regexp.MustCompile(`<[^>]*>`)

// mobSpeechHeard drains uid's queued lines the way its client prints them:
// minus those the deafen filter drops, tags stripped.
func mobSpeechHeard(uid int) []string {
	u := users.GetByUserId(uid)
	var out []string
	for _, m := range events.DrainQueuedMessageEventsForTest(uid) {
		if m.HiddenFromDeafened(u.Deafened) {
			continue
		}
		out = append(out, strings.TrimSpace(mobSpeechTag.ReplaceAllString(m.Text, "")))
	}
	return out
}

// mobSpeechRoom lights room 1 exactly (no sky, Lamp 60) and blinds Bobrick
// (user 2) with the perception machine; Aliceia (user 1) sees clearly. The
// speaker is mob 100, "Skeleton".
func mobSpeechRoom(t *testing.T) (*mobs.Mob, *rooms.Room) {
	t.Helper()
	mob, room := getTestMobAndRoom(t)
	room.SkyLight, room.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(60)
	bob := users.GetByUserId(2)
	bob.Character.Perception = characters.New().Perception
	require.NoError(t, bob.Character.Perception.TransitionTo(perception.Blinded, state.TransitionReason{Trigger: "test"}))
	require.Equal(t, messaging.SightFull, messaging.ParticipantSight(users.GetByUserId(1).Character, room))
	require.Equal(t, messaging.SightNone, messaging.ParticipantSight(bob.Character, room))
	events.DrainQueuedMessagesForTest(1)
	events.DrainQueuedMessagesForTest(2)
	return mob, room
}

// The defect sendAudioRoomText's lit-room shortcut carried: a blinded
// listener in a lit room was told the speaker's name.
func TestMobSay_BlindedListenerInALitRoomHearsNoName(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := mobSpeechRoom(t)

	_, err := Say("hello there", mob, room)
	require.NoError(t, err)
	require.Equal(t, []string{`Skeleton says, "hello there"`}, mobSpeechHeard(1))
	require.Equal(t, []string{`Someone says, "hello there"`}, mobSpeechHeard(2))
}

// Ruling 6: NPC speech is authored content, so a deafened player hears it.
func TestMobSay_ReachesADeafenedPlayer(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := mobSpeechRoom(t)
	users.GetByUserId(1).Deafened = true

	_, err := Say("hello there", mob, room)
	require.NoError(t, err)
	require.Equal(t, []string{`Skeleton says, "hello there"`}, mobSpeechHeard(1))
}

func TestMobShout_BlindedListenerHearsTheWordsNotTheName(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := mobSpeechRoom(t)

	_, err := Shout("intruders!", mob, room)
	require.NoError(t, err)
	require.Equal(t, []string{`Skeleton shouts, "intruders!"`}, mobSpeechHeard(1))
	require.Equal(t, []string{`Someone shouts, "intruders!"`}, mobSpeechHeard(2))
}

func TestMobEmote_SeenNotHeard(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := mobSpeechRoom(t)
	users.GetByUserId(1).Deafened = true

	_, err := Emote("growls.", mob, room)
	require.NoError(t, err)
	require.Equal(t, []string{"Skeleton growls."}, mobSpeechHeard(1), "a mob emote reaches the deafened (ruling 6)")
	require.Empty(t, mobSpeechHeard(2), "the blinded see nothing")
}

// The lit-room shortcut named a howling mob to a blinded listener.
func TestMobHowl_FumbleHeardWithoutAName(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := mobSpeechRoom(t)
	mob.Character.SetAggro(1, 0, characters.DefaultAttack)
	original := executeTauntAction
	executeTauntAction = func(actions.Actor) actions.TauntResult {
		return actions.TauntResult{Executed: true, Fumble: true}
	}
	t.Cleanup(func() { executeTauntAction = original })

	_, err := Howl("", mob, room)
	require.NoError(t, err)
	require.Equal(t, []string{"Skeleton lets out a pitiful howl that trails off weakly."}, mobSpeechHeard(1))
	require.Equal(t, []string{"Something lets out a pitiful howl that trails off weakly."}, mobSpeechHeard(2))
}

// Ported from audio_room_text_sight_test.go (lighting plan 5c finding 1): a
// night-vision holder in a dark room cannot tell who is speaking.
func TestMobSay_NightVisionInTheDarkHearsNoName(t *testing.T) {
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
		rooms.LoadRoom(1).RemovePlayer(1)
		u.Character.RoomId = 2
		room.AddPlayer(1)
		if !u.Character.HasCondition(nightId) {
			require.True(t, u.Character.Conditions.AddCondition(nightId, true))
		}
		require.NotEqual(t, messaging.SightFull, messaging.ParticipantSight(u.Character, room), "light %d", lamp)

		events.DrainQueuedMessagesForTest(1)
		_, err := Say("growls", mob, room)
		require.NoError(t, err)
		got := strings.Join(mobSpeechHeard(1), "\n")
		require.Contains(t, got, `says, "growls"`, "light %d: the words arrive", lamp)
		require.NotContains(t, got, "Skeleton", "light %d: a nightvision holder in a dark room cannot tell who", lamp)
	}
}
