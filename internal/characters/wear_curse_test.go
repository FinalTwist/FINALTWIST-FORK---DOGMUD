package characters

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func armourSpec(name string, t items.ItemType) items.ItemSpec {
	return items.ItemSpec{Name: name, Type: t, Subtype: items.Wearable}
}

// Every single-slot armour family refuses to displace a cursed piece, and a
// refusal leaves the body exactly as it was (spec ruling 8, E3).
func TestWear_RefusesToDisplaceCursedArmour(t *testing.T) {
	seedGoldenSpecies(t)
	for _, tc := range []struct {
		slot func(c *Character) *items.Item
		typ  items.ItemType
	}{
		{func(c *Character) *items.Item { return &c.Equipment.Head }, items.Head},
		{func(c *Character) *items.Item { return &c.Equipment.Neck }, items.Neck},
		{func(c *Character) *items.Item { return &c.Equipment.Body }, items.Body},
		{func(c *Character) *items.Item { return &c.Equipment.Belt }, items.Belt},
		{func(c *Character) *items.Item { return &c.Equipment.Gloves }, items.Gloves},
		{func(c *Character) *items.Item { return &c.Equipment.Back }, items.Back},
		{func(c *Character) *items.Item { return &c.Equipment.Shoulders }, items.Shoulders},
		{func(c *Character) *items.Item { return &c.Equipment.Legs }, items.Legs},
		{func(c *Character) *items.Item { return &c.Equipment.Feet }, items.Feet},
		{func(c *Character) *items.Item { return &c.Equipment.ComponentBag }, items.ComponentBag},
		{func(c *Character) *items.Item { return &c.Equipment.Light }, items.Light},
	} {
		c := goldenChar(0, goldenMedium, false)
		old := goldenItem(4001, armourSpec("old "+string(tc.typ), tc.typ), true)
		*tc.slot(c) = old
		before := goldenSnapshot(c)
		ret, worn, why := c.Wear(goldenItem(4002, armourSpec("new "+string(tc.typ), tc.typ), false))
		assert.False(t, worn, "%s", tc.typ)
		assert.Nil(t, ret, "%s", tc.typ)
		// DisplayName title-cases the name ("Old Head").
		assert.Equal(t, `Your `+old.DisplayName()+cursedLine, why, "%s", tc.typ)
		assert.Equal(t, before, goldenSnapshot(c), "%s: a refusal leaves the body as it was", tc.typ)
	}
}

func TestWear_AnUncursedItemSwapsAsToday(t *testing.T) {
	seedGoldenSpecies(t)
	c := goldenChar(0, goldenMedium, false)
	lifted := goldenItem(4101, armourSpec("hood", items.Head), true)
	lifted.Uncursed = true
	c.Equipment.Head = lifted
	ret, worn, why := c.Wear(goldenItem(4102, armourSpec("cap", items.Head), false))
	require.True(t, worn, why)
	assert.Equal(t, "4101,", goldenIds(ret))
	assert.Equal(t, 4102, c.Equipment.Head.ItemId)
}

// A cursed refusal reads as the curse even when the swap would also breach
// the reservation ceiling: the curse pass runs before the reservation check.
func TestWear_ACursedRefusalWinsOverAReservationRefusal(t *testing.T) {
	seedGoldenSpecies(t)
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		4201: {ItemId: 4201, Name: "hungry collar", Type: items.Neck, Subtype: items.Wearable, ReserveStaminaPct: 0.60},
		4202: {ItemId: 4202, Name: "cursed belt", Type: items.Belt, Subtype: items.Wearable, Cursed: true},
		4203: {ItemId: 4203, Name: "hungry belt", Type: items.Belt, Subtype: items.Wearable, ReserveStaminaPct: 0.30},
	})()
	c := New()
	c.StaminaMax.Base = 100
	c.Equipment.Neck = items.New(4201)
	c.Equipment.Belt = items.New(4202)
	c.Validate()
	_, worn, why := c.Wear(items.New(4203))
	assert.False(t, worn)
	assert.Equal(t, `Your Cursed Belt`+cursedLine, why)
	assert.False(t, strings.Contains(strings.ToLower(why), "reserve"))
}

// The 2-arm hand families, each with no uncursed alternative.
func TestWear_HandFamiliesAtTwoArms(t *testing.T) {
	seedGoldenSpecies(t)
	for _, tc := range []struct {
		name   string
		layout string
		item   items.ItemSpec
		why    string
	}{
		{"2H over a cursed main hand", "Ss", goldenGreat, "Your Sword" + cursedLine},
		{"shield over a cursed offhand item", "sB", goldenShield, "Your Buckler" + cursedLine},
		{"1H over a cursed main hand, no dual wield", "Ss", goldenSword, "Your Sword" + cursedLine},
		{"1H over a two-hander with a cursed stray behind it", "gB", goldenSword, "Your Buckler" + cursedLine},
	} {
		c := goldenChar(0, goldenMedium, false)
		id := 4300
		applyLayout(handSlotsInArmOrder(c), tc.layout, &id)
		before := goldenSnapshot(c)
		_, worn, why := c.Wear(goldenItem(9000, tc.item, false))
		assert.False(t, worn, tc.name)
		assert.Equal(t, tc.why, why, tc.name)
		assert.Equal(t, before, goldenSnapshot(c), tc.name)
	}
}

// Wear lands exactly where ChooseWornSlot said, for the whole hand table.
func TestWear_FollowsTheHelperAcrossArmCounts(t *testing.T) {
	seedGoldenSpecies(t)
	for _, extra := range []int{0, 1, 2, 4} {
		for _, item := range []items.ItemSpec{goldenSword, goldenShield, goldenGreat} {
			for _, cursedArm := range []int{0, 1, 2, 3, 4, 5, 6} {
				c := goldenChar(extra, goldenMedium, false)
				ov := map[int]rune{}
				if cursedArm > 0 && cursedArm <= 2+extra {
					ov[cursedArm] = 'S'
				}
				fillHands(c, ov)
				choice, refusal := c.ChooseWornSlot(goldenItem(9000, item, false), 0)
				ret, worn, why := c.Wear(goldenItem(9000, item, false))
				if refusal != `` {
					assert.False(t, worn)
					assert.Equal(t, refusal, why)
					continue
				}
				require.True(t, worn, why)
				assert.Equal(t, goldenIds(choice.Displaced), goldenIds(ret))
				assert.Equal(t, 9000, choice.Slots[0].Item.ItemId, "the chosen slot now holds the item")
			}
		}
	}
}

// Ruling 13's nothing-cursed change, and the disabled-Ring change, both
// carved out of the golden test.
func TestWear_SanctionedChangesWithNothingCursed(t *testing.T) {
	seedGoldenSpecies(t)
	c := goldenChar(1, goldenMedium, false)
	id := 4400
	applyLayout(handSlotsInArmOrder(c), "ges", &id)
	ret, worn, why := c.Wear(goldenItem(9000, goldenShield, false))
	require.True(t, worn, why)
	assert.Equal(t, 9000, c.Equipment.ExtraArm1.ItemId, "the shield takes arm 3")
	assert.Len(t, ret, 1)

	c = goldenChar(0, goldenMedium, false)
	applyLayout([]*items.Item{&c.Equipment.Ring, &c.Equipment.Ring2}, "dr", &id)
	_, worn, why = c.Wear(goldenItem(9001, goldenRing, false))
	require.True(t, worn, why)
	assert.Equal(t, 9001, c.Equipment.Ring2.ItemId)
	assert.True(t, c.Equipment.Ring.IsDisabled(), "a disabled Ring is never written")
}
