package aicompanion

import (
	"errors"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// withWebDomain pins the game's own host for one test. Test binaries never
// read config.yaml, so the ambient value is a Go default nobody chose.
func withWebDomain(t *testing.T, domain string) {
	t.Helper()
	prev := configs.GetFilePathsConfig().WebDomain
	if err := configs.AddOverlayOverrides(map[string]any{`FilePaths.WebDomain`: domain}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = configs.AddOverlayOverrides(map[string]any{`FilePaths.WebDomain`: string(prev)})
	})
}

// relayModule is a module offering player keys, with owner 5's relay up.
func relayModule(t *testing.T) *AICompanionModule {
	t.Helper()
	withWebDomain(t, `example.org`)
	m := &AICompanionModule{cfg: Config{Enabled: true, PlayerKeys: true, RelayOrigin: `https://keys.example.org`,
		BreakerErrors: 3, BreakerSeconds: 60,
		DailyTokenBudget: 5000, DailyTokensPerCompanion: 1000, StrangerDailyTokens: 1000}}
	m.relays = newRelayTable()
	m.relays.ready(5, `player-model`)
	return m
}

func TestRelayOriginMustBeHTTPSAndForeign(t *testing.T) {
	for _, tc := range []struct {
		origin, web string
		ok          bool
	}{
		{`https://keys.example.org`, `example.org`, true},
		{`https://keys.example.org:8443`, `example.org`, true},
		{`https://keys.example.org/`, `example.org`, true},
		{`http://keys.example.org`, `example.org`, false},
		{`https://example.org`, `example.org`, false},
		{`https://EXAMPLE.org`, `example.org`, false},
		{`https://example.org:8443`, `example.org`, false},
		{`https://example.org`, `example.org:8080`, false},
		{`https://keys.example.org/path`, `example.org`, false},
		{`https://keys.example.org?x=1`, `example.org`, false},
		{`https://keys.example.org#x`, `example.org`, false},
		{`https://user@keys.example.org`, `example.org`, false},
		{`keys.example.org`, `example.org`, false},
		{``, `example.org`, false},
	} {
		if got := validRelayOrigin(tc.origin, tc.web); got != tc.ok {
			t.Errorf("validRelayOrigin(%q, %q) = %v, want %v", tc.origin, tc.web, got, tc.ok)
		}
	}
}

func TestPlayerKeysAreOfferedOnlyOnAForeignOrigin(t *testing.T) {
	m := relayModule(t)
	if !m.playerKeysOffered() {
		t.Fatal("a valid relay origin on another host offers player keys")
	}
	withWebDomain(t, `keys.example.org`)
	if m.playerKeysOffered() {
		t.Fatal("a relay on the game's own host would let the game page read the key")
	}
	withWebDomain(t, `example.org`)
	m.cfg.Enabled = false
	if m.playerKeysOffered() {
		t.Fatal("the module switched off offers nothing")
	}
}

func TestPlayerKeysConfig(t *testing.T) {
	c := buildConfig(nil)
	if c.PlayerKeys || c.RelayOrigin != `` || c.RelayTimeoutSeconds != 30 {
		t.Fatalf("defaults: player keys off, no origin, 30s: %+v", []any{c.PlayerKeys, c.RelayOrigin, c.RelayTimeoutSeconds})
	}
	c = buildConfig(func(k string) any {
		return map[string]any{`PlayerKeys`: true, `RelayOrigin`: ` https://keys.example.org/ `, `RelayTimeoutSeconds`: 1}[k]
	})
	if !c.PlayerKeys || c.RelayOrigin != `https://keys.example.org` || c.RelayTimeoutSeconds != 5 {
		t.Fatalf("read, trimmed and clamped: %+v", []any{c.PlayerKeys, c.RelayOrigin, c.RelayTimeoutSeconds})
	}
}

func TestRouteOrderRelayThenServerThenNone(t *testing.T) {
	withWebDomain(t, `example.org`)
	m := &AICompanionModule{cfg: Config{Enabled: true, PlayerKeys: true, RelayOrigin: `https://keys.example.org`,
		APIKeyEnv: `AICOMPANION_TEST_KEY_NEVER_SET`}}
	m.relays = newRelayTable()
	if r := m.route(5); r.kind != routeNone {
		t.Fatalf("no relay and no server key is tier 1, got %v", r.kind)
	}
	m.relays.ready(5, `gpt-4.1-mini`)
	if r := m.route(5); r.kind != routeRelay || r.model != `gpt-4.1-mini` {
		t.Fatalf("a live relay is used first, got %+v", r)
	}
	if r := m.route(6); r.kind != routeNone {
		t.Fatalf("one owner's relay pays for nobody else, got %v", r.kind)
	}
	m.cfg.APIKey = `sk-server`
	if r := m.route(5); r.kind != routeRelay {
		t.Fatalf("the owner's relay is used before the server's key, got %v", r.kind)
	}
	m.relays.gone(5)
	if r := m.route(5); r.kind != routeServer {
		t.Fatalf("with the relay gone the server key covers, got %v", r.kind)
	}
	m.cfg.PlayerKeys = false
	m.relays.ready(5, `x`)
	if r := m.route(5); r.kind != routeServer {
		t.Fatal("player keys switched off ignores a relay")
	}
	m.cfg.PlayerKeys = true
	m.cfg.RelayOrigin = `http://keys.example.org`
	if r := m.route(5); r.kind != routeServer {
		t.Fatal("an invalid relay origin offers no relay")
	}
}

func TestRelayFailuresTripOnlyThatOwnersBreaker(t *testing.T) {
	m := relayModule(t)
	m.relays.ready(6, `m`)
	now := time.Now()
	for i := 0; i < 3; i++ {
		m.routeResult(route{kind: routeRelay, model: `m`}, 5, errors.New(`provider said no`), now)
	}
	if m.route(5).kind != routeNone {
		t.Fatal("owner 5's own breaker must be open")
	}
	if m.route(6).kind != routeRelay || m.breakerOpen(now) || m.consecutiveErrors != 0 {
		t.Fatal("owner 6 and the global breaker are untouched")
	}
	if _, ok := m.relays.live(5, now.Add(61*time.Second)); !ok {
		t.Fatal("the owner's breaker closes after its cooldown")
	}

	// A refusal at the door never left the server, and a success resets.
	m.relays.ready(7, `m`)
	for i := 0; i < 3; i++ {
		m.routeResult(route{kind: routeRelay}, 7, errNoConsent, now)
	}
	m.routeResult(route{kind: routeRelay}, 7, errors.New(`x`), now)
	m.routeResult(route{kind: routeRelay}, 7, errors.New(`x`), now)
	m.routeResult(route{kind: routeRelay}, 7, nil, now)
	m.routeResult(route{kind: routeRelay}, 7, errors.New(`x`), now)
	if m.route(7).kind != routeRelay {
		t.Fatal("refusals do not count and a success clears the count")
	}

	// The server's key still trips the global breaker.
	for i := 0; i < 3; i++ {
		m.routeResult(route{kind: routeServer}, 6, errors.New(`x`), now)
	}
	if !m.breakerOpen(now) {
		t.Fatal("server-key failures open the global breaker")
	}
}

func TestModelReadyFollowsTheRoute(t *testing.T) {
	m := relayModule(t)
	m.cfg.APIKeyEnv = `AICOMPANION_TEST_KEY_NEVER_SET`
	m.rollDay()
	m.tokensToday = m.cfg.DailyTokenBudget
	m.ownerTokens[5] = m.cfg.DailyTokensPerCompanion
	m.breakerUntil = time.Now().Add(time.Hour)
	if !m.modelReadyFor(5, 0) || !m.modelReady(5) {
		t.Fatal("the owner's own key is not held to the server's budgets or breaker")
	}
	if !m.modelReadyFor(5, 9) {
		t.Fatal("a passer-by talking to a relay companion is paid by the owner's key")
	}
	if m.modelReadyFor(6, 0) || m.modelReady() {
		t.Fatal("no relay and no server key is no call")
	}
	m.cfg.APIKey = `k`
	if m.modelReadyFor(6, 0) {
		t.Fatal("the server's key is still held to the breaker")
	}
	m.breakerUntil = time.Time{}
	if m.modelReadyFor(6, 0) {
		t.Fatal("and to the server's budget")
	}
	m.tokensToday = 0
	m.ownerTokens[6] = m.cfg.DailyTokensPerCompanion
	if m.modelReadyFor(6, 0) || !m.modelReadyFor(6, 9) {
		t.Fatal("the owner's allowance binds the owner's calls, not a passer-by's")
	}
}

func TestApplyRouteUsesThePlayersModel(t *testing.T) {
	m := relayModule(t)
	m.cfg.APIKey = `k`
	call := modelCall{BaseURL: `https://api.openai.com/v1`, APIKey: `k`, Model: `server-model`, Effort: `low`, OwnerUserId: 5}
	m.applyRoute(&call)
	if call.Route.kind != routeRelay || call.Model != `player-model` || call.Effort != `` || call.APIKey != `` || call.BaseURL != `` {
		t.Fatalf("a relay call carries the player's model and nothing of the server's: %+v", call)
	}
	call = modelCall{BaseURL: `https://api.openai.com/v1`, APIKey: `k`, Model: `server-model`, Effort: `low`, OwnerUserId: 6}
	m.applyRoute(&call)
	if call.Route.kind != routeServer || call.Model != `server-model` || call.Effort != `low` || call.APIKey != `k` {
		t.Fatalf("a server call is unchanged: %+v", call)
	}
}

func TestRelayCallsReserveNothingOfTheServers(t *testing.T) {
	m := relayModule(t)
	relay := route{kind: routeRelay, model: `player-model`}
	m.rollDay()
	m.tokensToday = m.cfg.DailyTokenBudget
	m.ownerTokens[5] = m.cfg.DailyTokensPerCompanion

	if !m.reserveRoute(relay, 5, 0, 900) {
		t.Fatal("the owner's own key is not refused for the server's spent budgets")
	}
	if m.tokensToday != m.cfg.DailyTokenBudget || m.outstanding != 0 || m.ownerTokens[5] != m.cfg.DailyTokensPerCompanion {
		t.Fatalf("and holds nothing against them: today=%d outstanding=%d owner=%d", m.tokensToday, m.outstanding, m.ownerTokens[5])
	}
	m.settleRoute(relay, 5, 0, 900, 700)
	if m.tokensToday != m.cfg.DailyTokenBudget || m.outstanding != 0 || m.ownerTokens[5] != m.cfg.DailyTokensPerCompanion {
		t.Fatalf("nor settles anything against them: today=%d outstanding=%d owner=%d", m.tokensToday, m.outstanding, m.ownerTokens[5])
	}
	if m.reserveRoute(route{kind: routeNone}, 5, 0, 1) {
		t.Fatal("tier 1 reserves nothing because it calls nothing")
	}
}

func TestStrangerRelayCallsStopAtTheStrangerCap(t *testing.T) {
	m := relayModule(t)
	relay := route{kind: routeRelay, model: `player-model`}

	if !m.reserveRoute(relay, 5, 2, 900) {
		t.Fatal("a passer-by's question that fits their allowance is admitted")
	}
	if m.reserveRoute(relay, 5, 2, 900) {
		t.Fatal("a second that would overshoot it is refused while the first is held")
	}
	if m.strangerTokens[2] != 900 || m.tokensToday != 0 || m.outstanding != 0 || m.ownerTokens[5] != 0 {
		t.Fatalf("held against the passer-by alone: stranger=%d today=%d outstanding=%d owner=%d",
			m.strangerTokens[2], m.tokensToday, m.outstanding, m.ownerTokens[5])
	}
	m.settleRoute(relay, 5, 2, 900, 100)
	if m.strangerTokens[2] != 100 {
		t.Fatalf("settled to what was used: %d", m.strangerTokens[2])
	}
	if !m.reserveRoute(relay, 5, 3, 900) {
		t.Fatal("another passer-by has their own allowance")
	}
	m.settleRoute(relay, 5, 3, 900, 0)
	if m.strangerTokens[3] != 0 {
		t.Fatalf("a failed call refunds all of it, once: %d", m.strangerTokens[3])
	}
}

func TestStrangerRelayReservationsCannotSlipPastTheCapTogether(t *testing.T) {
	m := relayModule(t)
	relay := route{kind: routeRelay, model: `player-model`}
	admitted := 0
	done := make(chan bool)
	for i := 0; i < 20; i++ {
		go func() {
			util.LockMud()
			ok := m.reserveRoute(relay, 5, 2, 400)
			util.UnlockMud()
			done <- ok
		}()
	}
	for i := 0; i < 20; i++ {
		if <-done {
			admitted++
		}
	}
	if admitted != 2 || m.strangerTokens[2] != 800 {
		t.Fatalf("a 1000-token allowance admits two 400-token holds, got %d (held %d)", admitted, m.strangerTokens[2])
	}
}

// A passer-by's talk with a relay companion is summed up on the owner's
// key and charged only to the passer-by's allowance: the server's spent
// budgets do not stop it, it never goes to the server's endpoint, and its
// failure reaches the owner's breaker, never the global one.
func TestRelaySummaryChargesOnlyThePasserBy(t *testing.T) {
	srv, hits := countingServer(t)
	m, c := strangerTalk(t, srv.URL)
	withWebDomain(t, `example.org`)
	m.cfg.PlayerKeys, m.cfg.RelayOrigin = true, `https://keys.example.org`
	m.relays = newRelayTable()
	m.relays.ready(1, `player-model`)
	m.rollDay()
	m.tokensToday = m.cfg.DailyTokenBudget
	m.ownerTokens[1] = m.cfg.DailyTokensPerCompanion

	util.LockMud()
	m.closeConversation(c, `test`)
	held := m.strangerTokens[2]
	util.UnlockMud()
	if held == 0 {
		t.Fatal("the summary was started on the owner's key and held against the passer-by")
	}
	if len(c.mind.Memories) != 0 {
		t.Fatalf("no plain note while the call is out: %+v", c.mind.Memories)
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		util.LockMud()
		failures := m.relays.owners[1].failures
		left := m.strangerTokens[2]
		util.UnlockMud()
		if failures == 1 && left == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("never settled: relay failures=%d stranger=%d", failures, left)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if hits.Load() != 0 {
		t.Fatalf("a relay call never goes to the server's endpoint: %d requests", hits.Load())
	}
	if m.tokensToday != m.cfg.DailyTokenBudget || m.ownerTokens[1] != m.cfg.DailyTokensPerCompanion || m.outstanding != 0 {
		t.Fatalf("the server's ledgers are untouched: today=%d owner=%d outstanding=%d", m.tokensToday, m.ownerTokens[1], m.outstanding)
	}
	if m.consecutiveErrors != 0 {
		t.Fatal("a relay failure never counts toward the global breaker")
	}
}

// The owner's own reflection and core memory run on the owner's key with
// the server's budgets spent, hold nothing of the server's, and fail to the
// owner's breaker. A core memory whose call fails keeps the bare fact, as
// it would with no model at all.
func TestRelayReflectionAndCoreMemorySpendNothingOfTheServers(t *testing.T) {
	srv, hits := countingServer(t)
	m, c := senderModule(srv.URL, true)
	withWebDomain(t, `example.org`)
	m.cfg.PlayerKeys, m.cfg.RelayOrigin = true, `https://keys.example.org`
	m.cfg.BreakerErrors = 10
	m.relays = newRelayTable()
	m.relays.ready(1, `player-model`)
	m.minds = map[string]*Mind{mindIdentifier(c.mind.OwnerUserId, c.mind.MobId): c.mind}
	m.ctrls = map[int]*controller{c.ownerUserId: c}
	m.rollDay()
	m.tokensToday = m.cfg.DailyTokenBudget
	m.ownerTokens[1] = m.cfg.DailyTokensPerCompanion

	now := time.Now().Unix()
	util.LockMud()
	for i := 0; i < 6; i++ {
		c.mind.addLine(Line{Speaker: `Corvin`, Kind: `said`, Text: `hello`, Unix: now}, 50)
	}
	m.startReflection(c.mind, c.profile, `Corvin`, 0)
	m.recordCore(c, `Corvin`, romanceCourting, true)
	cores := len(c.mind.CoreMemories)
	util.UnlockMud()
	if cores != 0 {
		t.Fatal("the core memory call was started, not skipped for the server's spent budget")
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		util.LockMud()
		failures := m.relays.owners[1].failures
		util.UnlockMud()
		if failures == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("both calls should have failed to the owner's breaker, got %d", failures)
		}
		time.Sleep(10 * time.Millisecond)
	}
	util.LockMud()
	defer util.UnlockMud()
	if hits.Load() != 0 || m.consecutiveErrors != 0 {
		t.Fatalf("nothing reached the server's endpoint (%d) or the global breaker (%d)", hits.Load(), m.consecutiveErrors)
	}
	if m.tokensToday != m.cfg.DailyTokenBudget || m.ownerTokens[1] != m.cfg.DailyTokensPerCompanion || m.outstanding != 0 {
		t.Fatalf("the server's ledgers are untouched: today=%d owner=%d outstanding=%d", m.tokensToday, m.ownerTokens[1], m.outstanding)
	}
	if len(c.mind.CoreMemories) != 1 || c.mind.CoreMemories[0].Text != `Something changed between Corvin and me here.` {
		t.Fatalf("a failed call keeps the bare fact: %+v", c.mind.CoreMemories)
	}
}
