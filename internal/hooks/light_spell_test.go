package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

func TestLightSpellScalesFromStatAndSkill(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.Validate()
	configs.SetConfigForTest(t, cfg)
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		9731: {ConditionId: 9731, Name: "Test Glow", TriggerCount: 4, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {UsesMagnitude: true}},
			Flags:   []conditions.Flag{conditions.Adjustable}},
		9732: {ConditionId: 9732, Name: "Test Plain", TriggerCount: 4, RoundInterval: 1},
	}))
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
		mag, trig, ok := lightSpellApplication(spell, caster, 9731)
		if !ok || mag != c.wantStrength || trig != c.wantTriggers {
			t.Errorf("stat %d skill %d: (%v, %d, %v), want (%v, %d, true)", c.stat, c.skill, mag, trig, ok, c.wantStrength, c.wantTriggers)
		}
	}
	if _, _, ok := lightSpellApplication(spell, characters.New(), 9732); ok {
		t.Error("a non-light condition was treated as a light spell")
	}
}
