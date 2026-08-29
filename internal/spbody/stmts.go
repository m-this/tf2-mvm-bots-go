package spbody

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

func (e *emitter) block(b *ast.BlockStmt) {
	e.line("{")
	e.indent++
	for _, s := range b.List {
		e.stmt(s)
	}
	e.indent--
	e.line("}")
}

func (e *emitter) stmt(s ast.Stmt) {
	switch n := s.(type) {
	case nil, *ast.EmptyStmt:
	case *ast.BlockStmt:
		e.block(n)
	case *ast.DeclStmt:
		e.localDecl(n.Decl.(*ast.GenDecl))
	case *ast.ExprStmt:
		e.line("%s;", e.expr(n.X))
	case *ast.AssignStmt:
		e.assign(n)
	case *ast.IncDecStmt:
		e.checkWritable(n.X)
		e.line("%s%s;", e.expr(n.X), n.Tok)
	case *ast.ReturnStmt:
		e.returnStmt(n)
	case *ast.IfStmt:
		e.ifStmt(n)
	case *ast.ForStmt:
		e.forStmt(n)
	case *ast.RangeStmt:
		e.rangeStmt(n)
	case *ast.SwitchStmt:
		e.switchStmt(n)
	case *ast.BranchStmt:
		e.line("%s;", n.Tok)
	default:
		e.fail(s.Pos(), "the statement %T has no SourcePawn", s)
	}
}

func (e *emitter) localDecl(d *ast.GenDecl) {
	if d.Tok != token.VAR && d.Tok != token.CONST {
		e.fail(d.Pos(), "a %s inside a function", d.Tok)
		return
	}
	for _, spec := range d.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			e.fail(d.Pos(), "an unrecognised declaration inside a function")
			continue
		}
		e.valueSpec(vs)
	}
}

func (e *emitter) valueSpec(vs *ast.ValueSpec) {
	for i, name := range vs.Names {
		if name.Name == "_" {
			continue
		}
		obj := e.info.Defs[name]
		if obj == nil {
			e.fail(name.Pos(), "%s has no type the checker could give it", name.Name)
			continue
		}
		tag, dims, err := e.spType(obj.Type())
		if err != nil {
			e.fail(name.Pos(), "%s: %v", name.Name, err)
			continue
		}
		decl := declare(tag, e.ident(name.Pos(), name.Name), dims)
		if i >= len(vs.Values) {
			e.line("%s;", decl)
			continue
		}
		e.line("%s = %s;", decl, e.expr(vs.Values[i]))
	}
}

func (e *emitter) assign(n *ast.AssignStmt) {
	if len(n.Rhs) == 1 && len(n.Lhs) > 1 {
		e.multiAssign(n)
		return
	}
	if len(n.Lhs) != len(n.Rhs) {
		e.fail(n.Pos(), "an assignment of %d values to %d names", len(n.Rhs), len(n.Lhs))
		return
	}
	for i, lhs := range n.Lhs {
		e.assignOne(n, lhs, n.Rhs[i])
	}
}

func (e *emitter) assignOne(n *ast.AssignStmt, lhs, rhs ast.Expr) {
	if id, ok := lhs.(*ast.Ident); ok && id.Name == "_" {
		e.line("%s;", e.expr(rhs))
		return
	}
	if e.arrayCall(n.Tok == token.DEFINE, lhs, rhs) {
		return
	}
	if n.Tok == token.DEFINE {
		e.define(lhs, rhs)
		return
	}
	e.checkWritable(lhs)
	e.line("%s %s %s;", e.expr(lhs), n.Tok, e.expr(rhs))
}

// define emits the declaration a := stands for. The type is the one go/types
// gave the name, so an untyped literal has already been defaulted.
func (e *emitter) define(lhs, rhs ast.Expr) {
	id, ok := lhs.(*ast.Ident)
	if !ok {
		e.fail(lhs.Pos(), "a short declaration of something that is not a name")
		return
	}
	obj := e.info.Defs[id]
	if obj == nil {
		// := over a name declared in an outer scope is an assignment.
		e.checkWritable(lhs)
		e.line("%s = %s;", e.expr(lhs), e.expr(rhs))
		return
	}
	tag, dims, err := e.spType(obj.Type())
	if err != nil {
		e.fail(id.Pos(), "%s: %v", id.Name, err)
		return
	}
	e.line("%s = %s;", declare(tag, e.ident(id.Pos(), id.Name), dims), e.expr(rhs))
}

/*
	arrayCall is the vector idiom

A function whose first result is an array returns nothing in SourcePawn and
fills a parameter instead, so `v := WorldSpaceCenter(e)` is a declaration and a
call, not an assignment. Doing it here rather than in the expression emitter is
deliberate: it only works as a whole statement, and a nested one is refused
where it is written.
*/
func (e *emitter) arrayCall(define bool, lhs, rhs ast.Expr) bool {
	call, ok := rhs.(*ast.CallExpr)
	if !ok || !e.isArrayValue(rhs) {
		return false
	}
	if tv, isType := e.info.Types[call.Fun]; isType && tv.IsType() {
		return false // a conversion, which is not a call at all
	}
	if define {
		e.declareTarget(lhs)
	} else {
		e.checkWritable(lhs)
	}
	e.line("%s;", e.callWith(call, []string{e.expr(lhs)}))
	return true
}

func (e *emitter) isArrayValue(x ast.Expr) bool {
	t := e.info.Types[x].Type
	if t == nil {
		return false
	}
	_, isArray := types.Unalias(t).Underlying().(*types.Array)
	return isArray
}

// multiAssign is the call with several results. SourcePawn has one return and
// by-reference parameters, so the names after the first are appended to the
// call, which is the shape signature emitted for the function being called.
func (e *emitter) multiAssign(n *ast.AssignStmt) {
	call, ok := n.Rhs[0].(*ast.CallExpr)
	if !ok {
		e.fail(n.Pos(), "an assignment of several values from something that is not a call")
		return
	}
	extra := make([]string, 0, len(n.Lhs)-1)
	for _, lhs := range n.Lhs[1:] {
		if id, ok := lhs.(*ast.Ident); ok && id.Name == "_" {
			e.fail(lhs.Pos(), "a discarded extra result; SourcePawn writes through the parameter, so name it")
			return
		}
		if n.Tok == token.DEFINE {
			e.declareTarget(lhs)
		} else {
			e.checkWritable(lhs)
		}
		extra = append(extra, e.expr(lhs))
	}
	text := e.callWith(call, extra)
	first := n.Lhs[0]
	if id, ok := first.(*ast.Ident); ok && id.Name == "_" {
		e.line("%s;", text)
		return
	}
	if n.Tok == token.DEFINE {
		if decl, ok := e.declaration(first); ok {
			e.line("%s = %s;", decl, text)
			return
		}
	}
	e.checkWritable(first)
	e.line("%s = %s;", e.expr(first), text)
}

func (e *emitter) declareTarget(lhs ast.Expr) {
	if decl, ok := e.declaration(lhs); ok {
		e.line("%s;", decl)
	}
}

// declaration is the SourcePawn declaration for a name a short declaration
// introduces, and false when the name was declared in an outer scope and this
// is an assignment to it.
func (e *emitter) declaration(lhs ast.Expr) (string, bool) {
	id, ok := lhs.(*ast.Ident)
	if !ok {
		return "", false
	}
	obj := e.info.Defs[id]
	if obj == nil {
		return "", false
	}
	tag, dims, err := e.spType(obj.Type())
	if err != nil {
		e.fail(id.Pos(), "%s: %v", id.Name, err)
		return "", false
	}
	return declare(tag, e.ident(id.Pos(), id.Name), dims), true
}

// checkWritable refuses a write to an array or enum struct parameter. Go
// copies it in and SourcePawn passes it by reference, so the write is invisible
// to the Go caller and visible to the SourcePawn one, and no amount of care at
// the call site makes the two agree.
func (e *emitter) checkWritable(lhs ast.Expr) {
	root := lhs
	for {
		switch x := root.(type) {
		case *ast.IndexExpr:
			root = x.X
		case *ast.SelectorExpr:
			root = x.X
		case *ast.ParenExpr:
			root = x.X
		default:
			if id, ok := root.(*ast.Ident); ok && e.byRef[id.Name] {
				e.fail(lhs.Pos(), "a write to the parameter %s, which SourcePawn passes by reference and Go copies; copy it into a local first", id.Name)
			}
			return
		}
	}
}

func (e *emitter) returnStmt(n *ast.ReturnStmt) {
	if len(n.Results) == 0 {
		if e.resultName == "" {
			e.line("return;")
			return
		}
		e.line("return %s;", e.resultName)
		return
	}
	first := 1
	if e.returnsArray {
		first = 0
	}
	if len(n.Results) != len(e.outParams)+first {
		e.fail(n.Pos(), "a return of %d values from a function with %d results", len(n.Results), len(e.outParams)+first)
		return
	}
	for i, out := range e.outParams {
		value := e.expr(n.Results[i+first])
		if value == out.name {
			continue // return centre, where centre is the parameter
		}
		e.line("%s = %s;", out.name, value)
	}
	if e.returnsArray {
		e.line("return;")
		return
	}
	e.line("return %s;", e.expr(n.Results[0]))
}

func (e *emitter) ifStmt(n *ast.IfStmt) {
	if n.Init != nil {
		// SourcePawn has no init clause, and lifting the statement out
		// widens the scope of what it declares. That is a difference,
		// so it is refused rather than emitted.
		e.fail(n.Init.Pos(), "an if with an init statement; declare the value on the line before the if")
		return
	}
	e.line("if (%s)", e.expr(n.Cond))
	e.block(n.Body)
	switch alt := n.Else.(type) {
	case nil:
	case *ast.BlockStmt:
		e.line("else")
		e.block(alt)
	case *ast.IfStmt:
		e.line("else")
		e.indent++
		e.ifStmt(alt)
		e.indent--
	default:
		e.fail(n.Else.Pos(), "an else that is neither a block nor an if")
	}
}

func (e *emitter) forStmt(n *ast.ForStmt) {
	init, post := "", ""
	if n.Init != nil {
		init = e.inlineStmt(n.Init)
	}
	if n.Post != nil {
		post = e.inlineStmt(n.Post)
	}
	cond := ""
	if n.Cond != nil {
		cond = e.expr(n.Cond)
	}
	e.line("for (%s; %s; %s)", init, cond, post)
	e.block(n.Body)
}

// inlineStmt is the init or post clause of a for, which SourcePawn writes on
// the same line and so cannot be a block.
func (e *emitter) inlineStmt(s ast.Stmt) string {
	switch n := s.(type) {
	case *ast.AssignStmt:
		if len(n.Lhs) != 1 || len(n.Rhs) != 1 {
			e.fail(s.Pos(), "a for clause assigning more than one value")
			return ""
		}
		if n.Tok == token.DEFINE {
			if decl, ok := e.declaration(n.Lhs[0]); ok {
				return fmt.Sprintf("%s = %s", decl, e.expr(n.Rhs[0]))
			}
		}
		e.checkWritable(n.Lhs[0])
		return fmt.Sprintf("%s %s %s", e.expr(n.Lhs[0]), n.Tok, e.expr(n.Rhs[0]))
	case *ast.IncDecStmt:
		e.checkWritable(n.X)
		return e.expr(n.X) + n.Tok.String()
	case *ast.ExprStmt:
		return e.expr(n.X)
	default:
		e.fail(s.Pos(), "a for clause that is not an assignment, an increment or a call")
		return ""
	}
}

// rangeStmt covers the two things the subset leaves to range over: an integer
// and a fixed-length array. Both become the counted for SourcePawn has.
func (e *emitter) rangeStmt(n *ast.RangeStmt) {
	if n.Value != nil {
		e.fail(n.Value.Pos(), "a range with a value variable; index the array, which is what the generated loop does")
		return
	}
	index := "_i"
	if n.Key != nil {
		id, ok := n.Key.(*ast.Ident)
		if !ok {
			e.fail(n.Key.Pos(), "a range key that is not a name")
			return
		}
		if id.Name != "_" {
			index = e.ident(id.Pos(), id.Name)
		}
	}
	limit, err := e.rangeLimit(n.X)
	if err != nil {
		e.fail(n.X.Pos(), "%v", err)
		return
	}
	e.line("for (int %s = 0; %s < %s; %s++)", index, index, limit, index)
	e.block(n.Body)
}

func (e *emitter) rangeLimit(x ast.Expr) (string, error) {
	t := e.info.Types[x].Type
	if t == nil {
		return "", fmt.Errorf("the range expression has no type")
	}
	switch u := types.Unalias(t).Underlying().(type) {
	case *types.Array:
		return fmt.Sprintf("%d", u.Len()), nil
	case *types.Basic:
		if u.Info()&types.IsInteger != 0 {
			return e.expr(x), nil
		}
	}
	return "", fmt.Errorf("range over %s; the subset ranges over an array or an integer", t)
}

func (e *emitter) switchStmt(n *ast.SwitchStmt) {
	if n.Init != nil {
		e.fail(n.Init.Pos(), "a switch with an init statement; declare the value on the line before it")
		return
	}
	if n.Tag == nil {
		e.fail(n.Pos(), "a switch with no value; SourcePawn switch needs one")
		return
	}
	e.line("switch (%s)", e.expr(n.Tag))
	e.line("{")
	e.indent++
	for _, s := range n.Body.List {
		clause, ok := s.(*ast.CaseClause)
		if !ok {
			e.fail(s.Pos(), "a switch body statement that is not a case")
			continue
		}
		e.caseClause(clause)
	}
	e.indent--
	e.line("}")
}

func (e *emitter) caseClause(clause *ast.CaseClause) {
	if clause.List == nil {
		e.line("default:")
	} else {
		values := make([]string, 0, len(clause.List))
		for _, v := range clause.List {
			values = append(values, e.expr(v))
		}
		e.line("case %s:", strings.Join(values, ", "))
	}
	e.line("{")
	e.indent++
	for _, s := range clause.Body {
		e.stmt(s)
	}
	e.indent--
	e.line("}")
}
