package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tickAtApplyConditionId is a heal-over-time record shaped like 32 Vital
// Surge, clear of the other hooks fixtures. TickVariance is 0 so the amount
// is deterministic.
const tickAtApplyConditionId = 7120

// seedTickAtApply seeds the shared registries plus one tick_pool record, and
// gives user 1 a real health pool so a 10% tick at different scales is
// distinguishable (a 1-point pool floors every scale at the minimum of 1).
func seedTickAtApply(t *testing.T) {
	t.Helper()
	t.Cleanup(seedAllRegistries())
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		tickAtApplyConditionId: {ConditionId: tickAtApplyConditionId, Name: "Test Surge",
			RoundInterval: 1, TriggerCount: 5, TickPool: "health", TickPercent: 0.10,
			Flags: []conditions.Flag{conditions.SilentStart}},
	}))
	u := users.GetByUserId(1)
	require.NotNil(t, u)
	seedRacialStats(u, 0)
	u.Character.HealthMax.Base = 200
	_ = u.Character.Validate()
	require.GreaterOrEqual(t, u.Character.HealthMax.Value, 50, "precondition: a real player health pool")
	// Every condition add runs Validate, which rebuilds HealthMax from Base,
	// so the mob needs a base too or its pool floors at 1.
	m := mobs.GetInstance(100)
	require.NotNil(t, m)
	m.Character.HealthMax.Base = 200
	_ = m.Character.Validate()
	require.GreaterOrEqual(t, m.Character.HealthMax.Value, 50, "precondition: a real mob health pool")
}

// heldTickAmount returns the TickAmount of c's held record for conditionId.
func heldTickAmount(t *testing.T, c *characters.Character, conditionId int) int {
	t.Helper()
	held := c.GetConditions(conditionId)
	require.Len(t, held, 1, "the condition must be held exactly once")
	return held[0].TickAmount
}

// A tick_pool condition used to land with TickAmount 0: every applier set the
// amount AFTER queueing the event, when the record did not exist yet, and the
// round tick's fallback later computed it at scale 1.0. The apply hook now
// computes it where the condition lands, at the event's scale, on a fresh
// application and on a refresh alike.
func TestApplyConditions_TickAmountComputedAtApply(t *testing.T) {
	seedTickAtApply(t)

	holders := []struct {
		name string
		evt  func(scale float64) events.Condition
		char func() *characters.Character
	}{
		{"player", func(s float64) events.Condition {
			return events.Condition{UserId: 1, ConditionId: tickAtApplyConditionId, TickScale: s}
		}, func() *characters.Character { return users.GetByUserId(1).Character }},
		{"mob", func(s float64) events.Condition {
			return events.Condition{MobInstanceId: 100, ConditionId: tickAtApplyConditionId, TickScale: s}
		}, func() *characters.Character { return &mobs.GetInstance(100).Character }},
	}

	for _, h := range holders {
		t.Run(h.name, func(t *testing.T) {
			c := h.char()
			c.RemoveCondition(tickAtApplyConditionId)
			require.Empty(t, c.GetConditions(tickAtApplyConditionId), "precondition: a fresh application")

			rows := []struct {
				name      string
				scale     float64
				wantScale float64
			}{
				{"fresh at 2.0", 2.0, 2.0},
				{"refresh at 3.0", 3.0, 3.0},
				{"refresh with no scale is 1.0", 0, 1.0},
			}
			for _, row := range rows {
				assert.Equal(t, events.Continue, ApplyConditions(h.evt(row.scale)))
				want := conditions.ComputeTickAmount(c.HealthMax.Value, 0.10, 0, 0, row.wantScale)
				assert.Equal(t, want, heldTickAmount(t, c, tickAtApplyConditionId), row.name)
			}
			assert.NotEqual(t,
				conditions.ComputeTickAmount(c.HealthMax.Value, 0.10, 0, 0, 1.0),
				conditions.ComputeTickAmount(c.HealthMax.Value, 0.10, 0, 0, 2.0),
				"precondition: the pool is large enough that the scale shows")
		})
	}
}
