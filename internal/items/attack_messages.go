package items

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/narration"
)

var (
	attackMessages map[ItemSubType]*WeaponAttackMessageGroup = map[ItemSubType]*WeaponAttackMessageGroup{}
)

type SkillTier string

const (
	Beginner SkillTier = "beginner"
	Expert   SkillTier = "expert"
	Master   SkillTier = "master"
)

type WeaponAttackMessageGroup struct {
	OptionId ItemSubType `yaml:"optionid"`
	Options  AttackTypes `yaml:"options"`
}

type AttackTypes map[Intensity]AttackOptions

type AttackOptions struct {
	Together TogetherMessages `yaml:"together"`
	Separate SeparateMessages `yaml:"separate"`
}

type TogetherMessages struct {
	ToAttacker SkillTieredMessages `yaml:"toattacker"`
	ToDefender SkillTieredMessages `yaml:"todefender"`
	ToRoom     SkillTieredMessages `yaml:"toroom"`
}

type SeparateMessages struct {
	ToAttacker     SkillTieredMessages `yaml:"toattacker"`
	ToDefender     SkillTieredMessages `yaml:"todefender"`
	ToAttackerRoom SkillTieredMessages `yaml:"toattackerroom"`
	ToDefenderRoom SkillTieredMessages `yaml:"todefenderroom"`
}

type SkillTieredMessages struct {
	Beginner MessageOptions `yaml:"beginner"`
	Expert   MessageOptions `yaml:"expert,omitempty"`
	Master   MessageOptions `yaml:"master,omitempty"`
}

type MessageOptions []ItemMessage

func (am ItemMessage) SetTokenValue(tokenName TokenName, tokenValue string) ItemMessage {
	return ItemMessage(strings.Replace(string(am), string(tokenName), tokenValue, -1))
}

// Get chooses a message using the default picker (narration.DefaultPicker,
// which routes through util.Rand).
func (mo MessageOptions) Get(seedNum ...int) ItemMessage {
	return mo.GetWith(nil, seedNum...)
}

// GetWith is Get with an explicit picker, for the snapshot harness. A nil
// picker means production behaviour: narration.DefaultPicker, i.e. util.Rand.
//
// The picker is consulted only in the no-seed branch, where a random pick
// used to happen directly. seedNum's explicit-index-override behaviour is
// untouched.
func (mo MessageOptions) GetWith(pick narration.Picker, seedNum ...int) ItemMessage {
	if pick == nil {
		pick = narration.DefaultPicker
	}

	if ct := len(mo); ct > 0 {

		if len(seedNum) == 0 || seedNum[0] == 0 {
			return mo[pick(ct)]
		}

		if seedNum[0] == 0 {
			return mo[0]
		}

		return mo[seedNum[0]%len(mo)]
	}

	return ItemMessage("")
}

// GetForSkillLevel selects a message based on character's skill level, using
// the default picker (narration.DefaultPicker, which routes through
// util.Rand). Returns messages from available tiers based on skill:
//   - Skill 1-33: beginner only
//   - Skill 34-66: beginner + expert
//   - Skill 67-100: beginner + expert + master
func (stm SkillTieredMessages) GetForSkillLevel(skillLevel int, msgSeed ...int) ItemMessage {
	return stm.GetForSkillLevelWith(nil, skillLevel, msgSeed...)
}

// GetForSkillLevelWith is GetForSkillLevel with an explicit picker, for the
// snapshot harness. A nil picker means production behaviour:
// narration.DefaultPicker, i.e. util.Rand.
//
// This is the store the core combat loop calls for all three viewpoints at
// once (ToAttacker, ToDefender, ToRoom), so it is the store that most needed
// a picker reachable without mutating shared state.
//
// The picker is consulted only in the no-seed branch, where a random pick
// used to happen directly. msgSeed's explicit-index-override behaviour is
// untouched.
func (stm SkillTieredMessages) GetForSkillLevelWith(pick narration.Picker, skillLevel int, msgSeed ...int) ItemMessage {
	if pick == nil {
		pick = narration.DefaultPicker
	}

	// Collect available message pools
	var allMessages []ItemMessage

	// Always include beginner messages
	allMessages = append(allMessages, stm.Beginner...)

	// Add expert if skill >= 34
	if skillLevel >= 34 {
		allMessages = append(allMessages, stm.Expert...)
	}

	// Add master if skill >= 67
	if skillLevel >= 67 {
		allMessages = append(allMessages, stm.Master...)
	}

	// Select from combined pool
	if len(allMessages) == 0 {
		return ItemMessage("")
	}

	// Use seed if provided
	if len(msgSeed) > 0 && msgSeed[0] != 0 {
		return allMessages[msgSeed[0]%len(allMessages)]
	}

	return allMessages[pick(len(allMessages))]
}

// PoolFor returns the tier union for a skill level as the core's plain-string
// form. The union is cumulative and matches what GetForSkillLevelWith built:
// beginner always, plus expert at 34, plus master at 67.
//
// Assembly stays in the store. The core coordinates the index and substitutes
// tokens; it knows nothing about tiers (internal/narration/context.md).
//
// Returns nil rather than an empty slice when nothing is authored, so a role
// the core sees as absent is absent rather than present-and-empty.
func (stm SkillTieredMessages) PoolFor(skillLevel int) []string {
	out := make([]string, 0, len(stm.Beginner)+len(stm.Expert)+len(stm.Master))
	for _, m := range stm.Beginner {
		out = append(out, string(m))
	}
	if skillLevel >= 34 {
		for _, m := range stm.Expert {
			out = append(out, string(m))
		}
	}
	if skillLevel >= 67 {
		for _, m := range stm.Master {
			out = append(out, string(m))
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Render renders one coordinated attacker/defender/room triad for a blow whose
// participants share a room.
//
// ALL ROLES COME FROM ONE VARIANT INDEX. The authored pools pair up by index,
// so picking per role narrates three different events to three audiences. That
// is the defect this migration removes, and it shipped twice before: melee
// defence (PR #112) and taunt (PR #115).
//
// The role mapping is the one thing here worth reading slowly. An attacker
// ACTS and a defender is ACTED UPON, so toattacker is the Actor and todefender
// is the Actee. Swapping those two lines inverts every combat message in the
// game, and three separately named pools becoming adjacent fields of one
// struct literal is exactly how that mistake gets made.
// combat_messages.golden keys its rows by the AUTHORED name, which is what
// catches it.
//
// ActeeObserver is deliberately left empty: when the participants share a room
// there is only one observer audience.
//
// A nil picker means production behaviour (narration.DefaultPicker).
func (m TogetherMessages) Render(skillLevel int, tokenReplacements map[TokenName]string, pick narration.Picker) narration.Roles {
	return narration.Render(
		narration.Variants{
			Actor:    m.ToAttacker.PoolFor(skillLevel),
			Actee:    m.ToDefender.PoolFor(skillLevel),
			Observer: m.ToRoom.PoolFor(skillLevel),
		},
		tokenStrings(tokenReplacements),
		pick,
	)
}

// Render renders one coordinated quartet for a ranged blow where attacker and
// defender are in different rooms, so there are genuinely two observer
// audiences. ActeeObserver is the observers where the DEFENDER is; this is the
// case narration.Roles grew its fourth field for
// (internal/narration/render.go:35-38).
//
// The two room roles are different audiences seeing different things and must
// never be treated as interchangeable.
//
// A nil picker means production behaviour (narration.DefaultPicker).
func (m SeparateMessages) Render(skillLevel int, tokenReplacements map[TokenName]string, pick narration.Picker) narration.Roles {
	return narration.Render(
		narration.Variants{
			Actor:         m.ToAttacker.PoolFor(skillLevel),
			Actee:         m.ToDefender.PoolFor(skillLevel),
			Observer:      m.ToAttackerRoom.PoolFor(skillLevel),
			ActeeObserver: m.ToDefenderRoom.PoolFor(skillLevel),
		},
		tokenStrings(tokenReplacements),
		pick,
	)
}

// Presumably to ensure the datafile hasn't messed something up.
func (w *WeaponAttackMessageGroup) Id() ItemSubType {
	return w.OptionId
}

// Presumably to ensure the datafile hasn't messed something up.
// Validate checks that every required intensity is present and that every
// authored group can be rendered from ONE coordinated index.
//
// Equality is checked PER TIER, not on the union totals. The runtime union is
// cumulative (beginner, plus expert at 34, plus master at 67), so equal totals
// with unequal tiers would still pair an expert line against a master one at
// the same index. Per-tier equality is exactly equivalent to union equality at
// all three skill levels.
//
// minVariants is 1, not the defence store's 5: the smallest authored group in
// the shipped store is generic/coupdegrace/separate/beginner at one line, and
// 414 of 534 groups hold fewer than five.
//
// The attack store had none of this until M3 item 8. It checked intensity
// presence and nothing else, while the defence sibling next door
// (defensive_messages.go) has always checked emptiness and equal lengths.
func (w *WeaponAttackMessageGroup) Validate() error {

	// Make sure all important options are present.
	optionsToCheck := []Intensity{Prepare, Wait, Miss, Weak, Normal, Heavy, Critical, Fumble}
	for _, option := range optionsToCheck {
		if _, ok := w.Options[option]; !ok {
			return fmt.Errorf("missing option[`%s`] for %s", option, w.OptionId)
		}
	}

	tiers := []struct {
		name string
		get  func(SkillTieredMessages) MessageOptions
	}{
		{"beginner", func(s SkillTieredMessages) MessageOptions { return s.Beginner }},
		{"expert", func(s SkillTieredMessages) MessageOptions { return s.Expert }},
		{"master", func(s SkillTieredMessages) MessageOptions { return s.Master }},
	}

	// Sorted so a file with several faults always reports the same one first,
	// rather than whichever the map happened to yield.
	intensities := make([]string, 0, len(w.Options))
	for intensity := range w.Options {
		intensities = append(intensities, string(intensity))
	}
	sort.Strings(intensities)

	for _, name := range intensities {
		intensity := Intensity(name)
		opts := w.Options[intensity]

		for _, tier := range tiers {
			together := narration.Variants{
				Actor:    messageStrings(tier.get(opts.Together.ToAttacker)),
				Actee:    messageStrings(tier.get(opts.Together.ToDefender)),
				Observer: messageStrings(tier.get(opts.Together.ToRoom)),
			}
			if anyPool(together) {
				if err := narration.ValidateVariants(together, 1,
					narration.RoleActor, narration.RoleActee, narration.RoleObserver); err != nil {
					return fmt.Errorf("%s option[`%s`].together.%s: %w", w.OptionId, intensity, tier.name, err)
				}
			}

			separate := narration.Variants{
				Actor:         messageStrings(tier.get(opts.Separate.ToAttacker)),
				Actee:         messageStrings(tier.get(opts.Separate.ToDefender)),
				Observer:      messageStrings(tier.get(opts.Separate.ToAttackerRoom)),
				ActeeObserver: messageStrings(tier.get(opts.Separate.ToDefenderRoom)),
			}
			if anyPool(separate) {
				if err := narration.ValidateVariants(separate, 1,
					narration.RoleActor, narration.RoleActee,
					narration.RoleObserver, narration.RoleActeeObserver); err != nil {
					return fmt.Errorf("%s option[`%s`].separate.%s: %w", w.OptionId, intensity, tier.name, err)
				}
			}
		}
	}

	return nil
}

// anyPool reports whether a group was authored at all, so a subtype with no
// separate block is skipped rather than reported as four missing roles.
func anyPool(v narration.Variants) bool {
	return len(v.Actor) > 0 || len(v.Actee) > 0 ||
		len(v.Observer) > 0 || len(v.ActeeObserver) > 0
}

func (w *WeaponAttackMessageGroup) Filepath() string {
	return fmt.Sprintf("%s.yaml", w.OptionId)
}

func GetPreAttackMessage(subType ItemSubType, messageType Intensity) AttackOptions {

	// Check whether this item subtype has any attack messages
	if attackMsgOptions, ok := attackMessages[subType]; ok {
		if attackMsgOptions, ok := attackMsgOptions.Options[messageType]; ok {
			// return a random message
			return attackMsgOptions
		}
	}

	// Fall back to generic, but NEVER recurse into ourselves. If Generic itself
	// lacks the key this would call itself forever and overflow the stack,
	// taking the server down. That was survivable only because generic.yaml
	// happened to define every intensity in use, which is a property of the data
	// rather than of the code. Returning the zero value degrades to no message.
	if subType == Generic {
		return AttackOptions{}
	}

	return GetPreAttackMessage(Generic, messageType)
}

func GetAttackMessage(subType ItemSubType, pctDamage int) AttackOptions {

	var intensity Intensity
	if pctDamage >= 101 {
		intensity = Critical
	} else if pctDamage >= 75 {
		intensity = Heavy
	} else if pctDamage >= 30 {
		intensity = Normal
	} else if pctDamage >= 1 {
		intensity = Weak
	} else {
		intensity = Miss
	}

	// Check whether this item subtype has any attack messages
	if attackMsgOptions, ok := attackMessages[subType]; ok {
		if attackMsgOptions, ok := attackMsgOptions.Options[intensity]; ok {
			// return a random message
			return attackMsgOptions
		}
	}
	// Fall back to generic, but NEVER recurse into ourselves -- the same guard
	// GetPreAttackMessage carries above, and for the same reason. If Generic
	// itself lacks the intensity (or attackMessages was never loaded at all)
	// this calls itself forever and overflows the stack, taking the process
	// down. Returning the zero value degrades to no message instead.
	if subType == Generic {
		return AttackOptions{}
	}

	return GetAttackMessage(Generic, pctDamage)
}
