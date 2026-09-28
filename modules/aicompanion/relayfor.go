package aicompanion

import (
	"context"
	"errors"
	"time"

	"github.com/GoMudEngine/GoMud/internal/apiframework"
)

// relayFor lends the companion's key relay to other features through
// apiframework (bauble naming), for what each player has allowed on the key
// page. It is the only way anything but a companion reaches a player's own
// key, and it carries only what the other feature built: a bauble request
// holds authored room text and nothing of any player's, so it passes the
// door as carriesNoPlayerData. Nothing is reserved against the server's
// budget; the player's finds breaker, not their companion's, is fed.
type relayFor struct {
	m *AICompanionModule
}

// Model is the player's relay model when their key may be used for purpose
// now.
func (r relayFor) Model(userId int, purpose string) (string, bool) {
	m := r.m
	if userId <= 0 || m.relays == nil || !m.playerKeysOffered() {
		return ``, false
	}
	return m.relays.liveFor(userId, purpose, time.Now())
}

// Send posts body through the player's browser.
func (r relayFor) Send(ctx context.Context, userId int, body []byte, carries apiframework.Carries) (int, []byte, bool, error) {
	m := r.m
	kind := carriesPlayerData
	if carries == apiframework.CarriesNoPlayerData {
		kind = carriesNoPlayerData
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(m.cfg.RelayTimeoutSeconds)*time.Second)
	defer cancel()
	status, raw, err := sendRelay(ctx, &m.consent, userId, kind, m.relayCalls, body, m.relayVia())
	sent := err == nil || !(errors.Is(err, errNoConsent) || errors.Is(err, errRelayUnsent))
	return status, raw, sent, err
}

// Result feeds the player's FINDS breaker (relayTable.findsResult), never
// the one their companion runs on: a bauble request the player's provider
// will not serve pauses bauble naming on that key and nothing else. What
// counts follows the companion's rule (routeResult): a door that refused, a
// relay that went away, a reply refused for looking like a key, or a call
// given up on is nobody's failure. No fallback notice is sent: the
// companion did not fall back.
func (r relayFor) Result(userId int, err error) {
	m := r.m
	if m.relays == nil || errors.Is(err, errNoConsent) || errors.Is(err, errRelayGone) ||
		errors.Is(err, errRelayKeyShaped) || errors.Is(err, context.Canceled) {
		return
	}
	m.relays.findsResult(userId, err != nil, time.Now(), m.cfg)
}
