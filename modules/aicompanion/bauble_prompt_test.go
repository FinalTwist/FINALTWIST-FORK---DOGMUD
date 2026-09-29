package aicompanion

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// What the companion is told about a room never carries text a player's
// own key wrote (spec S3; review finding f): a player-key bauble lying
// there reads as its carrier.
func TestRoomThingsKeepPlayerKeyTextOut(t *testing.T) {
	t.Cleanup(items.SeedItemsForTest(map[int]*items.ItemSpec{
		items.BaubleItemId: {ItemId: items.BaubleItemId, Name: `Curious Trinket`, NameSimple: `trinket`,
			Type: items.Object, Subtype: items.Mundane},
	}))
	items.SetBaubleResolver(func(id string) (items.BaubleView, bool) {
		return items.BaubleView{Name: `Painted Wooden Horse`, NameSimple: `horse`, PlayerText: true}, true
	})
	t.Cleanup(func() { items.SetBaubleResolver(nil) })

	b := items.New(items.BaubleItemId)
	b.Bauble = `b0000001`
	room := &rooms.Room{RoomId: 1, Items: []items.Item{b}}
	got := strings.Join(roomThings(room), `, `)
	if strings.Contains(got, `Horse`) || !strings.Contains(got, `Curious Trinket`) {
		t.Fatalf("the companion reads the carrier, never the player's text: %q", got)
	}
}
