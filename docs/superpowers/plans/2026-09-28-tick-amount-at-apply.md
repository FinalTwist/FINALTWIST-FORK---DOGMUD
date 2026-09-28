# Tick Amount At Apply Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A heal- or damage-over-time amount is computed where its condition lands, with caster scaling on every cast for player and mob casters alike.

**Architecture:** `events.Condition.TickScale` carries the scale; `Condition_ApplyConditions` computes the amount after a successful add; `spellTickScale(caster)` is the one caster formula; the spell hook passes it through a new `AddConditionTickScaled` event door; four post-queue `SetTickAmount` blocks are deleted.

**Tech Stack:** Go, `go test`.

**Spec:** `docs/superpowers/specs/2026-09-28-tick-amount-at-apply-design.md` (owner ruling A). Its 8-row facts table is authoritative.

---

### Task 0: Worktree

- [ ] From the main checkout: `git worktree add ../DOGMud-ticks -b feature/tick-amount-at-apply docs/ticks-parity-spec`; in it `go build ./... && go test ./internal/hooks/ ./internal/conditions/ -count=1` passes.

### Task 1: The event carries a scale and the apply hook computes the amount

**Files:** `internal/events/eventtypes.go`, `internal/hooks/Condition_ApplyConditions.go`, `internal/hooks/condition_tick_amount.go`, tests in `internal/hooks/condition_tick_at_apply_test.go`.

- [ ] **Step 1: failing test.** Seed a `tick_pool: health`, `tick_percent: 0.10` condition. Queue `events.Condition{UserId: <a test user>, ConditionId: id, TickScale: 2.0}` (and a mob twin with `MobInstanceId`) and run the handler the way existing `Condition_ApplyConditions` tests do (grep `Condition_ApplyConditions(` in `internal/hooks/*_test.go` and copy its fixture). Assert that right after the handler the held record's `TickAmount` equals `ComputeTickAmount(HealthMax, 0.10, variance, min, 2.0)`, i.e. is computed without waiting for a tick; with `TickVariance` 0 in the seed so the value is deterministic. A second row applies it again (refresh) with `TickScale` 3.0 and expects the amount recomputed. A third row with `TickScale` 0 expects scale 1.0. `tick_percent` is a FRACTION (0.10 = 10%).
- [ ] **Step 2:** it fails (amount 0).
- [ ] **Step 3: implement.**
  - `eventtypes.go`: add to `Condition`
    ```go
    // TickScale scales a tick_pool condition's per-round amount, computed
    // where the condition lands (Condition_ApplyConditions). 0 means 1.0.
    // Spells pass the caster's spellTickScale; potions and hazards pass none.
    TickScale float64
    ```
  - `condition_tick_amount.go`: extract the pool switch into `tickPoolMax(c *characters.Character, pool string) int` and add
    ```go
    // setTickAmountAtApply computes a tick_pool condition's per-round amount
    // from the holder's pool and the applier's scale, where the condition
    // lands. It replaces the post-queue SetTickAmount calls that found no
    // record on a first application.
    func setTickAmountAtApply(c *characters.Character, spec *conditions.ConditionSpec, conditionId int, scale float64) {
        if spec == nil || spec.TickPool == "" {
            return
        }
        if scale <= 0 {
            scale = 1.0
        }
        amt := conditions.ComputeTickAmount(tickPoolMax(c, spec.TickPool), spec.TickPercent, spec.TickVariance, spec.TickMin, scale)
        c.Conditions.SetTickAmount(conditionId, amt)
    }
    ```
    and make `fillZeroTickAmount` use `tickPoolMax`; update its comment: it is now the fallback for synchronous character-level adds only.
  - `Condition_ApplyConditions.go`: immediately after the `if addErr != nil { return events.Continue }` block, call `setTickAmountAtApply(targetChar, conditionInfo, evt.ConditionId, evt.TickScale)`.
- [ ] **Step 4:** tests pass; `go test ./internal/hooks/ -count=1` passes.
- [ ] **Step 5:** commit `feat(conditions): a tick amount is computed where its condition lands`.

### Task 2: One caster formula and the spell door

**Files:** `internal/hooks/light_spell.go` (the hook), new `internal/hooks/spell_tick_scale.go`, `internal/users/userrecord.go`, `internal/mobs/mobs.go`, tests `internal/hooks/spell_tick_scale_test.go`.

- [ ] **Step 1: failing tests.**
  - `spellTickScale`: skill 0, no weapon gives `SkillMultiplier(0)` (1.0 under validated defaults); a seeded weapon item with `SpellDamageMultiplier` 1.5 equipped multiplies by 1.5 (times `GearEffectivenessMultiplier` of an unmutated character, 1.0); a player character and a mob character with the same skill and weapon return the same value.
  - Spell path: casting a seeded `tick_pool` condition spell through `applySpellCondition` at a UserRecord target and at a Mob target queues an event whose `TickScale == spellTickScale(caster)` (drain with `events.DrainQueuedConditionsForTest` / `DrainQueuedMobConditionsForTest`). A non-ticking condition queues `TickScale` 0; a magnitude light still queues `Magnitude`.
- [ ] **Step 2:** they fail to compile.
- [ ] **Step 3: implement.**
  - `spell_tick_scale.go`:
    ```go
    // spellTickScale is the one caster formula for a spell's heal- or
    // damage-over-time (player/mob parity slice 2): the caster's
    // spellcasting SkillMultiplier times the equipped weapon's spell
    // multiplier, adjusted for gear effectiveness. It used to be computed
    // three ways (player caster with the weapon, mob self-cast without it,
    // mob on mob not at all).
    func spellTickScale(caster *characters.Character) float64 {
        if caster == nil {
            return 1.0
        }
        scale := combat.SkillMultiplier(caster.GetSkillLevel(skills.Spellcasting))
        if caster.Equipment.Weapon.ItemId > 0 {
            if ws := items.GetItemSpec(caster.Equipment.Weapon.ItemId); ws != nil && ws.SpellDamageMultiplier > 0 {
                scale *= ws.SpellDamageMultiplier * mutations.GearEffectivenessMultiplier(caster.Mutations)
            }
        }
        return scale
    }
    ```
  - `UserRecord.AddConditionTickScaled(conditionId int, scale float64, source string)` and `Mob.AddConditionTickScaled(...)`: queue `events.Condition` with `TickScale: scale`, `LifeEpoch` as their siblings do.
  - `spellConditionTarget` gains `AddConditionTickScaled`. `applySpellCondition`: magnitude path unchanged; else if the spec has a `TickPool`, `target.AddConditionTickScaled(conditionId, spellTickScale(caster), "spell")`; else `AddCondition`.
- [ ] **Step 4:** tests pass.
- [ ] **Step 5:** run `go test . -run TestPlayerConditionsTravelTheEventPath -count=1`; allowlist the two new door bodies as EVENT doors (reason like the `light_spell.go` entry). Commit `feat(spells): one caster tick scale for players and mobs, passed where the condition lands`.

### Task 3: Delete the dead post-queue blocks

**Files:** `internal/hooks/spell_resolution.go` (the three `// Compute tick snapshot` blocks, near 789, 1141, 1507), `internal/actions/drink.go` (the snapshot block near 359-379).

- [ ] **Step 1:** delete each block, keeping the `applySpellCondition(...)` call before it and the loop around it. In `drink.go` the tick block and its comment go; the drink path passes no scale (potions are 1.0 and are computed at apply).
- [ ] **Step 2:** `go build ./...`; remove now-unused imports.
- [ ] **Step 3:** the apply-path guard and `messaging_surface_guard_test.go` may be keyed by line; run the whole root package and re-key shifted entries with their reasons unchanged.
- [ ] **Step 4:** add a spell-level regression test: a player caster with spellcasting 50 (SkillMultiplier well above 1) casts the seeded vital-surge-shaped condition at a FRESH target; after the apply handler runs, `TickAmount` equals the scaled value (red on master: it was 0 then 1.0 at the first tick).
- [ ] **Step 5:** commit `refactor(spells): drop the post-queue tick snapshots`.

### Task 4: Gate, docs, PR

- [ ] `gofmt -l internal/ modules/ *.go`; `go vet ./...`; `go build ./...`; `go test ./... -count=1` (background); `~/go/bin/golangci-lint run --new-from-merge-base=origin/master`; boot check per `dogmud-shipping` (detached worktree `C:/tmp/dogmud-boot-check`, CONFIG_PATH port overrides, kill by PID).
- [ ] context.md for `internal/hooks` (`setTickAmountAtApply`, `spellTickScale`, `tickPoolMax`, the fallback's new role), `internal/events` (`TickScale`), `internal/users` and `internal/mobs` (`AddConditionTickScaled`), `internal/actions` (drink no longer snapshots). Verify symbols; `python tools/context_md_audit.py`. PATCH_NOTES entry (80 cols, no numbers, no dashes): healing spells that work over time now draw on the caster's skill and focus from the first cast, and creatures' healing spells do too. README plan row.
- [ ] PR with `gh pr create --repo pruuk/DOGMud --base master`, body stating the balance change for the three spells.
