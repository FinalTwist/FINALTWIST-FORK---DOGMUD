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
