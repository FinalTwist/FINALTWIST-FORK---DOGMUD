package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// Every real defence must score, cost and pool through the ONE vocabulary.
// GetDefenseScoreFor's default arm returns 0 and defenseCostRequest's returns
// false, so a defence the switches forgot would silently enter every contest
// at 0 and cost nothing. Only this assertion catches it.
func TestEveryDefenceScoresAndCosts(t *testing.T) {
	c := New()
	c.Stats.Dexterity.ValueAdj = 100
	c.Stats.Strength.ValueAdj = 100
	c.Stats.Willpower.ValueAdj = 100
	for _, d := range combatvocab.Defences() {
		if c.GetDefenseScore(d) <= 0 {
			t.Errorf("GetDefenseScore(%s) = 0: the switch has no arm for it", d)
		}
		if _, ok := defenseCostRequest(d); !ok {
			t.Errorf("defenseCostRequest(%s) unknown: the switch has no arm for it", d)
		}
	}
	if _, ok := defenseCostRequest(combatvocab.DefenceNone); ok {
		t.Error("DefenceNone must not price as anything")
	}
}
