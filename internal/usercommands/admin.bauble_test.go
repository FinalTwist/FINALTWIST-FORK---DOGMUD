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

// adminSaidRaw is adminSaid without the whitespace fold, for output whose
// exact bytes matter: a table (bauble corpus list) or an export meant to be
// pasted verbatim into a tracked YAML file (bauble corpus export). Folding
// would hide exactly the corruption these review findings are about: a
// capitalized pool key, a rewritten "a"/"an", a collapsed repeated word, or
// a stray trailing period.
func adminSaidRaw(t *testing.T, cmd string, admin *users.UserRecord, room *rooms.Room) string {
	t.Helper()
	events.DrainQueuedMessagesForTest(admin.UserId)
	_, err := Bauble(cmd, admin, room, 0)
	require.NoError(t, err, cmd)
	return strings.Join(events.DrainQueuedMessagesForTest(admin.UserId), "")
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
	overviewRaw := adminSaidRaw(t, "corpus list", admin, room)
	assert.False(t, strings.HasSuffix(strings.TrimRight(overviewRaw, "\r\n"), "."),
		"the last row's trailing digit must not get a stray period: %q", overviewRaw)
	// Sent raw (review finding 2): the messaging normalizer would otherwise
	// capitalize the lowercase pool key that starts the line and append a
	// stray period to the last row (whatever character it happens to end
	// with), corrupting a table that is meant to be read verbatim.
	outRaw := adminSaidRaw(t, "corpus list interior-cheap", admin, room)
	assert.True(t, strings.HasPrefix(outRaw, "interior-cheap: 0 seed, 1 promoted."), "the pool key is not capitalized: %q", outRaw)
	assert.Contains(t, outRaw, "Painted Wooden Spool (spool, 0.2 lb, 4 gold) from "+rec.Id+", zone ashwick")
	assert.False(t, strings.HasSuffix(strings.TrimRight(outRaw, "\r\n"), "."), "no stray period appended to the last row: %q", outRaw)

	// A key that does not parse, or names no pool the loaded corpus
	// recognizes, says so instead of silently reading "0 seed, 0 promoted".
	assert.Contains(t, adminSaid(t, "corpus list nonsense", admin, room), `"nonsense" is not a biome, group, pocket or tier key`)
	assert.NotContains(t, adminSaid(t, "corpus list nonsense", admin, room), "0 seed, 0 promoted")

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

// baubleEdit and baubleRetire tell the admin when a change took text out of
// the fallback corpus, and, in red, when the corpus could not be cleaned up
// even though the record itself changed (Task 7's ErrCorpusCleanup, left
// untested by that implementer).
func TestAdminBauble_EditAndRetireReportCorpusCleanup(t *testing.T) {
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
	overlayDir := filepath.Join(dir, "ovl")
	overlayPath := filepath.Join(overlayDir, "corpus.promoted.yaml")
	baubles.LoadCorpusFrom(seedPath, overlayPath)
	defer baubles.ClearCorpusForTest()

	admin, room := getTestUserAndRoom(t)

	newPromoted := func(name, simple string) baubles.Record {
		rec, err := baubles.Create(baubles.Record{
			Name: name, NameSimple: simple, Tier: baubles.TierCheap, Value: 4, WeightLbs: 0.2,
			Description: "A wooden thread spool painted with a band of faded blue.",
			Status:      baubles.StatusReady, Generator: baubles.GeneratorOpenAI, Moderated: true,
			Source: baubles.SourceSearch, Biome: "interior", Zone: "ashwick",
		})
		require.NoError(t, err)
		_, err = baubles.Promote(rec.Id)
		require.NoError(t, err)
		return rec
	}

	// A clean edit: its one promoted entry leaves the corpus, and the admin
	// is told how many.
	edited := newPromoted("Painted Wooden Spool", "spool")
	out := adminSaid(t, "edit "+edited.Id+" material copper", admin, room)
	assert.Contains(t, out, "Its old text left the fallback corpus (promoted entries removed: 1).")
	assert.Contains(t, out, "Bauble "+edited.Id+" is now")

	// The overlay's directory is blocked: a cleanup that cannot save the
	// overlay still lets the edit (and the retire below) stand, but is
	// reported in red rather than silently dropped.
	stuck := newPromoted("Carved Walnut Button", "button")
	require.NoError(t, os.RemoveAll(overlayDir))
	require.NoError(t, os.WriteFile(overlayDir, []byte("a file where the overlay's directory should be"), 0o644))

	out = adminSaid(t, "edit "+stuck.Id+" material oak", admin, room)
	assert.Contains(t, out, "Bauble "+stuck.Id+" is now")
	assert.Contains(t, out, "the fallback corpus entries promoted from it could not be removed")
	assert.Contains(t, out, `"bauble corpus remove <key> `+stuck.Id+`"`, "the admin is told the fix")
	assert.NotContains(t, out, "promoted entries removed", "removed is 0 when cleanup itself failed")
	got, _ := baubles.Get(stuck.Id)
	assert.Equal(t, "oak", got.Material, "the record itself still changed")

	out = adminSaid(t, "retire "+stuck.Id, admin, room)
	assert.Contains(t, out, "is retired")
	assert.Contains(t, out, "the fallback corpus entries promoted from it could not be removed")
	got, _ = baubles.Get(stuck.Id)
	assert.Equal(t, baubles.StatusRetired, got.Status, "retire still retires despite the cleanup failure")

	// Review finding 5: the overview header counts entries in use
	// (CorpusCounts), and the stuck retire above left one overlay entry
	// loaded but unused. The per-key row used to count every overlay entry
	// regardless, disagreeing with the header with no explanation; it now
	// labels the difference instead.
	out = adminSaid(t, "corpus list", admin, room)
	assert.Contains(t, out, "interior-cheap seed 0 promoted 0 (1 not in use)")
}

// bauble spawn tells the admin what will actually name the find: the model
// when one is set up, otherwise the fallback corpus when it has anything for
// this pool, otherwise a plain Trinket, matching what an empty corpus
// actually produces (review finding 4: the old text always claimed the
// fallback corpus, even with nothing in it).
func TestBaubleSpawnHow_MatchesWhatNamesTheFind(t *testing.T) {
	defer baubles.ClearCorpusForTest()

	baubles.ClearCorpusForTest()
	assert.Equal(t, `a plain Trinket (no model is set up and the fallback corpus is empty)`, baubleSpawnHow(),
		"nothing loaded: the find can only be a generic trinket")

	dir := t.TempDir()
	seedPath := filepath.Join(dir, "bauble-corpus.yaml")
	require.NoError(t, os.WriteFile(seedPath, []byte(
		"groups:\n  interior: dwelling\nentries:\n  cheap:\n    - name: Knotted Twine Bracelet\n"+
			"      name_simple: bracelet\n      description: A bracelet of knotted brown twine, frayed where a wrist rubbed it.\n"+
			"      weight_lbs: 0.1\n      value: 3\n"), 0o644))
	baubles.LoadCorpusFrom(seedPath, filepath.Join(dir, "corpus.promoted.yaml"))
	assert.Equal(t, `from the fallback corpus (no model is set up)`, baubleSpawnHow(),
		"a corpus with entries: the find comes from it")
}

// bauble corpus export (review finding 1): the output is meant to be pasted
// verbatim into the tracked seed file, so it must go out exactly as
// ExportPromoted built it, never through the messaging normalizer, which
// would capitalize its first letter, rewrite "a" to "an" before a vowel,
// collapse a repeated word, and append a stray period.
func TestAdminBaubleCorpusExport_SendsRawUnnormalizedBytes(t *testing.T) {
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
	// The description carries the two constructs the normalizer mangles:
	// "a" before a vowel word, and a word repeated back to back.
	rec, err := baubles.Create(baubles.Record{
		Name: "Scratched Copper Whistle", NameSimple: "whistle", Tier: baubles.TierCheap, Value: 5, WeightLbs: 0.3,
		Description: "A trinket that grants a useful trick, though that that vendor once used it for luck.",
		Status:      baubles.StatusReady, Generator: baubles.GeneratorOpenAI, Moderated: true,
		Source: baubles.SourceSearch, Biome: "interior", Zone: "ashwick",
	})
	require.NoError(t, err)
	_, err = baubles.Promote(rec.Id)
	require.NoError(t, err)

	wantExport, err := baubles.ExportPromoted()
	require.NoError(t, err)
	wantHeader := "Promoted entries in the seed's format. The output already carries its own " +
		"entries: line and indentation; merge it into bauble-corpus.yaml's entries: map, " +
		"then wrap the descriptions:\r\n"
	want := wantHeader + strings.ReplaceAll(wantExport, "\n", "\r\n")

	got := adminSaidRaw(t, "corpus export", admin, room)
	assert.Equal(t, want, got)
	assert.Contains(t, got, "a useful", `"a" must not become "an" before a vowel`)
	assert.NotContains(t, got, "an useful")
	assert.Contains(t, got, "that that", "a repeated word must not be collapsed")
}

// bauble show keeps a bauble's last sale in view after a buyback put it back
// in a pack (baubles slice D): the header shows its status now, the sold
// line what it last sold for.
func TestAdminBauble_ShowKeepsTheLastSaleAfterABuyback(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	baubles.SetDirForTest(t.TempDir())
	defer items.SetBaubleResolver(nil)
	admin, room := getTestUserAndRoom(t)

	rec, err := baubles.Create(baubles.Record{Name: "Bone Dice", NameSimple: "dice", Tier: baubles.TierAverage,
		Value: 12, WeightLbs: 0.2, Description: "A pair of yellowed bone dice.", Status: baubles.StatusReady,
		Generator: baubles.GeneratorOpenAI})
	require.NoError(t, err)
	require.True(t, baubles.MarkSold(rec.Id, 6, 7))
	require.True(t, baubles.MarkBought(rec.Id, 8))

	out := adminSaid(t, "show "+rec.Id, admin, room)
	assert.Contains(t, out, "[ready]", "the header shows where it is now")
	assert.Contains(t, out, "last sold: 6 gold", "the sale stays visible")
}

// "Was sold" is read from SoldAt, the field SalesSince reads, not from the
// gold: a sale for nothing is still a sale, and a record never sold shows no
// sold line.
func TestAdminBauble_ShowReadsTheSaleFromSoldAt(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	baubles.SetDirForTest(t.TempDir())
	defer items.SetBaubleResolver(nil)
	admin, room := getTestUserAndRoom(t)

	rec, err := baubles.Create(baubles.Record{Name: "Bone Dice", NameSimple: "dice", Tier: baubles.TierAverage,
		Value: 12, WeightLbs: 0.2, Description: "A pair of yellowed bone dice.", Status: baubles.StatusReady,
		Generator: baubles.GeneratorOpenAI})
	require.NoError(t, err)
	assert.NotContains(t, adminSaid(t, "show "+rec.Id, admin, room), "last sold", "never sold: no sold line")

	require.True(t, baubles.MarkSold(rec.Id, 0, 7))
	assert.Contains(t, adminSaid(t, "show "+rec.Id, admin, room), "last sold: 0 gold", "sold for nothing is still sold")
}
