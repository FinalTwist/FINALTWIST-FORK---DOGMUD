package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
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
	defer buffs.SeedConditionRecordsForTest()()

	u := users.GetByUserId(1)
	require.NotNil(t, u)
	require.NoError(t, u.Character.Validate())
	require.NoError(t, u.Character.Position.TransitionToProne(position.ProneData{MinRecoveryRounds: 2},
		state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward}))

	UserRoundTick(events.NewRound{RoundNumber: 1})
	require.True(t, u.Character.IsProne(), "precondition: round 1 is inside the minimum recovery period")
	assert.Equal(t, 1.0, u.Character.Buffs.Effect(buffs.EffectAttacksCap),
		"round 1: the swing cap must be live after the round tick, where DoCombat reads it")

	UserRoundTick(events.NewRound{RoundNumber: 2})
	require.True(t, u.Character.IsProne(), "precondition: round 2 consumes the last minimum round")
	assert.Equal(t, 1.0, u.Character.Buffs.Effect(buffs.EffectAttacksCap),
		"round 2: last round's record expired, and this round's attempt re-added it")

	UserRoundTick(events.NewRound{RoundNumber: 3})
	require.False(t, u.Character.IsProne(), "precondition: nobody holds the player down, so round 3 is a free stand")
	assert.Equal(t, 0.0, u.Character.Buffs.Effect(buffs.EffectAttacksCap),
		"round 3: stood up, so no cap carries into this round's combat")

	u.Character.RemoveBuff(buffs.BuffIdRecovering)
}
