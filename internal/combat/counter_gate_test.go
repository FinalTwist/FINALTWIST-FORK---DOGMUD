package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// Owner ruling (counters spec 2): area attacks earn no counter, and multi
// rides along because a counter answers one deliberate attack at one target.
// The gate lives in the primitive so no exit can bypass it by omission.
func TestExecuteCounter_OnlyASingleTargetAttackEarnsACounter(t *testing.T) {
	pinCounterConfig(t, 0.5)
	for _, tc := range []struct {
		name  string
		shape combatvocab.Attack
		want  bool
	}{
		{"single", combatvocab.Melee(combatvocab.TargetSingle), true},
		{"area spell", combatvocab.Spell(combatvocab.DamagePhysical, combatvocab.TargetArea), false},
		{"multi spell", combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetMulti), false},
		{"thrown area", combatvocab.Thrown(combatvocab.TargetArea), false},
	} {
		counterer, countered := counterTestPair()
		calls := 0
		restore := SetChannelAttackContestRunnerForTest(counterWinRunner(&calls))
		res := ExecuteCounter(counterer, countered, tc.shape, combatvocab.DefenceDodge, true)
		restore()
		if res.Countered != tc.want {
			t.Errorf("%s: Countered = %v, want %v", tc.name, res.Countered, tc.want)
		}
		if !tc.want && calls != 0 {
			t.Errorf("%s: a refused counter must run no contest, ran %d", tc.name, calls)
		}
		if !tc.want && countered.Health != 100000 {
			t.Errorf("%s: a refused counter must deal no damage", tc.name)
		}
	}
}
