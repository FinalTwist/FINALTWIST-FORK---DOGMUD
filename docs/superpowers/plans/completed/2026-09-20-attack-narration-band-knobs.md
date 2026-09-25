# Attack Narration Band Knobs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The two authorable cutoffs that decide how hard a swing reads, Normal
at 30 and Heavy at 75 percent of expected damage, become balance knobs shipping
at today's values, and the band rule gains the test it has never had.

**Architecture:** `items.GetAttackMessage` maps a damage percentage to one of
five authored pools. Two of its five cutoffs are feel dials (30, 75); the other
three are structural (101 is Critical and is already pinned to the crit flag by
`combat.attackMessagePct`, and 0 is Miss). This slice moves the two dials into
`Balance`, leaves the structural boundaries in Go, and pins the whole mapping
with a boundary test first. Nothing about damage changes: the band only picks
which pool narrates it.

**Ordering:** this slice lands BEFORE M4c (owner ruling 2026-09-20). It is
independent of M4c: attack bands read damage magnitude, defence bands read
contest outcome, and the two functions share no code.

**Tech Stack:** Go, `internal/items` (band selection, authored pools),
`internal/configs` (balance knobs), `internal/combat` (the one production
caller), golden files under `internal/narration/testdata/stores`.

---

## Facts verified against source (master `c7110dcc7`, 2026-09-20)

| Fact | Value | Source |
|---|---|---|
| Band function | `GetAttackMessage(subType, pctDamage)`: `>=101` Critical, `>=75` Heavy, `>=30` Normal, `>=1` Weak, else Miss | `internal/items/attack_messages.go:297-311` |
| Production callers | **exactly one**, plus the function's own Generic fallback | `internal/combat/combat_helpers.go:1599`; `attack_messages.go:328` |
| What the percentage is | `ceil(attackTargetDamage / sdp.dmgMean * 100)`: damage dealt against the EXPECTED damage of that swing against that target, not a share of the target's health | `internal/combat/combat_helpers.go:1528-1531` |
| What `dmgMean` includes | mitigation, stat mod bonus, weapon multiplier, prone penalty, mutation multiplier, warcry multiplier | `combat_helpers.go:455-494` |
| The clamp above it | `attackMessagePct(pct, isCrit)`: a crit is forced to 101, a non-crit is capped at 100, so the Critical pool is used exactly when the swing crit | `combat_helpers.go:1496-1507` |
| Why that clamp exists | without it roughly half of clean hits drew crit-worded text with no banner, and a mitigated real crit drew weak-worded text under one | its docstring, `combat_helpers.go:1489-1495` |
| It is tested | yes, a table test | `internal/combat/attack_message_pct_test.go` |
| 🔴 **The band rule is NOT tested** | grep for `GetAttackMessage` across `*_test.go` returns **only three comments** about seeding the map (hooks tests), no call | `internal/hooks/NewRound_DoCombat_HiddenDefender_test.go:57`, `NewRound_DoCombat_routing_test.go:165`, `progression_duplication_test.go:232` |
| 🔴 **No golden exercises it either** | `buildCombatMessagesGolden` walks `items.Intensity` values directly through `GetPreAttackMessage`; it never calls `GetAttackMessage` | `internal/narration/snapshot_test.go:328-400` |
| Authored pools | 20 subtype files, keys `prepare / wait / miss / weak / normal / heavy / critical / fumble` | `_datafiles/world/dogmud/combat-messages/*.yaml` |
| Tier structure inside a band | beginner/expert/master is a cumulative pool UNION, not a band | `attack_messages.go:99-104` |
| Selected by branch, not band | `prepare`, `wait`, `fumble`, `coupdegrace` come from `GetPreAttackMessage` | `combat_helpers.go:1572-1586` |
| `attack_messages.go` imports | `fmt`, `sort`, `internal/narration`: **`configs` must be added** (no cycle, `items` imports it elsewhere) | `attack_messages.go:1-8`, `internal/items/material_tier.go:3` |
| Int knob precedent | `ConcentrationDamageThresholdPct ConfigInt` | `internal/configs/config.balance.go:106` |
| Reject-zero precedent | `ContestFloor <= 0` reverts to 0.125 because **test binaries never load config.yaml** | `config.balance.misc.go:169-172` |

### Design decisions this plan takes

1. **Two knobs, not five.** 101 and 0 are structural: `attackMessagePct` already
   guarantees 101 means "this swing crit", and 0 means no damage landed. Making
   either authorable would let config break the crit banner pairing that
   `attackMessagePct`'s docstring exists to protect. They stay in Go with a
   comment saying so.
2. **Percent units, `ConfigInt`.** `pctDamage` is an `int` percentage; a float
   knob would invite a 0.75 that silently means "below 1 percent" and lands
   every hit in Weak.
3. **Ordered validation.** Normal must be below Heavy and both within 1..100. If
   the pair is inverted or out of range, BOTH revert to their defaults rather
   than one being clamped into the other, so a typo cannot half-apply.
4. **No patch note.** Defaults equal today's literals, so no player sees a
   different line. The knob blocks in `config.yaml` are the documentation.

---

## File Structure

**Created**
- `internal/items/attack_band_test.go`: the boundary test the rule has never had.
- `internal/configs/config_attack_band_test.go`: validation tests.

**Modified**
- `internal/items/attack_messages.go`: band cutoffs read from config.
- `internal/configs/config.balance.go`: two field declarations.
- `internal/configs/config.balance.combat.go`: one validation block.
- `_datafiles/config.yaml`: two documented knob blocks (**skip-worktree, see Task 4**).
- `internal/items/context.md`: the band function's contract.
- `docs/superpowers/specs/2026-09-17-messaging-m4-flip-design.md`: the Bands table gains the row it was missing.
- `docs/README.md`: this plan.

---

## Task 1: Pin the band rule before touching it

**Why first:** the rule has never had a test. A knob added under an
untested mapping cannot be shown to have preserved it.

**Files:**
- Create: `internal/items/attack_band_test.go`

- [ ] **Step 1: Write the boundary test**

```go
package items

import "testing"

// bandLabelAttackFixture gives Weak, Normal, Heavy, Critical and Miss one
// variant each, naming itself, so a test can assert WHICH POOL GetAttackMessage
// selected without depending on authored prose or on production randomness (a
// one-element pool has one possible pick).
func bandLabelAttackFixture() map[ItemSubType]*WeaponAttackMessageGroup {
	mk := func(label string) SkillTieredMessages {
		pool := MessageOptions{ItemMessage(label)}
		return SkillTieredMessages{Beginner: pool, Expert: pool, Master: pool}
	}
	opts := AttackIntensity{}
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
```

⚠️ `AttackIntensity`, `AttackOptions`, `TogetherMessages`, `SkillTieredMessages`
and `WeaponAttackMessageGroup` are the real type names as of this plan, taken
from `attack_messages.go` and `MinimalCombatMessageFixture`
(`internal/items/test_helpers_combat.go:83-100`). Confirm each against the file
before writing, and copy the shape of `MinimalCombatMessageFixture` where they
differ. Do not guess a field name.

- [ ] **Step 2: Run it**

Run: `go test ./internal/items/ -run TestGetAttackMessageBandBoundaries -v`

Expected: PASS, nine cases.

- [ ] **Step 3: Prove it can fail**

Temporarily change `attack_messages.go:305` from `pctDamage >= 30` to
`pctDamage >= 40`. Run the same command.

Expected: **FAIL**, naming `pct 30 selected WEAK, want NORMAL` and `pct 39`
untouched (the case list has no 39, which is fine: one red case is proof). Revert
and rerun; expected PASS. 🔴 A null probe must be proven capable of failing:
three false passes on this project so far.

- [ ] **Step 4: Commit**

```bash
git add internal/items/attack_band_test.go
git commit -m "test(items): pin GetAttackMessage's damage-percentage band boundaries

Nothing tested this mapping. The combat-messages golden walks Intensity values
through GetPreAttackMessage and never calls GetAttackMessage, and the only
test-file mentions of it are comments about seeding its map, so all five
cutoffs could have been retyped without a single red run.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Declare the two knobs

**Files:**
- Modify: `internal/configs/config.balance.go`
- Modify: `internal/configs/config.balance.combat.go`
- Create: `internal/configs/config_attack_band_test.go`

- [ ] **Step 1: Write the failing validation tests**

```go
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
```

⚠️ Confirm the validate entry point with
`grep -n "func (b \*Balance) Validate" internal/configs/config.balance*.go` and
call whatever `config_contestfloor_test.go` calls. Match it exactly.

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/configs/ -run TestAttackBandThresholds -v`

Expected: FAIL TO COMPILE, `unknown field AttackBandNormalThresholdPct`.

- [ ] **Step 3: Declare the fields**

In `internal/configs/config.balance.go`, beside the other combat knobs:

```go
	// AttackBandNormalThresholdPct and AttackBandHeavyThresholdPct are the two
	// authorable cutoffs deciding how hard a landed swing READS. The number
	// compared against them is the swing's damage as a percentage of the
	// EXPECTED damage of that swing against that target (post-mitigation,
	// post-modifier), not a share of the target's health.
	//
	// Narration only: no damage, hit chance or crit rate depends on either.
	//
	// The other three boundaries are structural and stay in Go. 101 means "this
	// swing crit" -- combat.attackMessagePct forces a crit to 101 and caps a
	// non-crit at 100, which is what keeps the crit-worded pool paired with the
	// *** banner -- and 0 means nothing landed.
	//
	// Zero is NOT legal on either: Go test binaries never load config.yaml, so a
	// permissive check would leave both at zero repo-wide and retire the Weak
	// band. An inverted or out-of-range pair reverts BOTH, never one.
	AttackBandNormalThresholdPct ConfigInt `yaml:"AttackBandNormalThresholdPct"` // Percent of expected damage at which a hit reads Normal (default 30)
	AttackBandHeavyThresholdPct  ConfigInt `yaml:"AttackBandHeavyThresholdPct"`  // Percent of expected damage at which a hit reads Heavy (default 75)
```

- [ ] **Step 4: Add the validation**

In `internal/configs/config.balance.combat.go`:

```go
	// Both cutoffs are validated as a PAIR: an inverted or out-of-range pair
	// reverts both, so a typo cannot half-apply and ship a band layout nobody
	// authored.
	if b.AttackBandNormalThresholdPct <= 0 || b.AttackBandHeavyThresholdPct > 100 ||
		b.AttackBandNormalThresholdPct >= b.AttackBandHeavyThresholdPct {
		b.AttackBandNormalThresholdPct = 30
		b.AttackBandHeavyThresholdPct = 75
	}
```

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./internal/configs/ -run TestAttackBandThresholds -v`

Expected: PASS, all four.

- [ ] **Step 6: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.combat.go internal/configs/config_attack_band_test.go
git commit -m "feat(config): the Normal and Heavy attack narration cutoffs are balance knobs

Declared and validated as a pair; not yet read. Nothing changes until the next
commit wires GetAttackMessage to them.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: `GetAttackMessage` reads the knobs

**Files:**
- Modify: `internal/items/attack_messages.go:297-311`
- Modify: `internal/items/attack_band_test.go` (one new test)

- [ ] **Step 1: Write the failing test that the knob actually moves the band**

Append to `internal/items/attack_band_test.go`:

```go
// TestGetAttackMessageBandHonoursConfig proves the cutoffs are READ, not merely
// declared. A knob nothing reads is the exact defect this repo has hit before:
// a check that cannot fail, shipped green.
func TestGetAttackMessageBandHonoursConfig(t *testing.T) {
	restore := SeedAttackMessagesForTest(bandLabelAttackFixture())
	defer restore()

	restoreCfg := configs.OverrideBalanceForTest(func(b *configs.Balance) {
		b.AttackBandNormalThresholdPct = 50
		b.AttackBandHeavyThresholdPct = 90
	})
	defer restoreCfg()

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
```

⚠️ **`configs.OverrideBalanceForTest` is an assumed helper.** Find the real one
first:

```bash
grep -rn "func.*ForTest\|func Override" internal/configs/testing_support.go internal/configs/overrides.go
```

Use whatever that package already exposes for temporarily replacing balance
values in a test, and follow how an existing test in another package does it
(`grep -rln "configs\..*ForTest" --include=*_test.go internal/ | head`). If no
such helper exists, add the smallest one to `internal/configs/testing_support.go`
that sets the two fields and returns a restore func, and say so in the commit
message. Do NOT write to the config singleton without a restore: the configs
package is concurrency-tested (`configs_concurrency_test.go`).

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/items/ -run TestGetAttackMessageBandHonoursConfig -v`

Expected: FAIL, `pct 49 with cutoffs 50/90 selected NORMAL, want WEAK`. The
literals are still in the function.

- [ ] **Step 3: Read the knobs**

In `internal/items/attack_messages.go`, add the import and replace the two
authorable comparisons, leaving 101 and the Miss floor as literals:

```go
func GetAttackMessage(subType ItemSubType, pctDamage int) AttackOptions {

	// 101 and the zero floor are STRUCTURAL and stay here: combat.attackMessagePct
	// forces a crit to 101 and caps a non-crit at 100, which is what pairs the
	// crit-worded pool with the *** banner, and 0 means nothing landed. Only the
	// two middle cutoffs are authorable.
	balance := configs.GetBalanceConfig()
	var intensity Intensity
	if pctDamage >= 101 {
		intensity = Critical
	} else if pctDamage >= int(balance.AttackBandHeavyThresholdPct) {
		intensity = Heavy
	} else if pctDamage >= int(balance.AttackBandNormalThresholdPct) {
		intensity = Normal
	} else if pctDamage >= 1 {
		intensity = Weak
	} else {
		intensity = Miss
	}
```

- [ ] **Step 4: Run both band tests**

Run: `go test ./internal/items/ -run "TestGetAttackMessageBand" -v`

Expected: PASS, both. The boundary test from Task 1 is unchanged and still
green: validated defaults equal the deleted literals.

- [ ] **Step 5: Prove nothing else moved**

```bash
go test ./internal/items/ ./internal/combat/ ./internal/narration/ ./internal/hooks/
go test .
git diff --stat internal/narration/testdata
```

Expected: tests PASS and **no output** from the `git diff --stat`. 🪤 `go test .`
at the repo ROOT is part of every gate on this project.

- [ ] **Step 6: Commit**

```bash
git add internal/items/attack_messages.go internal/items/attack_band_test.go
git commit -m "feat(items): GetAttackMessage bands on the configured cutoffs

The Normal and Heavy cutoffs come from Balance; 101 (crit) and the zero floor
(miss) stay in Go because attackMessagePct pins them. Defaults equal the
deleted literals, so no line a player sees changes.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Document the knobs in `config.yaml`

🪤 **`_datafiles/config.yaml` carries the git skip-worktree bit and desyncs in
both directions.** Build the commit from the committed blob, never from disk.

- [ ] **Step 1: Take the committed blob**

```bash
git show HEAD:_datafiles/config.yaml > /tmp/config_head.yaml
```

- [ ] **Step 2: Insert the block**

Edit `/tmp/config_head.yaml`, placing this beside the other combat narration
settings (after the `MinDefenseCritChance` line):

```yaml
  #
  # AttackBandNormalThresholdPct / AttackBandHeavyThresholdPct: how hard a
  #   landed swing READS. The number compared against them is the swing's
  #   damage as a percentage of the EXPECTED damage of that swing against that
  #   target, after mitigation and every modifier -- not a share of the
  #   target's health. At the defaults, a swing landing under a third of its
  #   expected damage reads weak, a third to three quarters reads normal, and
  #   above three quarters reads heavy.
  #   Narration only: no damage, hit chance or crit rate depends on either.
  #   The other boundaries are NOT authorable. A critical swing is pinned to
  #   the crit-worded pool in code so the wording always matches the banner,
  #   and a swing that lands nothing reads as a miss.
  #   Range: 1..100, and Normal must be BELOW Heavy. An inverted or
  #   out-of-range pair reverts BOTH to 30 and 75, because a half applied typo
  #   would ship a band layout nobody authored. Zero is not legal.
  AttackBandNormalThresholdPct: 30
  AttackBandHeavyThresholdPct: 75
```

- [ ] **Step 3: Stage the blob and restore skip-worktree**

```bash
git hash-object -w /tmp/config_head.yaml   # prints <sha>
git update-index --cacheinfo 100644,<sha>,_datafiles/config.yaml
git update-index --skip-worktree _datafiles/config.yaml
git ls-files -v _datafiles/config.yaml
```

🪤 **`--cacheinfo` CLEARS skip-worktree**, which is why the restore and the
check are separate steps. Expected from the last command: a lowercase `s`.

Apply the same edit to the working-tree file so the local server runs with the
knobs present. That is Claude's job, never a chore handed to the owner.

- [ ] **Step 4: Commit and verify the blob is what landed**

```bash
git commit -m "chore(config): document the attack narration band cutoffs

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>" -- _datafiles/config.yaml
git show HEAD:_datafiles/config.yaml | grep -n "AttackBand"
git ls-files -v _datafiles/config.yaml
```

Expected: both keys with their comment block, and a lowercase `s`.

- [ ] **Step 5: Boot with the real config**

```bash
go run . 2>&1 | tee /tmp/attackbands_boot.log
```

Then, each standalone (🪤 `grep -c` exits 1 on zero matches, and 🪤 **a failed
boot exits 0** because `main()` recovers at `main.go:135`):

```bash
grep -c "Server Ready" /tmp/attackbands_boot.log
grep -c "PANIC" /tmp/attackbands_boot.log
```

Expected: `Server Ready` present, `PANIC` absent. Kill the server by the PID you
started; the owner runs their own server on this machine.

---

## Task 5: Docs, verification, PR

**Files:**
- Modify: `internal/items/context.md`
- Modify: `docs/superpowers/specs/2026-09-17-messaging-m4-flip-design.md`
- Modify: `docs/README.md`

- [ ] **Step 1: Update `internal/items/context.md`**

State that `GetAttackMessage` bands on damage as a percentage of expected
damage; that its Normal and Heavy cutoffs are `Balance.AttackBandNormalThresholdPct`
and `Balance.AttackBandHeavyThresholdPct`; and that the Critical and Miss
boundaries are structural because `combat.attackMessagePct` pins them. Then:

```bash
python tools/context_md_audit.py
```

Expected: no findings for `internal/items`. 🪤 It cannot see struct fields or
prose claims, so it is a floor, not a ceiling.

- [ ] **Step 2: Add the missing row to the M4 spec's Bands table**

`docs/superpowers/specs/2026-09-17-messaging-m4-flip-design.md` has a `### Bands`
table that never listed this one. Add it, marked done with this PR's number:

```markdown
| Attack narration | `pctDamage` = damage / expected damage, clamped by `attackMessagePct` | Critical >= 101 and Miss at 0 structural; Normal and Heavy configurable (shipped 30 / 75) | `internal/items/attack_messages.go:297`; `internal/combat/combat_helpers.go:1496` |
```

Also correct the section's end-state sentence so it stops claiming the arc
leaves no hardcoded narration cutoff: two boundaries stay in Go deliberately,
and the M4c section should say so rather than overclaiming.

- [ ] **Step 3: Index this plan**

Add `docs/superpowers/plans/2026-09-20-attack-narration-band-knobs.md` to
`docs/README.md` with its full path and a one-line description.

- [ ] **Step 4: Sweep by meaning**

Run each standalone:

```bash
grep -rni "hardcoded" --include=*.md internal/items docs/superpowers/specs | grep -i "band\|cutoff\|narrat"
grep -rn "101\|>= 75\|>= 30" --include=*.md internal/items docs/superpowers
```

Every claim that the attack bands are hardcoded is now half false. Fix each.

- [ ] **Step 5: Full gate**

```bash
go build ./...
go test . ./...
golangci-lint run --new-from-rev=master
gofmt -l $(git diff --name-only master..HEAD | grep '\.go$')
```

Expected: build clean, tests PASS, **0 new lint issues**, and the `gofmt -l`
line prints **nothing**. 🪤 CI's `validate / test` job runs a gofmt gate that is
SEPARATE from golangci-lint and fails the whole job in about 30 seconds, before
a single test runs. `golangci-lint` passing says nothing about it. Run the
`gofmt -l` standalone: it exits 0 whether or not it names files, so chaining it
hides the result.

- [ ] **Step 6: Commit and open the PR**

```bash
git add internal/items/context.md docs/superpowers/specs/2026-09-17-messaging-m4-flip-design.md docs/README.md
git commit -m "docs(items): the attack narration bands, their knobs, and the two that stay in Go

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"

git push -u origin feature/messaging-attack-band-knobs
gh pr create --repo pruuk/DOGMud --base master \
  --title "Attack narration band cutoffs become balance knobs" \
  --body-file /tmp/attackbands_pr_body.md
```

🪤 **Every `gh` command carries `--repo pruuk/DOGMud`.** This repo is a fork and
`gh` defaults to the upstream parent.

The PR body must state, in this order:
1. **Nothing player-visible changes**: defaults equal today's literals, and the
   goldens are untouched (`git diff --stat internal/narration/testdata` empty).
2. **What the percentage actually measures**, because "percent damage" reads
   like a share of health and is not.
3. **Why only two of five cutoffs moved**, naming `attackMessagePct`'s crit
   pairing as the reason 101 stays in Go.
4. **That the band rule had no test or golden before this PR**, and what Task 1
   added.
5. That this slice lands before M4c by owner ruling and is independent of it.

---

## Self-review

**Coverage.** The owner's ruling was: the pctDamage cutoffs become their own
small slice, before M4c. Tasks 2 to 4 move the two authorable ones and document
them; Task 1 supplies the test the rule never had; Task 5 corrects the M4 spec's
Bands table, which is what made the gap invisible. The structural boundaries are
argued, not silently skipped.

**Placeholders.** None. Three steps carry an explicit "confirm this symbol
against the source first" instruction (`AttackIntensity` and its siblings, the
`Balance` validate entry point, the configs test-override helper) because those
are the three places where guessing a name would produce code that compiles
against nothing. Task 3 Step 1 names the fallback if the helper does not exist.

**Type consistency.** `AttackBandNormalThresholdPct` and
`AttackBandHeavyThresholdPct` are `ConfigInt`, declared in Task 2, validated as
a pair in Task 2 Step 4, read with `int(balance.X)` in Task 3, and authored as
plain integers in Task 4. `bandLabelAttackFixture` is defined in Task 1 Step 1
and reused by name in Task 3 Step 1.
