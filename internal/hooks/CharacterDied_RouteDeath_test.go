package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// deathProtectionConditionId is the condition carrying the ReviveOnDeath flag,
// _datafiles/world/default/buffs/35-death_protection.yaml.
const deathProtectionConditionId = 35

// newRouteDeathTestMob builds a mob and registers it in the instance registry
// so mobs.GetInstance can resolve it, restoring the registry on cleanup. A mob
// that is NOT registered cannot be resolved, and the test would pass for the
// wrong reason.
func newRouteDeathTestMob(t *testing.T, health int) *mobs.Mob {
	t.Helper()

	m := &mobs.Mob{
		MobId:      1,
		InstanceId: 90001,
		HomeRoomId: 1,
		Character: characters.Character{
			Name:       "Route-Death-Dummy",
			RoomId:     1,
			Health:     health,
			Conditions: conditions.New(),
			Cooldowns:  map[string]int{},
		},
	}
	m.Character.HealthMax.Base = 100
	m.Character.HealthMax.Recalculate()

	cleanup := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{m.InstanceId: m})
	t.Cleanup(cleanup)

	return m
}

// seedReviveCondition registers a condition spec carrying the ReviveOnDeath flag.
func seedReviveCondition(t *testing.T) {
	t.Helper()
	cleanup := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		deathProtectionConditionId: {
			ConditionId:  deathProtectionConditionId,
			Name:         "Death Protection",
			TriggerCount: 1000000,
			Flags:        []conditions.Flag{conditions.ReviveOnDeath},
		},
	})
	t.Cleanup(cleanup)
}

// A queued death is only valid while DeathQueued is still set. Anything that
// resolves a character's life state out of band clears it, which makes the
// stale event inert.
//
// The race this closes: a player takes a lethal hit (DeathQueued set, event
// queued, still IsAlive until the flush), then runs `suicide` in that window.
// With ReviveOnDeath they are healed and the condition is CONSUMED. The stale event
// then flushed, found them alive with no condition left, and killed them anyway —
// real corpse, real bounty, gold to the original killer, for a player who was
// healthy a moment earlier. The condition exists precisely to prevent that.
//
// !IsAlive() cannot catch this: the out-of-band resolution leaves them ALIVE.
func TestRouteAttributedDeath_StaleEventIsInertOnceDeathQueuedIsCleared(t *testing.T) {
	mob := newRouteDeathTestMob(t, 100) // healed back up out of band
	mob.Character.Life = life.NewMachine()
	mob.Character.DeathQueued = false // whatever resolved them cleared it

	RouteAttributedDeath(events.CharacterDied{
		MobInstanceId:       mob.InstanceId,
		KillerMobInstanceId: 4242,
		Trigger:             life.TriggerHealthZero,
	})

	if !mob.Character.IsAlive() {
		t.Fatal("a stale queued death killed a character who was already resolved")
	}
}

// Die clears the token itself, so every Die caller — the backstops, the suicide
// commands, the listener — invalidates any event still in flight.
func TestDie_ClearsDeathQueued(t *testing.T) {
	mob := newRouteDeathTestMob(t, -5)
	mob.Character.Life = life.NewMachine()
	mob.Character.DeathQueued = true

	mob.Character.Die(state.ActorRef{MobInstanceId: 1}, life.TriggerHealthZero)

	if mob.Character.DeathQueued {
		t.Error("Die did not clear DeathQueued; a stale event could still fire")
	}
}

func TestRouteAttributedDeath_WrongEventTypeIsRejected(t *testing.T) {
	if got := RouteAttributedDeath(events.Condition{}); got != events.Cancel {
		t.Errorf("got %v, want Cancel for a mismatched event type", got)
	}
}

func TestRouteAttributedDeath_UnknownVictimIsInert(t *testing.T) {
	got := RouteAttributedDeath(events.CharacterDied{MobInstanceId: 999999})
	if got != events.Continue {
		t.Errorf("got %v, want Continue when the victim is already gone", got)
	}
}

// ReviveOnDeath must heal, cancel the condition, clear DeathQueued and NOT die.
// Leaving DeathQueued set would make the character permanently unkillable;
// leaving health negative would just hand the kill to the sweep next tick.
func TestRouteAttributedDeath_ReviveHealsAndClearsQueue(t *testing.T) {
	seedReviveCondition(t)
	mob := newRouteDeathTestMob(t, -20)
	if err := mob.Character.AddCondition(deathProtectionConditionId, true); err != nil {
		t.Fatalf("AddCondition: %v", err)
	}

	if !mob.Character.HasConditionFlag(conditions.ReviveOnDeath) {
		t.Fatal("precondition: condition did not apply the ReviveOnDeath flag")
	}
	mob.Character.DeathQueued = true

	RouteAttributedDeath(events.CharacterDied{MobInstanceId: mob.InstanceId})

	if !mob.Character.IsAlive() {
		t.Error("revive did not prevent the death")
	}
	if mob.Character.Health < 1 {
		t.Errorf("health = %d, want positive — a revived character left dying is reaped by the sweep",
			mob.Character.Health)
	}
	if mob.Character.DeathQueued {
		t.Error("DeathQueued still set after a revive; the character can never be killed again")
	}
	if mob.Character.HasConditionFlag(conditions.ReviveOnDeath) {
		t.Error("revive condition was not consumed")
	}
}
