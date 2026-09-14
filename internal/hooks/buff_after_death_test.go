package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rendingAfterDeathBuffId is a tick record shaped like 115 Rending Bleed
// (one-round trigger rate, four triggers), clear of the other hooks fixtures.
const rendingAfterDeathBuffId = 7110

// setupBuffAfterDeath seeds the registries and a tick record, and gives user 1
// a real Life machine wired to the death cascade, so Die runs the same
// Alive -> Dead -> Respawning -> Alive sequence the game does.
func setupBuffAfterDeath(t *testing.T) *users.UserRecord {
	t.Helper()
	t.Cleanup(seedAllRegistries())
	// One seed call: each SeedBuffsForTest replaces the registry, so a second
	// call would drop the tick record.
	t.Cleanup(buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		rendingAfterDeathBuffId: {BuffId: rendingAfterDeathBuffId, Name: "Test Rending Bleed",
			RoundInterval: 1, TriggerCount: 4, StartUserText: "Your wounds tear open."},
		deathProtectionBuffId: {BuffId: deathProtectionBuffId, Name: "Death Protection",
			TriggerCount: 1000000, Flags: []buffs.Flag{buffs.ReviveOnDeath}},
	}))

	u := users.GetByUserId(1)
	require.NotNil(t, u)
	u.Character.Life = life.NewMachine()
	wireLifeCrossMachineCascades(u.Character)

	events.DrainQueuedCharacterDiedForTest()
	events.DrainQueuedBuffsForTest(0)
	drainPlain(1)
	return u
}

// A buff queued in the same combat round as the killing blow must not land on
// the respawned player.
//
// The playtest (run 7d0dad99c4709fc0): a scarred steppe wolf's rending-claws
// on-hit buff was queued by applyCombatDamageBonuses AFTER the swing's own
// ApplyHarm had queued the CharacterDied. The queue is FIFO within a priority,
// so RouteAttributedDeath ran first: Die stripped every buff and cascaded the
// player all the way back to Alive at 5% health, clearing DeathQueued. Only
// then did ApplyBuffs apply a fresh Rending Bleed, which bled the player to a
// second death in the Mending Hut.
//
// This replays that queue order through the real producer (UserRecord.AddBuff),
// the real death listener and the real buff listener.
func TestApplyBuffs_BuffQueuedBeforeDeathDoesNotLandAfterRespawn(t *testing.T) {
	u := setupBuffAfterDeath(t)

	// The killing blow, then the on-hit buff, in the order the combat round
	// queues them.
	u.Character.ApplyHarm(characters.PoolHealth, 500, state.ActorRef{MobInstanceId: 100})
	u.AddBuff(rendingAfterDeathBuffId, "mutation")

	died := events.DrainQueuedCharacterDiedForTest()
	require.Len(t, died, 1, "the lethal harm must queue the death")
	queuedBuffs := events.DrainQueuedBuffsForTest(1)
	require.Len(t, queuedBuffs, 1, "the on-hit buff must be queued")

	RouteAttributedDeath(died[0])

	// Why neither "is alive" nor "has a death queued" can be the refusal on
	// its own: by the time the buff event flushes, the player has already
	// respawned and the death token is spent.
	require.True(t, u.Character.IsAlive(), "precondition: Die cascades a player back to Alive")
	require.False(t, u.Character.DeathQueued, "precondition: the death token is cleared")
	drainPlain(1)

	assert.Equal(t, events.Continue, ApplyBuffs(queuedBuffs[0]))
	assert.False(t, u.Character.HasBuff(rendingAfterDeathBuffId),
		"a buff aimed at the life that just ended must not land on the respawned player")
	assert.Equal(t, 0, countContaining(drainPlain(1), "Your wounds tear open."),
		"and the respawned player must not be told it took hold")

	// A buff queued AFTER the respawn is aimed at the new life and lands. This
	// is the path every respawn-time buff takes (the room mutator buffs the
	// respawn teleport queues on arrival).
	u.AddBuff(rendingAfterDeathBuffId, "area")
	fresh := events.DrainQueuedBuffsForTest(1)
	require.Len(t, fresh, 1)
	assert.Equal(t, events.Continue, ApplyBuffs(fresh[0]))
	assert.True(t, u.Character.HasBuff(rendingAfterDeathBuffId),
		"a buff queued after the respawn must still apply")
}

// Control: the same producer and listener, with no death in between, applies
// and narrates. Without this the test above could pass because the fixture
// never applies anything at all.
func TestApplyBuffs_BuffQueuedOnALivingPlayerApplies(t *testing.T) {
	u := setupBuffAfterDeath(t)

	u.Character.ApplyHarm(characters.PoolHealth, 10, state.ActorRef{MobInstanceId: 100})
	u.AddBuff(rendingAfterDeathBuffId, "mutation")

	require.Empty(t, events.DrainQueuedCharacterDiedForTest(), "a survivable hit queues no death")
	queuedBuffs := events.DrainQueuedBuffsForTest(1)
	require.Len(t, queuedBuffs, 1)

	assert.Equal(t, events.Continue, ApplyBuffs(queuedBuffs[0]))
	assert.True(t, u.Character.HasBuff(rendingAfterDeathBuffId))
	assert.Equal(t, 1, countContaining(drainPlain(1), "Your wounds tear open."))
}

// A queued death that a ReviveOnDeath buff turns into a revive never ends the
// life, so a buff from the same blow still lands on the revived character.
// This is why the refusal is not "a death was queued".
func TestApplyBuffs_BuffQueuedBeforeAReviveStillApplies(t *testing.T) {
	u := setupBuffAfterDeath(t)
	require.NoError(t, u.Character.AddBuff(deathProtectionBuffId, true))

	u.Character.ApplyHarm(characters.PoolHealth, 500, state.ActorRef{MobInstanceId: 100})
	u.AddBuff(rendingAfterDeathBuffId, "mutation")

	died := events.DrainQueuedCharacterDiedForTest()
	require.Len(t, died, 1)
	queuedBuffs := events.DrainQueuedBuffsForTest(1)
	require.Len(t, queuedBuffs, 1)

	RouteAttributedDeath(died[0])
	require.False(t, u.Character.HasBuffFlag(buffs.ReviveOnDeath), "precondition: the revive fired")

	assert.Equal(t, events.Continue, ApplyBuffs(queuedBuffs[0]))
	assert.True(t, u.Character.HasBuff(rendingAfterDeathBuffId),
		"the revived character never died, so the blow's buff still lands")
}
