package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Condition ids clear of other rooms package fixtures (sightTestInfraredConditionId is
// 7401, seenByVeilConditionId is 7621).
const (
	refreshTestHeldConditionId    = 7301 // "Test Zone Press": already held, must refresh
	refreshTestGrantedConditionId = 7302 // "Test Zone Grant": not yet held, event path
	refreshTestMobConditionId     = 7303 // "Test Zone Press (mob)": mob-side refresh
)

// refreshTestSetTriggersLeft mutates a held condition's remaining triggers
// directly, mirroring internal/hooks/condition_room_text_test.go's unexported
// expire() helper (that helper is not visible outside package hooks, so it
// is mirrored here rather than imported).
func refreshTestSetTriggersLeft(t *testing.T, list []*conditions.Condition, conditionId, left int) {
	t.Helper()
	for _, b := range list {
		if b.ConditionId == conditionId {
			b.TriggersLeft = left
			return
		}
	}
	t.Fatalf("condition %d not found on held list", conditionId)
}

// A room mutator's playerbuffids run every round. A player who already holds
// the condition must have it REFRESHED (TriggersLeft reset), not skipped until it
// lapses and gets re-added a round later: the skip is what turned slice C's
// authored notices into a start/end loop every few rounds. A player who does
// not yet hold the condition still goes through the normal grant path, which is
// the async events.Condition queue (UserRecord.AddCondition only enqueues; it does not
// apply the condition synchronously), so it must NOT already show up on
// Character.HasCondition immediately after the call, and it must queue exactly one
// Condition event, while the refreshed holder must queue none.
func TestApplyConditionIdToPlayers_HeldConditionIsRefreshedNotRelapsed(t *testing.T) {
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		refreshTestHeldConditionId:    {ConditionId: refreshTestHeldConditionId, Name: "Test Zone Press", TriggerCount: 3, RoundInterval: 1},
		refreshTestGrantedConditionId: {ConditionId: refreshTestGrantedConditionId, Name: "Test Zone Grant", TriggerCount: 3, RoundInterval: 1},
	}))
	t.Cleanup(users.SeedUsersForTest(map[int]*users.UserRecord{
		7311: users.NewTestUser(7311, "presskeeper", "Presskeeper", 97311),
		7312: users.NewTestUser(7312, "newcomer", "Newcomer", 97312),
	}))

	// Separate rooms, each with a single mutator condition id: ApplyConditionIdToPlayers
	// applies every id in its list to every player in the room, so sharing a
	// room (or a conditionIds slice) between the held and the granted case would
	// have each player also receive a grant for the other case's id, muddying
	// the event counts this test asserts on.
	heldRoom := &Room{RoomId: 7310}
	heldRoom.AddPlayer(7311)
	grantRoom := &Room{RoomId: 7313}
	grantRoom.AddPlayer(7312)

	held := users.GetByUserId(7311)
	if err := held.Character.AddCondition(refreshTestHeldConditionId, false); err != nil {
		t.Fatalf("precondition: could not grant the held condition: %v", err)
	}
	// Drive it down as if two of its three triggers had already fired.
	refreshTestSetTriggersLeft(t, held.Character.Conditions.List, refreshTestHeldConditionId, 1)

	newcomer := users.GetByUserId(7312)
	if newcomer.Character.HasCondition(refreshTestGrantedConditionId) {
		t.Fatal("precondition: the newcomer should not start with the granted condition")
	}

	// Discard anything left over from AddCondition/SeedUsersForTest setup above,
	// then read only what ApplyConditionIdToPlayers itself queues.
	events.DrainQueuedConditionsForTest(7311)
	events.DrainQueuedConditionsForTest(7312)

	heldRoom.ApplyConditionIdToPlayers([]int{refreshTestHeldConditionId}, "area")
	grantRoom.ApplyConditionIdToPlayers([]int{refreshTestGrantedConditionId}, "area")

	if got := len(held.Character.Conditions.List); got != 1 {
		t.Fatalf("held player's condition list has %d entries, want 1 (refreshed in place, not removed and re-added)", got)
	}
	if got := held.Character.Conditions.List[0].TriggersLeft; got != 3 {
		t.Errorf("held player's TriggersLeft = %d, want 3 (refreshed synchronously)", got)
	}
	if got := events.DrainQueuedConditionsForTest(7311); len(got) != 0 {
		t.Errorf("held player has %d queued Condition events, want 0: a refresh queues no event and so renders no start text", len(got))
	}

	// The player without the condition goes through the async grant path
	// (UserRecord.AddCondition), which only enqueues events.Condition; it does not
	// apply the condition synchronously. Confirming it did NOT appear yet is what
	// distinguishes this call from the held player's synchronous refresh, and
	// confirming exactly one Condition event landed proves the grant still happens.
	if newcomer.Character.HasCondition(refreshTestGrantedConditionId) {
		t.Error("newcomer already carries the granted condition synchronously; expected the async event path (grant not yet applied)")
	}
	got := events.DrainQueuedConditionsForTest(7312)
	if len(got) != 1 {
		t.Fatalf("newcomer has %d queued Condition events, want 1", len(got))
	}
	if got[0].ConditionId != refreshTestGrantedConditionId || got[0].Source != "area" {
		t.Errorf("queued Condition event = %+v, want ConditionId %d Source \"area\"", got[0], refreshTestGrantedConditionId)
	}
}

// The mob-side room applier has the same lapse-and-reapply shape as the
// player one: ApplyConditionIdToMobs must refresh a mob that already holds the
// condition instead of skipping it until it lapses and gets re-added.
func TestApplyConditionIdToMobs_HeldConditionIsRefreshedNotRelapsed(t *testing.T) {
	t.Cleanup(conditions.SeedConditionsForTest(map[int]*conditions.ConditionSpec{
		refreshTestMobConditionId: {ConditionId: refreshTestMobConditionId, Name: "Test Zone Press (mob)", TriggerCount: 3, RoundInterval: 1},
	}))

	const mobInstanceId = 7321
	m := &mobs.Mob{InstanceId: mobInstanceId}
	m.Character.Name = "Test Guard"
	m.Character.Conditions = conditions.New()
	mobs.SetInstanceForTest(mobInstanceId, m)
	t.Cleanup(func() { mobs.SetInstanceForTest(mobInstanceId, nil) })

	r := &Room{RoomId: 7320}
	r.AddMob(mobInstanceId)

	if err := m.Character.AddCondition(refreshTestMobConditionId, false); err != nil {
		t.Fatalf("precondition: could not grant the held condition: %v", err)
	}
	// Drive it down as if two of its three triggers had already fired.
	refreshTestSetTriggersLeft(t, m.Character.Conditions.List, refreshTestMobConditionId, 1)

	r.ApplyConditionIdToMobs([]int{refreshTestMobConditionId}, "area")

	if got := len(m.Character.Conditions.List); got != 1 {
		t.Fatalf("mob's condition list has %d entries, want 1 (refreshed in place, not removed and re-added)", got)
	}
	if got := m.Character.Conditions.List[0].TriggersLeft; got != 3 {
		t.Errorf("mob's TriggersLeft = %d, want 3 (refreshed synchronously)", got)
	}
}
