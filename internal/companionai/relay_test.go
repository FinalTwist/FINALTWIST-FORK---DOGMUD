package companionai

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRelaySeamsAreNilSafe(t *testing.T) {
	SetRelaySender(nil)
	SetRelayInbound(nil)
	SetRelayPage(nil)
	if SendRelay(1, `Companion.Relay.Request`, []byte(`{}`)) {
		t.Fatal("with no sender installed nothing is sent")
	}
	RelayInbound(1, `Companion.Relay.Response`, []byte(`{}`)) // must not panic
	rec := httptest.NewRecorder()
	if ServeRelayPage(rec, httptest.NewRequest(http.MethodGet, `/companion-relay.html`, nil)) {
		t.Fatal("with no page installed nothing is claimed")
	}
	if RelayOrigin() != `` {
		t.Fatal("with no page installed there is no relay origin")
	}
}

func TestRelaySeamsCallWhatIsInstalled(t *testing.T) {
	var sentTo int
	SetRelaySender(func(userId int, module string, payload []byte) bool { sentTo = userId; return true })
	defer SetRelaySender(nil)
	if !SendRelay(7, `Companion.Relay.Request`, []byte(`{}`)) || sentTo != 7 {
		t.Fatal("the installed sender must be used")
	}
	var got string
	SetRelayInbound(func(userId int, command string, payload []byte) { got = command })
	defer SetRelayInbound(nil)
	RelayInbound(7, `Companion.Relay.Ready`, []byte(`{}`))
	if got != `Companion.Relay.Ready` {
		t.Fatal("the installed inbound handler must be used")
	}
	SetRelayPage(func(w http.ResponseWriter, r *http.Request) bool { return true }, func() string { return `https://keys.example.org` })
	if RelayOrigin() != `https://keys.example.org` {
		t.Fatal("the installed origin must be reported")
	}
	if !ServeRelayPage(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, `/companion-relay.html`, nil)) {
		t.Fatal("the installed page server must be asked")
	}
	// Switching the module off removes the page AND its origin: a stale
	// origin would keep the game page's CSP pointing at a relay nobody serves.
	SetRelayPage(nil)
	if RelayOrigin() != `` {
		t.Fatal("removing the page must remove its origin too")
	}
	if ServeRelayPage(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, `/companion-relay.html`, nil)) {
		t.Fatal("a removed page server must claim nothing")
	}
}
