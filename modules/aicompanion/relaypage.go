package aicompanion

import (
	"bytes"
	_ "embed"
	"html"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/companionai"
	"github.com/GoMudEngine/GoMud/internal/configs"
)

// The relay origin (RelayOrigin) is the only place a player's key ever
// exists. It serves two pages: the relay frame, which the game page frames
// but cannot read into and which holds the key and relays requests, and the
// key window, a top-level popup the frame opens on this same origin, where
// the key is typed. Both talk only by postMessage.

//go:embed relayweb/relay.html
var relayHTML []byte

//go:embed relayweb/relay-setup.html
var relaySetupHTML []byte

//go:embed relayweb/relay.js
var relayJS []byte

// relayCSP is the relay origin's policy. No inline or eval script at all:
// relay.js is its only script. Styles may be inline (each page's one style
// block). It may call any https endpoint and a model on this machine.
// ancestors is who may frame the page: the game's own origin for the relay
// frame, nobody for the key window, which must be a top-level window so its
// address bar is the player's proof of where the key is going.
func relayCSP(ancestors string) string {
	return `default-src 'none'; script-src 'self'; style-src 'unsafe-inline'; ` +
		`connect-src https: http://localhost:* http://127.0.0.1:*; ` +
		`base-uri 'none'; form-action 'none'; frame-ancestors ` + ancestors
}

// installRelayPage puts the relay page on the engine's web seam while
// player keys are offered, and takes it off otherwise.
func (m *AICompanionModule) installRelayPage() {
	if !m.playerKeysOffered() {
		companionai.SetRelayPage(nil)
		return
	}
	origin := m.cfg.RelayOrigin
	companionai.SetRelayPage(m.serveRelayPage, func() string { return origin })
}

// serveRelayPage answers every request for the relay host and nothing
// else. Every path but the two pages and their script is refused rather
// than passed on, so no game page, websocket or admin route ever runs on
// the origin that holds the key.
func (m *AICompanionModule) serveRelayPage(w http.ResponseWriter, r *http.Request) bool {
	if !m.playerKeysOffered() {
		return false
	}
	relay, err := url.Parse(m.cfg.RelayOrigin)
	if err != nil || !strings.EqualFold(requestHostname(r.Host), relay.Hostname()) {
		return false
	}

	game := gameOrigin(string(configs.GetFilePathsConfig().WebDomain))
	if game == `` {
		// No game origin to trust means nothing may frame the page and no
		// message could be accepted: serve nothing.
		http.NotFound(w, r)
		return true
	}

	h := w.Header()
	h.Set(`Referrer-Policy`, `no-referrer`)
	h.Set(`X-Content-Type-Options`, `nosniff`)
	h.Set(`Cross-Origin-Resource-Policy`, `same-origin`)
	h.Set(`Cache-Control`, `no-store`)

	var body []byte
	switch r.URL.Path {
	case `/companion-relay.html`:
		h.Set(`Content-Security-Policy`, relayCSP(game))
		h.Set(`Content-Type`, `text/html; charset=utf-8`)
		body = renderRelayHTML(game)
	case `/companion-relay-setup.html`:
		h.Set(`Content-Security-Policy`, relayCSP(`'none'`))
		h.Set(`Content-Type`, `text/html; charset=utf-8`)
		body = relaySetupHTML
	case `/companion-relay.js`:
		h.Set(`Content-Security-Policy`, relayCSP(`'none'`))
		h.Set(`Content-Type`, `text/javascript; charset=utf-8`)
		body = relayJS
	default:
		http.NotFound(w, r)
		return true
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		h.Set(`Allow`, `GET, HEAD`)
		http.Error(w, `method not allowed`, http.StatusMethodNotAllowed)
		return true
	}
	if r.Method == http.MethodGet {
		_, _ = w.Write(body)
	}
	return true
}

// renderRelayHTML puts the game origin into the page's meta tag. The value
// comes from config, so it is HTML-escaped like any other untrusted text.
func renderRelayHTML(game string) []byte {
	return bytes.ReplaceAll(relayHTML, []byte(`{{GAME_ORIGIN}}`), []byte(html.EscapeString(game)))
}

// gameOrigin turns FilePaths.WebDomain into the origin the game page is
// served from, "https://host[:port]". WebDomain is documented as a bare
// host ("Do not include the protocol"), but a pasted scheme or trailing
// path is tolerated and dropped. The game is always https here: the relay
// runs only in a secure context and a secure frame needs a secure parent.
// Anything that is not a plain host (user info, spaces, markup) gives "",
// which serves no relay at all.
func gameOrigin(webDomain string) string {
	d := strings.ToLower(strings.TrimSpace(webDomain))
	if i := strings.Index(d, `://`); i >= 0 {
		d = d[i+3:]
	}
	if i := strings.IndexAny(d, `/?#`); i >= 0 {
		d = d[:i]
	}
	if d == `` || strings.ContainsAny(d, "@ \t\r\n\"'<>\\") {
		return ``
	}
	u, err := url.Parse(`https://` + d)
	if err != nil || u.Host != d || u.User != nil || u.Hostname() == `` {
		return ``
	}
	return `https://` + d
}

// requestHostname is a request's Host without its port or IPv6 brackets.
func requestHostname(host string) string {
	host = strings.TrimSpace(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.TrimSuffix(strings.TrimPrefix(host, `[`), `]`)
}
