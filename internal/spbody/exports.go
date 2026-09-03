package spbody

import (
	"fmt"
	"go/ast"
	"go/token"
)

/*
What one generated package offers another

The emitted SourcePawn is one flat namespace, so a body calling another body is
a call by name and nothing more. What stops it is not the language, it is that
the caller has no way to learn the name: the emitter resolves a qualified call
through Config.Externs, and until now the only package in there was the engine.

So a body that wanted to share a decision with another one declared an extern
for it by hand, in internal/engine, restating a signature that already existed
in Go a directory away. There are hundreds of those, and every one of them is a
name typed twice.

Exports reads the names out of the package instead. It parses and does not type
check, because a name is all that is wanted and type checking the callee is the
caller's own type check's job: its importer already resolves the package from
source.
*/

// Export is one thing a generated package offers, by the name a Go caller
// writes and the name the SourcePawn calls. Const says it is a constant and not
// a function: it has no SourcePawn name, because the emitter writes its value.
type Export struct {
	Go    string
	SP    string
	Const bool
}

/*
Exports are the exported plain functions of the package in dir, under the prefix
its registry entry gives it.

Exported, because SourcePawn has no visibility and Go does: a lower-case
function is this package's business, and letting another one call it would make
every helper part of an interface nobody agreed to. Plain, because a method is
written on its receiver and a receiver is a type the caller cannot name.
*/
func Exports(dir, prefix string) ([]Export, error) {
	fset := token.NewFileSet()
	files, _, err := parseDir(fset, dir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("spbody: %s holds no Go file", dir)
	}
	var out []Export
	for _, f := range files {
		if isGenerated(f) {
			continue
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv != nil || !d.Name.IsExported() {
					continue
				}
				name := prefix + d.Name.Name
				if claimed, given := spName(d); given {
					name = claimed
				}
				out = append(out, Export{Go: d.Name.Name, SP: name})
			case *ast.GenDecl:
				// A constant, which a caller writes and the
				// emitter folds: SourcePawn has one flat
				// namespace for a #define, so only the package
				// that declares it may write the name.
				if d.Tok != token.CONST {
					continue
				}
				for _, spec := range d.Specs {
					vs, isValue := spec.(*ast.ValueSpec)
					if !isValue {
						continue
					}
					for _, id := range vs.Names {
						if id.IsExported() {
							out = append(out, Export{Go: id.Name, Const: true})
						}
					}
				}
			}
		}
	}
	return out, nil
}
