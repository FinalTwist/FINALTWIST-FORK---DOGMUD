package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// mobCanSee reports whether a mob can make out the room it is standing in.
//
// It is the SAME predicate combat already uses: combat/combat.go builds
// combatContext.sourceCanSee from CanSeeSightImpairedOnly, so a mob's decisions
// and its darkness combat penalty cannot disagree about whether it can see.
//
// GRADED LIGHTING PLAN 2. NightVision no longer restores this on its own in a
// pitch dark room: the window model shifts the usable band rather than
// granting sight outright, and CanSeeSightImpairedOnly demands SightFull
// specifically, which a shifted window still cannot produce below its floor
// (internal/messaging/window.go). So a nightvision mob is now sight-gated the
// same as one with no vision at all, in a room this dark; only actual room
// light changes the answer. A light carried by ANY player or mob lifts the
// darkness for everyone, because Room.LightLevel composes a term for anyone
// present with conditions.EmitsLight (internal/rooms/lighting.go). That is
// what keeps the Ironwind cave bosses attacking: neither has night vision.
//
// A nil mob or room returns true. These run on every behaviour tree tick and a
// missing instance must not silently blind the world.
func mobCanSee(mob *mobs.Mob, room *rooms.Room) bool {
	if mob == nil || room == nil {
		return true
	}
	return messaging.CanSeeSightImpairedOnly(&mob.Character, room)
}
