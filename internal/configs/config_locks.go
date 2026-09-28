package configs

import "strings"

// hardLocked names the config keys no in-game command may change, whatever
// Server.Locked says: credentials, where they are sent, which models they pay
// for, whether and how generated text is moderated, the game's own web
// domain, and the lock list itself. Exact paths, compared lowercase against
// the path FindFullPath resolved.
//
// The APIFramework entries name a section master does not have yet (the
// baubles PR adds it). They are plain strings: an absent key matches nothing
// and costs nothing, and they bind the day the section lands.
var hardLocked = []string{
	`APIFramework.APIKey`,
	`APIFramework.APIKeyEnv`,
	`APIFramework.BaseURL`,
	`APIFramework.AllowCustomEndpoint`,
	`Modules.aicompanion.APIKey`,
	`Modules.aicompanion.APIKeyEnv`,
	`Modules.aicompanion.BaseURL`,
	`Modules.aicompanion.AllowCustomEndpoint`,
	`Modules.aicompanion.RelayOrigin`,
	`Modules.aicompanion.PlayerKeys`,
	`Modules.aicompanion.Model`,
	`Modules.aicompanion.FastModel`,
	`Modules.aicompanion.DeepModel`,
	`Modules.aicompanion.ModerateOutput`,
	`Modules.aicompanion.ModerationModel`,
	`FilePaths.WebDomain`,
	`Server.Locked`,
}

// isHardLocked reports whether configPath is on the hard list.
func isHardLocked(configPath string) bool {
	lower := strings.ToLower(configPath)
	for _, h := range hardLocked {
		if lower == strings.ToLower(h) {
			return true
		}
	}
	return false
}

// IsLocked reports whether an operator may not change configPath in game:
// it is on the hard list, it ends in "locked", or it starts with (lowercase)
// an entry of Server.Locked, so "FilePaths" locks every FilePaths key. The
// server config menu also passes partial paths while browsing; a partial path
// is locked only by a Server.Locked prefix, never by the exact hard list.
func IsLocked(configPath string) bool {
	if isHardLocked(configPath) {
		return true
	}
	lower := strings.ToLower(configPath)
	if strings.HasSuffix(lower, `locked`) {
		return true
	}
	for _, v := range GetServerConfig().Locked {
		if strings.HasPrefix(lower, strings.ToLower(v)) {
			return true
		}
	}
	return false
}
