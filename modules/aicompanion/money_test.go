package aicompanion

import (
	"fmt"
	"strings"
	"testing"
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
