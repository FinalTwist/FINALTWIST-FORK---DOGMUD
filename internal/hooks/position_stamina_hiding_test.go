package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// File: position_stamina_hiding_test.go
//
// M4d PR 3 Task 4: sendCharacterMsg, fireStaminaWarningIfLow's dispatcher.
// Its room line names the character the warning fires for -- c.Name, per
// staminaWarningSubstitutions' override, the one asymmetric mapping in this
// store (a controlled character's warning is about THAT character, not the
// controller). It went out on Room.SendTextVisual, which passes no names, so
// a shapes-only observer got only the tag-based messaging.Anonymize stage.
//
// c.Name is the bare Character.Name field, never GetCharacterName's
// ansi-tagged form, so unlike the other M4d PR3 slices, today's shipped
// content is NOT incidentally safe by tag-wrap discipline -- this test uses
// the shipped stamina_warning.observer text verbatim (substituted), not an
// authored bare-name stand-in.

// TestSendCharacterMsg_RoomLineHidesTheWarnedCharactersBareName proves and
// pins the fix for fireStaminaWarningIfLow's room line.
//
// Pre-fix this fails: sendCharacterMsg's room half goes out on
// Room.SendTextVisual with no names argument, so a SightShapes reader gets
// only messaging.Anonymize (tag-based), which never touches a bare name.
func TestSendCharacterMsg_RoomLineHidesTheWarnedCharactersBareName(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	darken(t, 1)
	require.True(t, users.GetByUserId(2).Character.Conditions.AddCondition(heatEyesConditionId, true))
	drainPlain(1)
	drainPlain(2)

	// user 1, Aliceia, is the character the warning fires for -- the {actor}
	// slot per staminaWarningSubstitutions' override, regardless of which
	// side of the grapple she is on.
	c := users.GetByUserId(1).Character

	const selfLine = "You're getting gassed — your mount is hard to maintain."
	const roomLine = "Aliceia looks exhausted in the mount." // stamina_warning.observer, substituted

	sendCharacterMsg(c, selfLine, roomLine)

	selfGot := drainPlain(1)
	require.Equal(t, 1, countContaining(selfGot, "getting gassed"),
		"the warned character should still get her own self line: got %v", selfGot)

	watcherGot := drainPlain(2)
	require.Equal(t, 0, countContaining(watcherGot, "Aliceia"),
		"a shapes-only observer must not read the warned character's bare name in the stamina room line: got %v", watcherGot)
	require.Equal(t, 1, countContaining(watcherGot, "A figure looks exhausted in the mount."),
		"a shapes-only observer should read the hidden form of the room line: got %v", watcherGot)
}
