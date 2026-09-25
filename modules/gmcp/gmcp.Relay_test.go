package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/companionai"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// gmcpFrame wraps a GMCP message the way a client sends it: IAC SB GMCP
// <command> <payload> IAC SE.
func gmcpFrame(body string) []byte {
	return append(append([]byte{255, 250, 201}, body...), 255, 240)
}

type relayCall struct {
	userId  int
	command string
	payload []byte
}

func captureRelayInbound(t *testing.T) *[]relayCall {
	t.Helper()
	mudlog.SetupLogger(nil, "", "", false) // HandleIAC logs every frame
	calls := &[]relayCall{}
	companionai.SetRelayInbound(func(userId int, command string, payload []byte) {
		*calls = append(*calls, relayCall{userId, command, payload})
	})
	t.Cleanup(func() { companionai.SetRelayInbound(nil) })
	return calls
}

// The three client relay messages reach the module with the id of the user
// on that connection, and with a payload the module owns: HandleIAC's
// buffer is reused after it returns.
func TestHandleIAC_CompanionRelayReachesTheModuleAsThatUser(t *testing.T) {
	const connId = 90001
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{
		41: users.NewTestUser(41, `relayowner`, `Relayowner`, connId),
	}))
	calls := captureRelayInbound(t)

	for _, cmd := range []string{`Companion.Relay.Response`, `Companion.Relay.Ready`, `Companion.Relay.Gone`} {
		frame := gmcpFrame(cmd + ` {"id":"abc"}`)
		if !gmcpModule.HandleIAC(connId, frame) {
			t.Fatalf("%s: HandleIAC must claim a GMCP frame", cmd)
		}
		for i := range frame {
			frame[i] = 'X' // the read buffer is reused
		}
	}

	if len(*calls) != 3 {
		t.Fatalf("want 3 relay messages forwarded, got %d", len(*calls))
	}
	for i, want := range []string{`Companion.Relay.Response`, `Companion.Relay.Ready`, `Companion.Relay.Gone`} {
		got := (*calls)[i]
		if got.userId != 41 || got.command != want {
			t.Fatalf("call %d: got user %d command %q, want user 41 command %q", i, got.userId, got.command, want)
		}
		if string(got.payload) != `{"id":"abc"}` {
			t.Fatalf("call %d: payload %q was not copied out of the read buffer", i, got.payload)
		}
	}
}

// A connection with no logged-in user has nobody to relay for.
func TestHandleIAC_CompanionRelayFromNoUserIsDropped(t *testing.T) {
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{}))
	calls := captureRelayInbound(t)

	gmcpModule.HandleIAC(90002, gmcpFrame(`Companion.Relay.Response {"id":"abc"}`))

	if len(*calls) != 0 {
		t.Fatalf("a connection with no user must forward nothing, got %d", len(*calls))
	}
}

// The sender says whether the message can arrive, so the module does not
// wait out a deadline for a reply that cannot come.
func TestRelaySender_OnlySendsToANegotiatedClient(t *testing.T) {
	const webConn, rawConn = 90003, 90004
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{
		51: users.NewTestUser(51, `webplayer`, `Webplayer`, webConn),
		52: users.NewTestUser(52, `rawplayer`, `Rawplayer`, rawConn),
	}))
	web := GMCPSettings{GMCPAccepted: true, HelloReceived: true}
	web.Client.Name = `WebClient`
	gmcpModule.cache.Add(webConn, web)
	gmcpModule.cache.Add(rawConn, GMCPSettings{})
	t.Cleanup(func() {
		gmcpModule.cache.Remove(webConn)
		gmcpModule.cache.Remove(rawConn)
	})

	if !companionai.SendRelay(51, `Companion.Relay.Request`, []byte(`{}`)) {
		t.Fatal("a negotiated web client must be sent the request")
	}
	if companionai.SendRelay(52, `Companion.Relay.Request`, []byte(`{}`)) {
		t.Fatal("a client that has not negotiated GMCP cannot receive it")
	}
	if companionai.SendRelay(53, `Companion.Relay.Request`, []byte(`{}`)) {
		t.Fatal("a user who is not connected cannot receive it")
	}
}
