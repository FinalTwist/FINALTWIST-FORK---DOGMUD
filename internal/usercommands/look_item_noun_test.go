package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/require"
)

// File: look_item_noun_test.go
//
// Lighting plan 5a Task 10: items carry nouns the way rooms do. `look hood`
// must reach the hooded lantern's hood (exact match, ahead of FindItem's
// prefix matching, which would otherwise pick the lantern itself), and
// `look lantern` must highlight the noun inside the item's description.
func TestLook_ItemNounIsLookableAndHighlighted(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Midsummer noon, so the room is lit and look is not refused as blind
	// (see look_exit_visibility_test.go for the round arithmetic).
	cfg := configs.GetConfig()
	cfg.Timing.RoundsPerDay = 20
	configs.SetConfigForTest(t, cfg)
	gametime.ClearDateCacheForTest()
	t.Cleanup(gametime.ClearDateCacheForTest)
	util.SetRoundCountForTest(uint64(3430))
	t.Cleanup(util.ResetRoundCountForTest)

	const hoodText = "Close it with hood, open it with unhood."
	restoreItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		999950: {ItemId: 999950, Name: "test hooded lantern", Type: items.Light, Subtype: items.Wearable,
			Description: "A lantern with a hood.", Nouns: map[string]string{"hood": hoodText}},
	})
	defer restoreItems()

	user, room := getTestUserAndRoom(t)
	origLight := user.Character.Equipment.Light
	user.Character.Equipment.Light = items.New(999950)
	defer func() { user.Character.Equipment.Light = origLight }()

	events.DrainQueuedMessagesForTest(user.UserId)
	handled, err := Look("hood", user, room, 0)
	require.True(t, handled)
	require.NoError(t, err)
	joined := strings.Join(events.DrainQueuedMessagesForTest(user.UserId), "\n")
	require.Contains(t, joined, hoodText, "look hood must show the lantern's hood noun")
	require.Contains(t, joined, `You look at the <ansi fg="noun">hood</ansi>:`)

	handled, err = Look("lantern", user, room, 0)
	require.True(t, handled)
	require.NoError(t, err)
	joined = strings.Join(events.DrainQueuedMessagesForTest(user.UserId), "\n")
	require.Contains(t, joined, "You look at the", "look lantern must reach the lantern itself")
	require.Contains(t, joined, `<ansi fg="noun">hood</ansi>`,
		"the lantern's description must highlight its hood noun")
	require.NotContains(t, joined, hoodText, "look lantern shows the lantern, not the noun text")
}
