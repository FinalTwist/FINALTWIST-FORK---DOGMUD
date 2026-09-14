package usercommands

import (
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// A stacking spec (buffs.Stacking) can only be added through
// AddBuffMagnitude, which supplies the rounds and amount a stack needs.
// Buffs.AddBuff and Buffs.AddBuffScaled now refuse one outright (see
// internal/conditions review fixes), and admin.buff.go's `buff <id>` queues its
// add through exactly that door (UserRecord.AddBuff / Mob.AddBuff, both
// events.Buff with no magnitude or triggers). Before this fix the command
// told the admin the buff was "applied" regardless, which was a lie: the
// queued add was silently refused downstream and nothing landed.
const adminBuffStackingTestId = 9401

// seedAdminBuffStackingUser builds a minimal, self-contained fixture (own
// user, room and buff registry) rather than reusing seedAllRegistries, so
// adding the one stacking spec this test needs cannot disturb any other
// test's shared buff ids.
func seedAdminBuffStackingUser(t *testing.T) (*users.UserRecord, *rooms.Room, func()) {
	t.Helper()

	cleanupBuffs := conditions.SeedBuffsForTest(map[int]*conditions.BuffSpec{
		adminBuffStackingTestId: {
			BuffId: adminBuffStackingTestId, Name: "Test Gash",
			TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 4,
			Flags:    []conditions.Flag{conditions.Bleeding, conditions.Stacking},
			TickPool: "health", TickFromMagnitude: true,
		},
	})

	u := users.NewTestUser(9441, "stackadmin", "Stackadmin", uint64(9441))
	u.Role = users.RoleAdmin
	u.Character.RoomId = 1
	cleanupUsers := users.SeedUsersForTest(map[int]*users.UserRecord{9441: u})

	room := &rooms.Room{RoomId: 1}

	events.DrainQueuedMessagesForTest(u.UserId)

	return u, room, func() {
		events.DrainQueuedMessagesForTest(u.UserId)
		cleanupUsers()
		cleanupBuffs()
	}
}

func TestAdminBuff_RefusesAStackingSpecOnAPlayer(t *testing.T) {
	user, room, cleanup := seedAdminBuffStackingUser(t)
	defer cleanup()

	handled, err := Buff(strconv.Itoa(adminBuffStackingTestId), user, room, 0)
	if err != nil || !handled {
		t.Fatalf("command errored: handled=%v err=%v", handled, err)
	}

	msgs := strings.Join(events.DrainQueuedMessagesForTest(user.UserId), "\n")
	if strings.Contains(msgs, "applied to") {
		t.Errorf("must not claim the buff applied; got:\n%s", msgs)
	}
	if !strings.Contains(msgs, "stack") {
		t.Errorf("no stacking refusal sent; got:\n%s", msgs)
	}
	if user.Character.Buffs.HasBuff(adminBuffStackingTestId) {
		t.Error("a refused add must hold nothing")
	}
}

// seedAdminBuffStackingMob mirrors seedAdminBuffStackingUser for the MOB
// branch of the same refusal (admin.buff.go's second `buffSpec.IsStacking()`
// guard, ~line 150). It needs its own room registered with the rooms
// package, because the len(args)>=2 path in Buff() re-resolves the room via
// rooms.LoadRoom(user.Character.RoomId) rather than using the room argument
// the command was called with. The mob instance is seeded and placed in that
// room the same way internal/usercommands/attack_test.go builds a mob
// fixture (mobs.SeedMobsForTest + room.AddMob).
func seedAdminBuffStackingMob(t *testing.T) (*users.UserRecord, *rooms.Room, *mobs.Mob, func()) {
	t.Helper()

	cleanupBuffs := conditions.SeedBuffsForTest(map[int]*conditions.BuffSpec{
		adminBuffStackingTestId: {
			BuffId: adminBuffStackingTestId, Name: "Test Gash",
			TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 4,
			Flags:    []conditions.Flag{conditions.Bleeding, conditions.Stacking},
			TickPool: "health", TickFromMagnitude: true,
		},
	})

	u := users.NewTestUser(9441, "stackadmin", "Stackadmin", uint64(9441))
	u.Role = users.RoleAdmin
	u.Character.RoomId = 1
	cleanupUsers := users.SeedUsersForTest(map[int]*users.UserRecord{9441: u})

	room := &rooms.Room{RoomId: 1}
	cleanupRooms := rooms.SeedRoomsForTest(map[int]*rooms.Room{1: room}, nil)

	mob := &mobs.Mob{
		MobId:      1,
		InstanceId: 9442,
		HomeRoomId: 1,
		Character: characters.Character{
			Name:   "Stackratling",
			RoomId: 1,
			Health: 10,
			Buffs:  conditions.New(),
		},
	}
	cleanupMobs := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{9442: mob})
	room.AddMob(9442)

	events.DrainQueuedMessagesForTest(u.UserId)

	return u, room, mob, func() {
		events.DrainQueuedMessagesForTest(u.UserId)
		room.RemoveMob(9442)
		cleanupMobs()
		cleanupRooms()
		cleanupUsers()
		cleanupBuffs()
	}
}

func TestAdminBuff_RefusesAStackingSpecOnAMob(t *testing.T) {
	user, room, mob, cleanup := seedAdminBuffStackingMob(t)
	defer cleanup()

	handled, err := Buff("stackratling "+strconv.Itoa(adminBuffStackingTestId), user, room, 0)
	if err != nil || !handled {
		t.Fatalf("command errored: handled=%v err=%v", handled, err)
	}

	msgs := strings.Join(events.DrainQueuedMessagesForTest(user.UserId), "\n")
	if strings.Contains(msgs, "applied to") {
		t.Errorf("must not claim the buff applied; got:\n%s", msgs)
	}
	if !strings.Contains(msgs, "stack") {
		t.Errorf("no stacking refusal sent; got:\n%s", msgs)
	}
	if mob.Character.Buffs.HasBuff(adminBuffStackingTestId) {
		t.Error("a refused add must hold nothing")
	}
}
