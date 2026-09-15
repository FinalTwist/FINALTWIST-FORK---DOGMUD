package keywords

// SeedKeywordsForTest replaces the global loadedKeywords with an empty but
// non-nil Aliases struct so that keyword lookups don't panic in tests. An
// optional override supplies real alias data (e.g. CommandAliases,
// HelpAliases) for a test that must exercise real alias resolution end to
// end, such as an admin command alias reaching the same handler as its
// canonical name. Returns a cleanup function that restores the original.
func SeedKeywordsForTest(overrides ...Aliases) func() {
	orig := loadedKeywords

	a := &Aliases{
		Help:               map[string]map[string][]string{},
		HelpAliases:        map[string][]string{},
		CommandAliases:     map[string][]string{},
		DirectionAliases:   map[string]string{},
		MapLegendOverrides: map[string]map[string]string{},
	}
	if len(overrides) > 0 {
		o := overrides[0]
		if o.Help != nil {
			a.Help = o.Help
		}
		if o.HelpAliases != nil {
			a.HelpAliases = o.HelpAliases
		}
		if o.CommandAliases != nil {
			a.CommandAliases = o.CommandAliases
		}
		if o.DirectionAliases != nil {
			a.DirectionAliases = o.DirectionAliases
		}
		if o.MapLegendOverrides != nil {
			a.MapLegendOverrides = o.MapLegendOverrides
		}
	}
	// Validate() unrolls Help/HelpAliases/CommandAliases into the lowercased
	// lookup maps TryCommandAlias/TryHelpAlias actually read; it also reads
	// the package-level fileSystems var for data-overlays, which is empty in
	// a test binary, so it never touches disk here.
	_ = a.Validate()

	loadedKeywords = a
	return func() {
		loadedKeywords = orig
	}
}
