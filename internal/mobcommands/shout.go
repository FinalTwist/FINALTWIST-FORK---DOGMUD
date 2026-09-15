package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Shout(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	isSneaking := mob.Character.IsHidden()

	if isSneaking {
		msg := fmt.Sprintf(`someone shouts, "<ansi fg="saytext-mob">%s</ansi>"`, rest)
		room.SendText(messaging.CategoryShout, util.SplitStringNL(msg, 80))
	} else {
		anonMsg := fmt.Sprintf(`someone shouts, "<ansi fg="saytext-mob">%s</ansi>"`, rest)
		namedMsg := fmt.Sprintf(`<ansi fg="mobname">%s</ansi> shouts, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, rest)
		sendAudioRoomText(room, mob, messaging.CategoryShout,
			util.SplitStringNL(anonMsg, 80),
			util.SplitStringNL(namedMsg, 80))
	}

	// Walks standard, temporary, and mutator-added exits. Previously this only
	// walked room.Exits, so a mob alarm in a room connected by a temporary or
	// mutator exit failed to reach neighbours a player shout would have.
	room.ForEachAdjacentRoom(func(otherRoom *rooms.Room, sourceExit string) {
		otherRoom.SendText(messaging.CategoryShout, fmt.Sprintf(`Someone is shouting from the <ansi fg="exit">%s</ansi> direction.`, sourceExit))
	})

	// Chunk 3.3: mob shout wakes sleepers in the same room. Same-room only —
	// adjacent-room sound propagation is out of scope.
	for _, otherUserId := range room.GetPlayers() {
		if other := users.GetByUserId(otherUserId); other != nil {
			if other.Character.HasConditionFlag(conditions.Sleeping) {
				other.Character.CancelConditionsWithFlag(conditions.Sleeping)
				mobs.OnSleeperWoken(other.Character)
			}
		}
	}
	for _, mobInstanceId := range room.GetMobs() {
		if m := mobs.GetInstance(mobInstanceId); m != nil {
			if m.InstanceId == mob.InstanceId {
				continue
			}
			if m.Character.HasConditionFlag(conditions.Sleeping) {
				m.Character.CancelConditionsWithFlag(conditions.Sleeping)
				mobs.OnSleeperWoken(&m.Character)
			}
		}
	}

	return true, nil
}
