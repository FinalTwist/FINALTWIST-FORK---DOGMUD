package users

import "github.com/GoMudEngine/GoMud/internal/items"

// WalkItems calls fn with a pointer to every item this account holds: its
// active character's (Character.WalkItems), the bank (the slots and the
// legacy list), and any item waiting in the inbox. Alts are not in memory:
// they live only in <userid>.alts.yaml, which the bauble sweep reads from
// disk.
func (u *UserRecord) WalkItems(fn func(*items.Item)) {
	if u.Character != nil {
		u.Character.WalkItems(fn)
	}
	for _, it := range u.ItemStorage.AllItemPtrs() {
		if it.ItemId > 0 {
			fn(it)
		}
	}
	for i := range u.Inbox {
		if it := u.Inbox[i].Item; it != nil && it.ItemId > 0 {
			fn(it)
		}
	}
}

// GetAllLoadedUsers returns every user record in memory, link-dead
// (zombie) users included: their characters are still in the world.
// GetAllActiveUsers leaves zombies out.
func GetAllLoadedUsers() []*UserRecord {
	userManager.mu.RLock()
	defer userManager.mu.RUnlock()

	ret := make([]*UserRecord, 0, len(userManager.Users))
	for _, u := range userManager.Users {
		ret = append(ret, u)
	}
	return ret
}
