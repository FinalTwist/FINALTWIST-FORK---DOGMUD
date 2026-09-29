package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/require"
)

// A finder-only bauble (owner ruling 2026-09-29): its finder reads its own
// text in `inventory` and `look`; anyone else carrying it reads "Trinket".
func TestFinderOnlyBaubleReadsByViewer(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	// Midsummer noon, so look is not refused as blind (look_item_noun_test.go).
	cfg := configs.GetConfig()
	cfg.Timing.RoundsPerDay = 20
	configs.SetConfigForTest(t, cfg)
	gametime.ClearDateCacheForTest()
	t.Cleanup(gametime.ClearDateCacheForTest)
	util.SetRoundCountForTest(uint64(3430))
	t.Cleanup(util.ResetRoundCountForTest)

	restoreItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		items.BaubleItemId: {ItemId: items.BaubleItemId, Name: "Curious Trinket", NameSimple: "trinket",
			Type: items.Object, Subtype: items.Mundane, Weight: 0.2, Value: 1, NotSalable: true},
	})
	defer restoreItems()
	baubles.SetDirForTest(t.TempDir())
	defer items.SetBaubleResolver(nil)

	user, room := getTestUserAndRoom(t)
	origItems := user.Character.Items
	defer func() { user.Character.Items = origItems }()

	for _, finder := range []int{user.UserId, user.UserId + 1000} {
		rec, err := baubles.Create(baubles.Record{Name: "Painted Wooden Horse", NameSimple: "horse", Tier: baubles.TierCheap,
			Value: 3, WeightLbs: 0.5, Description: "A child's toy horse, its red paint flaking.", Status: baubles.StatusReady,
			PlayerKey: true, Moderated: false, FoundByUserId: finder})
		require.NoError(t, err)
		itm := items.New(items.BaubleItemId)
		itm.Bauble = rec.Id
		user.Character.Items = []items.Item{itm}

		events.DrainQueuedMessagesForTest(user.UserId)
		_, _ = Inventory("", user, room, 0)
		_, _ = Look("trinket", user, room, 0)
		out := strings.Join(events.DrainQueuedMessagesForTest(user.UserId), "\n")
		mine := finder == user.UserId
		if strings.Contains(out, "Painted Wooden Horse") != mine || strings.Contains(out, "toy horse") != mine {
			t.Errorf("found by %d, read by %d: the horse shows %v, want %v:\n%s", finder, user.UserId, !mine, mine, out)
		}
		require.Contains(t, out, "You look at the", "look reached the carried trinket")
	}
}
