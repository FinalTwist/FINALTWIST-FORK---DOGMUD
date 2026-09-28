package actions

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Lighting plan 5c playtest finding 1, the scan sibling. scan lists who is in
// each adjacent room, and read no light at all: a blind scanner named the
// creatures in every room around them. It now follows the scanner's sight:
// through the exit (messaging.SeesThroughExit from the scanner's room) and
// into the next room (messaging.ParticipantSight there).
const (
	scanSightHereId  = 9490
	scanSightThereId = 9491
	scanSightScoutId = 9492
)

func scanSightSent(t *testing.T, hereLamp, thereLamp int) string {
	t.Helper()
	t.Cleanup(rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"cave": {BiomeId: "cave", SkyLight: rooms.SkyLightPtr(0.0)},
	}))
	here := &rooms.Room{RoomId: scanSightHereId, Zone: "ScanSight", Biome: "cave", Lamp: rooms.LampPtr(hereLamp),
		Exits: map[string]exit.RoomExit{"north": {RoomId: scanSightThereId}}}
	there := &rooms.Room{RoomId: scanSightThereId, Zone: "ScanSight", Biome: "cave", Lamp: rooms.LampPtr(thereLamp),
		Exits: map[string]exit.RoomExit{"south": {RoomId: scanSightHereId}}}
	t.Cleanup(rooms.SeedRoomsForTest(
		map[int]*rooms.Room{scanSightHereId: here, scanSightThereId: there},
		map[string]*rooms.ZoneConfig{"ScanSight": {Name: "ScanSight", RoomId: scanSightHereId,
			RoomIds: map[int]struct{}{scanSightHereId: {}, scanSightThereId: {}}}},
	))
	scout := newScanTestMob(scanSightScoutId, "Midroad Scout", scanSightThereId)
	mobs.SetInstanceForTest(scanSightScoutId, scout)
	t.Cleanup(func() { mobs.SetInstanceForTest(scanSightScoutId, nil) })
	there.AddMob(scanSightScoutId)

	actor := newScanFakeActor("Scanner", here, true, 9493)
	Scan(actor, ScanOptions{})
	return strings.Join(actor.sent, "\n")
}

func TestScan_FollowsTheScannersSight(t *testing.T) {
	cases := []struct {
		name          string
		here, there   int
		named, figure bool
	}{
		{"both rooms bright: the name", 90, 90, true, false},
		{"next room dim: a figure", 90, 35, false, true},
		{"next room dark: nobody", 90, -50, false, false},
		{"own room too dark to see out: nobody", 10, 90, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sent := scanSightSent(t, c.here, c.there)
			if got := strings.Contains(sent, "Midroad Scout"); got != c.named {
				t.Errorf("named = %v, want %v: %q", got, c.named, sent)
			}
			if got := strings.Contains(sent, "a figure"); got != c.figure {
				t.Errorf("figure = %v, want %v: %q", got, c.figure, sent)
			}
		})
	}
}
