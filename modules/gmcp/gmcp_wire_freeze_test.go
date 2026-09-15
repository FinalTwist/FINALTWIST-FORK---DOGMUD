package gmcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Slice 2 of the conditions unification must not change any GMCP JSON field
// name; the web client and Mudlet packages read them. See wire_freeze_test.go
// at the repo root.
func TestWireFreeze_GMCPJSONFieldNames(t *testing.T) {
	keysOf := func(v any) map[string]any {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		m := map[string]any{}
		require.NoError(t, json.Unmarshal(b, &m))
		return m
	}
	assert.Contains(t, keysOf(itemUpdateReq{}), "buffIds")
	assert.Contains(t, keysOf(itemUpdateReq{}), "wornBuffIds")
	assert.Contains(t, keysOf(mobUpdateReq{}), "buffIds")
	assert.Contains(t, keysOf(mobEnums{}), "buffs")
	assert.Contains(t, keysOf(questEnums{}), "buffs")
	assert.Contains(t, keysOf(GMCPCondition{}), "duration_cur")
}
