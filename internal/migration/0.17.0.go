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
// save would load with its conditions, pet condition ids, trapped locks and
// item condition ids silently empty.
//
// Whole-DataFiles, not path lists. An item saves its full spec copy under
// `overrides:` once it is enchanted, affixed or renamed, and GetSpec() then
// never reads the template again, so a missed `wornbuffids` empties that list
// permanently. Items are saved in characters, bank storage, room instances,
// mob instances, shops, guild vaults, sealed crates and auction plugin data,
// and a path list missed most of them. So every .yaml and .plugin.dat file
// under DataFiles is scanned, and the old keys are renamed wherever they sit.
//
// Cheap and safe to scan: a file with no old spelling is skipped without
// parsing (conditionrename.ContainsOldSpelling: "buff" in any case, outside
// protected words such as "buffer", none of which is part of an old key), so
// content files, JSON plugin data and unrelated corrupt files are never
// parsed or rewritten. Only KEYS are renamed, never values,
// and the new name comes from conditionrename.Apply, the one spelling map.
//
// Idempotent without a marker: a file with no old key is not rewritten, so a
// second run changes nothing. That also avoids the alts trap, where a
// character-scoped marker re-runs per alt.
//
// A file with an old spelling that fails to parse, or a mapping with both an
// old key and its new name, is an error, so Run restores the backup rather
// than leaving a save that would lose its conditions on load.
//
// The config overrides file is also migrated when CONFIG_PATH points outside
// DataFiles, and it is reloaded after a rewrite: config was loaded before
// migrations run, and Run's closing SetVal writes the in-memory overrides map
// back to disk, which would restore the old key.
func migrate_ConditionKeys(dryRun bool) error {
	dataDir := string(configs.GetConfig().FilePaths.DataFiles)
	// Mirrors configs.overridePath (unexported).
	overridesPath := os.Getenv(`CONFIG_PATH`)
	if overridesPath == `` {
		overridesPath = filepath.Join(dataDir, `config-overrides.yaml`)
	}
	return migrateConditionKeys(dataDir, overridesPath, dryRun, configs.ReloadConfig)
}

// renamedKeys is the complete set of distinctive buff-spelled YAML keys a
// save, plugin data or config overrides file can carry (from the slice 3 yaml
// tag inventory). Each is renamed to conditionrename.Apply(key) wherever it
// appears. The generic key `buffs` is handled separately by isConditionsRecord.
var renamedKeys = map[string]bool{
	"buffid":                   true,
	"buffids":                  true,
	"wornbuffids":              true,
	"critbuffids":              true,
	"trapbuffids":              true,
	"prizebuffids":             true,
	"playerbuffids":            true,
	"mobbuffids":               true,
	"nativebuffids":            true,
	"start_remove_buffs":       true,
	"buff_ids":                 true,
	"buff_id":                  true,
	"permabuff":                true,
	"pinnacle_bandolier_buffs": true,
	"BuffsEnabled":             true,
}

// conditionKeysStats counts one migration pass.
type conditionKeysStats struct {
	scanned, parsed int
	rewritten       []string
}

// migrateConditionKeysIn is the testable core: dataDir is a DataFiles root.
func migrateConditionKeysIn(dataDir string, dryRun bool) error {
	_, err := renameConditionKeysUnder(dataDir, dryRun)
	return err
}

// migrateConditionKeys runs the DataFiles walk, then the overrides file when
// it lives outside DataFiles, and calls reload when the overrides file was
// rewritten.
func migrateConditionKeys(dataDir, overridesPath string, dryRun bool, reload func() error) error {
	stats, err := renameConditionKeysUnder(dataDir, dryRun)
	if err != nil {
		return err
	}
	overridesRewritten := false
	for _, p := range stats.rewritten {
		if samePath(p, overridesPath) {
			overridesRewritten = true
		}
	}
	if !isUnder(overridesPath, dataDir) {
		changed, err := renameConditionKeysInFile(overridesPath, dryRun)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		overridesRewritten = overridesRewritten || changed
	}
	if overridesRewritten && !dryRun {
		return reload()
	}
	return nil
}

func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	return errA == nil && errB == nil && strings.EqualFold(filepath.Clean(absA), filepath.Clean(absB))
}

func isUnder(path, dir string) bool {
	absP, errP := filepath.Abs(path)
	absD, errD := filepath.Abs(dir)
	if errP != nil || errD != nil {
		return false
	}
	rel, err := filepath.Rel(absD, absP)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func renameConditionKeysUnder(dataDir string, dryRun bool) (conditionKeysStats, error) {
	mode := "APPLY"
	if dryRun {
		mode = "DRY-RUN"
	}
	mudlog.Info("Migration 0.17.0", "message", "Renaming buff keys to condition keys under DataFiles", "mode", mode)

	var stats conditionKeysStats
	err := filepath.WalkDir(dataDir, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			if errors.Is(werr, fs.ErrNotExist) {
				return nil
			}
			return werr
		}
		if !d.Type().IsRegular() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".plugin.dat") {
			return nil
		}
		stats.scanned++
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("migration 0.17.0: read %s: %w", path, err)
		}
		if !conditionrename.ContainsOldSpelling(string(raw)) {
			return nil
		}
		stats.parsed++
		changed, err := renameConditionKeysInBytes(path, raw, dryRun)
		if err != nil {
			return err
		}
		if changed {
			stats.rewritten = append(stats.rewritten, path)
		}
		return nil
	})
	if err != nil {
		return stats, err
	}

	mudlog.Info("Migration 0.17.0", "scanned", stats.scanned, "parsed", stats.parsed, "rewritten", len(stats.rewritten), "mode", mode)
	return stats, nil
}

// renameConditionKeysInFile migrates one file outside the walk.
func renameConditionKeysInFile(path string, dryRun bool) (bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if !conditionrename.ContainsOldSpelling(string(raw)) {
		return false, nil
	}
	return renameConditionKeysInBytes(path, raw, dryRun)
}

// renameConditionKeysInBytes parses raw, renames old keys, and writes path
// when something changed and this is not a dry run.
func renameConditionKeysInBytes(path string, raw []byte, dryRun bool) (bool, error) {
	doc, err := decodeOrdered(raw)
	if err != nil {
		return false, fmt.Errorf("migration 0.17.0: %s: %w", path, err)
	}
	if doc == nil {
		return false, nil
	}
	changed, err := renameKeys(doc)
	if err != nil {
		return false, fmt.Errorf("migration 0.17.0: %s: %w", path, err)
	}
	if !changed {
		return false, nil
	}
	if _, mixed := doc.([]interface{}); mixed {
		// Its nested mappings decoded unordered; rewriting would reorder a
		// file of a shape no store saves. Refuse rather than guess.
		return false, fmt.Errorf("migration 0.17.0: %s: a list with non-mapping elements carries an old key", path)
	}
	mudlog.Info("Migration 0.17.0", "file", path, "renamed", true)
	if dryRun {
		return true, nil
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return false, fmt.Errorf("migration 0.17.0: marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return false, fmt.Errorf("migration 0.17.0: write %s: %w", path, err)
	}
	return true, nil
}

// decodeOrdered decodes a YAML document keeping key order. The root kind is
// read first because yaml.v2 will happily decode a sequence of mappings into
// a MapSlice (it is a []MapItem) and silently lose the data. A mapping root
// becomes yaml.MapSlice; a sequence root becomes []yaml.MapSlice when every
// element is a mapping (the alts shape), else []any, which is walked only to
// detect an old key (an error). A null, empty or scalar root returns nil.
func decodeOrdered(raw []byte) (any, error) {
	var probe any
	if err := yaml.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	switch p := probe.(type) {
	case map[interface{}]interface{}:
		var doc yaml.MapSlice
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			return nil, err
		}
		return doc, nil
	case []interface{}:
		allMaps := true
		for _, el := range p {
			if _, ok := el.(map[interface{}]interface{}); !ok {
				allMaps = false
				break
			}
		}
		if allMaps {
			var doc []yaml.MapSlice
			if err := yaml.Unmarshal(raw, &doc); err != nil {
				return nil, err
			}
			return doc, nil
		}
		return p, nil
	default:
		return nil, nil
	}
}

// renameKeys renames old keys in every mapping at any depth, in place.
func renameKeys(node any) (bool, error) {
	changed := false
	switch n := node.(type) {
	case yaml.MapSlice:
		for i := range n {
			c, err := renameKeys(n[i].Value)
			if err != nil {
				return changed, err
			}
			changed = changed || c
		}
		for i := range n {
			oldKey, ok := n[i].Key.(string)
			if !ok || !shouldRename(oldKey, n[i].Value) {
				continue
			}
			newKey := conditionrename.Apply(oldKey)
			for _, other := range n {
				if k, _ := other.Key.(string); k == newKey {
					return changed, fmt.Errorf("both %q and %q present", oldKey, newKey)
				}
			}
			n[i].Key = newKey
			changed = true
		}
	case []yaml.MapSlice:
		for _, el := range n {
			c, err := renameKeys(el)
			if err != nil {
				return changed, err
			}
			changed = changed || c
		}
	case []interface{}:
		for _, el := range n {
			c, err := renameKeys(el)
			if err != nil {
				return changed, err
			}
			changed = changed || c
		}
	case map[interface{}]interface{}:
		for k, v := range n {
			c, err := renameKeys(v)
			if err != nil {
				return changed, err
			}
			changed = changed || c
			oldKey, ok := k.(string)
			if !ok || !shouldRename(oldKey, v) {
				continue
			}
			newKey := conditionrename.Apply(oldKey)
			if _, exists := n[newKey]; exists {
				return changed, fmt.Errorf("both %q and %q present", oldKey, newKey)
			}
			delete(n, oldKey)
			n[newKey] = v
			changed = true
		}
	}
	return changed, nil
}

// shouldRename reports whether key, holding value, is an old buff spelling.
func shouldRename(key string, value any) bool {
	if renamedKeys[key] {
		return true
	}
	return key == "buffs" && isConditionsRecord(value)
}

// isConditionsRecord reports whether value has the conditions record shape,
// a mapping with a `list` key. Only then is a `buffs` key the old record.
func isConditionsRecord(value any) bool {
	switch v := value.(type) {
	case yaml.MapSlice:
		for _, item := range v {
			if k, _ := item.Key.(string); k == "list" {
				return true
			}
		}
	case map[interface{}]interface{}:
		_, ok := v["list"]
		return ok
	}
	return false
}
