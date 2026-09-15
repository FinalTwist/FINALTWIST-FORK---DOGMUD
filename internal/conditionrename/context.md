# internal/conditionrename

The single spelling map for conditions unification slice 3
(`docs/superpowers/specs/2026-09-15-conditions-unification-slice-3-disk-wire-design.md`).

- `Apply(s string) string` renames every buff spelling in `s` to its
  condition spelling. Case-preserving (`buff`/`Buff`/`BUFF`), with explicit
  exceptions (`melee_self_buff` → `melee_self_empower`, `aura_enemy_debuff` →
  `aura_enemy_condition`, `permabuff` → `permanent`, `debuff` → `harmful
  condition`) and protected words that are left alone (`buffer`, `buffet`,
  `buffed`, `rebuff`, `Buffalo`). Idempotent.
- `HasBuff(s string) bool` reports a remaining buff spelling outside the
  protected words. The root guard uses it.

Readers: `internal/migration/0.17.0.go` (new save key names), the root guard
`identifier_word_guard_test.go`. Do not add a second list of old spellings
anywhere else; extend this one.
