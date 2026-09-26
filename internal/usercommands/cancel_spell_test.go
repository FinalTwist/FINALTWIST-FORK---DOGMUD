package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

const (
	cancelTestRadiance = 9761
	cancelTestDraught  = 9762
)

// cancelFixture seeds one cancellable and one non-cancellable condition, a
// spell (alias "shine") that grants the cancellable one, and returns user 1
// holding neither.
func cancelFixture(t *testing.T) *users.UserRecord {
	t.Helper()
	t.Cleanup(seedAllRegistries())
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		cancelTestRadiance: {ConditionId: cancelTestRadiance, Name: "Test Radiance", TriggerCount: 4, RoundInterval: 1,
			Flags: []conditions.Flag{conditions.Cancellable}},
		cancelTestDraught: {ConditionId: cancelTestDraught, Name: "Test Draught", TriggerCount: 4, RoundInterval: 1},
	}))
	t.Cleanup(spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"test-radiance-spell": {SpellId: "test-radiance-spell", Name: "Radiant Shimmer", Aliases: []string{"shine"},
			ConditionIds: []int{cancelTestRadiance}},
	}))
	user := users.GetByUserId(1)
	require.NotNil(t, user)
	return user
}

// holds reports whether the user holds an unexpired record of the condition.
// HasCondition cannot answer this: it reads the id index, which keeps an
// expired record until the next prune.
func holds(user *users.UserRecord, id int) bool {
	return len(user.Character.Conditions.GetConditions(id)) > 0
}

func cancelOutput(user *users.UserRecord) string {
	return strings.Join(events.DrainQueuedMessagesForTest(user.UserId), "\n")
}

func TestCancelEndsACancellableSpell(t *testing.T) {
	user := cancelFixture(t)
	require.NoError(t, user.Character.AddCondition(cancelTestRadiance, false))
	require.NoError(t, user.Character.AddCondition(cancelTestDraught, false))

	_, err := Cancel("test draught", user, nil, 0)
	require.NoError(t, err)
	require.True(t, holds(user, cancelTestDraught), "a condition without the cancellable flag was cancelled")

	_, err = Cancel("test radiance", user, nil, 0)
	require.NoError(t, err)
	require.False(t, holds(user, cancelTestRadiance), "cancel did not end a cancellable condition")
}

func TestCancelResolvesASpellAlias(t *testing.T) {
	user := cancelFixture(t)
	require.NoError(t, user.Character.AddCondition(cancelTestRadiance, false))

	_, err := Cancel("shine", user, nil, 0)
	require.NoError(t, err)
	require.False(t, holds(user, cancelTestRadiance), "cancel <alias> did not reach the spell's condition")
}

func TestCancelWithNoArgumentKeepsTheActivityRefusal(t *testing.T) {
	user := cancelFixture(t)
	events.DrainQueuedMessagesForTest(user.UserId)

	_, err := Cancel("", user, nil, 0)
	require.NoError(t, err)
	require.Contains(t, cancelOutput(user), `You aren't doing anything to cancel.`)
}

func TestCancelRefusesWhatYouDoNotHold(t *testing.T) {
	user := cancelFixture(t)
	require.NoError(t, user.Character.AddCondition(cancelTestDraught, false))
	events.DrainQueuedMessagesForTest(user.UserId)

	_, err := Cancel("shine", user, nil, 0)
	require.NoError(t, err)
	require.Contains(t, cancelOutput(user), `You have no shine you can let go of.`)

	// A second cancel of an already let-go condition refuses too, rather
	// than succeeding silently against the expired record.
	require.NoError(t, user.Character.AddCondition(cancelTestRadiance, false))
	_, _ = Cancel("shine", user, nil, 0)
	events.DrainQueuedMessagesForTest(user.UserId)
	_, _ = Cancel("shine", user, nil, 0)
	require.Contains(t, cancelOutput(user), `You have no shine you can let go of.`)
}
