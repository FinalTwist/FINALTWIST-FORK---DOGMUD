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

## Part 1 tasks (sequential; each ends in its own commit)

Status 2026-09-25: Tasks 1-5 DONE (`d9e47f5ae` .. `46850274d`). Task 6 is
superseded by Task 13 at the end of Part 2, which runs last.

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

---

# Part 2: Player-keys tiers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a player run their bonded companion on their OWN OpenAI-compatible key, held in the browser on a separate origin and never on our server, while keeping the server-key tier for forks and set lines as the default.

**Architecture:** `send()` in `modules/aicompanion/openai.go` stays the single consent-guarded choke point; below it a relay transport ships the request BODY to the owner's web client over GMCP and waits for the provider's raw reply. The browser side is an embedded relay page on its own origin (hidden iframe) that alone holds the key. Two new nil-safe seams in `internal/companionai` connect the engine (`modules/gmcp`, `internal/web`) to the module without module-to-module imports.

**Tech Stack:** Go 1.x (existing), GMCP over the existing websocket, browser WebCrypto (PBKDF2 + AES-GCM), dependency-free node tests in `tools/jstest`.

**Spec:** `docs/superpowers/specs/2026-09-25-aicompanion-player-keys-design.md` (read it, including "Clarifications from planning").

## Rules for every Part 2 task

- Worktree `C:/tmp/dogmud-pr161` only; never the main checkout; never `C:/tmp/dogmud-boot-check`; no `git stash`; `git add` named paths only; no Python read-modify-write; no push, no `gh`; no memory edits.
- No em or en dashes in prose, comments or player text. Player text wraps at 80 columns and shows no raw numbers.
- `go test` by package path; never `go test .`.
- Every commit ends with `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`; write messages with a heredoc.
- Before writing a test, read `.claude/skills/dogmud-writing-tests/SKILL.md`. Every guard test is proven red (break the guard, run, see it fail, restore) and the proof is stated in the report.
- Code in this plan is the intended shape, written against the source read on 2026-09-25. Verify every signature you call against source first; where the code here disagrees with source, source wins and you report the difference.

## Part 2 facts (read from source at `aaa1aff42`)

| Fact | Where |
|---|---|
| `send(req *http.Request, gate *consentLedger, ownerUserId int, kind outboundKind) (*http.Response, error)` | `modules/aicompanion/openai.go:183` |
| `callModelOnce` marshals `chatRequest` into `body`, posts to `c.BaseURL+"/chat/completions"` with `Authorization: Bearer`, reads at most `1<<20` bytes, decodes `chatResponse` | `openai.go:209-300` |
| `modelCall{BaseURL, APIKey, Model, Timeout, ..., Ctx, OwnerUserId}` | `openai.go:112-131` |
| `modelCall` is built at four sites: `runtime.go:521`, `reflect.go:131`, `conversation.go:209`, `corememory.go:133` | grep `APIKey: *m.apiKey()` |
| `modelReady(ownerId ...int)` requires `m.apiKey() != ""`, closed breaker, server budget, owner budget | `aicompanion.go:327-342` |
| Global breaker `breakerUntil`, `breakerOpen`, `breakerResult` | `aicompanion.go:153`, `models.go:174-195` |
| `settingsFor(tier, tools)` picks `Model` from config or `m.models.pick(tier)` and sets `Effort` | `models.go:405-429` |
| `buildConfig(get getter) Config`; booleans read as `if v := get("X"); v != nil { c.X = asBool(v) }` | `modules/aicompanion/config.go:175-300` |
| Logout reflection starts at `runtime.go:333` (`m.startReflection(c.mind, c.profile, ownerName, c.sessionStartUnix)`) | `runtime.go:333`, `reflect.go:99` |
| Companion speech goes out as `mob.Command(l.Kind+" "+util.EscapeAnsiTags(piece), delay)` in `speak` | `runtime.go:966-990` |
| `UserRecord.Muted bool` | `internal/users/userrecord.go:49` |
| `internal/companionai` holds nil-safe seams (`SetHolder`/`HoldPosition`, `SetBondedCheck`, ...) | `internal/companionai/companionai.go` |
| No module except `modules/gmcp` emits `GMCPOut`; `GMCPOut{UserId int, Module string, Payload any}` with `[]byte` payloads sent as `Module + " " + payload` | `modules/gmcp/gmcp.go:88-94,463-520` |
| Inbound GMCP: `HandleIAC` splits `command payload`, `switch command`; `payload` aliases the read buffer and must be copied | `modules/gmcp/gmcp.go:250-395` |
| `users.GetByConnectionId(connections.ConnectionId) *UserRecord` | `internal/users/users.go:207` |
| `serveTemplate` is the `/` handler; `/webclient` renders `webclient-pure.html` as a Go template with `.CONFIG` | `internal/web/web.go:64,290,150` |
| Web client GMCP: inbound dispatch via `GMCPUpdateHandlers[path]()` after storing into `GMCPStructs`; outbound `SendGMCP(pkg, obj)` | `webclient-pure.html:2040-2075,2586-2596` |
| Web client script tags use `{{ .CONFIG.FilePaths.WebCDNLocation }}/static/js/...` | `webclient-pure.html:234-243` |
| `FilePaths.WebDomain` is the site's own host | `internal/configs/config.filepaths.go:4` |
| JS tests: `tools/jstest/*.test.js`, dependency-free, run by CI `validate / javascript`; node 20 has `globalThis.crypto.subtle` | `.github/workflows/validate.yml:90` |
| Built-HTML `innerHTML` assignments: `webclient-pure.html:957` (`host.innerHTML = html`) and `:2183` (`container.innerHTML = html`) | grep |
| CORS preflight: OpenAI 200 echoing origin; OpenRouter 204 `*` | curl, 2026-09-25 |

## File structure

| File | Responsibility |
|---|---|
| `internal/companionai/relay.go` (new) | Seams: relay sender (gmcp installs), relay inbound (module installs), relay page server and relay origin (module installs). Nil-safe. |
| `modules/gmcp/gmcp.Relay.go` (new) | Installs the relay sender; forwards inbound `Companion.Relay.*` to the seam. |
| `modules/gmcp/gmcp.go` (modify) | Three `case` labels in `HandleIAC`. |
| `internal/web/web.go` (modify) | Ask the relay-page seam first; CSP header on `/webclient`. |
| `modules/aicompanion/tiers.go` (new) | Config for tiers, per-owner relay state, route selection, per-owner breaker. |
| `modules/aicompanion/relay.go` (new) | Relay transport: pending requests, ids, waiting, key-shaped guard, inbound handling. |
| `modules/aicompanion/relaypage.go` (new) + `modules/aicompanion/relayweb/relay.html`, `relayweb/relay.js` (new, embedded) | Serves the relay page on the relay host with a strict CSP. |
| `modules/aicompanion/openai.go`, `models.go`, `runtime.go`, `reflect.go`, `conversation.go`, `corememory.go`, `commands.go`, `aicompanion.go`, `config.go` (modify) | Route-aware calls, tier-2 budget and breaker rules, reflection deferral, mute, speech log, `companion-ai` status and `strangers` toggle. |
| `_datafiles/html/public/static/js/companion-relay-glue.js` (new) | Game-page glue: iframe, GMCP forwarding, setup button. Never sees the key. |
| `_datafiles/html/public/webclient-pure.html` (modify) | Include the glue script, a Companion key button, `Companion` GMCP handler; fix the two `innerHTML` sites. |
| `tools/jstest/companion-relay.test.js`, `tools/jstest/companion-relay-glue.test.js`, `tools/jstest/webclient-html-escape.test.js` (new) | JS guards. |

---

### Findings from Task 7 (`7a12c974a`) that bind later tasks

- **Inbound websocket messages are capped at 64 KiB** (`wsMaxMessageBytes`,
  `internal/web/web.go:28`, applied at `:343`); a larger one closes the
  player's connection. Task 12's glue MUST measure the encoded
  `Companion.Relay.Response` frame (JSON-escaped body included) and, if it
  would exceed 60 KiB, send `{id, status: 0, body: ""}` instead. Task 9 keeps
  its 1 MiB check as a second line, and its tests should also cover a reply
  refused for size.
- **`/webclient` serves `webclient.html`, which frames `/webclient-pure.html`.**
  The game page CSP is set only on `webclient-pure.html`; the glue script and
  the relay iframe live in `webclient-pure.html`. Relay `frame-ancestors`
  must therefore allow the game origin that serves `webclient-pure.html` (the
  same `WebDomain` origin), which Task 11's CSP already does.
- **No `Core.Supports.Set` is needed** for a web client to receive
  `Companion.*`; skip that step in Task 12.
- The relay sender returns false when the client has not finished GMCP
  negotiation, so Task 9 sees `errRelayUnsent` rather than a timeout.

### Findings from Task 8 (`f75b37fea`) that bind later tasks

- The per-owner breaker's `failure`/`success` are called ONCE per call by
  `routeResult` at each call site. Task 9 must NOT call them inside
  `callModelOnce` (the plan's Task 9 code does; drop those two lines).
  `TestRelaySummaryChargesOnlyThePasserBy` would time out on a double count.
- Moderation is already skipped for relay routes (done in Task 8). Task 9
  adds only the transport branch in `callModelOnce`.
- Dispatch and summaries call `modelReadyFor(owner, asker)`; `relays` is
  created in `init()`. The route rides on `modelCall.Route`.

### Task 7: Engine seams for the relay

**Files:**
- Create: `internal/companionai/relay.go`, `internal/companionai/relay_test.go`
- Create: `modules/gmcp/gmcp.Relay.go`
- Modify: `modules/gmcp/gmcp.go` (the `switch command` in `HandleIAC`)
- Modify: `internal/web/web.go` (`serveTemplate`, top of function)
- Modify: `internal/companionai/context.md`, `modules/gmcp/context.md`

- [ ] **Step 1: Write the failing seam test** in `internal/companionai/relay_test.go`:

```go
package companionai

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRelaySeamsAreNilSafe(t *testing.T) {
	SetRelaySender(nil)
	SetRelayInbound(nil)
	SetRelayPage(nil)
	if SendRelay(1, `Companion.Relay.Request`, []byte(`{}`)) {
		t.Fatal("with no sender installed nothing is sent")
	}
	RelayInbound(1, `Companion.Relay.Response`, []byte(`{}`)) // must not panic
	rec := httptest.NewRecorder()
	if ServeRelayPage(rec, httptest.NewRequest(http.MethodGet, `/companion-relay.html`, nil)) {
		t.Fatal("with no page installed nothing is claimed")
	}
	if RelayOrigin() != `` {
		t.Fatal("with no page installed there is no relay origin")
	}
}

func TestRelaySeamsCallWhatIsInstalled(t *testing.T) {
	var sentTo int
	SetRelaySender(func(userId int, module string, payload []byte) bool { sentTo = userId; return true })
	defer SetRelaySender(nil)
	if !SendRelay(7, `Companion.Relay.Request`, []byte(`{}`)) || sentTo != 7 {
		t.Fatal("the installed sender must be used")
	}
	var got string
	SetRelayInbound(func(userId int, command string, payload []byte) { got = command })
	defer SetRelayInbound(nil)
	RelayInbound(7, `Companion.Relay.Ready`, []byte(`{}`))
	if got != `Companion.Relay.Ready` {
		t.Fatal("the installed inbound handler must be used")
	}
	SetRelayPage(func(w http.ResponseWriter, r *http.Request) bool { return true }, func() string { return `https://keys.example.org` })
	defer SetRelayPage(nil)
	if RelayOrigin() != `https://keys.example.org` {
		t.Fatal("the installed origin must be reported")
	}
}
```

- [ ] **Step 2: Run it, expect a compile failure** (`undefined: SetRelaySender`):
`go test github.com/GoMudEngine/GoMud/internal/companionai/...`

- [ ] **Step 3: Implement `internal/companionai/relay.go`:**

```go
package companionai

import "net/http"

// RelaySendFunc delivers one GMCP message to a player's client. The gmcp
// module installs it; it returns false when the player has no client to
// send to.
type RelaySendFunc func(userId int, module string, payload []byte) bool

// RelayInboundFunc receives a Companion.Relay.* message a player's client
// sent. The aicompanion module installs it. It runs on the connection's own
// goroutine, NOT under the mud lock, so it must touch no game state.
type RelayInboundFunc func(userId int, command string, payload []byte)

// RelayPageFunc serves the key relay page when a request is for the relay
// origin, and reports whether it did.
type RelayPageFunc func(w http.ResponseWriter, r *http.Request) bool

var (
	relaySend    RelaySendFunc
	relayInbound RelayInboundFunc
	relayPage    RelayPageFunc
	relayOrigin  func() string
)

// SetRelaySender installs the GMCP sender. Called by the gmcp module.
func SetRelaySender(f RelaySendFunc) { relaySend = f }

// SendRelay sends a relay message to a player's client. Nil-safe.
func SendRelay(userId int, module string, payload []byte) bool {
	if relaySend == nil {
		return false
	}
	return relaySend(userId, module, payload)
}

// SetRelayInbound installs the inbound handler. Called by the aicompanion
// module while it is switched on.
func SetRelayInbound(f RelayInboundFunc) { relayInbound = f }

// RelayInbound hands a client's relay message to the module. Nil-safe: with
// nothing installed the message is dropped.
func RelayInbound(userId int, command string, payload []byte) {
	if relayInbound == nil {
		return
	}
	relayInbound(userId, command, payload)
}

// SetRelayPage installs the relay page server and the relay origin it
// answers for. Called by the aicompanion module while it is switched on and
// player keys are allowed. SetRelayPage(nil) removes both.
func SetRelayPage(f RelayPageFunc, origin ...func() string) {
	relayPage = f
	relayOrigin = nil
	if f != nil && len(origin) > 0 {
		relayOrigin = origin[0]
	}
}

// ServeRelayPage lets the module claim a web request for the relay origin.
// Nil-safe: with nothing installed nothing is claimed.
func ServeRelayPage(w http.ResponseWriter, r *http.Request) bool {
	if relayPage == nil {
		return false
	}
	return relayPage(w, r)
}

// RelayOrigin is the relay page's origin ("https://host"), or empty when no
// relay is offered. The web client's CSP uses it for frame-src.
func RelayOrigin() string {
	if relayOrigin == nil {
		return ``
	}
	return relayOrigin()
}
```

- [ ] **Step 4: Run the seam tests; expect PASS.**

- [ ] **Step 5: gmcp side.** Create `modules/gmcp/gmcp.Relay.go`:

```go
package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/companionai"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Companion.Relay.* carries a companion's model request to the owner's own
// browser and the provider's reply back, for players who run their
// companion on their own key. The key never passes through here: requests
// carry only a request body, replies only a status and a body.
//
// Relay messages touch no game state, so unlike Char.* ops they are handed
// over on the connection goroutine; the module's pending-request table has
// its own lock.

func installRelaySender() {
	companionai.SetRelaySender(func(userId int, module string, payload []byte) bool {
		if users.GetConnectionId(userId) == 0 {
			return false
		}
		events.AddToQueue(GMCPOut{UserId: userId, Module: module, Payload: payload})
		return true
	})
}

// relayInbound forwards a relay message from a connection to the module.
func relayInbound(connectionId connections.ConnectionId, command string, payload []byte) {
	u := users.GetByConnectionId(connectionId)
	if u == nil {
		return
	}
	companionai.RelayInbound(u.UserId, command, append([]byte(nil), payload...))
}
```

Call `installRelaySender()` from `modules/gmcp/gmcp.go` `init()` (line 46) next to the other registrations. In `HandleIAC`'s `switch command`, add before the `Char.Automation.Set` case:

```go
		case `Companion.Relay.Response`, `Companion.Relay.Ready`, `Companion.Relay.Gone`:
			relayInbound(connectionId, command, payload)
```

Verify the type of `connectionId` in `HandleIAC` and convert if it is not `connections.ConnectionId`. Check whether `GMCPOut` delivery to a web client requires the module name to be enabled via `Core.Supports.Set` (read `dispatchGMCP` past line 520); if it does, the glue in Task 11 must add `Companion 1` to its supports list, so note that in your report.

- [ ] **Step 6: web side.** At the top of `serveTemplate` in `internal/web/web.go` (line 64, before `httpRoot`), add:

```go
	// The companion key relay lives on its own origin. The module claims
	// requests for that host; nothing else is served there.
	if companionai.ServeRelayPage(w, r) {
		return
	}
```

And where `/webclient` is rendered (find where `reqPath` resolves to `webclient-pure.html`; set the header only for that page, before the template executes):

```go
	if relay := companionai.RelayOrigin(); relay != `` {
		w.Header().Set(`Content-Security-Policy`,
			`frame-src `+relay+`; object-src 'none'; base-uri 'self'`)
	}
```

Keep the CSP to these directives only: the page's inline scripts and CDN scripts must keep working, and `frame-src` plus `object-src` plus `base-uri` break nothing it uses today. Add a test in `internal/web` (find the existing test file pattern there) that with a fake `SetRelayPage` claiming host `keys.example.org`, a request with that Host is answered by the fake and a request for another host falls through.

- [ ] **Step 7: Gates and commit.**

```bash
gofmt -l internal/ modules/
go build ./...
go test github.com/GoMudEngine/GoMud/internal/companionai/... github.com/GoMudEngine/GoMud/internal/web/... github.com/GoMudEngine/GoMud/modules/gmcp/...
~/go/bin/golangci-lint run --new-from-merge-base=origin/master
git add internal/companionai/relay.go internal/companionai/relay_test.go internal/companionai/context.md modules/gmcp/gmcp.Relay.go modules/gmcp/gmcp.go modules/gmcp/context.md internal/web/web.go <the web test file>
git commit -F - <<'EOF'
feat(companionai): seams for a player's own-key relay

...body...

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
EOF
```

Expected: gofmt prints nothing, all packages ok, lint `0 issues`.

---

### Task 8: Tier config, route selection and the per-owner breaker

**Files:**
- Create: `modules/aicompanion/tiers.go`, `modules/aicompanion/tiers_test.go`
- Modify: `modules/aicompanion/config.go` (Config fields + `buildConfig`), `aicompanion.go` (`modelReady`, module fields), `models.go` (`settingsFor`), the four `modelCall` construction sites

- [ ] **Step 1: Failing tests** in `tiers_test.go`:

```go
package aicompanion

import (
	"testing"
	"time"
)

func TestRelayOriginMustBeHTTPSAndForeign(t *testing.T) {
	for _, tc := range []struct {
		origin, web string
		ok          bool
	}{
		{`https://keys.example.org`, `example.org`, true},
		{`http://keys.example.org`, `example.org`, false},
		{`https://example.org`, `example.org`, false},
		{`https://keys.example.org/path`, `example.org`, false},
		{``, `example.org`, false},
	} {
		if got := validRelayOrigin(tc.origin, tc.web); got != tc.ok {
			t.Errorf("validRelayOrigin(%q, %q) = %v, want %v", tc.origin, tc.web, got, tc.ok)
		}
	}
}

func TestRouteOrderRelayThenServerThenNone(t *testing.T) {
	m := &AICompanionModule{cfg: Config{Enabled: true, PlayerKeys: true, RelayOrigin: `https://keys.example.org`}}
	m.relays = newRelayTable()
	if r := m.route(5); r.kind != routeNone {
		t.Fatalf("no relay and no server key is tier 1, got %v", r.kind)
	}
	m.relays.ready(5, `gpt-4.1-mini`)
	if r := m.route(5); r.kind != routeRelay || r.model != `gpt-4.1-mini` {
		t.Fatalf("a live relay is used first, got %+v", r)
	}
	m.cfg.APIKey = `sk-server`
	m.relays.gone(5)
	if r := m.route(5); r.kind != routeServer {
		t.Fatalf("with the relay gone the server key covers, got %v", r.kind)
	}
	m.cfg.PlayerKeys = false
	m.relays.ready(5, `x`)
	if r := m.route(5); r.kind != routeServer {
		t.Fatal("player keys switched off ignores a relay")
	}
}

func TestRelayFailuresTripOnlyThatOwnersBreaker(t *testing.T) {
	m := &AICompanionModule{cfg: Config{Enabled: true, PlayerKeys: true, RelayOrigin: `https://keys.example.org`, BreakerErrors: 3, BreakerSeconds: 60}}
	m.relays = newRelayTable()
	m.relays.ready(5, `m`)
	m.relays.ready(6, `m`)
	now := time.Now()
	for i := 0; i < 3; i++ {
		m.relays.failure(5, now, m.cfg)
	}
	if m.route(5).kind != routeNone {
		t.Fatal("owner 5's own breaker must be open")
	}
	if m.route(6).kind != routeRelay || m.breakerOpen(now) {
		t.Fatal("owner 6 and the global breaker are untouched")
	}
}
```

Check the real names of the breaker config fields (`BreakerErrors`, `BreakerSeconds`) in `config.go` and use them.

- [ ] **Step 2: Run; expect compile failure** (`undefined: validRelayOrigin`).

- [ ] **Step 3: Implement.** In `config.go` add to `Config`:

```go
	// PlayerKeys lets a player run their companion on their own key, held
	// in their browser on the relay origin. Off unless the config says so.
	PlayerKeys bool
	// RelayOrigin is where the relay page is served, "https://host". Tier 2
	// is offered only when this is a valid https origin other than the
	// game's own.
	RelayOrigin string
	// RelayTimeoutSeconds bounds how long a call waits for the browser.
	RelayTimeoutSeconds int
```

and in `buildConfig`: `RelayOrigin: strings.TrimRight(strings.TrimSpace(asString(get("RelayOrigin"))), "/")`, `RelayTimeoutSeconds: asInt(get("RelayTimeoutSeconds"), 30)`, and `if v := get("PlayerKeys"); v != nil { c.PlayerKeys = asBool(v) }`.

`tiers.go`:

```go
package aicompanion

import (
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// routeKind is who pays for a call and how it travels.
type routeKind int

const (
	routeNone   routeKind = iota // tier 1: set lines, nothing leaves
	routeRelay                   // tier 2: the owner's own key, through their browser
	routeServer                  // tier 3: the server's key
)

type route struct {
	kind  routeKind
	model string // tier 2 only: the owner's chosen model, used for every tier
}

// validRelayOrigin accepts only "https://host[:port]" with no path, and not
// the game's own host: the key must live on an origin the game page cannot
// read.
func validRelayOrigin(origin string, webDomain string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != `https` || u.Host == `` || (u.Path != `` && u.Path != `/`) || u.RawQuery != `` {
		return false
	}
	return !strings.EqualFold(u.Hostname(), strings.TrimSpace(webDomain))
}

// playerKeysOffered reports whether tier 2 is on offer at all.
func (m *AICompanionModule) playerKeysOffered() bool {
	return m.cfg.Enabled && m.cfg.PlayerKeys &&
		validRelayOrigin(m.cfg.RelayOrigin, configs.GetFilePathsConfig().WebDomain.String())
}

// relayTable is which owners have a live, unlocked relay, and each one's
// own breaker. It is read from model goroutines and written from the
// connection goroutine, so it has its own lock.
type relayTable struct {
	mu     sync.Mutex
	owners map[int]*relayOwner
}

type relayOwner struct {
	model        string
	failures     int
	breakerUntil time.Time
	noticeSent   bool
}

func newRelayTable() *relayTable { return &relayTable{owners: map[int]*relayOwner{}} }

func (t *relayTable) ready(userId int, model string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.owners[userId] = &relayOwner{model: strings.TrimSpace(model)}
}

func (t *relayTable) gone(userId int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.owners, userId)
}

// live returns the owner's model when their relay is up and their own
// breaker is closed.
func (t *relayTable) live(userId int, now time.Time) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	o := t.owners[userId]
	if o == nil || o.model == `` || now.Before(o.breakerUntil) {
		return ``, false
	}
	return o.model, true
}

func (t *relayTable) failure(userId int, now time.Time, cfg Config) {
	t.mu.Lock()
	defer t.mu.Unlock()
	o := t.owners[userId]
	if o == nil {
		return
	}
	o.failures++
	if cfg.BreakerErrors > 0 && o.failures >= cfg.BreakerErrors {
		o.breakerUntil = now.Add(time.Duration(cfg.BreakerSeconds) * time.Second)
		o.failures = 0
	}
}

func (t *relayTable) success(userId int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if o := t.owners[userId]; o != nil {
		o.failures = 0
	}
}

// route decides who pays for a call on this owner's behalf: their own
// relay first, then the server's key, else nothing and set lines.
func (m *AICompanionModule) route(ownerId int) route {
	if m.playerKeysOffered() && ownerId > 0 && m.relays != nil {
		if model, ok := m.relays.live(ownerId, time.Now()); ok {
			return route{kind: routeRelay, model: model}
		}
	}
	if m.cfg.Enabled && m.apiKey() != `` {
		return route{kind: routeServer}
	}
	return route{kind: routeNone}
}
```

Add `relays *relayTable` to the module struct (`aicompanion.go` ~line 146) and set `m.relays = newRelayTable()` where the module's maps are initialised in `onLoad`.

Rewrite `modelReady` so a relay route skips the server-key checks:

```go
func (m *AICompanionModule) modelReady(ownerId ...int) bool {
	owner := 0
	if len(ownerId) > 0 {
		owner = ownerId[0]
	}
	switch m.route(owner).kind {
	case routeRelay:
		return true // the player's key: the server's budgets and breaker do not apply
	case routeNone:
		return false
	}
	... the existing server-key body unchanged from the breaker check down ...
}
```

Add a `Route route` field to `modelCall`. At each of the four construction sites set `Route: m.route(ownerId)` (use each site's owner variable) and, when `Route.kind == routeRelay`, set `Model` to `Route.model` and `Effort` to `""`. Do this in one helper to avoid four copies:

```go
// applyRoute fills in who pays for a call and, for a player's own key,
// the model they chose.
func (m *AICompanionModule) applyRoute(c *modelCall) {
	c.Route = m.route(c.OwnerUserId)
	if c.Route.kind == routeRelay {
		c.Model, c.Effort, c.APIKey, c.BaseURL = c.Route.model, ``, ``, ``
	}
}
```

Call it at the four sites right after the struct literal. Stranger reservations (`tryReserveFor`, Task 3) still run for relay routes; the owner and server budgets are skipped: read `tryReserveFor`/`settleFor` in `models.go` and give them the route (or a `paysServer bool`) so a relay call reserves only against the stranger allowance when a stranger prompted it, and against nothing otherwise. Add a test that a relay call for the owner with the server budget exhausted still reserves successfully, and a stranger-prompted relay call still stops at the stranger cap.

- [ ] **Step 4: Run** `go test github.com/GoMudEngine/GoMud/modules/aicompanion/...`; expect PASS, including every existing test (tier 3 unchanged).

- [ ] **Step 5: Commit** (`feat(aicompanion): tiers, route selection and a per-owner relay breaker`), named paths, heredoc, trailer.

---

### Task 9: The relay transport and the key-shaped guard

**Files:**
- Create: `modules/aicompanion/relay.go`, `modules/aicompanion/relay_test.go`
- Modify: `modules/aicompanion/openai.go` (`callModelOnce`), `aicompanion.go` (install `companionai.SetRelayInbound` in `onLoad` after the `Enabled` check; remove it in `onUnload` if one exists)

- [ ] **Step 1: Failing tests** in `relay_test.go`:

```go
package aicompanion

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// fakeRelay captures what would go to the browser and lets a test answer.
type fakeRelay struct {
	sent []relayRequest
}

func (f *fakeRelay) send(userId int, module string, payload []byte) bool {
	var r relayRequest
	_ = json.Unmarshal(payload, &r)
	f.sent = append(f.sent, r)
	return true
}

func TestRelayRequestCarriesOnlyTheBody(t *testing.T) {
	p := newPendingRelays()
	f := &fakeRelay{}
	body := []byte(`{"model":"m","messages":[]}`)
	go func() {
		for len(f.sent) == 0 {
			time.Sleep(time.Millisecond)
		}
		p.deliver(5, relayResponse{Id: f.sent[0].Id, Status: 200, Body: `{"choices":[]}`})
	}()
	status, raw, err := p.do(context.Background(), 5, body, time.Second, f.send)
	if err != nil || status != 200 || string(raw) != `{"choices":[]}` {
		t.Fatalf("round trip: %d %q %v", status, raw, err)
	}
	wire, _ := json.Marshal(f.sent[0])
	for _, bad := range []string{`Authorization`, `Bearer`, `http://`, `https://`, `sk-`} {
		if strings.Contains(string(wire), bad) {
			t.Fatalf("a relay request must carry only an id and the body, found %q in %s", bad, wire)
		}
	}
}

func TestRelayRepliesMatchIdAndOwnerOnce(t *testing.T) {
	p := newPendingRelays()
	f := &fakeRelay{}
	done := make(chan error, 1)
	go func() {
		_, _, err := p.do(context.Background(), 5, []byte(`{}`), 200*time.Millisecond, f.send)
		done <- err
	}()
	for len(f.sent) == 0 {
		time.Sleep(time.Millisecond)
	}
	id := f.sent[0].Id
	if p.deliver(6, relayResponse{Id: id, Status: 200, Body: `{}`}) {
		t.Fatal("another player's reply must not be accepted")
	}
	if p.deliver(5, relayResponse{Id: `nope`, Status: 200, Body: `{}`}) {
		t.Fatal("an unknown id must not be accepted")
	}
	if !p.deliver(5, relayResponse{Id: id, Status: 200, Body: `{}`}) {
		t.Fatal("the owner's reply with the right id is accepted")
	}
	if p.deliver(5, relayResponse{Id: id, Status: 200, Body: `{}`}) {
		t.Fatal("a second reply to one id is dropped")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestRelayTimesOut(t *testing.T) {
	p := newPendingRelays()
	f := &fakeRelay{}
	_, _, err := p.do(context.Background(), 5, []byte(`{}`), 20*time.Millisecond, f.send)
	if err == nil {
		t.Fatal("no answer in time is a failed call")
	}
	if len(p.byId) != 0 {
		t.Fatal("a timed-out request leaves nothing pending")
	}
}

func TestKeyShapedRepliesAreRefused(t *testing.T) {
	for _, body := range []string{
		`{"x":"sk-proj-abc123def456ghi789"}`,
		`{"x":"sk-or-v1-0123456789abcdef"}`,
		`{"echo":"Authorization: Bearer abc"}`,
		`{"echo":"bearer abcdefghijklmnop"}`,
	} {
		if !looksLikeAKey([]byte(body)) {
			t.Errorf("must refuse %s", body)
		}
	}
	if looksLikeAKey([]byte(`{"choices":[{"message":{"content":"{\"speech\":[{\"kind\":\"say\",\"text\":\"I ask for nothing.\"}]}"}}]}`)) {
		t.Error("an ordinary reply must pass")
	}
}
```

Also a test in the same file that `callModelOnce` with `Route.kind == routeRelay` and an owner who has not consented returns `errNoConsent` and sends nothing (the choke point still guards the relay), and one that an oversized reply (over `1<<20`) is refused.

- [ ] **Step 2: Run; expect compile failure.**

- [ ] **Step 3: Implement `relay.go`:**

```go
package aicompanion

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sync"
	"time"
)

// relayRequest is all that goes to the owner's browser: an id and the chat
// completions body. No key, no URL, no header: the relay page adds its own.
type relayRequest struct {
	Id   string          `json:"id"`
	Body json.RawMessage `json:"body"`
}

// relayResponse is what comes back: the provider's status and raw body.
type relayResponse struct {
	Id     string `json:"id"`
	Status int    `json:"status"`
	Body   string `json:"body"`
}

type pendingRelay struct {
	owner int
	reply chan relayResponse
}

// pendingRelays matches replies to the calls waiting for them. Replies
// arrive on connection goroutines and waits run on model goroutines, so it
// has its own lock and never touches game state.
type pendingRelays struct {
	mu   sync.Mutex
	byId map[string]*pendingRelay
}

func newPendingRelays() *pendingRelays { return &pendingRelays{byId: map[string]*pendingRelay{}} }

var errRelayTimeout = errors.New(`the owner's browser did not answer in time`)
var errRelayUnsent = errors.New(`the owner has no browser to relay through`)
var errRelayKeyShaped = errors.New(`a relay reply looked like it carried a key; dropped`)

func newRelayId() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// do sends one request body to the owner's browser and waits for the reply.
func (p *pendingRelays) do(ctx context.Context, owner int, body []byte, timeout time.Duration,
	send func(userId int, module string, payload []byte) bool) (int, []byte, error) {

	id := newRelayId()
	w := &pendingRelay{owner: owner, reply: make(chan relayResponse, 1)}
	p.mu.Lock()
	p.byId[id] = w
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.byId, id)
		p.mu.Unlock()
	}()

	payload, err := json.Marshal(relayRequest{Id: id, Body: body})
	if err != nil {
		return 0, nil, err
	}
	if !send(owner, `Companion.Relay.Request`, payload) {
		return 0, nil, errRelayUnsent
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case r := <-w.reply:
		if len(r.Body) > 1<<20 {
			return 0, nil, errors.New(`relay reply too large`)
		}
		if looksLikeAKey([]byte(r.Body)) {
			return 0, nil, errRelayKeyShaped
		}
		return r.Status, []byte(r.Body), nil
	case <-timer.C:
		return 0, nil, errRelayTimeout
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	}
}

// deliver hands a reply to the call waiting for it. It accepts a reply only
// from the owner the request went to, only for an id still pending, and
// only once.
func (p *pendingRelays) deliver(from int, r relayResponse) bool {
	p.mu.Lock()
	w := p.byId[r.Id]
	if w == nil || w.owner != from {
		p.mu.Unlock()
		return false
	}
	delete(p.byId, r.Id)
	p.mu.Unlock()
	w.reply <- r
	return true
}

// keyShaped matches what an API key or an auth header looks like. A reply
// that contains one is a bug somewhere (the relay echoing its headers, or a
// provider echoing the request), and is dropped rather than parsed or logged.
var keyShaped = regexp.MustCompile(`(?i)(\bsk-[a-z0-9_-]{8,}|authorization\s*:|\bbearer\s+[a-z0-9._-]{8,})`)

func looksLikeAKey(b []byte) bool { return keyShaped.Match(b) }
```

Add `relayCalls *pendingRelays` to the module struct, initialised with the relay table. Add the inbound handler:

```go
// onRelayInbound receives Companion.Relay.* from a player's client. It runs
// on the connection goroutine and touches only the relay tables.
func (m *AICompanionModule) onRelayInbound(userId int, command string, payload []byte) {
	if !m.playerKeysOffered() {
		return
	}
	switch command {
	case `Companion.Relay.Ready`:
		var r struct {
			Model string `json:"model"`
		}
		if json.Unmarshal(payload, &r) == nil && r.Model != `` && !looksLikeAKey(payload) {
			m.relays.ready(userId, r.Model)
		}
	case `Companion.Relay.Gone`:
		m.relays.gone(userId)
	case `Companion.Relay.Response`:
		var r relayResponse
		if json.Unmarshal(payload, &r) == nil {
			m.relayCalls.deliver(userId, r)
		}
	}
}
```

Clear a player's relay on logout too: find the module's logout/`PlayerDespawn` listener and call `m.relays.gone(userId)` there (under the lock it already holds; `relays` has its own lock so this is safe).

In `callModelOnce`, after `body` is marshalled and the context is built, branch before the HTTP request:

```go
	if c.Route.kind == routeRelay {
		if !m.consent.allows(c.OwnerUserId) {
			m.consent.noteRefusal(c.OwnerUserId, `relay`)
			res.Err = errNoConsent
			return res
		}
		status, raw, err := m.relayCalls.do(ctx, c.OwnerUserId, body,
			time.Duration(m.cfg.RelayTimeoutSeconds)*time.Second, companionai.SendRelay)
		res.Latency = time.Since(start)
		res.Status = status
		if err != nil {
			res.Err = err
			m.relays.failure(c.OwnerUserId, time.Now(), m.cfg)
			return res
		}
		m.relays.success(c.OwnerUserId)
		return m.decodeChatResponse(res, status, raw)
	}
```

Refactor the existing status check, decode and choices handling below the HTTP read into `decodeChatResponse(res modelResult, status int, raw []byte) modelResult` so both paths share it (no second copy). Relay failures must not reach the global `breakerResult`: find where `breakerResult` is called on a result and skip it when the call's route was `routeRelay` (the per-owner breaker already counted it). Moderation (`moderate`, `moderateDecision`) runs only when the route is `routeServer`: for a relay call, skip it and leave `Moderated` at 0.

- [ ] **Step 4: Run tests; expect PASS.** Prove red: remove the `w.owner != from` check and see `TestRelayRepliesMatchIdAndOwnerOnce` fail; remove the `looksLikeAKey` call in `do` and see a key-shaped test through `do` fail (add that test); remove the consent check in the relay branch and see the consent test fail. Restore each.

- [ ] **Step 5: Commit** (`feat(aicompanion): relay transport through the owner's browser`), trailer.

---

### Task 10: Accountability, reflection, strangers toggle, status

**Files:**
- Modify: `modules/aicompanion/runtime.go` (`speak`, logout reflection at ~333), `reflect.go`, `commands.go` (`cmdAI`, admin status), `listeners.go` (stranger pacing gate), `meeting.go` or wherever bond records live (new `StrangersOff` field on the bond record)
- Modify: `modules/aicompanion/files/datafiles/templates/help/companion-ai.template`, `aicompanion.template`
- Test: `modules/aicompanion/tiers_test.go` (append)

- [ ] **Step 1: Failing tests.**
  - `TestMutedOwnerSilencesHer`: build a controller whose owner `UserRecord` has `Muted = true`; call the function that decides whether lines are spoken (extract `spokenLines(owner *users.UserRecord, lines []SpeechLine) []SpeechLine` from `speak` so it is testable without a live mob) and assert it returns nothing for say and emote alike; not muted returns them unchanged.
  - `TestRelaySpeechIsLoggedAgainstTheOwner`: `speechLogLine(route, ownerId, name, text)` returns a non-empty log attribute set for `routeRelay` containing the owner id, and nothing for `routeServer`.
  - `TestReflectionWaitsForTheRelay`: with a relay owner whose relay is gone at logout, `deferReflection` queues it; `relayCameBack(owner)` returns the queued reflection exactly once.
  - `TestStrangersOffStopsStrangerPrompts`: with the owner's `StrangersOff` set, `strangerMayAsk` returns false for any stranger without consuming their cooldown.

- [ ] **Step 2: Run; expect failures.**

- [ ] **Step 3: Implement.**
  - `speak`: at the top, `lines = spokenLines(owner, lines)`; return early when empty. Where each line is issued, when the controller's current route is `routeRelay`, log once per line: `mudlog.Info("aicompanion", "action", "speech", "owner", ownerId, "companion", c.profile.Name, "kind", l.Kind, "text", piece)`. Owners on `routeServer` are not logged (their lines passed moderation). If `speak` does not have the owner record, look it up with `users.GetByUserId(c.ownerUserId)`.
  - Reflection: at `runtime.go:333`, if `m.route(owner).kind == routeRelay` OR the owner's relay was live this session (track `c.lastRoute`), store `m.deferredReflect[owner] = reflectionArgs{...}` instead of starting it. In `onRelayInbound` `Ready`, do NOT start it there (connection goroutine); instead set a flag the module's round tick reads under the lock and starts the reflection from there. One deferred reflection per owner; a newer one replaces an older.
  - Strangers toggle: add `StrangersOff bool` to the bond record struct (find it: `bondRecord` in `meeting.go` or `aicompanion.go`; saved via `saveBonds`). `strangerMayAsk` returns false before touching the cooldown when the companion's owner has it set. `cmdAI` gains `strangers on|off`: "(Passers-by can talk to %s again, and their words are paid for from your key.)" / "(%s will hear passers-by but answer them only with a few set lines.)".
  - Status: `companion-ai` with no argument now also says which tier is live: "answering through your own key", "answering through this server's key", or "answering with a few set lines". Update the admin status line in `commands.go:~130` to print `tier=relay|server|none`.
  - Fallback notice: the first time in a session that a relay call fails for an owner (use `relayOwner.noticeSent`, reset on `ready`), tell the owner once, in plain words and never the raw error: "(%s falls back on a few set words: your key's provider did not answer.)". Test that a second failure in the same session sends nothing.
  - Help: `companion-ai.template` documents `companion-ai strangers on|off` and says when using your own key, strangers' talk is paid from your key. `aicompanion.template` gets a short "Your own key" section: set it up from the web client's Companion key button; the key stays in your browser and never reaches this server; put a spending limit on it; do not use a work computer.

- [ ] **Step 4: Run tests; expect PASS.** Prove red on the mute and strangers-off tests.

- [ ] **Step 5: Commit** (`feat(aicompanion): own-key accountability, deferred reflection, strangers toggle`), trailer.

---

### Task 11: The relay page (embedded, own origin)

**Files:**
- Create: `modules/aicompanion/relayweb/relay.html`, `modules/aicompanion/relayweb/relay.js`
- Create: `modules/aicompanion/relaypage.go`, `modules/aicompanion/relaypage_test.go`
- Create: `tools/jstest/companion-relay.test.js`
- Modify: `modules/aicompanion/aicompanion.go` (install `companionai.SetRelayPage` in `onLoad` when `playerKeysOffered()`)

- [ ] **Step 1: Failing Go test** `relaypage_test.go`:

```go
func TestRelayPageServedOnlyOnTheRelayHost(t *testing.T) {
	m := &AICompanionModule{cfg: Config{Enabled: true, PlayerKeys: true, RelayOrigin: `https://keys.example.org`}}
	req := httptest.NewRequest(http.MethodGet, `https://keys.example.org/companion-relay.html`, nil)
	rec := httptest.NewRecorder()
	if !m.serveRelayPage(rec, req) || rec.Code != 200 {
		t.Fatalf("the relay host serves the page, got %d", rec.Code)
	}
	csp := rec.Header().Get(`Content-Security-Policy`)
	for _, want := range []string{`script-src 'self'`, `connect-src https: http://localhost:* http://127.0.0.1:*`, `frame-ancestors https://`} {
		if !strings.Contains(csp, want) {
			t.Fatalf("CSP must contain %q: %s", want, csp)
		}
	}
	if strings.Contains(csp, `unsafe-inline`) {
		t.Fatal("the relay page allows no inline script")
	}
	other := httptest.NewRequest(http.MethodGet, `https://example.org/companion-relay.html`, nil)
	if m.serveRelayPage(httptest.NewRecorder(), other) {
		t.Fatal("the game host never serves the relay")
	}
	js := httptest.NewRequest(http.MethodGet, `https://keys.example.org/companion-relay.js`, nil)
	jrec := httptest.NewRecorder()
	if !m.serveRelayPage(jrec, js) || !strings.Contains(jrec.Header().Get(`Content-Type`), `javascript`) {
		t.Fatal("the relay script is served with a script type")
	}
}
```

The test must set `FilePaths.WebDomain` to `example.org` for `playerKeysOffered`; find how other tests override config (`configs.SetVal` or similar, per the testing skill) and restore it after.

- [ ] **Step 2: Implement `relaypage.go`:**

```go
package aicompanion

import (
	"bytes"
	_ "embed"
	"net/http"
	"net/url"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

//go:embed relayweb/relay.html
var relayHTML []byte

//go:embed relayweb/relay.js
var relayJS []byte

// serveRelayPage answers requests for the relay host and nothing else. The
// page is the only place a player's key ever exists; the game page frames
// it and cannot read inside it.
func (m *AICompanionModule) serveRelayPage(w http.ResponseWriter, r *http.Request) bool {
	if !m.playerKeysOffered() {
		return false
	}
	relay, _ := url.Parse(m.cfg.RelayOrigin)
	if !strings.EqualFold(stripPort(r.Host), relay.Hostname()) {
		return false
	}
	game := `https://` + strings.TrimSpace(configs.GetFilePathsConfig().WebDomain.String())
	w.Header().Set(`Content-Security-Policy`, `default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; `+
		`connect-src https: http://localhost:* http://127.0.0.1:*; frame-ancestors `+game)
	w.Header().Set(`Referrer-Policy`, `no-referrer`)
	w.Header().Set(`X-Content-Type-Options`, `nosniff`)
	switch r.URL.Path {
	case `/companion-relay.html`:
		w.Header().Set(`Content-Type`, `text/html; charset=utf-8`)
		_, _ = w.Write(bytes.ReplaceAll(relayHTML, []byte(`{{GAME_ORIGIN}}`), []byte(game)))
	case `/companion-relay.js`:
		w.Header().Set(`Content-Type`, `text/javascript; charset=utf-8`)
		_, _ = w.Write(relayJS)
	default:
		http.NotFound(w, r)
	}
	return true
}

func stripPort(host string) string {
	if i := strings.LastIndex(host, `:`); i > 0 && !strings.Contains(host[i:], `]`) {
		return host[:i]
	}
	return host
}
```

`WebDomain` must be a bare host; if it can contain a scheme or port in shipped config, normalise it (read `config.filepaths.go` and `config.yaml`).

- [ ] **Step 3: `relayweb/relay.html`** (no inline script; `{{GAME_ORIGIN}}` is replaced by the server):

```html
<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="game-origin" content="{{GAME_ORIGIN}}">
<title>Companion key</title>
<style>
 body{font:14px system-ui,sans-serif;margin:0;padding:12px;background:#111;color:#ddd}
 label{display:block;margin:8px 0 2px} input{width:100%;box-sizing:border-box}
 .warn{color:#e8b34a;font-size:12px} .row{display:flex;gap:8px;margin-top:10px}
 [hidden]{display:none}
</style>
</head>
<body>
<form id="setup" hidden autocomplete="off">
 <p class="warn">Your key stays in this browser and is never sent to the game server.
 Put a monthly spending limit on it. Do not use a work computer.
 When a passer-by talks to your companion, your key pays for that too.</p>
 <label for="endpoint">Endpoint</label>
 <input id="endpoint" required value="https://api.openai.com/v1">
 <label for="key">Key</label>
 <input id="key" type="password" required>
 <label for="model">Model</label>
 <input id="model" required value="gpt-4.1-mini">
 <label><input id="remember" type="checkbox"> Remember this key on this device</label>
 <label for="pass" id="passlabel" hidden>Passphrase (needed to unlock it next time)</label>
 <input id="pass" type="password" hidden>
 <div class="row"><button type="submit">Use this key</button>
 <button type="button" id="forget">Forget my key</button>
 <button type="button" id="close">Close</button></div>
 <p class="warn" id="status"></p>
</form>
<form id="unlock" hidden autocomplete="off">
 <label for="unlockpass">Passphrase to unlock your saved key</label>
 <input id="unlockpass" type="password">
 <div class="row"><button type="submit">Unlock</button> <button type="button" id="unlockforget">Forget it</button></div>
 <p class="warn" id="unlockstatus"></p>
</form>
<script src="/companion-relay.js"></script>
</body>
</html>
```

- [ ] **Step 4: `relayweb/relay.js`.** Written as a UMD-style module so node tests can `require` the pure parts (match `hotinput.js`'s export pattern: read it). Pure core (exported for tests):

```js
(function (root, factory) {
  var api = factory();
  if (typeof module === 'object' && module.exports) { module.exports = api; }
  else { root.CompanionRelay = api; api.boot(window, document); }
})(typeof self !== 'undefined' ? self : this, function () {
  'use strict';
  var ITER = 600000;

  function isAllowedEndpoint(u) {
    try {
      var p = new URL(u);
      if (p.protocol === 'https:') { return true; }
      return p.protocol === 'http:' && (p.hostname === 'localhost' || p.hostname === '127.0.0.1');
    } catch (e) { return false; }
  }

  function storageKey(account) { return 'companion-key:' + String(account || '').toLowerCase(); }

  async function deriveKey(subtle, pass, salt) {
    var base = await subtle.importKey('raw', new TextEncoder().encode(pass), 'PBKDF2', false, ['deriveKey']);
    return subtle.deriveKey({ name: 'PBKDF2', salt: salt, iterations: ITER, hash: 'SHA-256' },
      base, { name: 'AES-GCM', length: 256 }, false, ['encrypt', 'decrypt']);
  }

  function b64(u8) { var s = ''; for (var i = 0; i < u8.length; i++) { s += String.fromCharCode(u8[i]); } return btoa(s); }
  function unb64(s) { var b = atob(s), u = new Uint8Array(b.length); for (var i = 0; i < b.length; i++) { u[i] = b.charCodeAt(i); } return u; }

  async function seal(cryptoObj, pass, secret) {
    var salt = cryptoObj.getRandomValues(new Uint8Array(16));
    var iv = cryptoObj.getRandomValues(new Uint8Array(12));
    var k = await deriveKey(cryptoObj.subtle, pass, salt);
    var ct = await cryptoObj.subtle.encrypt({ name: 'AES-GCM', iv: iv }, k, new TextEncoder().encode(JSON.stringify(secret)));
    return JSON.stringify({ v: 1, salt: b64(salt), iv: b64(iv), ct: b64(new Uint8Array(ct)) });
  }

  async function unseal(cryptoObj, pass, sealed) {
    var o = JSON.parse(sealed);
    var k = await deriveKey(cryptoObj.subtle, pass, unb64(o.salt));
    var pt = await cryptoObj.subtle.decrypt({ name: 'AES-GCM', iv: unb64(o.iv) }, k, unb64(o.ct));
    return JSON.parse(new TextDecoder().decode(pt));
  }

  // relayOne posts one request body to the stored endpoint and returns only
  // {id, status, body}. The endpoint comes from the player's settings, never
  // from the message; the message cannot name a URL or a header.
  async function relayOne(fetchFn, settings, msg) {
    if (!settings || !settings.key || !isAllowedEndpoint(settings.endpoint)) {
      return { id: msg.id, status: 0, body: '' };
    }
    var url = settings.endpoint.replace(/\/+$/, '') + '/chat/completions';
    var body = msg.body;
    if (typeof body !== 'string') { body = JSON.stringify(body); }
    try {
      var r = await fetchFn(url, {
        method: 'POST', credentials: 'omit', referrerPolicy: 'no-referrer',
        headers: { 'Content-Type': 'application/json', 'Authorization': 'Bearer ' + settings.key },
        body: body
      });
      return { id: msg.id, status: r.status, body: await r.text() };
    } catch (e) {
      return { id: msg.id, status: 0, body: '' };
    }
  }

  function acceptMessage(ev, gameOrigin) {
    return !!ev && ev.origin === gameOrigin && ev.data && typeof ev.data === 'object';
  }

  function boot(win, doc) { /* DOM wiring: see Step 5 */ }

  return { isAllowedEndpoint: isAllowedEndpoint, storageKey: storageKey, seal: seal, unseal: unseal,
           relayOne: relayOne, acceptMessage: acceptMessage, boot: boot };
});
```

- [ ] **Step 5: `boot(win, doc)`** implements, in this order:
  1. If `!win.isSecureContext`, do nothing (no listeners, no UI).
  2. `gameOrigin` from `<meta name="game-origin">`.
  3. State: `settings` (in memory only), `account` (set by the `hello` message).
  4. `win.addEventListener('message', ...)` that ignores any event failing `acceptMessage(ev, gameOrigin)`, and handles `ev.data.type`:
     - `hello {account}`: remember `account`; if `localStorage[storageKey(account)]` exists show the unlock form, else post `{type:'status', ready:false}`.
     - `setup`: show the setup form (the glue makes the iframe visible).
     - `request {id, body}`: if `settings` is null post `{type:'response', id, status:0, body:''}`; else `relayOne(win.fetch.bind(win), settings, ev.data)` then post `{type:'response', ...}`.
     All posts go to `win.parent.postMessage(x, gameOrigin)`, never `'*'`.
  5. Setup submit: validate `isAllowedEndpoint`; keep `settings = {endpoint, key, model}`; if remember is ticked require a passphrase of at least 8 characters, `seal` `{endpoint,key,model}` into `localStorage[storageKey(account)]` (wrapped in try/catch); clear the key and passphrase inputs; post `{type:'status', ready:true, model}` and `{type:'hide'}`.
  6. Unlock submit: `unseal` with the passphrase; on failure show "That passphrase did not open it." and stay; on success set `settings`, clear the input, post ready.
  7. Forget: remove `localStorage[storageKey(account)]`, set `settings = null`, post `{type:'status', ready:false}`.
  8. Never log `settings`, never put the key in any posted message.

- [ ] **Step 6: `tools/jstest/companion-relay.test.js`** (dependency-free; copy the runner shape from `hotinput.test.js`), requiring `modules/aicompanion/relayweb/relay.js`. Cases:
  - `isAllowedEndpoint`: `https://api.openai.com/v1` true; `http://localhost:11434/v1` true; `http://127.0.0.1:1234/v1` true; `http://evil.example/v1` false; `javascript:alert(1)` false; `file:///x` false.
  - `relayOne` with a fake `fetch` records the URL and headers: URL is the STORED endpoint plus `/chat/completions` even when the message carries `url`/`endpoint`/`headers` fields; `Authorization` is set only on that call; `credentials` is `omit`; the returned object has exactly the keys `id,status,body`.
  - `relayOne` with no settings returns status 0 and never calls fetch.
  - `acceptMessage` false for another origin, true for the game origin.
  - `seal`/`unseal` round trip with `globalThis.crypto` (node 20); wrong passphrase rejects; the sealed string does not contain the key text.
  - `storageKey('Alice') !== storageKey('Bob')` and is case-insensitive per account.
  Run: `node tools/jstest/companion-relay.test.js`; expect all pass. Prove red: make `relayOne` use `msg.url` when present and see the stored-endpoint case fail; restore.

- [ ] **Step 7:** Check `.jshintignore` and the CI syntax job loop (`validate.yml`): ensure `modules/aicompanion/relayweb/relay.js` is covered by the syntax check if the job only scans `_datafiles/html`; if it is not, extend the job's file list in `validate.yml` to include it (this workflow file change is in scope; keep it minimal).

- [ ] **Step 8: Gates and commit** (`feat(aicompanion): the key relay page on its own origin`), trailer. `go test` the module; `node tools/jstest/companion-relay.test.js`.

---

### Task 12: Game-page glue, setup button and the innerHTML audit

**Files:**
- Create: `_datafiles/html/public/static/js/companion-relay-glue.js`, `tools/jstest/companion-relay-glue.test.js`, `tools/jstest/webclient-html-escape.test.js`
- Modify: `_datafiles/html/public/webclient-pure.html`
- Modify: whatever template data supplies the relay origin to the page (see Step 2)

- [ ] **Step 1: Failing glue test** `companion-relay-glue.test.js`, requiring the glue as a UMD module (same export pattern as `relay.js`). The glue's pure core is `createGlue({relayOrigin, sendGMCP, postToFrame})` returning `{onGMCPRequest(obj), onFrameMessage(ev), hello(account)}`. Cases:
  - `onGMCPRequest({id, body})` calls `postToFrame({type:'request', id, body}, relayOrigin)` with exactly those fields.
  - `onFrameMessage` ignores events whose `origin !== relayOrigin`.
  - A frame `response` becomes `sendGMCP('Companion.Relay.Response', {id, status, body})` with no other fields, even if the frame message carries extra fields such as `key`.
  - A frame `status {ready:true, model}` becomes `sendGMCP('Companion.Relay.Ready', {model})`; `ready:false` becomes `sendGMCP('Companion.Relay.Gone', {})`.
  - Nothing the glue sends ever contains a field named `key`, `endpoint`, `authorization` (case-insensitive).

- [ ] **Step 2: Implement the glue.** `createGlue` as above plus a `boot()` that runs only in a browser when `window.COMPANION_RELAY_ORIGIN` is a non-empty string: creates a hidden `<iframe src=relayOrigin + "/companion-relay.html" sandbox="allow-scripts allow-same-origin allow-forms" referrerpolicy="no-referrer">`, registers `message` listening, sends `hello` with the logged-in account name once the page knows it, and exposes `CompanionRelayGlue.openSetup()` which makes the iframe visible in a fixed overlay and posts `{type:'setup'}`; on `{type:'hide'}` it hides the iframe again. The account name comes from wherever the web client already knows the logged-in character (find it in `webclient-pure.html`: search `Char.Login`/`GMCPStructs.Char.Info` or similar); send `hello` when that becomes known.

  Supply `window.COMPANION_RELAY_ORIGIN` from the server. `internal/web/web.go` uses `text/template` (line 16), which does NOT escape, so encode in Go: in `serveTemplate` add `"COMPANION_RELAY_ORIGIN_JSON": relayOriginJSON()` to `templateData`, where `relayOriginJSON` returns `json.Marshal(companionai.RelayOrigin())` as a string (a JSON string literal, `""` when empty). In `webclient-pure.html` add `<script>window.COMPANION_RELAY_ORIGIN = {{ .COMPANION_RELAY_ORIGIN_JSON }};</script>` next to the existing `window.ITEM_ICON_BASE` line (243), with no quotes around the action. Add a Go test that an origin containing `"</script>` would be emitted escaped by `relayOriginJSON` (json.Marshal escapes `<`, `>` and `"`).

- [ ] **Step 3: Web client wiring** in `webclient-pure.html`:
  - Include `<script src="{{ .CONFIG.FilePaths.WebCDNLocation }}/static/js/companion-relay-glue.js"></script>` after `gmcp.js` (line 240).
  - Add a `GMCPUpdateHandlers["Companion.Relay.Request"]` handler that reads `GMCPStructs.Companion.Relay.Request` and calls `CompanionRelayGlue.onGMCPRequest(obj)`. Check the dispatcher at ~2040-2075: a dotted push replaces the sub-object wholesale, so each request is seen once; confirm and note it.
  - If Task 7 found that `GMCPOut` needs the module enabled via `Core.Supports.Set`, add `"Companion 1"` to the client's supports list (find where it sends `Core.Supports.Set`).
  - A "Companion key" button in the dashboard settings area (find an existing settings or menu button and follow its markup and XSS-safe construction) that calls `CompanionRelayGlue.openSetup()`; hidden when `COMPANION_RELAY_ORIGIN` is empty AND until the player is logged in (spec D6: key setup only while logged in); the glue also refuses `openSetup()` before `hello` has been sent with an account.

- [ ] **Step 4: innerHTML audit.** Read `webclient-pure.html:940-960` and `:2170-2185`. For each, trace every value interpolated into `html` back to its source. If any GMCP-supplied or player-authored string reaches it unescaped, rebuild that fragment with DOM calls (`createElement`/`textContent`), matching the file's own "never innerHTML" pattern at 545 and 1173. If every value is escaped (via `escapeHTML` at 2187) or constant, keep it and add a one-line comment naming the escaping. Add `tools/jstest/webclient-html-escape.test.js`: extract `escapeHTML` from the page source (read the file, pull the function text with a regex, `new Function` it; follow `safe-dom.test.js`'s approach if it already does this) and assert `<img src=x onerror=1>` and `"'&` are escaped.

- [ ] **Step 5: Run** `node tools/jstest/companion-relay-glue.test.js` and `node tools/jstest/webclient-html-escape.test.js`; PASS. Prove red on the field-stripping case (let an extra field through; see it fail; restore).

- [ ] **Step 6: Commit** (`feat(webclient): companion key relay glue and setup button`), trailer.

---

### Task 13: Docs, leftovers, gates, boot, PR (replaces Part 1 Task 6)

**Files:** many docs; `internal/actions/cast.go`; `modules/aicompanion/harm.go`; `_datafiles/config.yaml`; `docs/PATCH_NOTES.md`; `docs/audits/2026-09-23-ai-companion-pr161-review.md`; `docs/aicompanion/settings.md`, `testing-and-prompts.md`, `upstreaming.md`; `modules/aicompanion/context.md`, `internal/companionai/context.md`, `modules/gmcp/context.md`, `internal/web/context.md` (if it exists).

- [ ] **Step 1: Engine leftover from Task 5.** In `internal/actions/cast.go`, the no-target fallbacks of HarmSingle and HarmMulti add a player from the caster's or the party leader's current fight without `CanPvp`. Add the same `CanPvp` check the named-target branch uses, skipping (not refusing the cast) a player who fails it. Regression test next to `cast_pvp_multi_test.go`, proven red.
- [ ] **Step 2:** Fix `modules/aicompanion/harm.go`'s header comment: `cast.go` makes no party check for players; say what the helper itself checks.
- [ ] **Step 3: `_datafiles/config.yaml`** under `Modules.aicompanion`: add `PlayerKeys: true`, `RelayOrigin: ""`, `RelayTimeoutSeconds: 30`, each with a comment: player keys need `RelayOrigin` set to an https origin on its own subdomain (DNS plus a proxy route to this server), otherwise nothing is offered. Confirm first with `git ls-files -v _datafiles/config.yaml` that no skip-worktree bit is set in this worktree.
- [ ] **Step 4: Docs.** `docs/aicompanion/settings.md` (three new settings), `testing-and-prompts.md` (tier 2: what goes to the browser, what never does, no moderation, owner accountability), `upstreaming.md` (the tiers and the deploy step), every `context.md` touched (verify each symbol named exists: `grep -n` it), the Dependencies list in `modules/aicompanion/context.md` (Task 4 found about ten missing imports: list them from the package's actual import set).
- [ ] **Step 5: Review doc.** In `docs/audits/2026-09-23-ai-companion-pr161-review.md`, amend decision 2 in its Decisions table: the merge no longer waited for lighting plan 6 (owner, 2026-09-24); PR #161 merged 2026-09-25 as `d4a23b47a`, and this follow-up carries the fixes.
- [ ] **Step 6: `docs/PATCH_NOTES.md`**: a dated entry in player-facing terms: companions can run on your own key from the web client (it stays in your browser), `companion-ai strangers`, companions answer "i agree" again, companions refuse to harm what you could not. No raw numbers.
- [ ] **Step 7: Index.** Add any new doc to `docs/README.md`.
- [ ] **Step 8: Gates.**

```bash
gofmt -l internal/ modules/
go build ./...
go test ./...
~/go/bin/golangci-lint run --new-from-merge-base=origin/master
for t in tools/jstest/*.test.js; do node "$t" || echo "FAIL $t"; done
```

Expected: gofmt empty; `go test ./...` all ok (the full suite; `internal/combat` takes minutes); lint `0 issues`; every jstest passes.

- [ ] **Step 9: Boot check, module OFF and ON.** Use a NEW detached worktree at `C:/tmp/dogmud-pr161-boot` (NOT `C:/tmp/dogmud-boot-check`). Copy `_datafiles/config.yaml` in. Pick free ports (check with PowerShell `Get-NetTCPConnection -State Listen` first) and override them with a `CONFIG_PATH` overlay as `reference_boot_test_in_isolated_worktree` describes. Build to `boot-check.exe`, run with a 180s timeout, record the PID you started, and stop ONLY that PID. Run 1: module off. Run 2: `Modules.aicompanion.Enabled: true`, `PlayerKeys: true`, `RelayOrigin: https://keys.localtest.me` (or any host other than WebDomain), no server key. For each: zero `^panic:|goroutine [0-9]+ \[running\]|runtime error`, one `Server Ready`, no `no listener for event` lines. In run 2 also `curl -H "Host: keys.localtest.me" http://127.0.0.1:<webport>/companion-relay.html` returns the page with the CSP header, and the same path with the normal host returns 404. Remove the worktree after (PowerShell `Remove-Item -Recurse -Force`, then `git worktree prune`).
- [ ] **Step 10: Owner-assisted browser check (NOT a subagent step).** List for the owner: open the web client locally with the relay origin reachable, enter a capped OpenRouter or OpenAI key, confirm a companion answers, confirm the Network tab shows the key only on the provider request and never on the websocket. The main session runs this with the owner.
- [ ] **Step 11: PR.** Push `fix/aicompanion-integration` and open the PR with `gh pr create --repo pruuk/DOGMud --base master --head fix/aicompanion-integration`; read the printed URL and confirm it says `pruuk/DOGMud`. The body ends with the Claude Code line. Watch checks; the lint job may invert on size (check the file count and line count first; the log signals are in `dogmud-shipping`). The main session does this step, not a subagent.
