package rooms

import "github.com/GoMudEngine/GoMud/internal/items"

// WalkItems calls fn with a pointer to every item in the room: on the
// floor, in the stash, in each container, on each corpse (the dead
// character's own gear and the loot beside it), and in the sealed crate.
// A container's items are reached through the map value's slice, which
// shares its backing array with the map, so those pointers are live too.
func (r *Room) WalkItems(fn func(*items.Item)) {
	items.WalkSlice(r.Items, fn)
	items.WalkSlice(r.Stash, fn)
	for name := range r.Containers {
		items.WalkSlice(r.Containers[name].Items, fn)
	}
	for i := range r.Corpses {
		r.Corpses[i].Character.WalkItems(fn)
		items.WalkSlice(r.Corpses[i].Loot.Items, fn)
	}
	if r.SealedCrate != nil {
		r.SealedCrate.WalkItems(fn)
	}
}

// LoadedRooms returns every room in memory, ephemeral rooms included. The
// caller holds the mud lock (util.LockMud), which guards the room map.
func LoadedRooms() []*Room {
	out := make([]*Room, 0, len(roomManager.rooms))
	for _, r := range roomManager.rooms {
		out = append(out, r)
	}
	return out
}
