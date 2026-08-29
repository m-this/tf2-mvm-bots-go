package spgen

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

func (g *generator) block(b *ast.BlockStmt, depth int) {
	g.indent(depth, "{")
	for _, s := range b.List {
		g.stmt(s, depth+1)
	}
	g.indent(depth, "}")
}

func (g *generator) stmt(s ast.Stmt, depth int) {
	switch n := s.(type) {
	case nil, *ast.EmptyStmt:
	case *ast.BlockStmt:
		g.block(n, depth)
	case *ast.ReturnStmt:
		g.returnStmt(n, depth)
	case *ast.IfStmt:
		g.ifStmt(n, depth)
	case *ast.SwitchStmt:
		g.switchStmt(n, depth)
	case *ast.AssignStmt:
		g.assign(n, depth)
	case *ast.IncDecStmt:
		g.indent(depth, g.expr(n.X)+n.Tok.String()+";")
	case *ast.ExprStmt:
		call, ok := n.X.(*ast.CallExpr)
		if !ok {
			g.refuse(n.Pos(), "an expression evaluated for nothing", "make it a call or an assignment")
			return
		}
		g.indent(depth, g.expr(call)+";")
	case *ast.ForStmt:
		g.forStmt(n, depth)
	case *ast.DeclStmt:
		g.localDecl(n, depth)
	case *ast.BranchStmt:
		switch n.Tok {
		case token.BREAK, token.CONTINUE:
			g.indent(depth, n.Tok.String()+";")
		default:
			g.refuse(n.Pos(), "the branch statement "+n.Tok.String(), "restructure with if and for")
		}
	default:
		g.refuse(s.Pos(), fmt.Sprintf("the statement %T", s),
			"spgen emits if, for, switch, return, assignment, local declarations and calls")
	}
}

func (g *generator) returnStmt(n *ast.ReturnStmt, depth int) {
	switch len(n.Results) {
	case 0:
		g.indent(depth, "return;")
	case 1:
		g.indent(depth, "return "+g.expr(n.Results[0])+";")
	default:
		g.refuse(n.Pos(), "a return of several values",
			"return one value; several results become by-reference parameters, which spgen does not emit yet")
	}
}

func (g *generator) ifStmt(n *ast.IfStmt, depth int) {
	if n.Init != nil {
		g.refuse(n.Init.Pos(), "an if with an init statement",
			"declare the variable on the line before the if")
	}
	g.indent(depth, "if ("+g.condition(n.Cond)+")")
	g.block(n.Body, depth)
	switch e := n.Else.(type) {
	case nil:
	case *ast.BlockStmt:
		g.indent(depth, "else")
		g.block(e, depth)
	case *ast.IfStmt:
		g.indent(depth, "else if ("+g.condition(e.Cond)+")")
		g.block(e.Body, depth)
		g.elseTail(e, depth)
	default:
		g.refuse(n.Else.Pos(), "an else that is neither a block nor an if", "write a block")
	}
}

func (g *generator) elseTail(n *ast.IfStmt, depth int) {
	switch e := n.Else.(type) {
	case nil:
	case *ast.BlockStmt:
		g.indent(depth, "else")
		g.block(e, depth)
	case *ast.IfStmt:
		g.indent(depth, "else if ("+g.condition(e.Cond)+")")
		g.block(e.Body, depth)
		g.elseTail(e, depth)
	}
}

// condition drops the outermost parentheses, because the if writes its own.
func (g *generator) condition(e ast.Expr) string {
	s := g.expr(e)
	if strings.HasPrefix(s, "(") && matchesToEnd(s) {
		return s[1 : len(s)-1]
	}
	return s
}

// matchesToEnd says whether the opening parenthesis of s closes at its last
// byte, so stripping both is safe. "(a) && (b)" must keep its parentheses.
func matchesToEnd(s string) bool {
	depth := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i == len(s)-1
			}
		}
	}
	return false
}

func (g *generator) switchStmt(n *ast.SwitchStmt, depth int) {
	if n.Init != nil {
		g.refuse(n.Init.Pos(), "a switch with an init statement", "declare the value on the line before")
	}
	if n.Tag == nil {
		g.refuse(n.Pos(), "a switch with no value to switch on", "write if / else if")
		return
	}
	g.indent(depth, "switch ("+g.condition(n.Tag)+")")
	g.indent(depth, "{")

	var def *ast.CaseClause
	for _, s := range n.Body.List {
		c, ok := s.(*ast.CaseClause)
		if !ok {
			g.refuse(s.Pos(), "a statement outside a case clause", "put it in a case")
			continue
		}
		if c.List == nil {
			def = c
			continue
		}
		labels := make([]string, 0, len(c.List))
		for _, e := range c.List {
			labels = append(labels, g.expr(e))
		}
		g.indent(depth+1, "case "+strings.Join(labels, ", ")+":")
		g.caseBody(c, depth+1)
	}
	// SourcePawn wants default last; Go does not care where it is written.
	if def != nil {
		g.indent(depth+1, "default:")
		g.caseBody(def, depth+1)
	}
	g.indent(depth, "}")
}

func (g *generator) caseBody(c *ast.CaseClause, depth int) {
	g.indent(depth, "{")
	for _, s := range c.Body {
		g.stmt(s, depth+1)
	}
	g.indent(depth, "}")
}

func (g *generator) assign(n *ast.AssignStmt, depth int) {
	if len(n.Lhs) != 1 || len(n.Rhs) != 1 {
		g.refuse(n.Pos(), "an assignment of several values at once",
			"write one assignment per line; spgen emits no temporaries yet")
		return
	}
	rhs := g.expr(n.Rhs[0])
	if n.Tok == token.DEFINE {
		id, ok := n.Lhs[0].(*ast.Ident)
		if !ok {
			g.refuse(n.Pos(), "a short declaration of something that is not a name", "declare a name")
			return
		}
		g.indent(depth, g.declare(g.info.TypeOf(n.Rhs[0]), identifier(id.Name), n.Pos())+" = "+rhs+";")
		return
	}
	op, ok := assignOps[n.Tok]
	if !ok {
		g.refuse(n.TokPos, "the assignment operator "+n.Tok.String(), "use =, +=, -=, *=, /=, %=, &=, |=, ^=, <<= or >>=")
		return
	}
	g.indent(depth, g.expr(n.Lhs[0])+" "+op+" "+rhs+";")
}

var assignOps = map[token.Token]string{
	token.ASSIGN: "=", token.ADD_ASSIGN: "+=", token.SUB_ASSIGN: "-=",
	token.MUL_ASSIGN: "*=", token.QUO_ASSIGN: "/=", token.REM_ASSIGN: "%=",
	token.AND_ASSIGN: "&=", token.OR_ASSIGN: "|=", token.XOR_ASSIGN: "^=",
	token.SHL_ASSIGN: "<<=", token.SHR_ASSIGN: ">>=",
}

func (g *generator) localDecl(n *ast.DeclStmt, depth int) {
	d, ok := n.Decl.(*ast.GenDecl)
	if !ok || (d.Tok != token.VAR && d.Tok != token.CONST) {
		g.refuse(n.Pos(), "a declaration inside a function that is not var or const",
			"declare types and functions at package level")
		return
	}
	for _, spec := range d.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, name := range vs.Names {
			decl := g.declare(g.info.TypeOf(name), identifier(name.Name), name.Pos())
			if i < len(vs.Values) {
				g.indent(depth, decl+" = "+g.expr(vs.Values[i])+";")
				continue
			}
			if len(vs.Values) != 0 {
				g.refuse(vs.Pos(), "a declaration with fewer values than names", "write one name per value")
				return
			}
			g.indent(depth, decl+";")
		}
	}
}

func (g *generator) forStmt(n *ast.ForStmt, depth int) {
	if n.Init == nil && n.Cond == nil && n.Post == nil {
		g.refuse(n.Pos(), "a for with no bound at all",
			"give the loop a condition; a generated body has an upper bound on every loop")
		return
	}
	if n.Cond == nil {
		g.refuse(n.Pos(), "a for with no condition",
			"give the loop a condition; a generated body has an upper bound on every loop")
		return
	}
	init, post := "", ""
	if n.Init != nil {
		init = strings.TrimSuffix(strings.TrimSpace(g.capture(n.Init)), ";")
	}
	if n.Post != nil {
		post = strings.TrimSuffix(strings.TrimSpace(g.capture(n.Post)), ";")
	}
	g.indent(depth, "for ("+init+"; "+g.condition(n.Cond)+"; "+post+")")
	g.block(n.Body, depth)
}

// capture renders one statement into a string, for the header of a for loop,
// which is the one place a statement is not on a line of its own.
func (g *generator) capture(s ast.Stmt) string {
	var saved strings.Builder
	saved, g.b = g.b, strings.Builder{}
	g.stmt(s, 0)
	out := g.b.String()
	g.b = saved
	return out
}
