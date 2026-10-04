package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/timber"
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

// A crafted output takes the wood of its stave or shafts, else of its log.
func TestCraftWood(t *testing.T) {
	d, err := timber.Parse([]byte("species:\n  - {id: yew, name: yew, log: 40412, tier: 3}\nbiomes:\n  forest: [{species: yew, weight: 1}]\n"), timber.World{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	timber.Install(d)
	t.Cleanup(func() { timber.Install(nil) })
	if w := CraftWood([]items.Item{{ItemId: 1}, {ItemId: 2, Wood: `ash`}}); w != `ash` {
		t.Errorf("stave wood, got %q", w)
	}
	if w := CraftWood([]items.Item{{ItemId: 40412}}); w != `yew` {
		t.Errorf("log wood, got %q", w)
	}
	if w := CraftWood([]items.Item{{ItemId: 1}}); w != `` {
		t.Errorf("nothing wooden, got %q", w)
	}
}

// Harder ore is harder to dig and slower; more strength brings out more,
// never past the cap; gem odds stay small.
func TestMineNumbers(t *testing.T) {
	if !(MineTarget(1) < MineTarget(2) && MineTarget(2) < MineTarget(4)) {
		t.Error("difficulty rises with ore tier")
	}
	if MineRounds(4) <= MineRounds(1) {
		t.Error("harder ore takes longer")
	}
	if OreFor(100, items.QualityStandard) != 1 || OreFor(300, items.QualityPristine) > 3 {
		t.Error("ore yield: one for a baseline miner, capped at MiningMaxOre")
	}
	if c := gemChance(100, items.ToolTierIron); c <= 0 || c > 0.25 {
		t.Errorf("gem chance %v should be small and positive", c)
	}
	if gemChance(100, items.ToolTierMasterwork) <= gemChance(100, items.ToolTierCrude) {
		t.Error("a better pick finds more gems")
	}
}
