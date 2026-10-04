package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// A better sickle draws more of the patch: none for crude, a coin flip for
// iron, one for steel, one and a coin flip for masterwork (shipped
// RareToolMult* 0.5 / 1.0 / 1.5 / 2.0).
func TestSickleExtraDraws(t *testing.T) {
	cases := []struct {
		tier     items.ToolTier
		roll     int
		wantDraw int
	}{
		{items.ToolTierCrude, 0, 0},
		{items.ToolTierIron, 10, 1},
		{items.ToolTierIron, 90, 0},
		{items.ToolTierSteel, 99, 1},
		{items.ToolTierMasterwork, 10, 2},
		{items.ToolTierMasterwork, 90, 1},
	}
	for _, c := range cases {
		if got := SickleExtraDraws(c.tier, c.roll); got != c.wantDraw {
			t.Errorf("%s roll %d: %d extra draws, want %d", c.tier, c.roll, got, c.wantDraw)
		}
	}
}
