package characters

import "strings"

// FindItemNoun looks for an exact noun on anything this character wears or
// carries, worn items first. Exact only: look runs it before item matching,
// and a prefix would let "hood" shadow an item named "hooded lantern".
//
// It searches the same holdings FindItem matches by name (worn slots, the
// backpack, the bandolier), so look and FindItemNoun agree on what "you
// carry". Component-bag contents are reachable by FindItem only through an
// item handle, never by name, so they are not searched here either.
func (c *Character) FindItemNoun(word string) (noun, desc string, ok bool) {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return "", "", false
	}
	candidates := c.GetAllWornItems()
	candidates = append(candidates, c.Items...)
	candidates = append(candidates, c.PotionItems...)
	for _, itm := range candidates {
		if itm.ItemId < 1 {
			continue
		}
		spec := itm.GetSpec() // a value, never nil
		for n, d := range spec.Nouns {
			if strings.ToLower(n) == word {
				return n, d, true
			}
		}
	}
	return "", "", false
}
