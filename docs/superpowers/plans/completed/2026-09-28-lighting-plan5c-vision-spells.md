# Lighting Plan 5c: Vision Spells, Infravision Potion, Infravision Fixed

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Infravision sees shapes in any light down to minus its reach at a penalty that shrinks with reach, reach sources combine, and players gain a discoverable Night Vision spell, a costlier Heat Sight spell, and a crafted Pitsense Tincture.

**Architecture:** The optics stay in `internal/messaging`'s pure functions: `SightThroughWindow` loses its floor gate and `ComfortDistance` caps its dark fraction by a new pure `infraDarkCap`, so all 34 `SightMult` callers and the 8 melee `ComfortDistance` callers inherit the change untouched. `Character.InfraReach` combines every source through `lightscale.Combine` and caps. Glow's scaling hook generalises to one magnitude hook keyed by `ConditionSpec.ScaledKind`; the drink path reuses its existing potency for magnitude; the Purging Draught strips a set derived from item data.

**Tech Stack:** Go 1.x, YAML content under `_datafiles/world/dogmud`, `go test`.

**Spec:** `docs/superpowers/specs/2026-09-28-lighting-plan5c-vision-spells-design.md` (owner-approved 2026-09-28). Its facts table (20 rows) is authoritative; the rows below are the extra facts this plan relies on.

## Facts verified against source (2026-09-28, master `cf5af4c55`)

| # | Fact | Where |
|---|---|---|
| P1 | The window test row `{"reach does not help above the floor", 24, 0, 30, SightNone}` and the two `windowFloor` pin rows encode the gate this plan removes | `internal/messaging/window_test.go:50,62,65` |
| P2 | `comfortDistance` tests call the 5-argument pure form; keep it unchanged and add infra beside it | `internal/messaging/comfort_test.go:21,39` |
| P3 | Test helpers: `newChar` and `setBlinded` in `predicates_test.go:13,20`; `sightLight` (a `RoomVisibility` of a fixed level) in `participant_sight_test.go:31`; `conditions.SeedConditionsForTest(map) func()` replaces the registry | messaging tests; `internal/conditions` |
| P4 | Conditions tests build held records with `bs := Conditions{}; bs.Validate(true); bs.AddConditionMagnitude(id, triggers, magnitude)` | `internal/conditions/effects_test.go:84-89` |
| P5 | Mutation tests swap `allMutations` directly | `internal/mutations/mutations_test.go:905-917` |
| P6 | `characters` tests: `visionChar`, `pinMutationRankMultipliers`, `SeedMutationsForTest` | `internal/characters/vision_test.go:24,50,125` |
| P7 | Glow hook tests call `lightSpellApplication` directly with `spells.SpellData{PrimaryStat: "willpower"}` and `caster.SetSkill("spellcasting", n)` | `internal/hooks/light_spell_test.go` |
| P8 | `applySpellCondition` has 4 callers, all in `spell_resolution.go` (787, 1141, 1506, 1767); none change | grep |
| P9 | `drink.go`: `itemSpec := matchItem.GetSpec()` is an `ItemSpec` VALUE (line 135); the condition loop is at 264-289 | `internal/usercommands/drink.go` |
| P10 | `TestApplyPurgeEffects` loads NO items and expects condition 61 (inside the 54-75 block) stripped, so the block must stay as a floor under the derived set | `internal/usercommands/drink_purge_test.go:18-50` |
| P11 | `items.SeedItemsForTest(map[int]*ItemSpec) func()`; `items.GetAllItemSpecs() []ItemSpec`; `items.GetItemSpec(id) *ItemSpec`; `items.Potion` is `"potion"` | `internal/items/test_helpers.go:6`, `itemspec.go:516,723,130` |
| P12 | Boot loads conditions BEFORE items (`main.go:1644-1645`), but test binaries load items alone, so the potion magnitude check is a root guard test, not a load check | `main.go` |
| P13 | Root tests reach the shipped world with `configs.SetConfigForTest(t, configs.GetConfig())` then `configs.ReloadConfig()` then the loaders | `lighting_parity_golden_test.go:79-97` |
| P14 | The parity golden applies condition 29 (nightvision) and 85 (infrared) with `AddCondition(id, true)`; re-record flag `-update-lighting-parity`; row format `  infrared     sight=none   clear=false shapes=false` | `lighting_parity_golden_test.go:105-150` |
| P15 | `internal/devtools/helpfile_completeness_test.go` fails when a spell id or recipe id has no `templates/help/<id>.template` | `:77,:100` |
| P16 | Help template shape for a spell: `templates/help/chrysalis-glow.template`; for a recipe: `cats-eye-draught.template` | `_datafiles/world/dogmud/templates/help/` |
| P17 | Mob carried items: `character.items: [{itemid, dropchance}]`; per-item dropchance is a percent | `mobs/crash_site_interior/9554-hull_warden.yaml:35-37`; `internal/hooks/Death_MobLoot.go:37` |
| P18 | Pale Lurker 225 has no `character.items`; Blind Stalker 227 has `items: [- itemid: 40048]` | `mobs/ironwind_steppe/` |
| P19 | Condition file name is `{id}-{ConvertForFilename(name)}`: hyphens and spaces become `_` (`40206-gale_sinew_of_the_steppe.yaml`) | content convention |
| P20 | The config knob block to extend: `config.yaml` "LIGHT: SPELL SCALING" ends at `LightSpellDurationSkillDivisor: 20` (HEAD blob line 947); Go fields end at `config.balance.go:1229`; defaults loop at `config.balance.lighting.go:182-192`; `Lighting` struct at `config.lighting_accessor.go:10-30` | configs |
| P21 | `DarknessCombatPenalty` lives on `Balance`, not `Lighting`; ships 0.80 | `config.balance.go:306`; `config.yaml:918` |

## File map

| File | Change |
|---|---|
| `internal/configs/config.balance.go` | 8 new knob fields |
| `internal/configs/config.balance.lighting.go` | their defaults |
| `internal/configs/config.lighting_accessor.go` | 9 new `Lighting` fields (8 knobs plus `DarkCap`) |
| `internal/configs/config_lighting_5c_test.go` | new |
| `_datafiles/config.yaml` | 8 new keys |
| `internal/messaging/window.go`, `window_test.go`, `band_test.go` | drop the floor gate |
| `internal/messaging/comfort.go`, `comfort_test.go` | `infraDarkCap` and its wiring |
| `internal/conditions/effects.go`, `effects_test.go` | `EffectValues`, `ScaledKind`, the two-kind refusal |
| `internal/mutations/mutations.go`, `mutations_test.go` | `FlagValues` |
| `internal/characters/vision.go`, `vision_test.go` | `InfraReach` combines and caps |
| `internal/hooks/light_spell.go`, `light_spell_test.go` | `magnitudeSpellApplication` |
| `internal/items/itemspec.go`, `internal/items/potion_conditions.go` (new), `potion_conditions_test.go` (new) | `Magnitude` field, `PotionEffectConditionIds` |
| `internal/usercommands/drink.go`, `drink_magnitude.go` (new), `drink_magnitude_test.go` (new), `drink_purge_test.go` | magnitude path, derived purge |
| `potion_conditions_guard_test.go` (new, repo root) | shipped-world guard |
| content: 3 conditions, 2 spells, 1 material, 1 potion, 1 recipe, 2 mobs, 4 help templates | Task 9 |
| `testdata/lighting_parity.golden` | infrared rows only |
| 7 `context.md` files, `docs/PATCH_NOTES.md`, `docs/README.md` | Task 12 |

Subagent models: Tasks 1, 9, 12 sonnet; Tasks 2-8, 10, 11 opus; Task 14 per the playtest skill.

---

### Task 0: Worktree

- [ ] **Step 1: Create the worktree from the spec branch** (the spec and this plan ship in the PR, as 5b's did)

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
git worktree add ../DOGMud-plan5c -b feature/lighting-plan5c-vision-spells docs/lighting-plan5c-spec
```

- [ ] **Step 2: Confirm the worktree's `config.yaml` is the tracked blob with no skip-worktree bit**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud-plan5c"
git ls-files -v _datafiles/config.yaml
```
Expected: `H _datafiles/config.yaml` (a fresh worktree has no `S` bit, so edits here commit normally).

- [ ] **Step 3: Baseline**

```bash
go build ./... && go test ./internal/messaging/ ./internal/characters/ ./internal/hooks/ ./internal/usercommands/ -count=1
```
Expected: PASS. Every later task runs from this worktree.

---

### Task 1: Knobs

**Files:** Modify `internal/configs/config.balance.go` (after line 1229), `internal/configs/config.balance.lighting.go` (the loop at 182-192), `internal/configs/config.lighting_accessor.go`, `_datafiles/config.yaml`. Create `internal/configs/config_lighting_5c_test.go`.

- [ ] **Step 1: Write the failing test**

```go
package configs

import "testing"

// A zero Balance is what a test binary sees; every 5c knob must default.
func TestLighting5cKnobDefaults(t *testing.T) {
	var b Balance
	b.validateLighting()
	checks := []struct {
		name      string
		got, want float64
	}{
		{"LightNightVisionSpellBase", float64(b.LightNightVisionSpellBase), 4},
		{"LightNightVisionSpellStatDivisor", float64(b.LightNightVisionSpellStatDivisor), 12.5},
		{"LightNightVisionSpellSkillDivisor", float64(b.LightNightVisionSpellSkillDivisor), 6.5},
		{"LightInfraSpellBase", float64(b.LightInfraSpellBase), 5},
		{"LightInfraSpellStatDivisor", float64(b.LightInfraSpellStatDivisor), 7},
		{"LightInfraSpellSkillDivisor", float64(b.LightInfraSpellSkillDivisor), 3},
		{"LightInfraReachCap", float64(b.LightInfraReachCap), 50},
		{"LightInfraPenaltyFloor", float64(b.LightInfraPenaltyFloor), 0.90},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

// Out-of-range values revert rather than silently ship.
func TestLighting5cKnobRanges(t *testing.T) {
	b := Balance{LightInfraReachCap: 150, LightInfraPenaltyFloor: 1.5}
	b.validateLighting()
	if b.LightInfraReachCap != 50 || b.LightInfraPenaltyFloor != 0.90 {
		t.Errorf("cap %v floor %v, want 50 and 0.90", b.LightInfraReachCap, b.LightInfraPenaltyFloor)
	}
}

func TestLightingAccessorCarries5cKnobs(t *testing.T) {
	SetConfigForTest(t, GetConfig())
	l := GetLightingConfig()
	if l.InfraReachCap != 50 || l.InfraPenaltyFloor != 0.90 || l.DarkCap != 0.80 ||
		l.NightVisionSpellBase != 4 || l.InfraSpellSkillDivisor != 3 {
		t.Errorf("accessor = %+v", l)
	}
}
```

- [ ] **Step 2: Run it to see it fail**

Run: `go test ./internal/configs/ -run 'TestLighting5c|TestLightingAccessorCarries5c' -count=1`
Expected: FAIL to compile (unknown fields).

- [ ] **Step 3: Add the fields** after `LightSpellDurationSkillDivisor` in `config.balance.go`:

```go
	// Vision-spell scaling (lighting plan 5c), the house idiom of the light
	// trio above: base + stat/StatDivisor + spellcasting/SkillDivisor. A spell
	// whose condition declares nightvision_strength: magnitude or
	// infra_reach: magnitude is cast at that value, for the light trio's
	// duration. Shipped, for a new (100, 0), mid (130, 30) and endgame
	// (175, 65) caster: nightvision 12 / 19 / 28 (the window clamps at 24);
	// infra reach 19 / 34 / 52 (capped at LightInfraReachCap).
	LightNightVisionSpellBase         ConfigFloat `yaml:"LightNightVisionSpellBase"`         // default 4
	LightNightVisionSpellStatDivisor  ConfigFloat `yaml:"LightNightVisionSpellStatDivisor"`  // default 12.5
	LightNightVisionSpellSkillDivisor ConfigFloat `yaml:"LightNightVisionSpellSkillDivisor"` // default 6.5
	LightInfraSpellBase               ConfigFloat `yaml:"LightInfraSpellBase"`               // default 5
	LightInfraSpellStatDivisor        ConfigFloat `yaml:"LightInfraSpellStatDivisor"`        // default 7
	LightInfraSpellSkillDivisor       ConfigFloat `yaml:"LightInfraSpellSkillDivisor"`       // default 3

	// Infravision (lighting plan 5c). LightInfraReachCap bounds every
	// source's combined reach (spell, potion, mutation, condition).
	// LightInfraPenaltyFloor is the sight multiplier infravision gives at its
	// first point of reach; it rises linearly to no penalty at the cap, so
	// reach 25 reads 0.95. Infravision only ever eases the DARK side.
	LightInfraReachCap     ConfigInt   `yaml:"LightInfraReachCap"`     // default 50
	LightInfraPenaltyFloor ConfigFloat `yaml:"LightInfraPenaltyFloor"` // default 0.90
```

- [ ] **Step 4: Defaults.** In `validateLighting`, extend the loop's slice with six rows and add the two range checks after the loop:

```go
		{&b.LightNightVisionSpellBase, 4}, {&b.LightNightVisionSpellStatDivisor, 12.5}, {&b.LightNightVisionSpellSkillDivisor, 6.5},
		{&b.LightInfraSpellBase, 5}, {&b.LightInfraSpellStatDivisor, 7}, {&b.LightInfraSpellSkillDivisor, 3},
```

```go
	// Infravision. A cap of zero divides by zero in the penalty ramp; above
	// 100 reaches past the scale. The floor is a multiplier in (0, 1].
	if b.LightInfraReachCap <= 0 || b.LightInfraReachCap > 100 {
		b.LightInfraReachCap = 50
	}
	if b.LightInfraPenaltyFloor <= 0 || b.LightInfraPenaltyFloor > 1.0 {
		b.LightInfraPenaltyFloor = 0.90
	}
```

- [ ] **Step 5: Accessor.** Add to the `Lighting` struct after the `SpellDuration...` line:

```go
	NightVisionSpellBase, NightVisionSpellStatDivisor, NightVisionSpellSkillDivisor float64
	InfraSpellBase, InfraSpellStatDivisor, InfraSpellSkillDivisor                   float64

	InfraReachCap     int
	InfraPenaltyFloor float64
	// DarkCap is Balance.DarknessCombatPenalty, carried here because the
	// infravision dark cap (messaging.infraDarkCap) is expressed against it.
	DarkCap float64
```

and to the returned literal:

```go
		NightVisionSpellBase:         float64(b.LightNightVisionSpellBase),
		NightVisionSpellStatDivisor:  float64(b.LightNightVisionSpellStatDivisor),
		NightVisionSpellSkillDivisor: float64(b.LightNightVisionSpellSkillDivisor),
		InfraSpellBase:               float64(b.LightInfraSpellBase),
		InfraSpellStatDivisor:        float64(b.LightInfraSpellStatDivisor),
		InfraSpellSkillDivisor:       float64(b.LightInfraSpellSkillDivisor),
		InfraReachCap:                int(b.LightInfraReachCap),
		InfraPenaltyFloor:            float64(b.LightInfraPenaltyFloor),
		DarkCap:                      float64(b.DarknessCombatPenalty),
```

- [ ] **Step 6: `config.yaml`.** In the worktree (no skip-worktree bit there), insert after `LightSpellDurationSkillDivisor: 20`:

```yaml

  # ── LIGHT: VISION SPELLS AND INFRAVISION (lighting plan 5c) ────────────────
  # Night Vision and Heat Sight scale like the light spells above:
  #   base + stat/StatDivisor + spellcasting/SkillDivisor
  # and last the light spells' duration. Night vision for a new (100, 0),
  # mid (130, 30) and endgame (175, 65) caster: 12 / 19 / 28 (the window
  # clamps at 24). Infra reach: 19 / 34 / 52 (capped at LightInfraReachCap).
  LightNightVisionSpellBase: 4
  LightNightVisionSpellStatDivisor: 12.5
  LightNightVisionSpellSkillDivisor: 6.5
  LightInfraSpellBase: 5
  LightInfraSpellStatDivisor: 7
  LightInfraSpellSkillDivisor: 3
  # LightInfraReachCap bounds every infravision source's combined reach.
  # Infravision sees shapes in any light down to minus its reach. Its sight
  # multiplier is LightInfraPenaltyFloor at the first point of reach and
  # rises linearly to 1.0 (no penalty) at the cap. It never eases dazzle.
  LightInfraReachCap: 50
  LightInfraPenaltyFloor: 0.90
```

- [ ] **Step 7: Run the tests**

Run: `go test ./internal/configs/ -count=1`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.lighting.go internal/configs/config.lighting_accessor.go internal/configs/config_lighting_5c_test.go _datafiles/config.yaml
git commit -m "feat(lighting): 5c knobs for vision-spell scaling and infravision"
```

---

### Task 2: Infravision reads any light down to minus its reach

**Files:** Modify `internal/messaging/window.go:1-60`, `internal/messaging/window_test.go:49-66`, `internal/messaging/band_test.go:29-30`.

- [ ] **Step 1: Rewrite the reach rows in `window_test.go`.** Replace every row from `{"reach does not help above the floor", ...}` through `{"reach stops one step above the floor", ...}` (including the long `windowFloor` comment between them) with:

```go
		// Infravision reads heat, not light (lighting plan 5c): shapes at ANY
		// light down to minus its reach, never faces, and natural sight wins
		// wherever it reads better.
		{"infra reads a faint room", 12, 0, 30, SightShapes},
		{"infra reads light a normal eye is blind in", 24, 0, 30, SightShapes},
		{"reach without strength still reads dark", 0, 0, 10, SightShapes},
		{"a small reach still reads any lit room", 2, 0, 5, SightShapes},
		{"infra never gives faces, even at the cap", 10, 0, 50, SightShapes},
		{"natural faces beat infra", 60, 0, 30, SightFull},
```

and add below `TestSightThroughWindow`:

```go
// windowFloor binds only when an operator sets LightBlindBelow below 1 (the
// never-blind escape hatch): no ability shift can push the blind edge under
// it, and infra reach no longer consults it.
func TestWindowFloorHoldsForANeverBlindConfig(t *testing.T) {
	if got := SightThroughWindow(0, 0, 0, -10, 50); got != SightNone {
		t.Errorf("light 0 with blind edge -10 = %v, want SightNone (the floor)", got)
	}
	if got := SightThroughWindow(1, 0, 0, -10, 50); got != SightShapes {
		t.Errorf("light 1 with blind edge -10 = %v, want SightShapes", got)
	}
}
```

- [ ] **Step 2: Add a band row** after `{"reach has a limit", -11, 0, 10, BandDark},` in `band_test.go`:

```go
		{"reach reads a faint room", 12, 0, 10, BandShapes},
```

- [ ] **Step 3: Run to see them fail**

Run: `go test ./internal/messaging/ -run 'TestSightThroughWindow|TestWindowFloor|TestBand' -count=1`
Expected: FAIL on "infra reads a faint room", "infra reads light a normal eye is blind in", "a small reach still reads any lit room", "infra never gives faces, even at the cap", and the band row. "natural faces beat infra" and the floor test pass already; they pin behaviour that must not change.

- [ ] **Step 4: Change `window.go`.** Replace the `windowFloor` comment with:

```go
	// windowFloor is the light below which a shifted window reads nothing, no
	// matter how strong. Infra reach is independent of it: heat-sense reads
	// shapes at any light down to minus the reach (lighting plan 5c).
```

In `SightThroughWindow`'s doc comment replace the sentence beginning `reach is the separate` with:

```go
// heat-sensing extension: it reads SHAPES at any light down to the negation of
// reach, never faces, and only where the window itself reads worse (lighting
// plan 5c, the owner's ruling on 5b call 3).
```

and replace the final block (the comment starting `// Below the shifted window.` and the `if light <= windowFloor && ...` statement) with:

```go
	// Below the natural window. Infravision reads heat, not light, so it
	// gives shapes at ANY light down to minus its reach. It never yields
	// faces: natural sight has already won above wherever it reads fully.
	if reach > 0 && light >= -reach {
		return SightShapes
	}
	return SightNone
```

- [ ] **Step 5: Run the package**

Run: `go test ./internal/messaging/ -count=1`
Expected: PASS. If any other row in `internal/messaging` asserted infravision reads dark ABOVE light 1, it encodes the retired gate: flip it to shapes and name it in the commit body. Rows at or below light 1 must not change.

- [ ] **Step 6: Commit**

```bash
git add internal/messaging/window.go internal/messaging/window_test.go internal/messaging/band_test.go
git commit -m "feat(lighting): infravision reads shapes in any light down to minus its reach"
```

---

### Task 3: Infravision eases the dark side of the sight ramp

**Files:** Modify `internal/messaging/comfort.go`, `internal/messaging/comfort_test.go`.

- [ ] **Step 1: Write the failing tests** (append to `comfort_test.go`; add imports `configs` and `conditions` as needed; grep `7951` in `internal/messaging` first and pick a free id if taken):

```go
// Shipped knobs: floor 0.90, cap 50, dark cap 0.80. The cap is the dark
// fraction that makes SightScoreMultiplier read exactly infraMult.
func TestInfraDarkCap(t *testing.T) {
	cases := []struct {
		name        string
		light, reach int
		want        float64
		ok          bool
	}{
		{"reach 1", 0, 1, 0.49, true},
		{"reach 25", 0, 25, 0.25, true},
		{"reach 30, shipped condition 85", 0, 30, 0.20, true},
		{"reach 50 costs nothing", 0, 50, 0, true},
		{"reach above the cap clamps", 0, 80, 0, true},
		{"a faint room is in reach", 12, 30, 0.20, true},
		{"at minus reach", -30, 30, 0.20, true},
		{"below minus reach", -31, 30, 0, false},
		{"no reach", 0, 0, 0, false},
	}
	for _, c := range cases {
		got, ok := infraDarkCap(c.light, c.reach, 0.90, 50, 0.80)
		if ok != c.ok || !near(got, c.want) {
			t.Errorf("%s: (%v, %v), want (%v, %v)", c.name, got, ok, c.want, c.ok)
		}
	}
	if _, ok := infraDarkCap(0, 30, 0.90, 50, 1.0); ok {
		t.Error("a dark cap of 1.0 means no dark penalty exists; infra must not apply")
	}
}

const comfortInfraConditionId = 7951

func withInfraReach(t *testing.T, reach float64) *characters.Character {
	t.Helper()
	configs.SetConfigForTest(t, configs.GetConfig())
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		comfortInfraConditionId: {ConditionId: comfortInfraConditionId, Name: "Test Heat Sight",
			Flags:   []conditions.Flag{conditions.InfraredVision},
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectInfraReach: {Literal: reach}}},
	}))
	c := newChar(t)
	if err := c.AddCondition(comfortInfraConditionId, true); err != nil {
		t.Fatalf("AddCondition: %v", err)
	}
	return c
}

func TestComfortDistanceInfraEasesTheDarkOnly(t *testing.T) {
	c := withInfraReach(t, 30)
	if dark, bright := ComfortDistance(c, sightLight(0)); !near(dark, 0.20) || bright != 0 {
		t.Errorf("pitch dark, reach 30 = (%v, %v), want (0.20, 0)", dark, bright)
	}
	if dark, _ := ComfortDistance(c, sightLight(-40)); dark != 1 {
		t.Errorf("below minus reach = %v, want the full dark 1", dark)
	}
	if dark, _ := ComfortDistance(c, sightLight(45)); !near(dark, 0.20) {
		t.Errorf("dim light where the natural ramp is 0.2 = %v, want 0.20 (the better of the two)", dark)
	}
	if dark, _ := ComfortDistance(c, sightLight(48)); !near(dark, 0.08) {
		t.Errorf("light 48, natural ramp 0.08 beats infra = %v, want 0.08", dark)
	}
	if _, bright := ComfortDistance(c, sightLight(100)); bright != 1 {
		t.Errorf("glare must cost infravision in full: bright = %v, want 1", bright)
	}
	if m := SightMult(c, sightLight(0)); !near(m, 0.96) {
		t.Errorf("SightMult in the dark at reach 30 = %v, want 0.96", m)
	}
}
```

Light 45 check: natural dark is `(50-45)/25 = 0.2` and infra's cap is 0.2, so both read 0.20. Light 48: natural `0.08` is below infra's `0.20`, so natural wins.

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/messaging/ -run 'TestInfraDarkCap|TestComfortDistanceInfra' -count=1`
Expected: FAIL to compile (`infraDarkCap` undefined).

- [ ] **Step 3: Implement.** In `comfort.go`, replace the last two lines of `ComfortDistance` (`cfg := ...` and `return comfortDistance(...)`) with:

```go
	cfg := configs.GetLightingConfig()
	light := room.LightLevel()
	dark, bright = comfortDistance(light, observer.NightVisionStrength(), cfg.BlindBelow, cfg.DimBelow, cfg.DazzleAbove)
	if capped, ok := infraDarkCap(light, observer.InfraReach(), cfg.InfraPenaltyFloor, cfg.InfraReachCap, cfg.DarkCap); ok && capped < dark {
		dark = capped
	}
	return dark, bright
```

and add after `comfortDistance`:

```go
// infraDarkCap is the dark fraction infravision holds an observer to
// (lighting plan 5c). Infravision's sight multiplier runs linearly from floor
// at the first point of reach to 1.0 at reachCap; expressed as a dark
// fraction against darkCap (Balance.DarknessCombatPenalty) it makes
// SightScoreMultiplier read exactly that multiplier, so every caller of
// ComfortDistance and SightMult gets max(natural ramp, infra) with no change.
// ok is false when infravision does not apply: no reach, light below minus
// reach, or no dark penalty to ease. It never touches the bright side; glare
// costs an infravision creature in full (owner ruling 2026-09-28).
func infraDarkCap(light, reach int, floor float64, reachCap int, darkCap float64) (float64, bool) {
	if reach <= 0 || light < -reach || reachCap <= 0 || darkCap >= 1 {
		return 0, false
	}
	r := min(reach, reachCap)
	mult := floor + (1-floor)*float64(r)/float64(reachCap)
	return (1 - mult) / (1 - darkCap), true
}
```

Update `ComfortDistance`'s doc comment: after "A nil observer or room is comfortable; a Blinded observer is fully dark." add "Infravision caps the dark fraction (infraDarkCap) and never the bright one."

- [ ] **Step 4: Run the package**

Run: `go test ./internal/messaging/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/messaging/comfort.go internal/messaging/comfort_test.go
git commit -m "feat(lighting): infravision eases the dark side of the sight ramp by its reach"
```

---

### Task 4: Reach sources combine and cap

**Files:** Modify `internal/conditions/effects.go`, `effects_test.go`, `internal/mutations/mutations.go`, `mutations_test.go`, `internal/characters/vision.go`, `vision_test.go`.

- [ ] **Step 1: Failing test for `EffectValues`** (append to `effects_test.go`):

```go
func TestEffectValuesListsEveryHeldValue(t *testing.T) {
	withSpecs(t,
		&ConditionSpec{ConditionId: 960, Name: "Heat A", TriggerRate: "1 round", TriggerCount: 5, Effects: map[EffectKind]EffectValue{EffectInfraReach: {UsesMagnitude: true}}},
		&ConditionSpec{ConditionId: 961, Name: "Heat B", TriggerRate: "1 round", TriggerCount: 5, Effects: map[EffectKind]EffectValue{EffectInfraReach: {Literal: 30}}},
		&ConditionSpec{ConditionId: 962, Name: "Other", TriggerRate: "1 round", TriggerCount: 5, Effects: map[EffectKind]EffectValue{EffectNightVisionStrength: {Literal: 12}}},
	)
	bs := Conditions{}
	bs.Validate(true)
	bs.AddConditionMagnitude(960, 5, 22)
	bs.AddConditionMagnitude(961, 5, 0)
	bs.AddConditionMagnitude(962, 5, 0)
	got := bs.EffectValues(EffectInfraReach)
	sort.Float64s(got)
	if len(got) != 2 || got[0] != 22 || got[1] != 30 {
		t.Fatalf("EffectValues = %v, want [22 30]", got)
	}
	if bs.Effect(EffectInfraReach) != 30 {
		t.Fatal("Effect must still read the max for a max kind")
	}
}
```

(add `"sort"` to the imports).

- [ ] **Step 2: Failing test for `FlagValues`** (append to `mutations_test.go`):

```go
func TestFlagValuesListsEachRankScaledValue(t *testing.T) {
	prev := allMutations
	defer func() { allMutations = prev }()
	allMutations = map[string]*MutationSpec{
		"testmut-heat-a": {MutationId: "testmut-heat-a", Name: "Heat A", Rarity: 1,
			Pros: []MutationEffect{{Type: "flag", Target: "infraredvision", Value: 20}}},
		"testmut-heat-b": {MutationId: "testmut-heat-b", Name: "Heat B", Rarity: 1,
			Cons: []MutationEffect{{Type: "flag", Target: "infraredvision", Value: 10}}},
		"testmut-other": {MutationId: "testmut-other", Name: "Other", Rarity: 1,
			Pros: []MutationEffect{{Type: "flag", Target: "nightvision", Value: 18}}},
	}
	got := FlagValues(map[string]int{"testmut-heat-a": 1, "testmut-heat-b": 1, "testmut-other": 1}, "infraredvision")
	sort.Float64s(got)
	if len(got) != 2 || got[0] != 10 || got[1] != 20 {
		t.Fatalf("FlagValues = %v, want [10 20]", got)
	}
}
```

(add `"sort"` if absent).

- [ ] **Step 3: Failing test for the combine and cap** (append to `vision_test.go`; `visionCombineA`/`B` ids 7404, 7405):

```go
func TestInfraReachCombinesAndCaps(t *testing.T) {
	configs.SetConfigForTest(t, configs.GetConfig()) // doubling step 8, cap 50
	seed := func(reaches ...float64) *Character {
		specs := map[int]*conditions.ConditionSpec{}
		for i, r := range reaches {
			id := 7404 + i
			specs[id] = &conditions.ConditionSpec{ConditionId: id, Name: fmt.Sprintf("Heat %d", i),
				Flags:   []conditions.Flag{conditions.InfraredVision},
				Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectInfraReach: {Literal: r}}}
		}
		t.Cleanup(conditions.SeedConditionsForTest(specs))
		c := visionChar(t, "Viewer")
		for id := range specs {
			if err := c.AddCondition(id, true); err != nil {
				t.Fatalf("AddCondition(%d): %v", id, err)
			}
		}
		return c
	}
	if got := seed(30).InfraReach(); got != 30 {
		t.Errorf("one source 30 = %d, want 30", got)
	}
	if got := seed(30, 30).InfraReach(); got != 38 {
		t.Errorf("two equal sources 30 = %d, want 38 (one doubling step)", got)
	}
	if got := seed(40, 30, 25).InfraReach(); got != 46 {
		t.Errorf("40 + 30 + 25 = %d, want 46", got)
	}
	if got := seed(48, 48).InfraReach(); got != 50 {
		t.Errorf("48 + 48 = %d, want the cap 50", got)
	}
}
```

(add `"fmt"` to imports). Arithmetic: `40 + 8*log2(1 + 2^(-10/8) + 2^(-15/8)) = 40 + 8*log2(1.693) = 46.08`.

- [ ] **Step 4: Run to see them fail**

Run: `go test ./internal/conditions/ ./internal/mutations/ ./internal/characters/ -run 'TestEffectValues|TestFlagValues|TestInfraReachCombines' -count=1`
Expected: FAIL (undefined `EffectValues`, `FlagValues`; `InfraReach` reads 30 for two 30s).

- [ ] **Step 5: `EffectValues`** in `effects.go`, after `Effect`:

```go
// EffectValues returns every held, unexpired record's value for one kind,
// magnitude-aware exactly as Effect reads it, in list order. It exists for a
// reader that combines values some other way than Effect's own rule:
// Character.InfraReach log-sums reach through lightscale.Combine (lighting
// plan 5c). Records that do not declare the kind contribute nothing.
func (bs *Conditions) EffectValues(kind EffectKind) []float64 {
	var out []float64
	for _, b := range bs.List {
		if b.Expired() {
			continue
		}
		spec := GetConditionSpec(b.ConditionId)
		if spec == nil {
			continue
		}
		v, ok := spec.Effects[kind]
		if !ok {
			continue
		}
		val := v.Literal
		if v.UsesMagnitude {
			val = b.Magnitude
		}
		out = append(out, val)
	}
	return out
}
```

- [ ] **Step 6: `FlagValues`** in `mutations.go`, after `FlagValue`:

```go
// FlagValues returns each owned mutation's rank-scaled Value for flag, one
// entry per matching pro or con, zero values omitted. FlagValue takes the
// max of the same numbers; this is for a reader that combines them some
// other way (Character.InfraReach, lighting plan 5c).
func FlagValues(owned map[string]int, flag string) []float64 {
	var out []float64
	for id, level := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		mult := LevelMultiplier(level)
		for _, effects := range [][]MutationEffect{spec.Pros, spec.Cons} {
			for _, p := range effects {
				if p.Type == "flag" && p.Target == flag && p.Value != 0 {
					out = append(out, p.Value*mult)
				}
			}
		}
	}
	return out
}
```

- [ ] **Step 7: `InfraReach`** in `vision.go`. Replace its body and doc comment:

```go
// InfraReach reports how far into the dark this character still reads shapes
// by sensing heat: shapes at any light down to minus this number. Zero means
// not at all.
//
// Independent of NightVisionStrength on purpose: a creature can sense heat
// deeply while being no better than anyone else at using faint light.
//
// Unlike nightvision it COMBINES its sources (lighting plan 5c, owner ruling):
// every held condition's reach and every mutation's rank-scaled reach are
// log-summed through lightscale.Combine at the light scale's doubling step,
// the rule room light already uses, so two equal sources read one step above
// one and a much weaker source adds almost nothing. The result is capped at
// LightInfraReachCap and rounded once. A bare infrared flag with no number
// still reads zero.
func (c *Character) InfraReach() int {
	var vals []float64
	for _, v := range c.Conditions.EffectValues(conditions.EffectInfraReach) {
		if v > 0 {
			vals = append(vals, v)
		}
	}
	for _, v := range mutations.FlagValues(c.Mutations, string(conditions.InfraredVision)) {
		if v > 0 {
			vals = append(vals, v)
		}
	}
	if len(vals) == 0 {
		return 0
	}
	cfg := configs.GetLightingConfig()
	reach := lightscale.Combine(cfg.DoublingStep, vals...)
	if limit := float64(cfg.InfraReachCap); reach > limit {
		reach = limit
	}
	return int(math.Round(reach))
}
```

Add `"github.com/GoMudEngine/GoMud/internal/lightscale"` to the imports (`lightscale` imports only `math`, so no cycle). `bestVisionNumber` keeps serving `NightVisionStrength`; drop the `bareFlagNoDefault` constant if `go vet` reports it unused, and update `bestVisionNumber`'s doc comment to say it now serves nightvision only.

- [ ] **Step 8: Run the three packages**

Run: `go test ./internal/conditions/ ./internal/mutations/ ./internal/characters/ -count=1`
Expected: PASS, including the existing `TestVisionNumbers` bare-infrared subtest (still 0).

- [ ] **Step 9: Commit**

```bash
git add internal/conditions/effects.go internal/conditions/effects_test.go internal/mutations/mutations.go internal/mutations/mutations_test.go internal/characters/vision.go internal/characters/vision_test.go
git commit -m "feat(lighting): infravision reach sources log-sum and cap at 50"
```

---

### Task 5: `ScaledKind` and the one-magnitude-kind rule

**Files:** Modify `internal/conditions/effects.go`, `effects_test.go`.

- [ ] **Step 1: Failing tests** (append to `effects_test.go`):

```go
func TestScaledKind(t *testing.T) {
	cases := []struct {
		name   string
		fx     map[EffectKind]EffectValue
		want   EffectKind
		wantOk bool
	}{
		{"glow", map[EffectKind]EffectValue{EffectLightStrength: {UsesMagnitude: true}}, EffectLightStrength, true},
		{"night sight", map[EffectKind]EffectValue{EffectNightVisionStrength: {UsesMagnitude: true}}, EffectNightVisionStrength, true},
		{"heat sight", map[EffectKind]EffectValue{EffectNightVisionStrength: {Literal: 12}, EffectInfraReach: {UsesMagnitude: true}}, EffectInfraReach, true},
		{"literal only", map[EffectKind]EffectValue{EffectInfraReach: {Literal: 30}}, "", false},
		{"combat magnitude is not a sight kind", map[EffectKind]EffectValue{EffectDamageMult: {UsesMagnitude: true}}, "", false},
	}
	for _, c := range cases {
		s := &ConditionSpec{ConditionId: 970, Name: c.name, Effects: c.fx}
		got, ok := s.ScaledKind()
		if got != c.want || ok != c.wantOk {
			t.Errorf("%s: (%q, %v), want (%q, %v)", c.name, got, ok, c.want, c.wantOk)
		}
	}
}

func TestValidateRefusesTwoScaledKinds(t *testing.T) {
	s := &ConditionSpec{ConditionId: 971, Name: "Probe", TriggerRate: "1 round", TriggerCount: 1,
		Effects: map[EffectKind]EffectValue{EffectNightVisionStrength: {UsesMagnitude: true}, EffectInfraReach: {UsesMagnitude: true}}}
	if err := s.Validate(); err == nil {
		t.Fatal("a record carries one magnitude; two magnitude sight kinds must be refused at load")
	}
}
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/conditions/ -run 'TestScaledKind|TestValidateRefusesTwoScaledKinds' -count=1`
Expected: FAIL to compile.

- [ ] **Step 3: Implement** in `effects.go` after `isMax`:

```go
// ScaledKinds are the effect kinds a spell or potion scales from its source
// (lighting plan 5c): a light's strength, nightvision's strength, and infra's
// reach. A record carries one Magnitude, so a condition may declare at most
// one of them as "magnitude" (validateEffects refuses two).
var ScaledKinds = []EffectKind{EffectLightStrength, EffectNightVisionStrength, EffectInfraReach}

// ScaledKind reports which of ScaledKinds this condition reads from its
// record's magnitude, if any.
func (b *ConditionSpec) ScaledKind() (EffectKind, bool) {
	for _, k := range ScaledKinds {
		if v, ok := b.Effects[k]; ok && v.UsesMagnitude {
			return k, true
		}
	}
	return "", false
}
```

and in `validateEffects`, before the `light_strength` literal check:

```go
	scaled := 0
	for _, k := range ScaledKinds {
		if v, ok := b.Effects[k]; ok && v.UsesMagnitude {
			scaled++
		}
	}
	if scaled > 1 {
		return fmt.Errorf("conditionId %d (%s) reads more than one of %v from its magnitude; a record carries one magnitude", b.ConditionId, b.Name, ScaledKinds)
	}
```

- [ ] **Step 4: Run**

Run: `go test ./internal/conditions/ -count=1`
Expected: PASS (no shipped condition has two; verified 2026-09-28).

- [ ] **Step 5: Commit**

```bash
git add internal/conditions/effects.go internal/conditions/effects_test.go
git commit -m "feat(conditions): ScaledKind names a condition's one magnitude sight kind"
```

---

### Task 6: One magnitude hook for spells

**Files:** Modify `internal/hooks/light_spell.go`, `internal/hooks/light_spell_test.go`.

- [ ] **Step 1: Failing tests.** In `light_spell_test.go`, rename every `lightSpellApplication(` call to `magnitudeSpellApplication(`, then extend `seedTestGlowCondition`'s map with:

```go
		9733: {ConditionId: 9733, Name: "Test Night Sight", TriggerCount: 1, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectNightVisionStrength: {UsesMagnitude: true}},
			Flags:   []conditions.Flag{conditions.NightVision}},
		9734: {ConditionId: 9734, Name: "Test Heat Sight", TriggerCount: 1, RoundInterval: 1,
			Effects: map[conditions.EffectKind]conditions.EffectValue{
				conditions.EffectNightVisionStrength: {Literal: 12}, conditions.EffectInfraReach: {UsesMagnitude: true}},
			Flags: []conditions.Flag{conditions.InfraredVision}},
```

and append:

```go
func TestVisionSpellsScaleFromStatAndSkill(t *testing.T) {
	configs.SetConfigForTest(t, configs.GetConfig())
	seedTestGlowCondition(t)
	spell := &spells.SpellData{SpellId: "test-vision", PrimaryStat: "willpower"}
	cases := []struct {
		conditionId, stat, skill int
		wantMag                  float64
		wantTriggers             int
	}{
		{9733, 100, 0, 12, 4},
		{9733, 130, 30, 4 + 130/12.5 + 30/6.5, 6},
		{9733, 175, 65, 28, 9}, // uncapped here: the window clamps at 24
		{9734, 100, 0, 5 + 100/7.0, 4},
		{9734, 130, 30, 5 + 130/7.0 + 10, 6},
		{9734, 175, 65, 50, 9}, // 51.67 capped at LightInfraReachCap
	}
	for _, c := range cases {
		caster := characters.New()
		caster.Stats.Willpower.ValueAdj = c.stat
		caster.SetSkill("spellcasting", c.skill)
		mag, trig, ok := magnitudeSpellApplication(spell, caster, c.conditionId)
		if !ok || math.Abs(mag-c.wantMag) > 1e-9 || trig != c.wantTriggers {
			t.Errorf("condition %d stat %d skill %d: (%v, %d, %v), want (%v, %d, true)",
				c.conditionId, c.stat, c.skill, mag, trig, ok, c.wantMag, c.wantTriggers)
		}
	}
}
```

(add `"math"`). Duration uses the light trio: (100, 0) `2 + 2 + 0 = 4`; (130, 30) `2 + 2.6 + 1.5 = 6.1`, rounds to 6; (175, 65) `2 + 3.5 + 3.25 = 8.75`, rounds to 9.

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/hooks/ -run 'TestLightSpell|TestVisionSpells' -count=1`
Expected: FAIL to compile (`magnitudeSpellApplication` undefined).

- [ ] **Step 3: Implement.** Replace `lightSpellApplication` in `light_spell.go` with:

```go
// magnitudeSpellApplication reports how a spell's condition should be applied
// when it reads one of conditions.ScaledKinds from its magnitude: at a value
// and a duration scaled from the CASTER's primary stat and spellcasting skill
// (lighting plan 5a for light, 5c for nightvision and infra reach). Each kind
// has its own base + stat/D1 + skill/D2 trio; all three share the light
// duration trio. Infra reach is capped at LightInfraReachCap here so the
// record holds the value it acts at; nightvision is left to the window's own
// clamp. ok is false for any other condition, which keeps its authored
// application. A light then trims to its HOLDER's eyes, who may not be the
// caster.
func magnitudeSpellApplication(spellData *spells.SpellData, caster *characters.Character, conditionId int) (magnitude float64, triggers int, ok bool) {
	if spellData == nil || caster == nil {
		return 0, 0, false
	}
	spec := conditions.GetConditionSpec(conditionId)
	if spec == nil {
		return 0, 0, false
	}
	kind, scaled := spec.ScaledKind()
	if !scaled {
		return 0, 0, false
	}
	cfg := configs.GetLightingConfig()
	base, statDiv, skillDiv := cfg.SpellStrengthBase, cfg.SpellStrengthStatDivisor, cfg.SpellStrengthSkillDivisor
	switch kind {
	case conditions.EffectNightVisionStrength:
		base, statDiv, skillDiv = cfg.NightVisionSpellBase, cfg.NightVisionSpellStatDivisor, cfg.NightVisionSpellSkillDivisor
	case conditions.EffectInfraReach:
		base, statDiv, skillDiv = cfg.InfraSpellBase, cfg.InfraSpellStatDivisor, cfg.InfraSpellSkillDivisor
	}
	stat := float64(spellData.CasterStatValue(caster.Stats))
	skill := float64(caster.GetSkillLevel(skills.Spellcasting))
	magnitude = base + stat/statDiv + skill/skillDiv
	if kind == conditions.EffectInfraReach {
		if limit := float64(cfg.InfraReachCap); magnitude > limit {
			magnitude = limit
		}
	}
	triggers = int(math.Round(cfg.SpellDurationBase + stat/cfg.SpellDurationStatDivisor + skill/cfg.SpellDurationSkillDivisor))
	if triggers < 1 {
		triggers = 1
	}
	return magnitude, triggers, true
}
```

In `applySpellCondition`, change the call to `magnitudeSpellApplication` and its comment's "a magnitude light" to "a magnitude-scaled light or sight".

- [ ] **Step 4: Run**

Run: `go test ./internal/hooks/ -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/light_spell.go internal/hooks/light_spell_test.go
git commit -m "feat(lighting): one magnitude hook scales light, nightvision and infra spells"
```

---

### Task 7: Potions scale magnitude by their potency

**Files:** Modify `internal/items/itemspec.go` (the `ItemSpec` struct near `Toxicity`, line 352), `internal/usercommands/drink.go:254-289`. Create `internal/usercommands/drink_magnitude.go`, `internal/usercommands/drink_magnitude_test.go`.

- [ ] **Step 1: Failing test** (`drink_magnitude_test.go`):

```go
package usercommands

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestPotionMagnitudeApplication(t *testing.T) {
	configs.SetConfigForTest(t, configs.GetConfig())
	heat := &conditions.ConditionSpec{ConditionId: 9801, Name: "Test Tincture", TriggerCount: 400,
		Effects: map[conditions.EffectKind]conditions.EffectValue{
			conditions.EffectNightVisionStrength: {Literal: 12}, conditions.EffectInfraReach: {UsesMagnitude: true}}}
	plain := &conditions.ConditionSpec{ConditionId: 9802, Name: "Test Brew", TriggerCount: 400}
	tincture := &items.ItemSpec{ItemId: 39998, Magnitude: 20}

	cases := []struct {
		name         string
		mult         float64
		wantMag      float64
		wantTriggers int
	}{
		{"fresh, alchemy 30", 1.0 * 1.3, 26, 520},
		{"peak, alchemy 50", 1.3 * 1.5, 39, 780},
		{"peak, alchemy 100 caps", 1.3 * 2.0, 50, 1040},
	}
	for _, c := range cases {
		mag, trig, ok := potionMagnitudeApplication(tincture, heat, c.mult)
		if !ok || math.Abs(mag-c.wantMag) > 1e-9 || trig != c.wantTriggers {
			t.Errorf("%s: (%v, %d, %v), want (%v, %d, true)", c.name, mag, trig, ok, c.wantMag, c.wantTriggers)
		}
	}
	if _, _, ok := potionMagnitudeApplication(tincture, plain, 1.3); ok {
		t.Error("a condition with no scaled kind keeps the duration-only path")
	}
	if _, _, ok := potionMagnitudeApplication(&items.ItemSpec{ItemId: 39997}, heat, 1.3); ok {
		t.Error("an item with no magnitude must not apply a zero-strength record")
	}
}
```

- [ ] **Step 2: Run to see it fail**

Run: `go test ./internal/usercommands/ -run TestPotionMagnitudeApplication -count=1`
Expected: FAIL to compile.

- [ ] **Step 3: Item field.** Add to `ItemSpec` after `Toxicity`:

```go
	// Magnitude is the base strength for a condition in ConditionIds that
	// reads its magnitude (conditions.ScaledKinds, lighting plan 5c). The
	// drink path scales it by the same potency multiplier as duration. A
	// potion carrying such a condition must declare it (repo-root guard
	// potion_conditions_guard_test.go).
	Magnitude float64 `yaml:"magnitude,omitempty"`
```

- [ ] **Step 4: Helper** (`drink_magnitude.go`):

```go
package usercommands

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// potionMagnitudeApplication reports how a potion applies a condition that
// reads one of conditions.ScaledKinds from its magnitude (lighting plan 5c):
// the item's Magnitude and the condition's trigger count, both scaled by the
// drink path's existing potency multiplier (aging phase times the crafter's
// skill). Infra reach is capped at LightInfraReachCap. ok is false for a
// condition with no scaled kind or an item with no Magnitude, which keep the
// duration-only path.
func potionMagnitudeApplication(itemSpec *items.ItemSpec, spec *conditions.ConditionSpec, durationMult float64) (magnitude float64, triggers int, ok bool) {
	if itemSpec == nil || spec == nil || itemSpec.Magnitude <= 0 {
		return 0, 0, false
	}
	kind, scaled := spec.ScaledKind()
	if !scaled {
		return 0, 0, false
	}
	if durationMult <= 0 {
		durationMult = 1
	}
	magnitude = itemSpec.Magnitude * durationMult
	if kind == conditions.EffectInfraReach {
		if limit := float64(configs.GetLightingConfig().InfraReachCap); magnitude > limit {
			magnitude = limit
		}
	}
	triggers = int(math.Round(float64(spec.TriggerCount) * durationMult))
	if triggers < 1 {
		triggers = 1
	}
	return magnitude, triggers, true
}
```

- [ ] **Step 5: Wire it into `drink.go`.** Replace the head of the condition loop (the `for` line through `user.AddConditionScaled(conditionId, durationMult, `drink`)`) with:

```go
	for _, conditionId := range itemSpec.ConditionIds {
		conditionSpec := conditions.GetConditionSpec(conditionId)
		if mag, trig, ok := potionMagnitudeApplication(&itemSpec, conditionSpec, durationMult); ok {
			// A magnitude-scaled potion (lighting plan 5c) queues through the
			// same event door with its value and count; the holder still reads
			// the start line.
			user.AddConditionMagnitude(conditionId, trig, mag, `drink`)
		} else {
			user.AddConditionScaled(conditionId, durationMult, `drink`)
		}
```

and in the tick block below, replace `if conditionSpec := conditions.GetConditionSpec(conditionId); conditionSpec != nil && conditionSpec.TickPool != "" {` with `if conditionSpec != nil && conditionSpec.TickPool != "" {`.

- [ ] **Step 6: Run**

Run: `go build ./... && go test ./internal/usercommands/ ./internal/items/ -count=1`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/items/itemspec.go internal/usercommands/drink.go internal/usercommands/drink_magnitude.go internal/usercommands/drink_magnitude_test.go
git commit -m "feat(lighting): a potion scales its condition's magnitude by its potency"
```

---

### Task 8: The Purging Draught strips a derived potion set

**Files:** Create `internal/items/potion_conditions.go`, `internal/items/potion_conditions_test.go`. Modify `internal/usercommands/drink.go:38-84`, `internal/usercommands/drink_purge_test.go`.

- [ ] **Step 1: Failing test** (`potion_conditions_test.go`):

```go
package items

import "testing"

func TestPotionEffectConditionIds(t *testing.T) {
	t.Cleanup(SeedItemsForTest(map[int]*ItemSpec{
		1: {ItemId: 1, Type: Potion, ConditionIds: []int{7, 5}},
		2: {ItemId: 2, Type: Potion, ConditionIds: []int{130}},
		3: {ItemId: 3, Type: Food, ConditionIds: []int{5}},
		4: {ItemId: 4, Type: Legs, WornConditionIds: []int{7}},
	}))
	got := PotionEffectConditionIds()
	if !got[130] {
		t.Error("130 is named only by a potion; it must be in the set")
	}
	if got[5] {
		t.Error("5 is also granted by food; stripping it would undo a meal")
	}
	if got[7] {
		t.Error("7 is also granted while worn; stripping it would undo armour")
	}
}
```

`Food` and `Legs` are `ItemType` constants (`itemspec.go:125-145`).

- [ ] **Step 2: Extend `drink_purge_test.go`** with a derived-set case:

```go
func TestApplyPurgeEffectsStripsADerivedPotionCondition(t *testing.T) {
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		82: {ConditionId: 82, Name: "Steady Hand", TriggerCount: 400, RoundInterval: 1},
		93: {ConditionId: 93, Name: "Bloom Detox", TriggerCount: 400, RoundInterval: 1},
		76: {ConditionId: 76, Name: "Purging Weakness", TriggerCount: 50, RoundInterval: 1},
	}))
	t.Cleanup(items.SeedItemsForTest(map[int]*items.ItemSpec{
		30060:              {ItemId: 30060, Type: items.Potion, ConditionIds: []int{82}},
		ysoldesPurgeItemId: {ItemId: ysoldesPurgeItemId, Type: items.Potion, ConditionIds: []int{93}},
	}))
	c := characters.New()
	u := &users.UserRecord{UserId: 7105, Character: c}
	for _, id := range []int{82, 93} {
		if err := c.AddConditionScaled(id, 1.0); err != nil {
			t.Fatalf("setup %d: %v", id, err)
		}
	}
	events.DrainQueuedConditionsForTest(u.UserId)
	applyPurgeEffects(u)
	c.Conditions.Prune()
	if c.HasCondition(82) {
		t.Error("82 sits outside the old 54-75 block but only a potion grants it; the purge must strip it")
	}
	if !c.HasCondition(93) {
		t.Error("93 belongs to a detox item; one detox must not strip another")
	}
}
```

(add the `items` import).

- [ ] **Step 3: Run to see them fail**

Run: `go test ./internal/items/ ./internal/usercommands/ -run 'TestPotionEffectConditionIds|TestApplyPurgeEffects' -count=1`
Expected: FAIL to compile (`PotionEffectConditionIds` undefined).

- [ ] **Step 4: `potion_conditions.go`:**

```go
package items

// PotionEffectConditionIds returns every condition id that only potions
// grant: named by a potion's ConditionIds and by no non-potion item's
// ConditionIds or WornConditionIds. The Purging Draught strips this set
// (lighting plan 5c), which replaced a hardcoded id block that shipped
// potions had already outgrown. It is computed on each call from the loaded
// specs, so an admin item reload cannot leave it stale.
func PotionEffectConditionIds() map[int]bool {
	potion := map[int]bool{}
	other := map[int]bool{}
	for _, spec := range GetAllItemSpecs() {
		if spec.Type == Potion {
			for _, id := range spec.ConditionIds {
				potion[id] = true
			}
			continue
		}
		for _, id := range spec.ConditionIds {
			other[id] = true
		}
		for _, id := range spec.WornConditionIds {
			other[id] = true
		}
	}
	for id := range other {
		delete(potion, id)
	}
	return potion
}
```

- [ ] **Step 5: `drink.go`.** Replace the `// Potion-effect conditions occupy a contiguous id block.` comment with:

```go
// The original potion-effect conditions occupy a contiguous id block. It
// stays as the floor of what a purge strips (so a test binary with no items
// loaded keeps today's behaviour), and purgeableConditionIds adds every
// condition only a potion grants (lighting plan 5c closed the leak: 7, 44 to
// 51 and 82 had escaped the block). 76 is the purge's own weakness harmful
// condition: it is APPLIED by a purge, never stripped by one. 70 is the
// draught's flavour condition, which carries no statmods and expires after a
// round; a purge leaves it alone so the drinker still sees that they drank
// something.
```

Add after `bypassesToxicityGate`:

```go
// purgeableConditionIds is what a purge strips: the original block plus every
// condition only a potion grants, minus the conditions of both detox items
// (one detox does not undo another) and the weakness a purge applies.
func purgeableConditionIds() map[int]bool {
	set := items.PotionEffectConditionIds()
	for id := potionConditionIdMin; id <= potionConditionIdMax; id++ {
		set[id] = true
	}
	for _, detoxId := range []int{purgingDraughtItemId, ysoldesPurgeItemId} {
		if spec := items.GetItemSpec(detoxId); spec != nil {
			for _, id := range spec.ConditionIds {
				delete(set, id)
			}
		}
	}
	delete(set, purgingDraughtConditionId)
	delete(set, purgingWeaknessConditionId)
	return set
}
```

and in `applyPurgeEffects` replace the `for id := potionConditionIdMin; ...` loop with:

```go
	for id := range purgeableConditionIds() {
		c.RemoveCondition(id)
	}
```

- [ ] **Step 6: Run**

Run: `go test ./internal/items/ ./internal/usercommands/ -count=1`
Expected: PASS, including the original `TestApplyPurgeEffects` (61 is in the block).

- [ ] **Step 7: Commit**

```bash
git add internal/items/potion_conditions.go internal/items/potion_conditions_test.go internal/usercommands/drink.go internal/usercommands/drink_purge_test.go
git commit -m "fix(alchemy): the Purging Draught strips every potion-only condition, not just 54-75"
```

---

### Task 9: Content

Use the `dogmud-authoring-content` and `dogmud-player-copy` skills. Every line of player text wraps at 80 columns, shows no raw numbers, and carries no em or en dashes.

**Files (all under `_datafiles/world/dogmud/`):** Create `conditions/128-night_sight.yaml`, `conditions/129-heat_sight.yaml`, `conditions/130-pitsense_tincture.yaml`, `spells/night-vision.yaml`, `spells/heat-sight.yaml`, `items/materials-40000/40233-heat_pit_organ.yaml`, `items/consumables-30000/30068-pitsense_tincture.yaml`, `recipes/alchemy/pitsense-tincture.yaml`, `templates/help/night-vision.template`, `templates/help/heat-sight.template`, `templates/help/pitsense-tincture.template`. Modify `mobs/ironwind_steppe/225-pale_lurker.yaml`, `mobs/ironwind_steppe/227-blind_stalker.yaml`, `templates/help/light.template`.

- [ ] **Step 1: Re-confirm the ids are free**

Run: `python tools/id_inventory.py --alloc conditions 3 && python tools/id_inventory.py --alloc items 1`
Expected: conditions 128-130; items 40233. Check `ls _datafiles/world/dogmud/items/consumables-30000/30068-*` finds nothing.

- [ ] **Step 2: `conditions/128-night_sight.yaml`**

```yaml
conditionid: 128
name: Night Sight
description: Your eyes make use of light too faint for most people to read
  by, though pitch darkness still leaves you blind. Daylight or any bright
  lamp dazzles these sharpened eyes, and everything you do by sight suffers
  until the light fades or the spell wears off.
start_actee: The dark thins and you make out grey shapes around you.
end_actee: The dark closes in again and the shapes are gone.
start_observer: "{actee}'s eyes widen and catch the faint light."
end_observer: "The faint light leaves {actee}'s eyes."
secret: false
triggerrate: 3 real minutes
# triggercount is a floor for the born-dead guard; the Night Vision spell
# sets the real count from its caster (lighting plan 5c).
triggercount: 1
# nightvision_strength: magnitude. The spell scales it from willpower and
# spellcasting (LightNightVisionSpell* in config.yaml); the window clamps it
# at 24. Condition 29 stays literal because species grant it to mobs.
effects:
  nightvision_strength: magnitude
flags:
  - nightvision
```

- [ ] **Step 3: `conditions/129-heat_sight.yaml`**

```yaml
conditionid: 129
name: Heat Sight
description: You see the warmth of living things as pale shapes, in any
  darkness short of the deepest magic. Heat shows a shape, never a face.
  Your eyes are also a little sharper for faint light, so daylight or any
  bright lamp dazzles you, and everything you do by sight suffers until the
  light fades or the spell wears off.
start_actee: Warmth blooms into pale shapes wherever something lives.
end_actee: The pale shapes of warmth fade from your sight.
start_observer: "A dull red sheen settles over {actee}'s eyes."
end_observer: "The red sheen fades from {actee}'s eyes."
secret: false
triggerrate: 3 real minutes
triggercount: 1
# infra_reach: magnitude, scaled by the Heat Sight spell (LightInfraSpell*)
# and capped at LightInfraReachCap. nightvision_strength 12 matches
# condition 85, the mobs' infravision.
effects:
  nightvision_strength: 12
  infra_reach: magnitude
flags:
  - infraredvision
```

- [ ] **Step 4: `conditions/130-pitsense_tincture.yaml`**

```yaml
conditionid: 130
name: Pitsense Tincture
description: The warmth of living things shows as pale shapes in any
  darkness short of the deepest magic, though never clearly enough to know
  a face. Your eyes are also a little sharper for faint light, so daylight
  or any bright lamp dazzles you, and everything you do by sight suffers
  until the light fades or the tincture wears off.
start_actee: A dull heat spreads behind your eyes, and living warmth shows
  as pale shapes.
end_actee: The heat behind your eyes cools, and the pale shapes fade.
start_observer: "A dull red sheen settles over {actee}'s eyes."
end_observer: "The red sheen fades from {actee}'s eyes."
secret: false
triggerrate: 1 round
# 400 rounds before potency; the drink path scales both this count and the
# reach by the potion's aging phase and its crafter's alchemy.
triggercount: 400
effects:
  nightvision_strength: 12
  infra_reach: magnitude
flags:
  - infraredvision
```

- [ ] **Step 5: `spells/night-vision.yaml`**

```yaml
# Found by discovery, never taught (lighting plan 5c).
spellid: night-vision
name: Night Vision
aliases: [nightvision]
description: Sharpens the eyes for faint light. Strong light dazzles them
  while it lasts.
attack_type: none
damage_type: non_harm
targeting: single
schools:
  - mental
cost: 45
waitrounds: 2
difficulty: 15
primarystat: willpower
effect_type: condition
condition_ids:
  - 128
cast_actor: "You draw the faint light of this place into your eyes."
cast_observer: "{actor} blinks slowly as their pupils widen."
wait_actor: "You hold the widening steady..."
```

- [ ] **Step 6: `spells/heat-sight.yaml`**

```yaml
# Found by discovery, never taught (lighting plan 5c). Costs more and needs
# more spellcasting than Night Vision (owner ruling 2026-09-28).
spellid: heat-sight
name: Heat Sight
aliases: [infravision]
description: Turns the eyes from light to the warmth of living things.
attack_type: none
damage_type: non_harm
targeting: single
schools:
  - mental
cost: 80
waitrounds: 3
difficulty: 35
primarystat: willpower
effect_type: condition
condition_ids:
  - 129
cast_actor: "You turn your sight from light to warmth."
cast_observer: "{actor}'s eyes take on a dull red sheen."
wait_actor: "You let the warmth of living things gather in your sight..."
```

- [ ] **Step 7: `items/materials-40000/40233-heat_pit_organ.yaml`**

```yaml
itemid: 40233
name: Heat-Pit Organ
namesimple: organ
description: A shallow cup of pale tissue lined with fine nerves, cut from
  the face of a creature that hunts blind by the warmth of its prey. It is
  still faintly warm. Alchemists prize it for draughts that let an eye
  borrow the same sense.
type: object
subtype: mundane
component_tag: heat-pit
weight: 0.1
value: 45
rarity_tier: 20
is_component: true
vendor_categories:
- alchemy
material_tier: 3
```

- [ ] **Step 8: `items/consumables-30000/30068-pitsense_tincture.yaml`**

```yaml
itemid: 30068
name: Pitsense Tincture
vendor_categories:
- alchemy
namesimple: tincture
description: A dark red tincture steeped from a blind hunter's heat-pit.
  Living warmth shows as pale shapes even in darkness no eye could use,
  though never clearly enough to know a face.
type: potion
subtype: drinkable
uses: 1
conditionids:
  - 130
# Base infravision reach. The drink path scales it by aging and the
# crafter's alchemy (fresh at the recipe minimum reads 26, peak from a master
# reaches the cap of 50). See ItemSpec.Magnitude.
magnitude: 20
value: 60
rarity_tier: 30
weight: 0.4
toxicity: 25
aging:
  ferment_rounds: 1500
  peak_rounds: 6000
  decay_rounds: 18000
  spoil_rounds: 30000
```

- [ ] **Step 9: `recipes/alchemy/pitsense-tincture.yaml`**

```yaml
id: pitsense-tincture
name: Pitsense Tincture
aliases: [pitsense]
skill: alchemy
skill_minimum: 30
station: alchemy_bench
time_rounds: 6
ingredients:
  - item_tag: heat-pit
    quantity: 1
  - item_tag: moonpetal
    quantity: 2
  - item_tag: bottle
    quantity: 1
output:
  item_id: 30068
  quantity: 1
success_actor: "The heat-pit steeps into the moonpetal and the tincture darkens to red. A pitsense tincture!"
failure_actor: "The heat-pit tissue curdles and the tincture turns cloudy. The materials are wasted."
```

- [ ] **Step 10: Mob drops.** In `225-pale_lurker.yaml`, under `character:` after `level: 1` (two-space indent), add:

```yaml
  items:
    - itemid: 40233
      dropchance: 15
```

In `227-blind_stalker.yaml`, under the existing `items:` after `- itemid: 40048`, add:

```yaml
    - itemid: 40233
      dropchance: 15
```

- [ ] **Step 11: Help templates.** `templates/help/night-vision.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">night-vision</ansi> spell

The <ansi fg="command">Night Vision</ansi> spell sharpens the eyes for faint light, so dim
places show you faces and dark places show you shapes. Pitch darkness
still leaves you blind. The stronger your willpower and your
spellcasting, the sharper it makes you and the longer it lasts.

The same sharpened eyes are dazzled sooner. Daylight or any bright lamp
costs you while the spell lasts.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">cast night-vision</ansi> [target]

<ansi fg="yellow">Details: </ansi>

  <ansi fg="yellow">Type:        </ansi> Help Single
  <ansi fg="yellow">School:      </ansi> Mental
  <ansi fg="yellow">Conv. Cost:  </ansi> 45
  <ansi fg="yellow">Effect:      </ansi> Sees in faint light; dazzled by bright light

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help light</ansi>, <ansi fg="command">help heat-sight</ansi>, <ansi fg="command">help spells</ansi>
```

`templates/help/heat-sight.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="command">heat-sight</ansi> spell

The <ansi fg="command">Heat Sight</ansi> spell turns your eyes from light to the warmth of
living things. You see pale shapes in any darkness short of the
deepest magic, even where no light reaches at all. Heat shows a shape,
never a face. The stronger your willpower and your spellcasting, the
deeper it sees, the less the dark costs you, and the longer it lasts.

It does nothing against glare: daylight or a bright lamp still
dazzles you.

<ansi fg="yellow">Usage: </ansi>

  <ansi fg="command">cast heat-sight</ansi> [target]

<ansi fg="yellow">Details: </ansi>

  <ansi fg="yellow">Type:        </ansi> Help Single
  <ansi fg="yellow">School:      </ansi> Mental
  <ansi fg="yellow">Conv. Cost:  </ansi> 80
  <ansi fg="yellow">Effect:      </ansi> Sees the shapes of living warmth in darkness

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help light</ansi>, <ansi fg="command">help night-vision</ansi>, <ansi fg="command">help spells</ansi>
```

`templates/help/pitsense-tincture.template`:

```
<ansi fg="black-bold">.:</ansi> <ansi fg="magenta">Help for </ansi><ansi fg="skill">pitsense-tincture</ansi> (alchemy recipe)

A dark red tincture that lets you see the warmth of living things as
pale shapes in any darkness short of the deepest magic. A well-aged
tincture from a skilled alchemist sees deeper and lasts longer.

<ansi fg="yellow">━━━ Crafting ━━━</ansi>

  <ansi fg="yellow">Skill:</ansi>       alchemy (minimum 30)
  <ansi fg="yellow">Station:</ansi>     alchemy bench
  <ansi fg="yellow">Ingredients:</ansi> heat-pit organ (x1), moonpetal (x2), bottle (x1)

<ansi fg="magenta-bold">See also:</ansi> <ansi fg="command">help alchemy</ansi>, <ansi fg="command">help craft</ansi>, <ansi fg="command">help light</ansi>
```

- [ ] **Step 12: `light.template`.** Insert before the `See also` line:

```
<ansi fg="yellow">Eyes for the dark</ansi>

  Night vision lets your eyes use fainter light, but the same eyes are
  dazzled sooner. Infravision is different: it sees the warmth of
  living things, not light, so it shows you shapes in any darkness it
  can reach, and the stronger it is the less the dark costs you. It
  never shows a face, and it does nothing against glare.

  The Night Vision and Heat Sight spells grant them, as do the Cat's
  Eye Draught and the Pitsense Tincture.

```

and add `<ansi fg="command">help night-vision</ansi>, <ansi fg="command">help heat-sight</ansi>,` to the See also list, re-wrapping it so no rendered line passes 80 columns.

- [ ] **Step 13: Validate content**

Run: `go test ./internal/devtools/ ./internal/conditions/ ./internal/spells/ ./internal/items/ ./internal/crafting/ ./internal/templates/ -count=1` then `go test . -run 'TestShippedNarrationDataValidates|TestConditionNotice|TestConditionFlag' -count=1`
Expected: PASS. A failure naming one of the new files is a content bug; fix the file.

- [ ] **Step 14: Commit**

```bash
git add _datafiles/world/dogmud/conditions/128-night_sight.yaml _datafiles/world/dogmud/conditions/129-heat_sight.yaml _datafiles/world/dogmud/conditions/130-pitsense_tincture.yaml _datafiles/world/dogmud/spells/night-vision.yaml _datafiles/world/dogmud/spells/heat-sight.yaml _datafiles/world/dogmud/items/materials-40000/40233-heat_pit_organ.yaml _datafiles/world/dogmud/items/consumables-30000/30068-pitsense_tincture.yaml _datafiles/world/dogmud/recipes/alchemy/pitsense-tincture.yaml _datafiles/world/dogmud/mobs/ironwind_steppe/225-pale_lurker.yaml _datafiles/world/dogmud/mobs/ironwind_steppe/227-blind_stalker.yaml _datafiles/world/dogmud/templates/help/night-vision.template _datafiles/world/dogmud/templates/help/heat-sight.template _datafiles/world/dogmud/templates/help/pitsense-tincture.template _datafiles/world/dogmud/templates/help/light.template
git commit -m "content(lighting): Night Vision and Heat Sight spells, Pitsense Tincture, Heat-Pit Organ"
```

---

### Task 10: Shipped-world guard

**Files:** Create `potion_conditions_guard_test.go` (repo root).

- [ ] **Step 1: Write the guard**

```go
package main

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// loadShippedConditionsAndItems follows lighting_parity_golden_test.go: the
// real config (so DataFiles is the dogmud world), then conditions before
// items, the boot order in main.go.
func loadShippedConditionsAndItems(t *testing.T) {
	t.Helper()
	mudlog.SetupLogger(nil, `LOW`, ``, false)
	configs.SetConfigForTest(t, configs.GetConfig())
	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}
	conditions.LoadDataFiles()
	items.LoadDataFiles()
	if len(items.GetAllItemSpecs()) < 500 {
		t.Fatalf("loaded only %d items: the guard is not seeing the world", len(items.GetAllItemSpecs()))
	}
}

// TestPotionMagnitudeIsDeclared: a potion whose condition reads its
// magnitude (lighting plan 5c) with no item Magnitude would apply a
// zero-strength record, which reads as no effect at all. Checked here, at
// build time, because a load-time check would break every test binary that
// loads items without conditions.
func TestPotionMagnitudeIsDeclared(t *testing.T) {
	loadShippedConditionsAndItems(t)
	checked := 0
	for _, spec := range items.GetAllItemSpecs() {
		for _, id := range spec.ConditionIds {
			cs := conditions.GetConditionSpec(id)
			if cs == nil {
				continue
			}
			if _, scaled := cs.ScaledKind(); scaled {
				checked++
				if spec.Magnitude <= 0 {
					t.Errorf("item %d (%s) applies condition %d, which reads its magnitude, but declares no magnitude", spec.ItemId, spec.Name, id)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no shipped item applies a magnitude condition: the Pitsense Tincture is missing, so this guard proves nothing")
	}
}

// TestPurgeablePotionSetOnTheShippedWorld pins the derived set the Purging
// Draught strips against the 2026-09-28 audit.
func TestPurgeablePotionSetOnTheShippedWorld(t *testing.T) {
	loadShippedConditionsAndItems(t)
	set := items.PotionEffectConditionIds()
	for _, id := range []int{7, 44, 47, 48, 49, 51, 82, 130} {
		if !set[id] {
			t.Errorf("condition %d is granted only by a potion; it must be in the set", id)
		}
	}
	for id := 54; id <= 75; id++ {
		if id == 70 {
			continue
		}
		if !set[id] {
			t.Errorf("condition %d from the old block left the set", id)
		}
	}
	if set[5] {
		t.Error("condition 5 is also granted by food and bandages; it must not be in the set")
	}
}
```

The 70 and 93 exclusions are the drink path's (detox items), pinned by `TestApplyPurgeEffectsStripsADerivedPotionCondition`.

- [ ] **Step 2: Prove each guard can fail.**
  1. Temporarily delete the `magnitude: 20` line from `30068-pitsense_tincture.yaml`. Run `go test . -run TestPotionMagnitudeIsDeclared -count=1`; expect FAIL naming item 30068. Restore with `git checkout -- _datafiles/world/dogmud/items/consumables-30000/30068-pitsense_tincture.yaml` only if Task 9 is already committed; otherwise re-add the line by hand.
  2. Condition 5 is granted by both the Fungal Ration (30013) and the Cloth Bandage (30020). Temporarily change the `5` in BOTH files' `conditionids` to `6`. Run `go test . -run TestPurgeablePotionSetOnTheShippedWorld -count=1`; expect FAIL on condition 5. Restore with `git checkout -- _datafiles/world/dogmud/items/consumables-30000/30013-fungal_ration.yaml _datafiles/world/dogmud/items/consumables-30000/30020-cloth_bandage.yaml`.

- [ ] **Step 3: Run clean**

Run: `go test . -run 'TestPotionMagnitudeIsDeclared|TestPurgeablePotionSet' -count=1`
Expected: PASS. `git status` shows only the new test file.

- [ ] **Step 4: Commit**

```bash
git add potion_conditions_guard_test.go
git commit -m "test(guard): potion magnitude is declared and the purge set is pinned"
```

---

### Task 11: The parity golden

**Files:** `testdata/lighting_parity.golden`.

- [ ] **Step 1: See it fail**

Run: `go test . -run TestLightingParityAcrossEveryShippedRoom -count=1`
Expected: FAIL (infrared rows moved).

- [ ] **Step 2: Re-record, then prove the shape of the move before keeping it**

```bash
go test . -run TestLightingParityAcrossEveryShippedRoom -update-lighting-parity -count=1
git diff -U0 testdata/lighting_parity.golden | grep -E '^[-+] ' | grep -vc 'infrared'
```
Expected: `0` (only infrared rows moved; `grep -c` exits 1 on zero, so run it standalone). Then:

```bash
git diff -U0 testdata/lighting_parity.golden | grep -E '^- ' | grep -vc 'sight=none'
git diff -U0 testdata/lighting_parity.golden | grep -E '^\+ ' | grep -vc 'sight=shapes'
```
Expected: `0` and `0`: every moved row went none to shapes, which is the faint-room and dim-room gap closing. If the nightvision profile (condition 29) moved, or any row went the other way, stop: that is a defect, not a re-record.

- [ ] **Step 3: Other goldens.** Run `go test . -count=1`. `lighting_daycycle.golden` records light, not sight, and must not move. Any other golden that moves gets the same filtered diff; only infravision observers may change.

- [ ] **Step 4: Commit** with the counts in the body

```bash
git add testdata/lighting_parity.golden
git commit -m "test(golden): infravision rows read shapes in faint rooms (lighting 5c)"
```

---

### Task 12: Documentation

**Files:** `internal/messaging/context.md` (window section near line 234, `comfort.go` entry, file table line 430), `internal/characters/context.md` (1856-1888), `internal/conditions/context.md` (near 649-656), `internal/mutations/context.md` (155-167), `internal/hooks/context.md` (965), `internal/configs/context.md` (702), `internal/items/context.md` (the `ItemSpec` field list), `internal/usercommands/context.md` (the drink section, if it names the purge block), `docs/PATCH_NOTES.md`, `docs/README.md`.

- [ ] **Step 1:** Update each `context.md` to name what now exists: `infraDarkCap`; reach reads shapes at any light down to minus reach; `Conditions.EffectValues`, `ScaledKinds`, `ConditionSpec.ScaledKind`, the two-kind refusal; `mutations.FlagValues`; `InfraReach` log-sums and caps; `magnitudeSpellApplication` (replacing every `lightSpellApplication` mention); the eight knobs and `Lighting.DarkCap`; `ItemSpec.Magnitude`, `items.PotionEffectConditionIds`; `purgeableConditionIds`. Verify every symbol with `Select-String -Path internal\<pkg>\*.go -Pattern '^(func|type|const|var)\s'` before naming it.

- [ ] **Step 2:** Run `python tools/context_md_audit.py`. Expected: no phantom symbols in the touched files.

- [ ] **Step 3: `docs/PATCH_NOTES.md`**, a new top entry (80 columns, no numbers, no dashes):

```markdown
## 2026-09-28: Eyes for the dark

Infravision now does what it promises. It sees the warmth of living
things, not light, so it shows you shapes in any darkness it can reach,
from a faint cellar to a cave no lamp has ever touched. The stronger it
is, the less the dark costs you, and at its strongest the dark costs you
nothing at all. It still never shows you a face, and it does nothing
against glare.

Two new spells can be discovered: Night Vision, which sharpens your eyes
for faint light, and Heat Sight, a harder spell that grants infravision.
Both grow stronger with your willpower and spellcasting.

Alchemists can brew a Pitsense Tincture from the heat-pit of a blind cave
hunter. An aged tincture from a skilled hand sees deeper and lasts longer.

The Purging Draught now clears every potion's effect, including several
older potions it used to miss.
```

- [ ] **Step 3b:** In `docs/README.md`, add a row for this plan below the 5c spec row: "Implementation plan for slice 5c: knobs, the infravision window and dark cap, reach combining, `ScaledKind`, one magnitude hook, potion magnitude, the derived purge set, content, a shipped-world guard, the parity golden, docs, playtest."

- [ ] **Step 4: Commit**

```bash
git add internal/messaging/context.md internal/characters/context.md internal/conditions/context.md internal/mutations/context.md internal/hooks/context.md internal/configs/context.md internal/items/context.md internal/usercommands/context.md docs/PATCH_NOTES.md docs/README.md
git commit -m "docs(lighting): 5c context.md, patch notes, README"
```

(Drop `internal/usercommands/context.md` from the `git add` if Step 1 left it unchanged.)

---

### Task 13: Gate

Follow the `dogmud-shipping` skill's pre-push gate order.

- [ ] **Step 1:** `gofmt -l ./internal ./*.go` (on Windows a CRLF false positive is possible; confirm with `git diff --check`), `go vet ./...`, `go build ./...`.
- [ ] **Step 2:** `go test ./... -count=1`. Expected: PASS. A test that counts spells, recipes, conditions or items may need its count raised by the new content; raise it and say so in a commit. Any sight-related failure outside the rows this plan names is a finding: stop and report it.
- [ ] **Step 3:** Boot check per `dogmud-shipping` (detached worktree boot, PID-scoped teardown, never a blanket kill). The AI-companion boot check is waived for plan 5 (owner 2026-09-26).

---

### Task 14: Playtest gate

Use the `dogmud-playtesting` skill (ephemeral goals file, `--checkout` at the branch head, a fixture that survives several rounds, `docker rm -f` teardown).

- [ ] **Step 1:** One scenario, two actors granted the spells by admin (`setcondition` is admin only; or learn them through a seeded spellbook): an infravision caster and a nightvision caster, in (a) a faint room (the skylight-0.1 holding cells 5105/5106 at night), (b) a pitch-dark Ironwind cave, (c) a lit tavern by day. Each fights a condition-85 mob.
- [ ] **Step 2: Checks.** Infravision reads shapes (names hidden) in (a) and (b); nightvision reads shapes or faces per its strength; glare in (c) costs both; the Pale Lurker or Blind Stalker drops a Heat-Pit Organ within a reasonable kill count; a crafted Pitsense Tincture grants infravision and `drink` shows the start line; a Purging Draught strips it.
- [ ] **Step 3:** Extract findings to memory (reports are gitignored). A blocker is fixed on the branch; the rest go in the PR body.

---

### Task 15: PR

Follow `dogmud-shipping`. Every `gh` carries `--repo pruuk/DOGMud`. Named paths only.

- [ ] **Step 1:** `git push -u origin feature/lighting-plan5c-vision-spells`
- [ ] **Step 2:** `gh pr create --repo pruuk/DOGMud --base master --title "Lighting 5c: vision spells, infravision potion, infravision fixed" --body-file <file inside the worktree>` with: what changed, the owner rulings, the parity golden's filtered diff counts, the playtest summary, and the intended consequences (condition-85 mobs see shapes in faint rooms and pay less in the dark). End the body with the attribution line.
- [ ] **Step 3:** Report the PR URL. The owner merges and deploys.
