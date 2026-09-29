package characters

import "testing"

// The flee command publishes a PENDING admission before its state transition
// and a READY one once it has settled the cost. The round resolver may only
// take a ready one; terminal cancellation may retract either. This is the
// contract usercommands kept in the player's temp data until slice 4a moved
// it here so a mob could flee through the same handoff.
func TestFleeAdmission_PendingWaitsForReadyOrCancel(t *testing.T) {
	c := New()
	c.PublishFleeAdmission(FleeAdmission{})
	if _, ok := c.TakeFleeAdmission(); ok {
		t.Fatal("round resolver took a pending admission")
	}
	if !c.CancelFleeAdmission() {
		t.Fatal("cancel could not retract a pending admission")
	}
	if c.CancelFleeAdmission() {
		t.Fatal("an admission was cancelled twice")
	}
}

func TestFleeAdmission_ReadyIsTakenOnce(t *testing.T) {
	c := New()
	c.PublishFleeAdmission(FleeAdmission{IncludeSkill: true, Ready: true, PreferredExit: "east"})
	got, ok := c.TakeFleeAdmission()
	if !ok || !got.IncludeSkill || got.PreferredExit != "east" {
		t.Fatalf("take = %+v, %v; want the published ready admission", got, ok)
	}
	if _, ok := c.TakeFleeAdmission(); ok {
		t.Fatal("a ready admission was taken twice")
	}
}

func TestFleeAdmission_NilCharacterIsSafe(t *testing.T) {
	var c *Character
	if _, ok := c.TakeFleeAdmission(); ok {
		t.Fatal("nil character yielded an admission")
	}
	if c.CancelFleeAdmission() {
		t.Fatal("nil character cancelled an admission")
	}
}
