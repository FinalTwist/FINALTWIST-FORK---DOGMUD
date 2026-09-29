package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/baubles"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/guilds"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/sealedcrate"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/modules/auctions"
	"gopkg.in/yaml.v2"
)

// Every live root the guards check is registered as a sweep source, and
// nothing else is. The same names are the list the sweep expects at run
// time (baubleSweepSourceNames): a sweep missing one of them fails closed.
func TestBaubleSweepSourcesMatchTheGuardedRoots(t *testing.T) {
	registerBaubleSweepSources() // the auction house registers itself in its init
	want := []string{}
	for _, r := range sweepRoots() {
		want = append(want, r.name)
	}
	sort.Strings(want)
	if got := baubles.LiveSourceNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("live sources %v, want %v", got, want)
	}
	expected := append([]string{}, baubleSweepSourceNames...)
	sort.Strings(expected)
	if !reflect.DeepEqual(expected, want) {
		t.Fatalf("baubleSweepSourceNames %v, want %v", expected, want)
	}
	if got := baubles.ExpectedLiveSourceNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected live sources %v, want %v", got, want)
	}
}

func sweepBauble(n int) items.Item {
	return items.Item{ItemId: items.BaubleItemId, Bauble: fmt.Sprintf(`B%07d`, n)}
}

func writeSweepFile(t *testing.T, root string, rel string, v any) {
	t.Helper()
	data, err := yaml.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// Each store's real save shape, written with the library and layout its
// store uses, is read by the sweep's disk half.
func TestBaubleSweepReadsEveryStoreFromDisk(t *testing.T) {
	root := t.TempDir()
	inboxItem := sweepBauble(4)
	writeSweepFile(t, root, `users/7.yaml`, &users.UserRecord{
		UserId: 7, Username: `sweeper`,
		Character: &characters.Character{
			Name:      `Sweeper`,
			Items:     []items.Item{sweepBauble(1)},
			Equipment: characters.Worn{Neck: sweepBauble(2)},
		},
		ItemStorage: users.Storage{Slots: []users.StorageSlot{{Item: sweepBauble(3), Count: 1}}},
		Inbox:       users.Inbox{{FromName: `a friend`, Item: &inboxItem}},
	})
	writeSweepFile(t, root, `users/7.alts.yaml`, []characters.Character{{Name: `Alt`, Items: []items.Item{sweepBauble(5)}}})
	writeSweepFile(t, root, `rooms.instances/test_zone/1.yaml`, &rooms.Room{
		RoomId:     1,
		Items:      []items.Item{sweepBauble(6)},
		Stash:      []items.Item{sweepBauble(7)},
		Containers: map[string]rooms.Container{`chest`: {Items: []items.Item{sweepBauble(8)}}},
	})
	writeSweepFile(t, root, `mobs.instances/test_zone/1-guard-1.yaml`, &mobs.MobInstanceData{Equipment: &characters.Worn{Weapon: sweepBauble(9)}})
	writeSweepFile(t, root, `shops/test_zone/5-room1.yaml`, &shops.ShopInventory{AffixedStock: []shops.AffixedStockEntry{{Item: sweepBauble(10), Price: 5}}})
	writeSweepFile(t, root, `guilds/tst.yaml`, &guilds.Guild{Tag: `TST`, Name: `Testers`, Vault: []items.Item{sweepBauble(11)}})
	crate := sealedcrate.New(1, 5)
	crate.Add(sweepBauble(12))
	if err := sealedcrate.SaveTo(filepath.Join(root, `crates`, `1-test.yaml`), crate); err != nil {
		t.Fatal(err)
	}
	writeSweepFile(t, root, `plugin-data/auctions-v1.0/auctionhistory.plugin.dat`, &auctions.AuctionManager{
		ActiveAuction: &auctions.AuctionItem{ItemData: sweepBauble(13)},
		SeizedQueue:   []auctions.SeizedLot{{Item: sweepBauble(14), Count: 1}},
	})

	refs, _, _, err := baubles.DiskRefs(root, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 14; n++ {
		if id := fmt.Sprintf(`B%07d`, n); !refs[id] {
			t.Errorf("%s not found on disk", id)
		}
	}
}
