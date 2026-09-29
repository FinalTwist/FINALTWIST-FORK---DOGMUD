package baubles

import (
	"errors"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Place is where a bauble came into the world, as plain data. Callers build
// it from the room (see the admin bauble command, and search from Phase 3)
// so this package never needs to import rooms.
type Place struct {
	RoomId int
	Zone   string
	Region string // ZoneConfig.Region; NewPlace falls back to the zone
	Biome  string
}

// NewPlace builds a Place, using the zone as the region when the zone has
// no region set.
func NewPlace(roomId int, zone string, region string, biome string) Place {
	region = strings.TrimSpace(region)
	if region == `` {
		region = zone
	}
	return Place{RoomId: roomId, Zone: zone, Region: region, Biome: biome}
}

// MintOpts describes one new bauble.
type MintOpts struct {
	Source       Source
	Place        Place
	FinderUserId int
	Tier         ValueTier // chosen by the caller; an unknown tier is cheap
	FoundIn      string    // the feature searched, if any (Record.FoundIn)

	// Result is the finished text from Generate. nil draws from the
	// fallback corpus here (a generic trinket when it has nothing that
	// fits). Either way its numbers are clamped to Tier.
	Result *GenResult

	// Randn picks the corpus entry, or a generic trinket's value and
	// weight: randn(n) returns [0, n). nil means util.Rand. Tests pass a
	// fixed function.
	Randn func(n int) int
}

// ErrNoCarrier means the carrier item file (BaubleItemId) is not loaded.
var ErrNoCarrier = errors.New(`bauble carrier item is not loaded`)

// Mint creates one new bauble: a catalog record with FINAL text (from
// o.Result, or else from the fallback corpus), and the carrier item pointing
// at it. The record is on disk before Mint returns. Naming happens before
// this, off the mud lock (Generate); Mint itself never waits on anything.
func Mint(o MintOpts) (items.Item, Record, error) {
	item := items.New(items.BaubleItemId)
	if item.ItemId != items.BaubleItemId {
		return items.Item{}, Record{}, ErrNoCarrier
	}

	randn := o.Randn
	if randn == nil {
		randn = util.Rand
	}
	tier := o.Tier
	if !tier.Valid() {
		tier = TierCheap
	}
	source := o.Source
	if source == `` {
		source = SourceAdmin
	}

	var res GenResult
	if o.Result != nil {
		res = *o.Result
	} else {
		res = Fallback(o.Place, tier, source, RecentFallbackNames(o.Place.Zone, fallbackRecentNames), randn)
	}
	if res.Generator == `` {
		res.Generator = GeneratorLocal
	}
	status := StatusFallback
	if res.Generator.Named() {
		status = StatusReady
	}

	limited := ApplyLimitsFor(res.Reply, tier, source)
	if res.PlayerKey {
		// A value a player's own key proposed is not trusted, even clamped
		// into the tier: the server rolls it. ValueProposed keeps what the
		// key said, for the record (spec S3).
		limited.Reply.Value = tier.RollValue(randn)
	}
	rec, err := Create(Record{
		Status:         status,
		Name:           limited.Reply.Name,
		NameSimple:     limited.Reply.NameSimple,
		Description:    limited.Reply.Description,
		Material:       limited.Reply.Material,
		Tier:           tier,
		Value:          limited.Reply.Value,
		ValueProposed:  limited.ProposedValue,
		WeightLbs:      limited.Reply.WeightLbs,
		WeightProposed: limited.ProposedWeight,
		Source:         source,
		RoomId:         o.Place.RoomId,
		Zone:           o.Place.Zone,
		Region:         o.Place.Region,
		Biome:          o.Place.Biome,
		FoundByUserId:  o.FinderUserId,
		FoundRound:     util.GetRoundCount(),
		FoundIn:        o.FoundIn,
		Generator:      res.Generator,
		Model:          res.Model,
		PromptVersion:  res.PromptVersion,
		Tokens:         res.Tokens,
		Moderated:      res.Moderated,
		PlayerKey:      res.PlayerKey,
	})
	if err != nil || rec.Id == `` {
		// Not on disk, so not handed out (Create took the record back).
		return items.Item{}, Record{}, err
	}
	item.Bauble = rec.Id
	return item, rec, nil
}
