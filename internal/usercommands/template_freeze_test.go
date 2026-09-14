package usercommands

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Templates read Go field and method names at runtime, so a rename the
// compiler accepts can still break a template silently. These renders pin the
// four templates that read condition names (slice 2 of the conditions
// unification).

// useDogmudTemplates points templates.Process at the real DOGMud world's
// template files. This package's TestMain (usercommands_test.go) never
// registers a templates filesystem, so templates.readFile's registered-fs
// loop is empty and (per its documented zero-value behaviour, confirmed in
// internal/templates/process_test.go's TestMain comment) that makes every
// lookup silently "succeed" with empty content rather than fall through to
// disk. templates.SetFSForTest scopes a real os.DirFS over the dogmud world
// root to just the calling test (restored on cleanup), so Process() reads
// the actual shipped templates here without leaking a permanent
// templates.RegisterFS registration into every later test in this package's
// shared test binary — an earlier version of this helper used RegisterFS
// directly and broke TestHelp/TestMap/TestHelpSubCommands/TestHelpDeep,
// which depend on the untouched vacuous-success default because they never
// set FilePaths.DataFiles themselves.
func useDogmudTemplates(t *testing.T) {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Join(filepath.Dir(here), "..", "..", "_datafiles", "world", "dogmud")
	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = configs.ConfigString(root)
	configs.SetConfigForTest(t, cfg)
	templates.SetFSForTest(t, os.DirFS(root).(fs.ReadFileFS))
}

func TestTemplateFreeze_ConditionsListReadsPermanent(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	defer buffs.SeedConditionRecordsForTest()()
	useDogmudTemplates(t)

	out, err := templates.Process("character/conditions", []conditionEntry{
		{Name: "Bleeding (2)", Description: "Wounds seeping blood.", RoundsLeft: 4},
		{Name: "Stoneskin", Description: "Skin like rock.", PermaBuff: true},
	}, 0)
	require.NoError(t, err)
	assert.Contains(t, out, "Bleeding (2)")
	assert.Contains(t, out, "Stoneskin")
}

func TestTemplateFreeze_StatusReadsTheBrokenLimbRecord(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	useDogmudTemplates(t)

	user, _ := getTestUserAndRoom(t)
	restore := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		83: {BuffId: 83, Name: "Broken Limb", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 10},
	})
	defer restore()
	user.Character.Buffs.Validate(true)
	require.True(t, user.Character.Buffs.AddBuff(83, false))

	out, err := templates.Process("character/status", user, user.UserId)
	require.NoError(t, err)
	assert.Contains(t, out, "Broken limb", "status reads .Character.Buffs.HasBuff and .TriggersLeft")
}

func TestTemplateFreeze_IdentifyReadsConditionIds(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	useDogmudTemplates(t)
	restore := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		940: {BuffId: 940, Name: "Probe Glow", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 5},
		941: {BuffId: 941, Name: "Probe Rend", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 5},
	})
	defer restore()

	spec := items.ItemSpec{ItemId: 99940, Name: "probe", BuffIds: []int{940}}
	spec.Damage.CritBuffIds = []int{941}
	item := items.Item{ItemId: 99940}
	out, err := templates.Process("descriptions/identify", struct {
		Item     *items.Item
		ItemSpec *items.ItemSpec
	}{&item, &spec}, 0)
	require.NoError(t, err)
	assert.Contains(t, out, "Probe Glow", "identify reads $spec.BuffIds")
	assert.Contains(t, out, "Probe Rend", "identify reads $spec.Damage.CritBuffIds")
}

func TestTemplateFreeze_SpeciesHelpReadsConditionIds(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	useDogmudTemplates(t)
	restore := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		942: {BuffId: 942, Name: "Probe Hide", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 5},
	})
	defer restore()

	out, err := templates.Process("help/species", []species.Species{{Name: "Probe", BuffIds: []int{942}}}, 0)
	require.NoError(t, err)
	assert.Contains(t, out, "Probe Hide", "species help reads $speciesInfo.BuffIds")
}

// TestWireFreeze_SpellCategoryStillGroupsBuffEffectType pins spells.go:32's
// `case "buff", "shield", "purge":` inside spellCategory, the other string
// literal reading effect_type: buff (see wire_freeze_test.go at the repo
// root and internal/hooks/wire_freeze_test.go for the dispatch-side ones).
// It decides which sort bucket the `spells` command lists a spell under.
//
// A Neutral-type spell is the probe that actually distinguishes this case
// from its fallthrough: an effect_type: buff spell matches the case FIRST
// and returns 2 regardless of Type, but if that case literal ever stops
// matching "buff", a Neutral-type spell falls through to the `sp.Type ==
// spells.Neutral` branch below and returns 0 instead — a real, visible
// change to where the spell lists in the `spells` command.
func TestWireFreeze_SpellCategoryStillGroupsBuffEffectType(t *testing.T) {
	got := spellCategory(&spells.SpellData{EffectType: "buff", Type: spells.Neutral})
	assert.Equal(t, 2, got,
		"an effect_type: buff spell must still sort into the buff/shield/purge display category (usercommands/spells.go's spellCategory)")
}
