package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// Conditions unification slice 2 renames buffs to conditions in Go and in
// player-facing text, and must change NOTHING on disk or on the wire (owner
// disk/wire rule, 2026-09-14). These tests pin the literal keys, values and
// strings. They were written before the first rename and must stay green
// through the slice; slice 3 is the one that deliberately changes them.

func TestWireFreeze_CharacterSaveKeys(t *testing.T) {
	defer buffs.SeedConditionRecordsForTest()()

	c := characters.Character{}
	c.Buffs.Validate(true)
	require.True(t, c.Buffs.AddBuffMagnitude(buffs.BuffIdBleeding, 3, -2))
	require.True(t, c.Buffs.AddBuffMagnitude(buffs.BuffIdBleeding, 5, -3))
	c.Buffs.List = append(c.Buffs.List, &buffs.Buff{BuffId: buffs.BuffIdWarcry, PermaBuff: true, TriggersLeft: 1})

	out, err := yaml.Marshal(&c)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(out, &doc))
	held, ok := doc["buffs"].(map[any]any)
	require.True(t, ok, "the character save must keep the `buffs:` key; got top-level keys from:\n%s", out)
	list, ok := held["list"].([]any)
	require.True(t, ok, "the held records must stay under `list:`")
	require.Len(t, list, 2)

	bleed := list[0].(map[any]any)
	assert.EqualValues(t, buffs.BuffIdBleeding, bleed["buffid"], "a record's id must stay under `buffid:`")
	assert.Contains(t, bleed, "triggersleft")
	assert.Contains(t, bleed, "stacks", "stacks must stay under `stacks:`")
	stack := bleed["stacks"].([]any)[0].(map[any]any)
	assert.Contains(t, stack, "roundsleft")
	assert.Contains(t, stack, "amount")

	warcry := list[1].(map[any]any)
	assert.Equal(t, true, warcry["permabuff"], "a permanent record must stay under `permabuff:`")

	var back characters.Character
	require.NoError(t, yaml.Unmarshal(out, &back))
	require.Len(t, back.Buffs.List, 2)
	assert.Equal(t, buffs.BuffIdBleeding, back.Buffs.List[0].BuffId)
	assert.Len(t, back.Buffs.List[0].Stacks, 2)
	assert.True(t, back.Buffs.List[1].PermaBuff)
}

func TestWireFreeze_ConditionFileKeys(t *testing.T) {
	var s buffs.BuffSpec
	doc := "buffid: 950\nname: Probe\ntriggerrate: 1 round\ntriggercount: 2\n" +
		"start_remove_buffs: [3]\neffects:\n  damage_mult: magnitude\nflags:\n  - stacking\n" +
		"tick_pool: health\ntick_from_magnitude: true\n"
	require.NoError(t, yaml.Unmarshal([]byte(doc), &s))
	assert.Equal(t, 950, s.BuffId, "`buffid:` must still name the record")
	assert.Equal(t, []int{3}, s.StartRemoveBuffs, "`start_remove_buffs:` must still parse")
	assert.True(t, s.Effects[buffs.EffectDamageMult].UsesMagnitude)
	assert.Equal(t, []buffs.Flag{buffs.Stacking}, s.Flags)
}

func TestWireFreeze_ShippedConditionFilesLoadInBothWorlds(t *testing.T) {
	// This package (root, package main) has no TestMain; every test that
	// exercises a mudlog call sets up the logger itself (see
	// boot_smoke_test.go). buffs.LoadDataFiles logs on success and panics on
	// a nil logger if nothing else in this test binary run has set one up yet.
	mudlog.SetupLogger(nil, `LOW`, ``, false)

	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	for _, world := range []string{"dogmud", "default"} {
		t.Run(world, func(t *testing.T) {
			cfg := configs.GetConfig()
			cfg.FilePaths.DataFiles = configs.ConfigString(filepath.Join(filepath.Dir(here), "_datafiles", "world", world))
			cfg.Network.LogoutRounds = 3
			configs.SetConfigForTest(t, cfg)
			restore := buffs.SeedBuffsForTest(nil)
			defer restore()
			buffs.LoadDataFiles()
			assert.NotEmpty(t, buffs.GetAllBuffIds(), "the %s world's condition files must load under today's keys", world)
		})
	}
}

func TestWireFreeze_SpeciesAndSpellValues(t *testing.T) {
	var sp species.Species
	require.NoError(t, yaml.Unmarshal([]byte("name: Probe\nbuffids: [7]\n"), &sp))
	assert.Equal(t, []int{7}, sp.BuffIds, "species `buffids:` must still parse")

	var spell spells.SpellData
	require.NoError(t, yaml.Unmarshal([]byte("spellid: probe\nname: Probe\neffect_type: buff\nbuff_ids: [3]\n"), &spell))
	assert.Equal(t, "buff", spell.EffectType, "the `effect_type: buff` value is wire and stays")
	assert.Equal(t, []int{3}, spell.BuffIds, "`buff_ids:` must still parse")
}

func TestWireFreeze_MessagingCategoryStrings(t *testing.T) {
	assert.Equal(t, "buff-apply", messaging.CategoryBuffApply.String())
	assert.Equal(t, "buff-expire", messaging.CategoryBuffExpire.String())
}
