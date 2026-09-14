package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The SAME fixture and expectation as internal/usercommands'
// TestConditionEntries_LeavesOutHiddenAndSecretRecords.
func TestBuildConditionsPayload_ListsWhatTheConditionsCommandLists(t *testing.T) {
	t.Cleanup(buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		990: {BuffId: 990, Name: "Stoneskin", Description: "Skin like rock.", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 10},
		991: {BuffId: 991, Name: "Hidden", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 10, Flags: []buffs.Flag{buffs.Hidden}},
		992: {BuffId: 992, Name: "Respawn Grace", Secret: true, TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 10},
	}))
	t.Cleanup(buffs.SeedConditionRecordsForTest())

	c := &characters.Character{}
	c.Buffs.Validate(true)
	for _, id := range []int{990, 991, 992} {
		require.True(t, c.Buffs.AddBuff(id, false), "fixture: add %d", id)
	}
	require.True(t, c.Buffs.AddBuffMagnitude(buffs.BuffIdBleeding, 3, -2))
	require.True(t, c.Buffs.AddBuffMagnitude(buffs.BuffIdBleeding, 5, -3))

	payload := buildConditionsPayload(c)

	names := []string{}
	for _, cond := range payload {
		names = append(names, cond.Name)
	}
	assert.ElementsMatch(t, []string{"Stoneskin", "Bleeding (2)"}, names)
	assert.Contains(t, payload, "Bleeding", "the map key stays the plain name; only the label carries the count")
}
