package enchantments

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// A bauble is never enchanted: ApplyTier would bake its catalog text into
// Item.Spec, past a retire and past the finder-only view (slice H).
func TestApplyTier_NeverTouchesABauble(t *testing.T) {
	// Seed the carrier: without its spec ApplyTier returns before touching
	// anything, and this test could not fail.
	restore := items.SeedItemsForTest(map[int]*items.ItemSpec{
		items.BaubleItemId: {ItemId: items.BaubleItemId, Name: `Curious Trinket`, NameSimple: `trinket`, Type: items.Object, Subtype: items.Mundane},
	})
	defer restore()
	def := &EnchantmentDef{EnchantId: "test-edge", Tiers: []TierDef{{Tier: 0, Adjective: "gleaming"}}}
	itm := items.Item{ItemId: items.BaubleItemId, Bauble: "b0000001"}
	ApplyTier(&itm, def, 0)
	if itm.Spec != nil || len(itm.Adjectives) != 0 {
		t.Fatalf("the bauble is untouched: spec %v adjectives %v", itm.Spec != nil, itm.Adjectives)
	}
}
