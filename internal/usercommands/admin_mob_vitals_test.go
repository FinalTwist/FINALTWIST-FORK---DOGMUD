package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/require"
)

// Movement parity 4b: a playtester watches a mob tire through `mob schedule`.
func TestMobVitalsLine(t *testing.T) {
	m := &mobs.Mob{InstanceId: 5}
	m.Character.Stamina = 37
	m.Character.StaminaMax.Value = 120
	m.Character.ActionPointsMax.Value = 200
	line := mobVitalsLine(m, 10)
	require.True(t, strings.Contains(line, "stamina:         37 / 120"), line)
	require.True(t, strings.Contains(line, "action points:   200 / 200"), line)
	require.True(t, m.Character.ActionPointsSettled, "the readout shows settled points")
}
