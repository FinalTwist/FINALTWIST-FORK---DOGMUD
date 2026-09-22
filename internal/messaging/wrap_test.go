package messaging

import (
	"strings"
	"testing"
)

func TestWrapAnsiShortLineUnchanged(t *testing.T) {
	got := WrapAnsi("short", 80)
	if got != "short" {
		t.Fatalf("short line should be unchanged, got %q", got)
	}
}

func TestWrapAnsiWrapsAtMaxWidthWordBoundary(t *testing.T) {
	// 20-col wrap, ~40 chars of content
	got := WrapAnsi("the quick brown fox jumps over the lazy dog", 20)
	// Expect at least one newline, and no line longer than 20 visible chars.
	lines := splitLines(got)
	if len(lines) < 2 {
		t.Fatalf("expected wrap to produce >=2 lines, got: %q", got)
	}
	for i, line := range lines {
		if displayWidth(line) > 20 {
			t.Fatalf("line %d exceeds 20 cols (%d): %q", i, displayWidth(line), line)
		}
	}
}

func TestWrapAnsiIgnoresAnsiTagsInWidth(t *testing.T) {
	// 12 visible chars wrapped inside a long ANSI tag.
	input := `<ansi fg="hit-melee">strikes hard</ansi>`
	got := WrapAnsi(input, 80)
	if got != input {
		t.Fatalf("12-visible-char line wrapped at 80 must be unchanged, got %q", got)
	}
}

func TestWrapAnsiCarriesOpenTagAcrossLineBreak(t *testing.T) {
	// 10-col wrap forces a break inside an open tag.
	input := `<ansi fg="hit-melee">strikes deeply at the heart</ansi>`
	got := WrapAnsi(input, 10)
	// First line should END with </ansi>; second line should START
	// with <ansi fg="hit-melee">.
	lines := splitLines(got)
	if len(lines) < 2 {
		t.Fatalf("expected break, got 1 line: %q", got)
	}
	if !endsWith(lines[0], `</ansi>`) {
		t.Fatalf("first line must close the open tag, got %q", lines[0])
	}
	if !startsWith(lines[1], `<ansi fg="hit-melee">`) {
		t.Fatalf("second line must reopen the tag, got %q", lines[1])
	}
}

func TestWrapAnsiMalformedTagFallback(t *testing.T) {
	// Orphan opening tag — wrapper must not panic; should fall back
	// to byte-count wrap.
	input := `<ansi fg="bad" missing close ` +
		`this is fifty-plus characters of unwrapped text after`
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("WrapAnsi panicked on malformed input: %v", r)
		}
	}()
	_ = WrapAnsi(input, 20)
}

func TestWrapAnsiZeroWidthIsPassthrough(t *testing.T) {
	// LineWidth=0 (unset) must not wrap or hang.
	got := WrapAnsi("a long line that would otherwise wrap", 0)
	if got != "a long line that would otherwise wrap" {
		t.Fatalf("LineWidth=0 must pass through, got %q", got)
	}
}

func TestWrapAnsiExplicitNewlineInsideOpenTagNoLeadingSpace(t *testing.T) {
	got := WrapAnsi(`<ansi fg="red">line one`+"\n"+`line two</ansi>`, 80)
	lines := splitLines(got)
	if len(lines) < 2 {
		t.Fatalf("expected 2 lines after explicit newline, got %d: %q", len(lines), got)
	}
	// Second line must start with the reopener tag followed directly
	// by a visible character — no leading space.
	if !startsWith(lines[1], `<ansi fg="red">line`) {
		t.Fatalf("explicit-newline second line had leading space (or wrong reopen): %q", lines[1])
	}
}

func TestWrapAnsiRegexCoversBgAndCombinedAttrs(t *testing.T) {
	// <ansi bg="…"> alone.
	bg := `<ansi bg="black">moonlight</ansi>`
	got := WrapAnsi(bg, 80)
	if got != bg {
		t.Fatalf("bg-only tag should pass through 80-col wrap, got %q", got)
	}
	// Combined fg + bg.
	combo := `<ansi fg="red" bg="black">alert</ansi>`
	got = WrapAnsi(combo, 80)
	if got != combo {
		t.Fatalf("fg+bg combo should pass through 80-col wrap, got %q", got)
	}
	// Width-counting must exclude the tag content. 9 visible chars
	// ("moonlight") with maxWidth=20 stays single-line.
	if c := displayWidth(got); c > 20 {
		t.Fatalf("displayWidth %d unexpectedly high — regex may not be stripping the combined tag", c)
	}
}

// Test helpers — kept here, not exported.

func splitLines(s string) []string {
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	out = append(out, cur)
	return out
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// TestDisplayWidthCountsRunesNotBytes pins the test oracle itself. Every
// width assertion in this package leans on displayWidth, so an oracle that
// counts bytes makes those assertions meaningless for any non-ASCII text.
// Fixed before WrapAnsi's own byte-counting bug, so that fix has a truthful
// judge. Matches internal/messaging/hidenames.go:70, which already decodes
// runes correctly in this same package.
func TestDisplayWidthCountsRunesNotBytes(t *testing.T) {
	// Twenty two-byte runes: 40 bytes, 20 visible columns.
	s := strings.Repeat("é", 20) // e-acute, 2 bytes each in UTF-8
	if got := displayWidth(s); got != 20 {
		t.Fatalf("displayWidth(20 two-byte runes) = %d, want 20: the oracle is counting bytes", got)
	}

	// Same, wrapped in a tag the oracle must not count.
	tagged := `<ansi fg="username">` + s + `</ansi>`
	if got := displayWidth(tagged); got != 20 {
		t.Fatalf("displayWidth(tagged) = %d, want 20: tag content must not count toward width", got)
	}
}

func displayWidth(s string) int {
	// Inline scan to count visible RUNES (skip <ansi …> and </ansi>).
	// Ranging over a string decodes runes; indexing it would count bytes
	// and make every width assertion in this file lie about non-ASCII
	// text. See hidenames.go, which decodes runes for the same reason.
	w := 0
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			w++
		}
	}
	return w
}
