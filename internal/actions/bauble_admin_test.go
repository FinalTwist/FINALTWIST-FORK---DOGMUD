package actions

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/baubles"
)

// stubBaubleJobs runs background bauble jobs in line and captures what the
// admin is told.
func stubBaubleJobs(t *testing.T) *[]string {
	t.Helper()
	told := []string{}
	origRun, origTell := runBaubleJob, tellBaubleAdmin
	runBaubleJob = func(job func(lock bool)) { job(false) }
	tellBaubleAdmin = func(userId int, text string) { told = append(told, text) }
	t.Cleanup(func() { runBaubleJob, tellBaubleAdmin = origRun, origTell })
	return &told
}

func TestRegenerateBauble(t *testing.T) {
	seedBaubleSale(t)
	told := stubBaubleJobs(t)
	baubles.SetGenerator(nil, nil)
	t.Cleanup(func() { baubles.SetGenerator(nil, nil) })

	rec, err := baubles.Create(baubles.Record{
		Name: "Trinket", NameSimple: "trinket", Tier: baubles.TierAverage, Value: 12,
		Description: "A small trinket of no particular make.", Status: baubles.StatusFallback,
		Generator: baubles.GeneratorLocal, RoomId: 424242, Zone: "nowhere", Region: "Marches",
	})
	if err != nil {
		t.Fatal(err)
	}

	// No model: nothing changes, and the admin is told why.
	if err := RegenerateBauble(rec.Id, 1, "Admin"); err != nil {
		t.Fatal(err)
	}
	if got, _ := baubles.Get(rec.Id); got.Name != "Trinket" || len(*told) != 1 || !strings.Contains((*told)[0], "not regenerated") {
		t.Fatalf("no model: %+v %q", got, *told)
	}

	var asked baubles.GenRequest
	baubles.SetGenerator(func(ctx context.Context, req baubles.GenRequest) (baubles.GenResult, error) {
		asked = req
		return baubles.GenResult{Reply: baubles.Reply{
			Name: "Tin Whistle", NameSimple: "whistle", Material: "tin",
			Description: "A dented tin whistle that still gives a thin note.", WeightLbs: 0.2, Value: 13,
		}, Model: "gpt-test"}, nil
	}, nil)

	if err := RegenerateBauble(rec.Id, 1, "Admin"); err != nil {
		t.Fatal(err)
	}
	got, _ := baubles.Get(rec.Id)
	if got.Name != "Tin Whistle" || got.Status != baubles.StatusReady || got.Region != "Marches" {
		t.Fatalf("regenerated: %+v", got)
	}
	if asked.Place.Region != "Marches" || asked.Tier != baubles.TierAverage || asked.RecentNames[len(asked.RecentNames)-1] != "Trinket" {
		t.Fatalf("request from the record, avoiding its old name: %+v", asked)
	}
	if !strings.Contains((*told)[1], "Tin Whistle") {
		t.Fatalf("admin told: %q", *told)
	}

	if err := RegenerateBauble("B9999999", 1, "Admin"); err != baubles.ErrNoRecord {
		t.Fatal("unknown id")
	}
}

// RegenerateBauble tells the admin when the old text left the fallback
// corpus, and, in red, when the corpus could not be cleaned up even though
// the record itself was regenerated (Task 7's ErrCorpusCleanup, left
// untested by that implementer).
func TestRegenerateBaubleReportsCorpusCleanup(t *testing.T) {
	seedBaubleSale(t)
	told := stubBaubleJobs(t)

	dir := t.TempDir()
	seedPath := filepath.Join(dir, "bauble-corpus.yaml")
	if err := os.WriteFile(seedPath, []byte("groups:\n  interior: dwelling\nentries: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	overlayDir := filepath.Join(dir, "ovl")
	overlayPath := filepath.Join(overlayDir, "corpus.promoted.yaml")
	baubles.LoadCorpusFrom(seedPath, overlayPath)
	t.Cleanup(baubles.ClearCorpusForTest)

	newPromoted := func(name, simple string, roomId int) baubles.Record {
		rec, err := baubles.Create(baubles.Record{
			Name: name, NameSimple: simple, Tier: baubles.TierCheap, Value: 4, WeightLbs: 0.2,
			Description: "A wooden thread spool painted with a band of faded blue.",
			Status:      baubles.StatusReady, Generator: baubles.GeneratorOpenAI, Moderated: true,
			Source: baubles.SourceSearch, Biome: "interior", Zone: "ashwick", RoomId: roomId,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := baubles.Promote(rec.Id); err != nil {
			t.Fatal(err)
		}
		return rec
	}

	baubles.SetGenerator(func(ctx context.Context, req baubles.GenRequest) (baubles.GenResult, error) {
		return baubles.GenResult{Reply: baubles.Reply{
			Name: "Tin Whistle", NameSimple: "whistle", Material: "tin",
			Description: "A dented tin whistle that still gives a thin note.", WeightLbs: 0.2, Value: 4,
		}, Model: "gpt-test", Moderated: true}, nil
	}, nil)
	t.Cleanup(func() { baubles.SetGenerator(nil, nil) })

	// A clean regeneration: its one promoted entry leaves the corpus, and
	// the admin is told how many.
	clean := newPromoted("Painted Wooden Spool", "spool", 424244)
	*told = nil
	if err := RegenerateBauble(clean.Id, 1, "Admin"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(*told, "\n"), "Its old text left the fallback corpus (promoted entries removed: 1).") {
		t.Fatalf("admin told: %q", *told)
	}

	// The overlay's directory is blocked: the regeneration still stands,
	// but the cleanup failure is reported in red rather than silently
	// dropped.
	stuck := newPromoted("Carved Walnut Button", "button", 424245)
	if err := os.RemoveAll(overlayDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overlayDir, []byte("a file where the overlay's directory should be"), 0o644); err != nil {
		t.Fatal(err)
	}
	*told = nil
	if err := RegenerateBauble(stuck.Id, 1, "Admin"); err != nil {
		t.Fatal(err)
	}
	got, _ := baubles.Get(stuck.Id)
	if got.Name != "Tin Whistle" {
		t.Fatalf("the regeneration itself still takes: %+v", got)
	}
	saidAll := strings.Join(*told, "\n")
	if !strings.Contains(saidAll, "Tin Whistle") {
		t.Fatalf("admin told the new name: %q", *told)
	}
	if !strings.Contains(saidAll, "the fallback corpus entries promoted from it could not be removed") {
		t.Fatalf("admin told the cleanup failure in red: %q", *told)
	}
	if strings.Contains(saidAll, "promoted entries removed") {
		t.Fatalf("removed is 0 when the cleanup itself failed: %q", *told)
	}
}

// A player-key record's name is not sent back to the model as a name to
// avoid when it is regenerated (spec S3): it is a player's text.
func TestBaubleRequestForRecordOmitsAPlayerKeyName(t *testing.T) {
	seedBaubleSale(t)
	rec, err := baubles.Create(baubles.Record{
		Name: "Player Written Cup", NameSimple: "cup", Tier: baubles.TierAverage, Value: 12,
		Description: "A cup a player's own key described.", Status: baubles.StatusReady,
		Generator: baubles.GeneratorOpenAI, PlayerKey: true, RoomId: 424243, Zone: "nowhere",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range BaubleRequestForRecord(rec).RecentNames {
		if n == rec.Name {
			t.Fatalf("a player-key name went into the request: %v", BaubleRequestForRecord(rec).RecentNames)
		}
	}
}
