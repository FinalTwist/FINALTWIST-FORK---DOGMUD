# AI companion: player-supplied keys (three tiers)

Status: design approved in brainstorm 2026-09-25, awaiting spec review.
Ships in the same PR as `docs/superpowers/plans/2026-09-25-aicompanion-integration-followup.md`
(branch `fix/aicompanion-integration`).

## Why

As merged in #161 the module has one OpenAI key, the server's, so the operator
pays for every companion. The owner will not pay for API calls on prod, but
wants a fork (finaltwist's, or anyone's) to be able to pay server-side. So:

1. **Tier 1, default.** No usable key: the companion follows, fights and
   answers with authored set lines. Costs nothing.
2. **Tier 2, player keys (on for our prod).** A player opts in on the web
   client and supplies their own endpoint, key and model. The key never
   reaches our server; the player pays.
3. **Tier 3, server key (off by default).** Today's behaviour: the server's key
   pays, bounded by the daily budgets.

## Facts verified against source (branch head `46850274d`)

| Fact | Where |
|---|---|
| Server key comes only from env var `APIKeyEnv` (default `OPENAI_API_KEY`) or `APIKey` in config; no per-player key exists | `modules/aicompanion/aicompanion.go:290-297`, `config.go:16,20` |
| Every outbound request goes through `send(req *http.Request, gate *consentLedger, ownerUserId int, kind outboundKind)`; only `listModels` is `carriesNoPlayerData` | `openai.go:183-189` |
| `callModelOnce` builds the request (`BaseURL+"/chat/completions"`, `Authorization: Bearer`), calls `send`, reads at most 1 MiB | `openai.go:255-275` |
| `modelCall` carries `BaseURL`, `APIKey`, `Model`, `OwnerUserId` | `openai.go:112-131` |
| The model's answer is a `Decision`: speech, action, mood, memory, facts, opinion, promise, impression, goal, autonomy, combat, leave | `decision.go:14-30` |
| Opinion changes are clamped per event by `boundDelta`, and warm words stop counting after 3 in an hour | `opinion.go:121-158` |
| Romance stages need opinion thresholds, milestones (max 2 gained per day) and sessions, and advance only by `companion-court` | `romance.go:117-120,192`, `romance.go:414` |
| Output moderation (`ModerateOutput`, default true) calls the provider's moderation endpoint through `send` | `config.go:70-71,235` |
| Companion speech is issued as a mob `say`/`emote` command | `runtime.go:966-990` |
| `UserRecord.Muted` blocks a player's custom communication | `internal/users/userrecord.go:49` |
| A player's own `say` is not written to any log today | `internal/usercommands/say.go` (no `mudlog` call) |
| Inbound GMCP is parsed on the connection goroutine in `HandleIAC`; state-touching `Char.*` ops are copied and queued as `GMCPCharOp` for MainWorker | `modules/gmcp/gmcp.go:367-395`, `gmcp.CharOp.go:12-26` |
| Outbound GMCP is `events.AddToQueue(gmcp.GMCPOut{UserId, Module, Payload})` | `modules/gmcp/gmcp.go:88-94` |
| The web client is served by the Go web server at `/webclient` (`webclient-pure.html`) | `internal/web/web.go:150` |
| No Content-Security-Policy header or meta tag exists anywhere in the repo | grep, zero hits |
| Two `innerHTML` assignments of built HTML in the web client (`host.innerHTML = html`, `container.innerHTML = html`); the other hits are comments or clearing to `""` | `webclient-pure.html:957,2183` |
| JS tests are dependency-free node scripts in `tools/jstest/*.test.js`, run by CI's `validate / javascript` job | `.github/workflows/validate.yml:90`, `tools/jstest/safe-dom.test.js` |
| The Caddy config is not in the repo; it lives on the droplet | `git ls-files` |
| The site's own web host is already configured as `FilePaths.WebDomain` (also consulted by the `/ws` origin check) | `internal/configs/config.filepaths.go:4`, `config.network.go:15-18` |

## Decisions (brainstorm, 2026-09-25)

| # | Decision |
|---|---|
| D1 | Dialogue from any tier is shown to the room like any NPC's. For accountability it is the OWNER's speech: the owner's mute silences her, and in tier 2 each line is logged with the owner's user id. |
| D2 | Tier 2 gets the same opinion, romance and memory bounds as tier 3. No extra caps: forging your own companion's answers buys a somewhat faster arc, not a skipped one, and harms only yourself. |
| D3 | One provider path: any OpenAI-compatible chat completions endpoint (OpenAI, OpenRouter, Ollama, LM Studio). No native Anthropic path; OpenRouter covers it. |
| D4 | The key stays in the browser. Session-only by default; "remember on this device" is an opt-in checkbox that requires a passphrase. |
| D5 | Architecture: the browser is a relay. The server builds the request exactly as today; the browser adds the key, posts it, returns the raw reply. |
| D6 | Key setup, passphrase and the remember box require the player to be logged in, and a stored key is bound to the account that stored it. |
| D7 | The key lives on a separate origin (a relay page on its own subdomain) so no script in the game page can read it. |
| D8 | Setup warns: put a spending cap on the key, do not use a work computer, and your key also pays when strangers talk to your companion. |

## Design

### Operator config (`Modules.aicompanion`)

- `PlayerKeys: false` (Go default) and `true` in our shipped `config.yaml`.
- `RelayOrigin: ""`: the relay page's origin, e.g. `https://keys.example.org`.
  Tier 2 is offered only when `PlayerKeys` is true AND `RelayOrigin` is a
  non-empty `https://` origin different from the game's own.
- Server key: unchanged (`APIKeyEnv` / `APIKey`). Absent on our prod.
- Per call, the payer is chosen in this order: the owner's relay if the owner
  has a live, unlocked relay; else the server key if present; else tier 1.
- `ModerateOutput` applies only to tier 3. Tier 2 has no moderation (a
  player's provider may have no moderation endpoint, and the reply is
  forgeable anyway); D1's accountability replaces it.

### Server side (module)

1. **Delivery seam.** `send` stays the single choke point and keeps the
   consent check. Below it, delivery becomes one of two transports:
   `httpTransport` (today's `httpClient.Do`, server key) and
   `relayTransport` (tier 2). `callModelOnce` builds the same JSON body for
   both; only the relay path omits `Authorization` and the URL.
2. **Relay transport.** Sends GMCP `Companion.Relay.Request`
   `{id, body}` to the owner, where `body` is the chat completions JSON and
   `id` is a random 128-bit hex string. Waits on a per-request channel for
   `Companion.Relay.Response` `{id, status, body}` with the same deadline as
   the HTTP path. Timeout, disconnect, or a malformed reply is a failed call:
   the existing failure handling refunds the reservation and she uses set
   lines for that turn.
3. **Inbound reply.** `HandleIAC` parses `Companion.Relay.Response`, checks
   that the connection's user owns an outstanding request with that id, and
   hands the bytes to the waiting channel. It touches no game state, so it
   does not need MainWorker; the pending map has its own mutex. Unknown ids,
   a reply from another user, a second reply to one id, or a body over 1 MiB
   are dropped and counted.
4. **Key-shaped reply guard.** A reply whose body contains a key-shaped string
   (`sk-`, `sk-or-`, `Bearer `, or any `Authorization` header text) is
   dropped, logged once a minute without the matched text, and treated as a
   failed call.
5. **Relay state per player.** GMCP `Companion.Relay.Ready {model}` and
   `Companion.Relay.Gone` from the client toggle whether the owner has a live
   relay. It carries no key, no endpoint. Logout or disconnect clears it.
6. **Reflection.** For a relay owner, logout reflection cannot reach a closed
   browser; it is queued and runs at the next login once the relay is ready.
7. **Strangers.** A stranger talking to a tier-2 companion is paid for by the
   owner's key. Task 3's stranger pacing still applies. New per-player toggle
   `companion-ai strangers on|off` (default on) lets the owner stop strangers
   prompting calls; she still hears them and answers with set lines.
8. **Accountability (D1).** If the owner is `Muted`, companion speech is
   suppressed, emotes included (both are free text). In tier 2 each line is
   logged at Info as `aicompanion speech owner=<id> companion=<name>`.
9. **Status.** `companion-ai` reports which tier is live for this player, and
   the admin status line shows the same.

### Browser side

1. **Relay page** (`/companion-relay.html`, served by the Go web server only
   when the request's Host matches `RelayOrigin`; 404 otherwise). It is loaded
   in a hidden iframe by the web client and:
   - refuses to run unless `window.isSecureContext`;
   - accepts `postMessage` only from the game origin, `https://` plus
     `FilePaths.WebDomain`, rendered into the page by the server (never taken
     from the iframe URL or the message);
   - holds the key in memory, or encrypted in its own `localStorage` when
     remembered;
   - posts only to the endpoint the player stored, never to a URL from a
     message; sets `Authorization` only there;
   - returns `{id, status, body}` and never echoes headers.
2. **Setup panel** in the web client, shown only while logged in: endpoint
   URL, key, model, "remember on this device", passphrase (required when
   remember is ticked), "Forget my key", and the D8 warnings. The panel's
   key and passphrase fields are rendered INSIDE the relay iframe (made
   visible for setup), so the key is never typed into the game page.
3. **Remembered keys.** PBKDF2-SHA256 (600,000 iterations, random 16-byte
   salt) derives an AES-GCM key from the passphrase; the ciphertext, salt and
   IV are stored under the account name. Unlock asks for the passphrase once
   per session. A stored key for account A is never offered to account B.
   The account name reaches the relay from the game page, which the relay
   cannot fully trust, so the binding keeps honest users apart on a shared
   browser; the passphrase is what actually protects a stored key.
4. **Game page glue.** Forwards `Companion.Relay.Request` from GMCP to the
   iframe and the iframe's answer back as `Companion.Relay.Response`. It never
   sees the key.

### Web client hardening

- A Content-Security-Policy on the game page and the relay page, set by the Go
  web server. Game page: scripts from self (and the CDNs it already uses),
  `frame-src` the relay origin. Relay page: `connect-src https:` plus
  `http://localhost:*` and `http://127.0.0.1:*` for local models, no inline
  scripts.
- Audit both `innerHTML` assignments; replace with DOM building or prove the
  inputs are escaped, with a `tools/jstest` test for each.

### Clarifications from planning (2026-09-25, read from source)

- **Budgets and breaker.** `DailyTokenBudget`, `DailyTokensPerCompanion` and
  the circuit breaker (`models.go:174-195`, global `breakerUntil`) exist to
  bound the SERVER key's bill. Tier 2 calls do not reserve against the server
  or per-companion budgets (the player pays); stranger pacing and
  `StrangerDailyTokens` still apply as abuse limits on the owner's key.
  Relay failures count against a per-owner breaker, never the global one, so
  one player's broken provider cannot stop everyone's companions.
- **Model.** A relay owner's model (from `Companion.Relay.Ready`) is used for
  every tier (fast, main, deep); `settingsFor`'s auto-pick from the server's
  model list applies only to tier 3. Reasoning effort is not sent in tier 2.
- **Game page CSP.** `webclient-pure.html` is a Go template with large inline
  scripts (`<script>` blocks at lines 381 and 3162, among others), so its CSP
  must allow `'unsafe-inline'` scripts. It still sets `frame-src` to the
  relay origin, `object-src 'none'` and `base-uri 'self'`. The key's
  protection is the separate relay origin (D7), not the game page's CSP.
- **Serving the relay.** The relay page and its script are embedded in the
  module (`//go:embed`). The engine's `serveTemplate` asks a new nil-safe
  seam in `internal/companionai` whether the module claims the request (Host
  equals the `RelayOrigin` host) before anything else; with the module off,
  nothing is claimed and the relay paths 404.
- **Spike result (done 2026-09-25).** A CORS preflight from origin
  `https://keys.example.org` with `authorization,content-type` headers:
  `api.openai.com/v1/chat/completions` answers 200 with
  `Access-Control-Allow-Origin` echoing the origin;
  `openrouter.ai/api/v1/chat/completions` answers 204 with `*`. Both are
  usable from the relay. Ollama needs `OLLAMA_ORIGINS` set to the relay
  origin; the setup panel says so.

## Error handling

Every failure in tier 2 degrades to tier 1 for that turn and refunds the
reservation: no relay, locked relay, timeout, provider error status,
malformed JSON, no tool-call support, a dropped key-shaped reply. Repeated
failures open a per-owner breaker (tier 2) or the global one (tier 3).
A player sees at most one plain
notice per session ("Mara falls back on her own few words: your key's
provider did not answer"), never raw errors.

## Testing

- **Spike (preflight done, see Clarifications):** the final boot check makes
  one real browser call through the relay to OpenRouter or OpenAI with a
  capped key. A provider that refuses browser calls is documented as
  unsupported, never proxied (proxying would put the key on our server).
- Go: a relay request payload never contains the key, an endpoint or any
  header; replies are matched by id and owner only; key-shaped replies are
  dropped; timeout falls back and refunds; muted owner silences speech;
  tier selection order; tier-3 path unchanged (existing tests stay green);
  the consent check still refuses relay calls for an unconsented owner.
- JS (`tools/jstest`): the relay refuses a non-secure context, a message from
  another origin, and any URL other than the stored endpoint; encrypt then
  decrypt round-trips; a wrong passphrase fails; a key stored for one account
  is not offered to another.
- Boot the server with the module on in tier 2 (no server key) and confirm the
  relay page 404s on the game host and serves on the relay host.

## Deploy (owner)

DNS for the relay subdomain, a Caddy site block proxying it to the same Go
server, `RelayOrigin` set in `config-production.yaml`, `PlayerKeys: true`.
Until that is done, prod runs tier 1: `PlayerKeys` without a valid
`RelayOrigin` offers nothing.

## Out of scope

Telnet players using their own key; a native Anthropic path; server-side
proxying of any kind; engine-wide changes to how player speech is logged.
