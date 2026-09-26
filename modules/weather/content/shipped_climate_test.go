package content

import (
	"os"
	"reflect"
	"testing"
)

// Validates the shipped DOGMud climate profiles: this fixed set of 19
// biomes covered, parseable, indoor biomes have zero spawn weight. The
// three biomes plan 3c added (city_thoroughfare, city_backstreet, ruins)
// are included in that count.
//
// house is gone (folded into interior, plan 3b Task 4); this list is not
// exhaustive over every shipped biome, so the six biomes plan 3b Task 3
// added (sewer, interior, dense_forest, plains, river, ether) are not swept
// here either.
func TestShippedDogmudClimateProfiles(t *testing.T) {
	fsys := os.DirFS("../../../_datafiles/world/dogmud")
	climate, err := LoadClimate(fsys, "weather/climate")
	if err != nil {
		t.Fatalf("LoadClimate: %v", err)
	}

	biomes := []string{"water", "shore", "cliffs", "desert", "snow", "mountains",
		"swamp", "forest", "farmland", "land", "road",
		"city_thoroughfare", "city_backstreet", "ruins",
		"cave", "dungeon", "fort", "spiderweb"}
	for _, b := range biomes {
		p, ok := climate[b]
		if !ok {
			t.Errorf("missing climate profile for biome %q", b)
			continue
		}
		if len(p.Weather) == 0 {
			t.Errorf("%s: empty weather weights", b)
		}
	}

	for _, indoor := range []string{"cave", "dungeon", "fort", "spiderweb"} {
		if w := climate[indoor].SpawnWeight; w != 0 {
			t.Errorf("%s: indoor biome must have spawnWeight 0, got %v", indoor, w)
		}
	}

	// The two city tiers are a LIGHTING split, not a weather one: they must
	// carry the same climate profile, field for field (plan 3c). city.yaml
	// is gone (plan 3c-2 deleted it, and "city" with it from the required
	// biomes list above); this always-on check replaces the old parity
	// check against that file.
	if !reflect.DeepEqual(climate["city_thoroughfare"], climate["city_backstreet"]) {
		t.Error("city_thoroughfare and city_backstreet climate differ; the tiers split light only")
	}
	// Pin one literal the deleted city.yaml shipped, so the tiers cannot
	// drift together, unnoticed, away from the value both were seeded with.
	if w := climate["city_thoroughfare"].SpawnWeight; w != 0.7 {
		t.Errorf("city_thoroughfare: expected spawnWeight 0.7 (city.yaml's shipped value), got %v", w)
	}

	// Pin values that differ from the module's built-in defaults so a missing
	// file can't silently fall back (forest/desert/swamp exist in defaults too).
	// desert: DOGMud file adds overcast weight 1; DefaultClimate "desert" has none (0).
	if climate["desert"].Weather["overcast"] != 1 {
		t.Errorf("desert: expected overcast weight 1 from the DOGMud climate file, got %v",
			climate["desert"].Weather["overcast"])
	}
	// swamp: DOGMud file has fog weight 3; DefaultClimate "swamp" has fog weight 5.
	if climate["swamp"].Weather["fog"] != 3 {
		t.Errorf("swamp: expected fog weight 3 from the DOGMud climate file, got %v",
			climate["swamp"].Weather["fog"])
	}
	// forest: DOGMud file has spawnWeight 0.9; DefaultClimate "forest" has spawnWeight 1.0.
	if climate["forest"].SpawnWeight != 0.9 {
		t.Errorf("forest: expected spawnWeight 0.9 from the DOGMud climate file, got %v",
			climate["forest"].SpawnWeight)
	}
}
