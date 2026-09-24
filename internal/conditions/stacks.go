package conditions

import (
	"slices"
	"strconv"
)

// Stack is one application of a stacking record: its own remaining rounds and
// its own signed per-round amount (negative harms). See the Stacking flag.
type Stack struct {
	RoundsLeft int `yaml:"roundsleft"`
	Amount     int `yaml:"amount"`
}

// IsStacking reports whether the spec carries the Stacking flag.
func (b *ConditionSpec) IsStacking() bool {
	return slices.Contains(b.Flags, Stacking)
}

// DisplayName is the name a condition list shows for a held record: the
// spec's name, with the live stack count appended when more than one stack is
// live ("Bleeding (3)").
func DisplayName(b *Condition, spec *ConditionSpec) string {
	if len(b.Stacks) > 1 {
		return spec.Name + " (" + strconv.Itoa(len(b.Stacks)) + ")"
	}
	return spec.Name
}

// tickAmountFor converts an applier's magnitude into the signed per-round
// amount a tick_from_magnitude record lands. It truncates toward zero, except
// that a non-zero magnitude which truncates to zero becomes 1 in its sign: a
// zero snapshot would tick for nothing forever, because the round tick's
// fallback recomputes from tick_percent, which a tick_from_magnitude record
// may not set.
func tickAmountFor(magnitude float64) int {
	amt := int(magnitude)
	if amt == 0 && magnitude != 0 {
		if magnitude < 0 {
			return -1
		}
		return 1
	}
	return amt
}

// addStack appends one stack to a stacking record, creating the record on the
// first stack. rounds 0 means the spec's triggercount. magnitude 0 is
// refused: a zero-amount stack would still lengthen the record and print a
// bleed line on its own tick, for no harm landed.
//
// Every internal path that expires a held record now clears its stacks
// through Condition.expire() (RemoveCondition, HasFlag's expire branch, tickStacks), so
// an expired, unpruned record should already hold none. The clear below is a
// cheap defensive second guard, not the primary defense, for a record that
// somehow reached TriggersLeft <= 0 without going through expire().
func (bs *Conditions) addStack(spec *ConditionSpec, rounds int, magnitude float64) bool {
	if magnitude == 0 {
		return false
	}
	if idx, ok := bs.conditionIds[spec.ConditionId]; ok && bs.List[idx].Expired() {
		bs.List[idx].Stacks = nil
	}
	if !bs.addConditionScaled(spec.ConditionId, 1.0) {
		return false
	}
	idx, ok := bs.conditionIds[spec.ConditionId]
	if !ok {
		return false
	}
	if rounds <= 0 {
		rounds = spec.TriggerCount
	}
	b := bs.List[idx]
	b.Stacks = append(b.Stacks, Stack{RoundsLeft: rounds, Amount: tickAmountFor(magnitude)})
	b.syncStacks()
	return true
}

// syncStacks derives the record-level fields every other reader uses from the
// live stacks: TriggersLeft is the longest stack (so Expired, GetDurations,
// the prune pass and both condition lists see a record that lives as long as
// its longest stack). TickAmount and Magnitude are the sum of the live
// stacks' amounts, which is accurate for both right after an add. tickStacks
// calls this too, then overwrites TickAmount with the amount that just
// landed this round (summed before the round's stacks are decremented and
// dropped), which is a different figure from Magnitude once any tick has
// happened. Between ticks, a caller that wants "the whole bleed" should read
// Stacks or Magnitude, never TickAmount.
func (b *Condition) syncStacks() {
	longest, sum := 0, 0
	for _, s := range b.Stacks {
		longest = max(longest, s.RoundsLeft)
		sum += s.Amount
	}
	b.TriggersLeft = longest
	b.TickAmount = sum
	b.Magnitude = float64(sum)
}

// tickStacks lands one round of a stacking record. TickAmount becomes this
// round's amount, the sum of every live stack, which is what both round-tick
// paths read after Trigger returns. Each stack then loses a round and the
// spent ones drop; TriggersLeft becomes the longest remaining stack and
// Magnitude the sum still to come. Returns false, having expired the record,
// when there are no stacks to tick.
func (b *Condition) tickStacks() bool {
	if len(b.Stacks) == 0 {
		b.expire()
		return false
	}
	landed := 0
	live := b.Stacks[:0]
	for _, s := range b.Stacks {
		landed += s.Amount
		s.RoundsLeft--
		if s.RoundsLeft > 0 {
			live = append(live, s)
		}
	}
	b.Stacks = live
	b.syncStacks()
	b.TickAmount = landed
	return true
}
