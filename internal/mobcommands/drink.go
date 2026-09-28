package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Drink(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Chunk 4e: can't drink while grappled — both hands committed.
	if mob.Character.Position != nil && mob.Character.Position.IsGrappling() {
		return true, nil
	}

	// Check whether the user has an item in their inventory that matches
	if matchItem, found := mob.Character.FindInBackpack(rest); found {

		itemSpec := matchItem.GetSpec()

		if itemSpec.Subtype != items.Drinkable {
			return true, nil
		}

		mob.Character.CancelConditionsWithFlag(conditions.Hidden)

		mob.Character.UseItem(matchItem)

		room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> drinks <ansi fg="itemname">%s</ansi>.`, mob.Character.Name, matchItem.DisplayName()))

		// A magnitude-scaled condition (lighting plan 5c) applies at the
		// item's Magnitude through the same helper the player's drink uses.
		// A mob's potion has no aging or crafter scaling here, so durationMult
		// is 1. Everything else keeps its authored application.
		for _, conditionId := range itemSpec.ConditionIds {
			if mag, trig, ok := items.PotionMagnitudeApplication(&itemSpec, conditions.GetConditionSpec(conditionId), 1); ok {
				mob.AddConditionMagnitude(conditionId, trig, mag, `drink`)
				continue
			}
			mob.AddCondition(conditionId, `drink`)
		}
	}

	return true, nil
}
