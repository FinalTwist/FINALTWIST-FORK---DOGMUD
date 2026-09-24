package gossip

import "github.com/GoMudEngine/GoMud/internal/narration"

// SeedForTest replaces the store and returns a restore func to defer.
// Intended for cross-package tests (hooks).
func SeedForTest(m map[string][]string) func() {
	old := templates
	templates = m
	return func() { templates = old }
}

// RenderWithForTest exposes the picker seam to the narration snapshot harness,
// an external test package.
func RenderWithForTest(pool []string, token, value string, pick narration.Picker) string {
	return renderWith(pool, token, value, pick)
}
