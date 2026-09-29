package baubles

import (
	"fmt"
	"regexp"
)

// playerKeyTextRE is every character text named on a player's own key may
// hold: ASCII letters, the space, and ' " - , . ! ? (spec S3, ruling 15).
// Anything else (digits, accents, lookalike letters from other scripts,
// colons, semicolons, slashes, any other symbol) refuses the reply. The
// text is CLEANED first, so curly quotes, en and em dashes and the ellipsis
// have already been folded to ' " - and ... (cleanLine). " is allowed
// because the fold produces it: it carries no markup once cleanLine has
// stripped tags and < >, and it cannot form a link.
var playerKeyTextRE = regexp.MustCompile(`^[A-Za-z ',.!?"-]*$`)

// CheckPlayerKeyText refuses a CLEANED reply (CleanReply) named on a
// player's own key whose name, keyword, description or material strays
// outside playerKeyTextRE, or has a run of periods followed by anything but
// a space, a " or the end. That text is written by a key the server does
// not control and read by other players, so it is held to plain words as
// well as moderated.
func CheckPlayerKeyText(r Reply) error {
	fields := [...]struct{ field, text string }{
		{`name`, r.Name}, {`name_simple`, r.NameSimple}, {`description`, r.Description}, {`material`, r.Material},
	}
	for _, f := range fields {
		if !playerKeyTextRE.MatchString(f.text) || !periodsEndSentences(f.text) {
			return fmt.Errorf(`%w: player-key %s is not plain text: %s`, ErrUnusableReply, f.field, quoteShort(f.text))
		}
	}
	return nil
}

// periodsEndSentences reports whether every run of periods in s (one, or an
// ellipsis) is followed by a space, a closing " or the end of s, so no
// period sits inside a word the way a domain's does (evil.com, a...b).
func periodsEndSentences(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '.' {
			continue
		}
		j := i
		for j < len(s) && s[j] == '.' {
			j++
		}
		if j < len(s) && s[j] != ' ' && s[j] != '"' {
			return false
		}
		i = j
	}
	return true
}
