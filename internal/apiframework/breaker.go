package apiframework

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// The server key's circuit breakers, two levels of them:
//
//   - The PROVIDER breaker, one for every feature, since they share the key
//     and the provider. Only failures that say the provider or the key is
//     unwell feed it (ProviderFailure: no answer, a timeout, 401, 403, 408,
//     429, a 5xx). A reply the provider did give, even a refusal or a 400 for
//     one feature's model or schema, is the provider answering.
//   - A CONSUMER breaker per feature (ConsumerCompanion, ConsumerBaubles),
//     fed by every failure of that feature's own calls. A bauble model the
//     provider does not offer pauses baubles, never the companion.
//
// A call is let through only when both are (Allow). After
// APIFramework.BreakerErrors failures in a row a breaker opens for
// BreakerSeconds; when that is up it is HALF-OPEN: exactly one caller is
// let through as the probe (its Ticket says so), everyone else waits for
// its answer. The probe's success closes the breaker; its failure opens it
// again. A probe whose caller never reports at all expires after probeTTL,
// so the breaker cannot stick half-open. A player's own key has
// its own breakers, kept by whoever relays it.

type breaker struct {
	mu          sync.Mutex
	until       time.Time // open until
	tripped     bool      // has opened and not yet been closed by a probe
	consecutive int
	gen         uint64
	probe       uint64    // the probe in flight, 0 = none
	probeUntil  time.Time // when an unreported probe expires
}

// probeTTL is how long a probe may go unreported before another caller may
// probe: longer than any one logical call can take (an AI companion
// decision keeps its ticket across a retry and its tool rounds, about two
// and a half minutes at the most with the shipped timeouts), so a slow
// probe is never joined by a second. Callers hand every ticket back, a
// panic included (the companion's goroutines release it last); this only
// guards against a caller that never returns at all.
func probeTTL(cooldown time.Duration) time.Duration {
	if cooldown < 5*time.Minute {
		return 5 * time.Minute
	}
	return cooldown
}

// blocked is whether a caller would be turned away at now (open, or
// half-open with its probe in flight). It takes nothing.
func (b *breaker) blocked(now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.blockedLocked(now)
}

func (b *breaker) blockedLocked(now time.Time) bool {
	if now.Before(b.until) {
		return true
	}
	return b.tripped && b.probe != 0 && now.Before(b.probeUntil)
}

// admit lets a caller through: closed, with ticket 0; half-open with no
// probe out, as the probe (its ticket); open or probing, not at all.
func (b *breaker) admit(now time.Time, ttl time.Duration) (uint64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.blockedLocked(now) {
		return 0, false
	}
	if !b.tripped {
		return 0, true
	}
	b.gen++
	b.probe = b.gen
	b.probeUntil = now.Add(ttl)
	return b.probe, true
}

// record is one outcome. The probe's decides the half-open breaker; any
// other call's counts towards the run of failures as usual.
func (b *breaker) record(ticket uint64, failed bool, now time.Time, limit int, cooldown time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	isProbe := ticket != 0 && ticket == b.probe
	if isProbe {
		b.probe = 0
	}
	if !failed {
		b.consecutive = 0
		if isProbe {
			b.tripped = false
			b.until = time.Time{}
		}
		return
	}
	if isProbe {
		b.until = now.Add(cooldown)
		b.consecutive = 0
		return
	}
	b.consecutive++
	if limit < 1 {
		limit = 1
	}
	if b.consecutive >= limit {
		b.until = now.Add(cooldown)
		b.tripped = true
		b.consecutive = 0
	}
}

// release gives a probe back unjudged (its caller gave up on the call, or
// its own door refused it): the next caller may probe.
func (b *breaker) release(ticket uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if ticket != 0 && ticket == b.probe {
		b.probe = 0
	}
}

func (b *breaker) reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.until, b.tripped, b.consecutive, b.probe, b.probeUntil = time.Time{}, false, 0, 0, time.Time{}
}

// Ticket is a caller's leave to make one logical call on the server's key
// (retries included), from Allow. It is handed back with its outcome
// (Record) or unjudged (Release).
type Ticket struct {
	consumer uint64
	provider uint64
}

// Probing reports whether this call is a probe of a half-open breaker.
func (t Ticket) Probing() bool { return t.consumer != 0 || t.provider != 0 }

// ProviderFailure reports a failure that says the provider or the server's
// key is unwell, which is all that feeds the provider breaker every feature
// shares: no answer (a network error), a timeout, and HTTP 401, 403, 408,
// 429 and 5xx. A reply the provider did give (a 400 or 404 for one
// feature's model or schema, a refusal, content that does not parse) is the
// provider answering, and a call given up on (context.Canceled) is nobody's
// failure.
func ProviderFailure(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var se *StatusError
	if errors.As(err, &se) {
		return se.KeyOrProvider()
	}
	var ne net.Error
	return errors.As(err, &ne)
}

// StatusError is a provider's reply with a status other than 200.
type StatusError struct {
	Status int
	Detail string // a snippet of the provider's text; empty when it must not be kept
}

func (e *StatusError) Error() string {
	if e.Detail == `` {
		return `model API status ` + itoa(e.Status)
	}
	return `model API status ` + itoa(e.Status) + `: ` + e.Detail
}

// KeyOrProvider is a status that says the key or the provider is unwell,
// rather than that one request was wrong. A 403 is both: it can be the key
// barred altogether, or only this project barred from one model ("does not
// have access to model X"), which is one feature's model being refused, the
// same as a 404, and must not pause the other features.
func (e *StatusError) KeyOrProvider() bool {
	s := e.Status
	if s == http.StatusForbidden {
		return !ModelRefusal(e.Detail)
	}
	return s == http.StatusUnauthorized || s == http.StatusRequestTimeout ||
		s == http.StatusTooManyRequests || s >= 500
}

// ModelRefusal reports provider text that refuses a model rather than the
// key: the phrases the AI companion's model chooser (modelRefused) moves on
// from.
func ModelRefusal(detail string) bool {
	msg := strings.ToLower(detail)
	for _, p := range []string{`model_not_found`, `does not exist`, `do not have access`, `does not have access`, `unsupported model`} {
		if strings.Contains(msg, p) {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return `0`
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// consumerBreaker is consumer's own breaker in these books.
func (k *Books) consumerBreaker(consumer string) *breaker {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.byConsumer == nil {
		k.byConsumer = map[string]*breaker{}
	}
	b := k.byConsumer[consumer]
	if b == nil {
		b = &breaker{}
		k.byConsumer[consumer] = b
	}
	return b
}

func breakerSettings() (int, time.Duration) {
	s := Server()
	return s.BreakerErrors, time.Duration(s.BreakerSeconds) * time.Second
}

// Allow asks leave for one logical call by consumer on the server's key at
// now: both its own breaker and the provider's must let it through. Call it
// immediately before the request leaves, never merely to ask whether a call
// could be made (that is Blocked), and hand the ticket back with Record or
// Release.
func Allow(consumer string, now time.Time) (Ticket, bool) { return shared.Allow(consumer, now) }

// Allow on these books.
func (k *Books) Allow(consumer string, now time.Time) (Ticket, bool) {
	_, cooldown := breakerSettings()
	ttl := probeTTL(cooldown)
	c := k.consumerBreaker(consumer)
	ct, ok := c.admit(now, ttl)
	if !ok {
		return Ticket{}, false
	}
	pt, ok := k.b.admit(now, ttl)
	if !ok {
		c.release(ct)
		return Ticket{}, false
	}
	return Ticket{consumer: ct, provider: pt}, true
}

// Record is the outcome of the call t was given for: err nil, or its final
// failure after any retries. The consumer's breaker counts every failure;
// the provider's only ProviderFailure, and takes anything else as the
// provider having answered.
func Record(consumer string, t Ticket, err error, now time.Time) {
	shared.Record(consumer, t, err, now)
}

// Record on these books.
func (k *Books) Record(consumer string, t Ticket, err error, now time.Time) {
	limit, cooldown := breakerSettings()
	k.consumerBreaker(consumer).record(t.consumer, err != nil, now, limit, cooldown)
	k.b.record(t.provider, ProviderFailure(err), now, limit, cooldown)
}

// RecordConsumer counts one outcome against consumer's OWN breaker alone,
// never the provider's: for a check that belongs to one feature but is not
// a model call on the server's key (baubles' moderation of text a
// player's own key wrote; owner ruling 2026-09-29). err nil is a success.
// There is no ticket: the check never asked Allow for leave, so it is
// never the half-open probe. Once an opened breaker's cooldown is up,
// Blocked lets every such check through again (no probe is out), a
// success resets the run without clearing the trip, and BreakerErrors
// failures in a row open it for another cooldown. That is the right
// shape for a free check that cannot overload anyone; a caller that needs
// one-probe semantics uses Allow and Record.
func RecordConsumer(consumer string, err error, now time.Time) {
	shared.RecordConsumer(consumer, err, now)
}

// RecordConsumer on these books.
func (k *Books) RecordConsumer(consumer string, err error, now time.Time) {
	limit, cooldown := breakerSettings()
	k.consumerBreaker(consumer).record(0, err != nil, now, limit, cooldown)
}

// Release hands a ticket back unjudged: the caller gave up on the call, or
// its own door refused it, so it says nothing about the provider.
func Release(consumer string, t Ticket) { shared.Release(consumer, t) }

// Release on these books.
func (k *Books) Release(consumer string, t Ticket) {
	k.consumerBreaker(consumer).release(t.consumer)
	k.b.release(t.provider)
}

// Blocked reports whether consumer would be turned away at now, taking
// nothing: for deciding whether to try at all, and for status views.
func Blocked(consumer string, now time.Time) bool { return shared.Blocked(consumer, now) }

// Blocked on these books.
func (k *Books) Blocked(consumer string, now time.Time) bool {
	return k.b.blocked(now) || k.consumerBreaker(consumer).blocked(now)
}

// BreakerOpen reports whether the provider breaker every feature shares is
// turning callers away at now.
func BreakerOpen(now time.Time) bool { return shared.BreakerOpen(now) }

// BreakerOpen on these books.
func (k *Books) BreakerOpen(now time.Time) bool { return k.b.blocked(now) }

// BreakerUntil is when an open breaker closes for consumer (the later of
// its own and the provider's; zero when neither is open).
func BreakerUntil(consumer string) time.Time { return shared.BreakerUntil(consumer) }

// BreakerUntil on these books.
func (k *Books) BreakerUntil(consumer string) time.Time {
	c := k.consumerBreaker(consumer)
	c.mu.Lock()
	u := c.until
	c.mu.Unlock()
	k.b.mu.Lock()
	defer k.b.mu.Unlock()
	if k.b.until.After(u) {
		return k.b.until
	}
	return u
}

// BreakerFailures is the provider breaker's current run of failures.
func BreakerFailures() int { return shared.BreakerFailures() }

// BreakerFailures on these books.
func (k *Books) BreakerFailures() int {
	k.b.mu.Lock()
	defer k.b.mu.Unlock()
	return k.b.consecutive
}

// ConsumerFailures is consumer's own breaker's current run of failures.
func (k *Books) ConsumerFailures(consumer string) int {
	c := k.consumerBreaker(consumer)
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.consecutive
}

// ResetBreaker closes every breaker and forgets any run of failures (the
// aicompanion breaker admin command, and tests).
func ResetBreaker() { shared.ResetBreaker() }

// ResetBreaker on these books.
func (k *Books) ResetBreaker() {
	k.b.reset()
	k.mu.Lock()
	cs := make([]*breaker, 0, len(k.byConsumer))
	for _, c := range k.byConsumer {
		cs = append(cs, c)
	}
	k.mu.Unlock()
	for _, c := range cs {
		c.reset()
	}
}

// SetBreakerForTest sets the provider breaker's run of failures and when it
// closes (a future until also marks it tripped, so it half-opens then).
func SetBreakerForTest(consecutive int, until time.Time) {
	shared.SetBreakerForTest(consecutive, until)
}

// SetBreakerForTest on these books.
func (k *Books) SetBreakerForTest(consecutive int, until time.Time) {
	k.b.mu.Lock()
	defer k.b.mu.Unlock()
	k.b.consecutive = consecutive
	k.b.until = until
	k.b.tripped = !until.IsZero()
	k.b.probe = 0
}

// SetConsumerBreakerForTest sets consumer's own breaker's run of failures
// and when it closes.
func (k *Books) SetConsumerBreakerForTest(consumer string, consecutive int, until time.Time) {
	c := k.consumerBreaker(consumer)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.consecutive = consecutive
	c.until = until
	c.tripped = !until.IsZero()
	c.probe = 0
}
