package spbody

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/m-this/tf2-mvm-bots-go/internal/sp"
)

func floatLiteral(v float32) string { return sp.FloatLiteral(v) }

// reserved are the SourcePawn keywords a Go identifier may legally be and a
// SourcePawn one may not. Hitting one is a compile error in the generated file,
// which is a worse place to read it than here.
var reserved = map[string]bool{
	"any": true, "assert": true, "break": true, "case": true, "cast_to": true,
	"catch": true, "cellsof": true, "char": true, "const": true, "continue": true,
	"decl": true, "default": true, "defined": true, "delete": true, "do": true,
	"else": true, "enum": true, "explicit": true, "false": true, "finally": true,
	"for": true, "forward": true, "funcenum": true, "functag": true, "function": true,
	"goto": true, "if": true, "implicit": true, "import": true, "int": true,
	"intn": true, "let": true, "methodmap": true, "namespace": true, "native": true,
	"new": true, "null": true, "object": true, "operator": true, "package": true,
	"private": true, "property": true, "protected": true, "public": true,
	"readonly": true, "return": true, "sealed": true, "sizeof": true, "sleep": true,
	"static": true, "stock": true, "struct": true, "switch": true, "this": true,
	"throw": true, "true": true, "try": true, "typedef": true, "typeof": true,
	"typeset": true, "union": true, "using": true, "var": true, "view_as": true,
	"virtual": true, "void": true, "volatile": true, "while": true, "with": true,
}

// ident passes a Go name through, and refuses one SourcePawn will not take.
// Renaming it here would make the generated code disagree with the Go a reader
// has open beside it.
func (e *emitter) ident(pos token.Pos, name string) string {
	if reserved[name] {
		e.fail(pos, "%s is a SourcePawn keyword; rename it in the Go", name)
	}
	return name
}

// usesResult says whether the body needs the named result declared: it says the
// name, or it returns without one and the name is what comes back.
func usesResult(body *ast.BlockStmt, name string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.ReturnStmt:
			if len(t.Results) == 0 {
				found = true
			}
		case *ast.Ident:
			if t.Name == name {
				found = true
			}
		}
		return !found
	})
	return found
}

// endsInReturn says whether control can fall off the end of the body, which is
// the one way out a return statement does not cover.
func endsInReturn(body *ast.BlockStmt) bool {
	if len(body.List) == 0 {
		return false
	}
	_, ok := body.List[len(body.List)-1].(*ast.ReturnStmt)
	return ok
}

// outParam is a Go result that SourcePawn takes as a parameter.
type outParam struct {
	name string
	tag  string
	dims []int64
}

// zero clears an out parameter, which is what Go does to a named result.
func (e *emitter) zero(out outParam) {
	if len(out.dims) == 0 {
		e.line("%s = %s;", out.name, zeroOf(out.tag))
		return
	}
	e.line("for (int i = 0; i < %d; i++)", out.dims[0])
	e.line("{")
	e.indent++
	inner := out.name + "[i]"
	if len(out.dims) > 1 {
		e.fail(token.NoPos, "an out parameter of more than one dimension")
		return
	}
	e.line("%s = %s;", inner, zeroOf(out.tag))
	e.indent--
	e.line("}")
}

// emittedName is what the function is called in SourcePawn: the plugin's own
// name when the body claims one, and the prefixed Go name otherwise.
func (e *emitter) emittedName(d *ast.FuncDecl) string {
	if name, claimed := e.spNames[d.Name.Name]; claimed {
		return e.ident(d.Name.Pos(), name)
	}
	return e.cfg.Prefix + e.ident(d.Name.Pos(), d.Name.Name)
}

func (e *emitter) funcDecl(d *ast.FuncDecl) {
	obj, ok := e.info.Defs[d.Name].(*types.Func)
	if !ok {
		e.fail(d.Pos(), "the function %s has no definition", d.Name.Name)
		return
	}
	sig := obj.Type().(*types.Signature)
	if sig.Recv() != nil {
		e.fail(d.Pos(), "a method; write a plain function taking the receiver first")
		return
	}
	e.returnsValue = returnsArray(d)
	ret, params, err := e.signature(d, sig)
	if err != nil {
		e.fail(d.Pos(), "%v", err)
		return
	}
	e.byRef = byRefParams(sig)
	name := e.emittedName(d)
	e.emitted = append(e.emitted, name)
	switch decl, given := e.cfg.Declare[d.Name.Name]; {
	case given:
		e.line("%s", decl)
	case isPublic(d):
		e.line("public %s %s(%s)", ret, name, strings.Join(params, ", "))
	default:
		e.line("stock %s %s(%s)", ret, name, strings.Join(params, ", "))
	}
	e.line("{")
	e.indent++
	// A named first result is a local in SourcePawn, and it is only needed if
	// the body says its name or returns without one. Declaring it regardless
	// left an unused variable in every callback that names its result and
	// always returns a value.
	if e.resultName != "" && usesResult(d.Body, e.resultName) {
		e.line("%s;", e.resultDecl)
	} else if e.resultName != "" {
		e.resultName = ""
	}
	// Go zeroes a named result and SourcePawn hands the body whatever the
	// caller's variable held, so the out parameters are cleared here. A
	// body that reads one before writing it would otherwise see two
	// different values in the two languages.
	for _, out := range e.outParams {
		e.zero(out)
	}
	e.pending = nil
	e.checkClosed(d)
	for _, s := range d.Body.List {
		e.stmt(s)
	}
	// Falling off the end is a way out too.
	if !endsInReturn(d.Body) {
		e.discharge(d.Body.End(), nil)
	}
	e.pending = nil
	e.indent--
	e.line("}")
	e.blank()
	e.outParams, e.byRef = nil, nil
	e.resultName, e.resultDecl = "", ""
	e.returnsValue = false
	if hasDHook(d) {
		e.dhookWrapper(d, sig)
	}
}

// signature turns the Go signature into SourcePawn's. The first result is the
// return; the ones after it become by-reference parameters, in order, which is
// what a SourcePawn caller of a function with two answers writes anyway.
func (e *emitter) signature(d *ast.FuncDecl, sig *types.Signature) (ret string, params []string, err error) {
	ret = "void"
	results := sig.Results()
	e.resultName, e.resultDecl = "", ""
	e.returnsArray = false
	first := 0
	if results.Len() > 0 {
		r := results.At(0)
		tag, dims, terr := e.spType(r.Type())
		if terr != nil {
			return "", nil, terr
		}
		switch {
		case len(dims) > 0 && !e.returnsValue:
			// SourcePawn returns a cell, so an array result is a
			// parameter the caller supplies and the body fills.
			// That is the idiom the plugin already writes.
			e.returnsArray = true
		case len(dims) > 0:
			// The float[] form, asked for by //sp:returns. The
			// result is an ordinary local here and the caller gets
			// a copy.
			ret = tag + "[]"
			first = 1
			name := r.Name()
			if name == "" || name == "_" {
				return "", nil, errUnnamedResult
			}
			e.resultName = e.ident(d.Pos(), name)
			e.resultDecl = declare(tag, e.resultName, dims)
		default:
			ret = tag
			first = 1
			if name := r.Name(); name != "" && name != "_" {
				e.resultName = e.ident(d.Pos(), name)
				e.resultDecl = declare(tag, e.resultName, nil)
			}
		}
	}
	var names []string
	for i := range sig.Params().Len() {
		p := sig.Params().At(i)
		name := p.Name()
		if name == "" || name == "_" {
			return "", nil, errUnnamedParam
		}
		// Text a function is given rather than one it owns: SourcePawn
		// takes it as const char[], with no length, because the body may
		// only read it. Anything that writes into a buffer takes a Text
		// and its size, which is a different parameter.
		if b, ok := types.Unalias(p.Type()).(*types.Basic); ok && b.Kind() == types.String {
			names = append(names, name)
			params = append(params, "const char[] "+e.ident(d.Pos(), name))
			continue
		}
		tag, dims, terr := e.spType(p.Type())
		if terr != nil {
			return "", nil, terr
		}
		names = append(names, name)
		if tag == "char" && len(dims) == 1 {
			// A buffer a function is handed is const char[]: the
			// length belongs to whoever declared it, and writing
			// this port's own 512 here refuses every caller whose
			// buffer is the schema's length instead. Writing into
			// one needs that length, which is the gap mvm-z83.62
			// carries.
			params = append(params, "const char[] "+e.ident(d.Pos(), name))
			continue
		}
		params = append(params, declare(tag, e.ident(d.Pos(), name), dims))
	}
	params = e.applyDefaults(d, names, params, e.defaultsOf(d))
	e.outParams = nil
	for i := first; i < results.Len(); i++ {
		r := results.At(i)
		if r.Name() == "" || r.Name() == "_" {
			return "", nil, errUnnamedResult
		}
		tag, dims, terr := e.spType(r.Type())
		if terr != nil {
			return "", nil, terr
		}
		// An array is by reference in SourcePawn already, and & in
		// front of one is not a declaration it takes.
		name := e.ident(d.Pos(), r.Name())
		e.outParams = append(e.outParams, outParam{name: name, tag: tag, dims: dims})
		if len(dims) == 0 {
			name = "&" + name
		}
		params = append(params, declare(tag, name, dims))
	}
	return ret, params, nil
}

// byRefParams are the parameters SourcePawn passes by reference no matter what
// the Go says. Writing to one is the one place a faithful translation is not
// possible, so it is refused where it is written rather than left to differ.
func byRefParams(sig *types.Signature) map[string]bool {
	out := make(map[string]bool)
	for i := range sig.Params().Len() {
		p := sig.Params().At(i)
		switch types.Unalias(p.Type()).Underlying().(type) {
		case *types.Array, *types.Struct:
			out[p.Name()] = true
		}
	}
	return out
}
