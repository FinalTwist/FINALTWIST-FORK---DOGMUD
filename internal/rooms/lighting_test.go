package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TestLightLevelMapsTheOldModel pins the mapping that makes graded lighting
// plan 1 behaviour preserving. Every shipped room must land on one of three
// values, and each must sit on the correct side of every threshold.
//
// 0  dark, a normal observer is blind
// 60 lit enough to see the room, NOT down an exit
// 70 lit enough to see the room and its exits
//
// legacyVisibility (the old three-value visibility accessor's body, moved to
// this file in Task 3 and now the only surviving copy since Task 5 deleted
// that old accessor itself) reads two pieces of ambient global state that a
// bare *Room{} fixture does not control by default, so both are pinned
// explicitly rather than left to whatever the test binary happens to start
// with:
//
//   - Room.GetBiome() (rooms.go) reads the package-level `biomes` map, which
//     is empty until LoadBiomeDataFiles() has run. An empty map makes
//     GetBiome() return a nil *BiomeInfo, and BiomeInfo.IsDark()
//     (internal/rooms/biomes.go) has a pointer receiver with no nil check,
//     so calling it on a nil biome panics. This is seeded with
//     SeedBiomesForTest (internal/rooms/test_helpers.go), which already
//     exists in this package for exactly this purpose, rather than loading
//     the real _datafiles/biomes tree from disk, which would also depend on
//     the test binary's working directory being the repo root rather than
//     this package's own directory.
//
//   - gametime.IsNight() (internal/gametime/gametime.go) reads the current
//     round count through gametime.GetDate(), whose day/night split is a
//     function of the Timing config (RoundsPerDay, NightHours) and
//     util.GetRoundCount(). A bare test binary never loads config.yaml, and
//     Go's zero-value default for NightHours is 0, which makes IsNight()
//     always false, so "night" is not even reachable without pinning Timing
//     explicitly first. Both Timing and the round count are pinned below so
//     day and night are each deterministic, not incidental.
func TestLightLevelMapsTheOldModel(t *testing.T) {
	cleanupBiomes := SeedBiomesForTest(map[string]*BiomeInfo{
		"default": {BiomeId: "default", Name: "Default"},
		"cave":    {BiomeId: "cave", Name: "Cave", DarkArea: true},
	})
	defer cleanupBiomes()

	// Pin Timing so day/night is a function of the round count this test
	// chooses, not of whatever the test binary's ambient default happens to
	// resolve to (NightHours=0 -> always day, see the comment above).
	cfg := configs.GetConfig()
	cfg.Timing.RoundsPerDay = 20
	cfg.Timing.NightHours = 8
	configs.SetConfigForTest(t, cfg)

	// Pick a day round and a night round directly, by the same arithmetic
	// gametime.GameDate.ReCalculate uses (internal/gametime/gametime.go):
	// roundOfDay := round % RoundsPerDay; hour := floor(roundOfDay / RoundsPerDay * 24);
	// night := hour >= nightStart || hour < nightEnd, with
	// nightEnd = floor(NightHours/2) = 4 and nightStart = 24 - nightEnd = 20
	// for the NightHours=8 pinned above. dayCycleBase starts far past any
	// round number another test in this package's shared test binary might
	// already have passed to gametime.GetDate (its round -> date cache is
	// keyed only by round number, not by Timing config, so a collision on a
	// small round number could read back a stale date computed under a
	// different Timing) -- confirmed live at internal/rooms/corpse_test.go
	// (gametime.GetDate(0)) and internal/rooms/spawninfo_validate_test.go
	// (gametime.GetDate(1)).
	//
	// dayCycleBase is a multiple of RoundsPerDay, so roundOfDay is 0 there:
	// hour 0 is always night when NightHours > 0 (0 < nightEnd). Adding half
	// a day (RoundsPerDay/2 = 10 rounds) lands on hour 12, always day.
	const dayCycleBase = uint64(500000)
	nightRound := dayCycleBase
	dayRound := dayCycleBase + uint64(cfg.Timing.RoundsPerDay)/2

	// Guard the arithmetic above rather than trusting it silently: fail
	// loudly, naming the round and the computed date, if either round does
	// not land on the day/night side this test assumes.
	if !gametime.GetDate(nightRound).Night {
		t.Fatalf("nightRound %d did not compute as night under pinned Timing (RoundsPerDay=%d, NightHours=%d); the test fixture's arithmetic is wrong",
			nightRound, cfg.Timing.RoundsPerDay, cfg.Timing.NightHours)
	}
	if gametime.GetDate(dayRound).Night {
		t.Fatalf("dayRound %d did not compute as day under pinned Timing (RoundsPerDay=%d, NightHours=%d); the test fixture's arithmetic is wrong",
			dayRound, cfg.Timing.RoundsPerDay, cfg.Timing.NightHours)
	}

	tests := []struct {
		name       string
		room       *Room
		roundCount uint64
		want       int
	}{
		{"default biome, day", &Room{RoomId: 1}, dayRound, LightFull},
		{"default biome, night", &Room{RoomId: 3}, nightRound, LightRoomOnly},
		{"cave", &Room{RoomId: 2, Biome: "cave"}, dayRound, LightDark},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			util.SetRoundCountForTest(tc.roundCount)
			if got := tc.room.LightLevel(); got != tc.want {
				t.Errorf("LightLevel() = %d, want %d (round %d, night=%v)", got, tc.want, tc.roundCount, gametime.IsNight())
			}
		})
	}
}
