package questengine

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/narration"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/textutil"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// File: narrate_room_hiding_test.go
//
// M4d PR 3 Task 2: GameBridge.Narrate's observer line is a quest trigger's
// room line -- "the room watches the triggering player act" (see the
// docstring on Narrate). quests.RoomTextProblems / Quest.validateRoomText
// require every observer line to name the actor with {actor}, which always
// substitutes to the ansi-TAGGED name (Character.GetCharacterName(true)), so
// every shipped quest observer line is incidentally safe against
// messaging.Anonymize's tag-based strip today.
//
// This test authors a bare (untagged) name directly into a narration.Variants,
// the way any future caller of Narrate could (nothing enforces the {actor}
// convention or its tag-wrapping on the Go API itself, only on quest YAML
// content), to prove the mechanism -- exactly as
// internal/combat/wait_remote_room_hiding_test.go did for the ranged
// wait-round's second room.

const narrateHidingBareObserverLine = "Skarn unlocks the strongbox and pulls out a leather journal."

const narrateHidingInfraredConditionId = 179101

var narrateHidingTag = regexp.MustCompile(`<[^>]*>`)

func narrateHidingPlain(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, strings.TrimSpace(narrateHidingTag.ReplaceAllString(l, "")))
	}
	return out
}

// TestGameBridge_Narrate_RoomLineHidesBareNameFromShapesOnlyObserver proves
// and pins the fix: a shapes-only observer in the trigger room must never
// read the triggering player's bare name.
//
// Pre-fix this fails: Narrate's observer line goes out on
// room.SendTextVisual with no names argument, so a SightShapes reader gets
// only messaging.Anonymize (tag-based) over the raw text -- a bare name
// passes straight through untouched.
func TestGameBridge_Narrate_RoomLineHidesBareNameFromShapesOnlyObserver(t *testing.T) {
	t.Cleanup(rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"cave": {BiomeId: "cave", DarkArea: true},
	}))
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		narrateHidingInfraredConditionId: {
			ConditionId: narrateHidingInfraredConditionId,
			Name:        "Test Heat Eyes",
			Flags:       []conditions.Flag{conditions.InfraredVision},
		},
	}))

	room := &rooms.Room{RoomId: 179111, Biome: "cave"}
	t.Cleanup(rooms.SeedRoomsForTest(map[int]*rooms.Room{
		179111: room,
	}, map[string]*rooms.ZoneConfig{}))

	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{
		179121: users.NewTestUser(179121, "skarn", "Skarn", 979121),
		179122: users.NewTestUser(179122, "shadewatch", "Shadewatch", 979122),
	}))

	triggeringPlayer := users.GetByUserId(179121)
	observer := users.GetByUserId(179122)
	room.AddPlayer(observer.UserId)
	events.DrainQueuedMessagesForTest(observer.UserId) // clear login noise

	if !observer.Character.Conditions.AddCondition(narrateHidingInfraredConditionId, true) {
		t.Fatal("precondition: the observer should now carry infrared")
	}

	bridge := NewGameBridge(triggeringPlayer, room.RoomId)
	bridge.Narrate(narration.Variants{
		Observer: textutil.Pool(narrateHidingBareObserverLine),
	})

	got := narrateHidingPlain(events.DrainQueuedMessagesForTest(observer.UserId))
	if len(got) == 0 {
		t.Fatalf("shapes-only observer in the trigger room received nothing at all; expected a hidden version of the observer line")
	}
	for _, line := range got {
		if strings.Contains(line, "Skarn") {
			t.Fatalf("shapes-only observer in the trigger room read the triggering player's bare name: %q (full: %v)", line, got)
		}
	}
}
