package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
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

	RelocateMob(m, fromRoom, "north", toRoom, false)

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

// relocateWatchers seeds a player in each room of a RelocateMob move and
// returns the messages each one received, draining them.
func relocateWatchers(t *testing.T, sneaking bool) (fromMsgs, toMsgs []string) {
	t.Helper()
	const from, to, instId = 99421, 99422, 98421
	const fromWatcher, toWatcher = 99431, 99432
	cleanupRooms := rooms.SeedRoomsForTest(map[int]*rooms.Room{
		from: {RoomId: from, Zone: "test", Exits: map[string]exit.RoomExit{"north": {RoomId: to}}},
		to:   {RoomId: to, Zone: "test", Exits: map[string]exit.RoomExit{"south": {RoomId: from}}},
	}, map[string]*rooms.ZoneConfig{})
	defer cleanupRooms()

	fw := users.NewTestUser(fromWatcher, "fromwatch", "Fromwatch", 0)
	fw.Character.RoomId = from
	tw := users.NewTestUser(toWatcher, "towatch", "Towatch", 0)
	tw.Character.RoomId = to
	cleanupUsers := users.SeedUsersForTest(map[int]*users.UserRecord{fromWatcher: fw, toWatcher: tw})
	defer cleanupUsers()

	m := &mobs.Mob{InstanceId: instId, Character: *characters.New()}
	m.Character.Name = "Prowler"
	m.Character.RoomId = from
	mobs.SetInstanceForTest(instId, m)
	defer mobs.SetInstanceForTest(instId, nil)

	fromRoom, toRoom := rooms.LoadRoom(from), rooms.LoadRoom(to)
	fromRoom.AddPlayer(fromWatcher)
	toRoom.AddPlayer(toWatcher)
	fromRoom.AddMob(instId)
	events.DrainQueuedMessagesForTest(fromWatcher)
	events.DrainQueuedMessagesForTest(toWatcher)

	RelocateMob(m, fromRoom, "north", toRoom, sneaking)

	return events.DrainQueuedMessagesForTest(fromWatcher), events.DrainQueuedMessagesForTest(toWatcher)
}

// Owner ruling D1: a sneaking mob's step is not announced, as a sneaking
// player's never was. The walking control proves the watchers can hear one.
func TestRelocateMob_ASneakingMobMovesUnannounced(t *testing.T) {
	fromMsgs, toMsgs := relocateWatchers(t, false)
	if len(fromMsgs) == 0 || len(toMsgs) == 0 {
		t.Fatalf("control: a walking mob must be announced to both rooms, got from=%q to=%q", fromMsgs, toMsgs)
	}

	fromMsgs, toMsgs = relocateWatchers(t, true)
	if len(fromMsgs) != 0 || len(toMsgs) != 0 {
		t.Errorf("a sneaking mob was announced: from=%q to=%q", fromMsgs, toMsgs)
	}
}
