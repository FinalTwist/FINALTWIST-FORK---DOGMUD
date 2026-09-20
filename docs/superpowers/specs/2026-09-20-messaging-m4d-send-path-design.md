# Messaging M4d: One Send Path, One Sight Verdict

Design for M4d of the messaging unification arc, written 2026-09-20 on master
`523ea3cc4` (M4c merged as PR #147). Supersedes the M4d section of
`docs/superpowers/specs/2026-09-17-messaging-m4-flip-design.md`, whose numbers
and open question predate M4b-2, the counters slice and M4c.

M4d is **two PRs** by owner ruling: PR 1 is byte-identical plumbing, PR 2 is the
player-visible combat change and carries the playtest gate the owner deferred
from M4c.

---

## Facts verified against source (2026-09-20, master `523ea3cc4`)

Every row was read from the file named, today.

| Fact | Value | Source |
|---|---|---|
| `SendTrio` call sites | **151 in 32 files** (non-test); the 09-17 spec said 149 in 31 | grep |
| `Trio` roles | **3**: `Actor, Actee, Observer` | `internal/messaging/trio.go:40` |
| Core roles | 4, so `RemoteObserver` has no seat on `Trio` | `internal/narration` |
| `SendTrio` participant hiding | per reader, `HideNames(text, [otherName], room.ParticipantSight(readerId))` | `trio.go:99-119` |
| `SendTrio` observer hiding | `SendTextVisualHidingNames` with both names | `trio.go:106-109` |
| Hidden-name wording | `something` at `SightNone`, `a figure` at `SightShapes` | `internal/messaging/hidenames.go:42-45` |
| Sight predicates | `CanSeeClearly` (12 sites), `CanSeeSightImpairedOnly` (15), `CanSeeShapes` (4) | `predicates.go:24,72,92`; grep |
| Sleep inside the predicates | `CanSeeClearly` and `CanSeeShapes` test `HasConditionFlag(conditions.Sleeping)`; `CanSeeSightImpairedOnly` deliberately does NOT | `predicates.go:39,105` |
| `SightDecision` | `SightFull / SightShapes / SightNone` | `internal/messaging/pipeline.go:32` |
| 🔑 Verdict producer ALREADY EXISTS | `messaging.ParticipantSight(observer, room) SightDecision`, and `rooms.Room.ParticipantSight(userId)` wraps it | `predicates.go:139`; `internal/rooms/rooms.go:321` |
| Its dependency direction | **inverted**: it is built ON `CanSeeSightImpairedOnly`, so a boolean is the primitive | `predicates.go:140` |
| `CanSeeSightImpairedOnly` and infrared | returns NightVision only; an infrared-only observer in the dark is FALSE for it | `predicates.go:74-84` |
| Combat participant delivery | bare `u.SendText`, no sight gate, verbosity only | `internal/hooks/combat_verbosity.go:304-315` |
| Combat spectator delivery | `room.SendTextVisualToUser` | `combat_verbosity.go:349` |
| Combat named tally gate | `CanSeeClearly`, which also excludes sleepers | `combat_verbosity.go:386` |
| Combat's own sight flags | `sourceCanSee` / `targetCanSee`, set from `CanSeeSightImpairedOnly` | `internal/combat/combat.go:55,56,106,107,150,151,199,200` |
| What those flags DO | **only** the `DarknessCombatPenalty` on attack and defence scores | `internal/combat/combat_helpers.go:557,749` |
| 🔴 Dark combat prose | `replaceDarknessMessages` DISCARDS the composed line and substitutes hardcoded Go literals | `internal/hooks/NewRound_DoCombat_helpers.go:438-505` |
| How many such literals | **12**: six for the attacker who cannot see (`:450`), six for the defender who cannot see (`:478`) | same file |
| Where it is called | gated on `CanSeeSightImpairedOnly` for whichever side is a player | `NewRound_DoCombat_unified.go:549-558` |
| Status prompt | ALREADY correct: `{target}` prints `an unseen foe`, `{targethealth}` suppresses | `internal/users/userrecord.prompt.go:530-556` |
| 🔴 GMCP `Char.Enemies` | sends `mob.Character.Name`, `DisplayHealth()` and `MaxHp` with **no sight check** | `modules/gmcp/gmcp.Char.go:431-459` |
| Crafting delivery | `user.SendText` direct | `internal/usercommands/craft.go:73,79,85,89` |
| position_control delivery | `u.SendText` plus `r.SendTextVisual` | `internal/hooks/Position_Messaging.go:233,249,305` |
| Raw-message guard | `TestNoRawEventsMessageOutsidePipeline` | `raw_events_message_guard_test.go:27` (repo root) |

### The 09-17 spec's open question, answered

The old spec asked whether combat buffers "already anonymize at composition",
and said the answer decides whether combat's change is plumbing or player
visible. **Neither.** Combat does not anonymize; it *replaces*. When a
participant cannot see, `replaceDarknessMessages` throws the composed line away
and substitutes one of twelve hardcoded sentences.

That is worse than the spec assumed, in a specific way: the substitution
discards everything the line knew. Weapon and species flavour, the defence that
won, and the band it drew from all collapse into "Your attack is turned aside by
something!" **M4c's band work does not reach dark fights at all**, because dark
fights never render from the store.

---

## Owner rulings, 2026-09-20 (do not relitigate)

1. **Identity is hidden when you cannot see**, in combat as everywhere else.
   Darkness must not be cosmetic, and combat is where players spend their time,
   so it is the one place an exception would matter most.
2. **Mechanism is `HideNames` on the real line, plus a per-round notice.** Not
   the bespoke substitution (which loses flavour), and not silent anonymisation
   (which loses the explanation). The notice fires **every round**, not once per
   fight, and not per swing.
3. **Two PRs, cut by risk**: plumbing first, combat second.
4. **The `Char.Enemies` sight gate rides along in PR 2.** `Room.Info` and the
   contents roster stay with the separately filed GMCP slice.
5. Owner correction to a standing note: PR notification spam comes from **CI
   failures**, not from PR count. Slice by risk, not by inbox.
6. **Infrared characters take a REDUCED darkness combat penalty, not zero and
   not the full one.** Today `DarknessCombatPenalty` (0.80, a flat 20% to hit
   and defend) is binary and an infrared character takes it in full, because
   `CanSeeSightImpairedOnly` consults NightVision only. Seeing shapes should be
   worth something and should not be worth everything.

---

## PR 1: one sight verdict, four audiences, one path

Byte-identical. No player reads anything different.

### One producer, several named questions

🔑 **CORRECTION, 2026-09-20, found while planning: the producer already exists.**
`messaging.ParticipantSight(observer, room) SightDecision` (`predicates.go:139`)
already evaluates blindness, room light, NightVision and InfraredVision into one
verdict, and already leaves sleep out. This spec's first draft said it had to be
created, because a grep for `func Sight` found nothing. That grep could only
ever have matched a function with that exact name, which is the negative-result
trap: absence of a name is not absence of a mechanism.

So M4d does not build a producer. **It inverts a dependency.** Today
`ParticipantSight` is defined in terms of `CanSeeSightImpairedOnly`, so a
boolean predicate is the primitive and the richer verdict is derived from it.
That is backwards, and it is why the optics are written out three times.

After M4d, the verdict is the primitive: it computes optics directly, and the
three booleans are one-line policies over it. Fix the primitive, not the call
sites.

The three predicates **stay as named policies**. They are not deleted. The
defect is three independent implementations of the same optics, not three
names; deleting the names would push the rule out to 31 call sites and lose the
reason each site chose what it chose.

### Sleep leaves the sight verdict

This is the design point the current code is already straining against.
`CanSeeClearly` tests `Sleeping`; `CanSeeSightImpairedOnly` deliberately does
not, and both carry comments explaining the split. `combat_verbosity.go:378`
documents it a third time.

Sleep is an attention property, not an optical one. A sleeping character's eyes
work; they are simply not reading. So:

- `ParticipantSight()` answers optics only and never consults `Sleeping`, which
  is already true of it today.
- The **predicates** compose attention over it, inside themselves, so no call
  site changes and PR 1 stays byte-identical:
  - `CanSeeClearly(obs, room)` = `awake(obs) && ParticipantSight(...) == SightFull`
  - `CanSeeShapes(obs, room)` = `awake(obs)` and the verdict is `SightFull` or `SightShapes` (equality, not a comparison: the constants run best to worst)
  - `CanSeeSightImpairedOnly(obs, room)` = `ParticipantSight(...) == SightFull`,
    with no attention test, which is what it means today and why it exists.

⚠️ **That last line is `== SightFull`, not `>= SightShapes`.** An earlier draft
of this spec wrote the loose comparison from the function's NAME. Its body
returns `HasFlagFromAnySource(NightVision)` in a dark room and never consults
`InfraredVision`, so an infrared-only observer fails it today. Widening it to
`>= SightShapes` would hand every infrared character a silent removal of the
`DarknessCombatPenalty`, which is a balance change wearing a refactor's clothes.
The plan pins this with a test before the inversion.

One producer then serves all three questions without a boolean parameter, each
predicate states its own attention policy in one line, and the comment
explaining the split stops needing to exist in three files.

### Combat carries the verdict, not a boolean

Combat currently borrows `CanSeeSightImpairedOnly` for something that is not a
narration question at all: the `DarknessCombatPenalty` applied to attack and
defence scores. That is a combat rule wearing a messaging name, and it is why
`sourceCanSee` / `targetCanSee` look like narration flags and are not.

Ruling 6 settles the shape. A boolean cannot express three states, so combat
does not get a renamed boolean; it carries the `SightDecision` itself.
`combatContext` gains `sourceSight` and `targetSight`, and the two scoring sites
read them.

**In PR 1 this is byte-identical**: the sites apply the full penalty whenever
the verdict is not `SightFull`, which is exactly today's behaviour including for
infrared characters. **PR 2 changes the mapping**, not the plumbing, to a named
`DarknessScoreMultiplier(SightDecision) float64` returning no penalty at
`SightFull`, a reduced one at `SightShapes`, and the full one at `SightNone`.

That split is the point of doing the inversion first: once the verdict is the
primitive, a three-state balance rule is a table lookup rather than a new
predicate.

### Fourth audience

`Trio` gains `RemoteObserver`. The Trio literal guard moves to four roles and
existing literals get the field mechanically. Ranged combat's defender-room
audience, which M4b-1 named but never seated, becomes first class.

### One path

Crafting, quests, caster-only spell effects and `position_control` deliver
through `SendTrio`. Ambient stores use its observer-only form. A guard asserts
that narration categories leave only through `SendTrio`, sibling to
`TestNoRawEventsMessageOutsidePipeline`.

### Proof

Every golden byte-identical, proven by label translation, `-update` never run.
The M4b-1 rename checker is the model: a golden keyed by authored name cannot
prove a mechanical change by byte-identity alone, so the check translates and
compares data rows.

---

## PR 2: combat on the path

Player-visible. Playtested.

### `replaceDarknessMessages` is deleted

Its twelve literals go. Their text is recorded as M6 content-ledger rows **in
the same commit**, per the arc's standing rule, so authored prose is retired
rather than lost. The twelve, for the ledger:

Attacker-side (`:450`): stumble-in-darkness fumble, devastating-blow-in-the-dark
crit, "Something turns your blow aside, but you feel it land!", "Your attack is
turned aside by something!", "You strike blindly and connect!", "You swing
wildly in the darkness!"

Defender-side (`:478`): "You hear your attacker stumble!", "Something hits you
hard in the dark!", "You fend off something in the dark, but it still catches
you!", "You fend off something in the dark!", "Something strikes you in the
dark!", "You hear something whoosh past!"

### Dark lines render from the store

The composed line runs through `HideNames` with the reader's `SightDecision`,
exactly as `SendTrio` already does for every other defence. Dark fights gain
weapon and species flavour, the defence that won, and M4c's bands. Identity
reads `something` or `a figure`.

### The per-round blind notice

One short line per round, per participant who cannot see. Not per swing: a round
can carry several swings and the notice must not scale with them.

It carries its own category so verbosity can suppress it, and it is **not**
floor-protected, because unlike damage it repeats every round and a player who
has understood it should be able to turn it down.

Copy is subject to the player-copy rules: 80 column hard wrap, ESL-clear, no raw
numbers. Drafting is a task in the plan, not a decision made here.

### Infrared takes a reduced darkness penalty

A flagged balance commit of its own, disclosed in the PR body, not folded into
the narration diff.

`DarknessScoreMultiplier(SightDecision)` replaces the boolean test at
`combat_helpers.go:557,749`. A new balance knob carries the shapes case,
validated as a PAIR with `DarknessCombatPenalty`: it must sit at or above the
blind penalty and at or below 1.0, and zero is rejected, since a Go test binary
never loads `config.yaml`. An inverted pair reverts both, so a typo cannot ship
a world where seeing shapes is worse than seeing nothing.

The shipped value is the owner's to set. The plan proposes the midpoint of the
blind penalty and no penalty as a starting point and says plainly that it is a
starting point: it is a config knob, so retuning it later costs nothing.

This is the one change in M4d that alters a roll rather than a line, so it needs
its own before-and-after in the playtest: an infrared character fighting in the
dark should land and defend measurably more often than a character with no
vision at all, and still measurably less often than one in a lit room.

### `Char.Enemies` sight gate

Name becomes `an unseen foe`, HP fields are suppressed. This mirrors
`userrecord.prompt.go:530-556` exactly rather than inventing a second
convention. Without it the whole change is cosmetic for any GMCP client, and the
playtest could not honestly report that darkness works.

### Proof

1. A dark-versus-lit line matrix per seat, recorded **before** the change
   through the production path, the way M4c's band matrix was. Sabotage-proven
   before it is trusted.
2. The playtest gate: a fight and a spell in an unlit room, both participant
   seats and a witness seat, lines quoted verbatim from a telnet bridge **and** a
   GMCP bridge. The GMCP seat is what proves ruling 4 landed.

---

## Risks

**Deleting twelve authored lines is the real risk of this slice.** They name the
condition explicitly ("in the darkness", "blindly", "You hear"), which the
anonymised store line will not. If the playtest reads worse than the prose it
replaced, the fallback is defined: keep the bespoke prose, and PR 2 shrinks to
the `HideNames` seam plus the `Char.Enemies` gate. That is a retreat to a
working state, not a redesign.

**The notice could nag.** Every round in a long dark fight is a lot of lines.
Verbosity suppression is the release valve; the playtest should report how it
reads over a fight of at least several rounds rather than a single exchange.

**Sleep composition is a behaviour risk despite being "plumbing".** Composing
attention inside the predicates keeps all 31 call sites untouched, which is the
point of doing it that way, but it moves the sleep test for two of them. If
`CanSeeClearly` loses its attention test by mistake, sleeping players silently
start reading named combat tallies, and no golden covers that, because no golden
drives a sleeping observer.

So the guard is explicit and comes first: a test that a sleeping observer gets
nothing from `CanSeeClearly` and `CanSeeShapes`, and that `CanSeeSightImpairedOnly`
is unaffected by sleep. It must be proven capable of failing before the
refactor, by the standing rule that a null probe is worthless until it has been
red.

---

## Out of scope

- `Room.Info` and the GMCP contents roster: the filed GMCP slice.
- Crime witnessing, which is sight-blind today: M5.
- The twelve literals' replacements as *authored* dark-specific prose: M6, via
  the ledger rows this slice adds.
- Go narration literals outside combat: M4e.
