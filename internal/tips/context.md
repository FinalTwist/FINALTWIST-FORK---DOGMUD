# Tips Context

## Purpose

`internal/tips` is the periodic gameplay tip store: short pieces of advice
broadcast to every player who has not turned them off, one per interval, in
file order, loaded from `DataFiles/tips.yaml`. Before messaging M3 item 7
(2026-09-15) these were called hints, a name that collided with the quest
`hint` command and with per-dialogue hints; the broadcast, its toggle and the
saved on/off flag were all renamed to "tips" to end that collision. Dialogue
hints and the quest `hint` command are unchanged and keep the old word.

## API

```go
func Load()
func Validate(t []string) error
func Next() string
func Count() int
func All() []string

// test helper, in test_helpers.go
func SeedForTest(t []string) func()
```

`Next` returns the next tip in rotation order and advances a package-level
cursor that survives a reload; it returns `""` for an empty store.

## The no-length-rule decision

`Validate` refuses only a blank tip; there is deliberately no 80-column rule.
56 of the 74 shipped tips already exceed 80 characters once tags are stripped
(longest 211), because each is a folded YAML scalar that joins its authored
line breaks with spaces, and the tips broadcast has no server-side wrap
(`shouldWrap` returns false for `CategoryTip`). A length rule in `Validate`
would fail boot on data that already ships. Wrapping tips is filed as
messaging M6 content ledger row 22, for M5's wrap decision; item 7 only
corrected the file's own header comment, which had falsely claimed every tip
fit in 80 characters.

## Consumer

`internal/hooks.BroadcastTips` (`NewRound_BroadcastTips.go`) is the one
production caller: it calls `tips.Next()` every `tipIntervalRounds` rounds,
prefixes `[Tip]`, skips a `Deafened` user and a user whose `tips` config
option is `false`, sends on `messaging.CategoryTip`, and queues an
`events.Communication`. `Looking_HandleLookTips.go`'s `HandleLookTips` is a
separate one-shot contextual tip and does not read this store.

## The player setting

The on/off flag lives in `UserRecord.ConfigOptions["tips"]`
(`yaml:"configoptions"`), toggled with `set tips` (`set hints` stays a silent
alias, `internal/usercommands/set.go`). Migration 0.18.0
(`migrate_TipsConfigOption`, `internal/migration/0.18.0.go`) renames an
existing save's `configoptions.hints` to `configoptions.tips` on first boot
after the setting's rename, so a player who had turned hints off does not
find tips silently back on. The flag is on the user record, not the
character, so there is no alts trap.

## Dependencies

`configs`, `mudlog`, `yaml.v2`, plus stdlib.

## Gotchas

- **`Load` panics on a bad file; a missing file is an empty store.** Same
  contract as `internal/gossip.Load`: a read error other than "does not
  exist", a parse error, or a failed `Validate` panics.
- **The rotation cursor is process-global state, not per-player.** Every
  player who is due a tip in the same round gets the same tip; `Next()`
  advances once per broadcast, not once per recipient.
