package health_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The admin economy page groups shops by Type() and looks each group's
// score up by the key PerCraftSupportScores uses, labelling the empty key
// only when it renders it (baubles slice D). The page is JavaScript with no
// harness of its own, so this reads its source: a static check that the
// grouping key and the score lookup agree.
func TestEconomyPage_GroupsByTypeAndLooksUpScoresByKey(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	page, err := os.ReadFile(filepath.Join(filepath.Dir(here), "..", "..", "..", "_datafiles", "html", "admin", "economy", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(page)
	for _, want := range []string{
		`var disc = s.fence ? "fence" : (s.craft_support || "");`,
		`var score = d.scores.PerCraftSupport[disc] || 0;`,
		`disc || "(uncategorized)",`,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("index.html lacks %q", want)
		}
	}
	if strings.Contains(src, `s.craft_support || "(uncategorized)"`) {
		t.Error(`grouping under "(uncategorized)" looks the score up under a key PerCraftSupportScores never uses ("")`)
	}
}
