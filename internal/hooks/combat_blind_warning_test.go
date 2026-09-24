package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// File: combat_blind_warning_test.go
//
// Pins Task 4 (M4d PR 2): the once-per-round "you can't see clearly"
// notice for a player who fought this round while their sight verdict
// was not SightFull. Exercised through the real production seam
// (dispatchCritAndMessaging marks, flushBlindCombatNotices sends), the
// same pattern combat_verbosity_wiring_test.go uses for the tally flush.

// blindNoticeNeedle is a substring unique to blindCombatNoticeText that
// cannot collide with any other combat line in these fixtures.
const blindNoticeNeedle = "cannot see clearly"

func TestBlindCombatNotice(t *testing.T) {
	// ── 1. Blind combatant, several swings: exactly one notice ───────────
	t.Run("FiresOnceForBlindCombatant_NotPerSwing", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		restoreConditions := seedNarrationConditions()
		defer restoreConditions()
		roundBlindCombatants = map[int]bool{}

		darken(t, 2)
		room2 := rooms.LoadRoom(2)
		require.NotNil(t, room2)

		u1 := users.GetByUserId(1)
		require.NotNil(t, u1)
		u1.Character.RoomId = 2
		rooms.LoadRoom(1).RemovePlayer(1)
		room2.AddPlayer(1)
		// No NightVision/InfraredVision condition: default is SightNone in
		// an unlit room.

		mob := mobs.GetInstance(100)
		require.NotNil(t, mob)
		atk := actions.NewUserActorInRoom(u1, room2)
		def := actions.NewMobActorInRoom(mob, room2)

		captured, capCleanup := captureMessages(t)
		defer capCleanup()

		// Simulate a round in which this attacker/defender pair produced
		// more than one AttackResult (e.g. a multi-attack character, or
		// two separate mobs swinging at the same blind player) before the
		// round-end flush runs once.
		dispatchCritAndMessaging(atk, def, vbLandingResult())
		dispatchCritAndMessaging(atk, def, vbLandingResult())
		dispatchCritAndMessaging(atk, def, vbLandingResult())
		flushBlindCombatNotices()
		events.ProcessEvents()

		texts := textsForUser(*captured, 1)
		assert.Equal(t, 1, countContaining(texts, blindNoticeNeedle),
			"three swings in one round must still produce exactly one blind notice")
	})

	// ── 2. Sighted combatant: no notice ───────────────────────────────────
	t.Run("NoNoticeForSightedCombatant", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		restoreConditions := seedNarrationConditions()
		defer restoreConditions()
		roundBlindCombatants = map[int]bool{}

		room1 := rooms.LoadRoom(1)
		require.NotNil(t, room1)
		// Pins room 1 fully lit regardless of the ambient test round. Since
		// graded lighting plan 3a Task 8, LightLevel() reads the real
		// celestial term at whatever round util.GetRoundCount() holds,
		// which a bare unpinned round reads as shapes tier (light ~28),
		// not full sight, and this lane needs SightFull specifically.
		room1.Lamp = rooms.LampPtr(90)
		require.GreaterOrEqual(t, room1.LightLevel(), configs.GetLightingConfig().ExitsAbove,
			"room 1 must be fully lit for this lane")

		u1 := users.GetByUserId(1)
		require.NotNil(t, u1)

		mob := mobs.GetInstance(100)
		require.NotNil(t, mob)
		atk := actions.NewUserActorInRoom(u1, room1)
		def := actions.NewMobActorInRoom(mob, room1)

		captured, capCleanup := captureMessages(t)
		defer capCleanup()

		dispatchCritAndMessaging(atk, def, vbLandingResult())
		flushBlindCombatNotices()
		events.ProcessEvents()

		texts := textsForUser(*captured, 1)
		assert.Equal(t, 0, countContaining(texts, blindNoticeNeedle),
			"a combatant who can see clearly must never receive the blind notice")
	})

	// ── 3. Blind non-combatant: no notice ─────────────────────────────────
	// A player standing in the same dark room, who never appears as
	// attacker or defender this round, must not read a combat notice.
	t.Run("NoNoticeForBlindNonCombatant", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		restoreConditions := seedNarrationConditions()
		defer restoreConditions()
		roundBlindCombatants = map[int]bool{}

		darken(t, 2)
		room2 := rooms.LoadRoom(2)
		require.NotNil(t, room2)
		room1 := rooms.LoadRoom(1)
		require.NotNil(t, room1)

		// User 1 stands in the dark room but never fights this round.
		u1 := users.GetByUserId(1)
		require.NotNil(t, u1)
		u1.Character.RoomId = 2
		room1.RemovePlayer(1)
		room2.AddPlayer(1)

		// User 2 fights a mob in the SAME dark room, so this lane also
		// proves the non-combatant's silence isn't just an empty round.
		u2 := users.GetByUserId(2)
		require.NotNil(t, u2)
		u2.Character.RoomId = 2
		room1.RemovePlayer(2)
		room2.AddPlayer(2)

		mob := mobs.GetInstance(100)
		require.NotNil(t, mob)
		atk := actions.NewUserActorInRoom(u2, room2)
		def := actions.NewMobActorInRoom(mob, room2)

		captured, capCleanup := captureMessages(t)
		defer capCleanup()

		dispatchCritAndMessaging(atk, def, vbLandingResult())
		flushBlindCombatNotices()
		events.ProcessEvents()

		nonCombatantTexts := textsForUser(*captured, 1)
		assert.Equal(t, 0, countContaining(nonCombatantTexts, blindNoticeNeedle),
			"a blind bystander who did not fight this round must not receive the notice")

		combatantTexts := textsForUser(*captured, 2)
		assert.Equal(t, 1, countContaining(combatantTexts, blindNoticeNeedle),
			"sanity check: the blind combatant in the same round/room DOES get the notice")
	})
}
