package configs

import (
	"testing"
)

// TestSetConfigWithLookupsForTestResolvesKeys proves the helper gives
// FindFullPath real lookups (a test binary never runs ReloadConfig) and puts
// the previous lookups back when the test ends.
func TestSetConfigWithLookupsForTestResolvesKeys(t *testing.T) {
	beforePath, beforeType := FindFullPath(`apikey`)

	t.Run(`installed`, func(t *testing.T) {
		c := GetConfig()
		c.Modules = Modules{`aicompanion`: map[string]any{`APIKey`: ``}}
		SetConfigWithLookupsForTest(t, c)

		if p, typ := FindFullPath(`seed`); p != `Server.Seed` || typ != `configs.ConfigSecret` {
			t.Errorf(`FindFullPath("seed") = (%q, %q), want ("Server.Seed", "configs.ConfigSecret")`, p, typ)
		}
		if p, typ := FindFullPath(`apikey`); p != `Modules.aicompanion.APIKey` || typ != `string` {
			t.Errorf(`FindFullPath("apikey") = (%q, %q), want ("Modules.aicompanion.APIKey", "string")`, p, typ)
		}
	})

	if p, typ := FindFullPath(`apikey`); p != beforePath || typ != beforeType {
		t.Errorf(`lookups not restored: FindFullPath("apikey") = (%q, %q), was (%q, %q)`, p, typ, beforePath, beforeType)
	}
}
