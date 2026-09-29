package apiframework

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// DefaultBaseURL is the official endpoint, used when the configured one is
// refused by EndpointAllowed.
const DefaultBaseURL = `https://api.openai.com/v1`

// EndpointAllowed keeps the key and whatever is sent going where they are
// meant to: https, and exactly api.openai.com or an Azure OpenAI resource
// (*.openai.azure.com), unless the operator has deliberately allowed
// another provider (AllowCustomEndpoint, hard-locked).
func EndpointAllowed(raw string, allowCustom bool) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == `` {
		return false
	}
	if u.Scheme != `https` {
		return false
	}
	if allowCustom {
		return true
	}
	// Exactly OpenAI's API host, or an Azure OpenAI resource
	// (<resource>.openai.azure.com). Any other openai.com or azure.com host
	// is not where the key belongs. Azure's newer AI Services hosts
	// (*.cognitiveservices.azure.com, *.services.ai.azure.com) are refused
	// too; an operator who uses one sets AllowCustomEndpoint, which is
	// hard-locked, so only config.yaml can.
	host := strings.ToLower(u.Hostname())
	return host == `api.openai.com` || strings.HasSuffix(host, `.openai.azure.com`)
}

// DefaultKeyEnv is the environment variable read for the server's key when
// no config names one: the variable OpenAI's own tools read, and the one the
// AI companion always defaulted to.
const DefaultKeyEnv = `OPENAI_API_KEY`

// Defaults for a server whose config sets these nowhere, the AI companion's
// own defaults before the framework existed.
const (
	DefaultDailyTokenBudget = 2000000
	DefaultBreakerErrors    = 5
	DefaultBreakerSeconds   = 60
)

// ResolveKey returns a key: the environment variable envName when it is
// named and set, otherwise configured. It reads only the variable it is
// given (Server supplies the default name). The result is trimmed and must
// never be logged or printed, in a test failure message included.
func ResolveKey(envName string, configured string) string {
	if name := strings.TrimSpace(envName); name != `` {
		if k := strings.TrimSpace(os.Getenv(name)); k != `` {
			return k
		}
	}
	return strings.TrimSpace(configured)
}

// ServerSettings is the server's key and what bounds it, resolved from the
// APIFramework config section.
type ServerSettings struct {
	Endpoint         Endpoint
	RejectedBaseURL  string // a BaseURL refused by EndpointAllowed; the official one is used
	DailyTokenBudget int    // 0 is no cap
	BreakerErrors    int
	BreakerSeconds   int
	// Legacy names the settings read from the old place,
	// Modules.aicompanion, because APIFramework does not set them. A server
	// whose config.yaml predates the section keeps working exactly as
	// before; the companion logs these once at load so they can be moved.
	Legacy []string
}

// HasKey reports whether the server has a key at all.
func (s ServerSettings) HasKey() bool { return s.Endpoint.APIKey != `` }

// settingsOverride replaces Server() in tests.
var settingsOverride atomic.Pointer[ServerSettings]

// SetServerForTest makes Server() return s until the returned restore runs.
func SetServerForTest(s ServerSettings) (restore func()) {
	prev := settingsOverride.Swap(&s)
	return func() { settingsOverride.Store(prev) }
}

// legacyConfig is the AI companion's own config block (Modules.aicompanion,
// keyed by setting name), where the server's key, endpoint, budget and
// breaker lived before the framework.
type legacyConfig map[string]any

// companionBlock reads Modules.aicompanion without flattening every
// module's settings: Server() runs several times per companion decision,
// under the mud lock.
func companionBlock() legacyConfig {
	block := configs.GetModulesConfig()[`aicompanion`]
	switch m := block.(type) {
	case map[string]any:
		return legacyConfig(m)
	case map[any]any: // how config.yaml itself decodes
		out := make(legacyConfig, len(m))
		for k, v := range m {
			if name, ok := k.(string); ok {
				out[name] = v
			}
		}
		return out
	}
	// Any other shape: the general route, which is slower but sure.
	out := legacyConfig{}
	for k, v := range configs.Flatten(map[string]any{`aicompanion`: block}) {
		out[strings.TrimPrefix(k, `aicompanion.`)] = v
	}
	return out
}

func (l legacyConfig) str(name string) string {
	s, _ := l[name].(string)
	return strings.TrimSpace(s)
}

func (l legacyConfig) num(name string) (int, bool) {
	switch v := l[name].(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case int32:
		return int(v), true
	case uint:
		return int(v), true
	case uint64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		return n, err == nil
	}
	return 0, false
}

func (l legacyConfig) flag(name string) bool {
	switch v := l[name].(type) {
	case bool:
		return v
	case string:
		b, _ := strconv.ParseBool(strings.TrimSpace(v))
		return b
	}
	return false
}

// snapshot is the last settings RefreshServer read from the config.
var snapshot atomic.Pointer[ServerSettings]

// Server is the server's key and limits: the snapshot RefreshServer last
// took, so it is safe from any goroutine (a bauble is named off the game
// loop). The config itself may only be read on the game loop: the engine
// writes it there (configs.SetVal, from admin commands and room creation)
// without a lock of its own, and a map read racing that write is a fatal
// error. Before the first refresh it reads the config once.
func Server() ServerSettings {
	if s := settingsOverride.Load(); s != nil {
		return *s
	}
	if s := snapshot.Load(); s != nil {
		return *s
	}
	return RefreshServer()
}

// RefreshServer reads the settings from the config now and makes them what
// Server returns. Call it ONLY on the game loop (with the mud lock held, or
// before any goroutine that calls Server can run): the modules that use the
// framework call it at load and every round, and their admin views call it
// so a `config set` shows at once. Otherwise a change takes effect within a
// round.
//
// Each setting is taken from the APIFramework section when it sets it,
// else from where the AI companion kept it before (Modules.aicompanion, the
// same name), else the companion's old default. So a config.yaml written
// before the framework existed behaves exactly as it did:
//
//   - the key: the environment variable named by APIKeyEnv (else
//     OPENAI_API_KEY), which wins, else APIKey;
//   - BaseURL, refused unless EndpointAllowed with AllowCustomEndpoint (set
//     in either place), the official endpoint used instead;
//   - DailyTokenBudget: a positive APIFramework value is the cap and a
//     negative one no cap; 0 or absent falls to the companion's setting
//     (0 or less there was no cap), else 2,000,000;
//   - BreakerErrors and BreakerSeconds, at least 1 and 5.
func RefreshServer() ServerSettings {
	if s := settingsOverride.Load(); s != nil {
		return *s
	}
	s := resolveServer(configs.GetAPIFrameworkConfig(), companionBlock())
	snapshot.Store(&s)
	return s
}

// resolveServer is Server's rule, on the APIFramework section c and the
// companion's old block old.
func resolveServer(c configs.APIFramework, old legacyConfig) ServerSettings {
	s := ServerSettings{}

	envName := string(c.APIKeyEnv)
	if envName == `` {
		if envName = old.str(`APIKeyEnv`); envName != `` {
			s.Legacy = append(s.Legacy, `APIKeyEnv`)
		} else {
			envName = DefaultKeyEnv
		}
	}
	configured := string(c.APIKey)
	if configured == `` {
		if configured = old.str(`APIKey`); configured != `` {
			s.Legacy = append(s.Legacy, `APIKey`)
		}
	}
	s.Endpoint.APIKey = ResolveKey(envName, configured)

	base := string(c.BaseURL)
	if base == `` {
		if base = strings.TrimRight(old.str(`BaseURL`), `/`); base != `` {
			s.Legacy = append(s.Legacy, `BaseURL`)
		}
	}
	// Explicit in APIFramework wins either way; empty inherits the old one.
	var allowCustom bool
	switch string(c.AllowCustomEndpoint) {
	case `true`:
		allowCustom = true
	case `false`:
		allowCustom = false
	default:
		if allowCustom = old.flag(`AllowCustomEndpoint`); allowCustom {
			s.Legacy = append(s.Legacy, `AllowCustomEndpoint`)
		}
	}
	if base == `` {
		base = DefaultBaseURL
	} else if !EndpointAllowed(base, allowCustom) {
		s.RejectedBaseURL = base
		base = DefaultBaseURL
	}
	s.Endpoint.BaseURL = base

	switch budget := int(c.DailyTokenBudget); {
	case budget > 0:
		s.DailyTokenBudget = budget
	case budget < 0:
		s.DailyTokenBudget = 0
	default:
		if n, ok := old.num(`DailyTokenBudget`); ok {
			s.Legacy = append(s.Legacy, `DailyTokenBudget`)
			s.DailyTokenBudget = max(n, 0)
		} else {
			s.DailyTokenBudget = DefaultDailyTokenBudget
		}
	}

	s.BreakerErrors = pick(int(c.BreakerErrors), old, `BreakerErrors`, DefaultBreakerErrors)
	s.BreakerSeconds = pick(int(c.BreakerSeconds), old, `BreakerSeconds`, DefaultBreakerSeconds)
	s.BreakerErrors = max(s.BreakerErrors, 1)
	s.BreakerSeconds = max(s.BreakerSeconds, 5)
	return s
}

// pick is a positive APIFramework value, else the companion's old setting of
// the same name (whatever it is: the caller applies the companion's old
// floors, so 0 there is 1 error or 5 seconds, as it always was), else def. The companion's own BreakerErrors and
// BreakerSeconds still govern each player's relay breaker, so reading them
// here is what the server key always used; only an APIFramework value is a
// change, and it is not flagged as legacy.
func pick(framework int, old legacyConfig, name string, def int) int {
	if framework > 0 {
		return framework
	}
	if n, ok := old.num(name); ok {
		return n
	}
	return def
}
