package aicompanion

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Noticing and self-directed activity (F4.5, F6.6, F7.1, F7.5, F8.9-lite).
// Everything here is local and cheap: the scene is scored in code and only
// the few best things become a stimulus. The model then decides whether to
// comment, act or ignore them.

// perceive runs every round for a standing companion: it settles the last
// action's outcome, keeps its map up to date, moves any trip along, brings
// it back when it has been apart from its owner too long, notices arrivals
// in a new room and new things in the current one, and occasionally offers
// a quiet moment to deal with something nearby.
func (m *AICompanionModule) perceive(c *controller, u *users.UserRecord, round uint64, now time.Time) {
	mob := mobs.GetInstance(c.instanceId)
	if mob == nil {
		return
	}
	// While its owner is sneaking, a companion keeps out of it: no
	// noticing aloud, no errands, no talking. If it can get out of sight
	// itself, it goes with them; if it cannot, it stays where it is.
	if m.sneaking(c, u) {
		m.trySneakAlong(c, mob, round)
		if c.travel != nil {
			m.endTravel(c, `You stayed where you were rather than give them away.`, false)
		}
		c.pending = nil
		return
	}

	if c.pendingAct != nil && round >= c.pendingAct.Round+2 {
		m.verifyPending(c, mob, u.Character.Name)
	}

	m.noticeGold(c, mob, u)

	// Something going wrong with either of them is worth her saying
	// something about when it happens, not whenever she next speaks.
	newlyAiling, seenConditions := newAilments(mob, u, c.knownConditions, !cannotSeeOwner(mob, u))
	c.knownConditions = seenConditions
	if len(newlyAiling) > 0 && !c.paused && now.Unix()-c.lastAilment > 60 {
		c.lastAilment = now.Unix()
		c.push(stimulus{Kind: `ailing`, Text: strings.Join(newlyAiling, `; `), FromOwner: true})
	}

	inCombat := mob.Character.IsInCombat()
	if inCombat {
		c.mind.markDanger(mob.Character.RoomId, 1, now.Unix(), true)
	}

	sc := buildScene(mob, u, c.profile, c.mind, now.Unix())
	ownerHere := u.Character.RoomId == mob.Character.RoomId
	enteredRoom := mob.Character.RoomId != c.lastRoomId

	// Walked (or was taken) into a room: map it.
	if enteredRoom && c.convo != nil {
		m.closeConversation(c, `she moved on`)
	}
	if enteredRoom {
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil && !cannotSee(mob, room) {
			c.mind.recordRoom(room, sc, c.lastRoomId, now.Unix(), m.cfg.MaxKnownRooms)
			c.mind.confirmHearsay(room.RoomId)
			imp := impressionOf(c.mind.Places, room.RoomId, strings.TrimSpace(room.Title))
			imp.Visits++
			imp.Unix = now.Unix()
		}
		c.lastRoomId = mob.Character.RoomId
		c.seenKeys = sc.keysAbove(0)
		c.followPending = false
		c.bumpWorld()
		c.dirty = true
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			// A room she could not see is not a room she can name, even
			// when retracing her steps.
			where := `somewhere dark`
			if !cannotSee(mob, room) {
				where = strings.TrimSpace(room.Title)
			}
			c.recentPath = append(c.recentPath, where)
			if len(c.recentPath) > 5 {
				c.recentPath = c.recentPath[len(c.recentPath)-5:]
			}
		}
	}

	// A trip in progress owns the companion's movement.
	if c.travel != nil {
		m.advanceTravel(c, mob, u, round)
		return
	}

	// Apart from the owner with nowhere to be: head back after a while.
	if !ownerHere {
		m.headBack(c, mob, u, round)
		return
	}
	c.apartSince = 0

	if inCombat {
		return
	}

	// What the owner is doing shapes what the companion does (F7.1): while
	// the owner is talking to someone else, it does not interrupt.
	ownerBusy := now.Unix()-c.ownerTalkedAway < 30

	if enteredRoom {
		// A warrant, or a room of people who would gladly see the back of
		// him, comes before anything lying on the floor.
		if trouble := roomTrouble(rooms.LoadRoom(mob.Character.RoomId), mob, u); trouble != `` {
			c.push(stimulus{Kind: `trouble`, Text: trouble, FromOwner: true})
			return
		}
		if notable := sc.notable(m.cfg.NoticeThreshold, 3); len(notable) > 0 && !ownerBusy {
			m.notice(c, notable, now)
		}
		return
	}

	// Something new appeared in the same room.
	var fresh []thing
	for _, t := range sc.notable(m.cfg.NoticeThreshold, 5) {
		if c.seenKeys == nil || !c.seenKeys[t.Key] {
			fresh = append(fresh, t)
		}
	}
	c.seenKeys = sc.keysAbove(0)
	if len(fresh) > 0 && !ownerBusy {
		m.notice(c, fresh, now)
		return
	}

	// A quiet moment to deal with something nearby. When the owner has been
	// idle at the keyboard for a while, such moments come twice as often.
	if m.cfg.AutonomyMinutes <= 0 || ownerBusy || c.paused || c.inFlight || len(c.pending) > 0 || c.pendingAct != nil {
		return
	}
	interval := int64(m.cfg.AutonomyMinutes) * 60
	if round > u.GetLastInputRound()+10 {
		interval /= 2
	}
	switch c.mind.Autonomy {
	case autonomyClose:
		interval *= 2
	case autonomyFree:
		interval /= 2
	}
	if now.Unix()-c.lastAutonomy < interval || u.Character.IsInCombat() {
		return
	}
	c.lastAutonomy = now.Unix()
	if best := sc.notable(m.cfg.NoticeThreshold*0.75, 3); len(best) > 0 {
		c.push(stimulus{Kind: `idle`, Text: thingNames(best)})
	} else if agenda := agendaText(c.mind, c.agenda); agenda != `` && c.mind.Autonomy != autonomyClose {
		// Nothing here, but something to be getting on with (F12.9).
		c.push(stimulus{Kind: `idle`, Text: `nothing in particular`})
	}
}

// notice queues a "you notice" stimulus, rate limited.
func (m *AICompanionModule) notice(c *controller, things []thing, now time.Time) {
	if now.Unix()-c.lastNotice < int64(m.cfg.NoticeCooldownSeconds) {
		return
	}
	c.lastNotice = now.Unix()
	c.push(stimulus{Kind: `noticed`, Text: thingNames(things)})
}

func thingNames(ts []thing) string {
	names := make([]string, 0, len(ts))
	for _, t := range ts {
		names = append(names, fmt.Sprintf(`%s (%s)`, t.Name, t.Class))
	}
	return strings.Join(names, `; `)
}

// handleIdle owns a bonded companion's idle tick. Installed into
// internal/companionai and called from the engine's MobIdle handler under
// the mud lock. It keeps the one useful default a charmed mob had, first
// aid for whoever needs it, and adds the companion's own small gestures.
func (m *AICompanionModule) handleIdle(mobInstanceId int) bool {
	if !m.cfg.Enabled {
		return false
	}
	c := m.controllerForInstance(mobInstanceId)
	if c == nil || c.paused {
		// Paused: hand the tick back, so it behaves like any other
		// companion until it is resumed.
		return false
	}
	mob := mobs.GetInstance(mobInstanceId)
	if mob == nil {
		return true
	}

	if mob.Character.KnowsFirstAid() {
		m.noteRevive(c, mob)
		mob.Command(`lookforaid`)
	}

	m.pastime(c, mob)
	return true
}

// pastime is what the companion does with itself when nothing is
// happening: a small gesture, a look through the undergrowth, a search of
// the room. It is chosen locally, at random from what makes sense here, so
// two quiet evenings are not the same. None of it calls the model.
func (m *AICompanionModule) pastime(c *controller, mob *mobs.Mob) {
	if c.inFlight || len(c.pending) > 0 || c.pendingAct != nil || c.travel != nil || c.paused {
		return
	}
	if m.sneaking(c, users.GetByUserId(c.ownerUserId)) {
		return // still and quiet: they are not to be given away
	}
	now := time.Now().Unix()
	if now-c.lastSocialUnix < 60 || now-c.lastPastime < int64(m.cfg.IdlePastimeMinutes)*60 {
		return
	}
	// A chance, not a schedule: she does not fidget on a timer.
	c.lastPastime = now
	if float64(util.Rand(1000))/1000.0 >= m.cfg.IdlePastimeChance {
		return
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return
	}

	type option struct {
		kind   string
		weight int
	}
	opts := []option{}
	if m.cfg.IdleEmoteMinutes > 0 && len(c.profile.idlePool(c.mind.Opinion)) > 0 &&
		now-c.lastIdleEmote >= int64(m.cfg.IdleEmoteMinutes)*60 {
		opts = append(opts, option{`emote`, 4})
	}
	// Searching and foraging are also how a pair of hands gets better at
	// them: DOGMud grows skills by use, and there is no practice command.
	searchKey := fmt.Sprintf(`search@%d`, room.RoomId)
	if last, fails := c.mind.lastInteraction(searchKey); fails < 2 && (last == 0 || now-last > 1800) {
		opts = append(opts, option{`search`, 2})
	}
	// Gathering comes first where there is something to gather: a room that
	// has yielded before is worth working again.
	forageKey := fmt.Sprintf(`forage@%d`, room.RoomId)
	if last, fails := c.mind.lastInteraction(forageKey); fails < 2 && (last == 0 || now-last > 1800) {
		weight := 2
		if yielded, _ := c.mind.lastInteraction(forageKey); yielded > 0 && fails == 0 {
			weight = 6
		}
		opts = append(opts, option{`forage`, weight})
	}
	// A fire and something to cook is worth more than idling by it.
	if len(craftableHere(mob, c.profile, room)) > 0 && rawFoodCount(mob, c.profile) > 0 {
		opts = append(opts, option{`craft`, 6})
	}
	// Reading the ways out is second nature to a scout.
	scanKey := fmt.Sprintf(`scan@%d`, room.RoomId)
	if last, _ := c.mind.lastInteraction(scanKey); last == 0 || now-last > 900 {
		opts = append(opts, option{`scan`, 2})
	}
	// Stripping a body for materials, when the arrangement allows it.
	if len(room.Corpses) > 0 && lootAllowedByArrangement(c.mind.LootRule, nil) == `` {
		opts = append(opts, option{`salvage`, 2})
	}
	// Putting her gear right: rare, and only when there is something in the
	// pack that could be worn.
	if now-c.lastGearUp > 3600 && hasWearableInPack(mob) {
		opts = append(opts, option{`gearup`, 1})
	}
	// Rarely, one of the few things she never forgets comes back to her.
	if len(c.mind.CoreMemories) > 0 && c.mind.pickCoreToTell(now) != nil {
		opts = append(opts, option{`reminisce`, 1})
	}
	if len(opts) == 0 {
		return
	}
	total := 0
	for _, o := range opts {
		total += o.weight
	}
	roll := util.Rand(total)
	choice := opts[len(opts)-1].kind
	for _, o := range opts {
		if roll < o.weight {
			choice = o.kind
			break
		}
		roll -= o.weight
	}
	switch choice {
	case `emote`:
		pool := c.profile.idlePool(c.mind.Opinion)
		line := cleanText(pool[util.Rand(len(pool))], maxEmoteRunes)
		if line == `` {
			return
		}
		c.lastIdleEmote = now
		mob.Command(`emote ` + util.EscapeAnsiTags(line))
		c.mind.addLine(Line{Speaker: c.profile.Name, Kind: `emoted`, Text: line}, m.cfg.WorkingMemoryLines)
		c.dirty = true
	case `search`:
		out := m.issue(c, mob, `search`, `search`, searchKey, `this place`, ``, 0.5, util.GetRoundCount())
		if out.Pending != nil {
			c.pendingAct = out.Pending
		}
	case `forage`:
		out := m.issue(c, mob, `forage`, `forage`, forageKey, `what grows here`, ``, 0.5, util.GetRoundCount())
		if out.Pending != nil {
			c.pendingAct = out.Pending
		}
	case `scan`:
		out := m.issue(c, mob, `scan`, `scan`, scanKey, `the ways out`, ``, 0.5, util.GetRoundCount())
		if out.Pending != nil {
			c.pendingAct = out.Pending
		}
	case `salvage`:
		out := m.issue(c, mob, `salvage`, `salvage`, fmt.Sprintf(`salvage@%d`, room.RoomId), `what was left of the dead`, ``, 0.5, util.GetRoundCount())
		if out.Pending != nil {
			c.pendingAct = out.Pending
		}
	case `craft`:
		if made := craftableHere(mob, c.profile, room); len(made) > 0 {
			out := m.issue(c, mob, `craft`, `craft `+made[0].Id, `craft:`+made[0].Id, made[0].Name, ``, 0.5, util.GetRoundCount())
			if out.Pending != nil {
				c.pendingAct = out.Pending
			}
		}
	case `reminisce`:
		if cm := c.mind.pickCoreToTell(now); cm != nil {
			cm.Told++
			cm.LastTold = now
			c.dirty = true
			c.push(stimulus{Kind: `remembering`, Text: cm.Text, FromOwner: true})
		}
	case `gearup`:
		c.lastGearUp = now
		out := m.issue(c, mob, `gearup`, `gearup`, `gearup`, `your gear`, ``, 0.5, util.GetRoundCount())
		if out.Pending != nil {
			c.pendingAct = out.Pending
		}
	}
}

// hasWearableInPack reports whether anything in the pack could be put on,
// which is the only reason to sort her gear out.
func hasWearableInPack(mob *mobs.Mob) bool {
	for i := range mob.Character.Items {
		if isWearable(&mob.Character.Items[i]) {
			return true
		}
	}
	return false
}

// impressionLines lists what the companion thinks of the people and the
// place in the scene, for the prompt.
func impressionLines(sc *scene, mind *Mind) []string {
	var out []string
	if sc == nil {
		return out
	}
	if imp, ok := mind.Places[sc.RoomId]; ok {
		line := fmt.Sprintf(`This place (%s): you have been here %s`, imp.Name, visitWords(imp.Visits))
		if imp.Feeling != `` && imp.Feeling != `neutral` {
			line += `, and you ` + feelingVerb(imp.Feeling) + ` it`
		}
		if imp.Note != `` {
			line += ` (` + imp.Note + `)`
		}
		out = append(out, line+`.`)
	}
	seen := map[int]bool{}
	for _, t := range sc.Things {
		if t.Kind != `npc` || seen[t.MobId] {
			continue
		}
		seen[t.MobId] = true
		imp, ok := mind.NPCs[t.MobId]
		if !ok || (imp.Feeling == `neutral` && imp.Note == ``) {
			continue
		}
		line := fmt.Sprintf(`%s: you %s them`, t.Name, feelingVerb(imp.Feeling))
		if imp.Note != `` {
			line += ` (` + imp.Note + `)`
		}
		out = append(out, line+`.`)
	}
	return out
}

func visitWords(n int) string {
	switch {
	case n <= 1:
		return `for the first time`
	case n < 4:
		return `a few times before`
	case n < 15:
		return `many times`
	}
	return `more times than you can count`
}

// applyImpression records a view of someone present or of the place.
func (m *AICompanionModule) applyImpression(c *controller, sc *scene, p ImpressionProposal, nowUnix int64) {
	if p.Ref == `` || p.Feeling == `` || sc == nil {
		return
	}
	var imp *Impression
	if p.Ref == `here` {
		if existing, ok := c.mind.Places[sc.RoomId]; ok {
			imp = existing
		} else {
			imp = impressionOf(c.mind.Places, sc.RoomId, ``)
		}
	} else {
		t := sc.get(p.Ref)
		if t == nil || t.Kind != `npc` {
			return
		}
		imp = impressionOf(c.mind.NPCs, t.MobId, t.Name)
		imp.Visits++
	}
	imp.Feeling = p.Feeling
	if p.Note != `` {
		imp.Note = p.Note
	}
	imp.Unix = nowUnix
	c.dirty = true
}

// peopleLines tells the model which other travellers present it already
// knows, from its own memories, so a stranger is treated as a stranger and
// an acquaintance as an acquaintance.
func peopleLines(sc *scene, mind *Mind) []string {
	var out []string
	if sc == nil {
		return out
	}
	for _, t := range sc.Things {
		if t.Kind != `player` {
			continue
		}
		n := 0
		var last int64
		for _, mem := range mind.Memories {
			for _, who := range mem.People {
				if strings.EqualFold(who, t.Name) {
					n++
					if mem.Unix > last {
						last = mem.Unix
					}
				}
			}
		}
		switch {
		case n == 0:
			out = append(out, t.Name+`: a stranger to you.`)
		case n < 3:
			out = append(out, fmt.Sprintf(`%s: you have met them before, briefly (%s ago).`, t.Name, humanizeElapsed(time.Now().Unix()-last)))
		default:
			out = append(out, fmt.Sprintf(`%s: someone you know (you have several memories of them).`, t.Name))
		}
	}
	return out
}

// doingLines is what the companion is in the middle of, for the prompt.
func doingLines(c *controller) []string {
	var out []string
	if c.pendingAct != nil {
		out = append(out, fmt.Sprintf(`You have just tried to %s %s; you will know in a moment whether it worked.`,
			strings.ReplaceAll(c.pendingAct.Verb, `_`, ` `), c.pendingAct.Name))
	}
	if len(c.recentPath) > 1 {
		out = append(out, `You have just walked through: `+strings.Join(c.recentPath, `, then `)+`.`)
	}
	return out
}

// noticeGold spots coin arriving in the companion's purse that it did not
// earn itself: the owner handing gold over (the engine sends no event for
// that, unlike an item). It is treated like a gift, at the same daily cap.
func (m *AICompanionModule) noticeGold(c *controller, mob *mobs.Mob, u *users.UserRecord) {
	gold := mob.Character.Gold
	if c.lastGold == 0 && gold > 0 && c.lastGoldSeen == 0 {
		c.lastGold, c.lastGoldSeen = gold, 1
		return
	}
	c.lastGoldSeen = 1
	defer func() { c.lastGold = gold }()

	gained := gold - c.lastGold
	if gained <= 0 || c.pendingAct != nil {
		return
	}
	if u.Character.RoomId != mob.Character.RoomId {
		return // nobody here to have handed it over
	}
	now := time.Now().Unix()
	giver := u.Character.Name
	c.mind.addLine(Line{Speaker: giver, Kind: `event`, Text: giver + ` put some coin in your hand.`}, m.cfg.WorkingMemoryLines)
	c.mind.addMemory(Memory{Unix: now, Kind: `gift`, Text: giver + ` gave me money.`,
		Importance: 4, Emotion: `gratitude`, People: []string{giver}, PlaceId: mob.Character.RoomId}, m.cfg.MaxMemories)
	if c.mind.ruleChangesSince(`gift`, now-86400) < 3 {
		c.mind.applyOpinion(Opinion{Affection: 1}, `gift`, `rule`, `gave me money`, false)
	}
	c.snapshotDue = true
	c.dirty = true
	c.push(stimulus{Kind: `gift`, Speaker: giver, Text: `some gold`, FromOwner: true})
}

// holdFollow reports that the engine should not carry a companion along
// with its owner. There are two reasons: the owner is moving in secret and
// a second pair of footsteps would give them away, or the companion is
// walking after them under her own feet a moment later (followOnFoot). The
// engine asks this before it moves each companion (companionai.HoldPosition),
// and only the bonded companion this module drives is ever held: any other
// companion of the same owner answers false and follows as it always has.
func (m *AICompanionModule) holdFollow(userId int, mobInstanceId int) bool {
	if !m.cfg.Enabled {
		return false
	}
	c, ok := m.ctrls[userId]
	if !ok || c.instanceId == 0 || c.instanceId != mobInstanceId {
		return false
	}
	u := users.GetByUserId(userId)
	if u == nil || u.Character == nil {
		return false
	}
	if !u.Character.IsHidden() || !m.cfg.HoldWhenSneaking {
		// Not sneaking (or the server does not care): she may still be held
		// back a moment so she walks in after her owner rather than
		// appearing beside them.
		return m.followOnFoot(c, u)
	}
	// A companion that got out of sight itself can come along: two shadows
	// are no worse than one. One that could not keeps still instead.
	if mob := mobs.GetInstance(c.instanceId); mob != nil && mob.Character.IsHidden() {
		return false
	}
	return true
}

// sneaking reports whether the owner is currently moving in secret, in
// which case the companion does nothing at all of its own accord.
func (m *AICompanionModule) sneaking(c *controller, u *users.UserRecord) bool {
	return m.cfg.HoldWhenSneaking && u != nil && u.Character != nil && u.Character.IsHidden()
}

// trySneakAlong is the companion trying to melt into the shadows behind its
// owner: one attempt, its own skullduggery against the same roll a player
// makes (actions.Sneak through the ordinary mob command), and one more only
// after a while if it failed. Getting out of sight is what lets it follow;
// failing leaves it standing where it is, which is the safer of the two for
// whoever is creeping about ahead.
func (m *AICompanionModule) trySneakAlong(c *controller, mob *mobs.Mob, round uint64) {
	if mob == nil || mob.Character.IsHidden() || mob.Character.IsInCombat() {
		return
	}
	if round < c.lastSneakTry+sneakRetryRounds {
		return
	}
	c.lastSneakTry = round
	mob.Command(`sneak`)
	c.mind.addLine(Line{Kind: `event`, Text: `You tried to get out of sight and follow them.`}, m.cfg.WorkingMemoryLines)
	c.dirty = true
}

const sneakRetryRounds = 10

// noteRevive records bringing her owner round, but only once it has
// happened: the engine's first aid runs when the room is calm, and this is
// called after it, with the owner back on their feet.
func (m *AICompanionModule) noteRevive(c *controller, mob *mobs.Mob) {
	u := users.GetByUserId(c.ownerUserId)
	if u == nil || u.Character == nil || u.Character.RoomId != mob.Character.RoomId {
		return
	}
	if c.ownerWasDown && u.Character.Health > 0 {
		c.ownerWasDown = false
		m.noteMilestone(c, `revived`)
		return
	}
	c.ownerWasDown = u.Character.Health <= 0
}

// followOnFoot is the companion walking after her owner rather than being
// carried along with them. When the room they have just left has a way out
// leading where they went, she takes it herself a quarter of a second
// later, which puts an ordinary "leaves" and "enters" in the room and lets
// everyone see her come in. Anything the engine would have to do for her
// (a portal, a ferry, a locked door, being dragged out of a fight) is left
// to the engine: this reports false and the owner's companions are moved
// the old way.
func (m *AICompanionModule) followOnFoot(c *controller, u *users.UserRecord) bool {
	if !m.cfg.FollowOnFoot {
		return false
	}
	// A step is already on its way. Rather than queue a second one, let the
	// engine carry her: the step in flight will find her somewhere other
	// than where it set off from and do nothing.
	if c.followPending && util.GetRoundCount()-c.followSince < 3 {
		return false
	}
	mob := mobs.GetInstance(c.instanceId)
	if mob == nil || mob.Character.IsInCombat() || mob.Character.RoomId == u.Character.RoomId {
		return false
	}
	here := rooms.LoadRoom(mob.Character.RoomId)
	if here == nil {
		return false
	}
	exitName := here.FindExitTo(u.Character.RoomId)
	if exitName == `` || !safeExitName(exitName) {
		return false // they did not walk: whatever took them, it takes her too
	}
	if info, ok := here.GetExitInfo(exitName); ok && (info.Lock.IsLocked() || info.Secret) {
		return false
	}
	lo, hi := m.cfg.FollowDelayMin, m.cfg.FollowDelayMax
	delay := lo + (hi-lo)*float64(util.Rand(101))/100.0
	mob.Command(fmt.Sprintf(`%s %d %d`, cmdCompanionFollow, here.RoomId, u.Character.RoomId), delay)
	c.followPending, c.followSince = true, util.GetRoundCount()
	return true
}
