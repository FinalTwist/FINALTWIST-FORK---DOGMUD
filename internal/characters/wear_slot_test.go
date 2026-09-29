package characters

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cursedLine = ` is cursed and prevents you from removing it.`

// fillHands puts a plain sword in every hand slot the character has, then
// applies overrides (arm number to code, as applyLayout reads it).
func fillHands(c *Character, overrides map[int]rune) {
	slots := handSlotsInArmOrder(c)
	layout := []rune(strings.Repeat("s", len(slots)))
	for arm, code := range overrides {
		layout[arm-1] = code
	}
	id := 2000
	applyLayout(slots, string(layout), &id)
}

func TestCursedRefusal(t *testing.T) {
	c := New()
	assert.Equal(t, ``, c.CursedRefusal(items.Item{}), "an empty slot never refuses")
	assert.Equal(t, ``, c.CursedRefusal(goldenItem(11, goldenRing, false)))
	assert.Equal(t, `Your Ring`+cursedLine, c.CursedRefusal(goldenItem(12, goldenRing, true)))
	lifted := goldenItem(13, goldenRing, true)
	lifted.Uncursed = true
	assert.Equal(t, ``, c.CursedRefusal(lifted), "an uncursed cursed item comes off")
}

func TestChooseWornSlot_RingsSkipACursedRing(t *testing.T) {
	seedGoldenSpecies(t)
	c := goldenChar(0, goldenMedium, false)
	id := 3000
	// rings lays out both ring slots afresh: applyLayout skips an 'e', so the
	// slots are cleared first or the previous layout's ring stays behind.
	rings := func(layout string) {
		c.Equipment.Ring, c.Equipment.Ring2 = items.Item{}, items.Item{}
		applyLayout([]*items.Item{&c.Equipment.Ring, &c.Equipment.Ring2}, layout, &id)
	}
	rings("Rr")
	choice, why := c.ChooseWornSlot(goldenItem(9000, goldenRing, false), 0)
	require.Equal(t, ``, why)
	require.Len(t, choice.Slots, 1)
	assert.Equal(t, `ring2`, choice.Slots[0].Key)
	assert.Equal(t, goldenIds([]items.Item{c.Equipment.Ring2}), goldenIds(choice.Displaced))

	rings("RR")
	_, why = c.ChooseWornSlot(goldenItem(9000, goldenRing, false), 0)
	assert.Equal(t, `Your Ring`+cursedLine, why, "all cursed: the shared line naming the Ring item")

	rings("er")
	choice, _ = c.ChooseWornSlot(goldenItem(9000, goldenRing, false), 0)
	assert.Equal(t, `ring`, choice.Slots[0].Key, "an empty slot is filled first, in today's order")
	assert.Empty(t, choice.Displaced)

	rings("dr")
	choice, why = c.ChooseWornSlot(goldenItem(9000, goldenRing, false), 0)
	require.Equal(t, ``, why)
	assert.Equal(t, `ring2`, choice.Slots[0].Key, "a disabled Ring is never written")
}

func TestChooseWornSlot_WristsSkipCursedWristsOnEveryArm(t *testing.T) {
	seedGoldenSpecies(t)
	wrists := func(c *Character) []*items.Item {
		return []*items.Item{&c.Equipment.Wrist1, &c.Equipment.Wrist2,
			&c.Equipment.ExtraWrist1, &c.Equipment.ExtraWrist2, &c.Equipment.ExtraWrist3, &c.Equipment.ExtraWrist4}
	}
	for _, tc := range []struct {
		extra  int
		layout string
		want   string
		why    string
	}{
		{0, "Ww", "wrist2", ""},
		{0, "WW", "", "Your Bracer" + cursedLine},
		{2, "WWww", "extrawrist1", ""},
		{2, "Wwww", "wrist2", ""},
		{4, "WWWWWw", "extrawrist4", ""},
		{4, "WWWWWW", "", "Your Bracer" + cursedLine},
	} {
		c := goldenChar(tc.extra, goldenMedium, false)
		id := 3100
		applyLayout(wrists(c)[:len(tc.layout)], tc.layout, &id)
		choice, why := c.ChooseWornSlot(goldenItem(9000, goldenBracer, false), 0)
		if tc.want == "" {
			assert.Equal(t, tc.why, why, "extra=%d %s", tc.extra, tc.layout)
			continue
		}
		require.Equal(t, ``, why, "extra=%d %s", tc.extra, tc.layout)
		assert.Equal(t, tc.want, choice.Slots[0].Key, "extra=%d %s", tc.extra, tc.layout)
	}
}

// handExpect is one arm count's outcome: the slot key that receives the item,
// or the refusal (substring) when slot is empty.
type handExpect struct {
	slot string
	why  string
}

// The hand table across arm counts (spec ruling 12 and 13). Every other hand
// holds a plain sword unless the row overrides it; arm numbers are 1 to 6.
func TestChooseWornSlot_HandsAcrossArmCounts(t *testing.T) {
	seedGoldenSpecies(t)
	rows := []struct {
		name      string
		dual      bool
		overrides map[int]rune
		item      items.ItemSpec
		want      map[int]handExpect // keyed by extra arms 0, 1, 2, 4
	}{
		{"cursed main hand, plain offhand, dual wielder", true, map[int]rune{1: 'S'}, goldenSword,
			map[int]handExpect{0: {"offhand", ""}, 1: {"offhand", ""}, 2: {"offhand", ""}, 4: {"offhand", ""}}},
		{"cursed main hand, empty offhand, dual wielder", true, map[int]rune{1: 'S', 2: 'e'}, goldenSword,
			map[int]handExpect{0: {"offhand", ""}, 1: {"offhand", ""}, 2: {"offhand", ""}, 4: {"offhand", ""}}},
		{"cursed main hand, no dual wield", false, map[int]rune{1: 'S'}, goldenSword,
			map[int]handExpect{0: {"", "Your Sword" + cursedLine}, 1: {"extraarm1", ""}, 2: {"extraarm1", ""}, 4: {"extraarm1", ""}}},
		{"every usable hand cursed", true, map[int]rune{1: 'S', 2: 'S', 3: 'S', 4: 'S', 5: 'S', 6: 'S'}, goldenSword,
			map[int]handExpect{0: {"", "Your Sword" + cursedLine}, 1: {"", "Your Sword" + cursedLine}, 2: {"", "Your Sword" + cursedLine}, 4: {"", "Your Sword" + cursedLine}}},
		{"cursed two-hander in the main hands", true, map[int]rune{1: 'G', 2: 'e'}, goldenSword,
			map[int]handExpect{0: {"", "Your Greatsword" + cursedLine}, 1: {"extraarm1", ""}, 2: {"extraarm1", ""}, 4: {"extraarm1", ""}}},
		{"shield over a cursed offhand item", false, map[int]rune{2: 'B'}, goldenShield,
			map[int]handExpect{0: {"", "Your Buckler" + cursedLine}, 1: {"extraarm1", ""}, 2: {"extraarm1", ""}, 4: {"extraarm1", ""}}},
		{"shield over a cursed offhand item, every extra arm cursed", false, map[int]rune{2: 'B', 3: 'S', 4: 'S', 5: 'S', 6: 'S'}, goldenShield,
			map[int]handExpect{0: {"", "Your Buckler" + cursedLine}, 1: {"", "Your Buckler" + cursedLine}, 2: {"", "Your Buckler" + cursedLine}, 4: {"", "Your Buckler" + cursedLine}}},
		{"shield beside a two-hander, every hand full", false, map[int]rune{1: 'g', 2: 'e'}, goldenShield,
			map[int]handExpect{0: {"", "no room for a shield"}, 1: {"extraarm1", ""}, 2: {"extraarm2", ""}, 4: {"extraarm4", ""}}},
		{"shield beside a two-hander, the last hand cursed", false, map[int]rune{1: 'g', 2: 'e', 4: 'S', 6: 'S'}, goldenShield,
			map[int]handExpect{0: {"", "no room for a shield"}, 1: {"extraarm1", ""}, 2: {"extraarm1", ""}, 4: {"extraarm3", ""}}},
		{"shield beside a two-hander, every candidate cursed", false, map[int]rune{1: 'g', 2: 'e', 3: 'S', 4: 'S', 5: 'S', 6: 'S'}, goldenShield,
			map[int]handExpect{0: {"", "no room for a shield"}, 1: {"", "Your Sword" + cursedLine}, 2: {"", "Your Sword" + cursedLine}, 4: {"", "Your Sword" + cursedLine}}},
		{"two-hander, cursed item in the cheapest pair", false, map[int]rune{1: 'S'}, goldenGreat,
			map[int]handExpect{0: {"", "Your Sword" + cursedLine}, 1: {"", "Your Sword" + cursedLine}, 2: {"extraarm1", ""}, 4: {"extraarm1", ""}}},
	}
	for _, row := range rows {
		for _, extra := range []int{0, 1, 2, 4} {
			c := goldenChar(extra, goldenMedium, row.dual)
			ov := map[int]rune{}
			for arm, code := range row.overrides {
				if arm <= 2+extra {
					ov[arm] = code
				}
			}
			fillHands(c, ov)
			before := goldenSnapshot(c)
			choice, why := c.ChooseWornSlot(goldenItem(9000, row.item, false), 0)
			want := row.want[extra]
			assert.Equal(t, before, goldenSnapshot(c), "%s at %d arms: ChooseWornSlot must not mutate", row.name, 2+extra)
			if want.slot == "" {
				assert.Contains(t, why, want.why, "%s at %d arms", row.name, 2+extra)
				assert.Empty(t, choice.Slots, "%s at %d arms", row.name, 2+extra)
				continue
			}
			if assert.Equal(t, ``, why, "%s at %d arms", row.name, 2+extra) {
				assert.Equal(t, want.slot, choice.Slots[0].Key, "%s at %d arms", row.name, 2+extra)
			}
		}
	}
}

// With a cursed main-hand stray behind a plain two-hander, today's 1H swap
// took the stray off unchecked (spec E2); now the curse holds it.
func TestChooseWornSlot_StrayBehindATwoHanderIsCheckedToo(t *testing.T) {
	seedGoldenSpecies(t)
	c := goldenChar(0, goldenMedium, false)
	id := 3200
	applyLayout(handSlotsInArmOrder(c), "gB", &id)
	_, why := c.ChooseWornSlot(goldenItem(9000, goldenSword, false), 0)
	assert.Equal(t, `Your Buckler`+cursedLine, why)
}

// Too many hands is refused by the helper as well as by Wear, so the scorer
// never offers a slot Wear will refuse.
func TestChooseWornSlot_TooManyHands(t *testing.T) {
	seedGoldenSpecies(t)
	c := goldenChar(4, goldenSmall, false)
	_, why := c.ChooseWornSlot(goldenItem(9000, goldenGreat, false), 0)
	assert.Equal(t, `That requires too many hands.`, why)
}

// An unregistered species is read as Medium rather than a nil dereference.
func TestHandsRequired_UnregisteredSpeciesIsMedium(t *testing.T) {
	c := &Character{SpeciesId: 424242}
	assert.Equal(t, 2, c.HandsRequired(goldenItem(9000, goldenGreat, false)))
}

func TestWearInArm(t *testing.T) {
	seedGoldenSpecies(t)

	// A cursed arm-5 item refuses the named arm, though arms 1 to 4 and 6
	// are plain, and nothing moves (ruling 12).
	c := goldenChar(4, goldenMedium, false)
	fillHands(c, map[int]rune{5: 'S'})
	before := goldenSnapshot(c)
	_, worn, why := c.WearInArm(goldenItem(9000, goldenSword, false), 5)
	assert.False(t, worn)
	assert.Equal(t, `Your Sword`+cursedLine, why)
	assert.Equal(t, before, goldenSnapshot(c))

	c = goldenChar(2, goldenMedium, false)
	_, _, why = c.WearInArm(goldenItem(9000, goldenSword, false), 5)
	assert.Equal(t, `You don't have arm 5.`, why)

	// The five shape refusals keep the player's wording.
	c = goldenChar(0, goldenMedium, false)
	_, _, why = c.WearInArm(goldenItem(9000, armourSpec("cap", items.Head), false), 1)
	assert.Equal(t, `You can only wield weapons or shields in arm slots.`, why)
	_, _, why = c.WearInArm(goldenItem(9000, goldenShield, false), 1)
	assert.Equal(t, `You can't put a shield in your primary weapon hand (arm 1).`, why)
	_, _, why = c.WearInArm(goldenItem(9000, goldenSword, false), 3)
	assert.Equal(t, `You don't have arm 3.`, why)
	_, _, why = c.WearInArm(goldenItem(9000, goldenGreat, false), 2)
	assert.Equal(t, `A two-handed weapon needs a pair of arms. Try arm 1, 3, or 5.`, why)
	c = goldenChar(1, goldenMedium, false)
	_, _, why = c.WearInArm(goldenItem(9000, goldenGreat, false), 3)
	assert.Equal(t, `That arm doesn't have a partner for a two-handed weapon.`, why)

	// MinStrength runs before the arm's shape refusals.
	c = goldenChar(0, goldenMedium, false)
	c.Stats.Strength.ValueAdj = 1
	heavy := goldenSword
	heavy.MinStrength = 500
	_, _, why = c.WearInArm(goldenItem(9000, heavy, false), 3)
	assert.Equal(t, `You aren't strong enough to handle Sword.`, why)

	// Arm 2 beside a two-hander takes the two-hander off.
	c = goldenChar(0, goldenMedium, false)
	id := 4500
	applyLayout(handSlotsInArmOrder(c), "ge", &id)
	ret, worn, why := c.WearInArm(goldenItem(9000, goldenShield, false), 2)
	require.True(t, worn, why)
	assert.Equal(t, "4501,", goldenIds(ret))
	assert.Equal(t, 0, c.Equipment.Weapon.ItemId)
	assert.Equal(t, 9000, c.Equipment.Offhand.ItemId)

	// A cursed two-hander beside arm 2, and a cursed second slot under a
	// two-hander named at arm 1, both refuse with the shared line.
	c = goldenChar(0, goldenMedium, false)
	applyLayout(handSlotsInArmOrder(c), "Ge", &id)
	_, _, why = c.WearInArm(goldenItem(9000, goldenShield, false), 2)
	assert.Equal(t, `Your Greatsword`+cursedLine, why)
	c = goldenChar(0, goldenMedium, false)
	applyLayout(handSlotsInArmOrder(c), "sB", &id)
	_, _, why = c.WearInArm(goldenItem(9000, goldenGreat, false), 1)
	assert.Equal(t, `Your Buckler`+cursedLine, why)
}

func TestArmLabel(t *testing.T) {
	seedGoldenSpecies(t)
	c := goldenChar(1, goldenMedium, false)
	assert.Equal(t, `wielded`, c.ArmLabel(1))
	assert.Equal(t, `offhand`, c.ArmLabel(2))
	assert.Equal(t, `extra arm 1`, c.ArmLabel(3))
	assert.Equal(t, ``, c.ArmLabel(4), "a half pair has no arm 4")
	assert.Equal(t, ``, c.ArmLabel(0))
}
