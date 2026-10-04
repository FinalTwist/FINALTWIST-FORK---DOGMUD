package characters

import (
	"math/rand/v2"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// Gear wear in a fight (wilderness trades). A critical hit dealt may wear
// the striking weapon, and a critical hit taken may wear one piece of the
// defender's armour or shield. A bow is the exception: it never strikes, so
// it wears per arrow fired instead (WearBowOnShot). Wear lives on the item
// (items.Item.Wear); broken gear works badly until repaired.
//
// The rolls use math/rand/v2's own source, not util.Rand, so they never
// shift the sequence a seeded combat test depends on.

// chance reports whether a roll in [0,1) lands under p.
func chance(p float64) bool {
	return p > 0 && rand.Float64() < p
}

// CritWearWeapon rolls Balance.GearCritWearChance for the weapon that just
// landed a critical hit and wears it one point. Wielding two weapons, either
// may take it. Bows and other shooters are skipped (they wear per shot). It returns the weapon's
// name and whether that wear broke it; name is "" when nothing wore.
func (c *Character) CritWearWeapon() (name string, broke bool) {
	if c == nil || !chance(float64(configs.GetBalanceConfig().GearCritWearChance)) {
		return ``, false
	}
	var picks []*items.Item
	for _, p := range []*items.Item{&c.Equipment.Weapon, &c.Equipment.Offhand} {
		if p.ItemId < 1 {
			continue
		}
		spec := p.GetRawSpec()
		if spec.Type != items.Weapon || items.IsShooter(spec) || p.IsBroken() {
			continue
		}
		picks = append(picks, p)
	}
	if len(picks) == 0 {
		return ``, false
	}
	w := picks[rand.IntN(len(picks))]
	return w.NameSimple(), w.AddWear(1)
}

// CritWearArmor rolls Balance.GearArmorCritWearChance for a critical hit just
// taken and wears one worn armour piece or shield, picked at random. It
// returns the piece's name and whether that wear broke it.
func (c *Character) CritWearArmor() (name string, broke bool) {
	if c == nil || !chance(float64(configs.GetBalanceConfig().GearArmorCritWearChance)) {
		return ``, false
	}
	var picks []*items.Item
	for _, p := range c.Equipment.GetAllItemPtrs() {
		spec := p.GetRawSpec()
		if spec.Type == items.Weapon || !items.IsWearableGear(spec) || p.IsBroken() {
			continue
		}
		picks = append(picks, p)
	}
	if len(picks) == 0 {
		return ``, false
	}
	a := picks[rand.IntN(len(picks))]
	return a.NameSimple(), a.AddWear(1)
}

// WearBowOnShot rolls Balance.BowShotWearChance for the bow (or crossbow or
// sling) that just loosed a shot. It returns the bow's name and whether that wear broke it.
func (c *Character) WearBowOnShot(bow *items.Item) (name string, broke bool) {
	if c == nil || bow == nil || bow.ItemId < 1 || !items.IsShooter(bow.GetRawSpec()) || bow.IsBroken() {
		return ``, false
	}
	if !chance(float64(configs.GetBalanceConfig().BowShotWearChance)) {
		return ``, false
	}
	return bow.NameSimple(), bow.AddWear(1)
}
