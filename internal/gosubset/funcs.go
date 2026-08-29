package gosubset

import (
	"go/ast"
)

func (c *checker) checkFuncDecl(d *ast.FuncDecl) {
	if d.Recv != nil {
		c.refuse(d.Recv.Pos(), "a method receiver",
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
	c.checkBlock(d.Body)
}

func (c *checker) checkParams(t *ast.FuncType) {
	if t.Params != nil {
		for _, f := range t.Params.List {
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
