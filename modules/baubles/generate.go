package baubles

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/GoMudEngine/GoMud/internal/apiframework"
	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// schemaOverhead is the reply schema, sent with every call and not in the
// messages, in round numbers.
const schemaOverhead = 300

var (
	errNoRoute     = errors.New(`no key to name it with`)
	errBreakerOpen = errors.New(`the server key's breaker is open`)
)

// generate is the baubles.GeneratorFunc this module installs. It runs on a
// delivery goroutine WITHOUT the mud lock (see actions/search_bauble.go), so
// it touches no game state. Any error sends the find down the
// generic-trinket path, and the player never sees it.
//
// The route: the finder's own key first, when they allowed it on the key
// page (apiframework.PurposeFinds); then the server's key, reserved against
// the one daily budget every feature shares; else no name.
func (m *BaublesModule) generate(ctx context.Context, req baubles.GenRequest) (baubles.GenResult, error) {
	// A fixed number of calls at once. A find beyond that is not queued
	// (queuing would only make the player wait longer): it is a generic
	// trinket.
	m.mu.Lock()
	slots := m.slots
	m.mu.Unlock()
	select {
	case slots <- struct{}{}:
		defer func() { <-slots }()
	default:
		return baubles.GenResult{}, errors.New(`all generation slots busy`)
	}

	cfg := m.snapshot()
	msgs := buildMessages(req)
	chat := apiframework.Chat{
		Model:       cfg.Model,
		Messages:    msgs,
		SchemaName:  baubles.ReplySchemaName,
		Schema:      baubles.ReplySchema(),
		MaxTokens:   cfg.MaxCompletionTokens,
		Temperature: cfg.Temperature,
		Effort:      cfg.ReasoningEffort,
	}

	content, tokens, model, playerKey, report, err := m.name(ctx, cfg, req, chat)
	m.count(playerKey, err != nil)
	if cfg.LogRequests {
		mudlog.Info(`baubles`, `action`, `model call`, `zone`, req.Place.Zone, `tier`, string(req.Tier),
			`model`, model, `playerKey`, playerKey, `tokens`, tokens, `error`, errString(err))
	}
	if err != nil {
		report(err)
		return baubles.GenResult{}, fmt.Errorf(`model call: %w`, err)
	}

	// The find's outcome for the breakers is whether its answer can be
	// used, not only whether one came back: a model that keeps ignoring the
	// schema pauses baubles' own breaker (never the provider's, which it
	// answered) instead of being paid for again and again.
	reply, err := baubles.ParseReply(content)
	if err == nil {
		_, err = baubles.CleanReply(reply)
	}
	report(err)
	if err != nil {
		return baubles.GenResult{}, err
	}

	moderated, err := m.moderate(cfg, reply, playerKey)
	if err != nil {
		return baubles.GenResult{}, err
	}

	return baubles.GenResult{
		Reply:         reply,
		Generator:     baubles.GeneratorOpenAI,
		Model:         model,
		PromptVersion: PromptVersion,
		Tokens:        tokens,
		Moderated:     moderated,
		PlayerKey:     playerKey,
	}, nil
}

// name makes the call on the first route that is open and returns the
// reply's content, and report, which takes the find's final outcome (its
// answer parsed and checked, or the failure) to that route's breaker,
// exactly once. A failure on the finder's own key is reported there and
// falls back to the server's key.
func (m *BaublesModule) name(ctx context.Context, cfg Config, req baubles.GenRequest, chat apiframework.Chat) (content string, tokens int, model string, playerKey bool, report func(error), err error) {
	// A pickpocket's find takes this route too, on the thief's own key. Its
	// naming starts at the moment the roll succeeds, so a thief watching
	// their browser's network traffic can learn the outcome before the
	// reveal; the owner accepted that so every find follows one order.
	if cfg.UsePlayerKeys && req.FinderUserId > 0 {
		if r := apiframework.PlayerRelay(); r != nil {
			if relayModel, ok := r.Model(req.FinderUserId, apiframework.PurposeFinds); ok {
				content, tokens, report, err = viaPlayer(ctx, r, req.FinderUserId, relayModel, chat)
				if err == nil {
					return content, tokens, relayModel, true, report, nil
				}
				report(err)
				if ctx.Err() != nil {
					return ``, 0, relayModel, true, func(error) {}, err
				}
			}
		}
	}
	content, tokens, report, err = viaServer(ctx, cfg, chat)
	return content, tokens, cfg.Model, false, report, err
}

// canceled is a find given up on (a copyover's flush): nobody's failure.
// Its own time running out is not: that is the provider not answering.
func canceled(ctx context.Context) bool { return errors.Is(ctx.Err(), context.Canceled) }

// viaPlayer names the find through the finder's own key, in their browser.
// The player's provider may not accept a reasoning effort, and the relay
// sets its own model, so neither is the server's. Nothing is reserved
// against the server's budget; the player's finds breaker is fed.
func viaPlayer(ctx context.Context, r apiframework.Relay, userId int, model string, chat apiframework.Chat) (string, int, func(error), error) {
	// The outcome is held against the finder's key for finds only (the
	// relay keeps a breaker per purpose, and their companion's is never
	// touched), so a provider that cannot serve finds stops being asked.
	report := func(err error) {
		if !canceled(ctx) {
			r.Result(userId, err)
		}
	}
	chat.Model, chat.Effort = model, ``
	body, err := chat.Body()
	if err != nil {
		return ``, 0, func(error) {}, err
	}
	status, raw, _, err := r.Send(ctx, userId, body, apiframework.CarriesNoPlayerData)
	if err == nil && status != http.StatusOK {
		// The body is the provider's own text about the player's own
		// account: neither kept nor logged. The status says enough.
		err = &apiframework.StatusError{Status: status}
	}
	if err != nil {
		return ``, 0, report, err
	}
	reply := apiframework.DecodeChat(status, raw)
	return reply.Content, reply.Tokens, report, reply.Err
}

// viaServer names the find on the server's key: leave from the breakers
// (the baubles' own and the provider's, apiframework.Allow), the worst case
// reserved against the one daily budget, the call (retried once on a
// transient failure when RetryTransient), then the settlement. It returns
// report, through which the caller gives ONE outcome for the whole find,
// retries included, once its answer has been parsed and checked. Only a failure that says the provider or key is unwell reaches
// the provider breaker the companion shares (apiframework.ProviderFailure);
// a bauble model or schema the provider refuses pauses baubles only.
func viaServer(ctx context.Context, cfg Config, chat apiframework.Chat) (string, int, func(error), error) {
	none := func(error) {}
	s := apiframework.Server()
	if !s.HasKey() {
		return ``, 0, none, errNoRoute
	}
	body, err := chat.Body()
	if err != nil {
		return ``, 0, none, err
	}
	prompt := apiframework.EstimateTokens(chat.Messages) + schemaOverhead
	reserve := prompt + chat.MaxTokens
	if cfg.RetryTransient {
		reserve *= 2
	}
	ticket, ok := apiframework.Allow(apiframework.ConsumerBaubles, time.Now())
	if !ok {
		return ``, 0, none, errBreakerOpen
	}
	hold, err := apiframework.Reserve(apiframework.ConsumerBaubles, reserve)
	if err != nil {
		apiframework.Release(apiframework.ConsumerBaubles, ticket)
		return ``, 0, none, err
	}

	charged := 0
	var reply apiframework.Reply
	for attempt := 0; ; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSeconds)*time.Second)
		ex := apiframework.Post(callCtx, s.Endpoint, `/chat/completions`, body, apiframework.CarriesNoPlayerData, nil)
		cancel()
		if ex.Err == nil {
			reply = apiframework.DecodeChat(ex.Status, ex.Raw)
		} else {
			reply = apiframework.Reply{Err: ex.Err}
		}
		n, _ := apiframework.Charged(reply.Tokens, ex.Sent, ex.Status, prompt, chat.MaxTokens, false)
		charged += n
		if reply.Err == nil || ctx.Err() != nil || attempt > 0 || !cfg.RetryTransient || !transient(ex) {
			break
		}
		time.Sleep(1500 * time.Millisecond)
	}
	apiframework.Settle(hold, charged, reply.Err != nil)
	// ONE outcome for the find, given by the caller once the answer has
	// been parsed and checked (report); a find given up on is handed back
	// unjudged. A transport failure here is already its outcome.
	report := func(err error) {
		if canceled(ctx) {
			apiframework.Release(apiframework.ConsumerBaubles, ticket)
			return
		}
		apiframework.Record(apiframework.ConsumerBaubles, ticket, err, time.Now())
	}
	return reply.Content, charged, report, reply.Err
}

// transient reports a failure worth one retry: a rate limit, a server
// error, or no answer at all.
func transient(ex apiframework.Exchange) bool {
	return ex.Status == 0 || ex.Status == http.StatusTooManyRequests || ex.Status >= 500
}

// moderate checks the name and description when ModerateOutput is on,
// through the server's key (a player's key page reaches no moderation
// endpoint). The policy, decided and pinned by test:
//
//   - A flag always keeps the text out of the world: a generic trinket.
//   - A find named on the server's key whose check cannot be made (no
//     server key, the provider breaker open, the check failing) is kept
//     out too, as before: the server vouches for what its own key makes.
//   - A find named on the finder's own key whose check cannot be made is
//     accepted unmoderated (Moderated false in its record, where `bauble
//     show` and `bauble retire` find it), the same rule the AI companion
//     follows on a player's key. So a working player key never turns into a
//     trinket because the server's route is down.
//
// The check is free and is not a model call, so it reserves nothing and
// feeds no breaker; it does not try while the provider breaker is open.
func (m *BaublesModule) moderate(cfg Config, reply baubles.Reply, playerKey bool) (bool, error) {
	if !cfg.ModerateOutput {
		return false, nil
	}
	unavailable := func(cause error) (bool, error) {
		if playerKey {
			return false, nil
		}
		return false, cause
	}
	s := apiframework.Server()
	if !s.HasKey() {
		return unavailable(errNoRoute)
	}
	if apiframework.BreakerOpen(time.Now()) {
		return unavailable(errBreakerOpen)
	}
	flags, err := apiframework.Moderate(s.Endpoint, cfg.ModerationModel, time.Duration(cfg.TimeoutSeconds)*time.Second,
		[]string{reply.Name, reply.Description}, apiframework.CarriesNoPlayerData, nil)
	if err != nil {
		return unavailable(fmt.Errorf(`moderation: %w`, err))
	}
	for _, f := range flags {
		if f {
			return false, errors.New(`moderation flagged the reply`)
		}
	}
	return true, nil
}

func errString(err error) string {
	if err == nil {
		return ``
	}
	return err.Error()
}
