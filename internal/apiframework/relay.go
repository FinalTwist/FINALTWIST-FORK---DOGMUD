package apiframework

import (
	"context"
	"sync/atomic"
)

// A player's own key, held in their browser (the AI companion's key relay,
// opened from the web client's "Companion key" button). The companion module
// owns the relay: the page, the transport through the player's client and
// each player's own breaker. It registers itself here so another feature
// can send a request through a player's relay, but only for a purpose that
// player has allowed.
//
// Nothing here is the server's: a relayed call is never reserved against
// the server's budget and never feeds the server key's breaker.

// PurposeFinds is naming what the player finds while searching (baubles).
// The player allows it with "Also name things I find while searching" on
// the key page.
const PurposeFinds = `finds`

// Relay sends requests through a player's own key.
type Relay interface {
	// Model returns the model the player's live relay uses when it may be
	// used now for purpose: the relay is up, the player allowed purpose,
	// and the player's own breaker is closed.
	Model(userId int, purpose string) (model string, ok bool)
	// Send posts a chat completions body (built with Chat.Body; the relay
	// sets the model itself) through the player's browser and returns the
	// provider's status and raw reply. carries says what the body carries,
	// for the relay's consent door. sent is the request possibly having
	// reached the provider.
	Send(ctx context.Context, userId int, body []byte, carries Carries) (status int, raw []byte, sent bool, err error)
	// Result feeds the player's own breaker with a call's outcome.
	Result(userId int, err error)
}

var relay atomic.Pointer[Relay]

// SetRelay installs the relay (the companion module, at load). nil removes
// it.
func SetRelay(r Relay) {
	if r == nil {
		relay.Store(nil)
		return
	}
	relay.Store(&r)
}

// PlayerRelay returns the installed relay, or nil.
func PlayerRelay() Relay {
	if r := relay.Load(); r != nil {
		return *r
	}
	return nil
}
