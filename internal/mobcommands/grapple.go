package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/movenarration"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Grapple(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use grapple
	if !mob.Character.IsInCombat() {
		return true, nil
	}

	res := actions.ExecuteGrapple(&actions.MobActor{Mob: mob, Room: room})
	if res.Cost.Status == characters.CostRefused {
		return true, nil
	}

	if res.OnCooldown || res.NoTarget || res.GrappleImmune || !res.Executed {
		return true, nil
	}

	target := res.Target
	result := res.MoveResult
	mobName := mob.Character.Name

	// Resolve the target user record for direct messaging (player targets).
	var targetChar *users.UserRecord
	if target.UserId > 0 {
		targetChar = users.GetByUserId(target.UserId)
	}

	ids := moveIdentities{
		Actor:      fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, mobName),
		ActorPlain: mobName,
		Actee:      fmt.Sprintf(`<ansi fg="username">%s</ansi>`, target.Name),
		ActeePlain: target.Name,
	}

	// Declared as the interface and left unset when the target is not a player.
	// Assigning a typed-nil *users.UserRecord would make it a non-nil interface
	// value. There is no Actor: a mob has no client.
	var acteeRecipient messaging.Recipient
	if targetChar != nil {
		acteeRecipient = targetChar
	}
	aud := messaging.Audience{
		ActorName: mobName,
		Actee:     acteeRecipient,
		ActeeId:   target.UserId,
		ActeeName: target.Name,
		Room:      room,
	}

	positionTokens := map[string]string{movenarration.TokenPosition: result.PositionDesc}

	// Send messages based on result
	if result.Success {
		sendMoveEvent("grapple", "success", ids, aud, messaging.CategoryGrappleFlow, positionTokens)

		// Disarm messaging: a WORLD EVENT, so it carries the full trio. Not
		// migrated in this task: DisarmResult's literals live in
		// internal/combat/criteffects.go and get their own store plus pinning
		// test in a later commit.
		if result.DisarmResult != nil {
			disarmActee := messaging.NoLine
			if targetChar != nil {
				disarmActee = messaging.Say(messaging.CategoryGrappleFlow, result.DisarmResult.TargetMsg)
			}
			messaging.SendTrio(messaging.Trio{
				Actor:    messaging.NoLine,
				Actee:    disarmActee,
				Observer: messaging.Say(messaging.CategoryGrappleFlow, result.DisarmResult.RoomMessage),
			}, aud)
		}
	} else {
		sendMoveEvent("grapple", "fail", ids, aud, messaging.CategoryGrappleFlow, nil)

		// Critical failure messaging: a world event, full trio. Not migrated
		// in this task: CritFailure's literals live in
		// internal/combat/grapple.go and get their own store plus pinning
		// test in a later commit.
		if result.CritFailure != nil {
			critActee := messaging.NoLine
			if targetChar != nil {
				critActee = messaging.Say(messaging.CategoryGrappleFlow, result.CritFailure.TargetMessage)
			}
			messaging.SendTrio(messaging.Trio{
				Actor:    messaging.NoLine,
				Actee:    critActee,
				Observer: messaging.Say(messaging.CategoryGrappleFlow, result.CritFailure.RoomMessage),
			}, aud)
		}
	}

	return true, nil
}
