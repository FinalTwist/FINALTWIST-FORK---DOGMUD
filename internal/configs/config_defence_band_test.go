package configs

import "testing"

// A Go test binary never loads config.yaml, so an absent key arrives as 0. A
// zero cutoff would band EVERY non-crit defensive win as Normal and silently
// delete the Weak band from every test in the repo -- the same reasoning that
// makes ContestFloor reject zero. Zero is therefore rewritten to the shipped
// default rather than honoured.
func TestDefenceBandNormalThreshold_ZeroIsRejected(t *testing.T) {
	b := Balance{DefenceBandNormalThreshold: 0}
	b.Validate()
	if b.DefenceBandNormalThreshold != 0.5 {
		t.Fatalf("zero must revert to the shipped default 0.5, got %v", b.DefenceBandNormalThreshold)
	}
}

func TestDefenceBandNormalThreshold_AuthoredValueSurvives(t *testing.T) {
	b := Balance{DefenceBandNormalThreshold: 1.25}
	b.Validate()
	if b.DefenceBandNormalThreshold != 1.25 {
		t.Fatalf("an authored in-range value must survive validation, got %v", b.DefenceBandNormalThreshold)
	}
}
