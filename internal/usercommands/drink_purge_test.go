package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// TestApplyPurgeEffects pins the three things a purging draught is supposed to
// do and, before this change, did none of. Condition 70 -- the only thing the item
// declared -- carries a flavour line and no statmods, and there is no condition
// scripting layer, so the draught was inert: it charged toxicity and delivered
// nothing. Condition 76, the weakness it was designed to leave behind, was authored
// in full and referenced by nothing at all.
func TestApplyPurgeEffects(t *testing.T) {
	cleanup := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		61: {ConditionId: 61, Name: "Ironhide Brew", TriggerCount: 400, RoundInterval: 1},
		76: {ConditionId: 76, Name: "Purging Weakness", TriggerCount: 50, RoundInterval: 1},
	})
	defer cleanup()

	c := characters.New()
	c.Stats.Vitality.Base = 300
	c.Stats.Vitality.Recalculate()
	u := &users.UserRecord{UserId: 7104, Character: c}

	if err := c.AddConditionScaled(61, 1.0); err != nil {
		t.Fatalf("setup: AddConditionScaled(61) = %v", err)
	}
	c.Toxicity = 40
	events.DrainQueuedConditionsForTest(u.UserId) // start from a clean queue

	if !c.HasCondition(61) {
		t.Fatalf("setup: expected the potion condition to be present before the purge")
	}

	applyPurgeEffects(u)

	// RemoveCondition only marks TriggersLeft as expired; the map entry
	// HasCondition checks isn't evicted until the next round's Prune() sweep
	// (see internal/conditions/conditions.go RemoveCondition/Prune, and the
	// same pattern pinned by internal/hooks/pinnacle_ambient_smart_test.go).
	// Prune here to observe the post-sweep state a real drinker would see a
	// moment later.
	c.Conditions.Prune()

	if c.HasCondition(61) {
		t.Errorf("potion condition 61 survived the purge; it must be stripped")
	}
	if c.Toxicity != 0 {
		t.Errorf("Toxicity = %v after the purge, want 0", c.Toxicity)
	}

	// The weakness is QUEUED, not applied in place. Adding it through
	// Character.AddConditionScaled applied it silently: the drinker took a
	// fifty-round stat penalty and read nothing about it. Condition_ApplyConditions is
	// what narrates the start, and only the event reaches it.
	queued := events.DrainQueuedConditionsForTest(u.UserId)
	var weakness *events.Condition
	for i := range queued {
		if queued[i].ConditionId == 76 {
			weakness = &queued[i]
		}
	}
	if weakness == nil {
		t.Fatalf("purging weakness (condition 76) was not queued; the purge must cost something, and it must say so. queued: %+v", queued)
	}
	if weakness.DurationMult != 1.0 {
		t.Errorf("queued condition 76 DurationMult = %v, want 1.0 (the authored duration)", weakness.DurationMult)
	}
}

// TestDetoxItemsBypassTheToxicityGate pins that BOTH detox items skip the
// "would this exceed your tolerance?" pre-check. Gating a detox behind low
// toxicity makes the cure unavailable to exactly the players who need it, and
// at the current tuning the Purging Draught (34) sits above a fresh
// character's whole tolerance (33.3), so without this it could be bought and
// never drunk.
func TestDetoxItemsBypassTheToxicityGate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		itemId int
		want   bool
	}{
		{"Ysolde's Purge", ysoldesPurgeItemId, true},
		{"Purging Draught", purgingDraughtItemId, true},
		{"an ordinary potion", 30036, false},
	} {
		if got := bypassesToxicityGate(tc.itemId); got != tc.want {
			t.Errorf("%s: bypassesToxicityGate(%d) = %v, want %v",
				tc.name, tc.itemId, got, tc.want)
		}
	}
}
