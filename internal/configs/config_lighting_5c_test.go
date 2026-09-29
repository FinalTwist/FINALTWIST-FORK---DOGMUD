package configs

import "testing"

// A zero Balance is what a test binary sees; every 5c knob must default.
func TestLighting5cKnobDefaults(t *testing.T) {
	var b Balance
	b.validateLighting()
	checks := []struct {
		name      string
		got, want float64
	}{
		{"LightNightVisionSpellBase", float64(b.LightNightVisionSpellBase), 4},
		{"LightNightVisionSpellStatDivisor", float64(b.LightNightVisionSpellStatDivisor), 12.5},
		{"LightNightVisionSpellSkillDivisor", float64(b.LightNightVisionSpellSkillDivisor), 6.5},
		{"LightInfraSpellBase", float64(b.LightInfraSpellBase), 5},
		{"LightInfraSpellStatDivisor", float64(b.LightInfraSpellStatDivisor), 7},
		{"LightInfraSpellSkillDivisor", float64(b.LightInfraSpellSkillDivisor), 3},
		{"LightInfraReachCap", float64(b.LightInfraReachCap), 50},
		{"LightInfraPenaltyFloor", float64(b.LightInfraPenaltyFloor), 0.90},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// Out-of-range values revert rather than silently ship.
func TestLighting5cKnobRanges(t *testing.T) {
	b := Balance{LightInfraReachCap: 150, LightInfraPenaltyFloor: 1.5}
	b.validateLighting()
	if b.LightInfraReachCap != 50 || b.LightInfraPenaltyFloor != 0.90 {
		t.Errorf("cap %v floor %v, want 50 and 0.90", b.LightInfraReachCap, b.LightInfraPenaltyFloor)
	}
}

func TestLightingAccessorCarries5cKnobs(t *testing.T) {
	SetConfigForTest(t, GetConfig())
	l := GetLightingConfig()
	if l.InfraReachCap != 50 || l.InfraPenaltyFloor != 0.90 || l.DarkCap != 0.80 ||
		l.NightVisionSpellBase != 4 || l.InfraSpellSkillDivisor != 3 {
		t.Errorf("accessor = %+v", l)
	}
}
