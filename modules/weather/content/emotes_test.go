package content

import (
	"testing"
	"testing/fstest"

	"github.com/GoMudEngine/GoMud/internal/narration"
)

const stormYAML = `weather: storm
outdoor:
  default:
    - "Thunder cracks directly overhead."
    - "A blinding fork of lightning splits the sky."
  forest:
    - "Wind tears at the branches; the whole canopy roars."
indoor:
  default:
    mild: []
    strong:
      - "Rain hammers against the windows."
`

func loadTestTables(t *testing.T) Tables {
	t.Helper()
	fsys := fstest.MapFS{"emotes/storm.yaml": {Data: []byte(stormYAML)}}
	tables, err := LoadEmotes(fsys, "emotes")
	if err != nil {
		t.Fatal(err)
	}
	return tables
}

func TestPickSelectsByBiomeAndIndoor(t *testing.T) {
	tables := loadTestTables(t)
	first := func(n int) int { return 0 }

	if got := tables.Pick("storm", "forest", false, 0.7, "", first); got != "Wind tears at the branches; the whole canopy roars." {
		t.Errorf("forest outdoor: %q", got)
	}
	if got := tables.Pick("storm", "desert", false, 0.7, "", first); got != "Thunder cracks directly overhead." {
		t.Errorf("unknown biome should fall back to default: %q", got)
	}
	if got := tables.Pick("storm", "forest", true, 0.7, "", first); got != "Rain hammers against the windows." {
		t.Errorf("indoor falls back to indoor default (never outdoor): %q", got)
	}
	if got := tables.Pick("fog", "forest", false, 0.7, "", first); got != "" {
		t.Errorf("missing table must yield silence: %q", got)
	}
}

func TestPickUsesRoll(t *testing.T) {
	tables := loadTestTables(t)
	rolled := -1
	got := tables.Pick("storm", "default", false, 0.7, "", func(n int) int { rolled = n; return 1 })
	if rolled != 2 {
		t.Errorf("roll should receive the line count, got %d", rolled)
	}
	if got != "A blinding fork of lightning splits the sky." {
		t.Errorf("roll result not honored: %q", got)
	}
}

func TestLoadEmotesRejectsMissingWeatherKey(t *testing.T) {
	fsys := fstest.MapFS{"emotes/bad.yaml": {Data: []byte("outdoor:\n  default: [\"x\"]\n")}}
	if _, err := LoadEmotes(fsys, "emotes"); err == nil {
		t.Fatal("emote table without 'weather' must be rejected")
	}
}

func TestLoadEmotesMissingDir(t *testing.T) {
	tables, err := LoadEmotes(fstest.MapFS{}, "emotes")
	if err != nil || len(tables) != 0 {
		t.Fatalf("missing dir should be empty tables, nil error: %v %v", tables, err)
	}
}

func TestPick_IndoorIntensityBands(t *testing.T) {
	tables := Tables{
		"rain": {
			Weather: "rain",
			TableSection: TableSection{
				Outdoor: map[string][]string{"default": {"out"}},
				Indoor: map[string]IndoorPool{
					"default": {Mild: nil, Strong: []string{"roof"}},
				},
			},
		},
	}
	first := func(n int) int { return 0 }

	if got := tables.Pick("rain", "city", false, 0.1, "", first); got != "out" {
		t.Errorf("outdoor mild: got %q want %q", got, "out")
	}
	if got := tables.Pick("rain", "house", true, 0.2, "", first); got != "" {
		t.Errorf("indoor mild: got %q want silence", got)
	}
	if got := tables.Pick("rain", "house", true, 0.7, "", first); got != "roof" {
		t.Errorf("indoor strong: got %q want %q", got, "roof")
	}
}

func TestPick_IndoorBiomeFallback(t *testing.T) {
	tables := Tables{
		"storm": {
			Weather: "storm",
			TableSection: TableSection{
				Indoor: map[string]IndoorPool{
					"default": {Strong: []string{"generic"}},
					"fort":    {Strong: []string{"stone walls"}},
				},
			},
		},
	}
	first := func(n int) int { return 0 }
	if got := tables.Pick("storm", "fort", true, 0.9, "", first); got != "stone walls" {
		t.Errorf("biome-specific: got %q", got)
	}
	if got := tables.Pick("storm", "house", true, 0.9, "", first); got != "generic" {
		t.Errorf("default fallback: got %q", got)
	}
	if got := tables.Pick("storm", "fort", true, 0.1, "", first); got != "" {
		t.Errorf("mild with empty mild pool: got %q want silence", got)
	}
}

func TestPick_SeasonalVariant(t *testing.T) {
	tables := Tables{
		"rain": {
			Weather: "rain",
			TableSection: TableSection{
				Outdoor: map[string][]string{"default": {"base outdoor"}, "forest": {"base forest"}},
				Indoor:  map[string]IndoorPool{"default": {Strong: []string{"base indoor"}}},
			},
			Seasonal: map[string]TableSection{
				"winter": {
					Outdoor: map[string][]string{"forest": {"freezing rain"}},
					Indoor:  map[string]IndoorPool{"default": {Strong: []string{"sleet on glass"}}},
				},
			},
		},
	}
	first := func(n int) int { return 0 }

	// season set + variant present -> variant wins
	if got := tables.Pick("rain", "forest", false, 0.7, "winter", first); got != "freezing rain" {
		t.Errorf("winter outdoor variant: got %q", got)
	}
	if got := tables.Pick("rain", "forest", true, 0.7, "winter", first); got != "sleet on glass" {
		t.Errorf("winter indoor variant (strong band): got %q", got)
	}
	// season set but biome missing in variant outdoor -> fall through to BASE
	if got := tables.Pick("rain", "city", false, 0.7, "winter", first); got != "base outdoor" {
		t.Errorf("variant miss should fall through to base: got %q", got)
	}
	// no season -> base only, never variant
	if got := tables.Pick("rain", "forest", false, 0.7, "", first); got != "base forest" {
		t.Errorf("empty season must use base: got %q", got)
	}
	// season with no variant section -> base
	if got := tables.Pick("rain", "forest", false, 0.7, "summer", first); got != "base forest" {
		t.Errorf("unknown season must use base: got %q", got)
	}
}

func TestSeasonalTables_Pick(t *testing.T) {
	st := SeasonalTables{
		SeasonalKey{"temperate", "winter"}: {
			Outdoor: map[string][]string{"default": {"snow ambience"}},
			Indoor:  map[string]IndoorPool{"default": {Strong: []string{"hearth crackles"}}},
		},
	}
	first := func(n int) int { return 0 }

	if got := st.Pick("temperate", "winter", "forest", false, 0.7, first); got != "snow ambience" {
		t.Errorf("seasonal ambience outdoor: got %q", got)
	}
	if got := st.Pick("temperate", "winter", "house", true, 0.9, first); got != "hearth crackles" {
		t.Errorf("seasonal ambience indoor strong: got %q", got)
	}
	if got := st.Pick("temperate", "summer", "forest", false, 0.7, first); got != "" {
		t.Errorf("missing (track,season) must yield silence: got %q", got)
	}
}

func TestLoadSeasonalEmotes(t *testing.T) {
	src := []byte(`track: temperate
season: winter
outdoor:
  default:
    - "Frost rimes every edge."
indoor:
  default:
    strong:
      - "Wind moans in the chimney."
`)
	fsys := fstest.MapFS{"seasons/temperate_winter.yaml": {Data: src}}
	st, err := LoadSeasonalEmotes(fsys, "seasons")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	sec, ok := st[SeasonalKey{"temperate", "winter"}]
	if !ok || len(sec.Outdoor["default"]) != 1 || len(sec.Indoor["default"].Strong) != 1 {
		t.Fatalf("unexpected seasonal table: %+v", st)
	}
}

func TestLoadSeasonalEmotes_RejectsMissingKeys(t *testing.T) {
	fsys := fstest.MapFS{"seasons/bad.yaml": {Data: []byte("outdoor:\n  default: [\"x\"]\n")}}
	if _, err := LoadSeasonalEmotes(fsys, "seasons"); err == nil {
		t.Fatal("seasonal emote file without track/season must be rejected")
	}
}

// The store renders through the shared narration core, actorlessly: the
// Observer role carries the line and the other three roles stay empty. This
// test pins the seam, not the prose, so it must keep passing when a later PR
// rewrites the content.
//
// It also stands in for the old bare-clamp Pick's out-of-range defense.
// narration.Picker documents "an index in [0,n)" as a CONTRACT rather than
// something Render polices, so the sequence-picker loop below proves the
// FULL valid range reaches every line (including the last index, which a
// lingering clamp-to-0 bug would silently mishandle), in place of proving
// out-of-range inputs got clamped.
func TestPickRendersThroughTheNarrationCore(t *testing.T) {
	tables := Tables{
		"rain": {
			Weather: "rain",
			TableSection: TableSection{
				Outdoor: map[string][]string{
					"default": {"first line", "second line", "third line"},
				},
			},
		},
	}
	want := []string{"first line", "second line", "third line"}

	// FirstPicker always returns 0, so the first authored variant must come
	// back. If the store still rolled its own index this would be flaky
	// rather than exact.
	if got := tables.Pick("rain", "default", false, 0, "", narration.FirstPicker); got != want[0] {
		t.Fatalf("FirstPicker should select variant 0, got %q", got)
	}

	// A picker that walks the pool proves the index reaches the core rather
	// than being discarded, across the full valid range.
	seq := narration.SequencePicker()
	for i, w := range want {
		if got := tables.Pick("rain", "default", false, 0, "", seq); got != w {
			t.Fatalf("call %d: want %q, got %q", i, w, got)
		}
	}

	// A nil picker is the assertion that actually proves the core is in the
	// call path. The pre-migration code called the picker directly, so a nil
	// one panicked; narration.Render substitutes DefaultPicker instead. This
	// is the one observable difference the migration makes, since weather
	// authors no tokens and has only one role for Render to coordinate.
	//
	// DefaultPicker is random, so assert membership in the pool, not identity.
	got := tables.Pick("rain", "default", false, 0, "", nil)
	found := false
	for _, w := range want {
		if got == w {
			found = true
		}
	}
	if !found {
		t.Fatalf("a nil picker must fall back to DefaultPicker and return an authored line, got %q", got)
	}
}

func TestParseEmoteTable_IndoorBands(t *testing.T) {
	src := []byte(`weather: rain
outdoor:
  default:
    - "rain falls"
indoor:
  default:
    mild: []
    strong:
      - "rain drums on the roof"
`)
	tbl, err := ParseEmoteTable(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(tbl.Indoor["default"].Strong) != 1 {
		t.Errorf("expected 1 strong indoor line, got %+v", tbl.Indoor["default"])
	}
}

// The three sections are peers on the same struct, and Table embeds
// TableSection so a new section is declared once rather than twice (Table used
// to carry its own copy of Outdoor and Indoor alongside Seasonal's
// TableSection).
func TestUndergroundSectionParses(t *testing.T) {
	src := []byte(`
weather: rain
outdoor:
  default: ["outdoor line"]
indoor:
  default:
    mild: []
    strong: ["indoor line"]
underground:
  default:
    mild: []
    strong: ["underground line"]
seasonal:
  winter:
    underground:
      default:
        mild: []
        strong: ["winter underground line"]
`)
	tbl, err := ParseEmoteTable(src)
	if err != nil {
		t.Fatalf("ParseEmoteTable: %v", err)
	}
	if got := tbl.Underground["default"].Strong; len(got) != 1 || got[0] != "underground line" {
		t.Fatalf("base underground section did not parse: %#v", got)
	}
	// The seasonal variants inherit the new section through the embed. If
	// Table had kept its own field pair this would still be empty.
	if got := tbl.Seasonal["winter"].Underground["default"].Strong; len(got) != 1 {
		t.Fatalf("seasonal underground section did not parse: %#v", got)
	}
	// The existing sections must be unaffected by the embed.
	if got := tbl.Outdoor["default"]; len(got) != 1 || got[0] != "outdoor line" {
		t.Fatalf("outdoor section regressed: %#v", got)
	}
	if got := tbl.Indoor["default"].Strong; len(got) != 1 {
		t.Fatalf("indoor section regressed: %#v", got)
	}
	if tbl.Weather != "rain" {
		t.Fatalf("weather key regressed: %q", tbl.Weather)
	}
}

func TestClassResolution(t *testing.T) {
	tables := Tables{
		"rain": {
			Weather: "rain",
			TableSection: TableSection{
				Outdoor: map[string][]string{"default": {"OUT"}},
				Indoor: map[string]IndoorPool{
					"default": {Mild: []string{"IN-MILD"}, Strong: []string{"IN-STRONG"}},
				},
				Underground: map[string]IndoorPool{
					"default": {Mild: nil, Strong: []string{"UNDER-STRONG"}},
				},
			},
		},
	}

	cases := []struct {
		name   string
		biome  string
		indoor bool
		felt   float64
		want   string
	}{
		{"outdoor ignores the biome", "forest", false, 1.0, "OUT"},
		{"house is surface indoor", "house", true, 1.0, "IN-STRONG"},
		{"fort is surface indoor", "fort", true, 1.0, "IN-STRONG"},
		{"cave is underground", "cave", true, 1.0, "UNDER-STRONG"},
		{"dungeon is underground", "dungeon", true, 1.0, "UNDER-STRONG"},
		{"spiderweb is NOT underground", "spiderweb", true, 1.0, "IN-STRONG"},
		{"indoor mild band below threshold", "house", true, 0.0, "IN-MILD"},
		{"underground mild is empty, so silence", "cave", true, 0.0, ""},
		{"unknown biome falls back to default", "nowhere", true, 1.0, "IN-STRONG"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := tables.Pick("rain", c.biome, c.indoor, c.felt, "", narration.FirstPicker)
			if got != c.want {
				t.Fatalf("want %q, got %q", c.want, got)
			}
		})
	}
}

// Underground must never borrow indoor's or outdoor's prose. A cave with no
// authored underground pool is SILENT, which is the store's standing rule:
// silence beats wrong prose.
func TestUndergroundNeverFallsBackToAnotherClass(t *testing.T) {
	tables := Tables{
		"rain": {
			Weather: "rain",
			TableSection: TableSection{
				Outdoor: map[string][]string{"default": {"OUT"}},
				Indoor: map[string]IndoorPool{
					"default": {Strong: []string{"IN-STRONG"}},
				},
				// Underground deliberately absent.
			},
		},
	}
	if got := tables.Pick("rain", "cave", true, 1.0, "", narration.FirstPicker); got != "" {
		t.Fatalf("underground with no pool must be silent, got %q", got)
	}
}
