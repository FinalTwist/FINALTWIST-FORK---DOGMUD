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
