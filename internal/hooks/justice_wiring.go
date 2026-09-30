package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/justice"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// init wires justice's guard-speech seam to the actions-based broadcaster. It
// lives in hooks (not justice) so package justice imports no actions, keeping
// the actions->justice direction (theft bounty firing) cycle-free.
func init() {
	justice.SetGuardSay(func(room *rooms.Room, mob *mobs.Mob, line string) {
		if mob == nil || room == nil {
			return
		}
		// actions.Say sends the room line itself (sight gates slice 5b).
		actions.Say(&actions.MobActor{Mob: mob, Room: room}, line)
	})
}
