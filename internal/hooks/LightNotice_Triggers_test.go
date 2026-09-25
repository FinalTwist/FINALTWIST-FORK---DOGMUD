package hooks

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/lightnotice"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

const lightNoticeDir = "../../_datafiles/world/dogmud/narration/light-notices"

// lampRooms pins rooms 1 and 2 of the seeded registries to sky-less lamp light.
func lampRooms(t *testing.T, lamp1, lamp2 int) (*rooms.Room, *rooms.Room) {
	t.Helper()
	r1, r2 := rooms.LoadRoom(1), rooms.LoadRoom(2)
	require.NotNil(t, r1)
	require.NotNil(t, r2)
	r1.SkyLight, r1.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(lamp1)
	r2.SkyLight, r2.Lamp = rooms.SkyLightPtr(0), rooms.LampPtr(lamp2)
	return r1, r2
}

func loadLightNotices(t *testing.T) {
	t.Helper()
	require.NoError(t, lightnotice.LoadFrom(lightNoticeDir))
	t.Cleanup(lightnotice.ResetForTest)
}

func lightLines(msgs []events.Message, userId int) []string {
	var pool []string
	for _, c := range lightnotice.Causes() {
		for _, tr := range lightnotice.Transitions() {
			pool = append(pool, lightnotice.Pool(c, tr, false)...)
			pool = append(pool, lightnotice.Pool(c, tr, true)...)
		}
	}
	var out []string
	for _, m := range msgs {
		if m.UserId != userId {
			continue
		}
		text := strings.ReplaceAll(m.Text, "\n", " ")
		for _, l := range pool {
			if strings.Contains(text, l) {
				out = append(out, l)
				break
			}
		}
	}
	return out
}

func TestLightNoticeOnMove(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	loadLightNotices(t)
	lampRooms(t, 60, 30)

	u := users.GetByUserId(1)
	require.NotNil(t, u)
	u.Character.RoomId = 1
	lightnotice.Check(u, lightnotice.TriggerQuiet)

	captured, capCleanup := captureMessages(t)
	u.Character.RoomId = 2
	LightNoticeOnMove(events.RoomChange{UserId: 1, FromRoomId: 1, ToRoomId: 2})
	u.Character.RoomId = 1
	LightNoticeOnMove(events.RoomChange{UserId: 1, FromRoomId: 2, ToRoomId: 1})
	capCleanup()

	got := lightLines(*captured, 1)
	require.Len(t, got, 1, "darker move speaks once, lighter move is silent")
	require.Contains(t, lightnotice.Pool(lightnotice.CauseMovement, lightnotice.DarkerShapes, false), got[0])
}

func TestLightNoticeOnMoveIgnoresMobs(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	loadLightNotices(t)
	require.Equal(t, events.Continue, LightNoticeOnMove(events.RoomChange{MobInstanceId: 100, FromRoomId: 1, ToRoomId: 2}))
}

func TestLightNoticeDespawnForgets(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	loadLightNotices(t)
	r1, _ := lampRooms(t, 60, 30)

	u := users.GetByUserId(1)
	u.Character.RoomId = 1
	LightNoticeOnSpawn(events.PlayerSpawn{UserId: 1, RoomId: 1})
	LightNoticeOnDespawn(events.PlayerDespawn{UserId: 1, RoomId: 1})

	captured, capCleanup := captureMessages(t)
	r1.Lamp = rooms.LampPtr(30)
	lightnotice.Check(u, lightnotice.TriggerCommand)
	capCleanup()
	require.Empty(t, lightLines(*captured, 1), "after despawn the next check records silently")
}

// TestLightNoticeCombatRound drives the real combat pass: a fighting player
// whose light changed gets the notice from handlePlayerCombat.
func TestLightNoticeCombatRound(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	loadLightNotices(t)
	r1, _ := lampRooms(t, 60, 30)

	u := users.GetByUserId(1)
	u.Character.RoomId = 1
	lightnotice.Check(u, lightnotice.TriggerQuiet)
	u.Character.SetAggro(0, 100, characters.DefaultAttack)
	require.True(t, u.Character.IsInCombat())

	r1.Lamp = rooms.LampPtr(30)
	captured, capCleanup := captureMessages(t)
	handlePlayerCombat(events.NewRound{})
	capCleanup()

	got := lightLines(*captured, 1)
	require.Len(t, got, 1, "one notice per crossing during a combat round")
}

// TestLightNoticeAttentionSweep drives the per-round attention sweep rather
// than calling lightnotice.NoteAttention directly: a blinded player's next
// check must record silently, and the check after that speaks normally.
func TestLightNoticeAttentionSweep(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	loadLightNotices(t)
	r1, _ := lampRooms(t, 60, 30)

	u := users.GetByUserId(1)
	u.Character.RoomId = 1
	u.Character.Perception = perception.NewMachine()
	lightnotice.Check(u, lightnotice.TriggerQuiet)

	require.NoError(t, u.Character.Perception.TransitionTo(perception.Blinded, state.TransitionReason{Trigger: "test"}))
	LightNoticeAttention(events.NewRound{})
	r1.Lamp = rooms.LampPtr(30)
	require.NoError(t, u.Character.Perception.TransitionTo(perception.Sighted, state.TransitionReason{Trigger: "test"}))

	captured, capCleanup := captureMessages(t)
	lightnotice.Check(u, lightnotice.TriggerCommand)
	capCleanup()
	require.Empty(t, lightLines(*captured, 1), "the first check after blindness records silently")

	r1.Lamp = rooms.LampPtr(60)
	captured2, capCleanup2 := captureMessages(t)
	lightnotice.Check(u, lightnotice.TriggerCommand)
	capCleanup2()
	require.Len(t, lightLines(*captured2, 1), 1, "after the silent resync a real change speaks")
}
