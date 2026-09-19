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
		t.Run(tc.name, func(t *testing.T) {
			counterer, countered := counterTestPair()
			calls := 0
			restore := SetChannelAttackContestRunnerForTest(counterWinRunner(&calls))
			t.Cleanup(restore)
			res := ExecuteCounter(counterer, countered, tc.shape, combatvocab.DefenceDodge, true)
			if res.Countered != tc.want {
				t.Errorf("Countered = %v, want %v", res.Countered, tc.want)
			}
			if !tc.want && calls != 0 {
				t.Errorf("a refused counter must run no contest, ran %d", calls)
			}
			if !tc.want && countered.Health != 100000 {
				t.Errorf("a refused counter must deal no damage")
			}
		})
	}
}

// Words answer words (owner ruling 2026-09-18): a defy crit counter-taunts,
// for charm as well as for taunt, and the counter-taunt lives in
// internal/actions. The primitive therefore refuses a defy defence so nobody
// can route a retort into a sword-swing.
func TestExecuteCounter_RefusesADefyDefence(t *testing.T) {
	pinCounterConfig(t, 0.5)
	for _, shape := range []combatvocab.Attack{
		combatvocab.Rhetoric(combatvocab.TargetSingle),
		combatvocab.Spell(combatvocab.DamageSocial, combatvocab.TargetSingle),
	} {
		t.Run(shape.String(), func(t *testing.T) {
			counterer, countered := counterTestPair()
			calls := 0
			t.Cleanup(SetChannelAttackContestRunnerForTest(counterWinRunner(&calls)))
			res := ExecuteCounter(counterer, countered, shape, combatvocab.DefenceDefy, true)
			if res.Countered || calls != 0 || countered.Health != 100000 {
				t.Errorf("a defy defence must never swing (countered=%v calls=%d health=%d)",
					res.Countered, calls, countered.Health)
			}
		})
	}
}

// The primitive refuses a missing winner rather than rendering an empty
// pool. Unreachable in production (a defensive crit always names its
// defence), so pinned here directly.
func TestExecuteCounter_RefusesNoWinner(t *testing.T) {
	pinCounterConfig(t, 0.5)
	counterer, countered := counterTestPair()
	calls := 0
	t.Cleanup(SetChannelAttackContestRunnerForTest(counterWinRunner(&calls)))
	res := ExecuteCounter(counterer, countered, combatvocab.Melee(combatvocab.TargetSingle), combatvocab.DefenceNone, true)
	if res.Countered || calls != 0 || countered.Health != 100000 {
		t.Errorf("no winner must mean no counter (countered=%v calls=%d health=%d)", res.Countered, calls, countered.Health)
	}
}
