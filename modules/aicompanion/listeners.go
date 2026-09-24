package aicompanion

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Everything in this file runs under the mud lock (event listeners and the
// ask hook). Nothing here calls the model; it records what the companion
// perceived and queues stimuli for the next dispatch.

// speakerOf names a player the way a companion in the room would: their
// name, or "someone" if they are hidden.
// speakerOf names a player as this companion perceives them: by name when
// it can make them out, and otherwise as a voice in the dark. Its own
// senses decide, not the speaker's hiding alone.
func speakerOf(u *users.UserRecord, mob *mobs.Mob) string {
	if mob != nil && !mob.Character.Perceives(u.Character) {
		return `someone`
	}
	if mob == nil && u.Character.IsHidden() {
		return `someone`
	}
	return u.Character.Name
}

// companionsInRoom returns the standing controllers whose companion is in
// the room.
func (m *AICompanionModule) companionsInRoom(roomId int) []*controller {
	var out []*controller
	for _, c := range m.ctrls {
		if c.instanceId == 0 {
			continue
		}
		if mob := mobs.GetInstance(c.instanceId); mob != nil && mob.Character.RoomId == roomId {
			out = append(out, c)
		}
	}
	return out
}

// onCommunication records speech the companion can hear and queues a
// response when it is addressed.
func (m *AICompanionModule) onCommunication(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.Communication)
	if !ok || !m.cfg.Enabled || evt.CommType != `say` || len(m.ctrls) == 0 {
		return events.Continue
	}

	// A companion's own speech comes back through here; ignore it.
	if evt.SourceMobInstanceId > 0 && m.controllerForInstance(evt.SourceMobInstanceId) != nil {
		return events.Continue
	}

	roomId := 0
	speaker := evt.Name
	speakerUserId := 0

	if evt.SourceUserId > 0 {
		u := users.GetByUserId(evt.SourceUserId)
		if u == nil || u.Character == nil {
			return events.Continue
		}
		roomId = u.Character.RoomId
		speakerUserId = u.UserId
		speaker = u.Character.Name // named per companion below
	} else if evt.SourceMobInstanceId > 0 {
		if sm := mobs.GetInstance(evt.SourceMobInstanceId); sm != nil {
			roomId = sm.Character.RoomId
			speaker = sm.Character.Name
			if sm.Character.IsHidden() {
				speaker = `someone`
			}
		}
	}
	if roomId == 0 || strings.TrimSpace(evt.Message) == `` {
		return events.Continue
	}

	room := rooms.LoadRoom(roomId)
	now := time.Now().Unix()

	for _, c := range m.companionsInRoom(roomId) {
		mob := mobs.GetInstance(c.instanceId)
		others := 0
		heardFrom := speaker
		if mob != nil {
			others = otherPlayersPresent(room, speakerUserId, mob)
			if u := users.GetByUserId(speakerUserId); speakerUserId > 0 && u != nil {
				heardFrom = speakerOf(u, mob)
			}
		}
		speaker := heardFrom
		direct := false
		fromOwner := speakerUserId > 0 && speakerUserId == c.ownerUserId
		if speakerUserId > 0 {
			direct = isAddressed(evt.Message, c.profile.Name, fromOwner, others, m.cfg.RespondWhenAlone)
		}

		// Speech that was not for her is remembered only when the server
		// allows it: it is what lets her overhear, and it is also other
		// people's conversation going to the API.
		if direct || m.cfg.RecordBystanderSpeech {
			line := Line{Speaker: speaker, Kind: `said`, ToMe: direct, Text: evt.Message, Unix: now}
			c.mind.addLine(line, m.cfg.WorkingMemoryLines)
			c.dirty = true
			if direct {
				m.noteConversation(c, roomId, speaker, line)
			}
		}
		if speakerUserId > 0 {
			c.lastSocialUnix = now
		}
		if fromOwner && !direct {
			c.ownerTalkedAway = now
		}
		if direct {
			if fromOwner {
				m.interruptErrand(c, speaker)
			}
			// Spoken to by an owner who is sneaking, she hears it and holds
			// her tongue: answering aloud is what gets people caught.
			if fromOwner && m.sneaking(c, users.GetByUserId(speakerUserId)) {
				continue
			}
			c.push(stimulus{Kind: `heard`, Speaker: speaker, Text: evt.Message, FromOwner: fromOwner})
		}
	}
	return events.Continue
}

// onEmote notices emotes. An emote is something seen, so a companion that
// cannot see, in the dark or blind, or cannot make the actor out, never
// learns of it. One that names the companion is aimed at it and gets a
// response; any other is only remembered as something seen.
func (m *AICompanionModule) onEmote(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.Emote)
	if !ok || !m.cfg.Enabled || evt.UserId == 0 || len(m.ctrls) == 0 {
		return events.Continue
	}
	u := users.GetByUserId(evt.UserId)
	text := strings.TrimSpace(evt.Text)
	if u == nil || u.Character == nil || text == `` {
		return events.Continue
	}
	for _, c := range m.companionsInRoom(evt.RoomId) {
		mob := mobs.GetInstance(c.instanceId)
		room := rooms.LoadRoom(evt.RoomId)
		if mob == nil || room == nil || cannotSee(mob, room) || !mob.Character.Perceives(u.Character) {
			continue
		}
		speaker := speakerOf(u, mob)
		direct := mentionsName(text, c.profile.Name)
		fromOwner := u.UserId == c.ownerUserId
		if !direct && !m.cfg.RecordBystanderSpeech {
			continue // other people's business, by the server's choice
		}
		emoteLine := Line{Speaker: speaker, Kind: `emoted`, ToMe: direct, Text: text, Unix: time.Now().Unix()}
		c.mind.addLine(emoteLine, m.cfg.WorkingMemoryLines)
		c.dirty = true
		if direct {
			m.noteConversation(c, evt.RoomId, speaker, emoteLine)
		}
		c.lastSocialUnix = time.Now().Unix()
		if direct {
			if fromOwner {
				m.interruptErrand(c, speaker)
			}
			c.push(stimulus{Kind: `emote`, Speaker: speaker, Text: text, FromOwner: fromOwner})
		}
	}
	return events.Continue
}

// onGiftAccepted reacts to an item given to the companion. The engine has
// already moved the item; this only records what it meant.
func (m *AICompanionModule) onGiftAccepted(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.GiftAccepted)
	if !ok || !m.cfg.Enabled {
		return events.Continue
	}
	c := m.controllerForInstance(evt.MobInstanceId)
	u := users.GetByUserId(evt.UserId)
	if c == nil || u == nil || u.Character == nil {
		return events.Continue
	}

	itemName := `something`
	if spec := items.GetItemSpec(evt.ItemId); spec != nil && spec.Name != `` {
		itemName = spec.Name
	}
	giver := speakerOf(u, mobs.GetInstance(c.instanceId))
	fromOwner := u.UserId == c.ownerUserId
	now := time.Now().Unix()

	c.mind.addLine(Line{Speaker: giver, Kind: `event`, Text: fmt.Sprintf(`%s gave you %s.`, giver, itemName)}, m.cfg.WorkingMemoryLines)
	c.mind.addMemory(Memory{
		Unix: now, Kind: `gift`, Text: fmt.Sprintf(`%s gave me %s.`, giver, itemName),
		Importance: 5, Emotion: `gratitude`, People: []string{giver}, PlaceId: u.Character.RoomId,
	}, m.cfg.MaxMemories)

	// A gift from the owner is something she keeps (F9.5).
	if fromOwner {
		c.mind.protectItem(evt.ItemId, `a gift from `+giver)
		// And the exact thing, not merely one of its kind.
		if mob := mobs.GetInstance(c.instanceId); mob != nil {
			for i := range mob.Character.Items {
				if mob.Character.Items[i].ItemId == evt.ItemId {
					c.mind.protectInstance(mob.Character.Items[i].UUID.String())
				}
			}
		}
	}

	// A small, certain warmth for a gift from the owner, at most three
	// times a day so gifts cannot be farmed (the model may add a little
	// more within the gift envelope).
	if fromOwner && c.mind.ruleChangesSince(`gift`, now-86400) < 3 {
		c.mind.applyOpinion(Opinion{Affection: 1}, `gift`, `rule`, itemName, false)
	}

	c.dirty = true
	c.snapshotDue = true
	c.lastSocialUnix = now
	c.push(stimulus{Kind: `gift`, Speaker: giver, Text: itemName, FromOwner: fromOwner})
	return events.Continue
}

// onPlayerAttackedMob reacts to someone attacking the companion. The combat
// system handles the fight itself; this records the betrayal or threat.
func (m *AICompanionModule) onPlayerAttackedMob(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerAttackedMob)
	if !ok || !m.cfg.Enabled {
		return events.Continue
	}
	c := m.controllerForInstance(evt.MobInstanceId)
	u := users.GetByUserId(evt.UserId)
	if c == nil {
		// Not an attack on her. It may still be one she watched her owner
		// make on somebody who had done nothing.
		m.witnessAttack(evt.UserId, evt.MobInstanceId)
		return events.Continue
	}
	if u == nil || u.Character == nil {
		return events.Continue
	}

	// One reaction per attacker per ten minutes; a fight fires this often.
	now := time.Now().Unix()
	if last, seen := c.lastAttackBy[u.UserId]; seen && now-last < 600 {
		return events.Continue
	}
	c.lastAttackBy[u.UserId] = now

	attacker := speakerOf(u, mobs.GetInstance(c.instanceId))
	fromOwner := u.UserId == c.ownerUserId

	c.mind.addLine(Line{Speaker: attacker, Kind: `event`, Text: fmt.Sprintf(`%s attacked you.`, attacker)}, m.cfg.WorkingMemoryLines)
	importance := 6
	if fromOwner {
		importance = 9
	}
	c.mind.addMemory(Memory{
		Unix: now, Kind: `attack`, Text: fmt.Sprintf(`%s attacked me.`, attacker),
		Importance: importance, Emotion: `anger`, People: []string{attacker}, PlaceId: u.Character.RoomId,
	}, m.cfg.MaxMemories)

	// Being attacked by the person you travel with costs trust and
	// affection whatever the model says (F5.4); the model may add more
	// within the attack envelope, never less.
	if fromOwner {
		c.mind.applyOpinion(Opinion{Trust: -5, Affection: -5}, `attacked`, `rule`, `attacked me`, false)
		// Being struck by the person you have come to love is one of the
		// few things that changes what you are to each other.
		if romanceRank(c.mind.Romance.Stage) > 0 {
			m.recordCore(c, attacker, c.mind.Romance.Stage, false)
		}
	}

	c.dirty = true
	c.lastSocialUnix = now
	// An attack pre-empts anything that was waiting.
	c.pending = nil
	c.push(stimulus{Kind: `attacked`, Speaker: attacker, FromOwner: fromOwner})
	return events.Continue
}

// handleAsk claims `ask <companion> <text>` for a companion this module
// drives. Installed into internal/companionai.
func (m *AICompanionModule) handleAsk(userId int, mobInstanceId int, text string) bool {
	if !m.cfg.Enabled {
		return false
	}
	c := m.controllerForInstance(mobInstanceId)
	if c == nil {
		return false
	}
	u := users.GetByUserId(userId)
	mob := mobs.GetInstance(mobInstanceId)
	text = strings.TrimSpace(text)
	if u == nil || u.Character == nil || mob == nil || text == `` || u.Muted {
		return false
	}

	safe := util.EscapeAnsiTags(text)
	u.SendText(messaging.CategorySpeech, fmt.Sprintf(
		`You ask <ansi fg="mobname">%s</ansi>, "<ansi fg="saytext">%s</ansi>"`, mob.Character.Name, safe))
	if room := rooms.LoadRoom(u.Character.RoomId); room != nil {
		room.SendTextCommunication(fmt.Sprintf(
			`<ansi fg="username">%s</ansi> asks <ansi fg="mobname">%s</ansi>, "<ansi fg="saytext">%s</ansi>"`,
			u.Character.Name, mob.Character.Name, safe), u.UserId)
	}

	speaker := speakerOf(u, mob)
	if u.UserId == c.ownerUserId {
		m.interruptErrand(c, speaker)
	}
	askLine := Line{Speaker: speaker, Kind: `asked`, ToMe: true, Text: text, Unix: time.Now().Unix()}
	c.mind.addLine(askLine, m.cfg.WorkingMemoryLines)
	m.noteConversation(c, u.Character.RoomId, speaker, askLine)
	c.dirty = true
	c.lastSocialUnix = time.Now().Unix()
	c.push(stimulus{Kind: `asked`, Speaker: speaker, Text: text, FromOwner: u.UserId == c.ownerUserId})
	return true
}

// interruptErrand stops a trip when the owner speaks to the companion
// (F11.6): the owner comes first, and the model decides what next. A walk
// back to the owner is not interrupted.
func (m *AICompanionModule) interruptErrand(c *controller, ownerName string) {
	if c.travel == nil || c.travel.Purpose == `return` {
		return
	}
	dest := c.travel.DestName
	c.travel = nil
	c.mind.addLine(Line{Kind: `event`, Text: `You stopped on your way to ` + dest + ` because ` + ownerName + ` spoke to you.`}, m.cfg.WorkingMemoryLines)
	c.dirty = true
}

// onHealed reacts to someone healing the companion with magic.
func (m *AICompanionModule) onHealed(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.Healed)
	if !ok || !m.cfg.Enabled {
		return events.Continue
	}
	c := m.controllerForInstance(evt.MobInstanceId)
	u := users.GetByUserId(evt.HealerUserId)
	mob := mobs.GetInstance(evt.MobInstanceId)
	if c == nil || u == nil || u.Character == nil {
		return events.Continue
	}
	now := time.Now().Unix()
	healer := speakerOf(u, mobs.GetInstance(c.instanceId))
	fromOwner := u.UserId == c.ownerUserId

	c.mind.addLine(Line{Speaker: healer, Kind: `event`, Text: fmt.Sprintf(`%s healed you.`, healer)}, m.cfg.WorkingMemoryLines)
	c.mind.addMemory(Memory{
		Unix: now, Kind: `event`, Text: fmt.Sprintf(`%s healed my wounds.`, healer),
		Importance: 5, Emotion: `gratitude`, People: []string{healer}, PlaceId: u.Character.RoomId,
	}, m.cfg.MaxMemories)

	// Being tended by the person you travel with earns a little trust and
	// warmth whatever the model says, three times a day at most.
	if fromOwner && c.mind.ruleChangesSince(`healed`, now-86400) < 3 {
		c.mind.applyOpinion(Opinion{Trust: 1, Affection: 1}, `healed`, `rule`, `healed me`, false)
	}
	if fromOwner && mob != nil && healthPct(mob.Character.Health, mob.Character.HealthMax.Value) < 25 {
		m.noteMilestone(c, `revived`)
	}

	c.dirty = true
	c.lastSocialUnix = now
	c.push(stimulus{Kind: `healed`, Speaker: healer, FromOwner: fromOwner})
	return events.Continue
}

// witnessAttack is her owner setting about someone who was not fighting:
// a shopkeeper, a local, a child. The game calls that a crime, her
// faction standing pays for it, and so does what she thinks of him.
func (m *AICompanionModule) witnessAttack(userId int, mobInstanceId int) {
	c, ok := m.ctrls[userId]
	if !ok || c.instanceId == 0 {
		return
	}
	victim := mobs.GetInstance(mobInstanceId)
	mob := mobs.GetInstance(c.instanceId)
	u := users.GetByUserId(userId)
	if victim == nil || mob == nil || u == nil || u.Character == nil {
		return
	}
	if victim.Character.RoomId != mob.Character.RoomId || !mob.Character.Perceives(&victim.Character) {
		return // she did not see it
	}
	if !refusesToFight(c.profile, victim) {
		return // a fight, not a crime
	}
	now := time.Now().Unix()
	if last, seen := c.lastAttackBy[-mobInstanceId]; seen && now-last < 600 {
		return
	}
	c.lastAttackBy[-mobInstanceId] = now

	name := victim.Character.Name
	c.mind.addLine(Line{Kind: `event`, Text: u.Character.Name + ` set about ` + name + `, who had done nothing.`}, m.cfg.WorkingMemoryLines)
	c.mind.addMemory(Memory{Unix: now, Kind: `event`, Text: u.Character.Name + ` attacked ` + name + `, who had done nothing to anyone.`,
		Importance: 8, Emotion: `disgust`, People: []string{u.Character.Name, name}, PlaceId: mob.Character.RoomId}, m.cfg.MaxMemories)
	c.mind.applyOpinion(Opinion{Trust: -4, Respect: -5, Affection: -4}, `witnessed_crime`, `rule`, `set about `+name, false)
	c.dirty = true
	c.push(stimulus{Kind: `witnessed`, Speaker: u.Character.Name, Text: name, FromOwner: true})
}
