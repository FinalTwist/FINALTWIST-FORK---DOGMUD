package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The SAME fixture and expectation as modules/gmcp's
// TestBuildConditionsPayload_ListsWhatTheConditionsCommandLists: a plain, a
// hidden, a secret and a stacked record; only the plain and the stacked one
// are listed, and the stacked name carries its count.
func TestConditionEntries_LeavesOutHiddenAndSecretRecords(t *testing.T) {
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		990: {ConditionId: 990, Name: "Stoneskin", Description: "Skin like rock.", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 10},
		991: {ConditionId: 991, Name: "Hidden", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 10, Flags: []conditions.Flag{conditions.Hidden}},
		992: {ConditionId: 992, Name: "Respawn Grace", Secret: true, TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 10},
	}))
	t.Cleanup(conditions.SeedConditionRecordsForTest())

	c := &characters.Character{}
	c.Conditions.Validate(true)
	for _, id := range []int{990, 991, 992} {
		require.True(t, c.Conditions.AddCondition(id, false), "fixture: add %d", id)
	}
	require.True(t, c.Conditions.AddConditionMagnitude(conditions.ConditionIdBleeding, 3, -2))
	require.True(t, c.Conditions.AddConditionMagnitude(conditions.ConditionIdBleeding, 5, -3))

	names := []string{}
	for _, e := range conditionEntries(c) {
		names = append(names, e.Name)
	}
	assert.ElementsMatch(t, []string{"Stoneskin", "Bleeding (2)"}, names)
}
