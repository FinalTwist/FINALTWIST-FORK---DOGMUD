package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// sendTrioOnlyCategories is every messaging.Category whose production sends
// have ALL been verified, by grep across internal/ and modules/ on this
// branch (2026-09-20), to originate only from messaging.Say (building a
// messaging.Trio literal) or from acteeDefenceLine (which itself only wraps
// Say). No other Send-family call -- SendText, SendTextVisual*,
// SendToTarget*, SendToSource*, SendToRoomOld, or a free-function wrapper
// such as sendAudioRoomText/sendMovementMessage -- ever carries one of these
// three as a literal argument in production code today.
//
// This is a SMALL, DELIBERATELY NARROW set, not "every narration category."
// A full survey of all ~59 messaging.Category values (2026-09-20) found only
// these three fully migrated; every other combat/spell/speech category still
// has at least one live bypass:
//
//   - CategoryHitMelee/HitBlunt/HitNaturalSharp/HitRanged/HitCaster/
//     HitUnarmed/Dodge/Parry/Block: produced by combat.AttackResult's
//     TaggedMessage buffer (internal/combat/attackresult.go,
//     CategoryForWeaponSubtype / CategoryForDefenseVerb) and drained by raw
//     SendText in internal/hooks/combat_verbosity.go -- PR 2's subject.
//   - CategoryGrappleFlow/CategorySubmission: internal/hooks/
//     Position_Messaging.go and Position_GrappleTick.go send both directly;
//     deferred (personal lines name the other grappler).
//   - CategorySpellFold/Disruption/Elemental/Enhancement/Mental/Vital/
//     Manifestation: internal/hooks/spell_resolution.go, four of five
//     applyPlayerEffect self-cast branches still pair a personal line with a
//     direct sendVisualRoomText room line.
//   - CategoryNPCDialogue: internal/questengine/bridge.go's Narrate sends
//     trigger observer lines directly; every line carries {actor} by
//     validation, so moving it would add name hiding in the dark.
//   - CategoryTauntSuccess/TauntResist/TauntFailure/Rally/Warcry/Shout/
//     Speech/Whisper/NPCDialogue (again, via the audio helper): sent through
//     sendAudioRoomText, a pre-M4 raw helper with its own nightvision-based
//     name hiding, never SendTrio.
//   - CategoryRoomEntry/RoomExit/Logout: sendMovementMessage /
//     SendTextVisualWithAudio / sendVisualRoomText, single-audience movement
//     announcements, never routed through Trio.
//   - CategoryGrappleHigh/OOC/Login/Toxin: zero production senders at all
//     (vacuous; not included -- a guard needs a real sender to ever go red).
//
// This guard states a true invariant today. As PR 2 migrates a category's
// last raw sender, add it here; if a regression reopens a bypass for one of
// these three, add the file below with a comment naming why, same as
// rawEventsMessageAllowed in raw_events_message_guard_test.go.
var sendTrioOnlyCategories = []string{"Kick", "Trip", "Bash"}

// sendTrioOnlyAllowed lists production files permitted to reference one of
// sendTrioOnlyCategories outside a Say(...)/acteeDefenceLine(...) producer
// shape. It is empty: today nothing bypasses SendTrio for Kick, Trip, or
// Bash. A future entry must name the file, why it bypasses, and which PR
// closes it -- the same shape as rawEventsMessageAllowed.
var sendTrioOnlyAllowed = map[string]string{}

// sendTrioOnlyCategoryRE matches a literal reference to one of the guarded
// categories, qualified with its package (production code outside
// internal/messaging always writes messaging.CategoryX).
var sendTrioOnlyCategoryRE = regexp.MustCompile(
	`messaging\.Category(` + strings.Join(sendTrioOnlyCategories, "|") + `)\b`)

// sendTrioOnlyProducerRE matches the two shapes allowed to carry a guarded
// category on the same line: building a messaging.Line via Say, or the
// skill-move defence helper that itself only calls Say.
var sendTrioOnlyProducerRE = regexp.MustCompile(`messaging\.Say\(|acteeDefenceLine\(`)

func TestNarrationTrioOnlyCategoriesLeaveOnlyThroughSendTrio(t *testing.T) {
	seen := map[string]bool{}
	var outside []string

	for _, root := range messagingSurfaceGoRoots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			rel := filepath.ToSlash(path)
			for i, line := range strings.Split(string(src), "\n") {
				if !sendTrioOnlyCategoryRE.MatchString(line) {
					continue
				}
				if sendTrioOnlyProducerRE.MatchString(line) {
					continue
				}
				if _, ok := sendTrioOnlyAllowed[rel]; ok {
					seen[rel] = true
					continue
				}
				outside = append(outside, fmt.Sprintf("%s:%d: %s", rel, i+1, strings.TrimSpace(line)))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}

	// Each allowlist entry must still match, which also proves the pattern
	// can match at all: an empty "outside" list from a blind pattern proves
	// nothing. With no entries today this loop runs zero times; the pattern
	// is instead proven capable of matching by the sabotage step recorded in
	// this guard's commit message.
	for file, why := range sendTrioOnlyAllowed {
		if !seen[file] {
			t.Errorf("allowlist entry %s (%s) no longer references a guarded category; remove it", file, why)
		}
	}
	sort.Strings(outside)
	for _, o := range outside {
		t.Errorf("guarded narration category referenced outside SendTrio's producer shapes: %s\n  send through messaging.Say(...) into a messaging.Trio and messaging.SendTrio, not a raw Send call", o)
	}
}
