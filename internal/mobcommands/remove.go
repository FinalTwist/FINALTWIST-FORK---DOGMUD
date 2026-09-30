package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Remove(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if mob.Character.HasConditionFlag(conditions.PermaGear) {
		mob.Command(`emote struggles with their gear for a while, then gives up.`)
		return true, nil
	}

	actor := &actions.MobActor{Mob: mob, Room: room}

	// Busy and curse gates live in the shared bodies (slice 5a); a mob is
	// silent on every refusal.
	if rest == "all" {
		actions.RemoveAllEquipment(actor)
		return true, nil
	}

	result := actions.RemoveEquipment(actor, rest)
	if result.Removed {
		room.SendTextVisual(messaging.CategoryEquipment,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> removes their <ansi fg="item">%s</ansi> and stores it away.`, mob.Character.Name, result.Item.DisplayName()))
	}
	return true, nil
}
