package actions

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The hidden-detection Search awards moved from usercommands/go.go to
// EntryDetection (movement parity 4b). They must still fire through the
// progression seam on BOTH outcomes, passing the contest result, never a
// literal true and never OnSkillUse directly.
func TestEntryDetectionFiresSearchThroughTheSeamOnBothOutcomes(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	b, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "move.go"))
	require.NoError(t, err)
	src := string(b)

	require.NotContains(t, src, "OnSkillUse(string(skills.Search)",
		"move.go calls the OnSkillUse primitive directly; detection must fire through AwardResolved")
	require.Equal(t, 2, strings.Count(src, "mover.AwardResolved(success,"),
		"expected both hidden-detection sites (players and mobs) to pass the contest result as the won argument")
}
