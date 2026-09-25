package events

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDrainQueuedGoldGivenForTest(t *testing.T) {
	DrainQueuedGoldGivenForTest(0)
	AddToQueue(GoldGiven{UserId: 41, MobInstanceId: 301, Amount: 5})
	AddToQueue(GoldGiven{UserId: 42, MobInstanceId: 302, Amount: 7})

	require.Equal(t, []GoldGiven{{UserId: 41, MobInstanceId: 301, Amount: 5}},
		DrainQueuedGoldGivenForTest(41))
	require.Equal(t, []GoldGiven{{UserId: 42, MobInstanceId: 302, Amount: 7}},
		DrainQueuedGoldGivenForTest(0))
	require.Equal(t, "GoldGiven", GoldGiven{}.Type())
}
