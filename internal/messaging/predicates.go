package messaging

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// RoomVisibility is the minimal interface CanSeeClearly / CanSeeShapes
// need from a room. *rooms.Room satisfies this implicitly. Decoupled
// so messaging/ does not import rooms/ — rooms/ imports messaging/,
// and an interface here keeps the dependency arrow one-way.
//
// Graded lighting arc, plan 1 task 4: this interface's method was renamed
// from the old three-value visibility accessor's name to LightLevel() int.
// Room.LightLevel() (internal/rooms/lighting.go) reports the room's light
// on the graded -100..100 scale (Task 3). Task 5 deleted that old accessor
// and migrated its remaining callers, so LightLevel is this interface's
// only implementation obligation.
type RoomVisibility interface {
	LightLevel() int
}

// ParticipantSight is THE optics primitive. It answers what an observer can
// make out, and nothing else: blindness, room light, NightVision,
// InfraredVision.
//
// It does NOT consult sleep. Sleep is an attention property, not an optical
// one -- a sleeping character's eyes work, they are simply not reading -- and
// the policies below compose it where it belongs. Conflating the two is what
// left three predicates each carrying a comment explaining the split.
//
// WHO READS IT DIRECTLY, and why sleep's absence is load-bearing for them:
// messaging.SendTrio hides a name from its reader by this verdict, and
// actions.InitiateCast refuses a cast at something the caster cannot see. Both
// judge a PARTY to an event, and a sleeper struck in a lit room must still be
// told what hit them. Observers who are not a party go through CanSeeClearly
// and CanSeeShapes instead, which do compose attention, so a sleeper still
// receives no room lines.
//
// Full when the room's light or NightVision allow clear sight; shapes for an
// unblinded observer in a dim room, or with infrared in the dark; none
// otherwise. A nil observer sees fully, matching the policies below.
//
// PLAN 1 NOTE. The NightVision and InfraredVision branches below are the
// pre-graded-lighting flag shortcuts, kept deliberately. Plan 2 of the graded
// lighting arc replaces them with the window model, where an ability shifts
// where the observer's usable band sits rather than granting sight outright.
// They are left alone here because the window model is a real behaviour
// change for those holders (today a NightVision holder sees fully in a pitch
// dark room, and under the window model they are blind below 1), and changing
// the scale and their behaviour in one plan would make this plan's
// behaviour-preservation guarantee impossible to assert.
func ParticipantSight(observer *characters.Character, room RoomVisibility) SightDecision {
	if observer == nil {
		return SightFull
	}
	if observer.Perception != nil && observer.Perception.State() == perception.Blinded {
		return SightNone
	}
	if room == nil {
		// Reflection-free nil-interface guard: a typed-nil *rooms.Room
		// would panic on LightLevel; callers must pass nil interface,
		// not a typed-nil. The room/Room.SendText path always has a real
		// receiver, so this is safe in practice. (This was roomIsLit's
		// job before it was folded into this function; it had exactly
		// one caller, this one.)
		return SightFull
	}
	// Fetched once into a local rather than called from each case below.
	// GetBalanceConfig takes configDataLock (twice: once inside its own
	// ensureConfigValidated call, once itself) and returns Balance BY
	// VALUE -- a struct of well over 400 fields (424 counted directly off
	// internal/configs/config.balance.go at time of writing) -- so a
	// tagless switch that called it from both case expressions would pay
	// that cost twice on the dark path, where the first case is false and
	// the second is evaluated.
	balance := configs.GetBalanceConfig()
	light := room.LightLevel()
	switch {
	case light >= int(balance.LightDimBelow):
		return SightFull
	case light >= int(balance.LightBlindBelow):
		return SightShapes
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
