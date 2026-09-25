package aicompanion

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Model tiers (phase 7). Different moments need different models:
//
//   - fast: the moment needs an answer now more than a fine one: a combat
//     plan, noticing something, a quiet moment, a follow-up after a look,
//     a trip update, a finished goal. A small, low-latency model.
//   - main: conversation and relationship: someone speaks, asks, emotes,
//     gives, hurts or heals; greetings and goodbyes. The model that makes
//     the companion feel like a person.
//   - deep: the private reflection after a session. Nobody is waiting, so
//     the strongest model is worth its time.
//
// Every tier falls back to Model when its own model is not configured.

const (
	tierFast = `fast`
	tierMain = `main`
	tierDeep = `deep`
)

// tierOfKind is the tier each stimulus needs on its own.
var tierOfKind = map[string]string{
	`heard`:         tierMain,
	`asked`:         tierMain,
	`emote`:         tierMain,
	`gift`:          tierMain,
	`attacked`:      tierMain,
	`healed`:        tierMain,
	`first_meeting`: tierMain,
	`session_start`: tierMain,
	// A goodbye has about twenty seconds of logout meditation to arrive in,
	// so it goes to the quick model rather than the considered one.
	`farewell`:    tierFast,
	`recovered`:   tierMain,
	`arrived`:     tierMain,
	`fight_over`:  tierMain,
	`romance`:     tierMain,
	`romance_yes`: tierMain,
	`romance_no`:  tierMain,
	`night`:       tierMain,
	`party`:       tierMain,
	`fight`:       tierFast,
	`noticed`:     tierFast,
	// A quiet moment is where she acts on her own purposes, so it gets the
	// full picture: goals, pack, prices and map. Noticing something does
	// not.
	`idle`:        tierMain,
	`quiet`:       tierFast,
	`looked`:      tierFast,
	`trip`:        tierFast,
	`goal_done`:   tierFast,
	`remembering`: tierMain,
	`witnessed`:   tierMain,
	`errand_ask`:  tierMain,
	`ailing`:      tierFast,
	`trouble`:     tierFast,
	`grew`:        tierFast,
}

// tierFor picks the tier for a batch of stimuli. A fight always goes fast,
// because the fight will not wait; otherwise anything conversational makes
// the whole batch main.
func tierFor(stims []stimulus) string {
	tier := tierFast
	for _, s := range stims {
		if s.Kind == `fight` {
			return tierFast
		}
		if t, ok := tierOfKind[s.Kind]; !ok || t == tierMain {
			tier = tierMain
		}
	}
	return tier
}

// tierSettings is what a call uses for its tier.
type tierSettings struct {
	Model     string
	MaxTokens int
	Effort    string // reasoning effort, sent only when set
	Timeout   time.Duration
}

// settingsFor resolves a tier to a model and limits, falling back to the
// main model when a tier has none of its own.
func (c Config) settingsFor(tier string) tierSettings {
	s := tierSettings{
		Model:     c.Model,
		MaxTokens: c.MaxCompletionTokens,
		Effort:    c.MainReasoningEffort,
		Timeout:   time.Duration(c.RequestTimeoutSeconds) * time.Second,
	}
	switch tier {
	case tierFast:
		s.Effort = c.FastReasoningEffort
		if c.FastModel != `` {
			s.Model = c.FastModel
		}
		if c.FastMaxCompletionTokens > 0 {
			s.MaxTokens = c.FastMaxCompletionTokens
		}
		if c.FastTimeoutSeconds > 0 {
			s.Timeout = time.Duration(c.FastTimeoutSeconds) * time.Second
		}
	case tierDeep:
		s.Effort = c.DeepReasoningEffort
		if c.DeepModel != `` {
			s.Model = c.DeepModel
		}
		if c.DeepMaxCompletionTokens > 0 {
			s.MaxTokens = c.DeepMaxCompletionTokens
		}
		s.Timeout = 2 * s.Timeout
	}
	return s
}

// tierStats is running metrics per tier (F19.4).
type tierStats struct {
	Calls      int
	Errors     int
	Tokens     int
	LatencySum time.Duration
}

func (m *AICompanionModule) recordCall(tier string, res modelResult) {
	if m.stats == nil {
		m.stats = map[string]*tierStats{}
	}
	st, ok := m.stats[tier]
	if !ok {
		st = &tierStats{}
		m.stats[tier] = st
	}
	st.Calls++
	st.Tokens += res.Tokens
	st.LatencySum += res.Latency
	if res.Err != nil && !res.Canceled {
		st.Errors++
	}
}

func (m *AICompanionModule) statsLines() []string {
	tiers := make([]string, 0, len(m.stats))
	for t := range m.stats {
		tiers = append(tiers, t)
	}
	sort.Strings(tiers)
	var out []string
	for _, t := range tiers {
		st := m.stats[t]
		avg := time.Duration(0)
		if st.Calls > 0 {
			avg = st.LatencySum / time.Duration(st.Calls)
		}
		out = append(out, fmt.Sprintf(`  %s: calls=%d errors=%d tokens=%d avgLatency=%s`, t, st.Calls, st.Errors, st.Tokens, avg.Round(time.Millisecond)))
	}
	return out
}

// Circuit breaker. After BreakerErrors consecutive failures the module
// stops calling the model for BreakerSeconds and runs on fallback lines;
// the first call after that is a probe, and one success closes it again.

func (m *AICompanionModule) breakerOpen(now time.Time) bool {
	return now.Before(m.breakerUntil)
}

func (m *AICompanionModule) breakerResult(err error, now time.Time) {
	if errors.Is(err, errNoConsent) {
		// The door refused a request that never left the server. That is a
		// bug in the caller, not the provider failing, and must not pause
		// every other companion.
		return
	}
	if err == nil {
		m.consecutiveErrors = 0
		return
	}
	m.consecutiveErrors++
	if m.consecutiveErrors >= m.cfg.BreakerErrors {
		m.breakerUntil = now.Add(time.Duration(m.cfg.BreakerSeconds) * time.Second)
		m.consecutiveErrors = 0
	}
}

// ownerBudgetLeft reports whether one companion has any daily tokens left
// at all. Admission of a particular call goes through tryReserveTokens,
// which weighs that call's worst case.
func (m *AICompanionModule) ownerBudgetLeft(ownerId int) bool {
	if m.cfg.DailyTokensPerCompanion <= 0 {
		return true
	}
	return m.ownerTokens[ownerId] < m.cfg.DailyTokensPerCompanion
}

func (m *AICompanionModule) chargeOwner(ownerId int, tokens int) {
	if m.ownerTokens == nil {
		m.ownerTokens = map[int]int{}
	}
	m.ownerTokens[ownerId] += tokens
}

// chargeStranger is chargeOwner for a passer-by's own daily allowance. Like
// chargeOwner it takes a negative amount, which is how a settlement gives
// back what a reservation held and the call did not use.
func (m *AICompanionModule) chargeStranger(userId int, tokens int) {
	if userId <= 0 {
		return
	}
	if m.strangerTokens == nil {
		m.strangerTokens = map[int]int{}
	}
	m.strangerTokens[userId] += tokens
}

// strangerFits reports whether a passer-by's call of this size fits both
// what that passer-by may spend in a day (StrangerDailyTokens) and what all
// passers-by together may spend of this one owner's companion
// (StrangerTokensPerOwner): many strangers, each within their own
// allowance, could otherwise spend one owner's key without end. Zero is no
// cap for either.
func (m *AICompanionModule) strangerFits(ownerId int, askerId int, tokens int) bool {
	if m.cfg.StrangerDailyTokens > 0 && m.strangerTokens[askerId]+tokens > m.cfg.StrangerDailyTokens {
		return false
	}
	return m.cfg.StrangerTokensPerOwner <= 0 || m.strangersFor[ownerId]+tokens <= m.cfg.StrangerTokensPerOwner
}

// chargeStrangerFor charges a passer-by's call to both of the counts
// strangerFits weighs, and takes a negative amount the same way (a
// settlement), never leaving either below nothing.
func (m *AICompanionModule) chargeStrangerFor(ownerId int, askerId int, tokens int) {
	m.chargeStranger(askerId, tokens)
	if m.strangerTokens[askerId] < 0 {
		m.strangerTokens[askerId] = 0
	}
	if ownerId <= 0 || askerId <= 0 {
		return
	}
	if m.strangersFor == nil {
		m.strangersFor = map[int]int{}
	}
	m.strangersFor[ownerId] += tokens
	if m.strangersFor[ownerId] < 0 {
		m.strangersFor[ownerId] = 0
	}
}

// traceEntry is one decision kept for the admin trace view.
type traceEntry struct {
	Unix     int64
	Tier     string
	Model    string
	Triggers string
	Intent   string
	Lines    int
	Action   string
	Tokens   int
	Latency  time.Duration
	Tools    int
	Err      string
}

func (c *controller) addTrace(t traceEntry) {
	c.traces = append(c.traces, t)
	if len(c.traces) > 12 {
		c.traces = append([]traceEntry(nil), c.traces[len(c.traces)-12:]...)
	}
}

func (t traceEntry) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, `%s %s/%s [%s]`, time.Unix(t.Unix, 0).Format(`15:04:05`), t.Tier, t.Model, t.Triggers)
	if t.Err != `` {
		fmt.Fprintf(&b, ` ERROR %s`, t.Err)
		return b.String()
	}
	fmt.Fprintf(&b, ` lines=%d action=%q questions=%d tokens=%d %s intent=%q`, t.Lines, t.Action, t.Tools, t.Tokens, t.Latency.Round(time.Millisecond), t.Intent)
	return b.String()
}

// Zero-configuration model choice. With no model configured, each tier
// picks the first model from its preference list that the API key can use,
// learned from the API's model list at start-up. If a model is refused at
// runtime (retired, or not available to the key), the tier moves on to the
// next one by itself. Setting Model, FastModel or DeepModel overrides this.
//
// The lists run newest first and end in long-lived fallbacks, so the module
// keeps working as OpenAI's line-up changes.
var modelPreferences = map[string][]string{
	tierFast: {`gpt-5.4-nano`, `gpt-5-nano`, `gpt-4.1-nano`, `gpt-4o-mini`},
	tierMain: {`gpt-5.4-mini`, `gpt-5-mini`, `gpt-4.1-mini`, `gpt-4o-mini`},
	tierDeep: {`gpt-5.5`, `gpt-5.4`, `gpt-5`, `gpt-4.1`, `gpt-4o`},
}

// autoEffort is the reasoning effort each tier asks for when none is
// configured: none where speed matters (and where function tools require
// it on current models), medium for the unhurried reflection.
var autoEffort = map[string]string{
	tierFast: `none`,
	tierMain: `none`,
	tierDeep: `medium`,
}

// effortFor adapts a wanted reasoning effort to what a model family
// accepts, or returns "" when the model takes no reasoning effort at all.
//
//   - gpt-5.1 and later (gpt-5.4, gpt-5.5, ...): none, low, medium, high,
//     xhigh. Function tools on Chat Completions need none, so a call that
//     offers tools always sends none.
//   - the original gpt-5 family: minimal, low, medium, high (none becomes
//     minimal, xhigh becomes high).
//   - o-series: low, medium, high.
//   - everything else (gpt-4.x, gpt-4o): not sent.
func effortFor(model string, want string, tools bool) string {
	m := strings.ToLower(model)
	switch {
	case strings.HasPrefix(m, `gpt-5.`):
		if tools || want == `` {
			return `none`
		}
		if want == `minimal` {
			return `none`
		}
		return want
	case m == `gpt-5` || strings.HasPrefix(m, `gpt-5-`):
		switch want {
		case ``, `none`:
			return `minimal`
		case `xhigh`:
			return `high`
		}
		return want
	case strings.HasPrefix(m, `o1`) || strings.HasPrefix(m, `o3`) || strings.HasPrefix(m, `o4`):
		switch want {
		case ``, `none`, `minimal`:
			return `low`
		case `xhigh`:
			return `high`
		}
		return want
	}
	return ``
}

// modelChooser holds the resolved model per tier and what has been refused.
type modelChooser struct {
	available map[string]bool // model ids the key can use, from /models; nil = unknown
	refused   map[string]bool // models the API refused at runtime
	chosen    map[string]string
}

// pick returns the first preferred model for a tier that is available (when
// known) and not refused. With nothing left it returns the last preference.
func (mc *modelChooser) pick(tier string) string {
	if mc.chosen == nil {
		mc.chosen = map[string]string{}
	}
	if m, ok := mc.chosen[tier]; ok && !mc.refused[m] {
		return m
	}
	prefs := modelPreferences[tier]
	for _, m := range prefs {
		if mc.refused[m] {
			continue
		}
		if mc.available != nil && !mc.available[m] {
			continue
		}
		mc.chosen[tier] = m
		return m
	}
	// Nothing known to work: try the preferences in order regardless of the
	// model list, skipping only what has been refused.
	for _, m := range prefs {
		if !mc.refused[m] {
			mc.chosen[tier] = m
			return m
		}
	}
	// Every model this tier knows has been refused. Saying so stops the
	// module hammering a model the API will not serve; an operator can set
	// one explicitly, and `aicompanion models` shows the empty tier.
	return ``
}

// refuse records that the API would not serve a model, so every tier using
// it moves on.
func (mc *modelChooser) refuse(model string) {
	if mc.refused == nil {
		mc.refused = map[string]bool{}
	}
	mc.refused[model] = true
	for tier, m := range mc.chosen {
		if m == model {
			delete(mc.chosen, tier)
		}
	}
}

// modelRefused reports whether an API error means "this model is not
// available", as opposed to a transient or request problem.
func modelRefused(r modelResult) bool {
	if r.Err == nil {
		return false
	}
	msg := strings.ToLower(r.Err.Error())
	if r.Status == http.StatusNotFound {
		return true
	}
	if r.Status == http.StatusBadRequest || r.Status == http.StatusForbidden {
		return strings.Contains(msg, `model_not_found`) ||
			strings.Contains(msg, `does not exist`) ||
			strings.Contains(msg, `do not have access`) ||
			strings.Contains(msg, `does not have access`) ||
			strings.Contains(msg, `unsupported model`)
	}
	return false
}

// settingsFor resolves a tier to the model, effort and limits a call uses:
// the configured model if one is set, otherwise the automatic choice.
// tools says whether the call offers function tools.
func (m *AICompanionModule) settingsFor(tier string, tools bool) tierSettings {
	s := m.cfg.settingsFor(tier)
	configured := m.cfg.Model
	switch tier {
	case tierFast:
		if m.cfg.FastModel != `` {
			configured = m.cfg.FastModel
		}
	case tierDeep:
		if m.cfg.DeepModel != `` {
			configured = m.cfg.DeepModel
		}
	}
	if configured != `` {
		s.Model = configured
	} else {
		s.Model = m.models.pick(tier)
	}
	want := s.Effort
	if want == `` {
		want = autoEffort[tier]
	}
	s.Effort = effortFor(s.Model, want, tools)
	return s
}

// probeModels asks the API which models the key can use, off the game loop,
// and stores the answer under the mud lock. A failure leaves the list
// unknown and the tiers simply try their preferences in order.
func (m *AICompanionModule) probeModels() {
	baseURL, key := m.cfg.BaseURL, m.apiKey()
	if key == `` {
		return
	}
	go func() {
		ids := listModels(baseURL, key)
		if ids == nil {
			return
		}
		util.LockMud()
		defer util.UnlockMud()
		m.models.available = ids
		m.models.chosen = nil
		mudlog.Info(`aicompanion`, `action`, `probeModels`, `models`, len(ids),
			`fast`, m.models.pick(tierFast), `main`, m.models.pick(tierMain), `deep`, m.models.pick(tierDeep))
	}()
}

// Budgets are held before a call, not after it: the worst case is charged
// up front so several companions cannot all pass the check at once and
// overspend together, and the reservation is settled once the real usage
// is known.

// estimateTokens is a rough count of a request's prompt, about four
// characters to the token.
func estimateTokens(msgs []chatMessage) int {
	n := 0
	for _, msg := range msgs {
		// Bytes, not runes: a multi-byte character is more tokens, not
		// fewer, so counting bytes errs towards over-reserving.
		n += len(msg.Content)/4 + 8
		for _, tc := range msg.ToolCalls {
			n += (len(tc.Function.Name) + len(tc.Function.Arguments)) / 4
		}
	}
	return n
}

// requestOverhead is the schema and the tool definitions, which are sent
// with every call and are not in the messages.
func requestOverhead(call modelCall) int {
	n := 0
	if call.Schema != nil {
		n += 400 // the decision schema, in round numbers
	}
	for _, t := range call.Tools {
		n += (len(t.Function.Name) + len(t.Function.Description) + 200) / 4
	}
	return n
}

// worstCaseTokens is everything one decision could spend: the prompt and a
// full completion for each round the model may ask the game something, and
// again if the call is retried.
func worstCaseTokens(prompt int, maxTokens int, toolRounds int, retry bool) int {
	// Every round of questions sends the whole conversation again, with the
	// answers so far added to it, so the prompt grows as it goes.
	total := 0
	grown := prompt
	for i := 0; i <= toolRounds; i++ {
		total += grown + maxTokens
		// The model's questions (at most a completion), and the game's
		// answers: as many as a reply may ask, each as long as an answer
		// may be, at a token a rune (estimateTokens' worst case for bytes)
		// plus a message's framing.
		grown += maxTokens + maxToolCallsPerReply*(maxToolAnswerRunes+8)
	}
	if retry {
		total *= 2
	}
	return total
}

// tryReserveTokens admits a call only if its worst case still fits both
// budgets, and holds the tokens in the same step. Checking and charging
// apart is what let two calls slip past a nearly spent budget together.
func (m *AICompanionModule) tryReserveTokens(ownerId int, tokens int) bool {
	return m.tryReserveFor(ownerId, 0, tokens)
}

// tryReserveFor is tryReserveTokens with the payer named: a call a
// passer-by prompted (askerId above 0) is held against their own
// StrangerDailyTokens instead of the owner's companion allowance, so a
// stranger cannot spend somebody else's companion into silence. The
// server's budget holds either way. Check and hold are still one step.
func (m *AICompanionModule) tryReserveFor(ownerId int, askerId int, tokens int) bool {
	m.rollDay()
	if m.cfg.DailyTokenBudget > 0 && m.tokensToday+tokens > m.cfg.DailyTokenBudget {
		return false
	}
	if askerId > 0 {
		if !m.strangerFits(ownerId, askerId, tokens) {
			return false
		}
	} else if m.cfg.DailyTokensPerCompanion > 0 && m.ownerTokens[ownerId]+tokens > m.cfg.DailyTokensPerCompanion {
		return false
	}
	m.tokensToday += tokens
	m.outstanding += tokens
	if askerId > 0 {
		m.chargeStrangerFor(ownerId, askerId, tokens)
	} else {
		m.chargeOwner(ownerId, tokens)
	}
	return true
}

// settleTokens replaces a reservation with what the call really used.
func (m *AICompanionModule) settleTokens(ownerId int, reserved int, used int) {
	m.settleFor(ownerId, 0, reserved, used)
}

// settleFor settles a reservation made by tryReserveFor, against the same
// payer it was held against.
func (m *AICompanionModule) settleFor(ownerId int, askerId int, reserved int, used int) {
	m.rollDay()
	m.outstanding -= reserved
	if m.outstanding < 0 {
		m.outstanding = 0
	}
	diff := used - reserved
	m.tokensToday += diff
	if m.tokensToday < 0 {
		m.tokensToday = 0
	}
	switch {
	case askerId > 0:
		m.chargeStrangerFor(ownerId, askerId, diff)
	case ownerId > 0:
		m.chargeOwner(ownerId, diff)
		if m.ownerTokens[ownerId] < 0 {
			m.ownerTokens[ownerId] = 0
		}
	}
}

// budgetState is the day's spending, kept on disk so a restart does not
// hand every companion a fresh allowance. The per-companion and per-asker
// counters go with it, because they are what the caps are made of.
type budgetState struct {
	Day       string      `yaml:"day"`
	Tokens    int         `yaml:"tokens"`
	Calls     int         `yaml:"calls"`
	Owners    map[int]int `yaml:"owners,omitempty"`
	Strangers map[int]int `yaml:"strangers,omitempty"`
	// StrangersFor is what passers-by together spent of each owner's
	// companion (StrangerTokensPerOwner), by owner.
	StrangersFor map[int]int `yaml:"strangers_for,omitempty"`
	// Notices is each owner's "you notice" moments today (NoticeCallsPerDay).
	Notices map[int]int `yaml:"notices,omitempty"`
}

const budgetStateId = `budget-state`

func (m *AICompanionModule) loadBudget() {
	var st budgetState
	if err := m.plug.ReadIntoStruct(budgetStateId, &st); err != nil {
		return // nothing recorded yet, or unreadable: start the day fresh
	}
	if st.Day != time.Now().UTC().Format(`2006-01-02`) {
		return // a stale day is simply a new day
	}
	m.budgetDay = st.Day
	m.tokensToday = st.Tokens
	m.callsToday = st.Calls
	m.ownerTokens = st.Owners
	m.strangerTokens = st.Strangers
	m.strangersFor = st.StrangersFor
	m.noticesToday = st.Notices
	if m.ownerTokens == nil {
		m.ownerTokens = map[int]int{}
	}
	if m.strangerTokens == nil {
		m.strangerTokens = map[int]int{}
	}
	if m.noticesToday == nil {
		m.noticesToday = map[int]int{}
	}
	if m.strangersFor == nil {
		m.strangersFor = map[int]int{}
	}
}

func (m *AICompanionModule) saveBudget() {
	if !m.cfg.Enabled {
		return
	}
	st := budgetState{Day: m.budgetDay, Tokens: m.tokensToday, Calls: m.callsToday,
		Owners: m.ownerTokens, Strangers: m.strangerTokens, StrangersFor: m.strangersFor, Notices: m.noticesToday}
	if err := m.plug.WriteStruct(budgetStateId, &st); err != nil {
		mudlog.Error(`aicompanion`, `action`, `saveBudget`, `error`, err)
	}
}
