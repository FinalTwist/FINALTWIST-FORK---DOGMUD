package items

// WalkSlice calls fn with a pointer to each real item in s (ItemId above
// zero: an empty slot or ItemDisabledSlot is skipped), in order. The
// pointers are into s itself, so fn may change an item in place. Every
// store's item walker (Character.WalkItems, Room.WalkItems and the rest)
// is built on it; the bauble catalog sweep reads every live item through
// them.
func WalkSlice(s []Item, fn func(*Item)) {
	for i := range s {
		if s[i].ItemId > 0 {
			fn(&s[i])
		}
	}
}
