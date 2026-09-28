package apiframework

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v3"
)

// ONE daily token budget for the server's key (APIFramework.DailyTokenBudget),
// whatever feature spends it. Each call reserves its worst case before it is
// made (Reserve), so several callers cannot all pass the check at once and
// overspend together, and settles to what it really used afterwards
// (Settle). Spending is also kept per consumer ("companion", "baubles") for
// the admin views.
//
// The day is the UTC date. Calls still in flight at the rollover keep their
// reservation into the new day (it starts at what is still held), or
// settling them afterwards would credit back tokens the new day never
// spent.
//
// A player's own key (the companion key relay) is not the server's and is
// never reserved here.
//
// The day's spending is living state, saved to <DataFiles>/apiframework/
// budget.yaml (SaveBudget, from the modules' save hooks and at shutdown) and
// read on first use, so a restart does not hand out a fresh budget.

// ErrOverBudget is a reservation refused because the day's budget is spent.
var ErrOverBudget = errors.New(`daily token budget spent`)

// Consumer names who spends: the budget reports spending per consumer.
const (
	ConsumerCompanion = `companion`
	ConsumerBaubles   = `baubles`
)

// Hold is one call's reservation, returned by Reserve and given back to
// Settle.
type Hold struct {
	Consumer string
	Tokens   int
	Day      string
}

type ledgerState struct {
	Day        string         `yaml:"day"`
	Tokens     int            `yaml:"tokens"`
	Calls      int            `yaml:"calls"`
	Failures   int            `yaml:"failures,omitempty"`
	ByConsumer map[string]int `yaml:"by_consumer,omitempty"`
	CallsBy    map[string]int `yaml:"calls_by,omitempty"`
}

type ledger struct {
	mu          sync.Mutex
	st          ledgerState
	outstanding int
	loaded      bool
	dirty       bool
	dir         string // "" = <DataFiles>/apiframework
	now         func() time.Time
}

var budget = &ledger{now: time.Now}

func today(t time.Time) string { return t.UTC().Format(`2006-01-02`) }

// rollLocked starts a new day at the UTC boundary. Caller holds mu.
func (l *ledger) rollLocked() {
	if d := today(l.now()); d != l.st.Day {
		l.st = ledgerState{Day: d, Tokens: l.outstanding}
		l.dirty = true
	}
	if l.st.ByConsumer == nil {
		l.st.ByConsumer = map[string]int{}
	}
	if l.st.CallsBy == nil {
		l.st.CallsBy = map[string]int{}
	}
}

func (l *ledger) budgetDir() string {
	if l.dir != `` {
		return l.dir
	}
	return util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `apiframework`)
}

func (l *ledger) path() string { return util.FilePath(l.budgetDir(), `/`, `budget.yaml`) }

// loadLocked reads today's saved spending once. A stale day is simply a new
// day; a corrupt file is quarantined and the day starts from nothing.
// Caller holds mu.
func (l *ledger) loadLocked() {
	if l.loaded {
		return
	}
	l.loaded = true
	raw, err := util.ReadLivingState(l.path())
	if err != nil {
		if errors.Is(err, util.ErrStateCorrupt) {
			l.quarantine(err)
		}
		return
	}
	var st ledgerState
	if err := yaml.Unmarshal(raw, &st); err != nil {
		l.quarantine(err)
		return
	}
	if st.Day == today(l.now()) {
		l.st = st
	}
}

// quarantine moves a corrupt budget file aside and logs it; the day starts
// from nothing (the living-state contract).
func (l *ledger) quarantine(cause error) {
	moved, err := util.QuarantineCorrupt(l.path())
	mudlog.Error(`apiframework`, `action`, `loadBudget`, `error`, cause, `quarantinedTo`, moved, `quarantineError`, err)
}

// Reserve holds tokens against today's budget for consumer, or refuses with
// ErrOverBudget. limit is the day's budget (Server().DailyTokenBudget).
func Reserve(consumer string, tokens int) (Hold, error) {
	return shared.Reserve(consumer, tokens)
}

// Reserve on these books.
func (k *Books) Reserve(consumer string, tokens int) (Hold, error) {
	return k.l.reserve(consumer, tokens, Server().DailyTokenBudget)
}

func (l *ledger) reserve(consumer string, tokens int, limit int) (Hold, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.loadLocked()
	l.rollLocked()
	if tokens < 0 {
		tokens = 0
	}
	if limit > 0 && l.st.Tokens+tokens > limit {
		return Hold{}, ErrOverBudget
	}
	l.st.Tokens += tokens
	l.st.ByConsumer[consumer] += tokens
	l.st.Calls++
	l.st.CallsBy[consumer]++
	l.outstanding += tokens
	l.dirty = true
	return Hold{Consumer: consumer, Tokens: tokens, Day: l.st.Day}, nil
}

// Settle replaces a reservation with what the call really used (use
// Charged to work that out). failed counts a failed call in the day's
// figures. A hold from an earlier day settles against today the same way,
// since today started at what was still held.
func Settle(h Hold, used int, failed bool) {
	shared.Settle(h, used, failed)
}

// Settle on these books.
func (k *Books) Settle(h Hold, used int, failed bool) {
	k.l.settle(h, used, failed)
}

func (l *ledger) settle(h Hold, used int, failed bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.loadLocked()
	l.rollLocked()
	l.outstanding -= h.Tokens
	if l.outstanding < 0 {
		l.outstanding = 0
	}
	diff := used - h.Tokens
	l.st.Tokens += diff
	if l.st.Tokens < 0 {
		l.st.Tokens = 0
	}
	// What a consumer is shown is what it spent today; a hold from an
	// earlier day gives nothing back to today's share.
	if h.Day != l.st.Day && diff < 0 {
		diff = 0
	}
	l.st.ByConsumer[h.Consumer] += diff
	if l.st.ByConsumer[h.Consumer] < 0 {
		l.st.ByConsumer[h.Consumer] = 0
	}
	if failed {
		l.st.Failures++
	}
	l.dirty = true
}

// HasRoom reports whether the day's budget is not yet spent.
func HasRoom() bool {
	return shared.HasRoom()
}

// HasRoom on these books.
func (k *Books) HasRoom() bool {
	limit := Server().DailyTokenBudget
	k.l.mu.Lock()
	defer k.l.mu.Unlock()
	k.l.loadLocked()
	k.l.rollLocked()
	return limit <= 0 || k.l.st.Tokens < limit
}

// Usage is the day's spending, for the admin views.
type Usage struct {
	Day         string
	Tokens      int // spent and still held, all consumers
	Outstanding int // held by calls still in flight
	Calls       int
	Failures    int
	Limit       int
	ByConsumer  []ConsumerUsage // most tokens first
}

// ConsumerUsage is one consumer's share of the day.
type ConsumerUsage struct {
	Consumer string
	Tokens   int
	Calls    int
}

// Today is the day's spending.
func Today() Usage {
	return shared.Today()
}

// Today on these books.
func (k *Books) Today() Usage {
	limit := Server().DailyTokenBudget
	k.l.mu.Lock()
	defer k.l.mu.Unlock()
	k.l.loadLocked()
	k.l.rollLocked()
	u := Usage{Day: k.l.st.Day, Tokens: k.l.st.Tokens, Outstanding: k.l.outstanding,
		Calls: k.l.st.Calls, Failures: k.l.st.Failures, Limit: limit}
	for c, n := range k.l.st.ByConsumer {
		u.ByConsumer = append(u.ByConsumer, ConsumerUsage{Consumer: c, Tokens: n, Calls: k.l.st.CallsBy[c]})
	}
	sort.Slice(u.ByConsumer, func(a, b int) bool {
		if u.ByConsumer[a].Tokens != u.ByConsumer[b].Tokens {
			return u.ByConsumer[a].Tokens > u.ByConsumer[b].Tokens
		}
		return u.ByConsumer[a].Consumer < u.ByConsumer[b].Consumer
	})
	return u
}

// SeedTokens adds tokens a consumer spent today before the framework kept
// the books (the AI companion's own saved day, on the first boot after the
// move), so moving does not hand out a second budget. It only ever applies
// once per day and only to a fresh day.
func SeedTokens(consumer string, day string, tokens int) {
	shared.SeedTokens(consumer, day, tokens)
}

// SeedTokens on these books.
func (k *Books) SeedTokens(consumer string, day string, tokens int) {
	k.l.mu.Lock()
	defer k.l.mu.Unlock()
	k.l.loadLocked()
	k.l.rollLocked()
	if tokens <= 0 || day != k.l.st.Day || k.l.st.Tokens != k.l.outstanding || k.l.st.ByConsumer[consumer] != 0 {
		return
	}
	k.l.st.Tokens += tokens
	k.l.st.ByConsumer[consumer] += tokens
	k.l.dirty = true
}

// SaveBudget writes the day's spending when it changed. Safe to call often.
func SaveBudget() {
	budget.mu.Lock()
	if !budget.loaded || !budget.dirty {
		budget.mu.Unlock()
		return
	}
	st := budget.st
	byC := make(map[string]int, len(st.ByConsumer))
	for k, v := range st.ByConsumer {
		byC[k] = v
	}
	callsBy := make(map[string]int, len(st.CallsBy))
	for k, v := range st.CallsBy {
		callsBy[k] = v
	}
	st.ByConsumer, st.CallsBy = byC, callsBy
	path := budget.path()
	budget.dirty = false
	budget.mu.Unlock()

	raw, err := yaml.Marshal(&st)
	if err == nil {
		err = os.MkdirAll(filepath.Dir(path), 0755)
	}
	if err == nil {
		err = util.Save(path, raw)
	}
	if err != nil {
		mudlog.Error(`apiframework`, `action`, `saveBudget`, `error`, err)
		budget.mu.Lock()
		budget.dirty = true
		budget.mu.Unlock()
	}
}

// ResetBudgetForTest starts a fresh, unsaved day in dir ("" keeps nothing
// on disk: the ledger is marked loaded and never written unless SaveBudget
// is called).
func ResetBudgetForTest(dir string) {
	budget.mu.Lock()
	defer budget.mu.Unlock()
	budget.st = ledgerState{}
	budget.outstanding = 0
	budget.loaded = dir == ``
	budget.dirty = false
	budget.dir = dir
	budget.now = time.Now
}

// SetSpentForTest sets today's total spending and in-flight holds.
func SetSpentForTest(tokens int, outstanding int) {
	shared.SetSpentForTest(tokens, outstanding)
}

// SetSpentForTest on these books.
func (k *Books) SetSpentForTest(tokens int, outstanding int) {
	k.l.mu.Lock()
	defer k.l.mu.Unlock()
	k.l.loadLocked()
	k.l.rollLocked()
	k.l.st.Tokens = tokens
	k.l.outstanding = outstanding
}

// SetClockForTest replaces the ledger's clock.
func SetClockForTest(now func() time.Time) {
	budget.mu.Lock()
	defer budget.mu.Unlock()
	budget.now = now
}
