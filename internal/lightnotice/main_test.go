package lightnotice

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// shippedDir is the store as a booted dogmud world reads it. A test binary
// never reads config.yaml, so tests load it explicitly.
const shippedDir = "../../_datafiles/world/dogmud/narration/light-notices"

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}
