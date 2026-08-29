package spbody

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

// directive is the comment that says a function is an engine call, and how
// SourcePawn writes it.
const directive = "//sp:"

// ExternsFromDir reads the extern declarations of one package: every function
// carrying an //sp: directive, keyed by the qualified name a body writes.
//
// The directive lives on the Go declaration rather than in a table here,
// because the body compiles against that declaration. A table would be a second
// place to add a native to, and the two would disagree the first busy week.
func ExternsFromDir(dir string) (map[string]Extern, error) {
	fset := token.NewFileSet()
	files, _, err := parseDir(fset, dir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("spbody: %s holds no Go file", dir)
	}
	out := make(map[string]Extern)
	local := files[0].Name.Name
	for _, f := range files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Doc == nil {
				continue
			}
			extern, found, err := parseDirective(fn)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", fset.Position(fn.Pos()), err)
			}
			if !found {
				continue
			}
			key := local + "." + fn.Name.Name
			if _, dup := out[key]; dup {
				return nil, fmt.Errorf("%s: %s is declared twice", fset.Position(fn.Pos()), key)
			}
			out[key] = extern
		}
	}
	return out, nil
}

func parseDirective(fn *ast.FuncDecl) (Extern, bool, error) {
	for _, c := range fn.Doc.List {
		if !strings.HasPrefix(c.Text, directive) {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(c.Text, directive))
		if len(fields) != 2 {
			return Extern{}, false, fmt.Errorf("the directive %q needs a kind and one name", c.Text)
		}
		kind, name := fields[0], fields[1]
		switch kind {
		case "native":
			return Extern{Func: name}, true, nil
		case "sdkcall":
			return Extern{Func: "SDKCall", Lead: []string{name}}, true, nil
		case "address":
			return Extern{Func: "LoadFromAddress", Lead: []string{name}}, true, nil
		default:
			return Extern{}, false, fmt.Errorf("the directive kind %q is not native, sdkcall or address", kind)
		}
	}
	return Extern{}, false, nil
}
