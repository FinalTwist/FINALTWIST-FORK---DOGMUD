package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/narration"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// File: position_submission_hiding_test.go
//
// M4d PR 3 Task 4: sendSubmissionTriple. position_control.yaml's submission
// templates name the OTHER grappler inside the actor and actee lines too,
// not only the observer line (e.g. submission.opening.armbar's actor line
// reads "You isolate {actee}'s arm...", naming the recipient in the
// attacker's own personal line). The personal lines went out on
// UserRecord.SendText (audio channel, no sight gate or anonymize at all) and
// the room line on Room.SendTextVisual (same tag-only leak class as the
// stamina warning's room line).
//
// {actor}/{actee} substitute Character.Name, the bare field, never
// GetCharacterName's ansi-tagged form, so today's shipped content is a REAL
// leak, not merely incidentally safe by authoring discipline. This test uses
// the shipped submission.opening.armbar triple verbatim (substituted), not
// an authored bare-name stand-in.

// TestSendSubmissionTriple_HidesEachSideFromReadersWhoCannotSeeThem proves
// and pins the fix for sendSubmissionTriple's three audiences, using the
// shipped armbar-opening triple verbatim.
//
// Pre-fix this fails on all three assertions: the attacker's and recipient's
// personal lines go out on UserRecord.SendText, which bypasses the sight
// gate and the anonymizer entirely, and the room line goes out on
// Room.SendTextVisual with no names, same leak class as the stamina case.
func TestSendSubmissionTriple_HidesEachSideFromReadersWhoCannotSeeThem(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	darken(t, 1)

	attacker := users.GetByUserId(1)  // Aliceia -- attempter
	recipient := users.GetByUserId(2) // Bobrick -- recipient
	watcher := users.NewTestUser(3, "cara", "Carrow", 1003)
	watcher.Character.RoomId = 1
	restoreUsers := users.SeedUsersForTest(map[int]*users.UserRecord{
		1: attacker, 2: recipient, 3: watcher,
	})
	defer restoreUsers()
	room := rooms.LoadRoom(1)
	room.AddPlayer(3)

	// Recipient and the third-party watcher have infrared (shapes-only in the
	// dark); the attacker has none, so she is fully blind here. That gives
	// each of the three audiences a DIFFERENT sight of the room, so a test
	// that passes cannot be passing by accident of one shared sight value.
	require.True(t, recipient.Character.Conditions.AddCondition(heatEyesConditionId, true))
	require.True(t, watcher.Character.Conditions.AddCondition(heatEyesConditionId, true))
	drainPlain(1)
	drainPlain(2)
	drainPlain(3)

	// submission.opening.armbar, copied verbatim from position_control.yaml.
	tmpl := submissionMsgTriple{
		Attacker: "You isolate {actee}'s arm and crank — they feel the joint go past its limit!",
		Target:   "{actor} isolates your arm and cranks — pain shoots through the elbow!",
		Room:     "{actor} isolates {actee}'s arm in a brutal armbar.",
	}
	subs := map[string]string{
		narration.TokenActor: attacker.Character.Name,
		narration.TokenActee: recipient.Character.Name,
	}

	sendSubmissionTriple(attacker.Character, recipient.Character, tmpl, subs)

	atkLines := drainPlain(1)
	require.Equal(t, 0, countContaining(atkLines, "Bobrick"),
		"the attacker (blind here) must not read the recipient's bare name in her own personal line: got %v", atkLines)
	require.Equal(t, 1, countContaining(atkLines, "isolate something's arm"),
		"the attacker should read the hidden form of her own line: got %v", atkLines)

	tgtLines := drainPlain(2)
	require.Equal(t, 0, countContaining(tgtLines, "Aliceia"),
		"the recipient (shapes-only here) must not read the attacker's bare name in her own personal line: got %v", tgtLines)
	require.Equal(t, 1, countContaining(tgtLines, "A figure isolates your arm"),
		"the recipient should read the hidden form of her own line: got %v", tgtLines)

	roomLines := drainPlain(3)
	require.Equal(t, 0, countContaining(roomLines, "Aliceia"),
		"the shapes-only third party must not read the attacker's bare name in the room line: got %v", roomLines)
	require.Equal(t, 0, countContaining(roomLines, "Bobrick"),
		"the shapes-only third party must not read the recipient's bare name in the room line: got %v", roomLines)
}
