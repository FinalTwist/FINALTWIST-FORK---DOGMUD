package configs

import "testing"

// A Go test binary never loads config.yaml, so an absent key arrives as 0. Zero
// on either cutoff would collapse the bands: a zero Normal makes every landed
// hit read Normal or better and retires Weak entirely. Same reasoning as
// ContestFloor, which rejects zero for exactly this.
func TestAttackBandThresholds_ZeroIsRejected(t *testing.T) {
	b := Balance{AttackBandNormalThresholdPct: 0, AttackBandHeavyThresholdPct: 0}
	b.Validate()
	if b.AttackBandNormalThresholdPct != 30 || b.AttackBandHeavyThresholdPct != 75 {
		t.Fatalf("zero must revert to the shipped defaults 30/75, got %d/%d",
			b.AttackBandNormalThresholdPct, b.AttackBandHeavyThresholdPct)
	}
}

// An inverted pair reverts BOTH, never clamping one into the other: a half
// applied typo would ship a band layout nobody authored.
func TestAttackBandThresholds_InvertedPairRevertsBoth(t *testing.T) {
	b := Balance{AttackBandNormalThresholdPct: 80, AttackBandHeavyThresholdPct: 40}
	b.Validate()
	if b.AttackBandNormalThresholdPct != 30 || b.AttackBandHeavyThresholdPct != 75 {
		t.Fatalf("an inverted pair must revert both to 30/75, got %d/%d",
			b.AttackBandNormalThresholdPct, b.AttackBandHeavyThresholdPct)
	}
}

func TestAttackBandThresholds_AuthoredPairSurvives(t *testing.T) {
	b := Balance{AttackBandNormalThresholdPct: 25, AttackBandHeavyThresholdPct: 90}
	b.Validate()
	if b.AttackBandNormalThresholdPct != 25 || b.AttackBandHeavyThresholdPct != 90 {
		t.Fatalf("an authored in-range pair must survive validation, got %d/%d",
			b.AttackBandNormalThresholdPct, b.AttackBandHeavyThresholdPct)
	}
}

// 100 is the top of the non-crit range: attackMessagePct caps a non-crit swing
// at 100, so a Heavy cutoff of 100 means "only a perfectly average-or-better
// maximum roll reads Heavy". Legal, if extreme.
func TestAttackBandThresholds_HundredIsLegal(t *testing.T) {
	b := Balance{AttackBandNormalThresholdPct: 30, AttackBandHeavyThresholdPct: 100}
	b.Validate()
	if b.AttackBandHeavyThresholdPct != 100 {
		t.Fatalf("100 is the legal top of the non-crit range, got %d", b.AttackBandHeavyThresholdPct)
	}
}
