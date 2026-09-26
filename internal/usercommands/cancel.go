package usercommands

import (
	"fmt"
	"slices"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Cancel aborts any in-progress activity (casting, crafting, salvaging).
// Casting: refunds 50% of unspent conviction.
// Crafting: no refund (materials not consumed until completion).
// Salvaging: no refund (item not consumed until completion).
//
// With an argument, `cancel <spell>` instead ends a cancellable condition the
// user holds (see cancelCondition).
func Cancel(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if rest = strings.TrimSpace(rest); rest != `` {
		return cancelCondition(rest, user)
	}

	a := user.Character.Activity
	if a == nil || a.IsFree() {
		user.SendText(messaging.CategorySystem, `You aren't doing anything to cancel.`)
		return true, nil
	}

	switch a.State() {
	case activity.Casting:
		d, _ := a.CastingData()
		// Refund 50% of unspent conviction (existing behavior, preserved).
		unspent := d.TotalConvictionCost - d.ConvictionSpent
		if unspent > 0 {
			refund := unspent / 2
			user.Character.ApplyRestore(characters.PoolConviction, refund)
		}
		_ = a.TransitionToFree(state.TransitionReason{
			Trigger: activity.TriggerCastCancel,
			Actor:   state.ActorRef{UserId: user.UserId},
		})
		user.SendText(messaging.CategorySystem, `You stop casting.`)

	case activity.Crafting:
		_ = a.TransitionToFree(state.TransitionReason{
			Trigger: activity.TriggerCraftCancel,
			Actor:   state.ActorRef{UserId: user.UserId},
		})
		user.SendText(messaging.CategorySystem, `You stop crafting.`)

	case activity.Salvaging:
		_ = a.TransitionToFree(state.TransitionReason{
			Trigger: activity.TriggerSalvageCancel,
			Actor:   state.ActorRef{UserId: user.UserId},
		})
		user.SendText(messaging.CategorySystem, `You stop salvaging.`)
	}
	return true, nil
}

// cancelCondition ends a cancellable condition the user holds, named by the
// spell that grants it (id, alias or name, via spells.ResolveSpell) or by the
// prefix of the condition's own name. Only conditions carrying the
// conditions.Cancellable flag can be let go of. RemoveCondition only marks the
// record expired; the next NewTurn prune removes it and sends its end
// narration, so a successful cancel prints nothing of its own.
func cancelCondition(name string, user *users.UserRecord) (bool, error) {
	var ids []int
	if sd := spells.ResolveSpell(name); sd != nil {
		ids = append(ids, sd.ConditionIds...)
	}
	lower := strings.ToLower(name)
	for _, rec := range user.Character.Conditions.GetConditions() {
		if spec := conditions.GetConditionSpec(rec.ConditionId); spec != nil && strings.HasPrefix(strings.ToLower(spec.Name), lower) {
			ids = append(ids, rec.ConditionId)
		}
	}
	for _, id := range ids {
		spec := conditions.GetConditionSpec(id)
		// GetConditions, not HasCondition: HasCondition reads the id index,
		// which keeps a let-go record until the next prune, so a second
		// cancel in the same round would "succeed" silently.
		if spec == nil || !slices.Contains(spec.Flags, conditions.Cancellable) || len(user.Character.Conditions.GetConditions(id)) == 0 {
			continue
		}
		user.Character.RemoveCondition(id)
		return true, nil
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`You have no %s you can let go of.`, name))
	return true, nil
}
