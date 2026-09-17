package quests

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/narration"
	"github.com/GoMudEngine/GoMud/internal/textutil"
)

// Narration assembles a text action's variants: the authored `actor` key is
// the Actor line (the triggering player), `observer` the Observer line. An
// action sets one or the other; Validate refuses both.
func (a ActionDef) Narration() narration.Variants {
	return narration.Variants{Actor: textutil.Pool(a.SendText), Observer: textutil.Pool(a.RoomText)}
}

// Narrate renders a text action with the triggering player as the actor.
// A quest has no actee, so {actee} renders empty; RoomTextProblems refuses
// it in the observer line for that reason.
func (a ActionDef) Narrate(ctx textutil.TokenContext) narration.Roles {
	return textutil.Narrate(a.Narration(), ctx)
}

// Narration assembles a reward's variants: the authored `actor` key is the
// Actor line, `observer` the Observer line.
func (r QuestReward) Narration() narration.Variants {
	return narration.Variants{Actor: textutil.Pool(r.PlayerMessage), Observer: textutil.Pool(r.RoomMessage)}
}

// Narrate renders the reward lines with the completing player as the actor.
func (r QuestReward) Narrate(ctx textutil.TokenContext) narration.Roles {
	return textutil.Narrate(r.Narration(), ctx)
}

// validateNarration refuses an action that sets both `actor` and `observer`
// (one narration per action; ExecuteAction used to drop the room line of such
// an action silently), and any text line that is whitespace only, in actions,
// nested sequence actions, and the rewards.
func (r *Quest) validateNarration() error {
	var problems []string
	var walk func(where string, actions []ActionDef)
	walk = func(where string, actions []ActionDef) {
		for j, a := range actions {
			aw := fmt.Sprintf("%s action %d", where, j)
			if a.SendText != "" && a.RoomText != "" {
				problems = append(problems, aw+": sets both actor and observer; an action narrates one line")
			}
			if a.SendText != "" || a.RoomText != "" {
				if err := narration.ValidateVariants(a.Narration(), 1); err != nil {
					// The observer branch is unreachable through Quest.Validate today:
					// validateRoomText runs first and refuses any observer line
					// without {actor}, which a whitespace-only line cannot carry.
					// Kept so validateNarration names the right key when called on
					// its own.
					key := "actor"
					if a.SendText == "" {
						key = "observer"
					}
					problems = append(problems, aw+" "+key+": "+err.Error())
				}
			}
			if a.Sequence != nil {
				walk(aw+" sequence on_complete", a.Sequence.OnComplete)
			}
		}
	}
	for i, t := range r.Triggers {
		walk(fmt.Sprintf("trigger %d", i), t.Actions)
	}
	if r.Rewards.PlayerMessage != "" || r.Rewards.RoomMessage != "" {
		if err := narration.ValidateVariants(r.Rewards.Narration(), 1); err != nil {
			problems = append(problems, "rewards: "+err.Error())
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("quest %d (%s) narration problems:\n%s", r.QuestId, r.Name, strings.Join(problems, "\n"))
	}
	return nil
}
