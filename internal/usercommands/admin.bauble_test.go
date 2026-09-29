package usercommands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The admin `bauble` command (docs/baubles Phase 5): naming a bauble by id or
// by any word of its name, and the edit/retire/restore round trip.
func TestAdminBauble_ResolveEditRetire(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restoreItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		items.BaubleItemId: {ItemId: items.BaubleItemId, Name: "Curious Trinket", NameSimple: "trinket",
			Type: items.Object, Subtype: items.Mundane, Weight: 0.2, Value: 1, NotSalable: true},
	})
	defer restoreItems()
	baubles.SetDirForTest(t.TempDir())
	defer items.SetBaubleResolver(nil)

	admin, room := getTestUserAndRoom(t)
	admin.Character.Stats.Strength.ValueAdj = 50

	doll, err := baubles.Create(baubles.Record{
		Name: "Small Child's Doll", NameSimple: "doll", Tier: baubles.TierCheap, Value: 3, WeightLbs: 0.5,
		Description: "A rag doll with one button eye, much loved and much mended.",
		Status:      baubles.StatusReady, Generator: baubles.GeneratorOpenAI,
	})
	require.NoError(t, err)
	inPack := items.New(items.BaubleItemId)
	inPack.Bauble = doll.Id
	require.True(t, admin.Character.StoreItem(inPack))
	defer admin.Character.RemoveItem(inPack)

	whistle, err := baubles.Create(baubles.Record{
		Name: "Dented Tin Whistle", NameSimple: "whistle", Tier: baubles.TierCheap, Value: 2, WeightLbs: 0.2,
		Description: "A dented tin whistle that still gives a thin, reedy note.",
		Status:      baubles.StatusFallback, Generator: baubles.GeneratorLocal,
	})
	require.NoError(t, err)
	onFloor := items.New(items.BaubleItemId)
	onFloor.Bauble = whistle.Id
	room.AddItem(onFloor, false)
	defer room.RemoveItem(onFloor, false)

	// By id (any case), by any word of a name in the pack, and on the floor.
	for arg, want := range map[string]string{
		strings.ToLower(doll.Id): doll.Id,
		"doll":                   doll.Id,
		"childs doll":            doll.Id,
		"child's doll":           doll.Id,
		"whistle":                whistle.Id,
		"tin whistle":            whistle.Id,
	} {
		rec, ok := resolveBaubleArg(arg, admin, room)
		if assert.True(t, ok, "resolve %q", arg) {
			assert.Equal(t, want, rec.Id, "resolve %q", arg)
		}
	}
	_, ok := resolveBaubleArg("sword", admin, room)
	assert.False(t, ok)
	_, ok = resolveBaubleArg("B9999999", admin, room)
	assert.False(t, ok)

	// edit: the field word splits a multi-word target from the new text.
	_, err = Bauble("edit small doll name Rag Doll", admin, room, 0)
	require.NoError(t, err)
	got, _ := baubles.Get(doll.Id)
	assert.Equal(t, "Rag Doll", got.Name)
	assert.Equal(t, admin.Character.Name, got.EditedBy)
	assert.Equal(t, "Rag Doll", inPack.Name(), "the item in the pack shows the edit at once")

	_, _ = Bauble("edit rag doll value 99", admin, room, 0)
	got, _ = baubles.Get(doll.Id)
	assert.Equal(t, 6, got.Value, "value stays inside the cheap tier")

	// retire hides the text everywhere; restore brings it back.
	_, _ = Bauble("retire "+doll.Id, admin, room, 0)
	got, _ = baubles.Get(doll.Id)
	assert.Equal(t, baubles.StatusRetired, got.Status)
	assert.Equal(t, "Trinket", inPack.Name())
	_, _ = Bauble("restore "+doll.Id, admin, room, 0)
	got, _ = baubles.Get(doll.Id)
	assert.Equal(t, baubles.StatusReady, got.Status)
	assert.Equal(t, "Rag Doll", inPack.Name())

	// The read-only subcommands run without error.
	for _, sub := range []string{"stats", "status", "list", "show whistle", "prompt " + whistle.Id, "window", "regen " + whistle.Id} {
		_, err := Bauble(sub, admin, room, 0)
		assert.NoError(t, err, sub)
	}
}

// An admin sees whose text a bauble carries (review finding e), and can
// list only the player-key, the unmoderated or the finder-only ones.
func TestAdminBauble_ShowsAndFiltersPlayerKeyText(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	baubles.SetDirForTest(t.TempDir())
	defer items.SetBaubleResolver(nil)
	admin, room := getTestUserAndRoom(t)

	mine, err := baubles.Create(baubles.Record{Name: "Painted Wooden Horse", NameSimple: "horse", Tier: baubles.TierCheap,
		Value: 3, WeightLbs: 0.5, Description: "A toy horse.", Status: baubles.StatusReady,
		Generator: baubles.GeneratorOpenAI, PlayerKey: true, FoundByUserId: 7})
	require.NoError(t, err)
	_, err = baubles.Create(baubles.Record{Name: "Tin Soldier", NameSimple: "soldier", Tier: baubles.TierCheap,
		Value: 3, WeightLbs: 0.5, Description: "A tin soldier.", Status: baubles.StatusReady,
		Generator: baubles.GeneratorOpenAI, Moderated: true})
	require.NoError(t, err)

	events.DrainQueuedMessagesForTest(admin.UserId)
	_, _ = Bauble("show "+mine.Id, admin, room, 0)
	out := strings.Join(events.DrainQueuedMessagesForTest(admin.UserId), "\n")
	assert.Contains(t, out, "player key: yes")
	assert.Contains(t, out, "moderated: no")
	assert.Contains(t, out, "finder only: yes (user 7)")

	for filter, wantHorse := range map[string]bool{"playerkey": true, "unmoderated": true, "finderonly": true, "": true} {
		_, _ = Bauble(strings.TrimSpace("list 10 "+filter), admin, room, 0)
		out = strings.Join(events.DrainQueuedMessagesForTest(admin.UserId), "\n")
		assert.Equal(t, wantHorse, strings.Contains(out, "Painted Wooden Horse"), filter)
		assert.Equal(t, filter == "", strings.Contains(out, "Tin Soldier"), filter)
	}
}

// `bauble status` says when the catalog sweep last ran and what it did, or
// why it pruned nothing.
func TestBaubleSweepLine(t *testing.T) {
	every := 6 * time.Hour
	at := time.Date(2026, 9, 29, 14, 2, 0, 0, time.UTC)
	for name, tc := range map[string]struct {
		st   baubles.SweepStatus
		want []string
	}{
		`never`:  {baubles.SweepStatus{}, []string{`not run yet`, `every 6 hours`}},
		`failed`: {baubles.SweepStatus{At: at, Err: `parse users/5.yaml: bad`}, []string{`failed`, `2026-09-29 14:02 UTC`, `nothing was pruned`, `parse users/5.yaml: bad`}},
		`empty`:  {baubles.SweepStatus{At: at, OK: true, Skipped: true}, []string{`catalog was empty`}},
		`ran`: {baubles.SweepStatus{At: at, OK: true, Records: 40, Referenced: 31, Pruned: 3, Files: 412, Parsed: 17,
			Disk: 180 * time.Millisecond, Live: 4 * time.Millisecond}, []string{`40 records`, `31 still held`, `3 pruned`, `412 files`, `17 name a bauble`, `180ms`, `4ms`, `Every 6 hours`}},
		// A shard write can fail while the rest of the sweep succeeds
		// (applySweep used to drop that error): the count must be visible on
		// the line rather than folded into a silent OK.
		`shardErrors`: {baubles.SweepStatus{At: at, OK: true, Records: 40, Referenced: 31, Pruned: 3, Files: 412, Parsed: 17,
			Disk: 180 * time.Millisecond, Live: 4 * time.Millisecond, ShardErrors: 2}, []string{`2 shard write`, `failed`, `unpruned`}},
	} {
		line := baubleSweepLine(tc.st, every)
		for _, w := range tc.want {
			assert.Contains(t, line, w, name)
		}
	}
}

// adminSaid runs one bauble subcommand and returns what the admin was sent,
// whitespace folded, so a line the renderer wrapped still matches.
func adminSaid(t *testing.T, cmd string, admin *users.UserRecord, room *rooms.Room) string {
	t.Helper()
	events.DrainQueuedMessagesForTest(admin.UserId)
	_, err := Bauble(cmd, admin, room, 0)
	require.NoError(t, err, cmd)
	return strings.Join(strings.Fields(strings.Join(events.DrainQueuedMessagesForTest(admin.UserId), " ")), " ")
}

// bauble promote and bauble corpus (slice C): promote puts a record's text
// in the overlay under its biome and tier; remove takes it out again by
// name; every subcommand says what it did.
func TestAdminBauble_PromoteAndCorpus(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	restoreItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		items.BaubleItemId: {ItemId: items.BaubleItemId, Name: "Curious Trinket", NameSimple: "trinket",
			Type: items.Object, Subtype: items.Mundane, Weight: 0.2, Value: 1, NotSalable: true},
	})
	defer restoreItems()
	dir := t.TempDir()
	baubles.SetDirForTest(dir)
	defer items.SetBaubleResolver(nil)
	seedPath := filepath.Join(dir, "bauble-corpus.yaml")
	require.NoError(t, os.WriteFile(seedPath, []byte("groups:\n  interior: dwelling\nentries: {}\n"), 0o644))
	baubles.LoadCorpusFrom(seedPath, filepath.Join(dir, "corpus.promoted.yaml"))
	defer baubles.ClearCorpusForTest()

	admin, room := getTestUserAndRoom(t)
	rec, err := baubles.Create(baubles.Record{
		Name: "Painted Wooden Spool", NameSimple: "spool", Tier: baubles.TierCheap, Value: 4, WeightLbs: 0.2,
		Description: "A wooden thread spool painted with a band of faded blue.",
		Status:      baubles.StatusReady, Generator: baubles.GeneratorOpenAI, Moderated: true,
		Source: baubles.SourceSearch, Biome: "interior", Zone: "ashwick",
	})
	require.NoError(t, err)

	out := adminSaid(t, "promote "+rec.Id, admin, room)
	require.Len(t, baubles.CorpusList("interior-cheap").Promoted, 1, "promote puts it under its biome and tier")
	assert.Contains(t, out, "is now in the fallback corpus under interior-cheap")
	assert.Contains(t, adminSaid(t, "promote "+rec.Id, admin, room), "Not promoted: it is already in the corpus.")

	assert.Contains(t, adminSaid(t, "corpus", admin, room), "Usage: bauble corpus list")
	out = adminSaid(t, "corpus list", admin, room)
	assert.Contains(t, out, "Fallback corpus: 0 seed and 1 promoted entries in use.")
	assert.Contains(t, out, "interior-cheap seed 0 promoted 1")
	out = adminSaid(t, "corpus list interior-cheap", admin, room)
	// Capitalized: messaging's normalize stage capitalizes the start of every
	// CategorySystem message (pipeline.go), and here the pool key is that
	// first character.
	assert.Contains(t, out, "Interior-cheap: 0 seed, 1 promoted.")
	assert.Contains(t, out, "Painted Wooden Spool (spool, 0.2 lb, 4 gold) from "+rec.Id+", zone ashwick")
	out = adminSaid(t, "corpus export", admin, room)
	assert.Contains(t, out, "Promoted entries in the seed's format")
	assert.Contains(t, out, "name: Painted Wooden Spool")
	assert.NotContains(t, out, "from_record", "an export is seed format, no provenance")

	assert.Contains(t, adminSaid(t, "corpus remove interior-cheap Silver Spoon", admin, room),
		"Not removed: interior-cheap has no promoted entry called")
	assert.Contains(t, adminSaid(t, "corpus remove interior-cheap", admin, room), "Usage: bauble corpus remove")
	require.Len(t, baubles.CorpusList("interior-cheap").Promoted, 1, "a bad remove changes nothing")

	assert.Contains(t, adminSaid(t, "status", admin, room), "Fallback corpus: 0 seed and 1 promoted entries")
	assert.Contains(t, adminSaid(t, "stats", admin, room), "from the corpus 0")

	out = adminSaid(t, "corpus remove interior-cheap painted wooden spool", admin, room)
	assert.Contains(t, out, "Removed Painted Wooden Spool (promoted from "+rec.Id+") from interior-cheap.")
	assert.Empty(t, baubles.CorpusList("interior-cheap").Promoted)
	assert.Contains(t, adminSaid(t, "corpus export", admin, room), "No promoted entries to export.")

	out = adminSaid(t, "corpus reload", admin, room)
	assert.Contains(t, out, "Corpus reloaded: 0 seed and 0 promoted entries in use, 0 skipped.")
	_, promoted := baubles.CorpusCounts()
	assert.Equal(t, 0, promoted, "reload reads the saved overlay back from the corpus's own files")

	// A seed that breaks after boot: the reload says it kept the one in use.
	require.NoError(t, os.WriteFile(seedPath, []byte("entries: [not a map\n"), 0o644))
	out = adminSaid(t, "corpus reload", admin, room)
	assert.Contains(t, out, "The seed file could not be read")
	assert.Contains(t, out, "the seed already in use is kept")
}
