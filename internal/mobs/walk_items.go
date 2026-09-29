package mobs

import "github.com/GoMudEngine/GoMud/internal/items"

// WalkItems calls fn with a pointer to every item the mob holds (its
// character's, Character.WalkItems). Nothing else on a Mob holds items;
// TestItemWalkersVisitEveryItemField (repo root) fails if that changes.
func (m *Mob) WalkItems(fn func(*items.Item)) {
	m.Character.WalkItems(fn)
}
