package spbody

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"
)

func (e *emitter) expr(x ast.Expr) string {
	if lit, ok := e.folded(x); ok {
		return lit
	}
	switch n := x.(type) {
	case *ast.Ident:
		return e.identExpr(n)
	case *ast.BasicLit:
		return e.basicLit(n)
	case *ast.ParenExpr:
		return "(" + e.expr(n.X) + ")"
	case *ast.BinaryExpr:
		return e.binary(n)
	case *ast.UnaryExpr:
		return e.unary(n)
	case *ast.IndexExpr:
		return fmt.Sprintf("%s[%s]", e.expr(n.X), e.expr(n.Index))
	case *ast.SelectorExpr:
		return e.selector(n)
	case *ast.CallExpr:
		if e.isArrayValue(n) && !e.returnsArrayValue(n) {
			if tv, isType := e.info.Types[n.Fun]; !isType || !tv.IsType() {
				e.fail(n.Pos(), "a call returning an array used as a value; SourcePawn fills a parameter, so assign it to a name on a line of its own")
				return ""
			}
		}
		return e.callWith(n, nil)
	case *ast.CompositeLit:
		return e.compositeLit(n)
	default:
		e.fail(x.Pos(), "the expression %T has no SourcePawn", x)
		return ""
	}
}

// folded emits a constant expression as its value. A name is left alone so the
// generated code reads like the Go beside it; everything else the type checker
// could fold is folded, which is what spcomp would do anyway and what keeps
// len over an array and arithmetic over constants out of the output.
func (e *emitter) folded(x ast.Expr) (string, bool) {
	switch x.(type) {
	case *ast.Ident, *ast.SelectorExpr:
		return "", false
	}
	tv, ok := e.info.Types[x]
	if !ok || tv.Value == nil {
		return "", false
	}
	tag, dims, err := e.spType(tv.Type)
	if err != nil || len(dims) > 0 {
		return "", false
	}
	lit, err := literalOf(tv.Value, tag)
	if err != nil {
		e.fail(x.Pos(), "%v", err)
		return "", true
	}
	return lit, true
}

// identExpr writes a name, with the prefix if it is one this package declares
// at package level. A local keeps its own name.
func (e *emitter) identExpr(id *ast.Ident) string {
	if id.Name == "true" || id.Name == "false" {
		return id.Name
	}
	obj := e.info.Uses[id]
	if obj == nil {
		obj = e.info.Defs[id]
	}
	if obj != nil && obj.Parent() == e.pkg.Scope() {
		return e.cfg.Prefix + e.ident(id.Pos(), id.Name)
	}
	return e.ident(id.Pos(), id.Name)
}

func (e *emitter) basicLit(lit *ast.BasicLit) string {
	switch lit.Kind {
	case token.INT, token.CHAR:
		return lit.Value
	case token.STRING:
		return e.stringLit(lit)
	case token.FLOAT:
		tv := e.info.Types[lit]
		if tv.Value != nil {
			s, err := literalOf(tv.Value, "float")
			if err != nil {
				e.fail(lit.Pos(), "%v", err)
				return ""
			}
			return s
		}
		return lit.Value
	default:
		e.fail(lit.Pos(), "the literal %s has no SourcePawn", lit.Value)
		return ""
	}
}

/*
	stringLit is the one string the subset has

An entity classname, passed to an extern that wants one. Go and SourcePawn
escape differently, so rather than translate the escapes, anything that is not
plain printable text is refused: a classname is plain printable text, and a
generator that guessed wrong here would produce a plugin that compiles and looks
for the wrong entity.
*/
func (e *emitter) stringLit(lit *ast.BasicLit) string {
	text, err := strconv.Unquote(lit.Value)
	if err != nil {
		e.fail(lit.Pos(), "the string %s does not unquote: %v", lit.Value, err)
		return ""
	}
	for _, r := range text {
		if r < ' ' || r > '~' || r == '"' || r == '\\' {
			e.fail(lit.Pos(), "the string %s holds a character this package will not escape for SourcePawn; a classname is plain printable text", lit.Value)
			return ""
		}
	}
	return strconv.Quote(text)
}

func (e *emitter) binary(n *ast.BinaryExpr) string {
	if n.Op == token.AND_NOT {
		e.fail(n.OpPos, "the &^ operator; SourcePawn has no AND NOT, write x & ~y")
		return ""
	}
	return fmt.Sprintf("%s %s %s", e.expr(n.X), n.Op, e.expr(n.Y))
}

func (e *emitter) unary(n *ast.UnaryExpr) string {
	switch n.Op {
	case token.SUB, token.ADD, token.NOT:
		return n.Op.String() + e.expr(n.X)
	case token.XOR:
		return "~" + e.expr(n.X) // Go spells bitwise complement ^x
	default:
		e.fail(n.OpPos, "the unary operator %s has no SourcePawn", n.Op)
		return ""
	}
}

// selector is either a struct field, which SourcePawn spells the same way, or
// a name from the extern package, which is a call and is handled at the call.
func (e *emitter) selector(n *ast.SelectorExpr) string {
	if _, isExtern := e.externOf(n); isExtern {
		e.fail(n.Pos(), "%s used as a value; an extern is called, not passed", e.qualified(n))
		return ""
	}
	if e.info.Selections[n] == nil {
		e.fail(n.Pos(), "%s is not a field of anything this package declares", e.qualified(n))
		return ""
	}
	return fmt.Sprintf("%s.%s", e.expr(n.X), e.ident(n.Sel.Pos(), n.Sel.Name))
}

func (e *emitter) qualified(n *ast.SelectorExpr) string {
	if id, ok := n.X.(*ast.Ident); ok {
		return id.Name + "." + n.Sel.Name
	}
	return n.Sel.Name
}

func (e *emitter) globalExtern(call *ast.CallExpr) (Extern, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return Extern{}, false
	}
	x, isExtern := e.externOf(sel)
	return x, isExtern && x.Global
}

// returnsArrayValue says the call is an extern whose SourcePawn returns the
// array, which is the one array-valued expression that is not rewritten.
func (e *emitter) returnsArrayValue(call *ast.CallExpr) bool {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		x, isExtern := e.externOf(fun)
		return isExtern && x.ReturnsArray
	case *ast.Ident:
		return e.valueReturners[fun.Name]
	default:
		return false
	}
}

func (e *emitter) externOf(n *ast.SelectorExpr) (Extern, bool) {
	id, ok := n.X.(*ast.Ident)
	if !ok {
		return Extern{}, false
	}
	if _, isPkg := e.info.Uses[id].(*types.PkgName); !isPkg {
		return Extern{}, false
	}
	x, ok := e.cfg.Externs[id.Name+"."+n.Sel.Name]
	return x, ok
}

// callWith emits a call, appending extra arguments for the results after the
// first when the caller wanted them.
func (e *emitter) callWith(call *ast.CallExpr, extra []string) string {
	if tv, ok := e.info.Types[call.Fun]; ok && tv.IsType() {
		return e.conversion(call, tv.Type)
	}
	if name, ok := e.builtinHelper(call); ok {
		return name
	}
	if x, ok := e.globalExtern(call); ok {
		if len(call.Args) != 0 || len(extra) != 0 {
			e.fail(call.Pos(), "%s is a SourcePawn variable and takes no arguments", x.Func)
			return ""
		}
		return x.Func
	}
	args := make([]string, 0, len(call.Args)+len(extra))
	for _, a := range call.Args {
		args = append(args, e.expr(a))
	}
	name, lead, err := e.callee(call.Fun)
	if err != nil {
		e.fail(call.Pos(), "%v", err)
		return ""
	}
	if name == "" {
		return lead[0] // a builtin that folded into an expression
	}
	args = append(append(append([]string{}, lead...), args...), extra...)
	return fmt.Sprintf("%s(%s)", name, strings.Join(args, ", "))
}

/*
	builtinHelper writes out min and max

SourcePawn has neither, and a ternary at the call site evaluates an argument
twice, which changes what an argument holding an engine call does. So each one
used is emitted once as a stock and called, and the generated file has no
behaviour a reader has to hold in their head.
*/
func (e *emitter) builtinHelper(call *ast.CallExpr) (string, bool) {
	id, ok := call.Fun.(*ast.Ident)
	if !ok {
		return "", false
	}
	builtin, ok := e.info.Uses[id].(*types.Builtin)
	if !ok || (builtin.Name() != "min" && builtin.Name() != "max") {
		return "", false
	}
	if len(call.Args) != 2 {
		e.fail(call.Pos(), "%s of %d arguments; the emitted helper takes two", builtin.Name(), len(call.Args))
		return "", true
	}
	tag, dims, err := e.spType(e.info.Types[call].Type)
	if err != nil || len(dims) > 0 {
		e.fail(call.Pos(), "%s over %s has no SourcePawn", builtin.Name(), e.info.Types[call].Type)
		return "", true
	}
	op := "<"
	if builtin.Name() == "max" {
		op = ">"
	}
	name := fmt.Sprintf("%s%s_%s", e.cfg.Prefix, builtin.Name(), tag)
	e.helpers[name] = helper{tag: tag, op: op}
	return fmt.Sprintf("%s(%s, %s)", name, e.expr(call.Args[0]), e.expr(call.Args[1])), true
}

func (e *emitter) callee(fun ast.Expr) (name string, lead []string, err error) {
	switch f := fun.(type) {
	case *ast.ParenExpr:
		return e.callee(f.X)
	case *ast.SelectorExpr:
		x, ok := e.externOf(f)
		if !ok {
			return "", nil, fmt.Errorf("%s is not an extern this emission was given; add it to Config.Externs", e.qualified(f))
		}
		return x.Func, x.Lead, nil
	case *ast.Ident:
		if builtin, ok := e.info.Uses[f].(*types.Builtin); ok {
			return builtinCall(builtin.Name())
		}
		obj := e.info.Uses[f]
		if obj == nil || obj.Parent() != e.pkg.Scope() {
			return "", nil, fmt.Errorf("a call to %s, which this package does not declare", f.Name)
		}
		if name, claimed := e.spNames[f.Name]; claimed {
			return name, nil, nil
		}
		return e.cfg.Prefix + f.Name, nil, nil
	default:
		return "", nil, fmt.Errorf("a call to something that is not a name")
	}
}

// builtinCall is what is left after min, max and the constant folding that
// takes len over a fixed-length array.
func builtinCall(name string) (string, []string, error) {
	if name == "len" {
		return "", nil, fmt.Errorf("len of something that is not a fixed-length array")
	}
	return "", nil, fmt.Errorf("the builtin %s has no SourcePawn", name)
}

// conversion is where SourcePawn and Go disagree most quietly. Between cells it
// is a tag change and nothing else; across the int and float boundary it is a
// call, and Go truncates towards zero, which is RoundToZero.
func (e *emitter) conversion(call *ast.CallExpr, to types.Type) string {
	if len(call.Args) != 1 {
		e.fail(call.Pos(), "a conversion taking %d arguments", len(call.Args))
		return ""
	}
	inner := e.expr(call.Args[0])
	from := e.info.Types[call.Args[0]].Type
	toTag, dims, err := e.spType(to)
	if err != nil || len(dims) > 0 {
		e.fail(call.Pos(), "a conversion to %s has no SourcePawn", to)
		return ""
	}
	fromTag, _, err := e.spType(from)
	if err != nil {
		e.fail(call.Pos(), "a conversion from %s has no SourcePawn", from)
		return ""
	}
	switch {
	case fromTag == toTag:
		return inner
	case fromTag == "int" && toTag == "float":
		return fmt.Sprintf("float(%s)", inner)
	case fromTag == "float" && toTag == "int":
		return fmt.Sprintf("RoundToZero(%s)", inner)
	case fromTag == "float" || toTag == "float":
		e.fail(call.Pos(), "a conversion between %s and a tagged float", fromTag)
		return ""
	default:
		return fmt.Sprintf("view_as<%s>(%s)", toTag, inner)
	}
}

// compositeLit emits an array literal, which SourcePawn writes the same way. A
// struct has no literal there: its fields are assigned one at a time, and a
// generated assignment sequence in the middle of an expression is not something
// this package will invent.
func (e *emitter) compositeLit(n *ast.CompositeLit) string {
	t := e.info.Types[n].Type
	if t == nil {
		e.fail(n.Pos(), "a composite literal with no type")
		return ""
	}
	if _, isArray := types.Unalias(t).Underlying().(*types.Array); !isArray {
		e.fail(n.Pos(), "a composite literal of %s; SourcePawn has no struct literal, so declare the value and assign its fields", t)
		return ""
	}
	values := make([]string, 0, len(n.Elts))
	for _, elt := range n.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			e.fail(kv.Pos(), "an indexed array literal; write the elements in order")
			return ""
		}
		values = append(values, e.expr(elt))
	}
	return "{" + strings.Join(values, ", ") + "}"
}
