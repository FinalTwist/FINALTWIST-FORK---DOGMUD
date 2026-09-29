package auctions

import "github.com/GoMudEngine/GoMud/internal/items"

// WalkItems calls fn with a pointer to every item the auction house holds:
// the lot on the block and each seized lot waiting for one. Past auctions
// keep only names.
func (am *AuctionManager) WalkItems(fn func(*items.Item)) {
	if am.ActiveAuction != nil && am.ActiveAuction.ItemData.ItemId > 0 {
		fn(&am.ActiveAuction.ItemData)
	}
	for i := range am.SeizedQueue {
		if am.SeizedQueue[i].Item.ItemId > 0 {
			fn(&am.SeizedQueue[i].Item)
		}
	}
}
