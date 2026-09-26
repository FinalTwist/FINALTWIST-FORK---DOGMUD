# M5 PR 2: The Wrap Decision Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Repair `WrapAnsi`, then turn the pipeline's dead wrap stage on for the 44 narration categories that are safe to fold and leave it off for the 17 that are not.

**Architecture:** The wrap stage has existed since chunk 7 and has never fired, because `shouldWrap` returns `false` for every category. It cannot simply be switched on: `WrapAnsi` loses color across balanced nested tags and counts bytes instead of runes, and two categories carry pre-formatted tables, banners and ASCII art alongside free prose. So the wrapper is fixed first, then `shouldWrap` becomes an explicit allowlist on the `skipStages` idiom, then guards pin both the membership and the untouched-table invariant.

**Tech Stack:** Go, `internal/messaging` (`wrap.go`, `pipeline.go`, `normalize.go`), `unicode/utf8`.

---

## Facts verified against source

Read from the tree on 2026-09-22 at `a5613be95`. Rows marked REPRODUCED were executed in a probe this session, not inferred.

| # | Fact | Evidence |
|---|------|----------|
| 1 | `shouldWrap` returns `false` for every category, so stage 6 never fires | `internal/messaging/pipeline.go:97-99` |
| 2 | It is pinned off by a test looping the whole enum | `internal/messaging/pipeline_test.go:51-57`, `TestShouldWrapDisabledByDefault` |
| 3 | 61 categories are declared, bounded by the `categoryMax` sentinel | `internal/messaging/messaging.go:20-109` |
| 4 | REPRODUCED. `WrapAnsi` tracks ONE scalar `openTag` and clears it on ANY close, so balanced nesting loses the outer span on every continuation line | `internal/messaging/wrap.go:44`, `:85` |
| 5 | `applyCategoryColor` wraps every non-default category in an outer `<ansi>`, so fact 4 is the common case, not an edge | `internal/messaging/pipeline.go:111-116` |
| 6 | REPRODUCED. `WrapAnsi` counts BYTES: 20 two-byte runes plus `t ail` broke at visible column 21 against width 40 | `internal/messaging/wrap.go:116-117` |
| 7 | The test oracle `displayWidth()` counts bytes the same way, so it cannot catch fact 6 | `internal/messaging/wrap_test.go:133-152` |
| 8 | The same package already decodes runes correctly elsewhere | `internal/messaging/hidenames.go:70` |
| 9 | Prior art for per-category policy is a `switch` over an explicit list | `internal/messaging/normalize.go:25-37`, `skipStages` |
| 10 | 🔴 `CategoryRoomDescription` is ALREADY PROVEN unwrappable: a regression test pins the side-by-side minimap layout | `internal/messaging/wrap_test.go:75`, `TestRoomDescriptionSkipsWrap` |
| 11 | 🔴 `CategorySplash` carries rendered ASCII ART from a template | `internal/hooks/Splash_Deliver.go:31-38` |
| 12 | 🪤 Splash's own comment claims "per-recipient line wrapping happens later in SendText". That is false today and would be destructive if Splash were admitted | `internal/hooks/Splash_Deliver.go:25` |
| 13 | `CategorySkillProgress` carries a banner, and `skipStages` already excludes it with the comment "banner has its own formatting" | `internal/banner/banner.go:9`; `normalize.go:32` |
| 14 | `CategorySystem` carries both one-line refusals AND the score sheet, inventory, `who`, help, admin `DynamicList` tables and the ASCII map | `status.go:14`, `inventory.go:445`, `who.go:16`, `help.go:93`, `skill.map.go:71,212`, `admin.item.go:109`; refusals at `cast_admission.go:64,83` |
| 15 | `CategoryBroadcast` carries both the hand-drawn MOTD box and free channel chat | `motd.go:77`; `ChannelMessage_SendToAll.go:30` |
| 16 | All four speech self-echo sites pre-wrap at a HARDCODED 80, ignoring the reader's `LineWidth` | `say.go:39`, `shout.go:63`, `reply.go:37`, `whisper.go:56`, all `util.SplitStringNL(..., 80)` |
| 17 | Four categories have ZERO production sends: `GrappleHigh`, `Login`, `OOC`, `Toxin` | counted across `internal/` and `modules/`, excluding tests and the enum file |
| 18 | `CategoryCombatSummary` carries a per-round prose tally line, not a table | `internal/hooks/combat_verbosity.go:397-410` |
| 19 | `CategoryTip` sends 10 times in production; 64 of 74 shipped tips exceed 80 characters with the `[Tip] ` prefix, longest 211 | measured from `_datafiles/world/dogmud/tips.yaml` |
| 20 | `LineWidth` is plumbed into `RenderInput` and reaches the wrap call, so no new plumbing is needed | `internal/users/userrecord.go:486-500`; `pipeline.go:82` |

---

## Task 1: Fix the test oracle before anything leans on it

Every width assertion in this package calls `displayWidth`. It counts bytes (fact 7), exactly like the bug in `WrapAnsi` (fact 6). Fixing the function while the oracle shares its defect would produce a green suite that proves nothing, which is the "a check that cannot fail is not a check" family this project has been bitten by repeatedly.

**Files:**
- Modify: `internal/messaging/wrap_test.go:1-3` (imports), `:133-152` (the oracle)

- [ ] **Step 1: Write the failing test for the ORACLE itself**

`wrap_test.go` currently imports only `testing`. Widen the import block first, since this test and Tasks 2 and 3 all need `strings`:

```go
import (
	"strings"
	"testing"
)
```

Then add to `internal/messaging/wrap_test.go`:

```go
// TestDisplayWidthCountsRunesNotBytes pins the test oracle itself. Every
// width assertion in this package leans on displayWidth, so an oracle that
// counts bytes makes those assertions meaningless for any non-ASCII text.
// Fixed before WrapAnsi's own byte-counting bug, so that fix has a truthful
// judge. Matches internal/messaging/hidenames.go:70, which already decodes
// runes correctly in this same package.
func TestDisplayWidthCountsRunesNotBytes(t *testing.T) {
	// Twenty two-byte runes: 40 bytes, 20 visible columns.
	s := strings.Repeat("\u00e9", 20) // e-acute, 2 bytes each in UTF-8
	if got := displayWidth(s); got != 20 {
		t.Fatalf("displayWidth(20 two-byte runes) = %d, want 20: the oracle is counting bytes", got)
	}

	// Same, wrapped in a tag the oracle must not count.
	tagged := `<ansi fg="username">` + s + `</ansi>`
	if got := displayWidth(tagged); got != 20 {
		t.Fatalf("displayWidth(tagged) = %d, want 20: tag content must not count toward width", got)
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/messaging/ -run TestDisplayWidthCountsRunesNotBytes -v`

Expected: FAIL with `displayWidth(20 two-byte runes) = 40, want 20: the oracle is counting bytes`

- [ ] **Step 3: Fix the oracle**

Replace the body of `displayWidth` in `internal/messaging/wrap_test.go`:

```go
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
```

- [ ] **Step 4: Run the new test and the whole package**

Run: `go test ./internal/messaging/ -run TestDisplayWidthCountsRunesNotBytes -v`
Expected: PASS

Run: `go test ./internal/messaging/`
Expected: `ok  github.com/GoMudEngine/GoMud/internal/messaging`

All existing assertions use ASCII, so none of them changes value. That is the expected outcome and it is why this task is safe to land alone: the oracle is now capable of judging Task 3, where it previously could not.

- [ ] **Step 5: Commit**

```bash
git add internal/messaging/wrap_test.go
git commit -F - <<'EOF'
test(messaging): make displayWidth count runes, so it can judge the wrap fix

Every width assertion in this package leans on displayWidth, and it
counted bytes: twenty two-byte runes measured 40 columns instead of 20.
WrapAnsi has the identical bug, so fixing the function against this
oracle would have produced a green suite that proved nothing.

The oracle lands first and gets its own test. hidenames.go already
decodes runes correctly in this same package, so this is the package's
established practice, not a new one.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 2: Give WrapAnsi a tag stack

`wrap.go:44` declares `openTag string`. Any closing tag sets it to `""` (`:85`), so in `<ansi fg="214">outer <ansi fg="item">inner</ansi> more outer</ansi>` the inner close discards the outer span. Since `applyCategoryColor` puts an outer tag around every non-default category (fact 5) and narration routinely carries inner `mobname`, `username` and `item` tags, this fires on ordinary combat text, not on a contrived input.

**Files:**
- Modify: `internal/messaging/wrap.go:40-125`
- Test: `internal/messaging/wrap_test.go`

- [ ] **Step 1: Write the failing regression test**

This input was reproduced against the current code during the M5 design session. Add to `internal/messaging/wrap_test.go`:

```go
// TestWrapAnsiReopensNestedTagsAcrossBreak is the regression test for the
// scalar-openTag bug. The input is the exact shape applyCategoryColor
// produces: an outer category tag around prose that already carries an
// inner item tag. Before the stack fix, the inner </ansi> cleared the
// tracker and every continuation line rendered uncolored.
func TestWrapAnsiReopensNestedTagsAcrossBreak(t *testing.T) {
	in := `<ansi fg="214">The <ansi fg="item">iron sword</ansi> shatters into ` +
		`a thousand bright splinters that scatter across the floor</ansi>`

	got := WrapAnsi(in, 40)
	lines := splitLines(got)
	if len(lines) < 2 {
		t.Fatalf("expected the input to wrap into several lines, got %d:\n%s", len(lines), got)
	}

	// Every line after the first is still inside the outer fg="214" span,
	// so each must open with a reopener.
	for i, line := range lines[1:] {
		if !startsWith(line, `<ansi fg="214">`) {
			t.Errorf("line %d does not reopen the outer span: %q", i+2, line)
		}
	}

	// Every line must also be balanced: as many opens as closes.
	for i, line := range lines {
		opens := strings.Count(line, "<ansi")
		closes := strings.Count(line, "</ansi>")
		if opens != closes {
			t.Errorf("line %d is unbalanced (%d opens, %d closes): %q", i+1, opens, closes, line)
		}
	}
}

// TestWrapAnsiHandlesThreeLevelNesting proves the stack is a stack and not
// a two-slot special case.
func TestWrapAnsiHandlesThreeLevelNesting(t *testing.T) {
	in := `<ansi fg="214">alpha <ansi fg="item">bravo <ansi fg="username">charlie</ansi> ` +
		`delta</ansi> echo foxtrot golf hotel india juliet kilo lima</ansi>`

	got := WrapAnsi(in, 30)
	for i, line := range splitLines(got) {
		opens := strings.Count(line, "<ansi")
		closes := strings.Count(line, "</ansi>")
		if opens != closes {
			t.Errorf("line %d is unbalanced (%d opens, %d closes): %q", i+1, opens, closes, line)
		}
	}
}
```

`wrap_test.go` must import `strings`. Check the import block and add it if absent.

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/messaging/ -run "TestWrapAnsiReopensNestedTagsAcrossBreak|TestWrapAnsiHandlesThreeLevelNesting" -v`

Expected: FAIL. `TestWrapAnsiReopensNestedTagsAcrossBreak` reports `line 2 does not reopen the outer span:` followed by a line beginning `bright splinters`, and unbalanced-line errors on lines 2 and 3.

- [ ] **Step 3: Replace the scalar with a stack**

In `internal/messaging/wrap.go`, change the declaration block (currently `:40-48`) and add two closures immediately after it:

```go
	var (
		out            strings.Builder
		line           strings.Builder
		col            int
		openTags       []string // stack of currently-open tags, outermost first
		curWord        strings.Builder
		curWordW       int
		lineHasContent bool // visible content on line (not just tag re-openers)
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
```

Replace the wrap-before-word branch inside `flushWord`:

```go
	flushWord := func() {
		// Add space before word if line already has content and there's room.
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
	}
```

Replace the tag-scanning branch in the main loop:

```go
		// ANSI tag?
		if text[i] == '<' {
			loc := ansiTagPattern.FindStringIndex(text[i:])
			if loc != nil && loc[0] == 0 {
				tag := text[i : i+loc[1]]
				if strings.HasPrefix(tag, `</`) {
					if len(openTags) > 0 {
						openTags = openTags[:len(openTags)-1]
					}
				} else {
					openTags = append(openTags, tag)
				}
				curWord.WriteString(tag)
				i += loc[1]
				continue
			}
		}
```

Replace the explicit-newline branch:

```go
			if text[i] == '\n' {
				closeAll(&line)
				out.WriteString(line.String())
				out.WriteByte('\n')
				line.Reset()
				col = 0
				lineHasContent = false
				reopenAll(&line)
			}
```

Update the doc comment on `WrapAnsi` (`:13-17`) so it describes the stack:

```go
// WrapAnsi wraps text at maxWidth display columns. ANSI escape
// sequences (<ansi …> / </ansi> tags) don't count toward width.
// Open tags carry across line breaks: a break inside N nested spans
// closes all N before the newline and reopens all N, outermost first,
// on the next line.
```

- [ ] **Step 4: Run the new tests, then the whole package**

Run: `go test ./internal/messaging/ -run "TestWrapAnsiReopensNestedTagsAcrossBreak|TestWrapAnsiHandlesThreeLevelNesting" -v`
Expected: PASS for both

Run: `go test ./internal/messaging/ -v -run TestWrapAnsi`
Expected: PASS for all eleven existing `TestWrapAnsi*` tests, including `TestWrapAnsiCarriesOpenTagAcrossLineBreak`, `TestWrapAnsiMalformedTagFallback`, `TestWrapAnsiExplicitNewlineInsideOpenTagNoLeadingSpace` and the three in `wrap_panic_test.go`. The unbalanced-input fallback must still return non-empty text rather than panicking.

- [ ] **Step 5: Commit**

```bash
git add internal/messaging/wrap.go internal/messaging/wrap_test.go
git commit -F - <<'EOF'
fix(messaging): wrap with a tag stack, so nesting survives a line break

WrapAnsi tracked one open tag and cleared it on any close, so a
BALANCED inner span discarded the outer one. Reproduced: an outer
fg="214" around prose carrying an inner fg="item" wrapped at 40 left
lines two and three with no reopener, rendering uncolored.

This was never an edge case. applyCategoryColor puts an outer tag
around every non-default category, and narration routinely carries
inner mobname, username and item tags, so the bug fires on ordinary
combat text. motd.go is calling WrapAnsi today.

The scalar becomes a stack: a break inside N spans closes all N and
reopens all N, outermost first.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 3: Count runes, not bytes

`wrap.go:116-117` writes one byte and increments the width counter once per byte, so any multi-byte character counts double or worse. Task 1 made the oracle capable of seeing this.

**Files:**
- Modify: `internal/messaging/wrap.go` (the visible-character branch, and the import block)
- Test: `internal/messaging/wrap_test.go`

- [ ] **Step 1: Write the failing test**

```go
// TestWrapAnsiCountsRunesNotBytes pins the width unit. Reproduced against
// the old code: twenty two-byte runes plus "t ail" broke at visible column
// 21 against a requested width of 40, because each two-byte rune counted
// as two columns.
func TestWrapAnsiCountsRunesNotBytes(t *testing.T) {
	word := strings.Repeat("\u00e9", 20) // e-acute, 2 bytes each
	in := word + "t ail"                 // 21 visible columns, then a space, then 3 more

	got := WrapAnsi(in, 40)
	if strings.Contains(got, "\n") {
		t.Fatalf("25 visible columns must not wrap at width 40, got:\n%q", got)
	}

	// And a multi-byte character sitting exactly at the boundary.
	boundary := word + " " + word // 20 + 1 + 20 = 41 visible columns
	wrapped := WrapAnsi(boundary, 40)
	lines := splitLines(wrapped)
	if len(lines) != 2 {
		t.Fatalf("41 visible columns at width 40 must wrap into 2 lines, got %d:\n%q", len(lines), wrapped)
	}
	for i, line := range lines {
		if w := displayWidth(line); w > 40 {
			t.Errorf("line %d is %d visible columns, over the 40 requested: %q", i+1, w, line)
		}
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/messaging/ -run TestWrapAnsiCountsRunesNotBytes -v`

Expected: FAIL with `25 visible columns must not wrap at width 40`, showing a newline inserted after the twenty-first visible character.

- [ ] **Step 3: Decode runes**

Add `"unicode/utf8"` to the import block in `internal/messaging/wrap.go`:

```go
import (
	"regexp"
	"strings"
	"unicode/utf8"
)
```

Replace the visible-character branch at the foot of the main loop:

```go
		// Visible character. Advance by one RUNE: the counter is display
		// columns, and indexing the string byte-by-byte made every
		// multi-byte character count as two or more columns.
		_, size := utf8.DecodeRuneInString(text[i:])
		curWord.WriteString(text[i : i+size])
		curWordW++
		i += size
```

The whitespace and tag branches above it test `text[i]` against ASCII bytes and run before this point, so they are unaffected: a space and a newline are one byte each and no UTF-8 continuation byte can equal them.

- [ ] **Step 4: Run the test, then the package**

Run: `go test ./internal/messaging/ -run TestWrapAnsiCountsRunesNotBytes -v`
Expected: PASS

Run: `go test ./internal/messaging/`
Expected: `ok  github.com/GoMudEngine/GoMud/internal/messaging`

- [ ] **Step 5: Commit**

```bash
git add internal/messaging/wrap.go internal/messaging/wrap_test.go
git commit -F - <<'EOF'
fix(messaging): measure wrap width in runes, not bytes

WrapAnsi advanced one byte at a time and incremented the column
counter once per byte, so every multi-byte character counted double.
Reproduced: twenty two-byte runes plus "t ail" broke at visible column
21 against a requested width of 40.

Names, curly quotes and accented words all carry multi-byte runes, so
this would have folded ordinary narration short the moment the wrap
stage came on. The oracle was fixed first, in its own commit, so this
change has a judge capable of failing.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 4: shouldWrap becomes an explicit allowlist

**Files:**
- Modify: `internal/messaging/pipeline.go:73-99`

### The membership, and why

44 categories are admitted, 17 excluded. Every one of the 61 in the enum (fact 3) is accounted for.

**Admitted (44).** Each is free prose delivered one sentence at a time, with no column layout, box drawing or art.

| Group | Categories | Why it is safe to fold |
|---|---|---|
| Combat hits (6) | `HitMelee`, `HitBlunt`, `HitNaturalSharp`, `HitRanged`, `HitCaster`, `HitUnarmed` | One narration sentence per swing, authored in the combat message stores |
| Combat defence (3) | `Dodge`, `Parry`, `Block` | One sentence per defence, from `defense-messages` |
| Grapple (1) | `GrappleFlow` | Prose flow lines from `internal/grapplemessaging` |
| Outcome (4) | `Submission`, `Death`, `CombatSummary`, `CombatBlindWarning` | `CombatSummary` is a per-round prose tally (fact 18), not a table |
| Special moves (9) | `SurpriseAttack`, `Kick`, `Trip`, `Bash`, `Rally`, `Warcry`, `TauntSuccess`, `TauntResist`, `TauntFailure` | The M4e store's prose, one line per role per event |
| Spells (7) | `SpellFold`, `SpellDisruption`, `SpellElemental`, `SpellEnhancement`, `SpellMental`, `SpellVital`, `SpellManifestation` | Spell narration sentences |
| Ambient prose (4) | `RoomEntry`, `RoomExit`, `Weather`, `TimeOfDay` | Sentences about the world. `RoomDescription` is NOT here, see below |
| NPC prose (4) | `NPCDialogue`, `DialogueHint`, `MobIdle`, `MobEmote` | Authored sentences; none is pre-wrapped and none bypasses the pipeline |
| Other prose (5) | `Loot`, `Equipment`, `ConditionApply`, `ConditionExpire`, `Mutation` | `Equipment` carries decorated prose with `***` markers and nested tags, which is exactly the shape Task 2 repaired |
| Tips (1) | `Tip` | The motivating defect: 64 of 74 tips exceed 80 characters, longest 211 (fact 19) |

**Excluded (17), each for a stated reason.**

| Category | Why it must not wrap |
|---|---|
| `Default` | The unclassified bucket. Wrapping it would wrap anything nobody has categorized |
| `RoomDescription` | Side-by-side minimap columns. Already pinned by `TestRoomDescriptionSkipsWrap` (fact 10) |
| `Splash` | Rendered ASCII art (fact 11) |
| `SkillProgress` | A banner with its own formatting (fact 13) |
| `System` | Mixed bucket: refusals AND the score sheet, inventory, `who`, help, admin tables and the ASCII map (fact 14) |
| `Broadcast` | Mixed bucket: the hand-drawn MOTD box AND free channel chat (fact 15) |
| `Speech`, `Whisper`, `Shout`, `Emote` | Already wrapped at a hardcoded 80 (fact 16). Admitting them would wrap twice at two different widths |
| `Error`, `Warning` | System one-liners in Group A's territory, which M7 owns. Fail-safe: leave them alone |
| `Logout` | A departure notice, short by construction, and system-meta rather than narration |
| `GrappleHigh`, `Login`, `OOC`, `Toxin` | Zero production sends (fact 17). Policy on an unreachable category is unreachable policy |

- [ ] **Step 1: Write the failing test**

Add to `internal/messaging/pipeline_test.go`:

```go
// TestTipWrapsAtRecipientLineWidth is the motivating case: 64 of the 74
// shipped tips exceed 80 characters, longest 211, and a bare telnet client
// breaks them mid-word.
func TestTipWrapsAtRecipientLineWidth(t *testing.T) {
	long := "Healing up slow? Some rooms are sanctuaries, temples, certain camps, " +
		"the Sanctum Basin tutorial, and regenerate health, stamina, and conviction " +
		"much faster than ordinary rooms. Look for a peaceful description."

	got := RenderForRecipient(RenderInput{
		Category:  CategoryTip,
		Text:      long,
		Channel:   ChannelAudio,
		LineWidth: 60,
	})

	lines := splitLines(got)
	if len(lines) < 2 {
		t.Fatalf("a 211-character tip must fold at width 60, got one line:\n%q", got)
	}
	for i, line := range lines {
		if w := displayWidth(line); w > 60 {
			t.Errorf("line %d is %d visible columns, over the 60 requested: %q", i+1, w, line)
		}
	}
}

// TestSystemNeverWraps guards the mixed bucket. CategorySystem carries the
// score sheet, inventory, who, help, the admin DynamicList tables and the
// ASCII map alongside one-line refusals, so folding it would shatter every
// one of those layouts.
func TestSystemNeverWraps(t *testing.T) {
	table := "Item              Qty   Value\n" +
		"---------------   ---   -----\n" +
		"healing draught     3     45g"

	got := RenderForRecipient(RenderInput{
		Category:  CategorySystem,
		Text:      table,
		Channel:   ChannelAudio,
		LineWidth: 20,
	})

	if got != table {
		t.Fatalf("CategorySystem must pass through unwrapped at any width.\n got: %q\nwant: %q", got, table)
	}
}
```

- [ ] **Step 2: Run them and watch the first fail**

Run: `go test ./internal/messaging/ -run "TestTipWrapsAtRecipientLineWidth|TestSystemNeverWraps" -v`

Expected: `TestTipWrapsAtRecipientLineWidth` FAILs with `a 211-character tip must fold at width 60, got one line`. `TestSystemNeverWraps` PASSes already, because nothing wraps today; it is the guard that must keep passing after step 3.

- [ ] **Step 3: Write the allowlist**

Replace `shouldWrap` and the stage-6 comment in `internal/messaging/pipeline.go`. The stage call itself (`if shouldWrap(in.Category) { text = wrap(text, in.LineWidth) }`) does not change.

```go
// shouldWrap controls whether the pipeline's wrap stage fires, by
// Category. It is an explicit ALLOWLIST and the default is false, so a
// category nobody has classified keeps today's behavior. That is the
// fail-safe direction: a narration category missing from this list ships
// slightly ugly, where a table category wrongly added to it ships
// mangled.
//
// Shape follows skipStages in normalize.go, this package's established
// idiom for per-Category policy.
//
// DELIBERATELY ABSENT, and each one has a reason:
//
//   - CategorySystem and CategoryBroadcast are MIXED BUCKETS. System
//     carries one-line refusals alongside the score sheet, inventory,
//     who, help, the admin DynamicList tables and the ASCII map.
//     Broadcast carries free channel chat alongside the hand-drawn MOTD
//     box. Neither can be folded without shattering the other half.
//   - CategoryRoomDescription renders a side-by-side block, prose left
//     and minimap right. TestRoomDescriptionSkipsWrap pins it.
//   - CategorySplash is rendered ASCII art.
//   - CategorySkillProgress is a banner with its own formatting, and
//     skipStages already excludes it for the same reason.
//   - Speech, Whisper, Shout and Emote are ALREADY wrapped, at a
//     hardcoded 80 that ignores the reader's LineWidth (say.go,
//     shout.go, reply.go, whisper.go). Adding them here would wrap the
//     same text twice at two different widths.
//   - Error and Warning are Group A system output, which M7 owns.
//   - GrappleHigh, Login, OOC and Toxin have zero production senders, so
//     any policy here would be unreachable.
func shouldWrap(cat Category) bool {
	switch cat {
	// Combat: hits, defences, grapple flow, outcomes.
	case CategoryHitMelee, CategoryHitBlunt, CategoryHitNaturalSharp,
		CategoryHitRanged, CategoryHitCaster, CategoryHitUnarmed,
		CategoryDodge, CategoryParry, CategoryBlock,
		CategoryGrappleFlow,
		CategorySubmission, CategoryDeath,
		CategoryCombatSummary, CategoryCombatBlindWarning:
		return true

	// Special moves.
	case CategorySurpriseAttack, CategoryKick, CategoryTrip, CategoryBash,
		CategoryRally, CategoryWarcry,
		CategoryTauntSuccess, CategoryTauntResist, CategoryTauntFailure:
		return true

	// Spells.
	case CategorySpellFold, CategorySpellDisruption, CategorySpellElemental,
		CategorySpellEnhancement, CategorySpellMental, CategorySpellVital,
		CategorySpellManifestation:
		return true

	// Authored NPC and ambient prose. RoomDescription and Splash are
	// excluded above and are not in this list.
	case CategoryNPCDialogue, CategoryDialogueHint,
		CategoryMobIdle, CategoryMobEmote,
		CategoryRoomEntry, CategoryRoomExit,
		CategoryWeather, CategoryTimeOfDay:
		return true

	// Other narration, plus the tips broadcast this decision was made for.
	case CategoryLoot, CategoryEquipment,
		CategoryConditionApply, CategoryConditionExpire,
		CategoryMutation, CategoryTip:
		return true
	}
	return false
}
```

Also update the stage-6 comment above the call (`pipeline.go:73-81`), which currently says wrap is "disabled by default":

```go
	// Stage 6: wrap, for the narration categories shouldWrap admits.
	// Pre-formatted output (tables, banners, ASCII art, side-by-side
	// templates) is excluded by category; see shouldWrap for the full
	// list and the reason for each exclusion. WrapAnsi also remains
	// callable directly by sites that wrap themselves, such as motd.go's
	// box-bordered banner.
```

- [ ] **Step 4: Run both tests, then the package**

Run: `go test ./internal/messaging/ -run "TestTipWrapsAtRecipientLineWidth|TestSystemNeverWraps" -v`
Expected: PASS for both

Run: `go test ./internal/messaging/`
Expected: FAIL. `TestShouldWrapDisabledByDefault` now reports errors for all 44 admitted categories, beginning `category "HitMelee" must not wrap by default`. That is correct and Task 5 replaces it. Do not commit until Task 5 lands.

- [ ] **Step 5: Hold the commit**

This task and Task 5 land as one commit, because the tree does not pass between them. Continue straight to Task 5.

---

## Task 5: Replace the pin, do not delete it

`TestShouldWrapDisabledByDefault` cannot survive Task 4. Deleting it would leave the allowlist unguarded, so it is replaced by a test that pins exact membership: adding a category then becomes a deliberate edit with a visible diff in both the source and the test.

**Files:**
- Modify: `internal/messaging/pipeline_test.go:49-57`

- [ ] **Step 1: Replace the test**

Delete `TestShouldWrapDisabledByDefault` and put this in its place:

```go
// wrapAllowlist is the exact set of categories shouldWrap admits. It is
// duplicated here ON PURPOSE: the point of the pin is that changing
// shouldWrap without changing this list fails the build, so a category
// joins the wrap policy only by a deliberate edit in two places.
var wrapAllowlist = map[Category]bool{
	CategoryHitMelee: true, CategoryHitBlunt: true, CategoryHitNaturalSharp: true,
	CategoryHitRanged: true, CategoryHitCaster: true, CategoryHitUnarmed: true,
	CategoryDodge: true, CategoryParry: true, CategoryBlock: true,
	CategoryGrappleFlow: true,
	CategorySubmission: true, CategoryDeath: true,
	CategoryCombatSummary: true, CategoryCombatBlindWarning: true,

	CategorySurpriseAttack: true, CategoryKick: true, CategoryTrip: true,
	CategoryBash: true, CategoryRally: true, CategoryWarcry: true,
	CategoryTauntSuccess: true, CategoryTauntResist: true, CategoryTauntFailure: true,

	CategorySpellFold: true, CategorySpellDisruption: true, CategorySpellElemental: true,
	CategorySpellEnhancement: true, CategorySpellMental: true, CategorySpellVital: true,
	CategorySpellManifestation: true,

	CategoryNPCDialogue: true, CategoryDialogueHint: true,
	CategoryMobIdle: true, CategoryMobEmote: true,
	CategoryRoomEntry: true, CategoryRoomExit: true,
	CategoryWeather: true, CategoryTimeOfDay: true,

	CategoryLoot: true, CategoryEquipment: true,
	CategoryConditionApply: true, CategoryConditionExpire: true,
	CategoryMutation: true, CategoryTip: true,
}

// TestShouldWrapMatchesPinnedAllowlist pins wrap policy for all 61
// categories. It replaced TestShouldWrapDisabledByDefault, which asserted
// that nothing wraps: true until M5 PR 2, and no longer.
func TestShouldWrapMatchesPinnedAllowlist(t *testing.T) {
	admitted := 0
	for c := CategoryDefault; c < categoryMax; c++ {
		want := wrapAllowlist[c]
		if want {
			admitted++
		}
		if got := shouldWrap(c); got != want {
			t.Errorf("shouldWrap(%q) = %v, pinned %v: update both shouldWrap and wrapAllowlist, deliberately", c, got, want)
		}
	}
	if admitted != 44 {
		t.Errorf("pinned allowlist holds %d categories, expected 44: the pin was edited without updating its own count", admitted)
	}
}

// TestMixedAndPreformattedCategoriesNeverWrap is the standing rule, stated
// independently of the membership pin above so that a careless edit to
// wrapAllowlist cannot quietly admit one of these. Each carries
// pre-formatted output, wraps itself already, or has no production sender.
func TestMixedAndPreformattedCategoriesNeverWrap(t *testing.T) {
	never := map[Category]string{
		CategorySystem:          "mixed bucket: refusals plus score sheet, inventory, who, help, admin tables, ASCII map",
		CategoryBroadcast:       "mixed bucket: channel chat plus the hand-drawn MOTD box",
		CategoryRoomDescription: "side-by-side prose and minimap columns",
		CategorySplash:          "rendered ASCII art",
		CategorySkillProgress:   "banner with its own formatting",
		CategorySpeech:          "already wrapped at a hardcoded 80 in say.go",
		CategoryWhisper:         "already wrapped at a hardcoded 80 in whisper.go and reply.go",
		CategoryShout:           "already wrapped at a hardcoded 80 in shout.go",
		CategoryEmote:           "self-echo path wraps itself",
		CategoryDefault:         "the unclassified bucket",
	}
	for cat, why := range never {
		if shouldWrap(cat) {
			t.Errorf("shouldWrap(%q) must stay false: %s", cat, why)
		}
		if wrapAllowlist[cat] {
			t.Errorf("%q must never appear in wrapAllowlist: %s", cat, why)
		}
	}
}
```

- [ ] **Step 2: Run the package**

Run: `go test ./internal/messaging/ -v -run "TestShouldWrap|TestMixedAndPreformatted|TestTipWraps|TestSystemNeverWraps"`
Expected: PASS for all four. No test named `TestShouldWrapDisabledByDefault` remains.

Run: `go test ./internal/messaging/`
Expected: `ok  github.com/GoMudEngine/GoMud/internal/messaging`

- [ ] **Step 3: Prove the pin can fail**

Temporarily add `CategorySystem` to the `switch` in `shouldWrap`.

Run: `go test ./internal/messaging/ -run "TestShouldWrapMatchesPinnedAllowlist|TestMixedAndPreformattedCategoriesNeverWrap|TestSystemNeverWraps" -v`

Expected: all three FAIL. `TestShouldWrapMatchesPinnedAllowlist` reports `shouldWrap("System") = true, pinned false`, `TestMixedAndPreformattedCategoriesNeverWrap` reports the mixed-bucket reason, and `TestSystemNeverWraps` reports the mangled table. Revert the temporary edit and confirm green again.

- [ ] **Step 4: Commit Tasks 4 and 5 together**

```bash
git add internal/messaging/pipeline.go internal/messaging/pipeline_test.go
git commit -F - <<'EOF'
feat(messaging): wrap narration by category, leave pre-formatted output alone

shouldWrap returned false for every category, so the pipeline's wrap
stage had never fired and LineWidth was plumbed in and ignored. It
becomes an explicit allowlist of 44 narration categories, on the
skipStages idiom this package already uses for per-Category policy.

The default stays false, which is the fail-safe direction: a narration
category missing from the list ships slightly ugly, where a table
category wrongly added ships mangled.

Seventeen categories stay out, each for a recorded reason. System and
Broadcast are mixed buckets carrying tables, the ASCII map and the MOTD
box alongside refusals and chat. RoomDescription renders side-by-side
minimap columns and Splash renders ASCII art. SkillProgress is a
banner. Speech, Whisper, Shout and Emote already wrap themselves at a
hardcoded 80, so admitting them would wrap twice at two widths.
GrappleHigh, Login, OOC and Toxin have no production sender at all.

TestShouldWrapDisabledByDefault asserted that nothing wraps, which this
commit makes false. It is replaced rather than deleted: the new test
pins exact membership so a category joins wrap policy only by a
deliberate edit in two places, and a second test states the
never-wrap rule independently. Proven capable of failing by admitting
CategorySystem and watching all three guards go red.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 6: The pre-formatted passthrough golden

Task 5's guards assert policy. This asserts the consequence: real pre-formatted payloads survive the pipeline byte-identically at a hostile width.

🪤 The fixtures are representative shapes, not live renders. `internal/messaging` must not import `internal/templates` or `internal/usercommands`, both of which sit above it in the import graph, so a test here cannot call `templates.Process` or `motd.go`.

**That limitation is closed by Task 7, not accepted.** This task is the fast in-package feedback loop and pins the pipeline's behavior on a known shape; Task 7 renders real production templates from the root package and pins what a player actually receives. Both are kept, because they protect different things. Do not treat this task as sufficient on its own: a stand-in fixture cannot fail on the real defect, which is the trap that the whole `CategorySystem` exclusion rests on.

**Files:**
- Create: `internal/messaging/wrap_preformatted_golden_test.go`

- [ ] **Step 1: Write the golden**

```go
package messaging

import "testing"

// preformattedFixtures are the output shapes that must never be folded:
// a column table (the score sheet, inventory, who, help and the admin
// DynamicList all produce this shape), a box-drawn banner (motd.go), and
// a side-by-side block (the look command's minimap column).
//
// They are SHAPES, not live renders. internal/messaging sits below
// internal/templates and internal/usercommands in the import graph, so a
// test in this package cannot call templates.Process or motd.go. The
// invariant under test is that text of this shape, under these
// categories, comes back byte-identical at any width.
var preformattedFixtures = []struct {
	name string
	cat  Category
	text string
}{
	{
		name: "column table under System",
		cat:  CategorySystem,
		text: "Item              Qty   Value\n" +
			"---------------   ---   -----\n" +
			"healing draught     3     45g\n" +
			"iron sword          1    120g",
	},
	{
		name: "box banner under Broadcast",
		cat:  CategoryBroadcast,
		text: "╔══════════════════════════════════════╗\n" +
			"║  Welcome to the realm, adventurer.   ║\n" +
			"║  Type help to begin.                 ║\n" +
			"╚══════════════════════════════════════╝",
	},
	{
		name: "side-by-side block under RoomDescription",
		cat:  CategoryRoomDescription,
		text: "the cobblestone road winds west          ║·····║\n" +
			"past a shuttered forge                   ║··@··║",
	},
	{
		name: "ascii art under Splash",
		cat:  CategorySplash,
		text: "   /\\_/\\\n" +
			"  ( o.o )   THE SANCTUM BASIN\n" +
			"   > ^ <",
	},
	{
		name: "progress banner under SkillProgress",
		cat:  CategorySkillProgress,
		text: "[==========          ] Swordsmanship",
	},
}

// TestPreformattedCategoriesPassThroughUnwrapped renders each fixture at a
// width far narrower than its content and fails if a single byte changed.
// This is what proves the Task 4 allowlist did not mangle anything.
func TestPreformattedCategoriesPassThroughUnwrapped(t *testing.T) {
	for _, f := range preformattedFixtures {
		t.Run(f.name, func(t *testing.T) {
			got := RenderForRecipient(RenderInput{
				Category:  f.cat,
				Text:      f.text,
				Channel:   ChannelAudio,
				LineWidth: 20,
			})
			if got != f.text {
				t.Errorf("%q was altered by the pipeline at LineWidth 20.\n got: %q\nwant: %q", f.name, got, f.text)
			}
		})
	}
}
```

🪤 `ChannelAudio` is used so the sight gate does not drop the text; this test is about wrap, not visibility.

- [ ] **Step 2: Run it**

Run: `go test ./internal/messaging/ -run TestPreformattedCategoriesPassThroughUnwrapped -v`
Expected: PASS, five subtests

- [ ] **Step 3: Prove it can fail**

Temporarily add `CategorySystem` to `shouldWrap`'s switch.

Run: `go test ./internal/messaging/ -run TestPreformattedCategoriesPassThroughUnwrapped -v`
Expected: FAIL on `column_table_under_System` with the table folded at 20 columns. Revert the edit and confirm green.

Repeat once with `CategoryBroadcast` to prove the banner fixture is live too. Expected: FAIL on `box_banner_under_Broadcast`. Revert.

- [ ] **Step 4: Commit**

```bash
git add internal/messaging/wrap_preformatted_golden_test.go
git commit -F - <<'EOF'
test(messaging): pin that pre-formatted output survives the wrap stage

Task 4's guards assert policy. This asserts the consequence: a column
table, a box banner, a side-by-side minimap block, ASCII art and a
progress banner all come back byte-identical from the pipeline at a
LineWidth of 20.

The fixtures are shapes rather than live renders, because
internal/messaging sits below internal/templates and
internal/usercommands in the import graph and cannot call them. The
invariant that matters is still covered.

Proven capable of failing by admitting CategorySystem and then
CategoryBroadcast to the allowlist and watching the matching subtest
go red each time.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 7: The LIVE-RENDER passthrough golden, in the root package

Task 6 pins the pipeline's behavior on a known shape. It cannot fail on the thing the exclusion of `CategorySystem` is actually justified by, because it never renders a real table. That is precisely the failure mode M4a hit three separate times and that the arc spec names outright: **a golden covers the store, not the path production calls.** A proxy fixture is a check that cannot fail on the real defect.

The import-graph objection is real for `internal/messaging` and does NOT apply to the repo root. `package main` at the root already hosts cross-cutting guards that import freely across `internal/` (`send_trio_only_guard_test.go`, `shipped_narration_data_guard_test.go`, `messaging_surface_guard_test.go`, `boot_smoke_test.go`), so a root test can import both `internal/templates` and `internal/messaging`.

**Files:**
- Create: `wrap_live_render_golden_test.go` (repo root, `package main`)
- Create: `testdata/wrap_live_render.golden` (the root `testdata/` directory already exists and already holds `m2-routing.golden`, so this follows the established root convention)

### 🔴 Three things probed before this task was written

Written from live runs on 2026-09-22, not from reasoning. An executor who skips these will design the wrong assertion.

1. **A live `character/status` render works from the root package with a minimal fixture.** `users.NewUserRecord(1, 0)` plus a name is enough; the sheet comes back at 3485 bytes over 18 lines with real box drawing.
2. 🔴 **Byte-identity against the raw render is the WRONG assertion and will fail on day one.** `RenderForRecipient` already alters the text today, with wrap off, for two independent reasons: stage 5 `applyCategoryColor` wraps the whole payload in `<ansi fg="system">...</ansi>`, and stage 2 `normalize` mutates it, because `skipStages(CategorySystem)` returns 0 and therefore runs all five normalization stages on table output.
3. 🔴 **A newline-count assertion is NOT sufficient either.** `WrapAnsi` destroys column layout WITHOUT adding a single newline: a real `templates.DynamicList` table went from 494 bytes to 436 under `WrapAnsi(out, 55)` with the newline count unchanged at 1, because `flushWord` collapses a run of padding spaces to one space. Column alignment dies silently. The status sheet does fold (18 newlines to 25), but any assertion resting on newline count alone would have passed while an inventory listing was ruined.

So the assertion is a **checked-in golden of the pipeline's actual output**. It catches folding, space collapsing, and any future stage that starts touching this text.

📌 **Observation, filed not fixed:** finding 2 means normalization appends sentence punctuation to a status sheet today. The last row renders as `auto-tap-below 15.` where the template authored `auto-tap-below 15`. That is a live cosmetic defect in table output, it is a `skipStages` question rather than a wrap question, and it is out of PR 2's scope. This golden will lock in the current behavior including the stray period; fixing it is a separate change that re-records the golden deliberately. Record it as an M5 follow-up.

- [ ] **Step 1: Write the golden test**

Create `wrap_live_render_golden_test.go` at the repo root:

```go
package main

import (
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

var updateLiveRenderGolden = flag.Bool("update-live-render", false,
	"re-record testdata/wrap_live_render.golden")

// liveRenderWorld is the world players actually read. This test must not
// use _datafiles/world/default: that tree is vestigial, does not boot, and
// its templates are not what anybody receives.
const liveRenderWorld = `_datafiles/world/dogmud`

// liveRenderWidth is deliberately narrow. Every fixture below is wider than
// this, so if a pre-formatted category ever joined the wrap allowlist the
// damage would be unmistakable in the diff.
const liveRenderWidth = 55

// TestLiveRenderedOutputSurvivesThePipeline renders REAL production
// templates and pushes them through the REAL pipeline, then compares the
// result against a checked-in golden.
//
// WHY THIS EXISTS SEPARATELY FROM the in-package fixture test
// (internal/messaging/wrap_preformatted_golden_test.go). That one pins the
// pipeline's behavior on a representative shape and is the fast feedback
// loop. It cannot render a real template, because internal/messaging sits
// below internal/templates in the import graph. The entire justification for
// keeping CategorySystem out of the wrap allowlist is that REAL tables would
// be mangled, so testing a stand-in shape instead of a real table is the
// "a golden covers the store, not the path production calls" trap this arc
// has already been bitten by three times. The root package can import both,
// so the real path gets covered here.
//
// WHAT IS ASSERTED, and why it is not byte-identity against the raw render:
// RenderForRecipient legitimately alters this text today even with wrap off.
// Stage 5 wraps the payload in the category colour tag, and stage 2
// normalizes it, because skipStages(CategorySystem) returns 0. So the golden
// records the pipeline's OUTPUT, not the template's output. Anything that
// changes what a player receives, including a category joining the wrap
// allowlist, moves this file.
//
// 🪤 A newline count would NOT have been enough. WrapAnsi collapses runs of
// padding spaces, so it destroys column alignment without adding a line:
// measured, a real DynamicList table lost 58 bytes at width 55 with its
// newline count unchanged.
func TestLiveRenderedOutputSurvivesThePipeline(t *testing.T) {
	configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles": liveRenderWorld,
	})
	dirFS, ok := os.DirFS(liveRenderWorld).(fs.ReadFileFS)
	if !ok {
		t.Fatalf("os.DirFS(%q) is not a ReadFileFS", liveRenderWorld)
	}
	templates.RegisterFS(dirFS)

	var got strings.Builder
	for _, f := range liveRenderFixtures(t) {
		rendered := messaging.RenderForRecipient(messaging.RenderInput{
			Category:  f.cat,
			Text:      f.text,
			Channel:   messaging.ChannelAudio,
			LineWidth: liveRenderWidth,
		})
		got.WriteString("=== " + f.name + " (" + f.cat.String() + ") ===\n")
		got.WriteString(rendered)
		got.WriteString("\n")
	}

	goldenPath := filepath.Join("testdata", "wrap_live_render.golden")

	if *updateLiveRenderGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got.String()), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("re-recorded %s (%d bytes)", goldenPath, got.Len())
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v (record it with -update-live-render)", err)
	}
	if got.String() != string(want) {
		t.Errorf("live-rendered output changed.\n"+
			"If a pre-formatted category joined the wrap allowlist, REVERT that.\n"+
			"If the change is deliberate, re-record with:\n"+
			"  go test . -run TestLiveRenderedOutputSurvivesThePipeline -update-live-render\n"+
			"%s", firstDiff(string(want), got.String()))
	}
}

type liveRenderFixture struct {
	name string
	cat  messaging.Category
	text string
}

// liveRenderFixtures builds the three production shapes that must never be
// folded: the character sheet, a column listing, and the MOTD box.
func liveRenderFixtures(t *testing.T) []liveRenderFixture {
	t.Helper()

	u := users.NewUserRecord(1, 0)
	u.Username = "Goldwright"
	u.Character.Name = "Goldwright"

	status, err := templates.Process("character/status", u, u.UserId)
	if err != nil {
		t.Fatalf("render character/status: %v", err)
	}
	if !strings.Contains(status, "\n") {
		t.Fatalf("character/status rendered a single line (%d bytes): the fixture is not exercising the real sheet", len(status))
	}

	// templates.DynamicList is the real column packer behind the admin item
	// and mob listings (admin.item.go:109, admin.mob.go:170). Measured: at
	// width 55 WrapAnsi eats its padding without adding a newline.
	listing := templates.DynamicList([]templates.NameDescription{
		{Id: 1, Name: "healing draught", Description: "restores health"},
		{Id: 2, Name: "iron sword", Description: "a plain blade"},
		{Id: 3, Name: "dustwalk herb", Description: "a dry sprig"},
		{Id: 4, Name: "moonpetal", Description: "pale and cold"},
		{Id: 5, Name: "bottle", Description: "empty glass"},
		{Id: 6, Name: "cats eye draught", Description: "see in the dark"},
	}, 26, 80, 3, 18)

	return []liveRenderFixture{
		{name: "character/status", cat: messaging.CategorySystem, text: status},
		{name: "templates.DynamicList", cat: messaging.CategorySystem, text: listing},
		{name: "motd box", cat: messaging.CategoryBroadcast, text: motdBoxLikeProduction()},
	}
}

// motdBoxLikeProduction mirrors internal/usercommands/motd.go's real
// construction rather than inventing a stand-in box: the same wrap width
// (padWidth - 1, i.e. LineWidth - 5), the same frame arithmetic, the same
// border glyphs and the same per-line padding. Calling Motd() itself from
// here would need a live connection and a user session, which this test has
// no reason to build.
func motdBoxLikeProduction() string {
	const lw = 72 // a plausible configured line width
	innerWidth := lw - 2
	padWidth := lw - 4
	wrapWidth := padWidth - 1

	body := "Welcome to the realm. Type help to begin, and read the tips as they " +
		"arrive. The wilds past the gate are not forgiving."
	wrapped := strings.Split(messaging.WrapAnsi(body, wrapWidth), "\n")

	horiz := strings.Repeat("═", innerWidth)
	title := `.:  M E S S A G E   O F   T H E   D A Y`
	titleInner := " " + title
	titlePadded := titleInner + strings.Repeat(" ", padWidth-len(titleInner))
	titleStyled := strings.Replace(titlePadded, title,
		`<ansi fg="cyan-bold">`+title+`</ansi>`, 1)

	var out string
	out += `<ansi fg="yellow"> ╔` + horiz + `╗` + "\n"
	out += ` ║ ` + titleStyled + `║` + "\n"
	out += ` ╠` + horiz + `╣` + "\n"
	for _, line := range wrapped {
		padLen := padWidth - len([]rune(line))
		if padLen < 0 {
			padLen = 0
		}
		out += ` ║ ` + line + strings.Repeat(" ", padLen) + `║` + "\n"
	}
	out += ` ╚` + horiz + `╝</ansi>` + "\n"
	return out
}

// firstDiff reports the first rune position where two strings part company,
// with surrounding context, because a raw dump of two 3.5KB sheets is
// unreadable in a test log.
func firstDiff(want, got string) string {
	w, g := []rune(want), []rune(got)
	for i := 0; i < len(w) && i < len(g); i++ {
		if w[i] != g[i] {
			lo := i - 60
			if lo < 0 {
				lo = 0
			}
			hiW, hiG := i+60, i+60
			if hiW > len(w) {
				hiW = len(w)
			}
			if hiG > len(g) {
				hiG = len(g)
			}
			return "first difference at rune " + itoa(i) + ":\n  want: " +
				string(w[lo:hiW]) + "\n  got:  " + string(g[lo:hiG])
		}
	}
	return "one output is a prefix of the other: want " + itoa(len(w)) +
		" runes, got " + itoa(len(g)) + " runes"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
```

- [ ] **Step 2: Record the golden and read it**

Run: `go test . -run TestLiveRenderedOutputSurvivesThePipeline -update-live-render -v`
Expected: PASS, with a log line `re-recorded testdata/wrap_live_render.golden (N bytes)` where N is roughly 4500 to 5000.

Run: `head -20 testdata/wrap_live_render.golden`
Expected: the `=== character/status (System) ===` header followed by the real sheet, with `┌─` box drawing and `<ansi fg="system">` wrapping it. **Read it.** If it does not look like the status sheet a player sees, the fixture is wrong and nothing downstream is trustworthy.

- [ ] **Step 3: Run it as a check**

Run: `go test . -run TestLiveRenderedOutputSurvivesThePipeline -v`
Expected: PASS

- [ ] **Step 4: Prove it can fail, on the real defect**

Temporarily add `CategorySystem` to the `switch` in `shouldWrap` (`internal/messaging/pipeline.go`).

Run: `go test . -run TestLiveRenderedOutputSurvivesThePipeline -v`

Expected: FAIL, with `live-rendered output changed` and a `first difference at rune N` block showing the status sheet's columns folded and its padding collapsed. This is the assertion Task 6 could not make.

Revert the edit.

Run: `go test . -run TestLiveRenderedOutputSurvivesThePipeline -v`
Expected: PASS

Now repeat once for the banner: temporarily add `CategoryBroadcast` to the allowlist, run, confirm FAIL pointing into the `motd box (Broadcast)` section, revert, confirm PASS.

- [ ] **Step 5: Commit**

```bash
git add wrap_live_render_golden_test.go testdata/wrap_live_render.golden
git commit -F - <<'EOF'
test(root): pin REAL rendered tables and the MOTD box through the pipeline

The in-package fixture test pins pipeline behavior on a representative
shape. It cannot render a real template, because internal/messaging
sits below internal/templates in the import graph. But the entire
justification for keeping CategorySystem out of the wrap allowlist is
that REAL tables would be mangled, so a stand-in shape is the exact
"a golden covers the store, not the path production calls" trap that
bit M4a three times. The root package imports freely across internal/,
so the real path gets covered here.

Three things this had to be designed around, all measured rather than
reasoned:

- Byte-identity against the raw render fails on day one. The pipeline
  already alters this text with wrap off: stage 5 wraps it in the
  category colour tag, and stage 2 normalizes it, because
  skipStages(CategorySystem) returns 0 and runs all five stages on
  table output.
- A newline count is not enough either. WrapAnsi collapses runs of
  padding spaces, so it destroys column alignment without adding a
  line: a real DynamicList table lost 58 bytes at width 55 with its
  newline count unchanged.
- So the golden records the pipeline's OUTPUT, which catches folding,
  space collapsing and any future stage that starts touching it.

The MOTD fixture mirrors motd.go's real frame arithmetic and wrap
width rather than inventing a box.

Proven capable of failing by admitting CategorySystem and then
CategoryBroadcast and watching the matching section diff each time.

Filed, not fixed: normalization appends sentence punctuation to the
status sheet's last row, rendering "auto-tap-below 15." where the
template authored "auto-tap-below 15". That is a skipStages question,
not a wrap one. This golden locks in the current behavior including
the stray period.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Task 8: Correct internal/messaging/context.md

The file documents wrap as an unconditionally live pipeline stage, at `:21-22` in its numbered list and again at `:259` and `:264` in its file table. That was wrong before this PR (the stage never fired) and is wrong after it in a different way (it fires for 44 of 61 categories). It lands here, with the behavior change, so the document and the code change on the same day.

🪤 `tools/context_md_audit.py` cannot catch this. It checks that `func`/`type` names inside fenced Go blocks exist in the package, and every symbol this prose names (`WrapAnsi`, `LineWidth`, `RenderInput`) is real. The rot is behavioral, so the correction is manual and the audit staying green proves nothing about it.

**Files:**
- Modify: `internal/messaging/context.md`

- [ ] **Step 1: Read the current claims**

Run: `sed -n '1,30p' internal/messaging/context.md` and `sed -n '255,270p' internal/messaging/context.md`

Note the numbered stage list, the file table rows for `pipeline.go` and `wrap.go`, and the disagreement between the numbered list (7 stages, including the sight gate) and the file table (6 stages, sight gate omitted).

- [ ] **Step 2: Rewrite stage 6 in the numbered list**

Replace the stage 6 bullet (currently around `:21-22`) with:

```markdown
6. **Wrap** at the recipient's `UserRecord.LineWidth`, ANSI-aware, for the
   narration categories `shouldWrap` admits (44 of 61). Pre-formatted output
   is excluded by category: `System` and `Broadcast` are mixed buckets
   carrying tables, the ASCII map and the MOTD box alongside refusals and
   chat; `RoomDescription` renders side-by-side minimap columns; `Splash`
   renders ASCII art; `SkillProgress` is a banner; the speech categories
   already wrap themselves at a hardcoded 80. See `shouldWrap` in
   `pipeline.go` for the full list and the reason attached to each
   exclusion. `WrapAnsi` closes and reopens the whole stack of open ansi
   tags across a break, and measures width in runes.
```

- [ ] **Step 3: Fix the file table rows**

The `pipeline.go` row's stage ordering must list all seven stages including the sight gate, matching the numbered list:

```markdown
| `pipeline.go` | Stage ordering: compose, normalize, sight gate, anonymize, color, wrap, deliver. `shouldWrap` holds the per-category wrap allowlist |
```

The `wrap.go` row must stop implying a fixed 80 and stop implying unconditional application:

```markdown
| `wrap.go` | `WrapAnsi`: ANSI-aware folding at a caller-supplied width, measured in visible runes. Called by the pipeline for allowlisted categories, and directly by `motd.go` for its box-bordered banner |
```

- [ ] **Step 4: Verify**

Run: `python tools/context_md_audit.py internal/messaging`
Expected: `packages checked: 1, packages with phantom symbols: 0, total phantom symbols: 0, All documented symbols resolve.`

This was already green before the edit and proves only that no invented symbol was introduced. The behavioral correction is verified by reading the file against `shouldWrap`.

Run: `grep -n "wrap" internal/messaging/context.md`
Expected: no remaining claim that the wrap stage applies to every message.

- [ ] **Step 5: Commit**

```bash
git add internal/messaging/context.md
git commit -F - <<'EOF'
docs(messaging): context.md describes the wrap stage that actually runs

The file documented wrap as an unconditionally live stage. That was
wrong before this PR, because shouldWrap returned false for everything
and the stage never fired, and it is wrong after it in a different way,
because the stage now fires for 44 of 61 categories.

Corrected alongside the behavior change so the document and the code
agree on the same day. The file table's stage list also omitted the
sight gate and disagreed with the file's own numbered list about
whether the pipeline has six stages or seven; both now say seven.

tools/context_md_audit.py cannot catch this class of rot: every symbol
the prose names is real, so the audit was green throughout. The
correction is manual by necessity.

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>
EOF
```

---

## Proof obligations

Before opening the PR:

- [ ] **Full suite green.** `go test ./...` across all 121 packages.
- [ ] **Root gate green.** `go test .` for the repo-root guard tests, which include the messaging surface guard and the viewpoint audit.
- [ ] **Task 5's sabotage done and reverted.** `CategorySystem` admitted, all three guards seen red, edit reverted, green again. Record it in the PR body.
- [ ] **Task 6's sabotage done and reverted**, once for `CategorySystem` and once for `CategoryBroadcast`.
- [ ] **Task 7's sabotage done and reverted**, the same two categories, against the LIVE golden. This is the one that fails on a real mangled status sheet, so it is the sabotage that actually matters. Record the `first difference at rune N` output in the PR body.
- [ ] **The live golden was read, not just recorded.** `head -20 testdata/wrap_live_render.golden` must look like the sheet a player sees. A golden recorded from a broken fixture pins the breakage.
- [ ] **`gofmt` check.** 🪤 `gofmt -l` false-positives on Windows: it reads the working copy, and `core.autocrlf` can leave a file CRLF on disk while the committed blob is LF. Confirm with `git show HEAD:<file> | head -c 16 | od -c` before chasing a hit.
- [ ] **Boot check** in an isolated detached worktree. 🪤 A failed boot EXITS 0: `main()` recovers, logs `PANIC` and a stack, and returns normally. Grep the log for `Server Ready` and for a real panic line; never check `$?`. 🪤 Two `PANIC` hits are the documented false positive: `GamePlay.MapConsistencyEnforce` has the literal value `panic`.

### PR 2 carries its own playtest

This is the arc's first player-visible change to the SHAPE of a line, not its wording. The guards prove tables are untouched; nothing automated can prove that folded prose reads well. So a light playtest is required, not the full adversarial gate PR 3 owns:

- [ ] `set linewidth 60`, then read a tip through to the end of a long one. Confirm it folds at 60 and breaks between words, never mid-word.
- [ ] Fight something for several rounds. Confirm combat prose folds, that color survives every continuation line (this is what Task 2 fixed), and that the round tally still reads as one line per event.
- [ ] `status`, `inventory`, `who`, and the map. Confirm every column still lines up at linewidth 60.
- [ ] Reconnect to see the MOTD. Confirm the box is intact.
- [ ] `say` something long. Confirm it wraps once, at 80, not twice.
- [ ] `set linewidth 120` and repeat the first two. Confirm the fold follows the setting.
