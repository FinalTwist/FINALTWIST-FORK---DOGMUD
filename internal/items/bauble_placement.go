package items

import "time"

// Where a found bauble lies. A bauble named by a search is usually pocketed
// at once; these fields are set only when it is left lying in a room
// instead (actions/search_bauble.go):
//
//   - found in a household (an indoor room with a resident about): it
//     belongs to the household, and taking it there is theft
//     (actions/household_bauble.go);
//   - too heavy for the finder, or the finder gone by the time it was named.
//
// All three are cleared the moment any character carries the item
// (Character.StoreItem), so a bauble that has been picked up is an ordinary
// possession: it shows no spot, belongs to no household and never vanishes.
// Dropping it again does not set them.

// LeaveBaubleAt marks a bauble as left lying untaken at spot ("on the
// bookshelf", or "" for no particular spot) since at. household is the room
// whose household owns it, or 0.
func (i *Item) LeaveBaubleAt(spot string, household int, at time.Time) {
	i.BaubleSpot = spot
	i.BaubleHousehold = household
	i.BaubleLeftAt = at.Unix()
}

// ClearBaublePlacement forgets where a bauble lay: it has been taken.
func (i *Item) ClearBaublePlacement() {
	i.BaubleSpot = ``
	i.BaubleHousehold = 0
	i.BaubleLeftAt = 0
}

// BaubleBelongsTo reports whether this is a found bauble owned by the
// household of roomId, so that taking it in that room is theft.
func (i *Item) BaubleBelongsTo(roomId int) bool {
	return i.IsBauble() && roomId > 0 && i.BaubleHousehold == roomId
}

// BaubleUntakenFor reports how long a found bauble has lain untaken, and
// false for anything else (an ordinary item, or a bauble someone has
// carried and dropped).
func (i *Item) BaubleUntakenFor(now time.Time) (time.Duration, bool) {
	if !i.IsBauble() || i.BaubleLeftAt <= 0 {
		return 0, false
	}
	return now.Sub(time.Unix(i.BaubleLeftAt, 0)), true
}

// BaubleSpotSuffix is what follows a bauble's name in a room's list of
// things on the ground: " (on the bookshelf)", or "" when it has no spot.
func (i *Item) BaubleSpotSuffix() string {
	if !i.IsBauble() || i.BaubleSpot == `` {
		return ``
	}
	return ` <ansi fg="noun">(` + i.BaubleSpot + `)</ansi>`
}
