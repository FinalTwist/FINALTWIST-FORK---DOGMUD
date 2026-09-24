package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TestOnboardingRoomsAreNeverDark pins the constraint that made `ether` exist.
//
// 🔴 look.go refuses to print a room description below LightBlindBelow, and
// these six rooms are character creation: The Threshold, Knowing Yourself,
// What You Carry, The World Speaks, The Proving, The Landing. If they ever
// follow the sun, a share of every day's new players arrive unable to read
// their own introduction, and nothing else in the suite would say so.
//
// Sampled across the year AND the clock, because the failure mode is seasonal.
// A room can read fine at whatever round a test happens to pick and be black
// at midwinter midnight.
func TestOnboardingRoomsAreNeverDark(t *testing.T) {
	etherLamp := 60
	zeroSky := 0.0
	cleanupBiomes := SeedBiomesForTest(map[string]*BiomeInfo{
		"ether": {
			BiomeId: "ether", Name: "Ether", Symbol: "∴",
			SkyLight: &zeroSky, Lamp: &etherLamp, Indoor: true,
		},
	})
	t.Cleanup(cleanupBiomes)

	cfg := configs.GetConfig()
	cfg.Timing.RoundsPerDay = 900
	cfg.Timing.RoundSeconds = 4
	cfg.Timing.Validate()
	cfg.Balance.Validate() // shipped latitude, deliberately not overridden
	configs.SetConfigForTest(t, cfg)

	gametime.ClearDateCacheForTest()
	t.Cleanup(gametime.ClearDateCacheForTest)

	original := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCountForTest(original) })

	dim := configs.GetLightingConfig().DimBelow
	room := Room{Biome: "ether"}

	// Midwinter, equinox and midsummer, at midnight, dawn, noon and dusk.
	for _, doy := range []int{356, 81, 172} {
		for _, hour := range []float64{0, 6, 12, 18} {
			round := uint64(float64(doy-1)*900 + hour*37.5)
			util.SetRoundCountForTest(round)
			gametime.ClearDateCacheForTest()

			if got := room.LightLevel(); got < dim {
				t.Errorf("day %d hour %v: ether reads %d, below DimBelow %d. "+
					"A character-creation room that is not FULLY readable is a "+
					"new player who cannot read their own introduction.",
					doy, hour, got, dim)
			}
		}
	}
}
