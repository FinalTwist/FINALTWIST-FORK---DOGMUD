package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/require"
)

const moveCliffBiome = "movetestcliff"

func seedMoveBiomes(t *testing.T) {
	t.Helper()
	t.Cleanup(rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		moveCliffBiome: {BiomeId: moveCliffBiome, MovementCost: 2.5},
	}))
}

// setupMover shapes a character the same way for both actor kinds.
type moverSetup struct {
	stamina, staminaMax, apMax int
	hidden                     bool
	overloaded                 bool
}

func (s moverSetup) apply(t *testing.T, c *characters.Character) {
	t.Helper()
	c.Stamina = s.stamina
	c.StaminaMax.Value = s.staminaMax
	c.ActionPointsMax.Value = s.apMax
	c.Stats.Strength.ValueAdj = 10
	if s.overloaded {
		c.Items = []items.Item{{ItemId: 99101, Spec: &items.ItemSpec{ItemId: 99101, Name: "anvil", Weight: 100000}}}
	}
	if s.hidden {
		c.Awareness = awareness.NewMachine()
		reason := state.TransitionReason{Trigger: "move_test"}
		require.NoError(t, c.Awareness.TransitionToConcealing(awareness.ConcealingData{}, reason))
		c.Awareness.ResolveConcealment(true, reason)
		require.True(t, c.IsHidden())
	}
}

func newMoveUser(t *testing.T, s moverSetup) Actor {
	t.Helper()
	u := users.NewTestUser(9601, "mover", "Mover", 0)
	u.Character = characters.New()
	s.apply(t, u.Character)
	u.Character.ActionPoints = s.apMax // players are credited by the per-turn hook, never settled
	return NewUserActor(u)
}

func newMoveMob(t *testing.T, s moverSetup) Actor {
	t.Helper()
	m := &mobs.Mob{InstanceId: 9602}
	m.Character = *characters.New()
	s.apply(t, &m.Character)
	return NewMobActor(m) // never settled: the first quote or charge fills it
}

// The parity table: the same character pays the same price whether a player
// or a mob is walking (movement parity 4b, owner ruling 2).
func TestChargeMove_PlayerAndMobPayTheSame(t *testing.T) {
	seedMoveBiomes(t)
	road := &rooms.Room{RoomId: 9610}
	cliff := &rooms.Room{RoomId: 9611, Biome: moveCliffBiome}

	cases := []struct {
		name        string
		setup       moverSetup
		dest        *rooms.Room
		wantRefusal MoveRefusal
		wantAP      int
	}{
		{"plain step", moverSetup{stamina: 100, staminaMax: 100, apMax: 200}, road, MoveOK, 10},
		{"cliff step", moverSetup{stamina: 100, staminaMax: 100, apMax: 200}, cliff, MoveOK, 10},
		{"hidden step", moverSetup{stamina: 100, staminaMax: 100, apMax: 200, hidden: true}, road, MoveOK, 10},
		{"over capacity costs 50 points", moverSetup{stamina: 100, staminaMax: 100, apMax: 200, overloaded: true}, road, MoveOK, 50},
		{"too tired", moverSetup{stamina: 100, staminaMax: 100, apMax: 5}, road, MoveRefuseTired, 10},
		{"too encumbered", moverSetup{stamina: 100, staminaMax: 100, apMax: 40, overloaded: true}, road, MoveRefuseEncumbered, 50},
		{"exhausted", moverSetup{stamina: 0, staminaMax: 100, apMax: 200}, cliff, MoveRefuseExhausted, 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			player := newMoveUser(t, tc.setup)
			mob := newMoveMob(t, tc.setup)

			pc := ChargeMove(player, tc.dest)
			mc := ChargeMove(mob, tc.dest)

			require.Equal(t, tc.wantRefusal, pc.Refusal, "player refusal")
			require.Equal(t, pc.Refusal, mc.Refusal, "mob refusal must match the player's")
			require.Equal(t, tc.wantAP, pc.ActionCost)
			require.Equal(t, pc.ActionCost, mc.ActionCost)
			require.InDelta(t, pc.StaminaCost, mc.StaminaCost, 1e-12)

			_, wantStamina := MovePrice(player.GetCharacter(), tc.dest)
			require.InDelta(t, wantStamina, pc.StaminaCost, 1e-12, "ChargeMove must price through MovePrice")

			if tc.wantRefusal == MoveOK {
				require.Equal(t, tc.setup.apMax-tc.wantAP, player.GetCharacter().ActionPoints)
				require.Equal(t, tc.setup.apMax-tc.wantAP, mob.GetCharacter().ActionPoints)
			}
			if tc.wantRefusal == MoveRefuseExhausted {
				require.Equal(t, tc.setup.apMax, player.GetCharacter().ActionPoints, "a stamina refusal refunds the points")
				require.Equal(t, tc.setup.apMax, mob.GetCharacter().ActionPoints, "a stamina refusal refunds the points")
			}
		})
	}
}

// Terrain and hidden multiply the stamina price (V9); a cliff costs five road
// steps and hidden triples it, until the 20-point cap.
func TestMovePrice_TerrainAndHidden(t *testing.T) {
	seedMoveBiomes(t)
	road := &rooms.Room{RoomId: 9610}
	cliff := &rooms.Room{RoomId: 9611, Biome: moveCliffBiome}
	plain := newMoveUser(t, moverSetup{stamina: 100, staminaMax: 100, apMax: 200})
	hidden := newMoveUser(t, moverSetup{stamina: 100, staminaMax: 100, apMax: 200, hidden: true})

	_, r := MovePrice(plain.GetCharacter(), road)
	_, c := MovePrice(plain.GetCharacter(), cliff)
	_, h := MovePrice(hidden.GetCharacter(), road)
	require.InDelta(t, 2.5, c/r, 1e-9, "a 2.5 biome costs 2.5 road steps")
	require.InDelta(t, 3.0, h/r, 1e-9, "hidden triples the price")
}

// A step the character can never pay, even rested, reports Never so a walker
// gives up instead of waiting forever.
func TestQuoteMove_NeverWhenThePriceExceedsTheWholePool(t *testing.T) {
	seedMoveBiomes(t)
	cliff := &rooms.Room{RoomId: 9611, Biome: moveCliffBiome}

	tired := newMoveMob(t, moverSetup{stamina: 0, staminaMax: 100, apMax: 200})
	q := QuoteMove(tired, cliff)
	require.Equal(t, MoveRefuseExhausted, q.Refusal)
	require.False(t, q.Never, "a rested mob could pay this step")

	// Stamina 0 so the quote refuses (at 1 the whole point of a 1.37 step is
	// affordable); a pool of 1 so no rest can ever cover it.
	frail := newMoveMob(t, moverSetup{stamina: 0, staminaMax: 1, apMax: 200})
	q = QuoteMove(frail, cliff)
	require.Equal(t, MoveRefuseExhausted, q.Refusal)
	require.True(t, q.Never, "a 1-stamina mob can never pay a cliff step")
}

// A quote spends nothing.
func TestQuoteMove_SpendsNothing(t *testing.T) {
	road := &rooms.Room{RoomId: 9610}
	mob := newMoveMob(t, moverSetup{stamina: 100, staminaMax: 100, apMax: 200})
	q := QuoteMove(mob, road)
	require.True(t, q.OK())
	require.Equal(t, 200, mob.GetCharacter().ActionPoints, "the quote settled the mob full and spent nothing")
	require.Equal(t, 100, mob.GetCharacter().Stamina)
}

// Mob points refill lazily at one per turn; a player's are never settled by
// the charge (the per-turn hook credits them).
func TestChargeMove_SettlesMobsOnly(t *testing.T) {
	road := &rooms.Room{RoomId: 9610}

	// The turn counter only moves in world.go, so it may sit at 0 in a test
	// binary; push it past 20 so "20 turns ago" is a real turn.
	for util.GetTurnCount() < 100 {
		util.IncrementTurnCount()
	}
	mob := newMoveMob(t, moverSetup{stamina: 100, staminaMax: 100, apMax: 200})
	mc := mob.GetCharacter()
	mc.SettleActionPoints(util.GetTurnCount() - 20)
	mc.ActionPoints = 0
	require.Equal(t, MoveOK, ChargeMove(mob, road).Refusal, "20 elapsed turns buy a 10-point step")
	require.Equal(t, 10, mc.ActionPoints)

	player := newMoveUser(t, moverSetup{stamina: 100, staminaMax: 100, apMax: 200})
	pc := player.GetCharacter()
	pc.ActionPoints = 5
	require.Equal(t, MoveRefuseTired, ChargeMove(player, road).Refusal, "a player is not settled by the charge")
	require.False(t, pc.ActionPointsSettled)
}
