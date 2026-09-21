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

## Playtest result, 2026-09-21, run `4b7aba52aa33ad4f`

Commit `59a8d7f52`, clean tree, room 3101 (Cave Mouth, `biome: cave`),
profile `slice-a-infrared`, feature-tester. **Outcome: PARTIAL.**

### Verified: the blind tier renders correctly

Captured live, verbatim:

```
⚡ SWEEP! Something dodges and lashes at your legs. You stay up, but it still
catches you! (negligible damage)
Something sidesteps your clumsy Iron Longsword with ease!
Something hangs back in the dark, biding its time.
You hear something collapse to the ground.
You cannot see clearly, so your attacks and defense are weaker.
```

Four things this proves:

1. **The word is "Something", the `SightNone` word.** Capitalised at sentence
   start after the `⚡ SWEEP!` banner, which is `HideNames`' `atSentenceStart`
   looking back through tags, exactly the case its docstring gives as its
   example.
2. **The move is still identifiable.** The sweep reads as a sweep, with the
   defence, the knockdown result and the damage band intact. Only the identity
   is gone. That is the entire point of the slice: the deleted hardcoded twin
   could only ever say one fixed sentence.
3. **Zero raw tokens.** Scanned every output line for `{...}`: **0 found**.
   A literal `{actor}` or `{damage}` reaching a player was the most serious
   thing this run could have found.
4. **`look` returns "You can't see anything!"**, confirming the room is
   genuinely unlit and the reader is `SightNone`, so the lane is valid.

### NOT verified: the shapes tier, for the THIRD time

🔴 **`conditions` returned `None`.** The `slice-a-infrared` profile declares
condition 85 (InfraredVision) with `triggersleft: 999999`, and the character
still reached the world without it.

This is the **third** recurrence: 2026-09-11, 2026-09-20 (M4d PR 2) and now
2026-09-21. Each time the run silently tested the blind tier while the profile
name said otherwise. The profile's own comment already documents the
born-dead-condition trap and is not the cause, because the condition is not
expired, it is **absent**.

**So the `SightShapes` tier has never been exercised live, and it is the tier
this PR changes.** Reported as unverified rather than implied to pass.

The guard that exists covers template loading only, not materialize-to-login.
Filed as its own follow-up: something between the profile's saved condition
list and the character reaching the world drops it, and until that is found,
no playtest can verify an infrared behaviour at all.

### Also not covered

- **Only one distinct special move was captured.** The room's mobs died and did
  not respawn inside the window, so kick, bash, gore and the rest were not
  observed live. They share one code path with the sweep and are covered by the
  118-row net, but that is source-level proof, not play.
- **Colour is unverifiable by harness**: the AI port strips ANSI, so the
  `combat-anon` tag cannot be seen this way. Needs telnet on 33333.

### Manual check that would close both gaps

For the two things the harness structurally cannot do, over telnet 33333 with
an admin character in an unlit cave room such as 3101:

1. Grant infrared, then have a mob use a special move on you. The line must
   read **"a figure"**, not "something".
2. Capture the raw bytes and confirm the stand-in word carries
   `<ansi fg="combat-anon">`.

## Two leaks the owner found in play, and the census they prompted

The owner's manual telnet run (2026-09-21) caught two lines naming a mob
outright in a pitch-dark room, while every combat line in the same round
called that same mob "something":

```
You shift your focus to Cave Crawler!
Cave Crawler moves with increasing swiftness.
```

**Both fixed in this PR.** Neither was in the thirteen migrated files.

- **The focus notice** (`hooks/NewRound_DoCombat.go`, `usercommands/target.go`
  at two sites) rode `CategorySystem` on a raw `SendText`, which bypasses the
  sight gate entirely. Each personal line now hides its subject by the
  READER's own sight. A third personal line was found in the same block and
  fixed with them: `p.SendText(... "X shifts focus to you!")` told a target
  who was aiming at them even when that target could not see.
- **The mob stat-gain emote** (`hooks/NewRound_DoCombat_unified.go`, via
  `characters.MobStatGainMessages`) used `Room.SendText`. It is a PURELY
  VISUAL line, so it now uses `SendTextVisualHidingNames`: a reader who cannot
  see receives nothing at all, and a shapes-only reader reads "a figure".
- The two room lines in `target.go` were already sight-gated but passed NO
  names, so they leaned on tag-based `Anonymize` and collapsed the shapes and
  blind tiers into one word. They now pass both names.

### 📌 FILED, not fixed: eight more room broadcasts bypass the sight gate

Measured 2026-09-21 by sweeping for raw `SendText` room broadcasts carrying an
identity tag. **79 raw identity-tagged sends exist overall**, most of them
legitimately transactional (shop and purchase lines). Eight are room
broadcasts that skip the gate:

| Site | Line | The question it needs answered |
|---|---|---|
| `NewRound_DoCombat_helpers.go` | "X is blocked from fleeing by Y!" | visual only, or audible? |
| `NewRound_DoCombat_helpers.go` | "X flees to the Y exit!" | would a blind character HEAR someone flee? |
| `NewRound_DoCombat_helpers.go` (x2) | "The ITEM X was carrying breaks!" | a break is loud; audible, surely? |
| `mobcommands/sayto.go` (x4) | "X says to Y, ..." | speech is audio, so it arrives, but should it NAME? |

**These are not mechanical fixes.** Each needs a ruling on whether the event
is visual-only (suppress for a blind reader) or audible (arrives, name
hidden), and the speech four belong with the `sendAudioRoomText` follow-up
already filed for the four speech commands. Guessing per line is how the two
tiers got collapsed in the first place.

## What the playtest can and cannot show

- 🪤 **The shapes tier will not exercise itself.** M4d PR 2's run failed to
  load condition 85 at runtime and got the blind tier twice; its guard covers
  template loading only, not materialize-to-login, and it had already failed
  the same way on 2026-09-11. If infrared is not confirmed live in session, the
  shapes tier is UNVERIFIED and must be reported as such.
- 🪤 **Colour is unverifiable by the harness**, which strips ANSI on the AI
  port. Seeing `combat-anon` requires telnet on 33333.
