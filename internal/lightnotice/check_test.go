package lightnotice

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// seedLampWorld builds two sky-less rooms whose light is exactly their lamp, so
// bands are fixed without a clock: 60 faces, 30 shapes, 80 dazzled.
func seedLampWorld(t *testing.T) (*users.UserRecord, *rooms.Room, *rooms.Room) {
	t.Helper()
	r1 := &rooms.Room{RoomId: 1, Zone: "LightZone", SkyLight: rooms.SkyLightPtr(0), Lamp: rooms.LampPtr(60)}
	r2 := &rooms.Room{RoomId: 2, Zone: "LightZone", SkyLight: rooms.SkyLightPtr(0), Lamp: rooms.LampPtr(30)}
	t.Cleanup(rooms.SeedRoomsForTest(
		map[int]*rooms.Room{1: r1, 2: r2},
		map[string]*rooms.ZoneConfig{"LightZone": {Name: "LightZone", RoomId: 1, RoomIds: map[int]struct{}{1: {}, 2: {}}}},
	))
	t.Cleanup(rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{}))
	u := users.NewTestUser(1, "alice", "Aliceia", 1001)
	u.Character.RoomId = 1
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{1: u}))
	if err := LoadFrom(shippedDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ResetForTest)
	return u, r1, r2
}

// captureFor returns a drain function yielding every message text sent to one
// user since the capture began.
func captureFor(t *testing.T, userId int) func() []string {
	t.Helper()
	events.ProcessEvents() // drop anything queued before the capture
	var got []string
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if m, ok := e.(events.Message); ok && m.UserId == userId {
			got = append(got, strings.ReplaceAll(m.Text, "\n", " "))
		}
		return events.Continue
	})
	t.Cleanup(func() { events.UnregisterListener(events.Message{}, id) })
	return func() []string {
		events.ProcessEvents()
		out := got
		got = nil
		return out
	}
}

func containsAny(text string, pool []string) bool {
	for _, l := range pool {
		if strings.Contains(text, l) {
			return true
		}
	}
	return false
}

func TestCheckSpeaksOnceThenFallsSilent(t *testing.T) {
	u, r1, _ := seedLampWorld(t)
	drain := captureFor(t, 1)

	Check(u, TriggerQuiet)
	r1.Lamp = rooms.LampPtr(30)
	Check(u, TriggerCommand)
	got := drain()
	if len(got) != 1 || !containsAny(got[0], Pool(CauseLamp, DarkerShapes, false)) {
		t.Fatalf("want exactly one lamp darker_shapes line, got %q", got)
	}

	Check(u, TriggerCommand)
	if got := drain(); len(got) != 0 {
		t.Fatalf("a second check in the same band must be silent, got %q", got)
	}
}

func TestCheckMoveUsesMovementCause(t *testing.T) {
	u, _, _ := seedLampWorld(t)
	drain := captureFor(t, 1)

	Check(u, TriggerQuiet)
	u.Character.RoomId = 2
	Check(u, TriggerMove)
	got := drain()
	if len(got) != 1 || !containsAny(got[0], Pool(CauseMovement, DarkerShapes, false)) {
		t.Fatalf("want one movement darker_shapes line, got %q", got)
	}

	u.Character.RoomId = 1
	Check(u, TriggerMove)
	if got := drain(); len(got) != 0 {
		t.Fatalf("moving into better light must be silent, got %q", got)
	}
}

func TestBlindnessEndingIsSilent(t *testing.T) {
	u, r1, _ := seedLampWorld(t)
	drain := captureFor(t, 1)
	u.Character.Perception = perception.NewMachine()

	Check(u, TriggerQuiet)
	if err := u.Character.Perception.TransitionTo(perception.Blinded, state.TransitionReason{Trigger: "test"}); err != nil {
		t.Fatal(err)
	}
	NoteAttention(u)
	r1.Lamp = rooms.LampPtr(30)
	if err := u.Character.Perception.TransitionTo(perception.Sighted, state.TransitionReason{Trigger: "test"}); err != nil {
		t.Fatal(err)
	}
	Check(u, TriggerCommand)
	if got := drain(); len(got) != 0 {
		t.Fatalf("the first check after blindness must record silently, got %q", got)
	}

	r1.Lamp = rooms.LampPtr(60)
	Check(u, TriggerCommand)
	if got := drain(); len(got) != 1 {
		t.Fatalf("after the silent resync a real change must speak, got %q", got)
	}
}

func TestForgetMakesTheNextCheckSilent(t *testing.T) {
	u, r1, _ := seedLampWorld(t)
	drain := captureFor(t, 1)

	Check(u, TriggerQuiet)
	Forget(u.UserId)
	r1.Lamp = rooms.LampPtr(30)
	Check(u, TriggerCommand)
	if got := drain(); len(got) != 0 {
		t.Fatalf("after Forget the first check records silently, got %q", got)
	}
}

func TestUnloadedStoreNeverSpeaks(t *testing.T) {
	u, r1, _ := seedLampWorld(t)
	ResetForTest()
	drain := captureFor(t, 1)

	Check(u, TriggerQuiet)
	r1.Lamp = rooms.LampPtr(30)
	Check(u, TriggerCommand)
	if got := drain(); len(got) != 0 {
		t.Fatalf("an unloaded store must be silent, got %q", got)
	}
}

// TestObserveBandAtMatchesBand checks that the production observe builds a
// working bandAt, so attribution's counterfactual reads the observer's real
// sight rather than skipping the check (a nil bandAt).
func TestObserveBandAtMatchesBand(t *testing.T) {
	u, r1, _ := seedLampWorld(t)

	for _, lamp := range []int{30, 60, 80} {
		r1.Lamp = rooms.LampPtr(lamp)
		obs := observe(u.Character, r1)
		if obs.bandAt == nil {
			t.Fatalf("lamp %d: bandAt is nil", lamp)
		}
		if got := obs.bandAt(obs.terms.Level); got != obs.band {
			t.Fatalf("lamp %d: bandAt(terms.Level) = %v, want %v", lamp, got, obs.band)
		}
		if obs.indoor {
			t.Fatalf("lamp %d: indoor = true with an empty biome map", lamp)
		}
	}
}
