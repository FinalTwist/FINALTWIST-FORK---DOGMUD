// Package baubles is the model side of bauble loot
// (docs/baubles/implementation-plan.md, Phase 4). The engine side is
// internal/baubles; this module only installs the namer.
//
// Every call goes through internal/apiframework, the one mechanism the AI
// companion uses too: the same transport, the server key's one daily budget
// and one breaker, and the companion's key relay for a finder who allowed
// their own key to name their finds.
//
// Off by default. Switched off, it installs nothing and every bauble is a
// generic trinket. Switched on, a find is named through the finder's own
// key when they allowed it, else the server's key, else it is a generic
// trinket.
package baubles

import (
	"strconv"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/apiframework"
	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
)

// BaublesModule is the module's state. cfg is set once at load, on the game
// loop, and read by delivery goroutines behind mu.
type BaublesModule struct {
	plug *plugins.Plugin

	mu    sync.Mutex
	cfg   Config
	slots chan struct{}

	// finders holds the users with a find in flight on their own key: one
	// at a time each, and never one of the server's slots (spec S3).
	finders map[int]bool

	// Today's calls by route, since boot, for `bauble status`.
	stats struct {
		day      string
		server   int
		player   int
		failures int
	}
}

var module BaublesModule

func init() {
	module = BaublesModule{
		plug: plugins.New(`baubles`, `0.2.0`),
	}
	module.plug.Callbacks.SetOnLoad(module.onLoad)
	module.plug.Callbacks.SetOnSave(module.onSave)
}

func (m *BaublesModule) onLoad() {
	m.configure(buildConfig(func(k string) any { return m.plug.Config.Get(k) }))
	events.RegisterListener(events.NewRound{}, m.onNewRound)
}

// onNewRound re-reads the server key's settings on the game loop, where the
// config is written: a bauble is named off the game loop and must only ever
// read apiframework.Server()'s snapshot (see apiframework.RefreshServer).
func (m *BaublesModule) onNewRound(e events.Event) events.ListenerReturn {
	if m.snapshot().Enabled {
		apiframework.RefreshServer()
	}
	return events.Continue
}

// configure applies a config and installs (or removes) the namer.
func (m *BaublesModule) configure(cfg Config) {
	m.mu.Lock()
	m.cfg = cfg
	m.slots = make(chan struct{}, cfg.MaxConcurrent)
	m.mu.Unlock()

	// The prompt preview (admin `bauble prompt`) is installed whatever the
	// switch: an admin may want to read the prompt before turning it on.
	baubles.SetPromptPreview(previewMessages)

	if !cfg.Enabled {
		baubles.SetGenerator(nil, nil)
		mudlog.Info(`baubles`, `naming`, `generic trinkets`, `reason`, `Modules.baubles.Enabled is false`)
		return
	}
	baubles.SetGenerator(m.generate, m.info)
	s := apiframework.RefreshServer() // at load, on the game loop
	mudlog.Info(`baubles`, `naming`, `model`, `model`, cfg.Model, `serverKey`, s.HasKey(),
		`playerKeys`, cfg.UsePlayerKeys, `moderated`, cfg.ModerateOutput)
}

// snapshot is the config for one call.
func (m *BaublesModule) snapshot() Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg
}

// takeServerSlot takes one of the server key's MaxConcurrent slots, or
// reports none free. A find beyond them is not queued: it is a generic
// trinket.
func (m *BaublesModule) takeServerSlot() (release func(), ok bool) {
	m.mu.Lock()
	slots := m.slots
	m.mu.Unlock()
	select {
	case slots <- struct{}{}:
		return func() { <-slots }, true
	default:
		return nil, false
	}
}

// takeFinderSlot takes the finder's own slot: one call on their key at a
// time.
func (m *BaublesModule) takeFinderSlot(userId int) (release func(), ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.finders[userId] {
		return nil, false
	}
	if m.finders == nil {
		m.finders = map[int]bool{}
	}
	m.finders[userId] = true
	return func() {
		m.mu.Lock()
		delete(m.finders, userId)
		m.mu.Unlock()
	}, true
}

// count notes one call's route and outcome for the status view.
func (m *BaublesModule) count(playerKey bool, failed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d := time.Now().UTC().Format(`2006-01-02`); d != m.stats.day {
		m.stats.day, m.stats.server, m.stats.player, m.stats.failures = d, 0, 0, 0
	}
	if playerKey {
		m.stats.player++
	} else {
		m.stats.server++
	}
	if failed {
		m.stats.failures++
	}
}

// info is what `bauble status` shows about the namer.
func (m *BaublesModule) info() baubles.GeneratorInfo {
	m.mu.Lock()
	cfg := m.cfg
	server, player, failures := m.stats.server, m.stats.player, m.stats.failures
	m.mu.Unlock()

	s := apiframework.RefreshServer() // `bauble status`: on the game loop
	u := apiframework.Today()
	mine := 0
	for _, c := range u.ByConsumer {
		if c.Consumer == apiframework.ConsumerBaubles {
			mine = c.Tokens
		}
	}
	detail := `Today: ` + itoa(server) + ` named on the server's key, ` + itoa(player) + ` on finders' own keys, ` + itoa(failures) + ` failed. ` +
		`Server key tokens: ` + itoa(u.Tokens) + ` of ` + limitWords(u.Limit) + ` (baubles ` + itoa(mine) + `; one budget for every feature).`
	if !s.HasKey() {
		detail += ` No server key: finds named on a finder's own key are shown to that finder alone (nothing can moderate them); every other find is a generic trinket.`
	} else if !cfg.ModerateOutput {
		detail += ` ModerateOutput is off: server-key finds are shown to everyone with no check (the operator's choice)`
		if cfg.UsePlayerKeys {
			detail += `, and finds named on finders' own keys are shown to those finders alone`
		}
		detail += `.`
	}
	if now := time.Now(); cfg.ModerateOutput && apiframework.Blocked(moderationBreaker, now) && !apiframework.BreakerOpen(now) {
		detail += ` The moderation breaker is open (the moderation endpoint keeps failing): server-key finds are generic and finders' own keys' finds are shown to those finders alone until ` +
			apiframework.BreakerUntil(moderationBreaker).Format(`15:04:05`) + `.`
	}
	if now := time.Now(); apiframework.Blocked(apiframework.ConsumerBaubles, now) {
		if apiframework.BreakerOpen(now) {
			detail += ` The provider breaker (every feature's) is open`
		} else {
			detail += ` Baubles' naming breaker is open (the provider is fine; check Modules.baubles.Model)`
		}
		if until := apiframework.BreakerUntil(apiframework.ConsumerBaubles); now.Before(until) {
			detail += ` until ` + until.Format(`15:04:05`)
		} else {
			detail += `, a probe call is out`
		}
		detail += `.`
	}
	if !cfg.UsePlayerKeys {
		detail += ` Finders' own keys are not used (UsePlayerKeys false).`
	}
	return baubles.GeneratorInfo{Name: `openai`, Model: cfg.Model, Detail: detail}
}

// onSave writes the shared day's spending (apiframework keeps it).
func (m *BaublesModule) onSave() error {
	apiframework.SaveBudget()
	return nil
}

// limitWords is the day's budget as `bauble status` shows it.
func limitWords(limit int) string {
	if limit <= 0 {
		return `unlimited`
	}
	return itoa(limit)
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
