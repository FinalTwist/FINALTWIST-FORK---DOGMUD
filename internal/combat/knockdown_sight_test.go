package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/contest"
)

// Lighting plan 5b: resisting a knockdown is a defence, so the defender's
// knockdown score rides SituationalDefenceMult for the move's shape. A forced
// clean hit makes the knockdown contest run every time; the swapped
// knockdownContestRunner records the defender score it receives.
func TestKnockdownResistanceTakesTheDefendersEyes(t *testing.T) {
	pinDefenceAdmissionConfig(t)
	pinSightRampCaps(t)

	orig := knockdownContestRunner
	t.Cleanup(func() { knockdownContestRunner = orig })

	atk, def := newDefenceTestCharacter(t), newDefenceTestCharacter(t)
	capture := func(room verdictLight) float64 {
		got := -1.0
		knockdownContestRunner = func(_ float64, entries []contest.Entry) contest.Result {
			got = entries[0].Score
			return contest.Result{Contested: true}
		}
		def.Health = def.HealthMax.Value
		def.Stamina = def.StaminaMax.Value
		executeSkillMoveWithRunner(SkillMoveParams{
			Attacker:             atk,
			Defender:             def,
			Shape:                combatvocab.Melee(combatvocab.TargetSingle),
			Room:                 room,
			Attack:               side(148, 52),
			DamagePercent:        1.0,
			KnockdownFactor:      1.0,
			DamageStat:           100,
			MitigationMultiplier: 1.0,
		}, cleanWinRunner)
		return got
	}

	comfortable, dazzled := capture(sightComfortableLight), capture(sightDazzledLight)
	if comfortable <= 0 {
		t.Fatalf("fixture guard: the knockdown contest did not run or scored %v", comfortable)
	}
	if got := dazzled / comfortable; !sightNear(got, sightDazzledWant) {
		t.Errorf("knockdown resistance dazzled/comfortable = %v, want %v", got, sightDazzledWant)
	}
}
