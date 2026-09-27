package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Lighting plan 5b ruling (owner 2026-09-26): every opposed or difficulty
// roll pays the sight penalty; only voice contests are exempt. This file is
// the recurrence guard for that ruling. It does not duplicate
// internal/combat/contest_site_guard_test.go, which asks "does every contest
// site have an OWNER"; this one asks "does every roll site apply the SIGHT
// RAMP".

// guardedSightFuncs are the roll entry points, keyed by the package that
// defines them. A production call to one of these must sit in a function
// that also applies a sightPenaltyHelpers call, or be exempt by file|func.
//
// A call is seen three ways: as a package-qualified selector (combat.RunContest)
// anywhere, as a bare identifier (RunContest) inside the defining package, and
// through an ALIAS: a variable assigned a guarded function value
// (var runSpellChannelAttack = combat.ResolveChannelAttack) is guarded too,
// within the package that declares it. A guarded function passed as an
// argument (processGrapplePair hands combat.RunContest to its runner) counts as
// a site in the function that passes it.
//
// combat.AttemptGrapple and combat.RollSubmissionAttempt are deliberately NOT
// listed: each takes a room and applies messaging.SightMult to both sides
// itself, and the bare RunContest inside each is already guarded here, so
// their callers owe nothing and guarding them would only demand exemptions.
var guardedSightFuncs = map[string]map[string]bool{
	"combat":   {"RunContest": true, "ResolveChannelAttack": true, "RunConcentrationContest": true},
	"contest":  {"AgainstDifficulty": true},
	"crafting": {"RunCraftContest": true, "RunSalvageContest": true, "RollSalvageReturns": true, "RollSalvageReturnsFromSpec": true},
	"forager":  {"ForageCore": true},
}

// sightPenaltyHelpers are the names a complying function calls. The match is on
// the call's final name (selector or bare identifier), so
// messaging.SightMult and a same-package stealVictimScore both count.
var sightPenaltyHelpers = map[string]bool{
	"SightMult":              true,
	"SightScoreMultiplier":   true,
	"ComfortDistance":        true,
	"SituationalAttackMult":  true,
	"SituationalDefenceMult": true,
	"CalcDetectionScore":     true,
	"stealVictimScore":       true,
}

// sightExemptSites are the roll sites whose multiply happens somewhere the
// function-body check cannot see, keyed "file|func" (never by directory). Each
// reason names who multiplies, and that it happens exactly once per path.
//
// The steal and plant helpers (stealFromMob and kin) need no row: the thief's
// score is multiplied once in Steal / Plant, and each helper itself calls
// stealVictimScore or CalcDetectionScore for the other side, so the body
// check passes them. The stale-row check below keeps this list honest.
var sightExemptSites = map[string]string{
	// Salvage: the score handed to the crafting rollers is multiplied once in
	// Salvage before it dispatches.
	"internal/actions/salvage.go|salvageCorpse": "score multiplied once in Salvage before dispatch",
	"internal/actions/salvage.go|salvageItem":   "score multiplied once in Salvage before dispatch",
	// The rollers themselves take an already-multiplied score from their
	// callers (salvage.go above), so the RunSalvageContest inside them is pure.
	"internal/crafting/salvage.go|RollSalvageReturns":         "pure roller: callers pass a score already multiplied (actions.Salvage)",
	"internal/crafting/salvage.go|RollSalvageReturnsFromSpec": "pure roller: callers pass a score already multiplied (actions.Salvage)",
	// Track: the trail-read bands stay pure for their band tests; Track
	// multiplies the search score once before calling in.
	"internal/actions/track.go|resolveTrailDetail": "pure band ladder: Track multiplies the search score once before calling",
	// Forage: ForageCore is pure; its one caller (actions.Forage) multiplies
	// the SearchScore once before building the attempt.
	"internal/forager/forage_core.go|ForageCore": "pure core: actions.Forage multiplies SearchScore once before calling",
	// Rhetoric is voice, exempt by ruling.
	"internal/actions/combat_counter.go|executeCounterTaunt": "rhetoric (voice) is exempt by ruling, owner 2026-09-26",
	// Runner seams: these functions only hand RunContest (or its alias) to a
	// *WithRunner body that multiplies the defence side itself.
	"internal/combat/defence_multiplier.go|ResolveChannelAttack":      "the channel funnel: resolveChannelAttackWithRunner applies SituationalDefenceMult; callers fold SituationalAttackMult into AttackSide.Mult",
	"internal/combat/skill_moves.go|ExecuteSkillMove":                 "passes the runner to executeSkillMoveWithRunner (defence via the channel seam, knockdown via SituationalDefenceMult); callers fold SituationalAttackMult into SkillMoveParams.Mult",
	"internal/combat/combat_helpers.go|runBestOfAllDefense":           "melee: runBestOfAllDefenseWithRunner applies SightScoreMultiplier to the defence; calcAttackScore applies it to the attack",
	"internal/hooks/Position_GrappleTick.go|processGrapplePair":       "grapple drift: processGrapplePairWithContest applies SightMult to both sides once",
	"internal/hooks/spell_resolution.go|resolveAgainstMob":            "spell: side built once by spellAttackSideFor (SituationalAttackMult) in resolveSpell; defence inside the channel seam",
	"internal/hooks/spell_resolution.go|resolveAgainstPlayer":         "spell: side built once by spellAttackSideFor (SituationalAttackMult) in resolveSpell; defence inside the channel seam",
	"internal/hooks/spell_resolution.go|resolveMobSpellAgainstMob":    "mob spell: side built once by spellAttackSideFor (SituationalAttackMult) in resolveMobSpell; defence inside the channel seam",
	"internal/hooks/spell_resolution.go|resolveMobSpellAgainstPlayer": "mob spell: side built once by spellAttackSideFor (SituationalAttackMult) in resolveMobSpell; defence inside the channel seam",
}

// sightRuling is appended to every failure.
const sightRuling = "every opposed or difficulty roll pays the sight penalty; only voice contests are exempt; owner 2026-09-26"

// sightSite is one guarded reference found by the checker.
type sightSite struct {
	File string // repo-relative, slash-separated
	Func string // enclosing FuncDecl name, or "var <name>" at package level
	Call string // what was referenced, e.g. combat.RunContest
	Line int
}

func (s sightSite) key() string { return s.File + "|" + s.Func }

// sightFile is one parsed production file with its repo-relative path.
type sightFile struct {
	Rel  string
	File *ast.File
}

// guardedRefName reports the guarded name an expression refers to, as it
// should appear in a failure, using the file's own package name to recognise
// bare in-package identifiers and aliases.
func guardedRefName(expr ast.Expr, filePkg string, aliases map[string]string) (string, bool) {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		pkg, ok := e.X.(*ast.Ident)
		if !ok {
			return "", false
		}
		if guardedSightFuncs[pkg.Name][e.Sel.Name] {
			return pkg.Name + "." + e.Sel.Name, true
		}
	case *ast.Ident:
		if guardedSightFuncs[filePkg][e.Name] {
			return filePkg + "." + e.Name, true
		}
		if target, ok := aliases[e.Name]; ok {
			return e.Name + " (alias of " + target + ")", true
		}
	}
	return "", false
}

// collectSightAliases returns, per package directory, the variables assigned
// a guarded function value: name -> the guarded function it aliases.
func collectSightAliases(files []sightFile) map[string]map[string]string {
	out := map[string]map[string]string{}
	add := func(dir, name, target string) {
		if out[dir] == nil {
			out[dir] = map[string]string{}
		}
		out[dir][name] = target
	}
	for _, f := range files {
		dir := filepath.ToSlash(filepath.Dir(f.Rel))
		pkg := f.File.Name.Name
		ast.Inspect(f.File, func(n ast.Node) bool {
			switch s := n.(type) {
			case *ast.ValueSpec:
				for i, v := range s.Values {
					if i < len(s.Names) {
						if target, ok := guardedRefName(v, pkg, nil); ok {
							add(dir, s.Names[i].Name, target)
						}
					}
				}
			case *ast.AssignStmt:
				for i, v := range s.Rhs {
					if i >= len(s.Lhs) {
						break
					}
					id, ok := s.Lhs[i].(*ast.Ident)
					if !ok {
						continue
					}
					if target, ok := guardedRefName(v, pkg, nil); ok {
						add(dir, id.Name, target)
					}
				}
			}
			return true
		})
	}
	return out
}

// callsSightHelper reports whether a node contains a call to any
// sightPenaltyHelpers name.
func callsSightHelper(n ast.Node) bool {
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.SelectorExpr:
			found = sightPenaltyHelpers[fn.Sel.Name]
		case *ast.Ident:
			found = sightPenaltyHelpers[fn.Name]
		}
		return true
	})
	return found
}

// findUnpenalisedRollSites is the checker. It returns every guarded roll site
// whose enclosing function applies no sight helper and whose file|func is not
// in exempt. It takes parsed files so a test can feed it source strings.
func findUnpenalisedRollSites(fset *token.FileSet, files []sightFile, exempt map[string]string) []sightSite {
	aliases := collectSightAliases(files)
	var out []sightSite

	for _, f := range files {
		dir := filepath.ToSlash(filepath.Dir(f.Rel))
		pkg := f.File.Name.Name
		pkgAliases := aliases[dir]

		check := func(owner string, body ast.Node, complies bool) {
			ast.Inspect(body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				// The callee, and any guarded function passed as a value.
				refs := append([]ast.Expr{call.Fun}, call.Args...)
				for _, ref := range refs {
					name, ok := guardedRefName(ref, pkg, pkgAliases)
					if !ok {
						continue
					}
					site := sightSite{File: f.Rel, Func: owner, Call: name, Line: fset.Position(ref.Pos()).Line}
					if complies {
						continue
					}
					if _, ok := exempt[site.key()]; ok {
						continue
					}
					out = append(out, site)
				}
				return true
			})
		}

		for _, decl := range f.File.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Body == nil {
					continue
				}
				check(d.Name.Name, d.Body, callsSightHelper(d.Body))
			case *ast.GenDecl:
				// A package-level initialiser that CALLS a guarded roll has no
				// function to comply in; it is always a finding. An alias
				// declaration is not a call and is not reported here.
				for _, spec := range d.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, v := range vs.Values {
						owner := "var"
						if len(vs.Names) > 0 {
							owner = "var " + vs.Names[0].Name
						}
						check(owner, v, false)
					}
				}
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

// parseProductionGoFiles walks the repo exactly as contest_floor_guard_test.go
// does (dot-directories, vendor, tools, docs and _datafiles skipped; test
// files skipped) and parses every production Go file.
func parseProductionGoFiles(t *testing.T) (*token.FileSet, []sightFile) {
	t.Helper()
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	fset := token.NewFileSet()
	var files []sightFile
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return filepath.SkipDir
			}
			switch d.Name() {
			case "vendor", "node_modules", "bin", "_datafiles", "docs", "tools":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return nil
		}
		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return nil
		}
		files = append(files, sightFile{Rel: filepath.ToSlash(rel), File: file})
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo: %v", err)
	}
	return fset, files
}

func formatSightSites(sites []sightSite) string {
	lines := make([]string, 0, len(sites))
	for _, s := range sites {
		lines = append(lines, s.File+":"+strconv.Itoa(s.Line)+" in "+s.Func+": "+s.Call)
	}
	return strings.Join(lines, "\n  ")
}

// TestEveryRollSiteAppliesTheSightPenalty fails when production code rolls
// through a guarded entry point in a function that applies no sight penalty
// and is not exempt.
//
// KNOWN BLIND SPOT: a function-typed PARAMETER (runner(...) inside
// resolveChannelAttackWithRunner) is invisible; the value is seen where it is
// passed instead. Tools under tools/ are not scanned, matching the floor guard.
func TestEveryRollSiteAppliesTheSightPenalty(t *testing.T) {
	fset, files := parseProductionGoFiles(t)
	if len(files) < 100 {
		t.Fatalf("walk found only %d production files; the guard could not have found anything", len(files))
	}

	offenders := findUnpenalisedRollSites(fset, files, sightExemptSites)
	if len(offenders) > 0 {
		t.Errorf("roll sites with no sight penalty:\n  %s\n\n"+
			"Comply one of two ways: multiply the score in this function through "+
			"messaging.SightMult / messaging.SightScoreMultiplier / "+
			"messaging.ComfortDistance / combat.SituationalAttackMult / "+
			"combat.SituationalDefenceMult / actions.CalcDetectionScore / "+
			"stealVictimScore; or, if a CALLER multiplies exactly once on every "+
			"path, add \"file|func\" to sightExemptSites with a reason naming that "+
			"caller. Ruling: %s.",
			formatSightSites(offenders), sightRuling)
	}

	// Every exemption must still name a real roll site; a stale row would
	// silently pre-exempt whatever function later takes that name.
	all := findUnpenalisedRollSites(fset, files, nil)
	live := map[string]bool{}
	for _, s := range all {
		live[s.key()] = true
	}
	var stale []string
	for key := range sightExemptSites {
		if !live[key] {
			stale = append(stale, key)
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("sightExemptSites rows that no longer name an unpenalised roll site (delete them):\n  %s",
			strings.Join(stale, "\n  "))
	}
}

func parseSightSource(t *testing.T, fset *token.FileSet, rel, src string) sightFile {
	t.Helper()
	f, err := parser.ParseFile(fset, rel, src, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", rel, err)
	}
	return sightFile{Rel: rel, File: f}
}

// TestSightPenaltyGuardCatchesAnOmission proves the checker can fail: a
// combat.RunContest call in a function with no helper is reported, and the
// same call beside a helper is not.
func TestSightPenaltyGuardCatchesAnOmission(t *testing.T) {
	fset := token.NewFileSet()
	src := `package actions

func Bare(a, b float64) bool {
	return combat.RunContest(a, nil).Success
}

func Lit(c, room interface{}, a float64) bool {
	a *= messaging.SightMult(c, room)
	return combat.RunContest(a, nil).Success
}

func Passed() {
	run(combat.RunContest)
}
`
	files := []sightFile{parseSightSource(t, fset, "internal/actions/sabotage.go", src)}
	got := findUnpenalisedRollSites(fset, files, nil)
	if len(got) != 2 || got[0].Func != "Bare" || got[1].Func != "Passed" {
		t.Fatalf("want Bare and Passed reported, got:\n  %s", formatSightSites(got))
	}
	if got[0].Call != "combat.RunContest" {
		t.Errorf("finding names %q, want combat.RunContest", got[0].Call)
	}

	// Exempting by file|func silences exactly that site.
	got = findUnpenalisedRollSites(fset, files, map[string]string{"internal/actions/sabotage.go|Bare": "test"})
	if len(got) != 1 || got[0].Func != "Passed" {
		t.Errorf("exempting Bare should leave only Passed, got:\n  %s", formatSightSites(got))
	}

	// A bare call inside the DEFINING package is seen too.
	inPkg := parseSightSource(t, fset, "internal/combat/sabotage.go", `package combat

func Flee(a float64) bool { return RunContest(a, nil).Success }
`)
	got = findUnpenalisedRollSites(fset, []sightFile{inPkg}, nil)
	if len(got) != 1 || got[0].Func != "Flee" {
		t.Errorf("bare in-package RunContest not reported, got:\n  %s", formatSightSites(got))
	}
}

// TestSightPenaltyGuardSeesTheSpellAlias proves an alias declared from a
// guarded function (the spell path's runSpellChannelAttack shape) is guarded
// within its package, and not outside it.
func TestSightPenaltyGuardSeesTheSpellAlias(t *testing.T) {
	fset := token.NewFileSet()
	decl := parseSightSource(t, fset, "internal/hooks/alias.go", `package hooks

var runSpell = combat.ResolveChannelAttack

func init() { runLater = combat.RunContest }
`)
	use := parseSightSource(t, fset, "internal/hooks/cast.go", `package hooks

func castBare() { runSpell(nil, nil, nil, nil, nil) }

func castLit(c, room interface{}) {
	_ = combat.SituationalAttackMult(c, room, nil)
	runSpell(nil, nil, nil, nil, nil)
}

func later() { runLater(0, nil) }
`)
	elsewhere := parseSightSource(t, fset, "internal/actions/other.go", `package actions

func unrelated() { runSpell() }
`)
	got := findUnpenalisedRollSites(fset, []sightFile{decl, use, elsewhere}, nil)
	if len(got) != 2 || got[0].Func != "castBare" || got[1].Func != "later" {
		t.Fatalf("want castBare and later reported, got:\n  %s", formatSightSites(got))
	}
	if !strings.Contains(got[0].Call, "alias of combat.ResolveChannelAttack") {
		t.Errorf("finding should name the alias target, got %q", got[0].Call)
	}
}

// findSkillMovesWithoutRoom returns file:line for every SkillMoveParams (or
// combat.SkillMoveParams) composite literal that does not set Room. A missing
// Room silently means "comfortable" for the defender.
func findSkillMovesWithoutRoom(fset *token.FileSet, files []sightFile) []string {
	var out []string
	for _, f := range files {
		ast.Inspect(f.File, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			switch typ := lit.Type.(type) {
			case *ast.Ident:
				if typ.Name != "SkillMoveParams" {
					return true
				}
			case *ast.SelectorExpr:
				if typ.Sel.Name != "SkillMoveParams" {
					return true
				}
			default:
				return true
			}
			for _, elt := range lit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Room" {
					return true
				}
			}
			out = append(out, f.Rel+":"+strconv.Itoa(fset.Position(lit.Pos()).Line))
			return true
		})
	}
	sort.Strings(out)
	return out
}

// TestEverySkillMoveSetsItsRoom fails when a production SkillMoveParams
// literal omits Room.
func TestEverySkillMoveSetsItsRoom(t *testing.T) {
	fset, files := parseProductionGoFiles(t)
	if bad := findSkillMovesWithoutRoom(fset, files); len(bad) > 0 {
		t.Errorf("SkillMoveParams literals without Room (the defender would read as comfortable):\n  %s\n\nSet Room. Ruling: %s.",
			strings.Join(bad, "\n  "), sightRuling)
	}
}

// TestSkillMoveRoomGuardCatchesAnOmission proves the Room check can fail.
func TestSkillMoveRoomGuardCatchesAnOmission(t *testing.T) {
	fset := token.NewFileSet()
	f := parseSightSource(t, fset, "internal/actions/kick.go", `package actions

func kick() {
	combat.ExecuteSkillMove(combat.SkillMoveParams{Attacker: a})
	combat.ExecuteSkillMove(combat.SkillMoveParams{Attacker: a, Room: r})
	_ = &SkillMoveParams{}
}
`)
	got := findSkillMovesWithoutRoom(fset, []sightFile{f})
	want := []string{"internal/actions/kick.go:4", "internal/actions/kick.go:6"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", got, want)
	}
}
