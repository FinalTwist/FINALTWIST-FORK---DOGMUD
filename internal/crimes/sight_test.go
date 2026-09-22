package crimes

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// newCharWithCondition builds a character the way production does. A struct
// literal panics on AddCondition with "assignment to entry in nil map",
// because characters.New() is what allocates the condition maps.
func newCharWithCondition(t *testing.T, name string, conditionId int) *characters.Character {
	t.Helper()
	c := characters.New()
	c.Name = name
	if conditionId > 0 {
		if err := c.AddCondition(conditionId, true); err != nil {
			t.Fatalf("AddCondition(%d): %v", conditionId, err)
		}
	}
	return c
}

// TestSightTiersBehaveAsWitnessGateExpects pins the behaviour the witness gate
// is built on, in this package's own test binary. If TestMain stops loading
// biomes or conditions, this fails here rather than somewhere confusing.
func TestSightTiersBehaveAsWitnessGateExpects(t *testing.T) {
	lit := &rooms.Room{RoomId: 467}
	dark := &rooms.Room{RoomId: 468, Biome: "cave"}

	if got := lit.GetVisibility(); got < 1 {
		t.Fatalf("default-biome room visibility = %d, want >= 1 (lit)", got)
	}
	if got := dark.GetVisibility(); got != 0 {
		t.Fatalf("cave room visibility = %d, want 0 (unlit)", got)
	}

	tests := []struct {
		name        string
		conditionId int
		room        *rooms.Room
		wantClear   bool
		wantShapes  bool
	}{
		{"plain mob in a lit room", 0, lit, true, true},
		{"plain mob in the dark", 0, dark, false, false},
		{"nightvision mob in the dark", 29, dark, true, true},
		{"infrared mob in the dark", 85, dark, false, true},
		{"sleeping mob in a LIT room", 15, lit, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ch := newCharWithCondition(t, tc.name, tc.conditionId)
			if got := messaging.CanSeeClearly(ch, tc.room); got != tc.wantClear {
				t.Errorf("CanSeeClearly = %v, want %v", got, tc.wantClear)
			}
			if got := messaging.CanSeeShapes(ch, tc.room); got != tc.wantShapes {
				t.Errorf("CanSeeShapes = %v, want %v", got, tc.wantShapes)
			}
		})
	}
}
