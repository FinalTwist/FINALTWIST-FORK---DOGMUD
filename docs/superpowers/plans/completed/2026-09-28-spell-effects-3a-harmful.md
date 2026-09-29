# Spell Effects 3a (Harmful) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every harmful spell effect (damage, dot, knockdown) lands through one applier per effect, whoever casts it and whoever it hits, and a player's harmful spell on a mob is a crime exactly as melee is.

**Architecture:** A new file `internal/hooks/spell_effects.go` holds `spellEffectCtx`, the dispatcher `applySpellEffect`, the three harmful appliers, the shared aggro and crime helper, and the resolver steps every pairing now shares (backfire, interrupt, record). The four contested resolvers in `spell_resolution.go` keep their names and their single `runSpellChannelAttack` call; they build a context and call the dispatcher. Every non-harmful effect keeps running on its old per-pairing arms (renamed `*Arms`) until slice 3b.

**Tech Stack:** Go, `internal/hooks`, `internal/actions` (`Actor`, `SeedAggression`), `internal/messaging` (`SendTrio`), testify.

**Spec (binding):** `docs/superpowers/specs/2026-09-28-spell-effect-unification-design.md`, section "3a harmful", the owner rulings, and "Architecture".

**Subagent models:** Tasks 1, 2, 5, 6 opus (judgment: moves that must stay behaviour-identical, the crime wiring, the resolver rewrite, the parity net). Tasks 3, 4, 7, 8, 9 sonnet (mechanical, code given). Task 9's playtest follows the `dogmud-playtesting` skill.

---

## Facts verified against source (2026-09-28, branch `docs/spells-unification-spec` at `056b1078f`)

`spell_resolution.go` did not change between the spec's base `441495b32` and this commit, but the spec's line numbers had already drifted by 26 to 66 lines; every line below was read now.

| # | Fact | Where |
|---|---|---|
| F1 | `applyMobEffect` is at line 883 (body 884-915), serves PM and MM; `applyPlayerEffect` at 973 (body 974-1247), serves PP; the MP inline switch is 1550-1732 inside `resolveMobSpellAgainstPlayer` (1527-1742), with its locals `isCrit`, `mobSpellDmg`, `critTag` at 1544-1549; `applyMobSelfEffect` at 1440 | `internal/hooks/spell_resolution.go` |
| F2 | Harmful arm bodies: `applyMobEffect_damage` 577-620, `applyMobEffect_dot` 622-677, `applyMobEffect_knockdown` 679-703 plus `applyMobKnockdownOutcome` 705-762; PP `case "damage":` 988-1024; MP `case "damage":` 1551-1584, `case "dot":` 1585-1630, `case "knockdown":` 1631-1686 | same |
| F3 | PP has no dot or knockdown arm: both fall to `default` (1209-1247), which says "Your X takes effect on Y." and applies nothing. PP's early return `if out.Defended && spellData.EffectType != "damage"` is at 980 | same |
| F4 | MM passes `user == nil`, so every PM arm line is gated off (e.g. 609, 668, 745) and `setMobSpellAggro` (566-575) commits nothing (both commits need `user != nil`); MM dot reads `casterSkill := 0`, `casterWil := 100` (645-650) | same |
| F5 | MP damage `break`s on `out.DefensiveCrit` (1559-1561) before the aggro commit at 1582-1584; MP knockdown has no `cancelDamageConditions` (1638-1641); PM knockdown has it (719) | same |
| F6 | `combat.RecordSpell` calls: 406 (PM help), 442 (PM backfire), 462 (PM landed), 1541 (MP backfire), 1735 (MP landed). None in `resolveAgainstPlayer` (929-965) or `resolveMobSpellAgainstMob` (1482-1517). Signature `RecordSpell(src, tgt SourceTarget, hit, crit, backfire, fizzle bool, dmg int, zScore float64, srcChar, tgtChar *characters.Character, round uint64)` | `spell_resolution.go`; `internal/combat/analytics.go:322` |
| F7 | Backfire blocks: PM 432-444 (caster line, room line, record), PP 936-946 (lines, no record), MM 1501-1509 (room line only, no record), MP 1533-1543 (room line, record). Only PM calls `maybeInterruptSpellOnMob` (450-457); the helper is 478-489 and calls `actions.InterruptTargetCast(&mob.Character, by)` | same |
| F8 | `actions.InterruptTargetCast(target *characters.Character, by state.ActorRef) bool` already handles a player (queues `events.CastInterrupted` for `GetUserId() > 0`); only `modules/gmcp` listens, so a player target needs its own line | `internal/actions/cast_interrupt.go:14-34` |
| F9 | Boss-interrupt spell ids default to `neural-stun`, `sensory-overload`, `kinetic-shove` | `internal/configs/config.balance.misc.go:334`; `IsBossInterruptSpell` at `config.balance.go:1320` |
| F10 | `RecordAssaultCrime(user, mob, room)` at `aggression.go:25`; `SeedAggression(user, mob, room, freshAggro)` at `aggression.go:94` fires `events.PlayerAttackedMob` on every call and, when fresh, `opinions.Bump` plus `RecordAssaultCrime`. `throw.go:144` (inside `engageAfterThrow`, 130-156) judges `freshAggro := !mob.Character.IsInCombat()` per mob. No spell path calls either | `internal/actions/aggression.go`; `internal/usercommands/throw.go` |
| F11 | `spellAudience(caster, actorName, target, acteeName, room)` never stores a typed nil | `internal/hooks/spell_audience.go:16` |
| F12 | `mobDisplayName(mob, room, viewingUserId)` at `NewRound_DoCombat_helpers.go:390` (panics on a nil room); `sendVisualRoomText` at 401; `charActorRef` at `combat_shared_helpers.go:281` (reads `c.GetUserId()` and `c.MobInstanceId`); `calcSpellDamageForCharacter` at `combat_shared_helpers.go:37` (nil caster falls to the dice path); `cancelDamageConditions` at 539; `dispatchItemProcs` at `item_procs.go:107`; `fireSpellCounterTier` at `counter_tier.go:38` | files |
| F13 | `actions.MobActor.GetMobInstanceId()` returns `a.Mob.InstanceId`; `UserActor.GetUserId()` returns `a.User.UserId`. The seeded fixture mob 100 has `Character.MobInstanceId == 0`, so refs must come from the actor, not `charActorRef` | `internal/actions/actor_mob.go:67`, `actor_user.go:62`; `internal/hooks/hooks_test.go:92-104` |
| F14 | `GetSpellStatAndSkill` returns Perception (or Charisma) plus skill for FOLD calculation. The MP dot reads `spellData.CasterStatValue(caster.Stats)` with a Manifestation-aware skill (1598-1602), and two tests pin exactly that formula (`TestDotProducerRecordsNegativeHarm_MobTarget` ~1017, `..._PlayerTarget` ~1062). See "Spec correction 1" | `internal/actions/cast.go:391`; `spell_resolution.go`; `hooks_test.go` |
| F15 | Shipped dot spells `blood-boil`, `neural-toxin` and knockdown `kinetic-shove` all declare `primarystat: willpower`, none is Manifestation | `_datafiles/world/dogmud/spells/*.yaml` |
| F16 | Guard: `TestPlayerConditionsTravelTheEventPath` (`condition_apply_path_guard_test.go:356`) keys every `AddConditionMagnitude` call by `file|line`; spell rows at 179-180 (ward 1179, 1474), 188-190 (regen 845, 1080, 1452), 201-202 (dot 666, 1617). Stale keys and new keys both fail it | repo root |
| F17 | Guard: `sightExemptSites` rows for the four resolvers by `file|func` at `sight_penalty_guard_test.go:117-120` | repo root |
| F18 | Guard: `TestSpellResolversRunOneContestAndAppliersRollNone` (`channel_defence_routing_test.go:57`) parses ONLY `spell_resolution.go` and requires the four resolvers to call `runSpellChannelAttack` once each (91-97) | `internal/hooks` |
| F19 | Guard: `TestNarrationSitesMatchViewpointAudit` (`messaging_surface_guard_test.go:1451`) holds `hooks/spell_resolution.go|<literal>` rows 1286-1293 with set equality. Rows that 3a retires: 1286 (interrupt), 1287 (backfire), 1289 (dot afflicts), 1290 (damage strikes). An event sent through `SendTrio` leaves the walk | repo root |
| F20 | About 70 direct calls in 13 hooks test files name the resolvers or appliers. None names `applyMobEffect_damage`, `_dot`, `_knockdown`, `applyMobKnockdownOutcome` or `setMobSpellAggro`. `spell_interrupt_test.go` names `maybeInterruptSpellOnMob` five times (54, 69, 82, 91, 105). `hooks_test.go` calls `applyMobEffect` with a harmful spell at 1047, 2556, 2569, 2589, 2601 (nil caster), 2628 | `internal/hooks/*_test.go` |
| F21 | Test fixture: `seedAllRegistries()` (`hooks_test.go:56`) seeds users 1 Aliceia and 2 Bobrick, mob instance 100 Skeleton, room 1 with `Pvp: true` (144). `TestMain` (34-52) points `FilePaths.DataFiles` at a temp dir, so opinion, crime and bounty saves land there | `internal/hooks/hooks_test.go` |
| F22 | Test helpers exist: `drainPlain` (`narration_testhelpers_test.go:28`), `darken` (87), `countContaining` (`combat_verbosity_wiring_test.go:82`), `spellContestAttackWin` (`spell_collapse_test.go:35`), `physicalHarmSpellForCollapseTest` (211, "Stone Lash", DamageMultiplier 1.0, magnitude 30), `roomForCollapseTest` (241, lamp 90), `pinCounterTierKnobs` (`counter_tier_test.go:31`, 0 is the off switch), `attackWinContest` (166), `mobs.SetInstanceForTest` (`internal/mobs/test_helpers.go:65`), `events.DrainQueuedPlayerAttackedMobsForTest` (`events.go:359`, 0 drains all) | files |
| F23 | Faction test sandbox: env overrides `DOGMUD_FACTIONS_DIR_OVERRIDE`, `DOGMUD_FACTIONS_REP_DIR_OVERRIDE`, `DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE`, `DOGMUD_OPINIONS_DIR_OVERRIDE`, then `factions.LoadAllDefinitions()` and the three `ClearCache`s; `crimes.AllForFaction(id, false)`; `crimes.KindAssault`, `crimes.PerpPlayer`. A missing definitions dir loads an empty registry | `internal/actions/melee_target_admission_test.go:22-40`; `internal/factions/registry.go:41-50`; `internal/crimes/types.go` |
| F24 | `resolveMobDrainArea` (1364-1437) sends its per-player lines through raw `target.SendText` with a `<ansi fg="mobname">` literal (1416-1430), so a drained player in the dark reads the mob's name | `spell_resolution.go` |
| F25 | Imports only the harmful arms use in `spell_resolution.go`: `mudlog` (734, 1654), `position` (731-732, 1651-1652); `util` only at 406, 428, 1532 | `spell_resolution.go` |

### Spec corrections found while planning

1. **Dot duration stat (spec says `GetSpellStatAndSkill`).** The spec reads "duration from the caster's own cast skill and stat via `GetSpellStatAndSkill` (Manifestation-aware, as MP does today)". MP does not use `GetSpellStatAndSkill` (F14): that helper returns Perception, the fold-calculation stat, while every live dot duration reads the spell's `primarystat` through `CasterStatValue`. Routing through `GetSpellStatAndSkill` would move every player's Blood Boil and Neural Toxin duration from Willpower to Perception, a balance change no ruling asked for, and break the two tests that pin the formula. This plan follows the spec's stated intent, "as MP does today": `spellCasterStatAndSkill` returns `CasterStatValue` plus the Manifestation-aware skill. For the shipped spells (F15) PM and MP durations are unchanged; only MM changes (fact 4 / audit row 9). **The reviewer should confirm this reading.**
2. **Line numbers.** The spec's facts table cites lines that have drifted (e.g. `applyMobEffect` 909 is 883, the MP defensive-crit `break` 1625 is 1559-1561). Content matches; this plan uses the current lines.
3. **Fact 12 count.** About 70 direct calls in 13 files, not 64 in 11 (F20). No design effect.

### Behaviour changes this PR ships (for the PR body)

1. A player's harmful spell on a mob fires `PlayerAttackedMob` every cast and, on a fresh engagement, the opinion bump and the assault crime (owner ruling 2).
2. Mob-on-mob damage, dot and knockdown are narrated to the room, commit aggro and are recorded.
3. Mob-on-mob dot duration reads the caster's skill and stat, not skill 0 and stat 100.
4. Player-on-player dot and knockdown apply (they used to say "takes effect" and do nothing).
5. Mob-on-player knockdown breaks damage-fragile conditions; a mob-on-player spell negated by a defensive crit still starts the fight.
6. A boss-interrupt spell (`neural-stun`, `sensory-overload`, `kinetic-shove`) now cancels a casting player's or mob's cast from any caster, not only a player's spell on a mob.
7. A damage or knockdown spell fully negated by a defensive crit no longer breaks damage-fragile conditions on a mob (PP and MP already behaved this way).
8. A mob caster that is not fighting commits onto its target (MP used to commit only the target's side).
9. Mob names in mob-cast lines go through `mobDisplayName` (duplicate index shown); the PM caster line and the drain lines now hide names from a reader who cannot see.
10. Interrupt wording: "its spell collapses" becomes "the spell collapses".

---

## File map

| File | Change | Responsibility |
|---|---|---|
| `internal/hooks/spell_effects.go` | **Create** (Task 1), grows in 2-5, trimmed in 8 | `spellEffectCtx`, dispatcher, harmful appliers, aggro and crime, backfire, interrupt, record seam, test-only wrappers |
| `internal/hooks/spell_resolution.go` | Modify (1-5) | resolvers build contexts; old switches become `applyMobEffectArms`, `applyPlayerEffectArms`, `applyMobOnPlayerArms`; harmful arms deleted; drain narration |
| `internal/hooks/spell_effect_fixture_test.go` | **Create** (1) | four-combatant fixture, record capture, parity spells |
| `internal/hooks/spell_effects_test.go` | **Create** (1) | context accessors |
| `internal/hooks/spell_damage_test.go` | **Create** (2) | MM visible, MP defensive crit aggro, crime |
| `internal/hooks/spell_dot_test.go` | **Create** (3) | MM duration, PP dot |
| `internal/hooks/spell_knockdown_test.go` | **Create** (4) | PP knockdown, MP breaks fragile conditions |
| `internal/hooks/spell_resolver_steps_test.go` | **Create** (5) | pairings table, backfire, record, interrupt, drain names |
| `internal/hooks/spell_effect_parity_test.go` | **Create** (6) | the parity table |
| `internal/hooks/channel_defence_routing_test.go` | Modify (7) | parse `spell_effects.go` too |
| `condition_apply_path_guard_test.go` | Modify (1-5 re-keys, 7 comments) | allowlist keys |
| `messaging_surface_guard_test.go` | Modify (2, 3, 5) | drop retired rows |
| `internal/hooks/hooks_test.go`, `internal/hooks/spell_interrupt_test.go` | Modify (8) | migrate harmful callers off the wrappers |
| `internal/hooks/context.md`, `docs/PATCH_NOTES.md`, `docs/README.md` | Modify (9) | docs |

---

## Execution setup (once, before Task 1)

- [ ] Create the worktree (Bash). Base on `docs/spells-unification-spec`; if that branch has merged, base on `master` instead.

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud" && git worktree add C:/tmp/dogmud-3a-harmful -b feature/spell-effects-3a-harmful docs/spells-unification-spec
```

- [ ] Confirm the baseline is green for the packages this plan touches:

```bash
cd C:/tmp/dogmud-3a-harmful && go test ./internal/hooks/ -count=1 && go test . -run 'TestPlayerConditionsTravelTheEventPath|TestNarrationSitesMatchViewpointAudit|TestEveryTrioLiteralNamesAllThreeRoles|TestEveryRollSiteAppliesTheSightPenalty' -count=1
```

Expected: `ok  github.com/GoMudEngine/GoMud/internal/hooks` and `ok  github.com/GoMudEngine/GoMud`. If either is red on the untouched base, stop and report.

**The root-guard command above is called "the root guards" in every task below.** Run it from the worktree root.

**Keeping the root guards green (used by Tasks 1-5).** Moving code shifts lines, so `TestPlayerConditionsTravelTheEventPath` reports each old key as stale and each moved call as unallowlisted, printing the new key (`internal/hooks/spell_resolution.go|N` or `internal/hooks/spell_effects.go|N`). For each pair, edit the allowlist in `condition_apply_path_guard_test.go`: change the key's line number, keep its reason string exactly. Match pairs by the source line the failure prints (the regen, ward or dot call). `TestNarrationSitesMatchViewpointAudit` reports stale registry rows; delete exactly the rows each task names, nothing else. If it reports an UNREGISTERED event, stop: this plan expects none.

---

### Task 1: Context, dispatcher and the record seam (pure refactor)

**Model:** opus.

**Files:**
- Create: `internal/hooks/spell_effects.go`
- Create: `internal/hooks/spell_effect_fixture_test.go`
- Create: `internal/hooks/spell_effects_test.go`
- Modify: `internal/hooks/spell_resolution.go` (170, 405-406, 442, 459-462, 876-916, 967-973 plus 1248, 1494, 1510, 1519-1742)
- Modify: `condition_apply_path_guard_test.go` (re-key)

- [ ] **Step 1: Write the fixture** `internal/hooks/spell_effect_fixture_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// spellRecordCall is one captured recordSpell call.
type spellRecordCall struct {
	src, tgt                    combat.SourceTarget
	hit, crit, backfire, fizzle bool
	dmg                         int
}

// spellParityFixture puts two players, a watcher and two mobs in lit room 1,
// every combatant with the same stats and spell skill, so a spell cast
// through any pairing must land the same amount (parity slice 3a).
type spellParityFixture struct {
	room       *rooms.Room
	casterUser *users.UserRecord // Aliceia, user 1
	targetUser *users.UserRecord // Bobrick, user 2
	watcher    *users.UserRecord // Carys, user 3: reads only room lines
	casterMob  *mobs.Mob         // Skeleton, instance 100
	targetMob  *mobs.Mob         // Ghoul, instance 101
	records    []spellRecordCall
}

// newSpellParityFixture seeds the registries, pins the ONE spell contest to
// out, and captures every recordSpell call. Every restore is a t.Cleanup,
// so they run last-in first-out after the test body.
func newSpellParityFixture(t *testing.T, out combat.ChannelDefenceResult) *spellParityFixture {
	t.Helper()
	t.Cleanup(seedAllRegistries())
	t.Cleanup(conditions.SeedConditionRecordsForTest())
	room := roomForCollapseTest(t)

	u1, u2 := users.GetByUserId(1), users.GetByUserId(2)
	u3 := users.NewTestUser(3, "cara", "Carys", 1003)
	u3.Character.RoomId = 1
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{1: u1, 2: u2, 3: u3}))
	room.AddPlayer(3)

	ghoul := &mobs.Mob{MobId: 2, InstanceId: 101, HomeRoomId: 1, Character: characters.Character{
		Name: "Ghoul", RoomId: 1, MobInstanceId: 101,
		Conditions: conditions.New(), Cooldowns: map[string]int{},
		Position: position.NewMachine(), CombatPhase: combatphase.NewMachine(),
	}}
	mobs.SetInstanceForTest(101, ghoul)
	room.AddMob(101)

	skeleton := mobs.GetInstance(100)
	skeleton.Character.MobInstanceId = 100

	f := &spellParityFixture{room: room, casterUser: u1, targetUser: u2, watcher: u3,
		casterMob: skeleton, targetMob: ghoul}
	for _, c := range []*characters.Character{u1.Character, u2.Character, &skeleton.Character, &ghoul.Character} {
		equaliseSpellCombatant(c)
	}

	originalContest := runSpellChannelAttack
	runSpellChannelAttack = func(messaging.RoomVisibility, combatvocab.Attack, combat.AttackSide,
		*characters.Character, *characters.Character) combat.ChannelDefenceResult {
		return out
	}
	t.Cleanup(func() { runSpellChannelAttack = originalContest })

	originalRecord := recordSpell
	recordSpell = func(src, tgt combat.SourceTarget, hit, crit, backfire, fizzle bool, dmg int,
		_ float64, _, _ *characters.Character, _ uint64) {
		f.records = append(f.records, spellRecordCall{src: src, tgt: tgt, hit: hit, crit: crit,
			backfire: backfire, fizzle: fizzle, dmg: dmg})
	}
	t.Cleanup(func() { recordSpell = originalRecord })

	for _, id := range []int{1, 2, 3} {
		drainPlain(id)
	}
	events.DrainQueuedPlayerAttackedMobsForTest(0)
	return f
}

// equaliseSpellCombatant gives a player or a mob the same six stats, the same
// spellcasting rank, full conviction and a deep health pool.
func equaliseSpellCombatant(c *characters.Character) {
	c.Stats.Strength.ValueAdj = 100
	c.Stats.Dexterity.ValueAdj = 100
	c.Stats.Perception.ValueAdj = 100
	c.Stats.Vitality.ValueAdj = 100
	c.Stats.Willpower.ValueAdj = 100
	c.Stats.Charisma.ValueAdj = 100
	c.SetSkill(string(skills.Spellcasting), 3)
	c.Health = 1000
	c.HealthMax.Value = 1000
	c.Conviction = 50
	c.ConvictionMax.Value = 50
}

func dotSpellForParityTest() *spells.SpellData {
	return &spells.SpellData{
		SpellId: "test-blight", Name: "Blight", AttackType: combatvocab.AttackSpell,
		DamageType: combatvocab.DamageMental, Targeting: combatvocab.TargetSingle,
		EffectType: "dot", EffectMagnitude: 10, BaseFolds: 6, PrimaryStat: "willpower",
		Schools: []string{spells.SchoolVital},
	}
}

func knockdownSpellForParityTest() *spells.SpellData {
	return &spells.SpellData{
		SpellId: "test-shove", Name: "Shove", AttackType: combatvocab.AttackSpell,
		DamageType: combatvocab.DamagePhysical, Targeting: combatvocab.TargetSingle,
		EffectType: "knockdown", DamageMultiplier: 0.5, EffectMagnitude: 20, PrimaryStat: "willpower",
		Schools: []string{spells.SchoolElemental},
	}
}
```

- [ ] **Step 2: Write the failing context test** `internal/hooks/spell_effects_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/stretchr/testify/assert"
)

// The context answers "who is who" the same way for every pairing, and takes
// refs from the actor: fixture mob 100 has no Character.MobInstanceId of its
// own in production-shaped saves either, which is why charActorRef is only
// the fallback.
func TestSpellEffectCtx_NamesAndRefsPerPairing(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := physicalHarmSpellForCollapseTest()

	pm := newSpellEffectCtx(f.casterUser.Character, actions.NewUserActorInRoom(f.casterUser, f.room),
		actions.NewMobActorInRoom(f.targetMob, f.room), f.room, spell, 30, spellContestAttackWin())
	assert.Same(t, f.casterUser, pm.casterUser())
	assert.Nil(t, pm.casterMob())
	assert.Same(t, f.targetMob, pm.targetMob())
	assert.Nil(t, pm.targetUser())
	assert.Equal(t, state.ActorRef{UserId: 1}, pm.casterRef())
	assert.Equal(t, state.ActorRef{MobInstanceId: 101}, pm.targetRef())
	assert.Equal(t, `<ansi fg="username">Aliceia</ansi>`, pm.casterName())
	assert.Equal(t, mobDisplayName(f.targetMob, f.room, 1), pm.targetName())
	assert.Equal(t, 1, pm.viewerId())

	mp := newSpellEffectCtx(&f.casterMob.Character, actions.NewMobActorInRoom(f.casterMob, f.room),
		actions.NewUserActorInRoom(f.targetUser, f.room), f.room, spell, 30, spellContestAttackWin())
	assert.Same(t, f.casterMob, mp.casterMob())
	assert.Same(t, f.targetUser, mp.targetUser())
	assert.Equal(t, state.ActorRef{MobInstanceId: 100}, mp.casterRef())
	assert.Equal(t, state.ActorRef{UserId: 2}, mp.targetRef())
	assert.Equal(t, mobDisplayName(f.casterMob, f.room, 0), mp.casterName())
	assert.Equal(t, 0, mp.viewerId())

	anon := newSpellEffectCtx(nil, nil, actions.NewMobActorInRoom(f.targetMob, f.room),
		f.room, spell, 30, spellContestAttackWin())
	assert.Nil(t, anon.casterUser())
	assert.Nil(t, anon.casterMob())
	assert.Equal(t, "something", anon.casterName())
	assert.True(t, anon.casterRef().IsZero())
}
```

- [ ] **Step 3: Run it to verify it fails**

```bash
cd C:/tmp/dogmud-3a-harmful && go test ./internal/hooks/ -run 'TestSpellEffectCtx' -count=1
```

Expected: build failure, `undefined: recordSpell` and `undefined: newSpellEffectCtx`.

- [ ] **Step 4: Create** `internal/hooks/spell_effects.go`:

```go
package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Spell effect unification (parity slices 3a and 3b,
// docs/superpowers/specs/2026-09-28-spell-effect-unification-design.md).
//
// Every spell effect on one target is applied through one spellEffectCtx and
// one dispatcher, whoever casts it and whoever it hits. The four contested
// resolvers in spell_resolution.go keep their names and their one
// runSpellChannelAttack call each; after the contest they build a context and
// hand it here.

// recordSpell is the analytics seam for one resolved cast. It defaults to
// combat.RecordSpell; same-package tests replace it with a recorder and
// restore it with t.Cleanup, the pattern runSpellChannelAttack uses.
var recordSpell = combat.RecordSpell

// spellEffectCtx is everything one spell effect needs about one target.
type spellEffectCtx struct {
	casterChar *characters.Character // nil only in tests that pass no caster
	caster     actions.Actor         // *actions.UserActor, *actions.MobActor, or nil
	target     actions.Actor         // *actions.UserActor or *actions.MobActor, never nil
	room       *rooms.Room
	spell      *spells.SpellData
	magnitude  int
	out        combat.ChannelDefenceResult // zero-valued win for uncontested casts
}

func newSpellEffectCtx(casterChar *characters.Character, caster, target actions.Actor, room *rooms.Room,
	spell *spells.SpellData, magnitude int, out combat.ChannelDefenceResult) spellEffectCtx {
	return spellEffectCtx{casterChar: casterChar, caster: caster, target: target, room: room,
		spell: spell, magnitude: magnitude, out: out}
}

// spellCasterActor wraps whichever caster a legacy call site was handed: the
// player record when there is one, else the registered mob behind casterChar,
// else nil (an anonymous caster, which only tests produce).
func spellCasterActor(user *users.UserRecord, casterChar *characters.Character, room *rooms.Room) actions.Actor {
	if user != nil {
		return actions.NewUserActorInRoom(user, room)
	}
	if casterChar != nil && casterChar.MobInstanceId > 0 {
		if m := mobs.GetInstance(casterChar.MobInstanceId); m != nil {
			return actions.NewMobActorInRoom(m, room)
		}
	}
	return nil
}

func actorUser(a actions.Actor) *users.UserRecord {
	if ua, ok := a.(*actions.UserActor); ok && ua != nil {
		return ua.User
	}
	return nil
}

func actorMob(a actions.Actor) *mobs.Mob {
	if ma, ok := a.(*actions.MobActor); ok && ma != nil {
		return ma.Mob
	}
	return nil
}

func actorRefOf(a actions.Actor) state.ActorRef {
	if a == nil {
		return state.ActorRef{}
	}
	return state.ActorRef{UserId: a.GetUserId(), MobInstanceId: a.GetMobInstanceId()}
}

func (c spellEffectCtx) casterUser() *users.UserRecord { return actorUser(c.caster) }
func (c spellEffectCtx) casterMob() *mobs.Mob           { return actorMob(c.caster) }
func (c spellEffectCtx) targetUser() *users.UserRecord { return actorUser(c.target) }
func (c spellEffectCtx) targetMob() *mobs.Mob           { return actorMob(c.target) }

func (c spellEffectCtx) targetChar() *characters.Character { return c.target.GetCharacter() }

// casterRef names the caster for aggro and harm attribution. It reads the
// actor, not the character: a mob's Character.MobInstanceId is not reliably
// set, its Mob.InstanceId is.
func (c spellEffectCtx) casterRef() state.ActorRef {
	if c.caster == nil {
		return charActorRef(c.casterChar)
	}
	return actorRefOf(c.caster)
}

func (c spellEffectCtx) targetRef() state.ActorRef { return actorRefOf(c.target) }

// viewerId is the user id mob names are rendered for: the player caster's,
// or 0 when a mob casts.
func (c spellEffectCtx) viewerId() int {
	if u := c.casterUser(); u != nil {
		return u.UserId
	}
	return 0
}

// spellEffectName is a party's name exactly as the spell lines print it, and
// so exactly as SendTrio must hide it: players in the username tag, mobs
// through mobDisplayName with the room's duplicate index.
func spellEffectName(a actions.Actor, room *rooms.Room, viewerId int) string {
	switch v := a.(type) {
	case *actions.UserActor:
		return fmt.Sprintf(`<ansi fg="username">%s</ansi>`, v.User.Character.Name)
	case *actions.MobActor:
		return mobDisplayName(v.Mob, room, viewerId)
	}
	return "something"
}

func (c spellEffectCtx) casterName() string { return spellEffectName(c.caster, c.room, c.viewerId()) }
func (c spellEffectCtx) targetName() string { return spellEffectName(c.target, c.room, c.viewerId()) }

func (c spellEffectCtx) critTag() string {
	if c.out.AttackerCrit {
		return ` <ansi fg="yellow">[CRIT!]</ansi>`
	}
	return ""
}

func (c spellEffectCtx) category() messaging.Category { return spellSchoolCategory(c.spell) }

// audience is the SendTrio audience for a line between caster and target.
// A mob side gets no private line (spellAudience stores no nil recipient),
// and the room line excludes whichever sides are players.
func (c spellEffectCtx) audience() messaging.Audience {
	return spellAudience(c.casterUser(), c.casterName(), c.targetUser(), c.targetName(), c.room)
}

func spellSourceTarget(a actions.Actor) combat.SourceTarget {
	if a != nil && a.IsPlayer() {
		return combat.User
	}
	return combat.Mob
}

// applySpellEffect applies one spell effect to one target and returns the
// damage it dealt (0 for effects that deal none). Until slice 3b, effects
// without a unified applier run on the per-pairing arms they always had.
func applySpellEffect(c spellEffectCtx) int {
	switch {
	case c.targetMob() != nil:
		return applyMobEffectArms(c)
	case c.casterMob() != nil:
		return applyMobOnPlayerArms(c)
	default:
		applyPlayerEffectArms(c)
		return 0
	}
}

// ── Test-only wrappers. Slice 3b's last task deletes them once no test
// names them. ─────────────────────────────────────────────────────────────

// applyMobEffect applies a spell effect to a mob target. user may be nil
// (a mob caster); casterChar may be nil (an anonymous caster).
func applyMobEffect(user *users.UserRecord, casterChar *characters.Character, mob *mobs.Mob, room *rooms.Room,
	spellData *spells.SpellData, magnitude int, out combat.ChannelDefenceResult) int {
	return applySpellEffect(newSpellEffectCtx(casterChar, spellCasterActor(user, casterChar, room),
		actions.NewMobActorInRoom(mob, room), room, spellData, magnitude, out))
}

// applyPlayerEffect applies a player's spell effect to a player target.
func applyPlayerEffect(user *users.UserRecord, target *users.UserRecord, room *rooms.Room,
	spellData *spells.SpellData, magnitude int, out combat.ChannelDefenceResult) {
	applySpellEffect(newSpellEffectCtx(user.Character, actions.NewUserActorInRoom(user, room),
		actions.NewUserActorInRoom(target, room), room, spellData, magnitude, out))
}
```

- [ ] **Step 5: Turn the three switches into arms functions** in `spell_resolution.go`. Each move is verbatim; the two new first lines rebuild the old parameter names as locals, so the moved body needs **no substitutions**.

  a. Replace lines 876-883 (the `applyMobEffect` doc comment and signature) with:

```go
// applyMobEffectArms is the pre-unification switch for a MOB target (PM and
// MM). The dispatcher routes here every effect that has no unified applier
// yet. user is nil when a mob casts; casterChar is nil for an anonymous
// caster.
//
// U6b Task 4: `out` is the resolver's ONE channel contest, threaded through.
func applyMobEffectArms(c spellEffectCtx) int {
	user, casterChar, mob, room := c.casterUser(), c.casterChar, c.targetMob(), c.room
	spellData, magnitude, out := c.spell, c.magnitude, c.out
```

  Lines 884-916 (from `critTag := ""` through the closing `}`) stay as they are.

  b. Replace lines 967-973 (the `applyPlayerEffect` doc comment and signature) with:

```go
// applyPlayerEffectArms is the pre-unification switch for a player caster
// and a PLAYER target (PP). The dispatcher routes here every effect that has
// no unified applier yet.
//
// U6b Task 4: `out` is the resolver's ONE channel contest, threaded through
// (help spells with no defense pass an uncontested attack win). Non-damage
// effects are binary statuses: a defended cast narrates the channel defence
// triad and applies nothing, mirroring ExecuteSkillMove's StatusApplied split.
func applyPlayerEffectArms(c spellEffectCtx) {
	user, target, room := c.casterUser(), c.targetUser(), c.room
	spellData, magnitude, out := c.spell, c.magnitude, c.out
```

  Lines 974-1248 stay as they are.

  c. Extract the MP switch. Cut lines 1544-1732 (from `isCrit := out.AttackerCrit` through the closing `}` of `switch spellData.EffectType {`) out of `resolveMobSpellAgainstPlayer` and paste them, unchanged, into this new function placed directly after `resolveMobSpellAgainstPlayer`'s closing brace:

```go
// applyMobOnPlayerArms is the pre-unification switch for a MOB caster and a
// PLAYER target (MP), moved out of resolveMobSpellAgainstPlayer unchanged.
// The dispatcher routes here every effect that has no unified applier yet.
func applyMobOnPlayerArms(c spellEffectCtx) int {
	caster, target, room := c.casterMob(), c.targetUser(), c.room
	spellData, magnitude, out := c.spell, c.magnitude, c.out
	// <paste lines 1544-1732 here, unchanged>
	return mobSpellDmg
}
```

  In `resolveMobSpellAgainstPlayer`, where the cut block was, insert:

```go
	mobSpellDmg := applySpellEffect(newSpellEffectCtx(&caster.Character, actions.NewMobActorInRoom(caster, room),
		actions.NewUserActorInRoom(target, room), room, spellData, magnitude, out))
```

  and in its `combat.RecordSpell(combat.Mob, combat.User, ...)` call (old 1735) replace `isCrit` with `out.AttackerCrit`. Replace the resolver's doc comment sentence "consumes the result inline (this path predates the applier split and keeps its inline effect arms)" with "applies the effect through applySpellEffect".

- [ ] **Step 6: Point the resolvers at the dispatcher** (substitutions, all in `spell_resolution.go`):

| Old line | Old text | New text |
|---|---|---|
| 170 | `applyPlayerEffect(user, targetUser, room, spellData, magnitude, combat.ChannelDefenceResult{DamageMultiplier: 1})` | `applySpellEffect(newSpellEffectCtx(user.Character, actions.NewUserActorInRoom(user, room), actions.NewUserActorInRoom(targetUser, room), room, spellData, magnitude, combat.ChannelDefenceResult{DamageMultiplier: 1}))` |
| 405 | `dmgDealt := applyMobEffect(user, user.Character, mob, room, spellData, magnitude, combat.ChannelDefenceResult{DamageMultiplier: 1})` | `dmgDealt := applySpellEffect(newSpellEffectCtx(user.Character, actions.NewUserActorInRoom(user, room), actions.NewMobActorInRoom(mob, room), room, spellData, magnitude, combat.ChannelDefenceResult{DamageMultiplier: 1}))` |
| 459 | `dmgDealt := applyMobEffect(user, user.Character, mob, room, spellData, magnitude, out)` | `dmgDealt := applySpellEffect(newSpellEffectCtx(user.Character, actions.NewUserActorInRoom(user, room), actions.NewMobActorInRoom(mob, room), room, spellData, magnitude, out))` |
| 948 | `applyPlayerEffect(user, target, room, spellData, magnitude, out)` | `applySpellEffect(newSpellEffectCtx(user.Character, actions.NewUserActorInRoom(user, room), actions.NewUserActorInRoom(target, room), room, spellData, magnitude, out))` |
| 1494 | `applyMobEffect(nil, &caster.Character, target, room, spellData, magnitude, combat.ChannelDefenceResult{DamageMultiplier: 1})` | `applySpellEffect(newSpellEffectCtx(&caster.Character, actions.NewMobActorInRoom(caster, room), actions.NewMobActorInRoom(target, room), room, spellData, magnitude, combat.ChannelDefenceResult{DamageMultiplier: 1}))` |
| 1510 | `applyMobEffect(nil, &caster.Character, target, room, spellData, magnitude, out)` | `applySpellEffect(newSpellEffectCtx(&caster.Character, actions.NewMobActorInRoom(caster, room), actions.NewMobActorInRoom(target, room), room, spellData, magnitude, out))` |
| 406, 442, 462, 1541, 1735 | `combat.RecordSpell(` | `recordSpell(` |

Note the MM rows: the old calls passed `user == nil`, so `spellCasterActor` would have looked the caster up; passing the `*mobs.Mob` directly is equivalent and needs no registry.

- [ ] **Step 7: Build and run the new test**

```bash
cd C:/tmp/dogmud-3a-harmful && go build ./... && go test ./internal/hooks/ -run 'TestSpellEffectCtx' -count=1
```

Expected: `ok  github.com/GoMudEngine/GoMud/internal/hooks`.

- [ ] **Step 8: Run the whole hooks package** (the refactor must change nothing):

```bash
cd C:/tmp/dogmud-3a-harmful && go test ./internal/hooks/ -count=1
```

Expected: `ok`. Any failure here is a move error; fix the move, never the test.

- [ ] **Step 9: Root guards.** Run the root guards. Expected: `TestPlayerConditionsTravelTheEventPath` reports the moved `spell_resolution.go` keys (the regen and ward lines inside the PP and MP bodies and the MP dot line); re-key them per "Keeping the root guards green". `TestNarrationSitesMatchViewpointAudit` must pass unchanged (no literal left its file). Rerun until `ok`.

- [ ] **Step 10: Commit**

```bash
cd C:/tmp/dogmud-3a-harmful && git add internal/hooks/spell_effects.go internal/hooks/spell_effects_test.go internal/hooks/spell_effect_fixture_test.go internal/hooks/spell_resolution.go condition_apply_path_guard_test.go && git commit -F - <<'EOF'
refactor(spells): one spell effect context and dispatcher

spellEffectCtx carries caster, target, room, spell, magnitude and the
contest result; applySpellEffect routes to the three pre-unification
switches, now applyMobEffectArms, applyPlayerEffectArms and
applyMobOnPlayerArms (the MP switch moved out of its resolver
unchanged). recordSpell becomes a test seam over combat.RecordSpell.
applyMobEffect and applyPlayerEffect survive as test-only wrappers.
No behaviour change.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 2: One damage applier, with aggro and crime

**Model:** opus.

**Files:**
- Create: `internal/hooks/spell_damage_test.go`
- Modify: `internal/hooks/spell_effects.go`
- Modify: `internal/hooks/spell_resolution.go` (delete `applyMobEffect_damage`, three damage arms)
- Modify: `condition_apply_path_guard_test.go` (re-key), `messaging_surface_guard_test.go` (drop row 1290)

- [ ] **Step 1: Write the failing tests** `internal/hooks/spell_damage_test.go`:

```go
package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Slice 3a (audit row 7): a mob's damage spell at another mob used to land in
// silence and start no fight, because every line and both aggro commits were
// gated on a player caster.
func TestSpellDamage_MobOnMobIsSeenAndStartsAFight(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := physicalHarmSpellForCollapseTest()

	resolveMobSpellAgainstMob(f.casterMob, f.targetMob, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), spell.EffectMagnitude)

	assert.Less(t, f.targetMob.Character.Health, 1000, "the spell must land")
	assert.Equal(t, state.ActorRef{MobInstanceId: 100}, f.targetMob.Character.CurrentCombatTarget(),
		"the target must turn on its caster")
	assert.Equal(t, 1, countContaining(drainPlain(3), "Stone Lash strikes Ghoul!"),
		"a watcher must see a mob's spell land on another mob")
}

// A defensive crit negates the damage but the spell was still an attack. The
// mob-on-player arm used to break out before its aggro commit.
func TestSpellDamage_MobOnPlayerDefensiveCritStillStartsAFight(t *testing.T) {
	pinCounterTierKnobs(t, 0) // 0 is the documented off switch: no counter-swing muddies the read
	f := newSpellParityFixture(t, combat.ChannelDefenceResult{
		Defended: true, DefensiveCrit: true, Defence: combatvocab.DefenceQuell})
	spell := physicalHarmSpellForCollapseTest()

	resolveMobSpellAgainstPlayer(f.casterMob, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), spell.EffectMagnitude)

	assert.Equal(t, 1000, f.targetUser.Character.Health, "a defensive crit negates the damage")
	assert.Equal(t, state.ActorRef{MobInstanceId: 100}, f.targetUser.Character.CurrentCombatTarget(),
		"the target must still turn on its caster")
}

// Owner ruling 2 (2026-09-28): a player's harmful spell on a mob is an
// assault, recorded through actions.SeedAggression exactly as throw records
// one. The first cast is fresh aggression and records the crime; a second
// cast in the same fight is not fresh, so it records no new crime, but it
// still counts as aggression for the revenge and opinion seeders.
func TestSpellHarmOnAFactionMobIsACrime(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	// Registered BEFORE the Setenv calls so it runs after they restore the
	// environment: the reload then finds no definitions and leaves an empty
	// registry for the rest of the package.
	t.Cleanup(func() {
		_ = factions.LoadAllDefinitions()
		factions.ClearCache()
		crimes.ClearCache()
		opinions.ClearCache()
	})
	definitions := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", definitions)
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", t.TempDir())
	require.NoError(t, os.WriteFile(filepath.Join(definitions, "thornwall_citizens.yaml"), []byte(`faction_id: thornwall_citizens
display_name: "Citizens"
description: "test faction"
default_rep: 0
allies: []
enemies: []
`), 0644))
	require.NoError(t, factions.LoadAllDefinitions())
	factions.ClearCache()
	crimes.ClearCache()
	opinions.ClearCache()
	f.targetMob.Groups = []string{"thornwall_citizens"}

	spell := physicalHarmSpellForCollapseTest()
	cast := func() {
		resolveAgainstMob(f.casterUser, f.targetMob, f.room, spell,
			spellAttackSideFor(spell, f.casterUser.Character, nil), spell.EffectMagnitude)
	}

	cast()
	recorded := crimes.AllForFaction("thornwall_citizens", false)
	require.Len(t, recorded, 1, "the first harmful cast is an assault")
	assert.Equal(t, crimes.KindAssault, recorded[0].Kind)
	// If only this line fails, the victim could not see the caster: check
	// messaging.CanSeeClearly for a speciesless fixture mob before touching
	// production code. The crime row itself is the ruling under test.
	assert.Equal(t, crimes.PerpPlayer, recorded[0].Perpetrator.Type)
	assert.Len(t, events.DrainQueuedPlayerAttackedMobsForTest(1), 1)

	cast()
	assert.Len(t, crimes.AllForFaction("thornwall_citizens", false), 1,
		"a second cast in the same fight is not a new assault")
	assert.Len(t, events.DrainQueuedPlayerAttackedMobsForTest(1), 1,
		"every harmful cast still counts as aggression")
}
```

- [ ] **Step 2: Run them to verify they fail**

```bash
cd C:/tmp/dogmud-3a-harmful && go test ./internal/hooks/ -run 'TestSpellDamage_|TestSpellHarmOnAFactionMobIsACrime' -count=1 -v
```

Expected: three FAILs. MM: `CurrentCombatTarget` is the zero ref and the watcher count is 0. MP: `CurrentCombatTarget` is the zero ref. Crime: `crimes.AllForFaction` has length 0.

- [ ] **Step 3: Add the applier and the aggro helper** to `spell_effects.go`. Add `"github.com/GoMudEngine/GoMud/internal/targeting"` to its imports, and add the dispatch line at the top of `applySpellEffect`:

```go
func applySpellEffect(c spellEffectCtx) int {
	switch c.spell.EffectType {
	case "damage":
		return applySpellDamage(c)
	}
	switch {
	case c.targetMob() != nil:
		return applyMobEffectArms(c)
	case c.casterMob() != nil:
		return applyMobOnPlayerArms(c)
	default:
		applyPlayerEffectArms(c)
		return 0
	}
}
```

Then add, below `applySpellEffect`:

```go
// commitHarmfulSpellAggro is the one place a harmful spell starts a fight,
// for every pairing. fresh is whether the target was out of combat BEFORE
// this cast landed (the applier reads it first, because harm can end the
// target's fight). The target turns on its caster only when fresh, so an
// established fight is not yanked around; the caster turns on the target
// when it is not already fighting.
//
// A player's harm on a mob is also an assault (owner ruling, 2026-09-28):
// actions.SeedAggression fires PlayerAttackedMob on every cast and, when
// fresh, the opinion bump and the assault crime. Freshness is judged per
// target from the mob's own prior combat, exactly as usercommands/throw.go's
// engageAfterThrow judges it for an area throw.
func commitHarmfulSpellAggro(c spellEffectCtx, fresh bool) {
	tc := c.targetChar()
	if fresh {
		targeting.Commit(tc, c.casterRef(), targeting.ReasonAttack)
	}
	if c.casterChar != nil && !c.casterChar.IsInCombat() {
		targeting.Commit(c.casterChar, c.targetRef(), targeting.ReasonAttack)
	}
	if u, m := c.casterUser(), c.targetMob(); u != nil && m != nil {
		actions.SeedAggression(u, m, c.room, fresh)
	}
}

// applySpellDamage is the one damage applier (slice 3a). The resolver ran
// the ONE contest; this consumes it. A defended cast lands partial damage, a
// defensive crit negates it, and either way the cast was an attack.
func applySpellDamage(c spellEffectCtx) int {
	tc := c.targetChar()
	fresh := !tc.IsInCombat()
	dmg := scaleSpellDamageByDefence(
		calcSpellDamageForCharacter(c.spell, c.casterChar, tc, c.magnitude, c.out.AttackerCrit), c.out)
	sendSpellChannelDefenceMessages(c.room, c.category(), c.out,
		spellDefenceIdentity(c.casterChar, c.casterUser(), c.room),
		spellDefenceIdentity(tc, c.targetUser(), c.room), c.spell.Name, c.casterUser(), c.targetUser())
	if c.out.DefensiveCrit {
		dmg = 0
	} else {
		tc.ApplyHarm(characters.PoolHealth, dmg, c.casterRef())
		cancelDamageConditions(tc)
		// on_spell_hit item procs fire only on a harm hit that dealt damage;
		// the proc's own chance and cooldown pace an area cast.
		if dmg > 0 {
			dispatchItemProcs("on_spell_hit", c.casterChar, tc, nil, dmg)
		}
	}
	commitHarmfulSpellAggro(c, fresh)
	if c.out.Defended {
		return dmg // the defence triad above already told everyone
	}
	dmgDesc := combat.GetDamageDescription(dmg, tc.HealthMax.Value)
	messaging.SendTrio(messaging.Trio{
		Actor: messaging.Say(c.category(), fmt.Sprintf(
			`Your %s strikes %s! (<ansi fg="damage">%s</ansi>)%s`,
			c.spell.Name, c.targetName(), dmgDesc, c.critTag())),
		Actee: messaging.Say(c.category(), fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> strikes you! (<ansi fg="damage">%s</ansi>)%s`,
			c.casterName(), c.spell.Name, dmgDesc, c.critTag())),
		Observer: messaging.Say(c.category(), fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> strikes %s!`,
			c.casterName(), c.spell.Name, c.targetName())),
	}, c.audience())
	return dmg
}
```

- [ ] **Step 4: Delete the three old damage arms** in `spell_resolution.go`:
  - Delete `applyMobEffect_damage` with its two-line doc comment (old 577-620).
  - In `applyMobEffectArms`, delete the two lines `case "damage":` and `return applyMobEffect_damage(user, casterChar, mob, room, spellData, magnitude, out, critTag, mName)`.
  - In `applyPlayerEffectArms`, change `if out.Defended && spellData.EffectType != "damage" {` to `if out.Defended {`, and delete the whole `case "damage":` arm (old 988-1024, from `case "damage":` through the `}` that closes `if !out.Defended {`). The next line is then `case "purge":`.
  - In `applyMobOnPlayerArms`, delete the whole `case "damage":` arm (old 1551-1584, from `case "damage":` through the `}` closing `if !target.Character.IsInCombat() {`). The next line is then `case "dot":`.
  - Keep `setMobSpellAggro`: the dot, knockdown and condition arms still call it.

- [ ] **Step 5: Run the new tests**

```bash
cd C:/tmp/dogmud-3a-harmful && go build ./... && go test ./internal/hooks/ -run 'TestSpellDamage_|TestSpellHarmOnAFactionMobIsACrime|TestSpellEffectCtx' -count=1 -v
```

Expected: all PASS.

- [ ] **Step 6: Run the hooks package**

```bash
cd C:/tmp/dogmud-3a-harmful && go test ./internal/hooks/ -count=1
```

Expected: `ok`. A failure that pins old text or old silence is a behaviour this slice changes on purpose only if it is in "Behaviour changes" 1, 2, 5, 7, 8 or 9; update that assertion and say which change it reflects in the commit body. Anything else is a defect in the applier.

- [ ] **Step 7: Root guards.** Delete the row `"hooks/spell_resolution.go|Your %s strikes %s! (<ansi fg=\"damage\">%s</ansi>)%s"` (old line 1290) from `narrationViewpointRegistry` in `messaging_surface_guard_test.go`; the event now goes through `SendTrio` and left the walk. Re-key the condition rows. Rerun until `ok`.

- [ ] **Step 8: Commit**

```bash
cd C:/tmp/dogmud-3a-harmful && git add internal/hooks/spell_effects.go internal/hooks/spell_resolution.go internal/hooks/spell_damage_test.go condition_apply_path_guard_test.go messaging_surface_guard_test.go && git commit -F - <<'EOF'
feat(spells): one damage applier; a harmful spell on a mob is a crime

applySpellDamage replaces the PM/MM, PP and MP damage arms. It narrates
per audience through SendTrio, so a mob-on-mob hit reaches the room,
and commits aggro through commitHarmfulSpellAggro for every pairing,
including a mob-on-player cast negated by a defensive crit. A player
caster on a mob calls actions.SeedAggression with freshness judged per
target, as throw does, which records the assault (owner ruling 2).

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 3: One dot applier (the caster's own stat and skill; PP gains it)

**Model:** sonnet.

**Files:**
- Create: `internal/hooks/spell_dot_test.go`
- Modify: `internal/hooks/spell_effects.go`, `internal/hooks/spell_resolution.go`
- Modify: `condition_apply_path_guard_test.go` (the two dot rows collapse), `messaging_surface_guard_test.go` (drop row 1289)

- [ ] **Step 1: Write the failing tests** `internal/hooks/spell_dot_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Audit row 9: the mob-on-mob dot read its duration from a nil caster (skill
// 0, stat 100). The fixture caster has spellcasting 3 and willpower 100, so
// the old arm gives 30 triggers here and the caster's own numbers give 33.
func TestSpellDot_MobOnMobDurationReadsTheCaster(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := dotSpellForParityTest()

	resolveMobSpellAgainstMob(f.casterMob, f.targetMob, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), spell.EffectMagnitude)

	recs := f.targetMob.Character.GetConditions(conditions.ConditionIdPoisoned)
	require.Len(t, recs, 1)
	assert.Equal(t, calcSpellDuration(spell.BaseFolds, 3, 100)/3, recs[0].TriggersLeft)
	assert.Equal(t, 1, countContaining(drainPlain(3), "Blight afflicts Ghoul!"))
}

// A player's dot on a player fell through to applyPlayerEffect's default arm,
// which said "takes effect" and applied nothing.
func TestSpellDot_PlayerOnPlayerAfflicts(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := dotSpellForParityTest()

	resolveAgainstPlayer(f.casterUser, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), spell.EffectMagnitude)

	recs := f.targetUser.Character.GetConditions(conditions.ConditionIdPoisoned)
	require.Len(t, recs, 1)
	assert.Equal(t, calcSpellDuration(spell.BaseFolds, 3, 100)/3, recs[0].TriggersLeft)
	assert.Equal(t, 1, countContaining(drainPlain(2), "Aliceia's Blight afflicts you!"))
}
```

- [ ] **Step 2: Run them to verify they fail**

```bash
cd C:/tmp/dogmud-3a-harmful && go test ./internal/hooks/ -run 'TestSpellDot_' -count=1 -v
```

Expected: MM FAIL `expected: 33 actual: 30` and watcher count 0; PP FAIL on `require.Len` (length 0).

- [ ] **Step 3: Add the applier** to `spell_effects.go`. Add imports `"github.com/GoMudEngine/GoMud/internal/conditions"` and `"github.com/GoMudEngine/GoMud/internal/skills"`. Add `case "dot": return applySpellDot(c)` to the first switch in `applySpellEffect`, then:

```go
// spellCasterStatAndSkill is the caster side of a spell's duration: the
// spell's own primarystat through CasterStatValue and the school's cast
// skill (Manifestation for that school, Spellcasting otherwise), the formula
// the mob-on-player dot always used. A nil caster reads stat 100 and skill 0,
// the old anonymous-caster default, which only tests reach.
//
// Not actions.GetSpellStatAndSkill: that one returns Perception, the FOLD
// stat, and would move every dot off its declared primarystat.
func spellCasterStatAndSkill(spell *spells.SpellData, caster *characters.Character) (stat int, skill int) {
	if caster == nil {
		return 100, 0
	}
	castSkill := skills.Spellcasting
	if spell.HasSchool(spells.SchoolManifestation) {
		castSkill = skills.Manifestation
	}
	return spell.CasterStatValue(caster.Stats), caster.GetSkillLevel(castSkill)
}

// applySpellDot is the one damage-over-time applier (slice 3a). The
// affliction is binary: it lands only on an attack win, and a defended cast
// narrates the defence triad and applies nothing. Either way the cast was an
// attack.
func applySpellDot(c spellEffectCtx) int {
	tc := c.targetChar()
	fresh := !tc.IsInCombat()
	if c.out.Defended {
		sendSpellChannelDefenceMessages(c.room, c.category(), c.out,
			spellDefenceIdentity(c.casterChar, c.casterUser(), c.room),
			spellDefenceIdentity(tc, c.targetUser(), c.room), c.spell.Name, c.casterUser(), c.targetUser())
		commitHarmfulSpellAggro(c, fresh)
		return 0
	}
	stat, skill := spellCasterStatAndSkill(c.spell, c.casterChar)
	dotDuration := calcSpellDuration(c.spell.BaseFolds, skill, stat) / 3
	if dotDuration < 3 {
		dotDuration = 3
	}
	// Condition 121 ticks every round, so dotDuration is the trigger count.
	// The record's negative magnitude is the harm per tick, floored at one.
	dotAmount := c.magnitude
	if dotAmount < 1 {
		dotAmount = 1
	}
	// The character door, on purpose: an immune target refuses the record,
	// and nothing that did not happen may be narrated.
	afflicted := tc.AddConditionMagnitude(conditions.ConditionIdPoisoned, dotDuration, -float64(dotAmount), "spell") == nil
	commitHarmfulSpellAggro(c, fresh)
	if !afflicted {
		return 0
	}
	messaging.SendTrio(messaging.Trio{
		Actor: messaging.Say(c.category(), fmt.Sprintf(
			`Your %s afflicts %s!%s`, c.spell.Name, c.targetName(), c.critTag())),
		Actee: messaging.Say(c.category(), fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> afflicts you!%s`, c.casterName(), c.spell.Name, c.critTag())),
		Observer: messaging.Say(c.category(), fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> afflicts %s!`, c.casterName(), c.spell.Name, c.targetName())),
	}, c.audience())
	return 0
}
```

- [ ] **Step 4: Delete the old dot arms** in `spell_resolution.go`:
  - Delete `applyMobEffect_dot` with its doc comment (old 622-677).
  - In `applyMobEffectArms`, delete `case "dot":` and `return applyMobEffect_dot(user, casterChar, mob, room, spellData, magnitude, out, critTag, mName)`.
  - In `applyMobOnPlayerArms`, delete the whole `case "dot":` arm (old 1585-1630, from `case "dot":` through the `}` closing its trailing `if !target.Character.IsInCombat() {`). The next line is then `case "knockdown":`.

- [ ] **Step 5: Run the tests**

```bash
cd C:/tmp/dogmud-3a-harmful && go build ./... && go test ./internal/hooks/ -run 'TestSpellDot_|TestDotProducerRecordsNegativeHarm|TestApplyMobEffect_DotEffect|TestSpellEffectCtx|TestSpellDamage_' -count=1 -v
```

Expected: all PASS, including the two existing `TestDotProducerRecordsNegativeHarm_*` pins (the formula is unchanged for them, Spec correction 1).

- [ ] **Step 6: Run the hooks package.** `go test ./internal/hooks/ -count=1`, expected `ok`. `condition_notice_test.go`'s immune-victim test must still show no "afflicts" line.

- [ ] **Step 7: Root guards.**
  - `TestPlayerConditionsTravelTheEventPath` reports both dot keys (`spell_resolution.go|666` and `|1617`, or their Task-2 re-keyed numbers) stale and one new key `internal/hooks/spell_effects.go|N` for the `tc.AddConditionMagnitude(conditions.ConditionIdPoisoned, ...)` line. Replace the two dot rows with ONE row keyed `internal/hooks/spell_effects.go|N`, reason string unchanged. Re-key any other shifted rows.
  - Delete the row `"hooks/spell_resolution.go|Your %s afflicts %s!%s"` (old 1289) from `narrationViewpointRegistry`.
  - Rerun until `ok`.

- [ ] **Step 8: Commit**

```bash
cd C:/tmp/dogmud-3a-harmful && git add internal/hooks/spell_effects.go internal/hooks/spell_resolution.go internal/hooks/spell_dot_test.go condition_apply_path_guard_test.go messaging_surface_guard_test.go && git commit -F - <<'EOF'
feat(spells): one dot applier reading the caster's own stat and skill

applySpellDot replaces the PM/MM and MP dot arms, and PP gains the dot
it never had. Duration reads CasterStatValue plus the Manifestation-
aware cast skill, the formula MP always used, so a mob's dot on a mob
stops reading skill 0 and stat 100 (audit row 9). Not
GetSpellStatAndSkill, which is the Perception fold stat. The two dot
rows in the condition-path allowlist collapse to one.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 4: One knockdown applier (PP gains it; MP breaks damage-fragile conditions)

**Model:** sonnet.

**Files:**
- Create: `internal/hooks/spell_knockdown_test.go`
- Modify: `internal/hooks/spell_effects.go`, `internal/hooks/spell_resolution.go`
- Modify: `condition_apply_path_guard_test.go` (re-key)

- [ ] **Step 1: Write the failing tests** `internal/hooks/spell_knockdown_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A player's knockdown on a player fell through to the default arm and
// applied nothing.
func TestSpellKnockdown_PlayerOnPlayerKnocksDown(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	spell := knockdownSpellForParityTest()

	resolveAgainstPlayer(f.casterUser, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), spell.EffectMagnitude)

	assert.True(t, f.targetUser.Character.IsSupine() || f.targetUser.Character.IsProne(),
		"the target must be knocked down")
	assert.Less(t, f.targetUser.Character.Health, 1000, "the knockdown also deals damage")
	assert.Equal(t, 1, countContaining(drainPlain(2), "Aliceia's Shove slams you to the ground!"))
}

const knockdownTranceConditionId = 7101 // clear of the fixture's 100/101 and narration's 7001-7009

// The mob-on-player knockdown dealt damage but skipped cancelDamageConditions,
// so a condition that ends on damage survived a hit the player-cast
// knockdown would have broken.
func TestSpellKnockdown_MobOnPlayerBreaksDamageFragileConditions(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		knockdownTranceConditionId: {ConditionId: knockdownTranceConditionId, Name: "Test Trance",
			RoundInterval: 5, TriggerCount: 3, Flags: []conditions.Flag{conditions.CancelOnDamage}},
	}))
	require.True(t, f.targetUser.Character.Conditions.AddCondition(knockdownTranceConditionId, false))
	require.True(t, f.targetUser.Character.HasConditionFlag(conditions.CancelOnDamage))
	spell := knockdownSpellForParityTest()

	resolveMobSpellAgainstPlayer(f.casterMob, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, &f.casterMob.Character, nil), spell.EffectMagnitude)

	assert.Less(t, f.targetUser.Character.Health, 1000, "the knockdown must deal damage")
	assert.False(t, f.targetUser.Character.HasConditionFlag(conditions.CancelOnDamage),
		"damage must break a condition that ends on damage")
}
```

- [ ] **Step 2: Run them to verify they fail**

```bash
cd C:/tmp/dogmud-3a-harmful && go test ./internal/hooks/ -run 'TestSpellKnockdown_' -count=1 -v
```

Expected: PP FAIL "the target must be knocked down" (health also unchanged); MP FAIL "damage must break a condition that ends on damage".

- [ ] **Step 3: Add the applier** to `spell_effects.go`. Add imports `"github.com/GoMudEngine/GoMud/internal/mudlog"` and `"github.com/GoMudEngine/GoMud/internal/state/position"`. Add `case "knockdown": return applySpellKnockdown(c)` to the first switch in `applySpellEffect`, then:

```go
// applySpellKnockdown is the one knockdown applier (slice 3a). The defence
// scales the DAMAGE (a defended cast still lands a partial hit, a defensive
// crit negates it), while the knockdown is binary and lands only on an
// attack win: ExecuteSkillMove's Hit/StatusApplied split.
func applySpellKnockdown(c spellEffectCtx) int {
	tc := c.targetChar()
	fresh := !tc.IsInCombat()
	dmg := scaleSpellDamageByDefence(
		calcSpellDamageForCharacter(c.spell, c.casterChar, tc, c.magnitude, c.out.AttackerCrit), c.out)
	if c.out.DefensiveCrit {
		dmg = 0
	} else {
		tc.ApplyHarm(characters.PoolHealth, dmg, c.casterRef())
		cancelDamageConditions(tc)
		if dmg > 0 {
			dispatchItemProcs("on_spell_hit", c.casterChar, tc, nil, dmg)
		}
	}
	// Spell knockdowns put the target on its back (Supine); a target already
	// grappled or down takes the damage but no knockdown is narrated.
	knocked := false
	if !c.out.Defended {
		knocked = true
		if err := tc.Position.TransitionToSupine(
			position.SupineData{MinRecoveryRounds: 1},
			state.TransitionReason{Trigger: position.TriggerKnockdownSpell},
		); err != nil {
			mudlog.Warn("applySpellKnockdown: TransitionToSupine failed", "target", c.targetRef(), "err", err)
			knocked = false
		}
	}
	commitHarmfulSpellAggro(c, fresh)
	sendSpellChannelDefenceMessages(c.room, c.category(), c.out,
		spellDefenceIdentity(c.casterChar, c.casterUser(), c.room),
		spellDefenceIdentity(tc, c.targetUser(), c.room), c.spell.Name, c.casterUser(), c.targetUser())
	if c.out.Defended {
		return dmg // the triad narrated it; a defended cast never knocks down
	}
	dmgDesc := combat.GetDamageDescription(dmg, tc.HealthMax.Value)
	if knocked {
		messaging.SendTrio(messaging.Trio{
			Actor: messaging.Say(c.category(), fmt.Sprintf(
				`Your %s slams %s to the ground! (<ansi fg="damage">%s</ansi>)%s`,
				c.spell.Name, c.targetName(), dmgDesc, c.critTag())),
			Actee: messaging.Say(c.category(), fmt.Sprintf(
				`%s's <ansi fg="cyan">%s</ansi> slams you to the ground! (<ansi fg="damage">%s</ansi>)%s`,
				c.casterName(), c.spell.Name, dmgDesc, c.critTag())),
			Observer: messaging.Say(c.category(), fmt.Sprintf(
				`%s's <ansi fg="cyan">%s</ansi> knocks %s to the ground!`,
				c.casterName(), c.spell.Name, c.targetName())),
		}, c.audience())
		return dmg
	}
	messaging.SendTrio(messaging.Trio{
		Actor: messaging.Say(c.category(), fmt.Sprintf(
			`Your %s strikes %s, but %s is already down. (<ansi fg="damage">%s</ansi>)%s`,
			c.spell.Name, c.targetName(), c.targetName(), dmgDesc, c.critTag())),
		Actee: messaging.Say(c.category(), fmt.Sprintf(
			`%s's <ansi fg="cyan">%s</ansi> strikes you, but you're already down. (<ansi fg="damage">%s</ansi>)%s`,
			c.casterName(), c.spell.Name, dmgDesc, c.critTag())),
		Observer: messaging.NoLine,
	}, c.audience())
	return dmg
}
```

- [ ] **Step 4: Delete the old knockdown arms** in `spell_resolution.go`:
  - Delete `applyMobEffect_knockdown` and `applyMobKnockdownOutcome` with their doc comments (old 679-762).
  - In `applyMobEffectArms`, delete `case "knockdown":` and `return applyMobEffect_knockdown(user, casterChar, mob, room, spellData, magnitude, out, critTag, mName)`.
  - In `applyMobOnPlayerArms`, delete the whole `case "knockdown":` arm (old 1631-1686). Then, because the remaining `condition` and `default` arms use neither, make these edits so it compiles:

| Old | New |
|---|---|
| `spellData, magnitude, out := c.spell, c.magnitude, c.out` | `spellData, out := c.spell, c.out` |
| `isCrit := out.AttackerCrit` | (delete the line) |
| `mobSpellDmg := 0` | (delete the line) |
| `if isCrit {` | `if out.AttackerCrit {` |
| `return mobSpellDmg` | `return 0` |

  - Remove the now-unused imports `mudlog` and `position` from `spell_resolution.go` (F25). `go build` names any other.

- [ ] **Step 5: Run the tests**

```bash
cd C:/tmp/dogmud-3a-harmful && go build ./... && go vet ./internal/hooks/ && go test ./internal/hooks/ -run 'TestSpellKnockdown_|TestApplyMobEffect_Knockdown|TestSpellDot_|TestSpellDamage_|TestSpellEffectCtx' -count=1 -v
```

Expected: all PASS.

- [ ] **Step 6: Run the hooks package.** `go test ./internal/hooks/ -count=1`, expected `ok`.

- [ ] **Step 7: Root guards.** Re-key shifted rows only; `TestNarrationSitesMatchViewpointAudit` should need no change here. Rerun until `ok`.

- [ ] **Step 8: Commit**

```bash
cd C:/tmp/dogmud-3a-harmful && git add internal/hooks/spell_effects.go internal/hooks/spell_resolution.go internal/hooks/spell_knockdown_test.go condition_apply_path_guard_test.go && git commit -F - <<'EOF'
feat(spells): one knockdown applier

applySpellKnockdown replaces the PM/MM and MP knockdown arms. PP gains
the knockdown it never had, and a mob's knockdown on a player now
breaks damage-fragile conditions the way a player's always did. A hit
fully negated by a defensive crit breaks nothing, as PP and MP already
behaved.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 5: Shared resolver steps: backfire, record, interrupt, drain names

**Model:** opus.

**Files:**
- Create: `internal/hooks/spell_resolver_steps_test.go`
- Modify: `internal/hooks/spell_effects.go`, `internal/hooks/spell_resolution.go` (the four resolvers, `maybeInterruptSpellOnMob`, `resolveMobDrainArea`)
- Modify: `condition_apply_path_guard_test.go` (re-key), `messaging_surface_guard_test.go` (drop rows 1286, 1287)

- [ ] **Step 1: Write the failing tests** `internal/hooks/spell_resolver_steps_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spellParityPairing drives one caster-target pairing through its real
// resolver. PM player to mob, PP player to player (room 1 is Pvp), MM mob to
// mob, MP mob to player.
type spellParityPairing struct {
	name      string
	src, tgt  combat.SourceTarget
	caster    func(*spellParityFixture) *characters.Character
	target    func(*spellParityFixture) *characters.Character
	casterRef func(*spellParityFixture) state.ActorRef
	cast      func(*spellParityFixture, *spells.SpellData)
}

func spellParityPairings() []spellParityPairing {
	userCaster := func(f *spellParityFixture) *characters.Character { return f.casterUser.Character }
	mobCaster := func(f *spellParityFixture) *characters.Character { return &f.casterMob.Character }
	userRef := func(f *spellParityFixture) state.ActorRef { return state.ActorRef{UserId: f.casterUser.UserId} }
	mobRef := func(f *spellParityFixture) state.ActorRef { return state.ActorRef{MobInstanceId: f.casterMob.InstanceId} }
	return []spellParityPairing{
		{name: "PM", src: combat.User, tgt: combat.Mob, caster: userCaster, casterRef: userRef,
			target: func(f *spellParityFixture) *characters.Character { return &f.targetMob.Character },
			cast: func(f *spellParityFixture, s *spells.SpellData) {
				resolveAgainstMob(f.casterUser, f.targetMob, f.room, s,
					spellAttackSideFor(s, f.casterUser.Character, nil), s.EffectMagnitude)
			}},
		{name: "PP", src: combat.User, tgt: combat.User, caster: userCaster, casterRef: userRef,
			target: func(f *spellParityFixture) *characters.Character { return f.targetUser.Character },
			cast: func(f *spellParityFixture, s *spells.SpellData) {
				resolveAgainstPlayer(f.casterUser, f.targetUser, f.room, s,
					spellAttackSideFor(s, f.casterUser.Character, nil), s.EffectMagnitude)
			}},
		{name: "MM", src: combat.Mob, tgt: combat.Mob, caster: mobCaster, casterRef: mobRef,
			target: func(f *spellParityFixture) *characters.Character { return &f.targetMob.Character },
			cast: func(f *spellParityFixture, s *spells.SpellData) {
				resolveMobSpellAgainstMob(f.casterMob, f.targetMob, f.room, s,
					spellAttackSideFor(s, &f.casterMob.Character, nil), s.EffectMagnitude)
			}},
		{name: "MP", src: combat.Mob, tgt: combat.User, caster: mobCaster, casterRef: mobRef,
			target: func(f *spellParityFixture) *characters.Character { return f.targetUser.Character },
			cast: func(f *spellParityFixture, s *spells.SpellData) {
				resolveMobSpellAgainstPlayer(f.casterMob, f.targetUser, f.room, s,
					spellAttackSideFor(s, &f.casterMob.Character, nil), s.EffectMagnitude)
			}},
	}
}

// Every caster kind is hurt by its own backfire, the room sees it, and it is
// recorded. PP and MM used to record nothing.
func TestSpellBackfire_EveryCasterIsHurtToldAndRecorded(t *testing.T) {
	for _, p := range spellParityPairings() {
		t.Run(p.name, func(t *testing.T) {
			f := newSpellParityFixture(t, combat.ChannelDefenceResult{AttackerFumble: true})
			spell := physicalHarmSpellForCollapseTest()
			caster := p.caster(f)

			p.cast(f, spell)

			assert.Equal(t, 1000-spell.EffectMagnitude/4, caster.Health, "a backfire wounds its caster")
			assert.Equal(t, 1, countContaining(drainPlain(3), "spell backfires!"), "the room sees every backfire")
			require.Len(t, f.records, 1, "every backfire is recorded")
			assert.True(t, f.records[0].backfire)
			assert.Equal(t, p.src, f.records[0].src)
			assert.Equal(t, p.tgt, f.records[0].tgt)
		})
	}
}

// Every landed cast is recorded once, with the damage it dealt. PP and MM
// used to record nothing.
func TestSpellRecord_EveryPairingRecordsALandedCast(t *testing.T) {
	for _, p := range spellParityPairings() {
		t.Run(p.name, func(t *testing.T) {
			f := newSpellParityFixture(t, spellContestAttackWin())
			spell := physicalHarmSpellForCollapseTest()
			target := p.target(f)

			p.cast(f, spell)

			require.Len(t, f.records, 1)
			assert.True(t, f.records[0].hit)
			assert.False(t, f.records[0].backfire)
			assert.Equal(t, p.src, f.records[0].src)
			assert.Equal(t, p.tgt, f.records[0].tgt)
			assert.Equal(t, 1000-target.Health, f.records[0].dmg)
		})
	}
}

// A configured boss-interrupt spell cancels any casting target's cast. Only
// a player's spell on a mob used to.
func TestSpellInterrupt_ACastingPlayerIsInterrupted(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	f.targetUser.Character.Activity = activity.NewMachine()
	require.NoError(t, f.targetUser.Character.Activity.TransitionToCasting(
		activity.CastingData{SpellId: "fold-anchor"},
		state.TransitionReason{Trigger: activity.TriggerCastBegin}))
	require.True(t, f.targetUser.Character.IsCasting())
	spell := physicalHarmSpellForCollapseTest()
	spell.SpellId = "neural-stun" // a default boss-interrupt id, config.balance.misc.go:334

	resolveAgainstPlayer(f.casterUser, f.targetUser, f.room, spell,
		spellAttackSideFor(spell, f.casterUser.Character, nil), spell.EffectMagnitude)

	assert.False(t, f.targetUser.Character.IsCasting(), "the target's cast must be cancelled")
	assert.Equal(t, 1, countContaining(drainPlain(2), "your spell collapses!"))
}

// The drain's personal lines were raw SendText with the mob's name baked in,
// so a drained player in the dark read the name they could not see.
func TestMobDrainArea_DrainedPlayerInTheDarkReadsSomething(t *testing.T) {
	f := newSpellParityFixture(t, spellContestAttackWin())
	t.Cleanup(combat.SetChannelAttackContestRunnerForTest(attackWinContest(t)))
	f.room.Lamp = nil
	darken(t, 1)
	spell := &spells.SpellData{SpellId: "test-core-recharge", Name: "Core Recharge", EffectType: "drain_area",
		AttackType: combatvocab.AttackSpell, DamageType: combatvocab.DamagePhysical, Targeting: combatvocab.TargetArea}

	resolveMobDrainArea(f.casterMob, f.room, spell)

	target := drainPlain(2)
	assert.Equal(t, 1, countContaining(target, "Something's Core Recharge saps your strength!"))
	assert.Equal(t, 0, countContaining(target, "Skeleton"))
}
```

- [ ] **Step 2: Run them to verify they fail**

```bash
cd C:/tmp/dogmud-3a-harmful && go test ./internal/hooks/ -run 'TestSpellBackfire_|TestSpellRecord_|TestSpellInterrupt_|TestMobDrainArea_' -count=1 -v
```

Expected FAILs: backfire PP and MM (`f.records` length 0); record PP and MM (length 0); interrupt (still casting); drain (0 "Something's", 1 "Skeleton"). PM and MP subtests of the first two already pass.

- [ ] **Step 3: Add the shared steps** to `spell_effects.go`. Add imports `"github.com/GoMudEngine/GoMud/internal/configs"` and `"github.com/GoMudEngine/GoMud/internal/util"`. Then:

```go
// recordSpellResolution records one resolved (not backfired) cast for every
// pairing. A defended cast records in the old fizzle column but keeps its
// partial damage (Stage 30.1).
func recordSpellResolution(c spellEffectCtx, dmg int) {
	recordSpell(spellSourceTarget(c.caster), spellSourceTarget(c.target),
		!c.out.Defended, c.out.AttackerCrit, false, c.out.Defended, dmg,
		c.out.AttackRollZScore, c.casterChar, c.targetChar(), util.GetRoundCount())
}

// applySpellBackfire resolves a fumbled cast for every caster kind: the
// caster takes a quarter of the magnitude (at least one), is told if it is a
// player, the room sees it, and it is recorded.
func applySpellBackfire(c spellEffectCtx) {
	backfireDmg := c.magnitude / 4
	if backfireDmg < 1 {
		backfireDmg = 1
	}
	if c.casterChar != nil {
		c.casterChar.ApplyHarm(characters.PoolHealth, backfireDmg, c.casterRef())
	}
	name := c.casterName()
	messaging.SendTrio(messaging.Trio{
		Actor: messaging.Say(messaging.CategorySpellDisruption,
			`<ansi fg="red">Your spell backfires violently, wounding you!</ansi>`),
		Actee: messaging.NoLine,
		Observer: messaging.Say(messaging.CategorySpellDisruption, fmt.Sprintf(
			`<ansi fg="red">%s's spell backfires!</ansi>`, name)),
	}, spellAudience(c.casterUser(), name, nil, messaging.NoName, c.room))
	recordSpell(spellSourceTarget(c.caster), spellSourceTarget(c.target), false, false, true, false, 0,
		c.out.AttackRollZScore, c.casterChar, c.targetChar(), util.GetRoundCount())
}

// maybeInterruptSpellOnTarget cancels any character's in-progress cast when
// spellId is a configured boss-interrupt disruption spell
// (Balance.BossInterruptSpellIds) and the character is casting. It reuses
// actions.InterruptTargetCast (conviction refund, cast cancel, and the
// CastInterrupted event for a player). Returns whether a cast was cancelled.
func maybeInterruptSpellOnTarget(target *characters.Character, spellId string, by state.ActorRef) bool {
	if target == nil {
		return false
	}
	if !configs.GetBalanceConfig().IsBossInterruptSpell(spellId) {
		return false
	}
	if !target.IsCasting() {
		return false
	}
	return actions.InterruptTargetCast(target, by)
}

// interruptSpellTarget runs the boss-interrupt for every pairing, after the
// backfire check (a botched cast cannot interrupt) and whether or not the
// target defends the damage: the interrupt is the point of the spell.
func interruptSpellTarget(c spellEffectCtx) {
	if !maybeInterruptSpellOnTarget(c.targetChar(), c.spell.SpellId, c.casterRef()) {
		return
	}
	messaging.SendTrio(messaging.Trio{
		Actor: messaging.Say(messaging.CategorySpellDisruption, fmt.Sprintf(
			`<ansi fg="cyan-bold">Your %s scrambles %s's focus -- the spell collapses!</ansi>`,
			c.spell.Name, c.targetName())),
		Actee: messaging.Say(messaging.CategorySpellDisruption, fmt.Sprintf(
			`<ansi fg="cyan-bold">%s's %s scrambles your focus -- your spell collapses!</ansi>`,
			c.casterName(), c.spell.Name)),
		Observer: messaging.Say(messaging.CategorySpellDisruption, fmt.Sprintf(
			`<ansi fg="cyan">%s's spell collapses!</ansi>`, c.targetName())),
	}, c.audience())
}
```

- [ ] **Step 4: Replace `maybeInterruptSpellOnMob`'s body** in `spell_resolution.go` (old 478-489; keep its doc comment, add "Wrapper over maybeInterruptSpellOnTarget; Task 8 deletes it."):

```go
func maybeInterruptSpellOnMob(mob *mobs.Mob, spellId string, by state.ActorRef) bool {
	if mob == nil {
		return false
	}
	return maybeInterruptSpellOnTarget(&mob.Character, spellId, by)
}
```

- [ ] **Step 5: Rewrite the four resolvers' contested paths.** Keep each doc comment. Where this step says "keep", the existing lines stay exactly as they are.

`resolveAgainstMob` (old 390-469) becomes:

```go
func resolveAgainstMob(user *users.UserRecord, mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, side combat.AttackSide, magnitude int) (fumbled bool, landed bool) {
	caster := actions.NewUserActorInRoom(user, room)
	target := actions.NewMobActorInRoom(mob, room)

	// <keep the "Non-harm cast at a mob" comment, old 392-399>
	if spellData.AttackType == combatvocab.AttackNone {
		// <keep the "Every reachable non-harm arm" comment, old 401-404>
		c := newSpellEffectCtx(user.Character, caster, target, room, spellData, magnitude,
			combat.ChannelDefenceResult{DamageMultiplier: 1})
		recordSpellResolution(c, applySpellEffect(c))
		return false, true
	}

	// Task 17: the sleeping-victim forced crit reaches the spell channel.
	side.ForceCrit = combat.SleepingForceCrit(&mob.Character)

	// <keep the charm block, old 413-425, unchanged>
	out := runSpellChannelAttack(combat.SightRoom(room), spellData.Attack(), side, user.Character, &mob.Character)
	c := newSpellEffectCtx(user.Character, caster, target, room, spellData, magnitude, out)

	// Backfire on fumble, resolved BEFORE success per the seam's contract: a
	// fumbled cast aborts even a winning roll.
	if out.AttackerFumble {
		applySpellBackfire(c)
		return true, false
	}

	// Boss-interrupt, for every pairing (interruptSpellTarget).
	interruptSpellTarget(c)

	recordSpellResolution(c, applySpellEffect(c))

	// U6b Task 10: the MOB defender's crit defence counters the player caster.
	fireSpellCounterTier(room, out, spellData.Attack(),
		&mob.Character, user.Character, nil, user)

	return false, !out.Defended
}
```

`resolveAgainstPlayer` (old 929-965) body becomes:

```go
	// Task 17: the sleeping-victim forced crit reaches the spell channel.
	side.ForceCrit = combat.SleepingForceCrit(target.Character)
	out := runSpellChannelAttack(combat.SightRoom(room), spellData.Attack(), side, user.Character, target.Character)
	c := newSpellEffectCtx(user.Character, actions.NewUserActorInRoom(user, room),
		actions.NewUserActorInRoom(target, room), room, spellData, magnitude, out)

	// Backfire on fumble, resolved BEFORE success per the seam's contract.
	if out.AttackerFumble {
		applySpellBackfire(c)
		return true, false
	}

	interruptSpellTarget(c)

	recordSpellResolution(c, applySpellEffect(c))

	// <keep the "Set reciprocal aggro for harm spells" block, old 950-958:
	// the harmful appliers commit their own; this still serves harmful
	// condition spells until slice 3b>

	// U6b Task 10: the defending player's crit defence counters the caster.
	fireSpellCounterTier(room, out, spellData.Attack(),
		target.Character, user.Character, target, user)

	return false, !out.Defended
```

`resolveMobSpellAgainstMob` (old 1482-1517) body becomes:

```go
	casterActor := actions.NewMobActorInRoom(caster, room)
	targetActor := actions.NewMobActorInRoom(target, room)
	// <keep the "Non-harm effects" comment, old 1484-1492>
	if spellData.AttackType == combatvocab.AttackNone {
		applySpellEffect(newSpellEffectCtx(&caster.Character, casterActor, targetActor, room, spellData,
			magnitude, combat.ChannelDefenceResult{DamageMultiplier: 1}))
		// Uncontested cooperative cast: no defence to beat, so it landed.
		return true
	}
	// Task 17: the sleeping-victim forced crit reaches the spell channel.
	side.ForceCrit = combat.SleepingForceCrit(&target.Character)
	out := runSpellChannelAttack(combat.SightRoom(room), spellData.Attack(), side, &caster.Character, &target.Character)
	c := newSpellEffectCtx(&caster.Character, casterActor, targetActor, room, spellData, magnitude, out)
	if out.AttackerFumble {
		applySpellBackfire(c)
		return false
	}
	interruptSpellTarget(c)
	recordSpellResolution(c, applySpellEffect(c))

	// U6b Task 10: the defending mob's crit defence counters the mob caster.
	fireSpellCounterTier(room, out, spellData.Attack(),
		&target.Character, &caster.Character, nil, nil)

	return !out.Defended
```

`resolveMobSpellAgainstPlayer` body becomes:

```go
	// Task 17: the sleeping-victim forced crit reaches the spell channel.
	side.ForceCrit = combat.SleepingForceCrit(target.Character)
	out := runSpellChannelAttack(combat.SightRoom(room), spellData.Attack(), side, &caster.Character, target.Character)
	c := newSpellEffectCtx(&caster.Character, actions.NewMobActorInRoom(caster, room),
		actions.NewUserActorInRoom(target, room), room, spellData, magnitude, out)
	if out.AttackerFumble {
		applySpellBackfire(c)
		return false
	}
	interruptSpellTarget(c)
	recordSpellResolution(c, applySpellEffect(c))

	// U6b Task 10: the PLAYER defender's crit defence counters the mob caster.
	fireSpellCounterTier(room, out, spellData.Attack(),
		target.Character, &caster.Character, target, nil)

	return !out.Defended
```

- [ ] **Step 6: Drain narration onto the shared names.** In `resolveMobDrainArea`:
  - In the "finding no one to drain" line (old 1368-1370) and the "tears the life" line (old 1434-1436), replace `<ansi fg="mobname">%s</ansi>` with `%s` and the argument `mob.Character.Name` with `mobDisplayName(mob, room, 0)`.
  - Inside the `for _, pr := range result.PlayerResults` loop, after the `target == nil` check, add:

```go
		c := newSpellEffectCtx(&mob.Character, actions.NewMobActorInRoom(mob, room),
			actions.NewUserActorInRoom(target, room), room, spellData, 0, pr.MoveResult.Defence)
```

  - Replace the hit line (old 1416-1419) with:

```go
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.NoLine,
				Actee: messaging.Say(c.category(), fmt.Sprintf(
					`%s's <ansi fg="cyan">%s</ansi> saps your strength! (<ansi fg="damage">%s</ansi>)`,
					c.casterName(), spellData.Name,
					combat.GetDamageDescription(pr.MoveResult.Damage, target.Character.HealthMax.Value))),
				Observer: messaging.NoLine,
			}, c.audience())
```

  - Replace the partial line (old 1427-1430) with:

```go
			messaging.SendTrio(messaging.Trio{
				Actor: messaging.NoLine,
				Actee: messaging.Say(c.category(), fmt.Sprintf(
					`%s's <ansi fg="cyan">%s</ansi> fails to take full hold of you, but still saps a little of your strength! (<ansi fg="damage">%s</ansi>)`,
					c.casterName(), spellData.Name,
					combat.GetDamageDescription(pr.MoveResult.Damage, target.Character.HealthMax.Value))),
				Observer: messaging.NoLine,
			}, c.audience())
```

  The defended-with-zero branch (`sendSpellChannelDefenceMessages`) and the `targeting.Commit` line stay.

- [ ] **Step 7: Build.** `go build ./... && go vet ./internal/hooks/`. Remove the `util` import from `spell_resolution.go` if the compiler reports it unused (F25: its three uses are gone). Remove any `round := util.GetRoundCount()` line left behind.

- [ ] **Step 8: Run the tests**

```bash
cd C:/tmp/dogmud-3a-harmful && go test ./internal/hooks/ -run 'TestSpellBackfire_|TestSpellRecord_|TestSpellInterrupt_|TestMobDrainArea_|TestMaybeInterruptSpellOnMob|TestSpell' -count=1 -v
```

Expected: all PASS.

- [ ] **Step 9: Run the hooks package.** `go test ./internal/hooks/ -count=1`, expected `ok`. The counter-tier tests (`counter_tier_test.go`) must still count exactly two contests per crit-defended cast; a third means an applier rolled a contest.

- [ ] **Step 10: Root guards.** Delete these two rows from `narrationViewpointRegistry` (their events now go through `SendTrio`):
  - `"hooks/spell_resolution.go|<ansi fg=\"cyan-bold\">Your %s scrambles %s's focus -- its spell collapses!</ansi>"` (old 1286)
  - `"hooks/spell_resolution.go|<ansi fg=\"red\">Your spell backfires violently, wounding you!</ansi>"` (old 1287)

  Re-key the condition rows. Rerun until `ok`.

- [ ] **Step 11: Commit**

```bash
cd C:/tmp/dogmud-3a-harmful && git add internal/hooks/spell_effects.go internal/hooks/spell_resolution.go internal/hooks/spell_resolver_steps_test.go condition_apply_path_guard_test.go messaging_surface_guard_test.go && git commit -F - <<'EOF'
feat(spells): every resolver shares backfire, record and interrupt

applySpellBackfire hurts, tells and records every caster kind;
recordSpellResolution records PP and MM casts that went unrecorded;
interruptSpellTarget lets a boss-interrupt spell cancel any casting
target, player or mob, from any caster. The four resolvers keep their
names and their one contest. resolveMobDrainArea's personal lines move
onto SendTrio, so a drained player in the dark no longer reads the
caster's name.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 6: The parity table

**Model:** opus.

**Files:**
- Create: `internal/hooks/spell_effect_parity_test.go`

- [ ] **Step 1: Write the table** `internal/hooks/spell_effect_parity_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parityHarmAmount reads what a harmful effect did to its target: health
// lost for damage and knockdown, the dot record's trigger count for a dot
// (-1 when the record is missing).
func parityHarmAmount(spell *spells.SpellData, target *characters.Character, healthBefore int) int {
	if spell.EffectType == "dot" {
		recs := target.GetConditions(conditions.ConditionIdPoisoned)
		if len(recs) != 1 {
			return -1
		}
		return recs[0].TriggersLeft
	}
	return healthBefore - target.Health
}

// parityWantAmount is the amount every pairing must land. Damage runs the
// shared formula on this pairing's own equalised combatants. The dot's
// duration is written from the fixture's numbers (skill 3, willpower 100),
// not from the production helper, so a pairing that reads the wrong caster
// cannot agree with it by construction.
func parityWantAmount(spell *spells.SpellData, caster, target *characters.Character) int {
	if spell.EffectType == "dot" {
		return calcSpellDuration(spell.BaseFolds, 3, 100) / 3
	}
	return scaleSpellDamageByDefence(
		calcSpellDamageForCharacter(spell, caster, target, spell.EffectMagnitude, false), spellContestAttackWin())
}

// Parity slice 3a: each harmful effect, driven through all four pairings
// with the same caster stats and spell, lands the same amount, turns the
// target on its caster, is recorded once, is seen by a watcher, and (PM
// only) counts as aggression against the mob.
func TestSpellHarmParity_EveryPairingLandsTheSame(t *testing.T) {
	effects := []struct {
		name     string
		spell    func() *spells.SpellData
		roomWord string
	}{
		{"damage", physicalHarmSpellForCollapseTest, "strikes"},
		{"dot", dotSpellForParityTest, "afflicts"},
		{"knockdown", knockdownSpellForParityTest, "to the ground"},
	}
	for _, eff := range effects {
		t.Run(eff.name, func(t *testing.T) {
			amounts := map[string]int{}
			for _, p := range spellParityPairings() {
				t.Run(p.name, func(t *testing.T) {
					f := newSpellParityFixture(t, spellContestAttackWin())
					spell := eff.spell()
					casterChar, targetChar := p.caster(f), p.target(f)
					want := parityWantAmount(spell, casterChar, targetChar)
					before := targetChar.Health

					p.cast(f, spell)

					got := parityHarmAmount(spell, targetChar, before)
					assert.Equal(t, want, got, "the amount must match the shared formula")
					amounts[p.name] = got
					if eff.name == "knockdown" {
						assert.True(t, targetChar.IsSupine() || targetChar.IsProne(), "knocked down")
					}
					assert.Equal(t, p.casterRef(f), targetChar.CurrentCombatTarget(),
						"the target must turn on its caster")
					require.Len(t, f.records, 1, "one record per resolved cast")
					assert.Equal(t, p.src, f.records[0].src)
					assert.Equal(t, p.tgt, f.records[0].tgt)
					assert.True(t, f.records[0].hit)
					if eff.name != "dot" {
						assert.Equal(t, got, f.records[0].dmg)
					}
					assert.Equal(t, 1, countContaining(drainPlain(3), eff.roomWord),
						"a watcher sees the spell land")
					attacked := events.DrainQueuedPlayerAttackedMobsForTest(0)
					if p.name == "PM" {
						require.Len(t, attacked, 1, "a player's harm on a mob is aggression")
						assert.Equal(t, 101, attacked[0].MobInstanceId)
					} else {
						assert.Empty(t, attacked, "only a player caster on a mob seeds aggression")
					}
				})
			}
			require.Len(t, amounts, 4)
			for name, amount := range amounts {
				assert.Equal(t, amounts["PM"], amount, "%s must land what PM lands", name)
			}
		})
	}
}
```

- [ ] **Step 2: Run it**

```bash
cd C:/tmp/dogmud-3a-harmful && go test ./internal/hooks/ -run 'TestSpellHarmParity_' -count=1 -v
```

Expected: PASS for all twelve subtests. If only the cross-pairing equality fails while every per-pairing "shared formula" assertion passes, the fixture is not equal (a stat or mitigation input differs between a user and a mob): equalise the fixture in `equaliseSpellCombatant`, never production code, and record which input it was in the commit body.

- [ ] **Step 3: Prove the table can fail.** Temporarily comment out the line `commitHarmfulSpellAggro(c, fresh)` that sits after `afflicted := ...` in `applySpellDot` (Edit tool). Run Step 2's command. Expected: `dot/PM`, `dot/PP`, `dot/MM`, `dot/MP` FAIL on "the target must turn on its caster", and `dot/PM` also on "a player's harm on a mob is aggression". Restore the line with the Edit tool, rerun, expect PASS, and confirm `git diff --stat` lists only the new test file.

- [ ] **Step 4: Commit**

```bash
cd C:/tmp/dogmud-3a-harmful && git add internal/hooks/spell_effect_parity_test.go && git commit -F - <<'EOF'
test(spells): harmful effect parity table across PM, PP, MM and MP

Damage, dot and knockdown driven through all four pairings with the
same caster stats land the same amount, turn the target on its caster,
record once, reach a watcher, and (PM only) seed aggression. Proven
capable of failing by removing the dot applier's aggro commit.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 7: Guard audit

**Model:** sonnet.

**Files:**
- Modify: `internal/hooks/channel_defence_routing_test.go:57-99`
- Modify: `condition_apply_path_guard_test.go` (comments at the ward, regen and dot blocks)

- [ ] **Step 1: Widen the one-contest guard to the new file.** The appliers moved out of the only file `TestSpellResolversRunOneContestAndAppliersRollNone` parses (F18), so today an applier in `spell_effects.go` could roll its own contest unseen. In `channel_defence_routing_test.go`, replace the two lines

```go
	parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(filepath.Dir(here), "spell_resolution.go"), nil, 0)
	require.NoError(t, err)
```

and the `seamCallsByFunc := map[string]int{}` / `directCalls := 0` declarations plus the `for _, decl := range parsed.Decls { ... }` loop, with:

```go
	seamCallsByFunc := map[string]int{}
	directCalls := 0
	// spell_effects.go holds every applier since parity slice 3a; it must run
	// no contest of its own either.
	for _, name := range []string{"spell_resolution.go", "spell_effects.go"} {
		parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(filepath.Dir(here), name), nil, 0)
		require.NoError(t, err)
		for _, decl := range parsed.Decls {
			// <the existing loop body, unchanged>
		}
	}
```

The `require.Equal` map of the four resolvers and the `require.Zero(t, directCalls, ...)` stay as they are.

- [ ] **Step 2: Prove it can fail.** Temporarily add `_ = runSpellChannelAttack(nil, combatvocab.Attack{}, combat.AttackSide{}, nil, nil)` as the first line of `applySpellDamage` (add the `combatvocab` import if needed so it compiles, check with `go vet ./internal/hooks/`). Run `go test ./internal/hooks/ -run TestSpellResolversRunOneContestAndAppliersRollNone -count=1`. Expected FAIL naming `applySpellDamage: 1`. Remove the line (and any import you added) with the Edit tool; rerun; expect `ok`.

- [ ] **Step 3: Record the moves in the allowlist comments** of `condition_apply_path_guard_test.go` (comment text only; keys were re-keyed in Tasks 1-5):
  - Ward block: replace `and the mutations import were deleted)` with `and the mutations import were deleted; re-keyed again parity slice 3a\n\t// when the harmful arms moved to spell_effects.go)`.
  - Regen block: replace `re-keyed again parity slice 2, same deletion as above)` with `re-keyed again parity slice 2, same deletion as above; re-keyed again\n\t// parity slice 3a, same move as above)`.
  - Dot block: replace `re-keyed again parity slice 2, same deletion as\n\t// above)` with `re-keyed again parity slice 2, same deletion as\n\t// above; COLLAPSED to one row by parity slice 3a, when the three dot\n\t// arms became applySpellDot in spell_effects.go)`.

  (`\n\t` means a real newline and tab; the trailing box-drawing rule on each line stays where it is.)

- [ ] **Step 4: Confirm the other guards need nothing.** Run the root guards and `go test ./internal/hooks/ -count=1`. `TestEveryRollSiteAppliesTheSightPenalty` must pass with the four `sightExemptSites` rows untouched (the resolvers kept their names; `spell_effects.go` rolls nothing). Expected: all `ok`.

- [ ] **Step 5: Commit**

```bash
cd C:/tmp/dogmud-3a-harmful && git add internal/hooks/channel_defence_routing_test.go condition_apply_path_guard_test.go && git commit -F - <<'EOF'
test(guards): the one-contest guard reads spell_effects.go too

The appliers left spell_resolution.go, the only file the guard parsed,
so an applier could have rolled its own contest unseen. Proven red by a
seam call planted in applySpellDamage. The condition-path allowlist
comments record the slice 3a re-keys and the dot collapse.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 8: Retire the wrappers no harmful test needs

**Model:** sonnet.

**Files:**
- Modify: `internal/hooks/spell_interrupt_test.go` (54, 69, 82, 91, 105)
- Modify: `internal/hooks/hooks_test.go` (1047, 2556, 2569, 2589, 2601, 2628)
- Modify: `internal/hooks/spell_resolution.go` (delete `maybeInterruptSpellOnMob`)

`applyMobEffect` and `applyPlayerEffect` STAY: condition, heal, shield, purge and default tests (slice 3b's scope, e.g. `wire_freeze_test.go`, `selfcast_wording_test.go`) still name them. Slice 3b's last task deletes them.

- [ ] **Step 1: Migrate `spell_interrupt_test.go`.** Replace each `maybeInterruptSpellOnMob(mob, X, by)` with `maybeInterruptSpellOnTarget(&mob.Character, X, by)` (lines 54, 69, 82, 105) and `maybeInterruptSpellOnMob(nil, "neural-stun", by)` with `maybeInterruptSpellOnTarget(nil, "neural-stun", by)` (91). Rename the five test functions from `TestMaybeInterruptSpellOnMob_` to `TestMaybeInterruptSpellOnTarget_` and the two comment mentions of `maybeInterruptSpellOnMob` (27, and the doc line of each test) to `maybeInterruptSpellOnTarget`.

- [ ] **Step 2: Delete `maybeInterruptSpellOnMob`** and its doc comment from `spell_resolution.go`.

- [ ] **Step 3: Migrate the six harmful `applyMobEffect` calls in `hooks_test.go`** onto the dispatcher (add `"github.com/GoMudEngine/GoMud/internal/actions"` to its imports if absent):

| Line | Old | New |
|---|---|---|
| 1047 | `dmg := applyMobEffect(u, u.Character, mob, room, dotSpell, 10, spellContestAttackWin())` | `dmg := applySpellEffect(newSpellEffectCtx(u.Character, actions.NewUserActorInRoom(u, room), actions.NewMobActorInRoom(mob, room), room, dotSpell, 10, spellContestAttackWin()))` |
| 2556 | `dmg := applyMobEffect(u, u.Character, mob, room, spellData, 30, spellContestAttackWin())` | `dmg := applySpellEffect(newSpellEffectCtx(u.Character, actions.NewUserActorInRoom(u, room), actions.NewMobActorInRoom(mob, room), room, spellData, 30, spellContestAttackWin()))` |
| 2569 | `dmg := applyMobEffect(u, u.Character, mob, room, spellData, 30, spellContestAttackCrit())` | `dmg := applySpellEffect(newSpellEffectCtx(u.Character, actions.NewUserActorInRoom(u, room), actions.NewMobActorInRoom(mob, room), room, spellData, 30, spellContestAttackCrit()))` |
| 2589 | `dmg := applyMobEffect(u, u.Character, mob, room, dotSpell, 10, spellContestAttackWin())` | `dmg := applySpellEffect(newSpellEffectCtx(u.Character, actions.NewUserActorInRoom(u, room), actions.NewMobActorInRoom(mob, room), room, dotSpell, 10, spellContestAttackWin()))` |
| 2601 | `dmg := applyMobEffect(nil, nil, mob, room, spellData, 30, spellContestAttackWin())` | `dmg := applySpellEffect(newSpellEffectCtx(nil, nil, actions.NewMobActorInRoom(mob, room), room, spellData, 30, spellContestAttackWin()))` |
| 2628 | `dmg := applyMobEffect(u, u.Character, mob, room, kdSpell, 20, spellContestAttackWin())` | `dmg := applySpellEffect(newSpellEffectCtx(u.Character, actions.NewUserActorInRoom(u, room), actions.NewMobActorInRoom(mob, room), room, kdSpell, 20, spellContestAttackWin()))` |

Rename the five `TestApplyMobEffect_Damage`, `_DamageWithCrit`, `_DotEffect`, `_NilUser`, `_Knockdown` to `TestApplySpellEffect_Damage`, `_DamageWithCrit`, `_DotEffect`, `_NilCaster`, `_Knockdown`, and in the doc comment of `TestDotProducerRecordsNegativeHarm_MobTarget` replace "spell_resolution.go's applyMobEffect_dot" with "spell_effects.go's applySpellDot". Line numbers are pre-plan; find each by its text.

- [ ] **Step 4: Confirm no stale names remain**

```bash
cd C:/tmp/dogmud-3a-harmful && grep -rn "maybeInterruptSpellOnMob\|applyMobEffect_damage\|applyMobEffect_dot\|applyMobEffect_knockdown\|applyMobKnockdownOutcome" --include=*.go .
```

Expected: no output (grep exits 1; run it standalone, not in an `&&` chain). Prove the grep can match: `grep -rn "applyMobEffect_heal" --include=*.go internal/hooks | head -1` must print a line.

- [ ] **Step 5: Run** `go build ./... && go test ./internal/hooks/ -count=1` and the root guards. Expected: all `ok` (re-key the condition rows if Step 2 shifted them).

- [ ] **Step 6: Commit**

```bash
cd C:/tmp/dogmud-3a-harmful && git add internal/hooks/spell_interrupt_test.go internal/hooks/hooks_test.go internal/hooks/spell_resolution.go condition_apply_path_guard_test.go && git commit -F - <<'EOF'
refactor(spells): retire maybeInterruptSpellOnMob; harmful tests call the dispatcher

The interrupt tests call maybeInterruptSpellOnTarget, and the six
harmful applyMobEffect test calls build a spellEffectCtx directly.
applyMobEffect and applyPlayerEffect stay until slice 3b moves the
helpful-effect tests that still name them.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

---

### Task 9: Docs, gate, boot check, playtest, PR

**Model:** sonnet (the playtest per `dogmud-playtesting`).

**Files:**
- Modify: `internal/hooks/context.md`, `docs/PATCH_NOTES.md`, `docs/README.md`
- Create: `tools/playtest/goals/2026-09-28-spell-3a-town-crime.yaml`, `tools/playtest/goals/2026-09-28-spell-3a-mob-caster.yaml`

- [ ] **Step 1: `internal/hooks/context.md`.** Insert this section immediately before `## Counter tier wiring (U6b Task 10)`:

```markdown
## Spell effects (`spell_effects.go`, parity slice 3a)

Every spell effect on one target goes through one `spellEffectCtx` and one
dispatcher, `applySpellEffect`. The four contested resolvers in
`spell_resolution.go` (`resolveAgainstMob`, `resolveAgainstPlayer`,
`resolveMobSpellAgainstMob`, `resolveMobSpellAgainstPlayer`) keep their names
and their one `runSpellChannelAttack` call each, then build a context per
target: the caster's `*characters.Character`, caster and target as
`actions.Actor` (`*actions.UserActor` or `*actions.MobActor`; only tests pass
a nil caster), the room, the spell, the magnitude and the contest result.
Refs come from the actor (`casterRef`, `targetRef`), not the character.

The harmful effects have one applier each, whoever casts and whoever is hit:
`applySpellDamage`, `applySpellDot`, `applySpellKnockdown`. Each narrates
through `messaging.SendTrio` with the context's audience (players in the
username tag, mobs through `mobDisplayName`), so a mob-on-mob spell reaches
the room and a reader in the dark reads "something". Each starts the fight
through `commitHarmfulSpellAggro`: the target turns on the caster if it was
not already fighting, the caster on the target likewise, and a player caster
on a mob calls `actions.SeedAggression` with freshness judged per target, as
`throw` does, which records the assault crime on a fresh engagement (owner
ruling, 2026-09-28). A dot's duration reads the spell's primarystat and the
school's cast skill through `spellCasterStatAndSkill`, not
`actions.GetSpellStatAndSkill`, which is the fold stat.

The resolvers share three steps: `applySpellBackfire` (every caster kind is
hurt, told, seen and recorded), `interruptSpellTarget` (a configured
boss-interrupt spell cancels any casting target, player or mob, through
`maybeInterruptSpellOnTarget`) and `recordSpellResolution`. `recordSpell` is
the analytics seam over `combat.RecordSpell`, swapped by tests the way
`runSpellChannelAttack` is. `channel_defence_routing_test.go` parses both
files and allows the contest seam only in the four resolvers.

Every other effect (condition, heal, shield, purge, charm, default) still runs
on the per-pairing arms `applyMobEffectArms`, `applyPlayerEffectArms` and
`applyMobOnPlayerArms` until slice 3b; `applyMobEffect` and
`applyPlayerEffect` are test-only wrappers until then. `resolveMobDrainArea`
keeps its own `ExecuteSkillMove` contest; only its lines use the context.
```

  Also, in the "Names in the dark" paragraph near line 124, replace "used by `applyPlayerEffect`," with "used by the appliers in `spell_effects.go`, `applyPlayerEffectArms`,".

  Verify every symbol named exists, then run the audit:

```bash
cd C:/tmp/dogmud-3a-harmful && for s in spellEffectCtx applySpellEffect applySpellDamage applySpellDot applySpellKnockdown commitHarmfulSpellAggro spellCasterStatAndSkill applySpellBackfire interruptSpellTarget maybeInterruptSpellOnTarget recordSpellResolution recordSpell applyMobEffectArms applyPlayerEffectArms applyMobOnPlayerArms applyMobEffect applyPlayerEffect; do printf "%s %s\n" $s "$(cat internal/hooks/spell_effects.go internal/hooks/spell_resolution.go | grep -cE "^(func|var|type) $s\b")"; done
python tools/context_md_audit.py
```

  Expected: every symbol prints 1 (a 0 means the doc names something that does not exist; fix the doc); the audit reports nothing for `internal/hooks`.

- [ ] **Step 2: `docs/PATCH_NOTES.md`.** Add at the top, below `# DOGMud Patch Notes`, dated the day the PR opens (80 columns, no numbers, no dashes):

```markdown
## <YYYY-MM-DD>: Spells are attacks, whoever casts them

Casting a harmful spell at someone now counts as attacking them, exactly
as swinging a sword does. Burn, poison or knock down a townsperson in
front of witnesses and it is a crime: the city remembers who did it, and
its guards treat you as they would any other brawler.

Creatures now cast harmful spells by the same rules you do. A creature's
poison lasts as long as its own skill and focus allow, its spells break
the fragile effects damage always breaks, and when one creature casts at
another, everyone in the room sees it. Where fighting other players is
allowed, poisoning and knockdown spells now work on them too, and a
spell that shatters a creature's concentration can shatter a player's.
```

- [ ] **Step 3: `docs/README.md`.** If the planning commit did not add it, add this row directly below the `2026-09-28-spell-effect-unification-design.md` row:

```markdown
| [`superpowers/plans/2026-09-28-spell-effects-3a-harmful.md`](superpowers/plans/2026-09-28-spell-effects-3a-harmful.md) | Implementation plan for parity slice 3a in nine tasks: the context and dispatcher with the old switches kept as arms; one damage applier with aggro and the assault crime through `actions.SeedAggression`; one dot applier reading the caster's primarystat (correcting the spec's `GetSpellStatAndSkill`, which is the fold stat); one knockdown applier; backfire, record and interrupt shared by all four resolvers and the drain's lines on the context; the parity table; the guard audit; wrapper retirement; docs, boot check and playtest |
```

- [ ] **Step 4: Commit docs**

```bash
cd C:/tmp/dogmud-3a-harmful && git add internal/hooks/context.md docs/PATCH_NOTES.md docs/README.md && git commit -F - <<'EOF'
docs(spells): slice 3a context, patch notes

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

- [ ] **Step 5: Full gate** (Bash, from the worktree root; each standalone):

```bash
cd C:/tmp/dogmud-3a-harmful && gofmt -l internal/ modules/ *.go
```
Expected: no output. If it lists a file this plan never touched, check `git diff master -- <file>` before acting (Windows CRLF copies false-positive).

```bash
cd C:/tmp/dogmud-3a-harmful && go vet ./... && go build ./...
```
Expected: no output.

```bash
cd C:/tmp/dogmud-3a-harmful && go test ./... -count=1 > C:/tmp/dogmud-3a-test.log 2>&1; echo exit=$?
```
Run in the background (about ten minutes). Expected `exit=0`; `grep -E "^(FAIL|---  FAIL|panic)" C:/tmp/dogmud-3a-test.log` prints nothing.

```bash
cd C:/tmp/dogmud-3a-harmful && ~/go/bin/golangci-lint run --new-from-merge-base=origin/master
```
Expected: `0 issues.`

- [ ] **Step 6: Boot check** per `dogmud-shipping`, isolated ports, killed by PID. Bash:

```bash
cd C:/tmp/dogmud-3a-harmful && git worktree add --detach C:/tmp/dogmud-boot-check HEAD && cp "C:/Users/Calabe Davis/workspace/DOGMud/_datafiles/config.yaml" C:/tmp/dogmud-boot-check/_datafiles/config.yaml && cd C:/tmp/dogmud-boot-check && go build -o boot-check.exe .
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
cd C:/tmp/dogmud-3a-harmful && git worktree prune
```

- [ ] **Step 7: Playtest** per `dogmud-playtesting` (ephemeral goals, `--checkout C:/tmp/dogmud-3a-harmful`, `docker rm -f` teardown, never touch the owner's server). Write the two goals files with the Write tool.

`tools/playtest/goals/2026-09-28-spell-3a-town-crime.yaml`:

```yaml
name: Spell effects 3a, a harmful spell on a citizen is a crime
summary: >
  Parity slice 3a makes a player's harmful spell on a mob an assault, exactly
  as melee is. Cast single-target harm spells at a Thornwall citizen with a
  city guard watching, and compare with a melee attack as the control.
ephemeral:
  profile: mid
  start_room: 460
  overlays:
    grant_spells:
      mind-spike: 1
      blood-boil: 1
      kinetic-shove: 1
  budgets:
    wall_clock: 30m
goals:
  - >-
    You start in the Gate Ward of Thornwall (room 460), where a city guard and
    a crossbowman stand. The city beggar spawns next door (room 461) and
    wanders. Find a room where a citizen (the city beggar or the street
    performer) and a guard are both present. Cast mind-spike at the citizen
    once. Report VERBATIM every line you read, and what the guard does over
    the next few rounds: does it turn on you, call out, or pursue you?
  - >-
    Control: find a different citizen with a guard present, use `attack` on
    it once, then flee. Report whether the guard's reaction to the melee
    attack matches its reaction to the spell. Any difference is a finding.
  - >-
    Cast blood-boil and kinetic-shove at a citizen. Quote the lines you read
    (afflicts, slams to the ground) and say whether the citizen fights back.
  - >-
    General adversarial pass: a line naming the wrong party, a line sent
    twice, raw numbers, or Go error text is a finding. Report bluntly.
```

`tools/playtest/goals/2026-09-28-spell-3a-mob-caster.yaml`:

```yaml
name: Spell effects 3a, a creature's spells behave like a player's
summary: >
  A hostile caster's harmful spells now go through the same appliers as a
  player's. Stand against one and read every line.
ephemeral:
  profile: mid
  start_room: 4052
  budgets:
    wall_clock: 25m
goals:
  - >-
    You start on the North Road near a hostile bandit caster who casts
    mind-spike, mind-fog and nerve-disruption. Let her cast at you across
    many rounds, retreating to recover when needed. Quote every spell line
    addressed to you, including resisted ones, and say whether each landed
    spell also moved your health.
  - >-
    Watch for her spell backfiring on her. If it happens, quote the line.
  - >-
    General adversarial pass: a line naming the wrong party, a line sent
    twice, raw numbers, or Go error text is a finding. Report bluntly.
```

  Run lane A, then lane B:

```text
/playtest local --checkout C:/tmp/dogmud-3a-harmful feature-tester 2026-09-28-spell-3a-town-crime.yaml
/playtest local --checkout C:/tmp/dogmud-3a-harmful feature-tester 2026-09-28-spell-3a-mob-caster.yaml
```

  Before tearing lane A down, read the crime log from its container as ground truth, independent of what the agent saw:

```bash
docker exec <container> sh -c 'find / -path "*factions.crimes*" -name "thornwall_citizens.yaml" 2>/dev/null | xargs cat'
```

  Expected: at least one `kind: assault` row whose perpetrator is the playtest character, created by the spell goal (before the melee control). Extract findings to memory (reports are gitignored). A lane blocked by the environment (no citizen and guard together, the caster never casts) is reported as blocked, not as passed; the parity table and the crime unit test remain the evidence.

  Commit the goals files:

```bash
cd C:/tmp/dogmud-3a-harmful && git add tools/playtest/goals/2026-09-28-spell-3a-town-crime.yaml tools/playtest/goals/2026-09-28-spell-3a-mob-caster.yaml && git commit -F - <<'EOF'
test(playtest): slice 3a goals, spell crime in town and a mob caster

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

- [ ] **Step 8: PR.** Push and open it against the fork only:

```bash
cd C:/tmp/dogmud-3a-harmful && git push -u origin feature/spell-effects-3a-harmful
```

Write the body to a file in the scratchpad with the Write tool, containing: a summary (one applier per harmful effect, crime ruling), the owner rulings (full unification split in two; harmful spells are crimes), the ten "Behaviour changes" above verbatim, Spec correction 1 and how it was resolved, the guard changes (dot rows collapsed, four narration rows retired, the one-contest guard reads `spell_effects.go`), the gate results, the playtest summary with the container's crime row, and "Next: slice 3b (helpful effects)". End the body with a blank line and `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.

```bash
gh pr create --repo pruuk/DOGMud --base master --head feature/spell-effects-3a-harmful --title "Spell effects 3a: one applier per harmful effect; spells are crimes" --body-file <scratchpad body file>
```

Confirm the printed URL says `pruuk/DOGMud`. Then `gh pr checks <n> --repo pruuk/DOGMud --watch`, and confirm with `gh run list --repo pruuk/DOGMud --branch feature/spell-effects-3a-harmful` that lint and tests both ran. Do not deploy; the owner deploys.

---

## Self-review against the spec

| Spec requirement (3a) | Task |
|---|---|
| One context, one dispatcher; old names kept as wrappers | 1 |
| damage: one applier; PP and MP arms deleted; `cancelDamageConditions`, procs, aggro incl. defensive crit, crime for player-on-mob | 2 |
| dot: caster's own skill and stat (Manifestation-aware, as MP), MM fixed, PP gains it | 3 (Spec correction 1) |
| knockdown: one applier; PP gains it; MP gains `cancelDamageConditions` | 4 |
| drain_area stays in its resolver; narration moves to the shared helpers | 5, Step 6 |
| Backfire narration for every caster kind; `RecordSpell` in PP and MM; interrupt generalised; MM aggro and room narration | 5 (MM aggro and room lines land in 2-4) |
| Parity table PM, PP (PvP room), MM, MP: same amount, aggro, record, room line, PM crime and aggression | 6 (crime row itself: 2) |
| Guard rows re-keyed or collapsed; messaging rows re-derived | 1-5, 7 |
| Last commit deletes wrappers no test still names | 8 (`applyMobEffect`, `applyPlayerEffect` still named by 3b-scope tests) |
| Gate, boot check, playtest (guarded town crime; mob caster vs player), PR | 9 |
