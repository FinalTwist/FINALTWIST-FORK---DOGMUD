package configs

import (
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestBaubleDefaults(t *testing.T) {
	b := &Balance{}
	b.validateBaubles()

	if b.BaublesEnabled || b.BaubleSearchChancePct != 1 || b.BaubleSkillMaxBonus != 1 || b.BaubleRollsPerWindow != 2 || b.BaubleFeatureWindowMinutes != 60 || b.BaubleUntakenHours != 24 || b.BaubleWindowMinutes != 60 || b.BaubleWindowPerPlayer || b.BaubleRevealSeconds != 3 {
		t.Fatalf("roll defaults: %+v", b)
	}
	if b.BaubleTierWeightCheap != 70 || b.BaubleTierWeightAverage != 25 || b.BaubleTierWeightRare != 5 {
		t.Fatal("tier weight defaults")
	}
	got := []ConfigInt{b.BaubleCheapMinValue, b.BaubleCheapMaxValue, b.BaubleAverageMinValue, b.BaubleAverageMaxValue, b.BaubleRareMinValue, b.BaubleRareMaxValue}
	want := []ConfigInt{1, 6, 10, 15, 40, 200}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ladder defaults: %v", got)
	}
	if !reflect.DeepEqual([]string(b.BaubleBuyerCraftSupports), []string{`general`, `jewelcrafting`}) {
		t.Fatalf("buyers: %v", b.BaubleBuyerCraftSupports)
	}
}

func TestBaubleChanceAndRevealAreCapped(t *testing.T) {
	b := &Balance{BaubleSearchChancePct: 250, BaubleRevealSeconds: 600}
	b.validateBaubles()
	if b.BaubleSearchChancePct != 100 {
		t.Fatal("a chance over 100% is 100%")
	}
	if b.BaubleRevealSeconds != 30 {
		t.Fatal("a find is never held back more than 30 seconds by the pacing knob")
	}
}

func TestBaubleOneZeroWeightIsHonoured(t *testing.T) {
	b := &Balance{BaubleTierWeightCheap: 80, BaubleTierWeightAverage: 20}
	b.validateBaubles()
	if b.BaubleTierWeightRare != 0 || b.BaubleTierWeightCheap != 80 {
		t.Fatal("an explicit zero switches a tier off")
	}
	b = &Balance{BaubleTierWeightRare: -3}
	b.validateBaubles()
	if b.BaubleTierWeightRare != 5 {
		t.Fatal("a negative weight takes its default")
	}
}

func TestBaubleBrokenLadderResetsWhole(t *testing.T) {
	cases := map[string]Balance{
		"overlapping tiers": {BaubleCheapMinValue: 1, BaubleCheapMaxValue: 12, BaubleAverageMinValue: 10, BaubleAverageMaxValue: 15, BaubleRareMinValue: 40, BaubleRareMaxValue: 200},
		"inverted tier":     {BaubleCheapMinValue: 1, BaubleCheapMaxValue: 6, BaubleAverageMinValue: 15, BaubleAverageMaxValue: 10, BaubleRareMinValue: 40, BaubleRareMaxValue: 200},
		"zero minimum":      {BaubleCheapMinValue: 0, BaubleCheapMaxValue: 6, BaubleAverageMinValue: 10, BaubleAverageMaxValue: 15, BaubleRareMinValue: 40, BaubleRareMaxValue: 200},
	}
	for name, b := range cases {
		b := b
		b.validateBaubles()
		if b.BaubleCheapMaxValue != 6 || b.BaubleAverageMinValue != 10 || b.BaubleRareMaxValue != 200 {
			t.Fatalf("%s: ladder not reset: %d-%d %d-%d %d-%d", name,
				b.BaubleCheapMinValue, b.BaubleCheapMaxValue, b.BaubleAverageMinValue, b.BaubleAverageMaxValue, b.BaubleRareMinValue, b.BaubleRareMaxValue)
		}
	}

	custom := Balance{BaubleCheapMinValue: 2, BaubleCheapMaxValue: 8, BaubleAverageMinValue: 12, BaubleAverageMaxValue: 20, BaubleRareMinValue: 50, BaubleRareMaxValue: 300}
	custom.validateBaubles()
	if custom.BaubleCheapMinValue != 2 || custom.BaubleRareMaxValue != 300 {
		t.Fatal("a valid custom ladder is kept")
	}
}

// The shipped config.yaml and the Go defaults must agree, or a server started
// without the block behaves differently from one started with it.
func TestBaubleShippedConfigMatchesDefaults(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "_datafiles", "config.yaml"))
	if err != nil {
		t.Fatalf("read shipped config: %v", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("decode shipped config: %v", err)
	}
	shipped := cfg.Balance
	var defaults Balance
	defaults.validateBaubles()

	if shipped.BaublesEnabled != defaults.BaublesEnabled ||
		shipped.BaubleSearchChancePct != defaults.BaubleSearchChancePct ||
		shipped.BaubleSkillMaxBonus != defaults.BaubleSkillMaxBonus ||
		shipped.BaubleFeatureWindowMinutes != defaults.BaubleFeatureWindowMinutes ||
		shipped.BaubleUntakenHours != defaults.BaubleUntakenHours ||
		!reflect.DeepEqual(shipped.BaubleBiomeChancePct, defaults.BaubleBiomeChancePct) ||
		shipped.BaubleRollsPerWindow != defaults.BaubleRollsPerWindow ||
		shipped.BaubleWindowMinutes != defaults.BaubleWindowMinutes ||
		shipped.BaubleWindowPerPlayer != defaults.BaubleWindowPerPlayer ||
		shipped.BaubleRevealSeconds != defaults.BaubleRevealSeconds ||
		shipped.BaublePickpocketChancePct != defaults.BaublePickpocketChancePct ||
		shipped.BaublePickpocketMaxWeight != defaults.BaublePickpocketMaxWeight ||
		shipped.BaublePickpocketGraceSecs != defaults.BaublePickpocketGraceSecs ||
		shipped.BaubleTierWeightCheap != defaults.BaubleTierWeightCheap ||
		shipped.BaubleTierWeightAverage != defaults.BaubleTierWeightAverage ||
		shipped.BaubleTierWeightRare != defaults.BaubleTierWeightRare ||
		shipped.BaubleCheapMinValue != defaults.BaubleCheapMinValue ||
		shipped.BaubleCheapMaxValue != defaults.BaubleCheapMaxValue ||
		shipped.BaubleAverageMinValue != defaults.BaubleAverageMinValue ||
		shipped.BaubleAverageMaxValue != defaults.BaubleAverageMaxValue ||
		shipped.BaubleRareMinValue != defaults.BaubleRareMinValue ||
		shipped.BaubleRareMaxValue != defaults.BaubleRareMaxValue ||
		!reflect.DeepEqual(shipped.BaubleBuyerCraftSupports, defaults.BaubleBuyerCraftSupports) {
		t.Errorf("shipped bauble config differs from the Go defaults:\n shipped:  %+v\n defaults: %+v", shipped, defaults)
	}
}

// The owner's anchors: buildings 5%, streets about 2%, wilderness 0.25%.
func TestBaubleBiomeChanceDefaults(t *testing.T) {
	b := &Balance{}
	b.validateBaubles()
	want := map[string]float64{
		`interior`: 5, `city_thoroughfare`: 2, `city_backstreet`: 2.5,
		`forest`: 0.25, `plains`: 0.25, `mountains`: 0.25, `water`: 0, `ether`: 0,
	}
	for biome, pct := range want {
		if got, ok := b.BaubleBiomeChancePct[biome]; !ok || got != pct {
			t.Errorf("%s: got %v (listed %v), want %v", biome, got, ok, pct)
		}
	}
	// Every listed chance is a sane percentage, and built places beat the wild.
	for biome, pct := range b.BaubleBiomeChancePct {
		if pct < 0 || pct > 100 {
			t.Errorf("%s: %v is not a percentage", biome, pct)
		}
	}
	if b.BaubleBiomeChancePct[`interior`] <= b.BaubleBiomeChancePct[`city_thoroughfare`] ||
		b.BaubleBiomeChancePct[`city_thoroughfare`] <= b.BaubleBiomeChancePct[`road`] ||
		b.BaubleBiomeChancePct[`road`] <= b.BaubleBiomeChancePct[`forest`] {
		t.Error("buildings > streets > roads > wilderness")
	}
}

func TestBaubleBiomeTableAsGiven(t *testing.T) {
	b := &Balance{BaubleBiomeChancePct: map[string]float64{` Interior `: 8, `forest`: -1, `sewer`: 250}}
	b.validateBaubles()
	if len(b.BaubleBiomeChancePct) != 3 || b.BaubleBiomeChancePct[`interior`] != 8 ||
		b.BaubleBiomeChancePct[`forest`] != 0 || b.BaubleBiomeChancePct[`sewer`] != 100 {
		t.Fatalf("a given table is used as given, cleaned: %v", b.BaubleBiomeChancePct)
	}
	if _, listed := b.BaubleBiomeChancePct[`plains`]; listed {
		t.Fatal("unlisted biomes are not filled in; they use BaubleSearchChancePct")
	}
}

func TestBaubleSkillBonusBounds(t *testing.T) {
	b := &Balance{BaubleSkillMaxBonus: 50}
	b.validateBaubles()
	if b.BaubleSkillMaxBonus != 10 {
		t.Fatalf("skill bonus capped: %v", b.BaubleSkillMaxBonus)
	}
}

// The pickpocket pause ships at its defaults, and the bounds hold.
func TestStealPocketPauseShippedAndBounded(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "_datafiles", "config.yaml"))
	if err != nil {
		t.Fatalf("read shipped config: %v", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("decode shipped config: %v", err)
	}
	b := cfg.Balance
	if b.StealPocketSeconds != defaultStealPocketSeconds || b.StealPocketMinSeconds != defaultStealPocketMinSeconds ||
		b.StealPocketMaxSeconds != defaultStealPocketMaxSeconds {
		t.Fatalf("shipped pause %v/%v/%v differs from the defaults", b.StealPocketSeconds, b.StealPocketMinSeconds, b.StealPocketMaxSeconds)
	}
	var v Balance
	v.StealPocketMinSeconds, v.StealPocketMaxSeconds = 9, 4
	v.validateCombatSteal()
	if v.StealPocketSeconds != defaultStealPocketSeconds || v.StealPocketMinSeconds != 4 || v.StealPocketMaxSeconds != 4 {
		t.Fatalf("absent is the default, and min never exceeds max: %+v", []ConfigFloat{v.StealPocketSeconds, v.StealPocketMinSeconds, v.StealPocketMaxSeconds})
	}

	var c Balance
	c.BaublePickpocketChancePct = -7
	c.validateBaubles()
	if c.BaublePickpocketChancePct != -1 {
		t.Fatal("a negative chance is never")
	}
	var d Balance
	d.validateBaubles()
	if d.BaublePickpocketChancePct != 50 || d.BaublePickpocketMaxWeight != 1.0 {
		t.Fatal("absent: 50 percent, one pound")
	}
}

// A NaN from YAML `.nan` never reaches a bauble chance or weight.
func TestBaubleFloatsRefuseNaN(t *testing.T) {
	nan := ConfigFloat(math.NaN())
	b := Balance{BaubleSearchChancePct: nan, BaubleSkillMaxBonus: nan, BaublePickpocketChancePct: nan,
		BaublePickpocketMaxWeight: nan, BaublePickpocketGraceSecs: nan,
		BaubleBiomeChancePct: map[string]float64{`interior`: math.NaN()}}
	b.validateBaubles()
	b.StealPocketSeconds, b.StealPocketMinSeconds, b.StealPocketMaxSeconds = nan, nan, nan
	b.validateCombatSteal()
	for name, v := range map[string]float64{`search`: float64(b.BaubleSearchChancePct), `skill`: float64(b.BaubleSkillMaxBonus),
		`pickpocket`: float64(b.BaublePickpocketChancePct), `weight`: float64(b.BaublePickpocketMaxWeight),
		`grace`: float64(b.BaublePickpocketGraceSecs), `biome`: b.BaubleBiomeChancePct[`interior`],
		`pause`: float64(b.StealPocketSeconds), `min`: float64(b.StealPocketMinSeconds), `max`: float64(b.StealPocketMaxSeconds)} {
		if math.IsNaN(v) {
			t.Errorf("%s is NaN after validation", name)
		}
	}
}
