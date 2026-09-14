package usercommands

import (
	"strconv"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// A stacking spec (buffs.Stacking) can only be added through
// AddBuffMagnitude, which supplies the rounds and amount a stack needs.
// Buffs.AddBuff and Buffs.AddBuffScaled now refuse one outright (see
// internal/buffs review fixes), and admin.buff.go's `buff <id>` queues its
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

	cleanupBuffs := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		adminBuffStackingTestId: {
			BuffId: adminBuffStackingTestId, Name: "Test Gash",
			TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 4,
			Flags:    []buffs.Flag{buffs.Bleeding, buffs.Stacking},
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
