package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Drink is the player command; the rules live in actions.Drink (drink path
// unification), shared with mobs and the AI companion.
func Drink(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	actions.Drink(actions.NewUserActorInRoom(user, room).(actions.DrinkActor), rest)
	return true, nil
}
