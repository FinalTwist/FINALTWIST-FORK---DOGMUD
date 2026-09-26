package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func seedHoodedLanternForTest(t *testing.T) {
	t.Helper()
	t.Cleanup(items.SeedItemsForTest(map[int]*items.ItemSpec{
		999950: {ItemId: 999950, Name: "test hooded lantern", Type: items.Light, Subtype: items.Wearable,
			Description: "A lantern with a hood.", Nouns: map[string]string{"hood": "Close it with hood, open it with unhood."}},
	}))
}

func TestFindItemNounMatchesWornAndCarried(t *testing.T) {
	seedHoodedLanternForTest(t)
	c := New()
	c.Equipment.Light = items.New(999950)
	noun, desc, ok := c.FindItemNoun("hood")
	if !ok || noun != "hood" || desc != "Close it with hood, open it with unhood." {
		t.Errorf("FindItemNoun(hood) = (%q, %q, %v)", noun, desc, ok)
	}
	if _, _, ok := c.FindItemNoun("hoo"); ok {
		t.Error("an item noun matched a prefix; it must match exactly")
	}
}

func TestFindItemNounMatchesBackpackOnlyWhenCarried(t *testing.T) {
	seedHoodedLanternForTest(t)

	nobody := New()
	if _, _, ok := nobody.FindItemNoun("hood"); ok {
		t.Error("a noun matched on an item this character does not carry")
	}

	c := New()
	if !c.StoreItem(items.New(999950)) {
		t.Fatal("StoreItem refused the lantern; the fixture proves nothing")
	}
	noun, desc, ok := c.FindItemNoun("HOOD ")
	if !ok || noun != "hood" || desc != "Close it with hood, open it with unhood." {
		t.Errorf("FindItemNoun(HOOD) from the backpack = (%q, %q, %v)", noun, desc, ok)
	}
}
