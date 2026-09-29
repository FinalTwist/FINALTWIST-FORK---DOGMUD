package mobcommands

import (
	"fmt"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// sendMovementMessage sends a visual movement message to players who can see
// and a sound-based fallback to players in darkness without night vision.
//
// visualCat tags the visual (entry/exit) line; the audio soundMsg uses
// CategorySystem since it's an environment-cue ("you hear footsteps").
// sendMovementMessage shows visualMsg to whoever can see and soundMsg to
// whoever cannot.
//
// It now delegates to Room.SendTextVisualWithAudio rather than deciding for
// itself. The hand-rolled version it replaced tested ONLY
// conditions.NightVision, which meant it ignored blindness, sleep and infrared: a
// BLINDED player who happened to carry night vision was shown the named line,
// and a player with infrared got the sound cue when they should have got
// shapes. The shared primitive reads the same perception predicates the rest
// of the pipeline does.
//
// This was the third hand-rolled darkness check in this package. canSeeInDark
// is gone (M4e-1 Task 9: every visual reader now gets ParticipantSight +
// HideNames, the three-tier verdict, instead of a binary lit-or-nightvision
// check). What remains hand-rolled is darkness.go's sendAudioRoomText, still
// two-tier by construction for the four speech commands (say.go, shout.go,
// rally.go, warcry.go) that call it directly.
func sendMovementMessage(room *rooms.Room, visualCat messaging.Category, visualMsg string, soundMsg string) {
	room.SendTextVisualWithAudio(visualCat, visualMsg, soundMsg)
}

func Go(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// If has a condition that prevents combat, skip the player
	if mob.Character.HasConditionFlag(conditions.NoMovement) {
		return true, nil
	}

	// Special behavior allowed for mobs to travel to specific rooms, even if disconnected.
	if forceRoomId, err := strconv.Atoi(rest); err == nil {

		foundRoomExit := false
		for exitName, exitInfo := range room.Exits {
			if exitInfo.RoomId == forceRoomId {
				rest = exitName
				foundRoomExit = true
			}
		}

		if !foundRoomExit {
			c := configs.GetTextFormatsConfig()

			if forceRoomId == room.RoomId {
				return true, nil
			}

			destRoom := rooms.LoadRoom(forceRoomId)
			if destRoom == nil {
				return true, nil
			}

			room.RemoveMob(mob.InstanceId)
			actions.ClearRoomAggroOnDeparture(room, mob.InstanceId)
			destRoom.AddMob(mob.InstanceId)

			// Tell the old room they are leaving
			sendMovementMessage(room, messaging.CategoryRoomExit,
				fmt.Sprintf(string(c.ExitRoomMessageWrapper),
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> runs off suddenly.`, mob.Character.Name),
				),
				`You hear hurried footsteps receding.`)

			// Tell the new room they have arrived
			sendMovementMessage(destRoom, messaging.CategoryRoomEntry,
				fmt.Sprintf(string(c.EnterRoomMessageWrapper),
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> enters from nearby.`, mob.Character.Name),
				),
				`You hear footsteps approaching.`)

			return true, nil

		}
	}

	if rest == `home` {
		mob.Command(`pathto home`)
		return true, nil
	}

	exitResult := actions.FindExit(room, rest)
	exitName := exitResult.ExitName
	goRoomId := exitResult.RoomId

	exitInfo, _ := room.GetExitInfo(exitName)
	if exitInfo.Lock.IsLocked() {

		mob.Command(fmt.Sprintf(`emote tries to go the <ansi fg="exit">%s</ansi> exit, but it's locked.`, exitName))

		return true, nil
	}

	if exitName != `` {

		// Load current room details
		destRoom := rooms.LoadRoom(goRoomId)
		if destRoom == nil {
			return false, fmt.Errorf(`room %d not found`, goRoomId)
		}

		// Entering through the far side of a locked door would unlock it; for
		// now mobs do not do that.
		if back := destRoom.FindExitTo(room.RoomId); back != `` {
			if backInfo, _ := destRoom.GetExitInfo(back); backInfo.Lock.IsLocked() {
				return true, nil
			}
		}

		actions.RelocateMob(mob, room, exitName, destRoom)

		// We want the `waypoint` onPath event triggered right after they enter the room.
		if currentStep := mob.Path.Current(); currentStep != nil && currentStep.Waypoint() {

			// Anytime a mob reaches a waypoint, introduce a 1 second delay before they can perform any additional commands.
			// This gives a more natural feel to mob behavior, and gives those following a moment to catch up before the mob does something.
			mob.Command("noop", 1)
		}

		return true, nil
	}

	return false, nil
}
