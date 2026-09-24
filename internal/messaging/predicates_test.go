package messaging

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

func newChar(t *testing.T) *characters.Character {
	t.Helper()
	c := characters.New()
	c.Perception = perception.NewMachine()
	return c
}

func setBlinded(t *testing.T, c *characters.Character) {
	t.Helper()
	if err := c.Perception.TransitionTo(perception.Blinded,
		state.TransitionReason{Trigger: "test"}); err != nil {
		t.Fatalf("transition to Blinded failed: %v", err)
	}
}

func TestCanSeeClearlyLitRoomSighted(t *testing.T) {
	c := newChar(t)
	// Use nil room — the predicate short-circuits to "lit" on nil.
	// A zero-value &rooms.Room{} cannot be used here because
	// Room.LightLevel() calls into the biome registry which isn't
	// loaded in unit-test context (panics on nil BiomeInfo). Real
	// lit-room behavior is exercised in end-to-end tests with engine
	// boot.
	if !CanSeeClearly(c, nil) {
		t.Fatal("Sighted observer in default (nil) room should see clearly")
	}
}

func TestCanSeeClearlyBlinded(t *testing.T) {
	c := newChar(t)
	setBlinded(t, c)
	// Same nil-room caveat as TestCanSeeClearlyLitRoomSighted.
	if CanSeeClearly(c, nil) {
		t.Fatal("Blinded observer must NOT see clearly even in a lit room")
	}
}

func TestCanSeeShapesInfraredInDark(t *testing.T) {
	c := newChar(t)
	// Note: LightLevel() < LightBlindBelow = dark. We can't easily fabricate
	// a dark Room here without engine coupling — this test uses the
	// nil-room path which short-circuits to lit. Real darkness
	// behavior is exercised in pipeline_test.go's end-to-end suite.
	if !CanSeeShapes(c, nil) {
		t.Fatal("Sighted observer must see shapes (nil room defaults to lit)")
	}
}

func TestCanSeeShapesBlindedNoInfrared(t *testing.T) {
	c := newChar(t)
	setBlinded(t, c)
	if CanSeeShapes(c, nil) {
		t.Fatal("Blinded observer must NOT see shapes, even with nil/lit room")
	}
	_ = conditions.InfraredVision // ensure the flag constant exists
}

func TestNilCharacterDefaultsToSeeing(t *testing.T) {
	if !CanSeeClearly(nil, nil) {
		t.Fatal("nil observer must default to CanSeeClearly (defensive)")
	}
	if !CanSeeShapes(nil, nil) {
		t.Fatal("nil observer must default to CanSeeShapes (defensive)")
	}
}

// setSleeping gives the character the Sleeping condition flag.
//
// Unlike blindness, sleep is not a Perception state -- it is a condition flag, so
// this seeds a minimal spec into the global registry and applies it. The
// registry is restored by the returned cleanup.
func setSleeping(t *testing.T, c *characters.Character) {
	t.Helper()
	const sleepConditionId = 9001
	restore := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		sleepConditionId: {
			ConditionId: sleepConditionId,
			Name:        "Test Sleep",
			Flags:       []conditions.Flag{conditions.Sleeping},
		},
	})
	t.Cleanup(restore)
	if err := c.AddCondition(sleepConditionId, true); err != nil {
		t.Fatalf("applying the sleeping condition failed: %v", err)
	}
	if !c.HasConditionFlag(conditions.Sleeping) {
		t.Fatal("precondition: the character should now carry the Sleeping flag")
	}
}

// M0b Task 3. A sleeping character perceives nothing visual.
//
// The delivery pipeline had no concept of sleep at all -- grepping `Sleeping`
// under internal/messaging returned nothing -- so a sleeping player kept
// receiving every visual broadcast in the room: NPC dialogue, ambient flavour,
// arrivals and departures. Reported from play 2026-08-31.
//
// Audio is deliberately unaffected. Room.SendText bypasses this gate entirely,
// so a shout still reaches a sleeper and still wakes them; shout.go owns that.
func TestCanSeeClearly_SleeperPerceivesNothingVisual(t *testing.T) {
	c := newChar(t)
	setSleeping(t, c)
	// nil room short-circuits to LIT, so this proves sleep gates on its own
	// rather than riding on darkness.
	if CanSeeClearly(c, nil) {
		t.Error("a sleeping character must not see clearly, even in a lit room")
	}
	if CanSeeShapes(c, nil) {
		t.Error("a sleeping character must not see shapes either")
	}
}

// The direction a careless fix breaks: gating too broadly and blinding everyone.
// This passes BEFORE the change, which is what makes it a guard.
func TestCanSeeClearly_AwakeStillSees(t *testing.T) {
	c := newChar(t)
	if !CanSeeClearly(c, nil) {
		t.Error("an awake character in a lit room must still see clearly")
	}
	if !CanSeeShapes(c, nil) {
		t.Error("an awake character in a lit room must still see shapes")
	}
}

// The combat predicate must NOT gate on sleep, and the messaging one must.
//
// internal/combat/combat.go feeds CanSeeSightImpairedOnly into
// combatContext.sourceCanSee/targetCanSee, which drive DarknessCombatPenalty
// onto the attack score and every candidate defence score. If that predicate
// ever starts honouring sleep, a sleeping defender in a LIT room silently takes
// a darkness penalty, doubling a disadvantage ForceCrit already applies and
// writing a phantom darkness term into combat-analytics.jsonl -- which is the
// data tools/balance reads to tune the game.
//
// Found by blind adversarial review 2026-08-31, after the sleep gate was added
// to CanSeeClearly without auditing its non-messaging consumers.
func TestSleepGatesMessagingButNotTheCombatPredicate(t *testing.T) {
	c := newChar(t)
	setSleeping(t, c)

	if CanSeeClearly(c, nil) {
		t.Error("messaging: a sleeper must not see clearly")
	}
	if !CanSeeSightImpairedOnly(c, nil) {
		t.Error("combat: sleep must NOT count as sight impairment in a lit room; " +
			"that applies a darkness penalty in daylight and corrupts balance telemetry")
	}
}

// The combat predicate must still honour the things it always honoured.
func TestCanSeeSightImpairedOnly_StillHonoursBlindness(t *testing.T) {
	c := newChar(t)
	setBlinded(t, c)
	if CanSeeSightImpairedOnly(c, nil) {
		t.Error("a blinded character's sight IS impaired, sleep aside")
	}
}

// TestParticipantSightReadsTheBands reuses sightLight (participant_sight_test.go,
// same package) as its fixed-light RoomVisibility rather than adding a second
// stub. sightLight is already `type X int` implementing LightLevel() int, which
// is exactly what an arbitrary light value here needs; a second type with an
// identical body would just be the parallel-mechanism trap.
//
// It pins ParticipantSight's band switch for a plain observer: no blindness,
// no NightVision, no InfraredVision. Those three are already pinned by
// TestParticipantSight (participant_sight_test.go) and TestOpticsTruthTable
// (optics_pin_test.go); this test's only job is the light-band arithmetic
// itself.
//
// The config knobs are pinned explicitly to LightBlindBelow: 25 and
// LightDimBelow: 50 rather than trusted from the test binary's ambient Go
// defaults. A bare Balance{} only resolves to 25/50 because
// Balance.Validate() coerces zero to those defaults
// (config_lighting_thresholds_test.go pins that fact on the configs side),
// and this package's test binary also runs tests that call
// configs.SetConfigForTest. Pinning makes every boundary number below
// self-documenting and immune to drift from another test's config mutation,
// following the precedent in internal/rooms/lighting_test.go, which pins
// Timing explicitly for the equivalent reason.
func TestParticipantSightReadsTheBands(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.LightBlindBelow = 25
	cfg.Balance.LightDimBelow = 50
	configs.SetConfigForTest(t, cfg)

	tests := []struct {
		name  string
		light int
		want  SightDecision
	}{
		{"pitch dark", 0, SightNone},
		{"just below blind threshold", 24, SightNone},
		{"dim, bottom", 25, SightShapes},
		{"dim, top", 49, SightShapes},
		{"perfect, bottom", 50, SightFull},
		{"perfect, top", 75, SightFull},
		{"dazzled, still sees", 90, SightFull},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ch := characters.New()
			got := ParticipantSight(ch, sightLight(tc.light))
			if got != tc.want {
				t.Errorf("ParticipantSight at light %d = %v, want %v", tc.light, got, tc.want)
			}
		})
	}
}
