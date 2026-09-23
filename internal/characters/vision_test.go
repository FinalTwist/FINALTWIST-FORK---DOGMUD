package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// Test condition IDs, kept out of the range SeedConditionRecordsForTest uses
// (79, 80, 117-123) and the perceives_test.go fixture (7301), following that
// file's own pattern of a package-local const per fixture id.
const (
	visionBareFlagConditionId = 7401
	visionExplicitConditionId = 7402
	visionStrongerConditionId = 7403
)

// visionChar builds a bare fixture the way perceives_test.go does: New()
// plus a name, no other wiring. NightVisionStrength and InfraReach only read
// c.Conditions and c.Mutations, neither of which needs Awareness or the
// other machines perceivesChar sets up.
func visionChar(t *testing.T, name string) *Character {
	t.Helper()
	c := New()
	c.Name = name
	return c
}

// pinVisionDefault pins LightDefaultVisionStrength the way
// internal/rooms/lighting_test.go pins Timing: read the live config, set the
// one field this test cares about, hand it back with SetConfigForTest so a
// bare test binary's zero-valued Balance (which Validate would otherwise
// coerce to the shipped default of 12 anyway) never leaves the fallback
// value implicit.
func pinVisionDefault(t *testing.T, strength int) {
	t.Helper()
	cfg := configs.GetConfig()
	cfg.Balance.LightDefaultVisionStrength = configs.ConfigInt(strength)
	configs.SetConfigForTest(t, cfg)
}

// pinMutationRankMultipliers pins the shipped rank multipliers (1.6 / 2.5 /
// 4.0) rather than the Go zero-value defaults (1.5 / 2.0 / 2.5) a bare test
// binary would otherwise fall back to, per LevelMultiplier's own doc comment
// (internal/mutations/mutations.go) and the shipped values recorded in
// _datafiles/config.yaml.
func pinMutationRankMultipliers(t *testing.T) {
	t.Helper()
	cfg := configs.GetConfig()
	cfg.Balance.MutationLevel2Multiplier = 1.6
	cfg.Balance.MutationLevel3Multiplier = 2.5
	cfg.Balance.MutationLevel4Multiplier = 4.0
	configs.SetConfigForTest(t, cfg)
}

// TestVisionNumbers covers NightVisionStrength and InfraReach, the two
// numbers the window model (messaging.SightThroughWindow) asks of an
// observer.
func TestVisionNumbers(t *testing.T) {
	t.Run("no vision flag at all reads zero for both", func(t *testing.T) {
		c := visionChar(t, "Viewer")
		if got := c.NightVisionStrength(); got != 0 {
			t.Fatalf("NightVisionStrength with no flag = %d, want 0", got)
		}
		if got := c.InfraReach(); got != 0 {
			t.Fatalf("InfraReach with no flag = %d, want 0", got)
		}
	})

	t.Run("a bare nightvision flag reads the configured default", func(t *testing.T) {
		pinVisionDefault(t, 15)
		t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
			visionBareFlagConditionId: {ConditionId: visionBareFlagConditionId, Name: "Test Bare Nightvision", Flags: []conditions.Flag{conditions.NightVision}},
		}))
		c := visionChar(t, "Viewer")
		if err := c.AddCondition(visionBareFlagConditionId, true); err != nil {
			t.Fatalf("applying bare nightvision: %v", err)
		}
		if got := c.NightVisionStrength(); got != 15 {
			t.Fatalf("NightVisionStrength with a bare flag = %d, want the pinned default 15", got)
		}
	})

	t.Run("a bare infraredvision flag reads zero, not the default", func(t *testing.T) {
		pinVisionDefault(t, 15)
		t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
			visionBareFlagConditionId: {ConditionId: visionBareFlagConditionId, Name: "Test Bare Infrared", Flags: []conditions.Flag{conditions.InfraredVision}},
		}))
		c := visionChar(t, "Viewer")
		if err := c.AddCondition(visionBareFlagConditionId, true); err != nil {
			t.Fatalf("applying bare infraredvision: %v", err)
		}
		if got := c.InfraReach(); got != 0 {
			t.Fatalf("InfraReach with a bare flag = %d, want 0: a bare infrared flag has no sensible default", got)
		}
	})

	t.Run("a condition declaring an explicit strength reads that strength", func(t *testing.T) {
		t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
			visionExplicitConditionId: {
				ConditionId: visionExplicitConditionId,
				Name:        "Test Explicit Nightvision",
				Flags:       []conditions.Flag{conditions.NightVision},
				Effects:     map[conditions.EffectKind]conditions.EffectValue{conditions.EffectNightVisionStrength: {Literal: 5}},
			},
		}))
		c := visionChar(t, "Viewer")
		if err := c.AddCondition(visionExplicitConditionId, true); err != nil {
			t.Fatalf("applying explicit nightvision: %v", err)
		}
		if got := c.NightVisionStrength(); got != 5 {
			t.Fatalf("NightVisionStrength with an explicit strength = %d, want 5", got)
		}
	})

	// TestVisionNumbers/a_condition_and_a_mutation_both_granting_it also pins
	// the fractional-rounding decision: a rank-2 mutation carrying
	// value: 6 scales to 6*1.6 = 9.6 under the shipped multiplier (pinned
	// above, not the Go default 2.0 that would give 12). The condition
	// offers only 5, so the mutation's 9.6 must win, and it must round to
	// the nearest int (10), not truncate to 9. See vision.go's doc comment
	// for why comparing happens in float64 before that single rounding cast.
	t.Run("a condition and a mutation both granting it read the stronger, rounded", func(t *testing.T) {
		pinMutationRankMultipliers(t)
		t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
			visionStrongerConditionId: {
				ConditionId: visionStrongerConditionId,
				Name:        "Test Weaker Nightvision",
				Flags:       []conditions.Flag{conditions.NightVision},
				Effects:     map[conditions.EffectKind]conditions.EffectValue{conditions.EffectNightVisionStrength: {Literal: 5}},
			},
		}))
		t.Cleanup(mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
			"test-keen-eyes": {
				MutationId: "test-keen-eyes",
				Name:       "Test Keen Eyes",
				Pros:       []mutations.MutationEffect{{Type: "flag", Target: string(conditions.NightVision), Value: 6}},
			},
		}))
		c := visionChar(t, "Viewer")
		if err := c.AddCondition(visionStrongerConditionId, true); err != nil {
			t.Fatalf("applying weaker condition nightvision: %v", err)
		}
		c.Mutations = map[string]int{"test-keen-eyes": 2}

		if got := c.NightVisionStrength(); got != 10 {
			t.Fatalf("NightVisionStrength with condition 5 vs mutation 9.6 = %d, want 10 (the stronger, rounded)", got)
		}
	})
}
