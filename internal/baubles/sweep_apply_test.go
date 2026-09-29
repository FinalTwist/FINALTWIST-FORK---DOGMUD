package baubles

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// sweepRecord creates a record and applies change to it.
func sweepRecord(t *testing.T, name string, change func(r *Record)) string {
	t.Helper()
	r, err := Create(Record{Name: name, NameSimple: `thing`, Tier: TierCheap, Value: 3, Status: StatusReady})
	if err != nil {
		t.Fatal(err)
	}
	if change != nil {
		Update(r.Id, change)
	}
	return r.Id
}

// readShards returns the text of every catalog shard in dir.
func readShards(t *testing.T, dir string) string {
	t.Helper()
	shards, _ := filepath.Glob(filepath.Join(dir, `catalog-*.yaml`))
	out := ``
	for _, f := range shards {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		out += string(data)
	}
	return out
}

// A record goes once two complete sweeps in a row found nothing pointing at
// it and the keep window has passed since the last sign of it. Sold,
// vanished, retired or simply lost makes no difference; a held record stays
// (a sold one included, its sale left as it was); a credited return always
// stays; only catalog shards are rewritten.
func TestApplySweepPrunesOnlyWhatNothingHolds(t *testing.T) {
	dir := t.TempDir()
	SetDirForTest(dir)
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	setBaubleConfig(t, func(b *configs.Balance) { b.BaubleCatalogKeepDays = 30 })
	overlay := filepath.Join(dir, `corpus.promoted.yaml`)
	if err := os.WriteFile(overlay, []byte("entries: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	old, recent := now.Add(-31*24*time.Hour), now.Add(-2*24*time.Hour)
	// The records meant to go were last seen by a sweep long ago (a zero
	// LastSeenAt would get the deploy grace instead: see the next test).
	lost := sweepRecord(t, `Lost Long Ago`, func(r *Record) { r.FoundAt, r.LastSeenAt = old, old })
	lostRecent := sweepRecord(t, `Lost Lately`, func(r *Record) { r.FoundAt = recent })
	soldOld := sweepRecord(t, `Sold Long Ago`, func(r *Record) {
		r.FoundAt, r.LastSeenAt, r.Status, r.SoldAt, r.SoldValue = old, old, StatusSold, old, 2
	})
	soldRecent := sweepRecord(t, `Sold Lately`, func(r *Record) { r.FoundAt, r.Status, r.SoldAt, r.SoldValue = old, StatusSold, recent, 2 })
	vanished := sweepRecord(t, `Vanished Long Ago`, func(r *Record) { r.FoundAt, r.LastSeenAt, r.VanishedAt = old, old, old })
	retired := sweepRecord(t, `Retired And Lost`, func(r *Record) { r.FoundAt, r.LastSeenAt, r.Status = old, old, StatusRetired })
	credited := sweepRecord(t, `Credited Return`, func(r *Record) {
		r.FoundAt, r.Status, r.SoldAt, r.SoldValue = old, StatusSold, old, 2
		r.ReturnCreditUserId, r.ReturnCreditFactions, r.ReturnCreditAt = 7, []string{`town`}, old
	})
	held := sweepRecord(t, `Still Held`, func(r *Record) { r.FoundAt = old })
	soldHeld := sweepRecord(t, `Sold Then Rolled Back`, func(r *Record) { r.FoundAt, r.Status, r.SoldAt, r.SoldValue = old, StatusSold, old, 2 })
	refs := map[string]bool{held: true, soldHeld: true}
	keep := KeepDuration()

	if ref, n := applySweep(now, refs, keep); ref != 2 || n != 0 {
		t.Fatalf("first sweep: %d referenced, %d pruned; want 2 and 0 (one unseen sweep is not enough)", ref, n)
	}
	later := now.Add(time.Hour)
	if _, n := applySweep(later, refs, keep); n != 4 {
		t.Fatalf("second sweep pruned %d, want 4 (lost, sold, vanished, retired: all long unseen)", n)
	}
	check := func(when string) {
		t.Helper()
		for _, id := range []string{lost, soldOld, vanished, retired} {
			if _, ok := Get(id); ok {
				t.Errorf("%s: %s should be pruned", when, id)
			}
		}
		for _, id := range []string{lostRecent, soldRecent, credited, held, soldHeld} {
			if _, ok := Get(id); !ok {
				t.Errorf("%s: %s should be kept", when, id)
			}
		}
		if n := ReturnCredits(7, `town`, 0); n != 1 {
			t.Errorf("%s: the credited record still counts (index): %d", when, n)
		}
		if r, _ := Get(soldHeld); r.Status != StatusSold || !r.LastSeenAt.Equal(later) || r.UnseenSweeps != 0 {
			t.Errorf("%s: a held sold record is seen, its sale left as it was: %+v", when, r)
		}
	}
	check(`after the sweep`)

	onDisk := readShards(t, dir)
	if strings.Contains(onDisk, `Lost Long Ago`) || !strings.Contains(onDisk, `Still Held`) || !strings.Contains(onDisk, `last_seen_at`) {
		t.Fatalf("the shards are rewritten without the pruned records and with the seen times:\n%s", onDisk)
	}
	if err := loadFrom(dir); err != nil {
		t.Fatal(err)
	}
	check(`after reload`)
	if data, err := os.ReadFile(overlay); err != nil || string(data) != "entries: {}\n" {
		t.Fatalf("a file that is not a catalog shard is untouched: %q %v", data, err)
	}
}

// Deploy grace: a record from before the sweep existed has never been seen.
// The first sweep that counts it unseen starts its keep window, so nothing
// found long ago is pruned hours after the deploy.
func TestApplySweepGivesOldRecordsAFullWindowFromTheFirstSweep(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	setBaubleConfig(t, func(b *configs.Balance) { b.BaubleCatalogKeepDays = 30 })
	now := time.Now().UTC()
	id := sweepRecord(t, `Found Before The Sweep`, func(r *Record) { r.FoundAt = now.Add(-400 * 24 * time.Hour) })
	keep := KeepDuration()

	applySweep(now, nil, keep)
	if r, _ := Get(id); !r.LastSeenAt.Equal(now) {
		t.Fatalf("the first sweep starts the keep window: %+v", r)
	}
	if _, n := applySweep(now.Add(time.Hour), nil, keep); n != 0 {
		t.Fatal("hours after the deploy, nothing found long ago is pruned")
	}
	if _, n := applySweep(now.Add(keep-time.Second), nil, keep); n != 0 {
		t.Fatal("inside the window it stays")
	}
	if _, n := applySweep(now.Add(keep), nil, keep); n != 1 {
		t.Fatal("a full keep window after the first sweep it goes")
	}
}

// One second inside the keep window a record stays; at exactly the window
// it goes.
func TestApplySweepKeepWindowBoundary(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	setBaubleConfig(t, func(b *configs.Balance) { b.BaubleCatalogKeepDays = 30 })
	id := sweepRecord(t, `Boundary Pebble`, nil)
	base := time.Now().UTC()
	keep := KeepDuration()

	applySweep(base, map[string]bool{id: true}, keep) // last seen at base
	if _, n := applySweep(base.Add(time.Hour), nil, keep); n != 0 {
		t.Fatalf("pruned %d after one unseen sweep", n)
	}
	if _, n := applySweep(base.Add(keep-time.Second), nil, keep); n != 0 {
		t.Fatal("one second inside the keep window it stays")
	}
	if _, ok := Get(id); !ok {
		t.Fatal("still there inside the window")
	}
	if _, n := applySweep(base.Add(keep), nil, keep); n != 1 {
		t.Fatal("at exactly the keep window it goes")
	}
}

// Persist before publish: a shard write that fails takes nothing out of
// memory, and the next sweep prunes it.
func TestApplySweepWriteFailurePrunesNothing(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	now := time.Now().UTC()
	id := sweepRecord(t, `Unlucky Button`, func(r *Record) {
		r.FoundAt, r.LastSeenAt, r.UnseenSweeps = now.Add(-40*24*time.Hour), now.Add(-40*24*time.Hour), minUnseenSweeps
	})

	orig := shardWriter
	shardWriter = func(string, int, []*Record) error { return errors.New(`disk full`) }
	t.Cleanup(func() { shardWriter = orig })
	if _, n := applySweep(now, nil, KeepDuration()); n != 0 {
		t.Fatalf("pruned %d with every write failing", n)
	}
	if _, ok := Get(id); !ok {
		t.Fatal("a failed write takes nothing out of memory")
	}
	shardWriter = orig
	if _, n := applySweep(now, nil, KeepDuration()); n != 1 {
		t.Fatalf("pruned %d once writes work again, want 1", n)
	}
}

// A clock stepped back between sweeps never moves a sighting earlier: a
// seen record keeps the later LastSeenAt it already has, and is still
// counted seen.
func TestApplySweepNeverMovesLastSeenBackwards(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	now := time.Now().UTC()
	id := sweepRecord(t, `Skewed Clock Bead`, func(r *Record) {
		r.FoundAt, r.LastSeenAt, r.UnseenSweeps = now.Add(-48*time.Hour), now, 1
	})
	stored, _ := Get(id)

	if ref, _ := applySweep(now.Add(-time.Hour), map[string]bool{id: true}, KeepDuration()); ref != 1 {
		t.Fatalf("referenced %d, want 1", ref)
	}
	r, _ := Get(id)
	if !r.LastSeenAt.Equal(stored.LastSeenAt) || r.UnseenSweeps != 0 {
		t.Fatalf("LastSeenAt %v UnseenSweeps %d, want %v kept and 0", r.LastSeenAt, r.UnseenSweeps, stored.LastSeenAt)
	}
}

// A record found after the sweep began is left alone.
func TestApplySweepLeavesNewerRecordsAlone(t *testing.T) {
	SetDirForTest(t.TempDir())
	t.Cleanup(func() { items.SetBaubleResolver(nil) })
	now := time.Now().UTC()
	id := sweepRecord(t, `Fresh Find`, func(r *Record) { r.FoundAt = now.Add(time.Minute) })
	applySweep(now, nil, KeepDuration())
	if r, _ := Get(id); r.UnseenSweeps != 0 {
		t.Fatalf("a find newer than the sweep was counted unseen: %+v", r)
	}
}
