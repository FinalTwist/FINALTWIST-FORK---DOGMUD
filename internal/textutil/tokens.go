package textutil

import (
	"regexp"

	"github.com/GoMudEngine/GoMud/internal/narration"
)

// TokenContext holds the names for substitution in YAML text fields.
//
// Actor is the one acting; Actee is the one acted upon. The conditions store
// passes the HOLDER as the actee, because a condition happens to its holder
// (see internal/conditions/narration.go).
type TokenContext struct {
	ActorName      string // ANSI-tagged display name
	ActorPlainName string // Plain name (for possessives)
	ActeeName      string // ANSI-tagged display name (empty if none)
	ActeePlainName string // Plain name (empty if none)
}

// Tokens is the vocabulary as the narration core takes it. All four keys are
// always present, so an absent actee substitutes to an empty string.
func (ctx TokenContext) Tokens() map[string]string {
	return map[string]string{
		narration.TokenActor:      ctx.ActorName,
		narration.TokenActee:      ctx.ActeeName,
		narration.TokenActorPlain: ctx.ActorPlainName,
		narration.TokenActeePlain: ctx.ActeePlainName,
	}
}

// SubstituteTokens replaces known tokens in text with values from ctx.
// Unknown tokens are left as-is. Empty string input returns empty string.
//
// Since M3 item 5b it is a one-variant Narrate, so one engine substitutes
// for every store. The result is identical to the former four-pair
// strings.NewReplacer: no token is a prefix of another (each ends in "}"),
// so pair order cannot change the output.
func SubstituteTokens(text string, ctx TokenContext) string {
	if text == "" {
		return ""
	}
	return Narrate(narration.Variants{Actor: Pool(text)}, ctx).Actor
}

var tokenPattern = regexp.MustCompile(`\{[a-z_]+\}`)

var knownTokens = map[string]bool{
	narration.TokenActor:      true,
	narration.TokenActee:      true,
	narration.TokenActorPlain: true,
	narration.TokenActeePlain: true,
}

// ValidateTokens scans text for {token} patterns and returns warnings
// for any that are not in the known set.
func ValidateTokens(text string) []string {
	if text == "" {
		return nil
	}
	var warnings []string
	matches := tokenPattern.FindAllString(text, -1)
	for _, m := range matches {
		if !knownTokens[m] {
			warnings = append(warnings, "unknown token: "+m)
		}
	}
	return warnings
}
