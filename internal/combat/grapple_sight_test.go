package combat

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// Lighting plan 5b: a grapple and a submission are opposed rolls where BOTH
// parties need to see, so each side's pre-roll score pays its own sight ramp.
// AttackScore/DefenseScore are pre-roll values, so the ratio is exact.
func TestGrappleAndSubmissionScoresTakeBothPartiesEyes(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.Validate()
	configs.SetConfigForTest(t, cfg)

	near := func(got, want float64) bool { return math.Abs(got-want) <= 1e-9 }

	atk := newGrappleMarginChar(100, 20)
	def := newGrappleMarginChar(80, 10)
	comfortable := AttemptGrapple(atk, def, verdictLight(60))
	dazzled := AttemptGrapple(atk, def, verdictLight(90))
	if r := dazzled.AttackScore / comfortable.AttackScore; !near(r, 0.88) {
		t.Errorf("grapple attacker dazzled/comfortable = %v, want 0.88", r)
	}
	if r := dazzled.DefenseScore / comfortable.DefenseScore; !near(r, 0.88) {
		t.Errorf("grapple defender dazzled/comfortable = %v, want 0.88", r)
	}

	subAtk := newGrappleMarginChar(100, 20)
	subDef := newGrappleMarginChar(80, 10)
	subAtk.Stats.Strength.ValueAdj = 110
	subDef.Stats.Strength.ValueAdj = 90
	subDef.Stats.Vitality.ValueAdj = 100
	sc := RollSubmissionAttempt(subAtk, subDef, position.SubArmbar, verdictLight(60))
	sd := RollSubmissionAttempt(subAtk, subDef, position.SubArmbar, verdictLight(90))
	if r := sd.AttackerScore / sc.AttackerScore; !near(r, 0.88) {
		t.Errorf("submission attempter dazzled/comfortable = %v, want 0.88", r)
	}
	if r := sd.DefenderScore / sc.DefenderScore; !near(r, 0.88) {
		t.Errorf("submission recipient dazzled/comfortable = %v, want 0.88", r)
	}
}
