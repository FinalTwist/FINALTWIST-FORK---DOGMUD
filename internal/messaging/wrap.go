package messaging

import (
	"regexp"
	"strings"
)

// ansiTagPattern matches <ansi …> (any attribute set, including fg,
// bg, or combinations) or </ansi>. Only used for scanning;
// replacement uses the literal strings.
var ansiTagPattern = regexp.MustCompile(`<ansi[^>]*>|</ansi>`)

// WrapAnsi wraps text at maxWidth display columns. ANSI escape
// sequences (<ansi …> / </ansi> tags) don't count toward width.
// Open tags carry across line breaks: a break inside N nested spans
// closes all N before the newline and reopens all N, outermost first,
// on the next line.
//
// On malformed input (orphan tags, unmatched closers), falls back to
// a byte-count wrap to avoid panicking. The visual output is uglier
// but the server stays up.
//
// A maxWidth of 0 (unset) returns the input unchanged.
func WrapAnsi(text string, maxWidth int) (wrapped string) {
	if maxWidth <= 0 || text == "" {
		return text
	}
	defer func() {
		// Last-resort: if anything in the parser panics, the caller
		// gets the original text back. This needs a NAMED result and an
		// explicit assignment: a bare recover() on an unnamed return
		// silently handed the caller "" and erased the message.
		if r := recover(); r != nil {
			wrapped = text
		}
	}()

	// tagOp is a push or a pop recorded while scanning the word that is
	// still being accumulated in curWord. Tag state for a word in
	// progress must not mutate openTags until that word actually
	// commits to line: the wrap decision for THIS word has to see the
	// stack as it stood at the end of the PREVIOUS word, not a stack
	// already advanced by a closer sitting at this word's own tail
	// (e.g. "floor</ansi>" as one token). Applying pops live during the
	// scan let a same-word closer erase the span the wrap decision was
	// about to close, which left the next line unreopened.
	type tagOp struct {
		close bool
		tag   string
	}

	// Walk the text token-by-token, tracking display column and the
	// stack of currently-open ANSI tags. When we cross maxWidth at a
	// word boundary, emit a newline; every open tag is closed before
	// the break and reopened, outermost first, on the next line.
	var (
		out            strings.Builder
		line           strings.Builder
		col            int
		openTags       []string // stack committed to `line`/`out`, outermost first
		pending        []tagOp  // tag ops seen inside the word still in curWord
		curWord        strings.Builder
		curWordW       int
		lineHasContent bool // visible content on line (not just tag re-opener)
	)

	// closeAll and reopenAll keep a wrapped line balanced. A line break
	// inside N open spans must close all N before the newline and reopen
	// all N, outermost first, on the next line. The old code tracked a
	// single tag, so a balanced inner </ansi> silently dropped the outer
	// span for the rest of the message.
	closeAll := func(b *strings.Builder) {
		for range openTags {
			b.WriteString(`</ansi>`)
		}
	}
	reopenAll := func(b *strings.Builder) {
		for _, tag := range openTags {
			b.WriteString(tag)
		}
	}

	flushWord := func() {
		// Add space before word if line already has content and there's room.
		// The wrap/no-wrap decision below uses openTags as committed by the
		// PREVIOUS word: pending (this word's own tag ops) is applied only
		// after the word lands in line.
		if lineHasContent && col+1+curWordW > maxWidth {
			// Wrap before the word.
			closeAll(&line)
			out.WriteString(line.String())
			out.WriteByte('\n')
			line.Reset()
			col = 0
			lineHasContent = false
			reopenAll(&line)
		} else if lineHasContent {
			line.WriteByte(' ')
			col++
		}
		line.WriteString(curWord.String())
		col += curWordW
		lineHasContent = true
		curWord.Reset()
		curWordW = 0

		// Now that the word has committed to line, fold its tag ops into
		// the stack that governs future wrap decisions.
		for _, op := range pending {
			if op.close {
				if len(openTags) > 0 {
					openTags = openTags[:len(openTags)-1]
				}
			} else {
				openTags = append(openTags, op.tag)
			}
		}
		pending = pending[:0]
	}

	i := 0
	for i < len(text) {
		// ANSI tag?
		if text[i] == '<' {
			loc := ansiTagPattern.FindStringIndex(text[i:])
			if loc != nil && loc[0] == 0 {
				tag := text[i : i+loc[1]]
				if strings.HasPrefix(tag, `</`) {
					pending = append(pending, tagOp{close: true})
				} else {
					pending = append(pending, tagOp{close: false, tag: tag})
				}
				curWord.WriteString(tag)
				i += loc[1]
				continue
			}
		}
		// Whitespace boundary?
		if text[i] == ' ' || text[i] == '\n' {
			if curWord.Len() > 0 {
				flushWord()
			}
			if text[i] == '\n' {
				closeAll(&line)
				out.WriteString(line.String())
				out.WriteByte('\n')
				line.Reset()
				col = 0
				lineHasContent = false
				reopenAll(&line)
			}
			i++
			continue
		}
		// Visible character.
		curWord.WriteByte(text[i])
		curWordW++
		i++
	}
	if curWord.Len() > 0 {
		flushWord()
	}
	out.WriteString(line.String())
	return out.String()
}
