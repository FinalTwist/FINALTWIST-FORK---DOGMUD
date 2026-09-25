package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/lightnotice"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// TestLightNoticeArrivesBeforeTheCommandOutput pins the ordering: the notice is
// delivered before the command's own output, and only once.
//
// look's room title and description render through templates.Process, and
// this package's TestMain never registers a templates filesystem: per
// template_freeze_test.go's useDogmudTemplates, that leaves every lookup a
// vacuous success on an empty body, so "look" would print no title at all to
// match against. useDogmudTemplates points Process at the real shipped
// templates so the room title actually renders.
func TestLightNoticeArrivesBeforeTheCommandOutput(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	useDogmudTemplates(t)
	require.NoError(t, lightnotice.LoadFrom("../../_datafiles/world/dogmud/narration/light-notices"))
	t.Cleanup(lightnotice.ResetForTest)

	room1 := rooms.LoadRoom(1)
	require.NotNil(t, room1)
	room1.SkyLight, room1.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(30)
	u := users.GetByUserId(1)
	require.NotNil(t, u)
	lightnotice.Check(u, lightnotice.TriggerQuiet)
	room1.Lamp = rooms.LampPtr(60)

	pool := lightnotice.Pool(lightnotice.CauseLamp, lightnotice.LighterFaces, false)
	run := func() (noticeAt, titleAt, notices int) {
		events.ProcessEvents()
		var texts []string
		id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
			if m, ok := e.(events.Message); ok && m.UserId == 1 {
				texts = append(texts, strings.ReplaceAll(m.Text, "\n", " "))
			}
			return events.Continue
		})
		defer events.UnregisterListener(events.Message{}, id)
		_, err := TryCommand("look", "", 1, 0)
		require.NoError(t, err)
		events.ProcessEvents()
		noticeAt, titleAt = -1, -1
		for i, text := range texts {
			for _, l := range pool {
				if strings.Contains(text, l) {
					notices++
					if noticeAt < 0 {
						noticeAt = i
					}
				}
			}
			if titleAt < 0 && strings.Contains(text, "Town Square") {
				titleAt = i
			}
		}
		return
	}

	noticeAt, titleAt, notices := run()
	require.Equal(t, 1, notices, "exactly one notice")
	require.GreaterOrEqual(t, titleAt, 0, "look must print the room title")
	require.Less(t, noticeAt, titleAt, "the notice must precede the room description")

	_, _, notices = run()
	require.Equal(t, 0, notices, "a second look in the same band says nothing")
}
