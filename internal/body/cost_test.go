package body_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

/*
	What a call costs, and the shape that made it expensive

internal/engine annotates a declaration with //sp:cost path, and says whether it
takes a limit. A signature that does not say a call is a full path build is a
signature that reads as free, which is mvm-z83.30: three closed bugs were pure
native semantics and nothing in the type said so.

A route is a NavAreaBuildPath. It walks the mesh until it reaches the goal or
runs out of areas, so an unreachable goal costs the whole mesh. mvm-cf3 is the
1833 ms frame that came of it, and a core read on 2026-09-06 named
CTFBotLocomotion::IsAreaTraversable under NavAreaBuildPath with maxPathLength 0.

Two rules, and each is the shape rather than the call:

  - a bounded build is given its limit. A literal zero is no limit at all, and
    PathLengthCap is what the switch reads.
  - an unbounded build is not made inside a loop, because that is one whole mesh
    walk per candidate in one frame.

Either is allowed with //sp:unbounded <reason> on the line above, so an
exception is argued rather than accidental.
*/

// costDecl finds an annotated declaration and the Go name under it.
var (
	costDecl = regexp.MustCompile(`//sp:cost path (bounded|unbounded)\n(?://[^\n]*\n)*func (?:\([^)]*\) )?([A-Z]\w*)`)
	allowed  = regexp.MustCompile(`//sp:unbounded `)
)

// pathBuilders is every function internal/engine says builds a route, by name,
// with whether it takes a limit.
func pathBuilders(t *testing.T) map[string]string {
	t.Helper()

	out := map[string]string{}
	entries, err := os.ReadDir("../engine")
	if err != nil {
		t.Fatalf("reading internal/engine: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		source, err := os.ReadFile(filepath.Join("../engine", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range costDecl.FindAllStringSubmatch(string(source), -1) {
			out[m[2]] = m[1]
		}
	}
	if len(out) == 0 {
		t.Fatal("no //sp:cost path declaration in internal/engine, so this test proves nothing")
	}
	return out
}

// callName is the function a call expression names, ignoring the receiver.
func callName(call *ast.CallExpr) string {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fun.Sel.Name
	case *ast.Ident:
		return fun.Name
	}
	return ""
}

// TestEveryPathBuildIsBoundedOrArgued walks the bodies and the behaviours.
func TestEveryPathBuildIsBoundedOrArgued(t *testing.T) {
	builders := pathBuilders(t)

	for _, dir := range []string{"../body", "../action"} {
		walkGo(t, dir, func(path string, file *ast.File, fset *token.FileSet, source string) {
			lines := strings.Split(source, "\n")

			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				kind, watched := builders[callName(call)]
				if !watched {
					return true
				}
				at := fset.Position(call.Pos())
				if argued(lines, at.Line) {
					return true
				}
				switch kind {
				case "bounded":
					if zeroLimit(call) {
						t.Errorf("%s:%d: %s builds a path with no limit; pass PathLengthCap() or say why with //sp:unbounded",
							path, at.Line, callName(call))
					}
				case "unbounded":
					if inLoop(file, call) {
						t.Errorf("%s:%d: %s is a whole mesh search and this one is inside a loop; hoist it or say why with //sp:unbounded",
							path, at.Line, callName(call))
					}
				}
				return true
			})
		})
	}
}

// argued is an exception written on the line above the call.
func argued(lines []string, line int) bool {
	for i := line - 2; i >= 0 && i >= line-4; i-- {
		if i < len(lines) && allowed.MatchString(lines[i]) {
			return true
		}
	}
	return false
}

// zeroLimit is a literal zero where the limit goes, which every bounded builder
// takes as its third argument.
func zeroLimit(call *ast.CallExpr) bool {
	const limitArg = 2
	if len(call.Args) <= limitArg {
		return false
	}
	lit, ok := call.Args[limitArg].(*ast.BasicLit)
	if !ok {
		return false
	}
	return strings.TrimSuffix(strings.TrimSuffix(lit.Value, ".0"), ".") == "0"
}

// inLoop says the call sits under a for or a range.
func inLoop(file *ast.File, call *ast.CallExpr) bool {
	found := false

	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		switch n.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			ast.Inspect(n, func(under ast.Node) bool {
				if under == call {
					found = true
				}
				return !found
			})
		}
		return !found
	})

	return found
}

// walkGo reads every non-test Go file under dir and its subdirectories.
func walkGo(t *testing.T, dir string, visit func(path string, file *ast.File, fset *token.FileSet, source string)) {
	t.Helper()

	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		source, err := os.ReadFile(path) //nolint:gosec // this repository's own tree
		if err != nil {
			return err
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
		if err != nil {
			return err
		}
		visit(path, file, fset, string(source))
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", dir, err)
	}
}
