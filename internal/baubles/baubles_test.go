package baubles

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

// The ladder is the owner's spec, not a tuning knob: pin it exactly.
func TestTierRangesAreTheSpec(t *testing.T) {
	want := map[ValueTier]ValueRange{
		TierCheap:   {1, 6},
		TierAverage: {10, 15},
		TierRare:    {40, 200},
	}
	if len(Tiers()) != len(want) {
		t.Fatalf("tiers: %v", Tiers())
	}
	for _, tier := range Tiers() {
		if got := defaultTierRanges[tier]; got != want[tier] {
			t.Fatalf("%s default: got %+v want %+v", tier, got, want[tier])
		}
		// With no config.yaml loaded, the live ladder is the configs default,
		// which must be the same ladder.
		if got := tier.Range(); got != want[tier] {
			t.Fatalf("%s live: got %+v want %+v", tier, got, want[tier])
		}
	}
	// Cheapest first, and the ranges never touch.
	prev := 0
	for _, tier := range Tiers() {
		r := tier.Range()
		if r.Min <= prev || r.Min > r.Max {
			t.Fatalf("%s range %+v overlaps or is inverted", tier, r)
		}
		prev = r.Max
	}
}

func TestUnknownTierIsCheap(t *testing.T) {
	if ValueTier(`legendary`).Valid() {
		t.Fatal("unknown tier must not be valid")
	}
	if ValueTier(``).Range() != TierCheap.Range() || ValueTier(`x`).ClampValue(500) != 6 {
		t.Fatal("an unknown tier must never be worth more than cheap")
	}
	if tier, ok := ParseTier(`rare`); !ok || tier != TierRare {
		t.Fatal("parse rare")
	}
	if _, ok := ParseTier(`Rare`); ok {
		t.Fatal("tiers are stored lowercase; anything else is unknown")
	}
}

func TestClampValue(t *testing.T) {
	cases := []struct {
		tier ValueTier
		in   int
		out  int
	}{
		{TierCheap, 0, 1}, {TierCheap, 3, 3}, {TierCheap, 7, 6},
		{TierAverage, 9, 10}, {TierAverage, 12, 12}, {TierAverage, 99, 15},
		{TierRare, -5, 40}, {TierRare, 120, 120}, {TierRare, 50000, 200},
	}
	for _, c := range cases {
		if got := c.tier.ClampValue(c.in); got != c.out {
			t.Fatalf("%s clamp %d: got %d want %d", c.tier, c.in, got, c.out)
		}
	}
}

func TestRollValueStaysInRangeAndCoversIt(t *testing.T) {
	for _, tier := range Tiers() {
		r := tier.Range()
		seen := map[int]bool{}
		for i := 0; i <= r.Max-r.Min; i++ {
			i := i
			v := tier.RollValue(func(n int) int {
				if n != r.Max-r.Min+1 {
					t.Fatalf("%s: randn asked for %d", tier, n)
				}
				return i
			})
			if !r.Contains(v) {
				t.Fatalf("%s: rolled %d outside %+v", tier, v, r)
			}
			seen[v] = true
		}
		if len(seen) != r.Max-r.Min+1 {
			t.Fatalf("%s: roll does not reach every value", tier)
		}
		if mid := tier.RollValue(nil); mid != (r.Min+r.Max)/2 {
			t.Fatalf("%s: nil randn gives the midpoint, got %d", tier, mid)
		}
	}
}

func TestPromptLineStatesTheRange(t *testing.T) {
	line := TierRare.PromptLine()
	if !strings.Contains(line, `40 to 200`) || !strings.Contains(line, `rare`) {
		t.Fatalf("prompt line: %s", line)
	}
	if !strings.Contains(ValueTier(`bogus`).PromptLine(), `1 to 6`) {
		t.Fatal("an unknown tier is described as cheap")
	}
}

func TestClampWeight(t *testing.T) {
	cases := map[float64]float64{
		0:           DefaultWeightLbs,
		-3:          DefaultWeightLbs,
		math.NaN():  DefaultWeightLbs,
		math.Inf(1): DefaultWeightLbs,
		0.01:        MinWeightLbs,
		0.26:        0.3,
		1.24:        1.2,
		12:          12,
		400:         MaxWeightLbs,
	}
	for in, want := range cases {
		if got := ClampWeight(in); got != want {
			t.Fatalf("weight %v: got %v want %v", in, got, want)
		}
	}
}

func TestWeightGuidanceMatchesTheBounds(t *testing.T) {
	if !strings.Contains(WeightGuidance, `toy`) || !strings.Contains(WeightGuidance, `large vase`) {
		t.Fatal("guidance must anchor both ends with examples")
	}
	if !strings.Contains(WeightGuidance, `Never more than 25`) || MaxWeightLbs != 25 {
		t.Fatal("guidance and MaxWeightLbs disagree")
	}
}

func TestReplySchemaIsStrict(t *testing.T) {
	s := ReplySchema()
	if s[`additionalProperties`] != false {
		t.Fatal("strict schemas forbid additional properties")
	}
	props := s[`properties`].(map[string]any)
	req := s[`required`].([]string)
	if len(req) != len(props) {
		t.Fatalf("strict mode needs every property required: %v", req)
	}
	for _, k := range req {
		if _, ok := props[k]; !ok {
			t.Fatalf("required %q has no property", k)
		}
	}
	// Every schema property must be a field of Reply, and vice versa.
	b, _ := json.Marshal(Reply{})
	var fields map[string]any
	_ = json.Unmarshal(b, &fields)
	if len(fields) != len(props) {
		t.Fatalf("Reply fields %v vs schema %v", fields, props)
	}
	for k := range fields {
		if _, ok := props[k]; !ok {
			t.Fatalf("Reply field %q missing from schema", k)
		}
	}
	if props[`weight_lbs`].(map[string]any)[`type`] != `number` || props[`value`].(map[string]any)[`type`] != `integer` {
		t.Fatal("weight is a number, value an integer")
	}
}

func TestParseReply(t *testing.T) {
	r, err := ParseReply(`{"name":"Painted Wooden Horse","name_simple":"horse","description":"A child's toy.","material":"pine","weight_lbs":0.6,"value":4}`)
	if err != nil || r.Name != `Painted Wooden Horse` || r.WeightLbs != 0.6 || r.Value != 4 {
		t.Fatalf("parse: %+v %v", r, err)
	}
	if _, err := ParseReply(`{"name":"x","extra":1}`); err == nil {
		t.Fatal("unknown fields must be refused")
	}
	if _, err := ParseReply(`{"value":12.5}`); err == nil {
		t.Fatal("a fractional value must be refused")
	}
	if _, err := ParseReply(`not json`); err == nil {
		t.Fatal("garbage must be refused")
	}
}

func TestApplyLimits(t *testing.T) {
	in := Reply{Name: `Great Bronze Urn`, WeightLbs: 60, Value: 999}
	l := ApplyLimits(in, TierAverage)
	if l.Reply.Value != 15 || l.Reply.WeightLbs != MaxWeightLbs || l.ProposedValue != 999 || l.ProposedWeight != 60 || l.Tier != TierAverage {
		t.Fatalf("limits: %+v", l)
	}
	if l.Reply.Name != in.Name {
		t.Fatal("text passes through unchanged")
	}
	if l := ApplyLimits(Reply{Value: 150, WeightLbs: 1}, ValueTier(`?`)); l.Tier != TierCheap || l.Reply.Value != 6 {
		t.Fatalf("unknown tier falls back to cheap: %+v", l)
	}
}
