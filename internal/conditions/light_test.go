package conditions

import (
	"math"
	"testing"

	"gopkg.in/yaml.v2"
)

const (
	testLanternId = 9701
	testGlowId    = 9702
)

func seedLightSpecs(t *testing.T) {
	t.Helper()
	t.Cleanup(SeedConditionsForTest(map[int]*ConditionSpec{
		testLanternId: {ConditionId: testLanternId, Name: "Test Lantern", TriggerCount: 1, RoundInterval: 1,
			Effects: map[EffectKind]EffectValue{EffectLightStrength: {Literal: 54}},
			Flags:   []Flag{Adjustable}},
		testGlowId: {ConditionId: testGlowId, Name: "Test Glow", TriggerCount: 4, RoundInterval: 1,
			Effects: map[EffectKind]EffectValue{EffectLightStrength: {UsesMagnitude: true}},
			Flags:   []Flag{Adjustable, Cancellable}},
	}))
}

func TestLightRecordStates(t *testing.T) {
	seedLightSpecs(t)
	bs := New()
	bs.AddCondition(testLanternId, true)
	rec := bs.LightSources()[0]
	spec := GetConditionSpec(testLanternId)

	if v, ok := rec.LightNow(spec); !ok || v != 54 {
		t.Fatalf("fresh lantern = (%v, %v), want (54, true)", v, ok)
	}
	rec.SetLightOutput(40)
	if v, ok := rec.LightNow(spec); !ok || v != 40 {
		t.Errorf("trimmed lantern = (%v, %v), want (40, true)", v, ok)
	}
	rec.SetLightOutput(math.Inf(-1))
	if _, ok := rec.LightNow(spec); ok {
		t.Error("a lantern trimmed to nothing still adds light")
	}
	rec.ResetLight()
	rec.Hooded = true
	if _, ok := rec.LightNow(spec); ok {
		t.Error("a hooded lantern still adds light")
	}
	rec.ResetLight()
	if v, ok := rec.LightNow(spec); !ok || v != 54 || rec.Hooded {
		t.Errorf("reset lantern = (%v, %v, hooded %v), want (54, true, false)", v, ok, rec.Hooded)
	}
}

// Fact 7: the worn-item refresh re-adds a held record. It must keep the trim.
func TestRefreshKeepsTrimAndHood(t *testing.T) {
	seedLightSpecs(t)
	bs := New()
	bs.AddCondition(testLanternId, true)
	rec := bs.LightSources()[0]
	rec.SetLightOutput(30)
	rec.Hooded = true
	bs.AddCondition(testLanternId, true)
	if rec.LightTrim != LightTrimmed || rec.LightOutput != 30 || !rec.Hooded {
		t.Errorf("refresh changed light state: trim %q output %v hooded %v", rec.LightTrim, rec.LightOutput, rec.Hooded)
	}
}

// A recast is a fresh source at full strength.
func TestFreshMagnitudeResetsTrim(t *testing.T) {
	seedLightSpecs(t)
	bs := New()
	bs.AddConditionMagnitude(testGlowId, 4, 90)
	rec := bs.LightSources()[0]
	rec.SetLightOutput(10)
	bs.AddConditionMagnitude(testGlowId, 4, 90)
	if v, ok := rec.LightNow(GetConditionSpec(testGlowId)); !ok || v != 90 {
		t.Errorf("recast glow = (%v, %v), want (90, true)", v, ok)
	}
}

func TestLightStateSurvivesASave(t *testing.T) {
	in := Condition{ConditionId: testLanternId, LightTrim: LightTrimmed, LightOutput: 41.5, Hooded: true}
	raw, err := yaml.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Condition
	if err := yaml.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.LightTrim != in.LightTrim || out.LightOutput != in.LightOutput || out.Hooded != in.Hooded {
		t.Errorf("round trip lost light state: %+v", out)
	}
}

// omitempty drops a zero LightOutput, but the "trimmed" state survives, so a
// lantern trimmed to exactly zero reloads as (0, true), not full strength.
func TestZeroTrimSurvivesASave(t *testing.T) {
	seedLightSpecs(t)
	in := Condition{ConditionId: testLanternId, TriggersLeft: 1, LightTrim: LightTrimmed, LightOutput: 0}
	raw, err := yaml.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Condition
	if err := yaml.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if v, ok := out.LightNow(GetConditionSpec(testLanternId)); !ok || v != 0 {
		t.Errorf("reloaded zero trim = (%v, %v), want (0, true)", v, ok)
	}
}

// The admin path: a magnitude light added with no magnitude sheds nothing.
func TestMagnitudeLightWithoutMagnitudeIsDark(t *testing.T) {
	seedLightSpecs(t)
	bs := New()
	bs.AddCondition(testGlowId, true)
	rec := bs.LightSources()[0]
	if v, ok := rec.LightNow(GetConditionSpec(testGlowId)); ok {
		t.Errorf("a magnitude light with no magnitude = (%v, true), want false", v)
	}
}

func TestEffectNeverAggregatesLight(t *testing.T) {
	seedLightSpecs(t)
	bs := New()
	bs.AddCondition(testLanternId, true)
	if v := bs.Effect(EffectLightStrength); v != 0 {
		t.Errorf("Effect(light_strength) = %v with a lantern held, want 0", v)
	}
}

func TestLightSourcesSkipsExpired(t *testing.T) {
	seedLightSpecs(t)
	bs := New()
	bs.AddCondition(testLanternId, true)
	bs.RemoveCondition(testLanternId)
	if n := len(bs.LightSources()); n != 0 {
		t.Errorf("LightSources after removal = %d records, want 0", n)
	}
}

func TestTrimmedOutputOutOfRange(t *testing.T) {
	seedLightSpecs(t)
	bs := New()
	bs.AddCondition(testLanternId, true)
	rec := bs.LightSources()[0]
	spec := GetConditionSpec(testLanternId)

	rec.LightTrim, rec.LightOutput = LightTrimmed, -3
	if v, ok := rec.LightNow(spec); ok {
		t.Errorf("negative trimmed output = (%v, true), want false", v)
	}
	rec.LightTrim, rec.LightOutput = LightTrimmed, 80
	if v, ok := rec.LightNow(spec); !ok || v != 54 {
		t.Errorf("trimmed output above max = (%v, %v), want (54, true)", v, ok)
	}
}

func TestLightSpecValidation(t *testing.T) {
	stackingTick := func(effects map[EffectKind]EffectValue) *ConditionSpec {
		return &ConditionSpec{ConditionId: 9705, Name: "Stack", TriggerRate: "1 round", TriggerCount: 4, RoundInterval: 1,
			Flags: []Flag{Stacking}, TickPool: "health", TickFromMagnitude: true, Effects: effects}
	}
	if err := stackingTick(nil).Validate(); err != nil {
		t.Fatalf("control: the stacking spec without light should validate, got %v", err)
	}
	if err := stackingTick(map[EffectKind]EffectValue{EffectLightStrength: {Literal: 5}}).Validate(); err == nil {
		t.Error("a stacking light source validated")
	}

	adjustableNoLight := &ConditionSpec{ConditionId: 9703, Name: "Bad", TriggerCount: 1, RoundInterval: 1, Flags: []Flag{Adjustable}}
	if err := adjustableNoLight.Validate(); err == nil {
		t.Error("an adjustable condition with no light_strength validated")
	}
	zeroLight := &ConditionSpec{ConditionId: 9704, Name: "Bad", TriggerCount: 1, RoundInterval: 1,
		Effects: map[EffectKind]EffectValue{EffectLightStrength: {Literal: 0}}}
	if err := zeroLight.Validate(); err == nil {
		t.Error("a light_strength of 0 validated")
	}
}
