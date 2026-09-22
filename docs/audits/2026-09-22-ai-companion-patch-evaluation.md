# AI companion patch: evaluation and salvage list

**Date:** 2026-09-22. **Status:** evaluation only. Nothing here is adopted, and
no code from the patch has been merged.

**Subject:** `DOGMud_AI_Companion beta 10.patch`, a third-party contribution
from the GoMud Discord. 16,587 lines, 617 KB. Adds `modules/aicompanion/`
(about 25 Go files), `internal/companionai/`, hooks, user commands, and
`docs/aicompanion/`. Companions are NPCs whose decisions come from the OpenAI
API.

Point-in-time, per the `docs/audits/` convention. Line numbers below are
positions in the patch file, not in any checked-out tree.

## Verdict in three lines

1. The engineering is careful and the engine integration is small and additive.
   The shipped **defaults** are what make it unsafe, not the design.
2. The blocker is **consent, not cost**: every player is auto-bonded and their
   speech leaves the server before they can decline.
3. Most of what is valuable in it **never needed a model at all**, and that part
   folds into the behaviour rework arc.

## Why not to merge as shipped

| Finding | Consequence | Patch ref |
|---|---|---|
| `AutoBond: true` and `AutoBondExisting: true` | Every character, new and existing, is silently bonded within about 3 rounds. No pre-bond prompt exists. The only decline path runs after a bond already exists | 8950, 8952 |
| Zero matches for "consent", "opt-out", "redact", "privacy" across the whole diff | No opt-out, no redaction, anywhere | whole diff |
| `onCommunication` records every speaker in the room | A third party who never bonded a companion can have their speech sent to OpenAI on the owner's next request | 9592 |
| Daily budget counters are process memory only, never persisted or reloaded | Every restart resets the "daily" budget to zero. On a crash loop there is no ceiling. `Mind.TokensLifetime` is persisted but never read by any budget check | 3883-3893, 4022, 10546 |
| `DailyTokensPerCompanion` ships `0`, and any value at or below zero means unlimited | The advertised per-companion cap is off by default. Only one global 2,000,000 token per day pool applies | 7573, 8911, 11231 |
| Deep tier auto-selects newest flagship, not cheapest | `modelPreferences[tierDeep]` is the flagship line and is used for the once-per-logout reflection whenever `DeepModel` is unset, which is the default. The operator never chose that model or its price | 11286-11290, 11351 |
| `startReflection` checks the budget but never reserves against it | Unlike `dispatch`, which reserves correctly. Concurrent logouts all pass the same stale check | 13829, 13884-13896 |
| Moderation calls are never metered | Token usage is not added to `tokensToday` or `ownerTokens` at all. Low real risk only because `ModerateOutput` ships false | 14471, 8915 |
| Tool rounds cost up to 3 HTTP calls per dispatch, reserved as one | `ToolRounds: 2` default. `estimateTokens` also only sums message content, ignoring tool specs and schema | 14965, 11460 |
| No runtime config reload | Flipping `Enabled` needs a process restart. `aicompanion pause` is per-companion only, so there is no live server-wide stop | onLoad, loadConfig |

Spec self-contradiction worth knowing: lines 1346 and 1425 say the module is
off by default; lines 1719, 1844 and the sample config at 1811 say on. The code
settles it as **on** (`Enabled: true` at 7530 and 4160), reading an ambient
`OPENAI_API_KEY` via `os.Getenv` at 4015.

Five changes would make it safe merely to evaluate: default both auto-bond
flags off with a real opt-in, persist the budget counters, reserve before the
reflection call, pin `DeepModel` explicitly, ship a nonzero per-companion cap.

### What is NOT wrong with it

Blast radius into existing code is genuinely small and additive: two new event
types, a queued `Healed` event inside the existing heal branch, an emote event,
one early return in `MobIdle_HandleIdleMobs` that only intercepts instances the
module owns, a standard `all-modules.go` registration, and two mechanical
guard-test re-keys. `aicompanion_test.go` is about 64 real test functions
covering the breaker, budget mechanics and token reservation settling, and both
`context.md` files are present. Every HTTP call is timeout bounded and retries
at most once. This person read our conventions and followed them.

## The salvage list, for the behaviour rework arc

**The reframe that matters: `internal/goals` is already a utility AI engine.**
`effectiveScore = priority x archetypeWeight x contextMod`
(`select.go:150-159`), with switching hysteresis (`scoreGap < margin` plus held
rounds, `select.go:132-144`), per-goal-type registered context scorers,
dormant-goal pruning, per-mob persistence, pluggable lookups, and a
`goal scores` admin diagnostic that explains why a goal won. The deterministic
decider is built. The arc's job is to extend that scoring seam down from goal
level to action level.

| Piece | Needs a model? | Value to the arc |
|---|---|---|
| `combat.go` `reflex()` | No, the code comment says so | **Highest.** Ordered priority list (flee, self-heal, protect, special move, chosen target, ammo-out, aid) run every round, never waiting on a decider. This is the answer to the targeting race |
| `combat.go` `CombatProposal` (the plan) | Replaceable | Slow layer sets stance, target, style, flee threshold. Score it with the existing goals formula |
| `runtime.go` `dispatch` and `applyResult` | No | Snapshot under the mud lock, decide off-lock, re-validate a `worldRev` counter, discard the whole decision if the world moved. Only needed when a decider is slow |
| `worldmap.go` | No, Dijkstra plus BFS | Mob knowledge earned by walking, deliberately NOT `internal/mapper` (which searches the whole world). Fallible mobs instead of omniscient ones |
| `travel.go` | No, state machine | One `go` per round, replans on failed or guessed exits, timeout, key-try-once |
| `scene.go` | No, local scoring | Bounded typed observation: interest match, novelty, hostility, value, ranked, with refs assigned |
| `decision.go` | No, it is a struct | The bounded action vocabulary plus `sanitizeDecision`. See below |
| `memory.go` | No | Keyword plus exponential recency decay. The spec mentions embeddings (line 384) but the implementation plan admits they were never built (line 1501), so what shipped is portable as is |
| `autonomy.go` `pastime` | No, weighted random | Idle flavour at zero cost |
| `inventory.go` `Supply` | No | Declarative "keep N of X", feeds goal predicates |
| `opinion.go` | Partly | Model proposes a delta, code clamps it into a per-stimulus envelope with diminishing returns. Swap the proposer for a stimulus-to-delta table and it is deterministic with no structural change |
| `tools.go` read-only query set | Delivery only | Useful inventory of what any decider needs to ask: `look_closer`, `size_up`, `check_wares`, `recall`, `find_place` |

**Steal outright, five-line principle:** `GoalCheck`. A goal closes only when
code verifies live world state, never because the decider said so.

**The action vocabulary** (closed, every string validated against a Go slice):
`none, look_at, consider, get, drop, give, show, equip, remove, eat, drink,
forage, search, go_to, explore, find_place, put, sayto, browse, buy, sell,
loot, take_from`. Combat plan axes are separate closed enums: stance
(`fight, protect, hold_back, flee`), flee_at, style (`melee, ranged`), move
(`taunt, bash, kick, trip, grapple, hamstring, rally, warcry`). Speech is
bounded to at most 3 lines of `say` or `emote` per decision.

**Worth reading even with the code set aside:** the spec's R1 to R7 ground
rules. "The decider's output is data, never a command" and "mechanical state is
never duplicated in the decider's private state, only referenced live" are the
right constraints for the arc whether or not a model is ever involved. The
implementation plan is phased with per-phase spec-ID coverage and explicit
acceptance tests, and is honest about its own gaps.

### Do not take

- **`goals.go`.** It is a shallower reinvention of `internal/goals` (4 goal
  kinds, priority ints, no conflict resolution, no hysteresis). Porting it
  would leave us with two goal systems where the worse one is newer.
- **`economy.go`, `loot.go`** are companion-flavoured and do not transfer to
  patrols, schedules or targeting speed.
- **`decisionSchema()`** is OpenAI strict-JSON ceremony. Keep the `Decision`
  struct and `sanitizeDecision`, discard the schema builder.

### Traps

- `perception.go` and `scene.go` read `ParticipantSight` and the three-tier
  sight model that the graded lighting arc is replacing with bands. Re-derive
  against the post-lighting API, do not copy them.
- The patch assumes one monolithic decision covers speech, action, mood,
  memory, opinion, goal, combat and autonomy in a single object. A small local
  model or a rule engine would likely want several independent decisions
  instead, which changes `applyResult` more than the seam suggests.
- The admin command is `companion-unstick`, which resets controller and session
  state. There is no goal-specific unstick command, despite the spec's phrasing.

## The decider seam, and remote or bring-your-own-key deciders

The seam is narrow and already the right shape: one `Decision` struct in,
`sanitizeDecision` clamping every enum and free-text field (including
neutering `;` so a decision cannot smuggle a second command), then
`applyResult` fanning out, with `Action.Ref` validated against the scene the
decision was made from and the whole decision discarded if `worldRev` moved.
Nothing downstream trusts the decider. That property is what would make a
remote decider safe at all.

Constraints found while assessing a self-hosted or player-supplied decider:

- **GoMud modules are Go packages compiled into the binary** (hence the
  `all-modules.go` edit). A module is not a container. The shape available is a
  small compiled-in module, off by default, talking to a pluggable decider.
  The decider is the part that can live elsewhere.
- **The transport is not abstracted today.** `openai.go` plus the strict-JSON
  schema builder are OpenAI specific. The interface would have to be introduced.
- **Decisions are remote-able, actions are not.** The companion is a
  server-side NPC, so world mutation stays authoritative on our server.
- **Our server cannot reach a player's home machine.** Port forwarding is not
  something to ask players for. The variant that works uses our **web client**:
  the browser holds the player's key, calls the provider directly, and posts
  the decision back over the existing websocket. We never see the key and never
  pay. It only works for web-client players, so the deterministic path has to
  be good rather than a degraded mode.
- **Design decision, not a technical one:** a companion whose choices come from
  the player's own machine is a sanctioned bot. The bounded vocabulary and
  server-side validation mean it cannot break the world, but it can play well
  and tirelessly while the player is AFK. Settle this before building anything.
  Related: players with a key get a smarter companion, which cuts against the
  patch's own R1 fairness rule.

## Jev, for the record

Investigated 2026-09-22 as an alternative decider. TypeSafe AI's "System One"
model, launched 2026-09-15. Non-autoregressive: it prefills the answer position
and emits one token constrained to the supplied options, so it produces no text.
Question types are Choice, Score and Noul.

- **Cannot be self-hosted.** Closed managed API, no open weights, no VPC or
  on-premise option. A bigger or second droplet buys nothing because there is
  nothing to install.
- **Cost is a different regime:** $0.042 per 1M input tokens, output free
  (there are no output tokens). At a roughly 500 token state snapshot that is
  about $0.000021 per decision, roughly 47,000 decisions per dollar.
- **The bill is set by trigger rate, not unit price.** Plan-layer only at about
  0.1 decisions per second is roughly $5 per month; 1 per second is roughly $54
  per month; per-round per-mob at 10 per second is roughly $544 per month. Same
  lesson as the patch.
- **Latency 70 to 500 ms.** That fits a slow plan layer and can never serve a
  per-round reflex, which is exactly why the plan and reflex split above
  matters.
- **Open reimplementations of the interface do exist** and are self-hostable
  (OpenJev, AnyJev, Laya at 421M, NanoJev at 0.6B, OpenJev Verdict at 151M,
  minojev on Qwen3-1.7B). One constrained forward pass with no generation is
  CPU-feasible, so a second droplet at roughly $24 to $48 per month fixed is
  plausible where a chat LLM was hopeless. Calibration is measurably worse than
  hosted (Brier 0.048 and ECE 0.041 hosted versus 0.071 to 0.170 for zero-shot
  reproductions), which matters less inside a behaviour tree that already bounds
  the legal choices.
- **The production droplet is 1 CPU / 961 MB.** Nothing model-shaped runs there
  alongside the MUD. Any self-hosted decider needs its own box.

Sources: seangoedecke.com/jev-means-structured-output-is-interesting-again,
jev-agent.com, github.com/cobanov/awesome-jev,
github.com/zhangcy122/OpenJev, github.com/ikermoel/open-alternative-jev.

## Suggested sequencing

1. Fold the deterministic pieces into the behaviour rework arc. They benefit
   every mob, cost nothing, need no network, and the plan and reflex split
   addresses the arc's known blocker directly.
2. While doing it, define the decider interface and keep the bounded action
   vocabulary plus the sanitize step, even though the only implementation is
   our own utility scorer. That is a seam, not speculative machinery, and it
   keeps Jev or a player-supplied decider a config change rather than a rewrite.
3. Leave the remote decider unbuilt until there is something in step 1 worth
   improving on.
