package characters

import "testing"

func apChar(max int) *Character {
	c := &Character{}
	c.ActionPointsMax.Value = max
	return c
}

// A character never settled starts full: a fresh spawn, or a mob loaded from
// an instance file, must not start frozen at 0 (every live mob did before
// movement parity 4b).
func TestSettleActionPoints_FirstSettleFills(t *testing.T) {
	c := apChar(200)
	c.SettleActionPoints(1000)
	if c.ActionPoints != 200 {
		t.Fatalf("first settle: ActionPoints = %d, want 200", c.ActionPoints)
	}
	if !c.ActionPointsSettled || c.ActionPointsSettledTurn != 1000 {
		t.Fatalf("first settle must stamp the turn: settled=%v turn=%d", c.ActionPointsSettled, c.ActionPointsSettledTurn)
	}
}

// One point per elapsed turn, the player rate (hooks.ActionPoints).
func TestSettleActionPoints_AddsElapsedTurns(t *testing.T) {
	c := apChar(200)
	c.SettleActionPoints(1000)
	c.ActionPoints = 50
	c.SettleActionPoints(1030)
	if c.ActionPoints != 80 {
		t.Fatalf("30 turns later: ActionPoints = %d, want 80", c.ActionPoints)
	}
}

func TestSettleActionPoints_CapsAtMax(t *testing.T) {
	c := apChar(200)
	c.SettleActionPoints(10)
	c.ActionPoints = 190
	c.SettleActionPoints(1_000_000)
	if c.ActionPoints != 200 {
		t.Fatalf("long gap: ActionPoints = %d, want the cap 200", c.ActionPoints)
	}
}

// Settling twice at the same turn adds nothing: a quote followed by a charge
// in the same turn must not mint points.
func TestSettleActionPoints_SameTurnIsIdempotent(t *testing.T) {
	c := apChar(200)
	c.SettleActionPoints(500)
	c.ActionPoints = 40
	c.SettleActionPoints(500)
	c.SettleActionPoints(500)
	if c.ActionPoints != 40 {
		t.Fatalf("same turn: ActionPoints = %d, want 40", c.ActionPoints)
	}
}

// The turn counter restarts at 0 on a reboot. A stamp from the future is not
// a debt: the character refills.
func TestSettleActionPoints_CounterRestartRefills(t *testing.T) {
	c := apChar(200)
	c.SettleActionPoints(9000)
	c.ActionPoints = 3
	c.SettleActionPoints(5)
	if c.ActionPoints != 200 || c.ActionPointsSettledTurn != 5 {
		t.Fatalf("restart: ActionPoints = %d turn = %d, want 200 and 5", c.ActionPoints, c.ActionPointsSettledTurn)
	}
}
