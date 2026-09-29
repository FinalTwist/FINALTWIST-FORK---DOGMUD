package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// TestSetNextRoomIdWritesThroughServerLocked: the shipped config locks
// Server.NextRoomId against operators; room creation's own write must land.
func TestSetNextRoomIdWritesThroughServerLocked(t *testing.T) {
	c := configs.GetConfig()
	c.Server.Locked = configs.ConfigSliceString{`Server.NextRoomId`}
	c.Server.NextRoomId = 1002
	configs.SetConfigWithLookupsForTest(t, c)

	SetNextRoomId(5150)

	if got := GetNextRoomId(); got != 5150 {
		t.Fatalf(`GetNextRoomId() = %d after SetNextRoomId(5150), want 5150: the engine write was refused as an operator write`, got)
	}
}
