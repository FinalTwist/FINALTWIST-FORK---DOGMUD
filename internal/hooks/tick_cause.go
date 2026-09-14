package hooks

import (
	"slices"

	"github.com/GoMudEngine/GoMud/internal/conditions"
)

// tickCauseFor reports the death-cause tag a damaging health tick from spec
// should stamp onto Character.LastTickCause: "poison" for a record carrying
// the Poison flag, "bleeding out" for one carrying Bleeding, "" for anything
// else (including a nil spec).
//
// This exists because Buffs.Trigger() decrements TriggersLeft before
// returning the triggered buff, so a tick that is the record's LAST trigger
// arrives at the caller already Expired. deathCauseFor's flag read used to
// skip expired records outright, so the last tick of a bleed or a poison
// killed the holder while reporting "their own
// foolishness". Both round-tick paths (player and mob) call this right where
// the harm lands and stamp LastTickCause only on a non-empty result, so a
// record that is neither poison nor bleeding leaves whatever cause the last
// qualifying tick left behind untouched.
func tickCauseFor(spec *conditions.ConditionSpec) string {
	if spec == nil {
		return ""
	}
	if slices.Contains(spec.Flags, conditions.Poison) {
		return "poison"
	}
	if slices.Contains(spec.Flags, conditions.Bleeding) {
		return "bleeding out"
	}
	return ""
}
