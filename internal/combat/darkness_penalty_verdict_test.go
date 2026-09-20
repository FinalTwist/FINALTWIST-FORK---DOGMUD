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
// The row that matters most is "dark, infrared only": CanSeeSightImpairedOnly
// is FALSE there today (full penalty applies), and this test proves the
// verdict-carrying field still says the same thing. PR 2 is what gives
// SightShapes its own reduced value; this task must not deliver that early.

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

// verdictLight is a messaging.RoomVisibility with a fixed light level: 0
// dark, 1 lit. Local to this file rather than reused from messaging's
// unexported test fixtures, since those do not cross the package boundary.
type verdictLight int

func (l verdictLight) GetVisibility() int { return int(l) }

const (
	verdictInfraredConditionId = 9201
	verdictNightConditionId    = 9202
	verdictSleepConditionId    = 9203
)

// verdictObserver builds a fresh *characters.Character carrying the flags one
// of the eight optics_pin_test.go rows asks for.
func verdictObserver(t *testing.T, nightVision, infraredVision, asleep, blind bool) *characters.Character {
	t.Helper()
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		verdictInfraredConditionId: {ConditionId: verdictInfraredConditionId, Name: "Test Infrared", Flags: []conditions.Flag{conditions.InfraredVision}},
		verdictNightConditionId:    {ConditionId: verdictNightConditionId, Name: "Test Night", Flags: []conditions.Flag{conditions.NightVision}},
		verdictSleepConditionId:    {ConditionId: verdictSleepConditionId, Name: "Test Sleep", Flags: []conditions.Flag{conditions.Sleeping}},
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
		{name: "dark, nightvision", nightVision: true, wantOldImpairedOnly: true},
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
				room = verdictLight(1)
			}

			// Fixture sanity: confirm this row still matches what
			// optics_pin_test.go pins for CanSeeSightImpairedOnly before
			// trusting anything derived from it below.
			if got := messaging.CanSeeSightImpairedOnly(observer, room); got != tc.wantOldImpairedOnly {
				t.Fatalf("fixture broken: CanSeeSightImpairedOnly = %v, want %v (check optics_pin_test.go row %q)",
					got, tc.wantOldImpairedOnly, tc.name)
			}

			verdict := messaging.ParticipantSight(observer, room)
			wantPenalized := !tc.wantOldImpairedOnly

			clean := calcAttackScore(observer, target, items.Item{}, 0, combatContext{sourceSight: messaging.SightFull})
			ctxScore := calcAttackScore(observer, target, items.Item{}, 0, combatContext{sourceSight: verdict})

			gotPenalized := math.Abs(ctxScore-clean) > 1e-9
			if gotPenalized != wantPenalized {
				t.Fatalf("penalty applied = %v, want %v (verdict=%v, clean=%v, ctxScore=%v)",
					gotPenalized, wantPenalized, verdict, clean, ctxScore)
			}
			if wantPenalized {
				wantScore := clean * float64(cfg.Balance.DarknessCombatPenalty)
				if math.Abs(ctxScore-wantScore) > 1e-9 {
					t.Fatalf("penalized score = %v, want %v (clean %v x DarknessCombatPenalty %v)",
						ctxScore, wantScore, clean, cfg.Balance.DarknessCombatPenalty)
				}
			}

			if tc.name == "dark, infrared only" {
				if verdict != messaging.SightShapes {
					t.Fatalf("infrared in the dark must resolve to SightShapes, got %v", verdict)
				}
				if !gotPenalized {
					t.Fatal("infrared in the dark: PR 1 must still apply the full darkness penalty; PR 2 is what changes this")
				}
			}
		})
	}
}
