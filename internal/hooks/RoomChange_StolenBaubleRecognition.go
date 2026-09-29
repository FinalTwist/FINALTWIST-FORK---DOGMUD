package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
)

// recognizeStolenBaubles is actions.RecognizeStolenBaubles. A variable so
// tests can see how a move is routed.
var recognizeStolenBaubles = actions.RecognizeStolenBaubles

// StolenBaubleRecognition gives an NPC robbed of a bauble its chance to
// recognise it (docs/baubles Phase 6c) whenever someone moves into a room:
// a player walking in on the owner, or the owner walking in on the player.
// The work is actions.RecognizeStolenBaubles; this only routes the move.
func StolenBaubleRecognition(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok || evt.ToRoomId == 0 {
		return events.Continue
	}
	if evt.UserId == 0 && evt.MobInstanceId == 0 {
		return events.Continue
	}
	userId := evt.UserId
	mobInstanceId := 0
	if userId == 0 {
		mobInstanceId = evt.MobInstanceId
	}
	recognizeStolenBaubles(evt.ToRoomId, userId, mobInstanceId)
	return events.Continue
}
