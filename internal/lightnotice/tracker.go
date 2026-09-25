package lightnotice

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Trigger names why a check is running, which decides what it may announce.
type Trigger uint8

const (
	// TriggerMove runs after a player arrives in a room. It announces only a
	// darker band or dazzle: walking into light needs no notice, the room
	// description already says it.
	TriggerMove Trigger = iota
	// TriggerCombatRound runs once per combat round the player is fighting in.
	TriggerCombatRound
	// TriggerCommand runs before every command the player issues, so the
	// notice lands before the command's own output.
	TriggerCommand
	// TriggerQuiet records the current band and never speaks: login. Waking
	// and the end of blindness are NOT this trigger: they are handled by the
	// quiet flag decide sets on the record, which a per-round attention sweep
	// sets when a player falls asleep or is blinded, so the next attentive
	// check (whatever trigger it runs under) resyncs silently.
	TriggerQuiet
)

// observation is one moment of one player's light, gathered by Check.
type observation struct {
	roomId  int
	band    messaging.Band
	terms   rooms.LightTerms
	indoor  bool
	asleep  bool
	blinded bool
	// bandAt reports the band the observer's CURRENT sight reads at a given
	// light level. attribute uses it to ask whether the old light alone would
	// already give the new band, in which case the observer's sight changed,
	// not the light. A nil bandAt skips that check entirely (it is optional
	// so decide's other rules stay testable without it).
	bandAt func(light int) messaging.Band
}

// record is what was last announced (or silently recorded) to a player.
type record struct {
	roomId int
	band   messaging.Band
	terms  rooms.LightTerms
	// quiet says the player was asleep or blinded since the last record, so
	// the next attentive check re-records silently. Their end must never read
	// as the light changing.
	quiet bool
}

// notice is what decide asks Check to say.
type notice struct {
	cause      Cause
	transition Transition
	indoor     bool
}

// decide applies the trigger rules. It returns the notice to send, if any, and
// the record to store. It is pure so every rule is table-testable.
func decide(prev record, known bool, now observation, trigger Trigger) (notice, bool, record) {
	if now.asleep || now.blinded {
		// On a player's very first check, known is false and prev is the zero
		// record (room 0, BandDark). If that first check also finds them
		// asleep or blinded, this returns that zero record, only marked
		// quiet: it is a placeholder, not a claim they are actually in room 0
		// in the dark. The next attentive check overwrites it silently, same
		// as any other quiet resync.
		prev.quiet = true
		return notice{}, false, prev
	}
	next := record{roomId: now.roomId, band: now.band, terms: now.terms}
	if !known || prev.quiet || trigger == TriggerQuiet || now.band == prev.band {
		return notice{}, false, next
	}
	tr := transitionOf(prev.band, now.band)
	if trigger == TriggerMove && (tr == LighterShapes || tr == LighterFaces) {
		return notice{}, false, next
	}
	return notice{cause: attribute(prev, now), transition: tr, indoor: now.indoor}, true, next
}

// transitionOf names a band change. Bands run darkest to brightest. It
// assumes from != to; decide filters an unchanged band before calling it.
func transitionOf(from, to messaging.Band) Transition {
	switch {
	case to == messaging.BandDazzled:
		return IntoDazzle
	case to > from && to == messaging.BandFaces:
		return LighterFaces
	case to > from:
		return LighterShapes
	case to == messaging.BandFaces:
		return DarkerFaces
	case to == messaging.BandShapes:
		return DarkerShapes
	}
	return DarkerDark
}

// attribute names the likeliest cause of a band change.
//
// A room change is movement. Otherwise it asks a counterfactual: would the
// OLD light, read through the observer's CURRENT sight, already give the NEW
// band? If so the light never had to move; the observer's own sight did (a
// draught wearing off or taking hold), and that is checked BEFORE the terms.
//
// A plain direction test (did Level move the way the band moved) is not
// enough: the sky drifts a little almost every round, and when it drifts the
// same direction as an eyes-caused change, a direction test blames the sky
// for a change the observer's sight alone already explains. The
// counterfactual does not have that failure mode, because it holds the light
// fixed at its OLD value and only varies the sight.
//
// Otherwise, the first term that moved, in the order carried light, the
// room's own light, weather, sky.
func attribute(prev record, now observation) Cause {
	if prev.roomId != now.roomId {
		return CauseMovement
	}
	a, b := prev.terms, now.terms
	if now.bandAt != nil && now.bandAt(prev.terms.Level) == now.band {
		return CauseEyes
	}
	switch {
	case a.Carried != b.Carried:
		return CauseCarried
	case a.HasLamp != b.HasLamp || a.Lamp != b.Lamp || a.LightMod != b.LightMod:
		return CauseLamp
	case a.OcclusionSteps != b.OcclusionSteps:
		return CauseWeather
	case skyMoved(a.Sky, b.Sky):
		return CauseSky
	}
	return CauseEyes
}

func skyMoved(a, b float64) bool {
	aAbsent, bAbsent := math.IsInf(a, -1), math.IsInf(b, -1)
	if aAbsent || bAbsent {
		return aAbsent != bAbsent
	}
	return math.Abs(a-b) > 1e-9
}
