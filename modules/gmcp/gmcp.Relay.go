package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/companionai"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Companion.Relay.* carries a companion's model request to the owner's own
// browser and the provider's reply back, for players who run their
// companion on their own key. The key never passes through here: requests
// carry only a request body, replies only a status and a body.
//
// Relay messages touch no game state, so unlike Char.* ops they are handed
// over on the connection goroutine; the module's pending-request table has
// its own lock.

func installRelaySender() {
	companionai.SetRelaySender(relaySend)
}

// relaySend queues one relay message for a player's client. It reports
// false when the player has no connection, or one that has not finished
// GMCP negotiation, since dispatchGMCP would drop the message silently and
// the module would wait out its whole deadline for a reply that cannot come.
func relaySend(userId int, module string, payload []byte) bool {
	connId := users.GetConnectionId(userId)
	if connId == 0 || !isGMCPEnabled(connId) {
		return false
	}
	events.AddToQueue(GMCPOut{UserId: userId, Module: module, Payload: payload})
	return true
}

// relayInbound forwards a relay message from a connection to the module.
// The payload aliases the IAC read buffer, which is reused once HandleIAC
// returns, so the module gets a copy.
func relayInbound(connectionId connections.ConnectionId, command string, payload []byte) {
	u := users.GetByConnectionId(connectionId)
	if u == nil {
		return
	}
	companionai.RelayInbound(u.UserId, command, append([]byte(nil), payload...))
}
