package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Drink is the mob command; the rules live in actions.Drink, the same body a
// player drinks through.
func Drink(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actions.Drink(actions.NewMobActorInRoom(mob, room).(actions.DrinkActor), rest)
	return true, nil
}
