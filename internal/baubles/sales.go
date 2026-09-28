package baubles

import (
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// Selling (Phase 2). The sale itself happens in internal/actions; this file
// is the catalog's side of it: marking a record sold, and the day's totals
// for balance checks.
//
// Every record is sellable, a sold one included: a sold record only reaches a
// merchant again if a crash lost the seller's save after the sale was
// recorded, and refusing it would leave the player holding an object nobody
// will ever buy.

// MarkSold records a sale: status sold, when, and for how much. It returns
// false when there is no such record.
func MarkSold(id string, gold int, sellerUserId int) bool {
	rec, ok := Update(id, func(r *Record) {
		r.Status = StatusSold
		r.SoldAt = time.Now().UTC()
		r.SoldValue = gold
	})
	if ok {
		mudlog.Info(`baubles`, `action`, `sold`, `id`, id, `tier`, string(rec.Tier), `value`, rec.Value, `gold`, gold, `region`, rec.Region, `sellerUserId`, sellerUserId)
	}
	return ok
}

// SalesSince counts the baubles sold at or after t and the gold paid for
// them. Used by the admin command to watch how much gold baubles put into
// the economy.
func SalesSince(t time.Time) (count int, gold int) {
	cat.mu.RLock()
	defer cat.mu.RUnlock()
	for _, r := range cat.records {
		if r.Status == StatusSold && !r.SoldAt.Before(t) {
			count++
			gold += r.SoldValue
		}
	}
	return count, gold
}
