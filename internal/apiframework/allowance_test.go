package apiframework

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// R1, R3, R4, R8, R41: every charge is checked, and a refusal anywhere
// holds nothing anywhere.
func TestReserveChargesEveryAllowanceOrNone(t *testing.T) {
	ResetBudgetForTest(``)
	stranger := Charge{Dim: DimCompanionStranger, UserId: 2, Limit: 1000}
	perOwner := Charge{Dim: DimCompanionStrangersFor, UserId: 5, Limit: 1500}
	if _, err := budget.reserve(ConsumerCompanion, 900, 0, 0, true, []Charge{stranger, perOwner}); err != nil {
		t.Fatal(err)
	}
	other := Charge{Dim: DimCompanionStranger, UserId: 3, Limit: 1000}
	if _, err := budget.reserve(ConsumerCompanion, 700, 0, 0, true, []Charge{other, perOwner}); !errors.Is(err, ErrOverAllowance) {
		t.Fatalf("the second charge refuses the whole reservation: %v", err)
	}
	if Allowance(DimCompanionStranger, 3) != 0 || Allowance(DimCompanionStrangersFor, 5) != 900 || Today().Tokens != 900 {
		t.Fatalf("a refusal holds nothing anywhere: other=%d perOwner=%d server=%d",
			Allowance(DimCompanionStranger, 3), Allowance(DimCompanionStrangersFor, 5), Today().Tokens)
	}
	// The server's budget refusing charges no allowance either.
	if _, err := budget.reserve(ConsumerCompanion, 200, 1000, 0, true, []Charge{other}); !errors.Is(err, ErrOverBudget) {
		t.Fatalf("over the server's budget: %v", err)
	}
	if Allowance(DimCompanionStranger, 3) != 0 {
		t.Fatal("a server refusal leaves the allowance untouched")
	}
	// A limit of 0 is no cap, and the spend is still counted.
	if _, err := budget.reserve(ConsumerCompanion, 5000, 0, 0, true, []Charge{{Dim: DimCompanionOwner, UserId: 7}}); err != nil {
		t.Fatal(err)
	}
	if Allowance(DimCompanionOwner, 7) != 5000 {
		t.Fatalf("counted with no cap: %d", Allowance(DimCompanionOwner, 7))
	}
}

// R11: a relayed call (the player's own key) is charged to its allowances
// and to nothing of the server's, and the server's budget cannot refuse it.
func TestARelayReserveLeavesTheServerAlone(t *testing.T) {
	ResetBudgetForTest(``)
	c := Charge{Dim: DimCompanionStranger, UserId: 2, Limit: 1000}
	h, err := budget.reserve(ConsumerCompanion, 400, 1000, 0, false, []Charge{c})
	if err != nil {
		t.Fatal(err)
	}
	if h.SpendServer || len(h.Charges) != 1 || h.Tokens != 400 || h.Day != Today().Day {
		t.Fatalf("the hold records what it touched: %+v", h)
	}
	if u := Today(); u.Tokens != 0 || u.Outstanding != 0 || u.Calls != 0 || len(u.ByConsumer) != 0 {
		t.Fatalf("nothing of the server's: %+v", u)
	}
	SetSpentForTest(1000, 0)
	if _, err := budget.reserve(ConsumerCompanion, 400, 1000, 0, false, []Charge{c}); err != nil {
		t.Fatal("the server's spent budget does not refuse the player's own key")
	}
	if Allowance(DimCompanionStranger, 2) != 800 {
		t.Fatalf("both charged to the allowance: %d", Allowance(DimCompanionStranger, 2))
	}
}

// The ledger's clock is the only day.
func TestDayIsTheLedgersClock(t *testing.T) {
	k := NewBooksForTest()
	if k.Day() != Today().Day {
		t.Fatalf("today: %s vs %s", k.Day(), Today().Day)
	}
}

// SetAllowanceForTest sets one counter, for tests that start mid-day.
func TestSetAllowanceForTest(t *testing.T) {
	k := NewBooksForTest()
	k.SetAllowanceForTest(DimCompanionOwner, 5, 1234)
	if k.Allowance(DimCompanionOwner, 5) != 1234 || k.Allowance(DimCompanionOwner, 6) != 0 {
		t.Fatal("one counter, by dimension and user")
	}
}

// R43: a refusal says which counter refused it, so a log line can tell a
// spent server day from one player's spent allowance.
func TestARefusalNamesItsCounter(t *testing.T) {
	ResetBudgetForTest(``)
	_, err := budget.reserve(ConsumerCompanion, 200, 100, 0, true, nil)
	if !errors.Is(err, ErrOverBudget) || RefusedBy(err) != RefusedGlobal {
		t.Fatalf("the day's budget: %v (%q)", err, RefusedBy(err))
	}
	c := Charge{Dim: DimCompanionStrangersFor, UserId: 5, Limit: 100}
	_, err = budget.reserve(ConsumerCompanion, 200, 0, 0, true, []Charge{{Dim: DimCompanionStranger, UserId: 2}, c})
	if !errors.Is(err, ErrOverAllowance) || RefusedBy(err) != DimCompanionStrangersFor {
		t.Fatalf("the charge that refused: %v (%q)", err, RefusedBy(err))
	}
	if RefusedBy(nil) != `` || RefusedBy(errors.New(`other`)) != `` {
		t.Fatal("anything else names no counter")
	}
}

// R14, R15, R20: a relayed count is held between nothing and its hold; a
// server-key count is trusted, overage included; no counter goes negative.
func TestSettleClampsAndFloors(t *testing.T) {
	ResetBudgetForTest(``)
	c := Charge{Dim: DimCompanionStranger, UserId: 2, Limit: 10000}
	relay, _ := budget.reserve(ConsumerCompanion, 400, 0, 0, false, []Charge{c})
	budget.settle(relay, 5000, false)
	if Allowance(DimCompanionStranger, 2) != 400 {
		t.Fatalf("a relayed count never charges past its hold: %d", Allowance(DimCompanionStranger, 2))
	}
	relay2, _ := budget.reserve(ConsumerCompanion, 400, 0, 0, false, []Charge{c})
	budget.settle(relay2, -900, false)
	if Allowance(DimCompanionStranger, 2) != 400 {
		t.Fatalf("nor below nothing: %d", Allowance(DimCompanionStranger, 2))
	}
	if u := Today(); u.Tokens != 0 || u.Outstanding != 0 {
		t.Fatalf("a relayed settlement touches nothing of the server's: %+v", u)
	}

	owner := Charge{Dim: DimCompanionOwner, UserId: 5}
	h, _ := budget.reserve(ConsumerCompanion, 300, 0, 0, true, []Charge{owner})
	budget.settle(h, 450, false)
	share := 0
	for _, cu := range Today().ByConsumer {
		if cu.Consumer == ConsumerCompanion {
			share = cu.Tokens
		}
	}
	if Today().Tokens != 450 || share != 450 || Allowance(DimCompanionOwner, 5) != 450 || Today().Outstanding != 0 {
		t.Fatalf("server-key overage is charged everywhere: total=%d share=%d owner=%d", Today().Tokens, share, Allowance(DimCompanionOwner, 5))
	}

	// Both counters set below the hold, so the refund would take them
	// negative: the allowance and the server total (budget.go:186) floor.
	h2, _ := budget.reserve(ConsumerCompanion, 300, 0, 0, true, []Charge{owner})
	shared.SetAllowanceForTest(DimCompanionOwner, 5, 100)
	SetSpentForTest(100, 300)
	budget.settle(h2, 0, false)
	if Allowance(DimCompanionOwner, 5) != 0 {
		t.Fatalf("an allowance floors at nothing: %d", Allowance(DimCompanionOwner, 5))
	}
	if u := Today(); u.Tokens != 0 || u.Outstanding != 0 {
		t.Fatalf("the server total floors at nothing: %+v", u)
	}
}

// R17, R21: a hold made yesterday gives nothing back to today's
// allowances, and its overage is still charged.
func TestAHoldFromYesterdayRefundsNoAllowance(t *testing.T) {
	ResetBudgetForTest(``)
	day1 := time.Date(2026, 9, 26, 23, 59, 0, 0, time.UTC)
	SetClockForTest(func() time.Time { return day1 })
	t.Cleanup(func() { SetClockForTest(time.Now) })
	owner := Charge{Dim: DimCompanionOwner, UserId: 5}
	stranger := Charge{Dim: DimCompanionStranger, UserId: 2}
	hs, _ := budget.reserve(ConsumerCompanion, 900, 0, 0, true, []Charge{owner})
	hs2, _ := budget.reserve(ConsumerCompanion, 100, 0, 0, true, []Charge{owner})
	hr, _ := budget.reserve(ConsumerCompanion, 400, 0, 0, false, []Charge{stranger})

	SetClockForTest(func() time.Time { return day1.Add(2 * time.Minute) })
	if Allowance(DimCompanionOwner, 5) != 0 || Allowance(DimCompanionStranger, 2) != 0 {
		t.Fatal("a new day's allowances start at nothing")
	}
	shared.SetAllowanceForTest(DimCompanionOwner, 5, 500)
	shared.SetAllowanceForTest(DimCompanionStranger, 2, 300)
	budget.settle(hs, 100, false)
	budget.settle(hs2, 250, false)
	budget.settle(hr, 0, false)
	if Allowance(DimCompanionOwner, 5) != 650 {
		t.Fatalf("no refund from yesterday, overage still charged: %d", Allowance(DimCompanionOwner, 5))
	}
	if Allowance(DimCompanionStranger, 2) != 300 {
		t.Fatalf("the relayed hold gives nothing back: %d", Allowance(DimCompanionStranger, 2))
	}
	if u := Today(); u.Outstanding != 0 || u.Tokens != 350 {
		t.Fatalf("the server settles as before: %+v", u)
	}
}

// A consumer holds at most its share of the day's budget; with no global
// cap there is no share cap; a player's own key is never held to one.
func TestAConsumerIsHeldToItsShare(t *testing.T) {
	ResetBudgetForTest(``)
	restore := SetServerForTest(ServerSettings{Endpoint: Endpoint{BaseURL: DefaultBaseURL},
		DailyTokenBudget: 1000, BaublesSharePercent: 25, BreakerErrors: 2, BreakerSeconds: 60})
	t.Cleanup(func() { restore(); ResetBudgetForTest(``) })

	h, err := Reserve(ConsumerBaubles, 200, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Reserve(ConsumerBaubles, 100, true); !errors.Is(err, ErrOverShare) || RefusedBy(err) != RefusedShare {
		t.Fatalf("over a 250-token share, and it says so: %v (%q)", err, RefusedBy(err))
	}
	if _, err := Reserve(ConsumerCompanion, 700, true); err != nil {
		t.Fatal("a consumer with no share cap spends the rest")
	}
	Settle(h, 50, false)
	if _, err := Reserve(ConsumerBaubles, 200, true); err != nil {
		t.Fatal("settling frees share")
	}
	if _, err := Reserve(ConsumerBaubles, 5000, false); err != nil {
		t.Fatal("a player's own key is held to no share")
	}

	ResetBudgetForTest(``)
	noCap := SetServerForTest(ServerSettings{Endpoint: Endpoint{BaseURL: DefaultBaseURL},
		DailyTokenBudget: 0, BaublesSharePercent: 25, BreakerErrors: 2, BreakerSeconds: 60})
	defer noCap()
	if _, err := Reserve(ConsumerBaubles, 100000, true); err != nil {
		t.Fatal("no global cap is no share cap")
	}
}

func TestSharePercentByConsumer(t *testing.T) {
	s := ServerSettings{CompanionSharePercent: 60, BaublesSharePercent: 25}
	if s.SharePercent(ConsumerCompanion) != 60 || s.SharePercent(ConsumerBaubles) != 25 || s.SharePercent(`other`) != 0 {
		t.Fatal("each consumer's own share; an unknown one has none")
	}
}

// R38, R39: allowances and seed marks are living state with the rest of
// the day. A dimension is seeded once a day, today only; a same-day restart
// finds the mark and seeds nothing; a quarantine loses the counts and the
// marks together, so the next seed applies again.
func TestAllowancesSaveLoadAndSeedOnce(t *testing.T) {
	dir := t.TempDir()
	ResetBudgetForTest(dir)
	t.Cleanup(func() { ResetBudgetForTest(``) })
	h, err := Reserve(ConsumerBaubles, 500, true, Charge{Dim: DimBaublesFinder, UserId: 7, Limit: 20000})
	if err != nil {
		t.Fatal(err)
	}
	Settle(h, 123, false)
	day := Today().Day
	backup := map[int]int{5: 777, 8: 0}

	SeedAllowances(DimCompanionOwner, `1999-01-01`, map[int]int{6: 50})
	if Allowance(DimCompanionOwner, 6) != 0 {
		t.Fatal("a stale day seeds nothing, and leaves no mark")
	}
	SeedAllowances(DimCompanionOwner, day, backup)
	SeedAllowances(DimCompanionOwner, day, backup)
	if Allowance(DimCompanionOwner, 5) != 777 || Allowance(DimCompanionOwner, 8) != 0 {
		t.Fatalf("seeded once a day per dimension: %d", Allowance(DimCompanionOwner, 5))
	}
	if got := Allowances(DimCompanionOwner); !reflect.DeepEqual(got, map[int]int{5: 777}) {
		t.Fatalf("one dimension's spends, nothing spent left out: %v", got)
	}
	if got := Allowances(DimCompanionStranger); len(got) != 0 {
		t.Fatalf("a dimension is its own: %v", got)
	}
	SaveBudget()

	ResetBudgetForTest(dir) // a same-day restart
	if Allowance(DimBaublesFinder, 7) != 123 || Allowance(DimCompanionOwner, 5) != 777 {
		t.Fatalf("a restart keeps the day's allowances: finder=%d owner=%d", Allowance(DimBaublesFinder, 7), Allowance(DimCompanionOwner, 5))
	}
	SeedAllowances(DimCompanionOwner, day, backup)
	if Allowance(DimCompanionOwner, 5) != 777 {
		t.Fatalf("the mark is saved too: a same-day restart seeds nothing: %d", Allowance(DimCompanionOwner, 5))
	}
	raw, err := os.ReadFile(filepath.Join(dir, `budget.yaml`))
	if err != nil || !strings.Contains(string(raw), `by_user:`) || !strings.Contains(string(raw), `baubles.finder:7`) ||
		!strings.Contains(string(raw), `seeded:`) {
		t.Fatalf("by_user and seeded in budget.yaml: %s", raw)
	}

	if err := os.WriteFile(filepath.Join(dir, `budget.yaml`), []byte("day: [unclosed"), 0644); err != nil {
		t.Fatal(err)
	}
	ResetBudgetForTest(dir) // a boot that finds the file corrupt
	if Allowance(DimBaublesFinder, 7) != 0 || Allowance(DimCompanionOwner, 5) != 0 {
		t.Fatal("a quarantine restarts the day's allowances with its totals")
	}
	SeedAllowances(DimCompanionOwner, day, backup)
	if Allowance(DimCompanionOwner, 5) != 777 {
		t.Fatalf("the marks went with the counts, so the backup seeds again: %d", Allowance(DimCompanionOwner, 5))
	}
}

// R38: SaveBudget marshals copies, never the live maps. Only -race can see
// the shared map; run it in CI or the Docker test image (see the gate). The
// key space is bounded (ten users and ten seed dimensions per goroutine) and
// the loop capped, so the maps stay small and it runs in seconds under -race.
func TestSaveBudgetCopiesEveryMapUnderTheLock(t *testing.T) {
	dir := t.TempDir()
	ResetBudgetForTest(dir)
	restore := SetServerForTest(ServerSettings{Endpoint: Endpoint{BaseURL: DefaultBaseURL}, BreakerErrors: 2, BreakerSeconds: 60})
	t.Cleanup(func() { restore(); ResetBudgetForTest(``) })
	stop := make(chan struct{})
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 20000; i++ {
				// The work comes first, so goroutine 0 writes key 0 (read
				// back below) before it can see stop.
				c := Charge{Dim: DimBaublesFinder, UserId: g*10 + i%10}
				if h, err := Reserve(ConsumerBaubles, 1, true, c); err == nil {
					Settle(h, 1, false)
				}
				SeedAllowances(fmt.Sprintf(`test.seed%d`, g*10+i%10), Today().Day, map[int]int{1: 1})
				select {
				case <-stop:
					return
				default:
				}
			}
		}(g)
	}
	for i := 0; i < 200; i++ {
		SaveBudget()
	}
	close(stop)
	wg.Wait()
	SaveBudget()
	want := Allowance(DimBaublesFinder, 0)
	ResetBudgetForTest(dir)
	if want < 1 || Allowance(DimBaublesFinder, 0) != want {
		t.Fatalf("what was saved reads back: saved %d, read %d", want, Allowance(DimBaublesFinder, 0))
	}
}
