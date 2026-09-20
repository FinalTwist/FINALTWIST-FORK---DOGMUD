# Messaging M4c: One Defence Band Model Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every defence in the game picks its narration band from the same two
inputs, defensive crit and the normalized contest margin, with the Normal
cutoff and weather's felt cutoff read from config instead of Go literals.

**Architecture:** Melee auto-attacks are the last path still banding on
`best.defRoll.ZScore`, a self-relative roll that knows nothing about the
opponent. `internal/combat/combat_helpers.go:1300` stops calling
`items.GetDefenseMessage` and calls `items.RenderDefenseMessage`, the function
spells, ranged, special moves, taunt and counters already use, passing the
defensive-crit flag and the same normalized margin that already drives
`DefenceMitigation`. `GetDefenseMessage` and its z-score cutoffs are deleted.
The 0.5 cutoff inside `RenderDefenseMessage` and weather's `StrongFeltThreshold`
become balance knobs shipping at today's values.

**Tech Stack:** Go, `internal/items` (band selection and authored pools),
`internal/combat` (melee resolution), `internal/configs` (balance knobs),
`modules/weather/content` (felt banding), golden-file tests under
`internal/narration/testdata/stores` and a new one under `internal/combat/testdata`.

---

## Facts verified against source (master `c7110dcc7`, 2026-09-20)

Every row below was read from the file named, today, not recalled.

| Fact | Value | Source |
|---|---|---|
| Melee band function | `GetDefenseMessage(pool, zScore)`, Heavy >= 2.0, Normal >= 0.5, else Weak | `internal/items/defensive_messages.go:224-247` |
| Its production callers | **exactly one** | `internal/combat/combat_helpers.go:1300` |
| Its test references | 1 combat test, 1 items comment, 5 lines of the narration golden builder | `defense_message_coherence_test.go:22,48`, `defensive_messages_test.go:124`, `snapshot_test.go:496-555` |
| Canonical band function | `RenderDefenseMessage(pool, defensiveCrit, normalizedDefenceMargin, tokens, indexOverride...)`, Heavy on crit, Normal >= 0.5, else Weak | `internal/items/defensive_messages.go:135-152` |
| Its production callers | 3: counters x2, channel defence x1 | `internal/combat/counter.go:201,260`, `internal/combat/defence_multiplier.go:299` |
| Melee already has the margin | `defMargin` computed at the non-crit defensive win, zeroed when floored | `internal/combat/combat_helpers.go:1233-1239` |
| Margin helper | `normalizedDefenseMargin(best) (float64, bool)` = `best.margin / (StdDev * sqrt2)`, defence-positive | `internal/combat/margin_crit.go:73-84` |
| Channel margin | `out.NormalizedDefenceMargin` = `-res.Margin / (StdDev * sqrt2)` | `internal/combat/defence_multiplier.go:616` |
| Sign conventions | `contest.Result.Margin` is ATTACK-positive; `bestDefenseResult.margin` is DEFENCE-positive | `internal/contest/contest.go:45-51` |
| `sendDefenseMessages` call sites | 2 production (`:1204` crit, `:1244` partial) plus 1 test | `combat_helpers.go`, `defense_message_coherence_test.go:72` |
| `partial` today | `false` only on the defensive-crit branch, `true` only on the mitigated win | `combat_helpers.go:1198-1245` |
| Defensive crit threshold | `ContestCritThreshold = 2.0` standard deviations of margin | `internal/combat/margin_crit.go:90` |
| Floored outcomes | carry a +-1 sentinel margin, never crit, take `defMargin = 0` | `combat_helpers.go:1178-1239`, `contest.go:78-82` |
| Weather felt cutoff | `const StrongFeltThreshold = 0.5` | `modules/weather/content/emotes.go:17`, used at `:280` |
| Weather can read config | module already calls `configs.GetFilePathsConfig()` | `modules/weather/weather_tick.go:48` |
| `internal/items` can read config | already imports `internal/configs` | `internal/items/material_tier.go:3,39` |
| Balance knob pattern | `ConfigFloat` plus yaml tag plus a default in a `validateX()` | `config.balance.go:92`, `config.balance.misc.go:170-172` |
| Legal-zero precedent | `CritBarCeiling` 0 means uncapped and survives validation | `config.balance.combat.go:409`, `config_critbar_test.go:9` |
| Reject-zero precedent | `ContestFloor <= 0` reverts to 0.125 because **test binaries never load config.yaml** | `config.balance.misc.go:169-172`, `config_contestfloor_test.go` |
| Golden census | 208 lines: 90 store rows (10 pools x 3 bands x 3 roles), 3 empty-case, **93 melee-seam**, 3 header blocks | `internal/narration/testdata/stores/defense_messages.golden` |
| 🔴 **Melee-seam rows are duplicates** | the 90 `melee\|` rows, prefix stripped, are **byte-identical** to the 90 store rows (measured, not inferred: `diff` of the two sorted extracts is empty) | same file |
| Fixture helpers | `items.SeedDefenseMessagesForTest(map[DefencePool]*DefenseMessageGroup) func()`, `items.MinimalDefenseMessageFixture()` | `internal/items/test_helpers_combat.go:32,55` |
| Production contest runner | `combat.RunContest(atkScore, entries)` = `contest.RunWithFloors(..., ContestFloor)` | `internal/combat/run_contest.go:29` |

### What the verification changed about the spec

The M4c section of `docs/superpowers/specs/2026-09-17-messaging-m4-flip-design.md`
was written on 2026-09-17 and three of its line references have since drifted
(`defensive_messages.go:191-214` to `:224`, `combat_helpers.go:1309` to `:1300`,
`defence_multiplier.go:615` to `:299`). The design is unchanged; Task 6 corrects
the references.

Two findings the spec did not have:

1. 🔴 **The golden's melee seam is a false guard.** Its 90 rows duplicate the
   store rows byte for byte, because both render index 0 of the same pools. It
   freezes `RenderTriad` coordination, which the store rows already freeze, and
   says nothing about which band melee picks. It therefore cannot fail on the
   change this slice makes. Task 2 replaces it with a golden that drives the
   real `sendDefenseMessages`, per the M4a lesson: *a golden covers the store,
   not the path production calls.*
2. ⚠️ **A fourth hardcoded narration band exists that the spec's Bands table
   never listed:** `items.GetAttackMessage` bands on `pctDamage` at 101, 75,
   30 and 1 (`internal/items/attack_messages.go:297-311`). It is a different
   axis, damage magnitude rather than contest outcome, so it does not belong in
   "one defence band model", but the spec's end-state claim "no hardcoded
   narration cutoff" is false while it stands. **Open question for the owner**
   at the foot of this plan; NOT actioned here without a ruling.

---

## File Structure

**Created**
- `tools/melee_band_census/main.go`: one-off measurement, not wired into CI.
- `docs/superpowers/audits/2026-09-20-melee-defence-band-shares.md`: its output.
- `internal/combat/melee_defence_band_golden_test.go`: production-path band golden.
- `internal/combat/testdata/melee_defence_bands.golden`: recorded before the flip, regenerated once after.

**Modified**
- `internal/items/defensive_messages.go`: knob read; `GetDefenseMessage` deleted.
- `internal/combat/combat_helpers.go`: `defenceBand` param; `RenderDefenseMessage` call.
- `internal/combat/defense_message_coherence_test.go`: fixture moves off `ZScore`.
- `internal/narration/snapshot_test.go`: melee-seam builder deleted.
- `internal/narration/testdata/stores/defense_messages.golden`: melee section deleted.
- `internal/configs/config.balance.go`: two field declarations.
- `internal/configs/config.balance.combat.go`: two validation blocks.
- `_datafiles/config.yaml`: two documented knob blocks (**skip-worktree, see Task 3**).
- `modules/weather/content/emotes.go`: const becomes a config read.
- `internal/items/context.md`, `internal/combat/context.md`, `modules/weather/context.md`.
- `docs/superpowers/specs/2026-09-17-messaging-m4-flip-design.md`: M4c marked done.
- `_datafiles/world/dogmud/patchnotes/`: one note.
- `docs/README.md`: the two new docs.

---

## Task 1: Measure the band shares before changing anything

**Why first:** the slice's whole diff is "which band does a melee defence draw
from", and the standing rule is MEASURE a rate, never infer it. The owner
reviews the diff against real numbers, not a guess. It is a one-off tool rather
than a test because `internal/combat` already burns about 200s of CI's 300s
`-race` cap and must not grow.

**Files:**
- Create: `tools/melee_band_census/main.go`
- Create: `docs/superpowers/audits/2026-09-20-melee-defence-band-shares.md`

- [ ] **Step 1: Write the census tool**

```go
// Command melee_band_census measures how often a melee defensive win lands in
// each narration band under the OLD rule (defender's self-relative z-score) and
// the NEW rule (defensive crit, then normalized contest margin), across a grid
// of attacker/defender score matchups.
//
// One-off: run before and after the M4c flip, paste the table into
// docs/superpowers/audits/2026-09-20-melee-defence-band-shares.md. Not wired
// into CI -- internal/combat is already ~200s of the 300s -race cap.
package main

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/contest"
)

const samples = 50000

type shares struct{ weak, normal, heavy int }

func (s shares) row(label string, total int) string {
	if total == 0 {
		return fmt.Sprintf("| %s | - | - | - | 0 |", label)
	}
	pct := func(n int) float64 { return 100 * float64(n) / float64(total) }
	return fmt.Sprintf("| %s | %.1f%% | %.1f%% | %.1f%% | %d |",
		label, pct(s.weak), pct(s.normal), pct(s.heavy), total)
}

func main() {
	matchups := []struct {
		name               string
		atkScore, defScore float64
	}{
		{"defender outclassed (100 vs 60)", 100, 60},
		{"defender behind (100 vs 85)", 100, 85},
		{"even (100 vs 100)", 100, 100},
		{"defender ahead (85 vs 100)", 85, 100},
		{"defender dominant (60 vs 100)", 60, 100},
	}

	fmt.Println("| matchup | rule | weak | normal | heavy | defensive wins |")
	fmt.Println("|---|---|---|---|---|---|")
	for _, m := range matchups {
		var oldS, newS shares
		wins := 0
		for i := 0; i < samples; i++ {
			res := combat.RunContest(m.atkScore, []contest.Entry{{Name: "dodge", Score: m.defScore}})
			if res.Success { // the attacker won; no defence narration is sent
				continue
			}
			wins++

			// OLD rule: the defender's own roll, self-relative.
			switch {
			case res.DefenseRoll.ZScore >= 2.0:
				oldS.heavy++
			case res.DefenseRoll.ZScore >= 0.5:
				oldS.normal++
			default:
				oldS.weak++
			}

			// NEW rule: defensive crit, else the normalized margin. A floored
			// outcome carries a sentinel margin, never crits, and reads 0.
			norm := 0.0
			if !res.Floored && res.DefenseRoll.StdDev > 0 {
				norm = -res.Margin / (res.DefenseRoll.StdDev * math.Sqrt2)
			}
			switch {
			case !res.Floored && norm >= combat.ContestCritThreshold:
				newS.heavy++
			case norm >= 0.5:
				newS.normal++
			default:
				newS.weak++
			}
		}
		fmt.Println(oldS.row(m.name+" | old (z-score)", wins))
		fmt.Println(newS.row(m.name+" | new (crit+margin)", wins))
	}
}
```

- [ ] **Step 2: Run it**

Run: `go run ./tools/melee_band_census`

Expected: a ten-row markdown table on stdout, each matchup printing an old row
and a new row whose percentages sum to 100 and whose defensive-win counts match.
If any row prints `0` defensive wins the matchup is degenerate: report it, do
not silently drop it.

- [ ] **Step 3: Confirm the tool can see a difference**

The tool is a measurement, so its failure mode is "both rules print the same
numbers because I wired one input to the other". Temporarily change the NEW
rule's `norm` assignment to `norm = res.DefenseRoll.ZScore`, rerun, and confirm
the two rows of each matchup become identical. Revert. If they were ALREADY
identical before the sabotage, the tool is broken: stop and fix it.

- [ ] **Step 4: Write the audit doc**

Create `docs/superpowers/audits/2026-09-20-melee-defence-band-shares.md` with a
one-paragraph statement of what was measured and on which commit, the exact
command, the pasted table, and a closing paragraph naming which matchups move
most. State the sample size (50,000 per matchup) and that percentages are of
defensive wins, not of all swings.

- [ ] **Step 5: Commit**

```bash
git add tools/melee_band_census/main.go docs/superpowers/audits/2026-09-20-melee-defence-band-shares.md
git commit -m "measure(combat): melee defence band shares under the old and new rules

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: A golden that drives the real melee path, recorded BEFORE the flip

**Why:** three near-misses in M4a had one shape, *a golden covers the store, not
the path production calls*. The existing melee-seam rows are that mistake a
fourth time: they duplicate the store rows and cannot fail on a band change.
This golden calls `sendDefenseMessages` itself.

**Design:** synthetic single-variant pools labelled by pool and band
(`DODGE|HEAVY|actee`), so the recorded file is a BAND MATRIX, not prose. Shipped
prose stays frozen by the store rows of `defense_messages.golden`. A single
variant makes production randomness irrelevant, so no picker override is needed,
which is the only reason this path can be goldened at all.

**Files:**
- Create: `internal/combat/melee_defence_band_golden_test.go`
- Create: `internal/combat/testdata/melee_defence_bands.golden` (generated)

- [ ] **Step 1: Write the golden test**

```go
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
func firstText(msgs []AttackMessage) string {
	if len(msgs) == 0 {
		return "(none)"
	}
	return msgs[0].Text
}
```

⚠️ `AttackMessage` is the element type of `AttackResult.MessagesToTarget`.
Confirm the exact type name and its text field with
`grep -n "MessagesToTarget" internal/combat/attackresult.go` before writing
`firstText`, and use whatever that file declares. Do not guess.

- [ ] **Step 2: Record the golden**

Run: `go test ./internal/combat/ -run TestMeleeDefenceBandGolden -update-melee-bands -v`

Expected: PASS, logging `golden rewritten`. Then count the rows explicitly, a
census rather than a guess:

```bash
grep -c "=>" internal/combat/testdata/melee_defence_bands.golden
```

Expected: **165** (3 defences x 3 z-scores x 3 margins x 2 partial x 3 roles =
162, plus 3 floored rows). If the number differs, the grid is not what this plan
describes: stop and reconcile before recording.

- [ ] **Step 3: Read the recorded file and confirm it captures TODAY's rule**

```bash
grep "z=2.50|m=0.00|partial=true|observer" internal/combat/testdata/melee_defence_bands.golden
```

Expected: `HEAVY`. Today's rule bands on the z-score alone, so a high z with a
zero margin and no crit is Heavy. **This row is the one M4c moves**, and seeing
HEAVY here now is what makes the Task 4 diff meaningful.

```bash
grep "z=0.00|m=2.50|partial=false|observer" internal/combat/testdata/melee_defence_bands.golden
```

Expected: `WEAK`. A decisive margin on a defensive crit reads Weak today,
because neither input is consulted.

- [ ] **Step 4: Prove the golden can fail**

In `internal/items/defensive_messages.go`, temporarily swap the `Heavy` and
`Weak` assignments inside `GetDefenseMessage`. Run:

`go test ./internal/combat/ -run TestMeleeDefenceBandGolden`

Expected: **FAIL**, with a diff naming band labels. Revert the sabotage and
rerun; expected PASS. A green run before the sabotage proves nothing; a red run
during it is the only evidence this golden is capable of failing.

- [ ] **Step 5: Commit**

```bash
git add internal/combat/melee_defence_band_golden_test.go internal/combat/testdata/melee_defence_bands.golden
git commit -m "test(combat): freeze the melee defence band matrix through the production path

The melee rows in defense_messages.golden are byte-identical duplicates of its
store rows: both render index 0 of the same pools, so they freeze RenderTriad
coordination and nothing about which band melee picks. This golden drives
sendDefenseMessages itself over a grid crossing both banding inputs, and is
recorded BEFORE M4c moves the rule.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: The Normal-band cutoff becomes a balance knob

**Byte-identical by construction:** the default equals today's literal 0.5.

**Files:**
- Modify: `internal/configs/config.balance.go` (field declaration)
- Modify: `internal/configs/config.balance.combat.go` (validation)
- Create: `internal/configs/config_defence_band_test.go`
- Modify: `internal/items/defensive_messages.go:135-141`
- Modify: `_datafiles/config.yaml` (**skip-worktree, see Step 5**)

- [ ] **Step 1: Write the failing validation test**

```go
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
```

⚠️ `b.Validate()` is the entry point only if `Balance` exposes one that reaches
`validateCombat`. Confirm with
`grep -n "func (b \*Balance) Validate" internal/configs/config.balance*.go`
and call whatever the sibling tests (`config_contestfloor_test.go`,
`config_critbar_test.go`) call. Match them exactly.

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/configs/ -run TestDefenceBandNormalThreshold -v`

Expected: FAIL TO COMPILE with `unknown field DefenceBandNormalThreshold`. A
compile failure is the correct first red here.

- [ ] **Step 3: Declare the field and its validation**

In `internal/configs/config.balance.go`, beside the other combat narration knobs:

```go
	// DefenceBandNormalThreshold is the normalized contest margin at or above
	// which a NON-CRIT defensive win narrates from the Normal pool instead of
	// the Weak one. Margins are in standard deviations of the contest spread,
	// the same scale ContestCritThreshold (2.0) uses, so raising this to 2.0
	// would collapse Normal entirely and lowering it to 0 would delete Weak.
	// A defensive CRIT always narrates Heavy regardless of this knob.
	// Zero is NOT legal: a Go test binary never loads config.yaml, so a
	// permissive check would leave every test at zero and silently retire the
	// Weak band repo-wide. Same reasoning as ContestFloor.
	DefenceBandNormalThreshold ConfigFloat `yaml:"DefenceBandNormalThreshold"` // Normalized margin for the Normal defence band (default 0.5); 0 is rejected
```

In `internal/configs/config.balance.combat.go`, in the style of the
`CritBarCeiling` block:

```go
	if b.DefenceBandNormalThreshold <= 0 || b.DefenceBandNormalThreshold > 5.0 {
		b.DefenceBandNormalThreshold = 0.5
	}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/configs/ -run TestDefenceBandNormalThreshold -v`

Expected: PASS, both cases.

- [ ] **Step 5: Add the documented block to `_datafiles/config.yaml`**

🪤 **`_datafiles/config.yaml` carries the git skip-worktree bit and desyncs in
both directions.** Do not build the commit from the working-tree file. The
procedure (skill `dogmud-balance-config`):

```bash
git show HEAD:_datafiles/config.yaml > /tmp/config_head.yaml
```

Edit `/tmp/config_head.yaml`, inserting after the `MinDefenseCritChance` line:

```yaml
  #
  # DefenceBandNormalThreshold: the normalized contest margin at or above which
  #   a NON-CRIT defensive win is narrated from the Normal pool rather than the
  #   Weak one. Margins are measured in standard deviations of the contest
  #   spread, the same scale as the 2.0 crit threshold, so 0.5 means "the
  #   defence won by half a standard deviation or better".
  #   A defensive CRIT always narrates from the Heavy pool and ignores this.
  #   Range: above 0.0 and at most 5.0. Zero is NOT legal: it fails validation
  #   and reverts to 0.5, because Go test binaries never load this file and a
  #   zero here would retire the Weak band in every test in the repo.
  DefenceBandNormalThreshold: 0.5
```

Then stage the blob without touching the working tree:

```bash
git hash-object -w /tmp/config_head.yaml   # prints <sha>
git update-index --cacheinfo 100644,<sha>,_datafiles/config.yaml
```

🪤 **`--cacheinfo` CLEARS skip-worktree.** Restore it immediately and verify:

```bash
git update-index --skip-worktree _datafiles/config.yaml
git ls-files -v _datafiles/config.yaml   # MUST print a lowercase 's'
```

Apply the same edit to the working-tree file too, so the local server runs with
the knob present. That is Claude's job, never a chore handed to the owner.

- [ ] **Step 6: Read the knob in `RenderDefenseMessage`**

Replace the literal in `internal/items/defensive_messages.go`:

```go
func RenderDefenseMessage(defenseType DefencePool, defensiveCrit bool, normalizedDefenceMargin float64, tokenReplacements map[TokenName]string, indexOverride ...int) DefenseMessageTriad {
	intensity := Weak
	if defensiveCrit {
		intensity = Heavy
	} else if normalizedDefenceMargin >= float64(configs.GetBalanceConfig().DefenceBandNormalThreshold) {
		intensity = Normal
	}
```

- [ ] **Step 7: Prove nothing moved**

```bash
go test ./internal/narration/ ./internal/items/ ./internal/combat/
git diff --stat internal/narration/testdata internal/combat/testdata
```

Expected: tests PASS and **no output** from the `git diff --stat`. The knob's
validated default is today's literal, so no band may move.

- [ ] **Step 8: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.combat.go internal/configs/config_defence_band_test.go internal/items/defensive_messages.go
git commit -m "feat(config): the Normal defence band cutoff is a balance knob

Ships at today's hardcoded 0.5, so no narration moves. Zero is rejected for
the ContestFloor reason: test binaries never load config.yaml.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"

git commit -m "chore(config): document DefenceBandNormalThreshold in config.yaml

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>" -- _datafiles/config.yaml
```

Verify the second commit came from the staged blob, not the working tree:

```bash
git show HEAD:_datafiles/config.yaml | grep -n DefenceBandNormalThreshold
git ls-files -v _datafiles/config.yaml
```

Expected: the key with its comment block, and a lowercase `s`.

---

## Task 4: Melee bands on crit and margin, the deliberate diff

**Files:**
- Modify: `internal/combat/combat_helpers.go:1198-1340`
- Modify: `internal/items/defensive_messages.go` (delete `GetDefenseMessage`)
- Modify: `internal/combat/defense_message_coherence_test.go:22,48,63-72`
- Modify: `internal/narration/snapshot_test.go:493-560`
- Modify: `internal/narration/testdata/stores/defense_messages.golden`
- Modify: `internal/combat/testdata/melee_defence_bands.golden` (regenerated once)

- [ ] **Step 1: Introduce the band parameter**

In `internal/combat/combat_helpers.go`, above `sendDefenseMessages`:

```go
// defenceBand carries the two inputs every defence in the game now bands on.
// It is a struct rather than two more bare parameters because sendDefenseMessages
// already takes two bools, and a third would be positional-argument roulette.
//
// crit is NOT derived from the existing `partial` flag. They happen to be
// opposites at both call sites today, and relying on that would band a future
// non-crit caller as Heavy the moment someone adds one.
type defenceBand struct {
	crit   bool
	margin float64 // normalized, defence-positive; 0 when floored
}
```

Change the signature:

```go
func sendDefenseMessages(result *AttackResult, best bestDefenseResult, sourceChar *characters.Character, targetChar *characters.Character, isThirdParty bool, partial bool, band defenceBand) {
```

- [ ] **Step 2: Replace the band lookup**

Delete the `defenseMsgs := items.GetDefenseMessage(...)` line and the
`triad := defenseMsgs.RenderTriad(tokenReplacements, nil)` line, and build the
triad from the canonical function, leaving the token map between them untouched:

```go
	// M4c: melee bands through the same function as spells, ranged, special
	// moves, taunt and counters -- defensive crit, then the normalized contest
	// margin. It used to band on best.defRoll.ZScore, the defender's own roll
	// against their own mean, which is decisive about nothing: a defender who
	// rolled well for themselves and still barely scraped the swing narrated
	// as though they had dismissed it.
	triad := items.RenderDefenseMessage(itemsDefencePool, band.crit, band.margin, tokenReplacements)
```

- [ ] **Step 3: Fill the band at both call sites**

The crit-branch call at `:1204`:

```go
			sendDefenseMessages(result, best, sourceChar, targetChar, isThirdParty, false, defenceBand{crit: true})
```

The mitigated-win call at `:1244`, which already has `defMargin` in hand:

```go
	sendDefenseMessages(result, best, sourceChar, targetChar, isThirdParty, true, defenceBand{margin: defMargin})
```

⚠️ `defMargin` is computed at `:1233-1239` and is already zeroed when
`best.floored`. Do not recompute it inside `sendDefenseMessages`: the floored
rule lives with the mitigation maths and must stay in one place.

- [ ] **Step 4: Delete `GetDefenseMessage`**

Remove the whole function from `internal/items/defensive_messages.go:223-247`.
Then verify the deletion is total, trusting the compiler over the grep:

```bash
go build ./...
grep -rn "GetDefenseMessage" --include=*.go .
```

Expected: build clean, and only comment references left, which Task 6 sweeps.

- [ ] **Step 5: Fix the coherence test's fixture**

`internal/combat/defense_message_coherence_test.go` pins the Weak band with
`defRoll: dice.RollResult{ZScore: 0.0}`, which no longer selects anything.
Replace the `best` literal and the call:

```go
	best := bestDefenseResult{
		defenseType: combatvocab.DefenceBlock,
		margin:      0, // normalized margin 0, no crit => Weak band
		defRoll:     dice.RollResult{StdDev: 10},
	}
```

```go
		sendDefenseMessages(result, best, sourceChar, targetChar, false, false, defenceBand{})
```

Update the stale comment at `:22` (`items.GetDefenseMessage -> items.RenderTriad`)
to name `items.RenderDefenseMessage`, and the one at `:48` explaining why
Normal and Heavy are absent from the fixture: it still holds, but now because
`RenderDefenseMessage` looks up only the band it needs.

- [ ] **Step 6: Delete the false melee seam from the store golden**

In `internal/narration/snapshot_test.go`, delete the `MELEE SEAM` block and the
`EMPTY CASE (melee seam)` block (the `meleeBands` loop and both
`GetDefenseMessage` calls), and delete the five header lines at `:494-500` that
describe them. Add in their place:

```go
	fmt.Fprintf(&b, "# The melee path no longer has a banding rule of its own (M4c): internal/combat\n")
	fmt.Fprintf(&b, "# calls RenderDefenseMessage like every other defence. Its production-path matrix\n")
	fmt.Fprintf(&b, "# is frozen by internal/combat/testdata/melee_defence_bands.golden; the rows that\n")
	fmt.Fprintf(&b, "# used to sit here were byte-identical duplicates of the store rows above.\n")
```

- [ ] **Step 7: Regenerate both goldens ONCE and review the diff**

```bash
go test ./internal/narration/ -run TestSnapshotStores -update
go test ./internal/combat/ -run TestMeleeDefenceBandGolden -update-melee-bands
git diff --stat internal/narration/testdata internal/combat/testdata
```

Expected: `defense_messages.golden` loses exactly **93 rows** and gains 4 header
lines; `melee_defence_bands.golden` keeps its 165 rows with band labels moved.

Verify the store deletion lost no prose. Run this standalone, not in an `&&`
chain (🪤 `grep -c` exits 1 on zero matches):

```bash
git show HEAD:internal/narration/testdata/stores/defense_messages.golden | grep "^melee|" | grep -v nonexistent | sed 's/^melee|//' | sort > /tmp/deleted.txt
grep -v "^melee|" internal/narration/testdata/stores/defense_messages.golden | grep "=>" | grep -v nonexistent | sort > /tmp/kept.txt
diff /tmp/deleted.txt /tmp/kept.txt && echo "PROVEN: every deleted melee row survives as a store row"
```

Expected: the `PROVEN` line. If it differs, the deletion is losing coverage: stop.

**Then review the melee matrix diff row by row.** Only band labels may move. The
expected shape, checked against the Task 1 audit numbers:
- `partial=false` (defensive crit) rows: every one becomes `HEAVY`, whatever z
  or margin says.
- `partial=true` rows: `NORMAL` iff margin >= 0.5, else `WEAK`. The z-score
  column no longer influences anything, so all three z values within a
  (defence, margin) group must now read the same band.
- `FLOORED` rows: `WEAK`.
- Any row whose ROLE set changed (an `actee` or `actor` line appearing or
  vanishing) is a DEFECT, not a band move: `partial` still owns that.

- [ ] **Step 8: Run the full gates**

```bash
go test ./internal/combat/ ./internal/items/ ./internal/narration/ ./internal/hooks/
go test .
go build ./...
```

Expected: PASS everywhere. 🪤 `go test .` at the repo ROOT is not optional:
`condition_apply_path_guard_test.go` is a line-number allowlist over
`combat_*.go`, and Step 2 edits sit above allowlisted lines. Expect to re-key
it; if it passes untouched, confirm the allowlist actually covers the lines you
moved before believing it.

- [ ] **Step 9: Commit**

```bash
git add internal/combat/combat_helpers.go internal/items/defensive_messages.go internal/combat/defense_message_coherence_test.go internal/narration/snapshot_test.go internal/narration/testdata/stores/defense_messages.golden internal/combat/testdata/melee_defence_bands.golden
git commit -m "feat(combat)!: melee defence bands read contest margin and defensive crit

Melee auto-attacks were the last defence in the game banding on
best.defRoll.ZScore, the defender's roll against their own mean, which says
nothing about whether the swing was close. They now band through
items.RenderDefenseMessage like spells, ranged, special moves, taunt and
counters: Heavy on a defensive crit, Normal at or above the configured margin,
Weak otherwise.

PLAYER-VISIBLE. A defender who rolls high for themselves but barely beats the
attacker now reads as a narrow escape rather than a dismissal, and a defensive
crit always reads decisive. Band shares measured before and after in
docs/superpowers/audits/2026-09-20-melee-defence-band-shares.md.

items.GetDefenseMessage and its 2.0/0.5 z-score cutoffs are deleted.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Weather's felt cutoff becomes a balance knob

**Files:**
- Modify: `internal/configs/config.balance.go`, `internal/configs/config.balance.combat.go`
- Create: `internal/configs/config_weather_felt_test.go`
- Modify: `modules/weather/content/emotes.go:14-20,169,243,280`
- Modify: `internal/narration/snapshot_test.go:1297` (comment names the const)
- Modify: `_datafiles/config.yaml`

- [ ] **Step 1: Write the failing test**

```go
package configs

import "testing"

// Weather emotes band Mild/Strong on felt intensity. Zero would make every
// indoor moment Strong in every test binary, since no test loads config.yaml.
func TestWeatherStrongFeltThreshold_ZeroIsRejected(t *testing.T) {
	b := Balance{WeatherStrongFeltThreshold: 0}
	b.Validate()
	if b.WeatherStrongFeltThreshold != 0.5 {
		t.Fatalf("zero must revert to the shipped default 0.5, got %v", b.WeatherStrongFeltThreshold)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/configs/ -run TestWeatherStrongFeltThreshold -v`

Expected: FAIL TO COMPILE, `unknown field WeatherStrongFeltThreshold`.

- [ ] **Step 3: Declare, validate, and read it**

Field in `internal/configs/config.balance.go`:

```go
	// WeatherStrongFeltThreshold is the felt intensity at or above which indoor
	// and underground weather emotes draw from the Strong pool instead of Mild.
	// Felt is weather's own 0..1 scale and is unrelated to contest margins.
	// Zero is rejected for the same reason every other narration cutoff rejects
	// it: test binaries never load config.yaml.
	WeatherStrongFeltThreshold ConfigFloat `yaml:"WeatherStrongFeltThreshold"` // Felt intensity for Strong weather emotes (default 0.5); 0 is rejected
```

Validation in `internal/configs/config.balance.combat.go`, beside the defence
band knob:

```go
	if b.WeatherStrongFeltThreshold <= 0 || b.WeatherStrongFeltThreshold > 1.0 {
		b.WeatherStrongFeltThreshold = 0.5
	}
```

In `modules/weather/content/emotes.go`, delete the const at `:17` and replace
the comparison at `:280`:

```go
	if felt >= float64(configs.GetBalanceConfig().WeatherStrongFeltThreshold) {
		return pool.Strong
	}
```

Add the `github.com/GoMudEngine/GoMud/internal/configs` import. Keep the doc
comment at `:14-20` and rewrite its first line to name the knob rather than the
const. Fix the other comment references at `:20`, `:169` and `:243`, and the one
in `internal/narration/snapshot_test.go:1297`, which names the const and its
value.

- [ ] **Step 4: Run the tests**

```bash
go test ./internal/configs/ -run TestWeatherStrongFeltThreshold -v
go test ./modules/weather/... ./internal/narration/
git diff --stat internal/narration/testdata
```

Expected: tests PASS and **no output** from `git diff --stat`. The default
equals the deleted const, so no emote may move.

- [ ] **Step 5: Add the yaml block**

Same `git show HEAD:` procedure as Task 3 Step 5, including the skip-worktree
restore and the `git ls-files -v` check. Place it beside the defence band knob:

```yaml
  #
  # WeatherStrongFeltThreshold: the felt intensity at or above which indoor and
  #   underground weather emotes draw from the Strong pool instead of Mild.
  #   Felt is weather's own 0..1 scale; it is NOT a contest margin.
  #   Range: above 0.0 and at most 1.0. Zero is not legal (reverts to 0.5).
  WeatherStrongFeltThreshold: 0.5
```

- [ ] **Step 6: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.combat.go internal/configs/config_weather_felt_test.go modules/weather/content/emotes.go internal/narration/snapshot_test.go
git commit -m "feat(config): weather's Strong felt cutoff is a balance knob

Ships at the deleted const's 0.5, so no emote moves.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"

git commit -m "chore(config): document WeatherStrongFeltThreshold in config.yaml

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>" -- _datafiles/config.yaml
```

---

## Task 6: Docs, patch note, and the sweep by meaning

**Files:**
- Modify: `internal/items/context.md`, `internal/combat/context.md`, `modules/weather/context.md`
- Modify: `docs/superpowers/specs/2026-09-17-messaging-m4-flip-design.md`
- Create: `_datafiles/world/dogmud/patchnotes/<next>.yaml` (list the directory first and match its convention)
- Modify: `docs/README.md`

- [ ] **Step 1: Sweep by MEANING, not by name**

The counters slice earned this rule: *a comment sweep grep finds deleted NAMES,
never invalidated CLAIMS.* Run all three, each standalone:

```bash
grep -rn "GetDefenseMessage\|StrongFeltThreshold" --include=*.go --include=*.md .
grep -rni "z-score\|zscore" --include=*.go --include=*.md internal/combat internal/items docs/superpowers/specs | grep -i "defen\|band"
grep -rni "hardcoded\|self-relative\|its own banding\|own band" --include=*.md internal/ docs/superpowers/specs
```

Every hit asserting that melee bands on a z-score, or that a cutoff is
hardcoded, is now false. Fix each one.

- [ ] **Step 2: Update the three context.md files**

`internal/items/context.md`: remove `GetDefenseMessage` if listed; state that
`RenderDefenseMessage` is the single defence band function and that its Normal
cutoff is `Balance.DefenceBandNormalThreshold`.
`internal/combat/context.md`: state that `sendDefenseMessages` takes a
`defenceBand` and owns no banding rule of its own.
`modules/weather/context.md`: name the knob in place of the const.

Verify every symbol named actually exists:

```bash
python tools/context_md_audit.py
```

Expected: no findings for these three packages. 🪤 The audit cannot see struct
fields or prose claims, so it is a floor, not a ceiling.

- [ ] **Step 3: Correct the M4c section of the flip spec**

In `docs/superpowers/specs/2026-09-17-messaging-m4-flip-design.md`, mark M4c
done with its PR number, correct the three drifted line references named in this
plan's facts table, and replace the end-state sentence "no hardcoded narration
cutoff" with an accurate one: one defence band function and one weather
threshold, both configurable, with `items.GetAttackMessage`'s pctDamage cutoffs
remaining and recorded as an open question.

- [ ] **Step 4: Write the patch note**

```bash
ls _datafiles/world/dogmud/patchnotes/ | tail -5
```

Follow the newest file's schema exactly. Content, hard wrapped at 80 columns, no
raw numbers, no em dashes:

> Defence descriptions now match how decisively you actually defended.
> Turning a blow aside by a hair reads like a narrow escape, and a defence
> that completely shuts an attack down reads decisive. Melee defences used to
> describe how well you rolled rather than how close the attack came.

- [ ] **Step 5: Index the new files**

Add to `docs/README.md`, each with its full path and a one-line description:
`docs/superpowers/plans/2026-09-20-messaging-m4c-defence-bands.md` and
`docs/superpowers/audits/2026-09-20-melee-defence-band-shares.md`.

- [ ] **Step 6: Commit**

```bash
git add internal/items/context.md internal/combat/context.md modules/weather/context.md docs/superpowers/specs/2026-09-17-messaging-m4-flip-design.md docs/README.md _datafiles/world/dogmud/patchnotes/
git commit -m "docs(m4c): one defence band model, in context.md, the spec and a patch note

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Verify and open the PR

- [ ] **Step 1: Re-run the census on the new rule**

```bash
go run ./tools/melee_band_census
```

Append the post-change table to the audit doc under a second heading, so the
owner reads both rules' shares side by side from a measurement rather than from
this plan's prediction. Commit the audit update.

- [ ] **Step 2: Full local gate**

```bash
go build ./...
go test . ./...
golangci-lint run --new-from-rev=master
gofmt -l $(git diff --name-only master..HEAD | grep '\.go$')
```

Expected: build clean, tests PASS, **0 new lint issues**, and the `gofmt -l`
line prints **nothing**. 🪤 CI's `validate / lint` reddens on any PR over 300
files (diff API 406); this PR is far under that, so a red lint here is real.
🪤 CI's `validate / test` job ALSO runs a gofmt gate, separate from
golangci-lint, which fails the whole job in about 30 seconds before a single
test runs. A green `golangci-lint` says nothing about it. This bit the attack
band slice (PR #146): a comment misaligned by two spaces in that plan's own
code block. Run `gofmt -l` standalone, since it exits 0 whether or not it names
files, so chaining it hides the result.

- [ ] **Step 3: Boot check**

```bash
go run . 2>&1 | tee /tmp/m4c_boot.log
```

Then, each standalone:

```bash
grep -c "Server Ready" /tmp/m4c_boot.log
grep -c "PANIC" /tmp/m4c_boot.log
```

Expected: `Server Ready` present, `PANIC` absent. 🪤 **A failed boot exits 0**:
`main()` recovers and returns normally, so never test `$?`. Kill the server by
the PID you started, never by name or port sweep. The owner runs their own
server on this machine.

- [ ] **Step 4: Push and open the PR**

```bash
git push -u origin feature/messaging-m4c-bands
gh pr create --repo pruuk/DOGMud --base master \
  --title "M4c: one defence band model" \
  --body-file /tmp/m4c_pr_body.md
```

🪤 **Every `gh` command carries `--repo pruuk/DOGMud`.** This repo is a fork and
`gh` defaults to the upstream parent.

The PR body must disclose, in this order:
1. **The player-visible change**, with the before and after band-share table.
2. **That `defense_messages.golden` lost 93 rows**, with the proof that every
   deleted row survives as a store row, and why they were a false guard.
3. **The two new balance knobs**, both shipping at today's values, both
   rejecting zero, and why zero is rejected.
4. **The open question** on `items.GetAttackMessage`'s pctDamage cutoffs.
5. The root line-number guard re-key, if Task 4 Step 8 required one.

---

## Open questions for the owner

1. 🔴 **`items.GetAttackMessage` bands on `pctDamage` at 101, 75, 30 and 1**
   (`internal/items/attack_messages.go:297-311`), a fourth hardcoded narration
   cutoff the M4 spec's Bands table never listed. It is a different axis from
   the defence bands (damage magnitude, not contest outcome), so folding it into
   "one defence band model" would be a category error. Options: (a) four more
   balance knobs in this slice, (b) its own small slice after M4c, (c) leave it
   hardcoded and drop the "no hardcoded narration cutoff" claim from the spec.
   **This plan assumes (c) and states the limitation; say the word for (a) or (b).**
2. **No playtest gate.** The M4 spec gives M4c "deliberate, reviewed diff" and
   reserves the playtest for M4d. The band matrix golden plus the measured
   shares are the proof here. Flagged because this slice IS player-visible, and
   the counters slice's playtest found six defects a green suite could not see.

---

## Self-review

**Spec coverage.** Spec M4c has five bullets: delete `GetDefenseMessage` and its
z-score banding (Task 4), melee bands via `RenderDefenseMessage` with margin and
defensive crit (Task 4), the margin cutoff becomes a config knob at 0.5 (Task
3), `StrongFeltThreshold` moves to config the same way (Task 5), and a golden
grid of margins and crit flags recorded before the change whose diff is the
review (Task 2, reviewed in Task 4 Step 7). `AgingPhase`, skill tiers,
`TauntIntensity` and `items.Intensity` are explicitly out of scope per the spec
and are untouched. Covered.

**Placeholders.** None: every code step carries its code, every command carries
its expected output. Three steps carry a deliberate "confirm against the source
before writing it" instruction (`AttackMessage`'s field name, `Balance`'s
validate entry point, the patchnote schema) rather than a guessed symbol. That
is the read-the-code rule, not a placeholder.

**Type consistency.** `defenceBand{crit, margin}` is defined in Task 4 Step 1
and used with those field names in Steps 3 and 5.
`DefenceBandNormalThreshold` and `WeatherStrongFeltThreshold` are declared in
Tasks 3 and 5 and read as `float64(configs.GetBalanceConfig().X)` in both.
`items.SeedDefenseMessagesForTest`, `items.DefencePoolFor`,
`items.DefenseIntensity`, `items.DefenseOptions`,
`items.DefenseTogetherMessages` and `items.MessageOptions` are all
verified-existing symbols, not invented ones.
