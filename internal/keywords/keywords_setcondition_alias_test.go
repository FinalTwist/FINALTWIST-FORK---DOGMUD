package keywords

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/require"
)

// TestAliasResolvesToSetCondition loads each shipped world's real
// keywords.yaml (not a synthetic fixture) and proves the admin command alias
// kept for slice 2 of the conditions unification (owner ruling 2026-09-14:
// `setcondition` is the real command, `buff` stays a working alias until
// slice 3) actually resolves in both the command dispatcher and the help
// system. `TryCommandAlias` is what usercommands.TryCommand consults before
// indexing the command map, so this is the same resolution a typed `buff`
// command goes through; `TryHelpAlias` is what `help buff` consults.
func TestAliasResolvesToSetCondition(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)

	// LoadAliases() below overwrites the package-level loadedKeywords var
	// directly (see LoadAliases in keywords.go); configs.SetConfigForTest
	// only restores the config, not this. Save and restore it the same way
	// SeedKeywordsForTest does, so this test does not leave a later
	// test's keyword state pointed at whichever world ran last.
	origKeywords := loadedKeywords
	defer func() { loadedKeywords = origKeywords }()

	for _, world := range []string{"dogmud", "default"} {
		t.Run(world, func(t *testing.T) {
			cfg := configs.GetConfig()
			cfg.FilePaths.DataFiles = configs.ConfigString(filepath.Join(filepath.Dir(here), "..", "..", "_datafiles", "world", world))
			configs.SetConfigForTest(t, cfg)

			LoadAliases()

			require.Equal(t, "setcondition", TryCommandAlias("buff"),
				"the %s world's keywords.yaml must alias the `buff` command to `setcondition`", world)
			require.Equal(t, "setcondition", TryHelpAlias("buff"),
				"the %s world's keywords.yaml must alias `help buff` to `help setcondition`", world)
		})
	}
}
