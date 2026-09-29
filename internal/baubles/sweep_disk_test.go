package baubles

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

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
