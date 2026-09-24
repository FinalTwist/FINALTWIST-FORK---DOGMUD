package items

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// bandLabelAttackFixture gives Weak, Normal, Heavy, Critical and Miss one
// variant each, naming itself, so a test can assert WHICH POOL GetAttackMessage
// selected without depending on authored prose or on production randomness (a
// one-element pool has one possible pick).
func bandLabelAttackFixture() map[ItemSubType]*WeaponAttackMessageGroup {
	mk := func(label string) SkillTieredMessages {
		pool := MessageOptions{ItemMessage(label)}
		return SkillTieredMessages{Beginner: pool, Expert: pool, Master: pool}
	}
	opts := AttackTypes{}
	for intensity, label := range map[Intensity]string{
		Miss: "MISS", Weak: "WEAK", Normal: "NORMAL", Heavy: "HEAVY", Critical: "CRITICAL",
	} {
		opts[intensity] = AttackOptions{Together: TogetherMessages{
			ToAttacker: mk(label), ToDefender: mk(label), ToRoom: mk(label),
		}}
	}
	return map[ItemSubType]*WeaponAttackMessageGroup{
		Generic: {OptionId: Generic, Options: opts},
	}
}

// TestGetAttackMessageBandBoundaries pins the mapping from damage percentage to
// authored pool, at every boundary and on both sides of it.
//
// Until 2026-09-20 nothing in the repo tested this: the combat-messages golden
// walks Intensity values directly through GetPreAttackMessage and never calls
// GetAttackMessage, and the only test-file mentions of the function are
// comments about seeding its map. The cutoffs could have been retyped freely.
func TestGetAttackMessageBandBoundaries(t *testing.T) {
	restore := SeedAttackMessagesForTest(bandLabelAttackFixture())
	defer restore()

	cases := []struct {
		pct  int
		want string
	}{
		{0, "MISS"},
		{1, "WEAK"},
		{29, "WEAK"},
		{30, "NORMAL"},
		{74, "NORMAL"},
		{75, "HEAVY"},
		{100, "HEAVY"},
		{101, "CRITICAL"},
		{250, "CRITICAL"},
	}
	for _, tc := range cases {
		got := GetAttackMessage(Generic, tc.pct)
		if len(got.Together.ToAttacker.Beginner) != 1 {
			t.Fatalf("pct %d: fixture returned %d variants, want 1", tc.pct, len(got.Together.ToAttacker.Beginner))
		}
		if band := string(got.Together.ToAttacker.Beginner[0]); band != tc.want {
			t.Errorf("pct %d selected %s, want %s", tc.pct, band, tc.want)
		}
	}
}

// TestGetAttackMessageBandHonoursConfig proves the cutoffs are READ, not merely
// declared. A knob nothing reads is the exact defect this repo has hit before:
// a check that cannot fail, shipped green.
func TestGetAttackMessageBandHonoursConfig(t *testing.T) {
	restore := SeedAttackMessagesForTest(bandLabelAttackFixture())
	defer restore()

	c := configs.GetConfig()
	c.Balance.AttackBandNormalThresholdPct = 50
	c.Balance.AttackBandHeavyThresholdPct = 90
	configs.SetConfigForTest(t, c)

	cases := []struct {
		pct  int
		want string
	}{
		{49, "WEAK"}, // default 30 would have said NORMAL
		{50, "NORMAL"},
		{89, "NORMAL"}, // default 75 would have said HEAVY
		{90, "HEAVY"},
	}
	for _, tc := range cases {
		got := GetAttackMessage(Generic, tc.pct)
		if band := string(got.Together.ToAttacker.Beginner[0]); band != tc.want {
			t.Errorf("pct %d with cutoffs 50/90 selected %s, want %s", tc.pct, band, tc.want)
		}
	}
}
