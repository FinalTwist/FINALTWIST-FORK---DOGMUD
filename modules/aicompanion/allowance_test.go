package aicompanion

import (
	"reflect"
	"testing"

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
