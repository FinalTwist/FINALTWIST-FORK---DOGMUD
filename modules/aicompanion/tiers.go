package aicompanion

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Three tiers pay for a companion's thinking. Tier 1 is nobody: she answers
// with her authored lines. Tier 2 is her owner, through their own key held
// in their browser on the relay origin (the relay). Tier 3 is the server's
// key, bounded by the daily budgets and the global breaker. Every call is
// routed once, when it is built (applyRoute), and the route it carries
// decides what it is reserved against, how it settles and which breaker
// its outcome reaches.

// routeKind is who pays for a call and how it travels.
type routeKind int

const (
	routeNone   routeKind = iota // tier 1: set lines, nothing leaves
	routeRelay                   // tier 2: the owner's own key, through their browser
	routeServer                  // tier 3: the server's key
)

type route struct {
	kind  routeKind
	model string // tier 2 only: the owner's chosen model, used for every tier
}

// validRelayOrigin accepts only "https://host[:port]" with no path, query,
// fragment or user, and not on the game's own host: the key must live on an
// origin the game page cannot read.
func validRelayOrigin(origin string, webDomain string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != `https` || u.Host == `` || u.Opaque != `` || u.User != nil ||
		(u.Path != `` && u.Path != `/`) || u.RawQuery != `` || u.Fragment != `` || u.ForceQuery {
		return false
	}
	return !strings.EqualFold(u.Hostname(), gameHostname(webDomain))
}

// gameHostname is the game's own host, read from FilePaths.WebDomain
// exactly as gameOrigin reads it (a pasted scheme, path or port is
// dropped, case is ignored), or "" when it is not a plain host. The relay
// page and the relay origin check share it, so what one calls the game's
// host the other does too.
func gameHostname(webDomain string) string {
	origin := gameOrigin(webDomain)
	if origin == `` {
		return ``
	}
	u, err := url.Parse(origin)
	if err != nil {
		return ``
	}
	return u.Hostname()
}

// playerKeysOffered reports whether tier 2 is on offer at all.
func (m *AICompanionModule) playerKeysOffered() bool {
	return m.cfg.Enabled && m.cfg.PlayerKeys &&
		validRelayOrigin(m.cfg.RelayOrigin, string(configs.GetFilePathsConfig().WebDomain))
}

// relayTable is which owners have a live, unlocked relay, and each one's
// own breaker. It is read from model goroutines and written from the
// connection goroutine, so it has its own lock.
type relayTable struct {
	mu     sync.Mutex
	owners map[int]*relayOwner
}

type relayOwner struct {
	model        string
	failures     int
	breakerUntil time.Time
	noticeSent   bool // the owner was told this relay session that she fell back
}

func newRelayTable() *relayTable { return &relayTable{owners: map[int]*relayOwner{}} }

// ready records that the owner's relay is up with this model. A relay that
// comes back keeps its breaker: reloading the page must not reset it. It
// starts a new relay session for the fallback notice, which may be given
// once more.
func (t *relayTable) ready(userId int, model string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if o := t.owners[userId]; o != nil {
		o.model = strings.TrimSpace(model)
		o.noticeSent = false
		return
	}
	t.owners[userId] = &relayOwner{model: strings.TrimSpace(model)}
}

// gone records that the owner's relay is down (locked, closed, logged out).
// Its breaker is kept, for the same reason.
func (t *relayTable) gone(userId int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if o := t.owners[userId]; o != nil {
		o.model = ``
	}
}

// live returns the owner's model when their relay is up and their own
// breaker is closed.
func (t *relayTable) live(userId int, now time.Time) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	o := t.owners[userId]
	if o == nil || o.model == `` || now.Before(o.breakerUntil) {
		return ``, false
	}
	return o.model, true
}

// failure counts one failed call against the owner's own breaker, and opens
// it for BreakerSeconds after BreakerErrors in a row.
func (t *relayTable) failure(userId int, now time.Time, cfg Config) {
	t.mu.Lock()
	defer t.mu.Unlock()
	o := t.owners[userId]
	if o == nil {
		return
	}
	o.failures++
	if cfg.BreakerErrors > 0 && o.failures >= cfg.BreakerErrors {
		o.breakerUntil = now.Add(time.Duration(cfg.BreakerSeconds) * time.Second)
		o.failures = 0
	}
}

// noticeDue reports, once per relay session, that the owner should be told
// their companion fell back on set lines.
func (t *relayTable) noticeDue(userId int) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	o := t.owners[userId]
	if o == nil || o.noticeSent {
		return false
	}
	o.noticeSent = true
	return true
}

func (t *relayTable) success(userId int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if o := t.owners[userId]; o != nil {
		o.failures = 0
	}
}

// route decides who pays for a call on this owner's behalf: their own
// relay first, then the server's key, else nothing and set lines. A
// passer-by talking to her is still routed by her owner: the owner's key
// pays, and StrangerDailyTokens bounds what the passer-by may spend of it.
func (m *AICompanionModule) route(ownerId int) route {
	if ownerId > 0 && m.relays != nil && m.playerKeysOffered() {
		if model, ok := m.relays.live(ownerId, time.Now()); ok {
			return route{kind: routeRelay, model: model}
		}
	}
	if m.cfg.Enabled && m.apiKey() != `` {
		return route{kind: routeServer}
	}
	return route{kind: routeNone}
}

// applyRoute fills in who pays for a call and, for a player's own key, the
// model they chose. A relay call carries nothing of the server's: no key,
// no endpoint, and no reasoning effort, which a player's provider may not
// accept. Every modelCall is built through here, once, and keeps its route
// to the end: the reservation, the settlement and the breaker all read it.
func (m *AICompanionModule) applyRoute(c *modelCall) {
	c.Route = m.route(c.OwnerUserId)
	if c.Route.kind == routeRelay {
		c.Model, c.Effort, c.APIKey, c.BaseURL = c.Route.model, ``, ``, ``
	}
}

// reserveRoute holds a call's worst case against whoever pays for it, in
// one check-and-hold step. The server's key is held against the server's
// budget and the owner's or passer-by's allowance (tryReserveFor). A
// player's own key spends nothing of the server's, so it is held against
// nothing, except that a passer-by's question is still held against their
// StrangerDailyTokens and the owner's StrangerTokensPerOwner (strangerFits):
// the owner's key is not theirs to spend without end.
func (m *AICompanionModule) reserveRoute(r route, ownerId int, askerId int, tokens int) bool {
	switch r.kind {
	case routeServer:
		return m.tryReserveFor(ownerId, askerId, tokens)
	case routeRelay:
		if askerId <= 0 {
			return true
		}
		m.rollDay()
		if !m.strangerFits(ownerId, askerId, tokens) {
			return false
		}
		m.chargeStrangerFor(ownerId, askerId, tokens)
		return true
	}
	return false
}

// settleRoute settles a reservation made by reserveRoute with the same
// route, against the same payer, exactly once.
func (m *AICompanionModule) settleRoute(r route, ownerId int, askerId int, reserved int, used int) {
	switch r.kind {
	case routeServer:
		m.settleFor(ownerId, askerId, reserved, used)
	case routeRelay:
		if askerId <= 0 {
			return
		}
		// The count came back through the owner's browser, which the
		// owner can write: it may lower a passer-by's charge below the
		// reservation, never raise it past it, and never below nothing.
		used = max(0, min(used, reserved))
		m.rollDay()
		m.chargeStrangerFor(ownerId, askerId, used-reserved)
	}
}

// routeResult feeds a call's outcome to the breaker of whoever paid: the
// owner's own for their key, so one player's broken provider cannot stop
// everyone's companions, and the global one for the server's.
func (m *AICompanionModule) routeResult(r route, ownerId int, err error, now time.Time) {
	if r.kind != routeRelay {
		m.breakerResult(err, now)
		return
	}
	if m.relays == nil || errors.Is(err, errNoConsent) || errors.Is(err, errRelayGone) ||
		errors.Is(err, errRelayKeyShaped) || errors.Is(err, context.Canceled) {
		// The door refused a request that never left, the owner's page
		// went away, the guard refused a reply for looking like a key, or
		// the module gave up on the answer: none of them is the provider
		// failing.
		return
	}
	if err == nil {
		m.relays.success(ownerId)
		return
	}
	m.relays.failure(ownerId, now, m.cfg)
	m.noticeFallback(ownerId)
}

// noticeFallback tells the owner, once per relay session, that their
// companion fell back on set lines because their key's provider did not
// answer: in plain words, never the error, which may carry the provider's
// own text. It is reached only for a failure routeResult counts against the
// owner, so a relay that went away or a call the module gave up on says
// nothing. Runs under the mud lock, as every routeResult does.
func (m *AICompanionModule) noticeFallback(ownerId int) {
	if !m.relays.noticeDue(ownerId) {
		return
	}
	name := `Your companion`
	if c := m.ctrls[ownerId]; c != nil && c.profile != nil {
		name = c.profile.Name
	}
	m.tellOwner(ownerId, fmt.Sprintf(`(%s falls back on a few set words: your key's provider did not answer.)`, name))
}

// tellOwner sends a player a system line, wrapped at 80 columns: the
// system category is never wrapped for them.
func (m *AICompanionModule) tellOwner(userId int, text string) {
	text = messaging.WrapAnsi(text, 80)
	if m.tell != nil {
		m.tell(userId, text)
		return
	}
	if u := users.GetByUserId(userId); u != nil {
		u.SendText(messaging.CategorySystem, text)
	}
}
