package spbody

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

/*
	Package state

The plugin keeps its answers in globals: g_bIsDefenderBot, m_bTouchCredits and
several hundred more. A body that may not own state cannot be a port of a
function that reads one, so a package-level var here is that global, emitted as
a SourcePawn global with the same initial value.

What SourcePawn will not do is compute one at load, so the initialiser is a
constant or an array of constants and gosubset refuses the rest.

Putting them back between maps is the body's own job, written in Go as an
ordinary function and translated like any other. Generating it here would be a
function nobody proved; written there, the differential test walks it.
*/

// stateVar is one emitted global, kept so Reset can put it back.
type stateVar struct {
	name string
	tag  string
	dims []int64
}

func (e *emitter) varDecl(d *ast.GenDecl) {
	for _, spec := range d.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			e.fail(d.Pos(), "an unrecognised package-level declaration")
			continue
		}
		// A global keeps the plugin's name for the same reason a function
		// does: the files that have not been ported still read it.
		claimed, hasName := varName(vs, d)
		for i, name := range vs.Names {
			if name.Name == "_" {
				continue
			}
			var value ast.Expr
			if i < len(vs.Values) {
				value = vs.Values[i]
			}
			emitted := ""
			if hasName && len(vs.Names) == 1 {
				emitted = claimed
			}
			e.stateVar(name, value, emitted)
		}
	}
}

// varName reads //sp:name off a var declaration, from the spec's own doc or
// the group's when the group declares one variable.
func varName(vs *ast.ValueSpec, d *ast.GenDecl) (string, bool) {
	for _, doc := range []*ast.CommentGroup{vs.Doc, d.Doc} {
		if doc == nil {
			continue
		}
		for _, c := range doc.List {
			fields := strings.Fields(c.Text)
			if len(fields) == 2 && fields[0] == nameDirective {
				return fields[1], true
			}
		}
	}
	return "", false
}

/*
	stringTable is a fixed array of names, which SourcePawn spells char x[][]

The one shape of text the subset holds as state. The plugin has two of them, both
lists of entity classnames walked by index, and neither is ever written to: what
FindEntityByClassname wants is a real string and there is no id that would do
instead.

Read-only is the whole of the rule. The emitter has no way to write into one, so
a body that tries fails on the assignment rather than here.
*/
func (e *emitter) stringTable(name *ast.Ident, value ast.Expr, claimed string, t types.Type) bool {
	arr, ok := types.Unalias(t).(*types.Array)
	if !ok {
		return false
	}
	basic, ok := types.Unalias(arr.Elem()).(*types.Basic)
	if !ok || basic.Kind() != types.String {
		return false
	}
	if value == nil {
		e.fail(name.Pos(), "%s is a table of names with nothing in it", name.Name)
		return true
	}
	lit, ok := value.(*ast.CompositeLit)
	if !ok {
		e.fail(name.Pos(), "%s is a table of names and its value is not a literal", name.Name)
		return true
	}
	emitted := e.cfg.Prefix + e.ident(name.Pos(), name.Name)
	if claimed != "" {
		emitted = e.ident(name.Pos(), claimed)
	}
	names := make([]string, 0, len(lit.Elts))
	for _, el := range lit.Elts {
		text, ok := el.(*ast.BasicLit)
		if !ok || text.Kind != token.STRING {
			e.fail(el.Pos(), "%s holds something that is not a name", name.Name)
			return true
		}
		names = append(names, text.Value)
	}
	e.state = append(e.state, stateVar{name: emitted, tag: "char", dims: []int64{arr.Len(), 0}})
	e.line("static char %s[][] =", emitted)
	e.line("{")
	for _, n := range names {
		e.line("\t%s,", n)
	}
	e.line("};")
	return true
}

func (e *emitter) stateVar(name *ast.Ident, value ast.Expr, claimed string) {
	obj := e.info.Defs[name]
	if obj == nil {
		e.fail(name.Pos(), "the variable %s has no type", name.Name)
		return
	}
	if e.stringTable(name, value, claimed, obj.Type()) {
		return
	}
	tag, dims, err := e.spType(obj.Type())
	if err != nil {
		e.fail(name.Pos(), "%s: %v", name.Name, err)
		return
	}
	if len(dims) > 3 {
		// Three is the plugin's deepest: a vector, per attempt, per
		// client, which is the teleporter's route out of spawn. Deeper
		// than that is a shape nobody has written and almost certainly
		// a mistake.
		e.fail(name.Pos(), "%s is a global of more than three dimensions; the plugin has none, so this is almost certainly a mistake", name.Name)
		return
	}
	if len(dims) == 2 && value != nil {
		// A two dimensional global is a vector per client, and the plugin
		// never initialises one. Emitting the nested literal is work with
		// nothing asking for it.
		e.fail(name.Pos(), "%s is a two dimensional global with an initialiser, which nothing emits yet", name.Name)
		return
	}
	emitted := e.cfg.Prefix + e.ident(name.Pos(), name.Name)
	if claimed != "" {
		emitted = e.ident(name.Pos(), claimed)
	}
	v := stateVar{name: emitted, tag: tag, dims: dims}
	e.state = append(e.state, v)

	decl := declare(v.tag, v.name, v.dims)
	if value == nil {
		e.line("%s;", decl)
		return
	}
	elems := e.staticValues(value, len(dims) == 1)
	if len(dims) == 0 {
		e.line("%s = %s;", decl, elems[0])
		return
	}
	e.line("%s = {%s};", decl, strings.Join(elems, ", "))
}

// staticValues folds the initialiser into one literal per element. An array
// literal shorter than the array leaves the rest at the zero value, which is
// what both languages do.
func (e *emitter) staticValues(value ast.Expr, isArray bool) []string {
	if !isArray {
		return []string{e.expr(value)}
	}
	lit, ok := value.(*ast.CompositeLit)
	if !ok {
		e.fail(value.Pos(), "an array global initialised by something that is not a literal")
		return nil
	}
	out := make([]string, 0, len(lit.Elts))
	for _, elt := range lit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			e.fail(kv.Pos(), "an indexed array literal; write the elements in order")
			return nil
		}
		out = append(out, e.expr(elt))
	}
	return out
}

// zeroOf is the value a declaration with no initialiser holds, which is what Go
// gives a variable and what SourcePawn gives a global.
func zeroOf(tag string) string {
	switch tag {
	case "bool":
		return "false"
	case "float":
		return "0.0"
	case "int":
		return "0"
	default:
		return fmt.Sprintf("view_as<%s>(0)", tag)
	}
}
