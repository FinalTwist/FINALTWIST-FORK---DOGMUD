package timber

import "testing"

const sample = `
species:
  - {id: pine, name: pine, log: 1, tier: 1}
  - {id: yew,  name: yew,  log: 2, tier: 3}
biomes:
  forest: [{species: pine, weight: 3}, {species: yew, weight: 1}]
zones:
  Yew Vale: [{species: yew, weight: 1}]
`

func TestParse_ValidatesAndPools(t *testing.T) {
	d, err := Parse([]byte(sample), World{ItemExists: func(id int) bool { return id == 1 || id == 2 }})
	if err != nil {
		t.Fatal(err)
	}
	Install(d)
	defer Install(nil)
	if !IsChoppable(`forest`) || IsChoppable(`interior`) {
		t.Error("only forest grows timber here")
	}
	if p := Pool(`Yew Vale`, `forest`); len(p) != 1 || p[0].Species != `yew` {
		t.Errorf("zone pool should replace the biome pool, got %+v", p)
	}
	if p := Pool(`Yew Vale`, `interior`); p != nil {
		t.Error("a zone pool never makes a non-timber biome choppable")
	}
	if GetSpecies(`yew`).Tier != 3 {
		t.Error("species lookup")
	}

	bad := []string{
		"species:\n  - {id: pine, name: pine, log: 9, tier: 1}\nbiomes:\n  forest: [{species: pine, weight: 1}]\n",
		"species:\n  - {id: pine, name: pine, log: 1, tier: 7}\n",
		"species:\n  - {id: pine, name: pine, log: 1, tier: 1}\nbiomes:\n  forest: [{species: oak, weight: 1}]\n",
		"species:\n  - {id: pine, name: pine, log: 1, tier: 1}\nbiomes:\n  forest: [{species: pine, weight: 0}]\n",
		"species:\n  - {id: pine, name: pine, log: 1, tier: 1, typo: 3}\n",
	}
	for i, src := range bad {
		if _, err := Parse([]byte(src), World{ItemExists: func(id int) bool { return id == 1 }}); err == nil {
			t.Errorf("bad file %d accepted", i)
		}
	}
}

type memStore map[string]any

func (m memStore) GetLongTermData(k string) any    { return m[k] }
func (m memStore) SetLongTermData(k string, v any) { m[k] = v }

func TestStand_RoundTripFellAndRegrow(t *testing.T) {
	store := memStore{}
	if _, ok := LoadStand(store); ok {
		t.Fatal("an unseeded room has no stand")
	}
	st := NewStand(`oak`, 3, 3, 1000, func(int) int { return 0 })
	SaveStand(store, st)
	got, ok := LoadStand(store)
	if !ok || got != st {
		t.Fatalf("round trip: %+v vs %+v", got, st)
	}

	for i := 0; i < 3; i++ {
		if !got.Fell(2000) {
			t.Fatalf("felling %d failed", i)
		}
	}
	if got.Fell(2000) || got.Stock != 0 || !got.Felled {
		t.Fatal("a stand felled to nothing has no more trees and is marked felled")
	}
	if got.Regrow(2000+899, 900) {
		t.Error("nothing regrows inside one period")
	}
	if !got.Regrow(2000+900, 900) || got.Stock != 1 || got.Felled {
		t.Errorf("one period grows one tree and signals a re-roll: %+v", got)
	}
	got.Regrow(2000+900*10, 900)
	if got.Stock != got.Max {
		t.Errorf("regrowth caps at max: %+v", got)
	}
	if w := got.RoundsToNextTree(9999999, 900); w != 0 {
		t.Error("a full stand has nothing to wait for")
	}
}

func TestPickSpecies_NeighboursLean(t *testing.T) {
	pool := []Weighted{{`pine`, 1}, {`oak`, 1}}
	counts := map[string]int{}
	for i := 0; i < 200; i++ {
		n := i
		counts[PickSpecies(pool, []string{`oak`, `oak`}, func(k int) int { return n % k })]++
	}
	if counts[`oak`] <= counts[`pine`] {
		t.Errorf("neighbouring oak should lean the draw to oak, got %v", counts)
	}
	if counts[`pine`] == 0 {
		t.Error("no species is ever excluded")
	}
	if PickSpecies(nil, nil, func(int) int { return 0 }) != `` {
		t.Error("empty pool")
	}
}

// Common woods take any axe; yew and walnut want iron, ironwood steel.
func TestMinAxe(t *testing.T) {
	for tier, want := range map[int]int{1: 1, 2: 1, 3: 2, 4: 3} {
		sp := Species{Tier: tier}
		if got := sp.MinAxe(); got != want {
			t.Errorf("tier %d: MinAxe %d, want %d", tier, got, want)
		}
	}
}
