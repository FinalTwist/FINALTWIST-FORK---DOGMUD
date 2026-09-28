package actions

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/companionai"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// A player's pickpocket of an NPC takes a moment (docs/baubles, Phase 6a):
//
//  1. The contest is rolled AT ONCE (stealFromMob), and the player is told
//     only "You attempt to pick X's pocket...".
//  2. On a success, the NPC's own bauble is marked to be taken; an NPC
//     carrying none turns one up BaublePickpocketChancePct of the time, and
//     its naming starts now (baubles.Generate, off the lock), so the pause
//     pays for the model call.
//  3. The pause is StealPocketSeconds at Dexterity 100, scaled by
//     100/Dexterity (quicker hands are faster) and held between
//     StealPocketMinSeconds and StealPocketMaxSeconds (PocketDelay).
//  4. When it ends (and a naming not back yet has had up to
//     BaublePickpocketGraceSecs more), the outcome is revealed under the mud
//     lock: the loot, the bauble among it, with the ordinary success line
//     (takeFromMob); or being caught (caughtByMob).
//
// A thief who has left the room, logged off or started fighting by then, or
// a mark who has gone, loses the chance: nothing is taken, nothing is
// caught. A bauble already named for it stays in the mark's pocket, to be
// found by the next attempt. Pickpocketed baubles are pocket-sized: the
// prompt says so (modules/baubles, size_rule) and the catalog clamps their
// weight (baubles.MaxWeightFor).
//
// Every attempt is tracked from the moment it starts, so a copyover or
// shutdown finishes it (FlushPocketAttempts) rather than losing it.

// pocketAttempt is one player's pickpocket of an NPC, between the roll and
// its reveal.
type pocketAttempt struct {
	userId        int
	roomId        int
	mobInstanceId int
	mobName       string
	success       bool

	takeBauble string // the catalog id of a bauble the mark carries, to take
	newBauble  bool   // one is made for this attempt (req is its naming)
	req        baubles.GenRequest
	randn      func(n int) int
	delay      time.Duration
	grace      time.Duration

	// actor is the thief at the attempt, for tests that resolve in line;
	// production resolves the thief again (pocketThief), since they may
	// have moved or gone.
	actor Actor

	ctx    context.Context
	cancel context.CancelFunc
	named  chan struct{} // closed when the naming is back

	mu      sync.Mutex
	res     *baubles.GenResult
	claimed bool
}

// PocketDelay is the pickpocket pause for a thief of this Dexterity.
func PocketDelay(dexterity int) time.Duration {
	cfg := configs.GetBalanceConfig()
	if dexterity < 1 {
		dexterity = 1
	}
	secs := float64(cfg.StealPocketSeconds) * 100 / float64(dexterity)
	secs = math.Max(float64(cfg.StealPocketMinSeconds), math.Min(float64(cfg.StealPocketMaxSeconds), secs))
	return time.Duration(secs * float64(time.Second))
}

// pocketBaubleRoll is whether a successful pickpocket of an NPC carrying no
// bauble turns one up here. A variable so tests can stand in for the dice.
var pocketBaubleRoll = func(room *rooms.Room) bool {
	cfg := configs.GetBalanceConfig()
	if !bool(cfg.BaublesEnabled) || room == nil || !baubleRoomAllowed(room) || baubles.ZoneExcluded(room.Zone) {
		return false
	}
	pct := float64(cfg.BaublePickpocketChancePct)
	if pct <= 0 {
		return false
	}
	return float64(util.Rand(10000)) < pct*100
}

// pocketThief is the thief at the reveal, resolved again: nil when they are
// gone. A variable so tests can use their fake actor.
var pocketThief = func(p *pocketAttempt) (Actor, bool) {
	u := users.GetByUserId(p.userId)
	if u == nil || u.Character == nil {
		return nil, false
	}
	room := rooms.LoadRoom(u.Character.RoomId)
	if room == nil {
		return nil, false
	}
	return &UserActor{User: u, Room: room}, true
}

// runPocketAttempt starts the pause. Production tracks the attempt and runs
// it on its own goroutine; tests replace it to resolve in line.
var runPocketAttempt = func(p *pocketAttempt) StealResult {
	trackPocket(p)
	go p.run()
	return StealResult{Pending: true, DefenderName: p.mobName}
}

// startPocketAttempt is stealFromMob for a player: the roll is made, the
// outcome held back for the pause. Call under the mud lock.
func startPocketAttempt(actor Actor, m *mobs.Mob, success bool) StealResult {
	room := actor.GetRoom()
	cfg := configs.GetBalanceConfig()
	actor.SendText(messaging.CategorySystem, fmt.Sprintf(
		`You attempt to pick <ansi fg="mobname">%s</ansi>'s pocket...`, m.Character.Name))

	ctx, cancel := context.WithCancel(context.Background())
	p := &pocketAttempt{
		userId:        actor.GetUserId(),
		roomId:        room.RoomId,
		mobInstanceId: m.InstanceId,
		mobName:       m.Character.Name,
		success:       success,
		randn:         util.Rand,
		delay:         PocketDelay(actor.GetCharacter().Stats.Dexterity.ValueAdj),
		grace:         time.Duration(float64(cfg.BaublePickpocketGraceSecs) * float64(time.Second)),
		actor:         actor,
		ctx:           ctx,
		cancel:        cancel,
		named:         make(chan struct{}),
	}
	if success {
		if b, ok := carriedBauble(m); ok {
			p.takeBauble = b.Bauble
		} else if pocketBaubleAllowed(m) && pocketBaubleRoll(room) {
			tier := baubles.PickPocketTier(p.randn) // stolen, so richer than a find
			p.req = BaubleRequest(room, tier, baubles.SourcePickpocket, ``)
			p.req.Victim = m.Character.Name // authored: a companion never gets here
			p.req.FinderUserId = p.userId   // picks the finder's own key; never sent
			p.newBauble = true
		}
	}
	return runPocketAttempt(p)
}

// pocketBaubleAllowed: a bauble is never made in the pocket of anyone's
// companion (charmed, or an AI companion), nor of a mob that ever was one
// (a dismissed companion, or one whose charm ran out, keeps the name a
// player may have given it), since that name must not reach the model.
func pocketBaubleAllowed(m *mobs.Mob) bool {
	return !m.Character.IsCharmed() && !m.Character.EverCharmed && !companionai.IsBondedCompanion(m.InstanceId)
}

// markPocketStolen records a bauble taken from a mark's pocket, in room,
// as stolen.
func markPocketStolen(it items.Item, userId int, room *rooms.Room, m *mobs.Mob) {
	theft := baubles.Theft{ByUserId: userId, RoomId: room.RoomId, Zone: room.Zone, FromMob: int(m.MobId), FromName: m.Character.Name}
	if f := factions.FactionsForMob(m); len(f) > 0 {
		theft.Faction = f[0]
	}
	baubles.MarkStolen(it.Bauble, theft, baubleNow())
}

// intoPocket puts a bauble in the mob's pocket. It weighs a pound at most,
// so a mob already carrying all it can keeps it all the same: it is never
// dropped for want of strength.
func intoPocket(m *mobs.Mob, itm items.Item) {
	if m.Character.StoreItem(itm) {
		return
	}
	itm.ClearBaublePlacement()
	m.Character.Items = append(m.Character.Items, itm)
}

// carriedBauble is the first bauble the mob carries.
func carriedBauble(m *mobs.Mob) (items.Item, bool) {
	for _, it := range m.Character.Items {
		if it.IsBauble() {
			return it, true
		}
	}
	return items.Item{}, false
}

// run is the pause and the reveal, on the attempt's own goroutine.
func (p *pocketAttempt) run() {
	defer untrackPocket(p)
	defer func() {
		if r := recover(); r != nil {
			mudlog.Error(`steal`, `action`, `pickpocket`, `panic`, r)
		}
	}()
	if p.newBauble {
		go p.name()
	}
	select {
	case <-time.After(p.delay):
	case <-p.ctx.Done():
	}
	if p.newBauble {
		select {
		case <-p.named:
		case <-time.After(p.grace):
		case <-p.ctx.Done():
		}
	}
	util.LockMud()
	defer util.UnlockMud()
	if !p.claim() {
		return // finished by FlushPocketAttempts
	}
	p.resolve()
}

// name asks for the bauble's naming, off the lock.
func (p *pocketAttempt) name() {
	defer close(p.named)
	res := baubles.Generate(p.ctx, p.req, p.randn)
	p.mu.Lock()
	p.res = &res
	p.mu.Unlock()
}

func (p *pocketAttempt) claim() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.claimed {
		return false
	}
	p.claimed = true
	return true
}

// naming is the bauble's naming if it came back, else the generic
// trinket it would have been.
func (p *pocketAttempt) naming() (baubles.GenResult, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.res != nil {
		return *p.res, true
	}
	return baubles.GenResult{Reply: baubles.GenericTrinket(p.req.Tier, p.randn), Generator: baubles.GeneratorLocal}, false
}

// resolve reveals the outcome. Runs under the mud lock.
func (p *pocketAttempt) resolve() StealResult {
	p.cancel()
	thief, online := pocketThief(p)
	m := mobs.GetInstance(p.mobInstanceId)
	var room *rooms.Room
	if online {
		room = thief.GetRoom()
	}
	if !online || room == nil || room.RoomId != p.roomId || thief.GetCharacter().IsInCombat() ||
		room.AreMobsAttacking(p.userId) ||
		m == nil || m.Character.IsDead() || m.Character.RoomId != p.roomId {
		// The chance is gone: nothing taken, nothing caught, nothing
		// trained. A bauble named for it stays in the mark's pocket, to be
		// found by whoever tries next.
		if online {
			thief.SendText(messaging.CategorySystem, fmt.Sprintf(
				`You lose your chance at <ansi fg="mobname">%s</ansi>'s pocket.`, p.mobName))
		}
		if p.newBauble && m != nil && !m.Character.IsDead() {
			if res, ok := p.naming(); ok {
				if itm, _, err := p.mint(res, m); err == nil {
					intoPocket(m, itm)
				}
			}
		}
		return StealResult{DefenderName: p.mobName, Reason: `chance lost`}
	}

	// Awarded here, with the outcome it reveals (see stealFromMob).
	thief.AwardResolved(p.success, thief.GetCharacter().CandidateFor(string(skills.Skullduggery)))
	if !p.success {
		return caughtByMob(thief, m, room)
	}

	var extra []items.Item
	if p.takeBauble != `` {
		for _, it := range m.Character.Items {
			if it.Bauble == p.takeBauble {
				m.Character.RemoveItem(it)
				events.AddToQueue(events.ItemOwnership{MobInstanceId: m.InstanceId, Item: it, Gained: false})
				extra = append(extra, it)
				break
			}
		}
	} else if p.newBauble {
		res, _ := p.naming()
		itm, rec, err := p.mint(res, m)
		if err != nil {
			mudlog.Error(`steal`, `action`, `pickpocket bauble`, `userId`, p.userId, `error`, err)
		} else {
			extra = append(extra, itm)
			mudlog.Info(`baubles`, `action`, `pickpocket`, `id`, rec.Id, `mob`, p.mobName, `named`, rec.Generator == baubles.GeneratorOpenAI)
		}
	}
	for _, it := range extra {
		markPocketStolen(it, p.userId, room, m)
	}
	return takeFromMob(thief, m, extra)
}

// mint makes the attempt's new bauble, from res.
func (p *pocketAttempt) mint(res baubles.GenResult, m *mobs.Mob) (items.Item, baubles.Record, error) {
	return baubles.Mint(baubles.MintOpts{
		Source:       baubles.SourcePickpocket,
		Place:        p.req.Place,
		FinderUserId: p.userId,
		Tier:         p.req.Tier,
		Result:       &res,
		Randn:        p.randn,
	})
}

var pendingPockets = struct {
	sync.Mutex
	m map[*pocketAttempt]bool
}{m: map[*pocketAttempt]bool{}}

func trackPocket(p *pocketAttempt) {
	pendingPockets.Lock()
	defer pendingPockets.Unlock()
	pendingPockets.m[p] = true
}

// pocketPending is whether this thief has a pickpocket in its pause.
func pocketPending(userId int) bool {
	pendingPockets.Lock()
	defer pendingPockets.Unlock()
	for p := range pendingPockets.m {
		if p.userId == userId {
			return true
		}
	}
	return false
}

func untrackPocket(p *pocketAttempt) {
	pendingPockets.Lock()
	defer pendingPockets.Unlock()
	delete(pendingPockets.m, p)
}

// PendingPocketAttempts is how many pickpockets are in their pause.
func PendingPocketAttempts() int {
	pendingPockets.Lock()
	defer pendingPockets.Unlock()
	return len(pendingPockets.m)
}

// FlushPocketAttempts reveals every pickpocket still in its pause, now: the
// roll was made when it started, so only the pause is cut short (a bauble
// whose naming is not back is the generic trinket it would have been).
// Call it under the mud lock, before rooms and players are saved, at
// copyover and at shutdown: the attempt's goroutine needs the lock to
// finish, and the process is about to end. Returns how many it revealed.
func FlushPocketAttempts() int {
	pendingPockets.Lock()
	list := make([]*pocketAttempt, 0, len(pendingPockets.m))
	for p := range pendingPockets.m {
		list = append(list, p)
	}
	pendingPockets.Unlock()
	sort.Slice(list, func(a, b int) bool { return list[a].userId < list[b].userId })
	n := 0
	for _, p := range list {
		if !p.claim() {
			continue
		}
		p.resolve()
		n++
	}
	if n > 0 {
		mudlog.Info(`steal`, `action`, `flush pickpockets`, `revealed`, n)
	}
	return n
}
