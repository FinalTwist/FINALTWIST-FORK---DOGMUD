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

// The old bare-clamp Pick defended against a badly-behaved roll func by
// forcing anything outside [0,len(lines)) back to index 0. narration.Picker
// documents that contract instead of policing it — "Picker chooses an index
// in [0,n)" — and Render trusts a picker to honor it, exactly like every
// other store already on the core (e.g. itemvoices.VoiceSpec.LineWith). So
// this test now proves the FULL valid range reaches every line, in place of
// proving out-of-range inputs got clamped.
func TestPickHonorsFullPickerRange(t *testing.T) {
	tables := loadTestTables(t)
	if got := tables.Pick("storm", "default", false, 0.7, "", func(n int) int { return 0 }); got != "Thunder cracks directly overhead." {
		t.Errorf("index 0: %q", got)
	}
	if got := tables.Pick("storm", "default", false, 0.7, "", func(n int) int { return n - 1 }); got != "A blinding fork of lightning splits the sky." {
		t.Errorf("index n-1: %q", got)
	}
}

func TestPick_IndoorIntensityBands(t *testing.T) {
	tables := Tables{
		"rain": {
			Weather: "rain",
			Outdoor: map[string][]string{"default": {"out"}},
			Indoor: map[string]IndoorPool{
				"default": {Mild: nil, Strong: []string{"roof"}},
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
			Indoor: map[string]IndoorPool{
				"default": {Strong: []string{"generic"}},
				"fort":    {Strong: []string{"stone walls"}},
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
			Outdoor: map[string][]string{"default": {"base outdoor"}, "forest": {"base forest"}},
			Indoor:  map[string]IndoorPool{"default": {Strong: []string{"base indoor"}}},
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
