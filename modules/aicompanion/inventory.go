package aicompanion

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// Inventory management (F9.1 to F9.10). The companion's pack is always read
// live from the game. What the module adds is judgement: which kinds of
// supply it wants to keep and how many, when it is carrying too much, and
// which items it will never part with.

// Supply is one kind of thing the companion likes to keep stocked.
type Supply struct {
	Name    string `yaml:"name"`    // how the companion thinks of it: "arrows", "food"
	Type    string `yaml:"type"`    // item type to count (food, drink, potion, ammo...), optional
	Keyword string `yaml:"keyword"` // word in item names to count, optional
	Min     int    `yaml:"min"`     // below this it is a need
	Target  int    `yaml:"target"`  // restock up to this
}

// matches reports whether an item counts toward this supply.
func (s Supply) matches(it *items.Item) bool {
	spec := it.GetSpec()
	if s.Type != `` && string(spec.Type) != s.Type {
		return false
	}
	if s.Keyword != `` && !strings.Contains(strings.ToLower(it.Name()), strings.ToLower(s.Keyword)) {
		return false
	}
	return s.Type != `` || s.Keyword != ``
}

// supplyCount counts what the companion carries for one supply. Ammunition
// is counted by the arrows left in a bundle, not by the number of bundles:
// a quiver is one item holding many shots.
func supplyCount(mob *mobs.Mob, s Supply) int {
	n := 0
	count := func(it *items.Item) {
		if !s.matches(it) {
			return
		}
		if it.GetSpec().Type == items.Ammo && it.Uses > 0 {
			n += it.Uses
			return
		}
		n++
	}
	for i := range mob.Character.Items {
		count(&mob.Character.Items[i])
	}
	for i := range mob.Character.ComponentItems {
		count(&mob.Character.ComponentItems[i])
	}
	return n
}

// supplyNeed is one supply below its minimum.
type supplyNeed struct {
	Supply Supply
	Have   int
}

// supplyNeeds lists every supply below its minimum (F9.3).
func supplyNeeds(mob *mobs.Mob, supplies []Supply) []supplyNeed {
	var out []supplyNeed
	for _, s := range supplies {
		if have := supplyCount(mob, s); have < s.Min {
			out = append(out, supplyNeed{Supply: s, Have: have})
		}
	}
	return out
}

// stockWords turns a supply count into words.
func stockWords(have int) string {
	switch {
	case have <= 0:
		return `none left`
	case have == 1:
		return `only one left`
	case have < 5:
		return `only a few left`
	}
	return `running low`
}

// loadWords describes how heavy the pack is (F9.4).
func loadWords(mob *mobs.Mob) (string, bool) {
	capacity := mob.Character.CarryCapacity()
	if capacity <= 0 {
		return ``, false
	}
	ratio := mob.Character.GetCarriedWeight() / capacity
	switch {
	case ratio >= 1.5:
		return `your pack is far too heavy; you are struggling under it`, true
	case ratio >= 1.0:
		return `your pack is overloaded`, true
	case ratio >= 0.85:
		return `your pack is getting heavy`, true
	case ratio >= 0.5:
		return `your pack is comfortably full`, false
	}
	return `your pack is light`, false
}

// protectItem marks one more of an item as something the companion will
// not sell, drop or give away to anyone but its owner (F9.5). Items are
// protected by item id and count, because item UUIDs are not persisted.
func (m *Mind) protectItem(itemId int, reason string) {
	if itemId <= 0 {
		return
	}
	if m.Protected == nil {
		m.Protected = map[int]int{}
	}
	m.Protected[itemId]++
	if reason != `` {
		if m.ProtectedWhy == nil {
			m.ProtectedWhy = map[int]string{}
		}
		m.ProtectedWhy[itemId] = reason
	}
}

// protectInstance marks one exact item as a keepsake for as long as the
// world keeps running. Item uuids are not saved, so the count in Protected
// is what survives a restart; within a session this is what stops the very
// gift being handed over while an identical thing sits beside it.
func (m *Mind) protectInstance(uuid string) {
	if uuid == `` {
		return
	}
	if m.protectedInstances == nil {
		m.protectedInstances = map[string]bool{}
	}
	m.protectedInstances[uuid] = true
}

// canPartWithItem is the check for one exact item: the gift itself is
// never given away, and beyond that the count rule applies.
func canPartWithItem(mind *Mind, mob *mobs.Mob, it *items.Item) bool {
	if it == nil {
		return true
	}
	if mind.protectedInstances[it.UUID.String()] {
		return false
	}
	return canPartWith(mind, mob, it.ItemId)
}

// canPartWith reports whether the companion may let go of one of an item.
// It may, only while it carries more of that item than are protected.
func canPartWith(mind *Mind, mob *mobs.Mob, itemId int) bool {
	protected := mind.Protected[itemId]
	if protected <= 0 {
		return true
	}
	have := 0
	for _, it := range mob.Character.Items {
		if it.ItemId == itemId {
			have++
		}
	}
	return have > protected
}

// reconcileProtected drops protection for items the companion no longer
// carries at all (worn items still count), so the record cannot grow stale.
func reconcileProtected(mind *Mind, mob *mobs.Mob) {
	if len(mind.Protected) == 0 {
		return
	}
	have := map[int]int{}
	for _, it := range mob.Character.Items {
		have[it.ItemId]++
	}
	for _, it := range mob.Character.Equipment.GetAllItems() {
		have[it.ItemId]++
	}
	for id, n := range mind.Protected {
		if have[id] == 0 {
			delete(mind.Protected, id)
			delete(mind.ProtectedWhy, id)
		} else if have[id] < n {
			mind.Protected[id] = have[id]
		}
	}
}

// packLines describes the companion's needs and load for the prompt.
func packLines(mob *mobs.Mob, p *Profile, mind *Mind) []string {
	var out []string
	for _, n := range supplyNeeds(mob, p.Supplies) {
		out = append(out, fmt.Sprintf(`You are short of %s (%s; you like to keep about %d).`, n.Supply.Name, stockWords(n.Have), n.Supply.Target))
	}
	if words, heavy := loadWords(mob); words != `` {
		line := strings.ToUpper(words[:1]) + words[1:] + `.`
		if heavy {
			line += ` Consider selling, dropping or storing what you do not need.`
		}
		out = append(out, line)
	}
	ids := make([]int, 0, len(mind.ProtectedWhy))
	for id := range mind.ProtectedWhy {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		if mind.Protected[id] > 0 {
			if spec := items.GetItemSpec(id); spec != nil {
				out = append(out, fmt.Sprintf(`You will never sell or throw away the %s (%s).`, spec.Name, mind.ProtectedWhy[id]))
			}
		}
	}
	return out
}
