package configs

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

const displaySentinel = `sk-display-sentinel-51c9`

func displayTestConfig() Config {
	c := GetConfig()
	c.Integrations.Discord.WebhookUrl = ConfigSecret(displaySentinel)
	c.Modules = Modules{
		`aicompanion`: map[string]any{`APIKey`: displaySentinel, `APIKeyEnv`: `OPENAI_API_KEY`, `Model`: `gpt-test`},
		`othermod`:    map[string]any{`password`: displaySentinel, `Secret`: displaySentinel},
	}
	return c
}

func TestDisplayConfigDataRedactsSecrets(t *testing.T) {
	c := displayTestConfig()
	shown := c.DisplayConfigData()

	// %#v never calls String(), so a raw ConfigSecret would show here too.
	for name, value := range shown {
		if strings.Contains(fmt.Sprintf(`%#v`, value), displaySentinel) {
			t.Errorf(`DisplayConfigData shows the secret at %s`, name)
		}
	}
	for _, name := range []string{`Integrations.Discord.WebhookUrl`, `Server.Seed`,
		`Modules.aicompanion.APIKey`, `Modules.othermod.password`, `Modules.othermod.Secret`} {
		if shown[name] != RedactedValue {
			t.Errorf(`DisplayConfigData[%q] = %#v, want RedactedValue`, name, shown[name])
		}
	}
	// The leaf match is exact: an environment variable NAME is not a secret.
	if shown[`Modules.aicompanion.APIKeyEnv`] != `OPENAI_API_KEY` {
		t.Errorf(`APIKeyEnv = %#v, want it shown`, shown[`Modules.aicompanion.APIKeyEnv`])
	}
	if shown[`Modules.aicompanion.Model`] != `gpt-test` {
		t.Errorf(`Model = %#v, want it shown`, shown[`Modules.aicompanion.Model`])
	}

	raw := c.AllConfigData()
	if raw[`Modules.aicompanion.APIKey`] != displaySentinel {
		t.Errorf(`AllConfigData must stay raw (the lookups read it), got %#v`, raw[`Modules.aicompanion.APIKey`])
	}
	if len(shown) != len(raw) {
		t.Errorf(`DisplayConfigData has %d keys, AllConfigData %d: redaction must not drop keys`, len(shown), len(raw))
	}
}

func TestDisplayConfigDataKeepsExclusions(t *testing.T) {
	for name := range displayTestConfig().DisplayConfigData(`modules*`) {
		if strings.HasPrefix(strings.ToLower(name), `modules`) {
			t.Errorf(`excluded key %s still present`, name)
		}
	}
}

func TestConfigSecretStringIsRedactedValue(t *testing.T) {
	if got := ConfigSecret(`x`).String(); got != RedactedValue {
		t.Errorf(`ConfigSecret.String() = %q, want RedactedValue`, got)
	}
}

// TestSetValRefusesTheRedactionMarker: the server config prompt offers the
// displayed value as its default, so Enter on a redacted leaf would otherwise
// write the marker over the real secret. The target is Server.Seed, a
// ConfigSecret that is NOT on the hard lock list: Integrations.Discord.WebhookUrl
// (Task 2) is hard-locked, so a SetVal against it is refused as locked before
// the marker check ever runs (lock check first, by design; see the second
// assertion below). Server.Locked is cleared explicitly so nothing on the
// shipped or Go-default lock list can shadow the marker check for this key.
func TestSetValRefusesTheRedactionMarker(t *testing.T) {
	c := GetConfig()
	c.Server.Seed = ConfigSecret(displaySentinel)
	c.Server.Locked = nil
	c.Integrations.Discord.WebhookUrl = ConfigSecret(displaySentinel)
	overridePath := SetConfigWithLookupsForTest(t, c)

	err := SetVal(`Server.Seed`, RedactedValue)
	if !errors.Is(err, ErrRedactedValue) {
		t.Fatalf(`SetVal(seed, RedactedValue) = %v, want ErrRedactedValue`, err)
	}
	if got := string(GetServerConfig().Seed); got != displaySentinel {
		t.Errorf(`Seed = %q, want the original secret kept`, got)
	}
	if _, statErr := os.Stat(overridePath); !os.IsNotExist(statErr) {
		t.Errorf(`a refused SetVal wrote %s`, overridePath)
	}

	// Locking wins over the marker check: a hard-locked key is refused as
	// locked even when the offered value happens to be the marker.
	if err := SetVal(`Integrations.Discord.WebhookUrl`, RedactedValue); !errors.Is(err, ErrLockedConfig) {
		t.Errorf(`SetVal(webhook, RedactedValue) = %v, want ErrLockedConfig`, err)
	}

	// The marker refusal also applies to the engine write path.
	if err := SetEngineVal(`Server.Seed`, RedactedValue); !errors.Is(err, ErrRedactedValue) {
		t.Errorf(`SetEngineVal(seed, RedactedValue) = %v, want ErrRedactedValue`, err)
	}
}

// nestedLeakTestConfig builds a config whose Modules map exercises the two
// leak shapes buildDotPaths' default case cannot decompose: a slice stored
// whole (a nested map full of secrets inside it never reaches its own
// dotted path), and a map stored whole under a secret-named parent key
// (recursed to leaves, but its OWN key name is what marks it, not the leaf).
// It also exercises the widened secret-name rule (suffix match, one word)
// and its required negatives: an *Env variable NAME, and plural/compound
// token counters that must stay visible.
func nestedLeakTestConfig() Config {
	c := GetConfig()
	c.Modules = Modules{
		`x`: map[string]any{
			// Widened rule: exact/suffix match of apikey, api_key, secret,
			// password, token, webhookurl (case-insensitive, one segment).
			`api_key`:      displaySentinel,
			`ClientSecret`: displaySentinel,
			`OpenAIAPIKey`: displaySentinel,
			`token`:        displaySentinel,
			`WebhookUrl`:   displaySentinel,

			// Must stay visible: an *Env suffix names the variable, not a
			// secret value; the token counters are plural/compound, not the
			// literal word "token".
			`APIKeyEnv`:               `OPENAI_API_KEY`,
			`MaxCompletionTokens`:     10,
			`DailyTokenBudget`:        5,
			`StrangerDailyTokens`:     3,
			`DailyTokensPerCompanion`: 2,

			// A slice stored whole (buildDotPaths does not recurse into
			// slice elements): a map[string]any inside it carries a
			// secret-named key ("apikey") that never becomes its own
			// dotted path, so only a deep scan of the slice's contents
			// finds it.
			`providers`: []any{
				map[string]any{`apikey`: displaySentinel, `name`: `openai`},
			},

			// A slice containing a map with non-string (any) keys, proving
			// the deep scan walks "maps with string or any keys" as the
			// plan requires, not just map[string]any.
			`backends`: []any{
				map[any]any{`secret`: displaySentinel, `name`: `primary`},
			},

			// A map under a secret-named key: fully decomposed by
			// buildDotPaths to Modules.x.password.db, whose LEAF ("db") is
			// innocent. Only the mid-path segment "password" marks it.
			`password`: map[string]any{
				`db`: `innocuous-value`,
			},
		},
	}
	return c
}

func TestDisplayConfigDataRedactsNestedAndWidenedNames(t *testing.T) {
	shown := nestedLeakTestConfig().DisplayConfigData()

	for _, name := range []string{
		`Modules.x.api_key`,
		`Modules.x.ClientSecret`,
		`Modules.x.OpenAIAPIKey`,
		`Modules.x.token`,
		`Modules.x.WebhookUrl`,
		`Modules.x.providers`,
		`Modules.x.backends`,
		`Modules.x.password.db`,
	} {
		if shown[name] != RedactedValue {
			t.Errorf(`DisplayConfigData[%q] = %#v, want RedactedValue`, name, shown[name])
		}
	}

	visible := map[string]any{
		`Modules.x.APIKeyEnv`:               `OPENAI_API_KEY`,
		`Modules.x.MaxCompletionTokens`:     10,
		`Modules.x.DailyTokenBudget`:        5,
		`Modules.x.StrangerDailyTokens`:     3,
		`Modules.x.DailyTokensPerCompanion`: 2,
	}
	for name, want := range visible {
		if shown[name] != want {
			t.Errorf(`DisplayConfigData[%q] = %#v, want %#v (unredacted)`, name, shown[name], want)
		}
	}
}
