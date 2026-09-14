package buffs

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

func stackingSpec() *BuffSpec {
	return &BuffSpec{BuffId: 930, Name: "Gash", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 4,
		Flags: []Flag{Bleeding, Stacking}, TickPool: "health", TickFromMagnitude: true}
}

func heldOne(t *testing.T, bs *Buffs, id int) *Buff {
	t.Helper()
	held := bs.GetBuffs(id)
	if len(held) != 1 {
		t.Fatalf("want exactly one held record %d, got %d", id, len(held))
	}
	return held[0]
}

func TestStackingAddAppendsAStackInsteadOfOverwriting(t *testing.T) {
	withSpecs(t, stackingSpec())
	bs := New()
	if !bs.AddBuffMagnitude(930, 3, -2) || !bs.AddBuffMagnitude(930, 5, -3) {
		t.Fatal("both adds must land")
	}
	b := heldOne(t, &bs, 930)
	if want := []Stack{{RoundsLeft: 3, Amount: -2}, {RoundsLeft: 5, Amount: -3}}; !reflect.DeepEqual(b.Stacks, want) {
		t.Fatalf("Stacks = %+v, want %+v", b.Stacks, want)
	}
	if b.TriggersLeft != 5 {
		t.Fatalf("TriggersLeft = %d, want 5 (the longest stack)", b.TriggersLeft)
	}
	if b.TickAmount != -5 || b.Magnitude != -5 {
		t.Fatalf("TickAmount %d / Magnitude %v, want -5 / -5 (the sum of the stacks)", b.TickAmount, b.Magnitude)
	}
}

func TestStackingTriggerSumsDecrementsAndDropsStacks(t *testing.T) {
	withSpecs(t, stackingSpec())
	bs := New()
	bs.AddBuffMagnitude(930, 2, -2)
	bs.AddBuffMagnitude(930, 4, -3)

	type round struct {
		tick         int
		stacks       []Stack
		triggersLeft int
	}
	want := []round{
		{-5, []Stack{{1, -2}, {3, -3}}, 3},
		{-5, []Stack{{2, -3}}, 2},
		{-3, []Stack{{1, -3}}, 1},
		{-3, []Stack{}, 0},
	}
	for i, w := range want {
		fired := bs.Trigger()
		if len(fired) != 1 {
			t.Fatalf("round %d: want the record to fire once, got %d", i+1, len(fired))
		}
		b := fired[0]
		if b.TickAmount != w.tick {
			t.Fatalf("round %d: TickAmount = %d, want %d (every live stack summed)", i+1, b.TickAmount, w.tick)
		}
		if len(b.Stacks) != len(w.stacks) || (len(w.stacks) > 0 && !reflect.DeepEqual(b.Stacks, w.stacks)) {
			t.Fatalf("round %d: Stacks = %+v, want %+v", i+1, b.Stacks, w.stacks)
		}
		if b.TriggersLeft != w.triggersLeft {
			t.Fatalf("round %d: TriggersLeft = %d, want %d", i+1, b.TriggersLeft, w.triggersLeft)
		}
	}
	if !bs.List[0].Expired() {
		t.Fatal("the record must be expired once its last stack ends")
	}
	if fired := bs.Trigger(); len(fired) != 0 {
		t.Fatalf("an expired stacking record must not fire again, got %d", len(fired))
	}
}

func TestStackingZeroTriggersUsesTheSpecCount(t *testing.T) {
	withSpecs(t, stackingSpec())
	bs := New()
	bs.AddBuffMagnitude(930, 0, -1)
	if got := heldOne(t, &bs, 930).Stacks[0].RoundsLeft; got != 4 {
		t.Fatalf("RoundsLeft = %d, want the spec's triggercount 4", got)
	}
}

func TestStackingAmountFloorsToOneInSign(t *testing.T) {
	withSpecs(t, stackingSpec())
	bs := New()
	bs.AddBuffMagnitude(930, 2, -0.5)
	if got := heldOne(t, &bs, 930).Stacks[0].Amount; got != -1 {
		t.Fatalf("Amount = %d, want -1: a non-zero magnitude never snapshots to zero", got)
	}
}

// Nothing produces a stacking record with no stacks, but if one exists it must
// not reach the tick path: a zero TickAmount there falls back to tick_percent.
func TestStackingRecordWithNoStacksExpiresWithoutFiring(t *testing.T) {
	withSpecs(t, stackingSpec())
	bs := New()
	bs.List = append(bs.List, &Buff{BuffId: 930, TriggersLeft: 3, TickAmount: -5})
	bs.Validate(true)
	if fired := bs.Trigger(); len(fired) != 0 {
		t.Fatalf("a stacking record with no stacks must not fire, got %d", len(fired))
	}
	if !bs.List[0].Expired() {
		t.Fatal("and it must expire")
	}
}

func TestRemoveBuffClearsStacks(t *testing.T) {
	withSpecs(t, stackingSpec())
	bs := New()
	bs.AddBuffMagnitude(930, 3, -2)
	bs.RemoveBuff(930)
	if len(bs.List[0].Stacks) != 0 {
		t.Fatalf("RemoveBuff must clear the stacks, got %+v", bs.List[0].Stacks)
	}
}

// A cancel path can expire a record without clearing its stacks (HasFlag with
// expire, CancelBuffsWithFlag). A new stack landing before the prune must not
// resurrect the old ones.
func TestStackingAddOnAnExpiredRecordStartsFresh(t *testing.T) {
	withSpecs(t, stackingSpec())
	bs := New()
	bs.AddBuffMagnitude(930, 3, -2)
	bs.List[0].TriggersLeft = TriggersLeftExpired
	bs.AddBuffMagnitude(930, 5, -3)
	if want := []Stack{{RoundsLeft: 5, Amount: -3}}; !reflect.DeepEqual(bs.List[0].Stacks, want) {
		t.Fatalf("Stacks = %+v, want %+v", bs.List[0].Stacks, want)
	}
}

func TestNonStackingRecordStillOverwrites(t *testing.T) {
	withSpecs(t, &BuffSpec{BuffId: 931, Name: "Sting", TriggerRate: "1 round", RoundInterval: 1, TriggerCount: 4,
		TickPool: "health", TickFromMagnitude: true})
	bs := New()
	bs.AddBuffMagnitude(931, 3, -2)
	bs.AddBuffMagnitude(931, 5, -3)
	b := heldOne(t, &bs, 931)
	if b.TriggersLeft != 5 || b.TickAmount != -3 || len(b.Stacks) != 0 {
		t.Fatalf("a non-stacking record overwrites: TriggersLeft %d TickAmount %d Stacks %+v, want 5 -3 []", b.TriggersLeft, b.TickAmount, b.Stacks)
	}
}

func TestValidateRefusesStackingWithoutTickFromMagnitude(t *testing.T) {
	s := &BuffSpec{BuffId: 932, Name: "Bad", TriggerRate: "1 round", TriggerCount: 1, Flags: []Flag{Stacking}, TickPool: "health", TickPercent: -1}
	if err := s.Validate(); err == nil {
		t.Fatal("a stacking record without tick_from_magnitude must be refused")
	}
}

func TestValidateRefusesStackingSlowerThanOneRound(t *testing.T) {
	s := &BuffSpec{BuffId: 933, Name: "Bad", TriggerRate: "3 rounds", TriggerCount: 1, Flags: []Flag{Stacking}, TickPool: "health", TickFromMagnitude: true}
	if err := s.Validate(); err == nil {
		t.Fatal("a stacking record must tick every round: a stack counts rounds")
	}
}

func TestValidateAcceptsAWellFormedStackingRecord(t *testing.T) {
	if err := stackingSpec().Validate(); err != nil {
		t.Fatalf("a one-round tick_from_magnitude stacking record is legal: %v", err)
	}
}

func TestStacksRoundTripThroughYaml(t *testing.T) {
	in := []*Buff{{BuffId: 930, TriggersLeft: 5, TickAmount: -5, Magnitude: -5,
		Stacks: []Stack{{RoundsLeft: 3, Amount: -2}, {RoundsLeft: 5, Amount: -3}}}}
	out, err := yaml.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var back []*Buff
	if err := yaml.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	if len(back) != 1 || !reflect.DeepEqual(back[0].Stacks, in[0].Stacks) {
		t.Fatalf("stacks must survive a save: got %+v from\n%s", back, out)
	}
}

// The equilibrium the owner asked for: one stack every 4 rounds (the shipped
// SpecialMoveCooldown), ticked every round. After warm-up the live count is
// exactly Rounds/4 when that divides evenly, and moves between floor and ceil
// when it does not.
func TestStackEquilibriumAtTheShippedCooldown(t *testing.T) {
	cases := []struct{ rounds, lo, hi int }{{8, 2, 2}, {10, 2, 3}, {12, 3, 3}}
	for _, c := range cases {
		withSpecs(t, stackingSpec())
		bs := New()
		for r := 0; r < 40; r++ {
			bs.Trigger()
			if r%4 == 0 {
				bs.AddBuffMagnitude(930, c.rounds, -2)
			}
			if r < 12 {
				continue
			}
			if n := len(bs.List[0].Stacks); n < c.lo || n > c.hi {
				t.Fatalf("stack length %d, round %d: %d live stacks, want %d to %d", c.rounds, r, n, c.lo, c.hi)
			}
		}
	}
}
