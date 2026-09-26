package messaging

import "testing"

// TestCategoryLightIsTreatedAsTimeOfDay pins the spec's rule: light notices are
// prose-wrapped, skip every normalisation stage, and no verbosity tier drops
// them.
func TestCategoryLightIsTreatedAsTimeOfDay(t *testing.T) {
	if CategoryLight.String() != "light" {
		t.Fatalf("CategoryLight.String() = %q, want light", CategoryLight.String())
	}
	if shouldWrap(CategoryLight) != shouldWrap(CategoryTimeOfDay) {
		t.Error("CategoryLight must wrap exactly as CategoryTimeOfDay")
	}
	if skipStages(CategoryLight) != skipStages(CategoryTimeOfDay) {
		t.Error("CategoryLight must skip the same normalisation stages as CategoryTimeOfDay")
	}
	for _, v := range []Verbosity{VerbosityFull, VerbosityMedium, VerbosityLight} {
		if v.Suppresses(CategoryLight) {
			t.Errorf("verbosity %v suppresses CategoryLight; no tier may", v)
		}
	}
}
