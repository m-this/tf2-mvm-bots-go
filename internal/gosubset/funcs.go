package gosubset

import (
	"go/ast"
)

func (c *checker) checkFuncDecl(d *ast.FuncDecl) {
	/* A method is an enum struct's, and nothing else

	SourcePawn writes a method inside the braces of the enum struct it hangs
	off, so a method on a struct this package declares is exactly what an
	enum struct method is. A method on anything else has no form: the
	methodmap belongs to the actions generator, not to this one. */
	if d.Recv != nil && !c.isLocalStructMethod(d) {
		c.refuse(d.Recv.Pos(), "a method receiver on something that is not a struct this package declares",
			"write a plain function taking the receiver as its first parameter; the methodmap belongs to the actions generator, not the body generator")
	}
	if d.Name.Name == "init" && d.Recv == nil {
		c.refuse(d.Pos(), "an init function",
			"a generated body runs when it is called and never at load; move the setup into the caller")
	}
	if d.Type.TypeParams != nil {
		c.refuse(d.Type.TypeParams.Pos(), "a generic function",
			"write one function per concrete type; SourcePawn has no type parameters")
	}
	if d.Body == nil {
		c.refuse(d.Pos(), "a function with no body",
			"declare native bindings in the generated bindings package, not here")
		return
	}
	c.checkParams(d.Type)
	c.checkFuncBody(d.Body)
}

/*
	checkFuncBody is checkBlock with defer allowed

A handle is a lifetime, and the Go way to say so is defer. It is accepted at the
top level of a function and nowhere else: there it has always run by the time
any later return is reached, so the emitter can put the delete at every way out
and know it is right. Nested, it would have to know whether the branch was
taken.
*/
func (c *checker) checkFuncBody(b *ast.BlockStmt) {
	if b == nil {
		return
	}
	for _, s := range b.List {
		if d, isDefer := s.(*ast.DeferStmt); isDefer {
			c.checkDefer(d)
			continue
		}
		c.checkStmt(s)
	}
}

// checkDefer accepts the one shape the emitter can discharge: a method call on
// a name, with no arguments.
func (c *checker) checkDefer(d *ast.DeferStmt) {
	sel, ok := d.Call.Fun.(*ast.SelectorExpr)
	if !ok {
		c.refuse(d.Pos(), "a defer of something that is not a method call",
			"defer the close of the handle, x.Close(), which is the only cleanup the generator emits")
		return
	}
	if _, ok := sel.X.(*ast.Ident); !ok {
		c.refuse(d.Pos(), "a defer on something that is not a name",
			"assign the handle to a name first, then defer its close")
		return
	}
	if len(d.Call.Args) != 0 {
		c.refuse(d.Pos(), "a defer of a call with arguments",
			"the generator emits a delete, which takes none")
	}
}

func (c *checker) checkParams(t *ast.FuncType) {
	if t.Params != nil {
		for _, f := range t.Params.List {
			// Text a function is given is const char[] in
			// SourcePawn, which the plugin's own helpers take: a
			// reason to log, a name to compare. It is a parameter
			// and nothing else, and the emitter is the second gate
			// on what a body does with one.
			if id, ok := f.Type.(*ast.Ident); ok && id.Name == "string" {
				continue
			}
			c.checkType(f.Type)
		}
	}
	if t.Results == nil {
		return
	}
	for _, f := range t.Results.List {
		// pass3 of SourceGo turns the results after the first into
		// by-reference parameters, so several results are fine and a
		// returned pointer is not.
		c.checkType(f.Type)
	}
}

// isLocalStructMethod says the receiver is a struct type this package declares,
// which is what an enum struct method hangs off.
func (c *checker) isLocalStructMethod(d *ast.FuncDecl) bool {
	if d.Recv == nil || len(d.Recv.List) != 1 {
		return false
	}
	t := d.Recv.List[0].Type
	if star, isPointer := t.(*ast.StarExpr); isPointer {
		t = star.X
	}
	id, ok := t.(*ast.Ident)
	return ok && c.structs[id.Name]
}
