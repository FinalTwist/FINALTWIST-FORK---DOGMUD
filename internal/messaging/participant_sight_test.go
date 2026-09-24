package messaging

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// sightLight is a RoomVisibility with a fixed light level on the graded
// scale. sightLightDark (0) is pitch black, comfortably below
// LightBlindBelow's default of 25. sightLightLit (100, the scale's top) is
// comfortably above LightDimBelow's default of 50, so it reads as fully lit
// regardless of exactly where those two knobs sit. These tests exercise
// blindness and the NightVision / InfraredVision shortcuts, not the band
// boundaries themselves.
//
// TestParticipantSightReadsTheBands (predicates_test.go, same package)
// reuses this type directly -- sightLight(n) for an arbitrary n, not just
// the two named constants -- to pin behaviour ACROSS the graded scale, and
// pins the config knobs it depends on rather than trusting the Go defaults
// these two constants lean on.
type sightLight int

const (
	sightLightDark sightLight = 0
	sightLightLit  sightLight = 100
)

func (l sightLight) LightLevel() int { return int(l) }

const (
	sightInfraredConditionId = 9101
	sightNightConditionId    = 9102
	sightSleepConditionId    = 9103
)

// sightChar returns a fresh character carrying the given test flags. The three
// flag conditions are seeded once per test, so applying one never replaces another.
//
// GRADED LIGHTING PLAN 2. The infrared fixture declares an explicit
// infra_reach (matching shipped condition 85's authored 30, see
// _datafiles/world/dogmud/conditions/85-infraredvision.yaml), not a bare
// flag. A bare InfraredVision flag reads reach 0 by design
// (Character.InfraReach, internal/characters/vision.go: only nightvision
// defaults on a bare flag), so without a declared reach these fixtures would
// stop demonstrating infrared at all and would collapse into the same
// SightNone the no-vision fixture already covers. The nightvision fixture is
// deliberately left as a bare flag: under the window model a shifted window
// is still blind below its floor at light 0 regardless of strength, so a
// bare flag (falling back to LightDefaultVisionStrength) already proves the
// point the "dark with night vision" test cases below now make.
func sightChar(t *testing.T, flags ...conditions.Flag) *characters.Character {
	t.Helper()
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		sightInfraredConditionId: {
			ConditionId: sightInfraredConditionId,
			Name:        "Test Infrared",
			Flags:       []conditions.Flag{conditions.InfraredVision},
			Effects:     map[conditions.EffectKind]conditions.EffectValue{conditions.EffectInfraReach: {Literal: 30}},
		},
		sightNightConditionId: {ConditionId: sightNightConditionId, Name: "Test Night", Flags: []conditions.Flag{conditions.NightVision}},
		sightSleepConditionId: {ConditionId: sightSleepConditionId, Name: "Test Sleep", Flags: []conditions.Flag{conditions.Sleeping}},
	}))
	c := newChar(t)
	ids := map[conditions.Flag]int{
		conditions.InfraredVision: sightInfraredConditionId,
		conditions.NightVision:    sightNightConditionId,
		conditions.Sleeping:       sightSleepConditionId,
	}
	for _, f := range flags {
		if err := c.AddCondition(ids[f], true); err != nil {
			t.Fatalf("applying %s: %v", f, err)
		}
	}
	return c
}

func TestParticipantSight(t *testing.T) {
	cases := []struct {
		name  string
		light sightLight
		flags []conditions.Flag
		blind bool
		want  SightDecision
	}{
		{name: "lit room", light: sightLightLit, want: SightFull},
		{name: "dark room", light: sightLightDark, want: SightNone},
		// GRADED LIGHTING PLAN 2: a shifted window is still blind below its
		// floor (windowFloor = 1, internal/messaging/window.go). At light 0
		// even the shift cap (24) only pulls the blind edge down to 1, which
		// light 0 still fails, so nightvision alone can never produce
		// anything but SightNone in a pitch dark room. Only infra reach
		// reads past the floor.
		{name: "dark with night vision", light: sightLightDark, flags: []conditions.Flag{conditions.NightVision}, want: SightNone},
		{name: "dark with infrared", light: sightLightDark, flags: []conditions.Flag{conditions.InfraredVision}, want: SightShapes},
		{name: "blinded in a lit room", light: sightLightLit, blind: true, want: SightNone},
		{name: "blinded with infrared in the dark", light: sightLightDark, flags: []conditions.Flag{conditions.InfraredVision}, blind: true, want: SightNone},
		// Sleep is NOT a factor: a sleeper struck in a lit room is told what hit them.
		{name: "sleeping in a lit room", light: sightLightLit, flags: []conditions.Flag{conditions.Sleeping}, want: SightFull},
		{name: "sleeping with infrared in the dark", light: sightLightDark, flags: []conditions.Flag{conditions.Sleeping, conditions.InfraredVision}, want: SightShapes},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := sightChar(t, tc.flags...)
			if tc.blind {
				setBlinded(t, c)
			}
			if got := ParticipantSight(c, tc.light); got != tc.want {
				t.Fatalf("ParticipantSight = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParticipantSight_NilObserverSeesFully(t *testing.T) {
	if got := ParticipantSight(nil, sightLightDark); got != SightFull {
		t.Fatalf("nil observer = %v, want SightFull, matching the other predicates", got)
	}
}

// Infrared from a mutation counts the same as from a condition: the predicate reads
// HasFlagFromAnySource, and a change to HasConditionFlag would silently drop it.
func TestParticipantSight_InfraredFromAMutation(t *testing.T) {
	t.Cleanup(mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"test-heat-pits": {MutationId: "test-heat-pits", Name: "Test Heat Pits",
			Pros: []mutations.MutationEffect{{Type: "flag", Target: string(conditions.InfraredVision), Value: 1}}},
	}))
	c := newChar(t)
	c.Mutations = map[string]int{"test-heat-pits": 1}
	if got := ParticipantSight(c, sightLightDark); got != SightShapes {
		t.Fatalf("infrared from a mutation in the dark = %v, want SightShapes", got)
	}
}

func TestParticipantSight_NilRoomIsLit(t *testing.T) {
	if got := ParticipantSight(newChar(t), nil); got != SightFull {
		t.Fatalf("nil room = %v, want SightFull, matching the other predicates", got)
	}
}
