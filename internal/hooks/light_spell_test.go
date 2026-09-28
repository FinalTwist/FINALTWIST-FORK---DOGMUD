package hooks

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// testGlowConditionId is a magnitude-driven, adjustable light, the shape of
// the shipped condition 1 Illumination.
const testGlowConditionId = 9731

// seedTestGlowCondition replaces the condition registry with a magnitude
// light (9731), a plain condition (9732), a magnitude nightvision (9733) and
// a heat sight reading infra reach from its magnitude (9734) until the test
// ends.
func seedTestGlowCondition(t *testing.T) {
	t.Helper()
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		testGlowConditionId: {ConditionId: testGlowConditionId, Name: "Test Glow", TriggerCount: 4, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {UsesMagnitude: true}},
			Flags:   []conditions.Flag{conditions.Adjustable}},
		9732: {ConditionId: 9732, Name: "Test Plain", TriggerCount: 4, RoundInterval: 1},
		9733: {ConditionId: 9733, Name: "Test Night Sight", TriggerCount: 1, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectNightVisionStrength: {UsesMagnitude: true}},
			Flags:   []conditions.Flag{conditions.NightVision}},
		9734: {ConditionId: 9734, Name: "Test Heat Sight", TriggerCount: 1, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{
				conditions.EffectNightVisionStrength: {Literal: 12}, conditions.EffectInfraReach: {UsesMagnitude: true}},
			Flags: []conditions.Flag{conditions.InfraredVision}},
	}))
}

func TestLightSpellScalesFromStatAndSkill(t *testing.T) {
	// GetConfig validates before it returns, so the knobs carry their defaults.
	configs.SetConfigForTest(t, configs.GetConfig())
	seedTestGlowCondition(t)
	spell := &spells.SpellData{SpellId: "test-glow", PrimaryStat: "willpower"}

	cases := []struct {
		stat, skill  int
		wantStrength float64
		wantTriggers int
	}{
		{100, 0, 50, 4},  // a new character: today's 20 real minutes
		{175, 65, 90, 9}, // endgame: about 45 minutes
	}
	for _, c := range cases {
		caster := characters.New()
		caster.Stats.Willpower.ValueAdj = c.stat
		caster.SetSkill("spellcasting", c.skill)
		mag, trig, ok := magnitudeSpellApplication(spell, caster, testGlowConditionId)
		if !ok || mag != c.wantStrength || trig != c.wantTriggers {
			t.Errorf("stat %d skill %d: (%v, %d, %v), want (%v, %d, true)", c.stat, c.skill, mag, trig, ok, c.wantStrength, c.wantTriggers)
		}
	}
	if _, _, ok := magnitudeSpellApplication(spell, characters.New(), 9732); ok {
		t.Error("a non-light condition was treated as a light spell")
	}
}

// TestLightSpellReadsTheKnobs moves one knob off its default, so a formula
// that hard-coded the shipped 40 would fail.
func TestLightSpellReadsTheKnobs(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.LightSpellStrengthBase = 10
	configs.SetConfigForTest(t, cfg)
	seedTestGlowCondition(t)
	spell := &spells.SpellData{SpellId: "test-glow", PrimaryStat: "willpower"}

	caster := characters.New()
	caster.Stats.Willpower.ValueAdj = 100
	caster.SetSkill("spellcasting", 0)
	mag, trig, ok := magnitudeSpellApplication(spell, caster, testGlowConditionId)
	if !ok || mag != 20 || trig != 4 {
		t.Errorf("LightSpellStrengthBase 10, stat 100 skill 0: (%v, %d, %v), want (20, 4, true)", mag, trig, ok)
	}
}

func TestVisionSpellsScaleFromStatAndSkill(t *testing.T) {
	configs.SetConfigForTest(t, configs.GetConfig())
	seedTestGlowCondition(t)
	spell := &spells.SpellData{SpellId: "test-vision", PrimaryStat: "willpower"}
	cases := []struct {
		conditionId, stat, skill int
		wantMag                  float64
		wantTriggers             int
	}{
		{9733, 100, 0, 12, 4},
		{9733, 130, 30, 4 + 130/12.5 + 30/6.5, 6},
		{9733, 175, 65, 28, 9}, // uncapped here: the window clamps at 24
		{9734, 100, 0, 5 + 100/7.0, 4},
		{9734, 130, 30, 5 + 130/7.0 + 10, 6},
		{9734, 175, 65, 50, 9}, // 51.67 capped at LightInfraReachCap
	}
	for _, c := range cases {
		caster := characters.New()
		caster.Stats.Willpower.ValueAdj = c.stat
		caster.SetSkill("spellcasting", c.skill)
		mag, trig, ok := magnitudeSpellApplication(spell, caster, c.conditionId)
		if !ok || math.Abs(mag-c.wantMag) > 1e-9 || trig != c.wantTriggers {
			t.Errorf("condition %d stat %d skill %d: (%v, %d, %v), want (%v, %d, true)",
				c.conditionId, c.stat, c.skill, mag, trig, ok, c.wantMag, c.wantTriggers)
		}
	}
}
