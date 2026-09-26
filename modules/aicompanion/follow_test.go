package aicompanion

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
)

// The engine asks the hold once per companion. Only the bonded companion
// this module drives is ever held: an ordinary charmed companion of the same
// sneaking owner follows as it always did.
func TestHoldFollowHoldsOnlyTheBondedCompanion(t *testing.T) {
	owner, _, room, her := harmWorld(t, configs.PVPDisabled)
	wolf := harmMob(t, room, 300, `a charmed wolf`)

	owner.Character.Awareness = awareness.NewMachine()
	r := state.TransitionReason{Trigger: `test`}
	if err := owner.Character.Awareness.TransitionToConcealing(awareness.ConcealingData{}, r); err != nil {
		t.Fatalf("could not start concealing: %v", err)
	}
	owner.Character.Awareness.ResolveConcealment(true, r)
	if !owner.Character.IsHidden() {
		t.Fatal("fixture: the owner must be sneaking")
	}

	m := &AICompanionModule{
		cfg:   Config{Enabled: true, HoldWhenSneaking: true},
		ctrls: map[int]*controller{owner.UserId: {ownerUserId: owner.UserId, instanceId: her.InstanceId}},
	}
	if !m.holdFollow(owner.UserId, her.InstanceId) {
		t.Fatal("a sneaking owner's bonded companion stays put")
	}
	if m.holdFollow(owner.UserId, wolf.InstanceId) {
		t.Fatal("an ordinary companion of the same owner is not the module's to hold")
	}
}
