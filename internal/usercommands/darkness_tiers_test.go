package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// TestActeeDefenceLine_ThreeTierSightNotBinary pins the actual defect Task 10
// fixes: canSeeInDark was BINARY, so acteeDefenceLine routed every reader who
// failed it through messaging.Anonymize, which knows only the single word "a
// figure". A SightShapes reader (infrared, makes out shapes) and a SightNone
// reader (fully blind) both read "a figure" -- the blind reader should read
// "something" instead. Mirror of internal/mobcommands/darkness_tiers_test.go,
// which fixed the mob-attacker twin in e630a7876.
func TestActeeDefenceLine_ThreeTierSightNotBinary(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	restoreBiomes := rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"cave": {BiomeId: "cave", Name: "Cave", Symbol: ".", DarkArea: true, MovementCost: 1},
	})
	defer restoreBiomes()

	restoreConditions := conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		9001: {ConditionId: 9001, Name: "Test Infrared", RoundInterval: 1, TriggerCount: 1, Flags: []conditions.Flag{conditions.InfraredVision}},
	})
	defer restoreConditions()

	attacker := users.GetByUserId(1)
	require.NotNil(t, attacker)

	darkRoom := rooms.LoadRoom(2)
	require.NotNil(t, darkRoom)
	darkRoom.Biome = "cave"
	require.Zero(t, darkRoom.GetVisibility(), "fixture room must be unlit for this lane")

	text := `<ansi fg="username">Aliceia</ansi> catches your blow on its shield!`

	t.Run("blind reads something, not a figure", func(t *testing.T) {
		target := users.GetByUserId(2)
		require.NotNil(t, target)
		line := acteeDefenceLine(target, darkRoom, messaging.CategoryBash, text, attacker.Character.Name)
		require.Contains(t, strings.ToLower(line.Text), "something",
			"a fully blind reader must read \"something\", not the shapes-tier word")
		require.NotContains(t, strings.ToLower(line.Text), "a figure")
	})

	t.Run("infrared reads a figure, not something", func(t *testing.T) {
		target := users.GetByUserId(2)
		require.NotNil(t, target)
		require.True(t, target.Character.Conditions.AddCondition(9001, true))
		line := acteeDefenceLine(target, darkRoom, messaging.CategoryBash, text, attacker.Character.Name)
		require.Contains(t, strings.ToLower(line.Text), "a figure")
	})
}
