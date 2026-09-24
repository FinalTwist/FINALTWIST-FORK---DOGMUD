package messaging

import "testing"

func TestParseVerbosity(t *testing.T) {
	cases := []struct {
		in   string
		want Verbosity
	}{
		{"", VerbosityFull},
		{"full", VerbosityFull},
		{"FULL", VerbosityFull},
		{"medium", VerbosityMedium},
		{"light", VerbosityLight},
		{"garbage", VerbosityFull}, // unknown → safe default
	}
	for _, c := range cases {
		if got := ParseVerbosity(c.in); got != c.want {
			t.Errorf("ParseVerbosity(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestVerbosityString(t *testing.T) {
	if VerbosityFull.String() != "full" || VerbosityMedium.String() != "medium" || VerbosityLight.String() != "light" {
		t.Errorf("Verbosity String() mismatch: %q %q %q",
			VerbosityFull.String(), VerbosityMedium.String(), VerbosityLight.String())
	}
}

func TestVerbosityOneStepLower(t *testing.T) {
	if VerbosityFull.OneStepLower() != VerbosityMedium {
		t.Error("full should step to medium")
	}
	if VerbosityMedium.OneStepLower() != VerbosityLight {
		t.Error("medium should step to light")
	}
	if VerbosityLight.OneStepLower() != VerbosityLight {
		t.Error("light should stay light")
	}
}

func TestVerbositySuppresses(t *testing.T) {
	hitCats := []Category{CategoryHitMelee, CategoryHitBlunt, CategoryHitNaturalSharp,
		CategoryHitRanged, CategoryHitCaster, CategoryHitUnarmed}
	defenseCats := []Category{CategoryDodge, CategoryParry, CategoryBlock}
	neverSuppressed := []Category{CategoryDeath, CategoryKick, CategoryTrip, CategoryBash,
		CategorySubmission, CategorySystem, CategoryDefault, CategoryCombatSummary}

	for _, c := range hitCats {
		if VerbosityFull.Suppresses(c) {
			t.Errorf("full must not suppress %v", c)
		}
		if VerbosityMedium.Suppresses(c) {
			t.Errorf("medium must not suppress hit category %v", c)
		}
		if !VerbosityLight.Suppresses(c) {
			t.Errorf("light must suppress hit category %v", c)
		}
	}
	for _, c := range defenseCats {
		if VerbosityFull.Suppresses(c) {
			t.Errorf("full must not suppress %v", c)
		}
		if !VerbosityMedium.Suppresses(c) {
			t.Errorf("medium must suppress defense category %v", c)
		}
		if !VerbosityLight.Suppresses(c) {
			t.Errorf("light must suppress defense category %v", c)
		}
	}
	for _, c := range neverSuppressed {
		for _, v := range []Verbosity{VerbosityFull, VerbosityMedium, VerbosityLight} {
			if v.Suppresses(c) {
				t.Errorf("%v must never suppress %v", v, c)
			}
		}
	}
}

func TestCategoryCombatSummaryString(t *testing.T) {
	if CategoryCombatSummary.String() != "combat-summary" {
		t.Errorf("got %q", CategoryCombatSummary.String())
	}
}

// TestCategoryCombatBlindWarningSuppression pins the owner's option-C
// ruling (M4d PR 2 followup): the per-round "you can't see clearly"
// notice is delivered at Medium (the player still reads swing prose the
// notice explains) and suppressed at Light (a deliberate near-silence
// preference a per-round line should not override). Full never suppresses
// anything, so it is covered by TestVerbositySuppresses' general table
// shape and not repeated here.
func TestCategoryCombatBlindWarningSuppression(t *testing.T) {
	if VerbosityMedium.Suppresses(CategoryCombatBlindWarning) {
		t.Error("Medium must deliver the blind notice, not suppress it")
	}
	if !VerbosityLight.Suppresses(CategoryCombatBlindWarning) {
		t.Error("Light must suppress the blind notice")
	}
}
