package items_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// The 5a ladder, read from the shipped world: every carried light has its
// strength, lives in the light slot, and is secret so a light shows at most
// once in the conditions list (owner, 2026-09-26).
func TestShippedLightItemsMatchTheLadder(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = configs.ConfigString(`../../_datafiles/world/dogmud`)
	// Condition 0 derives its TriggerCount from this at Validate time and
	// refuses 0; a test binary never reads config.yaml.
	cfg.Network.LogoutRounds = 3
	configs.SetConfigForTest(t, cfg)
	conditions.LoadDataFiles()
	items.LoadDataFiles()

	cases := []struct {
		itemId     int
		strength   float64
		adjustable bool
	}{
		{40077, 38, false}, // Tallow Candle
		{40038, 52, false}, // Oil Lantern
		{20096, 56, false}, // Torch
		{20097, 54, true},  // Hooded Lantern
	}
	for _, c := range cases {
		spec := items.GetItemSpec(c.itemId)
		if spec == nil {
			t.Fatalf("item %d is not shipped", c.itemId)
		}
		if spec.Type != items.Light || spec.Subtype != items.Wearable {
			t.Errorf("item %d is %s/%s, want light/wearable", c.itemId, spec.Type, spec.Subtype)
		}
		if len(spec.WornConditionIds) != 1 {
			t.Fatalf("item %d grants %d conditions, want 1", c.itemId, len(spec.WornConditionIds))
		}
		cond := conditions.GetConditionSpec(spec.WornConditionIds[0])
		if cond == nil || !cond.Secret {
			t.Errorf("item %d's light condition must exist and be secret", c.itemId)
			continue
		}
		if v := cond.Effects[conditions.EffectLightStrength]; v.UsesMagnitude || v.Literal != c.strength {
			t.Errorf("item %d shines at %+v, want %v", c.itemId, v, c.strength)
		}
		adjustable := false
		for _, f := range cond.Flags {
			adjustable = adjustable || f == conditions.Adjustable
		}
		if adjustable != c.adjustable {
			t.Errorf("item %d adjustable = %v, want %v", c.itemId, adjustable, c.adjustable)
		}
	}
	if spec := items.GetItemSpec(20097); spec == nil || spec.Nouns["hood"] == "" {
		t.Error("the hooded lantern must carry a hood noun")
	}
}
