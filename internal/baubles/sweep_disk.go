package baubles

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v3"
)

// The sweep's disk half. Every item that is not in memory is in a save
// file under DataFiles: users and their alts (the bank and inbox inside
// them), rooms.instances, mobs.instances, shops, guilds, crates, and
// plugin-data for the auction house. Like migration 0.17.0 it reads the
// whole tree rather than a list of store folders, because a list goes stale
// the day someone adds a store. Two folders are skipped: the catalog itself
// (baubles/) and economy/snapshots/, the dashboard's metric snapshots
// (internal/economy/health), which hold no items and are most of the tree's
// bytes. The rest of economy/ is read like any other folder.
//
// Cheap: a file with no `bauble` key is read but never parsed. Fail closed:
// a file that cannot be read, or that names a bauble and does not parse,
// fails the whole sweep, because skipping it would prune what it holds.
// Quarantined files (`.corrupt-...`) and util.Save's `.new` temp files do
// not end in .yaml or .plugin.dat, so they are never read.

// sweepSkipDirs are folders under DataFiles the scan skips, as slash paths
// relative to DataFiles.
var sweepSkipDirs = map[string]bool{`baubles`: true, `economy/snapshots`: true}

// untakenDir is the one folder where a find can lie untaken on a floor: a
// room file's top-level items list (floorItems).
const untakenDir = `rooms.instances`

// baubleKeyRe finds a `bauble` key, bare or quoted as JSON writes it. Not
// `baublespot` or `baubleleftat`: the name must end at the colon.
var baubleKeyRe = regexp.MustCompile(`["']?\bbauble["']?\s*:`)

// sweepReadFile reads one data file. A variable so a test can make a read
// fail.
var sweepReadFile = os.ReadFile

// maxYAMLDepth bounds the walk of one document. Saves nest a dozen levels;
// anything deeper fails the sweep rather than being walked partway.
const maxYAMLDepth = 100

// DiskRefs reads every data file under root and returns the record ids the
// items in them point at, how many files it read, and how many of those it
// parsed. A find on a room's floor in rooms.instances/ (the file's
// top-level items list) that has lain untaken past UntakenLimit is not
// counted: loading its room removes it before anything can take it
// (rooms.LoadRoomInstance). A find in a stash or a container always counts,
// because nothing removes it from there. Any error means the result is
// incomplete and must not be used to prune.
func DiskRefs(root string, now time.Time) (refs map[string]bool, files int, parsed int, err error) {
	refs = map[string]bool{}
	files, parsed, err = scanDisk(root, now, func(id string) { refs[id] = true })
	return refs, files, parsed, err
}

func scanDisk(root string, now time.Time, add func(id string)) (files int, parsed int, err error) {
	if _, serr := os.Stat(root); serr != nil {
		return 0, 0, fmt.Errorf(`data files: %w`, serr)
	}
	limit := UntakenLimit()
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, werr error) error {
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		if werr != nil {
			if errors.Is(werr, fs.ErrNotExist) {
				return nil
			}
			return fmt.Errorf(`walk %s: %w`, rel, werr)
		}
		if d.IsDir() {
			if sweepSkipDirs[rel] {
				return filepath.SkipDir
			}
			return nil
		}
		regular := d.Type().IsRegular()
		if d.Type()&fs.ModeSymlink != 0 {
			// A symlink is resolved by hand: WalkDir never follows one on
			// its own, and d.Type() here describes the link, not what it
			// points at.
			target, staterr := os.Stat(path)
			if staterr != nil {
				return fmt.Errorf(`symlink %s: %w`, rel, staterr)
			}
			if target.IsDir() {
				// Not followed (a symlinked directory could loop back on
				// itself), and not silently skipped either: it could hide
				// saves the sweep would then miss.
				mudlog.Warn(`baubles`, `action`, `sweep`, `result`, `symlinked directory not followed`, `path`, rel)
				return fmt.Errorf(`symlinked directory %s: not followed, a save could be hiding behind it`, rel)
			}
			regular = target.Mode().IsRegular()
		}
		if !regular {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, `.yaml`) && !strings.HasSuffix(name, `.plugin.dat`) {
			return nil
		}
		raw, ferr := sweepReadFile(path)
		if ferr != nil {
			if errors.Is(ferr, fs.ErrNotExist) {
				return nil // removed since the listing, and whatever it held with it
			}
			return fmt.Errorf(`read %s: %w`, rel, ferr)
		}
		files++
		if !bytes.Contains(raw, []byte(`bauble`)) || !baubleKeyRe.Match(raw) {
			return nil
		}
		parsed++
		untaken := time.Duration(0)
		if strings.HasPrefix(rel, untakenDir+`/`) {
			untaken = limit
		}
		if perr := refsInYAML(raw, now, untaken, add); perr != nil {
			return fmt.Errorf(`parse %s: %w`, rel, perr)
		}
		return nil
	})
	return files, parsed, err
}

// refsInYAML adds the id of every item in raw, every document of it.
// untaken is the untaken limit in a room file, 0 elsewhere; it applies only
// to the floor (floorItems).
func refsInYAML(raw []byte, now time.Time, untaken time.Duration, add func(id string)) error {
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	for {
		var doc yaml.Node
		if err := dec.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		floor := map[*yaml.Node]bool{}
		if untaken > 0 {
			floor = floorItems(&doc)
		}
		if err := walkYAML(&doc, 0, now, untaken, floor, add); err != nil {
			return err
		}
	}
}

// floorItems are the entries of a room file's top-level `items` list: the
// floor, the only place removeUntakenBaubles (internal/rooms) takes an
// untaken find from. Stash and container items are not in it.
func floorItems(doc *yaml.Node) map[*yaml.Node]bool {
	out := map[*yaml.Node]bool{}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return out
	}
	m := doc.Content[0]
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == `items` && m.Content[i+1].Kind == yaml.SequenceNode {
			for _, e := range m.Content[i+1].Content {
				out[e] = true
			}
		}
	}
	return out
}

func walkYAML(n *yaml.Node, depth int, now time.Time, untaken time.Duration, floor map[*yaml.Node]bool, add func(id string)) error {
	if n == nil {
		return nil
	}
	if depth > maxYAMLDepth {
		return fmt.Errorf(`nested deeper than %d levels`, maxYAMLDepth)
	}
	switch n.Kind {
	case yaml.AliasNode:
		return walkYAML(n.Alias, depth+1, now, untaken, floor, add)
	case yaml.MappingNode:
		limit := time.Duration(0)
		if floor[n] {
			limit = untaken
		}
		id, ok, ierr := itemRefIn(n, now, limit)
		if ierr != nil {
			return ierr
		}
		if ok {
			add(id)
		}
	}
	for _, c := range n.Content {
		if err := walkYAML(c, depth+1, now, untaken, floor, add); err != nil {
			return err
		}
	}
	return nil
}

// itemRefIn reads a mapping as an item: its `bauble` id, unless it is a
// find left lying untaken past untaken (when untaken is not 0). A `bauble`
// key whose value is not a scalar (a map or a sequence, never a shape this
// reader writes) is a parse failure of the file, not "no reference": reading
// it as absent would be a false negative that lets a live record be pruned.
func itemRefIn(m *yaml.Node, now time.Time, untaken time.Duration) (string, bool, error) {
	it := items.Item{}
	for i := 0; i+1 < len(m.Content); i += 2 {
		k, v := m.Content[i], m.Content[i+1]
		if k.Value == `bauble` && v.Kind != yaml.ScalarNode {
			return ``, false, fmt.Errorf(`bauble at line %d: value is a %s, not a scalar`, v.Line, yamlKindName(v.Kind))
		}
		if v.Kind != yaml.ScalarNode {
			continue
		}
		switch k.Value {
		case `bauble`:
			it.Bauble = v.Value
		case `baubleleftat`:
			it.BaubleLeftAt, _ = strconv.ParseInt(v.Value, 10, 64)
		}
	}
	if _, ok := seqOf(it.Bauble); !ok {
		return ``, false, nil
	}
	if untaken > 0 {
		if age, lying := it.BaubleUntakenFor(now); lying && age >= untaken {
			return ``, false, nil
		}
	}
	return it.Bauble, true, nil
}

// yamlKindName names a node kind for an error message.
func yamlKindName(k yaml.Kind) string {
	switch k {
	case yaml.MappingNode:
		return `mapping`
	case yaml.SequenceNode:
		return `sequence`
	case yaml.AliasNode:
		return `alias`
	case yaml.DocumentNode:
		return `document`
	default:
		return `non-scalar`
	}
}
