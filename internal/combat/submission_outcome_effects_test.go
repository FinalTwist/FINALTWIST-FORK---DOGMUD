package combat_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// internal/combat sends no player text anywhere in the package, so conditions 83
// Broken Limb and 84 Stunned are applied synchronously and silently here and
// the submission hook owes the victim their authored start line. It can only
// do that if the resolver says what it applied, which is what
// SubmissionOutcomeEffects is for. These lanes pin that report.
//
// The registry has to be seeded: Character.AddCondition fails for a condition id with no
// spec, and the report is gated on the apply actually landing, so an unseeded
// binary would read as "nothing applied" and pass for the wrong reason.
func seedSubmissionConditionSpecs(t *testing.T) func() {
	t.Helper()
	return conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		combat.BrokenLimbConditionId: {
			ConditionId:   combat.BrokenLimbConditionId,
			Name:          "Broken Limb",
			RoundInterval: 1,
			TriggerCount:  900,
		},
		combat.StunnedConditionId: {
			ConditionId:   combat.StunnedConditionId,
			Name:          "Stunned",
			RoundInterval: 1,
			TriggerCount:  1,
		},
	})
}

func TestResolveSubmissionOutcome_ReportsTheStunItApplied(t *testing.T) {
	restore := seedSubmissionConditionSpecs(t)
	defer restore()

	atk, def := setupMountForOutcome(t)
	atk.SubmissionPolicy = characters.PolicyMercy
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierCrit,
		SubType: position.SubArmbar,
	}

	effects := combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)

	require.True(t, def.HasCondition(combat.StunnedConditionId),
		"crit + mercy must still apply the stun")
	assert.Same(t, def, effects.StunnedVictim,
		"the resolver must name the stunned victim so the caller can narrate it")
	assert.Nil(t, effects.BrokenLimbVictim, "no limb was broken on a mercy release")
}

func TestResolveSubmissionOutcome_ReportsTheBrokenLimbItApplied(t *testing.T) {
	restore := seedSubmissionConditionSpecs(t)
	defer restore()

	// A mob defender so the death cascade stops at Dead; a player would run
	// Dead → Respawning → Alive synchronously.
	atk, def := setupMountWithMobDefender(t)
	atk.SubmissionPolicy = characters.PolicyCripple
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierSuccess,
		SubType: position.SubArmbar,
	}

	effects := combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)

	require.True(t, def.HasCondition(combat.BrokenLimbConditionId),
		"cripple on a joint sub must still apply the broken limb")
	assert.Same(t, def, effects.BrokenLimbVictim,
		"the resolver must name the broken-limb victim so the caller can narrate it")
	assert.Equal(t, "arm", effects.BrokenBodyPart)
	assert.Nil(t, effects.StunnedVictim, "a success tier is not a crit")
}

func TestResolveSubmissionOutcome_ReportsNothingWhenNothingWasApplied(t *testing.T) {
	restore := seedSubmissionConditionSpecs(t)
	defer restore()

	atk, def := setupMountForOutcome(t)
	atk.SubmissionPolicy = characters.PolicyMercy
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierNeutral,
		SubType: position.SubArmbar,
	}

	effects := combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)

	assert.Nil(t, effects.StunnedVictim)
	assert.Nil(t, effects.BrokenLimbVictim)
	assert.Empty(t, effects.BrokenBodyPart)
	assert.False(t, def.HasCondition(combat.StunnedConditionId))
}

// A choke degrades cripple to subdue because a choke breaks nothing, so the
// report must stay empty even though the policy asked for a break.
func TestResolveSubmissionOutcome_ReportsNoBreakForAChokeCripple(t *testing.T) {
	restore := seedSubmissionConditionSpecs(t)
	defer restore()

	atk, def := setupMountWithMobDefender(t)
	atk.SubmissionPolicy = characters.PolicyCripple
	result := combat.SubmissionAttemptResult{
		Tier:    combat.SubTierSuccess,
		SubType: position.SubRNC,
	}

	effects := combat.ResolveSubmissionOutcome(atk, def, result, combat.RoleTop)

	assert.Nil(t, effects.BrokenLimbVictim, "a choke breaks no limb")
	assert.False(t, def.HasCondition(combat.BrokenLimbConditionId))
}
