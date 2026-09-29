package shops

import "github.com/GoMudEngine/GoMud/internal/items"

// WalkItems calls fn with a pointer to each unique item the shop resells
// (AffixedStock). Stock entries are counts of an item id, not items.
func (si *ShopInventory) WalkItems(fn func(*items.Item)) {
	for i := range si.AffixedStock {
		if si.AffixedStock[i].Item.ItemId > 0 {
			fn(&si.AffixedStock[i].Item)
		}
	}
}
