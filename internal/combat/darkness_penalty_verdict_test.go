package combat

// M4d PR 1 -- combatContext.sourceSight/targetSight carry the SightDecision
// verdict instead of the old sourceCanSee/targetCanSee booleans. This file
// pins which verdict each of the eight observer/room states pinned in
// internal/messaging/optics_pin_test.go produces.
//
// Lighting plan 5b split combatContext in two. The verdict fields
// (sourceSight/targetSight) are the NARRATION gate only: they decide name
// hiding and no longer move a score. The comfort fields
// (sourceDark/sourceBright, targetDark/targetBright, from
// messaging.ComfortDistance) are the only thing the scoring reads, through
// messaging.SightScoreMultiplier: a linear ramp from 1.0 at the edge of the
// comfortable band to DarknessCombatPenalty at the blind edge and to
// DazzleCap one ramp-width past the dazzle edge. The old flat band (shapes
// 0.90, blind 0.80) is gone; a shapes-only attacker now pays by distance.
//
// Every test here drives the real production scoring function,
// calcAttackScore (or the real defence core), rather than reimplementing
// the multiplier: a copy would only ever agree with itself.

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/contest"
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
// states, the combatContext.sourceSight verdict is SightFull exactly when
// messaging.CanSeeSightImpairedOnly -- the boolean combat used to store --
// said the source could see. Since plan 5b the verdict no longer scores, so
// this test also pins that a verdict alone leaves calcAttackScore untouched.
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
			// The verdict still follows the old boolean: SightFull exactly
			// when CanSeeSightImpairedOnly said the observer could see.
			if gotFull := verdict == messaging.SightFull; gotFull != tc.wantOldImpairedOnly {
				t.Fatalf("verdict = %v, want SightFull == %v", verdict, tc.wantOldImpairedOnly)
			}

			// Plan 5b: the verdict is the narration gate only. On its own it
			// must not move the score in any row; the comfort fields do that.
			clean := calcAttackScore(observer, target, items.Item{}, 0, combatContext{sourceSight: messaging.SightFull})
			verdictOnly := calcAttackScore(observer, target, items.Item{}, 0, combatContext{sourceSight: verdict})
			if math.Abs(verdictOnly-clean) > 1e-9 {
				t.Fatalf("verdict %v alone moved the score: %v, want clean %v", verdict, verdictOnly, clean)
			}

			if tc.name == "dark, infrared only" && verdict != messaging.SightShapes {
				t.Fatalf("infrared in the dark must resolve to SightShapes, got %v", verdict)
			}
		})
	}
}

// sightRampCases are the ramp points both scoring tests below pin, with the
// multiplier HARDCODED at the shipped caps (DarknessCombatPenalty 0.80,
// DazzleCap 0.80) rather than recomputed, so a test cannot agree with a
// broken SightScoreMultiplier by construction. dark 0.52 is a dim street at
// night for normal eyes (light 37); bright 0.6 is light 90 for normal eyes.
var sightRampCases = []struct {
	name         string
	dark, bright float64
	wantMult     float64
}{
	{name: "comfortable", wantMult: 1.0},
	{name: "dark 0.52", dark: 0.52, wantMult: 0.896},
	{name: "bright 0.6", bright: 0.6, wantMult: 0.88},
	{name: "dark 1 (blind edge)", dark: 1, wantMult: 0.80},
	{name: "bright 1 (dazzle cap)", bright: 1, wantMult: 0.80},
}

func pinSightRampCaps(t *testing.T) {
	t.Helper()
	cfg := configs.GetConfig()
	cfg.Balance.DarknessCombatPenalty = 0.80
	cfg.Balance.DazzleCap = 0.80
	configs.SetConfigForTest(t, cfg)
}

// TestAttackScoreRidesTheSightRamp pins plan 5b's attack side: the ramp
// replaces the old flat band. A shapes-only attacker used to take a flat
// 0.90 whatever the light; now it pays by its distance from the comfortable
// band, a dazzled attacker pays too, and a comfortable one pays nothing.
// Every case sets sourceSight to SightFull, so any move in the score comes
// from the comfort fields alone.
func TestAttackScoreRidesTheSightRamp(t *testing.T) {
	pinSightRampCaps(t)

	observer := characters.New()
	target := characters.New()
	clean := calcAttackScore(observer, target, items.Item{}, 0, combatContext{sourceSight: messaging.SightFull})
	if clean <= 0 {
		t.Fatalf("fixture guard: clean attack score %v must be positive or every ratio passes", clean)
	}

	for _, tc := range sightRampCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := combatContext{sourceSight: messaging.SightFull, sourceDark: tc.dark, sourceBright: tc.bright}
			got := calcAttackScore(observer, target, items.Item{}, 0, ctx)
			want := clean * tc.wantMult
			if math.Abs(got-want) > 1e-9 {
				t.Fatalf("attack score = %v, want %v (clean %v x %v)", got, want, clean, tc.wantMult)
			}

			// The defender's comfort must not reach the attack score.
			other := combatContext{sourceSight: messaging.SightFull, targetDark: tc.dark, targetBright: tc.bright}
			if got := calcAttackScore(observer, target, items.Item{}, 0, other); math.Abs(got-clean) > 1e-9 {
				t.Fatalf("target comfort moved the attack score: %v, want clean %v", got, clean)
			}
		})
	}
}

// TestDefenceScoreRidesTheSightRamp pins the defence side through the real
// defence core, capturing the dodge entry's score as handed to the contest.
func TestDefenceScoreRidesTheSightRamp(t *testing.T) {
	pinDefenceAdmissionConfig(t)
	pinSightRampCaps(t)

	capture := func(ctx combatContext) float64 {
		attacker, defender := defenceAdmissionCharacters()
		var captured float64
		found := false
		runner := func(atkScore float64, entries []contest.Entry) contest.Result {
			for _, e := range entries {
				if e.Name == string(combatvocab.DefenceDodge) {
					captured = e.Score
					found = true
				}
			}
			return deterministicDefenceResult(t, atkScore, entries, combatvocab.DefenceDodge, 0, entries[0].Score)
		}
		runBestOfAllDefenseWithRunner(&AttackResult{}, attacker, defender,
			[]combatvocab.Defence{combatvocab.DefenceDodge}, 100, false, ctx, runner)
		if !found {
			t.Fatal("fixture guard: the runner never saw a dodge entry")
		}
		return captured
	}

	full := combatContext{sourceSight: messaging.SightFull, targetSight: messaging.SightFull}
	clean := capture(full)
	if clean <= 0 {
		t.Fatalf("fixture guard: clean defence score %v must be positive or every ratio passes", clean)
	}

	for _, tc := range sightRampCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := full
			ctx.targetDark, ctx.targetBright = tc.dark, tc.bright
			if got, want := capture(ctx), clean*tc.wantMult; math.Abs(got-want) > 1e-9 {
				t.Fatalf("defence score = %v, want %v (clean %v x %v)", got, want, clean, tc.wantMult)
			}

			// The attacker's comfort must not reach the defence score.
			other := full
			other.sourceDark, other.sourceBright = tc.dark, tc.bright
			if got := capture(other); math.Abs(got-clean) > 1e-9 {
				t.Fatalf("source comfort moved the defence score: %v, want clean %v", got, clean)
			}
		})
	}
}
