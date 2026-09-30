package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Shout shouts for a mob through actions.Shout: a hidden mob is revealed, the
// room hears its name by each listener's sight, next door hears the words,
// and the room's sleepers wake (sight gates slice 5b).
func Shout(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actions.Shout(&actions.MobActor{Mob: mob, Room: room}, rest)
	return true, nil
}
