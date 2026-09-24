package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/worldevents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rendingAfterDeathConditionId is a tick record shaped like 115 Rending Bleed
// (one-round trigger rate, four triggers), clear of the other hooks fixtures.
const rendingAfterDeathConditionId = 7110

// setupConditionAfterDeath seeds the registries and a tick record, and gives user 1
// a real Life machine carrying the production death wiring, so Die runs the
// same Alive -> Dead -> Respawning -> Alive sequence the game does.
//
// The wiring comes from the character's first Validate, which fires the
// OnCharacterCreated callbacks exactly once, in production order. Wiring the
// cascade by hand as well would register it twice, because any later Validate
// (every condition add runs one) fires the callbacks anyway.
func setupConditionAfterDeath(t *testing.T) *users.UserRecord {
	t.Helper()
	t.Cleanup(seedAllRegistries())
	// One seed call: each SeedConditionsForTest replaces the registry, so a second
	// call would drop the tick record.
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		rendingAfterDeathConditionId: {ConditionId: rendingAfterDeathConditionId, Name: "Test Rending Bleed",
			RoundInterval: 1, TriggerCount: 4, StartUserText: "Your wounds tear open."},
		deathProtectionConditionId: {ConditionId: deathProtectionConditionId, Name: "Death Protection",
			TriggerCount: 1000000, Flags: []conditions.Flag{conditions.ReviveOnDeath}},
	}))

	u := users.GetByUserId(1)
	require.NotNil(t, u)
	u.Character.Life = life.NewMachine()
	// Validate recalculates stats and pools from Base, and the test user
	// carries only Value/ValueAdj, so give it real bases or its health max
	// floors at 1 and any hit kills.
	seedRacialStats(u, 0)
	u.Character.HealthMax.Base = 200
	u.Character.StaminaMax.Base = 100
	u.Character.ConvictionMax.Base = 50
	_ = u.Character.Validate()
	u.Character.Health = u.Character.HealthMax.Value
	// The real death announcement is part of that wiring and emits a PvE
	// death into the shared world event feed, which the gossip tests read.
	t.Cleanup(worldevents.ResetForTest)

	// The message queue is package-global: drain every seeded user, or a
	// line an earlier test sent to the room is read as this test's.
	events.DrainQueuedCharacterDiedForTest()
	events.DrainQueuedConditionsForTest(0)
	drainPlain(1)
	drainPlain(2)
	return u
}

// A condition queued in the same combat round as the killing blow must not land on
// the respawned player.
//
// The playtest (run 7d0dad99c4709fc0): a scarred steppe wolf's rending-claws
// on-hit condition was queued by applyCombatDamageBonuses AFTER the swing's own
// ApplyHarm had queued the CharacterDied. The queue is FIFO within a priority,
// so RouteAttributedDeath ran first: Die stripped every condition and cascaded the
// player all the way back to Alive at 5% health, clearing DeathQueued. Only
// then did ApplyConditions apply a fresh Rending Bleed, which bled the player to a
// second death in the Mending Hut.
//
// This replays that queue order through the real producer (UserRecord.AddCondition),
// the real death listener and the real condition listener.
func TestApplyConditions_ConditionQueuedBeforeDeathDoesNotLandAfterRespawn(t *testing.T) {
	u := setupConditionAfterDeath(t)

	// The killing blow, then the on-hit condition, in the order the combat round
	// queues them.
	u.Character.ApplyHarm(characters.PoolHealth, u.Character.HealthMax.Value+100, state.ActorRef{MobInstanceId: 100})
	u.AddCondition(rendingAfterDeathConditionId, "mutation")

	died := events.DrainQueuedCharacterDiedForTest()
	require.Len(t, died, 1, "the lethal harm must queue the death")
	queuedConditions := events.DrainQueuedConditionsForTest(1)
	require.Len(t, queuedConditions, 1, "the on-hit condition must be queued")

	RouteAttributedDeath(died[0])

	// Why neither "is alive" nor "has a death queued" can be the refusal on
	// its own: by the time the condition event flushes, the player has already
	// respawned and the death token is spent.
	require.True(t, u.Character.IsAlive(), "precondition: Die cascades a player back to Alive")
	require.False(t, u.Character.DeathQueued, "precondition: the death token is cleared")
	drainPlain(1)

	assert.Equal(t, events.Continue, ApplyConditions(queuedConditions[0]))
	assert.False(t, u.Character.HasCondition(rendingAfterDeathConditionId),
		"a condition aimed at the life that just ended must not land on the respawned player")
	assert.Equal(t, 0, countContaining(drainPlain(1), "Your wounds tear open."),
		"and the respawned player must not be told it took hold")

	// A condition queued AFTER the respawn is aimed at the new life and lands. This
	// is the path every respawn-time condition takes (the room mutator conditions the
	// respawn teleport queues on arrival).
	u.AddCondition(rendingAfterDeathConditionId, "area")
	fresh := events.DrainQueuedConditionsForTest(1)
	require.Len(t, fresh, 1)
	assert.Equal(t, events.Continue, ApplyConditions(fresh[0]))
	assert.True(t, u.Character.HasCondition(rendingAfterDeathConditionId),
		"a condition queued after the respawn must still apply")
}

// Control: the same producer and listener, with no death in between, applies
// and narrates. Without this the test above could pass because the fixture
// never applies anything at all.
func TestApplyConditions_ConditionQueuedOnALivingPlayerApplies(t *testing.T) {
	u := setupConditionAfterDeath(t)

	u.Character.ApplyHarm(characters.PoolHealth, 10, state.ActorRef{MobInstanceId: 100})
	u.AddCondition(rendingAfterDeathConditionId, "mutation")

	require.Empty(t, events.DrainQueuedCharacterDiedForTest(), "a survivable hit queues no death")
	queuedConditions := events.DrainQueuedConditionsForTest(1)
	require.Len(t, queuedConditions, 1)

	assert.Equal(t, events.Continue, ApplyConditions(queuedConditions[0]))
	assert.True(t, u.Character.HasCondition(rendingAfterDeathConditionId))
	assert.Equal(t, 1, countContaining(drainPlain(1), "Your wounds tear open."))
}

// A queued death that a ReviveOnDeath condition turns into a revive never ends the
// life, so a condition from the same blow still lands on the revived character.
// This is why the refusal is not "a death was queued".
func TestApplyConditions_ConditionQueuedBeforeAReviveStillApplies(t *testing.T) {
	u := setupConditionAfterDeath(t)
	require.NoError(t, u.Character.AddCondition(deathProtectionConditionId, true))

	u.Character.ApplyHarm(characters.PoolHealth, u.Character.HealthMax.Value+100, state.ActorRef{MobInstanceId: 100})
	u.AddCondition(rendingAfterDeathConditionId, "mutation")

	died := events.DrainQueuedCharacterDiedForTest()
	require.Len(t, died, 1)
	queuedConditions := events.DrainQueuedConditionsForTest(1)
	require.Len(t, queuedConditions, 1)

	RouteAttributedDeath(died[0])
	require.False(t, u.Character.HasConditionFlag(conditions.ReviveOnDeath), "precondition: the revive fired")

	assert.Equal(t, events.Continue, ApplyConditions(queuedConditions[0]))
	assert.True(t, u.Character.HasCondition(rendingAfterDeathConditionId),
		"the revived character never died, so the blow's condition still lands")
}
