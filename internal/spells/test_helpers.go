package spells

// SeedSpellsForTest replaces the global allSpells map with the supplied test
// data, rebuilds the alias index to match (mirroring what LoadSpells does at
// boot), and returns a cleanup function that restores both to their originals.
// Intended for cross-package integration tests (hooks, commands).
func SeedSpellsForTest(spellMap map[string]*SpellData) func() {
	origSpells := allSpells
	origAliases := spellsByAlias
	allSpells = spellMap
	buildSpellAliasIndex()
	return func() {
		allSpells = origSpells
		spellsByAlias = origAliases
	}
}
