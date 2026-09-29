package items

import "testing"

func seedBaubleCarrier(t *testing.T) {
	t.Helper()
	restore := SeedItemsForTest(map[int]*ItemSpec{
		BaubleItemId: {
			ItemId:      BaubleItemId,
			Name:        `Curious Trinket`,
			NameSimple:  `trinket`,
			Description: `carrier`,
			Type:        Object,
			Subtype:     Mundane,
			Weight:      0.2,
			Value:       1,
		},
	})
	t.Cleanup(func() {
		restore()
		SetBaubleResolver(nil)
	})
}

func testBaubleResolver(views map[string]BaubleView) BaubleResolver {
	return func(id string) (BaubleView, bool) {
		v, ok := views[id]
		return v, ok
	}
}

func TestBaubleSpecOverlaysCatalogFields(t *testing.T) {
	seedBaubleCarrier(t)
	SetBaubleResolver(testBaubleResolver(map[string]BaubleView{
		`B0000001`: {Name: `Painted Wooden Horse`, NameSimple: `horse`, Description: `A child's toy.`, Value: 4, WeightLbs: 0.6},
	}))

	b := New(BaubleItemId)
	b.Bauble = `B0000001`
	spec := b.GetSpec()
	if spec.Name != `Painted Wooden Horse` || spec.NameSimple != `horse` || spec.Description != `A child's toy.` || spec.Value != 4 || spec.Weight != 0.6 {
		t.Fatalf("overlay: %+v", spec)
	}
	if spec.ItemId != BaubleItemId || spec.Type != Object {
		t.Fatal("the rest of the spec is the carrier's")
	}
	if b.DisplayName() != `Painted Wooden Horse` {
		t.Fatalf("display name: %q", b.DisplayName())
	}
	if !b.IsBauble() || b.IsSpecial() {
		t.Fatal("a bauble is a bauble, and not special (it carries no Spec)")
	}
}

func TestBaubleWithoutRecordShowsTheCarrier(t *testing.T) {
	seedBaubleCarrier(t)

	b := New(BaubleItemId)
	b.Bauble = `B0000404`
	if b.Name() != `Curious Trinket` || b.GetSpec().Weight != 0.2 {
		t.Fatal("no resolver: the carrier")
	}
	SetBaubleResolver(testBaubleResolver(nil))
	if b.Name() != `Curious Trinket` || b.GetSpec().Value != 1 {
		t.Fatal("unknown record: the carrier")
	}
}

func TestBaubleZeroWeightKeepsTheCarrierWeight(t *testing.T) {
	seedBaubleCarrier(t)
	SetBaubleResolver(testBaubleResolver(map[string]BaubleView{`B1`: {Name: `X`, Value: 3}}))
	b := New(BaubleItemId)
	b.Bauble = `B1`
	if b.GetSpec().Weight != 0.2 {
		t.Fatal("a view with no weight must not make the item weightless")
	}
}

func TestBaublesNeverStackTogether(t *testing.T) {
	seedBaubleCarrier(t)
	a := New(BaubleItemId)
	a.Bauble = `B0000001`
	b := New(BaubleItemId)
	b.Bauble = `B0000002`
	plain := New(BaubleItemId)

	if SameStack(a, b) {
		t.Fatal("two baubles must never share a row")
	}
	if SameStack(a, plain) || SameStack(plain, a) {
		t.Fatal("a bauble never stacks with a bare carrier")
	}
	if !SameStack(a, a) {
		t.Fatal("an item is the same stack as itself")
	}
}

func TestBaubleAnswersToGenericKeywords(t *testing.T) {
	seedBaubleCarrier(t)
	SetBaubleResolver(testBaubleResolver(map[string]BaubleView{
		`B0000001`: {Name: `Tarnished Brass Buckle`, NameSimple: `buckle`, Value: 3},
	}))
	b := New(BaubleItemId)
	b.Bauble = `B0000001`
	for _, word := range []string{`buckle`, `bauble`, `trinket`, `tarnished brass buckle`} {
		if _, full := b.NameMatch(word, false); !full {
			t.Fatalf("%q must fully match", word)
		}
	}
	plain := New(BaubleItemId)
	if _, full := plain.NameMatch(`bauble`, false); full {
		t.Fatal("a bare carrier is not a bauble")
	}
}

func TestBaubleWordMatching(t *testing.T) {
	seedBaubleCarrier(t)
	SetBaubleResolver(testBaubleResolver(map[string]BaubleView{
		`B1`: {Name: `Small Child's Doll`, NameSimple: `doll`, Value: 3},
		`B2`: {Name: `Half-Burnt Tallow Candle`, NameSimple: `trinket`, Value: 2},
	}))
	doll := New(BaubleItemId)
	doll.Bauble = `B1`
	candle := New(BaubleItemId)
	candle.Bauble = `B2`

	// A bauble is a FULL match only for its exact name or its keyword, like
	// any item; any other word of its name, or several in order, is a
	// partial match, so a real item named in full always wins.
	full := []string{`doll`, `Doll`, `small child's doll`, `small childs doll`}
	for _, in := range full {
		if _, f := doll.NameMatch(in, false); !f {
			t.Errorf("%q must fully match %q", in, `Small Child's Doll`)
		}
	}
	partial := []string{`childs doll`, `child's doll`, `small doll`, `a doll`, `the doll`, `child`, `child doll`,
		`small child doll`, `dol`, `sm doll`, `chi do`}
	for _, in := range partial {
		if p, f := doll.NameMatch(in, false); !p || f {
			t.Errorf("%q must be a partial match only (p=%v f=%v)", in, p, f)
		}
	}
	for _, in := range []string{`doll small`, `horse`, `a`, `the`, `dolls`} {
		if p, f := doll.NameMatch(in, false); p || f {
			t.Errorf("%q must not match (p=%v f=%v)", in, p, f)
		}
	}

	// A word the model did not choose as the keyword still finds it, as
	// does a hyphenated word, as partial matches.
	for _, in := range []string{`candle`, `tallow candle`, `burnt candle`, `half burnt candle`, `half-burnt candle`} {
		if p, f := candle.NameMatch(in, false); !p || f {
			t.Errorf("%q must partially match %q (p=%v f=%v)", in, `Half-Burnt Tallow Candle`, p, f)
		}
	}
	if _, f := candle.NameMatch(`half-burnt tallow candle`, false); !f {
		t.Error("the whole name is a full match")
	}
}

// The lookups every command uses (get, drop, look, appraise, sell, give)
// all go through FindMatchIn, so this is what "get doll" resolves to.
func TestFindMatchInPicksTheBaubleByAnyWord(t *testing.T) {
	restore := SeedItemsForTest(map[int]*ItemSpec{
		BaubleItemId: {ItemId: BaubleItemId, Name: `Curious Trinket`, NameSimple: `trinket`, Type: Object, Weight: 0.2, Value: 1},
		7001:         {ItemId: 7001, Name: `Dollmaker's Awl`, NameSimple: `awl`, Type: Object, Value: 5},
	})
	t.Cleanup(func() { restore(); SetBaubleResolver(nil) })
	SetBaubleResolver(testBaubleResolver(map[string]BaubleView{
		`B1`: {Name: `Small Child's Doll`, NameSimple: `toy`, Value: 3},
		`B2`: {Name: `Rag Doll`, NameSimple: `doll`, Value: 2},
	}))
	awl := New(7001)
	doll := New(BaubleItemId)
	doll.Bauble = `B1`
	rag := New(BaubleItemId)
	rag.Bauble = `B2`

	// "doll" is the rag doll's keyword: a full match. The other doll and
	// the awl only match in part.
	if _, got := FindMatchIn(`doll`, awl, doll, rag); got.Bauble != `B2` {
		t.Fatalf("doll -> %+v", got)
	}
	// An explicit N. counts partial matches in list order.
	if got, _ := FindMatchIn(`3.doll`, awl, doll, rag); got.Bauble != `B2` {
		t.Fatalf("3.doll -> %+v", got)
	}
	if got, _ := FindMatchIn(`childs doll`, awl, rag, doll); got.Bauble != `B1` {
		t.Fatalf("childs doll -> %+v", got)
	}
	if got, _ := FindMatchIn(`rag`, awl, doll, rag); got.Bauble != `B2` {
		t.Fatalf("rag -> %+v", got)
	}
	if _, got := FindMatchIn(`awl`, awl, doll, rag); got.ItemId != 7001 {
		t.Fatal("ordinary items still match as before")
	}
}

// The owner's report: a bauble's name words were picked over a real item
// with similar words. A real item always wins: named in full it is a full
// match (a bauble's word match is only partial), and when both match only in
// part, the real item is preferred, wherever it is in the list.
func TestRealItemsBeatBaubles(t *testing.T) {
	restore := SeedItemsForTest(map[int]*ItemSpec{
		BaubleItemId: {ItemId: BaubleItemId, Name: `Curious Trinket`, NameSimple: `trinket`, Type: Object, Weight: 0.2, Value: 1},
		7002:         {ItemId: 7002, Name: `Iron Shield`, NameSimple: `shield`, Type: Object, Value: 50},
		7003:         {ItemId: 7003, Name: `Battered Kite Shield`, NameSimple: `kiteshield`, Type: Object, Value: 30},
	})
	t.Cleanup(func() { restore(); SetBaubleResolver(nil) })
	SetBaubleResolver(testBaubleResolver(map[string]BaubleView{
		`B1`: {Name: `Dented Shield Boss`, NameSimple: `boss`, Value: 3},
	}))
	charm := New(BaubleItemId)
	charm.Bauble = `B1`
	iron := New(7002)
	kite := New(7003)

	if _, got := FindMatchIn(`shield`, charm, iron); got.ItemId != 7002 {
		t.Fatalf("shield -> the iron shield, got %+v", got)
	}
	if got, _ := FindMatchIn(`shield`, charm, kite); got.ItemId != 7003 {
		t.Fatalf("both in part: the real item, even after the bauble in the list, got %+v", got)
	}
	if got, _ := FindMatchIn(`dented shield`, charm, kite); got.Bauble != `B1` {
		t.Fatalf("words only the bauble has still find it, got %+v", got)
	}
	if _, got := FindMatchIn(`boss`, charm, iron); got.Bauble != `B1` {
		t.Fatalf("its keyword is a full match, got %+v", got)
	}
}

// A real item beats a bauble only on an equal or stronger match (owner
// review): a bauble named by a whole word beats a real item the word only
// starts ("button" for a Buttoned Leather Vest) or sits inside, in either
// list order; a real item named by a whole word too still wins the tie.
// With no bauble among the matches, list order decides as it always has.
func TestARealItemWinsOnlyOnAnEqualOrStrongerMatch(t *testing.T) {
	restore := SeedItemsForTest(map[int]*ItemSpec{
		BaubleItemId: {ItemId: BaubleItemId, Name: `Curious Trinket`, NameSimple: `trinket`, Type: Object, Weight: 0.2, Value: 1},
		7101:         {ItemId: 7101, Name: `Buttoned Leather Vest`, NameSimple: `vest`, Type: Object, Value: 20},
		7102:         {ItemId: 7102, Name: `Button Hook`, NameSimple: `hook`, Type: Object, Value: 5},
		7103:         {ItemId: 7103, Name: `Rebuttoned Coat`, NameSimple: `coat`, Type: Object, Value: 20},
	})
	t.Cleanup(func() { restore(); SetBaubleResolver(nil) })
	SetBaubleResolver(testBaubleResolver(map[string]BaubleView{
		// Keywords are kept off words real items use, so neither bauble's
		// keyword is the word typed: they match "button" by name only.
		`B1`: {Name: `Tarnished Copper Button`, NameSimple: `copper`, Value: 3},
		`B2`: {Name: `Buttonwood Charm`, NameSimple: `charm`, Value: 3},
	}))
	button := New(BaubleItemId)
	button.Bauble = `B1`
	charm := New(BaubleItemId)
	charm.Bauble = `B2`
	vest, hook, coat := New(7101), New(7102), New(7103)

	pick := func(input string, list ...Item) Item {
		part, full := FindMatchIn(input, list...)
		if full.ItemId != 0 {
			return full
		}
		return part
	}

	if got := pick(`butt`, vest, button); got.Bauble != `` {
		t.Fatalf("both only start a word: the real item wins the tie, got %+v", got)
	}
	if got := pick(`copper`, vest, button); got.Bauble != `B1` {
		t.Fatalf("its keyword is a full match, got %+v", got)
	}
	for _, list := range [][]Item{{vest, button}, {button, vest}} {
		if got := pick(`button`, list...); got.Bauble != `B1` {
			t.Fatalf("a whole word beats a word the name only starts (order %v), got %+v", list[0].ItemId, got)
		}
	}
	for _, list := range [][]Item{{hook, button}, {button, hook}} {
		if got := pick(`button`, list...); got.ItemId != 7102 {
			t.Fatalf("a real item named by a whole word too wins the tie, got %+v", got)
		}
	}
	if got := pick(`button`, coat, charm); got.Bauble != `B2` {
		t.Fatalf("a bauble the word starts beats a real item it only sits inside, got %+v", got)
	}
	if got := pick(`2.button`, button, vest); got.ItemId != 7101 {
		t.Fatalf("an explicit N. keeps list order, got %+v", got)
	}
	if got := pick(`butt`, vest, hook); got.ItemId != 7101 {
		t.Fatalf("no bauble among the matches: list order as before, got %+v", got)
	}
}

// A bauble among the matches must not re-order ordinary items against each
// other (owner ruling, 2026-09-29): among real items the original
// list-order rule decides exactly as if no bauble were there, and the
// ranking only decides between that real item and a bauble. "iron" picks
// the Ironwood Staff, listed first, with or without a bauble that also
// starts with "iron", in any position.
func TestABaubleDoesNotReorderRealItems(t *testing.T) {
	restore := SeedItemsForTest(map[int]*ItemSpec{
		BaubleItemId: {ItemId: BaubleItemId, Name: `Curious Trinket`, NameSimple: `trinket`, Type: Object, Weight: 0.2, Value: 1},
		7201:         {ItemId: 7201, Name: `Ironwood Staff`, NameSimple: `staff`, Type: Object, Value: 20},
		7202:         {ItemId: 7202, Name: `Iron Dagger`, NameSimple: `dagger`, Type: Object, Value: 20},
	})
	t.Cleanup(func() { restore(); SetBaubleResolver(nil) })
	SetBaubleResolver(testBaubleResolver(map[string]BaubleView{
		`B1`: {Name: `Ironbound Locket`, NameSimple: `locket`, Value: 3},
	}))
	locket := New(BaubleItemId)
	locket.Bauble = `B1`
	staff, dagger := New(7201), New(7202)

	pick := func(input string, list ...Item) Item {
		part, full := FindMatchIn(input, list...)
		if full.ItemId != 0 {
			return full
		}
		return part
	}

	if got := pick(`iron`, staff, dagger); got.ItemId != 7201 {
		t.Fatalf("no bauble: list order picks the staff, got %+v", got)
	}
	for _, list := range [][]Item{{staff, dagger, locket}, {locket, staff, dagger}, {staff, locket, dagger}} {
		if got := pick(`iron`, list...); got.ItemId != 7201 {
			t.Fatalf("a bauble in the list must not change which real item wins: want the staff, got %+v", got)
		}
	}
}

// A household bauble named by a whole word ("Stub of Candle") must not
// beat a real item the word starts ("Candlestick") for `candle`.
func TestAHouseholdBaubleDoesNotBeatARealCandlestick(t *testing.T) {
	restore := SeedItemsForTest(map[int]*ItemSpec{
		BaubleItemId: {ItemId: BaubleItemId, Name: `Curious Trinket`, NameSimple: `trinket`, Type: Object, Weight: 0.2, Value: 1},
		7301:         {ItemId: 7301, Name: `Candlestick`, NameSimple: `candlestick`, Type: Object, Value: 10},
	})
	t.Cleanup(func() { restore(); SetBaubleResolver(nil) })
	SetBaubleResolver(testBaubleResolver(map[string]BaubleView{
		`B1`: {Name: `Stub of Candle`, NameSimple: `stub`, Value: 1},
	}))
	stub := New(BaubleItemId)
	stub.Bauble = `B1`
	stub.BaubleHousehold = 5
	stick := New(7301)

	for _, list := range [][]Item{{stub, stick}, {stick, stub}} {
		part, full := FindMatchIn(`candle`, list...)
		got := part
		if full.ItemId != 0 {
			got = full
		}
		if got.ItemId != 7301 {
			t.Fatalf("candle -> the candlestick, got %+v", got)
		}
	}
}
