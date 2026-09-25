package aicompanion

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Task 14B: what a companion's calls cost, and who is charged for them.

// Player text reaches her mind whole, and every stored line goes out again
// in each prompt: each writer keeps at most maxStoredRunes of it, cut on a
// rune boundary.
func TestStoredTextIsCapped(t *testing.T) {
	long := strings.Repeat(`ж`, 2000) // two bytes a rune: a byte cut would split one
	m, c := senderModule(`https://api.example.invalid`, true)
	mind := c.mind
	mind.addLine(Line{Kind: `said`, Speaker: `Bram`, Text: long}, 50)
	mind.addMemory(Memory{Kind: `event`, Text: long, Importance: 5}, 50)
	mind.addFact(Fact{Text: long}, 50)
	mind.addPromise(`Bram`, long)
	mind.addHearsay(`Bram`, long, 1)
	mind.addOwnPhrase(long, 10)
	mind.addCore(CoreMemory{Text: long})
	m.noteConversation(c, 7, `Bram`, 2, Line{Speaker: `Bram`, Kind: `said`, Text: long})

	got := map[string]string{
		`line`:     mind.RecentLines[len(mind.RecentLines)-1].Text,
		`memory`:   mind.Memories[len(mind.Memories)-1].Text,
		`fact`:     mind.Facts[len(mind.Facts)-1].Text,
		`promise`:  mind.Promises[len(mind.Promises)-1].Text,
		`hearsay`:  mind.Hearsay[len(mind.Hearsay)-1].Text,
		`phrase`:   mind.OwnPhrases[len(mind.OwnPhrases)-1],
		`core`:     mind.CoreMemories[len(mind.CoreMemories)-1].Text,
		`converse`: c.convo.Lines[len(c.convo.Lines)-1].Text,
	}
	for what, text := range got {
		if n := len([]rune(text)); n != maxStoredRunes {
			t.Errorf("%s keeps %d runes, want %d", what, n, maxStoredRunes)
		}
		if strings.ContainsRune(text, '�') {
			t.Errorf("%s was cut inside a character", what)
		}
	}
	if capRunes(`  short  `) != `short` {
		t.Fatal("short text is only trimmed")
	}
}

// A reply asking the game dozens of questions at once is answered for the
// first few only.
func TestToolCallsPerReplyAreCapped(t *testing.T) {
	var calls []string
	for i := 0; i < 12; i++ {
		calls = append(calls, fmt.Sprintf(`{"id":"c%d","type":"function","function":{"name":"recall","arguments":"{}"}}`, i))
	}
	raw := `{"choices":[{"finish_reason":"tool_calls","message":{"tool_calls":[` + strings.Join(calls, `,`) + `]}}],"usage":{"total_tokens":10}}`
	res := decodeChatResponse(modelResult{}, 200, []byte(raw))
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if len(res.ToolCalls) != maxToolCallsPerReply || maxToolCallsPerReply != 4 {
		t.Fatalf("at most four questions a reply, got %d", len(res.ToolCalls))
	}
	if res.ToolCalls[0].Id != `c0` || res.ToolCalls[3].Id != `c3` {
		t.Fatalf("the first ones are kept: %+v", res.ToolCalls)
	}
}

// An error reply that came through a player's browser keeps none of its
// body: not in the error, so not in the log or the trace either.
func TestRelayErrorReplyCarriesNoBody(t *testing.T) {
	m, f := relayCallModule(t, true)
	c := relayCall(m)
	c.Retry = false
	done := make(chan modelResult, 1)
	go func() { done <- m.callModel(c) }()
	req := f.next(t)
	m.relayCalls.deliver(5, relayResponse{Id: req.Id, Status: 401, Body: `{"error":{"message":"Incorrect key for account acct_4242 of jane@example.com"}}`})
	res := <-done
	if res.Err == nil || res.Status != 401 {
		t.Fatalf("an error status fails the call: %+v", res)
	}
	if msg := res.Err.Error(); strings.Contains(msg, `jane`) || strings.Contains(msg, `acct_4242`) || strings.Contains(msg, `Incorrect`) {
		t.Fatalf("the relayed body leaked into the error: %q", msg)
	}
	if !strings.Contains(res.Err.Error(), `401`) {
		t.Fatalf("the status is kept: %q", res.Err.Error())
	}
	// The server's own provider is the operator's business: its error text
	// is still kept for them.
	direct := decodeChatResponse(modelResult{}, 500, []byte(`upstream busy`))
	if direct.Err == nil || !strings.Contains(direct.Err.Error(), `upstream busy`) {
		t.Fatalf("control: the server key's error keeps its text: %v", direct.Err)
	}
}

// A talk among passers-by is summed up at the cost of the one who said the
// most in it, not whoever spoke last.
func TestConversationPayerIsWhoSaidTheMost(t *testing.T) {
	m, c := senderModule(`https://api.example.invalid`, true)
	for i := 0; i < 5; i++ {
		m.noteConversation(c, 7, `Bram`, 2, Line{Speaker: `Bram`, Kind: `said`, Text: fmt.Sprintf(`line %d`, i)})
	}
	m.noteConversation(c, 7, `Ilse`, 3, Line{Speaker: `Ilse`, Kind: `said`, Text: `bye`})
	if p := c.convo.payer(); p != 2 {
		t.Fatalf("the one who said the most pays, got user %d", p)
	}
	m.noteConversation(c, 7, `Ilse`, 3, Line{Speaker: `Ilse`, Kind: `said`, Text: `a`})
	for i := 0; i < 4; i++ {
		m.noteConversation(c, 7, `Ilse`, 3, Line{Speaker: `Ilse`, Kind: `said`, Text: `b`})
	}
	if p := c.convo.payer(); p != 3 {
		t.Fatalf("and it moves when somebody else says more, got user %d", p)
	}
	m.noteConversation(c, 7, `Corvin`, 1, Line{Speaker: `Corvin`, Kind: `said`, Text: `hello`})
	if p := c.convo.payer(); p != 0 {
		t.Fatalf("a talk the owner joined is the owner's, got user %d", p)
	}
}

// companion-unstick abandons a call that may have been paid for and frees
// her to start another at once: it waits a minute between uses.
func TestUnstickHasACooldown(t *testing.T) {
	m, c := senderModule(`https://api.example.invalid`, true)
	m.cfg.Enabled = true
	m.ctrls = map[int]*controller{1: c}
	owner := users.NewTestUser(1, `corvin`, `Corvin`, 0)
	if _, err := m.cmdUnstick(``, owner, nil, 0); err != nil {
		t.Fatal(err)
	}
	first := c.seq
	if first == 0 {
		t.Fatal("fixture: the first reset goes through")
	}
	c.pending = []stimulus{{Kind: `heard`}}
	if _, err := m.cmdUnstick(``, owner, nil, 0); err != nil {
		t.Fatal(err)
	}
	if c.seq != first || len(c.pending) != 1 {
		t.Fatalf("a second reset at once is refused: seq %d -> %d, pending %d", first, c.seq, len(c.pending))
	}
	rs := int(configs.GetTimingConfig().RoundSeconds)
	if n, want := owner.Character.GetCooldown(unstickCooldownTag), (60+rs-1)/rs; n != want {
		t.Fatalf("the wait is a real minute, %d rounds, got %d", want, n)
	}
}

// Character cooldowns count in rounds and read no seconds: cooldownFor
// turns real seconds into the rounds that last as long.
func TestCooldownForCountsRealSeconds(t *testing.T) {
	rs := int(configs.GetTimingConfig().RoundSeconds)
	if got := gametime.PeriodLength(cooldownFor(60)); int(got) != (60+rs-1)/rs {
		t.Fatalf("sixty seconds is %d rounds of %d seconds, got %d", (60+rs-1)/rs, rs, got)
	}
	if got := gametime.PeriodLength(cooldownFor(0)); got != 1 {
		t.Fatalf("at least one round, got %d", got)
	}
}

// usageServer answers every chat completion with the given body, after
// waiting on hold when it is not nil.
func usageServer(t *testing.T, body string, hold chan struct{}) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hold != nil {
			select {
			case <-hold:
			case <-r.Context().Done():
				return
			}
		}
		fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func usageCall(m *AICompanionModule, baseURL string) modelCall {
	c := modelCall{BaseURL: baseURL, APIKey: `k`, Model: `m`, Timeout: 2 * time.Second, MaxTokens: 100,
		Messages: []chatMessage{{Role: `user`, Content: strings.Repeat(`word `, 400)}}, SchemaName: `s`,
		Schema: decisionSchema(), OwnerUserId: 1}
	m.applyRoute(&c)
	return c
}

// A request that left and came back with no usage may still have been
// billed: it is counted at its prompt estimate, not as nothing. One that
// never left counts nothing.
func TestSentCallWithNoUsageCountsItsPrompt(t *testing.T) {
	m, _ := senderModule(`https://api.example.invalid`, true)
	probe := usageCall(m, `x`)
	prompt := estimateTokens(probe.Messages) + requestOverhead(probe)

	// An answer with no usage at all.
	srv := usageServer(t, `{"choices":[{"finish_reason":"stop","message":{"content":"{}"}}]}`, nil)
	res := m.callModelOnce(usageCall(m, srv.URL))
	if res.Err != nil || res.Tokens != prompt || !res.Estimated {
		t.Fatalf("an answer with no usage counts its prompt %d: %+v", prompt, res)
	}
	// Real usage is taken as given.
	srv = usageServer(t, `{"choices":[{"finish_reason":"stop","message":{"content":"{}"}}],"usage":{"total_tokens":7}}`, nil)
	if res = m.callModelOnce(usageCall(m, srv.URL)); res.Tokens != 7 || res.Estimated {
		t.Fatalf("reported usage stands: %+v", res)
	}
	// A timeout: sent, never answered.
	hold := make(chan struct{})
	defer close(hold)
	srv = usageServer(t, `{}`, hold)
	c := usageCall(m, srv.URL)
	c.Timeout = 200 * time.Millisecond
	if res = m.callModelOnce(c); res.Err == nil || res.Tokens != prompt {
		t.Fatalf("a timed-out request counts its prompt: %+v", res)
	}
	// Given up on after it was sent (companion-unstick, logout).
	c = usageCall(m, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	c.Ctx = ctx
	go func() { time.Sleep(200 * time.Millisecond); cancel() }()
	if res = m.callModelOnce(c); !res.Canceled || res.Tokens != prompt {
		t.Fatalf("a call cancelled after it was sent counts its prompt: %+v", res)
	}
	// Given up on before it left.
	if res = m.callModelOnce(c); !res.Canceled || res.Tokens != 0 || res.Sent {
		t.Fatalf("a call cancelled before it left counts nothing: %+v", res)
	}
	// Refused at the door.
	m.consent.agreed = map[int]bool{}
	if res = m.callModelOnce(usageCall(m, srv.URL)); !errors.Is(res.Err, errNoConsent) || res.Tokens != 0 {
		t.Fatalf("a request the door refused counts nothing: %+v", res)
	}
	m.consent.agreed = map[int]bool{1: true}
	// Nobody to connect to.
	dead := httptest.NewServer(http.NotFoundHandler())
	deadURL := dead.URL
	dead.Close()
	if res = m.callModelOnce(usageCall(m, deadURL)); res.Err == nil || res.Tokens != 0 {
		t.Fatalf("a connection never made counts nothing: %+v", res)
	}
}

// A cancelled call is neither a failure nor a success to the breaker: it
// reaches it not at all, so it cannot reset the count of real failures.
func TestCancelledCallNeverReachesTheBreaker(t *testing.T) {
	m, _ := senderModule(`https://api.example.invalid`, true)
	m.cfg.BreakerErrors = 3
	m.consecutiveErrors = 2
	m.ctrls = map[int]*controller{}
	m.applyResult(1, 1, 0, 0, 100, nil, nil, tierMain, `m`, route{kind: routeServer},
		modelResult{Err: context.Canceled, Canceled: true})
	if m.consecutiveErrors != 2 {
		t.Fatalf("a cancel leaves the failure count alone, got %d", m.consecutiveErrors)
	}
	m.applyResult(1, 1, 0, 0, 100, nil, nil, tierMain, `m`, route{kind: routeServer},
		modelResult{Err: errors.New(`status 500`)})
	if !m.breakerOpen(time.Now()) {
		t.Fatal("so the next real failure still opens the breaker")
	}
}

// Usage that came back through a player's browser is the player's to
// write: it is held between nothing and the most the request could cost,
// and a passer-by is never charged past their reservation.
func TestRelayUsageIsNeverTrusted(t *testing.T) {
	m, f := relayCallModule(t, true)
	for _, tc := range []struct {
		usage string
		want  func(prompt, max int) int
	}{
		{`"usage":{"total_tokens":999999999}`, func(p, x int) int { return p + x }},
		{`"usage":{"total_tokens":-5000}`, func(p, x int) int { return p }}, // no usage: the estimate
		{`"usage":{"total_tokens":12}`, func(p, x int) int { return 12 }},
	} {
		c := relayCall(m)
		c.Retry = false
		prompt := estimateTokens(c.Messages) + requestOverhead(c)
		done := make(chan modelResult, 1)
		go func() { done <- m.callModel(c) }()
		req := f.next(t)
		m.relayCalls.deliver(5, relayResponse{Id: req.Id, Status: 200,
			Body: `{"choices":[{"finish_reason":"stop","message":{"content":"{}"}}],` + tc.usage + `}`})
		res := <-done
		if want := tc.want(prompt, c.MaxTokens); res.Tokens != want {
			t.Fatalf("%s: counted %d, want %d", tc.usage, res.Tokens, want)
		}
	}

	relay := route{kind: routeRelay, model: `player-model`}
	if !m.reserveRoute(relay, 5, 2, 400) {
		t.Fatal("fixture: the passer-by's question fits")
	}
	m.settleRoute(relay, 5, 2, 400, 5000)
	if m.strangerTokens[2] != 400 {
		t.Fatalf("a passer-by pays at most what was held, got %d", m.strangerTokens[2])
	}
	if !m.reserveRoute(relay, 5, 3, 400) {
		t.Fatal("fixture: a second passer-by's question fits")
	}
	m.settleRoute(relay, 5, 3, 400, -900)
	if m.strangerTokens[3] != 0 {
		t.Fatalf("and never less than nothing, got %d", m.strangerTokens[3])
	}
}

// panicTransport stands for anything that blows up while a call is out.
type panicTransport struct{}

func (panicTransport) RoundTrip(*http.Request) (*http.Response, error) { panic(`transport blew up`) }

// A reflection, a conversation summary or a core memory whose goroutine
// panics before its reply is applied still gives its reservation back.
func TestPanickedBackgroundCallsSettle(t *testing.T) {
	prev := httpClient
	httpClient = &http.Client{Transport: panicTransport{}}
	t.Cleanup(func() { httpClient = prev })

	for name, start := range map[string]func(m *AICompanionModule, c *controller){
		`reflection`: func(m *AICompanionModule, c *controller) {
			m.startReflection(c.mind, c.profile, `Corvin`, 0)
		},
		`summary`: func(m *AICompanionModule, c *controller) {
			now := time.Now().Unix()
			c.convo = &conversation{RoomId: 7, Partner: `Corvin`, StartUnix: now, LastUnix: now, Exchanges: 5, OwnerSpoke: true,
				Lines: []Line{{Speaker: `Corvin`, Kind: `said`, Text: `a`}, {Speaker: `Corvin`, Kind: `said`, Text: `b`}}}
			m.closeConversation(c, `test`)
		},
		`core`: func(m *AICompanionModule, c *controller) {
			m.recordCore(c, `Corvin`, romanceCourting, true)
		},
	} {
		m, c := senderModule(`https://api.example.invalid`, true)
		m.minds = map[string]*Mind{mindIdentifier(c.mind.OwnerUserId, c.mind.MobId): c.mind}
		now := time.Now().Unix()
		util.LockMud()
		for i := 0; i < 6; i++ {
			c.mind.addLine(Line{Speaker: `Corvin`, Kind: `said`, Text: fmt.Sprintf(`line %d`, i), Unix: now}, 50)
		}
		start(m, c)
		held := m.outstanding
		util.UnlockMud()
		if held == 0 {
			t.Fatalf("%s: fixture: the call held a reservation", name)
		}
		deadline := time.Now().Add(3 * time.Second)
		for {
			util.LockMud()
			left, owner := m.outstanding, m.ownerTokens[1]
			util.UnlockMud()
			if left == 0 && owner == 0 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s: a panicked call never gave back its reservation: outstanding=%d owner=%d", name, left, owner)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}
