package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// allConfigDataCallers lists repo-relative files outside internal/configs
// allowed to call a raw config reader, each with its reason. AllConfigData,
// DotPaths and GetOverrides return secrets raw; they exist because
// internal/configs builds the key and type lookups and the override file
// from them. Anything a person reads (logs, command output, web pages) goes
// through DisplayConfigData. A caller that needs only a key's TYPE may be
// added here with its reason. Empty on master: every outside caller was a
// display.
var allConfigDataCallers = map[string]string{}

// rawConfigSelectors are the Go selectors that return config values raw.
var rawConfigSelectors = map[string]bool{
	"AllConfigData": true,
	"DotPaths":      true,
	"GetOverrides":  true,
}

// rawConfigInTemplate matches a page template reaching raw config values:
// the three raw readers by name, the untyped Modules map through the page's
// CONFIG, or through the getconfig template func
// (internal/web/template_func.go) inside one {{ }} action.
var rawConfigInTemplate = regexp.MustCompile(`AllConfigData|DotPaths|GetOverrides|\.CONFIG\.Modules\b|\{\{[^}]*getconfig[^}]*\.Modules\b`)

// TestNoDisplayReadsRawConfig fails when a Go file outside internal/configs,
// or any page template, reads the raw config. Slice M found four displays
// (boot log, server set listing, server config menu, /viewconfig) printing
// the companion's APIKey raw because each read the raw view.
func TestNoDisplayReadsRawConfig(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	var offenders []string
	scanned := map[string]bool{}
	fset := token.NewFileSet()

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			// Dot-dirs include agent worktrees under .claude/, a second copy of the tree.
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return filepath.SkipDir
			}
			switch d.Name() {
			case "vendor", "node_modules", "docs", "testdata":
				return filepath.SkipDir
			}
			if rel == "internal/configs" {
				return filepath.SkipDir
			}
			return nil
		}
		if _, ok := allConfigDataCallers[rel]; ok {
			return nil
		}

		switch {
		case strings.HasSuffix(rel, ".go"):
			if strings.HasSuffix(rel, "_test.go") {
				return nil
			}
			file, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if perr != nil {
				// A syntax error is the compiler's to report.
				return nil
			}
			scanned[rel] = true
			ast.Inspect(file, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok && rawConfigSelectors[sel.Sel.Name] {
					offenders = append(offenders, rel+":"+strconv.Itoa(fset.Position(sel.Pos()).Line)+" "+sel.Sel.Name)
				}
				return true
			})
		case strings.HasSuffix(rel, ".html"), strings.HasSuffix(rel, ".template"), strings.HasSuffix(rel, ".tmpl"):
			body, rerr := os.ReadFile(path)
			if rerr != nil {
				return nil
			}
			scanned[rel] = true
			for _, m := range rawConfigInTemplate.FindAllString(string(body), -1) {
				offenders = append(offenders, rel+" "+m)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo: %v", err)
	}

	// Verify the negative: the walk must have read the files that held the
	// four displays, or an empty offender list proves nothing.
	for _, must := range []string{"main.go", "boot_config_log.go",
		"internal/usercommands/admin.server.go", "_datafiles/html/public/viewconfig.html"} {
		if !scanned[must] {
			t.Errorf("guard never scanned %s; the walk cannot see what it guards", must)
		}
	}

	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("%d raw config read(s) outside internal/configs. AllConfigData, DotPaths, GetOverrides\n"+
			"and the Modules map return secrets raw.\n"+
			"Use DisplayConfigData for anything a person reads, or add the file to\n"+
			"allConfigDataCallers in this file with the reason it needs only key types.\n  %s",
			len(offenders), strings.Join(offenders, "\n  "))
	}
}
