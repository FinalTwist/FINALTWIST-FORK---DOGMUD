package combat

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
)

var updateMeleeBands = flag.Bool("update-melee-bands", false, "rewrite testdata/melee_defence_bands.golden")

// bandLabelFixture gives every (pool, band) pair ONE variant naming itself, so
// the golden records WHICH BAND the melee path selected rather than authored
// prose. Production randomness cannot move a one-element pool, which is why
// this path -- which has no picker override -- can be pinned at all.
func bandLabelFixture() map[items.DefencePool]*items.DefenseMessageGroup {
	pools := []items.DefencePool{
		items.DefencePoolFor(combatvocab.DefenceDodge),
		items.DefencePoolFor(combatvocab.DefenceParry),
		items.DefencePoolFor(combatvocab.DefenceBlock),
	}
	bands := map[items.Intensity]string{
		items.Weak: "WEAK", items.Normal: "NORMAL", items.Heavy: "HEAVY",
	}
	out := make(map[items.DefencePool]*items.DefenseMessageGroup, len(pools))
	for _, p := range pools {
		opts := items.DefenseIntensity{}
		for intensity, name := range bands {
			label := fmt.Sprintf("%s|%s", strings.ToUpper(string(p)), name)
			opts[intensity] = items.DefenseOptions{Together: items.DefenseTogetherMessages{
				ToDefender: items.MessageOptions{items.ItemMessage(label + "|actee")},
				ToAttacker: items.MessageOptions{items.ItemMessage(label + "|actor")},
				ToRoom:     items.MessageOptions{items.ItemMessage(label + "|observer")},
			}}
		}
		out[p] = &items.DefenseMessageGroup{OptionId: p, Options: opts}
	}
	return out
}

// TestMeleeDefenceBandGolden freezes the band the MELEE path selects, across a
// grid that crosses both banding inputs: the defender's self-relative z-score
// (what the path reads today) and the normalized contest margin plus the
// defensive-crit flag (what M4c moves it to). Crossing them is the point: a
// grid varying only one input cannot show the rule moving from one to the other.
func TestMeleeDefenceBandGolden(t *testing.T) {
	restore := items.SeedDefenseMessagesForTest(bandLabelFixture())
	defer restore()

	const stdDev = 10.0
	// margin is DEFENCE-positive in bestDefenseResult (contest.Result.Margin is
	// attack-positive; the conversion happens before this struct is built).
	marginFor := func(normalized float64) float64 { return normalized * stdDev * math.Sqrt2 }

	defences := []combatvocab.Defence{
		combatvocab.DefenceDodge, combatvocab.DefenceParry, combatvocab.DefenceBlock,
	}
	zScores := []float64{0.0, 0.6, 2.5}
	margins := []float64{0.0, 0.6, 2.5}

	var b strings.Builder
	fmt.Fprintf(&b, "# melee defence band matrix -- internal/combat sendDefenseMessages\n")
	fmt.Fprintf(&b, "# Each cell drives the REAL production function with a one-variant-per-band\n")
	fmt.Fprintf(&b, "# fixture, so a row names the BAND selected, not authored prose.\n")
	fmt.Fprintf(&b, "# columns: defence | defender z-score | normalized margin | partial | role => band\n")
	fmt.Fprintf(&b, "# partial=false is the defensive-CRIT call site (combat_helpers.go:1204);\n")
	fmt.Fprintf(&b, "# partial=true is the mitigated defensive win (:1244), which sends the room line only.\n\n")

	for _, d := range defences {
		for _, z := range zScores {
			for _, m := range margins {
				for _, partial := range []bool{false, true} {
					best := bestDefenseResult{
						defenseType: d,
						margin:      marginFor(m),
						defRoll:     dice.RollResult{ZScore: z, StdDev: stdDev},
					}
					src := characters.New()
					src.Name = "Attacker"
					tgt := characters.New()
					tgt.Name = "Defender"
					result := &AttackResult{}
					sendDefenseMessages(result, best, src, tgt, false, partial)

					key := fmt.Sprintf("%s|z=%.2f|m=%.2f|partial=%t", d, z, m, partial)
					fmt.Fprintf(&b, "%s|actee => %s\n", key, firstText(result.MessagesToTarget))
					fmt.Fprintf(&b, "%s|actor => %s\n", key, firstText(result.MessagesToSource))
					fmt.Fprintf(&b, "%s|observer => %s\n", key, firstText(result.MessagesToSourceRoom))
				}
			}
		}
	}

	// FLOORED: a floored outcome carries a +-1 sentinel margin rather than a
	// real one and must never read as a decisive defence.
	for _, d := range defences {
		best := bestDefenseResult{
			defenseType: d,
			margin:      marginFor(2.5),
			defRoll:     dice.RollResult{ZScore: 2.5, StdDev: stdDev},
			floored:     true,
		}
		src := characters.New()
		src.Name = "Attacker"
		tgt := characters.New()
		tgt.Name = "Defender"
		result := &AttackResult{}
		sendDefenseMessages(result, best, src, tgt, false, true)
		fmt.Fprintf(&b, "%s|FLOORED|observer => %s\n", d, firstText(result.MessagesToSourceRoom))
	}

	got := b.String()
	path := filepath.Join("testdata", "melee_defence_bands.golden")
	if *updateMeleeBands {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("writing golden: %v", err)
		}
		t.Logf("golden rewritten: %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden (run with -update-melee-bands to record): %v", err)
	}
	if string(want) != got {
		t.Fatalf("melee defence band matrix changed.\n--- want ---\n%s\n--- got ---\n%s", want, got)
	}
}

// firstText returns the single buffered message's text, or a marker when the
// call site sent nothing. "(none)" is a MEANINGFUL row: partial=true suppresses
// the two personal lines on purpose.
func firstText(msgs []TaggedMessage) string {
	if len(msgs) == 0 {
		return "(none)"
	}
	return msgs[0].Text
}
