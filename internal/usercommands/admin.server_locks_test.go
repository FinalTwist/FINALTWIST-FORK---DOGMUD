package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/stretchr/testify/require"
)

const serverLockSentinel = `sk-server-listing-sentinel-2d8e`

var shippedLockList = configs.ConfigSliceString{`FilePaths`, `Server.CurrentVersion`, `Server.NextRoomId`, `Server.Seed`, `Server.OnLoginCommands`, `Server.BannedNames`}

// serverLockConfig installs the shipped Server.Locked list and a sentinel
// secret in both key locations master has (a module map leaf and a
// ConfigSecret field), with lookups, so `server set` resolves keys the way a
// live server does. Modules is a fresh map, never one GetConfig returned.
func serverLockConfig(t *testing.T) {
	t.Helper()
	c := configs.GetConfig()
	c.Server.Locked = shippedLockList
	c.Server.Seed = `ShippedSeed`
	c.Integrations.Discord.WebhookUrl = configs.ConfigSecret(serverLockSentinel)
	c.Modules = configs.Modules{`aicompanion`: map[string]any{`APIKey`: serverLockSentinel, `Model`: `gpt-test`}}
	configs.SetConfigWithLookupsForTest(t, c)
}

func serverOutput(userId int) string {
	return strings.Join(events.DrainQueuedMessagesForTest(userId), "\n")
}

func TestServerSetRefusesLockedKeys(t *testing.T) {
	t.Cleanup(seedAllRegistries())
	serverLockConfig(t)
	user, room := getTestUserAndRoom(t)
	events.DrainQueuedMessagesForTest(user.UserId)

	for _, cmd := range []string{`set Server.Seed 1`, `set seed 1`, `set Server.Locked x`, `set Modules.aicompanion.APIKey x`,
		`set Modules.aicompanion.ModerateOutput false`, `set Modules.aicompanion.ModerationModel x`, `set webhookurl https://example.invalid`} {
		_, err := Server(cmd, user, room, 0)
		require.NoError(t, err, cmd)
		require.Contains(t, serverOutput(user.UserId), `config name is locked`, cmd)
	}
	require.Equal(t, `ShippedSeed`, string(configs.GetServerConfig().Seed))
	require.Equal(t, shippedLockList, configs.GetServerConfig().Locked)

	// Positive control: an unlocked key still changes. SendText's normalize
	// stage capitalises the line and appends a period, so the real output is
	// "Config changed: motd=hello." and the match skips the first letter.
	_, err := Server(`set motd hello`, user, room, 0)
	require.NoError(t, err)
	require.Contains(t, serverOutput(user.UserId), `onfig changed: motd=hello`)
	require.Equal(t, `hello`, string(configs.GetServerConfig().Motd))
}

func TestIsEditAllowedDelegatesToConfigsIsLocked(t *testing.T) {
	serverLockConfig(t)
	require.False(t, isEditAllowed(`filepaths`), `Server.Locked prefix`)
	require.False(t, isEditAllowed(`server.seed`), `Server.Locked entry`)
	require.False(t, isEditAllowed(`modules.aicompanion.apikey`), `hard list`)
	require.False(t, isEditAllowed(`filepaths.webdomain`), `hard list`)
	require.False(t, isEditAllowed(`modules.aicompanion.moderateoutput`), `hard list (ruling 13)`)
	require.True(t, isEditAllowed(`modules.aicompanion`), `a partial path stays browsable`)
	require.True(t, isEditAllowed(`server.motd`))
}
