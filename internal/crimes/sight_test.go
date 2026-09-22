package crimes

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
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

	if got := lit.LightLevel(); got <= rooms.LightDark {
		t.Fatalf("default-biome room light = %d, want > %d (lit)", got, rooms.LightDark)
	}
	if got := dark.LightLevel(); got != rooms.LightDark {
		t.Fatalf("cave room light = %d, want %d (unlit)", got, rooms.LightDark)
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

// TestWitnessesInRoom_SplitsBySight puts three same-faction mobs, one per
// sight tier, in one unlit room and checks that WitnessesInRoom sorts them
// into Identifying / ShapesOnly / not-a-witness-at-all rather than treating
// presence in the room as sight.
func TestWitnessesInRoom_SplitsBySight(t *testing.T) {
	setupTestCrimes(t)
	setupFactionsForCrimesTest(t)

	room := &rooms.Room{RoomId: 468, Biome: "cave"}

	plainMob := &mobs.Mob{MobId: 900, InstanceId: 401, Groups: []string{"thornwall_citizens"}}
	plainMob.Character = *newCharWithCondition(t, "plain mob", 0)
	mobs.SetInstanceForTest(401, plainMob)
	defer mobs.SetInstanceForTest(401, nil)
	room.AddMob(401)

	infraredMob := &mobs.Mob{MobId: 901, InstanceId: 402, Groups: []string{"thornwall_citizens"}}
	infraredMob.Character = *newCharWithCondition(t, "infrared mob", 85)
	mobs.SetInstanceForTest(402, infraredMob)
	defer mobs.SetInstanceForTest(402, nil)
	room.AddMob(402)

	nightvisionMob := &mobs.Mob{MobId: 902, InstanceId: 403, Groups: []string{"thornwall_citizens"}}
	nightvisionMob.Character = *newCharWithCondition(t, "nightvision mob", 29)
	mobs.SetInstanceForTest(403, nightvisionMob)
	defer mobs.SetInstanceForTest(403, nil)
	room.AddMob(403)

	got := WitnessesInRoom([]string{"thornwall_citizens"}, room, 0)

	if len(got.Identifying) != 1 || got.Identifying[0] != 403 {
		t.Errorf("Identifying = %v, want [403] (nightvision mob)", got.Identifying)
	}
	if len(got.ShapesOnly) != 1 || got.ShapesOnly[0] != 402 {
		t.Errorf("ShapesOnly = %v, want [402] (infrared mob)", got.ShapesOnly)
	}
	total := len(got.Identifying) + len(got.ShapesOnly)
	if total != 2 {
		t.Errorf("total witnesses = %d, want 2 (plain mob in the dark is not a witness at all)", total)
	}
}

// TestIdentifiedPerp_ShapesOnlyIsUnknown pins the owner's ruling in one
// assertion: a shapes-only witness never yields a named perpetrator.
func TestIdentifiedPerp_ShapesOnlyIsUnknown(t *testing.T) {
	got := IdentifiedPerp(17, Witnesses{ShapesOnly: []int{101}})
	if got.Type != PerpUnknown {
		t.Errorf("shapes-only witness: got type %q, want unknown", got.Type)
	}
}

// TestWitnessesInRoom_SleeperInLitRoomIsNotAWitness proves the sleep gate
// reaches the crime path, not just the sight predicates it is built on. A
// sleeping mob in a fully lit room still sees nothing.
func TestWitnessesInRoom_SleeperInLitRoomIsNotAWitness(t *testing.T) {
	setupTestCrimes(t)
	setupFactionsForCrimesTest(t)

	room := &rooms.Room{RoomId: 467}

	sleeper := &mobs.Mob{MobId: 903, InstanceId: 501, Groups: []string{"thornwall_citizens"}}
	sleeper.Character = *newCharWithCondition(t, "sleeping mob", 15)
	mobs.SetInstanceForTest(501, sleeper)
	defer mobs.SetInstanceForTest(501, nil)
	room.AddMob(501)

	got := WitnessesInRoom([]string{"thornwall_citizens"}, room, 0)

	for _, id := range got.Identifying {
		if id == 501 {
			t.Errorf("sleeping mob in a lit room appeared in Identifying: %v", got.Identifying)
		}
	}
	for _, id := range got.ShapesOnly {
		if id == 501 {
			t.Errorf("sleeping mob in a lit room appeared in ShapesOnly: %v", got.ShapesOnly)
		}
	}
}
