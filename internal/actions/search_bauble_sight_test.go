package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// darkBaubleRoom is a feature room in an unlit cave: a searcher with no
// vision of their own pays the sight ramp there.
func darkBaubleRoom(t *testing.T, roomId int) *rooms.Room {
	t.Helper()
	t.Cleanup(rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"cave":    {BiomeId: "cave", SkyLight: rooms.SkyLightPtr(0.0)},
		"default": {BiomeId: "default"},
	}))
	r := featureRoom(roomId)
	r.Biome = "cave"
	return r
}

// Both bauble rolls, the room's and a feature's, pass the searcher's sight
// as FindOpts.SightPenalty (owner ruling 2026-09-28).
func TestSearch_Bauble_RollsPayTheSightRamp(t *testing.T) {
	pinConfigForTest(t)
	room := darkBaubleRoom(t, 9631)
	actor := newSearchFakeActor("Groper", room, true, 7631)
	stubBaubleSearch(t, false, actor)
	var penalties []float64
	searchBaubleRoll = func(o baubles.FindOpts) (baubles.ValueTier, bool) {
		penalties = append(penalties, o.SightPenalty)
		return ``, false
	}

	want := 1 - messaging.SightMult(actor.char, room)
	if want <= 0 {
		t.Fatalf("fixture: an unlit cave must cost sight, SightMult %v", 1-want)
	}

	Search(actor, SearchOptions{})
	actor.char.Cooldowns = nil
	Search(actor, SearchOptions{Feature: "hearth"})

	if len(penalties) != 2 || penalties[0] != want || penalties[1] != want {
		t.Fatalf("room then feature roll, each at penalty %v: got %v", want, penalties)
	}
}
