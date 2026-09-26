# Lighting plan 5a: carried light with real strengths — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give every carried light a real strength on the graded light scale, add one trim function shared by every adjustable source, a `light` equipment slot, four working light items, `hood` / `unhood`, `cancel <spell>`, item nouns, and the help pages that explain it.

**Architecture:** Every light is a condition record. A new `light_strength` effect carries a source's full strength (a literal for items, `magnitude` for spells); the record stores its trimmed output and hood state. `rooms/lighting.go` combines every record's current output as its own term. One pure function, `lightscale.Trim`, computes the minimum cut from full strength that keeps a room inside the bearer's comfortable range, and `Room.TrimLightFor` runs it for a mover at the moment they enter a room.

**Tech Stack:** Go (the DOGMud fork of GoMud), YAML world data under `_datafiles/world/dogmud`, the web client in `_datafiles/html/public`.

**Spec:** `docs/superpowers/specs/2026-09-26-lighting-plan5-light-as-play-design.md`. Read it first; this plan does not repeat its reasoning.

---

## Before you start

- **Work in a worktree** on a new branch `feature/lighting-plan5a-carried-light`, created from `docs/lighting-plan5-spec` so the spec and this plan ship in the same PR. Use `superpowers:using-git-worktrees`. The main checkout's `_datafiles/config.yaml` carries owner edits under the git skip-worktree bit; a fresh worktree has the plain `HEAD` blob, which is what Task 7 must edit.
- **`-race` cannot run on this machine** (no C compiler). Use plain `go test`.
- **Never `git add -A` or `git add .`.** Stage named paths only, and check `git show --stat HEAD` after every commit.
- **Edit YAML and Go with the Edit tool, never `sed` or a Python read-modify-write.** `sed` silently rewrites CRLF files; room and item YAMLs are LF, check with `git diff --numstat`.
- `grep -c` exits 1 on zero matches; run "expect zero" checks on their own, not in an `&&` chain.
- A test binary never loads `config.yaml`: it runs on Go defaults and on `_datafiles/world/default` unless a test points `FilePaths.DataFiles` elsewhere. Pin config with `configs.SetConfigForTest`.

## Facts verified against source (2026-09-26, master `cf502a2f8`)

| # | Fact | Where |
|---|---|---|
| 1 | Carried light is one flat term of `cfg.DimBelow` if anyone in the room has the `lightsource` flag. | `internal/rooms/lighting.go:106` |
| 2 | `composeLight(cfg, celestial, skyFilter) LightTerms` is the one computation behind `LightLevel` and `LightTerms`. | `internal/rooms/lighting.go` |
| 3 | `lightscale.Combine(step, terms...)` and `Attenuate`; `Absent()` is `-Inf`; `present` is unexported. | `internal/lightscale/lightscale.go:27,43,55,97` |
| 4 | `EmitsLight Flag = "lightsource"`, listed in `AllFlags`; `ValidateFlags` rejects unknown flags at load. | `internal/conditions/conditionspec.go:66,135,389` |
| 5 | The seven code reads of the flag: `actions/skill_helpers.go:34`, `characters/description.go:158`, `hooks/Awareness_LightChange.go:69`, `hooks/NewTurn_PruneConditions.go:139`, `rooms/rooms.go:1643,1739`, `usercommands/go.go:819`. Tests using it: `actions/sneak_test.go` (3), `behaviortree/sight_test.go` (1), `hooks/narration_testhelpers_test.go` (2), `mutations/describe_test.go` (1), `usercommands/look_exit_visibility_test.go` (1). `mutations/describe.go:126` has a `"lightsource"` string case. | grep |
| 6 | `Condition` record fields end with `Magnitude float64` and `Stacks []Stack`; saves use `gopkg.in/yaml.v2`. | `internal/conditions/conditions.go:14-36`, `internal/users/autosave_prepare.go:9` |
| 7 | `Conditions.AddConditionMagnitude(id, triggers, magnitude)` sets `Magnitude`; re-adding a held record resets only `TriggersLeft`, `RoundCounter`, `Permanent`. | `internal/conditions/conditions.go:302-306,339-359` |
| 8 | `Conditions.Effect(kind)` aggregates held records by product, sum, cap or max. | `internal/conditions/effects.go` |
| 9 | `Conditions.GetConditions(ids...) []*Condition` returns held unexpired records. | `internal/conditions/conditions.go:515` |
| 10 | `SightThroughWindow`, `clampShift`, `windowDazzleEdge = 75`, `windowShiftCap = 24`, `windowFloor = 1`. | `internal/messaging/window.go` |
| 11 | `Character.NightVisionStrength() int`. | `internal/characters/vision.go` |
| 12 | `rooms` imports `messaging`; `messaging` does not import `rooms`. | grep |
| 13 | `MoveToRoom` adds the player with `newRoom.AddPlayer(userId)` then queues `events.RoomChange`; `Room.AddMob` appends to `r.mobs` at its end. | `internal/rooms/roommanager.go:356-481`, `internal/rooms/rooms.go:1153-1184` |
| 14 | Spell conditions are applied at exactly four sites, each `X.AddCondition(conditionId, "spell")`. Casters: `casterChar` (783), `user.Character` (1137), `&mob.Character` (1502), `&caster.Character` (1763). | `internal/hooks/spell_resolution.go:783,1137,1502,1763` |
| 15 | `UserRecord.AddConditionMagnitude(id, triggers, magnitude, source)` queues an `events.Condition` with `Magnitude` and `Triggers`; `Mob` has only `AddCondition`. `ApplyConditions` routes a non-zero magnitude through `AddConditionMagnitude`. | `internal/users/userrecord.go:462`, `internal/mobs/mobs.go:859`, `internal/hooks/Condition_ApplyConditions.go:102` |
| 16 | `SpellData.CasterStatValue(stats.Statistics) int` reads the spell's `primarystat` `ValueAdj`; the spell skill is `skills.Spellcasting`. `spells.ResolveSpell(token)` resolves id, alias or name. | `internal/spells/spells.go:167,254`, `internal/characters/spells.go:35` |
| 17 | `Worn` slot fields, `AllSlots()` (the guarded single source of truth), `StatMod`, a `"worn - <slot>"` pointer switch, `GetAllSlotTypes`, the wear switch, and the unequip chain. `Tail` is the precedent for a plain slot. | `internal/characters/worn.go` |
| 18 | Other slot lists: `characters/combat.go:168,213,252`, `characters/inventory.go:520`, `itemvalue/types.go:38`, `itemvalue/delta.go:31,101,163,206`, `hooks/PlayerSpawn_HandleJoin.go:194`, `goals/catalog/mastery_equip.go:158`, `planners/mastery_equip.go:147`, `usercommands/enchant_slot.go:150`, `caravan/visit.go:335`, `items/itemspec.go:61,126`. Web client: `webclient-pure.html:3315` (`EQUIPPABLE_TYPES`), `static/js/item-icon-map.js:177,210`. | grep |
| 19 | `equip` accepts only `Type == weapon` or `Subtype == wearable`. | `internal/usercommands/equip.go:94` |
| 20 | Rooms have `Nouns map[string]string`; `look` tries `FindItem` BEFORE room nouns. | `internal/rooms/rooms.go:113`, `internal/usercommands/look.go:306,357` |
| 21 | `cancel` aborts only an activity; commands register as `{Func, AllowedWhenDowned, AllowedInCombat, AdminOnly}`. | `internal/usercommands/cancel.go`, `usercommands.go:33,84` |
| 22 | Every command needs a help file (`helpfile_completeness_test.go`, with `commandHelpAliases`). Non-command topics go under `general:` in `keywords.yaml`; extra words under `help-aliases:`. | `internal/usercommands/helpfile_completeness_test.go`, `_datafiles/world/dogmud/keywords.yaml:147,227` |
| 23 | Lighting knobs are declared in `config.balance.go` (around line 1207), defaulted in `config.balance.lighting.go` `validateLighting`, read through `configs.Lighting` (`config.lighting_accessor.go`). None is in `config.yaml`; the darkness knobs sit at `config.yaml:913-928`. | `internal/configs/` |
| 24 | Next free condition ids start at 124; the highest armor item id is 20095. | `python tools/id_inventory.py --type conditions`, `--type items` |
| 25 | Oil Lantern 40038 (`type: object`) is in 9 mob files; Tallow Candle 40077 in 8 mob files and 2 shop files. Quest 14 gives 40038. | grep |
| 26 | Both worlds ship condition 1 with `flags: [lightsource]`; test binaries load `_datafiles/world/default` by default. | `_datafiles/world/{dogmud,default}/conditions/1-illumination.yaml`, `internal/configs/config.filepaths.go:23` |
| 27 | The parity and day-cycle goldens sample observers who carry no light, so a correct 5a moves neither. | `lighting_parity_golden_test.go`, `lighting_daycycle_golden_test.go` |
| 28 | Test helpers: `conditions.SeedConditionsForTest`, `items.SeedItemsForTest`, `users.SeedUsersForTest`, `users.NewTestUser` (its character has `conditions.New()`), `rooms.SeedRoomsForTest`, `characters.New()`, `c.SetSkill(name, level)`, `c.Stats.Willpower.ValueAdj`. `Item.GetSpec()` returns an `ItemSpec` value; `items.GetItemSpec(id)` a pointer. `Character.Wear` accepts only a weapon or `Subtype == wearable`. | grep, `internal/items/items.go:321`, `itemspec.go:717`, `internal/characters/worn.go:576` |

## File structure

| File | Responsibility |
|---|---|
| `internal/lightscale/trim.go` (new) | `Polarity`, `Trim`: the one adjustment function |
| `internal/messaging/window.go` | `LightTrimTarget`: the bearer's light target |
| `internal/conditions/light.go` (new) | `LightTrim` state, `LightMax`, `LightNow`, `SetLightOutput`, `ResetLight`, `LightSources` |
| `internal/conditions/effects.go`, `conditionspec.go`, `conditions.go` | the effect kind, two flags, three record fields, validation, reset on a fresh magnitude |
| `internal/characters/light.go` (new) | `EmitsLight`, `LightTerms` |
| `internal/rooms/lighting.go` | per-record carried terms, `Raw` on `LightTerms`, exclusion for trimming |
| `internal/rooms/light_trim.go` (new) | `Room.TrimLightFor` |
| `internal/hooks/light_spell.go` (new) | `lightSpellApplication`: glow strength and duration |
| `internal/usercommands/hood.go` (new) | `hood`, `unhood` |
| `internal/usercommands/cancel.go` | `cancel <spell>` |
| `internal/characters/itemnouns.go` (new) | `FindItemNoun` |
| world data, help templates, `keywords.yaml`, `config.yaml` | content |

---

### Task 1: `lightscale.Trim`, the one adjustment function

**Files:**
- Create: `internal/lightscale/trim.go`
- Test: `internal/lightscale/trim_test.go`

- [ ] **Step 1: Write the failing test**

```go
package lightscale

import (
	"math"
	"testing"
)

func TestTrimLightLandsExactlyOnTarget(t *testing.T) {
	for _, others := range []float64{0, 20, 50, 60, 73} {
		out := Trim(8, others, 100, 74, Brightens)
		if got := Combine(8, others, out); math.Abs(got-74) > 1e-9 {
			t.Errorf("others %v: Combine(others, Trim) = %v, want 74", others, got)
		}
	}
}

func TestTrimLightIsCappedAtFullStrength(t *testing.T) {
	if got := Trim(8, 0, 54, 74, Brightens); got != 54 {
		t.Errorf("a weak lantern in a faint room runs at %v, want its full 54", got)
	}
}

func TestTrimLightInAnUnlitRoomIsTheTarget(t *testing.T) {
	if got := Trim(8, Absent(), 90, 74, Brightens); got != 74 {
		t.Errorf("a strong glow in a cave trims to %v, want 74", got)
	}
	if got := Trim(8, Absent(), 54, 74, Brightens); got != 54 {
		t.Errorf("a weak lantern in a cave runs at %v, want 54", got)
	}
}

func TestTrimLightGoesDarkWhenTheRoomIsAlreadyBright(t *testing.T) {
	for _, others := range []float64{74, 80} {
		if got := Trim(8, others, 90, 74, Brightens); !math.IsInf(got, -1) {
			t.Errorf("others %v: Trim = %v, want Absent", others, got)
		}
	}
}

// Darkness is the same function inverted (5d wires it): the least darkness
// that keeps the room at or above the bearer's floor.
func TestTrimDarknessIsTheInverse(t *testing.T) {
	cases := []struct{ others, max, target, want float64 }{
		{70, 40, 25, 40},        // wants 45, capped at full strength
		{30, 40, 25, 5},         // cuts just to the floor
		{20, 40, 25, 0},         // already below the floor: no darkness
		{Absent(), 40, -30, 30}, // an unlit room counts as 0
	}
	for _, c := range cases {
		if got := Trim(8, c.others, c.max, c.target, Darkens); got != c.want {
			t.Errorf("Trim(dark, others %v, max %v, target %v) = %v, want %v", c.others, c.max, c.target, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/lightscale/ -run Trim`
Expected: FAIL, `undefined: Trim` and `undefined: Brightens`.

- [ ] **Step 3: Implement**

```go
package lightscale

import "math"

// Polarity says which way an adjustable source pushes its room: a light
// raises it, a darkness lowers it. It is the only difference between the two
// kinds of adjustable source, so one Trim serves both (owner ruling,
// 2026-09-26: "the same function, just inverted").
type Polarity int

const (
	Brightens Polarity = 1
	Darkens   Polarity = -1
)

// Trim returns the output an adjustable source should run at: the smallest
// cut from its full strength that keeps the room on the bearer's side of
// target.
//
// others is the room's light with this source left out, Absent when nothing
// else lights it.
//
// For a light, max and the result are light-scale terms fed to Combine. The
// result solves Combine(others, out) == target exactly, capped at max; it is
// Absent when the room already reaches target without this source. A linear
// "target - others" is wrong here: on a log scale adding a source does not add
// its value.
//
// For a darkness, max and the result are points subtracted from the combined
// light. The result is the cut that lands the room on target, capped at max,
// and 0 when the room is already at or below target. An Absent room counts as
// 0, the darkest light that occurs naturally.
func Trim(step, others, max, target float64, p Polarity) float64 {
	if !(step > 0) {
		step = 1
	}
	if p == Darkens {
		level := others
		if !present(level) {
			level = 0
		}
		cut := level - target
		if cut <= 0 || max <= 0 {
			return 0
		}
		return math.Min(cut, max)
	}
	if !present(others) {
		return math.Min(target, max)
	}
	if others >= target {
		return Absent()
	}
	need := target + step*math.Log2(1-math.Exp2((others-target)/step))
	return math.Min(need, max)
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/lightscale/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/lightscale/trim.go internal/lightscale/trim_test.go
git commit -m "feat(lightscale): Trim, one adjustment function for light and darkness"
```

---

### Task 2: `messaging.LightTrimTarget`

**Files:**
- Modify: `internal/messaging/window.go` (append after `clampShift`)
- Test: `internal/messaging/window_trim_target_test.go`

- [ ] **Step 1: Write the failing test**

```go
package messaging

import "testing"

func TestLightTrimTargetSitsJustUnderTheDazzleEdge(t *testing.T) {
	cases := []struct{ strength, want int }{{0, 74}, {24, 50}, {40, 50}, {-5, 74}}
	for _, c := range cases {
		if got := LightTrimTarget(c.strength); got != float64(c.want) {
			t.Errorf("LightTrimTarget(%d) = %v, want %d", c.strength, got, c.want)
		}
		// The target itself must read as faces, never dazzled, for that observer.
		if b := BandThroughWindow(c.want, c.strength, 0, 25, 50); b != BandFaces {
			t.Errorf("strength %d: the target %d reads %v, want faces", c.strength, c.want, b)
		}
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/messaging/ -run LightTrimTarget`
Expected: FAIL, `undefined: LightTrimTarget`.

- [ ] **Step 3: Implement**

Append to `internal/messaging/window.go`:

```go
// LightTrimTarget is the brightest room light an observer with this
// night-vision strength reads without being dazzled: one point under the
// shifted dazzle edge. An adjustable light trims toward it (lighting plan 5a).
// One point, not half: a room at exactly 74.5 would round up to the edge.
func LightTrimTarget(strength int) float64 {
	return float64(windowDazzleEdge - clampShift(strength) - 1)
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/messaging/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/messaging/window.go internal/messaging/window_trim_target_test.go
git commit -m "feat(messaging): LightTrimTarget, the bearer's comfortable ceiling"
```

---

### Task 3: Light records in `conditions`

**Files:**
- Create: `internal/conditions/light.go`
- Modify: `internal/conditions/effects.go`, `internal/conditions/conditionspec.go`, `internal/conditions/conditions.go`
- Test: `internal/conditions/light_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package conditions

import (
	"math"
	"testing"

	"gopkg.in/yaml.v2"
)

const (
	testLanternId = 9701
	testGlowId    = 9702
)

func seedLightSpecs(t *testing.T) {
	t.Helper()
	t.Cleanup(SeedConditionsForTest(map[int]*ConditionSpec{
		testLanternId: {ConditionId: testLanternId, Name: "Test Lantern", TriggerCount: 1, RoundInterval: 1,
			Effects: map[EffectKind]EffectValue{EffectLightStrength: {Literal: 54}},
			Flags:   []Flag{Adjustable}},
		testGlowId: {ConditionId: testGlowId, Name: "Test Glow", TriggerCount: 4, RoundInterval: 1,
			Effects: map[EffectKind]EffectValue{EffectLightStrength: {UsesMagnitude: true}},
			Flags:   []Flag{Adjustable, Cancellable}},
	}))
}

func TestLightRecordStates(t *testing.T) {
	seedLightSpecs(t)
	bs := New()
	bs.AddCondition(testLanternId, true)
	rec := bs.LightSources()[0]
	spec := GetConditionSpec(testLanternId)

	if v, ok := rec.LightNow(spec); !ok || v != 54 {
		t.Fatalf("fresh lantern = (%v, %v), want (54, true)", v, ok)
	}
	rec.SetLightOutput(40)
	if v, ok := rec.LightNow(spec); !ok || v != 40 {
		t.Errorf("trimmed lantern = (%v, %v), want (40, true)", v, ok)
	}
	rec.SetLightOutput(math.Inf(-1))
	if _, ok := rec.LightNow(spec); ok {
		t.Error("a lantern trimmed to nothing still adds light")
	}
	rec.ResetLight()
	rec.Hooded = true
	if _, ok := rec.LightNow(spec); ok {
		t.Error("a hooded lantern still adds light")
	}
	rec.ResetLight()
	if v, ok := rec.LightNow(spec); !ok || v != 54 || rec.Hooded {
		t.Errorf("reset lantern = (%v, %v, hooded %v), want (54, true, false)", v, ok, rec.Hooded)
	}
}

// Fact 7: the worn-item refresh re-adds a held record. It must keep the trim.
func TestRefreshKeepsTrimAndHood(t *testing.T) {
	seedLightSpecs(t)
	bs := New()
	bs.AddCondition(testLanternId, true)
	rec := bs.LightSources()[0]
	rec.SetLightOutput(30)
	rec.Hooded = true
	bs.AddCondition(testLanternId, true)
	if rec.LightTrim != LightTrimmed || rec.LightOutput != 30 || !rec.Hooded {
		t.Errorf("refresh changed light state: trim %q output %v hooded %v", rec.LightTrim, rec.LightOutput, rec.Hooded)
	}
}

// A recast is a fresh source at full strength.
func TestFreshMagnitudeResetsTrim(t *testing.T) {
	seedLightSpecs(t)
	bs := New()
	bs.AddConditionMagnitude(testGlowId, 4, 90)
	rec := bs.LightSources()[0]
	rec.SetLightOutput(10)
	bs.AddConditionMagnitude(testGlowId, 4, 90)
	if v, ok := rec.LightNow(GetConditionSpec(testGlowId)); !ok || v != 90 {
		t.Errorf("recast glow = (%v, %v), want (90, true)", v, ok)
	}
}

func TestLightStateSurvivesASave(t *testing.T) {
	in := Condition{ConditionId: testLanternId, LightTrim: LightTrimmed, LightOutput: 41.5, Hooded: true}
	raw, err := yaml.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Condition
	if err := yaml.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if out.LightTrim != in.LightTrim || out.LightOutput != in.LightOutput || out.Hooded != in.Hooded {
		t.Errorf("round trip lost light state: %+v", out)
	}
}

func TestLightSpecValidation(t *testing.T) {
	adjustableNoLight := &ConditionSpec{ConditionId: 9703, Name: "Bad", TriggerCount: 1, RoundInterval: 1, Flags: []Flag{Adjustable}}
	if err := adjustableNoLight.Validate(); err == nil {
		t.Error("an adjustable condition with no light_strength validated")
	}
	zeroLight := &ConditionSpec{ConditionId: 9704, Name: "Bad", TriggerCount: 1, RoundInterval: 1,
		Effects: map[EffectKind]EffectValue{EffectLightStrength: {Literal: 0}}}
	if err := zeroLight.Validate(); err == nil {
		t.Error("a light_strength of 0 validated")
	}
}
```

- [ ] **Step 2: Run them and watch them fail**

Run: `go test ./internal/conditions/ -run "Light|Refresh|FreshMagnitude"`
Expected: FAIL to compile (`undefined: EffectLightStrength`, `Adjustable`, `LightTrimmed`...).

- [ ] **Step 3: Add the effect kind** in `internal/conditions/effects.go`

After `EffectInfraReach`:

```go
	// EffectLightStrength is a light source's full strength on the light
	// scale: a literal for an item, "magnitude" for a spell cast at a scaled
	// strength (lighting plan 5a). It is NOT aggregated through Effect():
	// every held light record is its own term in the room's combine, read
	// through Conditions.LightSources. See light.go.
	EffectLightStrength EffectKind = `light_strength`
```

Add `EffectLightStrength` to `AllEffectKinds`. At the very top of `Effect()`, before its variable declarations, add:

```go
	if kind == EffectLightStrength {
		// Per-record, never aggregated: see LightSources.
		return 0
	}
```

In `validateEffects`, after the unknown-key loop and before the tick checks, add:

```go
	if v, ok := b.Effects[EffectLightStrength]; ok && !v.UsesMagnitude && v.Literal <= 0 {
		return fmt.Errorf("conditionId %d (%s) declares light_strength %v; a light must be brighter than nothing", b.ConditionId, b.Name, v.Literal)
	}
	if slices.Contains(b.Flags, Adjustable) {
		if _, ok := b.Effects[EffectLightStrength]; !ok {
			return fmt.Errorf("conditionId %d (%s) is adjustable but declares no light_strength", b.ConditionId, b.Name)
		}
	}
```

Add `"slices"` to the imports of `effects.go`.

- [ ] **Step 4: Add the two flags** in `internal/conditions/conditionspec.go`

In the flag `const` block, after `Stacking`:

```go
	// Adjustable marks a light source that trims itself to its bearer's eyes
	// each time the bearer enters a room (lighting plan 5a). It requires the
	// light_strength effect.
	Adjustable Flag = `adjustable`
	// Cancellable marks a condition its holder may end early with
	// `cancel <spell>`. Opt-in: the Cat's Eye Draught is ruled uncancellable.
	Cancellable Flag = `cancellable`
```

Append `Adjustable,` and `Cancellable,` to `AllFlags`. Add this method after `Listed`:

```go
// IsLightSource reports whether a record of this spec sheds light.
func (b *ConditionSpec) IsLightSource() bool {
	_, ok := b.Effects[EffectLightStrength]
	return ok
}
```

- [ ] **Step 5: Add the record fields** in `internal/conditions/conditions.go`

In `Condition`, after `Stacks`:

```go
	// Light-source state (lighting plan 5a), meaningful only on a record whose
	// spec declares light_strength. See light.go. The worn-item refresh
	// re-adds a held record without touching these, so a trim survives an
	// unrelated equipment change.
	LightTrim   LightTrim `yaml:"lighttrim,omitempty"`
	LightOutput float64   `yaml:"lightoutput,omitempty"`
	Hooded      bool      `yaml:"hooded,omitempty"`
```

In `AddConditionMagnitude`, replace the trailing block

```go
	bs.List[idx].Magnitude = magnitude
	if spec := GetConditionSpec(conditionId); spec != nil && spec.TickFromMagnitude {
		// The magnitude IS the signed per-round amount; see tickAmountFor.
		bs.List[idx].TickAmount = tickAmountFor(magnitude)
	}
	return true
```

with

```go
	bs.List[idx].Magnitude = magnitude
	if spec := GetConditionSpec(conditionId); spec != nil {
		if spec.TickFromMagnitude {
			// The magnitude IS the signed per-round amount; see tickAmountFor.
			bs.List[idx].TickAmount = tickAmountFor(magnitude)
		}
		if spec.IsLightSource() {
			// A fresh magnitude is a fresh cast: full strength, hood open.
			bs.List[idx].ResetLight()
		}
	}
	return true
```

- [ ] **Step 6: Create `internal/conditions/light.go`**

```go
package conditions

import "math"

// LightTrim is where an adjustable light record's output stands. It is an
// explicit state rather than a stored -Inf, so a save never has to encode an
// infinity.
type LightTrim string

const (
	LightFull    LightTrim = ""        // untrimmed: full strength
	LightTrimmed LightTrim = "trimmed" // running at LightOutput
	LightOff     LightTrim = "off"     // trimmed to nothing: the room was bright enough
)

// LightMax is the record's full strength: the applier's magnitude for a spell
// source, the authored number for an item. 0 when spec declares no light. A
// magnitude light added with no magnitude (an admin setcondition) is 0 and
// sheds nothing; cast the spell instead.
func (b *Condition) LightMax(spec *ConditionSpec) float64 {
	if spec == nil {
		return 0
	}
	v, ok := spec.Effects[EffectLightStrength]
	if !ok {
		return 0
	}
	if v.UsesMagnitude {
		return b.Magnitude
	}
	return v.Literal
}

// LightNow is the term this record adds to its room's light right now, and
// false when it adds none: expired, hooded, trimmed to nothing, or strengthless.
func (b *Condition) LightNow(spec *ConditionSpec) (float64, bool) {
	if b.Expired() || b.Hooded {
		return 0, false
	}
	max := b.LightMax(spec)
	if max <= 0 {
		return 0, false
	}
	switch b.LightTrim {
	case LightOff:
		return 0, false
	case LightTrimmed:
		return b.LightOutput, true
	}
	return max, true
}

// SetLightOutput records a trim result from lightscale.Trim. A non-finite
// output (lightscale.Absent) means the room needs nothing from this source.
func (b *Condition) SetLightOutput(out float64) {
	if math.IsInf(out, 0) || math.IsNaN(out) {
		b.LightTrim, b.LightOutput = LightOff, 0
		return
	}
	b.LightTrim, b.LightOutput = LightTrimmed, out
}

// ResetLight returns the record to full strength with any hood open: a fresh
// cast, a fresh equip, or unhood.
func (b *Condition) ResetLight() {
	b.LightTrim, b.LightOutput, b.Hooded = LightFull, 0, false
}

// LightSources returns every held, unexpired record whose spec declares a
// light strength, in held order. The order is stable, so a bearer's several
// sources always trim in the same sequence.
func (bs *Conditions) LightSources() []*Condition {
	var out []*Condition
	for _, b := range bs.List {
		if b.Expired() {
			continue
		}
		if spec := GetConditionSpec(b.ConditionId); spec != nil && spec.IsLightSource() {
			out = append(out, b)
		}
	}
	return out
}
```

- [ ] **Step 7: Run the package tests**

Run: `go test ./internal/conditions/`
Expected: PASS, including `TestAllFlagsNamesEveryDeclaredConstant`.

- [ ] **Step 8: Commit**

```bash
git add internal/conditions/light.go internal/conditions/light_test.go internal/conditions/effects.go internal/conditions/conditionspec.go internal/conditions/conditions.go
git commit -m "feat(conditions): light_strength records with trim and hood state"
```

---

### Task 4: Compose carried light per record

Condition 1 gets `light_strength: 50` in this task, the same 50 the flat term gives today, so this task changes no shipped light. Task 7 switches it to `magnitude`.

**Files:**
- Create: `internal/characters/light.go`, `internal/characters/light_test.go`
- Modify: `internal/rooms/lighting.go`, `internal/rooms/rooms.go:1643,1739`
- Modify: `_datafiles/world/dogmud/conditions/1-illumination.yaml`, `_datafiles/world/default/conditions/1-illumination.yaml`
- Test: `internal/rooms/carried_light_test.go`

- [ ] **Step 1: Write the failing tests**

`internal/characters/light_test.go`:

```go
package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
)

func TestEmitsLightReadsLightRecords(t *testing.T) {
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		9711: {ConditionId: 9711, Name: "Test Torch", TriggerCount: 1, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {Literal: 56}}},
	}))
	c := New()
	if c.EmitsLight() {
		t.Fatal("a character with no light emits light")
	}
	if err := c.AddCondition(9711, true); err != nil {
		t.Fatal(err)
	}
	if !c.EmitsLight() {
		t.Fatal("a torch bearer does not emit light")
	}
	if got := c.LightTerms(); len(got) != 1 || got[0] != 56 {
		t.Errorf("LightTerms = %v, want [56]", got)
	}
	c.Conditions.LightSources()[0].Hooded = true
	if c.EmitsLight() {
		t.Error("a hooded source still emits light")
	}
}
```

`internal/rooms/carried_light_test.go`:

```go
package rooms

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/lightscale"
)

// Every carried source is its own term: two equal torches are one doubling
// step brighter than one, not the flat single term plan 1 shipped.
func TestCarriedSourcesEachJoinTheCombine(t *testing.T) {
	cfg := modelCfg()
	zero := 0.0
	cave := Room{SkyLight: &zero}

	one := cave.composeWith(cfg, 60, 1, []float64{56})
	two := cave.composeWith(cfg, 60, 1, []float64{56, 56})
	if one.Level != 56 {
		t.Errorf("one torch in a cave = %d, want 56", one.Level)
	}
	if two.Level != 64 {
		t.Errorf("two torches in a cave = %d, want 64 (one step of 8 brighter)", two.Level)
	}
	if !one.Carried || cave.composeWith(cfg, 60, 1, nil).Carried {
		t.Error("Carried must be true exactly when a carried term is present")
	}
	if want := lightscale.Combine(cfg.DoublingStep, 56, 56); math.Abs(two.Raw-want) > 1e-9 {
		t.Errorf("Raw = %v, want %v", two.Raw, want)
	}
	if empty := cave.composeWith(cfg, 60, 1, nil); !math.IsInf(empty.Raw, -1) || empty.Level != 0 {
		t.Errorf("an empty cave: Raw %v Level %d, want -Inf and 0", empty.Raw, empty.Level)
	}
}
```

- [ ] **Step 2: Run them and watch them fail**

Run: `go test ./internal/characters/ -run EmitsLight` and `go test ./internal/rooms/ -run CarriedSources`
Expected: FAIL, `c.EmitsLight undefined` and `cave.composeWith undefined`.

- [ ] **Step 3: Create `internal/characters/light.go`**

```go
package characters

import "github.com/GoMudEngine/GoMud/internal/conditions"

// LightTerms is every light-scale term this character adds to its room right
// now, one per held light record (lighting plan 5a).
func (c *Character) LightTerms() []float64 {
	var out []float64
	for _, rec := range c.Conditions.LightSources() {
		if v, ok := rec.LightNow(conditions.GetConditionSpec(rec.ConditionId)); ok {
			out = append(out, v)
		}
	}
	return out
}

// EmitsLight reports whether this character sheds any light right now. A
// shut hood or a source trimmed to nothing does not count. It replaces the
// retired lightsource flag.
func (c *Character) EmitsLight() bool {
	return len(c.LightTerms()) > 0
}
```

- [ ] **Step 4: Recompose `internal/rooms/lighting.go`**

Add a `Raw` field to `LightTerms`, after `Level`:

```go
	// Raw is the combined light before rounding and clamping; Absent when
	// nothing lights the room. A trim solves against it.
	Raw float64
```

Replace `composeLight` with the three functions below, and add the `conditions`, `characters`, `mobs` and `users` imports (`rooms.go` already imports all four; copy its import paths).

```go
// composeLight is the one computation behind LightLevel and LightTerms.
func (r *Room) composeLight(cfg configs.Lighting, celestial, skyFilter float64) LightTerms {
	return r.composeLightExcluding(cfg, celestial, skyFilter, nil)
}

// composeLightExcluding is composeLight with one carried record left out, which
// is the room a trimming source sees: everything except itself.
func (r *Room) composeLightExcluding(cfg configs.Lighting, celestial, skyFilter float64, exclude *conditions.Condition) LightTerms {
	return r.composeWith(cfg, celestial, skyFilter, r.carriedLight(exclude))
}

// composeWith is the composition with the carried terms supplied, so a test
// needs no users or mobs.
func (r *Room) composeWith(cfg configs.Lighting, celestial, skyFilter float64, carried []float64) LightTerms {
	step := cfg.DoublingStep
	if !(step > 0) {
		step = 1
	}

	out := LightTerms{SkyFilter: skyFilter}
	terms := make([]float64, 0, 2+len(carried))

	// 1. The sky, attenuated by this room's fraction and then by any weather
	// filtering it. A filter multiplies the fraction, which on this log scale
	// is a fixed subtraction: 0.5 removes one doubling step at any hour, so a
	// storm is merely gloomy at noon and blinding at midnight. Attenuate
	// returns Absent for a fraction of zero, so a cave contributes no term
	// rather than a term of zero.
	out.Sky = lightscale.Attenuate(step, celestial, r.skyLightFraction()*skyFilter)
	terms = append(terms, out.Sky)

	// 2. The room's own lamp.
	if lamp, ok := r.lampValue(); ok {
		out.Lamp, out.HasLamp = lamp, true
		terms = append(terms, float64(lamp))
	}

	// 3. Every light anyone here carries, each its own term (lighting plan 5a):
	// a candle and a torch are different sources, and two torches are one
	// doubling step brighter than one.
	if len(carried) > 0 {
		out.Carried = true
		terms = append(terms, carried...)
	}

	v := lightscale.Combine(step, terms...)
	out.Raw = v
	if math.IsInf(v, -1) {
		// No light of any kind. Zero is the darkest light that NATURALLY
		// occurs, which is what an unlit cave is. Magical darkness goes below
		// this and arrives in plan 5d.
		v = 0
	}

	n := int(math.Round(v))
	if n < -100 {
		n = -100
	} else if n > 100 {
		n = 100
	}
	out.Level = n
	return out
}

// carriedLight is every carried light term in the room, leaving out one record
// (the source being trimmed) when exclude is non-nil.
func (r *Room) carriedLight(exclude *conditions.Condition) []float64 {
	var terms []float64
	add := func(c *characters.Character) {
		for _, rec := range c.Conditions.LightSources() {
			if rec == exclude {
				continue
			}
			if v, ok := rec.LightNow(conditions.GetConditionSpec(rec.ConditionId)); ok {
				terms = append(terms, v)
			}
		}
	}
	for _, id := range r.GetMobs(FindHasLight) {
		if m := mobs.GetInstance(id); m != nil {
			add(&m.Character)
		}
	}
	for _, id := range r.GetPlayers(FindHasLight) {
		if u := users.GetByUserId(id); u != nil {
			add(u.Character)
		}
	}
	return terms
}
```

Update the doc comment's term 3 at the top of the file to "3. Everything anyone in the room carries, one term per light."

- [ ] **Step 5: Point `FindHasLight` at the new predicate** in `internal/rooms/rooms.go`

At line 1643 replace `mob.Character.HasFlagFromAnySource(conditions.EmitsLight)` with `mob.Character.EmitsLight()`; at line 1739 replace `user.Character.HasFlagFromAnySource(conditions.EmitsLight)` with `user.Character.EmitsLight()`.

- [ ] **Step 6: Give both condition 1 files a strength**

In `_datafiles/world/dogmud/conditions/1-illumination.yaml` and `_datafiles/world/default/conditions/1-illumination.yaml`, add after `triggercount:`:

```yaml
effects:
  light_strength: 50
```

Keep the `lightsource` flag for now; Task 5 removes it.

- [ ] **Step 7: Run the affected packages and the goldens**

Run: `go test ./internal/conditions/ ./internal/characters/ ./internal/rooms/ ./internal/lightnotice/ ./internal/actions/ ./internal/usercommands/ .`
Expected: PASS. The root package holds the two lighting goldens; fact 27 says neither moves. If one does, stop and explain the move with `python tools/lighting_golden_diff.py` before touching it.

- [ ] **Step 8: Commit**

```bash
git add internal/characters/light.go internal/characters/light_test.go internal/rooms/lighting.go internal/rooms/rooms.go internal/rooms/carried_light_test.go _datafiles/world/dogmud/conditions/1-illumination.yaml _datafiles/world/default/conditions/1-illumination.yaml
git commit -m "feat(rooms): every carried light is its own term in the combine"
```

---

### Task 5: Retire the `lightsource` flag

Delete the flag and let the compiler list what reads it.

**Files:**
- Modify: `internal/conditions/conditionspec.go`, `internal/actions/skill_helpers.go`, `internal/characters/description.go`, `internal/hooks/Awareness_LightChange.go`, `internal/hooks/NewTurn_PruneConditions.go`, `internal/usercommands/go.go`, `internal/behaviortree/sight.go`, `internal/mutations/describe.go`
- Modify tests: `internal/actions/sneak_test.go`, `internal/behaviortree/sight_test.go`, `internal/hooks/narration_testhelpers_test.go`, `internal/mutations/describe_test.go`, `internal/usercommands/look_exit_visibility_test.go`
- Modify data: both `1-illumination.yaml` files

- [ ] **Step 1: Delete the flag**

In `internal/conditions/conditionspec.go` delete the line `EmitsLight     Flag = \`lightsource\`` and the `EmitsLight,` entry in `AllFlags`.

- [ ] **Step 2: Let the compiler enumerate**

Run: `go vet ./... 2>&1 | grep EmitsLight`
Expected: errors at exactly the seven sites of fact 5 minus the two Task 4 already moved (`rooms.go`), plus the five test files. If the list differs from fact 5, stop and find out why before continuing.

- [ ] **Step 3: Migrate each reader**

- `internal/actions/skill_helpers.go:34`: `emits := c.EmitsLight()`
- `internal/characters/description.go:158`: `if c.EmitsLight() {`
- `internal/hooks/Awareness_LightChange.go:69`: `if c == nil || !c.EmitsLight() {`; update the comments at lines 24, 83, 87, 114-115 to say `EmitsLight()` rather than the flag.
- `internal/hooks/NewTurn_PruneConditions.go:137-142`: replace the loop with

```go
	if spec.IsLightSource() {
		r.SendTextVisualAsLitHidingNames(messaging.CategoryConditionExpire, msg, names, skip...)
		return
	}
```

- `internal/usercommands/go.go:819`: `if user.Character.EmitsLight() {`
- `internal/behaviortree/sight.go:23`: reword the comment to "present with a light source (Character.EmitsLight)".
- `internal/mutations/describe.go:126`: delete the `case "lightsource":` arm and its return line. This string is invisible to the compiler; grep proves it is the only one: `grep -rn '"lightsource"' --include=*.go internal modules` must print nothing afterwards.

- [ ] **Step 4: Migrate the five test files**

In each, replace a spec's `Flags: []conditions.Flag{conditions.EmitsLight}` with

```go
Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {Literal: 50}},
```

keeping any other flags in its `Flags` list. For `describe_test.go`, delete the case that expects the `lightsource` phrase. Read each test's assertion after editing: 50 is what the flat term gave, so no expected light value should change.

- [ ] **Step 5: Drop the flag from data**

In both `1-illumination.yaml` files delete `- lightsource`, and the `flags:` key if the list is now empty. Then prove nothing else ships it:

Run: `grep -rn "lightsource" _datafiles`
Expected: no output. `ValidateFlags` would reject a leftover at boot, and this grep finds it first.

- [ ] **Step 6: Run everything that touched light**

Run: `go build ./... && go test ./internal/... ./modules/... .`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/conditions/conditionspec.go internal/actions/skill_helpers.go internal/characters/description.go internal/hooks/Awareness_LightChange.go internal/hooks/NewTurn_PruneConditions.go internal/usercommands/go.go internal/behaviortree/sight.go internal/mutations/describe.go internal/actions/sneak_test.go internal/behaviortree/sight_test.go internal/hooks/narration_testhelpers_test.go internal/mutations/describe_test.go internal/usercommands/look_exit_visibility_test.go _datafiles/world/dogmud/conditions/1-illumination.yaml _datafiles/world/default/conditions/1-illumination.yaml
git commit -m "refactor(conditions): retire the lightsource flag for EmitsLight()"
```

---

### Task 6: Trim on room entry

**Files:**
- Create: `internal/rooms/light_trim.go`, `internal/rooms/light_trim_test.go`
- Modify: `internal/rooms/roommanager.go` (after `newRoom.AddPlayer(userId)`), `internal/rooms/rooms.go` (end of `AddMob`)

- [ ] **Step 1: Write the failing tests**

```go
package rooms

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const (
	trimGlowId  = 9721 // adjustable, magnitude
	trimSightId = 9722 // nightvision 24
)

func seedTrimFixture(t *testing.T) (a, b *users.UserRecord) {
	t.Helper()
	withShippedBiomesAndClock(t)
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		trimGlowId: {ConditionId: trimGlowId, Name: "Test Glow", TriggerCount: 4, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {UsesMagnitude: true}},
			Flags:   []conditions.Flag{conditions.Adjustable}},
		trimSightId: {ConditionId: trimSightId, Name: "Test Sight", TriggerCount: 4, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectNightVisionStrength: {Literal: 24}}},
	}))
	a = users.NewTestUser(7721, "glowa", "Glowa", 97721)
	b = users.NewTestUser(7722, "glowb", "Glowb", 97722)
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{7721: a, 7722: b}))
	for _, u := range []*users.UserRecord{a, b} {
		u.Character.Conditions.AddConditionMagnitude(trimGlowId, 4, 90)
	}
	b.Character.Conditions.AddCondition(trimSightId, false)
	return a, b
}

func glowOutput(u *users.UserRecord) (float64, bool) {
	rec := u.Character.Conditions.LightSources()[0]
	return rec.LightNow(conditions.GetConditionSpec(rec.ConditionId))
}

// Entry order sets the level: the first arrival trims to their own eyes, the
// second finds the room already bright enough, and nobody re-trims.
func TestEntryOrderSetsTheTrim(t *testing.T) {
	a, b := seedTrimFixture(t)
	requireBiome(t, "cave")
	room := &Room{RoomId: 7720, Biome: "cave"}

	room.AddPlayer(a.UserId)
	room.TrimLightFor(a.Character)
	if v, ok := glowOutput(a); !ok || math.Abs(v-74) > 1e-9 {
		t.Fatalf("first arrival's glow = (%v, %v), want 74, the top of normal eyes' band", v, ok)
	}
	if got := room.LightLevel(); got != 74 {
		t.Fatalf("cave after the first arrival = %d, want 74", got)
	}

	room.AddPlayer(b.UserId)
	room.TrimLightFor(b.Character)
	if _, ok := glowOutput(b); ok {
		t.Error("the nightvision arrival's glow should trim to nothing: the room is already past their comfort")
	}
	if v, _ := glowOutput(a); math.Abs(v-74) > 1e-9 {
		t.Errorf("the first arrival re-trimmed to %v when someone else entered", v)
	}
}

// In a lamplit room the trim solves the combine: lamp 50 plus the glow lands
// exactly on 74.
func TestTrimSolvesTheCombineInALitRoom(t *testing.T) {
	a, _ := seedTrimFixture(t)
	requireBiome(t, "interior")
	setClock(172, 0) // midnight: the interior's sky adds almost nothing
	room := &Room{RoomId: 7723, Biome: "interior"}
	room.AddPlayer(a.UserId)
	room.TrimLightFor(a.Character)
	if got := room.LightLevel(); got != 74 {
		t.Errorf("a lamplit interior with a trimmed glow reads %d, want 74", got)
	}
}

func TestHoodedAndNonAdjustableSourcesDoNotTrim(t *testing.T) {
	a, _ := seedTrimFixture(t)
	room := &Room{RoomId: 7724, Biome: "cave"}
	room.AddPlayer(a.UserId)
	rec := a.Character.Conditions.LightSources()[0]
	rec.Hooded = true
	room.TrimLightFor(a.Character)
	if rec.LightTrim != conditions.LightFull {
		t.Errorf("a hooded source was trimmed to %q", rec.LightTrim)
	}
}
```

Then add one seam test proving `MoveToRoom` trims. Copy the fixture shape of `internal/rooms/instances_test.go:455-480` (it seeds a user and rooms with `SeedRoomsForTest` and calls `MoveToRoom`), give the moving user the glow at 90, move them into a `cave` room, and assert their glow output is 74.

- [ ] **Step 2: Run them and watch them fail**

Run: `go test ./internal/rooms/ -run "Trim|EntryOrder"`
Expected: FAIL, `room.TrimLightFor undefined`.

- [ ] **Step 3: Create `internal/rooms/light_trim.go`**

```go
package rooms

import (
	"slices"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/lightscale"
	"github.com/GoMudEngine/GoMud/internal/messaging"
)

// TrimLightFor trims every adjustable, unhooded light record c holds to c's own
// eyes: the least cut from full strength that keeps this room from dazzling
// them. The records trim one after another in held order, each seeing the
// room as the previous trims left it.
//
// It is plan 5a's only trim trigger. MoveToRoom and AddMob call it once the
// mover is in the room, so arrivals trim in entry order, and nobody already
// here re-trims when someone else walks in. Nothing calls it on a round tick:
// a room that changes around a standing bearer leaves their light as it was.
func (r *Room) TrimLightFor(c *characters.Character) {
	if r == nil || c == nil {
		return
	}
	sources := c.Conditions.LightSources()
	if len(sources) == 0 {
		return
	}
	cfg := configs.GetLightingConfig()
	celestial := gametime.CelestialLight()
	skyFilter := r.mutatorSkyFilter()
	target := messaging.LightTrimTarget(c.NightVisionStrength())

	for _, rec := range sources {
		spec := conditions.GetConditionSpec(rec.ConditionId)
		if spec == nil || rec.Hooded || !slices.Contains(spec.Flags, conditions.Adjustable) {
			continue
		}
		others := r.composeLightExcluding(cfg, celestial, skyFilter, rec).Raw
		rec.SetLightOutput(lightscale.Trim(cfg.DoublingStep, others, rec.LightMax(spec), target, lightscale.Brightens))
	}
}
```

- [ ] **Step 4: Wire the two seams**

In `internal/rooms/roommanager.go`, directly after `roomManager.roomsWithUsers[newRoom.RoomId] = playerCt`:

```go
	// Lighting plan 5a: the arrival's adjustable lights trim to their eyes
	// now, before anything reads the room, so arrivals trim in entry order.
	newRoom.TrimLightFor(user.Character)
```

In `internal/rooms/rooms.go` `AddMob`, directly after `r.mobs = append(r.mobs, mobInstanceId)`:

```go
	// Lighting plan 5a: see MoveToRoom.
	r.TrimLightFor(&mob.Character)
```

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/rooms/ ./internal/usercommands/ ./internal/hooks/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/rooms/light_trim.go internal/rooms/light_trim_test.go internal/rooms/roommanager.go internal/rooms/rooms.go
git commit -m "feat(rooms): adjustable lights trim to their bearer on room entry"
```

---

### Task 7: Glow strength and duration from stat and skill

**Files:**
- Modify: `internal/configs/config.balance.go` (after `LightMoonWeightEye`), `internal/configs/config.balance.lighting.go` (`validateLighting`), `internal/configs/config.lighting_accessor.go`
- Modify: `_datafiles/config.yaml` (after `DarknessShapesCombatPenalty`, line 928)
- Create: `internal/hooks/light_spell.go`, `internal/hooks/light_spell_test.go`
- Modify: `internal/hooks/spell_resolution.go:783,1137,1502,1763`, `internal/mobs/mobs.go` (after `AddCondition`)
- Modify: `_datafiles/world/dogmud/conditions/1-illumination.yaml`

- [ ] **Step 1: Write the failing test**

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

func TestLightSpellScalesFromStatAndSkill(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.Balance.Validate()
	configs.SetConfigForTest(t, cfg)
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		9731: {ConditionId: 9731, Name: "Test Glow", TriggerCount: 4, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {UsesMagnitude: true}},
			Flags:   []conditions.Flag{conditions.Adjustable}},
		9732: {ConditionId: 9732, Name: "Test Plain", TriggerCount: 4, RoundInterval: 1},
	}))
	spell := &spells.SpellData{SpellId: "test-glow", PrimaryStat: "willpower"}

	cases := []struct {
		stat, skill  int
		wantStrength float64
		wantTriggers int
	}{
		{100, 0, 50, 4},  // a new character: today's 20 real minutes
		{175, 65, 90, 9}, // endgame: about 45 minutes
	}
	for _, c := range cases {
		caster := characters.New()
		caster.Stats.Willpower.ValueAdj = c.stat
		caster.SetSkill("spellcasting", c.skill)
		mag, trig, ok := lightSpellApplication(spell, caster, 9731)
		if !ok || mag != c.wantStrength || trig != c.wantTriggers {
			t.Errorf("stat %d skill %d: (%v, %d, %v), want (%v, %d, true)", c.stat, c.skill, mag, trig, ok, c.wantStrength, c.wantTriggers)
		}
	}
	if _, _, ok := lightSpellApplication(spell, characters.New(), 9732); ok {
		t.Error("a non-light condition was treated as a light spell")
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/hooks/ -run LightSpellScales`
Expected: FAIL, `undefined: lightSpellApplication`.

- [ ] **Step 3: Declare the six knobs** in `internal/configs/config.balance.go`, after `LightMoonWeightEye`:

```go
	// Light-spell scaling (lighting plan 5a). A spell whose condition declares
	// light_strength: magnitude is cast at
	//   LightSpellStrengthBase + stat/LightSpellStrengthStatDivisor + spellcasting/LightSpellStrengthSkillDivisor
	// on the light scale, for
	//   LightSpellDurationBase + stat/LightSpellDurationStatDivisor + spellcasting/LightSpellDurationSkillDivisor
	// triggers (rounded, at least 1) of the condition's own trigger rate. stat is
	// the spell's primarystat ValueAdj; spellcasting runs 0 to 100. Shipped: a
	// new character (100, 0) glows at 50 for 4 triggers, 20 real minutes; an
	// endgame caster (175, 65) at 90 for 9, 45 minutes. The house idiom is
	// unarmed damage's base + stat/D1 + skill/D2. The spell scaling
	// unification arc should absorb these rather than keep glow an exception.
	LightSpellStrengthBase         ConfigFloat `yaml:"LightSpellStrengthBase"`         // default 40
	LightSpellStrengthStatDivisor  ConfigFloat `yaml:"LightSpellStrengthStatDivisor"`  // default 10
	LightSpellStrengthSkillDivisor ConfigFloat `yaml:"LightSpellStrengthSkillDivisor"` // default 2
	LightSpellDurationBase         ConfigFloat `yaml:"LightSpellDurationBase"`         // default 2
	LightSpellDurationStatDivisor  ConfigFloat `yaml:"LightSpellDurationStatDivisor"`  // default 50
	LightSpellDurationSkillDivisor ConfigFloat `yaml:"LightSpellDurationSkillDivisor"` // default 20
```

- [ ] **Step 4: Default them** at the end of `validateLighting` in `internal/configs/config.balance.lighting.go`:

```go
	// Light-spell scaling. Zero or negative is coerced: a zero divisor divides
	// by zero, and a test binary never loads config.yaml, so zero must mean
	// "unset" for the two bases too.
	for _, k := range []struct {
		v   *ConfigFloat
		def ConfigFloat
	}{
		{&b.LightSpellStrengthBase, 40}, {&b.LightSpellStrengthStatDivisor, 10}, {&b.LightSpellStrengthSkillDivisor, 2},
		{&b.LightSpellDurationBase, 2}, {&b.LightSpellDurationStatDivisor, 50}, {&b.LightSpellDurationSkillDivisor, 20},
	} {
		if !(*k.v > 0) {
			*k.v = k.def
		}
	}
```

- [ ] **Step 5: Expose them** in `internal/configs/config.lighting_accessor.go`. Add to `Lighting`:

```go
	SpellStrengthBase, SpellStrengthStatDivisor, SpellStrengthSkillDivisor float64
	SpellDurationBase, SpellDurationStatDivisor, SpellDurationSkillDivisor float64
```

and to the returned struct literal:

```go
		SpellStrengthBase:         float64(b.LightSpellStrengthBase),
		SpellStrengthStatDivisor:  float64(b.LightSpellStrengthStatDivisor),
		SpellStrengthSkillDivisor: float64(b.LightSpellStrengthSkillDivisor),
		SpellDurationBase:         float64(b.LightSpellDurationBase),
		SpellDurationStatDivisor:  float64(b.LightSpellDurationStatDivisor),
		SpellDurationSkillDivisor: float64(b.LightSpellDurationSkillDivisor),
```

- [ ] **Step 6: Create `internal/hooks/light_spell.go`**

```go
package hooks

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// lightSpellApplication reports how a condition from a spell should be applied
// when it is a magnitude-driven light: at a strength and a duration scaled
// from the CASTER's primary stat and spellcasting skill (lighting plan 5a).
// ok is false for any other condition, which keeps its authored application.
// The record then trims to its HOLDER's eyes, who may not be the caster.
func lightSpellApplication(spellData *spells.SpellData, caster *characters.Character, conditionId int) (magnitude float64, triggers int, ok bool) {
	if spellData == nil || caster == nil {
		return 0, 0, false
	}
	spec := conditions.GetConditionSpec(conditionId)
	if spec == nil {
		return 0, 0, false
	}
	if v, isLight := spec.Effects[conditions.EffectLightStrength]; !isLight || !v.UsesMagnitude {
		return 0, 0, false
	}
	cfg := configs.GetLightingConfig()
	stat := float64(spellData.CasterStatValue(caster.Stats))
	skill := float64(caster.GetSkillLevel(skills.Spellcasting))
	magnitude = cfg.SpellStrengthBase + stat/cfg.SpellStrengthStatDivisor + skill/cfg.SpellStrengthSkillDivisor
	triggers = int(math.Round(cfg.SpellDurationBase + stat/cfg.SpellDurationStatDivisor + skill/cfg.SpellDurationSkillDivisor))
	if triggers < 1 {
		triggers = 1
	}
	return magnitude, triggers, true
}
```

- [ ] **Step 7: Add `Mob.AddConditionMagnitude`** in `internal/mobs/mobs.go`, after `AddCondition`:

```go
// AddConditionMagnitude queues a record with an exact trigger count and a
// per-instance magnitude through the event path, the mob twin of
// UserRecord.AddConditionMagnitude.
func (m *Mob) AddConditionMagnitude(conditionId int, triggers int, magnitude float64, source string) {
	events.AddToQueue(events.Condition{
		MobInstanceId: m.InstanceId,
		ConditionId:   conditionId,
		Source:        source,
		Triggers:      triggers,
		Magnitude:     magnitude,
		LifeEpoch:     m.Character.LifeEpoch,
	})
}
```

- [ ] **Step 8: Route the four spell sites**

Each `X.AddCondition(conditionId, "spell")` in `internal/hooks/spell_resolution.go` becomes the pattern below, with the target and caster of that site (fact 14):

| Line | Target | Caster |
|---|---|---|
| 783 | `mob` | `casterChar` |
| 1137 | `target` | `user.Character` |
| 1502 | `mob` | `&mob.Character` |
| 1763 | `target` | `&caster.Character` |

```go
		if mag, trig, ok := lightSpellApplication(spellData, CASTER, conditionId); ok {
			TARGET.AddConditionMagnitude(conditionId, trig, mag, "spell")
		} else {
			TARGET.AddCondition(conditionId, "spell")
		}
```

At line 783 `casterChar` may be nil for some callers; `lightSpellApplication` returns `ok == false` for a nil caster, which keeps today's application.

- [ ] **Step 9: Make condition 1 a scaled, adjustable, cancellable light**

`_datafiles/world/dogmud/conditions/1-illumination.yaml` becomes:

```yaml
conditionid: 1
name: Illumination
description: Light surrounds you.
secret: false
triggerrate: 5 real minutes
triggercount: 4
effects:
  light_strength: magnitude
flags:
  - adjustable
  - cancellable
start_actee: "A warm glow surrounds you."
start_observer: "A warm glow surrounds {actee_plain}."
end_actee: "Your glow fades away."
end_observer: "The glow surrounding {actee_plain} fades away."
```

Leave the default world's condition 1 at `light_strength: 50`: nothing casts it there and tests read it as a fixed light.

- [ ] **Step 10: Surface the knobs in `config.yaml`**

In the worktree (plain `HEAD` blob), after line 928 `DarknessShapesCombatPenalty: 0.90`:

```yaml

  # ── LIGHT: SPELL SCALING (lighting plan 5a) ─────────────────────────────────
  # A light spell (condition with light_strength: magnitude) is cast at
  #   base + stat/StatDivisor + spellcasting/SkillDivisor
  # on the light scale, and lasts the same shape in triggers of its condition's
  # trigger rate. stat is the spell's primary stat; spellcasting runs 0-100.
  # A new character (100, 0) glows at 50 (faces in a cave) for 20 minutes; an
  # endgame caster (175, 65) at 90 (dazzles anyone) for about 45. A fresh cast
  # runs at full strength and trims to the holder's eyes when they next move.
  LightSpellStrengthBase: 40
  LightSpellStrengthStatDivisor: 10
  LightSpellStrengthSkillDivisor: 2
  LightSpellDurationBase: 2
  LightSpellDurationStatDivisor: 50
  LightSpellDurationSkillDivisor: 20
```

- [ ] **Step 11: Run the tests**

Run: `go build ./... && go test ./internal/configs/ ./internal/hooks/ ./internal/mobs/ ./internal/conditions/`
Expected: PASS. If a config test checks that every knob has a `config.yaml` line or a doc line, satisfy it with the lines above.

- [ ] **Step 12: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.lighting.go internal/configs/config.lighting_accessor.go internal/hooks/light_spell.go internal/hooks/light_spell_test.go internal/hooks/spell_resolution.go internal/mobs/mobs.go _datafiles/world/dogmud/conditions/1-illumination.yaml _datafiles/config.yaml
git commit -m "feat(spells): glow strength and duration scale from willpower and spellcasting"
```

---

### Task 8: The `light` equipment slot

**Files:** every site in fact 18, plus `internal/characters/worn.go` and a new test `internal/characters/light_slot_test.go`.

- [ ] **Step 1: Write the failing test**

```go
package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/items"
)

const (
	slotLanternItem = 999940
	slotGlovesItem  = 999941
	slotLanternCond = 9741
)

func seedLightSlot(t *testing.T) *Character {
	t.Helper()
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		slotLanternCond: {ConditionId: slotLanternCond, Name: "Test Lantern", Secret: true, TriggerCount: 1, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {Literal: 54}},
			Flags:   []conditions.Flag{conditions.Adjustable}},
	}))
	t.Cleanup(items.SeedItemsForTest(map[int]*items.ItemSpec{
		slotLanternItem: {ItemId: slotLanternItem, Name: "test hooded lantern", Type: items.Light, Subtype: items.Wearable,
			WornConditionIds: []int{slotLanternCond}},
		slotGlovesItem: {ItemId: slotGlovesItem, Name: "test gloves", Type: items.Gloves, Subtype: items.Wearable},
	}))
	c := New()
	c.Stats.Strength.ValueAdj = 100
	return c
}

func TestLightSlotWearsALight(t *testing.T) {
	c := seedLightSlot(t)
	if _, ok, why := c.Wear(items.New(slotLanternItem)); !ok {
		t.Fatalf("could not wear a light: %s", why)
	}
	if c.Equipment.Light.ItemId != slotLanternItem {
		t.Fatalf("light slot holds %d", c.Equipment.Light.ItemId)
	}
	if !c.EmitsLight() {
		t.Error("equipping a light did not turn it on")
	}
}

// Swapping gloves must not reset a lantern still worn (fact 7).
func TestAnotherSlotKeepsTheTrim(t *testing.T) {
	c := seedLightSlot(t)
	c.Wear(items.New(slotLanternItem))
	rec := c.Conditions.LightSources()[0]
	rec.SetLightOutput(30)
	rec.Hooded = true
	c.Wear(items.New(slotGlovesItem))
	if rec.LightTrim != conditions.LightTrimmed || !rec.Hooded {
		t.Errorf("wearing gloves reset the lantern: trim %q hooded %v", rec.LightTrim, rec.Hooded)
	}
}

// Re-equipping the light is its fresh start, even when the replaced item
// shared its condition and the refresh kept the record.
func TestReEquippingTheLightResetsIt(t *testing.T) {
	c := seedLightSlot(t)
	c.Wear(items.New(slotLanternItem))
	rec := c.Conditions.LightSources()[0]
	rec.SetLightOutput(30)
	rec.Hooded = true
	c.Wear(items.New(slotLanternItem))
	now := c.Conditions.LightSources()[0]
	if now.LightTrim != conditions.LightFull || now.Hooded {
		t.Errorf("re-equipped lantern: trim %q hooded %v, want full and open", now.LightTrim, now.Hooded)
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/characters/ -run "LightSlot|AnotherSlot|ReEquipping"`
Expected: FAIL, `undefined: items.Light`.

- [ ] **Step 3: The item type** in `internal/items/itemspec.go`: after `Tail`, add

```go
	Light        ItemType = "light" // A carried light: candle, lantern, torch (lighting plan 5a)
```

and in `ItemTypes()` after the `Tail` row:

```go
		{string(Light), `A light you carry to see by.`, 0, 20000, 29999},
```

- [ ] **Step 4: The slot** in `internal/characters/worn.go`

- `Worn`: after `ComponentBag`, `Light items.Item \`yaml:"light,omitempty"\` // Carried light (lighting plan 5a)`
- `AllSlots()`: append `{"light", "Light", &w.Light},` after the component bag entry.
- `StatMod`: change `w.ComponentBag.StatMod(stat...)` to `w.ComponentBag.StatMod(stat...) +` and add `w.Light.StatMod(stat...)`.
- The `"worn - …"` pointer switch: add `case "worn - light": return &w.Light` after `"worn - componentbag"`.
- `GetAllSlotTypes()`: add `string(items.Light),` after `ComponentBag`.
- The wear switch, before `default:`:

```go
	case items.Light:
		returnItems = append(returnItems, c.Equipment.Light)
		c.Equipment.Light = i
```

- The unequip chain: before the final `} else {` add

```go
	} else if i.Equals(c.Equipment.Light) {
		c.Equipment.Light = items.Item{}
```

- In `Wear`, directly after the `c.reapplyPermanentConditions(returnItems...)` block:

```go
	if spec.Type == items.Light {
		// Equipping a light is its fresh start (lighting plan 5a): full
		// strength, hood open, even when the item it replaced shared its
		// condition and the refresh above kept that record.
		for _, id := range spec.WornConditionIds {
			for _, rec := range c.Conditions.GetConditions(id) {
				rec.ResetLight()
			}
		}
	}
```

- [ ] **Step 5: The other slot lists** (fact 18)

- `internal/characters/combat.go:168,213,252`: add `c.Equipment.Light,` after `c.Equipment.ComponentBag,`.
- `internal/characters/inventory.go:520`: add `{c.Equipment.Light, "worn - light"},` after the component bag row.
- `internal/itemvalue/types.go`: `SlotLight SlotName = "Light"` after `SlotComponentBag`.
- `internal/itemvalue/delta.go`: `SlotLight` after `SlotComponentBag` in the slot list (line 31); `case items.Light: return []SlotName{SlotLight}` in `compatibleSlotsFor`; `case SlotLight: return e.Light` in the getter; `{SlotLight, e.Light},` in the pairs list.
- `internal/hooks/PlayerSpawn_HandleJoin.go:194`: append `|| e.Light.ItemId != 0`.
- `internal/goals/catalog/mastery_equip.go:158` and `internal/planners/mastery_equip.go:147`: `case "light": return "worn - light"`.
- `internal/usercommands/enchant_slot.go:150`: `case "light": scanOrder = []slotEntry{{"worn - light", eq.Light}}`.
- `internal/caravan/visit.go:335`: add `items.Light,` to the equipment case list.
- `modules/gmcp/gmcp.Mob.go:812`: update the slot-count comment.

- [ ] **Step 6: Run the tests, including the slot guard**

Run: `go build ./... && go test ./internal/characters/ ./internal/itemvalue/ ./internal/goals/... ./internal/planners/ ./internal/usercommands/ ./internal/hooks/ ./internal/caravan/ ./modules/gmcp/`
Expected: PASS, including `worn_allslots_test.go`, which fails if a `Worn` field is missing from `AllSlots`.

- [ ] **Step 7: Commit**

```bash
git add internal/items/itemspec.go internal/characters/worn.go internal/characters/combat.go internal/characters/inventory.go internal/characters/light_slot_test.go internal/itemvalue/types.go internal/itemvalue/delta.go internal/hooks/PlayerSpawn_HandleJoin.go internal/goals/catalog/mastery_equip.go internal/planners/mastery_equip.go internal/usercommands/enchant_slot.go internal/caravan/visit.go modules/gmcp/gmcp.Mob.go
git commit -m "feat(items): a light equipment slot for every character"
```

---

### Task 9: The light slot in the web client

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html:3315`, `_datafiles/html/public/static/js/item-icon-map.js:177,210`

- [ ] **Step 1: Make light items equippable from the inventory menu**

In `EQUIPPABLE_TYPES` add `light: true,` after `componentbag: true` (add the comma that entry now needs).

- [ ] **Step 2: Give light items an icon**

In `item-icon-map.js`, add `"light": 1` to `EQUIPMENT_TYPES` and `"light": "oil_lantern",` to `TYPE_MAP` after `"componentbag"`. `oil_lantern.png` already ships (`static/images/items/`).

- [ ] **Step 3: Check it by hand**

Boot the worktree's server on a spare port per `dogmud-shipping`, log in on the web client, equip an Oil Lantern (after Task 13, or with admin item spawn), and confirm the Worn tab shows a "Light" row with the lantern icon. Stop only the server you started, by its PID.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/html/public/webclient-pure.html _datafiles/html/public/static/js/item-icon-map.js
git commit -m "feat(webclient): light slot items are equippable and have an icon"
```

---

### Task 10: Item nouns

**Files:**
- Modify: `internal/items/itemspec.go` (`ItemSpec`), `internal/usercommands/look.go:306`
- Create: `internal/characters/itemnouns.go`, `internal/characters/itemnouns_test.go`

- [ ] **Step 1: Write the failing test**

```go
package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestFindItemNounMatchesWornAndCarried(t *testing.T) {
	t.Cleanup(items.SeedItemsForTest(map[int]*items.ItemSpec{
		999950: {ItemId: 999950, Name: "test hooded lantern", Type: items.Light, Subtype: items.Wearable,
			Description: "A lantern with a hood.", Nouns: map[string]string{"hood": "Close it with hood, open it with unhood."}},
	}))
	c := New()
	c.Equipment.Light = items.New(999950)
	noun, desc, ok := c.FindItemNoun("hood")
	if !ok || noun != "hood" || desc != "Close it with hood, open it with unhood." {
		t.Errorf("FindItemNoun(hood) = (%q, %q, %v)", noun, desc, ok)
	}
	if _, _, ok := c.FindItemNoun("hoo"); ok {
		t.Error("an item noun matched a prefix; it must match exactly")
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/characters/ -run FindItemNoun`
Expected: FAIL, `unknown field Nouns`.

- [ ] **Step 3: Add the field** to `ItemSpec` in `internal/items/itemspec.go`, after `WornConditionIds`:

```go
	// Nouns are details of the item a player can look at by name, highlighted
	// in the item's description the way a room's nouns are (lighting plan 5a:
	// the hooded lantern's hood).
	Nouns map[string]string `yaml:"nouns,omitempty"`
```

- [ ] **Step 4: Create `internal/characters/itemnouns.go`**

```go
package characters

import "strings"

// FindItemNoun looks for an exact noun on anything this character wears or
// carries, worn items first. Exact only: look runs it before item matching,
// and a prefix would let "hood" shadow an item named "hooded lantern".
func (c *Character) FindItemNoun(word string) (noun, desc string, ok bool) {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return "", "", false
	}
	candidates := c.GetAllWornItems()
	candidates = append(candidates, c.Items...)
	for _, itm := range candidates {
		spec := itm.GetSpec() // a value, never nil
		for n, d := range spec.Nouns {
			if strings.ToLower(n) == word {
				return n, d, true
			}
		}
	}
	return "", "", false
}
```

- [ ] **Step 5: Use it in `look`**

In `internal/usercommands/look.go`, directly before `lookItem, lookDestination, foundItem := user.Character.FindItem(lookAt)`:

```go
	//
	// A noun on something you wear or carry (lighting plan 5a). Exact match,
	// and before item matching, so "hood" reaches the lantern's hood rather
	// than the lantern.
	//
	if noun, desc, ok := user.Character.FindItemNoun(lookAt); ok {
		user.SendText(messaging.CategoryRoomDescription, ``)
		user.SendText(messaging.CategoryRoomDescription, fmt.Sprintf(`You look at the <ansi fg="noun">%s</ansi>:`, noun))
		user.SendText(messaging.CategoryRoomDescription, ``)
		user.SendText(messaging.CategoryRoomDescription, util.SplitStringNL(desc, 80))
		user.SendText(messaging.CategoryRoomDescription, ``)
		return true, nil
	}
```

In the item branch, highlight the item's own nouns. Replace

```go
		user.SendText(messaging.CategoryRoomDescription,
			util.SplitStringNL(lookItem.GetLongDescription(), 80),
		)
```

with

```go
		itemDesc := util.SplitStringNL(lookItem.GetLongDescription(), 80)
		for noun := range lookItem.GetSpec().Nouns {
			itemDesc = strings.Replace(itemDesc, noun, `<ansi fg="noun">`+noun+`</ansi>`, 1)
		}
		user.SendText(messaging.CategoryRoomDescription, itemDesc)
```

- [ ] **Step 6: Run the tests**

Run: `go test ./internal/characters/ ./internal/usercommands/ ./internal/items/`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/items/itemspec.go internal/characters/itemnouns.go internal/characters/itemnouns_test.go internal/usercommands/look.go
git commit -m "feat(items): item nouns you can look at, highlighted like room nouns"
```

---

### Task 11: `hood` and `unhood`

**Files:**
- Create: `internal/usercommands/hood.go`, `internal/usercommands/hood_test.go`, `_datafiles/world/dogmud/templates/help/hood.template`
- Modify: `internal/usercommands/usercommands.go` (registry), `internal/usercommands/helpfile_completeness_test.go` (`commandHelpAliases`), `_datafiles/world/dogmud/keywords.yaml` (`items:` list)

- [ ] **Step 1: Write the failing test**

```go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

func TestHoodAndUnhood(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		9751: {ConditionId: 9751, Name: "Test Hooded Lantern", Secret: true, TriggerCount: 1, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {Literal: 54}},
			Flags:   []conditions.Flag{conditions.Adjustable}},
	})()
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999960: {ItemId: 999960, Name: "test hooded lantern", Type: items.Light, Subtype: items.Wearable, WornConditionIds: []int{9751}},
	})()

	user := users.GetByUserId(1)
	require.NotNil(t, user)
	user.Character.Stats.Strength.ValueAdj = 100
	_, ok, why := user.Character.Wear(items.New(999960))
	require.True(t, ok, why)
	rec := user.Character.Conditions.LightSources()[0]
	rec.SetLightOutput(30)

	_, err := Hood("", user, nil, 0)
	require.NoError(t, err)
	require.True(t, rec.Hooded, "hood did not close the hood")
	require.False(t, user.Character.EmitsLight(), "a hooded lantern still sheds light")

	_, err = Unhood("", user, nil, 0)
	require.NoError(t, err)
	require.False(t, rec.Hooded)
	require.Equal(t, conditions.LightFull, rec.LightTrim, "unhood must return the lantern to full strength")
}
```

If `Hood` with a nil room panics on the observer line, pass the room from `rooms.LoadRoom(user.Character.RoomId)` instead; the command must tolerate a nil room anyway.

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/usercommands/ -run HoodAndUnhood`
Expected: FAIL, `undefined: Hood`.

- [ ] **Step 3: Create `internal/usercommands/hood.go`**

```go
package usercommands

import (
	"fmt"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// hoodedLight returns the adjustable light records of the item in the user's
// light slot: the hooded lantern's (lighting plan 5a).
func hoodedLight(user *users.UserRecord) []*conditions.Condition {
	lightItem := user.Character.Equipment.Light
	if lightItem.ItemId < 1 {
		return nil
	}
	var recs []*conditions.Condition
	for _, id := range lightItem.GetSpec().WornConditionIds {
		spec := conditions.GetConditionSpec(id)
		if spec == nil || !spec.IsLightSource() || !slices.Contains(spec.Flags, conditions.Adjustable) {
			continue
		}
		recs = append(recs, user.Character.Conditions.GetConditions(id)...)
	}
	return recs
}

// Hood closes the hood of the lantern in the light slot: it stays lit and
// held, and sheds no light until unhooded or re-equipped.
func Hood(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	recs := hoodedLight(user)
	if len(recs) == 0 {
		user.SendText(messaging.CategorySystem, `You have no hooded light to close.`)
		return true, nil
	}
	if recs[0].Hooded {
		user.SendText(messaging.CategorySystem, `Your lantern is already hooded.`)
		return true, nil
	}
	for _, rec := range recs {
		rec.Hooded = true
	}
	user.SendText(messaging.CategorySystem, `You lower the hood over your lantern, and its light narrows to nothing.`)
	if room != nil {
		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> lowers the hood of a lantern, and the light around them dies away.`, user.Character.Name),
			user.UserId)
	}
	return true, nil
}

// Unhood opens the hood at full strength. The next room the bearer enters
// trims it back to their eyes.
func Unhood(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	recs := hoodedLight(user)
	if len(recs) == 0 {
		user.SendText(messaging.CategorySystem, `You have no hooded light to open.`)
		return true, nil
	}
	if !recs[0].Hooded {
		user.SendText(messaging.CategorySystem, `Your lantern's hood is already open.`)
		return true, nil
	}
	for _, rec := range recs {
		rec.ResetLight()
	}
	user.SendText(messaging.CategorySystem, `You throw back the hood of your lantern, and light floods out around you.`)
	if room != nil {
		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> throws back the hood of a lantern, and light floods the area.`, user.Character.Name),
			user.UserId)
	}
	return true, nil
}
```

- [ ] **Step 4: Register both commands**

In `internal/usercommands/usercommands.go`'s command table, in alphabetical position:

```go
		`hood`:            {Hood, false, true, false},
		`unhood`:          {Unhood, false, true, false},
```

In `helpfile_completeness_test.go`'s `commandHelpAliases`, add `"unhood": "hood",`. In `_datafiles/world/dogmud/keywords.yaml`, add `- hood` to the `items:` command list.

- [ ] **Step 5: Write `_datafiles/world/dogmud/templates/help/hood.template`**

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">hood</ansi>

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">hood</ansi>     Close the hood of the lantern in your light slot.
  <ansi fg="command">unhood</ansi>   Open it again.

A hooded lantern trims its light to your eyes every time you walk
into a new place, so it never dazzles you there.

Closing the hood shuts its light off without putting it out. You
carry it dark, and you are harder to spot.

Opening the hood throws its light out at full strength until you
next move, which can dazzle anyone nearby whose eyes are used to
the dark.

Look at your lantern to find its hood.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help light</ansi>, <ansi fg="command">help equipment</ansi>
```

- [ ] **Step 6: Run the tests**

Run: `go test ./internal/usercommands/`
Expected: PASS, including `helpfile_completeness_test.go` and any message-literal guard. If a guard rejects the new literals, follow its own instructions for registering a command message.

- [ ] **Step 7: Commit**

```bash
git add internal/usercommands/hood.go internal/usercommands/hood_test.go internal/usercommands/usercommands.go internal/usercommands/helpfile_completeness_test.go _datafiles/world/dogmud/keywords.yaml _datafiles/world/dogmud/templates/help/hood.template
git commit -m "feat(commands): hood and unhood the lantern in your light slot"
```

---

### Task 12: `cancel <spell>`

**Files:**
- Modify: `internal/usercommands/cancel.go`, `_datafiles/world/dogmud/templates/help/cancel.template`
- Test: `internal/usercommands/cancel_spell_test.go`

- [ ] **Step 1: Write the failing test**

```go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

func TestCancelEndsACancellableSpell(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		9761: {ConditionId: 9761, Name: "Test Radiance", TriggerCount: 4, RoundInterval: 1,
			Flags: []conditions.Flag{conditions.Cancellable}},
		9762: {ConditionId: 9762, Name: "Test Draught", TriggerCount: 4, RoundInterval: 1},
	})()
	user := users.GetByUserId(1)
	require.NotNil(t, user)
	require.NoError(t, user.Character.AddCondition(9761, false))
	require.NoError(t, user.Character.AddCondition(9762, false))

	_, err := Cancel("test draught", user, nil, 0)
	require.NoError(t, err)
	require.True(t, user.Character.HasCondition(9762), "a condition without the cancellable flag was cancelled")

	_, err = Cancel("test radiance", user, nil, 0)
	require.NoError(t, err)
	require.False(t, user.Character.HasCondition(9761), "cancel did not end a cancellable condition")
}
```

`HasCondition` reads unexpired records only; confirm that in `internal/conditions/conditions.go:250` and use `Conditions.GetConditions(9761)` being empty instead if it does not.

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/usercommands/ -run CancelEnds`
Expected: FAIL, the radiance is still held.

- [ ] **Step 3: Implement** at the top of `Cancel` in `internal/usercommands/cancel.go`, before the activity check:

```go
	if rest = strings.TrimSpace(rest); rest != `` {
		return cancelCondition(rest, user)
	}
```

and add below `Cancel`:

```go
// cancelCondition ends a cancellable condition the user holds, named by the
// spell that grants it (id, alias or name, via spells.ResolveSpell) or by the
// condition's own name. Its end narration arrives with the next prune.
func cancelCondition(name string, user *users.UserRecord) (bool, error) {
	var ids []int
	if sd := spells.ResolveSpell(name); sd != nil {
		ids = append(ids, sd.ConditionIds...)
	}
	lower := strings.ToLower(name)
	for _, rec := range user.Character.Conditions.GetConditions() {
		if spec := conditions.GetConditionSpec(rec.ConditionId); spec != nil && strings.HasPrefix(strings.ToLower(spec.Name), lower) {
			ids = append(ids, rec.ConditionId)
		}
	}
	for _, id := range ids {
		spec := conditions.GetConditionSpec(id)
		if spec == nil || !slices.Contains(spec.Flags, conditions.Cancellable) || !user.Character.HasCondition(id) {
			continue
		}
		user.Character.RemoveCondition(id)
		return true, nil
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`You have no %s you can let go of.`, name))
	return true, nil
}
```

Add imports `fmt`, `slices`, `strings`, `conditions`, `spells`.

- [ ] **Step 4: Update the help** in `cancel.template`: add a usage line and a paragraph.

```
  <ansi fg="command">cancel <spell></ansi>      Let go of a spell you are holding, such as
                      <ansi fg="command">cancel glow</ansi>.

Some spells can be let go of before they run out. Chrysalis Glow is
one: <ansi fg="command">cancel glow</ansi> puts it out at once.
```

Extend its "See also" with `<ansi fg="command">help chrysalis-glow</ansi>`.

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/usercommands/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/cancel.go internal/usercommands/cancel_spell_test.go _datafiles/world/dogmud/templates/help/cancel.template
git commit -m "feat(commands): cancel <spell> ends a cancellable condition"
```

---

### Task 13: The four light items

**Files:**
- Create: four condition files and two item files (ids below)
- Modify: `items/materials-40000/40038-oil_lantern.yaml`, `items/materials-40000/40077-tallow_candle.yaml`, shop blocks in the merchant mob files
- Test: `internal/items/shipped_light_items_test.go`

Load `dogmud-authoring-content` first: it covers the filename-must-match-name panic and the unquoted-colon trap.

- [ ] **Step 1: Confirm the ids are still free**

Run: `python tools/id_inventory.py --type conditions` and `python tools/id_inventory.py --type items`
Expected: conditions 124-127 free; items 20096 and 20097 free. If any is taken, use the next free id and carry it through every step below.

- [ ] **Step 2: Write the shipped-data test first**

```go
package items_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// The 5a ladder, read from the shipped world: every carried light has its
// strength, lives in the light slot, and is secret so a light shows at most
// once in the conditions list (owner, 2026-09-26).
func TestShippedLightItemsMatchTheLadder(t *testing.T) {
	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = `../../_datafiles/world/dogmud`
	configs.SetConfigForTest(t, cfg)
	conditions.LoadDataFiles()
	items.LoadDataFiles()

	cases := []struct {
		itemId     int
		strength   float64
		adjustable bool
	}{
		{40077, 38, false}, // Tallow Candle
		{40038, 52, false}, // Oil Lantern
		{20096, 56, false}, // Torch
		{20097, 54, true},  // Hooded Lantern
	}
	for _, c := range cases {
		spec := items.GetItemSpec(c.itemId)
		if spec == nil {
			t.Fatalf("item %d is not shipped", c.itemId)
		}
		if spec.Type != items.Light || spec.Subtype != items.Wearable {
			t.Errorf("item %d is %s/%s, want light/wearable", c.itemId, spec.Type, spec.Subtype)
		}
		if len(spec.WornConditionIds) != 1 {
			t.Fatalf("item %d grants %d conditions, want 1", c.itemId, len(spec.WornConditionIds))
		}
		cond := conditions.GetConditionSpec(spec.WornConditionIds[0])
		if cond == nil || !cond.Secret {
			t.Errorf("item %d's light condition must exist and be secret", c.itemId)
			continue
		}
		if v := cond.Effects[conditions.EffectLightStrength]; v.UsesMagnitude || v.Literal != c.strength {
			t.Errorf("item %d shines at %+v, want %v", c.itemId, v, c.strength)
		}
		adjustable := false
		for _, f := range cond.Flags {
			adjustable = adjustable || f == conditions.Adjustable
		}
		if adjustable != c.adjustable {
			t.Errorf("item %d adjustable = %v, want %v", c.itemId, adjustable, c.adjustable)
		}
	}
}
```

Check the real loader names (`LoadDataFiles` in each package) before running; adjust the two calls to match.

- [ ] **Step 3: Run it and watch it fail**

Run: `go test ./internal/items/ -run ShippedLightItems`
Expected: FAIL, 40077 is `object`.

- [ ] **Step 4: The four conditions**, in `_datafiles/world/dogmud/conditions/`

`124-candle_light.yaml`:

```yaml
conditionid: 124
name: Candlelight
description: A small flame lights the space around you.
secret: true
triggerrate: 5 real minutes
triggercount: 1
effects:
  light_strength: 38
```

`125-oil_lantern_light.yaml`: the same with `conditionid: 125`, `name: Lantern Light`, `description: Your lantern throws a steady light around you.`, `light_strength: 52`.

`126-torch_light.yaml`: `conditionid: 126`, `name: Torchlight`, `description: Your torch burns bright and smoky.`, `light_strength: 56`.

`127-hooded_lantern_light.yaml`: `conditionid: 127`, `name: Hooded Lantern Light`, `description: Your hooded lantern trims its light to your eyes.`, `light_strength: 54`, plus

```yaml
flags:
  - adjustable
```

Worn conditions are applied permanent, so `triggercount: 1` only keeps them from being born expired (`TestNoShippedConditionIsBornDead`).

- [ ] **Step 5: Make the lantern and candle work**

In `40038-oil_lantern.yaml` change `type: object` / `subtype: mundane` to `type: light` / `subtype: wearable` and add `wornconditionids: [125]`. Do the same for `40077-tallow_candle.yaml` with `[124]`. Keep their descriptions: both already describe giving light.

- [ ] **Step 6: The two new items**, in a new folder `_datafiles/world/dogmud/items/armor-20000/light/`

`20096-torch.yaml`:

```yaml
itemid: 20096
name: Torch
namesimple: torch
description: A length of pine wrapped in pitch-soaked rag. It burns hot
  and bright and throws a harsh, smoky light, too much of it on a
  sunny day.
type: light
subtype: wearable
weight: 0.5
value: 1
wornconditionids:
  - 126
```

`20097-hooded_lantern.yaml`:

```yaml
itemid: 20097
name: Hooded Lantern
namesimple: lantern
description: A brass lantern with a hinged hood over its glass. It
  trims its own light to your eyes each time you step somewhere new,
  and the hood can close it off entirely.
type: light
subtype: wearable
weight: 0.3
value: 20
wornconditionids:
  - 127
nouns:
  hood: The hood swings down over the glass. Use hood to close it and
    shut the light away, and unhood to throw it open at full strength.
```

The `hood` noun appears in the description text ("the hood can close"), so `look lantern` highlights it.

- [ ] **Step 7: Stock the merchants**

Run: `grep -rln -E "itemid: 40038" _datafiles/world/dogmud/mobs _datafiles/world/dogmud/shops`
For each file, read it: if 40038 sits under a `shop:` list, add `- itemid: 20096` and `- itemid: 20097` beside it with the same indentation. If it sits in the mob's own carried items, leave the file alone. Record which files you changed in the commit message.

- [ ] **Step 8: Run the tests and boot**

Run: `go test ./internal/items/ ./internal/conditions/ ./internal/usercommands/ ./internal/mobs/`
Expected: PASS. Then boot the worktree server once per `dogmud-shipping` and confirm it loads with no condition or item error; stop it by its PID.

- [ ] **Step 9: Commit**

```bash
git add internal/items/shipped_light_items_test.go _datafiles/world/dogmud/conditions/124-candle_light.yaml _datafiles/world/dogmud/conditions/125-oil_lantern_light.yaml _datafiles/world/dogmud/conditions/126-torch_light.yaml _datafiles/world/dogmud/conditions/127-hooded_lantern_light.yaml _datafiles/world/dogmud/items/materials-40000/40038-oil_lantern.yaml _datafiles/world/dogmud/items/materials-40000/40077-tallow_candle.yaml _datafiles/world/dogmud/items/armor-20000/light/20096-torch.yaml _datafiles/world/dogmud/items/armor-20000/light/20097-hooded_lantern.yaml <each merchant file from Step 7>
git commit -m "feat(content): candle, oil lantern, torch and hooded lantern shed real light"
```

---

### Task 14: Help pages

**Files:**
- Create: `_datafiles/world/dogmud/templates/help/light.template`, `seasons.template`, `moons.template`
- Modify: `chrysalis-glow.template`, `weather.template`, `equipment.template`, `biome.template`, `_datafiles/world/dogmud/keywords.yaml`

Load `dogmud-player-copy` first: 80 columns, no raw numbers for strength or duration, plain English.

- [ ] **Step 1: `light.template`**

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">light</ansi>

What you can see depends on the light where you stand and on your own
eyes. In good light you read faces and details. In dim light you make
out only shapes, and cannot tell who is who. In darkness you see
nothing at all. Light that is too bright dazzles you.

<ansi fg="yellow">Where light comes from</ansi>

  The sun by day, the moons by night, the lamps of towns and taverns,
  and anything the people around you carry. Trees, roofs and rock
  shut out the sky, and cloud, rain and storms dim it.

<ansi fg="yellow">Carrying a light</ansi>

  Everyone has a light slot. Equip a candle, a lantern or a torch
  there and it burns until you take it off. A candle shows you shapes
  in the dark; a lantern or a torch shows you faces. Every light adds
  to the others in the room, but a second light adds less than the
  first did.

  A plain lantern or torch always burns at full strength. Out of doors
  near a summer noon that is too much, and it will dazzle you.

<ansi fg="yellow">Lights that adjust</ansi>

  The Chrysalis Glow spell and the hooded lantern trim themselves to
  your eyes each time you walk into a new place, so they never dazzle
  you there. Nothing else trims them: if the light changes around you,
  yours stays as it was until you move.

  A fresh glow, a light you have just equipped, or a hood you have
  just opened starts at full strength. A strong enough light can
  dazzle creatures whose eyes are made for the dark.

  Use <ansi fg="command">hood</ansi> to close a hooded lantern and <ansi fg="command">unhood</ansi> to open it. A closed
  hood gives no light, and you are harder to spot behind it.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help chrysalis-glow</ansi>, <ansi fg="command">help hood</ansi>, <ansi fg="command">help weather</ansi>,
  <ansi fg="command">help seasons</ansi>, <ansi fg="command">help moons</ansi>, <ansi fg="command">help biome</ansi>, <ansi fg="command">help equipment</ansi>
```

- [ ] **Step 2: `seasons.template`**

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">seasons</ansi>

The year turns through four seasons. In summer the days are long, the
sun climbs high and the nights are short. In winter the sun stays low,
the days are short and the nights are long, far longer than a summer
night. Spring and autumn sit between.

A higher sun gives brighter days. Even at noon a winter sun lights the
open ground less than a summer one, and under deep trees the
difference can be enough to hide faces. Near a summer noon the open
sky is bright enough that a lantern or torch carried under it will
dazzle you.

Weather dims the sky in any season, and a roof, a canopy or rock shuts
it out altogether.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help light</ansi>, <ansi fg="command">help moons</ansi>, <ansi fg="command">help weather</ansi>, <ansi fg="command">help biome</ansi>
```

- [ ] **Step 3: `moons.template`**

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">moons</ansi>

Three moons cross the night sky: Swiftmoon, the Wanderer and the Eye.
Each waxes and wanes on its own cycle. Swiftmoon turns quickest, the
Wanderer more slowly, and the Eye slowest of all.

At night the moons and the stars are the only light out of doors.
Swiftmoon gives the most light of the three and the Eye the least.
When they are full, open ground shows you shapes, but never enough to
read a face. On a night when all three are dark only starlight is
left, and without a light of your own you are blind.

Cloud and trees take away what little moonlight there is.

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help light</ansi>, <ansi fg="command">help seasons</ansi>, <ansi fg="command">help weather</ansi>
```

- [ ] **Step 4: Register the topics** in `_datafiles/world/dogmud/keywords.yaml`

Under `general:` add `- light`, `- moons`, `- seasons`. Under `help-aliases:` add

```yaml
  light:            [lighting, lantern, lanterns, torch, torches, candle, candles, dazzle, dazzled, dark, darkness]
  seasons:          [season, winter, summer, spring, autumn, calendar]
  moons:            [moon, swiftmoon, wanderer, eye]
```

Before adding each alias, grep `keywords.yaml` for it: an alias that already routes elsewhere (for example `eye`, `dark` or `lantern`) must not be stolen. Drop any that collide and say so in the commit message. `darkness` moves to its own topic in 5d.

- [ ] **Step 5: Update the existing pages**

- `chrysalis-glow.template`: replace the first paragraph with

```
The <ansi fg="command">Chrysalis Glow</ansi> spell coaxes dormant Chrysalis spores to luminesce,
surrounding you in a soft, otherworldly light. The stronger your
willpower and your spellcasting, the brighter it shines and the longer
it lasts.

A fresh glow starts at full strength. The next time you walk into a
new place it trims itself to your eyes, so it never dazzles you. Cast
it again for full strength, or use <ansi fg="command">cancel glow</ansi> to let it go.
```

  set `Effect:` to `Surrounds you with light that adjusts to your eyes`, and extend "See also" with `<ansi fg="command">help light</ansi>, <ansi fg="command">help cancel</ansi>`. (The old text claimed the spell "enhances their abilities"; it never did.)
- `weather.template`: add before "See also": `Cloud, rain, snow and storms dim the light of the sun and the moons.` and extend "See also" with `help light`, `help seasons`, `help moons`.
- `equipment.template`: add the row `  <ansi fg="yellow">Light</ansi>        A candle, lantern or torch to see by.` after `Components`, and extend "See also" with `help light`.
- `biome.template`: add `Some places are open to the sky, some are shaded by trees, and some are roofed or underground. See help light.` before its usage block.

- [ ] **Step 6: Check width and render**

Run: `go test ./internal/usercommands/ ./internal/templates/ ./internal/devtools/`
Expected: PASS, including help completeness. Then in the booted worktree server run `help light`, `help lantern`, `help seasons`, `help moons`, `help hood`, `help glow` and confirm every line fits 80 columns.

- [ ] **Step 7: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/light.template _datafiles/world/dogmud/templates/help/seasons.template _datafiles/world/dogmud/templates/help/moons.template _datafiles/world/dogmud/templates/help/chrysalis-glow.template _datafiles/world/dogmud/templates/help/weather.template _datafiles/world/dogmud/templates/help/equipment.template _datafiles/world/dogmud/templates/help/biome.template _datafiles/world/dogmud/keywords.yaml
git commit -m "docs(help): light, seasons and moons, cross-linked with glow, weather and equipment"
```

---

### Task 15: `context.md` updates and the full gate

**Files:** `internal/lightscale/context.md`, `internal/conditions/context.md`, `internal/rooms/context.md`, `internal/messaging/context.md`, `internal/characters/context.md`, `internal/items/context.md`, `internal/usercommands/context.md`, `internal/hooks/context.md`, `internal/mobs/context.md`, `internal/configs/context.md`, `internal/itemvalue/context.md`

- [ ] **Step 1: Update each `context.md`** for what this plan added or removed there: `Trim`/`Polarity`; `light_strength`, `Adjustable`, `Cancellable`, the light record fields and `LightSources`, and the retired `lightsource` flag; `TrimLightFor`, per-record carried terms and `LightTerms.Raw`; `LightTrimTarget`; `EmitsLight`, `LightTerms`, `FindItemNoun`, the light slot; `ItemSpec.Nouns` and `items.Light`; `hood`, `unhood`, `cancel <spell>`; `lightSpellApplication`; `Mob.AddConditionMagnitude`; the six knobs; `SlotLight`.

- [ ] **Step 2: Audit the docs**

Run: `python tools/context_md_audit.py`
Expected: no phantom symbols.

- [ ] **Step 3: Run the whole suite and the lint gate**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: PASS, and neither lighting golden moved (`git status` shows no change under `testdata/`). Then run the pre-push gate from `dogmud-shipping` in its order.

- [ ] **Step 4: Commit**

```bash
git add internal/lightscale/context.md internal/conditions/context.md internal/rooms/context.md internal/messaging/context.md internal/characters/context.md internal/items/context.md internal/usercommands/context.md internal/hooks/context.md internal/mobs/context.md internal/configs/context.md internal/itemvalue/context.md
git commit -m "docs(context): record plan 5a's light model in each package"
```

---

### Task 16: Party playtest and content gate

- [ ] **Step 1: Run the party playtest**

Invoke the `playtest-scenario` skill (and read `dogmud-playtesting` first). Scenario: three agents in one party, one with only the starter glow, one with an Oil Lantern, one with a Hooded Lantern. Route: a lit street at night, then a cave or dungeon, then open ground at midday. At each room every agent reports what it sees on arrival (`look`) and again after the next member arrives. Order the party so the hooded-lantern bearer enters a dark room first once and last once.

Expected: the first adjustable light into a dark room trims to its bearer's comfort and later adjustable arrivals trim to nothing; nobody's light changes when someone else enters; the oil lantern bearer is dazzled at midday outdoors in high summer; `hood` darkens the room and `unhood` floods it; `cancel glow` ends the glow.

- [ ] **Step 2: Extract findings to memory**

Reports are gitignored. Write each finding to the graded lighting arc memory, marked fixed or open.

- [ ] **Step 3: Run the adversarial content gate**

Follow `dogmud-authoring-content`'s mandatory playtest gate for the new items: buy a torch and a hooded lantern from a merchant, equip each, look at the hood, and use quest 14's lantern in its tunnels.

- [ ] **Step 4: Open the PR**

Follow `dogmud-shipping`: `gh pr create --repo pruuk/DOGMud`, file count under 300, body listing the owner rulings honoured and the playtest result. The owner merges and deploys.
