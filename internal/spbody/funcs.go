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
	ret, params, err := e.signature(d, sig)
	if err != nil {
		e.fail(d.Pos(), "%v", err)
		return
	}
	e.byRef = byRefParams(sig)
	e.line("stock %s %s%s(%s)", ret, e.cfg.Prefix, e.ident(d.Name.Pos(), d.Name.Name), strings.Join(params, ", "))
	e.line("{")
	e.indent++
	// A named first result is a local in SourcePawn, declared before the
	// body so a naked return has something to return.
	if e.resultName != "" {
		e.line("%s;", e.resultDecl)
	}
	for _, s := range d.Body.List {
		e.stmt(s)
	}
	e.indent--
	e.line("}")
	e.blank()
	e.outParams, e.byRef = nil, nil
	e.resultName, e.resultDecl = "", ""
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
	if results.Len() > 0 {
		first := results.At(0)
		tag, dims, terr := e.spType(first.Type())
		if terr != nil {
			return "", nil, terr
		}
		if len(dims) > 0 {
			return "", nil, errReturnsArray
		}
		ret = tag
		if name := first.Name(); name != "" && name != "_" {
			e.resultName = e.ident(d.Pos(), name)
			e.resultDecl = declare(tag, e.resultName, nil)
		}
	}
	for i := range sig.Params().Len() {
		p := sig.Params().At(i)
		tag, dims, terr := e.spType(p.Type())
		if terr != nil {
			return "", nil, terr
		}
		name := p.Name()
		if name == "" || name == "_" {
			return "", nil, errUnnamedParam
		}
		params = append(params, declare(tag, e.ident(d.Pos(), name), dims))
	}
	e.outParams = nil
	for i := 1; i < results.Len(); i++ {
		r := results.At(i)
		if r.Name() == "" || r.Name() == "_" {
			return "", nil, errUnnamedResult
		}
		tag, dims, terr := e.spType(r.Type())
		if terr != nil {
			return "", nil, terr
		}
		e.outParams = append(e.outParams, r.Name())
		params = append(params, declare(tag, "&"+e.ident(d.Pos(), r.Name()), dims))
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
