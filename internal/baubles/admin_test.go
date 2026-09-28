package baubles

import (
	"strings"
	"testing"
)

func seedRecord(t *testing.T, r Record) Record {
	t.Helper()
	if r.Description == `` {
		r.Description = `A child's toy horse, its red paint flaking from the mane.`
	}
	rec, err := Create(r)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestCatalogStats(t *testing.T) {
	withCatalog(t)
	seedRecord(t, Record{Name: `A`, Tier: TierCheap, Value: 3, Status: StatusFallback, Generator: GeneratorLocal, Region: `Marches`})
	seedRecord(t, Record{Name: `B`, Tier: TierRare, Value: 90, Status: StatusReady, Generator: GeneratorOpenAI, Region: `Marches`, Tokens: 200})
	seedRecord(t, Record{Name: `C`, Tier: TierAverage, Value: 12, Status: StatusSold, Generator: GeneratorOpenAI, Region: `Thornwall`, Tokens: 150, EditedBy: `admin`})

	st := CatalogStats()
	if st.Total != 3 || st.ByStatus[StatusSold] != 1 || st.ByGenerator[GeneratorOpenAI] != 2 || st.ByTier[TierRare] != 1 {
		t.Fatalf("counts: %+v", st)
	}
	if st.Unsold != 2 || st.UnsoldValue != 93 || st.Tokens != 350 || st.Edited != 1 {
		t.Fatalf("totals: %+v", st)
	}
	if len(st.TopRegions) != 2 || st.TopRegions[0] != (RegionCount{`Marches`, 2}) {
		t.Fatalf("regions: %+v", st.TopRegions)
	}
}

func TestRetireAndRestore(t *testing.T) {
	withCatalog(t)
	r := seedRecord(t, Record{Name: `Rude Name`, Tier: TierCheap, Value: 3, Status: StatusReady, Generator: GeneratorOpenAI})

	if err := Retire(r.Id, `Admin`); err != nil {
		t.Fatal(err)
	}
	got, _ := Get(r.Id)
	if got.Status != StatusRetired || got.EditedBy != `Admin` || got.View().Name != retiredName || got.View().Value != 3 {
		t.Fatalf("retired: %+v view %+v", got, got.View())
	}
	if err := Restore(r.Id, `Admin`); err != nil {
		t.Fatal(err)
	}
	if got, _ := Get(r.Id); got.Status != StatusReady || got.View().Name != `Rude Name` {
		t.Fatalf("restored: %+v", got)
	}
	if Retire(`B0009999`, `Admin`) != ErrNoRecord || Restore(`B0009999`, `Admin`) != ErrNoRecord {
		t.Fatal("unknown ids")
	}
}

func TestEdit(t *testing.T) {
	withCatalog(t)
	r := seedRecord(t, Record{Name: `Painted Wooden Horse`, NameSimple: `horse`, Tier: TierAverage, Value: 12, WeightLbs: 0.6, Status: StatusReady})

	got, err := Edit(r.Id, `name`, `  Small <b>Child's</b> Doll `, `Admin`)
	if err != nil || got.Name != `Small Child's Doll` || got.EditedBy != `Admin` {
		t.Fatalf("name: %+v %v", got, err)
	}
	if got, _ := Edit(r.Id, `keyword`, `sword`, `Admin`); got.NameSimple != `doll` {
		t.Fatalf("a real item's keyword is refused, as for the model: %q", got.NameSimple)
	}
	if got, _ := Edit(r.Id, `value`, `999`, `Admin`); got.Value != 15 {
		t.Fatal("value is clamped to the tier")
	}
	if got, _ := Edit(r.Id, `tier`, `rare`, `Admin`); got.Tier != TierRare || got.Value != 40 {
		t.Fatalf("changing tier re-clamps the value: %+v", got)
	}
	if got, _ := Edit(r.Id, `weight`, `60`, `Admin`); got.WeightLbs != MaxWeightLbs {
		t.Fatal("weight is clamped")
	}
	for field, value := range map[string]string{`value`: `lots`, `weight`: `heavy`, `tier`: `legendary`, `colour`: `red`, `name`: `Horse 3000`, `desc`: `short`} {
		if _, err := Edit(r.Id, field, value, `Admin`); err == nil {
			t.Fatalf("%s=%q must be refused", field, value)
		}
	}
	if _, err := Edit(`B0009999`, `name`, `X`, `Admin`); err != ErrNoRecord {
		t.Fatal("unknown id")
	}
}

func TestApplyRegenerated(t *testing.T) {
	withCatalog(t)
	r := seedRecord(t, Record{Name: `Trinket`, NameSimple: `trinket`, Tier: TierAverage, Value: 11, Status: StatusFallback, Generator: GeneratorLocal, Stolen: true, Region: `Marches`})

	if _, err := ApplyRegenerated(r.Id, GenResult{Reply: GenericTrinket(TierAverage, nil), Generator: GeneratorLocal}, `Admin`); err == nil {
		t.Fatal("a generic answer never replaces a record")
	}
	got, err := ApplyRegenerated(r.Id, GenResult{Reply: goodReply(), Generator: GeneratorOpenAI, Model: `gpt-test`, Tokens: 90, PromptVersion: 1}, `Admin`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != `Painted Wooden Horse` || got.Status != StatusReady || got.Generator != GeneratorOpenAI || got.Tokens != 90 {
		t.Fatalf("regenerated: %+v", got)
	}
	if !got.Stolen || got.Region != `Marches` || got.Tier != TierAverage || !strings.Contains(got.EditedBy, `regen`) {
		t.Fatalf("provenance and theft are kept: %+v", got)
	}
}

func TestPromptPreviewSeamAndIds(t *testing.T) {
	SetPromptPreview(nil)
	if _, ok := PreviewPrompt(GenRequest{}); ok {
		t.Fatal("nothing installed")
	}
	SetPromptPreview(func(req GenRequest) []string { return []string{`sys`, req.RoomTitle} })
	t.Cleanup(func() { SetPromptPreview(nil) })
	if msgs, ok := PreviewPrompt(GenRequest{RoomTitle: `Hall`}); !ok || msgs[1] != `Hall` {
		t.Fatal("preview")
	}
	if !LooksLikeId(`b0000012`) || !LooksLikeId(`B0000012`) || LooksLikeId(`doll`) || LooksLikeId(`b`) {
		t.Fatal("id shapes")
	}
}
