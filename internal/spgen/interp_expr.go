package spgen

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
)

// The expression half of the interpreter. && and || short circuit here, which
// is what makes the recorded order of predicates the source's order rather
// than a list of everything the expression mentions.

func (in *interp) eval(e ast.Expr, sc scope) value {
	switch n := e.(type) {
	case *ast.ParenExpr:
		return in.eval(n.X, sc)
	case *ast.Ident:
		return in.ident(n, sc)
	case *ast.BasicLit:
		return in.constValue(n)
	case *ast.SelectorExpr:
		return in.selector(n, sc)
	case *ast.UnaryExpr:
		if n.Op != token.NOT {
			in.fail(n.Pos(), "the interpreter has no rule for the unary operator %s", n.Op)
		}
		return value{kind: kindBool, b: !in.eval(n.X, sc).b}
	case *ast.BinaryExpr:
		return in.binary(n, sc)
	case *ast.CallExpr:
		return in.callExpr(n, sc)
	}
	in.fail(e.Pos(), "the interpreter has no rule for the expression %T", e)
	return value{}
}

func (in *interp) ident(n *ast.Ident, sc scope) value {
	switch n.Name {
	case "true":
		return value{kind: kindBool, b: true}
	case "false":
		return value{kind: kindBool, b: false}
	}
	if v, ok := sc[n.Name]; ok {
		return v
	}
	if c, ok := in.info.Uses[n].(*types.Const); ok {
		return constToValue(c.Val())
	}
	in.fail(n.Pos(), "the interpreter cannot resolve %s", n.Name)
	return value{}
}

func (in *interp) constValue(n *ast.BasicLit) value {
	tv, ok := in.info.Types[n]
	if !ok || tv.Value == nil {
		in.fail(n.Pos(), "a literal with no constant value")
	}
	return constToValue(tv.Value)
}

func constToValue(c constant.Value) value {
	if c.Kind() == constant.Bool {
		return value{kind: kindBool, b: constant.BoolVal(c)}
	}
	i, _ := constant.Int64Val(c)
	return value{kind: kindInt, i: i}
}

func (in *interp) selector(n *ast.SelectorExpr, sc scope) value {
	recv := in.eval(n.X, sc)
	if recv.kind != kindStruct {
		in.fail(n.Pos(), "the interpreter reads fields of the predicate struct only")
	}
	name := n.Sel.Name
	if recv.fields.asked != nil && !contains(*recv.fields.asked, name) {
		*recv.fields.asked = append(*recv.fields.asked, name)
	}
	v, known := recv.fields.known[name]
	if !known {
		panic(unknownField{name: name})
	}
	return value{kind: kindBool, b: v}
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func (in *interp) binary(n *ast.BinaryExpr, sc scope) value {
	switch n.Op {
	case token.LAND:
		if !in.eval(n.X, sc).b {
			return value{kind: kindBool, b: false}
		}
		return value{kind: kindBool, b: in.eval(n.Y, sc).b}
	case token.LOR:
		if in.eval(n.X, sc).b {
			return value{kind: kindBool, b: true}
		}
		return value{kind: kindBool, b: in.eval(n.Y, sc).b}
	}
	x, y := in.eval(n.X, sc), in.eval(n.Y, sc)
	switch n.Op {
	case token.EQL:
		return value{kind: kindBool, b: eq(x, y)}
	case token.NEQ:
		return value{kind: kindBool, b: !eq(x, y)}
	case token.LSS:
		return value{kind: kindBool, b: x.i < y.i}
	case token.LEQ:
		return value{kind: kindBool, b: x.i <= y.i}
	case token.GTR:
		return value{kind: kindBool, b: x.i > y.i}
	case token.GEQ:
		return value{kind: kindBool, b: x.i >= y.i}
	case token.ADD:
		return value{kind: kindInt, i: x.i + y.i}
	case token.SUB:
		return value{kind: kindInt, i: x.i - y.i}
	}
	in.fail(n.OpPos, "the interpreter has no rule for the operator %s", n.Op)
	return value{}
}

func eq(x, y value) bool {
	if x.kind == kindBool || y.kind == kindBool {
		return x.b == y.b
	}
	return x.i == y.i
}

func (in *interp) callExpr(n *ast.CallExpr, sc scope) value {
	id, ok := n.Fun.(*ast.Ident)
	if !ok {
		in.fail(n.Pos(), "the interpreter calls named functions only")
		return value{}
	}
	fn, ok := in.funcs[id.Name]
	if !ok {
		in.fail(n.Pos(), "the interpreter has no function named %s", id.Name)
		return value{}
	}
	args := make([]value, 0, len(n.Args))
	for _, a := range n.Args {
		args = append(args, in.eval(a, sc))
	}
	return in.call(fn, args)
}
