package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/stretchr/testify/require"
)

// Lighting plan 5c final review, finding 1: CalcSneakScoreVsObserver counted
// an observer as seeing a lit room when the room was lit or the observer held
// the nightvision FLAG. An infravision observer (flag infraredvision only)
// that the window gives shapes in a faint room was treated as blind, and a
// nightvision holder in a pitch-dark room its window reads as blind was
// treated as seeing. The observer now counts as lit exactly when its sight is
// not SightNone.
func TestCalcSneakScoreVsObserver_ReadsObserverSight(t *testing.T) {
	const infraId, nightId = 9621, 9622
	restore := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		infraId: {ConditionId: infraId, Name: "Test Heat Sight", RoundInterval: 1, TriggerCount: 10,
			Flags:   []conditions.Flag{conditions.InfraredVision},
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectInfraReach: {Literal: 30}}},
		nightId: {ConditionId: nightId, Name: "Test Night Sight", RoundInterval: 1, TriggerCount: 10,
			Flags:   []conditions.Flag{conditions.NightVision},
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectNightVisionStrength: {Literal: 24}}},
	})
	defer restore()

	sneaker := newTestChar()
	sneaker.Stats.Dexterity.ValueAdj = 100
	sneaker.Skills[string(skills.Skullduggery)] = 20
	dark := CalcSneakScore(sneaker, false)
	lit := CalcSneakScore(sneaker, true)
	require.NotEqual(t, dark, lit, "the lit modifier must move the score or this test proves nothing")

	observer := func(conditionId int) func() *characters.Character {
		return func() *characters.Character {
			c := newTestChar()
			if conditionId != 0 {
				require.True(t, c.Conditions.AddCondition(conditionId, true))
			}
			return c
		}
	}

	cases := []struct {
		name  string
		light int
		obs   func() *characters.Character
		want  float64
	}{
		{"normal observer at light 10 sees a dark room", 10, observer(0), dark},
		{"infravision observer at light 10 perceives", 10, observer(infraId), lit},
		{"infravision observer below minus its reach sees nothing", -40, observer(infraId), dark},
		{"nightvision observer at light 0 sees nothing", 0, observer(nightId), dark},
		{"nightvision observer at light 10 perceives", 10, observer(nightId), lit},
		{"normal observer in a lit room", 60, observer(0), lit},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := CalcSneakScoreVsObserver(sneaker, c.obs(), messaging.FixedLight(c.light))
			require.Equal(t, c.want, got)
		})
	}
}
