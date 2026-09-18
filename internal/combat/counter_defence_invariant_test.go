package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// The counter tier reads ChannelDefenceResult.Defence to choose its pool, so
// a defensive crit MUST name the defence that won it. Today it does, because
// DefensiveCrit is set only after the winner is recorded; this pins that a
// crit and a named winner arrive together on the result, so a future edit
// cannot make the pool lookup silently empty.
func TestResolveChannelAttack_ADefensiveCritNamesItsDefence(t *testing.T) {
	pinCounterConfig(t, 0.5)
	attacker, defender := counterTestPair()
	calls := 0
	restore := SetChannelAttackContestRunnerForTest(counterDefensiveCritRunner(&calls))
	defer restore()

	side := AttackSide{
		Stat: attacker.Stats.Strength.ValueAdj, StatName: "strength",
		Skill: attacker.GetCombatSkillTag(), SkillRank: attacker.GetCombatSkillLevel(),
		Mult: 1.0,
	}
	out := ResolveChannelAttack(combatvocab.Melee(combatvocab.TargetSingle), side, attacker, defender)
	if !out.DefensiveCrit {
		t.Fatalf("fixture error: the contest was supposed to be a defensive crit, got %+v", out)
	}
	if out.Defence == combatvocab.DefenceNone {
		t.Fatalf("a defensive crit must name the defence that won; got none")
	}
}
