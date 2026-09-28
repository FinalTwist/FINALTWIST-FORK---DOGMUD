# Spell Effects 3b (Helpful) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every helpful spell effect (condition, heal, shield, purge, and the default arm) lands through one applier per effect for every pairing (PM, PP, MS, MM, MP), a help spell is uncontested everywhere, and one area-help target filler serves player and mob casters.

**Architecture:** A new file `internal/hooks/spell_help_effects.go` holds the helpful appliers, the default arm and the area-help filler, built on slice 3a's `spellEffectCtx` and `applySpellEffect` in `spell_effects.go`. The four resolvers in `spell_resolution.go` keep their names and their one `runSpellChannelAttack` call each; they gain one shared uncontested step (`resolveHelpSpell`) for help spells, and `resolveMobSpell`'s self branch (MS) takes that step too. The per-pairing arms (`applyMobEffectArms`, `applyPlayerEffectArms`, `applyMobOnPlayerArms`), `applyMobSelfEffect`'s body and the test-only wrappers are deleted once nothing needs them.

**Tech Stack:** Go, `internal/hooks`, `internal/actions` (`Actor`), `internal/parties`, `internal/mobs` (`FindPackmatesInRoom`), `internal/messaging` (`SendTrio`), `internal/events`, testify.

**Spec (binding):** `docs/superpowers/specs/2026-09-28-spell-effect-unification-design.md`, sections "3b helpful", "Owner rulings" and "Architecture", plus the owner's 3b conditions: area help reaches mobs charmed by the caster or by anyone in the caster's party (`parties.Get(userId).IsMember`), so companions (the AI companion included) are healed; the dead player-to-player heal and shield crits are deleted; mob charm and mob purge-affliction are punted to the behaviour arc.

**Subagent models:** Tasks 1, 2, 5, 7, 8 opus (judgment: resolver edits that must keep every pairing green, the crime extension, deleting three switches at once, the ally rule, the parity net). Tasks 3, 4, 6, 9, 10 sonnet (code given, mechanical). Task 10's playtest follows the `dogmud-playtesting` skill and the `playtest-scenario` command.

---

## Facts verified against source (2026-09-28, branch `docs/spells-3b-plan` at master `b716619be`, which includes slice 3a, #181)

Every line number below was read from the file on this commit.

| # | Fact | Where |
|---|---|---|
| F1 | `applySpellEffect` (161-179) switches only `damage`, `dot`, `knockdown`, then routes a mob target to `applyMobEffectArms`, a mob caster to `applyMobOnPlayerArms`, else `applyPlayerEffectArms` (170-178). Test-only wrappers `applyMobEffect` (489-493) and `applyPlayerEffect` (496-500); `spellCasterActor` (58-68) is used only by `applyMobEffect` (491) | `internal/hooks/spell_effects.go` |
| F2 | Helpers 3b builds on, all in `spell_effects.go`: `spellEffectCtx` (39-47), `newSpellEffectCtx` (49-53), `actorUser` (70), `actorMob` (77), `casterRef` (101), `targetRef` (108), `casterName`/`targetName` (132-133, through `spellEffectName` 122-130: players in the username tag, mobs through `mobDisplayName`), `critTag` (135), `category` (142), `audience` (147), `commitHarmfulSpellAggro(c, fresh)` (193-204, `SeedAggression` for a player caster on a mob), `spellCasterStatAndSkill` (282-291: `CasterStatValue` plus the Manifestation-aware cast skill; nil caster reads 100 and 0), `recordSpellResolution` (418-422), `recordSpell` seam (36) | `internal/hooks/spell_effects.go` |
| F3 | PM/MM arms: `setMobSpellAggro` 518-532 (only caller 551); `applyMobEffect_condition` 534-579 (raw `user.SendText` 571-573 and `sendVisualRoomText` 574-576, all gated on `user != nil`, so MM is silent); `applyMobEffect_heal` 591-620 (skill `Spellcasting`, stat `CasterStatValue`, nil caster reads 0 and 0; room-only line 616-618; `AddConditionMagnitude` regen 615); `applyMobEffect_default` 622-644; `applyMobEffectArms` 652-681 (`critTag` 655-658 used only by the condition case 666-667; heal case 668-672 queues `events.Healed` at 670 when `user != nil`; charm 673-677; default 678-679). No shield and no purge arm on a mob target | `internal/hooks/spell_resolution.go` |
| F4 | PP arms: `applyPlayerEffectArms` 738-977: defended early return 747-752; `purge` 755-792; `heal` 794-840 (crit boost 801-804, regen `AddConditionMagnitude` 809); `condition` 842-888; `shield` 890-936 (crit x1.5 at 905-907, ward `AddConditionMagnitude` 908); `default` 938-976 (target told nothing: `Actee: messaging.NoLine` at 972) | same |
| F5 | MP arms: `applyMobOnPlayerArms` 1289-1344, a switch with only `condition` (1297-1327) and `default` (1328-1342); heal, shield and purge on a player from a mob fall to `default` ("takes effect on you.") and apply nothing | same |
| F6 | MS: `applyMobSelfEffect` 1179-1217 (heal 1181-1193 with regen `AddConditionMagnitude` 1191; condition 1194-1197; shield 1198-1215 with ward `AddConditionMagnitude` 1213; no purge; records nothing); called only from `resolveMobSpell`'s self branch 1059-1064 | same |
| F7 | Help skip: PP in `resolveSpell` 164-173 (before `resolveAgainstPlayer`, records nothing); PM `resolveAgainstMob` 399-408 (records); MM `resolveMobSpellAgainstMob` 1223-1239 (records nothing); MP `resolveMobSpellAgainstPlayer` 1265-1284 has NO skip (spec fact 3 / audit row 3). `resolveAgainstPlayer` 694-728 carries a harm-aggro block 712-721 whose comment says it serves harmful condition spells until slice 3b | same |
| F8 | Area help: `resolveSpell` 105-120 takes every room player and ANY charmed mob (`m.Character.IsCharmed()`, 115). `resolveMobSpell` 1017-1076 fills only HarmArea (1055-1057). `InitiateCast` fills a HelpArea cast, player or mob, with every player and every mob in the room (`internal/actions/cast.go:315-317`). `resolveSpell`'s doc bullet "HelpArea is player-only (mobs never cast area healing in this engine)" (51) is false: Temple Priest Olen (mob 95) has `cast cleansing-wave`, an area purge, in `combatcommands` | `spell_resolution.go`; `internal/actions/cast.go`; `_datafiles/world/dogmud/mobs/thornwall_city/95-temple_priest_olen.yaml:31` |
| F9 | The rule a mob's own AI uses to pick whom to heal: `behaviortree`'s `cast_best_in_category` resolves `target: most_wounded_packmate` and `tanking_packmate` through `mobs.FindPackmatesInRoom` (`action_cast_best_in_category.go:59-76`, `117-146`), which returns living, uncharmed mobs in the same room sharing a `Routine` or linked by `RoutineLinks` (`internal/mobs/packmates.go:17-51`). Archetypes `pure_caster` (46) and `support_caster` (33, 44) use it. Two other ally rules exist and are NOT heal targeting: `(*Mob).ConsidersAnAlly` (`mobs.go:1198`, used by `callforhelp.go:84` to decide who comes running) and `resolveMobHelpMultiTargets` (`internal/actions/cast.go:543-571`, a HelpMulti cast's target list at initiation) | files |
| F10 | `parties.Get(userId)` (`parties.go:152`) returns the party for its leader, members AND invitees (`InvitePlayer` maps invitees, 313-320); `IsMember` (295) checks `UserIds` only. `New(userId)` 137, `InvitePlayer` 313, `AcceptInvite` 323, `Disband` 356. The AI companion is charmed permanently: `mob.Character.Charm(owner.UserId, -1, "")` (`modules/aicompanion/commands.go:211`). `Charm` sets the `charmed` adjective (`internal/characters/charminfo.go:46-53`); `GetCharmedUserId` 55 | `internal/parties/parties.go`; `internal/characters/charminfo.go` |
| F11 | `events.Healed{HealerUserId, MobInstanceId}` (`eventtypes.go:596-607`, documented as "a player casts a healing spell at a mob"); only listener `modules/aicompanion` (`aicompanion.go:232`, `listeners.go:461`). No drain helper exists; `DrainQueuedGoldGivenForTest` (`events.go:405-425`) is the pattern; `internal/events/context.md:586-598` lists the drain seams | `internal/events` |
| F12 | `applySpellCondition(target spellConditionTarget, spellData, caster, conditionId)` (`light_spell.go:61-72`); `spellConditionTarget` (52-56) is satisfied by `*users.UserRecord` and `*mobs.Mob`, both of which queue `events.Condition`. `applyMobEffect_charm(user, targetMob, room, spellData, out, mName)` (`charm_spell.go:26-33`) returns 0 for a nil user. `spellNarratedByGoHook` (`spell_resolution.go:983-989`). `calcSpellDuration` (33-42). `purgeTarget` and `resolvePurgeAffliction` (`spell_purgeaffliction.go:16`, 59) serve the `purge-affliction` spell id only, from `resolveSpell`'s Go-hook switch (267-299) | files |
| F13 | Shipped helpful spells (`_datafiles/world/dogmud/spells/`): heal `heal` (Mend Flesh, magnitude 3), `mend-wounds` (5), `repair-pulse` (8), area heal `mend-all` (3), `mass-mend` (5), `communion-of-flesh` (4); shield `conviction-ward` (75), `chrysalis-cocoon` (125); purge `cleansing-wave` (area). All `attack_type: none`, `primarystat: willpower`, schools vital or enhancement; none is Manifestation, so `spellCasterStatAndSkill` equals the old `Spellcasting` plus `CasterStatValue` for every shipped heal and shield. Harmful condition spells (`attack_type: spell`): `mind-fog`, `nerve-disruption`, `neural-stun`, `psychic-anchor`, `sensory-overload`, `sensory-veil`. No shipped spell with `effect_type` none or empty is harmful | spell YAML |
| F14 | Mob healers: Bandit Caster 285 (`routine: bandit_camp_guard`, shared with 284 and 286; spellbook `mend-wounds: 30`) spawns in Bandit Camp 4052; Temple Priest Olen 95 (`cast heal`, `cast mend-wounds`, `cast cleansing-wave` in combat); Repair Frame 9585 casts `repair-pulse` AT a named boss (MM, `behaviors/crash_site_interior/9585-repair_frame.yaml:44-70`); summons 302, 303, 312 carry `heal`. No shipped mob casts a heal or shield AT a player (every `cast heal` is self or a named mob), so MP heal and shield are reachable only in tests | mob and behaviour YAML |
| F15 | Guards. `TestPlayerConditionsTravelTheEventPath` (`condition_apply_path_guard_test.go:377`) keys each `AddConditionMagnitude` call: ward rows 188-189 (`spell_resolution.go|908`, `|1213`), regen rows 202-204 (`|615`, `|809`, `|1191`), comments 170-187 and 191-201. `TestNarrationSitesMatchViewpointAudit` (`messaging_surface_guard_test.go:1447`, set equality) holds `"hooks/spell_resolution.go|Your %s takes effect on %s!%s"` at 1287, which is `applyMobEffect_condition`'s `user.SendText`; a send through `SendTrio` leaves the walk. `TestSpellResolversRunOneContestAndAppliersRollNone` (`internal/hooks/channel_defence_routing_test.go:58`) parses `spell_resolution.go` and `spell_effects.go` (66). `sightExemptSites` (`sight_penalty_guard_test.go:117-120`) keys the four resolvers by `file|func`; they keep their names, so nothing re-keys there. `send_trio_only_guard_test.go:76-84` is a survey comment naming `applyMobEffect_condition`, `_heal`, `_default` as the raw spell senders left | repo root, `internal/hooks` |
| F16 | Test callers of names 3b retires: 28 calls in 6 files. `hooks_test.go` 2648, 2665 (`applyMobEffect`); 2699, 2715, 2731, 2747, 2762, 2776, 2791 (`applyPlayerEffect`); 2881, 2896 (`applyMobSelfEffect`). `selfcast_spell_room_hiding_test.go:110`. `selfcast_wording_test.go` 36, 56, 77, 99, 131, 214 (`applyPlayerEffect`), 254, 260 (`applyMobEffect_default`). `spell_names_in_dark_test.go` 22, 75. `spell_tick_scale_test.go` 108 (`applyPlayerEffect`), 134 (`applyMobSelfEffect`). `wire_freeze_test.go` 67, 215 (`applyMobEffect`), 126, 240 (`applyMobSelfEffect`). Only `hooks_test.go` imports `internal/actions`. The five `applyMobEffect_charm` calls (`charm_effect_test.go`, `charm_expiry_test.go`) stay: charm keeps its function | `internal/hooks/*_test.go` |
| F17 | Wording pinned by tests (the unified appliers keep PP's lines): `selfcast_wording_test.go` 24-263 (self-cast purge, heal, condition, default; cross-cast caster, target and room lines; area heal and purge through `resolveSpell`), `spell_names_in_dark_test.go:34-65, 110` ("Something's Hex takes effect on you." is MP default), `selfcast_spell_room_hiding_test.go:113-115` (self shield) | `internal/hooks` |
| F18 | Fixture: `newSpellParityFixture` (`spell_effect_fixture_test.go:43`) seeds users 1 Aliceia, 2 Bobrick, 3 Carys (watcher), mobs Skeleton 100 and Ghoul 101 in lit room 1, all equalised to stats 100 and `Spellcasting` 3; pins the contest; captures `recordSpell`. `spellParityPairings` (`spell_resolver_steps_test.go:28`) drives PM, PP, MM, MP through their resolvers. Condition 100 (`hooks_test.go:59-65`) has no tick pool. `conditions.SeedConditionRecordsForTest` seeds 119 Minor Shield, 120 Regenerating, 121 Poisoned. `events.DrainQueuedConditionsForTest` (`events.go:567`), `DrainQueuedMobConditionsForTest` (594, zero drains every Condition event) | files |
| F19 | Imports of `spell_resolution.go` used only by the helpful arms: `events` (670), `configs` (892, 1200), `conditions` (615, 756, 809, 908, 1191, 1213). `math` stays (41, 367); `skills` stays (333, 335); `targeting` and `state` stay (1156) | `spell_resolution.go` |
| F20 | `SkillWeight` Go default 2.0 (`internal/configs/config.balance.go:326`), ships 5.0 (`_datafiles/config.yaml:1084`); test binaries read the Go default, so tests compute from `configs.GetBalanceConfig().SkillWeight` | files |
| F21 | Docs naming retired symbols: `internal/hooks/context.md` 121-143, 1004-1008, 1803-1804, 1868-1900, 1902-1945; `internal/conditions/context.md` 1183-1221; `internal/characters/context.md` 603-605; `internal/combat/context.md:1846`; `internal/items/context.md:1216-1218`. Comments: `charm_spell.go` 18-21, 34-37; `internal/actions/cast.go:135-137`; `spell_resolution.go` 51, 243-246, 979-982, 1013-1014; `wire_freeze_test.go` 20-43 and its four subtest comments; `send_trio_only_guard_test.go:76-84`; `_datafiles/world/dogmud/behaviors/crash_site_interior/9585-repair_frame.yaml:9-20` | files |

### Design decisions and spec readings (for the reviewer)

1. **The mob ally rule is `mobs.FindPackmatesInRoom`** (F9). The spec says to use "the rule the mob's own AI uses" for allies. Three rules exist; only `FindPackmatesInRoom` is the one the AI uses to choose whom to HEAL (`cast_best_in_category`'s `most_wounded_packmate`). `ConsidersAnAlly` decides who answers a call for help, and `resolveMobHelpMultiTargets` builds a HelpMulti cast list at initiation. So an uncharmed mob's area help lands on itself and its packmates and never on a player. A mob charmed by a player stands on that player's side and uses the player rule. **Please confirm.**
2. **Harmful condition and harmful default spells become crimes on a mob.** Slice 3a made damage, dot and knockdown crimes under ruling 2 ("harmful spells are crimes"). The unified condition applier starts the fight through the same `commitHarmfulSpellAggro`, so a player's `mind-fog` or `neural-stun` on a townsperson now records the assault too. The spec's 3b section is silent; ruling 2 reads as covering every harmful spell. **Please confirm.**
3. **`events.Healed` fires only for a mob target.** The spec says it "fires when the healer is a player"; the event carries only `MobInstanceId` and its one listener is the AI companion (F11), so a player healing a player has nothing to report. It fires for a player caster on a mob target, as today.
4. **Every help cast is recorded** (`RecordSpell`, one landed row), as PM already did; PP, MM, MP and MS recorded nothing for a help spell.
5. **Heal and shield read the caster through `spellCasterStatAndSkill`**, the helper the dot uses. For every shipped spell the numbers are unchanged (F13). A nil caster (tests only) now reads stat 100 instead of 0 for a heal.
6. **A defended help effect applies nothing.** Help spells are uncontested, so this is reachable only for a contested spell authored with a help effect. The old PM heal arm ignored `Defended`; PP's early return did not. The unified appliers follow PP.
7. **Heal, shield and purge lines drop the `[CRIT!]` tag**: they cannot crit (ruling 3). The condition line keeps it; a harmful condition can crit.
8. **MS never applies a harmful spell to the caster.** `resolveMobSpell`'s self branch skips a harmful spell, which the old `applyMobSelfEffect` did implicitly by having no harmful arms.
9. **Spec fact 12's count**: the names 3b retires have 28 test calls in 6 files (F16); the "about 70 in 13 files" from 3a counted the resolvers too.

No source fact contradicts the spec in a way that changes the design.

### Behaviour changes this PR ships (for the PR body)

1. A creature's help spell on a player is uncontested and lands: a creature's heal mends you and its shield shields you (audit row 3).
2. A shield on a creature works: your pet can be shielded, and creatures shield each other (audit row 15).
3. A cleansing spell purges poison from a companion or creature, and from a player when a creature casts it.
4. A player's area help lands on every player in the room plus creatures charmed by the caster or by any party member (the AI companion included); a stranger's pet is no longer healed (audit row 13, spec fact 5).
5. A creature's area help lands on itself and its packmates (`FindPackmatesInRoom`), never on players; a charmed creature's area help lands on its owner's side.
6. A creature casting on itself reads like a player: "A shimmering barrier surrounds X." (was "forms around X."), and a condition it casts on itself is announced ("Bless settles over X.").
7. Condition lines are one set for every pairing: the room reads "settles over" (PM and MP said "affects"; MM said nothing).
8. A creature healing a creature: the room reads "X's Mend envelops Y in healing light." (was "washes over Y, knitting wounds shut."), and a player caster or target reads their own line.
9. A player's harmful condition spell on a mob is an assault (`SeedAggression`, crime on a fresh engagement); a mob's harmful condition or default spell now commits both sides' aggro.
10. A player-to-player spell with no effect of its own now tells the target ("X's Y takes effect on you."), as a creature's already did.
11. Every help cast is recorded in spell analytics, for every pairing.
12. A creature's self-cast help spell counts as landed for its skill progression (it counted as not landed).
13. The unreachable heal and shield crit boosts are deleted; nothing a player sees changes.

---

## File map

| File | Change | Responsibility |
|---|---|---|
| `internal/hooks/spell_help_effects.go` | **Create** (Task 2), grows in 3-5, 7 | `selfCast`, `selfCastAudience`, `spellConditionTargetOf`, `spellStatusDefended`, the condition, heal, shield, purge and default appliers, `spellHelpAreaTargets`, `helpAreaCharmAlly` |
| `internal/hooks/spell_effects.go` | Modify (1, 2-5 dispatcher, 6 wrapper, 9 delete wrappers) | `uncontestedSpellResult`, `resolveHelpSpell`, the final dispatcher |
| `internal/hooks/spell_resolution.go` | Modify (1-7, 9) | help step in the resolvers, arms deleted, MS branch, area filler hookups, comments |
| `internal/events/events.go`, `internal/events/context.md` | Modify (3) | `DrainQueuedHealedForTest` |
| `internal/hooks/spell_help_test.go` | **Create** (1) | contest counter, condition spell, uncontested-and-recorded table |
| `internal/hooks/spell_condition_test.go` | **Create** (2) | room line per pairing, harmful condition is an assault, MM fight |
| `internal/hooks/spell_heal_test.go` | **Create** (3) | MP heal, crit changes nothing, `Healed` |
| `internal/hooks/spell_shield_test.go` | **Create** (4) | pet shield, MP shield, crit changes nothing |
| `internal/hooks/spell_purge_test.go` | **Create** (5) | purge on companion and from a creature, default tells the target |
| `internal/hooks/spell_selfcast_test.go` | **Create** (6) | `spellHelpParityPairings`, MS seen, cleansed, recorded, never self-harmed |
| `internal/hooks/spell_help_area_test.go` | **Create** (7) | the three area-help rules |
| `internal/hooks/spell_help_parity_test.go` | **Create** (8) | the five-pairing parity table |
| `internal/hooks/channel_defence_routing_test.go` | Modify (2) | parse `spell_help_effects.go` too |
| `condition_apply_path_guard_test.go` | Modify (1-6 re-keys, 9 comments) | allowlist keys |
| `messaging_surface_guard_test.go` | Modify (2) | drop the retired condition row |
| `send_trio_only_guard_test.go` | Modify (9) | survey comment |
| `internal/hooks/hooks_test.go`, `selfcast_wording_test.go`, `selfcast_spell_room_hiding_test.go`, `spell_names_in_dark_test.go`, `spell_tick_scale_test.go`, `wire_freeze_test.go` | Modify (3, 5, 9) | migrate callers off the wrappers |
| `internal/hooks/charm_spell.go`, `internal/actions/cast.go`, `_datafiles/world/dogmud/behaviors/crash_site_interior/9585-repair_frame.yaml` | Modify (9) | comments naming retired functions |
| `internal/hooks/context.md`, `internal/conditions/context.md`, `internal/characters/context.md`, `internal/combat/context.md`, `internal/items/context.md`, `docs/PATCH_NOTES.md`, `docs/README.md` | Modify (10) | docs |
| `tools/playtest/scenarios/spell-effects-3b.yaml`, `tools/playtest/goals/scenarios/spell-effects-3b/{healer,partner,mobwatch}.yaml` | **Create** (10) | playtest |

---

## Execution setup (once, before Task 1)

- [ ] Create the worktree (Bash). Base on `master` if this plan's branch has merged, else on `docs/spells-3b-plan`:

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && git worktree add C:/tmp/dogmud-3b-helpful -b feature/spell-effects-3b-helpful master
```

- [ ] Confirm the baseline is green for the packages this plan touches:

```bash
cd C:/tmp/dogmud-3b-helpful && go test ./internal/hooks/ -count=1 && go test . -run 'TestPlayerConditionsTravelTheEventPath|TestNarrationSitesMatchViewpointAudit|TestEveryTrioLiteralNamesAllThreeRoles|TestEveryRollSiteAppliesTheSightPenalty|TestNarrationTrioOnlyCategoriesLeaveOnlyThroughSendTrio' -count=1
```

Expected: `ok  github.com/GoMudEngine/GoMud/internal/hooks` and `ok  github.com/GoMudEngine/GoMud`. If either is red on the untouched base, stop and report.

**The root-guard command above is called "the root guards" in every task below.** Run it from the worktree root.

**Keeping the root guards green.** Moving code shifts lines, so `TestPlayerConditionsTravelTheEventPath` reports each old key as stale and each moved or new call as unallowlisted, printing the key (`internal/hooks/spell_resolution.go|N` or `internal/hooks/spell_help_effects.go|N`). Edit the allowlist in `condition_apply_path_guard_test.go` in the SAME commit as the move: change a moved key's line number and keep its reason string exactly; where a task says rows COLLAPSE, replace the stale keys with the one new key under the same reason string; where a task says a row is DELETED, remove it. Match by the source line the failure prints (a `ConditionIdRegenerating` call is a regen row, a `ConditionIdMinorShield` call is a ward row). `TestNarrationSitesMatchViewpointAudit` reports stale rows; delete exactly the rows a task names. If it reports an UNREGISTERED event, stop: this plan adds only `SendTrio` sends, which leave the walk.

**Edits.** Write and edit files with the Write and Edit tools only. Never Python, never `sed -i`.

---

### Task 1: One uncontested help step in all four resolvers

**Model:** opus.

**Files:**
- Create: `internal/hooks/spell_help_test.go`
- Modify: `internal/hooks/spell_effects.go` (after `recordSpellResolution`, 415-422)
- Modify: `internal/hooks/spell_resolution.go` (164-173, 391-408, 694-698, 1223-1239, 1265-1268)
- Modify: `condition_apply_path_guard_test.go` (re-key)

- [ ] **Step 1: Write the failing test** `internal/hooks/spell_help_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countSpellContests wraps the fixture's pinned contest with a counter. The
// restore is a t.Cleanup registered after the fixture's own, so it runs
// first and hands the seam back to the fixture's pin.
func countSpellContests(t *testing.T) *int {
	t.Helper()
	n := 0
	pinned := runSpellChannelAttack
	runSpellChannelAttack = func(v messaging.RoomVisibility, a combatvocab.Attack, s combat.AttackSide,
		atk, def *characters.Character) combat.ChannelDefenceResult {
		n++
		return pinned(v, a, s, atk, def)
	}
	t.Cleanup(func() { runSpellChannelAttack = pinned })
	return &n
}

// conditionSpellForParityTest is a help spell that queues condition 100,
// which seedAllRegistries defines with no tick pool and no scaled kind, so
// applySpellCondition takes the plain AddCondition door.
func conditionSpellForParityTest() *spells.SpellData {
	return &spells.SpellData{
		SpellId: "test-bless", Name: "Bless", AttackType: combatvocab.AttackNone,
		DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle,
		EffectType: "condition", ConditionIds: []int{100}, BaseFolds: 4, PrimaryStat: "willpower",
		Schools: []string{spells.SchoolEnhancement},
	}
}

// Parity slice 3b: a help spell is uncontested and recorded once as a landed
// cast on every pairing. MP was contested (audit row 3); PP skipped the
// contest only inside resolveSpell; MM recorded nothing.
func TestSpellHelp_EveryPairingIsUncontestedAndRecordedOnce(t *testing.T) {
	for _, p := range spellParityPairings() {
		t.Run(p.name, func(t *testing.T) {
			f := newSpellParityFixture(t, spellContestAttackWin())
			contests := countSpellContests(t)

			p.cast(f, conditionSpellForParityTest())

			assert.Zero(t, *contests, "a help spell runs no contest")
			require.Len(t, f.records, 1, "one record per resolved cast")
			assert.Equal(t, p.src, f.records[0].src)
			assert.Equal(t, p.tgt, f.records[0].tgt)
			assert.True(t, f.records[0].hit, "an uncontested cast landed")
		})
	}
}
```

- [ ] **Step 2: Run it to see it fail**

Run: `cd C:/tmp/dogmud-3b-helpful && go test ./internal/hooks/ -run TestSpellHelp_EveryPairingIsUncontestedAndRecordedOnce -count=1`
Expected: FAIL in `PP` and `MP` ("a help spell runs no contest", 1 contest) and `MM` ("one record per resolved cast", 0 records). `PM` passes.

- [ ] **Step 3: Add the shared step** to `internal/hooks/spell_effects.go`, directly after `recordSpellResolution` (ends at 422):

```go
// uncontestedSpellResult is the contest result a help spell (attack_type
// none) resolves with: an attack win at full strength and no crit. A help
// spell never enters the contest, the only source of a crit, so help spells
// do not crit (owner ruling 3, 2026-09-28).
func uncontestedSpellResult() combat.ChannelDefenceResult {
	return combat.ChannelDefenceResult{DamageMultiplier: 1}
}

// resolveHelpSpell is the one uncontested step every resolver takes for a
// help spell (attack_type none), whoever casts it and whoever it lands on:
// no contest, so no fumble, backfire, interrupt or counter; the effect
// applies and the cast is recorded as landed. It always reports landed:
// there was no defence to beat. A mob's help spell on a player used to be
// contested and then apply nothing (audit row 3).
func resolveHelpSpell(c spellEffectCtx) bool {
	recordSpellResolution(c, applySpellEffect(c))
	return true
}
```

- [ ] **Step 4: PP.** In `spell_resolution.go`'s `resolveSpell`, replace lines 164-174:

```go
		if spellData.AttackType == combatvocab.AttackNone {
			// Non-harm cast: uncontested, an attack win by construction.
			// Uncontested means it LANDED: there was no defence to beat.
			applySpellEffect(newSpellEffectCtx(user.Character, actions.NewUserActorInRoom(user, room), actions.NewUserActorInRoom(targetUser, room), room, spellData, magnitude, combat.ChannelDefenceResult{DamageMultiplier: 1}))
			anyLanded = true
		} else {
			fumbled, landed := resolveAgainstPlayer(user, targetUser, room, spellData, side, magnitude)
			castFumbled = castFumbled || fumbled
			anyLanded = anyLanded || landed
		}
		targetsResolved++
```

with:

```go
		// A help spell (attack_type none) resolves uncontested inside
		// resolveAgainstPlayer, as it does in the other three resolvers.
		fumbled, landed := resolveAgainstPlayer(user, targetUser, room, spellData, side, magnitude)
		castFumbled = castFumbled || fumbled
		anyLanded = anyLanded || landed
		targetsResolved++
```

Then, at the top of `resolveAgainstPlayer`'s body (the blank line 695, before `// Task 17: the sleeping-victim forced crit reaches the spell channel.`), insert:

```go
	if spellData.AttackType == combatvocab.AttackNone {
		return false, resolveHelpSpell(newSpellEffectCtx(user.Character, actions.NewUserActorInRoom(user, room),
			actions.NewUserActorInRoom(target, room), room, spellData, magnitude, uncontestedSpellResult()))
	}
```

- [ ] **Step 5: PM.** In `resolveAgainstMob`, replace lines 391-408 (from `	// Non-harm cast at a mob (a heal on your companion, an area mend over` through the closing `	}` of that `if`) with:

```go
	// A help spell (a heal on your companion, an area mend over allies) is
	// uncontested, as in every resolver (resolveHelpSpell). On master
	// 612b85d54 it ran a quell contest here, so a companion could "defend"
	// its own heal, a fumble backfired on the caster, and a defensive crit
	// earned the companion a counter-swing at its owner.
	if spellData.AttackType == combatvocab.AttackNone {
		return false, resolveHelpSpell(newSpellEffectCtx(user.Character, caster, target, room, spellData,
			magnitude, uncontestedSpellResult()))
	}
```

- [ ] **Step 6: MM.** In `resolveMobSpellAgainstMob`, replace lines 1223-1239 (from `	// Non-harm effects (a heal, or a condition buff cast on an ally mob) are` through `		return true\n	}`) with:

```go
	// A help spell (a heal, or a condition buff cast on an ally mob) is a
	// cooperative cast, not an attack: uncontested, as in every resolver
	// (resolveHelpSpell). Crash-site boss-mechanics Chunk B: the Repair Frame
	// add heals Warden-Prime and the Core Guardian this way.
	casterActor := actions.NewMobActorInRoom(caster, room)
	targetActor := actions.NewMobActorInRoom(target, room)
	if spellData.AttackType == combatvocab.AttackNone {
		return resolveHelpSpell(newSpellEffectCtx(&caster.Character, casterActor, targetActor, room, spellData,
			magnitude, uncontestedSpellResult()))
	}
```

- [ ] **Step 7: MP.** In `resolveMobSpellAgainstPlayer`, insert before `	// Task 17: the sleeping-victim forced crit reaches the spell channel.` (1267):

```go
	// A mob's help spell on a player is uncontested, as on every other
	// pairing (audit row 3): it used to be contested, then fall to the
	// default arm and apply nothing.
	if spellData.AttackType == combatvocab.AttackNone {
		return resolveHelpSpell(newSpellEffectCtx(&caster.Character, actions.NewMobActorInRoom(caster, room),
			actions.NewUserActorInRoom(target, room), room, spellData, magnitude, uncontestedSpellResult()))
	}
```

- [ ] **Step 8: Build, run the test, the package and the root guards**

```bash
cd C:/tmp/dogmud-3b-helpful && go build ./... && go test ./internal/hooks/ -count=1
```

Expected: `ok`. If `go vet ./internal/hooks/` reports `combat` unused in `spell_resolution.go`, it is not: `combat.` is used throughout; any other unused-import report means a step was misapplied.

Run the root guards; re-key the five ward and regen rows per the setup note (the lines shift; no row collapses yet). Expected: `ok`.

- [ ] **Step 9: Commit**

```bash
cd C:/tmp/dogmud-3b-helpful && git add internal/hooks/spell_help_test.go internal/hooks/spell_effects.go internal/hooks/spell_resolution.go condition_apply_path_guard_test.go && git commit -F - <<'EOF'
feat(spells): one uncontested help step in every resolver

A help spell (attack_type none) now takes resolveHelpSpell in all four
resolvers: no contest, the effect applies, one landed record. A mob's
help spell on a player was contested and then applied nothing (audit
row 3); mob-on-mob and player-on-player help casts recorded nothing.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 2: One condition applier, in a new file

**Model:** opus.

**Files:**
- Create: `internal/hooks/spell_help_effects.go`
- Create: `internal/hooks/spell_condition_test.go`
- Modify: `internal/hooks/spell_effects.go` (dispatcher)
- Modify: `internal/hooks/spell_resolution.go` (delete `setMobSpellAggro` and `applyMobEffect_condition`; drop the condition cases; rewrite `applyMobOnPlayerArms`; comment at `resolveAgainstPlayer`'s harm block)
- Modify: `internal/hooks/channel_defence_routing_test.go:64-66`
- Modify: `messaging_surface_guard_test.go:1287`, `condition_apply_path_guard_test.go` (re-key)

- [ ] **Step 1: Write the failing tests** `internal/hooks/spell_condition_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hexSpellForConditionTest is a harmful condition spell, mind-fog's shape.
func hexSpellForConditionTest() *spells.SpellData {
	return &spells.SpellData{
		SpellId: "test-hex", Name: "Hex", AttackType: combatvocab.AttackSpell,
		DamageType: combatvocab.DamageMental, Targeting: combatvocab.TargetSingle,
		EffectType: "condition", ConditionIds: []int{100}, PrimaryStat: "willpower",
		Schools: []string{spells.SchoolMental},
	}
}

// Slice 3b: one condition applier narrates every pairing to the room with
// the same line. Mob on mob was silent; player on mob and mob on player
// said "affects".
func TestSpellCondition_EveryPairingSettlesInViewOfTheRoom(t *testing.T) {
	for _, p := range spellParityPairings() {
		t.Run(p.name, func(t *testing.T) {
			f := newSpellParityFixture(t, spellContestAttackWin())
			target := "Ghoul"
			if p.tgt == combat.User {
				target = "Bobrick"
			}

			p.cast(f, conditionSpellForParityTest())

			assert.Equal(t, 1, countContaining(drainPlain(3), "Bless settles over "+target+"."),
				"a watcher sees the condition take hold")
		})
	}
}

// Owner ruling 2: a player's harmful spell on a mob is an assault. The old
// player-on-mob condition arm committed aggro but never called
// SeedAggression.
func TestSpellCondition_HarmfulOnAMobIsAnAssault(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := hexSpellForConditionTest()

	resolveAgainstMob(f.casterUser, f.targetMob, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), 0)

	attacked := events.DrainQueuedPlayerAttackedMobsForTest(0)
	require.Len(t, attacked, 1, "a player's harmful condition on a mob is aggression")
	assert.Equal(t, 101, attacked[0].MobInstanceId)
	assert.Equal(t, state.ActorRef{UserId: 1}, f.targetMob.Character.CurrentCombatTarget())
}

// A mob's harmful condition on another mob starts the fight. Every aggro
// commit in the old arm needed a player caster.
func TestSpellCondition_MobOnMobHarmStartsAFight(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := hexSpellForConditionTest()

	resolveMobSpellAgainstMob(f.casterMob, f.targetMob, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), 0)

	assert.Equal(t, state.ActorRef{MobInstanceId: 100}, f.targetMob.Character.CurrentCombatTarget())
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `cd C:/tmp/dogmud-3b-helpful && go test ./internal/hooks/ -run 'TestSpellCondition_' -count=1`
Expected: FAIL: `EveryPairingSettles.../PM` and `/MP` (the room read "affects"), `/MM` (nothing); `HarmfulOnAMobIsAnAssault` ("aggression", 0 events); `MobOnMobHarmStartsAFight` (zero ref). `/PP` passes.

- [ ] **Step 3: Create** `internal/hooks/spell_help_effects.go`:

```go
package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
)

// Spell effect unification, parity slice 3b
// (docs/superpowers/specs/2026-09-28-spell-effect-unification-design.md):
// the helpful effects (condition, heal, shield, purge), the default arm and
// the one area-help target filler. Each applier serves every pairing (PM,
// PP, MS, MM, MP) through the spellEffectCtx spell_effects.go defines.

// selfCast reports whether the caster is its own target: a player's
// self-cast, the caster's own place in an area spell, or a mob's MS path.
func (c spellEffectCtx) selfCast() bool {
	return c.caster != nil && c.casterRef() == c.targetRef()
}

// selfCastAudience is the audience for a line about a caster acting on
// itself: the caster's private line (a player caster only) and the room
// line, which excludes a player caster. name is the caster's name exactly as
// the room line prints it.
func (c spellEffectCtx) selfCastAudience(name string) messaging.Audience {
	return spellAudience(c.casterUser(), name, nil, messaging.NoName, c.room)
}

// spellConditionTargetOf is the target as applySpellCondition's event door
// takes it: the player record or the mob, both of which queue the narrating
// events.Condition. Nil for an actor that is neither.
func spellConditionTargetOf(a actions.Actor) spellConditionTarget {
	if u := actorUser(a); u != nil {
		return u
	}
	if m := actorMob(a); m != nil {
		return m
	}
	return nil
}

// spellStatusDefended narrates a defended cast of a binary status effect
// (condition, heal, shield, purge, default) and reports whether it was
// defended. A defended status applies nothing: ExecuteSkillMove's
// StatusApplied split. Help spells are uncontested, so only a contested
// cast (a harmful condition, say) is ever defended.
func spellStatusDefended(c spellEffectCtx) bool {
	if !c.out.Defended {
		return false
	}
	sendSpellChannelDefenceMessages(c.room, c.category(), c.out,
		spellDefenceIdentity(c.casterChar, c.casterUser(), c.room),
		spellDefenceIdentity(c.targetChar(), c.targetUser(), c.room), c.spell.Name, c.casterUser(), c.targetUser())
	return true
}

// applySpellConditionEffect is the one condition applier (slice 3b): every
// condition the spell names goes through applySpellCondition's event door,
// which scales a light or sight by the caster and a heal- or damage-over-time
// by spellTickScale. A harmful condition starts the fight through
// commitHarmfulSpellAggro whether or not the target defended; for a player
// caster on a mob that is also the assault crime (owner ruling 2).
func applySpellConditionEffect(c spellEffectCtx) int {
	fresh := !c.targetChar().IsInCombat()
	if spellStatusDefended(c) {
		if c.spell.IsHarm() {
			commitHarmfulSpellAggro(c, fresh)
		}
		return 0
	}
	// Names are read BEFORE the condition lands: a condition can add an
	// adjective to the target's rendered name, and SendTrio's redaction must
	// see the exact string the line prints.
	casterName, targetName := c.casterName(), c.targetName()
	if target := spellConditionTargetOf(c.target); target != nil {
		for _, conditionId := range c.spell.ConditionIds {
			applySpellCondition(target, c.spell, c.casterChar, conditionId)
		}
	}
	if c.spell.IsHarm() {
		commitHarmfulSpellAggro(c, fresh)
	}
	// KNOWN AND DEFERRED: a condition with authored start text also narrates
	// this moment through the event applySpellCondition queues, so an
	// audience can read it twice. The messaging arc's M6 merges them.
	if c.selfCast() {
		messaging.SendTrio(messaging.Trio{
			Actor: messaging.Say(c.category(), fmt.Sprintf(
				`Your %s takes effect.%s`, c.spell.Name, c.critTag())),
			Actee: messaging.NoLine,
			Observer: messaging.Say(c.category(), fmt.Sprintf(
				`<ansi fg="cyan">%s</ansi> settles over %s.`, c.spell.Name, casterName)),
		}, c.selfCastAudience(casterName))
		return 0
	}
	messaging.SendTrio(messaging.Trio{
		Actor: messaging.Say(c.category(), fmt.Sprintf(
			`Your %s takes effect on %s!%s`, c.spell.Name, targetName, c.critTag())),
		Actee: messaging.Say(c.category(), fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> takes effect on you!%s`, casterName, c.spell.Name, c.critTag())),
		Observer: messaging.Say(c.category(), fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> settles over %s.`, casterName, c.spell.Name, targetName)),
	}, spellAudience(c.casterUser(), casterName, c.targetUser(), targetName, c.room))
	return 0
}
```

- [ ] **Step 4: Dispatch it.** In `spell_effects.go`'s `applySpellEffect`, add to the first switch after the `knockdown` case:

```go
	case "condition":
		return applySpellConditionEffect(c)
```

- [ ] **Step 5: Delete the old condition code** in `spell_resolution.go`:
  - Delete `setMobSpellAggro` with its doc comment (518-532) and `applyMobEffect_condition` (534-579).
  - In `applyMobEffectArms`, delete the `case "condition":` line and its `return` (666-667) and the now-unused `critTag` block (655-658).
  - In `applyPlayerEffectArms`, delete the whole `case "condition":` block (842-888, through the closing `}` of its self-cast `else`).
  - Replace the body of `applyMobOnPlayerArms` (1290-1343, keep its doc comment and signature) with:

```go
	caster, target, room := c.casterMob(), c.targetUser(), c.room
	spellData, out := c.spell, c.out
	if out.Defended {
		sendSpellChannelDefenceMessages(room, spellSchoolCategory(spellData), out,
			spellDefenceIdentity(&caster.Character, nil, room),
			spellDefenceIdentity(target.Character, target, room), spellData.Name, nil, target)
		return 0
	}
	messaging.SendTrio(messaging.Trio{
		Actor: messaging.NoLine,
		Actee: messaging.Say(spellSchoolCategory(spellData), fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> takes effect on you.`,
			caster.Character.Name, spellData.Name)),
		Observer: messaging.NoLine,
	}, spellAudience(nil, caster.Character.Name, target, target.Character.Name, room))
	return 0
```

  - In `resolveAgainstPlayer`, replace the two comment lines above the harm block (712-713) with:

```go
	// Set reciprocal aggro for harm spells. Every applier commits its own
	// except the player-to-player default arm; Task 5 of slice 3b deletes
	// this when applySpellDefaultEffect takes that over.
```

- [ ] **Step 6: Widen the one-contest guard.** In `internal/hooks/channel_defence_routing_test.go`, replace lines 64-66:

```go
	// spell_effects.go holds every applier since parity slice 3a; it must run
	// no contest of its own either.
	for _, name := range []string{"spell_resolution.go", "spell_effects.go"} {
```

with:

```go
	// spell_effects.go (slice 3a) and spell_help_effects.go (slice 3b) hold
	// every applier; neither may run a contest of its own.
	for _, name := range []string{"spell_resolution.go", "spell_effects.go", "spell_help_effects.go"} {
```

Prove it can fail: temporarily add `_ = runSpellChannelAttack(nil, combatvocab.Attack{}, combat.AttackSide{}, nil, nil)` as the first line of `applySpellConditionEffect` (add the `combat` and `combatvocab` imports so it compiles). Run `go test ./internal/hooks/ -run TestSpellResolversRunOneContestAndAppliersRollNone -count=1`. Expected FAIL naming `applySpellConditionEffect: 1`. Remove the line and the two imports with the Edit tool; rerun; expect `ok`.

- [ ] **Step 7: Drop the retired narration row.** In `messaging_surface_guard_test.go`, delete line 1287, the row keyed `"hooks/spell_resolution.go|Your %s takes effect on %s!%s"` (its raw `user.SendText` is gone; the new line goes through `SendTrio`).

- [ ] **Step 8: Run**

```bash
cd C:/tmp/dogmud-3b-helpful && go build ./... && go test ./internal/hooks/ -count=1
```

Expected: `ok`, including `TestSpellCondition_*`, `TestSelfCastCondition_OneLineToCaster_RoomNamesCasterOnce`, `TestCrossCast_*` and `TestWireFreeze_EffectTypeConditionStillApplies`. Run the root guards; re-key the ward and regen rows (shifted). Expected: `ok`, and `TestNarrationSitesMatchViewpointAudit` reports nothing stale or unregistered.

- [ ] **Step 9: Commit**

```bash
cd C:/tmp/dogmud-3b-helpful && git add internal/hooks/spell_help_effects.go internal/hooks/spell_condition_test.go internal/hooks/spell_effects.go internal/hooks/spell_resolution.go internal/hooks/channel_defence_routing_test.go messaging_surface_guard_test.go condition_apply_path_guard_test.go && git commit -F - <<'EOF'
feat(spells): one condition applier for every pairing

applySpellConditionEffect in the new spell_help_effects.go replaces the
player-on-mob, player-on-player and mob-on-player condition arms. Every
pairing narrates through SendTrio with one line set, so a mob's condition
on a mob is seen; a harmful condition starts the fight for both sides
through commitHarmfulSpellAggro, which makes it an assault on a mob
(owner ruling 2). The one-contest guard now parses the new file; the
retired raw condition send leaves the viewpoint registry.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 3: One heal applier (MP gains it; the dead crit goes)

**Model:** sonnet.

**Files:**
- Create: `internal/hooks/spell_heal_test.go`
- Modify: `internal/events/events.go` (after `DrainQueuedGoldGivenForTest`, 405-425), `internal/events/context.md:592-595`
- Modify: `internal/hooks/spell_help_effects.go`, `internal/hooks/spell_effects.go` (dispatcher)
- Modify: `internal/hooks/spell_resolution.go` (delete `applyMobEffect_heal`, the heal cases, the `events` import)
- Modify: `internal/hooks/hooks_test.go` (delete `TestApplyPlayerEffect_HealCrit`, 2718-2732)
- Modify: `condition_apply_path_guard_test.go` (regen rows collapse)

- [ ] **Step 1: Write the failing tests** `internal/hooks/spell_heal_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func healSpellForParityTest() *spells.SpellData {
	return &spells.SpellData{
		SpellId: "test-mend", Name: "Mend", AttackType: combatvocab.AttackNone,
		DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle,
		EffectType: "heal", EffectMagnitude: 3, BaseFolds: 4, PrimaryStat: "willpower",
		Schools: []string{spells.SchoolVital},
	}
}

// parityHealRounds is the heal's duration on the fixture's equalised caster
// (base folds 4, skill 3, willpower 100): half the universal duration,
// floored at six. 4 x (10 + 5 + 1.5) = 66, so 33.
func parityHealRounds() int {
	rounds := calcSpellDuration(4, 3, 100) / 2
	if rounds < 6 {
		rounds = 6
	}
	return rounds
}

// regenRecord returns c's one Regenerating record, or fails the test.
func regenRecord(t *testing.T, c *characters.Character) *conditions.Condition {
	t.Helper()
	recs := c.GetConditions(conditions.ConditionIdRegenerating)
	require.Len(t, recs, 1, "the heal must leave one Regenerating record")
	return recs[0]
}

// Slice 3b (audit row 3): a creature's heal on a player was contested, fell
// to the default arm and applied nothing.
func TestSpellHeal_ACreatureHealsAPlayer(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := healSpellForParityTest()

	resolveMobSpellAgainstPlayer(f.casterMob, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), spell.EffectMagnitude)

	rec := regenRecord(t, f.targetUser.Character)
	assert.Equal(t, 3.0, rec.Magnitude)
	assert.Equal(t, parityHealRounds(), rec.TriggersLeft)
	assert.Equal(t, 1, countContaining(drainPlain(2), "Mend envelops you in healing energy."))
	assert.Equal(t, 1, countContaining(drainPlain(3), "Mend envelops Bobrick in healing light."))
}

// Owner ruling 3: help spells do not crit. The player-to-player heal doubled
// the part of its multiplier above 1x on a crit no cast could reach.
func TestSpellHeal_ACritChangesNothing(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	caster := actions.NewUserActorInRoom(f.casterUser, f.room)
	target := actions.NewUserActorInRoom(f.targetUser, f.room)

	applySpellEffect(newSpellEffectCtx(f.casterUser.Character, caster, target, f.room,
		healSpellForParityTest(), 3, spellContestAttackCrit()))

	assert.Equal(t, 3.0, regenRecord(t, f.targetUser.Character).Magnitude, "a crit must not boost a heal")
}

// A player healing a mob queues events.Healed for the AI companion; nothing
// else does, because the event names a player healer and a mob.
func TestSpellHeal_OnlyAPlayerHealingAMobQueuesHealed(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := healSpellForParityTest()
	events.DrainQueuedHealedForTest(0)

	resolveAgainstMob(f.casterUser, f.targetMob, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), spell.EffectMagnitude)
	healed := events.DrainQueuedHealedForTest(0)
	require.Len(t, healed, 1)
	assert.Equal(t, events.Healed{HealerUserId: 1, MobInstanceId: 101}, healed[0])

	resolveMobSpellAgainstMob(f.casterMob, f.targetMob, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), spell.EffectMagnitude)
	resolveAgainstPlayer(f.casterUser, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), spell.EffectMagnitude)
	assert.Empty(t, events.DrainQueuedHealedForTest(0), "only a player healing a mob is tended")
}
```

- [ ] **Step 2: Run to see them fail**

Run: `cd C:/tmp/dogmud-3b-helpful && go test ./internal/hooks/ -run 'TestSpellHeal_' -count=1`
Expected: build failure, `undefined: events.DrainQueuedHealedForTest`.

- [ ] **Step 3: Add the drain seam** to `internal/events/events.go`, directly after `DrainQueuedGoldGivenForTest` (ends at 425):

```go
// DrainQueuedHealedForTest removes all Healed events for the given healer
// and returns them. Pass 0 to drain every such event.
//
// FOR TEST USE ONLY. Mutates the queue.
func DrainQueuedHealedForTest(healerUserId int) []Healed {
	qLock.Lock()
	defer qLock.Unlock()
	var found []Healed
	remaining := make(priorityQueue, 0, len(globalQueue))
	for _, pe := range globalQueue {
		healed, ok := pe.event.(Healed)
		if !ok || (healerUserId != 0 && healed.HealerUserId != healerUserId) {
			remaining = append(remaining, pe)
			continue
		}
		found = append(found, healed)
	}
	globalQueue = remaining
	heap.Init(&globalQueue)
	return found
}
```

In `internal/events/context.md`, after the sentence ending "Zero drains every\nqueued `Condition` event in either." (595), add:

```markdown
`DrainQueuedHealedForTest(healerUserId)` drains queued `Healed` events the
same way (spell effects slice 3b); zero drains them all.
```

Rerun Step 2's command. Expected: FAIL `ACreatureHealsAPlayer` (no record) and `ACritChangesNothing` (5 is not 3). `OnlyAPlayerHealingAMobQueuesHealed` passes (it pins the event through the move).

- [ ] **Step 4: Add the applier** to `internal/hooks/spell_help_effects.go`. Add `"github.com/GoMudEngine/GoMud/internal/conditions"` and `"github.com/GoMudEngine/GoMud/internal/events"` to its imports, and append:

```go
// applySpellHeal is the one heal applier (slice 3b): a Regenerating record
// on the target at the spell's magnitude as a regen multiplier (floored at
// 1x) for half the universal spell duration (floored at six rounds), read
// from the caster's primarystat and cast skill. Every pairing gains it; a
// mob's heal on a player used to apply nothing (audit row 3). The dead
// player-to-player crit boost is gone (owner ruling 3).
//
// A player healing a mob queues events.Healed, which the AI companion reads
// as "somebody tended me"; the event names a mob, so no other pairing
// queues it.
func applySpellHeal(c spellEffectCtx) int {
	if spellStatusDefended(c) {
		return 0
	}
	stat, skill := spellCasterStatAndSkill(c.spell, c.casterChar)
	regenMult := float64(c.magnitude)
	if regenMult < 1.0 {
		regenMult = 1.0
	}
	durationRounds := calcSpellDuration(c.spell.BaseFolds, skill, stat) / 2
	if durationRounds < 6 {
		durationRounds = 6
	}
	casterName, targetName := c.casterName(), c.targetName()
	if u, m := c.casterUser(), c.targetMob(); u != nil && m != nil {
		events.AddToQueue(events.Healed{HealerUserId: u.UserId, MobInstanceId: m.InstanceId})
	}
	_ = c.targetChar().AddConditionMagnitude(conditions.ConditionIdRegenerating, durationRounds, regenMult, "heal spell")
	if c.selfCast() {
		messaging.SendTrio(messaging.Trio{
			Actor: messaging.Say(messaging.CategorySpellVital,
				`<ansi fg="green">A warm glow of healing magic envelops you. Your wounds begin to mend.</ansi>`),
			Actee: messaging.NoLine,
			Observer: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
				`%s channels restorative magic.`, casterName)),
		}, c.selfCastAudience(casterName))
		return 0
	}
	messaging.SendTrio(messaging.Trio{
		Actor: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
			`<ansi fg="green">You weave restorative magic around %s.</ansi>`, targetName)),
		Actee: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
			`<ansi fg="green">%s's %s envelops you in healing energy. Your wounds begin to mend.</ansi>`,
			casterName, c.spell.Name)),
		Observer: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> envelops %s in healing light.`, casterName, c.spell.Name, targetName)),
	}, spellAudience(c.casterUser(), casterName, c.targetUser(), targetName, c.room))
	return 0
}
```

- [ ] **Step 5: Dispatch it.** In `applySpellEffect`'s first switch, after the `condition` case:

```go
	case "heal":
		return applySpellHeal(c)
```

- [ ] **Step 6: Delete the old heal code** in `spell_resolution.go`:
  - `applyMobEffect_heal` with its doc comment (581-620 on the base; find it by name).
  - In `applyMobEffectArms`, the `case "heal":` arm (the `if user != nil { events.AddToQueue(...) }` and its `return`), and change `spellData, magnitude, out := c.spell, c.magnitude, c.out` to `spellData, out := c.spell, c.out`.
  - In `applyPlayerEffectArms`, the whole `case "heal":` block (794-840 on the base).
  - The `"github.com/GoMudEngine/GoMud/internal/events"` import (its only use was the heal arm, F19).
  - In `hooks_test.go`, delete `TestApplyPlayerEffect_HealCrit` (2718-2732): `TestSpellHeal_ACritChangesNothing` replaces it with an assertion.

- [ ] **Step 7: Run**

```bash
cd C:/tmp/dogmud-3b-helpful && go build ./... && go test ./internal/hooks/ ./internal/events/ -count=1
```

Expected: `ok`, including `TestSpellHeal_*`, `TestSelfCastHeal_*`, `TestAreaHeal_CasterIsTheirOwnTarget`, `TestCrossCastHeal_*` and `TestNonHarmCastAtAMobRunsNoContest`.

Root guards: the regen rows `spell_resolution.go|615` and `|809` (base numbering; shifted by Tasks 1-2) are gone and one new `internal/hooks/spell_help_effects.go|N` key appears. **Collapse** the two stale rows into that one new key, same reason string; re-key the surviving `spell_resolution.go` regen row (MS) and the two ward rows. Expected: `ok`.

- [ ] **Step 8: Commit**

```bash
cd C:/tmp/dogmud-3b-helpful && git add internal/hooks/spell_heal_test.go internal/events/events.go internal/events/context.md internal/hooks/spell_help_effects.go internal/hooks/spell_effects.go internal/hooks/spell_resolution.go internal/hooks/hooks_test.go condition_apply_path_guard_test.go && git commit -F - <<'EOF'
feat(spells): one heal applier; creatures heal players

applySpellHeal replaces the player-on-mob, mob-on-mob and
player-on-player heal arms, so a mob's heal on a player finally lands
(audit row 3). The unreachable player-to-player crit boost is deleted
(owner ruling 3). events.Healed still fires for a player healing a mob;
DrainQueuedHealedForTest lets a test see it. Two regen allowlist rows
collapse into one.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 4: One shield applier (pets and players from creatures gain it)

**Model:** sonnet.

**Files:**
- Create: `internal/hooks/spell_shield_test.go`
- Modify: `internal/hooks/spell_help_effects.go`, `internal/hooks/spell_effects.go` (dispatcher)
- Modify: `internal/hooks/spell_resolution.go` (delete the PP shield case)
- Modify: `condition_apply_path_guard_test.go` (ward row moves)

- [ ] **Step 1: Write the failing tests** `internal/hooks/spell_shield_test.go`:

```go
package hooks

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shieldSpellForParityTest() *spells.SpellData {
	return &spells.SpellData{
		SpellId: "test-ward", Name: "Ward", AttackType: combatvocab.AttackNone,
		DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle,
		EffectType: "shield", EffectMagnitude: 75, BaseFolds: 4, PrimaryStat: "willpower",
		Schools: []string{spells.SchoolEnhancement},
	}
}

// parityShieldBonus is the shield's strength on the fixture's equalised
// caster, from the fixture's numbers: willpower 100 plus spellcasting 3
// times SkillWeight, a third of that, scaled by the spell's magnitude 75.
// Test binaries read the Go default SkillWeight, not the shipped one.
func parityShieldBonus() float64 {
	weighted := int(math.Round(3 * float64(configs.GetBalanceConfig().SkillWeight)))
	return float64(int(math.Round(float64((100+weighted)/3) * 75 / 100.0)))
}

// shieldRecord returns c's one Minor Shield record, or fails the test.
func shieldRecord(t *testing.T, c *characters.Character) *conditions.Condition {
	t.Helper()
	recs := c.GetConditions(conditions.ConditionIdMinorShield)
	require.Len(t, recs, 1, "the shield must leave one Minor Shield record")
	return recs[0]
}

// Slice 3b (audit row 15): a shield on a charmed pet applied nothing, since
// a mob target had no shield arm.
func TestSpellShield_APetCanBeShielded(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	f.targetMob.Character.Charm(1, -1, "")
	spell := shieldSpellForParityTest()

	resolveAgainstMob(f.casterUser, f.targetMob, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), spell.EffectMagnitude)

	rec := shieldRecord(t, &f.targetMob.Character)
	assert.Equal(t, parityShieldBonus(), rec.Magnitude)
	assert.Equal(t, calcSpellDuration(4, 3, 100), rec.TriggersLeft)
	assert.Equal(t, 1, countContaining(drainPlain(3), "A shimmering barrier surrounds Ghoul"))
}

// Slice 3b (audit row 3): a creature's shield on a player applied nothing.
func TestSpellShield_ACreatureShieldsAPlayer(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := shieldSpellForParityTest()

	resolveMobSpellAgainstPlayer(f.casterMob, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), spell.EffectMagnitude)

	assert.Equal(t, parityShieldBonus(), shieldRecord(t, f.targetUser.Character).Magnitude)
	assert.Equal(t, 1, countContaining(drainPlain(2), "A shimmering magical barrier forms around you"))
}

// Owner ruling 3: a shield does not crit. The player-to-player arm multiplied
// it by 1.5 on a crit no cast could reach.
func TestSpellShield_ACritChangesNothing(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	caster := actions.NewUserActorInRoom(f.casterUser, f.room)
	target := actions.NewUserActorInRoom(f.targetUser, f.room)

	applySpellEffect(newSpellEffectCtx(f.casterUser.Character, caster, target, f.room,
		shieldSpellForParityTest(), 75, spellContestAttackCrit()))

	assert.Equal(t, parityShieldBonus(), shieldRecord(t, f.targetUser.Character).Magnitude,
		"a crit must not strengthen a shield")
}
```

- [ ] **Step 2: Run to see them fail**

Run: `cd C:/tmp/dogmud-3b-helpful && go test ./internal/hooks/ -run 'TestSpellShield_' -count=1`
Expected: FAIL `APetCanBeShielded` and `ACreatureShieldsAPlayer` ("one Minor Shield record", 0), `ACritChangesNothing` (1.5 times the bonus).

- [ ] **Step 3: Add the applier** to `spell_help_effects.go`. Add `"math"` and `"github.com/GoMudEngine/GoMud/internal/configs"` to its imports, and append:

```go
// applySpellShield is the one shield applier (slice 3b): a Minor Shield
// record on the target worth a third of the caster's primarystat plus its
// weighted cast skill (at least one), scaled by the spell's magnitude (100
// is 1x), for the full universal spell duration. Every pairing gains it: a
// shield on a charmed pet, or from a creature onto a player, applied
// nothing (audit rows 3 and 15). The dead player-to-player crit bump is
// gone (owner ruling 3).
func applySpellShield(c spellEffectCtx) int {
	if spellStatusDefended(c) {
		return 0
	}
	stat, skill := spellCasterStatAndSkill(c.spell, c.casterChar)
	weightedSkill := int(math.Round(float64(skill) * float64(configs.GetBalanceConfig().SkillWeight)))
	shieldBonus := (stat + weightedSkill) / 3
	if shieldBonus < 1 {
		shieldBonus = 1
	}
	// Scale shield strength by spell magnitude (100 = 1.0x baseline).
	if c.magnitude > 0 {
		shieldBonus = int(math.Round(float64(shieldBonus) * float64(c.magnitude) / 100.0))
		if shieldBonus < 1 {
			shieldBonus = 1
		}
	}
	duration := calcSpellDuration(c.spell.BaseFolds, skill, stat)
	casterName, targetName := c.casterName(), c.targetName()
	_ = c.targetChar().AddConditionMagnitude(conditions.ConditionIdMinorShield, duration, float64(shieldBonus), "spell")
	if c.selfCast() {
		messaging.SendTrio(messaging.Trio{
			Actor: messaging.Say(c.category(),
				`A shimmering magical barrier forms around you, bolstering your defenses.`),
			Actee: messaging.NoLine,
			Observer: messaging.Say(c.category(), fmt.Sprintf(
				`A shimmering barrier surrounds %s.`, casterName)),
		}, c.selfCastAudience(casterName))
		return 0
	}
	messaging.SendTrio(messaging.Trio{
		Actor: messaging.Say(c.category(), fmt.Sprintf(
			`A shimmering magical barrier forms around %s, bolstering their defenses.`, targetName)),
		Actee: messaging.Say(c.category(),
			`A shimmering magical barrier forms around you, bolstering your defenses.`),
		Observer: messaging.Say(c.category(), fmt.Sprintf(
			`A shimmering barrier surrounds %s.`, targetName)),
	}, spellAudience(c.casterUser(), casterName, c.targetUser(), targetName, c.room))
	return 0
}
```

- [ ] **Step 4: Dispatch it.** In `applySpellEffect`'s first switch, after `heal`:

```go
	case "shield":
		return applySpellShield(c)
```

- [ ] **Step 5: Delete the PP shield case** in `applyPlayerEffectArms` (890-936 on the base, the whole `case "shield":` block), and change that function's `spellData, magnitude, out := c.spell, c.magnitude, c.out` to `spellData, out := c.spell, c.out`. `configs` and `math` stay imported in `spell_resolution.go` (`applyMobSelfEffect` and `calcSpellDuration` still use them).

- [ ] **Step 6: Run**

```bash
cd C:/tmp/dogmud-3b-helpful && go build ./... && go test ./internal/hooks/ -count=1
```

Expected: `ok`, including `TestSpellShield_*`, `TestApplyPlayerEffect_Shield`, `TestApplyPlayerEffect_ShieldSelf` and `TestSelfCastShield*` in `selfcast_spell_room_hiding_test.go`.

Root guards: the ward row for the old PP call is stale and a `spell_help_effects.go|N` ward key appears; move the row to the new key (same reason string); re-key the MS ward row and the regen rows. Expected: `ok`.

- [ ] **Step 7: Commit**

```bash
cd C:/tmp/dogmud-3b-helpful && git add internal/hooks/spell_shield_test.go internal/hooks/spell_help_effects.go internal/hooks/spell_effects.go internal/hooks/spell_resolution.go condition_apply_path_guard_test.go && git commit -F - <<'EOF'
feat(spells): one shield applier; pets and players can be shielded

applySpellShield replaces the player-to-player shield arm and serves
every pairing, so a shield on a charmed pet (audit row 15) and a
creature's shield on a player (audit row 3) now hold. The unreachable
x1.5 crit bump is deleted (owner ruling 3).

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 5: Purge, the default arm, charm dispatch; delete the per-pairing arms

**Model:** opus.

**Files:**
- Create: `internal/hooks/spell_purge_test.go`
- Modify: `internal/hooks/spell_help_effects.go`, `internal/hooks/spell_effects.go` (final dispatcher)
- Modify: `internal/hooks/spell_resolution.go` (delete `applyMobEffect_default`, `applyMobEffectArms`, `applyPlayerEffectArms`, `applyMobOnPlayerArms`, `resolveAgainstPlayer`'s harm block; `spellNarratedByGoHook` comment)
- Modify: `internal/hooks/selfcast_wording_test.go` (243-263)
- Modify: `condition_apply_path_guard_test.go` (re-key)

- [ ] **Step 1: Write the failing tests** `internal/hooks/spell_purge_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func purgeSpellForParityTest() *spells.SpellData {
	return &spells.SpellData{
		SpellId: "test-cleanse", Name: "Cleanse", AttackType: combatvocab.AttackNone,
		DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle,
		EffectType: "purge", BaseFolds: 4, PrimaryStat: "willpower",
		Schools: []string{spells.SchoolVital},
	}
}

// poisonForPurgeTest gives c one spell-poison record.
func poisonForPurgeTest(t *testing.T, c *characters.Character) {
	t.Helper()
	require.NoError(t, c.AddConditionMagnitude(conditions.ConditionIdPoisoned, 10, -5, "test"))
}

func poisoned(c *characters.Character) bool {
	return len(c.GetConditions(conditions.ConditionIdPoisoned)) > 0
}

// Slice 3b: Cleansing Wave over a companion said it took effect and cleansed
// nothing, and so did a creature's cleanse on a player.
func TestSpellPurge_CompanionsAndPlayersAreCleansed(t *testing.T) {
	t.Run("PM", func(t *testing.T) {
		f := newSpellParityFixture(t, spellContestAttackWin())
		spell := purgeSpellForParityTest()
		poisonForPurgeTest(t, &f.targetMob.Character)

		resolveAgainstMob(f.casterUser, f.targetMob, f.room, spell,
			spellAttackSideFor(spell, f.casterUser.Character, nil), 0)

		assert.False(t, poisoned(&f.targetMob.Character), "the companion is cleansed")
		assert.Equal(t, 1, countContaining(drainPlain(1), "Your Cleanse cleanses Ghoul"))
	})
	t.Run("MP", func(t *testing.T) {
		f := newSpellParityFixture(t, spellContestAttackWin())
		spell := purgeSpellForParityTest()
		poisonForPurgeTest(t, f.targetUser.Character)

		resolveMobSpellAgainstPlayer(f.casterMob, f.targetUser, f.room, spell,
			spellAttackSideFor(spell, &f.casterMob.Character, nil), 0)

		assert.False(t, poisoned(f.targetUser.Character), "the player is cleansed")
		assert.Equal(t, 1, countContaining(drainPlain(2), "Cleanse purges the toxins from your body."))
	})
}

// Slice 3b: a spell with no effect of its own tells both sides. The
// player-to-player default arm told the target nothing.
func TestSpellDefault_TheTargetIsTold(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := &spells.SpellData{SpellId: "test-curiosity", Name: "Curiosity", AttackType: combatvocab.AttackNone,
		DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle, PrimaryStat: "willpower"}

	resolveAgainstPlayer(f.casterUser, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), 0)

	assert.Equal(t, 1, countContaining(drainPlain(1), "Your Curiosity takes effect on Bobrick."))
	assert.Equal(t, 1, countContaining(drainPlain(2), "Curiosity takes effect on you."))
}
```

- [ ] **Step 2: Run to see them fail**

Run: `cd C:/tmp/dogmud-3b-helpful && go test ./internal/hooks/ -run 'TestSpellPurge_|TestSpellDefault_' -count=1`
Expected: FAIL `PM` and `MP` (still poisoned) and `TheTargetIsTold` (target read nothing).

- [ ] **Step 3: Add the two appliers** to `spell_help_effects.go`:

```go
// applySpellPurge is the one purge applier (slice 3b): it cancels every
// poison on the target. A mob target gains it: Cleansing Wave over a
// charmed companion said it took effect and cleansed nothing. The
// Go-hooked Purge Affliction spell is a different path
// (resolvePurgeAffliction, spell_purgeaffliction.go) and never comes here.
func applySpellPurge(c spellEffectCtx) int {
	if spellStatusDefended(c) {
		return 0
	}
	// Names are read BEFORE the cure: cancelling the poison can drop an
	// adjective from the target's rendered name.
	casterName, targetName := c.casterName(), c.targetName()
	c.targetChar().CancelConditionsWithFlag(conditions.Poison)
	if c.selfCast() {
		messaging.SendTrio(messaging.Trio{
			Actor: messaging.Say(messaging.CategorySpellVital,
				`<ansi fg="green">You purge the afflictions from your body.</ansi>`),
			Actee: messaging.NoLine,
			Observer: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
				`<ansi fg="cyan">%s</ansi> cleanses %s of afflictions.`, c.spell.Name, casterName)),
		}, c.selfCastAudience(casterName))
		return 0
	}
	messaging.SendTrio(messaging.Trio{
		Actor: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
			`<ansi fg="green">Your %s cleanses %s of afflictions.</ansi>`, c.spell.Name, targetName)),
		Actee: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
			`<ansi fg="green">%s's %s purges the toxins from your body.</ansi>`, casterName, c.spell.Name)),
		Observer: messaging.Say(messaging.CategorySpellVital, fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> cleanses %s.`, casterName, c.spell.Name, targetName)),
	}, spellAudience(c.casterUser(), casterName, c.targetUser(), targetName, c.room))
	return 0
}

// applySpellDefaultEffect is the arm for an effect with no applier of its
// own (slice 3b): an unknown or empty effect_type, and charm on anything but
// a mob. It applies nothing and says so to both sides. A harmful spell still
// starts the fight (commitHarmfulSpellAggro), as resolveAgainstPlayer used
// to do for every harmful spell. A spell whose narration a Go hook in
// resolveSpell owns (spellNarratedByGoHook) gets no line here.
func applySpellDefaultEffect(c spellEffectCtx) int {
	fresh := !c.targetChar().IsInCombat()
	if spellStatusDefended(c) {
		if c.spell.IsHarm() {
			commitHarmfulSpellAggro(c, fresh)
		}
		return 0
	}
	if c.spell.IsHarm() {
		commitHarmfulSpellAggro(c, fresh)
	}
	if spellNarratedByGoHook(c.spell.SpellId) {
		return 0
	}
	casterName, targetName := c.casterName(), c.targetName()
	if c.selfCast() {
		messaging.SendTrio(messaging.Trio{
			Actor:    messaging.Say(c.category(), fmt.Sprintf(`Your %s takes effect.`, c.spell.Name)),
			Actee:    messaging.NoLine,
			Observer: messaging.NoLine,
		}, c.selfCastAudience(casterName))
		return 0
	}
	messaging.SendTrio(messaging.Trio{
		Actor: messaging.Say(c.category(), fmt.Sprintf(
			`Your %s takes effect on %s.`, c.spell.Name, targetName)),
		Actee: messaging.Say(c.category(), fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> takes effect on you.`, casterName, c.spell.Name)),
		Observer: messaging.NoLine,
	}, spellAudience(c.casterUser(), casterName, c.targetUser(), targetName, c.room))
	return 0
}
```

- [ ] **Step 4: The final dispatcher.** In `spell_effects.go`, replace `applySpellEffect` with its doc comment (158-179 on the base, plus the `condition`, `heal` and `shield` cases Tasks 2-4 added) by:

```go
// applySpellEffect applies one spell effect to one target, whoever casts it
// and whoever it hits, and returns the damage it dealt (0 for effects that
// deal none). Each effect has one applier (spell_effects.go for the harmful
// ones, spell_help_effects.go for the rest). Charm binds only a mob, and
// applyMobEffect_charm refuses a caster that is not a player; charm on
// anything else, and any effect with no applier, is the default arm.
func applySpellEffect(c spellEffectCtx) int {
	switch c.spell.EffectType {
	case "damage":
		return applySpellDamage(c)
	case "dot":
		return applySpellDot(c)
	case "knockdown":
		return applySpellKnockdown(c)
	case "condition":
		return applySpellConditionEffect(c)
	case "heal":
		return applySpellHeal(c)
	case "shield":
		return applySpellShield(c)
	case "purge":
		return applySpellPurge(c)
	case "charm":
		if m := c.targetMob(); m != nil {
			return applyMobEffect_charm(c.casterUser(), m, c.room, c.spell, c.out, c.targetName())
		}
	}
	return applySpellDefaultEffect(c)
}
```

- [ ] **Step 5: Delete the arms** in `spell_resolution.go`: `applyMobEffect_default` (base 622-644), `applyMobEffectArms` with its doc comment (base 646-681), `applyPlayerEffectArms` with its doc comment (base 730-977), and `applyMobOnPlayerArms` with its doc comment (base 1286-1344). Delete `resolveAgainstPlayer`'s harm-aggro block (the three comment lines Task 2 wrote and the `if spellData.IsHarm() { ... }` block under them). In `spellNarratedByGoHook`'s doc comment, replace `told twice, once by applyPlayerEffect's default\n// arm and once by its hook.` with `told twice, once by applySpellDefaultEffect\n// and once by its hook.`

Imports: `conditions`, `configs`, `math`, `skills` stay (`applyMobSelfEffect`, `calcSpellDuration`, `spellAttackSideFor`). `go build ./internal/hooks/` must report nothing; if it reports an unused import, remove exactly that one.

- [ ] **Step 6: Migrate the two `applyMobEffect_default` calls** in `selfcast_wording_test.go` (`TestHookSpellOnCompanion_NoGenericLine`, 243-263). Add `"github.com/GoMudEngine/GoMud/internal/actions"` and `"github.com/GoMudEngine/GoMud/internal/mobs"` to its imports. Replace the doc comment's `reaches\n// applyMobEffect_default, which also told the caster` with `reaches\n// the default arm (applySpellDefaultEffect), which also told the caster`, and replace the body from `	applyMobEffect_default(u, u.Character, room,` through the final assertion with:

```go
	mob := mobs.GetInstance(100)
	applySpellEffect(newSpellEffectCtx(u.Character, actions.NewUserActorInRoom(u, room),
		actions.NewMobActorInRoom(mob, room), room,
		&spells.SpellData{SpellId: "purge-affliction", Name: "Purge Affliction"}, 0, spellContestAttackWin()))
	assert.Equal(t, 0, countContaining(drainPlain(1), "takes effect"),
		"a spell narrated by its Go hook must not also get the generic line")

	// Control: a spell with no hook still gets the generic line.
	applySpellEffect(newSpellEffectCtx(u.Character, actions.NewUserActorInRoom(u, room),
		actions.NewMobActorInRoom(mob, room), room,
		&spells.SpellData{SpellId: "curiosity", Name: "Curiosity"}, 0, spellContestAttackWin()))
	assert.Equal(t, 1, countContaining(drainPlain(1), "Your Curiosity takes effect on Skeleton."))
```

- [ ] **Step 7: Run**

```bash
cd C:/tmp/dogmud-3b-helpful && go build ./... && go test ./internal/hooks/ -count=1
```

Expected: `ok`, including `TestSpellPurge_*`, `TestSpellDefault_*`, `TestSelfCastPurge_*`, `TestAreaPurge_*`, `TestSelfCastDefault_*`, `TestSelfCastPurgeAffliction_*`, `TestHookSpellOnCompanion_NoGenericLine`, `TestMobCastOnPlayer_TargetInTheDarkReadsSomething`, every `charm_*` test. Root guards: re-key the shifted rows. Expected: `ok`.

Confirm the arms are gone (standalone; grep exits 1 on zero, which is the pass):

```bash
cd C:/tmp/dogmud-3b-helpful && grep -rn "applyMobEffectArms\|applyPlayerEffectArms\|applyMobOnPlayerArms\|applyMobEffect_default\|applyMobEffect_heal\|applyMobEffect_condition\|setMobSpellAggro" --include=*.go .
```

Expected: no output. Prove the pattern can match: `grep -rn "applyMobEffect_charm" --include=*.go internal/hooks | head -1` prints a line.

- [ ] **Step 8: Commit**

```bash
cd C:/tmp/dogmud-3b-helpful && git add internal/hooks/spell_purge_test.go internal/hooks/spell_help_effects.go internal/hooks/spell_effects.go internal/hooks/spell_resolution.go internal/hooks/selfcast_wording_test.go condition_apply_path_guard_test.go && git commit -F - <<'EOF'
feat(spells): purge and default appliers; the per-pairing arms are gone

applySpellPurge cleanses a companion or a player a creature casts on;
applySpellDefaultEffect tells both sides and starts the fight for a
harmful spell, taking over resolveAgainstPlayer's harm block. The
dispatcher now names one applier per effect plus charm, and
applyMobEffectArms, applyPlayerEffectArms, applyMobOnPlayerArms and
applyMobEffect_default are deleted.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 6: A mob casting on itself (MS) goes through the shared step

**Model:** sonnet.

**Files:**
- Create: `internal/hooks/spell_selfcast_test.go`
- Modify: `internal/hooks/spell_resolution.go` (`resolveMobSpell` doc 1013-1014 and self branch 1059-1064; delete `applyMobSelfEffect`, the `conditions` and `configs` imports)
- Modify: `internal/hooks/spell_effects.go` (the `applyMobSelfEffect` test-only wrapper)
- Modify: `condition_apply_path_guard_test.go` (MS rows deleted)

- [ ] **Step 1: Write the failing tests** `internal/hooks/spell_selfcast_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spellHelpParityPairings adds MS, a mob casting on itself through
// resolveMobSpell's real self branch, to the four contested pairings.
func spellHelpParityPairings() []spellParityPairing {
	mobCaster := func(f *spellParityFixture) *characters.Character { return &f.casterMob.Character }
	return append(spellParityPairings(), spellParityPairing{
		name: "MS", src: combat.Mob, tgt: combat.Mob, caster: mobCaster, target: mobCaster,
		casterRef: func(f *spellParityFixture) state.ActorRef {
			return state.ActorRef{MobInstanceId: f.casterMob.InstanceId}
		},
		cast: func(f *spellParityFixture, s *spells.SpellData) {
			resolveMobSpell(f.casterMob, activity.CastingData{SpellId: s.SpellId,
				TargetMobInstanceIds: []int{f.casterMob.InstanceId}}, s, f.room)
		},
	})
}

// Slice 3b: a mob casting on itself takes the shared step and appliers. MS
// recorded nothing, had no purge, and its shield line read differently from
// a player's ("forms around").
func TestSpellSelfCast_AMobOnItselfIsSeenCleansedAndRecorded(t *testing.T) {
	ms := spellHelpParityPairings()[4]
	t.Run("shield", func(t *testing.T) {
		f := newSpellParityFixture(t, spellContestAttackWin())

		ms.cast(f, shieldSpellForParityTest())

		assert.Equal(t, 1, countContaining(drainPlain(3), "A shimmering barrier surrounds Skeleton"))
		require.Len(t, f.records, 1, "a self-cast is recorded")
		assert.Equal(t, combat.Mob, f.records[0].src)
		assert.True(t, f.records[0].hit)
	})
	t.Run("purge", func(t *testing.T) {
		f := newSpellParityFixture(t, spellContestAttackWin())
		poisonForPurgeTest(t, &f.casterMob.Character)

		ms.cast(f, purgeSpellForParityTest())

		assert.False(t, poisoned(&f.casterMob.Character), "the mob cleanses itself")
	})
}

// A mob never harms itself: a harmful spell that finds the caster in its own
// target list applies nothing to it. This passes before and after the
// change; it pins the guard the new self branch carries.
func TestSpellSelfCast_AMobNeverHarmsItself(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())

	spellHelpParityPairings()[4].cast(f, physicalHarmSpellForCollapseTest())

	assert.Equal(t, 1000, f.casterMob.Character.Health)
	assert.Empty(t, f.records)
}
```

- [ ] **Step 2: Run to see them fail**

Run: `cd C:/tmp/dogmud-3b-helpful && go test ./internal/hooks/ -run 'TestSpellSelfCast_' -count=1`
Expected: FAIL `shield` (the room read "forms around", no record) and `purge` (still poisoned). `AMobNeverHarmsItself` passes.

- [ ] **Step 3: The MS branch.** In `resolveMobSpell`, replace the self branch (1060-1064):

```go
		if mobInstId == mob.InstanceId {
			// Self-cast (HelpSingle with self target)
			applyMobSelfEffect(mob, room, spellData, magnitude)
			continue
		}
```

with:

```go
		if mobInstId == mob.InstanceId {
			// MS: the caster is its own target. Only a help spell puts a mob
			// in its own list (a HelpSingle with no target, or its own place
			// in an area help), and it takes the same uncontested step and
			// appliers as every other pairing. A mob never harms itself.
			if !spellData.IsHarm() {
				self := actions.NewMobActorInRoom(mob, room)
				anyLanded = resolveHelpSpell(newSpellEffectCtx(&mob.Character, self, self, room, spellData,
					magnitude, uncontestedSpellResult())) || anyLanded
			}
			continue
		}
```

In `resolveMobSpell`'s doc comment, replace:

```go
//   - Mob targets include a self-cast branch (applyMobSelfEffect) for help
//     spells; player casters never self-target via this dispatcher.
```

with:

```go
//   - Mob targets include a self-cast branch (MS) for help spells, through
//     resolveHelpSpell; a player's self-cast arrives as a player target.
```

- [ ] **Step 4: Delete `applyMobSelfEffect`** from `spell_resolution.go` (base 1178-1217, with its doc comment), then the `"github.com/GoMudEngine/GoMud/internal/conditions"` and `"github.com/GoMudEngine/GoMud/internal/configs"` imports (no use remains, F19). In `spell_effects.go`'s test-only wrapper section (after `applyPlayerEffect`), add the wrapper the tests still name until Task 9:

```go
// applyMobSelfEffect applies a mob's spell to itself (MS).
func applyMobSelfEffect(mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, magnitude int) {
	self := actions.NewMobActorInRoom(mob, room)
	applySpellEffect(newSpellEffectCtx(&mob.Character, self, self, room, spellData, magnitude,
		uncontestedSpellResult()))
}
```

- [ ] **Step 5: Run**

```bash
cd C:/tmp/dogmud-3b-helpful && go build ./... && go test ./internal/hooks/ -count=1
```

Expected: `ok`, including `TestSpellSelfCast_*`, `TestApplyMobSelfEffect_Heal`, `_Shield`, `TestConditionSpell_MobSelfCastHealsAtCasterScale`, `TestWireFreeze_*/MobSelfCast*` and `TestResolveMobSpell_SelfCast`.

Root guards: the MS regen and ward rows (`spell_resolution.go`, the `applyMobSelfEffect` calls) are stale with no replacement (MS now reaches the rows in `spell_help_effects.go`). **Delete** both rows. Re-key anything else that shifted. Expected: `ok`.

- [ ] **Step 6: Commit**

```bash
cd C:/tmp/dogmud-3b-helpful && git add internal/hooks/spell_selfcast_test.go internal/hooks/spell_resolution.go internal/hooks/spell_effects.go condition_apply_path_guard_test.go && git commit -F - <<'EOF'
feat(spells): a mob casting on itself uses the shared appliers

resolveMobSpell's self branch takes resolveHelpSpell, so a mob's
self-cast is recorded, can purge, and reads like a player's. It never
applies a harmful spell to the caster. applyMobSelfEffect's switch is
deleted; a thin test-only wrapper remains until the tests move.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 7: One area-help target filler

**Model:** opus.

**Files:**
- Create: `internal/hooks/spell_help_area_test.go`
- Modify: `internal/hooks/spell_help_effects.go`
- Modify: `internal/hooks/spell_resolution.go` (`resolveSpell` doc 51, filler 105-120; `resolveMobSpell` after 1057)

- [ ] **Step 1: Write the failing tests** `internal/hooks/spell_help_area_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func massMendSpellForAreaTest() *spells.SpellData {
	return &spells.SpellData{
		SpellId: "test-mass-mend", Name: "Mass Mend", AttackType: combatvocab.AttackNone,
		DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetArea,
		EffectType: "heal", EffectMagnitude: 3, BaseFolds: 4, PrimaryStat: "willpower",
		Schools: []string{spells.SchoolVital},
	}
}

func holdsRegen(c *characters.Character) bool {
	return len(c.GetConditions(conditions.ConditionIdRegenerating)) > 0
}

// addRatForAreaTest puts a third mob, Rat (instance 102), in the fixture room.
func addRatForAreaTest(t *testing.T, f *spellParityFixture) *mobs.Mob {
	t.Helper()
	rat := &mobs.Mob{MobId: 3, InstanceId: 102, HomeRoomId: 1, Character: characters.Character{
		Name: "Rat", RoomId: 1, MobInstanceId: 102,
		Conditions: conditions.New(), Cooldowns: map[string]int{},
		Position: position.NewMachine(), CombatPhase: combatphase.NewMachine(),
	}}
	equaliseSpellCombatant(&rat.Character)
	mobs.SetInstanceForTest(102, rat)
	t.Cleanup(func() { mobs.SetInstanceForTest(102, nil) })
	f.room.AddMob(102)
	return rat
}

// partyForAreaTest makes Aliceia (1) lead a party Bobrick (2) has joined;
// Carys (3) stays outside it.
func partyForAreaTest(t *testing.T) {
	t.Helper()
	p := parties.New(1)
	require.NotNil(t, p)
	require.True(t, p.InvitePlayer(2))
	require.True(t, p.AcceptInvite(2))
	t.Cleanup(p.Disband)
}

// Slice 3b (audit row 13, owner ruling 2026-09-28): a player's area heal
// lands on every player in the room and on mobs charmed by the caster or a
// party member, not on a stranger's pet. It used to take any charmed mob.
func TestHelpArea_PlayerHealsThePartysCompanionsNotAStrangersPet(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	partyForAreaTest(t)
	rat := addRatForAreaTest(t, f)
	rat.Character.Charm(1, -1, "")         // Aliceia's own pet
	f.targetMob.Character.Charm(2, -1, "") // Bobrick's companion, a party member's
	f.casterMob.Character.Charm(3, -1, "") // Carys's pet; Carys is not in the party
	spell := massMendSpellForAreaTest()

	resolveSpell(f.casterUser, activity.CastingData{SpellId: spell.SpellId}, spell, f.room)

	assert.True(t, holdsRegen(&rat.Character), "the caster's own pet is healed")
	assert.True(t, holdsRegen(&f.targetMob.Character), "a party member's companion is healed")
	assert.False(t, holdsRegen(&f.casterMob.Character), "a stranger's pet is not")
	for _, u := range []*users.UserRecord{f.casterUser, f.targetUser, f.watcher} {
		assert.True(t, holdsRegen(u.Character), "%s is a player in the room", u.Character.Name)
	}
}

// An uncharmed mob's area heal lands on itself and its packmates, the mobs
// its own AI would heal (mobs.FindPackmatesInRoom), and never on a player or
// a mob outside the pack.
func TestHelpArea_AMobHealsItsPackAndNoPlayer(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	rat := addRatForAreaTest(t, f)
	f.casterMob.Routine = "crypt"
	f.targetMob.Routine = "crypt"
	spell := massMendSpellForAreaTest()

	// InitiateCast fills a mob's area help with every player and mob in the
	// room (actions/cast.go); resolution must narrow it.
	resolveMobSpell(f.casterMob, activity.CastingData{SpellId: spell.SpellId,
		TargetUserIds: []int{1, 2, 3}, TargetMobInstanceIds: []int{100, 101, 102}}, spell, f.room)

	assert.True(t, holdsRegen(&f.casterMob.Character), "the caster heals itself")
	assert.True(t, holdsRegen(&f.targetMob.Character), "a packmate is healed")
	assert.False(t, holdsRegen(&rat.Character), "a mob outside the pack is not")
	for _, u := range []*users.UserRecord{f.casterUser, f.targetUser, f.watcher} {
		assert.False(t, holdsRegen(u.Character), "%s is not the mob's ally", u.Character.Name)
	}
}

// A charmed mob stands on its owner's side: its area heal lands as its
// owner's would, and on itself; a stranger's pet is left out.
func TestHelpArea_ACharmedMobHealsItsOwnersSide(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	partyForAreaTest(t)
	rat := addRatForAreaTest(t, f)
	f.casterMob.Character.Charm(1, -1, "") // Aliceia's pet casts
	f.targetMob.Character.Charm(2, -1, "") // Bobrick's companion
	rat.Character.Charm(3, -1, "")         // Carys's pet
	spell := massMendSpellForAreaTest()

	resolveMobSpell(f.casterMob, activity.CastingData{SpellId: spell.SpellId,
		TargetUserIds: []int{1, 2, 3}, TargetMobInstanceIds: []int{100, 101, 102}}, spell, f.room)

	assert.True(t, holdsRegen(&f.casterMob.Character), "the caster heals itself")
	assert.True(t, holdsRegen(&f.targetMob.Character), "the party's companion is healed")
	assert.False(t, holdsRegen(&rat.Character), "a stranger's pet is not on the owner's side")
	assert.True(t, holdsRegen(f.casterUser.Character), "the owner is healed")
}
```

- [ ] **Step 2: Run to see them fail**

Run: `cd C:/tmp/dogmud-3b-helpful && go test ./internal/hooks/ -run 'TestHelpArea_' -count=1`
Expected: FAIL `PlayerHeals...` ("a stranger's pet is not"), `AMobHealsItsPack...` (Rat and every player healed), `ACharmedMobHeals...` (Rat healed).

- [ ] **Step 3: Add the filler** to `spell_help_effects.go`. Add `"github.com/GoMudEngine/GoMud/internal/mobs"`, `"github.com/GoMudEngine/GoMud/internal/parties"` and `"github.com/GoMudEngine/GoMud/internal/rooms"` to its imports, and append:

```go
// spellHelpAreaTargets is the one area-help target filler (slice 3b, audit
// row 13), for player and mob casters alike. It returns the players and the
// mobs an area help spell (a mass heal, a cleansing wave) lands on,
// replacing whatever the cast's initiation step put in the target lists.
//
// A player caster, or a mob charmed by a player, stands on that player's
// side: every player in the room, plus every mob charmed by that player or
// by any member of that player's party (helpAreaCharmAlly), so a party
// member's companion is healed and a stranger's pet or an enemy is not. The
// AI companion is charmed to its owner permanently
// (modules/aicompanion/commands.go, Charm(owner.UserId, -1, ...)), so it
// counts. A charmed mob caster also lands on itself.
//
// An uncharmed mob caster helps itself and its packmates: the rule its own
// AI uses to choose whom to heal (behaviortree's cast_best_in_category picks
// most_wounded_packmate and tanking_packmate from mobs.FindPackmatesInRoom).
// No player is its ally.
func spellHelpAreaTargets(caster actions.Actor, room *rooms.Room) (userIds []int, mobIds []int) {
	self := actorMob(caster)
	sideUserId := caster.GetUserId()
	if self != nil {
		sideUserId = self.Character.GetCharmedUserId()
		mobIds = append(mobIds, self.InstanceId)
	}
	if sideUserId == 0 {
		for _, pm := range mobs.FindPackmatesInRoom(self) {
			mobIds = append(mobIds, pm.InstanceId)
		}
		return nil, mobIds
	}
	userIds = room.GetPlayers(rooms.FindAll)
	for _, mId := range room.GetMobs(rooms.FindAll) {
		if self != nil && mId == self.InstanceId {
			continue
		}
		if m := mobs.GetInstance(mId); m != nil && helpAreaCharmAlly(m, sideUserId) {
			mobIds = append(mobIds, mId)
		}
	}
	return userIds, mobIds
}

// helpAreaCharmAlly reports whether m is charmed by sideUserId or by a member
// of sideUserId's party (owner ruling, 2026-09-28). parties.Get also returns
// the party of an invitee, so the side must be a member itself.
func helpAreaCharmAlly(m *mobs.Mob, sideUserId int) bool {
	charmer := m.Character.GetCharmedUserId()
	if charmer == 0 {
		return false
	}
	if charmer == sideUserId {
		return true
	}
	p := parties.Get(sideUserId)
	return p != nil && p.IsMember(sideUserId) && p.IsMember(charmer)
}
```

- [ ] **Step 4: Hook it up.** In `resolveSpell`, replace the whole HelpArea block (105-120 on the base: from the `// --- Populate area targets for HelpArea ---` comment through the closing `}` after `cs.TargetMobInstanceIds = allies`, which holds the `m.Character.IsCharmed()` loop) with:

```go
	// --- Populate area targets for HelpArea ---
	// REPLACES whatever the cast's initiation step filled in, so the
	// caster's pre-spell aggro target (an enemy mob) is not healed alongside
	// its allies. Symmetric with HarmArea above; the same filler serves mob
	// casters in resolveMobSpell.
	if !spellData.IsHarm() && spellData.Targeting == combatvocab.TargetArea {
		cs.TargetUserIds, cs.TargetMobInstanceIds = spellHelpAreaTargets(actions.NewUserActorInRoom(user, room), room)
	}
```

In `resolveMobSpell`, directly after the HarmArea block (1055-1057), add:

```go
	if !spellData.IsHarm() && spellData.Targeting == combatvocab.TargetArea {
		cs.TargetUserIds, cs.TargetMobInstanceIds = spellHelpAreaTargets(actions.NewMobActorInRoom(mob, room), room)
	}
```

In `resolveSpell`'s doc comment, replace:

```go
//   - HelpArea is player-only (mobs never cast area healing in this engine).
```

with:

```go
//   - HelpArea fills through spellHelpAreaTargets on both paths: a player
//     or a charmed mob helps the players and the party's companions, an
//     uncharmed mob helps its packmates.
```

and in `resolveMobSpell`'s doc comment, after the HarmArea bullet, add:

```go
//   - HelpArea fills through the same spellHelpAreaTargets as resolveSpell.
```

- [ ] **Step 5: Run**

```bash
cd C:/tmp/dogmud-3b-helpful && go build ./... && go test ./internal/hooks/ -count=1
```

Expected: `ok`, including `TestHelpArea_*`, `TestAreaHeal_CasterIsTheirOwnTarget`, `TestAreaPurge_CasterIsTheirOwnTarget` and `TestMobAreaSpellEmitsOnePrivateShortagePerActualPlayerTarget`. `go vet ./internal/hooks/` clean (`rooms` and `mobs` are still used elsewhere in `spell_resolution.go`). Root guards. Expected: `ok`.

- [ ] **Step 6: Commit**

```bash
cd C:/tmp/dogmud-3b-helpful && git add internal/hooks/spell_help_area_test.go internal/hooks/spell_help_effects.go internal/hooks/spell_resolution.go && git commit -F - <<'EOF'
feat(spells): one area-help filler for players and mobs

spellHelpAreaTargets replaces resolveSpell's "any charmed mob" fill and
gives resolveMobSpell the fill it never had (audit row 13). A player, or
a mob charmed by one, helps the players in the room and the mobs charmed
by that player or a party member, so companions are healed and a
stranger's pet is not (owner ruling). An uncharmed mob helps itself and
its FindPackmatesInRoom packmates, the rule its AI uses to pick whom to
heal, and no player.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 8: The parity table

**Model:** opus.

**Files:**
- Create: `internal/hooks/spell_help_parity_test.go`

- [ ] **Step 1: Write the table**:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parityQueuedConditions drains the Condition events queued for p's target.
func parityQueuedConditions(f *spellParityFixture, p spellParityPairing) []events.Condition {
	switch p.target(f) {
	case f.targetUser.Character:
		return events.DrainQueuedConditionsForTest(f.targetUser.UserId)
	case &f.targetMob.Character:
		return events.DrainQueuedMobConditionsForTest(f.targetMob.InstanceId)
	}
	return events.DrainQueuedMobConditionsForTest(f.casterMob.InstanceId)
}

// Parity slice 3b: each helpful effect, driven through all five pairings
// with the same caster stats and spell, lands the same amount on its target
// with no contest, is recorded once as a landed cast, and is seen by a
// watcher. Heal and shield roll nothing, so their amounts compare exactly,
// written from the fixture's numbers (parityHealRounds, parityShieldBonus).
func TestSpellHelpParity_EveryPairingLandsTheSame(t *testing.T) {
	effects := []struct {
		name               string
		spell              func() *spells.SpellData
		roomWord, selfWord string
		prepare            func(t *testing.T, target *characters.Character)
		check              func(t *testing.T, f *spellParityFixture, p spellParityPairing)
	}{
		{name: "heal", spell: healSpellForParityTest,
			roomWord: "in healing light", selfWord: "channels restorative magic",
			check: func(t *testing.T, f *spellParityFixture, p spellParityPairing) {
				rec := regenRecord(t, p.target(f))
				assert.Equal(t, 3.0, rec.Magnitude)
				assert.Equal(t, parityHealRounds(), rec.TriggersLeft)
			}},
		{name: "shield", spell: shieldSpellForParityTest,
			roomWord: "shimmering barrier surrounds", selfWord: "shimmering barrier surrounds",
			check: func(t *testing.T, f *spellParityFixture, p spellParityPairing) {
				rec := shieldRecord(t, p.target(f))
				assert.Equal(t, parityShieldBonus(), rec.Magnitude)
				assert.Equal(t, calcSpellDuration(4, 3, 100), rec.TriggersLeft)
			}},
		{name: "condition", spell: conditionSpellForParityTest,
			roomWord: "settles over", selfWord: "settles over",
			check: func(t *testing.T, f *spellParityFixture, p spellParityPairing) {
				queued := parityQueuedConditions(f, p)
				require.Len(t, queued, 1, "the condition is queued once for the target")
				assert.Equal(t, 100, queued[0].ConditionId)
			}},
		{name: "purge", spell: purgeSpellForParityTest,
			roomWord: "cleanses", selfWord: "cleanses",
			prepare: func(t *testing.T, target *characters.Character) { poisonForPurgeTest(t, target) },
			check: func(t *testing.T, f *spellParityFixture, p spellParityPairing) {
				assert.False(t, poisoned(p.target(f)), "the target is cleansed")
			}},
	}
	for _, eff := range effects {
		t.Run(eff.name, func(t *testing.T) {
			for _, p := range spellHelpParityPairings() {
				t.Run(p.name, func(t *testing.T) {
					f := newSpellParityFixture(t, spellContestAttackWin())
					contests := countSpellContests(t)
					if eff.prepare != nil {
						eff.prepare(t, p.target(f))
					}
					events.DrainQueuedMobConditionsForTest(0) // zero drains player ones too

					p.cast(f, eff.spell())

					assert.Zero(t, *contests, "a help spell runs no contest")
					eff.check(t, f, p)
					require.Len(t, f.records, 1, "one record per resolved cast")
					assert.Equal(t, p.src, f.records[0].src)
					assert.Equal(t, p.tgt, f.records[0].tgt)
					assert.True(t, f.records[0].hit, "an uncontested cast landed")
					word := eff.roomWord
					if p.name == "MS" {
						word = eff.selfWord
					}
					assert.Equal(t, 1, countContaining(drainPlain(3), word), "a watcher sees it land")
				})
			}
		})
	}
}
```

- [ ] **Step 2: Run it**

Run: `cd C:/tmp/dogmud-3b-helpful && go test ./internal/hooks/ -run TestSpellHelpParity_EveryPairingLandsTheSame -count=1 -v`
Expected: PASS, twenty subtests (four effects by five pairings).

- [ ] **Step 3: Prove it can fail.** Temporarily add, in `applySpellHeal` right after `regenMult` is floored, `if c.casterMob() != nil { regenMult++ }`. Rerun Step 2's command. Expected: FAIL in `heal/MM`, `heal/MP`, `heal/MS` (4 is not 3). Remove the line with the Edit tool; rerun; expect PASS.

- [ ] **Step 4: Run the package and the root guards.** Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
cd C:/tmp/dogmud-3b-helpful && git add internal/hooks/spell_help_parity_test.go && git commit -F - <<'EOF'
test(spells): the helpful parity table across five pairings

Heal, shield, condition and purge, cast through player-on-mob,
player-on-player, mob-on-self, mob-on-mob and mob-on-player with the
same caster, land the same amount with no contest, record one landed
cast and are seen by a watcher. Proven red by a planted mob-only heal
bonus.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 9: Retire the wrappers; finish the sibling comments

**Model:** sonnet.

**Files:**
- Modify: `internal/hooks/hooks_test.go`, `selfcast_spell_room_hiding_test.go`, `selfcast_wording_test.go`, `spell_names_in_dark_test.go`, `spell_tick_scale_test.go`, `wire_freeze_test.go`
- Modify: `internal/hooks/spell_effects.go` (delete `spellCasterActor` and the wrapper section)
- Modify: `internal/hooks/spell_resolution.go:243-246`, `internal/hooks/charm_spell.go` (18-21, 34-37), `internal/actions/cast.go:135-137`, `_datafiles/world/dogmud/behaviors/crash_site_interior/9585-repair_frame.yaml:9-20`
- Modify: `condition_apply_path_guard_test.go` (comments), `send_trio_only_guard_test.go:76-84`

- [ ] **Step 1: Migrate every test call.** Add `"github.com/GoMudEngine/GoMud/internal/actions"` to the imports of `selfcast_spell_room_hiding_test.go`, `selfcast_wording_test.go` (if Task 5 has not), `spell_names_in_dark_test.go`, `spell_tick_scale_test.go` and `wire_freeze_test.go` (`hooks_test.go` already has it). Line numbers are the base's; find each call by its text. Apply these three patterns exactly:

| Old | New |
|---|---|
| `applyPlayerEffect(A, B, R, S, M, O)` | `applySpellEffect(newSpellEffectCtx(A.Character, actions.NewUserActorInRoom(A, R), actions.NewUserActorInRoom(B, R), R, S, M, O))` |
| `applyMobEffect(U, U.Character, MOB, R, S, M, O)` | `applySpellEffect(newSpellEffectCtx(U.Character, actions.NewUserActorInRoom(U, R), actions.NewMobActorInRoom(MOB, R), R, S, M, O))` |
| `applyMobSelfEffect(MOB, R, S, M)` | `applySpellEffect(newSpellEffectCtx(&MOB.Character, actions.NewMobActorInRoom(MOB, R), actions.NewMobActorInRoom(MOB, R), R, S, M, uncontestedSpellResult()))` |

A leading `dmg := ` stays. Where R, A or B is a call expression, hoist it first, as follows:

- `hooks_test.go`: 2648, 2665 (`applyMobEffect`, R is `room`); 2699, 2715, 2747, 2762, 2776, 2791 (`applyPlayerEffect`, R is `room`); 2881, 2896 (`applyMobSelfEffect`, R is `room`).
- `selfcast_spell_room_hiding_test.go:110`, `selfcast_wording_test.go` 36, 56, 77, 99, 131, 214: `applyPlayerEffect`, R is `room`.
- `spell_names_in_dark_test.go:22` (inside `castHealOnBobrick`, whose named results are `caster, target`): replace the line with

```go
	c, tg, r := users.GetByUserId(1), users.GetByUserId(2), rooms.LoadRoom(1)
	applySpellEffect(newSpellEffectCtx(c.Character, actions.NewUserActorInRoom(c, r),
		actions.NewUserActorInRoom(tg, r), r, spell, 3, spellContestAttackWin()))
```

  and line 75 with the same two lines, magnitude `10` in place of `3`.
- `spell_tick_scale_test.go:108`: replace with

```go
	r := rooms.LoadRoom(1)
	applySpellEffect(newSpellEffectCtx(caster.Character, actions.NewUserActorInRoom(caster, r),
		actions.NewUserActorInRoom(target, r), r, spell, 0, spellContestAttackWin()))
```

  and 134 with

```go
	r := rooms.LoadRoom(1)
	applySpellEffect(newSpellEffectCtx(&mob.Character, actions.NewMobActorInRoom(mob, r),
		actions.NewMobActorInRoom(mob, r), r, spell, 0, uncontestedSpellResult()))
```

- `wire_freeze_test.go`: 67 (`applyMobEffect`, R is `room`) and 126 (`applyMobSelfEffect`, R is `room`) by the patterns; 215 and 240 hoist `r := rooms.LoadRoom(1)` first, then `applySpellEffect(newSpellEffectCtx(u.Character, actions.NewUserActorInRoom(u, r), actions.NewMobActorInRoom(mob, r), r, lightSpell("test-glow-mob"), 0, spellContestAttackWin()))` and `applySpellEffect(newSpellEffectCtx(&mob.Character, actions.NewMobActorInRoom(mob, r), actions.NewMobActorInRoom(mob, r), r, lightSpell("test-glow-mobself"), 0, uncontestedSpellResult()))`.

- [ ] **Step 2: Rename the tests and headers in `hooks_test.go`.** `TestApplyMobEffect_Condition` to `TestApplySpellEffect_ConditionOnMob`; `TestApplyMobEffect_DefaultEffect` to `TestApplySpellEffect_DefaultOnMob`; `TestApplyPlayerEffect_Purge`, `_Heal`, `_Condition`, `_Shield`, `_ShieldSelf`, `_Default` to `TestApplySpellEffect_PurgeOnPlayer`, `_HealOnPlayer`, `_ConditionOnPlayer`, `_ShieldOnPlayer`, `_ShieldSelf`, `_DefaultOnPlayer`; `TestApplyMobSelfEffect_Heal`, `_Shield` to `TestApplySpellEffect_MobSelfHeal`, `_MobSelfShield`. Section headers: `// ─── Spell Resolution: applyMobEffect ───...` (2546) becomes `// ─── Spell Resolution: applySpellEffect on a mob ───...`; the `applyMobEffect` "more branches" header (2611) becomes `// ─── Spell Resolution: applySpellEffect, more branches ───...`; `applyPlayerEffect` (2683) becomes `applySpellEffect on a player`; `applyMobSelfEffect` (2866) becomes `applySpellEffect, a mob on itself`. Keep each header's trailing box rule.

- [ ] **Step 3: `wire_freeze_test.go` comments.** Replace the first paragraph of the doc comment (20-29) with:

```go
// TestWireFreeze_EffectTypeConditionStillApplies is a freeze test for the
// spell effect type `effect_type: condition`. Nothing reads that string
// except the `case "condition":` literal in applySpellEffect
// (spell_effects.go) and the spell-sorting switch in
// internal/usercommands/spells.go. A text replace across the codebase could
// change one of those literals without the compiler noticing (a string
// literal, not an identifier) and without a parse-only freeze test going
// red. Each of these four subtests drives a REAL `effect_type: condition`
// spell through one caster-target pairing and asserts the condition was
// actually queued for the target, so a broken case literal shows up here
// even though it can't show up in a yaml.Unmarshal-only test.
```

In the third paragraph replace `the\n// existing tests that already drive applyMobEffect and\n// resolveMobSpellAgainstPlayer this way.` with `the\n// existing tests that already drive applySpellEffect and\n// resolveMobSpellAgainstPlayer this way.` Replace the four subtest lead comments (they begin `// spell_resolution.go:906`, `:1093`, `:1493`, `:1740`) with, in order: `// PM: a player's spell landing on a mob.`, `// PP self: resolveAgainstPlayer with the caster as its own target.`, `// MS: a mob's spell on itself.`, `// MP: a mob's spell landing on a player.` In the four `require.Len` messages, replace each parenthetical `(spell_resolution.go's ... case \"condition\")` with `(applySpellConditionEffect)`.

- [ ] **Step 4: Delete the wrappers.** In `spell_effects.go`, delete `spellCasterActor` with its doc comment (58-68) and the whole test-only section: the `// ── Test-only wrappers...` header, `applyMobEffect`, `applyPlayerEffect` and `applyMobSelfEffect`. `go build ./internal/hooks/` must report nothing unused (`users`, `mobs`, `rooms`, `spells` are still used by the context helpers).

- [ ] **Step 5: Comments naming retired functions.**
  - `spell_resolution.go` 243-246: replace `It now resolves inside the loop, in applyMobEffect's "charm" arm,\n	// off that one contest.` with `It now resolves inside the loop, in applySpellEffect's "charm" case,\n	// off that one contest.`
  - `charm_spell.go` 20-21: replace `the channel contest the cast had already run and then discarded in\n// applyMobEffect_default.` with `the channel contest the cast had already run and then discarded in\n// the default arm.`; 34-35: replace `	// applyMobEffect is reached with a nil user when a MOB casts (see its\n	// docstring and resolveMobSpellAgainstMob). No mob carries charm today --` with `	// applySpellEffect reaches this with a nil user when a MOB casts (a mob\n	// caster has no *users.UserRecord). No mob carries charm today --`.
  - `internal/actions/cast.go` 135-137: replace `uncontested shortcut into applyPlayerEffect, which has no\n				// charm arm.` with `uncontested shortcut into the player-target effect switch,\n				// which had no charm arm.`
  - `9585-repair_frame.yaml` 13-20: replace from `# and resolveMobSpellAgainstMob dispatches non-self mob targets through` through `# never cast area healing in this engine."` with:

```yaml
# and resolveMobSpellAgainstMob dispatches non-self mob targets through
# the effect dispatcher (applySpellEffect, internal/hooks/spell_effects.go),
# whose heal applier serves every caster and target. A help spell resolves
# uncontested (a friendly heal is never "resisted"). Area help from a mob
# lands on its packmates (spellHelpAreaTargets), not on this boss by name,
# so this add keeps casting single-target heals.
```

  - `send_trio_only_guard_test.go` 76-84: replace from `//     Parity slice 3a moved damage, dot and knockdown onto SendTrio (now` through `//     untouched by PR 3 entirely.` with:

```go
//     Parity slices 3a and 3b moved every spell effect onto SendTrio
//     (spell_effects.go and spell_help_effects.go); the raw senders left
//     are applyMobEffect_charm (charm_spell.go:26), resolveSpell's
//     no-target and magic lines, resolveIdentify and
//     resolvePurgeAffliction's self-cast lines. CategorySpellFold was
//     untouched by PR 3 entirely.
```

  - `condition_apply_path_guard_test.go`: in the ward block comment, replace `parity slice 3a Task 8 when maybeInterruptSpellOnMob was deleted) ────` with `parity slice 3a Task 8 when maybeInterruptSpellOnMob was deleted;\n	// COLLAPSED to one row by parity slice 3b, when applySpellShield replaced\n	// the player-to-player shield and the mob self-cast shield) ────`; in the regen block comment, replace `slice 3a Task 8, same deletion as above) ─────` (the tail of that comment) with `slice 3a Task 8, same deletion as above; COLLAPSED to one row by parity\n	// slice 3b, when applySpellHeal replaced the three heal arms) ─────`. Keep each line's trailing box rule as it stands; `\n\t` means a real newline and tab.

- [ ] **Step 6: Confirm no stale names remain** (standalone; grep exits 1 on zero, the pass):

```bash
cd C:/tmp/dogmud-3b-helpful && grep -rn "applyMobEffect(\|applyPlayerEffect(\|applyMobSelfEffect\|spellCasterActor\|applyMobEffect'\|applyMobEffect is\|applyMobEffect_default\|applyMobEffect_heal\|applyMobEffect_condition\|EffectArms\|OnPlayerArms" --include=*.go --include=*.yaml internal modules _datafiles *.go
```

Expected: no output. Prove the grep can match: `grep -rn "applyMobEffect_charm" --include=*.go internal/hooks | head -1` prints a line.

- [ ] **Step 7: Run** `cd C:/tmp/dogmud-3b-helpful && go build ./... && go vet ./internal/hooks/ ./internal/actions/ && go test ./internal/hooks/ ./internal/actions/ -count=1` and the root guards. Expected: all `ok`.

- [ ] **Step 8: Commit**

```bash
cd C:/tmp/dogmud-3b-helpful && git add internal/hooks/hooks_test.go internal/hooks/selfcast_spell_room_hiding_test.go internal/hooks/selfcast_wording_test.go internal/hooks/spell_names_in_dark_test.go internal/hooks/spell_tick_scale_test.go internal/hooks/wire_freeze_test.go internal/hooks/spell_effects.go internal/hooks/spell_resolution.go internal/hooks/charm_spell.go internal/actions/cast.go _datafiles/world/dogmud/behaviors/crash_site_interior/9585-repair_frame.yaml condition_apply_path_guard_test.go send_trio_only_guard_test.go && git commit -F - <<'EOF'
refactor(spells): retire the effect wrappers; tests call the dispatcher

Every test that named applyMobEffect, applyPlayerEffect or
applyMobSelfEffect builds a spellEffectCtx and calls applySpellEffect,
so the wrappers and spellCasterActor are deleted. Comments in charm,
cast, the Repair Frame behaviour and two guards stop naming functions
that no longer exist.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 10: Docs, gate, boot check, playtest, PR

**Model:** sonnet (the playtest per `dogmud-playtesting` and `/playtest-scenario`).

**Files:**
- Modify: `internal/hooks/context.md`, `internal/conditions/context.md`, `internal/characters/context.md`, `internal/combat/context.md`, `internal/items/context.md`, `docs/PATCH_NOTES.md`, `docs/README.md`
- Create: `tools/playtest/scenarios/spell-effects-3b.yaml`, `tools/playtest/goals/scenarios/spell-effects-3b/healer.yaml`, `partner.yaml`, `mobwatch.yaml`

- [ ] **Step 1: `internal/hooks/context.md`.**

  (a) Replace the whole `## Spell effects (\`spell_effects.go\`, parity slice 3a)` section (from its heading through the line before `## Counter tier wiring (U6b Task 10)`) with:

```markdown
## Spell effects (`spell_effects.go`, `spell_help_effects.go`, parity slices 3a and 3b)

Every spell effect on one target goes through one `spellEffectCtx` and one
dispatcher, `applySpellEffect`, whoever casts it and whoever it hits: player
on mob, player on player, mob on itself, mob on mob, mob on player. The four
contested resolvers in `spell_resolution.go` (`resolveAgainstMob`,
`resolveAgainstPlayer`, `resolveMobSpellAgainstMob`,
`resolveMobSpellAgainstPlayer`) keep their names and their one
`runSpellChannelAttack` call each, then build a context per target: the
caster's `*characters.Character`, caster and target as `actions.Actor`
(`*actions.UserActor` or `*actions.MobActor`; only tests pass a nil caster),
the room, the spell, the magnitude and the contest result. Refs come from the
actor (`casterRef`, `targetRef`), not the character.

The harmful effects have one applier each: `applySpellDamage`,
`applySpellDot`, `applySpellKnockdown`. Each starts the fight through
`commitHarmfulSpellAggro`: the target turns on the caster if it was not
already fighting, the caster on the target likewise, and a player caster on a
mob calls `actions.SeedAggression` with freshness judged per target, as
`throw` does, which records the assault crime on a fresh engagement (owner
ruling, 2026-09-28). Damage and knockdown on a mob call `creditSpellDamage`
before the harm, as melee does with `TrackPlayerDamage`; the dot does not
(its ticks harm anonymously, a filed follow-up). A dot's duration reads the
spell's primarystat and the school's cast skill through
`spellCasterStatAndSkill`, not `actions.GetSpellStatAndSkill`, which is the
fold stat.

The helpful effects have one applier each in `spell_help_effects.go`:
`applySpellConditionEffect` (every named condition through
`applySpellCondition`'s event door), `applySpellHeal` (a Regenerating
record), `applySpellShield` (a Minor Shield record) and `applySpellPurge`
(cancels every poison). `applySpellDefaultEffect` serves an effect with no
applier of its own, and charm binds a mob through `applyMobEffect_charm`.
A defended status narrates the defence triad and applies nothing
(`spellStatusDefended`); a harmful condition or default spell still starts
the fight through `commitHarmfulSpellAggro`. When the caster is its own
target (`selfCast`), the caster reads its own line and the room reads the
caster named once (`selfCastAudience`). Heal and shield read the caster
through `spellCasterStatAndSkill` and never crit: a help spell never enters
the contest, the only source of a crit (owner ruling, 2026-09-28). A player
healing a mob queues `events.Healed` for the AI companion. Every applier
narrates through `messaging.SendTrio` (players in the username tag, mobs
through `mobDisplayName`), so a mob's spell on a mob reaches the room and a
reader in the dark reads "something".

Every resolver takes a help spell (`attack_type: none`) through
`resolveHelpSpell` with `uncontestedSpellResult()`: no contest, no fumble,
no counter, one landed record. `resolveMobSpell`'s self branch (a mob on
itself) takes the same step and never applies a harmful spell to the
caster. A contested cast shares three steps: `applySpellBackfire` (every
caster kind is hurt, told, seen and recorded), `interruptSpellTarget` (a
configured boss-interrupt spell cancels any casting target through
`maybeInterruptSpellOnTarget`) and `recordSpellResolution`. `recordSpell`
is the analytics seam over `combat.RecordSpell`, swapped by tests the way
`runSpellChannelAttack` is.

`spellHelpAreaTargets` fills an area help spell's targets for both caster
kinds, replacing what the cast's initiation step put there. A player, or a
mob charmed by one, helps every player in the room and every mob charmed by
that player or by a member of that player's party (`helpAreaCharmAlly`,
`parties.Get(...).IsMember`), so the party's companions, the bonded AI
companion included, are healed and a stranger's pet is not. An uncharmed
mob helps itself and its `mobs.FindPackmatesInRoom` packmates, the rule its
behaviour tree's `cast_best_in_category` uses to pick whom to heal, and no
player.

`resolveMobDrainArea` keeps its own `actions.ExecuteDrainArea` contest; only
its lines use the context. `channel_defence_routing_test.go` parses
`spell_resolution.go`, `spell_effects.go` and `spell_help_effects.go` and
allows the contest seam only in the four resolvers.
```

  (b) In the "Names in the dark" paragraph, replace from `(\`spellAudience\` in \`spell_audience.go\`, used by the appliers in` through `already on \`SendTrio\` before this PR and is unaffected.` with:

```markdown
(`spellAudience` in `spell_audience.go`, used by the appliers in
`spell_effects.go` and `spell_help_effects.go`,
`sendSpellChannelDefenceMessages` and
`resolvePurgeAffliction`, which takes a `purgeTarget`: a player, a mob such as a
charmed companion, or the caster) all go through `messaging.SendTrio`, so a reader who
cannot see the other party reads "something", or "a figure" with infrared.
A self-cast (the caster is its own target, `spellEffectCtx.selfCast`) has no
second party to pair against: the helpful appliers send the caster's own
line and a room line naming the caster once, through `SendTrio` with
`selfCastAudience` (`Actee` is `messaging.NoLine`), so a shapes-only
observer reads "a figure". **M4d PR 3** first moved the player self-cast
room lines off `sendVisualRoomText`, which never calls `messaging.HideNames`;
parity slice 3b made the same pair serve a mob casting on itself.
```

  (c) Replace `the four spell-condition sites in \`spell_resolution.go\`\n(\`applyMobEffect_condition\`, \`applyPlayerEffect\`, \`applyMobSelfEffect\`,\n\`resolveMobSpellAgainstPlayer\`) now call:` with `the one spell-condition applier, \`applySpellConditionEffect\`\n(\`spell_help_effects.go\`, every pairing since parity slice 3b), calls:`.

  (d) Replace `and moved it into \`applyMobEffect\`'s switch, because the old function ran a` with `and moved it into the effect dispatcher (now \`applySpellEffect\`), because the old function ran a`.

  (e) In "Spell Duration System", replace from `There are seven call sites, all in this file, and they fall into exactly` through `section, which is true on the player-cast shield path only.` with:

```markdown
There are three call sites, one per effect, since parity slice 3, and each
reads the caster through `spellCasterStatAndSkill` (the spell's primarystat
through `CasterStatValue`, and the school's cast skill):

- **Shield: full duration, no divisor.** `applySpellShield`
  (`spell_help_effects.go`) passes `calcSpellDuration(...)` unmodified to
  `AddConditionMagnitude(conditions.ConditionIdMinorShield, duration, ...)` as the trigger
  count (record 119 ticks once a round, so triggers and rounds coincide).
- **Heal: `/2`, floored at 6.** `applySpellHeal` (`spell_help_effects.go`)
  computes `calcSpellDuration(...) / 2`, then clamps `durationRounds < 6` up
  to 6, before
  `AddConditionMagnitude(conditions.ConditionIdRegenerating, durationRounds, regenMult, ...)`.
- **DoT: `/3`, floored at 3.** `applySpellDot` (`spell_effects.go`) computes
  `calcSpellDuration(...) / 3`, then clamps `dotDuration < 3` up to 3, and
  passes that rounds figure straight to
  `AddConditionMagnitude(conditions.ConditionIdPoisoned, dotDuration, ...)`: record 121 ticks
  every round (slice 1b; it was every third round before). See
  `internal/conditions/context.md` under "Cadence".

**Crit never touches a duration, and never touches a help spell.** A
harmful spell's crit shows in its damage and its `[CRIT!]` tag. Heal and
shield used to carry a player-only crit bump (x2 above 1x regen, x1.5
shield) that no cast could reach, because a help spell never enters the
contest, the only source of a crit; parity slice 3b deleted both (owner
ruling, 2026-09-28).
```

  Verify every symbol named exists, then run the audit:

```bash
cd C:/tmp/dogmud-3b-helpful && for s in spellEffectCtx applySpellEffect applySpellDamage applySpellDot applySpellKnockdown commitHarmfulSpellAggro creditSpellDamage spellCasterStatAndSkill applySpellBackfire interruptSpellTarget maybeInterruptSpellOnTarget recordSpellResolution recordSpell resolveHelpSpell uncontestedSpellResult applySpellConditionEffect applySpellHeal applySpellShield applySpellPurge applySpellDefaultEffect spellStatusDefended selfCast selfCastAudience spellHelpAreaTargets helpAreaCharmAlly applyMobEffect_charm; do printf "%s %s\n" $s "$(cat internal/hooks/*.go | grep -cE "^(func|var|type) (\([^)]*\) )?$s\b")"; done
python tools/context_md_audit.py
```

  Expected: every symbol prints 1 (a 0 means the doc names something that does not exist; fix the doc); the audit reports nothing for `internal/hooks`, `internal/conditions`, `internal/characters`, `internal/combat` or `internal/items`.

- [ ] **Step 2: The other package docs.**
  - `internal/conditions/context.md`: replace from ``The `case "shield"` branch of`` through `already exists and is used by search, track, and forage checks; no spell path\ncalls it.` with:

```markdown
`applySpellShield` in `internal/hooks/spell_help_effects.go` is the single
handler for every shield spell, whoever casts it and whoever it lands on
(parity slice 3b; a shield on a pet, or from a creature, applied nothing
before it).

```go
shieldBonus := (stat + weightedSkill) / 3
if c.magnitude > 0 {
    shieldBonus = int(math.Round(float64(shieldBonus) * float64(c.magnitude) / 100.0))
}
_ = c.targetChar().AddConditionMagnitude(conditions.ConditionIdMinorShield, duration, float64(shieldBonus), "spell")
```

`stat` and the skill come from `spellCasterStatAndSkill`: the spell's
primarystat through `CasterStatValue`, and the school's cast skill.
`weightedSkill` is that skill times `SkillWeight` (ships 5.0 against a Go
default of 2.0). `magnitude` is `spellData.EffectMagnitude`, and 100 is the
1.0x baseline: a spell carrying `effect_magnitude: 75` applies 0.75 of the
base roll, one carrying 125 applies 1.25x. A shield does not crit. The
player path used to carry a x1.5 crit bump no cast could reach: both
shipped shields are `attack_type: none`, so every resolver takes them
through `resolveHelpSpell` with `uncontestedSpellResult()` and no roll;
slice 3b deleted the bump (owner ruling, 2026-09-28). A future crit would
need a real roll: the static-difficulty seam
`contest.AgainstDifficulty(score, difficulty)` (`internal/contest/contest.go`)
already exists and is used by search, track, and forage checks; no spell path
calls it.
```

  - `internal/characters/context.md` 603-605: replace `floored at 1.0; a\n  crit doubles the portion above 1x). \`internal/hooks/spell_resolution.go\`'s\n  \`"heal"\` case calls \`target.Character.AddConditionMagnitude(conditions.ConditionIdRegenerating,` with `floored at 1.0; help\n  spells do not crit). \`internal/hooks/spell_help_effects.go\`'s\n  \`applySpellHeal\` calls \`c.targetChar().AddConditionMagnitude(conditions.ConditionIdRegenerating,`.
  - `internal/combat/context.md:1846`: replace `| \`hooks/spell_resolution.go\` | \`resolveSpell\`, \`resolveAgainstMob\`, \`resolveAgainstPlayer\`, \`applyPlayerEffect\` |` with `| \`hooks/spell_resolution.go\` | \`resolveSpell\`, \`resolveAgainstMob\`, \`resolveAgainstPlayer\` (effects apply through \`applySpellEffect\` in \`hooks/spell_effects.go\`) |`.
  - `internal/items/context.md` 1216-1218: replace the bullet beginning `` - `applyMobEffect_condition` (`internal/hooks/spell_resolution.go`) applies the`` with:

```markdown
- `spellTickScale` (`internal/hooks/spell_tick_scale.go`), which
  `applySpellCondition` uses for a tick-pool condition, reads the same
  multiplier separately, since that path doesn't route through
  `calcSpellDamageForCharacter`.
```

- [ ] **Step 3: `docs/PATCH_NOTES.md`.** Add at the top, below `# DOGMud Patch Notes`, dated the day the PR opens (80 columns, no numbers, no dashes):

```markdown
## <YYYY-MM-DD>: Healing and shields, whoever casts them

Creatures now heal and shield you and each other by the same rules you
do. A creature's mending spell on you now mends you, its shield now
holds, and a creature casting on itself or its packmates is seen by
everyone in the room.

Your area heals now reach your companions and those of everyone in
your party, a bonded companion included, and no longer patch up a
stranger's pet. A shield cast on your pet now holds, and a cleansing
spell now purges poison from a companion too.
```

- [ ] **Step 4: `docs/README.md`.** The planning commit added this plan's row; confirm it is present below the 3a plan row. If not, add it (text in the planning commit).

- [ ] **Step 5: Commit docs**

```bash
cd C:/tmp/dogmud-3b-helpful && git add internal/hooks/context.md internal/conditions/context.md internal/characters/context.md internal/combat/context.md internal/items/context.md docs/PATCH_NOTES.md docs/README.md && git commit -F - <<'EOF'
docs(spells): slice 3b context, patch notes

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

- [ ] **Step 6: Full gate** (Bash, from the worktree root; each standalone):

```bash
cd C:/tmp/dogmud-3b-helpful && gofmt -l internal/ modules/ *.go
```
Expected: no output. If it lists a file this plan never touched, check `git diff master -- <file>` before acting (Windows CRLF copies false-positive).

```bash
cd C:/tmp/dogmud-3b-helpful && go vet ./... && go build ./...
```
Expected: no output.

```bash
cd C:/tmp/dogmud-3b-helpful && go test ./... -count=1 > C:/tmp/dogmud-3b-test.log 2>&1; echo exit=$?
```
Run in the background (about ten minutes). Expected `exit=0`; `grep -E "^(FAIL|--- FAIL|panic)" C:/tmp/dogmud-3b-test.log` prints nothing.

```bash
cd C:/tmp/dogmud-3b-helpful && ~/go/bin/golangci-lint run --new-from-merge-base=origin/master
```
Expected: `0 issues.`

- [ ] **Step 7: Boot check** per `dogmud-shipping`, isolated ports, killed by PID. Bash:

```bash
cd C:/tmp/dogmud-3b-helpful && git worktree add --detach C:/tmp/dogmud-boot-check HEAD && cp "C:/Users/Calabe Davis/workspace/DOGMud/_datafiles/config.yaml" C:/tmp/dogmud-boot-check/_datafiles/config.yaml && cd C:/tmp/dogmud-boot-check && go build -o boot-check.exe .
```

Write `C:/tmp/dogmud-boot-check/boot-overrides.yaml` with the Write tool:

```yaml
Network.TelnetPort: [33334]
Network.LocalPort: 9998
Network.HttpPort: 8091
Network.HttpsPort: 0
Network.AIPort: 0
```

PowerShell (hidden window, prints the PID):

```powershell
$env:CONFIG_PATH = 'C:\tmp\dogmud-boot-check\boot-overrides.yaml'; $env:LOG_NOCOLOR = '1'; $p = Start-Process -FilePath 'C:\tmp\dogmud-boot-check\boot-check.exe' -WorkingDirectory 'C:\tmp\dogmud-boot-check' -RedirectStandardOutput 'C:\tmp\dogmud-boot-check\boot.log' -RedirectStandardError 'C:\tmp\dogmud-boot-check\boot.err' -WindowStyle Hidden -PassThru; $p.Id
```

Wait until `Server Ready` or a panic appears (Monitor with an until-loop on `C:/tmp/dogmud-boot-check/boot.log`, 180 s cap), then check each standalone:

```bash
grep -cE "^panic:|goroutine [0-9]+ \[running\]|runtime error" C:/tmp/dogmud-boot-check/boot.log
```
Expected `0` (grep exits 1 on zero; that is the pass).

```bash
grep -c "Server Ready" C:/tmp/dogmud-boot-check/boot.log
```
Expected `1`.

Stop only that PID, then clean up:

```powershell
Stop-Process -Id <PID printed above> -Force; Remove-Item -Recurse -Force 'C:\tmp\dogmud-boot-check'
```
```bash
cd C:/tmp/dogmud-3b-helpful && git worktree prune
```

- [ ] **Step 8: Playtest** per `dogmud-playtesting` and `.claude/commands/playtest-scenario.md` (ephemeral, `--checkout C:/tmp/dogmud-3b-helpful`, `docker rm -f` teardown, never touch the owner's server). Three actors on one server. Fixtures (verified 2026-09-28): 462 Main Street Central, Thornwall; Thornwall Thug 105 (`hostile: true`) spawns in Back Alley, West (474, reached south then west from 462 as in `party-formation`); Bandit Camp 4052 holds Bandit Fighter 284, Bandit Caster 285 (`mend-wounds` in her spellbook) and Soren 286, all `routine: bandit_camp_guard`. No shipped mob casts a heal or shield AT a player (F14), so the creature-on-player heal and shield are covered by `TestSpellHeal_ACreatureHealsAPlayer`, `TestSpellShield_ACreatureShieldsAPlayer` and the parity table, not by this playtest. Write the four files with the Write tool.

`tools/playtest/scenarios/spell-effects-3b.yaml`:

```yaml
# Spell effects parity slice 3b: one applier per helpful effect (condition,
# heal, shield, purge) whoever casts and whoever is helped; help spells are
# uncontested everywhere; one area-help filler. A player's area heal lands
# on the players in the room and on creatures charmed by the caster or a
# party member; an uncharmed creature's area help lands on its packmates.
#
# Fixtures (verified against YAML 2026-09-28):
#   462 Main Street Central, Thornwall. Thornwall Thug 105 (hostile)
#     spawns in Back Alley, West (474), south then west from 462.
#   4052 Bandit Camp, North Road: Bandit Fighter 284, Bandit Caster 285
#     (spellbook mend-wounds, mind-spike, mind-fog, nerve-disruption) and
#     Soren 286, all routine bandit_camp_guard, so they are packmates.
# No shipped creature casts a heal or shield at a player; the unit tests
# carry that case.
name: spell-effects-3b
mode: party
summary: >-
  A healer and a partner form a party in Thornwall; the healer charms a thug
  as a pet and casts area and single help spells over the party and the pet.
  A third caster fights the North Road bandits and reads the Bandit Caster's
  mending lines.
on_actor_stop: continue
budgets:
  wall_clock: 30m
requires:
  max_connections: 20
roster:
  - id: healer
    personality: feature-tester
    goals: goals/scenarios/spell-effects-3b/healer.yaml
  - id: partner
    personality: feature-tester
    goals: goals/scenarios/spell-effects-3b/partner.yaml
  - id: mobwatch
    personality: feature-tester
    goals: goals/scenarios/spell-effects-3b/mobwatch.yaml

group_goals:
  - id: party-formed
    do: Healer invites Partner by character name; Partner accepts.
    verify: Both see a two-member party in `party list`.
  - id: pet-charmed
    do: Healer charms the Thornwall Thug and brings it back to Partner.
    verify: The thug follows the healer and no longer attacks.
  - id: area-heal-reaches-party-and-pet
    do: Healer casts mend-all with Partner and the pet in the room.
    verify: >-
      Healer reads "You weave restorative magic around" for Partner AND for
      the thug, and one self line; Partner reads the healer's Mend All
      enveloping them in healing energy.
  - id: pet-shield-and-cleanse
    do: Healer casts conviction-ward and cleansing-wave on the pet.
    verify: >-
      A shimmering barrier forms around the thug; the cleanse line names
      it; neither says only "takes effect".
  - id: mob-healer
    do: Mobwatch fights the Bandit Caster and her packmates.
    verify: >-
      The Bandit Caster's mending reads "channels restorative magic" on
      herself or "envelops <bandit> in healing light" on a packmate.
  - id: reads-well
    do: Everyone judges every new spell line as prose.
    verify: No raw numbers, no em dashes, no " -- ", nothing over 80 columns.
```

`tools/playtest/goals/scenarios/spell-effects-3b/healer.yaml`:

```yaml
# Spell effects 3b: a healer's area heal reaches a party member and a pet.
# You are Bindsong, a charmer, starting on Main Street Central (462).
# Quote every line VERBATIM. 3 commands per round at most.
ephemeral:
  profile: charmer
  start_room: 462
  overlays:
    grant_spells:
      mend-all: 1
      conviction-ward: 1
      cleansing-wave: 1
  budgets:
    wall_clock: 30m

goals:
  - >-
    `look`, `spells`. Wait for the blackboard key 'partner-ready' (it holds
    your partner's character name). `party invite <that name>` and write
    'healer-invited'. Wait for 'partner-accepted', then `party list`.
  - >-
    Go south then west to Back Alley, West (474) alone. `cast charm thug`
    and quote every line. If it resists, try again. When it is yours,
    return to 462 and write 'pet-home'.
  - >-
    With your partner and the thug both in the room, `cast mend-all`. Quote
    EVERY line you read. Expected: one line about the glow around you, one
    "You weave restorative magic around" your partner and one around the
    thug. A missing line for the thug or the partner is a finding.
  - >-
    `cast conviction-ward thug`, then `cast cleansing-wave`. Quote every
    line. The ward should say a shimmering barrier forms around the thug;
    the cleanse should name the thug and your partner. A line that only
    says "takes effect" is a finding.
  - >-
    reads-well: a line naming the wrong party, a line sent twice, a raw
    number in player text, an em dash or " -- ", a line over 80 columns, or
    Go error text is a finding. Report bluntly.
```

`tools/playtest/goals/scenarios/spell-effects-3b/partner.yaml`:

```yaml
# Spell effects 3b: you are the healer's party member. You start on Main
# Street Central (462). Quote every line VERBATIM. 3 commands per round at
# most.
ephemeral:
  profile: mid
  start_room: 462
  budgets:
    wall_clock: 30m

goals:
  - >-
    `look`, `who`. Write the blackboard key 'partner-ready' with your
    character name. Wait for 'healer-invited', then `party accept` and write
    'partner-accepted'. `party list` to confirm two members.
  - >-
    Stay on Main Street Central (462). Do not follow the healer into the
    alley. Wait for 'pet-home'.
  - >-
    When the healer casts Mend All, quote every line you read. Expected: the
    healer's Mend All envelops you in healing energy, and a room line about
    it wrapping the thug in healing light. Then quote what you read when the
    healer casts Conviction Ward on the thug and Cleansing Wave.
  - >-
    reads-well: a line naming the wrong party, a line sent twice, a raw
    number in player text, an em dash or " -- ", a line over 80 columns, or
    Go error text is a finding. Report bluntly.
```

`tools/playtest/goals/scenarios/spell-effects-3b/mobwatch.yaml`:

```yaml
# Spell effects 3b: a creature heals itself and its packmates by the same
# rules a player does. You start in the Bandit Camp on the North Road
# (4052) with Bandit Fighter, Bandit Caster and Soren. Quote every line
# VERBATIM. 3 commands per round at most.
ephemeral:
  profile: specialist-caster
  start_room: 4052
  overlays:
    grant_spells:
      mind-spike: 1
    grant_skills:
      spellcasting: 30
  budgets:
    wall_clock: 30m

goals:
  - >-
    `look`. Fight the bandits (`attack fighter`, then `cast mind-spike
    caster`) so that one of them is hurt. Step `east` to 4043 to recover
    when needed. Watch for the Bandit Caster mending: quote every line that
    says "channels restorative magic" or "envelops ... in healing light",
    and say who she healed.
  - >-
    Note whether any of her help spells ever lands on YOU. It should not:
    a creature's area help reaches only its packmates.
  - >-
    reads-well: a line naming the wrong party, a line sent twice, a raw
    number in player text, an em dash or " -- ", a line over 80 columns, or
    Go error text is a finding. Report bluntly.
```

  Run it per `.claude/commands/playtest-scenario.md`:

```text
/playtest-scenario --checkout C:/tmp/dogmud-3b-helpful tools/playtest/scenarios/spell-effects-3b.yaml
```

  Extract findings to memory (reports are gitignored). A goal blocked by the environment (the charm never lands, the caster never mends) is reported as blocked, not as passed; the parity table and the area tests remain the evidence. Commit the scenario:

```bash
cd C:/tmp/dogmud-3b-helpful && git add tools/playtest/scenarios/spell-effects-3b.yaml tools/playtest/goals/scenarios/spell-effects-3b/healer.yaml tools/playtest/goals/scenarios/spell-effects-3b/partner.yaml tools/playtest/goals/scenarios/spell-effects-3b/mobwatch.yaml && git commit -F - <<'EOF'
test(playtest): slice 3b scenario, party and pet heals and a mob healer

Three feature-tester actors on one server: a charmer heals a party
member and a charmed thug with Mend All and wards and cleanses the pet
in Thornwall (462, 474); a caster fights the North Road bandits (4052)
and reads the Bandit Caster mending herself and her packmates. Fixtures
verified against YAML on 2026-09-28.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

- [ ] **Step 9: PR.** Push and open it against the fork only:

```bash
cd C:/tmp/dogmud-3b-helpful && git push -u origin feature/spell-effects-3b-helpful
```

Write the body to a file in the scratchpad with the Write tool, containing: a summary (one applier per helpful effect, help spells uncontested everywhere, one area filler), the owner rulings and 3b conditions, the thirteen "Behaviour changes" above verbatim, the design decisions 1 and 2 and how they were resolved, the guard changes (ward and regen rows collapsed into `spell_help_effects.go`, the MS rows deleted, one viewpoint row retired, the one-contest guard reads `spell_help_effects.go`), the gate results, the playtest summary, and "Slice 3 of the parity arc is complete." End the body with a blank line and `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.

```bash
gh pr create --repo pruuk/DOGMud --base master --head feature/spell-effects-3b-helpful --title "Spell effects 3b: one applier per helpful effect; creatures heal and shield" --body-file <scratchpad body file>
```

Confirm the printed URL says `pruuk/DOGMud`. Then `gh pr checks <n> --repo pruuk/DOGMud --watch`, and confirm with `gh run list --repo pruuk/DOGMud --branch feature/spell-effects-3b-helpful` that lint and tests both ran. Do not deploy; the owner deploys.

---

## Self-review against the spec

| Spec requirement (3b and owner conditions) | Task |
|---|---|
| condition: one applier over `applySpellCondition`; MP goes through it | 2 |
| heal: one applier; MP gains it; the dead PP crit deleted; `events.Healed` when a player heals (a mob, decision 3) | 3 |
| shield: one applier; PM, MM, MP gain it (a pet can be shielded); the dead PP crit deleted | 4 |
| Help skip on MP: a mob's help spell on a player is uncontested | 1 |
| Area help: one filler for player and mob casters; companions of the caster and of party members; the AI companion; the mob's own AI ally rule, cited (`FindPackmatesInRoom`, decision 1) | 7 |
| purge: one applier; mob targets gain it | 5 |
| default: the arm, unified | 5 |
| MS routes through the shared appliers | 6 |
| Parity table across PM, PP, MS, MM, MP for heal, shield, condition, purge; MP uncontested | 8 (and 1) |
| Retire `applyMobEffect`, `applyPlayerEffect`, the `*Arms` helpers, `applyMobSelfEffect`; migrate their test callers | 5 (arms), 6 (MS body), 9 (wrappers and 28 test calls) |
| Guards per spec fact 12: condition rows re-keyed and collapsed, narration row retired, one-contest guard widened, sight rows checked (unchanged names) | 1-6, 9 |
| Mob charm and mob purge-affliction punted | not built (charm on a non-mob falls to the default arm; purge-affliction untouched) |
| Docs (hooks `context.md`, PATCH_NOTES), gate, boot check, playtest (party member, pet, mob healer), PR | 10 |
