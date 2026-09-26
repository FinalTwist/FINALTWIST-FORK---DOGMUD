package aicompanion

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Meeting a companion in the world. With AutoBond on (the default), a new
// character meets its companion as soon as it steps into the world after
// character creation: once it has stood in its first real room for a few
// rounds (not the void, not the tutorial antechamber, not mid-fight), the
// companion arrives, and its first words are its own introduction. An
// existing character with no companion meets one the same way on its next
// login (AutoBondExisting). The companion may part ways if the player
// clearly asks it to, and then it does not come back on its own.

// bondRecord is what the module remembers about one character's bond.
type bondRecord struct {
	Profile  string `yaml:"profile,omitempty"`
	Met      bool   `yaml:"met,omitempty"`
	Declined bool   `yaml:"declined,omitempty"`
	Unix     int64  `yaml:"t,omitempty"`
	// Consented is the player having agreed, in as many words, that talking
	// to this companion sends what they say to OpenAI. Until they do,
	// nothing they say leaves the server: she falls back to her authored
	// lines, which is also what she does for a player who says no.
	Consented bool  `yaml:"consented,omitempty"`
	Refused   bool  `yaml:"refused,omitempty"`
	AskedAt   int64 `yaml:"asked_at,omitempty"`
	// StrangersOff and StrangersOn are the owner's own word on whether
	// passers-by may prompt a model call for their companion
	// (companion-ai strangers off, or on); at most one is set. With
	// strangers off she still hears them and answers with set lines.
	// Neither set, as in every record saved before the choice existed, is
	// the default for whoever pays (strangersOffOn): off on the owner's
	// own key, which is their money, and on for the server's key.
	StrangersOff bool `yaml:"strangers_off,omitempty"`
	StrangersOn  bool `yaml:"strangers_on,omitempty"`
}

type bondState struct {
	Users map[int]*bondRecord `yaml:"users"`
}

const bondStateId = `bond-state`

func (m *AICompanionModule) loadBonds() {
	m.bonds = bondState{Users: map[int]*bondRecord{}}
	if err := m.plug.ReadIntoStruct(bondStateId, &m.bonds); err != nil && !errors.Is(err, util.ErrStateAbsent) {
		mudlog.Error(`aicompanion`, `action`, `loadBonds`, `error`, err)
	}
	if m.bonds.Users == nil {
		m.bonds.Users = map[int]*bondRecord{}
	}
	m.syncConsent()
}

// saveBonds writes the bond records, and brings the consent door's copy up
// to date first: every change of mind about consent is saved, so this is
// the one place that sees them all.
func (m *AICompanionModule) saveBonds() {
	m.syncConsent()
	if m.plug == nil {
		return // built without storage, as the tests build her
	}
	if err := m.plug.WriteStruct(bondStateId, &m.bonds); err != nil {
		mudlog.Error(`aicompanion`, `action`, `saveBonds`, `error`, err)
	}
}

// markBond records that a character has met (or parted from) a companion.
func (m *AICompanionModule) markBond(userId int, profileId string, declined bool) {
	if m.bonds.Users == nil {
		m.bonds.Users = map[int]*bondRecord{}
	}
	rec, ok := m.bonds.Users[userId]
	if !ok {
		rec = &bondRecord{}
		m.bonds.Users[userId] = rec
	}
	rec.Profile, rec.Met, rec.Unix = profileId, true, time.Now().Unix()
	if declined {
		rec.Declined = true
	}
	m.saveBonds()
}

// onCharacterCreated queues a meeting for a brand-new character.
func (m *AICompanionModule) onCharacterCreated(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.CharacterCreated)
	if !ok || !m.cfg.Enabled || !m.cfg.AutoBond {
		return events.Continue
	}
	if m.pendingMeet == nil {
		m.pendingMeet = map[int]*meetWait{}
	}
	m.pendingMeet[evt.UserId] = &meetWait{}
	return events.Continue
}

// meetWait tracks where a character is standing while it waits to meet.
type meetWait struct {
	RoomId int
	Since  uint64
}

// meetingRoomOK reports whether a room is somewhere a companion can walk up
// to a new arrival: a real room, outside the skipped zones.
func (m *AICompanionModule) meetingRoomOK(room *rooms.Room) bool {
	if room == nil || room.RoomId <= 0 {
		return false
	}
	for _, z := range m.cfg.MeetSkipZones {
		if strings.EqualFold(strings.TrimSpace(z), room.Zone) {
			return false
		}
	}
	return true
}

// considerMeeting runs in sync for an online character with no bonded
// companion, and bonds one when the moment is right.
func (m *AICompanionModule) considerMeeting(u *users.UserRecord, round uint64) {
	if !m.cfg.AutoBond || u.Character == nil {
		return
	}
	rec := m.bonds.Users[u.UserId]
	if rec != nil && rec.Declined {
		// Sent away for good: the companion does not come back on its own.
		return
	}
	w, pending := m.pendingMeet[u.UserId]
	if !pending {
		// An existing character who has never met a companion.
		if !m.cfg.AutoBondExisting || (rec != nil && rec.Met) {
			return
		}
		if m.pendingMeet == nil {
			m.pendingMeet = map[int]*meetWait{}
		}
		w = &meetWait{}
		m.pendingMeet[u.UserId] = w
	}

	room := rooms.LoadRoom(u.Character.RoomId)
	if !m.meetingRoomOK(room) || u.Character.IsInCombat() || u.Character.HasCondition(0) {
		w.RoomId = 0
		return
	}
	// Let the character take in the room before anyone walks up.
	if w.RoomId != room.RoomId {
		w.RoomId, w.Since = room.RoomId, round
		return
	}
	if round < w.Since+uint64(m.cfg.MeetDelayRounds) {
		return
	}

	p, ok := m.profiles[strings.ToLower(m.cfg.AutoBondProfile)]
	if !ok {
		mudlog.Error(`aicompanion`, `action`, `meet`, `error`, fmt.Sprintf(`AutoBondProfile %q is not loaded`, m.cfg.AutoBondProfile))
		delete(m.pendingMeet, u.UserId)
		return
	}
	arrival := p.Meeting
	if arrival == `` {
		arrival = `comes up the path and stops a few paces away, looking you over.`
	}
	line := fmt.Sprintf(`<ansi fg="mobname">%s</ansi> %s`, p.Name, util.EscapeAnsiTags(arrival))
	if err := m.bondTo(u, p, line); err != nil {
		mudlog.Error(`aicompanion`, `action`, `meet`, `owner`, u.UserId, `error`, err)
		delete(m.pendingMeet, u.UserId)
		return
	}
	delete(m.pendingMeet, u.UserId)
	m.meetingPlace[u.UserId] = strings.TrimSpace(room.Title)
	if m.cfg.RequireConsent {
		if rec := m.bonds.Users[u.UserId]; rec != nil {
			rec.AskedAt = time.Now().Unix()
			m.saveBonds()
		}
		u.SendText(messaging.CategorySystem, consentQuestion(p.Name))
	}
}

// leave ends the bond at the companion's own decision: the owner clearly
// asked it to go, or it has come to dislike and distrust them completely
// (F5.7). It says goodbye in its own words first (already spoken), walks
// off, and will not come back on its own.
func (m *AICompanionModule) leave(c *controller, owner *users.UserRecord, why string) {
	if owner == nil || owner.Character == nil {
		return
	}
	now := time.Now().Unix()
	if m.mayRemember(c) {
		c.mind.addMemory(Memory{Unix: now, Kind: `event`, Text: `I parted ways with ` + owner.Character.Name + `. ` + why,
			Importance: 9, Emotion: `sadness`, People: []string{owner.Character.Name}}, m.cfg.MaxMemories)
		c.dirty = true
	}
	profileId := c.profile.Id
	if _, err := m.unbond(owner, `turns and walks away without looking back.`); err != nil {
		mudlog.Error(`aicompanion`, `action`, `leave`, `owner`, owner.UserId, `error`, err)
		return
	}
	m.markBond(owner.UserId, profileId, true)
	owner.SendText(messaging.CategorySystem, `Your companion has gone their own way.`)
}

// leaveRequestAllowed is the rule for a companion ASKING to part ways: its
// owner said something to it in this very moment (so it may have been told
// to go), or the relationship has collapsed entirely. Asking is never the
// same as going: the bond only ends when the owner confirms with
// `companion-part`, or when the companion has nothing left to stay for.
func leaveRequestAllowed(o Opinion, stims []stimulus) bool {
	if ownerAskedNow(stims) {
		return true
	}
	return o.Trust <= -70 && o.Affection <= -70
}

// collapsed reports a relationship with nothing left in it, the one case
// where a companion leaves without being told twice.
func collapsed(o Opinion) bool {
	return o.Trust <= -85 && o.Affection <= -85
}

// requestLeave is what the model's "leave" actually does: the companion has
// said its piece and now holds back, waiting for its owner to mean it. The
// owner confirms with `companion-part` (or takes it back by saying so).
func (m *AICompanionModule) requestLeave(c *controller, owner *users.UserRecord, why string) {
	c.leaveAskedAt = time.Now().Unix()
	c.leaveWhy = why
	c.mind.Autonomy = autonomyClose
	if m.mayRemember(c) {
		c.mind.addLine(Line{Kind: `event`, Text: `You told ` + owner.Character.Name + ` you would go, and hung back to see if they meant it.`},
			m.cfg.WorkingMemoryLines)
	}
	c.dirty = true
	owner.SendText(messaging.CategorySystem, fmt.Sprintf(
		`(If you truly want %s to leave for good, type <ansi fg="command">companion-part</ansi> within %d minutes. Otherwise just carry on; they will stay.)`,
		c.profile.Name, m.cfg.LeaveConfirmSeconds/60))
}

// confirmPart is the owner's side of parting ways, from the
// `companion-part` command: it ends the bond when the companion has asked
// to go, or when the owner asks twice in a row.
func (m *AICompanionModule) confirmPart(c *controller, owner *users.UserRecord) string {
	now := time.Now().Unix()
	asked := c.leaveAskedAt > 0 && now-c.leaveAskedAt <= int64(m.cfg.LeaveConfirmSeconds)
	if !asked && (c.partAskedAt == 0 || now-c.partAskedAt > 60) {
		c.partAskedAt = now
		return fmt.Sprintf(`%s looks at you and waits. Type companion-part again within a minute to part ways for good; their memories are kept, but they will not come back on their own.`, c.profile.Name)
	}
	m.leave(c, owner, c.leaveWhy)
	return ``
}

// The consent question. It is authored text, not a model call, and the
// answer is matched literally, so nothing a player says reaches OpenAI
// before they have agreed that it may.

const consentPhrase = `i agree`
const refusePhrase = `i decline`

// consentWindowSeconds is how long the question stays open after it is put.
// Outside it, and once it has been answered either way, those words are
// just words: the answer is only read when the question is on the table.
const consentWindowSeconds = 600

// consentQuestion is what she puts to a new companion, in brackets, as
// plainly as it can be put.
func consentQuestion(name string) string {
	return fmt.Sprintf(
		`(%s is driven by OpenAI. If you agree, what you say to %s, and what happens around you both, is sent to OpenAI to decide what %s says, and is kept on this server where administrators can read it. %s forms an opinion of you over time, and if you travel together long enough that may include becoming close to you, which only ever moves at your word. Say "%s" now to agree, or "%s" to keep %s as a plain companion: they will still travel with you, fight beside you and answer with a few set lines, and nothing you say will be sent anywhere. You can change your mind later with "companion-ai on" or "companion-ai off"; the question itself is only asked this once.)`,
		name, name, name, name, consentPhrase, refusePhrase, name)
}

// consented reports whether this owner has agreed to the model being used
// for their companion. Without agreement she is an ordinary companion.
func (m *AICompanionModule) consented(ownerUserId int) bool {
	if !m.cfg.RequireConsent {
		return true
	}
	rec := m.bonds.Users[ownerUserId]
	return rec != nil && rec.Consented
}

// mayRemember reports whether a deed with a person's name in it (a gift,
// an attack, healing, her owner calling her, a fight, a fall, a party) may
// be written into her mind. Her mind is what gets sent, so, as with speech,
// nothing naming anyone is written down before her owner has agreed (or
// while they have turned it off): writing first and gating the send later
// would send it the moment they agreed. She still reacts to it: the
// stimulus is queued and dispatch answers with her set lines, and the
// rules that are her owner's own relationship (a gift's warmth, an attack's
// cost) still apply. Events that name nobody ("You reached the mill") are
// written as before.
func (m *AICompanionModule) mayRemember(c *controller) bool {
	return m.consented(c.ownerUserId)
}

// consentLedger is the model door's own copy of who has agreed. The bond
// records live under the mud lock, and requests leave from goroutines that
// do not hold it, so the door in send reads this instead. It is rebuilt
// from the bond records whenever they are loaded or saved. Its zero value
// agrees to nothing, so a module that never filled it sends nothing.
type consentLedger struct {
	mu      sync.Mutex
	open    bool         // this server does not ask for consent at all
	agreed  map[int]bool // owners who have said yes
	lastLog time.Time    // last refusal written to the log
}

// allows reports whether a request carrying this owner's data may leave.
// An unknown owner never may, whatever the server's setting.
func (l *consentLedger) allows(ownerUserId int) bool {
	if l == nil || ownerUserId <= 0 {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.open || l.agreed[ownerUserId]
}

// noteRefusal logs a request the door turned away, at most once a minute.
// Every caller checks consent before building a request, so a refusal here
// means one of them forgot: worth a line, not a flood.
func (l *consentLedger) noteRefusal(ownerUserId int, path string) {
	if l != nil {
		l.mu.Lock()
		now := time.Now()
		if now.Sub(l.lastLog) < time.Minute {
			l.mu.Unlock()
			return
		}
		l.lastLog = now
		l.mu.Unlock()
	}
	mudlog.Error(`aicompanion`, `action`, `consentDoor`, `owner`, ownerUserId, `path`, path,
		`error`, `a request for an owner who has not agreed reached the door; a caller is missing its consent check`)
}

// syncConsent rebuilds the door's copy from the bond records. Runs under
// the mud lock.
func (m *AICompanionModule) syncConsent() {
	agreed := make(map[int]bool, len(m.bonds.Users))
	for id, rec := range m.bonds.Users {
		if rec != nil && rec.Consented {
			agreed[id] = true
		}
	}
	m.consent.mu.Lock()
	m.consent.open = !m.cfg.RequireConsent
	m.consent.agreed = agreed
	m.consent.mu.Unlock()
}

// answerConsent reads a literal yes or no from something the owner said. It
// returns true when the words were an answer, so the speech is not passed
// on as ordinary conversation.
func (m *AICompanionModule) answerConsent(c *controller, u *users.UserRecord, said string) bool {
	if !m.cfg.RequireConsent || m.consented(u.UserId) {
		return false
	}
	rec := m.bonds.Users[u.UserId]
	if rec == nil {
		return false
	}
	// These words only mean anything while the question is actually open:
	// asked, not yet answered, and answered soon. Otherwise "I agree" is an
	// ordinary thing to say to someone, and a player would find their own
	// conversation quietly swallowed by a settings prompt. Changing their
	// mind later is what companion-ai is for.
	if rec.Consented || rec.Refused || rec.AskedAt == 0 ||
		time.Now().Unix()-rec.AskedAt > consentWindowSeconds {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(said))
	answer = strings.Trim(answer, `."!,`)
	switch answer {
	case consentPhrase:
		rec.Consented, rec.Refused = true, false
		m.saveBonds()
		u.SendText(messaging.CategorySystem, fmt.Sprintf(
			`(Agreed. %s will answer in their own words from here on. "companion-ai off" stops it at any time, and "companion-part" ends the companionship altogether.)`, c.profile.Name))
		// Anything still queued was said before they agreed, and a queued
		// moment is exactly what the next call would carry.
		c.pending = nil
		m.keepFirstMeeting(c, u)
		return true
	case refusePhrase:
		rec.Refused, rec.Consented = true, false
		m.saveBonds()
		u.SendText(messaging.CategorySystem, fmt.Sprintf(
			`(Understood. %s stays with you as a plain companion, and nothing you say is sent anywhere. Use "companion-ai on" if you ever change your mind.)`,
			c.profile.Name))
		return true
	}
	return false
}
