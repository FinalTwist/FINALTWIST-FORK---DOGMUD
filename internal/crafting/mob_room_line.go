package crafting

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/textutil"
)

// MobRoomLine is what the room sees when a mob reaches outcome p on this
// recipe: the recipe's authored Observer line with the mob as the actor, or
// fallback when the recipe authors none. Mobs have no client, so only the
// room line matters. fallback may be empty, which keeps the outcome silent.
func (r *RecipeSpec) MobRoomLine(p Phase, mobName, fallback string) string {
	roles := r.Narrate(p, textutil.TokenContext{
		ActorName:      fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, mobName),
		ActorPlainName: mobName,
	})
	if roles.Observer != "" {
		return roles.Observer
	}
	return fallback
}
