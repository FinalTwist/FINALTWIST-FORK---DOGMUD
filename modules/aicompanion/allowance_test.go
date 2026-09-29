package aicompanion

import (
	"reflect"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/apiframework"
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
