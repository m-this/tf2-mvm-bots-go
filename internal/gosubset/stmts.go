package gosubset

import (
	"fmt"
	"go/ast"
	"go/token"
)

func (c *checker) checkBlock(b *ast.BlockStmt) {
	if b == nil {
		return
	}
	for _, s := range b.List {
		c.checkStmt(s)
	}
}

func (c *checker) checkStmt(s ast.Stmt) {
	switch n := s.(type) {
	case nil, *ast.EmptyStmt:
	case *ast.BlockStmt:
		c.checkBlock(n)
	case *ast.DeclStmt:
		c.checkGenDecl(n.Decl.(*ast.GenDecl), false)
	case *ast.ExprStmt:
		c.checkCallStmt(n)
	case *ast.AssignStmt:
		c.checkAssign(n)
	case *ast.IncDecStmt:
		c.checkExpr(n.X)
	case *ast.ReturnStmt:
		for _, r := range n.Results {
			c.checkExpr(r)
		}
	case *ast.IfStmt:
		c.checkIf(n)
	case *ast.ForStmt:
		c.checkStmt(n.Init)
		c.checkOptExpr(n.Cond)
		c.checkStmt(n.Post)
		c.checkBlock(n.Body)
	case *ast.RangeStmt:
		c.checkRange(n)
	case *ast.SwitchStmt:
		c.checkSwitch(n)
	case *ast.BranchStmt:
		c.checkBranch(n)
	default:
		c.refuseStmt(s)
	}
}

// refuseStmt names the statements that have no SourcePawn at all.
func (c *checker) refuseStmt(s ast.Stmt) {
	switch n := s.(type) {
	case *ast.GoStmt:
		c.refuse(n.Pos(), "a goroutine",
			"a generated body is one call on the server thread; do the work inline")
	case *ast.DeferStmt:
		c.refuse(n.Pos(), "a defer statement",
			"write the cleanup at each return, or restructure so there is one return")
	case *ast.SelectStmt:
		c.refuse(n.Pos(), "a select statement",
			"the subset has no channels; remove it")
	case *ast.SendStmt:
		c.refuse(n.Pos(), "a channel send",
			"the subset has no channels; return the value instead")
	case *ast.LabeledStmt:
		c.refuse(n.Pos(), "a label",
			"SourcePawn has no labels; restructure the loop with a flag or an early return")
	case *ast.TypeSwitchStmt:
		c.refuse(n.Pos(), "a type switch",
			"the subset has no interfaces; switch on an int32 tag field instead")
	default:
		c.refuse(s.Pos(), fmt.Sprintf("the statement %T", s),
			"the subset has assignment, if, for, switch, return, break and continue")
	}
}

func (c *checker) checkIf(n *ast.IfStmt) {
	c.checkStmt(n.Init)
	c.checkExpr(n.Cond)
	c.checkBlock(n.Body)
	c.checkStmt(n.Else)
}

func (c *checker) checkBranch(n *ast.BranchStmt) {
	switch n.Tok {
	case token.GOTO:
		c.refuse(n.Pos(), "a goto",
			"restructure with if and for; SourcePawn has no goto the generator will emit")
	case token.FALLTHROUGH:
		c.refuse(n.Pos(), "a fallthrough",
			"list the values on one case, `case a, b:`, which is what SourcePawn switch does")
	}
	if n.Label != nil {
		c.refuse(n.Pos(), "a labelled break or continue",
			"SourcePawn has no labels; use a flag checked by the outer loop")
	}
}

// checkRange accepts range because of what the type rules already refuse. A
// value in this subset is an integer, a fixed-length array or a struct, so a
// range expression cannot be a map, a channel, a slice or a function.
func (c *checker) checkRange(n *ast.RangeStmt) {
	if n.Tok == token.ASSIGN {
		c.refuse(n.Pos(), "a range loop assigning to existing variables",
			"declare the loop variables with := so the generated declaration is local to the loop")
	}
	c.checkExpr(n.X)
	c.checkBlock(n.Body)
}

func (c *checker) checkSwitch(n *ast.SwitchStmt) {
	c.checkStmt(n.Init)
	if n.Tag == nil {
		c.refuse(n.Pos(), "a switch with no value to switch on",
			"SourcePawn switch needs a tag; write if / else if for a chain of conditions")
	} else {
		c.checkExpr(n.Tag)
	}
	for _, s := range n.Body.List {
		clause, ok := s.(*ast.CaseClause)
		if !ok {
			c.refuseStmt(s)
			continue
		}
		for _, e := range clause.List {
			// A case may also be a constant the extern package names,
			// TFClass_Soldier and its like, which is a call in Go and
			// a constant in SourcePawn. The emitter has the types to
			// tell that from a call that computes something; this
			// checker does not.
			if _, isCall := e.(*ast.CallExpr); isCall {
				c.checkExpr(e)
				continue
			}
			c.checkConstExpr(e, "a switch case")
		}
		for _, body := range clause.Body {
			c.checkStmt(body)
		}
	}
}

func (c *checker) checkAssign(n *ast.AssignStmt) {
	if n.Tok == token.AND_NOT_ASSIGN {
		c.refuse(n.TokPos, "the &^= operator",
			"SourcePawn has no AND NOT; write x &= ^y")
	}
	for _, lhs := range n.Lhs {
		c.checkAssignTarget(lhs)
	}
	for _, rhs := range n.Rhs {
		c.checkExpr(rhs)
	}
}

func (c *checker) checkAssignTarget(lhs ast.Expr) {
	if id, ok := lhs.(*ast.Ident); ok && id.Name == "_" {
		return
	}
	c.checkExpr(lhs)
}

// checkCallStmt allows a discarded result: pass10 of SourceGo turns the extra
// results of a call statement into temporaries, so the caller need not name
// what it does not want.
func (c *checker) checkCallStmt(n *ast.ExprStmt) {
	if _, ok := n.X.(*ast.CallExpr); !ok {
		c.refuse(n.Pos(), "an expression evaluated for nothing",
			"a statement in the subset is a call, an assignment or a control statement")
		return
	}
	c.checkExpr(n.X)
}

func (c *checker) checkOptExpr(e ast.Expr) {
	if e != nil {
		c.checkExpr(e)
	}
}
