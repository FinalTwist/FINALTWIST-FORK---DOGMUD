# internal/messaging

Centralized player-facing-text pipeline. Every `Room.SendText` /
`Room.SendTextVisual` / `UserRecord.SendText` / `Actor.SendText` call
in the engine flows through this package's pipeline before reaching
the recipient's connection.

## Pipeline Stages

1. **Compose** — caller produces `(Category, text)`.
2. **Style normalize** — sentence-start caps, a/an agreement,
   duplicate-word collapse, sentence-end punctuation, ANSI canon for
   names. Per-Category skip table in `normalize.go`.
3. **Sight gate** (visual channel only) — per-recipient: CanSeeClearly,
   CanSeeShapes, or skip-visual-deliver-audio. Consumes the chunk-6
   Perception FSM (see `internal/state/perception/`).
4. **Anonymize** (infrared-only path) — regex strips `username` /
   `mobname` / `petname` ANSI name tags, including suffixed mob tags such as
   `mobname-dup2`, then substitutes "a figure" + the `combat-anon` color alias.
5. **Apply category color tag** — `<ansi fg="<category-alias>">…</ansi>`.
6. **Wrap** at recipient's `UserRecord.LineWidth` (default 80, range
   40–240), ANSI-aware.
7. **Deliver** to the recipient's connection.

## Channels

| Channel  | Helper            | Sight-gated | Stages run            |
|----------|-------------------|-------------|-----------------------|
| Audio    | `SendText`        | no          | 1, 2, 5, 6, 7         |
| Visual   | `SendTextVisual`  | yes         | all 7                 |

Audio bypasses the sight gate and the anonymizer; visual runs the
full per-recipient pipeline.

## Public API

Types and constants:

- `Category` — enum of 61 text classes (combat hits, defense, grapple,
  submissions, specials, spells by school, social, system, environment,
  loot/equipment/condition/mutation/toxin; plus `CategoryCombatSummary` for
  the per-round compact tally emitted by the light-verbosity path, and
  `CategoryCombatBlindWarning` for the per-round "you can't see clearly"
  notice, M4d PR 2 Task 4, sent by `internal/hooks`'
  `flushBlindCombatNotices`). `CategoryCombatBlindWarning` is deliberately
  absent from both `verbosity.go` suppression tables, so it passes at
  every verbosity level today.
  `Category.String()` no longer spells the three defence names as local
  literals: `CategoryDodge` / `CategoryParry` / `CategoryBlock` return
  `string(combatvocab.DefenceDodge)` / `DefenceParry` / `DefenceBlock`, the
  same one-declaration constants messaging M4b-2 gave `internal/combat`,
  `internal/characters` and `internal/items` (see
  `internal/combatvocab/context.md`).
- `Verbosity`, `ParseVerbosity`, `(Verbosity).Suppresses` — combat-text
  verbosity primitives in `verbosity.go`. The allowlists
  (`suppressibleAtMedium`, `suppressibleAtLight`) declare which
  categories each level may drop. Suppression is applied by the combat
  hooks (`internal/hooks/combat_verbosity.go`), not by this pipeline
  itself — the pipeline delivers whatever the hook passes through.
- `Channel` — `ChannelAudio`, `ChannelVisual`.
- `SightDecision` — `SightFull`, `SightShapes`, `SightNone`.
- `RenderInput` — bundles Category, Text, Channel, SightDecision,
  LineWidth for one recipient's pipeline pass.
- `RoomVisibility` — minimal interface (`GetVisibility() int`)
  satisfied by `*rooms.Room`.
- `Line` — `{Text string; Cat Category}`, one audience's view of an
  event. Built with `Say(cat, text)`.
- `NoLine` — the zero `Line`. A viewpoint that deliberately has
  nothing to say. Spelled out so a considered silence is legible as
  one; the root guard requires it rather than an omitted field.
- `Trio` — `{Actor, Actee, Observer, RemoteObserver Line}`, one narrated
  event as its FOUR audiences see it. `RemoteObserver` is the second
  room's line: ranged combat narrates to both the attacker's room and
  the defender's room, and until M4d that second audience had no seat
  here at all — it travelled outside the pipeline. The root guard
  `TestEveryTrioLiteralNamesAllThreeRoles` (repo root,
  `messaging_surface_guard_test.go`) still requires only the original
  three (`Actor`/`Actee`/`Observer`) on every `messaging.Trio{}`
  literal; `RemoteObserver` is deliberately NOT added to it. A census
  on 2026-09-20 found 150 `Trio` literals across 31 non-test files —
  requiring the fourth field on all of them would leave 149 carrying
  an always-empty field forever, which teaches an author to paste it
  unread rather than reason about it. The narrower, PAIRED guard that
  replaces it is `TestRemoteRoomIsPairedWithRemoteObserver` (same
  file): within one function, setting `RemoteRoom` on an `Audience`
  literal without also setting `RemoteObserver` on a `Trio` literal in
  that function fails, because `SendTrio` silently delivers nothing to
  a `RemoteRoom` whose `Trio` never named `RemoteObserver` — the exact
  "wired the mechanism, dropped the narration" defect class the M1
  viewpoint audit found repeatedly. Proved capable of failing with a
  probe in `internal/combat/combat.go`; see the M4d Task 5 report.
- `Recipient` — minimal interface (`SendText(cat, text)`) satisfied by
  `*users.UserRecord` and by `actions.Actor`.
- `Broadcaster`: interface satisfied by `*rooms.Room`:
  `SendTextVisualHidingNames(cat, txt, names, excludeUserIds ...int)` and
  `ParticipantSight(userId int) SightDecision`.
- `Audience`: who is present for one event: `Actor`/`ActorId`/`ActorName`,
  `Actee`/`ActeeId`/`ActeeName`, `Room`, and `RemoteRoom`. Ids are passed
  rather than derived because `users.UserRecord.UserId` is a FIELD while
  `actions.Actor` exposes `GetUserId()`. The names are exactly as the lines
  print them; the root guard requires both on every `Trio` literal.
  `RemoteRoom` is the second room for a ranged event (the defender's room);
  nil sends nothing, which is every event except ranged combat. A nil
  recipient/broadcaster must be assigned as the interface, never as a
  typed-nil pointer — a `(*users.UserRecord)(nil)` stored here is a
  non-nil interface value and `SendTrio` would call through it and panic.
- `NoName`: the empty string, for a side of an Audience with nobody on it.

Functions:

- `RenderForRecipient(in RenderInput) string` — entry point; runs the
  full pipeline for one recipient. Empty return = "don't deliver".
- `ParticipantSight(observer *characters.Character, room RoomVisibility) SightDecision`
  is THE optics primitive, added M4d (`01bbee127`). It answers what an
  observer can make out and nothing else — blindness, room light,
  NightVision, InfraredVision — and deliberately does NOT consult sleep,
  because sleep is an attention property, not an optical one: a sleeping
  character's eyes work, they are simply not reading. `SightFull` when
  light or NightVision allow clear sight; `SightShapes` for an unblinded
  observer with InfraredVision in the dark; `SightNone` otherwise. A nil
  observer sees fully.
- `CanSeeClearly`, `CanSeeShapes`, `CanSeeSightImpairedOnly` — each is now a
  ONE-LINE POLICY over `ParticipantSight` that composes its own attention
  rule, not three independently-implemented predicates:
  - `CanSeeClearly(observer, room) bool` = awake AND `ParticipantSight ==
    SightFull`. Read by the room-broadcast sight gate (visual channel).
    Sleep-gated: a sleeping player stops receiving visual room lines.
  - `CanSeeShapes(observer, room) bool` = awake AND (`ParticipantSight ==
    SightFull` OR `== SightShapes`). Also sleep-gated: closed eyes see no
    shapes either.
  - `CanSeeSightImpairedOnly(observer, room) bool` = `ParticipantSight ==
    SightFull`, WITHOUT the sleep gate. This is the one `internal/combat`
    used to read (as `CanSeeClearly`, before M4d) to drive
    `Balance.DarknessCombatPenalty`; adding the sleep gate to
    `CanSeeClearly` on 2026-08-31 would otherwise have applied a phantom
    darkness penalty to a sleeping defender standing in a LIT room, and
    corrupted `combat-analytics.jsonl`'s contest telemetry with a term
    nobody asked for. M4d closed that gap for good: combat no longer reads
    a messaging predicate by name at all (see `internal/combat/context.md`,
    "Sight and the darkness penalty").
  - `sleep_policy_test.go` pins the contract by absence: a sleeper reads
    NOTHING from `CanSeeClearly`/`CanSeeShapes` (both false regardless of
    light), and `CanSeeSightImpairedOnly` ignores sleep entirely.
    `optics_pin_test.go` pins all three against an eight-row truth table
    (light x blind x infrared x asleep).
- `HideNames(text string, names []string, d SightDecision) string`: replaces
  each name with "a figure" (shapes) or "something" (none), longest name first,
  capitalized at a sentence start. In bare prose the match is exact and
  whole-word, because mob names collide with ordinary words ("guard"). Inside an
  identity tag it ignores case and any duplicate index, and the whole tag is
  replaced: one Audience name then covers both the authored form ("skeleton")
  and the display form the channel defence triad prints ("Skeleton #2"). When
  the whole identity tag is hidden, one directly following adjective span
  (` <ansi fg="black-bold">(...)</ansi>`, as `FormattedName.String` prints it,
  with the adjective colour-patterned rune by rune into nested tags) goes with
  it, so a caller may pass formatted names through the seam without leaking
  "(dead)" or "(♥friend)". A match inside tag markup itself is never
  replaced.
- `Normalize(cat Category, text string) string`
- `Anonymize(text string) string`: the pipeline's infrared fallback for every
  visual line. Replaces each identity tag with "a figure" and takes the
  adjective span behind it (same pattern as `HideNames`), because
  `rooms.go` anonymizes BEFORE it hides names and the span would otherwise
  survive as "a figure (dead)".
- `WrapAnsi(text string, maxWidth int) string`
- `Say(cat Category, text string) Line`
- `SendTrio(t Trio, aud Audience)`: delivers one narrated event to
  everyone entitled to it — FOUR roles since M4d (`44cc90ceb`): Actor,
  Actee, Observer, and RemoteObserver. A line goes out only if it has BOTH
  text and a recipient. The room broadcast (`Observer`) and the remote-room
  broadcast (`RemoteObserver`, delivered to `aud.RemoteRoom` when set) ALWAYS
  exclude the actor and the actee. Each role is rendered for its reader: the
  actor's line hides `ActeeName` and the actee's hides `ActorName` by that
  reader's `ParticipantSight`; the observer and remote-observer lines hide
  both, judged per-observer by their own room's `ParticipantSight`.

## Two jobs, not one

This package now does two things, and the second is not the first.

1. **The seven-stage pipeline, per recipient.** `RenderForRecipient`.
2. **Fan-out of one event to its audiences (three, or four for a ranged
   event with a remote room).** `SendTrio`.

The fan-out lives here because the import graph rules out both
alternatives. `internal/narration` cannot import `messaging` (`items`
imports `narration`, and `messaging` reaches `items` through
`characters`), and `internal/actions` cannot be imported by
`questengine`, which has to reach the seam in M3. `messaging` hosts it
with **zero new imports**, using the same minimal-interface trick
`predicates.go:11` documents.

⚠️ **The category rides on the `Line`, not on the `Trio`, and that is
load-bearing.** The three roles of one event routinely differ: ten of
twelve player-side special-move verbs send personal lines as
`CategorySystem` and the room line as something verb-specific, `shoot`
uses four categories on the personal side, and `throw` uses three on
each side. A single-`Category` seam would silently recategorise
them, and category decides both the line's colour and whether a
light-verbosity player sees it at all.

## Import-direction discipline

`messaging` imports `internal/characters`, `internal/conditions`,
`internal/state/perception` directly (sight predicates need
Perception FSM state and the NightVision / InfraredVision condition
flags). Everything else — `rooms`, `users`, `mobs`, `combat`,
`hooks` etc. — is consumed via narrow interfaces (`RoomVisibility`,
`Recipient`, `Broadcaster`) so the dependency arrow stays one-way:

- Many packages import `messaging` (combat, hooks, rooms, users,
  actions, behaviortree, questengine, modules, world.go, …).
- `messaging` imports characters/conditions/state/perception, plus
  `internal/combatvocab` (added M4b-2, `d4e5ab20d`) for the three
  defence-name constants `Category.String()` returns. `combatvocab`
  imports nothing but the standard library, so this adds no cycle risk.
- Nothing in `characters` imports `messaging` (would close a cycle).

> **Corrected 2026-09-08.** This file previously documented four
> symbols that do not exist and never did in this package:
> `UserSender`, `ProgressionKind`, `TierChange`, `FormatProgression`
> and `SendProgression`, the last two described as living in a
> `progression.go` the package does not contain. A comment in
> `internal/banner/banner.go` pointed at that same missing file.
> Verify a `context.md` against `Select-String -Path
> internal\<pkg>\*.go -Pattern '^(func|type|const|var)\s'` before
> trusting it; a file describing an invented API is worse than none,
> because someone codes against it.

## Adding a new Category

1. Add a constant to the enum in `messaging.go`. Append at the end of
   its section.
2. Add the matching string in `Category.String()`.
3. Add the color alias in `_datafiles/world/dogmud/ansi-aliases.yaml`
   named `<category-name>` where `<category-name>` is the string the
   enum returns.
4. If the new Category needs style-normalization skips, edit the
   `normalize.go` skip table.

## See Also

- `docs/superpowers/specs/completed/2026-05-19-messaging-framework-design.md` —
  full design spec.
- `internal/state/perception/context.md` — the FSM whose state the
  sight gate reads (shipped dormant in chunk 6; this chunk is the
  consumer).
- `_datafiles/world/dogmud/ansi-aliases.yaml` — color aliases.

## Files

The package is the pipeline, one stage per file, plus the fan-out (`trio.go`):

| File | Stage |
|------|-------|
| `messaging.go` | Entry points and the `Category` vocabulary |
| `pipeline.go` | Stage ordering — compose → normalize → anonymize → color → wrap → deliver |
| `normalize.go` | Grammar and article normalisation |
| `anonymize.go` | Replacing names the observer should not see (infrared fallback, whole-line) |
| `hidenames.go` | `HideNames` — replacing specific names in bare prose, longest-first, whole-word |
| `hidenames_tagged.go` | Identity-tag-aware name replacement `HideNames` and `Anonymize` share, including the trailing adjective span |
| `wrap.go` | 80-column wrapping (uses visible width, not byte length) |
| `predicates.go` | `ParticipantSight` (the optics primitive) plus `CanSeeClearly`/`CanSeeShapes`/`CanSeeSightImpairedOnly`, the one-line attention policies built on it |
| `verbosity.go` | Per-player verbosity filtering |
| `trio.go` | `Line`/`Trio`/`Audience`/`SendTrio` — fan-out of one narrated event to its four audiences |

Adding a transformation means adding a stage here, not special-casing at a call
site — that centralisation is the point of the package.
