package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestBootConfigLogRedactsSecrets(t *testing.T) {
	const sentinel = `sk-bootlog-sentinel-7f3a`
	c := configs.GetConfig()
	c.Integrations.Discord.WebhookUrl = configs.ConfigSecret(sentinel)
	c.Modules = configs.Modules{`aicompanion`: map[string]any{`APIKey`: sentinel, `Model`: `gpt-test`}}

	logged := map[string]any{}
	var order []string
	logBootConfig(c, func(name string, value any) {
		logged[name] = value
		order = append(order, name)
	})

	// %#v never calls String(): a raw ConfigSecret shows here, as it would
	// under a JSON log handler.
	for name, value := range logged {
		if strings.Contains(fmt.Sprintf(`%#v`, value), sentinel) {
			t.Errorf(`boot log prints the secret at %s`, name)
		}
	}
	for _, name := range []string{`Modules.aicompanion.APIKey`, `Integrations.Discord.WebhookUrl`} {
		if logged[name] != configs.RedactedValue {
			t.Errorf(`boot log %s = %#v, want RedactedValue`, name, logged[name])
		}
	}
	if logged[`Modules.aicompanion.Model`] != `gpt-test` {
		t.Errorf(`a non-secret module value was redacted or dropped: %#v`, logged[`Modules.aicompanion.Model`])
	}
	for i := 1; i < len(order); i++ {
		if order[i-1] > order[i] {
			t.Fatalf(`boot log is not sorted: %q before %q`, order[i-1], order[i])
		}
	}
}
