# Banner Context

## Purpose

`internal/banner` formats the SKILL ADVANCEMENT / STATISTIC INCREASED block
the progression system shows when a skill or stat goes up: a rule line, a
centred title, the centred skill or stat name, an optional tier transition,
and a closing rule. It is a leaf package (imports only `strings` and
`unicode/utf8`) so `internal/characters` can use it without an import cycle.

## API

```go
type Kind int // Skill, Stat
type TierChange struct{ From, To string }

func Format(kind Kind, name string, tier *TierChange) string
```

`tier` is optional: pass nil when no tier boundary was crossed. The result has
no trailing newline.

## Gotchas

- **Centring is by rune count against the rule width** (64 runes of `━`), not
  by client width. A name longer than the rule is returned unpadded.
- **The returned string is plain text with no ANSI tags.** Color comes from
  delivery: since messaging M3 item 6 the banner is sent on
  `messaging.CategorySkillProgress`, whose color stage wraps the whole block in
  the `skill-progress` alias (179, gold).
- **`Format` returns text; it does not send.** Its only consumer,
  `internal/characters/progression.go`, hands the banner to
  `notifyProgression`, which calls the callback `main.go` registers
  (`hooks.ProgressionNotifyCallback`).

## Dependencies

`strings`, `unicode/utf8`. Nothing in the repo.

## Consumers

`internal/characters/progression.go` only (skill and stat advancement).
