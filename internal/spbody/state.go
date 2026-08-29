package spbody

import (
	"fmt"
	"go/ast"
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
		for i, name := range vs.Names {
			if name.Name == "_" {
				continue
			}
			var value ast.Expr
			if i < len(vs.Values) {
				value = vs.Values[i]
			}
			e.stateVar(name, value)
		}
	}
}

func (e *emitter) stateVar(name *ast.Ident, value ast.Expr) {
	obj := e.info.Defs[name]
	if obj == nil {
		e.fail(name.Pos(), "the variable %s has no type", name.Name)
		return
	}
	tag, dims, err := e.spType(obj.Type())
	if err != nil {
		e.fail(name.Pos(), "%s: %v", name.Name, err)
		return
	}
	if len(dims) > 1 {
		e.fail(name.Pos(), "%s is a global of more than one dimension; Reset writes one loop, so declare it flat", name.Name)
		return
	}
	v := stateVar{name: e.cfg.Prefix + e.ident(name.Pos(), name.Name), tag: tag, dims: dims}
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
