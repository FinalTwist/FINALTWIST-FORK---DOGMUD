package aicompanion

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// Casting. The engine's combat AI already casts what a companion knows
// while a fight is on; this is for the rest of the time, and for a spell
// she chooses herself: mending her owner after a fight, warding a road, a
// light in a dark room.
//
// She is shown only the spells she has actually learned and can afford
// right now, by name, with what they do. The cast itself is the ordinary
// mob command, so the roll, the cost, the cooldown and the interruption
// rules are all the engine's.

// spellOption is a spell she knows and could cast at this moment.
type spellOption struct {
	Ref    string
	Id     string
	Name   string
	What   string // what it does, in a word: mends, wards, harms, lights
	Cost   int
	SelfOK bool // needs no target
}

// spellsReady lists what she knows, can pay for, and is allowed to cast.
func spellsReady(mob *mobs.Mob) []spellOption {
	if len(mob.Character.SpellBook) == 0 {
		return nil
	}
	ids := make([]string, 0, len(mob.Character.SpellBook))
	for id := range mob.Character.SpellBook {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var out []spellOption
	for _, id := range ids {
		sd := spells.GetSpell(id)
		if sd == nil || sd.MobOnly {
			continue
		}
		if sd.Cost > 0 && mob.Character.Conviction < sd.Cost {
			continue // not enough conviction to pay for it
		}
		if sd.HealthCost > 0 && mob.Character.Health <= sd.HealthCost {
			continue // it would kill her
		}
		out = append(out, spellOption{
			Ref: fmt.Sprintf(`m%d`, len(out)+1), Id: sd.SpellId, Name: sd.Name,
			What: spellWhat(sd), Cost: sd.Cost, SelfOK: sd.Targeting == combatvocab.TargetSelf,
		})
		if len(out) >= 8 {
			break
		}
	}
	return out
}

// spellWhat says what a spell is for in a word, from what it actually does
// rather than from its name.
func spellWhat(sd *spells.SpellData) string {
	switch sd.EffectType {
	case `heal`:
		return `mends`
	case `shield`:
		return `wards`
	case `damage`, `dot`, `drain_area`:
		return `harms`
	case `condition`:
		return `changes how someone is`
	case `knockdown`:
		return `puts someone down`
	case `charm`:
		return `turns someone friendly`
	}
	return `does something`
}

// spellLines render what she could cast, for the prompt.
func spellLines(opts []spellOption) []string {
	var out []string
	for _, o := range opts {
		line := fmt.Sprintf(`[%s] %s (%s`, o.Ref, o.Name, o.What)
		if o.Cost > 0 {
			line += `, and it costs you`
		}
		if o.SelfOK {
			line += `, on yourself`
		}
		out = append(out, line+`)`)
	}
	return out
}

func findSpellOption(opts []spellOption, ref string) (spellOption, bool) {
	ref = strings.ToLower(strings.TrimSpace(ref))
	for _, o := range opts {
		if o.Ref == ref {
			return o, true
		}
	}
	return spellOption{}, false
}

// castCommand builds the ordinary mob cast for a spell and a target: the
// engine's own syntax, with the exact creature or player named.
func castCommand(o spellOption, t *thing) string {
	switch {
	case o.SelfOK || t == nil:
		return `cast ` + o.Id
	case t.Kind == `player` && t.UserId > 0:
		return fmt.Sprintf(`cast %s @%d`, o.Id, t.UserId)
	case t.MobInstanceId > 0:
		return fmt.Sprintf(`cast %s #%d`, o.Id, t.MobInstanceId)
	}
	return `cast ` + o.Id
}
