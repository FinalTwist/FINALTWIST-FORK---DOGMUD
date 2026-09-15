package hooks

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Condition room lines describe what the room SEES ("A warm glow surrounds Alice"),
// but went out on the audio channel, which is never sight-gated, so blind and
// unsighted observers received them. M2 fixed the same defect for
// cast_room_text; these three condition phases were never touched.

// expire sets a condition's remaining triggers to the pruning threshold, so the next
// PruneConditions removes it and sends its end text. Deterministic, unlike counting
// ticks.
func expire(t *testing.T, list []*conditions.Condition, conditionId int) {
	t.Helper()
	for _, b := range list {
		if b.ConditionId == conditionId {
			b.TriggersLeft = conditions.TriggersLeftExpired
			return
		}
	}
	t.Fatalf("condition %d not found to expire", conditionId)
}

// rawLineContaining returns the first raw (still tagged) line whose plain text
// contains want, or "" if none does.
func rawLineContaining(raw []string, want string) string {
	for _, line := range raw {
		if strings.Contains(plainText(line), want) {
			return line
		}
	}
	return ""
}

func TestConditionStartRoomText_SightedObserverSeesIt(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	drainPlain(2)

	assert.Equal(t, events.Continue, ApplyConditions(events.Condition{UserId: 1, ConditionId: glowConditionId}))
	assert.Equal(t, 1, countContaining(drainPlain(2), "Aliceia glows."))
}

func TestConditionStartRoomText_UnsightedObserverInTheDarkGetsNothing(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	darken(t, 1)
	drainPlain(2)

	ApplyConditions(events.Condition{UserId: 1, ConditionId: glowConditionId})
	assert.Equal(t, 0, countContaining(drainPlain(2), "glows."),
		"an observer who cannot see must not be told what a condition looks like")
}

func TestConditionStartRoomText_NightVisionSeesItInTheDark(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	darken(t, 1)
	require.True(t, users.GetByUserId(2).Character.Conditions.AddCondition(nightEyesConditionId, true))
	drainPlain(2)

	ApplyConditions(events.Condition{UserId: 1, ConditionId: glowConditionId})
	assert.Equal(t, 1, countContaining(drainPlain(2), "Aliceia glows."))
}

func TestConditionStartRoomText_MobHolderUsesTheMobTag(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	events.DrainQueuedMessagesForTest(2)

	ApplyConditions(events.Condition{MobInstanceId: 100, ConditionId: glowConditionId})
	line := rawLineContaining(events.DrainQueuedMessagesForTest(2), "Skeleton glows.")
	require.NotEmpty(t, line, "the observer must receive the mob's start text")
	assert.Contains(t, line, `fg="mobname`)
	assert.NotContains(t, line, `fg="username`,
		"a mob holder was tagged with the player colour")
}

func TestConditionTriggerRoomText_UnsightedObserverInTheDarkGetsNothing(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	darken(t, 1)
	require.True(t, users.GetByUserId(1).Character.Conditions.AddCondition(shiverConditionId, false))
	drainPlain(2)

	UserRoundTick(events.NewRound{RoundNumber: 1})
	assert.Equal(t, 0, countContaining(drainPlain(2), "shivers."))
}

func TestConditionTriggerRoomText_SightedObserverSeesIt(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	require.True(t, users.GetByUserId(1).Character.Conditions.AddCondition(shiverConditionId, false))
	drainPlain(2)

	UserRoundTick(events.NewRound{RoundNumber: 1})
	assert.Equal(t, 1, countContaining(drainPlain(2), "Aliceia shivers."))
}

func TestConditionEndRoomText_UnsightedObserverInTheDarkGetsNothing(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	darken(t, 1)
	holder := users.GetByUserId(1)
	require.True(t, holder.Character.Conditions.AddCondition(fadeConditionId, false))
	expire(t, holder.Character.Conditions.List, fadeConditionId)
	drainPlain(2)

	PruneConditions(events.NewTurn{TurnNumber: 1})
	assert.Equal(t, 0, countContaining(drainPlain(2), "fades."))
}

func TestConditionEndRoomText_SightedObserverSeesIt(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	holder := users.GetByUserId(1)
	require.True(t, holder.Character.Conditions.AddCondition(fadeConditionId, false))
	expire(t, holder.Character.Conditions.List, fadeConditionId)
	drainPlain(2)

	PruneConditions(events.NewTurn{TurnNumber: 1})
	assert.Equal(t, 1, countContaining(drainPlain(2), "Aliceia fades."))
}

func TestConditionEndRoomText_MobHolderIsVisualAndUsesTheMobTag(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	mob := mobs.GetInstance(100)
	require.True(t, mob.Character.Conditions.AddCondition(fadeConditionId, false))
	expire(t, mob.Character.Conditions.List, fadeConditionId)
	events.DrainQueuedMessagesForTest(2)

	PruneConditions(events.NewTurn{TurnNumber: 1})
	line := rawLineContaining(events.DrainQueuedMessagesForTest(2), "Skeleton fades.")
	require.NotEmpty(t, line)
	assert.Contains(t, line, `fg="mobname`)

	// And the same line is gated by sight.
	require.True(t, mob.Character.Conditions.AddCondition(fadeConditionId, false))
	expire(t, mob.Character.Conditions.List, fadeConditionId)
	darken(t, 1)
	drainPlain(2)
	PruneConditions(events.NewTurn{TurnNumber: 2})
	assert.Equal(t, 0, countContaining(drainPlain(2), "fades."))
}

// TestMobConditionTriggerRoomText is the D4 guard. The player round tick has always
// sent a triggered condition's trigger_room_text; tickMobConditions never did, so a mob
// holding a trigger-text condition showed nothing. No mob holder of the shipped
// trigger-text conditions could be staged in a playtest, so this is its only check.
func TestMobConditionTriggerRoomText(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	mob := mobs.GetInstance(100)
	require.True(t, mob.Character.Conditions.AddCondition(shiverConditionId, false))
	events.DrainQueuedMessagesForTest(2)

	tickMobConditions(mob, 100)
	raw := events.DrainQueuedMessagesForTest(2)
	line := rawLineContaining(raw, "Skeleton shivers.")
	require.NotEmpty(t, line, "a sighted observer must see the mob's trigger text")
	assert.Contains(t, line, `fg="mobname`)
	delivered := 0
	for _, l := range raw {
		if strings.Contains(plainText(l), "Skeleton shivers.") {
			delivered++
		}
	}
	assert.Equal(t, 1, delivered, "the trigger line must arrive exactly once")

	// Sight-gated like every other condition room line.
	darken(t, 1)
	drainPlain(2)
	tickMobConditions(mob, 100)
	assert.Equal(t, 0, countContaining(drainPlain(2), "shivers."))
}

// A light condition's end line describes the light going out, and the moment a
// light goes out is seen by everyone in the room with working eyes. But the
// light stops counting the instant the condition EXPIRES (Conditions.HasFlag skips
// expired conditions, and expiry happens on the round tick), while its end text is
// sent later, at the turn's prune. So a plain visual send judged sight in a
// room that was already dark, and silenced the line for exactly the people
// who had been seeing by that light. Found by the Task 2 review against
// shipped condition 1, Illumination.

func TestConditionEndRoomText_LightConditionEndIsSeenByItsOwnLight_Player(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	darken(t, 1)
	room := rooms.LoadRoom(1)
	holder := users.GetByUserId(1)
	require.True(t, holder.Character.Conditions.AddCondition(lanternConditionId, false))
	require.GreaterOrEqual(t, room.GetVisibility(), 1, "the lantern must light the cave, or this test proves nothing")
	expire(t, holder.Character.Conditions.List, lanternConditionId)
	require.Zero(t, room.GetVisibility(), "the light is already out once the condition expires, before any prune")
	drainPlain(2)

	PruneConditions(events.NewTurn{TurnNumber: 1})
	assert.Equal(t, 1, countContaining(drainPlain(2), "Aliceia's light gutters out."))
}

// TestConditionEndRoomText_LightConditionEnd_SleeperStillGetsNothing proves the light
// line is still a SIGHT line, not audio: an observer who cannot see for a
// reason other than darkness is not told.
func TestConditionEndRoomText_LightConditionEnd_SleeperStillGetsNothing(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	darken(t, 1)
	holder := users.GetByUserId(1)
	require.True(t, holder.Character.Conditions.AddCondition(lanternConditionId, false))
	require.True(t, users.GetByUserId(2).Character.Conditions.AddCondition(dozeConditionId, true))
	expire(t, holder.Character.Conditions.List, lanternConditionId)
	drainPlain(2)

	PruneConditions(events.NewTurn{TurnNumber: 1})
	assert.Equal(t, 0, countContaining(drainPlain(2), "light gutters out"))
}

func TestConditionEndRoomText_LightConditionEndIsSeenByItsOwnLight_Mob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restore := seedNarrationConditions()
	defer restore()
	darken(t, 1)
	room := rooms.LoadRoom(1)
	mob := mobs.GetInstance(100)
	require.True(t, mob.Character.Conditions.AddCondition(lanternConditionId, false))
	require.GreaterOrEqual(t, room.GetVisibility(), 1, "the lantern must light the cave, or this test proves nothing")
	expire(t, mob.Character.Conditions.List, lanternConditionId)
	drainPlain(2)

	PruneConditions(events.NewTurn{TurnNumber: 1})
	assert.Equal(t, 1, countContaining(drainPlain(2), "Skeleton's light gutters out."))
}
