# Messaging M4, counters slice: the counter answers the defence that won

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Counter narration is chosen by the defence that earned the counter (five pools: dodge, parry, block, quell, defy), parry and block get their own text, the counter primitive refuses any attack that is not single-target, every defy crit answers with a counter-taunt, and one adversarial playtest proves each defence line and its counter line agree.

**Architecture:** `combat.ExecuteCounter` gains the winning `combatvocab.Defence` beside the attack shape and gates on both: not single-target refuses, defy refuses (words answer words, and the counter-taunt lives in `internal/actions`, which `combat` cannot import). The two exits already hold the winner (`move.Defence.Defence`, `out.Defence`) and pass it through. `items.CounterPoolFor(defence)` is the one pool conversion. The spell exit in `hooks` branches on defy to a new exported `actions.FireCounterTaunt`, which taunt's own exit is rewritten to share. Data lands last, in one commit with the golden regenerated once and a check script that proves the dodge rows are the old melee rows.

**Tech Stack:** Go 1.23, `gopkg.in/yaml.v3`, snapshot goldens in `internal/narration/testdata/stores/`, Python 3 check script reading `git show` as bytes, the Docker playtest harness (`/playtest local`).

**Spec:** `docs/superpowers/specs/2026-09-18-messaging-counters-design.md` (owner-approved 2026-09-18). Its facts table was verified against master `f53b07d76`; the line numbers below come from it and were re-read while writing this plan.

**Three rules every task obeys:**
1. Every gate runs `go test . ./...` from the repo root. The root package holds `condition_apply_path_guard_test.go`, a line-number allowlist over `spell_resolution.go` and `combat_drain.go`, and Tasks 3 and 4 move lines in both.
2. No commit that does not compile and pass. Constants that data has not caught up with yet stay declared until the data task.
3. No em or en dashes anywhere: Go comments, YAML, patch notes, commit messages.

---

## Facts this plan relies on (re-read from master `f53b07d76`, 2026-09-18)

| Fact | Value | Source |
|---|---|---|
| Primitive | `ExecuteCounter(defender, attacker *characters.Character, shape combatvocab.Attack, sameRoom bool) CounterResult` | `internal/combat/counter.go:90` |
| Pool choice | `counterPoolFor(shape combatvocab.Attack) items.DefencePool`, charm carve-out | `counter.go:145-175` |
| Render | `fillCounterMessages` at `:195`, `items.RenderDefenseMessage(counterPoolFor(result.Shape), ...)` at `:197` | read |
| Skill-move exit | `counterSkillMoveExit(actor Actor, defender *characters.Character, move combat.SkillMoveResult, shape combatvocab.Attack, sameRoom bool) combat.CounterResult` | `internal/actions/combat_counter.go:56` |
| Counter-taunt | `executeCounterTaunt(counterer, target *characters.Character) CounterTauntResult` (`:151`); `counterTauntExit(actor Actor, char *characters.Character, target AggroTarget, out combat.ChannelDefenceResult) CounterTauntResult` (`:229`), dispatches with `messaging.CategoryTauntSuccess` | read |
| Spell exit | `fireSpellCounterTier(room *rooms.Room, out combat.ChannelDefenceResult, shape combatvocab.Attack, defender, caster *characters.Character, defenderUser, casterUser *users.UserRecord) combat.CounterResult`; four call sites use it as a statement | `internal/hooks/counter_tier.go:35`; `spell_resolution.go:461,979,1547,1772` |
| Winner recorded | `out.Defence = winner` where `winner := combatvocab.Defence(res.Winner)`; `DefensiveCrit` set at `:614` after `Defended = true` | `internal/combat/defence_multiplier.go:504,551,614` |
| Stub runners name entries[0] the winner | `Winner: entries[0].Name` | `combat/counter_test.go:63,88`; `hooks/spell_collapse_test.go:49` |
| Drain area | `DrainAreaPlayerResult.Counter` (`combat_drain.go:209-213`, comment plus field plus the blank below), filled at `:310`, dispatched by the loop at `spell_resolution.go:1449-1453` (blank plus comment plus loop) | read |
| Drain shape guard | `TestExecuteDrainAreaIsAPhysicalAreaSpell` requires `spellPhysicalArea >= 2` | `internal/actions/drain_area_shape_test.go` |
| Root guard keys that move | `internal/actions/combat_drain.go|317`; `internal/hooks/spell_resolution.go|1469`, `|1508`, `|1651` | `condition_apply_path_guard_test.go:159-179` |
| Store key | `items.DefencePool`; `DefencePoolFor`; constants `CounterPoolMelee/Ranged/Quell/Defy` | `internal/items/defensive_messages.go:15-39` |
| Fixture | `MinimalDefenseMessageFixture` lists the four counter constants | `items/test_helpers_combat.go:55-61` |
| Tests naming the constants | `combat/counter_social_pool_test.go`; `items/defence_pool_test.go:22-30`; `items/defensive_messages_newly_defendable_test.go:152-154`; `hooks/counter_tier_test.go:305-307,353-375` | grep |
| Golden | `defense_messages.golden`, rows `pool|band|role => text` and `melee|pool|band|role => text`; regenerate with `go test ./internal/narration/... -run TestSnapshotStores -update` | `narration/snapshot_test.go:203,843` |
| Charm test fixture | `charmTestSpellData()` in `hooks/charm_effect_test.go:15` | read |
| Hooks helpers | `pinCounterTierKnobs`, `sequencedContestRunner`, `alwaysDefensiveCritContest`, `attackWinContest`, `seedAllRegistries`, `mobInstanceForCollapseTest`, `roomForCollapseTest`, `physicalHarmSpellForCollapseTest`, `spellAttackSideFor` | `hooks/counter_tier_test.go`, `hooks/spell_collapse_test.go` |
| Combat helpers | `pinCounterConfig`, `counterTestPair`, `counterWinRunner`, `counterDefensiveCritRunner` | `combat/counter_test.go:24-101` |
| Actions helpers | `pinTauntCollapseKnobs`, `pinActionsCounterKnob`, `counterSequencedRunner`, `tauntDeterministicRunner(t, normMargin, atkZ, defZ)`, `newRhetoricActor` | `actions/counter_tier_test.go`, `actions/taunt_collapse_test.go:72` |
| Shipped spells for the playtest | physical single `kinetic-hurl`; mental single `mind-spike`; physical area `sparks`; social `charm` | `_datafiles/world/dogmud/spells/` |
| Ranged kit | sling item 10038, shot item 30064 | `tools/playtest/profiles/ranged-range.yaml` |
| Arena | Rift Chamber is room 5000 (`rooms/thornwall_city/5000.yaml`); `ask sable arena <gold>`, gold is the difficulty dial | MEMORY.md |

## File structure

| File | Responsibility | Task |
|---|---|---|
| `internal/items/defensive_messages.go` | `CounterPoolFor`; the five counter constants | 1, 5 |
| `internal/items/counter_pool_for_test.go` (new) | the conversion, none maps to empty | 1 |
| `internal/combat/counter.go` | primitive takes the defence; the three refusals; pool via `items.CounterPoolFor` | 2, 3, 4 |
| `internal/combat/counter_pool_rekey_test.go` (new, replaces `counter_social_pool_test.go`) | every reachable cell renders from the winner's pool | 2 |
| `internal/combat/counter_defence_invariant_test.go` (new) | a defensive crit always names its defence | 2 |
| `internal/combat/counter_gate_test.go` (new) | not-single refuses; defy refuses | 3, 4 |
| `internal/actions/combat_counter.go` | exit passes the winner; `FireCounterTaunt` exported; `counterTauntExit` shares it | 2, 4 |
| `internal/actions/combat_drain.go` | drain area loses its counter | 3 |
| `internal/actions/drain_area_shape_test.go` | assertion count | 3 |
| `internal/hooks/counter_tier.go` | spell exit branches on defy | 4 |
| `internal/hooks/counter_tier_test.go` | narration pin re-seeded on the dodge pool; area gate; charm retort | 2, 3, 4 |
| `internal/hooks/spell_resolution.go` | drain dispatcher loses its counter loop | 3 |
| `condition_apply_path_guard_test.go` | four keys re-numbered | 3 |
| `_datafiles/world/dogmud/defense-messages/counter-{dodge,parry,block,quell,defy}.yaml` | the five pools | 5 |
| `internal/narration/testdata/stores/defense_messages.golden` | regenerated once | 5 |
| `tools/counter_pool_rekey_check.py` (new) | proves the regeneration | 5 |
| `internal/items/test_helpers_combat.go`, `defence_pool_test.go`, `defensive_messages_newly_defendable_test.go` | five constants | 5 |
| `docs/PATCH_NOTES.md` | three entries | 3, 4, 5 |
| `internal/combat/context.md`, `internal/items/context.md`, `docs/superpowers/audits/messaging-m6-content-ledger.md`, `docs/README.md` | docs | 6 |
| `tools/playtest/profiles/counters.yaml`, `tools/playtest/goals/2026-09-18-counters-slice.yaml` | the playtest fixture | 8 |

---

## Task 0: Branch and baseline

**Files:** none

- [ ] **Step 1: Branch from master**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout master && git pull --ff-only origin master
git checkout -b feature/messaging-counters
git log --oneline -1
```
Expected: `f53b07d76` or a later master; record the SHA. The spec's design branch `feature/messaging-counters-design` holds only the spec commit and is merged by this branch's PR: `git merge --no-ff feature/messaging-counters-design` now, so the spec rides along.

- [ ] **Step 2: Baseline**

```bash
go build ./... && go test . ./... 2>&1 | grep -v "^ok\|no test files" ; echo "exit $?"
```
Expected: no output but the exit line (every package `ok`). Note: `internal/rooms` zone lifecycle tests fail locally on Windows under `DOGMUD_BOOT_SMOKE=1` only; do not set it.

- [ ] **Step 3: Prove the retiring names are findable**

```bash
grep -rln "CounterPoolMelee\|CounterPoolRanged\|counter-melee\|counter-ranged\|counterPoolFor" --include=*.go --include=*.md --include=*.yaml --include=*.golden . | grep -v "docs/superpowers" | sort
```
Expected, 12 files: `_datafiles/world/dogmud/defense-messages/counter-melee.yaml`, `counter-ranged.yaml`, `docs/roadmaps/UNIFIED_RESOLUTION_ROADMAP.md`, `internal/combat/context.md`, `internal/combat/counter.go`, `internal/combat/counter_social_pool_test.go`, `internal/items/context.md`, `internal/items/defence_pool_test.go`, `internal/items/defensive_messages.go`, `internal/items/defensive_messages_newly_defendable_test.go`, `internal/items/test_helpers_combat.go`, `internal/narration/testdata/stores/defense_messages.golden`. Task 7 runs the same grep and expects only the roadmap.

---

## Task 1: `items.CounterPoolFor`

**Files:**
- Modify: `internal/items/defensive_messages.go:21-25`
- Create: `internal/items/counter_pool_for_test.go`

- [ ] **Step 1: Write the failing test**

```go
package items

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// The counter earned by a defensive crit is narrated by the DEFENCE that won
// it, so its pool is the defence's name under a counter- prefix. None maps
// to the empty pool, which RenderDefenseMessage answers with an empty triad
// and combat then replaces with its generic fallback: never silent.
func TestCounterPoolForIsCounterDashTheDefence(t *testing.T) {
	for _, d := range combatvocab.Defences() {
		if got, want := CounterPoolFor(d), DefencePool("counter-"+string(d)); got != want {
			t.Errorf("CounterPoolFor(%s) = %q, want %q", d, got, want)
		}
	}
	if got := CounterPoolFor(combatvocab.DefenceNone); got != "" {
		t.Errorf("CounterPoolFor(none) = %q, want empty", got)
	}
}
```

- [ ] **Step 2: Run it**

```bash
go test ./internal/items/ -run TestCounterPoolForIsCounterDashTheDefence 2>&1 | tail -3
```
Expected: build failure, `undefined: CounterPoolFor`.

- [ ] **Step 3: Add the conversion under `DefencePoolFor`**

In `internal/items/defensive_messages.go`, directly after the `DefencePoolFor` function (line 25), add:

```go
// CounterPoolFor names the pool that narrates the counter EARNED by a
// defensive crit on d: the defence's own name under a counter- prefix, so a
// parry crit reads as a parry answered, a block crit as a block answered.
// DefenceNone maps to the empty pool, which RenderDefenseMessage answers
// with an empty triad; internal/combat then falls back to its generic
// counter lines, so the tier never goes silent.
func CounterPoolFor(d combatvocab.Defence) DefencePool {
	if d == combatvocab.DefenceNone {
		return ""
	}
	return DefencePool("counter-" + string(d))
}
```

- [ ] **Step 4: Run the package**

```bash
gofmt -l internal/items/ ; go test ./internal/items/ 2>&1 | tail -2
```
Expected: gofmt prints nothing; `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/items/defensive_messages.go internal/items/counter_pool_for_test.go
git commit -F - <<'EOF'
feat(items): CounterPoolFor names a counter pool by the defence that won

The counters slice keys counter narration by the defence that earned the
counter. This is the one conversion; the constants and the files follow
in the data commit.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
EOF
```

---

## Task 2: The primitive takes the winning defence

**Files:**
- Modify: `internal/combat/counter.go:17-52, 90, 145-175, 195-198`
- Modify: `internal/actions/combat_counter.go:56-63`
- Modify: `internal/hooks/counter_tier.go:35-43`
- Modify: `internal/hooks/counter_tier_test.go:301-375`
- Delete: `internal/combat/counter_social_pool_test.go`
- Create: `internal/combat/counter_pool_rekey_test.go`
- Create: `internal/combat/counter_defence_invariant_test.go`

- [ ] **Step 1: Write the failing parity test**

`internal/combat/counter_pool_rekey_test.go`:

```go
package combat

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// markedCounterPools seeds one counter pool per defence whose every line
// carries the pool's name, so a render can be traced back to the pool that
// produced it.
func markedCounterPools() map[items.DefencePool]*items.DefenseMessageGroup {
	out := map[items.DefencePool]*items.DefenseMessageGroup{}
	for _, d := range combatvocab.Defences() {
		pool := items.CounterPoolFor(d)
		mk := func(band string) items.DefenseOptions {
			messages := func(audience string) items.MessageOptions {
				result := make(items.MessageOptions, 5)
				for i := range result {
					result[i] = items.ItemMessage("pool=" + string(pool) + " " + audience + " " + band +
						" {actee} answers {actor}")
				}
				return result
			}
			return items.DefenseOptions{Together: items.DefenseTogetherMessages{
				ToDefender: messages("defender"),
				ToAttacker: messages("attacker"),
				ToRoom:     messages("room"),
			}}
		}
		out[pool] = &items.DefenseMessageGroup{OptionId: pool, Options: items.DefenseIntensity{
			items.Weak: mk("weak"), items.Normal: mk("normal"), items.Heavy: mk("heavy"),
		}}
	}
	return out
}

// Every reachable (attack, defence) cell of the spec's matrix renders from
// the WINNER's pool, never the attack's. A melee move parried must read the
// parry pool even though on master every melee counter read one pool, and a
// physical spell dodged must read the dodge pool even though on master it
// read counter-quell. Keying by shape again fails the parry and block rows.
func TestExecuteCounter_NarratesFromTheDefenceThatWon(t *testing.T) {
	pinCounterConfig(t, 0.5)
	restore := items.SeedDefenseMessagesForTest(markedCounterPools())
	defer restore()

	single := combatvocab.TargetSingle
	cases := []struct {
		name    string
		shape   combatvocab.Attack
		defence combatvocab.Defence
	}{
		{"melee dodged", combatvocab.Melee(single), combatvocab.DefenceDodge},
		{"melee parried", combatvocab.Melee(single), combatvocab.DefenceParry},
		{"melee blocked", combatvocab.Melee(single), combatvocab.DefenceBlock},
		{"shot dodged", combatvocab.Ranged(single), combatvocab.DefenceDodge},
		{"shot blocked", combatvocab.Ranged(single), combatvocab.DefenceBlock},
		{"physical spell dodged", combatvocab.Spell(combatvocab.DamagePhysical, single), combatvocab.DefenceDodge},
		{"physical spell blocked", combatvocab.Spell(combatvocab.DamagePhysical, single), combatvocab.DefenceBlock},
		{"mental spell quelled", combatvocab.Spell(combatvocab.DamageMental, single), combatvocab.DefenceQuell},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			counterer, countered := counterTestPair()
			calls := 0
			restoreRunner := SetChannelAttackContestRunnerForTest(counterWinRunner(&calls))
			defer restoreRunner()

			res := ExecuteCounter(counterer, countered, tc.shape, tc.defence, true)
			if !res.Countered {
				t.Fatalf("%s: the counter did not fire", tc.name)
			}
			if res.Defence != tc.defence {
				t.Errorf("%s: CounterResult.Defence = %q, want %q", tc.name, res.Defence, tc.defence)
			}
			want := "pool=" + string(items.CounterPoolFor(tc.defence))
			for role, msg := range map[string]string{
				"counterer": res.DefenderMsg, "countered": res.AttackerMsg, "room": res.RoomMsg,
			} {
				if !strings.Contains(msg, want) {
					t.Errorf("%s: %s line %q did not come from %s", tc.name, role, msg, want)
				}
			}
		})
	}
}
```

- [ ] **Step 2: Write the failing invariant test**

`internal/combat/counter_defence_invariant_test.go`:

```go
package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// The counter tier reads ChannelDefenceResult.Defence to choose its pool, so
// a defensive crit MUST name the defence that won it. Today it does, because
// DefensiveCrit is set only after the winner is recorded; this pins that
// order so a future edit cannot make the pool lookup silently empty.
func TestResolveChannelAttack_ADefensiveCritNamesItsDefence(t *testing.T) {
	pinCounterConfig(t, 0.5)
	attacker, defender := counterTestPair()
	calls := 0
	restore := SetChannelAttackContestRunnerForTest(counterDefensiveCritRunner(&calls))
	defer restore()

	side := AttackSide{
		Stat: attacker.Stats.Strength.ValueAdj, StatName: "strength",
		Skill: attacker.GetCombatSkillTag(), SkillRank: attacker.GetCombatSkillLevel(),
		Mult: 1.0,
	}
	out := ResolveChannelAttack(combatvocab.Melee(combatvocab.TargetSingle), side, attacker, defender)
	if !out.DefensiveCrit {
		t.Fatalf("fixture error: the contest was supposed to be a defensive crit, got %+v", out)
	}
	if out.Defence == combatvocab.DefenceNone {
		t.Fatalf("a defensive crit must name the defence that won; got none")
	}
}
```

- [ ] **Step 3: Run both to verify failure**

```bash
go test ./internal/combat/ -run 'TestExecuteCounter_NarratesFromTheDefenceThatWon|TestResolveChannelAttack_ADefensiveCritNamesItsDefence' 2>&1 | tail -3
```
Expected: build failure, `too many arguments in call to ExecuteCounter` and `res.Defence undefined`.

- [ ] **Step 4: Re-key the primitive**

In `internal/combat/counter.go`:

(a) In `CounterResult` (line 17), replace the `Shape` field and its comment with:

```go
	// Shape is the ORIGINAL attack that was crit-defended. The counters slice
	// reads its Targeting for the area gate; pool selection no longer uses it.
	Shape combatvocab.Attack

	// Defence is the defence that WON the original contest and earned this
	// counter. It chooses the narration pool (counters slice, spec ruling 1):
	// a parry crit reads the parry pool, a block crit the block pool.
	Defence combatvocab.Defence
```

(b) Change the signature and the first line of `ExecuteCounter` (line 90):

```go
func ExecuteCounter(defender, attacker *characters.Character, shape combatvocab.Attack, defence combatvocab.Defence, sameRoom bool) CounterResult {
	result := CounterResult{Shape: shape, Defence: defence}
```

and add, directly after the `if defender == nil || attacker == nil` guard:

```go
	// A counter is narrated by the defence that won it. No winner, no pool:
	// cannot happen today (DefensiveCrit is set only after the winner is
	// recorded, pinned by TestResolveChannelAttack_ADefensiveCritNamesItsDefence)
	// but the primitive refuses rather than rendering an empty pool.
	if defence == combatvocab.DefenceNone {
		return result
	}
```

Also update the doc comment above `ExecuteCounter`: replace the phrase `on a seam-resolved channel` in its first sentence with `on a seam-resolved channel, narrated from the pool of the defence that won it`.

(c) Delete `counterPoolFor` and its whole comment block (lines 145-175, from `// counterPoolFor maps` through the closing brace).

(d) In `fillCounterMessages` (line 197), replace `counterPoolFor(result.Shape)` with `items.CounterPoolFor(result.Defence)`. Update the function's comment: replace `from the counter-* pools` with `from the winning defence's counter-* pool (items.CounterPoolFor)`.

(e) In the `CounterResult` doc comment (lines 11-16), replace `chosen by the ORIGINAL attack's type` with `chosen by the defence that won`.

- [ ] **Step 5: Pass the winner at both exits**

`internal/actions/combat_counter.go:62`:

```go
	return combat.ExecuteCounter(defender, actor.GetCharacter(), shape, move.Defence.Defence, sameRoom)
```

`internal/hooks/counter_tier.go:42`:

```go
	res := combat.ExecuteCounter(defender, caster, shape, out.Defence, true)
```

- [ ] **Step 6: Re-seed the hooks narration pin on the dodge pool**

In `internal/hooks/counter_tier_test.go`, `TestSpellCounter_NarrationReachesCasterFromCounterQuellPool` (line 301) pins a PHYSICAL spell's counter on the quell pool. The stub runner names `entries[0]` the winner and a physical spell's first eligible defence is dodge, so the counter now reads the dodge pool. Rename the test `TestSpellCounter_NarrationReachesCasterFromTheWinningDefencePool`, change its doc comment to say so, and:

- the seed becomes `items.CounterPoolFor(combatvocab.DefenceDodge): counterPoolNarrationFixture(combatvocab.DefenceDodge),`
- the assertion string becomes `"counterdodge-attacker"`.

Replace `counterQuellNarrationFixture` (line 353) with:

```go
// counterPoolNarrationFixture seeds a marked counter pool for one defence,
// so a render can be traced to the pool that produced it.
func counterPoolNarrationFixture(d combatvocab.Defence) *items.DefenseMessageGroup {
	pool := items.CounterPoolFor(d)
	marker := "counter" + string(d) + "-"
	mk := func(band string) items.DefenseOptions {
		messages := func(audience string) items.MessageOptions {
			result := make(items.MessageOptions, 5)
			for i := range result {
				result[i] = items.ItemMessage(marker + audience + "-" + band +
					" {actee} steps through the gap {actor} left")
			}
			return result
		}
		return items.DefenseOptions{Together: items.DefenseTogetherMessages{
			ToDefender: messages("defender"),
			ToAttacker: messages("attacker"),
			ToRoom:     messages("room"),
		}}
	}
	return &items.DefenseMessageGroup{OptionId: pool, Options: items.DefenseIntensity{
		items.Weak: mk("weak"), items.Normal: mk("normal"), items.Heavy: mk("heavy"),
	}}
}
```

Add `"github.com/GoMudEngine/GoMud/internal/combatvocab"` to that file's imports if it is not there.

- [ ] **Step 7: Delete the shape-keyed test**

```bash
git rm internal/combat/counter_social_pool_test.go
```

- [ ] **Step 8: Gate**

```bash
gofmt -l internal/ ; go build ./... && go test . ./internal/combat/ ./internal/actions/ ./internal/hooks/ ./internal/items/ 2>&1 | tail -6
```
Expected: gofmt prints nothing; all `ok`. If `TestSpellCounter_NarrationReachesCasterFromTheWinningDefencePool` fails with a `counterquell` or empty render, the winner was not dodge: print `out.Defence` in the test and seed that defence instead, then say so in the commit.

- [ ] **Step 9: Sabotage check**

Temporarily change `items.CounterPoolFor(result.Defence)` in `fillCounterMessages` to `items.CounterPoolFor(combatvocab.DefenceDodge)`, run `go test ./internal/combat/ -run TestExecuteCounter_NarratesFromTheDefenceThatWon`, and confirm the parry, block and quell rows go red. Revert.

- [ ] **Step 10: Commit**

```bash
git add internal/combat/counter.go internal/combat/counter_pool_rekey_test.go internal/combat/counter_defence_invariant_test.go internal/actions/combat_counter.go internal/hooks/counter_tier.go internal/hooks/counter_tier_test.go
git commit -F - <<'EOF'
refactor(combat): the counter tier narrates from the defence that won

ExecuteCounter takes the winning combatvocab.Defence beside the attack
shape and chooses its pool with items.CounterPoolFor. The two exits pass
the winner they already held. Until the data commit lands, a dodge,
parry or block crit renders the generic fallback (the counter-dodge,
counter-parry and counter-block files do not exist yet); quell and defy
are unchanged. Sabotage: keying the render on a fixed defence turns the
parry, block and quell rows of the parity test red.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
EOF
```

---

## Task 3: Behaviour commit A: a counter answers a single-target attack

**Files:**
- Modify: `internal/combat/counter.go` (`ExecuteCounter`)
- Create: `internal/combat/counter_gate_test.go`
- Modify: `internal/actions/combat_drain.go:210-214, 308-312`
- Modify: `internal/actions/drain_area_shape_test.go`
- Modify: `internal/hooks/spell_resolution.go:1450-1453`
- Modify: `internal/hooks/counter_tier_test.go` (new test)
- Modify: `condition_apply_path_guard_test.go:159,164-166,174,179`
- Modify: `docs/PATCH_NOTES.md`

- [ ] **Step 1: Write the failing primitive test**

`internal/combat/counter_gate_test.go`:

```go
package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
)

// Owner ruling (counters spec 2): area attacks earn no counter, and multi
// rides along because a counter answers one deliberate attack at one target.
// The gate lives in the primitive so no exit can bypass it by omission.
func TestExecuteCounter_OnlyASingleTargetAttackEarnsACounter(t *testing.T) {
	pinCounterConfig(t, 0.5)
	for _, tc := range []struct {
		name  string
		shape combatvocab.Attack
		want  bool
	}{
		{"single", combatvocab.Melee(combatvocab.TargetSingle), true},
		{"area spell", combatvocab.Spell(combatvocab.DamagePhysical, combatvocab.TargetArea), false},
		{"multi spell", combatvocab.Spell(combatvocab.DamageMental, combatvocab.TargetMulti), false},
		{"thrown area", combatvocab.Thrown(combatvocab.TargetArea), false},
	} {
		counterer, countered := counterTestPair()
		calls := 0
		restore := SetChannelAttackContestRunnerForTest(counterWinRunner(&calls))
		res := ExecuteCounter(counterer, countered, tc.shape, combatvocab.DefenceDodge, true)
		restore()
		if res.Countered != tc.want {
			t.Errorf("%s: Countered = %v, want %v", tc.name, res.Countered, tc.want)
		}
		if !tc.want && calls != 0 {
			t.Errorf("%s: a refused counter must run no contest, ran %d", tc.name, calls)
		}
		if !tc.want && countered.Health != 100000 {
			t.Errorf("%s: a refused counter must deal no damage", tc.name)
		}
	}
}
```

- [ ] **Step 2: Write the failing spell-exit test**

Append to `internal/hooks/counter_tier_test.go`:

```go
// Counters spec ruling 2: an area cast earns no counter. The seven shipped
// physical area spells used to hand every victim a free swing at the caster.
// Pinned at the spell exit so the gate is proven reachable from a cast, not
// only from the primitive.
func TestSpellCounter_AnAreaCastEarnsNoCounter(t *testing.T) {
	pinCounterTierKnobs(t, 0.5)
	cleanup := seedAllRegistries()
	defer cleanup()
	restoreMessages := seedChannelRoutingMessages(t)
	defer restoreMessages()

	caster := users.GetByUserId(1)
	mob := mobInstanceForCollapseTest(t)
	room := roomForCollapseTest(t)
	caster.Character.Health = 100000
	caster.Character.HealthMax.Value = 100000
	caster.Character.Stamina = 500
	caster.Character.StaminaMax.Value = 500

	calls := 0
	restore := combat.SetChannelAttackContestRunnerForTest(sequencedContestRunner(t, &calls,
		alwaysDefensiveCritContest(t), // the cast is crit-defended
		attackWinContest(t),           // would be the counter-swing, must never run
	))
	t.Cleanup(restore)

	spell := physicalHarmSpellForCollapseTest()
	spell.Targeting = combatvocab.TargetArea
	side := spellAttackSideFor(spell, caster.Character)
	fumbled, _ := resolveAgainstMob(caster, mob, room, spell, side, spell.EffectMagnitude)
	require.False(t, fumbled)

	require.Equal(t, 1, calls, "an area cast runs its own contest and never a counter-swing")
	require.Equal(t, 100000, caster.Character.Health, "no counter damage may reach an area caster")
}
```

- [ ] **Step 3: Run both to verify failure**

```bash
go test ./internal/combat/ -run TestExecuteCounter_OnlyASingleTargetAttackEarnsACounter 2>&1 | grep -E "area|multi|thrown|FAIL|ok"
go test ./internal/hooks/ -run TestSpellCounter_AnAreaCastEarnsNoCounter 2>&1 | grep -E "contest|FAIL|ok"
```
Expected: three `Countered = true, want false` lines; the hooks test fails on `calls` (2, not 1).

- [ ] **Step 4: Gate in the primitive**

In `ExecuteCounter`, directly after the `DefenceNone` refusal added in Task 2:

```go
	// A counter answers one deliberate attack at one target (owner ruling,
	// counters spec 2). An area or multi attack earns none, however
	// decisively one victim turned it aside. The gate lives HERE so no exit
	// can bypass it: the spell exits pass the spell's authored targeting and
	// the area spells fall out; throw never had an exit, and now this says
	// why.
	if shape.Targeting != combatvocab.TargetSingle {
		return result
	}
```

Update the `Rules` list in the doc comment above `ExecuteCounter`: after the reach-gated bullet add

```go
//   - single-target only: an area or multi attack earns no counter (owner
//     ruling 2026-09-18). Targeting travels on the shape, so an exit cannot
//     bypass the gate by omission.
```

- [ ] **Step 5: Delete the drain-area counter**

`internal/actions/combat_drain.go`:

(a) Remove the `Counter` field, its three comment lines and the blank line below it from `DrainAreaPlayerResult` (lines 209-213, five lines).

(b) Replace lines 308-312 (the comment, the `counter :=` call, and the `pr :=` line) with:

```go
		// Counters slice: a room-wide drain is an AREA attack and earns no
		// counter, and the primitive would refuse one. The exit is not
		// called rather than called into a gate that always refuses.
		pr := DrainAreaPlayerResult{UserId: uid, MoveResult: moveResult}
```

(c) `internal/actions/drain_area_shape_test.go`: change the final assertion to

```go
	require.Equal(t, 2, spellPhysicalArea, "the skill move's Shape and its SituationalAttackMult must both carry Spell(DamagePhysical, TargetArea); the counter exit is gone (area attacks earn no counter)")
```

(d) `internal/hooks/spell_resolution.go:1449-1453`: delete the blank line, the comment and the loop (five lines), so the function's closing brace follows the `sendVisualRoomText` call directly:

```go

	// U6b Task 11: each earned counter renders AFTER the drain's own outcome.
	for _, pr := range result.PlayerResults {
		actions.DispatchCounterMessages(actions.NewMobActorInRoom(mob, room), pr.Counter)
	}
```

If `actions` is now an unused import in that file, `go build` says so; it is used elsewhere in the file (`actions.ExecuteDrainArea` at `:1377`), so it stays.

- [ ] **Step 6: Re-key the root guard**

Run `go test . 2>&1 | head -30`. The guard names each moved call. `combat_drain.go` lost five lines at the struct and one net line at the loop (five replaced by four), so every allowlisted line below both shifts by six; `spell_resolution.go` lost five lines, so its keys below shift by five. In `condition_apply_path_guard_test.go` change `combat_drain.go|317` to `|311`, `spell_resolution.go|1469` to `|1464`, `|1508` to `|1503`, `|1651` to `|1646`. `combat_drain.go|144` is above the struct and does not move. Confirm every number against the guard's own failure output rather than trusting the arithmetic, then re-run `go test .` and expect `ok`.

- [ ] **Step 7: Patch note**

Insert at the top of `docs/PATCH_NOTES.md`, above the `2026-09-18: Mending your companion` entry:

```markdown
## 2026-09-18: A room-wide working earns no answer

When a working or a sweep reaches everyone in the room, turning it aside
no longer hands each of you a free strike at the one who made it. A
decisive defence still stops it cold. The answering blow is for an attack
aimed at you alone: a swing, a shot from within reach, or a working sent
at you and nobody else.

```

- [ ] **Step 8: Gate**

```bash
gofmt -l internal/ ; go build ./... && go test . ./internal/combat/ ./internal/actions/ ./internal/hooks/ 2>&1 | tail -5
```
Expected: nothing from gofmt; all `ok`.

- [ ] **Step 9: Commit**

```bash
git add internal/combat/counter.go internal/combat/counter_gate_test.go internal/actions/combat_drain.go internal/actions/drain_area_shape_test.go internal/hooks/spell_resolution.go internal/hooks/counter_tier_test.go condition_apply_path_guard_test.go docs/PATCH_NOTES.md
git commit -F - <<'EOF'
balance(combat): only a single-target attack earns a counter

BEHAVIOUR CHANGE (owner ruling, counters spec 2). ExecuteCounter refuses
any shape whose Targeting is not single, so the seven physical area
spells and core-drain stop giving each victim a free swing at the
caster; multi rides along. The drain-area counter field and its
dispatcher loop are deleted rather than left calling a gate that always
refuses. Patch note added.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
EOF
```

---

## Task 4: Behaviour commit B: every defy crit counter-taunts

**Files:**
- Modify: `internal/actions/combat_counter.go:229-270` (export the dispatch)
- Modify: `internal/combat/counter.go` (`ExecuteCounter` refuses defy)
- Modify: `internal/combat/counter_gate_test.go` (new test)
- Modify: `internal/hooks/counter_tier.go:35-58`
- Modify: `internal/hooks/counter_tier_test.go` (new test)
- Modify: `docs/PATCH_NOTES.md`

- [ ] **Step 1: Write the failing primitive test**

Append to `internal/combat/counter_gate_test.go`:

```go
// Words answer words (owner ruling 2026-09-18): a defy crit counter-taunts,
// for charm as well as for taunt, and the counter-taunt lives in
// internal/actions. The primitive therefore refuses a defy defence so nobody
// can route a retort into a sword-swing.
func TestExecuteCounter_RefusesADefyDefence(t *testing.T) {
	pinCounterConfig(t, 0.5)
	for _, shape := range []combatvocab.Attack{
		combatvocab.Rhetoric(combatvocab.TargetSingle),
		combatvocab.Spell(combatvocab.DamageSocial, combatvocab.TargetSingle),
	} {
		counterer, countered := counterTestPair()
		calls := 0
		restore := SetChannelAttackContestRunnerForTest(counterWinRunner(&calls))
		res := ExecuteCounter(counterer, countered, shape, combatvocab.DefenceDefy, true)
		restore()
		if res.Countered || calls != 0 || countered.Health != 100000 {
			t.Errorf("%v: a defy defence must never swing (countered=%v calls=%d health=%d)",
				shape, res.Countered, calls, countered.Health)
		}
	}
}
```

- [ ] **Step 2: Write the failing spell-exit test**

Append to `internal/hooks/counter_tier_test.go`:

```go
// A defied charm is answered with words (owner ruling 2026-09-18). The spell
// exit routes a defy crit to the counter-taunt: conviction falls, health does
// not, and the caster reads a RETORT line from the counter-defy pool, never a
// COUNTER line.
func TestSpellCounter_ADefiedCharmIsAnsweredWithACounterTaunt(t *testing.T) {
	pinCounterTierKnobs(t, 0.5)
	cleanup := seedAllRegistries()
	defer cleanup()
	restoreMessages := items.SeedDefenseMessagesForTest(map[items.DefencePool]*items.DefenseMessageGroup{
		items.CounterPoolFor(combatvocab.DefenceDefy): counterPoolNarrationFixture(combatvocab.DefenceDefy),
	})
	defer restoreMessages()

	caster := users.GetByUserId(1)
	mob := mobInstanceForCollapseTest(t)
	room := roomForCollapseTest(t)
	caster.Character.Health = 100000
	caster.Character.HealthMax.Value = 100000
	caster.Character.Conviction = 10000
	caster.Character.ConvictionMax.Value = 10000
	caster.Character.Stamina = 500
	caster.Character.StaminaMax.Value = 500
	mob.Character.Stats.Charisma.Base = 150
	mob.Character.Stats.Charisma.Recalculate()

	calls := 0
	restore := combat.SetChannelAttackContestRunnerForTest(sequencedContestRunner(t, &calls,
		alwaysDefensiveCritContest(t), // the charm is defy-critted
		attackWinContest(t),           // the counter-taunt lands
	))
	t.Cleanup(restore)

	spell := charmTestSpellData()
	side := spellAttackSideFor(spell, caster.Character)
	events.DrainQueuedMessagesForTest(caster.UserId)
	resolveAgainstMob(caster, mob, room, spell, side, spell.EffectMagnitude)

	require.Equal(t, 2, calls, "the charm and the counter-taunt: two contests, never a third")
	require.Equal(t, 100000, caster.Character.Health, "a retort never wounds the body")
	require.Less(t, caster.Character.Conviction, 10000, "the retort must wound the caster's nerve")

	lines := events.DrainQueuedMessagesForTest(caster.UserId)
	retorts, counters := 0, 0
	for _, line := range lines {
		if strings.Contains(line, "RETORT!") {
			retorts++
			require.Contains(t, strings.ToLower(line), "counterdefy-attacker",
				"the caster's line must be the counter-defy pool's attacker-audience render")
		}
		if strings.Contains(line, "COUNTER!") {
			counters++
		}
	}
	require.Equal(t, 1, retorts, "exactly one retort line for the countered caster; got %v", lines)
	require.Zero(t, counters, "a defied charm must never print a swing")
}
```

- [ ] **Step 3: Run both to verify failure**

```bash
go test ./internal/combat/ -run TestExecuteCounter_RefusesADefyDefence 2>&1 | grep -E "never swing|FAIL|ok"
go test ./internal/hooks/ -run TestSpellCounter_ADefiedCharmIsAnsweredWithACounterTaunt 2>&1 | grep -E "body|nerve|retort|swing|FAIL|ok"
```
Expected: two `must never swing` lines; the hooks test fails on `a retort never wounds the body` (the swing damaged health).

- [ ] **Step 4: Refuse defy in the primitive**

In `ExecuteCounter`, directly after the targeting gate from Task 3:

```go
	// Words answer words: a defy crit counter-taunts, for charm as well as
	// for taunt (owner ruling 2026-09-18). That answer lives in
	// internal/actions.FireCounterTaunt, which this package cannot call, so
	// the primitive refuses the defence rather than swinging steel at a
	// jeer. Callers branch on the defence BEFORE reaching here.
	if defence == combatvocab.DefenceDefy {
		return result
	}
```

And replace the `defy crits COUNTER-TAUNT instead` bullet of the doc comment with:

```go
//   - defy crits COUNTER-TAUNT instead, replacing the swing, whatever the
//     attack was (taunt or charm). NOTE THE PLACEMENT: taunt resolution
//     lives in internal/actions, which IMPORTS internal/combat; this package
//     can never call it. Every exit that can see a defy win branches to
//     internal/actions.FireCounterTaunt first, and this function refuses a
//     defy defence outright.
```

- [ ] **Step 5: Export the counter-taunt dispatch**

In `internal/actions/combat_counter.go`, replace `counterTauntExit` (line 229 through its closing brace) with the two functions below. `maxOfOne` stays.

```go
// FireCounterTaunt is the defy answer, shared by taunt's exit here and the
// spell exit in internal/hooks (a defied charm). counterer is the one whose
// defy critted; countered the one whose words (taunt or charm) were defied.
// A nil user record is a mob and reads no private line. The narration is
// the counter-defy pool via combat.BuildCounterTauntMessages; the room line
// goes to everyone who can see, the two private lines to whichever party
// is a player. Dispatch stays on SendText/SendTextVisual as Task 10's
// review accepted it; moving the retort onto the darkness seam is M4d's.
func FireCounterTaunt(room *rooms.Room, counterer, countered *characters.Character,
	countererUser, counteredUser *users.UserRecord) CounterTauntResult {

	res := executeCounterTaunt(counterer, countered)
	if !res.Fired {
		return res
	}

	countererMsg, counteredMsg, roomMsg := combat.BuildCounterTauntMessages(
		counterer.Name, countered.Name,
		res.Defence.AttackerCrit, res.Damage, maxOfOne(countered.ConvictionMax.Value))

	exclude := []int{}
	if countererUser != nil {
		countererUser.SendText(messaging.CategoryTauntSuccess, countererMsg)
		exclude = append(exclude, countererUser.UserId)
	}
	if counteredUser != nil {
		counteredUser.SendText(messaging.CategoryTauntSuccess, counteredMsg)
		exclude = append(exclude, counteredUser.UserId)
	}
	if room != nil {
		room.SendTextVisual(messaging.CategoryTauntSuccess, roomMsg, exclude...)
	}
	return res
}

// counterTauntExit wires the defy carve-out at ExecuteTaunt's defensive-crit
// exit. actor is the ORIGINAL taunter (now being counter-taunted); target
// identifies the counterer. Resolves the two user records and hands off to
// FireCounterTaunt, the one dispatch every defy answer shares.
func counterTauntExit(actor Actor, char *characters.Character, target AggroTarget,
	out combat.ChannelDefenceResult) CounterTauntResult {

	if !out.DefensiveCrit || target.Char == nil {
		return CounterTauntResult{}
	}
	var countererUser, counteredUser *users.UserRecord
	if target.UserId > 0 {
		countererUser = users.GetByUserId(target.UserId)
	}
	if id := actor.GetUserId(); id > 0 {
		counteredUser = users.GetByUserId(id)
	}
	return FireCounterTaunt(rooms.LoadRoom(char.RoomId), target.Char, char, countererUser, counteredUser)
}
```

Note `actor.SendText` used to deliver the taunter's line; `users.GetByUserId(actor.GetUserId())` reaches the same record for a player and nil for a mob, which `TestCounterTaunt_RetortNarratesToThePlayerTaunter` (actions) already pins.

- [ ] **Step 6: Branch at the spell exit**

Replace the body of `fireSpellCounterTier` in `internal/hooks/counter_tier.go`:

```go
func fireSpellCounterTier(room *rooms.Room, out combat.ChannelDefenceResult,
	shape combatvocab.Attack, defender, caster *characters.Character,
	defenderUser, casterUser *users.UserRecord) combat.CounterResult {

	if !out.DefensiveCrit {
		return combat.CounterResult{}
	}

	// Words answer words: a defy crit (charm is the one social spell) fires
	// the counter-taunt, the same dispatch taunt's own exit uses. The swing
	// primitive refuses a defy defence, so this branch is the only way a
	// defied cast is answered. The counter-taunt has its own result type;
	// the four call sites use this function as a statement.
	if out.Defence == combatvocab.DefenceDefy {
		actions.FireCounterTaunt(room, defender, caster, defenderUser, casterUser)
		return combat.CounterResult{}
	}

	res := combat.ExecuteCounter(defender, caster, shape, out.Defence, true)
	if !res.Countered {
		return res
	}

	var countered messaging.Recipient
	counteredId := 0
	if casterUser != nil {
		countered = casterUser
		counteredId = casterUser.UserId
	}
	actions.SendCounterTrio(room, res, countered, counteredId)
	return res
}
```

Update the function's doc comment: replace `the counter-quell pool: the working put down, the gap stepped through` with `the winning defence's counter pool; a defy win goes to the counter-taunt instead`.

- [ ] **Step 7: Patch note**

Insert at the top of `docs/PATCH_NOTES.md`:

```markdown
## 2026-09-18: Words are answered with words

Shrug off a charm decisively and you now mock the attempt instead of
swinging at it, the same way a taunt thrown back stings the one who threw
it. The retort wounds their nerve, not their body.

```

- [ ] **Step 8: Gate**

```bash
gofmt -l internal/ ; go build ./... && go test . ./internal/combat/ ./internal/actions/ ./internal/hooks/ 2>&1 | tail -5
```
Expected: nothing from gofmt; all `ok`, including the pre-existing `TestExecuteTaunt_DefyCritCounterTaunts`, `TestCounterTaunt_BypassesCooldownCostAndAggro`, `TestCounterTaunt_NeverEarnsCounter` and `TestCounterTaunt_RetortNarratesToThePlayerTaunter`.

- [ ] **Step 9: Commit**

```bash
git add internal/combat/counter.go internal/combat/counter_gate_test.go internal/actions/combat_counter.go internal/hooks/counter_tier.go internal/hooks/counter_tier_test.go docs/PATCH_NOTES.md
git commit -F - <<'EOF'
balance(combat): every defy crit answers with a counter-taunt

BEHAVIOUR CHANGE (owner ruling 2026-09-18). A defied charm used to earn
the mob a sword-swing narrated as a retort. The spell exit now routes a
defy win to actions.FireCounterTaunt, the dispatch taunt's own exit is
rewritten to share, so the answer is conviction damage under the RETORT
prefix and never health damage. ExecuteCounter refuses a defy defence so
the two can never disagree again. Patch note added.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
EOF
```

---

## Task 5: The five pools, the golden, and the proof

**Files:**
- Rename: `_datafiles/world/dogmud/defense-messages/counter-melee.yaml` to `counter-dodge.yaml`
- Delete: `_datafiles/world/dogmud/defense-messages/counter-ranged.yaml`
- Create: `counter-parry.yaml`, `counter-block.yaml`
- Modify: `counter-defy.yaml` (re-toned), `counter-quell.yaml` (header)
- Modify: `internal/items/defensive_messages.go:27-39`, `test_helpers_combat.go:55-61`, `defence_pool_test.go:22-30`, `defensive_messages_newly_defendable_test.go:152-154`
- Regenerate: `internal/narration/testdata/stores/defense_messages.golden`
- Create: `tools/counter_pool_rekey_check.py`
- Modify: `docs/PATCH_NOTES.md`

Copy rules for every line (dogmud-player-copy): a template stays at or under 76 characters before names substitute (ansi stripped; the longest line shipped today, in counter-quell, is 76), no numbers, no idiom whose meaning cannot be composed from its words, no em or en dashes, no semicolons. `{actee}` is the COUNTERER, `{actor}` the countered party, as in every counter pool today. Each band has exactly five lines per role, in the same order across the three roles, because `RenderDefenseMessage` picks one index for all three.

- [ ] **Step 1: Rename melee to dodge and rewrite its header**

```bash
git mv _datafiles/world/dogmud/defense-messages/counter-melee.yaml _datafiles/world/dogmud/defense-messages/counter-dodge.yaml
```

Replace the header comment (lines 1-7) and the `optionid` line of `counter-dodge.yaml` with:

```yaml
# Counter narration for a DODGE crit (counters slice): the attacker's swing,
# point-blank shot or hurled working found nothing, and the defender steps
# into the opening it left. Attack-agnostic on purpose: dodge may answer a
# melee move, a same-room shot or a physical spell, and the pool is chosen
# by the DEFENCE that won, not the attack. {actee} is the COUNTERER;
# {actor} is the countered party. Bands: weak = the counter is turned aside
# (no damage), normal = the counter lands, heavy = the counter crits.
optionid: counter-dodge
```

The 45 lines below the header are not touched.

- [ ] **Step 2: Delete the ranged pool**

```bash
git rm _datafiles/world/dogmud/defense-messages/counter-ranged.yaml
```

- [ ] **Step 3: Author `counter-parry.yaml`**

```yaml
# Counter narration for a PARRY crit (counters slice): the defender turned
# the weapon and answers back along it. Parry may answer melee only, so
# every line here is steel on steel. {actee} is the COUNTERER (whose parry
# critted); {actor} is the countered attacker. Bands: weak = the counter is
# turned aside (no damage), normal = the counter lands, heavy = the counter
# crits.
optionid: counter-parry
options:
  weak:
    together:
      actee:
      - '<ansi fg="defense-good">You turn the weapon and cut back along it, but {actor} recovers in time.</ansi>'
      - '<ansi fg="defense-good">You slide your weapon down {actor}''s and thrust, but they twist clear.</ansi>'
      - '<ansi fg="defense-good">You beat {actor}''s weapon aside and lunge, but the lunge finds nothing.</ansi>'
      - '<ansi fg="defense-good">You bind {actor}''s blade and strike over it, but they duck the answer.</ansi>'
      - '<ansi fg="defense-good">Your answering cut follows the parry, but {actor} pulls their guard back.</ansi>'
      actor:
      - '<ansi fg="attack-bad">{actee} turns your weapon and cuts back along it, but you recover in time.</ansi>'
      - '<ansi fg="attack-bad">{actee} slides a weapon down yours and thrusts, but you twist clear.</ansi>'
      - '<ansi fg="attack-bad">{actee} beats your weapon aside and lunges, but the lunge finds nothing.</ansi>'
      - '<ansi fg="attack-bad">{actee} binds your blade and strikes over it, but you duck the answer.</ansi>'
      - '<ansi fg="attack-bad">{actee}''s answering cut follows the parry, but you pull your guard back.</ansi>'
      observer:
      - '<ansi fg="combat">{actee} turns {actor}''s weapon and cuts back along it, but {actor} recovers.</ansi>'
      - '<ansi fg="combat">{actee} slides a weapon down {actor}''s and thrusts, but {actor} twists away.</ansi>'
      - '<ansi fg="combat">{actee} beats {actor}''s weapon aside and lunges, but the lunge finds air.</ansi>'
      - '<ansi fg="combat">{actee} binds {actor}''s blade and strikes over it, but {actor} ducks.</ansi>'
      - '<ansi fg="combat">{actee}''s answering cut follows the parry, but {actor} pulls back in time.</ansi>'
  normal:
    together:
      actee:
      - '<ansi fg="defense-good">You turn the weapon and cut back along it, opening {actor}''s arm.</ansi>'
      - '<ansi fg="defense-good">You slide your weapon down {actor}''s and drive the point home.</ansi>'
      - '<ansi fg="defense-good">You beat {actor}''s weapon aside and land a clean cut.</ansi>'
      - '<ansi fg="defense-good">You bind {actor}''s blade, step in, and strike over the lock.</ansi>'
      - '<ansi fg="defense-good">Your answering cut follows the parry and finds {actor}.</ansi>'
      actor:
      - '<ansi fg="attack-bad">{actee} turns your weapon and cuts back along it, opening your arm.</ansi>'
      - '<ansi fg="attack-bad">{actee} slides a weapon down yours and drives the point home.</ansi>'
      - '<ansi fg="attack-bad">{actee} beats your weapon aside and lands a clean cut.</ansi>'
      - '<ansi fg="attack-bad">{actee} binds your blade, steps in, and strikes over the lock.</ansi>'
      - '<ansi fg="attack-bad">{actee}''s answering cut follows the parry and finds you.</ansi>'
      observer:
      - '<ansi fg="combat">{actee} turns {actor}''s weapon and cuts back along it, opening an arm.</ansi>'
      - '<ansi fg="combat">{actee} slides a weapon down {actor}''s and drives the point home.</ansi>'
      - '<ansi fg="combat">{actee} beats {actor}''s weapon aside and lands a clean cut.</ansi>'
      - '<ansi fg="combat">{actee} binds {actor}''s blade, steps in, and strikes over the lock.</ansi>'
      - '<ansi fg="combat">{actee}''s answering cut follows the parry and finds {actor}.</ansi>'
  heavy:
    together:
      actee:
      - '<ansi fg="defense-good">You turn the weapon and cut back with everything, and {actor} reels.</ansi>'
      - '<ansi fg="defense-good">You throw {actor}''s weapon wide and drive yours deep into the gap.</ansi>'
      - '<ansi fg="defense-good">You catch the blade, roll it aside, and strike {actor} full on.</ansi>'
      - '<ansi fg="defense-good">Your parry becomes a strike in one motion, and {actor} staggers.</ansi>'
      - '<ansi fg="defense-good">You read the swing early and answer it hard, and {actor} buckles.</ansi>'
      actor:
      - '<ansi fg="attack-bad">{actee} turns your weapon and cuts back with everything, and you reel.</ansi>'
      - '<ansi fg="attack-bad">{actee} throws your weapon wide and drives theirs deep into the gap.</ansi>'
      - '<ansi fg="attack-bad">{actee} catches your blade, rolls it aside, and strikes you full on.</ansi>'
      - '<ansi fg="attack-bad">{actee}''s parry becomes a strike in one motion, and you stagger.</ansi>'
      - '<ansi fg="attack-bad">{actee} reads your swing early and answers it hard, and you buckle.</ansi>'
      observer:
      - '<ansi fg="combat">{actee} turns {actor}''s weapon and cuts back hard, and {actor} reels.</ansi>'
      - '<ansi fg="combat">{actee} throws {actor}''s weapon wide and drives a blade deep into the gap.</ansi>'
      - '<ansi fg="combat">{actee} catches {actor}''s blade, rolls it aside, and strikes full on.</ansi>'
      - '<ansi fg="combat">{actee}''s parry becomes a strike in one motion, and {actor} staggers.</ansi>'
      - '<ansi fg="combat">{actee} reads the swing early and answers hard, and {actor} buckles.</ansi>'
```

- [ ] **Step 4: Author `counter-block.yaml`**

```yaml
# Counter narration for a BLOCK crit (counters slice): the attack was taken
# on the guard and the defender answers from behind it. Attack-agnostic on
# purpose: block may answer a melee move, a same-room shot or a physical
# working, and the pool is chosen by the DEFENCE that won, not the attack,
# so nothing here names a blade, an arrow or a spell. {actee} is the
# COUNTERER (whose block critted); {actor} is the countered party. Bands:
# weak = the counter is turned aside (no damage), normal = the counter
# lands, heavy = the counter crits.
optionid: counter-block
options:
  weak:
    together:
      actee:
      - '<ansi fg="defense-good">You take it on your guard and shove in, but {actor} gives ground in time.</ansi>'
      - '<ansi fg="defense-good">You catch the attack square and strike over it, but {actor} slips away.</ansi>'
      - '<ansi fg="defense-good">You brace and drive your shoulder in, but {actor} is already stepping back.</ansi>'
      - '<ansi fg="defense-good">You turn the attack off your guard and swing, but {actor} covers up.</ansi>'
      - '<ansi fg="defense-good">You ram your guard into {actor}, but they keep their feet.</ansi>'
      actor:
      - '<ansi fg="attack-bad">{actee} takes it on their guard and shoves in, but you give ground in time.</ansi>'
      - '<ansi fg="attack-bad">{actee} catches your attack square and strikes over it, but you slip away.</ansi>'
      - '<ansi fg="attack-bad">{actee} braces and drives a shoulder in, but you are already stepping back.</ansi>'
      - '<ansi fg="attack-bad">{actee} turns your attack off their guard and swings, but you cover up.</ansi>'
      - '<ansi fg="attack-bad">{actee} rams their guard into you, but you keep your feet.</ansi>'
      observer:
      - '<ansi fg="combat">{actee} takes it on the guard and shoves in, but {actor} gives ground.</ansi>'
      - '<ansi fg="combat">{actee} catches the attack square and strikes back, but {actor} slips away.</ansi>'
      - '<ansi fg="combat">{actee} braces and drives a shoulder in, but {actor} is already backing off.</ansi>'
      - '<ansi fg="combat">{actee} turns the attack off the guard and swings, but {actor} covers up.</ansi>'
      - '<ansi fg="combat">{actee} rams a guard into {actor}, who keeps their feet.</ansi>'
  normal:
    together:
      actee:
      - '<ansi fg="defense-good">You take it on your guard, shove in, and strike {actor} from behind it.</ansi>'
      - '<ansi fg="defense-good">You catch the attack square and hammer {actor} over the top of it.</ansi>'
      - '<ansi fg="defense-good">You brace, drive your shoulder in, and {actor} takes the hit.</ansi>'
      - '<ansi fg="defense-good">You turn the attack off your guard and land a solid strike on {actor}.</ansi>'
      - '<ansi fg="defense-good">You ram your guard into {actor} and follow it with a hard blow.</ansi>'
      actor:
      - '<ansi fg="attack-bad">{actee} takes it on their guard, shoves in, and strikes you from behind it.</ansi>'
      - '<ansi fg="attack-bad">{actee} catches your attack square and hammers you over the top of it.</ansi>'
      - '<ansi fg="attack-bad">{actee} braces and drives a shoulder in, and you take the hit.</ansi>'
      - '<ansi fg="attack-bad">{actee} turns your attack off their guard and lands a solid strike on you.</ansi>'
      - '<ansi fg="attack-bad">{actee} rams their guard into you and follows it with a hard blow.</ansi>'
      observer:
      - '<ansi fg="combat">{actee} takes it on the guard, shoves in, and strikes {actor} behind it.</ansi>'
      - '<ansi fg="combat">{actee} catches the attack square and hammers {actor} over the top of it.</ansi>'
      - '<ansi fg="combat">{actee} braces and drives a shoulder in, and {actor} takes the hit.</ansi>'
      - '<ansi fg="combat">{actee} turns the attack off the guard and lands a solid strike on {actor}.</ansi>'
      - '<ansi fg="combat">{actee} rams a guard into {actor} and follows it with a hard blow.</ansi>'
  heavy:
    together:
      actee:
      - '<ansi fg="defense-good">You take it on your guard and drive in hard, and {actor} reels.</ansi>'
      - '<ansi fg="defense-good">You catch the attack square and smash {actor} back a full step.</ansi>'
      - '<ansi fg="defense-good">You brace and ram your shoulder home, and {actor} goes stumbling.</ansi>'
      - '<ansi fg="defense-good">You turn the attack away and put a brutal strike into {actor}.</ansi>'
      - '<ansi fg="defense-good">Your guard swallows the attack and your answer crashes into {actor}.</ansi>'
      actor:
      - '<ansi fg="attack-bad">{actee} takes it on their guard and drives in hard, and you reel.</ansi>'
      - '<ansi fg="attack-bad">{actee} catches your attack square and smashes you back a full step.</ansi>'
      - '<ansi fg="attack-bad">{actee} braces and rams a shoulder home, and you go stumbling.</ansi>'
      - '<ansi fg="attack-bad">{actee} turns your attack away and puts a brutal strike into you.</ansi>'
      - '<ansi fg="attack-bad">{actee}''s guard swallows your attack and their answer crashes into you.</ansi>'
      observer:
      - '<ansi fg="combat">{actee} takes it on the guard and drives in hard, and {actor} reels.</ansi>'
      - '<ansi fg="combat">{actee} catches the attack square and smashes {actor} back a full step.</ansi>'
      - '<ansi fg="combat">{actee} braces and rams a shoulder home, and {actor} goes stumbling.</ansi>'
      - '<ansi fg="combat">{actee} turns the attack away and puts a brutal strike into {actor}.</ansi>'
      - '<ansi fg="combat">{actee}''s guard swallows the attack and the answer crashes into {actor}.</ansi>'
```

- [ ] **Step 5: Re-tone `counter-defy.yaml`**

Replace the whole file. Owner's brief: it must read for a charm as well as a taunt, "you tried it, I defied it, and I mocked the attempt", in generic wording. No line names a taunt or a jeer.

```yaml
# Counter narration for a DEFY crit (counters slice): the defender shrugged
# off an attempt on their nerve, a taunt or a charm, and mocks the one who
# made it. No swing, no steel: every line here is words and scorn, and none
# of them names the attempt, because a charm is honeyed and a taunt is not.
# {actee} is the COUNTERER (whose defy critted); {actor} is the one whose
# attempt was defied. Bands: weak = the retort fails to bite (no conviction
# damage), normal = the retort lands, heavy = the retort crits.
optionid: counter-defy
options:
  weak:
    together:
      actee:
      - '<ansi fg="defense-good">You mock {actor}''s attempt, but the mockery fails to bite.</ansi>'
      - '<ansi fg="defense-good">You laugh in {actor}''s face, but they only sneer.</ansi>'
      - '<ansi fg="defense-good">You snap a barb back at {actor}, but it slides off their pride.</ansi>'
      - '<ansi fg="defense-good">You throw {actor}''s words back at them, but they refuse the bait.</ansi>'
      - '<ansi fg="defense-good">Your scorn reaches {actor}, but their nerve holds.</ansi>'
      actor:
      - '<ansi fg="attack-bad">{actee} mocks your attempt, but the mockery fails to bite.</ansi>'
      - '<ansi fg="attack-bad">{actee} laughs in your face, but you only sneer.</ansi>'
      - '<ansi fg="attack-bad">{actee} snaps a barb back at you, but it slides off your pride.</ansi>'
      - '<ansi fg="attack-bad">{actee} throws your own words back at you, but you refuse the bait.</ansi>'
      - '<ansi fg="attack-bad">{actee}''s scorn reaches you, but your nerve holds.</ansi>'
      observer:
      - '<ansi fg="combat">{actee} mocks {actor}''s attempt, but the mockery fails to bite.</ansi>'
      - '<ansi fg="combat">{actee} laughs in {actor}''s face, but {actor} only sneers.</ansi>'
      - '<ansi fg="combat">{actee} snaps a barb back at {actor}, but it slides off their pride.</ansi>'
      - '<ansi fg="combat">{actee} throws {actor}''s words back at them, but the bait is refused.</ansi>'
      - '<ansi fg="combat">{actee}''s scorn reaches {actor}, but their nerve holds.</ansi>'
  normal:
    together:
      actee:
      - '<ansi fg="defense-good">You mock {actor} for the attempt, and they flinch.</ansi>'
      - '<ansi fg="defense-good">You laugh off {actor}''s words and answer with sharper ones.</ansi>'
      - '<ansi fg="defense-good">Your scorn lands, and {actor}''s bravado wavers.</ansi>'
      - '<ansi fg="defense-good">You answer {actor}''s attempt with mockery, and their face falls.</ansi>'
      - '<ansi fg="defense-good">You turn {actor}''s own words against them, and they bite deep.</ansi>'
      actor:
      - '<ansi fg="attack-bad">{actee} mocks you for the attempt, and you flinch.</ansi>'
      - '<ansi fg="attack-bad">{actee} laughs off your words and answers with sharper ones.</ansi>'
      - '<ansi fg="attack-bad">{actee}''s scorn lands, and your bravado wavers.</ansi>'
      - '<ansi fg="attack-bad">{actee} answers your attempt with mockery, and your face falls.</ansi>'
      - '<ansi fg="attack-bad">{actee} turns your own words against you, and they bite deep.</ansi>'
      observer:
      - '<ansi fg="combat">{actee} mocks {actor} for the attempt, and {actor} flinches.</ansi>'
      - '<ansi fg="combat">{actee} laughs off {actor}''s words and answers with sharper ones.</ansi>'
      - '<ansi fg="combat">{actee}''s scorn lands, and {actor}''s bravado wavers.</ansi>'
      - '<ansi fg="combat">{actee} answers {actor}''s attempt with mockery, and their face falls.</ansi>'
      - '<ansi fg="combat">{actee} turns {actor}''s own words against them, and they bite deep.</ansi>'
  heavy:
    together:
      actee:
      - '<ansi fg="defense-good">You tear {actor}''s attempt apart word by word, and they fall silent.</ansi>'
      - '<ansi fg="defense-good">Your scorn cuts to the bone, and {actor}''s composure cracks.</ansi>'
      - '<ansi fg="defense-good">You fling the words back sharper than they came, and {actor} pales.</ansi>'
      - '<ansi fg="defense-good">You strip {actor}''s bluster bare, and shame does the rest.</ansi>'
      - '<ansi fg="defense-good">Your answer lands like a slap, and {actor} falters.</ansi>'
      actor:
      - '<ansi fg="attack-bad">{actee} tears your attempt apart word by word, and you fall silent.</ansi>'
      - '<ansi fg="attack-bad">{actee}''s scorn cuts to the bone, and your composure cracks.</ansi>'
      - '<ansi fg="attack-bad">{actee} flings your words back sharper than they came, and you pale.</ansi>'
      - '<ansi fg="attack-bad">{actee} strips your bluster bare, and shame does the rest.</ansi>'
      - '<ansi fg="attack-bad">{actee}''s answer lands like a slap, and you falter.</ansi>'
      observer:
      - '<ansi fg="combat">{actee} tears {actor}''s attempt apart word by word, leaving them silent.</ansi>'
      - '<ansi fg="combat">{actee}''s scorn cuts to the bone, and {actor}''s composure cracks.</ansi>'
      - '<ansi fg="combat">{actee} flings the words back sharper than they came, and {actor} pales.</ansi>'
      - '<ansi fg="combat">{actee} strips {actor}''s bluster bare, and shame does the rest.</ansi>'
      - '<ansi fg="combat">{actee}''s answer lands like a slap, and {actor} falters.</ansi>'
```

- [ ] **Step 6: Trim the quell header**

`counter-quell.yaml` lines 1-9: replace the first two sentences so the header reads

```yaml
# Counter narration for a QUELL crit (counters slice): the defender put a
# mental working down and steps in through the gap it leaves. Quell answers
# workings only, so every line here is about the spell failing. Never a
# generic riposte line pasted under a spell (the owner's hard requirement).
# {actee} is the COUNTERER; {actor} is the countered caster. Bands:
# weak = the counter is turned aside (no damage), normal = the counter
# lands, heavy = the counter crits. No 'fizzle' here: that word belongs to
# quell's own heavy band only.
```

The 45 lines are not touched.

- [ ] **Step 7: Five constants**

`internal/items/defensive_messages.go`, replace the counter constants block (lines 27-39) with:

```go
const (
	// Counter-narration pools (U6b Task 11, re-keyed by the counters slice).
	// Not defences: each is the narration for the counter EARNED by a
	// defensive crit, named after the defence that won it, which is what
	// CounterPoolFor computes. They ride the same loader, shape and
	// validator as the defence pools. Band semantics differ: weak = the
	// counter is turned aside (no damage), normal = the counter lands,
	// heavy = the counter crits (or, for defy, the retort fails, lands,
	// crits).
	CounterPoolDodge DefencePool = "counter-dodge"
	CounterPoolParry DefencePool = "counter-parry"
	CounterPoolBlock DefencePool = "counter-block"
	CounterPoolQuell DefencePool = "counter-quell"
	CounterPoolDefy  DefencePool = "counter-defy"
)
```

- [ ] **Step 8: Fixture and the two tests that enumerate pools**

`internal/items/test_helpers_combat.go:60`:

```go
		CounterPoolDodge, CounterPoolParry, CounterPoolBlock, CounterPoolQuell, CounterPoolDefy,
```

`internal/items/defensive_messages_newly_defendable_test.go:152-154`:

```go
	counterPools := []DefencePool{
		CounterPoolDodge, CounterPoolParry, CounterPoolBlock, CounterPoolQuell, CounterPoolDefy,
	}
```

`internal/items/defence_pool_test.go`: replace `TestCounterPoolNamesAreTheShippedFileNames` (lines 22-30) with a test that reads the directory, so it can fail:

```go
// The five counter constants are exactly the counter-* files that ship, and
// each is what CounterPoolFor names for its defence. Read from disk so a
// renamed or missing file turns this red.
func TestCounterPoolConstantsAreTheShippedFilesAndTheConversion(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Join(filepath.Dir(here), "..", "..", "_datafiles", "world", "dogmud", "defense-messages")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	shipped := map[DefencePool]bool{}
	for _, e := range entries {
		if name := e.Name(); strings.HasPrefix(name, "counter-") && strings.HasSuffix(name, ".yaml") {
			shipped[DefencePool(strings.TrimSuffix(name, ".yaml"))] = true
		}
	}
	constants := map[DefencePool]combatvocab.Defence{
		CounterPoolDodge: combatvocab.DefenceDodge, CounterPoolParry: combatvocab.DefenceParry,
		CounterPoolBlock: combatvocab.DefenceBlock, CounterPoolQuell: combatvocab.DefenceQuell,
		CounterPoolDefy: combatvocab.DefenceDefy,
	}
	for pool, d := range constants {
		if !shipped[pool] {
			t.Errorf("constant %q has no shipped file", pool)
		}
		if got := CounterPoolFor(d); got != pool {
			t.Errorf("CounterPoolFor(%s) = %q, want the constant %q", d, got, pool)
		}
	}
	for pool := range shipped {
		if _, ok := constants[pool]; !ok {
			t.Errorf("shipped file %q has no constant", pool)
		}
	}
}
```

Add `"os"`, `"path/filepath"`, `"runtime"`, `"strings"` to that file's imports.

- [ ] **Step 9: Build and run items, then boot-validate the pools**

```bash
gofmt -l internal/ ; go build ./... && go test ./internal/items/ 2>&1 | tail -3
```
Expected: `ok`. `TestCounterPoolsRenderTriads` loads the real files and checks five variants per band per role, no leftover tokens, no dashes, no duplicate lines within a pool and audience: it is the author's proofreader.

- [ ] **Step 10: Regenerate the golden, once**

```bash
go test ./internal/narration/... -run TestSnapshotStores -update 2>&1 | tail -2
git diff --stat internal/narration/testdata/stores/
```
Expected: only `defense_messages.golden` changed.

- [ ] **Step 11: Write the check script**

`tools/counter_pool_rekey_check.py`:

```python
#!/usr/bin/env python3
"""Prove the counters-slice golden regeneration changed only what it should.

Compares defense_messages.golden at a base git ref against the working tree
copy. Rows are `pool|band|role => text` and `melee|pool|band|role => text`.

Expected differences, and ONLY these:
  - every counter-melee row reappears as counter-dodge with IDENTICAL text;
  - every counter-ranged row is gone;
  - counter-parry and counter-block rows are new;
  - counter-defy rows may change (re-toned);
  - every other row (dodge, parry, block, quell, defy, counter-quell) is
    byte-identical.
Header comment lines are reported, not compared.

Usage:
  python tools/counter_pool_rekey_check.py --base master
"""
import argparse
import subprocess
import sys

GOLDEN = "internal/narration/testdata/stores/defense_messages.golden"


def rows(text):
    out = {}
    for line in text.splitlines():
        if not line or line.startswith("#"):
            continue
        key, sep, value = line.partition(" => ")
        if not sep:
            continue
        out[key] = value
    return out


def pool_of(key):
    parts = key.split("|")
    return parts[1] if parts[0] == "melee" else parts[0]


def translate(key):
    return key.replace("counter-melee|", "counter-dodge|", 1)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="master")
    args = ap.parse_args()

    # bytes, decoded explicitly: text=True would use the Windows locale codec
    # and mojibake any non-ASCII character into a false difference.
    base_bytes = subprocess.run(["git", "show", f"{args.base}:{GOLDEN}"],
                                check=True, capture_output=True).stdout
    base = rows(base_bytes.decode("utf-8"))
    with open(GOLDEN, encoding="utf-8") as fh:
        new = rows(fh.read())

    problems = []
    seen_new = set()
    for key, text in base.items():
        pool = pool_of(key)
        if pool == "counter-ranged":
            if key in new:
                problems.append(f"counter-ranged row survived: {key}")
            continue
        target = translate(key) if pool == "counter-melee" else key
        seen_new.add(target)
        if target not in new:
            problems.append(f"row missing after regeneration: {target}")
        elif pool == "counter-defy":
            continue
        elif new[target] != text:
            problems.append(f"text changed: {target}\n  base: {text}\n  new:  {new[target]}")

    for key in new:
        if key in seen_new:
            continue
        pool = pool_of(key)
        if pool in ("counter-parry", "counter-block"):
            continue
        problems.append(f"unexpected new row: {key}")

    counts = {}
    for key in new:
        counts[pool_of(key)] = counts.get(pool_of(key), 0) + 1
    print("rows per pool after regeneration:", dict(sorted(counts.items())))
    if problems:
        print("\n".join(problems))
        print(f"\nFAIL: {len(problems)} problem(s)")
        return 1
    print("OK: dodge rows are the old melee rows, ranged rows gone, parry/block new, defy re-toned, rest identical")
    return 0


if __name__ == "__main__":
    sys.exit(main())
```

- [ ] **Step 12: Run it, then prove it can fail**

```bash
python tools/counter_pool_rekey_check.py --base master
```
Expected: the row counts and `OK`.

Sabotage: edit one `counter-dodge|normal|actee` row's text in the golden by one letter, run again, expect `text changed` and `FAIL`; then edit one `dodge|weak|actee` row the same way, expect the same. `git checkout -- internal/narration/testdata/stores/defense_messages.golden` is NOT the way back (it stages a revert of the regeneration): instead re-run the `-update` command from Step 10 and confirm `python tools/counter_pool_rekey_check.py --base master` is `OK` again.

- [ ] **Step 13: Patch note**

Insert at the top of `docs/PATCH_NOTES.md`:

```markdown
## 2026-09-18: Your counter reads like your defence

When a decisive parry, block or sidestep earns you a free answer, the
answer now describes what you did. A parry turns the weapon and cuts back
along it. A block drives in from behind the guard. A sidestep steps into
the opening. Before, all three read the same. Shrugging off a working or
mocking a failed charm or taunt keep their own voices.

```

- [ ] **Step 14: Gate**

```bash
gofmt -l internal/ tools/ 2>/dev/null; go build ./... && go test . ./... 2>&1 | grep -v "^ok\|no test files"; echo "exit $?"
```
Expected: only the exit line.

- [ ] **Step 15: Commit**

```bash
git add _datafiles/world/dogmud/defense-messages/ internal/items/ internal/narration/testdata/stores/defense_messages.golden tools/counter_pool_rekey_check.py docs/PATCH_NOTES.md
git commit -F - <<'EOF'
feat(content): five counter pools named for the defence that won

counter-melee becomes counter-dodge with its lines unchanged;
counter-parry and counter-block are new (45 lines each, three bands,
three roles); counter-defy is re-toned so every line reads for a charm
as well as a taunt (closes ledger row 5); counter-ranged retires (its
shot flavour is filed as an M6 ledger row in the docs commit).
defense_messages.golden regenerated once; tools/counter_pool_rekey_check.py
proves the dodge rows are the old melee rows byte for byte, only ranged
rows vanished, and every non-counter row is unchanged. The check was
sabotaged twice and went red both times.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
EOF
```

---

## Task 6: Docs

**Files:**
- Modify: `internal/combat/context.md:1649`
- Modify: `internal/items/context.md:419-428`
- Modify: `internal/combat/counter.go` (comment sweep)
- Modify: `docs/superpowers/audits/messaging-m6-content-ledger.md:89` and a new row 34
- Modify: `docs/README.md`

- [ ] **Step 1: `internal/combat/context.md`**

Replace the `combat/counter.go` row (line 1649) with:

```markdown
| `combat/counter.go` | U6b Task 10 counter tier, re-keyed by the counters slice: `ExecuteCounter(defender, attacker, shape, defence, sameRoom) CounterResult` — one free counter-swing for a defensive crit, priced by `CounterDamagePercent` (0 = off-switch, handled here because `CalcRawDamage` treats `itemMult <= 0` as "unset" 0.30), routed through `ExecuteSkillMove` with `IsCounter` so the countered party defends it (charged + progressed: the countered-party economy) and no counter can chain. Three refusals in the primitive: not `sameRoom` (the cross-room shot), `shape.Targeting != TargetSingle` (area and multi attacks earn no counter, owner ruling 2026-09-18), and `defence == DefenceDefy` (words answer words: every defy crit counter-taunts via `internal/actions.FireCounterTaunt`, which this package cannot call, so every exit branches on defy first). Narration is rendered from the WINNING DEFENCE's pool, `items.CounterPoolFor(defence)` (counter-dodge, counter-parry, counter-block, counter-quell, counter-defy; bands: weak = turned aside, normal = lands, heavy = crits), damage description appended to the two personal lines only, generic fallback when pools are not loaded. `BuildCounterTauntMessages` renders the defy retort triad from counter-defy. `CounterResult.CountererUserId` lets wrappers dispatch AFTER the move outcome (`actions.DispatchCounterMessages`) |
```

Also in that file, line 397 says `its callers fire ExecuteCounter (or the defy counter-taunt in internal/actions)`; append ` — since the counters slice, every defy crit takes the counter-taunt path and area attacks earn no counter` to that sentence.

- [ ] **Step 2: `internal/items/context.md`**

Replace lines 421-428 (from `Four counter-narration pools` through `(crit, margin)` inputs accordingly).`) with:

```markdown
Five counter-narration pools ride the same loader, shape, and validator (U6b
Task 11, re-keyed by the counters slice): `CounterPoolDodge`,
`CounterPoolParry`, `CounterPoolBlock`, `CounterPoolQuell` and
`CounterPoolDefy`, the narration for the counter earned by a defensive crit,
named for the DEFENCE that won it. `CounterPoolFor(combatvocab.Defence)` is
the conversion (`counter-` plus the defence's name; none maps to the empty
pool). Bands are reinterpreted (weak = the counter is turned aside, normal =
it lands, heavy = it crits; `internal/combat` maps outcomes to `(crit,
margin)` inputs accordingly). The dodge and block pools are written
attack-agnostic because either may answer a melee move, a same-room shot or
a physical spell; parry answers steel only and quell answers workings only;
defy reads for a charm as well as a taunt.
```

- [ ] **Step 3: Source comment sweep**

```bash
grep -n "until the counters slice\|ORIGINAL attack's type\|counter-melee\|counter-ranged\|counterPoolFor" internal/combat/counter.go internal/items/defensive_messages.go internal/actions/combat_counter.go internal/hooks/counter_tier.go
```
Rewrite every hit so it describes the shipped behaviour (pool by the winning defence; every defy crit counter-taunts; area attacks earn none). Expected after: no output.

- [ ] **Step 4: Ledger**

Row 5 (line 89): change its Status cell from `open` to `closed by the counters slice, 2026-09-18: counter-defy re-toned, no line names a taunt`.

Append after row 33 (line 129):

```markdown
| 34 | Per-attack counter flavour under a defence-keyed pool. The counters slice keys counter narration by the defence that won, so a dodge or block crit against a point-blank shot reads the attack-agnostic dodge or block pool; `counter-ranged.yaml`'s 45 lines (27 naming the shot, the aim or the shooter) were retired with it, and a dodge or block crit against a physical working no longer reads counter-quell's "the working put down" lines. Restoring the flavour means an attack-type override file beside each defence pool (`counter-dodge-ranged`, `counter-block-spell`) with a fallback lookup | 45 retired lines recoverable from git (`git show f53b07d76:_datafiles/world/dogmud/defense-messages/counter-ranged.yaml`); four override files to author or copy | A fallback lookup in `items.CounterPoolFor` or beside it; owner declined it for the slice (2026-09-18, "five pools by defence") | Counters spec, 2026-09-18, "Out of scope, carried" | The owner was shown the three keyings and chose the pure defence key; the ranged and spell flavour is a content question for M6, not a mechanism gap | open |
```

- [ ] **Step 5: `docs/README.md`**

Insert directly above the row for `superpowers/specs/2026-09-18-messaging-counters-design.md`:

```markdown
| [`superpowers/plans/2026-09-18-messaging-counters.md`](superpowers/plans/2026-09-18-messaging-counters.md) | Implementation plan for the counters spec below, in eight tasks. The pool conversion lands first; the primitive takes the winning defence with a parity test over every reachable cell of the matrix and an invariant test that a defensive crit always names its defence; the two behaviour changes (area attacks earn no counter, every defy crit counter-taunts) are their own flagged commits with patch notes and re-key the root line-number guard; the data commit renames melee to dodge, authors parry and block, re-tones defy, deletes ranged, regenerates the golden once and proves it with `tools/counter_pool_rekey_check.py`, sabotaged before trusted; docs and ledger; verification and PR; and one adversarial playtest whose fixture is a weak attacker against a strong arena opponent so that every attack type is decisively defended and each defence line can be read beside its counter line |
```

- [ ] **Step 6: Gate and commit**

```bash
python tools/context_md_audit.py 2>&1 | tail -5
go test ./internal/combat/ ./internal/items/ 2>&1 | tail -2
git add internal/combat/context.md internal/items/context.md internal/combat/counter.go internal/items/defensive_messages.go internal/actions/combat_counter.go internal/hooks/counter_tier.go docs/superpowers/audits/messaging-m6-content-ledger.md docs/README.md
git commit -F - <<'EOF'
docs: counters slice in the combat and items context files, ledger, index

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
EOF
```
Expected: the audit names no phantom symbol in `combat` or `items`.

---

## Task 7: Verification and the PR

**Files:** none new.

- [ ] **Step 1: The sweep grep, proven on master first**

```bash
git grep -l "CounterPoolMelee\|CounterPoolRanged\|counter-melee\|counter-ranged\|counterPoolFor" master -- '*.go' '*.md' '*.yaml' '*.golden' | grep -v docs/superpowers | wc -l
grep -rln "CounterPoolMelee\|CounterPoolRanged\|counter-melee\|counter-ranged\|counterPoolFor" --include=*.go --include=*.md --include=*.yaml --include=*.golden . | grep -v "docs/superpowers"
```
Expected: the first prints 12 (the grep can match); the second prints only `./docs/roadmaps/UNIFIED_RESOLUTION_ROADMAP.md` (historical, stays). Any other file is a leftover: fix it.

- [ ] **Step 2: Pre-push gates**

```bash
gofmt -l internal/ modules/
go build ./... && go test . ./... 2>&1 | grep -v "^ok\|no test files"; echo "exit $?"
golangci-lint run --new-from-rev=master 2>&1 | tail -3
```
Expected: nothing, only the exit line, and `0 issues`.

- [ ] **Step 3: Boot check in a detached worktree**

```bash
git worktree add --detach C:/tmp/dogmud-boot-check HEAD
cp _datafiles/config.yaml C:/tmp/dogmud-boot-check/_datafiles/config.yaml
cd C:/tmp/dogmud-boot-check && go build -o boot-check.exe . && timeout 180 ./boot-check.exe > boot.log 2>&1; echo "exit $?"
grep -cE "^panic:|goroutine [0-9]+ \[running\]|runtime error|PANIC" boot.log
grep -c "Server Ready" boot.log
cd "/c/Users/Calabe Davis/workspace/DOGMud" && git worktree remove --force C:/tmp/dogmud-boot-check || (rm -rf C:/tmp/dogmud-boot-check; git worktree prune)
```
Expected: exit 124, `0`, `1`. A failed boot exits 0 and logs `PANIC`; judge by the two greps, never by the exit code. The loader validates every pool at boot, so a band with four lines panics here and nowhere else.

- [ ] **Step 4: Push and open the PR**

```bash
git push -u origin feature/messaging-counters
gh pr create --repo pruuk/DOGMud --base master --head feature/messaging-counters --title "Messaging counters slice: the counter answers the defence that won" --body-file - <<'EOF'
Spec: docs/superpowers/specs/2026-09-18-messaging-counters-design.md (owner-approved 2026-09-18).
Plan: docs/superpowers/plans/2026-09-18-messaging-counters.md.

Counter narration is chosen by the defence that won, from five pools:
counter-dodge (counter-melee renamed, lines unchanged), counter-parry and
counter-block (new), counter-quell (unchanged), counter-defy (re-toned so
it reads for a charm as well as a taunt). counter-ranged retires; its shot
flavour is ledger row 34.

BEHAVIOUR CHANGES, each its own commit with a patch note:
1. Only a single-target attack earns a counter. The seven physical area
   spells and core-drain no longer give each victim a free swing at the
   caster; multi rides along (owner: "no counters on multi, same as aoe").
2. Every defy crit counter-taunts. A defied charm is answered with
   conviction damage under RETORT, never a health-damage swing.

Proof: parity test over every reachable (attack, defence) cell; invariant
test that a defensive crit always names its defence; gate tests at the
primitive and at the spell exit; the golden regenerated once and checked
by tools/counter_pool_rekey_check.py (dodge rows are the old melee rows,
only ranged rows vanished, every non-counter row identical), sabotaged
twice before trusted. Root line-number guard re-keyed. Boot check clean.

Playtest: see the final commit's report summary.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
```
Read the URL `gh` prints and confirm it says `pruuk/DOGMud`.

```bash
gh pr checks --repo pruuk/DOGMud --watch
```
Then confirm every expected workflow ran with `gh run list --repo pruuk/DOGMud --branch feature/messaging-counters`. A green `--watch` can return before a path-filtered job registers.

---

## Task 8: The adversarial playtest

**Files:**
- Create: `tools/playtest/profiles/counters.yaml`
- Create: `tools/playtest/goals/2026-09-18-counters-slice.yaml`

The fixture inverts the usual shape: the TESTER is a weak attacker with a broad kit, and the arena opponent is strong, so the tester's own attacks are decisively defended and the tester reads, as the countered party, the defence line and then the counter line for every attack type. Defensive crits need the defender's margin two standard deviations over the attacker's; low attack stats against a high-gold arena opponent make that common rather than a one-in-forty wait. The tester's health and pools are deep so the fight lasts.

- [ ] **Step 1: The profile**

`tools/playtest/profiles/counters.yaml`:

```yaml
role: user
username: template-counters
character:
  name: Counter Bait
  description: >
    Synthetic profile for the counters-slice playtest. A weak attacker with
    one of every attack type (a special move, a sling, a single physical
    working, a mental working, an area working, a taunt and a charm) and
    deep pools, seeded in the Rift Chamber so Sable can start an arena
    fight at high gold. The opponent's defences should crit often against
    these stats; that is the point.
  roomid: 5000
  zone: Thornwall City
  speciesid: 1
  stats:
    strength:
      base: 30
    dexterity:
      base: 30
    perception:
      base: 30
    vitality:
      base: 150
    willpower:
      base: 30
    charisma:
      base: 30
  health: 6000
  stamina: 3000
  conviction: 3000
  gold: 30000
  skills:
    weapon-combat: 3
    unarmed-combat: 3
    ranged-combat: 3
    spellcasting: 3
    rhetoric: 3
  spellbook:
    kinetic-hurl: 1
    mind-spike: 1
    sparks: 1
    charm: 1
  items:
  - itemid: 30064
  - itemid: 30064
  equipment:
    weapon:
      itemid: 10038
```

If the profile loader rejects a field, `playtestrun` fails closed and names it; fix the profile, not the loader.

- [ ] **Step 2: The goals file**

`tools/playtest/goals/2026-09-18-counters-slice.yaml`:

```yaml
# Counters slice gate: every counter now reads like the defence that earned
# it, area workings earn no counter, and a defied charm is mocked, not hit.
#
# You are the ATTACKER and you are weak on purpose. Your opponent will turn
# most of your attacks aside, and when a defence is decisive it earns them a
# free answer, printed right after the defence line with a "COUNTER!" or
# "RETORT!" prefix. The whole test is whether those two lines AGREE: a line
# that says your blow was parried must be followed by a counter about
# turning the weapon and cutting back along it, never about stepping into an
# opening or driving in from behind a guard.
#
# Not every attack will be decisively defended. Repeat each attack until you
# have seen at least two counters for it, or ten attempts, whichever comes
# first, and say which.
ephemeral:
  profile: counters
  start_room: 5000
  budgets:
    wall_clock: 45m

goals:
  - >-
    Establish the fixture. You start in the Rift Chamber with Sable. Run
    `ask sable arena 5000` to start an arena fight against a strong
    opponent. If the fight does not start, try `ask sable arena 2000` and
    report what Sable said. Once an opponent is present, `look` and note
    its name. Do not flee; your health is very deep.
  - >-
    Special move. Use `kick <opponent>` repeatedly. Each time your kick is
    turned aside, record the defence line VERBATIM (it will say you were
    dodged, parried or blocked). Whenever a "COUNTER!" line follows, record
    it verbatim directly under the defence line it followed. Stop after two
    counters or ten kicks.
  - >-
    Judge the melee counters. For each pair you recorded: a PARRY defence
    line must be followed by a counter about the weapon being turned and
    cut back along, a BLOCK defence line by a counter about a guard, a
    shoulder or driving in, and a DODGE defence line by a counter about
    stepping into an opening or striking at a gap. Any pair where the
    counter's action does not match the defence is the single most
    important thing to report. Quote both lines.
  - >-
    Point-blank shot. Use `shoot <opponent>` repeatedly (you carry a sling
    and shot; if the game says the weapon is not loaded, `load` it and say
    so). Record every defence line and every "COUNTER!" line that follows
    it, verbatim, as pairs. A shot can only be dodged or blocked, so the
    counter must read as a dodge or block answer, never as a parry. Stop
    after two counters or ten shots.
  - >-
    Single-target physical working. Use `cast kinetic-hurl <opponent>`
    repeatedly. Record the defence line and any "COUNTER!" line as pairs.
    The counter must read as a dodge or block answer. Stop after two
    counters or ten casts.
  - >-
    Mental working. Use `cast mind-spike <opponent>` repeatedly. Record the
    pairs. The defence line will say your working was quelled or shrugged
    off, and the counter must be about the working being put down or the
    spell unravelling, never about a weapon, a guard or an opening. Stop
    after two counters or ten casts.
  - >-
    Area working. Use `cast sparks` (no target) at least ten times. Record
    every defence line. NO "COUNTER!" line may EVER follow an area working,
    however decisively it was turned aside. If one does, quote it: that is
    a bug this change exists to fix.
  - >-
    Taunt. Use `taunt <opponent>` repeatedly. When your taunt is thrown
    back, the line carries a "RETORT!" prefix and your conviction (not your
    health) should fall. Record two retorts verbatim and check your
    `status` before and after one to confirm which pool dropped. Stop after
    two retorts or ten taunts.
  - >-
    Charm. Use `cast charm <opponent>` repeatedly. When the charm is
    decisively defied, a "RETORT!" line must follow, NEVER a "COUNTER!"
    line, and the retort must read as mockery of your attempt, in words that
    make sense for a charm (it must not call your charm a taunt). Check
    `status` before and after: conviction falls, health does not. Record
    two retorts verbatim. Stop after two retorts or ten casts.
  - >-
    Report. List every defence/counter pair you recorded, grouped by attack.
    Then answer, with quotes: did any counter contradict its defence line;
    did any area working earn a counter; did any defied charm earn a swing
    instead of a retort; did any retort call a charm a taunt; did any line
    show a raw number; did any line read as broken English or contain a
    token like {actor}. Also report anything else that looked wrong while
    fighting, but keep it separate from the pairs.
```

- [ ] **Step 3: Run it**

Confirm the harness is present (`ls ../gomud-playtest-harness/mudagent.exe`); if it is missing, restore it per the dogmud-playtesting skill. Then:

```text
/playtest local --checkout "C:/Users/Calabe Davis/workspace/DOGMud" bug-finder 2026-09-18-counters-slice.yaml
```

The checkout must be at the branch head. After the run, confirm the container is gone (`docker ps`) and remove it with `docker rm -f <name>` if `playtestrun stop` left it.

- [ ] **Step 4: Triage**

For every finding in the report: a contradiction between a defence line and its counter, a counter after an area working, a swing after a defied charm, or a retort that names a taunt is a defect in THIS slice and is fixed on the branch (test first, then the fix, then re-run the affected goal) before merge. Copy, wording and balance observations that are not defects go to `docs/superpowers/audits/messaging-m6-content-ledger.md` as rows or to memory. Write the findings to memory before the report is discarded (reports are gitignored).

- [ ] **Step 5: Commit the fixture and any fixes**

```bash
git add tools/playtest/profiles/counters.yaml tools/playtest/goals/2026-09-18-counters-slice.yaml
git commit -F - <<'EOF'
test(playtest): counters-slice fixture and goals

A weak attacker with one of every attack type against a strong arena
opponent, so every attack is decisively defended and each defence line
can be read beside its counter line. Report summary: <one line per
attack type, and the verdict on the three behaviour questions>.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>
EOF
git push
```

Then re-run `gh pr checks --repo pruuk/DOGMud --watch`, confirm the runs, and merge with `gh pr merge <n> --repo pruuk/DOGMud --merge --delete-branch`. Do not deploy; the owner runs deploys.

---

## Self-review against the spec

| Spec section | Task |
|---|---|
| 1 Pools (five files, dodge = melee verbatim, parry and block new, defy re-toned, ranged deleted, `CounterPoolFor`) | 1, 5 |
| 2 Re-key (primitive takes the defence, `CounterResult.Defence`, exits pass the winner, none refused, invariant test) | 2 |
| 3 Area gate in the primitive, drain call site and field deleted, multi rides along | 3 |
| 4 Every defy crit counter-taunts, `FireCounterTaunt` exported and shared, primitive refuses defy, charm test | 4 |
| 5 Parity test over the matrix; golden regenerated once with a sabotaged check; root package in every gate; boot check | 2, 5, 7 |
| 6 Two flagged behaviour commits with patch notes | 3, 4 (plus the narration note in 5) |
| 7 Docs: both context files, source comments, patch notes, ledger row 5 closed and row 34 added, README | 5, 6 |
| 8 Playtest: weak attacker, one goal per attack type, pairs quoted, findings to memory | 8 |
| Out of scope: riposte literals (M4e), per-attack flavour (M6 row 34), bands (M4c) | untouched |

Type consistency: `ExecuteCounter(defender, attacker, shape, defence, sameRoom)` in Tasks 2, 3, 4; `items.CounterPoolFor(combatvocab.Defence) DefencePool` in Tasks 1, 2, 4, 5; `actions.FireCounterTaunt(room, counterer, countered, countererUser, counteredUser) CounterTauntResult` in Task 4 only; `counterPoolNarrationFixture(combatvocab.Defence)` defined in Task 2, used in Task 4; the five constants named in Task 5 and Task 6 only.
