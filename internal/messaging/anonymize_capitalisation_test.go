package messaging

import (
	"regexp"
	"testing"
)

var capTestTagStripper = regexp.MustCompile(`<[^>]*>`)

func capPlain(s string) string { return capTestTagStripper.ReplaceAllString(s, "") }

// TestAnonymizeAndHideNamesAgreeOnCapitalisation pins that the two
// substitution paths capitalise identically.
//
// 🔴 WHY BOTH. rooms.sendTextVisualJudgedBy runs Anonymize BEFORE HideNames,
// deliberately, so whole name tags are matched first. That means Anonymize
// usually wins, and for a long time it substituted a flat lowercase
// "a figure" and never capitalised at all, so HideNames' capitalisation never
// got a chance on a tagged name. Read in play on 2026-09-21 among correctly
// capitalised siblings: "a figure moves with increasing swiftness."
//
// The `***` case is separate and was the same bug wearing a different hat:
// atSentenceStart treated '!' as a terminator but not '*', so a "!!!" banner
// capitalised and a "***" banner did not, in the same fight.
func TestAnonymizeAndHideNamesAgreeOnCapitalisation(t *testing.T) {
	const name = "Cave Crawler"
	tagged := `<ansi fg="mobname">` + name + `</ansi>`

	for _, tc := range []struct {
		why    string
		text   string
		shapes string
		blind  string
	}{
		{
			why:    "line begins with the name",
			text:   tagged + ` moves with increasing swiftness.`,
			shapes: `A figure moves with increasing swiftness.`,
			blind:  `Something moves with increasing swiftness.`,
		},
		{
			why:    "*** banner, the case that read lowercase in play",
			text:   `*** ` + tagged + ` lands a DEVASTATING SNAP on you! ***`,
			shapes: `*** A figure lands a DEVASTATING SNAP on you! ***`,
			blind:  `*** Something lands a DEVASTATING SNAP on you! ***`,
		},
		{
			why:    "!!! banner, which always worked",
			text:   `!!! ` + tagged + ` FUMBLES their snap! !!!`,
			shapes: `!!! A figure FUMBLES their snap! !!!`,
			blind:  `!!! Something FUMBLES their snap! !!!`,
		},
		{
			why:    "mid sentence stays lowercase",
			text:   `You shift your focus to ` + tagged + `!`,
			shapes: `You shift your focus to a figure!`,
			blind:  `You shift your focus to something!`,
		},
		{
			why:    "after a symbol banner and a sentence end",
			text:   `⚡ SWEEP! ` + tagged + ` dodges and lashes at your legs.`,
			shapes: `⚡ SWEEP! A figure dodges and lashes at your legs.`,
			blind:  `⚡ SWEEP! Something dodges and lashes at your legs.`,
		},
	} {
		t.Run(tc.why, func(t *testing.T) {
			if got := capPlain(HideNames(tc.text, []string{name}, SightShapes)); got != tc.shapes {
				t.Errorf("HideNames shapes:\n got %q\nwant %q", got, tc.shapes)
			}
			if got := capPlain(HideNames(tc.text, []string{name}, SightNone)); got != tc.blind {
				t.Errorf("HideNames blind:\n got %q\nwant %q", got, tc.blind)
			}
			// Anonymize has only the shapes word, and must agree on the case.
			if got := capPlain(Anonymize(tc.text)); got != tc.shapes {
				t.Errorf("Anonymize:\n got %q\nwant %q", got, tc.shapes)
			}
		})
	}
}
