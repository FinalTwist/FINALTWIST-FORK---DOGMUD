package messaging

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// RoomVisibility is the minimal interface CanSeeClearly / CanSeeShapes
// need from a room. *rooms.Room satisfies this implicitly. Decoupled
// so messaging/ does not import rooms/ — rooms/ imports messaging/,
// and an interface here keeps the dependency arrow one-way.
type RoomVisibility interface {
	GetVisibility() int
}

// roomIsLit returns true if the room is bright enough to read
// (visibility >= 1). Helper so callers don't need to know the
// threshold value.
func roomIsLit(room RoomVisibility) bool {
	if room == nil {
		return true
	}
	// Reflection-free nil-interface guard: a typed-nil *rooms.Room
	// would panic on GetVisibility; callers must pass nil interface,
	// not a typed-nil. The room/Room.SendText path always has a real
	// receiver, so this is safe in practice.
	return room.GetVisibility() >= 1
}

// ParticipantSight is THE optics primitive. It answers what an observer can
// make out, and nothing else: blindness, room light, NightVision,
// InfraredVision.
//
// It does NOT consult sleep. Sleep is an attention property, not an optical
// one -- a sleeping character's eyes work, they are simply not reading -- and
// the policies below compose it where it belongs. Conflating the two is what
// left three predicates each carrying a comment explaining the split.
func ParticipantSight(observer *characters.Character, room RoomVisibility) SightDecision {
	if observer == nil {
		return SightFull
	}
	if observer.Perception != nil && observer.Perception.State() == perception.Blinded {
		return SightNone
	}
	if room == nil || roomIsLit(room) {
		return SightFull
	}
	if observer.HasFlagFromAnySource(conditions.NightVision) {
		return SightFull
	}
	if observer.HasFlagFromAnySource(conditions.InfraredVision) {
		return SightShapes
	}
	return SightNone
}

// awake reports attention. Kept separate from optics on purpose; see
// ParticipantSight.
func awake(observer *characters.Character) bool {
	return observer == nil || !observer.HasConditionFlag(conditions.Sleeping)
}

// CanSeeClearly returns true if the observer can read normal-text
// visual broadcasts in this room. Composes Perception state, room
// lighting, and the NightVision condition flag.
//
// Blinded observers (any source) return false unconditionally.
// A nil observer defaults to true (defensive — pre-init characters
// during boot must not be silently dropped).
//
// Sleep is a perception state, even though it is carried as a condition flag
// rather than by the Perception machine. This pipeline had no concept of it
// at all until 2026-08-31, so a sleeping player kept receiving every visual
// broadcast in the room: NPC dialogue, ambient flavour, arrivals.
//
// AUDIO IS DELIBERATELY UNAFFECTED. Room.SendText bypasses this gate, so a
// shout still reaches a sleeper and still wakes them (shout.go owns that
// wake trigger). Gating audio here would make sleep unwakeable by sound.
func CanSeeClearly(observer *characters.Character, room RoomVisibility) bool {
	return awake(observer) && ParticipantSight(observer, room) == SightFull
}

// CanSeeSightImpairedOnly is CanSeeClearly WITHOUT the sleep gate: it reports
// whether sight is impaired by ROOM DARKNESS or BLINDNESS alone.
//
// WHY THIS EXISTS. CanSeeClearly is not a messaging-only predicate.
// internal/combat/combat.go feeds it into combatContext.sourceCanSee and
// targetCanSee, which drive Balance.DarknessCombatPenalty onto the attack score
// and onto every candidate defence score. When the sleep gate was added to
// CanSeeClearly on 2026-08-31, that silently applied a DARKNESS penalty to a
// sleeping defender standing in a LIT room.
//
// The final outcome was masked, because a sleeping victim is already auto-crit
// through AttackSide.ForceCrit and the contest result is overridden anyway. But
// the contest still ran, and its margins and z-scores are what
// combat-analytics.jsonl records and what tools/balance reads. Corrupting that
// telemetry with a phantom darkness term is not acceptable, and doubling a
// sleeper's disadvantage was never asked for.
//
// So combat keeps the pre-sleep semantics and messaging gets the sleep gate.
//
// It feeds Balance.DarknessCombatPenalty, so widening it would hand every
// infrared character a silent balance change; it is the optics question with
// NO attention test, and it is SightFull specifically -- infrared does not
// satisfy it.
//
// M4d closed the seam this comment used to describe: ParticipantSight is now
// the shared optics primitive both this function and CanSeeClearly are built
// on.
func CanSeeSightImpairedOnly(observer *characters.Character, room RoomVisibility) bool {
	return ParticipantSight(observer, room) == SightFull
}

// CanSeeShapes returns true if the observer can detect SOMETHING is
// happening — either full clarity (subsumes CanSeeClearly) OR
// infrared in the dark. Blindness gates this too — broken eyes don't
// see infrared. So does sleep: closed eyes see no shapes.
//
// A nil observer defaults to true (matches CanSeeClearly's defensive
// behavior).
//
// "Full sight OR shapes", written as two equalities on purpose: the
// SightDecision constants run BEST-TO-WORST (SightFull = 0, SightShapes = 1,
// SightNone = 2), so an ordered comparison such as `<= SightShapes` would
// read backwards and a `>=` would also match SightNone.
func CanSeeShapes(observer *characters.Character, room RoomVisibility) bool {
	if !awake(observer) {
		return false
	}
	d := ParticipantSight(observer, room)
	return d == SightFull || d == SightShapes
}
