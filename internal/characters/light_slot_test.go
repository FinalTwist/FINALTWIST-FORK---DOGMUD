package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/items"
)

const (
	slotLanternItem = 999940
	slotGlovesItem  = 999941
	slotLanternCond = 9741
)

func seedLightSlot(t *testing.T) *Character {
	t.Helper()
	// Wear dereferences the wearer's species (see seedMediumSpecies).
	t.Cleanup(seedMediumSpecies())
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		slotLanternCond: {ConditionId: slotLanternCond, Name: "Test Lantern", Secret: true, TriggerCount: 1, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {Literal: 54}},
			Flags:   []conditions.Flag{conditions.Adjustable}},
	}))
	t.Cleanup(items.SeedItemsForTest(map[int]*items.ItemSpec{
		slotLanternItem: {ItemId: slotLanternItem, Name: "test hooded lantern", Type: items.Light, Subtype: items.Wearable,
			WornConditionIds: []int{slotLanternCond}},
		slotGlovesItem: {ItemId: slotGlovesItem, Name: "test gloves", Type: items.Gloves, Subtype: items.Wearable},
	}))
	c := New()
	c.Stats.Strength.ValueAdj = 100
	return c
}

func TestLightSlotWearsALight(t *testing.T) {
	c := seedLightSlot(t)
	if _, ok, why := c.Wear(items.New(slotLanternItem)); !ok {
		t.Fatalf("could not wear a light: %s", why)
	}
	if c.Equipment.Light.ItemId != slotLanternItem {
		t.Fatalf("light slot holds %d", c.Equipment.Light.ItemId)
	}
	if !c.EmitsLight() {
		t.Error("equipping a light did not turn it on")
	}
}

// Swapping gloves must not reset a lantern still worn (fact 7).
func TestAnotherSlotKeepsTheTrim(t *testing.T) {
	c := seedLightSlot(t)
	c.Wear(items.New(slotLanternItem))
	rec := c.Conditions.LightSources()[0]
	rec.SetLightOutput(30)
	rec.Hooded = true
	c.Wear(items.New(slotGlovesItem))
	if rec.LightTrim != conditions.LightTrimmed || !rec.Hooded {
		t.Errorf("wearing gloves reset the lantern: trim %q hooded %v", rec.LightTrim, rec.Hooded)
	}
}

// Re-equipping the light is its fresh start, even when the replaced item
// shared its condition and the refresh kept the record.
func TestReEquippingTheLightResetsIt(t *testing.T) {
	c := seedLightSlot(t)
	c.Wear(items.New(slotLanternItem))
	rec := c.Conditions.LightSources()[0]
	rec.SetLightOutput(30)
	rec.Hooded = true
	c.Wear(items.New(slotLanternItem))
	now := c.Conditions.LightSources()[0]
	if now.LightTrim != conditions.LightFull || now.Hooded {
		t.Errorf("re-equipped lantern: trim %q hooded %v, want full and open", now.LightTrim, now.Hooded)
	}
}

// Removing the light empties the slot and puts the light out.
func TestRemovingTheLightSlotPutsItOut(t *testing.T) {
	c := seedLightSlot(t)
	lantern := items.New(slotLanternItem)
	if _, ok, why := c.Wear(lantern); !ok {
		t.Fatalf("could not wear a light: %s", why)
	}
	if !c.RemoveFromBody(c.Equipment.Light) {
		t.Fatal("could not remove the worn light")
	}
	if c.Equipment.Light.ItemId != 0 {
		t.Errorf("light slot still holds %d", c.Equipment.Light.ItemId)
	}
	if c.EmitsLight() {
		t.Error("removing the light left it burning")
	}
}

// Taking off one of two worn items that grant the same condition keeps the
// condition: the refresh counts what is still worn, and once subtracted the
// removed item a second time, which dropped the record here.
func TestRemovingOneOfTwoSharersKeepsTheCondition(t *testing.T) {
	c := seedLightSlot(t)
	// SeedItemsForTest replaces the whole store, so the lantern is re-seeded.
	t.Cleanup(items.SeedItemsForTest(map[int]*items.ItemSpec{
		slotLanternItem: {ItemId: slotLanternItem, Name: "test hooded lantern", Type: items.Light, Subtype: items.Wearable,
			WornConditionIds: []int{slotLanternCond}},
		slotGlovesItem: {ItemId: slotGlovesItem, Name: "test glowing gloves", Type: items.Gloves, Subtype: items.Wearable,
			WornConditionIds: []int{slotLanternCond}},
	}))
	c.Wear(items.New(slotLanternItem))
	c.Wear(items.New(slotGlovesItem))
	if !c.RemoveFromBody(c.Equipment.Gloves) {
		t.Fatal("could not remove the gloves")
	}
	if len(c.Conditions.GetConditions(slotLanternCond)) == 0 || !c.EmitsLight() {
		t.Error("removing the gloves put out the lantern still worn")
	}
}
