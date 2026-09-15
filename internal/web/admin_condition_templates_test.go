package web

import (
	"bytes"
	"path/filepath"
	"runtime"
	"testing"
	"text/template"

	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutators"
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
