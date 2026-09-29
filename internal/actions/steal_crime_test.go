package actions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// seedTheftFaction registers one faction, thornwall_citizens, and points
// every book a recorded theft writes (crimes, reputation, opinions, and the
// knowledge and bounty files under DataFiles) at temp dirs, so a test can
// run theftCrime for real and read back what it recorded.
func seedTheftFaction(t *testing.T) {
	t.Helper()
	// Registered before t.Setenv, so it runs after the environment is put
	// back: the registry reloads from wherever it loaded before.
	t.Cleanup(func() { _ = factions.LoadAllDefinitions() })
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", dir)
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", t.TempDir())
	body := "faction_id: thornwall_citizens\ndisplay_name: \"Thornwall Citizenry\"\ndescription: \"x\"\ndefault_rep: 0\nallies: []\nenemies: []\n"
	if err := os.WriteFile(filepath.Join(dir, "thornwall_citizens.yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	if err := factions.LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
	factions.ClearCache()
	crimes.ClearCache()
	opinions.ClearCache()
	t.Cleanup(func() {
		factions.ClearCache()
		crimes.ClearCache()
		opinions.ClearCache()
	})
	cfg := configs.GetConfig()
	cfg.FilePaths.DataFiles = configs.ConfigString(t.TempDir())
	configs.SetConfigForTest(t, cfg)
	t.Cleanup(rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{"default": {BiomeId: "default"}}))
}

// A pickpocket caught AWAY from the theft (the mark felt a hand after the
// thief had gone; slice H review finding b) has one witness, the mark: a
// same-faction bystander in the mark's room saw nothing, so it neither
// identifies the thief nor counts as an external witness. In the act,
// that bystander does both, as ever.
func TestTheftWitnessesAwayAreTheMarkAlone(t *testing.T) {
	seedTheftFaction(t)

	room := &rooms.Room{RoomId: 9702, Lamp: rooms.LampPtr(90)}
	mark := newStealTestMob(9921, 0, 100)
	mark.Character.RoomId = room.RoomId
	mark.Groups = []string{"thornwall_citizens"}
	bystander := newStealTestMob(9922, 0, 100)
	bystander.Character.RoomId = room.RoomId
	bystander.Groups = []string{"thornwall_citizens"}
	for _, m := range []*mobs.Mob{mark, bystander} {
		mobs.SetInstanceForTest(m.InstanceId, m)
		room.AddMob(m.InstanceId)
	}
	t.Cleanup(func() {
		mobs.SetInstanceForTest(mark.InstanceId, nil)
		mobs.SetInstanceForTest(bystander.InstanceId, nil)
	})
	factionIds := factions.FactionsForMob(mark)
	if len(factionIds) != 1 {
		t.Fatalf("fixture: the mark is of one faction, got %v", factionIds)
	}

	inAct, externalInAct := theftWitnesses(factionIds, mark, room, false)
	if len(inAct.Identifying) != 2 || !externalInAct {
		t.Fatalf("in the act, the bystander identifies the thief too: %+v external %v", inAct, externalInAct)
	}
	away, externalAway := theftWitnesses(factionIds, mark, room, true)
	if len(away.Identifying) != 1 || away.Identifying[0] != mark.InstanceId || len(away.ShapesOnly) != 0 || externalAway {
		t.Fatalf("away, the mark alone: %+v external %v", away, externalAway)
	}
}
