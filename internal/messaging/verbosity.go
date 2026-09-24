package messaging

import "strings"

// Verbosity is a player's combat-text verbosity preference. Full is the
// engine's historical behavior; Medium shows landed hits only; Light
// suppresses individual lines in favor of a per-round compact tally
// (built by the combat hook — see internal/hooks/combat_verbosity.go).
// Spectated fights render one step lower than the viewer's setting.
type Verbosity int

const (
	VerbosityFull Verbosity = iota
	VerbosityMedium
	VerbosityLight
)

// ParseVerbosity maps a stored/user-typed string to a Verbosity.
// Unknown or empty input is Full — the safe, historical default.
func ParseVerbosity(s string) Verbosity {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "medium":
		return VerbosityMedium
	case "light":
		return VerbosityLight
	}
	return VerbosityFull
}

func (v Verbosity) String() string {
	switch v {
	case VerbosityMedium:
		return "medium"
	case VerbosityLight:
		return "light"
	}
	return "full"
}

// OneStepLower is the spectator tier: fights you're merely watching
// render one level quieter than your setting. Light is the floor.
func (v Verbosity) OneStepLower() Verbosity {
	switch v {
	case VerbosityFull:
		return VerbosityMedium
	default:
		return VerbosityLight
	}
}

// suppressibleAtMedium / suppressibleAtLight are explicit allowlists of
// categories the verbosity gate may drop. Anything not listed always
// passes — new combat text is verbose-by-default (safe).
var suppressibleAtMedium = map[Category]bool{
	CategoryDodge: true,
	CategoryParry: true,
	CategoryBlock: true,
}

var suppressibleAtLight = map[Category]bool{
	CategoryDodge:              true,
	CategoryParry:              true,
	CategoryBlock:              true,
	CategoryHitMelee:           true,
	CategoryHitBlunt:           true,
	CategoryHitNaturalSharp:    true,
	CategoryHitRanged:          true,
	CategoryHitCaster:          true,
	CategoryHitUnarmed:         true,
	CategoryCombatBlindWarning: true,
}

// CategoryCombatBlindWarning is suppressible at Light ONLY (owner ruling,
// M4d PR 2 followup): NOT in suppressibleAtMedium, IS in
// suppressibleAtLight.
//
// At Medium the player still reads per-swing combat prose (defense
// categories aside), so the notice explains text they are actually
// seeing this round -- it stays unsuppressed there. At Light they have
// asked for near-silence, and a per-round line they cannot turn off
// would override a preference they deliberately set; a blind Light
// combatant already gets no per-swing prose (suppressed) and no
// CategoryCombatSummary tally (gated on sight at the record site,
// internal/hooks/NewRound_DoCombat_unified.go), so suppressing the
// notice too leaves them in the silence they asked for, not stranded
// without ANY explanation -- they chose that tradeoff at Light.

// Suppresses reports whether this verbosity level drops lines of the
// given category. Floor rules (damage-to-viewer always shows) are the
// caller's responsibility — this is a pure category table.
func (v Verbosity) Suppresses(cat Category) bool {
	switch v {
	case VerbosityMedium:
		return suppressibleAtMedium[cat]
	case VerbosityLight:
		return suppressibleAtLight[cat]
	}
	return false
}
