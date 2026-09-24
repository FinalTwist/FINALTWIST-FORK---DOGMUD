package crimes

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)

	// The witness gate reads room lighting and condition flags, so this
	// binary needs real biome and condition data.
	//
	// Network.LogoutRounds must be set BEFORE conditions load. A test binary
	// gets the Go default of 0, ConditionSpec.Validate overwrites condition
	// 0's triggercount from it, and then refuses any value below 1, so the
	// load panics on 0-meditating.yaml.
	configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles":  "../../_datafiles/world/dogmud",
		"Network.LogoutRounds": 3,
	})
	rooms.LoadBiomeDataFiles()
	conditions.LoadDataFiles()

	os.Exit(m.Run())
}
