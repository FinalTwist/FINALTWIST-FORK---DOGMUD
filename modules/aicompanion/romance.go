package aicompanion

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Romance runs on its own track, and it is meant to be hard to reach. Being
// liked, even deeply, is not the same as being loved: attachment only moves
// on things the game actually saw happen between the two of them, never on
// the model's say-so, and every step up needs the owner to say yes in their
// own words through `companion-court`. A companion can travel with someone
// for a hundred sessions, like them enormously, and never take this road.
//
// The stages:
//
//	none     nothing of the kind, and most companions stay here
//	drawn    she has noticed, and has not said so
//	courting both know, and are finding out
//	together settled, and private about it
//	devoted  the rest of it, whatever that turns out to be
//
// What the module writes is her manner, her habits and what she remembers.
// What happens between them is the model's to tell, in her voice, guided by
// the `intimacy_style` her profile carries, which is the operator's text.

const (
	romanceNone     = `none`
	romanceDrawn    = `drawn`
	romanceCourting = `courting`
	romanceTogether = `together`
	romanceDevoted  = `devoted`
)

var romanceOrder = []string{romanceNone, romanceDrawn, romanceCourting, romanceTogether, romanceDevoted}

func romanceRank(stage string) int {
	for i, s := range romanceOrder {
		if s == stage {
			return i
		}
	}
	return 0
}

// Romance is the part of her mind this track lives in.
type Romance struct {
	Stage       string   `yaml:"stage,omitempty"`
	Attachment  int      `yaml:"attachment,omitempty"` // 0..100, earned only by milestones
	Milestones  []string `yaml:"milestones,omitempty"` // what earned it, in order
	Boundary    string   `yaml:"boundary,omitempty"`   // none, friendship (never romance)
	ProposedBy  string   `yaml:"proposed_by,omitempty"`
	ProposedAt  int64    `yaml:"proposed_at,omitempty"`
	DeclinedAt  int64    `yaml:"declined_at,omitempty"`
	StageAt     int64    `yaml:"stage_at,omitempty"`
	Nights      int      `yaml:"nights,omitempty"`
	LastNight   int64    `yaml:"last_night,omitempty"`
	SessionsAt  int      `yaml:"sessions_at,omitempty"`  // session count when the stage last changed
	LongRoadAt  int      `yaml:"long_road_at,omitempty"` // session the last long road was counted in
	LastGainDay string   `yaml:"last_gain_day,omitempty"`
	GainsToday  int      `yaml:"gains_today,omitempty"`
}

// ProfileRomance is what a character is willing to have happen at all.
type ProfileRomance struct {
	Romanceable bool              `yaml:"romanceable"`
	Orientation string            `yaml:"orientation,omitempty"`
	Pace        string            `yaml:"pace,omitempty"` // slow, ordinary
	Manner      map[string]string `yaml:"manner,omitempty"`
	Appearance  map[string]string `yaml:"appearance,omitempty"`
	IdleEmotes  []string          `yaml:"idle_emotes,omitempty"`
	// IntimacyStyle is the operator's own guidance for how far and in what
	// register a private scene is told. It is authored content, carried to
	// the model unchanged.
	IntimacyStyle string `yaml:"intimacy_style,omitempty"`
}

// milestoneWeights are the only things that move attachment, and they are
// all things the server watched happen.
var milestoneWeights = map[string]int{
	`defended`:     8, // put herself between her owner and harm
	`revived`:      9, // one of them brought the other back
	`survived`:     6, // a fight they both nearly died in
	`promise_kept`: 5, // a promise carried a long time and then kept
	`keepsake`:     4, // a gift she has kept for many sessions
	`long_road`:    3, // a long stretch travelled together
	`confided`:     5, // she told him something she tells nobody
}

var milestoneWords = map[string]string{
	`defended`:     `you stood between them and something that would have killed them`,
	`revived`:      `one of you brought the other back`,
	`survived`:     `you both nearly died in the same fight and did not`,
	`promise_kept`: `a promise you had both been carrying was kept`,
	`keepsake`:     `you still carry what they gave you`,
	`long_road`:    `a long stretch of road, just the two of you`,
	`confided`:     `you told them something you have told nobody else`,
}

// requirements to move up, all of which must hold.
type stageGate struct {
	Trust      int
	Affection  int
	Attachment int
	Milestones int
	Sessions   int // sessions that must pass at the stage below
}

var stageGates = map[string]stageGate{
	romanceDrawn:    {Trust: 50, Affection: 60, Attachment: 10, Milestones: 2, Sessions: 6},
	romanceCourting: {Trust: 60, Affection: 70, Attachment: 30, Milestones: 4, Sessions: 4},
	romanceTogether: {Trust: 70, Affection: 80, Attachment: 55, Milestones: 6, Sessions: 6},
	romanceDevoted:  {Trust: 85, Affection: 90, Attachment: 80, Milestones: 9, Sessions: 10},
}

// paceFactor stretches the requirements for a slow character.
func paceFactor(p *Profile) float64 {
	if strings.EqualFold(p.Romance.Pace, `ordinary`) {
		return 1
	}
	return 1.5 // slow is the default: this is meant to be an achievement
}

// nextStage is the stage above the current one, or "".
func nextStage(stage string) string {
	r := romanceRank(stage)
	if r+1 >= len(romanceOrder) {
		return ``
	}
	return romanceOrder[r+1]
}

// romanceReady reports whether everything needed for the next step is in
// place, and why not when it is not.
func romanceReady(mind *Mind, p *Profile) (string, bool, string) {
	if !p.Romance.Romanceable {
		return ``, false, `this is not that kind of companionship`
	}
	if mind.Romance.Boundary == `friendship` {
		return ``, false, `you have been asked to keep this to friendship`
	}
	next := nextStage(mind.Romance.Stage)
	if next == `` {
		return ``, false, `there is nowhere further to go`
	}
	g, ok := stageGates[next]
	if !ok {
		return ``, false, `there is nowhere further to go`
	}
	f := paceFactor(p)
	need := stageGate{
		Trust:      int(float64(g.Trust) * 1),
		Affection:  int(float64(g.Affection) * 1),
		Attachment: int(float64(g.Attachment) * f),
		Milestones: int(float64(g.Milestones) * f),
		Sessions:   int(float64(g.Sessions) * f),
	}
	o := mind.Opinion
	switch {
	case o.Trust < need.Trust || o.Affection < need.Affection:
		return next, false, `they do not feel that way about you yet`
	case mind.Romance.Attachment < need.Attachment:
		return next, false, `not enough has happened between you`
	case len(mind.Romance.Milestones) < need.Milestones:
		return next, false, `not enough has happened between you`
	case mind.SessionCount-mind.Romance.SessionsAt < need.Sessions:
		return next, false, `it is too soon`
	case mind.Romance.DeclinedAt > 0 && time.Now().Unix()-mind.Romance.DeclinedAt < 7*86400:
		return next, false, `you turned this down not long ago`
	}
	return next, true, ``
}

// addMilestone records something that happened between them and moves
// attachment, at most twice a day so nothing can be farmed.
func (m *Mind) addMilestone(kind string, nowUnix int64) bool {
	weight, ok := milestoneWeights[kind]
	if !ok {
		return false
	}
	day := time.Unix(nowUnix, 0).UTC().Format(`2006-01-02`)
	if m.Romance.LastGainDay != day {
		m.Romance.LastGainDay, m.Romance.GainsToday = day, 0
	}
	if m.Romance.GainsToday >= 2 {
		return false
	}
	m.Romance.GainsToday++
	m.Romance.Milestones = append(m.Romance.Milestones, kind)
	if len(m.Romance.Milestones) > 40 {
		m.Romance.Milestones = m.Romance.Milestones[len(m.Romance.Milestones)-40:]
	}
	m.Romance.Attachment = clampInt(m.Romance.Attachment+weight, 0, 100)
	return true
}

// setStage moves the relationship and remembers the day it changed.
func (m *Mind) setStage(stage string, nowUnix int64) {
	m.Romance.Stage = stage
	m.Romance.StageAt = nowUnix
	m.Romance.SessionsAt = m.SessionCount
	m.Romance.ProposedBy, m.Romance.ProposedAt = ``, 0
}

// romanceLines tell the model where this stands, what it would take, and
// how she carries herself now.
func romanceLines(mind *Mind, p *Profile, owner string) []string {
	if !p.Romance.Romanceable && mind.Romance.Stage == `` {
		return nil
	}
	stage := mind.Romance.Stage
	if stage == `` {
		stage = romanceNone
	}
	var out []string
	switch stage {
	case romanceNone:
		if mind.Romance.Boundary == `friendship` {
			out = append(out, owner+` has asked that this stay friendship. It stays friendship, and you do not raise it.`)
			return out
		}
		// Nothing has happened, so nothing is said. A line here telling her
		// there is nothing between them is an invitation to talk about
		// whether there is, which is not what a scout does over breakfast.
		return nil
	default:
		line := fmt.Sprintf(`Between you and %s: %s.`, owner, stage)
		if manner, ok := p.Romance.Manner[stage]; ok && strings.TrimSpace(manner) != `` {
			line += ` ` + strings.TrimSpace(manner)
		}
		out = append(out, line)
	}
	if mind.Romance.Nights > 0 {
		out = append(out, fmt.Sprintf(`Nights you have spent together: %s.`, countWords(mind.Romance.Nights)))
	}
	if len(mind.Romance.Milestones) > 0 {
		last := mind.Romance.Milestones[len(mind.Romance.Milestones)-1]
		if w, ok := milestoneWords[last]; ok {
			out = append(out, `What is between you was earned: most recently, `+w+`.`)
		}
	}
	if mind.Romance.ProposedBy == `me` {
		out = append(out, `You have said how you feel and are waiting to hear what they say. Do not press it.`)
	}
	if _, ready, why := romanceReady(mind, p); !ready && why != `` && stage != romanceDevoted {
		out = append(out, `You are not ready to ask for more: `+why+`. Let it be.`)
	}
	return out
}

// appearanceFor is how she has taken to carrying herself at this stage: her
// own authored lines, appended to what anyone sees when they look at her.
func appearanceFor(p *Profile, stage string) string {
	if stage == `` || stage == romanceNone {
		return ``
	}
	return strings.TrimSpace(p.Romance.Appearance[stage])
}

// applyAppearance keeps the live mob's description in step with the stage.
// The base description is the template's; the stage line is added to it, so
// a look at her shows how things stand without anything being said.
func applyAppearance(mob *mobs.Mob, p *Profile, mind *Mind) {
	if mob == nil {
		return
	}
	var base string
	if mind.BaseDescription != `` {
		base = mind.BaseDescription
	} else {
		mind.BaseDescription = strings.TrimSpace(mob.Character.Description)
		base = mind.BaseDescription
	}
	extra := appearanceFor(p, mind.Romance.Stage)
	if extra == `` {
		mob.Character.Description = base
		return
	}
	mob.Character.Description = strings.TrimSpace(base + ` ` + extra)
}

// nightReady reports whether the two of them are settled somewhere for the
// night: together, both at rest in the same room, and not lately.
func nightReady(mind *Mind, mob *mobs.Mob, owner *users.UserRecord, nowUnix int64) bool {
	if romanceRank(mind.Romance.Stage) < romanceRank(romanceTogether) {
		return false
	}
	if owner == nil || owner.Character == nil || owner.Character.RoomId != mob.Character.RoomId {
		return false
	}
	if mob.Character.IsInCombat() || owner.Character.IsInCombat() {
		return false
	}
	if nowUnix-mind.Romance.LastNight < 6*3600 {
		return false
	}
	if !owner.Character.HasConditionFlag(conditions.Sleeping) {
		return false
	}
	// Somewhere private: no other travellers in the room, and she has
	// settled too rather than standing watch over a sleeping man.
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return false
	}
	for _, uid := range room.GetPlayers() {
		if uid != owner.UserId {
			return false
		}
	}
	return mob.Character.HasConditionFlag(conditions.Sleeping)
}

// noteMilestone records something the server watched happen between them,
// remembers it, and lets her know it mattered. Nothing else moves
// attachment: not the model's opinion of the evening, not kind words.
func (m *AICompanionModule) noteMilestone(c *controller, kind string) {
	if c == nil || !c.profile.Romance.Romanceable || c.mind.Romance.Boundary == `friendship` || !m.consented(c.ownerUserId) {
		return
	}
	now := time.Now().Unix()
	if !c.mind.addMilestone(kind, now) {
		return
	}
	if words, ok := milestoneWords[kind]; ok {
		c.mind.addMemory(Memory{Unix: now, Kind: `event`, Text: `Something passed between us: ` + words + `.`,
			Importance: 7, Emotion: `affection`}, m.cfg.MaxMemories)
	}
	c.dirty = true
}

// tendRomance runs with the other slow bookkeeping: it counts the long
// roads, notices when she has come to feel something she has not said, and
// offers her a night when the two of them are settled somewhere.
func (m *AICompanionModule) tendRomance(c *controller, u *users.UserRecord, nowUnix int64) {
	p := c.profile
	// Before consent she is a plain companion: nothing about a romance is
	// counted, felt or raised.
	if !p.Romance.Romanceable || c.mind.Romance.Boundary == `friendship` || !m.consented(c.ownerUserId) {
		return
	}
	mob := mobs.GetInstance(c.instanceId)
	if mob == nil {
		return
	}
	applyAppearance(mob, p, c.mind)

	// A long stretch of road together, counted once a session.
	// Counted once a session, against its own marker: using the stage clock
	// for this would reset it every session, so the sessions a stage needs
	// could never accumulate.
	if nowUnix-c.sessionStartUnix > 45*60 && c.mind.Romance.LongRoadAt != c.mind.SessionCount {
		if c.mind.addMilestone(`long_road`, nowUnix) {
			c.dirty = true
		}
		c.mind.Romance.LongRoadAt = c.mind.SessionCount
	}

	// When everything is in place she says so once, in her own words, and
	// then waits. The step itself is the owner's to take.
	// A proposal that never reached her mouth (the call failed, the moment
	// passed) lapses, rather than leaving her waiting for an answer to
	// something she never said.
	if c.mind.Romance.ProposedBy == `me` && nowUnix-c.mind.Romance.ProposedAt > 900 {
		c.mind.Romance.ProposedBy, c.mind.Romance.ProposedAt = ``, 0
		c.dirty = true
	}
	if next, ready, _ := romanceReady(c.mind, p); ready && c.mind.Romance.ProposedBy == `` && !c.inFlight && len(c.pending) == 0 {
		c.mind.Romance.ProposedBy, c.mind.Romance.ProposedAt = `me`, nowUnix
		c.dirty = true
		c.push(stimulus{Kind: `romance`, Text: next, FromOwner: true})
		u.SendText(messaging.CategorySystem, fmt.Sprintf(
			`(If you feel the same, <ansi fg="command">companion-court</ansi> says so. <ansi fg="command">companion-boundary friendship</ansi> keeps things as they are.)`))
		return
	}

	// Settled somewhere for the night, with the door shut.
	if nightReady(c.mind, mob, u, nowUnix) && !c.inFlight && len(c.pending) == 0 {
		c.mind.Romance.LastNight = nowUnix
		c.mind.Romance.Nights++
		c.dirty = true
		c.push(stimulus{Kind: `night`, FromOwner: true})
	}
}

// courtStep is the owner accepting: the stage moves, and it is remembered.
func (m *AICompanionModule) courtStep(c *controller, u *users.UserRecord) string {
	p := c.profile
	// Nothing about a romance moves before her owner has agreed that she
	// may think for herself: until then she is a plain companion.
	if !m.consented(c.ownerUserId) {
		return fmt.Sprintf(`(%s is a plain companion for now, answering with a few set lines, so nothing more grows between you. "companion-ai on" changes that.)`, p.Name)
	}
	if !p.Romance.Romanceable {
		return c.profile.Name + ` is not looking for that, and says so kindly.`
	}
	if c.mind.Romance.Boundary == `friendship` {
		c.mind.Romance.Boundary = ``
		c.dirty = true
		return `You have lifted the line you drew. Nothing else has changed; the rest is up to the two of you.`
	}
	next, ready, why := romanceReady(c.mind, p)
	if !ready {
		if why == `` {
			why = `it is not the moment`
		}
		return fmt.Sprintf(`%s hears you, and it is plain that %s.`, c.profile.Name, why)
	}
	now := time.Now().Unix()
	c.mind.setStage(next, now)
	c.mind.addMemory(Memory{Unix: now, Kind: `event`,
		Text:       fmt.Sprintf(`%s and I became %s.`, u.Character.Name, next),
		Importance: 10, Emotion: `affection`, People: []string{u.Character.Name}}, m.cfg.MaxMemories)
	c.mind.applyOpinion(Opinion{Affection: 5, Trust: 3}, `romance_`+next, `rule`, `what was said between us`, false)
	m.recordCore(c, u.Character.Name, next, true)
	if mob := mobs.GetInstance(c.instanceId); mob != nil {
		applyAppearance(mob, p, c.mind)
	}
	c.dirty = true
	c.push(stimulus{Kind: `romance_yes`, Text: next, FromOwner: true})
	return ``
}

// setBoundary is the owner drawing a line, and it is remembered for good
// until they lift it.
func (m *AICompanionModule) setBoundary(c *controller, kind string) string {
	now := time.Now().Unix()
	switch kind {
	case `friendship`:
		c.mind.Romance.Boundary = `friendship`
		c.mind.Romance.DeclinedAt = now
		c.mind.Romance.ProposedBy, c.mind.Romance.ProposedAt = ``, 0
		if romanceRank(c.mind.Romance.Stage) > 0 {
			c.mind.setStage(romanceNone, now)
		}
		c.mind.addMemory(Memory{Unix: now, Kind: `event`, Text: `It was made clear to me that this is friendship, and nothing more.`,
			Importance: 8, Emotion: `sadness`}, m.cfg.MaxMemories)
		if owner := users.GetByUserId(c.ownerUserId); owner != nil && owner.Character != nil {
			m.recordCore(c, owner.Character.Name, `friendship`, false)
		}
		if mob := mobs.GetInstance(c.instanceId); mob != nil {
			applyAppearance(mob, c.profile, c.mind)
		}
		c.dirty = true
		c.push(stimulus{Kind: `romance_no`, FromOwner: true})
		return `You have said where the line is. They will not raise it again.`
	case `none`, ``:
		c.mind.Romance.Boundary = ``
		c.dirty = true
		return `The line you drew is lifted.`
	}
	return `Say "companion-boundary friendship" to keep things as they are, or "companion-boundary none" to lift it.`
}

// intimacyGuidance is the operator's text for how a private scene is told.
// It is only carried once there is a romance to tell about: before that it
// has nothing to describe and no business in the prompt.
func intimacyGuidance(mind *Mind, p *Profile) string {
	if romanceRank(mind.Romance.Stage) < romanceRank(romanceCourting) {
		return ``
	}
	return p.Romance.IntimacyStyle
}
