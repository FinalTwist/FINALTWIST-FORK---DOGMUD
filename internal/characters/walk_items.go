package characters

import "github.com/GoMudEngine/GoMud/internal/items"

// WalkItems calls fn with a pointer to the item in every filled equipment
// slot, in AllSlots order. The pointers are the real slot fields.
func (w *Worn) WalkItems(fn func(*items.Item)) {
	for _, s := range w.AllSlots() {
		if s.Item.ItemId > 0 {
			fn(s.Item)
		}
	}
}

// WalkItems calls fn with a pointer to every item this character holds:
// backpack, component bag, potion bandolier, every equipment slot, the
// pet's pack, and each companion's saved pack and gear. The pointers are
// live, so a migration may change an item in place.
//
// The bauble catalog sweep reads every character in the world through
// this; an item it does not see can have its record pruned.
// TestItemWalkersVisitEveryItemField (repo root) plants an item in every
// items.Item field reachable from a character and fails naming any this
// does not walk.
func (c *Character) WalkItems(fn func(*items.Item)) {
	items.WalkSlice(c.Items, fn)
	items.WalkSlice(c.ComponentItems, fn)
	items.WalkSlice(c.PotionItems, fn)
	c.Equipment.WalkItems(fn)
	items.WalkSlice(c.Pet.Items, fn)
	for i := range c.Companions {
		items.WalkSlice(c.Companions[i].Items, fn)
		c.Companions[i].Equipment.WalkItems(fn)
	}
}
