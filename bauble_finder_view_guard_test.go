package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// finderViewSites is every function allowed to read a bauble's text as one
// viewer sees it (items.Item GetSpecFor, DisplayNameFor, NameFor,
// LongDescriptionFor; baubles.Record MaterialFor), keyed "path|function"
// as lookupFuncName names it, each with why its output reaches that viewer
// alone. A finder-only bauble's own text (owner ruling 2026-09-29) is read
// through these and nowhere else: every viewer-agnostic accessor shows the
// generic trinket (internal/items
// TestFinderOnlyBaubleIsGenericToEveryoneButItsFinder), so a new render
// path that uses them cannot leak it, and this guard stops a new caller of
// the finder's view from reaching anyone else.
//
// Two rules, so neither a new function nor a new line inside a listed one
// slips past (review finding c):
//   - Each listed function makes exactly `calls` finder-view calls. A new
//     call in Look (which also talks to another player) changes the count
//     and fails until someone reads it and updates the row.
//   - No finder-view call may sit inside anything that sends beyond its one
//     reader: a room send (roomSendSelectors), a mob's Command (a `say`),
//     merchantSay, an events.Message literal, or a SendText whose receiver
//     is not provably the viewer (the receiver's root identifier must be
//     the viewer argument's: user.SendText(..., x.NameFor(user.UserId))).
//
// KNOWN LIMITS (each is a way a real leak could pass):
//   - A finder-view value kept in a variable and later sent is not traced;
//     the call count makes every such variable a reviewed line.
//   - The five method names are matched by name, not by type, and only
//     inside function bodies (not package-level initialisers).
//   - internal/items and internal/baubles define the accessors and are not
//     scanned.
type finderViewSite struct {
	calls int
	why   string
}

var finderViewSites = map[string]finderViewSite{
	"internal/actions/search_bauble.go|BaubleDelivery.deliver": {1, "the find's own lines, sent to the finder alone (who.send); the room line names no item"},
	"internal/actions/steal.go|takeFromMob":                    {3, "the thief's own success line (actor.SendText); the room is not told what was taken"},
	"internal/usercommands/appraise.go|appraiseBauble":         {4, "the appraisal, sent to the player who asked for it (user.SendText); the room line names no item"},
	"internal/usercommands/inventory.go|Inventory":             {2, "the player's own inventory listing"},
	"internal/usercommands/look.go|Look":                       {4, "what the looker reads about an item they carry or one on the floor; the room lines beside them keep DisplayName"},
	"internal/usercommands/look.go|lookRoom":                   {2, "the looker's own view of the room's floor and their own stash"},
	"modules/gmcp/gmcp.Char.go|GMCPCharModule.GetCharNode":     {1, "the player's own Char.Inventory backpack"},
	"modules/gmcp/gmcp.Char.go|buildBandolierContainer":        {1, "the player's own bandolier payload (GetCharNode passes user.UserId)"},
	"modules/gmcp/gmcp.Char.go|buildComponentBagContainer":     {1, "the player's own component bag payload (GetCharNode passes user.UserId)"},
	"modules/gmcp/gmcp.Room.go|GMCPRoomModule.GetRoomNode":     {1, "Room.Info.Contents.Items, built for one user and sent to that user"},
}

// finderViewSelectors are the viewer-aware accessors.
var finderViewSelectors = map[string]bool{
	"GetSpecFor": true, "DisplayNameFor": true, "NameFor": true, "LongDescriptionFor": true, "MaterialFor": true,
}

// beyondReaderCalls send text beyond one reader, by callee name, whatever
// the receiver: the room sends, a mob's Command (a `say`), merchantSay.
var beyondReaderCalls = map[string]bool{
	"SendTextCommunication": true, "SendTextVisual": true, "SendTextVisualHidingNames": true,
	"SendTextVisualAsLit": true, "SendTextVisualAsLitHidingNames": true, "SendTextVisualWithAudio": true,
	"SendTextToExits": true, "SendRoomCommunication": true, "SendTrio": true, "Command": true, "merchantSay": true,
}

// rootIdent is the identifier an expression hangs from: user for
// user.UserId, actor for actor.GetUserId(), "" when there is none.
func rootIdent(e ast.Expr) string {
	for {
		switch x := e.(type) {
		case *ast.Ident:
			return x.Name
		case *ast.SelectorExpr:
			e = x.X
		case *ast.CallExpr:
			e = x.Fun
		case *ast.ParenExpr:
			e = x.X
		case *ast.StarExpr:
			e = x.X
		default:
			return ""
		}
	}
}

// calleeName is a call's function name: Sel for a selector, the identifier
// for a plain call.
func calleeName(call *ast.CallExpr) (string, *ast.SelectorExpr) {
	switch f := call.Fun.(type) {
	case *ast.SelectorExpr:
		return f.Sel.Name, f
	case *ast.Ident:
		return f.Name, nil
	}
	return "", nil
}

// finderViewCallsIn returns every finder-view call within n.
func finderViewCallsIn(n ast.Node) []*ast.CallExpr {
	var out []*ast.CallExpr
	ast.Inspect(n, func(m ast.Node) bool {
		if c, ok := m.(*ast.CallExpr); ok {
			if name, _ := calleeName(c); finderViewSelectors[name] {
				out = append(out, c)
			}
		}
		return true
	})
	return out
}

type finderViewFile struct {
	rel  string
	file *ast.File
}

// scanFinderView returns how many finder-view calls each function makes,
// and every such call that sits inside a send beyond its one reader.
func scanFinderView(fset *token.FileSet, files []finderViewFile) (calls map[string]int, leaks []string) {
	calls = map[string]int{}
	for _, f := range files {
		for _, decl := range f.file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			key := f.rel + "|" + lookupFuncName(fd)
			report := func(c *ast.CallExpr, via string) {
				name, _ := calleeName(c)
				leaks = append(leaks, fmt.Sprintf("%s:%d in %s: %s inside %s",
					f.rel, fset.Position(c.Pos()).Line, key, name, via))
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.CompositeLit:
					// events.Message{UserId: ..., Text: ...}: queued to some
					// user, not provably the viewer.
					if sel, ok := x.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Message" {
						for _, c := range finderViewCallsIn(x) {
							report(c, "events.Message")
						}
					}
				case *ast.CallExpr:
					name, sel := calleeName(x)
					if finderViewSelectors[name] {
						calls[key]++
					}
					switch {
					case beyondReaderCalls[name]:
						for _, arg := range x.Args {
							for _, c := range finderViewCallsIn(arg) {
								report(c, name)
							}
						}
					case name == "SendText" && sel != nil:
						receiver := rootIdent(sel.X)
						if inner, ok := sel.X.(*ast.CallExpr); ok {
							if n, _ := calleeName(inner); n == "GetRoom" {
								receiver = "" // a room reached through its reader is still a room
							}
						}
						if s, ok := sel.X.(*ast.SelectorExpr); ok && strings.HasSuffix(strings.ToLower(s.Sel.Name), "room") {
							receiver = "" // who.room.SendText, any xxxRoom field
						}
						if id, ok := sel.X.(*ast.Ident); ok && strings.HasSuffix(strings.ToLower(id.Name), "room") {
							receiver = "" // room, fromRoom, markRoom: a room variable
						}
						for _, arg := range x.Args {
							for _, c := range finderViewCallsIn(arg) {
								if len(c.Args) == 0 || receiver == "" || rootIdent(c.Args[0]) != receiver {
									report(c, receiver+".SendText")
								}
							}
						}
					}
				}
				return true
			})
		}
	}
	return calls, leaks
}

// TestFinderViewReachesOnlyItsReader fails when a function not in
// finderViewSites reads a bauble as one viewer sees it, when a listed
// function no longer does (stale), or when any finder-view call sits inside
// a room send's arguments. If you are here for a new single-reader site
// (the output goes to one player, and only them), add it with the reason.
// If the output reaches anyone else, use the viewer-agnostic accessor,
// which shows a finder-only bauble as the generic trinket.
func TestFinderViewReachesOnlyItsReader(t *testing.T) {
	fset := token.NewFileSet()
	var files []finderViewFile
	scanned := map[string]bool{}
	for _, root := range []string{"internal", "modules"} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			rel := filepath.ToSlash(path)
			if d.IsDir() {
				if rel == "internal/items" || rel == "internal/baubles" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
				return nil
			}
			file, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
			if perr != nil {
				return nil
			}
			scanned[rel] = true
			files = append(files, finderViewFile{rel: rel, file: file})
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s (test must run from the repo root): %v", root, err)
		}
	}
	for _, must := range []string{"internal/usercommands/look.go", "internal/usercommands/get.go", "modules/gmcp/gmcp.Char.go", "modules/auctions/auctions.go"} {
		if !scanned[must] {
			t.Fatalf("the walk never read %s: it cannot see what it guards", must)
		}
	}

	calls, leaks := scanFinderView(fset, files)
	var problems []string
	for key, got := range calls {
		want, ok := finderViewSites[key]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("%s: makes %d finder-view call(s), and is not in finderViewSites", key, got))
		case got != want.calls:
			problems = append(problems, fmt.Sprintf("%s: makes %d finder-view call(s), finderViewSites says %d: read each one before changing the row", key, got, want.calls))
		}
	}
	for key := range finderViewSites {
		if calls[key] == 0 {
			problems = append(problems, key+": in finderViewSites but reads no finder view (stale)")
		}
	}
	problems = append(problems, leaks...)
	sort.Strings(problems)
	if len(problems) > 0 {
		t.Errorf("%d finder-view problem(s):\n  %s", len(problems), strings.Join(problems, "\n  "))
	}
}

// TestFinderViewGuardCatchesALeak proves the scan can fail on every shape
// it claims to catch, and passes the one private shape.
func TestFinderViewGuardCatchesALeak(t *testing.T) {
	fset := token.NewFileSet()
	src := `package probe

func Shout(room *rooms.Room, itm items.Item, uid int) {
	room.SendTextVisual(1, fmt.Sprintf("%s", itm.DisplayNameFor(uid)))
}

func Mine(user *users.UserRecord, itm items.Item) { user.SendText(1, itm.NameFor(user.UserId)) }

func ToAnother(user *users.UserRecord, u *users.UserRecord, itm items.Item) {
	u.SendText(1, itm.NameFor(user.UserId))
}

func Chained(user *users.UserRecord, itm items.Item) { user.GetRoom().SendText(1, itm.NameFor(user.UserId)) }

func Says(room *rooms.Room, mob *mobs.Mob, itm items.Item, uid int) {
	merchantSay(room, mob, itm.NameFor(uid))
	mob.Command("say " + itm.NameFor(uid))
}

func Queued(itm items.Item, uid int) {
	events.AddToQueue(events.Message{UserId: 2, Text: itm.NameFor(uid)})
}
`
	file, err := parser.ParseFile(fset, "probe.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	calls, leaks := scanFinderView(fset, []finderViewFile{{rel: "probe.go", file: file}})
	if calls["probe.go|Says"] != 2 || calls["probe.go|Mine"] != 1 || len(calls) != 6 {
		t.Fatalf("every call counted, per function: %v", calls)
	}
	want := []string{"Shout", "ToAnother", "Chained", "Says", "Says", "Queued"}
	if len(leaks) != len(want) {
		t.Fatalf("want %d leaks (%v), got:\n  %s", len(want), want, strings.Join(leaks, "\n  "))
	}
	for _, l := range leaks {
		if strings.Contains(l, "|Mine:") || strings.Contains(l, " in probe.go|Mine") {
			t.Fatalf("the player's own SendText is private: %v", l)
		}
	}
}
