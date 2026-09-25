# Sending the AI Companion Upstream

What a server that does not want it is asked to carry, and how to be sure
it carries nothing at all while it is switched off.

## Switched off, nothing happens

`Modules.aicompanion.Enabled: false` is the shipped default, and it makes the
module inert:

- `onLoad` returns after reading the config: no profiles are loaded, no bond
  file is read, no listener is registered and no engine seam is installed,
  so the nil checks in the engine find nothing and take the old path.
- No round handler, no model calls, no files written, nothing saved.
- Its commands (`companion-part`, `companion-unstick`, `companion-court`,
  `companion-boundary`, `companion-ask`, and the admin `aicompanion`) hand
  the input straight back, so the server answers "unknown command" exactly
  as it did before the module existed.
- No companion is ever bonded, so every engine change below is on a code
  path that is never reached.

Removing the one line from `modules/all-modules.go` takes the module out of
the build entirely. Nothing in `internal/` depends on it: the seams in
`internal/companionai` are nil-safe function holders, and with no module
installed they return false and the engine carries on.

## What the engine carries either way

| Change | Effect with the module off |
|---|---|
| `characters.CompanionBonded` and `CompanionInfo.Gold` | A new constant and a field on a struct nothing creates. |
| Bonded branches in the companion hooks (despawn, spawn, death, reserve backfill, rename, dismiss) | Guarded by `SourceType == CompanionBonded`; no such companion exists. |
| `internal/companionai` (new package) | Nil-safe seams. Every call returns false. |
| `companionai` calls in `MobIdle_HandleIdleMobs`, `companion_follow`, `PlayerSpawn_HandleJoin` | One nil check per call. |
| `events.Emote`, fired by `emote` | One queued event per player emote, with no listeners. |
| `events.Healed`, fired by a heal on a mob | One queued event per healing spell on a mob, with no listeners. |
| `ask.go`: the NPC dialogue chain moved into `askNpcChain` | Pure refactor; the player path is byte-for-byte the same sequence. |
| `ask.go`: `companionai.RouteAsk` before the "it ignores you" reply | One nil check. |
| Guard-test allowlists (`condition_apply_path_guard_test.go`, `pool_mutation_guard_test.go`) | Line numbers and one exemption; no runtime effect. |
| `_datafiles/.../9800-mara_venn.yaml` | One mob template nothing spawns. |

So the honest cost to a server that does not want it: two events queued
that nobody reads, a handful of nil checks, one refactor in `ask.go`, and a
mob template.

## A note for the maintainers

A plugin's `files/data-overlays/config.yaml` is merged with
`AddOverlayOverrides` after `_datafiles/config.yaml` has been read, and only
values from `config-overrides.yaml` are protected from it. A module default
therefore silently overrides the file operators edit: setting
`Modules.<module>.Enabled: true` in `config.yaml` has no effect if the module
ships that key in its overlay. This module carries no overlay for that
reason (its defaults are in `buildConfig`), but the trap applies to any
module that does.

## The settings an operator sees

`_datafiles/config.yaml` carries an `aicompanion` block under `Modules:`,
beside `playtest` and `weather`, with the settings an operator is likely to
want: the toggle, the key, the three model tiers, both token budgets, how
companions are handed out, and the two privacy choices. Every other setting
is documented with its default in [`settings.md`](settings.md), and any of
them can be set in the same block.

Shipped defaults are the cautious ones: the module off, existing characters
left alone until an operator asks, and speech that was not addressed to a
companion neither remembered nor sent.

## Before opening the pull request

1. Say plainly in the PR that the module sends player text to OpenAI when
   it is switched on, what is sent (`testing-and-prompts.md`), and that it
   is inert without a key.
4. The two events are worth offering on their own merits: `Emote` and
   `Healed` are general and cheap, and any module could use them.
5. Expect the maintainers to want `internal/usercommands/ask.go` split into
   its own change, since it is the only engine refactor rather than an
   addition.
