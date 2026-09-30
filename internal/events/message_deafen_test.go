package events

import "testing"

func TestMessageHiddenFromDeafened(t *testing.T) {
	cases := []struct {
		communication, deafened, want bool
	}{
		{true, true, true},
		{true, false, false},
		{false, true, false},
		{false, false, false},
	}
	for _, c := range cases {
		m := Message{IsCommunication: c.communication}
		if got := m.HiddenFromDeafened(c.deafened); got != c.want {
			t.Errorf("IsCommunication=%v deafened=%v: got %v, want %v", c.communication, c.deafened, got, c.want)
		}
	}
}

func TestDrainQueuedMessageHelpersKeepTheWholeMessage(t *testing.T) {
	DrainQueuedMessageEventsForTest(99001)
	DrainQueuedRoomMessagesForTest(99002)
	AddToQueue(Message{UserId: 99001, Text: "to a user", IsCommunication: true})
	AddToQueue(Message{RoomId: 99002, Text: "to a room", IsCommunication: true})

	got := DrainQueuedMessageEventsForTest(99001)
	if len(got) != 1 || got[0].Text != "to a user" || !got[0].IsCommunication {
		t.Fatalf("user drain = %+v, want the one flagged message", got)
	}
	room := DrainQueuedRoomMessagesForTest(99002)
	if len(room) != 1 || room[0].Text != "to a room" || !room[0].IsCommunication {
		t.Fatalf("room drain = %+v, want the one flagged message", room)
	}
	if again := DrainQueuedMessageEventsForTest(99001); len(again) != 0 {
		t.Fatalf("drain did not remove: %+v", again)
	}
}
