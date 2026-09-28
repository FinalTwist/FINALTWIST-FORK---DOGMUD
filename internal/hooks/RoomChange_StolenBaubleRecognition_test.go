package hooks

import (
	"os"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

// A player's move looks only at that player's baubles; a mob's move lets
// only that mob look; a move to nowhere, or by nobody, does nothing.
func TestStolenBaubleRecognitionRoutesTheMove(t *testing.T) {
	type call struct{ room, user, mob int }
	var calls []call
	orig := recognizeStolenBaubles
	recognizeStolenBaubles = func(roomId int, userId int, mobInstanceId int) {
		calls = append(calls, call{roomId, userId, mobInstanceId})
	}
	t.Cleanup(func() { recognizeStolenBaubles = orig })

	StolenBaubleRecognition(events.RoomChange{UserId: 7, FromRoomId: 1, ToRoomId: 2})
	StolenBaubleRecognition(events.RoomChange{MobInstanceId: 301, FromRoomId: 1, ToRoomId: 3})
	StolenBaubleRecognition(events.RoomChange{UserId: 7, ToRoomId: 0})
	StolenBaubleRecognition(events.RoomChange{ToRoomId: 4})

	want := []call{{2, 7, 0}, {3, 0, 301}}
	if len(calls) != len(want) {
		t.Fatalf("calls %+v, want %+v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("call %d: %+v, want %+v", i, calls[i], want[i])
		}
	}
}

// The listener is registered (the events package cannot be asked, so the
// source is read, as TestLightNoticeListenersAreRegistered does). Without
// the line, owners would never recognise anything while every test of the
// listener itself kept passing.
func TestStolenBaubleRecognitionIsRegistered(t *testing.T) {
	src, err := os.ReadFile("hooks.go")
	if err != nil {
		t.Fatalf("reading hooks.go: %v", err)
	}
	if !strings.Contains(string(src), "events.RegisterListener(events.RoomChange{}, StolenBaubleRecognition)") {
		t.Error("hooks.go no longer registers StolenBaubleRecognition on RoomChange")
	}
}
