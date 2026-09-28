package baubles

import (
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Finding a bauble with `search` (Phase 3). The caller (actions.Search)
// decides who may roll at all (players only) and which rooms never offer
// baubles (instances and other room kinds it can see); this file owns the
// rest: the configured switch, excluded zones, the roll window, the chance
// and the tier. Naming and minting come after, off the lock (Phase 4).

// settings is the bauble slice of the balance config, read once per call.
type settings struct {
	disabled      bool
	chancePct     float64            // a biome not in biomeChance
	biomeChance   map[string]float64 // percent per roll, by lower-case biome id
	skillMaxBonus float64            // chance multiplier gained at full skill factor
	rolls         int                // per room window
	featureWindow time.Duration      // how long a searched feature stays searched
	window        time.Duration
	perPlayer     bool
	excludedZones []string
	weights       [3]int // cheap, average, rare
}

func currentSettings() settings {
	b := configs.GetBalanceConfig()
	return settings{
		disabled:      !bool(b.BaublesEnabled),
		chancePct:     float64(b.BaubleSearchChancePct),
		biomeChance:   b.BaubleBiomeChancePct,
		skillMaxBonus: float64(b.BaubleSkillMaxBonus),
		rolls:         int(b.BaubleRollsPerWindow),
		featureWindow: time.Duration(b.BaubleFeatureWindowMinutes) * time.Minute,
		window:        time.Duration(b.BaubleWindowMinutes) * time.Minute,
		perPlayer:     bool(b.BaubleWindowPerPlayer),
		excludedZones: []string(b.BaubleExcludedZones),
		weights:       [3]int{int(b.BaubleTierWeightCheap), int(b.BaubleTierWeightAverage), int(b.BaubleTierWeightRare)},
	}
}

// BaseChance is the percent chance per roll in a room of this biome, before
// the searcher's skill: BaubleBiomeChancePct for a listed biome (buildings
// high, streets middling, wilderness low), BaubleSearchChancePct otherwise.
func BaseChance(biome string) float64 {
	return currentSettings().baseChance(biome)
}

func (s settings) baseChance(biome string) float64 {
	if pct, ok := s.biomeChance[strings.ToLower(strings.TrimSpace(biome))]; ok {
		return pct
	}
	return s.chancePct
}

// ChanceFor is the percent chance per roll for a searcher with this skill
// factor in a room of this biome. skillFactor runs from 0 (no search skill)
// to 1 (search at SkillSoftCap); the caller computes it (actions uses the
// square root of rank over the soft cap, the same curve as
// combat.SkillMultiplier). The chance grows by up to BaubleSkillMaxBonus
// times itself, so skill matters everywhere in proportion: a skilled
// searcher doubles 5% indoors to 10% and 0.25% in the wild to 0.5%, and
// cannot make the wilderness as rich as a town. Capped at 100.
func ChanceFor(biome string, skillFactor float64) float64 {
	return currentSettings().chanceFor(biome, skillFactor)
}

func (s settings) chanceFor(biome string, skillFactor float64) float64 {
	if skillFactor < 0 {
		skillFactor = 0
	}
	if skillFactor > 1 {
		skillFactor = 1
	}
	pct := s.baseChance(biome) * (1 + s.skillMaxBonus*skillFactor)
	if pct > 100 {
		pct = 100
	}
	return pct
}

// ZoneExcluded reports whether the zone is listed in BaubleExcludedZones.
// Matching ignores case.
func ZoneExcluded(zone string) bool {
	for _, z := range currentSettings().excludedZones {
		if strings.EqualFold(strings.TrimSpace(z), zone) {
			return true
		}
	}
	return false
}

// PickTier chooses a find's tier from the configured weights. randn(n)
// returns [0, n); nil means util.Rand.
func PickTier(randn func(n int) int) ValueTier {
	return pickTier(currentSettings().weights, randn)
}

func pickTier(weights [3]int, randn func(n int) int) ValueTier {
	if randn == nil {
		randn = util.Rand
	}
	tiers := [3]ValueTier{TierCheap, TierAverage, TierRare}
	total := 0
	for _, w := range weights {
		if w > 0 {
			total += w
		}
	}
	if total <= 0 {
		return TierCheap
	}
	roll := randn(total)
	for i, w := range weights {
		if w <= 0 {
			continue
		}
		if roll < w {
			return tiers[i]
		}
		roll -= w
	}
	return TierCheap
}

// rollChance reports whether a roll at chancePct percent succeeds, to one
// part in a million (rollResolution).
func rollChance(chancePct float64, randn func(n int) int) bool {
	if chancePct <= 0 {
		return false
	}
	if chancePct >= 100 {
		return true
	}
	return randn(rollResolution) < int(chancePct*rollResolution/100)
}

// rollResolution is how finely a roll is cut: one in a million, so a
// wilderness chance of 0.25% scaled by skill (0.325%, say) is rolled as
// written rather than rounded to the nearest hundredth of a percent.
const rollResolution = 1_000_000

// FindOpts is one player's search, as RollFind needs it.
type FindOpts struct {
	Place  Place // its Biome sets the base chance
	UserId int

	// SkillFactor is the searcher's search skill from 0 to 1 (see ChanceFor).
	SkillFactor float64

	// Feature is the noun or container searched on its own (`search
	// bookshelf`); empty for a search of the whole room. A feature search
	// spends NO roll from the room's window: the caller has already claimed
	// the feature with ClaimFeatureSearch, which allows one search of it per
	// BaubleFeatureWindowMinutes.
	Feature string

	// Randn is the dice: randn(n) returns [0, n). nil means util.Rand.
	Randn func(n int) int
	// Now is the clock for the roll window. Zero means time.Now().
	Now time.Time
}

// RollFind is the bauble roll for one player search. It spends a roll from
// the room's window (when one is left), rolls the chance, and on a find
// picks the tier. It does NOT mint: naming the find takes a model call,
// which must happen off the mud lock (see actions/search_bauble.go), and
// the record is created only once its text is final.
//
// found is false whenever nothing was found, for any reason; the caller must
// not tell the player why, so a closed window and a failed roll look the
// same.
func RollFind(o FindOpts) (tier ValueTier, found bool) {
	s := currentSettings()
	if s.disabled || ZoneExcluded(o.Place.Zone) {
		return ``, false
	}

	randn := o.Randn
	if randn == nil {
		randn = util.Rand
	}
	now := o.Now
	if now.IsZero() {
		now = time.Now()
	}

	// Where nothing can be found (deep water, the ether), no window opens
	// and no roll is spent.
	chance := s.chanceFor(o.Place.Biome, o.SkillFactor)
	if chance <= 0 {
		return ``, false
	}
	if o.Feature == `` && !takeRoll(o.Place.RoomId, o.UserId, s.perPlayer, s.rolls, s.window, now) {
		return ``, false
	}
	if !rollChance(chance, randn) {
		return ``, false
	}
	tier = pickTier(s.weights, randn)
	mudlog.Info(`baubles`, `action`, `found`, `tier`, string(tier), `chancePct`, chance, `biome`, o.Place.Biome, `feature`, o.Feature, `roomId`, o.Place.RoomId, `zone`, o.Place.Zone, `userId`, o.UserId)
	return tier, true
}

// RevealDelay is the least time between a find and the bauble reaching the
// player, whatever names it: the search keeps going for a moment, so a find
// named by the model and a generic trinket arrive at the same pace.
func RevealDelay() time.Duration {
	return time.Duration(configs.GetBalanceConfig().BaubleRevealSeconds) * time.Second
}
