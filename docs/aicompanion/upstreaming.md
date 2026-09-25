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
| `internal/companionai/relay.go`: the relay seams (sender, inbound, page, origin) | Nil-safe. `ServeRelayPage` claims nothing, `RelayOrigin` is "". |
| `modules/gmcp`: `gmcp.Relay.go` and one `case` in `HandleIAC` for `Companion.Relay.Response`, `.Ready`, `.Gone` | The sender is installed but nothing calls it; an inbound relay message reaches a nil seam and is dropped. |
| `internal/web/web.go`: `serveTemplate` asks `ServeRelayPage` first; a CSP on `webclient-pure.html`; the `COMPANION_RELAY_ORIGIN_JSON` template value | One nil check per request. The CSP is set only while a relay origin exists, so none is sent; the template value is `""`. |
| `webclient-pure.html`: the glue script, a hidden "Companion key" button, a `Companion.Relay.Request` handler | The glue's `boot` returns nothing when the relay origin is empty, so no frame is built and the button stays hidden. |
| `webclient-pure.html`: `escapeHTML` also escapes single quotes | Hardening on its own merits: both built-HTML `innerHTML` sites (Quests, Status) already escaped every GMCP string, now proven by `tools/jstest/webclient-html-escape.test.js`. |
| `internal/actions/cast.go`: the no-target fallback of harmful single and multi spells asks `CanPvp` for a player foe | A fix on its own merits, not the module's: the named-target branches already asked. |

So the honest cost to a server that does not want it: two events queued
that nobody reads, a handful of nil checks, one refactor in `ask.go`, and a
mob template.

## Who pays: three tiers

Switched on, each call is paid for by the first of these the companion's
owner has: their own key, relayed through their web client (tier 2,
`PlayerKeys` plus `RelayOrigin`); the server's key (tier 3, `APIKeyEnv` or
`APIKey`); or nobody (tier 1, authored lines). Go defaults leave tier 2 off;
a fork that wants to pay server-side sets a key and leaves `PlayerKeys` as
it likes. Details, and what tier 2 does and does not send, are in
[`settings.md`](settings.md) ("Who pays for a call") and
[`testing-and-prompts.md`](testing-and-prompts.md) ("A player's own key").

Tier 2 needs a deploy step beyond the config:

1. DNS for the relay subdomain (for example `keys.example.org`) pointing at
   the game's server.
2. A reverse-proxy site block for that host proxying to the game's web port
   with the `Host` header preserved: the Go server tells the relay apart
   from the game by `Host` alone.
3. `RelayOrigin: "https://keys.example.org"` and `PlayerKeys: true` in the
   production config.

Until all three are in place, `PlayerKeys` alone offers nothing. Players
must then reach the game on exactly the `FilePaths.WebDomain` host (the
relay page's `frame-ancestors` names that host alone, so `www.` against the
bare domain fails), and a player using a local Ollama must start it with
`OLLAMA_ORIGINS` set to the relay origin.

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
want: the toggle, the key, the three model tiers, both token budgets, player
keys (`PlayerKeys`, `RelayOrigin`, `RelayTimeoutSeconds`), how
companions are handed out, and the two privacy choices. Every other setting
is documented with its default in [`settings.md`](settings.md), and any of
them can be set in the same block.

Shipped defaults are the cautious ones: the module off, existing characters
left alone until an operator asks, and speech that was not addressed to a
companion neither remembered nor sent.

## Before opening the pull request

1. Say plainly in the PR that the module sends player text to OpenAI (or,
   on tier 2, to the provider the player chose, from their browser) when
   it is switched on, what is sent (`testing-and-prompts.md`), and that it
   is inert without a key.
2. Say plainly that switching the module on also switches on the
   possibility of a companion romance. A player is asked for consent
   before anything they say is sent to OpenAI at all (`testing-and-prompts.md`,
   Consent); romance is a further, separate step on top of that consent,
   and it never starts or advances on its own. It only moves at the
   player's own command (`companion-court`), and `companion-boundary
   friendship` closes it at any point.
3. The two events are worth offering on their own merits: `Emote` and
   `Healed` are general and cheap, and any module could use them.
4. Expect the maintainers to want `internal/usercommands/ask.go` split into
   its own change, since it is the only engine refactor rather than an
   addition.
