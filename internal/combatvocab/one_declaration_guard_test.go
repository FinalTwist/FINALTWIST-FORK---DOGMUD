package combatvocab

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// A defence name declared as a Go string literal outside this package is
// the drift this package exists to end: three declarations of "dodge" is how
// M4b-2 started. actionspec's cost-action keys share the spelling and are a
// different namespace, so that one file is allowed.
//
// The literal must be the WHOLE right-hand side of the assignment (optionally
// followed by a trailing comment), not just the first value in a multi-value
// assignment: combat's narration helpers write
// `verbYou, verbThey = "dodge", "dodges"` to pick a verb pair, which is not a
// declaration of the name anywhere near the sense this guard polices. The
// looser form `=\s*["`](dodge|...)["`]` (no end anchor) matches that
// assignment too, at internal/combat/combat_helpers.go:1469 and
// internal/combat/surprise_narration.go:61, and is a false positive.
var defenceLiteral = regexp.MustCompile("(?m)=\\s*[\"`](dodge|parry|block|quell|defy)[\"`]\\s*(//.*)?$")

var allowedFiles = map[string]bool{
	"internal/combatvocab/vocab.go": true,
	"internal/actionspec/action.go": true,
}

func repoRoot(t *testing.T) string {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	return filepath.Join(filepath.Dir(here), "..", "..")
}

func scanForDefenceLiterals(t *testing.T, root string) (hits []string, scanned int) {
	for _, base := range []string{"internal", "modules"} {
		err := filepath.WalkDir(filepath.Join(root, base), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			rel = filepath.ToSlash(rel)
			if allowedFiles[rel] {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			scanned++
			if loc := defenceLiteral.FindIndex(raw); loc != nil {
				hits = append(hits, rel+": "+string(raw[loc[0]:loc[1]]))
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return hits, scanned
}

func TestDefenceNamesAreDeclaredOnce(t *testing.T) {
	hits, scanned := scanForDefenceLiterals(t, repoRoot(t))
	if scanned < 500 {
		t.Fatalf("scanned only %d files; the walk is not looking at the repo", scanned)
	}
	for _, h := range hits {
		t.Errorf("defence name declared outside combatvocab: %s", h)
	}
}

// The guard must be able to fail. A copy of a real declaration line, planted
// in a temp tree with the same shape, must be found.
func TestDefenceNamesGuardIsNotVacuous(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "somewhere")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte("package somewhere\n\nconst DefenseDodge string = \"dodge\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	hits, _ := scanForDefenceLiterals(t, root)
	if len(hits) != 1 {
		t.Fatalf("planted declaration not found: hits = %v", hits)
	}
}
