package items

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/configs"
)

// TestDefenseMessageRenderHonoursDefenceBandNormalThresholdKnob proves
// RenderDefenseMessage actually READS Balance.DefenceBandNormalThreshold
// rather than the hardcoded 0.5 literal it replaces. The two validation tests
// in internal/configs prove the knob validates correctly; they do not prove
// anything downstream consults it. A knob nothing reads is a defect this repo
// has shipped before.
//
// With the cutoff raised to 1.5, a margin of 0.6 -- which would have cleared
// today's 0.5 cutoff and read Normal -- must now read Weak, and a margin of
// 1.6 must clear the raised cutoff and read Normal.
func TestDefenseMessageRenderHonoursDefenceBandNormalThresholdKnob(t *testing.T) {
	restore := SeedDefenseMessagesForTest(map[DefencePool]*DefenseMessageGroup{DefencePoolFor(combatvocab.DefenceQuell): validDefenseMessageGroup()})
	defer restore()

	c := configs.GetConfig()
	c.Balance.DefenceBandNormalThreshold = 1.5
	configs.SetConfigForTest(t, c)

	tests := []struct {
		name          string
		margin        float64
		wantIntensity string
	}{
		{"below_raised_cutoff_is_weak", 0.6, "weak"},
		{"above_raised_cutoff_is_normal", 1.6, "normal"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			triad := RenderDefenseMessage(DefencePoolFor(combatvocab.DefenceQuell), false, tc.margin, map[TokenName]string{}, 3)
			want := tc.wantIntensity + "-def-3"
			if string(triad.ToDefender) != want {
				t.Fatalf("defender = %q, want %q", triad.ToDefender, want)
			}
		})
	}
}
