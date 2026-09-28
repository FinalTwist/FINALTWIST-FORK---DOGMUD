package baubles

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ReplySchemaName names the strict JSON schema sent with a generation
// request.
const ReplySchemaName = `bauble`

// Reply is the model's answer to a generation request, exactly as the
// schema below asks for it. Nothing in it is trusted until ApplyLimits (and,
// from Phase 4, the text validation) has run.
type Reply struct {
	Name        string  `json:"name"`
	NameSimple  string  `json:"name_simple"`
	Description string  `json:"description"`
	Material    string  `json:"material"`
	WeightLbs   float64 `json:"weight_lbs"`
	Value       int     `json:"value"`
}

// ReplySchema is the strict JSON schema for Reply. Strict mode needs every
// property listed as required and no additional properties. Ranges are not
// expressed as schema bounds (they are enforced in code, and the prompt
// states them), so the same schema serves all three tiers.
func ReplySchema() map[string]any {
	str := func(desc string) map[string]any {
		return map[string]any{`type`: `string`, `description`: desc}
	}
	return map[string]any{
		`type`:                 `object`,
		`additionalProperties`: false,
		`required`:             []string{`name`, `name_simple`, `description`, `material`, `weight_lbs`, `value`},
		`properties`: map[string]any{
			`name`:        str(`2 to 5 words, Title Case, no numbers.`),
			`name_simple`: str(`One lowercase noun a player would type to refer to it, such as locket or vase.`),
			`description`: str(`2 to 4 sentences, under 400 characters, third person.`),
			`material`:    str(`What it is chiefly made of, one or two words.`),
			`weight_lbs`: map[string]any{
				`type`:        `number`,
				`description`: `Its real weight in pounds, from 0.1 to 25.`,
			},
			`value`: map[string]any{
				`type`:        `integer`,
				`description`: `Its value in gold, inside the range the prompt gives.`,
			},
		},
	}
}

// ParseReply decodes the model's JSON content. Unknown fields are an error:
// under a strict schema they mean the reply is not what was asked for.
func ParseReply(content string) (Reply, error) {
	var r Reply
	dec := json.NewDecoder(bytes.NewReader([]byte(content)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return Reply{}, fmt.Errorf(`bauble reply: %w`, err)
	}
	return r, nil
}

// Limited is a Reply after the numeric limits have been applied, with what
// the model originally proposed kept for the catalog's audit fields.
type Limited struct {
	Reply          Reply
	Tier           ValueTier
	ProposedValue  int
	ProposedWeight float64
}

// ApplyLimits clamps the reply's value into the tier's range and its weight
// into the bauble weight bounds. The text fields are passed through
// untouched here; validating them is Phase 4 work.
func ApplyLimits(r Reply, t ValueTier) Limited {
	return ApplyLimitsFor(r, t, SourceSearch)
}

// ApplyLimitsFor is ApplyLimits for a bauble from source: a pickpocketed
// one is also held to pocket size (MaxWeightFor), whatever the model said.
func ApplyLimitsFor(r Reply, t ValueTier, s Source) Limited {
	if !t.Valid() {
		t = TierCheap
	}
	out := Limited{Reply: r, Tier: t, ProposedValue: r.Value, ProposedWeight: r.WeightLbs}
	out.Reply.Value = t.ClampValue(r.Value)
	out.Reply.WeightLbs = ClampWeightFor(r.WeightLbs, s)
	return out
}
