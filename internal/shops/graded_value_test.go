package shops

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestGradedValue(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		40070: {ItemId: 40070, Name: "Pack-Hide Pelt", Value: 5, IsComponent: true},
	})
	defer cleanup()

	if got := GradedValue(items.Item{ItemId: 40070}); got != 5 {
		t.Errorf("ungraded pelt = %d, want spec value 5", got)
	}
	std := GradedValue(items.Item{ItemId: 40070, Quality: items.QualityStandard})
	fine := GradedValue(items.Item{ItemId: 40070, Quality: items.QualityFine})
	crude := GradedValue(items.Item{ItemId: 40070, Quality: items.QualityCrude})
	pristine := GradedValue(items.Item{ItemId: 40070, Quality: items.QualityPristine})
	if !(crude < std && std < fine && fine < pristine) {
		t.Errorf("grades should order crude<standard<fine<pristine, got %d %d %d %d", crude, std, fine, pristine)
	}
	if crude < 1 {
		t.Error("a graded item never prices below 1")
	}
}

// The walk-in bug: the second unit of an item the shop does not stock used to
// price on the scarcity curve at about four times the first.
func TestWalkInBuyPrice_NeverRisesAndSlidesGently(t *testing.T) {
	cfg := DefaultPricingConfig()
	first := WalkInBuyPrice(100, 0, cfg)
	second := WalkInBuyPrice(100, 1, cfg)
	tenth := WalkInBuyPrice(100, 9, cfg)
	if second > first {
		t.Errorf("second unit (%d) must not pay more than the first (%d)", second, first)
	}
	if tenth >= first || tenth < first/2 {
		t.Errorf("the tenth unit (%d) should pay a little less than the first (%d), not crash", tenth, first)
	}
	if floor := WalkInBuyPrice(100, 1000, cfg); floor < 1 {
		t.Error("never below 1")
	}
}

func TestEvaluateBuyRules_WalkInEntryUsesFlatSlope(t *testing.T) {
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		500: {ItemId: 500, Name: "Wolf Pelt", Value: 20, Type: items.Object, IsComponent: true, VendorCategories: []string{"tailoring"}},
	})
	defer cleanup()
	shop := &ShopInventory{Gold: 100000, StartingGold: 100000, CraftSupport: CraftSupportTailoring}
	cfg := DefaultPricingConfig()
	first := EvaluateBuyRules(items.Item{ItemId: 500}, shop, "", false, cfg, nil).Price
	shop.AddStock(500, 1) // the shop now holds the first pelt (RestockQty 0)
	second := EvaluateBuyRules(items.Item{ItemId: 500}, shop, "", false, cfg, nil).Price
	if first == 0 || second == 0 {
		t.Fatalf("the tailor should buy pelts: %d, %d", first, second)
	}
	if second > first {
		t.Errorf("second pelt paid %d, more than the first (%d)", second, first)
	}
}
