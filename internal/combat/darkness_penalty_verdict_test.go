package combat

// M4d PR 1 -- combatContext.sourceSight/targetSight carry the SightDecision
// verdict instead of the old sourceCanSee/targetCanSee booleans. This file
// pins that the new field produces the SAME darkness-penalty decision the
// booleans did, for every one of the eight observer/room states pinned in
// internal/messaging/optics_pin_test.go.
//
// It drives the real production scoring function, calcAttackScore, rather
// than reimplementing the `!= messaging.SightFull` test here: a copy of the
// conditional would only ever agree with itself, never catch a drift in the
// real one.
//
// M4d PR 2 (owner ruling 6, 2026-09-20) changes the row that matters most:
// "dark, infrared only" now takes DarknessShapesCombatPenalty, a REDUCED
// penalty, not the full DarknessCombatPenalty every other impaired row
// still takes. TestDarknessShapesPenaltyIsBetweenBlindAndClean below is the
// test that pins the ruling itself; this test's job is only to keep pinning
// which verdict each observer/room state produces.

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// verdictLight is a messaging.RoomVisibility with a fixed light level on the
// graded scale: 0 dark (comfortably below LightBlindBelow's default of 25),
// 100 lit (the scale's top, comfortably above LightDimBelow's default of
// 50). Local to this file rather than reused from messaging's unexported
// test fixtures, since those do not cross the package boundary.
type verdictLight int

func (l verdictLight) LightLevel() int { return int(l) }

const (
	verdictInfraredConditionId = 9201
	verdictNightConditionId    = 9202
	verdictSleepConditionId    = 9203
)

// verdictObserver builds a fresh *characters.Character carrying the flags one
// of the eight optics_pin_test.go rows asks for.
//
// GRADED LIGHTING PLAN 2. The infrared fixture declares an explicit
// infra_reach (matching shipped condition 85's authored 30; see
// internal/characters/vision.go), not a bare flag. A bare InfraredVision
// flag reads reach 0 by design, so without a declared reach this fixture
// would stop demonstrating infrared reading shapes in the dark, which is
// exactly the row (M4d PR 2's reduced darkness penalty) this file exists to
// pin.
func verdictObserver(t *testing.T, nightVision, infraredVision, asleep, blind bool) *characters.Character {
	t.Helper()
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		verdictInfraredConditionId: {
			ConditionId: verdictInfraredConditionId,
			Name:        "Test Infrared",
			Flags:       []conditions.Flag{conditions.InfraredVision},
			Effects:     map[conditions.EffectKind]conditions.EffectValue{conditions.EffectInfraReach: {Literal: 30}},
		},
		verdictNightConditionId: {ConditionId: verdictNightConditionId, Name: "Test Night", Flags: []conditions.Flag{conditions.NightVision}},
		verdictSleepConditionId: {ConditionId: verdictSleepConditionId, Name: "Test Sleep", Flags: []conditions.Flag{conditions.Sleeping}},
	}))
	c := characters.New()
	if nightVision {
		if err := c.AddCondition(verdictNightConditionId, true); err != nil {
			t.Fatalf("applying night vision: %v", err)
		}
	}
	if infraredVision {
		if err := c.AddCondition(verdictInfraredConditionId, true); err != nil {
			t.Fatalf("applying infrared vision: %v", err)
		}
	}
	if asleep {
		if err := c.AddCondition(verdictSleepConditionId, true); err != nil {
			t.Fatalf("applying sleep: %v", err)
		}
	}
	if blind {
		if err := c.Perception.TransitionTo(perception.Blinded, state.TransitionReason{Trigger: "test"}); err != nil {
			t.Fatalf("transition to blinded: %v", err)
		}
	}
	return c
}

// TestDarknessPenaltyVerdictMatchesOldBoolean is the equivalence test M4d PR
// 1 requires: for every one of the eight optics_pin_test.go observer/room
// states, calcAttackScore applies Balance.DarknessCombatPenalty under the new
// combatContext.sourceSight field exactly when messaging.CanSeeSightImpairedOnly
// -- the boolean combat used to store -- said the source could not see.
func TestDarknessPenaltyVerdictMatchesOldBoolean(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.DarknessCombatPenalty = 0.4
	configs.SetConfigForTest(t, cfg)

	cases := []struct {
		name                string
		blind, asleep       bool
		nightVision         bool
		infraredVision      bool
		lit                 bool
		wantOldImpairedOnly bool // messaging.CanSeeSightImpairedOnly today
	}{
		{name: "lit, ordinary", lit: true, wantOldImpairedOnly: true},
		{name: "dark, no vision", wantOldImpairedOnly: false},
		// GRADED LIGHTING PLAN 2 moved this row (see
		// internal/messaging/optics_pin_test.go): a shifted window is still
		// blind below its floor at light 0, no matter the shift, so a
		// nightvision-only holder no longer reads SightFull in a pitch dark
		// room and now DOES take the darkness penalty.
		{name: "dark, nightvision", nightVision: true, wantOldImpairedOnly: false},
		{name: "dark, infrared only", infraredVision: true, wantOldImpairedOnly: false},
		{name: "blind in a lit room", blind: true, lit: true, wantOldImpairedOnly: false},
		{name: "blind with infrared", blind: true, infraredVision: true, wantOldImpairedOnly: false},
		{name: "asleep in a lit room", asleep: true, lit: true, wantOldImpairedOnly: true},
		{name: "asleep with infrared in the dark", asleep: true, infraredVision: true, wantOldImpairedOnly: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observer := verdictObserver(t, tc.nightVision, tc.infraredVision, tc.asleep, tc.blind)
			target := characters.New()

			var room messaging.RoomVisibility = verdictLight(0)
			if tc.lit {
				room = verdictLight(100)
			}

			// Fixture sanity: confirm this row still matches what
			// optics_pin_test.go pins for CanSeeSightImpairedOnly before
			// trusting anything derived from it below.
			if got := messaging.CanSeeSightImpairedOnly(observer, room); got != tc.wantOldImpairedOnly {
				t.Fatalf("fixture broken: CanSeeSightImpairedOnly = %v, want %v (check optics_pin_test.go row %q)",
					got, tc.wantOldImpairedOnly, tc.name)
			}

			verdict := messaging.ParticipantSight(observer, room)
			// wantPenalized only tracks whether ANY penalty applies; it is
			// derived from the old boolean and stays correct for that narrow
			// question even for SightShapes, since PR 2 only changes WHICH
			// multiplier a shapes verdict takes, not whether one applies at
			// all.
			wantPenalized := !tc.wantOldImpairedOnly

			clean := calcAttackScore(observer, target, items.Item{}, 0, combatContext{sourceSight: messaging.SightFull})
			ctxScore := calcAttackScore(observer, target, items.Item{}, 0, combatContext{sourceSight: verdict})

			gotPenalized := math.Abs(ctxScore-clean) > 1e-9
			if gotPenalized != wantPenalized {
				t.Fatalf("penalty applied = %v, want %v (verdict=%v, clean=%v, ctxScore=%v)",
					gotPenalized, wantPenalized, verdict, clean, ctxScore)
			}
			if wantPenalized {
				// M4d PR 2: the multiplier depends on the VERDICT, not just on
				// whether a penalty applies. SightShapes takes the reduced
				// shapes penalty; SightNone (and blind, which forces
				// SightNone) still takes the full blind penalty.
				wantMult := float64(cfg.Balance.DarknessCombatPenalty)
				if verdict == messaging.SightShapes {
					wantMult = float64(cfg.Balance.DarknessShapesCombatPenalty)
				}
				wantScore := clean * wantMult
				if math.Abs(ctxScore-wantScore) > 1e-9 {
					t.Fatalf("penalized score = %v, want %v (clean %v x multiplier %v, verdict %v)",
						ctxScore, wantScore, clean, wantMult, verdict)
				}
			}

			if tc.name == "dark, infrared only" {
				if verdict != messaging.SightShapes {
					t.Fatalf("infrared in the dark must resolve to SightShapes, got %v", verdict)
				}
				if !gotPenalized {
					t.Fatal("infrared in the dark: PR 2 still applies a reduced darkness penalty, never zero (owner ruling 6, 2026-09-20)")
				}
			}
		})
	}
}

// TestDarknessShapesPenaltyIsBetweenBlindAndClean is the ruling test for
// M4d PR 2 (owner ruling 6, 2026-09-20): "darkness combat penalty should be
// less for infra characters, but not zero." An infrared combatant who only
// makes out SHAPES must land and defend more often than one who is fully
// blind, and still worse than one who can see clearly.
//
// This drives calcAttackScore directly with each SightDecision rather than
// building room/vision fixtures: TestDarknessPenaltyVerdictMatchesOldBoolean
// above already pins which verdict each observer/room state produces, so
// this test only needs to pin what each verdict is WORTH.
func TestDarknessShapesPenaltyIsBetweenBlindAndClean(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.DarknessCombatPenalty = 0.50
	cfg.Balance.DarknessShapesCombatPenalty = 0.75
	configs.SetConfigForTest(t, cfg)

	observer := characters.New()
	target := characters.New()

	clearScore := calcAttackScore(observer, target, items.Item{}, 0, combatContext{sourceSight: messaging.SightFull})
	shapesScore := calcAttackScore(observer, target, items.Item{}, 0, combatContext{sourceSight: messaging.SightShapes})
	blindScore := calcAttackScore(observer, target, items.Item{}, 0, combatContext{sourceSight: messaging.SightNone})

	if !(blindScore < shapesScore && shapesScore < clearScore) {
		t.Fatalf("want blindScore < shapesScore < clearScore, got blind=%v shapes=%v clear=%v",
			blindScore, shapesScore, clearScore)
	}

	wantShapesScore := clearScore * float64(cfg.Balance.DarknessShapesCombatPenalty)
	if math.Abs(shapesScore-wantShapesScore) > 1e-9 {
		t.Fatalf("shapesScore = %v, want %v (clear %v x DarknessShapesCombatPenalty %v)",
			shapesScore, wantShapesScore, clearScore, cfg.Balance.DarknessShapesCombatPenalty)
	}
}
