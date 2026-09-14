package messaging

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// sightLight is a RoomVisibility with a fixed light level: 0 dark, 1 lit.
type sightLight int

func (l sightLight) GetVisibility() int { return int(l) }

const (
	sightInfraredConditionId = 9101
	sightNightConditionId    = 9102
	sightSleepConditionId    = 9103
)

// sightChar returns a fresh character carrying the given test flags. The three
// flag conditions are seeded once per test, so applying one never replaces another.
func sightChar(t *testing.T, flags ...conditions.Flag) *characters.Character {
	t.Helper()
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		sightInfraredConditionId: {ConditionId: sightInfraredConditionId, Name: "Test Infrared", Flags: []conditions.Flag{conditions.InfraredVision}},
		sightNightConditionId:    {ConditionId: sightNightConditionId, Name: "Test Night", Flags: []conditions.Flag{conditions.NightVision}},
		sightSleepConditionId:    {ConditionId: sightSleepConditionId, Name: "Test Sleep", Flags: []conditions.Flag{conditions.Sleeping}},
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
		{name: "lit room", light: 1, want: SightFull},
		{name: "dark room", light: 0, want: SightNone},
		{name: "dark with night vision", light: 0, flags: []conditions.Flag{conditions.NightVision}, want: SightFull},
		{name: "dark with infrared", light: 0, flags: []conditions.Flag{conditions.InfraredVision}, want: SightShapes},
		{name: "blinded in a lit room", light: 1, blind: true, want: SightNone},
		{name: "blinded with infrared in the dark", light: 0, flags: []conditions.Flag{conditions.InfraredVision}, blind: true, want: SightNone},
		// Sleep is NOT a factor: a sleeper struck in a lit room is told what hit them.
		{name: "sleeping in a lit room", light: 1, flags: []conditions.Flag{conditions.Sleeping}, want: SightFull},
		{name: "sleeping with infrared in the dark", light: 0, flags: []conditions.Flag{conditions.Sleeping, conditions.InfraredVision}, want: SightShapes},
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
	if got := ParticipantSight(nil, sightLight(0)); got != SightFull {
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
	if got := ParticipantSight(c, sightLight(0)); got != SightShapes {
		t.Fatalf("infrared from a mutation in the dark = %v, want SightShapes", got)
	}
}

func TestParticipantSight_NilRoomIsLit(t *testing.T) {
	if got := ParticipantSight(newChar(t), nil); got != SightFull {
		t.Fatalf("nil room = %v, want SightFull, matching the other predicates", got)
	}
}
