package aicompanion

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Harm, answered for. The engine gates harm by the actor: a player's
// attack, special move, ranged shot and harmful spell all pass
// mobs.CheckPlayerHarm for a creature and room.CanPvp for a person, and a
// mob's do not ("Mob casters are never gated", actions.rejectHarmTarget).
// The party check is not uniform: `attack`, the special moves
// (actions/melee_target.go) and `shoot` refuse a party member, but
// actions/cast.go makes no party check for a player target at all.
//
// harmAllowed below checks, for a creature, mobs.CheckPlayerHarm and that
// it stands in her room; for a person, that it is not her owner, that they
// stand in her room, room.CanPvp, and that they are not in her owner's
// party. That is the strictest of the player paths, applied to everything
// she starts, spells included.
//
// A bonded companion is a mob, so on her own
// she would pass nothing, and anyone who could talk her into a fight could
// reach what they themselves may not touch.
//
// So everything she starts is asked of the engine as though her owner had
// done it: she may harm only what her owner may harm, here, now. This
// calls the engine's own rules and copies none of them. It is the module's
// rule for the commands it issues; the engine's combat AI, which fights
// whatever is already fighting her, is untouched, and ordinary companions
// are not gated at all (that would be a separate change to the engine).
//
// A player has no self-defence exception either: `attack` with no target
// picks whoever is hitting them and still refuses a protected creature. So
// there is none here. A protected creature that turns on her is fought by
// the engine's round, not by anything she chooses.

// harmAllowed reports whether her owner could harm this target where she
// stands: a creature (targetMobId) or a person (targetUserId), one of the
// two. The reason is for the trace and her memory, and names nobody.
func harmAllowed(owner *users.UserRecord, room *rooms.Room, targetMobId int, targetUserId int) (bool, string) {
	if owner == nil || owner.Character == nil {
		return false, `there is nobody to answer for it`
	}
	if room == nil {
		return false, `they are not here`
	}
	if targetMobId > 0 {
		target := mobs.GetInstance(targetMobId)
		if target == nil || target.Character.RoomId != room.RoomId {
			return false, `they are not here`
		}
		switch mobs.CheckPlayerHarm(target) {
		case mobs.HarmBlockedCompanion:
			return false, `that is somebody's companion`
		case mobs.HarmBlockedNonCombatant, mobs.HarmBlockedAttackImmune:
			return false, `your companion could not raise a hand to them, so neither will you`
		}
		return true, ``
	}
	if targetUserId > 0 {
		if targetUserId == owner.UserId {
			return false, `you will not turn on your own companion`
		}
		target := users.GetByUserId(targetUserId)
		if target == nil || target.Character == nil || target.Character.RoomId != room.RoomId {
			return false, `they are not here`
		}
		if err := room.CanPvp(owner, target); err != nil {
			return false, `your companion could not fight them here, so neither will you`
		}
		// room.CanPvp does not cover the party; every player path checks it
		// beside it (attack, the special moves).
		if p := parties.Get(owner.UserId); p != nil && p.IsMember(targetUserId) {
			return false, `they travel with your companion`
		}
		return true, ``
	}
	return false, `nobody to aim it at`
}

// areaHarmAllowed is harmAllowed for a harmful spell that lands on the
// whole room. A mob's area harm spares only its caster, its owner, its
// owner's other companions and non-combatants, not whoever her owner may
// not touch, so she looses one only where everyone else it could land on
// passes.
func areaHarmAllowed(owner *users.UserRecord, room *rooms.Room, self *mobs.Mob) (bool, string) {
	if owner == nil || owner.Character == nil || room == nil || self == nil {
		return false, `there is nobody to answer for it`
	}
	for _, id := range room.GetMobs(rooms.FindAll) {
		if id == self.InstanceId {
			continue
		}
		// The engine's area spell passes over the owner's companions and
		// anyone who takes no part in fighting, so those cannot be caught.
		if m := mobs.GetInstance(id); m != nil && (m.Character.IsCharmed(owner.UserId) || m.IsNonCombatant()) {
			continue
		}
		if ok, reason := harmAllowed(owner, room, id, 0); !ok {
			return false, `it would catch someone: ` + reason
		}
	}
	for _, id := range room.GetPlayers(rooms.FindAll) {
		if id == owner.UserId {
			continue // the engine spares her owner
		}
		if ok, reason := harmAllowed(owner, room, 0, id); !ok {
			return false, `it would catch someone: ` + reason
		}
	}
	return true, ``
}
