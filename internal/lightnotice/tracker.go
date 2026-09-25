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
	// TriggerQuiet records the current band and never speaks: login.
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
	return notice{cause: attribute(prev, now, tr), transition: tr, indoor: now.indoor}, true, next
}

// transitionOf names a band change. Bands run darkest to brightest.
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
// A room change is movement. Otherwise, if the light LEVEL did not move in the
// band's direction, no light term explains the change and the observer's own
// sight did (a draught wearing off): that is checked BEFORE the terms, because
// the sky drifts a little almost every round and would otherwise take the
// blame for everything. Then the first term that moved, in the order carried
// light, the room's own light, weather, sky.
func attribute(prev record, now observation, tr Transition) Cause {
	if prev.roomId != now.roomId {
		return CauseMovement
	}
	a, b := prev.terms, now.terms
	darker := tr == DarkerFaces || tr == DarkerShapes || tr == DarkerDark
	if (darker && b.Level >= a.Level) || (!darker && b.Level <= a.Level) {
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
