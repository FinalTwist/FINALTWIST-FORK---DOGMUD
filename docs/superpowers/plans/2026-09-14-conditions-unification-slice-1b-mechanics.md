# Conditions Unification Slice 1b (Mechanics) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the owner's post-slice-1 rulings: stacking bleeds with config-driven numbers, a spell dot that ticks every round, a Recovering swing cap that bites for players, and secret records removed from both condition lists.

**Architecture:** A new `stacking` flag gives a buff record a list of per-application stacks inside the one record, ticked and summed by `Buffs.Trigger` so neither round-tick path changes. Bleed numbers move from Go literals into fifteen balance knobs. The player round tick moves the stand attempt after the buff tick, the order mobs already use. One `BuffSpec.Listed` predicate and one `DisplayName` helper feed both the `conditions` command and `Char.Conditions`.

**Tech Stack:** Go, `gopkg.in/yaml.v2`, testify, the DOGMud playtest harness.

**Spec:** `docs/superpowers/specs/2026-09-14-conditions-unification-slice-1b-mechanics-design.md` (read its "Facts verified against source" table first).

**Branch:** `feature/conditions-unification-slice-1b-mechanics` (already created off master `04134f7e5`; the spec commits are on it).

---

## Facts verified against source for this plan (2026-09-14, tree `d3b5d84ea`)

| Fact | Where |
|---|---|
| `TriggersLeftExpired = 0`, `TriggersLeftUnlimited = 1000000000`; `Expired()` is `TriggersLeft <= 0` | `internal/buffs/buffs.go:10-11,44-46` |
| `Buffs.AddBuffMagnitude(buffId int, triggers int, magnitude float64) bool` at `:306-337`; `Buffs.Trigger` at `:415-464`; `RemoveBuff` at `:108-114` | `internal/buffs/buffs.go` |
| `BuffSpec.Validate` derives `RoundInterval` from `TriggerRate` at `:321-331`; `validateEffects` holds the `TickFromMagnitude` rules | `internal/buffs/buffspec.go:279-334`; `internal/buffs/effects.go:74-101` |
| `AllFlags` ends `SilentStart, Bleeding, Quiet` at `:143-145`; flag consts at `:87-98` | `internal/buffs/buffspec.go` |
| `VisibleNameDesc` at `buffspec.go:215-220`; tested by `TestBuffSpec_VisibleNameDesc` `buffspec_test.go:87-143`; production callers: `internal/usercommands/conditions.go:53`, `modules/gmcp/gmcp.Char.go:686` only | grep |
| `withSpecs(t, specs...)` test helper replaces the spec map for one test | `internal/buffs/effects_test.go:22-29` |
| `SeedConditionRecordsForTest` builds 121 and 122 with `TriggerRate: "3 rounds", RoundInterval: 3`; `TestShippedConditionRecordsMatchTestHelperShape` compares flags, effects, tick fields, `TriggerCount`, `RoundInterval` and texts against the shipped YAML | `internal/buffs/test_helpers.go:29-30`; `internal/buffs/records_test.go:58-116` |
| `TickTriggers` production callers: `spell_resolution.go:648,1674`; `combat_throttle.go:144`, `combat_rake.go:132`, `combat_maul.go:132`, `combat_hamstring.go:135`, `combat_drain.go:147,317`; `item_procs.go:214`. Test callers: `predator_hooks_test.go:59,105,134,171`; `hooks_test.go:926,976,1008,1058,1096`; `conditions_pin_test.go:33,59`; `ticks_test.go` | grep `TickTriggers(` |
| Tests that assert the three-round cadence: `TestRoundTick_BleedDamagesPlayer`, `_BleedDamagesMob`, `_BleedMinDamageOne`, `_BleedLineLandsOnExpiringTick` (`predator_hooks_test.go:47-189`); `TestRoundTick_PoisonDamage`, `_PoisonDamage_TriggerLineOnNonFinalTick`, `_TriggerLineLandsOnExpiringTick`, `TestDotProducerRecordsNegativeHarm_MobTarget`, `_PlayerTarget` (`hooks_test.go:915-1098`); `TestPin_PoisonTickKillsAndNamesTheCause`, `TestPin_BleedTickKillsAndNamesTheCause` (`conditions_pin_test.go:27-74`); `TestProcApplyCondition_Bleed` expects `TriggersLeft == 2` (`item_procs_test.go:235-261`) | read |
| Actions tests assert `BleedDmg >= 2`: `combat_drain_test.go:176,389,462`; `combat_throttle_test.go:167` | grep |
| Special-move knobs: fields `config.balance.go:244-251`, defaults `config.balance.combat.go:130-142`; `Balance.Validate()` calls `validateCombat()` first (`config.balance.go:1005-1013`); `loadConfig(document []byte) (Config, error)` (`configs.go:465`); `shippedConfigSource(t)` and `bytesContainsKey(src, key)` test helpers (`config_fumble_outpays_win_test.go:18-41`) | read |
| `config.yaml` blob: `RhetoricActionBaseConvictionCost: 4` appears once; `SpecialMoveCooldown: 4`; `HealthBase: 5`, `HealthPerVitality: 3`, `HealthPerStrength: 1`; index entry mode `100644` | `git show HEAD:_datafiles/config.yaml`; `git ls-files -s` |
| `UserRoundTick`: `AttemptRecovery` block `:244-258`, charm decrement `:260-262`, `Buffs.Trigger()` block `:264-395`; `MobRoundTick` ticks buffs `:131` before recovery `:165` | `internal/hooks/NewRound_UserRoundTick.go`; `NewRound_MobRoundTick.go` |
| Hooks test fixtures: `seedAllRegistries()`, `users.GetByUserId(1)`, `mobs.GetInstance(100)`, `drainPlain(userId)`, `countContaining(lines, substr)`; knock a player down with `u.Character.Position.TransitionToProne(position.ProneData{...}, state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward})` | `predator_hooks_test.go`; `flee_refusal_copy_test.go:38-39` |
| `TestRecoveringRecordExpiredByItsOwnTickCapsNothing` comment says the player cap is inert and a filed owner call | `internal/combat/hitroll_test.go:151-172` |
| `Char.Conditions` loop `gmcp.Char.go:646-720`; `GMCPCondition` `:769-777`; `conditionDurationLabel` `:781`; no test calls `GetCharNode` | read; grep |
| The `conditions` command builds a local `buffInfo` slice and renders template `character/conditions` | `internal/usercommands/conditions.go:22-67` |
| Every bleed producer's result carries `BleedDmg`; nothing outside `internal/actions` reads it | grep `BleedDmg` |
| `combat/ai.go` scores rake, maul, hamstring, drain, throttle with no bleeding-aware term, so the AI keeps re-applying | `internal/combat/ai.go:143-167`; grep `Bleed` |
| Only item with `apply_condition`: 40186 Thornwall Harness (`duration: 6`, `magnitude: 14`); its description names no number | `_datafiles/world/dogmud/items/materials-40000/40186-thornwall_harness.yaml` |
| Secret records: 81 Respawn Grace, 85 InfraredVision, 99 Alt Character Mob | grep `^secret: true` |
| Apply-path guard: `TestPlayerBuffsTravelTheEventPath` in package `main` at the repo root, allowlist keyed `"file|line"` | `buff_apply_path_guard_test.go:90-172,278` |
| Player-facing patch notes live in `docs/PATCH_NOTES.md`, newest first | read |
| Steppe wolves (205, 206, 215, 223) spawn in `ironwind_steppe` rooms 3013, 3015, 3016, 3019, 3024 and list `hamstring` | grep |

## Starting numbers (measured, then chosen; Task 1 ships them)

Strength centres on 100 for players, and the steppe wolves ship `strength: base: 95` plus a random share of a 70 to 120 stat pool, so 100 is the representative attacker. A Strength 100 / Vitality 100 player has `5 + 100×1 + 100×3 = 405` health.

Old single bleed hit at Strength 100 (one trigger): rake 8, drain 8, hamstring 10, throttle 10, maul 12.

| Move | Rounds | Divisor | Min | Per round @100 | Stack total | × old hit | Stacks at cooldown 4 |
|---|---|---|---|---|---|---|---|
| rake | 10 | 50 | 1 | 2 | 20 | 2.5 | 2 to 3 |
| drain | 10 | 50 | 1 | 2 | 20 | 2.5 | 2 to 3 |
| hamstring | 12 | 50 | 1 | 2 | 24 | 2.4 | 3 |
| throttle | 8 | 33 | 1 | 3 | 24 | 2.4 | 2 |
| maul | 12 | 35 | 1 | 2 | 24 | 2.0 | 3 |

A lone attacker using one move every cooldown settles at about 5 to 6 health per round of bleeding, about 1.2% to 1.5% of a 405-health pool. Thornwall Harness becomes `duration: 10`, `magnitude: 7` (70 per stack against the old two ticks of 14 = 28, so 2.5×).

A weak attacker hits the floor: below Strength 50 (35 for maul, 33 for throttle) a stack deals 1 per round. That is up to 4× the old floor hit for maul (12 against 3), accepted as a consequence of spreading damage over more rounds.

## File map

| File | Change |
|---|---|
| `internal/configs/config.balance.go` | 15 knob fields |
| `internal/configs/config.balance.combat.go` | 15 defaults |
| `internal/configs/config_bleed_stacks_test.go` | NEW: defaults, legal values, shipped keys, slice targets |
| `_datafiles/config.yaml` | 15 keys + comment (disk AND committed blob) |
| `internal/buffs/buffspec.go` | `Stacking` flag, `Validate` rule, later `Listed`, delete `VisibleNameDesc` |
| `internal/buffs/stacks.go` | NEW: `Stack`, `IsStacking`, `tickAmountFor`, `addStack`, `syncStacks`, `tickStacks`, later `DisplayName` |
| `internal/buffs/stacks_test.go` | NEW |
| `internal/buffs/buffs.go` | `Buff.Stacks`, `AddBuffMagnitude` branch, `Trigger` branch, `RemoveBuff` clears |
| `internal/buffs/ticks.go`, `ticks_test.go` | DELETE (Task 4) |
| `internal/buffs/test_helpers.go`, `records_test.go` | 121 and 122 shape |
| `_datafiles/world/dogmud/buffs/121-*.yaml`, `122-*.yaml` | cadence, `stacking` |
| `internal/hooks/spell_resolution.go` | two dot producers |
| `internal/actions/bleed.go` | NEW: `bleedPerRound` |
| `internal/actions/bleed_test.go` | NEW |
| `internal/actions/combat_{rake,maul,hamstring,drain,throttle}.go` | knobs, rounds |
| `internal/hooks/item_procs.go` | rounds |
| `_datafiles/world/dogmud/items/materials-40000/40186-thornwall_harness.yaml` | params |
| `internal/hooks/NewRound_UserRoundTick.go` | move the recovery block |
| `internal/usercommands/conditions.go` | `conditionEntries`, `Listed`, `DisplayName` |
| `modules/gmcp/gmcp.Char.go` | `buildConditionsPayload`, `Listed`, `DisplayName` |
| tests in `internal/hooks`, `internal/actions`, `internal/usercommands`, `modules/gmcp`, `internal/combat` | per task |
| `buff_apply_path_guard_test.go` | re-key producer lines |
| context.md files, comments, `docs/PATCH_NOTES.md`, `docs/README.md` | Task 7 |
| `tools/playtest/scenarios/conditions-slice-1b.yaml` + goals | NEW (Task 8) |

## Conventions for every task

- Run Go tests with Bash from the repo root: `go test ./internal/buffs/ -run 'TestName' -count=1`.
- **Null probe, every new assertion:** after it passes, break the exact line it pins, rerun, confirm red with a failure that names that behaviour, restore, confirm green. Record the probe (line broken, failure text) in the commit body.
- `git add` named paths only. Commit messages end with `Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>`; use a heredoc (`git commit -F - <<'EOF'`).
- No em or en dashes in prose, comments or commit messages.
- `gofmt -w` every Go file you touch before committing.

---

### Task 1: Fifteen bleed stack knobs

**Files:**
- Modify: `internal/configs/config.balance.go` (after the `RhetoricActionBaseConvictionCost` field, line 251)
- Modify: `internal/configs/config.balance.combat.go` (after the `RhetoricActionBaseConvictionCost` default, line 140-142)
- Create: `internal/configs/config_bleed_stacks_test.go`
- Modify: `_datafiles/config.yaml` (disk and committed blob)

- [ ] **Step 1: Write the failing tests**

Create `internal/configs/config_bleed_stacks_test.go`:

```go
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
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/configs/ -run 'BleedStack|BleedTuning' -count=1`
Expected: FAIL to compile, `b.RakeBleedRounds undefined`.

- [ ] **Step 3: Declare the fields**

In `internal/configs/config.balance.go`, directly after the `RhetoricActionBaseConvictionCost` field line, add:

```go
	RakeBleedRounds                  ConfigInt   `yaml:"RakeBleedRounds"`                  // Rounds one rake bleed stack lasts (default 10)
	RakeBleedStrengthDivisor         ConfigInt   `yaml:"RakeBleedStrengthDivisor"`         // A rake stack's per-round health loss is attacker Strength / this (default 50)
	RakeBleedMin                     ConfigInt   `yaml:"RakeBleedMin"`                     // Floor on a rake stack's per-round health loss (default 1)
	MaulBleedRounds                  ConfigInt   `yaml:"MaulBleedRounds"`                  // Rounds one maul bleed stack lasts (default 12)
	MaulBleedStrengthDivisor         ConfigInt   `yaml:"MaulBleedStrengthDivisor"`         // A maul stack's per-round health loss is attacker Strength / this (default 35)
	MaulBleedMin                     ConfigInt   `yaml:"MaulBleedMin"`                     // Floor on a maul stack's per-round health loss (default 1)
	HamstringBleedRounds             ConfigInt   `yaml:"HamstringBleedRounds"`             // Rounds one hamstring bleed stack lasts (default 12)
	HamstringBleedStrengthDivisor    ConfigInt   `yaml:"HamstringBleedStrengthDivisor"`    // A hamstring stack's per-round health loss is attacker Strength / this (default 50)
	HamstringBleedMin                ConfigInt   `yaml:"HamstringBleedMin"`                // Floor on a hamstring stack's per-round health loss (default 1)
	DrainBleedRounds                 ConfigInt   `yaml:"DrainBleedRounds"`                 // Rounds one drain bleed stack lasts, single and area drain alike (default 10)
	DrainBleedStrengthDivisor        ConfigInt   `yaml:"DrainBleedStrengthDivisor"`        // A drain stack's per-round health loss is attacker Strength / this (default 50)
	DrainBleedMin                    ConfigInt   `yaml:"DrainBleedMin"`                    // Floor on a drain stack's per-round health loss (default 1)
	ThrottleBleedRounds              ConfigInt   `yaml:"ThrottleBleedRounds"`              // Rounds one throttle bleed stack lasts (default 8)
	ThrottleBleedStrengthDivisor     ConfigInt   `yaml:"ThrottleBleedStrengthDivisor"`     // A throttle stack's per-round health loss is attacker Strength / this (default 33)
	ThrottleBleedMin                 ConfigInt   `yaml:"ThrottleBleedMin"`                 // Floor on a throttle stack's per-round health loss (default 1)
```

- [ ] **Step 4: Add the defaults**

In `internal/configs/config.balance.combat.go`, directly after the `RhetoricActionBaseConvictionCost` default block, add:

```go

	// ── BLEED STACKS (slice 1b) ──
	// Every guard is `< 1`. A zero stack length never ticks, a zero divisor
	// divides by zero, and a zero floor lets a weak attacker's stack tick for
	// nothing, so none of the three has a legal zero. An absent key reads 0
	// and takes the default, and each default is the shipped value.
	if b.RakeBleedRounds < 1 {
		b.RakeBleedRounds = 10
	}
	if b.RakeBleedStrengthDivisor < 1 {
		b.RakeBleedStrengthDivisor = 50
	}
	if b.RakeBleedMin < 1 {
		b.RakeBleedMin = 1
	}
	if b.MaulBleedRounds < 1 {
		b.MaulBleedRounds = 12
	}
	if b.MaulBleedStrengthDivisor < 1 {
		b.MaulBleedStrengthDivisor = 35
	}
	if b.MaulBleedMin < 1 {
		b.MaulBleedMin = 1
	}
	if b.HamstringBleedRounds < 1 {
		b.HamstringBleedRounds = 12
	}
	if b.HamstringBleedStrengthDivisor < 1 {
		b.HamstringBleedStrengthDivisor = 50
	}
	if b.HamstringBleedMin < 1 {
		b.HamstringBleedMin = 1
	}
	if b.DrainBleedRounds < 1 {
		b.DrainBleedRounds = 10
	}
	if b.DrainBleedStrengthDivisor < 1 {
		b.DrainBleedStrengthDivisor = 50
	}
	if b.DrainBleedMin < 1 {
		b.DrainBleedMin = 1
	}
	if b.ThrottleBleedRounds < 1 {
		b.ThrottleBleedRounds = 8
	}
	if b.ThrottleBleedStrengthDivisor < 1 {
		b.ThrottleBleedStrengthDivisor = 33
	}
	if b.ThrottleBleedMin < 1 {
		b.ThrottleBleedMin = 1
	}
```

- [ ] **Step 5: Add the block to config.yaml, disk AND committed blob**

`_datafiles/config.yaml` carries skip-worktree. Load the `dogmud-balance-config` skill before this step. The block goes directly after the line `  RhetoricActionBaseConvictionCost: 4` (it occurs once):

```yaml
  #
  # ── BLEED STACKS ──────────────────────────────────────────────────────────
  # Rake, maul, hamstring, drain and throttle bleed on a landed hit. Each hit
  # adds a STACK to the target's bleeding with its own timer: it lasts
  # <Move>BleedRounds rounds and takes Strength / <Move>BleedStrengthDivisor
  # health every round (never less than <Move>BleedMin). Stacks add up, and
  # bleeding keeps going after the fight ends.
  #
  # A lone attacker lands one special move per SpecialMoveCooldown rounds, so
  # the stacks it keeps alive settle near Rounds / cooldown: at the shipped
  # cooldown of 4, 8 rounds holds 2, 10 holds 2 to 3, 12 holds 3. A group
  # keeps more. TestShippedBleedTuningMeetsTheSliceTargets holds each stack to
  # 2 to 3 cooldowns long, and its total at Strength 100 to 1.5 to 3 times the
  # single bleed hit it replaced (owner ruling, 2026-09-14).
  RakeBleedRounds: 10
  RakeBleedStrengthDivisor: 50
  RakeBleedMin: 1
  MaulBleedRounds: 12
  MaulBleedStrengthDivisor: 35
  MaulBleedMin: 1
  HamstringBleedRounds: 12
  HamstringBleedStrengthDivisor: 50
  HamstringBleedMin: 1
  DrainBleedRounds: 10
  DrainBleedStrengthDivisor: 50
  DrainBleedMin: 1
  ThrottleBleedRounds: 8
  ThrottleBleedStrengthDivisor: 33
  ThrottleBleedMin: 1
```

1. Edit the disk copy with the Edit tool (anchor on `  RhetoricActionBaseConvictionCost: 4`).
2. Build the committed version from the blob, never from disk (Bash; `$SCRATCH` is the session scratchpad directory):

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git show HEAD:_datafiles/config.yaml > "$SCRATCH/config.blob.yaml"
```

3. Insert the same block into `$SCRATCH/config.blob.yaml` with the Edit tool, same anchor.
4. Stage it and restore the skip-worktree bit:

```bash
blob=$(git hash-object -w "$SCRATCH/config.blob.yaml")
git update-index --cacheinfo 100644,"$blob",_datafiles/config.yaml
git update-index --skip-worktree _datafiles/config.yaml
git ls-files -v _datafiles/config.yaml
git diff --cached --stat -- _datafiles/config.yaml
```

Expected: `S _datafiles/config.yaml`, and the cached diff is `1 file changed, 29 insertions(+)` with no deletions. Anything else means the blob copy picked up line-ending or local-only changes: `git restore --staged _datafiles/config.yaml`, re-apply the skip-worktree bit, redo from 2.

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./internal/configs/ -count=1`
Expected: PASS (the whole package, so no neighbouring config test broke).

- [ ] **Step 7: Null probes**

1. In `config.balance.combat.go` change `b.RakeBleedRounds = 10` to `= 11`: `TestBleedStackKnobs_AbsentKeysTakeTheDefaults` fails naming `RakeBleedRounds`. Restore.
2. On disk only, change `ThrottleBleedRounds: 8` to `20`: `TestShippedBleedTuningMeetsTheSliceTargets` fails naming throttle and `5.00 cooldowns`. Restore.
3. On disk only, rename `MaulBleedMin:` to `MaulBleedMinn:`: `TestShippedConfigNamesEveryBleedStackKnob` fails naming `MaulBleedMin`. Restore.

Confirm green again.

- [ ] **Step 8: Commit**

```bash
gofmt -w internal/configs/config.balance.go internal/configs/config.balance.combat.go internal/configs/config_bleed_stacks_test.go
git add internal/configs/config.balance.go internal/configs/config.balance.combat.go internal/configs/config_bleed_stacks_test.go
git commit -F - <<'EOF'
feat(configs): fifteen bleed stack knobs (rounds, strength divisor, floor per bleed move)

Nothing reads them yet (Task 4). Shipped values meet the slice 1b targets:
a stack lasts 2 to 3 special-move cooldowns and totals 1.5 to 3 times the old
single hit at Strength 100.

Null probes: <record the three here>

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
git ls-files -v _datafiles/config.yaml
```

The config.yaml change was already staged in Step 5, so it lands in this commit. Expected after commit: `S _datafiles/config.yaml`.

---

### Task 2: Stacking records in `internal/buffs`

**Files:**
- Create: `internal/buffs/stacks.go`
- Create: `internal/buffs/stacks_test.go`
- Modify: `internal/buffs/buffs.go` (`Buff` struct `:14-32`, `RemoveBuff` `:108-114`, `AddBuffMagnitude` `:306-337`, `Trigger` `:444-457`)
- Modify: `internal/buffs/buffspec.go` (flag consts `:87-98`, `AllFlags` `:143-145`, `Validate` `:321-333`)

- [ ] **Step 1: Write the failing tests**

Create `internal/buffs/stacks_test.go`:

```go
package buffs

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

func stackingSpec() *BuffSpec {
	return &BuffSpec{BuffId: 930, Name: "Gash", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 4,
		Flags: []Flag{Bleeding, Stacking}, TickPool: "health", TickFromMagnitude: true}
}

func heldOne(t *testing.T, bs *Buffs, id int) *Buff {
	t.Helper()
	held := bs.GetBuffs(id)
	if len(held) != 1 {
		t.Fatalf("want exactly one held record %d, got %d", id, len(held))
	}
	return held[0]
}

func TestStackingAddAppendsAStackInsteadOfOverwriting(t *testing.T) {
	withSpecs(t, stackingSpec())
	bs := New()
	if !bs.AddBuffMagnitude(930, 3, -2) || !bs.AddBuffMagnitude(930, 5, -3) {
		t.Fatal("both adds must land")
	}
	b := heldOne(t, &bs, 930)
	if want := []Stack{{RoundsLeft: 3, Amount: -2}, {RoundsLeft: 5, Amount: -3}}; !reflect.DeepEqual(b.Stacks, want) {
		t.Fatalf("Stacks = %+v, want %+v", b.Stacks, want)
	}
	if b.TriggersLeft != 5 {
		t.Fatalf("TriggersLeft = %d, want 5 (the longest stack)", b.TriggersLeft)
	}
	if b.TickAmount != -5 || b.Magnitude != -5 {
		t.Fatalf("TickAmount %d / Magnitude %v, want -5 / -5 (the sum of the stacks)", b.TickAmount, b.Magnitude)
	}
}

func TestStackingTriggerSumsDecrementsAndDropsStacks(t *testing.T) {
	withSpecs(t, stackingSpec())
	bs := New()
	bs.AddBuffMagnitude(930, 2, -2)
	bs.AddBuffMagnitude(930, 4, -3)

	type round struct {
		tick         int
		stacks       []Stack
		triggersLeft int
	}
	want := []round{
		{-5, []Stack{{1, -2}, {3, -3}}, 3},
		{-5, []Stack{{2, -3}}, 2},
		{-3, []Stack{{1, -3}}, 1},
		{-3, []Stack{}, 0},
	}
	for i, w := range want {
		fired := bs.Trigger()
		if len(fired) != 1 {
			t.Fatalf("round %d: want the record to fire once, got %d", i+1, len(fired))
		}
		b := fired[0]
		if b.TickAmount != w.tick {
			t.Fatalf("round %d: TickAmount = %d, want %d (every live stack summed)", i+1, b.TickAmount, w.tick)
		}
		if len(b.Stacks) != len(w.stacks) || (len(w.stacks) > 0 && !reflect.DeepEqual(b.Stacks, w.stacks)) {
			t.Fatalf("round %d: Stacks = %+v, want %+v", i+1, b.Stacks, w.stacks)
		}
		if b.TriggersLeft != w.triggersLeft {
			t.Fatalf("round %d: TriggersLeft = %d, want %d", i+1, b.TriggersLeft, w.triggersLeft)
		}
	}
	if !bs.List[0].Expired() {
		t.Fatal("the record must be expired once its last stack ends")
	}
	if fired := bs.Trigger(); len(fired) != 0 {
		t.Fatalf("an expired stacking record must not fire again, got %d", len(fired))
	}
}

func TestStackingZeroTriggersUsesTheSpecCount(t *testing.T) {
	withSpecs(t, stackingSpec())
	bs := New()
	bs.AddBuffMagnitude(930, 0, -1)
	if got := heldOne(t, &bs, 930).Stacks[0].RoundsLeft; got != 4 {
		t.Fatalf("RoundsLeft = %d, want the spec's triggercount 4", got)
	}
}

func TestStackingAmountFloorsToOneInSign(t *testing.T) {
	withSpecs(t, stackingSpec())
	bs := New()
	bs.AddBuffMagnitude(930, 2, -0.5)
	if got := heldOne(t, &bs, 930).Stacks[0].Amount; got != -1 {
		t.Fatalf("Amount = %d, want -1: a non-zero magnitude never snapshots to zero", got)
	}
}

// Nothing produces a stacking record with no stacks, but if one exists it must
// not reach the tick path: a zero TickAmount there falls back to tick_percent.
func TestStackingRecordWithNoStacksExpiresWithoutFiring(t *testing.T) {
	withSpecs(t, stackingSpec())
	bs := New()
	bs.List = append(bs.List, &Buff{BuffId: 930, TriggersLeft: 3, TickAmount: -5})
	bs.Validate(true)
	if fired := bs.Trigger(); len(fired) != 0 {
		t.Fatalf("a stacking record with no stacks must not fire, got %d", len(fired))
	}
	if !bs.List[0].Expired() {
		t.Fatal("and it must expire")
	}
}

func TestRemoveBuffClearsStacks(t *testing.T) {
	withSpecs(t, stackingSpec())
	bs := New()
	bs.AddBuffMagnitude(930, 3, -2)
	bs.RemoveBuff(930)
	if len(bs.List[0].Stacks) != 0 {
		t.Fatalf("RemoveBuff must clear the stacks, got %+v", bs.List[0].Stacks)
	}
}

// A cancel path can expire a record without clearing its stacks (HasFlag with
// expire, CancelBuffsWithFlag). A new stack landing before the prune must not
// resurrect the old ones.
func TestStackingAddOnAnExpiredRecordStartsFresh(t *testing.T) {
	withSpecs(t, stackingSpec())
	bs := New()
	bs.AddBuffMagnitude(930, 3, -2)
	bs.List[0].TriggersLeft = TriggersLeftExpired
	bs.AddBuffMagnitude(930, 5, -3)
	if want := []Stack{{RoundsLeft: 5, Amount: -3}}; !reflect.DeepEqual(bs.List[0].Stacks, want) {
		t.Fatalf("Stacks = %+v, want %+v", bs.List[0].Stacks, want)
	}
}

func TestNonStackingRecordStillOverwrites(t *testing.T) {
	withSpecs(t, &BuffSpec{BuffId: 931, Name: "Sting", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 4,
		TickPool: "health", TickFromMagnitude: true})
	bs := New()
	bs.AddBuffMagnitude(931, 3, -2)
	bs.AddBuffMagnitude(931, 5, -3)
	b := heldOne(t, &bs, 931)
	if b.TriggersLeft != 5 || b.TickAmount != -3 || len(b.Stacks) != 0 {
		t.Fatalf("a non-stacking record overwrites: TriggersLeft %d TickAmount %d Stacks %+v, want 5 -3 []", b.TriggersLeft, b.TickAmount, b.Stacks)
	}
}

func TestValidateRefusesStackingWithoutTickFromMagnitude(t *testing.T) {
	s := &BuffSpec{BuffId: 932, Name: "Bad", TriggerRate: "1 round", TriggerCount: 1, Flags: []Flag{Stacking}, TickPool: "health", TickPercent: -1}
	if err := s.Validate(); err == nil {
		t.Fatal("a stacking record without tick_from_magnitude must be refused")
	}
}

func TestValidateRefusesStackingSlowerThanOneRound(t *testing.T) {
	s := &BuffSpec{BuffId: 933, Name: "Bad", TriggerRate: "3 rounds", TriggerCount: 1, Flags: []Flag{Stacking}, TickPool: "health", TickFromMagnitude: true}
	if err := s.Validate(); err == nil {
		t.Fatal("a stacking record must tick every round: a stack counts rounds")
	}
}

func TestValidateAcceptsAWellFormedStackingRecord(t *testing.T) {
	if err := stackingSpec().Validate(); err != nil {
		t.Fatalf("a one-round tick_from_magnitude stacking record is legal: %v", err)
	}
}

func TestStacksRoundTripThroughYaml(t *testing.T) {
	in := []*Buff{{BuffId: 930, TriggersLeft: 5, TickAmount: -5, Magnitude: -5,
		Stacks: []Stack{{RoundsLeft: 3, Amount: -2}, {RoundsLeft: 5, Amount: -3}}}}
	out, err := yaml.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var back []*Buff
	if err := yaml.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	if len(back) != 1 || !reflect.DeepEqual(back[0].Stacks, in[0].Stacks) {
		t.Fatalf("stacks must survive a save: got %+v from\n%s", back, out)
	}
}

// The equilibrium the owner asked for: one stack every 4 rounds (the shipped
// SpecialMoveCooldown), ticked every round. After warm-up the live count is
// exactly Rounds/4 when that divides evenly, and moves between floor and ceil
// when it does not.
func TestStackEquilibriumAtTheShippedCooldown(t *testing.T) {
	cases := []struct{ rounds, lo, hi int }{{8, 2, 2}, {10, 2, 3}, {12, 3, 3}}
	for _, c := range cases {
		withSpecs(t, stackingSpec())
		bs := New()
		for r := 0; r < 40; r++ {
			bs.Trigger()
			if r%4 == 0 {
				bs.AddBuffMagnitude(930, c.rounds, -2)
			}
			if r < 12 {
				continue
			}
			if n := len(bs.List[0].Stacks); n < c.lo || n > c.hi {
				t.Fatalf("stack length %d, round %d: %d live stacks, want %d to %d", c.rounds, r, n, c.lo, c.hi)
			}
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/buffs/ -run 'Stack|RemoveBuffClears|NonStacking' -count=1`
Expected: FAIL to compile, `undefined: Stacking` / `b.Stacks undefined`.

- [ ] **Step 3: Add the flag**

In `internal/buffs/buffspec.go`, after the `Quiet Flag = `quiet`` line add:

```go
	// Stacking marks a tick record where every application is its own stack
	// with its own timer, instead of refreshing the one instance. The record
	// ticks the sum of its live stacks once a round and ends with its longest
	// stack. It requires tick_from_magnitude and a one-round triggerrate. The
	// bleed record carries it (slice 1b, 2026-09-14).
	Stacking Flag = `stacking`
```

and add `Stacking,` after `Quiet,` in `AllFlags`.

- [ ] **Step 4: Add the validation rule**

In `BuffSpec.Validate`, replace the closing `return nil` of the function (after the `if b.TriggerRate != "" { ... }` block) with:

```go
	// A stack's amount IS the applier's magnitude and a stack counts rounds,
	// so a stacking record must be tick_from_magnitude and tick every round.
	// Checked after RoundInterval is derived above; an empty triggerrate
	// leaves it 0 and is refused too.
	if b.IsStacking() {
		if !b.TickFromMagnitude {
			return fmt.Errorf("buffId %d (%s) is stacking without tick_from_magnitude; a stack's amount is the applier's magnitude", b.BuffId, b.Name)
		}
		if b.RoundInterval != 1 {
			return fmt.Errorf("buffId %d (%s) is stacking with triggerrate %q; a stack counts rounds, so the record must tick every round", b.BuffId, b.Name, b.TriggerRate)
		}
	}

	return nil
```

- [ ] **Step 5: Create `internal/buffs/stacks.go`**

```go
package buffs

import "slices"

// Stack is one application of a stacking record: its own remaining rounds and
// its own signed per-round amount (negative harms). See the Stacking flag.
type Stack struct {
	RoundsLeft int `yaml:"roundsleft"`
	Amount     int `yaml:"amount"`
}

// IsStacking reports whether the spec carries the Stacking flag.
func (b *BuffSpec) IsStacking() bool {
	return slices.Contains(b.Flags, Stacking)
}

// tickAmountFor converts an applier's magnitude into the signed per-round
// amount a tick_from_magnitude record lands. It truncates toward zero, except
// that a non-zero magnitude which truncates to zero becomes 1 in its sign: a
// zero snapshot would tick for nothing forever, because the round tick's
// fallback recomputes from tick_percent, which a tick_from_magnitude record
// may not set.
func tickAmountFor(magnitude float64) int {
	amt := int(magnitude)
	if amt == 0 && magnitude != 0 {
		if magnitude < 0 {
			return -1
		}
		return 1
	}
	return amt
}

// addStack appends one stack to a stacking record, creating the record on the
// first stack. rounds 0 means the spec's triggercount. An expired record that
// has not been pruned yet (a cancel path expires without clearing) starts
// fresh rather than resurrecting its old stacks.
func (bs *Buffs) addStack(spec *BuffSpec, rounds int, magnitude float64) bool {
	if idx, ok := bs.buffIds[spec.BuffId]; ok && bs.List[idx].Expired() {
		bs.List[idx].Stacks = nil
	}
	if !bs.AddBuffScaled(spec.BuffId, 1.0) {
		return false
	}
	idx, ok := bs.buffIds[spec.BuffId]
	if !ok {
		return false
	}
	if rounds <= 0 {
		rounds = spec.TriggerCount
	}
	b := bs.List[idx]
	b.Stacks = append(b.Stacks, Stack{RoundsLeft: rounds, Amount: tickAmountFor(magnitude)})
	b.syncStacks()
	return true
}

// syncStacks derives the record-level fields every other reader uses from the
// live stacks: TriggersLeft is the longest stack (so Expired, GetDurations,
// the prune pass and both condition lists see a record that lives as long as
// its longest stack), and TickAmount and Magnitude are the sum.
func (b *Buff) syncStacks() {
	longest, sum := 0, 0
	for _, s := range b.Stacks {
		longest = max(longest, s.RoundsLeft)
		sum += s.Amount
	}
	b.TriggersLeft = longest
	b.TickAmount = sum
	b.Magnitude = float64(sum)
}

// tickStacks lands one round of a stacking record. TickAmount becomes this
// round's amount, the sum of every live stack, which is what both round-tick
// paths read after Trigger returns. Each stack then loses a round and the
// spent ones drop; TriggersLeft becomes the longest remaining stack and
// Magnitude the sum still to come. Returns false, having expired the record,
// when there are no stacks to tick.
func (b *Buff) tickStacks() bool {
	if len(b.Stacks) == 0 {
		b.TriggersLeft = TriggersLeftExpired
		return false
	}
	landed := 0
	live := b.Stacks[:0]
	for _, s := range b.Stacks {
		landed += s.Amount
		s.RoundsLeft--
		if s.RoundsLeft > 0 {
			live = append(live, s)
		}
	}
	b.Stacks = live
	b.syncStacks()
	b.TickAmount = landed
	return true
}
```

- [ ] **Step 6: Wire it into `buffs.go`**

1. `Buff` struct: after the `Magnitude float64 ...` field add

```go

	// Stacks holds a stacking record's applications, each with its own
	// timer. Empty for every other record. See stacks.go.
	Stacks []Stack `yaml:"stacks,omitempty"`
```

2. `RemoveBuff`: inside the `if`, after `bs.List[index].TriggersLeft = TriggersLeftExpired`, add `bs.List[index].Stacks = nil`.

3. `AddBuffMagnitude`: make the first statements of the body

```go
	if spec := GetBuffSpec(buffId); spec != nil && spec.IsStacking() {
		return bs.addStack(spec, triggers, magnitude)
	}
```

and replace the `amt := int(magnitude)` ... `bs.List[idx].TickAmount = amt` block (keep the comment above it, shortened to one line pointing at `tickAmountFor`) with

```go
		// The magnitude IS the signed per-round amount; see tickAmountFor.
		bs.List[idx].TickAmount = tickAmountFor(magnitude)
```

Add one sentence to the doc comment: `A stacking record (see the Stacking flag) appends a stack instead of overwriting; triggers is then that stack's rounds.`

4. `Trigger`: replace

```go
				if b.RoundCounter%buffInfo.RoundInterval == 0 {
					// It cannot be pruned unless it is triggered
					triggeredBuffs = append(triggeredBuffs, b)
					if b.TriggersLeft != TriggersLeftUnlimited {
						b.TriggersLeft--
					} else {
						// If unimited, reset the counter to prevent some future overflow
						b.RoundCounter = 0
					}
				}
```

with

```go
				if b.RoundCounter%buffInfo.RoundInterval == 0 {
					if buffInfo.IsStacking() {
						// A stacking record ticks its stacks and derives
						// TriggersLeft from them; see tickStacks.
						if b.tickStacks() {
							triggeredBuffs = append(triggeredBuffs, b)
						}
					} else {
						// It cannot be pruned unless it is triggered
						triggeredBuffs = append(triggeredBuffs, b)
						if b.TriggersLeft != TriggersLeftUnlimited {
							b.TriggersLeft--
						} else {
							// If unimited, reset the counter to prevent some future overflow
							b.RoundCounter = 0
						}
					}
				}
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `go test ./internal/buffs/ -count=1`
Expected: PASS for the whole package (including `TestAllFlagsNamesEveryDeclaredConstant` and the existing `AddBuffMagnitude` tests).

- [ ] **Step 8: Null probes**

1. In `addStack`, replace the append with `b.Stacks = []Stack{{RoundsLeft: rounds, Amount: tickAmountFor(magnitude)}}`: `TestStackingAddAppendsAStackInsteadOfOverwriting` fails. Restore.
2. In `tickStacks`, change `if s.RoundsLeft > 0` to `if s.RoundsLeft >= 0`: `TestStackingTriggerSumsDecrementsAndDropsStacks` fails at round 1. Restore.
3. In `tickStacks`, delete `b.TickAmount = landed`: the same test fails at round 1 (`-3`, want `-5`). Restore.
4. Delete the `bs.List[idx].Stacks = nil` line in `addStack`: `TestStackingAddOnAnExpiredRecordStartsFresh` fails. Restore.
5. In `Trigger`, move the append above `if b.tickStacks()`: `TestStackingRecordWithNoStacksExpiresWithoutFiring` fails. Restore.
6. Delete the `RoundInterval != 1` check: `TestValidateRefusesStackingSlowerThanOneRound` fails. Restore.

- [ ] **Step 9: Commit**

```bash
gofmt -w internal/buffs/stacks.go internal/buffs/stacks_test.go internal/buffs/buffs.go internal/buffs/buffspec.go
git add internal/buffs/stacks.go internal/buffs/stacks_test.go internal/buffs/buffs.go internal/buffs/buffspec.go
git commit -F - <<'EOF'
feat(buffs): stacking records (each application its own timer, one summed tick per round)

No record carries the flag yet (Task 4 puts it on Bleeding).

Null probes: <record the six here>

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 3: The spell dot ticks every round

**Files:**
- Modify: `_datafiles/world/dogmud/buffs/121-poisoned.yaml` (confirm the exact filename with `ls _datafiles/world/dogmud/buffs/121-*`)
- Modify: `internal/buffs/test_helpers.go:29`
- Modify: `internal/hooks/spell_resolution.go:636-648` and `:1662-1674`
- Modify: `internal/hooks/hooks_test.go:910-1098`, `internal/hooks/conditions_pin_test.go:14-48`
- Modify: `buff_apply_path_guard_test.go:158-162`

- [ ] **Step 1: Rewrite the cadence tests to the new behaviour (they will fail)**

In `internal/hooks/hooks_test.go`:

Replace the doc comment and body of `TestRoundTick_PoisonDamage` (lines 910-960) with:

```go
// TestRoundTick_PoisonDamage pins slice 1b's cadence: the spell dot record
// (buff 121) ticks every round (owner ruling 2026-09-14; it used to keep the
// old AutoHeal hook's every-third-round cadence). A one-round record lands its
// harm and its line on the first round tick, which is also its last.
func TestRoundTick_PoisonDamage(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer buffs.SeedConditionRecordsForTest()()

	u1 := users.GetByUserId(1)
	// The door validates on add, and Validate() recomputes HealthMax from
	// stats/balance config, clobbering the fixture's raw HealthMax.Value.
	// Seeding Base keeps HealthMax comfortably above the 80 this test needs.
	u1.Character.HealthMax.Base = 100
	u1.Character.Health = 80
	_ = u1.Character.AddBuffMagnitude(buffs.BuffIdPoisoned, 1, -5, "test")
	u1.Character.Health = 80
	drainPlain(1)

	UserRoundTick(events.NewRound{RoundNumber: 1})
	assert.Equal(t, 75, u1.Character.Health, "the record ticks every round: the first round tick lands the harm")
	assert.Equal(t, 1, countContaining(drainPlain(1), "The poison burns through your veins!"),
		"the first, and here final, trigger sends the flavour line")

	// Cross-hook pin: NewRound_AutoHeal.go's hand-rolled poison block was
	// deleted in slice 1 (the record's own tick path in UserRoundTick applies
	// the harm). If it were revived in record-reading form, this same round's
	// regen-gate call would double the damage already applied above instead of
	// only adding a small regen.
	result := AutoHeal(events.NewRound{RoundNumber: 3})
	require.Equal(t, events.Continue, result)
	hpr := u1.Character.HealthPerRound()
	assert.GreaterOrEqual(t, u1.Character.Health, 75, "no extra poison harm from AutoHeal")
	assert.Less(t, u1.Character.Health, 80, "at most one small regen tick (HealthPerRound=%d)", hpr)

	healthAfterAutoHeal := u1.Character.Health
	UserRoundTick(events.NewRound{RoundNumber: 2})
	assert.Equal(t, healthAfterAutoHeal, u1.Character.Health, "the record expired at round 1; a second tick is a no-op")
}
```

Replace `TestRoundTick_PoisonDamage_TriggerLineOnNonFinalTick` (lines 962-991) with:

```go
// TestRoundTick_PoisonDamage_TriggerLineOnNonFinalTick: a 2-round record's
// FIRST trigger (round 1) is not the one that expires it, so the line sends;
// its second (round 2) lands the harm, expires the record, and sends the line
// too. Consecutive rounds: the dot lands every round (slice 1b).
func TestRoundTick_PoisonDamage_TriggerLineOnNonFinalTick(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer buffs.SeedConditionRecordsForTest()()

	u1 := users.GetByUserId(1)
	u1.Character.HealthMax.Base = 100
	u1.Character.Health = 80
	_ = u1.Character.AddBuffMagnitude(buffs.BuffIdPoisoned, 2, -5, "test")
	u1.Character.Health = 80
	drainPlain(1)

	UserRoundTick(events.NewRound{RoundNumber: 1})
	assert.Equal(t, 75, u1.Character.Health, "the first of two triggers lands on round 1")
	assert.Equal(t, 1, countContaining(drainPlain(1), "The poison burns through your veins!"),
		"a trigger that is not also the last one sends its flavour line")

	UserRoundTick(events.NewRound{RoundNumber: 2})
	assert.Equal(t, 70, u1.Character.Health, "the second trigger lands on the very next round and expires the record")
	assert.Equal(t, 1, countContaining(drainPlain(1), "The poison burns through your veins!"),
		"the final, expiring trigger still sends the flavour line")
}
```

Replace `TestRoundTick_TriggerLineLandsOnExpiringTick` (lines 993-1017) with:

```go
// TestRoundTick_TriggerLineLandsOnExpiringTick is the ONE-trigger sibling of
// TestRoundTick_PoisonDamage_TriggerLineOnNonFinalTick: a record whose only
// trigger is also its last must land both the harm and the flavour line on
// that same, already-expiring tick.
func TestRoundTick_TriggerLineLandsOnExpiringTick(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer buffs.SeedConditionRecordsForTest()()

	u1 := users.GetByUserId(1)
	u1.Character.HealthMax.Base = 100
	u1.Character.Health = 80
	_ = u1.Character.AddBuffMagnitude(buffs.BuffIdPoisoned, 1, -5, "test")
	drainPlain(1)

	UserRoundTick(events.NewRound{RoundNumber: 1})
	assert.Equal(t, 75, u1.Character.Health, "the record's one trigger lands the harm on round 1")
	assert.Equal(t, 1, countContaining(drainPlain(1), "The poison burns through your veins!"),
		"the expiring, one-and-only trigger still sends the flavour line")
}
```

In `TestDotProducerRecordsNegativeHarm_MobTarget` and `_PlayerTarget`, change the doc sentence `and that it converts dotDuration through buffs.TickTriggers rather than passing it straight through as TriggersLeft.` to `and that it passes dotDuration straight through as TriggersLeft (the record ticks every round, slice 1b).`, and replace both assertions

```go
	assert.Equal(t, buffs.TickTriggers(wantDuration), recs[0].TriggersLeft,
		"TriggersLeft must be TickTriggers(dotDuration), not dotDuration itself")
```

with

```go
	assert.Equal(t, wantDuration, recs[0].TriggersLeft,
		"TriggersLeft must be dotDuration itself: the record ticks every round")
```

In `internal/hooks/conditions_pin_test.go`, in the doc comment of `TestPin_PoisonTickKillsAndNamesTheCause` replace `a ONE-trigger poison record (buffs.TickTriggers(3)) has its only trigger land on the third UserRoundTick call (buff 121's triggerrate is three rounds), and that trigger is also the record's LAST:` with `a ONE-trigger poison record has its only trigger land on the first UserRoundTick call (buff 121 ticks every round), and that trigger is also the record's LAST:`, and replace the body lines

```go
	_ = u.Character.AddBuffMagnitude(buffs.BuffIdPoisoned, buffs.TickTriggers(3), -5, "pin")
	u.Character.Health = 1

	// No regen lands in the round tick: 1 - 5 <= 0.
	UserRoundTick(events.NewRound{RoundNumber: 1})
	UserRoundTick(events.NewRound{RoundNumber: 2})
	UserRoundTick(events.NewRound{RoundNumber: 3})
```

with

```go
	_ = u.Character.AddBuffMagnitude(buffs.BuffIdPoisoned, 1, -5, "pin")
	u.Character.Health = 1

	// No regen lands in the round tick: 1 - 5 <= 0.
	UserRoundTick(events.NewRound{RoundNumber: 1})
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/hooks/ -run 'TestRoundTick_PoisonDamage|TestRoundTick_TriggerLineLandsOnExpiringTick|TestDotProducerRecordsNegativeHarm|TestPin_PoisonTickKills' -count=1`
Expected: FAIL (health still 80 after round 1; `TriggersLeft` is `dotDuration/3`).

- [ ] **Step 3: Change the record and the fixture**

`_datafiles/world/dogmud/buffs/121-poisoned.yaml`: `triggerrate: 3 rounds` becomes `triggerrate: 1 round`.

`internal/buffs/test_helpers.go:29`: in the `BuffIdPoisoned` entry change `TriggerRate: "3 rounds", RoundInterval: 3` to `TriggerRate: "1 round", RoundInterval: 1`.

- [ ] **Step 4: Change both producers**

In `internal/hooks/spell_resolution.go`, in BOTH dot producers (`applyMobEffect_dot` near line 636 and `resolveMobSpellAgainstPlayer` near line 1662) replace the comment lines

```go
	// The hook applied int(magnitude) per round with a floor of one; the
	// record's negative snapshot is that harm. The old hook landed that
	// harm only every third round while dotDuration ticked down every
	// round; the record keeps that cadence itself now (buff 121's
	// triggerrate is three rounds), so TickTriggers converts dotDuration
	// into the matching trigger count instead of one trigger per round.
```

(indented one tab deeper in the second site) with

```go
	// The record's negative snapshot is int(magnitude) harm, floored at
	// one. Buff 121 ticks every round (slice 1b, owner ruling 2026-09-14;
	// it used to land every third round), so dotDuration is the trigger
	// count as it stands.
```

and replace `buffs.TickTriggers(dotDuration)` with `dotDuration` on the `AddBuffMagnitude` line at each site.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/hooks/ ./internal/buffs/ -count=1`
Expected: PASS, including `TestShippedConditionRecordsMatchTestHelperShape`.

- [ ] **Step 6: Re-key the guard and run it**

Run: `go test . -run TestPlayerBuffsTravelTheEventPath -count=1`
If it fails naming `internal/hooks/spell_resolution.go|<line>` entries (the comment shrank by two lines), update the two `spell_resolution.go` keys in `buff_apply_path_guard_test.go` to the reported lines and change their section comment to `// ── former combat condition: the spell dot is now one record (Task 8; re-keyed slice 1b when the dot moved to every round) ──`. Rerun until PASS.

- [ ] **Step 7: Null probes**

1. Put `triggerrate: 3 rounds` back in `121-poisoned.yaml` AND `TriggerRate: "3 rounds", RoundInterval: 3` in the helper: `TestRoundTick_PoisonDamage` fails (`80`, want `75`). Restore both.
2. Change only the YAML back to `3 rounds`: `TestShippedConditionRecordsMatchTestHelperShape` fails naming buff 121 `RoundInterval`. Restore.
3. At the mob-target producer, pass `dotDuration/3`: `TestDotProducerRecordsNegativeHarm_MobTarget` fails. Restore.

- [ ] **Step 8: Commit**

```bash
gofmt -w internal/hooks/spell_resolution.go internal/hooks/hooks_test.go internal/hooks/conditions_pin_test.go internal/buffs/test_helpers.go buff_apply_path_guard_test.go
git add _datafiles/world/dogmud/buffs/121-poisoned.yaml internal/buffs/test_helpers.go internal/hooks/spell_resolution.go internal/hooks/hooks_test.go internal/hooks/conditions_pin_test.go buff_apply_path_guard_test.go
git commit -F - <<'EOF'
feat(conditions): the spell dot ticks every round (owner ruling 2026-09-14)

Per-tick amount unchanged, so a spell dot's total damage triples; accepted
("DOTs are pretty underused").

Null probes: <record the three here>

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 4: Bleeds stack, from the knobs

**Files:**
- Modify: `_datafiles/world/dogmud/buffs/122-bleeding.yaml` (confirm filename)
- Modify: `internal/buffs/test_helpers.go:30`
- Create: `internal/actions/bleed.go`, `internal/actions/bleed_test.go`
- Modify: `internal/actions/combat_rake.go:56-62,124-133`, `combat_maul.go:56-62,124-134`, `combat_hamstring.go:53,64,130-136`, `combat_drain.go:56-58,69,136-149,218-220,310-319`, `combat_throttle.go:54-56,66,136-146`
- Modify: `internal/hooks/item_procs.go:194-218`
- Modify: `_datafiles/world/dogmud/items/materials-40000/40186-thornwall_harness.yaml:28-29`
- Delete: `internal/buffs/ticks.go`, `internal/buffs/ticks_test.go`
- Modify tests: `internal/hooks/predator_hooks_test.go:33-189`, `internal/hooks/conditions_pin_test.go:50-74`, `internal/hooks/item_procs_test.go:235-261`, `internal/actions/combat_drain_test.go:175-177,389,462`, `internal/actions/combat_throttle_test.go:166-168`
- Modify: `buff_apply_path_guard_test.go:164-172`

- [ ] **Step 1: Write the failing helper test**

Create `internal/actions/bleed_test.go`:

```go
package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestBleedPerRound(t *testing.T) {
	cases := []struct {
		strength         int
		divisor, floor   configs.ConfigInt
		want             int
	}{
		{100, 50, 1, 2},
		{149, 50, 1, 2}, // integer division, not rounding
		{150, 50, 1, 3},
		{30, 50, 1, 1},  // below the divisor: the floor
		{100, 33, 1, 3},
		{100, 50, 4, 4}, // a floor above the quotient wins
		// The shipped knobs at Strength 100 (config_bleed_stacks_test.go's
		// TestShippedBleedTuningMeetsTheSliceTargets mirrors this formula
		// because configs cannot import actions): rake, drain, hamstring 2;
		// maul 2; throttle 3.
		{100, 35, 1, 2},
	}
	for _, c := range cases {
		if got := bleedPerRound(c.strength, c.divisor, c.floor); got != c.want {
			t.Errorf("bleedPerRound(%d, %d, %d) = %d, want %d", c.strength, c.divisor, c.floor, got, c.want)
		}
	}
}
```

Run: `go test ./internal/actions/ -run TestBleedPerRound -count=1`
Expected: FAIL, `undefined: bleedPerRound`.

- [ ] **Step 2: Create `internal/actions/bleed.go`**

```go
package actions

import "github.com/GoMudEngine/GoMud/internal/configs"

// bleedPerRound is one bleed stack's per-round health loss: the attacker's
// Strength divided by the move's <Move>BleedStrengthDivisor knob, never less
// than its <Move>BleedMin knob. Both knobs are validated to at least 1, so
// the division is safe. Every bleed move passes the result as the stack's
// magnitude (negated) and <Move>BleedRounds as its rounds.
func bleedPerRound(strength int, divisor, floor configs.ConfigInt) int {
	amt := strength / int(divisor)
	if amt < int(floor) {
		amt = int(floor)
	}
	return amt
}
```

Run: `go test ./internal/actions/ -run TestBleedPerRound -count=1`
Expected: PASS.

- [ ] **Step 3: Rewrite the bleed tests to the stacking behaviour (they will fail)**

In `internal/hooks/predator_hooks_test.go`, replace everything from the line `// ─── Bleeding record tick ───` down to (not including) `// ─── PackFlee ───` with:

```go
// ─── Bleeding record tick ──────────────────────────────────────────────────

// The Bleeding record (122) ticks every round and STACKS (slice 1b, owner
// ruling 2026-09-14): each AddBuffMagnitude is its own stack with its own
// rounds, and one round tick lands the sum of the live stacks as ONE harm with
// ONE flavour line.
func TestRoundTick_BleedDamagesPlayer(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer buffs.SeedConditionRecordsForTest()()

	u := users.GetByUserId(1)
	require.NotNil(t, u)

	// The door validates on add, and Validate() recomputes HealthMax from
	// stats/balance config, clobbering a raw HealthMax.Value. Seeding Base
	// keeps HealthMax comfortably above the 40 this test needs.
	u.Character.HealthMax.Base = 100
	_ = u.Character.AddBuffMagnitude(buffs.BuffIdBleeding, 20, -5, "test")
	u.Character.Health = 40
	_ = drainPlain(1)

	UserRoundTick(events.NewRound{RoundNumber: 1})
	assert.Equal(t, 35, u.Character.Health, "the bleed ticks on the first round tick")
	assert.Equal(t, 1, countContaining(drainPlain(1), "Blood seeps from your wounds!"),
		"the tick sends the bleed flavour line")

	// Cross-hook pin: AutoHeal's own bleed block is gone, so firing it after
	// the tick must not apply a second, redundant bleed hit. Health may move
	// by ordinary out-of-combat regen, but not by another 5-point bleed. The
	// record still has 19 rounds left here (Bleeding flag still held), so a
	// revived flag-gated block in AutoHeal is reachable and would be caught.
	AutoHeal(events.NewRound{RoundNumber: 3})
	assert.GreaterOrEqual(t, u.Character.Health, 35, "AutoHeal must not re-apply the bleed")
	assert.Less(t, u.Character.Health, 40, "AutoHeal's own regen should be small next to the 5-point bleed it must not repeat")

	u.Character.RemoveBuff(buffs.BuffIdBleeding)
	u.Character.Health = 50
}

func TestRoundTick_BleedDamagesMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer buffs.SeedConditionRecordsForTest()()

	mob := mobs.GetInstance(100)
	require.NotNil(t, mob)

	mob.Character.HealthMax.Base = 100
	_ = mob.Character.AddBuffMagnitude(buffs.BuffIdBleeding, 1, -50, "test")
	mob.Character.Health = 2

	// tickMobBuffs runs in MobRoundTick's idle lane, before the active-zone
	// check, so it fires for every mob regardless of zone activity.
	MobRoundTick(events.NewRound{RoundNumber: 1})

	assert.Less(t, mob.Character.Health, 0,
		"a 50-magnitude bleed on a 2-health mob should store overkill, not clamp to 0; health=%d", mob.Character.Health)
	assert.Less(t, mob.Character.Health, 1,
		"overkilled health must still satisfy the `< 1` death gate; health=%d", mob.Character.Health)

	mob.Character.RemoveBuff(buffs.BuffIdBleeding)
	mob.Character.Health = 50
}

func TestRoundTick_BleedMinDamageOne(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer buffs.SeedConditionRecordsForTest()()

	mob := mobs.GetInstance(100)
	require.NotNil(t, mob)

	mob.Character.HealthMax.Base = 100
	// Magnitude -0.5 truncates to 0, so the stack's amount is floored to -1.
	_ = mob.Character.AddBuffMagnitude(buffs.BuffIdBleeding, 1, -0.5, "test")
	mob.Character.Health = 50

	MobRoundTick(events.NewRound{RoundNumber: 1})

	assert.Equal(t, 49, mob.Character.Health)

	mob.Character.RemoveBuff(buffs.BuffIdBleeding)
	mob.Character.Health = 50
}

// A one-round stack's only tick is also its last. The player tick used to gate
// its whole body on !buff.Expired(), so that tick applied nothing and said
// nothing. This pins the expiring tick: the harm lands AND the flavour line
// goes out, exactly once.
//
// Null probe: restoring `!buff.Expired() &&` to the text gate in
// NewRound_UserRoundTick.go turns the line assertion red.
func TestRoundTick_BleedLineLandsOnExpiringTick(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer buffs.SeedConditionRecordsForTest()()

	u := users.GetByUserId(1)
	require.NotNil(t, u)

	// Discard anything an earlier test left queued, so the count below is
	// only what this round produced.
	_ = drainPlain(1)

	u.Character.HealthMax.Base = 100
	_ = u.Character.AddBuffMagnitude(buffs.BuffIdBleeding, 1, -5, "test")
	require.Equal(t, 1, u.Character.Buffs.GetBuffs(buffs.BuffIdBleeding)[0].TriggersLeft,
		"a one-round stack: the tick under test is its last")
	// Health is set AFTER the add, because the door validates on add.
	u.Character.Health = 80

	UserRoundTick(events.NewRound{RoundNumber: 1})

	assert.Equal(t, 75, u.Character.Health,
		"the stack's only tick is also its last, and it must still apply its harm")
	assert.Equal(t, 1, countContaining(drainPlain(1), "Blood seeps from your wounds!"),
		"the expiring tick must still send the bleed flavour line, exactly once")

	u.Character.RemoveBuff(buffs.BuffIdBleeding)
	u.Character.Health = 50
}

// Two stacks sum into ONE harm and ONE line per round, and each stack ends on
// its own: 3 rounds of -2 and 5 rounds of -3 land -5, -5, -5, -3, -3, then
// nothing.
func TestRoundTick_BleedStacksSumIntoOneHarmPerRound_Player(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer buffs.SeedConditionRecordsForTest()()

	u := users.GetByUserId(1)
	require.NotNil(t, u)
	u.Character.HealthMax.Base = 100
	_ = u.Character.AddBuffMagnitude(buffs.BuffIdBleeding, 3, -2, "test")
	_ = u.Character.AddBuffMagnitude(buffs.BuffIdBleeding, 5, -3, "test")
	require.Len(t, u.Character.Buffs.GetBuffs(buffs.BuffIdBleeding), 1, "two stacks, one record")
	u.Character.Health = 80
	_ = drainPlain(1)

	for round, want := range []int{75, 70, 65, 62, 59, 59} {
		UserRoundTick(events.NewRound{RoundNumber: uint64(round + 1)})
		assert.Equal(t, want, u.Character.Health, "round %d", round+1)
		lines := countContaining(drainPlain(1), "Blood seeps from your wounds!")
		if round < 5 {
			assert.Equal(t, 1, lines, "round %d: one line however many stacks are live", round+1)
		} else {
			assert.Equal(t, 0, lines, "round %d: the bleed has ended", round+1)
		}
	}

	u.Character.RemoveBuff(buffs.BuffIdBleeding)
	u.Character.Health = 50
}

func TestRoundTick_BleedStacksSumIntoOneHarmPerRound_Mob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer buffs.SeedConditionRecordsForTest()()

	mob := mobs.GetInstance(100)
	require.NotNil(t, mob)
	mob.Character.HealthMax.Base = 100
	_ = mob.Character.AddBuffMagnitude(buffs.BuffIdBleeding, 3, -2, "test")
	_ = mob.Character.AddBuffMagnitude(buffs.BuffIdBleeding, 5, -3, "test")
	mob.Character.Health = 80

	for round, want := range []int{75, 70, 65, 62, 59, 59} {
		MobRoundTick(events.NewRound{RoundNumber: uint64(round + 1)})
		assert.Equal(t, want, mob.Character.Health, "round %d", round+1)
	}

	mob.Character.RemoveBuff(buffs.BuffIdBleeding)
	mob.Character.Health = 50
}
```

In `internal/hooks/conditions_pin_test.go`, in `TestPin_BleedTickKillsAndNamesTheCause` replace

```go
	_ = u.Character.AddBuffMagnitude(buffs.BuffIdBleeding, buffs.TickTriggers(3), -5, "pin")
	u.Character.Health = 1

	// No regen lands in the round tick: 1 - 5 <= 0.
	UserRoundTick(events.NewRound{RoundNumber: 1})
	UserRoundTick(events.NewRound{RoundNumber: 2})
	UserRoundTick(events.NewRound{RoundNumber: 3})
```

with

```go
	_ = u.Character.AddBuffMagnitude(buffs.BuffIdBleeding, 1, -5, "pin")
	u.Character.Health = 1

	// No regen lands in the round tick: 1 - 5 <= 0.
	UserRoundTick(events.NewRound{RoundNumber: 1})
```

In `internal/hooks/item_procs_test.go`, `TestProcApplyCondition_Bleed`: replace

```go
	if got := target.Buffs.TriggersLeft(buffs.BuffIdBleeding); got != 2 {
		t.Fatalf("expected 2 triggers (duration 6 at a 3-round triggerrate), got %d", got)
	}
```

with

```go
	if got := target.Buffs.TriggersLeft(buffs.BuffIdBleeding); got != 6 {
		t.Fatalf("expected 6: duration is the stack's rounds and the record ticks every round, got %d", got)
	}
```

and after the `held` length check add

```go
	if len(held[0].Stacks) != 1 || held[0].Stacks[0].RoundsLeft != 6 || held[0].Stacks[0].Amount != -12 {
		t.Fatalf("expected one stack of 6 rounds at -12, got %+v", held[0].Stacks)
	}
```

In `internal/actions/combat_drain_test.go`, replace the assertion at `:176-177`

```go
	assert.GreaterOrEqual(t, res.BleedDmg, 2,
		"BleedDmg should be at least 2 (min floor)")
```

with

```go
	assert.GreaterOrEqual(t, res.BleedDmg, int(configs.GetBalanceConfig().DrainBleedMin),
		"BleedDmg should be at least DrainBleedMin")
	if assert.Len(t, held, 1) && assert.Len(t, held[0].Stacks, 1, "one landed drain is one stack") {
		assert.Equal(t, int(configs.GetBalanceConfig().DrainBleedRounds), held[0].Stacks[0].RoundsLeft,
			"the stack lasts DrainBleedRounds")
	}
```

(`held` is the variable declared at `:169`; add `"github.com/GoMudEngine/GoMud/internal/configs"` to the imports if absent.) At `:389` and `:462` change `2, "bleed magnitude should be at least the floor of 2"` to `int(configs.GetBalanceConfig().DrainBleedMin), "bleed magnitude should be at least DrainBleedMin"`.

In `internal/actions/combat_throttle_test.go:167-168`, make the same change with `ThrottleBleedMin` and add the same stack-length assertion with `ThrottleBleedRounds` against the `held` declared at `:160`.

- [ ] **Step 4: Run to verify they fail**

Run: `go test ./internal/hooks/ -run 'Bleed|TestProcApplyCondition_Bleed' -count=1` and `go test ./internal/actions/ -run 'Drain|Throttle' -count=1`
Expected: FAIL (three-round cadence, no stacks).

- [ ] **Step 5: Change the record and the fixture**

`_datafiles/world/dogmud/buffs/122-bleeding.yaml` becomes:

```yaml
buffid: 122
name: Bleeding
description: Wounds seeping blood, taking damage over time.
triggerrate: 1 round
triggercount: 4
tick_pool: health
tick_from_magnitude: true
trigger_user_text: '<ansi fg="red">Blood seeps from your wounds!</ansi>'
end_user_text: "Your wounds stop bleeding."
flags:
  - bleeding
  - silent-start
  - stacking
```

(`triggercount` is only the stack length used when a producer passes 0; every producer passes rounds. 4 matches `procApplyCondition`'s fallback.)

`internal/buffs/test_helpers.go:30`, the `BuffIdBleeding` entry becomes:

```go
		{BuffId: BuffIdBleeding, Name: "Bleeding", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 4, Flags: []Flag{Bleeding, SilentStart, Stacking}, TickPool: "health", TickFromMagnitude: true, TriggerUserText: `<ansi fg="red">Blood seeps from your wounds!</ansi>`, EndUserText: "Your wounds stop bleeding."},
```

- [ ] **Step 6: Change the five moves**

`combat_rake.go`, replace the hit block (`:124-133`) with:

```go
	// On hit: add a bleed stack (RakeBleedRounds rounds, Strength /
	// RakeBleedStrengthDivisor per round, floor RakeBleedMin).
	bleedDmg := 0
	if result.Hit {
		bleedDmg = bleedPerRound(char.Stats.Strength.ValueAdj, cfg.RakeBleedStrengthDivisor, cfg.RakeBleedMin)
		_ = target.Char.AddBuffMagnitude(buffs.BuffIdBleeding, int(cfg.RakeBleedRounds), -float64(bleedDmg), "rake")
	}
```

`combat_maul.go`, replace `:124-134` with:

```go
	// On hit: add a bleed stack (MaulBleedRounds rounds, Strength /
	// MaulBleedStrengthDivisor per round, floor MaulBleedMin).
	bleedDmg := 0
	if result.Hit {
		bleedDmg = bleedPerRound(char.Stats.Strength.ValueAdj, cfg.MaulBleedStrengthDivisor, cfg.MaulBleedMin)
		_ = target.Char.AddBuffMagnitude(buffs.BuffIdBleeding, int(cfg.MaulBleedRounds), -float64(bleedDmg), "maul")
	}
```

`combat_hamstring.go`, replace `:130-136` with:

```go
	// On hit: add a bleed stack (HamstringBleedRounds rounds, Strength /
	// HamstringBleedStrengthDivisor per round, floor HamstringBleedMin).
	bleedDmg := 0
	if result.Hit {
		bleedDmg = bleedPerRound(char.Stats.Strength.ValueAdj, cfg.HamstringBleedStrengthDivisor, cfg.HamstringBleedMin)
		_ = target.Char.AddBuffMagnitude(buffs.BuffIdBleeding, int(cfg.HamstringBleedRounds), -float64(bleedDmg), "hamstring")
	}
```

`combat_drain.go`, single target (`:136-149`):

```go
	// On hit: bleed the victim. Bleed is a status effect (binary), so it stays
	// gated on a clean hit. One stack: DrainBleedRounds rounds, Strength /
	// DrainBleedStrengthDivisor per round, floor DrainBleedMin.
	bleedDmg := 0
	if result.Hit {
		bleedDmg = bleedPerRound(char.Stats.Strength.ValueAdj, cfg.DrainBleedStrengthDivisor, cfg.DrainBleedMin)
		_ = target.Char.AddBuffMagnitude(buffs.BuffIdBleeding, int(cfg.DrainBleedRounds), -float64(bleedDmg), "drain")
	}
```

area drain (`:310-319`):

```go
		// Bleed is a status effect (binary), so it stays gated on a clean hit.
		if moveResult.Hit {
			pr.BleedDmg = bleedPerRound(char.Stats.Strength.ValueAdj, cfg.DrainBleedStrengthDivisor, cfg.DrainBleedMin)
			_ = target.Character.AddBuffMagnitude(buffs.BuffIdBleeding, int(cfg.DrainBleedRounds), -float64(pr.BleedDmg), "drain")
		}
```

`combat_throttle.go`, replace the bleed lines inside `if result.Hit {` (`:137-145`) with:

```go
		// Health-over-time: add a bleed stack (ThrottleBleedRounds rounds,
		// Strength / ThrottleBleedStrengthDivisor per round, floor
		// ThrottleBleedMin); the choke's primary DoT is stamina drain.
		bleedDmg = bleedPerRound(char.Stats.Strength.ValueAdj, cfg.ThrottleBleedStrengthDivisor, cfg.ThrottleBleedMin)
		_ = target.Char.AddBuffMagnitude(buffs.BuffIdBleeding, int(cfg.ThrottleBleedRounds), -float64(bleedDmg), "throttle")
```

Before editing each file, confirm `cfg := configs.GetBalanceConfig()` is in scope at the edit (it is declared at the cooldown check in rake, maul, hamstring, drain and throttle; for the area drain loop, check `ExecuteDrainArea` and declare `cfg := configs.GetBalanceConfig()` once above the loop if it is not already). Also update each file's function doc bullet (`On hit: apply the Bleeding record (duration N, magnitude = Strength/D, min M)`) to `On hit: add a Bleeding stack (<Move>BleedRounds, <Move>BleedStrengthDivisor, <Move>BleedMin)` and each `BleedDmg` field comment to `BleedDmg is the per-round amount of the bleed stack added on a hit`.

- [ ] **Step 7: Item proc and Thornwall Harness**

`internal/hooks/item_procs.go`, replace the `procApplyCondition` doc comment's `duration (rounds, default 4 if unset/<1), magnitude (per-tick, default 2 if unset/<1)` with `duration (the stack's rounds, default 4 if unset/<1), magnitude (per-round health loss, default 2 if unset/<1). Each proc that fires adds one stack; see the Stacking flag.`, and change `buffs.TickTriggers(dur)` to `dur`.

`40186-thornwall_harness.yaml`: `duration: 6` becomes `duration: 10`, `magnitude: 14` becomes `magnitude: 7`.

- [ ] **Step 8: Delete `TickTriggers`**

```bash
git rm internal/buffs/ticks.go internal/buffs/ticks_test.go
```

Run: `go build ./... && go vet ./internal/buffs/ ./internal/hooks/ ./internal/actions/`
Expected: builds. If anything still names `TickTriggers`, grep `TickTriggers(` across `*.go` and fix that call to pass rounds.

- [ ] **Step 9: Run the tests to verify they pass**

Run: `go test ./internal/buffs/ ./internal/hooks/ ./internal/actions/ -count=1`
Expected: PASS.

- [ ] **Step 10: Re-key the guard**

Run: `go test . -run TestPlayerBuffsTravelTheEventPath -count=1`
For every unlisted `file|line` it reports for the bleed producers and `item_procs.go`, update the key in the Bleeding section of `buff_apply_path_guard_test.go` and change that section comment to `// ── former combat condition: Bleeding is now one stacking record (Task 9; re-keyed slice 1b) ──`. Remove any key it reports as stale. Rerun until PASS.

- [ ] **Step 11: Null probes**

1. Remove `- stacking` from `122-bleeding.yaml` and `Stacking` from the helper: `TestRoundTick_BleedStacksSumIntoOneHarmPerRound_Player` fails at round 1 (`77`, want `75`: the second add overwrote). Restore both.
2. Remove `- stacking` from the YAML only: `TestShippedConditionRecordsMatchTestHelperShape` fails naming buff 122 `Flags`. Restore.
3. In `combat_drain.go` single target, pass `4` instead of `int(cfg.DrainBleedRounds)`: the drain test's stack-length assertion fails. Restore.
4. In `item_procs.go`, pass `dur/3`: `TestProcApplyCondition_Bleed` fails (`2`, want `6`). Restore.

- [ ] **Step 12: Commit**

```bash
gofmt -w internal/actions/bleed.go internal/actions/bleed_test.go internal/actions/combat_rake.go internal/actions/combat_maul.go internal/actions/combat_hamstring.go internal/actions/combat_drain.go internal/actions/combat_throttle.go internal/actions/combat_drain_test.go internal/actions/combat_throttle_test.go internal/hooks/item_procs.go internal/hooks/predator_hooks_test.go internal/hooks/conditions_pin_test.go internal/hooks/item_procs_test.go internal/buffs/test_helpers.go buff_apply_path_guard_test.go
git add _datafiles/world/dogmud/buffs/122-bleeding.yaml _datafiles/world/dogmud/items/materials-40000/40186-thornwall_harness.yaml internal/buffs/test_helpers.go internal/actions/bleed.go internal/actions/bleed_test.go internal/actions/combat_rake.go internal/actions/combat_maul.go internal/actions/combat_hamstring.go internal/actions/combat_drain.go internal/actions/combat_throttle.go internal/actions/combat_drain_test.go internal/actions/combat_throttle_test.go internal/hooks/item_procs.go internal/hooks/predator_hooks_test.go internal/hooks/conditions_pin_test.go internal/hooks/item_procs_test.go buff_apply_path_guard_test.go
git commit -F - <<'EOF'
feat(conditions): bleeds stack, each hit its own timer, numbers from config (owner ruling 2026-09-14)

Bleeding ticks every round and carries the stacking flag. Rake, maul,
hamstring, drain and throttle read their stack length and per-round amount
from the Task 1 knobs; the Thornwall Harness proc passes rounds and is retuned
to 10 rounds at 7. TickTriggers had no caller left and is deleted.

Null probes: <record the four here>

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

`git rm` in Step 8 already staged the deletions.

---

### Task 5: The Recovering cap bites for players

**Files:**
- Modify: `internal/hooks/NewRound_UserRoundTick.go:244-258` (move to after `:395`)
- Create: `internal/hooks/recovering_order_test.go`
- Modify: `internal/combat/hitroll_test.go:151-158` (comment)

- [ ] **Step 1: Write the failing test**

Create `internal/hooks/recovering_order_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The Recovering record (118, attacks_cap 1) must still be live when DoCombat
// runs, which is after UserRoundTick returns (hook order in hooks.go). It used
// to be added by the stand attempt and expired by the same hook's buff tick,
// so a player never felt it while mobs always did (owner ruling 2026-09-14:
// make it bite). This drives the real round tick, not a direct add: the
// direct-add test in internal/combat passed the whole time the player path was
// inert.
func TestUserRoundTick_RecoveringIsLiveWhenCombatRuns(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer buffs.SeedConditionRecordsForTest()()

	u := users.GetByUserId(1)
	require.NotNil(t, u)
	require.NoError(t, u.Character.Validate())
	require.NoError(t, u.Character.Position.TransitionToProne(position.ProneData{MinRecoveryRounds: 2},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward}))

	UserRoundTick(events.NewRound{RoundNumber: 1})
	require.True(t, u.Character.IsProne(), "precondition: round 1 is inside the minimum recovery period")
	assert.Equal(t, 1.0, u.Character.Buffs.Effect(buffs.EffectAttacksCap),
		"round 1: the swing cap must be live after the round tick, where DoCombat reads it")

	UserRoundTick(events.NewRound{RoundNumber: 2})
	require.True(t, u.Character.IsProne(), "precondition: round 2 consumes the last minimum round")
	assert.Equal(t, 1.0, u.Character.Buffs.Effect(buffs.EffectAttacksCap),
		"round 2: last round's record expired, and this round's attempt re-added it")

	UserRoundTick(events.NewRound{RoundNumber: 3})
	require.False(t, u.Character.IsProne(), "precondition: nobody holds the player down, so round 3 is a free stand")
	assert.Equal(t, 0.0, u.Character.Buffs.Effect(buffs.EffectAttacksCap),
		"round 3: stood up, so no cap carries into this round's combat")

	u.Character.RemoveBuff(buffs.BuffIdRecovering)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/hooks/ -run TestUserRoundTick_RecoveringIsLiveWhenCombatRuns -count=1`
Expected: FAIL at round 1, `expected: 1, actual: 0`. If a precondition fails instead, stop and report: the fixture does not reach the path under test.

- [ ] **Step 3: Move the block**

In `NewRound_UserRoundTick.go`, cut the whole block from `// Stage 7.5: Attempt automatic recovery from prone` through its closing `}` (`:244-258`) and paste it directly after the closing `}` of the `if triggeredBuffs := user.Character.Buffs.Trigger(); ...` block (the line after `events.AddToQueue(events.BuffsTriggered{...})` and its `}`), before `// Pinnacle item upkeep`. Replace its first comment line with:

```go
				// Stage 7.5: Attempt automatic recovery from prone (contested
				// if someone is holding the character down, free otherwise).
				// AFTER the buff tick, the order MobRoundTick uses: a failed or
				// gated attempt adds the one-round Recovering record (118,
				// attacks_cap 1), and running before the tick expired it before
				// DoCombat could read it, so a player never felt the cap
				// (slice 1b, owner ruling 2026-09-14).
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/hooks/ -count=1`
Expected: PASS.

- [ ] **Step 5: Fix the stale combat test comment**

In `internal/combat/hitroll_test.go`, replace the comment above `TestRecoveringRecordExpiredByItsOwnTickCapsNothing` with:

```go
// A Recovering record applied and then ticked contributes nothing to the swing
// count: the cap lives exactly one tick. Both round ticks therefore add it
// AFTER their buff tick (MobRoundTick always did; UserRoundTick since slice
// 1b), so it is live when DoCombat runs. TestUserRoundTick_RecoveringIsLive
// WhenCombatRuns in internal/hooks pins the player order.
```

- [ ] **Step 6: Null probe**

Move the block back above the `Trigger()` block: `TestUserRoundTick_RecoveringIsLiveWhenCombatRuns` fails at round 1 (`actual: 0`). Restore; confirm green.

- [ ] **Step 7: Commit**

```bash
gofmt -w internal/hooks/NewRound_UserRoundTick.go internal/hooks/recovering_order_test.go internal/combat/hitroll_test.go
git add internal/hooks/NewRound_UserRoundTick.go internal/hooks/recovering_order_test.go internal/combat/hitroll_test.go
git commit -F - <<'EOF'
fix(conditions): the Recovering swing cap bites for players (stand attempt after the buff tick, as mobs do)

Null probe: <record it here>

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 6: One visibility rule for both condition lists

**Files:**
- Modify: `internal/buffs/buffspec.go` (add `Listed`, delete `VisibleNameDesc` `:215-220`)
- Modify: `internal/buffs/stacks.go` (add `DisplayName`)
- Modify: `internal/buffs/buffspec_test.go` (delete `TestBuffSpec_VisibleNameDesc` `:87-143`; add `TestBuffSpec_Listed`)
- Modify: `internal/buffs/stacks_test.go` (add `TestDisplayName`)
- Modify: `internal/usercommands/conditions.go`
- Create: `internal/usercommands/conditions_entries_test.go`
- Modify: `modules/gmcp/gmcp.Char.go:646-720`
- Create: `modules/gmcp/gmcp.Conditions_test.go`
- Modify: `_datafiles/html/public/webclient-pure.html:2143-2148` (comment only)

- [ ] **Step 1: Write the failing buffs tests**

In `internal/buffs/buffspec_test.go`, delete `TestBuffSpec_VisibleNameDesc` entirely and add:

```go
func TestBuffSpec_Listed(t *testing.T) {
	cases := []struct {
		name string
		spec BuffSpec
		want bool
	}{
		{"plain", BuffSpec{Name: "Stoneskin"}, true},
		{"hidden", BuffSpec{Name: "Hidden", Flags: []Flag{Hidden}}, false},
		{"secret", BuffSpec{Name: "Respawn Grace", Secret: true}, false},
		{"hidden and secret", BuffSpec{Name: "Both", Secret: true, Flags: []Flag{Hidden}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.spec.Listed())
		})
	}
}
```

In `internal/buffs/stacks_test.go` add:

```go
func TestDisplayName(t *testing.T) {
	spec := stackingSpec()
	one := &Buff{BuffId: 930, Stacks: []Stack{{RoundsLeft: 2, Amount: -1}}}
	three := &Buff{BuffId: 930, Stacks: []Stack{{2, -1}, {3, -1}, {4, -1}}}
	plain := &Buff{BuffId: 930}
	if got := DisplayName(plain, spec); got != "Gash" {
		t.Fatalf("no stacks: %q, want %q", got, "Gash")
	}
	if got := DisplayName(one, spec); got != "Gash" {
		t.Fatalf("one stack: %q, want %q (a count of one says nothing)", got, "Gash")
	}
	if got := DisplayName(three, spec); got != "Gash (3)" {
		t.Fatalf("three stacks: %q, want %q", got, "Gash (3)")
	}
}
```

Run: `go test ./internal/buffs/ -run 'TestBuffSpec_Listed|TestDisplayName' -count=1`
Expected: FAIL, `undefined: Listed` / `undefined: DisplayName`.

- [ ] **Step 2: Implement `Listed` and `DisplayName`, delete `VisibleNameDesc`**

In `internal/buffs/buffspec.go`, replace the whole `VisibleNameDesc` function with:

```go
// Listed reports whether a held record appears in the player's condition
// lists: the in-game `conditions` command and the Char.Conditions GMCP
// payload. Both call this, so the two can never disagree. A hidden record is
// left out because it would tell you that you are hidden; a secret record is
// engine bookkeeping or a state the player is not meant to know about (owner
// ruling 2026-09-14: shown in neither list, not as "Mysterious Affliction").
func (b *BuffSpec) Listed() bool {
	return !b.Secret && !slices.Contains(b.Flags, Hidden)
}
```

(`slices` is already imported by `buffspec.go`; confirm.) In `internal/buffs/stacks.go` add `"strconv"` to the imports and:

```go
// DisplayName is the name a condition list shows for a held record: the
// spec's name, with the live stack count appended when more than one stack is
// live ("Bleeding (3)").
func DisplayName(b *Buff, spec *BuffSpec) string {
	if len(b.Stacks) > 1 {
		return spec.Name + " (" + strconv.Itoa(len(b.Stacks)) + ")"
	}
	return spec.Name
}
```

Run: `go test ./internal/buffs/ -count=1`
Expected: PASS. `go build ./...` now FAILS in `usercommands` and `gmcp` (`VisibleNameDesc` undefined); the next steps fix both.

- [ ] **Step 3: Write the failing `conditions` command test**

Create `internal/usercommands/conditions_entries_test.go`:

```go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The SAME fixture and expectation as modules/gmcp's
// TestBuildConditionsPayload_ListsWhatTheConditionsCommandLists: a plain, a
// hidden, a secret and a stacked record; only the plain and the stacked one
// are listed, and the stacked name carries its count.
func TestConditionEntries_LeavesOutHiddenAndSecretRecords(t *testing.T) {
	t.Cleanup(buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		990: {BuffId: 990, Name: "Stoneskin", Description: "Skin like rock.", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 10},
		991: {BuffId: 991, Name: "Hidden", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 10, Flags: []buffs.Flag{buffs.Hidden}},
		992: {BuffId: 992, Name: "Respawn Grace", Secret: true, TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 10},
	}))
	t.Cleanup(buffs.SeedConditionRecordsForTest())

	c := &characters.Character{}
	c.Buffs.Validate(true)
	for _, id := range []int{990, 991, 992} {
		require.True(t, c.Buffs.AddBuff(id, false), "fixture: add %d", id)
	}
	require.True(t, c.Buffs.AddBuffMagnitude(buffs.BuffIdBleeding, 3, -2))
	require.True(t, c.Buffs.AddBuffMagnitude(buffs.BuffIdBleeding, 5, -3))

	names := []string{}
	for _, e := range conditionEntries(c) {
		names = append(names, e.Name)
	}
	assert.ElementsMatch(t, []string{"Stoneskin", "Bleeding (2)"}, names)
}
```

Run: `go test ./internal/usercommands/ -run TestConditionEntries -count=1`
Expected: FAIL to compile (`undefined: conditionEntries`, plus the `VisibleNameDesc` build error).

- [ ] **Step 4: Rewrite `internal/usercommands/conditions.go`**

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// conditionEntry is one row of the `conditions` template.
type conditionEntry struct {
	Name        string
	Description string
	RoundsLeft  int
	PermaBuff   bool
}

// Conditions lists everything currently affecting the player: one entry per
// held, unexpired, listed buff record, with its display name, description and
// a duration.
//
// There is one loop because there is one source. This command used to print
// the buff list and then a second list of combat conditions from an enum with
// its own tick, which is why warcry and rally needed a mirror flag to keep
// them out of the first list. The enum is gone; records are the conditions.
func Conditions(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	tplTxt, _ := templates.Process("character/conditions", conditionEntries(user.Character), user.UserId)
	user.SendText(messaging.CategorySystem, tplTxt)
	return true, nil
}

// conditionEntries builds the rows. BuffSpec.Listed decides what appears, the
// same predicate the Char.Conditions GMCP payload uses, so the text list and
// the web client show the same records. buffs.DisplayName appends a stacking
// record's live count.
func conditionEntries(c *characters.Character) []conditionEntry {
	entries := []conditionEntry{}
	for _, buff := range c.GetBuffs() {
		spec := buffs.GetBuffSpec(buff.BuffId)
		if spec == nil || !spec.Listed() {
			continue
		}
		roundsLeft, _ := buffs.GetDurations(buff, spec)
		entries = append(entries, conditionEntry{
			Name:        buffs.DisplayName(buff, spec),
			Description: spec.Description,
			RoundsLeft:  roundsLeft,
			PermaBuff:   buff.PermaBuff,
		})
	}
	return entries
}
```

Check the template first: `grep -rn "RoundsLeft\|PermaBuff\|\.Name\|\.Description" _datafiles/world/dogmud/templates/character/conditions.template` (adjust the path with `Glob` if it differs); it must read only these four fields.

- [ ] **Step 5: Write the failing GMCP test**

Create `modules/gmcp/gmcp.Conditions_test.go`:

```go
package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The SAME fixture and expectation as internal/usercommands'
// TestConditionEntries_LeavesOutHiddenAndSecretRecords.
func TestBuildConditionsPayload_ListsWhatTheConditionsCommandLists(t *testing.T) {
	t.Cleanup(buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		990: {BuffId: 990, Name: "Stoneskin", Description: "Skin like rock.", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 10},
		991: {BuffId: 991, Name: "Hidden", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 10, Flags: []buffs.Flag{buffs.Hidden}},
		992: {BuffId: 992, Name: "Respawn Grace", Secret: true, TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 10},
	}))
	t.Cleanup(buffs.SeedConditionRecordsForTest())

	c := &characters.Character{}
	c.Buffs.Validate(true)
	for _, id := range []int{990, 991, 992} {
		require.True(t, c.Buffs.AddBuff(id, false), "fixture: add %d", id)
	}
	require.True(t, c.Buffs.AddBuffMagnitude(buffs.BuffIdBleeding, 3, -2))
	require.True(t, c.Buffs.AddBuffMagnitude(buffs.BuffIdBleeding, 5, -3))

	payload := buildConditionsPayload(c)

	names := []string{}
	for _, cond := range payload {
		names = append(names, cond.Name)
	}
	assert.ElementsMatch(t, []string{"Stoneskin", "Bleeding (2)"}, names)
	assert.Contains(t, payload, "Bleeding", "the map key stays the plain name; only the label carries the count")
}
```

Run: `go test ./modules/gmcp/ -run TestBuildConditionsPayload -count=1`
Expected: FAIL to compile (`undefined: buildConditionsPayload`).

- [ ] **Step 6: Extract `buildConditionsPayload`**

In `modules/gmcp/gmcp.Char.go`, replace the body of `if all || g.wantsGMCPPayload(`Char.Conditions`, gmcpModule) { ... }` (from `// One payload, one source.` through the closing `}` of the `for` loop, keeping the `if !all { return ... }` tail) with:

```go
		payload.Conditions = buildConditionsPayload(user.Character)
```

and add below `conditionDurationLabel`:

```go
// buildConditionsPayload builds Char.Conditions for one character. One
// payload, one source: buff records ARE the conditions, so this map is
// everything the client used to get as Char.Affects plus the qualitative
// duration word the old Char.Conditions list carried. BuffSpec.Listed decides
// what appears, the same predicate the in-game `conditions` command uses, so
// the web client and the text list show the same records. The map is keyed by
// the plain spec name (a repeat takes a `#n` suffix); the entry's name is
// buffs.DisplayName, which appends a stacking record's live count.
func buildConditionsPayload(ch *characters.Character) map[string]GMCPCondition {
	c := configs.GetTimingConfig()
	conditions := make(map[string]GMCPCondition)

	nameIncrement := 0
	for _, buff := range ch.GetBuffs() {

		buffSpec := buffs.GetBuffSpec(buff.BuffId)
		if buffSpec == nil || !buffSpec.Listed() {
			continue
		}

		timeLeft, timeMax := -1, -1
		roundsLeft := 0

		if !buff.PermaBuff {
			var totalRounds int
			roundsLeft, totalRounds = buffs.GetDurations(buff, buffSpec)
			if roundsLeft < 0 {
				roundsLeft = 0
			}
			timeMax = c.RoundsToSeconds(totalRounds)
			timeLeft = c.RoundsToSeconds(roundsLeft)
		}

		buffSource := buff.Source
		if buffSource == `` {
			buffSource = `unknown`
		}
		cond := GMCPCondition{
			Name:         buffs.DisplayName(buff, buffSpec),
			Description:  buffSpec.Description,
			DurationMax:  timeMax,
			DurationLeft: timeLeft,
			// A permabuff reports roundsLeft 0, which the label reads as
			// "sustained", the same word the old condition list used for a
			// permanent entry.
			Duration: conditionDurationLabel(roundsLeft),
			Type:     buffSource,
		}

		cond.Mods = make(map[string]int)
		for name, value := range buffSpec.StatMods {
			cond.Mods[name] = value
		}

		key := buffSpec.Name
		if _, ok := conditions[key]; ok {
			nameIncrement++
			key += `#` + strconv.Itoa(nameIncrement)
		}

		conditions[key] = cond
	}

	return conditions
}
```

If `slices` is no longer used anywhere in `gmcp.Char.go` after this, remove it from the imports (`go build` says so).

- [ ] **Step 7: Build and run everything this task touched**

Run: `go build ./... && go test ./internal/buffs/ ./internal/usercommands/ ./modules/gmcp/ -count=1`
Expected: PASS. Then `grep -rn "VisibleNameDesc" --include=*.go .` returns nothing (run it standalone; `grep` exits 1 on no match).

- [ ] **Step 8: Web client comment**

In `_datafiles/html/public/webclient-pure.html`, in the comment at `:2143-2148` change `(the server omits stealth buffs like Hidden/Empathic Shroud, and` to `(the server omits hidden records like Hidden/Empathic Shroud and secret ones like Respawn Grace, and`. Comment only; no behaviour change.

- [ ] **Step 9: Null probes**

1. In `Listed`, drop `!b.Secret &&`: both list tests fail with `Respawn Grace` present, and `TestBuffSpec_Listed` fails on `secret`. Restore.
2. In `conditionEntries`, replace `buffs.DisplayName(buff, spec)` with `spec.Name`: the usercommands test fails (`Bleeding`, want `Bleeding (2)`). Restore.
3. In `buildConditionsPayload`, drop `|| !buffSpec.Listed()`: the GMCP test fails with `Hidden` and `Respawn Grace` present. Restore.

- [ ] **Step 10: Commit**

```bash
gofmt -w internal/buffs/buffspec.go internal/buffs/stacks.go internal/buffs/buffspec_test.go internal/buffs/stacks_test.go internal/usercommands/conditions.go internal/usercommands/conditions_entries_test.go modules/gmcp/gmcp.Char.go modules/gmcp/gmcp.Conditions_test.go
git add internal/buffs/buffspec.go internal/buffs/stacks.go internal/buffs/buffspec_test.go internal/buffs/stacks_test.go internal/usercommands/conditions.go internal/usercommands/conditions_entries_test.go modules/gmcp/gmcp.Char.go modules/gmcp/gmcp.Conditions_test.go _datafiles/html/public/webclient-pure.html
git commit -F - <<'EOF'
feat(conditions): secret records leave both condition lists; stacked bleeds show their count

One BuffSpec.Listed predicate feeds the conditions command and Char.Conditions
(owner ruling 2026-09-14: the text list matches the web client). With no
secret record reaching either list, VisibleNameDesc and "Mysterious
Affliction" had no reader and are deleted.

Null probes: <record the three here>

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

### Task 7: Documentation, comments, patch note

**Files:** `internal/buffs/context.md`, `internal/characters/context.md`, `internal/combat/context.md`, `internal/hooks/context.md`, `modules/gmcp/context.md`, `internal/actions/context.md`, `internal/characters/buffs.go:148-155`, `internal/users/userrecord.go:454-461`, `internal/events/eventtypes.go:30-33`, `internal/hooks/tick_cause.go:14-24`, `internal/hooks/Death_PlayerAnnouncement.go:160-172`, `internal/hooks/NewRound_UserRoundTick.go:274-304`, `docs/PATCH_NOTES.md`, `docs/README.md`

Every symbol named in a doc must exist: check with `codegraph_search` or grep before writing it.

- [ ] **Step 1: `internal/buffs/context.md`**

1. Under "### Flags added" (`:72-79`) add:

```markdown
- `stacking` (slice 1b): every application is its own `Stack` with its own
  rounds, held inside the one record. `Trigger` lands the SUM of the live
  stacks as the record's `TickAmount` once a round, drops spent stacks, and
  sets `TriggersLeft` to the longest remaining one; the record ends with its
  last stack. Requires `tick_from_magnitude` and a one-round `triggerrate`
  (`Validate` refuses anything else). Only 122 Bleeding carries it. See
  "Stacking records (`stacks.go`)" below.
```

2. Replace the whole "### Cadence" section (`:81-92`) with:

```markdown
### Cadence

121 Poisoned and 122 Bleeding tick every round (`triggerrate: 1 round`). Slice
1 shipped them at `3 rounds` to reproduce the AutoHeal hook they replaced,
which was gated on `RoundNumber%3`; the owner ruled on 2026-09-14 that both
tick every round. A producer passes a duration in rounds, which for a
one-round record IS the trigger count. The spell dot's per-tick amount did not
change, so its total damage tripled, which the owner accepted.
```

3. Replace the "**The prone recovery cap never bites a player.**" bullet (`:99-103`) with:

```markdown
- **The prone recovery cap bites for players and mobs alike.** 118 Recovering
  lives exactly one tick, so both round ticks add it AFTER their buff tick
  (`MobRoundTick` always did; `UserRoundTick` since slice 1b), and it is live
  when `DoCombat` reads `attacks_cap`.
```

4. Delete the "### Converting a rounds duration into a trigger count (`ticks.go`)" section (`:414-430`) and put in its place:

```markdown
### Stacking records (`stacks.go`)

A spec with the `stacking` flag keeps `Buff.Stacks []Stack`, each
`Stack{RoundsLeft, Amount}`. `Buffs.AddBuffMagnitude` routes such a spec to
`addStack`: `triggers` becomes the new stack's rounds (0 means the spec's
`triggercount`), the magnitude becomes its signed amount through
`tickAmountFor` (a non-zero magnitude never snapshots to zero), and
`syncStacks` derives the record-level fields every other reader uses:
`TriggersLeft` is the longest stack, `TickAmount` and `Magnitude` the sum. So
`Expired`, `GetDurations`, the prune pass, poison immunity, `HasBuff`, the
death cause and both condition lists work unchanged.

`Buffs.Trigger` calls `tickStacks` for such a record: `TickAmount` becomes this
round's landed sum, every stack loses a round, spent stacks drop, and
`TriggersLeft` becomes the longest remaining stack. Both round-tick paths read
`TickAmount` after `Trigger` returns, so one tick is one harm, one wake, one
`cancel-on-damage` pass, one death-cause stamp and one flavour line however
many stacks are live. A stacking record with no stacks is expired without
being returned, so a zero amount never reaches the `tick_percent` fallback.

`RemoveBuff` clears the stacks, and `addStack` clears them first if the record
is expired but not yet pruned, so a cancel followed by a new hit starts fresh.

Every bleed producer runs inside combat, after that round's tick, so a new
stack first ticks on the next round. `buffs.DisplayName` names a held record
for the condition lists and appends the live count above one stack
("Bleeding (3)"). Bleed stack numbers are the fifteen `<Move>Bleed*` balance
knobs; see `internal/actions/bleed.go` and the BLEED STACKS block in
`config.yaml`.
```

5. In "### Magnitude Records (`AddBuffMagnitude`)" (`:595-609`), replace `**a trigger count, not a duration in rounds**: for a one-round-interval record the two coincide, but the three-round-interval dot and bleed records need `TickTriggers` (above) to convert a rounds-literal duration first.` with `**a trigger count, not a duration in rounds**; every record that goes through this door ticks once a round, so the two coincide. A `stacking` spec appends a stack instead of refreshing; see "Stacking records" above.`

6. Replace the "## Display and Visibility" code block's `VisibleNameDesc` function (`:764-771`) with the real `Listed` function and a one-line `DisplayName` signature, both copied from source.

7. In "## Files" delete the `ticks.go` row and add `| `stacks.go` | `Stack`, the stacking tick (`addStack`, `syncStacks`, `tickStacks`), `tickAmountFor`, `DisplayName` |`.

8. Grep the file for `Mysterious`, `TickTriggers`, `three-round` and `3 rounds` (each standalone) and fix any remaining mention.

- [ ] **Step 2: The other context.md files**

- `internal/characters/context.md:936-945`: replace the paragraph starting `**The recovery penalty is a record, and it never bites a player.**` with:

```markdown
**The recovery penalty is a record, and it bites.** Buff 118 Recovering carries
`attacks_cap: 1` as a LITERAL, so the `magnitude` argument above is unused and
passed as 0; `calcSwingCount` reads it through
`Buffs.Effect(buffs.EffectAttacksCap)`. The record lives exactly one tick, so
both round ticks call `AttemptRecovery` AFTER their buff tick and the record
is live when `DoCombat` runs. `UserRoundTick` called it before the tick until
slice 1b (owner ruling 2026-09-14), which is why players never felt the cap
while mobs always did. See `internal/buffs/context.md`.
```

- `internal/combat/context.md:622-627`: replace from `Failed/gated attempts add the 118 Recovering record` to the end of that item with `Failed/gated attempts add the 118 Recovering record (`attacks_cap: 1`, read by `calcSwingCount` through `Buffs.Effect(buffs.EffectAttacksCap)`). Both round ticks attempt recovery after their buff tick, so the one-tick record is live when `DoCombat` reads it, for players and mobs alike (players since slice 1b).`

- `internal/hooks/context.md:1619-1625`: replace the DoT bullet's last two sentences with `Both pass that rounds figure straight to `AddBuffMagnitude(buffs.BuffIdPoisoned, dotDuration, ...)`: record 121 ticks every round (slice 1b; it was every third round before).` At `:464-467`, after `ran every round.` add ` Both records tick every round since slice 1b.`

- `modules/gmcp/context.md:292`: the `name` row becomes `| `name` | `Name` | `buffs.DisplayName`: the spec name, plus the live stack count above one ("Bleeding (3)") |`. Under "Things worth knowing" add a bullet: `- **Hidden and secret records are left out**, by `BuffSpec.Listed`, the same predicate the `conditions` command uses (owner ruling 2026-09-14). The map is built by `buildConditionsPayload(ch)`. The key is the plain spec name.`

- `_datafiles/config.yaml` bleed block comment (Task 1 review nits; disk AND blob, skip-worktree procedure from Task 1 Step 5): rewrap the `<Move>BleedRounds rounds and takes the attacker's Strength / <Move>BleedStrengthDivisor` line to about 78 columns like its neighbours, and change "(misses and spells share that slot)" to "(a miss still spends it, and spells share it)". Add `_datafiles/config.yaml` is already staged by that procedure; confirm `S` after committing.
- `internal/buffs/context.md` stacking section: add that `Buff.Source` is the LAST applier's source (each `Character.AddBuffMagnitude` overwrites it), so for a stacking record it is not per-stack; nothing reads a bleed's `Source` today (Task 4 review).
- `_datafiles/world/dogmud/templates/help/throttle.template` ~23-24 says the wounds "bleed briefly"; a throttle stack now lasts 8 rounds and stacks. Load `dogmud-player-copy`, then grep every help template naming rake, maul, hamstring, drain, throttle or bleeding and correct any claim about bleed length or a single wound (no raw numbers). Add each touched template to the commit.
- `internal/configs/context.md`: add the fifteen `<Move>Bleed*` knobs to the per-subsystem knob table for combat special moves (match how neighbouring rows such as `SpecialMoveCooldown` or `TauntHoldRounds` are listed).
- `.claude/skills/dogmud-balance-config/SKILL.md`: the knob counts (lines 3, 101, 117 at plan time) are stale; recount with the skill's own grep commands (`grep -cE '^\s*[A-Za-z_]+\s+Config[A-Za-z]+\b' internal/configs/config.balance.go`) and update each figure with today's date.
- `internal/actions/context.md`: find the bleed moves' description (grep `bleed` in the file). State that each landed hit adds a Bleeding stack sized by `bleedPerRound` from the `<Move>BleedRounds` / `<Move>BleedStrengthDivisor` / `<Move>BleedMin` knobs, and add `bleed.go` to the file table.

- [ ] **Step 3: Stale comments in Go**

- `internal/buffs/buffs.go` `AddBuffMagnitude` doc comment (~:331-333): drop the sentence that the three-round dot and bleed records need `buffs.TickTriggers`; say instead that every record through this door ticks once a round, so the trigger count is the rounds.
- Then run standalone `grep -rn "TickTriggers" --include=*.go .` (bare identifier, not `TickTriggers(`) and expect only the two history clauses in `NewRound_UserRoundTick.go` if you kept them; rewrite those too so the identifier is gone entirely.
- `internal/characters/buffs.go:153-155`: `own triggercount; the exact trigger count, not rounds — use buffs.TickTriggers for the three-round dot and bleed records.` becomes `own triggercount. Every record that uses this door today ticks once a round, so the trigger count is the rounds; a stacking record takes it as the new stack's rounds.`
- `internal/users/userrecord.go:459-461`: `triggers is the exact trigger count, not a duration in rounds — use buffs.TickTriggers for the three-round dot and bleed records.` becomes `triggers is the exact trigger count, which for a one-round record is the rounds.`
- `internal/events/eventtypes.go:31-33`: `(not a duration in rounds — buffs.TickTriggers converts a rounds-literal duration for the three-round dot and bleed records)` becomes `(for a one-round record, the rounds)`.
- `internal/hooks/tick_cause.go:17-19`: `so an ordinary one-trigger bleed (every bleed produced by buffs.TickTriggers for a short duration) or the rare last-trigger poison tick` becomes `so the last tick of a bleed or a poison`.
- `internal/hooks/Death_PlayerAnnouncement.go:166-168`: `every ordinary bleed, since buffs.TickTriggers commonly produces one trigger, and one poison tick in ten` becomes `the last tick of every bleed and every poison`.
- `internal/hooks/NewRound_UserRoundTick.go:286-291` and `:297-299`: replace the clause `which buffs.TickTriggers produces for every ordinary Bleeding duration` with `which slice 1's three-round Bleeding produced for every ordinary duration`, and `(every combat bleed producer passes TickTriggers 3, 4 or 5, all of which return 1)` with `(slice 1's bleed producers all made one-trigger records)`. These are history notes; keep them accurate, not current.

Run: `go build ./... && gofmt -l internal/ modules/` (expect no output from gofmt), then standalone `grep -rn "TickTriggers" --include=*.go --include=*.md internal modules` expecting only history references you chose to keep in the two NewRound comments (none should remain; if the grep finds one, fix it).

- [ ] **Step 4: Patch note**

Load the `dogmud-player-copy` skill. Add at the top of `docs/PATCH_NOTES.md` (below the `# DOGMud Patch Notes` title) a `## 2026-09-14: ...` entry of one paragraph, no raw numbers, no "tick", covering: wounds from beasts and draining creatures now keep bleeding for longer and pile up with every new wound, so a long fight against a pack wears you down, and the bleeding goes on after the fight until it closes (the list shows how many wounds are open); spell poison now burns every moment it lasts rather than now and then, so it does far more over its span; scrambling up from the ground costs you your full flurry that moment, for players as it always did for creatures; and a few housekeeping effects no longer show in the conditions list or the client at all.

- [ ] **Step 5: `docs/README.md`**

The plan's own row was added when the plan was committed. Add a row for each new source file this slice created that the README indexes by convention (check how `internal/buffs` and `internal/actions` files are listed there, if at all; if the README indexes only docs, add nothing here). The playtest scenario rows are added in Task 8.

- [ ] **Step 6: Commit**

```bash
git add internal/buffs/context.md internal/characters/context.md internal/combat/context.md internal/hooks/context.md modules/gmcp/context.md internal/actions/context.md internal/configs/context.md .claude/skills/dogmud-balance-config/SKILL.md docs/superpowers/plans/2026-09-14-conditions-unification-slice-1b-mechanics.md internal/characters/buffs.go internal/users/userrecord.go internal/events/eventtypes.go internal/hooks/tick_cause.go internal/hooks/Death_PlayerAnnouncement.go internal/hooks/NewRound_UserRoundTick.go docs/PATCH_NOTES.md docs/README.md
git commit -F - <<'EOF'
docs(conditions): slice 1b in context.md, comments and patch notes (stacking bleeds, dot every round, Recovering bites, secret records unlisted)

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

Run `python tools/context_md_audit.py` before committing and fix anything it reports for the six context.md files touched.

---

### Task 8: Gate, playtest, ship

Load `dogmud-shipping` and `dogmud-playtesting` before starting. The owner deploys; do not deploy.

- [ ] **Step 1: Whole-branch verification**

Run the pre-push gate in the order `dogmud-shipping` gives (build, vet, full `go test ./...`, lint). Every failure is investigated, not retried. Record the exact commands and results.

- [ ] **Step 2: Whole-branch review**

Dispatch a reviewer (superpowers:requesting-code-review) against `04134f7e5..HEAD` with the spec. Fix every finding in place and re-review.

- [ ] **Step 3: Boot check**

Follow `dogmud-shipping`'s detached-worktree boot check on HEAD. Confirm the boot log loads 122 Bleeding with no flag or validation panic and shows the server bound its own port.

- [ ] **Step 4: Playtest scenario**

Create `tools/playtest/scenarios/conditions-slice-1b.yaml` and goals under `tools/playtest/goals/scenarios/conditions-slice-1b/`, modelled on `conditions-slice-1.yaml`:

- **Lane bleed (veteran profile, `start_room: 3015`, Ironwind Steppe wolf spawns):** fight steppe wolves for many rounds without killing them fast; type `conditions` every few rounds and quote the Bleeding row. Verify: the row reads "Bleeding (2)" or "Bleeding (3)" at some point; "Blood seeps from your wounds!" arrives once per round, not once per wound; after the fight, the bleeding continues until "Your wounds stop bleeding."; no raw numbers anywhere. If the wolves never hamstring (the AI picks its moves), report which moves they used.
- **Lane caster (specialist-caster profile, `start_room: 5000`):** `ask sable arena 350`, cast `blood-boil` at the arena mob, and quote the affliction line; then report whether the caster, knocked down by the arena mob at any point, read "You attempt to stand, but slip back down..." and how many attack lines followed in that round.
- **Lane list (either tester):** die once (the arena is fine), then type `conditions` and check the web client status panel if available. Verify: no "Respawn Grace" and no "Mysterious Affliction" in either.

Add both scenario files to `docs/README.md` in the same rows style the slice 1 scenario used. Run the scenario locally per `dogmud-playtesting` (ephemeral goals, `--checkout` HEAD, a port the owner's server does not use, kill only your own PIDs). Extract findings to the arena memory file `project-conditions-unification-arc.md` (reports are gitignored).

- [ ] **Step 5: Push, PR, merge**

Per `dogmud-shipping`: every `gh` command carries `--repo pruuk/DOGMud`. PR body lists the four player-visible changes, the fifteen knobs with the starting numbers table, the null probes, the playtest run id and findings, and ends with `🤖 Generated with [Claude Code](https://claude.com/claude-code)`. Merge with `--merge` once CI is green. Do not deploy.

- [ ] **Step 6: Memory**

Update `project-conditions-unification-arc.md` and the `MEMORY.md` status line: slice 1b merged (PR number, merge SHA), not deployed, next is slice 2.
