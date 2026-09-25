package content

import (
	"os"
	"reflect"
	"testing"
)

// TestCityTiersShareTheCityPools: the city split is about light, so both tiers
// must say the same urban things about the weather. Holds before and after
// 3c-2 deletes the city key: wherever any of the three keys appears, both
// tiers must be present and identical, and equal to city while city exists.
func TestCityTiersShareTheCityPools(t *testing.T) {
	root := os.DirFS("../../../_datafiles/world/dogmud")

	check := func(where string, sec TableSection) bool {
		city, hasCity := sec.Outdoor["city"]
		th, hasTh := sec.Outdoor["city_thoroughfare"]
		bk, hasBk := sec.Outdoor["city_backstreet"]
		if !hasCity && !hasTh && !hasBk {
			return false
		}
		if !hasTh || !hasBk {
			t.Errorf("%s: an urban pool exists but city_thoroughfare=%v city_backstreet=%v",
				where, hasTh, hasBk)
			return true
		}
		if !reflect.DeepEqual(th, bk) {
			t.Errorf("%s: city_thoroughfare and city_backstreet pools differ", where)
		}
		if hasCity && !reflect.DeepEqual(city, th) {
			t.Errorf("%s: the tiers' pool differs from city's", where)
		}
		return true
	}

	found := 0
	tables, err := LoadEmotes(root, "weather/emotes")
	if err != nil {
		t.Fatalf("LoadEmotes: %v", err)
	}
	for wt, tbl := range tables {
		if check(string(wt), tbl.TableSection) {
			found++
		}
		for season, sec := range tbl.Seasonal {
			if check(string(wt)+" season:"+season, sec) {
				found++
			}
		}
	}
	seasonal, err := LoadSeasonalEmotes(root, "weather/emotes/seasons")
	if err != nil {
		t.Fatalf("LoadSeasonalEmotes: %v", err)
	}
	for k, sec := range seasonal {
		if check("ambience "+k.Track+"/"+k.Season, sec) {
			found++
		}
	}
	// Six pools shipped when this was written; fewer means one was lost.
	if found < 6 {
		t.Errorf("found %d urban pools, want at least 6", found)
	}
}
