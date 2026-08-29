package spgen

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"
)

// expr renders one expression. Binary and unary expressions are always
// parenthesised: Go and C disagree about where & and == sit relative to each
// other, so reproducing Go's precedence in SourcePawn by omitting parentheses
// is a silent wrong answer.
func (g *generator) expr(e ast.Expr) string {
	switch n := e.(type) {
	case *ast.Ident:
		return g.ident(n)
	case *ast.BasicLit:
		return g.literal(n)
	case *ast.ParenExpr:
		// Every binary and unary expression parenthesises itself, so Go's own
		// parentheses would only double up.
		inner := g.expr(n.X)
		if strings.HasPrefix(inner, "(") && matchesToEnd(inner) {
			return inner
		}
		return "(" + inner + ")"
	case *ast.SelectorExpr:
		return g.expr(n.X) + "." + n.Sel.Name
	case *ast.IndexExpr:
		return g.expr(n.X) + "[" + g.expr(n.Index) + "]"
	case *ast.BinaryExpr:
		return g.binary(n)
	case *ast.UnaryExpr:
		return g.unary(n)
	case *ast.CallExpr:
		return g.call(n)
	}
	g.refuse(e.Pos(), fmt.Sprintf("the expression %T", e),
		"spgen emits names, numeric literals, arithmetic, indexing, field access and calls")
	return "0"
}

func (g *generator) ident(n *ast.Ident) string {
	switch n.Name {
	case "true", "false":
		return n.Name
	}
	obj := g.info.Uses[n]
	if obj == nil {
		obj = g.info.Defs[n]
	}
	if obj != nil && obj.Parent() == g.pkg.Scope() {
		return g.name(n.Name)
	}
	return identifier(n.Name)
}

func (g *generator) literal(n *ast.BasicLit) string {
	tv, ok := g.info.Types[n]
	if !ok || tv.Value == nil {
		g.refuse(n.Pos(), "a literal with no constant value", "write a numeric literal")
		return "0"
	}
	switch tv.Value.Kind() {
	case constant.Int:
		// Rendered from the value, not the source, so 1_000 and 0b1010 come
		// out as decimals SourcePawn's lexer accepts.
		return tv.Value.ExactString()
	case constant.Float:
		f, _ := constant.Float64Val(tv.Value)
		return floatLiteral(f)
	}
	g.refuse(n.Pos(), "a literal that is neither an integer nor a float",
		"the subset has no strings; use an int32 identifier")
	return "0"
}

func (g *generator) binary(n *ast.BinaryExpr) string {
	op, ok := binaryOps[n.Op]
	if !ok {
		g.refuse(n.OpPos, "the operator "+n.Op.String(),
			"SourcePawn has no AND NOT; write x & ~y")
		return "0"
	}
	return "(" + g.expr(n.X) + " " + op + " " + g.expr(n.Y) + ")"
}

var binaryOps = map[token.Token]string{
	token.ADD: "+", token.SUB: "-", token.MUL: "*", token.QUO: "/", token.REM: "%",
	token.AND: "&", token.OR: "|", token.XOR: "^", token.SHL: "<<", token.SHR: ">>",
	token.LAND: "&&", token.LOR: "||",
	token.EQL: "==", token.NEQ: "!=", token.LSS: "<", token.LEQ: "<=",
	token.GTR: ">", token.GEQ: ">=",
}

func (g *generator) unary(n *ast.UnaryExpr) string {
	switch n.Op {
	case token.NOT:
		return "!" + g.expr(n.X)
	case token.SUB:
		return "(-" + g.expr(n.X) + ")"
	case token.ADD:
		return g.expr(n.X)
	case token.XOR:
		return "(~" + g.expr(n.X) + ")"
	}
	g.refuse(n.OpPos, "the unary operator "+n.Op.String(), "spgen emits !, -, + and ^")
	return "0"
}

func (g *generator) call(n *ast.CallExpr) string {
	if id, ok := n.Fun.(*ast.Ident); ok {
		if _, isType := g.info.Uses[id].(*types.TypeName); isType {
			return g.conversion(n, id)
		}
		if id.Name == "len" {
			if len(n.Args) != 1 {
				g.refuse(n.Pos(), "len with the wrong number of arguments", "call len on one array")
				return "0"
			}
			return "sizeof(" + g.expr(n.Args[0]) + ")"
		}
	}
	fun, ok := n.Fun.(*ast.Ident)
	if !ok {
		g.refuse(n.Pos(), "a call to something other than a named function",
			"call a function declared in this package")
		return "0"
	}
	if obj := g.info.Uses[fun]; obj == nil || obj.Parent() != g.pkg.Scope() {
		g.refuse(fun.Pos(), "a call to "+fun.Name+", which is not a function of this package",
			"declare it here, or leave the work to the hand-written SourcePawn")
		return "0"
	}
	args := make([]string, 0, len(n.Args))
	for _, a := range n.Args {
		args = append(args, g.expr(a))
	}
	return g.name(fun.Name) + "(" + strings.Join(args, ", ") + ")"
}

// conversion is a Go conversion. Between two things that are the same cell it
// is view_as; between a float and an int it is a rounding decision the author
// has to make in Go, so it is refused.
func (g *generator) conversion(n *ast.CallExpr, id *ast.Ident) string {
	if len(n.Args) != 1 {
		g.refuse(n.Pos(), "a conversion with more than one argument", "convert one value")
		return "0"
	}
	to := g.spType(g.info.TypeOf(n), id.Pos())
	from := g.spType(g.info.TypeOf(n.Args[0]), n.Args[0].Pos())
	if (to == "float") != (from == "float") {
		g.refuse(n.Pos(), fmt.Sprintf("a conversion between %s and %s", from, to),
			"do the rounding explicitly on the Go side, or keep the value in one representation")
		return "0"
	}
	inner := g.expr(n.Args[0])
	if to == from {
		return inner
	}
	return "view_as<" + to + ">(" + inner + ")"
}
