package hooks

import (
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
// light (9731) and a plain condition (9732) until the test ends.
func seedTestGlowCondition(t *testing.T) {
	t.Helper()
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		testGlowConditionId: {ConditionId: testGlowConditionId, Name: "Test Glow", TriggerCount: 4, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {UsesMagnitude: true}},
			Flags:   []conditions.Flag{conditions.Adjustable}},
		9732: {ConditionId: 9732, Name: "Test Plain", TriggerCount: 4, RoundInterval: 1},
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
		mag, trig, ok := lightSpellApplication(spell, caster, testGlowConditionId)
		if !ok || mag != c.wantStrength || trig != c.wantTriggers {
			t.Errorf("stat %d skill %d: (%v, %d, %v), want (%v, %d, true)", c.stat, c.skill, mag, trig, ok, c.wantStrength, c.wantTriggers)
		}
	}
	if _, _, ok := lightSpellApplication(spell, characters.New(), 9732); ok {
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
	mag, trig, ok := lightSpellApplication(spell, caster, testGlowConditionId)
	if !ok || mag != 20 || trig != 4 {
		t.Errorf("LightSpellStrengthBase 10, stat 100 skill 0: (%v, %d, %v), want (20, 4, true)", mag, trig, ok)
	}
}
