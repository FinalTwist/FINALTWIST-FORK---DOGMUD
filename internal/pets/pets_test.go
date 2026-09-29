package pets

import (
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// A find put straight into a pack pet has been taken, as one put in a pack
// has (Character.StoreItem): no spot, no household, no untaken time.
func TestPetStoreItemClearsBaublePlacement(t *testing.T) {
	p := Pet{Type: `packmule`, Capacity: 4}
	it := items.Item{ItemId: items.BaubleItemId, Bauble: `B0000001`}
	it.LeaveBaubleAt(`on the shelf`, 5, time.Now().Add(-time.Hour))
	if !p.StoreItem(it) {
		t.Fatal("the pet took nothing")
	}
	if got := p.Items[0]; got.BaubleSpot != `` || got.BaubleHousehold != 0 || got.BaubleLeftAt != 0 {
		t.Fatalf("stored with its placement: %+v", got)
	}
}
