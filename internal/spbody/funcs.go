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
	// record says the result is an enum struct, which is array-like: it
	// takes no & and there is nothing sensible to zero it with.
	record bool
}

// zero clears an out parameter, which is what Go does to a named result.
func (e *emitter) zero(out outParam) {
	if out.record {
		// The shipped GetBombInfo writes every field before it returns
		// true and leaves the record alone when it returns false.
		return
	}
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

// isEnumStruct says the type is a named struct, which SourcePawn spells as an
// enum struct and passes the way it passes an array.
func isEnumStruct(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	_, isStruct := named.Underlying().(*types.Struct)
	return isStruct
}

/*
	mutatesDirective marks a parameter the body writes through

SourcePawn passes an array by reference and Go copies one, so a body that writes a
parameter means different things in the two languages: the emitted SourcePawn
changes the caller's array and the Go changes its own copy. Every such write is
refused for that reason.

The exception is a native that works in place. VMX_VectorNormalize hands its
vector to ScaleVector, which scales the caller's array, and the shipped file
depends on that. The directive says so at the one function that needs it, and the
Go there is not a description of what the SourcePawn does to its caller.
*/
const mutatesDirective = "//sp:mutates"

/*
	byrefDirective marks a scalar parameter the caller reads back

An array is by reference in SourcePawn already; a float or an int is not, and the
plugin says so with an & in the declaration. IsPathToVectorPossible is the one
that needs it: the caller passes a float it wants the path's length written into,
and the parameter carries a default as well, which a result cannot.

The write goes to the caller either way, so this also says the write is meant.
*/
const byrefDirective = "//sp:byref"

// byrefsOf reads //sp:byref <parameter> off a declaration.
func byrefsOf(d *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	if d.Doc == nil {
		return out
	}
	for _, c := range d.Doc.List {
		for line := range strings.Lines(c.Text) {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == byrefDirective {
				out[fields[1]] = true
			}
		}
	}
	return out
}

// mutatesOf reads //sp:mutates <parameter> off a declaration.
func mutatesOf(d *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	if d.Doc == nil {
		return out
	}
	for _, c := range d.Doc.List {
		for line := range strings.Lines(c.Text) {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == mutatesDirective {
				out[fields[1]] = true
			}
		}
	}
	return out
}

// writableDirective marks a text parameter the shipped declaration leaves
// writable, which a caller passing its own writable buffer depends on.
const writableDirective = "//sp:writable"

// writablesOf reads //sp:writable <parameter> off a declaration.
func writablesOf(d *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	if d.Doc == nil {
		return out
	}
	for _, c := range d.Doc.List {
		for line := range strings.Lines(c.Text) {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == writableDirective {
				out[fields[1]] = true
			}
		}
	}
	return out
}

// lengthDirective names the parameter carrying a buffer parameter's length.
/*
	dimDirective spells an array parameter's dimension

SourcePawn compares a callback against its typedef by prototype, dimensions
included: AddNormalSoundHook takes a NormalSHook and refuses one declared
int clients[101] where the typedef says int clients[MAXPLAYERS], even though the
define is 101. Go has to name a number to have a type, so this says what to
write in its place.
*/
const dimDirective = "//sp:dim"

// dimsOf reads //sp:dim <parameter> <spelling> off a declaration.
func dimsOf(d *ast.FuncDecl) map[string]string {
	out := map[string]string{}
	if d.Doc == nil {
		return out
	}
	for _, c := range d.Doc.List {
		for line := range strings.Lines(c.Text) {
			fields := strings.Fields(line)
			if len(fields) == 3 && fields[0] == dimDirective {
				out[fields[1]] = fields[2]
			}
		}
	}
	return out
}

const lengthDirective = "//sp:length"

/*
	constDirective marks a parameter the callee may not write

SourcePawn's const is a promise to the caller, and it has to be exact in both
directions: a const array cannot be handed to a native that declares its
parameter writable, and a caller holding a const array cannot pass it to a
function that does not promise. Neither always-const nor never-const compiles
across the plugin, so the port says which, and the comparison against the shipped
declaration is what checks it.
*/
const constDirective = "//sp:const"

// constsOf reads //sp:const <parameter> off a declaration.
func constsOf(d *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	if d.Doc == nil {
		return out
	}
	for _, c := range d.Doc.List {
		for line := range strings.Lines(c.Text) {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == constDirective {
				out[fields[1]] = true
			}
		}
	}
	return out
}

// lengthsOf reads //sp:length <buffer> <parameter> off a declaration.
func lengthsOf(d *ast.FuncDecl) map[string]string {
	out := map[string]string{}
	if d.Doc == nil {
		return out
	}
	for _, c := range d.Doc.List {
		for line := range strings.Lines(c.Text) {
			fields := strings.Fields(line)
			if len(fields) == 3 && fields[0] == lengthDirective {
				out[fields[1]] = fields[2]
			}
		}
	}
	return out
}

func (e *emitter) funcDecl(d *ast.FuncDecl) {
	obj, ok := e.info.Defs[d.Name].(*types.Func)
	if !ok {
		e.fail(d.Pos(), "the function %s has no definition", d.Name.Name)
		return
	}
	e.lengths = lengthsOf(d)
	e.consts = constsOf(d)
	e.mutates = mutatesOf(d)
	e.byrefs = byrefsOf(d)
	e.writable = writablesOf(d)
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
	e.declares = append(e.declares, Declaration{
		Go: d.Name.Name, SP: name, Sig: sig,
		Exported: d.Name.IsExported(),
		Optional: optionalParams(sig, e.defaultsOf(d), e.byrefs, e.mutates, e.lengths),
	})
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
	e.variadic = false
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
	spelling := dimsOf(d)

	var names []string
	for i := range sig.Params().Len() {
		p := sig.Params().At(i)
		name := p.Name()

		/* The variadic tail, which SourcePawn spells any ...

		Only the plugin's own printers take one, and they hand it
		straight to VFormat with the index it starts at. Nothing in a
		generated body reads the arguments: there is no way to, and no
		caller wants one. */
		if sig.Variadic() && i == sig.Params().Len()-1 {
			e.variadic = true
			continue
		}

		if name == "" || name == "_" {
			return "", nil, errUnnamedParam
		}
		// Text a function is given rather than one it owns: SourcePawn
		// takes it as const char[], with no length, because the body may
		// only read it. Anything that writes into a buffer takes a Text
		// and its size, which is a different parameter.
		if b, ok := types.Unalias(p.Type()).(*types.Basic); ok && b.Kind() == types.String {
			names = append(names, name)
			/* Text is const unless the shipped declaration is not

			A const buffer cannot be handed to something that
			declares it writable, and the plugin leaves a few
			writable that it never writes: GiveItemToPlayer takes a
			char[] classname and passes it straight on. The port says
			which with //sp:writable, and the comparison against the
			shipped declaration checks it. */
			if e.writable[name] {
				if spelt, given := spelling[name]; given {
					params = append(params, "char "+e.ident(d.Pos(), name)+"["+spelt+"]")
					continue
				}
				params = append(params, "char[] "+e.ident(d.Pos(), name))
				continue
			}
			params = append(params, "const char[] "+e.ident(d.Pos(), name))
			continue
		}
		tag, dims, terr := e.spType(p.Type())
		if terr != nil {
			return "", nil, terr
		}
		names = append(names, name)
		if tag == "char" && len(dims) == 1 {
			if spelt, given := spelling[name]; given {
				// The dimension is written out, because a callback
				// typedef compares prototypes and not values.
				params = append(params, "char "+e.ident(d.Pos(), name)+"["+spelt+"]")
				continue
			}
			if _, written := e.lengths[name]; written {
				// A buffer this function fills, whose length came
				// with it. Not const: filling it is the point.
				params = append(params, "char[] "+e.ident(d.Pos(), name))
				continue
			}
			// The shipped declaration leaves a few buffers
			// writable that the function never writes, and hands
			// them on to something that demands one. //sp:writable
			// says which, the same way it does for a string.
			if e.writable[name] {
				params = append(params, "char[] "+e.ident(d.Pos(), name))
				continue
			}
			// A buffer a function is handed is const char[]: the
			// length belongs to whoever declared it, and writing
			// this port's own 512 here refuses every caller whose
			// buffer is the schema's length instead. Writing into
			// one needs that length, which is the gap mvm-z83.62
			// carries.
			params = append(params, "const char[] "+e.ident(d.Pos(), name))
			continue
		}
		/* An array parameter is not marked const, though a generated body
		can never write one

		Saying const would match the shipped declarations, and it breaks
		the calls: SourcePawn refuses a const array handed to a native
		that declares its parameter writable, which AimHeadTowards and
		half a dozen others do. So the restriction stays where it is
		enforced, in the emitter's refusal to write an array parameter,
		and the declaration says nothing about it. */
		/* The outer dimension of an array parameter belongs to the caller

		SourcePawn writes float spots[][3] for a list of vectors: how many
		is the caller's business and only the inner shape is fixed. The Go
		has to name a length to have a type at all, and the emitted
		declaration drops it, which is what the shipped SpawnRoutePoints
		takes. */
		if len(dims) > 1 {
			inner := declare(tag, e.ident(d.Pos(), name), dims[1:])
			at := strings.Index(inner, "[")
			params = append(params, inner[:at]+"[]"+inner[at:])
			continue
		}
		if spelt, given := spelling[name]; given {
			// The dimension is written out, because a callback typedef
			// compares prototypes and not values.
			params = append(params, tag+" "+e.ident(d.Pos(), name)+"["+spelt+"]")
			continue
		}
		if e.consts[name] {
			params = append(params, "const "+declare(tag, e.ident(d.Pos(), name), dims))
			continue
		}
		if e.byrefs[name] && len(dims) == 0 {
			params = append(params, declare(tag, "&"+e.ident(d.Pos(), name), nil))
			continue
		}
		params = append(params, declare(tag, e.ident(d.Pos(), name), dims))
	}
	defaults := e.defaultsOf(d)
	params = e.applyDefaults(d, names, params, defaults)

	/* A defaulted parameter has to be last, so the results go before it

	SourcePawn defaults every parameter after the first defaulted one, which
	means an out-parameter appended after one would not compile. The shipped
	files put the out-parameter in the middle for exactly this reason:
	ShouldRelocateNest takes the destination before its defaulted range. */
	var plain, defaulted []string
	for i, name := range names {
		if _, has := defaults[name]; has {
			defaulted = append(defaulted, params[i])
			continue
		}
		plain = append(plain, params[i])
	}
	params = plain

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
		/* An array is by reference in SourcePawn already, and & in front
		of one is not a declaration it takes

		An enum struct is array-like for the same reason and refuses the
		& as well, and there is nothing sensible to zero it with: the
		shipped GetBombInfo writes every field before it returns true and
		leaves the record alone when it returns false. */
		name := e.ident(d.Pos(), r.Name())
		record := isEnumStruct(r.Type())
		e.outParams = append(e.outParams, outParam{name: name, tag: tag, dims: dims, record: record})
		if len(dims) == 0 && !record {
			name = "&" + name
		}
		params = append(params, declare(tag, name, dims))
	}

	params = append(params, defaulted...)

	// The tail goes last, which is the only place SourcePawn takes it.
	if e.variadic {
		params = append(params, "any ...")
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

/*
	enumStructMethod emits one method inside its enum struct's braces

SourcePawn writes a method the way it writes a function, with no keyword in
front of it and with the receiver spelled this. Everything else is the ordinary
body emitter, so a method is checked and written exactly as a plain function is.
*/
func (e *emitter) enumStructMethod(d *ast.FuncDecl) {
	obj, ok := e.info.Defs[d.Name].(*types.Func)
	if !ok {
		e.fail(d.Pos(), "the method %s has no definition", d.Name.Name)
		return
	}
	e.lengths = lengthsOf(d)
	e.consts = constsOf(d)
	e.mutates = mutatesOf(d)
	e.byrefs = byrefsOf(d)
	e.writable = writablesOf(d)

	sig := obj.Type().(*types.Signature)
	e.receiver = sig.Recv().Name()

	e.returnsValue = returnsArray(d)
	ret, params, err := e.signature(d, sig)
	if err != nil {
		e.fail(d.Pos(), "%v", err)
		return
	}
	e.byRef = byRefParams(sig)

	name := e.emittedName(d)
	e.line("%s %s(%s)", ret, name, strings.Join(params, ", "))
	e.line("{")
	e.indent++
	if e.resultName != "" && usesResult(d.Body, e.resultName) {
		e.line("%s;", e.resultDecl)
	} else if e.resultName != "" {
		e.resultName = ""
	}
	for _, out := range e.outParams {
		e.zero(out)
	}
	e.pending = nil
	e.checkClosed(d)
	for _, s := range d.Body.List {
		e.stmt(s)
	}
	if !endsInReturn(d.Body) {
		e.discharge(d.Body.End(), nil)
	}
	e.pending = nil
	e.indent--
	e.line("}")
	e.outParams, e.byRef = nil, nil
	e.resultName, e.resultDecl = "", ""
	e.returnsValue = false
	e.receiver = ""
}

/*
	methodmapMethod emits one method inside its methodmap's braces

SourcePawn writes public in front of a methodmap's method, where an enum
struct's has no keyword. Everything else is the ordinary body emitter.
*/
func (e *emitter) methodmapMethod(d *ast.FuncDecl) {
	obj, ok := e.info.Defs[d.Name].(*types.Func)
	if !ok {
		e.fail(d.Pos(), "the method %s has no definition", d.Name.Name)
		return
	}
	e.lengths = lengthsOf(d)
	e.consts = constsOf(d)
	e.mutates = mutatesOf(d)
	e.byrefs = byrefsOf(d)
	e.writable = writablesOf(d)

	sig := obj.Type().(*types.Signature)
	e.receiver = sig.Recv().Name()

	e.returnsValue = returnsArray(d)
	ret, params, err := e.signature(d, sig)
	if err != nil {
		e.fail(d.Pos(), "%v", err)
		return
	}
	e.byRef = byRefParams(sig)

	/* A constructor has no return type

	SourcePawn writes public Name(...) inside a methodmap called Name, and
	writing the type in front of it is a syntax error rather than a
	redundancy. */
	if name := e.emittedName(d); name == e.methodmapName {
		e.line("public %s(%s)", name, strings.Join(params, ", "))
	} else {
		e.line("public %s %s(%s)", ret, name, strings.Join(params, ", "))
	}
	e.line("{")
	e.indent++
	if e.resultName != "" && usesResult(d.Body, e.resultName) {
		e.line("%s;", e.resultDecl)
	} else if e.resultName != "" {
		e.resultName = ""
	}
	for _, out := range e.outParams {
		e.zero(out)
	}
	e.pending = nil
	e.checkClosed(d)
	for _, s := range d.Body.List {
		e.stmt(s)
	}
	if !endsInReturn(d.Body) {
		e.discharge(d.Body.End(), nil)
	}
	e.pending = nil
	e.indent--
	e.line("}")
	e.outParams, e.byRef = nil, nil
	e.resultName, e.resultDecl = "", ""
	e.returnsValue = false
	e.receiver = ""
}
