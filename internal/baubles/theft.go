package baubles

import (
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// The catalog's side of a bauble's life after it is found: taken from a
// household (MarkStolen), noted as left in one (MarkHousehold), or gone
// because nobody took it (MarkVanished). The game side is in
// internal/actions (household_bauble.go) and internal/rooms (the untaken
// sweep).

// Theft is who took a household's bauble, and from whom.
type Theft struct {
	ByUserId int
	RoomId   int    // the household's room
	FromMob  int    // mob id of the resident who was watching; 0 if nobody was
	FromName string // that resident's name
	Faction  string // the resident's first faction, if any
}

// MarkStolen records that a bauble was stolen. It returns false when there
// is no such record.
func MarkStolen(id string, t Theft, at time.Time) bool {
	_, ok := Update(id, func(r *Record) {
		r.Stolen = true
		r.StolenByUserId = t.ByUserId
		r.StolenFromRoom = t.RoomId
		r.StolenFromMob = t.FromMob
		r.StolenFromName = t.FromName
		r.StolenFaction = t.Faction
		r.StolenAt = at.UTC()
	})
	if ok {
		mudlog.Info(`baubles`, `action`, `stolen`, `id`, id, `byUserId`, t.ByUserId, `roomId`, t.RoomId, `fromMob`, t.FromMob)
	}
	return ok
}

// MarkHousehold records that a find was left in the room because it belongs
// to the household there.
func MarkHousehold(id string) bool {
	_, ok := Update(id, func(r *Record) { r.Household = true })
	return ok
}

// MarkVanished records that a bauble lay untaken for BaubleUntakenHours and
// was removed from the world. It returns false when there is no such record.
func MarkVanished(id string, at time.Time) bool {
	_, ok := Update(id, func(r *Record) { r.VanishedAt = at.UTC() })
	if ok {
		mudlog.Info(`baubles`, `action`, `vanished`, `id`, id)
	}
	return ok
}

// UntakenLimit is how long a found bauble may lie untaken before it
// vanishes (Balance.BaubleUntakenHours).
func UntakenLimit() time.Duration {
	return time.Duration(configs.GetBalanceConfig().BaubleUntakenHours) * time.Hour
}
