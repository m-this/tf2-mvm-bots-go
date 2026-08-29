package spgen

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
)

// The interpreter runs the same checked Go the translator emits, over a
// partial input: the free parameters are concrete and the struct of predicates
// is only partly known. Reading a field that is not known stops the run and
// names the field.
//
// That is what makes the lazy table faithful. The order the predicates come
// out in is the order this evaluation reaches them, which is the order the
// source asks them in, short circuits and all. Nothing is inferred from the
// answers: an ordering derived from the pure function alone could ask a
// predicate the shipped chain would never have asked, and three of the
// predicates in this package have side effects.

// value is one runtime value: a bool, an integer or the predicate struct.
type value struct {
	kind   valueKind
	b      bool
	i      int64
	fields *fieldSet
}

type valueKind int

const (
	kindBool valueKind = iota
	kindInt
	kindStruct
)

// fieldSet is the partly-known struct of predicates.
type fieldSet struct {
	known map[string]bool
	// asked records every field read, in order, with repeats collapsed, so a
	// full run yields the sequence of predicates the source asks for.
	asked *[]string
}

// unknownField is what a read of a field with no answer unwinds with.
type unknownField struct{ name string }

type interp struct {
	fset  *token.FileSet
	info  *types.Info
	pkg   *types.Package
	funcs map[string]*ast.FuncDecl
	depth int
}

// maxCallDepth bounds a run, because SourcePawn's stack is small and a
// recursive body would loop here first.
const maxCallDepth = 64

func newInterp(p *Package) *interp {
	in := &interp{fset: p.fset, info: p.info, pkg: p.pkg, funcs: map[string]*ast.FuncDecl{}}
	for _, f := range p.files {
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok {
				in.funcs[fd.Name.Name] = fd
			}
		}
	}
	return in
}

// run calls name with args. It returns the result, or the field the run
// stopped on when one of the predicates it reached had no answer.
func (in *interp) run(name string, args []value) (result value, stopped string, err error) {
	fn, ok := in.funcs[name]
	if !ok {
		return value{}, "", fmt.Errorf("spgen: %s is not a function of this package", name)
	}
	defer func() {
		switch r := recover().(type) {
		case nil:
		case unknownField:
			stopped = r.name
		case error:
			err = r
		default:
			panic(r)
		}
	}()
	return in.call(fn, args), "", nil
}

func (in *interp) fail(pos token.Pos, format string, a ...any) {
	panic(fmt.Errorf("%s: %s", in.fset.Position(pos), fmt.Sprintf(format, a...)))
}

type scope map[string]value

func (in *interp) call(fn *ast.FuncDecl, args []value) value {
	in.depth++
	defer func() { in.depth-- }()
	if in.depth > maxCallDepth {
		in.fail(fn.Pos(), "the call depth passed %d: a generated body must not recurse", maxCallDepth)
	}

	sc := scope{}
	i := 0
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			for _, n := range field.Names {
				if i >= len(args) {
					in.fail(fn.Pos(), "%s was called with %d arguments", fn.Name.Name, len(args))
				}
				sc[n.Name] = args[i]
				i++
			}
		}
	}
	out, returned := in.block(fn.Body, sc)
	if !returned && fn.Type.Results != nil {
		in.fail(fn.Pos(), "%s ran off the end without returning", fn.Name.Name)
	}
	return out
}

func (in *interp) block(b *ast.BlockStmt, sc scope) (value, bool) {
	for _, s := range b.List {
		if v, returned := in.stmt(s, sc); returned {
			return v, true
		}
	}
	return value{}, false
}

func (in *interp) stmt(s ast.Stmt, sc scope) (value, bool) {
	switch n := s.(type) {
	case nil, *ast.EmptyStmt:
	case *ast.BlockStmt:
		return in.block(n, sc)
	case *ast.ReturnStmt:
		if len(n.Results) != 1 {
			in.fail(n.Pos(), "the interpreter runs functions with exactly one result")
		}
		return in.eval(n.Results[0], sc), true
	case *ast.IfStmt:
		return in.ifStmt(n, sc)
	case *ast.SwitchStmt:
		return in.switchStmt(n, sc)
	case *ast.AssignStmt:
		in.assign(n, sc)
	default:
		in.fail(s.Pos(), "the interpreter has no rule for the statement %T", s)
	}
	return value{}, false
}

func (in *interp) ifStmt(n *ast.IfStmt, sc scope) (value, bool) {
	if in.eval(n.Cond, sc).b {
		return in.block(n.Body, sc)
	}
	if n.Else == nil {
		return value{}, false
	}
	return in.stmt(n.Else, sc)
}

func (in *interp) switchStmt(n *ast.SwitchStmt, sc scope) (value, bool) {
	tag := in.eval(n.Tag, sc)
	var def *ast.CaseClause
	for _, s := range n.Body.List {
		c := s.(*ast.CaseClause)
		if c.List == nil {
			def = c
			continue
		}
		for _, e := range c.List {
			if in.eval(e, sc).i == tag.i {
				return in.clause(c, sc)
			}
		}
	}
	if def != nil {
		return in.clause(def, sc)
	}
	return value{}, false
}

func (in *interp) clause(c *ast.CaseClause, sc scope) (value, bool) {
	for _, s := range c.Body {
		if v, returned := in.stmt(s, sc); returned {
			return v, true
		}
	}
	return value{}, false
}

func (in *interp) assign(n *ast.AssignStmt, sc scope) {
	if len(n.Lhs) != 1 || len(n.Rhs) != 1 || n.Tok != token.DEFINE && n.Tok != token.ASSIGN {
		in.fail(n.Pos(), "the interpreter has no rule for this assignment")
	}
	id, ok := n.Lhs[0].(*ast.Ident)
	if !ok {
		in.fail(n.Pos(), "the interpreter assigns to names only")
		return
	}
	sc[id.Name] = in.eval(n.Rhs[0], sc)
}

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
