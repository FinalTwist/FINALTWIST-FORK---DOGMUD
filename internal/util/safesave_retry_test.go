package util

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// On Windows a reader holding the file open (Go opens without
// FILE_SHARE_DELETE) makes the rename in SafeSave fail for a moment; the
// bauble sweep reads every save file, so SafeSave retries briefly. Linux
// renames over an open file freely and never needs the retry.
func TestSafeSave_RetriesARenameBlockedByAReader(t *testing.T) {
	if runtime.GOOS != `windows` {
		t.Skip(`only Windows refuses a rename onto a file another handle has open`)
	}
	path := filepath.Join(t.TempDir(), `state.yaml`)
	require.NoError(t, SafeSave(path, []byte("one\n")))
	f, err := os.Open(path)
	require.NoError(t, err)
	go func() {
		time.Sleep(40 * time.Millisecond)
		_ = f.Close()
	}()
	require.NoError(t, SafeSave(path, []byte("two\n")))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "two\n", string(data))
}
