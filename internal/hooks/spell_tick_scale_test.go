package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spellTickScaleWandId is a seeded weapon with a spell damage multiplier.
const spellTickScaleWandId = 7130

func seedSpellTickScaleWand(t *testing.T) {
	t.Helper()
	t.Cleanup(items.SeedItemsForTest(map[int]*items.ItemSpec{
		spellTickScaleWandId: {ItemId: spellTickScaleWandId, Name: "Test Wand", Type: items.Weapon, SpellDamageMultiplier: 1.5},
	}))
}

func TestSpellTickScale_SkillZeroNoWeaponIsTheSkillBase(t *testing.T) {
	caster := characters.New()
	// New seeds every skill at rank 1 (initAllSkills); zero removes it.
	caster.SetSkill("spellcasting", 0)
	assert.Equal(t, combat.SkillMultiplier(0), spellTickScale(caster))
	assert.InDelta(t, 1.0, spellTickScale(caster), 1e-9, "SkillMultiplierBase defaults to 1.0")
	assert.InDelta(t, 1.0, spellTickScale(nil), 1e-9, "no caster is 1.0")
}

func TestSpellTickScale_WeaponMultiplies(t *testing.T) {
	seedSpellTickScaleWand(t)
	caster := characters.New()
	caster.SetSkill("spellcasting", 20)
	bare := spellTickScale(caster)
	caster.Equipment.Weapon = items.Item{ItemId: spellTickScaleWandId}
	assert.InDelta(t, bare*1.5, spellTickScale(caster), 1e-9,
		"an unmutated caster's gear effectiveness is 1.0, so the weapon's 1.5 applies whole")
	assert.Greater(t, bare, 1.0, "precondition: skill 20 scales above the base")
}

// One formula for both: a mob caster used to skip the weapon (self-cast) or
// skip scaling altogether (mob on mob).
func TestSpellTickScale_PlayerAndMobCastersMatch(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	seedSpellTickScaleWand(t)
	player := users.GetByUserId(1).Character
	mob := &mobs.GetInstance(100).Character
	for _, c := range []*characters.Character{player, mob} {
		c.SetSkill("spellcasting", 30)
		c.Equipment.Weapon = items.Item{ItemId: spellTickScaleWandId}
	}
	assert.Equal(t, spellTickScale(player), spellTickScale(mob))
	assert.Greater(t, spellTickScale(mob), combat.SkillMultiplier(30), "the mob gets the weapon too")
}

// Spell-path conditions for applySpellCondition: a tick_pool heal, a plain
// record, and a magnitude light.
const (
	spellPathTickConditionId  = 7131
	spellPathPlainConditionId = 7132
	spellPathGlowConditionId  = 7133
)

func seedSpellPathConditions(t *testing.T) {
	t.Helper()
	t.Cleanup(seedAllRegistries())
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		spellPathTickConditionId: {ConditionId: spellPathTickConditionId, Name: "Test Surge",
			RoundInterval: 1, TriggerCount: 5, TickPool: "health", TickPercent: 0.05},
		spellPathPlainConditionId: {ConditionId: spellPathPlainConditionId, Name: "Test Plain",
			RoundInterval: 1, TriggerCount: 4},
		spellPathGlowConditionId: {ConditionId: spellPathGlowConditionId, Name: "Test Glow",
			RoundInterval: 1, TriggerCount: 4,
			Effects: map[conditions.EffectKind]conditions.EffectValue{conditions.EffectLightStrength: {UsesMagnitude: true}},
			Flags:   []conditions.Flag{conditions.Adjustable}},
	}))
	events.DrainQueuedMobConditionsForTest(0)
}

func TestApplySpellCondition_TickPoolQueuesTheCasterScale(t *testing.T) {
	seedSpellPathConditions(t)
	seedSpellTickScaleWand(t)
	spell := &spells.SpellData{SpellId: "test-surge", PrimaryStat: "willpower"}
	caster := characters.New()
	caster.SetSkill("spellcasting", 40)
	caster.Equipment.Weapon = items.Item{ItemId: spellTickScaleWandId}
	want := spellTickScale(caster)
	require.Greater(t, want, 1.0, "precondition: the caster scale is not the default")

	targets := []struct {
		name  string
		door  spellConditionTarget
		drain func() []events.Condition
	}{
		{"player target", users.GetByUserId(1), func() []events.Condition { return events.DrainQueuedConditionsForTest(1) }},
		{"mob target", mobs.GetInstance(100), func() []events.Condition { return events.DrainQueuedMobConditionsForTest(100) }},
	}
	for _, tg := range targets {
		t.Run(tg.name, func(t *testing.T) {
			applySpellCondition(tg.door, spell, caster, spellPathTickConditionId)
			q := tg.drain()
			require.Len(t, q, 1)
			assert.Equal(t, want, q[0].TickScale, "a tick_pool condition carries the caster's scale")

			applySpellCondition(tg.door, spell, caster, spellPathPlainConditionId)
			q = tg.drain()
			require.Len(t, q, 1)
			assert.Zero(t, q[0].TickScale, "a non-ticking condition carries no scale")

			applySpellCondition(tg.door, spell, caster, spellPathGlowConditionId)
			q = tg.drain()
			require.Len(t, q, 1)
			assert.NotZero(t, q[0].Magnitude, "a magnitude light still queues its magnitude")
			assert.Zero(t, q[0].TickScale)
		})
	}
}
