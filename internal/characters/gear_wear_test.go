package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// Critical hits wear swords and armour but never bows; bows wear on shots.
func TestCritWear(t *testing.T) {
	t.Cleanup(items.SeedItemsForTest(map[int]*items.ItemSpec{
		1: {ItemId: 1, Name: "Sword", Type: items.Weapon, Subtype: items.Slashing, DamageMultiplier: 1.0},
		2: {ItemId: 2, Name: "Bow", Type: items.Weapon, Subtype: items.Shooting, AmmoTag: `arrows`, DamageMultiplier: 1.0},
		3: {ItemId: 3, Name: "Helm", Type: items.Head, PhysicalMitigation: 5},
	}))
	c := &Character{}
	c.Equipment.Weapon = items.Item{ItemId: 1}
	c.Equipment.Head = items.Item{ItemId: 3}
	for i := 0; i < 200; i++ {
		c.CritWearWeapon()
		c.CritWearArmor()
	}
	if c.Equipment.Weapon.Wear == 0 || c.Equipment.Head.Wear == 0 {
		t.Errorf("two hundred crits should wear the sword and helm: %d, %d", c.Equipment.Weapon.Wear, c.Equipment.Head.Wear)
	}

	archer := &Character{}
	archer.Equipment.Weapon = items.Item{ItemId: 2}
	for i := 0; i < 200; i++ {
		archer.CritWearWeapon()
	}
	if archer.Equipment.Weapon.Wear != 0 {
		t.Error("a bow never wears from critical hits")
	}
	for i := 0; i < 1000; i++ {
		archer.WearBowOnShot(&archer.Equipment.Weapon)
	}
	if archer.Equipment.Weapon.Wear == 0 {
		t.Error("a thousand shots should wear the bow")
	}
}
