package actions

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
)

// Household baubles (owner rulings, 2026-09-26 and 27).
//
// A bauble found by searching indoors while one of the household is about
// (a resident: see isResident) is not pocketed. It is left where it was
// found, on the feature searched ("on the bookshelf"), and it belongs to the
// household: items.Item.BaubleHousehold is the room's id.
//
// Taking it is theft, and theft is `steal`: `steal doll` goes through
// Steal's checks and the container theft's observer contest
// (stealHouseholdBauble in steal.go). `get` never takes it and never commits
// a crime: it refuses and names the steal command (usercommands/get.go),
// and `get all` leaves it where it is.

// isResident reports whether m, in room, counts as one of the household: a
// person (the player species, a shopkeeper, or a member of a faction), awake,
// able to see, not a monster that attacks on sight and not anyone's
// companion. Animals and the undead do not keep house.
func isResident(m *mobs.Mob, room *rooms.Room) bool {
	if m == nil || m.Character.IsDead() {
		return false
	}
	if m.AutoAggro || m.Character.IsCharmed() {
		return false
	}
	if m.Character.HasConditionFlag(conditions.Sleeping) {
		return false
	}
	if !messaging.CanSeeShapes(&m.Character, room) {
		return false
	}
	if sp := species.GetSpecies(m.Character.SpeciesId); sp != nil && sp.Selectable {
		return true
	}
	return m.HasShop() || len(factions.FactionsForMob(m)) > 0
}

// householdResidents lists the residents in room, by instance id, when the
// room is indoors; nil outdoors or with nobody home.
func householdResidents(room *rooms.Room) []*mobs.Mob {
	if room == nil {
		return nil
	}
	if b := room.GetBiome(); b == nil || !b.Indoor {
		return nil
	}
	ids := append([]int(nil), room.GetMobs()...)
	sort.Ints(ids)
	out := []*mobs.Mob{}
	for _, id := range ids {
		if m := mobs.GetInstance(id); isResident(m, room) {
			out = append(out, m)
		}
	}
	return out
}

// findHouseholdResidents is householdResidents. A variable so tests can
// stand in residents without a mob registry.
var findHouseholdResidents = householdResidents

// HouseholdResident returns a resident of room when a find there would
// belong to the household (the room is indoors and one of the household is
// about), for the find's message and the admin `bauble window` view.
func HouseholdResident(room *rooms.Room) (*mobs.Mob, bool) {
	residents := findHouseholdResidents(room)
	if len(residents) == 0 {
		return nil, false
	}
	return residents[0], true
}

// householdCaught is thiefCaught: what one of the household catching a
// thief brings down (the crime, the attack). A variable so tests can see who
// caught whom without a crime registry.
var householdCaught = thiefCaught

// householdMember is isResident. A variable so tests can make a test mob one
// of the household without a species or faction registry.
var householdMember = isResident
