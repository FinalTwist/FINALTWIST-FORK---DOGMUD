package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/lightnotice"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Light-notice event seams into internal/lightnotice. The command and
// combat-round seams are direct calls in usercommands.TryCommand and
// handlePlayerCombat; these four ride events.

// LightNoticeOnMove checks a player after they arrive in a room. RoomChange is
// queued, so this runs after the move command's own output; the command's
// TryCommand check ran BEFORE the move, in the old room, so the two never
// announce the same crossing.
func LightNoticeOnMove(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok || evt.UserId == 0 || evt.MobInstanceId != 0 {
		return events.Continue
	}
	lightnotice.Check(users.GetByUserId(evt.UserId), lightnotice.TriggerMove)
	return events.Continue
}

// LightNoticeOnSpawn records a logging-in player's band silently.
func LightNoticeOnSpawn(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.PlayerSpawn); ok {
		lightnotice.Check(users.GetByUserId(evt.UserId), lightnotice.TriggerQuiet)
	}
	return events.Continue
}

// LightNoticeOnDespawn drops a leaving player's record.
func LightNoticeOnDespawn(e events.Event) events.ListenerReturn {
	if evt, ok := e.(events.PlayerDespawn); ok {
		lightnotice.Forget(evt.UserId)
	}
	return events.Continue
}

// LightNoticeAttention marks every sleeping or blinded player so waking or
// seeing again records silently. Two flag reads per player, no light maths.
func LightNoticeAttention(e events.Event) events.ListenerReturn {
	for _, userId := range users.GetOnlineUserIds() {
		lightnotice.NoteAttention(users.GetByUserId(userId))
	}
	return events.Continue
}
