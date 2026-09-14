package configs

import "testing"

// Slice 1b (2026-09-14): a bleed is a stack per landed hit. These knobs set
// each move's stack length and per-round amount. See the "BLEED STACKS" block
// in config.yaml for the equilibrium they are tuned to.

var bleedStackKeys = []string{
	"RakeBleedRounds", "RakeBleedStrengthDivisor", "RakeBleedMin",
	"MaulBleedRounds", "MaulBleedStrengthDivisor", "MaulBleedMin",
	"HamstringBleedRounds", "HamstringBleedStrengthDivisor", "HamstringBleedMin",
	"DrainBleedRounds", "DrainBleedStrengthDivisor", "DrainBleedMin",
	"ThrottleBleedRounds", "ThrottleBleedStrengthDivisor", "ThrottleBleedMin",
}

// An absent key reads 0 and must take the default: a zero stack length never
// ticks, a zero divisor divides by zero, a zero floor lets a stack tick for
// nothing. The defaults equal the shipped values, so test binaries (which
// never load config.yaml) see the shipped tuning.
func TestBleedStackKnobs_AbsentKeysTakeTheDefaults(t *testing.T) {
	b := Balance{}
	b.Validate()
	cases := []struct {
		name      string
		got, want ConfigInt
	}{
		{"RakeBleedRounds", b.RakeBleedRounds, 10},
		{"RakeBleedStrengthDivisor", b.RakeBleedStrengthDivisor, 50},
		{"RakeBleedMin", b.RakeBleedMin, 1},
		{"MaulBleedRounds", b.MaulBleedRounds, 12},
		{"MaulBleedStrengthDivisor", b.MaulBleedStrengthDivisor, 35},
		{"MaulBleedMin", b.MaulBleedMin, 1},
		{"HamstringBleedRounds", b.HamstringBleedRounds, 12},
		{"HamstringBleedStrengthDivisor", b.HamstringBleedStrengthDivisor, 50},
		{"HamstringBleedMin", b.HamstringBleedMin, 1},
		{"DrainBleedRounds", b.DrainBleedRounds, 10},
		{"DrainBleedStrengthDivisor", b.DrainBleedStrengthDivisor, 50},
		{"DrainBleedMin", b.DrainBleedMin, 1},
		{"ThrottleBleedRounds", b.ThrottleBleedRounds, 8},
		{"ThrottleBleedStrengthDivisor", b.ThrottleBleedStrengthDivisor, 33},
		{"ThrottleBleedMin", b.ThrottleBleedMin, 1},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d after Validate on an empty Balance, want default %d", c.name, c.got, c.want)
		}
	}
}

func TestBleedStackKnobs_LegalValuesSurvive(t *testing.T) {
	b := Balance{
		RakeBleedRounds: 7, RakeBleedStrengthDivisor: 41, RakeBleedMin: 2,
		MaulBleedRounds: 7, MaulBleedStrengthDivisor: 41, MaulBleedMin: 2,
		HamstringBleedRounds: 7, HamstringBleedStrengthDivisor: 41, HamstringBleedMin: 2,
		DrainBleedRounds: 7, DrainBleedStrengthDivisor: 41, DrainBleedMin: 2,
		ThrottleBleedRounds: 7, ThrottleBleedStrengthDivisor: 41, ThrottleBleedMin: 2,
	}
	b.Validate()
	got := []ConfigInt{
		b.RakeBleedRounds, b.RakeBleedStrengthDivisor, b.RakeBleedMin,
		b.MaulBleedRounds, b.MaulBleedStrengthDivisor, b.MaulBleedMin,
		b.HamstringBleedRounds, b.HamstringBleedStrengthDivisor, b.HamstringBleedMin,
		b.DrainBleedRounds, b.DrainBleedStrengthDivisor, b.DrainBleedMin,
		b.ThrottleBleedRounds, b.ThrottleBleedStrengthDivisor, b.ThrottleBleedMin,
	}
	want := []ConfigInt{7, 41, 2}
	for i, g := range got {
		if g != want[i%3] {
			t.Errorf("%s = %d, a legal value must survive Validate (want %d)", bleedStackKeys[i], g, want[i%3])
		}
	}
}

func TestShippedConfigNamesEveryBleedStackKnob(t *testing.T) {
	src := shippedConfigSource(t)
	for _, k := range bleedStackKeys {
		if !bytesContainsKey(src, k+":") {
			t.Errorf("config.yaml must name %s: the shipped tuning is the owner's, not a Go default", k)
		}
	}
}

// The slice 1b targets (spec, "Bleed record and producers"): a stack lasts 2
// to 3 special-move cooldowns, so a lone attacker keeps 2 to 3 alive, and a
// stack's total at Strength 100 is 1.5 to 3 times the single hit it replaced.
// oldHit is that single hit: one trigger of Strength/12 (rake, drain),
// /10 (hamstring, throttle), /8 (maul).
//
// Reads config.yaml ON DISK (skip-worktree): the slice adds the same block to
// the disk copy and the committed blob, and CI checks out the blob.
func TestShippedBleedTuningMeetsTheSliceTargets(t *testing.T) {
	cfg, err := loadConfig(shippedConfigSource(t))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	cfg.Balance.Validate()
	b := cfg.Balance

	const strength = 100
	cooldown := float64(b.SpecialMoveCooldown)
	rows := []struct {
		move                   string
		rounds, divisor, floor ConfigInt
		oldHit                 int
	}{
		{"rake", b.RakeBleedRounds, b.RakeBleedStrengthDivisor, b.RakeBleedMin, 8},
		{"maul", b.MaulBleedRounds, b.MaulBleedStrengthDivisor, b.MaulBleedMin, 12},
		{"hamstring", b.HamstringBleedRounds, b.HamstringBleedStrengthDivisor, b.HamstringBleedMin, 10},
		{"drain", b.DrainBleedRounds, b.DrainBleedStrengthDivisor, b.DrainBleedMin, 8},
		{"throttle", b.ThrottleBleedRounds, b.ThrottleBleedStrengthDivisor, b.ThrottleBleedMin, 10},
	}
	for _, r := range rows {
		if n := float64(r.rounds) / cooldown; n < 2 || n > 3 {
			t.Errorf("%s: a stack lasts %d rounds = %.2f cooldowns of %v; want 2 to 3 so a lone attacker keeps 2 to 3 stacks", r.move, r.rounds, n, cooldown)
		}
		perRound := strength / int(r.divisor)
		if perRound < int(r.floor) {
			perRound = int(r.floor)
		}
		if ratio := float64(int(r.rounds)*perRound) / float64(r.oldHit); ratio < 1.5 || ratio > 3 {
			t.Errorf("%s: a stack totals %d at Strength 100 = %.2f times the old %d hit; want 1.5 to 3", r.move, int(r.rounds)*perRound, ratio, r.oldHit)
		}
	}
}
