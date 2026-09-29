package items

// Model-safe accessors. Text a player's own key wrote, moderated or not,
// never goes into any language model's prompt (spec S3; the baubles
// module's RecentNames and regeneration keep it out of theirs, and these
// keep it out of the AI companion's). Such a bauble reads as its carrier
// item's own spec ("Curious Trinket"). Use these, never Name or
// GetLongDescription, for anything a model is told.

// ModelName is Name for a model's prompt.
func (i *Item) ModelName() string {
	if spec, ok := i.playerTextCarrier(); ok {
		return spec.Name
	}
	return i.Name()
}

// ModelDescription is GetLongDescription for a model's prompt.
func (i *Item) ModelDescription() string {
	if spec, ok := i.playerTextCarrier(); ok {
		return i.longDescriptionFrom(spec)
	}
	return i.GetLongDescription()
}

// playerTextCarrier is the carrier's own spec when this is a bauble whose
// text a player's own key wrote.
func (i *Item) playerTextCarrier() (ItemSpec, bool) {
	if i.Bauble == `` {
		return ItemSpec{}, false
	}
	p := baubleResolver.Load()
	if p == nil || *p == nil {
		return ItemSpec{}, false
	}
	v, ok := (*p)(i.Bauble)
	if !ok || !v.PlayerText {
		return ItemSpec{}, false
	}
	spec := GetItemSpec(i.ItemId)
	if spec == nil {
		return ItemSpec{}, false
	}
	return *spec, true
}
