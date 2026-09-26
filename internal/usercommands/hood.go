package usercommands

import (
	"fmt"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// hoodedLight returns the adjustable light records of the item in the user's
// light slot: the hooded lantern's (lighting plan 5a).
func hoodedLight(user *users.UserRecord) []*conditions.Condition {
	lightItem := user.Character.Equipment.Light
	if lightItem.ItemId < 1 {
		return nil
	}
	var recs []*conditions.Condition
	for _, id := range lightItem.GetSpec().WornConditionIds {
		spec := conditions.GetConditionSpec(id)
		if spec == nil || !spec.IsLightSource() || !slices.Contains(spec.Flags, conditions.Adjustable) {
			continue
		}
		recs = append(recs, user.Character.Conditions.GetConditions(id)...)
	}
	return recs
}

// Hood closes the hood of the lantern in the light slot: it stays lit and
// held, and sheds no light until unhooded or re-equipped.
func Hood(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	recs := hoodedLight(user)
	if len(recs) == 0 {
		user.SendText(messaging.CategorySystem, `You have no hooded light to close.`)
		return true, nil
	}
	if recs[0].Hooded {
		user.SendText(messaging.CategorySystem, `Your lantern is already hooded.`)
		return true, nil
	}
	for _, rec := range recs {
		rec.Hooded = true
	}
	user.SendText(messaging.CategorySystem, `You lower the hood over your lantern, and its light narrows to nothing.`)
	if room != nil {
		// Judged as if lit: the hood is the end of a light, and the room may
		// already be dark by the time this line goes out, which would silence
		// it for everyone who was seeing by the lantern.
		room.SendTextVisualAsLit(messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> lowers the hood of a lantern, and the light around them dies away.`, user.Character.Name),
			user.UserId)
	}
	return true, nil
}

// Unhood opens the hood at full strength. The next room the bearer enters
// trims it back to their eyes.
func Unhood(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	recs := hoodedLight(user)
	if len(recs) == 0 {
		user.SendText(messaging.CategorySystem, `You have no hooded light to open.`)
		return true, nil
	}
	if !recs[0].Hooded {
		user.SendText(messaging.CategorySystem, `Your lantern's hood is already open.`)
		return true, nil
	}
	for _, rec := range recs {
		rec.ResetLight()
	}
	user.SendText(messaging.CategorySystem, `You throw back the hood of your lantern, and light floods out around you.`)
	if room != nil {
		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> throws back the hood of a lantern, and light floods the area.`, user.Character.Name),
			user.UserId)
	}
	return true, nil
}
