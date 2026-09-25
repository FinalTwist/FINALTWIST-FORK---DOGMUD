package gmcp

import (
	"os"
	"path/filepath"
	"strings"
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

// A relay reply carries the model's answer, and Ready carries the owner's
// model choice: neither belongs in the debug log, which writes every other
// GMCP payload verbatim.
func TestHandleIAC_CompanionRelayPayloadIsRedactedFromTheDebugLog(t *testing.T) {
	const connId = 90005
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{
		42: users.NewTestUser(42, `relayquiet`, `Relayquiet`, connId),
	}))
	companionai.SetRelayInbound(func(int, string, []byte) {})
	t.Cleanup(func() { companionai.SetRelayInbound(nil) })

	// SetupLogger with no path writes to os.Stderr, captured here to a file.
	// (A log path would hand the file to lumberjack, which never closes it,
	// and TempDir's cleanup then fails on Windows.)
	logPath := filepath.Join(t.TempDir(), `gmcp.log`)
	capture, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	realStderr := os.Stderr
	os.Stderr = capture
	mudlog.SetupLogger(nil, ``, ``, false) // "" is the debug level
	t.Cleanup(func() {
		os.Stderr = realStderr
		mudlog.SetupLogger(nil, ``, ``, false)
	})

	const secret = `she says the passphrase is correct horse`
	gmcpModule.HandleIAC(connId, gmcpFrame(`Companion.Relay.Response {"id":"abc","status":200,"body":"`+secret+`"}`))
	gmcpModule.HandleIAC(connId, gmcpFrame(`Core.Hello {"client":"plain","version":"1"}`))

	os.Stderr = realStderr
	mudlog.SetupLogger(nil, ``, ``, false)
	if err := capture.Close(); err != nil {
		t.Fatal(err)
	}
	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(logged), secret) {
		t.Fatalf("a Companion.Relay payload must not reach the debug log:\n%s", logged)
	}
	if !strings.Contains(string(logged), `Companion.Relay.Response`) || !strings.Contains(string(logged), `redacted`) {
		t.Fatalf("the relay command is still logged, with its payload marked redacted:\n%s", logged)
	}
	if !strings.Contains(string(logged), `plain`) || !strings.Contains(string(logged), `version`) {
		t.Fatalf("other GMCP payloads are logged as before:\n%s", logged)
	}
}

// A zombie connection (the socket dropped, the character lingering) can
// carry nothing to a browser: the module must see an unsent request, not a
// timeout.
func TestRelaySender_RefusesAZombieConnection(t *testing.T) {
	const conn = 90006
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{
		61: users.NewTestUser(61, `zombieplayer`, `Zombieplayer`, conn),
	}))
	web := GMCPSettings{GMCPAccepted: true, HelloReceived: true}
	web.Client.Name = `WebClient`
	gmcpModule.cache.Add(conn, web)
	t.Cleanup(func() { gmcpModule.cache.Remove(conn) })

	if !companionai.SendRelay(61, `Companion.Relay.Request`, []byte(`{}`)) {
		t.Fatal("a live negotiated client is sent the request")
	}
	users.SetZombieUser(61)
	if !users.IsZombieConnection(conn) {
		t.Fatal("fixture: the connection must be a zombie")
	}
	if companionai.SendRelay(61, `Companion.Relay.Request`, []byte(`{}`)) {
		t.Fatal("a zombie connection cannot receive a relay request")
	}
}
