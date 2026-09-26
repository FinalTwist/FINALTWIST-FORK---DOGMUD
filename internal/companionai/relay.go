package companionai

import (
	"net/http"
	"sync/atomic"
)

// RelaySendFunc delivers one GMCP message to a player's client. The gmcp
// module installs it; it returns false when the player has no client to
// send to.
type RelaySendFunc func(userId int, module string, payload []byte) bool

// RelayInboundFunc receives a Companion.Relay.* message a player's client
// sent. The aicompanion module installs it. It runs on the connection's own
// goroutine, NOT under the mud lock, so it must touch no game state.
type RelayInboundFunc func(userId int, command string, payload []byte)

// RelayPageFunc serves the key relay page when a request is for the relay
// origin, and reports whether it did.
type RelayPageFunc func(w http.ResponseWriter, r *http.Request) bool

// relayPageSeam keeps the page server and its origin together, so a reader
// never sees one from an old install and the other from a new one.
type relayPageSeam struct {
	serve  RelayPageFunc
	origin func() string
}

// The relay seams are read on connection and web request goroutines, which
// do not hold the mud lock, so unlike the seams in companionai.go they are
// stored atomically.
var (
	relaySend    atomic.Pointer[RelaySendFunc]
	relayInbound atomic.Pointer[RelayInboundFunc]
	relayPage    atomic.Pointer[relayPageSeam]
)

// SetRelaySender installs the GMCP sender. Called by the gmcp module.
func SetRelaySender(f RelaySendFunc) {
	if f == nil {
		relaySend.Store(nil)
		return
	}
	relaySend.Store(&f)
}

// SendRelay sends a relay message to a player's client. Nil-safe: with no
// sender installed nothing is sent and it returns false.
func SendRelay(userId int, module string, payload []byte) bool {
	f := relaySend.Load()
	if f == nil {
		return false
	}
	return (*f)(userId, module, payload)
}

// SetRelayInbound installs the inbound handler. Called by the aicompanion
// module while it is switched on.
func SetRelayInbound(f RelayInboundFunc) {
	if f == nil {
		relayInbound.Store(nil)
		return
	}
	relayInbound.Store(&f)
}

// RelayInbound hands a client's relay message to the module. Nil-safe: with
// nothing installed the message is dropped.
func RelayInbound(userId int, command string, payload []byte) {
	f := relayInbound.Load()
	if f == nil {
		return
	}
	(*f)(userId, command, payload)
}

// SetRelayPage installs the relay page server and the relay origin it
// answers for. Called by the aicompanion module while it is switched on and
// player keys are allowed. SetRelayPage(nil) removes both.
func SetRelayPage(f RelayPageFunc, origin ...func() string) {
	if f == nil {
		relayPage.Store(nil)
		return
	}
	seam := &relayPageSeam{serve: f}
	if len(origin) > 0 {
		seam.origin = origin[0]
	}
	relayPage.Store(seam)
}

// ServeRelayPage lets the module claim a web request for the relay origin.
// Nil-safe: with nothing installed nothing is claimed.
func ServeRelayPage(w http.ResponseWriter, r *http.Request) bool {
	seam := relayPage.Load()
	if seam == nil {
		return false
	}
	return seam.serve(w, r)
}

// RelayOrigin is the relay page's origin ("https://host"), or empty when no
// relay is offered. The web client's CSP uses it for frame-src.
func RelayOrigin() string {
	seam := relayPage.Load()
	if seam == nil || seam.origin == nil {
		return ``
	}
	return seam.origin()
}
