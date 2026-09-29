package baubles

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

// skipIfSymlinkPrivilegeDenied lets an account without
// SeCreateSymbolicLinkPrivilege (common on Windows outside Developer Mode or
// an elevated shell) skip a symlink-based test rather than fail it; any
// other error from os.Symlink still fails the test.
func skipIfSymlinkPrivilegeDenied(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(strings.ToLower(err.Error()), `privilege`) {
		t.Skip(`symlink creation needs a privilege this account does not have: ` + err.Error())
	}
	t.Fatal(err)
}

// writeDataFiles writes each file (slash path relative to root).
func writeDataFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func sortedIds(refs map[string]bool) []string {
	out := make([]string, 0, len(refs))
	for id := range refs {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// Items are found in every save shape, as YAML or JSON, in any document of
// a file; the catalog and the economy snapshots are skipped, and
// quarantined and half-written files are never read.
func TestDiskRefsFindsItemsInEverySaveShape(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	young := now.Add(-time.Hour).Unix()
	writeDataFiles(t, root, map[string]string{
		`users/7.yaml`:                      "userid: 7\ncharacter:\n  items:\n  - itemid: 900\n    bauble: B0000001\nitemstorage:\n  slots:\n  - item:\n      itemid: 900\n      bauble: B0000002\n    count: 1\ninbox:\n- item:\n    itemid: 900\n    bauble: B0000003\n",
		`users/7.alts.yaml`:                 "- name: Alt\n  items:\n  - itemid: 900\n    bauble: B0000004\n",
		`rooms.instances/ashwick/4023.yaml`: fmt.Sprintf("items:\n- itemid: 900\n  bauble: B0000005\n  baubleleftat: %d\ncontainers:\n  chest:\n    items:\n    - itemid: 900\n      bauble: B0000006\n", young),
		`plugin-data/auctions-v1.0/auctionhistory.plugin.dat`: "ActiveAuction:\n  ItemData:\n    itemid: 900\n    bauble: B0000007\n",
		`plugin-data/other-v1.0/state.plugin.dat`:             `{"lot": {"itemid": 900, "bauble": "B0000008"}}`,
		`shops/town/5-room1.yaml`:                             "---\naffixed_stock:\n- item:\n    itemid: 900\n    bauble: B0000009\n---\naffixed_stock:\n- item:\n    itemid: 900\n    bauble: B0000010\n",
		`economy/snapshots/1.yaml`:                            "item:\n  bauble: B0000011\n",
		`economy/ledger.yaml`:                                 "lot:\n  itemid: 900\n  bauble: B0000015\n",
		`baubles/catalog-0000.yaml`:                           "records:\n- id: B0000012\n  bauble: B0000012\n",
		`users/7.yaml.corrupt-20260929T000000.000000000Z`:     "character:\n  items:\n  - bauble: B0000013\n",
		`users/8.yaml.new`:                                    "character:\n  items:\n  - bauble: B0000014\n",
		`rooms.instances/ashwick/4024.yaml`:                   "items:\n- itemid: 12\n  baublespot: on the shelf\n",
		`users/9.yaml`:                                        "character:\n  description: \"a bauble: of no account\"\n  items:\n  - itemid: 900\n    bauble: not-an-id\n",
	})

	refs, files, parsed, err := DiskRefs(root, now)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{`B0000001`, `B0000002`, `B0000003`, `B0000004`, `B0000005`, `B0000006`, `B0000007`, `B0000008`, `B0000009`, `B0000010`, `B0000015`}
	if got := sortedIds(refs); !reflect.DeepEqual(got, want) {
		t.Fatalf("refs %v, want %v (economy/snapshots skipped, the rest of economy read)", got, want)
	}
	if files != 9 || parsed != 8 {
		t.Fatalf("read %d files and parsed %d, want 9 and 8 (4024 names no bauble key)", files, parsed)
	}
}

// A find lying untaken past the limit is not a reference, but only on a
// room's floor in rooms.instances (the file's top-level items list), where
// loading the room removes it (rooms.LoadRoomInstance). The same shape in
// the room's stash or a container, or in any other file, still counts:
// nothing removes it from there.
func TestDiskRefsSkipsExpiredUntakenFindsOnlyOnFloors(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	expired := now.Add(-UntakenLimit() - time.Minute).Unix()
	young := now.Add(-time.Minute).Unix()
	room := fmt.Sprintf("items:\n- itemid: 900\n  bauble: B0000001\n  baubleleftat: %d\n- itemid: 900\n  bauble: B0000002\n  baubleleftat: %d\n", expired, young) +
		fmt.Sprintf("stash:\n- itemid: 900\n  bauble: B0000004\n  baubleleftat: %d\n", expired) +
		fmt.Sprintf("containers:\n  chest:\n    items:\n    - itemid: 900\n      bauble: B0000005\n      baubleleftat: %d\n", expired)
	writeDataFiles(t, root, map[string]string{
		`rooms.instances/ashwick/4023.yaml`: room,
		`users/7.yaml`:                      fmt.Sprintf("character:\n  items:\n  - itemid: 900\n    bauble: B0000003\n    baubleleftat: %d\n", expired),
	})
	refs, _, _, err := DiskRefs(root, now)
	if err != nil {
		t.Fatal(err)
	}
	if got := sortedIds(refs); !reflect.DeepEqual(got, []string{`B0000002`, `B0000003`, `B0000004`, `B0000005`}) {
		t.Fatalf("refs %v, want [B0000002 B0000003 B0000004 B0000005]", got)
	}
}

// Anything that leaves the scan incomplete is an error; a broken file that
// names no bauble, or one deleted mid-scan, is not.
func TestDiskRefsFailsClosed(t *testing.T) {
	now := time.Now().UTC()
	t.Run(`a file that names a bauble must parse`, func(t *testing.T) {
		root := t.TempDir()
		writeDataFiles(t, root, map[string]string{`users/5.yaml`: "character:\n  items:\n  - bauble: B0000001\n   bad: [\n"})
		if _, _, _, err := DiskRefs(root, now); err == nil || !strings.Contains(err.Error(), `parse users/5.yaml`) {
			t.Fatalf("err %v, want a parse error naming users/5.yaml", err)
		}
	})
	t.Run(`a broken file that names no bauble is not parsed`, func(t *testing.T) {
		root := t.TempDir()
		writeDataFiles(t, root, map[string]string{`weather/broken.yaml`: "a: [\n"})
		if _, files, parsed, err := DiskRefs(root, now); err != nil || files != 1 || parsed != 0 {
			t.Fatalf("files %d parsed %d err %v, want 1, 0, nil", files, parsed, err)
		}
	})
	t.Run(`an unreadable file`, func(t *testing.T) {
		root := t.TempDir()
		writeDataFiles(t, root, map[string]string{`users/6.yaml`: "userid: 6\n"})
		orig := sweepReadFile
		sweepReadFile = func(string) ([]byte, error) { return nil, errors.New(`permission denied`) }
		t.Cleanup(func() { sweepReadFile = orig })
		if _, _, _, err := DiskRefs(root, now); err == nil || !strings.Contains(err.Error(), `read users/6.yaml`) {
			t.Fatalf("err %v, want a read error naming users/6.yaml", err)
		}
	})
	t.Run(`a file deleted since the listing`, func(t *testing.T) {
		root := t.TempDir()
		writeDataFiles(t, root, map[string]string{`users/6.yaml`: "userid: 6\n"})
		orig := sweepReadFile
		sweepReadFile = func(p string) ([]byte, error) { return nil, &fs.PathError{Op: `open`, Path: p, Err: fs.ErrNotExist} }
		t.Cleanup(func() { sweepReadFile = orig })
		if _, files, _, err := DiskRefs(root, now); err != nil || files != 0 {
			t.Fatalf("files %d err %v, want 0 and nil", files, err)
		}
	})
	t.Run(`a missing data folder`, func(t *testing.T) {
		if _, _, _, err := DiskRefs(filepath.Join(t.TempDir(), `nope`), now); err == nil {
			t.Fatal("a data folder that is not there must fail the scan")
		}
	})
}

// A symlinked save file is not a regular DirEntry, but the file it resolves
// to must still be read like any other save, or the id inside it is a false
// negative that lets a live record be pruned.
func TestDiskRefsFollowsSymlinkedSaveFile(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	// Outside root: if it lived inside root under a .yaml name, the walk
	// would read it directly and the test would pass without ever
	// exercising the symlink path.
	real := filepath.Join(t.TempDir(), `real-7.yaml`)
	if err := os.WriteFile(real, []byte("character:\n  items:\n  - itemid: 900\n    bauble: B0000001\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, `users`, `7.yaml`)
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	skipIfSymlinkPrivilegeDenied(t, os.Symlink(real, link))

	refs, _, _, err := DiskRefs(root, now)
	if err != nil {
		t.Fatal(err)
	}
	if got := sortedIds(refs); !reflect.DeepEqual(got, []string{`B0000001`}) {
		t.Fatalf("refs %v, want [B0000001]", got)
	}
}

// A dangling symlink cannot be resolved to a file to read, so the sweep
// fails closed naming it rather than silently skipping whatever it might
// have pointed at.
func TestDiskRefsFailsClosedOnDanglingSymlink(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	link := filepath.Join(root, `users`, `7.yaml`)
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	skipIfSymlinkPrivilegeDenied(t, os.Symlink(filepath.Join(root, `does-not-exist.yaml`), link))

	_, _, _, err := DiskRefs(root, now)
	want := filepath.ToSlash(filepath.Join(`users`, `7.yaml`))
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("err %v, want an error naming %s", err, want)
	}
}

// A symlinked directory is never followed (it could loop, or it could hide
// saves the sweep would then miss), so it fails the whole sweep closed
// rather than being silently skipped.
func TestDiskRefsFailsClosedOnSymlinkedDirectory(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	realDir := filepath.Join(t.TempDir(), `elsewhere`)
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, `1.yaml`), []byte("character:\n  items:\n  - bauble: B0000099\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, `users`, `alts`)
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	skipIfSymlinkPrivilegeDenied(t, os.Symlink(realDir, link))

	_, _, _, err := DiskRefs(root, now)
	want := filepath.ToSlash(filepath.Join(`users`, `alts`))
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("err %v, want an error naming %s", err, want)
	}
}

// A `bauble:` key whose value is a mapping or a sequence is not a reference
// this reader understands; treating it as "no reference" would be a false
// negative, so the file fails to parse instead.
func TestDiskRefsFailsClosedOnNonScalarBaubleValue(t *testing.T) {
	now := time.Now().UTC()
	t.Run(`a mapping`, func(t *testing.T) {
		root := t.TempDir()
		writeDataFiles(t, root, map[string]string{`users/5.yaml`: "character:\n  items:\n  - bauble:\n      nested: B0000001\n"})
		if _, _, _, err := DiskRefs(root, now); err == nil || !strings.Contains(err.Error(), `parse users/5.yaml`) {
			t.Fatalf("err %v, want a parse error naming users/5.yaml", err)
		}
	})
	t.Run(`a sequence`, func(t *testing.T) {
		root := t.TempDir()
		writeDataFiles(t, root, map[string]string{`users/5.yaml`: "character:\n  items:\n  - bauble:\n      - B0000001\n"})
		if _, _, _, err := DiskRefs(root, now); err == nil || !strings.Contains(err.Error(), `parse users/5.yaml`) {
			t.Fatalf("err %v, want a parse error naming users/5.yaml", err)
		}
	})
}

// Directory-level WalkDir errors (a folder that cannot be listed) must name
// the folder the same way file-level errors name the file, or the failure
// is unactionable.
func TestDiskRefsWrapsDirectoryWalkErrorsWithPath(t *testing.T) {
	if runtime.GOOS == `windows` {
		t.Skip(`chmod-based unreadability is not reliable on windows`)
	}
	if os.Geteuid() == 0 {
		t.Skip(`running as root defeats permission checks`)
	}
	root := t.TempDir()
	now := time.Now().UTC()
	locked := filepath.Join(root, `users`, `locked`)
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, `7.yaml`), []byte("character:\n  items:\n  - bauble: B0000001\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	_, _, _, err := DiskRefs(root, now)
	want := filepath.ToSlash(filepath.Join(`users`, `locked`))
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("err %v, want an error naming %s", err, want)
	}
}
