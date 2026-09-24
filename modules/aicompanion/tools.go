package aicompanion

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Asking the game (phase 7). Within one decision the model may ask a few
// read-only questions before it answers: look closer at something, size
// someone up, see what a merchant sells, recall its own memories, or search
// its own map. "What do you think of this room?" can then be answered from
// the room as the companion actually sees it, not from a summary.
//
// Every answer is built from exactly what a player standing there could
// see or already knows (the same functions as the rest of the module's
// perception), plus the companion's own memories and its own map. The
// model goroutine takes the mud lock only while the answers are read, then
// releases it before calling the model again.

func toolSpecs() []toolSpec {
	ref := func(desc string) map[string]any {
		return object([]string{`ref`}, map[string]any{`ref`: str(desc)})
	}
	query := func(desc string) map[string]any {
		return object([]string{`query`}, map[string]any{`query`: str(desc)})
	}
	return []toolSpec{
		{Type: `function`, Function: toolDef{Name: `look_closer`, Strict: true,
			Description: `Look closely at something or someone you can see, as anyone standing there could: a [ref] from the list of things here, "here" for the place itself, or "owner" for your travelling companion. Others see you look.`,
			Parameters:  ref(`A [ref] (t2, p1, w1, s3), "here" or "owner".`)}},
		{Type: `function`, Function: toolDef{Name: `size_up`, Strict: true,
			Description: `Judge how a fight with someone here would go, as anyone can by sizing them up.`,
			Parameters:  ref(`The [ref] of a person or creature here, or "owner".`)}},
		{Type: `function`, Function: toolDef{Name: `check_wares`, Strict: true,
			Description: `See what the merchants here sell and for how much, as a customer would.`,
			Parameters:  object([]string{}, map[string]any{})}},
		{Type: `function`, Function: toolDef{Name: `recall`, Strict: true,
			Description: `Think back: search your own memories, what you know about your companion, and how past days went.`,
			Parameters:  query(`What you are trying to remember, in a few words.`)}},
		{Type: `function`, Function: toolDef{Name: `find_place`, Strict: true,
			Description: `Think back over the places you have been, and what you have been told about places, to remember where something is and how far.`,
			Parameters:  query(`What place or thing you are trying to remember the way to.`)}},
	}
}

type toolArgs struct {
	Ref   string `json:"ref"`
	Query string `json:"query"`
}

// answerTools answers the model's questions. Runs under the mud lock. It
// returns false if the companion is no longer the one the call was for.
func (m *AICompanionModule) answerTools(ownerId int, seq uint64, rev uint64, sc *scene, calls []toolCall) ([]string, bool) {
	c := m.ctrls[ownerId]
	if c == nil || c.seq != seq || c.worldRev != rev {
		return nil, false
	}
	mob := mobs.GetInstance(c.instanceId)
	owner := users.GetByUserId(ownerId)
	if mob == nil || owner == nil || owner.Character == nil {
		return nil, false
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return nil, false
	}
	out := make([]string, len(calls))
	for i, tc := range calls {
		var a toolArgs
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &a)
		a.Ref = strings.ToLower(strings.TrimSpace(a.Ref))
		a.Query = strings.TrimSpace(a.Query)
		out[i] = cleanText(m.answerTool(c, mob, owner, room, sc, tc.Function.Name, a), 1500)
		if out[i] == `` {
			out[i] = `Nothing more to learn there.`
		}
	}
	c.dirty = true
	return out, true
}

func (m *AICompanionModule) answerTool(c *controller, mob *mobs.Mob, owner *users.UserRecord, room *rooms.Room,
	sc *scene, name string, a toolArgs) string {

	switch name {
	case `look_closer`:
		switch a.Ref {
		case `here`:
			return describeRoomFully(mob, room)
		case `owner`:
			if owner.Character.RoomId != room.RoomId {
				return owner.Character.Name + ` is not here.`
			}
			if cannotSee(mob, room) || !mob.Character.Perceives(owner.Character) {
				return `You cannot see them just now; you only know they are close by.`
			}
			return describePerson(owner.Character, true)
		}
		t := sc.get(a.Ref)
		if t == nil || !stillThere(t, mob, room) {
			return `You cannot see that here.`
		}
		if t.Kind == `player` {
			if u := users.GetByUserId(t.UserId); u != nil && u.Character != nil {
				if cannotSee(mob, room) || !mob.Character.Perceives(u.Character) {
					return `You cannot make them out from here.`
				}
				return describePerson(u.Character, true)
			}
		}
		if t.Kind == `ware` {
			item := t.Item
			return fmt.Sprintf(`%s (for sale at %d gold): %s`, t.Name, t.Price, plainText(item.GetLongDescription()))
		}
		return m.lookAt(c, mob, room, t, 0, false).Perceived

	case `size_up`:
		if a.Ref == `owner` {
			return `You know ` + owner.Character.Name + ` too well to size them up like a stranger.`
		}
		t := sc.get(a.Ref)
		if t == nil || !stillThere(t, mob, room) {
			return `They are not here.`
		}
		out := m.consider(c, mob, room, t)
		if out.Perceived != `` {
			return out.Perceived
		}
		return out.Refused

	case `check_wares`:
		listings := browseShops(room)
		if len(listings) == 0 {
			return `No merchant here is open for business.`
		}
		now := time.Now().Unix()
		var parts []string
		for _, l := range listings {
			c.mind.rememberShop(l, room.RoomId, now)
			parts = append(parts, describeListing(l, nil))
		}
		return strings.Join(parts, ` `) + ` (To buy, browse first so the wares get refs.)`

	case `recall`:
		return recallAnswer(c.mind, a.Query, time.Now().Unix())

	case `find_place`:
		results := searchPlaces(c.mind, room.RoomId, a.Query, 5, time.Now().Unix())
		if len(results) == 0 {
			return `You cannot remember anywhere like that.`
		}
		return strings.Join(results, ` | `)
	}
	return `You cannot do that.`
}

// describeRoomFully is what `look` shows a player: the whole description,
// the notable features, the ways out, and who and what is here.
func describeRoomFully(mob *mobs.Mob, room *rooms.Room) string {
	if cannotSee(mob, room) {
		return `It is too dark to make anything out.`
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s. %s", strings.TrimSpace(room.Title), strings.Join(strings.Fields(room.GetDescription()), ` `))
	nouns := make([]string, 0, len(room.Nouns))
	for n := range room.Nouns {
		nouns = append(nouns, n)
	}
	sort.Strings(nouns)
	if len(nouns) > 0 {
		fmt.Fprintf(&b, ` Things worth a closer look: %s.`, strings.Join(nouns, `, `))
	}
	var exits []string
	for name, ex := range room.Exits {
		if !ex.Secret {
			exits = append(exits, name)
		}
	}
	sort.Strings(exits)
	if len(exits) > 0 {
		fmt.Fprintf(&b, ` Ways out: %s.`, strings.Join(exits, `, `))
	}
	if things := roomThings(room); len(things) > 0 {
		fmt.Fprintf(&b, ` Lying about: %s.`, strings.Join(things, `, `))
	}
	return b.String()
}

// describePerson is what `look <player>` shows: description, condition,
// kind, and what they wear and carry in plain view.
func describePerson(ch *characters.Character, showGear bool) string {
	var b strings.Builder
	d, _ := characters.ResolveDescriptionToken(ch.Description)
	fmt.Fprintf(&b, `%s: %s They look %s.`, ch.Name, plainText(d), healthWords(ch))
	if sp := species.GetSpecies(ch.SpeciesId); sp != nil && sp.Name != `` {
		fmt.Fprintf(&b, ` Kind: %s.`, strings.ToLower(sp.Name))
	}
	if showGear {
		var gear []string
		for _, it := range ch.Equipment.GetAllItems() {
			if it.ItemId > 0 {
				gear = append(gear, it.Name())
			}
		}
		if len(gear) > 0 {
			fmt.Fprintf(&b, ` Wearing or carrying: %s.`, strings.Join(gear, `, `))
		}
	}
	return b.String()
}

// recallAnswer searches the companion's own memories, facts and session
// summaries for a query (F3.11).
func recallAnswer(mind *Mind, query string, nowUnix int64) string {
	words := keywordsOf(query)
	if len(words) == 0 {
		return `You cannot bring anything particular to mind.`
	}
	score := func(text string) int {
		n := 0
		for w := range keywordsOf(text) {
			if words[w] {
				n++
			}
		}
		return n
	}
	type hit struct {
		text  string
		score float64
	}
	var hits []hit
	for _, mem := range mind.Memories {
		if s := score(mem.Text + ` ` + strings.Join(mem.People, ` `) + ` ` + mem.Place); s > 0 {
			prefix := ``
			if isVague(mem, nowUnix) {
				prefix = `(vaguely) `
			}
			hits = append(hits, hit{fmt.Sprintf(`(%s ago) %s%s`, humanizeElapsed(nowUnix-mem.Unix), prefix, mem.Text),
				float64(s) + float64(mem.Importance)/10})
		}
	}
	for _, f := range mind.Facts {
		if s := score(f.Text); s > 0 {
			hits = append(hits, hit{`You know: ` + f.Text, float64(s) + 0.5})
		}
	}
	for _, sm := range mind.Summaries {
		if s := score(sm.Text); s > 0 {
			hits = append(hits, hit{fmt.Sprintf(`(%s ago) %s`, humanizeElapsed(nowUnix-sm.Unix), sm.Text), float64(s)})
		}
	}
	if len(hits) == 0 {
		return `Nothing comes to mind about that.`
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	if len(hits) > 6 {
		hits = hits[:6]
	}
	lines := make([]string, len(hits))
	for i, h := range hits {
		lines[i] = h.text
	}
	return strings.Join(lines, ` | `)
}
