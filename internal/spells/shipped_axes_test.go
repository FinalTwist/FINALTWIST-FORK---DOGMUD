package spells

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
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
		}
	}
	if seen < 60 {
		t.Fatalf("scanned only %d spell files; the guard is not looking at the shipped worlds", seen)
	}
}
