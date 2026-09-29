package mobs

import "testing"

// Every live mob held 0 action points (movement parity 4b, fact V15). A spawn
// now starts full and settled, so charging a step does not freeze it.
func TestSpawnStartsWithFullActionPoints(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	m := NewMobByIdFresh(MobId(1), 4242)
	if m == nil {
		t.Fatal("spawn returned nil")
	}
	defer DestroyInstance(m.InstanceId)

	if m.Character.ActionPointsMax.Value < 50 {
		t.Fatalf("fixture premise: ActionPointsMax = %d, Validate floors it at 50", m.Character.ActionPointsMax.Value)
	}
	if m.Character.ActionPoints != m.Character.ActionPointsMax.Value {
		t.Fatalf("spawn ActionPoints = %d, want full %d", m.Character.ActionPoints, m.Character.ActionPointsMax.Value)
	}
	if !m.Character.ActionPointsSettled {
		t.Fatal("spawn must stamp the settlement so the first charge does not refill twice")
	}
}
