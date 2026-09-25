package actions

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
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

// With no target named, a harmful spell falls back on the caster's current
// foe, or the party leader's. When that foe is a player, the fallback must ask
// Room.CanPvp exactly as a named target does: with PvP off the player is
// skipped, so a caster fighting a player (a duel begun before PvP was turned
// off, or a leader's foe) cannot spell them by leaving the name out.
func TestInitiateCast_HarmFallback_PlayerFoeHonoursPvp(t *testing.T) {
	const casterId, targetId = 1, 7313

	cfg := configs.GetConfig()
	cfg.GamePlay.PVP = configs.ConfigString(configs.PVPDisabled)
	configs.SetConfigForTest(t, cfg)

	caster := seedDrainAreaPlayer(casterId, "fallbacker", "Fallbacker")
	target := seedDrainAreaPlayer(targetId, "foe", "Foe")
	defer users.SeedUsersForTest(map[int]*users.UserRecord{casterId: caster, targetId: target})()

	for _, shape := range []struct {
		spell     string
		targeting combatvocab.Targeting
	}{
		{"pvp-fallback-single", combatvocab.TargetSingle},
		{"pvp-fallback-multi", combatvocab.TargetMulti},
	} {
		_, cleanupSpell := seedTestSpell(shape.spell, combatvocab.Spell(combatvocab.DamageMental, shape.targeting), 4)

		actor, char, room := newPlayerActor()
		actor.userId = casterId
		room.AddPlayer(targetId)
		char.SetAggro(targetId, 0, characters.DefaultAttack)
		require.Equal(t, targetId, char.CurrentCombatTarget().UserId,
			"%s: the fixture must put the player in the fallback slot", shape.spell)

		result := InitiateCast(actor, shape.spell, "")

		room.RemovePlayer(targetId)
		cleanupSpell()

		require.False(t, result.Initiated, "%s: the cast must not begin", shape.spell)
		require.Empty(t, result.TargetUserIds, "%s: the player foe is skipped", shape.spell)
	}
}
