# M3 item 9 PR 1: weather emotes join the narration core

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the weather emote store's two hand-rolled picks onto
`narration.Render`, with a byte-identical golden proving nothing a player reads
changed.

**Architecture:** Weather is actorless, so `narration.Variants{Observer: lines}`
is populated and the other three roles stay empty. `modules/weather/content`
becomes the first `modules/` package to import `internal/narration`, which is
cycle-free because that package imports stdlib only. No schema change, no
content change, no change to `engine.EmitAmbient`.

**Tech Stack:** Go, `gopkg.in/yaml.v2`, the `internal/narration` snapshot
harness (`package narration_test`, external, so importing the weather module
back is also cycle-free).

Spec: [`docs/superpowers/specs/2026-09-16-messaging-m3-item9-weather-emotes-design.md`](../specs/2026-09-16-messaging-m3-item9-weather-emotes-design.md)

---

## Deviation from the spec's PR split, and why

The spec put the three-way outdoors/indoors/underground split in PR 1 and the
prose in PR 2. **Implementation moves the split into PR 2, with the content.**

The reason is a silence window. If `cave` is reclassified as underground in PR 1
while the `underground:` section is still empty, `bandedSectionLines` correctly
refuses to fall back to indoor (silence beats wrong prose), so **124 of the 161
indoor rooms would get no weather line at all** until PR 2 landed. The section
and the prose that fills it are inseparable.

PR 1 is therefore a pure refactor with a byte-identical golden, which is what
M3 promises everywhere else. PR 2 owns the schema, the maps, the guards and the
prose, and is the one PR where the golden legitimately changes.

---

## File Structure

| File | Responsibility | Change |
|---|---|---|
| `internal/narration/snapshot_test.go` | The M1 golden harness | Modify: add `buildWeatherEmotesGolden`, two sorted-key helpers, one `t.Run` |
| `internal/narration/testdata/stores/weather_emotes.golden` | Frozen store output | **Create** (via `-update`) |
| `modules/weather/content/emotes.go` | The store: schema, load, pick | Modify: two `Pick` bodies only |
| `modules/weather/content/emotes_test.go` | Store unit tests | Modify: picker-type test |
| `modules/weather/content/context.md` | Package doc | Modify: record the core join |

---

### Task 0: Record `weather_emotes.golden` from pre-migration code

**This task must be completed and committed before any production code changes.**
A golden recorded after the refactor bakes the refactor's bugs into the
baseline invisibly. This is the arc's standing rule and it has been earned
twice.

**Files:**
- Modify: `internal/narration/snapshot_test.go`
- Create: `internal/narration/testdata/stores/weather_emotes.golden`

- [ ] **Step 1: Add the two sorted-key helpers**

The file already has `sortedKeysStrSlice(m map[string][]string) []string` at
`snapshot_test.go:577`, which covers `Outdoor`. Add two more beside it for the
other two map types. Place them immediately after `sortedKeysStrSlice`.

```go
func sortedKeysIndoorPool(m map[string]content.IndoorPool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedKeysTableSection(m map[string]content.TableSection) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
```

- [ ] **Step 2: Add the imports**

Add to the import block of `internal/narration/snapshot_test.go`:

```go
	"github.com/GoMudEngine/GoMud/modules/weather/content"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
```

`snapshot_test.go` is `package narration_test` (an external test package), so
importing the weather module, which will itself import `internal/narration` in
Task 2, does **not** create a cycle. Verify this is still true before
proceeding: the first line of the file must read `package narration_test`.

- [ ] **Step 3: Write the golden builder**

Add at the end of `internal/narration/snapshot_test.go`:

```go
// ---------------------------------------------------------------------
// Store 13: weather emotes (modules/weather/content)
//
// The ONLY actorless store in the arc: an ambient line has no Actor, so it is
// rendered with narration.Variants{Observer: lines} and the other three roles
// stay empty. Dimensions: weather type x section(outdoor/sheltered) x biome
// (authored keys UNION a representative set) x band(mild/strong) x season(base
// + each authored variant), then the seasonal-ambience tables by (track,
// season).
//
// The sheltered axis is named for the ROOM, not for the section it resolves
// to, because item 9 PR 2 splits that one section into indoor and underground
// and the row keys must survive it.
//
// Recorded 2026-09-16 from PRE-migration code. Weather already had a picker
// seam (Pick takes `roll func(int) int`, and narration.Picker has that exact
// underlying type, so SequencePicker is assignable with no production change),
// which is why this baseline needed no `*With` variant the way itemvoices did.
//
// Indoor bands are forced by the felt value, not by naming a band: felt 0.0 is
// below content.StrongFeltThreshold (0.5) and selects Mild; felt 1.0 is at or
// above it and selects Strong. An empty Mild pool rendering "" is DELIBERATE
// (light weather is inaudible through walls) and that emptiness is frozen here
// too, so a pool silently disappearing shows as a row changing from text to "".
func buildWeatherEmotesGolden(t *testing.T) string {
	t.Helper()
	root := os.DirFS(dogmudDataDir(t))

	tables, err := content.LoadEmotes(root, "weather/emotes")
	if err != nil {
		t.Fatalf("LoadEmotes: %v", err)
	}
	if len(tables) == 0 {
		t.Fatal("no weather emote tables loaded; the golden would be vacuous")
	}
	seasonal, err := content.LoadSeasonalEmotes(root, "weather/emotes/seasons")
	if err != nil {
		t.Fatalf("LoadSeasonalEmotes: %v", err)
	}
	if len(seasonal) == 0 {
		t.Fatal("no seasonal ambience tables loaded; the golden would be vacuous")
	}

	bands := []struct {
		name string
		felt float64
	}{{"mild", 0.0}, {"strong", 1.0}}

	// Representative biomes, swept IN ADDITION to the authored keys.
	//
	// 🔑 THIS IS WHAT MAKES THE GOLDEN ABLE TO SEE PR 2. The authored indoor
	// keys are "default" only, so sweeping authored keys alone would never
	// exercise a cave, and the three-way split would land with no diff to
	// inspect. Today all five of these resolve to indoor["default"]; after PR 2
	// classifies them, cave and dungeon must move to the underground section
	// and the other three must not, which shows up here as a diff on exactly
	// those rows.
	repBiomes := []string{"cave", "dungeon", "house", "fort", "spiderweb", "forest"}

	// union merges the authored keys with the representative set, de-duplicated
	// and sorted, so every row is stable across runs.
	union := func(authored []string) []string {
		seen := map[string]bool{}
		out := []string{}
		for _, k := range append(append([]string{}, authored...), repBiomes...) {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
		sort.Strings(out)
		return out
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# weather emotes store snapshot\n")
	fmt.Fprintf(&b, "# weather tables: %d   seasonal-ambience tables: %d\n", len(tables), len(seasonal))
	fmt.Fprintf(&b, "# dimensions: type x section x biome x band x season, then (track,season) ambience.\n")
	fmt.Fprintf(&b, "# Single role (the room is told), no tokens authored anywhere in this store.\n")
	fmt.Fprintf(&b, "# A fresh SequencePicker per row pins index 0.\n")
	fmt.Fprintf(&b, "# An empty row (\"\") in a mild band is deliberate silence, not a missing pool.\n\n")

	types := make([]string, 0, len(tables))
	for wt := range tables {
		types = append(types, string(wt))
	}
	sort.Strings(types)

	for _, wt := range types {
		w := sim.WeatherType(wt)
		tbl := tables[w]

		for _, biome := range sortedKeysStrSlice(tbl.Outdoor) {
			fmt.Fprintf(&b, "%s|base|outdoor|%s => %q\n", wt, biome,
				tables.Pick(w, biome, false, 0, "", narration.SequencePicker()))
		}
		// "sheltered" rather than "indoor": after PR 2 this axis covers two
		// prose classes, and the row key must not have to be renamed then.
		for _, biome := range union(sortedKeysIndoorPool(tbl.Indoor)) {
			for _, bd := range bands {
				fmt.Fprintf(&b, "%s|base|sheltered|%s|%s => %q\n", wt, biome, bd.name,
					tables.Pick(w, biome, true, bd.felt, "", narration.SequencePicker()))
			}
		}
		for _, season := range sortedKeysTableSection(tbl.Seasonal) {
			sec := tbl.Seasonal[season]
			for _, biome := range sortedKeysStrSlice(sec.Outdoor) {
				fmt.Fprintf(&b, "%s|season:%s|outdoor|%s => %q\n", wt, season, biome,
					tables.Pick(w, biome, false, 0, season, narration.SequencePicker()))
			}
			for _, biome := range union(sortedKeysIndoorPool(sec.Indoor)) {
				for _, bd := range bands {
					fmt.Fprintf(&b, "%s|season:%s|sheltered|%s|%s => %q\n", wt, season, biome, bd.name,
						tables.Pick(w, biome, true, bd.felt, season, narration.SequencePicker()))
				}
			}
		}
	}

	// Seasonal ambience: the persistent voice of a season in CALM weather.
	fmt.Fprintf(&b, "\n# seasonal ambience tables, keyed (track, season)\n")
	keys := make([]string, 0, len(seasonal))
	index := map[string]content.SeasonalKey{}
	for k := range seasonal {
		flat := k.Track + "/" + k.Season
		keys = append(keys, flat)
		index[flat] = k
	}
	sort.Strings(keys)
	for _, flat := range keys {
		k := index[flat]
		sec := seasonal[k]
		for _, biome := range sortedKeysStrSlice(sec.Outdoor) {
			fmt.Fprintf(&b, "%s|%s|outdoor|%s => %q\n", k.Track, k.Season, biome,
				seasonal.Pick(k.Track, k.Season, biome, false, 0, narration.SequencePicker()))
		}
		for _, biome := range union(sortedKeysIndoorPool(sec.Indoor)) {
			for _, bd := range bands {
				fmt.Fprintf(&b, "%s|%s|sheltered|%s|%s => %q\n", k.Track, k.Season, biome, bd.name,
					seasonal.Pick(k.Track, k.Season, biome, true, bd.felt, narration.SequencePicker()))
			}
		}
	}

	// EMPTY CASES, frozen deliberately.
	fmt.Fprintf(&b, "\n# EMPTY CASE: unknown weather type -> \"\"\n")
	fmt.Fprintf(&b, "bogus-weather|base|outdoor|default => %q\n",
		tables.Pick(sim.WeatherType("bogus-weather"), "default", false, 0, "", narration.SequencePicker()))
	fmt.Fprintf(&b, "\n# EMPTY CASE: unknown (track,season) ambience -> \"\"\n")
	fmt.Fprintf(&b, "bogus-track|bogus-season|outdoor|default => %q\n",
		seasonal.Pick("bogus-track", "bogus-season", "default", false, 0, narration.SequencePicker()))

	return b.String()
}
```

- [ ] **Step 4: Register it in the dispatch**

In `TestSnapshotStores` (`snapshot_test.go:785`), add after the `tips` subtest
and before `post_pipeline`:

```go
	t.Run("weather_emotes", func(t *testing.T) {
		checkGolden(t, "weather_emotes.golden", buildWeatherEmotesGolden(t))
	})
```

- [ ] **Step 5: Run it and watch it fail for the right reason**

Run: `go test ./internal/narration/ -run 'TestSnapshotStores/weather_emotes' -v`

Expected: FAIL, because `testdata/stores/weather_emotes.golden` does not exist.
The message must name the missing golden file. If it fails for any other
reason (a compile error, an empty table set, an import cycle), fix that first
and do not proceed.

- [ ] **Step 6: Record the golden**

Run: `go test ./internal/narration/ -run 'TestSnapshotStores/weather_emotes' -update`

Then read the file:

Run: `head -40 internal/narration/testdata/stores/weather_emotes.golden`

Expected: the header comment block, then rows such as
`blizzard|base|indoor|default|mild => ""` and
`blizzard|base|outdoor|default => "..."`.

- [ ] **Step 7: Verify the golden is not vacuous**

Run: `grep -c '=>' internal/narration/testdata/stores/weather_emotes.golden`

Expected: a count comfortably over 200. There are 9 weather types, 6 seasonal
files, 6 authored biome keys, 6 representative biomes and 2 bands, so a count
under 200 means a dimension is not being swept and the golden is not protecting
what it claims to. Investigate before continuing.

Confirm specifically that the representative biomes are present, since they are
what lets this golden see PR 2 at all:

Run: `grep -c 'sheltered|cave' internal/narration/testdata/stores/weather_emotes.golden`

Expected: 18 (9 weather types x 2 bands). A zero here means `union` is not
wired in and the golden is blind to the change it exists to catch.

Run: `grep -c '=> ""' internal/narration/testdata/stores/weather_emotes.golden`

Expected: a non-zero count, all of them `mild` rows and the two bogus rows.
Confirm no `outdoor` row is empty:

Run: `grep 'outdoor' internal/narration/testdata/stores/weather_emotes.golden | grep '=> ""'`

Expected: only the `bogus-weather` row. Any other empty outdoor row is a real
content hole and must be reported, not silently frozen.

Note: `grep -c` exits 1 when it finds zero matches, so run these checks
standalone rather than in an `&&` chain.

- [ ] **Step 8: Commit**

```bash
git add internal/narration/snapshot_test.go internal/narration/testdata/stores/weather_emotes.golden
git commit -m "test(narration): golden for the weather emote store, pre-migration

Recorded from today's Tables.Pick and SeasonalTables.Pick before the core
join, so it is a baseline of existing behaviour rather than a record of
whatever the migration produces.

Weather already had a picker seam: Pick takes roll func(int) int, which is
narration.Picker's exact underlying type, so SequencePicker is assignable
with no production change. itemvoices needed a LineWith variant added first;
this store did not.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 1: Prove the golden is capable of failing

A green golden proves nothing until it has been seen to go red. Three false
passes have already cost this project real work.

**Files:**
- Modify (temporarily): `modules/weather/content/emotes.go`

- [ ] **Step 1: Sabotage the pick by line number**

In `Tables.Pick` (`emotes.go:119`), change:

```go
	i := roll(len(lines))
```

to:

```go
	i := roll(len(lines))
	if len(lines) > 1 {
		i = len(lines) - 1 // SABOTAGE: last variant, not the picked one
	}
```

- [ ] **Step 2: Verify it compiles**

Run: `go vet ./modules/weather/content/`

Expected: clean. **A sabotage that does not compile proves nothing.** If vet
complains, fix the sabotage until it is valid Go, then continue.

- [ ] **Step 3: Verify the golden goes red**

Run: `go test ./internal/narration/ -run 'TestSnapshotStores/weather_emotes' -v`

Expected: FAIL, with a diff naming a row whose text changed from the first
authored variant to the last. Confirm the named row is a real `outdoor` or
`indoor|strong` row with more than one authored line.

- [ ] **Step 4: Revert the sabotage**

```bash
git checkout -- modules/weather/content/emotes.go
```

This file has no uncommitted work at this point, so `git checkout --` is safe
here. **Never** use it to undo a sabotage in a file that also holds
uncommitted work: it restores from HEAD and discards the work.

- [ ] **Step 5: Confirm green again**

Run: `go test ./internal/narration/ -run 'TestSnapshotStores/weather_emotes' -v`

Expected: PASS. Nothing to commit; this task changes no tracked file.

---

### Task 2: Join the narration core

**Files:**
- Modify: `modules/weather/content/emotes.go` (both `Pick` bodies, imports, doc comments)
- Modify: `modules/weather/content/emotes_test.go` (picker type)

- [ ] **Step 1: Write the failing test**

Add to `modules/weather/content/emotes_test.go`:

```go
// The store renders through the shared narration core, actorlessly: the
// Observer role carries the line and the other three roles stay empty. This
// test pins the seam, not the prose, so it must keep passing when PR 2
// rewrites the content.
func TestPickRendersThroughTheNarrationCore(t *testing.T) {
	tables := Tables{
		"rain": {
			Weather: "rain",
			Outdoor: map[string][]string{
				"default": {"first line", "second line", "third line"},
			},
		},
	}

	// FirstPicker always returns 0, so the first authored variant must come
	// back. If the store still rolled its own index this would be flaky
	// rather than exact.
	got := tables.Pick("rain", "default", false, 0, "", narration.FirstPicker)
	if got != "first line" {
		t.Fatalf("FirstPicker should select variant 0, got %q", got)
	}

	// A picker that walks the pool proves the index reaches the core rather
	// than being discarded.
	seq := narration.SequencePicker()
	want := []string{"first line", "second line", "third line"}
	for i, w := range want {
		if got := tables.Pick("rain", "default", false, 0, "", seq); got != w {
			t.Fatalf("call %d: want %q, got %q", i, w, got)
		}
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./modules/weather/content/ -run TestPickRendersThroughTheNarrationCore -v`

Expected: FAIL to compile, with `undefined: narration`. That is the correct
failure: the package does not import the core yet.

- [ ] **Step 3: Change both Pick signatures and bodies**

In `modules/weather/content/emotes.go`, add the import:

```go
	"github.com/GoMudEngine/GoMud/internal/narration"
```

Change `Tables.Pick` (`emotes.go:100`). The signature's last parameter becomes
a `narration.Picker`, and the tail of the body becomes a `Render` call:

```go
func (ts Tables) Pick(weather sim.WeatherType, biome string, indoor bool, felt float64, season string, pick narration.Picker) string {
	t, ok := ts[weather]
	if !ok {
		return ""
	}

	var lines []string
	if season != "" {
		if v, ok := t.Seasonal[season]; ok {
			lines = bandedSectionLines(v.Outdoor, v.Indoor, biome, indoor, felt)
		}
	}
	if len(lines) == 0 {
		lines = bandedSectionLines(t.Outdoor, t.Indoor, biome, indoor, felt)
	}

	return renderAmbient(lines, pick)
}
```

Change `SeasonalTables.Pick` (`emotes.go:196`) the same way:

```go
func (st SeasonalTables) Pick(track, season, biome string, indoor bool, felt float64, pick narration.Picker) string {
	sec, ok := st[SeasonalKey{track, season}]
	if !ok {
		return ""
	}
	return renderAmbient(bandedSectionLines(sec.Outdoor, sec.Indoor, biome, indoor, felt), pick)
}
```

Add the shared helper beside them, so the actorless shape is stated once:

```go
// renderAmbient renders one ambient line through the shared narration core.
//
// Weather is the arc's only ACTORLESS store: an ambient line has no Actor and
// no Actee, so only Observer is populated and Render's coordination across
// roles is a no-op here. The core is still the right home, because it owns the
// picker seam (which is what makes the golden possible) and token
// substitution, which this store's content does not use today but can.
//
// An empty pool renders "" rather than a fallback. That is deliberate at every
// layer of this store: silence beats wrong prose.
func renderAmbient(lines []string, pick narration.Picker) string {
	if len(lines) == 0 {
		return ""
	}
	return narration.Render(narration.Variants{Observer: lines}, nil, pick).Observer
}
```

Delete the now-unreachable clamp that `Pick` used to carry (`i < 0 || i >= len(lines)`);
`narration.Render` owns index safety.

- [ ] **Step 4: Update the two call sites' parameter names only**

In `modules/weather/engine/emotes.go`, the calls at `:79` and `:90` pass
`roll`, which is declared in `EmitAmbient`'s signature as `roll func(int) int`.
`narration.Picker` has that exact underlying type and `func(int) int` is
unnamed, so the value is assignable and **no change is required**. Confirm by
compiling rather than by editing:

Run: `go build ./modules/weather/...`

Expected: success, with no edit to `engine/emotes.go`.

- [ ] **Step 5: Run the new test**

Run: `go test ./modules/weather/content/ -run TestPickRendersThroughTheNarrationCore -v`

Expected: PASS.

- [ ] **Step 6: Run the whole weather module**

Run: `go test ./modules/weather/...`

Expected: PASS. `emotes_test.go` and `shipped_emotes_test.go` both exercise
`Pick`; any existing call passing a bare `func(int) int` literal still
compiles for the same assignability reason.

- [ ] **Step 7: The golden must be byte-identical**

Run: `go test ./internal/narration/ -run 'TestSnapshotStores/weather_emotes' -v`

Expected: PASS, with **no** `-update`. This is the proof that the refactor
changed nothing a player reads.

If it goes red, do not run `-update`. A diff here means the migration changed
behaviour, which is the one thing this PR promises it does not do. Read the
named row and find out why.

- [ ] **Step 8: Confirm the whole harness is still green**

Run: `go test ./internal/narration/`

Expected: PASS, all 14 goldens.

- [ ] **Step 9: Commit**

```bash
git add modules/weather/content/emotes.go modules/weather/content/emotes_test.go
git commit -m "refactor(weather): render ambient emotes through the narration core

Both Pick methods now render through narration.Render with Observer alone.
Weather is the arc's only actorless store: an ambient line has no Actor, so
the other three roles stay empty rather than being given a fake subject.

modules/weather/content is the first modules/ package to import
internal/narration. That package imports stdlib only, so there is no cycle,
and the snapshot harness is package narration_test, so it can import the
weather module back.

engine.EmitAmbient is unchanged: narration.Picker and roll func(int) int
share an underlying type, so the existing argument is assignable as-is.

weather_emotes.golden is byte-identical.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Documentation

**Files:**
- Modify: `modules/weather/content/context.md`

- [ ] **Step 1: Update the package context**

The project rule is that any work reshaping a package's API must update its
`context.md`. `Pick`'s signature changed on both types, so this is required.

Record: both `Pick` methods take a `narration.Picker` rather than a bare
`roll func(int) int`; rendering goes through `narration.Render` with `Observer`
alone because weather is actorless; the store is frozen by
`internal/narration/testdata/stores/weather_emotes.golden`; and an empty pool
renders `""` deliberately at every layer.

**Verify before you document.** Every symbol named in the file must exist.
Check with:

Run: `Select-String -Path modules\weather\content\*.go -Pattern '^(func|type|const|var)\s'`

- [ ] **Step 2: Run the context.md audit**

Run: `python tools/context_md_audit.py`

Expected: no findings for `modules/weather/content`. This tool exists because
`internal/items/context.md` carried three phantom symbols for months.

- [ ] **Step 3: Commit**

```bash
git add modules/weather/content/context.md
git commit -m "docs(weather): context.md for the narration core join

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Gate and PR

- [ ] **Step 1: Format check**

Run: `gofmt -l internal/narration modules/weather`

Expected: no output. Any file listed is misformatted. This matters because
Python edits on Windows write CRLF and break `gofmt`; use the Edit tool for Go
files, or run `gofmt -w` on anything listed.

- [ ] **Step 2: Vet**

Run: `go vet ./internal/narration/... ./modules/weather/...`

Expected: clean.

- [ ] **Step 3: Full suite**

Run: `go test ./...`

Expected: PASS.

Two known false reds, neither caused by this change:
- `internal/playtestrun` can go red under full-suite load while passing
  standalone. Check `go list -deps` before hunting: if the changed packages are
  not in its graph, it is contention.
- `internal/rooms` zone lifecycle tests fail locally on Windows under
  `DOGMUD_BOOT_SMOKE=1`, on master too. CI is green.

- [ ] **Step 4: Confirm the diff is what it claims**

Run: `git diff --stat master..HEAD`

Expected: exactly five files, and **no** content YAML under `_datafiles/`.
A weather YAML in this diff means content leaked into the refactor PR.

- [ ] **Step 5: Push and open the PR**

```bash
git push -u origin feature/messaging-m3-item9-weather-emotes
gh pr create --repo pruuk/DOGMud --base master \
  --title "M3 item 9 PR 1: weather emotes join the narration core" \
  --body-file <path to a body file>
```

**Every `gh` command carries `--repo pruuk/DOGMud`.** This repo is a fork of
`GoMudEngine/GoMud` and `gh` defaults to the parent; a bare `gh pr create` once
opened a PR on upstream.

The body should state: the store is actorless and renders with `Observer`
alone; the golden was recorded pre-migration and is byte-identical; the schema
split moved to PR 2 to avoid a silence window in 124 rooms; and that `jungle`
dead content and the depth guard are PR 2's.

---

## What this PR deliberately does not do

- **No schema change.** `Underground` arrives in PR 2 with the prose that fills
  it.
- **No classification maps and no coupling guards.** They have nothing to
  classify until the schema exists.
- **No depth validator.** Shipped pools are 1 to 4 deep; wiring a minimum of 6
  before the content pass would fail every load.
- **No content.** Including the two dead `jungle` lines in
  `seasons/monsoon_wet.yaml`, which are keyed to a biome that exists in neither
  world and are fixed in PR 2.
