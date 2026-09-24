package content

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/narration"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
	"gopkg.in/yaml.v2"
)

// The felt-intensity cutoff at which weather becomes perceptible indoors
// (drumming on roofs, wind in the eaves) is not a package const here: content
// stays pure of the engine's config package (arch_test.go enforces this), so
// every Pick call takes it as an explicit strongFeltThreshold parameter
// instead. The engine reads the live value from
// Balance.WeatherStrongFeltThreshold and supplies it -- see
// modules/weather/engine/emotes.go. Below the threshold, indoor rooms get
// the mild pool — usually empty, i.e. silence.

// IndoorPool holds intensity-banded indoor lines for one biome key.
// Mild plays below the caller-supplied strongFeltThreshold (usually empty:
// light weather doesn't register through walls); Strong plays at/above it.
type IndoorPool struct {
	Mild   []string `yaml:"mild"`
	Strong []string `yaml:"strong"`
}

// TableSection is one outdoor/indoor/underground set of biome-keyed lines.
// Indoor and Underground are felt-banded (IndoorPool) to match the base
// section's schema; Outdoor is a flat list because weather outdoors is never
// attenuated. Used for per-season weather variants and for seasonal-ambience
// tables.
//
// The three are PROSE CLASSES, not room flags: a house and a cave are both
// sheltered, but rain on a roof and water finding a seam in rock are
// different sentences, and 124 of the game's 161 indoor rooms are
// underground.
type TableSection struct {
	Outdoor     map[string][]string   `yaml:"outdoor"`
	Indoor      map[string]IndoorPool `yaml:"indoor"`
	Underground map[string]IndoorPool `yaml:"underground"`
}

// Table holds the ambient lines for one weather type, keyed by biome with a
// "default" fallback, split by prose class (spec §9.4). Outdoor lines are
// uniform random picks; indoor and underground lines are intensity-banded
// (see IndoorPool). The base sections are embedded rather than repeated so
// that adding a class adds it once and the per-season variants inherit it.
// The spec's per-line weights are an unneeded refinement for shipped defaults;
// builders wanting bias can repeat a line.
type Table struct {
	Weather      string `yaml:"weather"`
	TableSection `yaml:",inline"`
	// Seasonal holds optional per-season variants, keyed by season NAME
	// (matching across tracks by design — "winter" is temperate's winter).
	// Missing seasons/sections fall through to the base lines (spec §6).
	Seasonal map[string]TableSection `yaml:"seasonal"`
}

// Tables maps weather type -> emote table.
type Tables map[sim.WeatherType]Table

// ParseEmoteTable parses one emote table file.
func ParseEmoteTable(b []byte) (Table, error) {
	var t Table
	if err := yaml.Unmarshal(b, &t); err != nil {
		return Table{}, err
	}
	if t.Weather == "" {
		return Table{}, fmt.Errorf("emote table missing required 'weather' key")
	}
	return t, nil
}

// minPoolDepth is the floor for a NON-EMPTY ambient pool.
//
// Ambient emotes fire every few rounds for a whole session in one zone, so
// repetition shows far faster here than in combat, where a given pool is drawn
// from only during a fight. Six is where a session stops feeling looped.
const minPoolDepth = 6

// ValidatePool enforces the depth contract for ONE pool.
//
// 🔑 AN EMPTY POOL IS LEGAL AND MEANS DELIBERATE SILENCE: light weather is
// inaudible through walls and imperceptible through stone, and `mild: []` is
// how an author says so. Weather is the only store in the arc where silence is
// an authored value, which is why this wraps narration.ValidateVariants rather
// than calling it directly: everything except the empty case is delegated.
func ValidatePool(lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	return narration.ValidateVariants(
		narration.Variants{Observer: lines}, minPoolDepth, narration.RoleObserver)
}

// LoadEmotes loads every *.yaml emote table under dir in fsys, keyed by the
// table's weather type. A missing dir yields empty tables (silence). The first
// malformed file aborts with an error; the caller decides whether to fail soft.
// On duplicate weather keys, the later file in sorted filename order wins.
func LoadEmotes(fsys fs.FS, dir string) (Tables, error) {
	tables := Tables{}
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return tables, nil
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return tables, fmt.Errorf("%s: %w", e.Name(), err)
		}
		t, err := ParseEmoteTable(b)
		if err != nil {
			return tables, fmt.Errorf("%s: %w", e.Name(), err)
		}
		// Validate every pool in the base section and each seasonal variant
		// before this table is trusted. The caller (weatherModule.loadContent)
		// fails soft on the error this returns: it logs a warning and runs
		// with whatever tables loaded before the bad file, which is silence
		// for the rest. Weather is ambient, so a bad emote file must not stop
		// the world booting — the shipped-data guard (biome_coupling_test.go)
		// is what actually holds the depth contract, because it fails the
		// BUILD rather than the running server. Do not "fix" this into a
		// panic or a hard fail.
		if err := validateTableSection(t.TableSection); err != nil {
			return tables, fmt.Errorf("%s: %w", e.Name(), err)
		}
		for season, sec := range t.Seasonal {
			if err := validateTableSection(sec); err != nil {
				return tables, fmt.Errorf("%s: season %q: %w", e.Name(), season, err)
			}
		}
		tables[sim.WeatherType(t.Weather)] = t
	}
	return tables, nil
}

// validateTableSection runs ValidatePool over every pool in one section,
// naming the specific pool that failed.
func validateTableSection(sec TableSection) error {
	for biome, lines := range sec.Outdoor {
		if err := ValidatePool(lines); err != nil {
			return fmt.Errorf("outdoor/%s: %w", biome, err)
		}
	}
	for _, pair := range []struct {
		name  string
		pools map[string]IndoorPool
	}{{"indoor", sec.Indoor}, {"underground", sec.Underground}} {
		for biome, pool := range pair.pools {
			if err := ValidatePool(pool.Mild); err != nil {
				return fmt.Errorf("%s/%s/mild: %w", pair.name, biome, err)
			}
			if err := ValidatePool(pool.Strong); err != nil {
				return fmt.Errorf("%s/%s/strong: %w", pair.name, biome, err)
			}
		}
	}
	return nil
}

// Pick selects one ambient line for (weather, biome, indoor, felt, season),
// or "" when nothing matches. Lookup order: the season's variant section
// (when season != "" and a variant exists) -> the base section; within a
// section, exact biome -> "default" biome. season "" skips the variant layer
// (seasons off / unbound zone). Indoor never falls back to outdoor — silence
// beats wrong prose — and is felt-banded: felt below strongFeltThreshold
// picks Mild (usually empty), otherwise Strong. strongFeltThreshold is
// supplied by the caller rather than read here: content stays pure of the
// engine's config package (see arch_test.go), and the engine tier reads the
// live value from Balance.WeatherStrongFeltThreshold and passes it down. pick
// is handed to the narration core as-is: per narration.Picker's contract it
// must return a value in [0,n), and the core trusts that rather than
// clamping, same as every other migrated store. Pass util.Rand-backed
// pickers — NEVER the sim RNG.
func (ts Tables) Pick(weather sim.WeatherType, biome string, indoor bool, felt float64, strongFeltThreshold float64, season string, pick narration.Picker) string {
	t, ok := ts[weather]
	if !ok {
		return ""
	}

	var lines []string
	if season != "" {
		if v, ok := t.Seasonal[season]; ok {
			lines = bandedSectionLines(v, biome, indoor, felt, strongFeltThreshold)
		}
	}
	if len(lines) == 0 {
		lines = bandedSectionLines(t.TableSection, biome, indoor, felt, strongFeltThreshold)
	}

	return renderAmbient(lines, pick)
}

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

// undergroundBiomes names the biomes whose weather is felt through STONE
// rather than through walls: seepage, draughts, transmitted sound, mineral
// cold. 124 of the game's 161 indoor rooms are one of these, which is why the
// class exists at all.
//
// 🔑 ADDING, RENAMING OR REMOVING A BIOME REQUIRES EDITING THIS MAP OR
// surfaceIndoorBiomes. biome_coupling_test.go fails the build otherwise; it is
// the only thing standing between a new indoor biome and silently inheriting
// prose about roofs and windowpanes.
var undergroundBiomes = map[string]bool{
	"cave":    true,
	"dungeon": true,
	// sewer is brick vaults and channels under a city: felt through stone
	// (seepage, draughts, mineral cold), not through a roof.
	"sewer": true,
}

// surfaceIndoorBiomes names the sheltered-but-not-underground biomes: built
// structures, where rain on a roof and wind in the eaves are the right images.
//
// This map exists so classification is TOTAL. Without it a newly added indoor
// biome would fall through to this class silently, which is exactly the defect
// the underground split was written to fix.
//
// spiderweb is here rather than in undergroundBiomes deliberately: it is dark
// and sheltered, but its darkness is webbing, not stone, so stone prose would
// be wrong. It currently has ZERO rooms, so no prose is authored for it; if it
// is ever used it wants its own biome-keyed pool rather than either default.
var surfaceIndoorBiomes = map[string]bool{
	"fort":      true,
	"spiderweb": true,
	// interior is a built structure: houses, halls, temples, archives.
	// Roofs, eaves, windows are the right images. house was folded into it.
	"interior": true,
	// ether has no weather at all, so it is genuinely neither class. It is
	// filed here rather than in undergroundBiomes because surface-indoor is
	// the right default for a place with no stone around it, and because
	// classification must be TOTAL (TestEveryIndoorBiomeIsClassified). It has
	// zero rooms authoring weather prose today, so this choice is currently
	// theoretical, same as spiderweb above.
	"ether": true,
}

// bandedSectionLines resolves one prose class and then biome -> "default"
// within it. Outdoor is a flat list; Indoor and Underground are felt-banded
// (Mild below strongFeltThreshold, else Strong). strongFeltThreshold is the
// caller-supplied value of Balance.WeatherStrongFeltThreshold; see Pick's
// doc comment for why it is a parameter rather than a package const.
//
// A class NEVER falls back to another class. An unauthored underground pool
// renders silence rather than borrowing house prose, which is the whole point
// of the split.
//
// 🪤 THE BIOME KEY IS LOWERCASED BEFORE EVERY LOOKUP, and that is load-bearing.
// EmitAmbient passes rooms.BiomeInfo.BiomeId RAW, which is the authored yaml
// value and NOT canonicalised; the room model's own BiomeInfo.Id() lowercases
// precisely because of that. Every shipped biomeid happens to be lowercase
// today, so a raw lookup works by luck.
//
// Without this, a builder authoring `biomeid: Crypt` would get a classification
// MISS here and silently receive prose written for a built interior, while
// biome_coupling_test.go, which keys its map with strings.ToLower, would still
// pass. A guard that stays green while production is wrong is worse than no
// guard, which is why the normalisation lives here rather than in the caller.
func bandedSectionLines(sec TableSection, biome string, useIndoor bool, felt float64, strongFeltThreshold float64) []string {
	biome = strings.ToLower(biome)

	if !useIndoor {
		lines := sec.Outdoor[biome]
		if len(lines) == 0 {
			lines = sec.Outdoor["default"]
		}
		return lines
	}

	pools := sec.Indoor
	if undergroundBiomes[biome] {
		pools = sec.Underground
	}

	pool, ok := pools[biome]
	if !ok || (len(pool.Mild) == 0 && len(pool.Strong) == 0) {
		pool = pools["default"]
	}
	if felt >= strongFeltThreshold {
		return pool.Strong
	}
	return pool.Mild
}

// SeasonalKey identifies one (track, season) ambience table.
type SeasonalKey struct{ Track, Season string }

// SeasonalTables holds the seasonal-ambience emote tables — the persistent
// voice of a season in CALM weather (the weather tables' Seasonal variants
// cover weathered moments). Loaded from weather/emotes/seasons/.
type SeasonalTables map[SeasonalKey]TableSection

// seasonalEmoteFile mirrors the on-disk schema for seasons/*.yaml. It must
// carry every class TableSection does: yaml.v2 silently drops unknown keys,
// so a class missing here means prose authored under that key on disk is
// parsed, discarded, and never reported — the same failure mode as an
// authored biome key that matches no known biome.
type seasonalEmoteFile struct {
	Track       string                `yaml:"track"`
	Season      string                `yaml:"season"`
	Outdoor     map[string][]string   `yaml:"outdoor"`
	Indoor      map[string]IndoorPool `yaml:"indoor"`
	Underground map[string]IndoorPool `yaml:"underground"`
}

// LoadSeasonalEmotes loads every *.yaml under dir, keyed by (track, season).
// Missing dir = empty tables; the first malformed file aborts with an error
// (caller fails soft). Requires both 'track' and 'season' keys. On duplicate
// (track, season) keys, the later file in sorted filename order wins.
func LoadSeasonalEmotes(fsys fs.FS, dir string) (SeasonalTables, error) {
	out := SeasonalTables{}
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return out, nil
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return out, fmt.Errorf("%s: %w", e.Name(), err)
		}
		var f seasonalEmoteFile
		if err := yaml.Unmarshal(b, &f); err != nil {
			return out, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if f.Track == "" || f.Season == "" {
			return out, fmt.Errorf("%s: missing required 'track' or 'season' key", e.Name())
		}
		sec := TableSection{Outdoor: f.Outdoor, Indoor: f.Indoor, Underground: f.Underground}
		// See the matching validation in LoadEmotes: the caller
		// (weatherModule.loadContent) fails soft on this error, logging a
		// warning and running with empty seasonal-ambience tables (silence)
		// rather than stopping the world booting. The shipped-data guard is
		// what holds the depth contract by failing the build.
		if err := validateTableSection(sec); err != nil {
			return out, fmt.Errorf("%s: %w", e.Name(), err)
		}
		out[SeasonalKey{f.Track, f.Season}] = sec
	}
	return out, nil
}

// Pick selects one seasonal-ambience line for the zone's exact (track,
// season); "" when no table or no matching lines. Same biome/indoor banding
// and picker contract as the weather tables, including strongFeltThreshold
// being caller-supplied rather than a package const (see Tables.Pick).
func (st SeasonalTables) Pick(track, season, biome string, indoor bool, felt float64, strongFeltThreshold float64, pick narration.Picker) string {
	sec, ok := st[SeasonalKey{track, season}]
	if !ok {
		return ""
	}
	return renderAmbient(bandedSectionLines(sec, biome, indoor, felt, strongFeltThreshold), pick)
}
