package usercommands

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Stolen baubles at the player's commands (docs/baubles Phase 6c): storage
// will not hold a hot one, and `give` back to its owner is a return.

// seedStolenCatalog seeds the bauble carrier and an empty catalog.
func seedStolenCatalog(t *testing.T) {
	t.Helper()
	restore := items.SeedItemsForTest(map[int]*items.ItemSpec{
		items.BaubleItemId: {ItemId: items.BaubleItemId, Name: "Curious Trinket", NameSimple: "trinket",
			Type: items.Object, Subtype: items.Mundane, Weight: 0.2, Value: 1, NotSalable: true},
	})
	baubles.SetDirForTest(t.TempDir())
	t.Cleanup(func() {
		restore()
		items.SetBaubleResolver(nil)
	})
}

// cmdBauble makes a bauble, stolen from fromMob at stolenAt, in zone
// cmdTheftZone, when stolenAt is not zero.
var cmdTheftZone = ``

func cmdBauble(t *testing.T, name string, noun string, fromMob int, stolenAt time.Time) items.Item {
	t.Helper()
	rec, err := baubles.Create(baubles.Record{Name: name, NameSimple: noun, Tier: baubles.TierCheap, Value: 4, WeightLbs: 0.3, Status: baubles.StatusReady})
	require.NoError(t, err)
	if !stolenAt.IsZero() {
		require.True(t, baubles.MarkStolen(rec.Id, baubles.Theft{ByUserId: 1, FromMob: fromMob, Zone: cmdTheftZone}, stolenAt))
	}
	it := items.New(items.BaubleItemId)
	it.Bauble = rec.Id
	return it
}

// Storage refuses a hot bauble on every path (one, several, all of a name,
// everything) and still takes what is not hot: a cooled stolen bauble and an
// honest one.
func TestStorage_RefusesAHotBauble(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	seedStolenCatalog(t)
	now := time.Now()
	origNow := storageNow
	storageNow = func() time.Time { return now }
	t.Cleanup(func() { storageNow = origNow })

	user, room := getTestUserAndRoom(t)
	room.IsStorage = true
	t.Cleanup(func() { room.IsStorage = false })
	user.Character.Stats.Strength.ValueAdj = 50
	require.NotEmpty(t, room.Zone, "fixture: the room has a zone, so 'stolen here' is a real zone")
	cmdTheftZone = room.Zone // stolen here
	t.Cleanup(func() { cmdTheftZone = `` })

	hot := cmdBauble(t, "Tarnished Brass Thimble", "thimble", 2, now.Add(-time.Hour))
	cold := cmdBauble(t, "Bone Dice", "dice", 2, now.Add(-8*24*time.Hour))
	honest := cmdBauble(t, "Painted Wooden Horse", "horse", 0, time.Time{})
	for _, it := range []items.Item{hot, cold, honest} {
		require.True(t, user.Character.StoreItem(it))
	}
	has := func(noun string) bool {
		for _, it := range user.Character.Items {
			if it.Bauble != `` {
				if rec, _ := baubles.Get(it.Bauble); rec.NameSimple == noun {
					return true
				}
			}
		}
		return false
	}

	for _, cmd := range []string{"add thimble", "add 2 thimble", "add all thimble"} {
		_, err := Storage(cmd, user, room, 0)
		require.NoError(t, err)
		assert.True(t, has("thimble"), "%q: the hot one stays with the player", cmd)
	}

	// A cool thimble of the same name is stored past the hot one.
	coolThimble := cmdBauble(t, "Pewter Thimble", "thimble", 0, time.Time{})
	require.True(t, user.Character.StoreItem(coolThimble))
	_, err := Storage("add thimble", user, room, 0)
	require.NoError(t, err)
	assert.True(t, has("thimble"), "the hot thimble is kept")
	for _, it := range user.Character.Items {
		assert.NotEqual(t, coolThimble.Bauble, it.Bauble, "the cool thimble went into storage")
	}

	// Named by its item handle, an explicit pick, the hot one is still
	// refused.
	_, err = Storage("add "+characters.ItemHandleSigil+hot.UUID.String(), user, room, 0)
	require.NoError(t, err)
	assert.True(t, has("thimble"), "a handle is no way round the refusal")

	// Stolen lately but in another town: not hot here, so stored.
	cmdTheftZone = room.Zone + " (far away)"
	farAway := cmdBauble(t, "Jade Button", "button", 2, now.Add(-time.Hour))
	require.True(t, user.Character.StoreItem(farAway))
	cmdTheftZone = room.Zone

	_, err = Storage("add all", user, room, 0)
	require.NoError(t, err)
	assert.True(t, has("thimble"), "add all leaves the hot one")
	assert.False(t, has("button"), "a bauble hot only elsewhere is stored")
	assert.False(t, has("dice"), "a cooled stolen bauble is stored")
	assert.False(t, has("horse"), "an honest bauble is stored")
}

// Giving a stolen bauble back to the mob it was taken from is a return: it
// cools at once.
func TestGive_AStolenBaubleBackToItsOwnerIsAReturn(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	seedStolenCatalog(t)
	user, room := getTestUserAndRoom(t)
	user.Character.Stats.Strength.ValueAdj = 50

	_, mobInstanceId := room.FindByName("skeleton")
	require.NotZero(t, mobInstanceId)
	owner := mobs.GetInstance(mobInstanceId)
	require.NotNil(t, owner)

	it := cmdBauble(t, "Tarnished Brass Thimble", "thimble", int(owner.MobId), time.Now().Add(-time.Hour))
	require.True(t, user.Character.StoreItem(it))

	_, err := Give("thimble skeleton", user, room, 0)
	require.NoError(t, err)

	rec, _ := baubles.Get(it.Bauble)
	assert.False(t, rec.ReturnedAt.IsZero(), "recorded as returned")
	assert.False(t, rec.Hot(time.Now()), "and cooled")
}

// `offer` and `appraise` reach a fence who keeps no shop, for baubles.
func TestOfferAndAppraise_AGoBetweenFence(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	seedStolenCatalog(t)
	user, room := getTestUserAndRoom(t)
	user.Character.Stats.Strength.ValueAdj = 50

	_, mobInstanceId := room.FindByName("skeleton")
	require.NotZero(t, mobInstanceId)
	fence := mobs.GetInstance(mobInstanceId)
	require.False(t, fence.HasShop(), "fixture: the skeleton keeps no shop")
	origGroups := fence.Groups
	fence.Groups = []string{`fence`}
	t.Cleanup(func() { fence.Groups = origGroups })

	cmdTheftZone = room.Zone
	t.Cleanup(func() { cmdTheftZone = `` })
	it := cmdBauble(t, "Tarnished Brass Thimble", "thimble", 99, time.Now().Add(-time.Hour)) // value 4
	require.True(t, user.Character.StoreItem(it))

	// Room lines (a merchant speaking) carry no user id; the player's own
	// lines carry theirs.
	said := func() string {
		return strings.Join(append(events.DrainQueuedMessagesForTest(0), events.DrainQueuedMessagesForTest(user.UserId)...), "\n")
	}
	said() // leftovers

	_, err := Offer("thimble", user, room, 0)
	require.NoError(t, err)
	assert.Contains(t, said(), "3 gold", "the go-between offers 60% of 4, rounded up")

	_, err = Appraise("thimble", user, room, 0)
	require.NoError(t, err)
	out := said()
	assert.NotContains(t, out, "need to be at a merchant")
	assert.Contains(t, out, "Tarnished Brass Thimble", "the go-between looks it over")
}
