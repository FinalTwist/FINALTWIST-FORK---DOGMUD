package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Who(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Refused exactly where look is: a viewer who makes out nothing here has
	// no roster to read. At shapes GetDetails lists anonymous figures.
	if messaging.ParticipantSight(user.Character, room) == messaging.SightNone {
		user.SendText(messaging.CategorySystem, `You can't see anything!`)
		return true, nil
	}

	details := rooms.GetDetails(room, user)

	whoTxt, _ := templates.Process("descriptions/who", details, user.UserId)
	user.SendText(messaging.CategorySystem, whoTxt)

	return true, nil
}
