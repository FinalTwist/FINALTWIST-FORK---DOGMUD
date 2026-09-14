package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"gopkg.in/yaml.v3"
)

// Slice E: every flag a dogmud condition carries must be one the engine
// declares, spelled exactly. An unknown flag used to load silently and do
// nothing: the Cat's Eye Draught shipped `night-vision` for `nightvision`
// and gave no night vision for weeks, and Stone Stomach's `poison-immunity`
// was read by nothing at all. LoadDataFiles now panics on one; this fails
// the merge before it can reach a boot.
func TestEveryDogmudConditionFlagIsDeclared(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("_datafiles", "world", "dogmud", "buffs", "*.yaml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no condition files: %v", err)
	}
	sort.Strings(files)
	var problems []string
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var b struct {
			ConditionId int      `yaml:"buffid"`
			Name        string   `yaml:"name"`
			Flags       []string `yaml:"flags"`
		}
		if err := yaml.Unmarshal(raw, &b); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		spec := &conditions.ConditionSpec{ConditionId: b.ConditionId, Name: b.Name}
		for _, fl := range b.Flags {
			spec.Flags = append(spec.Flags, conditions.Flag(fl))
		}
		if err := spec.ValidateFlags(); err != nil {
			problems = append(problems, filepath.Base(f)+": "+err.Error())
		}
	}
	if len(problems) > 0 {
		t.Fatalf("%d unknown condition flags:\n  %s", len(problems), strings.Join(problems, "\n  "))
	}
}
