package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

const (
	cancelTestRadiance     = 9761
	cancelTestDraught      = 9762
	cancelTestIllumination = 9763
	cancelTestOm           = 9764
	cancelTestUnheld       = 9765
)

// cancelFixture seeds cancellable conditions (a radiance, an illumination, a
// two-letter "Om"), a non-cancellable draught, a spell (alias "shine") that
// grants the radiance, and an unrelated spell (alias "zap") granting a
// condition the user never holds. Returns user 1 holding none of them.
func cancelFixture(t *testing.T) *users.UserRecord {
	t.Helper()
	t.Cleanup(seedAllRegistries())
	cancellable := []conditions.Flag{conditions.Cancellable}
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		cancelTestRadiance:     {ConditionId: cancelTestRadiance, Name: "Test Radiance", TriggerCount: 4, RoundInterval: 1, Flags: cancellable},
		cancelTestDraught:      {ConditionId: cancelTestDraught, Name: "Test Draught", TriggerCount: 4, RoundInterval: 1},
		cancelTestIllumination: {ConditionId: cancelTestIllumination, Name: "Illumination", TriggerCount: 4, RoundInterval: 1, Flags: cancellable},
		cancelTestOm:           {ConditionId: cancelTestOm, Name: "Om", TriggerCount: 4, RoundInterval: 1, Flags: cancellable},
		cancelTestUnheld:       {ConditionId: cancelTestUnheld, Name: "Test Unheld", TriggerCount: 4, RoundInterval: 1, Flags: cancellable},
	}))
	t.Cleanup(spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"test-radiance-spell": {SpellId: "test-radiance-spell", Name: "Radiant Shimmer", Aliases: []string{"shine"},
			ConditionIds: []int{cancelTestRadiance}},
		"test-unrelated-spell": {SpellId: "test-unrelated-spell", Name: "Zapping Bolt", Aliases: []string{"zap"},
			ConditionIds: []int{cancelTestUnheld}},
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
	events.DrainQueuedMessagesForTest(user.UserId)

	_, err := Cancel("test draught", user, nil, 0)
	require.NoError(t, err)
	require.True(t, holds(user, cancelTestDraught), "a condition without the cancellable flag was cancelled")
	require.Contains(t, cancelOutput(user), `You have no test draught you can let go of.`)

	_, err = Cancel("test radiance", user, nil, 0)
	require.NoError(t, err)
	require.False(t, holds(user, cancelTestRadiance), "cancel did not end a cancellable condition")
	require.Contains(t, cancelOutput(user), `You let your test radiance go.`)
}

func TestCancelResolvesASpellAlias(t *testing.T) {
	user := cancelFixture(t)
	require.NoError(t, user.Character.AddCondition(cancelTestRadiance, false))

	_, err := Cancel("shine", user, nil, 0)
	require.NoError(t, err)
	require.False(t, holds(user, cancelTestRadiance), "cancel <alias> did not reach the spell's condition")
}

func TestCancelResolvesASpellNamePrefix(t *testing.T) {
	user := cancelFixture(t)
	require.NoError(t, user.Character.AddCondition(cancelTestRadiance, false))

	_, err := Cancel("radiant", user, nil, 0)
	require.NoError(t, err)
	require.False(t, holds(user, cancelTestRadiance), "cancel <spell name prefix> did not reach the spell's condition")
}

func TestCancelPrefixNeedsThreeCharacters(t *testing.T) {
	user := cancelFixture(t)
	require.NoError(t, user.Character.AddCondition(cancelTestIllumination, false))
	events.DrainQueuedMessagesForTest(user.UserId)

	_, _ = Cancel("il", user, nil, 0)
	require.True(t, holds(user, cancelTestIllumination), "a two-character prefix ended a condition")
	require.Contains(t, cancelOutput(user), `You have no il you can let go of.`)

	_, _ = Cancel("ill", user, nil, 0)
	require.False(t, holds(user, cancelTestIllumination), "a three-character prefix did not end the condition")
}

func TestCancelExactShortName(t *testing.T) {
	user := cancelFixture(t)
	require.NoError(t, user.Character.AddCondition(cancelTestOm, false))

	_, _ = Cancel("o", user, nil, 0)
	require.True(t, holds(user, cancelTestOm), "a one-character prefix ended a condition")

	_, _ = Cancel("OM", user, nil, 0)
	require.False(t, holds(user, cancelTestOm), "an exact short name did not end the condition")
}

func TestCancelNeverReachesASpellYouDoNotHold(t *testing.T) {
	user := cancelFixture(t)
	require.NoError(t, user.Character.AddCondition(cancelTestRadiance, false))
	events.DrainQueuedMessagesForTest(user.UserId)

	_, _ = Cancel("zap", user, nil, 0)
	require.True(t, holds(user, cancelTestRadiance), "an unrelated spell's name ended a held condition")
	require.Contains(t, cancelOutput(user), `You have no zap you can let go of.`)
}

func TestCancelWithNoArgumentKeepsTheActivityRefusal(t *testing.T) {
	user := cancelFixture(t)
	events.DrainQueuedMessagesForTest(user.UserId)

	_, err := Cancel("", user, nil, 0)
	require.NoError(t, err)
	require.Contains(t, cancelOutput(user), `You aren't doing anything to cancel.`)
}

func TestCancelWithAnArgumentStillStopsACast(t *testing.T) {
	user := cancelFixture(t)
	require.NoError(t, user.Character.AddCondition(cancelTestRadiance, false))
	if user.Character.Activity == nil {
		user.Character.Activity = activity.NewMachine()
	}
	require.NoError(t, user.Character.Activity.TransitionToCasting(
		activity.CastingData{SpellId: "sparks", ConvictionSpent: 3, FoldsNeeded: 4},
		state.TransitionReason{Trigger: activity.TriggerCastBegin},
	))
	t.Cleanup(func() {
		_ = user.Character.Activity.TransitionToFree(state.TransitionReason{Trigger: "test-cleanup"})
	})
	events.DrainQueuedMessagesForTest(user.UserId)

	_, err := Cancel("cast", user, nil, 0)
	require.NoError(t, err)
	require.True(t, user.Character.Activity.IsFree(), "cancel <arg> did not stop the cast")
	require.Contains(t, cancelOutput(user), `You stop casting.`)
	require.True(t, holds(user, cancelTestRadiance), "cancel during a cast also ended a held condition")
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
