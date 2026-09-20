# Melee defence narration band shares: old rule vs new rule

Date: 2026-09-20
Branch: feature/messaging-m4c-bands
Commit measured on: `b0a2ddedac7e46413aa8625a5f6acdcfb43944e1`

This measures how often a melee defensive win lands in each narration band
(weak, normal, heavy) under two rules, across five attacker/defender score
matchups, using `internal/combat.RunContest` directly so the numbers come
from the real contest and floor pipeline rather than a hand-rolled model.
The OLD rule bands on the defender's own roll expressed self-relative to
their own mean (`DefenseRoll.ZScore`), the input every melee auto-attack
path still uses today. The NEW rule bands on defensive crit first, then the
contest margin normalized against the attacker/defender shared standard
deviation (`-Margin / (DefenseRoll.StdDev * sqrt(2))`), the same two inputs
M4c intends to route every defence narration through. This is a
measurement only; no production code changed.

Command: `go run ./tools/melee_band_census`

Sample size: 50,000 contest rolls per matchup. Percentages are of
**defensive wins only** (the defender's win count is the row's last
column), not of all 50,000 swings; the win count differs per matchup
because how often the defender wins is itself a function of the score gap.

## Measured table

| matchup | rule | weak | normal | heavy | defensive wins |
|---|---|---|---|---|---|
| defender outclassed (100 vs 60) | old (z-score) | 59.8% | 34.1% | 6.1% | 7247 |
| defender outclassed (100 vs 60) | new (crit+margin) | 95.3% | 4.7% | 0.0% | 7247 |
| defender behind (100 vs 85) | old (z-score) | 46.1% | 48.0% | 6.0% | 15358 |
| defender behind (100 vs 85) | new (crit+margin) | 67.3% | 31.8% | 1.0% | 15358 |
| even (100 vs 100) | old (z-score) | 53.3% | 43.1% | 3.6% | 24868 |
| even (100 vs 100) | new (crit+margin) | 46.1% | 49.9% | 4.0% | 24868 |
| defender ahead (85 vs 100) | old (z-score) | 63.3% | 34.0% | 2.7% | 36122 |
| defender ahead (85 vs 100) | new (crit+margin) | 23.8% | 61.6% | 14.5% | 36122 |
| defender dominant (60 vs 100) | old (z-score) | 69.0% | 28.7% | 2.3% | 43608 |
| defender dominant (60 vs 100) | new (crit+margin) | 0.3% | 12.3% | 87.4% | 43608 |

## Which matchups move most

The two outclassed-defender rows barely differ in win count but move hard
in the opposite direction of intuition: under the old rule a defender who
is losing overall still reads as "heavy" 6.1% of the time on the rare wins
they do get, because a self-relative z-score only asks whether that one
roll beat their own mean, not whether it beat a strong attacker. The new
rule correctly crushes that matchup's defensive wins into "weak" (95.3%,
0.0% heavy), because most of those wins are floor-granted rather than
decisive: `DefenseRoll.Floored` is true for a large share of them, and the
new rule's crit gate explicitly zeroes the normalized margin whenever the
outcome is floor-granted, which the old rule has no equivalent check for.

The two defender-ahead rows move the opposite direction and by the widest
margin of the five: "defender ahead" goes from 2.7% heavy under the old
rule to 14.5% under the new rule, and "defender dominant" goes from 2.3%
heavy to 87.4% heavy. A defender who is actually winning by a wide score
gap is rarely allowed to read as decisive today, because self-relative
z-score cannot see the gap at all; the new rule reads that dominance
directly off the margin and reports it as heavy far more often, which is
the intended fix.

The "even" matchup is the one row pair that stays close in every band
(53.3/43.1/3.6 old vs 46.1/49.9/4.0 new), which is expected: at parity the
attacker and defender scores are equal, so the self-relative z-score and
the normalized margin are measuring nearly the same thing.

## Sabotage check (tool self-test)

Per the task's Step 3, the NEW rule's `norm` assignment was temporarily
changed to `norm = res.DefenseRoll.ZScore` (aliasing it to the OLD rule's
input) and the tool rerun. Non-floored defensive wins then classify
identically under both rules, since both switches compare the same
z-score against the same 2.0/0.5 thresholds. A residual gap remained on
floor-heavy matchups (most visibly "defender outclassed", where most
defensive wins are floor-granted): the new rule's `!res.Floored` crit gate
still forces those wins to "weak" regardless of what `norm` holds, while
the old rule has no such gate and classifies a floor-granted win by its
real (and sometimes high) self-relative z-score. That gap is intrinsic to
the new rule's structure, not evidence the sabotage failed to alias the
inputs; it confirms the tool is sensitive to the input source rather than
silently printing the same numbers twice. The change was reverted before
committing; `gofmt -l tools/melee_band_census/main.go` prints nothing on
the reverted file.
