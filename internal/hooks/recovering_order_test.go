package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The Recovering record (118, attacks_cap 1) must still be live when DoCombat
// runs, which is after UserRoundTick returns (hook order in hooks.go). It used
// to be added by the stand attempt and expired by the same hook's buff tick,
// so a player never felt it while mobs always did (owner ruling 2026-09-14:
// make it bite). This drives the real round tick, not a direct add: the
// direct-add test in internal/combat passed the whole time the player path was
// inert.
func TestUserRoundTick_RecoveringIsLiveWhenCombatRuns(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer conditions.SeedConditionRecordsForTest()()

	u := users.GetByUserId(1)
	require.NotNil(t, u)
	require.NoError(t, u.Character.Validate())
	require.NoError(t, u.Character.Position.TransitionToProne(position.ProneData{MinRecoveryRounds: 2},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward}))

	UserRoundTick(events.NewRound{RoundNumber: 1})
	require.True(t, u.Character.IsProne(), "precondition: round 1 is inside the minimum recovery period")
	assert.Equal(t, 1.0, u.Character.Conditions.Effect(conditions.EffectAttacksCap),
		"round 1: the swing cap must be live after the round tick, where DoCombat reads it")

	UserRoundTick(events.NewRound{RoundNumber: 2})
	require.True(t, u.Character.IsProne(), "precondition: round 2 consumes the last minimum round")
	assert.Equal(t, 1.0, u.Character.Conditions.Effect(conditions.EffectAttacksCap),
		"round 2: last round's record expired, and this round's attempt re-added it")

	UserRoundTick(events.NewRound{RoundNumber: 3})
	require.False(t, u.Character.IsProne(), "precondition: nobody holds the player down, so round 3 is a free stand")
	// Reads the cap at the point DoCombat reads it: hook registration order in
	// internal/hooks/hooks.go is UserRoundTick, then MobRoundTick, then DoCombat.
	assert.Equal(t, 0.0, u.Character.Conditions.Effect(conditions.EffectAttacksCap),
		"round 3: stood up, so no cap carries into this round's combat")
}

// A dying player must not stand up mid-death. A lethal bleed/poison tick in
// the buff-trigger block above queues the death (ApplyHarm sets DeathQueued,
// Health < 1) before the recovery block now runs, so without a guard the
// moved block would send "You scramble to your feet!" and award progression
// to a character whose death is already in flight. MobRoundTick already skips
// a dying mob (NewRound_MobRoundTick.go, the Health <= 0 continue) before its
// own recovery step; the player path needs the same guard.
func TestUserRoundTick_DyingPlayerDoesNotScrambleToFeet(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer conditions.SeedConditionRecordsForTest()()

	u := users.GetByUserId(1)
	require.NotNil(t, u)
	require.NoError(t, u.Character.Validate())
	require.NoError(t, u.Character.Position.TransitionToProne(position.ProneData{MinRecoveryRounds: 0},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward}))

	require.NoError(t, u.Character.AddConditionMagnitude(conditions.ConditionIdBleeding, 1, -1000, "test"))
	u.Character.Health = 5

	drainPlain(u.UserId)

	UserRoundTick(events.NewRound{RoundNumber: 1})

	lines := drainPlain(u.UserId)
	assert.Equal(t, 0, countContaining(lines, "scramble"),
		"a dying player must not stand up mid-death")
	assert.Equal(t, 0, countContaining(lines, "attempts to stand"),
		"a dying player must not attempt to stand mid-death either")
	require.True(t, u.Character.IsProne(), "still prone: the recovery attempt must be skipped, not merely silent")
}
