package aicompanion

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/apiframework"
	"github.com/GoMudEngine/GoMud/internal/companionai"
)

// The model's wire format, the HTTP transport, moderation, the model list,
// the server's one daily budget and its breaker all live in
// internal/apiframework, shared with every other feature that calls a
// model (bauble naming). What is the companion's own stays here: the
// consent door (admit), the route (her owner's key through their browser,
// or the server's), and the per-owner and per-passer-by allowances.

// chatMessage, toolCall and the tool types are the framework's.
type (
	chatMessage  = apiframework.Message
	toolCall     = apiframework.ToolCall
	toolFunction = apiframework.ToolFunction
	toolSpec     = apiframework.ToolSpec
	toolDef      = apiframework.ToolDef
)

// modelCall is everything the background goroutine needs. It is built under
// the mud lock and then owned by the goroutine; it holds no game pointers.
type modelCall struct {
	BaseURL     string
	APIKey      string
	Model       string
	Timeout     time.Duration
	MaxTokens   int
	Temperature float64
	Messages    []chatMessage
	SchemaName  string
	Schema      map[string]any
	Effort      string // reasoning effort for reasoning models; empty = not sent
	Retry       bool   // retry once on a transient failure (429, 5xx, timeout)
	Tools       []toolSpec
	ToolChoice  string          // auto, none; empty = not sent
	Ctx         context.Context // cancelled when the answer can no longer be used

	// OwnerUserId is the player whose companion, and so whose words and
	// doings, this request carries. The door refuses it unless that player
	// has agreed; left at 0 it is refused outright.
	OwnerUserId int

	// Route is who pays for the call and how it travels, set once by
	// applyRoute when the call is built.
	Route route

	// ticket is the server key breakers' leave for this logical call
	// (apiframework.Allow), taken at its first send and kept across its
	// retry and tool rounds, so one decision is one outcome.
	ticket   apiframework.Ticket
	admitted bool
	// ticketOut, when set, is given the ticket the moment it is taken, so
	// the goroutine that made the call can hand it back even if a panic
	// cuts the call short.
	ticketOut *apiframework.Ticket
}

// modelResult is what comes back: the raw JSON content, which the caller
// parses against the schema it asked for. Err is set on any failure.
type modelResult struct {
	Content  string
	Tokens   int
	Latency  time.Duration
	Err      error
	Status   int  // HTTP status, 0 when the request never got an answer
	Canceled bool // the caller gave up on it; not the provider's fault

	// Sent is the request having left for the provider (or the owner's
	// browser), so it may have been billed whatever came back. Estimated
	// is Tokens being the prompt estimate for a sent request that
	// reported no usage (callModelOnce).
	Sent      bool
	Estimated bool

	// ToolCalls are the model's requests for more information, when it
	// asked instead of answering. ToolsUsed counts them across a decision.
	ToolCalls []toolCall
	ToolsUsed int

	// Filled by the decision path on the goroutine: the parsed decision
	// with any lines the moderation check flagged already removed.
	Parsed    *Decision
	ParseErr  error
	Moderated int // lines removed by moderation

	// Ticket is the breakers' leave the call went out on (server key
	// only), handed back with its outcome by routeResult.
	Ticket   apiframework.Ticket
	Admitted bool
}

// errServerResting is a server-key call the breakers would not let through
// just as it was about to leave (another call is probing a half-open
// breaker). It never left; it is nobody's failure.
var errServerResting = errors.New(`not sent: the server key's breaker is resting`)

// errNoConsent is the door refusing a request for a player who has not
// agreed that anything of theirs may be sent. The request never left.
var errNoConsent = errors.New(`not sent: the companion's owner has not agreed to the model`)

// outboundKind says what a request carries. The zero value is the guarded
// kind, so a request nobody thought to classify is treated as carrying a
// player's words.
type outboundKind int

const (
	// carriesPlayerData is every chat completion and every moderation
	// check: a prompt built from her mind, or lines she means to say that
	// were shaped by it.
	carriesPlayerData outboundKind = iota
	// carriesNoPlayerData carries nothing of any player's: the model list,
	// and a bauble name asked through a player's own key (relayFor).
	carriesNoPlayerData
)

// admit is the one door through which anything of a player's leaves for a
// provider. Every caller already checks consent before it builds a
// request; this is the check that holds when one of them forgets. It is
// keyed on the owner the request carries, reads the ledger rather than the
// bond records because it runs off the mud lock, and fails closed: no
// owner, or no ledger, is no send. There are two ways out, the server's key
// over HTTP (apiframework.Post, with doorFor as its Admit hook) and the
// owner's key through their browser (sendRelay), and each passes admit
// before it reaches its transport.
func admit(gate *consentLedger, ownerUserId int, kind outboundKind, path string) error {
	if kind != carriesNoPlayerData && !gate.allows(ownerUserId) {
		gate.noteRefusal(ownerUserId, path)
		return errNoConsent
	}
	return nil
}

// doorFor is admit as the framework's Admit hook, run immediately before an
// HTTP request leaves.
func doorFor(gate *consentLedger, ownerUserId int, kind outboundKind) apiframework.Admit {
	return func(path string) error { return admit(gate, ownerUserId, kind, path) }
}

// sendRelay takes a request body to the owner's browser, through the same
// door, and returns the provider's status and raw reply.
func sendRelay(ctx context.Context, gate *consentLedger, ownerUserId int, kind outboundKind, calls *pendingRelays,
	body []byte, via relaySender) (int, []byte, error) {
	if err := admit(gate, ownerUserId, kind, `relay`); err != nil {
		return 0, nil, err
	}
	return calls.do(ctx, ownerUserId, body, via)
}

// callModel performs a chat completions request, retrying once after a
// short pause when the failure looks transient and the call allows it.
func (m *AICompanionModule) callModel(c modelCall) modelResult {
	if c.Route.kind != routeRelay && !c.admitted {
		// Leave from the server key's breakers, asked immediately before
		// the first send of this logical call (a half-open breaker lets
		// exactly one caller through as its probe).
		t, ok := m.fw().Allow(apiframework.ConsumerCompanion, time.Now())
		if !ok {
			return modelResult{Err: errServerResting}
		}
		c.ticket, c.admitted = t, true
		if c.ticketOut != nil {
			*c.ticketOut = t
		}
	}
	res := m.callModelOnce(c)
	if c.Retry && transient(res) {
		time.Sleep(1500 * time.Millisecond)
		again := m.callModelOnce(c)
		again.Latency += res.Latency + 1500*time.Millisecond
		again.Tokens += res.Tokens
		res = again
	}
	res.Ticket, res.Admitted = c.ticket, c.admitted
	return res
}

// transient reports failures worth one retry: rate limits, server errors
// and requests that never got an answer.
func transient(r modelResult) bool {
	if r.Err == nil || r.Canceled || errors.Is(r.Err, errNoConsent) || relayFinal(r.Err) {
		return false
	}
	return r.Status == 0 || r.Status == http.StatusTooManyRequests || r.Status >= 500
}

// callModelOnce performs one blocking chat completions request with the
// call's strict JSON schema. It must only ever run on a goroutine that does
// not hold the mud lock. The API key is sent in a header and never logged or
// returned.
//
// What it reports as spent is what the budgets are settled with, so it is
// made trustworthy here, once, for every caller (apiframework.Charged): a
// request that left but came back with no usage is counted at its prompt
// estimate plus its MaxTokens, and a count relayed through a player's
// browser, which that player can write, is held between nothing and the
// most this one request could have cost.
func (m *AICompanionModule) callModelOnce(c modelCall) modelResult {
	res := m.exchangeOnce(c)
	prompt := estimateTokens(c.Messages) + requestOverhead(c)
	res.Tokens, res.Estimated = apiframework.Charged(res.Tokens, res.Sent, res.Status, prompt, c.MaxTokens,
		c.Route.kind == routeRelay)
	return res
}

// neverConnected reports an HTTP failure that happened before any byte of
// the request could have left.
func neverConnected(err error) bool { return apiframework.NeverConnected(err) }

// exchangeOnce is one request and its reply, on whichever transport the
// call's route names, with nothing counted yet (callModelOnce).
func (m *AICompanionModule) exchangeOnce(c modelCall) modelResult {
	start := time.Now()
	res := modelResult{}

	body, err := apiframework.Chat{
		Model:       c.Model,
		Messages:    c.Messages,
		Tools:       c.Tools,
		ToolChoice:  c.ToolChoice,
		SchemaName:  c.SchemaName,
		Schema:      c.Schema,
		MaxTokens:   c.MaxTokens,
		Temperature: c.Temperature,
		Effort:      c.Effort,
	}.Body()
	if err != nil {
		res.Err = err
		return res
	}

	parent := c.Ctx
	if parent == nil {
		parent = context.Background()
	}
	if err := parent.Err(); err != nil {
		// Given up on before it left: nothing was sent, nothing spent.
		res.Err, res.Canceled = err, errors.Is(err, context.Canceled)
		return res
	}

	if c.Route.kind == routeRelay {
		// The owner's own key, through their browser. The wait covers the
		// browser's round trip as well as the provider, so it is the
		// relay's own deadline. The per-owner breaker is fed once per call
		// by routeResult at the call site, not here.
		ctx, cancel := context.WithTimeout(parent, time.Duration(m.cfg.RelayTimeoutSeconds)*time.Second)
		defer cancel()
		status, raw, err := sendRelay(ctx, &m.consent, c.OwnerUserId, carriesPlayerData, m.relayCalls, body, m.relayVia())
		res.Latency = time.Since(start)
		res.Status = status
		// It left unless the door refused it or there was no browser to
		// take it; a relay that went away after taking it may already
		// have posted it.
		res.Sent = err == nil || !(errors.Is(err, errNoConsent) || errors.Is(err, errRelayUnsent))
		if err != nil {
			res.Err = err
			res.Canceled = errors.Is(err, context.Canceled) || errors.Is(parent.Err(), context.Canceled)
			return res
		}
		if status != http.StatusOK {
			// The body is the provider's own text about the owner's own
			// account, relayed by a page the server does not control: it is
			// neither kept in the error nor logged. The status says enough.
			res.Err = fmt.Errorf(`model API status %d through the owner's own key`, status)
			return res
		}
		return decodeChatResponse(res, status, raw)
	}

	ctx, cancel := context.WithTimeout(parent, c.Timeout)
	defer cancel()
	ex := apiframework.Post(ctx, apiframework.Endpoint{BaseURL: c.BaseURL, APIKey: c.APIKey}, `/chat/completions`,
		body, apiframework.CarriesPlayerData, doorFor(&m.consent, c.OwnerUserId, carriesPlayerData))
	res.Latency = ex.Latency
	res.Sent = ex.Sent
	res.Status = ex.Status
	if ex.Err != nil {
		res.Err = ex.Err
		res.Canceled = errors.Is(ex.Err, context.Canceled) || errors.Is(parent.Err(), context.Canceled)
		return res
	}
	return decodeChatResponse(res, ex.Status, ex.Raw)
}

// relayVia is how a relay request reaches the owner's browser.
func (m *AICompanionModule) relayVia() relaySender {
	if m.relaySend != nil {
		return m.relaySend
	}
	return companionai.SendRelay
}

// maxToolCallsPerReply is the most questions one reply may put to the game.
const maxToolCallsPerReply = apiframework.MaxToolCallsPerReply

// decodeChatResponse reads a provider's chat completions reply into res,
// whichever way it came back: over HTTP or through the owner's browser.
func decodeChatResponse(res modelResult, status int, raw []byte) modelResult {
	r := apiframework.DecodeChat(status, raw)
	res.Tokens = r.Tokens
	res.ToolCalls = r.ToolCalls
	res.Content = r.Content
	res.Err = r.Err
	return res
}

// moderate checks texts with the moderation endpoint (F19.1) and returns
// which were flagged. On any error it returns nil: moderation failing must
// not silence the companion, and the in-character filters still apply. The
// texts are what she means to say, shaped by her mind, so the check goes
// through the same door as the call that produced them.
func (m *AICompanionModule) moderate(ownerUserId int, baseURL string, apiKey string, model string, timeout time.Duration, texts []string) ([]bool, error) {
	return apiframework.Moderate(apiframework.Endpoint{BaseURL: baseURL, APIKey: apiKey}, model, timeout, texts,
		apiframework.CarriesPlayerData, doorFor(&m.consent, ownerUserId, carriesPlayerData))
}

// moderateDecision removes flagged speech (and a flagged sayto) from a
// decision. Runs on the model goroutine; touches no game state.
// moderateDecision checks what the companion means to say. strict is the
// policy for a check that could not be made: for speech a stranger
// prompted the lines are dropped (fail closed), and for the owner's own
// conversation they are kept (fail open), so a moderation outage costs a
// server its harassment cover rather than its companions.
func (m *AICompanionModule) moderateDecision(ownerUserId int, d *Decision, baseURL string, apiKey string, model string, timeout time.Duration, strict bool) int {
	var texts []string
	for _, l := range d.Speech {
		texts = append(texts, l.Text)
	}
	hasSayto := d.Action.Verb == `sayto` && strings.TrimSpace(d.Action.Query) != ``
	if hasSayto {
		texts = append(texts, d.Action.Query)
	}
	flags, err := m.moderate(ownerUserId, baseURL, apiKey, model, timeout, texts)
	if err != nil || flags == nil {
		if !strict {
			return 0
		}
		// The check failed and these words were not the owner's to prompt:
		// say nothing rather than say something unchecked.
		removed := len(d.Speech)
		d.Speech = nil
		if hasSayto {
			d.Action = ActionProposal{Verb: `none`}
			removed++
		}
		return removed
	}
	removed := 0
	kept := d.Speech[:0]
	for i, l := range d.Speech {
		if flags[i] {
			removed++
			continue
		}
		kept = append(kept, l)
	}
	d.Speech = kept
	if hasSayto && flags[len(flags)-1] {
		d.Action = ActionProposal{Verb: `none`}
		removed++
	}
	return removed
}

// listModels returns the model ids the key can use (GET /models), or nil
// when the list cannot be read. It carries the key and nothing of any
// player's, so it needs no door.
func listModels(baseURL string, apiKey string) map[string]bool {
	return apiframework.ListModels(apiframework.Endpoint{BaseURL: baseURL, APIKey: apiKey})
}
