package seeders

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// setGoalsTempDirForTest points DOGMUD_GOALS_DIR_OVERRIDE at a per-test
// temp directory so goals.Add / goals.GoalsOf persist to a throwaway
// location instead of a real datafiles tree. Mirrors the same pattern in
// internal/hooks/NewRound_MobRoundTick_RecomputeGoals_test.go. Returns a
// cleanup func restoring the previous value.
func setGoalsTempDirForTest(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	prev := os.Getenv("DOGMUD_GOALS_DIR_OVERRIDE")
	os.Setenv("DOGMUD_GOALS_DIR_OVERRIDE", filepath.Join(dir, "goals"))
	t.Cleanup(func() {
		os.Setenv("DOGMUD_GOALS_DIR_OVERRIDE", prev)
	})
}

// hasRevengeGoal reports whether the mob has a "revenge-mob" goal
// targeting (targetKind, targetId).
func hasRevengeGoal(t *testing.T, mob *mobs.Mob, targetKind string, targetId int) bool {
	t.Helper()
	name := mob.Character.Name
	for _, g := range goals.GoalsOf(int(mob.MobId), name) {
		if g.Type != "revenge-mob" {
			continue
		}
		kind, _ := g.Params["target_kind"].(string)
		id, _ := g.Params["target_id"].(int)
		if kind == targetKind && id == targetId {
			return true
		}
	}
	return false
}

func TestClassifyWitnessResponse(t *testing.T) {
	guard := &mobs.Mob{Groups: []string{"humanoid", "guard"}}
	if got := classifyWitnessResponse(guard); got != ResponseReportOnly {
		t.Fatalf("guard: want ResponseReportOnly, got %v", got)
	}

	civilian := &mobs.Mob{Groups: []string{"humanoid"}}
	civilian.Character.NonCombatant = true
	if got := classifyWitnessResponse(civilian); got != ResponseAlarm {
		t.Fatalf("noncombatant: want ResponseAlarm, got %v", got)
	}

	fighter := &mobs.Mob{Groups: []string{"humanoid"}}
	if got := classifyWitnessResponse(fighter); got != ResponseRevenge {
		t.Fatalf("combat mob: want ResponseRevenge, got %v", got)
	}

	if got := classifyWitnessResponse(nil); got != ResponseReportOnly {
		t.Fatalf("nil: want ResponseReportOnly (safe), got %v", got)
	}

	ncGuard := &mobs.Mob{Groups: []string{"guard"}}
	ncGuard.Character.NonCombatant = true
	if got := classifyWitnessResponse(ncGuard); got != ResponseReportOnly {
		t.Fatalf("noncombat guard: want ResponseReportOnly, got %v", got)
	}
}

// TestSeedShapesOnlyWitnessResponse_Noncombatant_AlarmNoRevenge covers a
// shapes-only witness that classifies ResponseAlarm: it must not seed a
// revenge goal. (alarmReaction itself has no observable seam beyond the
// mob.Command queue call, which is exercised for panics only here; see
// the fighter case below for the assertion that actually distinguishes
// the two tiers.)
func TestSeedShapesOnlyWitnessResponse_Noncombatant_AlarmNoRevenge(t *testing.T) {
	setGoalsTempDirForTest(t)
	mob := &mobs.Mob{MobId: mobs.MobId(99101), Groups: []string{"humanoid"}}
	mob.Character.Name = "shapes_noncombatant"
	mob.Character.NonCombatant = true

	if got := classifyWitnessResponse(mob); got != ResponseAlarm {
		t.Fatalf("fixture sanity: want ResponseAlarm, got %v", got)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("seedShapesOnlyWitnessResponse panicked: %v", r)
		}
	}()
	seedShapesOnlyWitnessResponse(mob)

	if hasRevengeGoal(t, mob, "player", 5) {
		t.Errorf("shapes-only noncombatant must not get a revenge goal")
	}
}

// TestSeedShapesOnlyWitnessResponse_Fighter_AlarmNoRevenge is the ruling's
// actual content: a shapes-only witness that WOULD classify ResponseRevenge
// (a combat-capable non-guard) still must not seed a revenge goal, because
// it never identified who to hunt. The fixture is asserted to really
// classify ResponseRevenge first, so a misbuilt fixture fails loudly
// instead of passing this test for the wrong reason.
func TestSeedShapesOnlyWitnessResponse_Fighter_AlarmNoRevenge(t *testing.T) {
	setGoalsTempDirForTest(t)
	mob := &mobs.Mob{MobId: mobs.MobId(99102), Groups: []string{"humanoid"}}
	mob.Character.Name = "shapes_fighter"

	if got := classifyWitnessResponse(mob); got != ResponseRevenge {
		t.Fatalf("fixture sanity: want ResponseRevenge, got %v", got)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("seedShapesOnlyWitnessResponse panicked: %v", r)
		}
	}()
	seedShapesOnlyWitnessResponse(mob)

	if hasRevengeGoal(t, mob, "player", 5) {
		t.Errorf("shapes-only fighter must not get a revenge goal naming the player by ID")
	}
}

// TestSeedWitnessResponse_IdentifyingFighter_StillGetsRevengeGoal proves
// the shapes-only gate did not disarm the identifying tier: an identifying
// fighter goes through the unchanged seedWitnessResponse path and still
// gets its revenge goal.
func TestSeedWitnessResponse_IdentifyingFighter_StillGetsRevengeGoal(t *testing.T) {
	setGoalsTempDirForTest(t)
	mob := &mobs.Mob{MobId: mobs.MobId(99103), Groups: []string{"humanoid"}}
	mob.Character.Name = "identifying_fighter"

	if got := classifyWitnessResponse(mob); got != ResponseRevenge {
		t.Fatalf("fixture sanity: want ResponseRevenge, got %v", got)
	}

	seedWitnessResponse(mob, 5, aggressiveWitnessRevengePriority)

	if !hasRevengeGoal(t, mob, "player", 5) {
		t.Errorf("identifying fighter should still get a revenge goal targeting the player")
	}
}

// TestSeedShapesOnlyWitnessResponse_Guard_NoOpBothTiers covers the ruling's
// explicit carve-out: a guard is a no-op in BOTH tiers, because a personal
// reaction would derail enforcement regardless of what the guard saw.
func TestSeedShapesOnlyWitnessResponse_Guard_NoOpBothTiers(t *testing.T) {
	setGoalsTempDirForTest(t)
	guard := &mobs.Mob{MobId: mobs.MobId(99104), Groups: []string{"humanoid", "guard"}}
	guard.Character.Name = "shapes_guard"

	if got := classifyWitnessResponse(guard); got != ResponseReportOnly {
		t.Fatalf("fixture sanity: want ResponseReportOnly, got %v", got)
	}

	seedWitnessResponse(guard, 5, aggressiveWitnessRevengePriority)
	seedShapesOnlyWitnessResponse(guard)

	if hasRevengeGoal(t, guard, "player", 5) {
		t.Errorf("guard must not get a revenge goal in either tier")
	}
}
