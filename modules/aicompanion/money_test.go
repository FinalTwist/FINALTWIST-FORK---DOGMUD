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

	"gopkg.in/yaml.v2"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
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

// On the owner's own key passers-by prompt nothing until the owner says
// so; on the server's key they may, until the owner says not. The owner's
// word, either way, holds on both keys, and a record saved before the
// choice had a default still means what it meant.
func TestStrangersDefaultOffOnlyOnTheOwnersKey(t *testing.T) {
	m := relayModule(t)
	m.bonds = bondState{Users: map[int]*bondRecord{5: {Profile: `mara`}}}
	relay, server := route{kind: routeRelay}, route{kind: routeServer}
	if !m.strangersOffOn(5, relay) || m.strangersOffOn(5, server) {
		t.Fatal("unset: off on the owner's key, on for the server's")
	}
	if !m.strangersOff(5) || m.strangerMayPrompt(5, 2) {
		t.Fatal("with the owner's relay live, a passer-by prompts nothing by default")
	}
	if !m.strangerMayPrompt(5, 0) {
		t.Fatal("the owner's own calls are untouched")
	}
	m.bonds.Users[5].StrangersOn = true
	if m.strangersOffOn(5, relay) || m.strangersOffOn(5, server) || !m.strangerMayPrompt(5, 2) {
		t.Fatal("strangers on: on for both keys")
	}
	m.bonds.Users[5].StrangersOn, m.bonds.Users[5].StrangersOff = false, true
	if !m.strangersOffOn(5, relay) || !m.strangersOffOn(5, server) {
		t.Fatal("strangers off: off for both keys")
	}
	if m.strangersOffOn(6, server) || !m.strangersOffOn(6, relay) {
		t.Fatal("no record at all is the default too")
	}

	var old bondState
	if err := yaml.Unmarshal([]byte("users:\n  5:\n    profile: mara\n    strangers_off: true\n"), &old); err != nil {
		t.Fatal(err)
	}
	m.bonds = old
	if !m.strangersOffOn(5, server) {
		t.Fatal("an old record that said off is still off, on the server's key too")
	}
	old.Users[5].StrangersOff, old.Users[5].StrangersOn = false, true
	b, err := yaml.Marshal(&old)
	if err != nil {
		t.Fatal(err)
	}
	var back bondState
	if err := yaml.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !back.Users[5].StrangersOn || back.Users[5].StrangersOff {
		t.Fatalf("strangers on survives a save: %s", b)
	}
}

// companion-ai strangers on and off record the owner's word, and the bare
// status says, on the owner's own key, that passers-by are held off until
// the owner lets them.
func TestCompanionAIStrangersDefaultWording(t *testing.T) {
	owner, _, _, _ := harmWorld(t, `off`)
	m, _ := senderModule(`https://api.example.invalid`, true)
	withWebDomain(t, `example.org`)
	m.cfg.PlayerKeys, m.cfg.RelayOrigin = true, `https://keys.example.org`
	m.relays = newRelayTable()
	m.relays.ready(1, `player-model`)
	events.DrainQueuedMessagesForTest(1)
	say := func(arg string) string {
		t.Helper()
		if _, err := m.cmdAI(arg, owner, nil, 0); err != nil {
			t.Fatal(err)
		}
		sent := events.DrainQueuedMessagesForTest(1)
		if len(sent) != 1 {
			t.Fatalf("%q: one reply, got %q", arg, sent)
		}
		for _, line := range strings.Split(strings.TrimRight(sent[0], "\n"), "\n") {
			if n := len([]rune(ansiTag.ReplaceAllString(line, ``))); n > 80 {
				t.Fatalf("%q: a line of %d columns: %q", arg, n, line)
			}
		}
		return strings.Join(strings.Fields(sent[0]), ` `)
	}
	if got := say(`strangers`); !strings.Contains(got, `your own key`) || !strings.Contains(got, `strangers on`) {
		t.Fatalf("by default on the owner's key, the status says they are held off and how to allow them: %q", got)
	}
	if got := say(`strangers on`); !m.bonds.Users[1].StrangersOn || m.bonds.Users[1].StrangersOff || !strings.Contains(got, `paid for from your key`) {
		t.Fatalf("strangers on is recorded and says who pays: %q %+v", got, m.bonds.Users[1])
	}
	if !m.strangerMayPrompt(1, 2) {
		t.Fatal("and passers-by may now prompt calls on the owner's key")
	}
	if got := say(`strangers off`); !m.bonds.Users[1].StrangersOff || m.bonds.Users[1].StrangersOn || !strings.Contains(got, `set lines`) {
		t.Fatalf("strangers off is recorded: %q %+v", got, m.bonds.Users[1])
	}
}

// Passers-by together may spend only so much of one owner's companion in a
// day, on either key, however many of them there are.
func TestStrangerTokensPerOwnerCapsThemTogether(t *testing.T) {
	if got := buildConfig(nil).StrangerTokensPerOwner; got != 100000 {
		t.Fatalf("default StrangerTokensPerOwner is 100000, got %d", got)
	}
	for _, rt := range []route{{kind: routeServer}, {kind: routeRelay, model: `player-model`}} {
		m := relayModule(t)
		m.cfg.DailyTokenBudget, m.cfg.StrangerDailyTokens, m.cfg.StrangerTokensPerOwner = 100000, 1000, 1500
		if !m.reserveRoute(rt, 5, 2, 900) || !m.reserveRoute(rt, 5, 3, 500) {
			t.Fatalf("%v: two passers-by within both caps are admitted", rt.kind)
		}
		if m.reserveRoute(rt, 5, 4, 200) {
			t.Fatalf("%v: a third, within their own allowance, would overshoot the owner's cap", rt.kind)
		}
		if !m.reserveRoute(rt, 6, 4, 200) {
			t.Fatalf("%v: another owner's companion has its own cap", rt.kind)
		}
		m.settleRoute(rt, 5, 2, 900, 100)
		if m.strangersFor[5] != 600 {
			t.Fatalf("%v: a settlement gives back what was not used, got %d", rt.kind, m.strangersFor[5])
		}
		if !m.reserveRoute(rt, 5, 4, 200) {
			t.Fatalf("%v: and the room it frees is usable", rt.kind)
		}
	}

	m := relayModule(t)
	m.cfg.StrangerTokensPerOwner = 1500
	c := &controller{ownerUserId: 5, instanceId: 42}
	m.bonds = bondState{Users: map[int]*bondRecord{5: {StrangersOn: true}}}
	m.rollDay()
	m.strangersFor[5] = 1500
	bram := &users.UserRecord{UserId: 2, Character: &characters.Character{Name: `Bram`}}
	if m.strangerMayAsk(bram, c) {
		t.Fatal("a spent owner's cap queues nothing more from passers-by")
	}
}

// The day's stranger spend per owner survives a restart with the rest of
// the budget file.
func TestStrangersForIsKeptWithTheBudget(t *testing.T) {
	st := budgetState{Day: `2026-09-25`, StrangersFor: map[int]int{5: 1234}}
	b, err := yaml.Marshal(&st)
	if err != nil {
		t.Fatal(err)
	}
	var back budgetState
	if err := yaml.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.StrangersFor[5] != 1234 {
		t.Fatalf("strangers_for survives a save: %s", b)
	}
}

// A fight a passer-by started is theirs to pay for: they are named as its
// starter, its plans are billed to them without refusing her anything, and
// a fight her owner picked stays the owner's.
func TestFightAPasserByStartedIsTheirs(t *testing.T) {
	owner, bram, _, her := harmWorld(t, `on`)
	c := &controller{ownerUserId: 1, instanceId: her.InstanceId, lastAttackBy: map[int]int64{}}
	now := time.Now().Unix()
	enemies := map[int]string{2: `Bram`}

	if got := fightStarter(c, her, owner, enemies, now); got != 2 {
		t.Fatalf("a passer-by fighting them whom neither was fighting started it, got %d", got)
	}
	owner.Character.SetAggro(2, 0, characters.DefaultAttack)
	if owner.Character.CurrentCombatTarget().UserId != 2 {
		t.Fatal("fixture: the owner is fighting Bram")
	}
	if got := fightStarter(c, her, owner, enemies, now); got != 0 {
		t.Fatalf("a fight her owner picked is the owner's, got %d", got)
	}
	c.lastAttackBy[2] = now - 5
	if got := fightStarter(c, her, owner, enemies, now); got != 2 {
		t.Fatalf("a passer-by who attacked her started it, whoever hit back, got %d", got)
	}
	c.lastAttackBy[2] = now - 700
	if got := fightStarter(c, her, owner, enemies, now); got != 0 {
		t.Fatalf("an attack long ago is not this fight, got %d", got)
	}
	_ = bram

	plan := stimulus{Kind: `fight`, Text: `start: Bram`, PaidBy: 2}
	over := stimulus{Kind: `fight_over`, Text: `over`, FromOwner: true, PaidBy: 2}
	if strangerBehind([]stimulus{plan}, 1) != 2 || strangerBehind([]stimulus{over}, 1) != 2 {
		t.Fatal("the plans and the end of their fight are billed to the passer-by")
	}
	if promptedBy(plan, 1) != 2 {
		t.Fatal("and never share a call with her owner's words")
	}
	if !ownerPrompted([]stimulus{plan}) {
		t.Fatal("paying for it refuses her nothing: she may still defend herself as she sees fit")
	}
	if strangerBehind([]stimulus{{Kind: `fight`, PaidBy: 1}}, 1) != 0 {
		t.Fatal("the owner is never a passer-by")
	}
}

// requestPlan carries the fight's starter on the stimulus it queues.
func TestRequestPlanCarriesTheStarter(t *testing.T) {
	m, c := senderModule(`https://api.example.invalid`, true)
	c.inFlight = true // queue only
	c.fight = &fightState{StarterUserId: 2}
	m.requestPlan(c, 1, `start`, `Bram`)
	if len(c.pending) != 1 || c.pending[0].PaidBy != 2 {
		t.Fatalf("the plan is queued as the starter's: %+v", c.pending)
	}
}
