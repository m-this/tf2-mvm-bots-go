package spbody

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
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
		if _, isGlobal := e.globalExtern(n); isGlobal {
			return e.callWith(n, nil)
		}
		if _, isSlot := e.slotExtern(n); isSlot {
			// A slot is a subscript, so an array one is an array
			// expression and not a call that fills a parameter.
			return e.callWith(n, nil)
		}
		if x, isExtern := e.externOfCall(n); isExtern && (x.Choice || x.Cast || x.Same) {
			// Neither is a call: one is ?: and the other view_as, so
			// an array coming out of one is the array that went in.
			return e.callWith(n, nil)
		}
		if _, _, isProperty := e.propertyExtern(n); isProperty {
			// A field read is a value whatever its shape: an array one
			// is the plugin's own g_arr[i].vecField.
			return e.callWith(n, nil)
		}
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
		if name, claimed := e.spNames[id.Name]; claimed {
			return e.ident(id.Pos(), name)
		}
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
	stringLit is a string the subset passes through

An entity classname or a message, passed to an extern that wants one. Go and
SourcePawn escape the same two characters the same way and differ on everything
else, so a quote and a backslash pass and anything outside printable ASCII is
refused: a generator that guessed wrong here would produce a plugin that compiles
and looks for the wrong entity.
*/
func (e *emitter) stringLit(lit *ast.BasicLit) string {
	text, err := strconv.Unquote(lit.Value)
	if err != nil {
		e.fail(lit.Pos(), "the string %s does not unquote: %v", lit.Value, err)
		return ""
	}
	for _, r := range text {
		/* A quote and a backslash are spelled the same way in both
		languages, and strconv.Quote below writes exactly that, so they
		pass. So do a newline, a tab and a carriage return: SourcePawn
		writes \n, \t and \r for those and means the same thing, and a
		menu title with a line in it needs the first.

		Anything else outside printable ASCII does not pass: the escapes
		differ there, and a generator that guessed would produce a plugin
		that compiles and looks for the wrong thing. */
		if r == '\n' || r == '\t' || r == '\r' {
			continue
		}
		if r < ' ' || r > '~' {
			e.fail(lit.Pos(), "the string %s holds a character this package will not escape for SourcePawn; a message is plain printable text", lit.Value)
			return ""
		}
	}
	return strconv.Quote(text)
}

/*
	binary, and the parentheses that are not optional

Go and SourcePawn do not agree on precedence, and the disagreement was measured
rather than assumed, because guessing it from C got it wrong twice. With
a, b, c = 3, 5, 2:

	a + b << c    Go 23, spcomp 32
	a | b ^ c     Go 5,  spcomp 7
	a & b | c     both 3
	flags & mask == 0   both the same, spcomp binds & tighter than ==, unlike C

Two of those compile in either language and answer differently, which is the
worst kind of wrong this generator can be. Rather than carry a table of which
pairs agree, an operand that is itself a binary expression with a different
operator is parenthesised, always. The grouping then says what the Go said. Same
operator on both sides needs nothing: it groups the same way either way, and
a + b + c reads better without.
*/
func (e *emitter) binary(n *ast.BinaryExpr) string {
	if n.Op == token.AND_NOT {
		e.fail(n.OpPos, "the &^ operator; SourcePawn has no AND NOT, write x & ~y")
		return ""
	}
	return fmt.Sprintf("%s %s %s", e.operand(n.X, n.Op), n.Op, e.operand(n.Y, n.Op))
}

func (e *emitter) operand(x ast.Expr, parent token.Token) string {
	text := e.expr(x)
	if inner, ok := x.(*ast.BinaryExpr); ok && inner.Op != parent {
		return "(" + text + ")"
	}
	return text
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
	if _, _, isMethod := e.externMethod(n); isMethod {
		e.fail(n.Pos(), "%s used as a value; an extern is called, not passed", e.qualified(n))
		return ""
	}
	sel := e.info.Selections[n]
	if sel == nil {
		e.fail(n.Pos(), "%s is not a field of anything this package declares", e.qualified(n))
		return ""
	}
	return fmt.Sprintf("%s.%s", e.expr(n.X), e.fieldName(n, sel))
}

/*
	fieldName is what SourcePawn calls the field

A struct the extern package declares stands for one the plugin already has, and
the plugin's field names are not Go's: BombInfo_t has vPosition, which Go cannot
export. The Go field carries the SourcePawn name in a struct tag, which is what
struct tags are for.
*/
func (e *emitter) fieldName(n *ast.SelectorExpr, sel *types.Selection) string {
	if len(sel.Index()) == 1 {
		if st, ok := sel.Recv().Underlying().(*types.Struct); ok {
			i := sel.Index()[0]
			if name, ok := reflect.StructTag(st.Tag(i)).Lookup("sp"); ok {
				return name
			}
		}
	}
	return e.ident(n.Sel.Pos(), n.Sel.Name)
}

func (e *emitter) qualified(n *ast.SelectorExpr) string {
	if id, ok := n.X.(*ast.Ident); ok {
		return id.Name + "." + n.Sel.Name
	}
	return n.Sel.Name
}

/*
	slot is a plugin array indexed by the actor

The bot state the behaviours share lives in nextbot_behavior.sp as arrays over
client slots, and a behaviour both reads and writes it. Go has no way to say
"assign to the result of a call", so the read and the write are two declarations
and this emits the subscript for both.

It is transitional by construction. When that state moves here it becomes
ordinary package state, the two declarations go, and the subscript is the
generator's own.
*/
func (e *emitter) slot(call *ast.CallExpr, x Extern) string {
	want := 1
	if x.Set {
		want = 2
	}
	if len(call.Args) != want {
		e.fail(call.Pos(), "%s takes %d argument(s), and was given %d", x.Func, want, len(call.Args))
		return ""
	}
	read := fmt.Sprintf("%s[%s]", x.Func, e.expr(call.Args[0]))
	if !x.Set {
		return read
	}
	return fmt.Sprintf("%s = %s", read, e.expr(call.Args[1]))
}

// propertyExtern is a read written without parentheses, which is what
// SourcePawn calls a property and Go has no form for but a method.
func (e *emitter) propertyExtern(call *ast.CallExpr) (Extern, string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return Extern{}, "", false
	}
	x, recv, isMethod := e.externMethod(sel)
	return x, recv, isMethod && x.Property
}

func (e *emitter) slotExtern(call *ast.CallExpr) (Extern, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return Extern{}, false
	}
	x, isExtern := e.externOf(sel)
	return x, isExtern && x.Slot
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
		if x, isExtern := e.externOf(fun); isExtern {
			return x.ReturnsArray
		}
		x, _, isMethod := e.externMethod(fun)
		return isMethod && x.ReturnsArray
	case *ast.Ident:
		return e.valueReturners[fun.Name]
	default:
		return false
	}
}

/*
	externMethod resolves a call written on a receiver

The receiver's type is what picks the method, so this asks go/types what the
expression on the left is rather than reading the name. A type from the extern
package carries a //sp:tag saying what SourcePawn calls it, and the method
carries its own directive; anything else is not an extern and is refused where
it is written.
*/
func (e *emitter) externMethod(n *ast.SelectorExpr) (Extern, string, bool) {
	tv, ok := e.info.Types[n.X]
	if !ok || tv.Type == nil {
		return Extern{}, "", false
	}
	named, ok := types.Unalias(tv.Type).(*types.Named)
	if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg() == e.pkg {
		return Extern{}, "", false
	}
	key := named.Obj().Pkg().Name() + "." + named.Obj().Name() + "." + n.Sel.Name
	x, declared := e.cfg.Externs[key]
	if !declared || !x.Method {
		return Extern{}, "", false
	}
	return x, e.expr(n.X), true
}

// externOf2 is externOf for a call site, which is where the package-level
// externs that are not plain calls have to be recognised.
func (e *emitter) externOf2(call *ast.CallExpr) (Extern, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return Extern{}, false
	}
	return e.externOf(sel)
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
// callWith emits a call, appending extra arguments for the results after the
// first when the caller wanted them, and putting lead ones in front.
func (e *emitter) callWith(call *ast.CallExpr, extra []string, front ...string) string {
	if tv, ok := e.info.Types[call.Fun]; ok && tv.IsType() {
		return e.conversion(call, tv.Type)
	}
	if name, ok := e.builtinHelper(call); ok {
		return name
	}
	if x, recv, ok := e.propertyExtern(call); ok {
		if x.Set {
			if len(call.Args) != 1 {
				e.fail(call.Pos(), "%s is a property being written and takes one value", x.Func)
				return ""
			}
			return fmt.Sprintf("%s.%s = %s", recv, x.Func, e.expr(call.Args[0]))
		}
		if len(call.Args) != 0 {
			e.fail(call.Pos(), "%s is a property and takes no arguments", x.Func)
			return ""
		}
		return recv + "." + x.Func
	}
	if x, ok := e.externOf2(call); ok && x.Cast {
		if len(call.Args) != 1 {
			e.fail(call.Pos(), "%s is a tag change and takes one value", x.Func)
			return ""
		}
		return fmt.Sprintf("view_as<%s>(%s)", x.Func, e.expr(call.Args[0]))
	}
	if x, ok := e.externOf2(call); ok && x.Same {
		if len(call.Args) != 1 {
			e.fail(call.Pos(), "%s is the value itself and takes one argument", x.Func)
			return ""
		}
		return e.expr(call.Args[0])
	}
	if x, ok := e.externOf2(call); ok && x.Choice {
		if len(call.Args) != 3 {
			e.fail(call.Pos(), "%s is SourcePawn's ?: and takes a condition and two values", x.Func)
			return ""
		}
		// Always parenthesised: ?: binds looser than everything around it,
		// so "charge < cond ? a : b" is (charge < cond) ? a : b, which is
		// not what the Go says and not what the plugin wrote.
		return fmt.Sprintf("(%s ? %s : %s)", e.expr(call.Args[0]), e.expr(call.Args[1]), e.expr(call.Args[2]))
	}
	if x, ok := e.slotExtern(call); ok {
		return e.slot(call, x)
	}
	if x, ok := e.globalExtern(call); ok && x.Set {
		if len(call.Args) != 1 || len(extra) != 0 {
			e.fail(call.Pos(), "%s is a SourcePawn variable being written and takes one value", x.Func)
			return ""
		}
		return fmt.Sprintf("%s = %s", x.Func, e.expr(call.Args[0]))
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
	var trail []string
	if x, ok := e.externOfCall(call); ok {
		trail = x.Trail
	}
	if err != nil {
		e.fail(call.Pos(), "%v", err)
		return ""
	}
	if name == "" {
		return lead[0] // a builtin that folded into an expression
	}
	/* The same name cannot be both an argument and a destination, for a
	function this port generates

	SourcePawn passes an array by reference, and a generated function zeroes
	its out-parameters before it reads anything. So a call that hands one
	variable in and takes the answer back out through it reads zeros: the
	nest position was computed from the map origin for one release because of
	exactly this, and it compiled and ran and lost a wave.

	A native is a different matter and is left alone. NormalizeVector(away,
	away) is what SourceMod documents and what the plugin writes, because the
	native reads its input before it writes its output. This port cannot make
	that promise about its own functions, so it refuses to rely on it. */
	if e.generates(call) {
		for _, out := range extra {
			for _, in := range args {
				if out == in {
					e.fail(call.Pos(), "%s is passed to %s as an argument and taken back as a result, and a generated function zeroes its results before reading its arguments; give the result its own name", out, name)
				}
			}
		}
	}
	args = append(append(append(append(append([]string{}, lead...), front...), args...), extra...), trail...)
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

// generates says the call goes to a function this port emits: one of this
// package's own, or an extern marked //sp:body because another package emits it.
func (e *emitter) generates(call *ast.CallExpr) bool {
	if x, isExtern := e.externOfCall(call); isExtern {
		return x.Body
	}
	id, plain := call.Fun.(*ast.Ident)
	if !plain {
		return false
	}
	obj := e.info.Uses[id]
	return obj != nil && obj.Parent() == e.pkg.Scope()
}

func (e *emitter) callee(fun ast.Expr) (name string, lead []string, err error) {
	switch f := fun.(type) {
	case *ast.ParenExpr:
		return e.callee(f.X)
	case *ast.SelectorExpr:
		if x, ok := e.externOf(f); ok {
			return x.Func, x.Lead, nil
		}
		if x, recv, ok := e.externMethod(f); ok {
			// SourceMod's API is methodmaps, so this one is written
			// on its receiver and there is no plain function behind
			// it to call instead.
			return recv + "." + x.Func, x.Lead, nil
		}
		return "", nil, fmt.Errorf("%s is not an extern this emission was given; add it to internal/engine", e.qualified(f))
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
