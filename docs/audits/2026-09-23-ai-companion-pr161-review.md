# AI companion PR #161: review, severity ranking and change requests

**Date:** 2026-09-23. **Status:** review only. Nothing merged, nothing pushed to
the PR, no comment posted.

**Subject:** [pruuk/DOGMud#161](https://github.com/pruuk/DOGMud/pull/161)
"Introduction of Aicompanion (modular)", from FinalTwist, cross-repo from
`FinalTwist:aicompanion`. 71 files, +19,645 / -18, head `83f6b8ef3`, based on
master `350cb6e88`. Adds `modules/aicompanion/` (about 30 Go files),
`internal/companionai/`, hooks, user commands and `docs/aicompanion/`.
Companion NPCs whose decisions come from the OpenAI API.

Second-round review of the work assessed in
`docs/audits/2026-09-22-ai-companion-patch-evaluation.md`, now arriving as a
pull request rather than a patch file.

Point-in-time, per the `docs/audits/` convention. Line numbers are positions in
PR head `83f6b8ef3` unless attributed to `master`.

## Verdict

1. **The merge decision is not "fix these and land it".** 09-22 recommended
   salvaging the deterministic pieces into the behaviour rework arc and not
   adopting the module. This review does not overturn that, and section
   "The decision to actually make" says why.
2. Three defects can freeze the world or remove the spend ceiling, and a green
   test suite cannot see any of them, because coverage is 28.3% and the entire
   apply path is at zero.
3. The PR's claim that the `ask.go` change is a byte-for-byte refactor does not
   hold, and that regression fires with the module switched **off**.
4. The consent blocker from 09-22 is half closed. Bystanders are protected now;
   the bonded player is still never asked and never told.
5. It is not a rewrite of `internal/goals`. It is a new layer that re-derived a
   dozen engine primitives instead of calling them, and four live defects come
   from that drift.

## What changed since the 09-22 evaluation

| 09-22 blocker | State in #161 |
|---|---|
| `Enabled: true` | Fixed. `_datafiles/config.yaml:2357` and `config.go:176` both ship `false` |
| `AutoBondExisting: true` | Fixed. Ships `false`; existing characters untouched on enable |
| Every speaker in the room recorded | Fixed. `RecordBystanderSpeech: false` gates `onCommunication` (`listeners.go:112`) and `onEmote` (`listeners.go:164`) |
| `startReflection` checked the budget but never reserved | Fixed. `reflect.go:137` now calls `tryReserveTokens` |
| `AutoBond: true`, no opt-in | **Open.** See S7 |
| Budget counters never persisted | **Open.** `budgetDay`, `tokensToday` (`aicompanion.go:128-129`) and `ownerTokens` (`models.go:496`) have no save or load path |
| `DailyTokensPerCompanion: 0` means unlimited | **Open.** `models.go:197` returns `true` when the value is at or below zero |
| `DeepModel` unset auto-selects the flagship | **Open.** `models.go:254` ships `tierDeep: {gpt-5.5, gpt-5.4, gpt-5, gpt-4.1, gpt-4o}` |
| Moderation calls unmetered | Open, unchanged, low risk only because `ModerateOutput` ships `false`, which is itself S8 |

## The decision to actually make

This document is a defect list. The owner's question is a different one, and it
should be answered before any of the fixes below matter.

**The module cannot be taken in pieces.** `git log master..HEAD` returns two
commits for all 71 files, so there is nothing to cherry-pick. Any "take the good
parts" plan means asking the contributor to re-author, not selecting commits.

**What taking it costs long term:** 19,645 lines of a volunteer's code, on a
single-operator MUD, depending on a paid third-party API, with 28.3% coverage
and the entire decision-apply path untested. The OpenAI model names in
`models.go:251-254` will rot on someone else's release schedule.

**What declining it costs:** the two new events and the `internal/companionai`
seam shape are genuinely general and would be worth having on their own. Nothing
else in the module is reachable without the module.

Three live options, and this review does not have the standing to pick for you:

- **A. Merge after the blockers.** Take the feature, own the code. Choose this
  only if you want AI companions in DOGMud as a product decision, not because
  the code is good.
- **B. Decline the module, ask for the general pieces as small PRs** (`events.Emote`,
  `events.Healed`, and the `askNpcChain` extraction done correctly), and revisit
  when the behaviour arc reaches the action layer. This is what 09-22
  recommended, and nothing in this review contradicts it.
- **C. Merge and never enable.** Explicitly the worst option: you own every line
  and get no feature, while S4 and the S12 items still reach players.

**What "do nothing" looks like, and it is defensible:** leave #161 open, take no
code, and revisit after lighting plan 6 lands. Master is not currently broken by
anything in this review; every defect here arrives with the merge.

## Findings by severity

Severity is impact times likelihood, with one override: a defect that can stop
the world outranks everything else when its fix is unconditional. S1 is ranked
first on that override, not on likelihood. Its panic reachability is unconfirmed
and that does not matter, because the missing `defer` is wrong whether or not a
panic exists today.

**All of S1, S2, S3, S5, S6, S7, S8, S10 and S11 are enabled-only.** They cannot
fire on a server that merges and never sets `Enabled: true`. Only S4 and part of
S12 reach players with the module off.

---

### S1. A panic in the tool path deadlocks the whole server

`runtime.go:1143-1145` takes the mud lock with no `defer`:

```go
util.LockMud()
answers, ok := m.answerTools(ownerId, seq, rev, sc, res.ToolCalls)
util.UnlockMud()
```

If `answerTools` panics, `UnlockMud` never runs. The panic unwinds into the
goroutine's deferred recover at `runtime.go:539`, which swallows it and then
takes the lock again at `runtime.go:547`. `mudLock` is a `sync.RWMutex` and is
not reentrant, so the goroutine blocks forever **while holding the write lock**.
Every command, combat round and save stops, and the process never exits, so no
supervisor restart fires. `MainWorker` (`world.go:733`, at the repo root) recover
deliberately re-panics on the stated grounds that a live process with a frozen
world is worse than a crash; this produces exactly that outcome.

Five lock sites in the module already use `defer`: `conversation.go:193`,
`corememory.go:152`, `models.go:424`, `reflect.go:154`, `runtime.go:576`. Two do
not: `runtime.go:1143` and the cleanup at `runtime.go:547`, which unlocks
explicitly at `:553`. The cleanup one is currently safe; `:1143` is not.

**Prescribed fix.** Wrap the call so the unlock is deferred. `answers` and `ok`
are consumed after the block, so they must be predeclared and assigned with `=`:
about five lines, not one. `answerTools` returns `([]string, bool)`
(`tools.go:62`).

```go
var answers []string
var ok bool
func() {
	util.LockMud()
	defer util.UnlockMud()
	answers, ok = m.answerTools(ownerId, seq, rev, sc, res.ToolCalls)
}()
```

**Also do, independently:** `answerTools` already bails with
`if mob == nil || owner == nil` at `tools.go:69`, but `tools.go:100,101,103,106,128`
then dereference `owner.Character`, which line 113 of the same function checks for
the other user. Extend that existing bail to
`if mob == nil || owner == nil || owner.Character == nil` rather than adding
scattered guards. This removes the only identified panic candidate; the `defer`
removes the failure mode regardless of candidates.

*Alternative considered and rejected:* making the deferred cleanup tolerate an
already-held lock. There is no clean way to do that with `sync.RWMutex`, and it
would mask lock leaks rather than prevent them.

---

### S2. Speech in any multibyte script panics

`runtime.go:1176-1194` mixes byte and rune indexing. `window` is a *string* built
from `max` runes, so `strings.LastIndex` returns a **byte** offset, but `cut` is
used to slice the **rune** array. `cut` is assigned in three branches and two of
them are byte offsets:

```go
window := string(runes[:max])
cut := -1
for _, end := range []string{`. `, `! `, `? `, `." `, `!" `, `?" `} {
	if i := strings.LastIndex(window, end); i > max/3 && i+len(end) > cut {
		cut = i + len(end)        // BYTE
	}
}
if cut < 0 {
	if i := strings.LastIndex(window, ` `); i > max/3 {
		cut = i + 1               // BYTE
	} else {
		cut = max                 // runes, correct
	}
}
piece := strings.TrimSpace(string(runes[:cut]))
runes = runes[cut:]               // panics
```

Reachable, not theoretical. `sanitizeDecision` caps a spoken line at
`maxSpeechRunes = 1200` (`decision.go:105`) while `speak` chunks at
`maxSayChunkRunes = 240` (`runtime.go:918`), and `cleanText` strips only control
characters. Running the function verbatim on multibyte text **that contains
sentence separators inside the first 240-rune window**, so the separator branch
is taken: accented Latin panics with `slice bounds out of range [:412] with
capacity 352`. Multibyte text with no separators takes the `cut = max` branch and
does not panic, so a reproduction must include `. `, `! ` or `? `.

**Prescribed fix.** Convert to rune space immediately after each `LastIndex` and
compare in rune space everywhere. The `i > max/3` guard must move too: it
compares a byte index against a rune count today, so with Cyrillic a separator at
rune 45 passes a guard meant to stop chunks shorter than 80.

```go
for _, end := range []string{`. `, `! `, `? `, `." `, `!" `, `?" `} {
	if i := strings.LastIndex(window, end); i >= 0 {
		if r := utf8.RuneCountInString(window[:i+len(end)]); r > max/3 && r > cut {
			cut = r
		}
	}
}
if cut < 0 {
	if i := strings.LastIndex(window, ` `); i >= 0 {
		if r := utf8.RuneCountInString(window[:i+1]); r > max/3 {
			cut = r
		}
	}
	if cut < 0 {
		cut = max
	}
}
```

`runtime.go` does not currently import `unicode/utf8`; add it.

**The test must assert chunk lengths, not merely the absence of a panic.** Two of
the three defects here (wrong separator chosen, guard passing too early) produce
short chunks silently and would survive a no-panic test. Cover Cyrillic, Greek
and accented Latin, with and without sentence separators, at lengths either side
of 240.

*Alternative worth taking if the contributor prefers it:* rewrite the search over
`[]rune` throughout, so the two representations never coexist. That is the
durable end state; three separate mixing sites in twenty lines is the argument
for it.

---

### S3. A panic during apply refunds the reservation twice and uncaps the budget

`applyResult` calls `settleTokens` at `runtime.go:587`, near its top. `applied =
true` is set at `runtime.go:579`, only after `applyResult` returns. Any panic in
between leaves `applied == false`, so the deferred cleanup at
`runtime.go:545-553` settles the **same** reservation again with `used = 0`.

`settleTokens` (`models.go:501-512`) decrements `m.outstanding` and does
`m.tokensToday += used - reserved`, clamped at zero, plus the same for
`ownerTokens[ownerId]`. Each panicking decision therefore credits back tokens
never spent, to three counters. After a few, `tryReserveTokens` stops refusing
anything. `m.outstanding` is also double-decremented, and `rollDay` carries it
into the next day at `aicompanion.go:297`.

S2 and S3 compound: `m.speak` is called from `applyResult` at `runtime.go:651`,
so a companion speaking Cyrillic loses its speech *and* removes the spend ceiling.

**Prescribed fix.** Settle in a `defer` inside the closure that already exists at
`runtime.go:576-580`, and delete the call from `applyResult`. Defers run LIFO, so
the settle runs under the lock and the unlock runs after it:

```go
func() {
	util.LockMud()
	defer util.UnlockMud()
	defer func() {
		applied = true
		m.settleTokens(ownerId, reserved, res.Tokens)
		if c := m.ctrls[ownerId]; c != nil && c.seq == seq {
			c.inFlight = false
			c.cancelCall = nil
		}
	}()
	m.applyResult(...)
}()
```

One settle site, the real token count on every path, no second flag, no signature
change. **`applied = true` must be the first statement in that defer**, not the
last: a panic inside `settleTokens` itself would otherwise leave it false and the
outer cleanup would settle again, which is the bug this fix exists to remove.
**The `c.seq == seq` guard must survive**: `applyResult` clears `inFlight` at
`runtime.go:610-611` and a re-dispatch bumps `c.seq` at `runtime.go:510` before
capturing `seq` at `:511`, so clearing unconditionally would clobber a newer
in-flight call.

**Two tempting one-liners, both regressions. Name them in the change request:**

- `applied = true` early inside `applyResult`. The flag also guards the `inFlight`
  and `cancelCall` reset, so a panic then pins the controller `inFlight` forever
  and that companion never speaks again. Trades a budget bug for a dead companion.
- `defer func(){ applied = true }()` inside the existing closure. Never
  double-refunds, but **leaks the whole reservation** on a pre-settle panic:
  `m.outstanding` stays inflated and `rollDay` carries it into the next day.

---

### S4. `ask.go` is not a byte-for-byte refactor, and the regression fires with the module off

`master:internal/usercommands/ask.go:152-159` exits `Ask` entirely on the
behaviour-tree path, so the broadcast at `master:...:213` is skipped:

```go
if behaviortree.TryMobBehavior(mobId, ...) {
    return true, nil                                    // leaves Ask
}
...
room.SendTextToExits(`You hear someone talking.`, true) // skipped
```

In the PR that early exit became a `return` from the extracted `askNpcChain`
(`internal/usercommands/ask.go:184`), and the caller runs the broadcast
unconditionally (`:147-149`). So every `ask` answered by a behaviour tree now
leaks "You hear someone talking." into adjacent rooms. 13 behaviour files handle
`player_ask`; `archetypes/noncombat_questgiver.yaml` returns Failure by design,
leaving 12 live mobs: Sable, Barmaid Dal, Cleric Hadwen, Warden Esk, the
Threshold Keeper, the bandit leader, and the six ferry, quay and passage agents
(9571 to 9576). No test catches it.

**Prescribed fix: give `askNpcChain` a `handled bool` return and broadcast only
when it is false.** It has exactly one early return, so this is a two-line change
and the helper stays free of room side effects.

**`askNpcChain` has two callers and both already broadcast**, so the fix must be
applied at both or the bug inverts:

```go
internal/usercommands/ask.go:147-149   // Ask
internal/usercommands/ask.go:265-266   // AskNpcForOwner
```

Moving `SendTextToExits` back inside the helper, which is the other obvious
repair, would make the companion-asks-an-NPC path broadcast **twice** to every
adjacent room unless line 266 is deleted in the same change.

**Split this into its own PR.** It is the only engine refactor rather than an
addition, the contributor predicted maintainers would want it separated
(`docs/aicompanion/upstreaming.md`, item 5), and it is the one piece that is
worth having whichever way the merge decision goes. Land it with a regression
test asserting no exit broadcast on the behaviour-tree path.

---

### S5. Any player can drive unbounded model calls through someone else's companion

`internal/usercommands/ask.go:132` routes `ask` to the module before any
ownership check, and `handleAsk` (`listeners.go:299-330`) accepts it from anyone.
The owner id appears only at `:324` to decide `interruptErrand` and to set
`FromOwner`; it never refuses. The only throttle is per-controller: `c.inFlight`
plus `MinSecondsBetweenCalls: 2` (`runtime.go:333`). There is no per-caller limit.

Stand in a room with any online player's companion and loop `ask Mara hi` every
2.5 seconds. `DailyTokensPerCompanion` ships `0`, meaning unlimited, so only the
global 2,000,000 pool applies and exhausting it silences every companion on the
server for the rest of the UTC day. Closing a conversation also fires a fast-tier
summary call (`conversation.go:146`), so the burn exceeds the ask count.

**How fast the cap falls is an arithmetic question this review did not close.**
It needs the per-call prompt size, which varies with memories and scene. The
shape to compute before deciding severity: `2,000,000 / (tokens per call) / 1440
calls per hour`. Do the same in money, at your current OpenAI rate card, for the
tier `models.go:254` actually selects. If the honest answer is "the cap is
reached in an hour", this is severe; if it is "a day", it is an annoyance.

**Fix options**

| Option | Pros | Cons |
|---|---|---|
| A. Require `userId == c.ownerUserId` before dispatching | Closes the request flood and the forced egress outright | Strangers can no longer talk to companions at all, removing a stated design goal |
| **B. A separate cooldown and daily cap for non-owner askers** | Preserves the design, bounds the rate to a number the operator picks. The throttle half needs no new state: `characters.Character` already carries `Cooldowns` with `TryCooldown(tag, period)` (`internal/characters/cooldowns.go:93`), persisted with the character | **Does not close the forced egress.** A stranger still pushes the owner's memories, facts, pack and quest state to OpenAI, only slower. Only the daily cap needs new state |
| C. Non-owner asks push a stimulus but never trigger a dispatch of their own | Zero added cost, preserves overhearing | Reads as unresponsive to strangers, which is close to A's outcome without A's simplicity |

**Recommend B**, with A as the interim, and ship a non-zero
`DailyTokensPerCompanion` regardless so one companion's exhaustion is not the
whole server's.

**Implementation hazard.** Do not implement the refusal by returning `false` from
`handleAsk`. `companionai.RouteAsk` returning false lets `Ask` fall through to
`askNpcChain`, so the companion answers as a generic NPC instead of refusing. The
check belongs at the dispatch decision.

---

### S6. A world event in the same batch as a stranger's speech unlocks the owner-only verbs

`ownerDrivenOnly` (`actions.go:143`) covers `give`, `drop`, `put`, `sell`, `buy`,
`loot`, `take_from`, `get`. `ownerPrompted` (`actions.go:148-162`) gates them and
returns on the **first** owner-flagged stimulus without ever consulting
`strangerSpoke`.

The root cause is that `FromOwner: true` does not mean the owner acted. There are
**17** such push sites, and they include world events and even third-party
actions: `quiet` (`runtime.go:274`), `fight_over` (`combat.go:760`), `ailing`,
`trouble`, `remembering` (`autonomy.go:53,120,345`), `party` (`runtime.go:1027`),
`recovered` (`runtime.go:234`), `first_meeting`, `session_start`, `farewell`
(`runtime.go:123,130,165`), four romance pushes, `gift` (`autonomy.go:521`), and
`witnessed` (`listeners.go:418`), which carries a **third party's name** as
`Speaker`.

Batching is normal: `controller.push` keeps the last 6 (`aicompanion.go:109`) and
they drain only when `dispatch` runs. Wait for a fight near the companion to end,
speak to it in that window, and `give`, `drop` and `sell` are live. Items gifted
by the owner are held back by `canPartWithItem` (`inventory.go:147`); anything
bought, looted, foraged or starting kit is not. The test at
`aicompanion_test.go:1697` covers only a pure-stranger batch.

**Prescribed fix: repair `ownerPrompted` itself, in both of its faults.** Compute
`strangerSpoke` across the whole slice before returning, **and** require
`s.FromOwner && (s.Kind == "heard" || s.Kind == "asked")` for the positive case,
so world events stop counting as owner intent. Keep the existing autonomy case,
where no stranger spoke at all, intact.

**Set `strangerSpoke` only for `!s.FromOwner`.** Today the `if s.FromOwner {
return true }` short-circuit means owner-flagged stimuli never reach the `switch`.
Remove the short-circuit and mark the whole slice unqualified, and the owner's own
actions start blocking their own requests: `listeners.go:178` pushes `emote` and
`listeners.go:234` pushes `gift` with `FromOwner: fromOwner`, and both Kinds are
in the stranger list, so an owner handing the companion an item would be refused
`give`, `drop` and `put`. Of the 17 `FromOwner: true` sites only `autonomy.go:521`
carries a Kind in that list, and it is owner-attributed, so the qualification is
safe.

*Do not substitute the existing `ownerAskedNow` for the gate*, which is the
obvious reuse and is wrong three ways:

- It has a second branch, `if s.Kind == "arrived" && s.Authorized`
  (`actions.go:76`), which does not require `FromOwner`. `travel.go:169` pushes
  `arrived` without `FromOwner`, so for the batch `[arrived(Authorized),
  heard(stranger)]` today's gate refuses and `ownerAskedNow` would **allow**.
  Strictly looser than the bug being fixed.
- It would make the `take_freely` loot arrangement dead. The gate at
  `actions.go:167` runs before `lootAllowedByArrangement` (`actions.go:805`),
  which already uses `ownerAskedNow` for `ask_first` and lets `take_freely`
  through; gating `loot`/`get`/`take_from` on it collapses the two rules into one
  and kills the autonomous salvage path at `autonomy.go:274`.
- It would make `purchaseCheck`'s spend guards unreachable. `actions.go:384`
  already passes `ownerAskedNow(stims)` in, and `economy.go:212` short-circuits
  on `meetsNeed || ownerAsked`, so gating `buy` on the same predicate makes the
  argument constant true and the `purse.Reserve` and `largeShare()` branches dead
  code.

**Also do:** restrict `give`'s recipient to the owner. It is the verb that
actually moves an item to an attacker, and it is defence in depth on a gate that
has now been wrong twice.

---

### S7. No opt-in, and no disclosure anywhere a player will see

09-22 called consent *the* blocker. It is ranked seventh here only because the
module ships disabled and cannot enable itself. In exposure terms it is unchanged:
it covers 100% of players created after the first enable.

`AutoBond: true` ships on. `onCharacterCreated` (`meeting.go:73`) queues a meeting
for every brand-new character and `considerMeeting` bonds it a few rounds after it
reaches a real room. With `RespondWhenAlone: true` and `isAddressed`
(`prompt.go:596-600`), once bonded, **every line the owner says while alone in a
room** is sent to OpenAI. The `Declined` record (`meeting.go:30`) is only written
after a bond exists, and the sentence that ends it is itself shipped as the
stimulus that drives the leave decision.

`AutoBondExisting: false` bounds the *backfill*, not the exposure: nobody who
already exists is bonded on enable, but every character created afterwards is.

Retention is undisclosed too. `Mind.RecentLines` (`mind.go:48`) holds speaker
names and exact text, written to
`_datafiles/plugin-data/aicompanion-v0-1-0/mind-<userId>-<mobId>.plugin.dat` with
three rotating backups, bounded by count and never by age. `aicompanion prompt
<char>` (`commands.go:398`) dumps the last full request to any admin. The three
player-facing help templates mention none of this.

**Fix options**

| Option | Pros | Cons |
|---|---|---|
| A. Ship `AutoBond: false` | One line, and the right gate before any public enable | Nobody discovers the feature, and it adds no disclosure for players who do opt in, so the privacy gap survives for exactly the people it applies to |
| **B. Make the meeting a consent moment: the companion introduces itself, states plainly that talking to it sends what is said to OpenAI and is kept on the server, and the player accepts or declines. Nothing sent before accept** | Real consent at the moment the player is thinking about it, and it preserves discovery. **Cheaper than it looks: two of the three pieces exist.** `meeting.go:150-154` already builds the greeting from `Profile.Meeting` with a hardcoded fallback and is not a model call; `requestLeave` (`meeting.go:206-215`) already implements the accept-within-N-minutes shape with `c.leaveAskedAt` and `LeaveConfirmSeconds` | Only the pending-consent persistence is genuinely new; `pendingMeet` is an in-memory map today |
| C. Opt-in by command only (`companion-bond`) | Simplest honest consent | Discovery depends entirely on a help file, so in practice this is A with extra steps |

**Recommend B**, with A as the hard gate before we enable on a server anyone else
plays on. Either way, add a player-facing help entry covering the third-party API
and the on-disk retention, and say in it that admins can read the transcript.

---

### S8. Model speech reaches the room unmoderated by default

`ModerateOutput` ships `false` (`config.go:230`). The only content filter is
`characterBreakers` (`decision.go:265`), which catches AI-reveal phrases such as
`language model` and `chatgpt`, not abuse. `sayto` is not in `ownerDrivenOnly`,
so naming the companion is enough and no batching trick is needed.

**The verifiable claim, which is sufficient for the finding:** no content control
is enabled by default, and `moderate` fails open at `openai.go:300`. Whether a
model actually complies with a puppeting injection is not something this review
measured, and the finding does not rest on it.

**Prescribed fix: default `ModerateOutput: true` whenever the module is enabled.**
`ModerationModel` defaults to `omni-moderation-latest` (`config.go:448-449`), so
this costs latency and not money.

**Separately, the fail-open behaviour cannot be changed as a policy switch.**
`moderate` returns bare `nil` for the empty case, for a transport error and for a
non-200 (`openai.go:300-330`), so nothing downstream can tell a clean pass from an
outage. Fixing it means changing the signature to `([]bool, error)`. Say that in
the change request rather than describing it as a config choice.

---

### S9. Reserving gear makes a companion flee every fight, permanently

`combat.go:346` computes self health from `HealthMax.Value`.
`internal/behaviortree/actions_archer.go:156-170` already solved this and wrote
the warning:

> `EffectivePoolMax`, not `HealthMax.Value`. Every caller is self-side, and a
> companion carrying reserving gear would otherwise read as permanently wounded
> at a completely full pool: at the U7b ceiling, permanently at 34%.

`fleeThreshold("badly_hurt")` is 50 (`combat.go:99`), so with any reserving item
equipped the companion sits permanently past its flee threshold. The module can
buy and equip gear, so this is reachable in normal play.

**There are 15 `HealthMax.Value` reads in the module, not one.** Patching only
`combat.go:346` leaves `combat.go:358` twelve lines below it feeding the model the
same permanently-"badly hurt" band, plus `combat.go:206,222,224,249`,
`actions.go:486,491`, `listeners.go:376`, `perception.go:134,169`,
`runtime.go:91`, `tools.go:203`. The engine precedent is also not self-only:
`internal/behaviortree/conditions_party.go:45-61` uses `EffectivePoolMax` for
other party members, so the owner-side reads at `combat.go:350,361` are in scope.

**Prescribed fix: change the module's own helpers to take the character, not two
ints.** `healthPct(char *characters.Character) int` and
`healthWords(char *characters.Character) string`, calling
`char.EffectivePoolMax(characters.PoolHealth)` inside. The compiler then
enumerates the call sites, which is the same trick `dogmud-refactoring`
prescribes for removals.

Three things the contributor needs to know before starting:

- **It enumerates 14 of the 15, not all of them.** `runtime.go:91` is
  `if half := mob.Character.HealthMax.Value / 2`, a direct read with no helper
  call, so it survives the signature change silently. Grep for it.
- **`healthWords` lives in `perception.go:24`, which does not import
  `internal/characters`.** That import has to be added; `combat.go`, which holds
  `healthPct`, already has it.
- **`tools.go:203` needs no change**: its `ch` is already a
  `*characters.Character` (`describePerson`, `tools.go:200`).

---

### S10. The three cost controls that do not bind

All three were filed on 09-22 and are unchanged.

- **The daily budget does not survive a restart.** `budgetDay` and `tokensToday`
  (`aicompanion.go:128-129`) are plain fields with no save or load path. **So is
  `ownerTokens` (`models.go:496`)**, which is the counter the per-companion cap
  uses, and therefore the counter S5's mitigation depends on. Any persistence fix
  must name all three.
- **`DailyTokensPerCompanion: 0` means unlimited** (`models.go:197`), so the
  advertised per-companion cap is off as shipped.
- **`DeepModel: ""` auto-selects the flagship** (`models.go:254`), on every logout
  reflection, at a price the operator never chose.

**Prescribed fix.** Persist all three counters through the existing
`plug.WriteStruct` path (`plugins.go:427`), which the module already uses for
bonds (`meeting.go:51`) and minds (`mind.go:263,326`), keyed by the UTC day so a
stale day resets cleanly. Ship a non-zero `DailyTokensPerCompanion` and an
explicit `DeepModel`; those two are config edits, not code.

*Rejected:* deriving the day's spend from the persisted per-mind
`TokensLifetime`. That is a lifetime counter and answers a different question;
making it daily is more new state, not less.

**Note for the correctness of the argument:** neither S1 nor S2 crashes the
process, so neither resets these counters by restart. S1 deadlocks without
exiting and S2's panic is swallowed by the goroutine's recover. The unbounded
path is S3, which walks the counters down with no restart at all.

---

### S11. Opinions are written in two places and never decay

`internal/opinions` already stores a per-(mob, user) score with `Get`/`Bump`, a
plus or minus 100 range, `TierOf` cut points and a config-driven decay half-life.
The module keeps its own three-axis `Opinion` (`opinion.go:13-17`), re-derives the
same cut points at `opinion.go:210-216`, and has no decay path. Meanwhile
`internal/actions/aggression.go:107` still calls `opinions.Bump` for companion
mobs with no exclusion, as does the gift seeder, so two numbers move on one event
and `admin.opinion` and `planners/befriend.go` read the one that does not drive
behaviour.

**The obvious fix does not work as stated.** Projecting the three axes into
`opinions.Set` is callable (`internal/opinions/opinions.go:137`) but `Set` is
keyed by **mob template id**, and companion profiles are one-to-one with a
template (`profile.go:254-259` rejects a duplicate `mob_id`). So every player
bonded to the same profile shares one `(template, user)` slot, and owner A's
companion's feelings about a stranger overwrite owner B's. This is the same
template-keying objection this document accepts for `internal/goals` two sections
below. `Set` also calls `saveToDisk` synchronously (`opinions.go:152`), putting a
disk write on the game loop per update.

**What to do instead, in two separable parts:**

1. **Suppress the engine's own bumps for bonded companions.** This half is clean
   and worth doing now: `characters.CompanionBonded` exists
   (`internal/characters/companions.go:56`) and `internal/hooks` already tests
   `SourceType == characters.CompanionBonded` in five places. Guard
   `aggression.go:107` and the gift seeder (`internal/seeders/gift_to_opinion_boost.go:68`)
   the same way, so only one store moves. Note the shape: `SourceType` is a field
   on the owner's `characters.CompanionInfo`, not on the mob, so the guard has to
   resolve the instance back to a bonded record the way
   `internal/hooks/companion_bonded.go:108` does. It is a lookup, not a field test.
2. **Leave the module's store authoritative** and accept that `admin.opinion`
   does not describe a bonded companion, documenting that in
   `modules/aicompanion/context.md`. Unifying the two properly needs an
   instance-keyed opinion store, which `internal/opinions` does not have and which
   is its own piece of work.

**Add a decay path either way.** A companion that never forgives, where every
other NPC does, is a design decision nobody made.

---

### S12. The module is less absent than advertised when disabled, and this is fixable

`onLoad` genuinely stops dead at `aicompanion.go:191`, before profiles, bonds,
listeners, seams and model probing, and `onSave` returns early. That part of the
claim holds. These do not:

- **Commands are registered in `init()`**, before any config is read
  (`aicompanion.go:147-182`), and `plugins.Load` copies them into the registry at
  `internal/plugins/plugins.go:557` while `onLoad` only runs at `:580`. With the
  module off: `modules/gmcp/gmcp.Commands.go:174` advertises five dead commands to
  every web client; `help companion-part` and `help companion-unstick` render full
  help for a feature that is not there; and typing one **in combat** answers
  "You can't do that while fighting!" rather than "not recognized", because
  `Plugin.AddUserCommand` (`plugins.go:269-280`) has no `allowedInCombat`
  parameter and so always leaves it false.
- **`companion-ask` is registered twice** (`aicompanion.go:171-172`). Both writes
  hit the same map key, so this is redundant rather than harmful; delete one line.
- **About 60 `Modules.aicompanion.*` keys** merge into the live config tree from
  `files/data-overlays/config.yaml` at plugin load regardless of the toggle. This
  is structurally unavoidable, since it is how `Enabled` is read, but the PR does
  not mention it.
- **`internal/actions/divergences.go:156` and `:206` carry comments** saying
  "Registered only while that module is enabled", which the code does not do.

**Prescribed fix, and it needs no engine change.**
`usercommands.RegisterCommand(command, handler, disabledWhenDowned, allowedInCombat, isAdminOnly)`
is exported at `internal/usercommands/usercommands.go:580` and writes the live
map directly; `plugins.Load` merely calls it. So: keep the admin `aicompanion`
command in `init()` where its comment requires, and move the five player commands
to a `RegisterCommand` loop in `onLoad`, behind `m.cfg.Enabled`.

That fixes two of the three symptoms. `buildCommandsList`
(`gmcp.Commands.go:174`) calls `GetCommandRegistry()` on demand rather than at
startup, so a command registered only when enabled never reaches the client, and
`RegisterCommand` takes the `allowedInCombat` flag the plugin API cannot express,
which fixes the in-combat answer.

**It does not fix the help files, and nothing in the command registry will.**
`GetHelpContents` resolves a topic through `keywords.TryHelpAlias` and
`templates.Process("help/"+name)` (`internal/usercommands/help.go:148-204`) and
never reads `userCommands`. The templates are mounted by
`module.plug.AttachFileSystem(files)` in `init()`, unconditionally, so
`help companion-part` renders exactly as it does today. Treat that as its own
small item.

Fix the duplicate `companion-ask` line in the same change. Note that the fix
moves only the five **user** commands: the four mob commands stay in `init()`, so
`divergences.go:206` stays false unless they move too.

---

### S13. Smaller items

| Item | Evidence | Fix |
|---|---|---|
| The `hamstring` comment is false, and the move is offered in two places | `decision.go:46-48` calls it a move "a two-armed, two-legged companion can use"; `internal/actions/command_readiness.go:117-127` requires **no** hands plus fangs or claws | Fix the comment, not the enum. `Profile.MobId` (`profile.go:25`) can point at any template, so a fanged, handless companion is legitimately authorable; only the shipped `mara.yaml` is humanoid. Note that `prompt.go:93` lists the same moves to the model in prose, so the enum is not the only site |
| `events.Emote.MobInstanceId` is never set | Both emit sites (`usercommands/emote.go:31,56`) set only `UserId`, `RoomId`, `Text` | A mob emote fires no event. Populate it or drop the field before it ships as API |
| `events.Healed` covers mob targets only, and its name overstates it | Queued at `internal/hooks/spell_resolution.go:926` in the `case "heal"` arm; the player-target branch emits nothing | `applyMobEffect_heal` restores **zero** health: it applies condition 120 `Regenerating`, a `regen_mult` multiplier on natural regen for `max(duration/2, 6)` rounds, and returns 0. Cast time is a defensible moment to fire, but the doc comment should say it fires on the cast. It is also queued before `AddConditionMagnitude`, whose error is discarded with `_ =`, so it can fire when nothing was applied |
| `pool_mutation_guard_test.go:59` gains a whole-file exemption | The guard's header warns to prefer a file key over a directory key, and they used a file key | Currently 5 writes in one function, but any pool write later added to `companion_bonded.go` is exempt for free. Narrow it to the function if the guard supports that |
| `commands.go:533` `topicPresent` returns true when `words == 0` | An owner topic made only of words shorter than 3 characters grants unconstrained NPC-question authority for 60s | Owner-only, narrow |

`condition_apply_path_guard_test.go` is a mechanical line-number re-key, and
`lookup_viewer_guard_test.go:83` adds a required-coverage census row rather than
an exemption. Neither is a weakened guard.

## Is it a rewrite of what we already have?

No, and the import list settles it. `modules/aicompanion/` imports 24 engine
packages and **none** of `internal/goals`, `opinions`, `behaviortree`, `mapper`,
`llm`, `relationships` or `planners`. All seven exist. It reuses the engine's
*mechanics* correctly, acting only through `mob.Command` and calling
`messaging.ParticipantSight`, `actions.CommandIsReady`, `actions.Consider`,
`crafting.HasIngredients` and `factions.GetRep`. It re-derived the engine's
*decision primitives*.

Several divergences are defensible and should not be argued away:
`internal/goals` is keyed per mob **template**, so two owners of one profile would
share a goal file; `internal/opinions` is a single scalar where the prompt needs
three axes, and is template-keyed for the same reason;
`internal/relationships` is structurally mob-to-mob; and `mapper.GetPath`
searches the **authored** world graph, which is the omniscience this design
rejects in favour of a walked-only map.

What is not defensible is that the copies drifted. S9 and S11 are both
duplication defects, and in S9's case the engine had already written a comment
naming this precise actor class. `internal/llm` also already implements the
background-call-plus-locked-callback pattern, with a response cache the module
lacks.

## Merge mechanics, with the calls made

- **No CI has ever run** (`gh pr checks 161 --repo pruuk/DOGMud`). Locally
  `go build ./...` and `go vet ./...` are clean and all 85 tests pass. **Call:**
  if the merge goes ahead, run `go test ./...` on the merge commit locally rather
  than pushing the head to origin for CI, because pushing a branch to the fork
  advertises it on upstream's PR page.
- **Coverage is 28.3%** with the apply path at zero. **Call:** S1, S2 and S3 each
  land with a test that can fail, or there is no reason to believe the next one
  is caught either. S2's test must assert chunk lengths, not just absence of panic.
- **`_datafiles/config.yaml` gains 51 lines** and carries the skip-worktree bit in
  the main checkout (`git ls-files -v` returns `S`). **Call:** build any commit
  touching it from the `git show HEAD:` blob, never from disk.
- **`perception.go:304` reads `messaging.ParticipantSight`**, which the graded
  lighting arc is reshaping across plans 3 to 6. **Call:** this is the strongest
  structural argument for deferring the merge until plan 6 lands. 19.6k lines
  reading a sight API that is mid-rewrite is a rebase somebody pays for twice.

## What is not wrong with it

Worth stating plainly, because the change request is long and the work is good.

- **Model output never becomes a command string.** `actionVerbs` is a closed enum
  (`decision.go:133`) enforced in the strict JSON schema and again in
  `sanitizeDecision`. Refs resolve against the captured scene and are rechecked
  against the live world. Commands are assembled server-side from item UUIDs and
  mob instance ids, with `;` stripped and ANSI escaped. Nothing downstream trusts
  the decider, and that property is what would make any remote decider safe.
- **SSRF is blocked.** `config.go:488` requires https and an `openai.com` or
  `azure.com` host unless `AllowCustomEndpoint` is set, which ships off.
  `url.Hostname()` defeats the `https://api.openai.com@evil.com/v1` trick, and a
  rejected URL is logged with the official endpoint substituted.
- **API key handling is clean.** Read from `OPENAI_API_KEY` first, never
  marshalled into a Mind, never sent to a player, never logged. Ships as `""`.
- **Goroutine discipline is careful, with S1 and S3 as the exceptions.** All five
  goroutines build their payload on the game loop, capture only scalars and
  copies, and re-enter the world under the mud lock. No lock is held across a
  network call, so a hung OpenAI call cannot stall the world. Timeouts floored at
  5s, one retry, response body capped at 1 MiB, breaker after 5 consecutive
  errors.
- **Mind persistence is sound.** Writes route through the durable autosave queue,
  only dirty minds are saved, and every collection is bounded. The *budget*
  counters are a separate matter and are not persisted at all, which is S10.
- **`internal/companionai` is a genuine dependency-inversion seam**, and
  `CompanionBonded` extends the existing enum rather than forking it.
- **No leaked personal data.** A sweep of the diff for key-shaped strings,
  emails, bearer tokens, home paths, private IPs and personal handles returns
  nothing; commits are authored as GitHub's noreply alias; the docs carry no play
  transcripts and no real usernames.
- **Both `context.md` files are present**, new files are listed in
  `docs/README.md`, and the house prose style is followed.

## If the answer is "fix it", what to ask for and in what order

**Split by who has to decide, not only by merge gate.** A volunteer can start the
left column today; the right column needs a ruling first.

| Do now, no decision needed | Needs a ruling from the owner first |
|---|---|
| S1 (`defer` plus the `answerTools` bail), S2 (rune-space conversion plus length-asserting tests), S4 (`handled bool`, both call sites, own PR), S9 (helpers take the character), S12 (`RegisterCommand` in `onLoad`, duplicate line, the `:156` comment), S10's three config values, S13 | S5 (what is the non-owner cap, and is A or B wanted?), S6 (confirm the `give`-to-owner restriction), S7 (consent prompt, or opt-in command?), S10's persistence shape, S11 (does resentment decay, and by what rule?) |

**Then by gate.** The module ships off in two places, `config.yaml:2357` and the
Go default at `config.go:176`, and there is no runtime reload anywhere in the
module, so enabling takes a deliberate config edit plus a restart. A config
desync cannot enable it. That double lock is what makes the second tier safe to
defer.

0. **Independent of everything: S4, as its own PR**, since it is the only change
   that reaches players with the module off and the only one worth having
   whichever way the merge decision goes.
1. **Before merge:** the S12 items visible with the module off.
2. **Before the first `Enabled: true` anywhere, including a test server with one
   other player on it:** S1, S2, S3, S5, S6, then S7, S8, S10 in that order.
3. **Merge-window quality:** S9, S11.
4. **Follow-up:** S13.

## Claims re-verified after the first draft, and the method

The first draft of this document got six remedies wrong, which is why this
section records method rather than asserting blanket verification. Items below
were read from the PR head or from `master` at review time.

| Claim | Verified by |
|---|---|
| `ask.go` reachability change | Read `master:...:152-159,213` and PR `:147-149,184` side by side |
| `askNpcChain` has two callers, both broadcasting | `grep -n askNpcChain internal/usercommands/ask.go` returns `:147` and `:265`; read `:265-266` |
| 12 live `player_ask` behaviour mobs | `grep -rln player_ask _datafiles/world/dogmud/behaviors/` returns 13, minus the archetype that returns Failure |
| Byte/rune mix, all three `cut` branches | Read `runtime.go:1176-1194`; panic offsets reproduced by running the function |
| Missing `defer` and the re-lock | Read `runtime.go:1138-1150` and `:536-556` |
| Double settle, and `settleTokens` rolling the day itself | Read `runtime.go:572-590` and `models.go:501-512` |
| `handleAsk` has no owner check | Read `listeners.go:299-330` |
| `ownerPrompted` early return; 17 `FromOwner: true` sites | Read `actions.go:143,148-162`; `grep -rn 'FromOwner: *true' modules/aicompanion/*.go` |
| Owner gift and emote carry `FromOwner`, so `strangerSpoke` needs qualifying | Read `listeners.go:178,234` |
| `answerTools` returns `([]string, bool)`; bail at `tools.go:69` | Read `tools.go:62-72` |
| `healthWords` is in `perception.go`, which lacks the `characters` import | Read `perception.go:1-24`; `runtime.go:91` is a direct read, not a helper call |
| `help` does not read the command registry | `grep 'GetCommandRegistry\|userCommands' internal/usercommands/help.go` returns nothing |
| `ownerAskedNow`'s second branch | Read `actions.go:71-80` and `travel.go:32,62,169` |
| Gating on `ownerAskedNow` would kill `take_freely` and `purchaseCheck` | Read `actions.go:167,805-815`, `economy.go:205-215`, `actions.go:384` |
| 15 `HealthMax.Value` sites | `grep -rn 'HealthMax.Value' modules/aicompanion/*.go` |
| `healthPct` versus `EffectivePoolMax` | Read `combat.go:88-99,346` and `internal/behaviortree/actions_archer.go:150-172` |
| `usercommands.RegisterCommand` is exported and takes `allowedInCombat` | Read `internal/usercommands/usercommands.go:580-587`; `plugins.Load` at `plugins.go:557` calls it |
| `opinions.Set` is template-keyed | Read `internal/opinions/opinions.go:137-145` |
| `characters.TryCooldown` exists | Read `internal/characters/cooldowns.go:93` |
| Seven engine packages not imported | `grep` per package over `modules/aicompanion/*.go`; existence confirmed with `ls -d` |
| Heal is a regen multiplier, not flat | Read `spell_resolution.go:843-873`, `conditions/120-regenerating.yaml`, `NewRound_AutoHeal.go:308` |
| `Enabled: false` in Go as well as YAML; no runtime reload | Read `config.go:176`; grep for reload hooks returns nothing |
| Coverage 28.3%; 85 tests pass; build and vet clean | `go test -cover ./modules/aicompanion/`, `go build ./...`, `go vet ./...` |
| `opinions.Bump` still fires for companions | Read `internal/actions/aggression.go:107` |
| `events.Emote.MobInstanceId` never set | `grep 'events.Emote{'` across `internal/` and `modules/` |
| `hamstring` gate | Read `internal/actions/command_readiness.go:117-127` |
| Two commits for 71 files | `git log master..HEAD` |
| config.yaml skip-worktree | `git ls-files -v _datafiles/config.yaml` in the main checkout returns `S` |
| No CI run | `gh pr checks 161 --repo pruuk/DOGMud` |

**Not verified, and flagged in place:** whether a panic is actually reachable
inside `answerTools` (S1); how reliably a model complies with a puppeting
injection (S8); and the per-call token cost that sets how fast S5's attack
exhausts the daily budget. `-race` was not run, for want of a C toolchain on this
machine; no test in the module starts a goroutine, so it would have had nothing
to detect.
