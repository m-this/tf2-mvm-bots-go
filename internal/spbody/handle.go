package spbody

import (
	"go/ast"
	"go/token"
	"go/types"
)

/*
	Handles, which are lifetimes

TheNavMesh hands back an AreasCollector, and it has to be deleted. The plugin
deletes it once, after the loop, which is right until somebody adds a return
inside the loop; then it leaks, every frame, and nothing says so.

So the Go says it the Go way:

	areas := engine.CollectAreasInRadius(origin, 300.0)
	defer areas.Close()

and the generator puts the delete at every way out rather than at the one the
author remembered. That is stronger than what it replaces, and it is the only
place in this port where the generated SourcePawn is deliberately not what the
plugin wrote.

Two rules make it provable rather than hopeful:

  - the defer is at the top level of the function, so it has always run by the
    time a later return is reached. gosubset refuses it anywhere else.
  - a local of a handle type that is never closed is refused. A leak the author
    did not write is a leak nobody will find.

Only the ones this function opens. A handle it was handed, or one it reads off
the map configuration, belongs to whoever made it, and closing that is worse
than leaking it.
*/

// deferred is one pending close: the name, and where in the body it was
// declared, so returns before it are left alone.
type deferred struct {
	name  string
	after token.Pos
}

// handleType says whether the type is one this package tracks, which is any
// named type from the extern package with a //sp:delete method on it.
func (e *emitter) handleType(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg() == e.pkg {
		return false
	}
	return e.handles[named.Obj().Pkg().Name()+"."+named.Obj().Name()]
}

// collectHandles finds the types the extern package deletes, so a local of one
// can be held to being closed.
func (e *emitter) collectHandles() {
	e.handles = make(map[string]bool)
	for qualified, x := range e.cfg.Externs {
		if !x.Delete {
			continue
		}
		// engine.Areas.Close -> engine.Areas
		if i := lastDot(qualified); i >= 0 {
			e.handles[qualified[:i]] = true
		}
	}
}

func lastDot(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}

/*
	checkClosed refuses a handle nobody closes

Every local whose type is a handle has to be the receiver of a close somewhere
in the function, deferred or not. Whether it is on every path is what the defer
rule above settles; whether it happens at all is this.
*/
// opens is the locals this function created, which are the only ones it owes a
// close. A handle read off a SourcePawn variable is not created: the map
// configuration's lists outlive every function that looks at one, and deleting
// one is worse than leaking it.
func (e *emitter) opens(d *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	record := func(lhs []ast.Expr, rhs []ast.Expr) {
		if len(lhs) != 1 || len(rhs) != 1 {
			return
		}
		call, ok := rhs[0].(*ast.CallExpr)
		if !ok {
			return
		}
		if x, isExtern := e.externOfCall(call); isExtern && x.Global {
			return
		}
		if id, ok := lhs[0].(*ast.Ident); ok {
			out[id.Name] = true
		}
	}
	ast.Inspect(d.Body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			record(s.Lhs, s.Rhs)
		case *ast.ValueSpec:
			for i, name := range s.Names {
				if i < len(s.Values) {
					record([]ast.Expr{name}, []ast.Expr{s.Values[i]})
				}
			}
		}
		return true
	})
	return out
}

func (e *emitter) checkClosed(d *ast.FuncDecl) {
	opened := e.opens(d)
	closed := map[string]bool{}
	ast.Inspect(d.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if x, _, isMethod := e.externMethod(sel); isMethod && x.Delete {
			if id, ok := sel.X.(*ast.Ident); ok {
				closed[id.Name] = true
			}
		}
		return true
	})

	for id, obj := range e.info.Defs {
		v, ok := obj.(*types.Var)
		if !ok || v.Parent() == nil || v.Parent() == e.pkg.Scope() {
			continue
		}
		if !e.handleType(v.Type()) || !opened[id.Name] || closed[id.Name] {
			continue
		}
		if !within(d.Body, id.Pos()) {
			continue
		}
		e.fail(id.Pos(), "%s is a handle and nothing closes it; write defer %s.Close() under the line that opens it", id.Name, id.Name)
	}
}

func within(body *ast.BlockStmt, pos token.Pos) bool {
	return pos >= body.Pos() && pos <= body.End()
}
