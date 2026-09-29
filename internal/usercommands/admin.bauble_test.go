package usercommands

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/items"
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
	} {
		line := baubleSweepLine(tc.st, every)
		for _, w := range tc.want {
			assert.Contains(t, line, w, name)
		}
	}
}
