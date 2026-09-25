package actions

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// A harmful multi-target spell aimed at a named person is a PvP act exactly
// as a single-target one is. HarmSingle and HarmArea both asked Room.CanPvp;
// HarmMulti's named-target branch appended the player unasked, so with PvP
// off a chain-style spell could still open on another player.
func TestInitiateCast_HarmMulti_NamedPlayerHonoursPvp(t *testing.T) {
	const casterId, targetId = 1, 7312

	cfg := configs.GetConfig()
	cfg.GamePlay.PVP = configs.ConfigString(configs.PVPDisabled)
	configs.SetConfigForTest(t, cfg)

	caster := seedDrainAreaPlayer(casterId, "chainer", "Chainer")
	target := seedDrainAreaPlayer(targetId, "bystander", "Bystander")
	defer users.SeedUsersForTest(map[int]*users.UserRecord{casterId: caster, targetId: target})()

	for _, shape := range []struct {
		spell     string
		targeting combatvocab.Targeting
	}{
		// The control: the sibling branch the fix copies.
		{"pvp-single", combatvocab.TargetSingle},
		{"pvp-multi", combatvocab.TargetMulti},
	} {
		_, cleanupSpell := seedTestSpell(shape.spell, combatvocab.Spell(combatvocab.DamageMental, shape.targeting), 4)

		actor, _, room := newPlayerActor()
		actor.userId = casterId
		room.AddPlayer(targetId)

		result := InitiateCast(actor, shape.spell, "Bystander")

		room.RemovePlayer(targetId)
		cleanupSpell()

		require.True(t, result.NoTarget, "%s: with PvP off a named player is no target", shape.spell)
		require.False(t, result.Initiated, "%s: the cast must not begin", shape.spell)
		require.Empty(t, result.TargetUserIds, "%s", shape.spell)
		require.Contains(t, strings.Join(actor.sent, "\n"), "PVP is disabled.",
			"%s: the caster is told why, in the same words", shape.spell)
	}
}
