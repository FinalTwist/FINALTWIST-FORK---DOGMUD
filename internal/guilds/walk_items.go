package guilds

import "github.com/GoMudEngine/GoMud/internal/items"

// WalkItems calls fn with a pointer to each item in the guild vault.
func (g *Guild) WalkItems(fn func(*items.Item)) {
	items.WalkSlice(g.Vault, fn)
}
