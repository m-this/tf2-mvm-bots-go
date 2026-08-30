package gosubset

import (
	"fmt"
	"go/ast"
	"go/token"
)

type checker struct {
	fset     *token.FileSet
	natives  map[string]bool
	packages map[string]map[string]bool
	// packageNames are the last path elements of the mapped packages, so a
	// selector that reads as one of them in a file that did not import it
	// can be refused as the leak it is rather than read as a method call.
	packageNames map[string]bool
	imports      map[string]string // local name -> import path
	types        map[string]bool   // package-level type names
	funcs        map[string]bool   // package-level function names
	diags        []Diagnostic
}

func newChecker(fset *token.FileSet, cfg Config) *checker {
	c := &checker{
		fset:         fset,
		natives:      make(map[string]bool, len(cfg.Natives)),
		packages:     make(map[string]map[string]bool, len(cfg.Packages)),
		packageNames: make(map[string]bool, len(cfg.Packages)),
		imports:      make(map[string]string),
		types:        make(map[string]bool),
		funcs:        make(map[string]bool),
	}
	for _, n := range cfg.Natives {
		c.natives[n] = true
	}
	for path, members := range cfg.Packages {
		set := make(map[string]bool, len(members))
		for _, m := range members {
			set[m] = true
		}
		c.packages[path] = set
		name := path
		if i := lastSlash(path); i >= 0 {
			name = path[i+1:]
		}
		c.packageNames[name] = true
	}
	return c
}

func (c *checker) refuse(pos token.Pos, construct, fix string) {
	if len(c.diags) >= maxDiagnostics {
		return
	}
	c.diags = append(c.diags, Diagnostic{Pos: c.fset.Position(pos), Construct: construct, Fix: fix})
}

// beginFile drops the previous file's imports. An import name is file-scoped
// in Go, so carrying it into the next file would accept a selector that file
// never imported.
func (c *checker) beginFile() {
	c.imports = make(map[string]string)
}

// collect records the package-level names a body may refer to, so that an
// unknown identifier in call or type position can be refused by name. It runs
// over every file of the package before any file is checked, because the
// declaration and its use need not be in the same file.
func (c *checker) collect(f *ast.File) {
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			c.funcs[d.Name.Name] = true
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					c.types[ts.Name.Name] = true
				}
			}
		}
	}
}

func (c *checker) checkFile(f *ast.File) {
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			c.checkFuncDecl(d)
		case *ast.GenDecl:
			c.checkGenDecl(d, true)
		default:
			c.refuse(decl.Pos(), "an unrecognised top-level declaration",
				"a subset file holds imports, constants, type declarations and functions")
		}
	}
}

func (c *checker) checkGenDecl(d *ast.GenDecl, atPackageLevel bool) {
	switch d.Tok {
	case token.IMPORT:
		for _, spec := range d.Specs {
			c.checkImport(spec.(*ast.ImportSpec))
		}
	case token.TYPE:
		if !atPackageLevel {
			c.refuse(d.Pos(), "a type declared inside a function",
				"declare the type at package level; SourcePawn has no function-local enum or struct")
			return
		}
		for _, spec := range d.Specs {
			c.checkTypeSpec(spec.(*ast.TypeSpec))
		}
	case token.CONST, token.VAR:
		for _, spec := range d.Specs {
			c.checkValueSpec(spec.(*ast.ValueSpec), d.Tok, atPackageLevel)
		}
	}
}

func (c *checker) checkImport(spec *ast.ImportSpec) {
	path := ""
	if _, err := fmt.Sscanf(spec.Path.Value, "%q", &path); err != nil {
		return
	}
	members, ok := c.packages[path]
	if !ok || members == nil {
		c.refuse(spec.Pos(), fmt.Sprintf("an import of %q", path),
			"the subset imports nothing the generator has no mapping for; move the call to a native binding or to a function in this package")
		return
	}
	name := path
	if i := lastSlash(path); i >= 0 {
		name = path[i+1:]
	}
	if spec.Name != nil {
		if spec.Name.Name == "." || spec.Name.Name == "_" {
			c.refuse(spec.Pos(), "a dot or blank import",
				"import the package under its own name so every use is visible in the source")
			return
		}
		name = spec.Name.Name
	}
	c.imports[name] = path
}

func lastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

func (c *checker) checkValueSpec(spec *ast.ValueSpec, tok token.Token, atPackageLevel bool) {
	/* A table of names, which SourcePawn spells char x[][]

	The one shape of text the subset holds as state. Both of the plugin's are
	lists of entity classnames walked by index, and FindEntityByClassname
	wants a real string: there is no id that would do instead. Read only,
	which is enforced by there being no way to emit a write into one. */
	if tok == token.VAR && atPackageLevel && specIsNameTable(spec) {
		for _, v := range spec.Values {
			c.checkNameTable(v)
		}
		return
	}
	if tok == token.VAR && atPackageLevel {
		// A package-level var is the plugin's own state, emitted as a
		// SourcePawn global. Its initialiser is written once at load, so
		// it has to be something SourcePawn can write there.
		for _, v := range spec.Values {
			c.checkStaticInit(v)
		}
	}
	if spec.Type != nil {
		c.checkType(spec.Type)
	}
	for _, v := range spec.Values {
		c.checkExpr(v)
	}
}

// specIsNameTable reads the [N]string off the declaration or off the literal,
// because both `var x [5]string = ...` and `var x = [5]string{...}` are written.
func specIsNameTable(spec *ast.ValueSpec) bool {
	if isStringArray(spec.Type) {
		return true
	}
	for _, v := range spec.Values {
		if lit, ok := v.(*ast.CompositeLit); ok && isStringArray(lit.Type) {
			return true
		}
	}
	return false
}

// isStringArray is the [N]string a table of names is written as.
func isStringArray(t ast.Expr) bool {
	arr, ok := t.(*ast.ArrayType)
	if !ok || arr.Len == nil {
		return false
	}
	elem, ok := arr.Elt.(*ast.Ident)
	return ok && elem.Name == "string"
}

// checkNameTable accepts a literal of names and nothing else: no expression, no
// nesting, no computed element.
func (c *checker) checkNameTable(v ast.Expr) {
	lit, ok := v.(*ast.CompositeLit)
	if !ok {
		c.refuse(v.Pos(), "a table of names that is not a literal",
			"write the names out; SourcePawn fills a char[][] at load and computes nothing")
		return
	}
	for _, elt := range lit.Elts {
		text, ok := elt.(*ast.BasicLit)
		if !ok || text.Kind != token.STRING {
			c.refuse(elt.Pos(), "an entry of a name table that is not a name",
				"write a string literal; the table is emitted as it is written")
		}
	}
}

// checkStaticInit accepts what SourcePawn writes at load: a constant, or an
// array literal of constants. Anything computed runs at load in Go and never
// runs at all in SourcePawn, so it is refused rather than half emitted.
func (c *checker) checkStaticInit(v ast.Expr) {
	lit, ok := v.(*ast.CompositeLit)
	if !ok {
		c.checkConstExpr(v, "a package-level initialiser")
		return
	}
	if lit.Type != nil {
		c.checkType(lit.Type)
	}
	for _, elt := range lit.Elts {
		c.checkStaticInit(elt)
	}
}

func (c *checker) checkTypeSpec(spec *ast.TypeSpec) {
	if spec.TypeParams != nil {
		c.refuse(spec.TypeParams.Pos(), "a generic type",
			"write the type out for each element type it is used with; SourcePawn has no type parameters")
	}
	if spec.Assign.IsValid() {
		c.refuse(spec.Pos(), "a type alias",
			"declare a defined type with `type X int32`; an alias has no name of its own to generate")
	}
	if st, ok := spec.Type.(*ast.StructType); ok {
		c.checkStructFields(st)
		return
	}
	c.checkType(spec.Type)
}

func (c *checker) checkStructFields(st *ast.StructType) {
	if st.Fields == nil {
		return
	}
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			c.refuse(f.Pos(), "an embedded struct field",
				"name the field; the generator emits a flat SourcePawn enum struct with no inheritance")
			continue
		}
		c.checkType(f.Type)
	}
}
