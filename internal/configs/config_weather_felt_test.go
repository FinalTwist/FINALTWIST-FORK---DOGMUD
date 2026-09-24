package configs

import "testing"

// Weather emotes band Mild/Strong on felt intensity. Zero would make every
// indoor moment Strong in every test binary, since no test loads config.yaml.
func TestWeatherStrongFeltThreshold_ZeroIsRejected(t *testing.T) {
	b := Balance{WeatherStrongFeltThreshold: 0}
	b.Validate()
	if b.WeatherStrongFeltThreshold != 0.5 {
		t.Fatalf("zero must revert to the shipped default 0.5, got %v", b.WeatherStrongFeltThreshold)
	}
}

func TestWeatherStrongFeltThreshold_AuthoredValueSurvives(t *testing.T) {
	b := Balance{WeatherStrongFeltThreshold: 0.75}
	b.Validate()
	if b.WeatherStrongFeltThreshold != 0.75 {
		t.Fatalf("an authored in-range value must survive validation, got %v", b.WeatherStrongFeltThreshold)
	}
}
