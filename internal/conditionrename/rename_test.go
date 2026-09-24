package conditionrename

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApply_Keys(t *testing.T) {
	cases := map[string]string{
		"buffid: 83":               "conditionid: 83",
		"buffids: [1, 2]":          "conditionids: [1, 2]",
		"buff_ids: [3]":            "condition_ids: [3]",
		"buff_id: 9":               "condition_id: 9",
		"critbuffids: [4]":         "critconditionids: [4]",
		"wornbuffids: [5]":         "wornconditionids: [5]",
		"trapbuffids: [6]":         "trapconditionids: [6]",
		"playerbuffids: [7]":       "playerconditionids: [7]",
		"mobbuffids: [8]":          "mobconditionids: [8]",
		"nativebuffids: []":        "nativeconditionids: []",
		"prizebuffids: []":         "prizeconditionids: []",
		"start_remove_buffs: [47]": "start_remove_conditions: [47]",
		"buffs:":                   "conditions:",
		"permabuff: true":          "permanent: true",
		"apply_buff:":              "apply_condition:",
		"  buff: 3":                "  condition: 3",
		`yaml:"buffid"`:            `yaml:"conditionid"`,
		`json:"wornBuffIds"`:       `json:"wornConditionIds"`,
	}
	for in, want := range cases {
		assert.Equal(t, want, Apply(in), "Apply(%q)", in)
	}
}

func TestApply_Values(t *testing.T) {
	cases := map[string]string{
		"effect_type: buff":                         "effect_type: condition",
		"behavior_archetype: melee_self_buff":       "behavior_archetype: melee_self_empower",
		"check: mob_has_buff":                       "check: mob_has_condition",
		"do: add_buff":                              "do: add_condition",
		"do: remove_buff":                           "do: remove_condition",
		"category: buff_friendly":                   "category: condition_friendly",
		"- type: on_hit_buff":                       "- type: on_hit_condition",
		"- type: aura_ally_buff":                    "- type: aura_ally_condition",
		"- type: on_reflect_buff":                   "- type: on_reflect_condition",
		"- type: aura_enemy_debuff":                 "- type: aura_enemy_condition",
		`<ansi fg="buff">`:                          `<ansi fg="condition">`,
		"buff-apply: 109":                           "condition-apply: 109",
		"{{ buffname $id }} {{ buffduration $id }}": "{{ conditionname $id }} {{ conditionduration $id }}",
		`"pinnacle_bandolier_buffs"`:                `"pinnacle_bandolier_conditions"`,
		"/buffs":                                    "/conditions",
		"BuffsEnabled":                              "ConditionsEnabled",
	}
	for in, want := range cases {
		assert.Equal(t, want, Apply(in), "Apply(%q)", in)
	}
}

func TestApply_CasePreservingProse(t *testing.T) {
	assert.Equal(t, "add new condition", Apply("add new buff"))
	assert.Equal(t, "Condition does not exist", Apply("Buff does not exist"))
	assert.Equal(t, "CONDITION", Apply("BUFF"))
	assert.Equal(t, "a harmful condition", Apply("a debuff"))
	assert.Equal(t, "Harmful conditions fade", Apply("Debuffs fade"))
	assert.Equal(t, "Permanent", Apply("PermaBuff"))
}

func TestApply_ProtectedWords(t *testing.T) {
	for _, s := range []string{
		"bytes.Buffer", "a ring buffer", "WORLD_EVENT_BUFFER",
		"the wind buffets the cliff", "buffeted by gusts",
		"surfaces buffed to a dull shine", "rebuff like shopkeepers",
		"Buffalo",
	} {
		assert.Equal(t, s, Apply(s), "protected text must not change")
	}
}

func TestApply_Idempotent(t *testing.T) {
	in := "buffid: 3\neffect_type: buff\nmelee_self_buff\npermabuff: true\na debuff\n"
	once := Apply(in)
	assert.Equal(t, once, Apply(once))
}

func TestContainsOldSpelling(t *testing.T) {
	assert.True(t, ContainsOldSpelling("buffid: 3"))
	assert.True(t, ContainsOldSpelling("a Debuff"))
	assert.False(t, ContainsOldSpelling("bytes.Buffer and a buffet and Buffalo"))
	assert.False(t, ContainsOldSpelling(Apply("buffid: 3 permabuff melee_self_buff")))
	assert.False(t, ContainsOldSpelling("bufbufferf"), "a protected word must not splice its neighbours into a match")
}
