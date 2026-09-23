package combat

import (
	"regexp"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// File: wait_remote_room_hiding_test.go
//
// M4d Task 1: GetWaitMessages' defender-room line (toDefenderRoomMsg, built
// from roles.ActeeObserver) is the "second room" of a ranged attack -- the
// defender's room, when attacker and defender are not together. It is sent
// with attackResult.SendToTargetRoom, which is only ACCUMULATION: the real
// delivery happens downstream, in internal/hooks -> internal/rooms, and that
// path sight-gates correctly but never calls messaging.HideNames. A
// shapes-only (infrared, in the dark) observer standing in the defender's
// room gets only messaging.Anonymize, whose own docstring records that bare,
// untagged names embedded in prose are not touched.

// waitRemoteObserverBareNameLine is authored the way a "remote_observer"
// (SeparateMessages.ToDefenderRoom, yaml `remote_observer`) line COULD be
// authored: the attacker's name spelled directly into the prose, with no
// ansi identity tag around it. Today's shipped remote_observer content
// happens to wrap every name token in a tag, which is why this leak has not
// bitten yet -- that is author discipline, not code, and this line is the
// counter-example.
const waitRemoteObserverBareNameLine = `Skarn nocks another arrow, watching you from across the way.`

const remoteRoomHidingInfraredConditionId = 179001

// seedRemoteObserverWaitFixture installs a Generic/Wait/Separate.ToDefenderRoom
// authored line so GetWaitMessages has something to render for the ranged
// (cross-room) case. GetPreAttackMessage falls back to items.Generic for any
// subtype it does not recognize, which is what a weaponless test character
// resolves to -- the same fallback internal/hooks' own wait-round fixture
// relies on.
func seedRemoteObserverWaitFixture() func() {
	toDefenderRoom := items.SkillTieredMessages{Beginner: items.MessageOptions{items.ItemMessage(waitRemoteObserverBareNameLine)}}
	options := items.AttackOptions{
		Separate: items.SeparateMessages{
			ToDefenderRoom: toDefenderRoom,
		},
	}
	return items.SeedAttackMessagesForTest(map[items.ItemSubType]*items.WeaponAttackMessageGroup{
		items.Generic: {
			OptionId: items.Generic,
			Options: items.AttackTypes{
				items.Wait: options,
			},
		},
	})
}

var remoteRoomHidingTag = regexp.MustCompile(`<[^>]*>`)

func remoteRoomHidingPlain(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, strings.TrimSpace(remoteRoomHidingTag.ReplaceAllString(l, "")))
	}
	return out
}

// TestGetWaitMessages_RemoteRoomLineHidesBareNameFromShapesOnlyObserver
// proves and pins the fix for the ranged wait-round's second room: a
// shapes-only observer standing in the DEFENDER's room must never read the
// attacker's real (bare) name, whether the line reaches them through
// today's raw AttackResult.MessagesToTargetRoom drain (pre-fix; this test
// runs that exact drain step itself, Room.SendTextVisual, the same call
// internal/hooks.sendVisualRoomText wraps) or through the fixed direct
// delivery seated on RemoteObserver/RemoteRoom (post-fix; GetWaitMessages
// sends it itself and leaves MessagesToTargetRoom empty).
//
// Pre-fix this fails: SendTextVisual has no names argument, so a
// SightShapes reader gets only messaging.Anonymize, which does not touch a
// bare name.
func TestGetWaitMessages_RemoteRoomLineHidesBareNameFromShapesOnlyObserver(t *testing.T) {
	restoreMsgs := seedRemoteObserverWaitFixture()
	defer restoreMsgs()

	t.Cleanup(rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"city": {BiomeId: "city"},
		"cave": {BiomeId: "cave", SkyLight: rooms.SkyLightPtr(0.0)},
	}))
	// GRADED LIGHTING PLAN 2: a bare InfraredVision flag reads reach 0 by
	// design (internal/characters/vision.go), so this fixture declares an
	// explicit infra_reach, matching shipped condition 85, or the observer
	// would no longer be shapes-only in the dark at all.
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		remoteRoomHidingInfraredConditionId: {
			ConditionId: remoteRoomHidingInfraredConditionId,
			Name:        "Test Heat Eyes",
			Flags:       []conditions.Flag{conditions.InfraredVision},
			Effects:     map[conditions.EffectKind]conditions.EffectValue{conditions.EffectInfraReach: {Literal: 30}},
		},
	}))

	attackerRoom := &rooms.Room{RoomId: 179011, Biome: "city"}
	defenderRoom := &rooms.Room{RoomId: 179012, Biome: "cave"}
	t.Cleanup(rooms.SeedRoomsForTest(map[int]*rooms.Room{
		179011: attackerRoom,
		179012: defenderRoom,
	}, map[string]*rooms.ZoneConfig{}))

	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{
		179021: users.NewTestUser(179021, "shadewatch", "Shadewatch", 979021),
	}))
	observer := users.GetByUserId(179021)
	defenderRoom.AddPlayer(observer.UserId)
	events.DrainQueuedMessagesForTest(observer.UserId) // clear login noise

	if !observer.Character.Conditions.AddCondition(remoteRoomHidingInfraredConditionId, true) {
		t.Fatal("precondition: the observer should now carry infrared")
	}

	atk := characters.New()
	atk.Name = "Skarn"
	atk.RoomId = attackerRoom.RoomId

	def := characters.New()
	def.Name = "Rowan"
	def.RoomId = defenderRoom.RoomId

	result := GetWaitMessages(items.Wait, atk, def, Mob, Mob)

	// Today (pre-fix): the line is only ACCUMULATED on the AttackResult;
	// nothing has been delivered yet. Run the exact downstream call
	// production uses (internal/hooks.sendVisualRoomText wraps this one
	// function) so the test exercises the real leak, not a re-implementation
	// of it.
	for _, msg := range result.MessagesToTargetRoom {
		defenderRoom.SendTextVisual(msg.Category, msg.Text)
	}

	got := remoteRoomHidingPlain(events.DrainQueuedMessagesForTest(observer.UserId))
	if len(got) == 0 {
		t.Fatalf("shapes-only observer in the defender's room received nothing at all; expected a hidden version of the remote_observer line")
	}
	for _, line := range got {
		if strings.Contains(line, "Skarn") {
			t.Fatalf("shapes-only observer in the defender's room read the attacker's bare name: %q (full: %v)", line, got)
		}
	}
}
