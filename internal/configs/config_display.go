package configs

import (
	"errors"
	"regexp"
	"strings"
)

// RedactedValue is what every human-facing view of the config prints in
// place of a secret. ConfigSecret.String returns it too.
const RedactedValue = `*** REDACTED ***`

// ErrRedactedValue refuses the redaction marker as a new value: a prompt that
// offers the displayed value as its default must not write the marker over
// the real secret.
var ErrRedactedValue = errors.New("value is the redaction marker, not a real value")

// secretLeafName matches the last path element of a key whose value is a
// credential even when it is not typed ConfigSecret. Module config lives in
// the untyped Modules map, so Modules.aicompanion.APIKey arrives as a plain
// string and only its name marks it.
var secretLeafName = regexp.MustCompile(`(?i)^(apikey|secret|password)$`)

// DisplayConfigData is AllConfigData for anything a person reads: the boot
// log, the server set listing, the server config menu and /viewconfig. Every
// ConfigSecret, and every leaf secretLeafName matches, reads RedactedValue.
// AllConfigData stays raw because the key and type lookups are built from it.
func (c Config) DisplayConfigData(excludeStrings ...string) map[string]any {
	out := c.AllConfigData(excludeStrings...)
	for name, value := range out {
		if isSecretConfigValue(name, value) {
			out[name] = RedactedValue
		}
	}
	return out
}

func isSecretConfigValue(name string, value any) bool {
	if _, ok := value.(ConfigSecret); ok {
		return true
	}
	leaf := name
	if i := strings.LastIndex(name, `.`); i >= 0 {
		leaf = name[i+1:]
	}
	return secretLeafName.MatchString(leaf)
}
