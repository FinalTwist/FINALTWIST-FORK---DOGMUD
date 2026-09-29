package baubles

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// Text validation for a model reply. The strict JSON schema fixes the SHAPE
// of the answer; everything the schema cannot say is checked here, and any
// failure sends the find down the generic-trinket path instead of into the
// world.

const (
	maxNameLen        = 40
	minNameWords      = 1
	maxNameWords      = 6
	maxNameSimpleLen  = 20
	minDescriptionLen = 20
	maxDescriptionLen = 400
	maxMaterialLen    = 30
)

var (
	tagRE        = regexp.MustCompile(`<[^>]*>`)
	ansiRE       = regexp.MustCompile("\x1b\\[[0-9;]*[A-Za-z]")
	whitespaceRE = regexp.MustCompile(`\s+`)
	nameSimpleRE = regexp.MustCompile(`^[a-z]{2,20}$`)
)

// reservedNouns are keywords the game's real items answer to. A bauble whose
// one-word keyword is one of these would hijack `get key`, `drink potion`,
// `wield sword` and the like, so the keyword is replaced with "trinket"
// (the name itself may keep the word: "Bent Iron Key" is fine to read).
var reservedNouns = map[string]bool{
	`key`: true, `sword`: true, `dagger`: true, `knife`: true, `axe`: true, `mace`: true,
	`bow`: true, `arrow`: true, `bolt`: true, `spear`: true, `staff`: true, `shield`: true,
	`potion`: true, `elixir`: true, `scroll`: true, `book`: true, `map`: true, `note`: true,
	`letter`: true, `token`: true, `coin`: true, `gold`: true, `ring`: true, `amulet`: true,
	`bread`: true, `food`: true, `water`: true, `ale`: true, `wine`: true, `rope`: true,
	`torch`: true, `lantern`: true, `lockpick`: true, `pick`: true, `corpse`: true,
	`bag`: true, `backpack`: true, `chest`: true, `crate`: true, `door`: true,
	`herb`: true, `ore`: true, `gem`: true, `bauble`: true,
	// Natural finds (prompt version 3) must not take the keywords of real
	// materials, quest items and props: the Reckoning Bone (quest 58), the
	// weighted and river stones, and crafting materials (a black pearl, a
	// shrimp shell, a leviathan tooth, a geode, amber, claws and fangs).
	`bone`: true, `stone`: true, `shell`: true, `tooth`: true, `geode`: true,
	`amber`: true, `pearl`: true, `claw`: true, `fang`: true, `crystal`: true,
}

// ErrUnusableReply wraps every reason a reply cannot be used.
var ErrUnusableReply = errors.New(`unusable bauble reply`)

// cleanLine strips markup and control characters and collapses whitespace.
func cleanLine(s string) string {
	s = ansiRE.ReplaceAllString(s, ``)
	s = tagRE.ReplaceAllString(s, ``)
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	s = strings.NewReplacer(`<`, ``, `>`, ``, "`", `'`).Replace(s)
	return strings.TrimSpace(whitespaceRE.ReplaceAllString(s, ` `))
}

// CleanReply validates and tidies the text of a model reply. It returns the
// cleaned reply, or an error wrapping ErrUnusableReply. Numbers are not
// touched here; ApplyLimits clamps those.
func CleanReply(r Reply) (Reply, error) {
	bad := func(format string, a ...any) (Reply, error) {
		return Reply{}, fmt.Errorf(`%w: `+format, append([]any{ErrUnusableReply}, a...)...)
	}

	r.Name = cleanLine(r.Name)
	r.NameSimple = strings.ToLower(cleanLine(r.NameSimple))
	r.Description = cleanLine(r.Description)
	r.Material = strings.ToLower(cleanLine(r.Material))

	words := strings.Fields(r.Name)
	if r.Name == `` || len(r.Name) > maxNameLen || len(words) < minNameWords || len(words) > maxNameWords {
		return bad(`name %q`, r.Name)
	}
	if strings.IndexFunc(r.Name, unicode.IsDigit) >= 0 {
		return bad(`name has digits: %q`, r.Name)
	}
	if len(r.Description) < minDescriptionLen || len(r.Description) > maxDescriptionLen {
		return bad(`description length %d`, len(r.Description))
	}
	if len(r.Material) > maxMaterialLen {
		r.Material = ``
	}

	// The keyword must be one plain lowercase word that no real item
	// answers to (reservedNouns, and every loaded item's own keyword and
	// head noun: items.AuthoredKeyword). Otherwise the name's own last word;
	// then its other words, last to first, of four letters or more ("silver"
	// for a Tarnished Silver Locket, when a real locket exists); then
	// "trinket". Players can still use any word of the name.
	if !usableKeyword(r.NameSimple) {
		r.NameSimple = genericNameSimple
		for i := len(words) - 1; i >= 0; i-- {
			w := strings.ToLower(strings.TrimSuffix(strings.TrimSuffix(words[i], `'s`), `'`))
			if (i == len(words)-1 || len(w) >= 4) && usableKeyword(w) {
				r.NameSimple = w
				break
			}
		}
	}
	return r, nil
}

// usableKeyword is a keyword a bauble may take.
func usableKeyword(w string) bool {
	return nameSimpleRE.MatchString(w) && len(w) <= maxNameSimpleLen && !reservedNouns[w] && !authoredKeyword(w)
}

// authoredKeyword is items.AuthoredKeyword. A variable for tests.
var authoredKeyword = items.AuthoredKeyword

// PlainText is text with markup, colour codes and control characters removed
// and whitespace collapsed: room text as it is sent to the model.
func PlainText(s string) string {
	return cleanLine(s)
}
