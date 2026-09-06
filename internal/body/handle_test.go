package body_test

import (
	"go/ast"
	"go/token"
	"testing"
)

/*
	Closing a handle is not forgetting it

Close frees the handle and leaves the variable holding the number it used to
be. SourceMod hands that number out again, so the next read is not a null one:
it is somebody else's list, and the guard that asks whether the variable is
null says no.

mvm-b41 is what that cost. Config_LoadServerLoadout closes the loadout and the
pending seats on every map change, and closed neither of them out of its own
variable, so NoteBotSeatPending took the length of a handle SourceMod had
already reallocated and threw error 1 on the first bot the new map asked for.
It only ever happened on a changelevel, because a restart makes the variables
null again and a changelevel does not.

The rule is the shape rather than the call: a package-level handle that is
closed is assigned on the next line, either to its null or to its replacement.
*/

// handleType is a type this package holds a SourceMod handle in, so closing one
// leaves a number behind rather than nothing.
var handleType = map[string]bool{
	"List": true, "KeyValues": true, "Menu": true, "Panel": true,
	"StringMap": true, "GameData": true, "DHook": true, "Cookie": true,
	"Handle": true, "Timer": true,
}

// globalHandles is every package-level variable of one of those types.
func globalHandles(file *ast.File) map[string]bool {
	out := map[string]bool{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			sel, ok := value.Type.(*ast.SelectorExpr)
			if !ok || !handleType[sel.Sel.Name] {
				continue
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "engine" {
				continue
			}
			for _, name := range value.Names {
				out[name.Name] = true
			}
		}
	}
	return out
}

// closes is the variable a statement closes, empty for every other statement.
func closes(stmt ast.Stmt, global map[string]bool) string {
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return ""
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Close" {
		return ""
	}
	receiver, ok := sel.X.(*ast.Ident)
	if !ok || !global[receiver.Name] {
		return ""
	}
	return receiver.Name
}

// assigns says a statement writes the named variable.
func assigns(stmt ast.Stmt, name string) bool {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok {
		return false
	}
	for _, target := range assign.Lhs {
		if ident, ok := target.(*ast.Ident); ok && ident.Name == name {
			return true
		}
	}
	return false
}

// TestEveryClosedHandleIsForgotten walks the bodies and the behaviours.
func TestEveryClosedHandleIsForgotten(t *testing.T) {
	closed := 0

	for _, dir := range []string{"../body", "../action"} {
		walkGo(t, dir, func(path string, file *ast.File, fset *token.FileSet, _ string) {
			global := globalHandles(file)
			if len(global) == 0 {
				return
			}
			ast.Inspect(file, func(n ast.Node) bool {
				block, ok := n.(*ast.BlockStmt)
				if !ok {
					return true
				}
				for i, stmt := range block.List {
					name := closes(stmt, global)
					if name == "" {
						continue
					}
					closed++
					if i+1 < len(block.List) && assigns(block.List[i+1], name) {
						continue
					}
					t.Errorf("%s:%d: %s is closed and still holds the handle; assign it on the next line",
						path, fset.Position(stmt.Pos()).Line, name)
				}
				return true
			})
		})
	}
	if closed == 0 {
		t.Fatal("no package-level handle is closed anywhere, so this test proves nothing")
	}
}
