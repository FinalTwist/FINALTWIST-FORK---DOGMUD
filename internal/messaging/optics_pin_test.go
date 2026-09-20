package messaging

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
)

// opticsCase is one observer state crossed with one room state.
type opticsCase struct {
	name             string
	blind, asleep    bool
	nightVision      bool
	infraredVision   bool
	lit              bool
	wantClearly      bool
	wantImpairedOnly bool
	wantShapes       bool
}

// newOpticsObserver builds a *characters.Character carrying the flags an
// opticsCase asks for. It reuses the fixtures participant_sight_test.go
// already built for this package: sightChar seeds the condition-flag
// registry and applies NightVision/InfraredVision/Sleeping, and setBlinded
// (from predicates_test.go) drives blindness through the real Perception
// state machine rather than a condition flag.
func newOpticsObserver(t *testing.T, tc opticsCase) *characters.Character {
	t.Helper()
	var flags []conditions.Flag
	if tc.nightVision {
		flags = append(flags, conditions.NightVision)
	}
	if tc.infraredVision {
		flags = append(flags, conditions.InfraredVision)
	}
	if tc.asleep {
		flags = append(flags, conditions.Sleeping)
	}
	c := sightChar(t, flags...)
	if tc.blind {
		setBlinded(t, c)
	}
	return c
}

// newOpticsRoom builds a RoomVisibility at the given light level, reusing
// sightLight (participant_sight_test.go) rather than a real *rooms.Room --
// Room.GetVisibility() calls into the biome registry, which is not loaded
// in unit-test context.
func newOpticsRoom(t *testing.T, lit bool) RoomVisibility {
	t.Helper()
	if lit {
		return sightLight(1)
	}
	return sightLight(0)
}

// TestOpticsTruthTable pins what the three sight predicates answer TODAY, for
// every combination that distinguishes them. It is the net for M4d's
// dependency inversion: the predicates are about to be redefined over
// ParticipantSight, and byte-identical goldens cannot prove a boolean's
// semantics.
//
// Two rows here are the ones worth reading slowly, because an earlier draft of
// the M4d spec got both wrong by reasoning from the function NAMES:
//
//   - infrared in the dark: CanSeeSightImpairedOnly is FALSE. It consults
//     NightVision only. Widening it to "shapes or better" would silently
//     remove DarknessCombatPenalty from every infrared character.
//   - asleep in a lit room: CanSeeSightImpairedOnly is TRUE. It has no
//     attention test on purpose, so combat does not double a sleeper's
//     disadvantage (they are already auto-crit).
func TestOpticsTruthTable(t *testing.T) {
	cases := []opticsCase{
		{name: "lit, ordinary", lit: true,
			wantClearly: true, wantImpairedOnly: true, wantShapes: true},
		{name: "dark, no vision",
			wantClearly: false, wantImpairedOnly: false, wantShapes: false},
		{name: "dark, nightvision", nightVision: true,
			wantClearly: true, wantImpairedOnly: true, wantShapes: true},
		{name: "dark, infrared only", infraredVision: true,
			wantClearly: false, wantImpairedOnly: false, wantShapes: true},
		{name: "blind in a lit room", blind: true, lit: true,
			wantClearly: false, wantImpairedOnly: false, wantShapes: false},
		{name: "blind with infrared", blind: true, infraredVision: true,
			wantClearly: false, wantImpairedOnly: false, wantShapes: false},
		{name: "asleep in a lit room", asleep: true, lit: true,
			wantClearly: false, wantImpairedOnly: true, wantShapes: false},
		{name: "asleep with infrared in the dark", asleep: true, infraredVision: true,
			wantClearly: false, wantImpairedOnly: false, wantShapes: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obs := newOpticsObserver(t, tc)
			room := newOpticsRoom(t, tc.lit)

			if got := CanSeeClearly(obs, room); got != tc.wantClearly {
				t.Errorf("CanSeeClearly = %v, want %v", got, tc.wantClearly)
			}
			if got := CanSeeSightImpairedOnly(obs, room); got != tc.wantImpairedOnly {
				t.Errorf("CanSeeSightImpairedOnly = %v, want %v", got, tc.wantImpairedOnly)
			}
			if got := CanSeeShapes(obs, room); got != tc.wantShapes {
				t.Errorf("CanSeeShapes = %v, want %v", got, tc.wantShapes)
			}
		})
	}
}
