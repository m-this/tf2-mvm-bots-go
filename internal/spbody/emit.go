package spbody

import (
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"sort"
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
	// handles are the extern types that have to be deleted, by qualified Go
	// name. A local of one is a lifetime this package will not leave open.
	handles map[string]bool
	// pending are the defers seen so far in the function being emitted, in
	// the order they were written; they discharge in reverse.
	pending []deferred
	// emitted are the SourcePawn function names this emission declares.
	emitted []string
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
	e.eachDecl(files, func(d ast.Decl) bool { g, ok := d.(*ast.GenDecl); return ok && g.Tok == token.CONST })
	e.eachDecl(files, func(d ast.Decl) bool { g, ok := d.(*ast.GenDecl); return ok && g.Tok == token.TYPE })
	e.eachDecl(files, func(d ast.Decl) bool { g, ok := d.(*ast.GenDecl); return ok && g.Tok == token.VAR })
	if len(e.state) > 0 {
		e.blank()
	}
	e.eachDecl(files, func(d ast.Decl) bool { _, ok := d.(*ast.FuncDecl); return ok })
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
			e.typeSpec(spec.(*ast.TypeSpec))
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
func (e *emitter) typeSpec(spec *ast.TypeSpec) {
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
		return // an enum, emitted with its constants
	}
	e.line("enum struct %s%s", e.cfg.Prefix, spec.Name.Name)
	e.line("{")
	e.indent++
	for i := range st.NumFields() {
		f := st.Field(i)
		tag, dims, err := e.spType(f.Type())
		if err != nil {
			e.fail(spec.Pos(), "field %s: %v", f.Name(), err)
			continue
		}
		e.line("%s;", declare(tag, e.ident(spec.Pos(), f.Name()), dims))
	}
	e.indent--
	e.line("}")
	e.blank()
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
	e.line("enum %s%s", e.cfg.Prefix, tagName)
	e.line("{")
	e.indent++
	for i, c := range group {
		comma := ","
		if i == len(group)-1 {
			comma = ""
		}
		e.line("%s%s = %s%s", e.cfg.Prefix, c.name, c.value, comma)
	}
	e.indent--
	e.line("};")
	e.blank()
}

func (e *emitter) constLiteral(c *types.Const) (string, error) {
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
		f, ok := constant.Float32Val(v)
		if !ok {
			return "", fmt.Errorf("%s does not fit a 32 bit SourcePawn float", v)
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
