package conditions

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// TestMain initializes the mudlog logger once for all tests in this package.
// Conditions.Validate warns on a condition id with no live spec (a save can carry a
// dead id), and mudlog.Warn panics on a nil logger without the engine's
// normal boot init. Mirrors internal/characters/testmain_test.go.
func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}
