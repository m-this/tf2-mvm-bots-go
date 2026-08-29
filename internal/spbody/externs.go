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

// Declared is what one extern package says a body may reach: the calls, and the
// types that already have a SourcePawn name.
type Declared struct {
	// Funcs are the externs, by the qualified name a body writes.
	Funcs map[string]Extern
	// Tags map a qualified Go type name onto the SourcePawn tag it stands
	// for, so a ported signature keeps TFClassType rather than widening to
	// int and making every caller a tag mismatch.
	Tags map[string]string
}

// ExternsFromDir reads the extern declarations of one package: every function
// carrying an //sp: directive, and every type carrying an //sp:tag.
//
// The directive lives on the Go declaration rather than in a table here,
// because the body compiles against that declaration. A table would be a second
// place to add a native to, and the two would disagree the first busy week.
func ExternsFromDir(dir string) (Declared, error) {
	fset := token.NewFileSet()
	files, _, err := parseDir(fset, dir)
	if err != nil {
		return Declared{}, err
	}
	if len(files) == 0 {
		return Declared{}, fmt.Errorf("spbody: %s holds no Go file", dir)
	}
	out := Declared{Funcs: make(map[string]Extern), Tags: make(map[string]string)}
	local := files[0].Name.Name
	for _, f := range files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Doc == nil {
					continue
				}
				extern, found, err := parseDirective(d.Doc)
				if err != nil {
					return Declared{}, fmt.Errorf("%s: %w", fset.Position(d.Pos()), err)
				}
				if !found {
					continue
				}
				key := local + "." + d.Name.Name
				if _, dup := out.Funcs[key]; dup {
					return Declared{}, fmt.Errorf("%s: %s is declared twice", fset.Position(d.Pos()), key)
				}
				out.Funcs[key] = extern
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					doc := ts.Doc
					if doc == nil {
						doc = d.Doc
					}
					tag, found := parseTag(doc)
					if !found {
						continue
					}
					out.Tags[local+"."+ts.Name.Name] = tag
				}
			}
		}
	}
	return out, nil
}

// parseTag reads //sp:tag NAME, which says the Go type stands for a SourcePawn
// tag that already exists and is not emitted here.
func parseTag(doc *ast.CommentGroup) (string, bool) {
	if doc == nil {
		return "", false
	}
	for _, c := range doc.List {
		fields := strings.Fields(c.Text)
		if len(fields) == 2 && fields[0] == directive+"tag" {
			return fields[1], true
		}
	}
	return "", false
}

func parseDirective(doc *ast.CommentGroup) (Extern, bool, error) {
	for _, c := range doc.List {
		if !strings.HasPrefix(c.Text, directive) {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(c.Text, directive))
		if len(fields) < 2 || len(fields) > 3 {
			return Extern{}, false, fmt.Errorf("the directive %q needs a kind, one name and at most one flag", c.Text)
		}
		kind, name := fields[0], fields[1]
		returnsArray := false
		if len(fields) == 3 {
			if fields[2] != "returns" {
				return Extern{}, false, fmt.Errorf("the directive flag %q is not returns", fields[2])
			}
			returnsArray = true
		}
		switch kind {
		case "native":
			return Extern{Func: name, ReturnsArray: returnsArray}, true, nil
		case "global":
			return Extern{Func: name, Global: true}, true, nil
		case "plugin":
			return Extern{Func: name, Plugin: true, ReturnsArray: returnsArray}, true, nil
		case "sdkcall":
			return Extern{Func: "SDKCall", Lead: []string{name}, ReturnsArray: returnsArray}, true, nil
		case "address":
			return Extern{Func: "LoadFromAddress", Lead: []string{name}, ReturnsArray: returnsArray}, true, nil
		default:
			return Extern{}, false, fmt.Errorf("the directive kind %q is not native, global, plugin, sdkcall or address", kind)
		}
	}
	return Extern{}, false, nil
}
