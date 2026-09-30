package usercommands

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

var speechWrapperTag = regexp.MustCompile(`<[^>]*>`)

// speechWrapperHeard drains uid's queued lines the way its client prints
// them: minus those the deafen filter drops, tags stripped.
func speechWrapperHeard(uid int) []string {
	u := users.GetByUserId(uid)
	var out []string
	for _, m := range events.DrainQueuedMessageEventsForTest(uid) {
		if m.HiddenFromDeafened(u.Deafened) {
			continue
		}
		out = append(out, strings.TrimSpace(speechWrapperTag.ReplaceAllString(m.Text, "")))
	}
	return out
}

// speechWrapperScene is the fixture's room 1 lit exactly (no sky, Lamp 60):
// Aliceia (user 1) speaks, Bobrick (user 2) listens.
func speechWrapperScene(t *testing.T) (*users.UserRecord, *users.UserRecord, *rooms.Room) {
	t.Helper()
	alice, room := getTestUserAndRoom(t)
	room.SkyLight, room.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(60)
	bob := users.GetByUserId(2)
	require.Equal(t, messaging.SightFull, messaging.ParticipantSight(alice.Character, room))
	require.Equal(t, messaging.SightFull, messaging.ParticipantSight(bob.Character, room))
	events.DrainQueuedMessagesForTest(1)
	events.DrainQueuedMessagesForTest(2)
	return alice, bob, room
}

func blindForSpeechTest(t *testing.T, u *users.UserRecord) {
	t.Helper()
	u.Character.Perception = characters.New().Perception
	require.NoError(t, u.Character.Perception.TransitionTo(perception.Blinded, state.TransitionReason{Trigger: "test"}))
}

func TestSay_BlindedListenerInALitRoomHearsNoName(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	alice, bob, room := speechWrapperScene(t)
	blindForSpeechTest(t, bob)

	_, err := Say("hello there", alice, room, 0)
	require.NoError(t, err)
	require.Equal(t, []string{`Someone says, "hello there"`}, speechWrapperHeard(2))
	require.Equal(t, []string{`You say, "hello there"`}, speechWrapperHeard(1))
}

func TestSay_DeafenedListenerIsSpared(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	alice, bob, room := speechWrapperScene(t)
	bob.Deafened = true

	_, err := Say("hello there", alice, room, 0)
	require.NoError(t, err)
	require.Empty(t, speechWrapperHeard(2))
}

func TestShout_BlindedListenerHearsTheWordsNotTheName(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	alice, bob, room := speechWrapperScene(t)
	blindForSpeechTest(t, bob)

	_, err := Shout("help", alice, room, 0)
	require.NoError(t, err)
	require.Equal(t, []string{`Someone shouts, "HELP"`}, speechWrapperHeard(2))
	require.Equal(t, []string{`You shout, "HELP"`}, speechWrapperHeard(1))
}

func TestEmote_FreeFormFollowsSightAndDeafen(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	alice, bob, room := speechWrapperScene(t)

	_, err := Emote("waves.", alice, room, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"Aliceia waves."}, speechWrapperHeard(2))
	require.Equal(t, []string{"You Emote: Aliceia waves."}, speechWrapperHeard(1))

	bob.Deafened = true
	_, err = Emote("waves.", alice, room, 0)
	require.NoError(t, err)
	require.Empty(t, speechWrapperHeard(2), "free text is chatter: the deafened are spared it")
	bob.Deafened = false

	blindForSpeechTest(t, bob)
	_, err = Emote("waves.", alice, room, 0)
	require.NoError(t, err)
	require.Empty(t, speechWrapperHeard(2), "an emote is seen, not heard")
}

func TestEmote_AtFormSkipsOnlyTheSelfLine(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	alice, _, room := speechWrapperScene(t)

	_, err := Emote("@waves silently.", alice, room, 0)
	require.NoError(t, err)
	require.Empty(t, speechWrapperHeard(1))
	require.Equal(t, []string{"Aliceia waves silently."}, speechWrapperHeard(2))
}

func TestEmote_AliasAndEmptyReachTheDeafened(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	alice, bob, room := speechWrapperScene(t)
	bob.Deafened = true

	_, err := Emote("beam", alice, room, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"Aliceia beams with pride."}, speechWrapperHeard(2))

	_, err = Emote("", alice, room, 0)
	require.NoError(t, err)
	require.Equal(t, []string{"Aliceia emotes."}, speechWrapperHeard(2))
}
