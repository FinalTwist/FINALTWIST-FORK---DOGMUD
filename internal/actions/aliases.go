package actions

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/keywords"
)

// ExpandAliases resolves a command's system and user aliases, mirroring the
// expansion dance in usercommands.TryCommand: user alias first (splitting a
// multi-word expansion into its own verb + rest and appending any
// pre-existing rest), then system alias over the resulting verb. userAlias
// may be nil (an actor with no user aliases, e.g. a mob), in which case only
// the system alias applies.
//
// This is the ONE shared implementation the command-execution path
// (usercommands.TryCommand) and the client action-readiness probe
// (ActionReadiness) both call. They drifted once already: a trigger-queued
// command sent through a user alias (e.g. "sa" -> "cast skill-attunement")
// matched neither the literal "cast" verb nor a specialMoveVerbs entry in
// ActionReadiness, so it fell through to ActionReady and fired immediately
// instead of being retried while the caster was busy.
func ExpandAliases(verb, rest string, userAlias func(string) string) (string, string) {
	verb = strings.ToLower(strings.TrimSpace(verb))

	if userAlias != nil {
		if expanded := userAlias(verb); expanded != verb {
			verb, rest = mergeAliasExpansion(expanded, rest)
		}
	}

	if expanded := keywords.TryCommandAlias(verb); expanded != verb {
		verb, rest = mergeAliasExpansion(expanded, rest)
	}

	return verb, rest
}

// mergeAliasExpansion splits a (possibly multi-word) alias expansion into its
// own verb and rest, then appends any pre-existing rest onto that rest —
// e.g. expansion "cast skill-attunement" with a pre-existing rest of "bob"
// yields ("cast", "skill-attunement bob").
func mergeAliasExpansion(expansion, rest string) (verb, newRest string) {
	parts := strings.SplitN(expansion, " ", 2)
	verb = parts[0]
	if len(parts) == 1 {
		return verb, rest
	}
	if rest == "" {
		return verb, parts[1]
	}
	return verb, parts[1] + " " + rest
}
