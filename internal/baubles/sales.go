package baubles

import (
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// Selling (Phase 2). The sale itself happens in internal/actions; this file
// is the catalog's side of it: marking a record sold, and the day's totals
// for balance checks.
//
// Every record is sellable, a sold one included. A record goes back into a
// pack two ways: bought back off a shop's shelf (MarkBought, baubles slice
// D), which makes it unsold again, and a crash that lost the seller's save
// after the sale was recorded, the only way a record still marked sold
// reaches a merchant again. Refusing that one would leave the player holding
// an object nobody will ever buy.

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

// MarkBought records that a bauble was bought back off a shop's shelf
// (baubles slice D): a sold record returns to its unsold status
// (unsoldStatus, Restore's rule). A record in any other status is left as
// it is, so one an admin retired while it sat on the shelf stays retired.
// SoldAt, SoldValue and every theft field are kept: the sale happened. It
// returns false when there is no such record.
func MarkBought(id string, buyerUserId int) bool {
	rec, ok := Update(id, func(r *Record) {
		if r.Status == StatusSold {
			r.Status = r.unsoldStatus()
		}
	})
	if ok {
		mudlog.Info(`baubles`, `action`, `bought`, `id`, id, `status`, string(rec.Status), `buyerUserId`, buyerUserId)
	}
	return ok
}

// SalesSince counts the baubles sold at or after t and the gold paid for
// them. Used by the admin command to watch how much gold baubles put into
// the economy. It reads SoldAt alone, so a sale still counts after a
// buyback; a record sold twice keeps only its latest SoldAt and SoldValue
// and counts once, at that sale.
func SalesSince(t time.Time) (count int, gold int) {
	cat.mu.RLock()
	defer cat.mu.RUnlock()
	for _, r := range cat.records {
		if !r.SoldAt.IsZero() && !r.SoldAt.Before(t) {
			count++
			gold += r.SoldValue
		}
	}
	return count, gold
}
