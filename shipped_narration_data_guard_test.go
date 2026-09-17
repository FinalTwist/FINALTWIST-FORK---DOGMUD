package main

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

// narrationStoreWalkRoots is exactly what TestNoLegacyRoleKeysInShippedData
// walks: thirteen paths covering the fourteen shipped narration stores
// (messaging/ holds two of them). It is a list of stores rather than a walk of
// the world root, and both exclusions that buys are load-bearing.
//
// _datafiles/world/default is OUT OF BOUNDS. The owner ruled on 2026-09-17
// that the vestigial default world is left alone, so it still carries the old
// spellings on purpose: 39 conditions, 2 quests and 8 combat-messages files
// were deliberately not rewritten by M4b-1. That world cannot boot anyway (it
// has no defense-messages directory, so the item loader panics first) and no
// test loads a narration store from it, because every loader call in a test
// points FilePaths.DataFiles at the dogmud world or a temp dir first. A walk
// that reached it would fail on data nobody serves.
//
// Everything under the dogmud world that is not a narration store is out of
// scope too. Other stores own some of these spellings legitimately:
// internal/behaviortree reads `room_text` and `user_text` action params out of
// behaviors/ (internal/behaviortree/actions_dialogue.go:40), and
// internal/items reads `on_use_room_text`. Those are different stores in a
// different arc. Naming the narration stores keeps them out by construction,
// which cannot rot the way a path exemption can.
var narrationStoreWalkRoots = []string{
	shippedWorldRoot + "/taunt-messages",
	shippedWorldRoot + "/combat-messages",
	shippedWorldRoot + "/defense-messages",
	shippedWorldRoot + "/itemvoices",
	shippedWorldRoot + "/conditions",
	shippedWorldRoot + "/spells",
	shippedWorldRoot + "/quests",
	shippedWorldRoot + "/recipes",
	shippedWorldRoot + "/messaging",
	shippedWorldRoot + "/weather/emotes",
	shippedWorldRoot + "/casting-messages.yaml",
	shippedWorldRoot + "/gossip_templates.yaml",
	shippedWorldRoot + "/tips.yaml",
}

// legacyRoleKeysAnyStore are retired spellings that no narration store may use
// anywhere, mapped to what replaced them.
//
// The table mirrors tools/messaging_token_rewrite.py's KEY_GROUPS, which is
// what actually performed the renames, so the guard bans exactly the
// vocabulary the slice retired and nothing it invented. `controlled` is here
// with one documented exemption; see legacyKeyIsExempt.
var legacyRoleKeysAnyStore = map[string]string{
	// combat, defence, taunt (commit e2e6795e4)
	"toattacker":     "actor",
	"todefender":     "actee",
	"toroom":         "observer",
	"toattackerroom": "observer",
	"todefenderroom": "remote_observer",

	// grapple and position_control sides (620188c7f, 93fbc3ccc)
	"controller": "actor",
	"controlled": "actee",
	"observers":  "observer",
	"partner":    "actee",

	// conditions (acd556e82)
	"start_user_text":   "start_actee",
	"start_room_text":   "start_observer",
	"trigger_user_text": "trigger_actee",
	"trigger_room_text": "trigger_observer",
	"end_user_text":     "end_actee",
	"end_room_text":     "end_observer",

	// spells (acd556e82)
	"cast_user_text":  "cast_actor",
	"cast_room_text":  "cast_observer",
	"wait_user_text":  "wait_actor",
	"wait_room_text":  "wait_observer",
	"magic_user_text": "magic_actor",
	"magic_room_text": "magic_observer",

	// quests (acd556e82)
	"playermessage": "actor",
	"roommessage":   "observer",
	"send_text":     "actor",
	// `room_text` IS banned here, and the plan's warning that it might not be
	// safe to ban was checked rather than assumed. It was a real quest trigger
	// action key and acd556e82 renamed it to `observer`; no shipped quest in
	// either world authors it any more (grep over dogmud/quests and
	// default/quests: zero hits). Its one surviving reader,
	// internal/behaviortree, reads it out of behaviors/, which this walk does
	// not cover. So within these roots the spelling is retired, full stop.
	"room_text": "observer",

	// crafting recipes (acd556e82)
	"success_message":      "success_actor",
	"success_room_message": "success_observer",
	"failure_message":      "failure_actor",
	"failure_room_message": "failure_observer",
}

// legacyRoleKeysMessagingOnly are spellings that are retired inside
// messaging/ but are ordinary, live keys elsewhere, so banning them worldwide
// would be a guard that fails on correct data.
//
// `room` is the proof: internal/quests/triggers.go:13 declares TriggerDef.Room
// with the yaml tag "room" as a trigger's room filter, and 102 shipped quest
// lines author it. A blanket ban would redden every one of them.
// `attacker`, `target` and `self` are the same shape of word: they were
// position_control and grapple_outcomes role keys (the `position` and
// `grapple` groups in the rewrite tool) and nothing else in these stores uses
// them, but they are plausible future keys for a store that never had the old
// vocabulary, so the ban stays where the rename happened.
var legacyRoleKeysMessagingOnly = map[string]string{
	"self":     "actor",
	"attacker": "actor",
	"target":   "actee",
	"room":     "observer",
}

// legacyKeyIsExempt carves out the one place a banned spelling is correct
// authored data.
//
// `controlled` is BOTH a retired role key and a live gradient STATE name. In
// position_control.yaml the sides used to be spelled controller/controlled and
// are now actor/actee, but the gradient states are in_control,
// losing_control, neutral, becoming_controlled and controlled, and the state
// keeps its spelling because renaming it would turn
// gradient_messages.actor.actee into nonsense. So after Task 7,
// `gradient_messages.actor.controlled.actor` is legal and correct, and a guard
// that banned the word outright would fail on shipped data.
//
// The discriminator is the same one the rewrite tool used: nesting depth. The
// sides are direct children of gradient_messages and the states are one level
// below them. This scopes by ancestor PATH rather than by column, which is the
// stricter form of the same rule: `controlled` is exempt only as a grandchild
// of gradient_messages in that one file. A `controlled:` reintroduced as a
// side, as an audience key inside a state, or anywhere in any other file, is
// still caught.
func legacyKeyIsExempt(path, key string, ancestors []string) bool {
	return key == "controlled" &&
		filepath.ToSlash(path) == positionControlPath &&
		len(ancestors) == 2 &&
		ancestors[0] == "gradient_messages"
}

// TestNoLegacyRoleKeysInShippedData fails the build when a narration YAML file
// still spells a role the old way. The renames of M4b-1 are only durable if a
// newly authored file cannot reintroduce the old vocabulary, and a store whose
// struct no longer declares the tag would load that file SILENTLY EMPTY, which
// is the exact failure this slice exists to make impossible.
//
// It walks narrationStoreWalkRoots, which is the shipped dogmud world's
// narration stores only; the scoping and the reasons for it are documented on
// that variable, on the two ban tables, and on legacyKeyIsExempt.
//
// It inspects MAPPING KEYS from a parsed yaml.v3 node tree, not lines of text.
// Half these spellings are ordinary English words, so a line-oriented scan
// would have to guess whether `room:` inside a block scalar is a key or prose.
// The node tree does not guess, and it hands over a real ancestor path, which
// is what the gradient-state exemption is keyed on.
//
// It logs how many files and how many keys it inspected, and refuses to pass
// on a walk that found nothing. An absence guard whose walk silently scans
// zero files passes in 0.00s and proves nothing; this repo has been bitten by
// exactly that.
func TestNoLegacyRoleKeysInShippedData(t *testing.T) {
	filesInspected := 0
	keysInspected := 0

	for _, root := range narrationStoreWalkRoots {
		files := yamlFilesUnder(t, root)
		if len(files) == 0 {
			t.Errorf("walk root %s yielded zero YAML files: the guard would scan nothing there", root)
			continue
		}
		for _, path := range files {
			filesInspected++
			keysInspected += checkFileForLegacyRoleKeys(t, path)
		}
	}

	if keysInspected == 0 {
		t.Fatal("inspected zero mapping keys: the walk found nothing, so a green run proves nothing")
	}
	// A floor, not a pin. Content volume moves; a walk collapsing to a handful
	// of files does not happen for a legitimate reason.
	if filesInspected < 100 {
		t.Errorf("inspected only %d files across %d walk roots: expected the whole narration tree", filesInspected, len(narrationStoreWalkRoots))
	}
	t.Logf("inspected %d YAML files and %d mapping keys across %d narration store roots",
		filesInspected, keysInspected, len(narrationStoreWalkRoots))
}

// yamlFilesUnder returns every .yaml/.yml file at or under root. root may name
// a single file, which three of the stores are.
func yamlFilesUnder(t *testing.T, root string) []string {
	t.Helper()

	info, err := os.Stat(root)
	if err != nil {
		t.Errorf("walk root %s: %v", root, err)
		return nil
	}
	if !info.IsDir() {
		return []string{root}
	}

	var out []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if ext := filepath.Ext(path); ext == ".yaml" || ext == ".yml" {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		t.Errorf("walk root %s: %v", root, err)
	}
	return out
}

// checkFileForLegacyRoleKeys reports every banned mapping key in one file and
// returns how many keys it looked at, so the caller can prove the walk is not
// silently inspecting nothing.
func checkFileForLegacyRoleKeys(t *testing.T, path string) int {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("%s: %v", path, err)
		return 0
	}

	// Multi-document files are not used by these stores today, but decoding in
	// a loop costs nothing and means a second document could not hide a key.
	keys := 0
	dec := yamlv3.NewDecoder(bytes.NewReader(data))
	for {
		var doc yamlv3.Node
		if err := dec.Decode(&doc); err != nil {
			if err == io.EOF {
				break
			}
			// A parse failure is TestShippedNarrationDataValidates' business,
			// but reporting it here too beats scanning zero keys quietly.
			t.Errorf("%s: parse: %v", path, err)
			return keys
		}
		keys += walkNodeForLegacyRoleKeys(t, path, &doc, nil)
	}
	return keys
}

func walkNodeForLegacyRoleKeys(t *testing.T, path string, n *yamlv3.Node, ancestors []string) int {
	t.Helper()

	keys := 0
	switch n.Kind {
	case yamlv3.DocumentNode, yamlv3.SequenceNode:
		for _, child := range n.Content {
			keys += walkNodeForLegacyRoleKeys(t, path, child, ancestors)
		}
	case yamlv3.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			keys++
			reportLegacyRoleKey(t, path, k, ancestors)
			keys += walkNodeForLegacyRoleKeys(t, path, v, append(ancestors, k.Value))
		}
	}
	return keys
}

func reportLegacyRoleKey(t *testing.T, path string, key *yamlv3.Node, ancestors []string) {
	t.Helper()

	replacement, banned := legacyRoleKeysAnyStore[key.Value]
	if !banned && strings.HasPrefix(filepath.ToSlash(path), shippedWorldRoot+"/messaging/") {
		replacement, banned = legacyRoleKeysMessagingOnly[key.Value]
	}
	if !banned || legacyKeyIsExempt(path, key.Value, ancestors) {
		return
	}

	where := "(top level)"
	if len(ancestors) > 0 {
		where = strings.Join(ancestors, ".")
	}
	t.Errorf("%s:%d: legacy role key %q under %s: M4b-1 renamed it to %q, and the store's Go struct no longer declares the old tag, so this file would load SILENTLY EMPTY",
		filepath.ToSlash(path), key.Line, key.Value, where, replacement)
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
