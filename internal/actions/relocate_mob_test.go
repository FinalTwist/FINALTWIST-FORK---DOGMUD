package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// RelocateMob is the mob's move with no gate and no charge: walking (after
// its gates and, from 4b, its charge) and a successful flee both end in it.
func TestRelocateMob_MovesTheMobBetweenRooms(t *testing.T) {
	const from, to, instId = 99411, 99412, 98411
	cleanup := rooms.SeedRoomsForTest(map[int]*rooms.Room{
		from: {RoomId: from, Zone: "test", Exits: map[string]exit.RoomExit{"north": {RoomId: to}}},
		to:   {RoomId: to, Zone: "test", Exits: map[string]exit.RoomExit{"south": {RoomId: from}}},
	}, map[string]*rooms.ZoneConfig{})
	defer cleanup()

	m := &mobs.Mob{InstanceId: instId, Character: *characters.New()}
	m.Character.Name = "Walker"
	m.Character.RoomId = from
	mobs.SetInstanceForTest(instId, m)
	defer mobs.SetInstanceForTest(instId, nil)

	fromRoom, toRoom := rooms.LoadRoom(from), rooms.LoadRoom(to)
	fromRoom.AddMob(instId)

	RelocateMob(m, fromRoom, "north", toRoom)

	if contains := func(ids []int) bool {
		for _, id := range ids {
			if id == instId {
				return true
			}
		}
		return false
	}; contains(fromRoom.GetMobs(rooms.FindAll)) || !contains(toRoom.GetMobs(rooms.FindAll)) {
		t.Fatalf("mob not moved: from=%v to=%v", fromRoom.GetMobs(rooms.FindAll), toRoom.GetMobs(rooms.FindAll))
	}
	if m.Character.RoomId != to {
		t.Fatalf("mob RoomId = %d, want %d", m.Character.RoomId, to)
	}
}
