package configs

import (
	"strconv"
	"strings"
)

// APIFramework is the server's own model key and what bounds its use,
// shared by every feature that calls a language model with it (the AI
// companion and bauble naming): internal/apiframework reads this section.
//
// There is ONE daily token budget for the server's key, whichever feature
// spends it, and its circuit breakers (the provider's, shared, and one per
// feature) all take BreakerErrors and BreakerSeconds. A player's own key
// (the companion key relay) is not the server's and is bounded by nothing
// here.
type APIFramework struct {
	// APIKeyEnv names the environment variable holding the key. Empty: the
	// companion's old Modules.aicompanion.APIKeyEnv if set, else
	// OPENAI_API_KEY, as the companion always read.
	APIKeyEnv ConfigString `yaml:"APIKeyEnv"`
	// APIKey is the key itself, for a private server (empty: the companion's
	// old Modules.aicompanion.APIKey). The environment variable wins when
	// both are set. A ConfigSecret: every config listing prints it redacted.
	// Never logged.
	APIKey ConfigSecret `yaml:"APIKey"`
	// BaseURL is the provider's API root. Empty: the companion's old
	// Modules.aicompanion.BaseURL if set, else https://api.openai.com/v1.
	BaseURL ConfigString `yaml:"BaseURL"`
	// AllowCustomEndpoint lets BaseURL name a non-OpenAI provider (https
	// only): "true" or "false" (a bare YAML true/false is read the same).
	// Empty: the companion's old Modules.aicompanion.AllowCustomEndpoint,
	// else false. A string, not a bool, so an explicit false here can
	// override an old true (a bool cannot tell false from absent); anything
	// else is read as false, since this guards where the key is sent.
	AllowCustomEndpoint ConfigString `yaml:"AllowCustomEndpoint"`
	// DailyTokenBudget is the most tokens the server's key may spend per UTC
	// day, across every feature. A negative value is no cap. 0 or absent:
	// the companion's old Modules.aicompanion.DailyTokenBudget when that is
	// set (where 0 always meant no cap), else 2,000,000.
	DailyTokenBudget ConfigInt `yaml:"DailyTokenBudget"`
	// BreakerErrors failures in a row on the server's key stop every call on
	// it for BreakerSeconds. 0 or absent: the companion's old settings of the
	// same names when set, else 5 and 60.
	BreakerErrors  ConfigInt `yaml:"BreakerErrors"`
	BreakerSeconds ConfigInt `yaml:"BreakerSeconds"`
	// CompanionSharePercent and BaublesSharePercent cap what each feature
	// may hold of DailyTokenBudget in a UTC day, as a percentage, so one
	// feature cannot spend the day for the others. 0 or absent: the
	// default (the companion 100, baubles 25). -1, or 100 and above: no
	// share cap. With no DailyTokenBudget there is no share cap either.
	CompanionSharePercent ConfigInt `yaml:"CompanionSharePercent"`
	BaublesSharePercent   ConfigInt `yaml:"BaublesSharePercent"`
}

// Validate tidies the strings. It sets no numeric defaults: an absent key
// decodes as zero, and internal/apiframework (Server) reads zero as "not set
// here", so a config.yaml written before this section existed keeps the
// companion's old settings rather than being silently given new ones.
func (a *APIFramework) Validate() {
	a.APIKeyEnv = ConfigString(strings.TrimSpace(string(a.APIKeyEnv)))
	a.APIKey = ConfigSecret(strings.TrimSpace(string(a.APIKey)))
	a.BaseURL = ConfigString(strings.TrimRight(strings.TrimSpace(string(a.BaseURL)), `/`))
	if v := strings.TrimSpace(string(a.AllowCustomEndpoint)); v != `` {
		if b, err := strconv.ParseBool(v); err == nil && b {
			a.AllowCustomEndpoint = `true`
		} else {
			a.AllowCustomEndpoint = `false`
		}
	}
}

// GetAPIFrameworkConfig returns a thread-safe copy of the APIFramework
// section.
func GetAPIFrameworkConfig() APIFramework {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()
	return configData.APIFramework
}
