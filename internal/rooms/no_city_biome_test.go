package rooms

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestNoDogmudRoomOrZoneDefaultsToCityBiome guards plan 3c-2b's deletion of
// the `city` biome from the dogmud world. Every room that was `biome: city`
// has been reclassified into a tier (`city_thoroughfare`, `city_backstreet`,
// `interior`, `dungeon`, ...), and every zone-config that defaulted new rooms
// to `city` now defaults to `city_backstreet`. `city.yaml` no longer exists,
// so a stray reference here would leave a room or a whole zone falling back
// onto a biome the game cannot load.
//
// Reads files as text and matches the exact anchored lines `biome: city` and
// `defaultbiome: city`, so `city_backstreet` / `city_thoroughfare` never
// match: those are the tiers this guard exists to protect.
func TestNoDogmudRoomOrZoneDefaultsToCityBiome(t *testing.T) {
	root := "../../_datafiles/world/dogmud/rooms"

	var badRooms []string
	var badZones []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)

		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimRight(line, "\r")
			switch line {
			case "biome: city":
				badRooms = append(badRooms, rel)
			case "defaultbiome: city":
				badZones = append(badZones, rel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	if len(badRooms) == 0 && len(badZones) == 0 {
		return
	}

	sort.Strings(badRooms)
	sort.Strings(badZones)

	var msg strings.Builder
	if len(badRooms) > 0 {
		msg.WriteString("rooms still authored biome: city:\n")
		for _, r := range badRooms {
			msg.WriteString("  " + r + "\n")
		}
	}
	if len(badZones) > 0 {
		msg.WriteString("zone-configs still default to city:\n")
		for _, z := range badZones {
			msg.WriteString("  " + z + "\n")
		}
	}
	t.Errorf("the dogmud world no longer ships a city biome; reclassify these into a tier:\n%s", msg.String())
}
