# Lighting plan 4: weather occlusion, implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Weather dims the sky by kind through a mutator `skylight` fraction that replaces the `-2..2` `LightMod`, and the positive bridge and the unreferenced `bioluminescent_caves` mutator go.

**Architecture:** `MutatorSpec` gains `SkyLight *float64` (validated 0 to 1). `rooms.composeLight` multiplies the room's sky fraction by the product of active mutators' `SkyLight` (`mutatorSkyFilter`), so weather is a subtraction on the log scale and never touches lamps or carried light. `LightTerms.SkyFilter` replaces `OcclusionSteps` and `LightMod`; `lightnotice` attributes the weather cause from it. `LightMod` is deleted from Go first so the compiler enumerates its consumers; the admin template (reflection) is fixed in the same commit.

**Tech Stack:** Go, YAML via `internal/fileloader`, `text/template` admin pages.

**Spec:** `docs/superpowers/specs/2026-09-25-lighting-plan4-occlusion-design.md` (owner approved 2026-09-25).

---

## Facts verified against source

Branch `feature/lighting-plan4-occlusion` at `82594e0bb` (master `e4d5b8985` plus the spec), 2026-09-25. The spec's own facts table (12 rows) also holds; these are the extra facts this plan relies on.

| # | Fact | Source |
|---|---|---|
| 1 | `lightLevel(cfg, celestial)` calls `mutatorLightTerms()` then `lightLevelWithMutatorBridge(cfg, celestial, lightMod, occlusionSteps int) int`, which returns `composeLight(...).Level`; `LightTerms()` calls `composeLight(GetLightingConfig(), CelestialLight(), lightMod, occlusionSteps)` | `internal/rooms/lighting.go:43-83` |
| 2 | `composeLight` builds up to 4 terms: sky (`Attenuate(step, celestial, skyFraction * 2^-occ)`), lamp, positive bridge (`DimBelow + (lightMod-1)*step`), carried (`DimBelow`) | `internal/rooms/lighting.go:85-144` |
| 3 | `mutatorLightTerms` ranges `r.ActiveMutators`, `spec := mut.GetSpec()` (nil possible) | `internal/rooms/lighting.go:152-167` |
| 4 | Tests on the old signatures/fields: `lighting_model_test.go:64-84` (`TestPositiveLightModBridgeKeepsLitCavesLit`, `TestNegativeLightModAttenuatesSkyNotLamp`), `light_terms_test.go` (`TestComposeLightLevelMatchesBridgeAcrossInputs`, `TestLightTermsReportEachTerm`), `lightnotice/tracker_test.go:153-156` | grep |
| 5 | `lightnotice.attribute`: `case a.HasLamp != b.HasLamp \|\| a.Lamp != b.Lamp \|\| a.LightMod != b.LightMod: CauseLamp` then `case a.OcclusionSteps != b.OcclusionSteps: CauseWeather` | `internal/lightnotice/tracker.go:157-160` |
| 6 | `MutatorSpec.Validate()` only normalises text-modifier behaviours and returns nil; `mutators.go` imports `fmt` and `github.com/pkg/errors` | `internal/mutators/mutators.go:3-14,321-336` |
| 7 | `mutators.SeedSpecsForTest(specs ...MutatorSpec) func()` copies the registry and restores it; with no args it just snapshots | `internal/mutators/test_helpers.go` |
| 8 | Room-plus-zone mutator test pattern: `seedRegistry()`, `SeedBiomesForTest` with an `Indoor: true` biome, `mutators.SeedSpecsForTest`, `GetZoneConfig("TestZone").Mutators.Add(id)`, `roomManager.rooms[1]` / `[2]` | `internal/rooms/rooms_mutator_filter_test.go` |
| 9 | Shipped-world clock helpers in package rooms tests: `withShippedBiomesAndClock(t)`, `setClock(doy, hour)`, `requireBiome(t, id)` | `internal/rooms/city_tier_light_test.go:15-55` |
| 10 | `mutators.LoadDataFiles()` reads `configs.GetFilePathsConfig().DataFiles + "/mutators"` and panics on error | `internal/mutators/mutators.go:339-352` |
| 11 | The admin mutator page is `text/template` (`internal/web/admin.mutators.go:6,42`); `text/template` indirects a pointer when printing (`printableValue`), so `{{ $mutator.SkyLight }}` prints the number; `TestAdminMutatorTemplateExecutesWithConditionIds` already executes the template (`internal/web/admin_condition_templates_test.go:86-117`) and will fail while the template names a deleted field | those files |
| 12 | Comments naming the bridge outside rooms: `internal/usercommands/look_exit_visibility_test.go:39`, `lighting_parity_golden_test.go:91`; docs: `internal/rooms/context.md:65,82`, `internal/lightnotice/context.md:59`, `internal/mutators/context.md:42`, `internal/lightnotice/store.go:26` | grep |
| 13 | Weather YAML: `lightmod: -1` at line 15 of storm, dust, blizzard; fog, overcast, rain, snow, heatwave carry no light key | `_datafiles/world/dogmud/mutators/weather_*.yaml` |

---

## File map

| Path | Change |
|---|---|
| `internal/mutators/mutators.go` | add `SkyLight`, validate it (Task 1); delete `LightMod` (Task 3) |
| `internal/mutators/sky_light_test.go` | create: Validate range (Task 1) |
| `internal/mutators/shipped_sky_light_test.go` | create: shipped values (Task 2), no `lightmod` key guard (Task 4) |
| `_datafiles/world/dogmud/mutators/weather_{fog,overcast,rain,snow,storm,dust,blizzard}.yaml` | add `skylight` (Task 2); drop `lightmod` (Task 4) |
| `_datafiles/world/dogmud/mutators/bioluminescent_caves.yaml` | delete (Task 4) |
| `internal/rooms/lighting.go` | sky filter replaces bridge (Task 3) |
| `internal/rooms/lighting_model_test.go`, `light_terms_test.go` | rewrite old-signature tests (Task 3) |
| `internal/rooms/sky_filter_test.go` | create: filter product (Task 3) |
| `internal/rooms/weather_occlusion_test.go` | create: shipped-data behaviour (Task 5) |
| `internal/lightnotice/tracker.go`, `tracker_test.go`, `store.go` | weather cause from `SkyFilter` (Task 3) |
| `_datafiles/html/admin/mutators/mutator.data.html`, `internal/web/admin_condition_templates_test.go` | sky-light display (Task 3) |
| `internal/usercommands/look_exit_visibility_test.go`, `lighting_parity_golden_test.go` | comments (Task 3) |
| `internal/{rooms,lightnotice,mutators}/context.md`, `_datafiles/world/dogmud/narration/light-notices/lamp.yaml`, `docs/README.md`, `docs/PATCH_NOTES.md` | docs (Task 6) |

---

### Task 1: `MutatorSpec.SkyLight` and its validation

**Files:** Modify `internal/mutators/mutators.go`; create `internal/mutators/sky_light_test.go`.

- [ ] **Step 1: Failing test** `internal/mutators/sky_light_test.go`:

```go
package mutators

import (
	"math"
	"testing"
)

func TestSkyLightValidate(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	for _, ok := range []*float64{nil, f(0), f(0.5), f(1)} {
		spec := MutatorSpec{MutatorId: "sky-ok", SkyLight: ok}
		if err := spec.Validate(); err != nil {
			t.Errorf("skylight %v refused: %v", ok, err)
		}
	}
	for _, bad := range []float64{-0.1, 1.5, math.NaN()} {
		spec := MutatorSpec{MutatorId: "sky-bad", SkyLight: f(bad)}
		if err := spec.Validate(); err == nil {
			t.Errorf("skylight %v accepted; it must lie within 0 to 1", bad)
		}
	}
}
```

- [ ] **Step 2:** `go test ./internal/mutators/ -run TestSkyLightValidate -count=1` fails (`unknown field SkyLight`).

- [ ] **Step 3: Implement.** In `MutatorSpec`, directly after the `LightMod` line add:

```go
	// SkyLight is the fraction of the sky's light this mutator lets through,
	// the same word and meaning as a biome's or a room's skylight: 1 changes
	// nothing, 0.5 is one doubling step darker. Several active mutators
	// multiply. Nil leaves the sky alone. Lamps and carried light are never
	// touched.
	SkyLight *float64 `yaml:"skylight,omitempty"`
```

At the top of `Validate`, before the text-modifier checks:

```go
	if m.SkyLight != nil {
		if v := *m.SkyLight; math.IsNaN(v) || v < 0 || v > 1 {
			return errors.Errorf("mutator %q skylight %v is outside 0 to 1", m.MutatorId, v)
		}
	}
```

Add `"math"` to the imports if absent.

- [ ] **Step 4:** `go test ./internal/mutators/ -count=1` passes; `go build ./...` clean.

- [ ] **Step 5: Commit** `internal/mutators/mutators.go internal/mutators/sky_light_test.go`, message `feat(mutators): a skylight fraction a mutator lets through`.

---

### Task 2: Shipped weather `skylight` values

**Files:** Modify seven `_datafiles/world/dogmud/mutators/weather_*.yaml`; create `internal/mutators/shipped_sky_light_test.go`.

- [ ] **Step 1: Failing test** `internal/mutators/shipped_sky_light_test.go`:

```go
package mutators

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/fileloader"
)

// shippedMutatorDir is the dogmud world's mutator folder. A test binary never
// reads config.yaml, so this test loads it explicitly.
const shippedMutatorDir = "../../_datafiles/world/dogmud/mutators"

func loadShippedMutators(t *testing.T) map[string]*MutatorSpec {
	t.Helper()
	specs, err := fileloader.LoadAllFlatFiles[string, *MutatorSpec](shippedMutatorDir)
	if err != nil {
		t.Fatalf("loading shipped mutators: %v", err)
	}
	if len(specs) == 0 {
		t.Fatal("no shipped mutators loaded; the test would be vacuous")
	}
	return specs
}

// TestShippedWeatherSkyLight pins the owner's gentle grading: light weather
// lets 0.7 of the sky through, heavy weather 0.5, and nothing else filters it.
func TestShippedWeatherSkyLight(t *testing.T) {
	want := map[string]float64{
		"weather-fog":      0.7,
		"weather-overcast": 0.7,
		"weather-rain":     0.7,
		"weather-snow":     0.7,
		"weather-storm":    0.5,
		"weather-dust":     0.5,
		"weather-blizzard": 0.5,
	}
	for id, spec := range loadShippedMutators(t) {
		w, filtered := want[id]
		switch {
		case filtered && spec.SkyLight == nil:
			t.Errorf("%s: no skylight, want %v", id, w)
		case filtered && *spec.SkyLight != w:
			t.Errorf("%s: skylight %v, want %v", id, *spec.SkyLight, w)
		case !filtered && spec.SkyLight != nil:
			t.Errorf("%s: skylight %v, but only weather filters the sky", id, *spec.SkyLight)
		}
		delete(want, id)
	}
	for id := range want {
		t.Errorf("%s is not shipped", id)
	}
}
```

Before writing, confirm each mutator id by reading the `mutatorid:` line of each file (fact 13 names files, not ids).

- [ ] **Step 2:** `go test ./internal/mutators/ -run TestShippedWeatherSkyLight -count=1` fails (seven missing).

- [ ] **Step 3:** Add `skylight: 0.7` to `weather_fog.yaml`, `weather_overcast.yaml`, `weather_rain.yaml`, `weather_snow.yaml`, and `skylight: 0.5` to `weather_storm.yaml`, `weather_dust.yaml`, `weather_blizzard.yaml`, each on its own line directly above `decayrate:`. Leave the existing `lightmod: -1` lines alone in this task (Task 4 removes them).

- [ ] **Step 4:** `go test ./internal/mutators/ -count=1` passes.

- [ ] **Step 5: Commit** the seven YAML files and the test, message `content(weather): skylight fractions for every sky-dimming weather`.

---

### Task 3: The sky filter replaces the bridge (delete `LightMod`)

**Files:** `internal/mutators/mutators.go`, `internal/rooms/lighting.go`, `internal/rooms/lighting_model_test.go`, `internal/rooms/light_terms_test.go`, create `internal/rooms/sky_filter_test.go`, `internal/lightnotice/tracker.go`, `tracker_test.go`, `store.go`, `_datafiles/html/admin/mutators/mutator.data.html`, `internal/web/admin_condition_templates_test.go`, comments in `internal/usercommands/look_exit_visibility_test.go` and `lighting_parity_golden_test.go`.

- [ ] **Step 1: Failing tests.**

Create `internal/rooms/sky_filter_test.go`:

```go
package rooms

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutators"
)

// A filter multiplies the sky fraction, which on the log scale is a fixed
// subtraction: 0.7 about 4 points, 0.5 one step, 0.35 about 12. Lamps and a
// sky-less room are untouched.
func TestSkyFilterSubtractsFromTheSkyOnly(t *testing.T) {
	cfg := modelCfg()
	open, zero := 1.0, 0.0
	lamp := 40

	sky := Room{SkyLight: &open}
	clear := sky.lightLevelWithSkyFilter(cfg, 60, 1)
	for _, c := range []struct {
		filter float64
		drop   int
	}{{0.7, 4}, {0.5, 8}, {0.35, 12}} {
		if got := clear - sky.lightLevelWithSkyFilter(cfg, 60, c.filter); got != c.drop {
			t.Errorf("filter %v dropped the sky by %d, want %d", c.filter, got, c.drop)
		}
	}

	lampOnly := Room{SkyLight: &zero, Lamp: &lamp}
	if a, b := lampOnly.lightLevelWithSkyFilter(cfg, 60, 1), lampOnly.lightLevelWithSkyFilter(cfg, 60, 0.35); a != b {
		t.Errorf("a filter moved a lamp: %d clear, %d filtered", a, b)
	}

	cave := Room{SkyLight: &zero}
	if got := cave.lightLevelWithSkyFilter(cfg, 60, 0.5); got != 0 {
		t.Errorf("a sky-less room under a filter = %d, want 0", got)
	}
}

// Active mutators multiply; one without a skylight changes nothing; an
// outdoor-only mutator never reaches an indoor biome.
func TestMutatorSkyFilterMultipliesAndStaysOutdoors(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()
	defer SeedBiomesForTest(map[string]*BiomeInfo{
		"testfield": {BiomeId: "testfield", Name: "Field", Symbol: "."},
		"testhouse": {BiomeId: "testhouse", Name: "House", Symbol: "H", Indoor: true},
	})()
	half, most := 0.5, 0.7
	defer mutators.SeedSpecsForTest(
		mutators.MutatorSpec{MutatorId: "test-storm", OutdoorOnly: true, SkyLight: &half},
		mutators.MutatorSpec{MutatorId: "test-fog", OutdoorOnly: true, SkyLight: &most},
		mutators.MutatorSpec{MutatorId: "test-sanctuary"},
	)()

	outdoor, indoor := roomManager.rooms[1], roomManager.rooms[2]
	outdoor.Biome, indoor.Biome = "testfield", "testhouse"
	zc := GetZoneConfig("TestZone")

	if got := outdoor.mutatorSkyFilter(); got != 1 {
		t.Errorf("clear weather filter = %v, want 1", got)
	}
	zc.Mutators.Add("test-sanctuary")
	if got := outdoor.mutatorSkyFilter(); got != 1 {
		t.Errorf("a mutator with no skylight changed the filter to %v", got)
	}
	zc.Mutators.Add("test-storm")
	zc.Mutators.Add("test-fog")
	if got := outdoor.mutatorSkyFilter(); math.Abs(got-0.35) > 1e-9 {
		t.Errorf("storm and fog = %v, want 0.35", got)
	}
	if got := indoor.mutatorSkyFilter(); got != 1 {
		t.Errorf("weather reached an indoor biome: filter %v", got)
	}
}
```

In `internal/rooms/lighting_model_test.go`, delete `TestPositiveLightModBridgeKeepsLitCavesLit` and `TestNegativeLightModAttenuatesSkyNotLamp` (both superseded by `TestSkyFilterSubtractsFromTheSkyOnly`); drop the `lightscale` import if it becomes unused.

In `internal/rooms/light_terms_test.go`, delete `TestComposeLightLevelMatchesBridgeAcrossInputs` (with the bridge gone it compares two names for one call), and change `TestLightTermsReportEachTerm`'s middle to:

```go
	lit := Room{SkyLight: &open, Lamp: &lamp}
	got := lit.composeLight(cfg, 60, 0.25)
	if !got.HasLamp || got.Lamp != 40 {
		t.Errorf("lamp = (%v, %d), want (true, 40)", got.HasLamp, got.Lamp)
	}
	if got.SkyFilter != 0.25 {
		t.Errorf("SkyFilter = %v, want 0.25", got.SkyFilter)
	}
	wantSky := lightscale.Attenuate(cfg.DoublingStep, 60, 0.25)
	if math.Abs(got.Sky-wantSky) > 1e-9 {
		t.Errorf("Sky = %v, want %v (sky after a 0.25 filter)", got.Sky, wantSky)
	}
```

and its cave call to `cave.composeLight(cfg, 60, 1)`.

In `internal/lightnotice/tracker_test.go`, delete the `"LightMod bridge changing is the lamp"` case, and in the weather case replace `x.OcclusionSteps = 1` with `x.SkyFilter = 0.5` (read the test's `base` terms: if `base` leaves `SkyFilter` at 0, set `SkyFilter: 1` in `base` so the fixture reads as clear weather).

- [ ] **Step 2: Delete the field.** Remove `LightMod` from `MutatorSpec`. Run `go build ./... 2>&1 | head -40` and `go vet ./internal/rooms/ ./internal/lightnotice/` to list every consumer; they should be exactly those in facts 1 to 5. Report any other.

- [ ] **Step 3: Implement `internal/rooms/lighting.go`.** Replace `lightLevel` through `mutatorLightTerms` (keep `skyLightFraction`, `lampValue` and everything above `lightLevel` except the `LightLevel` doc comment's bridge paragraph, see below) with:

```go
// lightLevel is LightLevel with its two reads injected, so it is testable
// without global state and so a caller holding both can avoid reading twice.
func (r *Room) lightLevel(cfg configs.Lighting, celestial float64) int {
	return r.lightLevelWithSkyFilter(cfg, celestial, r.mutatorSkyFilter())
}

// lightLevelWithSkyFilter is the composition itself, with the active mutators'
// sky filter already multiplied out, so tests can drive it directly.
func (r *Room) lightLevelWithSkyFilter(cfg configs.Lighting, celestial, skyFilter float64) int {
	return r.composeLight(cfg, celestial, skyFilter).Level
}

// LightTerms is the room's light broken into the terms LightLevel combines,
// for a caller that needs to know WHY the light is what it is.
// internal/lightnotice names the cause of a band change from them.
type LightTerms struct {
	// Level is exactly LightLevel(): both come from composeLight.
	Level int
	// Sky is the sky term after the sky fraction and the weather filter, in
	// light-scale units; lightscale.Absent() when the room has no sky.
	Sky float64
	// SkyFilter is the fraction of the sky active weather lets through: the
	// product of the active mutators' skylight values, 1 when clear.
	SkyFilter float64
	// Lamp is the room's own lamp; 0 when HasLamp is false.
	Lamp    int
	HasLamp bool
	// Carried reports that someone in the room carries a light.
	Carried bool
}

// LightTerms reports the terms behind LightLevel, from the same single
// computation.
func (r *Room) LightTerms() LightTerms {
	return r.composeLight(configs.GetLightingConfig(), gametime.CelestialLight(), r.mutatorSkyFilter())
}

// composeLight is the one computation behind LightLevel and LightTerms.
func (r *Room) composeLight(cfg configs.Lighting, celestial, skyFilter float64) LightTerms {
	step := cfg.DoublingStep
	if !(step > 0) {
		step = 1
	}

	out := LightTerms{SkyFilter: skyFilter}
	terms := make([]float64, 0, 3)

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

	// 3. Anyone carrying a light. Plan 5 gives carried sources real magnitudes
	// that scale from stat and skill; until then any light source lifts the
	// room to the bottom of the perfect band, which is what the old model's
	// "someone has light, cancel the darkness" rule effectively did.
	if len(r.GetMobs(FindHasLight)) > 0 || len(r.GetPlayers(FindHasLight)) > 0 {
		out.Carried = true
		terms = append(terms, float64(cfg.DimBelow))
	}

	v := lightscale.Combine(step, terms...)
	if math.IsInf(v, -1) {
		// No light of any kind. Zero is the darkest light that NATURALLY
		// occurs, which is what an unlit cave is. Magical darkness goes below
		// this and arrives in plan 5.
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

// mutatorSkyFilter multiplies the skylight fraction of every active mutator
// that declares one; 1 means nothing filters the sky. Outdoor-only mutators
// never reach an indoor biome (ActiveMutators skips them), so weather stays
// out of roofed rooms.
func (r *Room) mutatorSkyFilter() float64 {
	filter := 1.0
	for mut := range r.ActiveMutators {
		if spec := mut.GetSpec(); spec != nil && spec.SkyLight != nil {
			filter *= *spec.SkyLight
		}
	}
	return filter
}
```

In `LightLevel`'s doc comment, replace the paragraph beginning "Weather and mutators attenuate the SKY only" with:

```go
// Weather attenuates the SKY only, through each active mutator's skylight
// fraction: a blizzard does not dim a lantern.
```

and change "Three terms compose it" list item 2/3 wording only if it now names the bridge (it does not today; check). If `lightscale` or `math` imports go unused, the compiler says so.

- [ ] **Step 4: `internal/lightnotice`.** In `tracker.go` `attribute`, replace the lamp and weather cases with:

```go
	case a.HasLamp != b.HasLamp || a.Lamp != b.Lamp:
		return CauseLamp
	case a.SkyFilter != b.SkyFilter:
		return CauseWeather
```

In `store.go` change the `CauseLamp` comment to `// the room's own lamp` and `CauseWeather`'s to `// weather filtering the sky`. Grep the package's Go comments for `LightMod`, `bridge` and `occlusion` and correct any that describe the old terms.

- [ ] **Step 5: Admin template.** In `_datafiles/html/admin/mutators/mutator.data.html` replace the whole `<div class="form-group col-sm">` block that contains the `lightmod` select (label "Adjust Light") with:

```html
        <div class="form-group col-sm">
            <label for="skylight">Sky Light</label>
            <input class="form-control form-control-sm" type="text" id="skylight" readonly value="{{ if $mutator.SkyLight }}{{ $mutator.SkyLight }}{{ else }}unchanged{{ end }}" aria-describedby="skylight-help">
            <small id="skylight-help" class="form-text text-muted">Fraction of the sky's light this mutator lets through (1 changes nothing, 0.5 is one step darker).</small>
        </div>
```

Then `grep -rn -i "lightmod" _datafiles/html internal/web` must print nothing. In `internal/web/admin_condition_templates_test.go`, add a sibling test:

```go
// TestAdminMutatorTemplateShowsSkyLight pins the sky-light display that
// replaced the -2..2 LightMod select: the template reaches the field by
// reflection, which the compiler cannot check.
func TestAdminMutatorTemplateShowsSkyLight(t *testing.T) {
	tmpl, err := template.New(`mutator.data.html`).Funcs(funcMap).ParseFiles(filepath.Join(adminHtmlDir(t), `mutators`, `mutator.data.html`))
	require.NoError(t, err)

	half := 0.5
	for _, c := range []struct {
		spec mutators.MutatorSpec
		want string
	}{
		{mutators.MutatorSpec{MutatorId: `probe-storm`, SkyLight: &half}, `value="0.5"`},
		{mutators.MutatorSpec{MutatorId: `probe-clear`}, `value="unchanged"`},
	} {
		tplData := map[string]any{
			`mutatorSpec`:    c.spec,
			`conditionSpecs`: []conditions.ConditionSpec{},
			`colorPatterns`:  colorpatterns.GetColorPatternNames(),
		}
		var out bytes.Buffer
		require.NoError(t, tmpl.Execute(&out, tplData))
		require.Contains(t, out.String(), c.want)
	}
}
```

- [ ] **Step 6: Comments.** `internal/usercommands/look_exit_visibility_test.go:39` names `lightLevelWithMutatorBridge, term 4`: it is now `composeLight`, term 3 (carried). `lighting_parity_golden_test.go:91` quotes `spec.LightMod != 0`: reword to name `mutatorSkyFilter`'s nil guard on `spec`. Read each surrounding comment and keep its point.

- [ ] **Step 7: Run.** `go build ./...`; `go test ./internal/mutators/ ./internal/rooms/ ./internal/lightnotice/ ./internal/web/ ./internal/usercommands/ ./internal/hooks/ ./internal/narration/ . -count=1` (no `-race`: no C compiler here). Every lighting golden must be unchanged; if one moves, STOP and report which rows.

- [ ] **Step 8: Prove the filter test can fail:** temporarily make `mutatorSkyFilter` return `1.0` unconditionally, confirm `TestMutatorSkyFilterMultipliesAndStaysOutdoors` FAILS, restore.

- [ ] **Step 9: Commit** every file above by name, message:

```
refactor(lighting): a sky filter replaces the LightMod bridge

MutatorSpec.LightMod is deleted. Weather now reaches the sky only through
each active mutator's skylight fraction, multiplied; the positive bridge
term is gone, so the combine is sky, lamp and carried light. LightTerms
carries SkyFilter in place of OcclusionSteps and LightMod, and the admin
mutator page shows it (the template reads the field by reflection).
```

---

### Task 4: Retire `lightmod` from the data

**Files:** `_datafiles/world/dogmud/mutators/weather_{storm,dust,blizzard}.yaml`, delete `_datafiles/world/dogmud/mutators/bioluminescent_caves.yaml`, `internal/mutators/shipped_sky_light_test.go`.

- [ ] **Step 1: Failing guard.** Append to `internal/mutators/shipped_sky_light_test.go` (add `os`, `path/filepath`, `regexp` imports):

```go
// TestNoShippedMutatorCarriesLightmod exists because the loader is non-strict:
// a stale lightmod key would be silently ignored, and an author would believe
// it still did something.
func TestNoShippedMutatorCarriesLightmod(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(shippedMutatorDir, "*.yaml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("glob found no mutator files (%v); the guard cannot run", err)
	}
	key := regexp.MustCompile(`(?m)^\s*lightmod\s*:`)
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if key.Match(src) {
			t.Errorf("%s carries lightmod, which nothing reads; use skylight", filepath.Base(f))
		}
	}
}
```

- [ ] **Step 2:** It fails, naming storm, dust, blizzard and bioluminescent_caves.

- [ ] **Step 3:** Remove the `lightmod: -1` line from the three weather files. Delete `bioluminescent_caves.yaml` with `git rm`. Re-grep `bioluminescent-caves` across `_datafiles`, `internal`, `modules` to confirm nothing references it.

- [ ] **Step 4:** `go test ./internal/mutators/ -count=1` passes; then the full suite (`go test ./... -count=1`, run in the background, allow ten minutes). If a test counts shipped mutators, update the count and say so in the report.

- [ ] **Step 5: Commit** the three YAML files, the deletion and the test, message `content(mutators): lightmod retired from the data, bioluminescent_caves deleted`.

---

### Task 5: Shipped-data behaviour under weather

**Files:** create `internal/rooms/weather_occlusion_test.go`.

- [ ] **Step 1: Write the tests.**

```go
package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mutators"
)

// openSkyUnder loads the shipped biomes, clock and mutators and returns an
// open-sky plains room whose zone carries the given weather (none for clear).
func openSkyUnder(t *testing.T, weather ...string) *Room {
	t.Helper()
	withShippedBiomesAndClock(t)
	t.Cleanup(mutators.SeedSpecsForTest())
	mutators.LoadDataFiles()
	t.Cleanup(seedRegistry())
	requireBiome(t, "plains")

	room := roomManager.rooms[1]
	room.Biome, room.SkyLight, room.Lamp = "plains", nil, nil
	zc := GetZoneConfig(room.Zone)
	zc.Mutators = mutators.MutatorList{}
	for _, w := range weather {
		if !zc.Mutators.Add(w) {
			t.Fatalf("weather %q is not a shipped mutator", w)
		}
	}
	return room
}

// setWeather swaps the room's zone weather in place.
func setWeather(t *testing.T, room *Room, weather ...string) {
	t.Helper()
	zc := GetZoneConfig(room.Zone)
	zc.Mutators = mutators.MutatorList{}
	for _, w := range weather {
		if !zc.Mutators.Add(w) {
			t.Fatalf("weather %q is not a shipped mutator", w)
		}
	}
}

var sampleDays = []int{172, 81, 356}

// brightestMidnight is the day, within 30 of doy, whose midnight sky is the
// brightest: the fullest moons.
func brightestMidnight(doy int) int {
	best, bestV := doy, -1e9
	for d := doy - 30; d <= doy+30; d++ {
		setClock(d, 0)
		if v := gametime.CelestialLight(); v > bestV {
			best, bestV = d, v
		}
	}
	return best
}

// Light cloud must leave the brightest moonlit night readable as shapes.
func TestFogLeavesAMoonlitNightReadable(t *testing.T) {
	room := openSkyUnder(t)
	blind := configs.GetLightingConfig().BlindBelow
	for _, doy := range sampleDays {
		d := brightestMidnight(doy)
		setClock(d, 0)
		setWeather(t, room)
		clear := room.LightLevel()
		setWeather(t, room, "weather-fog")
		foggy := room.LightLevel()
		if foggy < blind {
			t.Errorf("day %d midnight under fog = %d, below BlindBelow %d (clear %d)", d, foggy, blind, clear)
		}
		if drop := clear - foggy; drop < 3 || drop > 5 {
			t.Errorf("day %d midnight: fog took %d points, want about 4", d, drop)
		}
	}
}

// Heavy weather never hides faces at noon, at any season.
func TestStormNeverHidesFacesAtNoon(t *testing.T) {
	room := openSkyUnder(t, "weather-storm")
	dim := configs.GetLightingConfig().DimBelow
	for _, doy := range sampleDays {
		setClock(doy, 12)
		if got := room.LightLevel(); got < dim {
			t.Errorf("day %d noon under a storm = %d, below DimBelow %d", doy, got, dim)
		}
	}
}

// Heavy weather takes one full step off every night, so a night readable in
// clear weather goes blind under a storm unless the moons are near their
// brightest.
func TestStormTakesAStepOffTheNight(t *testing.T) {
	room := openSkyUnder(t)
	blind := configs.GetLightingConfig().BlindBelow
	flipped := false
	for _, doy := range sampleDays {
		for d := doy - 30; d <= doy+30; d++ {
			setClock(d, 0)
			setWeather(t, room)
			clear := room.LightLevel()
			setWeather(t, room, "weather-storm")
			stormy := room.LightLevel()
			if drop := clear - stormy; clear > 8 && (drop < 7 || drop > 9) {
				t.Errorf("day %d midnight: storm took %d points, want 8", d, drop)
			}
			if clear >= blind && stormy < blind {
				flipped = true
			}
		}
	}
	if !flipped {
		t.Error("no sampled night went from readable to blind under a storm")
	}
}
```

Before running, read `seedRegistry` in the rooms tests and confirm it does not reset the biome registry `withShippedBiomesAndClock` loaded, and that room 1's zone has a zone config. If `seedRegistry` does reset biomes, call `withShippedBiomesAndClock` after it instead, and say so.

- [ ] **Step 2:** `go test ./internal/rooms/ -run 'TestFog|TestStorm' -count=1 -v` passes. Log the measured levels with `t.Logf` in each test so the report shows real numbers.

- [ ] **Step 3: Prove each can fail:** set `weather_fog.yaml`'s skylight to `0.3` temporarily and confirm `TestFogLeavesAMoonlitNightReadable` fails; set the storm's to `0.25` and confirm `TestStormNeverHidesFacesAtNoon` fails; restore both (the YAML diff must be empty).

- [ ] **Step 4: Commit** `internal/rooms/weather_occlusion_test.go`, message `test(rooms): what weather does to shipped noons and nights`.

---

### Task 6: Documentation

- [ ] **Step 1:** Extract the real surface first: `grep -nE '^(func|type|const|var)\s' internal/rooms/lighting.go internal/mutators/mutators.go` and read the changed code.
- [ ] **Step 2:** `internal/mutators/context.md`: replace the `LightMod int // -2..2` line with `SkyLight *float64` and a sentence on meaning, multiplication and validation.
- [ ] **Step 3:** `internal/rooms/context.md`: remove the bridge as a term (line ~65) and renumber; describe the sky filter and `mutatorSkyFilter`; `LightTerms` fields (line ~82) now `SkyFilter`; note weather stays out of indoor biomes and that `fort` and `interior` are the only indoor biomes with sky.
- [ ] **Step 4:** `internal/lightnotice/context.md`: lamp cause is the lamp only; weather cause is the sky filter changing.
- [ ] **Step 5:** `_datafiles/world/dogmud/narration/light-notices/lamp.yaml` header: no mutator reaches the lamp lines any more; they wait for runtime lamps (plan 5). Run `go test ./internal/lightnotice/ ./internal/narration/ -count=1` (the golden reads the lines, not comments, so it must not move).
- [ ] **Step 6:** `docs/README.md`: add this plan's row after the plan 4 spec row, in the style of neighbouring plan rows.
- [ ] **Step 7:** `docs/PATCH_NOTES.md`: a dated entry at the top, player-facing, no numbers, no dashes: fog, cloud, rain and snow now dim the sky a little and storms, dust and blizzards a lot; a moonlit night under heavy weather is too dark to see without a light; daylight under a storm stays bright enough to read faces; weather never dims a lamp, a torch, or the inside of a building.
- [ ] **Step 8:** `python tools/context_md_audit.py`: no finding for mutators, rooms or lightnotice.
- [ ] **Step 9: Commit** all of the above by name, message `docs(lighting): plan 4, the sky filter`.

---

### Task 7: Verification and ship

Load `dogmud-shipping` first and follow its gate order.

- [ ] **Step 1:** `go test ./... -count=1` (background, up to ten minutes). Report failures verbatim.
- [ ] **Step 2:** `git diff --stat master -- '*.golden'` prints nothing.
- [ ] **Step 3:** `gofmt -l` on every changed Go file; `~/go/bin/golangci-lint run --new-from-merge-base=origin/master` reports `0 issues`.
- [ ] **Step 4:** Boot check in a detached worktree with CONFIG_PATH port overrides and `Modules.aicompanion.Enabled: true` (as plan 3d did): `Server Ready`, zero panics, `mutators.LoadDataFiles()` loaded count one lower than master's. Kill only that PID; remove the worktree.
- [ ] **Step 5:** Push, `gh pr create --repo pruuk/DOGMud`, watch checks. Merge only on the owner's word.
