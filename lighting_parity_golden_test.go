package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

var updateLightingParity = flag.Bool("update-lighting-parity", false,
	"re-record testdata/lighting_parity.golden")

// TestLightingParityAcrossEveryShippedRoom is the behaviour-preservation
// proof for graded lighting plan 1. It records, for every shipped room, what
// each kind of observer can see and whether exits are visible.
//
// It is recorded against the UNMODIFIED tree and must come back byte
// identical after the scale swap. A golden recorded after the change would
// prove nothing, which is why Task 1 comes first.
//
// The four observer kinds are the ones ParticipantSight actually branches on
// (internal/messaging/predicates.go:51-68): an ordinary character, one
// carrying NightVision (condition 29), one carrying InfraredVision
// (condition 85), and one that is Blinded (condition 3).
//
// The blinded fixture is condition 3, not a hand-set flag: ParticipantSight
// tests observer.Perception.State() == perception.Blinded, the Perception
// state machine, not a condition flag directly.
// Character.AddCondition (internal/characters/conditions.go:116-129) drives
// that transition itself when conditionId == perception.ConditionIdBlinded
// (3) and the machine is currently Sighted, so AddCondition(3, true) is the
// real path. Verified live: TestParticipantSight's table
// (internal/messaging/participant_sight_test.go:45) exercises the Blinded
// branch, and a throwaway probe built during this task (built the same way
// this fixture is built, then run against a definitely-lit room) asserted
// SightNone, then was sabotaged (swapped condition 3 for condition 29) and
// confirmed the assertion goes red naming the right condition before being
// restored. See the task report for the full transcript; the probe file
// itself was not committed.
func TestLightingParityAcrossEveryShippedRoom(t *testing.T) {
	// "LOW" suppresses the very noisy per-file loader logging; without any
	// logger set up at all, the loaders panic on a nil *slog.Logger.
	mudlog.SetupLogger(nil, `LOW`, ``, false)

	// Read the REAL shipped config rather than fabricating overrides.
	// SetConfigForTest snapshots the pre-reload config first and self-
	// restores it via t.Cleanup, so ReloadConfig's mutation of the package
	// global does not leak into the rest of this test binary process.
	// Pattern lifted from shipped_narration_data_guard_test.go's
	// "conditions" subtest, which reads the same knob for the same reason:
	// ConditionSpec.Validate overwrites conditionId 0's (Meditating)
	// TriggerCount with Network.LogoutRounds and refuses a count below 1
	// (internal/conditions/conditionspec.go:338), and a bare test binary
	// never reads config.yaml, so that knob would otherwise come back as
	// the Go default 0 and conditions.LoadDataFiles would panic.
	configs.SetConfigForTest(t, configs.GetConfig())
	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}

	// Order matches main.go's own boot sequence (main.go:1631, 1636, 1643,
	// 1862): biomes, then rooms, then conditions, then mutators last.
	// Mutators load after rooms in main.go too, because Room.GetVisibility()
	// -> ActiveMutators() is only ever called once content is fully up, and
	// GetVisibility() itself dereferences the mutator registry
	// unconditionally (internal/rooms/rooms.go:169-170: `spec :=
	// mut.GetSpec(); if spec.LightMod != 0` has no nil guard on spec). Load
	// it before the walk below calls GetVisibility for the first time.
	rooms.LoadBiomeDataFiles()
	rooms.LoadDataFiles()
	conditions.LoadDataFiles()
	mutators.LoadDataFiles()

	ids := rooms.GetAllRoomIds()
	if len(ids) < 1000 {
		t.Fatalf("loaded only %d rooms: the walk is not seeing the world, so a green run proves nothing", len(ids))
	}
	sort.Ints(ids)

	observers := []struct {
		name        string
		conditionId int
	}{
		{"plain", 0},
		{"nightvision", 29},
		{"infrared", 85},
		{"blinded", 3},
	}

	var b strings.Builder
	skipped := 0
	for _, id := range ids {
		room := rooms.LoadRoom(id)
		if room == nil {
			// rooms.LoadRoom can return nil (failed LoadRoomInstance, or a
			// lost addRoomToMemory race — internal/rooms/save_and_load.go:
			// 79-121, the whole function). Every id the enumerator handed
			// back should load; a silent `continue` here would let a mass
			// load failure shrink the golden with no signal, since the
			// recorded byte count would still clear the >= 1000 sanity
			// floor above. Failed below instead of skipped, on both the
			// record and verify path since this loop runs before that
			// branch.
			skipped++
			continue
		}
		bi := room.GetBiome()
		biomeName := ``
		if bi != nil {
			biomeName = bi.BiomeId
		}
		fmt.Fprintf(&b, "room %d biome=%s\n", id, biomeName)
		for _, o := range observers {
			ch := characters.New()
			ch.Name = o.name
			if o.conditionId > 0 {
				if err := ch.AddCondition(o.conditionId, true); err != nil {
					t.Fatalf("AddCondition(%d): %v", o.conditionId, err)
				}
			}
			fmt.Fprintf(&b, "  %-12s sight=%-6s clear=%v shapes=%v\n",
				o.name,
				sightName(messaging.ParticipantSight(ch, room)),
				messaging.CanSeeClearly(ch, room),
				messaging.CanSeeShapes(ch, room))
		}
	}
	if skipped > 0 {
		t.Fatalf("%d of %d room ids failed to load (rooms.LoadRoom returned nil): the golden would silently lose coverage for those rooms", skipped, len(ids))
	}

	goldenPath := filepath.Join("testdata", "lighting_parity.golden")
	if *updateLightingParity {
		if err := os.WriteFile(goldenPath, []byte(b.String()), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("re-recorded %s (%d bytes, %d rooms)", goldenPath, b.Len(), len(ids))
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v (record it with -update-lighting-parity)", err)
	}
	if b.String() != string(want) {
		pos, ctxWant, ctxGot := firstDiffContext(string(want), b.String())
		t.Fatalf(
			"lighting parity changed (first difference at byte %d). This plan is behaviour preserving, so a diff here is a DEFECT, not something to re-record.\n\n"+
				"want context:\n%s\n\ngot context:\n%s\n\n"+
				"If this change is genuinely intended, re-record with:\n"+
				"  go test . -run TestLightingParityAcrossEveryShippedRoom -update-lighting-parity -v",
			pos, ctxWant, ctxGot,
		)
	}
}

// sightName renders a messaging.SightDecision as a stable, human-readable
// name for the golden. SightDecision has no String() method today; this is
// a LOCAL rendering, not a substitute for one, and deliberately stays that
// way. If SightDecision ever grows a String() method for an unrelated
// reason, this keeps the golden's own vocabulary fixed instead of inheriting
// whatever that method happens to print, so the golden only churns when
// lighting behaviour actually changes.
func sightName(d messaging.SightDecision) string {
	switch d {
	case messaging.SightFull:
		return "full"
	case messaging.SightShapes:
		return "shapes"
	case messaging.SightNone:
		return "none"
	default:
		return fmt.Sprintf("unknown(%d)", int(d))
	}
}
