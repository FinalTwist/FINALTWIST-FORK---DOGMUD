package usercommands

import (
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// A stacking spec (conditions.Stacking) can only be added through
// AddConditionMagnitude, which supplies the rounds and amount a stack needs.
// Conditions.AddCondition and Conditions.AddConditionScaled now refuse one outright (see
// internal/conditions review fixes), and admin.setcondition.go's `setcondition <id>`
// queues its add through exactly that door (UserRecord.AddCondition / Mob.AddCondition,
// both events.Condition with no magnitude or triggers). Before this fix the command
// told the admin the condition was "applied" regardless, which was a lie: the
// queued add was silently refused downstream and nothing landed.
const adminConditionStackingTestId = 9401

// seedAdminConditionStackingUser builds a minimal, self-contained fixture (own
// user, room and condition registry) rather than reusing seedAllRegistries, so
// adding the one stacking spec this test needs cannot disturb any other
// test's shared condition ids.
func seedAdminConditionStackingUser(t *testing.T) (*users.UserRecord, *rooms.Room, func()) {
	t.Helper()

	cleanupConditions := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		adminConditionStackingTestId: {
			ConditionId: adminConditionStackingTestId, Name: "Test Gash",
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
		cleanupConditions()
	}
}

func TestAdminSetCondition_RefusesAStackingSpecOnAPlayer(t *testing.T) {
	user, room, cleanup := seedAdminConditionStackingUser(t)
	defer cleanup()

	handled, err := SetCondition(strconv.Itoa(adminConditionStackingTestId), user, room, 0)
	if err != nil || !handled {
		t.Fatalf("command errored: handled=%v err=%v", handled, err)
	}

	msgs := strings.Join(events.DrainQueuedMessagesForTest(user.UserId), "\n")
	if strings.Contains(msgs, "applied to") {
		t.Errorf("must not claim the condition applied; got:\n%s", msgs)
	}
	if !strings.Contains(msgs, "stack") {
		t.Errorf("no stacking refusal sent; got:\n%s", msgs)
	}
	if user.Character.Conditions.HasCondition(adminConditionStackingTestId) {
		t.Error("a refused add must hold nothing")
	}
}

// seedAdminConditionStackingMob mirrors seedAdminConditionStackingUser for the MOB
// branch of the same refusal (admin.setcondition.go's second `conditionSpec.IsStacking()`
// guard, ~line 150). It needs its own room registered with the rooms
// package, because the len(args)>=2 path in SetCondition() re-resolves the room via
// rooms.LoadRoom(user.Character.RoomId) rather than using the room argument
// the command was called with. The mob instance is seeded and placed in that
// room the same way internal/usercommands/attack_test.go builds a mob
// fixture (mobs.SeedMobsForTest + room.AddMob).
func seedAdminConditionStackingMob(t *testing.T) (*users.UserRecord, *rooms.Room, *mobs.Mob, func()) {
	t.Helper()

	cleanupConditions := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		adminConditionStackingTestId: {
			ConditionId: adminConditionStackingTestId, Name: "Test Gash",
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
			Name:       "Stackratling",
			RoomId:     1,
			Health:     10,
			Conditions: conditions.New(),
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
		cleanupConditions()
	}
}

func TestAdminSetCondition_RefusesAStackingSpecOnAMob(t *testing.T) {
	user, room, mob, cleanup := seedAdminConditionStackingMob(t)
	defer cleanup()

	handled, err := SetCondition("stackratling "+strconv.Itoa(adminConditionStackingTestId), user, room, 0)
	if err != nil || !handled {
		t.Fatalf("command errored: handled=%v err=%v", handled, err)
	}

	msgs := strings.Join(events.DrainQueuedMessagesForTest(user.UserId), "\n")
	if strings.Contains(msgs, "applied to") {
		t.Errorf("must not claim the condition applied; got:\n%s", msgs)
	}
	if !strings.Contains(msgs, "stack") {
		t.Errorf("no stacking refusal sent; got:\n%s", msgs)
	}
	if mob.Character.Conditions.HasCondition(adminConditionStackingTestId) {
		t.Error("a refused add must hold nothing")
	}
}
