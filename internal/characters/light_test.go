package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
)

func TestEmitsLightReadsLightRecords(t *testing.T) {
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		9711: {ConditionId: 9711, Name: "Test Torch", TriggerCount: 1, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {Literal: 56}}},
	}))
	c := New()
	if c.EmitsLight() {
		t.Fatal("a character with no light emits light")
	}
	if err := c.AddCondition(9711, true); err != nil {
		t.Fatal(err)
	}
	if !c.EmitsLight() {
		t.Fatal("a torch bearer does not emit light")
	}
	if got := c.LightTerms(); len(got) != 1 || got[0] != 56 {
		t.Errorf("LightTerms = %v, want [56]", got)
	}
	c.Conditions.LightSources()[0].Hooded = true
	if c.EmitsLight() {
		t.Error("a hooded source still emits light")
	}
}
