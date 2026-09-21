# M4e PR 1a: the darkness diff, all three sight tiers

**What this is:** the deliberate, player-visible half of M4e PR 1a, recorded so
a reviewer can judge it without running the server.

The wording half of the PR is byte-identical and proved so by three independent
tests (the 118-row net, the grapple pin test, the store golden). This document
covers the part that is **meant** to change.

## The change in one line

The thirteen mob special-move files each carried a hand-rolled darkness branch
that chose between a named sentence and a hardcoded anonymous twin. Those
branches and their twins are deleted. `messaging.SendTrio` already hides each
party's name from a reader who cannot make them out, judged by that reader's
`messaging.SightDecision`, and it does so at **three** tiers where the deleted
code had **two**.

## Measured output

Rendered through `messaging.HideNames` on the real shipped line
(`kick`, `standard_hit`, actee role), mob named "Stone Beetle Queen", damage
description "serious wounds". These are exact strings, not reconstructions.

| Reader | Line |
|---|---|
| **Before**, could see | `<ansi fg="mobname">Stone Beetle Queen</ansi> kicks you hard! (<ansi fg="damage">serious wounds</ansi>)` |
| **Before**, could not see (BOTH infrared and blind) | `Something kicks you hard! (<ansi fg="damage">serious wounds</ansi>)` |
| **After**, `SightFull` | `<ansi fg="mobname">Stone Beetle Queen</ansi> kicks you hard! (<ansi fg="damage">serious wounds</ansi>)` |
| **After**, `SightShapes` (infrared in the dark) | `<ansi fg="combat-anon">A figure</ansi> kicks you hard! (<ansi fg="damage">serious wounds</ansi>)` |
| **After**, `SightNone` (fully blind) | `<ansi fg="combat-anon">Something</ansi> kicks you hard! (<ansi fg="damage">serious wounds</ansi>)` |

**The improvement is the middle row.** An infrared character in a dark room
previously read the fully-blind word. They now read the shapes word, which is
what every other part of the game already tells them.

Two smaller changes ride along:

- The anonymous word now carries `<ansi fg="combat-anon">`. Everyone sees it in
  the anonymity colour. This is the accepted side effect carried over from M4d
  PR 2, where the same tag was applied to combat defence lines.
- The word is capitalised at sentence start (`A figure`, `Something`), handled
  by `HideNames`' `atSentenceStart`, which looks back through ansi tags.

## The defect this fixes, in the words of the code

`internal/mobcommands/darkness.go` defined, until this PR:

```go
func canSeeInDark(u *users.UserRecord, room *rooms.Room) bool {
	return room.GetVisibility() >= 1 || u.Character.HasFlagFromAnySource(conditions.NightVision)
}
```

It is binary and predates `SightDecision`. Everyone who failed it was routed
through `messaging.Anonymize`, which knows exactly one word, "a figure". So a
**fully blind** player read the shapes-tier word. A live playtest during M4d
PR 2 found this and it was filed; this PR is the fix.

Measured surface at the start: **30 references across 20 files**, 24 of them
executable call sites, plus an unexported byte-identical twin in
`internal/usercommands`. All retired. `grep -rn canSeeInDark --include=*.go
internal/` now finds only historical comments.

## Where the deletion was NOT safe

Deleting a darkness branch is only correct where `SendTrio` delivers the line
**and** the `Audience` declares the name, because that is what lets the
pipeline hide it. Each site was tested against both conditions before being
touched.

- **`internal/mobcommands/attack.go` makes ZERO `SendTrio` calls.** Deleting
  its branch would have leaked the mob's name to a blind player. It keeps an
  explicit `ParticipantSight` + `HideNames` substitution at the site, and its
  wording stays in Go: putting it on `SendTrio` is a separate change.
- **The audio-channel siblings in `howl.go` and `taunt.go`** bypass the sight
  gate by design: you hear a howl whether or not you can see, but it must not
  tell you who. They moved onto a new per-listener helper,
  `sendAudioRoomTextHidingNames`, rather than onto the pipeline.
- **`skill_move_defence.go` (both packages)** deliberately hands `SendTrio`
  pre-anonymised text; `internal/messaging/trio.go`'s own docstring names that
  file as the sanctioned exception. It keeps doing so, but through `HideNames`
  instead of `Anonymize`, so the three tiers survive.

## Tests that had encoded the defect

Three pre-existing tests asserted that a fully blind reader sees "a figure".
They had pinned the bug as expected behaviour. All three now assert the real
split, and are stronger than before because they distinguish the two readers in
the same test:

- `internal/mobcommands/predator_test.go`,
  `TestMobDefyRoutingExcludesDefenderAndAnonymizesDarkIdentity`
- `internal/mobcommands/predator_test.go`,
  `TestMobTauntAndHowlRuntimeHideIndexedActorAndExcludeDefender`
- `internal/mobcommands/taunt_store_test.go`,
  `TestMobTauntTriadAnonymizesInTheDark`

Two hits in `internal/usercommands/craft_instant_room_hiding_test.go` were
checked and **left alone**: their watcher genuinely carries infrared, so "a
figure" is the correct word there. That distinction is the whole point of the
change, so it is recorded here rather than assumed.

## One wording change beyond the tier fix

`howl`'s blind personal line previously had a bespoke blind-only sentence. It
now renders the named sentence with the name substituted, so for example
"A menacing howl shakes your resolve!" becomes "Something's menacing howl
shakes your resolve!". Grammatical, and consistent with the existing
`hideIdentitiesInPersonalLines` idiom in `internal/combat`, but it is a
different sentence from the deleted literal and a reviewer should see it.

## New tests proving the fix

Both were proven capable of failing before being trusted: with the old
predicate restored, the blind subtest goes red reading "a figure".

- `internal/mobcommands/darkness_tiers_test.go`
- `internal/usercommands/darkness_tiers_test.go`

## What the playtest can and cannot show

- 🪤 **The shapes tier will not exercise itself.** M4d PR 2's run failed to
  load condition 85 at runtime and got the blind tier twice; its guard covers
  template loading only, not materialize-to-login, and it had already failed
  the same way on 2026-09-11. If infrared is not confirmed live in session, the
  shapes tier is UNVERIFIED and must be reported as such.
- 🪤 **Colour is unverifiable by the harness**, which strips ANSI on the AI
  port. Seeing `combat-anon` requires telnet on 33333.
