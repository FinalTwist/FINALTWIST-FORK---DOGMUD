package usercommands

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/require"
)

// File: craft_instant_room_hiding_test.go
//
// M4d PR 3 Task 5: the instant-craft success narration (case
// result.ImmediateComplete in Craft(), and completeCraft, the enchanting
// instant path). Both share one delivery tail: a self-only success line to
// the crafter, and a room line built from recipe.Narrate's {actor}
// substitution and sent on Room.SendTextVisual with no names.
//
// {actor} resolves to user.Character.GetCharacterName(true), which wraps the
// name in an <ansi fg="username"> identity tag, so today's shipped content is
// incidentally safe: RenderForRecipient's anonymize stage (pipeline.go)
// strips ANY tag-wrapped identity name for a SightShapes reader, regardless
// of which delivery call sends it. See internal/messaging/pipeline.go and
// internal/messaging/anonymize.go.
//
// Because the tag comes from GetCharacterName(true) rather than caller-
// supplied content, there is no way to drive the real crafting path into
// emitting a bare name. This test proves the MECHANISM directly, the same
// way internal/hooks/selfcast_spell_room_hiding_test.go proved its own:
//
//   - Room.SendTextVisual (no names), what both instant-craft sites called
//     before this task, passes no names, so a bare name in that position
//     leaks straight through to a shapes-only observer.
//   - messaging.SendTrio, what both sites call now (via craftDeliverInstant),
//     passes the crafter's name, so HideNames catches a bare name too.
//
// Aliceia (user 1) crafts; Bobrick (user 2) is the shapes-only bystander.

const craftHidingInfraredConditionId = 9101

// seedCraftHidingCondition installs a test condition that grants
// InfraredVision, additive on top of seedAllRegistries' condition registry.
// Call it AFTER `defer cleanup := seedAllRegistries()` and defer its own
// result so it restores BEFORE the fixture does (SeedConditionsForTest
// replaces the whole registry).
func seedCraftHidingCondition() func() {
	return conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		craftHidingInfraredConditionId: {
			ConditionId: craftHidingInfraredConditionId,
			Name:        "Test Heat Eyes",
			Flags:       []conditions.Flag{conditions.InfraredVision},
		},
	})
}

// darkenCraftRoom turns a fixture room into an unlit cave, so CanSeeClearly
// fails and a bystander with InfraredVision sight lands at SightShapes.
func darkenCraftRoom(t *testing.T, roomId int) {
	t.Helper()
	room := rooms.LoadRoom(roomId)
	require.NotNil(t, room)
	room.Biome = "cave"
	require.Zero(t, room.GetVisibility(), "room %d must actually be unlit", roomId)
}

var craftHidingTagPattern = regexp.MustCompile(`<[^>]*>`)

// craftPlainLines drains a user's queued messages and returns them tag-stripped.
func craftPlainLines(userId int) []string {
	raw := events.DrainQueuedMessagesForTest(userId)
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		out = append(out, strings.TrimSpace(craftHidingTagPattern.ReplaceAllString(line, "")))
	}
	return out
}

func craftCountContaining(lines []string, needle string) int {
	n := 0
	for _, l := range lines {
		if strings.Contains(l, needle) {
			n++
		}
	}
	return n
}

const craftHidingBareRoomLine = `Aliceia ladles out a steaming stew.`

// TestCraftInstantRoomLine_SendTrioHidesABareNameSendTextVisualDoesNot proves
// the mechanism the migration depends on.
func TestCraftInstantRoomLine_SendTrioHidesABareNameSendTextVisualDoesNot(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restoreCond := seedCraftHidingCondition()
	defer restoreCond()
	darkenCraftRoom(t, 1)
	require.True(t, users.GetByUserId(2).Character.Conditions.AddCondition(craftHidingInfraredConditionId, true))
	craftPlainLines(1)
	craftPlainLines(2)

	// Pre-fix shape: Room.SendTextVisual, what both instant-craft sites called
	// before this task. It passes no names, so only the tag-based Anonymize
	// stage runs, and a bare name leaks straight through.
	rooms.LoadRoom(1).SendTextVisual(messaging.CategoryEmote, craftHidingBareRoomLine, 1)
	leaked := craftPlainLines(2)
	require.Equal(t, 1, craftCountContaining(leaked, "Aliceia"),
		"Room.SendTextVisual should leak a bare name to a shapes-only observer (the pre-fix call shape): got %v", leaked)

	// Post-fix shape: messaging.SendTrio, what craftDeliverInstant calls now.
	// It passes the crafter's name, so HideNames runs too.
	messaging.SendTrio(messaging.Trio{
		Actor:    messaging.NoLine,
		Actee:    messaging.NoLine,
		Observer: messaging.Say(messaging.CategoryEmote, craftHidingBareRoomLine),
	}, messaging.Audience{
		Actor:     users.GetByUserId(1),
		ActorId:   1,
		ActorName: "Aliceia",
		ActeeName: messaging.NoName,
		Room:      rooms.LoadRoom(1),
	})
	hidden := craftPlainLines(2)
	require.Equal(t, 0, craftCountContaining(hidden, "Aliceia"),
		"messaging.SendTrio must hide the crafter's bare name from a shapes-only observer: got %v", hidden)
	require.Equal(t, 1, craftCountContaining(hidden, "A figure ladles out a steaming stew."))
}

// craftInstantHidingRecipe returns a minimal recipe for the two real-branch
// regression tests below: no station, no skill floor, no ingredients, so the
// only thing under test is the delivery of its narration.
func craftInstantHidingRecipe(id string) *crafting.RecipeSpec {
	return &crafting.RecipeSpec{
		RecipeId:           id,
		Name:               id,
		Skill:              "test-craft-hiding-skill",
		TimeRounds:         0,
		Output:             crafting.RecipeOutput{ItemId: 10001, Quantity: 1},
		SuccessMessage:     "You finish your work.",
		SuccessRoomMessage: "{actor} finishes a piece of work.",
	}
}

// TestCompleteCraft_RealBranch_ShapesOnlyThirdPartyReadsAFigure is a
// regression guard, not a red test (today's tag-wrapped content is
// incidentally safe either way, per the file comment above): it drives the
// real completeCraft function (craft.go:656, the enchanting instant-complete
// path) with a shapes-only third party in the room, and pins that the
// migration to craftDeliverInstant kept the crafter exclusion, the category
// and the delivered wording intact.
//
// The hidden form reads lowercase "a figure": {actor} is tag-wrapped, so
// sendTextVisualJudgedBy's own Anonymize (tag-based, no sentence-position
// awareness) resolves it before messaging.HideNames ever sees a name to act
// on -- identical to what the pre-fix Room.SendTextVisual call produced,
// which is exactly the "text-identical" property an incidentally-safe
// migration is supposed to have.
func TestCompleteCraft_RealBranch_ShapesOnlyThirdPartyReadsAFigure(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restoreCond := seedCraftHidingCondition()
	defer restoreCond()
	darkenCraftRoom(t, 1)

	crafter := users.GetByUserId(1)
	watcher := users.NewTestUser(3, "cara", "Carrow", 1003)
	watcher.Character.RoomId = 1
	restoreUsers := users.SeedUsersForTest(map[int]*users.UserRecord{1: crafter, 2: users.GetByUserId(2), 3: watcher})
	defer restoreUsers()
	room := rooms.LoadRoom(1)
	room.AddPlayer(3)
	require.True(t, watcher.Character.Conditions.AddCondition(craftHidingInfraredConditionId, true))
	craftPlainLines(1)
	craftPlainLines(3)

	recipe := craftInstantHidingRecipe("test-complete-craft-hiding")
	completeCraft(crafter, room, recipe)

	crafterLines, watcherLines := craftPlainLines(1), craftPlainLines(3)
	require.Equal(t, 1, craftCountContaining(crafterLines, "You finish your work."))
	require.Equal(t, 1, craftCountContaining(watcherLines, "a figure finishes a piece of work."))
	require.Equal(t, 0, craftCountContaining(watcherLines, "Aliceia"))
}

// TestCraftImmediateComplete_RealBranch_ShapesOnlyThirdPartyReadsAFigure is
// the same regression guard for the OTHER instant-craft site: case
// result.ImmediateComplete inside Craft() itself (craft.go:134), reached
// through actions.InitiateCraft with a zero-time-rounds recipe. Same
// lowercase "a figure" form as completeCraft's test, and for the same
// reason (see that test's comment).
func TestCraftImmediateComplete_RealBranch_ShapesOnlyThirdPartyReadsAFigure(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restoreCond := seedCraftHidingCondition()
	defer restoreCond()
	darkenCraftRoom(t, 1)

	crafter := users.GetByUserId(1)
	watcher := users.NewTestUser(3, "cara", "Carrow", 1003)
	watcher.Character.RoomId = 1
	restoreUsers := users.SeedUsersForTest(map[int]*users.UserRecord{1: crafter, 2: users.GetByUserId(2), 3: watcher})
	defer restoreUsers()
	room := rooms.LoadRoom(1)
	room.AddPlayer(3)
	require.True(t, watcher.Character.Conditions.AddCondition(craftHidingInfraredConditionId, true))

	recipe := craftInstantHidingRecipe("test-immediate-complete-hiding")
	crafting.RegisterRecipeForTest(recipe)
	defer crafting.UnregisterRecipeForTest(recipe.RecipeId)
	crafter.Character.KnownRecipes = map[string]int{recipe.RecipeId: 1}

	craftPlainLines(1)
	craftPlainLines(3)

	handled, err := Craft(recipe.Name, crafter, room, 0)
	require.NoError(t, err)
	require.True(t, handled)

	crafterLines, watcherLines := craftPlainLines(1), craftPlainLines(3)
	require.Equal(t, 1, craftCountContaining(crafterLines, "You finish your work."))
	require.Equal(t, 1, craftCountContaining(watcherLines, "a figure finishes a piece of work."))
	require.Equal(t, 0, craftCountContaining(watcherLines, "Aliceia"))
}
