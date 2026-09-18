package spells

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"gopkg.in/yaml.v3"
)

// Both shipped worlds, every spell file: the three keys present, no legacy
// key. yaml.v3 ignores unknown keys, so a leftover `type:` would load
// silently; this is the only thing that catches it.
func TestShippedSpellsCarryTheAxesAndNoLegacyKeys(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	root := filepath.Join(filepath.Dir(here), "..", "..")
	legacy := regexp.MustCompile(`(?m)^(type|target_defense_type):`)
	required := []*regexp.Regexp{
		regexp.MustCompile(`(?m)^attack_type:\s*\S`),
		regexp.MustCompile(`(?m)^damage_type:\s*\S`),
		regexp.MustCompile(`(?m)^targeting:\s*\S`),
	}
	seen := 0
	for _, world := range []string{"dogmud", "default"} {
		dir := filepath.Join(root, "_datafiles", "world", world, "spells")
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			if filepath.Ext(e.Name()) != ".yaml" {
				continue
			}
			seen++
			raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if loc := legacy.FindIndex(raw); loc != nil {
				t.Errorf("%s/%s still carries a legacy key: %q", world, e.Name(), raw[loc[0]:loc[1]])
			}
			for _, re := range required {
				if !re.Match(raw) {
					t.Errorf("%s/%s lacks %s", world, e.Name(), re.String())
				}
			}
			var sd SpellData
			if err := yaml.Unmarshal(raw, &sd); err != nil {
				t.Errorf("%s/%s: unmarshal: %v", world, e.Name(), err)
				continue
			}
			if err := sd.validateAxes(); err != nil {
				t.Errorf("%s/%s: validateAxes: %v", world, e.Name(), err)
			}
		}
	}
	if seen < 60 {
		t.Fatalf("scanned only %d spell files; the guard is not looking at the shipped worlds", seen)
	}
}

// charm and core-drain are the two spells whose routing matters most: charm
// is the only social spell in the game and must reach defy, not quell (the
// deleted internal/hooks/charm_channel_test.go tested the transitional shim
// that inferred this from the legacy field, never the shipped data itself).
// core-drain is the one spell with an owner-ruled special case (physical,
// not the mental default) baked into tools/spell_axes_rewrite.py's rewrite
// table -- this pins the DATA half of that ruling, so a future hand-edit of
// the YAML that drifts from the ruling fails here, not in play.
func TestShippedCharmAndCoreDrainAxes(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	root := filepath.Join(filepath.Dir(here), "..", "..")
	dir := filepath.Join(root, "_datafiles", "world", "dogmud", "spells")

	for _, tc := range []struct {
		file string
		want combatvocab.Attack
	}{
		{"charm.yaml", combatvocab.Spell(combatvocab.DamageSocial, combatvocab.TargetSingle)},
		{"core-drain.yaml", combatvocab.Spell(combatvocab.DamagePhysical, combatvocab.TargetArea)},
	} {
		path := filepath.Join(dir, tc.file)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		var sd SpellData
		if err := yaml.Unmarshal(raw, &sd); err != nil {
			t.Fatalf("unmarshalling %s: %v", path, err)
		}
		if got := sd.Attack(); got != tc.want {
			t.Errorf("%s: Attack() = %v, want %v", tc.file, got, tc.want)
		}
	}
}
