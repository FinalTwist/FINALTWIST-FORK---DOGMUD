package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Rally is the mob-side shout that applies the rally mitigation condition
// to the casting mob. Mob rally applies the condition to itself only; ally
// fan-out is a player-command concern.
func Rally(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	result := actions.ExecuteRally(&actions.MobActor{Mob: mob, Room: room})
	if result.Cost.Status == characters.CostRefused {
		return true, nil
	}
	if !result.Executed {
		return true, nil
	}

	// Heard, not seen: the name follows each listener's sight (sight gates 5b).
	actions.SendHeard(&actions.MobActor{Mob: mob, Room: room}, messaging.CategoryRally,
		fmt.Sprintf(`<ansi fg="cyan-bold"><ansi fg="mobname">%s</ansi> lets out a rallying roar!</ansi>`, mob.Character.Name))

	return true, nil
}
