package gametime

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func lightingForTest() configs.Lighting {
	return configs.Lighting{
		BlindBelow: 25, DimBelow: 50, ExitsAbove: 65,
		DoublingStep: 8, WorldLatitude: 46.5, EquinoxNoon: 70,
		Starlight: 10, MoonsFull: 35,
		MoonWeightSwiftmoon: 4, MoonWeightWanderer: 1, MoonWeightEye: 0.5,
	}
}

// Declination is zero at the equinoxes and hits the axial tilt at the
// solstices. Everything else in the model rests on this.
func TestDeclinationAtSolsticesAndEquinoxes(t *testing.T) {
	if d := declinationDegrees(356); math.Abs(d+23.44) > 0.2 {
		t.Errorf("midwinter declination %v, want about -23.44", d)
	}
	if d := declinationDegrees(172); math.Abs(d-23.44) > 0.2 {
		t.Errorf("midsummer declination %v, want about 23.44", d)
	}
	if d := declinationDegrees(81); math.Abs(d) > 0.5 {
		t.Errorf("equinox declination %v, want about 0", d)
	}
}

// At 46.5 degrees, real geometry gives 8h23m of night at midsummer and 15h37m
// at midwinter. Seattle at 47.6 publishes 15h59m of daylight; the small
// difference is refraction and the solar disc, both deliberately omitted.
func TestNightHoursAtLatitude46Point5(t *testing.T) {
	if n := NightHoursAt(46.5, 172); math.Abs(n-8.374) > 0.02 {
		t.Errorf("midsummer night %v, want about 8.374", n)
	}
	if n := NightHoursAt(46.5, 356); math.Abs(n-15.626) > 0.02 {
		t.Errorf("midwinter night %v, want about 15.626", n)
	}
	if n := NightHoursAt(46.5, 81); math.Abs(n-12) > 0.05 {
		t.Errorf("equinox night %v, want about 12", n)
	}
}

func TestNightHoursAtEquatorIsTwelveAllYear(t *testing.T) {
	for _, doy := range []int{1, 81, 172, 356} {
		if n := NightHoursAt(0, doy); math.Abs(n-12) > 1e-6 {
			t.Errorf("day %d at the equator: night %v, want 12", doy, n)
		}
	}
}

// Beyond the polar circles the half-day angle has no solution. It must clamp
// to a full day or a full night rather than returning NaN.
func TestPolarLatitudesClampInsteadOfNaN(t *testing.T) {
	if n := NightHoursAt(80, 172); n != 0 {
		t.Errorf("polar midsummer night %v, want 0", n)
	}
	if n := NightHoursAt(80, 356); n != 24 {
		t.Errorf("polar midwinter night %v, want 24", n)
	}
}

// The calibration anchor: equinox noon reads exactly EquinoxNoon, because
// declination is zero there and sin(altitude) is exactly cos(latitude).
func TestEquinoxNoonHitsTheCalibrationExactly(t *testing.T) {
	cfg := lightingForTest()
	if got := SunLight(cfg, 81, 12); math.Abs(got-70) > 0.25 {
		t.Errorf("equinox noon %v, want 70", got)
	}
}

func TestSolsticeNoonsAreDerivedNotInvented(t *testing.T) {
	cfg := lightingForTest()
	if got := SunLight(cfg, 172, 12); math.Abs(got-73) > 0.6 {
		t.Errorf("midsummer noon %v, want about 73", got)
	}
	if got := SunLight(cfg, 356, 12); math.Abs(got-62) > 0.6 {
		t.Errorf("midwinter noon %v, want about 62", got)
	}
}

// 🔑 The sun is ABSENT below the horizon, not zero. A zero term would combine
// with moonlight and make a moonlit night brighter for having a sun that has
// set.
func TestSunIsAbsentBelowTheHorizon(t *testing.T) {
	cfg := lightingForTest()
	if got := SunLight(cfg, 356, 0); !math.IsInf(got, -1) {
		t.Errorf("midwinter midnight sun %v, want -Inf", got)
	}
}

// Natural daylight must never reach the dazzle edge of 75 at this calibration.
// If this fails, the spec's claim that dazzle is purely a play mechanic is
// broken and plan 5's design rests on a false premise.
func TestNaturalDaylightNeverDazzles(t *testing.T) {
	cfg := lightingForTest()
	for doy := 1; doy <= 365; doy++ {
		for h := 0.0; h < 24; h += 0.25 {
			if v := SunLight(cfg, doy, h); v >= 75 {
				t.Fatalf("day %d hour %v reached %v, at or above the dazzle edge", doy, h, v)
			}
		}
	}
}

func TestMoonLightHitsBothAnchors(t *testing.T) {
	cfg := lightingForTest()
	if got := MoonLight(cfg, 0, 0, 0); math.Abs(got-10) > 1e-6 {
		t.Errorf("all new %v, want 10", got)
	}
	if got := MoonLight(cfg, 1, 1, 1); math.Abs(got-35) > 1e-6 {
		t.Errorf("all full %v, want 35", got)
	}
}

func TestMoonLightIsMonotonic(t *testing.T) {
	cfg := lightingForTest()
	prev := math.Inf(-1)
	for f := 0.0; f <= 1.0; f += 0.05 {
		got := MoonLight(cfg, f, f, f)
		if got < prev {
			t.Fatalf("moonlight fell from %v to %v at fullness %v", prev, got, f)
		}
		prev = got
	}
}

// Swiftmoon carries four times the Wanderer's weight, so it must move the sky
// further on its own.
func TestSwiftmoonOutweighsTheWanderer(t *testing.T) {
	cfg := lightingForTest()
	swift := MoonLight(cfg, 1, 0, 0)
	wander := MoonLight(cfg, 0, 1, 0)
	if swift <= wander {
		t.Errorf("Swiftmoon %v did not outweigh the Wanderer %v", swift, wander)
	}
}

func TestCelestialLightIsMemoisedPerRound(t *testing.T) {
	c := configs.GetConfig()
	c.Timing.RoundsPerDay = 900
	c.Timing.NightHours = 8
	c.Timing.RoundSeconds = 4
	c.Timing.Validate()
	configs.SetConfigForTest(t, c)

	original := util.GetRoundCount()
	t.Cleanup(func() { util.SetRoundCount(original) })

	util.SetRoundCount(1000)
	first := CelestialLight()
	second := CelestialLight()
	if first != second {
		t.Fatalf("same round returned %v then %v", first, second)
	}

	util.SetRoundCount(1000 + 450) // twelve hours later
	if third := CelestialLight(); third == first {
		t.Fatalf("half a day later still returned %v; the memo is not keyed on the round", third)
	}
}
