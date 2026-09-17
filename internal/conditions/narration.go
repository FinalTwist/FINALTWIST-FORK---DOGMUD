package conditions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/narration"
	"github.com/GoMudEngine/GoMud/internal/textutil"
)

// Phase selects which of a condition's three narrated moments to render. A condition
// holds one line per phase and audience, so the phase IS the selector: there
// is no pool and nothing to pick.
type Phase uint8

const (
	PhaseStart Phase = iota
	PhaseTrigger
	PhaseEnd
)

// Narration assembles the variants for one phase.
//
// The holder's line is the ACTEE: the condition happens to them. The room's line is
// the Observer. Actor is empty and reserved for the caster, which M6 authors
// once events.Condition carries a caster (owner ruling, 2026-09-12). Start and End
// go through StartUserNotice / EndUserNotice, so the secret, hidden,
// silent-start and generic-fallback rules stay in their one door.
func (b *ConditionSpec) Narration(p Phase) narration.Variants {
	var holder, room string
	switch p {
	case PhaseStart:
		holder, room = b.StartUserNotice(), b.StartRoomText
	case PhaseTrigger:
		holder, room = b.TriggerUserText, b.TriggerRoomText
	case PhaseEnd:
		holder, room = b.EndUserNotice(), b.EndRoomText
	}
	return narration.Variants{Actee: textutil.Pool(holder), Observer: textutil.Pool(room)}
}

// Narrate renders one phase for its audiences. It takes the HOLDER, not a
// token context: the holder is always the actee (a condition happens to them),
// and building the context here means no call site can put the name in the
// wrong slot. Actor stays empty until events.Condition carries a caster (M6).
func (b *ConditionSpec) Narrate(p Phase, holderName, holderPlainName string) narration.Roles {
	return textutil.Narrate(b.Narration(p), textutil.TokenContext{
		ActeeName:      holderName,
		ActeePlainName: holderPlainName,
	})
}

// AuthoredStartLine renders start_user_text as written, ignoring the notice
// rules. It is the door for the applier of a silent-start condition, which narrates
// the start itself because the condition never travels the event that would:
// sleep (15), arrest (88), stun (84) and broken limb (83). Enchant Withdrawal
// (123) is not silent-start, but disenchant applies it through
// AddConditionMagnitude, which is also synchronous and never travels events.Condition,
// so it reads the same door for the same reason.
//
// It takes the HOLDER for the same reason Narrate does: the start line is
// authored against {actee} and {actee_plain}, so a call site free to pick a
// slot could fill the actor and render the name as an empty string. No
// silent-start condition's start_user_text carries a name token today, which
// is exactly why that defect would stay invisible until one is authored.
func (b *ConditionSpec) AuthoredStartLine(holderName, holderPlainName string) string {
	return textutil.SubstituteTokens(b.StartUserText, textutil.TokenContext{
		ActeeName:      holderName,
		ActeePlainName: holderPlainName,
	})
}

// validateNarration refuses a phase whose authored text cannot be rendered:
// a whitespace-only line, which ValidateVariants reports as an empty variant.
// It checks the RAW fields, not the notices, so a silent-start condition's hidden
// start line is checked too. The trigger phase has no notice wrapper, so for
// it the raw fields and Narration agree; it is listed here so all three
// phases are checked in one place.
func (b *ConditionSpec) validateNarration() error {
	phases := []struct{ name, user, room string }{
		{"start", b.StartUserText, b.StartRoomText},
		{"trigger", b.TriggerUserText, b.TriggerRoomText},
		{"end", b.EndUserText, b.EndRoomText},
	}
	for _, ph := range phases {
		if ph.user == "" && ph.room == "" {
			continue
		}
		v := narration.Variants{Actee: textutil.Pool(ph.user), Observer: textutil.Pool(ph.room)}
		// No expected role set: a condition may legitimately author only a holder
		// line or only a room line, so no fixed shape exists to declare. The
		// blank-variant check is what this call is for.
		if err := narration.ValidateVariants(v, 1); err != nil {
			return fmt.Errorf("conditionId %d (%s) %s text: %w", b.ConditionId, b.Name, ph.name, err)
		}
	}
	return nil
}
