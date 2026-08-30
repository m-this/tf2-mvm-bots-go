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
				if recv, ok := receiverType(d); ok {
					// A method is keyed by the type it hangs
					// off, because that is what resolves a
					// call site: the receiver's type, then
					// the name.
					extern.Method = true
					key = local + "." + recv + "." + d.Name.Name
				}
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
	for _, line := range docLines(doc) {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == directive+"tag" {
			return fields[1], true
		}
	}
	return "", false
}

// receiverType is the named type a method hangs off, without its pointer, and
// false for a plain function.
func receiverType(d *ast.FuncDecl) (string, bool) {
	if d.Recv == nil || len(d.Recv.List) != 1 {
		return "", false
	}
	switch t := d.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return t.Name, true
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name, true
		}
	}
	return "", false
}

func parseDirective(doc *ast.CommentGroup) (Extern, bool, error) {
	// Line by line inside each comment: a doc comment may be a /* */ block
	// with the directive somewhere in the middle of it.
	for _, line := range docLines(doc) {
		if !strings.HasPrefix(line, directive) {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, directive))
		// after NAME... are constant arguments written last, so they are
		// taken off before the one optional flag is read.
		var lead []string
		for i, f := range fields {
			if f == "before" {
				if i+1 == len(fields) {
					return Extern{}, false, fmt.Errorf("the directive %q says before and names no argument", line)
				}
				lead = fields[i+1:]
				fields = fields[:i]
				break
			}
		}
		var trail []string
		for i, f := range fields {
			if f == "after" {
				if i+1 == len(fields) {
					return Extern{}, false, fmt.Errorf("the directive %q says after and names no argument", line)
				}
				trail = fields[i+1:]
				fields = fields[:i]
				break
			}
		}
		if len(fields) < 2 || len(fields) > 3 {
			return Extern{}, false, fmt.Errorf("the directive %q needs a kind, one name and at most one flag", line)
		}
		kind, name := fields[0], fields[1]
		returnsArray, sized, fills, inPlace := false, false, false, false
		if len(fields) == 3 {
			switch fields[2] {
			case "returns":
				returnsArray = true
			case "sized":
				sized = true
			case "fills":
				fills = true
			case "inplace":
				inPlace = true
			default:
				return Extern{}, false, fmt.Errorf("the directive flag %q is not returns, sized, fills or inplace", fields[2])
			}
		}
		switch kind {
		case "native":
			return Extern{Func: name, Lead: lead, ReturnsArray: returnsArray, Sized: sized, Fills: fills, InPlace: inPlace, Trail: trail}, true, nil
		case "propertyset":
			// The same read, written to: recv.Name = value, from a call
			// taking the value.
			return Extern{Func: name, Method: true, Property: true, Set: true}, true, nil
		case "property":
			// Written on the receiver like a method, and without the
			// parentheses: convar.BoolValue, not convar.BoolValue().
			return Extern{Func: name, Method: true, Property: true}, true, nil
		case "cast":
			// view_as<Tag>(x): a tag change and nothing else, which is
			// what a callback handed a raw handle has to do before it
			// can use the methodmap.
			return Extern{Func: name, Cast: true}, true, nil
		case "choice":
			return Extern{Func: name, Choice: true}, true, nil
		case "delete":
			return Extern{Func: name, Delete: true}, true, nil
		case "new":
			// SourcePawn spells a constructor new Thing(), which is
			// the only call whose name has a space in it.
			return Extern{Func: "new " + name}, true, nil
		case "method":
			// The receiver is what picks it; the name is what
			// SourcePawn writes after the dot. A method fills a
			// buffer the same way a native does, so it takes the
			// same flags.
			return Extern{Func: name, ReturnsArray: returnsArray, Sized: sized, Fills: fills}, true, nil
		case "slot":
			return Extern{Func: name, Slot: true}, true, nil
		case "slotset":
			return Extern{Func: name, Slot: true, Set: true}, true, nil
		case "global":
			return Extern{Func: name, Global: true}, true, nil
		case "body":
			// A ported function fills a buffer the same way an
			// unported one does, so it takes the same flags.
			return Extern{Func: name, Body: true, ReturnsArray: returnsArray, Sized: sized, Fills: fills, Trail: trail}, true, nil
		case "plugin":
			return Extern{Func: name, Plugin: true, ReturnsArray: returnsArray, Sized: sized, Fills: fills, Trail: trail}, true, nil
		case "sdkcall":
			return Extern{Func: "SDKCall", Lead: []string{name}, ReturnsArray: returnsArray}, true, nil
		case "address":
			return Extern{Func: "LoadFromAddress", Lead: []string{name}, ReturnsArray: returnsArray}, true, nil
		default:
			return Extern{}, false, fmt.Errorf("the directive kind %q is not native, cast, choice, new, method, property, propertyset, delete, global, slot, slotset, body, plugin, sdkcall or address", kind)
		}
	}
	return Extern{}, false, nil
}

// docLines is every line of a doc comment, trimmed, whether it was written as
// a run of // or as one /* */ block.
func docLines(doc *ast.CommentGroup) []string {
	var out []string
	for _, c := range doc.List {
		for line := range strings.Lines(c.Text) {
			out = append(out, strings.TrimSpace(line))
		}
	}
	return out
}
