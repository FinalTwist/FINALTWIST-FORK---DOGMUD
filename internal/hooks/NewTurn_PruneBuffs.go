package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/textutil"
	"github.com/GoMudEngine/GoMud/internal/users"
)

//
// Prune all buffs that have expired.
//

func PruneConditions(e events.Event) events.ListenerReturn {

	/*
		evt, typeOk := e.(events.NewTurn)
		if !typeOk {
			mudlog.Error("Event", "Expected Type", "NewTurn", "Actual Type", e.Type())
			return events.Cancel
		}
	*/

	roomsWithPlayers := rooms.GetRoomsWithPlayers()
	for _, roomId := range roomsWithPlayers {
		// Get rooom
		if room := rooms.LoadRoom(roomId); room != nil {

			// Handle outstanding player buffs
			logOff := false
			for _, uId := range room.GetPlayers(rooms.FindWithConditions) {

				user := users.GetByUserId(uId)

				logOff = false
				if conditionsToPrune := user.Character.Conditions.Prune(); len(conditionsToPrune) > 0 {
					for _, conditionInfo := range conditionsToPrune {
						// Send the end notice (authored, or the generic line;
						// a secret buff is silent).
						endConditionSpec := conditions.GetConditionSpec(conditionInfo.ConditionId)
						if endConditionSpec != nil && endConditionSpec.Narration(conditions.PhaseEnd).Len() > 0 {
							roles := endConditionSpec.Narrate(conditions.PhaseEnd, textutil.TokenContext{
								SourceName:      user.Character.GetCharacterName(true),
								SourcePlainName: user.Character.GetCharacterName(false),
							})
							if roles.Actee != "" {
								user.SendText(messaging.CategoryConditionExpire, roles.Actee)
							}
							if roles.Observer != "" {
								if r := rooms.LoadRoom(user.Character.RoomId); r != nil {
									sendConditionEndRoomText(r, endConditionSpec, roles.Observer, user.UserId)
								}
							}
						}

						if conditionInfo.ConditionId == 0 { // Log them out // logoff // logout
							if !user.Character.HasAdjective(`zombie`) { // if they are currently a zombie, we don't log them out from this buff being removed
								logOff = true
							}
						}
					}

					user.Character.Validate()

					// Push a Char update so the web client's Status &
					// Conditions panel refreshes immediately on buff
					// expiry. GMCP listens to BuffsTriggered and queues
					// Char.Conditions; reusing it here (on
					// the removal batch) avoids a stale panel until some
					// unrelated Char event fires. Player-only — the mob
					// prune branch below has no UserId and is skipped.
					prunedIds := make([]int, 0, len(conditionsToPrune))
					for _, conditionInfo := range conditionsToPrune {
						prunedIds = append(prunedIds, conditionInfo.ConditionId)
					}
					events.AddToQueue(events.ConditionsTriggered{UserId: user.UserId, ConditionIds: prunedIds})

					if logOff {
						mudlog.Info("MEDITATION LOGOFF")
						events.AddToQueue(events.System{Command: "logoff", Data: uId})
					}
				}

			}
		}
	}

	// Handle outstanding mob buffs
	for _, mobInstanceId := range mobs.GetAllMobInstanceIds() {

		mob := mobs.GetInstance(mobInstanceId)

		if conditionsToPrune := mob.Character.Conditions.Prune(); len(conditionsToPrune) > 0 {
			for _, conditionInfo := range conditionsToPrune {
				// Send YAML end text (if defined).
				endConditionSpec := conditions.GetConditionSpec(conditionInfo.ConditionId)
				if endConditionSpec != nil && len(endConditionSpec.Narration(conditions.PhaseEnd).Observer) > 0 {
					// The mob tag, not the player one: see Buff_ApplyBuffs.go.
					// Visual, not audio, for the same reason as start text. The
					// holder line is rendered and dropped: a mob has no client.
					sourceName := mob.Character.GetCharacterName(true)
					if r := rooms.LoadRoom(mob.Character.RoomId); r != nil {
						sourceName = mobDisplayName(mob, r, 0)
					}
					roles := endConditionSpec.Narrate(conditions.PhaseEnd, textutil.TokenContext{
						SourceName:      sourceName,
						SourcePlainName: mob.Character.GetCharacterName(false),
					})
					if roles.Observer != "" {
						if r := rooms.LoadRoom(mob.Character.RoomId); r != nil {
							sendConditionEndRoomText(r, endConditionSpec, roles.Observer)
						}
					}
				}
			}

			mob.Character.Validate()
		}

	}

	return events.Continue

}

// sendConditionEndRoomText sends a buff's end room line on the visual channel. A
// light buff's line is judged as if the room were still lit, because its light
// went out when the buff expired, a round before this prune: see
// Room.SendTextVisualAsLit. Every other end line is judged by the room as it is.
func sendConditionEndRoomText(r *rooms.Room, spec *conditions.ConditionSpec, msg string, skip ...int) {
	for _, flag := range spec.Flags {
		if flag == conditions.EmitsLight {
			r.SendTextVisualAsLit(messaging.CategoryConditionExpire, msg, skip...)
			return
		}
	}
	r.SendTextVisual(messaging.CategoryConditionExpire, msg, skip...)
}
