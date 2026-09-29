package mobcommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Flee starts a mob's flee through the player's rules (actions.BeginFlee):
// it must be fighting, standing, unrooted and not frenzied, it pays the flee
// cost, and it enters Disengaging. The escape resolves on the next round in
// hooks.handleMobFlee. rest, when given, is the exit the mob would rather
// take (a kiting archer passes the exit toward home).
func Flee(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	// Non-combatant mobs never flee (they should never be in combat).
	if mob.IsNonCombatant() {
		return true, nil
	}

	begin := actions.BeginFlee(actions.NewMobActorInRoom(mob, room), strings.TrimSpace(rest))

	// Every refusal is silent for a mob except the grapple: the player holding
	// it needs to see the hold working.
	if begin.Refusal == actions.FleeRefuseGrappled {
		room.SendTextVisual(messaging.CategoryGrappleFlow,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to break free but you've got them locked down!`, mob.Character.Name))
	}
	return true, nil
}
