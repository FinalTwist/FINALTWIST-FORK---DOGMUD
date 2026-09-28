package configs

import (
	"errors"
	"fmt"
	"reflect"
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

// secretNameRule matches a single dotted-path segment (or a map key found
// while deep-scanning a value) that names a credential even when it is not
// typed ConfigSecret. Module config lives in the untyped Modules map, so
// Modules.aicompanion.APIKey arrives as a plain string and only its name
// marks it. Exact or suffix match, case-insensitive, checked one path
// element (or map key) at a time: ClientSecret and OpenAIAPIKey match,
// APIKeyEnv (a variable NAME, not a secret) does not, and the plural or
// compound token counters (MaxCompletionTokens, DailyTokenBudget,
// StrangerDailyTokens, DailyTokensPerCompanion) do not, because their
// suffix is "Tokens" or "Budget", never the literal word "token". Checked
// against shipped keys (_datafiles/config.yaml and every internal/configs
// yaml tag) on 2026-09-28: only APIKey and WebhookUrl match, both secrets.
var secretNameRule = regexp.MustCompile(`(?i)(apikey|api_key|secret|password|token|webhookurl)$`)

// DisplayConfigData is AllConfigData for anything a person reads: the boot
// log, the server set listing, the server config menu and /viewconfig. A
// value reads RedactedValue when it is a ConfigSecret, when any element of
// its dotted path names a credential (secretNameRule), or when the value
// itself is a slice or map that a deep scan finds carries one: buildDotPaths
// stores a slice whole at its parent path (never recursing into elements),
// so a secret nested inside a slice, or inside a map nested inside a slice,
// never earns its own dotted path and only a deep scan of the stored value
// catches it. AllConfigData stays raw because the key and type lookups are
// built from it.
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
	return pathHasSecretSegment(name) || containsSecret(value)
}

// pathHasSecretSegment reports whether any dotted-path element (not just the
// leaf) matches secretNameRule, so Modules.x.password.db is caught by the
// mid-path "password" segment even though its leaf ("db") is innocent.
func pathHasSecretSegment(path string) bool {
	for _, segment := range strings.Split(path, `.`) {
		if secretNameRule.MatchString(segment) {
			return true
		}
	}
	return false
}

// containsSecret deep-scans a value that AllConfigData stored whole (a slice
// or a map, at any nesting depth, reached through maps with string or any
// keys, slices, arrays and interfaces) and reports whether it is or carries
// a ConfigSecret, or a map key matching secretNameRule. A match redacts the
// entire stored value, since DisplayConfigData cannot redact just the
// offending piece without decomposing a value AllConfigData never
// decomposed.
func containsSecret(value any) bool {
	if value == nil {
		return false
	}
	if _, ok := value.(ConfigSecret); ok {
		return true
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Interface, reflect.Ptr:
		if rv.IsNil() {
			return false
		}
		return containsSecret(rv.Elem().Interface())
	case reflect.Map:
		for _, key := range rv.MapKeys() {
			if secretNameRule.MatchString(fmt.Sprintf(`%v`, key.Interface())) {
				return true
			}
			if containsSecret(rv.MapIndex(key).Interface()) {
				return true
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			if containsSecret(rv.Index(i).Interface()) {
				return true
			}
		}
	}
	return false
}
