package aicompanion

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/apiframework"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// R5, R6: who pays, one to one with the old rules. A passer-by pays from
// their own allowance and from what passers-by together may spend of this
// owner's companion (only when there is an owner), never from the owner's
// allowance; everything else is the owner's.
func TestAllowanceChargesMapOneToOne(t *testing.T) {
	m := &AICompanionModule{cfg: Config{DailyTokensPerCompanion: 300, StrangerDailyTokens: 50, StrangerTokensPerOwner: 100}}
	owner := func(id int) apiframework.Charge {
		return apiframework.Charge{Dim: apiframework.DimCompanionOwner, UserId: id, Limit: 300}
	}
	stranger := apiframework.Charge{Dim: apiframework.DimCompanionStranger, UserId: 2, Limit: 50}
	perOwner := apiframework.Charge{Dim: apiframework.DimCompanionStrangersFor, UserId: 5, Limit: 100}
	for _, tc := range []struct {
		name         string
		owner, asker int
		want         []apiframework.Charge
	}{
		{`the owner's own call`, 5, 0, []apiframework.Charge{owner(5)}},
		{`a passer-by`, 5, 2, []apiframework.Charge{stranger, perOwner}},
		{`a passer-by, no owner`, 0, 2, []apiframework.Charge{stranger}},
		{`no owner, no asker`, 0, 0, []apiframework.Charge{owner(0)}},
	} {
		if got := m.allowanceCharges(tc.owner, tc.asker); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

// R16 (owner ruling 12): an owner-less server-key call is charged to key 0
// and, like any other, settled back to what it used.
func TestAnOwnerlessHoldIsRefunded(t *testing.T) {
	freshServer(t, 5000, 5, 60)
	m := &AICompanionModule{cfg: Config{DailyTokensPerCompanion: 1000}}
	h, ok := m.reserveRoute(route{kind: routeServer}, 0, 0, 900)
	if !ok || ownerSpent(m, 0) != 900 {
		t.Fatalf("fixture: held against key 0: %v %d", ok, ownerSpent(m, 0))
	}
	m.settleRoute(h, 100)
	if ownerSpent(m, 0) != 100 {
		t.Fatalf("key 0 is refunded what it did not use: %d", ownerSpent(m, 0))
	}
}

// R43: a refused reservation keeps the ledger's reason, so the refusal log
// can say which counter said no.
func TestARefusedRouteKeepsTheLedgersReason(t *testing.T) {
	freshServer(t, 5000, 5, 60)
	m := &AICompanionModule{cfg: Config{StrangerDailyTokens: 100}}
	h, ok := m.reserveRoute(route{kind: routeServer}, 1, 2, 500)
	if ok || apiframework.RefusedBy(h.refusal) != apiframework.DimCompanionStranger {
		t.Fatalf("refused by the passer-by's allowance: %v %v", ok, h.refusal)
	}
	h, ok = m.reserveRoute(route{kind: routeServer}, 1, 0, 6000)
	if ok || apiframework.RefusedBy(h.refusal) != apiframework.RefusedGlobal {
		t.Fatalf("refused by the day's budget: %v %v", ok, h.refusal)
	}
	if h, ok := m.reserveRoute(route{kind: routeNone}, 1, 0, 10); ok || h.refusal != nil {
		t.Fatal("no route is no refusal of the ledger's")
	}
}

// R32, R33: the companion's own counts (calls, errors, "you notice"
// moments) roll when the ledger's day turns, on the ledger's clock.
func TestCountersRollOnTheLedgersClock(t *testing.T) {
	m := &AICompanionModule{cfg: Config{NoticeCallsPerDay: 1}}
	m.rollCounters()
	m.callsToday, m.errorsToday = 3, 2
	m.noticesToday[5] = 1
	m.rollCounters()
	if m.callsToday != 3 || m.noticesToday[5] != 1 {
		t.Fatal("the same day keeps its counts")
	}
	tomorrow := time.Now().UTC().Add(24 * time.Hour)
	m.fw().SetClockForTest(func() time.Time { return tomorrow })
	m.rollCounters()
	if m.callsToday != 0 || m.errorsToday != 0 || m.noticesToday[5] != 0 || m.countersDay != tomorrow.Format(`2006-01-02`) {
		t.Fatalf("a new ledger day starts them afresh: calls=%d errors=%d notices=%d day=%s",
			m.callsToday, m.errorsToday, m.noticesToday[5], m.countersDay)
	}
}

// Correction 9, owner ruling 12: the companion's own file is the backup of
// its allowances. A quarantined budget.yaml loses the ledger's counts and
// its seed marks together, so the next boot seeds them again from that
// backup: a corrupt file hands nobody a fresh allowance.
func TestAQuarantinedLedgerReseedsFromTheCompanionsBackup(t *testing.T) {
	dir := t.TempDir()
	apiframework.ResetBudgetForTest(dir)
	t.Cleanup(func() { apiframework.ResetBudgetForTest(``) })
	cfg := Config{DailyTokensPerCompanion: 100000, StrangerDailyTokens: 100000, StrangerTokensPerOwner: 100000}
	m := &AICompanionModule{cfg: cfg}
	m.books.Store(apiframework.Shared())            // the real ledger, on disk in dir
	m.restoreBudget(budgetState{Day: m.fw().Day()}) // this boot's seed: it marks all three dimensions
	server := route{kind: routeServer}
	h1, ok1 := m.reserveRoute(server, 5, 0, 900)
	h2, ok2 := m.reserveRoute(server, 5, 2, 900)
	if !ok1 || !ok2 {
		t.Fatal("fixture: both holds fit")
	}
	m.settleRoute(h1, 400)
	m.settleRoute(h2, 300)
	backup := m.budgetStateToSave() // what saveBudget writes to the companion's own file
	if backup.Owners[5] != 400 || backup.Strangers[2] != 300 || backup.StrangersFor[5] != 300 {
		t.Fatalf("the backup is the ledger's counts: %+v", backup)
	}
	apiframework.SaveBudget()

	if err := os.WriteFile(filepath.Join(dir, `budget.yaml`), []byte("day: [unclosed"), 0644); err != nil {
		t.Fatal(err)
	}
	apiframework.ResetBudgetForTest(dir) // the next boot finds budget.yaml corrupt
	booted := &AICompanionModule{cfg: cfg}
	booted.books.Store(apiframework.Shared())
	booted.restoreBudget(backup)
	if ownerSpent(booted, 5) != 400 || strangerSpent(booted, 2) != 300 || strangersForSpent(booted, 5) != 300 {
		t.Fatalf("re-seeded from the backup: owner=%d stranger=%d perOwner=%d",
			ownerSpent(booted, 5), strangerSpent(booted, 2), strangersForSpent(booted, 5))
	}
	if serverSpent(booted) != 700 {
		t.Fatalf("and the companion's share of the day, as SeedTokens always did: %d", serverSpent(booted))
	}
}

// A normal same-day restart finds budget.yaml whole, with its seed marks,
// so the backup is not added a second time.
func TestASameDayRestartDoesNotSeedTwice(t *testing.T) {
	dir := t.TempDir()
	apiframework.ResetBudgetForTest(dir)
	t.Cleanup(func() { apiframework.ResetBudgetForTest(``) })
	cfg := Config{DailyTokensPerCompanion: 100000, StrangerDailyTokens: 100000, StrangerTokensPerOwner: 100000}
	m := &AICompanionModule{cfg: cfg}
	m.books.Store(apiframework.Shared())
	m.restoreBudget(budgetState{Day: m.fw().Day(), // the first boot after the move
		Owners: map[int]int{5: 1200}, Strangers: map[int]int{2: 300}, StrangersFor: map[int]int{5: 300}})
	h, ok := m.reserveRoute(route{kind: routeServer}, 5, 0, 900)
	if !ok {
		t.Fatal("fixture: the hold fits")
	}
	m.settleRoute(h, 100)
	backup := m.budgetStateToSave()
	apiframework.SaveBudget()

	apiframework.ResetBudgetForTest(dir) // a normal restart: budget.yaml reads back
	booted := &AICompanionModule{cfg: cfg}
	booted.books.Store(apiframework.Shared())
	booted.restoreBudget(backup)
	if ownerSpent(booted, 5) != 1300 || strangerSpent(booted, 2) != 300 || strangersForSpent(booted, 5) != 300 {
		t.Fatalf("seeded once, not twice: owner=%d stranger=%d perOwner=%d",
			ownerSpent(booted, 5), strangerSpent(booted, 2), strangersForSpent(booted, 5))
	}
}

// A day the ledger started without seed marks (it rolled over while the
// server ran, or the boot seeded nothing because the companion's file was
// yesterday's or missing) still has its own counts in budget.yaml. The next
// same-day restart must not add the backup on top of them.
func restartKeepsOneCount(t *testing.T, now time.Time, boot func(m *AICompanionModule)) {
	t.Helper()
	dir := t.TempDir()
	apiframework.ResetBudgetForTest(dir)
	t.Cleanup(func() { apiframework.ResetBudgetForTest(``) })
	clock := func() time.Time { return now }
	apiframework.SetClockForTest(clock)
	cfg := Config{DailyTokensPerCompanion: 100000, StrangerDailyTokens: 100000, StrangerTokensPerOwner: 100000}
	m := &AICompanionModule{cfg: cfg}
	m.books.Store(apiframework.Shared())
	boot(m)
	server := route{kind: routeServer}
	h1, ok1 := m.reserveRoute(server, 5, 0, 900)
	h2, ok2 := m.reserveRoute(server, 5, 2, 900)
	if !ok1 || !ok2 {
		t.Fatal("fixture: both holds fit")
	}
	m.settleRoute(h1, 400)
	m.settleRoute(h2, 300)
	backup := m.budgetStateToSave()
	if backup.Day != today(now) || backup.Owners[5] != 400 {
		t.Fatalf("fixture: the backup is today's counts: %+v", backup)
	}
	apiframework.SaveBudget()

	apiframework.ResetBudgetForTest(dir) // a same-day restart: budget.yaml reads back
	apiframework.SetClockForTest(clock)
	booted := &AICompanionModule{cfg: cfg}
	booted.books.Store(apiframework.Shared())
	booted.restoreBudget(backup)
	if ownerSpent(booted, 5) != 400 || strangerSpent(booted, 2) != 300 || strangersForSpent(booted, 5) != 300 {
		t.Fatalf("counted once, not twice: owner=%d stranger=%d perOwner=%d",
			ownerSpent(booted, 5), strangerSpent(booted, 2), strangersForSpent(booted, 5))
	}
}

func today(t time.Time) string { return t.UTC().Format(`2006-01-02`) }

// The server ran past midnight on the ledger's clock: the new day has no
// seed marks, only its own spending.
func TestADayThatStartsWithARolloverDoesNotSeedTwice(t *testing.T) {
	late := time.Date(2026, 3, 4, 23, 30, 0, 0, time.UTC)
	past := late.Add(time.Hour)
	restartKeepsOneCount(t, past, func(m *AICompanionModule) {
		apiframework.SetClockForTest(func() time.Time { return late })
		m.restoreBudget(budgetState{Day: m.fw().Day()}) // the boot marks yesterday
		apiframework.SetClockForTest(func() time.Time { return past })
	})
}

// The server booted on yesterday's companion file, so restoreBudget seeded
// nothing and marked nothing.
func TestABootOnYesterdaysFileDoesNotSeedTwice(t *testing.T) {
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	restartKeepsOneCount(t, now, func(m *AICompanionModule) {
		m.restoreBudget(budgetState{Day: today(now.Add(-24 * time.Hour)), Owners: map[int]int{5: 999}})
	})
}

// The server booted with no companion file at all (loadBudget returns
// before restoreBudget).
func TestABootWithNoFileDoesNotSeedTwice(t *testing.T) {
	now := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	restartKeepsOneCount(t, now, func(m *AICompanionModule) {})
}

// logged is every line tee kept with its colours taken out, one string.
func logged(tee *logTee) string {
	tee.mu.Lock()
	defer tee.mu.Unlock()
	out := ``
	for _, line := range tee.lines {
		out += regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(line, ``) + "\n"
	}
	return out
}

// R43, the logging half: a decision the ledger refuses is logged with the
// counter that refused it, through dispatch, reserveRoute and
// logBudgetRefusal as the game runs them.
func TestARefusedDecisionLogsWhichCounterSaidNo(t *testing.T) {
	_, _, _, her := harmWorld(t, `off`)
	srv, hits := countingServer(t)
	tee := &logTee{}
	mudlog.SetupLogger(tee, "", "", false)
	t.Cleanup(func() { mudlog.SetupLogger(nil, "", "", false) })
	for _, tc := range []struct {
		name    string
		stim    stimulus
		setCaps func(m *AICompanionModule)
		want    string
	}{
		{`a passer-by past their allowance`,
			stimulus{Kind: `heard`, Speaker: `Bram`, Text: `Mara, hello`, AskerUserId: 2},
			func(m *AICompanionModule) { m.cfg.StrangerDailyTokens = 10 },
			apiframework.DimCompanionStranger},
		{`the owner past the companion's allowance`,
			stimulus{Kind: `heard`, Speaker: `Corvin`, Text: `Mara, hello`, FromOwner: true, AskerUserId: 1},
			func(m *AICompanionModule) { m.cfg.DailyTokensPerCompanion = 10 },
			apiframework.DimCompanionOwner},
	} {
		m, c := senderModule(srv.URL, true)
		c.instanceId = her.InstanceId
		m.ctrls = map[int]*controller{1: c}
		m.minds = map[string]*Mind{mindIdentifier(c.mind.OwnerUserId, c.mind.MobId): c.mind}
		tc.setCaps(m)
		tee.mu.Lock()
		tee.lines = nil
		tee.mu.Unlock()

		util.LockMud()
		c.push(tc.stim)
		m.dispatch(c)
		inFlight := c.inFlight
		util.UnlockMud()
		m.decisions.Wait()
		if inFlight {
			t.Fatalf("%s: fixture: the call is refused, not started", tc.name)
		}
		if got := logged(tee); !regexp.MustCompile(`budgetRefused.*refusedBy="` + regexp.QuoteMeta(tc.want) + `"`).MatchString(got) {
			t.Fatalf("%s: the refusal log names %s:\n%s", tc.name, tc.want, got)
		}
	}
	if hits.Load() != 0 {
		t.Fatalf("no refused call reaches the provider: %d", hits.Load())
	}
}

// R37: saveBudget writes the ledger's budget.yaml before the companion's
// own file. The module here has no plugin, so the companion's write
// panics where it starts: budget.yaml is already on disk only if it was
// written first.
func TestSaveBudgetWritesTheLedgerFirst(t *testing.T) {
	dir := t.TempDir()
	apiframework.ResetBudgetForTest(dir)
	t.Cleanup(func() { apiframework.ResetBudgetForTest(``) })
	m := &AICompanionModule{cfg: Config{Enabled: true, DailyTokensPerCompanion: 100000}}
	m.books.Store(apiframework.Shared())
	h, ok := m.reserveRoute(route{kind: routeServer}, 5, 0, 900)
	if !ok {
		t.Fatal("fixture: the hold fits")
	}
	m.settleRoute(h, 400)
	panicked := func() (p bool) {
		defer func() { p = recover() != nil }()
		m.saveBudget()
		return false
	}()
	if !panicked {
		t.Fatal("fixture: with no plugin the companion's own write panics")
	}
	raw, err := os.ReadFile(filepath.Join(dir, `budget.yaml`))
	if err != nil || !regexp.MustCompile(`companion\.owner:5: 400`).Match(raw) {
		t.Fatalf("budget.yaml was written before the companion's file: %v %s", err, raw)
	}
}
