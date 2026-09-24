package messaging

import "testing"

// NOTE (2026-09-21): every input in this file puts the name at the START of
// the line, so the expected word is "A figure", capitalised. Anonymize used to
// substitute a flat lowercase "a figure" regardless of position, and these
// tests pinned that. It reads wrong in play next to every other line, and the
// room pipeline runs Anonymize BEFORE HideNames, so HideNames' capitalisation
// never got a chance on a tagged name. See
// anonymize_capitalisation_test.go, which pins both paths agreeing, including
// the mid-sentence case that must stay lowercase.

func TestAnonymizeReplacesUsernameTag(t *testing.T) {
	in := `<ansi fg="username">Calabe</ansi> attacks`
	want := `<ansi fg="combat-anon">A figure</ansi> attacks`
	if got := Anonymize(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAnonymizeReplacesMobnameTag(t *testing.T) {
	in := `<ansi fg="mobname">Thornwall Thug</ansi> snarls`
	want := `<ansi fg="combat-anon">A figure</ansi> snarls`
	if got := Anonymize(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAnonymizeReplacesIndexedAndSuffixedMobnameTags(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"duplicate_two", `<ansi fg="mobname-dup2">Thornwall Thug #2</ansi> snarls`},
		{"duplicate_four", `<ansi fg="mobname-dup4">Thornwall Thug #4</ansi> snarls`},
		{"display_suffix", `<ansi fg="mobname-target">Thornwall Thug</ansi> snarls`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := `<ansi fg="combat-anon">A figure</ansi> snarls`
			if got := Anonymize(tc.in); got != want {
				t.Fatalf("indexed/suffixed mob identity leaked: got %q want %q", got, want)
			}
		})
	}
}

func TestAnonymizeReplacesPetnameTag(t *testing.T) {
	in := `<ansi fg="petname">Rex</ansi> follows`
	want := `<ansi fg="combat-anon">A figure</ansi> follows`
	if got := Anonymize(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAnonymizeReplacesMultipleNamesInOneLine(t *testing.T) {
	in := `<ansi fg="mobname">Thug</ansi> strikes ` +
		`<ansi fg="username">Calabe</ansi> with a longsword`
	// The first name opens the line and capitalises; the second is mid
	// sentence and must not.
	want := `<ansi fg="combat-anon">A figure</ansi> strikes ` +
		`<ansi fg="combat-anon">a figure</ansi> with a longsword`
	if got := Anonymize(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAnonymizeLeavesOtherTagsAlone(t *testing.T) {
	in := `<ansi fg="hit-melee">strikes deeply</ansi>`
	if got := Anonymize(in); got != in {
		t.Fatalf("non-name tag must pass through, got %q", got)
	}
}

func TestAnonymizeEmpty(t *testing.T) {
	if got := Anonymize(""); got != "" {
		t.Fatalf("empty must pass through, got %q", got)
	}
}

// TestAnonymizeReplacesSuffixedUsernameTags guards the player half of the
// suffix rule. FormattedName.String renders `username-aggro` and
// `username-dead` as well as plain `username`. Until slice B,
// GetCharacterName(true) rendered `username-aggro` for any character not
// fighting a player, so the suffixed form was the COMMON one in room text; it
// is now the form a player reads for a foe fighting them. Only mob tags
// accepted a suffix, so a player's name reached infrared-only observers in
// full.
func TestAnonymizeReplacesSuffixedUsernameTags(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"aggro", `<ansi fg="username-aggro">Calabe</ansi> glows`},
		{"dead", `<ansi fg="username-dead">Calabe</ansi> glows`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want := `<ansi fg="combat-anon">A figure</ansi> glows`
			if got := Anonymize(tc.in); got != want {
				t.Fatalf("suffixed player identity leaked: got %q want %q", got, want)
			}
		})
	}
}

// TestAnonymizeLeavesLookalikeTagsAlone keeps the suffix rule from swallowing
// a colour alias that merely starts with an identity word.
func TestAnonymizeLeavesLookalikeTagsAlone(t *testing.T) {
	in := `<ansi fg="usernames">roster</ansi> <ansi fg="mobnamex">x</ansi>`
	if got := Anonymize(in); got != in {
		t.Fatalf("non-identity tag was anonymized: got %q", got)
	}
}

// FormattedName.String prints adjectives in a black-bold span after the identity
// tag, colour-patterned rune by rune. The room broadcast path anonymizes BEFORE
// it hides names, so the span has to go here or an infrared observer reads
// "a figure (dead)".
func TestAnonymizeTakesTheAdjectiveSpanWithTheTag(t *testing.T) {
	// Sentence-initial and mid-sentence forms. Every `in` below opens with a
	// name, so the first substitution capitalises; the one that also names a
	// second party mid-sentence does not.
	const anon = `<ansi fg="combat-anon">A figure</ansi>`
	const anonMid = `<ansi fg="combat-anon">a figure</ansi>`
	cases := []struct{ name, in, want string }{
		{"plain adjective", `<ansi fg="mobname">Skeleton</ansi> <ansi fg="black-bold">(dead)</ansi> recoils.`, anon + ` recoils.`},
		{"colour-patterned adjective", `<ansi fg="mobname-dead">Skeleton</ansi> <ansi fg="black-bold">(<ansi fg="52">☠</ansi><ansi fg="88">d</ansi><ansi fg="124">e</ansi><ansi fg="160">a</ansi><ansi fg="196">d</ansi>)</ansi> recoils.`, anon + ` recoils.`},
		{"two names, one adjectived", `<ansi fg="username">Kesh</ansi> <ansi fg="black-bold">(hidden)</ansi> hits <ansi fg="mobname-dup2">Rat #2</ansi>.`, anon + ` hits ` + anonMid + `.`},
		{"a black-bold span that is not adjectives stays", `<ansi fg="mobname">Skeleton</ansi> <ansi fg="black-bold">hisses</ansi>.`, anon + ` <ansi fg="black-bold">hisses</ansi>.`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Anonymize(tc.in); got != tc.want {
				t.Fatalf("Anonymize =\n  %q\nwant\n  %q", got, tc.want)
			}
		})
	}
}
