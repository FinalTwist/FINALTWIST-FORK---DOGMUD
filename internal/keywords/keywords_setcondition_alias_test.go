package keywords

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/require"
)

// TestOldAdminAliasIsGone loads each shipped world's real keywords.yaml and
// proves conditions unification slice 3 removed the `buff` alias that slice 2
// kept for `setcondition` (owner ruling 2026-09-14: alias until slice 3).
// TryCommandAlias and TryHelpAlias return their input unchanged when no alias
// matches.
func TestOldAdminAliasIsGone(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)

	origKeywords := loadedKeywords
	defer func() { loadedKeywords = origKeywords }()

	for _, world := range []string{"dogmud", "default"} {
		t.Run(world, func(t *testing.T) {
			cfg := configs.GetConfig()
			cfg.FilePaths.DataFiles = configs.ConfigString(filepath.Join(filepath.Dir(here), "..", "..", "_datafiles", "world", world))
			configs.SetConfigForTest(t, cfg)

			LoadAliases()

			require.Equal(t, "buff", TryCommandAlias("buff"),
				"the %s world's keywords.yaml must not alias `buff` to any command", world)
			require.Equal(t, "buff", TryHelpAlias("buff"),
				"the %s world's keywords.yaml must not alias `help buff`", world)
			require.Equal(t, "setcondition", TryCommandAlias("setcondition"))
		})
	}
}
