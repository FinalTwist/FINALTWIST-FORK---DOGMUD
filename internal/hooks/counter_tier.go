package hooks

// U6b Task 10 — the counter tier's spell-exit wiring. A defensive crit
// against a cast earns the defender one free counter-swing, fired here at the
// four spell quadrants (player->mob, player->player, mob->mob, mob->player).
// BOTH directions matter: wiring only the player-attacker direction would
// hand mobs a counter immunity nobody decided.

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// fireSpellCounterTier fires the counter tier at one spell exit. Spell
// targets always share the caster's room, so the reach gate passes true by
// construction (the cross-room shot — internal/actions.ExecuteFire — is the
// one uncounterable attack). The primitive also refuses any cast whose
// authored targeting is not single, so an area cast earns no counter without
// any branch here.
//
// The channel-correct counter narration (U6b Task 11, rendered from the pool
// of the defence that won the contest; a defy win goes to the counter-taunt
// instead) is dispatched with the same audience routing the melee crit-effects
// use (CategoryHitMelee: the counter-swing IS a melee answer). Dispatching
// here is ordering-correct for spells: the cast's own outcome narration has
// already been sent by the time these exits fire. Nil user records represent
// mob participants, which receive no private text.
//
// Recursion is impossible here by construction: casts are never made under
// IsCounter (the counter-swing is a melee-shaped ExecuteSkillMove, never a
// cast), and ExecuteCounter marks its own swing IsCounter.
func fireSpellCounterTier(room *rooms.Room, out combat.ChannelDefenceResult,
	shape combatvocab.Attack, defender, caster *characters.Character,
	defenderUser, casterUser *users.UserRecord) combat.CounterResult {

	if !out.DefensiveCrit {
		return combat.CounterResult{}
	}

	// Words answer words: a defy crit (charm is the one social spell) fires
	// the counter-taunt, the same dispatch taunt's own exit uses. The swing
	// primitive refuses a defy defence, so this branch is the only way a
	// defied cast is answered. The counter-taunt has its own result type;
	// the four call sites use this function as a statement.
	if out.Defence == combatvocab.DefenceDefy {
		var defenderRecipient, casterRecipient messaging.Recipient
		defenderId, casterId := 0, 0
		if defenderUser != nil {
			defenderRecipient, defenderId = defenderUser, defenderUser.UserId
		}
		if casterUser != nil {
			casterRecipient, casterId = casterUser, casterUser.UserId
		}
		actions.FireCounterTaunt(room, defender, caster, defenderId, defenderRecipient, casterId, casterRecipient)
		return combat.CounterResult{}
	}

	res := combat.ExecuteCounter(defender, caster, shape, out.Defence, true)
	if !res.Countered {
		return res
	}

	var countered messaging.Recipient
	counteredId := 0
	if casterUser != nil {
		countered = casterUser
		counteredId = casterUser.UserId
	}
	actions.SendCounterTrio(room, res, countered, counteredId)
	return res
}
