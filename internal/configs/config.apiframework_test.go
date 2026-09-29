package configs

import (
	"fmt"
	"strings"
	"testing"
)

// The server's key is a secret wherever the config is printed: a boot log, a
// server listing, /viewconfig (spec S1). Validate keeps it one.
func TestAPIFrameworkKeyIsASecret(t *testing.T) {
	a := APIFramework{APIKey: `  sk-sentinel-s1-0001  `}
	a.Validate()
	if string(a.APIKey) != `sk-sentinel-s1-0001` {
		t.Fatalf("Validate trims the value: got %d bytes", len(string(a.APIKey)))
	}
	if got := fmt.Sprint(a.APIKey); strings.Contains(got, `sentinel`) || got != `*** REDACTED ***` {
		t.Fatalf("printed, the value must be redacted, got %d bytes", len(got))
	}
	if got := fmt.Sprintf(`%v`, a); strings.Contains(got, `sentinel`) {
		t.Fatal("printing the whole section must not show the value")
	}
}

// The server key's destination, what spends it and whether its text is
// moderated cannot be changed from inside the game (spec M1 and S2, ruling
// 13): an admin who could set BaseURL or the model could send the key, or
// its budget, anywhere, and one who could turn moderation off would let
// unchecked text into the world. IsLocked is what SetVal and the server
// config menu both ask.
func TestKeyAndBaublesSpendSettingsAreHardLocked(t *testing.T) {
	for _, p := range []string{
		`APIFramework.APIKey`, `APIFramework.APIKeyEnv`, `APIFramework.BaseURL`, `APIFramework.AllowCustomEndpoint`,
		`Modules.baubles.Model`, `Modules.baubles.MaxCompletionTokens`, `Modules.baubles.MaxConcurrent`, `Modules.baubles.UsePlayerKeys`,
		`Modules.baubles.ModerateOutput`, `Modules.baubles.ModerationModel`,
		`modules.baubles.model`,
	} {
		if !isHardLocked(p) || !IsLocked(p) {
			t.Errorf("%v is not hard-locked", p)
		}
	}
	// A setting that only bounds how long a find waits is not the key's
	// destination or its spend: TimeoutSeconds stays tunable in game.
	if isHardLocked(`Modules.baubles.TimeoutSeconds`) {
		t.Error("Modules.baubles.TimeoutSeconds bounds a wait, not the spend, and stays tunable")
	}
}
