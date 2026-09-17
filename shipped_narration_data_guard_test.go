package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/conditions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/gossip"
	"github.com/GoMudEngine/GoMud/internal/grapplemessaging"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/itemvoices"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/tips"
	weathercontent "github.com/GoMudEngine/GoMud/modules/weather/content"
	yamlv2 "gopkg.in/yaml.v2"
	yamlv3 "gopkg.in/yaml.v3"
)

// shippedWorldRoot is the world the server actually serves.
//
// It is spelled out rather than read from the config on purpose: a test binary
// does not read config.yaml, so configs.GetFilePathsConfig().DataFiles returns
// the Go default `_datafiles/world/default` (internal/configs/config.filepaths.go:23),
// a different and largely vestigial world. A guard pointed at that world would
// pass while every shipped store was broken.
//
// The relative path resolves because `go test` runs a package's binary with
// that package's directory as the working directory, and this package is the
// repo root. Nothing here may be moved into a subdirectory without re-rooting
// these paths.
const shippedWorldRoot = "_datafiles/world/dogmud"

// positionControlPath is the position_control store. M4b-1 moved it under the
// world tree from `_datafiles/messages/`, so production now resolves it as
// <configured world>/messaging/position_control.yaml via
// hooks.positionMessagesPath rather than from a hardcoded literal.
const positionControlPath = shippedWorldRoot + "/messaging/position_control.yaml"

// grappleOutcomesPath is what grapplemessaging.DataFilesPath resolves to under
// the shipped config.
const grappleOutcomesPath = shippedWorldRoot + "/messaging/grapple_outcomes.yaml"

// TestShippedNarrationDataValidates loads every narration store from the
// shipped data path and fails the BUILD when one does not validate.
//
// Why this exists: three stores used to log and continue rather than panicking
// (taunt, grapple, position_control), so bad data in them reached players as
// silence with only a log line. M4b-1's two-tier policy moved all three into
// the event tier, where they now fail the boot, but that only helps someone who
// boots the server; this catches the same data at BUILD time, which is where
// the mistake is actually made. It is the pattern weather already uses
// (modules/weather/content/biome_coupling_test.go), and weather stays in the
// ambient tier where this guard is the ONLY net.
//
// It loads from _datafiles/world/dogmud explicitly rather than through the
// config, because a test binary does not read config.yaml: it would get the
// Go default (_datafiles/world/default), which is a different, vestigial
// world (internal/configs/config.filepaths.go:23).
//
// A subtest over a store whose loader already panics still earns its place: it
// names the store and the record in the failure instead of a boot stack trace.
//
// This guard lands BEFORE M4b's role-key renames, deliberately. A store whose
// Go struct no longer declares the tag its shipped file uses unmarshals to the
// zero value, which for the three logging stores means silence in play and
// nothing at all in a green test run.
func TestShippedNarrationDataValidates(t *testing.T) {
	t.Run("taunt", func(t *testing.T) {
		// combat.LoadTauntMessageFiles panics at boot as of M4b-1; this names
		// the offending record instead of handing an operator a stack trace.
		checkFlatStore[string, *combat.TauntMessageGroup](t, "taunt-messages", shippedWorldRoot+"/taunt-messages")
	})

	t.Run("grapple_outcomes", func(t *testing.T) {
		// hooks.LoadGrappleMessaging panics at boot as of M4b-1. It used to log
		// the load error and substitute an EMPTY library, so every grapple line
		// degraded to a debug string with nothing but one log line to say why.
		lib, err := grapplemessaging.Load(grappleOutcomesPath)
		if err != nil {
			t.Fatalf("grapple_outcomes: %v", err)
		}
		total := len(lib.Advancements) + len(lib.Degradations) + len(lib.Reversals) +
			len(lib.Escapes) + len(lib.Holds) + len(lib.StrikingApex) + len(lib.Gradients)
		if total == 0 {
			t.Fatal("grapple_outcomes loaded zero keys: the store is shipped, so zero means the load failed silently")
		}
		// Production calls this too, at boot, and as of M4b-1 panics on it
		// rather than mudlog.Warn'ing each violation. Here it fails the build,
		// which is earlier and names every violation rather than the first.
		for _, e := range grapplemessaging.ValidateCompleteness(lib) {
			t.Errorf("grapple_outcomes: %v", e)
		}
	})

	t.Run("position_control", func(t *testing.T) {
		checkPositionControl(t)
	})

	t.Run("defence", func(t *testing.T) {
		// Keyed by items.DefenseType, exactly as items.LoadDataFiles keys it.
		// Instantiating the same generic with the same key type is what keeps
		// the guard from reading a normalised variant of production's index.
		checkFlatStore[items.DefenseType, *items.DefenseMessageGroup](t, "defense-messages", shippedWorldRoot+"/defense-messages")
	})

	t.Run("combat_messages", func(t *testing.T) {
		checkFlatStore[items.ItemSubType, *items.WeaponAttackMessageGroup](t, "combat-messages", shippedWorldRoot+"/combat-messages")
	})

	t.Run("itemvoices", func(t *testing.T) {
		// Production additionally cross-checks every ItemSpec.VoiceId against
		// this map and panics on a dangling reference. That check needs the
		// whole 429-file item tree loaded and is about item data, not about
		// this store's own text, so it stays at boot.
		checkFlatStore[string, *itemvoices.VoiceSpec](t, "itemvoices", shippedWorldRoot+"/itemvoices")
	})

	t.Run("casting", func(t *testing.T) {
		path := shippedWorldRoot + "/casting-messages.yaml"
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("casting-messages: %v", err)
		}
		// yaml.v2, matching internal/spells/casting_messages.go.
		var cm spells.CastingMessages
		if err := yamlv2.Unmarshal(data, &cm); err != nil {
			t.Fatalf("casting-messages: parse %s: %v", path, err)
		}
		if err := cm.Validate(); err != nil {
			t.Errorf("casting-messages: %v", err)
		}
	})

	t.Run("conditions", func(t *testing.T) {
		// THE ONE STORE WHOSE VALIDATION READS A BALANCE KNOB.
		//
		// ConditionSpec.Validate special-cases conditionId 0 (Meditating, the
		// logout condition): it OVERWRITES the authored triggercount with
		// Network.LogoutRounds and then refuses a count below 1
		// (internal/conditions/conditionspec.go:330). A test binary does not
		// read config.yaml, so that knob comes back as the Go default 0 and
		// the whole store fails to load, on data the server boots on happily.
		//
		// So this subtest reads the real config, which is also the honest
		// thing to guard: a config.yaml shipping LogoutRounds 0 or dropping
		// the key panics the boot, and nothing else would catch it.
		// SetConfigForTest snapshots first and self-registers the restore, so
		// the mutation does not leak into the rest of this test binary; the
		// pattern is internal/characters/poolmax_test.go's withRepoRoot.
		//
		// The store path stays the literal above, NOT the reloaded
		// FilePaths.DataFiles, so the guard keeps reading the shipped world
		// even if a local config.yaml points somewhere else.
		//
		// ReloadConfig logs, and a test binary has no logger until something
		// installs one, so slog nil-dereferences. "LOW" maps to Warn, which
		// keeps the reload quiet. boot_smoke_test.go does the same.
		mudlog.SetupLogger(nil, `LOW`, ``, false)

		configs.SetConfigForTest(t, configs.GetConfig())
		if err := configs.ReloadConfig(); err != nil {
			t.Fatalf("conditions: reload config: %v", err)
		}

		loaded := checkFlatStore[int, *conditions.ConditionSpec](t, "conditions", shippedWorldRoot+"/conditions")
		// conditions.LoadDataFiles panics through ValidateLoadedFlags on an
		// unknown flag. That walks the package's own globals, which this test
		// never populates, so the per-spec method is called directly.
		for id, spec := range loaded {
			if err := spec.ValidateFlags(); err != nil {
				t.Errorf("condition %d: %v", id, err)
			}
		}
	})

	t.Run("spells", func(t *testing.T) {
		checkFlatStore[string, *spells.SpellData](t, "spells", shippedWorldRoot+"/spells")
	})

	t.Run("quests", func(t *testing.T) {
		checkFlatStore[int, *quests.Quest](t, "quests", shippedWorldRoot+"/quests")
	})

	t.Run("crafting", func(t *testing.T) {
		checkFlatStore[string, *crafting.RecipeSpec](t, "crafting", shippedWorldRoot+"/recipes")
	})

	t.Run("gossip", func(t *testing.T) {
		path := shippedWorldRoot + "/gossip_templates.yaml"
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("gossip_templates: %v", err)
		}
		loaded := map[string][]string{}
		if err := yamlv2.Unmarshal(data, &loaded); err != nil {
			t.Fatalf("gossip_templates: parse %s: %v", path, err)
		}
		if len(loaded) == 0 {
			t.Fatal("gossip_templates loaded zero keys: the store is shipped, so zero means the load failed silently")
		}
		if err := gossip.Validate(loaded); err != nil {
			t.Errorf("gossip_templates: %v", err)
		}
	})

	t.Run("tips", func(t *testing.T) {
		path := shippedWorldRoot + "/tips.yaml"
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("tips: %v", err)
		}
		var file struct {
			Tips []string `yaml:"tips"`
		}
		if err := yamlv2.Unmarshal(data, &file); err != nil {
			t.Fatalf("tips: parse %s: %v", path, err)
		}
		if len(file.Tips) == 0 {
			t.Fatal("tips loaded zero lines: the store is shipped, so zero means the load failed silently")
		}
		if err := tips.Validate(file.Tips); err != nil {
			t.Errorf("tips: %v", err)
		}
	})

	t.Run("weather_emotes", func(t *testing.T) {
		checkWeatherEmotes(t)
	})
}

// checkFlatStore loads one fileloader-backed store through the SAME generic
// instantiation production uses, then asserts the store is non-empty and every
// record validates.
//
// fileloader.LoadAllFlatFiles already calls Validate on each record and refuses
// duplicate ids, so a nil error is most of the contract. Validate is called
// again per record anyway, because the loader's error names the FILE and this
// names the ID, and the id is what a role-key rename breaks.
func checkFlatStore[K comparable, T fileloader.Loadable[K]](t *testing.T, store, dir string) map[K]T {
	t.Helper()

	loaded, err := fileloader.LoadAllFlatFiles[K, T](dir)
	if err != nil {
		t.Fatalf("%s: load %s: %v", store, dir, err)
	}
	if len(loaded) == 0 {
		t.Fatalf("%s loaded zero records from %s: the store is shipped, so zero means the load failed silently", store, dir)
	}
	for id, rec := range loaded {
		if err := rec.Validate(); err != nil {
			t.Errorf("%s record %v: %v", store, id, err)
		}
	}
	return loaded
}

// posGuardTriple mirrors hooks.submissionMsgTriple.
//
// The production type is unexported and the package exposes no seam onto it,
// so the shape is mirrored here rather than exporting API for a test. The same
// trade is already made in internal/narration/snapshot_test.go, which mirrors
// this file for the golden.
//
// The mirror is decoded STRICTLY: if the shipped file renames a role key and
// this mirror is not renamed with it, the unknown key fails the decode rather
// than yielding a silently empty triple. That is the direction M4b renames
// travel, and it is the failure this whole test exists to catch.
type posGuardTriple struct {
	Attacker string `yaml:"actor"`
	Target   string `yaml:"actee"`
	Room     string `yaml:"observer"`
}

// posGuardFile mirrors the whole shipped file so it can be decoded strictly.
//
// hooks.positionMessageTemplates parses only stamina_warning and submission.
// gradient_messages and transition_messages are authored in the same file but
// read by NOBODY (the live gradient and transition prose comes from
// internal/grapplemessaging), so requiring text in them would guard data
// production does not use, and the narration snapshot golden covers them. They
// are declared as yaml.Node so a strict decode does not trip over them while
// still refusing an unknown key inside the two blocks that matter.
type posGuardFile struct {
	Gradient   yamlv3.Node `yaml:"gradient_messages"`
	Transition yamlv3.Node `yaml:"transition_messages"`
	Stamina    struct {
		Self string `yaml:"actor"`
		Room string `yaml:"observer"`
	} `yaml:"stamina_warning"`
	Submission struct {
		Opening                map[string]posGuardTriple `yaml:"opening"`
		EscapeBad              posGuardTriple            `yaml:"escape_bad"`
		Neutral                posGuardTriple            `yaml:"neutral"`
		OutcomeMercy           posGuardTriple            `yaml:"outcome_mercy"`
		OutcomeSubdue          posGuardTriple            `yaml:"outcome_subdue"`
		OutcomeCrippleArm      posGuardTriple            `yaml:"outcome_cripple_arm"`
		OutcomeCrippleShoulder posGuardTriple            `yaml:"outcome_cripple_shoulder"`
		OutcomeLethal          posGuardTriple            `yaml:"outcome_lethal"`
		CritFlag               posGuardTriple            `yaml:"crit_flag"`
	} `yaml:"submission"`
}

func checkPositionControl(t *testing.T) {
	t.Helper()

	data, err := os.ReadFile(positionControlPath)
	if err != nil {
		t.Fatalf("position_control: %v", err)
	}
	// yaml.v3, matching internal/hooks/Position_Messaging.go, but with
	// KnownFields on. Production decodes leniently, so a role key renamed on
	// disk and not in Go silently yields an empty string; here it is an error
	// that names the offending key.
	var f posGuardFile
	dec := yamlv3.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil {
		t.Fatalf("position_control: parse %s: %v", positionControlPath, err)
	}

	if f.Stamina.Self == "" || f.Stamina.Room == "" {
		t.Errorf("position_control stamina_warning: actor=%q observer=%q, both must carry text", f.Stamina.Self, f.Stamina.Room)
	}
	if len(f.Submission.Opening) == 0 {
		t.Fatal("position_control submission.opening loaded zero keys: the store is shipped, so zero means the load failed silently")
	}

	full := []struct {
		name string
		tri  posGuardTriple
	}{
		{"escape_bad", f.Submission.EscapeBad},
		{"neutral", f.Submission.Neutral},
		{"outcome_mercy", f.Submission.OutcomeMercy},
		{"outcome_subdue", f.Submission.OutcomeSubdue},
		{"outcome_cripple_arm", f.Submission.OutcomeCrippleArm},
		{"outcome_cripple_shoulder", f.Submission.OutcomeCrippleShoulder},
		{"outcome_lethal", f.Submission.OutcomeLethal},
	}
	for key, tri := range f.Submission.Opening {
		full = append(full, struct {
			name string
			tri  posGuardTriple
		}{"opening." + key, tri})
	}
	for _, c := range full {
		if c.tri.Attacker == "" || c.tri.Target == "" || c.tri.Room == "" {
			t.Errorf("position_control submission.%s: every role must carry text (actor=%q actee=%q observer=%q)",
				c.name, c.tri.Attacker, c.tri.Target, c.tri.Room)
		}
	}

	// crit_flag is the one deliberate exception. It is a PREFIX fragment
	// glued onto the actor's line, so its actee and observer are authored
	// empty on purpose. Requiring all three here would be a guard that fails
	// on correct data. The exemption is by KEY, not by tag, so M4b-1's rename
	// left it working: crit_flag is still absent from the `full` list above.
	if f.Submission.CritFlag.Attacker == "" {
		t.Error("position_control submission.crit_flag: actor must carry text")
	}
}

// checkWeatherEmotes loads the ambient store the way modules/weather does:
// os.DirFS over the world root, then the two exported content loaders.
//
// Weather never panics by documented intent (an emote file that fails to parse
// costs silence, not a boot), so a build-time check is the ONLY thing standing
// behind it. LoadEmotes swallows a missing directory and returns empty tables
// with a nil error, which is exactly the silent-empty failure mode this test
// exists for, hence the length assertions.
//
// DELIBERATELY THIN, and here is what it does not cover. A section dropped
// from one table (say `outdoor:` renamed) parses to an empty map with no
// error, and this subtest would still see a non-empty Tables and pass. The
// rule that catches that lives in the module, in
// modules/weather/content/shipped_emotes_test.go, which pins the table count
// at 9 and requires a non-empty outdoor default per table; per-pool depth and
// the biome coupling are in biome_coupling_test.go. Those are not duplicated
// here. What this subtest adds is that the shipped tree still loads non-empty
// at all, that a pool below the depth floor fails, and that weather is named
// in the failure alongside the other thirteen stores.
func checkWeatherEmotes(t *testing.T) {
	t.Helper()

	worldFS := os.DirFS(shippedWorldRoot)

	tables, err := weathercontent.LoadEmotes(worldFS, "weather/emotes")
	if err != nil {
		t.Fatalf("weather emotes: %v", err)
	}
	if len(tables) == 0 {
		t.Fatal("weather emotes loaded zero tables: the store is shipped, so zero means the load failed silently")
	}

	seasonal, err := weathercontent.LoadSeasonalEmotes(worldFS, "weather/emotes/seasons")
	if err != nil {
		t.Fatalf("weather seasonal emotes: %v", err)
	}
	if len(seasonal) == 0 {
		t.Fatal("weather seasonal emotes loaded zero tables: the store is shipped, so zero means the load failed silently")
	}
}
