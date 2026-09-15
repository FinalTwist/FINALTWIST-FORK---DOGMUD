package migration

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/conditionrename"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// Description:
// Conditions unification slice 3 renamed every buff-spelled save key to its
// condition spelling. Every loader ignores unknown keys, so an unmigrated
// save would load with its conditions, pet condition ids and trapped locks
// silently empty.
//
// Path-anchored: only the listed key paths are renamed, and the new name comes
// from conditionrename.Apply, the one spelling map.
//
// Idempotent without a marker: a file with no old key is not rewritten, so a
// second run changes nothing. That also avoids the alts trap, where a
// character-scoped marker re-runs per alt.
//
// Unlike 0.14.0, alts files (<id>.alts.yaml, a YAML LIST of characters) are
// migrated, and an unparseable file or a collision (old and new key both
// present) is an error, so Run restores the backup rather than leaving a save
// that would lose its conditions on load.
func migrate_ConditionKeys(dryRun bool) error {
	return migrateConditionKeysIn(string(configs.GetConfig().FilePaths.DataFiles), dryRun)
}

// keyRename renames key `old` inside every mapping reached by `parent`.
// Path segments: a key name, "*" for every value of a mapping, "[]" for every
// element of a list.
type keyRename struct {
	parent []string
	old    string
}

// characterRenames are relative to one character mapping. Order matters: the
// later paths use the already-renamed `conditions`.
var characterRenames = []keyRename{
	{nil, "buffs"},
	{[]string{"conditions", "list", "[]"}, "buffid"},
	{[]string{"conditions", "list", "[]"}, "permabuff"},
	{[]string{"pet"}, "buffids"},
	{[]string{"shop", "[]"}, "buffid"},
	{[]string{"miscdata"}, "pinnacle_bandolier_buffs"},
}

var roomInstanceRenames = []keyRename{
	{[]string{"containers", "*", "lock"}, "trapbuffids"},
}

// applyRenames walks node along rename.parent and renames rename.old in each
// mapping it reaches. It reports whether anything changed.
func applyRenames(node any, renames []keyRename) (bool, error) {
	changed := false
	for _, r := range renames {
		c, err := renameAt(node, r.parent, r.old, conditionrename.Apply(r.old))
		if err != nil {
			return changed, err
		}
		changed = changed || c
	}
	return changed, nil
}

func renameAt(node any, path []string, oldKey, newKey string) (bool, error) {
	if len(path) == 0 {
		m, ok := node.(yaml.MapSlice)
		if !ok {
			return false, nil
		}
		oldIdx, hasNew := -1, false
		for i, item := range m {
			switch k, _ := item.Key.(string); k {
			case oldKey:
				oldIdx = i
			case newKey:
				hasNew = true
			}
		}
		if oldIdx < 0 {
			return false, nil
		}
		if hasNew {
			return false, fmt.Errorf("both %q and %q present", oldKey, newKey)
		}
		// MapSlice shares its backing array with the parent, so this
		// renames the key in the decoded document in place.
		m[oldIdx].Key = newKey
		return true, nil
	}
	seg, rest := path[0], path[1:]
	changed := false
	switch seg {
	case "[]":
		list, ok := node.([]any)
		if !ok {
			return false, nil
		}
		for _, el := range list {
			c, err := renameAt(el, rest, oldKey, newKey)
			if err != nil {
				return changed, err
			}
			changed = changed || c
		}
	case "*":
		m, ok := node.(yaml.MapSlice)
		if !ok {
			return false, nil
		}
		for _, item := range m {
			c, err := renameAt(item.Value, rest, oldKey, newKey)
			if err != nil {
				return changed, err
			}
			changed = changed || c
		}
	default:
		m, ok := node.(yaml.MapSlice)
		if !ok {
			return false, nil
		}
		for _, item := range m {
			if k, _ := item.Key.(string); k == seg {
				return renameAt(item.Value, rest, oldKey, newKey)
			}
		}
	}
	return changed, nil
}

// migrateConditionKeysIn is the testable core: dataDir is a DataFiles root.
func migrateConditionKeysIn(dataDir string, dryRun bool) error {
	mode := "APPLY"
	if dryRun {
		mode = "DRY-RUN"
	}
	mudlog.Info("Migration 0.17.0", "message", "Renaming buff save keys to condition keys", "mode", mode)

	counts := map[string]int{}

	usersDir := filepath.Join(dataDir, "users")
	userFiles, err := filepath.Glob(filepath.Join(usersDir, "*.yaml"))
	if err != nil {
		return err
	}
	for _, path := range userFiles {
		if strings.HasSuffix(path, ".alts.yaml") {
			if err := migrateFile(path, dryRun, migrateAltsDoc); err != nil {
				return err
			}
			counts["alts"]++
			continue
		}
		if err := migrateFile(path, dryRun, migrateUserDoc); err != nil {
			return err
		}
		counts["users"]++
	}

	roomsDir := filepath.Join(dataDir, "rooms.instances")
	err = filepath.WalkDir(roomsDir, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			if errors.Is(werr, fs.ErrNotExist) {
				return nil
			}
			return werr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		counts["rooms.instances"]++
		return migrateFile(path, dryRun, migrateRoomDoc)
	})
	if err != nil {
		return err
	}

	mudlog.Info("Migration 0.17.0", "users", counts["users"], "alts", counts["alts"], "rooms.instances", counts["rooms.instances"], "mode", mode)
	return nil
}

type docMigrator func(raw []byte) (out any, changed bool, err error)

func migrateFile(path string, dryRun bool, migrate docMigrator) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("migration 0.17.0: read %s: %w", path, err)
	}
	doc, changed, err := migrate(raw)
	if err != nil {
		return fmt.Errorf("migration 0.17.0: %s: %w", path, err)
	}
	if !changed {
		return nil
	}
	mudlog.Info("Migration 0.17.0", "file", path, "renamed", true)
	if dryRun {
		return nil
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("migration 0.17.0: marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("migration 0.17.0: write %s: %w", path, err)
	}
	return nil
}

func migrateUserDoc(raw []byte) (any, bool, error) {
	var doc yaml.MapSlice
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, false, err
	}
	for _, item := range doc {
		if k, _ := item.Key.(string); k == "character" {
			changed, err := applyRenames(item.Value, characterRenames)
			return doc, changed, err
		}
	}
	return doc, false, nil
}

func migrateAltsDoc(raw []byte) (any, bool, error) {
	var doc []yaml.MapSlice
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, false, err
	}
	changed := false
	for _, character := range doc {
		c, err := applyRenames(character, characterRenames)
		if err != nil {
			return nil, false, err
		}
		changed = changed || c
	}
	return doc, changed, nil
}

func migrateRoomDoc(raw []byte) (any, bool, error) {
	var doc yaml.MapSlice
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, false, err
	}
	changed, err := applyRenames(doc, roomInstanceRenames)
	return doc, changed, err
}
