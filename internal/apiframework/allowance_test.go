package apiframework

import (
	"errors"
	"testing"
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
