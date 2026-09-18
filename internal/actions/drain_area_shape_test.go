package actions

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// core-drain (spell) resolves through ExecuteDrainArea, which used to
// hardcode the melee set so a construct's room-wide drain could be PARRIED.
// Owner ruling (M4 spec 9, carried to M4b-2): it is a physical spell, so its
// victims dodge or block. Pinned at the source: the loop's Shape must be
// Spell(DamagePhysical, TargetArea), and no Melee(...) may remain in
// ExecuteDrainArea.
func TestExecuteDrainAreaIsAPhysicalAreaSpell(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(filepath.Dir(here), "combat_drain.go"), nil, 0)
	require.NoError(t, err)

	var fn *ast.FuncDecl
	for _, d := range parsed.Decls {
		if f, ok := d.(*ast.FuncDecl); ok && f.Name.Name == "ExecuteDrainArea" {
			fn = f
		}
	}
	require.NotNil(t, fn, "ExecuteDrainArea not found")

	spellPhysicalArea, melee := 0, 0
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "combatvocab" {
			return true
		}
		switch sel.Sel.Name {
		case "Melee":
			melee++
		case "Spell":
			if len(call.Args) == 2 && isSel(call.Args[0], "DamagePhysical") && isSel(call.Args[1], "TargetArea") {
				spellPhysicalArea++
			}
		}
		return true
	})
	require.Zero(t, melee, "ExecuteDrainArea still builds a melee shape; core-drain must not be parryable")
	require.GreaterOrEqual(t, spellPhysicalArea, 1, "ExecuteDrainArea must carry Spell(DamagePhysical, TargetArea) at least once; the zero-Melee assertion above is the real invariant")
}

func isSel(e ast.Expr, name string) bool {
	s, ok := e.(*ast.SelectorExpr)
	return ok && s.Sel.Name == name
}
