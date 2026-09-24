package hooks

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A quest action's observer line (authored `room_text` until M4b-1 renamed it)
// went out RAW on the audio channel: no token substitution, so
// quest 77 showed players a literal {actor}; and no sight gate, so a blind
// observer still read "unlocks the strongbox". The behaviour tree's own
// `room_text` action param, which is a different store, already did both
// correctly (behaviortree/actions_dialogue.go).

func TestQuestRoomText_NamesThePlayerToASightedObserver(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	drainPlain(1)
	drainPlain(2)

	questengine.NewGameBridge(users.GetByUserId(1), 1).Narrate(quests.ActionDef{RoomText: "{actor} unlocks the strongbox."}.Narration())

	observer := drainPlain(2)
	assert.Equal(t, 1, countContaining(observer, "Aliceia unlocks the strongbox."))
	assert.Equal(t, 0, countContaining(observer, "{actor}"), "a literal token reached a player")
	assert.Equal(t, 0, countContaining(drainPlain(1), "unlocks the strongbox"),
		"the acting player is excluded from their own room line")
}

func TestQuestRoomText_UnsightedObserverInTheDarkGetsNothing(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	darken(t, 1)
	drainPlain(2)

	questengine.NewGameBridge(users.GetByUserId(1), 1).Narrate(quests.ActionDef{RoomText: "{actor} unlocks the strongbox."}.Narration())
	assert.Equal(t, 0, countContaining(drainPlain(2), "unlocks the strongbox"))
}

// TestQuestRoomText_InfraredObserverSeesAFigure proves the name carries the tag
// messaging.Anonymize strips. An untagged name would reach an infrared-only
// observer in full.
func TestQuestRoomText_InfraredObserverSeesAFigure(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	darken(t, 1)
	require.True(t, users.GetByUserId(2).Character.Conditions.AddCondition(heatEyesConditionId, true))
	drainPlain(2)

	questengine.NewGameBridge(users.GetByUserId(1), 1).Narrate(quests.ActionDef{RoomText: "{actor} unlocks the strongbox."}.Narration())

	observer := drainPlain(2)
	require.Equal(t, 1, countContaining(observer, "unlocks the strongbox"))
	for _, line := range observer {
		if strings.Contains(line, "unlocks the strongbox") {
			assert.NotContains(t, line, "Aliceia", "an infrared-only observer read the name")
			// Lowercased because nothing capitalises the placeholder: normalize
			// runs before anonymize, and CategoryNPCDialogue skips every
			// normalize stage anyway, so the observer reads "a figure ...".
			assert.Contains(t, strings.ToLower(line), "a figure")
		}
	}
}
