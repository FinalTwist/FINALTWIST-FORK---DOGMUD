package templates

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"text/template"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/combatvocab"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// The directive as it appears in help/spell.template. Rendering it here catches
// the two ways the fix could fail silently: DefenceNames not deriving the
// right defence set from the axes, and a non-harm cast printing an empty
// Resisted-by row instead of omitting the line.
func TestSpellTemplateDefenceRow(t *testing.T) {
	const fragment = `{{ with .DefenceNames -}}` +
		"Resisted by: {{ . }}\n" +
		`{{ end -}}`

	tmpl, err := template.New("frag").Funcs(funcMap).Parse(fragment)
	if err != nil {
		t.Fatalf("parsing the spell.template defence row: %v", err)
	}

	for _, tc := range []struct {
		name string
		sd   *spells.SpellData
		want string
	}{
		{"social", &spells.SpellData{AttackType: combatvocab.AttackSpell, DamageType: combatvocab.DamageSocial, Targeting: combatvocab.TargetSingle}, "Resisted by: defy\n"},
		{"mental", &spells.SpellData{AttackType: combatvocab.AttackSpell, DamageType: combatvocab.DamageMental, Targeting: combatvocab.TargetSingle}, "Resisted by: quell\n"},
		{"physical", &spells.SpellData{AttackType: combatvocab.AttackSpell, DamageType: combatvocab.DamagePhysical, Targeting: combatvocab.TargetSingle}, "Resisted by: dodge or block\n"},
		{"non-harm", &spells.SpellData{AttackType: combatvocab.AttackNone, DamageType: combatvocab.DamageNonHarm, Targeting: combatvocab.TargetSingle}, ""},
	} {
		var sb strings.Builder
		if err := tmpl.Execute(&sb, tc.sd); err != nil {
			t.Fatalf("executing %s: %v", tc.name, err)
		}
		if got := sb.String(); got != tc.want {
			t.Errorf("defence row for %s = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// The spell helpfile's three reference rows used to print raw numbers at the
// player: "Base Folds: 36", "Conv. Cost: 120", "Wait Time: 0 rounds". The last
// one also read like a placeholder.
//
// Each func DELEGATES to internal/combat rather than restating the bands, so
// this test's real job is to prove the delegation is wired: if a func were
// missing from the map, Process would fail at render time on a page no other
// test renders, and if one were reimplemented here it would drift from the
// `spells` command silently.
func TestSpellTemplateReferenceRows(t *testing.T) {
	const fragment = "Cast Time: {{ castlength .BaseFolds }}\n" +
		"Conv. Cost: {{ convictioncost .Cost }}\n" +
		"Recovery: {{ waittime .WaitRounds }}\n"

	tmpl, err := template.New("rows").Funcs(funcMap).Parse(fragment)
	if err != nil {
		t.Fatalf("parsing the spell.template reference rows: %v", err)
	}

	var sb strings.Builder
	// Charm's real values.
	data := struct {
		BaseFolds  int
		Cost       int
		WaitRounds int
	}{BaseFolds: 36, Cost: 120, WaitRounds: 0}
	if err := tmpl.Execute(&sb, data); err != nil {
		t.Fatalf("executing: %v", err)
	}

	got := sb.String()
	want := "Cast Time: very long\nConv. Cost: demanding\nRecovery: instant\n"
	if got != want {
		t.Errorf("reference rows =\n%q\nwant\n%q", got, want)
	}

	// No digit may survive into what the player reads.
	for _, r := range got {
		if r >= '0' && r <= '9' {
			t.Errorf("a raw number reached the player in %q", got)
			break
		}
	}
}

// The vocabulary must match the `spells` command, which renders cost with the
// same helper. Two vocabularies for one value is the drift this delegation
// exists to prevent.
func TestSpellTemplateCostMatchesSpellsCommand(t *testing.T) {
	fn, ok := funcMap["convictioncost"].(func(int) string)
	if !ok {
		t.Fatal("convictioncost is not registered as func(int) string")
	}
	if got, want := fn(120), combat.GetConvictionCostDescription(120); got != want {
		t.Errorf("helpfile says %q where the spells command says %q", got, want)
	}
}

// Parse the REAL template files, not a retyped fragment. Go templates resolve
// function names at parse time, and help templates are parsed lazily on first
// use -- so a misspelled func name here would not fail the build, would not fail
// a boot test, and would surface as "[TEMPLATE ERROR]" the first time a player
// typed `help spell <anything>`.
func TestSpellTemplatesParseWithTheRealFuncMap(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	// Both worlds: dogmud is live (FilePaths.DataFiles), default ships alongside
	// it and uses the same func map.
	for _, world := range []string{"dogmud", "default"} {
		path := filepath.Join(repoRoot, "_datafiles", "world", world,
			"templates", "help", "spell.template")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		if _, err := template.New("spell").Funcs(funcMap).Parse(string(raw)); err != nil {
			t.Errorf("%s does not parse against the live func map: %v", world, err)
		}
		if bytes.Contains(raw, []byte("{{ .BaseFolds }}")) ||
			bytes.Contains(raw, []byte("{{ .Cost }}")) ||
			bytes.Contains(raw, []byte("{{ .WaitRounds }}")) {
			t.Errorf("%s still prints a raw spell number at the player", world)
		}
	}
}
