package baubles

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func goodReply() Reply {
	return Reply{
		Name:        `Painted Wooden Horse`,
		NameSimple:  `horse`,
		Description: `A child's toy horse, its red paint flaking from the mane. One wheel is missing.`,
		Material:    `pine`,
		WeightLbs:   0.6,
		Value:       14,
	}
}

func installGenerator(t *testing.T, fn GeneratorFunc) {
	t.Helper()
	SetGenerator(fn, func() GeneratorInfo { return GeneratorInfo{Name: `test`, Model: `m`} })
	t.Cleanup(func() { SetGenerator(nil, nil) })
}

func TestGenerateWithNoGeneratorIsAGenericTrinket(t *testing.T) {
	SetGenerator(nil, nil)
	res := Generate(context.Background(), GenRequest{Tier: TierRare}, nil)
	if res.Generator != GeneratorLocal || res.Reply.Name != `Trinket` || !TierRare.Range().Contains(res.Reply.Value) {
		t.Fatalf("no key, no generator: %+v", res)
	}
	if _, ok := CurrentGenerator(); ok {
		t.Fatal("nothing installed")
	}
}

func TestGenerateUsesTheModelsAnswer(t *testing.T) {
	var got GenRequest
	installGenerator(t, func(ctx context.Context, req GenRequest) (GenResult, error) {
		got = req
		return GenResult{Reply: goodReply(), Model: `gpt-test`, Tokens: 321, PromptVersion: 1}, nil
	})
	res := Generate(context.Background(), GenRequest{Tier: TierAverage, RoomTitle: `Toy Shop`}, nil)
	if res.Generator != GeneratorOpenAI || res.Reply.Name != `Painted Wooden Horse` || res.Tokens != 321 {
		t.Fatalf("model answer: %+v", res)
	}
	if got.RoomTitle != `Toy Shop` || got.Tier != TierAverage {
		t.Fatalf("request passed through: %+v", got)
	}
	if info, ok := CurrentGenerator(); !ok || info.Name != `test` {
		t.Fatal("installed generator is reported")
	}
}

func TestGenerateFallsBackOnErrorOrBadText(t *testing.T) {
	installGenerator(t, func(ctx context.Context, req GenRequest) (GenResult, error) {
		return GenResult{}, errors.New(`boom`)
	})
	if res := Generate(context.Background(), GenRequest{Tier: TierCheap}, nil); res.Generator != GeneratorLocal || res.Reply.Name != `Trinket` {
		t.Fatalf("error: %+v", res)
	}

	installGenerator(t, func(ctx context.Context, req GenRequest) (GenResult, error) {
		r := goodReply()
		r.Name = `Horse 3000`
		return GenResult{Reply: r}, nil
	})
	if res := Generate(context.Background(), GenRequest{Tier: TierCheap}, nil); res.Generator != GeneratorLocal {
		t.Fatalf("a name with digits is unusable: %+v", res)
	}
}

func TestGenerateCapsTheWait(t *testing.T) {
	installGenerator(t, func(ctx context.Context, req GenRequest) (GenResult, error) {
		<-ctx.Done()
		return GenResult{}, ctx.Err()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	res := Generate(ctx, GenRequest{Tier: TierCheap}, nil)
	if time.Since(start) > 2*time.Second || res.Generator != GeneratorLocal {
		t.Fatalf("a cancelled call returns promptly with a generic trinket: %+v", res)
	}
}

func TestCleanReply(t *testing.T) {
	r := goodReply()
	r.Name = "  Painted\t<ansi fg=\"red\">Wooden</ansi>   Horse "
	r.NameSimple = `Horse`
	r.Description = "A toy.\x1b[31m It has\nno wheels, and a tail of real horsehair."
	got, err := CleanReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != `Painted Wooden Horse` || got.NameSimple != `horse` || strings.ContainsAny(got.Description, "\x1b\n<>") {
		t.Fatalf("cleaned: %+v", got)
	}

	cases := map[string]func(*Reply){
		`empty name`:        func(r *Reply) { r.Name = `  ` },
		`digits`:            func(r *Reply) { r.Name = `Lot 42 Cup` },
		`long name`:         func(r *Reply) { r.Name = strings.Repeat(`Long `, 10) },
		`short description`: func(r *Reply) { r.Description = `A cup.` },
		`long description`:  func(r *Reply) { r.Description = strings.Repeat(`word `, 100) },
	}
	for name, change := range cases {
		r := goodReply()
		change(&r)
		if _, err := CleanReply(r); !errors.Is(err, ErrUnusableReply) {
			t.Fatalf("%s must be refused, got %v", name, err)
		}
	}
}

func TestCleanReplyKeywords(t *testing.T) {
	r := goodReply()
	r.NameSimple = `sword`
	if got, _ := CleanReply(r); got.NameSimple != `horse` {
		t.Fatalf("a real item keyword would hijack commands; the name's own last word instead, got %q", got.NameSimple)
	}
	r.NameSimple = `two words`
	if got, _ := CleanReply(r); got.NameSimple != `horse` {
		t.Fatalf("a bad keyword falls back to the last word of the name, got %q", got.NameSimple)
	}
	r.Name = `Bent Iron Key`
	r.NameSimple = ``
	if got, _ := CleanReply(r); got.Name != `Bent Iron Key` || got.NameSimple != `iron` {
		t.Fatalf("the name may keep the word; the keyword may not (the name's next word instead): %+v", got)
	}
	r.Name = `Odd Wee Key`
	if got, _ := CleanReply(r); got.NameSimple != `trinket` {
		t.Fatalf("no word of four letters or more left: trinket, got %q", got.NameSimple)
	}

	// Natural finds: general words belong to real items (the quest's
	// Reckoning Bone, the weighted stones, crafting materials); a specific
	// noun is fine.
	for _, general := range []string{`bone`, `stone`, `shell`, `gem`, `crystal`, `pearl`, `amber`} {
		r = goodReply()
		r.Name = `Weathered Fox Jawbone`
		r.NameSimple = general
		if got, _ := CleanReply(r); got.NameSimple != `jawbone` {
			t.Fatalf("%q is a real item's keyword; the specific noun of the name instead, got %q", general, got.NameSimple)
		}
	}
	for _, specific := range []string{`jawbone`, `quartz`, `agate`, `fossil`, `antler`} {
		r = goodReply()
		r.Name = `Small Natural Curio`
		r.NameSimple = specific
		if got, _ := CleanReply(r); got.NameSimple != specific {
			t.Fatalf("%q is fine as a keyword, got %q", specific, got.NameSimple)
		}
	}
}

func TestMintFromAModelResult(t *testing.T) {
	withCatalog(t)
	r := goodReply()
	r.Value = 900     // outside average
	r.WeightLbs = 400 // outside the bounds
	res := GenResult{Reply: r, Generator: GeneratorOpenAI, Model: `gpt-test`, PromptVersion: 1, Tokens: 250, Moderated: true}

	itm, rec, err := Mint(MintOpts{Source: SourceSearch, Place: NewPlace(1, `z`, ``, `city`), Tier: TierAverage, Result: &res})
	if err != nil {
		t.Fatal(err)
	}
	if rec.Status != StatusReady || rec.Generator != GeneratorOpenAI || rec.Model != `gpt-test` || rec.Tokens != 250 || !rec.Moderated {
		t.Fatalf("audit fields: %+v", rec)
	}
	if rec.Value != 15 || rec.ValueProposed != 900 || rec.WeightLbs != MaxWeightLbs || rec.WeightProposed != 400 {
		t.Fatalf("clamped, with the proposal kept: %+v", rec)
	}
	if itm.Name() != `Painted Wooden Horse` || itm.GetSpec().Weight != MaxWeightLbs {
		t.Fatal("the item shows the model's name and the clamped weight")
	}
	if !itm.IsBauble() || itm.ItemId != items.BaubleItemId {
		t.Fatal("carrier item")
	}
}

func TestRecentNames(t *testing.T) {
	withCatalog(t)
	for _, n := range []string{`Old Cup`, `Tin Bell`, `Bone Comb`} {
		_, _ = Create(Record{Name: n, Zone: `ashwick`, Generator: GeneratorOpenAI})
	}
	_, _ = Create(Record{Name: `Trinket`, Zone: `ashwick`, Generator: GeneratorLocal})
	_, _ = Create(Record{Name: `Elsewhere`, Zone: `thornwall`, Generator: GeneratorOpenAI})

	got := RecentNames(`ashwick`, 2)
	if len(got) != 2 || got[0] != `Bone Comb` || got[1] != `Tin Bell` {
		t.Fatalf("newest model names in the zone only: %v", got)
	}
}

// A keyword a loaded, authored item answers to (its keyword or its head
// noun) is refused as well as the fixed list, so `get lantern` is never a
// model-named trinket (analysis: normal-item collisions).
func TestCleanReplyKeepsOffLoadedItemsKeywords(t *testing.T) {
	orig := authoredKeyword
	authored := map[string]bool{`lantern`: true, `horse`: true}
	authoredKeyword = func(w string) bool { return authored[w] }
	t.Cleanup(func() { authoredKeyword = orig })

	r := goodReply() // "Painted Wooden Horse"
	r.NameSimple = `lantern`
	if got, _ := CleanReply(r); got.NameSimple != `wooden` {
		t.Fatalf("a loaded item's keyword, and the name's last word is one too: the next word, got %q", got.NameSimple)
	}
	authored[`wooden`], authored[`painted`] = true, true
	if got, _ := CleanReply(r); got.NameSimple != `trinket` {
		t.Fatalf("every word taken: trinket, got %q", got.NameSimple)
	}
	r.Name, r.NameSimple = `Tarnished Tin Whistle`, `lantern`
	if got, _ := CleanReply(r); got.NameSimple != `whistle` {
		t.Fatalf("falls back to the name's own last word, got %q", got.NameSimple)
	}
	r.NameSimple = `whistle`
	if got, _ := CleanReply(r); got.NameSimple != `whistle` {
		t.Fatal("a keyword no item answers to is kept")
	}
}

// With a real "Silver Dagger" and "Brass Locket" loaded, a "Tarnished Silver
// Locket" is keyed by neither word: a bauble keyed "silver" would fully
// match `get silver` and be taken instead of the dagger, which that word
// only partly matches (review of the keyword fallback).
func TestKeywordFallbackAvoidsRealItemsNameWords(t *testing.T) {
	restore := items.SeedItemsForTest(map[int]*items.ItemSpec{
		10: {ItemId: 10, Name: `Silver Dagger`, NameSimple: `dagger`},
		11: {ItemId: 11, Name: `Brass Locket`, NameSimple: `locket`},
	})
	defer restore()
	r := goodReply()
	r.Name, r.NameSimple = `Tarnished Silver Locket`, `locket`
	got, err := CleanReply(r)
	if err != nil {
		t.Fatal(err)
	}
	if got.NameSimple != `tarnished` {
		t.Fatalf("neither silver nor locket: the name's one word no real item uses, got %q", got.NameSimple)
	}
}

// Pickpocketed finds are pocket-sized: the weight is clamped to the pocket
// limit (Balance.BaublePickpocketMaxWeight, 1 lb) wherever it is set, and
// a naming the model itself weighed over the limit, or whose name is a
// thing no pocket holds, is not used at all.
func TestPickpocketFindsArePocketSized(t *testing.T) {
	if MaxWeightFor(SourcePickpocket) != 1.0 || MaxWeightFor(SourceSearch) != MaxWeightLbs {
		t.Fatalf("limits: pocket %.1f, search %.1f", MaxWeightFor(SourcePickpocket), MaxWeightFor(SourceSearch))
	}
	if got := ClampWeightFor(1.6, SourcePickpocket); got != 1.0 {
		t.Fatalf("a little over is clamped to the pocket: %.1f", got)
	}
	if got := ClampWeightFor(1.6, SourceSearch); got != 1.6 {
		t.Fatalf("a search find is not: %.1f", got)
	}
	if l := ApplyLimitsFor(Reply{WeightLbs: 1.8, Value: 3}, TierCheap, SourcePickpocket); l.Reply.WeightLbs != 1.0 || l.ProposedWeight != 1.8 {
		t.Fatalf("limits keep what was proposed, apply the pocket: %+v", l)
	}

	heavy := goodReply()
	heavy.Name, heavy.NameSimple, heavy.WeightLbs = `Iron Strongbox`, `strongbox`, 6
	installGenerator(t, func(ctx context.Context, req GenRequest) (GenResult, error) {
		return GenResult{Reply: heavy, Model: `m`}, nil
	})
	if res := Generate(context.Background(), GenRequest{Tier: TierCheap, Source: SourcePickpocket}, nil); res.Generator != GeneratorLocal || res.Reply.Name != `Trinket` {
		t.Fatalf("a strongbox from a pocket is refused: %+v", res.Reply)
	}
	if res := Generate(context.Background(), GenRequest{Tier: TierCheap, Source: SourceSearch}, nil); res.Reply.Name != `Iron Strongbox` {
		t.Fatalf("the same object found by searching is fine: %+v", res.Reply)
	}
	for name, r := range map[string]Reply{
		`over the limit`:           {Name: `Tarnished Silver Snuffbox`, NameSimple: `snuffbox`, Description: `A silver snuffbox, heavier than it looks.`, WeightLbs: 1.9, Value: 3},
		`a big thing, heavy`:       {Name: `Heavy Iron Candlestick`, NameSimple: `candlestick`, Description: `A heavy iron candlestick, crusted with wax.`, WeightLbs: 1.9, Value: 3},
		`a big thing called light`: {Name: `Small Bronze Urn`, NameSimple: `urn`, Description: `A small bronze urn with a chipped lip.`, WeightLbs: 0.4, Value: 3},
	} {
		r := r
		installGenerator(t, func(ctx context.Context, req GenRequest) (GenResult, error) {
			return GenResult{Reply: r, Model: `m`}, nil
		})
		if res := Generate(context.Background(), GenRequest{Tier: TierCheap, Source: SourcePickpocket}, nil); res.Generator != GeneratorLocal {
			t.Errorf("%s: refused, a generic trinket: %+v", name, res.Reply)
		}
	}
	small := goodReply()
	small.Name, small.NameSimple, small.WeightLbs = `Tarnished Brass Thimble`, `thimble`, 0.1
	installGenerator(t, func(ctx context.Context, req GenRequest) (GenResult, error) {
		return GenResult{Reply: small, Model: `m`}, nil
	})
	if res := Generate(context.Background(), GenRequest{Tier: TierCheap, Source: SourcePickpocket}, nil); res.Generator != GeneratorOpenAI {
		t.Fatalf("a pocket-sized thing is kept: %+v", res)
	}
}

// Minting a pickpocketed find clamps even the generic trinket to the pocket.
func TestMintHoldsAPickpocketFindToThePocket(t *testing.T) {
	withCatalog(t)
	_, rec, err := Mint(MintOpts{Source: SourcePickpocket, Tier: TierRare,
		Result: &GenResult{Reply: Reply{Name: `Silver Snuff Box`, NameSimple: `snuffbox`, Description: `A small silver snuff box with a hinged lid.`, WeightLbs: 1.9, Value: 90}, Generator: GeneratorOpenAI}})
	if err != nil || rec.WeightLbs != 1.0 || rec.WeightProposed != 1.9 || rec.Source != SourcePickpocket {
		t.Fatalf("pocket weight: %+v %v", rec, err)
	}
}
