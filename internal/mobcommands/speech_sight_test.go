package mobcommands

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
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
