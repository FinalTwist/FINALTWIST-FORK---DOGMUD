package web

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"
	"text/template"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/stretchr/testify/require"
)

// adminHtmlDir resolves _datafiles/html/admin the same way the real handlers
// do (configs.GetFilePathsConfig().AdminHtml), without depending on
// config.yaml: a test binary never loads it (see internal/usercommands
// u8_help_test.go's useU8DataFilesAt for the same pattern).
func adminHtmlDir(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(here), "..", "..", "_datafiles", "html", "admin")
}

// TestAdminItemTemplateExecutesWithConditionIds pins item.data.html against the
// slice 2 rename: itemSpec.ConditionIds / WornConditionIds /
// Damage.CritConditionIds and buffSpecs' ConditionId field are all read by
// this template at runtime, with no compile-time check. A stale .BuffId /
// .BuffIds reference renders nothing until an admin actually opens an item
// that has a condition on it (see the review that caught 61 such refs across
// the admin html pages). Executes the exact file and funcMap itemData uses.
func TestAdminItemTemplateExecutesWithConditionIds(t *testing.T) {
	cleanup := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		950: {ConditionId: 950, Name: `Probe Glow`, TriggerRate: `1 round`, RoundInterval: 1, TriggerCount: 5},
	})
	defer cleanup()

	tmpl, err := template.New(`item.data.html`).Funcs(funcMap).ParseFiles(filepath.Join(adminHtmlDir(t), `items`, `item.data.html`))
	require.NoError(t, err)

	itemSpec := items.ItemSpec{
		ItemId:           950,
		Name:             `Probe Blade`,
		ConditionIds:     []int{950},
		WornConditionIds: []int{950},
	}
	itemSpec.Damage.CritConditionIds = []int{950}

	tplData := map[string]any{}
	tplData[`itemSpec`] = itemSpec

	conditionSpecs := []conditions.ConditionSpec{}
	for _, conditionId := range conditions.GetAllConditionIds() {
		if b := conditions.GetConditionSpec(conditionId); b != nil {
			conditionSpecs = append(conditionSpecs, *b)
		}
	}
	tplData[`buffSpecs`] = conditionSpecs
	tplData[`itemTypes`] = items.ItemTypes()
	tplData[`itemSubtypes`] = items.ItemSubtypes()

	var out bytes.Buffer
	require.NoError(t, tmpl.Execute(&out, tplData), `item.data.html must render an item carrying condition ids without error`)
	require.Contains(t, out.String(), `Probe Glow`, `the crit/use/worn condition checkbox lists must render the seeded condition's name`)
}

// TestAdminMutatorTemplateExecutesWithConditionIds pins mutator.data.html the
// same way: mutatorSpec.PlayerConditionIds / MobConditionIds /
// NativeConditionIds and buffSpecs' ConditionId. The template also used to
// read a nonexistent .raceInfo key for all three checkbox lists (a
// pre-existing bug the rename's own field renames exposed as unreachable
// dead code once .raceInfo.BuffIds started erroring); fixed to read the
// bound $mutator instead.
func TestAdminMutatorTemplateExecutesWithConditionIds(t *testing.T) {
	cleanup := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		951: {ConditionId: 951, Name: `Probe Aura`, TriggerRate: `1 round`, RoundInterval: 1, TriggerCount: 5},
	})
	defer cleanup()

	tmpl, err := template.New(`mutator.data.html`).Funcs(funcMap).ParseFiles(filepath.Join(adminHtmlDir(t), `mutators`, `mutator.data.html`))
	require.NoError(t, err)

	mutSpec := mutators.MutatorSpec{
		MutatorId:          `probe`,
		PlayerConditionIds: []int{951},
		MobConditionIds:    []int{951},
		NativeConditionIds: []int{951},
	}

	tplData := map[string]any{}
	tplData[`mutatorSpec`] = mutSpec

	conditionSpecs := []conditions.ConditionSpec{}
	for _, conditionId := range conditions.GetAllConditionIds() {
		if b := conditions.GetConditionSpec(conditionId); b != nil {
			conditionSpecs = append(conditionSpecs, *b)
		}
	}
	tplData[`buffSpecs`] = conditionSpecs
	tplData[`colorPatterns`] = colorpatterns.GetColorPatternNames()

	var out bytes.Buffer
	require.NoError(t, tmpl.Execute(&out, tplData), `mutator.data.html must render a mutator carrying condition ids without error`)
	require.Contains(t, out.String(), `Probe Aura`, `the player/mob/native condition checkbox lists must render the seeded condition's name`)
}

// buildConditionSpecsForTest mirrors the buffSpecs slice every admin data
// handler (mobData, roomData, speciesData, itemData, mutatorData) builds from
// the live condition registry, so a test can hand a template the exact shape
// it expects at ".buffSpecs" (a template map key, unaffected by the slice 2
// rename per the disk/wire rule).
func buildConditionSpecsForTest() []conditions.ConditionSpec {
	conditionSpecs := []conditions.ConditionSpec{}
	for _, conditionId := range conditions.GetAllConditionIds() {
		if b := conditions.GetConditionSpec(conditionId); b != nil {
			conditionSpecs = append(conditionSpecs, *b)
		}
	}
	return conditionSpecs
}

// TestAdminMobTemplateExecutesWithConditionIds pins mob.data.html against the
// slice 2 rename: mobInfo.ConditionIds and buffSpecs' ConditionId field are
// read by this template at runtime (review of Tasks 2 and 3 caught the same
// class of stale .BuffId/.BuffIds reference here as in item.data.html and
// mutator.data.html; this test closes the one page that review left
// unrendered).
func TestAdminMobTemplateExecutesWithConditionIds(t *testing.T) {
	cleanup := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		952: {ConditionId: 952, Name: `Probe Claw`, TriggerRate: `1 round`, RoundInterval: 1, TriggerCount: 5},
	})
	defer cleanup()

	tmpl, err := template.New(`mob.data.html`).Funcs(funcMap).ParseFiles(filepath.Join(adminHtmlDir(t), `mobs`, `mob.data.html`))
	require.NoError(t, err)

	mob := mobs.Mob{
		MobId:        1,
		ConditionIds: []int{952},
		Character:    characters.Character{Name: `Probe Mob`},
	}

	tplData := map[string]any{}
	tplData[`mobInfo`] = mob
	tplData[`mobShop`] = map[string]characters.Shop{`Items`: {}, `Buffs`: {}, `Mercenaries`: {}, `Pets`: {}}
	tplData[`characterInfo`] = &mob.Character
	tplData[`allZoneNames`] = []string{}
	tplData[`allSpecies`] = []species.Species{}
	tplData[`activityLevels`] = []int{}
	tplData[`dropChances`] = []int{}
	tplData[`allMobGroups`] = []string{}
	tplData[`buffSpecs`] = buildConditionSpecsForTest()

	var out bytes.Buffer
	require.NoError(t, tmpl.Execute(&out, tplData), `mob.data.html must render a mob carrying condition ids without error`)
	require.Contains(t, out.String(), `Probe Claw`, `the mob condition checkbox list must render the seeded condition's name`)
}

// TestAdminRoomTemplateExecutesWithConditionIds pins room.data.html against
// the slice 2 rename: spawnInfo.ConditionIds, exitInfo.Lock.TrapConditionIds
// and buffSpecs' ConditionId are all read by this template at runtime. It
// also exercises the room's zone-level gate, `.zoneConfig` (RoomId,
// Mutators): rooms.Room carries no ZoneConfig field of its own, so the
// template's old `$room.ZoneConfig...` reads always errored regardless of
// any condition id, before ever reaching the condition checkboxes below them
// (verified by probe: reverting roomData's `zoneConfig` wiring reproduces
// "can't evaluate field ZoneConfig in type *rooms.Room" at template.Execute).
// Fixed alongside this test since the room page could not otherwise be
// rendered at all to prove the condition ids render clean.
func TestAdminRoomTemplateExecutesWithConditionIds(t *testing.T) {
	cleanup := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		953: {ConditionId: 953, Name: `Probe Spawn`, TriggerRate: `1 round`, RoundInterval: 1, TriggerCount: 5},
		954: {ConditionId: 954, Name: `Probe Trap`, TriggerRate: `1 round`, RoundInterval: 1, TriggerCount: 5},
	})
	defer cleanup()

	tmpl, err := template.New(`room.data.html`).Funcs(funcMap).ParseFiles(filepath.Join(adminHtmlDir(t), `rooms`, `room.data.html`))
	require.NoError(t, err)

	room := &rooms.Room{
		RoomId: 1,
		Zone:   `TestZone`,
		Exits: map[string]exit.RoomExit{
			`north`: {
				RoomId: 2,
				Lock: gamelock.Lock{
					Difficulty:       1,
					TrapConditionIds: []int{954},
				},
			},
		},
		SpawnInfo: []rooms.SpawnInfo{
			{MobId: 1, ConditionIds: []int{953}},
		},
	}

	tplData := map[string]any{}
	tplData[`roomInfo`] = room
	tplData[`zoneConfig`] = &rooms.ZoneConfig{RoomId: 1}
	tplData[`biomes`] = []rooms.BiomeInfo{}
	tplData[`allSkillNames`] = []string{}
	tplData[`allSlotTypes`] = []string{}
	tplData[`mapDirections`] = []string{}
	tplData[`mutSpecs`] = []mutators.MutatorSpec{}
	tplData[`buffSpecs`] = buildConditionSpecsForTest()

	var out bytes.Buffer
	require.NoError(t, tmpl.Execute(&out, tplData), `room.data.html must render a room carrying condition ids without error`)
	require.Contains(t, out.String(), `Probe Spawn`, `the mob-spawn condition checkbox list must render the seeded condition's name`)
	require.Contains(t, out.String(), `Probe Trap`, `the exit-lock trap condition checkbox list must render the seeded condition's name`)
}

// TestAdminSpeciesTemplateExecutesWithConditionIds pins species.data.html
// against the slice 2 rename: speciesInfo.ConditionIds and buffSpecs'
// ConditionId are read by this template at runtime.
func TestAdminSpeciesTemplateExecutesWithConditionIds(t *testing.T) {
	cleanup := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		955: {ConditionId: 955, Name: `Probe Hide`, TriggerRate: `1 round`, RoundInterval: 1, TriggerCount: 5},
	})
	defer cleanup()

	tmpl, err := template.New(`species.data.html`).Funcs(funcMap).ParseFiles(filepath.Join(adminHtmlDir(t), `species`, `species.data.html`))
	require.NoError(t, err)

	speciesInfo := species.Species{
		SpeciesId:    1,
		Name:         `Probe Species`,
		ConditionIds: []int{955},
	}

	tplData := map[string]any{}
	tplData[`speciesInfo`] = speciesInfo
	tplData[`buffSpecs`] = buildConditionSpecsForTest()
	tplData[`allSlotTypes`] = []string{}

	var out bytes.Buffer
	require.NoError(t, tmpl.Execute(&out, tplData), `species.data.html must render a species carrying condition ids without error`)
	require.Contains(t, out.String(), `Probe Hide`, `the species natural-condition checkbox list must render the seeded condition's name`)
}
