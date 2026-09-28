package migration

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/version"
)

// TestRecordMigratedVersionWritesThroughServerLocked: the shipped config
// locks Server.CurrentVersion against operators. The migration's own write
// must still land, or every migration re-runs on every boot.
func TestRecordMigratedVersionWritesThroughServerLocked(t *testing.T) {
	c := configs.GetConfig()
	c.Server.Locked = configs.ConfigSliceString{`Server.CurrentVersion`}
	c.Server.CurrentVersion = `0.1.0`
	configs.SetConfigWithLookupsForTest(t, c)

	recordMigratedVersion(version.New(0, 99, 0))

	if got := string(configs.GetServerConfig().CurrentVersion); got != `0.99.0` {
		t.Fatalf(`Server.CurrentVersion = %q after recordMigratedVersion(0.99.0), want "0.99.0"`, got)
	}
}
