# AI companion integration follow-up (after PR #161)

PR #161 (finaltwist's `modules/aicompanion`) merged at `d4a23b47a` on
2026-09-25. This plan carries the fixes we took on ourselves, the defects found
reviewing his last commit `a381b9abe`, and belt-and-suspenders hardening so the
module fails closed. Reviews: `docs/audits/2026-09-23-ai-companion-pr161-review.md`
(13 findings, 11 decisions).

Branch `fix/aicompanion-integration`, worktree `C:/tmp/dogmud-pr161`. Another
session works lighting and messaging in the main checkout: stay in this
worktree, never touch `C:/tmp/dogmud-boot-check`, no bare `git stash`.

## Facts verified against source (at `d4a23b47a`)

| Fact | Where |
|---|---|
| New consent gate `if !m.consented(...) { continue }` sits ABOVE `answerConsent`, whose only caller is `listeners.go:148` | `listeners.go:125` |
| So spoken `i agree`/`i decline` is never read, and no `heard` stimulus is pushed for an unconsented owner | `listeners.go:125-162` |
| `dispatch` already sends unconsented owners to `m.fallback(c, mob, stims)` (set lines) | `runtime.go:393-397` |
| `handleAsk` returns before `c.push` when unconsented, so `companion-ask` goes unanswered too | `listeners.go:364-366` |
| `onEmote` has the same early `continue` | `listeners.go:191` |
| `TestConsentIsLiteralAndGatesEverything` tests only `consented()` and the question text; no test drives a model path with consent declined | `aicompanion_test.go:1959` |
| `strangerMayAsk` (cooldown + daily cap) is called only from `handleAsk`; a stranger's directly-addressed `say` pushes `heard` with `AskerUserId` and no throttle | `listeners.go:334`, `:161-162` |
| `tryReserveTokens(ownerId, tokens)` charges the OWNER for every reservation; `chargeStranger` adds to the stranger on top | `models.go:489-501`, `runtime.go:1283-1293` |
| `ownerPrompted` returns true whenever the owner spoke, even if a stranger also spoke in the batch (S6) | `actions.go:162-181` |
| `ownerDrivenOnly` lacks `cast`; the `cast` branch has no harm check | `actions.go:144-150`, `:223-245` |
| Engine harm policy is keyed on a PLAYER actor: `mobs.CheckPlayerHarm(m *Mob) HarmBlock`, `(*Room).CanPvp(att, def *users.UserRecord) error`, `rejectHarmTarget` ("Mob casters are never gated") | `internal/mobs/harm_authorization.go:41`, `internal/rooms/rooms.go:2976`, `internal/actions/cast.go:389-391` |
| `refusesToFight` is a module copy that checks `HasShop`/`IsNonCombatant`/profile refuse list and misses `player_attack_immune` | `combat.go:813-827` |
| `holdFollow(userId)` is installed via `companionai.SetHolder`; it holds every companion of that owner | `autonomy.go:525-530`, `aicompanion.go:234` |
| Engine emits `events.Emote` and `events.Healed` unconditionally; only the module listens, and it registers nothing when off, so `DoListeners` logs "no listener for event" | `usercommands/emote.go:31,56`, `hooks/spell_resolution.go:927`, `events/listeners.go:234`, `aicompanion.go:189-229` |
| `config.yaml` comment points at `modules/aicompanion/files/data-overlays/config.yaml`, which does not exist (module comment and `settings.md` say none ships) | `_datafiles/config.yaml:2353`, `aicompanion.go:171`, `docs/aicompanion/settings.md:7` |
| Five files tracked on master before #161 were deleted by it | `git diff --diff-filter=D 5fab94896 d4a23b47a` |
| `golangci-lint --new-from-merge-base` reports 9 issues in the module: errcheck `resp.Body.Close` at `openai.go:233,320,400`; ineffassign at `combat.go:741`, `perception.go:265`, `romance.go:274`, `runtime.go:769`; SA5011 at `runtime.go:438/470` | local lint run |
| `config.yaml` carries NO skip-worktree bit in this worktree (`git ls-files -v` shows no `S`) | checked |

## Tasks (sequential; each ends in its own commit)

1. **Housekeeping.** Restore the five deleted files from `5fab94896`. Fix the
   `config.yaml` pointer (to `docs/aicompanion/settings.md`) and add the
   romance disclosure to the `Enabled` comment: enabling the module enables
   the possibility of a romance, which only moves at the player's own command
   (`companion-court`; `companion-boundary friendship` closes it). Add the
   same note to `docs/aicompanion/upstreaming.md`. Add `companion-follow` to
   the parity allowlist if the parity test lacks it. Clear the nine lint
   findings with real fixes, not suppressions.
2. **Consent, fixed and made structural.** Move the gate so only memory writes
   (`addLine`, `noteConversation`, `dirty`) are skipped: `answerConsent`
   runs, stimuli are pushed, `dispatch` answers with set lines. Belt and
   suspenders: one consent check at the single outbound model choke point
   (every HTTP request to OpenAI), keyed by owner, so a future path cannot
   leak even if its caller forgets. Tests: spoken `i agree` inside the window
   consents; `i decline` refuses; an unconsented owner's speech gets a
   fallback reply; reflection, summary, core memory and dispatch with
   consent declined make ZERO requests against an `httptest` server (prove
   the test fails with the choke-point guard and caller gates removed).
3. **Strangers (S5, S6).** A stranger's directly-addressed `say` passes
   `strangerMayAsk` before a stimulus is pushed (heard and remembered if
   consented, but not answered by the model). Stranger-prompted calls reserve
   against the stranger's allowance, not the owner's. Owner-only verbs are
   refused when any stranger stimulus is in the batch.
4. **Harm gating.** `cast` of a harmful spell joins the owner-driven set.
   Companion `attack` and harmful `cast` targets are authorised through the
   engine's policy with the OWNER as the actor (`CheckPlayerHarm` for mobs,
   `CanPvp(owner, target)` for players, charm rules), in a module-only helper.
   `refusesToFight` keeps only the profile refuse list as personality.
   Engine-wide "a mob acting for a player is gated as that player" is OUT of
   scope (changes regular companions; separate owner call).
5. **Module-off and follow.** `holdFollow` holds only the bonded AI companion.
   Module off: no "no listener" error for `Emote`/`Healed`. A companion bonded
   while the module was on can be dismissed after it is switched off.
6. **Docs and gates.** `modules/aicompanion/context.md`, amend decision 2 in
   the 09-23 review doc (lighting-plan-6 gate dropped, owner 09-24),
   `docs/PATCH_NOTES.md`, this plan in `docs/README.md`. `gofmt`, build,
   package tests, local lint `0 issues`, boot with the module OFF and ON
   (own port, own PID), PR with `--repo pruuk/DOGMud`.
