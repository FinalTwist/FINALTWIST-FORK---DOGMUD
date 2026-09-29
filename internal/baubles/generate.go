package baubles

import (
	"context"
	"errors"
	"sort"
	"sync/atomic"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// Naming a bauble (Phase 4).
//
// The model is reached through a seam: modules/baubles installs a
// GeneratorFunc at boot when it is enabled AND an OpenAI API key is present.
// With nothing installed, every bauble is a generic trinket (fallback.go).
//
// ⚠️ Generate BLOCKS for as long as the model takes. It must only run on a
// goroutine that does not hold util.LockMud(); the caller takes the lock
// afterwards, once, to mint and deliver (actions/search_bauble.go).

// GenRequest is everything the model is told about a find. It is built under
// the mud lock from AUTHORED text only (room title, description and noun
// keys, never signs or anything a player typed) and holds no game pointers.
type GenRequest struct {
	Tier            ValueTier
	Source          Source
	Place           Place
	RoomTitle       string
	RoomDescription string
	RoomNouns       []string
	Container       string // what was searched; empty means the room itself
	// ContainerDescription is the authored description of what was searched
	// (a room noun's or a discovered hidden noun's text); empty for the room
	// itself or a container, which has none.
	ContainerDescription string
	TimeOfDay            string // day or night
	RecentNames          []string

	// Victim is whom a pickpocketed bauble was lifted from: the NPC's
	// authored name ("a harried clerk"), never a player's and never a
	// companion's (a name a player may have chosen). Empty otherwise.
	Victim string

	// FinderUserId is who found it. It is NEVER sent to the model: it only
	// decides whose own key may name the find (a player who allowed it on
	// the companion key page). 0 is nobody (an admin spawn, say).
	FinderUserId int
}

// GenResult is a generator's answer: the reply plus what the catalog records
// about how it was made.
type GenResult struct {
	Reply         Reply
	Generator     Generator
	Model         string
	PromptVersion int
	Tokens        int
	Moderated     bool
	PlayerKey     bool // named through the finder's own key, not the server's

	// FinderOnly is player-key text the server could not moderate: its
	// finder reads it, everyone else the generic trinket (the record derives
	// it: Record.KeptToFinder; owner ruling 2026-09-29). Only ever with
	// PlayerKey and a finder.
	FinderOnly bool
}

// GeneratorFunc names one bauble. It blocks; it must respect ctx.
type GeneratorFunc func(ctx context.Context, req GenRequest) (GenResult, error)

// GeneratorInfo describes the installed generator for the admin command.
type GeneratorInfo struct {
	Name   string // e.g. "openai"
	Model  string
	Detail string // anything else worth showing, e.g. budget left
}

type installedGenerator struct {
	fn   GeneratorFunc
	info func() GeneratorInfo
}

var generator atomic.Pointer[installedGenerator]

// SetGenerator installs the model-backed namer. nil uninstalls it, after
// which every bauble is a generic trinket. info may be nil.
func SetGenerator(fn GeneratorFunc, info func() GeneratorInfo) {
	if fn == nil {
		generator.Store(nil)
		return
	}
	generator.Store(&installedGenerator{fn: fn, info: info})
}

// CurrentGenerator reports what names baubles now: false means generic
// trinkets.
func CurrentGenerator() (GeneratorInfo, bool) {
	g := generator.Load()
	if g == nil {
		return GeneratorInfo{}, false
	}
	if g.info == nil {
		return GeneratorInfo{Name: `custom`}, true
	}
	return g.info(), true
}

// MaxGenerateTime is the hard ceiling on one Generate call, whatever the
// generator's own timeout says, so a find is never held up for longer.
const MaxGenerateTime = 30 * time.Second

// Generate names one find. It returns the model's answer when a generator
// is installed and its answer is usable (validated and clamped), and a
// generic trinket otherwise. It never fails. randn picks a generic
// trinket's value and weight: pass util.Rand in production (nil gives the
// deterministic midpoint, for tests).
func Generate(ctx context.Context, req GenRequest, randn func(n int) int) GenResult {
	tier := req.Tier
	if !tier.Valid() {
		tier = TierCheap
	}
	generic := func() GenResult {
		return GenResult{Reply: GenericTrinket(tier, randn), Generator: GeneratorLocal}
	}

	g := generator.Load()
	if g == nil {
		return generic()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, MaxGenerateTime)
	defer cancel()

	res, err := g.fn(ctx, req)
	if err != nil {
		mudlog.Warn(`baubles`, `action`, `generate`, `result`, `generic trinket`, `error`, err)
		return generic()
	}
	cleaned, err := CleanReply(res.Reply)
	if err == nil && (res.PlayerKey || res.FinderOnly) {
		// Text a player's own key wrote is held to plain words, and is
		// either moderated (everyone reads it) or kept to its finder
		// (FinderOnly: the server could not moderate it; owner ruling
		// 2026-09-29). Nothing else is ever finder-only (spec S3).
		switch {
		case !res.PlayerKey:
			err = errors.New(`only text a player's own key wrote is kept to its finder`)
		case res.Moderated:
			res.FinderOnly = false // moderated text is everyone's
			err = CheckPlayerKeyText(cleaned)
		case res.FinderOnly && req.FinderUserId > 0:
			err = CheckPlayerKeyText(cleaned)
		default:
			err = errors.New(`player-key text was neither moderated nor kept to its finder`)
		}
	}
	if err != nil {
		mudlog.Warn(`baubles`, `action`, `generate`, `result`, `generic trinket`, `error`, err)
		return generic()
	}
	if TooBigFor(cleaned, req.Source) {
		// The model's own weight, or the thing its name names, says it
		// described something no pocket holds, whatever its weight would be
		// clamped to. A generic (small) trinket instead.
		mudlog.Warn(`baubles`, `action`, `generate`, `result`, `generic trinket`, `error`, `too big for a pocket`, `weight`, cleaned.WeightLbs)
		return generic()
	}
	res.Reply = cleaned
	if res.Generator == `` {
		res.Generator = GeneratorOpenAI
	}
	return res
}

// RecentNames returns up to n names of server-key model-named baubles found in the zone,
// newest first, so the prompt can ask for something different.
func RecentNames(zone string, n int) []string {
	cat.mu.RLock()
	recs := make([]*Record, 0, 32)
	for _, r := range cat.records {
		// Never a name a player's own key wrote: this list goes into
		// every later find's prompt, on anyone's key (spec S3).
		if r.Zone == zone && r.Generator == GeneratorOpenAI && !r.PlayerKey {
			recs = append(recs, r)
		}
	}
	sort.Slice(recs, func(a, b int) bool { return recs[a].Id > recs[b].Id })
	out := []string{}
	for _, r := range recs {
		if len(out) >= n {
			break
		}
		out = append(out, r.Name)
	}
	cat.mu.RUnlock()
	return out
}
