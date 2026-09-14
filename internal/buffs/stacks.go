package buffs

import "slices"

// Stack is one application of a stacking record: its own remaining rounds and
// its own signed per-round amount (negative harms). See the Stacking flag.
type Stack struct {
	RoundsLeft int `yaml:"roundsleft"`
	Amount     int `yaml:"amount"`
}

// IsStacking reports whether the spec carries the Stacking flag.
func (b *BuffSpec) IsStacking() bool {
	return slices.Contains(b.Flags, Stacking)
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
// first stack. rounds 0 means the spec's triggercount. An expired record that
// has not been pruned yet (a cancel path expires without clearing) starts
// fresh rather than resurrecting its old stacks.
func (bs *Buffs) addStack(spec *BuffSpec, rounds int, magnitude float64) bool {
	if idx, ok := bs.buffIds[spec.BuffId]; ok && bs.List[idx].Expired() {
		bs.List[idx].Stacks = nil
	}
	if !bs.AddBuffScaled(spec.BuffId, 1.0) {
		return false
	}
	idx, ok := bs.buffIds[spec.BuffId]
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
// its longest stack), and TickAmount and Magnitude are the sum.
func (b *Buff) syncStacks() {
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
func (b *Buff) tickStacks() bool {
	if len(b.Stacks) == 0 {
		b.TriggersLeft = TriggersLeftExpired
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
