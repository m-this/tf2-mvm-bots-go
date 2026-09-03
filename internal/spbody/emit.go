package spbody

import (
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// maxErrors stops a file the generator cannot read at all from producing a
// report nobody will read.
const maxErrors = 50

// helper is one of the two builtins SourcePawn does not have, written out.
type helper struct {
	tag string
	op  string
}

type emitter struct {
	fset *token.FileSet
	cfg  Config
	info *types.Info
	pkg  *types.Package

	b      strings.Builder
	hooks  strings.Builder
	inHook bool
	errs   []error
	indent int

	// outParams are the results after the first, which SourcePawn takes as
	// by-reference parameters. Set for the function being emitted.
	outParams []outParam
	// resultName is the named first result, which SourcePawn has no name
	// for: it becomes a local, so a naked return has a value.
	resultName string
	resultDecl string
	// returnsArray says the first Go result is an array, which SourcePawn
	// cannot return: it became a trailing parameter and there is no return
	// value at all.
	returnsArray bool
	// variadic says the function takes the caller's own any ... tail.
	variadic bool
	// returnsValue says the body carries //sp:returns, so an array result is
	// returned rather than filled.
	returnsValue bool
	// valueReturners are the functions in this package that do, so a call to
	// one is emitted as an expression and not rewritten.
	valueReturners map[string]bool
	// spNames maps a Go function name onto the SourcePawn name it is
	// emitted under, so a call to a body that claimed the plugin's name
	// with //sp:name is emitted under that name too.
	spNames map[string]string
	// typeNames are the tags a //sp:name on a type declaration claimed.
	typeNames map[string]string

	// methodmaps are the type names //sp:methodmap claimed. A methodmap is a
	// tag over an integer with no constants of its own, which is the one
	// named integer that has no enum to emit.
	methodmaps map[string]bool
	// lengths maps a buffer parameter of the function being emitted onto
	// the parameter that carries its length, from //sp:length.
	lengths map[string]string
	// closers name the method that releases a handle type, for the ones
	// that are not released with delete.
	closers map[string]string
	// borrowed are the functions whose handle result the caller does not
	// own, from //sp:borrowed.
	borrowed map[string]bool
	// mutates are the parameters the function being emitted writes through,
	// from //sp:mutates.
	mutates map[string]bool
	// byrefs are the scalar parameters the caller reads back, from
	// //sp:byref.
	byrefs map[string]bool
	// writable are the text parameters the shipped declaration leaves
	// writable, from //sp:writable.
	writable map[string]bool
	// consts are the parameters of the function being emitted that carry
	// //sp:const, which SourcePawn writes in front of the type.
	consts map[string]bool
	// handles are the extern types that have to be deleted, by qualified Go
	// name. A local of one is a lifetime this package will not leave open.
	handles map[string]bool
	// pending are the defers seen so far in the function being emitted, in
	// the order they were written; they discharge in reverse.
	pending []deferred
	// declares is every plain function this emission writes, in source
	// order: what it is called in both languages, and the Go signature it
	// was checked against.
	declares []Declaration
	// state is every package-level var, in declaration order, so Reset puts
	// them back in the order they were declared.
	state []stateVar
	// helpers are the builtins that had to be written out, because
	// SourcePawn has neither min nor max.
	helpers map[string]helper

	// byRef are the parameters SourcePawn passes by reference whatever the
	// source says: arrays and enum structs. Assigning to one changes the
	// caller's value in SourcePawn and not in Go, so it is refused.
	byRef map[string]bool

	// methods are the enum struct methods, by the type they hang off. They
	// are emitted inside its braces and nowhere else.
	methods map[string][]*ast.FuncDecl

	// receiver is the name the method being emitted binds its receiver to,
	// which SourcePawn spells this.
	receiver string

	// methodmapName is the methodmap being emitted, so a method that shares
	// its name is written as the constructor it is.
	methodmapName string
}

func (e *emitter) fail(pos token.Pos, format string, args ...any) {
	if len(e.errs) >= maxErrors {
		return
	}
	e.errs = append(e.errs, fmt.Errorf("%s: %s", e.fset.Position(pos), fmt.Sprintf(format, args...)))
}

func (e *emitter) err() error {
	if len(e.errs) == 0 {
		return nil
	}
	return fmt.Errorf("spbody: %d construct(s) with no SourcePawn:\n%w", len(e.errs), errors.Join(e.errs...))
}

func (e *emitter) out() *strings.Builder {
	if e.inHook {
		return &e.hooks
	}
	return &e.b
}

// blank ends a declaration. Separate from line so the one place a write to the
// builder is discarded is here.
func (e *emitter) blank() {
	_, _ = e.out().WriteString("\n")
}

func (e *emitter) line(format string, args ...any) {
	b := e.out()
	b.WriteString(strings.Repeat("\t", e.indent))
	fmt.Fprintf(b, format, args...)
	b.WriteByte('\n')
}

func (e *emitter) run(files []*ast.File) {
	e.collectHandles()
	e.helpers = make(map[string]helper)
	e.valueReturners = make(map[string]bool)
	e.spNames = make(map[string]string)
	e.typeNames = make(map[string]string)
	e.methodmaps = make(map[string]bool)
	e.borrowed = make(map[string]bool)
	for _, f := range files {
		if isGenerated(f) {
			continue
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if returnsArray(fn) {
				e.valueReturners[fn.Name.Name] = true
			}
			if name, claimed := spName(fn); claimed {
				e.spNames[fn.Name.Name] = name
			}
			if isBorrowed(fn) {
				e.borrowed[fn.Name.Name] = true
			}
		}
		// A type claims one too, and it has to be read here rather than
		// where it is emitted: an enum comes out with its constants,
		// which is a pass that runs before the type is reached.
		for _, decl := range f.Decls {
			g, ok := decl.(*ast.GenDecl)
			if !ok || g.Tok != token.TYPE {
				continue
			}
			for _, spec := range g.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if name, claimed := typeName(ts, g); claimed {
					e.typeNames[ts.Name.Name] = name
				}
				if _, isMethodmap := methodmapBase(ts, g); isMethodmap {
					e.methodmaps[ts.Name.Name] = true
				}
			}
		}
		// Package level vars and constants claim a name the same way, and
		// by the same argument: the files that still read them have not
		// moved.
		for _, decl := range f.Decls {
			g, ok := decl.(*ast.GenDecl)
			if !ok || (g.Tok != token.VAR && g.Tok != token.CONST) {
				continue
			}
			for _, spec := range g.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok || len(vs.Names) != 1 {
					continue
				}
				if name, claimed := varName(vs, g); claimed {
					e.spNames[vs.Names[0].Name] = name
				}
			}
		}
	}
	// SourcePawn reads top to bottom for everything but a function call, so
	// the declarations come out in dependency order rather than source
	// order: the defines an array length may use, then the enums and enum
	// structs, then the globals, then the bodies.
	e.collectMethods(files)
	e.eachDecl(files, func(d ast.Decl) bool { g, ok := d.(*ast.GenDecl); return ok && g.Tok == token.CONST })
	e.eachDecl(files, func(d ast.Decl) bool { g, ok := d.(*ast.GenDecl); return ok && g.Tok == token.TYPE })
	e.eachDecl(files, func(d ast.Decl) bool { g, ok := d.(*ast.GenDecl); return ok && g.Tok == token.VAR })
	if len(e.state) > 0 {
		e.blank()
	}
	e.eachDecl(files, func(d ast.Decl) bool { f, ok := d.(*ast.FuncDecl); return ok && f.Recv == nil })
	// Anything left is a declaration no pass claimed, and decl refuses it by
	// name rather than dropping it.
	e.eachDecl(files, func(d ast.Decl) bool {
		g, ok := d.(*ast.GenDecl)
		return ok && g.Tok != token.CONST && g.Tok != token.TYPE && g.Tok != token.VAR && g.Tok != token.IMPORT
	})
}

// prologue is the helpers the body needed, in name order so the output is the
// same twice running.
func (e *emitter) prologue() string {
	if len(e.helpers) == 0 {
		return ""
	}
	names := make([]string, 0, len(e.helpers))
	for name := range e.helpers {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		h := e.helpers[name]
		fmt.Fprintf(&b, "stock %s %s(%s a, %s b)\n{\n\tif (a %s b)\n\t{\n\t\treturn a;\n\t}\n\treturn b;\n}\n\n", h.tag, name, h.tag, h.tag, h.op)
	}
	return b.String()
}

// eachDecl emits the declarations one pass wants, in the order they were
// written, so a reordering here never reorders two things of the same kind.
func (e *emitter) eachDecl(files []*ast.File, want func(ast.Decl) bool) {
	for _, f := range files {
		if isGenerated(f) {
			continue
		}
		for _, decl := range f.Decls {
			if want(decl) {
				e.decl(decl)
			}
		}
	}
}

func (e *emitter) decl(decl ast.Decl) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		e.funcDecl(d)
	case *ast.GenDecl:
		e.genDecl(d)
	default:
		e.fail(decl.Pos(), "an unrecognised declaration")
	}
}

func (e *emitter) genDecl(d *ast.GenDecl) {
	switch d.Tok {
	case token.IMPORT:
	case token.TYPE:
		for _, spec := range d.Specs {
			e.typeSpec(d, spec.(*ast.TypeSpec))
		}
	case token.CONST:
		e.constDecl(d)
	case token.VAR:
		e.varDecl(d)
	default:
		e.fail(d.Pos(), "an unrecognised package-level %s declaration", d.Tok)
	}
}

// typeSpec emits a named struct as an enum struct and a named integer as an
// enum. The enum's constants come from the const declarations, not from here,
// because their order is the order they were written in.
func (e *emitter) typeSpec(d *ast.GenDecl, spec *ast.TypeSpec) {
	obj, ok := e.info.Defs[spec.Name].(*types.TypeName)
	if !ok {
		e.fail(spec.Pos(), "the type %s has no definition", spec.Name.Name)
		return
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		e.fail(spec.Pos(), "the type %s is an alias; declare a defined type", spec.Name.Name)
		return
	}
	st, isStruct := named.Underlying().(*types.Struct)
	if !isStruct {
		/* A methodmap, when the type carries methods

		SourcePawn's other way of hanging behaviour off a value: a tag
		over an integer with methods written inside its braces, which is
		what the game's own addresses are reached through. An enum with
		no methods is emitted with its constants and not here. */
		if base, given := methodmapBase(spec, d); given {
			e.methodmap(spec, d, base)
			return
		}
		if len(e.methods[spec.Name.Name]) > 0 {
			e.fail(spec.Pos(), "%s carries methods and is not a struct; say //sp:methodmap <base> to emit it as a methodmap", spec.Name.Name)
		}
		return // an enum, emitted with its constants
	}
	/* A record keeps the plugin's names, type and fields alike

	It is one declaration shared by everything that reads it, so a port that
	renamed it would rename it for files that have not moved. //sp:name says
	the type's name and an sp struct tag says each field's, the same way
	fieldName reads one at a call site. */
	name := e.cfg.Prefix + spec.Name.Name
	if claimed, ok := typeName(spec, d); ok {
		name = e.ident(spec.Pos(), claimed)
	}
	e.line("enum struct %s", name)
	e.line("{")
	e.indent++
	for i := range st.NumFields() {
		f := st.Field(i)
		tag, dims, err := e.spType(f.Type())
		if err != nil {
			e.fail(spec.Pos(), "field %s: %v", f.Name(), err)
			continue
		}
		field := e.ident(spec.Pos(), f.Name())
		if claimed, ok := reflect.StructTag(st.Tag(i)).Lookup("sp"); ok {
			field = claimed
		}
		e.line("%s;", declare(tag, field, dims))
	}
	for _, m := range e.methods[spec.Name.Name] {
		e.blank()
		e.enumStructMethod(m)
	}
	e.indent--
	e.line("}")
	e.blank()
}

/*
	collectMethods files each method under the type it hangs off

An enum struct's methods are written inside its braces, so they cannot be
emitted where they were declared. Go keeps them beside the type; SourcePawn
keeps them in it.
*/
func (e *emitter) collectMethods(files []*ast.File) {
	e.methods = map[string][]*ast.FuncDecl{}
	for _, f := range files {
		if isGenerated(f) {
			continue
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
				continue
			}
			name, ok := receiverTypeName(fn.Recv.List[0].Type)
			if !ok {
				e.fail(fn.Pos(), "a method on something that is not a named type")
				continue
			}
			e.methods[name] = append(e.methods[name], fn)
		}
	}
}

// receiverTypeName is the type a method hangs off, pointer or not.
func receiverTypeName(t ast.Expr) (string, bool) {
	if star, isPointer := t.(*ast.StarExpr); isPointer {
		t = star.X
	}
	id, ok := t.(*ast.Ident)
	if !ok {
		return "", false
	}
	return id.Name, true
}

// typeName reads //sp:name off a type declaration, from its own doc.
func typeName(spec *ast.TypeSpec, d *ast.GenDecl) (string, bool) {
	// A lone type declaration carries its doc on the group, not the spec,
	// which is where a var declaration keeps it too.
	for _, doc := range []*ast.CommentGroup{spec.Doc, d.Doc} {
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

// constDecl emits one const block. A block whose constants share a named
// integer type is that type's enum; anything else is a define, because a
// define is the only constant SourcePawn accepts as an array length.
func (e *emitter) constDecl(d *ast.GenDecl) {
	type entry struct {
		name  string
		value string
	}
	var group []entry
	tagName := ""
	claimedName := false
	for _, spec := range d.Specs {
		vs := spec.(*ast.ValueSpec)
		for _, name := range vs.Names {
			if name.Name == "_" {
				continue
			}
			c, ok := e.info.Defs[name].(*types.Const)
			if !ok {
				e.fail(name.Pos(), "the constant %s has no value the type checker could fold", name.Name)
				continue
			}
			lit, err := e.constLiteral(c)
			if err != nil {
				e.fail(name.Pos(), "the constant %s: %v", name.Name, err)
				continue
			}
			if named, ok := c.Type().(*types.Named); ok {
				if tagName != "" && tagName != named.Obj().Name() {
					e.fail(name.Pos(), "one const block declares constants of %s and of %s; write one block per enum", tagName, named.Obj().Name())
					continue
				}
				tagName = named.Obj().Name()
			}
			emitted := e.ident(name.Pos(), name.Name)
			if claimed, ok := varName(vs, d); ok && len(vs.Names) == 1 {
				// A constant keeps the plugin's name for the same
				// reason a function does: what still reads it has
				// not moved.
				emitted = e.ident(name.Pos(), claimed)
				e.spNames[name.Name] = claimed
				claimedName = true
			}
			group = append(group, entry{name: emitted, value: lit})
		}
	}
	if len(group) == 0 {
		return
	}
	if tagName == "" {
		for _, c := range group {
			prefix := e.cfg.Prefix
			if claimedName {
				prefix = ""
			}
			e.line("#define %s%s (%s)", prefix, c.name, c.value)
		}
		e.blank()
		return
	}
	/* An enum keeps the plugin's names when the port claimed them

	The members already carry whatever //sp:name said, so prefixing them here
	renamed exactly the constants the port had asked to keep. The tag is the
	same: it is a type other files declare variables of. */
	enumName := e.cfg.Prefix + tagName
	if claimed, ok := e.typeNames[tagName]; ok {
		enumName = claimed
	}
	e.line("enum %s", enumName)
	e.line("{")
	e.indent++
	for i, c := range group {
		comma := ","
		if i == len(group)-1 {
			comma = ""
		}
		prefix := e.cfg.Prefix
		if claimedName {
			prefix = ""
		}
		e.line("%s%s = %s%s", prefix, c.name, c.value, comma)
	}
	e.indent--
	e.line("};")
	e.blank()
}

func (e *emitter) constLiteral(c *types.Const) (string, error) {
	/* A constant naming a piece of text, which SourcePawn writes as a define

	The plugin has a handful: MVM_TANK_CLASS_ICON is "tank", and the wave bar
	is read by comparing against it. There is no id that would do instead,
	and a define is what the shipped file writes. */
	if c.Val().Kind() == constant.String {
		text := constant.StringVal(c.Val())
		for _, r := range text {
			// The same three escapes stringLit allows, for the
			// same reason: SourcePawn spells them identically.
			if r == '\n' || r == '\t' || r == '\r' {
				continue
			}
			if r < ' ' || r > '~' {
				return "", fmt.Errorf("the text %q holds a character this package will not escape for SourcePawn", text)
			}
		}
		return strconv.Quote(text), nil
	}
	tag, _, err := e.spType(c.Type())
	if err != nil {
		return "", err
	}
	return literalOf(c.Val(), tag)
}

func literalOf(v constant.Value, tag string) (string, error) {
	switch tag {
	case "bool":
		if constant.BoolVal(v) {
			return "true", nil
		}
		return "false", nil
	case "float":
		// Float32Val says whether the value survived exactly, and almost
		// none do: 0.3 is not representable in 32 bits and neither is the
		// 0.3 the plugin writes. What matters is that it is a number and
		// not an overflow, and that the literal reads back as the same
		// float, which internal/sp is what guarantees.
		f, _ := constant.Float32Val(v)
		if math.IsInf(float64(f), 0) {
			return "", fmt.Errorf("%s is outside the range of a 32 bit SourcePawn float", v)
		}
		return floatLiteral(f), nil
	default:
		i, ok := constant.Int64Val(constant.ToInt(v))
		if !ok {
			return "", fmt.Errorf("%s is not an integer a cell holds", v)
		}
		return fmt.Sprintf("%d", i), nil
	}
}

// methodmapDirective names the tag a methodmap is written over.
const methodmapDirective = "//sp:methodmap"

// methodmapBase reads //sp:methodmap <base> off a type declaration.
func methodmapBase(spec *ast.TypeSpec, d *ast.GenDecl) (string, bool) {
	for _, doc := range []*ast.CommentGroup{spec.Doc, d.Doc} {
		if doc == nil {
			continue
		}
		for _, c := range doc.List {
			for line := range strings.Lines(c.Text) {
				fields := strings.Fields(line)
				if len(fields) == 2 && fields[0] == methodmapDirective {
					return fields[1], true
				}
				if len(fields) == 1 && fields[0] == methodmapDirective {
					// No base: a methodmap over nothing, which is what
					// the plugin writes for a tag it invented.
					return "", true
				}
			}
		}
	}
	return "", false
}

/*
	methodmap emits a tag over an integer with its methods inside it

The name is the plugin's, the base is what //sp:methodmap says, and the methods
are the ordinary body emitter with the receiver spelled this, exactly as an
enum struct's are.
*/
func (e *emitter) methodmap(spec *ast.TypeSpec, d *ast.GenDecl, base string) {
	name := e.cfg.Prefix + spec.Name.Name
	if claimed, ok := typeName(spec, d); ok {
		name = e.ident(spec.Pos(), claimed)
	}
	e.methodmapName = name
	defer func() { e.methodmapName = "" }()

	if base == "" {
		e.line("methodmap %s", name)
	} else {
		e.line("methodmap %s < %s", name, base)
	}
	e.line("{")
	e.indent++
	for i, m := range e.methods[spec.Name.Name] {
		if i > 0 {
			e.blank()
		}
		e.methodmapMethod(m)
	}
	e.indent--
	e.line("}")
	e.blank()
}
